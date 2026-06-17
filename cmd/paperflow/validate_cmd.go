package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alcxyz/paperflow/internal/config"
)

func runValidate(f flags) error {
	configPath := f.config
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	fmt.Printf("Validating config: %s\n\n", configPath)

	errors := 0
	warnings := 0

	// Check config file exists and parses.
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("  FAIL  config: %v\n", err)
		return fmt.Errorf("validation failed")
	}
	fmt.Println("  OK    config file parsed successfully")

	// Check watch directory exists.
	watchDir := config.ExpandTilde(cfg.WatchDir)
	if info, err := os.Stat(watchDir); err != nil {
		fmt.Printf("  FAIL  watch_dir: %s does not exist\n", watchDir)
		errors++
	} else if !info.IsDir() {
		fmt.Printf("  FAIL  watch_dir: %s is not a directory\n", watchDir)
		errors++
	} else {
		fmt.Printf("  OK    watch_dir: %s\n", watchDir)
	}

	if delay, err := time.ParseDuration(cfg.SettleDelay); err != nil {
		fmt.Printf("  FAIL  settle_delay: invalid duration %q\n", cfg.SettleDelay)
		errors++
	} else if delay < 0 {
		fmt.Printf("  FAIL  settle_delay: must not be negative (%s)\n", cfg.SettleDelay)
		errors++
	} else {
		fmt.Printf("  OK    settle_delay: %s\n", cfg.SettleDelay)
	}

	// Check ingest method.
	switch cfg.Ingest {
	case "none":
		fmt.Println("  OK    ingest: none (sorting only)")
	case "directory":
		ingestDir := config.ExpandTilde(cfg.IngestDir)
		if info, err := os.Stat(ingestDir); err != nil {
			fmt.Printf("  FAIL  ingest_dir: %s does not exist\n", ingestDir)
			errors++
		} else if !info.IsDir() {
			fmt.Printf("  FAIL  ingest_dir: %s is not a directory\n", ingestDir)
			errors++
		} else {
			fmt.Printf("  OK    ingest_dir: %s\n", ingestDir)
		}

		if cfg.IngestArchiveDir != "" {
			archiveDir := config.ExpandTilde(cfg.IngestArchiveDir)
			if info, err := os.Stat(archiveDir); err != nil {
				fmt.Printf("  WARN  ingest_archive_dir: %s does not exist (will be created)\n", archiveDir)
				warnings++
			} else if !info.IsDir() {
				fmt.Printf("  FAIL  ingest_archive_dir: %s is not a directory\n", archiveDir)
				errors++
			} else {
				fmt.Printf("  OK    ingest_archive_dir: %s\n", archiveDir)
			}
			if _, err := time.ParseDuration(cfg.IngestArchiveAfter); err != nil {
				fmt.Printf("  FAIL  ingest_archive_after: invalid duration %q\n", cfg.IngestArchiveAfter)
				errors++
			} else {
				fmt.Printf("  OK    ingest_archive_after: %s\n", cfg.IngestArchiveAfter)
			}
		}
	case "api":
		if cfg.PaperlessURL == "" {
			fmt.Println("  FAIL  paperless_url: not set")
			errors++
		} else {
			fmt.Printf("  OK    paperless_url: %s\n", cfg.PaperlessURL)
		}

		// Check token file.
		tokenPath := config.DefaultTokenPath()
		info, err := os.Stat(tokenPath)
		if err != nil {
			fmt.Printf("  FAIL  token: %s does not exist\n", tokenPath)
			errors++
		} else {
			perm := info.Mode().Perm()
			if perm&0077 != 0 {
				fmt.Printf("  WARN  token: %s has permissions %04o, should be 0600\n", tokenPath, perm)
				warnings++
			} else {
				fmt.Printf("  OK    token: %s (permissions %04o)\n", tokenPath, perm)
			}
		}
	default:
		fmt.Printf("  FAIL  ingest: unknown method %q (must be directory, api, or none)\n", cfg.Ingest)
		errors++
	}

	// Check buckets are defined.
	if len(cfg.Buckets) == 0 {
		fmt.Println("  WARN  buckets: none defined")
		warnings++
	} else {
		fmt.Printf("  OK    buckets: %d defined\n", len(cfg.Buckets))
	}

	// Summary.
	fmt.Println()
	if errors > 0 {
		fmt.Printf("Validation failed: %d error(s), %d warning(s)\n", errors, warnings)
		return fmt.Errorf("validation failed with %d error(s)", errors)
	}
	if warnings > 0 {
		fmt.Printf("Validation passed with %d warning(s)\n", warnings)
	} else {
		fmt.Println("Validation passed")
	}
	return nil
}
