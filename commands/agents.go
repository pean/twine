package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/agents"
	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/mux"
	"github.com/pean/twine/internal/ui"
)

var agentsCmd = &cobra.Command{
	Use: "agents",
	Short:   "List and switch to AI coding agents in your sessions",
	Long: `Display a unified dashboard of all running AI coding agents.

Detects:
  - Claude Code sessions (via ~/.claude/sessions/*.json)
  - OpenCode sessions (via pane commands)

Agents are searched in every available multiplexer (or only the one given
with --mux). Select an agent to switch to its session.`,
	RunE: runAgents,
}

func runAgents(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load()
	set, err := muxSet(cfg)
	if err != nil {
		return err
	}

	type found struct {
		agent   agents.Agent
		backend mux.Multiplexer
	}
	var all []found
	for _, b := range set.Backends("") {
		list, err := agents.FindAll(b)
		if err != nil {
			return fmt.Errorf("failed to discover agents in %s: %w", b.Name(), err)
		}
		for _, a := range list {
			all = append(all, found{agent: a, backend: b})
		}
	}

	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "No AI coding agents found running in any session.")
		return nil
	}

	items := make([]ui.Item, 0, len(all))
	for i, f := range all {
		a := f.agent
		indicator := "🤖"
		if a.Type == agents.AgentOpenCode {
			indicator = "🔓"
		}
		coloredState := agents.ColorizeState(a.State)
		title := fmt.Sprintf(
			"%s %-12s %s  %s%s  %s  %s",
			indicator, string(a.Type),
			coloredState,
			a.Session, muxLabel(set, f.backend),
			a.Directory,
			a.Model,
		)
		items = append(items, ui.Item{
			Title:  title,
			Value:  strconv.Itoa(i),
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

	i, _ := strconv.Atoi(chosen.Value)
	return all[i].backend.AttachOrSwitch(all[i].agent.Session)
}
