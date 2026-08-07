---
id: T-077
title: show a ticket's rick artifacts in pickle serve, read-only, with the effective-instance rule
project: pickle
depends-on: [T-076]
spawned-by: []
family: T-075
impact: medium
complexity: medium
cost: M
---

# T-077 — show a ticket's rick artifacts in pickle serve, read-only, with the effective-instance rule

## Description

The reading surface, and the reason the whole family is worth building: reviewing a
400-line solution design in a terminal, before approving it, is miserable. rick's approval
gate asks a human to accept an artifact it has just rendered into a scrollback buffer.
pickle already knows how to render markdown well.

Scope — **read-only, zero writes to `docs/specs/**`**:

- a route (`GET /specs/{key}/{name}`) rendering an artifact through the existing pipeline in
  `internal/serve/markdown.go` — GFM on, `goldmark.WithUnsafe` off, so raw HTML in an
  artifact is escaped rather than executed, exactly as for ticket bodies;
- on the ticket page, the ticket's artifacts listed with their `Kind` and a `Status` badge
  (`draft` / `complete` / `approved`) from T-076, so "what is this ticket waiting on" is
  visible from the board rather than only from inside the agent session.

### The effective-instance rule (the part that earns its keep)

rick's artifact paths embed a date and topic — `solution-design-2026-06-14-<topic>.md` — so
a single *kind* can accumulate several instances. `[R]evise` replaces at the same path only
within a day; across days it writes a new file. Meanwhile `rick verify` resolves which one
counts by taking the newest: filename-date descending, then mtime, then name — and
**status-blind** (`sdlc-cli/internal/verify/plan.go`). Nothing in rick surfaces that choice
to a human, so it is entirely possible to read, amend and approve an artifact that is not
the one rick will consume.

So this view must: mark which instance is **effective** per kind using rick's own rule, flag
any kind holding more than one instance, and warn when the effective instance is not the
approved one. That is brine's invariant-audit discipline (`internal/audit`) pointed at rick's
tree — a genuine contribution back, not a nicer font.

Refinement must settle whether the duplicate/effective mismatch is a warning in the UI only,
or also a `board audit` finding. The latter is tempting and probably wrong: `board audit`
asserts *brine's* invariants, and a red audit caused by another product's file layout would
violate T-075's fail-open invariant.

Soft coupling: T-055 (the serve board's at-limit WIP badge) touches the same board templates.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
