---
id: T-012
title: harden test coverage + TOML-safe render (config, project, board audit)
project: pickle
depends-on: [T-001, T-002, T-003]
impact: medium
complexity: low
cost: S-M
---

# T-012 — harden test coverage + TOML-safe render (config, project, board audit)

## Description

Non-blocking robustness follow-up surfaced by the T-001 review. T-001 shipped the
`internal/config` package and the `pickle project add|list|remove` commands with strong config
coverage (91.8%) but thinner command-layer coverage and a couple of edge gaps. This ticket
closes those without changing user-facing behaviour.

Scope (cohesive test-hardening + render items). Item 1's `board audit` half was added by the
T-002 review; items 4 and 5 by the T-003 review (which is why this now also depends on T-003):

1. **cli-level tests for `project add|list|remove` and `board audit`** — `internal/cli` sits at
   ~29.5% coverage; the `project` commands and `runBoardAudit` are only exercised by manual
   acceptance tests. Add table-driven tests that drive `runProject*` and `runBoardAudit` against
   a temp overarching root (temp `pickle.toml` + child dirs / tickets tree), asserting: `add`
   appends with defaults and rejects duplicate-name / missing-dir; `list` output; `remove`
   succeeds and the live-ticket remove-guard refuses when a `tickets/…/T-*.md` targets the
   child; `board audit` exits 0 on a clean tree and non-zero on a broken one.
2. **TOML-safe rendering** — `config.Render` currently formats string values with Go `%q`,
   which is not identical to TOML basic-string escaping (control characters / certain runes
   would emit `\xNN`, which is invalid TOML) and would break round-trip for exotic values.
   Escape per the TOML basic-string rules (or route values through the encoder), and add a
   round-trip test with an awkward value (e.g. a tab or non-ASCII rune in a command string).
3. **defaulting test** — the existing `config_test.go` "zero wip" case actually asserts `-1`,
   not `0`. Rename it (e.g. "negative wip") and add a case proving an **omitted / `0`** WIP
   field defaults to 1 rather than erroring.

4. **cli-level tests for `ticket new`** — `runTicketNew` is only exercised by manual acceptance
   tests. Add table-driven tests against a temp overarching root asserting: a fresh id is
   allocated (`max+1`), the scaffold file is written to `1-to-do/` and passes
   `audit.Audit` with zero errors, the board row lands under the child's sub-group in impact
   order, and the failure modes exit non-zero (unregistered `--project`, illegal grade,
   missing title).
5. **board-row title sanitization** — `ticket new` writes the raw title into both the board
   row (`| T-NNN | <title> | … |`) and the `# T-NNN — <title>` heading. A title containing a
   pipe (`|`) or newline corrupts the markdown table (extra columns) and the heading; today
   `board audit` still passes because it only parses the id. Escape or reject markdown-breaking
   characters in the title (at minimum `|` and control/newline chars) in `board.AddTODORow`
   and/or `ticket.Scaffold`, with a test proving a piped title round-trips to a well-formed
   single-cell board row.

6. **`LastHistoryStatus` transition parsing** — the parser locates the status transition with
   `LastIndex(body, "→")`, so a History **reason** clause (after the `:`) that itself contains a
   `→` is mis-parsed as the transition (surfaced while dogfooding the T-003 review: a
   `… → DONE: … 2 non-blocking → T-012` line parsed to `T-012`, failing `board audit`). Fix:
   isolate the transition by splitting on the first `:` **before** finding the arrow (the
   `OLD → NEW` transition always precedes any reason text). Add a test with an arrow in the
   reason clause.

Realistic inputs today (commands, relative paths, names, ordinary titles, History reasons
without stray arrows) are unaffected — this is hardening, hence non-blocking.

## Implementation Plan

<!-- empty until refined; must meet the READY gate (skill rules §4) before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: T-001 review (non-blocking findings N1–N3)
- 2026-07-23 — broadened by the T-002 review: added board-audit cli tests + audit path coverage (N1/N2); depends-on now [T-001, T-002]
- 2026-07-23 — broadened by the T-003 review: added ticket new cli tests + board-row title sanitization (items 4–5); depends-on now [T-001, T-002, T-003]
- 2026-07-23 — broadened again (T-003 review, dogfooding): added LastHistoryStatus transition-parsing fix (item 6)
