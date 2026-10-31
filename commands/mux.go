package commands

import (
	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/mux"
	"github.com/pean/twine/internal/mux/tmux"
)

// multiplexer returns the backend twine drives for cfg. cfg may be nil, in
// which case defaults apply.
func multiplexer(cfg *config.Config) mux.Multiplexer {
	opts := tmux.Options{}
	if cfg != nil {
		opts.UseTmuxinator = cfg.ShouldUseTmuxinator()
		opts.TmuxinatorLayout = cfg.TmuxinatorLayout
	}
	return tmux.New(opts)
}

// sessionNames lists session names on m, ignoring errors as callers did
// before the multiplexer abstraction.
func sessionNames(m mux.Multiplexer) []string {
	sessions, _ := m.ListSessions()
	return mux.SessionNames(sessions)
}
