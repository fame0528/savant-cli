package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ChatItem represents a single rendered chat message with caching.
type ChatItem struct {
	role      string
	content   string
	tool      string
	timestamp time.Time
	finished  bool // Once true, the rendered output is frozen

	// Cached rendered output
	cacheMu    sync.RWMutex
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
		finished:  true, // Non-streaming items are immediately finished
	}
}

// SetContent updates the content (for streaming messages).
func (ci *ChatItem) SetContent(content string) {
	ci.cacheMu.Lock()
	defer ci.cacheMu.Unlock()
	ci.content = content
	ci.cachedLines = nil // Invalidate cache
}

// MarkFinished freezes the rendered output.
func (ci *ChatItem) MarkFinished() {
	ci.finished = true
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

	// Render fresh
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
	var lines []string

	switch ci.role {
	case "user":
		timeStr := ci.timestamp.Format("15:04:05")
		header := theme.UserMsgHeader.Render(fmt.Sprintf(" ▸ YOU [%s]", timeStr))
		content := theme.UserMessage.Render("   " + ci.content)
		lines = []string{header, content}

	case "assistant":
		timeStr := ci.timestamp.Format("15:04:05")
		header := theme.AssistantMsgHeader.Render(fmt.Sprintf(" ▸ SAVANT [%s]", timeStr))

		// Word-wrap content to width
		wrapped := wordWrap(ci.content, width-6)
		contentLines := make([]string, 0, len(wrapped)+1)
		contentLines = append(contentLines, header)
		for _, line := range wrapped {
			contentLines = append(contentLines, theme.AssistantMessage.Render("   "+line))
		}
		lines = contentLines

	case "tool":
		icon := theme.ToolIcon.Render("⚡")
		name := theme.ToolName.Render(ci.tool)
		content := ci.content
		if width > 18 && len(content) > width-18 {
			content = content[:width-21] + "..."
		}
		lines = []string{theme.ToolMessage.Render(fmt.Sprintf("   %s %s: %s", icon, name, content))}

	case "system":
		lines = []string{theme.SystemMessage.Render("  ✦ " + ci.content)}

	default:
		lines = []string{ci.content}
	}

	return lines
}

// IsFinished returns whether this item's content is frozen.
func (ci *ChatItem) IsFinished() bool {
	return ci.finished
}

// ChatList manages a lazy-rendered, cached list of chat messages.
type ChatList struct {
	items      []*ChatItem
	scrollPos  int
	totalLines int
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

// ScrollToBottom scrolls to the latest messages.
func (cl *ChatList) ScrollToBottom(height int) {
	total := cl.countLines(nil, 0)
	if total > height {
		cl.scrollPos = total - height
	} else {
		cl.scrollPos = 0
	}
}

// Render renders the visible portion of the chat list.
func (cl *ChatList) Render(theme *Theme, width, height int) []string {
	// Render all items and collect lines
	var allLines []string
	for _, item := range cl.items {
		lines := item.Render(theme, width)
		allLines = append(allLines, lines...)
	}

	// Apply scroll offset
	if cl.scrollPos > 0 && cl.scrollPos < len(allLines) {
		allLines = allLines[cl.scrollPos:]
	}

	// Truncate to height
	if len(allLines) > height {
		allLines = allLines[len(allLines)-height:]
	}

	// Auto-scroll to bottom if not manually scrolled
	if !itemStreaming(cl.items) && cl.scrollPos == 0 {
		// Auto-scroll is handled by taking the last N lines
	}

	return allLines
}

func (cl *ChatList) countLines(theme *Theme, width int) int {
	total := 0
	for _, item := range cl.items {
		lines := item.Render(theme, width)
		total += len(lines)
	}
	return total
}

func itemStreaming(items []*ChatItem) bool {
	for _, item := range items {
		if !item.finished {
			return true
		}
	}
	return false
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

// wordWrap wraps text to fit within the given width.
func wordWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if len(line) <= width {
			result = append(result, line)
			continue
		}

		// Wrap long lines
		words := strings.Fields(line)
		current := ""
		for _, word := range words {
			if current == "" {
				current = word
			} else if len(current)+1+len(word) <= width {
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

// RenderChatList is a convenience function for rendering the chat list in the TUI.
func RenderChatList(items []*ChatItem, theme *Theme, width, height int, streaming string, spinnerFrame int) string {
	var allLines []string

	// Render header
	header := safeRepeat("═", max(1, width-12))
	allLines = append(allLines, theme.ChatHeader.Render(" ╔═ CHAT "+header+"╗"))

	// Render all items
	for _, item := range items {
		lines := item.Render(theme, width)
		allLines = append(allLines, lines...)
	}

	// Render streaming content
	if streaming != "" {
		spinner := theme.Spinner(spinnerFrame)
		header := theme.AssistantMsgHeader.Render(fmt.Sprintf(" ▸ SAVANT %s", spinner))
		allLines = append(allLines, header)
		wrapped := wordWrap(streaming, width-6)
		for _, line := range wrapped {
			allLines = append(allLines, theme.AssistantMessage.Render("   "+line+"▌"))
		}
	}

	// Render footer
	footer := safeRepeat("═", max(1, width-3))
	allLines = append(allLines, theme.ChatHeader.Render(" ╚"+footer+"╝"))

	// Apply viewport
	if len(allLines) > height {
		allLines = allLines[len(allLines)-height:]
	}

	return strings.Join(allLines, "\n")
}
