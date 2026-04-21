# ADR-012: VERSION file with CI auto-tagging

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `VERSION`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yml`, `flake.nix`

## Context

The previous approach ([ADR-011](ADR-011-git-derived-version.md)) derived versions from git metadata at build time, with releases requiring manual tag creation and pushing. This meant cutting a release was a multi-step process (`git tag vX.Y.Z && git push origin vX.Y.Z`) that was easy to fat-finger or forget. The grove project solved this with a `VERSION` file and CI auto-tagging, making releases a matter of bumping one file and merging.

## Decision

Adopt the same release workflow as grove:

1. A `VERSION` file at the repo root is the single source of truth for the current version.
2. CI (`ci.yml`) reads `VERSION` on every push to `main`. If the corresponding `v`-prefixed tag doesn't exist, it creates the tag and runs GoReleaser.
3. GoReleaser handles all distribution: GitHub Releases, Homebrew tap, and AUR (via native `aurs` section).
4. A separate `release.yml` exists as a fallback for manually re-triggering a release by pushing a tag.
5. Nix flake reads version from `VERSION` instead of `self.shortRev`.

The release process becomes: bump `VERSION` on `dev`, merge `dev` into `main`, done.

## Alternatives Considered

- **Keep manual tagging:** Works but adds friction and diverges from how grove operates. Consistency across projects reduces cognitive load.
- **GitHub Release UI:** Creates tags but doesn't integrate with the existing GoReleaser pipeline.
- **Commit-message-based version bumps (e.g. semantic-release):** Adds a heavy dependency and convention that the project doesn't need at this scale.

## Consequences

- Releasing is a one-step merge instead of a multi-step tag dance.
- The `VERSION` file must be bumped before merging to `main`; forgetting means no new release (safe failure mode — CI just skips tagging).
- AUR publishing moves from a custom CI job with manual PKGBUILD/SRCINFO generation to GoReleaser's native `aurs` publisher, removing the `aur/` directory.
- Both paperflow and grove now follow the same release workflow, reducing context-switching.
