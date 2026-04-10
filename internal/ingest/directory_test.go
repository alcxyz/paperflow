package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngestDirectory(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	ingestDir := filepath.Join(tmp, "ingest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(srcDir, "invoice.pdf")
	if err := os.WriteFile(srcFile, []byte("pdf content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := IngestDirectory(srcFile, ingestDir); err != nil {
		t.Fatalf("IngestDirectory: %v", err)
	}

	// File should exist in ingest dir.
	destFile := filepath.Join(ingestDir, "invoice.pdf")
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("ingested file not found: %v", err)
	}
	if string(data) != "pdf content" {
		t.Errorf("content = %q, want %q", string(data), "pdf content")
	}

	// Source should still exist (it's a copy, not move).
	if _, err := os.Stat(srcFile); err != nil {
		t.Error("source file should still exist after ingestion")
	}
}

func TestIngestDirectory_Collision(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	ingestDir := filepath.Join(tmp, "ingest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ingestDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Pre-create a file with the same name in ingest dir.
	existing := filepath.Join(ingestDir, "invoice.pdf")
	if err := os.WriteFile(existing, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(srcDir, "invoice.pdf")
	if err := os.WriteFile(srcFile, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := IngestDirectory(srcFile, ingestDir); err != nil {
		t.Fatalf("IngestDirectory: %v", err)
	}

	// Should have two files in ingest dir.
	entries, err := os.ReadDir(ingestDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %d", len(entries))
	}
}

func TestIngestDirectory_CreatesDir(t *testing.T) {
	tmp := t.TempDir()
	srcFile := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	ingestDir := filepath.Join(tmp, "new", "nested", "ingest")
	if err := IngestDirectory(srcFile, ingestDir); err != nil {
		t.Fatalf("IngestDirectory: %v", err)
	}

	destFile := filepath.Join(ingestDir, "invoice.pdf")
	if _, err := os.Stat(destFile); err != nil {
		t.Errorf("file not found in created directory: %v", err)
	}
}

func TestResolveCollision_NoConflict(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "invoice.pdf")
	got := ResolveCollision(path)
	if got != path {
		t.Errorf("expected original path, got %q", got)
	}
}

func TestResolveCollision_WithConflict(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(path, []byte("exists"), 0644); err != nil {
		t.Fatal(err)
	}

	got := ResolveCollision(path)
	if got == path {
		t.Error("expected different path when collision exists")
	}
	if !strings.HasPrefix(filepath.Base(got), "invoice_") {
		t.Errorf("expected timestamp prefix, got %q", filepath.Base(got))
	}
	if !strings.HasSuffix(got, ".pdf") {
		t.Errorf("expected .pdf extension, got %q", got)
	}
}
