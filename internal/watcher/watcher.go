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

const defaultSettleDelay = 2 * time.Second

// Watcher monitors a directory for new files and processes them.
type Watcher struct {
	config    *config.Config
	organizer *organizer.Organizer
	notifier  *notify.Notifier
	archiver  *ingest.Archiver

	settleDelay time.Duration

	mu      sync.Mutex
	pending map[string]*pendingEvent
}

type pendingEvent struct {
	timer *time.Timer
}

// NewWatcher creates a Watcher with the given config.
func NewWatcher(cfg *config.Config) (*Watcher, error) {
	settleDelay := defaultSettleDelay
	if cfg.SettleDelay != "" {
		parsed, err := time.ParseDuration(cfg.SettleDelay)
		if err != nil {
			return nil, fmt.Errorf("parsing settle_delay %q: %w", cfg.SettleDelay, err)
		}
		if parsed < 0 {
			return nil, fmt.Errorf("settle_delay must not be negative: %s", cfg.SettleDelay)
		}
		settleDelay = parsed
	}

	archiver, err := ingest.NewArchiver(cfg.IngestArchiveDir, cfg.IngestArchiveAfter)
	if err != nil {
		return nil, fmt.Errorf("creating archiver: %w", err)
	}
	return &Watcher{
		config:      cfg,
		organizer:   organizer.NewOrganizer(cfg, archiver),
		notifier:    notify.NewNotifier(cfg),
		archiver:    archiver,
		settleDelay: settleDelay,
		pending:     make(map[string]*pendingEvent),
	}, nil
}

// Run starts watching cfg.WatchDir for Create, Rename, and Write events.
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
			if event.Op&(fsnotify.Create|fsnotify.Rename|fsnotify.Write) == 0 {
				continue
			}
			w.scheduleEvent(event.Name)

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)

		case sig := <-sigCh:
			log.Printf("received %s, shutting down", sig)
			w.cancelPending()
			if w.archiver != nil {
				w.archiver.Close()
			}
			w.notifier.Close()
			return nil
		}
	}
}

func (w *Watcher) scheduleEvent(path string) {
	if w.settleDelay == 0 {
		go w.handleEvent(path)
		return
	}

	w.mu.Lock()
	if pending, ok := w.pending[path]; ok {
		pending.timer.Stop()
	}

	pending := &pendingEvent{}
	pending.timer = time.AfterFunc(w.settleDelay, func() {
		w.mu.Lock()
		if w.pending[path] != pending {
			w.mu.Unlock()
			return
		}
		delete(w.pending, path)
		w.mu.Unlock()

		w.handleEvent(path)
	})
	w.pending[path] = pending
	w.mu.Unlock()
}

func (w *Watcher) cancelPending() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for path, pending := range w.pending {
		pending.timer.Stop()
		delete(w.pending, path)
	}
}

// handleEvent processes a single file event.
func (w *Watcher) handleEvent(path string) {
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
