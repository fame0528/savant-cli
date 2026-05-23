package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ChatItem represents a single rendered chat message with caching.
type ChatItem struct {
	role      string
	content   string
	tool      string
	timestamp time.Time
	finished  bool
	expanded  bool // for collapsible tool output

	cacheMu     sync.RWMutex
	cachedWidth int
	cachedLines []string
}

// NewChatItem creates a new chat item.
func NewChatItem(role, content, tool string, timestamp time.Time) *ChatItem {
	return &ChatItem{
		role:      role,
		content:   content,
		tool:      tool,
		timestamp: timestamp,
		finished:  true,
	}
}

// SetContent updates the content (for streaming messages).
func (ci *ChatItem) SetContent(content string) {
	ci.cacheMu.Lock()
	defer ci.cacheMu.Unlock()
	ci.content = content
	ci.cachedLines = nil
}

// MarkFinished freezes the rendered output.
func (ci *ChatItem) MarkFinished() {
	ci.finished = true
}

// ToggleExpanded toggles collapsible tool output.
func (ci *ChatItem) ToggleExpanded() {
	ci.cacheMu.Lock()
	defer ci.cacheMu.Unlock()
	ci.expanded = !ci.expanded
	ci.cachedLines = nil
}

// Render returns the cached or freshly rendered lines for this item.
func (ci *ChatItem) Render(theme *Theme, width int) []string {
	ci.cacheMu.RLock()
	if ci.finished && ci.cachedLines != nil && ci.cachedWidth == width {
		lines := ci.cachedLines
		ci.cacheMu.RUnlock()
		return lines
	}
	ci.cacheMu.RUnlock()

	lines := ci.renderFresh(theme, width)

	ci.cacheMu.Lock()
	if ci.finished {
		ci.cachedLines = lines
		ci.cachedWidth = width
	}
	ci.cacheMu.Unlock()

	return lines
}

func (ci *ChatItem) renderFresh(theme *Theme, width int) []string {
	switch ci.role {
	case "user":
		return ci.renderUser(theme, width)
	case "assistant":
		return ci.renderAssistant(theme, width)
	case "tool":
		return ci.renderTool(theme, width)
	case "thinking":
		return ci.renderThinking(theme, width)
	case "system":
		return []string{theme.SystemMessage.Render("  ✦ " + ci.content)}
	default:
		return []string{ci.content}
	}
}

func (ci *ChatItem) renderUser(theme *Theme, width int) []string {
	timeStr := ci.timestamp.Format("15:04")
	header := theme.UserMsgHeader.Render(fmt.Sprintf(" YOU [%s]", timeStr))
	wrapped := wordWrap(ci.content, width-6)
	result := []string{header}
	for _, line := range wrapped {
		result = append(result, theme.UserMessage.Render("  "+line))
	}
	return result
}

func (ci *ChatItem) renderAssistant(theme *Theme, width int) []string {
	timeStr := ci.timestamp.Format("15:04")
	header := theme.AssistantMsgHeader.Render(fmt.Sprintf(" SAVANT [%s]", timeStr))
	wrapped := wordWrap(ci.content, width-6)
	result := []string{header}
	for _, line := range wrapped {
		result = append(result, theme.AssistantMessage.Render("  "+line))
	}
	return result
}

func (ci *ChatItem) renderThinking(theme *Theme, width int) []string {
	wrapped := wordWrap(ci.content, width-6)
	result := []string{}
	for _, line := range wrapped {
		result = append(result, theme.ThinkingMessage.Render("  💭 "+line))
	}
	return result
}

// renderTool renders tool output in a bordered box with tool-specific colors.
func (ci *ChatItem) renderTool(theme *Theme, width int) []string {
	borderStyle := theme.ToolBorder(ci.tool)
	boxWidth := width - 4
	if boxWidth < 20 {
		boxWidth = 20
	}
	innerWidth := boxWidth - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Top border with tool name
	icon := "⚡"
	headerText := fmt.Sprintf("%s %s", icon, ci.tool)
	topBorder := borderStyle.Render("  ┌─ " + headerText + " " + strings.Repeat("─", max(1, innerWidth-len(headerText)-2)) + "┐")

	// Content lines
	contentLines := strings.Split(ci.content, "\n")
	maxLines := 10
	if ci.expanded {
		maxLines = len(contentLines)
	}

	var displayLines []string
	truncated := false
	for i, line := range contentLines {
		if i >= maxLines {
			truncated = true
			break
		}
		// Truncate long lines
		runeCount := utf8.RuneCountInString(line)
		if runeCount > innerWidth {
			runes := []rune(line)
			line = string(runes[:innerWidth-3]) + "..."
		}
		// Pad to inner width
		padding := innerWidth - utf8.RuneCountInString(line)
		if padding > 0 {
			line += strings.Repeat(" ", padding)
		}
		displayLines = append(displayLines, borderStyle.Render("  │ ") + theme.TextPrimary.Render(line) + borderStyle.Render(" │"))
	}

	// Bottom border with collapse/expand indicator
	bottomText := ""
	if truncated {
		bottomText = fmt.Sprintf(" (%d more lines, Enter to expand)", len(contentLines)-maxLines)
	} else if ci.expanded && len(contentLines) > 10 {
		bottomText = " (Enter to collapse)"
	}
	bottomBorder := borderStyle.Render("  └" + strings.Repeat("─", max(1, boxWidth-4-len(bottomText))) + bottomText + "┘")

	result := []string{topBorder}
	result = append(result, displayLines...)
	result = append(result, bottomBorder)
	return result
}

// IsFinished returns whether this item's content is frozen.
func (ci *ChatItem) IsFinished() bool {
	return ci.finished
}

// ChatList manages a lazy-rendered, cached list of chat messages.
type ChatList struct {
	items     []*ChatItem
	scrollPos int
}

// NewChatList creates a new chat list.
func NewChatList() *ChatList {
	return &ChatList{}
}

// Add adds a new message to the list.
func (cl *ChatList) Add(role, content, tool string, timestamp time.Time) *ChatItem {
	item := NewChatItem(role, content, tool, timestamp)
	cl.items = append(cl.items, item)
	return item
}

// AddStreaming adds a streaming message that can be updated.
func (cl *ChatList) AddStreaming(role, tool string, timestamp time.Time) *ChatItem {
	item := &ChatItem{
		role:      role,
		tool:      tool,
		timestamp: timestamp,
		finished:  false,
	}
	cl.items = append(cl.items, item)
	return item
}

// ScrollUp moves the viewport up.
func (cl *ChatList) ScrollUp() {
	if cl.scrollPos > 0 {
		cl.scrollPos--
	}
}

// ScrollDown moves the viewport down.
func (cl *ChatList) ScrollDown() {
	cl.scrollPos++
}

// Items returns the list of chat items.
func (cl *ChatList) Items() []*ChatItem {
	return cl.items
}

// Clear removes all items.
func (cl *ChatList) Clear() {
	cl.items = nil
	cl.scrollPos = 0
}

// Count returns the number of items.
func (cl *ChatList) Count() int {
	return len(cl.items)
}

// wordWrap wraps text to fit within the given width, using rune count.
func wordWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if utf8.RuneCountInString(line) <= width {
			result = append(result, line)
			continue
		}

		words := strings.Fields(line)
		current := ""
		for _, word := range words {
			currentLen := utf8.RuneCountInString(current)
			wordLen := utf8.RuneCountInString(word)
			if current == "" {
				current = word
			} else if currentLen+1+wordLen <= width {
				current += " " + word
			} else {
				result = append(result, current)
				current = word
			}
		}
		if current != "" {
			result = append(result, current)
		}
	}

	if len(result) == 0 {
		return []string{""}
	}
	return result
}
