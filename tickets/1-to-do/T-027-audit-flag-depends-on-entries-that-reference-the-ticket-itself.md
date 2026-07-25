---
id: T-027
title: audit: flag depends-on entries that reference the ticket itself
project: pickle
depends-on: []
spawned-by: []
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

### Also worth folding in: id **shape** validation

Added by the T-024 review (finding N3). Neither `depends-on` nor `spawned-by` checks that a
token even looks like a ticket id, so `--spawned-by "banana"` produces
`ERROR: … spawned-by banana does not exist` — framing a malformed token as a missing ticket.
Duplicates (`[T-001, T-001]`) are likewise accepted. A shared `^T-\d+$` check in the same loops
fixes both messages at once. Coordinate with **T-030**, which proposes validating the same
tokens at *creation* time; the two should share one helper rather than diverge.

**Resolved by T-030's refinement (2026-07-25) — the helper is decided, so import it, don't invent
one.** T-030 (now READY) exports it from `internal/ticket`: `ticket.ValidID(s string) bool` for the
shape check and `ticket.ParseIDList(raw string) ([]string, error)` for the validate-and-de-duplicate
list form, both sitting beside `ParseDepends`/`ValidGrade`. This ticket's audit-side loops should call
`ticket.ValidID`. Note the division of labour T-030 fixed deliberately and this ticket must preserve:
**shape** is checked at creation, **existence** stays the audit's job — a `depends-on` pointing at a
not-yet-filed ticket is legal input that the audit flags, not a creation-time error. If T-027 somehow
lands before T-030, put the regex in `internal/ticket` anyway, in that exported form.

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

**Drive-by requested by T-029's review (finding N6).** T-029 added a comment at
`internal/move/move.go:98-100` that names the test guarding the "lineage never gates a pickup"
invariant (`TestSpawnedByDoesNotGatePickup`). Its twin at `internal/audit/audit.go:138-139` states
the same invariant but does **not** name its guard,
`TestAudit/"in-dev spawned-by parent not done"` (`internal/audit/audit_test.go:155`) — so the
"greppable pair" only greps in one direction. Since this ticket already edits both
`internal/audit/audit.go` and `internal/audit/audit_test.go`, add the case name to that comment
here rather than opening a ticket for one line. Purely cosmetic; no behaviour change.

Also note that T-029 now pins the audit string `"spawned-by T-404 does not exist"` from a **second**
package (`internal/cli/cli_test.go:232`, alongside `audit_test.go:143`). If this ticket's wording
alignment touches that message, two packages must be updated.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: refinement of T-024 (scope split: self-reference check
  for the shipped `depends-on` validator kept out of T-024)
- 2026-07-25 — scope extended: id shape validation (`^T-\d+$`) for both id lists, from the
  T-024 review's non-blocking finding N3
