package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents the paperflow configuration.
type Config struct {
	WatchDir  string `toml:"watch_dir"`
	Ingest    string `toml:"ingest"`
	IngestDir string `toml:"ingest_dir"`

	PaperlessURL       string `toml:"paperless_url"`
	IngestArchiveDir   string `toml:"ingest_archive_dir"`
	IngestArchiveAfter string `toml:"ingest_archive_after"`

	Notifications NotificationsConfig `toml:"notifications"`
	Buckets       map[string][]string `toml:"buckets"`
	IngestTypes   IngestTypesConfig   `toml:"ingest_types"`
	Exclude       ExcludeConfig       `toml:"exclude"`

	// DryRun is set via flag only, not in the config file.
	DryRun bool `toml:"-"`

	// Token is loaded from a separate file, not from config.toml.
	Token string `toml:"-"`
}

// NotificationsConfig holds notification settings.
type NotificationsConfig struct {
	Enabled     bool   `toml:"enabled"`
	BatchWindow string `toml:"batch_window"`
	AppName     string `toml:"app_name"`
}

// IngestTypesConfig holds the list of ingestible file types.
type IngestTypesConfig struct {
	Types []string `toml:"types"`
}

// ExcludeConfig holds glob patterns for files to ignore.
type ExcludeConfig struct {
	Patterns []string `toml:"patterns"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		WatchDir:           "~/Documents",
		Ingest:             "none",
		IngestDir:          "~/paperless-ingest",
		IngestArchiveAfter: "5m",
		Notifications: NotificationsConfig{
			Enabled:     true,
			BatchWindow: "3s",
			AppName:     "Paperflow",
		},
		Buckets: map[string][]string{
			"pdf":    {"pdf"},
			"images": {"jpg", "jpeg", "png", "gif", "webp", "tiff", "tif"},
			"docx":   {"docx", "doc", "odt", "rtf"},
			"xlsx":   {"xlsx", "xls", "ods"},
		},
		IngestTypes: IngestTypesConfig{
			Types: []string{"pdf", "jpg", "jpeg", "png", "gif", "webp", "tiff", "tif", "docx", "odt", "xlsx"},
		},
		Exclude: ExcludeConfig{
			Patterns: []string{"*.tmp", "*.part", "~$*", ".~lock.*"},
		},
	}
}

// DefaultConfigPath returns the default config file path, respecting XDG.
func DefaultConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "~"
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "paperflow", "config.toml")
}

// DefaultTokenPath returns the default token file path, respecting XDG.
func DefaultTokenPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "~"
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "paperflow", "token")
}

// LoadConfig loads configuration from the given path.
// If the file doesn't exist, it returns default config.
// Environment variables with PAPERFLOW_ prefix override config values.
func LoadConfig(path string) (*Config, error) {
	path = ExpandTilde(path)

	defaults := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(defaults)
			return defaults, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Fill in defaults for unset fields.
	if cfg.WatchDir == "" {
		cfg.WatchDir = defaults.WatchDir
	}
	if cfg.Ingest == "" {
		cfg.Ingest = defaults.Ingest
	}
	if cfg.IngestDir == "" {
		cfg.IngestDir = defaults.IngestDir
	}
	if cfg.IngestArchiveAfter == "" {
		cfg.IngestArchiveAfter = defaults.IngestArchiveAfter
	}
	if cfg.Notifications.BatchWindow == "" {
		cfg.Notifications = defaults.Notifications
	}
	if cfg.Buckets == nil {
		cfg.Buckets = defaults.Buckets
	}
	if cfg.IngestTypes.Types == nil {
		cfg.IngestTypes = defaults.IngestTypes
	}
	if cfg.Exclude.Patterns == nil {
		cfg.Exclude = defaults.Exclude
	}

	// Expand tildes in paths.
	cfg.WatchDir = ExpandTilde(cfg.WatchDir)
	cfg.IngestDir = ExpandTilde(cfg.IngestDir)
	cfg.IngestArchiveDir = ExpandTilde(cfg.IngestArchiveDir)

	// Apply environment variable overrides.
	applyEnvOverrides(&cfg)

	// Load token from separate file.
	if cfg.Ingest == "api" {
		token, err := LoadToken()
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading token: %w", err)
		}
		cfg.Token = token
	}

	return &cfg, nil
}

// LoadToken reads the API token from the token file and warns about permissions.
func LoadToken() (string, error) {
	tokenPath := DefaultTokenPath()

	info, err := os.Stat(tokenPath)
	if err != nil {
		return "", err
	}

	// Warn if permissions are too open.
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: token file %s has permissions %04o, should be 0600\n", tokenPath, perm)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// applyEnvOverrides applies PAPERFLOW_ environment variable overrides to config.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PAPERFLOW_WATCH_DIR"); v != "" {
		cfg.WatchDir = ExpandTilde(v)
	}
	if v := os.Getenv("PAPERFLOW_INGEST"); v != "" {
		cfg.Ingest = v
	}
	if v := os.Getenv("PAPERFLOW_INGEST_DIR"); v != "" {
		cfg.IngestDir = ExpandTilde(v)
	}
	if v := os.Getenv("PAPERFLOW_PAPERLESS_URL"); v != "" {
		cfg.PaperlessURL = v
	}
	if v := os.Getenv("PAPERFLOW_INGEST_ARCHIVE_DIR"); v != "" {
		cfg.IngestArchiveDir = ExpandTilde(v)
	}
	if v := os.Getenv("PAPERFLOW_INGEST_ARCHIVE_AFTER"); v != "" {
		cfg.IngestArchiveAfter = v
	}
	if v := os.Getenv("PAPERFLOW_NO_NOTIFY"); v == "1" || v == "true" {
		cfg.Notifications.Enabled = false
	}
}

// ExpandTilde replaces a leading ~ with the user's home directory.
func ExpandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
