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
)

// CuratorConfig configures the background skill lifecycle manager.
type CuratorConfig struct {
	Enabled          bool
	IntervalHours    int // Default: 168 (7 days)
	MinIdleHours     int // Default: 2
	StaleAfterDays   int // Default: 30
	ArchiveAfterDays int // Default: 90
}

// DefaultCuratorConfig returns sensible defaults.
func DefaultCuratorConfig() CuratorConfig {
	return CuratorConfig{
		Enabled:          true,
		IntervalHours:    168,
		MinIdleHours:     2,
		StaleAfterDays:   30,
		ArchiveAfterDays: 90,
	}
}

// CuratorState tracks the curator's run history.
type CuratorState struct {
	LastRunAt       *time.Time `json:"last_run_at"`
	LastRunDuration float64    `json:"last_run_duration_seconds"`
	LastRunSummary  string     `json:"last_run_summary"`
	Paused          bool       `json:"paused"`
	RunCount        int        `json:"run_count"`
}

// Curator manages the lifecycle of agent-created skills.
type Curator struct {
	config    CuratorConfig
	skillsDir string
	store     *UsageStore
	state     CuratorState
}

// NewCurator creates a new curator.
func NewCurator(config CuratorConfig, skillsDir string, store *UsageStore) *Curator {
	c := &Curator{
		config:    config,
		skillsDir: skillsDir,
		store:     store,
	}
	c.loadState()
	return c
}

// ShouldRunNow returns true if the curator should run.
func (c *Curator) ShouldRunNow() bool {
	if !c.config.Enabled || c.state.Paused {
		return false
	}

	// Check if we have any agent-created skills to maintain
	agentSkills := c.store.AgentCreatedSkills()
	if len(agentSkills) == 0 {
		return false
	}

	// Check interval
	if c.state.LastRunAt != nil {
		elapsed := time.Since(*c.state.LastRunAt)
		interval := time.Duration(c.config.IntervalHours) * time.Hour
		if elapsed < interval {
			return false
		}
	}

	return true
}

// Run executes the curator lifecycle management.
func (c *Curator) Run(ctx context.Context, consolidateFn func(ctx context.Context, prompt string) (string, error)) error {
	start := time.Now()
	slog.Info("Curator starting", "skills_dir", c.skillsDir)

	// Phase 1: Automatic state transitions (pure time-based, no LLM)
	transitioned := c.applyAutomaticTransitions()
	slog.Info("Curator auto-transitions", "count", transitioned)

	// Phase 2: LLM consolidation pass
	if consolidateFn != nil {
		summary, err := c.runConsolidation(ctx, consolidateFn)
		if err != nil {
			slog.Warn("Curator consolidation failed", "error", err)
		} else {
			c.state.LastRunSummary = summary
		}
	}

	// Update state
	now := time.Now()
	c.state.LastRunAt = &now
	c.state.LastRunDuration = time.Since(start).Seconds()
	c.state.RunCount++
	c.saveState()

	slog.Info("Curator completed", "duration", c.state.LastRunDuration, "transitions", transitioned)
	return nil
}

// applyAutomaticTransitions moves skills through lifecycle states based on activity.
func (c *Curator) applyAutomaticTransitions() int {
	count := 0
	records := c.store.AllRecords()
	now := time.Now()

	staleThreshold := time.Duration(c.config.StaleAfterDays) * 24 * time.Hour
	archiveThreshold := time.Duration(c.config.ArchiveAfterDays) * 24 * time.Hour

	for name, rec := range records {
		// Only touch agent-created skills
		if rec.CreatedBy != "agent" {
			continue
		}
		// Pinned skills are exempt
		if rec.Pinned {
			continue
		}

		// Find the most recent activity
		lastActivity := rec.LastUsedAt
		if rec.LastViewedAt.After(lastActivity) {
			lastActivity = rec.LastViewedAt
		}
		if rec.LastPatchedAt.After(lastActivity) {
			lastActivity = rec.LastPatchedAt
		}

		elapsed := now.Sub(lastActivity)

		switch rec.State {
		case "active":
			if elapsed > staleThreshold {
				c.store.SetState(name, "stale")
				count++
				slog.Info("Curator: skill became stale", "skill", name, "idle_days", int(elapsed.Hours()/24))
			}
		case "stale":
			if elapsed > archiveThreshold {
				c.archiveSkill(name)
				count++
				slog.Info("Curator: skill archived", "skill", name, "idle_days", int(elapsed.Hours()/24))
			}
		}
	}

	return count
}

// archiveSkill moves a skill to the .archive/ directory.
func (c *Curator) archiveSkill(name string) {
	archiveDir := filepath.Join(c.skillsDir, ".archive")
	os.MkdirAll(archiveDir, 0o755)

	skillDir := filepath.Join(c.skillsDir, name)
	destDir := filepath.Join(archiveDir, name)

	if err := os.Rename(skillDir, destDir); err != nil {
		slog.Warn("Curator: failed to archive skill", "skill", name, "error", err)
		return
	}

	c.store.SetState(name, "archived")
}

// runConsolidation runs the LLM consolidation pass.
func (c *Curator) runConsolidation(ctx context.Context, consolidateFn func(ctx context.Context, prompt string) (string, error)) (string, error) {
	// Build the consolidation prompt with current agent-created skills
	agentSkills := c.store.AgentCreatedSkills()
	if len(agentSkills) == 0 {
		return "", nil
	}

	// Read each skill's content
	var skillSummaries []string
	for _, name := range agentSkills {
		state := c.store.GetState(name)
		if state == "archived" {
			continue
		}
		skillPath := filepath.Join(c.skillsDir, name, SkillFileName)
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}
		skillSummaries = append(skillSummaries, fmt.Sprintf("--- %s (state: %s) ---\n%s", name, state, string(data)))
	}

	if len(skillSummaries) == 0 {
		return "No active agent-created skills to consolidate.", nil
	}

	prompt := fmt.Sprintf(`%s

Current agent-created skills:

%s

Analyze these skills. If any skills share a domain keyword and could be merged
into a broader "umbrella" skill, create the umbrella skill and archive the narrow ones.
If any skills are too narrow or redundant, consolidate them.
If all skills are well-organized and distinct, respond with "No changes needed."

When creating or editing skills, use the skill_manage tool.
When archiving skills, use the skill_manage(action="delete") tool.

Respond with a brief summary of what you changed or "No changes needed."`,
		curatorPrompt, strings.Join(skillSummaries, "\n\n"))

	result, err := consolidateFn(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("consolidation LLM call: %w", err)
	}
	return result, nil
}

// State file management

func (c *Curator) stateFile() string {
	return filepath.Join(c.skillsDir, ".curator_state")
}

func (c *Curator) loadState() {
	data, err := os.ReadFile(c.stateFile())
	if err != nil {
		return
	}
	json.Unmarshal(data, &c.state)
}

func (c *Curator) saveState() {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return
	}
	tmpPath := c.stateFile() + ".tmp"
	os.WriteFile(tmpPath, data, 0o644)
	os.Rename(tmpPath, c.stateFile())
}

// curatorPrompt is the system prompt for the curator's LLM consolidation pass.
const curatorPrompt = `You are a skill curator for the Savant AI coding assistant.
Your job is to review agent-created skills and maintain the collection.

Rules:
1. Only touch agent-created skills. Never modify bundled or user-installed skills.
2. Never delete skills. Only archive them (move to .archive/).
3. Pinned skills are exempt from all operations.
4. When merging skills, create an umbrella skill that combines the knowledge.
5. Move narrow content to references/ or templates/ subdirectories.
6. Skills must be broadly applicable, not run-specific.
7. Descriptions must be <= 60 characters, one sentence, ending with a period.

Your goal is to keep the skill collection organized, non-redundant, and high-quality.`
