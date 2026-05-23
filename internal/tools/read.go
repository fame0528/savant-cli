package tools

import (
	"bufio"
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

	file, err := os.Open(params.Path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	defer file.Close()

	if params.Limit <= 0 {
		params.Limit = 2000
	}
	if params.Offset < 1 {
		params.Offset = 1
	}

	startLine := params.Offset // 1-indexed
	endLine := startLine + params.Limit - 1

	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			sb.WriteString(fmt.Sprintf("%d\t%s\n", lineNum, scanner.Text()))
		}
		if lineNum > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan file: %w", err)
	}

	if sb.Len() == 0 {
		return fmt.Sprintf("File has fewer than %d lines.", startLine), nil
	}

	return sb.String(), nil
}
