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

Both questions refinement left open are now closed (see the plan's Confirmed design decisions):
`CheckPreCommit` has **no** analogous exposure — it reads `symbolic-ref HEAD` and never sees a
refspec — and the rejection message's `range:` line does need to change, because after the flip
it would otherwise name a destination branch that need not exist locally while the range actually
measured is `<base>...<LocalSHA>`.

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

### 0. Feature branch (mandatory)

```
cd .                                   # `pickle` is the root-path child
git checkout main
git checkout -b feat/T-100-prepush-destination-ref
```

WIP commits encouraged. **This repo is a root-path child** (`path = "."`), so the Finish step
tidies WIP into atomic commits and keeps that history rather than squashing. Ticket and board
bookkeeping stays on `main` — never on this branch, which is the very rule the code under edit
enforces, and this branch will be pushed through that guard.

### Prerequisite gate (hard)

None. `depends-on:` is empty and the ground this ticket stands on is shipped and merged: T-082
(the guard, both reviews, PR #45 on `main`). T-071 is still in `1-to-do/` and reworks
`probeCapable`/`checkHooks`; decision 8 keeps this ticket's `doctor` edit to a single string so
the two do not collide in either order. T-042 touches the offender scan, not the branch test.

### Confirmed design decisions (do not deviate without asking)

1. **The destination ref decides the branch, with no fallback.** `branchBeingPushed` reads
   `RemoteRef` **only**. The invariant is about what a merge request *built from the destination
   branch* would carry, so the destination is the semantically correct input in every case — not
   merely the rescue case for `HEAD:`. Do **not** implement "prefer `RemoteRef`, fall back to
   `LocalRef`": git always hands the hook a fully-qualified `RemoteRef` on stdin, so the fallback
   could only ever fire on a mixed refspec such as
   `git push origin refs/heads/x:refs/tags/v1` — where it would resurrect a **tag** destination as
   a branch check and break decision 2. A fallback here can only re-introduce a wrong answer.
2. **A push whose destination is not under `refs/heads/` is skipped, whatever the source says.**
   Tags stay outside this guard's concern in *both* precedence orders and in the mixed refspec
   above.
3. **Pushing the base branch is never refused, whatever the source says.**
   `git push origin feat/T-901-x:refs/heads/main` is allowed: the base branch is bookkeeping's
   correct destination (T-082 decision 3). This is the guard's single most important case and it
   keeps its dedicated regression test.
4. **`branchBeingPushed` keeps its `(string, bool)` shape** and stays the one place the question
   is answered. Do not inline the `CutPrefix` into `CheckPrePush`.
5. **The measured range does not change.** It stays `<base>...<ref.LocalSHA>` — three dots, the
   forge's MR-diff form (T-082 decision 4). Only *which ref decides whether to measure at all*
   moves. The all-zero-`LocalSHA` deletion skip stays ahead of the branch test, unchanged.
6. **The `range:` line prints what was actually diffed.** `branch:` names the destination branch;
   `range:` prints `<base>...<branch>` when `LocalRef` names that same branch, and
   `<base>...<short LocalSHA>` when the two sides differ. Readable in the ordinary case, honest in
   the split-refspec case this ticket exists for. Nothing else in `writePushRejection` changes.
7. **The degraded stderr line unifies on the per-hook name**: `pickle: <hook-name> guard skipped
   (…)`, matching what the installed shims already emit. The five `bookkeeping guard skipped` sites
   in `internal/cli/hooks.go` and the one in `prepush.go` adopt it — six in total.
   **`Shim()` is not touched and `ShimVersion` is not bumped** — the shims are already correct, and
   a bump would mark every installed hook stale for a change that does not affect them.
8. **`doctor`'s PATH-capability line becomes
   `hooks: the pickle on PATH can run the installed guards`.** Plural, matching the per-hook loop
   that prints above it. Confine the edit to that one string; `checkHooks`'s structure is T-071's
   scope.
9. **`CheckPreCommit` is not changed.** It resolves the branch with `symbolic-ref HEAD`, never
   sees a refspec, and therefore has no analogous exposure. This was verified at refinement; it is
   a closed question, not an invitation to re-audit.
10. **The fail-open contract is inherited verbatim** (T-082 decision 2). Only exit 1 means
    violation; every undecidable state still returns ok.
11. **Self-modify policy** (`AGENTS.md`): never run `pickle install|upgrade|uninstall` against
    this repo from this branch. Any manual verification goes to a throwaway dir with the binary
    copied in and renamed `pickle-test`.

### Tasks

#### Task 1 — decide the branch from the destination ref

`internal/hook/prepush.go`:

- Rewrite `branchBeingPushed` to read `RemoteRef` only:
  ```go
  func branchBeingPushed(ref PushRef) (string, bool) {
  	return strings.CutPrefix(ref.RemoteRef, refsHeadsPrefix)
  }
  ```
- Replace its doc comment. The current one explains a *fallback*; the new one must explain the
  **precedence**: the guard's question is what an MR built from the destination would carry, so
  the destination ref is the input in every case. Record the two spellings this fixes (a
  base-branch source pushed to a feature-branch destination was a false pass; a feature-branch
  source pushed to the base branch was a false refusal), note that both predate T-082's rework,
  and state why there is no `LocalRef` fallback (decision 1 — it could only fire on a mixed
  refspec, where it would wrongly treat a tag destination as a branch).
- Leave `CheckPrePush`'s loop otherwise untouched: the deletion skip, `onFeatureBranch`, base
  resolution, the scan, and the three-dot range are all unchanged.

#### Task 2 — make the rejection's `range:` line name what was diffed

`internal/hook/prepush.go`, `writePushRejection`:

- Pass the information needed to tell the two cases apart — the simplest shape is to give
  `writePushRejection` the `PushRef` (or a precomputed `rangeRHS string`) instead of adding two
  more positional strings to an already seven-argument function; prefer whichever keeps the call
  site readable.
- Print `range:   <base>...<branch>` when `ref.LocalRef == refsHeadsPrefix+branch`, and
  `range:   <base>...<short LocalSHA>` otherwise (7 chars is git's default short form; guard the
  slice against a shorter string). The `branch:` line continues to name the destination branch.

#### Task 3 — unify the degraded stderr wording

- `internal/cli/hooks.go`: the three sites in `runHooksRunPrePush` (~298, ~306, ~311 — the first
  is the `hookConfig` error site, already inside that function) and the two in
  `runHooksRunPreCommit` (~267, ~275) — five in total — become `pickle: %s guard skipped (%v)`
  with `hook.PrePush` / `hook.PreCommit` respectively. Each function knows its own name — do not
  thread a parameter through `hookConfig`; format at the call site.
- `internal/hook/prepush.go:139` (the unresolvable-base line) becomes
  `pickle: pre-push guard skipped (%v)` — six sites in total across the two files.
- Grep afterwards: `rg 'bookkeeping guard skipped' --glob '!CHANGELOG.md' --glob '!tickets/'`
  must print nothing. `CHANGELOG.md:332` is a historical entry for a shipped release and stays as
  written.

#### Task 4 — give `doctor`'s PATH-capability line an antecedent

- `internal/doctor/doctor.go:327`: `r.ok("hooks: the pickle on PATH can run the installed guards")`.
- `internal/doctor/hooks_test.go`: update the two assertions that match on `"can run it"`
  (lines ~240 and ~290) to the new substring. Match on something stable — `"can run the installed
  guards"` — not on the whole sentence.

#### Task 5 — close the two `pre-push` test gaps

`internal/hook/prepush_test.go`:

- **Drop `pushRefFor`'s unused `dir` parameter** (T-082 F8) and update its five call sites
  (~173, ~238, ~256, ~280, ~308). Note that the call at ~308 passes `child` while loading the
  config from `base` — that distinction lives in the `loadConfig` argument, not in `pushRefFor`,
  so dropping the parameter loses nothing; confirm that reading before deleting.
- **Add a linked-worktree case**, mirroring the `pre-commit` shape in `hook_test.go` (~780–801):
  `git worktree add -q -b feat/T-001-x <wt>`, commit a `tickets/` path there, `t.Chdir(wt)`, and
  assert `CheckPrePush` refuses. This matters for the same reason it does for `pre-commit`: both
  rules run through `gitHere`, not `gitAt`, and a hardcoded root would inspect the main
  worktree's repository instead of the one being pushed.

#### Task 6 — pin the new precedence with tests

`internal/hook/prepush_test.go`, extending `TestBranchBeingPushed` and `TestCheckPrePush`:

- `TestBranchBeingPushed` table gains, alongside its four existing rows:
  - `main:refs/heads/feat/T-1-x` → `feat/T-1-x`, ok (the false pass, table row 1 of the
    Description).
  - `feat/T-1-x:refs/heads/main` → `main`, ok — and therefore skipped downstream by
    `onFeatureBranch`, not by `branchBeingPushed` (the false refusal, table row 2).
  - `refs/heads/x:refs/tags/v1.0.0` → not ok. **This is the row that fails under the rejected
    fallback design** and is the test decision 1 exists to protect; say so in a comment.
  - The existing `HEAD:refs/tags/...` and both-sides-`refs/tags/` rows must still pass
    unchanged — decision 2 in both precedence orders.
- `TestCheckPrePush` gains two end-to-end cases against a real fixture repo, both asserting on
  behaviour rather than on `branchBeingPushed` directly:
  - **destination is a feature branch, source is `main`** carrying unpushed bookkeeping → refused,
    and the rejection names `feat/T-900-x`, not `main`. Assert on the `branch:` line.
  - **destination is `main`, source is a feature branch** carrying bookkeeping → allowed. Comment
    it as decision 3: this is the destination the rule exists to send bookkeeping *to*.
  - The second case needs a `PushRef` the existing `pushRefFor` cannot build; add a small
    `pushRefTo(branch, dest, head string) PushRef` helper (or inline the literal) rather than
    overloading `pushRefFor`.

#### Task 7 — docs

See the Docs update step below; it is a task, done on this branch.

### Acceptance test

All four run from the repo root and must be green:

```
just build
just test
just lint
just docs-check
```

Plus the two behavioural checks this ticket exists for, in a throwaway clone (never against this
repo — decision 11), with the freshly built binary copied in as `pickle-test`:

```
D=$(mktemp -d) && cp pickle "$D/pickle-test"
# in $D: a bare remote + a clone with pickle.toml, tickets/ on main, hooks armed via ./pickle-test

# 1. false pass, now refused: destination is a feature branch, source is main
git push origin main:refs/heads/feat/T-900-x     # => refused; message names feat/T-900-x
                                                 #    and a range ending in a short SHA

# 2. false refusal, now allowed: destination is the base branch
git push origin feat/T-901-x:refs/heads/main     # => allowed

# 3. unchanged: ordinary base-branch push still allowed, ordinary feature-branch push
#    carrying tickets/ still refused, HEAD:refs/heads/feat/... still refused, tag push silent
```

And the wording sweep:

```
rg 'bookkeeping guard skipped' --glob '!CHANGELOG.md' --glob '!tickets/'   # no matches
rg 'can run it' internal/                                                  # no matches
```

### Docs update (mandatory when user-facing)

User-facing: the guard's rule changes and two stderr strings change.

- `docs/user-manual/cli-reference.adoc`
  - `:378` — "refuses a push, **from** a feature branch, whose range…" is now wrong. Reword to the
    destination: a push **whose destination is** a feature branch. Add one sentence saying the
    branch is decided from the push's destination ref, so `HEAD:refs/heads/feat/…` and
    `main:refs/heads/feat/…` are judged the same way a forge would judge the resulting MR.
  - `:392` — "Pushing the base branch itself is never refused" is still the headline promise, but
    it now holds for *any* source ref. Make that explicit; it is precisely what was broken.
  - `:499` — the shape was hedged to `pickle: … guard skipped (…)` only because the two forms
    disagreed. Replace with the now-exact `pickle: <hook> guard skipped (…)`.
- `skill/resources/tickets-README.md:95–96` — "a `pre-push` hook that refuses a push of that
   branch whose range…" → phrase it as the push's **destination** being that branch. Keep it to
   the rule; no pickle-internal paths, ids or counts (the foreign-workspace test in `AGENTS.md`).
- `CHANGELOG.md` — one Unreleased entry under a `### Fixed` heading: the guard decided the branch
  from the wrong side of a refspec, so a push whose destination was a feature branch could escape
  it and a push of bookkeeping to the base branch could be wrongly refused. Do **not** edit the
  shipped v0.8.0 entries, including the historical `bookkeeping guard skipped` string at `:332`.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated as above.
3. Write a **summary** of everything done (files touched, decisions made, anything deferred).
4. Suggest a **Conventional Commit message**, e.g.:

   ```
   fix(hooks): decide the pushed branch from the destination ref (T-100)

   <body — what and why>
   ```

5. **Tidy up before presenting** — this repo is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed commits (a natural split: the
   precedence fix + its tests; the wording unification; the test-gap closures). Keep that history
   rather than squashing.
6. Commit locally on the ticket branch. Do **not** push or open a merge request without explicit
   user approval. Present the commit messages; after approval, verify the remote base is not
   behind (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`
   must print nothing), then push and open the merge request — merging is the human's. Then
   `pickle ticket move T-100 in-review --reason "acceptance green"` and hand back.

## Review

**2026-08-14 — full review. Verdict: pass, no blocking findings.** Reviewed at
`fcf79f8` on `feat/T-100-prepush-destination-ref` (three implementation commits
`97ab532`/`b17bfd3`/`8829b55`, plus this review's own inline-fix commit).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b)
- [x] Findings recorded with severity, class and disposition; summary + cost line below (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages & MR attributes presented for approval (step 9)

### Implementation audit

All seven tasks **met**; all eleven confirmed decisions **honoured**.

- **Task 1** — `branchBeingPushed` is now `strings.CutPrefix(ref.RemoteRef, refsHeadsPrefix)`
  with no `LocalRef` branch (`internal/hook/prepush.go:106-108`), and its doc comment explains
  the precedence, names both mis-decided spellings, notes both predate T-082's rework, and
  states why there is no fallback. `CheckPrePush`'s loop is otherwise untouched.
- **Task 2** — `writePushRejection` takes the `PushRef` (decision 6's "keep the call site
  readable" option) and prints `range: <base>...<branch>` only when
  `ref.LocalRef == refsHeadsPrefix+branch`, else `<base>...<shortSHA>`; `shortSHA` guards a
  string shorter than 7.
- **Task 3** — five sites in `internal/cli/hooks.go` now format from `hook.PreCommit`/
  `hook.PrePush`; `prepush.go`'s unresolvable-base line joins them. Sweep clean (below).
  `hook.go` is untouched, so `Shim()` and `ShimVersion = 3` are unchanged (decision 7).
- **Task 4** — `internal/doctor/doctor.go:327` and both `hooks_test.go` assertions updated,
  matching on `can run the installed guards` only (decision 8: `checkHooks`'s structure is
  untouched, so T-071 does not collide in either order).
- **Task 5** — `pushRefFor`'s `dir` dropped at all five call sites;
  `TestCheckPrePushInALinkedWorktree` added, mirroring `hook_test.go`'s `pre-commit` shape.
- **Task 6** — `TestBranchBeingPushed` gains the three rows the plan names, including the
  tag-destination row that fails under the rejected fallback design; `TestCheckPrePush` gains
  both end-to-end cases via a new `pushRefTo` helper, asserting on the `branch:` and `range:`
  lines rather than on `branchBeingPushed`.
- **Task 7 / docs** — `cli-reference.adoc:378`, `:396`, `:504` and
  `skill/resources/tickets-README.md:95-97` reworded as specified;
  one `### Fixed` entry under `[Unreleased]`; the shipped v0.8.0 entries, including the
  historical string at `:332`, untouched.

**Acceptance test, re-run verbatim:** `just build`, `just test` (15 packages, all `ok`),
`just lint`, `just docs-check` — all green, before and after this review's inline fixes.

**Behavioural checks**, in a throwaway clone with the freshly built binary copied in as
`pickle-test` (per `AGENTS.md`; driven through `pickle-test hooks run pre-push`, which is what
the shim invokes — the shim itself resolves the *name* `pickle`, which on this machine is the
released v0.8.0 and therefore served as a control):

| push | old binary (v0.8.0) | new binary |
|---|---|---|
| `main:refs/heads/feat/T-900-x` carrying `tickets/` | allowed (the false pass) | **refused**, `branch: feat/T-900-x`, `range: …origin/main...14a84d9` |
| `feat/T-901-x:refs/heads/main` over the same non-empty offending range | — | **allowed** |
| ordinary `feat/…` push carrying `tickets/` | — | refused, `range: …origin/main...feat/T-901-x` |
| `HEAD:refs/heads/feat/…` | — | refused, range falls back to the short SHA |
| ordinary base push | — | allowed |
| `refs/tags/v1:refs/tags/v1` and `refs/heads/x:refs/tags/v1` | — | allowed, silent |

The second row is a real test, not a vacuous one: the third row refuses the *same* SHA over the
*same* range, so the allow is caused by the destination being the base branch and nothing else.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | stale-xref | fixed inline | `CheckPrePush`'s own doc comment still stated the pre-T-100 rule — "refuse a push whose **local ref** is a feature branch" — fifteen lines below the rewritten `branchBeingPushed` comment that inverts it | `internal/hook/prepush.go:110` (pre-fix) | reword to the destination ref and point at `branchBeingPushed` for the why — done in `fcf79f8` |
| F2 | non-blocking | stale-xref | fixed inline | `TestBranchBeingPushed`'s pre-existing prose still described a `LocalRef` fallback in four places (function doc "pins the fallback finding F2"; row name "ordinary push: LocalRef names the branch"; row name "HEAD:refs/heads/… push: falls back to RemoteRef"; row 4's name and comment "fallback still outside refs/heads/") — three lines above a *new* row whose comment asserts there is no fallback | `internal/hook/prepush_test.go:133,143,150,163` (pre-fix) | restate all four against the destination-ref rule, keeping the F2 provenance — done in `fcf79f8` |
| F3 | non-blocking | stale-xref | fixed inline | Two manual pages outside the plan's docs list still summarised the guard from the source side, which this branch made false at the margin (a feature branch pushed *to the base* is now allowed): "refuses pushing such a branch when its range…" and "the range a feature-branch push carries against the remote base" | `docs/user-manual/installation.adoc:62-64`, `docs/user-manual/concepts/project-structure.adoc:160-162` (pre-fix) | reword both to the destination — done in `fcf79f8`. The authoritative page (`cli-reference.adoc`) was already correct, which is why this is not a coverage gap |
| F4 | non-blocking | other | noted | The plan's own acceptance sweep `rg 'can run it' internal/` is over-broad and does **not** come back empty: it matches `internal/hook/probe.go:77`, a pre-existing comment reading "can run **its** own guard" | `rg 'can run it' internal/` → one hit, present on `main` too | the sweep's *intent* (no `doctor` line reading "can run it") is met; a future plan writing this check should anchor it, e.g. `rg "can run it\"\|can run it$"` |
| F5 | non-blocking | design | fixed inline | The wording rule decision 7 unifies was itself spelled two ways: `internal/cli/hooks.go` formats the line from `hook.PreCommit`/`hook.PrePush`, while `prepush.go` — inside package `hook`, where the constant is in scope — hardcoded the literal `"pre-push"` | `internal/hook/prepush.go:152` (pre-fix) | format from `PrePush` there too, so the one file that owns the constant uses it — done in `fcf79f8` |

disposition summary: 5 non-blocking findings — 4 fixed inline (F1, F2, F3, F5), 1 noted (F4);
0 blocking, 0 folded, 0 new tickets.

cost: estimated S, actual S

### Docs-readability pass (step 4b)

Run over the three changed prose files (`docs/user-manual/cli-reference.adoc`,
`skill/resources/tickets-README.md`, `CHANGELOG.md`). Sixteen suggestions returned; fourteen
land on prose this branch never touched and were discarded as out of scope for the ticket. The
two that touch changed prose — splitting the `[Unreleased]` entry's problem and fix into
separate clauses, and shortening the `pre-push` bullet's closing sentence — were read and
declined: the entry's paired before/after spellings are the point of the fix, and the bullet's
long sentence is pre-existing text this branch only re-wrapped. Readability suggestions never
enter the findings table and never block (protocol §4b).

### Consistency & quality notes (no finding)

- `writePushRejection` now carries both `branch` and the `PushRef` it was derived from. That is
  redundant on paper, but decision 4 keeps `branchBeingPushed` the single place the question is
  answered, so re-deriving the branch inside the writer would be worse. Left as is.
- `pushRefFor` keeps its `t *testing.T` for `t.Helper()` while the new `pushRefTo` takes none.
  Deliberate and correct — `pushRefTo` cannot fail.
- The remedy block still suggests `git push origin main` on a split refspec, where the offending
  commits are already on the local base. The advice is still the right one (push the base first);
  no change.
- `internal/hook/hook.go` is untouched, so no installed shim goes stale (decision 7 verified,
  not merely asserted).

### Impact sweep (step 8)

No ticket lists T-100 in `depends-on:`. One ticket references it in prose: **T-071**
(`1-to-do/`), whose History already anticipated both landing orders for the `doctor`
PATH-capability string. T-100 landed first, so T-071 has been patched with a History line
recording the string's new text. `pickle board audit`: 103 tickets, 0 errors, 0 warnings.

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
- 2026-08-14 — refined. Both Description-level open questions closed: `CheckPreCommit` has no
  analogous exposure (verified — it resolves the branch with `symbolic-ref HEAD` and never sees a
  refspec), and the rejection's `range:` line does change. The precedence itself was decided
  *against* the Description's own suggestion: destination-only, **no `LocalRef` fallback**, because
  git always sends a fully-qualified `RemoteRef` and the fallback could therefore only fire on a
  mixed refspec (`refs/heads/x:refs/tags/v1`), where it would treat a tag destination as a branch
  and break the tag skip the ticket requires. Nothing split out — none of the four items is
  independently schedulable (rules §3); all live in the same two files. Grades held at
  medium/low/S: refinement confirmed the scope the grading assumed rather than moving it
- 2026-08-14 — TO DO → READY: plan complete
- 2026-08-14 — plan amended inline: applicability audit at pickup found Task 3 and decision 7
  overcounted the `bookkeeping guard skipped` call sites ("four... plus the hookConfig one" /
  "six" in `internal/cli/hooks.go` alone) — the true count is five in that file (three in
  `runHooksRunPrePush`, two in `runHooksRunPreCommit`, the first of the three *being* the
  hookConfig site, not additional to it) plus one in `prepush.go`, six total across both files.
  Line numbers were already correct; only the prose arithmetic was off. No other findings
- 2026-08-14 — READY → IN DEVELOPMENT: picked up
- 2026-08-14 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-14 — IN REVIEW → DONE: review passed: 5 non-blocking findings (4 fixed inline, 1 noted), 0 blocking, 0 new tickets
