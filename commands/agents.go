package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/agents"
	"github.com/pean/twine/internal/tmux"
	"github.com/pean/twine/internal/ui"
)

var agentsCmd = &cobra.Command{
	Use: "agents",
	Short:   "List and switch to AI coding agents in tmux",
	Long: `Display a unified dashboard of all running AI coding agents.

Detects:
  - Claude Code sessions (via ~/.claude/sessions/*.json)
  - OpenCode sessions (via tmux pane commands)

Select an agent to switch to its tmux session.`,
	RunE: runAgents,
}

func runAgents(cmd *cobra.Command, args []string) error {
	if !tmux.IsInsideTmux() {
		sessions, _ := tmux.ListSessions()
		if len(sessions) == 0 {
			fmt.Fprintln(os.Stderr, "No tmux sessions found.")
			return nil
		}
	}

	agentList, err := agents.FindAll()
	if err != nil {
		return fmt.Errorf("failed to discover agents: %w", err)
	}

	if len(agentList) == 0 {
		fmt.Fprintln(
			os.Stderr,
			"No AI coding agents found running in tmux sessions.",
		)
		return nil
	}

	items := make([]ui.Item, 0, len(agentList))
	for _, a := range agentList {
		indicator := "🤖"
		if a.Type == agents.AgentOpenCode {
			indicator = "🔓"
		}
		coloredState := agents.ColorizeState(a.State)
		title := fmt.Sprintf(
			"%s %-12s %s  %s  %s  %s",
			indicator, string(a.Type),
			coloredState,
			a.Session,
			a.Directory,
			a.Model,
		)
		items = append(items, ui.Item{
			Title:  title,
			Value:  a.Session,
			Active: true,
		})
	}

	chosen, err := ui.Select(items, "AI Coding Agents – select to switch:")
	if errors.Is(err, ui.ErrCancelled) {
		return nil
	}
	if err != nil {
		return err
	}

	return tmux.AttachOrSwitch(chosen.Value)
}