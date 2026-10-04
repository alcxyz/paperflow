package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/ingest"
)

// report counts and prints validation results.
type report struct {
	errors   int
	warnings int
}

func (r *report) ok(format string, args ...any) {
	fmt.Printf("  OK    "+format+"\n", args...)
}

func (r *report) warn(format string, args ...any) {
	r.warnings++
	fmt.Printf("  WARN  "+format+"\n", args...)
}

func (r *report) fail(format string, args ...any) {
	r.errors++
	fmt.Printf("  FAIL  "+format+"\n", args...)
}

func runValidate(opts *options) error {
	fmt.Printf("Validating config: %s\n\n", opts.configPath)

	cfg, err := config.LoadConfig(opts.configPath, opts.overrides)
	if err != nil {
		fmt.Printf("  FAIL  config: %v\n", err)
		return errors.New("validation failed")
	}
	fmt.Println("  OK    config loaded")

	r := &report{}
	settingsErr := cfg.Check()
	if settingsErr == nil {
		r.ok("settings")
	}
	for _, err := range splitErrors(settingsErr) {
		r.fail("%v", err)
	}

	r.checkDir("watch_dir", cfg.WatchDir)

	switch cfg.Ingest {
	case config.IngestNone:
		r.ok("ingest: none (sorting only)")
	case config.IngestDirectory:
		r.checkDir("ingest_dir", cfg.IngestDir)
		if cfg.IngestArchiveDir != "" {
			if info, err := os.Stat(cfg.IngestArchiveDir); err != nil {
				r.warn("ingest_archive_dir: %s does not exist (will be created)", cfg.IngestArchiveDir)
			} else if !info.IsDir() {
				r.fail("ingest_archive_dir: %s is not a directory", cfg.IngestArchiveDir)
			} else {
				r.ok("ingest_archive_dir: %s", cfg.IngestArchiveDir)
			}
		}
	case config.IngestAPI:
		if info, err := os.Stat(cfg.TokenFile); err == nil {
			if perm := info.Mode().Perm(); perm&0077 != 0 {
				r.warn("token: %s has permissions %04o, should be 0600", cfg.TokenFile, perm)
			} else {
				r.ok("token: %s (permissions %04o)", cfg.TokenFile, perm)
			}
		}
		// Only contact Paperless once the URL and token are known to be usable.
		if settingsErr == nil {
			if err := ingest.CheckAPI(cfg.PaperlessURL, cfg.Token); err != nil {
				r.fail("paperless API: %v", err)
			} else {
				r.ok("paperless API: authenticated at %s", cfg.PaperlessURL)
			}
		}
	}

	if len(cfg.Buckets) == 0 {
		r.warn("buckets: none defined")
	} else {
		r.ok("buckets: %d defined", len(cfg.Buckets))
	}

	fmt.Println()
	if r.errors > 0 {
		fmt.Printf("Validation failed: %d error(s), %d warning(s)\n", r.errors, r.warnings)
		return fmt.Errorf("validation failed with %d error(s)", r.errors)
	}
	if r.warnings > 0 {
		fmt.Printf("Validation passed with %d warning(s)\n", r.warnings)
	} else {
		fmt.Println("Validation passed")
	}
	return nil
}

func (r *report) checkDir(name, path string) {
	if path == "" {
		return // reported by Config.Check
	}
	if info, err := os.Stat(path); err != nil {
		r.fail("%s: %s does not exist", name, path)
	} else if !info.IsDir() {
		r.fail("%s: %s is not a directory", name, path)
	} else {
		r.ok("%s: %s", name, path)
	}
}

// splitErrors returns the errors combined by errors.Join, or err itself.
func splitErrors(err error) []error {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return joined.Unwrap()
	}
	return []error{err}
}
