package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GlobTool struct{}

func NewGlobTool() *GlobTool { return &GlobTool{} }

func (t *GlobTool) Name() string        { return "glob" }
func (t *GlobTool) Description() string { return "Find files matching a glob pattern. Supports ** for recursive matching." }
func (t *GlobTool) Kind() ToolKind      { return KindSearch }

func (t *GlobTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Glob pattern (e.g., '**/*.go', 'src/**/*.ts', '*.md')"
			},
			"path": {
				"type": "string",
				"description": "Directory to search in (default: current directory)"
			}
		},
		"required": ["pattern"]
	}`)
}

func (t *GlobTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if params.Path == "" {
		var err error
		params.Path, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	// Split pattern into base (non-recursive prefix) and the glob part
	// e.g., "src/**/*.go" -> base="src", glob="**/*.go"
	pattern := params.Pattern
	hasDoubleStar := strings.Contains(pattern, "**")

	var matches []string
	err := filepath.Walk(params.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip hidden directories and common large dirs
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(params.Path, path)
		if err != nil {
			return nil
		}
		// Normalize to forward slashes for cross-platform matching
		rel = filepath.ToSlash(rel)

		if hasDoubleStar {
			// For ** patterns, use custom matching
			if matchDoubleStar(pattern, rel) {
				matches = append(matches, rel)
			}
		} else {
			// For simple patterns, use filepath.Match on each path component
			matched, _ := filepath.Match(pattern, rel)
			if !matched {
				// Also try matching just the filename
				matched, _ = filepath.Match(pattern, info.Name())
			}
			if matched {
				matches = append(matches, rel)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(matches)

	if len(matches) == 0 {
		return "No files matched.", nil
	}

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(m)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// matchDoubleStar handles ** glob patterns.
// "**/*.go" matches any .go file at any depth.
// "src/**/test.go" matches test.go anywhere under src/.
func matchDoubleStar(pattern, path string) bool {
	// Split pattern on **
	parts := strings.Split(pattern, "**")
	if len(parts) == 1 {
		// No **, just do normal match
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// For "**/*.go": match any suffix that matches "*.go"
	if len(parts) == 2 && parts[0] == "" {
		suffix := strings.TrimPrefix(parts[1], "/")
		// Match against each path component
		pathParts := strings.Split(path, "/")
		for _, part := range pathParts {
			matched, _ := filepath.Match(suffix, part)
			if matched {
				return true
			}
		}
		// Also try matching the full suffix against the full path
		matched, _ := filepath.Match(suffix, filepath.Base(path))
		return matched
	}

	// For "src/**/test.go": prefix must match, suffix must match
	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// Check if path starts with prefix
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(path, prefix)
	remainder = strings.TrimPrefix(remainder, "/")

	// Check if the remainder matches the suffix
	if suffix == "" {
		return true
	}
	matched, _ := filepath.Match(suffix, filepath.Base(remainder))
	return matched
}
