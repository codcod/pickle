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

Exit non-zero on any error; print each error with a path/line reference. Phase P1. Depends on
T-001; underpins T-007 and T-008.

> **Impact note (from the T-001 review, 2026-07-23):** T-001 introduced a ticket-frontmatter
> scanner in `internal/cli/project.go` (`ticketProject` reads `project:`; `liveTicketsTargeting`
> globs `tickets/{1-to-do..5-rework}/T-*.md`). Board audit parses the *same* frontmatter (`id`,
> `project:`, `depends-on:`, …) and enumerates the same status dirs — so extract a shared
> parser (e.g. an `internal/ticket` package: parse frontmatter + status-dir list) and refactor
> the T-001 remove-guard to reuse it, rather than duplicating the scan. Consume
> `config.Config`/`Project(name)` for the `project:`-is-registered check and the per-child WIP
> limits.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1)
