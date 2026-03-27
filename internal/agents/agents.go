package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pean/twine/internal/tmux"
)

// AgentType identifies the AI coding agent.
type AgentType string

const (
	AgentClaude   AgentType = "Claude Code"
	AgentOpenCode AgentType = "OpenCode"
)

// AgentState represents the current activity state of an agent.
type AgentState string

const (
	StateWorking AgentState = "⚡Working"
	StateInput   AgentState = "❗Input"
	StateIdle    AgentState = "⏸ Idle"
	StateNew     AgentState = "🆕 New"
)

// Agent holds information about a discovered AI coding agent.
type Agent struct {
	Type      AgentType
	State     AgentState
	Session   string
	Directory string
	Model     string
	PaneID    string
}

type claudeSession struct {
	Model     string `json:"model"`
	Directory string `json:"directory"`
}

// FindAll discovers all running AI agents across Claude Code and OpenCode.
func FindAll() ([]Agent, error) {
	var result []Agent

	claude, err := findClaudeAgents()
	if err == nil {
		result = append(result, claude...)
	}

	opencode, err := findOpenCodeAgents()
	if err == nil {
		result = append(result, opencode...)
	}

	return result, nil
}

func findClaudeAgents() ([]Agent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sessionDir := filepath.Join(home, ".claude", "sessions")
	entries, err := filepath.Glob(filepath.Join(sessionDir, "*.json"))
	if err != nil || len(entries) == 0 {
		return nil, nil
	}

	panes, err := tmux.ListPanes()
	if err != nil {
		return nil, err
	}
	// Build a set of pane PIDs for quick lookup.
	panePIDs := make(map[int]tmux.PaneInfo, len(panes))
	for _, p := range panes {
		panePIDs[p.PID] = p
	}

	var result []Agent
	for _, entry := range entries {
		base := filepath.Base(entry)
		pidStr := strings.TrimSuffix(base, ".json")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		pane, found := walkToPanePID(pid, panePIDs, 10)
		if !found {
			continue
		}

		var sess claudeSession
		if data, err := os.ReadFile(entry); err == nil {
			_ = json.Unmarshal(data, &sess)
		}

		dir := pane.Path
		if sess.Directory != "" {
			dir = sess.Directory
		}
		model := sess.Model
		if model == "" {
			model = "unknown"
		}

		content, _ := tmux.CapturePaneContent(pane.PaneID, 10)
		state := detectClaudeState(content)

		shortDir := shortenPath(dir, home)
		result = append(result, Agent{
			Type:      AgentClaude,
			State:     state,
			Session:   pane.Session,
			Directory: shortDir,
			Model:     model,
			PaneID:    pane.PaneID,
		})
	}
	return result, nil
}

// walkToPanePID walks the PPID chain up to maxDepth levels from pid until it
// finds a process whose PID matches a tmux pane_pid.
func walkToPanePID(
	pid int,
	panePIDs map[int]tmux.PaneInfo,
	maxDepth int,
) (tmux.PaneInfo, bool) {
	current := pid
	for i := 0; i < maxDepth; i++ {
		if pane, ok := panePIDs[current]; ok {
			return pane, true
		}
		ppid, err := ppidOf(current)
		if err != nil || ppid <= 1 {
			break
		}
		current = ppid
	}
	return tmux.PaneInfo{}, false
}

// ppidOf returns the parent PID of the given process using ps.
func ppidOf(pid int) (int, error) {
	out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

func findOpenCodeAgents() ([]Agent, error) {
	panes, err := tmux.ListPanes()
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	var result []Agent
	for _, p := range panes {
		if !strings.EqualFold(p.Command, "opencode") {
			continue
		}
		content, _ := tmux.CapturePaneContent(p.PaneID, 20)
		state := detectOpenCodeState(content)
		result = append(result, Agent{
			Type:      AgentOpenCode,
			State:     state,
			Session:   p.Session,
			Directory: shortenPath(p.Path, home),
			Model:     "-",
			PaneID:    p.PaneID,
		})
	}
	return result, nil
}

func detectClaudeState(content string) AgentState {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "esc to interrupt"):
		return StateWorking
	case strings.Contains(content, "Esc to cancel"):
		return StateInput
	case strings.Contains(content, "Enter to submit"):
		return StateIdle
	case strings.Contains(content, "Welcome to Claude Code"):
		return StateNew
	default:
		return StateIdle
	}
}

func detectOpenCodeState(content string) AgentState {
	switch {
	case strings.Contains(content, "Running") ||
		strings.Contains(content, "Executing"):
		return StateWorking
	case strings.Contains(content, "waiting") ||
		strings.Contains(content, "Approve"):
		return StateInput
	case strings.Contains(content, "What can I help") ||
		strings.Contains(content, "Enter your"):
		return StateIdle
	default:
		return StateIdle
	}
}

func shortenPath(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// ColorizeState returns an ANSI-colored state string for terminal display.
func ColorizeState(state AgentState) string {
	switch state {
	case StateWorking:
		return fmt.Sprintf("\033[33m%s\033[0m", state)
	case StateInput:
		return fmt.Sprintf("\033[1;35m%s\033[0m", state)
	case StateIdle:
		return fmt.Sprintf("\033[32m%s\033[0m", state)
	case StateNew:
		return fmt.Sprintf("\033[36m%s\033[0m", state)
	default:
		return string(state)
	}
}