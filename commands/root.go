package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "twine",
	Short: "Git worktree + terminal multiplexer session management",
	Long: `Twine manages git worktrees and multiplexer sessions together.

New sessions go to the multiplexer chosen by, in order: --mux, the
TWINE_MUX environment variable, a matching [[repos]] entry in the config,
and the top-level multiplexer setting ("auto" by default). Existing
sessions are found in every installed multiplexer unless --mux is given.`,
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
	rootCmd.PersistentFlags().StringVar(
		&muxFlag, "mux", "",
		"multiplexer to use, ignoring all others: auto, tmux, or rex",
	)
	_ = rootCmd.RegisterFlagCompletionFunc(
		"mux",
		cobra.FixedCompletions(config.Multiplexers, cobra.ShellCompDirectiveNoFileComp),
	)

	rootCmd.AddCommand(worktreeCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(listCmd)
}