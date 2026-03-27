package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// SessionInfo holds basic information about a tmux session.
type SessionInfo struct {
	Name    string
	Path    string
	Windows int
}

// PaneInfo holds information about a tmux pane.
type PaneInfo struct {
	PID     int
	Session string
	Path    string
	Command string
	PaneID  string
}

// IsInsideTmux returns true when the process is running inside a tmux session.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// HasSession returns true if the named tmux session exists.
func HasSession(name string) bool {
	err := exec.Command("tmux", "has-session", "-t", name).Run()
	return err == nil
}

// ListSessions returns all session names.
func ListSessions() ([]string, error) {
	out, err := exec.Command(
		"tmux", "list-sessions", "-F", "#{session_name}",
	).Output()
	if err != nil {
		// No sessions is not a fatal error.
		return nil, nil
	}
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// ListSessionsFull returns sessions with path and window count info.
func ListSessionsFull() ([]SessionInfo, error) {
	out, err := exec.Command(
		"tmux",
		"list-sessions",
		"-F",
		"#{session_name}|#{session_path}|#{session_windows}",
	).Output()
	if err != nil {
		return nil, nil
	}
	var result []SessionInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		windows, _ := strconv.Atoi(parts[2])
		result = append(result, SessionInfo{
			Name:    parts[0],
			Path:    parts[1],
			Windows: windows,
		})
	}
	return result, nil
}

// NewSession creates a detached tmux session named name in dir.
func NewSession(name, dir string) error {
	return exec.Command(
		"tmux", "new-session", "-d", "-s", name, "-c", dir,
	).Run()
}

// KillSession kills the named tmux session.
func KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// AttachOrSwitch attaches (outside tmux) or switches (inside tmux) to name.
// When attaching, uses syscall.Exec to replace the current process.
func AttachOrSwitch(name string) error {
	if IsInsideTmux() {
		return exec.Command("tmux", "switch-client", "-t", name).Run()
	}
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return syscall.Exec(
		tmuxPath,
		[]string{"tmux", "attach", "-t", name},
		os.Environ(),
	)
}

// ListPanes returns pane info for all panes across all sessions.
func ListPanes() ([]PaneInfo, error) {
	out, err := exec.Command(
		"tmux",
		"list-panes",
		"-a",
		"-F",
		"#{pane_pid}|#{session_name}|#{pane_current_path}"+
			"|#{pane_current_command}|#{pane_id}",
	).Output()
	if err != nil {
		return nil, err
	}
	var result []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 {
			continue
		}
		pid, _ := strconv.Atoi(parts[0])
		result = append(result, PaneInfo{
			PID:     pid,
			Session: parts[1],
			Path:    parts[2],
			Command: parts[3],
			PaneID:  parts[4],
		})
	}
	return result, nil
}

// CapturePaneContent captures the last lines lines of a pane's content.
func CapturePaneContent(paneID string, lines int) (string, error) {
	start := fmt.Sprintf("-%d", lines)
	out, err := exec.Command(
		"tmux", "capture-pane", "-p", "-t", paneID, "-S", start,
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}