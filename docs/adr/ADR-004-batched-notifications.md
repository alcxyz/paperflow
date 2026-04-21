# ADR-004: Timer-based batched desktop notifications

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/notify/`

## Context

When a user drops multiple files into the watch directory at once (e.g., dragging a folder of scanned documents), paperflow processes each file individually. Sending a desktop notification per file would flood the notification center with dozens of identical-looking alerts, which is disruptive and unhelpful.

## Decision

Batch notifications using a timer-based approach. When a file is processed, its result is added to a pending batch. A timer (default 3 seconds) starts or resets on each addition. When the timer fires (3 seconds of quiet), a single summary notification is sent listing all files in the batch. Separate batches are maintained for sorted and ingested files.

## Alternatives Considered

- **One notification per file:** Simplest implementation but creates notification spam during bulk operations.
- **Count-based batching (e.g., every 10 files):** Would leave stale partial batches if fewer files arrive than the threshold. Timer-based avoids this by always flushing after a quiet period.
- **No notifications, log only:** Loses the "set and forget" UX where users get confirmation that files were handled without checking logs.

## Consequences

- Bulk file drops produce a single, readable summary notification.
- There is a minimum 3-second delay before any notification appears, even for single files. This is acceptable for a background service.
- Platform-specific notification commands (`notify-send` on Linux, `osascript` on macOS) are called via `os/exec`, keeping the dependency footprint minimal.
