package mux

import "fmt"

// Layout describes the windows and panes a new session starts with.
type Layout struct {
	Windows []LayoutWindow `toml:"windows"`
}

// LayoutWindow is one window (a tab in rex) of a Layout.
type LayoutWindow struct {
	Name  string       `toml:"name"`
	Panes []LayoutPane `toml:"panes"`
}

// LayoutPane is one terminal. Every pane after the first splits the pane before it.
type LayoutPane struct {
	// Command is typed into the pane's shell, which stays open after it.
	Command string `toml:"command"`
	// Split puts the pane left, right, above, or below the previous pane.
	// Empty means right. The first pane of a window cannot set it.
	Split string `toml:"split"`
	// Size is the percentage of the previous pane this pane takes. Zero means
	// 50. The first pane of a window cannot set it.
	Size int `toml:"size"`
}

// Split directions.
const (
	SplitLeft  = "left"
	SplitRight = "right"
	SplitAbove = "above"
	SplitBelow = "below"
)

// SplitDirection returns the pane's direction with the default applied.
func (p LayoutPane) SplitDirection() string {
	if p.Split == "" {
		return SplitRight
	}
	return p.Split
}

// SplitSize returns the pane's size percentage with the default applied.
func (p LayoutPane) SplitSize() int {
	if p.Size == 0 {
		return 50
	}
	return p.Size
}

// Validate reports the first problem that would make the layout unusable.
func (l Layout) Validate() error {
	if len(l.Windows) == 0 {
		return fmt.Errorf("no windows")
	}
	for i, w := range l.Windows {
		if len(w.Panes) == 0 {
			return fmt.Errorf("windows[%d]: no panes", i)
		}
		for j, p := range w.Panes {
			if err := p.validate(j == 0); err != nil {
				return fmt.Errorf("windows[%d].panes[%d]: %w", i, j, err)
			}
		}
	}
	return nil
}

func (p LayoutPane) validate(first bool) error {
	if first {
		if p.Split != "" || p.Size != 0 {
			return fmt.Errorf("the first pane of a window cannot set split or size")
		}
		return nil
	}
	switch p.SplitDirection() {
	case SplitLeft, SplitRight, SplitAbove, SplitBelow:
	default:
		return fmt.Errorf(
			"split %q must be left, right, above, or below", p.Split,
		)
	}
	if p.Size != 0 && (p.Size < 1 || p.Size > 99) {
		return fmt.Errorf("size %d must be between 1 and 99", p.Size)
	}
	return nil
}
