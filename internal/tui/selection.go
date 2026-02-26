package tui

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// configureListSearch customizes list search: changes prompt to "Search: ",
// removes vim j/k from cursor navigation, and prevents arrow/navigation keys
// from closing the search input.
func configureListSearch(l *list.Model) {
	l.FilterInput.Prompt = "Search: "
	l.KeyMap.AcceptWhileFiltering = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "apply"),
	)
	// Arrow-only navigation (remove j/k)
	l.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	)
	l.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	)
}

// isSearchTrigger returns true for printable letter/digit keys that should
// auto-activate the search prompt from a list view.
func isSearchTrigger(k string) bool {
	if len(k) != 1 {
		return false
	}
	c := k[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// startListSearch activates the search prompt and seeds it with the given key.
func startListSearch(l *list.Model, k string) tea.Cmd {
	if l.Width() == 0 {
		return nil
	}
	// Simulate "/" to open filter, then type the letter
	openMsg := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'/'}})
	_, cmd1 := l.Update(openMsg)
	letterMsg := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(k)})
	_, cmd2 := l.Update(letterMsg)
	return tea.Batch(cmd1, cmd2)
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func openInPager(styledContent string) tea.Cmd {
	plain := stripANSI(styledContent)
	tmpFile, err := os.CreateTemp("", "csb-*.txt")
	if err != nil {
		return nil
	}
	tmpFile.WriteString(plain)
	tmpFile.Close()

	c := exec.Command("less", tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(tmpFile.Name())
		return nil
	})
}
