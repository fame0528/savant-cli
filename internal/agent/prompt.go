package agent

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/*.md
var templateFS embed.FS

// PromptData holds all data injected into the system prompt template.
type PromptData struct {
	WorkingDir    string
	Platform      string
	Date          string
	IsGitRepo     bool
	GitBranch     string
	GitStatus     string
	GitLog        string
	ContextFiles  []ContextFile
	KnowledgeFiles []ContextFile
}

// ContextFile represents a loaded project instruction file.
type ContextFile struct {
	Path    string
	Content string
}

// BuildSystemPrompt renders the system prompt template with runtime data.
func BuildSystemPrompt() (string, error) {
	data := gatherPromptData()

	tmpl, err := template.ParseFS(templateFS, "templates/system.md")
	if err != nil {
		return "", fmt.Errorf("parse system template: %w", err)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("execute system template: %w", err)
	}

	return sb.String(), nil
}

// BuildInstructionsPrompt returns the per-message instructions reminder.
func BuildInstructionsPrompt() (string, error) {
	data, err := templateFS.ReadFile("templates/instructions.md")
	if err != nil {
		return "", fmt.Errorf("read instructions template: %w", err)
	}
	return string(data), nil
}

// BuildStepPrompt returns the per-step reminder.
func BuildStepPrompt() (string, error) {
	data, err := templateFS.ReadFile("templates/step.md")
	if err != nil {
		return "", fmt.Errorf("read step template: %w", err)
	}
	return string(data), nil
}

// BuildSummaryPrompt returns the context compaction summary prompt.
func BuildSummaryPrompt() (string, error) {
	data, err := templateFS.ReadFile("templates/summary.md")
	if err != nil {
		return "", fmt.Errorf("read summary template: %w", err)
	}
	return string(data), nil
}

// gatherPromptData collects all runtime data for the system prompt.
func gatherPromptData() PromptData {
	cwd, _ := os.Getwd()
	data := PromptData{
		WorkingDir: cwd,
		Platform:   platformName(),
		Date:       time.Now().Format("Monday, January 2, 2006 3:04 PM"),
	}

	// Git info
	if isGitRepo(cwd) {
		data.IsGitRepo = true
		data.GitBranch = gitBranch()
		data.GitStatus = gitStatus()
		data.GitLog = gitLog()
	}

	// Context files (4-tier hierarchy)
	data.ContextFiles = loadContextFiles(cwd)

	// Knowledge files (auto-discovered)
	data.KnowledgeFiles = loadKnowledgeFiles(cwd)

	return data
}

func platformName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func gitBranch() string {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitStatus() string {
	out, err := exec.Command("git", "status", "--short").Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	return strings.Join(lines, "\n")
}

func gitLog() string {
	out, err := exec.Command("git", "log", "--oneline", "-5").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// loadContextFiles loads project instruction files using the 4-tier hierarchy.
func loadContextFiles(cwd string) []ContextFile {
	var files []ContextFile

	// Tier 1: Global user preferences
	home, _ := os.UserHomeDir()
	if home != "" {
		files = append(files, loadFileIfExists(filepath.Join(home, ".savant", "SAVANT.md"), "Global")...)
		files = append(files, loadFileIfExists(filepath.Join(home, ".savant", "instructions.md"), "Global")...)
	}

	// Tier 2: Project root
	files = append(files, loadFileIfExists(filepath.Join(cwd, "AGENTS.md"), "Project")...)
	files = append(files, loadFileIfExists(filepath.Join(cwd, "SAVANT.md"), "Project")...)
	files = append(files, loadFileIfExists(filepath.Join(cwd, "CLAUDE.md"), "Project")...)
	files = append(files, loadFileIfExists(filepath.Join(cwd, "GEMINI.md"), "Project")...)
	files = append(files, loadFileIfExists(filepath.Join(cwd, "CRUSH.md"), "Project")...)
	files = append(files, loadFileIfExists(filepath.Join(cwd, "OpenCode.md"), "Project")...)

	// Tier 3: Subdirectory context (walk up to git root)
	gitRoot := findGitRoot(cwd)
	if gitRoot != "" && gitRoot != cwd {
		files = append(files, loadFileIfExists(filepath.Join(gitRoot, "AGENTS.md"), "Workspace")...)
	}

	return files
}

// loadKnowledgeFiles auto-discovers knowledge files in the project.
func loadKnowledgeFiles(cwd string) []ContextFile {
	var files []ContextFile
	seen := make(map[string]bool)

	gitRoot := findGitRoot(cwd)
	if gitRoot == "" {
		gitRoot = cwd
	}

	// Only walk from CWD to git root (not entire project tree)
	filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Don't walk above git root
		if gitRoot != "" && !strings.HasPrefix(path, gitRoot) {
			return filepath.SkipDir
		}
		// Skip hidden dirs and common large dirs
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		base := strings.ToLower(info.Name())
		if base == "knowledge.md" || base == "agents.md" || base == "claude.md" || base == "savant.md" {
			if !seen[path] {
				seen[path] = true
				data, err := os.ReadFile(path)
				if err == nil {
					rel, _ := filepath.Rel(cwd, path)
					files = append(files, ContextFile{
						Path:    rel,
						Content: string(data),
					})
				}
			}
		}
		return nil
	})

	return files
}

func loadFileIfExists(path string, tier string) []ContextFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return []ContextFile{{
		Path:    path,
		Content: string(data),
	}}
}

func findGitRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
