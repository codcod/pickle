---
id: T-040
title: board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift)
project: pickle
depends-on: []
spawned-by: [T-027, T-028, T-033]
impact: medium
complexity: low
cost: M
---

# T-040 — board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift)

## Description

**Epic — merged from T-027, T-028 and T-033 by the 2026-07-26 board triage.** All three are in
`tickets/7-dropped/` with their full analysis and line references; read them for detail.

Three gaps in what `pickle board audit` considers a valid ticket. All three land in
`internal/audit/audit.go`, all three are "the audit is the only component that sees *every*
ticket however it was authored" — which matters because the flow explicitly permits agents and
humans to write ticket files directly (`pickle ticket new` is a convenience, not a gatekeeper).
One file, one table-driven test, one review.

### Absorbed scope

| from | check | substance |
|---|---|---|
| T-033 | duplicate frontmatter keys | `ticket.ParseFrontmatter` (`internal/ticket/ticket.go:105-123`) assigns into a `map[string]string`, so a duplicate key **silently overwrites** — last wins — and a ticket with two `impact:` or two `project:` lines audits clean. A duplicate key is malformed however it arrived: a hand-edit, a bad merge resolution leaving two `depends-on:` lines, or a future command. |
| T-027 | self-referencing `depends-on` | The existence loop never checks whether a ticket lists **itself**. `T-042` with `depends-on: [T-042]` audits clean, then silently self-blocks: the pickup gate demands the dependency be in `6-done/`, which it can never be while in development. The failure surfaces as a confusing "dependency not done" error about the ticket itself, at pickup, instead of a frontmatter error at audit time. One condition in the existing loop. |
| T-028 | TEMPLATE.md drift | `audit.requiredKeys` (`internal/audit/audit.go:23`) and `skill/resources/TEMPLATE.md` must agree on the frontmatter key set, and **nothing enforces it**. The only guard, `TestScaffoldSectionsMatchTemplate` (`internal/ticket/ticket_test.go:146-162`), compares `## ` headings and is blind to frontmatter. A key added to the audit while TEMPLATE keeps advertising the old set makes every hand-authored ticket fail audit — **in the user's project, not in this repo's tests**. T-024 walked this tightrope by hand. |

### Correction carried over from the T-030 review (finding N3, 2026-07-26)

T-027's refinement note implied `internal/audit` holds a duplicate `T-\d+` regex to unify. **It
does not** — `internal/audit/audit.go` contains no regex and does not import `regexp`; its only
shape-adjacent checks are `t.Front["id"] != t.ID` (`:52`) and the existence lookups (`:67`,
`:80`). So the self-reference check **adds the first external caller** of `ticket.ValidID`
(`internal/ticket/ticket.go:146-175`), which today has no consumer outside its own package.
Slightly larger than "swap the regex for the helper" — worth knowing before estimating.

The `T-\d+` shape *is* still literally duplicated in `filenameRE` (`internal/ticket/ticket.go:95`)
and `board.rowRE` (`internal/board/board.go:29`). Composing all three from one fragment is
optional and this ticket's call; if it is deferred, it belongs with **T-042** (duplicated
internals) rather than here.

### Cross-references

- **T-039** adds *board-row shape* checks to the same audit. Disjoint subject (rows vs
  frontmatter) but the same file and test table — sequence them to avoid edit collisions, and
  consider a shared table-driven fixture.
- **T-036** proposes a TO DO cap enforced by the audit; if it lands first, this epic inherits its
  test scaffolding.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-027, T-028 and T-033,
  all three moved to 7-dropped/ as absorbed
