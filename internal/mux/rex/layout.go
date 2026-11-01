package rex

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/pean/twine/internal/mux"
)

const shellFlavor = "com.superlogical.terminal.shell"

type node struct {
	Block *blockNode `json:"block,omitempty"`
	Split *splitNode `json:"split,omitempty"`
}

type blockNode struct {
	Flavor  string         `json:"flavor"`
	Options map[string]any `json:"options"`
}

type splitNode struct {
	Direction string  `json:"direction"`
	Ratio     float64 `json:"ratio"`
	Before    *node   `json:"before"`
	After     *node   `json:"after"`
}

type windowSpec struct {
	Label  string `json:"window_label,omitempty"`
	Layout *node  `json:"layout"`
}

func newBlock(dir string, p mux.LayoutPane) *blockNode {
	opts := map[string]any{"cwd": dir}
	if p.Command != "" {
		opts["initial_input"] = p.Command + "\n"
	}
	return &blockNode{Flavor: shellFlavor, Options: opts}
}

// windowLayout turns a window's panes into a split tree. Each pane splits the
// one before it, so the tree grows by replacing the previous pane with a split
// that holds it and the new pane. It also returns where the first pane ends
// up in layout order, which is how rex orders a window's block IDs.
func windowLayout(dir string, w mux.LayoutWindow) (windowSpec, int) {
	first := newBlock(dir, w.Panes[0])
	root := &node{Block: first}
	last := root
	for _, p := range w.Panes[1:] {
		fresh := &node{Block: newBlock(dir, p)}
		old := &node{Block: last.Block}
		share := float64(p.SplitSize()) / 100
		s := &splitNode{}
		switch p.SplitDirection() {
		case mux.SplitLeft:
			s.Direction, s.Ratio, s.Before, s.After = "horizontal", share, fresh, old
		case mux.SplitAbove:
			s.Direction, s.Ratio, s.Before, s.After = "vertical", share, fresh, old
		case mux.SplitBelow:
			s.Direction, s.Ratio, s.Before, s.After = "vertical", 1-share, old, fresh
		default:
			s.Direction, s.Ratio, s.Before, s.After = "horizontal", 1-share, old, fresh
		}
		s.Ratio = math.Round(s.Ratio*1e4) / 1e4
		*last = node{Split: s}
		last = fresh
	}
	for i, b := range leaves(root) {
		if b == first {
			return windowSpec{Label: w.Name, Layout: root}, i
		}
	}
	return windowSpec{Label: w.Name, Layout: root}, 0
}

func leaves(n *node) []*blockNode {
	if n.Block != nil {
		return []*blockNode{n.Block}
	}
	return append(leaves(n.Split.Before), leaves(n.Split.After)...)
}

// createWithLayout makes the session and all its windows in one call, so
// either all of it exists or none of it does.
func (b *Backend) createWithLayout(name, dir string, l mux.Layout) error {
	req := struct {
		Label          string       `json:"label"`
		InitialWindows []windowSpec `json:"initial_windows"`
	}{Label: name}
	firsts := make([]int, 0, len(l.Windows))
	for _, w := range l.Windows {
		spec, first := windowLayout(dir, w)
		req.InitialWindows = append(req.InitialWindows, spec)
		firsts = append(firsts, first)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	out, err := b.run("api", "call", "session.create", string(body))
	if err != nil {
		return err
	}
	var resp struct {
		SessionID      string `json:"session_id"`
		InitialWindows []struct {
			BlockIDs []string `json:"block_ids"`
		} `json:"initial_windows"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return fmt.Errorf("parsing rex session.create: %w", err)
	}
	// Rex activates the last window and focuses the first block in layout
	// order. Focus each window's first pane, last to first, so the session
	// opens on the first window. This is cosmetic, so failures are ignored.
	for i := len(resp.InitialWindows) - 1; i >= 0; i-- {
		ids := resp.InitialWindows[i].BlockIDs
		if i < len(firsts) && firsts[i] < len(ids) {
			_, _ = b.run("focus", "-s", resp.SessionID, ids[firsts[i]])
		}
	}
	return nil
}
