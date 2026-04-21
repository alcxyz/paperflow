package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewArchiver_DisabledWhenEmpty(t *testing.T) {
	a, err := NewArchiver("", "5m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != nil {
		t.Error("expected nil archiver when dir is empty")
	}
}

func TestNewArchiver_InvalidDuration(t *testing.T) {
	_, err := NewArchiver("/tmp/archive", "bad")
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestArchiver_ScheduleAndWait(t *testing.T) {
	tmp := t.TempDir()
	ingestDir := filepath.Join(tmp, "ingest")
	archiveDir := filepath.Join(tmp, "archive")
	if err := os.MkdirAll(ingestDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in the ingest dir.
	ingestFile := filepath.Join(ingestDir, "invoice.pdf")
	if err := os.WriteFile(ingestFile, []byte("pdf content"), 0644); err != nil {
		t.Fatal(err)
	}

	a, err := NewArchiver(archiveDir, "50ms")
	if err != nil {
		t.Fatal(err)
	}

	a.Schedule(ingestFile)

	// Wait for the timer to fire.
	time.Sleep(150 * time.Millisecond)

	// File should be gone from ingest dir.
	if _, err := os.Stat(ingestFile); !os.IsNotExist(err) {
		t.Error("file should have been moved from ingest dir")
	}

	// File should be in archive dir with timestamp prefix.
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 archived file, got %d", len(entries))
	}
	name := entries[0].Name()
	if !strings.HasSuffix(name, "_invoice.pdf") {
		t.Errorf("expected timestamp_invoice.pdf, got %q", name)
	}

	// Verify content.
	data, err := os.ReadFile(filepath.Join(archiveDir, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "pdf content" {
		t.Errorf("content = %q, want %q", string(data), "pdf content")
	}
}

func TestArchiver_CloseFlushesImmediately(t *testing.T) {
	tmp := t.TempDir()
	ingestDir := filepath.Join(tmp, "ingest")
	archiveDir := filepath.Join(tmp, "archive")
	if err := os.MkdirAll(ingestDir, 0755); err != nil {
		t.Fatal(err)
	}

	ingestFile := filepath.Join(ingestDir, "invoice.pdf")
	if err := os.WriteFile(ingestFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	a, err := NewArchiver(archiveDir, "1h") // Long delay.
	if err != nil {
		t.Fatal(err)
	}

	a.Schedule(ingestFile)
	a.Close()

	// File should be archived immediately.
	if _, err := os.Stat(ingestFile); !os.IsNotExist(err) {
		t.Error("file should have been archived on Close")
	}

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 archived file, got %d", len(entries))
	}
}

func TestArchiver_MissingFileSkipped(t *testing.T) {
	tmp := t.TempDir()
	archiveDir := filepath.Join(tmp, "archive")

	a, err := NewArchiver(archiveDir, "50ms")
	if err != nil {
		t.Fatal(err)
	}

	// Schedule a file that doesn't exist.
	a.Schedule(filepath.Join(tmp, "nonexistent.pdf"))

	time.Sleep(150 * time.Millisecond)

	// Archive dir should not have been created (nothing to archive).
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(archiveDir)
		if len(entries) > 0 {
			t.Error("no files should be archived for missing source")
		}
	}
}

func TestArchiver_MultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	ingestDir := filepath.Join(tmp, "ingest")
	archiveDir := filepath.Join(tmp, "archive")
	if err := os.MkdirAll(ingestDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := []string{"a.pdf", "b.pdf", "c.pdf"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(ingestDir, f), []byte(f), 0644); err != nil {
			t.Fatal(err)
		}
	}

	a, err := NewArchiver(archiveDir, "1h")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		a.Schedule(filepath.Join(ingestDir, f))
	}

	a.Close()

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 archived files, got %d", len(entries))
	}
}
