# Architecture Decision Records

| ADR | Title | Area |
|---|---|---|
| [ADR-001](ADR-001-single-binary-minimal-deps.md) | Single binary with minimal dependencies | build, dependencies |
| [ADR-002](ADR-002-token-file-separation.md) | Separate token file with strict permissions | config, security |
| [ADR-003](ADR-003-event-deduplication.md) | Filesystem event deduplication with time window | watcher |
| [ADR-004](ADR-004-batched-notifications.md) | Timer-based batched desktop notifications | notify |
| [ADR-005](ADR-005-cross-device-move-fallback.md) | Cross-device move fallback to copy+remove | organizer |
| [ADR-006](ADR-006-pluggable-ingestion.md) | Pluggable ingestion via interface pattern | ingest |
| [ADR-007](ADR-007-systemd-launchd-service-generation.md) | Generated systemd/launchd service definitions | service |
| [ADR-008](ADR-008-config-hierarchy.md) | Config hierarchy: defaults, file, env vars, flags | config |
| [ADR-009](ADR-009-buffered-multipart-upload.md) | Buffer multipart form in memory instead of streaming | ingest |
| [ADR-010](ADR-010-startup-auth-with-retry.md) | Fail-fast API auth with retry and backoff | ingest, startup |
| [ADR-011](ADR-011-git-derived-version.md) | ~~Version derived from git, no manual bumping~~ (superseded by ADR-012) | build |
| [ADR-012](ADR-012-version-file-auto-tag.md) | VERSION file with CI auto-tagging | build, CI |
| [ADR-013](ADR-013-systemd-path-injection.md) | Inject PATH into generated systemd unit | service |
| [ADR-014](ADR-014-ingest-archive.md) | Archive ingested files from consume directory | ingest |
