package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GrepTool struct{}

func NewGrepTool() *GrepTool { return &GrepTool{} }

func (t *GrepTool) Name() string        { return "grep" }
func (t *GrepTool) Description() string { return "Search file contents by regex or literal text." }
func (t *GrepTool) Kind() ToolKind      { return KindSearch }

func (t *GrepTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Search pattern (regex supported)"
			},
			"path": {
				"type": "string",
				"description": "Directory or file to search in (default: current directory)"
			},
			"glob": {
				"type": "string",
				"description": "File glob filter (e.g., '*.go')"
			},
			"case_insensitive": {
				"type": "boolean",
				"description": "Case insensitive search"
			}
		},
		"required": ["pattern"]
	}`)
}

func (t *GrepTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Pattern         string `json:"pattern"`
		Path            string `json:"path"`
		Glob            string `json:"glob"`
		CaseInsensitive bool   `json:"case_insensitive"`
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

	flags := ""
	if params.CaseInsensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + params.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	var results []string
	count := 0
	maxResults := 250

	err = filepath.Walk(params.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Directory filtering: skip hidden dirs and common large dirs
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// File filtering
		name := info.Name()

		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			return nil
		}

		// Glob filter
		if params.Glob != "" {
			matched, _ := filepath.Match(params.Glob, name)
			if !matched {
				return nil
			}
		}

		// Skip files > 10MB
		if info.Size() > 10*1024*1024 {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if re.MatchString(scanner.Text()) {
				if count >= maxResults {
					results = append(results, "... (truncated)")
					return filepath.SkipAll
				}
				rel, _ := filepath.Rel(params.Path, path)
				results = append(results, fmt.Sprintf("%s:%d: %s", rel, lineNum, strings.TrimSpace(scanner.Text())))
				count++
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}
	return strings.Join(results, "\n"), nil
}
