# ADR-011: Version derived from git, no manual bumping

**Status:** Superseded by [ADR-012](ADR-012-version-file-auto-tag.md)
**Date:** 2026-04-21
**Applies to:** `cmd/paperflow/main.go`, `default.nix`, `flake.nix`, `.goreleaser.yml`

## Context

Early versions hardcoded the version string in `default.nix`, requiring a manual bump on every release. This was easy to forget and created a class of "wrong version" bugs where the binary reported a stale version. The project is built through multiple paths: `nix build`, `go install`, `go build`, and GoReleaser for releases.

## Decision

Inject the version at build time via Go's `-ldflags -X` mechanism. Each build path provides the version differently:

- **GoReleaser** (releases): injects the git tag automatically.
- **Nix flake**: passes `self.shortRev` (the git short hash) as the version.
- **`default.nix`**: accepts version as a parameter, defaults to `"dev"`.
- **Plain `go build`**: gets `"dev"` (the default value in `main.go`).

No file in the repo contains a hardcoded version string that needs manual updating.

## Alternatives Considered

- **Hardcoded version constant:** Simple but requires discipline to update. Forgotten bumps are a recurring issue in every project that tries this.
- **`go generate` with a version file:** Would work but adds a build step that can be skipped. ldflags is the standard Go approach.
- **`debug.ReadBuildInfo()` for VCS metadata:** Go 1.18+ embeds git info automatically, but only when built with `go install` or `go build` in a git checkout. Doesn't work for Nix builds or GoReleaser. ldflags works uniformly across all build paths.

## Consequences

- Releases always report the correct version with zero manual steps.
- Local development builds show `"dev"`, which is clear and unambiguous.
- Nix builds show the git short hash, which is traceable but not a semver string.
- The version source differs by build path, which could confuse users running `paperflow --version` if they don't know how it was built. This is acceptable since release binaries (the primary distribution) always have correct semver tags.
