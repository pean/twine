// Package rex is the Rex backend for twine. It drives the rex CLI.
package rex

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pean/twine/internal/mux"
)

// readOnly keeps queries from starting a Rex server as a side effect.
const readOnly = "--autostart=false"

type runFunc func(args ...string) ([]byte, error)

// Backend drives Rex through its CLI.
type Backend struct {
	run runFunc
}

var (
	_ mux.Multiplexer   = (*Backend)(nil)
	_ mux.PaneInspector = (*Backend)(nil)
)

// panesScript lists every terminal in one connection to the server.
//
//go:embed panes.lua
var panesScript string

// New returns a Rex backend.
func New() *Backend {
	return &Backend{run: execRex}
}

func execRex(args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("rex", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("%w: %s", err, msg)
		}
	}
	return out, err
}

// Name implements mux.Multiplexer.
func (b *Backend) Name() string { return "rex" }

// Available implements mux.Multiplexer.
func (b *Backend) Available() bool {
	_, err := exec.LookPath("rex")
	return err == nil
}

// Inside implements mux.Multiplexer.
func (b *Backend) Inside() bool { return os.Getenv("REX_SESSION") != "" }

type sessionRef struct {
	ID    string `json:"session_id"`
	Label string `json:"label"`
}

func (b *Backend) sessions() ([]sessionRef, error) {
	out, err := b.run(readOnly, "ls", "--json")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Sessions []sessionRef `json:"sessions"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing rex ls: %w", err)
	}
	return resp.Sessions, nil
}

// HasSession implements mux.Multiplexer.
func (b *Backend) HasSession(name string) bool {
	refs, _ := b.sessions()
	for _, r := range refs {
		if r.Label == name {
			return true
		}
	}
	return false
}

// ListSessions implements mux.Multiplexer. Rex lists only IDs and labels, so
// Path and Windows are left empty.
func (b *Backend) ListSessions() ([]mux.Session, error) {
	refs, _ := b.sessions()
	result := make([]mux.Session, 0, len(refs))
	for _, r := range refs {
		result = append(result, mux.Session{Name: r.Label})
	}
	return result, nil
}

// NewSession implements mux.Multiplexer. Rex allows duplicate labels, so the
// name is checked first.
func (b *Backend) NewSession(
	name, dir string,
	_ mux.NewSessionOptions,
) error {
	if b.HasSession(name) {
		return fmt.Errorf("%w: %s", mux.ErrSessionExists, name)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	_, err = b.run("new", name, "--cwd", abs)
	return err
}

// resolve returns the ID of the one session labelled name. Targeting by ID
// avoids rex matching labels case-insensitively or by position.
func (b *Backend) resolve(name string) (string, error) {
	refs, err := b.sessions()
	if err != nil {
		return "", err
	}
	var ids []string
	for _, r := range refs {
		if r.Label == name {
			ids = append(ids, r.ID)
		}
	}
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("no rex session named %q", name)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("%d rex sessions are named %q", len(ids), name)
	}
}

// KillSession implements mux.Multiplexer.
func (b *Backend) KillSession(name string) error {
	id, err := b.resolve(name)
	if err != nil {
		return err
	}
	_, err = b.run("kill", id)
	return err
}

// AttachOrSwitch implements mux.Multiplexer. Rex has no terminal attach, so
// it asks the connected Rex app to show the session. The app only accepts
// this when Remote Control is turned on in its settings.
func (b *Backend) AttachOrSwitch(name string) error {
	id, err := b.resolve(name)
	if err != nil {
		return err
	}
	app, err := b.appClient()
	if err != nil {
		return fmt.Errorf("session %q is running: %w", name, err)
	}
	req, err := json.Marshal(map[string]any{
		"client_id": app,
		"action":    "session.select",
		"args":      map[string]string{"session_id": id},
		"origin":    "cli",
	})
	if err != nil {
		return err
	}
	if _, err := b.run("api", "call", "client.perform", string(req)); err != nil {
		return fmt.Errorf("switching rex to %q: %w", name, err)
	}
	return nil
}

func (b *Backend) appClient() (string, error) {
	out, err := b.run(readOnly, "client", "ls", "--json")
	if err != nil {
		return "", err
	}
	var resp struct {
		Clients []struct {
			ID    string `json:"client_id"`
			State string `json:"connection_state"`
			Info  struct {
				Kind string `json:"kind"`
			} `json:"info"`
		} `json:"clients"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("parsing rex client ls: %w", err)
	}
	for _, c := range resp.Clients {
		if c.Info.Kind == "app" && c.State == "connected" {
			return c.ID, nil
		}
	}
	return "", fmt.Errorf("no Rex app is connected; open Rex to view it")
}

type paneJSON struct {
	Session   string `json:"session"`
	SessionID string `json:"session_id"`
	Block     string `json:"block"`
	PID       int    `json:"pid"`
	Name      string `json:"name"`
	Cwd       string `json:"cwd"`
	Status    *struct {
		State string `json:"state"`
		App   string `json:"app"`
	} `json:"status"`
}

// ListPanes implements mux.PaneInspector. Each pane's PID is its foreground
// process, so an agent started there matches without walking its ancestors.
func (b *Backend) ListPanes() ([]mux.Pane, error) {
	out, err := b.run(readOnly, "do", "-e", panesScript)
	if err != nil {
		return nil, err
	}
	// Lua encodes an empty list as an object.
	if strings.TrimSpace(string(out)) == "{}" {
		return nil, nil
	}
	var raw []paneJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing rex panes: %w", err)
	}
	panes := make([]mux.Pane, 0, len(raw))
	for _, r := range raw {
		p := mux.Pane{
			PID:     r.PID,
			Session: r.Session,
			Path:    r.Cwd,
			Command: strings.TrimPrefix(filepath.Base(r.Name), "-"),
			ID:      r.SessionID + "/" + r.Block,
		}
		if r.Status != nil {
			p.Status = &mux.PaneStatus{State: r.Status.State, App: r.Status.App}
		}
		panes = append(panes, p)
	}
	return panes, nil
}

// CapturePane implements mux.PaneInspector. Rex has no way to ask for only
// the last lines, so it reads the whole screen and scrollback.
func (b *Backend) CapturePane(paneID string, lines int) (string, error) {
	sessionID, blockID, ok := strings.Cut(paneID, "/")
	if !ok {
		return "", fmt.Errorf("bad rex pane id %q", paneID)
	}
	out, err := b.run("capture", "-s", sessionID, "-b", blockID, "--trim")
	if err != nil {
		return "", err
	}
	all := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n"), nil
}
