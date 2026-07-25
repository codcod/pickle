---
id: T-031
title: harden the internal/cli test harness (captureStdout stdout restore + pipe lifecycle, TestMain sandbox lifecycle)
project: pickle
depends-on: []
spawned-by: [T-029]
impact: medium
complexity: low
cost: S
---

# T-031 — harden the internal/cli test harness (captureStdout stdout restore + pipe lifecycle, TestMain sandbox lifecycle)

## Description

T-029 built the first real test harness in package `internal/cli`: a `TestMain` that sandboxes the
process CWD, a `TestCWDIsSandboxed` guard, and a `captureStdout` helper. All of it works for
T-029's own call sites, and T-029's review found nothing blocking. But T-029 explicitly bills the
harness as reusable — **T-012** is already earmarked to assert `board audit` / `project` output
through `captureStdout` — and its review turned up four robustness gaps plus a comment nit that are
cheap to close *before* a second consumer arrives and inherits them.

The four gaps (T-029 review findings N1–N4, N7):

1. **`captureStdout` leaves `os.Stdout` pointing at a closed pipe after it returns**
   (`internal/cli/cli_test.go:150,154,167,171`). `w` and `r` are closed at the end of the helper,
   but the only restore is the `t.Cleanup` registered at `:154`, which runs at *test* end. So from
   the helper's return until then, `os.Stdout` is a closed fd: every command in this package prints
   with `fmt.Printf` and ignores the error, so the output silently vanishes. Measured with a probe
   test: after `captureStdout` returns, `fmt.Println` yields `n=0 err=write |1: file already
   closed`. Harmless in T-029 only because its single call site is the last statement of its test.
   **Fix:** assign `os.Stdout = orig` immediately after `fn()` returns (before `w.Close()`), and
   keep the `t.Cleanup` as the Goexit/panic backstop. This does **not** reverse T-029's decision 4
   ("restore through `t.Cleanup`, never by an assignment after `fn()`") — the cleanup stays and
   remains the guarantee; the assignment is an additional early release. Say so in the comment, or
   the next reader will think the decision was undone.

2. **The pipe and its reader goroutine leak whenever `fn` calls `t.Fatal` or panics**
   (`internal/cli/cli_test.go:165-173`). `w.Close()` and `<-done` sit after `fn()`, so
   `runtime.Goexit` skips both and `io.ReadAll(r)` blocks forever on a write end that is never
   closed. Same leak on the helper's own `t.Fatalf` paths at `:167` and `:172`. This is precisely
   the failure mode the comment at `:151-153` reasons about — it was closed for `os.Stdout` and
   left open for the pipe. **Fix:** move close + join into the same cleanup, e.g. a
   `sync.OnceFunc`-wrapped `closeW` referenced by both the happy path and
   `t.Cleanup(func() { os.Stdout = orig; closeW(); <-done; _ = r.Close() })`.

3. **`TestMain` leaks its sandbox directory on its own error path and on a panic in `m.Run`**
   (`internal/cli/cli_test.go:43-51,55-60`). If `os.Chdir(sandbox)` fails, `os.Exit(1)` runs before
   the `RemoveAll`. **Fix:** wrap the run in an inner function so a `defer` is legal —
   `code := func() int { defer func() { _ = os.Chdir(repoRoot); _ = os.RemoveAll(sandbox) }(); … ; return m.Run() }()`.
   That keeps both of T-029's constraints intact (the `sandbox` path stays a named variable the
   cleanup references, so mutation C still yields a test failure rather than an unused-variable
   compile error; and the chdir back still precedes the removal). While there, correct the comment
   at `:57`: "`os.Exit` runs no deferred functions, so this cannot be a defer" is imprecise — a
   defer inside an inner function *does* run before `os.Exit`; the real constraint is "not a defer
   in `TestMain`'s own body".

4. **`repoRoot` is never validated** (`internal/cli/cli_test.go:35-39`). It is `wd/../..` on faith;
   a wrong value surfaces far away as an opaque `install: …` failure in every test that calls
   `newProject` (`:115`). The whole point of T-029's change was to stop depending on an implicit
   CWD, so the assumption should assert itself: after resolving `repoRoot`, `os.Stat` the payload
   marker `skill/SKILL.md` (what `install.Run` actually needs) and `os.Exit(1)` with a clear
   message if it is missing. Note that **T-032** proposes replacing this computation outright with a
   `runtime.Caller`-based helper, which subsumes this item — see Couplings.

5. **Comment nit (N7):** `captureStdout`'s comment at `:156-157` calls the pipe buffer "~64 KiB";
   that is Linux's default. On darwin — this repo's dev platform — a pipe starts at 16 KiB and grows
   toward 64 KiB, so the threshold the comment justifies is 4× lower there. The reasoning is
   unaffected; the number should read "16–64 KiB, OS-dependent".

Also worth folding in while in this code: the `TestMain` doc comment does not warn that **Go permits
exactly one `TestMain` per package**, so the next ticket adding tests here (T-012) must *extend* it
rather than write a second one. T-029 records this in its own Couplings, but the code is where the
next author will look.

### Scope

Test-only, one file (`internal/cli/cli_test.go`), no production code. Every change must be shown to
be load-bearing the way T-029 did it — in particular, mutation C from T-029's acceptance test
(delete `TestMain`'s `os.Chdir`) must still produce a **test failure, not a compile error**, and
`go test -race -count=2 ./internal/cli/` must stay green. A probe test that asserts stdout still
works after `captureStdout` returns is the natural guard for item 1; whether it ships or is only run
during development is a decision for refinement.

### Couplings

`spawned-by: [T-029]` — findings N1–N4 and N7 of T-029's review. No hard dependency, but the code
this ticket edits **only exists on T-029's branch**, so it cannot start before T-029 is merged to
`main`; that is a merge-order fact, not a `depends-on:` (T-029 is already in `6-done/`).

Soft couplings (no `depends-on`, no ordering enforced):

- **T-032** ("unify the test payload-root idiom") replaces `repoRoot`'s `wd/../..` computation with
  a shared CWD-independent helper. That **subsumes item 4** of this ticket: if T-032 lands first,
  drop item 4 here; if this one lands first, T-032 must delete the validation along with the
  computation. Both are small — whoever picks up the second should re-read the first.
- **T-012** ("harden test coverage + TOML-safe render") is the intended second consumer of
  `captureStdout` and of `TestMain`, in this same file. It must reuse both rather than invent a
  parallel mechanism (Go allows one `TestMain` per package). Landing this ticket first means T-012
  inherits a fixed harness; landing it second means reconciling two sets of edits to the same
  helpers. T-012 is unrefined and gated behind `[T-001, T-002, T-003]`, so nothing can be enforced
  from here.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
