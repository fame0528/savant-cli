package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var skillNameRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const (
	maxSkillNameLen  = 64
	maxSkillDescLen  = 60
	maxSkillContLen  = 100_000
	maxSkillFileSize = 1_048_576 // 1 MiB
)

var allowedSubdirs = map[string]bool{
	"references": true,
	"templates":  true,
	"scripts":    true,
	"assets":     true,
}

type SkillManageTool struct {
	skillsDir string
}

func NewSkillManageTool(skillsDir string) *SkillManageTool {
	return &SkillManageTool{skillsDir: skillsDir}
}

func (t *SkillManageTool) Name() string        { return "skill_manage" }
func (t *SkillManageTool) Description() string { return "Create, edit, patch, or delete skills. Skills are reusable procedural knowledge." }
func (t *SkillManageTool) Kind() ToolKind      { return KindWrite }

func (t *SkillManageTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "edit", "patch", "delete", "write_file", "remove_file"],
				"description": "The action to perform"
			},
			"name": {
				"type": "string",
				"description": "Skill name (lowercase alphanumeric with hyphens)"
			},
			"description": {
				"type": "string",
				"description": "Skill description (max 60 chars, one sentence, ends with period)"
			},
			"content": {
				"type": "string",
				"description": "Full SKILL.md content (for create/edit) or new content (for write_file)"
			},
			"path": {
				"type": "string",
				"description": "Relative path within the skill directory (for write_file/remove_file)"
			},
			"old_string": {
				"type": "string",
				"description": "Text to find (for patch action)"
			},
			"new_string": {
				"type": "string",
				"description": "Replacement text (for patch action)"
			}
		},
		"required": ["action", "name"]
	}`)
}

func (t *SkillManageTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action      string `json:"action"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
		Path        string `json:"path"`
		OldString   string `json:"old_string"`
		NewString   string `json:"new_string"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	// Validate name
	if err := validateSkillName(params.Name); err != nil {
		return "", err
	}

	skillDir := filepath.Join(t.skillsDir, params.Name)

	switch params.Action {
	case "create":
		return t.create(skillDir, params.Name, params.Description, params.Content)
	case "edit":
		return t.edit(skillDir, params.Name, params.Content)
	case "patch":
		return t.patch(skillDir, params.Name, params.Path, params.OldString, params.NewString)
	case "delete":
		return t.delete(skillDir, params.Name)
	case "write_file":
		return t.writeFile(skillDir, params.Path, params.Content)
	case "remove_file":
		return t.removeFile(skillDir, params.Path)
	default:
		return "", fmt.Errorf("unknown action: %s", params.Action)
	}
}

func (t *SkillManageTool) create(skillDir, name, description, content string) (string, error) {
	if description == "" {
		return "", fmt.Errorf("description is required")
	}
	if len(description) > maxSkillDescLen {
		return "", fmt.Errorf("description too long: %d > %d chars", len(description), maxSkillDescLen)
	}
	if !strings.HasSuffix(description, ".") {
		return "", fmt.Errorf("description must end with a period")
	}
	if len(content) > maxSkillContLen {
		return "", fmt.Errorf("content too long: %d > %d chars", len(content), maxSkillContLen)
	}

	// Check if already exists
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err == nil {
		return "", fmt.Errorf("skill %q already exists", name)
	}

	// Build SKILL.md content
	skillMd := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, content)

	// Create directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return "", fmt.Errorf("create skill directory: %w", err)
	}

	// Write SKILL.md atomically
	if err := atomicWrite(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd)); err != nil {
		return "", fmt.Errorf("write SKILL.md: %w", err)
	}

	return fmt.Sprintf("Created skill %q at %s", name, skillDir), nil
}

func (t *SkillManageTool) edit(skillDir, name, content string) (string, error) {
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return "", fmt.Errorf("skill %q not found", name)
	}
	if len(content) > maxSkillContLen {
		return "", fmt.Errorf("content too long: %d > %d chars", len(content), maxSkillContLen)
	}

	if err := atomicWrite(skillPath, []byte(content)); err != nil {
		return "", fmt.Errorf("write SKILL.md: %w", err)
	}

	return fmt.Sprintf("Updated skill %q", name), nil
}

func (t *SkillManageTool) patch(skillDir, name, relPath, oldString, newString string) (string, error) {
	if oldString == "" {
		return "", fmt.Errorf("old_string must not be empty")
	}

	// Determine target file
	targetPath := filepath.Join(skillDir, "SKILL.md")
	if relPath != "" {
		if err := validateSubdirPath(relPath); err != nil {
			return "", err
		}
		targetPath = filepath.Join(skillDir, relPath)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, oldString) {
		return "", fmt.Errorf("old_string not found in %s", filepath.Base(targetPath))
	}

	newContent := strings.Replace(content, oldString, newString, 1)

	info, err := os.Stat(targetPath)
	if err != nil {
		return "", err
	}

	if err := atomicWrite(targetPath, []byte(newContent)); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	_ = info // preserve permissions in future

	return fmt.Sprintf("Patched %s in skill %q", filepath.Base(targetPath), name), nil
}

func (t *SkillManageTool) delete(skillDir, name string) (string, error) {
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return "", fmt.Errorf("skill %q not found", name)
	}

	// Archive instead of delete
	archiveDir := filepath.Join(t.skillsDir, ".archive")
	os.MkdirAll(archiveDir, 0o755)
	destDir := filepath.Join(archiveDir, name)

	if err := os.Rename(skillDir, destDir); err != nil {
		return "", fmt.Errorf("archive skill: %w", err)
	}

	return fmt.Sprintf("Archived skill %q to %s", name, destDir), nil
}

func (t *SkillManageTool) writeFile(skillDir, relPath, content string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("path is required")
	}
	if err := validateSubdirPath(relPath); err != nil {
		return "", err
	}
	if len(content) > maxSkillFileSize {
		return "", fmt.Errorf("file too large: %d > %d bytes", len(content), maxSkillFileSize)
	}

	// Ensure skill directory exists
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); os.IsNotExist(err) {
		return "", fmt.Errorf("skill directory %q does not exist", filepath.Base(skillDir))
	}

	fullPath := filepath.Join(skillDir, relPath)
	os.MkdirAll(filepath.Dir(fullPath), 0o755)

	if err := atomicWrite(fullPath, []byte(content)); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("Wrote %s to skill %q", relPath, filepath.Base(skillDir)), nil
}

func (t *SkillManageTool) removeFile(skillDir, relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("path is required")
	}
	if err := validateSubdirPath(relPath); err != nil {
		return "", err
	}

	fullPath := filepath.Join(skillDir, relPath)
	if err := os.Remove(fullPath); err != nil {
		return "", fmt.Errorf("remove file: %w", err)
	}

	return fmt.Sprintf("Removed %s from skill %q", relPath, filepath.Base(skillDir)), nil
}

func validateSkillName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name is required")
	}
	if len(name) > maxSkillNameLen {
		return fmt.Errorf("skill name too long: %d > %d", len(name), maxSkillNameLen)
	}
	if !skillNameRegex.MatchString(name) {
		return fmt.Errorf("invalid skill name %q: must be lowercase alphanumeric with hyphens", name)
	}
	return nil
}

func validateSubdirPath(relPath string) error {
	parts := strings.SplitN(relPath, "/", 2)
	if len(parts) < 1 {
		return fmt.Errorf("invalid path: %s", relPath)
	}
	if !allowedSubdirs[parts[0]] {
		return fmt.Errorf("path must be under references/, templates/, scripts/, or assets/: got %s", parts[0])
	}
	return nil
}

func atomicWrite(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
