package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary      = lipgloss.Color("#7C3AED")
	colorDim          = lipgloss.Color("#6B7280")
	colorAccent       = lipgloss.Color("#10B981")
	colorUser         = lipgloss.Color("#3B82F6")
	colorAssistant    = lipgloss.Color("#F59E0B")
	colorError        = lipgloss.Color("#EF4444")
	colorWorktree     = lipgloss.Color("#8B5CF6")
	colorFilter       = lipgloss.Color("#EC4899")
	colorBorderFocused = lipgloss.Color("#38BDF8")
	colorBorderDim     = lipgloss.Color("#374151")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(colorDim)

	userLabelStyle      = lipgloss.NewStyle().Foreground(colorUser).Bold(true)
	assistantLabelStyle = lipgloss.NewStyle().Foreground(colorAssistant).Bold(true)
	toolStyle           = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	toolBlockStyle      = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	errorStyle          = lipgloss.NewStyle().Foreground(colorError)
	dimStyle            = lipgloss.NewStyle().Foreground(colorDim)
	selectedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	selectedRowStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#1E293B"))
	worktreeBadge       = lipgloss.NewStyle().Foreground(colorWorktree).Bold(true)
	filterBadge         = lipgloss.NewStyle().Foreground(colorFilter).Bold(true)
	agentBadgeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Bold(true)
	compactBadgeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	mcpBadgeStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#F472B6")).Bold(true)
	memoryBadge         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true)
	liveBadge           = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true)
	blockCursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8")).Bold(true)
	previewBorder       = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(colorDim)
)
