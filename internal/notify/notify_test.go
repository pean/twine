package notify

import (
	"reflect"
	"testing"
)

func TestCommandDarwinPassesTextAsArguments(t *testing.T) {
	title, body := `Claude "needs" input`, `proj/main'; do shell script "x"`
	name, args := command("darwin", title, body)
	if name != "osascript" {
		t.Fatalf("name = %q", name)
	}
	tail := args[len(args)-2:]
	if !reflect.DeepEqual(tail, []string{body, title}) {
		t.Errorf("text args = %q, want body then title", tail)
	}
	for _, a := range args[:len(args)-2] {
		if a != "-e" && (a == title || a == body) {
			t.Errorf("text spliced into script: %q", a)
		}
	}
}

func TestCommandLinux(t *testing.T) {
	name, args := command("linux", "t", "b")
	if name != "notify-send" || !reflect.DeepEqual(args, []string{"t", "b"}) {
		t.Errorf("command = %q %q", name, args)
	}
}

func TestCommandUnsupported(t *testing.T) {
	if name, _ := command("plan9", "t", "b"); name != "" {
		t.Errorf("name = %q, want none", name)
	}
}
