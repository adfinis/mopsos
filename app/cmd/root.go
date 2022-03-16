package cmd

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	mopsos "github.com/adfinis-sygroup/mopsos/app"
	"github.com/adfinis-sygroup/mopsos/app/db"
	"github.com/adfinis-sygroup/mopsos/app/instrumentation"
)

const envPrefix = "MOPSOS"

var rootCmd = &cobra.Command{
	Use:   "mopsos",
	Short: "Mopsos receives events and stores them in a database",
	Long:  "Mopsos receives events and stores them in a database for later analysis.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
	// Run builds the applications object composition and starts the server
	Run: func(cmd *cobra.Command, args []string) {

		// check logging early so we can use it from here on out
		logrus.SetLevel(logrus.WarnLevel)
		verboseMode, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			logrus.WithError(err).Fatal("failed to get verbose flag")
		}
		if verboseMode {
			logrus.SetLevel(logrus.InfoLevel)
		}
		debugMode, err := cmd.Flags().GetBool("debug")
		if err != nil {
			logrus.WithError(err).Fatal("failed to get debug flag")
		}
		if debugMode {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.Debug("Debug mode enabled")
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				logrus.Debugf("flag '%s': %s", f.Name, f.Value.String())
			})
		}

		// read DB flags
		provider := cmd.Flag("db-provider").Value.String()
		dsn := cmd.Flag("db-dsn").Value.String()
		migrate, err := cmd.Flags().GetBool("db-migrate")
		if err != nil {
			logrus.Fatal(err)
		}

		// read http flags
		listener := cmd.Flag("http-listener").Value.String()

		// read otel flags
		enableTracing, err := cmd.Flags().GetBool("otel")
		if err != nil {
			logrus.Fatal(err)
		}
		tracingTarget, err := cmd.Flags().GetString("otel-collector")
		if err != nil {
			logrus.Fatal(err)
		}

		// build config struct
		cfg := &mopsos.Config{
			DBProvider: provider,
			DBDSN:      dsn,
			DBMigrate:  migrate,

			HttpListener: listener,

			EnableTracing: enableTracing,
			TracingTarget: tracingTarget,
		}
		log := logrus.WithField("config", fmt.Sprintf("%+v", cfg))

		if enableTracing {
			shutdown := instrumentation.InitInstrumentation(cfg.TracingTarget)
			defer shutdown()
		}

		// prepare gorm.io database connection
		log.Debug("Connecting to database")
		dbConn, err := db.NewDBConnection(cfg)
		if err != nil {
			log.Fatal("Failed to connect to database: ", err)
		}
		log.Info("Connected to database")

		// prepare main app
		app, err := mopsos.NewApp(cfg, dbConn)
		if err != nil {
			log.Fatal(err)
		}

		// run main loop of the application (server and event receiver)
		app.Run()
	},
}

func Execute() {
	// database flags
	rootCmd.Flags().String("db-provider", "sqlite", "Database provider, either 'sqlite' or 'postgres'")
	rootCmd.Flags().String("db-dsn", "file::memory:?cache=shared", "Database DSN")
	rootCmd.Flags().Bool("db-migrate", true, "Migrate database schema on startup")

	// webserver flags
	rootCmd.Flags().String("http-listener", ":8080", "HTTP listener")

	// otel flags
	rootCmd.Flags().Bool("otel", false, "Enable OpenTelemetry tracing")
	rootCmd.Flags().String("otel-collector", "localhost:30079", `Endpoint for OpenTelemetry Collector. `+
		`On a local cluster the collector should be accessible through a NodePort service at the localhost:30078 `+
		`endpoint. Otherwise replace localhost with the collector endpoint.`)

	// logging flags
	rootCmd.Flags().Bool("debug", false, "Enable debug mode")
	rootCmd.Flags().Bool("verbose", false, "Enable verbose mode")

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

/**
 * initConfig reads in config file and ENV variables if set.
 *
 * based on [stingoftheviper](https://github.com/carolynvs/stingoftheviper)
 */
func initConfig(cmd *cobra.Command) error {
	cfg := viper.New()
	cfg.SetConfigName("mopsos")
	cfg.AddConfigPath(".")

	viper.AutomaticEnv()

	if err := cfg.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return err
		}
	}

	cfg.SetEnvPrefix(envPrefix)
	cfg.AutomaticEnv()
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to underscorized variants
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			if err := cfg.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix)); err != nil {
				logrus.WithFields(logrus.Fields{
					"flag": f.Name,
				}).WithError(err).Fatal("failed to bind environment variable")
			}
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && cfg.IsSet(f.Name) {
			if err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", cfg.Get(f.Name))); err != nil {
				logrus.WithFields(logrus.Fields{
					"flag": f.Name,
				}).WithError(err).Fatal("failed to set flag")
			}
		}

	})

	return nil
}
