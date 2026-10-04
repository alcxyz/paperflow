package ingest

import (
	"fmt"

	"github.com/alcxyz/paperflow/internal/config"
)

// Ingester hands a sorted file to Paperless-ngx.
type Ingester interface {
	Ingest(path string) error
}

// New returns the Ingester selected by cfg.Ingest, or nil when ingestion is
// disabled. Files ingested through a directory are scheduled on archiver,
// which may be nil.
func New(cfg *config.Config, archiver *Archiver) (Ingester, error) {
	switch cfg.Ingest {
	case config.IngestNone:
		return nil, nil
	case config.IngestDirectory:
		return &DirectoryIngester{Dir: cfg.IngestDir, Archiver: archiver}, nil
	case config.IngestAPI:
		return &APIIngester{URL: cfg.PaperlessURL, Token: cfg.Token}, nil
	default:
		return nil, fmt.Errorf("unknown ingest method %q", cfg.Ingest)
	}
}

// DirectoryIngester copies files into a Paperless-ngx consume directory.
type DirectoryIngester struct {
	Dir      string
	Archiver *Archiver
}

// Ingest copies path into the consume directory and schedules archival.
func (d *DirectoryIngester) Ingest(path string) error {
	destPath, err := IngestDirectory(path, d.Dir)
	if err != nil {
		return err
	}
	d.Archiver.Schedule(destPath)
	return nil
}

// APIIngester uploads files through the Paperless-ngx REST API.
type APIIngester struct {
	URL   string
	Token string
}

// Ingest uploads path to Paperless-ngx.
func (a *APIIngester) Ingest(path string) error {
	return IngestAPI(path, a.URL, a.Token)
}
