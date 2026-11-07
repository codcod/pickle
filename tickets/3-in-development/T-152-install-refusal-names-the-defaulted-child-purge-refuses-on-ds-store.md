---
id: T-152
title: install refusal names the defaulted child; purge refuses on .DS_Store
project: pickle
depends-on: []
spawned-by: [T-148]
impact: medium
complexity: low
cost: S
---

# T-152 — install refusal names the defaulted child; purge refuses on .DS_Store

## Outcome

A re-install refused because nothing is registered at the given path says so by path, instead of
naming a child the user never typed. `pickle uninstall --purge` is no longer refused by a Finder
`.DS_Store` in the `tickets/` scaffold; it removes it along with the scaffold.

## Description

Two findings from T-148's review, promoted to blocking by the user after T-148 reached `6-done/`
(review protocol §6c), and batched as one theme: the re-install/purge messages and allowlist T-148
introduced.

1. **R4 (T-148 review 2).** In an umbrella install with child `web` at `sub`,
   `pickle install --path other` exits 2 with `child "<dir name>" is not registered`. The name
   is the CLI's dir-name default; the user typed only the path. Since T-148's F2 fix already
   resolves an omitted `--project` to the child registered at the path, this message is reached
   with a defaulted name only when nothing is registered there.
2. **F6 (T-148 review 1).** On macOS, Finder writes `.DS_Store` into a browsed folder;
   `tickets/1-to-do/.DS_Store` makes `uninstall --purge` refuse, naming it as work.

Soft coupling: T-148 (the code this amends, unpushed on its branch).

## Implementation Plan

### 0. Feature branch (mandatory)

User decision 2026-11-07: implement on T-148's unpushed branch so one PR ships both, instead of a
`feat/T-152-…` branch.

```
git checkout feat/T-148-reinstall-refuses-conflicting-flags-and-uninstall-purge
git rebase main
```

Self-modify policy as in T-148: test installs only in a throwaway dir as `pickle-test`. Under
`in-tree`, `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`
must print nothing before pushing. No push without approval; merging is the human's.

### Prerequisite gate (hard)

T-148 in `6-done/` (it is; its branch is unmerged by this decision, and this ticket rides on it).

### Confirmed design decisions (do not deviate without asking)

1. **Word the "not registered" refusal from the path, with no new `Options` field.** In
   `checkExistingConfig`, when `cfg.Project(name)` is not found: if no child is registered at
   `ProjectPath`, say `no child is registered at path "<path>"`; if one is, under another name,
   say `path "<path>" is registered to child "<other>", not "<name>"` (reachable only with an
   explicit `--project`).
2. **`.DS_Store` is allowlisted only directly in `tickets/` or a status dir**, never inside an
   unknown dir (which must stay a blocker), and `purgeScaffold` removes those files before the
   dirs. Dry run lists nothing extra for them: they are part of `tickets/ (7 status dirs)`.
   Other OS cruft (`Thumbs.db`, `desktop.ini`) stays out of scope.

### Tasks

1. `internal/install/install.go` `checkExistingConfig`: decision 1.
2. `internal/install/install.go` `purgeAllowed` and `purgeScaffold`: decision 2.
3. `internal/install/install_test.go`: a refusal case asserting the path wording; a purge case
   with `.DS_Store` in `tickets/` and in a status dir that purges to the pre-install tree; a case
   with `tickets/retros/.DS_Store` that still refuses.

### Acceptance test

```
just build && just test && just lint && just docs-check
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q
mkdir sub other && ./pickle-test install --path sub --project web >/dev/null
./pickle-test install --path other; echo "exit=$?"   # exit=2; 'no child is registered at path "other"'
touch tickets/.DS_Store tickets/1-to-do/.DS_Store
./pickle-test uninstall --purge | tail -1          # fully removed
find . -path ./.git -prune -o -print | sort        # ., ./other, ./pickle-test, ./sub
```

### Docs update (mandatory when user-facing)

`docs/user-manual/cli-reference.adoc` `[#cmd-uninstall]`: the `--purge` paragraph names a
`.DS_Store` in the scaffold as removed rather than refused. `CHANGELOG.md`: no new entry; the
T-148 entries are unreleased and cover it.

### Finish (mandatory)

1. Acceptance green; build/test/lint/docs-check clean.
2. One commit on T-148's branch: `fix(install): word re-install refusals by path and let purge clear .DS_Store (T-152)`.
3. `pickle ticket move T-152 in-review --reason "acceptance green"`, then a scoped review.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-07 — created (TO DO). source: pickle ticket new
- 2026-11-07 — refined: user promoted T-148 review findings R4 and F6 to blocking after T-148 reached done (protocol §6c); filed here, batched by theme
- 2026-11-07 — TO DO → READY: plan complete; decisions confirmed with the user 2026-11-07
- 2026-11-07 — READY → IN DEVELOPMENT: picked up
