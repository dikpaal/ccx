package tui

import (
	"fmt"
	"strings"

	"github.com/gavin-jeong/csb/internal/session"
)

// mergedMsg represents a logical conversation turn, potentially combining
// multiple raw API entries into one.
type mergedMsg struct {
	entry    session.Entry
	startIdx int // first original entry index (0-based)
	endIdx   int // last original entry index (0-based)
}

// mergeConversationTurns groups raw entries into logical conversation turns.
// An assistant turn includes all consecutive assistant messages plus any
// interleaved user messages that contain only tool_result blocks (automated
// tool responses). User messages with actual text content stay standalone.
func mergeConversationTurns(entries []session.Entry) []mergedMsg {
	if len(entries) == 0 {
		return nil
	}

	var result []mergedMsg
	i := 0

	for i < len(entries) {
		e := entries[i]

		// User message with actual text → standalone turn
		if e.Role == "user" && hasUserText(e) {
			result = append(result, mergedMsg{
				entry:    e,
				startIdx: i,
				endIdx:   i,
			})
			i++
			continue
		}

		// Assistant message → start a turn, absorb consecutive messages
		// until the next user message with text
		if e.Role == "assistant" {
			merged := mergedMsg{
				entry:    cloneEntry(e),
				startIdx: i,
				endIdx:   i,
			}
			j := i + 1
			for j < len(entries) {
				next := entries[j]
				if next.Role == "user" && hasUserText(next) {
					break
				}
				merged.entry.Content = append(merged.entry.Content, next.Content...)
				merged.endIdx = j
				j++
			}
			result = append(result, merged)
			i = j
			continue
		}

		// Orphan user tool_result (no preceding assistant)
		result = append(result, mergedMsg{
			entry:    e,
			startIdx: i,
			endIdx:   i,
		})
		i++
	}

	return result
}

// hasUserText returns true if the entry has at least one non-empty text block.
func hasUserText(e session.Entry) bool {
	for _, block := range e.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return true
		}
	}
	return false
}

// cloneEntry creates a shallow copy of an Entry with its own Content slice.
func cloneEntry(e session.Entry) session.Entry {
	clone := e
	clone.Content = make([]session.ContentBlock, len(e.Content))
	copy(clone.Content, e.Content)
	return clone
}

// filterMerged applies a filter mode to merged messages.
func filterMerged(msgs []mergedMsg, mode filterMode) []mergedMsg {
	switch mode {
	case filterNone:
		return msgs
	case filterSummary:
		return summarizeMerged(msgs)
	default:
		var result []mergedMsg
		for _, m := range msgs {
			if mergedMatchesFilter(m.entry, mode) {
				result = append(result, m)
			}
		}
		return result
	}
}

func mergedMatchesFilter(e session.Entry, mode filterMode) bool {
	switch mode {
	case filterUser:
		return e.Role == "user"
	case filterAssistant:
		return e.Role == "assistant"
	case filterToolCalls:
		for _, b := range e.Content {
			if b.Type == "tool_use" {
				return true
			}
		}
		return false
	case filterAgents:
		for _, b := range e.Content {
			if b.Type == "tool_use" && b.ToolName == "Task" {
				return true
			}
		}
		return false
	case filterSkills:
		for _, b := range e.Content {
			if b.Type == "tool_use" && b.ToolName == "Skill" {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// summarizeMerged keeps user messages + the last assistant turn before each user.
func summarizeMerged(msgs []mergedMsg) []mergedMsg {
	if len(msgs) == 0 {
		return nil
	}
	var result []mergedMsg
	var lastAsst *mergedMsg
	for i := range msgs {
		m := msgs[i]
		if m.entry.Role == "user" {
			if lastAsst != nil {
				result = append(result, *lastAsst)
				lastAsst = nil
			}
			result = append(result, m)
		} else {
			msg := m
			lastAsst = &msg
		}
	}
	if lastAsst != nil {
		result = append(result, *lastAsst)
	}
	return result
}

// reverseMerged reverses a slice of mergedMsg in place.
func reverseMerged(msgs []mergedMsg) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

// mergedToolSummary returns a compact tool summary with counts for duplicates.
// e.g. "[Bash, Read×3, Edit]"
func mergedToolSummary(e session.Entry) string {
	seen := make(map[string]int)
	var order []string
	for _, block := range e.Content {
		if block.Type == "tool_use" {
			if seen[block.ToolName] == 0 {
				order = append(order, block.ToolName)
			}
			seen[block.ToolName]++
		}
	}
	if len(order) == 0 {
		return ""
	}
	var parts []string
	for _, name := range order {
		if seen[name] > 1 {
			parts = append(parts, fmt.Sprintf("%s×%d", name, seen[name]))
		} else {
			parts = append(parts, name)
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
