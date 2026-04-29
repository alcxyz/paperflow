package ingest

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Archiver moves files from the ingest directory to an archive directory
// after a configurable delay. This prevents Paperless-ngx from re-ingesting
// files if it restarts and re-scans its consume directory.
type Archiver struct {
	archiveDir string
	delay      time.Duration

	mu      sync.Mutex
	pending map[string]*time.Timer
}

// NewArchiver creates an Archiver. If archiveDir is empty, returns nil
// (feature disabled). The caller should nil-check before calling methods.
func NewArchiver(archiveDir string, delayStr string) (*Archiver, error) {
	if archiveDir == "" {
		return nil, nil
	}
	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		return nil, fmt.Errorf("parsing ingest_archive_after %q: %w", delayStr, err)
	}
	return &Archiver{
		archiveDir: archiveDir,
		delay:      delay,
		pending:    make(map[string]*time.Timer),
	}, nil
}

// Schedule queues a file for archival after the configured delay.
func (a *Archiver) Schedule(ingestPath string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	timer := time.AfterFunc(a.delay, func() {
		a.archiveFile(ingestPath)
	})
	a.pending[ingestPath] = timer
	log.Printf("archive scheduled for %s in %s", filepath.Base(ingestPath), a.delay)
}

// Close flushes all pending archives immediately (for graceful shutdown).
func (a *Archiver) Close() {
	a.mu.Lock()
	pending := make(map[string]*time.Timer, len(a.pending))
	for k, v := range a.pending {
		pending[k] = v
	}
	a.mu.Unlock()

	for path, timer := range pending {
		timer.Stop()
		a.archiveFile(path)
	}
}

// archiveFile moves a single file from the ingest dir to the archive dir.
func (a *Archiver) archiveFile(ingestPath string) {
	a.mu.Lock()
	delete(a.pending, ingestPath)
	a.mu.Unlock()

	filename := filepath.Base(ingestPath)

	// File may already be consumed by Paperless.
	if _, err := os.Stat(ingestPath); os.IsNotExist(err) {
		log.Printf("archive: %s already consumed, skipping", filename)
		return
	}

	if err := os.MkdirAll(a.archiveDir, 0755); err != nil {
		log.Printf("archive: failed to create dir %s: %v", a.archiveDir, err)
		return
	}

	ts := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf("%s_%s", ts, filename)
	destPath := filepath.Join(a.archiveDir, archiveName)
	destPath = ResolveCollision(destPath)

	if err := moveFile(ingestPath, destPath); err != nil {
		log.Printf("archive: failed to move %s: %v", filename, err)
		return
	}

	log.Printf("archived %s -> %s", filename, destPath)
}

// moveFile moves src to dst, falling back to copy+remove for cross-device moves.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
