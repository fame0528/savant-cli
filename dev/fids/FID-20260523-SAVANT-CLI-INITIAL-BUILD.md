# FID-20260523-SAVANT-CLI-INITIAL-BUILD

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-SAVANT-CLI-INITIAL-BUILD          |
| **Date Created** | 2026-05-23                                      |
| **Status**       | OPEN                                            |
| **Priority**     | CRITICAL                                        |
| **Phase**        | Phase 3 - Perfection Loop (Deep Audit Complete) |

## Context

Build a next-generation AI coding assistant CLI that beats all existing competitors (Claude Code, Crush, OpenCode, Gemini CLI, Kilo Code, OpenClaude, Codebuff) by synthesizing their best architectural patterns into a single, uncompromised executable.

**Key architectural decisions:**
- Language: Go (native Bubble Tea v2 TUI)
- MVP Scope: Full agent loop + TUI + multi-provider routing
- Providers: 9router gateway as primary + native OpenAI-compatible fallback (Xiaomi MiMo V2 Pro free, Ollama local)
- Design: Cyberpunk aesthetic (Void Indigo #0D0221, Hyper-Cyan #00F0FF, Solar Orange #FF6B35)
- Perfection Loop FSM to prevent infinite breakage cycles
- SQLite persistence via ncruces/go-sqlite3 (CGO-free)

## Issue: Build Savant CLI from scratch

### Symptoms
- No source code exists yet
- No project structure, no Go module, no dependencies
- Only docs/ and resoruces/ directories exist with reference materials

### Root Cause Analysis
This is a greenfield project. The architecture is fully designed in `docs/Architecting Next-Gen AI Coding Assistant.md` and validated against 8 competitor codebases in `resoruces/`.

### Fix Plan

#### Phase 1: Project Scaffold & Core Types
- Create `go.mod` with Go 1.25, Bubble Tea v2, Lip Gloss v2, ultraviolet deps
- Create `main.go` entry point with CLI flag parsing
- Create `internal/types/types.go` - Message, ToolCall, ToolResult, Session, Provider types
- Create `internal/config/config.go` - Config loading from `~/.savant/config.json`

#### Phase 2: Provider Layer
- `internal/provider/provider.go` - Provider interface
- `internal/provider/openai.go` - OpenAI-compatible client (9router, MiMo, Ollama)
- `internal/provider/anthropic.go` - Anthropic client
- `internal/provider/router.go` - Multi-provider router with fallback
- `internal/provider/9router.go` - 9router gateway integration

#### Phase 3: Tool System
- `internal/tools/tools.go` - Tool interface
- `internal/tools/bash.go`, `edit.go`, `write.go`, `read.go`, `glob.go`, `grep.go`, `fetch.go`, `agent.go`

#### Phase 4: Agent Loop & Perfection Loop
- `internal/agent/agent.go` - Main agent loop
- `internal/agent/perfection.go` - Perfection Loop FSM
- `internal/agent/context.go` - Context management
- `internal/agent/hooks.go` - Pre/post tool hooks

#### Phase 5: Bubble Tea v2 TUI (Cyberpunk)
- `internal/tui/tui.go` - Root model
- `internal/tui/chat.go`, `editor.go`, `sidebar.go`, `statusbar.go`
- `internal/tui/theme.go` - Cyberpunk theme
- `internal/tui/dialogs.go`, `layout.go`

#### Phase 6: Persistence
- `internal/db/db.go` - SQLite init + migrations
- `internal/session/service.go` - Session CRUD

#### Phase 7: Slash Commands
- `internal/commands/` - Command registry, /provider, /model, /session, /config, /help

### Verification Criteria
- [ ] `go build` succeeds with zero errors
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes (unit tests for each package)
- [ ] Binary runs and shows version
- [ ] TUI renders cyberpunk theme correctly
- [ ] Agent loop can stream a response from MiMo (free)
- [ ] Tool execution works (bash, read, edit, glob, grep)
- [ ] Perfection Loop FSM halts on thrashing detection
- [ ] 9router gateway routing works with fallback
- [ ] Session persistence survives restart

### Execution Phases

#### Phase 1: Init
- [x] Create FID
- [x] Set up git repo with remote (github.com/fame0528/savant-cli)
- [ ] Create directory structure

#### Phase 2: Planning
- [x] Architecture doc reviewed
- [x] 8 competitor codebases analyzed
- [x] Technology decisions made
- [x] OpenClaude provider system analyzed (openaiShim, smartRouting, providerConfig, agentRouting)

#### Phase 3: Perfection Loop
- [x] Deep Audit of architecture (see DEEP-AUDIT.md)
- [ ] Heuristic Enhancement
- [ ] Validation Strike

#### Phase 4: Implementation
- [ ] Phase 1-7 as described above

#### Phase 5: Test Repair
- [ ] All tests passing

#### Phase 6: Documentation
- [ ] README updated
- [ ] Code comments where non-obvious

#### Phase 7: Commit
- [ ] Local commit only (Push Gate applies)
