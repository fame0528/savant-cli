// Package agent - Orchestrator manages parallel sub-agent execution.
// Uses errgroup.Group to spawn multiple sub-agents concurrently, collects
// all results (successes and failures), and merges them back into the blackboard.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// OrchestratorConfig configures the parallel orchestration behavior.
type OrchestratorConfig struct {
	// MaxParallel limits how many sub-agents run simultaneously.
	// If 0, defaults to 3.
	MaxParallel int
	// TimeoutPerAgent is the maximum duration for each sub-agent.
	// If 0, no per-agent timeout.
	TimeoutPerAgent time.Duration
	// ContinueOnError determines whether other agents continue when one fails.
	ContinueOnError bool
}

// DefaultOrchestratorConfig returns sensible defaults.
func DefaultOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		MaxParallel:     3,
		TimeoutPerAgent: 5 * time.Minute,
		ContinueOnError: true,
	}
}

// Orchestrator manages parallel sub-agent execution.
type Orchestrator struct {
	config OrchestratorConfig
}

// NewOrchestrator creates a new orchestrator with the given config.
func NewOrchestrator(config OrchestratorConfig) *Orchestrator {
	if config.MaxParallel <= 0 {
		config.MaxParallel = 3
	}
	return &Orchestrator{config: config}
}

// SpawnRequest defines a single sub-agent to spawn.
type SpawnRequest struct {
	// AgentType for this sub-agent.
	AgentType AgentType
	// Task description for this sub-agent.
	Task string
	// MaxTurns for this sub-agent (0 = use type default).
	MaxTurns int
	// AgentID is an optional unique identifier. Generated if empty.
	AgentID string
}

// SpawnResult holds the result of a single sub-agent spawn.
type SpawnResult struct {
	Request  SpawnRequest
	Result   SubAgentResult
	Duration time.Duration
	Err      error
}

// SpawnParallel spawns multiple sub-agents in parallel and collects results.
// It uses errgroup.Group with a semaphore to limit concurrency.
// All results are collected (successes and failures) and returned.
func (o *Orchestrator) SpawnParallel(ctx context.Context, requests []SpawnRequest, runFn func(context.Context, SpawnRequest) SubAgentResult) []SpawnResult {
	if len(requests) == 0 {
		return nil
	}

	results := make([]SpawnResult, len(requests))
	var mu sync.Mutex

	// Create errgroup with limited concurrency
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(o.config.MaxParallel)

	for i, req := range requests {
		i, req := i, req // Capture loop variables
		g.Go(func() error {
			// Create per-agent timeout context if configured
			agentCtx := gctx
			var cancel context.CancelFunc
			if o.config.TimeoutPerAgent > 0 {
				agentCtx, cancel = context.WithTimeout(gctx, o.config.TimeoutPerAgent)
				defer cancel()
			}

			start := time.Now()
			result := runFn(agentCtx, req)
			duration := time.Since(start)

			mu.Lock()
			results[i] = SpawnResult{
				Request:  req,
				Result:   result,
				Duration: duration,
				Err:      result.Error,
			}
			mu.Unlock()

			// If ContinueOnError is false and agent errored, return error
			if !o.config.ContinueOnError && result.Error != nil {
				return fmt.Errorf("agent %s failed: %w", req.AgentID, result.Error)
			}

			return nil
		})
	}

	// Wait for all agents to complete
	if err := g.Wait(); err != nil {
		slog.Warn("Orchestrator: some agents had errors", "error", err)
	}

	return results
}

// MergeResults merges sub-agent results back into the parent's blackboard.
// It adds files_modified, files_read, and decisions from each result into the blackboard.
func MergeResults(bb *Blackboard, results []SpawnResult, parentID string) {
	if bb == nil {
		return
	}

	for _, sr := range results {
		if sr.Err != nil {
			continue
		}
		if sr.Result.Error != nil {
			continue
		}

		// Merge files modified
		for _, f := range sr.Result.FilesModified {
			bb.Append(BlackboardFilesModified, f, parentID)
		}

		// Merge files read
		for _, f := range sr.Result.FilesRead {
			bb.Append(BlackboardFilesRead, f, parentID)
		}

		// Merge decisions
		for _, d := range sr.Result.Decisions {
			bb.Append(BlackboardDecisions, d, parentID)
		}
	}
}

// FormatResults formats a set of spawn results into a readable string
// suitable for injecting into the parent's conversation.
func FormatResults(results []SpawnResult) string {
	if len(results) == 0 {
		return "No sub-agents were spawned."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sub-agent results (%d total):\n", len(results)))
	sb.WriteString("─────────────────────────────────────\n")

	for i, sr := range results {
		sb.WriteString(fmt.Sprintf("\n[%d/%d] %s (%s)\n", i+1, len(results), sr.Request.AgentID, sr.Request.AgentType))

		if sr.Err != nil {
			sb.WriteString(fmt.Sprintf("  Status: FAILED - %s\n", sr.Err))
			continue
		}

		if sr.Result.Error != nil {
			sb.WriteString(fmt.Sprintf("  Status: ERROR - %s\n", sr.Result.Error))
		} else {
			sb.WriteString(fmt.Sprintf("  Status: COMPLETED (%s, %d turns)\n", formatDuration(sr.Duration), sr.Result.TurnCount))
		}

		if len(sr.Result.FilesModified) > 0 {
			sb.WriteString(fmt.Sprintf("  Files modified: %d\n", len(sr.Result.FilesModified)))
		}
		if len(sr.Result.FilesRead) > 0 {
			sb.WriteString(fmt.Sprintf("  Files read: %d\n", len(sr.Result.FilesRead)))
		}
	}

	return sb.String()
}
