# Contributing to paperflow

## Development setup

Prerequisites: Go 1.26+

```bash
git clone https://github.com/alcxyz/paperflow.git
cd paperflow
go build ./cmd/paperflow
```

## Running tests

```bash
go test -race ./...
```

## Linting

CI runs [golangci-lint](https://golangci-lint.run/). Install it locally to catch issues before pushing:

```bash
golangci-lint run
```

## Project structure

- `cmd/paperflow/` -- entry point, CLI parsing, subcommands
- `internal/config/` -- config loading (TOML, XDG paths)
- `internal/watcher/` -- filesystem watching (fsnotify)
- `internal/organizer/` -- file sorting by type and date
- `internal/ingest/` -- Paperless-ngx ingestion (API and directory)
- `internal/notify/` -- batched desktop notifications
- `internal/bucket/` -- extension-to-bucket mapping
- `internal/fileops/` -- atomic copies, moves, and collision-free paths
- `internal/buildinfo/` -- development build identity

## Making changes

1. Fork the repo and create a branch from `dev`
2. Make your changes
3. Add or update tests as needed
4. Run `gofmt`, `go test -race ./...`, `go vet ./...`, and `golangci-lint run`
5. Open a pull request against `dev`

CI checks formatting, module tidiness, build, vet, lint, race tests, and
`govulncheck`, then builds release snapshots and the Nix package. All checks
must pass before merging.

If you change Go dependencies, update `vendorHash` in `default.nix`; the CI Nix
job fails with the expected hash when it is stale.

## Commit messages

Use conventional-ish prefixes to keep history scannable:

- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation only
- `chore:` maintenance, CI, dependencies
- `refactor:` code changes that don't add features or fix bugs

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions. The `VERSION` file is the single source of truth.

Development builds identify their source revision. Ordinary `go build` output
uses `dev-<commit>[-dirty]`; branch-based Nix packages use
`X.Y.Z-dev.<commit>[.dirty]`. GoReleaser artifacts retain the plain `X.Y.Z`
release version; an intentional Nix release build uses `.#release` from clean,
identified release source.

To cut a release:

1. Bump the `VERSION` file on `dev`
2. Merge `dev` into `main`
3. CI automatically creates the git tag and runs GoReleaser

This builds binaries for linux/darwin x amd64/arm64, creates a GitHub release with changelog, updates the [Homebrew tap](https://github.com/alcxyz/homebrew-tap), and publishes to the [AUR](https://aur.archlinux.org/packages/paperflow-bin) (`paperflow-bin`).

CI builds releases with the latest stable Go release; `go.mod` only states the
minimum supported version.

### Version numbering

Follow [semver](https://semver.org/):

- **Patch** (`v0.2.x`): bug fixes, minor tweaks
- **Minor** (`v0.x.0`): new features, non-breaking changes
- **Major** (`vx.0.0`): breaking changes to config format, CLI flags, or behavior

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
