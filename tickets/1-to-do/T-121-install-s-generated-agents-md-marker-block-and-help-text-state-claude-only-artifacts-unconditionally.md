---
id: T-121
title: install's generated AGENTS.md marker block and help text state Claude-only artifacts unconditionally
project: pickle
depends-on: []
spawned-by: [T-119]
impact: medium
complexity: low
cost: S
---

# T-121 — install's generated AGENTS.md marker block and help text state Claude-only artifacts unconditionally

## Outcome

After this ships, a project installed without `claude` in its agent set no longer finds its own
generated `AGENTS.md` telling it that "Claude Code sees it via `.claude/skills/brine`" — a
directory that install never created there. `pickle help` stops describing the `CLAUDE.md` marker
as something every install writes.

## Description

T-119 removed the agent-autodetection claim and the unconditional Claude-symlink claim from the
**skill payload**. Two surfaces outside the payload still have the same defect — they state an
artifact that only `--agent claude` produces as though every install produced it. T-119's
confirmed decision 5 explicitly barred it from touching `internal/`, so both were left for this
ticket.

1. **The generated `AGENTS.md` marker block.** `internal/install/install.go` (the `MarkerBlock`
   text, at the line reading ``"  (`resources/review-protocol.md`). Claude Code sees it via
   `.claude/skills/brine`.\n"``) writes that sentence into every project's `AGENTS.md`,
   regardless of the agent set. Reproduced during T-119's review: a throwaway
   `install --agent opencode` produced an `AGENTS.md` carrying the Claude sentence next to no
   `.claude/` directory at all. This is the sharper of the two — it is generated *into the user's
   own repo*, and `pickle upgrade` refreshes it, so it persists.

2. **`pickle help`.** The `install` summary in `internal/cli/cli.go` reads "inject
   AGENTS.md/CLAUDE.md markers". T-013 fixed this line's autodetection half; the
   `AGENTS.md`/`CLAUDE.md` pairing survived and carries the same unconditional reading. Only
   `AGENTS.md` is unconditional — `CLAUDE.md` follows the agent set, and with `--claude-symlink`
   it is a symlink to `AGENTS.md` rather than a marker block at all.

**Note the marker-block constraint.** This repo self-hosts brine, so its own `AGENTS.md` marker
block is maintained **by hand**, mirroring `markerBlock()`, inside the ticket's diff — never by
running `pickle install|upgrade` against this repo from a feature branch (see `AGENTS.md`'s
self-modify policy). Changing the generated text therefore means changing the Go string **and**
hand-mirroring it into this repo's own marker block in the same commit, or `doctor`'s drift check
and the two will disagree.

Worth checking whether the marker block should say anything agent-specific at all, or whether the
sentence is better phrased so it is true for any agent set — that is the design call refinement
should settle, not a mechanical find-and-replace.

Soft coupling to T-119 (which fixed the payload half); nothing here is blocked on it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: review: T-119's review, non-blocking findings F2 and F3
  (disposition: new ticket, batched by theme). The two non-payload surfaces stating Claude-only
  artifacts unconditionally; T-119 decision 5 barred it from touching `internal/`
