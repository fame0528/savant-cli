// Package agent - Modes defines agent types and their tool access permissions.
// Three agent types exist: code (full access), explore (read-only), review (read-only).
// The mode system controls which tools a spawned sub-agent can use.
package agent

import (
	"fmt"
	"strings"

	"github.com/spenc/savant-cli/internal/tools"
)

// AgentType represents the type of agent and its capabilities.
type AgentType int

const (
	// AgentTypeCode has full tool access: read, write, edit, bash, glob, grep.
	// This is the primary coding mode.
	AgentTypeCode AgentType = iota
	// AgentTypeExplore can only read and search files: read, glob, grep.
	// Cannot modify files or execute shell commands.
	AgentTypeExplore
	// AgentTypeReview can only read and search files: read, glob, grep.
	// Identical to explore but with a different system prompt focused on review.
	AgentTypeReview
	// AgentTypeDebug can read files and run read-only bash commands.
	// This is for diagnosing issues without modifying anything.
	AgentTypeDebug
	// AgentTypeAsk can only read and search files.
	// Strictly read-only for Q&A.
	AgentTypeAsk
)

// String returns the human-readable name of the agent type.
func (a AgentType) String() string {
	switch a {
	case AgentTypeCode:
		return "code"
	case AgentTypeExplore:
		return "explore"
	case AgentTypeReview:
		return "review"
	case AgentTypeDebug:
		return "debug"
	case AgentTypeAsk:
		return "ask"
	default:
		return "unknown"
	}
}

// ParseAgentType parses a string into an AgentType.
// Accepts: "code", "explore", "review", "debug", "ask" (case-insensitive).
func ParseAgentType(s string) (AgentType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "code":
		return AgentTypeCode, nil
	case "explore":
		return AgentTypeExplore, nil
	case "review":
		return AgentTypeReview, nil
	case "debug":
		return AgentTypeDebug, nil
	case "ask":
		return AgentTypeAsk, nil
	default:
		return AgentTypeCode, fmt.Errorf("unknown agent type %q: valid types are: code, explore, review, debug, ask", s)
	}
}

// AgentTypeFromTask attempts to auto-detect the best agent type from a task description.
func AgentTypeFromTask(task string) AgentType {
	lower := strings.ToLower(task)

	// Review keywords
	reviewKeywords := []string{"review", "audit", "check", "inspect", "verify", "validate"}
	for _, kw := range reviewKeywords {
		if strings.Contains(lower, kw) {
			return AgentTypeReview
		}
	}

	// Explore keywords
	exploreKeywords := []string{"explore", "find", "search", "lookup", "investigate", "research", "discover", "what is", "where is"}
	for _, kw := range exploreKeywords {
		if strings.Contains(lower, kw) {
			return AgentTypeExplore
		}
	}

	// Debug keywords
	debugKeywords := []string{"debug", "diagnose", "fix", "repair", "bug", "error", "crash", "issue"}
	for _, kw := range debugKeywords {
		if strings.Contains(lower, kw) {
			return AgentTypeDebug
		}
	}

	// Ask keywords
	askKeywords := []string{"explain", "what", "why", "how", "when", "question", "tell me"}
	for _, kw := range askKeywords {
		if strings.Contains(lower, kw) {
			return AgentTypeAsk
		}
	}

	// Default to code for implementation tasks
	return AgentTypeCode
}

// AllowedToolKinds returns the tool side-effect kinds this agent type is allowed to use.
// code: all kinds
// explore, review, ask: read-only (KindRead, KindSearch)
// debug: read + search (no write or execute)
func (a AgentType) AllowedToolKinds() []tools.ToolKind {
	switch a {
	case AgentTypeCode:
		return []tools.ToolKind{tools.KindRead, tools.KindSearch, tools.KindWrite, tools.KindExecute}
	case AgentTypeExplore:
		return []tools.ToolKind{tools.KindRead, tools.KindSearch}
	case AgentTypeReview:
		return []tools.ToolKind{tools.KindRead, tools.KindSearch}
	case AgentTypeDebug:
		return []tools.ToolKind{tools.KindRead, tools.KindSearch}
	case AgentTypeAsk:
		return []tools.ToolKind{tools.KindRead, tools.KindSearch}
	default:
		return []tools.ToolKind{tools.KindRead}
	}
}

// IsToolAllowed checks if a specific tool is allowed for this agent type.
func (a AgentType) IsToolAllowed(tool tools.Tool) bool {
	allowedKinds := a.AllowedToolKinds()
	toolKind := tool.Kind()

	for _, kind := range allowedKinds {
		if kind == toolKind {
			return true
		}
	}
	return false
}

// FilterTools filters a slice of tools to only those allowed by this agent type.
func (a AgentType) FilterTools(allTools []tools.Tool) []tools.Tool {
	var filtered []tools.Tool
	for _, t := range allTools {
		if a.IsToolAllowed(t) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// DefaultMaxTurns returns the default maximum turns for this agent type.
// code: 20, explore: 10, review: 15, debug: 15, ask: 5
func (a AgentType) DefaultMaxTurns() int {
	switch a {
	case AgentTypeCode:
		return 20
	case AgentTypeExplore:
		return 10
	case AgentTypeReview:
		return 15
	case AgentTypeDebug:
		return 15
	case AgentTypeAsk:
		return 5
	default:
		return 20
	}
}

// Description returns a human-readable description of the agent type.
func (a AgentType) Description() string {
	switch a {
	case AgentTypeCode:
		return "Full access: read, write, edit, bash, glob, grep. Use for implementation tasks."
	case AgentTypeExplore:
		return "Read-only: read, glob, grep. Use for codebase exploration and research."
	case AgentTypeReview:
		return "Read-only: read, glob, grep. Use for code review and auditing."
	case AgentTypeDebug:
		return "Read + diagnostics: read, glob, grep, bash (read-only). Use for diagnosing issues."
	case AgentTypeAsk:
		return "Read-only Q&A: read, glob, grep. Use for questions without modifications."
	default:
		return "Unknown agent type."
	}
}
