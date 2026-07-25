---
id: T-027
title: audit: flag depends-on entries that reference the ticket itself
project: pickle
depends-on: []
impact: low
complexity: low
cost: S
---

# T-027 — audit: flag depends-on entries that reference the ticket itself

## Description

`pickle board audit` validates that every `depends-on:` target **exists** (`internal/audit/audit.go`,
the `t.DependsOn` loop) but never checks whether a ticket lists **itself**. `T-042` with
`depends-on: [T-042]` therefore audits clean, and then silently self-blocks: the
`3-in-development/` dependency gate demands the target be in `6-done/`, which a ticket can
never be while it is being developed. The failure surfaces as a confusing "dependency not
done" error about the ticket itself, at pickup time, instead of as a frontmatter error at
audit time.

The fix is one condition in the existing existence loop: error
`"%s: depends-on lists itself"` when a listed id equals the ticket's own id — the same check
T-024 adds for `spawned-by:`.

### Why it is a separate ticket

Surfaced while refining T-024 (which introduces `spawned-by:` with a self-reference check from
day one). Extending the check to `depends-on:` changes the behaviour of a **shipped** validator
— a previously-clean tree could start erroring — so it was deliberately kept out of T-024's
scope rather than smuggled in alongside the new field.

### Scope

- `internal/audit/audit.go` — one condition in the `depends-on` existence loop.
- `internal/audit/audit_test.go` — a self-reference fixture expecting the error.
- Docs: the `depends-on` invariant lines in `README.md` (the `board audit` bullet list) and
  `skill/SKILL.md` (*audit the board*), if they enumerate the checks.
- Confirm the real board stays clean (no existing ticket self-references).

### Couplings

Soft coupling to **T-024** (which lands the parallel `spawned-by` self-check and is the reason
this asymmetry exists). Not a hard dependency: the two edits touch the same loop but neither
needs the other. Whichever lands second should match the wording of the first.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: refinement of T-024 (scope split: self-reference check
  for the shipped `depends-on` validator kept out of T-024)
