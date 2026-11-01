package commands

import (
	"testing"

	"github.com/pean/twine/internal/agents"
)

func TestEventText(t *testing.T) {
	a := agents.Agent{
		Type: agents.AgentClaude, Session: "proj/main", Directory: "~/src/proj",
	}
	cases := []struct {
		kind        agents.EventKind
		title, body string
	}{
		{agents.EventNeedsInput, "Claude Code needs input", "proj/main  ~/src/proj"},
		{agents.EventFinished, "Claude Code finished", "proj/main  ~/src/proj"},
	}
	for _, tc := range cases {
		title, body := eventText(agents.Event{Agent: a, Kind: tc.kind})
		if title != tc.title || body != tc.body {
			t.Errorf("eventText(%v) = %q, %q", tc.kind, title, body)
		}
	}
}
