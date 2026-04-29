package ingest

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// IngestDirectory copies the file to the configured ingest directory.
// It returns the destination path on success.
func IngestDirectory(path string, ingestDir string) (string, error) {
	filename := filepath.Base(path)
	destPath := filepath.Join(ingestDir, filename)

	// Handle collision in ingest dir too.
	destPath = ResolveCollision(destPath)

	if err := os.MkdirAll(ingestDir, 0755); err != nil {
		return "", fmt.Errorf("creating ingest directory: %w", err)
	}

	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}

	log.Printf("ingested %s -> %s", filename, destPath)
	return destPath, out.Close()
}

// ResolveCollision returns a unique destination path. If destPath already
// exists, it appends a Unix timestamp before the extension.
func ResolveCollision(destPath string) string {
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return destPath
	}

	ext := filepath.Ext(destPath)
	base := destPath[:len(destPath)-len(ext)]
	ts := time.Now().Unix()
	return fmt.Sprintf("%s_%d%s", base, ts, ext)
}
