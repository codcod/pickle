---
id: T-002
title: board audit engine
project: pickle
depends-on: [T-001]
impact: high
complexity: medium-high
cost: M-L
---

# T-002 — board audit engine

## Description

Native, dependency-free reimplementation of the invariant checker as `pickle board audit` (the
keystone — a pure function over `tickets/` + `pickle.toml`, testable with fixture directories).

Verify every invariant:

- each ticket file sits in a known status dir and its filename is `T-NNN-<slug>.md`;
- frontmatter is complete (`id`, `title`, `project`, `depends-on`, `impact`, `complexity`,
  `cost`), grades are legal (single values or adjacent-pair ranges), the id matches the
  filename, and ids are unique across all status dirs (one global namespace);
- `project:` names a **registered child** (needs T-001);
- every `depends-on:` target exists;
- every ticket appears exactly once on `BOARD.md`, in the section **and** child sub-group
  matching its directory, and every board row has a backing file;
- **per-child** WIP limits hold;
- each ticket's last History transition matches the directory it lives in;
- no ticket is in `3-in-development/` while a dependency is not in `6-done/` (warning if a done
  dependency has no `merged` History line — checked against the dependency's **own** child repo,
  since dependencies may cross children).

Exit non-zero on any error; print each error with a path/line reference. Phase P1. Soft-coupled
to T-001; underpins T-007 and T-008.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1)
