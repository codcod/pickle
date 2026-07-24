---
id: T-007
title: ticket move (state machine + per-child WIP + cross-child merge gate)
project: pickle
depends-on: [T-002]
impact: high
complexity: medium-high
cost: M-L
---

# T-007 — ticket move (state machine + per-child WIP + cross-child merge gate)

## Description

Implement `pickle ticket move T-NNN <status> --reason "<why>"` as one atomic operation:
relocate the ticket file between status directories, append the dated `## History` transition
line, and update the board row (correct section **and** child sub-group) — the three edits the
board rule requires, done together.

Enforce:

- the **state machine** — forward transitions plus the allowed backward/abort moves
  (`in-development → ready`, `ready → to-do`, `→ dropped`), with backward/abort **sign-off**
  rules;
- **per-child WIP limits** (counts only the ticket's own `project`);
- the **dependency gate** on pickup (`→ in-development`): every `depends-on:` target is in
  `6-done/` **and merged to the base of its own child-project's repo** (cross-child aware).

Reject illegal moves with a clear message. Needs T-001 and T-002. Phase P3.

> **Impact note (from the T-002 review, 2026-07-23):** the shared model now exists — build on
> `internal/ticket` (`Statuses` table + terminal set, `LoadAll`, `LastHistoryStatus`,
> `HasMergeLine`), `internal/board` (row model), and `config.Config` (per-child WIP). After a
> move, the result must pass `internal/audit.Audit` with zero errors — reuse it as the
> post-move self-check rather than re-deriving invariants.

## Implementation Plan

> Self-contained prompt. **Start by creating the branch `feat/T-007-ticket-move` inside the
> repo.** Build on the existing shared model (`internal/ticket`, `internal/board`,
> `internal/audit`, `internal/config`) — do not re-derive invariants. Mirror the
> `internal/install` shape: a testable core package + a thin cli wrapper.

### Goal

One command performs a ticket's status transition as the three edits the board rule demands,
together: **move the file** between `tickets/<status>/` dirs, **append a dated `## History`
line**, and **rewrite the board row** into the target section (correct child sub-group *and*
column shape). Reject illegal moves before touching anything; a completed move must leave
`internal/audit.Audit` reporting **zero errors**.

### 1. `internal/move` — the orchestrator (new package)

`func Move(root string, cfg *config.Config, id, token, reason string) (Result, error)` where
`Result{From, To, Path string; Warnings []string}`. Steps, in order, **validating fully before
any write**:

1. **Load** the tree with `ticket.LoadAll(root)`; index by id; find the ticket for `id`
   (error `ticket %s not found` if absent).
2. **Resolve the target status** from `token` via a new `ticket.StatusByToken(tok string)
   (Status, bool)`, case-insensitively accepting three forms: the dir name (`3-in-development`),
   the dir minus its number (`in-development`), and the display name lower-cased with spaces to
   hyphens (`in-development` from `IN DEVELOPMENT`). Error `unknown status %q` otherwise.
3. **State machine.** A transition table keyed by source `Status.Dir` to allowed target `Dir`s:
   - `1-to-do` to `2-ready`
   - `2-ready` to `3-in-development` *(pickup: dependency gate + WIP)*
   - `3-in-development` to `4-in-review`
   - `4-in-review` to `6-done` *(clean / non-blocking only)*
   - `4-in-review` to `5-rework` *(blocking findings)*
   - `5-rework` to `4-in-review` *(scoped re-review)*
   - backward/abort: `3-in-development` to `2-ready`, `2-ready` to `1-to-do`, and **any
     non-terminal to `7-dropped`**.
   No transitions leave a terminal status (`6-done`/`7-dropped`). Same-status is an error
   (`T-NNN is already in <status>`). Anything not in the table is
   `illegal transition <FROM> -> <TO>` (list the legal targets from the current status).
4. **Sign-off / reason.** `--reason` (non-empty) is **required** for every backward move
   (to `2-ready` from in-development, to `1-to-do` from ready), for the move to `5-rework`, and
   for the move to `7-dropped`; optional for forward moves. The human running/approving the
   command *is* the sign-off; `--reason` records why. Error `moving to <status> requires
   --reason` when missing.
5. **WIP gate** (moving into `3-in-development` or `4-in-review`): count existing tickets of
   the **same `project`** already in that dir; reject if `count + 1 > cfg.Project(p).WIP...`
   with `WIP: child %q already at its <dir> limit (%d)`. Counts only the ticket's own child.
6. **Dependency / cross-child merge gate** (pickup, the move to `3-in-development` only): for
   every `depends-on` target, require it is in `6-done` **and** `ticket.HasMergeLine(dep.Text)`
   is true (the `MERGED` History line is the cross-child proxy; we cannot inspect another
   child's git). Reject `cannot pick up: dependency %s is <state>` otherwise. This is stricter
   than the audit (which only *warns* on a missing merge line), which is right for an active
   pickup. The mechanical READY-gate content check is **out of scope** (a judgment call owned
   by refine/review), so `1-to-do` to `2-ready` enforces only legality.
7. **Apply** (only after every check passes):
   a. Append the History line via a helper: `- <YYYY-MM-DD> -- <FROM> -> <TO>: <reason>` using
      the em-dash and the real arrow character (omit `: <reason>` when empty). Insert it after
      the last bullet under `## History` (before the next `## ` heading, or EOF).
      **Sanitize the reason**: strip newlines and replace any arrow character with `->` so a
      reason can never be mis-read as the transition arrow (guards `LastHistoryStatus`; see
      T-012 item 6).
   b. Write the updated text to the current path, then `os.Rename` it to
      `tickets/<target.Dir>/<base>.md` (`os.MkdirAll` the target dir first).
   c. Rewrite the board row (section 2 below).
8. **Post-move self-check.** Reload cfg and run `audit.Audit(root, cfg)`; if it reports any
   errors, return them (the move is already applied, so surface loudly for inspection). Fold
   audit *warnings* into `Result.Warnings`.

### 2. `internal/board` -- status-aware row move

Add `func MoveRow(boardPath, statusName, child string, d RowData) error` and a
`RowData` struct carrying `ID, Title, Impact, Complexity, Cost, DependsOn, Branch, Merged,
Reason` strings. It (a) removes any existing row for `d.ID` from **any** section (match
`rowRE`, id-equal; leave now-empty sub-groups in place, matching the skeleton's
empty-placeholder convention), then (b) inserts a freshly rendered row into the
`## <statusName>` section under `### <child>` (creating the sub-group with the
section-appropriate header if absent).

Column shape is **section-specific** -- drive it from one table:

| section | columns |
|---|---|
| TO DO, READY | id, title, impact, complexity, cost, depends-on |
| IN DEVELOPMENT, IN REVIEW | id, title, branch, depends-on |
| REWORK | id, title, branch, open findings |
| DONE | id, title, merged |
| DROPPED | id, title, reason |

Render the header row, separator, and data row from that column list. TO DO/READY insert in
descending-impact order (reuse the `impactRank` logic already in `AddTODORow`); every other
section appends. **Refactor** the impact-ordered sub-group insert out of `AddTODORow` into a
shared helper that `MoveRow` also uses, and make `AddTODORow` a thin wrapper -- the existing
`TestAddTODORowImpactOrder`/`TestAddTODORowCreatesSubgroup` tests guard the refactor.

`internal/move` builds `RowData` from the ticket + config: `Title/Impact/Complexity/Cost` from
frontmatter, `DependsOn` reconstructed as `[T-00x, ...]` or `[]`, `Branch` as
`cfg.Project(p).BranchPrefix + id + "-" + slug` (e.g. `feat/T-007-ticket-move`), `Merged`
defaulting to `no -- publish-gated (branch <branch>)` when moving to DONE (the actual
`merged: yes` stamp stays a human bookkeeping step), and `Reason` (from `--reason`) for the
DROPPED and REWORK columns.

### 3. cli wrapper

Replace the `runTicketMove` stub in `internal/cli/ticket.go`:
`pickle ticket move <T-NNN> <status> [--reason "<why>"]`. Parse the two positional args + the
flag; `loadConfig()` for cwd/root (same helper `board audit`/`ticket new` use); call
`move.Move`; print `moved T-NNN: <FROM> -> <TO>` plus the new path, echo any warnings, and
surface errors. Exit codes: missing/extra args -> `exitUsage`; a rejected move (illegal
transition, WIP, dependency gate, missing reason, unknown status/ticket) -> `exitError`; a
post-move audit failure -> `exitError`.

### 4. Tests

- `internal/move`: table-driven `Move` over a temp `tickets/` tree (build it with
  `install.Run` against `t.TempDir()`, or hand-scaffold): a full forward walk
  (to-do -> ready -> in-development -> in-review -> done) leaves `audit.Audit` clean at each
  step; illegal transition rejected; missing `--reason` on drop/rework/backward rejected; WIP
  gate rejected when the child is already at its in-development limit; dependency gate rejected
  when a `depends-on` target is not done+merged, allowed once it is; the DONE row shows the
  publish-gated default; `LastHistoryStatus` of the moved file equals the new status.
- `internal/board`: extend the existing tests for `MoveRow` -- removes the old row, inserts the
  correctly-shaped row per section, creates a missing sub-group, and (TO DO/READY) preserves
  impact order.
- `internal/ticket`: `TestStatusByToken` across the three accepted forms + a bad token.

### 5. Acceptance test (run verbatim; also re-run at review)

Into a throwaway `/tmp` repo, using the freshly built binary:

1. `git init` + `pickle install --project demo --path .`; `pickle board audit` clean.
2. `pickle ticket new "Alpha" --project demo` (-> T-001, TO DO).
3. `pickle ticket move T-001 ready` -> file in `tickets/2-ready/`, a `TO DO -> READY` History
   line, row under READY/`### demo`; `board audit` clean.
4. `pickle ticket move T-001 in-development` -> file moved, branch column
   `feat/T-001-alpha`; `board audit` clean.
5. `pickle ticket new "Beta" --project demo` (-> T-002); `pickle ticket move T-002 ready` then
   `pickle ticket move T-002 in-development` -> **rejected** (WIP: demo already at its
   in-development limit); `board audit` still clean.
6. `pickle ticket move T-002 done` from READY -> **rejected** (illegal transition).
7. `pickle ticket move T-002 dropped` -> **rejected** (missing --reason); rerun with
   `--reason "superseded"` -> file in `tickets/7-dropped/`, DROPPED row with the reason;
   `board audit` clean.
8. `pickle ticket move T-001 in-review` then `pickle ticket move T-001 done` -> DONE row shows
   `no -- publish-gated ...`; `board audit` clean.
9. Dependency gate: `pickle ticket new "Gamma" --project demo` (T-003), hand-add
   `depends-on: [T-002]`, `pickle ticket move T-003 ready`, then `... in-development` ->
   **rejected** (dependency T-002 is dropped/not done+merged).

The killer check throughout: **`pickle board audit` reports 0 errors** after every applied
move.

### 6. Docs + wrap-up

- **README:** flip `ticket move` from `[P3]` to `[done: T-007]` and add a `## pickle ticket
  move` section (usage, the transition set, the sign-off/`--reason` rule, WIP + dependency
  gates, and the atomic file+History+board contract).
- Run `just lint`, `just test`, `just build`; paste the acceptance transcript into the summary.
- **Suggested commit (Conventional Commit, ticket id in brackets at the end):**
  `feat(cli): add ticket move (state machine + WIP + dependency gate) (T-007)`.
- Per the commit policy, `pickle` is publish-gated: commit locally on the branch, present the
  message + MR attributes for approval; the human merges.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P3)
- 2026-07-23 — TO DO → READY: Implementation Plan written (internal/move orchestrator + internal/board status-aware MoveRow + cli wrapper + tests + acceptance); meets the READY gate
- 2026-07-23 — READY → IN DEVELOPMENT: picked up, branch feat/T-007-ticket-move (dependency T-002 done+merged)
- 2026-07-24 — IN DEVELOPMENT → IN REVIEW
