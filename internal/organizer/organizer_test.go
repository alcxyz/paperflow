package organizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alcxyz/paperflow/internal/config"
)

func testConfig(watchDir string) *config.Config {
	return &config.Config{
		WatchDir:  watchDir,
		Ingest:    "none",
		IngestDir: filepath.Join(watchDir, "ingest"),
		Buckets: map[string][]string{
			"pdf":    {"pdf"},
			"images": {"jpg", "jpeg", "png"},
		},
		IngestTypes: config.IngestTypesConfig{
			Types: []string{"pdf", "jpg", "jpeg", "png"},
		},
	}
}

func TestProcessFile_BucketSorting(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	org := NewOrganizer(cfg, nil)

	// Create a test PDF file.
	src := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src, []byte("fake pdf"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := org.ProcessFile(src)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	if result.Bucket != "pdf" {
		t.Errorf("bucket = %q, want %q", result.Bucket, "pdf")
	}
	if result.Filename != "invoice.pdf" {
		t.Errorf("filename = %q, want %q", result.Filename, "invoice.pdf")
	}

	// File should no longer exist at source.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file should have been moved")
	}

	// File should exist in bucket dir.
	dest := filepath.Join(tmp, "pdf", result.Year, result.Month, "invoice.pdf")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("destination file not found: %v", err)
	}
}

func TestProcessFile_MiscBucket(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	org := NewOrganizer(cfg, nil)

	src := filepath.Join(tmp, "readme.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := org.ProcessFile(src)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	if result.Bucket != "misc" {
		t.Errorf("bucket = %q, want %q", result.Bucket, "misc")
	}
}

func TestProcessFile_ImageBucket(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	org := NewOrganizer(cfg, nil)

	src := filepath.Join(tmp, "photo.jpg")
	if err := os.WriteFile(src, []byte("fake jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := org.ProcessFile(src)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	if result.Bucket != "images" {
		t.Errorf("bucket = %q, want %q", result.Bucket, "images")
	}
}

func TestProcessFile_CollisionHandling(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	org := NewOrganizer(cfg, nil)

	// Create the first file and process it.
	src1 := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src1, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}

	result1, err := org.ProcessFile(src1)
	if err != nil {
		t.Fatalf("ProcessFile (first): %v", err)
	}

	destDir := filepath.Join(tmp, "pdf", result1.Year, result1.Month)

	// Verify the first file is there.
	dest1 := filepath.Join(destDir, "invoice.pdf")
	if _, err := os.Stat(dest1); err != nil {
		t.Fatalf("first file not at destination: %v", err)
	}

	// Create a second file with the same name and process it.
	src2 := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src2, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = org.ProcessFile(src2)
	if err != nil {
		t.Fatalf("ProcessFile (second): %v", err)
	}

	// There should be two files in the destination directory.
	entries, err := os.ReadDir(destDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %d", len(entries))
	}

	// One should be the original, the other should have a timestamp suffix.
	foundOriginal := false
	foundSuffixed := false
	for _, e := range entries {
		if e.Name() == "invoice.pdf" {
			foundOriginal = true
		} else if strings.HasPrefix(e.Name(), "invoice_") && strings.HasSuffix(e.Name(), ".pdf") {
			foundSuffixed = true
		}
	}

	if !foundOriginal {
		t.Error("original file not found")
	}
	if !foundSuffixed {
		t.Error("suffixed collision file not found")
	}
}

func TestProcessFile_DryRun(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	cfg.DryRun = true
	org := NewOrganizer(cfg, nil)

	src := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src, []byte("fake pdf"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := org.ProcessFile(src)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	if result.Bucket != "pdf" {
		t.Errorf("bucket = %q, want %q", result.Bucket, "pdf")
	}

	// Source file should NOT have been moved.
	if _, err := os.Stat(src); err != nil {
		t.Error("source file should still exist in dry-run mode")
	}

	// Destination should NOT exist.
	dest := filepath.Join(tmp, "pdf", result.Year, result.Month, "invoice.pdf")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("destination should not exist in dry-run mode")
	}
}

func TestProcessFile_DirectoryIngestion(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	cfg.Ingest = "directory"
	cfg.IngestDir = filepath.Join(tmp, "ingest")
	org := NewOrganizer(cfg, nil)

	src := filepath.Join(tmp, "invoice.pdf")
	if err := os.WriteFile(src, []byte("fake pdf"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := org.ProcessFile(src)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	if !result.Ingested {
		t.Error("expected Ingested = true for pdf with directory ingest")
	}

	// File should exist in ingest dir.
	ingestFile := filepath.Join(cfg.IngestDir, "invoice.pdf")
	if _, err := os.Stat(ingestFile); err != nil {
		t.Errorf("ingested file not found: %v", err)
	}
}

func TestProcessFile_MiscNotIngested(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(tmp)
	cfg.Ingest = "directory"
	cfg.IngestDir = filepath.Join(tmp, "ingest")
	org := NewOrganizer(cfg, nil)

	src := filepath.Join(tmp, "readme.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := org.ProcessFile(src)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	if result.Ingested {
		t.Error("misc files should not be ingested")
	}
}
