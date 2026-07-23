---
id: T-004
title: install (scaffold + skill install + marker injection + first child)
project: pickle
depends-on: []
impact: high
complexity: high
cost: L
---

# T-004 — install (scaffold + skill install + marker injection + first child)

## Description

Implement `pickle install`, run once in an overarching project:

- create `tickets/` with the seven ordered status dirs;
- copy the embedded board skeleton (`skill/resources/BOARD.md`) → `tickets/BOARD.md` (set the
  date);
- write the short `tickets/README.md` pointer;
- install the embedded `skill/` tree → `.agents/skills/ticket-flow/`, and symlink
  `.claude/skills/ticket-flow` → it for Claude Code;
- inject an **idempotent** `<!-- pickle:begin -->` / `<!-- pickle:end -->` marker block into
  `AGENTS.md` (and `CLAUDE.md`, or symlink `CLAUDE.md → AGENTS.md` — a flag) stating "start at
  `tickets/BOARD.md`" + the project configuration;
- write `pickle.toml` and register the first child-project (prompt, or `--project <name> <path>`).

Per-project (never writes to `~/`), idempotent, safe to re-run. Detect/select agents
(`--agent claude,pi,opencode` or auto-detect from existing `.claude/`, `.pi/`, `AGENTS.md`).
This bootstrap repo is the reference for what a correct install produces. Needs T-001. Phase P2.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
