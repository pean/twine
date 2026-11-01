package tmux

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/pean/twine/internal/mux"
)

func TestParseSessions(t *testing.T) {
	out := "proj/main|/src/proj.git/main|2\n" +
		"proj/feature/x|/src/proj.git/feature/x|1\n" +
		"malformed\n" +
		"\n"
	got := parseSessions(out)
	want := []mux.Session{
		{Name: "proj/main", Path: "/src/proj.git/main", Windows: 2},
		{Name: "proj/feature/x", Path: "/src/proj.git/feature/x", Windows: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseSessions() = %+v, want %+v", got, want)
	}
}

func TestParseSessionsEmpty(t *testing.T) {
	if got := parseSessions(""); len(got) != 0 {
		t.Errorf("parseSessions(\"\") = %+v, want empty", got)
	}
}

func TestParsePanes(t *testing.T) {
	out := "123|proj/main|/src/proj|claude|%1\n" +
		"456|other|/tmp|fish|%7\n" +
		"bad|line\n"
	got := parsePanes(out)
	want := []mux.Pane{
		{PID: 123, Session: "proj/main", Path: "/src/proj", Command: "claude", ID: "%1"},
		{PID: 456, Session: "other", Path: "/tmp", Command: "fish", ID: "%7"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsePanes() = %+v, want %+v", got, want)
	}
}

func TestNewSessionErrorDuplicate(t *testing.T) {
	base := errors.New("exit status 1")
	err := newSessionError(base, "duplicate session: proj/main\n")
	if !errors.Is(err, mux.ErrSessionExists) {
		t.Errorf("expected ErrSessionExists, got %v", err)
	}

	err = newSessionError(base, "no server running\n")
	if errors.Is(err, mux.ErrSessionExists) {
		t.Errorf("unexpected ErrSessionExists for %v", err)
	}
	if !errors.Is(err, base) {
		t.Errorf("expected wrapped base error, got %v", err)
	}
}

func TestName(t *testing.T) {
	if got := New().Name(); got != "tmux" {
		t.Errorf("Name() = %q, want tmux", got)
	}
}

type fakeTmux struct {
	calls    []string
	failOn   string
	nextPane int
}

func (f *fakeTmux) run(args ...string) (string, error) {
	line := strings.Join(args, " ")
	f.calls = append(f.calls, line)
	if f.failOn != "" && strings.HasPrefix(line, f.failOn) {
		return "boom", errors.New("exit status 1")
	}
	for _, a := range args {
		if a == "-P" {
			f.nextPane++
			return fmt.Sprintf("%%%d", f.nextPane), nil
		}
	}
	return "", nil
}

func newFake() (*Backend, *fakeTmux) {
	f := &fakeTmux{}
	return &Backend{run: f.run}, f
}

var devLayout = mux.Layout{Windows: []mux.LayoutWindow{
	{Name: "code", Panes: []mux.LayoutPane{
		{Command: "nvim"},
		{Split: "left", Size: 40, Command: "git status"},
		{Split: "below", Size: 25},
	}},
	{Name: "server", Panes: []mux.LayoutPane{{Command: "ls"}}},
}}

func TestNewSessionWithoutLayoutIsPlain(t *testing.T) {
	b, f := newFake()
	if err := b.NewSession("proj/main", "/src/proj", mux.NewSessionOptions{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"new-session -d -s proj/main -c /src/proj"}
	if !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls = %q, want %q", f.calls, want)
	}
}

func TestNewSessionBuildsLayout(t *testing.T) {
	b, f := newFake()
	opts := mux.NewSessionOptions{Layout: &devLayout}
	if err := b.NewSession("proj/main", "/src/proj", opts); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"new-session -d -s proj/main -c /src/proj -P -F #{pane_id} -n code",
		"send-keys -t %1 -l nvim",
		"send-keys -t %1 Enter",
		"split-window -t %1 -c /src/proj -P -F #{pane_id} -l 40% -h -b",
		"send-keys -t %2 -l git status",
		"send-keys -t %2 Enter",
		"split-window -t %2 -c /src/proj -P -F #{pane_id} -l 25% -v",
		"new-window -t proj/main: -c /src/proj -P -F #{pane_id} -n server",
		"send-keys -t %4 -l ls",
		"send-keys -t %4 Enter",
		"select-pane -t %1",
		"select-pane -t %4",
		"select-window -t %1",
	}
	if !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls:\n%s\nwant:\n%s",
			strings.Join(f.calls, "\n"), strings.Join(want, "\n"))
	}
}

func TestNewSessionKillsHalfBuiltSession(t *testing.T) {
	b, f := newFake()
	f.failOn = "split-window"
	opts := mux.NewSessionOptions{Layout: &devLayout}
	if err := b.NewSession("proj/main", "/src/proj", opts); err == nil {
		t.Fatal("expected an error")
	}
	if last := f.calls[len(f.calls)-1]; last != "kill-session -t proj/main" {
		t.Errorf("last call = %q, want kill-session", last)
	}
}

func TestSplitFlags(t *testing.T) {
	cases := map[string]string{
		"left": "-h -b", "right": "-h", "above": "-v -b", "below": "-v",
	}
	for dir, want := range cases {
		if got := strings.Join(splitFlags(dir), " "); got != want {
			t.Errorf("splitFlags(%q) = %q, want %q", dir, got, want)
		}
	}
}
