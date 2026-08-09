---
id: T-076
title: consume rick status --json as an artifact-state source
project: pickle
depends-on: []
spawned-by: []
family: T-075
impact: high
complexity: medium
cost: M
---

# T-076 — consume rick status --json as an artifact-state source

## Outcome

After this ships, pickle can tell, for a given ticket, which rick artifacts exist and which are awaiting human approval — read from rick's own versioned `rick status --json` contract, never re-derived by scanning disk.

## Description

The first member of T-075: teach pickle to learn, for a given ticket, which rick artifacts
exist and which are awaiting human approval. Everything else in the family reads this.

**Consume, do not reimplement.** `rick status --json` is a versioned public contract
(`sdlc-cli/internal/status/report.go`: `schemaVersion = 2`, documented as additive-only —
"removing or renaming a field is a breaking change that requires bumping this constant").
Its `Workflow.Tickets[].Artifacts[]` carries `Path`, `Kind`, `Status`, `Date` per artifact,
derived from disk by rick's own oracle. Crucially the **JSON is multi-ticket** even though
rick's text renderer collapses to a single one, so it is already the shape pickle needs.

Re-deriving that state by scanning `docs/specs/` ourselves would fork rick's kind-detection
and status rules and drift from them silently. Shelling out keeps rick the authority on rick's
state and pickle the presentation layer — the same judgment/mechanics split `DESIGN.md` §2
already draws.

Scope:

- **Opt-in per child-project**, config-declared, never auto-detected — matching `DESIGN.md`
  §3 decision 6 (children are registered intentionally, not discovered). Something like a
  `rick = true` / `specs_root = "docs/specs"` pair on `[[project]]`; the exact schema is a
  refinement decision.
- **Invoke and parse** `rick status --json` against the child's path, keyed by
  `schemaVersion`. An unknown version is refused with a clear message rather than
  best-effort parsed — a silently mis-parsed gate state is worse than no gate state.
- **Map artifacts to tickets via `ticket_prefix` alone.** T-058 (done) already makes a
  pickle id like `DR-142` identical to rick's `docs/specs/DR-142/` directory name, and
  `audit.go` already enforces that a ticket's id prefix matches its child's configured
  prefix. No new frontmatter field, no mapping table.
- **Fail open, always.** rick not installed, not on `PATH`, erroring, or the child not
  opted in ⇒ no artifacts, no warning, no non-zero exit. This must not become a new way for
  `board audit` to go red in projects that never heard of rick.

The deliverable is the library layer plus whatever surfacing is needed to prove it works
(a `--json` field or a `doctor` line); the visual surface is T-077.

Soft coupling: T-065 (expose board and ticket state as a versioned JSON read projection) is
the exact mirror of this ticket — pickle emitting what rick here consumes. They should agree
on versioning discipline, and whichever lands second should copy the first's conventions.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
