// Package notify shows desktop notifications.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Send shows a desktop notification. It returns an error when the platform
// has no supported notifier.
func Send(title, body string) error {
	name, args := command(runtime.GOOS, title, body)
	if name == "" {
		return fmt.Errorf("desktop notifications are not supported on %s", runtime.GOOS)
	}
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, out)
	}
	return nil
}

// command returns the notifier for goos. The text is passed as arguments, not
// spliced into a script, so session names cannot inject anything.
func command(goos, title, body string) (string, []string) {
	switch goos {
	case "darwin":
		return "osascript", []string{
			"-e", "on run argv",
			"-e", "display notification (item 1 of argv) with title (item 2 of argv)",
			"-e", "end run",
			body, title,
		}
	case "linux":
		return "notify-send", []string{title, body}
	}
	return "", nil
}
