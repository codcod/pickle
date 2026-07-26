# Notes

Hand-written planning notes live here — triage records, parked-ticket notes, cross-ticket
decisions, dependency rationale. `BOARD.md` is generated from the ticket files (run
`pickle board sync`), so nothing hand-written survives there.

Migrated from BOARD.md's prose sections on 2026-07-26 (T-044). The old board banner warning
against running `pickle board sync` was **deleted, not migrated**: its three confirmed
data-loss behaviours (prose deletion, pipe un-escaping, branch-cell overwrite) are gone by
construction now that the board is a pure generated artifact — see T-044.

## Parked tickets (triage 2026-07-26)

**Parked: T-009, T-010, T-016.** Real but explicitly **unscheduled** — nothing is blocked on
them and no user has asked. Do not pick one up without a demand signal. Parked status is also
recorded in each ticket's Description and History.

## Epic merge (executed 2026-07-26 — 14 tickets → 5 epics)

The sources are in `7-dropped/` with reason `absorbed into T-0NN`; each opens with an ABSORBED
banner and keeps its full analysis, measurements and line references. They are the
authoritative detail; the epic is the refinable, reviewable unit. Do not re-file a source or
implement from it.

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

## Cross-epic decisions

**T-044 won the T-039-vs-T-044 design decision** (2026-07-26): the board becomes a generated
artifact; T-039 (harden the hand-maintained design) was dropped as superseded, and its
move-atomicity residue (T-014·4) is folded into T-044. Escape-vs-replace is settled by T-044's
one-way cell sanitisation — **T-043 item 5 defers to T-044**.
T-042 collides with T-044 (`internal/board`, `internal/sync`) and with T-043 (`cli_test.go`) —
sequence, do not run concurrently.

## Dependency chain (hard `depends-on:`, human-approved 2026-07-23)

- **T-001** (config/registry) → **T-002**, **T-003**, **T-004**.
- **T-002** (audit) → **T-007**, **T-008**.
- **T-003** (ticket new) → **T-012** (hardening).
- **T-004** (install) → **T-005**, **T-006**, **T-009**, **T-010**, **T-013**.

## Known soft couplings (cross-referenced in ticket Descriptions, not `depends-on`)

- **T-011** (distribution) wants the command set (P1–P3) essentially complete — narrative
  coupling only, no hard `depends-on`.
