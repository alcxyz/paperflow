# ADR-014: Archive ingested files from consume directory

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/ingest/archiver.go`, `internal/config/config.go`

## Context

When using directory-mode ingestion, paperflow copies files into Paperless-ngx's consume directory. If Paperless restarts (e.g. container redeployment, auto-update), it re-scans the consume directory and re-ingests any files still present, potentially creating duplicates. API mode does not have this problem since uploads are fire-and-forget.

## Decision

Add an optional archiver that moves files from the consume directory to a separate archive directory after a configurable delay. The delay gives Paperless time to pick up the file first.

Configuration:
- `ingest_archive_dir`: path to the archive directory (empty = disabled, the default)
- `ingest_archive_after`: delay before archiving (default: `"5m"`)

Archive filenames use a flat layout with timestamp prefix: `20260421-164532_invoice.pdf`. The original organized copy in the watch directory is untouched.

If the file is already gone when the timer fires (Paperless consumed and deleted it), the archiver silently skips. On shutdown, all pending archives are flushed immediately.

## Alternatives Considered

- **Delete from consume dir after delay:** Simpler but loses the safety net. If Paperless missed a file, it's gone.
- **Document only, no mitigation:** Honest but doesn't solve the problem for users who can't switch to API mode.
- **Use API mode instead:** Sidesteps the problem entirely but requires network access to Paperless. Some deployments use local directory mounts by design.

## Consequences

- Directory-mode users can opt in to archive and avoid re-ingestion on Paperless restart.
- The archive directory serves as an audit trail of what was sent to Paperless.
- Files can be manually re-ingested from the archive if needed.
- Disabled by default — zero behavior change for existing users.
- Adds a per-file timer goroutine; memory is bounded by the number of files ingested within the delay window.
