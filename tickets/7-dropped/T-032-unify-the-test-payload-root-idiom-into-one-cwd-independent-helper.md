---
id: T-032
title: unify the test payload-root idiom into one CWD-independent helper
project: pickle
depends-on: []
spawned-by: [T-029]
impact: low
complexity: low
cost: S
---

# T-032 — unify the test payload-root idiom into one CWD-independent helper

## Description

> **ABSORBED into T-042 (board triage, 2026-07-26) — this ticket is closed, its work is not.**
> Everything below stands as the record: the analysis, measurements and line references
> are still the authoritative detail for this part of T-042's scope. Do not re-file it;
> do not implement from here. T-042 is the refinable, reviewable unit.

Four test packages need the same thing — the directory whose `os.DirFS` exposes the install payload
(`skill/SKILL.md` et al.) at the same paths the binary's embedded `payloadFS` uses, i.e. the repo
root. They express it three different ways, and T-029 added the third:

| site | idiom |
|---|---|
| `internal/install/install_test.go:15` | `func payloadRoot() string { return filepath.Join("..", "..") }` |
| `internal/doctor/doctor_test.go:14` | the same function, duplicated verbatim — its comment even says "mirrors install_test.go" |
| `internal/move/move_test.go:20` | inlined `os.DirFS(filepath.Join("..", ".."))` |
| `internal/sync/sync_test.go:21` | inlined `os.DirFS(filepath.Join("..", ".."))` |
| `internal/cli/cli_test.go:19` (new in T-029) | package-level `repoRoot`, absolute, computed in `TestMain` as `filepath.Abs(filepath.Join(wd, "..", ".."))` |

All five are the same fact written five times, and the first four are **relative to the process
CWD** — they work only because `go test` happens to start each package in its own source directory.
T-029 discovered this the hard way: introducing a `TestMain` that chdirs package `cli` into a
sandbox broke `newProject`'s relative payload path, which is why that package now needs an absolute
`repoRoot` and a comment warning that "nothing in this package may reach for the payload with a
relative path". The trap is still armed in the other four: the moment anyone adds a chdir to
`install`, `doctor`, `move`, or `sync` tests, their payload resolution silently starts reading the
wrong directory.

### Scope

Replace all five with one shared, CWD-independent helper — a small `internal/testroot` package
exposing `Dir() string` resolved from `runtime.Caller(0)` (the file's own compile-time path), which
is correct regardless of the process CWD and regardless of which package imports it. Because only
`_test.go` files import it, it is never linked into the shipped binary. Then delete
`install_test.go`'s and `doctor_test.go`'s `payloadRoot()`, replace both inlined
`filepath.Join("..","..")` calls, and have `internal/cli`'s `TestMain` use it instead of computing
`wd/../..` before the chdir.

Points refinement must settle:

- **Whether `internal/testroot` is the right shape.** A package whose only consumers are tests but
  which is not itself a `_test.go` file is unusual in a small repo; the alternative is a
  `runtime.Caller`-based helper duplicated per package (better than today — CWD-independent — but
  still duplicated). Decide explicitly, don't drift.
- **Verification that the fix is real, not cosmetic.** The load-bearing check is that each converted
  package still passes *after* a chdir is introduced into it — e.g. temporarily add a sandboxing
  `TestMain` to `internal/move` and confirm its tests still find the payload, where today they would
  not. Mirror T-029's mutation discipline: a refactor that passes identically before and after has
  demonstrated nothing.
- **Whether `internal/cli`'s `TestMain` still needs its own `repoRoot` var** once `testroot.Dir()`
  exists, or whether call sites should just call the helper directly.

### Couplings

`spawned-by: [T-029]` — finding N5 of T-029's review (T-029 made the divergence worse: 2 idioms →
3). No hard dependency, but the `internal/cli` half of the change **only exists on T-029's branch**,
so this cannot start before T-029 is merged to `main`; that is a merge-order fact, not a
`depends-on:`.

Soft couplings (no `depends-on`, no ordering enforced):

- **T-031** ("harden the internal/cli test harness") includes an item to validate `repoRoot` by
  `os.Stat`-ing `skill/SKILL.md`. This ticket **subsumes** that item by removing the computation it
  guards. Whichever lands second must reconcile: if T-031 landed first, delete its validation along
  with the `wd/../..` computation; if this lands first, T-031 drops the item.
- **T-012** ("harden test coverage + TOML-safe render") adds tests to `internal/cli` and would
  inherit whichever idiom is current. Low conflict — different helpers — but the file overlaps.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-26 — TO DO → DROPPED: absorbed into T-042 (board triage merge); content preserved here as the record
