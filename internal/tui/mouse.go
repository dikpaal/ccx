package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const mouseScrollLines = 3

func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// In copy mode, only allow scroll on the detail viewport
	if a.copyModeActive {
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			if vp := a.activeDetailVP(); vp != nil {
				mouseScrollVP(vp, msg.Button == tea.MouseButtonWheelUp)
			}
		}
		return a, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		return a.handleMouseScroll(msg)
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			return a.handleMouseClick(msg)
		}
	}
	return a, nil
}

func (a *App) handleMouseScroll(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	up := msg.Button == tea.MouseButtonWheelUp

	switch a.state {
	case viewSessions:
		a.sessSplit.HandleMouseScroll(msg.X, up, a.width, a.splitRatio)
		if a.sessSplit.Show && msg.X > a.sessSplit.ListWidth(a.width, a.splitRatio) {
			a.sessPreviewPinned = !a.sessPreviewAtBottom()
		} else {
			a.updateSessionPreview()
		}

	case viewMessages:
		a.msgSplit.HandleMouseScroll(msg.X, up, a.width, a.splitRatio)
		if !(a.msgSplit.Show && msg.X > a.msgSplit.ListWidth(a.width, a.splitRatio)) {
			if a.msgSplit.Show {
				a.updateMsgPreview(&a.msgSplit)
			}
		}

	case viewDetail:
		mouseScrollVP(&a.detailVP, up)

	case viewAgents:
		a.agentSplit.HandleMouseScroll(msg.X, up, a.width, a.splitRatio)
		if !(a.agentSplit.Show && msg.X > a.agentSplit.ListWidth(a.width, a.splitRatio)) {
			if a.agentSplit.Show {
				a.updateAgentPreview()
			}
		}

	case viewAgentMessages:
		a.agentMsgSplit.HandleMouseScroll(msg.X, up, a.width, a.splitRatio)
		if !(a.agentMsgSplit.Show && msg.X > a.agentMsgSplit.ListWidth(a.width, a.splitRatio)) {
			if a.agentMsgSplit.Show {
				a.updateMsgPreview(&a.agentMsgSplit)
			}
		}

	case viewAgentDetail:
		mouseScrollVP(&a.agentDetailVP, up)

	case viewToolCalls:
		a.toolSplit.HandleMouseScroll(msg.X, up, a.width, a.splitRatio)
	}

	return a, nil
}

func (a *App) handleMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Ignore clicks on title bar (Y=0) and help bar (Y>=height-1)
	if msg.Y == 0 || msg.Y >= a.height-1 {
		return a, nil
	}

	contentY := msg.Y - 1 // adjust for title bar

	switch a.state {
	case viewSessions:
		a.sessSplit.HandleMouseClick(msg.X, contentY, a.width, a.splitRatio)
		a.updateSessionPreview()

	case viewMessages:
		a.msgSplit.HandleMouseClick(msg.X, contentY, a.width, a.splitRatio)
		if a.msgSplit.Show {
			a.updateMsgPreview(&a.msgSplit)
		}

	case viewAgents:
		a.agentSplit.HandleMouseClick(msg.X, contentY, a.width, a.splitRatio)
		if a.agentSplit.Show {
			a.updateAgentPreview()
		}

	case viewAgentMessages:
		a.agentMsgSplit.HandleMouseClick(msg.X, contentY, a.width, a.splitRatio)
		if a.agentMsgSplit.Show {
			a.updateMsgPreview(&a.agentMsgSplit)
		}

	case viewToolCalls:
		a.toolSplit.HandleMouseClick(msg.X, contentY, a.width, a.splitRatio)
	}

	return a, nil
}

// mouseScrollVP scrolls a viewport by mouseScrollLines.
func mouseScrollVP(vp *viewport.Model, up bool) {
	if up {
		vp.ScrollUp(mouseScrollLines)
	} else {
		vp.ScrollDown(mouseScrollLines)
	}
}

// mouseScrollList scrolls a list by simulating up/down key presses.
// This correctly handles filtering, pagination, and all list states.
func mouseScrollList(l *list.Model, up bool) {
	keyType := tea.KeyDown
	if up {
		keyType = tea.KeyUp
	}
	msg := tea.KeyMsg{Type: keyType}
	for i := 0; i < mouseScrollLines; i++ {
		*l, _ = l.Update(msg)
	}
}

// mouseClickList selects the list item at the given content Y position.
func mouseClickList(l *list.Model, contentY int) {
	if contentY < 0 || len(l.Items()) == 0 {
		return
	}
	// Skip during active filtering (indices don't map cleanly)
	if l.FilterState() == list.Filtering || l.FilterState() == list.FilterApplied {
		return
	}

	itemHeight := 2 // delegate Height() + Spacing()
	clickOffset := contentY / itemHeight
	pageStart := l.Paginator.Page * l.Paginator.PerPage
	target := pageStart + clickOffset
	if target >= 0 && target < len(l.Items()) {
		l.Select(target)
	}
}
