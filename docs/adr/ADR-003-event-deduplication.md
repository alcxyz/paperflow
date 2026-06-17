# ADR-003: Filesystem event deduplication with quiet period

**Status:** Accepted
**Date:** 2026-04-21
**Updated:** 2026-06-17
**Applies to:** `internal/watcher/watcher.go`

## Context

fsnotify delivers raw filesystem events, and a single user action (e.g., saving a file) can produce multiple events. macOS is particularly noisy, emitting Create, Write, and Chmod events for a single file drop. Without deduplication, paperflow would attempt to sort and ingest the same file multiple times, causing errors or duplicate uploads.

## Decision

Deduplicate events using a per-file timer with a configurable quiet period (`settle_delay`, default `2s`). When a create, rename, or write event arrives for a file, any pending timer for that file is reset. The file is only processed after the quiet period elapses. This coalesces bursts of events into a single processing action and gives placeholder files time to receive their content before paperflow moves or ingests them.

## Alternatives Considered

- **Process every event idempotently:** Let the organizer handle duplicates by checking if the file was already moved. This would work but wastes I/O on stat calls and complicates error handling when a file disappears mid-processing.
- **Debounce globally (single timer for all events):** Simpler but delays processing of unrelated files. A burst of events for file A would delay processing of file B.
- **Longer/shorter window:** `2s` avoids common zero-byte placeholder races while keeping the tool responsive. Slower storage, scanners, or sync clients can increase `settle_delay`; users who prefer immediate local-only processing can reduce it.

## Consequences

- Each watched file has a small memory footprint (timer + path string) during the debounce window.
- Files that are still being written to when `settle_delay` expires may be processed before writing completes. This remains possible for very slow writers that do not emit write events, but the default greatly reduces the common placeholder race.
- The quiet period is configurable through `settle_delay`, `PAPERFLOW_SETTLE_DELAY`, or `--settle-delay`.
