package organizer

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alcxyz/paperflow/internal/bucket"
	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/ingest"
)

// Result describes the outcome of processing a single file.
type Result struct {
	Filename string
	Bucket   string
	Year     string
	Month    string
	Ingested bool
}

// Organizer handles sorting files into bucket/year/month directories.
type Organizer struct {
	config   *config.Config
	archiver *ingest.Archiver
}

// NewOrganizer creates an Organizer with the given config.
// The archiver may be nil if archive is disabled.
func NewOrganizer(cfg *config.Config, archiver *ingest.Archiver) *Organizer {
	return &Organizer{config: cfg, archiver: archiver}
}

// ProcessFile sorts a file into the appropriate bucket/year/month directory
// and optionally ingests it into Paperless-ngx.
func (o *Organizer) ProcessFile(path string) (*Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	b := bucket.GetBucket(ext, o.config.Buckets)

	modTime := info.ModTime()
	year := strconv.Itoa(modTime.Year())
	month := fmt.Sprintf("%02d", int(modTime.Month()))

	destDir := filepath.Join(o.config.WatchDir, b, year, month)
	destPath := filepath.Join(destDir, filename)

	result := &Result{
		Filename: filename,
		Bucket:   b,
		Year:     year,
		Month:    month,
	}

	if o.config.DryRun {
		log.Printf("[dry-run] would move %s -> %s", filename, destPath)
		// Check ingestion eligibility even in dry-run.
		if o.config.Ingest != "none" && b != "misc" && bucket.IsIngestible(ext, o.config.IngestTypes.Types) {
			log.Printf("[dry-run] would ingest %s", filename)
			result.Ingested = true
		}
		return result, nil
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("creating directory %s: %w", destDir, err)
	}

	// Handle collision by appending a timestamp suffix.
	destPath = ingest.ResolveCollision(destPath)

	if err := moveFile(path, destPath); err != nil {
		return nil, fmt.Errorf("moving %s to %s: %w", path, destPath, err)
	}

	log.Printf("sorted %s -> %s", filename, destPath)

	// Ingest if applicable.
	if o.config.Ingest != "none" && b != "misc" && bucket.IsIngestible(ext, o.config.IngestTypes.Types) {
		if err := o.doIngest(destPath); err != nil {
			log.Printf("warning: ingest failed for %s: %v", filename, err)
		} else {
			result.Ingested = true
			log.Printf("ingested %s via %s", filename, o.config.Ingest)
		}
	}

	return result, nil
}

// moveFile moves src to dst. It first tries os.Rename, and falls back to
// copy+remove for cross-device moves.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fallback: copy then remove.
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	return os.Remove(src)
}

// doIngest copies or uploads the file to Paperless-ngx.
func (o *Organizer) doIngest(path string) error {
	switch o.config.Ingest {
	case "directory":
		destPath, err := ingest.IngestDirectory(path, o.config.IngestDir)
		if err != nil {
			return err
		}
		if o.archiver != nil {
			o.archiver.Schedule(destPath)
		}
		return nil
	case "api":
		return ingest.IngestAPI(path, o.config.PaperlessURL, o.config.Token)
	default:
		return nil
	}
}
