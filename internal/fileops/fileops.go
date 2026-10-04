// Package fileops provides the file moves and copies shared by the organizer
// and ingesters. When a destination name is taken, a variant with a timestamp
// suffix is used instead. Placements made by this process are serialized, so
// they never replace each other; only a file another process creates at the
// chosen name in the instant before the rename could be replaced.
package fileops

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// maxAttempts bounds the search for a free destination name.
const maxAttempts = 1000

var (
	// placeMu serializes choosing a free name and renaming into it.
	placeMu sync.Mutex
	tempSeq atomic.Uint64
)

// Move moves src to dst, or to a free variant of dst if that name is taken,
// and returns the final path. If src cannot be renamed (for example across
// filesystems), it is copied and then removed.
func Move(src, dst string) (string, error) {
	path, err := renameUnique(src, dst)
	if err == nil || errors.Is(err, errNoFreeName) {
		return path, err
	}

	path, err = Copy(src, dst)
	if err != nil {
		return "", err
	}
	return path, os.Remove(src)
}

// Copy copies src to dst, or to a free variant of dst if that name is taken,
// preserving the modification time, and returns the final path. The data is
// written to a hidden temporary file in dst's directory and renamed into
// place, so anything watching that directory never sees a partial file.
func Copy(src, dst string) (string, error) {
	tmpPath, err := copyToTemp(src, filepath.Dir(dst))
	if err != nil {
		return "", err
	}

	path, err := renameUnique(tmpPath, dst)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return path, nil
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
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	// Keeping the modification time is best effort: filesystems that map
	// ownership (for example NFS with all_squash) refuse to set it.
	_ = os.Chtimes(tmpPath, time.Now(), info.ModTime())
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

var errNoFreeName = errors.New("no free destination name")

// renameUnique renames src to the first name derived from dst that does not
// exist and returns that name. A rename keeps Paperless-ngx's inotify
// consumer informed (IN_MOVED_TO), unlike a hard link.
func renameUnique(src, dst string) (string, error) {
	placeMu.Lock()
	defer placeMu.Unlock()

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
	return "", fmt.Errorf("%w for %s after %d attempts", errNoFreeName, dst, maxAttempts)
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
