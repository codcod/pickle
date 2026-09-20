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
lines — instead of failing with `pickle ticket: unknown subcommand "--help"`. An agent unsure of
`ticket` subcommand syntax gets an answer instead of an error.

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
`ticketSetUsage` consts already defined in the same file). Worth checking whether the top-level
`pickle --help` and other command groups (`board`, etc.) have the same gap while in there.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-20 — created (TO DO). source: field-use: reproduced twice dogfooding brine in a
  separate project (messgr) on pickle 0.19.0 and 0.20.0; surfaced via a coordinator review of
  that project's Claude Code session transcripts, then confirmed by reading
  `internal/cli/ticket.go`.
