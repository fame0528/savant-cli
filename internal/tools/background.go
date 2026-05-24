package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// BackgroundJobManager tracks running background processes.
type BackgroundJobManager struct {
	mu   sync.RWMutex
	jobs map[string]*BackgroundJob
}

// BackgroundJob represents a running background process.
type BackgroundJob struct {
	ID          string
	Command     string
	WorkingDir  string
	PID         int
	StartTime   time.Time
	EndTime     *time.Time
	Status      string // "running", "completed", "killed", "failed"
	ExitCode    int
	Stdout      bytes.Buffer
	Stderr      bytes.Buffer
	cmd         *exec.Cmd
	done        chan struct{}
}

// NewBackgroundJobManager creates a new manager.
func NewBackgroundJobManager() *BackgroundJobManager {
	return &BackgroundJobManager{
		jobs: make(map[string]*BackgroundJob),
	}
}

// Start begins a background command. Returns immediately with a job ID.
func (bjm *BackgroundJobManager) Start(command, workingDir string) (*BackgroundJob, error) {
	bjm.mu.Lock()
	defer bjm.mu.Unlock()

	id := fmt.Sprintf("job_%d", time.Now().UnixNano())

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = workingDir
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	job := &BackgroundJob{
		ID:        id,
		Command:   command,
		WorkingDir: workingDir,
		StartTime: time.Now(),
		Status:    "running",
		cmd:       cmd,
		done:      make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		job.Status = "failed"
		job.ExitCode = -1
		return job, fmt.Errorf("start command: %w", err)
	}

	job.PID = cmd.Process.Pid
	bjm.jobs[id] = job

	// Monitor completion in background
	go func() {
		err := cmd.Wait()
		now := time.Now()
		job.EndTime = &now
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				job.ExitCode = exitErr.ExitCode()
			}
			job.Status = "failed"
		} else {
			job.ExitCode = 0
			job.Status = "completed"
		}
		close(job.done)
	}()

	return job, nil
}

// Get returns a job by ID.
func (bjm *BackgroundJobManager) Get(id string) (*BackgroundJob, bool) {
	bjm.mu.RLock()
	defer bjm.mu.RUnlock()
	job, ok := bjm.jobs[id]
	return job, ok
}

// List returns all jobs.
func (bjm *BackgroundJobManager) List() []*BackgroundJob {
	bjm.mu.RLock()
	defer bjm.mu.RUnlock()
	var jobs []*BackgroundJob
	for _, j := range bjm.jobs {
		jobs = append(jobs, j)
	}
	return jobs
}

// Kill terminates a running job.
func (bjm *BackgroundJobManager) Kill(id string) error {
	bjm.mu.RLock()
	job, ok := bjm.jobs[id]
	bjm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("job %s not found", id)
	}
	if job.Status != "running" {
		return fmt.Errorf("job %s is not running (status: %s)", id, job.Status)
	}
	job.Status = "killed"
	if job.cmd != nil && job.cmd.Process != nil {
		return job.cmd.Process.Kill()
	}
	return nil
}

// Output returns the stdout+stderr of a job.
func (bjm *BackgroundJobManager) Output(id string, tail int) (string, error) {
	bjm.mu.RLock()
	job, ok := bjm.jobs[id]
	bjm.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("job %s not found", id)
	}

	stdout := job.cmd.Stdout.(*bytes.Buffer).String()
	stderr := job.cmd.Stderr.(*bytes.Buffer).String()

	output := stdout
	if stderr != "" {
		output += "\n" + stderr
	}

	if tail > 0 {
		lines := strings.Split(output, "\n")
		if len(lines) > tail {
			lines = lines[len(lines)-tail:]
		}
		output = strings.Join(lines, "\n")
	}

	return output, nil
}

// Cleanup removes completed/killed/failed jobs older than maxAge.
func (bjm *BackgroundJobManager) Cleanup(maxAge time.Duration) {
	bjm.mu.Lock()
	defer bjm.mu.Unlock()
	for id, job := range bjm.jobs {
		if job.Status != "running" && job.EndTime != nil {
			if time.Since(*job.EndTime) > maxAge {
				delete(bjm.jobs, id)
			}
		}
	}
}

// Global instance
var globalJobManager *BackgroundJobManager
var jobManagerOnce sync.Once

func GetGlobalJobManager() *BackgroundJobManager {
	jobManagerOnce.Do(func() {
		globalJobManager = NewBackgroundJobManager()
	})
	return globalJobManager
}

// ─────────────────────────────────────────────────────────────
// Job Output Tool
// ─────────────────────────────────────────────────────────────

type JobOutputTool struct {
	manager *BackgroundJobManager
}

func NewJobOutputTool(manager *BackgroundJobManager) *JobOutputTool {
	return &JobOutputTool{manager: manager}
}

func (t *JobOutputTool) Name() string        { return "job_output" }
func (t *JobOutputTool) Description() string  { return "Read output from a background job. Returns stdout and stderr." }
func (t *JobOutputTool) Kind() ToolKind       { return KindRead }
func (t *JobOutputTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"job_id": {
				"type": "string",
				"description": "The job ID returned by bash with run_in_background"
			},
			"tail": {
				"type": "integer",
				"description": "Number of trailing lines to return (default: all)"
			}
		},
		"required": ["job_id"]
	}`)
}

func (t *JobOutputTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		JobID string `json:"job_id"`
		Tail  int    `json:"tail"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	job, ok := t.manager.Get(params.JobID)
	if !ok {
		return "", fmt.Errorf("job %s not found", params.JobID)
	}

	output, err := t.manager.Output(params.JobID, params.Tail)
	if err != nil {
		return "", err
	}

	status := job.Status
	if status == "running" {
		select {
		case <-job.done:
			status = job.Status
		default:
			status = "running"
		}
	}

	return fmt.Sprintf("Job %s [%s] (exit code: %d)\n\n%s", params.JobID, status, job.ExitCode, output), nil
}

// ─────────────────────────────────────────────────────────────
// Job Kill Tool
// ─────────────────────────────────────────────────────────────

type JobKillTool struct {
	manager *BackgroundJobManager
}

func NewJobKillTool(manager *BackgroundJobManager) *JobKillTool {
	return &JobKillTool{manager: manager}
}

func (t *JobKillTool) Name() string        { return "job_kill" }
func (t *JobKillTool) Description() string  { return "Kill a running background job." }
func (t *JobKillTool) Kind() ToolKind       { return KindExecute }
func (t *JobKillTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"job_id": {
				"type": "string",
				"description": "The job ID to kill"
			}
		},
		"required": ["job_id"]
	}`)
}

func (t *JobKillTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	if err := t.manager.Kill(params.JobID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Job %s killed.", params.JobID), nil
}

// ─────────────────────────────────────────────────────────────
// Job List Tool
// ─────────────────────────────────────────────────────────────

type JobListTool struct {
	manager *BackgroundJobManager
}

func NewJobListTool(manager *BackgroundJobManager) *JobListTool {
	return &JobListTool{manager: manager}
}

func (t *JobListTool) Name() string        { return "job_list" }
func (t *JobListTool) Description() string  { return "List all background jobs and their status." }
func (t *JobListTool) Kind() ToolKind       { return KindRead }
func (t *JobListTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": []
	}`)
}

func (t *JobListTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	jobs := t.manager.List()
	if len(jobs) == 0 {
		return "No background jobs.", nil
	}

	var sb strings.Builder
	sb.WriteString("Background jobs:\n")
	for _, j := range jobs {
		elapsed := time.Since(j.StartTime).Round(time.Second)
		sb.WriteString(fmt.Sprintf("  %s [%s] (PID: %d, %s) %s\n", j.ID, j.Status, j.PID, elapsed, j.Command))
	}
	return sb.String(), nil
}
