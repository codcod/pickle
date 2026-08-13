---
id: T-100
title: pre-push guard reads the wrong side of a refspec: decide the branch from the push's destination ref
project: pickle
depends-on: []
spawned-by: [T-082]
impact: medium
complexity: low
cost: S
---

# T-100 — pre-push guard reads the wrong side of a refspec: decide the branch from the push's destination ref

## Outcome

After this ships, the `pre-push` guard decides *which branch is being pushed* from the push's
**destination** ref rather than its source, so a push whose destination is a feature branch is
checked whatever refspec spelled it, and a push whose destination is the base branch is never
refused. The remaining hook-surface wording and test gaps that T-082's two reviews recorded but
deliberately left open close alongside it.

## Description

**Spawned by T-082's scoped re-review (2026-08-14), which is where the evidence below was
measured.** T-082 shipped the `pre-push` bookkeeping guard; its first review found that
`git push <remote> HEAD:refs/heads/feat/T-NNN-x` escaped the guard entirely, and the rework
fixed that one spelling by falling back to the push's destination ref (`RemoteRef`) when the
source ref (`LocalRef`) is the literal `HEAD`. That fix is correct and is shipped. This ticket
is about the *precedence* it left in place.

### 1. The branch is decided from the wrong side of the refspec

`branchBeingPushed` (`internal/hook/prepush.go`) tries `LocalRef` **first** and only falls back
to `RemoteRef`. But the invariant the guard enforces is about what a **merge request built from
the destination branch** would carry, so the destination ref is the semantically correct input
in every case, not just the one where the source does not resolve. Two spellings are decided
wrongly today, both measured against the shipped binary in a throwaway clone:

| push | destination | today | should be |
|---|---|---|---|
| `git push origin main:refs/heads/feat/T-900-x` (local `main` carrying unpushed bookkeeping) | a feature branch | **allowed** — `LocalRef` resolves to `main`, which is not a feature branch, so the ref is skipped before any range is measured | refused: an MR from `feat/T-900-x` would carry those `tickets/` paths |
| `git push origin feat/T-901-x:refs/heads/main` (feature branch carrying bookkeeping) | the base branch | **refused** | allowed: the base branch is bookkeeping's correct destination (T-082 decision 3) |

The first is a **false pass** — the failure direction T-082 decision 5 says the design must
never take. The second is a **false refusal**, which decision 5 does sanction as the safe
direction, but here it is refusing the one destination the rule exists to send bookkeeping *to*,
which is a different thing from erring safe on a stale ref.

Both predate the rework — the original implementation read `LocalRef` only, so neither is a
regression the rework introduced; the rework simply made the precedence question visible. The
likely fix is to prefer `RemoteRef` and fall back to `LocalRef`, which resolves all four
spellings (the two above, the ordinary push, and the `HEAD:` push T-082 already fixed) the same
way a forge would. **A tag push must stay skipped** — `refs/tags/...` on both sides — and that
needs a test in both precedence orders.

Refinement should also decide whether `CheckPreCommit`'s branch test has any analogous exposure
(it reads `symbolic-ref HEAD`, so probably not) and whether the rejection message's `range:`
line should name the destination branch rather than the source when the two differ.

### 2. Three items T-082's reviews recorded and left open

All in the same hook surface, none independently schedulable, all cheap once the file is open:

- **The degraded stderr line has two names** (T-082 F6, `noted`). The installed shims say
  `pickle: <hook-name> guard skipped (…)`; the binary's own five call sites in
  `internal/cli/hooks.go` plus `prepush.go`'s unresolvable-base line still say
  `pickle: bookkeeping guard skipped (…)`. `cli-reference.adoc` had to be reworded to quote the
  shared shape `pickle: … guard skipped (…)` because neither form covers both. Unify on one.
- **`doctor`'s PATH-capability pass line lost its antecedent** (T-082 F9, `noted`). It reads
  `hooks: the pickle on PATH can run it` — "it" used to be the single hook path the line named,
  which the per-hook loop now prints separately. Name the guards or the resolved binary.
- **Two `pre-push` test gaps** (T-082 F8, `noted`). `internal/hook/prepush_test.go` covers
  neither a linked worktree (`hook_test.go` covers that shape for `pre-commit`, and both rules
  share the `gitHere`-not-`gitAt` constraint that makes it matter) nor `pushRefFor`'s unused
  `dir` parameter, which should just go.

### Soft couplings

- **T-082** — lineage (`spawned-by`); shipped the guard, both reviews, and every measurement
  above. Not a dependency: this ticket is coherent against the shipped guard as it stands.
- **T-071** — hardens `Probe()`/`probeCapable` and `doctor`'s hook reporting. Item 2's `doctor`
  wording line sits in the same function T-071 reworks (`checkHooks`), so whichever lands second
  should re-read the other's text. No ordering requirement beyond that.
- **T-042** — item 5 of that epic collapses the offender-scan duplication across
  `CheckPreCommit`/`CheckPrePush`; this ticket edits `CheckPrePush`'s branch test, not the scan,
  so the two do not collide.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-14 — created (TO DO). source: review: filed from T-082's scoped re-review. Item 1 is
  a correctness hole measured against the shipped binary (a push whose destination is a feature
  branch escapes the guard when the source ref is the base branch); items 2 are the three
  non-blocking findings T-082's two reviews `noted` in the same subsystem, promoted here by
  citing those rows rather than minting a ticket each. Batched by theme (the pre-push guard's
  ref handling and the hook surface's remaining wording/test polish) per rules §5. Not folded
  into T-071, whose scope is the PATH probe rather than the rule, nor into T-082, which is
  concluding
- 2026-08-14 — graded medium/low/S against the backlog's comparable hardening tickets (T-071
  and T-070 are both low-medium/low/S). Impact a step above theirs because item 1 lets a real
  violation through the guard rather than misreporting a healthy one; complexity and cost below
  `ticket new`'s medium/medium/M defaults because item 1 is a one-function precedence flip plus
  tests and items 2 are three small edits in files the same change already opens
