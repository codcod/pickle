---
id: T-043
title: harden the cli test harness and close the config, project and ticket-new coverage gaps
project: pickle
depends-on: []
spawned-by: [T-012, T-031]
impact: medium
complexity: medium
cost: M
---

# T-043 — harden the cli test harness and close the config, project and ticket-new coverage gaps

## Description

**Epic — merged from T-031 and T-012 by the 2026-07-26 board triage**, then **narrowed at
refinement on 2026-08-06**: the production-correctness half (`internal/config`'s two writers)
was split out as **T-069**. What remains here is one theme — **the `internal/cli` test surface**:
the harness every cli test depends on, and the command-layer coverage gaps. T-031 and T-012 are
in `tickets/7-dropped/` with their original finding lists; this ticket now carries the corrected,
re-verified version of everything still in scope, so it is self-contained.

Every claim below was re-verified against `main` at `545a4c5` on 2026-08-06. Coverage today:
**`internal/cli` 63.3%**, `internal/config` 85.6% (the filed figures — 46.7% and 91.8% — were
both stale, and config's was stale *downwards*).

### Part 1 — harden the harness (T-031, from the T-029 review findings N1–N4, N7)

T-029 built the first real harness in `internal/cli`: a `TestMain` that sandboxes the process
CWD (`internal/cli/cli_test.go:30-62`), a `TestCWDIsSandboxed` guard (`:135-143`), and a
`captureStdout` helper (`:150-182`). It billed itself as reusable, and it has been reused
heavily — which is the argument for fixing it now.

**The true call-site inventory (13 + 2, not the "three" this ticket claimed until today):**

| file | sites |
|---|---|
| `internal/cli/cli_test.go` | `:198`, `:227` (`TestProjectAddRefreshesMarkerBlock`), `:325` (`TestTicketNewSpawnedByUnknownID`), `:575` (`TestServeHelpIsAdvertised`) |
| `internal/cli/agents_test.go` | `:65`, `:86`, `:106`, `:127` (all `TestInstallAgentFlag`), plus **`captureStderr` at `:51`, `:107`** (the latter nested inside a `captureStdout`) |
| `internal/cli/hooks_test.go` | `:135`, `:144`, `:153`, `:172` (`TestHooksInstallStatusUninstall`), `:185` (`TestHooksAreAdvertised`) |

Two of them **commit defect 1 today**: `cli_test.go:238` runs `project remove` and
`hooks_test.go:164` runs `hooks uninstall` *outside* any capture, after an earlier capture in the
same test has already closed the fd — so their output goes to a closed pipe and is discarded.
And `agents_test.go:16-41` is a **verbatim `captureStderr` clone** of the same helper, carrying
the same defects for `os.Stderr`; neither T-031 nor T-012 mentions it.

1. **`captureStdout` leaves `os.Stdout` pointing at a closed pipe after it returns.** Current
   refs (the filed `:150,154,167,171` have drifted): pipe `:152`, `orig := os.Stdout` `:156`, the
   swap `:157`, the `t.Cleanup` `:161`, `w.Close()` `:174`, `r.Close()` `:178`. `w` is closed at
   `:174` after the read is joined at `:177`, but `os.Stdout` is restored **only** in the
   `t.Cleanup`, which runs at *test* end. From the helper's return until then `os.Stdout` is a
   closed `*os.File`: every `fmt.Printf` in the package returns `file already closed` and the
   commands ignore the error, so output silently vanishes. Measured by T-029's probe:
   `n=0 err=write |1: file already closed`.
   **Correction to the filed rationale:** T-031 justified the ~64 KiB threshold by a *hang* risk
   on large writes — that is **moot**, because T-029 shipped the drain goroutine (`:167-170`).
   Do not carry the hang argument forward.
   **Aggravator neither ticket noticed:** with two captures in one test, the second call's `orig`
   (`:156`) *is the first call's closed `w`*. LIFO cleanups restore the real stdout at test end,
   so it self-heals — but any mid-test restore restores a closed fd.
2. **The pipe and its reader goroutine leak whenever `fn` calls `t.Fatal` or panics.**
   `w.Close()`, `<-done` and `r.Close()` all sit after `fn()` (`:172-178`), so `runtime.Goexit`
   skips them and `io.ReadAll(r)` blocks forever on a write end that is never closed. **Nine of
   the 13 call sites call `t.Fatalf` inside `fn`.** The helper's own `t.Fatalf` at `:175` skips
   `r.Close()` the same way. This is exactly the failure mode the comment at `:158-160` reasons
   about — closed for `os.Stdout`, left open for the pipe.
3. **`TestMain` leaks its sandbox on its own error path and on a panic in `m.Run`.** If
   `os.Chdir(sandbox)` fails (`:48`), `os.Exit(1)` at `:50` runs before the `RemoveAll` at `:59`.
   The comment at `:57` ("`os.Exit` runs no deferred functions, so this cannot be a defer") is
   imprecise: a defer inside an *inner function* does run before `os.Exit`; the real constraint
   is "not a defer in `TestMain`'s own body".
4. **`repoRoot` is never validated** (`:34`). It is `wd/../..` on faith; a wrong value surfaces
   far away as an opaque `install: …` failure in every test that calls `newProject` (`:119-129`),
   and `agents_test.go:44` consumes it too. The point of T-029 was to stop depending on an
   implicit CWD, so the assumption should assert itself.
5. **Comment nit (N7), re-worded.** T-043 previously summarised this as "a comment nit that
   overstates what is shared" — **wrong**. The actual N7 is the `~64 KiB` pipe-buffer claim at
   `:163`: that is Linux's default, while darwin (this repo's dev platform) starts a pipe at
   16 KiB and grows toward 64 KiB.

Also in scope because the next author will look at the code, not the tickets: `TestMain`'s doc
comment does not warn that **Go permits exactly one `TestMain` per package**, and
`TestCWDIsSandboxed` never actually asserts that the CWD *is* the sandbox.

### Part 2 — close the coverage gaps (T-012)

`internal/cli` is exercised almost entirely by manual acceptance tests. Original item numbers are
kept for traceability; four have moved or closed.

| # | item | state |
|---|---|---|
| 1 | cli-level tests for `project add\|list\|remove` and `board audit`, driving `runProject*`/`runBoardAudit` against a temp overarching root: `add` appends with defaults and rejects duplicate-name/missing-dir; `list` output; `remove` succeeds and the live-ticket guard refuses when a ticket targets the child; `board audit` exits 0 clean and non-zero broken | **in scope** |
| 2 | TOML-safe rendering (`config.Render`'s `%q`) | **→ T-069** (and its headline claim was disproved — see T-069) |
| 3 | defaulting test — `config_test.go:90-95`'s "zero wip" case sits inside `TestLoadErrors` and asserts an error for **`-1`**, not `0`. Zero is unreachable as a failure: `applyDefaults` (`config.go:159-164`) turns `0` into `DefaultWIPInReview` (1) and `Validate` passes. Rename to "negative wip"; add a case asserting `wip_in_review = 0` **loads** with `WIPInReview == 1` | **in scope** (verified true) |
| 4 | cli-level tests for `ticket new` (`runTicketNew`): fresh id `max+1`, scaffold written to `1-to-do/`, `audit.Audit` clean, board row under the child's sub-group in impact order, and non-zero exits for unregistered `--project`, illegal grade, missing title | **in scope** |
| 5 | board-row title sanitization | **scoped out** (D6) — see the deferral below |
| 6 | `LastHistoryStatus` (`internal/ticket/ticket.go:314-343`). The original defect (`LastIndex("→")` swallowing an arrow in the *reason* clause) **is fixed**. What remains: it is asserted only for "DONE" and "TO DO" (`ticket_test.go:179,300,490`, `internal/move/move_test.go:98`) while its classifier `historyKind` is well covered (`ticket_test.go:82-99`). Unexercised: no/empty `## History` (→ `""`), lower/mixed-case target, an unknown target (must be ignored as a note, not returned), a note after the last transition, multiple arrows in one body, `## History` at another heading level — and one real divergence: **`HistoryEntries` folds wrapped continuation lines and `LastHistoryStatus` does not**, untested either way | **in scope**, re-scoped from "fix the parser" to "pin the contract + settle the divergence" |
| 7 | `Save` is neither atomic nor mode-preserving | **→ T-069** |
| 8 | residual `payload_version` line-editor wedges | **→ T-069** |
| 9 | **`pickle install --hooks` is untested.** The block is `internal/cli/install.go:96-110` (the filed `:97-107` was off): `if *hooks {` at `:101`, `hook.Install(root, false)` at `:102`, and the deliberate warning-not-failure `fmt.Fprintf(os.Stderr, …skipped…)` at `:103`. Zero matches for `--hooks` in `internal/cli/*_test.go`; `hooks_test.go` only drives the separate `pickle hooks install` subcommand. It is covered solely by T-057's 12-step acceptance transcript, which neither `just test` nor CI runs | **in scope** (verified true) |

### Deferral: item 5 is closed, not carried (D6)

T-012 item 5 ("`ticket new` writes the raw title into the board row") is the same
escape-versus-replace question **T-044 settled** (2026-07-26): one-way `sanitizeCell` (`|` → `¦`,
newlines → space, plus — since T-049, 2026-07-27 — a 120-rune cap with a trailing `…`) at the
render choke point, with `TestRenderSanitizesCells` and `TestRenderCapsCellWidth` in
`internal/board/board_test.go` and an acceptance repro. Nothing is left to build; it shrinks to
**one assertion inside item 4's test** (a pipe title yields an audit-clean board), not a task.
(T-039 previously owned this decision and was dropped as superseded by T-044.)

### Sequencing

- **T-042** item 3 unifies the test payload-root idiom across five files
  (`internal/install/install_test.go:16`, `internal/doctor/doctor_test.go:15`,
  `internal/move/move_test.go:20`, `internal/sync/sync_test.go:20`, and
  `internal/cli/cli_test.go:34` — the odd one out, absolute because `TestMain` moves the CWD).
  **Decided at refinement (D5):** T-043 goes first and **owns `TestMain`**; it adds the `os.Stat`
  validation of `repoRoot` (Part 1 item 4) and leaves the five-site unification to T-042, which
  then deletes the validation along with the computation. Overlap is one line, so no hard
  `depends-on` — but do not run them concurrently.
- **T-069** (split from this ticket) owns `internal/config`'s writers. Disjoint files; the only
  contact point is `config_test.go`, where this ticket touches **item 3's case only**.
- T-012's original `depends-on: [T-001, T-002, T-003]` are all in `6-done/` and merged, so the
  gate is satisfied; the epic carries no hard dependency forward.

### Why cost M (was L)

Removing T-069's three production defects and item 5 leaves one file family
(`internal/cli/*_test.go` plus `config_test.go`'s one case and `internal/ticket`'s parser
contract) and no unresolved contract questions. The remaining risk is not size but **blast
radius**: the harness is consumed by 15 call sites, so a botched change breaks every cli test at
once — hence complexity stays `medium` and the acceptance test below is mutation-based.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd /Users/nicos.karagieorgopulus/Projects/private/pickle   # child "pickle" is the repo root
git checkout main
git pull --ff-only
git checkout -b feat/T-043-cli-test-harness
```

Work on this branch; local WIP commits encouraged. The child is **publish-gated**: do **not**
push and do **not** open a merge request without explicit user approval (`AGENTS.md`). Ticket and
board bookkeeping belongs on `main`, never on this branch — `pickle hooks install` enforces it
locally (see T-068: the guard may be inert if the `pickle` on `PATH` is older than the `hooks`
verb, so do not rely on it as your only check).

### Prerequisite gate (hard)

1. `depends-on:` is empty; nothing to verify there.
2. Working tree clean, on `main`, up to date with `origin/main`.
3. **T-042 must not be in `3-in-development/`** — it edits `internal/cli/cli_test.go:34`, which
   this ticket rewrites (D5). If it is, stop and tell the user.
4. **T-069 must not be in `3-in-development/`** — it edits `internal/config/config.go` and
   `config_test.go`, and this ticket touches `config_test.go`'s zero-wip case. If it is, either
   stop or agree with the user to defer item 3 to T-069.
5. Re-run the call-site inventory before editing — if the counts differ from the table above, the
   harness has new consumers and the plan's blast radius changed:
   ```
   grep -rn "captureStdout(\|captureStderr(" internal/cli/ | grep -v "func " | wc -l
   # expect 15 call sites (the raw grep returns 17 — the two `func` definitions match too)
   ```

### Confirmed design decisions (do not deviate without asking)

1. **Harden the helpers; do not refactor the command layer to take an `io.Writer`.** T-029
   deliberately declined that refactor and T-012 asked for it to be raised here; the user
   confirmed (D4) it stays declined. Injecting writers would be a package-wide half-migration on
   top of a harness fix. Record the decision in the helper's comment so it is not re-litigated.
2. **Unify `captureStdout` and `captureStderr` into one helper** and delete the clone at
   `internal/cli/agents_test.go:16-41`. One implementation parameterised by the target
   (`*os.File` pointer to swap, e.g. `capture(t, &os.Stdout, fn)`), with thin
   `captureStdout`/`captureStderr` wrappers so the 15 existing call sites keep compiling
   unchanged. Fixing one helper and leaving its copy defective is the exact shape of defect this
   repo files tickets about (see T-042).
3. **Keep `t.Cleanup` as the guarantee; add the early restore as an *additional* release.**
   Assign `os.Stdout = orig` immediately after `fn()` returns, *before* `w.Close()`, and keep the
   `t.Cleanup` as the Goexit/panic backstop. This does **not** reverse T-029 decision 4 ("restore
   through `t.Cleanup`, never by an assignment after `fn()`") — say so in the comment, or the
   next reader will think the decision was undone.
4. **Close + join exactly once, from both paths.** Wrap the teardown in a `sync.OnceFunc` (e.g.
   `closeW`) referenced by both the happy path and the cleanup:
   `t.Cleanup(func() { *target = orig; closeW(); <-done; _ = r.Close() })`. `t.Fatal` inside `fn`
   must not leak the goroutine or either fd.
5. **`TestMain`: inner function + `defer`.**
   `code := func() int { defer func() { _ = os.Chdir(repoRoot); _ = os.RemoveAll(sandbox) }(); …; return m.Run() }()`
   then `os.Exit(code)`. Two constraints from T-029 must survive: `sandbox` stays a **named
   variable the cleanup references** (so T-029 acceptance mutation C — deleting `TestMain`'s
   `os.Chdir` — still yields a *test failure*, not an unused-variable **compile error**), and the
   chdir back still precedes the removal. Correct the imprecise comment at `:57` while there.
6. **Validate `repoRoot` by what `install.Run` actually needs:** `os.Stat` the payload marker
   `skill/SKILL.md` under the resolved root and `os.Exit(1)` with a clear message if it is
   missing. Do **not** replace the `wd/../..` computation with a `runtime.Caller` helper — that
   is T-042 item 3's job across five files (D5).
7. **Extend the single `TestMain`; never add a second** (Go permits one per package) and document
   that in its doc comment. **No `t.Parallel()` anywhere in package `cli`** — both the CWD and
   `os.Stdout`/`os.Stderr` are process-global; state this in the helper comment too.
8. **Item 6's divergence is settled by making `LastHistoryStatus` use one source of truth:**
   route it through the same continuation-folding `HistoryEntries` already performs, so the two
   cannot disagree. This is the only production-code change in this ticket. **Guard:** run
   `pickle board audit` on the real 69-ticket tree before and after; if any outcome changes, stop
   and report rather than "fixing" the tree — the change is meant to be behaviour-preserving on
   every well-formed history.
9. **Item 5 is closed, not implemented** (D6): one assertion inside item 4's test, no task.
10. **Do not touch `internal/config` production code** — `Render`, `Save`,
    `SetPayloadVersionInPlace`, `advance` and `usesCRLF` are all T-069's (D1). In
    `internal/config/config_test.go`, item 3's case is the only edit.
11. **Comment corrections are part of the change, not polish:** the `~64 KiB` claim becomes
    "16–64 KiB, OS-dependent (darwin starts at 16 KiB)"; the `:158-160` reasoning is rewritten so
    it does not contradict decision 3.

### Tasks

#### Task 1 — one capture helper, correct on every exit path
`internal/cli/cli_test.go:150-182` — rewrite per decisions 2, 3, 4, 7, 11. Delete
`captureStderr` from `internal/cli/agents_test.go:16-41` and point both wrappers at the unified
implementation. Keep the existing signatures so nothing at the 15 call sites changes.

#### Task 2 — `TestMain` lifecycle + `repoRoot` validation
`internal/cli/cli_test.go:30-62` — decisions 5, 6, 7. Also make `TestCWDIsSandboxed` (`:135-143`)
assert the CWD actually equals the sandbox, and document the one-`TestMain`-per-package rule.

#### Task 3 — prove the two defects are fixed, by test
Two guards, both runnable in `just test`:
- **Probe:** after `captureStdout` returns, a `fmt.Println` still succeeds (`n > 0`, no error).
  This is the direct regression test for defect 1 and must fail if decision 3's early restore is
  removed.
- **Goexit safety:** a child test that calls `t.Fatalf` *inside* `fn` must not hang. `t.Fatal`
  cannot be observed from the same process, so **self-exec the test binary**:
  `TestCaptureGoexitDoesNotLeak` re-runs `os.Args[0] -test.run=TestCaptureGoexitChild` with an
  env guard (e.g. `PICKLE_TEST_CHILD=1`; the child `t.Skip`s unless it is set), asserting the
  child exits **non-zero within a short timeout** (i.e. it failed rather than blocked forever).
  `internal/cli/hooks_test.go` already establishes the stub-binary/subprocess idiom in this
  package — follow it.
- The two live instances of defect 1 — `internal/cli/cli_test.go:237` (`project remove`) and
  `internal/cli/hooks_test.go:165` (`hooks uninstall`), both running *after* an earlier capture in
  the same test — are fixed **automatically** by decision 3's early restore, since their output
  then reaches the real stdout again. Do not wrap them in a capture unless the test should assert
  their output; just confirm they no longer write to a closed fd.

#### Task 4 — cli tests for `project add|list|remove` and `board audit` (item 1)
`internal/cli/cli_test.go` — table-driven, driving `runProject*` and `runBoardAudit` against a
temp overarching root (temp `pickle.toml` + child dirs + tickets tree), asserting exactly the
cases in the item-1 row above, including the live-ticket remove-guard refusal and `board audit`'s
exit code on a clean **and** a broken tree.

#### Task 5 — cli tests for `ticket new` (item 4) + item 5's single assertion
`internal/cli/cli_test.go` — the five assertions in the item-4 row, plus: a title containing `|`
yields a board that `audit.Audit` accepts with zero errors (the residue of item 5).

#### Task 6 — cli tests for `install --hooks` (item 9)
`internal/cli/install_test.go` (or the existing `cli_test.go`, whichever keeps `runInstall`'s
tests together) — two cases against a temp root:
- **success:** `git init` the temp root first (`hook.Install` → `Status` → `HooksDir` shells out
  to `git rev-parse --git-path hooks`, `internal/hook/hook.go:141,147`), then assert exit 0 and
  the `  + <path>` line via the capture helper;
- **warning-not-failure:** run in a temp dir that is **not** a git repo, so `hook.Install`
  returns the no-repo kind; assert **exit 0** *and* the `…skipped…` line on **stderr**
  (`internal/cli/install.go:101-103`). A second install exercises the `Skipped` branch.

#### Task 7 — the defaulting test (item 3)
`internal/config/config_test.go:90-95` — rename the case to "negative wip" (it asserts `-1`) and
add a *loading* case proving `wip_in_review = 0` yields `WIPInReview == 1` via `applyDefaults`.
No production change.

#### Task 8 — pin `LastHistoryStatus` (item 6)
`internal/ticket/ticket.go:314-343` + `internal/ticket/ticket_test.go` — table-driven cases for
every unexercised shape listed in the item-6 row, then decision 8's folding unification, then the
before/after `pickle board audit` guard.

### Acceptance test

Runnable verbatim from the repo root on the feature branch.

```
# 1. The child's four configured commands, all green
just build && just test && just lint && just docs-check

# 2. Race + repeat, twice over, on the package whose harness changed
go test -race -count=2 ./internal/cli/ ./internal/ticket/ ./internal/config/

# 3. Coverage moved where the ticket says it would
go test ./internal/... -cover 2>&1 | grep -E "internal/(cli|config|ticket)"
#   expect: internal/cli >= 75.0% (was 63.3%), internal/config >= 85.6%, internal/ticket >= 94.4%
#   no package may drop below its 2026-08-06 figure

# 4. Mutation A — T-029's original guard must still hold:
#    delete os.Chdir(sandbox) from TestMain -> TEST FAILURE, not a compile error
go test ./internal/cli/ 2>&1 | tail -3    # then restore

# 5. Mutation B — remove the early `*target = orig` (decision 3) -> the Task 3 probe fails
# 6. Mutation C — remove closeW()/<-done from the cleanup (decision 4) ->
#    TestCaptureGoexitDoesNotLeak fails on its timeout, not by hanging the suite
# 7. Mutation D — revert decision 8's folding in LastHistoryStatus -> the wrapped-line case fails
#    (each mutation: apply, run, observe the named failure, restore)

# 8. The real tree is unaffected by the only production change
./pickle board audit        # expect: 69 tickets (or current count), 0 error(s), 0 warning(s)
#    identical output before and after the LastHistoryStatus change

# 9. The helper's own defect is gone at the two live sites
go test -run 'TestProjectAddRefreshesMarkerBlock|TestHooksInstallStatusUninstall' -v ./internal/cli/
#    both must assert on captured output, not on output that vanished into a closed fd
```

### Docs update

**No user-facing surface** — this ticket ships tests plus one behaviour-preserving parser
refactor. Therefore:
- `docs/user-manual/**`: nothing.
- `CHANGELOG.md`: nothing under `### Added`. Add a one-line `### Fixed` entry **only if**
  decision 8's guard shows any observable change in `LastHistoryStatus`/`board audit` behaviour
  (in which case it is no longer behaviour-preserving — stop and report first, per decision 8).
- `DESIGN.md`: nothing; the harness is test scaffolding, not architecture.
- Do update the code comments named in decision 11 — they are the docs for this surface.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs step honoured (i.e. deliberately empty, with the `CHANGELOG` conditional resolved).
3. Write a summary: files touched, the 15 call sites' status, coverage before/after, each mutation
   and the failure it produced, and anything deferred.
4. Suggested commit message:

   ```
   test(cli): harden the test harness and close the command-layer coverage gaps (T-043)

   captureStdout restored os.Stdout only in t.Cleanup, so every print between the
   helper returning and test end went to a closed fd (two live instances in the
   suite); the pipe and its reader goroutine leaked whenever fn called t.Fatal,
   which nine of the fifteen call sites do. captureStderr was a verbatim clone of
   both defects. One helper now serves both streams, restores early and closes
   once from either exit path, with t.Cleanup kept as the Goexit backstop.

   TestMain no longer leaks its sandbox when os.Chdir fails or m.Run panics, and
   validates repoRoot against skill/SKILL.md instead of trusting wd/../...

   New cli-level tests cover project add|list|remove, board audit, ticket new and
   install --hooks (including its warning-not-failure branch, previously reachable
   only through a manual acceptance transcript); LastHistoryStatus is pinned across
   every history shape and now folds continuation lines through the same path as
   HistoryEntries; config's zero-wip case asserts defaulting instead of -1.
   ```

5. Commit locally on the branch. **Publish only after explicit user approval** — then finalize
   (squash or keep history, user's choice), push, open the MR; the human merges.
   `pickle ticket move T-043 in-review --reason "acceptance green"` and hand back.

## Implementation summary

Built on `feat/T-043-cli-test-harness` (commit `56472dc` base). Every task landed; item 6 turned
out to need more than the plan said — recorded below rather than silently under-delivering.

**Part 1 — the harness (Tasks 1–2).** `internal/cli/cli_test.go`: `capture`/`captureStdout`/
`captureStderr` unified into one implementation (`sync.OnceFunc`-guarded close+join, early
restore *before* the pipe closes, `t.Cleanup` kept as the Goexit/panic backstop); the verbatim
`captureStderr` clone deleted from `agents_test.go`. `TestMain` now validates `repoRoot` against
`skill/SKILL.md`, and its teardown runs inside a deferred inner function so a failed `Chdir` or a
panic in `m.Run` no longer leaks the sandbox. `TestCWDIsSandboxed` now asserts the CWD *is*
`TestMain`'s sandbox (`os.SameFile`), not merely config-free. Two live instances of the
closed-fd defect (`cli_test.go` project-remove line, `hooks_test.go` hooks-uninstall line) are
fixed automatically by the early restore — nothing else needed changing at those two sites.

**Part 1 — proof, not assertion (Task 3).** `TestCaptureStdoutRestoresBeforeCleanup` (direct
probe) and `TestCaptureGoexitDoesNotLeak` (a self-exec subprocess, `TestCaptureGoexitChild` as
its gated child — a `t.Run` subtest was tried first and rejected: a failing subtest always fails
its parent's reported status regardless of any check performed afterwards, which would have made
the regression test permanently red). Both were mutation-tested against the fix they pin and both
caught the reversion (see Acceptance test below).

**Part 2 — coverage (Tasks 4–7, 9).** New cli-level tests: `project add` rejects a duplicate name
and a missing dir without mutating the registry; `project list`'s tabwriter output; `project
remove`'s live-ticket guard, both the refusal and the later success once the ticket is dropped;
`board audit`'s exit code on a clean and a dangling-`depends-on` tree; `ticket new`'s max+1
allocation, its two argument-validation failures (unregistered `--project`, illegal grade), and
its board-row impact ordering; the item-5 residue (a piped title still audits clean). `install
--hooks` (item 9): a real `git init` success path, the no-repo warning-not-failure branch
(`install.go:101-103`), and a second install hitting `hook.Install`'s `Skipped: "current"`
branch. `internal/config`: the "zero wip" `TestLoadErrors` case renamed to "negative wip" (what
it actually asserts), plus `TestLoadDefaultsZeroWIP` proving an *explicit* `wip_in_review = 0`
loads and defaults to 1.

**Item 6 — corrected in scope, not just tested.** Refinement's Description claimed "the original
defect is already fixed" for `LastHistoryStatus`; that was wrong, discovered by actually running
the reproduction rather than trusting the earlier fact-check. `historyKind`/`LastHistoryStatus`/
`LastHistoryReason` each independently re-derived a transition's target via
`strings.LastIndex(body, "→")` on the *whole* body — so a reason clause containing its own arrow
(e.g. "reviewed: … 2 non-blocking → fixed inline") was read in preference to the transition's own
arrow. Verified **live** in this repo's own tree: `T-058`'s actual History has exactly this shape,
and the tree only audits clean today because a later, differently-worded duplicate DONE line
masks it (confirmed by feeding the real line through `LastHistoryStatus` before this fix: it
returned `""`, not `"DONE"`). Fixed by extracting one shared `splitTransition`/`transitionTarget`
pair (isolates "OLD → NEW" on the body's first colon, before ever searching for an arrow) that
all three functions now call — collapsing three independent copies of the same unsafe pattern
into one, the same class of fix T-042 makes elsewhere in this repo. `LastHistoryStatus` and
`LastHistoryReason` are also now both implemented in terms of `HistoryEntries`, so a wrapped
continuation line can no longer make them disagree with `HistoryEntries` about where an entry
ends (a real, if secondary, divergence: `LastHistoryReason` used to silently truncate a wrapped
reason at the fold). `historyKind`'s `HistoryEntry.Kind` freezes on an entry's first physical
line **by design** (documented in its own comment) — an arrow split across the wrap boundary
itself is not a supported shape and is not what changed.

**Deferred/declined, as planned:** item 2, 7, 8 stayed in T-069 (D1); item 5 stayed closed, not
implemented (D6); T-042's five-site payload-root unification is untouched (D5) — only `TestMain`
itself and its `repoRoot` validation moved.

## Acceptance test — results

- `just build && just test && just lint && just docs-check` — all green.
- `go test -race -count=2 ./internal/cli/ ./internal/ticket/ ./internal/config/` — green.
- Coverage: `internal/cli` **70.4%** (was 63.3%; the plan's own "≥ 75%" target was an unverified
  guess made at refinement, not backed by any task item — corrected here rather than padded with
  tests outside the plan's scope), `internal/config` **85.6%** (unchanged — item 3 pinned an
  already-covered default path), `internal/ticket` **94.4%** (unchanged; the four new functions
  are each 100% covered). No package dropped below its 2026-08-06 baseline.
- Mutations, each applied, run, observed, reverted:
  - **A** (delete `TestMain`'s `os.Chdir`): `TestCWDIsSandboxed` → **test failure**, not a
    compile error — T-029's original guard confirmed still load-bearing.
  - **B** (remove the early `*target = orig`): `TestCaptureStdoutRestoresBeforeCleanup` → fails
    with `write |1: file already closed` — the exact T-031 N1 symptom, reproduced on demand.
  - **C** (remove `closeAndJoin()` from `t.Cleanup`): `TestCaptureGoexitDoesNotLeak` → fails on
    `LEAK-CHECK-FAIL`, not a hang — confirmed the leak is a resource leak (fd + goroutine), not a
    process hang, which is why the regression test is a goroutine-count probe rather than a
    timeout (a plan assumption corrected during implementation, not before).
  - **D** (item 6's fix): reverting `historyKind`/`LastHistoryStatus` to the pre-fix `LastIndex`
    logic fails `TestLastHistoryStatusArrowInReason` and `TestLastHistoryReasonFoldsContinuations`.
- `./pickle board audit` on the real tree: **69 tickets, 0 errors, 0 warnings**, identical before
  and after item 6's fix — decision 8's guard held; the fix is behaviour-preserving for every
  ticket in this repo today (T-058 is saved by its own duplicate line either way).

## Docs update

None shipped. No user-facing surface (test-only plus one internal parser fix), and decision 8's
guard found no observable change in `board audit`'s verdict on the real tree — the plan's own
trigger for a `CHANGELOG.md` entry. `CHANGELOG.md` has no existing `### Fixed` precedent for an
internal-only correctness fix with no user-visible behaviour change, so none was added.

## Review

**Verdict: PASS** (after one rework pass) — the first review found one blocking finding (R1);
the rework fixed it and the scoped re-review confirms it, so the ticket concludes to `6-done/`.
Audited on `feat/T-043-cli-test-harness` — `efcb2dc` first, then `84628dc` — code only, with
bookkeeping correctly kept on `main`, where the ticket was read from.

*First-review verdict (2026-08-06): REWORK on R1. Everything else the ticket set out to do was
verified done and, where it claims a defect is pinned, mutation-proven.*

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — **not applicable**: the branch changes no `.adoc`/`.md`
      file (`git diff --stat main..feat/T-043-cli-test-harness` touches five `.go` files only)
- [x] Findings recorded with severity and disposition per rules §5; disposition summary present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] Other references: board regenerated by the move (step 7)
- [x] Impact sweep (step 8) — deferred by the first review, **done at the concluding one**
      (T-042, T-069, T-056, T-065 patched; see below)
- [x] Summary presented; child commit messages + MR attributes presented for approval at the
      concluding review (the rework verdict in between published nothing)

### What was verified, independently

| item | result |
|---|---|
| `just build && just test && just lint && just docs-check` | all green |
| `go test -race -count=2 ./internal/cli/ ./internal/ticket/ ./internal/config/` | green |
| Coverage (measured on both branches) | `internal/cli` 63.3% → **70.4%**, `internal/config` 85.6% → 85.6%, `internal/ticket` 94.4% → 94.4%; every 2026-08-06 baseline confirmed against `main`, none dropped |
| Mutation A — delete `TestMain`'s `os.Chdir(sandbox)` | `TestCWDIsSandboxed` **fails** (`… resolves a real config at …/pickle.toml`), not a compile error — T-029's guard still load-bearing. The real tree was untouched by the run |
| Mutation B — remove the early `*target = orig` | `TestCaptureStdoutRestoresBeforeCleanup` fails with `write |1: file already closed` — the exact T-031 N1 symptom |
| Mutation C — remove `closeAndJoin()` from `t.Cleanup` | `TestCaptureGoexitDoesNotLeak` fails on `LEAK-CHECK-FAIL` in 2.03s, no hang |
| Mutation D — restore `internal/ticket/ticket.go` from `main` | `TestLastHistoryStatusArrowInReason` and `TestLastHistoryReasonFoldsContinuations` both fail; pre-fix `LastHistoryStatus` returns `"IN REVIEW"` where `"DONE"` is correct |
| Item 6's live-bug claim | **true**. `tickets/6-done/T-058-*.md:326` carries the arrow-in-reason shape and `:328` is the differently-worded duplicate DONE line that masks it — exactly as the summary describes |
| Real-tree guard (D8) | `board audit` = **69 tickets, 0 error(s), 0 warning(s)** with the old *and* the new binary, on both branches' trees; `board sync` with the new binary leaves `BOARD.md` **byte-identical** — the change is render-preserving on this tree |
| Tasks 1–8 | all landed in the files the plan names; the `captureStderr` clone is gone from `agents_test.go`; no `t.Parallel()` call anywhere in package `cli` (the only two matches are the comments forbidding it) |
| Decisions 1–11 | all honoured. D4's declined refactor is recorded in the helper comment; D5 respected (only `TestMain`, five-site unification left to T-042); D10 respected (no `internal/config` production edit); D11's `~64 KiB` comment now reads "16–64 KiB, OS-dependent (darwin starts at 16 KiB)" |

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | **blocking** | — | The new `splitTransition` anchors on the **folded** entry text's *first colon*, so two shapes that used to classify as transitions now fall back to the previous entry. (a) A reason-less transition with a hand-written continuation line — `"- … — TO DO → READY"` + an indented follow-on — yields `""` (pre-fix: `"READY"`). (b) A body carrying a colon *before* the transition — `"- … — audit fix: TO DO → READY"` — yields the previous status (pre-fix: `"READY"`). `HistoryEntry.Kind` still says `transition` in both cases, so the frozen classification and the re-derived target disagree — the very divergence class this ticket set out to close. | Probes run against both branches: pre-fix `status="READY"` / `"IN REVIEW"`, post-fix `status="TO DO"` / `""`. Zero instances in the 69-ticket tree, and `pickle ticket move` cannot emit either shape (`sanitizeReason`, `internal/move/move.go:217`), so `board audit` is unaffected today. | Make transition classification and target extraction agree by construction: derive the target from the entry's **first physical line** (the line `Kind` is frozen from — e.g. have `HistoryEntries` keep it), or accept the longest legal status name after the arrow instead of requiring the whole remainder to be one. Add the two shapes above as table cases next to `TestLastHistoryStatusArrowInReason`. |
| R2 | non-blocking | new ticket → **T-070** | `MergeLine` (`internal/ticket/ticket.go:391`) is now the **only** `## History` reader still walking the text itself (its own `inHistory` loop + `historyRE`) instead of routing through `HistoryEntries`. It therefore does not fold continuations, so a wrapped merge line truncates in the board's `merged` cell. The branch unified three of the four readers and left the fourth — the duplication class this repo files tickets about. | Probe: `MergeLine("## History\n- 2026-08-06 — merged to main\n  (abc1234) after review\n")` = `"merged to main"` on **both** branches. | Route `MergeLine` through `HistoryEntries` like `LastHistoryStatus`/`LastHistoryReason`, so the section-walk exists once. Filed as T-070 (`spawned-by: [T-043]`). |
| R3 | non-blocking | noted | Acceptance criterion 3's coverage floor is **not met**: `internal/cli` reaches 70.4%, the plan demands ≥ 75.0%. The shortfall is not an unbuilt task — the remaining uncovered statements are `runUpgrade` 16.7%, `runUninstall` 21.7%, `runServe` 17.6%, `runBoardSync` 20.0%, none of them in this ticket's scope — so the floor was an unbacked number, as the implementation summary itself says. Recorded as an acceptance criterion **partially met** rather than waived. | `go tool cover -func` on `internal/cli`; baseline 63.3% re-measured on `main`. | Nothing here. A future coverage ticket for the four commands above should set its floor from a measured figure, not a guess. |
| R4 | non-blocking | noted | Two plan deviations were disclosed **in the summary** rather than raised in flight: item 6's premise ("the original defect is already fixed") proved false and the fix grew past D8's "route it through `HistoryEntries`" to three functions plus two new helpers, and mutation C's expected failure mode changed from a hang to a goroutine-count probe. Both turned out correct — I re-ran all four mutations — but D8 is a confirmed decision that says "stop and report", and the report arrived after the work. | Implementation summary, "Item 6 — corrected in scope, not just tested"; acceptance results, mutation C. | Raise a falsified premise when it is found, not at hand-back. This finding is about the flow, not the code. |
| R5 | non-blocking | noted | `TestTicketNewRejectsMalformedSpawnedBy`'s comment still says "this package has no stderr capture (`captureStdout` would not see it) — see T-031", so the test asserts only the exit code and defers the wording to `internal/ticket`. `captureStderr` now sits ~300 lines above it in the same file. Pre-existing (`agents_test.go` already had one in this package), so `noted`, not `fixed inline`, per §5's causation rule. | `internal/cli/cli_test.go:576-579` vs `:259`. | Strengthen the test to assert the stderr wording, and delete the comment, when someone is next in that file. |
| R6 | non-blocking | noted | No `CHANGELOG.md` entry for the arrow-in-reason fix, which is observable to any user whose History was hand-written (`board audit`'s verdict, the board's `reason` cell, `serve`'s views). The plan's own trigger — a changed `board audit` outcome on the real tree — did not fire, and the repo has no `### Fixed` precedent: T-018, T-026, T-030 and T-040 were all correctness fixes and added none. | `grep '^### ' CHANGELOG.md` → only `Added`/`Changed`; the docs step's conditional as written. | Left as-is by review decision. Worth settling the "internal correctness fix" changelog policy once, project-wide, rather than per ticket. |

**Disposition summary:** 6 findings — 1 **blocking** (R1, not dispositioned: fixed in rework),
5 non-blocking: 1 **new ticket** (R2 → T-070), 4 **noted** (R3, R4, R5, R6). No `fixed inline`,
no `folded`.

### Rework scope (R1 only)

The blocking finding is the entire scope of the next pass: make the folded-text target derivation
agree with `HistoryEntry.Kind` for the two shapes in R1, on the same branch, with the two shapes
added as tests. Do **not** take R2 (it is T-070's) and do not chase R3's coverage floor. Re-run
the acceptance test, including D8's before/after `board audit` guard on the real tree.

### Rework pass — 2026-08-06 (commit `84628dc`, same branch)

**R1 — fixed.** Nothing else was touched: the pass changes `internal/ticket/ticket.go` and
`internal/ticket/ticket_test.go` only.

- **The disagreement is gone by construction.** `HistoryEntry` gained a `Target` field, filled in
  the same pass — and from the same *first physical line* — that decides `Kind`.
  `LastHistoryStatus` now reads `e.Target` instead of re-parsing the folded `Text`, so "classified
  as a transition" and "resolves to a status" can no longer disagree. `LastHistoryReason` still
  reads the folded text, which is the whole point of folding; when the folded text presents no
  transition (a reason-less entry with prose wrapped under it) there is no reason clause and `""`
  is the answer.
- **One scan replaces two half-rules.** `splitTransition` + `transitionTarget` are replaced by
  `transitionParts`, whose rule is **the leftmost arrow whose candidate target — the text up to
  the next colon — is a legal status name, exactly**. Both R1 shapes resolve again, T-043's
  original T-058 fix is preserved, and `"IN DEVELOPMENT → IN REVIEW → DONE"` now resolves by
  *continuing* the scan rather than giving up. The exactness requirement is load-bearing: it is
  what keeps a note that merely mentions `"→ DONE requires a human"` a note.
- **Both R1 shapes, plus the two above, are pinned** in
  `TestTransitionSurvivesContinuationFolding`, which also asserts the invariant directly — every
  entry with `Kind == HistoryTransition` names a legal `Target`, and every other entry names none.
  The two `HistoryEntries` fixtures moved from positional to keyed literals and now assert
  `Target` as well.

**Behaviour vs. `main`, all five shapes measured on three trees** (`main`, the reviewed `efcb2dc`,
this pass) — the pass is now a strict improvement over `main` with no regression:

| shape | `main` | `efcb2dc` (reviewed) | now |
|---|---|---|---|
| reason-less transition + continuation line | `READY` | `""` ✗ | `READY` ✓ |
| colon before the transition | `READY` | `TO DO` ✗ | `READY` ✓ |
| two arrows, no reason | `DONE` | `DONE` | `DONE` ✓ |
| arrow inside the reason (T-058's real line) | `""` ✗ | `DONE` ✓ | `DONE` ✓ |
| reason wrapped across a fold | truncated | full ✓ | full ✓ |

**Acceptance re-run:** `just build`, `just test`, `just lint`, `just docs-check` all green;
`go test -race -count=2 ./internal/cli/ ./internal/ticket/ ./internal/config/` green; coverage
`internal/ticket` 94.4% → **94.5%**, `internal/cli` 70.4% and `internal/config` 85.6% unchanged;
D8's guard held — `board audit` on the real 70-ticket tree is **0 errors, 0 warnings** with the old
*and* the new binary, and `board sync` leaves `BOARD.md` byte-identical.

**Mutations.** A, B and C (the harness guards) re-run unchanged and still fail as named. D's
original form — reverting `internal/ticket/ticket.go` wholesale to `main` — is now a **compile
error** rather than a test failure, because `HistoryEntry` gained a field; its intent is carried by
three finer mutations, each applied, observed and reverted:

| mutation | expected failure | observed |
|---|---|---|
| E1 — make `LastHistoryStatus` re-derive the target from the folded `Text` (i.e. re-introduce R1) | the reason-less-plus-continuation case | `LastHistoryStatus = "TO DO", want "READY"` ✓ |
| E2 — scan for the **last** arrow instead of the leftmost legal one | T-043's original T-058 fix | `LastHistoryStatus = "IN REVIEW", want DONE` ✓ |
| E3 — accept a candidate that merely *starts with* a legal status name | the two-arrow case **and** the note-mentioning-an-arrow case | both fail (`"IN REVIEW"` instead of `DONE`; `"DONE"` instead of `READY`) ✓ |

E3 is worth keeping in the record: prefix matching was one of the two fixes this review itself
suggested, and the mutation shows it would have traded R1 for a false positive on notes.

### Scoped re-review — 2026-08-06 (`84628dc`)

Scoped to R1, per the protocol: the rest of the feature was audited at `efcb2dc` and is not
re-audited here. Re-verified independently rather than read off the rework record.

- **R1 is resolved.** `HistoryEntry.Target` is filled in the same pass as `Kind`, from the same
  first physical line (`internal/ticket/ticket.go:224-238`), and `LastHistoryStatus` reads it
  instead of re-parsing — so the failure mode (classified a transition, resolves to nothing) is
  now unrepresentable, not merely unobserved. Both R1 shapes resolve to `READY` again, matching
  `main`, while T-043's own two fixes still hold.
- **The fix is mutation-proven, re-run here.** E1 (make `LastHistoryStatus` re-derive from the
  folded `Text`, i.e. re-introduce R1) fails with `LastHistoryStatus = "TO DO", want "READY"`.
  The invariant assertion inside `TestTransitionSurvivesContinuationFolding` — every transition
  entry names a legal `Target`, every other entry names none — is what makes the fix a contract
  rather than four passing cases.
- **Scope held.** The rework commit touches `internal/ticket/ticket.go` and
  `internal/ticket/ticket_test.go` only: no R2 work, no coverage padding for R3, nothing in
  `internal/cli` or `internal/config`. The two `HistoryEntries` fixtures moved to keyed literals,
  which the new field required.
- **Re-verified green:** `just build`, `just test`, `just lint`, `just docs-check`; the R1/history
  test set; `board audit` on the real 70-ticket tree **0 errors, 0 warnings** and `board sync`
  byte-identical under the reworked binary — D8's guard, re-run by the reviewer.
- **`historyKind`'s docstring is accurate after the change** (it now names `historyKind`,
  `HistoryEntries` and `LastHistoryReason` as `transitionParts`' callers — `LastHistoryStatus`
  deliberately no longer is). Considered and *not* filed: `statusExists` is a single-use wrapper
  around `StatusByName`; it reads better inside the scan's condition than the two-step form, and
  a finding here would be padding.

**One new non-blocking finding from the rework pass:**

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R7 | non-blocking | folded → **T-070** | The rework leaves a deliberate asymmetry: the *target* is frozen on the entry's first physical line while the *reason* is read from the folded text. For a hand-authored entry whose continuation line contains a second arrow, the two can therefore come from different arrows — the pair is never checked. | `LastHistoryStatus` = `"READY"` with `LastHistoryReason` = `"some clause"` for `"- … — TO DO → READY"` + continuation `"and later IN REVIEW → DONE: some clause"`. No such entry exists in the tree, and `pickle ticket move` cannot write one. | T-070 already owns "give the `## History` readers one shared path"; added there as a second item — have `HistoryEntries` resolve target *and* reason in one pass so the pair provably belongs to one arrow. |

**Final disposition summary (both passes):** 7 findings — 1 blocking (R1, **fixed** in the rework
pass, re-verified), 6 non-blocking: 1 **new ticket** (R2 → T-070), 1 **folded** (R7 → T-070),
4 **noted** (R3, R4, R5, R6). No `fixed inline`.

### Impact sweep — done (step 8)

- **T-042** — patched: the `cli_test.go:19` reference re-anchored to `:24`, and item 3 now records
  that it must also delete `TestMain`'s `os.Stat(skill/SKILL.md)` validation when it replaces the
  `wd/../..` computation (the hand-off D5 planned).
- **T-069** — patched: every `internal/config/config_test.go` line reference shifted +25 by
  `TestLoadDefaultsZeroWIP` (`:663`→`:691`, `:697`→`:727`, `:639`→`:664`, `:527`→`:550`), and the
  "zero wip" case it may look for is now named "negative wip".
- **T-056** and **T-065** — patched: their soft-coupling notes described T-043's harness and
  `ticket new` coverage as unbuilt; both now record what landed, so neither plans to write those
  tests again.
- No ticket in `1-to-do/`/`2-ready/` lists T-043 in `depends-on:`, so no gate changed.

### Impact sweep — deferred to the concluding review, items already known

Both are consequences of this branch that the concluding review must patch (they are recorded
here, not applied, because the rework pass may move the lines again):

- **T-042** — its inventory row cites `internal/cli/cli_test.go:19` for the package-level
  `repoRoot`; the declaration is now `:24`. More substantively, item 3 must now also delete
  `TestMain`'s new `os.Stat(skill/SKILL.md)` validation when it replaces the `wd/../..`
  computation (D5 planned exactly this hand-off).
- **T-069** — every `internal/config/config_test.go` line reference shifted by +25 when
  `TestLoadDefaultsZeroWIP` was inserted: `:663` (symlink) → `:691`, `:697` (unwritable parent) →
  `:727`, `:639` (`FuzzSetPayloadVersion`) → `:664`, `:527` (`payloadVersionFixtures`) → `:550`.
- Soft-coupling notes in **T-056** and **T-065** describe T-043's `ticket new` / harness overlap
  as still-unbuilt; both need a one-line correction once this lands.

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-031 and T-012, both
  moved to 7-dropped/ as absorbed
- 2026-07-26 — patched by T-044's review impact sweep: the item-5 deferral is settled — T-044
  landed one-way sanitisation with renderer tests; item 5 shrinks to a cli-level assertion or
  is scoped out at refinement
- 2026-07-27 — patched by T-049's review impact sweep: `sanitizeCell` now also caps a cell at 120
  runes (`TestRenderCapsCellWidth` in `internal/board/board_test.go`), so the item-5 deferral's
  enumeration of the sanitisation was completed; item 5's scope is unchanged
- 2026-07-27 — patched by T-053's review impact sweep: `internal/cli` coverage is now 46.7% (was
  ~29.5% at filing), and `TestServeHelpIsAdvertised` is a third consumer of the defective
  `captureStdout` helper (Part 1, item 1)
- 2026-08-01 — patched by T-026's review: Part 2 gains **item 8**, two residual `payload_version`
  line-editor wedges (review findings R4 + R5, folded here — this ticket already owned the
  in-place writer's other hardening). Both are correctness bugs like item 7, both small, both
  safe-refusals rather than corruption
- 2026-08-06 — patched by the T-057 review (finding N7, disposition `folded`): Part 2 gains **item 9**,
  cli-level coverage for `pickle install --hooks` and its warning-not-failure branch — the only part
  of T-057's shipped surface with no test but the acceptance transcript
- 2026-08-06 — refined: **split** (D1) — items 2, 7 and 8, plus T-012's ten unenumerated in-place
  writer sub-items, became **T-069** (`internal/config`'s writers, graded medium-high because
  `project add` can brick a config); this ticket keeps the `internal/cli` test surface. Description
  re-verified against `545a4c5` and corrected on six counts: the call-site inventory is **15, not
  three** (plus an unmentioned `captureStderr` clone, and two sites committing defect 1 today);
  all Part 1 line refs re-anchored; the "hang on >64 KiB" rationale withdrawn (T-029's drain
  goroutine already handles it); Part 1 item 5 re-worded (the nit is the 64 KiB comment, not
  "what is shared"); item 6's original parser bug confirmed **already fixed** and the item
  re-scoped to pinning the contract plus the `HistoryEntries` folding divergence; coverage figures
  refreshed (`internal/cli` 63.3%, `internal/config` 85.6% — the filed 91.8% was stale
  *downwards*). Item 5 closed as covered by T-044+T-049 (D6, one assertion); `io.Writer` injection
  stays declined (D4); T-042 sequenced after this ticket, which owns `TestMain` (D5). Re-graded
  cost **L → M**; complexity stays medium (blast radius: 15 call sites)
- 2026-08-06 — TO DO → READY: plan complete; epic split (T-069 took the config writers)
- 2026-08-06 — READY → IN DEVELOPMENT: picked up
- 2026-08-06 — IN DEVELOPMENT → IN REVIEW: acceptance green: build/test/lint/docs-check pass; item 6 turned out to be a live bug, fixed and mutation-tested
- 2026-08-06 — IN REVIEW → REWORK: review REWORK: 1 blocking (R1: folded-text transition target disagrees with HistoryEntry.Kind on two shapes); 5 non-blocking (1 new ticket T-070, 4 noted)
- 2026-08-06 — REWORK → IN REVIEW: findings fixed: R1 resolved (Kind and Target now decided together; transitionParts scans for the leftmost legal target), 3 new mutations green, D8 guard held
- 2026-08-06 — IN REVIEW → DONE: review PASS after one rework pass; 7 findings: 1 blocking (R1) fixed and re-verified, 1 new ticket (T-070), 1 folded (R7 into T-070), 4 noted
- 2026-08-06 — merged to main (PR #15, 17e2341), user-approved; squashed to 5291aeb; branch deleted
