package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/alcxyz/paperflow/internal/config"
)

func runInit(opts *options) error {
	configPath := config.ExpandTilde(opts.configPath)
	defaults := config.DefaultConfig()

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
	if ingest != config.IngestDirectory && ingest != config.IngestAPI && ingest != config.IngestNone {
		return fmt.Errorf("invalid ingestion method: %s", ingest)
	}

	var ingestDir, archiveDir, archiveAfter, paperlessURL, token string

	switch ingest {
	case config.IngestDirectory:
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

	case config.IngestAPI:
		fmt.Print("Paperless URL (e.g. https://paperless.example.com): ")
		paperlessURL, _ = reader.ReadString('\n')
		paperlessURL = strings.TrimSpace(paperlessURL)
		if paperlessURL == "" {
			return fmt.Errorf("paperless URL is required for API ingestion")
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
	fmt.Fprintf(&b, "settle_delay = %q\n", defaults.SettleDelay)
	fmt.Fprintf(&b, "ingest = %q\n", ingest)

	if ingest == config.IngestDirectory {
		fmt.Fprintf(&b, "ingest_dir = %q\n", ingestDir)
		if archiveDir != "" {
			fmt.Fprintf(&b, "ingest_archive_dir = %q\n", archiveDir)
			fmt.Fprintf(&b, "ingest_archive_after = %q\n", archiveAfter)
		}
	}
	if ingest == config.IngestAPI {
		fmt.Fprintf(&b, "paperless_url = %q\n", paperlessURL)
		fmt.Fprintf(&b, "# Token stored separately in %s\n", config.DefaultTokenPath())
	}

	n := defaults.Notifications
	fmt.Fprintf(&b, "\n[notifications]\nenabled = %t\nbatch_window = %q\napp_name = %q\n", n.Enabled, n.BatchWindow, n.AppName)

	b.WriteString("\n[buckets]\n")
	var names []string
	for name := range defaults.Buckets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, "%s = %s\n", name, tomlList(defaults.Buckets[name]))
	}

	fmt.Fprintf(&b, "\n[ingest_types]\ntypes = %s\n", tomlList(defaults.IngestTypes.Types))
	fmt.Fprintf(&b, "\n[exclude]\npatterns = %s\n", tomlList(defaults.Exclude.Patterns))

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
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
			return fmt.Errorf("creating token directory: %w", err)
		}
		if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0600); err != nil {
			return fmt.Errorf("writing token: %w", err)
		}
		fmt.Printf("Token written to %s\n", tokenPath)
	}

	fmt.Println("\nSetup complete. Run 'paperflow watch' to start.")
	return nil
}

// tomlList formats values as a TOML array of strings.
func tomlList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
