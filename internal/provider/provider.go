// Package provider defines the provider interface and routing logic.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// ToolCall represents a model's request to call a tool.
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // "function"
	Function FunctionCall    `json:"function"`
}

// FunctionCall is the function portion of a tool call.
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Tool defines a tool schema for the model.
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function the model can call.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatRequest is a request to the chat completion API.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// ChatResponse is a non-streaming response from the API.
type ChatResponse struct {
	ID      string         `json:"id"`
	Choices []Choice       `json:"choices"`
	Usage   Usage          `json:"usage"`
	Model   string         `json:"model"`
}

// Choice is a single completion choice.
type Choice struct {
	Index        int          `json:"index"`
	Message      ChatMessage  `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk is a single chunk from a streaming response.
type StreamChunk struct {
	ID      string        `json:"id"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage        `json:"usage,omitempty"`
}

// StreamChoice is a choice within a stream chunk.
type StreamChoice struct {
	Index        int            `json:"index"`
	Delta        StreamDelta    `json:"delta"`
	FinishReason string         `json:"finish_reason,omitempty"`
}

// StreamDelta contains the incremental content.
type StreamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Provider is the interface all AI providers implement.
type Provider interface {
	// Name returns the provider's display name.
	Name() string

	// Chat sends a non-streaming chat completion request.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Stream sends a streaming chat completion request.
	Stream(ctx context.Context, req ChatRequest) (StreamReader, error)

	// IsAvailable checks if the provider is reachable.
	IsAvailable(ctx context.Context) bool
}

// StreamReader reads streaming chunks from a provider.
type StreamReader interface {
	// Next returns the next chunk. Returns io.EOF when done.
	Next() (*StreamChunk, error)
	// Close closes the stream.
	Close() error
}

// ErrProviderUnavailable is returned when a provider is not reachable.
var ErrProviderUnavailable = fmt.Errorf("provider unavailable")
