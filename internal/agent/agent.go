// Package agent implements the core agentic loop for Savant CLI.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/tools"
)

// Event represents an event from the agent loop.
type Event struct {
	Type    EventType
	Content string
	Tool    string
	Error   error
}

// EventType categorizes agent events.
type EventType int

const (
	EventText       EventType = iota // Model produced text
	EventToolCall                    // Model wants to call a tool
	EventToolResult                  // Tool returned a result
	EventDone                        // Agent loop finished
	EventError                       // An error occurred
)

// Agent runs the agentic loop.
type Agent struct {
	provider provider.Provider
	registry *tools.Registry
	messages []provider.ChatMessage
	maxTurns int
	events   chan<- Event // channel to send events to the TUI
}

// NewAgent creates a new agent.
// The events channel receives real-time events from the agent loop.
// The messages parameter provides prior conversation history for continuity.
func NewAgent(p provider.Provider, registry *tools.Registry, maxTurns int, events chan<- Event, priorMessages []provider.ChatMessage) *Agent {
	return &Agent{
		provider: p,
		registry: registry,
		maxTurns: maxTurns,
		events:   events,
		messages: priorMessages,
	}
}

// Run executes the agentic loop for a user prompt.
func (a *Agent) Run(ctx context.Context, userPrompt string) error {
	// Add user message
	a.messages = append(a.messages, provider.ChatMessage{
		Role:    "user",
		Content: userPrompt,
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
		req := provider.ChatRequest{
			Messages: a.messages,
			Tools:    toolDefs,
		}

		stream, err := a.provider.Stream(ctx, req)
		if err != nil {
			a.emit(Event{Type: EventError, Error: fmt.Errorf("stream error: %w", err)})
			return err
		}

		// Collect streaming response
		var (
			fullContent string
			toolCalls   []provider.ToolCall
			streamErr   error
		)

		for {
			chunk, err := stream.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // Normal end of stream
				}
				// Real error - capture it
				streamErr = fmt.Errorf("stream read error: %w", err)
				break
			}

			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					fullContent += choice.Delta.Content
					a.emit(Event{Type: EventText, Content: choice.Delta.Content})
				}
				if len(choice.Delta.ToolCalls) > 0 {
					for _, tc := range choice.Delta.ToolCalls {
						if tc.ID != "" {
							// New tool call
							toolCalls = append(toolCalls, tc)
						} else if len(toolCalls) > 0 {
							// Delta for existing tool call - append arguments
							last := &toolCalls[len(toolCalls)-1]
							last.Function.Arguments = append(last.Function.Arguments, tc.Function.Arguments...)
						}
					}
				}
			}
		}
		stream.Close()

		// If stream had an error, surface it
		if streamErr != nil {
			a.emit(Event{Type: EventError, Error: streamErr})
			return streamErr
		}

		// If no tool calls, the model is done
		if len(toolCalls) == 0 {
			a.messages = append(a.messages, provider.ChatMessage{
				Role:    "assistant",
				Content: fullContent,
			})
			a.emit(Event{Type: EventDone})
			return nil
		}

		// Add assistant message with tool calls
		assistantMsg := provider.ChatMessage{
			Role:      "assistant",
			Content:   fullContent,
			ToolCalls: toolCalls,
		}
		a.messages = append(a.messages, assistantMsg)

		// Execute tool calls (concurrently)
		var calls []tools.ToolCall
		for _, tc := range toolCalls {
			calls = append(calls, tools.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		results := a.registry.ExecuteAll(ctx, calls)

		// Add tool results as messages
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
		}
	}

	a.emit(Event{Type: EventDone})
	return nil
}

func (a *Agent) emit(e Event) {
	if a.events != nil {
		// Non-blocking send - if channel is full, drop the event
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
