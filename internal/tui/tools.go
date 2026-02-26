package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gavin-jeong/csb/internal/session"
)

type toolCallItem struct {
	toolName  string
	input     string
	timestamp string
	msgIndex  int // index into the full messages list to jump back
	entry     session.Entry
}

func (t toolCallItem) FilterValue() string {
	return t.toolName + " " + t.input
}

type toolDelegate struct{}

func (d toolDelegate) Height() int                             { return 2 }
func (d toolDelegate) Spacing() int                            { return 0 }
func (d toolDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d toolDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(toolCallItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	width := m.Width()

	cursor := "  "
	if selected {
		cursor = "> "
	}

	ts := dimStyle.Render(ti.timestamp)
	name := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(ti.toolName)
	idx := dimStyle.Render(fmt.Sprintf("msg#%d", ti.msgIndex+1))
	line1 := fmt.Sprintf("%s%s  %s  %s", cursor, ts, name, idx)

	input := ti.input
	maxW := width - 6
	if maxW > 3 && len(input) > maxW {
		input = input[:maxW-3] + "..."
	}
	pStyle := dimStyle
	if selected {
		pStyle = selectedStyle
	}
	line2 := "    " + pStyle.Render(input)

	fmt.Fprintf(w, "%s\n%s", line1, line2)
}

func extractToolCalls(messages []session.Entry) []toolCallItem {
	var items []toolCallItem
	for i, e := range messages {
		for _, block := range e.Content {
			if block.Type != "tool_use" {
				continue
			}
			ts := ""
			if !e.Timestamp.IsZero() {
				ts = e.Timestamp.Format("15:04:05")
			}
			input := block.ToolInput
			if input == "" {
				input = "(no input)"
			}
			items = append(items, toolCallItem{
				toolName:  block.ToolName,
				input:     input,
				timestamp: ts,
				msgIndex:  i,
				entry:     e,
			})
		}
	}
	return items
}

func newToolList(items []toolCallItem, width, height int) list.Model {
	listItems := make([]list.Item, len(items))
	for i, t := range items {
		listItems[i] = t
	}

	l := list.New(listItems, toolDelegate{}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	configureListSearch(&l)
	return l
}
