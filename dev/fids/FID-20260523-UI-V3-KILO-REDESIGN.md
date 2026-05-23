# FID-20260523-UI-V3-KILO-REDESIGN

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-UI-V3-KILO-REDESIGN               |
| **Date Created** | 2026-05-23                                      |
| **Status**       | OPEN (awaiting approval)                        |
| **Priority**     | CRITICAL                                        |

## Context

User wants a complete UI redesign inspired by Kilo CLI's design language:
- Near-black background with neon accents (yellow, pink, red, green)
- Sidebar with useful content (not file tree)
- Kilo-style left-border thread aesthetic
- MiMo V2 Pro as foundation model

## Design Spec

### Color Palette (Cyberpunk Neon on Black)

| Role | Color | Hex |
|------|-------|-----|
| Background | Near black | `#0A0A0A` |
| Surface/Panel | Dark gray | `#141414` |
| Border | Dim gray | `#1E1E1E` |
| Text | Light gray | `#E0E0E0` |
| Text muted | Medium gray | `#808080` |
| Primary accent | Neon yellow | `#FAF74F` |
| Secondary accent | Neon pink | `#FF00FF` |
| Error/Danger | Neon red | `#FF0040` |
| Success | Neon green | `#00FF41` |
| Info/Link | Neon cyan | `#00F0FF` |
| Warning | Neon orange | `#FF6B35` |

### Layout (Session Mode)

```
┌──────────────────────────────────────────────────────────────────┐
│ SAVANT [provider] ═══╪═╪═╪═╪═╪═╪═╪═ [model] [tokens] [cost]   │
├────────────────────────────────────────────────┬─────────────────┤
│                                                │                 │
│  ┃ SAVANT                                      │ Session Info    │
│  ┃ Hello! How can I help?                      │ ─────────────── │
│                                                │ CWD: /proj      │
│  ┃ YOU                                         │ Model: mimo     │
│  ┃ Read the config file                        │ Tokens: 1.2k    │
│                                                │ Cost: $0.00     │
│  ┃ SAVANT                                      │ ─────────────── │
│  ┃ I'll read the config file.                  │ Todo            │
│  ┃                                             │ □ Fix bug       │
│  ┃ ✱ Read config.json                          │ □ Add tests     │
│  ┃ ┃ { "key": "value", ... }                   │ ─────────────── │
│  ┃ ┃ (24 lines)                                │ Modified Files  │
│  ┃                                             │ src/main.go +5  │
│  ┃ ✱ Bash: `cat config.json`                   │ src/util.go -2  │
│  ┃ ┃ output: {"key":"value"}                   │ ─────────────── │
│  ┃                                             │ LSP             │
│  ┃ The config contains...                      │ ● gopls (OK)    │
│                                                │ ─────────────── │
│                                                │ Pet: Byte 🐣    │
│                                                │ HP ████████░░   │
├────────────────────────────────────────────────┤ XP ████░░░░░░   │
│ ▸ Type a message...                        █   │                 │
├────────────────────────────────────────────────┴─────────────────┤
│ mimo-v2-pro │ Turns:3 │ Ctrl+C:Quit │ Ctrl+S:Sidebar │ Ctrl+L:Logs │
└──────────────────────────────────────────────────────────────────┘
```

### Sidebar Sections (Kilo-style, 30-42 cols)

1. **Session Info** — provider, model, tokens used, cost
2. **Todo List** — agent-managed task list (collapsible)
3. **Modified Files** — files changed this session (+additions/-deletions)
4. **LSP Status** — connected language servers with status dots
5. **Pet** — compact pet status (HP/XP bars, mood)
6. **Footer** — working directory, version

### Message Design (Kilo-style)

- Left border: `┃` (thick vertical bar) in neon yellow
- User messages: left border + content
- Assistant messages: left border + content
- Tool output: left border + indented panel with collapsible content
- System messages: no border, dim text

### Title Bar (Single Line)

` SAVANT ─── provider ─── model ─── tokens ─── cost ──── turns `

Compact, single line. No animated logo in title bar (logo only in welcome screen).

### Welcome Screen (No Messages)

```
    [Animated Savant ASCII Logo - shimmer/glow effect]
    ─────────────────────────────────────────
    
    Type a message to start. MiMo V2 Pro ready.
    
    /help  /provider  /model  /config  /pet
    
    Ctrl+S  Sidebar  Ctrl+L  Logs  Ctrl+P  Commands
```

### Input Area

Single line with prompt `▸ ` and cursor block `█`.
Working state shows spinner: `⠋ Processing... (Ctrl+C to cancel)`

### Status Bar (Bottom)

` provider │ Turns:N │ Ctrl+C:Quit │ Ctrl+S:Sidebar `

### Keybindings

| Key | Action |
|-----|--------|
| Enter | Submit message |
| Ctrl+C | Cancel (working) / Quit |
| Ctrl+S | Toggle sidebar |
| Ctrl+L | Toggle log panel |
| Ctrl+P | Command palette (dialog) |
| Tab | Cycle sidebar sections |
| Up/Down | Scroll chat |
| Esc | Clear input |

### Files to Change

| File | Change |
|------|--------|
| `internal/tui/theme.go` | Complete color palette overhaul (near-black + neons) |
| `internal/tui/tui.go` | New layout: Kilo-style sidebar, thread borders, single-line title |
| `internal/tui/logo.go` | Keep animated logo for welcome screen only |
| `internal/tui/filetree.go` | Remove (replace with session info sidebar) |
| `internal/tui/completions.go` | No changes |
| `internal/tui/dialog.go` | No changes |
| `internal/tui/chatlist.go` | Add left-border rendering |

### Verification Criteria

- [ ] Background is near-black (#0A0A0A), not blue-purple
- [ ] All accents are neon (yellow, pink, red, green)
- [ ] Sidebar shows session info, todos, modified files, LSP, pet
- [ ] No file tree in sidebar
- [ ] Messages have left `┃` border (Kilo thread style)
- [ ] Title bar is single line
- [ ] Animated logo only in welcome screen
- [ ] MiMo V2 Pro as default model
- [ ] Chat works (Enter submits, conversation history preserved)
- [ ] Auto-resize on terminal size change
