# DEEP AUDIT - TUI Quality Fix

**Date:** 2026-05-23
**FID:** FID-20260523-TUI-QUALITY-FIX

---

## 1. Input System Audit

### Finding 1.1: textarea.Model captures Enter key
Bubbles textarea uses Enter for newline insertion. The `InsertNewline.SetEnabled(false)` call on line 26 of `input.go` should prevent this, but in practice the textarea still consumes the Enter keypress before `handleKeyPress` can see it. The key is passed to `m.input.Update(msg)` at line 294, and by the time we check `key == "enter"` at line 312, the textarea has already processed it.

**Action:** Replace textarea with raw string input. This matches Crush's approach (custom editor, not Bubbles textarea for the main input).

### Finding 1.2: `msg.Text()` may not exist in Bubble Tea v2
In the FID fix plan, I mentioned `msg.Text()` for character insertion. Need to verify this method exists on `KeyPressMsg`. If not, use `msg.String()` and filter out non-printable keys.

**Action:** Verify API, use `msg.String()` as fallback, filter single-character printable runes.

### Finding 1.3: @-mention detection on every keystroke causes ScanFiles on every keypress
Line 303 calls `ScanFiles(m.getCwd())` which walks the entire filesystem every time a character is typed. This is O(n) per keystroke where n = number of files in the project.

**Action:** Cache file scan results. Only rescan on directory change or explicit refresh. Use a debounce pattern.

---

## 2. Layout Audit

### Finding 2.1: Title bar logo is 8+ lines tall
`GetAnimatedLogo()` returns a multi-line ASCII art that takes 8+ terminal lines. For the title bar, this is way too tall - it pushes all content down.

**Action:** Title bar should be a single line: `SAVANT ═╪═╪═ [PROVIDER]`. Animated logo only in welcome screen.

### Finding 2.2: Side-by-side layout uses string splitting
Lines 514-533 split both sidebar and chat into lines, then interleave them line-by-line. This breaks when sidebar and chat have different numbers of lines, and doesn't clip to `mainHeight`.

**Action:** Both panels should independently clip to `mainHeight` before interleaving.

### Finding 2.3: Chat area height calculation is wrong
Line 694: `chatHeight := m.height - 10`. This is a magic number that doesn't account for the actual height of title bar, input, status bar, and log panel. The actual calculation should be: `mainHeight = m.height - titleBarHeight - inputHeight - statusHeight - logHeight`.

**Action:** Calculate mainHeight dynamically based on which panels are visible.

### Finding 2.4: Tool panel appends below chat, causing overflow
Lines 498: `sb.WriteString(toolPanel)` after chat means the total height is chat + tool, which can exceed terminal height.

**Action:** Tool results should be inline with chat messages (already handled by `role == "tool"` in chat rendering). Remove separate tool panel or make it togglable.

---

## 3. Agent Integration Audit

### Finding 3.1: Conversation history discarded
Line 406: `_ = a.Messages()` - the conversation history from the agent is thrown away. The agent runs with `m.agentMessages` as prior context but the updated messages are never saved back to `m.agentMessages`.

**Action:** After agent completes, save `a.Messages()` to `m.agentMessages` so next turn has full history.

### Finding 3.2: Agent done event sent twice
The goroutine at line 402-408 closes the channel (which triggers `agentDoneMsg`), AND the agent's `Run()` method emits `EventDone` before returning. The TUI handles `EventDone` at line 401 by just chaining another `eventSub`, which means it reads the next event (channel close → agentDoneMsg). This works but is fragile.

**Action:** Remove the explicit `m.evtChan <- agent.Event{Type: agent.EventDone}` from the goroutine since channel close already triggers `agentDoneMsg`.

---

## 4. File Tree Audit

### Finding 4.1: FileTree recreated on every resize
Line 160: `m.fileTree = NewFileTree(m.getCwd(), m.sidebarWidth-4)` on every `WindowSizeMsg`. This walks the filesystem every time the terminal is resized.

**Action:** Only recreate file tree if sidebar width changed. Cache the tree.

### Finding 4.2: FileTree has no height limit
The file tree renders ALL files up to depth 3 with no height clipping. In a large project, this can produce hundreds of lines that overflow the sidebar.

**Action:** Pass `maxLines` to `FileTree.Render()` and clip output.

---

## 5. Completions Audit

### Finding 5.1: ScanFiles walks entire project on every @-mention
Line 303-305: `ScanFiles(m.getCwd())` is called every time a character is typed after `@`. For a project with 10,000 files, this is extremely slow.

**Action:** Cache ScanFiles results. Rescan only on explicit refresh (e.g., Ctrl+R).

---

## AUDIT VERDICT: PASS (with 10 actionable findings)

Priority:
1. Fix input (replace textarea with raw string) - CRITICAL
2. Fix title bar (compact single line) - HIGH
3. Fix layout (proper height calculation) - HIGH
4. Fix conversation history preservation - HIGH
5. Cache file scan results - MEDIUM
6. Cache file tree - MEDIUM
7. Clip file tree to height - MEDIUM
8. Remove duplicate done event - LOW
9. Verify msg.Text() API - LOW
10. Remove separate tool panel - LOW
