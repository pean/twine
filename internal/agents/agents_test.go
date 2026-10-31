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
