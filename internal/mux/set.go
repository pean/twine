package mux

import (
	"fmt"
	"strings"
)

// Auto is the Choice name that picks a backend from the environment.
const Auto = "auto"

// Choice is a preference for which backend to use.
type Choice struct {
	// Name is Auto or a backend name.
	Name string
	// Strict restricts twine to this backend, also when looking for
	// existing sessions. Otherwise the choice only applies to new sessions
	// and existing sessions are found in any available backend.
	Strict bool
	// Source says where the choice came from, for error messages, such as
	// "--mux" or "config".
	Source string
}

// Chooser returns the backend preference for a repository. repo is empty
// when no repository is involved.
type Chooser func(repo string) Choice

// Set is every backend twine knows about plus the rules for picking one.
type Set struct {
	backends []Multiplexer
	choose   Chooser
}

// NewSet returns a Set over backends. Their order is the priority Auto
// uses when twine is not running inside any of them.
func NewSet(backends []Multiplexer, choose Chooser) *Set {
	if choose == nil {
		choose = func(string) Choice { return Choice{Name: Auto} }
	}
	return &Set{backends: backends, choose: choose}
}

// For returns the backend that new sessions for repo are created in.
func (s *Set) For(repo string) (Multiplexer, error) {
	c := s.choose(repo)
	if c.Name == "" || c.Name == Auto {
		return s.auto()
	}
	b := s.byName(c.Name)
	if b == nil {
		return nil, fmt.Errorf(
			"multiplexer %q%s is not supported yet (supported: %s)",
			c.Name, sourceSuffix(c), strings.Join(s.names(), ", "),
		)
	}
	if !b.Available() {
		return nil, fmt.Errorf(
			"multiplexer %q%s is not installed", c.Name, sourceSuffix(c),
		)
	}
	return b, nil
}

func (s *Set) auto() (Multiplexer, error) {
	for _, b := range s.backends {
		if b.Inside() && b.Available() {
			return b, nil
		}
	}
	for _, b := range s.backends {
		if b.Available() {
			return b, nil
		}
	}
	return nil, fmt.Errorf(
		"no terminal multiplexer found (tried: %s)",
		strings.Join(s.names(), ", "),
	)
}

// Backends returns the backends searched for existing sessions of repo, the
// one For(repo) picks first. With a strict choice it is only that one.
func (s *Set) Backends(repo string) []Multiplexer {
	preferred, err := s.For(repo)
	if s.choose(repo).Strict {
		if err != nil {
			return nil
		}
		return []Multiplexer{preferred}
	}
	var result []Multiplexer
	if err == nil {
		result = append(result, preferred)
	}
	for _, b := range s.backends {
		if b != preferred && b.Available() {
			result = append(result, b)
		}
	}
	return result
}

// Find returns the backend holding the session name, searching in the order
// of Backends(repo).
func (s *Set) Find(repo, name string) (Multiplexer, bool) {
	for _, b := range s.Backends(repo) {
		if b.HasSession(name) {
			return b, true
		}
	}
	return nil, false
}

// Owners returns every searched backend holding a session called name.
func (s *Set) Owners(name string) []Multiplexer {
	var result []Multiplexer
	for _, b := range s.Backends("") {
		if b.HasSession(name) {
			result = append(result, b)
		}
	}
	return result
}

// OwnedSession is a session together with the backend running it.
type OwnedSession struct {
	Session
	Backend Multiplexer
}

// Sessions lists the sessions of every searched backend.
func (s *Set) Sessions() []OwnedSession {
	var result []OwnedSession
	for _, b := range s.Backends("") {
		sessions, _ := b.ListSessions()
		for _, sess := range sessions {
			result = append(result, OwnedSession{Session: sess, Backend: b})
		}
	}
	return result
}

// SessionNames lists session names across searched backends without
// duplicates, in the order of Sessions.
func (s *Set) SessionNames() []string {
	seen := map[string]bool{}
	var result []string
	for _, sess := range s.Sessions() {
		if !seen[sess.Name] {
			seen[sess.Name] = true
			result = append(result, sess.Name)
		}
	}
	return result
}

func (s *Set) byName(name string) Multiplexer {
	for _, b := range s.backends {
		if b.Name() == name {
			return b
		}
	}
	return nil
}

func (s *Set) names() []string {
	names := make([]string, 0, len(s.backends))
	for _, b := range s.backends {
		names = append(names, b.Name())
	}
	return names
}

func sourceSuffix(c Choice) string {
	if c.Source == "" {
		return ""
	}
	return " (from " + c.Source + ")"
}
