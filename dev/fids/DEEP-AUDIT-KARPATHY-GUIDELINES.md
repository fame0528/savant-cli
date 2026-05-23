# DEEP AUDIT - Karpathy Guidelines FID

**Date:** 2026-05-23
**FID:** FID-20260523-KARPATHY-GUIDELINES
**Auditor:** Perfection Loop (5 phases)

---

## Phase 1: Deep Audit

### Finding 1.1: Current system prompt already has partial coverage - LOW
Our system prompt already has critical_rule 13 ("THINK BEFORE ACTING") and critical_rule 16 ("SIMPLICITY AND MINIMALISM"). The Karpathy principles overlap significantly. The risk is duplicating rules and making the prompt longer without adding value.

**Resolution:** Don't duplicate. Instead, enhance existing rules with Karpathy's specific language and examples. The transformation table ("Add validation" → "Write tests") and the "surgical changes" guidance are the truly new material.

### Finding 1.2: Examples are the highest-value addition - HIGH
The EXAMPLES.md file has 8 concrete wrong/right comparisons. These are far more effective at calibrating model behavior than abstract rules. Our current response_examples section has 5 examples but they're all "correct" examples — no "wrong" examples showing what NOT to do.

**Resolution:** Add the most impactful wrong/right pairs from EXAMPLES.md to the response_examples section. Focus on: over-abstraction (Example 2.1), drive-by refactoring (Example 3.1), and vague goals (Example 4.1).

### Finding 1.3: Builtin skill needs adaptation, not copy - MEDIUM
The SKILL.md from the repo uses generic examples (Python, CSV exports). Our builtin skill should use Go examples since Savant is a Go project.

**Resolution:** Adapt the skill content to use Go examples and reference Savant-specific patterns.

### Finding 1.4: Instructions prompt is already at capacity - MEDIUM
The per-message instructions reminder already has 8 bullet points. Adding more dilutes effectiveness. Karpathy's "State assumptions explicitly" is important enough to include, but "surgical changes" is already covered by existing rules.

**Resolution:** Add only "State assumptions explicitly before implementing" to the instructions prompt. The rest goes in the system prompt.

### Finding 1.5: The "Goal-Driven Execution" transformation table is the most unique addition - HIGH
Our current workflow says "Plan → Act → Validate" but doesn't show how to transform vague tasks into verifiable goals. Karpathy's table is the most actionable guidance:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

**Resolution:** Add this transformation table to the task_completion section of the system prompt.

### Finding 1.6: The "Would a senior engineer say this is overcomplicated?" test is valuable - LOW
This is a great self-check heuristic. Our current rule says "make as few changes as possible" but doesn't provide a mental model for judging complexity.

**Resolution:** Add this as a check in the code_conventions section.

### Finding 1.7: "Don't improve adjacent code" should be its own rule - MEDIUM
Currently merged into critical_rule 16 ("SIMPLICITY AND MINIMALISM"). But Karpathy's EXAMPLES.md shows this is the #1 source of diff noise. It deserves its own emphasis.

**Resolution:** Split into a separate bullet under code_conventions rather than merging into critical_rule 16. Keep critical_rule 16 focused on simplicity.

### Finding 1.8: The skill should reference the full EXAMPLES.md - LOW
The builtin skill only has the principles, not the examples. The examples are what make the principles concrete.

**Resolution:** Add a reference to EXAMPLES.md in the skill's references/ directory. But since we're embedding in the binary, include the key examples directly in the SKILL.md.

---

## Phase 2: Heuristic Enhancement

### Enhancement 1: Enhance, don't duplicate
Don't add new critical rules. Enhance existing rules 13, 16, and the task_completion section with Karpathy's specific language.

### Enhancement 2: Add wrong/right examples to response_examples
The most impactful additions are the "What LLMs Do Wrong" vs "What Should Happen" pairs. Add 3 key examples.

### Enhancement 3: Add transformation table to task_completion
The "vague → verifiable" transformation is the most unique Karpathy contribution.

### Enhancement 4: Add "senior engineer" test to code_conventions
Simple self-check heuristic for complexity.

### Enhancement 5: Split "don't improve adjacent code" into its own emphasis
This is the #1 source of diff noise and deserves separate treatment.

### Enhancement 6: Keep instructions prompt minimal
Only add "State assumptions explicitly before implementing" — the rest goes in the system prompt.

---

## Phase 3: Validation Strike

### V1: Will this actually change agent behavior?
PASS — The examples are the key. Concrete wrong/right comparisons are proven to calibrate LLM behavior better than abstract rules. The transformation table gives the agent a concrete pattern for task decomposition.

### V2: Is the system prompt getting too long?
POSSIBLE — The system prompt is already ~16K chars. Adding Karpathy examples will add ~2K more. This is acceptable given the behavioral improvement, but we should monitor.

### V3: Does the builtin skill add value beyond the system prompt?
YES — The skill provides deeper guidance with full examples that the system prompt can reference. The system prompt gives the rules; the skill gives the detailed playbook.

### V4: Are the examples Go-specific enough?
PARTIAL — The original examples use Python. For the builtin skill, we should adapt to Go. For the system prompt, the principles are language-agnostic so Python examples work fine.

### V5: Is there overlap with existing critical rules?
YES — But that's intentional. Karpathy's language is more specific and actionable. Enhancing existing rules is better than adding new ones.

### V6: Is the scope achievable?
PASS — 3 files to change, ~200 lines of additions. No new dependencies.

---

## Phase 4: Iterative Convergence

### Remaining issues:
1. System prompt length increase — acceptable, monitor
2. Examples not Go-specific in system prompt — acceptable, language-agnostic
3. Builtin skill needs Go examples — adapt from Python originals
4. Instructions prompt should stay minimal — only add 1 line
5. "Don't improve adjacent code" needs separate emphasis — add to code_conventions

All 5 are targeted changes. No architectural issues.

---

## Phase 5: Final Certification

### Checklist:
- [x] Gap analysis complete (4 principles mapped to existing rules)
- [x] Enhancement strategy clear (enhance, don't duplicate)
- [x] Examples identified (3 most impactful from EXAMPLES.md)
- [x] Transformation table designed
- [x] Files to change identified (3 files)
- [x] Verification criteria complete (8 items)
- [x] All audit findings addressed

### CERTIFICATION: PASS
The FID is ready for implementation. 3 files, ~200 lines. Enhances existing rules with Karpathy's specific language and concrete examples. Highest-value addition is the wrong/right examples and the transformation table.
