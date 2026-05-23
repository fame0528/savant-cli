# FID-20260524-SUB-AGENT-SYSTEM

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260524-SUB-AGENT-SYSTEM                   |
| **Date Created** | 2026-05-24                                      |
| **Status**       | OPEN                                            |
| **Priority**     | HIGH                                            |

## Context

savant-cli is a coding CLI tool. It currently has a single-agent loop: user sends a message, agent responds, agent calls tools, repeat. For complex coding tasks, a single agent isn't enough — sometimes you need to explore the codebase AND implement a feature in parallel, or have one agent do the coding while another reviews it.

The user's Rust Savant framework has a shared-state architecture (blackboard) that solves the "island problem" where sub-agents lose parent context. We can apply that pattern here without rebuilding the entire framework.

## Problem: The Island Problem

When a sub-agent is spawned by the parent, it typically gets:
- A task description (string)
- Nothing else

It doesn't get:
- The parent's conversation history
- What files the parent has already read/modified
- What decisions have been made
- What the overall plan is

This causes the sub-agent to waste time re-reading files the parent already read, make decisions that conflict with the parent's approach, or produce results that don't fit the parent's context.

## Solution: Shared Blackboard

Instead of passing raw conversation history (expensive), all agents read/write a shared **blackboard** — a thread-safe key-value store that holds the session's shared state.

### What Goes on the Blackboard

| Key | Value | Who Writes | Who Reads |
|-----|-------|-----------|-----------|
| `plan` | Current approach/strategy | Parent | All |
| `files_modified` | List of files changed this session | All | All |
| `files_read` | List of files read this session | All | All |
| `decisions` | Key decisions made | Parent | All |
| `blockers` | Open issues | All | All |
| `cwd` | Working directory | Parent | All |
| `goal` | Top-level user request | Parent | All |

### How Sub-Agents Get Context

When spawning a sub-agent, build its system prompt from:

```
{agent_definition}  // mode, permissions, tools

## Task
{task_description}

## Session Context (from blackboard)
Goal: {goal}
Plan: {plan}
Files modified: {files_modified}
Files read: {files_read}
Decisions: {decisions}
Blockers: {blockers}

## Project Context
{AGENTS.md / SAVANT.md content if they exist}
```

This gives the sub-agent everything it needs without passing raw conversation history.

### How Results Merge Back

When a sub-agent completes:
1. Its result is injected as a tool result in the parent's conversation
2. Files it modified are added to the blackboard's `files_modified`
3. Decisions it made are added to `decisions`

## Modes System

Different coding tasks need different tool access:

| Mode | Tools | Use Case |
|------|-------|----------|
| `code` | All tools | Primary coding mode |
| `debug` | read, bash (read-only commands), glob, grep | Diagnosing issues |
| `ask` | read, glob, grep | Q&A, no file modifications |
| `review` | read, glob, grep | Code review, read-only |

Switch modes via `/mode code`, `/mode debug`, etc. Or let the agent auto-select based on the task.

## Implementation

### Files to Create

| File | Purpose |
|------|---------|
| `internal/agent/blackboard.go` | Thread-safe shared state (sync.RWMutex) |
| `internal/agent/spawn.go` | spawn_agent tool — creates sub-agent with blackboard context |
| `internal/agent/modes.go` | Mode definitions and per-mode permissions |
| `internal/agent/orchestrator.go` | Coordinates parallel sub-agents (errgroup) |
| `internal/tools/spawn_agent.go` | spawn_agent tool implementation |
| `internal/agent/templates/subagent.md` | Sub-agent system prompt template |

### Files to Modify

| File | Change |
|------|--------|
| `internal/agent/agent.go` | Create blackboard on init, pass to sub-agents |
| `internal/tui/tui.go` | Add /mode command, sub-agent progress display |
| `internal/commands/commands.go` | Add /mode command |
| `main.go` | Wire blackboard creation |

## Verification Criteria

- [ ] Blackboard is thread-safe (sync.RWMutex)
- [ ] Parent writes plan/files/decisions to blackboard before spawning
- [ ] Sub-agent's system prompt includes blackboard context
- [ ] Sub-agent results merge back into parent conversation
- [ ] Sub-agent's modified files appear in blackboard
- [ ] /mode command switches between code/debug/ask/review
- [ ] debug mode blocks write/edit/bash tools
- [ ] ask mode blocks all modification tools
- [ ] Parallel sub-agents work (errgroup)
- [ ] Sub-agent progress shown in TUI
- [ ] No stubs or placeholders
