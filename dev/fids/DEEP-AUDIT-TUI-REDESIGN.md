# DEEP AUDIT - TUI Redesign FID

**Date:** 2026-05-23
**FID:** FID-20260523-TUI-REDESIGN
**Auditor:** Perfection Loop (5 phases)
**Codebase Reviewed:** All 29 Go source files, 8 competitor codebases, MiMo API verified

---

## Phase 1: Deep Audit

### Finding 1.1: StreamDelta lacks Reasoning field - CRITICAL
**File:** `internal/provider/provider.go:92-96`

Current `StreamDelta` struct:
```go
type StreamDelta struct {
    Role      string     `json:"role,omitempty"`
    Content   string     `json:"content,omitempty"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
```

MiMo V2 Pro returns `"reasoning"` and `"reasoning_details"` in the delta. These fields are silently dropped by `json.Unmarshal`. The agent never sees the thinking content. The `EventThinking` type from the FID has no corresponding `EventType` constant.

**Impact:** MiMo's reasoning/thinking content is completely lost. The user sees only the final content, not the model's chain of thought.

**Fix:** Add `Reasoning string \`json:"reasoning,omitempty"\`` to `StreamDelta`. Add `EventThinking EventType = iota` to agent.go. Emit thinking events during stream processing.

### Finding 1.2: Conversation history discarded in TUI - CRITICAL
**File:** `internal/tui/tui.go:406-408`

Current code:
```go
go func() {
    a.Run(m.ctx, prompt)
    close(m.evtChan)
}()
```

`a.Messages()` is never called. The agent's accumulated conversation history (including tool calls and results) is thrown away after every turn. Each message is a one-shot request with zero context.

**Impact:** The AI has no memory of previous turns. Multi-turn conversations are impossible.

**Fix:** After `a.Run()`, save `a.Messages()` to `m.agentMessages` before closing the channel. The TUI needs to pass `m.agentMessages` into the next agent.

### Finding 1.3: ContextManager exists but is never used - HIGH
**File:** `internal/agent/context.go` (entire file)

The `ContextManager` with `NeedsCompaction()` and `Compact()` is fully implemented but never called. The agent loop in `agent.go` appends messages indefinitely. Long conversations will eventually exceed the model's context window and get API errors.

**Impact:** No context compaction. API errors on long conversations.

**Fix:** In the agent loop, check `cm.NeedsCompaction(a.messages)` before each turn. If true, call `cm.Compact()` and replace `a.messages` with the compacted result.

### Finding 1.4: Theme is blue/purple, not near-black with neons - HIGH
**File:** `internal/tui/theme.go`

Current colors:
- Background: `#0D0221` (Void Indigo - purple)
- Primary accent: `#00F0FF` (HyperCyan)
- Secondary accent: `#FF6B35` (SolarOrange)

None of these match the FID spec. The entire theme needs rewriting.

**Impact:** UI looks wrong per user's requirements.

**Fix:** Rewrite all color constants and style initialization in theme.go.

### Finding 1.5: File tree sidebar still exists - HIGH
**File:** `internal/tui/tui.go:417` + `internal/tui/filetree.go`

The sidebar still shows the file tree from `filetree.go`. The FID specifies removing it in favor of session/status info.

**Impact:** Sidebar shows useless file tree instead of session context.

**Fix:** Remove `fileTree` field from Model. Replace `renderFileTreePanel()` with session info panel. Keep `filetree.go` for now (used by completions ScanFiles), just don't render it in the sidebar.

### Finding 1.6: Tool output is single-line, not bordered boxes - MEDIUM
**File:** `internal/tui/chatlist.go:94-101`

Current tool rendering:
```go
case "tool":
    icon := theme.ToolIcon.Render("⚡")
    name := theme.ToolName.Render(ci.tool)
    content := ci.content
    if width > 18 && len(content) > width-18 {
        content = content[:width-21] + "..."
    }
    lines = []string{theme.ToolMessage.Render(fmt.Sprintf("   %s %s: %s", icon, name, content))}
```

Single line with truncation. The FID specifies bordered boxes with tool-specific colors and collapsible output.

**Impact:** Tool output is cramped and hard to read.

**Fix:** Rewrite `renderFresh()` for "tool" role to produce bordered box with first N lines, collapsible with Enter/Space.

### Finding 1.7: No permission system for write tools - MEDIUM
**File:** `internal/tools/tools.go:91-120`

`ExecuteAll` runs all tools without any approval check. The FID specifies that write tools (bash, edit, write) require user confirmation.

**Impact:** No safety check on destructive operations.

**Fix:** Add permission check before executing write tools. Use DialogOverlay to show confirmation dialog. Read-only tools (read, glob, grep) auto-approved.

### Finding 1.8: Session persistence wired but not functional - MEDIUM
**File:** `internal/tui/tui.go` (sessionSvc field exists but never called)

The TUI model stores `sessionSvc` but never creates sessions or saves messages. The session service in `internal/session/service.go` is fully implemented but disconnected.

**Impact:** No session history across restarts.

**Fix:** Create a session on first message. Save each message after it's added. Load most recent session on startup.

### Finding 1.9: `wordWrap` counts bytes, not runes - LOW
**File:** `internal/tui/chatlist.go:248`

```go
if len(current)+1+len(word) <= width {
```

`len()` counts bytes. For CJK characters or emoji, this produces incorrect wrapping. A 2-character CJK string has byte length 6 but display width 4.

**Impact:** Lines may overflow for non-ASCII text.

**Fix:** Use `utf8.RuneCountInString()` or `runewidth.StringWidth()` for width calculation.

### Finding 1.10: No virtual scrolling - MEDIUM
**File:** `internal/tui/tui.go:548-559`

```go
for _, msg := range m.messages {
    // ... render each message
}
if len(lines) > height {
    lines = lines[len(lines)-height:]
}
```

Every message is rendered on every frame. For 100+ messages, this is O(n) per frame with no caching benefit (since messages are `chatMessage` structs, not `ChatItem` with cache).

**Impact:** Performance degrades with long conversations.

**Fix:** Use the `ChatList` from `chatlist.go` which has per-item caching. Only re-render items that changed (streaming content).

---

## Phase 2: Heuristic Enhancement

### Enhancement 1: Semantic neon color assignments
Assign each neon color to a specific UI semantic:
- **NeonPink (#FF00FF)** → User messages border, input prompt
- **NeonCyan (#00FFFF)** → Assistant messages border, info, status
- **NeonGreen (#00FF41)** → Tool output boxes, success indicators
- **NeonYellow (#F0FF00)** → Warnings, active/selected items, thinking text
- **NeonRed (#FF0040)** → Errors, critical, permission deny
- **NeonOrange (#FF6B35)** → Tool names, secondary accent, edit diffs

### Enhancement 2: Compact status bar with neon separators
` SAVANT │ mimo-v2 │ $0.00 │ 47 tok │ 3 turns `
Where `│` characters are rendered in NeonCyan, text in TextDim.

### Enhancement 3: Tool output bordered box pattern
```
  ┌─ ⚡ bash ─────────────────────────┐
  │ ls -la                             │
  │ total 48                           │
  │ drwxr-xr-x 12 user staff 384 May  │
  └────────────────────────────────────┘ (10 lines, expandable)
```
- Top border: tool icon + tool name (in tool's color)
- Side borders: dim gray (`│`)
- Content: light gray (#E0E0E0)
- Bottom border: collapse indicator

### Enhancement 4: MiMo reasoning display
When `reasoning` field appears in stream delta:
- Emit `EventThinking` event with reasoning content
- Display as: `  💭 Thinking: <reasoning text>` in NeonYellow dim style
- Separate from final `content` which appears as normal assistant message

### Enhancement 5: Use ChatList for message rendering
Replace the raw `[]chatMessage` slice with the `ChatList` from chatlist.go. This gives:
- Per-item render caching (frozen items not re-rendered)
- Streaming message updates without cache invalidation of other items
- Built-in scroll management
- Foundation for virtual scrolling

### Enhancement 6: Conversation history bridge
After agent completes, save messages to `m.agentMessages`:
```go
go func() {
    a.Run(m.ctx, prompt)
    m.agentMessages = a.Messages() // Save before channel close
    close(m.evtChan)
}()
```

But this has a race condition - the goroutine writes to `m.agentMessages` while the TUI goroutine reads `m`. Need a message-based approach:
```go
go func() {
    a.Run(m.ctx, prompt)
    evtChan <- agent.Event{Type: EventDone}
    close(evtChan)
}()
// In handleAgentEvent for EventDone:
// a.Messages() is accessible via closure
```

### Enhancement 7: Permission flow integration
Before executing write tools in the agent loop:
1. Check if tool is read-only (read, glob, grep) → auto-approve
2. If write tool → emit `EventPermissionRequest` with tool name and args
3. TUI shows ConfirmDialog
4. User approves/denies
5. Result sent back to agent via a response channel
6. Agent executes or skips the tool

This requires a bidirectional channel: agent → TUI for events, TUI → agent for permission responses.

---

## Phase 3: Validation Strike

### V1: Color palette correctness
PASS. Near-black `#0A0A0A` with neon accents matches user's explicit requirement. Semantic assignments ensure readability (body text stays `#E0E0E0`, neons only for accents/borders/headers).

### V2: Layout feasibility
PASS. Full-width chat with optional sidebar is proven by Gemini CLI (no sidebar) and Crush (optional sidebar). Single-line status bar at top, single-line keybind bar at bottom. All achievable within Bubble Tea v2's View struct.

### V3: MiMo V2 Pro compatibility
PASS (with fix needed). API verified working via curl test. The `reasoning` field IS returned but our `StreamDelta` struct doesn't capture it. Fix is adding one field to the struct and one EventType constant.

### V4: Scope achievability
PASS. All 12 features modify existing files. No new packages. Estimated changes:
- theme.go: ~200 lines (complete rewrite of colors)
- tui.go: ~300 lines (layout, sidebar, session wiring, permissions)
- chatlist.go: ~150 lines (tool renderers, collapsible output)
- agent.go: ~30 lines (reasoning field, history preservation)
- provider.go: ~5 lines (add Reasoning field)
- context.go: ~10 lines (wire into agent loop)
- dialog.go: ~50 lines (permission dialog)
- main.go: ~20 lines (session restore)
- logo.go: ~20 lines (color updates)

### V5: Missing features
NO DEFERRALS. All features from the FID are included:
- Context compaction: ✅ context.go exists, needs wiring
- Session persistence: ✅ session/service.go exists, needs wiring
- Permission system: ✅ dialog.go exists, needs PermissionDialog type
- Virtual scrolling: ✅ chatlist.go has caching, needs wiring into tui.go
- Tool renderers: ✅ chatlist.go needs tool-specific render method
- MiMo reasoning: ✅ provider.go needs field, agent.go needs event type

### V6: Conversation history preservation
NEEDS FIX. Current code in `handleSubmit()` starts a goroutine that runs the agent but never captures the resulting messages. The `a.Messages()` call is not made. Fix is in Enhancement 6.

### V7: Race condition in agent message preservation
NEEDS DESIGN. The goroutine writes `m.agentMessages` while the TUI goroutine reads the model. Bubble Tea v2 uses a value-receiver Update pattern, so the model is copied on each update. The goroutine needs to send the messages back via a channel event, not write to the model directly.

**Solution:** Add `EventHistoryUpdate` event type that carries `[]provider.ChatMessage`. The goroutine sends this event before closing the channel. The TUI handles it by updating `m.agentMessages`.

---

## Phase 4: Iterative Convergence

### Remaining issues after Phase 3:

1. **Reasoning field missing** → 5-line change in provider.go + 3 lines in agent.go
2. **History discarded** → 10-line change in tui.go handleSubmit + new EventHistoryUpdate
3. **Context not compacted** → 15-line change in agent.go to call Compact()
4. **Theme colors wrong** → 200-line rewrite of theme.go (straightforward)
5. **File tree sidebar** → 30-line change in tui.go to show session info
6. **Tool renderers** → 100-line addition to chatlist.go
7. **Permission system** → 50-line addition to dialog.go + 20 lines in agent.go
8. **Session persistence** → 30-line wiring in tui.go + main.go
9. **Virtual scrolling** → Already implemented in chatlist.go, need to wire into tui.go (replace raw []chatMessage with ChatList)
10. **Byte vs rune in wordWrap** → 5-line fix in chatlist.go

All 10 issues are targeted, small-to-medium changes. No architectural rewrites needed.

### Convergence check:
- No issue requires changing the Provider interface (only adding a field to StreamDelta)
- No issue requires changing the Tool interface
- No issue requires new packages
- All issues can be fixed in the 11 files listed in the FID

### Race condition resolution:
The agent goroutine communicates via `chan agent.Event`. Adding `EventHistoryUpdate` is the clean solution:
```go
type Event struct {
    Type     EventType
    Content  string
    Tool     string
    Error    error
    Messages []provider.ChatMessage // For EventHistoryUpdate
}
```

The goroutine sends `Event{Type: EventHistoryUpdate, Messages: a.Messages()}` before closing the channel. The TUI handles it by assigning to `m.agentMessages`.

---

## Phase 5: Final Certification

### Build verification chain:
1. `go build -o savant.exe .` → PASS (verified before this audit)
2. `go vet ./...` → PASS
3. `./savant.exe --version` → PASS
4. MiMo V2 Pro API test → PASS (`mimo-v2-pro` model, `/v1/chat/completions`)

### Design completeness:
- [x] Color palette: 11 colors defined with semantic assignments
- [x] Layout: ASCII mockup with status bar, chat, input, keybind bar
- [x] Tool rendering: Bordered boxes per tool type, collapsible
- [x] MiMo reasoning: Field addition + event type + display logic
- [x] Context compaction: Existing code, needs wiring
- [x] Session persistence: Existing code, needs wiring
- [x] Permission system: Existing DialogOverlay, needs PermissionDialog
- [x] Virtual scrolling: Existing ChatList caching, needs wiring
- [x] Conversation history: Race-safe via EventHistoryUpdate
- [x] Sidebar: Session info instead of file tree

### No stubs:
Every feature has a concrete implementation path with existing code to build on. No placeholders, no "future work", no deferrals.

### CERTIFICATION: **PASS**

The FID is complete and ready for implementation. All 5 phases of the Perfection Loop have been executed. 10 actionable findings identified, all with concrete fixes. Zero deferrals. Scope is achievable in 11 file changes totaling approximately 600 lines of new/modified code.
