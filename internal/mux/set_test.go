package mux

import (
	"reflect"
	"strings"
	"testing"
)

type fake struct {
	name      string
	available bool
	inside    bool
	sessions  []string
}

func (f *fake) Name() string                { return f.name }
func (f *fake) Available() bool             { return f.available }
func (f *fake) Inside() bool                { return f.inside }
func (f *fake) KillSession(string) error    { return nil }
func (f *fake) AttachOrSwitch(string) error { return nil }
func (f *fake) NewSession(string, string, NewSessionOptions) error {
	return nil
}

func (f *fake) HasSession(name string) bool {
	for _, s := range f.sessions {
		if s == name {
			return true
		}
	}
	return false
}

func (f *fake) ListSessions() ([]Session, error) {
	var result []Session
	for _, s := range f.sessions {
		result = append(result, Session{Name: s})
	}
	return result, nil
}

func fixed(c Choice) Chooser { return func(string) Choice { return c } }

func names(bs []Multiplexer) []string {
	var result []string
	for _, b := range bs {
		result = append(result, b.Name())
	}
	return result
}

func TestForAuto(t *testing.T) {
	cases := []struct {
		desc      string
		tmux, rex *fake
		want      string
		wantErr   bool
	}{
		{
			desc: "prefers tmux when outside both",
			tmux: &fake{name: "tmux", available: true},
			rex:  &fake{name: "rex", available: true},
			want: "tmux",
		},
		{
			desc: "prefers the backend twine runs inside",
			tmux: &fake{name: "tmux", available: true},
			rex:  &fake{name: "rex", available: true, inside: true},
			want: "rex",
		},
		{
			desc: "tmux inside rex is tmux, since tmux comes first",
			tmux: &fake{name: "tmux", available: true, inside: true},
			rex:  &fake{name: "rex", available: true, inside: true},
			want: "tmux",
		},
		{
			desc: "falls back to the installed one",
			tmux: &fake{name: "tmux"},
			rex:  &fake{name: "rex", available: true},
			want: "rex",
		},
		{
			desc:    "errors when nothing is installed",
			tmux:    &fake{name: "tmux"},
			rex:     &fake{name: "rex"},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			s := NewSet([]Multiplexer{tc.tmux, tc.rex}, nil)
			got, err := s.For("")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %s", got.Name())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Name() != tc.want {
				t.Errorf("For() = %s, want %s", got.Name(), tc.want)
			}
		})
	}
}

func TestForNamed(t *testing.T) {
	tmux := &fake{name: "tmux", available: true}
	missing := &fake{name: "rex"}

	s := NewSet([]Multiplexer{tmux, missing}, fixed(Choice{Name: "tmux"}))
	if got, err := s.For(""); err != nil || got != tmux {
		t.Errorf("For() = %v, %v; want tmux", got, err)
	}

	s = NewSet([]Multiplexer{tmux, missing}, fixed(Choice{Name: "rex", Source: "--mux"}))
	_, err := s.For("")
	if err == nil || !strings.Contains(err.Error(), "not installed") ||
		!strings.Contains(err.Error(), "--mux") {
		t.Errorf("For() error = %v, want not installed from --mux", err)
	}

	s = NewSet([]Multiplexer{tmux}, fixed(Choice{Name: "rex"}))
	if _, err := s.For(""); err == nil ||
		!strings.Contains(err.Error(), "not supported") {
		t.Errorf("For() error = %v, want not supported", err)
	}
}

func TestForPerRepo(t *testing.T) {
	tmux := &fake{name: "tmux", available: true}
	rex := &fake{name: "rex", available: true}
	s := NewSet([]Multiplexer{tmux, rex}, func(repo string) Choice {
		if repo == "work" {
			return Choice{Name: "rex"}
		}
		return Choice{Name: Auto}
	})
	if got, _ := s.For("work"); got != rex {
		t.Errorf("For(work) = %s, want rex", got.Name())
	}
	if got, _ := s.For("home"); got != tmux {
		t.Errorf("For(home) = %s, want tmux", got.Name())
	}
}

func TestFindSearchesAllBackendsPreferredFirst(t *testing.T) {
	tmux := &fake{name: "tmux", available: true, sessions: []string{"a", "both"}}
	rex := &fake{name: "rex", available: true, sessions: []string{"b", "both"}}
	gone := &fake{name: "gone", sessions: []string{"c"}}
	s := NewSet(
		[]Multiplexer{tmux, rex, gone},
		fixed(Choice{Name: "rex"}),
	)

	if got := names(s.Backends("")); !reflect.DeepEqual(got, []string{"rex", "tmux"}) {
		t.Errorf("Backends() = %v, want [rex tmux]", got)
	}
	for name, want := range map[string]string{"a": "tmux", "b": "rex", "both": "rex"} {
		got, ok := s.Find("", name)
		if !ok || got.Name() != want {
			t.Errorf("Find(%q) = %v, %v; want %s", name, got, ok, want)
		}
	}
	if _, ok := s.Find("", "c"); ok {
		t.Error("Find found a session in an unavailable backend")
	}
	if got := names(s.Owners("both")); !reflect.DeepEqual(got, []string{"rex", "tmux"}) {
		t.Errorf("Owners(both) = %v", got)
	}
	if got := s.SessionNames(); !reflect.DeepEqual(got, []string{"b", "both", "a"}) {
		t.Errorf("SessionNames() = %v", got)
	}
}

func TestStrictChoiceOnlySearchesThatBackend(t *testing.T) {
	tmux := &fake{name: "tmux", available: true, sessions: []string{"a"}}
	rex := &fake{name: "rex", available: true, sessions: []string{"b"}}
	s := NewSet([]Multiplexer{tmux, rex}, fixed(Choice{Name: "rex", Strict: true}))

	if _, ok := s.Find("", "a"); ok {
		t.Error("strict rex choice found a tmux session")
	}
	if got := s.SessionNames(); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("SessionNames() = %v, want [b]", got)
	}

	s = NewSet([]Multiplexer{tmux}, fixed(Choice{Name: "rex", Strict: true}))
	if got := s.Backends(""); len(got) != 0 {
		t.Errorf("Backends() = %v, want none for unsupported strict choice", names(got))
	}
}
