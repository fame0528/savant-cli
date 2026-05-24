package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

type BashTool struct{}

func NewBashTool() *BashTool { return &BashTool{} }

func (t *BashTool) Name() string        { return "bash" }
func (t *BashTool) Description() string { return "Execute a shell command and return its output. Use run_in_background=true for long-running commands." }
func (t *BashTool) Kind() ToolKind      { return KindExecute }

func (t *BashTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			},
			"timeout": {
				"type": "integer",
				"description": "Timeout in seconds (default 120)"
			},
			"run_in_background": {
				"type": "boolean",
				"description": "Set to true to run this command in the background. Use job_output to read output later."
			}
		},
		"required": ["command"]
	}`)
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Command         string `json:"command"`
		Timeout         int    `json:"timeout"`
		RunInBackground bool   `json:"run_in_background"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	// Background mode: start and return job ID immediately
	if params.RunInBackground {
		job, err := GetGlobalJobManager().Start(params.Command, "")
		if err != nil {
			return "", fmt.Errorf("background start failed: %w", err)
		}
		return fmt.Sprintf("Background job started.\nJob ID: %s\nPID: %d\n\nUse job_output to read output, job_kill to terminate.", job.ID, job.PID), nil
	}

	if params.Timeout == 0 {
		params.Timeout = 120
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, "powershell", "-Command", params.Command)
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", params.Command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output) + "\n" + err.Error(), nil
	}
	return string(output), nil
}
