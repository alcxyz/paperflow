package fileops

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	dst := filepath.Join(dir, "out", "dst.pdf")
	writeFile(t, src, "content")
	modTime := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := os.Chtimes(src, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}

	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if got := readFile(t, dst); got != "content" {
		t.Errorf("content = %q, want content", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(modTime) {
		t.Errorf("ModTime = %v, want %v", info.ModTime(), modTime)
	}
	entries, err := os.ReadDir(filepath.Dir(dst))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("destination dir has %d entries, want only dst (no temp files)", len(entries))
	}
}

func TestCopyMissingSourceLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()

	if err := Copy(filepath.Join(dir, "missing.pdf"), filepath.Join(dir, "dst.pdf")); err == nil {
		t.Fatal("Copy should fail for a missing source")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("dir has %d entries, want none", len(entries))
	}
}

func TestMove(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	dst := filepath.Join(dir, "dst.pdf")
	writeFile(t, src, "content")

	if err := Move(src, dst); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source should be gone after Move")
	}
	if got := readFile(t, dst); got != "content" {
		t.Errorf("content = %q, want content", got)
	}
}

func TestUniquePath_NoConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoice.pdf")
	if got := UniquePath(path); got != path {
		t.Errorf("UniquePath = %q, want %q", got, path)
	}
}

func TestUniquePath_WithConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoice.pdf")
	writeFile(t, path, "exists")

	got := UniquePath(path)
	if got == path {
		t.Error("expected a different path when the file exists")
	}
	if !strings.HasPrefix(filepath.Base(got), "invoice_") || !strings.HasSuffix(got, ".pdf") {
		t.Errorf("UniquePath = %q, want invoice_<timestamp>.pdf", got)
	}
}

func TestUniquePath_NeverReturnsExistingPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.pdf")
	writeFile(t, path, "1")

	// Occupy the timestamped names for this second and the next, in case
	// the clock ticks during the test.
	for i := range 2 {
		ts := time.Now().Add(time.Duration(i) * time.Second).Unix()
		base := filepath.Join(dir, "invoice_"+strconv.FormatInt(ts, 10))
		writeFile(t, base+".pdf", "2")
		writeFile(t, base+"_2.pdf", "3")
	}

	got := UniquePath(path)
	if _, err := os.Stat(got); err == nil {
		t.Errorf("UniquePath returned existing path %q", got)
	}
}
