package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/pean/twine/internal/mux"
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
	BaseDirs      []string              `toml:"base_dirs"`
	SessionPrefix string                `toml:"session_prefix"`
	Multiplexer   string                `toml:"multiplexer"`
	Layout        string                `toml:"layout"`
	Layouts       map[string]mux.Layout `toml:"layouts"`
	Repos         []RepoConfig          `toml:"repos"`
}

// RepoConfig overrides settings for repositories whose name matches Match.
type RepoConfig struct {
	// Match is a repository name or a glob such as "work-*".
	Match       string `toml:"match"`
	Multiplexer string `toml:"multiplexer"`
	Layout      string `toml:"layout"`
}

// legacyKeys are settings from the removed tmuxinator integration.
var legacyKeys = []string{"use_tmuxinator", "tmuxinator_layout"}

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
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	for _, key := range legacyKeys {
		if md.IsDefined(key) {
			return nil, fmt.Errorf(
				"%s: %s is no longer supported, tmuxinator was replaced by "+
					"layouts\nDefine a layout under [layouts.<name>] and set "+
					"layout = \"<name>\" (see the README)",
				path, key,
			)
		}
	}

	// Expand ~ in base_dirs entries.
	for i, dir := range cfg.BaseDirs {
		if strings.HasPrefix(dir, "~/") {
			cfg.BaseDirs[i] = filepath.Join(home, dir[2:])
		}
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
		if err := c.checkLayoutName(r.Layout); err != nil {
			return fmt.Errorf("repos[%d] (%s): layout: %w", i, r.Match, err)
		}
	}
	if err := c.checkLayoutName(c.Layout); err != nil {
		return fmt.Errorf("layout: %w", err)
	}
	for name, l := range c.Layouts {
		if err := l.Validate(); err != nil {
			return fmt.Errorf("layouts.%s: %w", name, err)
		}
	}
	return nil
}

func (c *Config) checkLayoutName(name string) error {
	if name == "" {
		return nil
	}
	if _, ok := c.Layouts[name]; !ok {
		return fmt.Errorf("no layout named %q under [layouts]", name)
	}
	return nil
}

// LayoutFor returns the layout new sessions of repo start with: the first
// [[repos]] entry that matches and sets one, otherwise the top-level setting.
// It is nil when none is configured.
func (c *Config) LayoutFor(repo string) *mux.Layout {
	name := c.Layout
	if repo != "" {
		for _, r := range c.Repos {
			if r.Layout == "" {
				continue
			}
			if ok, _ := path.Match(r.Match, repo); ok {
				name = r.Layout
				break
			}
		}
	}
	if name == "" {
		return nil
	}
	l := c.Layouts[name]
	return &l
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
# layout = "dev"

# Layout new sessions start with (optional), defined under [layouts].
# Each pane after the first splits the previous one: split is left, right,
# above, or below (default right), size is its percentage (default 50).
# layout = "dev"
#
# [layouts.dev]
# windows = [
#   { name = "code", panes = [
#       { command = "nvim" },
#       { split = "below", size = 25 },
#   ]},
# ]
`
