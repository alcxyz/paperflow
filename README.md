# paperflow

A file organizer and [Paperless-ngx](https://docs.paperless-ngx.com/) ingestion tool. Watches a directory for new files, sorts them into structured subdirectories by type and date, and optionally ingests them into Paperless-ngx via local directory or API.

Single binary. Cross-platform (Linux and macOS). No race conditions. Batched notifications.

## How it works

```
~/Documents/
  invoice.pdf        <-- you drop a file here
  receipt.jpg

       |
       v  paperflow sorts by type + file date

~/Documents/
  pdf/2026/04/invoice.pdf
  images/2026/04/receipt.jpg

       |
       v  paperflow ingests to Paperless (if enabled)

~/paperless-ingest/invoice.pdf   (directory mode)
       -- or --
POST /api/documents/post_document/  (API mode)
```

## Features

- **File watching** -- monitors a directory for new or moved files using OS-native events (inotify on Linux, FSEvents on macOS)
- **Configurable type-based sorting** -- files are moved into subdirectories based on extension, fully customizable via config
- **Date-based structure** -- sorted files are placed in `<type>/<year>/<month>/` based on the file's modification time
- **Paperless ingestion** -- ingestible file types are forwarded to Paperless-ngx via one of:
  - `directory` -- copies to a local ingest directory that Paperless watches
  - `api` -- uploads directly to the Paperless-ngx REST API
  - `none` -- sorting only, no ingestion
- **Collision handling** -- if a file with the same name already exists at the destination, a timestamp suffix is appended (e.g. `invoice_1775660096.pdf`)
- **Batched notifications** -- multiple files processed in quick succession produce a single summary notification instead of one per file
- **Startup notification** -- confirms the watch directory and ingest mode on startup
- **API auth check** -- verifies Paperless API connectivity and token validity before starting the watcher (fails fast on bad credentials)
- **Exclude patterns** -- glob-based patterns to ignore files (e.g. `*.tmp`, `~$*`)
- **Interactive setup** -- `paperflow init` walks you through configuration
- **XDG-compliant** -- config stored in `~/.config/paperflow/`

## Installation

### Homebrew (macOS)

```bash
brew install alcxyz/tap/paperflow
```

### Arch Linux (AUR)

```bash
yay -S paperflow-bin
```

### Nix flake

```nix
# flake.nix
{
  inputs.paperflow.url = "github:alcxyz/paperflow";

  # Use as: inputs.paperflow.packages.${system}.default
}
```

### Go

```bash
go install github.com/alcxyz/paperflow@latest
```

### From source

```bash
git clone https://github.com/alcxyz/paperflow.git
cd paperflow
go build -o paperflow ./cmd/paperflow
```

### GitHub releases

Pre-built binaries for Linux and macOS (amd64/arm64) are available on the [releases page](https://github.com/alcxyz/paperflow/releases).

## Quick start

```bash
# Interactive setup -- creates config and stores secrets
paperflow init

# Start watching
paperflow watch
```

## Configuration

paperflow uses XDG base directories. Config lives at `~/.config/paperflow/config.toml` (or `$XDG_CONFIG_HOME/paperflow/config.toml`).

Run `paperflow init` for an interactive setup wizard, or create the config manually:

```toml
# ~/.config/paperflow/config.toml

# Directory to watch for new files
watch_dir = "~/Documents"

# Ingestion method: "directory", "api", or "none"
ingest = "directory"

# Local ingest directory (when ingest = "directory")
ingest_dir = "~/paperless-ingest"

# Paperless API settings (when ingest = "api")
# paperless_url = "https://paperless.example.com"
# Token is stored separately in ~/.config/paperflow/token

[notifications]
# Enable/disable desktop notifications
enabled = true
# Duration to wait for more files before sending a batched notification
batch_window = "3s"
# App name shown in notifications
app_name = "Paperflow"

[buckets]
# Map bucket names to file extensions
# Files matching these extensions are sorted into <bucket>/<year>/<month>/
pdf    = ["pdf"]
images = ["jpg", "jpeg", "png", "gif", "webp", "tiff", "tif"]
docx   = ["docx", "doc", "odt", "rtf"]
xlsx   = ["xlsx", "xls", "ods"]
# Files not matching any bucket go to "misc/"

[ingest_types]
# Which file types are eligible for Paperless ingestion
# Must be a subset of extensions defined in [buckets]
# Files in misc/ are never ingested regardless of this setting
types = ["pdf", "jpg", "jpeg", "png", "gif", "webp", "tiff", "tif", "docx", "odt", "xlsx"]

[exclude]
# Glob patterns for files to ignore
patterns = [
  "*.tmp",
  "*.part",
  "~$*",
  ".~lock.*",
]
```

### Secrets

The Paperless API token is stored separately at `~/.config/paperflow/token` with `0600` permissions. `paperflow init` handles this automatically when you choose API ingestion.

paperflow warns on startup if the token file has overly permissive permissions.

### Flags

Flags override config values for a single run:

| Flag | Description |
|------|-------------|
| `--watch` | Override watch directory |
| `--ingest` | Override ingestion method |
| `--ingest-dir` | Override ingest directory |
| `--paperless-url` | Paperless-ngx base URL (for API ingestion) |
| `--paperless-token-file` | Path to file containing Paperless API token |
| `--config` | Path to config file (default: `$XDG_CONFIG_HOME/paperflow/config.toml`) |
| `--no-notify` | Disable notifications for this run |
| `--dry-run` | Log what would happen without moving or ingesting files |

## Commands

### `paperflow init`

Interactive setup wizard:

1. Asks for the directory to watch
2. Asks for the ingestion method (directory, api, or none)
3. If directory: asks for the ingest directory path
4. If api: asks for Paperless URL and API token
5. Writes config to `~/.config/paperflow/config.toml`
6. Stores token (if applicable) at `~/.config/paperflow/token` with `0600` permissions

Re-running `paperflow init` offers to update the existing config.

### `paperflow watch`

Starts watching the configured directory. This is the main runtime command. Runs in the foreground, intended to be managed by a service manager (systemd, launchd) or run in a terminal.

On startup, paperflow sends a desktop notification confirming the watch directory and ingest mode. When using API ingestion, it verifies the Paperless API connection and token before starting the watcher.

### `paperflow validate`

Checks the config file for errors and verifies that:
- Watch directory exists
- Ingest directory exists (if using directory ingestion)
- Paperless URL is reachable (if using API ingestion)
- Token file exists and has correct permissions

## Ingestible file types

By default, the following file types are forwarded to Paperless when ingestion is enabled:

`pdf`, `jpg`, `jpeg`, `png`, `gif`, `webp`, `tiff`, `tif`, `docx`, `odt`, `xlsx`

This is configurable via `[ingest_types]` in the config. Files sorted into `misc/` are never ingested regardless of configuration.

## Notifications

paperflow sends desktop notifications via `notify-send` on Linux and `osascript` on macOS.

When multiple files arrive within the batch window (default 3 seconds), a single summary notification is sent:

- **1 file**: `Sorted: invoice.pdf -> pdf/2026/04/`
- **5 files**: `Sorted 5 files: invoice.pdf -> pdf/2026/04/, receipt.jpg -> images/2026/04/, ...`

Ingestion notifications follow the same batching pattern.

## Running as a service

### systemd (Linux)

```ini
# ~/.config/systemd/user/paperflow.service
[Unit]
Description=Paperflow document organizer
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/paperflow watch
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=default.target
```

```bash
systemctl --user enable --now paperflow
```

### launchd (macOS)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.alcxyz.paperflow</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/paperflow</string>
        <string>watch</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

### NixOS / home-manager

```nix
systemd.user.services.paperflow = {
  Unit = {
    Description = "Paperflow document organizer";
    After = [ "network-online.target" ];
    Wants = [ "network-online.target" ];
  };
  Service = {
    Type = "simple";
    ExecStart = "${pkgs.paperflow}/bin/paperflow watch";
    Restart = "on-failure";
    RestartSec = "5s";
  };
  Install.WantedBy = [ "default.target" ];
};
```

## Project structure

```
paperflow/
  cmd/paperflow/
    main.go              # Entry point, CLI parsing, flags
    init_cmd.go          # Interactive setup wizard
    validate_cmd.go      # Config validation command
  internal/
    bucket/
      bucket.go          # File type -> bucket mapping
      bucket_test.go
    config/
      config.go          # Config loading, XDG paths, defaults
      config_test.go
    ingest/
      api.go             # Paperless API ingestion + auth check
      directory.go       # Directory-based ingestion
    notify/
      notify.go          # Batched notification logic
      notify_linux.go    # Linux notification (notify-send)
      notify_darwin.go   # macOS notification (osascript)
      notify_test.go
    organizer/
      organizer.go       # File sorting logic
      organizer_test.go
    watcher/
      watcher.go         # File system watching (fsnotify)
  go.mod
  go.sum
  flake.nix
  default.nix
  .goreleaser.yml
  README.md
  LICENSE
```

## License

MIT
