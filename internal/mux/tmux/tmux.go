// Package tmux is the tmux backend for twine, with optional tmuxinator
// layouts.
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/pean/twine/internal/mux"
)

// Options configures the tmux backend.
type Options struct {
	// UseTmuxinator creates sessions through tmuxinator when the caller
	// asks for a layout.
	UseTmuxinator bool
	// TmuxinatorLayout is the tmuxinator project to start. Defaults to "dev".
	TmuxinatorLayout string
}

// Backend drives tmux through its CLI.
type Backend struct {
	opts Options
}

var (
	_ mux.Multiplexer   = (*Backend)(nil)
	_ mux.PaneInspector = (*Backend)(nil)
)

// New returns a tmux backend.
func New(opts Options) *Backend {
	return &Backend{opts: opts}
}

// Name implements mux.Multiplexer.
func (b *Backend) Name() string { return "tmux" }

// Inside implements mux.Multiplexer.
func (b *Backend) Inside() bool { return os.Getenv("TMUX") != "" }

// HasSession implements mux.Multiplexer.
func (b *Backend) HasSession(name string) bool {
	sessions, _ := b.ListSessions()
	for _, s := range sessions {
		if s.Name == name {
			return true
		}
	}
	return false
}

const sessionFormat = "#{session_name}|#{session_path}|#{session_windows}"

// ListSessions implements mux.Multiplexer.
func (b *Backend) ListSessions() ([]mux.Session, error) {
	out, err := exec.Command(
		"tmux", "list-sessions", "-F", sessionFormat,
	).Output()
	if err != nil {
		// No server or no sessions is not a fatal error.
		return nil, nil
	}
	return parseSessions(string(out)), nil
}

func parseSessions(out string) []mux.Session {
	var result []mux.Session
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || parts[0] == "" {
			continue
		}
		windows, _ := strconv.Atoi(parts[2])
		result = append(result, mux.Session{
			Name:    parts[0],
			Path:    parts[1],
			Windows: windows,
		})
	}
	return result
}

// NewSession implements mux.Multiplexer.
func (b *Backend) NewSession(
	name, dir string,
	opts mux.NewSessionOptions,
) error {
	if opts.UseLayout && b.opts.UseTmuxinator {
		return b.startTmuxinator(name, dir)
	}
	out, err := exec.Command(
		"tmux", "new-session", "-d", "-s", name, "-c", dir,
	).CombinedOutput()
	if err != nil {
		return newSessionError(err, string(out))
	}
	return nil
}

func newSessionError(err error, output string) error {
	msg := strings.TrimSpace(output)
	if strings.Contains(msg, "duplicate session") {
		return fmt.Errorf("%w: %s", mux.ErrSessionExists, msg)
	}
	if msg != "" {
		return fmt.Errorf("%w: %s", err, msg)
	}
	return err
}

func (b *Backend) startTmuxinator(name, dir string) error {
	layout := b.opts.TmuxinatorLayout
	if layout == "" {
		layout = "dev"
	}
	c := exec.Command("tmuxinator", "start", "-n", name, layout, "false")
	c.Dir = dir
	return c.Run()
}

// KillSession implements mux.Multiplexer.
func (b *Backend) KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// AttachOrSwitch implements mux.Multiplexer. Inside tmux it switches the
// client; outside it replaces the current process with `tmux attach`.
func (b *Backend) AttachOrSwitch(name string) error {
	if b.Inside() {
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

const paneFormat = "#{pane_pid}|#{session_name}|#{pane_current_path}" +
	"|#{pane_current_command}|#{pane_id}"

// ListPanes implements mux.PaneInspector.
func (b *Backend) ListPanes() ([]mux.Pane, error) {
	out, err := exec.Command(
		"tmux", "list-panes", "-a", "-F", paneFormat,
	).Output()
	if err != nil {
		return nil, err
	}
	return parsePanes(string(out)), nil
}

func parsePanes(out string) []mux.Pane {
	var result []mux.Pane
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 {
			continue
		}
		pid, _ := strconv.Atoi(parts[0])
		result = append(result, mux.Pane{
			PID:     pid,
			Session: parts[1],
			Path:    parts[2],
			Command: parts[3],
			ID:      parts[4],
		})
	}
	return result
}

// CapturePane implements mux.PaneInspector.
func (b *Backend) CapturePane(paneID string, lines int) (string, error) {
	start := fmt.Sprintf("-%d", lines)
	out, err := exec.Command(
		"tmux", "capture-pane", "-p", "-t", paneID, "-S", start,
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
