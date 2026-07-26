---
id: T-033
title: board audit: flag duplicate frontmatter keys
project: pickle
depends-on: []
spawned-by: [T-030]
impact: medium
complexity: low
cost: S
---

# T-033 — board audit: flag duplicate frontmatter keys

## Description

> **ABSORBED into T-040 (board triage, 2026-07-26) — this ticket is closed, its work is not.**
> Everything below stands as the record: the analysis, measurements and line references
> are still the authoritative detail for this part of T-040's scope. Do not re-file it;
> do not implement from here. T-040 is the refinable, reviewable unit.

`ticket.ParseFrontmatter` (`internal/ticket/ticket.go:105-123`) assigns into a `map[string]string`
as it walks the frontmatter block, so a **duplicate key silently overwrites** — last occurrence
wins — and nothing anywhere reports that the file had two of them. `pickle board audit` therefore
calls a ticket with two `impact:` lines, or two `project:` lines, perfectly clean.

Split out of **T-030** at refinement (2026-07-25), where the duplicate keys came from newline
injection through `ticket new`'s unvalidated `title` and `--spawned-by`. T-030 closes that *input*
boundary. This ticket closes the *validation* boundary, which is the one that generalises: a
duplicate key is malformed regardless of how it got there — a hand-edit, a bad merge resolution
leaving two `depends-on:` lines, a future command, or an agent writing a ticket file directly (which
this flow explicitly permits — `pickle ticket new` is a convenience, not a gatekeeper). The audit is
the only component that sees *every* ticket however it was authored.

Measured behaviour (built binary, throwaway install, 2026-07-25):

```
pickle ticket new "$(printf 'y\ncost: XL')" --project demo
→ created T-002; frontmatter line 4 is `cost: XL`, line 10 is `cost: M`
→ board audit: 2 tickets, 0 error(s), 0 warning(s)
```

Note the **last-wins** direction matters for how this is graded. Today it is accidentally
protective: `Scaffold` emits the legitimate keys in a fixed order, and an injected line always
lands *before* the real key it shadows, so the real value survives and the damage is confined to
visible garbage in the file. The one exception is `id:`, which `Scaffold` emits *first* — an
injected `id:` does win, and is already caught by the separate `frontmatter id != filename id`
check. But nothing anchors last-wins: it is an emergent property of a map assignment in a loop, not
a documented decision. Any future change that makes the parse first-wins, or that reorders
`Scaffold`, silently converts today's cosmetic corruption into wrong metadata. Detecting the
duplicate removes the dependence on that accident entirely.

### Scope

- `internal/ticket/ticket.go` — `ParseFrontmatter` must surface duplicates. It currently returns
  `(map[string]string, bool)` and has several callers (`internal/audit`, `internal/ticket`'s own
  loader, `internal/sync`, `internal/doctor` — enumerate them at refinement), so the signature
  change is the main design question: return a `[]string` of offending keys, expose a second
  function, or have the loader record the issue in the existing structural-issues channel
  (`LoadAll`'s second return, which already carries "bad filename, no frontmatter"). Prefer the
  last if it fits — it needs no signature change and the audit already prints those.
- `internal/audit/audit.go` — report duplicates as an **error** (a malformed ticket, not a style
  preference), with the key named.
- `internal/audit/audit_test.go` — a duplicate-key fixture.
- **Regression risk gate:** this changes a *shipped* validator, so a previously-clean tree can
  start erroring. Run `pickle board audit` against this repo's real `tickets/` (33+ tickets) and
  against `skill/resources/TEMPLATE.md`'s frontmatter before declaring done — the same trap T-028
  exists for.
- Docs: the `board audit` check list in `README.md` and the *audit the board* section of
  `skill/SKILL.md`, if they enumerate checks.

### Couplings

`spawned-by: [T-030]` — split out of T-030's refinement with user sign-off. No hard dependency in
either direction: T-030 stops the injection at the input boundary, this one detects the malformation
wherever it came from, and each is useful alone. **T-030 should land first** (it is the actual bug
report and it removes the only known *producer* of duplicates), but nothing enforces that.

Soft couplings (no `depends-on`):

- **T-027** ("audit: flag depends-on self-references") and **T-028** ("guard TEMPLATE.md frontmatter
  against audit requiredKeys") also edit `internal/audit`. T-028 is directly relevant: it exists
  because TEMPLATE.md's frontmatter is not validated the way real tickets are, and any new
  frontmatter check inherits that question. Whoever lands second should re-run the others' cases.
- **T-015** ("consolidate board status-heading matching…") is the ticket already holding
  parser-consolidation debt; if `ParseFrontmatter`'s signature has to change here, check whether
  T-015 wants the same call sites.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-26 — TO DO → DROPPED: absorbed into T-040 (board triage merge); content preserved here as the record
