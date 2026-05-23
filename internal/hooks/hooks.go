// Package hooks runs user-defined shell commands that fire on hook events
// (e.g. PreToolUse), returning decisions that control agent behavior.
package hooks

import (
	"strings"
)

// Event constants.
const (
	EventPreToolUse = "PreToolUse"
)

// HaltExitCode is the exit code that halts the whole turn.
// Exit code 2 blocks the current tool call.
// Exit code 49 halts the entire turn.
const HaltExitCode = 49

// Decision represents the outcome of a single hook execution.
type Decision int

const (
	DecisionNone  Decision = iota // No opinion
	DecisionAllow                 // Explicitly allowed
	DecisionDeny                  // Blocked
)

func (d Decision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionDeny:
		return "deny"
	default:
		return "none"
	}
}

// HookConfig defines a single hook rule.
type HookConfig struct {
	Name    string `json:"name"`
	Matcher string `json:"matcher"` // Regex to match tool name (empty = match all)
	Command string `json:"command"` // Shell command to run
	Timeout int    `json:"timeout"` // Seconds (default 30)
	Event   string `json:"event"`   // EventPreToolUse
}

// HookResult is the outcome of running a single hook.
type HookResult struct {
	HookName     string   `json:"hook_name"`
	Decision     Decision `json:"decision"`
	Reason       string   `json:"reason,omitempty"`
	Context      []string `json:"context,omitempty"`
	UpdatedInput string   `json:"updated_input,omitempty"` // JSON patch for tool input
}

// AggregatedResult is the combined result of all hooks for one tool call.
type AggregatedResult struct {
	Decision     Decision
	Reason       string
	Context      []string
	UpdatedInput string
	HookCount    int
}

// Aggregate combines multiple hook results into a single decision.
// Processing order: deny wins over allow, allow wins over none, halt is sticky.
func Aggregate(results []HookResult) AggregatedResult {
	agg := AggregatedResult{
		Decision:  DecisionNone,
		HookCount: len(results),
	}

	var reasons []string
	var contexts []string

	for _, r := range results {
		switch r.Decision {
		case DecisionDeny:
			agg.Decision = DecisionDeny
			if r.Reason != "" {
				reasons = append(reasons, r.Reason)
			}
		case DecisionAllow:
			if agg.Decision != DecisionDeny {
				agg.Decision = DecisionAllow
			}
		}

		if len(r.Context) > 0 {
			contexts = append(contexts, r.Context...)
		}

		if r.UpdatedInput != "" {
			agg.UpdatedInput = r.UpdatedInput
		}
	}

	if len(reasons) > 0 {
		agg.Reason = strings.Join(reasons, "; ")
	}
	agg.Context = contexts

	return agg
}
