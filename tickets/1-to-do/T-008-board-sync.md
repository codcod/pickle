---
id: T-008
title: board sync
project: pickle
depends-on: [T-002]
impact: medium
complexity: medium
cost: M
---

# T-008 — board sync

## Description

Implement `pickle board sync` — regenerate/repair `BOARD.md` rows from ticket frontmatter and
file locations: correct sections, per-child `### <child>` sub-groups, impact ordering within
each group, WIP counts, and dependency columns. An escape hatch for when hand-edits drift, so
the board stays hand-maintainable but is always recoverable. Shares the parsing/model layer
with `board audit` (T-002). Phase P3.

> **Impact note (from the T-002 review, 2026-07-23):** the parsing/model layer landed — reuse
> `internal/ticket` (`Statuses`, `LoadAll`) and `internal/board` (`Parse` + `Row{Status, Child,
> ID}`) to read current state, and treat `internal/audit.Audit` returning zero errors as the
> definition of “in sync” (sync should make `board audit` pass).

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P3)
