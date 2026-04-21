package notify

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/organizer"
)

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

// mockSender records all calls to sendNotification for test assertions.
type mockSender struct {
	mu    sync.Mutex
	calls []mockCall
}

type mockCall struct {
	appName, title, body string
}

func (m *mockSender) send(appName, title, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCall{appName, title, body})
	return nil
}

func (m *mockSender) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockSender) lastCall() mockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func newTestNotifier(batchWindow string) (*Notifier, *mockSender) {
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Enabled:     true,
			BatchWindow: batchWindow,
			AppName:     "Test",
		},
	}
	n := NewNotifier(cfg)
	m := &mockSender{}
	n.send = m.send
	return n, m
}

func TestNotifierBatchesSingleResult(t *testing.T) {
	n, _ := newTestNotifier("50ms")

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
	n, _ := newTestNotifier("100ms")

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
	n, m := newTestNotifier("10s") // Long window so it won't auto-flush.

	n.Notify(&organizer.Result{Filename: "test.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	// Close should flush immediately.
	n.Close()

	n.mu.Lock()
	remaining := len(n.sorted)
	n.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 pending results after Close, got %d", remaining)
	}
	if m.callCount() != 1 {
		t.Errorf("expected 1 send call after Close, got %d", m.callCount())
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
	m := &mockSender{}
	n.send = m.send

	n.Notify(&organizer.Result{Filename: "test.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	time.Sleep(100 * time.Millisecond)

	if m.callCount() != 0 {
		t.Error("send should not be called when notifications are disabled")
	}
}

func TestFlushCallsSendForSortedFiles(t *testing.T) {
	n, m := newTestNotifier("50ms")

	n.Notify(&organizer.Result{Filename: "invoice.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	time.Sleep(150 * time.Millisecond)

	if m.callCount() != 1 {
		t.Fatalf("expected 1 send call, got %d", m.callCount())
	}
	call := m.lastCall()
	if call.title != "Sorted: invoice.pdf -> pdf/2026/04/" {
		t.Errorf("unexpected title: %q", call.title)
	}
	if call.appName != "Test" {
		t.Errorf("unexpected appName: %q", call.appName)
	}
}

func TestFlushCallsSendForIngestedFiles(t *testing.T) {
	n, m := newTestNotifier("50ms")

	n.Notify(&organizer.Result{Filename: "invoice.pdf", Bucket: "pdf", Year: "2026", Month: "04", Ingested: true})

	time.Sleep(150 * time.Millisecond)

	// Should get two calls: one for sorted, one for ingested.
	if m.callCount() != 2 {
		t.Fatalf("expected 2 send calls (sorted + ingested), got %d", m.callCount())
	}
}

func TestFlushBatchesMultipleFilesIntoOneSend(t *testing.T) {
	n, m := newTestNotifier("100ms")

	n.Notify(&organizer.Result{Filename: "a.pdf", Bucket: "pdf", Year: "2026", Month: "04"})
	time.Sleep(20 * time.Millisecond)
	n.Notify(&organizer.Result{Filename: "b.pdf", Bucket: "pdf", Year: "2026", Month: "04"})

	time.Sleep(200 * time.Millisecond)

	// Two files but only one sorted notification (batched).
	if m.callCount() != 1 {
		t.Fatalf("expected 1 batched send call, got %d", m.callCount())
	}
	call := m.lastCall()
	if call.title != "Sorted 2 files" {
		t.Errorf("unexpected title: %q", call.title)
	}
}

func TestSendCallsSendFunc(t *testing.T) {
	n, m := newTestNotifier("50ms")

	n.Send("Starting", "Watching ~/Documents")

	if m.callCount() != 1 {
		t.Fatalf("expected 1 send call, got %d", m.callCount())
	}
	call := m.lastCall()
	if call.title != "Starting" {
		t.Errorf("unexpected title: %q", call.title)
	}
	if call.body != "Watching ~/Documents" {
		t.Errorf("unexpected body: %q", call.body)
	}
}

func TestSendErrorIsLogged(t *testing.T) {
	n, _ := newTestNotifier("50ms")
	n.send = func(_, _, _ string) error {
		return fmt.Errorf("notify-send not found")
	}

	// Should not panic; error is logged.
	n.Send("Test", "body")
}

func TestFlushSendErrorDoesNotLoseResults(t *testing.T) {
	n, _ := newTestNotifier("50ms")

	var mu sync.Mutex
	calls := 0
	n.send = func(_, _, _ string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return fmt.Errorf("command not found")
	}

	n.Notify(&organizer.Result{Filename: "a.pdf", Bucket: "pdf", Year: "2026", Month: "04", Ingested: true})

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	got := calls
	mu.Unlock()

	// Both sorted and ingested sends should be attempted even if the first fails.
	if got != 2 {
		t.Errorf("expected 2 send attempts (sorted + ingested), got %d", got)
	}
}
