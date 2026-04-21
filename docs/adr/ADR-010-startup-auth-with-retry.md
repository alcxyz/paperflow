# ADR-010: Fail-fast API auth check on startup with retry and backoff

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `cmd/paperflow/main.go`, `internal/ingest/api.go`

## Context

When running in API ingest mode, a misconfigured URL or invalid token would only surface when the first file was ingested — potentially hours after startup. Users running paperflow as a service would not notice the silent failure until documents piled up unsorted. An initial implementation that crashed immediately on auth failure caused a different problem: systemd/launchd restart loops produced repeated desktop notifications, spamming the user.

## Decision

Verify the Paperless-ngx API connection and token validity at startup before starting the file watcher. If the check fails, retry up to 3 times with increasing backoff. Send a desktop notification only on the first failure. If all retries fail, exit with a clear error. This gives transient issues (Paperless restarting, network blip) a chance to resolve while still failing fast on genuine misconfiguration.

## Alternatives Considered

- **No startup check, fail on first ingest:** The original behavior. Users discover problems too late and debugging requires reading logs.
- **Crash immediately on first failure:** Correct but hostile to service managers. A systemd `Restart=on-failure` loop with a notification on each attempt floods the desktop.
- **Infinite retry with backoff:** Would mask permanent misconfiguration. The user might never realize paperflow isn't ingesting. Bounded retries strike a balance.

## Consequences

- Misconfigured credentials are caught within seconds of startup, not hours later.
- Transient Paperless unavailability (e.g., during a container restart) is tolerated for a short window.
- The retry count (3) and backoff schedule are hardcoded. If users need longer tolerance windows, these could be made configurable.
