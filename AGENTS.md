# Paperflow repository instructions

- GitHub `origin` is the development, CI, and release host. Forgejo is a secondary continuity remote; use the
  GitHub workflow in [CONTRIBUTING.md](CONTRIBUTING.md).
- Develop from `dev` and target GitHub pull requests at `dev`; release by updating `VERSION` on `dev` before
  integrating to `main`.
- Read the [ADR index](docs/adr/README.md) and relevant accepted decisions before lasting design changes. ADR-011
  is superseded by [ADR-012](docs/adr/ADR-012-version-file-auto-tag.md) for versioning.
- Keep watcher, organizer, ingestion, and notification behavior in their existing `internal` packages; see the
  package map in [CONTRIBUTING.md](CONTRIBUTING.md).
- For Go changes, run `gofmt`, `go test -race ./...`, `go vet ./...`, and `golangci-lint run`; CI also checks the
  build, Go module tidiness, release archives, and Nix package. See
  [.github/workflows/ci.yml](.github/workflows/ci.yml).
