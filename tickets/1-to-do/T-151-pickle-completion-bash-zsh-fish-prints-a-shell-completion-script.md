---
id: T-151
title: pickle completion bash|zsh|fish prints a shell completion script
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-151 — pickle completion bash|zsh|fish prints a shell completion script

## Outcome

`pickle completion zsh` (and `bash`, `fish`) prints a completion script, so a user can
tab-complete pickle's subcommands and flags after one line in their shell rc.

## Description

pickle has no shell completion. The CLI is hand-dispatched (stdlib `flag`, no cobra), so
completion means emitting a static script per shell, listing subcommands, sub-subcommands and
their flags. Status names for `ticket move` and child names for `--project` are good targets for
dynamic completion via `pickle project list`. Ticket-id completion is out of scope until someone
asks for it.

Keep the script's command list from drifting: a test asserts that every subcommand in `pickle help`
appears in each script. If the release packaging (Homebrew formula, .deb) installs completion files,
wire them in there; otherwise document the rc-file line in the user manual.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-07 — created (TO DO). source: chat: user noted pickle generates no shell completion
