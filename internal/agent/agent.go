// Package agent implements the core agentic loop for Savant CLI.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/tools"
)

// Event represents an event from the agent loop.
type Event struct {
	Type     EventType
	Content  string
	Tool     string
	Error    error
	Messages []provider.ChatMessage // For EventHistoryUpdate
}

// EventType categorizes agent events.
type EventType int

const (
	EventText          EventType = iota // Model produced text
	EventThinking                       // Model produced reasoning/thinking
	EventToolCall                       // Model wants to call a tool
	EventToolResult                     // Tool returned a result
	EventDone                           // Agent loop finished
	EventError                          // An error occurred
	EventHistoryUpdate                  // Conversation history update
)

// Agent runs the agentic loop.
type Agent struct {
	provider        provider.Provider
	registry        *tools.Registry
	messages        []provider.ChatMessage
	maxTurns        int
	events          chan<- Event
	systemPrompt    string
	instructionsMsg string
	stepMsg         string
}

// NewAgent creates a new agent with template-based grounding.
func NewAgent(p provider.Provider, registry *tools.Registry, maxTurns int, events chan<- Event, priorMessages []provider.ChatMessage) (*Agent, error) {
	// Build system prompt from template
	sysPrompt, err := BuildSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("build system prompt: %w", err)
	}

	instructions, err := BuildInstructionsPrompt()
	if err != nil {
		return nil, fmt.Errorf("build instructions prompt: %w", err)
	}

	step, err := BuildStepPrompt()
	if err != nil {
		return nil, fmt.Errorf("build step prompt: %w", err)
	}

	return &Agent{
		provider:        p,
		registry:        registry,
		maxTurns:        maxTurns,
		events:          events,
		messages:        priorMessages,
		systemPrompt:    sysPrompt,
		instructionsMsg: instructions,
		stepMsg:         step,
	}, nil
}

// Run executes the agentic loop for a user prompt.
func (a *Agent) Run(ctx context.Context, userPrompt string) error {
	// Ensure system prompt is first message
	if len(a.messages) == 0 || a.messages[0].Role != "system" {
		a.messages = append([]provider.ChatMessage{
			{Role: "system", Content: a.systemPrompt},
		}, a.messages...)
	}

	// Add user message
	a.messages = append(a.messages, provider.ChatMessage{
		Role:    "user",
		Content: userPrompt,
	})

	// Inject instructions reminder after user message
	a.messages = append(a.messages, provider.ChatMessage{
		Role:    "system",
		Content: a.instructionsMsg,
	})

	// Build tool definitions for the model
	var toolDefs []provider.Tool
	for _, t := range a.registry.All() {
		toolDefs = append(toolDefs, provider.Tool{
			Type: "function",
			Function: provider.ToolFunction{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  json.RawMessage(t.Parameters()),
			},
		})
	}

	// Agent loop
	for turn := 0; turn < a.maxTurns; turn++ {
		// Context compaction check
		cm := NewContextManager(128000, 0.80, nil)
		if cm.NeedsCompaction(a.messages) {
			compacted, err := cm.Compact(ctx, a.messages)
			if err == nil {
				a.messages = compacted
			}
		}

		// Inject step reminder before each model call
		messages := make([]provider.ChatMessage, len(a.messages))
		copy(messages, a.messages)
		messages = append(messages, provider.ChatMessage{
			Role:    "system",
			Content: a.stepMsg,
		})

		req := provider.ChatRequest{
			Messages: messages,
			Tools:    toolDefs,
		}

		stream, err := a.provider.Stream(ctx, req)
		if err != nil {
			a.emit(Event{Type: EventError, Error: fmt.Errorf("stream error: %w", err)})
			return err
		}

		// Collect streaming response
		var (
			fullContent   string
			fullReasoning string
			toolCalls     []provider.ToolCall
			streamErr     error
		)

		for {
			chunk, err := stream.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				streamErr = fmt.Errorf("stream read error: %w", err)
				break
			}

			for _, choice := range chunk.Choices {
				// Capture reasoning silently
				if choice.Delta.Reasoning != "" {
					fullReasoning += choice.Delta.Reasoning
				}

				if choice.Delta.Content != "" {
					fullContent += choice.Delta.Content
					a.emit(Event{Type: EventText, Content: choice.Delta.Content})
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

		if streamErr != nil {
			a.emit(Event{Type: EventError, Error: streamErr})
			return streamErr
		}

		// If no tool calls, the model is done
		if len(toolCalls) == 0 {
			msg := provider.ChatMessage{
				Role:    "assistant",
				Content: fullContent,
			}
			a.messages = append(a.messages, msg)
			a.emit(Event{Type: EventDone})
			a.emit(Event{Type: EventHistoryUpdate, Messages: a.Messages()})
			return nil
		}

		// Add assistant message with tool calls
		a.messages = append(a.messages, provider.ChatMessage{
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

		results := a.registry.ExecuteAll(ctx, calls)

		// Add tool results as messages and handle follow-ups
		var followUps []tools.ToolCall
		for i, result := range results {
			toolName := ""
			if i < len(toolCalls) {
				toolName = toolCalls[i].Function.Name
			}

			a.emit(Event{
				Type:    EventToolResult,
				Tool:    toolName,
				Content: result.Content,
			})

			a.messages = append(a.messages, provider.ChatMessage{
				Role:       "tool",
				ToolCallID: result.ToolCallID,
				Content:    result.Content,
				Name:       toolName,
			})

			// Collect follow-up tool calls (tail calls)
			if result.FollowUp != nil {
				followUps = append(followUps, *result.FollowUp)
			}
		}

		// Execute tail tool calls (without returning to model)
		if len(followUps) > 0 {
			slog.Info("Executing tail tool calls", "count", len(followUps))
			followUpResults := a.registry.ExecuteAll(ctx, followUps)
			for _, result := range followUpResults {
				a.messages = append(a.messages, provider.ChatMessage{
					Role:       "tool",
					ToolCallID: result.ToolCallID,
					Content:    result.Content,
				})
			}
		}
	}

	a.emit(Event{Type: EventDone})
	a.emit(Event{Type: EventHistoryUpdate, Messages: a.Messages()})
	return nil
}

func (a *Agent) emit(e Event) {
	if a.events != nil {
		select {
		case a.events <- e:
		default:
		}
	}
}

// Messages returns the current conversation messages.
func (a *Agent) Messages() []provider.ChatMessage {
	return a.messages
}
