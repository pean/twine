package rex

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/pean/twine/internal/mux"
)

func panes(ps ...mux.LayoutPane) mux.LayoutWindow {
	return mux.LayoutWindow{Name: "w", Panes: ps}
}

// shape renders a tree as nested text, e.g. H0.4(a,b), with each pane named by
// the command it types.
func shape(n *node) string {
	if n.Block != nil {
		cmd, _ := n.Block.Options["initial_input"].(string)
		if cmd == "" {
			return "-"
		}
		return cmd[:len(cmd)-1]
	}
	s := n.Split
	dir := map[string]string{"horizontal": "H", "vertical": "V"}[s.Direction]
	return dir + json.Number(formatRatio(s.Ratio)).String() +
		"(" + shape(s.Before) + "," + shape(s.After) + ")"
}

func formatRatio(r float64) string {
	b, _ := json.Marshal(r)
	return string(b)
}

func TestWindowLayoutDirections(t *testing.T) {
	cases := []struct {
		split string
		size  int
		want  string
	}{
		{"right", 25, "H0.75(a,b)"},
		{"", 0, "H0.5(a,b)"},
		{"left", 40, "H0.4(b,a)"},
		{"below", 30, "V0.7(a,b)"},
		{"above", 30, "V0.3(b,a)"},
	}
	for _, tc := range cases {
		w := panes(
			mux.LayoutPane{Command: "a"},
			mux.LayoutPane{Command: "b", Split: tc.split, Size: tc.size},
		)
		spec, _ := windowLayout("/src", w)
		if got := shape(spec.Layout); got != tc.want {
			t.Errorf("split %q size %d: %s, want %s", tc.split, tc.size, got, tc.want)
		}
	}
}

func TestWindowLayoutEachPaneSplitsThePreviousOne(t *testing.T) {
	w := panes(
		mux.LayoutPane{Command: "main"},
		mux.LayoutPane{Command: "console", Split: "left", Size: 40},
		mux.LayoutPane{Command: "diff", Split: "below", Size: 66},
		mux.LayoutPane{Command: "status", Split: "below", Size: 50},
	)
	spec, first := windowLayout("/src", w)
	want := "H0.4(V0.34(console,V0.5(diff,status)),main)"
	if got := shape(spec.Layout); got != want {
		t.Errorf("shape = %s, want %s", got, want)
	}
	// Layout order is console, diff, status, main.
	if first != 3 {
		t.Errorf("first pane index = %d, want 3", first)
	}
	if spec.Label != "w" {
		t.Errorf("label = %q", spec.Label)
	}
}

func TestWindowLayoutSetsDirectoryAndTypesCommands(t *testing.T) {
	spec, first := windowLayout("/src/proj", panes(
		mux.LayoutPane{Command: "nvim"}, mux.LayoutPane{},
	))
	if first != 0 {
		t.Errorf("first = %d, want 0", first)
	}
	got := leaves(spec.Layout)
	opts := got[0].Options
	if opts["initial_input"] != "nvim\n" || opts["cwd"] != "/src/proj" {
		t.Errorf("first block options = %v", opts)
	}
	if _, typed := got[1].Options["initial_input"]; typed {
		t.Errorf("empty command should type nothing: %v", got[1].Options)
	}
	if got[1].Flavor != shellFlavor {
		t.Errorf("flavor = %q", got[1].Flavor)
	}
}

func TestNewSessionWithLayoutCreatesInOneCallThenFocuses(t *testing.T) {
	b, f := newFake()
	f.out["api"] = `{"session_id":"session:9","initial_windows":[
	  {"block_ids":["block:w1a","block:w1b"]},
	  {"block_ids":["block:w2a"]}]}`
	layout := mux.Layout{Windows: []mux.LayoutWindow{
		panes(mux.LayoutPane{Command: "a"}, mux.LayoutPane{Split: "left"}),
		panes(mux.LayoutPane{Command: "c"}),
	}}
	opts := mux.NewSessionOptions{Layout: &layout}
	if err := b.NewSession("proj/new", "/src", opts); err != nil {
		t.Fatal(err)
	}

	var create, focus [][]string
	for _, c := range f.calls {
		switch c[0] {
		case "api":
			create = append(create, c)
		case "focus":
			focus = append(focus, c)
		}
	}
	if len(create) != 1 || create[0][2] != "session.create" {
		t.Fatalf("create calls = %v", create)
	}
	var req struct {
		Label   string `json:"label"`
		Windows []struct {
			Label string `json:"window_label"`
		} `json:"initial_windows"`
	}
	if err := json.Unmarshal([]byte(create[0][3]), &req); err != nil {
		t.Fatal(err)
	}
	if req.Label != "proj/new" || len(req.Windows) != 2 {
		t.Errorf("request = %+v", req)
	}

	// Window 1's first pane is the second block in layout order (left split),
	// and the first window is focused last so it ends up active.
	want := [][]string{
		{"focus", "-s", "session:9", "block:w2a"},
		{"focus", "-s", "session:9", "block:w1b"},
	}
	if !reflect.DeepEqual(focus, want) {
		t.Errorf("focus calls = %v, want %v", focus, want)
	}
}

func TestNewSessionWithLayoutFailureCreatesNothingElse(t *testing.T) {
	b, f := newFake()
	f.err["api"] = errString("bad_request")
	layout := mux.Layout{Windows: []mux.LayoutWindow{panes(mux.LayoutPane{})}}
	opts := mux.NewSessionOptions{Layout: &layout}
	if err := b.NewSession("x", "/src", opts); err == nil {
		t.Fatal("expected an error")
	}
	for _, c := range f.calls {
		if c[0] == "focus" {
			t.Errorf("unexpected focus call %v", c)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
