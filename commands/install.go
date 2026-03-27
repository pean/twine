package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/ui"
)

const aliasBlock = `# Twine aliases
alias tw='twine worktree'
alias ts='twine session'
alias tk='twine kill'`

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Set up config, aliases, and completions",
	Long: `Interactively set up Twine: config file, shell aliases, and tab completions.

Shell is detected from $SHELL. Aliases are appended to your shell config
file; completions are written to the appropriate completions directory.`,
	RunE: runInstall,
}

func runInstall(_ *cobra.Command, _ []string) error {
	shell := detectShell()

	items := []ui.Item{
		{Title: "Config   (~/.config/twine/config.toml)", Value: "config"},
		{Title: "Aliases  (tw, ts, tk)", Value: "aliases"},
		{Title: "Completions  (" + shell + ")", Value: "completions"},
	}
	chosen, err := ui.MultiSelect(items, "Select what to install:")
	if errors.Is(err, ui.ErrCancelled) || len(chosen) == 0 {
		fmt.Println("Nothing installed.")
		return nil
	}
	if err != nil {
		return err
	}

	for _, item := range chosen {
		switch item.Value {
		case "config":
			if err := installConfig(); err != nil {
				return err
			}
		case "aliases":
			if err := installAliases(shell); err != nil {
				return err
			}
		case "completions":
			if err := installCompletions(shell); err != nil {
				return err
			}
		}
	}
	return nil
}

func installConfig() error {
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
	fmt.Printf("✓ Config written to %s\n", path)
	fmt.Println("  Edit it to set your base_dirs.")
	return nil
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(shell, "fish"):
		return "fish"
	case strings.HasSuffix(shell, "zsh"):
		return "zsh"
	default:
		return "bash"
	}
}

func installAliases(shell string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cfgPath := shellConfigFile(home, shell)

	existing, _ := os.ReadFile(cfgPath)
	if strings.Contains(string(existing), "# Twine aliases") {
		fmt.Printf("Aliases already present in %s\n", cfgPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("could not create config dir: %w", err)
	}
	f, err := os.OpenFile(cfgPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", cfgPath, err)
	}
	defer f.Close()

	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		_, _ = f.WriteString("\n")
	}
	if _, err := fmt.Fprintf(f, "\n%s\n", aliasBlock); err != nil {
		return fmt.Errorf("failed to write aliases: %w", err)
	}
	fmt.Printf("✓ Aliases appended to %s\n", cfgPath)
	return nil
}

func installCompletions(shell string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	destDir, destFile, note := shellCompletionPath(home, shell)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("could not create completions dir: %w", err)
	}

	var buf bytes.Buffer
	switch shell {
	case "fish":
		if err := rootCmd.GenFishCompletion(&buf, true); err != nil {
			return fmt.Errorf("failed to generate completions: %w", err)
		}
	case "zsh":
		if err := rootCmd.GenZshCompletion(&buf); err != nil {
			return fmt.Errorf("failed to generate completions: %w", err)
		}
	default:
		if err := rootCmd.GenBashCompletion(&buf); err != nil {
			return fmt.Errorf("failed to generate completions: %w", err)
		}
	}

	path := filepath.Join(destDir, destFile)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write completions: %w", err)
	}
	fmt.Printf("✓ Completions written to %s\n", path)
	if note != "" {
		fmt.Println(" ", note)
	}
	return nil
}

func shellConfigFile(home, shell string) string {
	switch shell {
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	case "zsh":
		return filepath.Join(home, ".zshrc")
	default:
		return filepath.Join(home, ".bashrc")
	}
}

func shellCompletionPath(home, shell string) (dir, file, note string) {
	switch shell {
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions"),
			"twine.fish", ""
	case "zsh":
		return filepath.Join(home, ".zfunc"),
			"_twine",
			"Ensure ~/.zfunc is in your fpath: fpath=(~/.zfunc $fpath)"
	default:
		return filepath.Join(home, ".local", "share", "bash-completion", "completions"),
			"twine", ""
	}
}
