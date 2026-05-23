---
name: karpathy-guidelines
description: Behavioral guidelines to reduce common LLM coding mistakes. Use when writing, reviewing, or refactoring code.
license: MIT
compatibility: savant-cli>=0.1.0
---

# Karpathy Guidelines

Behavioral guidelines to reduce common LLM coding mistakes, derived from Andrej Karpathy's observations on LLM coding pitfalls.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform imperative tasks into verifiable goals:
- "Add validation" -> "Write tests for invalid inputs, then make them pass"
- "Fix the bug" -> "Write a test that reproduces it, then make it pass"
- "Refactor X" -> "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]

Strong success criteria let the agent loop independently. Weak criteria ("make it work") require constant clarification.

## Common Anti-Patterns

### Over-Abstraction
WRONG: Strategy pattern, abstract base class, and factory for a single discount calculation.
RIGHT: `func calculateDiscount(amount, percent float64) float64 { return amount * percent / 100 }`

### Drive-By Refactoring
WRONG: User asks to fix empty email validation. Agent also adds type hints, changes quote style, adds docstrings, and refactors username validation.
RIGHT: Only change the lines that handle empty emails. Leave everything else untouched.

### Vague Goals
WRONG: "I'll review and improve the code"
RIGHT: "Write test for bug X -> make it pass -> verify no regressions"

### Hidden Assumptions
WRONG: User says "add export feature." Agent assumes JSON format, all fields, file output, no pagination.
RIGHT: "Before implementing, I need to clarify: export format? which fields? pagination?"

## Key Insight

> Good code is code that solves today's problem simply, not tomorrow's problem prematurely.
