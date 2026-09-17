---
id: T-128
title: doctor: warn when a feature branch's ticket file is stale relative to the base branch
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-128 — doctor: warn when a feature branch's ticket file is stale relative to the base branch

## Outcome

After this ships, `pickle doctor` on an in-tree project checks out on a `feat/T-NNN-*` branch
reports a warning when that ticket's own file disagrees with its base-branch copy (a different
status directory, or a shorter `## History`) — the same "mirror-image hazard" `pickle serve`
already warns about (T-108), now reachable from the command an agent actually runs mid-workflow,
without needing a running `serve` process.

## Description

T-108 documented and partly fixed the in-tree "mirror-image hazard": bookkeeping only lands on
the base branch, so a feature branch cut before a later status move (or review-disposition
commit) carries a stale copy of its own ticket, and every command that reads `tickets/` reports
that stale copy without saying so. T-108 shipped the warning on `pickle serve` only, and its
Implementation Plan (decision 9) explicitly deferred the rest: "Extending the signal to other
readers is a follow-up, not part of this ticket." `tickets-README.md` §0 still states the fix as
prose discipline for readers ("read the ticket and the board from the base branch... not the
branch under review"), with nothing mechanical catching the drift outside `serve`.

**This is not theoretical — it reproduced in a real installed project.** In `messgr` (an in-tree
brine install, unrelated to pickle's own repo), a ticket's board-bookkeeping move (`IN
DEVELOPMENT` → `IN REVIEW`, "acceptance green") was committed correctly on `main` after its
`feat/T-NNN-*` branch had already been cut. The feature branch's own worktree still showed the
pre-move status and a `## History` one line short. An agent resuming work on that branch, reading
the ticket from the base branch per the documented discipline, correctly reported the ticket as
not eligible for the `implement` trigger — technically correct, but surprising and blocking,
because nothing had told it *why* its own branch disagreed with base, and the fix (rebase onto
base to pick up the bookkeeping commit) is mechanical and could have been suggested on the spot.
`doctor` is the natural place for it: it is what an agent runs to sanity-check state before
acting, it is already layout-aware (T-108's `checkLayoutInvariant`), and T-046 already made it
self-host-aware for a related in-tree quirk.

**Scope.** Detection only, on `doctor`, for the in-tree layout. `doctor` already knows the
resolved layout and the checked-out branch; it needs the base branch name (recorded or resolved
the same way `serve`'s `staleBoardBranch` gets it — no new `main`/`master` guessing per T-108
decision 8) and a way to diff one ticket file between `HEAD` and that base. The check applies
only when `HEAD` matches a registered child's `branch_prefix` (mirrors `serve`'s condition), and
only to the ticket(s) whose id the branch name carries (a `feat/T-NNN-*` branch names its own
ticket) — not a full-board diff, which is a different and heavier check `board audit` might
reasonably own some day but is out of scope here.

**Soft couplings (not `depends-on`):**

- **T-108** (done) — supplies `checkLayoutInvariant`, the recorded `layout` key, and
  `staleBoardBranch`'s base-branch resolution in `internal/serve/serve.go`; this ticket reuses
  both rather than re-deriving them.
- **T-046** (done) — established `doctor` as self-host-aware; this ticket's warning must not fire
  on pickle's own `feat/T-NNN-*` branches in ways that misfire against the self-modify policy's
  throwaway-install testing (it only reads the branch's own ticket file, not any installed
  payload, so this is expected to be a non-issue, worth confirming in refinement).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-128-doctor-warn-stale-ticket-branch
```

Root-path child (`path = "."`) — tidy WIP commits into atomic ones before presenting, per
`tickets-README.md` §0. Publish only after user approval (no push/MR without it).

### Prerequisite gate (hard)

None. T-108 and T-046 are both `6-done/`; no `depends-on:` is warranted — both couplings are
soft (reuse, not a blocker).

### Confirmed design decisions (do not deviate without asking)

1. **Shared detection logic moves to `internal/vcs`, not to a new doctor→serve import.**
   `staleBoardBranch` (`internal/serve/serve.go:348`) already does the git-plumbing half of this
   (resolve `HEAD` via `symbolic-ref`, detect detached, match a registered child's
   `branch_prefix`) — `doctor` needs the identical answer. `doctor` and `serve` do not currently
   import each other; making `doctor` depend on `serve` for one git helper would be a backwards
   layering (a filesystem/git checker depending on the HTTP UI package), so the branch-detection
   loop is extracted into `internal/vcs` (already the shared git-plumbing layer both packages use
   via `vcs.Output`) as a new exported function, and `serve.go`'s `staleBoardBranch` is rewritten
   to call it — same behaviour, one implementation.
2. **Base-branch resolution is local-only: try `refs/heads/main`, then `refs/heads/master`.**
   T-108 decision 8 deliberately never guesses a *base* branch name for `serve`'s warning — but
   that check never needed to diff anything, only to name a suspicious branch. This check must
   read the base copy of a file, so it needs an actual ref. `internal/hook/prepush.go`'s
   `resolveBase` (line 187) solves the equivalent problem for pushes by trying
   `refs/remotes/<remote>/HEAD`, then `.../main`, then `.../master` — remote-tracking refs, since
   a push always has a remote. `doctor` runs against a local worktree with no guaranteed remote
   (a fresh `pickle install --in-tree` before any `git remote add`), so it tries the equivalent
   **local** branch refs instead: `refs/heads/main`, then `refs/heads/master`. Neither resolving
   is silent (no warning, no error) — advisory checks degrade to nothing rather than a false
   positive, matching `staleBoardBranch`'s own fail-open comment.
3. **Scope to the one ticket the branch names, via `ticket.IDShapePattern`.** A
   `feat/T-NNN-<slug>` branch names its own ticket id right after the child's `branch_prefix`.
   Strip the prefix, then match `^` + `ticket.IDShapePattern` (`internal/ticket/ticket.go:66`,
   already exported) against what remains. No match (a feature branch that doesn't encode an id,
   e.g. a spike branch) → skip silently; this check has nothing to say about it.
4. **Locate the ticket on each side by filename glob / `git ls-tree`, not by trusting a shared
   path.** The whole point of the check is that the ticket's status directory can differ between
   `HEAD` and base, so the same relative path cannot be assumed to resolve on both. `HEAD`'s copy:
   `filepath.Glob(filepath.Join(root, "tickets", "*", id+"-*.md"))`. Base's copy:
   `git ls-tree -r --name-only <base> -- tickets` (via `vcs.Output`), scanned for a path whose
   basename starts with `id+"-"`. Zero or multiple matches on either side → skip silently (a
   malformed tree is `board audit`'s finding, not this check's).
5. **Two comparisons, in order, first match wins:** (a) status directory (`filepath.Dir` of the
   two paths) differs → warn naming both directories and the fix (`rebase onto <base>`) — this
   alone catches the `messgr` T-039 case. (b) directories agree but the base copy's `## History`
   section (via `ticket.SectionBody(text, "History")`, already exported, counting lines whose
   trimmed form starts with `"- "`) has strictly more entries than `HEAD`'s copy → warn that the
   base branch carries newer bookkeeping (a disposition or merge line appended without a status
   move). Equal or fewer → `r.ok(...)`, no warning. This is deliberately *not* a full-content diff
   (a legitimately amended plan on the feature branch would false-positive) — see soft coupling
   note on scope in the Description.
6. **Git failure at any step (no repo, base ref vanishes mid-check, `ls-tree`/`show` error)
   degrades to silent skip**, never an error and never a crash — same fail-open contract
   `staleBoardBranch` documents for the identical reason: this is advisory, and a broken git probe
   must not turn a clean `doctor` run into one that looks broken.

### Tasks

#### Task 1 — extract shared branch-detection into `internal/vcs`
Add `func FeatureBranchHead(root string, prefixes []string) string` to `internal/vcs/vcs.go`,
carrying the git-mechanics doc comment (symbolic-ref vs. rev-parse, the unborn-branch and
detached-HEAD cases) currently on `staleBoardBranch`. Body is `staleBoardBranch`'s existing logic
from the `symbolic-ref` call through the prefix loop, generalized to take `prefixes []string`
instead of reading `cfg.Projects` itself. Returns `""` (not a repo / no match), `"HEAD"`
(detached), or the matched branch name.

#### Task 2 — rewrite `staleBoardBranch` to call it
In `internal/serve/serve.go`, `staleBoardBranch` keeps its own doc comment (the in-tree/layout
gating rationale — that part is `serve`-specific, not raw git) but its body becomes: return `""`
if not in-tree; build `prefixes` from `cfg.Projects` (falling back to
`config.DefaultBranchPrefix` per project, as today); `return vcs.FeatureBranchHead(root,
prefixes)`. No behavior change — existing `serve` tests must still pass unmodified.

#### Task 3 — local base resolution in `internal/vcs`
Add `func ResolveLocalBase(root string) (branch string, ok bool)` to `internal/vcs/vcs.go`: tries
`rev-parse --verify --quiet refs/heads/main` then `refs/heads/master` (via `vcs.Output`),
returning the short name (`"main"`/`"master"`) and `true` on the first that resolves, `("",
false)` if neither does or git fails.

#### Task 4 — the new doctor check
Add `checkStaleTicketBranch(root string, cfg *config.Config, r *Result)` to
`internal/doctor/doctor.go`, implementing confirmed decisions 3–6 above, and call it from
`checkChildren` right after `checkLayoutInvariant(cfg, r)` (same guard shape: both are in-tree-
layout checks over `cfg.Projects`). Reuse `vcs.FeatureBranchHead`, `vcs.ResolveLocalBase`,
`vcs.Output`, `ticket.IDShapePattern`, `ticket.SectionBody`. New imports needed in
`internal/doctor/doctor.go`: `regexp`, `github.com/codcod/pickle/internal/ticket`.

### Acceptance test

- `internal/vcs`: unit tests for `FeatureBranchHead` (not-a-repo, detached HEAD, matching prefix,
  non-matching prefix — covering what `staleBoardBranch`'s existing tests already exercise
  indirectly) and `ResolveLocalBase` (main exists, master exists, neither exists).
- `internal/serve`: existing `staleBoardBranch` tests pass unmodified (behavior-preserving
  refactor).
- `internal/doctor`: new `doctor_stale_ticket_branch_test.go`, using the existing `gitInit`
  helper (`doctor_test.go:95`) to build a two-commit, two-branch fixture repo:
  - stale status directory (ticket in `3-in-development/` on the feature branch, moved to
    `4-in-review/` on `main`) → one warning naming both directories.
  - same directory, base has one more `## History` line → one warning naming the History drift.
  - identical on both sides → no warning, one passed line.
  - `layout = "umbrella"` → check does not run (no git calls made — verifiable by pointing `root`
    at a non-repo without a crash).
  - detached `HEAD` → silent, no warning.
  - branch name carries an id that resolves on `HEAD` but not on base (newly created ticket, not
    yet on base) → silent.
  Run via `just test`.

### Docs update (mandatory when user-facing)

Add one new bullet to `docs/user-manual/cli-reference.adoc` `[#cmd-doctor]` section, immediately
after the existing "the recorded `layout` ... agrees with the registered children" bullet
(currently ending "...reported as a passed line under `--verbose`;", around line 479) and before
"each registered child's `path` is a git repository": a bullet describing the new warning
(in-tree layout, `HEAD` on a registered child's feature branch, that branch's own ticket file
disagrees with its base-branch copy — status directory or `## History` length — naming the fix,
`rebase onto <base>`). Match the existing bullets' style (states the condition, the finding
severity, and — where one exists — the command it points at). Run `just docs-check` after.

### Finish (mandatory)

1. `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs bullet added and registered per above.
3. Write the summary (files touched, decisions made, anything deferred) and hand back.
4. Suggested commit message: `feat(doctor): warn when a feature branch's ticket is stale vs
   base (T-128)` — tidy the branch's WIP commits into this one atomic commit first (root-path
   child, `tickets-README.md` §0), then present it; do not push without approval.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-17 — created (TO DO). source: field-use: observed in `messgr` (an in-tree brine
  install) — a ticket's board-bookkeeping move landed on `main` after its feature branch was cut,
  leaving the branch's own worktree copy stale; an agent resuming work read the ticket from base
  per the documented "mirror-image hazard" discipline (`tickets-README.md` §0) and correctly, but
  confusingly, reported the ticket as not ready for its trigger, with no mechanical signal
  pointing at the fix (rebase onto base).
- 2026-09-17 — TO DO → READY: plan complete
- 2026-09-17 — READY → IN DEVELOPMENT: picked up
- 2026-09-17 — IN DEVELOPMENT → IN REVIEW: acceptance green
