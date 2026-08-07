---
id: T-051
title: surface the workspace-side consequences of registering a child-project
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-051 — surface the workspace-side consequences of registering a child-project

## Description

`pickle project add <name> <path>` writes one `[[project]]` block into `pickle.toml`, and
`pickle upgrade` then re-renders the marker blocks. Everything else that makes a workspace
actually treat the new directory as a child stays a hand edit, discovered by imitating the
first child — and pickle already knows the two facts (name, path) that all of it derives from.

Measured on a real second-child registration (`snowball` added alongside `rick` in the
`unity` workspace, pickle 0.1.0, 2026-07-27): after `project add` reported success, five
edits were still required, none of them prompted or mentioned.

1. **`.gitignore`** — `/snowball/`, so the nested child repo is never staged into the
   overarching repo. This one has teeth: until the entry exists the child *is* stageable, and
   the window is open from the moment the directory appears until a human remembers.
2. The workspace's **own Pi extension** — a `CHILD_DIRS` array that duplicates the child
   registry (`unity-guardrails.ts`).
3. **`opencode.jsonc`** — per-child never-stage glob patterns.
4. **`AGENTS.md` outside the markers** — the per-child records table row and provenance prose.
5. **`development/<child>/`** — the per-child record directory that workspace's convention
   requires.

Items 2–5 are that workspace's inventions and are arguably none of pickle's business; item 1
is not — it is mechanical, derived from `path`, and its omission is a staging accident rather
than a cosmetic gap. The interesting question this ticket must answer is **where the line
sits**, because multi-child is pickle's advertised shape: the second child is not an edge case.

Three candidate shapes, deliberately left un-chosen until refinement (a field-note discipline:
see `tickets/NOTES.md` § *Field-finding triage (2026-07-27)*, where two notes arrived with
implementations attached and both diagnoses were wrong):

- **Tell the truth on stdout** — `project add` prints the consequences it did *not* perform.
  Smallest change; helps only at registration time.
- **Perform the mechanical ones** — e.g. append the `.gitignore` entry. Narrow, but pickle
  starts editing a file it does not own.
- **A `doctor` check** — "registered child `X` has no `.gitignore` entry". The only shape that
  also catches a workspace that has *drifted* (a child registered before the check existed,
  or an entry deleted), not just a fresh registration.

Hard constraints on any of them: pickle must not edit hand-written prose outside its own
markers, and must not touch a workspace's non-pickle extension files — that separation is
precisely why `upgrade` can safely refresh `pickle-guardrails.ts` while leaving
`unity-guardrails.ts` alone.

Soft coupling: **T-052** (the post-upgrade audit error that the same `project add` → `upgrade`
sequence produces) — same onboarding session, same command pair, but a separate defect;
whichever lands second inherits the other's understanding of what that sequence should feel
like. Note also that if a child-directory guard is ever added to the *shipped*
`pickle-guardrails.ts` as part of this work, it must anchor child paths at the pathspec start
(plus `../` climbs) rather than anywhere in the string — the unanchored form also matches
`development/<child>/…`, which is ordinary bookkeeping (defect found and fixed workspace-side
during the same session).

## Implementation Plan

### Feature branch

`feat/T-051-surface-the-workspace-side-consequences-of-registering-a-child-project`, cut in
the `pickle` child (the repo root) from `main`. Local WIP commits are fine; **no push and no
MR without explicit user approval**. Never run `pickle install|upgrade|project add` against
this repo from the branch — test installs go to a throwaway dir with the binary copied in
(`AGENTS.md`, self-modify policy).

### Prerequisites

None. `git` on `PATH` is a runtime dependency of the new check, not a build one; the tests
`git init` real fixtures, exactly as `internal/doctor/hooks_test.go` already does.

### Confirmed decisions

1. **Shape: report, never write.** A `doctor` check (the only shape that also catches a
   workspace that *drifted*) plus the **same finding printed at registration time** by
   `pickle project add` and by `pickle install --path <sub>`. pickle does **not** append to
   `.gitignore`: it does not own that file, and a child deliberately tracked as a submodule
   would make the append actively wrong.
2. **Ask git, do not parse `.gitignore`.** `git check-ignore -q -- <path>` (exit 0 = ignored,
   1 = not, ≥2 = unknown) and, when not ignored, `git ls-files --error-unmatch -- <path>`
   (exit 0 = already tracked, i.e. a deliberate gitlink/submodule). A textual scan of the root
   `.gitignore` misses `.git/info/exclude`, negations and nested ignore files, and a
   false-positive warning that can never be silenced is worse than no check — this repo already
   tolerates one such standing warning and does not need a second.
3. **Scope.** Only children whose `path != "."`. Silent (no finding, not even a passed line)
   when git is absent, the root is not a repository, or the answer is otherwise unknown; a
   child already tracked in the index is reported as fine, not as a problem.
4. **Severity: warning** (`doctor` still exits 0). Consistent with every other advisory doctor
   finding; an error would break `pickle doctor && …` on a legitimate submodule layout.
5. **Items 2–5 of the Description stay out of scope** — the workspace's pi extension,
   `opencode.jsonc` globs, `AGENTS.md` prose outside the markers, and `development/<child>/`
   are that workspace's inventions. pickle states only what its own two registry facts (name,
   path) imply. No generic "your workspace may have other conventions" reminder: that is the
   noise this ticket exists to remove.
6. **No child-directory deny-list is added to the shipped `agents/pi/extensions/pickle-guardrails.ts`.**
   Decided against, not deferred. The Description's anchoring lesson is preserved for whoever
   revisits it: such a guard would have to anchor child paths at the **pathspec start** (plus
   `../` climbs), never match-anywhere, or it also matches `development/<child>/…`.
7. **`pickle upgrade` gains nothing.** It is not a registry event, and `doctor` already covers
   drift. Keeping upgrade out of this is the same restraint that keeps it out of the board.
8. **The exec stays behind a package.** `internal/doctor`'s standing rule (`checkHooks`
   comment, T-057 decision 12) is that the package returns findings and never spawns a process
   *directly*; the new check honours it by living behind `internal/vcs`, and that comment is
   updated to name the second caller.

### Tasks

1. **New package `internal/vcs`** (`internal/vcs/vcs.go`). Small and single-purpose: answer
   "would the overarching repo stage this nested child?".
   - `type State int` with `Unknown`, `Ignored`, `Tracked`, `Stageable` (`Unknown` as the zero
     value, so any failure path is silent by construction).
   - `func ChildState(root, relPath string) State` — runs `git -C <root> check-ignore -q --
     <relPath>`; on exit 1 runs `git -C <root> ls-files --error-unmatch -- <relPath>`; maps
     exit codes per decision 2 and returns `Unknown` for anything else (git missing, exit 128,
     timeout).
   - `func (s State) Advice(relPath string) string` — the **single source of the wording**,
     mirroring `hook.Reach.Problem()`: `""` for every state but `Stageable`, and for that one
     `<relPath>/ is a nested git repository that this repository does not ignore — add
     "/<relPath>/" to .gitignore so it is never staged`. Callers prepend their own context.
   - Plumbing: `exec.CommandContext` with a package-level `probeTimeout`-style `var` (3s, a
     var so a test can shrink it) so `doctor` can never hang, and the `GIT_DIR`/`GIT_WORK_TREE`/
     `GIT_INDEX_FILE`/`GIT_PREFIX` scrub that `internal/hook`'s `withoutRepoEnv` performs —
     `-C` and an inherited `GIT_DIR` would otherwise disagree about which repository is meant.
     Comment it as a deliberate mirror; **do not** refactor `internal/hook` to share it (an
     unrelated diff in the guard's plumbing is not this ticket's risk to take).
2. **`internal/vcs/vcs_test.go`** — real `git init` fixtures (pattern:
   `internal/doctor/hooks_test.go:36`): ignored child (`/child/` in `.gitignore`) → `Ignored`;
   plain nested repo → `Stageable`; nested repo added to the index as a gitlink → `Tracked`;
   a root that is not a repository → `Unknown`; `Advice` empty for all but `Stageable`.
3. **`internal/doctor/doctor.go`** — extend `checkChildren`: after the existing `.git` check
   passes and only when `p.Path != "."`, call `vcs.ChildState(root, p.Path)` and
   - `Stageable` → `r.warnf("child %q: %s", p.Name, st.Advice(p.Path))`;
   - `Ignored` / `Tracked` → an `r.ok(...)` line (verbose-only, so a healthy tree stays quiet);
   - `Unknown` → nothing at all.
   Update the `checkHooks` closing comment (`internal/doctor/doctor.go`, "this package never
   spawns a process directly") so it names `internal/vcs` as the second confined exec rather
   than reading as an invariant the code now breaks.
4. **`internal/doctor/doctor_test.go`** — a stageable child yields exactly one warning and zero
   errors (exit stays 0); adding the `.gitignore` entry clears it; a `path = "."` child never
   produces the finding (guards the self-host case). The existing `installFixture` makes a bare
   `.git` *directory* with no repository inside, so `ChildState` returns `Unknown` there and
   `TestCheckHealthyInstall` must stay unchanged — assert that explicitly.
5. **`internal/cli/project.go`** — add `func noteIfStageable(root string, p config.Project)`
   next to `refreshMarkers` (same package, so `install.go` reuses it): prints
   `note: <advice>` to stdout when `p.Path != "."` and the state is `Stageable`, nothing
   otherwise. Call it at the end of `runProjectAdd`, after `refreshMarkers`, before returning
   `exitOK`. Registration succeeded — this never changes the exit code.
6. **`internal/cli/install.go`** — call the same helper in `runInstall` after the post-install
   audit and before the closing `pickle installed in …` line, for the first child when
   `*path != "."`.
7. **`internal/cli/cli_test.go`** — end-to-end: `install` + `project add` in a temp git repo
   with a nested child repo prints the note and exits 0; with the `.gitignore` entry present it
   does not.

### Docs

- `docs/user-manual/cli-reference.adoc` — add a bullet to the `pickle doctor` check list
  (after "each registered child's `path` is a git repository"), stating the warning, that a
  child at `.` and an already-tracked child are exempt, and that pickle never edits
  `.gitignore`; and one sentence in `== pickle project add | list | remove` that `add` says so
  at registration time.
- `docs/user-manual/concepts/project-structure.adoc:64` — **correct a now-false claim.** "It
  produces no build artifacts, so there is nothing to add to `.gitignore`" holds only for the
  single-repo default; a child registered at a nested path must be ignored so the outer repo
  never stages it.
- `docs/user-manual/concepts/multi-project.adoc` — one line under *What a child carries* (or
  the registration bullet) pointing at the same rule.
- `CHANGELOG.md` — an entry under `## [Unreleased]` → `### Added`, in the existing voice:
  symptom first (a nested child is stageable until someone remembers), then the shape
  (report, never write) and why `.gitignore` is not edited.

### Acceptance test

From the repo root, all four must be green:

```
just build && just test && just lint && just docs-check
```

Then the end-to-end, in a throwaway dir with the binary copied in (self-modify policy):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q -b main .
./pk install --project root --agent claude
mkdir child && (cd child && git init -q -b main .)
./pk project add child child      # expect: registered … + a `note:` naming /child/
./pk doctor; echo "exit=$?"       # expect: 0 error(s), 1 warning(s) — exit=0
printf '/child/\n' >> .gitignore
./pk doctor; echo "exit=$?"       # expect: 0 error(s), 0 warning(s) — exit=0
./pk doctor -v | grep child       # expect: an ok: line saying the child is ignored
```

Expected results in full: the `project add` note and the first `doctor` warning quote the same
sentence (one source of wording); neither ever changes an exit code; the second `doctor` is
silent about the child; and nothing in `$D` writes to `.gitignore` except the operator's own
`printf`.

### Finish

Summary of what shipped (the check, the two call sites, the doc correction) plus what was
decided **against** — no `.gitignore` write, no guardrail deny-list, no `upgrade` change — so a
later reader does not re-litigate it. Suggested commit message:

```
feat(doctor): warn when a registered child is not gitignored (T-051)
```

## Review

2026-08-07 — reviewed on `feat/T-051-…` at `9e775f9`, with the audit run by an independent
sub-agent (the reviewer was also the implementer, so the quality/consistency/correctness pass
was delegated to fresh eyes and every finding then re-verified by hand before classification).

- [x] Implementation audit — acceptance test re-run verbatim, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc` files (step 4b) — run during implementation;
      two in-scope suggestions applied, and F6 below records that one of them made the prose
      *worse* and was re-fixed here
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message & MR attributes presented for approval (step 9) — **deferred:
      blocking findings, so nothing is publishable yet**

**Verified green:** all 7 plan tasks landed in the files named; decisions 1, 4, 5, 6, 7, 8
honoured; `just build`/`test`/`lint`/`docs-check` all pass; the acceptance test reproduces every
expected line, and `.gitignore` in the throwaway dir contained only the operator's own entry —
confirming decision 1 (report, never write). Exit codes were re-derived empirically against git
2.55: `check-ignore -q` → 1 unignored / 128 outside a repo, `ls-files --error-unmatch` → 1
untracked (not 128, the code that would have silently broken the feature).

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | fixed inline | `DESIGN.md` still claimed `internal/hook` is the *only* package that shells out to git — a statement this branch made false. `hook.go`'s own doc was updated; `DESIGN.md` was missed. | `DESIGN.md:231-232`, `:239` | Amend both to name `internal/vcs`. Done in `81dcfc5`. |
| F2 | **blocking** | — | A child registered in `./x` form yields advice git does not honour: it says to add `/./dot/`, and `/./dot/` leaves `check-ignore` at exit 1 — so following pickle's own instruction never silences the warning. Precisely the un-silenceable false positive decision 2 exists to prevent. | `./pk project add dotchild ./dot` prints `note: ./dot/ … add "/./dot/" to .gitignore`; with that line in `.gitignore`, `git check-ignore -q dot` still exits 1 and `doctor` still warns | Normalise the path in `State.Advice` (`internal/vcs/vcs.go:70`), e.g. `path.Clean` before the trailing-slash logic; keep the sentence otherwise byte-identical. Add a case to `vcs_test.go`. |
| F3 | **blocking** | — | `noteIfStageable` calls a plain, non-repo directory a "nested git repository", and `doctor` simultaneously errors that the same child is *not* one. Two commands, one child, contradictory statements — and the helper exists precisely so the two moments cannot disagree. `doctor` reaches the vcs check only after its `.git` stat passes; the CLI helper has no such guard. | `mkdir plain && ./pk project add plain plain` prints `note: plain/ is a nested git repository …`, while `./pk doctor` prints `ERROR: child "plain": plain is not a git repository` | Mirror doctor's guard in `noteIfStageable` (`internal/cli/project.go:117`): stat `<root>/<path>/.git` and return early when absent. |
| F4 | non-blocking | noted | Plan task 6 (the `runInstall` call site) has no test — every `install` case in `internal/cli` uses the default `--path .`, so the branch that fires the note at install time is unexercised. Verified manually that it does work (`install --project root --path sub` prints the note, exit 0). | `grep -n '"install"' internal/cli/*_test.go` — no `--path` case | A sibling of `TestProjectAddNotesStageableChild` using `install --path sub`. Cheap, but it did not pass the promotion test as its own ticket; the rework pass for F2/F3 is the natural place to add it if it touches those lines anyway. |
| F5 | non-blocking | noted | `probeTimeout` is documented as "a var so a test can shrink it", but no test shrinks it and nothing exercises the `-1` (git-absent/timeout) branch; there is also no doctor- or CLI-level `Tracked` test, so the `Tracked` ok-line is only covered by the vcs unit test. | `internal/vcs/vcs_test.go` has no timeout case; cf. `internal/hook/probe_test.go:102-104`, which does shrink its copy | A stub-`git`-on-PATH test mirroring `probe_test.go` would close both gaps at once. Not scheduled on its own. |
| F6 | non-blocking | fixed inline | Garbled sentence in the new `project add` prose: "That means no `.gitignore` entry — or index entry, for a deliberate submodule/gitlink — yet says otherwise" reads as a fragment. Introduced by a docs-readability suggestion applied during implementation — a reminder that the readability pass suggests, and the author still owns the result. | `docs/user-manual/cli-reference.adoc:170-173` | Rewritten to "…whether anything yet tells git to leave it alone: a `.gitignore` entry, or an index entry for a deliberate submodule/gitlink." Done in `81dcfc5`. |
| F7 | non-blocking | fixed inline | The package doc did not state that git answers about whichever repository it *discovers* from `root`. If the install root is not itself a repo but sits inside one, the answer — and the root-anchored entry `Advice` suggests — belong to the enclosing repo. | Verified: `git -C proj check-ignore` resolves `--show-toplevel` to the enclosing repo when `proj` is not one | Caveat added to `internal/vcs`'s package doc. Done in `81dcfc5`. |
| F8 | non-blocking | fixed inline | `Tracked` was reported as "presumed a deliberate gitlink/submodule", but `ls-files --error-unmatch` on a directory exits 0 if *any* file beneath it is tracked. So a child whose contents were already committed as ordinary files reads exactly like a gitlink — meaning the check goes quiet on the very accident it exists to prevent, once that accident has already happened. The *behaviour* is per decision 3 (tracked ⇒ not a problem); only the claim about *which* kind of tracked was wrong. | `git ls-files --error-unmatch -- kid` exits 0 for a dir holding one tracked blob | ok-line reworded to state what git reports rather than presuming a gitlink, and the limit written into the package doc. Done in `81dcfc5`. Closing the gap properly needs a gitlink-vs-blob distinction — deliberately not in scope. |
| F9 | non-blocking | noted | The ticket's own Description opens with "`pickle project add` writes one `[[project]]` block into `pickle.toml`, and `pickle upgrade` then re-renders the marker blocks" — stale since T-041 made `project add` refresh the markers itself. Pre-existing (this branch did not falsify it), so not eligible for `fixed inline`. | `internal/cli/project.go:104` `refreshMarkers(cfg)`; T-041's own History already records "T-051 premise now false — no action" | Leave it. The Description is a historical record of the field finding; the Implementation Plan, which is what was built, never depended on the claim. |

**Disposition summary:** 9 findings — **2 blocking** (F2, F3 → `5-rework/`); 7 non-blocking:
4 fixed inline (F1, F6, F7, F8, all in `81dcfc5`), 3 noted (F4, F5, F9), 0 folded, 0 new tickets.

**Rework scope (and nothing else):** F2 — normalise the path before rendering advice; F3 — guard
`noteIfStageable` on the child actually being a git repository. Both are one-liners in code this
branch authored, both ship a false statement to the user today, and neither is eligible for
`fixed inline` because both change behaviour rather than prose.

### Rework 2026-08-07 (`cbf169c`)

Both blocking findings fixed on the same branch; nothing else touched.

| id | fix | test that pins it |
|---|---|---|
| F2 | `State.Advice` now `path.Clean`s `relPath` before rendering, so `./x` and `x/` both render `x/` and the entry `"/x/"`. `path`, not `path/filepath`: config paths and gitignore patterns are slash-form on every platform. | `TestAdviceEntryActuallyIgnores` (`internal/vcs/vcs_test.go`) — asserts the **property**, not the spelling: it writes the advised entry into `.gitignore` and requires the child to flip to `Ignored`, for `child`, `./child` and `child/`. |
| F3 | `noteIfStageable` now mirrors doctor's guard — `os.Stat(<root>/<path>/.git)` and return early when absent — so a non-repo child is doctor's error to report and nothing else claims it is a nested git repository. | `TestProjectAddSilentOnNonRepoChild` (`internal/cli/cli_test.go`) — no `note:` at registration, and doctor still errors, without the contradictory wording. |

Both regression tests were **verified against the pre-fix code**: reverting F2's one-liner fails
`TestAdviceEntryActuallyIgnores/./child` with `after writing the advised entry "/./child/" to
.gitignore, ChildState("./child") = 3, want Ignored`, and dropping F3's guard fails
`TestProjectAddSilentOnNonRepoChild` on the stray `note:`. A regression test that cannot fail
would have been the wrong deliverable here.

Re-verified end to end in a throwaway dir: the original acceptance test is unchanged
(`0 error(s), 1 warning(s)` → `0 error(s), 0 warning(s)` after the advised entry, `ok: … is
git-ignored` under `-v`); `./dot` now advises `/dot/`, which silences the warning; a plain
directory registers with no note and is reported only by doctor, as an error. All four gates
green.

**Not done, deliberately:** F4/F5/F9 keep their `noted` disposition — rework scope is the
blocking findings only. F4's `install --path` test was flagged in the review as something the
rework could pick up "if it touches those lines anyway"; it did not — the F3 guard sits in
`noteIfStageable`, not at the install call site — so it stays noted rather than smuggled in.

**Impact sweep (step 8):** no ticket lists T-051 in `depends-on:`. **T-052** (soft-coupled, same
onboarding session) is unaffected — its premise is the post-`upgrade` board-staleness verdict,
which this branch does not touch; the `project add` → `upgrade` sequence still produces it.
**T-083** cites T-051 only as an example of outcome-first ticket prose. No ticket patch required.

### Scoped re-review 2026-08-07 (`81dcfc5` + `cbf169c`)

Scoped to the two rework commits and whether F2/F3 are genuinely fixed; `9e775f9` was not
re-litigated. Audit again delegated to a fresh sub-agent, every finding re-verified by hand.

**Verified green:** F3 is fully fixed, including the case the fix could plausibly have missed —
`os.Stat` succeeds for a `.git` *file*, so a worktree/submodule child is treated as a repository
by `noteIfStageable` and `checkChildren` alike; no state was found where the two disagree. F2 is
fixed for `./x`, `x/`, `a//b` and nested `apps/frontend` (the anchored `/apps/frontend/` entry
really does silence it); absolute paths are rejected earlier by `project add`. Both regression
tests were independently re-derived as failing pre-fix, so the claim that they can fail is now
confirmed by someone other than their author. All four gates green; acceptance test verbatim;
`git diff --stat main...HEAD -- tickets/` empty, so the feature branch still carries no
bookkeeping.

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | **blocking** | — | The F2 fix was applied at the render site but not at the gate, so a path that *cleans to* `"."` (`./`, `sub/..`) slips past the `p.Path == "."` string comparisons in both callers. pickle then calls **the repository root itself** a nested git repository and advises `add "/./"`, which git does not honour — the same un-silenceable advice F2 was raised for, one layer up. Reachable through the real CLI. | `./pk project add c ./` → `note: ./ is a nested git repository … add "/./"`; `pickle.toml` stores `path = "./"`; after `printf '/./\n' >> .gitignore`, `git check-ignore -q -- .` still exits 1 and doctor still reports `1 warning(s)` | Clean once at the gate rather than only at render: `path.Clean(p.Path) == "."` in `noteIfStageable` (`internal/cli/project.go`) and in `checkChildren` (`internal/doctor/doctor.go`). Fixing it in `Advice` alone would still leave the root mislabelled. |
| R2 | non-blocking | fixed inline | A child whose directory name contains a gitignore metacharacter (`[`, `]`, `\`) yields an entry matching something other than itself. | `printf '/foo[1]/\n' > .gitignore; git check-ignore -q -- 'foo[1]'` → exit 1 | Recorded as a deliberate limit in `internal/vcs`'s package doc rather than escaped — the wrong escape would be worse than none. Done in `1bc612b`. |
| R3 | non-blocking | fixed inline | The F7 caveat *I wrote in `81dcfc5`* understated its own bug: when the root sits inside another repo the advised entry is not merely in the wrong file, it is the wrong *pattern* — anchored at root (`/child/`) where the enclosing repo needs `/sub/child/`. | with root `outer/sub`, writing `/child/` in `outer/.gitignore` still warns; only `/sub/child/` silences it | Caveat rewritten to say the entry is mis-anchored and only indicative in that layout. Done in `1bc612b`. |
| R4 | non-blocking | fixed inline | Also *self-inflicted in `81dcfc5`*: the reworded `Tracked` ok-line promised "— not stageable by accident", which is false for exactly the child F8 describes, and contradicted the comment immediately above it saying the package cannot tell the two apart. A fix for an over-claim that introduced a new one. | `ok: child "kid" is tracked by this repository (kid) — not stageable by accident`, yet `git add kid` then stages `A kid/b.txt` | Editorial clause dropped; the line now says what git reports and nothing more. Done in `1bc612b`. |
| R5 | non-blocking | folded | `TestAdviceEntryActuallyIgnores` covers only three flat forms — which is why R1 survived a test written to pin the property. The table, not the technique, was too narrow. | `internal/vcs/vcs_test.go` table `{"child", "./child", "child/"}` | Folded into R1's rework scope: the table gains `./` (catches R1) and `apps/frontend`. |
| R6 | non-blocking | fixed inline | Ragged short line mid-paragraph left by F6's rewrite. | `docs/user-manual/cli-reference.adoc:174-175` | Re-wrapped. Done in `1bc612b`. |

**Disposition summary:** 6 findings — **1 blocking** (R1 → `5-rework/`); 5 non-blocking:
4 fixed inline (R2, R3, R4, R6, all in `1bc612b`), 1 folded into R1 (R5), 0 noted, 0 new tickets.

**Rework scope (and nothing else):** R1 — clean the path at the `"."` gate in both callers; plus
R5's two extra rows in the advice test table.

**Note on this cycle.** Three of the six findings (R3, R4, and R1 itself) are defects in the
review-and-rework commits rather than in the original implementation — the corrections were
sloppier than the code they corrected. R1 in particular is the first fix applied to the symptom
(`"/./x/"` rendering) instead of the class (an uncleaned path reaching the gate), which is why a
second, structurally identical bug survived it.

### Rework 2 — 2026-08-07 (`87b1b65`)

R1 fixed; R5 folded in. Nothing else touched.

| id | fix | test that pins it |
|---|---|---|
| R1 | The gate is now a single predicate, `vcs.IsRepoRoot(relPath)` (`path.Clean(relPath) == "."`), asked by **both** `noteIfStageable` and `checkChildren` instead of each comparing the raw string. Cleaning inside `Advice` alone would have left the root mislabelled — the defect was the gate, not the rendering. | `TestIsRepoRoot` (`internal/vcs/vcs_test.go`) over `.`, `./`, `sub/..`, `./.`, `a/../` and the negative cases; `TestProjectAddSilentOnRootSpelledDotSlash` (`internal/cli/cli_test.go`) pins it end to end — no `note:` at registration, no doctor warning. |
| R5 | The advice round-trip table gains the nested form `apps/frontend`. | `TestAdviceEntryActuallyIgnores` |

**Deviation from the review's literal suggestion, deliberate.** R5 proposed adding `./` to
`TestAdviceEntryActuallyIgnores`. That row would have encoded the wrong expectation: for a
root-denoting path the correct behaviour is *never to reach advice at all*, not to render advice
that round-trips. Putting it in the advice table would have demanded a usable `.gitignore` entry
for the repository root, which does not exist. The case is covered at the gate instead
(`TestIsRepoRoot` plus the CLI test), and the table comment now says why root forms are absent.

Both new tests were **verified against the pre-fix gate**: restoring the raw `relPath == "."`
comparison fails `TestIsRepoRoot` on all four non-canonical spellings and fails
`TestProjectAddSilentOnRootSpelledDotSlash` with the original symptom, `note: ./ is a nested git
repository … add "/./"` plus the matching doctor `WARNING`.

**Considered and rejected:** normalising `path` at registration so `pickle.toml` stores `.`
rather than `./`. It is the deepest fix, but it changes what is written to the config file and
every other consumer of `p.Path` would inherit the change — too wide for a rework scoped to one
finding. `IsRepoRoot` makes the callers correct regardless of how the path was spelled, and a
normalise-on-write change remains available later without contradicting it.

Re-verified end to end: `./` registers with no note and doctor reports `0 error(s),
0 warning(s)`; the original acceptance test still reproduces verbatim; `./dot` still advises
`/dot/`. All four gates green.

### Scoped re-review 3 — 2026-08-07 (`1bc612b` + `87b1b65`) — PASS

Scoped to the cycle-2 correction commits. Audit again delegated to a fresh sub-agent, findings
re-verified by hand. **0 blocking.**

**Verified green:** R1 is fixed at the class level — `vcs.IsRepoRoot` is used by *both* gates and
a repo-wide grep found no third raw `== "."` comparison on a project path. Crucially, the
reviewer checked the question I had not: whether any *other* consumer of `p.Path` is vulnerable
to the `./` spelling. None is — every other `cfg.Projects` loop keys on `p.Name`, and the
remaining `p.Path` uses go through `filepath.Join`, which normalises. An empty path is
unreachable (`Validate` rejects it) and would fail safe. Absolute paths are rejected at
`project add`; a hand-edited one yields git 128 → `Unknown` → silent. Both new tests were
independently re-derived as failing pre-fix. R2, R3, R4, R6 confirmed by running git rather than
by reading the prose. Acceptance test verbatim; all gates green; no bookkeeping on the branch.

The R5 deviation was explicitly judged **sound rather than a dodge**: no `.gitignore` entry can
ignore the repository root, so a root row in an advice *round-trip* table would have had to
assert an impossible property.

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | fixed inline | The metacharacter caveat added in `1bc612b` lumped `*`/`?` together with `[`/`]`, but they misfire in *opposite* directions: a bracket entry fails to match the child (warning unsilenced), while `/foo*/` matches the child **and every sibling starting `foo`** — silently over-ignoring, the worse of the two, and undocumented. | dir `foo*` with `/foo*/`: `check-ignore -q -- 'foo*'` → 0 **and** `check-ignore -q -- foobar` → 0 | Both directions now described, with the over-ignoring case named as the worse. Done in `59e2f64`. |
| N2 | non-blocking | fixed inline | Same caveat block: "pickle **cannot** render a usable entry for that layout" is a false capability claim — precisely the class R3 and R4 were raised for, committed while fixing R3 and R4. A usable entry exists; pickle simply does not compute the prefix. | `/sub/child/` in the enclosing `.gitignore` silences it (exit 0); `git -C sub rev-parse --show-prefix` → `sub/` | Reworded to "does not compute", naming `--show-prefix`. Done in `59e2f64`. |
| N3 | non-blocking | fixed inline | After R1 the exemption is "any path that cleans to `.`", but `checkChildren`'s doc comment and two lines of the manual still said "a path other than `.`" — the code had outgrown its description. | `doctor.go:281-288`, `cli-reference.adoc:171,264` | Both now say "the repository root itself". Done in `59e2f64` (paragraph re-wrapped again, since the longer phrase re-raggedized R6's fix). |

**Disposition summary:** 3 findings — 0 blocking; 3 non-blocking, all fixed inline (`59e2f64`);
0 noted, 0 folded, 0 new tickets.

**Reviewer's judgement on whether to continue:** explicitly *not* churning. The R1 fix addressed
the class rather than the symptom and held against every spelling reachable through the real
CLI; N1–N3 are wording, and none would mislead an operator into a wrong action.

**Cycle tally, for the record.** Three review cycles: 2 blocking findings in cycle 1 (F2, F3),
1 in cycle 2 (R1), 0 in cycle 3. Six of the eighteen findings were defects in *correction*
commits rather than in the original implementation — twice in prose asserting a capability the
code did not have, which is why cycle 3 verified every factual claim by running git instead of
reading. The original implementation (`9e775f9`) needed two behavioural fixes; the reviews of it
needed four.

## History

- 2026-07-27 — created (TO DO). source: idea — field finding from adding a second child-project to the `unity` workspace with pickle 0.1.0
- 2026-08-07 — TO DO → READY: plan complete
- 2026-08-07 — READY → IN DEVELOPMENT: picked up
- 2026-08-07 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-07 — IN REVIEW → REWORK: review: 2 blocking (F2 unusable advice for ./x paths, F3 non-repo child mislabelled)
- 2026-08-07 — REWORK → IN REVIEW: rework: F2/F3 fixed, regression tests verified against pre-fix code
- 2026-08-07 — IN REVIEW → REWORK: re-review: R1 blocking — path cleaning to '.' slips the gate, root mislabelled
- 2026-08-07 — REWORK → IN REVIEW: rework 2: R1 fixed via vcs.IsRepoRoot gate, R5 folded
- 2026-08-07 — IN REVIEW → DONE: review: 0 blocking after 3 cycles; 3 fixed inline (N1-N3)
- 2026-08-07 — merged to main (PR #20, ee1086e); kept full history (6 commits across 3 review cycles); branch deleted
