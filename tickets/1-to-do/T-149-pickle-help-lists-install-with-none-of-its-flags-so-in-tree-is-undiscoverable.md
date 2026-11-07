---
id: T-149
title: pickle help lists install with none of its flags, so --in-tree is undiscoverable
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-149 — pickle help lists install with none of its flags, so --in-tree is undiscoverable

## Outcome

`pickle help` shows the install flags a new user needs on day one (`--in-tree`, `--agent`,
`--path`, `--project`), so choosing a layout no longer depends on reading `pickle install --help` or
the manual first.

## Description

On `main` (0017563), the `install` entry in `pickle help` describes what install does but lists
none of its flags. `--in-tree` appears only in `pickle install --help` (the `flag` package's usage
dump). It is the one choice that is costly to undo after the fact (see T-148), so it belongs in the
first help a user reads.

Add an indented flag summary under `install` in the top-level help, in the style the
`ticket new` entry already uses. Keep `docs_xref_test.go` green, and update the user manual
wherever it quotes the help text. Soft coupling: T-121 (install help text accuracy).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-07 — created (TO DO). source: chat: user found --in-tree absent from pickle help; confirmed on main
