# FID-20260523-AUTO-SKILL-GENERATION

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-AUTO-SKILL-GENERATION              |
| **Date Created** | 2026-05-23                                      |
| **Status**       | OPEN (awaiting approval)                        |
| **Priority**     | HIGH                                            |

## Context

The agent currently has a skill discovery system (Agent Skills standard) but no way to create skills from experience. Three competitors have auto-skill generation:

- **Hermes**: `skill_manage` tool (agent creates inline) + Curator (background lifecycle management)
- **Gemini CLI**: Background `SkillExtractionAgent` that analyzes past sessions
- **Codebuff**: Inline "Phase 7 — Lessons" at task end with thinker critique

Savant should combine all three approaches into a superior system. The agent should be able to:
1. Create skills inline during a session when it discovers a reusable procedure
2. Extract skills from completed session transcripts in the background
3. Track skill usage and auto-manage lifecycle (active → stale → archived)

## What Each Competitor Does Well

| Competitor | Pattern | Savant Will Include |
|------------|---------|---------------------|
| **Hermes** | `skill_manage` tool with CRUD actions | YES — create, edit, patch, delete, write_file, remove_file |
| **Hermes** | Provenance tracking (`created_by: "agent"`) | YES — distinguish agent-created from builtin/user |
| **Hermes** | Curator background lifecycle (active → stale → archived) | YES — with configurable thresholds |
| **Hermes** | Usage telemetry sidecar (.usage.json) | YES — use_count, view_count, last_activity_at |
| **Hermes** | Pinned skills exempt from auto-transitions | YES |
| **Hermes** | Never deletes, only archives | YES |
| **Hermes** | LLM consolidation pass (merge narrow → umbrella) | YES |
| **Gemini CLI** | Background SkillExtractionAgent | YES — analyze past sessions |
| **Gemini CLI** | "No-op by default" — 0-2 skills per run | YES — quality over quantity |
| **Gemini CLI** | .inbox/ for user review before applying | YES — safety gate |
| **Gemini CLI** | Session eligibility filtering (idle 3hr, 10+ msgs) | YES |
| **Gemini CLI** | Lock file coordination for multi-instance | YES |
| **Codebuff** | Inline "Phase 7 — Lessons" at task end | YES — capture learnings immediately |
| **Codebuff** | Thinker critique loop (improve until no changes) | YES |
| **Codebuff** | Skills must NOT include run-specific details | YES — only broadly applicable knowledge |
| **Codebuff** | Category-based organization | YES |

## Architecture: 3-Component System

### Component 1: `skill_manage` Tool (Hermes pattern)

An agent-callable tool for inline skill CRUD during sessions.

**Actions:**
- `create` — Create a new skill (SKILL.md + directory)
- `edit` — Full rewrite of SKILL.md
- `patch` — Targeted find-and-replace within SKILL.md
- `delete` — Remove a user skill (blocked if pinned)
- `write_file` — Add supporting file (reference, template, script)
- `remove_file` — Remove supporting file

**Validation:**
- Name: lowercase alphanumeric with hyphens, max 64 chars
- Description: required, max 1024 chars, one sentence
- Content: max 100KB
- Supporting files: max 1MB each
- Allowed subdirectories: references/, templates/, scripts/, assets/
- Security scan on create (configurable)

**Storage:** `~/.savant/skills/<name>/SKILL.md`

**Provenance:** All agent-created skills marked with `created_by: "agent"` in `.usage.json`

### Component 2: Curator (Hermes pattern)

Background skill lifecycle management.

**Triggers:**
- Runs at session start if idle for 2+ hours and last run was 7+ days ago
- Deferred first run by one full interval (prevents immediate runs on fresh installs)

**Automatic transitions (pure time-based, no LLM):**
- `active` → `stale` after 30 days of no activity
- `stale` → `archived` after 90 days of no activity
- Pinned skills are exempt from all transitions

**LLM consolidation pass:**
- Spawns a sub-agent with the CURATOR_REVIEW_PROMPT
- Scans agent-created skills for "prefix clusters" (shared domain keywords)
- Identifies umbrella-class opportunities
- Merges narrow skills into broader ones
- Moves narrow content to references/ or templates/
- Archives absorbed skills (never deletes)

**Output:** Per-run report in `~/.savant/logs/curator/<timestamp>/REPORT.md`

### Component 3: Session Extraction (Gemini pattern)

Background analysis of completed sessions for reusable patterns.

**Triggers:**
- Runs at session startup
- Gated by: 30-minute throttle, lock file, eligible sessions exist

**Eligibility:**
- Session idle for 3+ hours
- 10+ user messages
- Not a sub-agent session
- Not previously processed

**Process:**
1. Scan eligible sessions
2. Build session index with summaries
3. Run extraction agent (up to 30 turns, 10 minutes)
4. Agent reads transcripts looking for:
   - Repeated cross-session workflows
   - Validated procedures (tried, failed, succeeded)
   - User corrections that reveal domain knowledge
   - Failure-to-success arcs
5. Write extracted skills to `~/.savant/skills/`
6. Write memory patches to `~/.savant/skills/.inbox/` for user review
7. Record extraction state in `.extraction-state.json`

**Quality bar:** "Default to NO skill. Aim for 0-2 skills per run."

## Usage Telemetry

Sidecar file: `~/.savant/skills/.usage.json`

```json
{
  "go-rest-api": {
    "use_count": 12,
    "view_count": 15,
    "patch_count": 3,
    "last_used_at": "2026-05-23T14:30:00Z",
    "last_viewed_at": "2026-05-23T14:30:00Z",
    "last_patched_at": "2026-05-22T10:00:00Z",
    "state": "active",
    "pinned": false,
    "created_by": "agent"
  }
}
```

**State transitions:**
- `active` → `stale` after `stale_after_days` (default 30) of no activity
- `stale` → `archived` after `archive_after_days` (default 90) of no activity
- `pinned` — exempt from all auto-transitions

**Counter bumps:**
- `view_count` — when skill is loaded into context
- `use_count` — when skill is actively used during task execution
- `patch_count` — when skill is modified

## Skill Directory Structure

```
~/.savant/skills/
├── .usage.json              # Usage telemetry sidecar
├── .curator_state           # Curator scheduler state
├── .extraction-state.json   # Session extraction state
├── .inbox/                  # Pending memory patches for user review
│   └── extraction.patch
├── .archive/                # Archived skills (recoverable)
│   └── old-skill/
│       └── SKILL.md
├── go-rest-api/             # Active skill
│   ├── SKILL.md
│   ├── references/
│   ├── templates/
│   └── scripts/
└── docker-debugging/
    └── SKILL.md
```

## Config

```json
{
  "skills": {
    "guard_agent_created": false,
    "curator_enabled": true,
    "curator_interval_hours": 168,
    "curator_min_idle_hours": 2,
    "curator_stale_after_days": 30,
    "curator_archive_after_days": 90,
    "extraction_enabled": true,
    "extraction_throttle_minutes": 30
  }
}
```

## Files to Change

| File | Change |
|------|--------|
| `internal/tools/skill_manage.go` | NEW: skill_manage tool (CRUD actions, validation, security scan) |
| `internal/skills/usage.go` | NEW: usage telemetry sidecar (.usage.json) |
| `internal/skills/curator.go` | NEW: background lifecycle manager (auto-transitions + LLM consolidation) |
| `internal/skills/extraction.go` | NEW: session extraction agent (analyze transcripts, write skills, .inbox/) |
| `internal/skills/skills.go` | Update: add provenance tracking, pin support |
| `internal/skills/builtin/skill-manage/SKILL.md` | NEW: builtin skill documenting the skill management workflow |
| `internal/agent/templates/extraction.md` | NEW: extraction agent system prompt |
| `internal/agent/templates/curator.md` | NEW: curator consolidation prompt |
| `internal/agent/agent.go` | Update: wire skill_manage tool into tool registry |
| `internal/config/config.go` | Update: add skills config section |

## Verification Criteria

- [ ] `skill_manage(action="create")` creates a valid skill in ~/.savant/skills/
- [ ] `skill_manage(action="edit")` rewrites SKILL.md
- [ ] `skill_manage(action="patch")` applies find-and-replace
- [ ] `skill_manage(action="delete")` removes unpinned skills
- [ ] `skill_manage(action="delete")` refuses pinned skills
- [ ] `skill_manage(action="write_file")` adds supporting files
- [ ] Validation: name must be lowercase alphanumeric with hyphens
- [ ] Validation: description required, max 1024 chars
- [ ] Validation: content max 100KB
- [ ] Provenance: agent-created skills marked in .usage.json
- [ ] Usage tracking: use_count, view_count, patch_count updated
- [ ] Curator: active → stale after 30 days
- [ ] Curator: stale → archived after 90 days
- [ ] Curator: pinned skills exempt from transitions
- [ ] Curator: never deletes, only archives
- [ ] Extraction: scans eligible sessions (idle 3hr, 10+ msgs)
- [ ] Extraction: 0-2 skills per run (quality over quantity)
- [ ] Extraction: writes to .inbox/ for user review
- [ ] Lock file coordination for multi-instance safety
- [ ] Agent can create skills inline during session
- [ ] Skills discovered and injected in system prompt
- [ ] No stubs or placeholders
