# ADR-008: Config hierarchy: defaults, file, env vars, flags

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/config/config.go`, `cmd/paperflow/main.go`

## Context

Paperflow needs to work in multiple environments: interactive use on a developer's machine, headless service mode, containers, and CI pipelines. Each context has different ergonomics for providing configuration. A fixed config file works for personal machines but not for containers. Environment variables work for containers but are tedious for interactive use.

## Decision

Apply configuration in a four-level hierarchy where each level overrides the previous:

1. **Compiled defaults** — sensible values for all settings so paperflow works out of the box.
2. **Config file** (`~/.config/paperflow/config.toml`) — persistent user preferences, XDG-compliant path.
3. **Environment variables** (`PAPERFLOW_*` prefix) — container and CI overrides without touching files.
4. **CLI flags** — one-off overrides for testing or debugging.

## Alternatives Considered

- **Config file only:** Simple but unusable in containers where mounting a config file is friction. Also makes one-off testing (e.g., dry-run with different path) require editing the file.
- **Environment variables only:** Works for 12-factor apps but is cumbersome for users with many settings. TOML is more readable for complex configs like bucket definitions.
- **Viper/koanf library:** These Go config libraries implement similar hierarchies but bring transitive dependencies and behavior that is more complex than needed. The custom implementation is ~100 lines and fully transparent.

## Consequences

- Users can start with just defaults, add a config file when they want persistence, and override specific values in specific contexts.
- The `PAPERFLOW_` prefix avoids collisions with other tools' environment variables.
- The hierarchy is implicit: there is no command to show "where did this value come from." If debugging config resolution becomes a common support issue, a `paperflow config show --verbose` command could be added.
