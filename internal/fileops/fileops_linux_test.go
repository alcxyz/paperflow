package fileops

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

// TestCopyEmitsMovedTo guards Paperless-ngx's inotify consumer, which only
// reacts to IN_CLOSE_WRITE, IN_MOVED_TO and IN_MODIFY. The final file name
// must receive one of them (a hard link, for example, emits only IN_CREATE).
func TestCopyEmitsMovedTo(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(t.TempDir(), "invoice.pdf")
	writeFile(t, src, "content")

	fd, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		t.Skipf("inotify unavailable: %v", err)
	}
	defer func() { _ = syscall.Close(fd) }()
	mask := uint32(syscall.IN_CLOSE_WRITE | syscall.IN_MOVED_TO | syscall.IN_MODIFY)
	if _, err := syscall.InotifyAddWatch(fd, dir, mask); err != nil {
		t.Fatal(err)
	}

	if _, err := Copy(src, filepath.Join(dir, "invoice.pdf")); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	buf := make([]byte, 64*1024)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		t.Fatalf("reading inotify events: %v", err)
	}
	for offset := 0; offset+syscall.SizeofInotifyEvent <= n; {
		event := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
		nameStart := offset + syscall.SizeofInotifyEvent
		name := string(bytes.TrimRight(buf[nameStart:nameStart+int(event.Len)], "\x00"))
		if name == "invoice.pdf" && event.Mask&syscall.IN_MOVED_TO != 0 {
			return
		}
		offset = nameStart + int(event.Len)
	}
	t.Error("final file name received no IN_MOVED_TO event")
}

func TestMoveAcrossFilesystemsPreservesPermissions(t *testing.T) {
	dir, err := os.MkdirTemp("/dev/shm", "paperflow-test-*")
	if err != nil {
		t.Skipf("second filesystem unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	src := filepath.Join(t.TempDir(), "source.pdf")
	writeFile(t, src, "synthetic document")
	if err := os.Chmod(src, 0600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "copy.pdf")
	// Prove the move must take the copy fallback on this host.
	if err := os.Rename(src, dst); !errors.Is(err, syscall.EXDEV) {
		t.Skipf("cross-device rename unavailable: %v", err)
	}
	got, err := Move(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 || readFile(t, got) != "synthetic document" {
		t.Error("move did not preserve permissions and content")
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Errorf("source still present: %v", err)
	}
	if names := dirNames(t, dir); len(names) != 1 {
		t.Errorf("destination directory = %v, want only final file", names)
	}
}
