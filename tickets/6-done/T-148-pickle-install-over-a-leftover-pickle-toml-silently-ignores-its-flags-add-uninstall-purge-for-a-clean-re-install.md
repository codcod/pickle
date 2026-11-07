---
id: T-148
title: pickle install over a leftover pickle.toml silently ignores its flags; add uninstall --purge for a clean re-install
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-148 — pickle install over a leftover pickle.toml silently ignores its flags; add uninstall --purge for a clean re-install

## Outcome

Re-running `pickle install` with different flags (the "forgot `--in-tree`" case) either takes
effect or is refused with a message naming the conflict, never silently ignored. `pickle uninstall
--purge` returns a project with no tickets to its pre-install state, so a clean re-install is one
command.

## Description

Reproduced on `main` (0017563) in a throwaway dir: `pickle-test install`, `pickle-test uninstall`,
`pickle-test install --in-tree`. The second install reports `= pickle.toml (exists)` and keeps the
first install's `layout = "umbrella"` with no `[[projects]]`, yet prints
`pickle installed in … (child "<dir>")`. The `--in-tree` flag is silently dropped, the success line
names a child that was never registered, and the regenerated `AGENTS.md` marker block still
describes the umbrella layout.

Two parts:

1. **Install over an existing `pickle.toml`** (the root cause; any leftover config hits it, not only
   after uninstall). When the flags ask for a layout or first child that disagrees with the
   existing config, refuse with a message naming the disagreement and the way out
   (`uninstall --purge`, or `project add`). When they agree, or no layout/child flags were given,
   keep today's idempotent behaviour. The success line names a child only if one is registered.
2. **`pickle uninstall --purge`.** On top of today's removals, it also removes `pickle.toml`, the
   untouched `tickets/` scaffold (the `.gitkeep` files, the status dirs, and `BOARD.md`, `NOTES.md`
   and `README.md` while they are unchanged from what install wrote), the now-empty
   `.agents/skills/` and `.claude/skills/` dirs, and `AGENTS.md`/`CLAUDE.md` if they are empty once
   the marker is stripped. It **refuses** if any ticket file exists or `NOTES.md` was edited. That
   keeps T-006's decision D1 (uninstall never destroys work): `--purge` only deletes what install
   itself created and nobody has touched since. It honours `--dry-run`.

Soft couplings: T-006 (uninstall scope, D1), T-108 (in-tree layout, the conflicting-flags refusal
this extends).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                          # the pickle child is the repo root
git checkout main && git pull --ff-only
git checkout -b feat/T-148-reinstall-refuses-conflicting-flags-and-uninstall-purge
```

Commit locally as you go (WIP commits are encouraged). Follow this repo's self-modify policy:
never run `pickle install|upgrade|uninstall` against this repo. Do test installs in a throwaway
dir with the binary copied in as `pickle-test`. `layout = "in-tree"`: before pushing, run
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`, which
must print nothing. No push and no MR without explicit user approval. Merging is always the
human's.

### Prerequisite gate (hard)

None. `main` is clean and `just test` passes on it. T-149, T-150 and T-151 are not needed; each
adds its own help and docs lines when it lands.

### Confirmed design decisions (do not deviate without asking)

1. **Install never rewrites an existing `pickle.toml`; a contradicting flag is refused instead.**
   User decision, 2026-11-07. `writeConfig` keeps its load-only behaviour for an existing file.
   The refusal exits 2 (usage), because the invocation is what is wrong, not the tree.
2. **A mismatch is a flag that states something the existing config contradicts. Nothing else is.**
   The cases: `--in-tree` while the config's `ResolvedLayout()` is not `in-tree`; `--path` or
   `--in-tree` naming a child that is not registered at that path (with an explicit `--project`, by
   that name; without one, whatever child is registered at the path matches — amended 2026-11-07,
   review 1 F2); and `--build`, `--test`, `--lint` or `--docs` whose value differs
   from that registered child's (an omitted command flag states nothing, so it never mismatches). Silently dropping a differing command is the same bug. Plain
   `pickle install`, with none of these flags, states no layout (`--in-tree` is the only layout
   flag, per T-108 decision 1). It is therefore never a mismatch: it re-attaches exactly as today,
   but adds one note naming the kept layout and child count, and that flags cannot change an
   existing config.
3. **The check runs before anything is written.** A refused install leaves the tree byte-identical:
   no payload copy, no marker refresh.
4. **The refusal names every mismatch and both ways out.** The ways out are
   `pickle uninstall --purge` followed by the reinstall, or `pickle project add` to register a
   further child.
5. **The success line reports what `pickle.toml` actually registers, read from the loaded config,
   never from the flags.**
6. **`uninstall --purge` deletes only what install creates, and refuses when it finds work.**
   User decision, 2026-11-07. The allowlist under `tickets/` is: the seven status dirs and their
   `.gitkeep`; `BOARD.md` (generated, always regenerable); `NOTES.md` only while byte-identical
   to `notesScaffold`; and `README.md` only while byte-identical to `ticketsReadme`. Anything
   else under `tickets/` makes the purge refuse: a ticket, an edited `NOTES.md` or `README.md`, or
   an unknown file such as a retro script. The refusal exits 1, names each offending path, and
   **removes nothing**, not even the skill dir. The work check therefore runs before any removal.
7. **`--purge` removes, on top of plain uninstall's set:** `pickle.toml`, even if hand-edited,
   since config is what a purge targets; the allowlisted `tickets/` files and then the
   `tickets/` dirs, bottom-up; and `AGENTS.md` / `CLAUDE.md` when stripping the marker leaves
   nothing, or only the `# <MarkerTitle>` header `injectMarker` wrote when it created the file. It
   also prunes `.agents/skills/`, `.agents/`, `.claude/skills/` and `.claude/`, using `os.Remove`
   only, so a non-empty dir survives (the same contract as the `.pi/` prune).
8. **`--purge` honours `--dry-run` (`-n`).** It lists the extra items with ` (dry-run)` and runs
   the same work check: a dry run that would refuse says so and exits 1.
9. **Plain `uninstall` is unchanged** (T-006 decision D1: it keeps `tickets/` and `pickle.toml`).

### Tasks

#### Task 1 — Existing-config check in `internal/install`
In `internal/install/install.go`, add an exported sentinel `ErrConfigMismatch` and
`checkExistingConfig(root string, opts Options) error`. When `pickle.toml` exists, load it and
collect every decision-2 mismatch into one error wrapping `ErrConfigMismatch`, whose text follows
decision 4. Call it first in `Run`, before `copyPayload`. When the file exists and no
layout/child flag was given, add the decision-2 note via `res.note`. `Options` already carries
every field compared (`InTree`, `ProjectName`, `ProjectPath`, the four commands). The CLI's name
default is resolved before `Run`, so the check sees the same name the CLI would register.

#### Task 2 — CLI: exit code and truthful success line
In `internal/cli/install.go` `runInstall`, map `errors.Is(err, install.ErrConfigMismatch)` to
printing the error and `return exitUsage`. Replace the closing `if *path != ""` success branch
with one driven by the already-loaded `cfg`. If `cfg.Project(name)` is found, print today's child
line. Otherwise, if `cfg.Projects` is non-empty, print a re-attach line naming the layout and the
registered children. Otherwise print today's umbrella / no-child line.

#### Task 3 — `UninstallOptions.Purge`
In `internal/install/install.go`, add `Purge bool` to `UninstallOptions`. Add
`purgeBlockers(root string) ([]string, error)`, which walks `tickets/` against the decision-6
allowlist; it uses `flow.Default().States()` for the dirs, and `notesScaffold` and
`ticketsReadme` for the byte comparisons. In `Uninstall`, when `opts.Purge`, call it before the
legacy sweep. Return an error listing the blockers if there are any. After the existing removals,
call a new `purgeScaffold(root, opts, &res)` doing decision 7. Match the existing `res.removed`
labels and the dry-run suffix. The empty/header-only test on `AGENTS.md`/`CLAUDE.md` computes the
post-strip text in memory (the before/after logic pulled out of `stripMarker`), so a dry run
answers the same as a real one.

#### Task 4 — CLI flag and help
In `internal/cli/install.go` `runUninstall`, add `--purge` and pass it through. For a purge,
make the summary line say pickle was fully removed instead of "tickets/ and pickle.toml left
intact". In `internal/cli/cli.go` `usage()`, make the uninstall entry read
`uninstall [--dry-run] [--purge]`, adding one continuation line that `--purge` also removes
`pickle.toml` and the untouched `tickets/` scaffold, and refuses if any work is found.

#### Task 5 — Tests
In `internal/install/install_test.go`:
- `TestRunRefusesConflictingFlagsOverExistingConfig`, table-driven: `--in-tree` over an umbrella
  config; `--path sub` over an umbrella config with no children; a differing `--build` for the
  registered child. Each returns `ErrConfigMismatch`, and a before/after snapshot (file list
  plus bytes) proves the tree is unchanged.
- `TestRunReattachesWithMatchingOrNoFlags`: identical flags succeed, and no flags over an in-tree
  config succeed with the note.
- `TestUninstallPurgeRestoresPreInstallTree`: in a temp dir holding only a `.git/` dir and a
  user file, install then purge, and only those two remain.
- `TestUninstallPurgeRefuses`, table-driven: a ticket file, an edited `NOTES.md`, an unknown
  `tickets/retros/x.py`. Each refuses, names the path, and leaves the skill dir in place.
- `TestUninstallPurgeKeepsUserAgentsProse`: an `AGENTS.md` with user text survives, marker
  stripped.
- `TestUninstallPurgeDryRunChangesNothing`.

In `internal/cli/cli_test.go`, add a case that `install --in-tree` over an umbrella install
returns `exitUsage`.

### Acceptance test

From the repo root on the feature branch:

```
just build && just test && just lint && just docs-check

D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q
./pickle-test install                                 # umbrella, no child
snap() { find . -path ./.git -prune -o -type f -print | sort | xargs shasum; }
snap > /tmp/before.$$
./pickle-test install --in-tree; echo "exit=$?"       # exit=2; stderr names layout umbrella vs
                                                      # --in-tree, uninstall --purge and project add
snap | diff /tmp/before.$$ - && echo unchanged        # unchanged
./pickle-test install                                 # re-attach note; umbrella success line
./pickle-test uninstall --purge --dry-run             # lists pickle.toml, tickets/…, AGENTS.md,
snap | diff /tmp/before.$$ - && echo unchanged        # CLAUDE.md as (dry-run); unchanged
./pickle-test uninstall --purge
find . -path ./.git -prune -o -print | sort           # only "." and "./pickle-test"
./pickle-test install --in-tree && grep '^layout' pickle.toml     # layout = "in-tree"
./pickle-test ticket new "x" --project "$(basename "$D")"
./pickle-test uninstall --purge; echo "exit=$?"       # exit=1, names tickets/1-to-do/T-001-x.md
test -d .agents/skills/brine && echo skill-kept       # skill-kept (nothing removed)
```

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`:
  - In `== pickle install`, add a short "Re-running install" paragraph covering decisions 1, 2
    and 4.
  - In `[#cmd-uninstall]`, add the `--purge` synopsis and a paragraph covering decisions 6–8.
    Keep the existing "left intact" sentence, scoped to plain `uninstall`.
  - Overview table: make the uninstall row read `[--dry-run] [--purge]`.
- `CHANGELOG.md` `[Unreleased]`:
  - *Added*: `pickle uninstall --purge` (T-148).
  - *Fixed*: `pickle install` over an existing `pickle.toml` refuses flags that config
    contradicts, instead of silently ignoring them, and its success line no longer names a child
    that was never registered (T-148).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs and the `CHANGELOG.md` entry above in place.
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggested commit(s), Conventional Commits with the id at the end of the subject:
   - `fix(install): refuse flags an existing pickle.toml contradicts (T-148)`
   - `feat(uninstall): add --purge to remove pickle.toml and the untouched scaffold (T-148)`
   - `docs: document re-install refusal and uninstall --purge (T-148)`
5. Root-path child: before presenting, tidy the WIP commits into the atomic commits above.
6. Commit locally. Do not push. `pickle ticket move T-148 in-review --reason "acceptance green"`.

## Review

### Review 1 — 2026-11-07

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | `uninstall --purge` through a symlinked `tickets/` skips the work check and deletes outside the root; with a ticket present it half-purges, then fails (T-148 decision 6) | reproduced: `install`; `mv tickets ../shared/tickets && ln -s ../shared/tickets tickets`; `echo precious > tickets/NOTES.md`; `uninstall --purge` → exit 0 "pickle fully removed", `../shared/tickets/` left empty. `filepath.WalkDir` does not descend a symlinked root, so `purgeBlockers` finds nothing; `os.Remove` then follows the symlinked parent. With `tickets/1-to-do/T-001-x.md` added: skill, markers, `pickle.toml`, BOARD/NOTES/README removed, then `remove …/tickets/1-to-do: directory not empty` | in `purgeBlockers`, Lstat `tickets/` and block when it is not a real dir; block on any symlink entry inside the walk, allowlisted name or not; add a symlinked-`tickets/` case to `TestUninstallPurgeRefuses` |
| F2 | blocking | correctness | — | `install --in-tree` (or `--path`) without `--project` is refused whenever the registered child's name differs from the dir name (a renamed clone, a worktree), because the CLI's defaulted name is compared as if the user had typed it; the refusal then advises `project add`, wrong for in-tree (T-148 decision 2) | reproduced: in dir `svc.PpfM`, `install --in-tree --project api`, then `install --in-tree` → exit 2 `child "svc.PpfM" is not registered` | when `--project` was not given, match the child registered at the given path rather than by the defaulted name; needs user sign-off, since it amends decision 2 |
| F3 | non-blocking | correctness | noted | a symlinked `.agents`/`.claude` dir, or a symlinked status dir, is unlinked by the purge prune even when its target is non-empty (and that dir's `.gitkeep` inside the target goes); no user data lost | reproduced by the delegated reviewer: `ln -s ../shared/agents .agents` with `shared/agents/other.txt`; install; purge → link gone, `other.txt` survives | Lstat each prune target and skip non-dirs; natural to fold into F1's fix if the rework touches the same lines |
| F4 | non-blocking | test-gap | noted | the CLI's re-attach success line and the `--purge` summary line have no CLI-level test | grep of `re-attached`/`fully removed` across `internal/**/*_test.go` finds none | one CLI test per line |
| F5 | non-blocking | stale-xref | fixed inline | `cli-reference.adoc` `[#cmd-uninstall]` said a marker file "itself is never deleted", which `--purge` made false | `cli-reference.adoc`, shared removals paragraph | scoped to "without `--purge`" (e6b549b) |
| F6 | non-blocking | correctness | noted | purge refuses on OS cruft such as a Finder `.DS_Store` under `tickets/`; refusal is safe, nothing lost | reproduced: `touch tickets/1-to-do/.DS_Store`; `uninstall --purge -n` → refused naming it | allowlist (and remove) known OS cruft if it bites |
| F7 | non-blocking | correctness | noted | `purgeScaffold` removes `pickle.toml` first, so a later removal failure leaves a tree `uninstall` can no longer find | unreproduced: needs a mid-purge I/O failure; traced `purgeScaffold` → `rm(config.FileName)` precedes the `tickets/` removals | remove `pickle.toml` last |
| F8 | non-blocking | correctness | noted | `NOTES.md`/`README.md` count as untouched only when byte-identical to this binary's text; an install from a release with different text is refused | unreproduced: `ticketsReadme` changed once (T-044), so installs older than that hit it; refusal is safe | accept any shipped version, if it ever matters |
| F9 | non-blocking | correctness | noted | a pre-existing empty `CLAUDE.md`/`AGENTS.md` (or one holding only `# Brine`), or a pre-existing empty `.claude/`/`.agents/`, is removed by the purge | traced: `purgeScaffold` cannot tell whether install created them; content is empty, so nothing is lost | none |
| F10 | non-blocking | correctness | noted | path comparison is lexical: `--path $PWD/sub` is refused against a child registered as `sub` | traced: `checkExistingConfig` compares `filepath.Clean` of both | resolve both against root before comparing, if it ever bites |
| F11 | non-blocking | correctness | noted | a config with no `layout` key in a dir without `.git` resolves to umbrella, so `install --in-tree` over it is refused | traced: `ResolvedLayout` infers in-tree from `vcs.IsRepoRoot`; long-standing inference, not introduced here | none |
| F12 | non-blocking | design | noted | output wording: a purge that kept user files still says "pickle fully removed", and a matching-flags re-run prints "installed … (child)" where a flag-less one prints "re-attached" | read `runUninstall`/`runInstall` summary branches | reword if users trip on it |

Disposition summary: 2 blocking (F1, F2); 1 fixed inline (F5); 9 noted (F3, F4, F6–F12); 0 folded; 0 new tickets.
cost: estimated M, actual M

- [x] Reviewer independence settled (step 0): delegated — the orchestrator wrote the branch this session; a fresh sub-agent ran steps 1–4a (hunt: angle 7 only). No delegated findings discarded.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned the branch had T-148 in 3-in-development; rebased onto `main`, re-run clean.
- [x] Implementation audit (steps 1, 2): acceptance test re-run by the delegated reviewer, green; Tasks 1–5 present (the CLI refusal test landed in `layout_test.go`, not `cli_test.go` — harmless).
- [x] Correctness hunt (step 3): host tool path — the orchestrator ran the host code-review tool (medium) on `main...feat/T-148-…`; 10 findings, 0 discarded on re-verification (recorded as F2, F6–F12). Angle 7 ran separately in the delegated reviewer (F1, F3). Every correctness row reproduced or marked unreproduced.
- [x] Consistency audit (step 4): delegated; one contradiction (F5).
- [x] Documentation audit (step 4a): delegated; coverage present for both changes, `just docs-check` clean.
- [ ] Docs-readability pass (step 4b): conscious skip — the changed prose is short reference text.
- [x] Findings recorded with severity, class and disposition; summary and cost lines present (step 5).
- [x] Ticket moved to `tickets/5-rework/`; History appended (step 6).
- [x] Other references (step 7): none beyond F5; governing docs unaffected.
- [x] Impact sweep (step 8): T-150 adds an install flag (`--ticket-prefix`) that must join `checkExistingConfig`'s mismatch check; T-149 and T-151 must list `uninstall --purge` in help/completion. Each ticket already owns its own help/docs lines.
- [x] Summary + commit messages for approval (step 9): approved 2026-11-07; remote base verified (no `tickets/` path in `origin/main...HEAD`); pushed, PR #108

### Rework fix record — round 1 (commits ac869c2..5692f68)

- **F1** (2c864c7): `purgeBlockers` Lstats `tickets/` and refuses when it is not a plain directory, and refuses on any symlink entry inside the walk, allowlisted name or not, so no removal ever runs through a link. Regression: `TestUninstallPurgeRefuses/symlinked_tickets_dir` (fails before the fix). The F1 repro now exits 1 with `tickets (not a plain directory)`, and the shared `NOTES.md` survives.
- **F2** (5692f68): when `--project` is not given, `runInstall` takes the name of the child already registered at the given path, so the existing-config check and the success line see the registered child. Decision 2 was amended to match (user sign-off 2026-11-07). Regression: `TestInstallInTreeRerunMatchesChildByPath` (fails before the fix).
- Acceptance test re-run green; `just build`, `just test`, `just lint` and `just docs-check` are clean.

### Review 2 — 2026-11-07 (scoped re-review of F1, F2 and `ac869c2..5692f68`)

F1 and F2 are resolved: the original reproductions and their variants hold (symlinked status dir and `BOARD.md` refused; an unnamed re-run re-attaches to `api`, by `./sub` and `sub/` alike, and after a root rename), and an explicit differing `--project` or `--build` is still refused.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | non-blocking | stale-xref | fixed inline | the rework fix record named pre-rebase SHAs | `diff <(git diff 0b2c9d6..d703fd2) <(git diff ac869c2..5692f68)` is empty | record re-pointed at `ac869c2..5692f68` (this commit) |
| R2 | non-blocking | test-gap | noted | F1's two guards each mask the other in the regression test, so a symlink *inside* `tickets/` has no test; the assertion text is loose | mutation check by the delegated reviewer: disabling either guard alone keeps `-run Purge` green; by hand both inside cases refuse (`tickets/2-ready (symlink)`, `tickets/BOARD.md (symlink)`) | add a symlinked-status-dir case asserting `(symlink)` |
| R3 | non-blocking | stale-xref | fixed inline | `--project` flag help and the "Re-running install" docs still stated the dir-name default the F2 fix changed | `internal/cli/install.go` flag help; `cli-reference.adoc` § Re-running install | both reworded (fixup commits, folded into the tidy) |
| R4 | non-blocking | design | noted | with nothing registered at the path, the refusal names the defaulted name the user never typed | reproduced: umbrella with `web` at `sub`; `install --path other` → `child "umb.URLN" is not registered` | say `no child registered at path "other"` when `--project` was omitted |
| R5 | non-blocking | correctness | noted | two children sharing a path: the name lookup takes the last match | unreproduced: needs a hand-built config; traced `runInstall` loop without `break` | break on first match or treat as ambiguous |
| R6 | non-blocking | design | noted | `pickle.toml` is loaded in the CLI and again in `checkExistingConfig` | read `runInstall` and `Run` | leave |

Disposition summary: 0 blocking; 2 fixed inline (R1, R3); 4 noted (R2, R4, R5, R6); 0 folded; 0 new tickets.
cost: estimated M, actual M

- [x] Reviewer independence (step 0): delegated — fresh sub-agent, `hunt: one pass`; nothing discarded.
- [x] In-tree stale-branch check (step 0a): branch rebased onto `main`; `pickle doctor` clean.
- [x] Implementation audit (steps 1, 2): scoped to F1, F2 and the diff that closed them; acceptance test re-run green; `just build/test/lint/docs-check` clean.
- [x] Correctness hunt (step 3): one pass over all eight angles on the fix diff, by the delegated reviewer, with mutation checks of both F1 guards; R5 unreproduced, path traced.
- [x] Consistency audit (step 4): R3.
- [x] Documentation audit (step 4a): R3; docs build clean.
- [ ] Docs-readability pass (step 4b): conscious skip — two short reference sentences changed.
- [x] Findings recorded; summary and cost lines present (step 5).
- [x] Ticket moved to `tickets/6-done/`; History appended (step 6).
- [x] Other references (step 7): R3; nothing else.
- [x] Impact sweep (step 8): unchanged from review 1 (T-150's new install flag joins the mismatch check; T-149/T-151 list `--purge`).
- [x] Summary + commit messages for approval (step 9): approved 2026-11-07; remote base verified (no `tickets/` path in `origin/main...HEAD`); pushed, PR #108

## History

- 2026-11-07 — created (TO DO). source: chat: reproduced the user-reported re-install trap (install, uninstall, install --in-tree silently keeps the old umbrella config and misreports a child) in a throwaway dir
- 2026-11-07 — TO DO → READY: plan complete; decisions confirmed with the user 2026-11-07
- 2026-11-07 — READY → IN DEVELOPMENT: picked up
- 2026-11-07 — plan amended inline: applicability gate (independent sub-agent) found no blocking findings; decision 2 now states an omitted command flag never mismatches, and Task 3 computes the post-strip AGENTS.md/CLAUDE.md text in memory so --dry-run agrees; docs anchor is `[#cmd-install]` (note-and-close)
- 2026-11-07 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-11-07 — IN REVIEW → REWORK: review 1: 2 blocking (F1 symlinked tickets/ purge, F2 defaulted child name refused); 1 fixed inline, 9 noted
- 2026-11-07 — plan amended inline: decision 2 — without `--project`, the child is matched by its registered path, not by the CLI's defaulted name (review 1 F2; user sign-off 2026-11-07)
- 2026-11-07 — REWORK → IN REVIEW: findings fixed
- 2026-11-07 — IN REVIEW → DONE: review 2: F1, F2 resolved; 0 blocking; fixed inline R1, R3; noted R2, R4-R6; 0 new tickets
- 2026-11-07 — filed T-152: user promoted R4 and F6 to blocking after done (protocol §6c); it ships on this ticket's unpushed branch, in the same PR
- 2026-11-07 — published: user approved; PR https://github.com/codcod/pickle/pull/108 (T-148 + T-152 on one branch); awaiting human merge
- 2026-11-07 — merged to main (PR #108, 3361777, https://github.com/codcod/pickle/commit/3361777)
