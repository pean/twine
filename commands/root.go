package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "twine",
	Short: "Git worktree + tmux session management",
	Long:  `Twine manages git worktrees and tmux sessions together.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(installCmd)
}