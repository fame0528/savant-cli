# DEEP AUDIT - Tool Forge FID

**Date:** 2026-05-24
**FID:** FID-20260524-TOOL-FORGE
**Auditor:** Perfection Loop (5 phases)
**Source:** Savant Rust ToolForge (6 files, 559 lines forge_tool.rs + 235 lines quality.rs + 154 lines provenance.rs + 88 lines registry.rs + 65 lines curator.rs), Hermes skill_manager_tool.py (931 lines)

---

## Phase 1: Deep Audit (10 findings)

### Finding 1.1: FID stores tools at wrong path - LOW
The FID says `~/.savant/tools/forge/` but our existing skill system uses `~/.savant/skills/`. Savant Rust uses `skills/forge/`. We should keep forge tools under `~/.savant/skills/forge/` for consistency with the Agent Skills standard.

**Resolution:** Store at `~/.savant/skills/forge/`. Forged tools are discoverable by the existing skills system AND have a dedicated namespace.

### Finding 1.2: "Forged tools are procedural knowledge, not code" - CORRECT
The FID says forged tools are SKILL.md files, not executable code. When the model calls a forged tool, the agent reads the SKILL.md and follows its instructions. This matches how skills already work. The FID is correct.

### Finding 1.3: Quality Gate needs Go-specific stub patterns - MEDIUM
Savant Rust detects `TODO!()`, `unimplemented!()`. Go equivalents are `// TODO`, `// FIXME`, `panic("not implemented")`, `// placeholder`. Need to adapt the regex.

**Resolution:** Pre-compile regex at init(): `(?i)(//\s*todo|//\s*fixme|panic\("not implemented"\)|placeholder|\[STUB\]|__STUB__|TBD)`.

### Finding 1.4: Provenance needs atomic appends - MEDIUM
Multiple agents writing to the same JSONL file concurrently could corrupt it. Savant Rust uses `spawn_blocking` for async writes.

**Resolution:** Use `sync.Mutex` to serialize writes. File opened in append mode, small writes (< 4096 bytes) are atomic at OS level.

### Finding 1.5: Curator needs provenance replay - MEDIUM
For MVP, replay is acceptable. JSONL file grows slowly. The curator replays all entries, filters by tool name, finds max timestamp, archives if older than threshold.

**Resolution:** `ProvenanceTracker.Replay()` returns all entries. Curator filters and archives.

### Finding 1.6: Version management needs semver parsing - LOW
Use `strings.Split` + `strconv.Atoi`. No semver library needed.

**Resolution:** Parse X.Y.Z, increment appropriate part.

### Finding 1.7: Forged tools need to be registered in the tool registry - HIGH
Create `ForgedTool` struct that implements `Tool` interface. Its `Execute()` reads the SKILL.md and returns content. The agent interprets instructions using built-in tools.

**Resolution:** `ForgedTool.Execute()` reads SKILL.md, returns the content as instructions. The agent then uses its existing tools to follow them.

### Finding 1.8: Quality gate keyword overlap needs word-boundary matching - MEDIUM
"web-scraper" and "web-fetcher" would have high overlap even though they're different tools. Need word-level Jaccard similarity with stop words excluded.

**Resolution:** Jaccard similarity on words > 2 chars, excluding stop words. Threshold 0.8.

### Finding 1.9: Forge directory needs creation on startup - LOW
`os.MkdirAll` on startup in main.go.

**Resolution:** Create `~/.savant/skills/forge/` in main.go initialization.

### Finding 1.10: Shared registry needs dynamic addition - HIGH
Our current `tools.Registry` doesn't support dynamic addition after creation. Forged tools need to be registered at runtime.

**Resolution:** Add `RegisterDynamic(name, tool)` method to `tools.Registry`. Broadcast event so TUI can update.

---

## Phase 2: Heuristic Enhancement (6 enhancements)

1. Store at `~/.savant/skills/forge/` (consistent with skills infrastructure)
2. ForgedTool implements Tool interface (reads SKILL.md, returns content)
3. Quality gate uses pre-compiled regex at init()
4. Provenance uses sync.Mutex for thread-safe appends
5. Curator wired alongside existing skills curator in main.go
6. Version storage as flat files: `~/.savant/skills/forge/name/versions/0.1.0.md`

---

## Phase 3: Validation Strike (6 checks)

| Check | Result |
|-------|--------|
| Forge tool approach correct | PASS (procedural knowledge, not code) |
| Quality gate comprehensive | PASS (9 checks prevent garbage) |
| Provenance tracker necessary | YES (rollback, stats, curator input) |
| Curator necessary | YES (prevents stale tool accumulation) |
| Scope achievable | PASS (5 new files, ~800 lines) |
| Missing features acceptable | YES (all items included, zero deferrals) |

---

## Phase 4: Iterative Convergence

10 remaining issues, all targeted changes. No architectural issues:
1. Store at ~/.savant/skills/forge/
2. ForgedTool implements Tool interface
3. Quality gate Go stub regex
4. Provenance sync.Mutex
5. Curator provenance replay
6. Version semver parsing
7. Dynamic registry addition
8. Quality gate Jaccard similarity
9. Forge directory creation
10. Version storage as flat files

---

## Phase 5: Final Certification

### CERTIFICATION: PASS

All 5 phases executed. 10 audit findings, 6 enhancements, 6 validation checks. Zero deferrals. 5 new files, ~800 lines. Quality gate prevents garbage. Provenance enables rollback. Curator keeps collection clean. Version storage enables restore.
