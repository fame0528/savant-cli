---
name: skill-manage
description: Guide for creating, editing, and managing skills as reusable procedural knowledge.
license: MIT
compatibility: savant-cli>=0.1.0
---

# Skill Management Guide

## What Are Skills?

Skills are the agent's procedural memory. They capture HOW to do a specific type of task based on proven experience. General memory (MEMORY.md, USER.md) is broad and declarative. Skills are narrow and actionable.

## When to Create a Skill

Create a skill when:
- You've successfully completed a task that will recur (e.g., setting up a Go project, debugging a specific framework)
- You've discovered a validated procedure through trial and error
- The user has corrected you and the correction applies broadly
- You notice a pattern across multiple sessions

Do NOT create a skill for:
- One-off tasks that won't recur
- Generic knowledge the model already knows
- Run-specific details (file paths, timestamps, user names)
- Things that are better suited to project documentation

## Creating a Skill

Use the `skill_manage` tool:

```
skill_manage(
  action: "create",
  name: "go-rest-api",
  description: "Set up a Go REST API with chi router, middleware, and error handling.",
  content: "## When to Use\n\nWhen the user asks to create a REST API in Go.\n\n## Prerequisites\n\n- Go 1.21+ installed\n- chi router available (`go get github.com/go-chi/chi/v5`)\n\n## Procedure\n\n1. Create `main.go` with chi router setup\n2. Add middleware (logging, CORS, recovery)\n3. Create route handlers in `handlers/` directory\n4. Add error handling with structured JSON responses\n5. Add graceful shutdown\n\n## Pitfalls\n\n- Don't forget to set `Content-Type: application/json` in responses\n- Always use `context.Context` for timeouts\n- Return proper HTTP status codes (not always 200)"
)
```

## Skill Naming Rules

- Lowercase alphanumeric with hyphens only
- Max 64 characters
- Must match the parent directory name
- Examples: `go-rest-api`, `docker-debugging`, `python-venv`

## Skill Description Rules

- Max 60 characters for agent-created skills
- One sentence, ending with a period
- Should describe WHEN to use the skill, not WHAT it contains
- Examples: "Set up a Go REST API with chi router.", "Debug Docker container issues."

## Skill Directory Structure

```
~/.savant/skills/
└── my-skill/
    ├── SKILL.md          # Required: YAML frontmatter + markdown instructions
    ├── references/       # Optional: documentation, examples
    ├── templates/        # Optional: code templates
    ├── scripts/          # Optional: executable scripts
    └── assets/           # Optional: images, configs, etc.
```

## Editing Skills

```
skill_manage(action: "edit", name: "go-rest-api", content: "full new content")
skill_manage(action: "patch", name: "go-rest-api", old_string: "old text", new_string: "new text")
```

## Pinning Skills

Pinned skills cannot be deleted or archived by the curator. Pin skills that are essential.

## Skill Lifecycle

Skills transition through states based on usage:
- **Active** → **Stale** (after 30 days of no activity)
- **Stale** → **Archived** (after 90 days of no activity)
- **Pinned** skills are exempt from all transitions

Archived skills are never deleted - they're moved to `~/.savant/skills/.archive/`.
