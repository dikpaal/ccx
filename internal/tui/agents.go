package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gavin-jeong/csb/internal/session"
)

type agentItem struct {
	agent session.Subagent
}

func (a agentItem) FilterValue() string {
	return a.agent.FirstPrompt + " " + a.agent.ShortID + " " + a.agent.AgentType
}

type agentDelegate struct{}

func (d agentDelegate) Height() int                             { return 2 }
func (d agentDelegate) Spacing() int                            { return 0 }
func (d agentDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d agentDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ai, ok := item.(agentItem)
	if !ok {
		return
	}

	a := ai.agent
	selected := index == m.Index()
	width := m.Width()

	cursor := "  "
	if selected {
		cursor = "> "
	}

	idStr := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(a.ShortID)
	ts := ""
	if !a.Timestamp.IsZero() {
		ts = dimStyle.Render(a.Timestamp.Format("15:04:05"))
	}
	msgStr := lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%dm", a.MsgCount))

	typeStr := ""
	if a.AgentType != "" {
		typeStr = "  " + dimStyle.Render(a.AgentType)
	}

	line1 := fmt.Sprintf("%s%s  %s  %s%s", cursor, idStr, ts, msgStr, typeStr)

	prompt := a.FirstPrompt
	maxW := width - 6
	if maxW > 3 && len(prompt) > maxW {
		prompt = prompt[:maxW-3] + "..."
	}
	pStyle := dimStyle
	if selected {
		pStyle = selectedStyle
	}
	line2 := "    " + pStyle.Render(prompt)

	fmt.Fprintf(w, "%s\n%s", line1, line2)
}

func newAgentList(agents []session.Subagent, width, height int) list.Model {
	items := make([]list.Item, len(agents))
	for i, a := range agents {
		items[i] = agentItem{agent: a}
	}

	l := list.New(items, agentDelegate{}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Filter = substringFilter
	l.DisableQuitKeybindings()
	configureListSearch(&l)
	return l
}
