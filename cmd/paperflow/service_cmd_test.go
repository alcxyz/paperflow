package main

import (
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

func TestFindServiceArgs(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{[]string{"service", "install"}, []string{"install"}},
		{[]string{"service", "uninstall"}, []string{"uninstall"}},
		{[]string{"--no-notify", "service", "install"}, []string{"install"}},
		{[]string{"service"}, nil},
		{[]string{"watch"}, nil},
	}

	for _, tt := range tests {
		got := findServiceArgs(tt.args)
		if len(got) != len(tt.want) {
			t.Errorf("findServiceArgs(%v) = %v, want %v", tt.args, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("findServiceArgs(%v)[%d] = %q, want %q", tt.args, i, got[i], tt.want[i])
			}
		}
	}
}
