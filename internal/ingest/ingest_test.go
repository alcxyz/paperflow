package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alcxyz/paperflow/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()

	cfg.Ingest = config.IngestNone
	if ing, err := New(cfg, nil); err != nil || ing != nil {
		t.Errorf("New(none) = %v, %v; want nil, nil", ing, err)
	}

	cfg.Ingest = config.IngestDirectory
	if ing, err := New(cfg, nil); err != nil {
		t.Errorf("New(directory): %v", err)
	} else if _, ok := ing.(*DirectoryIngester); !ok {
		t.Errorf("New(directory) = %T, want *DirectoryIngester", ing)
	}

	cfg.Ingest = config.IngestAPI
	if ing, err := New(cfg, nil); err != nil {
		t.Errorf("New(api): %v", err)
	} else if _, ok := ing.(*APIIngester); !ok {
		t.Errorf("New(api) = %T, want *APIIngester", ing)
	}

	cfg.Ingest = "ftp"
	if _, err := New(cfg, nil); err == nil {
		t.Error("New should reject an unknown ingest method")
	}
}

func TestDirectoryIngester_SchedulesArchive(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	archiveDir := filepath.Join(tmp, "archive")
	archiver, err := NewArchiver(archiveDir, "1h")
	if err != nil {
		t.Fatal(err)
	}

	ing := &DirectoryIngester{Dir: filepath.Join(tmp, "ingest"), Archiver: archiver}
	if err := ing.Ingest(src); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	archiver.Close() // flushes the pending archive
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("archive has %d entries, want 1", len(entries))
	}
}

func TestDirectoryIngester_NilArchiver(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	ing := &DirectoryIngester{Dir: filepath.Join(tmp, "ingest")}
	if err := ing.Ingest(src); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
}
