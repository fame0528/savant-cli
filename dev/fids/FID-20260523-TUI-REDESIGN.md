# FID-20260523-TUI-REDESIGN

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-TUI-REDESIGN                       |
| **Date Created** | 2026-05-23                                      |
| **Status**       | APPROVED                                        |
| **Priority**     | CRITICAL                                        |
| **Certification** | Perfection Loop PASSED (all 5 phases)           |

## Context

Complete TUI redesign based on:
- User: "UI is horrible and non-functional"
- User: "Near-black background with neon highlights"
- User: "I like the design of the Kilo CLI"
- User: "MiMo V2 Pro is the best model"
- User: "Nothing gets deferred - build everything"
- Competitor research: Crush, OpenCode, Gemini CLI, Kilo Code, Codebuff

## Design Spec

### Color Palette (Neon on Black)

| Name | Hex | Usage |
|------|-----|-------|
| Background | #0A0A0A | Main background |
| Surface | #141414 | Panel/card backgrounds |
| Border | #222222 | Default borders |
| Text | #E0E0E0 | Primary body text |
| TextDim | #666666 | Secondary text |
| NeonPink | #FF00FF | User messages, input prompt border |
| NeonCyan | #00FFFF | Assistant messages, info, status |
| NeonGreen | #00FF41 | Tool output, success |
| NeonYellow | #F0FF00 | Warnings, active/selected items |
| NeonRed | #FF0040 | Errors, critical |
| NeonOrange | #FF6B35 | Tool names, secondary accent |

### Layout

```
┌──────────────────────────────────────────────────────┐
│ SAVANT │ mimo-v2 │ $0.00 │ 47 tok │ 3 turns          │ <- status bar (top)
├──────────────────────────────────────────────────────┤
│                                                      │
│  YOU [14:23]                                         │
│  What files are in this project?                     │
│                                                      │
│  SAVANT [14:23]                                      │
│  Let me check the project structure.                 │
│                                                      │
│  ┌─ ⚡ bash ────────────────────────────────┐        │
│  │ ls -la                                   │        │
│  │ total 48                                 │        │
│  │ drwxr-xr-x 12 user staff 384 May 23 .   │        │
│  └──────────────────────────────────────────┘        │
│                                                      │
│  ┌─ ⚡ glob ────────────────────────────────┐        │
│  │ main.go                                   │        │
│  │ internal/agent/agent.go                   │        │
│  │ ... (15 files)                            │        │
│  └──────────────────────────────────────────┘        │
│                                                      │
│  The project has 15 Go files organized in...         │
│                                                      │
├──────────────────────────────────────────────────────┤
│ ▸ █                                                   │ <- input
├──────────────────────────────────────────────────────┤
│ Ctrl+B Sidebar │ Ctrl+L Logs │ Ctrl+P Commands       │ <- keybind bar
└──────────────────────────────────────────────────────┘
```

### Sidebar (toggle with Ctrl+B)

When toggled on, shrinks chat area and shows right-side panel (30 cols):
- Session info (turn count, messages, provider, model)
- Token usage + cost
- Modified files list
- Pet status (name, mood, HP/XP bars)

### Tool-Specific Renderers (ALL tools, not deferred)

Each tool gets a distinct bordered box with collapsible output (10 lines default):

| Tool | Border Color | Header | Content |
|------|-------------|--------|---------|
| bash | NeonGreen | ⚡ bash: `command` | Output in TextDim |
| read | NeonCyan | ⚡ read: path | Line-numbered content |
| edit | NeonOrange | ⚡ edit: path | Diff (old -> new) |
| write | NeonOrange | ⚡ write: path | File content |
| glob | NeonGreen | ⚡ glob: pattern | File list with count |
| grep | NeonYellow | ⚡ grep: pattern | Match results with line numbers |

Enter/Space expands collapsed tool output.

### Message Styles

| Type | Left Border | Text Color | Header |
|------|-------------|-----------|--------|
| User | NeonPink | #FFFFFF | `YOU [HH:MM]` |
| Assistant | NeonCyan | #E0E0E0 | `SAVANT [HH:MM]` |
| Tool | NeonGreen | #E0E0E0 | `tool_name` (in NeonOrange) |
| System | None | TextDim | `message` |

### MiMo Reasoning Field (not deferred)

MiMo V2 Pro returns `reasoning` and `reasoning_details` in the API response:
- Capture reasoning content separately
- Display as dim/italic text prefixed with `Thinking:`
- Show between tool calls and final content
- Update `internal/provider/openai.go` to parse `reasoning` field
- Update `internal/agent/agent.go` to emit reasoning events

### Context Compaction (not deferred)

When conversation approaches the model's context limit:
- Summarize messages older than the last 6 turns
- Replace with a system message: "[Context compacted: N messages summarized]"
- Preserve the last 6 messages in full
- Implementation in `internal/agent/context.go`

### Session Persistence (not deferred)

- Save messages to SQLite after each turn
- Load previous session on startup (most recent)
- `/session list` shows available sessions
- `/session new` starts fresh
- `/session switch <id>` loads a session
- Wire `session.Service` into TUI model properly

### Permission System (not deferred)

- Read-only tools (read, glob, grep): auto-approved
- Write tools (bash, edit, write): require user confirmation
- Dialog overlay with Approve/Deny buttons
- `/permissions` to configure auto-approve rules
- Integration with existing DialogOverlay system

### Virtual Scrolling (not deferred)

- Only render messages visible in the viewport
- Track total message count and viewport position
- Efficient scroll with Up/Down/Page keys
- Lazy-render cached items (reuse ChatItem caching from chatlist.go)

### MiMo Reasoning Handling

MiMo V2 Pro returns reasoning/thinking content alongside regular content. The agent loop must:
1. Parse `reasoning` field from SSE stream chunks
2. Emit as `EventThinking` type events
3. Display thinking content in dim/italic style
4. Separate from final response content

## Files to Change

| File | Change |
|------|--------|
| `internal/tui/theme.go` | Complete color palette rewrite (neon on black) |
| `internal/tui/tui.go` | New layout, inline tool rendering, sidebar redesign, session wiring, permission system, virtual scrolling |
| `internal/tui/logo.go` | Update animated logo colors to neon palette |
| `internal/tui/chatlist.go` | Tool-specific renderers, collapsible output, virtual scrolling |
| `internal/tui/completions.go` | Update colors to neon palette |
| `internal/tui/dialog.go` | Add permission dialog type |
| `internal/agent/agent.go` | Handle MiMo reasoning field, conversation history preservation |
| `internal/agent/context.go` | Context compaction logic |
| `internal/provider/openai.go` | Parse reasoning field from response |
| `internal/provider/provider.go` | Add Reasoning field to ChatResponse/StreamDelta |
| `main.go` | Wire session save/restore, permission system |

## Verification Criteria

- [ ] Near-black background (#0A0A0A) with neon accents
- [ ] Full-width chat by default (no sidebar)
- [ ] Sidebar toggleable with Ctrl+B (30 cols, right side)
- [ ] Tool output in bordered boxes with tool-specific colors
- [ ] Tool output collapsible (10 lines default, expandable)
- [ ] Status bar at top: provider, model, cost, tokens, turns
- [ ] Keybind bar at bottom
- [ ] User messages have pink left border, white text
- [ ] Assistant messages have cyan left border, gray text
- [ ] MiMo reasoning field displayed as thinking text
- [ ] Context compaction works when approaching token limit
- [ ] Session persistence: save/restore from SQLite
- [ ] Permission system: write tools require approval dialog
- [ ] Virtual scrolling for 100+ messages
- [ ] Conversation history preserved between turns
- [ ] Enter submits, all keys work correctly
- [ ] MiMo V2 Pro responds via OpenGateway
- [ ] Animated logo in welcome screen only
