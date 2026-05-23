---
name: savant-hooks
description: Guide for writing Savant CLI pre-tool-use hooks that can allow, deny, or modify tool calls.
license: MIT
compatibility: savant-cli>=0.1.0
---

# Savant CLI Hooks Guide

## Overview

Hooks are user-defined shell commands that fire before tool execution. They can:
- **Allow** a tool call (auto-approve)
- **Deny** a tool call (block it)
- **Halt** the entire turn
- **Inject context** into the tool response
- **Rewrite tool input** before execution

## Configuration

Hooks are configured in `~/.savant/config.json`:

```json
{
  "hooks": [
    {
      "name": "block-rm-rf",
      "matcher": "bash",
      "command": "cat - | jq -e '.tool_input.command' | grep -q 'rm -rf' && exit 2 || exit 0",
      "timeout": 5,
      "event": "PreToolUse"
    }
  ]
}
```

## Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success - parse stdout JSON for decision |
| 2 | Deny - block this tool call (stderr = reason) |
| 49 | Halt - halt the entire turn (stderr = reason) |
| Other | Non-blocking error, tool proceeds normally |

## Stdin Payload

Hooks receive JSON on stdin:

```json
{
  "event": "PreToolUse",
  "session_id": "abc123",
  "cwd": "/path/to/project",
  "tool_name": "bash",
  "tool_input": {"command": "ls -la"}
}
```

## Environment Variables

- `SAVANT_EVENT` - Event name (e.g., "PreToolUse")
- `SAVANT_TOOL_NAME` - Name of the tool being called
- `SAVANT_SESSION_ID` - Current session ID
- `SAVANT_CWD` - Current working directory
- `SAVANT_PROJECT_DIR` - Project root directory

## Stdout Format

Exit code 0: stdout is parsed as JSON:

```json
{
  "decision": "allow",
  "reason": "Auto-approved read-only operation",
  "context": ["Additional context for the agent"],
  "updated_input": "{\"command\": \"ls -la --color=never\"}"
}
```

Or plain text (treated as context injection).

## Examples

### Auto-approve read-only tools
```json
{
  "name": "auto-approve-reads",
  "matcher": "^(read|glob|grep)$",
  "command": "echo '{\"decision\": \"allow\"}'",
  "event": "PreToolUse"
}
```

### Block destructive commands
```json
{
  "name": "block-destructive",
  "matcher": "bash",
  "command": "INPUT=$(cat); echo \"$INPUT\" | jq -e '.tool_input.command' | grep -qE '(rm -rf|git push --force|DROP TABLE)' && echo 'Destructive command blocked' >&2 && exit 2 || exit 0",
  "event": "PreToolUse"
}
```

### Inject context
```json
{
  "name": "add-git-context",
  "matcher": "bash",
  "command": "echo '{\"context\": [\"Current branch: '$(git branch --show-current)'\"]}'",
  "event": "PreToolUse"
}
```

## Matcher

The `matcher` field is a regex matched against the tool name. Empty string or omitted matches all tools.

Examples:
- `"bash"` - matches only the bash tool
- `"^(edit|write)$"` - matches edit and write tools
- `"^(read|glob|grep)$"` - matches read-only tools
- `".*"` or `""` - matches all tools
