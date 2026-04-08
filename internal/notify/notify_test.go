package notify

import (
	"sync"
	"testing"
	"time"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/organizer"
)

// testSendNotification captures notifications for testing.
var (
	testNotifications []testNotification
	testMu            sync.Mutex
)

type testNotification struct {
	title string
	body  string
}

func resetTestNotifications() {
	testMu.Lock()
	testNotifications = nil
	testMu.Unlock()
}

func getTestNotifications() []testNotification {
	testMu.Lock()
	defer testMu.Unlock()
	cp := make([]testNotification, len(testNotifications))
	copy(cp, testNotifications)
	return cp
}

func TestFormatSortedNotificationSingle(t *testing.T) {
	results := []*organizer.Result{
		{Filename: "invoice.pdf", Bucket: "pdf", Year: "2026", Month: "04"},
	}
	title, body := FormatSortedNotification(results)
	expected := "Sorted: invoice.pdf -> pdf/2026/04/"
	if title != expected {
		t.Errorf("expected title %q, got %q", expected, title)
	}
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
}

func TestFormatSortedNotificationMultiple(t *testing.T) {
	results := []*organizer.Result{
		{Filename: "invoice.pdf", Bucket: "pdf", Year: "2026", Month: "04"},
		{Filename: "receipt.jpg", Bucket: "images", Year: "2026", Month: "04"},
	}
	title, body := FormatSortedNotification(results)
	if title != "Sorted 2 files" {
		t.Errorf("expected title %q, got %q", "Sorted 2 files", title)
	}
	if body == "" {
		t.Error("expected non-empty body for multiple results")
	}
}

func TestFormatIngestNotificationSingle(t *testing.T) {
	results := []*organizer.Result{
		{Filename: "invoice.pdf", Bucket: "pdf", Year: "2026", Month: "04", Ingested: true},
	}
	title, body := FormatIngestNotification(results)
	if title != "Ingesting: invoice.pdf" {
		t.Errorf("expected title %q, got %q", "Ingesting: invoice.pdf", title)
	}
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
}

func TestFormatIngestNotificationMultiple(t *testing.T) {
	results := []*organizer.Result{
		{Filename: "invoice.pdf", Bucket: "pdf", Year: "2026", Month: "04", Ingested: true},
		{Filename: "receipt.jpg", Bucket: "images", Year: "2026", Month: "04", Ingested: true},
	}
	title, body := FormatIngestNotification(results)
	if title != "Ingesting 2 files" {
		t.Errorf("expected title %q, got %q", "Ingesting 2 files", title)
	}
	if body == "" {
		t.Error("expected non-empty body for multiple results")
	}
}

func TestNotifierBatchesSingleResult(t *testing.T) {
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Enabled:     true,
			BatchWindow: "50ms",
			AppName:     "Test",
		},
	}
	n := NewNotifier(cfg)

	n.Notify(&organizer.Result{Filename: "test.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	// Wait for the batch to flush.
	time.Sleep(150 * time.Millisecond)

	n.mu.Lock()
	remaining := len(n.sorted)
	n.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 pending results after flush, got %d", remaining)
	}
}

func TestNotifierBatchesMultipleResults(t *testing.T) {
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Enabled:     true,
			BatchWindow: "100ms",
			AppName:     "Test",
		},
	}
	n := NewNotifier(cfg)

	// Add multiple results within the batch window.
	n.Notify(&organizer.Result{Filename: "a.pdf", Bucket: "pdf", Year: "2026", Month: "04"})
	time.Sleep(20 * time.Millisecond)
	n.Notify(&organizer.Result{Filename: "b.jpg", Bucket: "images", Year: "2026", Month: "04"})
	time.Sleep(20 * time.Millisecond)
	n.Notify(&organizer.Result{Filename: "c.docx", Bucket: "docx", Year: "2026", Month: "04"})

	// Before the batch window expires, all should still be pending.
	n.mu.Lock()
	pending := len(n.sorted)
	n.mu.Unlock()

	if pending != 3 {
		t.Errorf("expected 3 pending results before flush, got %d", pending)
	}

	// Wait for the batch to flush.
	time.Sleep(200 * time.Millisecond)

	n.mu.Lock()
	remaining := len(n.sorted)
	n.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 pending results after flush, got %d", remaining)
	}
}

func TestNotifierCloseFlushes(t *testing.T) {
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Enabled:     true,
			BatchWindow: "10s", // Long window so it won't auto-flush.
			AppName:     "Test",
		},
	}
	n := NewNotifier(cfg)

	n.Notify(&organizer.Result{Filename: "test.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	// Close should flush immediately.
	n.Close()

	n.mu.Lock()
	remaining := len(n.sorted)
	n.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 pending results after Close, got %d", remaining)
	}
}

func TestNotifierDisabled(t *testing.T) {
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Enabled:     false,
			BatchWindow: "50ms",
			AppName:     "Test",
		},
	}
	n := NewNotifier(cfg)

	n.Notify(&organizer.Result{Filename: "test.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	// When disabled, nothing should be pending.
	n.mu.Lock()
	remaining := len(n.sorted)
	n.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 pending results when disabled, got %d", remaining)
	}
}
