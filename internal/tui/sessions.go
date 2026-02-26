package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gavin-jeong/csb/internal/session"
)

// substringFilter matches items whose FilterValue contains the search term as a substring.
// Supports space-separated multi-term AND matching (e.g., "role=user bash").
func substringFilter(term string, targets []string) []list.Rank {
	terms := strings.Fields(strings.ToLower(term))
	if len(terms) == 0 {
		return nil
	}
	var ranks []list.Rank
	for i, t := range targets {
		lower := strings.ToLower(t)
		allMatch := true
		var firstIdx int
		for ti, tt := range terms {
			idx := strings.Index(lower, tt)
			if idx < 0 {
				allMatch = false
				break
			}
			if ti == 0 {
				firstIdx = idx
			}
		}
		if !allMatch {
			continue
		}
		// Use first term match for highlight indices
		matched := make([]int, len(terms[0]))
		for j := range len(terms[0]) {
			matched[j] = firstIdx + j
		}
		ranks = append(ranks, list.Rank{Index: i, MatchedIndexes: matched})
	}
	return ranks
}

type sessionItem struct {
	sess session.Session
}

func (s sessionItem) FilterValue() string {
	return s.sess.ProjectPath + " " + s.sess.ProjectName + " " + s.sess.GitBranch + " " + s.sess.ShortID + " " + s.sess.FirstPrompt
}

type sessionDelegate struct {
	timeW int // max width of time-ago column
	msgW  int // max width of message count column
	projW int // max width of project name column
}

func (d sessionDelegate) Height() int                             { return 2 }
func (d sessionDelegate) Spacing() int                            { return 0 }
func (d sessionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sessionItem)
	if !ok {
		return
	}

	s := si.sess
	selected := index == m.Index()
	width := m.Width()

	cursor := "  "
	if selected {
		cursor = "> "
	}

	// Aligned columns: ID  TIME  MSG  PROJECT  [badges]
	idStyle := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	timeStyle := dimStyle
	msgStyle := lipgloss.NewStyle().Foreground(colorAccent)
	projStyle := lipgloss.NewStyle()
	branchStyle := dimStyle
	promptStyle := dimStyle
	if selected {
		idStyle = idStyle.Foreground(lipgloss.Color("#A78BFA"))
		timeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
		msgStyle = msgStyle.Bold(true)
		projStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Bold(true)
		branchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
		promptStyle = selectedStyle
	}

	idStr := idStyle.Render(s.ShortID)

	timeRaw := timeAgo(s.ModTime)
	timePad := fmt.Sprintf("%-*s", d.timeW, timeRaw)
	timeStr := timeStyle.Render(timePad)

	msgRaw := fmt.Sprintf("%dm", s.MsgCount)
	msgPad := fmt.Sprintf("%*s", d.msgW, msgRaw)
	msgStr := msgStyle.Render(msgPad)

	// Build badges first to know their width
	badges := ""
	badgesW := 0
	if s.IsLive {
		badges += " " + liveBadge.Render("[LIVE]")
		badgesW += 7
	}
	if s.HasMemory {
		badges += " " + memoryBadge.Render("[mem]")
		badgesW += 6
	}
	if s.IsWorktree {
		badges += " " + worktreeBadge.Render("[wt]")
		badgesW += 5
	}

	// Calculate available width for project column
	// cursor(2) + id(8) + 2 + time + 2 + msg + 2 + project + badges
	fixedW := 2 + 8 + 2 + d.timeW + 2 + d.msgW + 2 + badgesW
	maxProjW := width - fixedW
	if maxProjW < 10 {
		maxProjW = 10
	}

	projPlain := s.ProjectName
	if s.GitBranch != "" {
		projPlain += " (" + s.GitBranch + ")"
	}

	// Truncate project if needed
	if len(projPlain) > maxProjW {
		projPlain = projPlain[:maxProjW-3] + "..."
	}

	// Re-render with possibly truncated text
	projName := s.ProjectName
	branch := ""
	if s.GitBranch != "" {
		branch = " (" + s.GitBranch + ")"
	}
	fullProj := projName + branch
	if len(fullProj) > maxProjW {
		fullProj = fullProj[:maxProjW-3] + "..."
		// Render as single string since it's truncated
		project := projStyle.Render(fullProj)
		projPlain = fullProj
		_ = project
	}

	project := projStyle.Render(projName)
	if len(projName+branch) > maxProjW {
		trunc := (projName + branch)[:maxProjW-3] + "..."
		project = projStyle.Render(trunc)
		projPlain = trunc
	} else if branch != "" {
		project += branchStyle.Render(branch)
	}

	// Pad project to align badges
	if pad := min(d.projW, maxProjW) - len(projPlain); pad > 0 {
		project += strings.Repeat(" ", pad)
	}

	line1 := fmt.Sprintf("%s%s  %s  %s  %s%s", cursor, idStr, timeStr, msgStr, project, badges)

	prompt := s.FirstPrompt
	maxW := width - 6
	if maxW > 0 && len(prompt) > maxW {
		prompt = prompt[:maxW-3] + "..."
	}
	line2 := "    " + promptStyle.Render(prompt)

	if selected {
		// Pad lines to full width for background highlight
		l1Bare := lipgloss.Width(line1)
		if l1Bare < width {
			line1 += strings.Repeat(" ", width-l1Bare)
		}
		l2Bare := lipgloss.Width(line2)
		if l2Bare < width {
			line2 += strings.Repeat(" ", width-l2Bare)
		}
		line1 = selectedRowStyle.Render(line1)
		line2 = selectedRowStyle.Render(line2)
	}

	fmt.Fprintf(w, "%s\n%s", line1, line2)
}

func computeSessionColWidths(sessions []session.Session) (timeW, msgW, projW int) {
	for _, s := range sessions {
		if tw := len(timeAgo(s.ModTime)); tw > timeW {
			timeW = tw
		}
		if mw := len(fmt.Sprintf("%dm", s.MsgCount)); mw > msgW {
			msgW = mw
		}
		pw := len(s.ProjectName)
		if s.GitBranch != "" {
			pw += len(" (") + len(s.GitBranch) + len(")")
		}
		if pw > projW {
			projW = pw
		}
	}
	return
}

func newSessionList(sessions []session.Session, width, height int) list.Model {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{sess: s}
	}

	timeW, msgW, projW := computeSessionColWidths(sessions)

	l := list.New(items, sessionDelegate{timeW: timeW, msgW: msgW, projW: projW}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Filter = substringFilter
	l.DisableQuitKeybindings()
	configureListSearch(&l)
	return l
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 02")
	}
}
