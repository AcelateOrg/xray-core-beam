# Beam Core

Network core of the **Beam** VPN client.

Beam Core is a modified fork of [Xray-core](https://github.com/XTLS/Xray-core)
by Project X (XTLS), adapted for Beam: integration with the Beam TUN stack,
connection tracking and startup optimizations.

> Not affiliated with or endorsed by Project X / XTLS.

## How Beam uses it

Beam does not run the core as a separate process. [`beam-bridge`](https://github.com/AcelateOrg/beam-bridge)
links it as a Go library: it builds the core config from the user's profile,
starts the core in-process and connects it to the TUN adapter from
[`beam-tun`](https://github.com/AcelateOrg/beam-tun).

The module path stays `github.com/xtls/xray-core`, so `beam-bridge` swaps in
this fork with a `replace` directive pointing at a local checkout:

```go
replace github.com/xtls/xray-core => ../xray-core-beam
```

## Build

A standalone binary is only needed for testing the core on its own.

### Windows (PowerShell)

```powershell
$env:CGO_ENABLED=0
go build -o beam-core.exe -trimpath -buildvcs=false -ldflags="-s -w -buildid=" ./main
```

### Linux / macOS

```bash
CGO_ENABLED=0 go build -o beam-core -trimpath -buildvcs=false -ldflags="-s -w -buildid=" ./main
```

Tests: `go test ./...`

## Upstream

Based on [XTLS/Xray-core](https://github.com/XTLS/Xray-core), which itself was
forked from [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core).
Third-party dependencies are listed in [go.mod](go.mod).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues: [SECURITY.md](SECURITY.md).

## License

[Mozilla Public License 2.0](LICENSE).

Original code © Project X (XTLS) and V2Ray / v2fly contributors.
Beam modifications © AcelateOrg.
