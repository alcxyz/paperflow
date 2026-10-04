package ingest

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alcxyz/paperflow/internal/fileops"
)

// IngestDirectory copies the file to the configured ingest directory.
// It returns the destination path on success.
func IngestDirectory(path string, ingestDir string) (string, error) {
	if err := os.MkdirAll(ingestDir, 0755); err != nil {
		return "", fmt.Errorf("creating ingest directory: %w", err)
	}

	filename := filepath.Base(path)
	destPath, err := fileops.Copy(path, filepath.Join(ingestDir, filename))
	if err != nil {
		return "", err
	}

	log.Printf("ingested %s -> %s", filename, destPath)
	return destPath, nil
}
