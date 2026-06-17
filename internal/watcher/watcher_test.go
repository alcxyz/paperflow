package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alcxyz/paperflow/internal/config"
)

func testWatcher(t *testing.T, watchDir string) *Watcher {
	t.Helper()
	cfg := &config.Config{
		WatchDir: watchDir,
		Ingest:   "none",
		Buckets: map[string][]string{
			"pdf":    {"pdf"},
			"images": {"jpg", "jpeg", "png"},
		},
		IngestTypes: config.IngestTypesConfig{
			Types: []string{"pdf"},
		},
		Notifications: config.NotificationsConfig{
			Enabled:     false,
			BatchWindow: "50ms",
			AppName:     "Test",
		},
	}
	w, err := NewWatcher(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func mkdirTest(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func TestHandleEvent_SkipsSubdirectory(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	subDir := filepath.Join(tmp, "sub")
	mkdirTest(t, subDir)
	subFile := filepath.Join(subDir, "test.pdf")
	writeTestFile(t, subFile, []byte("data"))

	w.handleEvent(subFile)

	if _, err := os.Stat(subFile); err != nil {
		t.Error("file in subdirectory should not have been moved")
	}
}

func TestHandleEvent_SkipsDotfiles(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	dotFile := filepath.Join(tmp, ".hidden.pdf")
	writeTestFile(t, dotFile, []byte("data"))

	w.handleEvent(dotFile)

	if _, err := os.Stat(dotFile); err != nil {
		t.Error("dotfile should not have been moved")
	}
}

func TestHandleEvent_SkipsExcludedPatterns(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)
	w.config.Exclude.Patterns = []string{"*.tmp", "~$*"}

	tmpFile := filepath.Join(tmp, "document.tmp")
	writeTestFile(t, tmpFile, []byte("data"))

	w.handleEvent(tmpFile)

	if _, err := os.Stat(tmpFile); err != nil {
		t.Error("excluded file should not have been moved")
	}

	lockFile := filepath.Join(tmp, "~$document.docx")
	writeTestFile(t, lockFile, []byte("data"))

	w.handleEvent(lockFile)

	if _, err := os.Stat(lockFile); err != nil {
		t.Error("excluded lock file should not have been moved")
	}
}

func TestHandleEvent_SkipsDirectories(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	dir := filepath.Join(tmp, "pdf")
	mkdirTest(t, dir)

	w.handleEvent(dir)

	if _, err := os.Stat(dir); err != nil {
		t.Error("directory should not have been removed")
	}
}

func TestHandleEvent_SkipsEmptyFiles(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	emptyFile := filepath.Join(tmp, "empty.pdf")
	writeTestFile(t, emptyFile, []byte{})

	w.handleEvent(emptyFile)

	if _, err := os.Stat(emptyFile); err != nil {
		t.Error("empty file should not have been moved")
	}
}

func TestHandleEvent_SkipsNonExistentFiles(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	// Should not panic or error.
	w.handleEvent(filepath.Join(tmp, "nonexistent.pdf"))
}

func TestHandleEvent_ProcessesValidFile(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte("pdf data"))

	w.handleEvent(src)

	// Source should be moved.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file should have been moved by organizer")
	}
}

func TestNewWatcher_RejectsInvalidSettleDelay(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		WatchDir:      tmp,
		SettleDelay:   "not-a-duration",
		Ingest:        "none",
		Buckets:       map[string][]string{"pdf": {"pdf"}},
		IngestTypes:   config.IngestTypesConfig{Types: []string{"pdf"}},
		Notifications: config.NotificationsConfig{Enabled: false},
	}

	if _, err := NewWatcher(cfg); err == nil {
		t.Fatal("NewWatcher should reject an invalid settle_delay")
	}
}

func TestScheduleEvent_DelaysProcessing(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)
	w.settleDelay = 30 * time.Millisecond
	defer w.cancelPending()

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte("pdf data"))

	w.scheduleEvent(src)

	if _, err := os.Stat(src); err != nil {
		t.Error("file should not be processed before settle delay")
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		_, err := os.Stat(src)
		return os.IsNotExist(err)
	})
}

func TestScheduleEvent_ZeroDelayProcessesImmediately(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)
	w.settleDelay = 0

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte("pdf data"))

	w.scheduleEvent(src)

	waitFor(t, 500*time.Millisecond, func() bool {
		_, err := os.Stat(src)
		return os.IsNotExist(err)
	})
}

func TestScheduleEvent_ResetsDelayForSamePath(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)
	w.settleDelay = 50 * time.Millisecond
	defer w.cancelPending()

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte{})

	w.scheduleEvent(src)
	time.Sleep(30 * time.Millisecond)
	writeTestFile(t, src, []byte("pdf data"))
	w.scheduleEvent(src)

	time.Sleep(30 * time.Millisecond)
	if _, err := os.Stat(src); err != nil {
		t.Error("file should not be processed until the reset delay elapses")
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		_, err := os.Stat(src)
		return os.IsNotExist(err)
	})
}

func TestScheduleEvent_SkipsStaleTimer(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)
	w.settleDelay = 40 * time.Millisecond
	defer w.cancelPending()

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte("data"))

	w.scheduleEvent(src)
	w.scheduleEvent(src)

	waitFor(t, 500*time.Millisecond, func() bool {
		_, err := os.Stat(src)
		return os.IsNotExist(err)
	})
}
