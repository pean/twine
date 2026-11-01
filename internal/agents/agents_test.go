package agents

import (
	"testing"

	"github.com/pean/twine/internal/mux"
)

// fakeMux implements mux.Multiplexer only.
type fakeMux struct{}

func (fakeMux) Name() string                         { return "fake" }
func (fakeMux) Available() bool                      { return true }
func (fakeMux) Inside() bool                         { return false }
func (fakeMux) HasSession(string) bool               { return false }
func (fakeMux) ListSessions() ([]mux.Session, error) { return nil, nil }
func (fakeMux) NewSession(string, string, mux.NewSessionOptions) error {
	return nil
}
func (fakeMux) KillSession(string) error    { return nil }
func (fakeMux) AttachOrSwitch(string) error { return nil }

// fakeInspector adds mux.PaneInspector with canned panes and screens.
type fakeInspector struct {
	fakeMux
	panes   []mux.Pane
	screens map[string]string
}

func (f fakeInspector) ListPanes() ([]mux.Pane, error) { return f.panes, nil }
func (f fakeInspector) CapturePane(id string, _ int) (string, error) {
	return f.screens[id], nil
}

func TestFindAllWithoutPaneInspector(t *testing.T) {
	got, err := FindAll(fakeMux{})
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("FindAll() = %+v, want no agents", got)
	}
}

func TestFindOpenCodeAgents(t *testing.T) {
	inspector := fakeInspector{
		panes: []mux.Pane{
			{PID: 1, Session: "proj/main", Path: "/src/proj", Command: "opencode", ID: "%1"},
			{PID: 2, Session: "proj/main", Path: "/src/proj", Command: "fish", ID: "%2"},
		},
		screens: map[string]string{"%1": "Executing tool"},
	}
	got, err := findOpenCodeAgents(inspector)
	if err != nil {
		t.Fatalf("findOpenCodeAgents() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("findOpenCodeAgents() = %+v, want 1 agent", got)
	}
	a := got[0]
	if a.Session != "proj/main" || a.PaneID != "%1" || a.State != StateWorking {
		t.Errorf("unexpected agent %+v", a)
	}
}

func TestStateFromStatus(t *testing.T) {
	claude := func(state string) *mux.PaneStatus {
		return &mux.PaneStatus{State: state, App: "claude-code"}
	}
	cases := []struct {
		desc   string
		status *mux.PaneStatus
		want   AgentState
		ok     bool
	}{
		{"working", claude("working"), StateWorking, true},
		{"blocked", claude("blocked"), StateInput, true},
		{"idle", claude("idle"), StateIdle, true},
		{"done", claude("done"), StateIdle, true},
		{"error", claude("error"), StateIdle, true},
		{"unknown state", claude("later"), "", false},
		{"other app", &mux.PaneStatus{State: "working", App: "cargo"}, "", false},
		{"no status", nil, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, ok := stateFromStatus(tc.status, "claude-code")
			if got != tc.want || ok != tc.ok {
				t.Errorf("got %q, %v; want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestFindOpenCodeAgentsPrefersReportedStatus(t *testing.T) {
	inspector := fakeInspector{
		panes: []mux.Pane{{
			Session: "proj/main", Command: "opencode", ID: "%1",
			Status: &mux.PaneStatus{State: "blocked", App: "opencode"},
		}},
		// The screen says working; the program's own report wins.
		screens: map[string]string{"%1": "Executing tool"},
	}
	got, _ := findOpenCodeAgents(inspector)
	if len(got) != 1 || got[0].State != StateInput {
		t.Errorf("findOpenCodeAgents() = %+v, want state %q", got, StateInput)
	}
}
