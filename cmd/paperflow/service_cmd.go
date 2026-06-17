package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func runService(f flags, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: paperflow service [install|uninstall|status]")
	}

	switch args[0] {
	case "install":
		return serviceInstall(f)
	case "uninstall":
		return serviceUninstall()
	case "status":
		return serviceStatus()
	default:
		return fmt.Errorf("unknown service subcommand: %s", args[0])
	}
}

func serviceInstall(f flags) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Build extra flags to pass through to paperflow watch.
	var extraFlags []string
	if f.watchDir != "" {
		extraFlags = append(extraFlags, "--watch", f.watchDir)
	}
	if f.settleDelay != "" {
		extraFlags = append(extraFlags, "--settle-delay", f.settleDelay)
	}
	if f.ingest != "" {
		extraFlags = append(extraFlags, "--ingest", f.ingest)
	}
	if f.ingestDir != "" {
		extraFlags = append(extraFlags, "--ingest-dir", f.ingestDir)
	}
	if f.ingestArchiveDir != "" {
		extraFlags = append(extraFlags, "--ingest-archive-dir", f.ingestArchiveDir)
	}
	if f.ingestArchiveAfter != "" {
		extraFlags = append(extraFlags, "--ingest-archive-after", f.ingestArchiveAfter)
	}
	if f.paperlessURL != "" {
		extraFlags = append(extraFlags, "--paperless-url", f.paperlessURL)
	}
	if f.paperlessTokenFile != "" {
		extraFlags = append(extraFlags, "--paperless-token-file", f.paperlessTokenFile)
	}
	if f.config != "" {
		extraFlags = append(extraFlags, "--config", f.config)
	}
	if f.noNotify {
		extraFlags = append(extraFlags, "--no-notify")
	}

	switch runtime.GOOS {
	case "linux":
		return installSystemd(exePath, extraFlags)
	case "darwin":
		return installLaunchd(exePath, extraFlags)
	default:
		return fmt.Errorf("service install not supported on %s", runtime.GOOS)
	}
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
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "systemd", "user", systemdServiceName)
}

func generateSystemdUnit(exePath string, extraFlags []string) string {
	execStart := exePath + " watch"
	if len(extraFlags) > 0 {
		execStart += " " + strings.Join(extraFlags, " ")
	}

	// Capture current PATH so the service can find tools like notify-send.
	envLine := ""
	if p := os.Getenv("PATH"); p != "" {
		envLine = fmt.Sprintf("Environment=PATH=%s\n", p)
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

	var argLines string
	for _, a := range args {
		argLines += fmt.Sprintf("        <string>%s</string>\n", a)
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
`, launchdLabel, argLines)
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
