package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage twine configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter config at ~/.config/twine/config.toml",
	Args:  cobra.NoArgs,
	RunE:  runConfigInit,
}

func init() {
	configCmd.AddCommand(configInitCmd)
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ".config", "twine")
	path := filepath.Join(dir, "config.toml")

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Config already exists: %s\n", path)
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(config.DefaultConfig), 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Created: %s\n", path)
	fmt.Println("Edit it to set your base_dirs, then run `twine worktree`.")
	return nil
}