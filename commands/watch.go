package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/agents"
	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/notify"
)

var watchInterval time.Duration

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Notify when an AI coding agent needs input or finishes",
	Long: `Watch the AI coding agents in every multiplexer and show a desktop
notification when one starts waiting for you or stops working.

Runs until interrupted, so start it in a spare pane or session. An agent must
hold its new state for two checks in a row before it counts.`,
	Args: cobra.NoArgs,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().DurationVar(
		&watchInterval, "interval", 3*time.Second,
		"how often to check the agents (at least 1s)",
	)
}

func runWatch(cmd *cobra.Command, args []string) error {
	if watchInterval < time.Second {
		return fmt.Errorf("--interval must be at least 1s")
	}
	cfg, _ := config.Load()
	set, err := muxSet(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	var names []string
	for _, b := range set.Backends("") {
		names = append(names, b.Name())
	}
	fmt.Printf(
		"Watching agents in %s every %s. Ctrl-C to stop.\n",
		strings.Join(names, ", "), watchInterval,
	)

	tracker := agents.NewTracker()
	warned := false
	poll := func() {
		found, err := discoverAgents(set)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			return
		}
		current := make(map[string]agents.Agent, len(found))
		for _, f := range found {
			current[f.backend.Name()+":"+f.agent.PaneID] = f.agent
		}
		for _, ev := range tracker.Update(current) {
			title, body := eventText(ev)
			fmt.Printf("%s  %s  %s\n", time.Now().Format("15:04:05"), title, body)
			if err := notify.Send(title, body); err != nil && !warned {
				warned = true
				fmt.Fprintf(os.Stderr, "Warning: notifications failed: %v\n", err)
			}
		}
	}

	poll()
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			poll()
		}
	}
}

// eventText returns the notification title and body for an event.
func eventText(ev agents.Event) (title, body string) {
	what := "finished"
	if ev.Kind == agents.EventNeedsInput {
		what = "needs input"
	}
	return string(ev.Agent.Type) + " " + what, ev.Agent.Session + "  " + ev.Agent.Directory
}
