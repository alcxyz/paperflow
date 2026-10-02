package ingest

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alcxyz/paperflow/internal/fileops"
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

// NewArchiver creates an Archiver. If archiveDir is empty, it returns nil
// (feature disabled); a nil Archiver's methods are no-ops.
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
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if prev, ok := a.pending[ingestPath]; ok {
		prev.Stop()
	}
	timer := time.AfterFunc(a.delay, func() {
		a.archiveFile(ingestPath)
	})
	a.pending[ingestPath] = timer
	log.Printf("archive scheduled for %s in %s", filepath.Base(ingestPath), a.delay)
}

// Close flushes all pending archives immediately (for graceful shutdown).
func (a *Archiver) Close() {
	if a == nil {
		return
	}

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
	destPath = fileops.UniquePath(destPath)

	if err := fileops.Move(ingestPath, destPath); err != nil {
		log.Printf("archive: failed to move %s: %v", filename, err)
		return
	}

	log.Printf("archived %s -> %s", filename, destPath)
}
