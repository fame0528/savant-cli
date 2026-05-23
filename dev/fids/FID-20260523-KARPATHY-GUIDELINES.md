# FID-20260523-KARPATHY-GUIDELINES

| Field            | Value                                          |
|------------------|-------------------------------------------------|
| **Document ID**  | FID-20260523-KARPATHY-GUIDELINES                |
| **Date Created** | 2026-05-23                                      |
| **Status**       | OPEN (awaiting approval)                        |
| **Priority**     | HIGH                                            |

## Context

Andrej Karpathy's four principles for reducing LLM coding mistakes, packaged as an Agent Skill at `resoruces/andrej-karpathy-skills/`. The repo contains:
- `CLAUDE.md` — The four principles (66 lines)
- `EXAMPLES.md` — 8 concrete wrong/right comparisons (523 lines)
- `skills/karpathy-guidelines/SKILL.md` — Agent Skills format (68 lines)

These principles directly address the most common agent failures that our system prompt partially covers. The examples are particularly valuable — concrete wrong/right comparisons that calibrate the model's behavior.

## Gap Analysis: Current System Prompt vs Karpathy Principles

| Karpathy Principle | Our Current Coverage | Gap |
|---|---|---|
| **Think Before Coding** (surface assumptions, ask when uncertain) | critical_rule 13: "THINK BEFORE ACTING" — reads code first, understands architecture | MISSING: explicit assumption surfacing, "ask when uncertain", presenting multiple interpretations |
| **Simplicity First** (minimum code, no speculative features) | critical_rule 16: "SIMPLICITY AND MINIMALISM" — make few changes, only do what was asked | PARTIAL: missing "would a senior engineer say this is overcomplicated?" test, missing concrete examples |
| **Surgical Changes** (touch only what you must) | critical_rule 16: same rule covers both simplicity and surgical changes | MISSING: "don't improve adjacent code", "match existing style", "mention dead code but don't delete it" |
| **Goal-Driven Execution** (define success criteria, loop until verified) | workflow section: "Plan → Act → Validate" cycle | MISSING: concrete transformation table (vague task → verifiable goals), test-first verification pattern |

## What to Build

### 1. Add `karpathy-guidelines` as a Builtin Skill
Copy the SKILL.md from `resoruces/andrej-karpathy-skills/skills/karpathy-guidelines/` into `internal/skills/builtin/karpathy-guidelines/SKILL.md`. This makes it discoverable and loadable via the Agent Skills standard.

### 2. Integrate Principles into System Prompt Template
Update `internal/agent/templates/system.md` to incorporate the Karpathy principles:

**Add to `<communication_style>`:**
- "State assumptions explicitly. If uncertain, ask."
- "If multiple interpretations exist, present them — don't pick silently."
- "If a simpler approach exists, say so. Push back when warranted."

**Add to `<task_completion>`:**
- "Transform vague tasks into verifiable goals"
- Concrete transformation table: "Add validation" → "Write tests for invalid inputs, then make them pass"

**Add to `<code_conventions>`:**
- "Don't improve adjacent code, comments, or formatting"
- "Don't refactor things that aren't broken"
- "If you notice unrelated dead code, mention it — don't delete it"
- "Remove imports/variables/functions that YOUR changes made unused"
- "Don't remove pre-existing dead code unless asked"

**Add to `<response_examples>`:**
- Add the "What LLMs Do Wrong" vs "What Should Happen" examples from EXAMPLES.md
- Focus on the most impactful: over-abstraction, drive-by refactoring, vague goals

### 3. Update Instructions Prompt
Add Karpathy principles to the per-message reminder:
- "State assumptions explicitly before implementing"
- "Surgical changes only — every changed line traces to the user's request"

## Files to Change

| File | Change |
|------|--------|
| `internal/skills/builtin/karpathy-guidelines/SKILL.md` | NEW: Builtin skill from the repo |
| `internal/agent/templates/system.md` | Update: Add Karpathy principles to communication_style, task_completion, code_conventions, response_examples |
| `internal/agent/templates/instructions.md` | Update: Add Karpathy reminders to per-message prompt |

## Verification Criteria

- [ ] `karpathy-guidelines` skill discovered by `skills.DiscoverBuiltin()`
- [ ] System prompt contains all 4 Karpathy principles
- [ ] System prompt contains concrete wrong/right examples from EXAMPLES.md
- [ ] Instructions prompt includes Karpathy reminders
- [ ] Agent states assumptions before implementing
- [ ] Agent doesn't do drive-by refactoring
- [ ] Agent transforms vague tasks into verifiable goals
- [ ] No stubs or placeholders
