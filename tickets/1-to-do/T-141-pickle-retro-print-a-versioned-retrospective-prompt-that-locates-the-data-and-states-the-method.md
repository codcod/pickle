---
id: T-141
title: pickle retro: print a versioned retrospective prompt that locates the data and states the method
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-141 — pickle retro: print a versioned retrospective prompt that locates the data and states the method

## Outcome

After this ships, `pickle retro [question] | claude -p` (or any agent) runs a retrospective over
the project's own agent sessions and ticket record: pickle prints a prompt that already knows
where the project's transcripts, board and previous retro reports are, and states the method —
evaluate open targets, find what the human keeps doing by hand, check the model timeline,
challenge the findings — ending in a dated report under `tickets/retros/`.

## Description

Modelled on `snowball docs-prompt`: the binary prints a prompt; an agent does the work. The prompt
lives in pickle's source, so it is versioned and released with the binary — no skill to install,
nothing added to every session's skill list for a command run once in a while, and it works with
any agent that takes a prompt.

**Overrides NOTES.md § "Rejected outright, so they are not re-proposed"** ("a metrics command, a
retro command, or a dashboard") — by human direction, the third override after T-105 and T-126.
It keeps that entry's reasoning ("let the queries be ad-hoc"): pickle computes nothing; the
agent runs the queries.

**What pickle fills in** (facts it knows or can find cheaply):

- project root, layout, children with their paths, base branches, commands and ticket prefixes;
- the window — since the newest report in `tickets/retros/`, or `--since YYYY-MM-DD`;
- **where the sessions are** — per known agent host, the session directories derived from the
  root and child paths (Claude Code under its config dir's `projects/`, pi under
  `~/.pi/agent/sessions/`), listing those that exist with file counts and date range; `--sessions
  DIR` (repeatable) adds or overrides. pickle only locates directories, never parses transcripts;
  a host it does not know is left to the agent, and the prompt says so;
- the open targets from the newest report, with the paths of their measurement scripts.

**The method the prompt states** (from the 2026-09-29 exploration, see
`tickets/retros/2026-09-29-self-improvement-loop.md`):

1. Evaluate every open target from the previous report first, with its own script; record
   met / missed / inconclusive and apply its decision rule.
2. Harvest the human's messages that are not trigger phrases and group them — each recurring
   one is a missing automation or a flow defect.
3. Build the dominant-model timeline; if it changed since the previous report, review whether the
   payload still fits the models in use (prose for a strong reader, rails in mechanism).
4. Cross-check the ticket record (`board state --json`, `board metrics`, `board decisions`) and
   the project's recorded decisions; name any proposal that re-opens a rejected item.
5. **Validate every metric by hand on a sample before quoting it.** In the exploration, one
   metric took three script versions (6%, 92%, 15%) before it was right.
6. Challenge the findings independently (a sub-agent where the host has one, else a second pass).
7. Write `tickets/retros/YYYY-MM-DD-<slug>.md`: findings, decisions, and for every proposal a
   **target** — claim, unit, metric, baseline measured now, threshold with minimum sample,
   guard, decision rule — with its script saved beside the report.

It never files tickets, moves them or edits the payload; the human decides from the report. An
optional positional `question` replaces the general retrospective with a focused one.

Constraints for refinement:

- **Foreign-workspace test** (pickle's `AGENTS.md`): the prompt runs in other projects, so
  `payload_lint_test.go` must cover it wherever it lives.
- **Privacy**: reports are committed, maybe to a public repository — the prompt forbids quoting
  the human's messages beyond short fragments and naming repositories the report's own repo
  does not already name.
- **Headless permissions**: `| claude -p` may be refused reads outside the project directory;
  the printed prompt's preamble says how to run it interactively instead.
- Stable-surface addition under the 1.0 promise (new command) → minor release; planned for 1.1.0
  with T-140.

**Pre-registered criterion (T-126 style):** if three months after this ships no ticket filing,
drop or re-grade cites a retro report, remove the command rather than extending it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: chat: self-improvement loop exploration 2026-09-29 — the user asked for a snowball docs-prompt-style retro command, overriding the rejected-outright retro command by direction
