# Contributing to paperflow

## Development setup

Prerequisites: Go 1.24+

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

## Making changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Add or update tests as needed
4. Run `go test -race ./...` and `golangci-lint run`
5. Open a pull request against `main`

CI runs tests on both Linux and macOS, plus linting. All checks must pass before merging.

## Commit messages

Use conventional-ish prefixes to keep history scannable:

- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation only
- `chore:` maintenance, CI, dependencies
- `refactor:` code changes that don't add features or fix bugs

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions.

To cut a release:

```bash
# 1. Make sure main is clean and CI is green
git checkout main
git pull

# 2. Tag the release
git tag v0.3.0
git push origin v0.3.0
```

This triggers the release workflow which:

1. Builds binaries for linux/darwin x amd64/arm64
2. Creates a GitHub release with changelog
3. Updates the [Homebrew tap](https://github.com/alcxyz/homebrew-tap)
4. Publishes to the [AUR](https://aur.archlinux.org/packages/paperflow-bin) (`paperflow-bin`)

Version is injected into the binary via `-ldflags -X main.version=v<tag>`.

### Version numbering

Follow [semver](https://semver.org/):

- **Patch** (`v0.2.x`): bug fixes, minor tweaks
- **Minor** (`v0.x.0`): new features, non-breaking changes
- **Major** (`vx.0.0`): breaking changes to config format, CLI flags, or behavior

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
