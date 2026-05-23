// Package agent - Sub-agent spawning with blackboard context sharing.
// The spawn package creates sub-agents with filtered tool access and injects
// blackboard context into their system prompts. Results are extracted and
// merged back into the parent's blackboard.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/tools"
)

// SubAgentConfig configures a sub-agent spawn.
type SubAgentConfig struct {
	// AgentType determines tool access and system prompt (code/explore/review/debug/ask).
	AgentType AgentType
	// Task is the description of what the sub-agent should do.
	Task string
	// MaxTurns limits the sub-agent's conversation turns.
	// If 0, the agent type's default is used.
	MaxTurns int
	// Blackboard is the shared blackboard. If nil, a new empty one is created.
	Blackboard *Blackboard
	// Provider is the AI provider to use. Must not be nil.
	Provider provider.Provider
	// Registry is the tool registry. Must not be nil.
	Registry *tools.Registry
	// Events is an optional channel for receiving progress events from the sub-agent.
	Events chan<- Event
	// AgentID is a unique identifier for this sub-agent (for logging and blackboard tracking).
	AgentID string
	// ProjectDir is the project directory for loading context files.
	ProjectDir string
}

// SubAgentResult holds the structured result of a sub-agent execution.
type SubAgentResult struct {
	// AgentID is the unique identifier of the sub-agent.
	AgentID string `json:"agent_id"`
	// AgentType is the type of agent that ran.
	AgentType AgentType `json:"agent_type"`
	// Task is the original task description.
	Task string `json:"task"`
	// Result is the final assistant message content.
	Result string `json:"result"`
	// FilesModified is the list of files modified by the sub-agent.
	FilesModified []string `json:"files_modified"`
	// FilesRead is the list of files read by the sub-agent.
	FilesRead []string `json:"files_read"`
	// Decisions is the list of decisions made by the sub-agent.
	Decisions []string `json:"decisions"`
	// Duration is how long the sub-agent ran.
	Duration time.Duration `json:"duration_seconds"`
	// TurnCount is how many turns the sub-agent took.
	TurnCount int `json:"turn_count"`
	// Error is non-nil if the sub-agent encountered a fatal error.
	Error error `json:"error,omitempty"`
}

// SubAgentData is the data injected into the subagent.md template.
type SubAgentData struct {
	AgentType         string
	Task              string
	BlackboardContext string
	ProjectContext    string
}

// RunSubAgent spawns and runs a sub-agent with the given configuration.
// It blocks until the sub-agent completes or the context is cancelled.
// The result includes the final assistant message, files modified, and metadata.
func RunSubAgent(ctx context.Context, cfg SubAgentConfig) SubAgentResult {
	start := time.Now()
	agentID := cfg.AgentID
	if agentID == "" {
		agentID = fmt.Sprintf("sub-%d", time.Now().UnixNano())
	}

	result := SubAgentResult{
		AgentID:   agentID,
		AgentType: cfg.AgentType,
		Task:      cfg.Task,
	}

	// Resolve max turns
	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = cfg.AgentType.DefaultMaxTurns()
	}

	// Create or use existing blackboard
	bb := cfg.Blackboard
	if bb == nil {
		bb = NewBlackboard()
	}
	bb.Set(BlackboardAgentType, cfg.AgentType.String(), agentID)

	// Build sub-agent system prompt with blackboard context
	sysPrompt, err := BuildSubAgentSystemPrompt(cfg)
	if err != nil {
		result.Error = fmt.Errorf("build system prompt: %w", err)
		return result
	}

	// Build instructions prompt (reuse the standard one)
	instructions, err := BuildInstructionsPrompt()
	if err != nil {
		instructions = ""
	}

	// Build step prompt
	step, err := BuildStepPrompt()
	if err != nil {
		step = ""
	}

	// Filter tools based on agent type
	filteredTools := cfg.AgentType.FilterTools(cfg.Registry.All())

	// Build the sub-agent's message list with system prompt
	messages := []provider.ChatMessage{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: cfg.Task},
	}

	if instructions != "" {
		messages = append(messages, provider.ChatMessage{
			Role: "system", Content: instructions,
		})
	}

	// Build tool definitions for the model
	var toolDefs []provider.Tool
	for _, t := range filteredTools {
		toolDefs = append(toolDefs, provider.Tool{
			Type: "function",
			Function: provider.ToolFunction{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  []byte(t.Parameters()),
			},
		})
	}

	// Agent loop
	turnCount := 0
	var lastAssistantContent string
	var allFilesModified []string
	var allFilesRead []string
	var allDecisions []string

	for turn := 0; turn < maxTurns; turn++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			result.Duration = time.Since(start)
			result.TurnCount = turnCount
			result.Result = lastAssistantContent
			result.FilesModified = allFilesModified
			result.FilesRead = allFilesRead
			result.Decisions = allDecisions
			return result
		default:
		}

		// Inject step reminder
		turnMessages := make([]provider.ChatMessage, len(messages))
		copy(turnMessages, messages)
		if step != "" {
			turnMessages = append(turnMessages, provider.ChatMessage{
				Role:    "system",
				Content: step,
			})
		}

		req := provider.ChatRequest{
			Messages: turnMessages,
			Tools:    toolDefs,
		}

		stream, err := cfg.Provider.Stream(ctx, req)
		if err != nil {
			result.Error = fmt.Errorf("stream error at turn %d: %w", turn, err)
			result.Duration = time.Since(start)
			result.TurnCount = turnCount
			result.Result = lastAssistantContent
			result.FilesModified = allFilesModified
			result.FilesRead = allFilesRead
			result.Decisions = allDecisions
			return result
		}

		// Collect streaming response
		var fullContent string
		var toolCalls []provider.ToolCall

		for {
			chunk, err := stream.Next()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				result.Error = fmt.Errorf("stream read error at turn %d: %w", turn, err)
				break
			}

			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					fullContent += choice.Delta.Content
				}
				if len(choice.Delta.ToolCalls) > 0 {
					for _, tc := range choice.Delta.ToolCalls {
						if tc.ID != "" {
							toolCalls = append(toolCalls, tc)
						} else if len(toolCalls) > 0 {
							last := &toolCalls[len(toolCalls)-1]
							last.Function.Arguments = append(last.Function.Arguments, tc.Function.Arguments...)
						}
					}
				}
			}
		}
		stream.Close()

		if result.Error != nil {
			result.Duration = time.Since(start)
			result.TurnCount = turnCount
			result.Result = lastAssistantContent
			result.FilesModified = allFilesModified
			result.FilesRead = allFilesRead
			return result
		}

		// Update last assistant content
		if fullContent != "" {
			lastAssistantContent = fullContent
		}

		// No tool calls = done
		if len(toolCalls) == 0 {
			messages = append(messages, provider.ChatMessage{
				Role:    "assistant",
				Content: fullContent,
			})
			turnCount++
			break
		}

		// Add assistant message with tool calls
		messages = append(messages, provider.ChatMessage{
			Role:      "assistant",
			Content:   fullContent,
			ToolCalls: toolCalls,
		})

		// Execute tool calls
		var calls []tools.ToolCall
		for _, tc := range toolCalls {
			calls = append(calls, tools.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		results := cfg.Registry.ExecuteAll(ctx, calls)

		// Process results
		for i, tResult := range results {
			toolName := ""
			if i < len(toolCalls) {
				toolName = toolCalls[i].Function.Name
			}

			// Track file operations by parsing tool call arguments
			if i < len(calls) {
				filePath := extractFilePath(calls[i].Arguments)
				if filePath != "" {
					switch toolName {
					case "edit", "write":
						allFilesModified = append(allFilesModified, filePath)
						bb.Append(BlackboardFilesModified, filePath, agentID)
					case "read":
						allFilesRead = append(allFilesRead, filePath)
						bb.Append(BlackboardFilesRead, filePath, agentID)
					}
				}
			}

			// Emit progress event if channel exists
			if cfg.Events != nil {
				cfg.Events <- Event{
					Type:    EventToolResult,
					Tool:    toolName,
					Content: tResult.Content,
				}
			}

			messages = append(messages, provider.ChatMessage{
				Role:       "tool",
				ToolCallID: tResult.ToolCallID,
				Content:    tResult.Content,
				Name:       toolName,
			})
		}

		turnCount++
	}

	result.Result = lastAssistantContent
	result.FilesModified = allFilesModified
	result.FilesRead = allFilesRead
	result.Decisions = allDecisions
	result.Duration = time.Since(start)
	result.TurnCount = turnCount

	// Emit done event
	if cfg.Events != nil {
		cfg.Events <- Event{
			Type:    EventDone,
			Content: lastAssistantContent,
		}
	}

	return result
}

// BuildSubAgentSystemPrompt builds the system prompt for a sub-agent
// from the subagent.md template with blackboard context injected.
func BuildSubAgentSystemPrompt(cfg SubAgentConfig) (string, error) {
	// Read the subagent template
	tmplContent, err := templateFS.ReadFile("templates/subagent.md")
	if err != nil {
		return "", fmt.Errorf("read subagent template: %w", err)
	}

	tmpl, err := template.New("subagent").Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse subagent template: %w", err)
	}

	// Build blackboard context string
	bb := cfg.Blackboard
	if bb == nil {
		bb = NewBlackboard()
	}
	snapshot := bb.Snapshot()
	blackboardContext := snapshot.FormatSnapshot()

	// Build project context string
	projectContext := buildProjectContext(cfg.ProjectDir)

	// Build template data
	data := SubAgentData{
		AgentType:         cfg.AgentType.String(),
		Task:              cfg.Task,
		BlackboardContext: blackboardContext,
		ProjectContext:    projectContext,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute subagent template: %w", err)
	}

	return buf.String(), nil
}

// buildProjectContext loads project context files from the given directory.
func buildProjectContext(projectDir string) string {
	if projectDir == "" {
		return ""
	}

	var sb strings.Builder
	contextFiles := []string{"AGENTS.md", "SAVANT.md", "CLAUDE.md", "GEMINI.md", "CRUSH.md", "OpenCode.md"}

	for _, file := range contextFiles {
		path := filepath.Join(projectDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("--- %s ---\n", file))
		sb.WriteString(string(data))
		sb.WriteString("\n\n")

		// Limit total project context to 8KB to avoid blowing the token budget
		if sb.Len() > 8192 {
			sb.WriteString("(project context truncated at 8KB limit)\n")
			break
		}
	}

	return sb.String()
}

// extractFilePath attempts to extract a file path from tool call arguments JSON.
func extractFilePath(args []byte) string {
	var parsed struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &parsed); err != nil {
		return ""
	}
	return parsed.Path
}

// SubAgentSummary formats a SubAgentResult as a readable summary string
// suitable for injecting as a tool result message into the parent conversation.
func SubAgentSummary(result SubAgentResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Agent %s (%s) completed in %s (%d turns)\n",
		result.AgentID, result.AgentType, formatDuration(result.Duration), result.TurnCount))

	if result.Result != "" {
		sb.WriteString("\nResult:\n")
		sb.WriteString(result.Result)
		sb.WriteString("\n")
	}

	if len(result.FilesModified) > 0 {
		sb.WriteString("\nFiles modified:\n")
		for _, f := range result.FilesModified {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}

	if len(result.FilesRead) > 0 {
		sb.WriteString(fmt.Sprintf("\nFiles read: %d\n", len(result.FilesRead)))
	}

	if len(result.Decisions) > 0 {
		sb.WriteString("\nDecisions:\n")
		for _, d := range result.Decisions {
			sb.WriteString(fmt.Sprintf("  - %s\n", d))
		}
	}

	if result.Error != nil {
		sb.WriteString(fmt.Sprintf("\nError: %s\n", result.Error))
	}

	return sb.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", mins, secs)
}
