package main

import (
	"fmt"
	"os"

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
