# FID-20260523-SAVANT-V2-OVERHAUL

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-SAVANT-V2-OVERHAUL                |
| **Date Created** | 2026-05-23                                      |
| **Status**       | OPEN                                            |
| **Priority**     | CRITICAL                                        |
| **Phase**        | Phase 1 - UI Overhaul & Core Fixes              |

## Context

The current TUI is non-functional and visually broken. Three parallel research agents analyzed Crush, Gemini CLI, and Codebuff codebases. Combined with the TUI+core deep audit (2 CRITICAL, 5 HIGH, 15 MEDIUM issues), this FID captures ALL improvements needed to make Savant CLI a real competitor.

## Issues Found (Perfection Loop Audit)

### CRITICAL (2 - already fixed)
- [x] CRITICAL-01: Agent events swallowed by empty callback → Fixed with channel-based pub/sub
- [x] CRITICAL-02: Conversation history lost between turns → Fixed with agentMessages preservation

### HIGH (5 - 3 fixed, 2 remaining)
- [x] HIGH-01/02/03: Panic on narrow terminals → Fixed with safeRepeat()
- [ ] HIGH-04: 9router sends empty model string → Config fixed but code still fragile
- [ ] HIGH-05: Zero test files exist

### MEDIUM (15 remaining)
- [ ] MEDIUM-01: Scroll position computed but never used in rendering
- [ ] MEDIUM-04: Session service imported but never used (messages never persisted)
- [ ] MEDIUM-06: ProviderBadgeLabel field unused (dead code)
- [ ] MEDIUM-08: Session.Get returns (nil, nil) instead of error
- [ ] MEDIUM-09: nil ToolCalls marshaled as "null" string in DB
- [ ] MEDIUM-10: /config show returns hardcoded values, not actual config
- [ ] MEDIUM-11: Pet state never persisted between sessions
- [ ] MEDIUM-12: pet.Tick() never called by TUI (now fixed in tick handler)
- [ ] MEDIUM-14: Regex compiled per-keyword per-call (now pre-compiled)
- [ ] MEDIUM-15: Tool results added sequentially (now concurrent via WaitGroup)

---

## Feature Roadmap (from Crush, Gemini CLI, Codebuff research)

### Tier 1: UI Overhaul (CRITICAL - immediate)

| # | Feature | Source | Effort | Status |
|---|---------|--------|--------|--------|
| F01 | **Fix TUI rendering** - proper layout, no crashes, responsive | Audit | Medium | TODO |
| F02 | **Animated cyberpunk title bar** with boot sequence | Crush | Easy | DONE |
| F03 | **Completions popup** for @file mentions | Crush | Medium | TODO |
| F04 | **Dialog overlay system** for modals (model select, permissions) | Crush | Medium | TODO |
| F05 | **Lazy-rendered chat list** with per-item caching | Crush | Medium | TODO |
| F06 | **Proper input editor** using textarea.Model from Bubbles | Crush | Easy | TODO |
| F07 | **Tab cycling sidebar** (Files/Sessions/Tasks/Pet) | Done | Easy | DONE |
| F08 | **Sidebar with file tree** using real filesystem | Crush | Medium | TODO |
| F09 | **Status bar** with provider, tokens, cost, pet status | Done | Easy | DONE |
| F10 | **Cyberpunk theme** with glow effects | Done | Easy | DONE |

### Tier 2: Agent Architecture (HIGH)

| # | Feature | Source | Effort | Status |
|---|---------|--------|--------|--------|
| F11 | **Spawn agents tool** - subagent primitive | Codebuff | High | TODO |
| F12 | **Specialized agents** (file-picker, code-reviewer, basher) | Codebuff | High | TODO |
| F13 | **Best-of-N implementation selection** | Codebuff | High | TODO |
| F14 | **Intelligent context pruning** with per-tool token budgets | Codebuff | Medium | TODO |
| F15 | **Declarative agent definitions** (YAML/Go structs) | Codebuff | Medium | TODO |
| F16 | **Mode-based variants** (fast/standard/thorough) | Codebuff | Medium | TODO |
| F17 | **Propose-then-apply editing** pattern | Codebuff | Low | TODO |
| F18 | **set_messages tool** for context manipulation | Codebuff | Low | TODO |
| F19 | **Perfection Loop FSM** integration with agent | Self | Medium | TODO |
| F20 | **Auto-compaction** at 80% context threshold | Gemini | Medium | TODO |

### Tier 3: Code Intelligence (HIGH)

| # | Feature | Source | Effort | Status |
|---|---------|--------|--------|--------|
| F21 | **LSP Client Manager** with auto-discovery | Crush | Hard | TODO |
| F22 | **diagnostics tool** (LSP-powered) | Crush | Medium | TODO |
| F23 | **references tool** (LSP-powered find-all-refs) | Crush | Medium | TODO |
| F24 | **Skills system** (SKILL.md standard) | Crush | Easy | TODO |
| F25 | **Background jobs** (run_in_background, job_output, job_kill) | Crush | Medium | TODO |
| F26 | **Sourcegraph integration** for code search | Crush/OpenCode | Medium | TODO |

### Tier 4: Policy & Safety (MEDIUM)

| # | Feature | Source | Effort | Status |
|---|---------|--------|--------|--------|
| F27 | **TOML policy engine** for tool approval | Gemini | Medium | TODO |
| F28 | **Tool execution scheduler** with state machine | Gemini | High | TODO |
| F29 | **Hooks system** (Claude Code compatible) | Crush | Medium | TODO |
| F30 | **Permission system** (pre/post tool hooks) | Done | Easy | DONE |

### Tier 5: Persistence & Sessions (MEDIUM)

| # | Feature | Source | Effort | Status |
|---|---------|--------|--------|--------|
| F31 | **Session persistence** (save/load conversations) | Crush | Medium | TODO |
| F32 | **File change tracking** with rollback | Crush | Medium | TODO |
| F33 | **Pet state persistence** across sessions | Self | Easy | TODO |
| F34 | **Config management** (/config reads/writes real config) | Self | Easy | TODO |

### Tier 6: Polish & DX (LOW)

| # | Feature | Source | Effort | Status |
|---|---------|--------|--------|--------|
| F35 | **Unit tests** for all packages | Audit | Medium | TODO |
| F36 | **@mention completions** for agents | Codebuff | Medium | TODO |
| F37 | **write_todos tool** for task tracking | Codebuff | Low | TODO |
| F38 | **MCP support** | All | High | TODO |
| F39 | **Voice input** | OpenClaude | High | TODO |
| F40 | **gRPC headless mode** | OpenClaude | High | TODO |

---

## Execution Plan

### Night 1 (Tonight - automated):
1. Fix all MEDIUM TUI issues (scroll, dead code, session wiring)
2. Build proper input editor using Bubbles textarea
3. Build completions popup for @file mentions
4. Build dialog overlay system
5. Wire session persistence properly
6. Fix pet persistence
7. Build lazy-rendered chat list
8. Build real file tree in sidebar
9. Fix all remaining audit findings

### Day 2:
1. Spawn agents tool (F11)
2. Specialized agents (F12)
3. Intelligent context pruning (F14)
4. Background jobs (F25)
5. Skills system (F24)

### Day 3+:
1. LSP integration (F21-F23)
2. Best-of-N (F13)
3. Policy engine (F27)
4. Tests (F35)

---

## Verification Criteria

- [ ] TUI renders correctly at any terminal size (including narrow)
- [ ] Agent streaming works in real-time
- [ ] Conversation history preserved across turns
- [ ] Session persistence works (save/load)
- [ ] Pet evolves over time
- [ ] /config shows real config values
- [ ] All slash commands functional
- [ ] Provider routing works with fallback
- [ ] Context pruning prevents API errors
- [ ] No panics on any input
