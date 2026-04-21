# ADR-003: Filesystem event deduplication with time window

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/watcher/watcher.go`

## Context

fsnotify delivers raw filesystem events, and a single user action (e.g., saving a file) can produce multiple events. macOS is particularly noisy, emitting Create, Write, and Chmod events for a single file drop. Without deduplication, paperflow would attempt to sort and ingest the same file multiple times, causing errors or duplicate uploads.

## Decision

Deduplicate events using a per-file timer with a 500ms window. When an event arrives for a file, any pending timer for that file is reset. The file is only processed after 500ms of quiet. This coalesces bursts of events into a single processing action.

## Alternatives Considered

- **Process every event idempotently:** Let the organizer handle duplicates by checking if the file was already moved. This would work but wastes I/O on stat calls and complicates error handling when a file disappears mid-processing.
- **Debounce globally (single timer for all events):** Simpler but delays processing of unrelated files. A burst of events for file A would delay processing of file B.
- **Longer/shorter window:** 500ms is a pragmatic middle ground. Shorter windows miss slow write completion on network mounts; longer windows add noticeable latency to the user experience.

## Consequences

- Each watched file has a small memory footprint (timer + path string) during the debounce window.
- Files that are still being written to when the 500ms window expires may be processed before writing completes. This is acceptable because most use cases involve completed files being dropped into the watch directory.
- The 500ms value is hardcoded. If this proves problematic on specific platforms or storage backends, it could be made configurable.
