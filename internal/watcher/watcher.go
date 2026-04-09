package watcher

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/ingest"
	"github.com/alcxyz/paperflow/internal/notify"
	"github.com/alcxyz/paperflow/internal/organizer"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a directory for new files and processes them.
type Watcher struct {
	config    *config.Config
	organizer *organizer.Organizer
	notifier  *notify.Notifier
}

// NewWatcher creates a Watcher with the given config.
func NewWatcher(cfg *config.Config) (*Watcher, error) {
	return &Watcher{
		config:    cfg,
		organizer: organizer.NewOrganizer(cfg),
		notifier:  notify.NewNotifier(cfg),
	}, nil
}

// Run starts watching cfg.WatchDir for Create and Rename events.
// It blocks until SIGINT or SIGTERM is received.
func (w *Watcher) Run() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fsw.Close()

	if err := fsw.Add(w.config.WatchDir); err != nil {
		return err
	}

	// Verify Paperless API authentication on startup.
	if w.config.Ingest == "api" {
		log.Printf("checking Paperless API connection...")
		if err := ingest.CheckAPI(w.config.PaperlessURL, w.config.Token); err != nil {
			return fmt.Errorf("paperless API check failed: %w", err)
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
			w.notifier.Close()
			return nil
		}
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
