package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSystemdUnit_NoFlags(t *testing.T) {
	unit := generateSystemdUnit("/usr/bin/paperflow", nil)

	if !strings.Contains(unit, "ExecStart=/usr/bin/paperflow watch") {
		t.Error("unit should contain ExecStart with watch command")
	}
	if !strings.Contains(unit, "Type=simple") {
		t.Error("unit should be Type=simple")
	}
	if !strings.Contains(unit, "Restart=on-failure") {
		t.Error("unit should restart on failure")
	}
	if !strings.Contains(unit, "WantedBy=default.target") {
		t.Error("unit should be wanted by default.target")
	}
}

func TestGenerateSystemdUnit_IncludesPATH(t *testing.T) {
	t.Setenv("PATH", "/nix/profile/bin:/usr/bin")
	unit := generateSystemdUnit("/usr/bin/paperflow", nil)

	if !strings.Contains(unit, "Environment=PATH=/nix/profile/bin:/usr/bin") {
		t.Error("unit should contain Environment=PATH from current env")
	}
}

func TestGenerateSystemdUnit_WithFlags(t *testing.T) {
	unit := generateSystemdUnit("/usr/bin/paperflow", []string{"--watch", "/tmp/docs", "--no-notify"})

	expected := "ExecStart=/usr/bin/paperflow watch --watch /tmp/docs --no-notify"
	if !strings.Contains(unit, expected) {
		t.Errorf("unit should contain %q, got:\n%s", expected, unit)
	}
}

func TestGenerateLaunchdPlist_NoFlags(t *testing.T) {
	plist := generateLaunchdPlist("/usr/local/bin/paperflow", nil)

	if !strings.Contains(plist, "<string>com.alcxyz.paperflow</string>") {
		t.Error("plist should contain label")
	}
	if !strings.Contains(plist, "<string>/usr/local/bin/paperflow</string>") {
		t.Error("plist should contain executable path")
	}
	if !strings.Contains(plist, "<string>watch</string>") {
		t.Error("plist should contain watch argument")
	}
	if !strings.Contains(plist, "<key>KeepAlive</key>") {
		t.Error("plist should have KeepAlive")
	}
}

func TestGenerateLaunchdPlist_WithFlags(t *testing.T) {
	plist := generateLaunchdPlist("/opt/bin/paperflow", []string{"--config", "/etc/paperflow.toml"})

	if !strings.Contains(plist, "<string>--config</string>") {
		t.Error("plist should contain --config flag")
	}
	if !strings.Contains(plist, "<string>/etc/paperflow.toml</string>") {
		t.Error("plist should contain config path")
	}
}

func TestGenerateSystemdUnit_QuotesArguments(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	unit := generateSystemdUnit("/opt/my apps/paperflow", []string{"--watch", "/home/a/My Docs", "--ingest-dir", "/srv/100%/$x"})

	expected := `ExecStart="/opt/my apps/paperflow" watch --watch "/home/a/My Docs" --ingest-dir /srv/100%%/$$x`
	if !strings.Contains(unit, expected) {
		t.Errorf("unit should contain %q, got:\n%s", expected, unit)
	}
}

func TestGenerateSystemdUnit_QuotesPATH(t *testing.T) {
	t.Setenv("PATH", "/opt/a b/bin:/usr/bin")
	unit := generateSystemdUnit("/usr/bin/paperflow", nil)

	if !strings.Contains(unit, `Environment="PATH=/opt/a b/bin:/usr/bin"`) {
		t.Errorf("PATH with spaces should be quoted, got:\n%s", unit)
	}
}

func TestGenerateLaunchdPlist_EscapesXML(t *testing.T) {
	plist := generateLaunchdPlist("/opt/bin/paperflow", []string{"--watch", "/Users/a/R&D <docs>"})

	if !strings.Contains(plist, "<string>/Users/a/R&amp;D &lt;docs&gt;</string>") {
		t.Errorf("plist should XML-escape arguments, got:\n%s", plist)
	}
}

func TestServiceFlags(t *testing.T) {
	opts, _, err := parseArgs([]string{
		"--dry-run", "--no-notify", "--ingest=api", "--watch", "~/Inbox/",
		"--config", "rel/config.toml", "service", "install",
	})
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()

	got := serviceFlags(opts.flags)
	want := []string{
		"--config", filepath.Join(wd, "rel/config.toml"),
		"--ingest", "api",
		"--no-notify",
		"--watch", filepath.Join(home, "Inbox"),
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("serviceFlags = %q, want %q", got, want)
	}
}
