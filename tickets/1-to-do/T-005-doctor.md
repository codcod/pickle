---
id: T-005
title: doctor
project: pickle
depends-on: [T-004]
impact: medium
complexity: low-medium
cost: S-M
---

# T-005 — doctor

## Description

Implement `pickle doctor` — the install-side analogue of `board audit`. Verify install
integrity and report actionable diagnostics: the skill is present with the expected payload
version, the `.agents/`/`.claude/` symlinks resolve, the `<!-- pickle:begin/end -->` marker
block is present in `AGENTS.md`/`CLAUDE.md`, `pickle.toml` parses, and every registered
child-project path resolves to a git repository. Exit non-zero when anything is wrong. Needs
T-001 and T-004. Phase P2.

> **T-004 artifact set (what `doctor` verifies)**, as implemented in `internal/install`:
> `.agents/skills/ticket-flow/` (copied payload — real dir, unless a dev/self-host symlink),
> `.claude/skills/ticket-flow -> ../../.agents/skills/ticket-flow`, the
> `<!-- pickle:begin -->`/`<!-- pickle:end -->` pair in `AGENTS.md` (+`CLAUDE.md`, or a
> `CLAUDE.md -> AGENTS.md` symlink under `--claude-symlink`), a `.gitkeep` in each of the seven
> `tickets/<status>/` dirs, `tickets/BOARD.md` + `tickets/README.md`, and `pickle.toml`. Reuse
> `internal/install`'s constants (`skillDir`, `markerBegin/End`) rather than re-hardcoding.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
