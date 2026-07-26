# Board

The maintained index of every ticket in `tickets/`, across all registered child-projects.
**Update this file on every ticket move** — see `tickets/README.md` (board rule). A move that
doesn't touch this file is a bug.

Within each status section, tickets are **sub-grouped by child-project** under a `### <child>`
heading; TO DO / READY are ordered by descending impact inside each group. This repo has one
child, **`pickle`** (the repo root; see `../pickle.toml`).

**WIP limits (per child-project):** `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1

> ⚠️ **Do not run `pickle board sync` on this file until T-037 lands.** Three confirmed
> data-loss behaviours, all reproduced 2026-07-26: it **deletes the prose notes** in the TO DO
> section while reporting "reformat only" (T-037), it **un-escapes the `\|`** in T-021's title
> back into a malformed 7-cell row (T-014·2), and it **overwrites a hand-corrected `branch`
> cell** with the slug-derived guess (T-023). `pickle board audit` and `pickle ticket move` are
> safe and were used for every move recorded here.

Last updated: 2026-07-26 (triage: T-036/T-037 filed; T-035/T-025 dropped; T-026 regraded high; T-009/T-010/T-016 parked)

---

## IN DEVELOPMENT

### pickle (1/1)

| id | title | branch | depends-on |
|---|---|---|---|
| T-030 | ticket new writes unsanitised input into frontmatter (newline injection) | feat/T-030-ticket-new-writes-unsanitised-input-into-frontmatter-newline-injection | [] |

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

## TO DO (impact order, per child)

### pickle

**Parked: T-009, T-010, T-016.** Real but explicitly **unscheduled** — nothing is blocked on
them and no user has asked. Do not pick one up without a demand signal. Parked status is
recorded in each ticket's Description and History (the board's grade cells are rewritten from
frontmatter by `pickle board sync`, so it cannot be encoded there). Triage 2026-07-26.

**Merge candidates** (proposed 2026-07-26, not yet executed): audit hardening + cell escaping
(T-027, T-028, T-033, T-034, T-014·2) · marker-block freshness (T-020, T-021) · one-predicate
refactors (T-015, T-017, T-032) · test harness + cli coverage (T-031, T-012) · surface polish
(T-013, T-014·1, T-019, T-023).

| id | title | impact | complexity | cost | depends-on |
|---|---|---|---|---|---|
| T-036 | review protocol spawns unbounded follow-up tickets; add inline-fix, note-and-close and backlog-cap valves | high | medium | M | [] |
| T-026 | upgrade refuses legal pickle.toml files and misdiagnoses why | high | medium | M | [] |
| T-037 | board sync silently deletes hand-written prose from BOARD.md sections | high | medium | M | [] |
| T-022 | skill payload states commit policy, branch prefix and WIP limits unconditionally | medium | low | S | [] |
| T-012 | harden test coverage + TOML-safe render (config, project, board audit) | medium | low | S-M | [T-001, T-002, T-003] |
| T-017 | unify marker-pair detection + dry-run fidelity | medium | low | S | [] |
| T-020 | doctor: detect AGENTS.md marker-block drift | medium | low | S | [] |
| T-021 | project add\|remove leave the AGENTS.md marker block stale | medium | low | S | [] |
| T-031 | harden the internal/cli test harness (captureStdout stdout restore + pipe lifecycle, TestMain sandbox lifecycle) | medium | low | S | [] |
| T-033 | board audit: flag duplicate frontmatter keys | medium | low | S | [] |
| T-034 | board audit: flag table rows with the wrong cell count; harden AddTODORow insert point | medium | low | S | [] |
| T-013 | install polish (marker spacing, summary labels, cli tests, --agent) | low | low | S | [T-004] |
| T-014 | board-row and move polish (WIP counts, cell escaping, subgroup spacing, atomicity) | low | low | S | [T-007] |
| T-015 | consolidate board status-heading matching and fill sync test gaps | low | low | S | [] |
| T-019 | README accuracy polish (prose duplicates command table, phased-plan tagging) | low | low | S | [] |
| T-023 | board branch column is derived from the filename slug, not the ticket's real branch | low | low | S | [] |
| T-027 | audit: flag depends-on entries that reference the ticket itself | low | low | S | [] |
| T-028 | guard TEMPLATE.md frontmatter against audit requiredKeys | low | low | S | [] |
| T-032 | unify the test payload-root idiom into one CWD-independent helper | low | low | S | [] |
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

## DROPPED

### pickle

| id | title | reason |
|---|---|---|
| T-035 | repair the corrupted T-021 board row (unescaped pipe in title) | board row repaired inline during triage; ticket overhead exceeded the one-character fix |
| T-025 | backfill true historical spawned-by lineage from existing source: lines | lineage archaeology with no consumer; source: History lines already carry provenance |

---

## Dependency chain (hard `depends-on:`, human-approved 2026-07-23)

- **T-001** (config/registry) → **T-002**, **T-003**, **T-004**.
- **T-002** (audit) → **T-007**, **T-008**.
- **T-003** (ticket new) → **T-012** (hardening).
- **T-004** (install) → **T-005**, **T-006**, **T-009**, **T-010**, **T-013**.

## Known soft couplings (cross-referenced in ticket Descriptions, not `depends-on`)

- **T-011** (distribution) wants the command set (P1–P3) essentially complete — narrative
  coupling only, no hard `depends-on`.
