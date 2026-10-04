package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.WatchDir != "~/Documents" {
		t.Errorf("WatchDir = %q, want ~/Documents", cfg.WatchDir)
	}
	if cfg.Ingest != "none" {
		t.Errorf("Ingest = %q, want none", cfg.Ingest)
	}
	if cfg.SettleDelay != "2s" {
		t.Errorf("SettleDelay = %q, want 2s", cfg.SettleDelay)
	}
	if !cfg.Notifications.Enabled {
		t.Error("Notifications.Enabled = false, want true")
	}
	if cfg.Notifications.BatchWindow != "3s" {
		t.Errorf("BatchWindow = %q, want 3s", cfg.Notifications.BatchWindow)
	}
	if len(cfg.Buckets) == 0 {
		t.Error("Buckets should not be empty")
	}
	if len(cfg.IngestTypes.Types) == 0 {
		t.Error("IngestTypes.Types should not be empty")
	}
	if len(cfg.Exclude.Patterns) == 0 {
		t.Error("Exclude.Patterns should not be empty")
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	cfg, err := LoadConfig("/tmp/paperflow-test-nonexistent/config.toml", Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig with nonexistent file should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig should return default config for nonexistent file")
	}
	if cfg.Ingest != "none" {
		t.Errorf("Ingest = %q, want none", cfg.Ingest)
	}
	if !filepath.IsAbs(cfg.WatchDir) {
		t.Errorf("WatchDir = %q, want an expanded absolute path", cfg.WatchDir)
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
watch_dir = "/tmp/watch"
settle_delay = "4s"
ingest = "directory"
ingest_dir = "/tmp/ingest"

[notifications]
enabled = false
batch_window = "5s"
app_name = "Test"

[buckets]
pdf = ["pdf"]
images = ["jpg", "png"]

[ingest_types]
types = ["pdf", "jpg"]

[exclude]
patterns = ["*.tmp"]
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath, Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.WatchDir != "/tmp/watch" {
		t.Errorf("WatchDir = %q, want /tmp/watch", cfg.WatchDir)
	}
	if cfg.Ingest != "directory" {
		t.Errorf("Ingest = %q, want directory", cfg.Ingest)
	}
	if cfg.IngestDir != "/tmp/ingest" {
		t.Errorf("IngestDir = %q, want /tmp/ingest", cfg.IngestDir)
	}
	if cfg.SettleDelay != "4s" {
		t.Errorf("SettleDelay = %q, want 4s", cfg.SettleDelay)
	}
	if cfg.Notifications.Enabled {
		t.Error("Notifications.Enabled = true, want false")
	}
	if cfg.Notifications.BatchWindow != "5s" {
		t.Errorf("BatchWindow = %q, want 5s", cfg.Notifications.BatchWindow)
	}
	if len(cfg.Buckets) != 2 {
		t.Errorf("len(Buckets) = %d, want 2", len(cfg.Buckets))
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(configPath, []byte("invalid [[[toml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(configPath, Overrides{})
	if err == nil {
		t.Error("LoadConfig should error on invalid TOML")
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/Documents", filepath.Join(home, "Documents")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		got := ExpandTilde(tt.input)
		if got != tt.want {
			t.Errorf("ExpandTilde(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
watch_dir = "/original/watch"
ingest = "none"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PAPERFLOW_WATCH_DIR", "/env/watch")
	t.Setenv("PAPERFLOW_SETTLE_DELAY", "7s")
	t.Setenv("PAPERFLOW_INGEST", "directory")

	cfg, err := LoadConfig(configPath, Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.WatchDir != "/env/watch" {
		t.Errorf("WatchDir = %q, want /env/watch", cfg.WatchDir)
	}
	if cfg.Ingest != "directory" {
		t.Errorf("Ingest = %q, want directory", cfg.Ingest)
	}
	if cfg.SettleDelay != "7s" {
		t.Errorf("SettleDelay = %q, want 7s", cfg.SettleDelay)
	}
}

func TestLoadToken(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenPath, []byte("  test-token\n"), 0600); err != nil {
		t.Fatal(err)
	}

	token, err := LoadToken(tokenPath)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if token != "test-token" {
		t.Errorf("token = %q, want test-token", token)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigPartialSectionsKeepDefaults(t *testing.T) {
	path := writeConfig(t, `
[notifications]
enabled = false
`)

	cfg, err := LoadConfig(path, Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Notifications.Enabled {
		t.Error("Notifications.Enabled = true, want false")
	}
	if cfg.Notifications.BatchWindow != "3s" {
		t.Errorf("BatchWindow = %q, want default 3s", cfg.Notifications.BatchWindow)
	}
	if cfg.Notifications.AppName != "Paperflow" {
		t.Errorf("AppName = %q, want default Paperflow", cfg.Notifications.AppName)
	}
}

func TestLoadConfigBucketsReplaceDefaults(t *testing.T) {
	path := writeConfig(t, `
[buckets]
scans = ["pdf"]

[ingest_types]
types = ["pdf"]
`)

	cfg, err := LoadConfig(path, Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Buckets) != 1 || len(cfg.Buckets["scans"]) != 1 {
		t.Errorf("Buckets = %v, want only scans", cfg.Buckets)
	}
	if len(cfg.IngestTypes.Types) != 1 {
		t.Errorf("IngestTypes = %v, want [pdf]", cfg.IngestTypes.Types)
	}
}

func TestLoadConfigNormalizesPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	path := writeConfig(t, `
watch_dir = "/tmp/watch/"
ingest_dir = "~/ingest"
`)

	cfg, err := LoadConfig(path, Overrides{IngestArchiveDir: "/tmp/archive/../archive2"})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.WatchDir != "/tmp/watch" {
		t.Errorf("WatchDir = %q, want /tmp/watch", cfg.WatchDir)
	}
	if want := filepath.Join(home, "ingest"); cfg.IngestDir != want {
		t.Errorf("IngestDir = %q, want %q", cfg.IngestDir, want)
	}
	if cfg.IngestArchiveDir != "/tmp/archive2" {
		t.Errorf("IngestArchiveDir = %q, want /tmp/archive2", cfg.IngestArchiveDir)
	}
}

func TestLoadConfigOverridesTakePrecedence(t *testing.T) {
	path := writeConfig(t, `
watch_dir = "/file/watch"
settle_delay = "4s"
`)
	t.Setenv("PAPERFLOW_WATCH_DIR", "/env/watch")
	t.Setenv("PAPERFLOW_SETTLE_DELAY", "7s")

	cfg, err := LoadConfig(path, Overrides{WatchDir: "/flag/watch", NoNotify: true, DryRun: true})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.WatchDir != "/flag/watch" {
		t.Errorf("WatchDir = %q, want /flag/watch", cfg.WatchDir)
	}
	if cfg.SettleDelay != "7s" {
		t.Errorf("SettleDelay = %q, want 7s from env", cfg.SettleDelay)
	}
	if cfg.Notifications.Enabled || !cfg.DryRun {
		t.Error("NoNotify and DryRun overrides should apply")
	}
}

func TestLoadConfigLoadsTokenWhenAPISelectedByOverride(t *testing.T) {
	path := writeConfig(t, `ingest = "none"`)
	tokenPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenPath, []byte("flag-token\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path, Overrides{Ingest: IngestAPI, TokenFile: tokenPath})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Token != "flag-token" {
		t.Errorf("Token = %q, want flag-token", cfg.Token)
	}
}

func TestLoadConfigTokenFileFromEnv(t *testing.T) {
	path := writeConfig(t, `ingest = "api"`)
	tokenPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenPath, []byte("env-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAPERFLOW_PAPERLESS_TOKEN_FILE", tokenPath)

	cfg, err := LoadConfig(path, Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want env-token", cfg.Token)
	}
}

func TestLoadConfigMissingTokenIsNotAnError(t *testing.T) {
	path := writeConfig(t, `ingest = "api"`)

	cfg, err := LoadConfig(path, Overrides{})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if err := cfg.Check(); err == nil {
		t.Error("Check should report the missing token")
	}
}

func TestCheck(t *testing.T) {
	valid := func() *Config {
		cfg := DefaultConfig()
		cfg.WatchDir = "/watch"
		cfg.IngestDir = "/ingest"
		return cfg
	}

	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{"defaults", func(*Config) {}, false},
		{"directory", func(c *Config) { c.Ingest = IngestDirectory }, false},
		{"api", func(c *Config) {
			c.Ingest = IngestAPI
			c.PaperlessURL = "https://paperless.example.com"
			c.Token = "token"
		}, false},
		{"unknown ingest", func(c *Config) { c.Ingest = "API" }, true},
		{"empty watch dir", func(c *Config) { c.WatchDir = "" }, true},
		{"bad settle delay", func(c *Config) { c.SettleDelay = "soon" }, true},
		{"negative settle delay", func(c *Config) { c.SettleDelay = "-1s" }, true},
		{"bad batch window", func(c *Config) { c.Notifications.BatchWindow = "" }, true},
		{"ingest dir is watch dir", func(c *Config) {
			c.Ingest = IngestDirectory
			c.IngestDir = c.WatchDir
		}, true},
		{"bad archive delay", func(c *Config) {
			c.Ingest = IngestDirectory
			c.IngestArchiveDir = "/archive"
			c.IngestArchiveAfter = "later"
		}, true},
		{"api without url", func(c *Config) {
			c.Ingest = IngestAPI
			c.Token = "token"
		}, true},
		{"api with relative url", func(c *Config) {
			c.Ingest = IngestAPI
			c.PaperlessURL = "paperless.local"
			c.Token = "token"
		}, true},
		{"api without token", func(c *Config) {
			c.Ingest = IngestAPI
			c.PaperlessURL = "https://paperless.example.com"
		}, true},
		{"bad exclude pattern", func(c *Config) { c.Exclude.Patterns = []string{"[abc"} }, true},
		{"bad exclude pattern after star", func(c *Config) { c.Exclude.Patterns = []string{"invoice*["} }, true},
		{"star inside class", func(c *Config) { c.Exclude.Patterns = []string{"a[*]b*", "~$*", "\\*.tmp"} }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			tt.modify(cfg)
			if err := cfg.Check(); (err != nil) != tt.wantErr {
				t.Errorf("Check() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAbsPath(t *testing.T) {
	if got := AbsPath(""); got != "" {
		t.Errorf("AbsPath(\"\") = %q, want empty", got)
	}
	if got := AbsPath("/a/b/"); got != "/a/b" {
		t.Errorf("AbsPath(/a/b/) = %q, want /a/b", got)
	}
	if got := AbsPath("rel"); !filepath.IsAbs(got) {
		t.Errorf("AbsPath(rel) = %q, want absolute", got)
	}
}
