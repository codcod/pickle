---
id: T-091
title: a bookkeeping commit can stage a ticket move's add without its delete: move prints only the new path
project: pickle
depends-on: []
spawned-by: [T-081]
impact: medium
complexity: low
cost: S
---

# T-091 — a bookkeeping commit can stage a ticket move's add without its delete: move prints only the new path

## Outcome

After this ships, `pickle ticket move` and `pickle ticket new` hand you a ready-to-paste
`git add` naming **every** path they wrote — for a move, both halves of the rename plus the
board — so a bookkeeping commit made with the explicit pathspecs rules §0 mandates can no longer
land the add without the delete, leaving a duplicate ticket id in git history that the
worktree-based `board audit` cannot see.

## Description

Rules §0 requires bookkeeping commits to use **explicit pathspecs** (`git add <paths>`, never
`git add -A`/`.`), so that deliberately-untracked material is never swept in. `pickle ticket move`
is a **rename**: it writes the ticket at its new status path and removes the old one
(`internal/move/move.go:131,134`). But `move.Result` carries only the new path
(`Path string // new path, relative to root`, `move.go:27`) and the CLI prints only that
(`internal/cli/ticket.go:66`: `moved T-081: IN REVIEW → DONE  (tickets/6-done/T-081-…md)`).

So the operator is handed exactly one of the two paths a complete rename needs, and must supply
the other from memory. Nothing then objects: `git add` with the paths you *did* name succeeds,
and the commit lands with the add and without the delete.

**Deliberately not the cause: any awkwardness in git.** `git add <old path>` stages a deletion
perfectly well when the file is gone from the worktree but still in the index — verified on git
2.55.0: exit 0, `D a/f.md` staged. (An earlier draft of this ticket claimed it refuses; that
reading came from a `fatal: pathspec … did not match any files` produced by naming a path already
removed with `git rm`, which is a different situation.) The fix therefore needs no new git idiom
and no change to §0's explicit-pathspec rule: **naming the old path is sufficient, and the only
reason it goes unnamed is that the tool never prints it.**

**Observed, 2026-08-11 (T-081's own review — this ticket's `spawned-by`).** The reviewer moved
T-081 to `6-done/` with `pickle ticket move`, then committed with explicit pathspecs naming the
added file and the board, but not the deleted `tickets/4-in-review/T-081-…md`. The commit
(`ba01f41`) kept the old copy tracked. Nothing complained: the **worktree** was correct, so
`pickle board audit` reported `90 tickets, 0 error(s), 0 warning(s)`. The corruption existed only
in git history, and surfaced two steps later, when merging PR #30 restored the stale file into the
worktree and audit finally reported `duplicate id — also at 4-in-review/T-081-…md`. A fresh clone
would have shown the same. Fixed in `7e3dbb2`.

**The underlying gap is that `board audit` audits the worktree, not the commit.** Every invariant
it checks — unique ids, status-matches-history, the board matching the tickets — can hold in the
working tree while being broken in `HEAD`. That is fine for the flow's normal use, and this ticket
does not propose making the audit git-aware in general; it proposes closing the one hole the
flow's own instructions open, at the cheapest point.

**Shape chosen at refinement (2026-08-13): shape 1 — make the writing commands name every path
they wrote.** `move.Result` gains `OldPath`, and both `ticket move` and `ticket new` print the
full stage set as a ready-to-paste `git add` line. It lands the fix at the moment the mistake is
made, needs no new git idiom (see above), and touches one command's output rather than the hook's
staged-path machinery. `ticket new` is included because it has the same defect at lower stakes: it
prints one path (`internal/cli/ticket.go:149`) and also rewrites the board, so its stage set is
under-reported too — the consequence there is a stale board, not a duplicate id.

The two rejected shapes, and why:

2. **Have the `pre-commit` hook catch it.** T-057's hook already inspects `git diff --cached` for
   `tickets/` paths (`internal/hook/hook.go:352–408`) to enforce branch placement, so the staged
   file list is already in its hands: a staged ticket add whose id also exists at another status
   path in `HEAD`, with no corresponding staged delete, is a mechanical check on data it has
   already read. **Dispositioned note-and-close**, not dropped on its merits: it is a genuine
   second line of defence, but it needs staged-path plumbing that **T-082** will own, and building
   it here would duplicate that. Whoever implements T-082 should read this bullet and decide
   whether the check comes along for the ride.
3. **An `--staged`/`HEAD` audit mode** — rejected: it duplicates the tree walk against a git
   object source for one failure mode.

Soft coupling: **T-082** (pre-push hook refusing a feature-branch push carrying `tickets/` paths)
is the same *family* — git hygiene around bookkeeping — but a different failure: T-082 catches
bookkeeping on the **wrong branch**, this catches an **incomplete** bookkeeping commit on the
right one. If both land, shape 2 above and T-082's guard are two checks in one hook and should be
built to share its staged-path plumbing; neither blocks the other.

Soft coupling: **T-071** (harden the PATH probe) and **T-050** (pi staging gate) both touch hook
behaviour; if shape 2 wins, sequence with them rather than merging.

Soft coupling: **T-092** (detect an unfinalized merge) is the other half of the same incident that
spawned this ticket — that ticket makes the omission *detectable* after the fact (generalising the
DONE-without-a-merge-line audit check, and running `board audit` in CI — both shipped in T-092,
reviewed 2026-08-13, so CI now audits the board on every push and PR); this ticket fixes the
staging mistake at its source (`move` naming both paths). Neither blocks the other.

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-091-move-names-both-paths`, created in the `pickle` child-project's repo (path `.`)
before any change. Slug shortened from the filename's, per the precedent set by
`feat/T-081-gate-table` and `feat/T-092-unfinalized-merge-audit`. Local WIP commits are fine;
**no push and no MR without explicit user approval**, and merging is always the human's. Because
this child is root-path (`path = "."`), tidy the WIP commits by interactive rebase into a small
number of atomic commits and **keep that history** on merge (rules §0) rather than squashing.

Bookkeeping (this ticket file + `BOARD.md`) is committed on `main`, never on this branch — and
this ticket, of all tickets, is the one to get that right.

### Prerequisite gate (hard)

None. `move.Result` and both CLI print sites already ship; no ticket needs to land first. T-082
is a soft coupling only (see Description) and may land in either order.

### Confirmed design decisions (do not deviate without asking)

1. **Shape 1 only.** Add `OldPath` to `move.Result` and print the full stage set from both
   `ticket move` and `ticket new`. Do **not** add a hook check (shape 2 — note-and-closed to
   T-082's staged-path plumbing) and do **not** add a `--staged`/`HEAD` audit mode (shape 3,
   rejected). See the Description.
2. **The first output line stays byte-identical.** `moved %s: %s → %s  (%s)\n` and
   `created %s  (%s)\n` keep their exact current form; the new information is added as
   **additional indented lines** below, in the two-space style `move`'s warnings already use
   (`internal/cli/ticket.go:67-69`). Anything reading the first line keeps working.
3. **Exact new output**, for `ticket move`:

   ```
   moved T-081: IN REVIEW → DONE  (tickets/6-done/T-081-….md)
     removed  tickets/4-in-review/T-081-….md
     stage:   git add tickets/6-done/T-081-….md tickets/4-in-review/T-081-….md tickets/BOARD.md
   ```

   and for `ticket new`:

   ```
   created T-099  (tickets/1-to-do/T-099-….md)
     stage:   git add tickets/1-to-do/T-099-….md tickets/BOARD.md
   ```

   `ticket new` has no `removed` line — it removes nothing. Order within the `git add`
   argument list is **new path, then old path (when there is one), then `tickets/BOARD.md`** —
   fixed, so the line is diffable across runs and testable as a literal.
4. **The `stage:` line is printed unconditionally** — no git-repo probe, no
   "am I inside a work tree" check. The flow assumes git (rules §0), the line is advice rather
   than an action, and a probe would give `internal/cli/ticket.go` a git dependency it does not
   have today. A user outside a repo ignores one line.
5. **Paths are the same repo-root-relative strings already printed** (`move`'s `newRel`,
   `ticket new`'s `rel`) — forward slashes via `filepath.Join`, no quoting, no shell escaping.
   Ticket filenames come from `ticket.Slugify`, which cannot emit a space or a shell
   metacharacter, so the line is paste-safe by construction. Do not add quoting "just in case":
   it would make the common line uglier to defend against input that cannot occur.
6. **`tickets/BOARD.md` is a literal, not a computed path.** Both commands call
   `board.Regenerate`, which always writes exactly that file; deriving it through a new helper
   buys nothing.
7. **Nothing changes about what is written to disk.** This ticket is output-only: no new file
   operation, no change to `move`'s write-then-remove ordering (D7/T-014·4), no change to the
   post-move audit self-check.

### Tasks

#### Task 1 — `move.Result` carries the path it removed

In `internal/move/move.go`:

- Add `OldPath string // previous path, relative to root (the file this move removed)` to
  `Result` (next to `Path`, line 27), and keep `Path`'s comment as-is.
- Set it in the apply block: the old relative path is `t.Path` made relative to `root` — compute
  it as `filepath.Join("tickets", from.Dir, t.Base()+".md")`, mirroring how `newRel` is built at
  line 128, rather than string-trimming `t.Path`. Assign it alongside `res.Path = newRel`
  (line 137), **after** the remove succeeds, so a `Result` never claims to have removed a file
  it did not.

#### Task 2 — one helper renders the stage line

In `internal/cli/ticket.go`, add an unexported helper next to the two run functions:

```go
// stageLine renders the ready-to-paste `git add` for the paths a writing
// command just touched — rules §0 requires bookkeeping commits to name explicit
// pathspecs, and naming them from memory is how a rename lands add-without-delete
// (T-091). Order is fixed: new path, removed path (if any), then the board.
func stageLine(newPath, oldPath string) string
```

It returns `"git add " + strings.Join(paths, " ")`, skipping `oldPath` when empty, and always
appending `filepath.Join("tickets", "BOARD.md")`.

#### Task 3 — both commands print it

- `runTicketMove` (`internal/cli/ticket.go:66`): after the existing `moved …` line and **before**
  the warnings loop, print `  removed  %s\n` (`res.OldPath`) and `  stage:   %s\n`
  (`stageLine(res.Path, res.OldPath)`). Warnings stay last — they are the thing a reader must not
  scroll past.
- `runTicketNew` (`internal/cli/ticket.go:149`): after the existing `created …` line, print
  `  stage:   %s\n` (`stageLine(rel, "")`).

#### Task 4 — tests

- `internal/move/move_test.go`: assert a completed move's `Result.OldPath` is the source status
  path (`tickets/<from-dir>/<base>.md`) and that the file at that path is gone — the pairing is
  the point.
- `internal/cli/cli_test.go`: two `captureStdout` tests, `TestTicketMovePrintsStageLine` and
  `TestTicketNewPrintsStageLine`, asserting the literal `git add …` line including argument
  order (decision 3) and that `ticket new`'s output has **no** `removed` line. Also assert the
  first line still matches its current form verbatim (decision 2) — that is the regression that
  catches someone "tidying" the summary line later.
- A unit test for `stageLine` itself is not required; the two CLI tests cover both arities.

### Acceptance test

From the repo root on the feature branch:

```
just build && just test && just lint && just docs-check
```

All four green. Then reproduce the original failure's *fix* in a **throwaway directory — never
against this repo** (self-modify policy, `AGENTS.md`):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q .
./pk install --project demo          # --project is a child *name*; --path defaults to "."
./pk ticket new "stage line fixture" --project demo
```

Expected: stdout's second line is exactly
`  stage:   git add tickets/1-to-do/T-001-stage-line-fixture.md tickets/BOARD.md`.

Then exercise the rename that started this ticket:

```
git add tickets/1-to-do/T-001-stage-line-fixture.md tickets/BOARD.md && git commit -qm "board: T-001 created"
./pk ticket move T-001 ready --reason "fixture"
```

Expected output:

```
moved T-001: TO DO → READY  (tickets/2-ready/T-001-stage-line-fixture.md)
  removed  tickets/1-to-do/T-001-stage-line-fixture.md
  stage:   git add tickets/2-ready/T-001-stage-line-fixture.md tickets/1-to-do/T-001-stage-line-fixture.md tickets/BOARD.md
```

(The `ready` move will refuse on the READY gate unless the fixture's plan is filled — if so, use
`./pk ticket move T-001 dropped --reason "fixture"`, which is a legal move from TO DO and
exercises the identical print path.)

Finally, prove the printed line is *sufficient* — the whole point of the ticket: paste it
verbatim, then confirm the index carries **both** halves of the rename:

```
git add tickets/2-ready/T-001-stage-line-fixture.md tickets/1-to-do/T-001-stage-line-fixture.md tickets/BOARD.md
git diff --cached --name-status
```

Expected: an `A`/`R` row for the new path **and** a `D` row for `tickets/1-to-do/T-001-….md`
(git may render the pair as a single `R` rename row — either is correct; what must not appear is
an add with no delete). Commit, then `./pk board audit` against a fresh `git clone` of that
throwaway repo and expect `0 error(s)` — the clone is what made T-081's corruption visible, and
it is the check the worktree audit cannot perform.

### Docs update (mandatory when user-facing)

- **`docs/user-manual/cli-reference.adoc`**, `[#cmd-ticket-move]` (from line 467): document the
  two extra output lines and say plainly *why* the old path is printed — a move is a rename, and
  a bookkeeping commit made with rules §0's explicit pathspecs must stage both halves. Show the
  three-line output verbatim.
- **Same file**, the `pickle ticket new` section: document its `stage:` line the same way, noting
  the board is in the set because `ticket new` regenerates it.
- **`CHANGELOG.md`** — one entry under `[Unreleased]` → `### Added` (create the heading; the
  section currently has only `### Changed`) for both commands' stage line, naming T-091.
- **No skill-payload change.** Rules §0 already mandates explicit pathspecs; this ticket makes
  the tool supply them rather than adding new instruction. Do not amend
  `skill/resources/tickets-README.md`.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (`cli-reference.adoc`, `CHANGELOG.md`).
3. Write a summary of everything done — files touched, the fixed argument order, and the
   note-and-closed shape 2 handed to T-082.
4. Suggest a Conventional Commit message, e.g.:

   ```
   feat(cli): print the full stage set from ticket move and ticket new (T-091)

   A move is a rename, but only the new path was printed, so a bookkeeping
   commit made with explicit pathspecs could stage the add without the delete
   — leaving a duplicate ticket id in history that the worktree audit cannot
   see. Both writing commands now print a ready-to-paste `git add`.
   ```

5. **Tidy up before presenting** — root-path child: interactive-rebase the WIP commits into a
   small number of atomic, correctly typed commits and keep that history.
6. Commit locally on the ticket branch. Do **not** push or open an MR without user approval.
   Present the commit message; after approval, verify the remote base is not behind
   (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing), push, and open the merge request — merging is always the human's. Hand back.

## Review

2026-08-13 — reviewed on `feat/T-091-move-names-both-paths` (1 commit, `154795b`).
**Verdict: DONE — no blocking findings.**

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — run over the two changed prose files; 1 in-scope suggestion
      (the `[Unreleased]` CHANGELOG bullet) presented to the user, the rest concern prose this
      ticket did not touch (step 4b)
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval (step 9)

**Gates.** `just build`, `just test`, `just lint`, `just docs-check` all green on the branch;
`pickle board audit` on this repo reports `97 tickets, 0 error(s), 0 warning(s)` and
`pickle changelog check` reports `no candidates`.

**Tasks 1–4 all met.** `move.Result.OldPath` is set from `filepath.Join("tickets", from.Dir,
t.Base()+".md")` **after** the remove succeeds (`internal/move/move.go:28,135-140`), exactly as
task 1 required; `stageLine` renders the fixed order new → old → board
(`internal/cli/ticket.go:76-84`); both commands print it (`:67-68`, `:165`) with the warnings
loop left last; the four tests exist and pass (`TestMoveOldPathIsSourceStatusPath`,
`TestTicketMovePrintsStageLine`, `TestTicketNewPrintsStageLine`, plus the byte-identical
first-line assertions decision 2 asked for). All seven confirmed decisions honoured — no hook
check, no audit mode, no git probe, no quoting, `tickets/BOARD.md` a literal, and nothing
changed about what is written to disk.

**Behaviour verified live**, in a throwaway install (`mktemp -d` + copied binary, per the
self-modify policy), reproducing the plan's acceptance test verbatim:
`ticket new` printed `  stage:   git add tickets/1-to-do/T-001-stage-line-fixture.md
tickets/BOARD.md`; the `dropped` move printed the three expected lines including
`  removed  tickets/1-to-do/T-001-stage-line-fixture.md`; pasting the printed `git add`
verbatim staged the rename as `R093 tickets/1-to-do/… → tickets/7-dropped/…` plus `M
tickets/BOARD.md` — an add with no delete is exactly what could not be produced; and a fresh
`git clone` of that repo audits `1 tickets, 0 error(s), 0 warning(s)`, which is the check the
worktree audit could not perform and the one T-081's corruption failed.

| id | severity | disposition | finding | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | noted | `stageLine` guards `oldPath == ""` but the caller prints `  removed  %s` unconditionally, so the two sites disagree about whether an empty `OldPath` is possible | `internal/cli/ticket.go:67` vs `:78-81` | unreachable today (`move.Move` sets `OldPath` on every success and returns an error otherwise), so the guard is the redundant half, not the print; leave both |
| N2 | non-blocking | noted | `pickle board sync` also writes `tickets/BOARD.md` and prints no `stage:` line, so "every writing command names what it wrote" is not yet uniform | `internal/cli/board.go:64-66` | out of scope by decision 1, and materially different: `sync` writes **one** file and already names it (`board sync: rebuilt tickets/BOARD.md`), so there is no half-set to forget. Revisit only if a third writing command joins them |
| N3 | non-blocking | noted | The `git add` line is assembled with `filepath.Join`, so a Windows build would print backslashes, and the two CLI tests assert forward-slash literals | `internal/cli/ticket.go:82`, `internal/cli/cli_test.go:622,647` | not a defect today: `.goreleaser.yaml` targets darwin/linux only and CI is ubuntu-only, and `res.Path` has been `filepath.Join`-built since T-007 — the new line inherits the repo's existing convention rather than introducing a variance |
| N4 | non-blocking | noted | No test pins task 3's ordering requirement that warnings still print **after** the stage line — the move fixture is warning-free, so a reordering would go unnoticed | `internal/cli/cli_test.go:598-624` | a future test that moves a `6-done/` ticket without a merge line (guaranteed one warning) would pin it in three lines; below the promotion bar alone |
| N5 | non-blocking | noted | `concepts/project-structure.adoc` explains the failure this ticket fixes ("a ticket move whose board regeneration never got staged") without mentioning that the command now hands you the paths | `docs/user-manual/concepts/project-structure.adoc:113` | coverage is satisfied — `cli-reference.adoc` documents both commands' new output — and the concept page is deliberately about *why the board is committed*; add an xref only if that page is next edited |

**Disposition summary:** 5 non-blocking findings, 0 blocking — 0 fixed inline, 5 noted
(N1–N5), 0 folded, 0 new tickets.

**Impact sweep (step 8).** `T-082` is the only non-terminal ticket this change touches: the
rejected shape 2 (a `pre-commit` check for a staged add without its delete) was note-and-closed
*to T-082's staged-path plumbing*, but T-082's own file never said so — a hand-off recorded only
in the ticket handing it over. Its Description now carries the pointer, per the sweep's expected
outcome. `T-050` (pi staging gate) needs nothing: the printed line uses explicit pathspecs, which
that guardrail permits by construction, and was verified to pass it during this review.

## History

- 2026-08-12 — created (TO DO). source: pickle ticket new — filed from T-081's post-merge
  bookkeeping (`spawned-by: [T-081]`), where the failure was hit for real: `ba01f41` staged a
  ticket move's add without its delete, `board audit` stayed clean because it reads the worktree,
  and the duplicate id only surfaced when the merge of PR #30 restored the stale file. Graded
  against the backlog: impact `medium` (silent, and it corrupts git history rather than the
  worktree, so it is found late — but it is recoverable in one commit and has been seen once),
  complexity `low`, cost `S-M`, on the assumption refinement picks shape 1; shape 2 would raise
  both. Adjacent T-082 (`medium`/`medium`/`M`) is the same family but a distinct failure, and is
  deliberately not absorbed here
- 2026-08-13 — TO DO → READY: plan complete: shape 1 confirmed (move+new print the full stage set), cost collapsed to S
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
- 2026-08-13 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-13 — IN REVIEW → DONE: review passed: 5 non-blocking (all noted), 0 blocking
- 2026-08-13 — merged to main (PR #39, 7e60dce, https://github.com/codcod/pickle/commit/7e60dce); merge commit, 1 commit kept
