---
id: T-029
title: regression-test the non-gating guarantee at the move.go pickup gate
project: pickle
depends-on: []
spawned-by: [T-024]
impact: medium
complexity: low
cost: S
---

# T-029 — regression-test the non-gating guarantee at the move.go pickup gate

## Description

T-024's defining property is that `spawned-by:` **never gates a pickup**. That guarantee is
enforced in **two** places — the audit's in-development loop (`internal/audit/audit.go:140-156`)
and `move.Move`'s own, stricter pickup gate (`internal/move/move.go:98-113`) — but only the
first is guarded by a test.

The T-024 review proved the gap by mutation: adding `SpawnedBy` to move.go's gate loop
(`range append(append([]string{}, t.DependsOn...), t.SpawnedBy...)`) left the **entire suite
green, 9/9 packages**. The same mutation applied to the audit loop correctly fails
`TestAudit/in-dev_spawned-by_parent_not_done`. So the gate a human actually hits when running
`pickle ticket move T-NNN in-development` — the one that produces the user-visible error — has
no regression guard for the decision it must honour. **Re-confirmed at refinement time
(2026-07-25):** the same mutation still leaves `go test ./...` green across all 9 packages.

### Scope

One case in `internal/move/move_test.go`: create a ticket whose `spawned-by` names a parent
still in `1-to-do/`, move it to `in-development`, and assert the move **succeeds**. Mirror the
naming of the audit's `in-dev spawned-by parent not done` case so the pair is greppable. Verify
the test is load-bearing by re-running the mutation above and confirming it now fails.

Note the Description's original advice — copy the `depends-on` string-rewrite trick at
`move_test.go:39` — is **stale**: `ticket.Scaffold` (`internal/ticket/ticket.go:293`) takes a
typed `spawnedBy []string` parameter, which `move_test.go`'s `newTicket` currently passes as
`nil`. The lineage goes in typed; only `depends-on` still needs the rewrite (Scaffold hardcodes
`depends-on: []`). This is the reverse of `internal/audit/audit_test.go`, whose `withSpawnedBy`
rewrite exists because its fixtures are raw string literals, not `Scaffold` output.

Two smaller test-hardening items from the same review, folded in while in this code:

- `internal/cli/cli_test.go:111-113` asserts only that `board audit` exits non-zero for a
  dangling `spawned-by` id, so it would also pass on an unrelated audit error. `runBoardAudit`
  prints findings with `fmt.Printf` to `os.Stdout` (`internal/cli/board.go:80-84`), so pinning
  the message means capturing stdout — no test in the repo does that yet.
- `internal/cli/cli_test.go:55-65`'s `newProject` helper makes CWD safety **opt-in**: any future
  test in package `cli` that calls a config-resolving command without it will walk up and write
  into the live board (the original T-024 blocking finding). A package-level `TestMain` that
  chdirs to a temp dir makes safety the default. Constraint discovered at refinement: `newProject`
  resolves its payload through the *relative* `os.DirFS(filepath.Join("..", ".."))`, so the repo
  root must be captured **before** `TestMain` chdirs away, or every install in the package breaks.

### Couplings

`spawned-by: [T-024]` — born from T-024's review (non-blocking findings N1, N5, N6). No hard
dependency: T-024's code is what makes the test meaningful, but the test can be written
whenever.

Soft couplings (no `depends-on`, no ordering constraint):

- **T-012** ("harden test coverage + TOML-safe render") also adds tests to **`internal/cli`** —
  its own text asks for "cli-level tests for `project add|list|remove` and `board audit`" and for
  `ticket new`. That is the *same package and same file* as this ticket's Tasks 2 and 3, so the
  constraint runs the opposite way to a plain "disjoint files" note: **Go allows only one
  `TestMain` per package**, and T-012's `board audit` output assertions need exactly the
  `captureStdout` helper this ticket builds. **T-029 should land first**; T-012 must then *reuse*
  `TestMain`/`captureStdout` rather than invent a second mechanism. Still no `depends-on` — T-012
  is unrefined in `1-to-do/` and gated behind `[T-001, T-002, T-003]`, so no ordering can be
  enforced from here; this is a note for whoever refines T-012.
- **T-027** ("audit: flag depends-on entries that reference the ticket itself") edits
  `internal/audit/audit_test.go`, which mutation B in the acceptance test touches transiently. No
  conflict, but whichever lands second should re-run the other's cases.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child-project `pickle` is this repo itself (`pickle.toml`, path `.`):

```
git checkout main
git checkout -b feat/T-029-regression-test-non-gating-pickup-gate
```

Local WIP commits encouraged. **Never push or open a merge request without explicit user
approval** (child-projects are publish-gated); merging is always the human's.

### Prerequisite gate (hard)

None. `depends-on: []`; T-024's code is already in `6-done/` and is what makes the test
meaningful, but nothing here waits on a merge.

### Confirmed design decisions (do not deviate without asking)

1. **Test-only change.** No production code is touched. In particular `runBoardAudit` is **not**
   refactored to take an injectable `io.Writer` — every other command in `internal/cli/` prints
   with `fmt.Printf`, so doing it for one command would be a half-migration. If the stdout
   capture in Task 2 turns out to be unpleasant, file a separate ticket rather than widening
   this one.
2. **Lineage goes in typed, not by string rewrite.** Add a core helper
   `newTicketFull(t, root, id, title string, deps, spawnedBy []string)` to
   `internal/move/move_test.go` that passes `spawnedBy` straight to `ticket.Scaffold`; keep the
   existing `newTicket(t, root, id, title string, deps ...string)` as a thin wrapper delegating
   to it. All **seven** existing `newTicket` call sites (`move_test.go:70, 101, 113, 128, 129,
   141, 142`) stay **byte-for-byte unchanged**. `depends-on` keeps its `strings.Replace`
   (Scaffold hardcodes `depends-on: []`).
3. **The new move case asserts success, not an error.** It is the mirror image of
   `TestDependencyGate`: same tree shape, opposite verdict. Name it so the pair greps together
   with the audit's `in-dev spawned-by parent not done` case.
4. **The cli assertion pins the printed message** by capturing `os.Stdout`, not by calling
   `audit.Audit` from the cli test. The point of a cli-package test is the user-visible surface.
   Stdout is process-global, like CWD: **no test in package `cli` may call `t.Parallel()`** —
   extend the existing comment that says so.
5. **`TestMain` is a backstop, not a replacement.** `newProject`'s own `t.Chdir(root)` stays —
   tests need to be *inside* their install, not merely outside the real repo. `TestMain` only
   guarantees the default CWD is harmless for tests that never call `newProject`.
6. **The repo root is captured before the chdir.** `TestMain` resolves the module root as
   `filepath.Abs(filepath.Join(os.Getwd(), "..", ".."))` into a package-level `repoRoot` var
   *before* chdirring, and `newProject` uses `os.DirFS(repoRoot)`. No relative path may survive
   in the package: `os.DirFS(filepath.Join("..", ".."))` silently resolves against whatever the
   CWD happens to be, which is exactly what TestMain changes.
7. **Every new/changed test must be shown to be load-bearing** by the mutation runs in the
   acceptance test. A test that passes both with and without the mutation has not fixed anything.

### Tasks

#### Task 1 — the missing regression guard in `internal/move/move_test.go`

1. Refactor the ticket helper per decision 2:

   ```go
   // newTicketFull writes a TO DO ticket directly (bypassing the CLI) with the
   // given depends-on and spawned-by, and registers it on the board.
   func newTicketFull(t *testing.T, root, id, title string, deps, spawnedBy []string) {
   	t.Helper()
   	body := ticket.Scaffold(id, title, "demo", "medium", "medium", "M", spawnedBy)
   	if len(deps) > 0 {
   		body = strings.Replace(body, "depends-on: []", "depends-on: ["+strings.Join(deps, ", ")+"]", 1)
   	}
   	// … unchanged: write to 1-to-do/, then board.AddTODORow
   }

   func newTicket(t *testing.T, root, id, title string, deps ...string) {
   	t.Helper()
   	newTicketFull(t, root, id, title, deps, nil)
   }
   ```

   Keep the existing comment about why the board row is added (audit cleanliness) on
   `newTicketFull`.

2. Add the case — the exact inverse of `TestDependencyGate` (`move_test.go:139-165`):

   ```go
   // TestSpawnedByDoesNotGatePickup is the move-side twin of the audit's
   // "in-dev spawned-by parent not done" case: lineage never blocks a pickup, so a
   // ticket whose spawned-by parent is still in 1-to-do/ picks up cleanly. The
   // identical shape with depends-on is TestDependencyGate's first rejection.
   //
   // Load-bearing check: adding t.SpawnedBy to move.go's pickup gate loop must make
   // this test fail. Before T-029 that mutation left the whole suite green.
   func TestSpawnedByDoesNotGatePickup(t *testing.T) {
   	root, cfg := newProject(t)
   	newTicket(t, root, "T-001", "Parent")                                  // stays in 1-to-do
   	newTicketFull(t, root, "T-002", "Child", nil, []string{"T-001"})
   	mustMove(t, root, cfg, "T-002", "ready", "")
   	mustMove(t, root, cfg, "T-002", "in-development", "") // must NOT be gated
   	assertClean(t, root, cfg)
   }
   ```

   `mustMove` already fails the test with the `Move()` error text, so a regression reports the
   exact gate message. `assertClean` additionally proves the audit agrees the tree is legal —
   covering both halves of the guarantee from the move side.

#### Task 2 — pin the `board audit` diagnostic in `internal/cli/cli_test.go`

1. Add a stdout-capture helper near the other helpers:

   ```go
   // captureStdout runs fn with os.Stdout redirected to a pipe and returns what it
   // printed. The commands in this package print with fmt.Printf to the real
   // os.Stdout, so this is the only way to assert their diagnostics. os.Stdout is
   // process-global — like the CWD, it forbids t.Parallel() in this package.
   func captureStdout(t *testing.T, fn func()) string { … }
   ```

   Implementation notes (two of these are **mandatory**, not stylistic — this helper is a
   harness other tickets will reuse):

   - `os.Pipe()`, swap `os.Stdout`, and **restore via `t.Cleanup(func() { os.Stdout = orig })`**,
     never by a sequential assignment after `fn()`. If `fn` ever calls `t.Fatal` or panics,
     `runtime.Goexit` skips the sequential restore and leaves the pipe installed as `os.Stdout`
     for **every later test in the package**, whose output then vanishes into a filling pipe.
   - **Drain with a reader goroutine** (`go func() { buf, _ = io.ReadAll(r) }()`, joined before
     returning) rather than reading after `fn` finishes. With no concurrent reader, any `fn` that
     writes more than the OS pipe buffer (~64 KiB) blocks forever and the test *hangs* instead of
     failing. Today's output is two lines, but a 30-ticket `board audit` or a `board sync` summary
     is a plausible reuse.
   - Close the writer before joining so the read terminates; `t.Fatal` on any pipe error.

2. Rewrite the tail of `TestTicketNewSpawnedByUnknownID` (`cli_test.go:111-113`) to assert both
   the exit code **and** the message:

   ```go
   var code int
   out := captureStdout(t, func() { code = Run(nil, "test", []string{"board", "audit"}) })
   if code == exitOK {
   	t.Error("board audit = 0, want non-zero for a dangling spawned-by id")
   }
   if !strings.Contains(out, "spawned-by T-404 does not exist") {
   	t.Errorf("board audit did not report the dangling id:\n%s", out)
   }
   ```

   The substring must match `internal/audit`'s wording — it is the same string
   `TestAudit/dangling_spawned-by` pins (`audit_test.go:143`).

#### Task 3 — make CWD safety the default in package `cli`

1. Add `TestMain` plus the captured root, per decision 6:

   ```go
   // repoRoot is the module root, captured before TestMain moves the process CWD.
   // newProject's payload used to be the relative "../.." — which only worked because
   // tests ran from internal/cli/. Anything relative here breaks under TestMain.
   var repoRoot string

   // TestMain makes CWD safety the default rather than opt-in: commands in this
   // package resolve their target by walking *up* from the process CWD
   // (loadConfig -> config.Find), so a test that forgets newProject's t.Chdir would
   // otherwise find pickle's own pickle.toml and mutate the real board.
   func TestMain(m *testing.M) { … }
   ```

   Body: `os.Getwd()` → `repoRoot = filepath.Abs(filepath.Join(wd, "..", ".."))` →
   `os.MkdirTemp("", "pickle-cli-sandbox")` → `os.Chdir(sandbox)` → `code := m.Run()` →
   `os.Chdir(repoRoot)` → `os.RemoveAll(sandbox)` → `os.Exit(code)`. Do the cleanup **before**
   `os.Exit` (deferred functions do not run through `os.Exit`), and `os.Exit(1)` with a clear
   message on any setup error. `t.TempDir()` is unavailable here — there is no `*testing.T`.

   Two constraints on the shape of that body:

   - **Keep the sandbox path in a named variable that the cleanup references.** Mutation C in the
     acceptance test deletes the `os.Chdir(sandbox)` call and requires the result to be a *test
     failure*. If the path is inlined (`os.Chdir(mustTempDir())`) or the `RemoveAll` is dropped,
     deleting that line instead yields an unused-variable **compile error** — which "fails" for
     the wrong reason and proves nothing about the guard.
   - **Chdir back to `repoRoot` before `RemoveAll`.** After `m.Run()` the process CWD is still the
     sandbox (each test's `t.Chdir` restored it), so removing it would delete the CWD. Legal on
     macOS/Linux, but pointlessly sloppy.

2. Point `newProject` at the absolute root: `install.Run(os.DirFS(repoRoot), root, "test", …)`.
   Update its doc comment — the chdir is now a *second* line of defence, not the only one — and
   keep the "no `t.Parallel()`" warning.

3. Add the guard that makes the sandbox load-bearing:

   ```go
   // TestCWDIsSandboxed proves TestMain's sandbox actually holds: from the default
   // process CWD no pickle.toml is discoverable, so a config-resolving command run
   // without newProject fails loudly instead of writing into the real board.
   func TestCWDIsSandboxed(t *testing.T) {
   	wd, err := os.Getwd()
   	if err != nil {
   		t.Fatal(err)
   	}
   	if path, err := config.Find(wd); err == nil {
   		t.Fatalf("default test CWD %s resolves a real config at %s — TestMain sandbox is broken", wd, path)
   	}
   }
   ```

   This adds an `internal/config` import to `cli_test.go` (no cycle — package `cli` already
   depends on it).

### Acceptance test

All commands run from the repo root. Steps 3–5 are mutation checks that prove each new test is
load-bearing; **commit the real work first** so `git checkout` restores only the mutation, and
revert all three before finishing.

```
# 1. baseline: suite + lint green
just test
just lint

# 2. the new/changed tests exist and pass by name
go test ./internal/move/ -run TestSpawnedByDoesNotGatePickup -v
go test ./internal/cli/  -run 'TestTicketNewSpawnedByUnknownID|TestCWDIsSandboxed' -v

# 3. mutation A — lineage wrongly added to move.go's pickup gate.
#    Edit internal/move/move.go:100 to:
#      for _, dep := range append(append([]string{}, t.DependsOn...), t.SpawnedBy...) {
go test ./internal/move/ -run TestSpawnedByDoesNotGatePickup     # MUST FAIL
git checkout internal/move/move.go

# 4. mutation B — the cli assertion is not vacuous.
#    In internal/audit/audit.go, change the dangling-spawned-by message wording
#    (e.g. "does not exist" -> "is unknown"), and fix audit_test.go's expected
#    substring so only the cli test is left asserting the old text.
go test ./internal/cli/ -run TestTicketNewSpawnedByUnknownID     # MUST FAIL
git checkout internal/audit/audit.go internal/audit/audit_test.go

# 5. mutation C — the sandbox guard is load-bearing: delete TestMain's os.Chdir call.
go test ./internal/cli/ -run TestCWDIsSandboxed                  # MUST FAIL
git checkout internal/cli/cli_test.go

# 6. the real board is untouched by a full test run (the T-024 blocking finding).
#    Compare before/after rather than requiring an empty status: this ticket's own
#    flow bookkeeping legitimately shows up under tickets/.
git status --short tickets/ > /tmp/t029-before
just test
git status --short tickets/ > /tmp/t029-after
diff /tmp/t029-before /tmp/t029-after   # MUST print nothing

# 7. final state clean
just lint
just build && ./pickle board audit      # 0 errors
```

Expected: steps 1, 2, 6, 7 green; steps 3, 4, 5 each fail on exactly the named test and on
nothing else; step 6's `diff` silent — in particular no new untracked file under
`tickets/1-to-do/` and no edit to `tickets/BOARD.md` caused by running the suite.

### Docs update

**No user-facing surface** — this ticket ships tests and a test harness only. Nothing in
`README.md`, the skill payload, or `AGENTS.md` changes. The `internal/audit/audit.go:138-139`
comment already documents *why* lineage is absent from the gate; the analogous invariant comment
at `internal/move/move.go:98` may be extended to name the guarding test (`TestSpawnedByDoesNotGatePickup`)
so the next reader can find it — a code comment, not docs.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint` clean; all mutations reverted
   (`git diff` shows only test files plus, optionally, the one-line comment in `move.go`).
2. No docs to update (see above) — state that explicitly in the summary.
3. Write a summary: files touched, the mutation results proving each test is load-bearing, and
   anything deferred.
4. Suggested Conventional Commit message:

   ```
   test: guard the spawned-by non-gating rule at the pickup gate (T-029)

   move.Move's pickup gate is the second enforcement point of T-024's
   non-gating guarantee and was unguarded: adding SpawnedBy to its loop left
   the whole suite green. Adds the inverse of TestDependencyGate, pins the
   board-audit diagnostic for a dangling spawned-by id by capturing stdout,
   and makes CWD safety in package cli the default via TestMain (with a
   TestCWDIsSandboxed guard) instead of opt-in per test.
   ```

   The scope is deliberately **omitted**: the change spans `internal/move` and `internal/cli`
   with no single scope, and per Conventional Commits an omitted scope beats a placeholder.

5. Commit locally on the branch. **Do not push or open a merge request without user approval.**
   Present the message; after approval, finalize (squash or keep history — user's choice), push,
   open the MR. Merging is the human's.

## Implementation notes (2026-07-25)

Executed on `feat/T-029-regression-test-non-gating-pickup-gate`. All three tasks landed as
planned; no deviations from the confirmed design decisions.

**Files touched** (test code only, plus one comment):

- `internal/move/move_test.go` — `newTicket` split into typed `newTicketFull(t, root, id, title,
  deps, spawnedBy []string)` + thin variadic wrapper; all seven existing call sites byte-for-byte
  unchanged. Added `TestSpawnedByDoesNotGatePickup`.
- `internal/cli/cli_test.go` — added `repoRoot` + `TestMain` (sandbox chdir), `TestCWDIsSandboxed`,
  and `captureStdout`; repointed `newProject` at `os.DirFS(repoRoot)`; strengthened
  `TestTicketNewSpawnedByUnknownID` to assert the printed diagnostic.
- `internal/move/move.go` — comment only (names the guarding test at the gate, per the Docs step).
  No behaviour change.

**Mutation results — every new test is load-bearing:**

| Mutation | Expected | Observed |
|---|---|---|
| A: `SpawnedBy` appended to move.go's gate loop | `TestSpawnedByDoesNotGatePickup` fails | **Fails, and it is the only failure in the whole suite** — `cannot pick up T-002: dependency T-001 is in TO DO (must be DONE and merged)`. Pre-T-029 this mutation was green in all 9 packages. |
| B: audit's dangling-spawned-by wording changed to "is unknown" (+ audit_test substring updated) | only `internal/cli` fails | **Only `internal/cli` failed**, printing the captured output — proving the assertion reads the real message, not just the exit code. |
| C: `os.Chdir(sandbox)` deleted from `TestMain` | `TestCWDIsSandboxed` fails *as a test* | **Still compiles** (finding N5 honoured: `sandbox` stays referenced by the cleanup), then fails naming the real config it would otherwise have mutated: `default test CWD …/internal/cli resolves a real config at …/pickle.toml`. |

**Step 6 (the T-024 blocking finding this hardens against):** `git status --short tickets/`
before/after a full `go test ./...` diffs empty; `--untracked-files=all` under `tickets/` is 0
lines. No test writes into the real board.

**Step 7:** `just test` green (9/9), `just lint` clean, `gofmt -l .` empty, `./pickle board audit`
→ 30 tickets, 0 errors, 0 warnings.

**Deferred:** nothing from this ticket's scope. The audit findings N6–N9 were informational and
needed no code (N6's chdir-before-remove was implemented anyway); N2's amendment is advice for
whoever refines T-012 and is recorded in this ticket's Couplings.

## Review

Reviewed 2026-07-25 on `feat/T-029-regression-test-non-gating-pickup-gate` (2 commits ahead of
`main`; diff limited to `internal/cli/cli_test.go`, `internal/move/move_test.go`, a 2-line comment
in `internal/move/move.go`, plus this ticket and `tickets/BOARD.md`). The generic protocol only —
`pickle.toml` defines no overarching and no per-child `review_addendum`.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep (step 4a; no docs build configured)
- [x] Findings classified blocking/non-blocking and recorded here; non-blocking → new `1-to-do/` tickets (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] `BOARD.md` updated (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

### Step 2 — implementation audit: all criteria **met**

Every acceptance-test step re-run verbatim from the repo root:

| Step | Command | Result |
|---|---|---|
| 1 | `just test` / `just lint` | **green**, 9/9 packages; `go vet` + `gofmt -l .` clean |
| 2 | `go test ./internal/move/ -run TestSpawnedByDoesNotGatePickup -v` | **PASS** |
| 2 | `go test ./internal/cli/ -run 'TestTicketNewSpawnedByUnknownID\|TestCWDIsSandboxed' -v` | **PASS** (both) |
| 3 | mutation A — `SpawnedBy` appended to `move.go`'s gate loop | **FAILS**, and is the *only* failure across all 9 packages: `cannot pick up T-002: dependency T-001 is in TO DO (must be DONE and merged)`. Independently re-confirmed that the same mutation on `main` leaves the whole suite green — the gap the ticket exists to close was real. |
| 4 | mutation B — audit wording → "is unknown" (+ `audit_test` substring fixed) | **only `internal/cli` fails**, printing the captured output: the assertion reads the real message, not just the exit code |
| 5 | mutation C — `os.Chdir(sandbox)` deleted from `TestMain` | **still compiles** (decision honoured: `sandbox` stays referenced by the cleanup), then fails *as a test*: `default test CWD …/internal/cli resolves a real config at …/pickle.toml` |
| 6 | `git status --short tickets/` before/after a full `go test ./...` (cache cleared) | **diff empty**; `--untracked-files=all tickets/` → 0 lines. The T-024 blocking finding stays closed. |
| 7 | `just lint`; `just build && ./pickle board audit` | clean; **30 tickets, 0 errors, 0 warnings** |

Beyond the plan: `go test -race ./internal/cli/ ./internal/move/` and `go test -count=2
./internal/cli/` both pass — the new process-global manipulation (CWD, `os.Stdout`) is race-clean
and repeat-safe, and no `pickle-cli-sandbox*` directory is left behind in `$TMPDIR`.

All seven confirmed design decisions honoured: no `io.Writer` refactor of `runBoardAudit`; lineage
typed into `ticket.Scaffold`; the seven pre-existing `newTicket` call sites
(`move_test.go:81,112,124,139,140,152,153`) byte-for-byte unchanged; restore via `t.Cleanup` with a
concurrent drain; named `sandbox` var referenced by the cleanup; `os.Chdir(repoRoot)` before
`RemoveAll`; `newProject`'s own `t.Chdir` kept; no `t.Parallel()` anywhere under `internal/`.

### Step 4a — documentation audit

The ticket ships **no user-facing surface** (tests, a test harness, and one code comment), so its
"no docs update" claim is correct and verified: `README.md`, `AGENTS.md`/`CLAUDE.md`, and the
`skill/` payload are untouched by the diff and none of them documents the pickup gate's internals
or this repo's test conventions. `pickle.toml` configures no `docs` command, so there is no docs
build to run. Whole-tree sweep of the five root `*.md` files plus `skill/` found no reference to
`TestMain`, `captureStdout`, `t.Parallel`, or test-helper conventions that could go stale.

### Findings

| # | Severity | Finding | Evidence | Disposition |
|---|---|---|---|---|
| B— | — | **No blocking findings.** | — | — |
| N1 | non-blocking | `captureStdout` leaves `os.Stdout` pointing at a **closed** pipe from the moment it returns until test cleanup, so any later stdout write in the same test is silently discarded (callers ignore `fmt.Printf`'s error). Harmless today — the sole call site is its test's last statement — but the ticket bills this as a harness T-012 will reuse. | `internal/cli/cli_test.go:150,154,167,171`; measured with a probe test on a scratch copy: after `captureStdout` returns, `fmt.Println` gives `n=0 err=write \|1: file already closed`. | → **T-031** |
| N2 | non-blocking | On the `t.Fatal`/panic-inside-`fn` path, `captureStdout` leaks the reader goroutine and both pipe fds for the life of the test process: `w.Close()` and `<-done` sit *after* `fn()`, so `runtime.Goexit` skips them and `io.ReadAll(r)` blocks forever. This is the exact failure mode the comment at `:151-153` defends against — fixed for `os.Stdout`, not for the pipe. | `internal/cli/cli_test.go:165-173` | → **T-031** |
| N3 | non-blocking | `TestMain` leaks the sandbox directory on its own error path (`os.Exit(1)` at `:50` precedes the `RemoveAll` at `:59`) and on a panic inside `m.Run`. Also, the comment at `:57` ("`os.Exit` runs no deferred functions, so this cannot be a defer") is imprecise: a `defer` inside an *inner* function does run before `os.Exit`; the real constraint is "not a defer in `TestMain`'s own body". | `internal/cli/cli_test.go:43-51,55-60` | → **T-031** |
| N4 | non-blocking | `TestMain` computes `repoRoot` as `wd/../..` and never asserts it is really the payload root; a wrong value surfaces much later as an opaque `install: …` failure in every test. The point of the change is to stop relying on an implicit CWD, so the assumption should assert itself (`os.Stat(repoRoot/skill/SKILL.md)`). | `internal/cli/cli_test.go:35-39,115` | → **T-031** |
| N5 | non-blocking | Three idioms for "the payload root" now coexist across five test files: a duplicated `payloadRoot()` returning relative `../..` (`install_test.go:15`, `doctor_test.go:14` — whose comment even says "mirrors install_test.go"), inlined `os.DirFS(filepath.Join("..",".."))` (`move_test.go:20`, `sync_test.go:21`), and now an absolute `repoRoot` (`cli_test.go:19`). This ticket makes the divergence worse (2 idioms → 3). Any future chdir in those packages silently breaks their relative paths. | the five sites above | → **T-032** |
| N6 | non-blocking (informational) | `internal/audit/audit.go`'s lineage comment does **not** name its guarding case (`TestAudit/"in-dev spawned-by parent not done"`), while the new `move.go:98-100` comment does name `TestSpawnedByDoesNotGatePickup`. Cosmetic asymmetry in the "greppable pair". Better as a drive-by in **T-027**, which already edits `internal/audit`. | `internal/audit/audit.go:138-139` vs `internal/move/move.go:98-100` | noted in T-027's Couplings (step 8) |
| N7 | non-blocking (informational) | `captureStdout`'s comment calls the pipe buffer "~64 KiB"; that is Linux's default — darwin, this repo's dev platform, starts at 16 KiB. The rationale is unaffected, only the number. Comment nit. | `internal/cli/cli_test.go:156-157` | → **T-031** |
| N8 | informational, no action | `"spawned-by T-404 does not exist"` is now pinned in two packages (`cli_test.go:232` and `audit_test.go:143`), so re-wording that audit error breaks two tests in two packages. Deliberate per decision 4 and accurately explained at `cli_test.go:223-224`. Flagged so it is not a surprise. | — | none |
| N9 | informational, no action | `BOARD.md`'s branch cell for T-029 reads `feat/T-029-regression-test-the-non-gating-guarantee-at-the-move-go-pickup-gate`, but the real branch is `feat/T-029-regression-test-non-gating-pickup-gate`. Pre-existing, already ticketed. | `tickets/BOARD.md:30` | already **T-023** |

Consistency sweep also confirmed the guard is **correctly scoped**: root discovery by walking up
from the CWD exists *only* in `internal/cli` (`project.go:39-43` plus the four
`os.Getwd`+`config.Find` pairs in `install.go`). `audit`, `board`, `move`, `sync`, `doctor`, and
`ticket` all take an explicit `root`, and every test there passes `t.TempDir()`; the lone other
`os.Getwd` (`install/install.go:403`, `ensureSymlink`) only prettifies a path for the result
summary. `config_test.go:138` already asserts `Find(t.TempDir())` errors, so `TestCWDIsSandboxed`
adds no new environmental assumption. No `TestMain` existed in package `cli` before, so nothing was
displaced.

**Verdict: no blocking findings → `6-done/`.** Seven non-blocking findings, five spawned as
**T-031** (cli test-harness robustness: N1–N4, N7) and **T-032** (unify the payload-root idiom:
N5); N6 recorded against T-027; N8/N9 informational.

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-25 — TO DO → READY: plan complete
- 2026-07-25 — READY → IN DEVELOPMENT: picked up; applicability audit clean, findings N1-N5 folded in
- 2026-07-25 — IN DEVELOPMENT → IN REVIEW: acceptance green; all 3 mutations confirmed load-bearing
- 2026-07-25 — IN REVIEW → DONE: review clean: no blocking findings; 7 non-blocking (T-031, T-032 spawned; N6 -> T-027)
