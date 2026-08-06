---
id: T-057
title: guard ticket bookkeeping from landing on a feat/ branch (pre-commit hook)
project: pickle
depends-on: []
spawned-by: [T-054]
impact: high
complexity: high
cost: L
---

# T-057 — guard ticket bookkeeping from landing on a feat/ branch (pre-commit hook)

## Description

The flow's commit policy splits every change in two: **code** goes on the child-project's
`feat/T-NNN-<slug>` branch, **bookkeeping** (`tickets/`, `BOARD.md`) is committed on the base
branch. Nothing enforces that split, and it has now been violated three times in this repo:

| when | what | repair |
|---|---|---|
| T-053 | two bookkeeping commits rode the branch; the squash-merge ate them | `59dc0fd docs(tickets): restore T-053 bookkeeping after the squash merge` |
| T-054 | the in-review move committed on the branch | caught as review finding **Q1**; branch reset, commit cherry-picked to `main` |
| T-054 again | `T-056`'s ticket file committed on the branch **while closing the review that flagged Q1** | noticed only because `pickle board audit` reported 55 tickets instead of 56 |
| T-022 | the *inverse* case, and a new failure mode: bookkeeping went to `main` correctly (`b1621da`), but the review branch was cut one commit earlier, so the reviewer checking out the feature branch (as the review protocol instructs) read `tickets/3-in-development/` and opened by recording the handback move as missing | caught as review finding **F6**; the review's own bookkeeping was moved off the branch onto `main` and the branch rebased |

Four occurrences, one of them immediately after the same mistake was written up as a finding,
is not inattention — it is a missing guardrail. The failure is silent at the moment it happens
(`git commit` succeeds, the board still renders) and only surfaces later, either as a wrong
ticket count or as bookkeeping destroyed by a squash-merge.

**The split has a second, opposite hazard the fix must not ignore** (T-022's row above). Doing it
*right* — bookkeeping on the base — makes the ticket's true status **invisible from the feature
branch**, which is precisely where `review-protocol.md` step 1 sends the reviewer to locate the
ticket. Whichever enforcement lands, refinement should decide what the reviewer is told to do
about it: read `tickets/` from the base branch rather than the checkout, or have the tooling say
so. The repo is not yet consistent either way — T-019's bookkeeping rode its branch and survived
only because that PR was merged, not squashed.

### This is not a self-hosting quirk — it affects default installs

The obvious dismissal is "pickle self-hosts, so the overarching repo *is* the child; a normal
install keeps `tickets/` in a different repo from the code, where this cannot happen." That is
wrong. `pickle install` registers the first child with **`--path` defaulting to `.`**
(`internal/install/install.go:95`; documented at `cli-reference.adoc:69`), so the **default
install is single-repo** — `tickets/` and the code share one tree and one branch namespace.
Every default installation has this hazard; pickle's own repo is simply where it has been
observed, because it is the one being worked in daily.

### Design question the refinement must settle: hook or guardrail?

Pickle already ships a guard for git behaviour — the pi extension
`agents/pi/extensions/pickle-guardrails.ts`, embedded in the payload and installed to
`.pi/extensions/`. It enforces staging discipline, a publish gate and a self-modify rule. A
fourth rule there would be cheap and consistent with existing machinery.

**But a pi extension only guards a pi session.** All three violations above were made by an
agent shelling out to `git` outside any such hook. A `pre-commit` hook guards the repository
regardless of who or what is committing — agent, human, or script — which is the property
actually needed here. The likely answer is *both*: the hook as the real enforcement, a
guardrail rule as the fast, in-session explanation. Refinement must decide, and must confront
what makes hooks awkward:

- `.git/hooks/` is not version-controlled, so the hook has to be **installed** — `core.hooksPath`
  pointing at a tracked directory is the usual answer, but it is global to the repo and would
  collide with a project that already sets it (Husky, pre-commit.com, Lefthook).
- `pickle install` writing into a user's git config is materially more invasive than anything
  it does today, so it likely needs to be **opt-in** (`--hooks`) and removed by
  `pickle uninstall`.
- The rule needs an **escape hatch** (`--no-verify` already exists, but the message should name
  the legitimate case) and must not fire on the base branch, on a detached HEAD, during a
  rebase/merge, or in a repo where `tickets/` is genuinely part of the product — which is
  exactly pickle's own case for `skill/`, though not for `tickets/`.
- The branch-name test must come from config (`branch_prefix` in `pickle.toml`, default
  `feat/`), not a hardcoded literal.

A cheaper, near-zero-risk subset worth pricing separately during refinement: have
**`pickle board audit` fail** when the current branch matches `branch_prefix` and `tickets/`
has staged or committed changes not on the base. That reuses machinery that already exists and
already runs, needs no git config, and `DESIGN.md` §7 already anticipates wiring audit into
"CI + a pre-commit hint".

> **Settled by refinement (2026-08-05).** Hook only, as the real enforcement; no pi-guardrail
> rule and no `board audit` check. See the Implementation Plan's *Confirmed decisions* for each
> answer and why.

### Soft couplings

- **T-050** — the pi guardrail's verdict semantics (hard block vs confirm). If this ticket adds
  a fourth rule there, it inherits whatever T-050 decides; a hard block for a rule this
  mechanical is likely right, but T-050 is the ticket that reasons about that choice. No hard
  dependency; whichever lands first, the other adapts.
- **T-046** — same self-host-awareness family, different subsystem (doctor/upgrade). Worth
  reading together, not merging.
- Lineage: `spawned-by: T-054`, whose review raised the violation as finding Q1 and then
  reproduced it.

## Implementation Plan

> **Re-refined 2026-08-05 (second pass).** The pickup applicability gate found four **blocking**
> mechanical defects; the user routed the ticket back to READY, and this pass applied all eight of
> the gate's `fixed inline` amendments. The design (decisions 1–10) is unchanged — what changed is
> the branch-name git call (decision 7), the shim's exit-code contract (decisions 3–4), git
> cwd/environment discipline and no-git degradation (new decisions 11–12), and an acceptance
> transcript that now runs verbatim.

### Pickup gate findings (2026-08-05)

Run by a fresh sub-agent per rules §8; every blocking finding re-verified by hand before being
recorded here. Severity + disposition per rules §5. **All eight `fixed inline` items are applied
in the plan below**; the table stays as the permanent evidence record (rules §5: `noted` is not
"ignored"), with the amendment each one produced named in its row.

| # | finding | evidence | severity | disposition |
|---|---|---|---|---|
| B1 | Acceptance step 4 cannot pass: step 3's rejection leaves `tickets/` **in the index**, so the "code only" commit stages `code.txt` *on top of it* and is rejected too; the end state collapses into one commit holding both. Needs `git restore --staged tickets/` after step 3 (which also exercises the remedy the message prints) and a re-`git add tickets` before step 5. | simulated the decision-7 predicate as a real hook | blocking | fixed inline |
| B2 | Predicate step 1 uses the wrong git call. On an **unborn branch** (`git init -b main`, no commits) `git rev-parse --abbrev-ref HEAD` fails: `fatal: ambiguous argument 'HEAD'`, rc=128, prints `HEAD` — so a fresh repo is misread as detached and bookkeeping on `feat/…` slips through via fail-open. `git symbolic-ref --quiet --short HEAD` is correct: `main` rc=0 on an unborn branch, `''` rc=1 when detached. Transcript step 2 also passes by fail-open rather than by the rule, so it proves nothing — it needs a commit first. | re-verified locally, git 2.55 | blocking | fixed inline |
| B3 | The shim is fail-**closed** on version skew, inverting decision 4. `pickle hooks run pre-commit \|\| exit 1` treats *any* non-zero as a violation, and an older `pickle` first on `PATH` (`go install`, a second clone, Homebrew lag) exits `exitUsage`=2 on an unknown verb — **every commit in the repo blocked**. Needs a reserved exit code for "violation" (only `exitError`=1) with everything else passed through, or a `--probe` handshake. | `internal/cli/cli.go:26,70` | blocking | fixed inline |
| B4 | `Upgrade`/`doctor` wiring assumes a real git repo, but the existing fixtures fake one with an empty dir (`internal/doctor/doctor_test.go:30`, `internal/cli/cli_test.go:191`), where `git rev-parse --path-format=absolute --git-path hooks` exits 128. `hook.Refresh` in `Upgrade` and `checkHooks` in `Check` would error in every current test root and in any non-git install. The plan must specify: not-a-repo → silent no-op (upgrade) and one `ok` line (doctor). Also worth stating: this is pickle's **first** `os/exec` use (0 hits repo-wide today). | verified | blocking | fixed inline |
| F5 | The rebase branch of the in-progress check is dead code: `rebase --continue` never runs `pre-commit`, and during a rebase HEAD is detached anyway (step 1 already passes). `--amend`, a conflicted merge's `git commit`, and `cherry-pick --continue` **do** run it, so `MERGE_HEAD`/`CHERRY_PICK_HEAD`/`REVERT_HEAD` stay; drop `rebase-merge`/`rebase-apply` or mark it belt-and-braces. | hook-logging probe | non-blocking | fixed inline |
| F6 | The manual currently states the **opposite** of what task 6 writes down: "a feature branch can hold the code change *and* the ticket's move to `4-in-review/`". Task 7 says only "state the split" — it must **replace** that sentence, or the manual contradicts the hook. (`review-protocol.md:30-32` and `SKILL.md:192,234` are correctly covered by 6b/6c.) | `docs/user-manual/concepts/project-structure.adoc:139-142` | non-blocking | fixed inline |
| F8 | Git env leaks into both the implementation and the tests: the hook runs with `GIT_INDEX_FILE=.git/index` (**relative**), `GIT_PREFIX=<subdir>/`, and in a linked worktree `GIT_DIR=…/.git/worktrees/<wt>` — so any `git -C <dir>` where `<dir>` ≠ the hook's cwd reads the wrong index/repo. Tests must clear `GIT_DIR`/`GIT_INDEX_FILE`/`GIT_WORK_TREE` and pin `HOME`/`GIT_CONFIG_GLOBAL`, or a developer's global `core.hooksPath` makes the install test write outside `t.TempDir()`. | env probe | non-blocking | fixed inline |
| F12 | Transcript hygiene: the closing "verify `pickle doctor` reports no marker drift" sits *after* step 8's `pickle uninstall` has stripped the markers (move it before step 7); `git add pickle.toml tickets AGENTS.md` is a partial stage (leaves `CLAUDE.md`, `.agents/…`, `.claude/skills/ticket-flow` untracked) and should say so; and the transcript cannot be pasted into a pi session in this repo — `.pi/extensions/workspace-guardrails.ts:102` blocks any segment matching `pickle install` that does not target `/tmp`, so it must be run from a script under `/tmp`. | verified | non-blocking | fixed inline |
| F7 | `--amend` is guarded only against the index: the hook does run on `--amend`, but `git diff --cached` is index-vs-HEAD, so amending a commit that *already* contains `tickets/` shows nothing. Acceptable — but the docs should say it rather than imply amends are covered. | — | non-blocking | noted |
| F9 | Fail-open plus `PATH` lookup makes the guard silently absent in GUI/IDE commits (Fork, SourceTree, JetBrains all run with a minimal `PATH` lacking `/opt/homebrew/bin` and `$GOPATH/bin`), so `command -v pickle` fails. One honest sentence in `cli-reference.adoc`: the guard is best-effort and terminal-first. | — | non-blocking | noted |
| F10 | `cli-reference.adoc` overlaps T-066's declared file ownership ("No file overlap (T-066 owns `cli-reference.adoc`)"), and dead anchors are unvalidated today — which is T-067's whole premise — so a broken xref to a new `[#cmd-hooks]` would not be caught. Keep the new section self-contained and re-check both tickets at implementation time. Recorded here rather than added to T-066: T-057's section does not exist yet, so an item there would be speculative. | `tickets/1-to-do/T-067-…md:74` | non-blocking | noted (cross-ref T-066, T-067) |
| F11 | `just docs-check` is not in CI (`.github/workflows/ci.yml` runs vet/gofmt/test/build only), so `snowball check` is local-only and task 7's docs breakage could land unnoticed. The new hook tests themselves are CI-safe: git exists, the plan sets `user.email`/`user.name` locally, and the file skips without git. | verified | non-blocking | **folded → T-067** (its ground: the docs pipeline validates nothing; item added there 2026-08-05) |

**Where each `fixed inline` amendment landed:** B1 + F12 → the rewritten *Acceptance test*;
B2 + F5 → decision 7 steps 1–2 and task 1's `PreCommit`; B3 → decisions 3–4, `Shim()` in task 1,
and task 3's exit-code contract; B4 → new decision 12 and task 5; F8 → new decision 11 and
task 2's env hygiene; F6 → task 7's first bullet.

### Feature branch

`feat/T-057-bookkeeping-pre-commit-hook`, cut from `main` in the `pickle` child (this repo).
Local WIP commits are encouraged; **no push and no MR without explicit user approval** — then
finalize + push + open the MR; merging is the human's.

**This branch is itself subject to the rule it ships:** every `tickets/` edit below (this plan,
the move to `3-in-development/`, the move to `4-in-review/`) is committed **on `main`**, never on
the feature branch.

### Prerequisites

None. No `depends-on:`. T-050 (guardrail verdict semantics) is untouched by this ticket — the
decision below is *not* to add a fourth guardrail rule, so nothing is inherited from it.

### Confirmed decisions

1. **Hook only; no pi-guardrail rule.** All four recorded violations were an agent shelling out
   to `git`. A `pre-commit` hook fires there too — including inside a pi session — so a fourth
   rule in `agents/pi/extensions/pickle-guardrails.ts` would only restate the hook's message in a
   second language, with no access to `pickle.toml`'s `branch_prefix`. Do not add one.
2. **No `board audit` check.** `internal/audit` is git-free and its tests are plain temp dirs;
   shelling out to `git rev-parse`/`merge-base` there would make the audit environment-dependent
   and would need a `base_branch` config key that does not exist. Keeping the audit git-free is a
   deliberate property — state it in the code comment that declines the check (`DESIGN.md` §7,
   task 8).
3. **The hook is a shim; the rule lives in Go.** `.git/hooks/pre-commit` is a short `sh` script
   that runs `pickle hooks run pre-commit`. The rule therefore reads live `pickle.toml`
   and can never go stale, and it is unit-testable. A generated self-contained `sh` script with
   the prefixes baked in was rejected for going stale silently.
   **Exit-code contract (finding B3):** `pickle hooks run pre-commit` exits **`1` if and only if
   the commit is a violation**, `0` when the commit is allowed *or* the guard degraded, `2` for a
   usage error. The shim blocks on `1` and on nothing else — `|| exit 1` would have been
   fail-*closed*, because an older `pickle` first on `PATH` (a second clone, `go install`, Homebrew
   lag) exits `2` on the unknown verb (`internal/cli/cli.go:26,70`) and would have blocked **every
   commit in the repo**. This contract is part of the CLI's surface: never reuse exit `1` in
   `hooks run` for anything but a violation.
4. **The guard fails open, always.** If `pickle` is not on `PATH`, if no `pickle.toml` is found, if
   the config fails to parse, if `git` is missing, or if the tree is not a git repo, the guard
   **exits 0**. A missing or misconfigured guard must never brick `git commit`. On an *unexpected*
   exit code the shim prints one stderr line naming the code and continues — silence there would
   hide a dead guard, which is the failure mode this whole ticket exists to prevent.
5. **`.git/hooks/pre-commit`, not `core.hooksPath`.** Setting `core.hooksPath` is repo-global and
   collides with Husky / pre-commit.com / Lefthook. Resolve the target directory with
   `git rev-parse --path-format=absolute --git-path hooks`, which **honours an existing
   `core.hooksPath`** (verified, git 2.55) — so a project that already redirects its hooks gets
   the shim in the right place instead of a silently-dead one.
6. **Opt-in, presence-based ownership, no config key.** `pickle hooks install` is explicit;
   `pickle install --hooks` is a convenience. Ownership is recorded **in the file** as a
   `# pickle:hook v1` marker line, exactly like the pi scaffolds and `opencode.jsonc` — *not* as a
   key in `pickle.toml`.
   *(This amends the refinement discussion's "record `hooks = true` so doctor isn't blind":
   `Config.Save` re-renders and drops comments, so persisting intent would need
   `SetPayloadVersionInPlace`-grade in-place editing for marginal value, and would add a second
   source of truth that can disagree with the file on disk. Instead `pickle doctor` reports the
   hook's state unconditionally and, when it is absent, hints at `pickle hooks install` — which
   also covers the case a config key was wanted for: a fresh clone, where hooks are never carried
   over.)*
7. **The predicate.** Reject a commit when **all** of: HEAD is a branch (not detached);
   its name starts with the `branch_prefix` of **any** registered child (union; default `feat/`);
   and the staged paths intersect the `tickets/` directory of the `pickle.toml` that governs this
   repo. Pass otherwise. Explicitly pass during a merge / cherry-pick / revert in progress. Only
   `tickets/` is guarded — never `pickle.toml`, never `skill/`.
   **The branch name comes from `git symbolic-ref --quiet --short HEAD`, never from
   `git rev-parse --abbrev-ref HEAD`** (finding B2): on an **unborn** branch — a freshly
   `git init`-ed repo with no commits, which is exactly what `pickle install` lands in — `rev-parse`
   exits 128 and prints the literal `HEAD`, which reads as "detached" and would wave bookkeeping
   through on `feat/…`. `symbolic-ref` returns `main` there (exit 0) and empty with exit 1 when
   genuinely detached. Verified on git 2.55.
8. **Escape hatch.** `git commit --no-verify`, named in the rejection message together with the
   legitimate case (a change whose *product* lives under `tickets/`).
9. **The rule gets written down.** The split it enforces is currently stated **nowhere** in the
   shipped payload; enforcing an unwritten rule is not acceptable, so tasks 6a–6d add it, and fix
   the inverse hazard the split creates for reviewers (T-022 finding F6) in the same pass.
10. **Not split.** The docs half is arguably schedulable alone, but a hook without the written
    rule enforces something undocumented, and the written rule without the hook is exactly
    today's state (four violations). One ticket.
11. **Git cwd/env discipline (finding F8).** In the `hooks run` path **every git call inherits the
    hook's working directory and environment — no `-C`**. Git invokes `pre-commit` from the
    worktree top with `GIT_INDEX_FILE` set to a **relative** path and, inside a linked worktree,
    `GIT_DIR` pointing at `.git/worktrees/<wt>`; a `git -C <root>` call would therefore inspect a
    *different index* than the one being committed. `-C` is used only by `Install`/`Uninstall`/
    `Status`/`Refresh`, which the user invokes from an arbitrary directory with no `GIT_*` set.
    Config discovery for `hooks run` is `config.Find(cwd)`, so the governing `pickle.toml` is the
    one above the commit, not one passed in.
12. **Degrade silently where there is no git (finding B4).** `git` missing or the tree not a
    repository is a normal state, not an error: `Refresh` no-ops, `doctor`'s `checkHooks` emits one
    `ok` line, `hooks run` exits 0, and `Install` is the *only* entry point that reports it as a
    failure (the user asked for a hook and cannot have one). This matters because the existing test
    fixtures fake a child repo with an empty directory (`internal/doctor/doctor_test.go:30`,
    `internal/cli/cli_test.go:191`). Related: this ticket introduces pickle's **first** `os/exec`
    use (0 hits repo-wide today) — confine it to `internal/hook` behind the single `git()` helper so
    the rest of the tree stays exec-free and `internal/audit` stays git-free (decision 2).

### Tasks

**1. New package `internal/hook/hook.go`.** No dependency on `internal/install` (avoids a cycle;
`install` imports `hook`, not the reverse). Shells out to `git` via **two** unexported helpers, so
decision 11 is structural rather than a habit: `gitAt(dir string, args ...string)` (used only by
Install/Uninstall/Status/Refresh) and `gitHere(args ...string)` (no `-C`, inherits cwd + env; the
**only** helper the `PreCommit` path may call).

- `const ShimVersion = 1`; `const marker = "# pickle:hook v1"`.
- `Shim() string` — the script text, ending in a trailing newline. Note the exit-code handling: it
  is the whole of decision 3's contract, so it must not be "simplified" back to `|| exit 1`:
  ```sh
  #!/bin/sh
  # pickle:hook v1 — installed by `pickle hooks install`, removed by `pickle hooks uninstall`.
  # Refuses ticket bookkeeping (tickets/) staged on a feature branch. The rule lives in the
  # binary so it tracks pickle.toml. Bypass one commit with `git commit --no-verify`.
  command -v pickle >/dev/null 2>&1 || exit 0   # guard absent, never blocking
  pickle hooks run pre-commit
  rc=$?
  [ "$rc" -eq 1 ] && exit 1                     # 1 = violation, and only 1
  [ "$rc" -ne 0 ] && echo "pickle: bookkeeping guard skipped (hooks run exited $rc)" >&2
  exit 0
  ```
- `HooksDir(root string) (string, error)` — `git -C root rev-parse --path-format=absolute
  --git-path hooks`; on a git too old for `--path-format`, fall back to `--git-path hooks`
  resolved against `root`. Error ("not a git repository") when `root` is not one.
- `Install(root string, force bool) (Result, error)` — `MkdirAll` the hooks dir; if `pre-commit`
  exists and does **not** contain `marker`, refuse with a message naming `--force` and the
  one-line snippet to chain by hand; write `Shim()` with mode `0o755`.
- `Uninstall(root string, dryRun bool) (Result, error)` — remove only when `marker` is present;
  a foreign hook is reported as skipped, never touched.
- `Status(root string) (State, error)` — `Absent` / `Owned{Version, Stale bool}` / `Foreign`,
  plus the resolved path (so `hooks status` can show a `core.hooksPath` redirect).
- `Refresh(root string) (bool, error)` — rewrite an owned-but-stale shim; no-op otherwise.
- `PreCommit(cfg *config.Config, w io.Writer) (ok bool, err error)` — the rule of decision 7. Takes
  **no root**: per decision 11 it works on the cwd's repo via `gitHere`, which is the index git is
  actually committing.
  1. `git symbolic-ref --quiet --short HEAD` — exit 1 / empty → detached → ok. **Not**
     `rev-parse --abbrev-ref HEAD` (finding B2; a comment must say why, or it will be "tidied"
     back).
  2. in-progress operation → ok: any of `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD` exists
     (via `git rev-parse --git-path <name>`). **No `rebase-merge`/`rebase-apply` check** — verified
     dead code (finding F5): `rebase --continue` does not run `pre-commit`, and mid-rebase HEAD is
     detached, so step 1 already returns ok.
  3. no configured `BranchPrefix` matches the branch name → ok.
  4. `git rev-parse --show-toplevel`; compute `tickets/`'s path **relative to that top level**
     from `cfg.Root()`. If it escapes the repo (`..`), → ok (multi-repo child: nothing to guard).
  5. `git diff --cached --name-only -z`; ok unless a staged path is under that prefix. Paths are
     top-level-relative even when the hook runs from a subdirectory — do not join them with
     `GIT_PREFIX`.
  6. On violation write the rejection message to `w`: the branch, the offending staged paths
     (cap the list at 10 plus an `… and N more` line), *why* (a squash-merge of this branch eats
     the bookkeeping; the board then disagrees with the tickets), the remedy
     (`git restore --staged tickets/`, commit the code, commit the bookkeeping on the base
     branch), and the `--no-verify` escape hatch with its legitimate case.
  Not covered, by design (finding F7): `git commit --amend` of a commit that *already* contains
  `tickets/` — the hook does run, but `git diff --cached` is index-vs-HEAD and shows nothing. The
  docs say so rather than implying amends are guarded.

**2. `internal/hook/hook_test.go`.** `t.Skip` the whole file when `exec.LookPath("git")` fails.
Helper builds a real temp repo (`git init -b main`, `user.email`/`user.name` set locally, a
`pickle.toml` + `tickets/` tree via the existing `config` API). Cases:

**Env hygiene (finding F8), applied to every test:** clear `GIT_DIR`, `GIT_INDEX_FILE`,
`GIT_WORK_TREE` and `GIT_PREFIX`, and pin `HOME` + `GIT_CONFIG_GLOBAL` into `t.TempDir()` —
otherwise a developer's global `core.hooksPath` makes the install tests write **outside** the temp
tree. Set `user.email`/`user.name` locally per repo, never globally.

- install → file exists, mode `0o755`, contains the marker; re-install is idempotent;
- a foreign `pre-commit` is refused, survives byte-identical, and `--force` overwrites it;
- `core.hooksPath` set → the shim lands in that directory;
- uninstall removes an owned hook, skips a foreign one, and is idempotent; `dryRun` mutates
  nothing;
- `Status` reports absent / owned / foreign / owned-stale (write a `v0` marker), and `Refresh`
  fixes only the stale-owned case;
- `PreCommit`: `main` + staged `tickets/` → ok; `feat/T-1-x` + staged `tickets/` → violation,
  message names the branch and the path; `feat/T-1-x` + staged code only → ok; mixed staging →
  violation; detached HEAD → ok; a child configured with `branch_prefix = "wip/"` → `wip/…`
  rejected and `feat/…` allowed; `tickets/` outside the repo → ok;
- **regression tests for the four blocking gate findings**, each named after it so the reason
  survives:
  - **B2** — an **unborn** `feat/T-1-x` (`git init -b feat/T-1-x`, no commits) with staged
    `tickets/` → violation. This is the case `rev-parse --abbrev-ref HEAD` waves through;
  - **F5/F7** — with `MERGE_HEAD` present → ok (write the file directly; no need to stage a real
    conflict);
  - **B4** — `PreCommit` in a non-repo directory, and with `git` forced missing via `PATH` → ok,
    no error surfaced to the caller; `Refresh`/`Status` in a non-repo → no-op, no error;
  - **F8** — run `PreCommit` from a **subdirectory** with `GIT_PREFIX` set and a *relative*
    `GIT_INDEX_FILE`, and inside a **linked worktree** (`git worktree add`) → correct verdict in
    both, proving no `-C` crept into the path;
- **B3, at the CLI level** (in `internal/cli/cli_test.go`): `hooks run pre-commit` returns exactly
  `1` on a violation and `0` when allowed or degraded, and an unknown `hooks` subcommand returns
  `2` — the contract the shim's `[ "$rc" -eq 1 ]` depends on.

**3. CLI verb — new `internal/cli/hooks.go`, wired in `internal/cli/cli.go`.**
`pickle hooks install [--force] | uninstall [--dry-run|-n] | status | run pre-commit`. One verb
(`hooks`), no separate `hook` verb. `run pre-commit` is the shim's entry point: it locates
`pickle.toml` with `config.Find(cwd)`/`Load` and, per decisions 3–4, **exits `1` only for a
violation** and `0` on every other outcome — no config found, unparseable config, no git, not a
repo. An unknown subcommand is `exitUsage` (`2`). Document the three exit codes in the handler's
doc comment as a contract the shipped shim depends on. Add the `hooks` block to `usage()`'s *Setup
commands* group.

**4. `pickle install --hooks`** in `internal/cli/install.go` (`runInstall`): after the post-install
audit passes, call `hook.Install(root, false)` and print the created path in the same `  + %s`
style. A failure here prints a warning and does **not** fail the install.

**5. Lifecycle wiring.**
- `internal/install/install.go` → `Uninstall`: remove the pickle-owned hook (honouring
  `UninstallOptions.DryRun`, reported through `res.removed`/`res.skipped` like the pi scaffolds).
- `internal/install/install.go` → `Upgrade`: `hook.Refresh(root)` — refresh an owned, stale shim;
  never install one that is absent; **no-op without error when the root is not a git repo**
  (decision 12 — today's `Upgrade` tests run in exactly such a tree).
- `internal/doctor/doctor.go`: new `checkHooks(root, r)` called from `Check` — owned+current →
  `ok`; owned+stale → warning pointing at `pickle upgrade` (mirrors the agent-scaffold check);
  absent → `ok` line noting it is optional and naming `pickle hooks install`; foreign → `ok` line
  saying the hook is not pickle's and is left alone; **not a git repo / no git → one `ok` line**.
  Never an error and never a *new* warning class: the hook is opt-in, and `checkChildren` already
  owns "is this a git repo" as a real check (`doctor.go:226`) — do not duplicate that verdict here.
- Extend `internal/install/install_test.go`, `internal/doctor/doctor_test.go` and
  `internal/cli/cli_test.go` for the uninstall/upgrade/doctor/CLI paths above. **The existing
  fixtures fake a child repo with an empty directory** (`doctor_test.go:30`, `cli_test.go:191`), so
  they exercise decision 12's no-git path by default — add one fixture that is a *real* `git init`
  repo to cover the owned/stale/foreign branches.

**6. Write the rule into the payload** (the half that makes the hook legitimate):
- **6a** `skill/resources/tickets-README.md` §0 — new bullet **Where commits land**: code on the
  child's `feat/T-NNN-<slug>` branch; ticket/board bookkeeping on the **base branch of the
  overarching repo**; in the single-repo default (`path = "."`) these are one repo and one branch
  namespace, which is what makes the split easy to violate and a squash-merge able to eat it.
  Name `pickle hooks install` as the local enforcement.
- **6b** `skill/resources/review-protocol.md` — the *Review on the ticket's feature branch* box
  and step 1 (*Locate the ticket*): the ticket file and board are authoritative **on the base
  branch**; from a checked-out feature branch read them with `git show <base>:tickets/…` rather
  than trusting the worktree, and record the review's own moves on the base branch. This is
  T-022 finding F6.
- **6c** `skill/SKILL.md` — one clause in the *Project configuration* commit-policy bullet
  pointing at the same split.
- **6d** `internal/install/install.go` → `MarkerBlock()` — extend the **Commit policy** bullet's
  overarching sentence with the base-branch rule (applies to both the `overarching_auto` true and
  false variants). **Self-host mirror:** hand-edit this repo's `AGENTS.md` marker block to match
  `MarkerBlock()`'s new output byte-for-byte, inside this ticket's diff (`CLAUDE.md` is a symlink
  to `AGENTS.md`, so it needs nothing). Do **not** run `pickle install|upgrade` against this repo.

**7. Docs** (`just docs-check` must stay green):
- `docs/user-manual/concepts/project-structure.adoc`: **replace, do not extend**, the sentence at
  lines 139–142 — "so a feature branch **can** hold the code change *and* the ticket's move to
  `4-in-review/`" is the exact opposite of the rule this ticket ships, and leaving it makes the
  manual contradict the hook (finding F6). It becomes: one history carries both roles, which is
  *why* the split must be deliberate — code on the feature branch, ticket/board bookkeeping on the
  base branch — with a pointer to `pickle hooks install`.
- `docs/user-manual/cli-reference.adoc`: new `[#cmd-hooks] == pickle hooks` section after
  `== pickle uninstall`, documenting all four subcommands, the shim, `core.hooksPath` handling,
  the three exit codes, `--no-verify`, and that hooks are per-clone (never cloned). Amend the
  `pickle install` section (`--hooks`), the `pickle uninstall` section (removes the owned hook),
  and the `pickle doctor` bullet list (reports hook state). Three honesty sentences the gate asked
  for: the guard is **best-effort and terminal-first** — GUI/IDE clients (Fork, SourceTree,
  JetBrains) commit with a minimal `PATH` where `command -v pickle` fails and the guard is silently
  skipped (F9); `--amend` is checked against the index only, so amending a commit that already
  contains `tickets/` is not caught (F7); and an older `pickle` on `PATH` degrades to no guard
  rather than to a blocked repo (B3). Keep the section self-contained — T-066 owns this file's
  other gaps and anchors are unvalidated until T-067 (F10).
- `docs/user-manual/installation.adoc`: one short paragraph — after installing, single-repo
  projects should run `pickle hooks install`, once per clone.
- `CHANGELOG.md`: an `### Added` entry under `## [Unreleased]`.

**8. `DESIGN.md` §7:** replace the "still a live recommendation (see T-057)" bullet with what
shipped — the pre-commit hook — and record that the audit-side check was deliberately declined to
keep `board audit` git-free (decision 2).

### Acceptance test

All four child commands, from the repo root:

```sh
just build && just test && just lint && just docs-check
```

Then the end-to-end scenario, in a **throwaway directory with the binary copied in** (self-modify
policy). The shim resolves `pickle` from `PATH`, so the copy is named `pickle` inside a temp `bin`
that is prepended to `PATH` — the policy's point is that no in-repo binary path is referenced:

**Write the transcript below to a file under `/tmp` and run it with `sh -e`** — it cannot be pasted
into a pi session in this repo, because `.pi/extensions/workspace-guardrails.ts:102` blocks any
segment matching `pickle install` that does not clearly target a throwaway dir (finding F12). That
guard is correct and stays; the script is the way through it.

```sh
just build
D=$(mktemp -d /tmp/t057.XXXXXX)   # literal /tmp: bare `mktemp -d` gives /var/folders/… on macOS,
                                 # which the self-modify guard does not recognise as throwaway
mkdir -p "$D/bin"; cp pickle "$D/bin/pickle"; export PATH="$D/bin:$PATH"
mkdir -p "$D/repo" && cd "$D/repo" && git init -q -b main .
git config user.email t@example.com && git config user.name test
pickle install --project demo --hooks
test -x .git/hooks/pre-commit && grep -q 'pickle:hook' .git/hooks/pre-commit   # 1. installed
pickle doctor -v | grep -i hook                                               # 2. state reported,
pickle doctor                                                                 #    no marker drift

# Partial stage on purpose: .agents/, .claude/ and CLAUDE.md stay untracked (F12) — the
# bookkeeping paths are what the guard cares about. `main` here is still unborn: with
# decision 7's symbolic-ref this is a genuine base-branch pass, not a fail-open (B2).
git add pickle.toml tickets AGENTS.md && git commit -qm 'chore: scaffold'      # 3. base: allowed

git checkout -qb feat/T-001-demo
pickle ticket new "demo" --project demo >/dev/null
before=$(git rev-parse HEAD)
git add tickets && git commit -m 'docs(tickets): file T-001'                   # 4. MUST FAIL (rc 1)
test "$before" = "$(git rev-parse HEAD)"                                      #    nothing committed

# The index still holds tickets/ after a rejection — unstage it, exactly as the message says,
# or the next commit is rejected for the same reason (B1).
git restore --staged tickets/
echo x > code.txt && git add code.txt && git commit -qm 'feat: code (T-001)'   # 5. code: allowed
git add tickets && git commit -q --no-verify -m 'docs(tickets): file T-001'    # 6. escape hatch
git checkout -q --detach && git commit -q --allow-empty -m 'detached'          # 7. detached: ok
git checkout -q feat/T-001-demo

# 8. Unborn feature branch — the B2 hole, in its own repo (no commit exists yet).
mkdir -p "$D/unborn" && cd "$D/unborn" && git init -q -b feat/T-001-x .
git config user.email t@example.com && git config user.name test
pickle install --project demo --hooks >/dev/null
pickle ticket new "demo" --project demo >/dev/null
git add tickets && git commit -m 'docs(tickets): file T-001'                   #    MUST FAIL (rc 1)
cd "$D/repo"

# 9. Old-binary skew must degrade to "no guard", never to a blocked repo (B3). Stage the
#    bookkeeping with the real binary FIRST — the stub shadows `pickle` entirely.
pickle ticket new "skew" --project demo >/dev/null
mkdir -p "$D/oldbin" && printf '#!/bin/sh\nexit 2\n' > "$D/oldbin/pickle"
chmod +x "$D/oldbin/pickle"
git add tickets
( PATH="$D/oldbin:$PATH"; git commit -qm 'skew: allowed with a warning' )      #    MUST SUCCEED

pickle hooks status                                                           # owned, current, path
pickle hooks uninstall && pickle hooks status                                 # 10. absent
printf '#!/bin/sh\nexit 0\n' > .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit
sum=$(cksum .git/hooks/pre-commit)
pickle hooks install || true                                                  # 11. MUST refuse
test "$sum" = "$(cksum .git/hooks/pre-commit)"                                #     unchanged
pickle hooks install --force && grep -q 'pickle:hook' .git/hooks/pre-commit    #     overwrites
pickle uninstall -n | grep pre-commit && pickle uninstall                      # 12. removed
test ! -e .git/hooks/pre-commit
```

Expected: every numbered step succeeds except **4, 8 and 11, which must fail** — 4 and 8 with the
rejection message naming the branch, the staged `tickets/` path and `--no-verify`, and with `HEAD`
unmoved; 11 refusing to touch a foreign hook (byte-identical `cksum`) until `--force`. Step 2 is
placed **before** any uninstall so the marker-drift check still has markers to read (F12), and it
is what proves task 6d's `MarkerBlock()` change agrees with the hand-mirrored `AGENTS.md`. Step 9
is the fail-open regression: with a stub `pickle` that exits 2, the commit **goes through** with a
one-line stderr notice. Re-runnable verbatim from a clean `mktemp -d`.

### Implementation notes (2026-08-06) — deviations from the plan

All 8 tasks shipped on `feat/T-057-bookkeeping-pre-commit-hook` (commit `cc96393`). The plan held
up; what differs, for the reviewer:

1. **`GIT_DIR=` (empty) is not the same as unset** — the first cut of `gitAt` blanked the `GIT_*`
   variables via `append(os.Environ(), "GIT_DIR=", …)`, and git reads that as an empty repository
   path: `fatal: not a git repository: ''`. Every git call failed, so the guard was **inert** and
   fail-open hid it — `pickle doctor` reported "no git repository at the install root" *inside this
   repo*, which is what caught it. `withoutRepoEnv` now removes the entries. The same trap is in the
   tests (`t.Setenv(name, "")` followed by `os.Unsetenv`), so it is commented in both places.
2. **`ticketsPrefix` resolves symlinks on both sides.** `git rev-parse --show-toplevel` answers with
   the real path (`/private/tmp/…`) while `cfg.Root()` carries whatever the caller walked in on
   (`/tmp/…`) — on macOS the two never compare equal, so the guard would silently never fire in a
   temp dir, and could miss real violations wherever the config root is reached through a symlink.
   Not in the plan; found by probing git before writing the code.
3. **`hook.Refresh` returns `(Result, error)`, not `(bool, error)`** (plan, task 1). `Upgrade` needs
   the resolved path to label its `  + … (refreshed)` line, and that path is not
   `.git/hooks/pre-commit` when `core.hooksPath` redirects it. `Result` also gained a `Kind` field
   so `install.Uninstall` can branch on "foreign" without string-matching `Skipped`.
4. **The rejection message's remedy block computes its comment column.** `tickets/` is
   configurable, so a hardcoded column renders ragged for any other path length.
5. **`internal/install/testdata/markerblock.golden` regenerated** (`UPDATE_GOLDEN=1`). The plan named
   `MarkerBlock()` but not its golden test; the diff is exactly the new six-line bullet.
6. **Tests beyond the plan's list:** `TestShimExitCodes` runs the *real shim* against stub `pickle`
   binaries exiting 0/1/2/7 and absent-from-`PATH`, which is the only way to prove decision 3's
   contract end to end; `TestShimBlocksOnlyExitCodeOne` asserts on the shipped text and fails if
   anyone restores `|| exit 1`; `TestHooksAreAdvertised` guards the help text;
   `TestMarkerBlockStatesWhereCommitsLand` guards task 6d's prose. The B2 regression test was
   verified to bite by reintroducing the `rev-parse --abbrev-ref HEAD` bug (it fails, along with two
   `TestPreCommit` subtests whose fixtures are also unborn).
7. **Acceptance transcript, one mechanical fix on top of the re-refined version:** step 4's exit-code
   assertion cannot read `$?` after an `if` (that is the `if`'s own status, always 0) — it needs
   `set +e` around the commit. The runnable script lives at `/tmp/t057-acceptance.sh` during the
   session; its 12 steps all pass, including step 9 printing
   `pickle: bookkeeping guard skipped (hooks run exited 2)` and allowing the commit.
8. **F11 was folded into T-067 during the re-refinement**, not here (committed on `main` as
   `8adb56c`); F10 stayed `noted`, and the new `cli-reference.adoc` section is self-contained.

### Docs update

Task 7 in full (`cli-reference.adoc`, `project-structure.adoc`, `installation.adoc`,
`CHANGELOG.md`) plus the payload prose in task 6 (`tickets-README.md`, `review-protocol.md`,
`SKILL.md`, the `AGENTS.md` marker block) and `DESIGN.md` §7 in task 8. `just docs-check` is part
of the acceptance test.

### Finish

Summary of the shipped surface (`pickle hooks` verb, the shim, the written rule, the reviewer
fix), the acceptance transcript, and the suggested commit message:

```
feat(hooks): guard ticket bookkeeping with a pre-commit hook (T-057)
```

Then `pickle ticket move T-057 in-review --reason "acceptance green"` — committed **on `main`**,
not on the feature branch. Note for the human in the handback: hooks are per-clone, so after the
merge this repo's own guard is armed by a human running `pickle hooks install` from a clean
`main` build (it writes only to untracked `.git/hooks/`, so it is not a self-modify).

## Review

Reviewed 2026-08-06 on `feat/T-057-bookkeeping-pre-commit-hook` (`cc96393`), base `main`.
**Verdict: no blocking findings — DONE.**

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [ ] Docs-readability pass (step 4b) — **skipped: reviewer unavailable.** The `docs_readability`
      tool errored (`gemini-2.5-pro` via github-copilot: `model_not_supported`). Sanctioned
      conscious skip; the docs were read by hand instead.
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; History appended (step 6)
- [x] Board regenerated by the move; no other references needed updating (step 7)
- [x] Impact sweep done (step 8)
- [x] Summary + commit messages & MR attributes presented for approval; bookkeeping committed on
      `main` with explicit pathspecs (step 9)

### Implementation audit — all 8 tasks met

| task | verdict | evidence |
|---|---|---|
| 1 `internal/hook` | met | `hook.go` (486 lines): `ShimVersion`/marker, `Shim()`, `HooksDir`, `Install`, `Uninstall`, `Status`, `Refresh`, `PreCommit`; the `gitAt`/`gitHere` split makes decision 11 structural, and `os/exec` appears in exactly one file tree-wide (`rg -l os/exec` → `internal/hook/hook.go`) |
| 2 `hook_test.go` | met | 673 lines, package-level `TestMain` skip without git, `isolate()` pins `HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` and *unsets* the four `GIT_*` vars; named regressions for B2 (`TestPreCommitUnbornFeatureBranch`), F5/F7 (`TestPreCommitDuringAMerge`), B4 (`TestPreCommitDegradesWithoutGit`), F8 (`TestPreCommitInheritsTheHooksEnvironment`, `TestPreCommitInALinkedWorktree`) |
| 3 CLI verb | met | `internal/cli/hooks.go`; exit contract documented on `runHooksRun` and asserted by `TestHooksRunExitCodes` (1 / 0 / 0 / 0 / 0) + `TestHooksUsageErrors` (2) |
| 4 `install --hooks` | met | `internal/cli/install.go:97-107`, post-audit, failure is a warning; verified end-to-end by acceptance step 1 (no unit test — finding N7) |
| 5 lifecycle wiring | met | `Upgrade` → `hook.Refresh`, `Uninstall` → `hook.Uninstall` (dry-run honoured, foreign reported), `doctor.checkHooks`; `internal/install/hooks_test.go`, `internal/doctor/hooks_test.go`, `internal/cli/hooks_test.go` cover owned/stale/foreign/absent/no-git |
| 6a–6d payload prose | met | rules §0 *Where commits land* (+ the single-repo and mirror-image sub-bullets), `review-protocol.md` intro box + step 1, `SKILL.md` commit-policy bullet, `MarkerBlock()` + regenerated golden; **self-host mirror verified**: `pickle doctor` on the branch build reports `ok: AGENTS.md marker block current` |
| 7 docs | met | `cli-reference.adoc` `[#cmd-hooks]` (+ install/uninstall/doctor amendments, all three honesty sentences: F9 terminal-first, F7 `--amend`, B3 skew), `project-structure.adoc` sentence **replaced** (F6), `installation.adoc`, `CHANGELOG.md` |
| 8 `DESIGN.md` §7 | met | the stale "see T-057" bullet is gone; what shipped and the *declined* audit-side check (decision 2) are both recorded |

**Acceptance test re-run verbatim.** `just build && just test && just lint && just docs-check` all
green (`internal/hook` 6.8s, 12 packages ok). The 12-step transcript
(`/tmp/t057-acceptance.sh`, the implementation's runnable form) → `ALL 12 STEPS PASSED`, with step
9 printing `pickle: bookkeeping guard skipped (hooks run exited 2)` and allowing the commit.
All ten confirmed decisions plus 11–12 honoured; decisions 1 and 2 verified as *absences*
(no fourth guardrail rule, `internal/audit` still git-free).

### Findings

| # | severity | disposition | finding | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | **fixed inline** | `pickle hooks` was missing from the manual's Overview table, which opens *"Every command, its synopsis, and its contract"* — the branch added a command and left the index incomplete | `cli-reference.adoc:9-46` | row added after `pickle uninstall` (`a7e2ada`) |
| N2 | non-blocking | **fixed inline** | Two new cross-file references used `xref:cli-reference.adoc#cmd-hooks[…]`, which the single-document build renders as links to `cli-reference.pdf#cmd-hooks` / `cli-reference.epub#cmd-hooks` — **neither file exists**, so both shipped artifacts carried a dead link. Every other cross-file reference in the manual uses `<<anchor>>` (e.g. `installation.adoc:7` → `<<cmd-install>>`) | `dist/docs/pickle-user-manual.pdf` (2 `cli-reference.pdf` link targets), `EPUB/project-structure.xhtml` + `EPUB/installation.xhtml` `href="cli-reference.epub#cmd-hooks"` | both switched to `<<cmd-hooks>>`; re-rendered → 0 dead targets (`a7e2ada`) |
| N3 | non-blocking | **folded → T-067** | N2's whole class is invisible to the docs gate: `just docs-check` passed with both dead links, and T-067's proposed checker diffs `[#id]`/`[[id]]` against `<<id>>` only — an inter-document `xref:<file>.adoc#id[]` would still slip through | `justfile:23`, T-067 *Shape of the fix* | item added to T-067: in a single assembled book, **any** `xref:*.adoc#…` is a defect; flag the form, not just unresolved anchors |
| N4 | non-blocking | **fixed inline** | `writeRejection`'s comment justified its computed column with *"the `tickets/` path is configurable"* — it is not: `tickets` is a hardcoded literal in 14 places across `internal/`. The computation is still correct (the prefix is worktree-top-relative, so a child registered deeper renders `sub/dir/tickets/`), but the stated reason was wrong | `internal/hook/hook.go` (old comment) vs `rg '"tickets"' internal` | comment corrected to name the real reason (`a7e2ada`) |
| N5 | non-blocking | **fixed inline** | `pickle-guardrails.ts`'s header says the marker block's *"non-negotiable git rules … are encoded here"* and lists two. Task 6d gave the marker block a **third** git rule that decision 1 deliberately does not mirror there, so the branch made its own scaffold's header comment false | `agents/pi/extensions/pickle-guardrails.ts:1-19` vs `MarkerBlock()` | paragraph added stating the third rule lives in the hook, and why (guards every committer, reads `branch_prefix`) — no behaviour change, so T-050 is untouched (`a7e2ada`) |
| F9′ | non-blocking | **promoted → T-068** | *(added 2026-08-06, post-merge.)* The review's `noted` row on F9 (best-effort, terminal-first: a minimal `PATH` skips the guard) was **measured within minutes of arming the guard in this repo** — `PATH` held `/opt/homebrew/bin/pickle` 0.2.2, which predates the `hooks` verb, so the shim degraded to a no-op while `hooks status` and `doctor` both reported the guard as installed and current. Promoted per rules §5 (a later reviewer may promote a `noted` row by citing it) | live smoke test on a throwaway `feat/` branch: `pickle: bookkeeping guard skipped (hooks run exited 2)` and the commit went through | **T-068** — batches this with F9; the fail-open contract stays, the reporting is what must change |
| N6 | non-blocking | noted | The branch introduced 8 `**bold**` spans into a manual whose house style is single-asterisk `*bold*` (the only prior use was one `**and**` at `cli-reference.adoc:197`). Renders identically in AsciiDoc, so this is style only — and several of the new spans open on a backtick, where unconstrained bold is the safer form | `grep -c '\*\*'` per file | leave as is; if the manual ever gets a style guide, normalise there rather than in a feature branch |
| N7 | non-blocking | **folded → T-043** | `runInstall`'s `--hooks` block (task 4) has no automated test — neither the success path nor the "warning, install still succeeds" branch. It is covered only by the acceptance transcript, which is not run by `just test` or CI | `internal/cli/install.go:97-107`; no `--hooks` occurrence in `internal/cli/*_test.go` | item added to T-043 Part 2 (its theme is exactly `internal/cli` coverage gaps) |
| N8 | non-blocking | noted | The rule is now written in four places, but **not** in `SKILL.md`'s *implement* procedure step 8 — the exact moment three of the four recorded violations happened ("commit locally on the ticket branch … then `pickle ticket move … in-review`" never says the move is committed on the base branch). The *review* procedure got an explicit reminder; the implement one did not | `skill/SKILL.md` *Procedure: implement a ticket* step 8 vs `resources/review-protocol.md:38-45` | pre-existing omission, so not an inline fix (rules §5). Rules §0 + the marker block + the hook now cover it; if the violation recurs at handback, promote this row |
| N9 | non-blocking | noted | `README.md`'s "Try it" block (`pickle install` / `pickle version`) and `installation.adoc`'s equivalent block have diverged — only the manual gained `pickle hooks install` | `README.md:43-44` vs `installation.adoc:55-60` | defensible (the README is deliberately slim and the guard is optional); revisit if the two blocks drift further |
| N10 | non-blocking | **folded → T-066** | Whole-tree docs sweep, **pre-existing**: `cli-reference.adoc`'s doctor bullet writes `` `.pi/extensions/*.ts` `` and the `*` opens an unconstrained bold that swallows to the next `*`, so the PDF renders *"extensions/.ts … is a \*warning pointing at pickle upgrade"* — a mangled sentence in a shipped artifact | `pdftotext` line 1171 of the pre-fix render; source unchanged since `main` | item added to T-066 (it owns this file): pass the glob through (`+.pi/extensions/*.ts+`) or escape the asterisk |

**Disposition summary:** 10 non-blocking findings, 0 blocking — **4 fixed inline** (N1, N2, N4,
N5, all in `a7e2ada`), **3 folded** (N3 → T-067, N7 → T-043, N10 → T-066), **3 noted** (N6, N8,
N9). No new tickets at review time: every promotable finding landed in a ticket that already owned
its ground. **Amended post-merge:** F9's `noted` row was promoted to **T-068** once field evidence
turned it from a caveat into a measured no-op — the recoverability `noted` is supposed to provide,
used for real.

### Process note (not a finding against the work)

The ticket was found in `3-in-development/` although its History recorded the implementation as
complete and green: the plan's own *Finish* step — `pickle ticket move T-057 in-review` — was
never run. The review made that move first (`6c786c1`, on `main`). Worth stating because it is
the fifth handback-bookkeeping slip in the sequence this ticket exists to stop — and **nothing
mechanical catches this one**. The hook guards bookkeeping landing in the wrong *place*; a move
that is never made stages nothing, and `board audit` was clean throughout, because the ticket's
last transition (`READY → IN DEVELOPMENT`) agreed with the directory it sat in. The board simply
showed a finished feature as still in development until a human asked for the review.

## History

- 2026-07-28 — created (TO DO). source: chat — user request for a pre-commit hook rejecting
  `tickets/` paths on a `feat/` branch, after the T-054 review caught the violation (Q1) and
  it then recurred while that same review was being closed
- 2026-08-05 — patched by the T-022 review's impact sweep (finding F6, disposition `folded`): a fourth occurrence added to the violation table — the inverse case, where correct base-branch bookkeeping made the ticket's status invisible to a reviewer on the feature branch — plus a note that refinement must settle what the reviewer reads
- 2026-08-05 — refined: hook-only enforcement (no pi-guardrail rule, no `board audit` check — see
  the plan's Confirmed decisions 1–2); opt-in `pickle hooks` verb with presence-based ownership,
  no `pickle.toml` key; the split rule and the reviewer's base-branch read (T-022 F6) written into
  the payload. Re-graded impact medium-high → high, complexity medium → high, cost M → L: the
  ticket now spans a new git-touching package, a CLI verb, install/upgrade/uninstall/doctor wiring
  and the payload prose
- 2026-08-05 — TO DO → READY: plan complete
- 2026-08-05 — pickup applicability gate run (rules §8) before any move or branch: 4 blocking +
  8 non-blocking findings, recorded in the plan's *Pickup gate findings*. User routed the ticket
  **back to READY for re-refinement** instead of amending the plan inline at pickup, so no move
  and no `feat/` branch happened — the ticket stayed in `2-ready/` throughout and its plan is
  marked STALE. Design decisions 1–10 survived the audit; the defects are mechanical
  (`git rev-parse --abbrev-ref HEAD` misreads an unborn branch, the shim is fail-closed on version
  skew, no-git degradation unspecified, acceptance transcript not runnable verbatim)
- 2026-08-05 — re-refined (second pass) in `2-ready/`: applied all eight of the gate's `fixed
  inline` amendments — `symbolic-ref` for the branch name (B2), an explicit exit-code contract with
  the shim blocking only on `1` (B3), new decisions 11–12 for git cwd/env discipline (F8) and no-git
  degradation (B4), the dead rebase check dropped (F5), `project-structure.adoc:139-142` now
  *replaced* rather than extended (F6), and an acceptance transcript that runs verbatim — index
  unstaging after a rejection (B1), an unborn-`feat/` repo, an old-binary skew step, `/tmp`-anchored
  `mktemp`, and the marker-drift check moved before uninstall (F12). F11 folded into T-067 (CI half
  added there). Grades unchanged; status unchanged (READY throughout)
- 2026-08-06 — READY → IN DEVELOPMENT: picked up; gate re-verified, plan applies
- 2026-08-06 — implemented on `feat/T-057-bookkeeping-pre-commit-hook` (`cc96393`): `internal/hook`
  (the only package that shells out to git), the `pickle hooks` verb, `install --hooks`,
  upgrade/uninstall/doctor wiring, the rule written into the payload (rules §0, review protocol,
  `SKILL.md`, `MarkerBlock()` + the hand-mirrored `AGENTS.md`), the manual, `CHANGELOG.md` and
  `DESIGN.md` §7. `just build && just test && just lint && just docs-check` green; the 12-step
  acceptance transcript passes. Deviations recorded in the plan's *Implementation notes* — notably a
  blank-vs-unset `GIT_DIR` bug that made the guard inert until `pickle doctor`'s new line exposed it
- 2026-08-06 — IN DEVELOPMENT → IN REVIEW: acceptance green (handback move missed at implementation)
- 2026-08-06 — IN REVIEW → DONE: reviewed: no blocking findings; 10 non-blocking (4 fixed inline, 3 folded to T-043/T-066/T-067, 3 noted)
- 2026-08-06 — merged to main (PR #14, 9a9af59)
- 2026-08-06 — post-merge verification: guard armed in this repo (`pickle hooks install`, untracked
  `.git/hooks/pre-commit`) and smoke-tested on a throwaway `feat/` branch — it was **inert**,
  because the `pickle` on `PATH` (Homebrew 0.2.2) predates the `hooks` verb and the shim's fail-open
  waved the commit through. Correct behaviour, invisible reporting: **T-068** filed for it,
  promoting this review's `noted` finding F9
