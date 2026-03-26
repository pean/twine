package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var installAliasesWrite bool

const aliasBlock = `# Twine aliases
alias tw='twine worktree'
alias t='twine session'
alias tk='twine kill'
alias ts='twine start'
alias agents='twine agents'`

var installAliasesCmd = &cobra.Command{
	Use:   "install-aliases",
	Short: "Print or install fish shell aliases for twine commands",
	Long: `Print the alias block or append it to ~/.config/fish/config.fish.

Without --write, the alias block is printed to stdout so you can review it.
With --write, it is appended to your fish config file.`,
	RunE: runInstallAliases,
}

func init() {
	installAliasesCmd.Flags().BoolVar(
		&installAliasesWrite, "write", false,
		"append aliases to ~/.config/fish/config.fish",
	)
}

func runInstallAliases(cmd *cobra.Command, args []string) error {
	if !installAliasesWrite {
		fmt.Println(aliasBlock)
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	cfgPath := filepath.Join(home, ".config", "fish", "config.fish")

	existing, _ := os.ReadFile(cfgPath)
	if strings.Contains(string(existing), "# Twine aliases") {
		fmt.Println("Twine aliases already present in", cfgPath)
		return nil
	}

	f, err := os.OpenFile(cfgPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", cfgPath, err)
	}
	defer f.Close()

	if len(existing) > 0 {
		lastByte := existing[len(existing)-1]
		if lastByte != '\n' {
			_, _ = f.WriteString("\n")
		}
	}
	_, err = fmt.Fprintf(f, "\n%s\n", aliasBlock)
	if err != nil {
		return fmt.Errorf("failed to write aliases: %w", err)
	}

	fmt.Printf("Twine aliases appended to %s\n", cfgPath)
	return nil
}