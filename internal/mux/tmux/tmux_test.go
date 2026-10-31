package tmux

import (
	"errors"
	"reflect"
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
	if got := New(Options{}).Name(); got != "tmux" {
		t.Errorf("Name() = %q, want tmux", got)
	}
}
