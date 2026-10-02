// Package fileops provides the file moves and copies shared by the organizer
// and ingesters.
package fileops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

var tempSeq atomic.Uint64

// Move renames src to dst. If the rename fails (for example across
// filesystems), it falls back to Copy followed by removing src.
func Move(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := Copy(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

// Copy copies src to dst, preserving the modification time. The data is
// written to a hidden temporary file in dst's directory and renamed into
// place, so anything watching that directory never sees a partial file.
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	tmp, err := createTemp(filepath.Dir(dst))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op after a successful rename

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(tmpPath, time.Now(), info.ModTime()); err != nil {
		return err
	}
	return os.Rename(tmpPath, dst)
}

// createTemp creates a new hidden file in dir with the same permissions
// os.Create would use. The ".tmp" extension keeps Paperless-ngx from
// consuming it.
func createTemp(dir string) (*os.File, error) {
	for range 100 {
		name := fmt.Sprintf(".paperflow-%d-%d.tmp", os.Getpid(), tempSeq.Add(1))
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("creating temporary file in %s: too many collisions", dir)
}

// UniquePath returns path if nothing exists there. Otherwise it appends a
// Unix timestamp before the extension, followed by a counter if that name is
// also taken.
func UniquePath(path string) string {
	if !exists(path) {
		return path
	}

	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)] + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	candidate := base + ext
	for n := 2; exists(candidate); n++ {
		candidate = base + "_" + strconv.Itoa(n) + ext
	}
	return candidate
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, os.ErrNotExist)
}
