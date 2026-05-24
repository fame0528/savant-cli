package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spenc/savant-cli/internal/provider"
)

// ToolOutputDistiller manages tool output distillation.
// When tool output exceeds a threshold, the full output is saved to disk
// and a summary is kept in the conversation context.
type ToolOutputDistiller struct {
	outputDir   string
	maxChars    int // Max characters before distilling
	keepHead    int // Lines to keep from head
	keepTail    int // Lines to keep from tail
}

// NewToolOutputDistiller creates a distiller with the given limits.
func NewToolOutputDistiller(outputDir string) *ToolOutputDistiller {
	return &ToolOutputDistiller{
		outputDir: outputDir,
		maxChars:  4000,
		keepHead:  50,
		keepTail:  10,
	}
}

// Distill checks if a tool output needs distillation.
// If it does, saves full output to disk and returns a summary.
// If not, returns the original content unchanged.
func (d *ToolOutputDistiller) Distill(toolName, content string) string {
	if len(content) <= d.maxChars {
		return content
	}

	// Save full output to disk
	_ = d.saveToDisk(toolName, content)

	// Build summary: head + truncation notice + tail
	lines := strings.Split(content, "\n")

	headEnd := d.keepHead
	if headEnd > len(lines) {
		headEnd = len(lines)
	}

	tailStart := len(lines) - d.keepTail
	if tailStart < headEnd {
		tailStart = headEnd
	}

	var sb strings.Builder
	for i := 0; i < headEnd; i++ {
		sb.WriteString(lines[i])
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\n[... %d lines truncated, full output saved to disk ...]\n\n", len(lines)-headEnd-(len(lines)-tailStart)))
	for i := tailStart; i < len(lines); i++ {
		sb.WriteString(lines[i])
		sb.WriteString("\n")
	}

	return sb.String()
}

// saveToDisk saves the full output to a file.
func (d *ToolOutputDistiller) saveToDisk(toolName, content string) string {
	os.MkdirAll(d.outputDir, 0o755)

	// Generate filename from tool name and content hash
	filename := fmt.Sprintf("%s_output.txt", toolName)
	path := filepath.Join(d.outputDir, filename)

	// Append to existing file (multiple outputs from same tool)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return ""
	}
	defer f.Close()

	f.WriteString("---\n")
	f.WriteString(content)
	f.WriteString("\n---\n")

	return path
}

// MaskOldToolOutputs masks tool outputs in older messages to save context.
// Protects the newest `protectChars` characters of tool content.
// Older outputs are truncated to head+tail.
func MaskOldToolOutputs(messages []provider.ChatMessage, protectChars int) []provider.ChatMessage {
	if len(messages) <= 4 {
		return messages
	}

	// Find the cutoff: protect the last N chars of tool output
	totalToolChars := 0
	cutoffIdx := len(messages)

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			totalToolChars += len(messages[i].Content)
			if totalToolChars > protectChars {
				cutoffIdx = i
				break
			}
		}
	}

	// Mask messages before the cutoff
	result := make([]provider.ChatMessage, len(messages))
	copy(result, messages)

	for i := 0; i < cutoffIdx; i++ {
		if result[i].Role == "tool" && len(result[i].Content) > 500 {
			lines := strings.Split(result[i].Content, "\n")
			if len(lines) > 20 {
				head := strings.Join(lines[:10], "\n")
				tail := strings.Join(lines[len(lines)-5:], "\n")
				result[i].Content = head + fmt.Sprintf("\n[... %d lines masked ...]\n", len(lines)-15) + tail
			}
		}
	}

	return result
}

// CompactWithSummary creates a compact version of the conversation.
// Keeps system messages, summarizes older turns, preserves recent turns.
func CompactWithSummary(ctx context.Context, messages []provider.ChatMessage, maxRecent int) []provider.ChatMessage {
	if len(messages) <= maxRecent+2 {
		return messages
	}

	// Separate system messages, old messages, and recent messages
	var systemMsgs []provider.ChatMessage
	var oldMsgs []provider.ChatMessage
	var recentMsgs []provider.ChatMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemMsgs = append(systemMsgs, m)
		}
	}

	nonSystem := messages[len(systemMsgs):]
	if len(nonSystem) > maxRecent {
		oldMsgs = nonSystem[:len(nonSystem)-maxRecent]
		recentMsgs = nonSystem[len(nonSystem)-maxRecent:]
	} else {
		recentMsgs = nonSystem
	}

	// Summarize old messages
	summary := summarizeOldMessages(oldMsgs)

	// Build compacted messages
	var result []provider.ChatMessage
	result = append(result, systemMsgs...)
	if summary != "" {
		result = append(result, provider.ChatMessage{
			Role:    "system",
			Content: "[Context compacted: " + summary + "]",
		})
	}
	result = append(result, recentMsgs...)

	return result
}

func summarizeOldMessages(messages []provider.ChatMessage) string {
	var userMsgs, assistantMsgs, toolCalls int
	var topics []string

	for _, m := range messages {
		switch m.Role {
		case "user":
			userMsgs++
			words := strings.Fields(m.Content)
			if len(words) > 3 {
				topics = append(topics, strings.Join(words[:3], " ")+"...")
			}
		case "assistant":
			assistantMsgs++
		case "tool":
			toolCalls++
		}
	}

	if userMsgs == 0 && assistantMsgs == 0 {
		return ""
	}

	summary := fmt.Sprintf("%d turns with %d user messages, %d assistant responses, %d tool calls.",
		userMsgs+assistantMsgs, userMsgs, assistantMsgs, toolCalls)

	uniqueTopics := dedupTopics(topics)
	if len(uniqueTopics) > 3 {
		uniqueTopics = uniqueTopics[:3]
	}
	if len(uniqueTopics) > 0 {
		summary += " Topics: " + strings.Join(uniqueTopics, "; ")
	}

	return summary
}

func dedupTopics(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
