package organizer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alcxyz/paperflow/internal/bucket"
	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/fileops"
	"github.com/alcxyz/paperflow/internal/ingest"
)

// Result describes the outcome of processing a single file.
type Result struct {
	Filename string
	Bucket   string
	Year     string
	Month    string
	Ingested bool
	// IngestFailed reports that the file was sorted but could not be ingested.
	IngestFailed bool
}

// Organizer handles sorting files into bucket/year/month directories.
type Organizer struct {
	config   *config.Config
	ingester ingest.Ingester
}

// NewOrganizer creates an Organizer with the given config.
// The ingester is nil when ingestion is disabled.
func NewOrganizer(cfg *config.Config, ingester ingest.Ingester) *Organizer {
	return &Organizer{config: cfg, ingester: ingester}
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
		if o.shouldIngest(b, ext) {
			log.Printf("[dry-run] would ingest %s", filename)
			result.Ingested = true
		}
		return result, nil
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("creating directory %s: %w", destDir, err)
	}

	// On a name collision, Move appends a timestamp suffix.
	destPath, err = fileops.Move(path, destPath)
	if err != nil {
		return nil, fmt.Errorf("moving %s to %s: %w", path, destDir, err)
	}

	log.Printf("sorted %s -> %s", filename, destPath)

	// Ingest if applicable.
	if o.shouldIngest(b, ext) {
		if err := o.ingester.Ingest(destPath); err != nil {
			result.IngestFailed = true
			log.Printf("warning: ingest failed for %s: %v", filename, err)
		} else {
			result.Ingested = true
			log.Printf("ingested %s via %s", filename, o.config.Ingest)
		}
	}

	return result, nil
}

// shouldIngest reports whether a file in bucket b with extension ext is
// forwarded to Paperless-ngx. Files in misc are never ingested.
func (o *Organizer) shouldIngest(b, ext string) bool {
	return o.ingester != nil && b != "misc" && bucket.IsIngestible(ext, o.config.IngestTypes.Types)
}
