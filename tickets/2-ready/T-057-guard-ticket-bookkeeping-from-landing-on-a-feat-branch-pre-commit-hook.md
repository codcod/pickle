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
3. **The hook is a shim; the rule lives in Go.** `.git/hooks/pre-commit` is a five-line `sh`
   script that execs `pickle hooks run pre-commit`. The rule therefore reads live `pickle.toml`
   and can never go stale, and it is unit-testable. A generated self-contained `sh` script with
   the prefixes baked in was rejected for going stale silently.
4. **The guard fails open, always.** If `pickle` is not on `PATH`, if no `pickle.toml` is found,
   or if the config fails to parse, the hook prints one line to stderr and **exits 0**. A missing
   or misconfigured guard must never brick `git commit`. The only non-zero exit is an actual
   violation.
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
   repo. Pass otherwise. Explicitly pass during a merge / rebase / cherry-pick / revert in
   progress. Only `tickets/` is guarded — never `pickle.toml`, never `skill/`.
8. **Escape hatch.** `git commit --no-verify`, named in the rejection message together with the
   legitimate case (a change whose *product* lives under `tickets/`).
9. **The rule gets written down.** The split it enforces is currently stated **nowhere** in the
   shipped payload; enforcing an unwritten rule is not acceptable, so tasks 6a–6d add it, and fix
   the inverse hazard the split creates for reviewers (T-022 finding F6) in the same pass.
10. **Not split.** The docs half is arguably schedulable alone, but a hook without the written
    rule enforces something undocumented, and the written rule without the hook is exactly
    today's state (four violations). One ticket.

### Tasks

**1. New package `internal/hook/hook.go`.** No dependency on `internal/install` (avoids a cycle;
`install` imports `hook`, not the reverse). Shells out to `git` via one unexported helper
`git(dir string, args ...string) (string, error)`.

- `const ShimVersion = 1`; `const marker = "# pickle:hook v1"`.
- `Shim() string` — the script text, ending in a trailing newline:
  ```sh
  #!/bin/sh
  # pickle:hook v1 — installed by `pickle hooks install`, removed by `pickle hooks uninstall`.
  # Refuses ticket bookkeeping (tickets/) staged on a feature branch. The rule lives in the
  # binary so it tracks pickle.toml. Bypass one commit with `git commit --no-verify`.
  if command -v pickle >/dev/null 2>&1; then
    pickle hooks run pre-commit || exit 1
  else
    echo "pickle: not on PATH — bookkeeping guard skipped" >&2
  fi
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
- `PreCommit(root string, cfg *config.Config, w io.Writer) (ok bool, err error)` — the rule of
  decision 7:
  1. `git rev-parse --abbrev-ref HEAD`; `HEAD` (detached) → ok.
  2. in-progress operation → ok: any of `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD` (via
     `git rev-parse --git-path <name>`) or the `rebase-merge` / `rebase-apply` directories exists.
  3. no configured `BranchPrefix` matches the branch name → ok.
  4. `git rev-parse --show-toplevel`; compute `tickets/`'s path **relative to that top level**
     from `cfg.Root()`. If it escapes the repo (`..`), → ok (multi-repo child: nothing to guard).
  5. `git diff --cached --name-only -z`; ok unless a staged path is under that prefix.
  6. On violation write the rejection message to `w`: the branch, the offending staged paths
     (cap the list at 10 plus an `… and N more` line), *why* (a squash-merge of this branch eats
     the bookkeeping; the board then disagrees with the tickets), the remedy
     (`git restore --staged tickets/`, commit the code, commit the bookkeeping on the base
     branch), and the `--no-verify` escape hatch with its legitimate case.

**2. `internal/hook/hook_test.go`.** `t.Skip` the whole file when `exec.LookPath("git")` fails.
Helper builds a real temp repo (`git init -b main`, `user.email`/`user.name` set locally, a
`pickle.toml` + `tickets/` tree via the existing `config` API). Cases:

- install → file exists, mode `0o755`, contains the marker; re-install is idempotent;
- a foreign `pre-commit` is refused, survives byte-identical, and `--force` overwrites it;
- `core.hooksPath` set → the shim lands in that directory;
- uninstall removes an owned hook, skips a foreign one, and is idempotent; `dryRun` mutates
  nothing;
- `Status` reports absent / owned / foreign / owned-stale (write a `v0` marker), and `Refresh`
  fixes only the stale-owned case;
- `PreCommit`: `main` + staged `tickets/` → ok; `feat/T-1-x` + staged `tickets/` → violation,
  message names the branch and the path; `feat/T-1-x` + staged code only → ok; mixed staging →
  violation; detached HEAD → ok; mid-rebase → ok; a child configured with
  `branch_prefix = "wip/"` → `wip/…` rejected and `feat/…` allowed; `tickets/` outside the repo
  → ok.

**3. CLI verb — new `internal/cli/hooks.go`, wired in `internal/cli/cli.go`.**
`pickle hooks install [--force] | uninstall [--dry-run|-n] | status | run pre-commit`. One verb
(`hooks`), no separate `hook` verb. `run pre-commit` is the shim's entry point: it locates
`pickle.toml` with `config.Find`/`Load` and, per decision 4, **exits 0 on any error other than a
violation** (a violation is `exitError`). An unknown subcommand is `exitUsage`. Add the `hooks`
block to `usage()`'s *Setup commands* group.

**4. `pickle install --hooks`** in `internal/cli/install.go` (`runInstall`): after the post-install
audit passes, call `hook.Install(root, false)` and print the created path in the same `  + %s`
style. A failure here prints a warning and does **not** fail the install.

**5. Lifecycle wiring.**
- `internal/install/install.go` → `Uninstall`: remove the pickle-owned hook (honouring
  `UninstallOptions.DryRun`, reported through `res.removed`/`res.skipped` like the pi scaffolds).
- `internal/install/install.go` → `Upgrade`: `hook.Refresh(root)` — refresh an owned, stale shim;
  never install one that is absent.
- `internal/doctor/doctor.go`: new `checkHooks(root, r)` called from `Check` — owned+current →
  `ok`; owned+stale → warning pointing at `pickle upgrade` (mirrors the agent-scaffold check);
  absent → `ok` line noting it is optional and naming `pickle hooks install`; foreign → `ok` line
  saying the hook is not pickle's and is left alone. Never an error: the hook is opt-in.
- Extend `internal/install/install_test.go`, `internal/doctor/doctor_test.go` and
  `internal/cli/cli_test.go` for the uninstall/upgrade/doctor/CLI paths above.

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
- `docs/user-manual/cli-reference.adoc`: new `[#cmd-hooks] == pickle hooks` section after
  `== pickle uninstall`, documenting all four subcommands, the shim, `core.hooksPath` handling,
  fail-open behaviour, `--no-verify`, and that hooks are per-clone (never cloned). Amend the
  `pickle install` section (`--hooks`), the `pickle uninstall` section (removes the owned hook),
  and the `pickle doctor` bullet list (reports hook state).
- `docs/user-manual/concepts/project-structure.adoc`: in the single-repo bullet list ("One history
  carries both roles" …), state the split and point at `pickle hooks install`.
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

```sh
just build
D=$(mktemp -d); mkdir -p "$D/bin"; cp pickle "$D/bin/pickle"; export PATH="$D/bin:$PATH"
mkdir -p "$D/repo" && cd "$D/repo" && git init -q -b main .
git config user.email t@example.com && git config user.name test
pickle install --project demo --hooks
test -x .git/hooks/pre-commit && grep -q 'pickle:hook' .git/hooks/pre-commit   # 1. installed

git add pickle.toml tickets AGENTS.md && git commit -qm 'chore: scaffold'      # 2. base: allowed
git checkout -qb feat/T-001-demo
pickle ticket new "demo" --project demo >/dev/null
git add tickets && git commit -m 'docs(tickets): file T-001'                   # 3. MUST FAIL (1)
echo x > code.txt && git add code.txt && git commit -qm 'feat: code (T-001)'   # 4. code: allowed
git add tickets && git commit -q --no-verify -m 'docs(tickets): file T-001'    # 5. escape hatch
git checkout -q --detach && git commit -q --allow-empty -m 'detached'          # 6. detached: ok
git checkout -q feat/T-001-demo

pickle hooks status          # owned, current, path shown
pickle doctor -v | grep -i hook
pickle hooks uninstall && pickle hooks status                                  # 7. absent
printf '#!/bin/sh\nexit 0\n' > .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit
pickle hooks install         # MUST refuse (foreign hook), file unchanged
pickle hooks install --force # overwrites
pickle uninstall -n | grep pre-commit && pickle uninstall                      # 8. removed
```

Expected: steps 1, 2, 4, 5, 6, 7 succeed; **step 3 exits non-zero** with the rejection message
naming the branch, the staged `tickets/` path and `--no-verify`, and leaves `git log` unchanged;
the foreign hook is refused without modification and only `--force` replaces it; `pickle uninstall`
removes the owned hook and `-n` lists it. Re-runnable verbatim from a clean `mktemp -d`.

Also verify the payload edits round-trip: `pickle doctor` in that throwaway install reports no
marker drift (proving task 6d's `MarkerBlock()` change and the hand-mirrored `AGENTS.md` agree).

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

<!-- empty until IN REVIEW -->

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
