---
id: T-002
title: board audit engine
project: pickle
depends-on: [T-001]
impact: high
complexity: medium-high
cost: M-L
---

# T-002 — board audit engine

## Description

Native, dependency-free reimplementation of the invariant checker as `pickle board audit` (the
keystone — a pure function over `tickets/` + `pickle.toml`, testable with fixture directories).

Verify every invariant:

- each ticket file sits in a known status dir and its filename is `T-NNN-<slug>.md`;
- frontmatter is complete (`id`, `title`, `project`, `depends-on`, `impact`, `complexity`,
  `cost`), grades are legal (single values or adjacent-pair ranges), the id matches the
  filename, and ids are unique across all status dirs (one global namespace);
- `project:` names a **registered child** (needs T-001);
- every `depends-on:` target exists;
- every ticket appears exactly once on `BOARD.md`, in the section **and** child sub-group
  matching its directory, and every board row has a backing file;
- **per-child** WIP limits hold;
- each ticket's last History transition matches the directory it lives in;
- no ticket is in `3-in-development/` while a dependency is not in `6-done/` (warning if a done
  dependency has no `merged` History line — checked against the dependency's **own** child repo,
  since dependencies may cross children).

Exit non-zero on any error; print each error with a path/line reference. Phase P1. Depends on
T-001; underpins T-007 and T-008.

> **Impact note (from the T-001 review, 2026-07-23):** T-001 introduced a ticket-frontmatter
> scanner in `internal/cli/project.go` (`ticketProject` reads `project:`; `liveTicketsTargeting`
> globs `tickets/{1-to-do..5-rework}/T-*.md`). Board audit parses the *same* frontmatter (`id`,
> `project:`, `depends-on:`, …) and enumerates the same status dirs — so extract a shared
> parser (e.g. an `internal/ticket` package: parse frontmatter + status-dir list) and refactor
> the T-001 remove-guard to reuse it, rather than duplicating the scan. Consume
> `config.Config`/`Project(name)` for the `project:`-is-registered check and the per-child WIP
> limits.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # child-project 'pickle' is the repo root
git checkout main
git checkout -b feat/T-002-board-audit-engine
```
Local WIP commits fine; **no push / no MR without user approval**.

### Prerequisite gate (hard)

`depends-on: [T-001]` — T-001 must be in `6-done/` **and merged to `main`**. Satisfied:
merged 2026-07-23 (cdad65e). Clean tree on `main`.

### Confirmed design decisions (do not deviate without asking)

1. **Three new internal packages, keeping the audit a pure, fixture-testable function:**
   - `internal/ticket` — the shared model: the ordered status-dir table
     (`1-to-do`…`7-dropped` ↔ `TO DO`…`DROPPED`, with the terminal set `6-done`/`7-dropped`),
     `ParseFrontmatter`, `LoadAll(root) ([]Ticket, []parseErr)`, `LastHistoryStatus(text)`,
     `HasMergeLine(text)`. `Ticket` carries id/num/slug/dir/path/frontmatter/text.
   - `internal/board` — `Parse(path)` into rows `{Status, Child, ID}` (walk `## ` section
     headers, longest-match-first so `IN DEVELOPMENT` beats `IN REVIEW`; track the current
     `### <child>` sub-group; skip the `T-NNN` template placeholder row).
   - `internal/audit` — `Audit(root string, cfg *config.Config) (errs, warns []string)`,
     pure over the filesystem + parsed config; **no printing, no `os.Exit`**.
2. **Reuse, don't duplicate (T-001 review impact note):** refactor the T-001 remove-guard
   (`liveTicketsTargeting`/`ticketProject` in `internal/cli/project.go`) to consume
   `internal/ticket` — delete the ad-hoc frontmatter scan.
3. **Invariants** = the full list in this ticket's Description. Grade legality ported from the
   reference set (`impact`: low/medium/high/critical + adjacent pairs; `complexity`:
   low/medium/high + adjacent pairs; `cost`: S/M/L/XL + adjacent pairs). **`project` is a
   required frontmatter key** and must name a **registered child** (`cfg.Project`); each board
   row's `### <child>` sub-group must match the ticket's `project:`.
4. **WIP limits are per-child, from `pickle.toml`** (`wip_in_development`/`wip_in_review` on each
   `[[project]]`), counted per child — **not** from a board text line.
5. **In-development dependency gate:** a ticket in `3-in-development/` must have every
   `depends-on:` target in `6-done/`; **warn** (not error) if a done dependency lacks a
   `merged` History line — checked in the dependency's **own** child (for the self-hosted
   `pickle` child that is this repo). Errors vs. warnings: an unmet invariant is an **error**;
   soft/advisory conditions are **warnings**.
6. **CLI contract:** `pickle board audit` prints `WARNING:`/`ERROR:` lines then a
   `board audit: N tickets, E error(s), W warning(s)` summary; exit `0` when `E==0` (warnings
   allowed), `exitError` (1) otherwise.
7. **A missing empty status dir is not an error — treat it as empty** (diverges from the
   reference checker, which errored). Git does not track empty directories, so a status dir
   legitimately vanishes when its last ticket moves out (observed 2026-07-23: `2-ready/`
   disappeared after emptying). `internal/ticket.LoadAll` must tolerate an absent status dir.
   (Keeping the dirs present via `.gitkeep` is an *install* concern, deferred to T-004.)

### Tasks

1. **`internal/ticket`** — model + parsing helpers above, with unit tests.
2. **`internal/board`** — `Parse` + row model, with unit tests.
3. **`internal/audit`** — `Audit` implementing every Description invariant over
   `internal/ticket` + `internal/board` + `config.Config`.
4. **`internal/cli/board.go`** — implement `runBoardAudit` (find+load config via the existing
   `loadConfig`, call `audit.Audit`, print, exit); **refactor `project.go`** remove-guard onto
   `internal/ticket`.
5. **Fixtures + tests** — `internal/audit/testdata/<case>/` mini trees (a clean `pickle.toml`
   + `tickets/`), one clean and several broken (bad filename, missing/extra frontmatter,
   illegal grade, duplicate id, dangling `depends-on`, unregistered `project`, board
   row/dir/child mismatch, WIP breach, History↔dir mismatch, in-dev dep not done). Table-driven:
   assert error counts + that a signature substring appears.
6. **Docs** — `README.md`: mark `board audit` implemented and summarise what it checks.

### Acceptance test

```
cd /Users/codcod/Projects/private/pickle
just lint && just test && just build

# dogfood: pickle audits its own board — must be clean
./pickle board audit ; test $? -eq 0

# a deliberately broken tree must fail
rm -rf /tmp/pickle-audit && mkdir -p /tmp/pickle-audit/tickets/1-to-do && cp pickle.toml /tmp/pickle-audit/
printf -- '---\nid: T-999\ntitle: broken\nproject: nope\ndepends-on: [T-404]\nimpact: banana\ncomplexity: medium\ncost: M\n---\n' > /tmp/pickle-audit/tickets/1-to-do/T-999-broken.md
# minimal BOARD.md so board parse runs
printf '# Board\n## TO DO\n### pickle\n' > /tmp/pickle-audit/tickets/BOARD.md
for d in 2-ready 3-in-development 4-in-review 5-rework 6-done 7-dropped; do mkdir -p /tmp/pickle-audit/tickets/$d; done
( cd /tmp/pickle-audit && /Users/codcod/Projects/private/pickle/pickle board audit ; test $? -eq 1 )
```
Expected: real board audits **0 errors, exit 0**; the broken tree exits **1** and reports the
unregistered `project`, the illegal `impact`, the dangling `depends-on`, and the missing board
row.

### Docs update (mandatory)

`README.md` — `board audit` marked implemented + the invariant summary. No `docs/` book yet.

### Finish (mandatory)

1. Acceptance test green; `just lint`/`test`/`build` clean.
2. README updated.
3. Summary of files + decisions.
4. Suggested commit: `feat(audit): implement native board audit (T-002)`.
5. Commit locally on the branch; **do not push / open MR without approval**.

## Review

**Reviewer verdict: PASS — no blocking findings.** Moved to `6-done/`; non-blocking test-
coverage gaps folded into the existing **T-012** (broadened). Generic protocol only (no
addenda configured); prerequisite T-001 is done **and merged** (cdad65e), so the dependency
gate is satisfied.

### Audit summary

- **Implementation (step 2) — all tasks & criteria MET.** Acceptance test re-run **verbatim**
  on branch `feat/T-002-board-audit-engine`: `just lint`/`test`/`build` clean; **dogfood**
  `pickle board audit` on pickle's own board = **12 tickets, 0 errors, 0 warnings, exit 0**;
  the broken fixture exits **1** reporting the unregistered `project`, illegal `impact`,
  dangling `depends-on`, and missing board row (plus an expected "no History" warning, since
  that fixture body omits `## History`). Task 1 `internal/ticket` (87.6% cover), Task 2
  `internal/board` (96.0%), Task 3 `internal/audit` (87.5%), Task 4 cli wiring + **remove-guard
  refactored onto `internal/ticket`** (verified no `ticketProject`/`bufio` remnants — the
  duplicate scan is gone, honouring decision 2 / the T-001 impact note), Task 6 README all
  present. Decision 7 (tolerate vanished-empty dirs) verified by a `ticket` unit test.
- **Quality (step 3):** idiomatic; deterministic output (errors/warnings sorted); `-race`
  clean. Gap: the `internal/cli` layer is 29.5% — `runBoardAudit` is a thin print/exit wrapper
  and the `project` commands remain uncovered (→ N1).
- **Consistency (step 4):** the three new packages compose cleanly (`audit` over
  `ticket`+`board`+`config`); README invariant list matches the implementation; no contradictions
  or stale refs. Deviation D1 below.
- **Docs (step 4a):** README `board audit` section accurate and complete; no `docs/` book yet.

### Findings

| severity | description | evidence | disposition |
|---|---|---|---|
| non-blocking (N1) | cli-layer coverage thin (29.5%): no tests for `runBoardAudit` or the `project` commands | `go test -cover ./internal/cli` | → **T-012** (broadened) |
| non-blocking (N2) | `internal/audit` has no test for the board **child-mismatch** (`row.Child != project`) or **duplicate board row** paths | `audit_test.go` cases | → **T-012** (broadened) |
| note (D1) | Task 5 named `internal/audit/testdata/<case>/` fixture dirs; implemented instead as programmatic temp-tree fixtures (table-driven, clean + 12 broken cases). Same intent, no committed fixture sprawl — accepted, no action | `internal/audit/audit_test.go` | accepted |
| trivial (patched) | dead `emptyFS` type in `cli_test.go` (skeleton leftover, never instantiated) | `internal/cli/cli_test.go` | removed in this review |

N1+N2 are the same non-blocking test-hardening class as T-012's existing scope, so they were
folded into **T-012** (now depends on T-002) rather than opening near-duplicate rows.

### Checklist

- [x] Implementation audit — acceptance test re-run verbatim, tasks & criteria verified (step 2)
- [x] Quality audit — idiomatic, deterministic, `-race` clean (step 3)
- [x] Consistency audit — packages compose; duplicate scan removed; README matches (step 4)
- [x] Documentation audit — README `board audit` section accurate; no docs book (step 4a)
- [x] Findings classified; N1/N2 folded into T-012; trivial dead code patched (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] `BOARD.md` updated; T-012 broadened; dependents (T-007/T-008) noted (step 7)
- [x] Impact sweep — T-007/T-008 build on the new packages (step 8)
- [x] Summary + commit message & MR attributes presented for approval; bookkeeping per policy; next-ticket suggestion (step 9)

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1)
- 2026-07-23 — TO DO → READY: implementation plan complete (READY gate met); prerequisite T-001 done+merged
- 2026-07-23 — READY → IN DEVELOPMENT: picked up, branch feat/T-002-board-audit-engine (applicability gate clean)
- 2026-07-23 — IN DEVELOPMENT → IN REVIEW: acceptance test green (internal/ticket+board+audit; dogfood audit 0 errors on own board; broken-tree exits 1)
- 2026-07-23 — IN REVIEW → DONE: review PASS, no blocking findings; N1/N2 folded into T-012; acceptance re-run verbatim (branch not yet merged — publish-gated)
