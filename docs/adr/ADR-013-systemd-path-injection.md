# ADR-013: Inject PATH into generated systemd unit

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `cmd/paperflow/service_cmd.go`

## Context

Desktop notifications use `notify-send`, which paperflow invokes by name via `exec.Command`. On NixOS (and similar systems where tools live outside standard `/usr/bin` paths), the systemd user service's default `PATH` does not include the user's profile paths (e.g. `~/.nix-profile/bin`). This caused notifications to silently fail with `executable file not found in $PATH`.

## Decision

Capture the user's current `PATH` at `paperflow service install` time and write it as an `Environment=PATH=...` directive in the generated systemd unit file. This makes all tools available to the service at the same paths the user had when they installed it.

## Alternatives Considered

- **Hardcode notify-send path at compile time:** Doesn't work across distributions; the path varies.
- **Look up notify-send at startup with extra search paths:** Fragile — requires maintaining a list of known paths and still misses non-standard layouts.
- **Require users to edit the unit file manually:** Defeats the purpose of `service install`.

## Consequences

- `notify-send` (and any future external tools) work out of the box when running as a systemd service.
- If the user's PATH changes significantly (e.g. new Nix generation with different store paths), they need to re-run `paperflow service install` to update the unit.
- The generated unit file is now environment-specific rather than fully portable, which is acceptable since it's generated per-machine anyway.
