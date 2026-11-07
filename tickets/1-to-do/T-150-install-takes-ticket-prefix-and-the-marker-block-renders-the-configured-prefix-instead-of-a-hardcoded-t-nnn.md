---
id: T-150
title: install takes --ticket-prefix and the marker block renders the configured prefix instead of a hardcoded T-NNN
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-150 — install takes --ticket-prefix and the marker block renders the configured prefix instead of a hardcoded T-NNN

## Outcome

A project can pick its ticket prefix at install time (`pickle install --in-tree
--ticket-prefix ABC`), and the generated `AGENTS.md` marker block shows that prefix in its trigger
phrases and bookkeeping grammar, so agents are not pointed at `T-NNN` in a project whose ids are
`ABC-NNN`.

## Description

On `main` (0017563): `pickle project add` already takes `--ticket-prefix` (T-058), but `install`
does not, so the first child's prefix can only be changed by hand-editing `pickle.toml`. After that
edit, `pickle upgrade` re-renders the branch line (`feat/ABC-NNN-<slug>`) and `ticket new` allocates
`ABC-001`. But the trigger phrases and the `board: T-NNN <verb phrase>` grammar in `markerBlock()`
(`internal/install/install.go`, the hardcoded lines near 1608-1618) stay `T-NNN`. Nothing tells the
user to run `upgrade` after editing either.

Scope:
- Add `install --ticket-prefix`, validated exactly like `project add`'s flag and written to the
  first child.
- Render the marker block's `T-NNN` placeholders from the prefix when all registered children share
  one; keep the generic `<PREFIX>-NNN` form (or `T-NNN`) when they differ. Update the golden file.
- `pickle doctor` warns when the on-disk marker block differs from what `pickle.toml` would render
  now (if it does not already), naming `pickle upgrade` as the fix.

The skill payload's `T-NNN` examples are generic syntax and stay as they are (the payload is shared
by every install).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-07 — created (TO DO). source: chat: user changed ticket_prefix after install and found AGENTS.md still says T; confirmed upgrade fixes only the branch line
