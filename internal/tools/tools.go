// Package tools implements the built-in tool system for Savant CLI.
package tools

import (
	"context"
	"encoding/json"
	"sync"
)

// Tool is the interface all built-in tools implement.
type Tool interface {
	// Name returns the tool's name (sent to the model).
	Name() string

	// Description returns a description for the model.
	Description() string

	// Parameters returns the JSON Schema for the tool's parameters.
	Parameters() json.RawMessage

	// Execute runs the tool with the given arguments.
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// ToolCall represents a model's request to call a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// Registry holds all available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a tool registry with all built-in tools.
func NewRegistry() *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	r.Register(NewBashTool())
	r.Register(NewReadTool())
	r.Register(NewEditTool())
	r.Register(NewWriteTool())
	r.Register(NewGlobTool())
	r.Register(NewGrepTool())
	return r
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *Registry) All() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ExecuteAll runs multiple tool calls concurrently and returns results.
func (r *Registry) ExecuteAll(ctx context.Context, calls []ToolCall) []ToolResult {
	results := make([]ToolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c ToolCall) {
			defer wg.Done()

			t, ok := r.tools[c.Name]
			if !ok {
				results[idx] = ToolResult{
					ToolCallID: c.ID,
					Content:    "unknown tool: " + c.Name,
					IsError:    true,
				}
				return
			}

			content, err := t.Execute(ctx, c.Arguments)
			if err != nil {
				results[idx] = ToolResult{
					ToolCallID: c.ID,
					Content:    err.Error(),
					IsError:    true,
				}
				return
			}
			results[idx] = ToolResult{
				ToolCallID: c.ID,
				Content:    content,
			}
		}(i, call)
	}

	wg.Wait()
	return results
}
