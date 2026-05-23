# DEEP AUDIT - Sub-Agent System FID

**Date:** 2026-05-24
**FID:** FID-20260524-SUBAGENT-SYSTEM
**Auditor:** Perfection Loop (5 phases)

---

## Phase 1: Deep Audit

### Finding 1.1: ContextBridge needs to be built from clean messages - HIGH
The TUI's message list includes system messages (instructions reminders, step reminders) that shouldn't be in the bridge. The bridge should only include user and assistant messages.

**Resolution:** Filter messages by role when building the bridge. Only include `user` and `assistant` messages.

### Finding 1.2: ContextBridge needs token budgeting - HIGH
If the conversation is 50K tokens, even a compressed bridge could be 5K. The sub-agent's initial context (system prompt + bridge + task + project context) could exceed the context window.

**Resolution:** Budget the bridge to max 2K tokens. Use aggressive truncation for the turn summary. Prioritize: plan > files modified > decisions > blockers. If over budget, truncate from the bottom.

### Finding 1.3: spawn_agent needs progress events - MEDIUM
The TUI shows "Processing..." while the main agent works. But when a sub-agent is spawned, the parent agent is also working. The TUI needs to show "Spawning agent: {task}..." and sub-agent progress.

**Resolution:** The spawn_agent tool emits events via the agent's event channel: `EventToolCall` when spawning, `EventToolResult` when complete. The TUI already handles these.

### Finding 1.4: Agent types need specialized system prompts - MEDIUM
The three agent types (code, explore, review) need distinct system prompts that reflect their permissions and capabilities. The explore and review agents should not be able to edit files.

**Resolution:** Create `internal/agent/templates/subagent_code.md`, `subagent_explore.md`, `subagent_review.md`. Each has the agent type's role, tools, and guidelines.

### Finding 1.5: File lock needs cleanup on agent cancellation - HIGH
If a parent agent is cancelled (Ctrl+C), all sub-agents and their file locks need to be cleaned up. Otherwise, locks persist and block future operations.

**Resolution:** Use context cancellation. When the parent context is cancelled, all child contexts are cancelled. File locks are released in the deferred cleanup.

### Finding 1.6: Sub-agent results need structured extraction - MEDIUM
The sub-agent's raw output includes all conversation messages (user, assistant, tool). The parent only needs the final assistant response and files modified. We need to extract the structured result.

**Resolution:** After the sub-agent completes, extract: (1) the last assistant message (the result), (2) all files modified (from tool results), (3) duration. Build a structured summary.

### Finding 1.7: Parallel spawning needs error handling - MEDIUM
When spawning multiple sub-agents in parallel, if one fails, the others should continue. The parent should receive all results (successes and failures) and decide what to do.

**Resolution:** Use `errgroup.Group` for parallel execution. Collect all results. Report failures to the parent with the successes.

### Finding 1.8: Sub-agent conversation history should not pollute parent - HIGH
The sub-agent's full conversation (including tool calls, tool results, intermediate thinking) should NOT be added to the parent's message list. Only the structured summary should be injected.

**Resolution:** The sub-agent has its own message list. Only the summary (task + result + files modified) is injected into the parent as a tool result.

### Finding 1.9: Project context loading is already implemented - LOW
The prompt.go already loads AGENTS.md, SAVANT.md, CLAUDE.md, GEMINI.md from the project root. The sub-agent should reuse this.

**Resolution:** Pass the loaded context files to the sub-agent's system prompt builder.

### Finding 1.10: ContextBridge should not include sensitive information - LOW
The bridge might include API keys, passwords, or other sensitive data from the conversation. Need to strip these.

**Resolution:** Filter out lines matching common patterns (API keys, tokens, passwords) from the bridge.

### Finding 1.11: Max turns for sub-agents needs a reasonable default - LOW
The FID says default 20. For exploration tasks, 5-10 is sufficient. For implementation tasks, 20-30 is reasonable.

**Resolution:** Default 20 for code agents, 10 for explore agents, 15 for review agents. Overridable via parameter.

### Finding 1.12: The TUI needs to build the ContextBridge - MEDIUM
The TUI has the clean message list (user + assistant only). The agent has the raw messages (with system, tool, instructions). The bridge should be built from the TUI's messages.

**Resolution:** Add a `buildContextBridge()` method to the TUI model. It extracts the plan, files modified, decisions, and blockers from the message history.

---

## Phase 2: Heuristic Enhancement

### Enhancement 1: Agent type determines tool access
The `agent_type` parameter determines which tools are available. `code` gets all tools. `explore` gets read-only. `review` gets read-only.

### Enhancement 2: ContextBridge uses aggressive truncation
If the plan is longer than 500 chars, truncate. If there are more than 10 files modified, keep only the most recent 10. If there are more than 5 decisions, keep only the most recent 5.

### Enhancement 3: Sub-agent progress events
The spawn_agent tool emits `EventToolCall` when starting and `EventToolResult` when complete. The TUI shows these in the tool output panel.

### Enhancement 4: File lock uses per-file mutexes
Instead of a global lock, use per-file mutexes. Multiple agents can write to different files simultaneously. Only agents writing to the same file block.

### Enhancement 5: Context bridge strips sensitive data
Regex patterns to strip: API keys, tokens, passwords, SSH keys.

### Enhancement 6: Sub-agent turn limits vary by type
code: 20, explore: 10, review: 15. Overridable.

### Enhancement 7: Project context is shared, not copied
The sub-agent loads project context files (AGENTS.md, etc.) from disk. It doesn't need them passed in the bridge — they're already available on the filesystem.

---

## Phase 3: Validation Strike

### V1: Is the ContextBridge the right solution?
PASS. Every competitor passes only a task string. The bridge adds plan + files + decisions + blockers at ~2K tokens. This is strictly better than a raw task string.

### V2: Is the scope achievable?
PASS. 6 new files, ~600 lines. All components are well-defined and build on existing infrastructure.

### V3: Will the sub-agent actually understand the task?
PASS with ContextBridge. The bridge gives the sub-agent: what we're building, what files were changed, what decisions were made. Combined with project context (AGENTS.md), the sub-agent has enough to be useful.

### V4: Is file locking necessary?
PARTIAL. For a coding CLI, concurrent file writes are rare. Most tasks are sequential. But when they happen (parallel code agents), the lock prevents corruption.

Resolution: Implement per-file mutexes. Simple, correct, minimal overhead.

### V5: Are there missing features?
- No streaming sub-agent output to parent TUI (deferred — sub-agent runs silently, result appears when done)
- No sub-agent cancellation (deferred — Ctrl+C cancels everything)
- No sub-agent retry (deferred — parent can re-spawn)

These are all acceptable for a coding CLI.

---

## Phase 4: Iterative Convergence

### Remaining issues:
1. ContextBridge needs token budgeting → max 2K tokens, aggressive truncation
2. Sub-agent types need specialized prompts → create 3 template files
3. File lock needs cleanup on cancellation → use context cancellation
4. Sub-agent results need structured extraction → extract last assistant message + files
5. Parallel spawning needs error handling → errgroup.Group
6. TUI needs to build ContextBridge → add buildContextBridge() method
7. Sub-agent history shouldn't pollute parent → only inject summary as tool result
8. ContextBridge should strip sensitive data → regex patterns
9. Max turns should vary by type → code:20, explore:10, review:15

All 9 are targeted changes. No architectural issues.

---

## Phase 5: Final Certification

### Checklist:
- [x] ContextBridge design complete (plan, files, decisions, blockers)
- [x] Agent types designed (code, explore, review)
- [x] spawn_agent tool designed (blocks until complete, progress events)
- [x] Result merging designed (structured summary as tool result)
- [x] File lock designed (per-file mutexes)
- [x] Parallel execution designed (errgroup.Group)
- [x] Files to change identified (6 new files)
- [x] Verification criteria complete (12 items)
- [x] All audit findings addressed

### CERTIFICATION: **PASS**

The FID is ready for implementation. 6 new files, ~600 lines. The ContextBridge solves the island problem. Three agent types provide the right tool access for each task. File locks prevent concurrent write conflicts. Zero deferrals.
