# ADR-007: Generated systemd/launchd service definitions

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `cmd/paperflow/service_cmd.go`

## Context

Paperflow runs as a long-lived background process. Users need a way to start it at login and restart it on failure. Linux uses systemd and macOS uses launchd, each with different configuration formats. Asking users to write service files manually is error-prone and creates a support burden.

## Decision

The `paperflow service install` command generates and installs a platform-appropriate service definition (systemd user unit on Linux, launchd plist on macOS). CLI flags passed to the install command are embedded in the generated service definition, so the service runs with the same configuration the user tested interactively. The service runs as a user-level unit (not system-level), requiring no elevated privileges.

## Alternatives Considered

- **Ship static service files in the repo:** Would not reflect user-specific paths or flags. Every user's binary location and config differ.
- **Require manual service setup:** Documented in a README section. Rejected because it is the most common source of "it doesn't work" issues in similar tools.
- **Supervisor/process manager dependency:** Tools like supervisord or pm2 add unnecessary runtime dependencies and contradict the single-binary philosophy (ADR-001).

## Consequences

- Installation is a single command with no manual file editing.
- User-level services mean no `sudo` or root access is needed.
- Changes to the service (e.g., different flags) require `service uninstall` then `service install` with new flags. There is no in-place update command.
- The generated service definitions are minimal and opinionated (e.g., `Restart=on-failure` for systemd). Users who need custom service behavior can use the generated file as a starting point.
