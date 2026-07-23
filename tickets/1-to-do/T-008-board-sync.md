---
id: T-008
title: board sync
project: pickle
depends-on: []
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

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P3)
