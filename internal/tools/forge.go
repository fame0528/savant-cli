package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ForgeTool implements the tool_forge tool for creating/managing forged tools.
type ForgeTool struct {
	forgeDir   string
	registry   *Registry
	provenance *ProvenanceTracker
	quality    *QualityGate
}

// NewForgeTool creates a new forge tool.
func NewForgeTool(forgeDir string, registry *Registry, provenance *ProvenanceTracker) *ForgeTool {
	os.MkdirAll(forgeDir, 0o755)
	return &ForgeTool{
		forgeDir:   forgeDir,
		registry:   registry,
		provenance: provenance,
		quality:    NewQualityGate(),
	}
}

func (t *ForgeTool) Name() string { return "tool_forge" }
func (t *ForgeTool) Description() string {
	return "Create, update, and manage forged tools. Forged tools are reusable procedural knowledge that all agents can call. Actions: forge, patch, list, view, stats, archive, pin, rollback, rate."
}
func (t *ForgeTool) Kind() ToolKind { return KindWrite }

func (t *ForgeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["forge", "patch", "list", "view", "stats", "archive", "pin", "rollback", "rate"],
				"description": "The action to perform"
			},
			"name": {
				"type": "string",
				"description": "Tool name (kebab-case)"
			},
			"description": {
				"type": "string",
				"description": "One-sentence description of what the tool does"
			},
			"body": {
				"type": "string",
				"description": "SKILL.md body content (markdown instructions)"
			},
			"category": {
				"type": "string",
				"description": "Category (e.g., web, file, debug, test)"
			},
			"old_string": {
				"type": "string",
				"description": "Text to find (for patch action)"
			},
			"new_string": {
				"type": "string",
				"description": "Replacement text (for patch action)"
			},
			"version_bump": {
				"type": "string",
				"enum": ["patch", "minor", "major"],
				"description": "Version bump type (for patch action, default: patch)"
			},
			"reason": {
				"type": "string",
				"description": "Reason for archiving"
			},
			"superseded_by": {
				"type": "string",
				"description": "Name of the tool that replaces this one"
			},
			"pinned": {
				"type": "boolean",
				"description": "Whether to pin (exempt from auto-archive)"
			},
			"rating": {
				"type": "string",
				"enum": ["thumbs_up", "thumbs_down"],
				"description": "Rating for the tool"
			},
			"comment": {
				"type": "string",
				"description": "Optional comment with rating"
			},
			"target_version": {
				"type": "string",
				"description": "Target version for rollback"
			}
		},
		"required": ["action"]
	}`)
}

func (t *ForgeTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action       string `json:"action"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Body         string `json:"body"`
		Category     string `json:"category"`
		OldString    string `json:"old_string"`
		NewString    string `json:"new_string"`
		VersionBump  string `json:"version_bump"`
		Reason       string `json:"reason"`
		SupersededBy string `json:"superseded_by"`
		Pinned       *bool  `json:"pinned"`
		Rating       string `json:"rating"`
		Comment      string `json:"comment"`
		TargetVersion string `json:"target_version"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	switch params.Action {
	case "forge":
		return t.handleForge(params.Name, params.Description, params.Body, params.Category)
	case "patch":
		return t.handlePatch(params.Name, params.OldString, params.NewString, params.VersionBump)
	case "list":
		return t.handleList()
	case "view":
		return t.handleView(params.Name)
	case "stats":
		return t.handleStats(params.Name)
	case "archive":
		return t.handleArchive(params.Name, params.Reason, params.SupersededBy)
	case "pin":
		return t.handlePin(params.Name, params.Pinned)
	case "rollback":
		return t.handleRollback(params.Name, params.TargetVersion)
	case "rate":
		return t.handleRate(params.Name, params.Rating, params.Comment)
	default:
		return "", fmt.Errorf("unknown action: %s", params.Action)
	}
}

func (t *ForgeTool) handleForge(name, description, body, category string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if description == "" {
		return "", fmt.Errorf("description is required")
	}
	if body == "" {
		return "", fmt.Errorf("body is required")
	}

	// Quality gate
	existing := t.existingNames()
	qr := t.quality.Validate(name, description, "0.1.0", body, existing)
	if !qr.Passed {
		msg := "Quality gate rejected:\n"
		for _, f := range qr.Failures {
			msg += "  - [" + f.Code + "] " + f.Detail + "\n"
		}
		return msg, nil
	}

	// Create directory and SKILL.md
	toolDir := filepath.Join(t.forgeDir, name)
	os.MkdirAll(toolDir, 0o755)

	skillMd := fmt.Sprintf("---\nname: %s\ndescription: %s\nversion: %s\n", name, description, "0.1.0")
	if category != "" {
		skillMd += "category: " + category + "\n"
	}
	skillMd += "---\n\n" + body + "\n"

	skillPath := filepath.Join(toolDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillMd), 0o644); err != nil {
		return "", fmt.Errorf("write SKILL.md: %w", err)
	}

	// Record provenance
	t.provenance.Append(ProvenanceEntry{
		Name:        name,
		Action:      "forge",
		Version:     "0.1.0",
		Description: description,
		Category:    category,
		AuditResult: "PASSED",
	})

	return fmt.Sprintf("Forged tool '%s' v0.1.0. Available to all agents.", name), nil
}

func (t *ForgeTool) handlePatch(name, oldString, newString, versionBump string) (string, error) {
	if name == "" || oldString == "" || newString == "" {
		return "", fmt.Errorf("name, old_string, and new_string are required")
	}

	skillPath := filepath.Join(t.forgeDir, name, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return "", fmt.Errorf("tool '%s' not found", name)
	}

	content := string(data)
	if !strings.Contains(content, oldString) {
		return "", fmt.Errorf("old_string not found in SKILL.md")
	}

	// Get current version
	currentVersion := extractVersion(content)
	newVersion := bumpVersion(currentVersion, versionBump)

	// Apply patch
	updated := strings.Replace(content, oldString, newString, 1)
	updated = strings.Replace(updated, "version: "+currentVersion, "version: "+newVersion, 1)

	if err := os.WriteFile(skillPath, []byte(updated), 0o644); err != nil {
		return "", fmt.Errorf("write SKILL.md: %w", err)
	}

	t.provenance.Append(ProvenanceEntry{
		Name:        name,
		Action:      "patch",
		Version:     newVersion,
		FromVersion: currentVersion,
		ToVersion:   newVersion,
	})

	return fmt.Sprintf("Patched '%s' %s -> %s", name, currentVersion, newVersion), nil
}

func (t *ForgeTool) handleList() (string, error) {
	entries, err := os.ReadDir(t.forgeDir)
	if err != nil {
		return "", err
	}

	var tools []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".archive" {
			continue
		}
		skillPath := filepath.Join(t.forgeDir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}
		desc := extractDescription(string(data))
		ver := extractVersion(string(data))
		stats := t.provenance.ComputeStats(entry.Name())
		tools = append(tools, fmt.Sprintf("  %s (v%s) - %s [↑%d ↓%d %.0f%%]",
			entry.Name(), ver, desc, stats.ThumbsUp, stats.ThumbsDown, stats.SuccessRate*100))
	}

	if len(tools) == 0 {
		return "No forged tools.", nil
	}
	return "Forged tools:\n" + strings.Join(tools, "\n"), nil
}

func (t *ForgeTool) handleView(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	skillPath := filepath.Join(t.forgeDir, name, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return "", fmt.Errorf("tool '%s' not found", name)
	}
	return string(data), nil
}

func (t *ForgeTool) handleStats(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	stats := t.provenance.ComputeStats(name)
	return fmt.Sprintf("Stats for '%s':\n  Uses: %d\n  Agents: %d\n  👍 %d  👎 %d\n  Success: %.0f%%",
		name, stats.UseCount, stats.UniqueAgents, stats.ThumbsUp, stats.ThumbsDown, stats.SuccessRate*100), nil
}

func (t *ForgeTool) handleArchive(name, reason, supersededBy string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	toolDir := filepath.Join(t.forgeDir, name)
	if _, err := os.Stat(toolDir); os.IsNotExist(err) {
		return "", fmt.Errorf("tool '%s' not found", name)
	}

	archiveDir := filepath.Join(t.forgeDir, ".archive")
	os.MkdirAll(archiveDir, 0o755)
	dest := filepath.Join(archiveDir, name)
	if err := os.Rename(toolDir, dest); err != nil {
		return "", fmt.Errorf("archive: %w", err)
	}

	t.provenance.Append(ProvenanceEntry{
		Name:         name,
		Action:       "archive",
		Reason:       reason,
		SupersededBy: supersededBy,
	})

	return fmt.Sprintf("Archived tool '%s'", name), nil
}

func (t *ForgeTool) handlePin(name string, pinned *bool) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	isPinned := true
	if pinned != nil {
		isPinned = *pinned
	}

	t.provenance.Append(ProvenanceEntry{
		Name:   name,
		Action: "pin",
		Pinned: &isPinned,
	})

	if isPinned {
		return fmt.Sprintf("Pinned tool '%s' (exempt from auto-archive)", name), nil
	}
	return fmt.Sprintf("Unpinned tool '%s'", name), nil
}

func (t *ForgeTool) handleRollback(name, targetVersion string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	// Find version directory
	versionsDir := filepath.Join(t.forgeDir, name, "versions")
	if targetVersion == "" {
		// Find previous version
		entries := t.provenance.Replay()
		var versions []string
		for _, e := range entries {
			if e.Name == name && (e.Action == "forge" || e.Action == "patch") && e.Version != "" {
				versions = append(versions, e.Version)
			}
		}
		if len(versions) < 2 {
			return "", fmt.Errorf("no previous version to roll back to")
		}
		targetVersion = versions[len(versions)-2]
	}

	// Find the version file
	versionFile := filepath.Join(versionsDir, targetVersion+".md")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("version %s not found for tool '%s'", targetVersion, name)
	}

	// Restore
	skillPath := filepath.Join(t.forgeDir, name, "SKILL.md")
	if err := os.WriteFile(skillPath, data, 0o644); err != nil {
		return "", fmt.Errorf("restore: %w", err)
	}

	t.provenance.Append(ProvenanceEntry{
		Name:        name,
		Action:      "rollback",
		ToVersion:   targetVersion,
	})

	return fmt.Sprintf("Rolled back '%s' to v%s", name, targetVersion), nil
}

func (t *ForgeTool) handleRate(name, rating, comment string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if rating != "thumbs_up" && rating != "thumbs_down" {
		return "", fmt.Errorf("rating must be 'thumbs_up' or 'thumbs_down'")
	}

	t.provenance.Append(ProvenanceEntry{
		Name:    name,
		Action:  "rate",
		Rating:  rating,
		Comment: comment,
	})

	return fmt.Sprintf("Rated '%s': %s", name, rating), nil
}

// Helpers

func (t *ForgeTool) existingNames() map[string]bool {
	names := make(map[string]bool)
	entries, _ := os.ReadDir(t.forgeDir)
	for _, e := range entries {
		if e.IsDir() && e.Name() != ".archive" {
			names[strings.ToLower(e.Name())] = true
		}
	}
	return names
}

func extractVersion(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		}
	}
	return "0.1.0"
}

func extractDescription(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}

func bumpVersion(current, bump string) string {
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "0.1.1"
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	switch bump {
	case "major":
		return fmt.Sprintf("%d.0.0", major+1)
	case "minor":
		return fmt.Sprintf("%d.%d.0", major, minor+1)
	default:
		return fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
	}
}
