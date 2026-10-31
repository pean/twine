// Package mux defines the terminal multiplexer abstraction twine drives.
//
// A backend (tmux today) implements Multiplexer for the session lifecycle
// twine needs. Optional abilities, such as inspecting panes for the agents
// dashboard, are separate interfaces that callers detect with a type
// assertion, so a backend only implements what it can support.
package mux

import "errors"

// ErrSessionExists is returned by NewSession when a session with the same
// name already exists.
var ErrSessionExists = errors.New("session already exists")

// Session describes a multiplexer session.
type Session struct {
	Name    string
	Path    string
	Windows int
}

// Pane describes one terminal inside a session.
type Pane struct {
	PID     int
	Session string
	Path    string
	Command string
	ID      string
}

// NewSessionOptions controls how a session is created.
type NewSessionOptions struct {
	// UseLayout applies the backend's configured layout (for example a
	// tmuxinator project) when one is configured. When false, or when no
	// layout is configured, a plain session with one shell is created.
	UseLayout bool
}

// Multiplexer is the session lifecycle every backend supports.
type Multiplexer interface {
	// Name is the backend's identifier, such as "tmux".
	Name() string
	// Available reports whether the backend can be used on this machine.
	Available() bool
	// Inside reports whether twine runs inside a session of this backend.
	Inside() bool
	// HasSession reports whether a session with exactly this name exists.
	HasSession(name string) bool
	// ListSessions returns every session. No sessions is not an error.
	ListSessions() ([]Session, error)
	// NewSession creates a detached session named name rooted at dir.
	NewSession(name, dir string, opts NewSessionOptions) error
	// KillSession ends the named session.
	KillSession(name string) error
	// AttachOrSwitch brings the named session into view.
	AttachOrSwitch(name string) error
}

// PaneInspector is implemented by backends that can list panes and read
// their screen contents.
type PaneInspector interface {
	ListPanes() ([]Pane, error)
	// CapturePane returns the last lines lines of the pane's content.
	CapturePane(paneID string, lines int) (string, error)
}

// SessionNames returns the names of sessions, preserving order.
func SessionNames(sessions []Session) []string {
	names := make([]string, 0, len(sessions))
	for _, s := range sessions {
		names = append(names, s.Name)
	}
	return names
}
