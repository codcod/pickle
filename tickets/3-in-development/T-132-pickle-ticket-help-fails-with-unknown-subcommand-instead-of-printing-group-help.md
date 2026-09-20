---
id: T-132
title: pickle ticket --help fails with unknown subcommand instead of printing group help
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-132 — pickle ticket --help fails with unknown subcommand instead of printing group help

## Outcome

`pickle ticket --help` (and `-h`) prints the group's usage — the three `new`/`move`/`set` usage
lines — instead of failing with `pickle ticket: unknown subcommand "--help"`. The same fix
applies to every other group-level dispatcher that shares this exact bug shape (`board`,
`changelog`, `flow`, `hooks`, `scaffold`, `project`): each now prints its own group usage and
exits 0 on `-h`/`--help` instead of erroring. An agent unsure of a group's subcommand syntax gets
an answer instead of an error, at any of the seven groups, not just `ticket`.

## Description

`runTicket` in `internal/cli/ticket.go` (~line 21-37) dispatches on `args[0]` with a `switch`
covering `"new"`, `"move"`, `"set"`, and a `default` that prints `unknown subcommand %q` and
exits `exitUsage`. `--help`/`-h` fall into that default, so `pickle ticket --help` errors instead
of helping. Reproduced against both pickle 0.19.0 and 0.20.0 while dogfooding the flow in a
separate project (messgr) — an agent ran `pickle ticket move --help` successfully, then tried the
group-level `pickle ticket --help` and got the "unknown subcommand" error both times, across the
version bump. `pickle ticket new`/`move`/`set --help` (the leaf commands, via each `flag.FlagSet`)
already work correctly today — this is only the group level. Likely fix: recognize `-h`/`--help`
in `runTicket`'s switch and print a combined usage (the three `ticketNewUsage`/`ticketMoveUsage`/
`ticketSetUsage` consts already defined in the same file).

Confirmed during refinement: the identical `switch args[0] { ...; default: unknown subcommand }`
shape, with no `-h`/`--help` case, exists verbatim in six more group dispatchers —
`internal/cli/board.go` (`runBoard`), `changelog.go` (`runChangelog`), `flow.go` (`runFlow`),
`hooks.go` (`runHooks`), `scaffold.go` (`runScaffold`) and `project.go` (`runProject`). The
top-level `pickle --help`/`-h` (`internal/cli/cli.go` `Run`) already handles this correctly and
needs no change. This ticket now fixes all seven group dispatchers — one repeated one-line bug,
not seven independently schedulable features (rules §3) — per user sign-off during refinement.
`pickle hooks run <hook>` is a deeper, third dispatch level (a hook *name*, not a further
subcommand set) and stays out of scope.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-132-group-help
```

### Prerequisite gate (hard)

None.

### Confirmed design decisions (do not deviate without asking)

1. **Scope covers all seven group-level dispatchers that share this exact bug shape** —
   `ticket`, `board`, `changelog`, `flow`, `hooks`, `scaffold`, `project` — not just `ticket`
   alone. Confirmed with the user during refinement: it is the same one-line fix repeated
   identically seven times, not independently schedulable work (rules §3).
2. **`pickle hooks run <hook>` is out of scope.** `runHooksRun` (`internal/cli/hooks.go`
   ~line 222) dispatches on a hook *name* (`hook.Name(args[0])`), not a further subcommand set —
   a different shape one level deeper than the seven group dispatchers this ticket fixes.
3. **`-h`/`--help` at the group level prints to stdout and returns `exitOK` (0)** — matching
   `pickle --help`'s existing top-level behavior and this ticket's own Outcome ("an answer
   instead of an error"). The empty-args case in each dispatcher is unchanged: still `exitUsage`
   to stderr, since no subcommand was named at all. The `default` case for any other unrecognized
   subcommand is also unchanged.
4. **`ticket` and `board` print their existing rich per-subcommand usage consts** combined
   (`ticketNewUsage`/`ticketMoveUsage`/`ticketSetUsage`; `boardSyncUsage`/`boardStateUsage`/
   `boardDecisionsUsage`/`boardMetricsUsage`, plus one new `boardAuditUsage` const added for
   symmetry since `board audit` takes no flags) — all but one already exist in the same file, per
   this ticket's own "likely fix" for `ticket`. The other five groups (`changelog`, `flow`,
   `hooks`, `scaffold`, `project`) have no such per-leaf usage consts to reuse cheaply, so their
   `-h`/`--help` case prints only the same short summary line already shown on empty args —
   factored into a package-level const shared between both call sites in that file (avoids
   duplicating the literal, not a new abstraction). No new leaf-usage consts are invented for
   these five groups; that would be a bigger diff than this bug warrants.
5. **Leaf-level `--help` is untouched** (e.g. `ticket move --help` already prints its own usage
   correctly today, even though it exits `exitUsage` rather than 0) — this ticket's own
   Description scopes the fix to "only the group level."

### Tasks

#### Task 1 — `internal/cli/ticket.go`

In `runTicket`'s switch (~line 26), add a case before `default`:

```go
case "-h", "--help":
	fmt.Println(ticketNewUsage)
	fmt.Println(ticketMoveUsage)
	fmt.Println(ticketSetUsage)
	return exitOK
```

#### Task 2 — `internal/cli/board.go`

Add `const boardAuditUsage = "usage: pickle board audit"` beside the other board usage consts
(~line 53). In `runBoard`'s switch (~line 36), add:

```go
case "-h", "--help":
	fmt.Println(boardAuditUsage)
	fmt.Println(boardSyncUsage)
	fmt.Println(boardStateUsage)
	fmt.Println(boardDecisionsUsage)
	fmt.Println(boardMetricsUsage)
	return exitOK
```

#### Task 3 — `internal/cli/changelog.go`

Factor the literal at the empty-args branch (~line 45, `"usage: pickle changelog <check> ..."`)
into `const changelogGroupUsage = "usage: pickle changelog <check> ..."`, used at both call
sites. In `runChangelog`'s switch (~line 48), add:

```go
case "-h", "--help":
	fmt.Println(changelogGroupUsage)
	fmt.Println(changelogCheckUsage)
	return exitOK
```

#### Task 4 — `internal/cli/flow.go`

Factor the empty-args literal (~line 18, `"usage: pickle flow <show|list>"`) into
`const flowGroupUsage = "usage: pickle flow <show|list>"`, used at both call sites. In
`runFlow`'s switch (~line 21), add:

```go
case "-h", "--help":
	fmt.Println(flowGroupUsage)
	return exitOK
```

#### Task 5 — `internal/cli/hooks.go`

Factor the empty-args literal (~line 18,
`"pickle hooks: expected install, uninstall, status or run"`) into
`const hooksGroupUsage = "pickle hooks: expected install, uninstall, status or run"`, used at
both call sites. In `runHooks`'s switch (~line 21), add:

```go
case "-h", "--help":
	fmt.Println(hooksGroupUsage)
	return exitOK
```

#### Task 6 — `internal/cli/scaffold.go`

Factor the empty-args literal (~line 19, `"pickle scaffold: expected release"`) into
`const scaffoldGroupUsage = "pickle scaffold: expected release"`, used at both call sites. In
`runScaffold`'s switch (~line 22), add:

```go
case "-h", "--help":
	fmt.Println(scaffoldGroupUsage)
	return exitOK
```

#### Task 7 — `internal/cli/project.go`

Factor the empty-args literal (~line 26, `"usage: pickle project <add|list|remove> ..."`) into
`const projectGroupUsage = "usage: pickle project <add|list|remove> ..."`, used at both call
sites. In `runProject`'s switch (~line 29), add:

```go
case "-h", "--help":
	fmt.Println(projectGroupUsage)
	return exitOK
```

### Acceptance test

```
just build
just test
just lint
```

Extend the exit-code table in `internal/cli/cli_test.go` (~line 95-131, the same table already
carrying `"board unknown subcommand"` etc.) with one `exitOK` case per group for each of `-h` and
`--help` (14 new cases), e.g.:

```go
{"ticket help flag", []string{"ticket", "-h"}, exitOK},
{"ticket help flag long", []string{"ticket", "--help"}, exitOK},
{"board help flag", []string{"board", "--help"}, exitOK},
{"changelog help flag", []string{"changelog", "--help"}, exitOK},
{"flow help flag", []string{"flow", "--help"}, exitOK},
{"hooks help flag", []string{"hooks", "--help"}, exitOK},
{"scaffold help flag", []string{"scaffold", "--help"}, exitOK},
{"project help flag", []string{"project", "--help"}, exitOK},
```

Add one `captureStdout`-based test (pattern from `board_decisions_test.go`) confirming the
content actually reaches stdout and names the subcommands, e.g.
`TestTicketHelpPrintsCombinedUsage`: `Run(nil, "test", []string{"ticket", "--help"})` inside
`captureStdout`, assert the output contains `"ticket new"`, `"ticket move"` and `"ticket set"`.

Manual smoke (mirrors the dogfooding repro in the Description):

```
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree
./pickle-test ticket --help; echo "exit=$?"      # expect: 3 usage lines, exit=0
./pickle-test board --help;  echo "exit=$?"      # expect: 5 usage lines, exit=0
./pickle-test ticket bogus;  echo "exit=$?"      # unchanged: unknown subcommand, exit=2
```

### Docs update (mandatory when user-facing)

No README/docs page enumerates per-group `--help` behavior to update — the fix only changes CLI
output text, which is self-documenting. No doc changes needed.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` clean.
2. No docs to update (see above).
3. Write a summary (files touched: the seven `internal/cli/*.go` dispatchers plus
   `cli_test.go`; decisions made: scope, exit code, which groups get combined vs. summary-only
   usage).
4. Suggested commit message: `feat(cli): print group usage on --help instead of erroring (T-132)`.
5. Root-path child (`path = "."`) — tidy WIP commits into atomic ones before presenting.
6. Commit locally on the ticket branch; do not push or open an MR without user approval.
   `pickle ticket move T-132 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-20 — created (TO DO). source: field-use: reproduced twice dogfooding brine in a
  separate project (messgr) on pickle 0.19.0 and 0.20.0; surfaced via a coordinator review of
  that project's Claude Code session transcripts, then confirmed by reading
  `internal/cli/ticket.go`.
- 2026-09-20 — TO DO → READY: plan complete
- 2026-09-20 — READY → IN DEVELOPMENT: picked up
