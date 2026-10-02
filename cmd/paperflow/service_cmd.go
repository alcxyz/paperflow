package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alcxyz/paperflow/internal/config"
)

func runService(opts *options, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: paperflow service [install|uninstall|status]")
	}

	switch args[0] {
	case "install":
		return serviceInstall(opts)
	case "uninstall":
		return serviceUninstall()
	case "status":
		return serviceStatus()
	default:
		return fmt.Errorf("unknown service subcommand: %s", args[0])
	}
}

func serviceInstall(opts *options) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	extraFlags := serviceFlags(opts.flags)

	switch runtime.GOOS {
	case "linux":
		return installSystemd(exePath, extraFlags)
	case "darwin":
		return installLaunchd(exePath, extraFlags)
	default:
		return fmt.Errorf("service install not supported on %s", runtime.GOOS)
	}
}

// serviceFlags returns the flags set on the command line as arguments for the
// service's watch command. Path flags are made absolute because the service
// does not run in the caller's working directory.
func serviceFlags(fs *flag.FlagSet) []string {
	var args []string
	fs.Visit(func(f *flag.Flag) {
		value := f.Value.String()
		switch f.Name {
		case "dry-run", "version", "v":
			return
		case "watch", "ingest-dir", "ingest-archive-dir", "paperless-token-file", "config":
			value = config.AbsPath(value)
		}

		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			if value == "true" {
				args = append(args, "--"+f.Name)
			}
			return
		}
		args = append(args, "--"+f.Name, value)
	})
	return args
}

func serviceUninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	default:
		return fmt.Errorf("service uninstall not supported on %s", runtime.GOOS)
	}
}

func serviceStatus() error {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("systemctl", "--user", "status", "paperflow")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run() // systemctl returns non-zero for inactive services
		return nil
	case "darwin":
		cmd := exec.Command("launchctl", "list", "com.alcxyz.paperflow")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		return nil
	default:
		return fmt.Errorf("service status not supported on %s", runtime.GOOS)
	}
}

// systemd

const systemdServiceName = "paperflow.service"

func systemdServicePath() string {
	return filepath.Join(config.XDGConfigHome(), "systemd", "user", systemdServiceName)
}

func generateSystemdUnit(exePath string, extraFlags []string) string {
	words := []string{systemdExecArg(exePath), "watch"}
	for _, f := range extraFlags {
		words = append(words, systemdExecArg(f))
	}
	execStart := strings.Join(words, " ")

	// Capture current PATH so the service can find tools like notify-send.
	envLine := ""
	if p := os.Getenv("PATH"); p != "" {
		envLine = "Environment=" + systemdQuote("PATH="+p) + "\n"
	}

	return fmt.Sprintf(`[Unit]
Description=Paperflow document organizer
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
%sRestart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
`, execStart, envLine)
}

// systemdQuote escapes specifiers in s and quotes it when needed, so systemd
// reads it as a single literal word.
func systemdQuote(s string) string {
	s = strings.ReplaceAll(s, "%", "%%")
	if s != "" && !strings.ContainsAny(s, " \t\n\"'\\;") {
		return s
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s) + `"`
}

// systemdExecArg quotes an ExecStart argument, where $ also needs escaping.
func systemdExecArg(s string) string {
	return systemdQuote(strings.ReplaceAll(s, "$", "$$"))
}

func installSystemd(exePath string, extraFlags []string) error {
	servicePath := systemdServicePath()

	if err := os.MkdirAll(filepath.Dir(servicePath), 0755); err != nil {
		return fmt.Errorf("creating systemd user directory: %w", err)
	}

	unit := generateSystemdUnit(exePath, extraFlags)
	if err := os.WriteFile(servicePath, []byte(unit), 0644); err != nil {
		return fmt.Errorf("writing service file: %w", err)
	}
	fmt.Printf("Wrote %s\n", servicePath)

	// Reload and enable.
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := exec.Command("systemctl", "--user", "enable", "--now", "paperflow").Run(); err != nil {
		return fmt.Errorf("enable: %w", err)
	}

	fmt.Println("Service installed and started.")
	fmt.Println("  systemctl --user status paperflow")
	return nil
}

func uninstallSystemd() error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", "paperflow").Run()

	servicePath := systemdServicePath()
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing service file: %w", err)
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	fmt.Println("Service uninstalled.")
	return nil
}

// launchd

const launchdLabel = "com.alcxyz.paperflow"

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

func generateLaunchdPlist(exePath string, extraFlags []string) string {
	args := []string{exePath, "watch"}
	args = append(args, extraFlags...)

	var argLines strings.Builder
	for _, a := range args {
		fmt.Fprintf(&argLines, "        <string>%s</string>\n", xmlEscape(a))
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
%s    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/paperflow.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/paperflow.err</string>
</dict>
</plist>
`, launchdLabel, argLines.String())
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func installLaunchd(exePath string, extraFlags []string) error {
	plistPath := launchdPlistPath()

	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}

	// Unload existing service if present.
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	plist := generateLaunchdPlist(exePath, extraFlags)
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}
	fmt.Printf("Wrote %s\n", plistPath)

	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	fmt.Println("Service installed and started.")
	fmt.Println("  launchctl list com.alcxyz.paperflow")
	return nil
}

func uninstallLaunchd() error {
	plistPath := launchdPlistPath()
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}

	fmt.Println("Service uninstalled.")
	return nil
}
