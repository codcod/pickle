---
id: T-082
title: pre-push hook: refuse a feature-branch push whose range carries tickets/ paths
project: pickle
depends-on: []
spawned-by: [T-072]
impact: medium
complexity: medium
cost: M
---

# T-082 — pre-push hook: refuse a feature-branch push whose range carries tickets/ paths

## Outcome

After this ships, a `git push` whose range still carries a `tickets/` path on a feature branch is refused before it reaches the remote, closing the one gap the existing pre-commit hook and the origin-base prose check both leave open at publish time.

## Description

Rules §0 splits every change in two — code on the child's feature branch, ticket and board
bookkeeping on the base branch — because a squash-merge of a branch carrying bookkeeping folds or
drops it and leaves `BOARD.md` disagreeing with the tickets it indexes. (T-084 note: for a
root-path child the default merge is now rebase/keep-history, under which the same bookkeeping is
*preserved* rather than folded — landing on the base as out-of-order commits instead of
vanishing. The hazard and this ticket's fix are unchanged; only the folding half of the rationale
above is specific to squash.) T-057 shipped a
`pre-commit` hook enforcing it at **commit** time. T-072 established that the same failure also
arrives at **publish** time, where that hook is structurally blind, and closed it **in prose**
(§0, review-protocol step 9, and `TEMPLATE.md`).

This ticket asks whether that prose should also be mechanical. The hazard has now appeared **four
times** — T-053/T-054 (bookkeeping committed on the branch), T-022 (branch cut before bookkeeping
landed → stale ticket), T-068 (caught during publish, pre-push), and T-073 (**not** caught: it
reached `origin/main` in squash `7b33876`, PR #18). Prose is followed by whoever reads it; the
fourth occurrence happened to an operator who had read it.

### The correction this ticket rests on

T-072's Description originally dismissed this shape: *"the failure here was the absence of a
push, not a bad one, so `pre-push` on the feature branch would not have caught it."* **That
reasoning is wrong**, and T-072 records the correction. The guard does not need to observe the
missing *base* push. It fires on the **feature-branch push** — which does happen, in every one of
these incidents — and measures the branch against `origin/<base>`. Verified against T-073's real
SHAs (`origin/main` = `152fea8`, branch head = `850ea3c`): `git diff --name-only 152fea8...850ea3c`
named **7 `tickets/` paths**, so the guard would have refused the push and printed the one-line
repair. It would equally have caught T-068's.

### Shape (decided at refinement — the plan's Confirmed design decisions are binding)

1. **The check.** Identical in substance to the one T-072 put in §0: on a push whose local ref is
   a **feature branch**, refuse if the range names any `tickets/` path. `pre-push` receives
   `<local ref> <local sha1> <remote ref> <remote sha1>` on **stdin**, one line per ref, and the
   deleted-branch case (`local sha1` all-zero) must be skipped.

   **The range is *not* the stdin one.** This ticket was filed assuming `<remote-sha>...<local-sha>`
   could be read straight off stdin; refinement found that wrong in both directions. On the
   **first** push of a new branch the remote sha1 is all-zero — there is no remote-side range at
   all. On a **second** push the stdin range is `last-pushed…local`, which is not what the forge
   will diff and would wave through `tickets/` paths that rode in on the first push. The
   invariant §0 actually states is that **the MR carries no `tickets/` path**, so the guard
   measures `<remote>/<base>...<local-sha>` — the same three-dot, merge-base form T-072's prose
   check uses and the forge uses. Stdin still supplies the local sha and the deletion skip; the
   push's remote name arrives as the hook's `$1`.

   **The base has to be discovered**, because `pickle.toml` has no base-branch key — `config.Project`
   carries `branch_prefix` but nothing naming the base (`internal/config/config.go:104-116`).
   Resolution order and the fail-open fallback are decision 4 in the plan.
2. **It must not fire when the ref being pushed is the base branch.** Bookkeeping on the base is
   the *correct* destination — every `git push origin main` in this repo carries `tickets/`
   paths by design. This is the single most important way to get the guard wrong.
3. **Fail-open, exactly as `pre-commit` does.** `hook.Shim()`'s contract (T-057 decision 3) is
   that **only exit 1 means violation**; any other non-zero is reported on stderr and waved
   through, because an older `pickle` first on `PATH` exits 2 on an unknown verb and must not
   block every push in the repository. The `pre-push` shim inherits this verbatim — it must never
   grow an `exit 1` on the guard-absent branch.
4. **The structural cost, which is the real reason this is not S.** `internal/hook` is currently
   written around *exactly one* hook: `const HookName = "pre-commit"` (`hook.go:48`), and
   `Status()`, `Install()`, `Uninstall()` and the `doctor` probe each assume that single name and
   path. Supporting a second hook means generalizing that to a set — install/uninstall/status
   over N hooks, N marker lines, N staleness checks — and `pickle upgrade` must refresh both.
   The check itself is a dozen lines; this generalization is the bulk of the work.
5. **`ShimVersion` bump and ownership.** Adding a hook is a shim-text change (→ v3), which
   `pickle upgrade` already refreshes on. `.git/hooks/pre-push` may already exist and be the
   user's own: the same marker-prefix ownership rule as `pre-commit` applies (`# pickle:hook v`),
   and a foreign hook must never be clobbered.
6. **`doctor` and the docs.** `doctor` should report the new hook the way it reports `pre-commit`
   (absent / stale / foreign). `hooks install`'s output, `cli-reference.adoc`, and the
   `hooks`-related manual prose all describe "a `pre-commit` hook" in the singular and would need
   re-wording.

### Questions closed at refinement

- **`pre-push` hook, not a `pickle publish-check` subcommand.** Opt-in is precisely the property
  that failed four times, and the discoverability argument for a subcommand is already satisfied:
  the rule is exposed as `pickle hooks run pre-push`, a plain verb runnable by hand and unit-
  testable without git plumbing. A separate `publish-check` would be a second name for that call.
- **Alongside T-072's prose, not superseding it.** Hooks live in `.git/`, are never cloned, and
  `--no-verify` bypasses them; the prose remains the rule and the hook is its local enforcement.
  §0 and review-protocol step 9 are re-worded to say which is which, so the rule has one owner
  and one enforcer rather than two half-owners.
- **T-091's add-without-delete check does not come along for the ride.** T-091 note-and-closed it
  "to T-082's staged-path plumbing… it runs on the same staged-path data this ticket's guard
  already reads". **That premise is wrong**: this guard reads a *commit range*
  (`git diff <base>...<head>`), never the index. The only genuinely shared code is
  `ticketsPrefix()` plus the path-prefix match, which already ships and needs no new plumbing.
  Folding it in would put a `pre-commit` behaviour change inside a `pre-push` ticket. It stays
  note-and-closed on T-091's record, now with a reason rather than a hand-off.
- **Grading confirmed at medium / medium / M.** Item 4's structural cost is real: six functions
  parameterized plus a doc sweep across six files and the payload prose. Not an S.

### Soft couplings

- **T-072** — lineage (`spawned-by`); shipped the prose this would mechanize, and carries the
  measured evidence plus the corrected reasoning above. Not a dependency: this ticket is coherent
  whether or not T-072's prose is ever changed again.
- **T-057** — shipped `internal/hook`, the shim contract, and the one-hook assumption in item 4.
  Its decision 2 (keep `board audit` git-free) is why a publish check is not bolted onto the
  audit, and its decision 3 is the fail-open rule in item 3.
- **T-068** — shipped the inert-guard probe; whatever this adds must be probed the same way, or
  it inherits the failure mode where a dead guard is indistinguishable from a satisfied one.
- **T-071** — hardens that probe; overlaps this ticket's item 6 if both touch `doctor`'s hook
  reporting.
- **T-066** — CLI-surface documentation gaps; item 6's `cli-reference.adoc` re-wording lands in
  the same tree.
- **T-091** — same family (git hygiene around bookkeeping), different failure: T-091 fixed an
  *incomplete* bookkeeping commit on the right branch by making `ticket move`/`ticket new` print
  the full stage set, while this ticket catches bookkeeping on the **wrong** branch. T-091
  **note-and-closed a second line of defence to this ticket**: a `pre-commit` check that a staged
  ticket add whose id also exists at another status path in `HEAD` carries a matching staged
  delete. It runs on the same staged-path data this ticket's guard already reads, so decide at
  refinement whether it comes along for the ride — building it separately would duplicate the
  plumbing. Not a dependency in either direction.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                                   # `pickle` is the root-path child
git checkout main
git checkout -b feat/T-082-pre-push-bookkeeping-guard
```

WIP commits encouraged. **This repo is a root-path child** (`path = "."`), so the Finish step
tidies WIP into atomic commits and keeps that history rather than squashing. Ticket and board
bookkeeping stays on `main` — never on this branch (the very rule this ticket mechanizes).

### Prerequisite gate (hard)

None. `depends-on:` is empty and every soft coupling is already shipped: T-057 (`internal/hook`,
the shim contract), T-068 (`Probe`), T-072 (the §0 prose), T-091 (the stage-set print). T-071
is still in `1-to-do/` and reworks `probeCapable`; decision 6 keeps this ticket out of that
function so the two do not collide in either order.

### Confirmed design decisions (do not deviate without asking)

1. **A `pre-push` hook, not a new subcommand.** The rule ships as `pickle hooks run pre-push`,
   invoked by an installed shim. Do not add a `pickle publish-check` verb.
2. **The fail-open contract is inherited verbatim** (T-057 decision 3). Only **exit 1** means
   violation. The guard-absent branch of the shim must never grow an `exit 1`, and every
   undecidable state — no repo, no config, unresolvable base, a `git diff` that errors — returns
   ok and exits 0. An older `pickle` first on `PATH` exits 2 on the unknown verb and must not
   block every push in the repository.
3. **The guard fires only on a ref that is a feature branch of a registered child.** Reuse
   `onFeatureBranch` (`internal/hook/hook.go`) unchanged. Pushing the **base** branch carries
   `tickets/` paths by design and must never be refused — this is the single most important way
   to get the guard wrong, and it needs a dedicated regression test.
4. **The range is `<remote>/<base>...<local-sha>` (three dots), not the stdin range.** Resolve
   `<base>` in this order, stopping at the first that exists:
   `git symbolic-ref --quiet --short refs/remotes/<remote>/HEAD` → `refs/remotes/<remote>/main` →
   `refs/remotes/<remote>/master` (each verified with `git rev-parse --verify --quiet`). If none
   resolves, **fail open** with one stderr line naming what was tried. Two dots is a different
   question and will mislead (rules §0).
5. **The hook performs no network I/O.** No `git fetch`. A stale remote-tracking ref can only
   widen the range, so the failure direction is a false *refusal*, never a false pass — the
   fail-safe direction. The rejection message must therefore name the stale-ref case and suggest
   `git fetch <remote> <base>` before `--no-verify`.
6. **`Probe()` is not generalized.** It keeps probing with `pre-commit`
   (`internal/hook/probe.go`), because it measures whether the binary on `PATH` can dispatch the
   `hooks run` verb at all, and both hooks ship in the same binary. Do not add a second probe and
   do not touch `probeCapable` — that function is T-071's scope.
7. **One shared `ShimVersion`, bumped to 3.** Both shims carry the same marker version; a bump
   refreshes both. Do not introduce per-hook versions.
8. **Marker-prefix ownership is unchanged** (`# pickle:hook v`). An existing foreign
   `.git/hooks/pre-push` is refused, never clobbered, exactly as `pre-commit` is — including the
   `--force` escape and the "chain it from your own hook" advice.
9. **T-091's add-without-delete `pre-commit` check is out of scope** (see the Description). Do
   not implement it here.
10. **Self-modify policy** (`AGENTS.md`): never run `pickle install|upgrade|uninstall` against
    this repo from this branch. The marker-block change in Task 6 is made **by hand**, mirroring
    `markerBlock()`. Any test install goes to a throwaway dir with the binary copied in.

### Tasks

#### Task 1 — generalize `internal/hook` from one hook to a set

`internal/hook/hook.go`:

- Replace `const HookName = "pre-commit"` (`hook.go:48`) with a named type and the set:
  ```go
  type Name string
  const (
      PreCommit Name = "pre-commit"
      PrePush   Name = "pre-push"
  )
  func Names() []Name { return []Name{PreCommit, PrePush} }
  ```
- `ShimVersion` → `3`, with a doc-comment note recording *why* (a second hook is a shim-text
  change), in the same style as the existing v2 note.
- `Shim()` → `Shim(name Name) string`. Factor the shared parts — shebang, marker line, the
  `command -v pickle` guard-absent branch, the `rc` exit-code handling — and vary only the
  one-line description comment and the invocation: `pickle hooks run pre-commit` versus
  `pickle hooks run pre-push "$@"` (pre-push forwards argv and inherits stdin).
- Parameterize by name: `Status(root string, name Name)`, `Install(root string, name Name, force bool)`,
  `Uninstall(root string, name Name, dryRun bool)`, `Refresh(root string, name Name)`. Add
  `Name` to both `State` and `Result` so callers can report per hook without threading it
  separately. Update `Install`'s foreign-hook error text, which currently interpolates `HookName`.
- Add the plural wrappers every caller will use, so no caller grows its own loop:
  `StatusAll(root) ([]State, error)`, `InstallAll(root string, force bool) ([]Result, error)`,
  `UninstallAll(root string, dryRun bool) ([]Result, error)`, `RefreshAll(root) ([]Result, error)`.
  Each iterates `Names()` in order and is the only place that ordering is decided.
- `KindNoRepo` is a whole-repository property: `StatusAll` must not report it once per hook. Emit
  it once and short-circuit.

> **Plan amended inline (implementation):** the constant `PreCommit Name = "pre-commit"` this
> task specifies collides with the package's pre-existing `func PreCommit(cfg *config.Config, w
> io.Writer) (bool, error)` (the commit-time rule itself) — Go does not allow a const and a func
> to share one top-level identifier — and Task 2's own proposed `func PrePush(...)` would collide
> with the sibling `PrePush Name = "pre-push"` constant the same way. Resolved by renaming the two
> rule functions to `CheckPreCommit` and `CheckPrePush`, keeping the `Name` constants exactly as
> specified (they are what CLI dispatch and the shim text read). Every reference below to
> `PreCommit(cfg, w)` / `PrePush(cfg, remote, refs, w)` as a *function* means `CheckPreCommit` /
> `CheckPrePush`; `PreCommit` / `PrePush` alone still mean the `Name` constants.

#### Task 2 — the pre-push rule

New `internal/hook/prepush.go`:

```go
type PushRef struct{ LocalRef, LocalSHA, RemoteRef, RemoteSHA string }
func ParsePushRefs(r io.Reader) ([]PushRef, error)
func PrePush(cfg *config.Config, remote string, refs []PushRef, w io.Writer) (bool, error)
```

`ParsePushRefs` is pure and takes an `io.Reader`, so the stdin format is testable with a
`strings.Reader` and no git at all. Malformed lines are skipped, not fatal (decision 2).

`PrePush`, per ref, in this order:

1. Skip when `LocalSHA` is all-zero (branch deletion).
2. Skip when `LocalRef` is not under `refs/heads/`; otherwise the branch is the remainder.
3. Skip unless `onFeatureBranch(cfg, branch)` (decision 3).
4. `ticketsPrefix(cfg)` — skip when `tickets/` lies outside this repository (the multi-repo case,
   same as `PreCommit`).
5. Resolve the base per decision 4; on failure write one stderr line and return ok.
6. `git diff --name-only -z <base>...<LocalSHA>`; on error return ok (decision 2).
7. Collect offenders with the existing prefix match; if any, write the rejection and return
   `ok=false`.

Use **`gitHere`**, never `gitAt` — same reasoning as `PreCommit` and T-057 decision 11: git runs
`pre-push` from the top of the worktree with `GIT_*` set, and `git -C <root>` could inspect a
different repository. Add a short package-level comment saying so, because this is now the second
rule with that constraint and the next reader will not re-derive it.

`writePushRejection` follows `writeRejection`'s shape — what, why, and the ways out — reusing
`maxListedPaths`. It must name three remedies: push the base first, `git fetch <remote> <base>`
if the remote-tracking ref is stale (decision 5), and `git push --no-verify` when the branch's
own product genuinely lives under `tickets/`.

#### Task 3 — `pickle hooks run pre-push`

`internal/cli/hooks.go`, `runHooksRun` (`:169-215`): dispatch on the hook name instead of
rejecting everything but one.

- `pre-commit` — unchanged, still rejects extra arguments.
- `pre-push` — accepts up to two positional arguments (`<remote-name> [<remote-url>]`, git's own
  `$1`/`$2`) and reads stdin. A missing remote name defaults to `origin`.
- Unknown name: usage error listing both, exit 2.
- The exit-code contract in the doc comment is unchanged and must be restated as covering both
  hooks, not silently left describing one.

Also in that file: `runHooksInstall`, `runHooksUninstall` and `runHooksStatus` switch to the
`*All` wrappers and print one line per hook; the `--force` flag description and the
"guard armed" summary become plural; `runHooks`'s dispatch error text is unchanged.

`internal/cli/install.go`: the `--hooks` flag description (`:29`) and the post-install block
(`:114-125`) go plural via `InstallAll`.

#### Task 4 — `doctor`

`internal/doctor/doctor.go:250-295`: iterate `hook.StatusAll` and report absent / foreign / stale
/ owned per hook, keeping today's severities (absent is `ok`, stale is a warning). Call
`hook.Probe()` **once**, after the loop, and only when at least one hook is `KindOwned` — the
probe answers a per-binary question, not a per-hook one (decision 6).

#### Task 5 — `upgrade` and `uninstall`

`internal/install/install.go`: `hook.Refresh` at `:354` → `RefreshAll`, `hook.Uninstall` at
`:475` → `UninstallAll`, each reporting one line per hook through the existing `hookLabel`.
Verify the dry-run path still reports every hook it would remove.

#### Task 6 — tests

- `internal/hook/hook_test.go` — table-drive the existing install/uninstall/status/refresh cases
  over `Names()`; add foreign-hook refusal and marker-version parsing for `pre-push`.
- New `internal/hook/prepush_test.go` — `ParsePushRefs` against literal stdin fixtures (new
  branch with an all-zero remote sha, a deletion with an all-zero local sha, several refs in one
  push, a malformed line). `PrePush` against real temp-dir repositories: a feature branch
  carrying `tickets/` paths (refused), a feature branch carrying none (allowed), **the base
  branch carrying `tickets/` paths (allowed — decision 3)**, an unresolvable base (allowed, one
  stderr line), and `tickets/` outside the repository (allowed).
- `internal/cli/hooks_test.go` — `hooks run pre-push` exit codes: 1 only on violation, 0 on every
  degraded path, 2 on a bad invocation.
- `internal/doctor/hooks_test.go` — both hooks reported; the probe warning appears once, not
  twice.
- `internal/install/hooks_test.go` — `RefreshAll` bumps a v2 shim to v3 for both hooks and leaves
  a foreign one alone.

#### Task 7 — docs and payload prose

Every site below currently says "a `pre-commit` hook" in the singular:

- `docs/user-manual/cli-reference.adoc` — the command-table row (`:30-31`), `install --hooks`
  (`:79`), the `doctor` bullet (`:271-274`), the `uninstall` note (`:318`), the `cmd-hooks`
  synopsis and body (`:334-364`), the per-clone note (`:404`) and the exit-code/inertness block
  (`:424-431`). Add the pre-push rule, decision 4's base resolution, and decision 5's stale-ref
  caveat.
- `docs/user-manual/installation.adoc:58-73`
- `docs/user-manual/concepts/project-structure.adoc:159-161`
- `skill/resources/tickets-README.md` — the §0 hook bullet (`:94-97`) and the publish-time bullet
  (`:98-108`), which must say the hook now enforces locally what the prose rules, **alongside**
  it, not instead of it (decision from the Description).
- `skill/resources/review-protocol.md` — step 9 and its checklist line (`:238`): the manual
  `origin/<base>...HEAD` check stays; note that an installed `pre-push` hook performs it.
- `internal/install/install.go:940` — the rendered marker-block text.
- **This repo's own `AGENTS.md` marker block, by hand**, mirroring the change above (decision 10).
- `CHANGELOG.md` — an Unreleased entry.
- `DESIGN.md:226-244` — the hooks paragraph, which currently describes exactly one hook.

### Acceptance test

```sh
just build && just test && just lint && just docs-check
```

Then the behaviour, in a **throwaway directory with the binary copied in** (never the in-repo
path — `AGENTS.md` self-modify policy):

```sh
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"
export PATH="$D:$PATH"

# a bare remote plus a clone, so origin/main is a real remote-tracking ref
git init -q --bare remote.git
git clone -q remote.git work && cd work
pickle-test install --project pickle . >/dev/null
git add -- $(git status --porcelain | sed 's/^...//') && git commit -qm init && git push -q origin main
pickle-test hooks install

# 1. feature branch carrying bookkeeping — REFUSED
git checkout -qb feat/T-999-demo
mkdir -p tickets/1-to-do && echo x > tickets/1-to-do/T-999-demo.md
git add tickets/1-to-do/T-999-demo.md && git commit -q --no-verify -m 'board: T-999 demo'
git push origin feat/T-999-demo    # expect: non-zero, rejection naming tickets/1-to-do/T-999-demo.md

# 2. the base branch carrying bookkeeping — ALLOWED (decision 3)
git checkout -q main && echo y > tickets/1-to-do/T-998-demo.md
git add tickets/1-to-do/T-998-demo.md && git commit -qm 'board: T-998 demo'
git push origin main               # expect: exit 0, no output from the guard

# 3. feature branch carrying only code — ALLOWED
git checkout -qb feat/T-997-code && echo 'package p' > p.go
git add p.go && git commit -qm 'feat: code (T-997)'
git push origin feat/T-997-code    # expect: exit 0

# 4. the bypass still works
git checkout -q feat/T-999-demo
git push --no-verify origin feat/T-999-demo   # expect: exit 0

# 5. both hooks are reported
pickle-test hooks status           # expect: one line each for pre-commit and pre-push, both v3
pickle-test doctor -v              # expect: both hooks reported, probe warning at most once
```

(Note: the binary must resolve as literally `pickle` on `PATH` for the installed shim's own
`pickle hooks run <name>` call to find it — a symlink `"$D/pickle" -> pickle-test` placed
alongside the throwaway copy satisfies this without ever referencing the in-repo path.)

Expected results are the inline comments. Case 1 is the ticket (T-073's real failure), case 2 is
the way to get the guard wrong, and case 4 proves the guard stayed bypassable.

### Docs update (mandatory when user-facing)

User-facing. Task 7 is the docs step in full: `cli-reference.adoc`, `installation.adoc`,
`concepts/project-structure.adoc`, the two payload files (`skill/resources/tickets-README.md`,
`skill/resources/review-protocol.md`), the marker block in `internal/install/install.go` **and**
this repo's `AGENTS.md` by hand, `CHANGELOG.md`, and `DESIGN.md`. `just docs-check` must pass.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated per Task 7, including the hand-edited `AGENTS.md` marker block.
3. Write a summary: files touched, the `ShimVersion` 2→3 bump, and anything deferred.
4. Suggested Conventional Commit message:

   ```
   feat(hooks): refuse a feature-branch push carrying tickets/ paths (T-082)

   Generalize internal/hook from a single pre-commit hook to a set, and add a
   pre-push guard measuring <remote>/<base>...<local> — the same three-dot range
   a forge diffs — so bookkeeping cannot reach an MR. Fail-open contract and
   marker-prefix ownership are inherited unchanged; ShimVersion 2 -> 3.
   ```

5. **Tidy up before presenting** — this is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed commits (suggested split: the
   `internal/hook` generalization, the pre-push rule + tests, the CLI/doctor/install wiring, the
   docs sweep) and keep that history rather than squashing.
6. Commit locally on the branch. Do **not** push or open an MR without explicit approval. Present
   the commit messages; after approval, verify
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing, then push and open the MR. Merging is the human's.

## Review

### 2026-08-14 — first review (verdict: REWORK)

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — run; only four of
      its ~20 suggestions touch prose this branch authored, and they are offered to the user
      separately (they are polish, not findings)
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages presented for approval (step 9 — deferred to the re-review,
      since publishing a branch with blocking findings is not on the table)

**Acceptance test: green, verbatim.** `just build`, `just test`, `just lint`, `just docs-check`
all clean. The five throwaway-dir cases behave exactly as the plan predicts — case 1 refused with
the three-remedy message, case 2 (`git push origin main` carrying `tickets/`) allowed, case 3
allowed, case 4 (`--no-verify`) allowed, case 5 reporting both hooks and probing at most once
under `doctor -v`. Two extra cases were run beyond the plan's: a **second** push of a branch
whose `tickets/` path rode in on the *first* push is still refused (the whole point of not using
the stdin range — verified), and a tag push and a branch deletion both pass through.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | docs-gap | — | Task 7's last two doc sites were not done: the **rendered marker block** and **this repo's own `AGENTS.md` marker block** both still describe the guard as commit-time only | `internal/install/install.go:1057` and `AGENTS.md:93` are byte-identical to `main`; `git diff --stat main...HEAD` lists no `AGENTS.md` | Name the `pre-push` hook in `markerBlock()`'s "Where commits land" bullet and widen its bypass clause beyond `git commit --no-verify`; mirror it by hand into `AGENTS.md` (decision 10, Finish item 2) |
| F2 | blocking | correctness | — | `git push <remote> HEAD:refs/heads/feat/T-NNN-x` bypasses the guard **silently**: stdin's local ref is the literal `HEAD`, so the `refs/heads/` `CutPrefix` fails and the ref is skipped before any range is measured | verified with a stdin-dumping `pre-push`: `HEAD <sha> refs/heads/feat/T-995-head 0000…`; the same push carrying `tickets/1-to-do/T-999-demo.md` exited 0 and reached the remote | In `CheckPrePush` step 2, fall back to `RemoteRef` when `LocalRef` is not under `refs/heads/` — git always sends a fully-qualified remote ref, so a tag push (`refs/tags/…` on both sides) is still skipped and `HEAD:refs/heads/main` still resolves to the base and is still allowed. Add the regression test |
| F3 | blocking | correctness | — | `hooks status` probes `PATH` **once per owned hook**, contradicting the doc this branch wrote ("checked once, not once per hook") and printing the capability line twice | `cli-reference.adoc:433-435` vs. observed output: two identical `PATH: … can run the guard` lines, two probe execs | Hoist `hook.Probe()` out of `printHookStatus` and call it once in `runHooksStatus`, as `checkHooks` already does (decision 6) — keep the doc as written |
| F4 | blocking | correctness | — | `InstallAll`'s partial-failure path misreports in both callers: `hooks install` prints `  = <path> ()` with an empty parenthetical for the refused foreign hook, and `install --hooks` skips its whole result loop on any error, so a `pre-commit` hook it *did* write is never reported and `warnIfInert` never runs | `hooks install` with a foreign `pre-push`: `  = …/pre-push ()`; `install --hooks` in the same tree prints no `+ …/pre-commit` line although the file is on disk | Set `Skipped` on `Install`'s foreign-refusal `Result` (e.g. `"not pickle's — left in place"`, matching `Uninstall`), and in `install.go` print the results before branching on the error |
| F5 | non-blocking | stale-xref | fixed inline | the doc quoted the degraded stderr line as `pickle: bookkeeping guard skipped (…)`, but this branch changed the shim's own text to `pickle: <name> guard skipped (…)` | `cli-reference.adoc:499` vs `hook.go` `Shim()` | fixed in `b05f8bd`: quote the shared shape `pickle: … guard skipped (…)` |
| F6 | non-blocking | design | noted | one degraded state now has two names: the shims say `pickle: <name> guard skipped`, the binary's own five call sites still say `pickle: bookkeeping guard skipped` | `hook.go` `Shim()` vs `internal/cli/hooks.go:236,244,267,275,280` and `prepush.go:121` | unify on one form (probably the hook-named one) if the pair ever confuses a bug report |
| F7 | non-blocking | design | folded | the offender collection — `strings.Split(out, "\x00")` + the `p == TrimSuffix(prefix,"/") \|\| HasPrefix(p, prefix)` match — is now duplicated verbatim in `CheckPreCommit` and `CheckPrePush`; the plan predicted this shared code would need "no new plumbing", which is true, but it is now written twice | `hook.go:505-515` vs `prepush.go:129-139` | folded into **T-042** (collapse duplicated internal predicates) as a fourth site |
| F8 | non-blocking | test-gap | noted | `prepush_test.go` covers neither a ref outside `refs/heads/` (a tag push) nor a linked worktree, the two shapes `hook_test.go` does cover for `pre-commit`; `pushRefFor`'s `dir` parameter is unused | `prepush_test.go` vs `TestPreCommitInALinkedWorktree`; both behaviours verified by hand in this review instead | F2's fix should bring the non-`refs/heads/` case with it; the worktree case and the dead parameter can wait |
| F9 | non-blocking | spec-unclear | noted | `doctor`'s new pass line "hooks: the pickle on PATH can run it" has lost its antecedent — the per-hook path it used to name is now printed on separate lines | `doctor.go` `checkHooks`, observed under `doctor -v` | while fixing F3, consider "…can run the guards" or naming the resolved binary |
| F10 | non-blocking | stale-xref | fixed inline | a `(see <<cmd-hooks>>)` xref this branch added points at the very section it sits in | `cli-reference.adoc:485`, inside `[#cmd-hooks]` (`:362`) | fixed in `b05f8bd`: replaced with "by the base-resolution order given above" |

dispositions: 4 blocking (F1–F4, not dispositioned — fixed in rework); 6 non-blocking — 2 fixed inline (F5, F10, both in `b05f8bd`), 1 folded (F7 → T-042), 3 noted (F6, F8, F9); 0 new tickets.
cost: estimated M, actual M

**What the four blocking findings have in common** is worth saying, because the rule itself is
sound: the guard's core — the three-dot range, the base-branch exemption, the fail-open contract,
the no-network resolution — is right, tested, and demonstrably catches T-073's real failure,
including on a *second* push where the stdin range would have waved it through. What slipped is
the perimeter: one push spelling the branch-name test does not recognise (F2), the two doc sites
that describe the guard to every installed project (F1), and two output paths the one-hook →
N-hooks generalization changed without anyone looking at them (F3, F4). All four are small,
local, and independently testable.

### 2026-08-14 — rework: F1–F4 fixed

- **F1** — `internal/install/install.go`'s rendered marker block and the hand-edited `AGENTS.md`
  both now name the `pre-push` hook and both bypasses (`git commit --no-verify` /
  `git push --no-verify`); golden fixture regenerated (`066212d`).
- **F2** — `CheckPrePush` no longer skips a `git push <remote> HEAD:refs/heads/feat/T-x`
  push: `branchBeingPushed` falls back to `RemoteRef` (always fully qualified) when `LocalRef`
  is the literal `HEAD`. A tag push's `RemoteRef` is `refs/tags/...`, so it still falls outside
  `refs/heads/` and stays skipped. Verified against a real push of that shape carrying
  `tickets/`, both before (silently allowed) and after (refused, naming the branch from
  `RemoteRef`) (`1c0788c`).
- **F3** — `hooks status` probes `hook.Probe()` at most once now (hoisted into
  `runHooksStatus`, matching `checkHooks`' existing shape), not once per owned hook; the
  PATH-capability line still prints per hook, matching the doc's own phrasing (`d429ec7`).
- **F4** — `hook.Install`'s foreign-hook refusal now sets `Skipped`, so a caller that only
  checks `Path != ""` no longer prints an empty parenthetical; both `hooks install` and
  `install --hooks` now report every attempted hook and run the inert-PATH check regardless of
  a sibling hook's refusal (`d429ec7`).

Re-ran the acceptance test's five throwaway-dir cases verbatim (all still pass) plus the F2 and
F4 regressions by hand against the real binary. `just build`, `just test`, `just lint`,
`just docs-check` all clean. New regression tests: `TestBranchBeingPushed`, two `TestCheckPrePush`
cases (`HEAD:refs/heads/...`, a tag push), `TestHooksStatusProbesPathOnce` (counts real execs via
a counting stub), `TestHooksInstallReportsSiblingHookDespiteAForeignOne`,
`TestInstallHooksReportsSiblingHookDespiteAForeignOne`, and a `Skipped`-is-set assertion added to
the existing foreign-hook test in `internal/hook`.

No new findings; the four non-blocking dispositions from the first review (F5–F10) are
unchanged and already closed. Ready for the scoped re-review.

### 2026-08-14 — scoped re-review (verdict: DONE)

Scope: the four blocking findings only, per the protocol — not a re-audit of the feature.

- [x] Blocking findings re-verified (F1–F4), each against the tree and the real binary (step 2)
- [x] Quality/consistency: scoped to the rework's own diff (step 3, 4)
- [x] Documentation audit — the two marker-block sites, `doctor`'s drift check, docs build clean (step 4a)
- [x] Docs-readability pass — **conscious skip**: the rework changed no `.adoc`/`.md` prose except
      `AGENTS.md`'s marker block, which is generated text that must match `markerBlock()`
      byte-for-byte and so cannot take readability edits (step 4b)
- [x] New findings recorded with severity, class and disposition; summary + `cost:` lines present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages & MR attributes presented for approval (step 9)

| finding | verified | evidence |
|---|---|---|
| F1 | **resolved** | `pickle doctor -v` reports `ok: AGENTS.md marker block current`, i.e. the hand-edited block matches `markerBlock()` byte-for-byte (T-041's drift check is the mechanical proof, stronger than reading both). `install.go`, `AGENTS.md` and the golden fixture agree; both hooks and both `--no-verify` bypasses are named |
| F2 | **resolved** | `git push origin HEAD:refs/heads/feat/T-994-h` carrying `tickets/` now exits 1 and names the branch resolved from `RemoteRef`; a tag push in the same shape still passes. `TestBranchBeingPushed` covers all four ref shapes |
| F3 | **resolved** | probe hoisted into `runHooksStatus`; the per-hook state line and the stale case are unchanged. The new test is **not vacuous**: reverting the hoist makes it fail with `exec'd 3 time(s), want 1` |
| F4 | **resolved** | `hooks install` with a foreign `pre-push` now prints `= …/pre-push (not pickle's — left in place)` (no empty parenthetical) *and* `+ …/pre-commit`; `install --hooks` reports both in the same situation; the `ErrNoRepo` path still prints only the skip warning and exits 0 |

Re-ran the acceptance test verbatim: cases 1–5 give exit 1, 0, 0, 0, 0 with both hooks reported
and `doctor` clean. `just build`, `just test`, `just lint`, `just docs-check` all clean.

Two new non-blocking findings, neither reopening the ticket:

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F11 | non-blocking | correctness | new ticket | `branchBeingPushed` prefers the refspec's **source** ref, so the branch test reads the wrong side when the two differ: `main:refs/heads/feat/T-900-x` carrying unpushed bookkeeping is **allowed** (a false pass — an MR from that destination would carry it), and `feat/T-901-x:refs/heads/main` is **refused** (a false refusal at bookkeeping's correct destination) | both measured against the shipped binary in a throwaway clone; F2's own fix is unaffected either way | **T-100**, batched with F6/F8/F9: prefer `RemoteRef`, fall back to `LocalRef`, keep tag pushes skipped in both orders |
| F12 | non-blocking | stale-xref | fixed inline | the rework's own comment on `printHookStatus` called `hook.Reach` "Reachability", a type that does not exist | `internal/cli/hooks.go:184` | fixed in `ce88d60` |

F11 predates the rework — the original implementation read `LocalRef` only, so it is not a
regression the fix introduced; the fix made the precedence question visible. It is **not**
blocking because the shape F2 named (and the one measured against T-073's real incident) is
closed, the guard's fail-open contract is intact, and the remaining spellings need a decision
about which side of a refspec the rule reads — a refinement question, not a rework one.

dispositions: 2 non-blocking — 1 new ticket (F11 → **T-100**, carrying F6, F8 and F9 with it by
citing their rows), 1 fixed inline (F12, `ce88d60`); 0 blocking.
cost: estimated M, actual M — the rework added four small fixes, none of which shifted the band.

**Verdict: DONE.** All four blocking findings are resolved, each with a test that fails without
its fix. The guard does what the ticket set out to make it do: it refuses the push T-073 actually
shipped, on a plain `git push` and on a second push where the stdin range would have waved it
through, while never refusing the base branch.

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — filed from T-072's review at the user's request. T-072 closed the publish-time
  bookkeeping leak in prose and, in doing so, corrected its own earlier reasoning that a
  `pre-push` guard could not catch it — it can, verified against T-073's real SHAs (7 `tickets/`
  paths in `152fea8...850ea3c`). Filed rather than folded into T-072 because it is Go code across
  `internal/hook`, `internal/cli` and `doctor` plus a `ShimVersion` bump, against T-072's
  prose-only scope. Passes the promotion test on frequency alone: four occurrences, one of which
  shipped
- 2026-08-09 — T-084's review impact sweep: noted in the Description that the rebase/keep-history
  default for a root-path child changes the failure's *shape* (preserved, not folded) without
  changing this ticket's scope or its three-dot `origin/<base>...HEAD` check.
- 2026-08-13 — T-091's review impact sweep: recorded the soft coupling to T-091 and, with it, the
  `pre-commit` add-without-delete check T-091 note-and-closed to this ticket's staged-path
  plumbing — a hand-off that until now existed only in T-091's own Description. Scope, grading
  and open questions unchanged; refinement decides whether to absorb the extra check.
- 2026-08-13 — refined: all three open questions closed (a `pre-push` hook rather than a
  `publish-check` verb; alongside T-072's prose, not superseding it; T-091's add-without-delete
  check declined), and two premises the ticket was filed on corrected. First, the range cannot be
  read off `pre-push` stdin — it is all-zero on a new branch and only `last-pushed…local`
  afterwards, neither of which is what a forge diffs — so the guard measures
  `<remote>/<base>...<local>`, which in turn forced a base-resolution rule, since `pickle.toml`
  has no base-branch key (`config.Project`, `internal/config/config.go:104-116`). Second,
  T-091's "same staged-path data" hand-off is factually wrong: this guard reads a commit range,
  not the index, so the shared plumbing it assumed does not exist. Grading re-assessed against
  the backlog and confirmed unchanged at medium/medium/M — the six-function generalization plus a
  six-file doc sweep is not an S
- 2026-08-13 — TO DO → READY: plan complete: pre-push hook, range = <remote>/<base>...<local>
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
- 2026-08-13 — plan amended inline: Task 1's `PreCommit`/`PrePush` `Name` constants collide with
  the pre-existing `PreCommit` rule function and Task 2's proposed `PrePush` rule function (a
  const and a func cannot share one identifier in Go); renamed the two rule functions to
  `CheckPreCommit`/`CheckPrePush` and kept the `Name` constants as specified. Also switched the
  acceptance test's throwaway binary name from `pk` to `pickle-test` to match `AGENTS.md`'s
  self-modify policy, with a same-directory `pickle` symlink so the installed shim's own `pickle
  hooks run <name>` call resolves it. Neither changes scope, grading, or any confirmed design
  decision.
- 2026-08-13 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-13 — IN REVIEW → REWORK: review: 4 blocking (F1 marker-block/AGENTS.md docs sites undone, F2 HEAD:refs/heads/feat push bypasses the guard, F3 hooks status probes per hook against its own doc, F4 InstallAll partial-failure misreports), 6 non-blocking (2 fixed inline, 1 folded to T-042, 3 noted)
- 2026-08-13 — REWORK → IN REVIEW: F1-F4 fixed on the same branch, findings F5-F10 unchanged
- 2026-08-13 — IN REVIEW → DONE: scoped re-review: F1-F4 all resolved with non-vacuous tests; 2 new non-blocking (F11 -> T-100, F12 fixed inline)
