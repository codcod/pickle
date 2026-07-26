# Board

The maintained index of every ticket in `tickets/`, across all registered child-projects.
**Update this file on every ticket move** — see `tickets/README.md` (board rule). A move that
doesn't touch this file is a bug.

Within each status section, tickets are **sub-grouped by child-project** under a `### <child>`
heading; TO DO / READY are ordered by descending impact inside each group. This repo has one
child, **`pickle`** (the repo root; see `../pickle.toml`).

**WIP limits (per child-project):** `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1

> ⚠️ **Do not run `pickle board sync` on this file until T-044 lands.** Three confirmed
> data-loss behaviours, all reproduced 2026-07-26: it **deletes the prose notes** below while
> reporting "reformat only", it **un-escapes the pipe** in T-021's title back into a malformed
> row, and it **overwrites a hand-corrected `branch` cell** with the slug-derived guess. All
> three dissolve under T-044 (READY), which makes this file a pure generated artifact — T-039,
> which would have hardened the current design instead, was dropped as superseded 2026-07-26.
> `pickle board audit` is safe; `pickle ticket move` is safe except that it re-renders the
> moved row unescaped — see below.
>
> **T-021's DROPPED row is a live reproduction.** Escaping it to `\|` fixes the rendered table
> but not `insertIntoBoard`, whose `strings.Split(line, "|")` is not escape-aware, so the row
> still splits into 4 fields in a 3-column section. It re-corrupted on the 2026-07-26 merge when
> `MoveRow` re-rendered it from frontmatter. Left escaped for legibility; T-044 deletes
> `insertIntoBoard` outright and needs no fixture to write the failing test.

Last updated: 2026-07-26 (T-036 reviewed → DONE: 0 blocking, 11 non-blocking all dispositioned,
0 new tickets; awaiting the human's merge)

---

## IN DEVELOPMENT

### pickle (0/1)

| id | title | branch | depends-on |
|---|---|---|---|

## IN REVIEW

### pickle (0/1)

| id | title | branch | depends-on |
|---|---|---|---|

## REWORK

### pickle

| id | title | branch | open findings |
|---|---|---|---|

## READY (impact order, per child)

### pickle

| id | title | impact | complexity | cost | depends-on |
|---|---|---|---|---|---|
| T-044 | demote BOARD.md to a generated artifact; ticket files become the single source of truth | high | medium | M | [] |

## TO DO (impact order, per child)

### pickle

**Parked: T-009, T-010, T-016.** Real but explicitly **unscheduled** — nothing is blocked on
them and no user has asked. Do not pick one up without a demand signal. Parked status is
recorded in each ticket's Description and History (the board's grade cells are rewritten from
frontmatter by `pickle board sync`, so it cannot be encoded there). Triage 2026-07-26.

**Epic merge executed 2026-07-26 — 14 tickets → 5 epics.** The sources are in `7-dropped/` with
reason `absorbed into T-0NN`; each opens with an ABSORBED banner and keeps its full analysis,
measurements and line references. They are the authoritative detail; the epic is the refinable,
reviewable unit. Do not re-file a source or implement from it.

| epic | absorbs |
|---|---|
| T-039 — BOARD.md write/validate integrity | T-014, T-023, T-034, T-037 |
| T-040 — ticket frontmatter validation | T-027, T-028, T-033 |
| T-041 — marker-block freshness | T-020, T-021 |
| T-042 — collapse duplicated internals | T-015, T-017, T-032 |
| T-043 — test harness + cli coverage | T-031, T-012 |

Deliberately **not** merged: T-013 (10 items, its own epic already), T-019 (docs-only), T-038
(input contract, successor to T-030), T-022, T-026, T-036.

**T-045 is measurement-gated, not just low priority.** It holds the two valves split out of
T-036 (backlog cap, `user-visible:` axis). Both are backstops for the leak T-036 plugs, so it
must not be refined until T-036 has landed and the spawn rate has been re-measured over at
least three reviews. Dropping it is a legitimate outcome.

**Known cross-epic decisions.** **T-044 won the T-039-vs-T-044 design decision** (2026-07-26):
the board becomes a generated artifact; T-039 (harden the hand-maintained design) was dropped as
superseded, and its move-atomicity residue (T-014·4) is folded into T-044. Escape-vs-replace is
settled by T-044's one-way cell sanitisation — **T-043 item 5 defers to T-044**.
T-042 collides with T-044 (`internal/board`, `internal/sync`) and with T-043 (`cli_test.go`) —
sequence, do not run concurrently.

| id | title | impact | complexity | cost | depends-on |
|---|---|---|---|---|---|
| T-026 | upgrade refuses legal pickle.toml files and misdiagnoses why | high | medium | M | [] |
| T-022 | skill payload states commit policy, branch prefix and WIP limits unconditionally | medium | low | S | [] |
| T-040 | board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift) | medium | low | M | [] |
| T-041 | keep the AGENTS.md marker block fresh and detect drift | medium | medium | M | [] |
| T-043 | harden the cli test harness and close the config, project and ticket-new coverage gaps | medium | medium | L | [] |
| T-038 | tighten ticket new's title contract: Unicode line terminators and length cap | low-medium | low | S | [] |
| T-045 | backlog cap and user-visible axis: decide after measuring whether the T-036 disposition valves lowered the spawn rate | low-medium | medium | M | [] |
| T-042 | collapse duplicated internal predicates into single helpers (status headings, marker span, test payload root) | low | low | M | [] |
| T-013 | install polish (marker spacing, summary labels, cli tests, --agent) | low | low | S | [T-004] |
| T-019 | README accuracy polish (prose duplicates command table, phased-plan tagging) | low | low | S | [] |
| T-009 | opencode wiring | medium | medium | M | [T-004] |
| T-010 | Pi guardrail scaffold | medium | medium | M | [T-004] |
| T-016 | ship docs-readability as an optional review step (Step 4b) | low | medium | M | [] |

## DONE

### pickle

| id | title | merged |
|---|---|---|
| T-001 | pickle.toml config model + project registry | yes — merged to main 2026-07-23 (cdad65e) |
| T-002 | board audit engine | yes — merged to main 2026-07-23 (fca3ea1) |
| T-003 | ticket new (id allocation + template + board row) | yes — merged to main 2026-07-23 (6a6fa72) |
| T-004 | install (scaffold + skill install + marker injection + first child) | yes — merged to main 2026-07-23 (33f05e3) |
| T-005 | doctor | yes — merged to main 2026-07-24 (b199215) |
| T-006 | upgrade + uninstall | yes — merged to main 2026-07-25 (4bcfc00) |
| T-007 | ticket move (state machine + per-child WIP + cross-child merge gate) | yes — merged to main 2026-07-24 (fd70a82) |
| T-008 | board sync | yes — merged to main 2026-07-24 (9b87a61) |
| T-011 | distribution (goreleaser + Homebrew tap + releases + docs) | yes — merged to main 2026-07-24 (e4aaed7) |
| T-018 | upgrade must not silently discard user content (pickle.toml comments, AGENTS.md marker body) | yes — merged to main 2026-07-25 (1485242) |
| T-024 | add spawned-by: lineage frontmatter field (provenance, non-gating) | yes — main (3c4c131, squashed) |
| T-029 | regression-test the non-gating guarantee at the move.go pickup gate | yes — main (0b7cd91, squashed) |
| T-030 | ticket new writes unsanitised input into frontmatter (newline injection) | yes — main (a29fde1, squashed) |
| T-036 | ratify the four review-finding dispositions already in use; make note-and-close the default | no — publish-gated (branch feat/T-036-review-disposition-valves) |

## DROPPED

### pickle

| id | title | reason |
|---|---|---|
| T-035 | repair the corrupted T-021 board row (unescaped pipe in title) | board row repaired inline during triage; ticket overhead exceeded the one-character fix |
| T-025 | backfill true historical spawned-by lineage from existing source: lines | lineage archaeology with no consumer; source: History lines already carry provenance |
| T-014 | board-row and move polish (WIP counts, cell escaping, subgroup spacing, atomicity) | absorbed into T-039 (board triage merge); content preserved here as the record |
| T-023 | board branch column is derived from the filename slug, not the ticket's real branch | absorbed into T-039 (board triage merge); content preserved here as the record |
| T-034 | board audit: flag table rows with the wrong cell count; harden AddTODORow insert point | absorbed into T-039 (board triage merge); content preserved here as the record |
| T-037 | board sync silently deletes hand-written prose from BOARD.md sections | absorbed into T-039 (board triage merge); content preserved here as the record |
| T-027 | audit: flag depends-on entries that reference the ticket itself | absorbed into T-040 (board triage merge); content preserved here as the record |
| T-028 | guard TEMPLATE.md frontmatter against audit requiredKeys | absorbed into T-040 (board triage merge); content preserved here as the record |
| T-033 | board audit: flag duplicate frontmatter keys | absorbed into T-040 (board triage merge); content preserved here as the record |
| T-020 | doctor: detect AGENTS.md marker-block drift | absorbed into T-041 (board triage merge); content preserved here as the record |
| T-021 | project add\|remove leave the AGENTS.md marker block stale | absorbed into T-041 (board triage merge); content preserved here as the record |
| T-015 | consolidate board status-heading matching and fill sync test gaps | absorbed into T-042 (board triage merge); content preserved here as the record |
| T-017 | unify marker-pair detection + dry-run fidelity | absorbed into T-042 (board triage merge); content preserved here as the record |
| T-032 | unify the test payload-root idiom into one CWD-independent helper | absorbed into T-042 (board triage merge); content preserved here as the record |
| T-031 | harden the internal/cli test harness (captureStdout stdout restore + pipe lifecycle, TestMain sandbox lifecycle) | absorbed into T-043 (board triage merge); content preserved here as the record |
| T-012 | harden test coverage + TOML-safe render (config, project, board audit) | absorbed into T-043 (board triage merge); content preserved here as the record |
| T-039 | BOARD.md write and validate integrity (escaping, sync preservation, row shape, branch cell) | superseded by T-044 (generated board); move-atomicity residue folded into T-044 |

---

## Dependency chain (hard `depends-on:`, human-approved 2026-07-23)

- **T-001** (config/registry) → **T-002**, **T-003**, **T-004**.
- **T-002** (audit) → **T-007**, **T-008**.
- **T-003** (ticket new) → **T-012** (hardening).
- **T-004** (install) → **T-005**, **T-006**, **T-009**, **T-010**, **T-013**.

## Known soft couplings (cross-referenced in ticket Descriptions, not `depends-on`)

- **T-011** (distribution) wants the command set (P1–P3) essentially complete — narrative
  coupling only, no hard `depends-on`.
