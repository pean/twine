package agents

import (
	"reflect"
	"testing"
)

func snap(states ...AgentState) map[string]Agent {
	m := map[string]Agent{}
	keys := []string{"a", "b"}
	for i, s := range states {
		m[keys[i]] = Agent{State: s, Session: "s" + keys[i]}
	}
	return m
}

func kinds(events []Event) []EventKind {
	var k []EventKind
	for _, e := range events {
		k = append(k, e.Kind)
	}
	return k
}

func TestTrackerBaselineRaisesNothing(t *testing.T) {
	tr := NewTracker()
	if ev := tr.Update(snap(StateInput, StateWorking)); len(ev) != 0 {
		t.Errorf("first snapshot raised %v", ev)
	}
}

func TestTrackerNeedsInputAfterTwoPolls(t *testing.T) {
	tr := NewTracker()
	tr.Update(snap(StateWorking))
	if ev := tr.Update(snap(StateInput)); len(ev) != 0 {
		t.Fatalf("raised %v after one poll", ev)
	}
	ev := tr.Update(snap(StateInput))
	if !reflect.DeepEqual(kinds(ev), []EventKind{EventNeedsInput}) {
		t.Fatalf("events = %v, want needs input", ev)
	}
	if ev[0].Key != "a" || ev[0].Agent.Session != "sa" {
		t.Errorf("event = %+v", ev[0])
	}
	if ev := tr.Update(snap(StateInput)); len(ev) != 0 {
		t.Errorf("raised %v again for the same state", ev)
	}
}

func TestTrackerFinishedOnlyAfterWorking(t *testing.T) {
	tr := NewTracker()
	tr.Update(snap(StateWorking))
	tr.Update(snap(StateIdle))
	ev := tr.Update(snap(StateIdle))
	if !reflect.DeepEqual(kinds(ev), []EventKind{EventFinished}) {
		t.Fatalf("events = %v, want finished", ev)
	}

	tr = NewTracker()
	tr.Update(snap(StateIdle))
	tr.Update(snap(StateWorking))
	tr.Update(snap(StateWorking))
	tr.Update(snap(StateInput))
	tr.Update(snap(StateInput))
	// From waiting for input to idle is not "finished".
	tr.Update(snap(StateIdle))
	if ev := tr.Update(snap(StateIdle)); len(ev) != 0 {
		t.Errorf("input to idle raised %v", ev)
	}
}

func TestTrackerIgnoresFlicker(t *testing.T) {
	tr := NewTracker()
	tr.Update(snap(StateWorking))
	// One idle reading between working readings is noise.
	for _, s := range []AgentState{StateIdle, StateWorking, StateIdle, StateWorking} {
		if ev := tr.Update(snap(s)); len(ev) != 0 {
			t.Fatalf("flicker to %q raised %v", s, ev)
		}
	}
}

func TestTrackerAgentsAreIndependentAndForgotten(t *testing.T) {
	tr := NewTracker()
	tr.Update(snap(StateWorking, StateWorking))
	tr.Update(snap(StateIdle, StateWorking))
	ev := tr.Update(snap(StateIdle, StateWorking))
	if len(ev) != 1 || ev[0].Key != "a" {
		t.Fatalf("events = %v, want only agent a", ev)
	}

	// An agent that disappears and comes back starts from a new baseline.
	tr.Update(snap())
	tr.Update(snap(StateInput))
	if ev := tr.Update(snap(StateInput)); len(ev) != 0 {
		t.Errorf("returning agent raised %v", ev)
	}
}
