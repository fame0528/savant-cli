package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ReadTool struct{}

func NewReadTool() *ReadTool { return &ReadTool{} }

func (t *ReadTool) Name() string        { return "read" }
func (t *ReadTool) Description() string { return "Read a file from disk. Returns line-numbered content." }

func (t *ReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Absolute path to the file"
			},
			"offset": {
				"type": "integer",
				"description": "Line number to start reading from (1-indexed)"
			},
			"limit": {
				"type": "integer",
				"description": "Maximum number of lines to read (default 2000)"
			}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if params.Limit == 0 {
		params.Limit = 2000
	}
	if params.Offset > 0 {
		params.Offset-- // Convert to 0-indexed
	}

	end := params.Offset + params.Limit
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := params.Offset; i < end; i++ {
		sb.WriteString(fmt.Sprintf("%d\t%s\n", i+1, lines[i]))
	}
	return sb.String(), nil
}
