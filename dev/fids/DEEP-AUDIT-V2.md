# DEEP AUDIT V2 - Savant CLI Overhaul FID

**Date:** 2026-05-23
**FID:** FID-20260523-SAVANT-V2-OVERHAUL

---

## 1. UI Layer Audit

### Finding 1.1: TUI is a monolith - needs component extraction
The entire UI is in a single 800+ line file. Crush uses a component-based architecture where Chat, List, Sidebar, Editor, and Dialogs are separate structs with their own methods. The main model delegates to them.

**Action:** Extract components: ChatView, SidebarView, InputEditor, StatusBar, DialogManager.

### Finding 1.2: Input editor uses manual character insertion instead of Bubbles textarea
The current input handling manually inserts runes and tracks cursor position. Bubble Tea v2 has `textarea.Model` from the Bubbles library that handles all of this correctly (multi-line, undo, paste, selection).

**Action:** Replace manual input with `charm.land/bubbles/v2/textarea`.

### Finding 1.3: Chat rendering has no caching
Every frame re-renders all messages. Crush's list component caches rendered items and freezes finished ones. This is critical for performance with long conversations.

**Action:** Implement per-message rendering cache. Once a message is finalized (not streaming), cache its rendered output.

### Finding 1.4: No completions/autocomplete system
When user types `@`, no file picker appears. This is a core UX pattern in every competitor.

**Action:** Add `@`-triggered completions popup that shows matching files.

### Finding 1.5: No dialog system for modals
No way to show model selector, permission prompts, or settings dialogs as overlays.

**Action:** Implement dialog overlay system with push/pop stack.

### Finding 1.6: Sidebar doesn't show real files
The file tree shows "No files opened yet" with no actual filesystem browsing.

**Action:** Build real file tree from current working directory.

### Finding 1.7: Session service never called
sessionSvc is stored but never used. Messages never persisted.

**Action:** Wire session creation on first message, save messages after each turn.

### Finding 1.8: Scroll doesn't work
scrollPos is tracked but never applied to rendering. Always shows last N lines.

**Action:** Implement proper scroll with scrollPos offset.

---

## 2. Agent Architecture Audit

### Finding 2.1: No subagent spawning
Single agent loop with no delegation capability. Codebuff spawns 15+ specialized agents.

**Action:** Implement spawn_agent tool as foundation for all subagent patterns.

### Finding 2.2: No context pruning between turns
Messages grow unboundedly. Codebuff has a context-pruner agent that runs between steps.

**Action:** Implement context manager with token budgeting and per-tool summarization.

### Finding 2.3: No propose-then-apply pattern
All edits are immediate. No way to preview or compare alternatives.

**Action:** Add propose_edit and propose_write tools.

---

## 3. Code Intelligence Audit

### Finding 3.1: No LSP integration
Crush auto-discovers LSP servers and provides diagnostics/references tools.

**Action:** Defer to Day 2 but design the interface now.

### Finding 3.2: No background jobs
bash.go runs synchronously. Crush has run_in_background, job_output, job_kill.

**Action:** Implement BackgroundShellManager.

---

## 4. Persistence Audit

### Finding 4.1: Session.Get returns (nil, nil)
Violates Go conventions. Should return error.

**Action:** Fix to return proper error.

### Finding 4.2: Pet not persisted
NewPet called every startup. Need Save/Load.

**Action:** Add pet persistence via JSON file.

### Finding 4.3: /config show is hardcoded
Returns static string instead of actual config.

**Action:** Read real config and format it.

---

## AUDIT VERDICT: PASS (with 15 actionable findings)

Priority order for tonight:
1. Fix session.Get error handling
2. Wire session persistence
3. Fix /config show
4. Add pet persistence
5. Extract UI components
6. Replace input with textarea.Model
7. Add completions popup
8. Add dialog overlay
9. Implement scroll
10. Build real file tree
11. Add message caching
12. Wire session creation
13. Fix remaining MEDIUM issues
