package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type EditTool struct{}

func NewEditTool() *EditTool { return &EditTool{} }

func (t *EditTool) Name() string        { return "edit" }
func (t *EditTool) Description() string { return "Replace a string in a file. The old_string must be unique in the file." }

func (t *EditTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file to edit"
			},
			"old_string": {
				"type": "string",
				"description": "The exact string to find and replace (must not be empty)"
			},
			"new_string": {
				"type": "string",
				"description": "The replacement string"
			},
			"replace_all": {
				"type": "boolean",
				"description": "Replace all occurrences (default false)"
			}
		},
		"required": ["path", "old_string", "new_string"]
	}`)
}

func (t *EditTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path       string `json:"path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	if params.OldString == "" {
		return "", fmt.Errorf("old_string must not be empty")
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	// Preserve original permissions
	info, err := os.Stat(params.Path)
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}
	perm := info.Mode().Perm()

	content := string(data)
	count := strings.Count(content, params.OldString)

	if count == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}

	if params.ReplaceAll {
		content = strings.ReplaceAll(content, params.OldString, params.NewString)
		if err := os.WriteFile(params.Path, []byte(content), perm); err != nil {
			return "", fmt.Errorf("write file: %w", err)
		}
		return fmt.Sprintf("Replaced %d occurrences in %s", count, params.Path), nil
	}

	if count > 1 {
		return "", fmt.Errorf("old_string is not unique in file (%d matches). Use a larger context or set replace_all", count)
	}

	content = strings.Replace(content, params.OldString, params.NewString, 1)
	if err := os.WriteFile(params.Path, []byte(content), perm); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return fmt.Sprintf("Edited %s", params.Path), nil
}
