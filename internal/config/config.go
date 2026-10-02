package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Ingestion methods accepted in Config.Ingest.
const (
	IngestNone      = "none"
	IngestDirectory = "directory"
	IngestAPI       = "api"
)

// Config represents the paperflow configuration.
type Config struct {
	WatchDir    string `toml:"watch_dir"`
	SettleDelay string `toml:"settle_delay"`
	Ingest      string `toml:"ingest"`
	IngestDir   string `toml:"ingest_dir"`

	PaperlessURL       string `toml:"paperless_url"`
	IngestArchiveDir   string `toml:"ingest_archive_dir"`
	IngestArchiveAfter string `toml:"ingest_archive_after"`

	Notifications NotificationsConfig `toml:"notifications"`
	Buckets       map[string][]string `toml:"buckets"`
	IngestTypes   IngestTypesConfig   `toml:"ingest_types"`
	Exclude       ExcludeConfig       `toml:"exclude"`

	// DryRun is set via flag only, not in the config file.
	DryRun bool `toml:"-"`

	// TokenFile is the path of the Paperless API token file. It is set via
	// environment or flag only, keeping config.toml free of secret wiring.
	TokenFile string `toml:"-"`

	// Token is loaded from TokenFile, not from config.toml.
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

// Overrides holds command-line values, which take precedence over the config
// file and environment. Empty strings and false leave a setting unchanged.
type Overrides struct {
	WatchDir           string
	SettleDelay        string
	Ingest             string
	IngestDir          string
	IngestArchiveDir   string
	IngestArchiveAfter string
	PaperlessURL       string
	TokenFile          string
	NoNotify           bool
	DryRun             bool
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		WatchDir:           "~/Documents",
		SettleDelay:        "2s",
		Ingest:             IngestNone,
		IngestDir:          "~/paperless-ingest",
		IngestArchiveAfter: "5m",
		TokenFile:          DefaultTokenPath(),
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

// XDGConfigHome returns $XDG_CONFIG_HOME, or ~/.config when it is unset.
func XDGConfigHome() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return filepath.Join(home, ".config")
}

// DefaultConfigPath returns the default config file path, respecting XDG.
func DefaultConfigPath() string {
	return filepath.Join(XDGConfigHome(), "paperflow", "config.toml")
}

// DefaultTokenPath returns the default token file path, respecting XDG.
func DefaultTokenPath() string {
	return filepath.Join(XDGConfigHome(), "paperflow", "token")
}

// LoadConfig builds the configuration from defaults, the config file at path
// (if it exists), PAPERFLOW_ environment variables, and o, in increasing
// order of precedence. Paths are made absolute, and the API token is loaded
// when API ingestion is selected.
func LoadConfig(path string, o Overrides) (*Config, error) {
	cfg := DefaultConfig()
	if err := cfg.loadFile(ExpandTilde(path)); err != nil {
		return nil, err
	}
	applyEnvOverrides(cfg)
	o.apply(cfg)

	cfg.WatchDir = AbsPath(cfg.WatchDir)
	cfg.IngestDir = AbsPath(cfg.IngestDir)
	cfg.IngestArchiveDir = AbsPath(cfg.IngestArchiveDir)
	cfg.TokenFile = AbsPath(cfg.TokenFile)

	if cfg.Ingest == IngestAPI {
		token, err := LoadToken(cfg.TokenFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("loading token: %w", err)
		}
		cfg.Token = token
	}

	return cfg, nil
}

// loadFile decodes the config file over the current values. Settings the file
// omits keep their defaults. A missing file is not an error.
func (c *Config) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	// Configured buckets replace the defaults instead of merging with them.
	defaultBuckets := c.Buckets
	c.Buckets = nil
	if err := toml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	if c.Buckets == nil {
		c.Buckets = defaultBuckets
	}
	return nil
}

// LoadToken reads the API token from path and warns about loose permissions.
func LoadToken(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if perm := info.Mode().Perm(); perm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: token file %s has permissions %04o, should be 0600\n", path, perm)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// applyEnvOverrides applies PAPERFLOW_ environment variable overrides to config.
func applyEnvOverrides(cfg *Config) {
	setIfNotEmpty(&cfg.WatchDir, os.Getenv("PAPERFLOW_WATCH_DIR"))
	setIfNotEmpty(&cfg.SettleDelay, os.Getenv("PAPERFLOW_SETTLE_DELAY"))
	setIfNotEmpty(&cfg.Ingest, os.Getenv("PAPERFLOW_INGEST"))
	setIfNotEmpty(&cfg.IngestDir, os.Getenv("PAPERFLOW_INGEST_DIR"))
	setIfNotEmpty(&cfg.PaperlessURL, os.Getenv("PAPERFLOW_PAPERLESS_URL"))
	setIfNotEmpty(&cfg.TokenFile, os.Getenv("PAPERFLOW_PAPERLESS_TOKEN_FILE"))
	setIfNotEmpty(&cfg.IngestArchiveDir, os.Getenv("PAPERFLOW_INGEST_ARCHIVE_DIR"))
	setIfNotEmpty(&cfg.IngestArchiveAfter, os.Getenv("PAPERFLOW_INGEST_ARCHIVE_AFTER"))
	if v := os.Getenv("PAPERFLOW_NO_NOTIFY"); v == "1" || v == "true" {
		cfg.Notifications.Enabled = false
	}
}

func (o Overrides) apply(cfg *Config) {
	setIfNotEmpty(&cfg.WatchDir, o.WatchDir)
	setIfNotEmpty(&cfg.SettleDelay, o.SettleDelay)
	setIfNotEmpty(&cfg.Ingest, o.Ingest)
	setIfNotEmpty(&cfg.IngestDir, o.IngestDir)
	setIfNotEmpty(&cfg.PaperlessURL, o.PaperlessURL)
	setIfNotEmpty(&cfg.TokenFile, o.TokenFile)
	setIfNotEmpty(&cfg.IngestArchiveDir, o.IngestArchiveDir)
	setIfNotEmpty(&cfg.IngestArchiveAfter, o.IngestArchiveAfter)
	if o.NoNotify {
		cfg.Notifications.Enabled = false
	}
	if o.DryRun {
		cfg.DryRun = true
	}
}

func setIfNotEmpty(dst *string, value string) {
	if value != "" {
		*dst = value
	}
}

// Check reports settings that would keep paperflow from running correctly.
// It does not touch the filesystem or network. Multiple problems are joined.
func (c *Config) Check() error {
	var errs []error

	if c.WatchDir == "" {
		errs = append(errs, errors.New("watch_dir: must be set"))
	}
	errs = append(errs, checkDuration("settle_delay", c.SettleDelay))
	errs = append(errs, checkDuration("notifications.batch_window", c.Notifications.BatchWindow))

	switch c.Ingest {
	case IngestNone:
	case IngestDirectory:
		switch c.IngestDir {
		case "":
			errs = append(errs, errors.New("ingest_dir: must be set for directory ingestion"))
		case c.WatchDir:
			errs = append(errs, errors.New("ingest_dir: must differ from watch_dir"))
		}
		if c.IngestArchiveDir != "" {
			errs = append(errs, checkDuration("ingest_archive_after", c.IngestArchiveAfter))
		}
	case IngestAPI:
		if u, err := url.Parse(c.PaperlessURL); err != nil || u.Scheme == "" || u.Host == "" {
			errs = append(errs, fmt.Errorf("paperless_url: must be an absolute URL for API ingestion (got %q)", c.PaperlessURL))
		}
		if c.Token == "" {
			errs = append(errs, fmt.Errorf("token: no Paperless API token found in %s", c.TokenFile))
		}
	default:
		errs = append(errs, fmt.Errorf("ingest: unknown method %q (must be directory, api, or none)", c.Ingest))
	}

	for _, pattern := range c.Exclude.Patterns {
		if _, err := filepath.Match(pattern, ""); err != nil {
			errs = append(errs, fmt.Errorf("exclude: invalid pattern %q: %w", pattern, err))
		}
	}

	return errors.Join(errs...)
}

func checkDuration(name, value string) error {
	d, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q", name, value)
	}
	if d < 0 {
		return fmt.Errorf("%s: must not be negative (%s)", name, value)
	}
	return nil
}

// AbsPath expands a leading ~ and returns a clean absolute path. Empty paths
// stay empty.
func AbsPath(path string) string {
	if path == "" {
		return ""
	}
	path = ExpandTilde(path)
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return filepath.Clean(path)
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
