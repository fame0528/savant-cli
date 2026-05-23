# FID-20260523-TUI-QUALITY-FIX

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-TUI-QUALITY-FIX                    |
| **Date Created** | 2026-05-23                                      |
| **Status**       | OPEN                                            |
| **Priority**     | CRITICAL                                        |
| **Phase**        | Phase 1 - Perfection Loop                       |

## Context

User reports: "UI is horrible, doesn't actually function, has two logos, chat doesn't work, doesn't auto-resize or scale."

## Issue: TUI Quality Problems

### Symptoms
1. Two logos visible - title bar logo AND welcome screen logo
2. Chat doesn't work - Enter key conflicts with Bubbles textarea (textarea captures Enter for newline)
3. No auto-resize - terminal resize doesn't properly relayout
4. Looks terrible - layout is messy, components don't fit together

### Root Cause Analysis
1. **Two logos**: `renderTitleBar()` uses the huge multi-line animated logo (8+ lines), and `renderWelcome()` also renders the full logo. The title bar should be a compact single-line header.
2. **Chat broken**: Using `textarea.Model` from Bubbles captures Enter for newline insertion. The `handleKeyPress` function checks for Enter AFTER passing to textarea, but textarea already consumed it. Need to either disable Enter in textarea or use raw string input.
3. **No auto-resize**: `WindowSizeMsg` updates `m.width` and `m.height` but the sidebar width, chat area, and input width don't properly recalculate. The `renderMainArea` doesn't account for height properly.
4. **Layout issues**: Side-by-side layout uses string splitting and manual line-by-line assembly which doesn't handle different-height panels well.

### Fix Plan

#### 1. Compact Title Bar
- Replace multi-line animated logo with single-line `SAVANT` text in title bar
- Keep animated logo ONLY in welcome screen (when no messages)
- Title bar: `SAVANT ═══╪═╪═ [PROVIDER]`

#### 2. Fix Input (replace textarea with raw string)
- Remove Bubbles textarea dependency for input
- Use raw string + cursor position (which we had before and worked)
- Handle Enter directly in key handler to submit
- Handle all keys manually (backspace, left/right, home/end)
- This gives us full control over key behavior

#### 3. Fix Auto-Resize
- On `WindowSizeMsg`, recalculate:
  - `chatWidth = width - sidebarWidth - 1` (if sidebar shown)
  - `inputWidth = chatWidth - 4` (for prompt padding)
  - `mainHeight = height - titleBar - input - status - logs`
- Use height-aware rendering: clip content to available height

#### 4. Fix Layout
- Title bar: single line, always
- Main area: sidebar + chat side by side, both clipped to `mainHeight`
- Input: single line with prompt, full width
- Status bar: single line, full width
- Welcome screen: animated logo + help text, fills available space

### Verification Criteria
- [ ] Only ONE logo visible at a time (welcome screen has animated, title bar has compact)
- [ ] Enter key submits message immediately
- [ ] Typing works normally (characters appear, backspace deletes)
- [ ] Terminal resize causes proper relayout
- [ ] Sidebar + chat area fit together without overflow
- [ ] Input area stays at bottom of screen
- [ ] Status bar stays at very bottom

### Files to Change
- `internal/tui/tui.go` - Major rewrite of View() and handleKeyPress()
- `internal/tui/input.go` - Remove textarea dependency, use raw string input
- `internal/tui/theme.go` - No changes needed
- `internal/tui/logo.go` - No changes needed
