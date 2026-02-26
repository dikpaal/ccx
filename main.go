package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gavin-jeong/csb/internal/session"
	"github.com/gavin-jeong/csb/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("csb", version)
		os.Exit(0)
	}

	sessions, err := session.ScanSessions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning sessions: %v\n", err)
		os.Exit(1)
	}

	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "No sessions found in ~/.claude/projects/")
		os.Exit(0)
	}

	app := tui.NewApp(sessions)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
