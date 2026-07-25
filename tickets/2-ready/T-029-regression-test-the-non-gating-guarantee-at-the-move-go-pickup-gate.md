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

- **T-012** ("harden test coverage + TOML-safe render") also adds tests, but to `internal/config`,
  `internal/project`, and the board-audit engine — disjoint files from this ticket's
  `internal/move/move_test.go` and `internal/cli/cli_test.go`. If T-012 lands first it may already
  have introduced a stdout-capture or `TestMain` pattern in another package; reuse its shape
  rather than inventing a second one.
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
   to it. The five existing `newTicket` call sites stay **byte-for-byte unchanged**. `depends-on`
   keeps its `strings.Replace` (Scaffold hardcodes `depends-on: []`).
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

   Implementation notes: `os.Pipe()`, swap `os.Stdout`, restore with `t.Cleanup`/`defer`, close
   the writer **before** reading so the read terminates, and drain with `io.ReadAll` (output here
   is a handful of lines — no goroutine needed, but if you prefer one, join it before returning).
   `t.Fatal` on any pipe error.

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
   `os.RemoveAll(sandbox)` → `os.Exit(code)`. Do the cleanup **before** `os.Exit` (deferred
   functions do not run through `os.Exit`), and `panic`/`os.Exit(1)` with a clear message on any
   setup error. `t.TempDir()` is unavailable here — there is no `*testing.T`.

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

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-25 — TO DO → READY: plan complete
