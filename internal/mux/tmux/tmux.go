// Package tmux is the tmux backend for twine.
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

// Backend drives tmux through its CLI.
type Backend struct {
	run runFunc
}

// runFunc runs tmux with args and returns its combined output.
type runFunc func(args ...string) (string, error)

var (
	_ mux.Multiplexer   = (*Backend)(nil)
	_ mux.PaneInspector = (*Backend)(nil)
)

// New returns a tmux backend.
func New() *Backend {
	return &Backend{run: execTmux}
}

func execTmux(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Name implements mux.Multiplexer.
func (b *Backend) Name() string { return "tmux" }

// Available implements mux.Multiplexer.
func (b *Backend) Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

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
	if opts.Layout == nil || len(opts.Layout.Windows) == 0 {
		out, err := b.run("new-session", "-d", "-s", name, "-c", dir)
		if err != nil {
			return newSessionError(err, out)
		}
		return nil
	}
	if err := b.buildLayout(name, dir, *opts.Layout); err != nil {
		// Do not leave a half-built session behind.
		_, _ = b.run("kill-session", "-t", name)
		return err
	}
	return nil
}

func newSessionError(err error, output string) error {
	if strings.Contains(output, "duplicate session") {
		return fmt.Errorf("%w: %s", mux.ErrSessionExists, output)
	}
	if output != "" {
		return fmt.Errorf("%w: %s", err, output)
	}
	return err
}

// buildLayout creates the session with its first window, then adds the
// remaining panes and windows, and leaves the first pane of the first window
// selected.
func (b *Backend) buildLayout(name, dir string, l mux.Layout) error {
	var firstPanes []string
	for i, w := range l.Windows {
		args := []string{"-c", dir, "-P", "-F", "#{pane_id}"}
		if w.Name != "" {
			args = append(args, "-n", w.Name)
		}
		if i == 0 {
			args = append([]string{"new-session", "-d", "-s", name}, args...)
		} else {
			args = append([]string{"new-window", "-t", name + ":"}, args...)
		}
		first, err := b.run(args...)
		if err != nil {
			if i == 0 {
				return newSessionError(err, first)
			}
			return fmt.Errorf("window %q: %w: %s", w.Name, err, first)
		}
		if err := b.fillWindow(first, dir, w); err != nil {
			return err
		}
		firstPanes = append(firstPanes, first)
	}
	// Each window ends up with its last split pane active, so reselect the
	// first one, then show the first window.
	for _, pane := range firstPanes {
		if out, err := b.run("select-pane", "-t", pane); err != nil {
			return fmt.Errorf("selecting pane: %w: %s", err, out)
		}
	}
	if out, err := b.run("select-window", "-t", firstPanes[0]); err != nil {
		return fmt.Errorf("selecting window: %w: %s", err, out)
	}
	return nil
}

// fillWindow splits first into the window's remaining panes and types the
// commands.
func (b *Backend) fillWindow(first, dir string, w mux.LayoutWindow) error {
	prev := first
	for i, p := range w.Panes {
		pane := first
		if i > 0 {
			args := []string{
				"split-window", "-t", prev, "-c", dir, "-P", "-F", "#{pane_id}",
				"-l", fmt.Sprintf("%d%%", p.SplitSize()),
			}
			args = append(args, splitFlags(p.SplitDirection())...)
			out, err := b.run(args...)
			if err != nil {
				return fmt.Errorf("splitting pane: %w: %s", err, out)
			}
			pane = out
		}
		if p.Command != "" {
			// -l types the text as is, so words like "Enter" stay text.
			out, err := b.run("send-keys", "-t", pane, "-l", p.Command)
			if err == nil {
				out, err = b.run("send-keys", "-t", pane, "Enter")
			}
			if err != nil {
				return fmt.Errorf("starting %q: %w: %s", p.Command, err, out)
			}
		}
		prev = pane
	}
	return nil
}

// splitFlags returns split-window's flags for placing the new pane. -b puts
// it before (left of or above) the target.
func splitFlags(direction string) []string {
	switch direction {
	case mux.SplitLeft:
		return []string{"-h", "-b"}
	case mux.SplitAbove:
		return []string{"-v", "-b"}
	case mux.SplitBelow:
		return []string{"-v"}
	default:
		return []string{"-h"}
	}
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
