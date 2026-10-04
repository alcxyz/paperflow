package fileops

import (
	"bytes"
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
