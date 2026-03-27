package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// Item represents a selectable entry in the interactive list.
type Item struct {
	Title  string // display text
	Value  string // actual value to return
	Active bool   // true = running session (show ▶)
}

var (
	ErrCancelled = errors.New("cancelled")

	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	inputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

// ---------- single-select model ----------

type selectModel struct {
	prompt    string
	allItems  []Item
	filtered  []Item
	cursor    int
	input     textinput.Model
	done      bool
	cancelled bool
	selected  Item
}

func newSelectModel(items []Item, prompt string) selectModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter…"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60
	return selectModel{
		prompt:   prompt,
		allItems: items,
		filtered: items,
		input:    ti,
	}
}

func (m selectModel) Init() tea.Cmd { return textinput.Blink }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				m.selected = m.filtered[m.cursor]
				m.done = true
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = filterItems(m.allItems, m.input.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m selectModel) View() string {
	var sb strings.Builder
	sb.WriteString(promptStyle.Render(m.prompt) + "\n")
	sb.WriteString(inputStyle.Render(m.input.View()) + "\n\n")

	limit := 15
	start := 0
	if m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	end := start + limit
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := start; i < end; i++ {
		item := m.filtered[i]
		indicator := "  📁 "
		if item.Active {
			indicator = activeStyle.Render("  ▶  ")
		}
		line := indicator + item.Title
		if i == m.cursor {
			line = selectedStyle.Render("> " + indicator[2:] + item.Title)
		}
		sb.WriteString(line + "\n")
	}
	if len(m.filtered) == 0 {
		sb.WriteString(dimStyle.Render("  (no matches)") + "\n")
	}
	return sb.String()
}

// ---------- multi-select model ----------

type multiModel struct {
	prompt    string
	allItems  []Item
	filtered  []Item
	cursor    int
	selected  map[int]bool // indices into allItems
	input     textinput.Model
	done      bool
	cancelled bool
	result    []Item
}

func newMultiModel(items []Item, prompt string) multiModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter…"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60
	return multiModel{
		prompt:   prompt,
		allItems: items,
		filtered: items,
		selected: map[int]bool{},
		input:    ti,
	}
}

func (m multiModel) Init() tea.Cmd { return textinput.Blink }

func (m multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			for _, item := range m.allItems {
				// find by value
				for idx, orig := range m.allItems {
					if m.selected[idx] && orig.Value == item.Value {
						m.result = append(m.result, orig)
						break
					}
				}
			}
			m.done = true
			return m, tea.Quit
		case "tab", " ":
			if len(m.filtered) > 0 {
				cur := m.filtered[m.cursor]
				for idx, orig := range m.allItems {
					if orig.Value == cur.Value {
						m.selected[idx] = !m.selected[idx]
						break
					}
				}
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = filterItems(m.allItems, m.input.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m multiModel) View() string {
	var sb strings.Builder
	sb.WriteString(
		promptStyle.Render(m.prompt) +
			dimStyle.Render(" (Tab/Space to select, Enter to confirm)") + "\n",
	)
	sb.WriteString(inputStyle.Render(m.input.View()) + "\n\n")

	limit := 15
	start := 0
	if m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	end := start + limit
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := start; i < end; i++ {
		item := m.filtered[i]
		// find original index for selected state
		isSelected := false
		for idx, orig := range m.allItems {
			if orig.Value == item.Value {
				isSelected = m.selected[idx]
				break
			}
		}
		check := "[ ]"
		if isSelected {
			check = "[x]"
		}
		indicator := "  📁 "
		if item.Active {
			indicator = activeStyle.Render("  ▶  ")
		}
		line := fmt.Sprintf("%s %s%s", check, indicator, item.Title)
		if i == m.cursor {
			line = selectedStyle.Render("> " + line[2:])
		}
		sb.WriteString(line + "\n")
	}
	if len(m.filtered) == 0 {
		sb.WriteString(dimStyle.Render("  (no matches)") + "\n")
	}
	return sb.String()
}

// ---------- spinner model ----------

type spinnerMsg struct{}
type doneMsg struct{ err error }

type spinnerModel struct {
	label   string
	frames  []string
	frame   int
	done    bool
	err     error
	fn      func() error
	started bool
}

func newSpinnerModel(label string, fn func() error) spinnerModel {
	return spinnerModel{
		label:  label,
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		fn:     fn,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			err := m.fn()
			return doneMsg{err}
		},
		tick(),
	)
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return spinnerMsg{}
	})
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinnerMsg:
		m.frame = (m.frame + 1) % len(m.frames)
		return m, tick()
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s\n", m.frames[m.frame], m.label)
}

// ---------- public API ----------

// Select shows an interactive list and returns the selected item.
func Select(items []Item, prompt string) (Item, error) {
	if len(items) == 0 {
		return Item{}, fmt.Errorf("no items to select")
	}
	m := newSelectModel(items, prompt)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return Item{}, err
	}
	final := result.(selectModel)
	if final.cancelled {
		return Item{}, ErrCancelled
	}
	return final.selected, nil
}

// MultiSelect shows a list with Tab to select multiple, Enter to confirm.
func MultiSelect(items []Item, prompt string) ([]Item, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select")
	}
	m := newMultiModel(items, prompt)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(multiModel)
	if final.cancelled {
		return nil, ErrCancelled
	}
	return final.result, nil
}

// Spinner runs fn() while showing a spinner with label.
func Spinner(label string, fn func() error) error {
	m := newSpinnerModel(label, fn)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return err
	}
	return result.(spinnerModel).err
}

// ---------- helpers ----------

func filterItems(items []Item, query string) []Item {
	if query == "" {
		return items
	}
	titles := make([]string, len(items))
	for i, item := range items {
		titles[i] = item.Title
	}
	matches := fuzzy.Find(query, titles)
	result := make([]Item, 0, len(matches))
	for _, m := range matches {
		result = append(result, items[m.Index])
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}