//go:build linux

package notify

import "os/exec"

// sendNotification sends a desktop notification using notify-send.
func sendNotification(appName, title, body string) error {
	args := []string{"-a", appName, title}
	if body != "" {
		args = append(args, body)
	}
	return exec.Command("notify-send", args...).Run()
}
