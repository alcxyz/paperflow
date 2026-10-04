package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/alcxyz/paperflow/internal/buildinfo"
	"github.com/alcxyz/paperflow/internal/config"
	"github.com/alcxyz/paperflow/internal/watcher"
)

// version is set at build time via ldflags.
var version = "dev"

// options holds the parsed command line.
type options struct {
	configPath  string
	overrides   config.Overrides
	showVersion bool

	// flags is kept so service install can pass set flags through.
	flags *flag.FlagSet
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts, commandArgs, err := parseArgs(args)
	if errors.Is(err, flag.ErrHelp) {
		printUsage(os.Stdout, opts.flags)
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		printUsage(os.Stderr, opts.flags)
		return 2
	}
	if opts.showVersion {
		printVersion()
		return 0
	}
	if len(commandArgs) == 0 {
		printUsage(os.Stderr, opts.flags)
		return 1
	}

	switch command := commandArgs[0]; command {
	case "init":
		err = runInit(opts)
	case "watch":
		err = runWatch(opts)
	case "validate":
		err = runValidate(opts)
	case "service":
		err = runService(opts, commandArgs[1:])
	case "help":
		printUsage(os.Stdout, opts.flags)
	case "version":
		printVersion()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		printUsage(os.Stderr, opts.flags)
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// parseArgs parses flags placed anywhere on the command line and returns the
// remaining positional arguments (the command and its subcommand).
func parseArgs(args []string) (*options, []string, error) {
	opts := &options{}
	fs := flag.NewFlagSet("paperflow", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	o := &opts.overrides
	fs.StringVar(&o.WatchDir, "watch", "", "watch `dir` instead of the configured directory")
	fs.StringVar(&o.SettleDelay, "settle-delay", "", "wait `duration` after file events before processing (default 2s)")
	fs.StringVar(&o.Ingest, "ingest", "", "ingestion `method`: directory, api, or none")
	fs.StringVar(&o.IngestDir, "ingest-dir", "", "copy ingested files to `dir`")
	fs.StringVar(&o.IngestArchiveDir, "ingest-archive-dir", "", "archive ingested files to `dir`")
	fs.StringVar(&o.IngestArchiveAfter, "ingest-archive-after", "", "archive ingested files after `duration` (default 5m)")
	fs.StringVar(&o.PaperlessURL, "paperless-url", "", "Paperless-ngx base `url` (for API ingestion)")
	fs.StringVar(&o.TokenFile, "paperless-token-file", "", "read the Paperless API token from `file`")
	fs.StringVar(&opts.configPath, "config", "", "read config from `file`")
	fs.BoolVar(&o.NoNotify, "no-notify", false, "disable notifications")
	fs.BoolVar(&o.DryRun, "dry-run", false, "log actions without moving or ingesting files")
	fs.BoolVar(&opts.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&opts.showVersion, "v", false, "print version and exit")
	opts.flags = fs

	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return opts, nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		positional = append(positional, args[0])
		args = args[1:]
	}

	if opts.configPath == "" {
		opts.configPath = config.DefaultConfigPath()
	}
	return opts, positional, nil
}

func printUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprint(w, `Usage: paperflow <command> [flags]

Commands:
  init                  Interactive setup wizard
  watch                 Start watching for files
  validate              Check config for errors
  service install       Install as a system service (systemd/launchd)
  service uninstall     Remove the system service
  service status        Show service status
  version               Print version

Flags:
`)
	fs.VisitAll(func(f *flag.Flag) {
		if f.Name == "v" {
			return
		}
		arg, usage := flag.UnquoteUsage(f)
		name := "--" + f.Name
		if arg != "" {
			name += " <" + arg + ">"
		}
		_, _ = fmt.Fprintf(w, "  %-32s %s\n", name, usage)
	})
}

func printVersion() {
	fmt.Printf("paperflow %s\n", buildinfo.Resolve(version))
}

func runWatch(opts *options) error {
	cfg, err := config.LoadConfig(opts.configPath, opts.overrides)
	if err != nil {
		return err
	}
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid configuration (run 'paperflow validate' for details):\n%w", err)
	}

	w, err := watcher.NewWatcher(cfg)
	if err != nil {
		return err
	}

	return w.Run()
}
