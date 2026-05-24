package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spenc/savant-cli/internal/config"
	"github.com/spenc/savant-cli/internal/provider"
)

// ContextManager handles conversation context with automatic compaction.
type ContextManager struct {
	maxTokens           int
	compactThreshold    float64
	compactFunc         func(ctx context.Context, messages []provider.ChatMessage) ([]provider.ChatMessage, error)
}

// NewContextManager creates a context manager with the given limits.
func NewContextManager(maxTokens int, compactThreshold float64, compactFunc func(ctx context.Context, messages []provider.ChatMessage) ([]provider.ChatMessage, error)) *ContextManager {
	return &ContextManager{
		maxTokens:        maxTokens,
		compactThreshold: compactThreshold,
		compactFunc:      compactFunc,
	}
}

// EstimateTokens provides a rough token count estimate.
// Uses the ~4 chars per token heuristic.
func EstimateTokens(messages []provider.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / 4
		// Tool calls add overhead
		for range m.ToolCalls {
			total += 50
		}
	}
	return total
}

// NeedsCompaction checks if the conversation is approaching the context limit.
func (cm *ContextManager) NeedsCompaction(messages []provider.ChatMessage) bool {
	if cm.maxTokens <= 0 {
		return false
	}
	estimated := EstimateTokens(messages)
	threshold := float64(cm.maxTokens) * cm.compactThreshold
	return float64(estimated) > threshold
}

// Compact reduces the conversation history to fit within the context window.
// It summarizes older messages while preserving recent ones.
func (cm *ContextManager) Compact(ctx context.Context, messages []provider.ChatMessage) ([]provider.ChatMessage, error) {
	if cm.compactFunc != nil {
		return cm.compactFunc(ctx, messages)
	}

	// Default: keep system message + last N messages, summarize the rest
	if len(messages) <= 4 {
		return messages, nil
	}

	// Find system message (if any)
	var systemMsg *provider.ChatMessage
	startIdx := 0
	if len(messages) > 0 && messages[0].Role == "system" {
		msg := messages[0]
		systemMsg = &msg
		startIdx = 1
	}

	// Keep the last 6 messages
	keepCount := 6
	if len(messages)-startIdx <= keepCount {
		return messages, nil
	}

	// Summarize older messages
	oldMessages := messages[startIdx : len(messages)-keepCount]
	summary := summarizeMessages(oldMessages)

	// Build compacted messages
	var compacted []provider.ChatMessage
	if systemMsg != nil {
		compacted = append(compacted, *systemMsg)
	}
	compacted = append(compacted, provider.ChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("[Context compacted: %d messages summarized]\n%s", len(oldMessages), summary),
	})
	compacted = append(compacted, messages[len(messages)-keepCount:]...)

	return compacted, nil
}

// summarizeMessages creates a brief summary of a set of messages.
func summarizeMessages(messages []provider.ChatMessage) string {
	var userMsgs, assistantMsgs, toolCalls int
	var topics []string

	for _, m := range messages {
		switch m.Role {
		case "user":
			userMsgs++
			// Extract topic from first few words
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

	summary := fmt.Sprintf("Conversation with %d user messages, %d assistant responses, %d tool calls.",
		userMsgs, assistantMsgs, toolCalls)

	if len(topics) > 0 {
		uniqueTopics := dedup(topics)
		if len(uniqueTopics) > 3 {
			uniqueTopics = uniqueTopics[:3]
		}
		summary += " Topics discussed: " + strings.Join(uniqueTopics, "; ")
	}

	return summary
}

func dedup(items []string) []string {
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

// ─────────────────────────────────────────────────────────────
// Tool Output Distillation (from Gemini CLI)
// ─────────────────────────────────────────────────────────────

const (
	DistillThreshold = 4000
	DistillKeepFirst = 50
	DistillKeepLast  = 10
)

// DistillToolOutput reduces a large tool output to a summary.
func DistillToolOutput(output, toolName string) (string, bool) {
	if len(output) <= DistillThreshold {
		return output, false
	}

	savedPath := saveDistilledOutput(toolName, output)

	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	if totalLines <= DistillKeepFirst+DistillKeepLast {
		return output, false
	}

	first := lines[:DistillKeepFirst]
	last := lines[totalLines-DistillKeepLast:]

	var sb strings.Builder
	for _, line := range first {
		sb.WriteString(line + "\n")
	}
	sb.WriteString(fmt.Sprintf("\n... [%d lines truncated, full: %s] ...\n\n",
		totalLines-DistillKeepFirst-DistillKeepLast, savedPath))
	for _, line := range last {
		sb.WriteString(line + "\n")
	}
	return sb.String(), true
}

func saveDistilledOutput(toolName, output string) string {
	outputDir := filepath.Join(config.ConfigDir(), "tool-outputs")
	os.MkdirAll(outputDir, 0o755)
	safeName := strings.ReplaceAll(toolName, "/", "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")
	filename := fmt.Sprintf("%s_%d.txt", safeName, time.Now().UnixMilli())
	path := filepath.Join(outputDir, filename)
	os.WriteFile(path, []byte(output), 0o644)
	return path
}
