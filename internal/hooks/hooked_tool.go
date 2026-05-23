package hooks

import (
	"context"
	"encoding/json"
	"fmt"
)

// HookedToolCall checks hooks before executing a tool call.
// Returns: (shouldExecute bool, updatedInput []byte, hookContext []string, err error)
func HookedToolCall(ctx context.Context, runner *Runner, toolName string, toolInput []byte) (bool, []byte, []string, error) {
	if runner == nil {
		return true, toolInput, nil, nil
	}

	result := runner.Run(ctx, EventPreToolUse, toolName, toolInput)

	switch result.Decision {
	case DecisionDeny:
		return false, nil, nil, fmt.Errorf("hook denied: %s", result.Reason)
	case DecisionAllow:
		// Hooks approved - use updated input if provided
		input := toolInput
		if result.UpdatedInput != "" {
			input = []byte(result.UpdatedInput)
		}
		return true, input, result.Context, nil
	default:
		// No opinion - proceed normally
		return true, toolInput, result.Context, nil
	}
}

// HookedToolCallJSON is a convenience wrapper that handles JSON marshaling.
func HookedToolCallJSON(ctx context.Context, runner *Runner, toolName string, args json.RawMessage) (bool, json.RawMessage, []string, error) {
	ok, updated, context, err := HookedToolCall(ctx, runner, toolName, []byte(args))
	if err != nil {
		return false, nil, nil, err
	}
	return ok, json.RawMessage(updated), context, nil
}
