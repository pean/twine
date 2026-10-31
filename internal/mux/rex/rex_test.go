package rex

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pean/twine/internal/mux"
)

const lsJSON = `{"sessions":[
  {"session_id":"session:1","label":"proj/main"},
  {"session_id":"session:2","label":"proj/feat x"},
  {"session_id":"session:3","label":"dup"},
  {"session_id":"session:4","label":"dup"}]}`

const clientsJSON = `{"clients":[
  {"client_id":"client:cli","info":{"kind":"cli"},
   "connection_state":"connected"},
  {"client_id":"client:old","info":{"kind":"app"},
   "connection_state":"reconnecting"},
  {"client_id":"client:app","info":{"kind":"app"},
   "connection_state":"connected"}]}`

type fakeRex struct {
	calls [][]string
	out   map[string]string
	err   map[string]error
}

func (f *fakeRex) run(args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	for _, a := range args {
		if err, ok := f.err[a]; ok {
			return nil, err
		}
	}
	for _, a := range args {
		if out, ok := f.out[a]; ok {
			return []byte(out), nil
		}
	}
	return nil, nil
}

func newFake() (*Backend, *fakeRex) {
	f := &fakeRex{
		out: map[string]string{"ls": lsJSON, "client": clientsJSON},
		err: map[string]error{},
	}
	return &Backend{run: f.run}, f
}

func (f *fakeRex) last() []string { return f.calls[len(f.calls)-1] }

func TestHasSessionIsExact(t *testing.T) {
	b, _ := newFake()
	for name, want := range map[string]bool{
		"proj/main": true,
		"PROJ/MAIN": false,
		"proj":      false,
		"missing":   false,
	} {
		if got := b.HasSession(name); got != want {
			t.Errorf("HasSession(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestQueriesDoNotAutostartServer(t *testing.T) {
	b, f := newFake()
	b.HasSession("x")
	if f.last()[0] != readOnly {
		t.Errorf("rex ls args = %v, want %s first", f.last(), readOnly)
	}
}

func TestListSessionsWhenServerDown(t *testing.T) {
	b, f := newFake()
	f.err["ls"] = errors.New("exit status 4")
	got, err := b.ListSessions()
	if err != nil || len(got) != 0 {
		t.Errorf("ListSessions() = %v, %v; want none", got, err)
	}
	if b.HasSession("proj/main") {
		t.Error("HasSession true while server is down")
	}
}

func TestListSessions(t *testing.T) {
	b, _ := newFake()
	got, _ := b.ListSessions()
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"proj/main", "proj/feat x", "dup", "dup"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestNewSession(t *testing.T) {
	b, f := newFake()
	err := b.NewSession("proj/new", ".", mux.NewSessionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got := f.last()
	if got[0] != "new" || got[1] != "proj/new" || got[2] != "--cwd" {
		t.Fatalf("args = %v", got)
	}
	if !filepath.IsAbs(got[3]) {
		t.Errorf("cwd %q is not absolute", got[3])
	}
}

func TestNewSessionExists(t *testing.T) {
	b, f := newFake()
	err := b.NewSession("proj/main", "/tmp", mux.NewSessionOptions{})
	if !errors.Is(err, mux.ErrSessionExists) {
		t.Errorf("err = %v, want ErrSessionExists", err)
	}
	for _, c := range f.calls {
		if c[0] == "new" {
			t.Error("created a duplicate session")
		}
	}
}

func TestKillSessionTargetsByID(t *testing.T) {
	b, f := newFake()
	if err := b.KillSession("proj/feat x"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"kill", "session:2"}; !reflect.DeepEqual(f.last(), want) {
		t.Errorf("args = %v, want %v", f.last(), want)
	}
}

func TestKillSessionRefusesMissingAndAmbiguous(t *testing.T) {
	b, f := newFake()
	for _, name := range []string{"missing", "dup"} {
		if err := b.KillSession(name); err == nil {
			t.Errorf("KillSession(%q) succeeded", name)
		}
	}
	for _, c := range f.calls {
		if c[0] == "kill" {
			t.Errorf("unexpected kill call %v", c)
		}
	}
}

func TestAttachOrSwitchAsksTheConnectedApp(t *testing.T) {
	b, f := newFake()
	if err := b.AttachOrSwitch("proj/main"); err != nil {
		t.Fatal(err)
	}
	got := f.last()
	if !reflect.DeepEqual(got[:3], []string{"api", "call", "client.perform"}) {
		t.Fatalf("args = %v", got)
	}
	var req struct {
		ClientID string            `json:"client_id"`
		Action   string            `json:"action"`
		Args     map[string]string `json:"args"`
	}
	if err := json.Unmarshal([]byte(got[3]), &req); err != nil {
		t.Fatal(err)
	}
	if req.ClientID != "client:app" || req.Action != "session.select" ||
		req.Args["session_id"] != "session:1" {
		t.Errorf("request = %+v", req)
	}
}

func TestAttachOrSwitchWithoutApp(t *testing.T) {
	b, f := newFake()
	f.out["client"] = `{"clients":[{"client_id":"c","info":{"kind":"cli"},
		"connection_state":"connected"}]}`
	err := b.AttachOrSwitch("proj/main")
	if err == nil || !strings.Contains(err.Error(), "no Rex app") {
		t.Errorf("err = %v, want no Rex app", err)
	}
}

func TestAttachOrSwitchSurfacesRefusal(t *testing.T) {
	b, f := newFake()
	f.err["api"] = errors.New("Turn on Remote Control in its Settings")
	err := b.AttachOrSwitch("proj/main")
	if err == nil || !strings.Contains(err.Error(), "Remote Control") {
		t.Errorf("err = %v, want Remote Control hint", err)
	}
}

func TestInside(t *testing.T) {
	b, _ := newFake()
	t.Setenv("REX_SESSION", "")
	if b.Inside() {
		t.Error("Inside() true without REX_SESSION")
	}
	t.Setenv("REX_SESSION", "session:1")
	if !b.Inside() {
		t.Error("Inside() false with REX_SESSION")
	}
}
