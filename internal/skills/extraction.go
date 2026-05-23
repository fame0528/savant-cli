package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spenc/savant-cli/internal/session"
)

// ExtractionConfig configures the session extraction agent.
type ExtractionConfig struct {
	Enabled          bool
	ThrottleMinutes  int // Default: 30
	MinIdleHours     int // Default: 3
	MinMessages      int // Default: 10
	MaxSkillsPerRun  int // Default: 2
	MaxTurns         int // Default: 30
	MaxMinutes       int // Default: 10
}

// DefaultExtractionConfig returns sensible defaults.
func DefaultExtractionConfig() ExtractionConfig {
	return ExtractionConfig{
		Enabled:          true,
		ThrottleMinutes:  30,
		MinIdleHours:     3,
		MinMessages:      10,
		MaxSkillsPerRun:  2,
		MaxTurns:         30,
		MaxMinutes:       10,
	}
}

// ExtractionState tracks the extraction agent's run history.
type ExtractionState struct {
	LastRunAt       *time.Time `json:"last_run_at"`
	LastRunDuration float64    `json:"last_run_duration_seconds"`
	LastRunSummary  string     `json:"last_run_summary"`
	RunCount        int        `json:"run_count"`
	ProcessedSessions []string `json:"processed_sessions"`
}

// ExtractionResult is the outcome of an extraction run.
type ExtractionResult struct {
	SkillsCreated int
	SkillsUpdated int
	InboxPatches  int
	Summary       string
}

// Extractor analyzes past sessions and extracts reusable skills.
type Extractor struct {
	config    ExtractionConfig
	skillsDir string
	sessionSvc SessionReader
	store     *UsageStore
	state     ExtractionState
}

// SessionReader provides access to session data for extraction.
type SessionReader interface {
	ListEligibleForExtraction(ctx context.Context, minMessages int, idleDuration time.Duration) ([]session.SessionSummary, error)
	GetMessageSummaries(ctx context.Context, sessionID string) ([]session.MessageSummary, error)
}

// NewExtractor creates a new session extractor.
func NewExtractor(config ExtractionConfig, skillsDir string, sessionSvc SessionReader, store *UsageStore) *Extractor {
	e := &Extractor{
		config:     config,
		skillsDir:  skillsDir,
		sessionSvc: sessionSvc,
		store:      store,
	}
	e.loadState()
	return e
}

// ShouldRunNow returns true if extraction should run.
func (e *Extractor) ShouldRunNow() bool {
	if !e.config.Enabled {
		return false
	}

	// Check throttle
	if e.state.LastRunAt != nil {
		elapsed := time.Since(*e.state.LastRunAt)
		throttle := time.Duration(e.config.ThrottleMinutes) * time.Minute
		if elapsed < throttle {
			return false
		}
	}

	return true
}

// Run executes the session extraction agent.
func (e *Extractor) Run(ctx context.Context, extractFn func(ctx context.Context, prompt string) (string, error)) (*ExtractionResult, error) {
	if !e.ShouldRunNow() {
		return nil, nil
	}

	start := time.Now()
	slog.Info("Extraction agent starting")

	// Find eligible sessions
	idleDuration := time.Duration(e.config.MinIdleHours) * time.Hour
	sessions, err := e.sessionSvc.ListEligibleForExtraction(ctx, e.config.MinMessages, idleDuration)
	if err != nil {
		return nil, fmt.Errorf("list eligible sessions: %w", err)
	}

	// Filter out already-processed sessions
	processed := make(map[string]bool)
	for _, id := range e.state.ProcessedSessions {
		processed[id] = true
	}
	var eligible []session.SessionSummary
	for _, s := range sessions {
		if !processed[s.ID] {
			eligible = append(eligible, s)
		}
	}

	if len(eligible) == 0 {
		slog.Info("Extraction: no eligible sessions")
		return nil, nil
	}

	// Build session index
	var sessionIndex []string
	for _, s := range eligible {
		sessionIndex = append(sessionIndex, fmt.Sprintf("- %s: %s (%d messages, idle since %s)",
			s.ID, s.Title, s.MessageCount, s.UpdatedAt.Format("2006-01-02 15:04")))
	}

	// Read transcripts for eligible sessions
	var transcripts []string
	for _, s := range eligible {
		messages, err := e.sessionSvc.GetMessageSummaries(ctx, s.ID)
		if err != nil {
			continue
		}
		var msgStrings []string
		for _, m := range messages {
			msgStrings = append(msgStrings, fmt.Sprintf("[%s] %s", m.Role, m.Content))
		}
		transcripts = append(transcripts, fmt.Sprintf("--- Session: %s ---\n%s", s.Title, strings.Join(msgStrings, "\n")))
	}

	// Build the extraction prompt
	existingSkills := e.store.AgentCreatedSkills()
	prompt := fmt.Sprintf(`%s

## Eligible Sessions
%s

## Session Transcripts
%s

## Existing Agent-Created Skills
%s

Analyze the session transcripts above. Extract reusable skills ONLY if they are:
1. Procedural (describe HOW to do something, not WHAT happened)
2. Durable (will be useful in future sessions, not just this one)
3. Evidence-backed (appeared in multiple sessions or was validated by success)
4. Project-specific (not generic knowledge the agent already has)

Default to NO SKILL. Aim for 0-%d skills per run.

When creating a skill, use the skill_manage tool:
skill_manage(action="create", name="skill-name", description="...", content="---\\nname: skill-name\\ndescription: ...\\n---\\nInstructions here")

If no skills are worth extracting, respond with "No skills extracted."

When done, respond with a brief summary of what you found/created.`,
		extractionPrompt,
		strings.Join(sessionIndex, "\n"),
		strings.Join(transcripts, "\n\n"),
		strings.Join(existingSkills, ", "),
		e.config.MaxSkillsPerRun)

	result, err := extractFn(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("extraction LLM call: %w", err)
	}

	// Record processed sessions
	for _, s := range eligible {
		e.state.ProcessedSessions = append(e.state.ProcessedSessions, s.ID)
	}

	// Update state
	now := time.Now()
	e.state.LastRunAt = &now
	e.state.LastRunDuration = time.Since(start).Seconds()
	e.state.LastRunSummary = result
	e.state.RunCount++
	e.saveState()

	slog.Info("Extraction completed", "duration", e.state.LastRunDuration, "sessions", len(eligible))
	return &ExtractionResult{
		Summary: result,
	}, nil
}

// State file management

func (e *Extractor) stateFile() string {
	return filepath.Join(e.skillsDir, ".extraction-state.json")
}

func (e *Extractor) loadState() {
	data, err := os.ReadFile(e.stateFile())
	if err != nil {
		return
	}
	json.Unmarshal(data, &e.state)
}

func (e *Extractor) saveState() {
	data, err := json.MarshalIndent(e.state, "", "  ")
	if err != nil {
		return
	}
	tmpPath := e.stateFile() + ".tmp"
	os.WriteFile(tmpPath, data, 0o644)
	os.Rename(tmpPath, e.stateFile())
}

// extractionPrompt is the system prompt for the extraction agent.
const extractionPrompt = `You are a skill extraction agent for the Savant AI coding assistant.
Your job is to analyze completed conversation sessions and extract reusable skills.

Rules:
1. Default to NO SKILL. Aim for 0-2 skills per run.
2. Only extract skills that are procedural, durable, evidence-backed, and project-specific.
3. Never include run-specific details (file paths, timestamps, user names).
4. Check existing skills before creating duplicates.
5. Skills must follow the Agent Skills format (SKILL.md with YAML frontmatter).
6. Description must be <= 60 characters, one sentence, ending with a period.
7. Skill name must be lowercase alphanumeric with hyphens.
8. When creating a skill, include step-by-step instructions.
9. When done, respond with a brief summary of what you found/created.

Quality bar: If in doubt, don't create the skill. Better to have fewer, high-quality skills than many mediocre ones.`
