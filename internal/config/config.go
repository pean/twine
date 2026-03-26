package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigPath is the canonical location of the config file.
const ConfigPath = ".config/twine/config.toml"

// Config holds all Twine settings.
type Config struct {
	BaseDirs         []string `toml:"base_dirs"`
	SessionPrefix    string   `toml:"session_prefix"`
	UseTmuxinator    string   `toml:"use_tmuxinator"`
	TmuxinatorLayout string   `toml:"tmuxinator_layout"`
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

	return &cfg, nil
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

# Prefix prepended to all tmux session names (optional).
# session_prefix = ""

# Tmuxinator integration (optional).
# use_tmuxinator = "auto"  # auto | true | false
# tmuxinator_layout = "dev"
`