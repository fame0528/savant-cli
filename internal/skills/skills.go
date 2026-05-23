// Package skills implements the Agent Skills open standard.
// See https://agentskills.io for the specification.
// Spec: https://agentskills.io/specification
package skills

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	SkillFileName        = "SKILL.md"
	MaxNameLength        = 64
	MaxDescriptionLength = 1024
	MaxCompatLength      = 500
	MaxInstructionsLines = 500
)

// namePattern matches valid skill names: lowercase alphanumeric with hyphens,
// no leading/trailing/consecutive hyphens. Must match parent directory name.
var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Skill represents a parsed SKILL.md file per the Agent Skills specification.
type Skill struct {
	Name          string            `yaml:"name" json:"name"`
	Description   string            `yaml:"description" json:"description"`
	License       string            `yaml:"license,omitempty" json:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`
	Instructions  string            `yaml:"-" json:"instructions"`
	Path          string            `yaml:"-" json:"path"`
	Builtin       bool              `yaml:"-" json:"builtin"`
}

//go:embed builtin/*
var builtinFS embed.FS

// Discover finds all skills from builtin, global, and project paths.
func Discover(projectDir string) []Skill {
	var skills []Skill

	// Builtin skills
	skills = append(skills, DiscoverBuiltin()...)

	// Global user skills
	home, _ := os.UserHomeDir()
	if home != "" {
		skills = append(skills, DiscoverFromPath(filepath.Join(home, ".savant", "skills"))...)
		skills = append(skills, DiscoverFromPath(filepath.Join(home, ".agents", "skills"))...)
		skills = append(skills, DiscoverFromPath(filepath.Join(home, ".claude", "skills"))...)
	}

	// Project skills
	if projectDir != "" {
		skills = append(skills, DiscoverFromPath(filepath.Join(projectDir, ".savant", "skills"))...)
		skills = append(skills, DiscoverFromPath(filepath.Join(projectDir, ".agents", "skills"))...)
		skills = append(skills, DiscoverFromPath(filepath.Join(projectDir, ".crush", "skills"))...)
		skills = append(skills, DiscoverFromPath(filepath.Join(projectDir, ".claude", "skills"))...)
		skills = append(skills, DiscoverFromPath(filepath.Join(projectDir, ".kilo", "skills"))...)
	}

	// Deduplicate (last occurrence wins per spec)
	return Deduplicate(skills)
}

// DiscoverBuiltin loads skills embedded in the binary.
func DiscoverBuiltin() []Skill {
	var skills []Skill

	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := "builtin/" + entry.Name() + "/" + SkillFileName // embed.FS requires forward slashes
		data, err := builtinFS.ReadFile(skillPath)
		if err != nil {
			continue
		}
		skill, err := Parse(entry.Name(), string(data))
		if err != nil {
			continue
		}
		skill.Path = "builtin://" + entry.Name()
		skill.Builtin = true
		skills = append(skills, skill)
	}

	return skills
}

// DiscoverFromPath walks a directory looking for SKILL.md files.
func DiscoverFromPath(dir string) []Skill {
	var skills []Skill

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() != SkillFileName {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		dirName := filepath.Base(filepath.Dir(path))
		skill, err := Parse(dirName, string(data))
		if err != nil {
			return nil
		}
		skill.Path = path
		skills = append(skills, skill)

		return nil
	})

	return skills
}

// Parse parses a SKILL.md file into a Skill struct per the Agent Skills spec.
// The format is YAML frontmatter delimited by --- followed by markdown instructions.
func Parse(dirName, content string) (Skill, error) {
	skill := Skill{}

	// Split frontmatter
	parts := splitFrontmatter(content)
	if len(parts) >= 2 {
		// Parse YAML frontmatter line by line
		frontmatter := parts[0]
		var inMetadata bool
		for _, line := range strings.Split(frontmatter, "\n") {
			trimmed := strings.TrimSpace(line)

			// Handle nested metadata keys
			if inMetadata && strings.HasPrefix(line, "  ") {
				kv := strings.SplitN(strings.TrimSpace(line), ":", 2)
				if len(kv) == 2 && skill.Metadata != nil {
					skill.Metadata[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
				continue
			}
			inMetadata = false

			if strings.HasPrefix(trimmed, "name:") {
				skill.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			} else if strings.HasPrefix(trimmed, "description:") {
				skill.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			} else if strings.HasPrefix(trimmed, "license:") {
				skill.License = strings.TrimSpace(strings.TrimPrefix(trimmed, "license:"))
			} else if strings.HasPrefix(trimmed, "compatibility:") {
				skill.Compatibility = strings.TrimSpace(strings.TrimPrefix(trimmed, "compatibility:"))
			} else if strings.HasPrefix(trimmed, "allowed-tools:") {
				skill.AllowedTools = strings.TrimSpace(strings.TrimPrefix(trimmed, "allowed-tools:"))
			} else if strings.HasPrefix(trimmed, "metadata:") {
				inMetadata = true
				skill.Metadata = make(map[string]string)
			}
		}
		skill.Instructions = strings.TrimSpace(parts[1])
	} else {
		skill.Instructions = strings.TrimSpace(content)
	}

	// Validate per spec
	if skill.Name == "" {
		skill.Name = dirName
	}
	if len(skill.Name) > MaxNameLength {
		return skill, fmt.Errorf("skill name too long: %d > %d", len(skill.Name), MaxNameLength)
	}
	if !namePattern.MatchString(skill.Name) {
		return skill, fmt.Errorf("invalid skill name %q: must be lowercase alphanumeric with hyphens, no leading/trailing/consecutive hyphens", skill.Name)
	}
	// Spec: name must match parent directory name
	if skill.Name != dirName {
		return skill, fmt.Errorf("skill name %q does not match directory name %q", skill.Name, dirName)
	}
	// Spec: description is required
	if skill.Description == "" {
		return skill, fmt.Errorf("skill %q is missing required 'description' field", skill.Name)
	}
	if len(skill.Description) > MaxDescriptionLength {
		skill.Description = skill.Description[:MaxDescriptionLength]
	}
	if len(skill.Compatibility) > MaxCompatLength {
		skill.Compatibility = skill.Compatibility[:MaxCompatLength]
	}

	return skill, nil
}

// splitFrontmatter splits YAML frontmatter from markdown body.
func splitFrontmatter(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return []string{content}
	}

	// Find second ---
	rest := strings.TrimSpace(content)
	rest = strings.TrimPrefix(rest, "---")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return []string{content}
	}

	frontmatter := strings.TrimSpace(rest[:idx])
	body := strings.TrimSpace(rest[idx+4:])

	return []string{frontmatter, body}
}

// Deduplicate removes duplicate skills by name, keeping the last occurrence.
func Deduplicate(skills []Skill) []Skill {
	seen := make(map[string]int)
	for i, s := range skills {
		seen[s.Name] = i
	}

	result := make([]Skill, 0, len(seen))
	for _, idx := range seen {
		result = append(result, skills[idx])
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// ToPromptXML converts skills to XML for system prompt injection.
// Per spec: only name and description are included (progressive disclosure).
func ToPromptXML(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("  <skill name=%q>\n", s.Name))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", s.Description))
		}
		sb.WriteString(fmt.Sprintf("    <location>%s</location>\n", s.Path))
		sb.WriteString("  </skill>\n")
	}
	sb.WriteString("</available_skills>\n")
	return sb.String()
}

// Filter removes disabled skills.
func Filter(skills []Skill, disabled []string) []Skill {
	if len(disabled) == 0 {
		return skills
	}

	disabledSet := make(map[string]bool)
	for _, d := range disabled {
		disabledSet[d] = true
	}

	var result []Skill
	for _, s := range skills {
		if !disabledSet[s.Name] {
			result = append(result, s)
		}
	}
	return result
}
