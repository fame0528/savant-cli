// Package types defines core domain types for Savant CLI.
package types

import (
	"encoding/json"
	"time"
)

// Role represents a message participant.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

// Message represents a single conversation message.
type Message struct {
	ID        string        `json:"id"`
	SessionID string        `json:"session_id"`
	Role      Role          `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []ToolCall    `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

// ToolCall represents a request from the model to execute a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the outcome of executing a tool.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Session represents a conversation session.
type Session struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	MessageCount  int       `json:"message_count"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	Cost          float64   `json:"cost"`
	ProviderName  string    `json:"provider_name"`
	ModelName     string    `json:"model_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProviderConfig holds configuration for a single AI provider.
type ProviderConfig struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	Enabled bool   `json:"enabled"`
}

// ProviderOverride allows per-agent provider routing.
type ProviderOverride struct {
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

// SmartRoutingConfig controls cheap-for-simple, strong-for-complex routing.
type SmartRoutingConfig struct {
	Enabled       bool   `json:"enabled"`
	SimpleModel   string `json:"simple_model"`
	StrongModel   string `json:"strong_model"`
	SimpleMaxChars int   `json:"simple_max_chars,omitempty"`
	SimpleMaxWords int   `json:"simple_max_words,omitempty"`
}

// RoutingDecision records why a particular model was chosen.
type RoutingDecision struct {
	Model      string `json:"model"`
	Complexity string `json:"complexity"` // "simple" or "strong"
	Reason     string `json:"reason"`
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read,omitempty"`
	CacheCreate  int `json:"cache_create,omitempty"`
}
