# ADR-005: Cross-device move fallback to copy+remove

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/organizer/organizer.go`

## Context

`os.Rename` performs an atomic move on the same filesystem but fails with `EXDEV` (cross-device link) when source and destination are on different mount points. This is a realistic scenario: users may watch a download directory on one partition and sort into a NAS-mounted destination, or use tmpfs for incoming files.

## Decision

Attempt `os.Rename` first. If it fails with a cross-device link error, fall back to copying the file content and then removing the original. The copy preserves file content but not filesystem-level metadata like extended attributes.

## Alternatives Considered

- **Always copy+remove:** Would work universally but is slower for same-device moves where rename is O(1). The common case (same device) should be fast.
- **Require same filesystem:** Would simplify the code but impose an artificial constraint on users' directory layout. This is a tool for real-world file organization where mixed mount points are common.
- **Use a library for cross-platform file operations:** Available Go libraries either add unnecessary dependencies or don't handle this specific case better than the ~15 lines of fallback code.

## Consequences

- Same-device moves remain atomic and instantaneous.
- Cross-device moves are not atomic: a crash during copy leaves both source and partial destination. This is acceptable because paperflow operates on non-critical files (documents being organized, not database transactions).
- Extended attributes, ACLs, and hardlinks are not preserved on cross-device moves. Standard file permissions are preserved.
