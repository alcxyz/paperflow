# ADR-009: Buffer multipart form in memory instead of streaming

**Status:** Accepted
**Date:** 2026-04-21
**Applies to:** `internal/ingest/api.go`

## Context

The API ingester uploads files to Paperless-ngx via multipart form POST. The initial implementation used `io.Pipe` to stream the multipart form body directly into the HTTP request, avoiding buffering the entire file in memory. However, this introduced a race condition: the HTTP request could start before the file data was fully written to the pipe, causing Paperless to reject the upload with "No file was submitted" errors.

## Decision

Buffer the entire multipart form body in memory before sending the HTTP request. The file content and multipart boundaries are written to a `bytes.Buffer`, then the buffer is used as the request body.

## Alternatives Considered

- **io.Pipe with synchronization:** Could fix the race with a WaitGroup or channel to ensure the writer goroutine completes before the request sends. Adds concurrency complexity for marginal memory savings — paperflow processes documents, not multi-gigabyte files.
- **Temporary file on disk:** Write the multipart form to a temp file, then stream from disk. Avoids memory pressure but adds I/O and cleanup logic for no practical benefit at typical document sizes.
- **Chunked/resumable upload:** Paperless-ngx doesn't support this, so not an option.

## Consequences

- Uploads are reliable regardless of timing.
- Memory usage scales with file size during upload. For typical documents (PDFs, images, office files under ~50MB) this is negligible.
- Very large files could cause memory pressure. This is acceptable because Paperless-ngx itself has practical limits on document size.
