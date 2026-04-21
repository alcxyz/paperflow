# ADR-002: Separate token file with strict permissions

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/config/config.go`

## Context

Paperflow needs a Paperless-ngx API token for API-based ingestion. The config file (`config.toml`) contains non-sensitive settings like paths and bucket definitions. Storing the token alongside these settings risks accidental exposure when sharing configs, committing dotfiles, or debugging.

## Decision

Store the API token in a separate file (`~/.config/paperflow/token`) with 0600 permissions. The config file references this implicitly by convention rather than containing the secret. On load, paperflow checks the token file permissions and warns if they are more permissive than 0600.

## Alternatives Considered

- **Token in config.toml:** Simpler to implement but mixes secrets with shareable configuration. Users who version-control their dotfiles would risk leaking credentials.
- **System keychain (libsecret/Keychain):** More secure but requires CGO or platform-specific binaries, conflicting with the single-binary goal (ADR-001). Also adds setup friction.
- **Environment variable only:** Viable but less ergonomic for service mode where env vars must be embedded in service definitions. The file approach works naturally with both interactive and service usage.

## Consequences

- Users can safely share or version-control `config.toml` without leaking credentials.
- The token file permission check provides defense-in-depth against misconfiguration.
- Environment variable override (`PAPERFLOW_TOKEN`) is still supported for CI or container use cases.
