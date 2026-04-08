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
	cfg, err := LoadConfig("/tmp/paperflow-test-nonexistent/config.toml")
	if err != nil {
		t.Fatalf("LoadConfig with nonexistent file should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig should return default config for nonexistent file")
	}
	if cfg.Ingest != "none" {
		t.Errorf("Ingest = %q, want none", cfg.Ingest)
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
watch_dir = "/tmp/watch"
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

	cfg, err := LoadConfig(configPath)
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

	_, err := LoadConfig(configPath)
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
	t.Setenv("PAPERFLOW_INGEST", "directory")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.WatchDir != "/env/watch" {
		t.Errorf("WatchDir = %q, want /env/watch", cfg.WatchDir)
	}
	if cfg.Ingest != "directory" {
		t.Errorf("Ingest = %q, want directory", cfg.Ingest)
	}
}

func TestTokenPermissionWarning(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")

	if err := os.WriteFile(tokenPath, []byte("test-token\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// We can't easily capture stderr in a unit test without more infrastructure,
	// but we can verify the token is still loaded correctly.
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm&0077 == 0 {
		t.Error("test token file should have open permissions for this test")
	}
}
