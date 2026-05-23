// Package agent - spawn_agent tool definition for the AI model.
// This tool is registered in the tool registry so the model knows about it.
// The actual execution is intercepted in the agent's Run method to allow
// direct access to the provider, blackboard, and events channel.
package agent

import (
	"context"
	"encoding/json"

	"github.com/spenc/savant-cli/internal/tools"
)

// SpawnAgentTool is a tool the AI model can call to spawn sub-agents.
// The tool's Execute method is intentionally lightweight — the agent loop
// intercepts spawn_agent calls and handles them directly with full access
// to the provider, blackboard, and events channel.
type SpawnAgentTool struct{}

// NewSpawnAgentTool creates a new spawn_agent tool definition.
func NewSpawnAgentTool() *SpawnAgentTool {
	return &SpawnAgentTool{}
}

// Name returns the tool name.
func (t *SpawnAgentTool) Name() string {
	return "spawn_agent"
}

// Description returns a description for the AI model.
func (t *SpawnAgentTool) Description() string {
	return "Spawn a sub-agent to complete a task in parallel. " +
		"The sub-agent gets the current session context (goal, plan, files modified, decisions) " +
		"and can operate independently. Use for codebase exploration, code review, " +
		"or parallel implementation. The result includes the sub-agent's findings " +
		"and any files it modified."
}

// Parameters returns the JSON schema for the tool's arguments.
func (t *SpawnAgentTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"task": {
				"type": "string",
				"description": "The task for the sub-agent to complete. Be specific about what to do, what files to read/modify, and what to report back."
			},
			"agent_type": {
				"type": "string",
				"enum": ["code", "explore", "review", "debug", "ask"],
				"description": "Type of agent: code (full tool access), explore (read-only exploration), review (read-only code review), debug (read + diagnostic commands), ask (read-only Q&A). Default: code."
			},
			"max_turns": {
				"type": "integer",
				"description": "Maximum conversation turns for the sub-agent. Default is type-specific: code=20, explore=10, review=15, debug=15, ask=5."
			}
		},
		"required": ["task"]
	}`)
}

// Kind returns the tool's side-effect profile.
func (t *SpawnAgentTool) Kind() tools.ToolKind {
	return tools.KindExecute
}

// Execute is intentionally lightweight — it returns an error because
// spawn_agent calls are intercepted and handled directly in the agent
// loop where the provider, blackboard, and events channel are available.
func (t *SpawnAgentTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	return "", errSpawnAgentIntercepted
}

// errSpawnAgentIntercepted is a sentinel error used to identify spawn_agent
// tool calls that should be handled by the agent loop rather than the tool registry.
var errSpawnAgentIntercepted = &spawnInterceptedError{}

// spawnInterceptedError indicates that spawn_agent was handled by the agent loop.
type spawnInterceptedError struct{}

func (e *spawnInterceptedError) Error() string {
	return "spawn_agent handled by agent loop"
}
