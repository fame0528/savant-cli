package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Runner executes hook commands and aggregates their results.
type Runner struct {
	hooks      []compiledHook
	cwd        string
	projectDir string
}

type compiledHook struct {
	cfg     HookConfig
	matcher *regexp.Regexp // nil = match everything
}

// NewRunner creates a Runner from the given hook configs.
func NewRunner(hooks []HookConfig, cwd, projectDir string) *Runner {
	compiled := make([]compiledHook, 0, len(hooks))
	for _, h := range hooks {
		ch := compiledHook{cfg: h}
		if h.Matcher != "" {
			re, err := regexp.Compile(h.Matcher)
			if err != nil {
				slog.Warn("Skipping hook with invalid matcher", "hook", h.Name, "error", err)
				continue
			}
			ch.matcher = re
		}
		compiled = append(compiled, ch)
	}
	return &Runner{hooks: compiled, cwd: cwd, projectDir: projectDir}
}

// Run executes all matching hooks for the given event and tool name.
func (r *Runner) Run(ctx context.Context, event, toolName string, toolInput []byte) AggregatedResult {
	var matching []compiledHook
	for _, h := range r.hooks {
		if h.cfg.Event != "" && h.cfg.Event != event {
			continue
		}
		if h.matcher != nil && !h.matcher.MatchString(toolName) {
			continue
		}
		matching = append(matching, h)
	}

	if len(matching) == 0 {
		return AggregatedResult{Decision: DecisionNone}
	}

	// Run hooks concurrently, collect results
	results := make([]HookResult, len(matching))
	for i, h := range matching {
		results[i] = r.runOne(ctx, h, toolName, toolInput)
	}

	return Aggregate(results)
}

// runOne executes a single hook command.
func (r *Runner) runOne(ctx context.Context, h compiledHook, toolName string, toolInput []byte) HookResult {
	timeout := time.Duration(h.cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build stdin payload
	payload := buildPayload(h.cfg.Event, "", r.cwd, toolName, string(toolInput))

	// Build environment
	env := buildEnv(h.cfg.Event, toolName, "", r.cwd, r.projectDir, string(toolInput))

	cmd := shellCommand(hookCtx, h.cfg.Command)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = env
	cmd.Dir = r.cwd

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	hr := HookResult{HookName: h.cfg.Name}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 2:
				hr.Decision = DecisionDeny
				hr.Reason = outputStr
				return hr
			case HaltExitCode:
				hr.Decision = DecisionDeny
				hr.Reason = fmt.Sprintf("HALT: %s", outputStr)
				return hr
			default:
				// Non-blocking error, hook has no opinion
				slog.Warn("Hook exited with error", "hook", h.cfg.Name, "exit_code", exitErr.ExitCode(), "output", outputStr)
				return hr
			}
		}
		slog.Warn("Hook execution failed", "hook", h.cfg.Name, "error", err)
		return hr
	}

	// Exit code 0: parse stdout for structured output
	hr = parseStdout(h.cfg.Name, outputStr)
	return hr
}

// buildPayload constructs the JSON stdin payload for a hook command.
func buildPayload(event, sessionID, cwd, toolName, toolInputJSON string) []byte {
	p := map[string]interface{}{
		"event":      event,
		"session_id": sessionID,
		"cwd":        cwd,
		"tool_name":  toolName,
		"tool_input": json.RawMessage(toolInputJSON),
	}
	data, _ := json.Marshal(p)
	return data
}

// buildEnv constructs the environment variables for a hook command.
func buildEnv(event, toolName, sessionID, cwd, projectDir, toolInputJSON string) []string {
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("SAVANT_EVENT=%s", event),
		fmt.Sprintf("SAVANT_TOOL_NAME=%s", toolName),
		fmt.Sprintf("SAVANT_SESSION_ID=%s", sessionID),
		fmt.Sprintf("SAVANT_CWD=%s", cwd),
		fmt.Sprintf("SAVANT_PROJECT_DIR=%s", projectDir),
	)
	return env
}

// parseStdout parses the hook's stdout as JSON for structured output.
func parseStdout(hookName, output string) HookResult {
	hr := HookResult{HookName: hookName, Decision: DecisionAllow}

	if output == "" {
		return hr
	}

	// Try JSON parsing
	var parsed struct {
		Decision string   `json:"decision"`
		Reason   string   `json:"reason"`
		Context  []string `json:"context"`
		Input    string   `json:"updated_input"`
	}

	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		// Not JSON - treat as context
		hr.Context = []string{output}
		return hr
	}

	switch strings.ToLower(parsed.Decision) {
	case "deny":
		hr.Decision = DecisionDeny
	case "allow":
		hr.Decision = DecisionAllow
	}

	hr.Reason = parsed.Reason
	hr.Context = parsed.Context
	hr.UpdatedInput = parsed.Input

	return hr
}

// shellCommand creates an exec.Cmd that runs the given command string
// in the appropriate shell for the current platform.
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}
