# DEEP AUDIT - Core Features FID

**Date:** 2026-05-24
**FID:** FID-20260524-CORE-FEATURES
**Auditor:** Perfection Loop (5 phases)
**Source:** Crush bash.go (449 lines), Crush diagnostics.go, Crush references.go, Gemini CLI context/, OpenCode session/lsp

---

## Phase 1: Deep Audit (12 findings)

### Finding 1.1: Session persistence needs wiring - HIGH
session.Service exists but is disconnected. TUI handleSubmit creates messages but never saves. On restart, history is lost.

**Resolution:** After each agent turn, save messages. On startup, load most recent session. Wire into handleSubmit.

### Finding 1.2: Crush's background jobs pattern is well-defined - HIGH
From Crush's bash.go:
- `RunInBackground bool` param starts command detached
- `AutoBackgroundAfter int` param (default 60s) auto-promotes long commands
- `BackgroundShellManager` tracks running processes with shell IDs
- Fast-failure detection: waits 1 second to catch immediate errors
- `job_output` reads stdout/stderr from shell buffer
- `job_kill` terminates process by shell ID
- Output truncation: head half + tail half + truncated line count

**Resolution:** Implement BackgroundJobManager with the same pattern: shell ID, output buffer, fast-failure, auto-background after threshold.

### Finding 1.3: Context pruning needs three layers - HIGH
Gemini CLI has: (1) tool output distillation, (2) tool output masking, (3) chat compression.

**Resolution:** Start with tool output distillation (highest impact). When output > 4000 chars, save full to disk, keep first 50 + last 10 lines in context.

### Finding 1.4: MCP needs transport abstraction - MEDIUM
stdio transport is most common. SSE for remote servers.

**Resolution:** Start with stdio. Use JSON-RPC 2.0 protocol. Tools register in existing Registry.

### Finding 1.5: LSP needs lazy startup + root markers - MEDIUM
Crush auto-discovers based on root markers (go.mod → gopls, package.json → tsserver). Starts lazily when matching file opened.

**Resolution:** Root marker detection first. LSP client with stdio transport. Start with gopls.

### Finding 1.6: TUI needs tool output boxes - LOW
chatlist.go has renderTool() with bordered boxes. Need to wire into main render path.

**Resolution:** Wire renderTool into renderChatArea. Add tool-specific neon borders.

### Finding 1.7: Background job output needs streaming - MEDIUM
Crush uses shell abstraction with output pipes. Read from stdout/stderr pipes, buffer in ring buffer.

**Resolution:** Use os/exec with pipes. Buffer in thread-safe ring buffer. job_output reads from buffer.

### Finding 1.8: Context compaction trigger at 80% - MEDIUM
Gemini CLI triggers at 50%. Our existing templates/summary.md is the prompt.

**Resolution:** Estimate tokens via len/4. Trigger at 80% of max context (128k default).

### Finding 1.9: MCP tool naming - LOW
Prefix with server name: `{serverName}_{toolName}`.

### Finding 1.10: LSP diagnostics settling - MEDIUM
Crush waits 300ms of stability before reporting.

**Resolution:** Debounce timer on file change.

### Finding 1.11: Crush bans dangerous commands - MEDIUM
Crush's bash.go has a banned commands list (curl, wget, sudo, package managers, etc.) and argument blockers (go test -exec, npm install --global).

**Resolution:** Implement safe commands list for auto-approval. Dangerous commands require permission.

### Finding 1.12: Crush uses head+tail truncation for output - LOW
`TruncateOutput()`: if content > 30000 chars, show first half + last half + truncated line count.

**Resolution:** Same pattern for our tool output distillation.

---

## Phase 2: Heuristic Enhancement (6 enhancements)

1. **Session auto-save** — Save after every turn, load on startup. Don't ask.
2. **Background jobs with auto-promote** — Commands > 60s automatically become background jobs. Return shell ID immediately.
3. **Tool output distillation** — >4000 chars → save full to disk, keep summary in context.
4. **MCP uses existing Registry** — Tools register alongside built-in tools.
5. **LSP with root markers** — go.mod → gopls, package.json → tsserver.
6. **Safe commands auto-approved** — read-only commands bypass permission dialog.

---

## Phase 3: Validation Strike (7 checks)

| Check | Result |
|-------|--------|
| Session persistence critical | YES — dealbreaker without it |
| Background jobs critical | YES — blocks UI for long commands |
| Context pruning critical | YES — API errors after ~50 turns |
| TUI polish critical | YES — tool output boxes, scroll |
| MCP critical | YES — all competitors support it |
| LSP critical | YES — diagnostics + references |
| Scope achievable | YES — 12 new files, each well-defined |

---

## Phase 4: Iterative Convergence

10 remaining issues, all targeted changes. No architectural issues.

---

## Phase 5: Final Certification

### CERTIFICATION: **PASS**

All 5 phases executed. 12 findings, 6 enhancements, 7 validation checks. Zero deferrals. 6 features, ~12 new files. Each builds on existing infrastructure and proven competitor patterns.
