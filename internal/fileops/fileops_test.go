package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	outDir := filepath.Join(dir, "out")
	writeFile(t, src, "content")
	modTime := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := os.Chtimes(src, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(outDir, "dst.pdf")
	got, err := Copy(src, dst)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if got != dst {
		t.Errorf("Copy returned %q, want %q", got, dst)
	}
	if content := readFile(t, dst); content != "content" {
		t.Errorf("content = %q, want content", content)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(modTime) {
		t.Errorf("ModTime = %v, want %v", info.ModTime(), modTime)
	}
	if names := dirNames(t, outDir); len(names) != 1 {
		t.Errorf("destination dir = %v, want only dst.pdf (no temp files)", names)
	}
}

func TestCopyMissingSourceLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()

	if _, err := Copy(filepath.Join(dir, "missing.pdf"), filepath.Join(dir, "dst.pdf")); err == nil {
		t.Fatal("Copy should fail for a missing source")
	}
	if names := dirNames(t, dir); len(names) != 0 {
		t.Errorf("dir = %v, want empty", names)
	}
}

func TestCopyNeverReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	dst := filepath.Join(dir, "out.pdf")
	writeFile(t, src, "new")
	writeFile(t, dst, "old")

	got, err := Copy(src, dst)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if got == dst {
		t.Fatal("Copy should pick a new name when dst exists")
	}
	if !strings.HasPrefix(filepath.Base(got), "out_") || !strings.HasSuffix(got, ".pdf") {
		t.Errorf("Copy returned %q, want out_<timestamp>.pdf", got)
	}
	if readFile(t, dst) != "old" || readFile(t, got) != "new" {
		t.Error("existing file was modified")
	}
}

func TestConcurrentCopiesKeepEveryFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "invoice.pdf")

	const copies = 20
	var wg sync.WaitGroup
	errs := make(chan error, copies)
	for i := range copies {
		src := filepath.Join(dir, fmt.Sprintf("src%d", i))
		writeFile(t, src, fmt.Sprint(i))
		wg.Go(func() {
			if _, err := Copy(src, dst); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("Copy: %v", err)
	}

	seen := make(map[string]bool)
	for _, name := range dirNames(t, dir) {
		if strings.HasPrefix(name, "invoice") {
			seen[readFile(t, filepath.Join(dir, name))] = true
		}
	}
	if len(seen) != copies {
		t.Errorf("found %d distinct copies, want %d", len(seen), copies)
	}
}

func TestMove(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	dst := filepath.Join(dir, "dst.pdf")
	writeFile(t, src, "content")

	got, err := Move(src, dst)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got != dst {
		t.Errorf("Move returned %q, want %q", got, dst)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source should be gone after Move")
	}
	if content := readFile(t, dst); content != "content" {
		t.Errorf("content = %q, want content", content)
	}
}

func TestMoveCollisionSequence(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "invoice.pdf")

	var got []string
	for i := range 3 {
		src := filepath.Join(dir, fmt.Sprintf("src%d", i))
		writeFile(t, src, fmt.Sprint(i))
		path, err := Move(src, dst)
		if err != nil {
			t.Fatalf("Move %d: %v", i, err)
		}
		got = append(got, path)
	}

	if got[0] != dst {
		t.Errorf("first Move = %q, want %q", got[0], dst)
	}
	for i, path := range got {
		if content := readFile(t, path); content != fmt.Sprint(i) {
			t.Errorf("%s = %q, want %d", path, content, i)
		}
	}
}

func TestMoveFailsOnLongName(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	writeFile(t, src, "content")

	done := make(chan error, 1)
	go func() {
		_, err := Move(src, filepath.Join(dir, strings.Repeat("x", 300)+".pdf"))
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("Move should fail for a name longer than the filesystem allows")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Move did not return")
	}
}

func TestCopyFailsOnUnsearchableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions are not enforced for root")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src.pdf")
	locked := filepath.Join(dir, "locked")
	writeFile(t, src, "content")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0755) })

	done := make(chan error, 1)
	go func() {
		_, err := Copy(src, filepath.Join(locked, "dst.pdf"))
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("Copy into an unsearchable directory should fail")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Copy did not return")
	}
}

func TestCandidate(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "/d/invoice.pdf"},
		{1, "/d/invoice_42.pdf"},
		{2, "/d/invoice_42_2.pdf"},
	}
	for _, tt := range tests {
		if got := candidate("/d/invoice.pdf", 42, tt.n); got != tt.want {
			t.Errorf("candidate(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
