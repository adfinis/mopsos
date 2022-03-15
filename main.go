/**
 * mopsos - Mopsos receives events and stores them in a database for later analysis.
 */
package main

import "github.com/adfinis-sygroup/mopsos/app/cmd"

func main() {
	// This starts the root command which is the main entry point for the application.
	cmd.Execute()
}
