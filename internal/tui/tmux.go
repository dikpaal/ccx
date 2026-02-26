package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type tmuxPane struct {
	PaneID  string
	Command string
	Session string
	Window  string
	Pane    string
	PID     int
	Path    string
}

func inTmux() bool {
	return os.Getenv("TMUX") != ""
}

// findTmuxPane finds the tmux pane whose cwd matches projectPath
// and (optionally) has a claude process running in it.
func findTmuxPane(projectPath string) (tmuxPane, bool) {
	if !inTmux() || projectPath == "" {
		return tmuxPane{}, false
	}

	panes, err := listTmuxPanes()
	if err != nil || len(panes) == 0 {
		return tmuxPane{}, false
	}

	absProject, _ := filepath.Abs(projectPath)
	if absProject == "" {
		absProject = projectPath
	}

	// First pass: match by path AND has claude child process
	for _, p := range panes {
		absPane, _ := filepath.Abs(p.Path)
		if absPane == "" {
			absPane = p.Path
		}
		if absPane == absProject && hasClaude(p.PID) {
			return p, true
		}
	}

	// Second pass: match by path only (claude might show differently)
	for _, p := range panes {
		absPane, _ := filepath.Abs(p.Path)
		if absPane == "" {
			absPane = p.Path
		}
		if absPane == absProject {
			return p, true
		}
	}

	return tmuxPane{}, false
}

func listTmuxPanes() ([]tmuxPane, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{pane_id}|#{pane_current_command}|#{session_name}|#{window_index}|#{pane_index}|#{pane_pid}|#{pane_current_path}",
	).Output()
	if err != nil {
		return nil, err
	}

	var panes []tmuxPane
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 7 {
			continue
		}
		pid, _ := strconv.Atoi(parts[5])
		panes = append(panes, tmuxPane{
			PaneID:  parts[0],
			Command: parts[1],
			Session: parts[2],
			Window:  parts[3],
			Pane:    parts[4],
			PID:     pid,
			Path:    parts[6],
		})
	}
	return panes, nil
}

// hasClaude checks if a pane's shell has a claude child process.
func hasClaude(shellPID int) bool {
	if shellPID == 0 {
		return false
	}
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(shellPID), "-f", "claude").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func switchToTmuxPane(p tmuxPane) error {
	target := p.Session + ":" + p.Window + "." + p.Pane
	// Select the window first (in case it's in a different tmux window)
	exec.Command("tmux", "select-window", "-t", p.Session+":"+p.Window).Run()
	return exec.Command("tmux", "select-pane", "-t", target).Run()
}

// moveWithAndSwitchPane moves the current pane (CSB) to the target's tmux window
// as a side-by-side split, then focuses the target pane.
func moveWithAndSwitchPane(target tmuxPane) error {
	out, err := exec.Command("tmux", "display-message", "-p",
		"#{pane_id}|#{session_name}:#{window_index}").Output()
	if err != nil {
		return switchToTmuxPane(target)
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	if len(parts) < 2 {
		return switchToTmuxPane(target)
	}
	myPaneID := parts[0]
	myWindow := parts[1]
	targetWindow := target.Session + ":" + target.Window

	// Select target window first
	exec.Command("tmux", "select-window", "-t", targetWindow).Run()

	// Move CSB pane to target window if in a different window
	if myWindow != targetWindow {
		exec.Command("tmux", "join-pane",
			"-s", myPaneID, "-t", target.PaneID,
			"-h", "-l", "30%").Run()
	}

	// Focus the target pane
	return exec.Command("tmux", "select-pane", "-t", target.PaneID).Run()
}
