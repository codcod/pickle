---
id: T-043
title: harden the cli test harness and close the config, project and ticket-new coverage gaps
project: pickle
depends-on: []
spawned-by: [T-012, T-031]
impact: medium
complexity: medium
cost: L
---

# T-043 — harden the cli test harness and close the config, project and ticket-new coverage gaps

## Description

**Epic — merged from T-031 and T-012 by the 2026-07-26 board triage.** Both are in
`tickets/7-dropped/` with their full finding lists and line references; read them for detail.

T-029 built the first real test harness in `internal/cli` — a `TestMain` that sandboxes the
process CWD, a `TestCWDIsSandboxed` guard, and a `captureStdout` helper — and explicitly billed it
as reusable. **T-012 is that harness's first real consumer.** Landing the consumer before the
harness's known defects are closed means inheriting them at every new call site; landing the
harness without a consumer means shipping an unexercised abstraction. They are one job.

### Part 1 — harden the harness (T-031, from the T-029 review findings N1–N4, N7)

1. **`captureStdout` leaves `os.Stdout` pointing at a closed pipe after it returns**
   (`internal/cli/cli_test.go:150,154,167,171`). `w` and `r` are closed at the end of the helper,
   but the only restore is the `t.Cleanup` registered at `:154`, which runs at *test* end. Between
   the helper returning and the test finishing, every print in the package writes to a closed fd.
2. Pipe lifecycle — the remaining three findings concern ordering and leak-safety around the same
   helper.
3. `TestMain` sandbox lifecycle — setup/teardown ordering when a test fails mid-flight.
4. A comment nit that overstates what is shared.

These are cheap to close **before** a second consumer arrives and inherits them; that was T-031's
entire argument, and this epic is that second consumer arriving.

### Part 2 — close the coverage gaps (T-012, seven items)

`internal/cli` sits at ~29.5%; `internal/config` at 91.8%. The command layer is exercised almost
entirely by manual acceptance tests.

| # | item |
|---|---|
| 1 | cli-level tests for `project add\|list\|remove` and `board audit`, driving `runProject*`/`runBoardAudit` against a temp overarching root: `add` appends with defaults and rejects duplicate-name/missing-dir; `list` output; `remove` succeeds and the live-ticket guard refuses when a ticket targets the child; `board audit` exits 0 clean and non-zero broken. |
| 2 | **TOML-safe rendering** — `config.Render` formats strings with Go `%q`, which is not TOML basic-string escaping; control characters emit `\xNN`, which is invalid TOML and breaks round-trip. Escape per TOML rules or route through the encoder; round-trip test with a tab and a non-ASCII rune. |
| 3 | defaulting test — the existing `config_test.go` "zero wip" case actually asserts `-1`. |
| 4 | cli-level tests for `ticket new` (`runTicketNew`). |
| 5 | **board-row title sanitization** — see the deferral below. |
| 6 | `LastHistoryStatus` transition parsing. |
| 7 | **`Save` is neither atomic nor mode-preserving** (added 2026-07-25 by a T-018 gate finding) — a crash mid-write truncates `pickle.toml`, and the file's mode is not preserved. This is the only item here that is a correctness bug rather than coverage; consider hoisting it if the epic is split. |

### Deferral: item 5 belongs to T-044 (was T-039, dropped as superseded 2026-07-26)

T-012 item 5 ("`ticket new` writes the raw title into the board row") is the **same escape-versus-
replace question** that **T-044 settled** (2026-07-26, landed): one-way `sanitizeCell` (`|` → `¦`,
newlines → space, and — since T-049 landed, 2026-07-27 — a 120-rune cap with a trailing `…`) at
the render choke point; the parse-back machinery is deleted. T-044 already
ships renderer-level tests (`TestRenderSanitizesCells` in `internal/board/board_test.go`) and an
acceptance repro, so item 5 shrinks to at most a **cli-level** assertion that `ticket new` with a
pipe title yields an audit-clean board — or is scoped out at refinement as already covered.
(T-039, which previously owned this decision, was dropped as superseded by T-044, 2026-07-26.)

### Sequencing

- **T-042** item 3 unifies the test payload-root idiom across five files including
  `internal/cli/cli_test.go`. Arguably a prerequisite here, though not a hard `depends-on` —
  decide the order at refinement rather than editing the same files concurrently.
- **T-044** owns the item 5 decision, as above (previously T-039, dropped as superseded).
- T-012's original `depends-on: [T-001, T-002, T-003]` are all in `6-done/` and merged, so the
  gate is satisfied; the epic carries no hard dependency forward.

### Why cost L

Two multi-item tickets, one of which has seven items and a correctness bug in it. If refinement
cannot fit this under the READY gate as one unit, the clean split is **harness + item 7** (the
defects) versus **items 1–6** (the coverage), not back into T-031 and T-012.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-031 and T-012, both
  moved to 7-dropped/ as absorbed
- 2026-07-26 — patched by T-044's review impact sweep: the item-5 deferral is settled — T-044
  landed one-way sanitisation with renderer tests; item 5 shrinks to a cli-level assertion or
  is scoped out at refinement
- 2026-07-27 — patched by T-049's review impact sweep: `sanitizeCell` now also caps a cell at 120
  runes (`TestRenderCapsCellWidth` in `internal/board/board_test.go`), so the item-5 deferral's
  enumeration of the sanitisation was completed; item 5's scope is unchanged
