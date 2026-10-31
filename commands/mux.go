package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/mux"
	"github.com/pean/twine/internal/mux/rex"
	"github.com/pean/twine/internal/mux/tmux"
)

// muxEnv sets the default multiplexer for one shell, like the config
// setting but taking precedence over it.
const muxEnv = "TWINE_MUX"

// muxFlag is the persistent --mux flag. It is strict: only that backend is
// used, including when looking for existing sessions.
var muxFlag string

// backends returns every backend twine supports, in the priority order
// "auto" uses when twine runs inside none of them.
func backends(cfg *config.Config) []mux.Multiplexer {
	opts := tmux.Options{}
	if cfg != nil {
		opts.UseTmuxinator = cfg.ShouldUseTmuxinator()
		opts.TmuxinatorLayout = cfg.TmuxinatorLayout
	}
	return []mux.Multiplexer{tmux.New(opts), rex.New()}
}

// muxSet returns the backends for cfg with the selection rules applied:
// --mux, then $TWINE_MUX, then the repo's config, then the top-level config.
// cfg may be nil, in which case "auto" applies.
func muxSet(cfg *config.Config) (*mux.Set, error) {
	if muxFlag != "" {
		if err := config.ValidateMultiplexer(muxFlag); err != nil {
			return nil, fmt.Errorf("--mux: %w", err)
		}
	}
	env := os.Getenv(muxEnv)
	if env != "" {
		if err := config.ValidateMultiplexer(env); err != nil {
			return nil, fmt.Errorf("%s: %w", muxEnv, err)
		}
	}

	choose := func(repo string) mux.Choice {
		switch {
		case muxFlag != "":
			return mux.Choice{Name: muxFlag, Strict: true, Source: "--mux"}
		case env != "":
			return mux.Choice{Name: env, Source: muxEnv}
		case cfg != nil:
			return mux.Choice{Name: cfg.MultiplexerFor(repo), Source: "config"}
		default:
			return mux.Choice{Name: mux.Auto}
		}
	}
	return mux.NewSet(backends(cfg), choose), nil
}

// ensureSession returns the backend holding the session name, creating it in
// the backend chosen for repo when no backend has it.
func ensureSession(
	set *mux.Set,
	repo, name, dir string,
	opts mux.NewSessionOptions,
) (mux.Multiplexer, error) {
	if m, ok := set.Find(repo, name); ok {
		return m, nil
	}
	m, err := set.For(repo)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Creating %s session: %s\n", m.Name(), name)
	err = m.NewSession(name, dir, opts)
	if err != nil && !errors.Is(err, mux.ErrSessionExists) {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return m, nil
}

// muxLabel returns " [name]" when sessions may come from several backends,
// so lists can show where each session runs.
func muxLabel(set *mux.Set, m mux.Multiplexer) string {
	if len(set.Backends("")) < 2 {
		return ""
	}
	return " [" + m.Name() + "]"
}
