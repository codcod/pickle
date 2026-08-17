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

**2026-08-17 — review 1 (full).** Branch `feat/T-108-in-tree-layout` at `979366d`, read against
`main`. Verdict: **blocking findings — to `5-rework/`**. The code is in good shape and the
acceptance test passes verbatim; every blocking finding is documentation the branch made false
outside the one file the plan named.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on `docs/user-manual/cli-reference.adoc` (step 4b) — 12 suggestions
      returned, 11 of them on prose this branch did not touch and therefore out of scope; the one
      on the new `Layout: umbrella (default) or in-tree` section is carried into rework
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated (step 7) — T-109's plan gains F5
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary presented; no publish (step 9)

### Implementation audit (step 2)

All 8 acceptance-test steps re-run verbatim from the repository root on the feature branch, each
install into a throwaway directory with the binary copied in as `pickle-test`:

| step | result |
|---|---|
| 1 `just build && just test && just lint && just docs-check` | **met** — all clean |
| 2 umbrella is the default, registers no child | **met** — `layout = "umbrella"`, `0` `[[project]]` |
| 3 `--in-tree` records layout + sole child | **met** — `layout = "in-tree"`, one `[[project]]` with `path = "."` |
| 4 `--in-tree --path sub` refused | **met** — exit 2, names `"sub"` |
| 5 `doctor` catches a contradicting layout | **met** — `ERROR: layout: "umbrella" must have no child registered at ".", found 1`, exit 1; restored → exit 0 |
| 6 `upgrade` back-fills | **met** — `layout = "in-tree"` returns by inference |
| 7 banner only where it should | **met** — present on `feat/T-001-x` naming the branch; absent on `main`; absent in the umbrella tree |
| 8 banner survives the poll | **met** — `/fragments/board` contains no warning text |

All six tasks are done in the files they name. All nine confirmed decisions are honoured;
decisions 2 and 3 were also *extended* with two further refusals (`--path .` alone, and child
flags with no `--path`/`--in-tree`) that no decision states and no History line records — see
F9.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | docs-gap | — | The manual still teaches the removed `--path` default in four places, and one documented command now fails | `docs/user-manual/concepts/project-structure.adoc:64,131,137` — the shown `pickle install --project backend --path .` exits 2 (`--path "." selects the in-tree layout; pass --in-tree explicitly`); `:140-142` still says "nothing in the tool branches on `path = "."`", which `layout`/`checkLayoutInvariant` made false; `quickstart.adoc:36`; `your-first-project.adoc:24`; `concepts/multi-project.adoc:25` | Correct all four. `your-first-project.adoc` is the sharpest: it installs, then files a ticket — with no child registered, its step 3 now has nothing to target (`pickle ticket new` → `--project is required`), so the tutorial needs `--in-tree` or a `project add` step |
| F2 | blocking | docs-gap | — | `configuration.adoc`, the canonical `pickle.toml` key reference, omits `layout` and asserts something this branch made false | `docs/user-manual/configuration.adoc:10` "*`upgrade`* rewrites only the `payload_version` line" — it now also inserts `layout`; `:36-44` "The keys" lists `payload_version`, `review_addendum`, `[commit]` and no `layout` | Add `layout` with its two legal values and the inference fallback; correct the `upgrade` bullet. (`flow` is missing from the same list — pre-existing, worth fixing in passing) |
| F3 | blocking | docs-gap | — | No `CHANGELOG.md` entry, and the plan's own Docs step required the `--path` default to be "called out as a breaking change" — no such callout exists anywhere | `pickle changelog check` → `1 candidate(s) shipped but not named in "Unreleased": T-108`; the branch touches no `CHANGELOG.md`; every comparable ticket (T-104, T-105, T-106, T-065, T-101) landed its entry on the feature branch | Add an `Unreleased` entry: *Added* for `--in-tree`/`layout`/the `doctor` invariant/the `serve` banner, *Changed* for the breaking `--path` default |
| F4 | blocking | docs-gap | — | The new default install renders a malformed marker block: two bullets end in a colon with nothing after them | fresh `pickle-test install` → `AGENTS.md` contains `Branch per child:` and `- **WIP limits** (per child):` with empty bodies, because `branches`/`wip` are empty with no child registered | `internal/install/install.go:1105-1106` need the empty-case fallback the children line already has at `:1032-1034` (or omit both bullets when no child is registered) — this block is the first thing an agent reads in every new default install |
| F5 | non-blocking | docs-gap | folded (T-109) | `skill/SKILL.md` still says `install` "registers the first child-project" | `skill/SKILL.md:54-56` | T-109 already owns every payload edit; added to its Task 3 with a History line rather than editing `skill/` from this branch |
| F6 | non-blocking | stale-xref | fixed inline | `checkChildren`'s doc comment now sits directly above the newly inserted `checkLayoutInvariant`, so it documents the wrong function and `checkChildren` has none | `internal/doctor/doctor.go:337-352` | Move `checkLayoutInvariant` below `checkChildren` so each comment rejoins its function |
| F7 | non-blocking | other | fixed inline | Three new test comments cite a review that had not happened when they were written | `internal/cli/layout_test.go:100-105` ("pins the gap T-108 review found"); `internal/serve/layout_test.go:92-96` ("pins a real gap review found") | Name the actual origin (a self-audit during implementation) |
| F8 | non-blocking | design | noted | `pickle project add <name> .` on a default umbrella install succeeds and leaves a config `doctor` errors on forever, with no command that repairs it — and the two new layout errors are the only errors in `doctor.go` carrying no remedy clause | reproduced: `install` (default) → `project add self .` → `ERROR: layout: "umbrella" must have no child registered at "."`, and `upgrade` never rewrites an existing `layout`; contrast `doctor.go:81,263,433` ("— run `pickle upgrade`", "— edit `payload_version` by hand") with `:362-364` | Name the remedy in both messages (edit `layout` by hand), and/or have `project add` refuse or warn when registering `.` under a recorded umbrella layout |
| F9 | non-blocking | docs-gap | noted | Two refusals ship beyond decision 3 — a bare `--path .`, and `--project/--build/--test/--lint/--docs` with no `--path`/`--in-tree`. The first is documented; the second is documented nowhere, and neither is recorded as a plan amendment | `internal/cli/install.go:38-56`; `cli-reference.adoc` mentions only the bare `--path .` refusal; `## History` carries no `plan amended inline` line | Document the child-flag refusal in `== pickle install`. Both refusals are right; only the record of them is missing |
| F10 | non-blocking | docs-gap | fixed inline | `cli-reference.adoc` claims the layout back-fill "refuse[s] outright if it cannot do so safely", which is true of the `payload_version` stamp and not of the layout insert | `docs/user-manual/cli-reference.adoc:293-296` vs `internal/config/config.go:562-586` (writes, then `install.go:485-493` verifies) | Narrow the sentence to what the code does, or add the gate (F11) and keep the sentence |
| F11 | non-blocking | design | noted | `SetLayoutInPlace` has no parse-back gate, unlike `SetPayloadVersionInPlace`: it writes first and verifies after, so a pathological `pickle.toml` is left unparseable rather than untouched | `internal/config/config.go:562-586` has no `verifyOnly…` equivalent of `setPayloadVersion`'s (`:636-650`); e.g. an existing dotted `layout.x = …` key is not matched by `topLevelKeyPresent`, so a duplicate `layout` is inserted | Route the insert through the same decode-both-sides gate. Exotic inputs only — hence non-blocking |
| F12 | non-blocking | design | noted | `staleBoardBranch` shells out to git on every 5-second fragment poll and the answer is discarded | `internal/serve/serve.go:267-280` — `boardFragment`/`activityFragment` call `newPage`, but the fragment templates never render `.StaleBoard` (which is exactly what acceptance step 8 proves) | Resolve it only for full-page renders, or memoise per request |
| F13 | non-blocking | design | noted | This repo's own `pickle.toml` records no `layout`, so pickle self-hosts on the compatibility inference decision 5 reserves for pre-existing configs | `pickle.toml` has no `layout` key; `./pickle doctor -v` → `ok: layout "in-tree" is consistent with 1 root-path child(ren)` | No branch action: `AGENTS.md`'s self-modify policy routes this to the post-merge human `pickle upgrade` from `main`, which back-fills the key. Recorded so that upgrade is actually run |

dispositions: 4 blocking (F1–F4, not dispositioned — fixed in rework); 9 non-blocking — 1 folded
(F5 → T-109), 3 fixed inline (F6, F7, F10), 5 noted (F8, F9, F11, F12, F13); 0 new tickets.

```
cost: estimated M, actual M
```

### Notes that are not findings

- The docs-readability pass suggested splitting the new *Why the choice matters* paragraph into
  two (umbrella cannot fork / in-tree can, and what that costs). Worth applying while F1–F2 are
  being written; the other 11 suggestions land on prose this branch never touched.
- The stale-ticket hazard this ticket exists to warn about showed up during the review itself:
  the feature branch's worktree still carries T-108 in `3-in-development/`, because the move to
  `4-in-review/` was committed on `main` after the branch was cut. The ticket was read from
  `main` throughout, per the review protocol.

**2026-08-17 — rework: F1–F4 fixed, F6/F7/F10 applied.** Same branch, 7 new commits
(`ad0866e`..`0e5b3bd`). Scope was exactly the four blocking findings plus the three
`fixed inline` non-blocking ones the review already dispositioned; F5 (folded into T-109), F8,
F9, F11, F12 and F13 (all `noted`) are untouched, as recorded.

| id | fix |
|---|---|
| F1 | `docs/user-manual/concepts/project-structure.adoc`, `multi-project.adoc`, `your-first-project.adoc`, `quickstart.adoc` no longer teach the removed `--path` default. The broken example (`pickle install --project backend --path .`) is now `pickle install --in-tree --project backend`; the quickstart/first-project walkthroughs now install `--in-tree` in step 1, so a child exists before the tutorial files a ticket in step 3 |
| F2 | `configuration.adoc`'s key list now documents `layout` (both values, the pre-`layout` inference) and the `upgrade` bullet states that it inserts `layout` as well as stamping `payload_version` |
| F3 | `CHANGELOG.md` `[Unreleased]` gained an *Added* entry (`--in-tree`, `layout`, the `doctor` invariant, the `serve` banner) and a *Changed* entry calling the `--path` default change out as breaking. `pickle changelog check` now reports no unmentioned candidates |
| F4 | `internal/install/install.go`'s `MarkerBlock` gives "Branch per child:" and "WIP limits (per child):" the same empty-case guard the children summary already had; a childless install now reads "No children registered yet" / "(none yet)" instead of a dangling colon. Regression test added: `TestMarkerBlockNoChildrenHasNoDanglingBullets` |
| F6 | `checkLayoutInvariant` moved below `checkChildren` in `internal/doctor/doctor.go`; each function's doc comment is back over its own function |
| F7 | The two test comments in `internal/cli/layout_test.go` and `internal/serve/layout_test.go` claiming "T-108 review found" this gap now say "caught during implementation", which is what actually happened |
| F10 | `cli-reference.adoc`'s upgrade section no longer claims the `layout` back-fill "refuses outright if it cannot do so safely"; it now states what the code does — insert, then verify by re-reading — in contrast to `payload_version`'s pre-write parse-back gate. The *Why the choice matters* paragraph was also split per the docs-readability suggestion carried over from review 1 |

Re-ran the full acceptance test (all 8 steps) verbatim after the fixes: all still pass
(umbrella/in-tree installs, the conflicting-flags refusal, the `doctor` invariant both
directions, the `upgrade` back-fill, and the banner present/absent/surviving-the-poll cases).
`just build && just test && just lint && just docs-check` clean; `pickle changelog check`
clean.

**2026-08-17 — review 2 (scoped re-review).** Branch at `e2ddefa`, read against `main`. Scope
was the seven findings review 1 sent to rework (F1–F4 blocking, F6/F7/F10 `fixed inline`); the
feature was not re-audited from scratch. Verdict: **all seven resolved → `6-done/`**.

| id | verified | how |
|---|---|---|
| F1 | **resolved** | All four files corrected. The previously-broken example now runs: `pickle install --in-tree --project backend` exits 0 and writes `layout = "in-tree"` with one `[[project]]` at `.`. Both walkthroughs were executed end to end — quickstart's install→`ticket new` sequence now succeeds (it created `T-001`), and `your-first-project`'s step 1 (`--in-tree`) followed by step 2's two nested `project add`s leaves `doctor` at 0 errors, so the new `--in-tree` step did not trade one broken tutorial for another |
| F2 | **resolved** | `configuration.adoc` documents `layout` in "The keys" with both values and the inference; the `upgrade` bullet now names the `layout` insert. Both new claims check out against the code — insert-only (`SetLayoutInPlace`'s `topLevelKeyPresent` guard plus `Upgrade`'s `cfg.Layout == ""`) and verify-after-write (`verifyLayoutBackfilled`) |
| F3 | **resolved** | `CHANGELOG.md` carries an *Added* entry and a *Changed* entry that calls the `--path` default change out as breaking. `pickle changelog check` → `no candidates — every shipped ticket is mentioned` |
| F4 | **resolved** | A childless install now renders "No children registered yet — see above." and "(none yet — see above)". `doctor -v` on that install reports the marker block *current* in both `AGENTS.md` and `CLAUDE.md`, so the generated form and the drift-detection render agree. The regression test is genuine, not tautological: applied against the pre-fix commit (`979366d`) in a scratch worktree it **fails** |
| F6 | **resolved** | `checkLayoutInvariant` moved below `checkChildren`; each doc comment sits over its own function again |
| F7 | **resolved** | Both comments now say "caught during implementation" |
| F10 | **resolved** | The upgrade section distinguishes `payload_version`'s refuse-before-write gate from `layout`'s verify-after-write insert — which is what the code does. The *Why the choice matters* paragraph was also split |

Also checked, because `docs-check` cannot: the `<<install-layout>>` anchor is defined exactly
once (`cli-reference.adoc:106`) and all 11 references across six files resolve, every one of
those files being in `docs/user-manual.adoc`'s `include::` tree. (Anchor validation is still
absent from the docs pipeline — T-067 — so this was verified by hand.)

Two findings of its own, neither blocking:

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F14 | non-blocking | stale-xref | fixed inline | The rework's own guard comment named a variable that does not exist (`wipBullet`; the variable is `wipLine`) | `internal/install/install.go:1066` | Fixed in `e2ddefa` |
| F15 | non-blocking | docs-gap | noted | The walkthroughs use `T-1` for the first ticket, but a fresh project allocates `T-001` — so `tickets/1-to-do/T-1-….md`, `refine ticket T-1`, `pickle ticket move T-1 ready` and `feat/T-1-<slug>` all misname it | `quickstart.adoc:52`; `your-first-project.adoc:71,78,88,90`; observed `created T-001` on a fresh install | Pre-existing — the same text is in the pre-T-108 base, so this branch did not make it false and the inline bar (rules §5) excludes it. Left for a docs pass that owns the tutorials |

dispositions: 0 blocking; 2 non-blocking — 1 fixed inline (F14), 1 noted (F15); 0 new tickets.
Carried forward untouched from review 1, as recorded there: F5 (folded into T-109), F8, F9,
F11, F12, F13 (all `noted`).

```
cost: estimated M, actual M
```

## History

- 2026-08-17 — created (TO DO). source: pickle ticket new
- 2026-08-17 — TO DO → READY: refined: 9 confirmed decisions, 6 tasks, 8-step acceptance test
- 2026-08-17 — READY → IN DEVELOPMENT: picked up
- 2026-08-17 — IN DEVELOPMENT → IN REVIEW: acceptance test green: all 8 checks re-run verbatim; just build/test/lint/docs-check clean
- 2026-08-17 — IN REVIEW → REWORK: review 1: 4 blocking findings (F1-F4), all documentation the branch made false outside cli-reference.adoc
- 2026-08-17 — REWORK → IN REVIEW: findings fixed
- 2026-08-17 — IN REVIEW → DONE: review 2 (scoped): F1-F4 + F6/F7/F10 all resolved; 2 new non-blocking (F14 fixed inline, F15 noted)
