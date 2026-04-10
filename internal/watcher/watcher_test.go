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

func TestDedup_SuppressesDuplicateEvents(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte("pdf data"))

	// First event processes the file.
	w.handleEvent(src)

	// Re-create the file to simulate a second event.
	writeTestFile(t, src, []byte("pdf data again"))

	// Second event within dedup window should be suppressed.
	w.handleEvent(src)

	// File should still exist (second event was suppressed).
	if _, err := os.Stat(src); err != nil {
		t.Error("second event should have been suppressed by dedup")
	}
}

func TestDedup_AllowsAfterWindow(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	src := filepath.Join(tmp, "invoice.pdf")
	writeTestFile(t, src, []byte("pdf data"))

	// Record the path as seen in the past.
	w.mu.Lock()
	w.seen[src] = time.Now().Add(-dedupWindow * 2)
	w.mu.Unlock()

	w.handleEvent(src)

	// File should have been processed (dedup window expired).
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("file should have been processed after dedup window expired")
	}
}

func TestDedup_PrunesOldEntries(t *testing.T) {
	tmp := t.TempDir()
	w := testWatcher(t, tmp)

	// Seed some old entries.
	w.mu.Lock()
	w.seen["/old/file1.pdf"] = time.Now().Add(-dedupWindow * 3)
	w.seen["/old/file2.pdf"] = time.Now().Add(-dedupWindow * 3)
	w.mu.Unlock()

	// Trigger a new event which will prune old entries.
	src := filepath.Join(tmp, "new.pdf")
	writeTestFile(t, src, []byte("data"))
	w.handleEvent(src)

	w.mu.Lock()
	_, hasOld1 := w.seen["/old/file1.pdf"]
	_, hasOld2 := w.seen["/old/file2.pdf"]
	w.mu.Unlock()

	if hasOld1 || hasOld2 {
		t.Error("old entries should have been pruned")
	}
}
