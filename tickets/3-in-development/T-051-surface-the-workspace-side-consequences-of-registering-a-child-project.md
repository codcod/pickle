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

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: idea — field finding from adding a second child-project to the `unity` workspace with pickle 0.1.0
- 2026-08-07 — TO DO → READY: plan complete
- 2026-08-07 — READY → IN DEVELOPMENT: picked up
