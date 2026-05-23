# FID-20260524-TOOL-FORGE

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260524-TOOL-FORGE                         |
| **Date Created** | 2026-05-24                                      |
| **Status**       | OPEN                                            |
| **Priority**     | HIGH                                            |

## Context

savant-cli currently has a fixed set of tools (bash, read, edit, write, glob, grep, skill_manage, spawn_agent). The agent cannot create new tools at runtime. Two reference systems inform the design:

**Savant Rust ToolForge** (`C:\Users\spenc\dev\Savant\crates\toolforge\`):
- Creates actual callable tools that are immediately available to all agents
- 10 actions: forge, patch, list, view, stats, archive, pin, rollback, rate, share
- Quality Gate validates tools before registration (name, description, stubs, duplicates, actionable content)
- Provenance Tracker logs all lifecycle events to JSONL
- Shared Registry with epoch-based concurrency and broadcast events
- Collective Curator auto-archives stale tools after 30 days
- Tools stored as SKILL.md files in `skills/forge/` directory

**Hermes** (`C:\Users\spenc\dev\savant-cli\resoruces\hermes-agent\tools\`):
- Creates skills (knowledge documents), NOT executable tools
- AST-based auto-discovery of tools at startup
- Tool registry with override protection, TTL-cached availability checks
- Plugin system for registering custom tools at load time
- Key insight: Hermes CANNOT create new executable tools at runtime — it can only create skills

**Decision:** Adopt Savant Rust's ToolForge approach (actual callable tools), not Hermes's approach (knowledge documents). Savant's approach is more powerful — forged tools become immediately callable by all agents.

## Architecture: Tool Forge System

### Component 1: Forge Tool (from Savant Rust `forge_tool.rs`)

An agent-callable tool that creates/manages other tools:

```go
type ForgeTool struct {
    forgeDir  string
    registry  *SharedToolRegistry
    provenance *ProvenanceTracker
}
```

**Actions:**
- `forge` — Create a new tool (SKILL.md + directory structure)
- `patch` — Update an existing tool (find-and-replace with version bump)
- `list` — List all forged tools with stats
- `view` — View a tool's SKILL.md content
- `stats` — View usage stats (use_count, unique_agents, thumbs_up/down, success_rate)
- `archive` — Archive a tool (removes from registry, keeps files)
- `pin` / `unpin` — Pin a tool (exempt from auto-archiving)
- `rollback` — Roll back to a previous version
- `rate` — Rate a tool (thumbs_up/thumbs_down with optional comment)
- `share` — All forge tools are shared across agents by default

### Component 2: Quality Gate (from Savant Rust `quality.rs`)

Validates forged tools before registration:

| Check | Rule | Error Code |
|-------|------|------------|
| Name required | Non-empty | E_NAME_REQUIRED |
| Name format | kebab-case, starts with letter, max 64 chars | E_NAME_FORMAT |
| Duplicate name | No existing tool with same name | E_DUPLICATE_NAME |
| Duplicate similar | Keyword overlap > 80% with existing tool | E_DUPLICATE_SIMILAR |
| Description | Minimum 10 characters | E_DESC_REQUIRED |
| Version | Must match X.Y.Z (semver) | E_VERSION_REQUIRED |
| No stubs | No `TODO`, `FIXME`, `placeholder`, `[STUB]`, `TBD` | E_STUBS_FOUND |
| Body length | Minimum 200 characters (excluding frontmatter) | E_BODY_TOO_SHORT |
| Actionable content | Must contain numbered list, bullet list, or code block | E_NO_ACTIONABLE |

### Component 3: Provenance Tracker (from Savant Rust `provenance.rs`)

JSONL append-only log of all tool lifecycle events:

```go
type ProvenanceEntry struct {
    Name           string   `json:"name"`
    CreatorAgentID string   `json:"creator_agent_id"`
    Action         string   `json:"action"` // forge, patch, archive, pin, rate, rollback
    Version        string   `json:"version,omitempty"`
    Description    string   `json:"description,omitempty"`
    Category       string   `json:"category,omitempty"`
    Rating         string   `json:"rating,omitempty"` // thumbs_up, thumbs_down
    Comment        string   `json:"comment,omitempty"`
    Pinned         *bool    `json:"pinned,omitempty"`
    Reason         string   `json:"reason,omitempty"`
    SupersededBy   string   `json:"superseded_by,omitempty"`
    AuditResult    string   `json:"audit_result,omitempty"`
    FromVersion    string   `json:"from_version,omitempty"`
    ToVersion      string   `json:"to_version,omitempty"`
    Timestamp      string   `json:"timestamp"`
}

type ToolStats struct {
    UseCount     int
    UniqueAgents int
    ThumbsUp     int
    ThumbsDown   int
    LastUsedAt   *time.Time
    SuccessRate  float64
}
```

### Component 4: Shared Tool Registry (from Savant Rust `registry.rs`)

Epoch-based concurrent registry with event broadcasting:

```go
type SharedToolRegistry struct {
    mu        sync.RWMutex
    tools     map[string]Tool
    epochID   uint64
    listeners []chan ToolRegistryEvent
}

type ToolRegistryEvent struct {
    Type ToolRegistryEventType // ToolAdded, ToolRemoved, ToolUpdated
    Name string
}
```

- `Register(name, tool)` — adds tool, bumps epoch, broadcasts ToolAdded
- `Remove(name)` — removes tool, bumps epoch, broadcasts ToolRemoved
- `Get(name)` — thread-safe lookup
- `ListAll()` — returns copy of all tools
- `Subscribe()` — returns channel for registry events
- Forged tools are registered here and immediately available to all agents

### Component 5: Collective Curator (from Savant Rust `curator.rs`)

Auto-archives stale forged tools:

```go
type Curator struct {
    registry       *SharedToolRegistry
    provenance     *ProvenanceTracker
    staleThreshold time.Duration // default: 30 days
}
```

- Runs periodically (configurable interval)
- Checks provenance log for last activity per tool
- Auto-archives tools inactive for `staleThreshold` days
- Pinned tools are exempt
- Never deletes — only archives (files remain on disk)

### Component 6: Tool Templates (Savant Rust pattern)

Forged tools are stored as SKILL.md files with a standard structure:

```
~/.savant/skills/forge/
├── web-scraper/
│   ├── SKILL.md          # Tool definition (YAML frontmatter + markdown)
│   ├── references/       # Supporting documentation
│   ├── templates/        # Reusable templates
│   ├── scripts/          # Helper scripts
│   └── assets/           # Other assets
├── api-tester/
│   ├── SKILL.md
│   └── scripts/
│       └── test_api.sh
└── code-analyzer/
    └── SKILL.md
```

SKILL.md format:
```markdown
---
name: web-scraper
description: Scrapes web pages and extracts structured data.
version: 0.1.0
category: web
---

# Web Scraper

## Usage
Extracts structured data from web pages.

## Procedure
1. Navigate to the target URL using the fetch tool
2. Extract the page HTML
3. Parse the content using the steps below
4. Return the structured data to the user

## Error Handling
- If fetch fails, retry up to 3 times
- If the page is empty, report the error
```

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tools/forge.go` | Forge tool (create/manage forged tools) |
| `internal/tools/quality.go` | Quality gate (validates forged tools) |
| `internal/tools/provenance.go` | Provenance tracker (JSONL lifecycle log) |
| `internal/tools/shared_registry.go` | Shared tool registry (epoch-based, broadcast events) |
| `internal/tools/curator.go` | Collective curator (auto-archive stale tools) |

## Files to Modify

| File | Change |
|------|--------|
| `internal/tools/tools.go` | Update Registry to support dynamic tool addition + event broadcasting |
| `internal/agent/agent.go` | Wire forge tool, handle forged tool calls |
| `main.go` | Create forge directory, wire shared registry and curator |

## Verification Criteria

- [ ] forge action creates a new tool (SKILL.md + directory)
- [ ] Quality gate validates name, description, version, body, stubs, duplicates
- [ ] Forged tool is immediately callable by all agents
- [ ] patch action updates existing tool with version bump
- [ ] list action shows all forged tools with stats
- [ ] view action shows SKILL.md content
- [ ] stats action shows use_count, unique_agents, thumbs_up/down, success_rate
- [ ] archive action removes from registry, keeps files
- [ ] pin/unpin exempt from auto-archiving
- [ ] rollback restores previous version
- [ ] rate action records thumbs_up/down with optional comment
- [ ] Provenance tracker logs all lifecycle events to JSONL
- [ ] Shared registry broadcasts ToolAdded/ToolRemoved events
- [ ] Curator auto-archives stale tools after 30 days
- [ ] Pinned tools exempt from curator
- [ ] No stubs or placeholders
