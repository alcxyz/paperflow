package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alcxyz/paperflow/internal/config"
)

func runInit(f flags) error {
	configPath := f.config
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	reader := bufio.NewReader(os.Stdin)

	// Check if config already exists.
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Update existing config? [y/N] ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println("paperflow setup")
	fmt.Println()

	// Watch directory.
	fmt.Print("Directory to watch [~/Documents]: ")
	watchDir, _ := reader.ReadString('\n')
	watchDir = strings.TrimSpace(watchDir)
	if watchDir == "" {
		watchDir = "~/Documents"
	}

	// Ingestion method.
	fmt.Print("Ingestion method (directory, api, none) [none]: ")
	ingest, _ := reader.ReadString('\n')
	ingest = strings.TrimSpace(strings.ToLower(ingest))
	if ingest == "" {
		ingest = "none"
	}
	if ingest != "directory" && ingest != "api" && ingest != "none" {
		return fmt.Errorf("invalid ingestion method: %s", ingest)
	}

	var ingestDir, archiveDir, archiveAfter, paperlessURL, token string

	switch ingest {
	case "directory":
		fmt.Print("Ingest directory [~/paperless-ingest]: ")
		ingestDir, _ = reader.ReadString('\n')
		ingestDir = strings.TrimSpace(ingestDir)
		if ingestDir == "" {
			ingestDir = "~/paperless-ingest"
		}

		fmt.Print("Archive ingested files to prevent re-ingestion on Paperless restart? [y/N] ")
		archiveAnswer, _ := reader.ReadString('\n')
		archiveAnswer = strings.TrimSpace(strings.ToLower(archiveAnswer))
		if archiveAnswer == "y" || archiveAnswer == "yes" {
			fmt.Print("Archive directory [~/paperflow-archive]: ")
			archiveDir, _ = reader.ReadString('\n')
			archiveDir = strings.TrimSpace(archiveDir)
			if archiveDir == "" {
				archiveDir = "~/paperflow-archive"
			}
			fmt.Print("Archive delay [5m]: ")
			archiveAfter, _ = reader.ReadString('\n')
			archiveAfter = strings.TrimSpace(archiveAfter)
			if archiveAfter == "" {
				archiveAfter = "5m"
			}
		}

	case "api":
		fmt.Print("Paperless URL (e.g. https://paperless.example.com): ")
		paperlessURL, _ = reader.ReadString('\n')
		paperlessURL = strings.TrimSpace(paperlessURL)
		if paperlessURL == "" {
			return fmt.Errorf("Paperless URL is required for API ingestion")
		}

		fmt.Print("Paperless API token: ")
		token, _ = reader.ReadString('\n')
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("API token is required for API ingestion")
		}
	}

	// Build the config file content.
	var b strings.Builder
	fmt.Fprintf(&b, "# paperflow config\n\n")
	fmt.Fprintf(&b, "watch_dir = %q\n", watchDir)
	fmt.Fprintf(&b, "ingest = %q\n", ingest)

	if ingest == "directory" {
		fmt.Fprintf(&b, "ingest_dir = %q\n", ingestDir)
		if archiveDir != "" {
			fmt.Fprintf(&b, "ingest_archive_dir = %q\n", archiveDir)
			fmt.Fprintf(&b, "ingest_archive_after = %q\n", archiveAfter)
		}
	}
	if ingest == "api" {
		fmt.Fprintf(&b, "paperless_url = %q\n", paperlessURL)
		fmt.Fprintf(&b, "# Token stored separately in %s\n", config.DefaultTokenPath())
	}

	b.WriteString("\n[notifications]\n")
	b.WriteString("enabled = true\n")
	b.WriteString("batch_window = \"3s\"\n")
	b.WriteString("app_name = \"Paperflow\"\n")

	b.WriteString("\n[buckets]\n")
	b.WriteString("pdf    = [\"pdf\"]\n")
	b.WriteString("images = [\"jpg\", \"jpeg\", \"png\", \"gif\", \"webp\", \"tiff\", \"tif\"]\n")
	b.WriteString("docx   = [\"docx\", \"doc\", \"odt\", \"rtf\"]\n")
	b.WriteString("xlsx   = [\"xlsx\", \"xls\", \"ods\"]\n")

	b.WriteString("\n[ingest_types]\n")
	b.WriteString("types = [\"pdf\", \"jpg\", \"jpeg\", \"png\", \"gif\", \"webp\", \"tiff\", \"tif\", \"docx\", \"odt\", \"xlsx\"]\n")

	b.WriteString("\n[exclude]\n")
	b.WriteString("patterns = [\"*.tmp\", \"*.part\", \"~$*\", \".~lock.*\"]\n")

	// Ensure config directory exists.
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Write config file.
	if err := os.WriteFile(configPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	fmt.Printf("Config written to %s\n", configPath)

	// Write token file if API mode.
	if token != "" {
		tokenPath := config.DefaultTokenPath()
		if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0600); err != nil {
			return fmt.Errorf("writing token: %w", err)
		}
		fmt.Printf("Token written to %s\n", tokenPath)
	}

	fmt.Println("\nSetup complete. Run 'paperflow watch' to start.")
	return nil
}
