---
id: T-108
title: pickle install --in-tree: record the board layout in config, and warn when an in-tree board is read from a feature branch
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-108 — pickle install --in-tree: record the board layout in config, and warn when an in-tree board is read from a feature branch

## Outcome

After this ships, `pickle install` no longer infers where the board lives: the umbrella layout
is the default, `--in-tree` explicitly selects the layout where the board sits inside its sole
child's own repository, and the choice is recorded as `layout` in `pickle.toml`. `pickle doctor`
reports an error when the recorded layout contradicts the registered children, and `pickle
upgrade` back-fills the key for existing projects by inference rather than requiring a migration
command. `pickle serve` states which layout it is reading and warns when an in-tree board is
being read from a non-base branch — the one situation in which the board UI can show a ticket
status that is silently out of date.

## Description

Ticket status is a single-valued fact about a project, but it is stored as a file's location
inside a git working tree. Git's purpose is to let branches disagree about file locations, so
any layout where the board lives inside a repository that gets feature branches will fork the
board, and no git mechanism can say which fork is authoritative — that is a convention, not a
property.

**pickle's primary layout does not have this problem.** In the umbrella layout the overarching
project holds `tickets/` and `pickle.toml`, and the child-projects are separate repositories
registered by path; children know nothing about the umbrella, by design. Cutting a feature
branch in a child cannot fork the umbrella's board, because the board is not in that repository.
This is the layout `internal/vcs`'s `ChildState`/`Advice` already serve, with their "nested git
repository that this repository does not ignore" guidance.

**The in-tree layout is the exception, and it accepts a known price.** When the overarching root
and the sole child are the same repository (`path = "."`, described in-code as "the single-repo
default"), the board is inside the branching medium. Tickets become visible to anyone who clones
the code, which is genuinely valuable — but bookkeeping lives on the base branch only, so any
feature branch cut before the latest bookkeeping commit carries a stale copy, and every pickle
command that reads tickets reports that stale copy without saying so.

**This is demonstrated, not theoretical.** Running `pickle serve` against a worktree checked out
at `8b4caa6` — the real tip of T-065's feature branch, after `main` had already moved that ticket
to `6-done/` and merged it — rendered:

```
GET /t/T-065  ->  <dt>status</dt><dd>IN DEVELOPMENT</dd>
GET /         ->  T-065 under the "IN DEVELOPMENT 1/1" lane
```

while `main`'s own `serve` showed `DONE` at the same instant. Identical markup, identical route,
opposite answers, and nothing on the page distinguishes them. The drift is one-directional:
because bookkeeping only lands on the base branch, a stale worktree can only under-report
progress, never falsely claim `DONE`.

**Why the layout must be recorded rather than detected.** Both facts needed to warn accurately —
which layout this is, and what the base branch is called — are currently unavailable. There is
no `layout` key at all, and the base branch is *guessed* from a hardcoded list at
`internal/hook/prepush.go:206` (`for _, name := range []string{"main", "master"}`), so a project
based on `develop` or `trunk` would be silently mis-served by any check built on that guess.
Recording the layout at install time, when the human knows the answer, replaces both guesses
with one stated fact.

**Why `--in-tree` and not `--sibling`.** In that layout nothing is a sibling: the tickets are
*inside* the code repository. "Sibling" was also used during design for the opposite arrangement
(a board directory beside the code), so the term would arrive in the manual already ambiguous.
`--in-tree` names where the tickets live.

**Scope boundary.** The warning ships on `pickle serve`, which is where the incorrect status was
actually observed. `board state --json` is deliberately left unchanged here so this ticket raises
no wire-format question against T-065's versioned envelope; extending it is a follow-up if
wanted.

**Supersedes T-107**, which proposed printing the checked-out branch name derived from the
`main`/`master` guess. This ticket does the same job from a recorded fact instead, so T-107 is
dropped rather than kept alongside it.

**Soft couplings (not `depends-on`):**

- **T-109** — the payload and manual rewrite that makes the base-branch rule layout-conditional.
  It hard-depends on this ticket, because documentation may only describe behaviour that exists.
- **T-065** (done) — supplied `internal/state` and the `health` shape; its envelope is what this
  ticket deliberately does not modify.
- **T-057, T-072, T-082, T-100** (all done) — the base-branch enforcement family. They remain
  correct and load-bearing in the in-tree layout, and are vacuous in the umbrella layout; this
  ticket records the fact that distinguishes those two cases, and T-109 documents it.
- **T-046** (done) — made `doctor` self-host-aware, so `doctor` is the established home for a
  layout-conditional check.
- **`docs/proposals/post-merge-done-move.adoc`** — unaffected and still live: the in-tree layout
  persists, so the `4-in-review` to `6-done` timing question it raises is untouched.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child is `pickle` at `path = "."` (a root-path child), so the branch is created in
this repository:

```
git checkout main
git checkout -b feat/T-108-in-tree-layout
```

Local WIP commits are encouraged. **Publish-gated**: no push and no merge request without
explicit user approval; merging is always the human's. Because this is a root-path child,
interactive-rebase the WIP commits into a small number of atomic, correctly typed commits and
**keep that history** rather than squashing (rules §0). Before pushing, verify the remote base
is not behind the local base — `git fetch origin main && git diff --name-only origin/main...HEAD
| grep '^tickets/'` must print nothing, or push `origin main` first.

Bookkeeping for this ticket (its own status moves, the board) is committed on `main`, never on
this branch.

### Prerequisite gate (hard)

None. T-105 and T-106 are merged, and nothing this ticket touches is in flight. Start from a
clean tree on `main` — note that any unrelated local edits (for example locally-generated tool
output) must be left uncommitted and untouched.

### Confirmed design decisions (do not deviate without asking)

1. **The umbrella layout is the default; `--in-tree` is the only way to select the in-tree
   layout.** `install` never infers the layout from the working directory, because only the
   human knows which arrangement is intended.
2. **`--path` stops defaulting to `.`.** Today `internal/cli/install.go:21` defaults it, which
   silently produces an in-tree project — the current default is the exception, not the rule.
   After this ticket, omitting both `--in-tree` and `--path` installs an overarching project
   with **no child registered**, and `pickle project add` registers children afterwards. This
   is a deliberate breaking change to `install`'s CLI contract and is released as one.
3. **`--in-tree` implies the sole child at `.`.** Passing `--in-tree` together with an explicit
   `--path` other than `.` is an error, not a silent precedence rule.
4. **The layout is recorded in `pickle.toml` as `layout`, with exactly two legal values —
   `"umbrella"` and `"in-tree"`.** Where the key is present its value wins over any inference.
5. **A config with no `layout` key is resolved by inference** — a child registered at `.` means
   in-tree, otherwise umbrella — so projects installed before this ticket behave correctly
   before `upgrade` has run. Inference is a compatibility fallback for old configs only, never
   a substitute for the recorded value.
6. **`upgrade` back-fills `layout` using that same inference and stamps it.** It does so exactly
   as it already stamps `payload_version`, so no migration command ships.
7. **`doctor` enforces the invariant: `layout == "in-tree"` if and only if exactly one child is
   registered at `.`.** A recorded claim that nothing verifies drifts from reality.
   `checkChildren` (`internal/doctor/doctor.go:346`) is its home.
8. **The warning condition uses the configured branch prefix, and no base-branch name is ever
   guessed.** `serve` warns when the recorded layout is `in-tree` **and** the checked-out branch
   matches a registered child's configured `branch_prefix` (or HEAD is detached). This
   deliberately avoids needing to know what the base branch is called: the `main`/`master`
   guess at `internal/hook/prepush.go:206` is not extended, copied, or relied on. The trade-off
   is accepted — a branch that is neither the base nor prefix-matching produces no warning,
   which is a silent miss rather than a wrong claim.
9. **The warning ships on `pickle serve` only.** `board state --json` is left unchanged so this
   ticket raises no compatibility question against T-065's versioned envelope. Extending the
   signal to other readers is a follow-up, not part of this ticket.

### Tasks

#### Task 1 — record the layout in the config schema
`internal/config/config.go`: add ``Layout string `toml:"layout,omitempty"` `` to `Config`
(alongside `Flow`, near line 81) with exported constants for the two legal values. Reject any
other value in the existing validation path (near the child-path validation at line 278). Add
a resolver method that returns the recorded value when present and the decision-5 inference
when absent, so no call site re-implements the fallback.

#### Task 2 — `pickle install --in-tree`
`internal/cli/install.go`: add the `--in-tree` bool flag; change `--path`'s default from `"."`
to `""` (decision 2); error when `--in-tree` is combined with a non-`.` `--path` (decision 3);
register the child at `.` when `--in-tree` is given, and register none when neither flag is
supplied. `internal/install/install.go`: write the `layout` key into the generated
`pickle.toml`.

#### Task 3 — `upgrade` back-fills the key
In the upgrade path that already stamps `payload_version`, add the decision-5 inference and
stamp `layout` when the key is absent, leaving an existing value alone.

#### Task 4 — `doctor` enforces the invariant
`internal/doctor/doctor.go`, extending `checkChildren` (line 346): error when `layout` is
`"in-tree"` without exactly one child at `.`, and when it is `"umbrella"` with a child at `.`.
Report the resolved layout as an informational passed line under `-v`.

#### Task 5 — `serve` states the layout and warns
`internal/cli/serve.go`: print the resolved layout on the existing startup line.
`internal/serve/serve.go` + `internal/serve/templates/layout.html`: render a persistent banner
when decision 8's condition holds, placed **outside** `#board` so it is not swapped out by the
5-second fragment poll (the same constraint the existing filter bar and search input already
observe). The banner names the checked-out branch and states that ticket status may be behind
the base branch.

#### Task 6 — reference documentation
`docs/user-manual/cli-reference.adoc`: document `--in-tree` and the changed `--path` default in
`== pickle install` (line 80), the `layout` back-fill in `== pickle upgrade` (line 215), the new
invariant in `== pickle doctor` (line 290), and the banner in the `serve` section.

### Acceptance test

Run from the repository root on the feature branch. Per this project's self-modify policy, every
install test uses a throwaway directory with the binary copied in and renamed to `pickle-test`.

1. `just build && just test && just lint && just docs-check` — all clean.
2. **Umbrella is the default, and registers no child:**
   `D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q -b main . && ./pickle-test install --agent claude`
   then `grep '^layout' pickle.toml` prints `layout = "umbrella"`, and `grep -c '\[\[project\]\]' pickle.toml` prints `0`.
3. **`--in-tree` records the layout and the sole child:** same setup with
   `./pickle-test install --in-tree --agent claude` — `grep '^layout' pickle.toml` prints
   `layout = "in-tree"` and the file contains one `[[project]]` with `path = "."`.
4. **Conflicting flags are refused:** `./pickle-test install --in-tree --path sub` exits
   non-zero and names the conflict.
5. **`doctor` catches a contradicting layout:** in the case-3 directory, edit `layout` to
   `"umbrella"`; `./pickle-test doctor` exits non-zero and reports the invariant. Restore the
   value and `doctor` passes.
6. **`upgrade` back-fills:** delete the `layout` line from the case-3 config, run
   `./pickle-test upgrade`, and confirm `layout = "in-tree"` returns (inferred from the child
   at `.`).
7. **The banner appears only where it should:** in the case-3 directory,
   `git checkout -b feat/T-001-x` then run `./pickle-test serve --addr 127.0.0.1:8830` and
   `curl -s http://127.0.0.1:8830/` — the body contains the warning and the branch name. Repeat
   on `main`: the warning is absent. Repeat in the case-2 (umbrella) directory on any branch:
   the warning is absent.
8. **The banner survives the poll:** with the banner showing, `curl -s
   http://127.0.0.1:8830/fragments/board` does **not** contain the warning text — proving it
   lives outside the polled fragment.

### Docs update (mandatory when user-facing)

`docs/user-manual/cli-reference.adoc` as itemised in Task 6: the `--in-tree` flag, the changed
`--path` default (called out as a breaking change), the `layout` config key and its two values,
the `upgrade` back-fill, the `doctor` invariant, and the `serve` banner. The conceptual
"which layout should I choose, and what does in-tree cost" chapter is **not** in this ticket —
that is T-109, which depends on this one.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated and registered.
3. Write a summary of files touched, decisions honoured, and anything deferred (notably the
   `board state --json` extension, per decision 9).
4. Suggested Conventional Commit message:

   ```
   feat(install): record the board layout and warn on stale in-tree reads (T-108)

   Add --in-tree to select the layout explicitly, record it as `layout` in
   pickle.toml, enforce the layout/children invariant in doctor, back-fill the
   key on upgrade, and warn in serve when an in-tree board is read from a
   feature branch. --path no longer defaults to "." (breaking).
   ```

5. Root-path child: interactive-rebase the WIP commits into atomic, correctly typed commits and
   keep that history rather than squashing.
6. Commit locally; present for approval. Do not push or open a merge request without it.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-17 — created (TO DO). source: pickle ticket new
- 2026-08-17 — TO DO → READY: refined: 9 confirmed decisions, 6 tasks, 8-step acceptance test
- 2026-08-17 — READY → IN DEVELOPMENT: picked up
