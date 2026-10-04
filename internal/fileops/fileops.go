// Package fileops provides the file moves and copies shared by the organizer
// and ingesters. Neither ever replaces an existing file: when the destination
// name is taken, a variant with a timestamp suffix is used instead.
package fileops

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

// maxAttempts bounds the search for a free destination name.
const maxAttempts = 1000

var tempSeq atomic.Uint64

// Move moves src to dst, or to a free variant of dst if that name is taken,
// and returns the final path. Moves across filesystems copy and then remove
// src.
func Move(src, dst string) (string, error) {
	path, err := linkUnique(src, dst)
	switch {
	case err == nil:
		return path, os.Remove(src)
	case errors.Is(err, syscall.EXDEV):
		path, err := Copy(src, dst)
		if err != nil {
			return "", err
		}
		return path, os.Remove(src)
	case linkUnsupported(err):
		return renameUnique(src, dst)
	default:
		return "", err
	}
}

// Copy copies src to dst, or to a free variant of dst if that name is taken,
// preserving the modification time, and returns the final path. The data is
// written to a hidden temporary file in dst's directory first, so anything
// watching that directory never sees a partial file.
func Copy(src, dst string) (string, error) {
	tmpPath, err := copyToTemp(src, filepath.Dir(dst))
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmpPath) }() // dst keeps its own link

	path, err := linkUnique(tmpPath, dst)
	if linkUnsupported(err) {
		return renameUnique(tmpPath, dst)
	}
	return path, err
}

// copyToTemp copies src into a new hidden temporary file in dir and returns
// its path.
func copyToTemp(src, dir string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return "", err
	}

	tmp, err := createTemp(dir)
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()

	_, err = io.Copy(tmp, in)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chtimes(tmpPath, time.Now(), info.ModTime())
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

// createTemp creates a new hidden file in dir with the same permissions
// os.Create would use. The ".tmp" extension keeps Paperless-ngx from
// consuming it.
func createTemp(dir string) (*os.File, error) {
	for range maxAttempts {
		name := fmt.Sprintf(".paperflow-%d-%d.tmp", os.Getpid(), tempSeq.Add(1))
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("creating temporary file in %s: too many collisions", dir)
}

// linkUnique hard-links src to the first free name derived from dst. Unlike a
// rename, a link fails rather than replacing a file created concurrently.
func linkUnique(src, dst string) (string, error) {
	ts := time.Now().Unix()
	for n := range maxAttempts {
		path := candidate(dst, ts, n)
		err := os.Link(src, path)
		if err == nil {
			return path, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("no free name for %s after %d attempts", dst, maxAttempts)
}

// renameUnique renames src to the first name derived from dst that does not
// exist. It is the fallback for filesystems without hard links, where a file
// created at the chosen name between the check and the rename is replaced.
func renameUnique(src, dst string) (string, error) {
	ts := time.Now().Unix()
	for n := range maxAttempts {
		path := candidate(dst, ts, n)
		_, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return path, os.Rename(src, path)
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("no free name for %s after %d attempts", dst, maxAttempts)
}

// candidate returns the nth name to try for path: path itself, then
// name_<ts>.ext, then name_<ts>_<n>.ext.
func candidate(path string, ts int64, n int) string {
	if n == 0 {
		return path
	}
	ext := filepath.Ext(path)
	name := path[:len(path)-len(ext)] + "_" + strconv.FormatInt(ts, 10)
	if n > 1 {
		name += "_" + strconv.Itoa(n)
	}
	return name + ext
}

// linkUnsupported reports whether err means the filesystem cannot create hard
// links.
func linkUnsupported(err error) bool {
	return errors.Is(err, syscall.EPERM) ||
		errors.Is(err, syscall.ENOTSUP) ||
		errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.ENOSYS) ||
		errors.Is(err, syscall.EMLINK)
}
