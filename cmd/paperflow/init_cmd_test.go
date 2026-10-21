package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/alcxyz/paperflow/internal/config"
)

func TestWriteTokenFile(t *testing.T) {
	for _, mode := range []os.FileMode{0, 0400, 0600, 0644, 0666} {
		t.Run(fmt.Sprintf("%04o", mode), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "token")
			if mode != 0 {
				if err := os.WriteFile(path, []byte("old synthetic value\n"), mode); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, mode); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeTokenFile(path, "new synthetic value"); err != nil {
				t.Fatal(err)
			}
			assertSyntheticToken(t, path)
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Errorf("temporary file left behind: %v, %v", entries, err)
			}
		})
	}
}

func assertSyntheticToken(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Errorf("token mode = %v, want regular 0600", info.Mode())
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "new synthetic value\n" {
		t.Errorf("unexpected synthetic token content or read error: %v", err)
	}
}

func TestWriteTokenFileRejectsUnsuitableDestinations(t *testing.T) {
	for _, kind := range []string{"symlink", "dangling symlink", "directory", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "token")
			target := filepath.Join(t.TempDir(), "target")
			switch kind {
			case "fifo":
				if err := syscall.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			default:
				if kind == "symlink" {
					if err := os.WriteFile(target, []byte("old synthetic value"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := writeTokenFile(path, "new synthetic value"); err == nil {
				t.Fatal("unsuitable destination accepted")
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(before, after) {
				t.Errorf("destination changed: %v", err)
			}
			switch kind {
			case "symlink":
				data, err := os.ReadFile(target)
				if err != nil || string(data) != "old synthetic value" {
					t.Error("symlink target changed")
				}
			case "dangling symlink":
				if _, err := os.Lstat(target); !os.IsNotExist(err) {
					t.Errorf("dangling target created: %v", err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Errorf("temporary file left behind: %v, %v", entries, err)
			}
		})
	}
}

func TestWriteTokenFileReplacesHardLink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, []byte("old synthetic value"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(other, path); err != nil {
		t.Fatal(err)
	}
	if err := writeTokenFile(path, "new synthetic value"); err != nil {
		t.Fatal(err)
	}
	assertSyntheticToken(t, path)
	data, err := os.ReadFile(other)
	if err != nil || string(data) != "old synthetic value" {
		t.Error("other hard link changed")
	}
}

func TestRunInitReplacesToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := config.DefaultTokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old synthetic value"), 0644); err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(dir, "input")
	if err := os.WriteFile(inputPath, []byte("\napi\nhttps://paperless.example.com\nnew synthetic value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdin
	os.Stdin = input
	t.Cleanup(func() {
		os.Stdin = original
		_ = input.Close()
	})
	if err := runInit(&options{configPath: config.DefaultConfigPath()}); err != nil {
		t.Fatal(err)
	}
	assertSyntheticToken(t, path)
}
