package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/watcher"
)

// version is set at build time via ldflags.
var version = "dev"

// flags holds CLI flag values that override config.
type flags struct {
	watchDir           string
	ingest             string
	ingestDir          string
	paperlessURL       string
	paperlessTokenFile string
	config             string
	noNotify           bool
	dryRun             bool
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	f := parseFlags(args)
	command := findCommand(args)

	switch command {
	case "init":
		if err := runInit(f); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "watch":
		if err := runWatch(f); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := runValidate(f); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Printf("paperflow %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: paperflow <command> [flags]

Commands:
  init       Interactive setup wizard
  watch      Start watching for files
  validate   Check config for errors

Flags:
  --watch <dir>       Override watch directory
  --ingest <method>   Override ingestion method (directory, api, none)
  --ingest-dir <dir>          Override ingest directory
  --paperless-url <url>       Paperless-ngx base URL (for API ingestion)
  --paperless-token-file <p>  Path to Paperless API token file
  --config <path>             Path to config file
  --no-notify         Disable notifications
  --dry-run           Log actions without moving or ingesting files`)
}

// findCommand returns the first non-flag argument.
func findCommand(args []string) string {
	skip := false
	for _, arg := range args {
		if skip {
			skip = false
			continue
		}
		if strings.HasPrefix(arg, "--") {
			// Flags that take a value.
			switch arg {
			case "--watch", "--ingest", "--ingest-dir", "--paperless-url", "--paperless-token-file", "--config":
				skip = true
			}
			continue
		}
		return arg
	}
	return ""
}

// parseFlags extracts flag values from args.
func parseFlags(args []string) flags {
	var f flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--watch":
			if i+1 < len(args) {
				i++
				f.watchDir = args[i]
			}
		case "--ingest":
			if i+1 < len(args) {
				i++
				f.ingest = args[i]
			}
		case "--ingest-dir":
			if i+1 < len(args) {
				i++
				f.ingestDir = args[i]
			}
		case "--paperless-url":
			if i+1 < len(args) {
				i++
				f.paperlessURL = args[i]
			}
		case "--paperless-token-file":
			if i+1 < len(args) {
				i++
				f.paperlessTokenFile = args[i]
			}
		case "--config":
			if i+1 < len(args) {
				i++
				f.config = args[i]
			}
		case "--no-notify":
			f.noNotify = true
		case "--dry-run":
			f.dryRun = true
		}
	}
	return f
}

func runWatch(f flags) error {
	cfg, err := loadConfigWithFlags(f)
	if err != nil {
		return err
	}

	w, err := watcher.NewWatcher(cfg)
	if err != nil {
		return err
	}

	return w.Run()
}

// loadConfigWithFlags loads config and applies flag overrides.
func loadConfigWithFlags(f flags) (*config.Config, error) {
	path := f.config
	if path == "" {
		path = config.DefaultConfigPath()
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	if f.watchDir != "" {
		cfg.WatchDir = f.watchDir
	}
	if f.ingest != "" {
		cfg.Ingest = f.ingest
	}
	if f.ingestDir != "" {
		cfg.IngestDir = f.ingestDir
	}
	if f.paperlessURL != "" {
		cfg.PaperlessURL = f.paperlessURL
	}
	if f.paperlessTokenFile != "" {
		data, err := os.ReadFile(f.paperlessTokenFile)
		if err != nil {
			return nil, fmt.Errorf("reading token file: %w", err)
		}
		cfg.Token = strings.TrimSpace(string(data))
	}
	if f.noNotify {
		cfg.Notifications.Enabled = false
	}
	if f.dryRun {
		cfg.DryRun = true
	}

	return cfg, nil
}
