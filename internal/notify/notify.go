package notify

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/organizer"
)

// Notifier batches file processing results and sends grouped desktop notifications.
type Notifier struct {
	enabled     bool
	appName     string
	batchWindow time.Duration

	mu       sync.Mutex
	sorted   []*organizer.Result
	ingested []*organizer.Result
	timer    *time.Timer
}

// NewNotifier creates a Notifier from config. If notifications are disabled,
// Notify calls are no-ops.
func NewNotifier(cfg *config.Config) *Notifier {
	dur, err := time.ParseDuration(cfg.Notifications.BatchWindow)
	if err != nil {
		dur = 3 * time.Second
	}
	return &Notifier{
		enabled:     cfg.Notifications.Enabled,
		appName:     cfg.Notifications.AppName,
		batchWindow: dur,
	}
}

// Notify adds a result to the pending batch and resets the batch timer.
// When the timer fires after a quiet period, all pending results are flushed
// as a single notification.
func (n *Notifier) Notify(result *organizer.Result) {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.sorted = append(n.sorted, result)
	if result.Ingested {
		n.ingested = append(n.ingested, result)
	}

	// Reset the batch timer.
	if n.timer != nil {
		n.timer.Stop()
	}
	n.timer = time.AfterFunc(n.batchWindow, n.flush)
}

// Send sends a notification immediately (non-batched).
func (n *Notifier) Send(title, body string) {
	if !n.enabled {
		return
	}
	if err := sendNotification(n.appName, title, body); err != nil {
		log.Printf("notification error: %v", err)
	}
}

// Close flushes any remaining pending results and stops the timer.
func (n *Notifier) Close() {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}
	n.mu.Unlock()

	n.flush()
}

// flush sends notifications for all pending results and clears the batches.
func (n *Notifier) flush() {
	n.mu.Lock()
	sorted := n.sorted
	ingested := n.ingested
	n.sorted = nil
	n.ingested = nil
	n.mu.Unlock()

	if len(sorted) > 0 {
		title, body := FormatSortedNotification(sorted)
		if err := sendNotification(n.appName, title, body); err != nil {
			log.Printf("notification error: %v", err)
		}
	}

	if len(ingested) > 0 {
		title, body := FormatIngestNotification(ingested)
		if err := sendNotification(n.appName, title, body); err != nil {
			log.Printf("notification error: %v", err)
		}
	}
}

// ResultPath returns a short display path like "pdf/2026/04/".
func ResultPath(r *organizer.Result) string {
	return fmt.Sprintf("%s/%s/%s/", r.Bucket, r.Year, r.Month)
}

// FormatSortedNotification returns a title and body for sorted file notifications.
func FormatSortedNotification(results []*organizer.Result) (string, string) {
	if len(results) == 1 {
		r := results[0]
		return fmt.Sprintf("Sorted: %s -> %s", r.Filename, ResultPath(r)), ""
	}

	title := fmt.Sprintf("Sorted %d files", len(results))
	var lines []string
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("%s -> %s", r.Filename, ResultPath(r)))
	}
	return title, strings.Join(lines, ", ")
}

// FormatIngestNotification returns a title and body for ingestion notifications.
func FormatIngestNotification(results []*organizer.Result) (string, string) {
	if len(results) == 1 {
		return fmt.Sprintf("Ingesting: %s", results[0].Filename), ""
	}

	title := fmt.Sprintf("Ingesting %d files", len(results))
	var lines []string
	for _, r := range results {
		lines = append(lines, r.Filename)
	}
	return title, strings.Join(lines, ", ")
}
