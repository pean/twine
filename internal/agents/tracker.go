package agents

// EventKind says why a state change is worth telling the user about.
type EventKind int

const (
	// EventNeedsInput means the agent is waiting for the user.
	EventNeedsInput EventKind = iota + 1
	// EventFinished means the agent stopped working.
	EventFinished
)

// Event is a notable change in an agent's state.
type Event struct {
	Key   string
	Agent Agent
	Kind  EventKind
}

// confirmations is how many polls in a row must show a new state before it
// counts. States read from the screen can flicker for a moment.
const confirmations = 2

type tracked struct {
	stable    AgentState
	candidate AgentState
	seen      int
}

// Tracker turns repeated snapshots of the running agents into events.
type Tracker struct {
	agents map[string]*tracked
}

// NewTracker returns a Tracker with no history. The first snapshot only sets
// the baseline.
func NewTracker() *Tracker {
	return &Tracker{agents: map[string]*tracked{}}
}

// Update takes a snapshot of the running agents by key and returns the events
// since the last one. An agent seen for the first time raises none.
func (t *Tracker) Update(current map[string]Agent) []Event {
	var events []Event
	for key, a := range current {
		prev, known := t.agents[key]
		if !known {
			t.agents[key] = &tracked{stable: a.State}
			continue
		}
		if a.State == prev.stable {
			prev.candidate, prev.seen = "", 0
			continue
		}
		if a.State != prev.candidate {
			prev.candidate, prev.seen = a.State, 0
		}
		prev.seen++
		if prev.seen < confirmations {
			continue
		}
		if kind := eventFor(prev.stable, a.State); kind != 0 {
			events = append(events, Event{Key: key, Agent: a, Kind: kind})
		}
		prev.stable, prev.candidate, prev.seen = a.State, "", 0
	}
	for key := range t.agents {
		if _, ok := current[key]; !ok {
			delete(t.agents, key)
		}
	}
	return events
}

func eventFor(from, to AgentState) EventKind {
	switch {
	case to == StateInput:
		return EventNeedsInput
	case from == StateWorking && to == StateIdle:
		return EventFinished
	}
	return 0
}
