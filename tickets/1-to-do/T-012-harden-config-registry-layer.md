---
id: T-012
title: harden test coverage + TOML-safe render (config, project, board audit)
project: pickle
depends-on: [T-001, T-002]
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

Scope (cohesive test-hardening + render items). Items 1 and 4 were added by the T-002 review
(findings N1/N2), which is why this now also depends on T-002:

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

Realistic inputs today (commands, relative paths, names) are unaffected — this is hardening,
hence non-blocking.

## Implementation Plan

<!-- empty until refined; must meet the READY gate (skill rules §4) before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: T-001 review (non-blocking findings N1–N3)
- 2026-07-23 — broadened by the T-002 review: added board-audit cli tests + audit path coverage (N1/N2); depends-on now [T-001, T-002]
