//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

// sendNotification sends a desktop notification using osascript.
func sendNotification(appName, title, body string) error {
	script := fmt.Sprintf(`display notification %q with title %q subtitle %q`, body, title, appName)
	return exec.Command("osascript", "-e", script).Run()
}
