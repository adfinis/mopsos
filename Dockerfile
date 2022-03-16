FROM scratch

COPY mopsos /
COPY etc/passwd /etc/passwd

USER nobody

ENTRYPOINT ["/mopsos"]
