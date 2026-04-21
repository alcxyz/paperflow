# ADR-001: Single binary with minimal dependencies

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `go.mod`, build system

## Context

Paperflow targets end-users who want a simple file organizer that integrates with Paperless-ngx. The tool needs to run on Linux and macOS across x86_64 and ARM64. Users should not need to install runtimes, manage virtual environments, or resolve dependency conflicts.

## Decision

Build paperflow as a single statically-linked Go binary with only two external dependencies: `github.com/BurntSushi/toml` for config parsing and `github.com/fsnotify/fsnotify` for filesystem watching. All other functionality (HTTP client, service management, notifications) uses the Go standard library. CGO is disabled in release builds.

## Alternatives Considered

- **Python with packaging (PyInstaller/shiv):** Would simplify development but produce larger binaries, require bundling a runtime, and complicate cross-compilation. Rejected for distribution complexity.
- **Rust:** Similar single-binary benefits but higher development friction for a tool of this scope. Go's standard library covers HTTP, file I/O, and process management without additional crates.
- **More Go libraries (cobra, viper, etc.):** Would reduce boilerplate but add transitive dependencies. The CLI surface is small enough that flag parsing and subcommand routing work fine with stdlib.

## Consequences

- Distribution is straightforward: one binary per platform, no runtime prerequisites.
- Adding features that need external libraries requires deliberate justification.
- Platform-specific behavior (notifications, service management) must be handled with build tags and `os/exec` calls rather than higher-level abstractions.
