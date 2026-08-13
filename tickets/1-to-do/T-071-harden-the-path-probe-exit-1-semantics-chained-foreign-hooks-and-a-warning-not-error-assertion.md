---
id: T-071
title: harden the PATH probe: exit-1 semantics, chained foreign hooks, and a warning-not-error assertion
project: pickle
depends-on: []
spawned-by: [T-068]
impact: low-medium
complexity: low
cost: S
---

# T-071 — harden the PATH probe: exit-1 semantics, chained foreign hooks, and a warning-not-error assertion

## Outcome

After this ships, `hooks status`/`doctor` no longer misreads a probe's exit code 1 as "the guard can't run" when it actually proves the guard ran and found a violation, and the probe's two remaining edge cases (chained foreign hooks, the warning-not-error assertion) are closed the same way.

## Description

T-068 shipped `internal/hook.Probe()` — the check that stops `hooks status`/`doctor` reporting a
pre-commit guard as healthy when the `pickle` on `PATH` cannot actually run it. Its review found
three small gaps in the probe's *reach*, batched here (T-068 review findings R3, R5, R11). None of
them breaks the golden path; together they are one coherent tightening pass on one function plus
its tests.

### 1. Exit 1 proves capability, but is read as incapability (R3)

`probeCapable` (`internal/hook/probe.go`) treats **only exit 0** as "can run the guard". But the
exit-code contract T-057 fixed says exit `1` means *the guard ran and found a violation* — which
is proof the verb exists. Because the probe runs in `os.MkdirTemp("", …)`, and that honours
`TMPDIR`, the "empty" directory can land **inside** a pickle project: if that project is a git
repo on a `feat/` branch with `tickets/` staged, the probed binary correctly reports a violation,
exits 1, and the probe concludes the binary is inert.

Measured during the T-068 review, with the very build that ships the probe:

```
$ TMPDIR=$D/proj/tmpdir pickle-current doctor
WARNING: hooks: …/.git/hooks/pre-commit is installed and current, but the pickle on PATH
cannot run the guard — it is inert (pickle v0.2.2-74-g1c5cc03 at …/bin/pickle)
```

— i.e. the inverse of the lie T-068 exists to kill: it declares a *working* guard dead. The
control run with the default `TMPDIR` warns about nothing.

Likely fix: accept exit `0` **or** `1` as capable (only `1` is load-bearing in the shim's
contract, and both prove the verb dispatched); optionally also pin the probe's working directory
somewhere `TMPDIR` cannot redirect, or set `GIT_CEILING_DIRECTORIES`. Refinement decides which,
and whether the exit-2-and-above mapping stays as-is.

### 2. A chained foreign hook is never probed (R5)

`Probe()` is called only for `KindOwned` (T-068 decision 5). But `cli-reference.adoc` explicitly
recommends keeping your own `pre-commit` hook and chaining the guard into it:

```sh
pickle hooks run pre-commit || exit 1
```

That configuration has **exactly the same** PATH-skew exposure — the chained line resolves
`pickle` from `PATH` too — yet `hooks status` and `doctor` both report the foreign hook and say
nothing about whether the guard behind it can run. Decision 5's reasoning ("PATH reachability is
only a defect when pickle's own shim is armed") is sound for `KindAbsent`, but a foreign hook may
or may not chain the guard and pickle cannot tell without reading it.

Refinement must decide whether that is worth acting on at all, and if so how: grep the foreign
hook for `hooks run` before probing (cheap, slightly presumptuous), probe unconditionally for
`KindForeign` and word the line as conditional, or close this as won't-do and instead say the
limitation out loud in the `hooks status` foreign-hook line. A defensible outcome is **no code
change plus one sentence of docs**.

### 3. Nothing pins "warning, not error" at unit level (R11)

`TestCheckHooksProbesPATH/incapable pickle on PATH warns and is inert`
(`internal/doctor/hooks_test.go`) asserts the warning exists but never asserts `res.Errors` is
empty — so a future change promoting the inert-guard finding to an *error* (which would make
`pickle doctor` exit non-zero, and could fail a user's CI over their local `PATH`) would pass the
suite. Only the T-068 acceptance transcript covers it, and transcripts are not run in CI. One
assertion closes it.

### Soft couplings

- **T-068** — lineage (`spawned-by`); it shipped the probe these three items tighten. No
  dependency: each item is independent of anything T-068 might still do.
- **T-046** (self-host-aware doctor) — edits the same `checkHooks` branch. Sequence, do not run
  concurrently, exactly as T-046 and T-068 were sequenced.
- **T-067** (docs link/anchor validation) — if item 2 lands as a docs-only sentence, it touches
  `cli-reference.adoc`'s `[#cmd-hooks]` section that T-068 extended.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-06 — created (TO DO). source: pickle ticket new
- 2026-08-06 — filed from the **T-068 review**, batching its three non-blocking probe-reach
  findings by theme (R3 exit-1 semantics, R5 chained foreign hooks, R11 a missing
  warning-not-error assertion) rather than one ticket per finding. R3 is the one with teeth and
  was reproduced with the shipped binary: `TMPDIR` inside a pickle project on a `feat/` branch
  with staged `tickets/` makes `doctor` declare a *working* guard inert. Graded low-medium/low/S:
  one function plus its tests, and item 2 may legitimately resolve to a single docs sentence
- 2026-08-07 — **noted, not folded in — a fourth probe-reach gap, of a kind this ticket cannot
  fix: the diagnostic ships only in the binary you do not have.** Observed on `main` while
  committing bookkeeping: shim v2 calls `pickle hooks run`, the `PATH` binary is Homebrew 0.2.2
  which predates the `hooks` subcommand, and the shim degraded exactly as designed — warn, `exit
  0`, guard inert on every commit. T-068's `checkHooks` inert-warning and `internal/hook.Probe()`
  would both diagnose this precisely, and **neither runs**, because they live in the binary the
  user does not have on `PATH`; `pickle doctor` (0.2.2) reported `0 error(s), 1 warning(s)` and
  said nothing about the hook. So the probe's reach has a bootstrap floor: it cannot warn a user
  whose `pickle` predates the probe. Recorded here because R3 is the adjacent finding, but it is
  **not in this ticket's scope** — no change to `Probe()` reaches a binary that lacks it. The only
  real remedies are shim-side (have the shim version-check, since *it* is refreshed by `upgrade`)
  or release-side, and both are bigger than "harden the probe". Do not silently absorb it;
  refinement should either scope it out explicitly or spawn it
- 2026-08-12 — patched by **T-046's review impact sweep**: the stated coupling above is
  **discharged, and was overstated**. T-046 has landed and did **not** touch `checkHooks` at all —
  its only `internal/doctor/doctor.go` edits were `Check`, `checkSkill` and `checkVersion` (its
  decision D6 explicitly protected every hook branch, and it added
  `TestCheckSelfHostLinkStillReportsIncapablePATHPickle` to `internal/doctor/hooks_test.go` to pin
  that the self-host skip never mutes the inert-`PATH` warning). So there is no "same `checkHooks`
  branch" to sequence around: this ticket's ground is untouched, and the only overlap left is the
  file itself plus that one new test living beside `TestCheckHooksProbesPATH`, which this ticket
  should expect to see when it edits the probe's assertions
- 2026-08-14 — patched by **T-082's review impact sweep** (step 8): T-082 generalized
  `internal/hook` to a `Name`-keyed set and rewrote `doctor`'s `checkHooks` around
  `hook.StatusAll`, which touches two of this ticket's assumptions without changing its scope.
  (1) The warning this ticket quotes verbatim — `hooks: …/.git/hooks/pre-commit is installed and
  current, but the pickle on PATH cannot run the guard …` — **no longer names a hook path**: the
  probe now runs once after the per-hook loop, so the text is `hooks: installed and current,
  but …`. The quotation stays accurate as the T-068 measurement it records; treat it as history,
  not as a string to match against. T-082's own review filed a follow-up wording finding (F9) on
  that same line, so re-read it after T-082's rework lands rather than trusting either form.
  (2) Item 2 (a chained foreign hook is never probed) now spans **two** hooks — `cli-reference.adoc`
  recommends `pickle hooks run pre-push "$@" || exit 1` alongside the `pre-commit` chain line — so
  whatever this ticket does about `KindForeign` should be decided for the set, not for one name.
  `probeCapable` itself is **unchanged and still probes with `pre-commit`** (T-082 decision 6
  deliberately kept it out of this ticket's function), so no collision in either order. Scope and
  grade unchanged
