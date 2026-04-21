# ADR-006: Pluggable ingestion via interface pattern

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/ingest/`

## Context

Paperless-ngx supports two ingestion methods: watching a consumption directory for new files, and uploading via REST API. Users have different setups — some run paperflow on the same machine as Paperless (directory mode is simpler), others run it remotely (API mode is necessary). A third option is to use paperflow purely as a file organizer without any Paperless integration.

## Decision

Define ingestion as a Go interface with two implementations: `DirectoryIngester` (copies files to a watched directory) and `APIIngester` (uploads via multipart HTTP POST). A `none` mode skips ingestion entirely. The active implementation is selected at startup based on configuration.

## Alternatives Considered

- **API only:** Would simplify the code but force all users through HTTP even when files are already local. Directory mode avoids network overhead and Paperless handles the file directly.
- **Directory only:** Would lock out remote users and those who prefer not to expose a shared directory.
- **Plugin system (dynamic loading):** Massive overkill for two well-defined implementations. The interface pattern gives the same extensibility with compile-time safety.

## Consequences

- Adding a new ingestion method (e.g., S3, WebDAV) requires implementing one interface and adding a config option. No changes to the organizer or watcher.
- The interface is deliberately narrow: `Ingest(filePath string) error`. This keeps implementations simple and testable.
- Ingest errors are logged but do not fail the sort operation. A file that was sorted successfully but failed to ingest remains in its sorted location for manual handling or retry.
