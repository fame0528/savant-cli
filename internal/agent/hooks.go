package agent

import (
	"context"
	"os/exec"
	"regexp"
	"time"
)

// HookEvent represents when a hook fires.
type HookEvent int

const (
	HookPreTool  HookEvent = iota // Before tool execution
	HookPostTool                  // After tool execution
	HookPreAgent                  // Before agent turn
	HookPostAgent                 // After agent turn
)

// HookDecision is the result of a hook evaluation.
type HookDecision int

const (
	HookAllow HookDecision = iota
	HookDeny
	HookHalt // Halt the entire turn
)

// HookConfig defines a hook rule.
type HookConfig struct {
	Name    string        `json:"name"`
	Matcher string        `json:"matcher"` // Regex to match tool name
	Command string        `json:"command"` // Shell command to run
	Timeout time.Duration `json:"timeout"` // Default 30s
	Event   HookEvent     `json:"event"`
}

// HookResult is the outcome of running a hook.
type HookResult struct {
	HookName string       `json:"hook_name"`
	Decision HookDecision `json:"decision"`
	Reason   string       `json:"reason,omitempty"`
	Output   string       `json:"output,omitempty"`
}

// HookRunner executes hooks and aggregates results.
type HookRunner struct {
	hooks []HookConfig
}

// NewHookRunner creates a hook runner with the given hooks.
func NewHookRunner(hooks []HookConfig) *HookRunner {
	return &HookRunner{hooks: hooks}
}

// Run executes all matching hooks for a given event and tool name.
// Returns the aggregated decision (deny > allow > none; halt is sticky).
func (hr *HookRunner) Run(ctx context.Context, event HookEvent, toolName string, input map[string]interface{}) HookResult {
	result := HookResult{Decision: HookAllow}

	for _, hook := range hr.hooks {
		if hook.Event != event {
			continue
		}

		// Check matcher
		if hook.Matcher != "" {
			re, err := regexp.Compile(hook.Matcher)
			if err != nil {
				continue
			}
			if !re.MatchString(toolName) {
				continue
			}
		}

		// Run hook command
		timeout := hook.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		hookCtx, cancel := context.WithTimeout(ctx, timeout)

		cmd := exec.CommandContext(hookCtx, "sh", "-c", hook.Command)
		output, err := cmd.CombinedOutput()
		cancel()

		hr := HookResult{
			HookName: hook.Name,
			Output:   string(output),
		}

		if err != nil {
			// Exit code 2 = block, Exit code 49 = halt
			if exitErr, ok := err.(*exec.ExitError); ok {
				switch exitErr.ExitCode() {
				case 2:
					hr.Decision = HookDeny
					hr.Reason = string(output)
				case 49:
					hr.Decision = HookHalt
					hr.Reason = string(output)
				default:
					continue // Non-blocking error
				}
			} else {
				continue
			}
		} else {
			// Exit 0: parse stdout for decision
			hr.Decision = HookAllow
		}

		// Aggregate: deny > allow; halt is sticky
		if hr.Decision == HookHalt {
			result.Decision = HookHalt
			result.Reason = hr.Reason
			result.HookName = hr.HookName
			return result
		}
		if hr.Decision == HookDeny && result.Decision != HookHalt {
			result.Decision = HookDeny
			result.Reason = hr.Reason
			result.HookName = hr.HookName
		}
	}

	return result
}
