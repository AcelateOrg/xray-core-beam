# Contributing

Thanks for helping improve Beam Core.

## Where to send changes

- Bugs that also exist in upstream [Xray-core](https://github.com/XTLS/Xray-core)
  belong there. We pick up upstream fixes when we sync.
- This repository takes changes specific to Beam: TUN integration, connection
  tracking, startup and memory optimizations, and fixes to Beam's own patches.
- Security issues: see [SECURITY.md](SECURITY.md), not public issues.

## Process

1. Open an issue first for anything larger than a small fix.
2. Fork, create a branch, keep the change focused.
3. Run `go vet ./...` and `go test ./...`; format with `gofmt`.
4. If the change affects how Beam uses the core, check that
   [beam-bridge](https://github.com/AcelateOrg/beam-bridge) still builds
   against it.
5. Open a pull request that explains what changed and why.

## License

This repository is under the [Mozilla Public License 2.0](LICENSE).
By contributing you agree that your contribution is licensed under MPL-2.0.
Keep existing copyright and license notices in files you modify.
