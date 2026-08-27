---
id: T-126
title: Ticket cycle-time / lead-time report mined from each ticket's dated History lines
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-126 — Ticket cycle-time / lead-time report mined from each ticket's dated History lines

## Outcome

After this ships, a question like "how long does a ticket sit in READY before pickup?" or "has
review turnaround gotten worse?" is answered by a `pickle` command reading `## History`, instead
of hand-parsing dated transition lines across the tree.

## Description

Inspired by Anthropic's [*The AI-Native SDLC playbook*](https://claude.com/blog/the-ai-native-sdlc-playbook),
which instruments **every** play with a leading and a lagging indicator, almost always phrased as
"elapsed time between two committed-artifact timestamps, read from git/PR history" — e.g. "time
from the `intent.md` commit to the `spec.md` commit," "share of changes merged from the first
implementation pass," "review time per PR from PR metadata." brine already carries the exact
substrate that pattern needs: every ticket's `## History` is a dated, append-only log of typed
transitions (`TO DO → READY`, `READY → IN DEVELOPMENT`, `IN DEVELOPMENT → IN REVIEW`,
`IN REVIEW → DONE`, `merged to <base> (...)`, plus non-transition lines like
`plan amended inline: ...`) — it just isn't mined today. `rg 'time between|cycle.time|lead.time'
internal/` and the board/audit code turn up nothing: `pickle board audit` checks that History
exists and that the *last* line matches the directory, but nothing aggregates the *deltas*
between lines.

Proposal: a read-only report (e.g. `pickle board metrics` or a mode of the existing
`pickle board state --json` projection from **T-065**, which already parses History into
structured form) that computes, per ticket and aggregated per child-project:

- time in each status (dated transition N → transition N+1);
- TO DO → READY ("refinement time"), READY → IN DEVELOPMENT ("queue time"), IN DEVELOPMENT →
  IN REVIEW ("build time"), IN REVIEW → DONE ("review time"), and DONE → `merged to <base>`
  where that line exists;
- an aggregate (median/mean, count) per child-project and per transition kind, so drift is
  visible without re-deriving it by hand every time someone asks.

This is deliberately a **different** metric surface from **T-085** (done) — T-085 made findings
*classifiable* (defect class, disposition, cost actual/estimate, provenance class); this ticket
makes the *timeline* between statuses countable. Both read the same per-ticket record; neither
supersedes the other. Soft coupling to **T-065** (JSON projection) as the natural data source —
this ticket can consume it rather than re-parsing History itself, but does not hard-depend on any
further T-065 work, since T-065 already shipped History parsing.

No existing ticket covers this — checked `tickets/` for `elapsed`, `lead time`, `cycle time`,
`time between`, `time-in-status` and `metrics` before filing; only T-085 (different metric
surface, see above) and T-065 (the data source) matched.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-27 — created (TO DO). source: chat: filed while comparing `rick` and `pickle` against
  Anthropic's *The AI-Native SDLC playbook*
  (https://claude.com/blog/the-ai-native-sdlc-playbook); its per-play leading/lagging-indicator
  discipline maps directly onto brine's own dated `## History` lines, which nothing aggregates yet.
