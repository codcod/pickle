# Board

The maintained index of every ticket in `tickets/`, across all registered child-projects.
**Update this file on every ticket move** — see `tickets/README.md` (board rule). A move that
doesn't touch this file is a bug.

Within each status section, tickets are **sub-grouped by child-project** under a `### <child>`
heading; TO DO / READY are ordered by descending impact inside each group. This repo has one
child, **`pickle`** (the repo root; see `../pickle.toml`).

**WIP limits (per child-project):** `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1

Last updated: 2026-07-23

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

## TO DO (impact order, per child)

### pickle

| id | title | impact | complexity | cost | depends-on |
|---|---|---|---|---|---|
| T-001 | pickle.toml config model + project registry | high | medium | M | [] |
| T-002 | board audit engine | high | medium-high | M-L | [] |
| T-003 | ticket new (id allocation + template + board row) | high | medium | M | [] |
| T-004 | install (scaffold + skill install + marker injection + first child) | high | high | L | [] |
| T-007 | ticket move (state machine + per-child WIP + cross-child merge gate) | high | medium-high | M-L | [] |
| T-011 | distribution (goreleaser + Homebrew tap + releases + docs) | high | medium | M-L | [] |
| T-005 | doctor | medium | low-medium | S-M | [] |
| T-006 | upgrade + uninstall | medium | medium | M | [] |
| T-008 | board sync | medium | medium | M | [] |
| T-009 | opencode wiring | medium | medium | M | [] |
| T-010 | Pi guardrail scaffold | medium | medium | M | [] |

## DONE

### pickle

| id | title | merged |
|---|---|---|

## DROPPED

### pickle

| id | title | reason |
|---|---|---|

---

## Known soft couplings (cross-referenced in ticket Descriptions, not `depends-on`)

Suggested build-order chain (hard `depends-on:` to be confirmed with the human at refine time —
rules §3):

- **T-001** (config/registry) is the foundation consumed by **T-002**, **T-003**, **T-004**.
- **T-002** (audit) underpins **T-007** and **T-008**.
- **T-004** (install) underpins **T-005**, **T-006**, **T-009**, **T-010**.
- **T-011** (distribution) wants the command set (P1–P3) essentially complete.
