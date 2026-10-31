package config

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigPath is the canonical location of the config file.
const ConfigPath = ".config/twine/config.toml"

// MultiplexerAuto picks a backend from the environment: the one twine runs
// inside, otherwise the first one installed.
const MultiplexerAuto = "auto"

// Multiplexers lists the accepted values of the multiplexer settings.
var Multiplexers = []string{MultiplexerAuto, "tmux", "rex"}

// Config holds all Twine settings.
type Config struct {
	BaseDirs         []string     `toml:"base_dirs"`
	SessionPrefix    string       `toml:"session_prefix"`
	Multiplexer      string       `toml:"multiplexer"`
	UseTmuxinator    string       `toml:"use_tmuxinator"`
	TmuxinatorLayout string       `toml:"tmuxinator_layout"`
	Repos            []RepoConfig `toml:"repos"`
}

// RepoConfig overrides settings for repositories whose name matches Match.
type RepoConfig struct {
	// Match is a repository name or a glob such as "work-*".
	Match       string `toml:"match"`
	Multiplexer string `toml:"multiplexer"`
}

// Load reads ~/.config/twine/config.toml and returns the parsed config.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(home, ConfigPath)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"no config file found at %s\n"+
				"Run `twine config init` to create one",
			path,
		)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Expand ~ in base_dirs entries.
	for i, dir := range cfg.BaseDirs {
		if strings.HasPrefix(dir, "~/") {
			cfg.BaseDirs[i] = filepath.Join(home, dir[2:])
		}
	}

	if cfg.UseTmuxinator == "" {
		cfg.UseTmuxinator = "auto"
	}
	if cfg.Multiplexer == "" {
		cfg.Multiplexer = MultiplexerAuto
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if err := ValidateMultiplexer(c.Multiplexer); err != nil {
		return fmt.Errorf("multiplexer: %w", err)
	}
	for i, r := range c.Repos {
		if r.Match == "" {
			return fmt.Errorf("repos[%d]: match is required", i)
		}
		if _, err := path.Match(r.Match, ""); err != nil {
			return fmt.Errorf("repos[%d]: bad match %q: %w", i, r.Match, err)
		}
		if r.Multiplexer != "" {
			if err := ValidateMultiplexer(r.Multiplexer); err != nil {
				return fmt.Errorf("repos[%d] (%s): multiplexer: %w", i, r.Match, err)
			}
		}
	}
	return nil
}

// ValidateMultiplexer returns an error unless name is an accepted
// multiplexer setting.
func ValidateMultiplexer(name string) error {
	for _, m := range Multiplexers {
		if name == m {
			return nil
		}
	}
	return fmt.Errorf(
		"unknown value %q (want one of: %s)",
		name, strings.Join(Multiplexers, ", "),
	)
}

// MultiplexerFor returns the configured multiplexer for repo: the first
// [[repos]] entry that matches and sets one, otherwise the top-level
// setting. The result may be "auto".
func (c *Config) MultiplexerFor(repo string) string {
	if repo != "" {
		for _, r := range c.Repos {
			if r.Multiplexer == "" {
				continue
			}
			if ok, _ := path.Match(r.Match, repo); ok {
				return r.Multiplexer
			}
		}
	}
	if c.Multiplexer == "" {
		return MultiplexerAuto
	}
	return c.Multiplexer
}

// ShouldUseTmuxinator returns true when tmuxinator should be used.
func (c *Config) ShouldUseTmuxinator() bool {
	switch c.UseTmuxinator {
	case "true":
		_, err := exec.LookPath("tmuxinator")
		return err == nil
	case "false":
		return false
	default: // "auto"
		_, err := exec.LookPath("tmuxinator")
		return err == nil && c.TmuxinatorLayout != ""
	}
}

// DefaultConfig is the starter config written by `twine config init`.
const DefaultConfig = `# Twine configuration
# ~/.config/twine/config.toml

# Directories to search for repositories (required).
# Entries support ~ for home directory.
base_dirs = [
  "~/src",
]

# Prefix prepended to all session names (optional).
# session_prefix = ""

# Terminal multiplexer for new sessions (optional): auto | tmux | rex.
# "auto" uses the one twine runs inside, otherwise the first installed.
# Override per command with --mux, or per shell with TWINE_MUX.
# multiplexer = "auto"

# Per-repository overrides (optional). The first matching entry wins.
# [[repos]]
# match = "work-*"
# multiplexer = "rex"

# Tmuxinator integration (optional).
# use_tmuxinator = "auto"  # auto | true | false
# tmuxinator_layout = "dev"
`