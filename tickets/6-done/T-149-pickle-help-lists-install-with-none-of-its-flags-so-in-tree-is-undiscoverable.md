---
id: T-149
title: pickle help lists install with none of its flags, so --in-tree is undiscoverable
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-149 — pickle help lists install with none of its flags, so --in-tree is undiscoverable

## Outcome

`pickle help` shows the install flags a new user needs on day one (`--in-tree`, `--agent`,
`--path`, `--project`), so choosing a layout no longer depends on reading `pickle install --help` or
the manual first.

## Description

On `main` (0017563), the `install` entry in `pickle help` describes what install does but lists
none of its flags. `--in-tree` appears only in `pickle install --help` (the `flag` package's usage
dump). It is the one choice that is costly to undo after the fact (see T-148), so it belongs in the
first help a user reads.

Add an indented flag summary under `install` in the top-level help, in the style the
`ticket new` entry already uses. Keep `docs_xref_test.go` green, and update the user manual
wherever it quotes the help text. Soft coupling: T-121 (install help text accuracy).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                          # the pickle child is the repo root
git checkout main && git pull --ff-only
git checkout -b feat/T-149-install-flags-in-help
```

Commit locally as you go (WIP commits are encouraged). Follow this repo's self-modify policy:
never run `pickle install|upgrade|uninstall` against this repo. Do test installs in a throwaway
dir with the binary copied in as `pickle-test`. `layout = "in-tree"`: before pushing, run
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`, which
must print nothing. No push and no MR without explicit user approval. Merging is always the
human's.

### Prerequisite gate (hard)

None. `main` is clean and `just test` passes on it.

### Confirmed design decisions (do not deviate without asking)

1. **Top-level help lists only the install flags that shape the install, plus a pointer to the
   rest.** Those flags are `--in-tree`, `--path <dir>`, `--project <name>`, `--agent <list>` and
   `--hooks`, followed by one line: `pickle install --help` lists every flag. The four command
   flags and `--claude-symlink` stay out, to keep `pickle help` scannable.
2. **The format follows the existing entries.** Continuation lines are indented to the
   description column, in the style of `ticket new`'s second line.
3. **Each later flag ticket adds its own help line.** T-148 (`uninstall --purge`) and T-150
   (`install --ticket-prefix`) update `usage()` for their own flags; this ticket does not
   anticipate them.
4. **A test keeps the listed flags advertised.** It sits beside `TestServeHelpIsAdvertised` and
   fails if any decision-1 flag disappears from `pickle help`.

### Tasks

#### Task 1 — Help text
In `internal/cli/cli.go` `usage()`, add the decision-1 flags under the `install` entry, one short
clause each:
- `--in-tree`: board inside its sole child, registered at "."
- `--path`/`--project`: first child for the umbrella layout
- `--agent`: which agents to wire up
- `--hooks`: also install the guard hooks

#### Task 2 — Test
In `internal/cli/cli_test.go`, add `TestInstallFlagsAreAdvertised`. It captures `Run(nil, "test",
[]string{"help"})` with `captureStdout` and asserts that each of `--in-tree`, `--path`,
`--project`, `--agent`, `--hooks` and `pickle install --help` appears.

### Acceptance test

```
just build && just test && just lint && just docs-check
./pickle help | grep -- '--in-tree'                     # prints the new line
./pickle help | grep -c -- '--agent\|--hooks\|--path'   # ≥ 3
go test ./internal/cli/ -run TestInstallFlagsAreAdvertised -v   # PASS
```

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc` Overview table: change the install row to
  `pickle install [--in-tree] [--agent …]`. The `== pickle install` synopsis already lists every
  flag.
- `CHANGELOG.md` `[Unreleased]` *Changed*: `pickle help` lists install's layout, child-project and
  agent flags (T-149).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs and the `CHANGELOG.md` entry above in place.
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggested commit(s), Conventional Commits with the id at the end of the subject:
   - `fix(cli): list install's layout, child and agent flags in pickle help (T-149)`
5. Root-path child: before presenting, tidy the WIP commits into the atomic commits above.
6. Commit locally. Do not push. `pickle ticket move T-149 in-review --reason "acceptance green"`.

## Review

- [x] Reviewer independence settled (step 0): the orchestrator wrote this branch in this same
  session, so steps 1-4a were **delegated** to a freshly spawned independent sub-agent using the
  verbatim spawn prompt (`hunt: one pass`, low complexity). Its one finding was re-verified by hand
  against `internal/cli/install.go` before recording. Classification, disposition and the move
  stayed with the orchestrator.
- [x] Step 0a: ticket resolved from `main`; `pickle doctor` flagged the branch's stale ticket
  copy (3-in-development vs main's 4-in-review). This is expected: the branch carries no `tickets/`
  path in its diff.
- [x] Implementation audit (step 2): Tasks 1-2 and decisions 1-4 met. Acceptance test re-run
  verbatim and green (`just build && just test && just lint && just docs-check`; `grep -c` = 3;
  `TestInstallFlagsAreAdvertised` PASS).
- [x] Hunt (step 3, one pass, all eight angles): static help text plus one test, with no
  behaviour, flag or exit-code change. No correctness findings.
- [x] Quality audit (step 3): the test is scoped to the install entry and fails if any listed flag
  line is removed.
- [x] Consistency audit (step 4): no other doc quotes the old install help prose. The
  `== pickle install` synopsis already lists every flag and agrees with the new lines.
- [x] Documentation audit (step 4a): Overview row and CHANGELOG `### Changed` entry in place,
  ordered between Added and Fixed. `just docs-check` green.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | docs-gap | fixed inline | `--project <name>   that child's name` reads as the `--path` child only, but `--project` also names the sole `--in-tree` child, and the line omitted the default. | `internal/cli/install.go`: `--in-tree` sets `path = "."`, and `--project` defaults to the root dir name | Reworded to `the child's name (default: root dir name)`, folded into the branch commit (`4c20951`). |

Disposition summary: 1 finding, non-blocking — F1 fixed inline. 0 blocking findings.

cost: estimated S, actual S

Verdict: approved → DONE.

## History

- 2026-11-07 — created (TO DO). source: chat: user found --in-tree absent from pickle help; confirmed on main
- 2026-11-07 — TO DO → READY: plan complete; decisions confirmed with the user 2026-11-07
- 2026-11-08 — READY → IN DEVELOPMENT: picked up
- 2026-11-08 — plan amended inline: TestInstallFlagsAreAdvertised asserts on the install entry only (--agent and --project already appear elsewhere in help, so a whole-text check passes vacuously); install prose drops its --agent clause now that --agent has its own line. Applicability gate: 5 non-blocking findings, 2 amended inline, 3 noted and closed.
- 2026-11-08 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-11-08 — IN REVIEW → DONE: review approved, 0 blocking
- 2026-11-08 — published: pushed feat/T-149-install-flags-in-help, opened PR #109 (91450a6)
