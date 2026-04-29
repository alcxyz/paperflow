package watcher

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/ingest"
	"github.com/alcxyz/paperflow/internal/notify"
	"github.com/alcxyz/paperflow/internal/organizer"

	"github.com/fsnotify/fsnotify"
)

// dedupWindow is the duration within which duplicate events for the same path
// are suppressed. fsnotify commonly fires multiple events for a single file
// operation (especially on macOS).
const dedupWindow = 500 * time.Millisecond

// Watcher monitors a directory for new files and processes them.
type Watcher struct {
	config    *config.Config
	organizer *organizer.Organizer
	notifier  *notify.Notifier
	archiver  *ingest.Archiver

	mu   sync.Mutex
	seen map[string]time.Time
}

// NewWatcher creates a Watcher with the given config.
func NewWatcher(cfg *config.Config) (*Watcher, error) {
	archiver, err := ingest.NewArchiver(cfg.IngestArchiveDir, cfg.IngestArchiveAfter)
	if err != nil {
		return nil, fmt.Errorf("creating archiver: %w", err)
	}
	return &Watcher{
		config:    cfg,
		organizer: organizer.NewOrganizer(cfg, archiver),
		notifier:  notify.NewNotifier(cfg),
		archiver:  archiver,
		seen:      make(map[string]time.Time),
	}, nil
}

// Run starts watching cfg.WatchDir for Create and Rename events.
// It blocks until SIGINT or SIGTERM is received.
func (w *Watcher) Run() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = fsw.Close() }()

	if err := fsw.Add(w.config.WatchDir); err != nil {
		w.notifier.Send("Failed to start", fmt.Sprintf("Cannot watch %s: %v", w.config.WatchDir, err))
		return err
	}

	// Verify Paperless API authentication on startup.
	if w.config.Ingest == "api" {
		const maxRetries = 3
		var lastErr error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			log.Printf("checking Paperless API connection (attempt %d/%d)...", attempt, maxRetries)
			lastErr = ingest.CheckAPI(w.config.PaperlessURL, w.config.Token)
			if lastErr == nil {
				break
			}
			if attempt == 1 {
				w.notifier.Send("Startup issue", fmt.Sprintf("API auth failed, retrying: %v", lastErr))
			}
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt*5) * time.Second)
			}
		}
		if lastErr != nil {
			return fmt.Errorf("paperless API check failed after %d attempts: %w", maxRetries, lastErr)
		}
		log.Printf("Paperless API authenticated successfully")
	}

	log.Printf("watching %s", w.config.WatchDir)

	// Send startup notification.
	mode := w.config.Ingest
	if mode == "none" {
		mode = "sort only"
	}
	w.notifier.Send("Started", fmt.Sprintf("Watching %s (%s)", w.config.WatchDir, mode))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			w.handleEvent(event.Name)

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)

		case sig := <-sigCh:
			log.Printf("received %s, shutting down", sig)
			if w.archiver != nil {
				w.archiver.Close()
			}
			w.notifier.Close()
			return nil
		}
	}
}

// handleEvent processes a single file event.
func (w *Watcher) handleEvent(path string) {
	// Deduplicate events for the same path within the dedup window.
	w.mu.Lock()
	if lastSeen, ok := w.seen[path]; ok && time.Since(lastSeen) < dedupWindow {
		w.mu.Unlock()
		return
	}
	w.seen[path] = time.Now()
	// Prune old entries to prevent unbounded growth.
	for p, t := range w.seen {
		if time.Since(t) > dedupWindow*2 {
			delete(w.seen, p)
		}
	}
	w.mu.Unlock()

	// Only process files at the root of WatchDir (not in subdirectories).
	if filepath.Dir(path) != w.config.WatchDir {
		return
	}

	filename := filepath.Base(path)

	// Skip dotfiles.
	if strings.HasPrefix(filename, ".") {
		return
	}

	// Skip excluded patterns.
	for _, pattern := range w.config.Exclude.Patterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			log.Printf("skipping excluded file: %s", filename)
			return
		}
	}

	// Check file exists (may have been moved by a prior event).
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	// Skip directories (e.g. bucket dirs created by the organizer).
	if info.IsDir() {
		return
	}
	if info.Size() == 0 {
		log.Printf("skipping empty file: %s", filename)
		return
	}

	result, err := w.organizer.ProcessFile(path)
	if err != nil {
		log.Printf("error processing %s: %v", filename, err)
		return
	}
	w.notifier.Notify(result)
}
