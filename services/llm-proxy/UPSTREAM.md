# Upstream dependency

The subscription proxy imports `github.com/router-for-me/CLIProxyAPI/v8`
version `v8.0.19` as a Go module. The application repository does not copy or
patch the SDK source. Builds use the version and checksums in `go.mod` and
`go.sum`.

The wrapper uses the SDK config loader, service builder, and file token store
through its credential bridge. SDK v8 preserves these interfaces and accepts
the existing YAML configuration. `Service.Run` starts model catalog updates;
shutdown closes active HTTP connections immediately.

CLIProxyAPI is MIT licensed. Its license is included in
`THIRD_PARTY_LICENSE_CLIProxyAPI.txt` and copied into the container image.
