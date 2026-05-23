# FID-20260524-CORE-FEATURES

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260524-CORE-FEATURES                      |
| **Date Created** | 2026-05-24                                      |
| **Status**       | OPEN                                            |
| **Priority**     | CRITICAL                                        |

## Context

savant-cli has the foundation built (agent loop, tools, TUI, providers, sub-agents, tool forge) but 6 critical features are missing to make it competitive. This FID covers all 6.

## Feature 1: Session Persistence

The `session.Service` exists but is never called. Conversations are lost on restart.

**What to build:**
- Save messages to SQLite after each agent turn
- Load most recent session on startup
- `/session list` shows available sessions
- `/session new` starts fresh
- Wire `sessionSvc` into TUI handleSubmit

**Files:** `internal/tui/tui.go` (modify), `internal/session/service.go` (exists)

## Feature 2: Background Jobs (from Crush)

The bash tool blocks until completion. Long-running commands (builds, tests, servers) need fire-and-forget.

**What to build:**
- `run_in_background` tool - starts command, returns job ID
- `job_output` tool - reads output from a background job
- `job_kill` tool - kills a background job
- Background job manager that tracks running processes

**Files:** `internal/tools/background.go` (new), `internal/tools/tools.go` (modify)

## Feature 3: Context Pruning (from Gemini CLI)

No token budget management. Long conversations will hit API errors.

**What to build:**
- Tool output distillation - save full output to disk, keep summary in context
- Tool output masking - protect newest 50k tokens, truncate older outputs
- Chat compression - summarize old messages when context approaches limit
- Token estimation using ~4 chars/token heuristic

**Files:** `internal/agent/context.go` (enhance), `internal/agent/compressor.go` (new)

## Feature 4: TUI Polish

The cyberpunk theme works but needs refinement.

**What to build:**
- Better message formatting with word wrapping
- Tool output in bordered boxes with tool-specific neon colors
- Proper scroll behavior (PgUp/PgDown, Home/End)
- Spinner animation during agent work
- Better status bar with token count and cost

**Files:** `internal/tui/tui.go` (modify), `internal/tui/chatlist.go` (modify)

## Feature 5: MCP Support

All competitors support MCP. Would unlock thousands of community tools.

**What to build:**
- MCP client that connects to configured servers (stdio/SSE transport)
- Tool discovery from MCP servers
- Dynamic tool registration
- Config section for MCP servers

**Files:** `internal/mcp/client.go` (new), `internal/mcp/transport.go` (new), `internal/config/config.go` (modify)

## Feature 6: LSP Integration (from Crush/OpenCode)

Code intelligence for better tool output.

**What to build:**
- LSP client manager with auto-discovery (gopls, typescript-language-server, etc.)
- `diagnostics` tool - shows LSP diagnostics for files
- `references` tool - find all references to a symbol
- Root marker detection (go.mod, package.json, etc.)
- Lazy startup (only start LSP when matching file type opened)

**Files:** `internal/lsp/client.go` (new), `internal/lsp/manager.go` (new), `internal/tools/lsp.go` (new)

## Verification Criteria

- [ ] Session saves to SQLite after each turn
- [ ] Session restores on restart
- [ ] `/session list` shows past sessions
- [ ] `run_in_background` starts a job and returns ID
- [ ] `job_output` reads background job output
- [ ] `job_kill` kills a background job
- [ ] Context compaction triggers at 80% usage
- [ ] Tool outputs are truncated when context is large
- [ ] Tool output bordered boxes with neon colors
- [ ] Proper scroll with PgUp/PgDown
- [ ] MCP client connects to configured servers
- [ ] MCP tools discoverable and callable
- [ ] LSP auto-discovers language servers
- [ ] `diagnostics` tool shows LSP diagnostics
- [ ] `references` tool finds all references
- [ ] No stubs or placeholders
