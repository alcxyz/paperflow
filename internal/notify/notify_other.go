//go:build !linux && !darwin

package notify

import (
	"fmt"
	"runtime"
)

// sendNotification reports that desktop notifications are unsupported here.
func sendNotification(appName, title, body string) error {
	return fmt.Errorf("desktop notifications are not supported on %s", runtime.GOOS)
}
