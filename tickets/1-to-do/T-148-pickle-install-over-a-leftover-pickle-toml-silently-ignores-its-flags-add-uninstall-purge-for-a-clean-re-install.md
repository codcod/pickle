---
id: T-148
title: pickle install over a leftover pickle.toml silently ignores its flags; add uninstall --purge for a clean re-install
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-148 — pickle install over a leftover pickle.toml silently ignores its flags; add uninstall --purge for a clean re-install

## Outcome

Re-running `pickle install` with different flags (the "forgot `--in-tree`" case) either takes
effect or is refused with a message naming the conflict, never silently ignored. `pickle uninstall
--purge` returns a project with no tickets to its pre-install state, so a clean re-install is one
command.

## Description

Reproduced on `main` (0017563) in a throwaway dir: `pickle-test install`, `pickle-test uninstall`,
`pickle-test install --in-tree`. The second install reports `= pickle.toml (exists)` and keeps the
first install's `layout = "umbrella"` with no `[[projects]]`, yet prints
`pickle installed in … (child "<dir>")`. The `--in-tree` flag is silently dropped, the success line
names a child that was never registered, and the regenerated `AGENTS.md` marker block still
describes the umbrella layout.

Two parts:

1. **Install over an existing `pickle.toml`** (the root cause; any leftover config hits it, not only
   after uninstall). When the flags ask for a layout or first child that disagrees with the
   existing config, refuse with a message naming the disagreement and the way out
   (`uninstall --purge`, or `project add`). When they agree, or no layout/child flags were given,
   keep today's idempotent behaviour. The success line names a child only if one is registered.
2. **`pickle uninstall --purge`.** On top of today's removals, it also removes `pickle.toml`, the
   untouched `tickets/` scaffold (the `.gitkeep` files, the status dirs, and `BOARD.md`, `NOTES.md`
   and `README.md` while they are unchanged from what install wrote), the now-empty
   `.agents/skills/` and `.claude/skills/` dirs, and `AGENTS.md`/`CLAUDE.md` if they are empty once
   the marker is stripped. It **refuses** if any ticket file exists or `NOTES.md` was edited. That
   keeps T-006's decision D1 (uninstall never destroys work): `--purge` only deletes what install
   itself created and nobody has touched since. It honours `--dry-run`.

Soft couplings: T-006 (uninstall scope, D1), T-108 (in-tree layout, the conflicting-flags refusal
this extends).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-07 — created (TO DO). source: chat: reproduced the user-reported re-install trap (install, uninstall, install --in-tree silently keeps the old umbrella config and misreports a child) in a throwaway dir
