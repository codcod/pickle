---
id: T-001
title: pickle.toml config model + project registry
project: pickle
depends-on: []
impact: high
complexity: medium
cost: M
---

# T-001 — pickle.toml config model + project registry

## Description

Define and implement the `pickle.toml` schema and a Go package to load, validate, and save it,
plus the `pickle project add|list|remove` commands on top.

Schema (see the hand-written bootstrap `pickle.toml` for the current shape): an overarching
block (`payload_version`, optional overarching `review_addendum`, a `[commit]` policy) and a
`[[project]]` array of registered child-projects — each with `name`, `path`, build/validate
commands (`build`/`test`/`lint`/`docs`), `branch_prefix`, per-child WIP limits
(`wip_in_development`/`wip_in_review`), and an optional per-child `review_addendum`.

`project add <name> <path>` appends a validated `[[project]]` block (unique name, resolvable
path, sensible defaults); `project list` prints the registry; `project remove <name>` refuses
while any live ticket targets that child.

Foundation for the rest of the CLI: **board audit** (T-002) validates each ticket's `project:`
against this registry; **ticket new** (T-003) validates `--project`; **install** (T-004) writes
the file and registers the first child; **doctor** (T-005) resolves child paths. Phase P1/P2
foundation. Soft-coupled to T-002, T-003, T-004.

## Implementation Plan

<!-- empty until refined; must meet the READY gate (skill rules §4) before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1/P2)
