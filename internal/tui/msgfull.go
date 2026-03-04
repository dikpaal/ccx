package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/keyolk/ccx/internal/session"
)

// navFrame stores state for navigating back from an agent drill-down.
type navFrame struct {
	sess     session.Session
	messages []session.Entry
	merged   []mergedMsg
	agents   []session.Subagent
	items    []convItem
	listIdx  int // cursor position to restore
	agent    session.Subagent
	fromView viewState // which view pushed this frame
}

// openMsgFullForEntry opens viewMessageFull for a specific merged message.
func (a *App) openMsgFullForEntry(m mergedMsg) (tea.Model, tea.Cmd) {
	a.msgFull.sess = a.currentSess
	a.msgFull.agent = session.Subagent{}
	a.msgFull.messages = a.conv.messages
	a.msgFull.merged = a.conv.merged
	a.msgFull.agents = a.conv.agents

	// Find the index of this merged message
	idx := 0
	for i, mm := range a.msgFull.merged {
		if mm.startIdx == m.startIdx {
			idx = i
			break
		}
	}
	a.msgFull.idx = idx
	a.navToMsgFull(idx)
	a.state = viewMessageFull
	return a, nil
}

// navToMsgFull sets up the viewport and fold state for message at index.
func (a *App) navToMsgFull(idx int) {
	if idx < 0 || idx >= len(a.msgFull.merged) {
		return
	}
	a.msgFull.idx = idx
	entry := a.msgFull.merged[idx].entry

	a.msgFull.folds = FoldState{}
	a.msgFull.folds.Reset(entry)

	contentH := ContentHeight(a.height)
	content := renderFullMessage(entry, a.width)
	a.msgFull.content = content

	a.msgFull.vp = viewport.New(a.width, contentH)

	// Render with block cursor
	rp := renderFullMessageWithCursor(entry, a.width, a.msgFull.folds.Collapsed, a.msgFull.folds.Formatted, a.msgFull.folds.BlockCursor)
	a.msgFull.folds.BlockStarts = rp.blockStarts
	a.msgFull.vp.SetContent(rp.content)
}

// handleMessageFullKeys handles keyboard input for viewMessageFull.
func (a *App) handleMessageFullKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Copy mode intercepts all keys
	if a.copyModeActive {
		return a.handleCopyModeKeys(msg)
	}

	key := msg.String()

	switch key {
	case "q":
		return a, tea.Quit
	case "esc":
		return a.popNavFrame()
	case "v":
		a.enterMsgFullCopyMode()
		return a, nil
	case "y":
		copyToClipboard(stripANSI(a.msgFull.content))
		a.copiedMsg = "Copied!"
		return a, nil
	case "o":
		openInPager(stripANSI(a.msgFull.content))
		return a, nil
	}

	// allMessages mode: only scroll, no blocks/folds/navigation
	if a.msgFull.allMessages {
		switch key {
		case "up", "down", "pgup", "pgdown", "home", "end":
			scrollPreview(&a.msgFull.vp, key)
		}
		return a, nil
	}

	switch key {
	case "n", "]":
		// Next message
		if a.msgFull.idx < len(a.msgFull.merged)-1 {
			a.navToMsgFull(a.msgFull.idx + 1)
		}
		return a, nil
	case "N", "[":
		// Previous message
		if a.msgFull.idx > 0 {
			a.navToMsgFull(a.msgFull.idx - 1)
		}
		return a, nil
	case "enter":
		// On Agent block: recursive drill-down
		fs := &a.msgFull.folds
		if fs.BlockCursor >= 0 && fs.BlockCursor < len(fs.Entry.Content) {
			block := fs.Entry.Content[fs.BlockCursor]
			if block.Type == "tool_use" && block.ToolName == "Task" {
				if agent, found := a.findAgentInMsgFull(fs.Entry); found {
					a.pushMsgFullFrame()
					return a.openAgentConversation(agent)
				}
			}
		}
		return a, nil
	}

	// Fold navigation
	fs := &a.msgFull.folds
	fr := fs.HandleKey(key)
	if fr == foldCursorMoved || fr == foldHandled {
		a.refreshMsgFullPreview()
		return a, nil
	}

	// Scroll viewport
	switch key {
	case "up", "down", "pgup", "pgdown", "home", "end":
		scrollPreview(&a.msgFull.vp, key)
		return a, nil
	}

	return a, nil
}

// refreshMsgFullPreview re-renders the message full viewport.
func (a *App) refreshMsgFullPreview() {
	fs := &a.msgFull.folds
	rp := renderFullMessageWithCursor(fs.Entry, a.width, fs.Collapsed, fs.Formatted, fs.BlockCursor)
	fs.BlockStarts = rp.blockStarts

	oldOffset := a.msgFull.vp.YOffset
	a.msgFull.vp.SetContent(rp.content)

	maxOffset := max(a.msgFull.vp.TotalLineCount()-a.msgFull.vp.Height, 0)
	if oldOffset > maxOffset {
		oldOffset = maxOffset
	}
	a.msgFull.vp.YOffset = oldOffset

	// Scroll to keep block cursor visible
	if fs.BlockCursor >= 0 && fs.BlockCursor < len(fs.BlockStarts) {
		blockLine := fs.BlockStarts[fs.BlockCursor]
		if blockLine < a.msgFull.vp.YOffset {
			a.msgFull.vp.YOffset = max(blockLine-1, 0)
		} else if blockLine >= a.msgFull.vp.YOffset+a.msgFull.vp.Height {
			a.msgFull.vp.YOffset = min(blockLine-a.msgFull.vp.Height/2, maxOffset)
		}
	}
}

// renderMessageFull renders the full-screen message detail view.
func (a *App) renderMessageFull() string {
	// Clamp YOffset to prevent out-of-bounds panic after content change
	maxOffset := max(a.msgFull.vp.TotalLineCount()-a.msgFull.vp.Height, 0)
	if a.msgFull.vp.YOffset > maxOffset {
		a.msgFull.vp.YOffset = maxOffset
	}
	return a.msgFull.vp.View()
}

// pushNavFrame saves current conversation state onto the nav stack.
func (a *App) pushNavFrame() {
	frame := navFrame{
		sess:     a.conv.sess,
		messages: a.conv.messages,
		merged:   a.conv.merged,
		agents:   a.conv.agents,
		items:    a.conv.items,
		listIdx:  a.convList.Index(),
		agent:    a.conv.agent,
		fromView: a.state,
	}
	a.navStack = append(a.navStack, frame)
}

// pushMsgFullFrame saves current msgFull state for recursive agent drill-down.
func (a *App) pushMsgFullFrame() {
	// Store the current msgFull context as a nav frame
	frame := navFrame{
		sess:     a.msgFull.sess,
		messages: a.msgFull.messages,
		merged:   a.msgFull.merged,
		agents:   a.msgFull.agents,
		listIdx:  a.msgFull.idx,
		fromView: viewMessageFull,
	}
	a.navStack = append(a.navStack, frame)
}

// popNavFrame restores the previous view from the nav stack.
func (a *App) popNavFrame() (tea.Model, tea.Cmd) {
	if len(a.navStack) == 0 {
		// No stack: go back to conversation or sessions
		if a.state == viewMessageFull {
			a.msgFull.allMessages = false
			a.state = viewConversation
			a.updateConvPreview()
			return a, nil
		}
		// viewConversation with agent → back to sessions
		a.conv.agent = session.Subagent{}
		a.state = viewSessions
		return a, nil
	}

	frame := a.navStack[len(a.navStack)-1]
	a.navStack = a.navStack[:len(a.navStack)-1]

	// Restore to conversation view
	if frame.fromView == viewConversation {
		a.conv.sess = frame.sess
		a.conv.messages = frame.messages
		a.conv.merged = frame.merged
		a.conv.agents = frame.agents
		a.conv.items = frame.items
		a.conv.agent = frame.agent
		a.msgFull.allMessages = false

		contentH := ContentHeight(a.height)
		a.convList = newConvList(a.conv.items, a.conv.split.ListWidth(a.width, a.splitRatio), contentH)
		if frame.listIdx < len(a.conv.items) {
			a.convList.Select(frame.listIdx)
		}
		a.conv.split.CacheKey = ""
		a.state = viewConversation
		a.updateConvPreview()
		return a, nil
	}

	// Pop back to parent msgFull (recursive agent)
	a.msgFull.sess = frame.sess
	a.msgFull.messages = frame.messages
	a.msgFull.merged = frame.merged
	a.msgFull.agents = frame.agents
	a.msgFull.allMessages = false
	a.navToMsgFull(frame.listIdx)
	a.state = viewMessageFull
	return a, nil
}

// findAgentInMsgFull finds the agent matching a Task tool_use in the current msgFull context.
func (a *App) findAgentInMsgFull(entry session.Entry) (session.Subagent, bool) {
	agents := a.msgFull.agents
	if len(agents) == 0 {
		return session.Subagent{}, false
	}
	if entry.Timestamp.IsZero() {
		return session.Subagent{}, false
	}

	var best session.Subagent
	bestDiff := float64(1e18)
	for _, ag := range agents {
		if ag.Timestamp.IsZero() {
			continue
		}
		diff := ag.Timestamp.Sub(entry.Timestamp).Seconds()
		if diff >= -5 && diff < 60 {
			absDiff := diff
			if absDiff < 0 {
				absDiff = -absDiff
			}
			if absDiff < bestDiff {
				bestDiff = absDiff
				best = ag
			}
		}
	}
	if bestDiff < 1e18 {
		return best, true
	}
	return session.Subagent{}, false
}

// enterMsgFullCopyMode enters copy mode for the message full view.
func (a *App) enterMsgFullCopyMode() {
	a.copyLines = strings.Split(stripANSI(a.msgFull.content), "\n")
	a.copyModeActive = true
	a.copyCursor = a.msgFull.vp.YOffset
	a.copyAnchor = -1
	a.renderMsgFullCopyMode()
}

// renderMsgFullCopyMode renders copy mode overlay on msgFull viewport.
func (a *App) renderMsgFullCopyMode() {
	offset := a.msgFull.vp.YOffset
	selStart, selEnd := a.copySelRange()

	var sb strings.Builder
	for i, line := range a.copyLines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		padded := line + strings.Repeat(" ", max(a.width-lipgloss.Width(line), 0))
		if a.copyAnchor != -1 && i >= selStart && i <= selEnd {
			sb.WriteString(selectBg.Render(padded))
		} else if i == a.copyCursor {
			sb.WriteString(cursorBg.Render(padded))
		} else {
			sb.WriteString(line)
		}
	}

	a.msgFull.vp.SetContent(sb.String())
	a.msgFull.vp.YOffset = offset
}

// msgFullBreadcrumb returns the breadcrumb suffix for the message full view.
func (a *App) msgFullBreadcrumb() string {
	if a.msgFull.agent.ShortID != "" {
		return fmt.Sprintf(" > agent:%s > #%d/%d",
			a.msgFull.agent.ShortID,
			a.msgFull.idx+1,
			len(a.msgFull.merged))
	}
	m := a.msgFull.merged[a.msgFull.idx]
	return fmt.Sprintf(" > #%d %s", m.startIdx+1, strings.ToUpper(m.entry.Role))
}
