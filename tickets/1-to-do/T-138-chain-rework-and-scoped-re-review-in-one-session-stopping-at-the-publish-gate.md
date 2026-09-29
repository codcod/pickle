---
id: T-138
title: chain rework and scoped re-review in one session, stopping at the publish gate
project: pickle
depends-on: [T-137]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-138 — chain rework and scoped re-review in one session, stopping at the publish gate

## Outcome

After this ships, "rework ticket T-NNN" carries on through an independent scoped re-review in
the same session and stops at the publish gate, and "implement ticket T-NNN" can hand straight
to an independent review — instead of the human opening a fresh session to type the next
trigger after every phase.

## Description

Evidence, 2026-09-21..29: about 16 rework → validate cycles where the human's only input was the
trigger phrase in a fresh session — SMP-015 went validate → rework → validate → rework → validate
across five sessions in 25 minutes; the same for SMP-007/009/016/017, POR-008/010/018,
T-057/058/060 and two unity tickets. Implement → `/clear` → validate shows the same pattern.

The fresh session exists for reviewer independence, and `review-protocol.md` step 0 already
accepts a delegated reviewer sub-agent with its own context as meeting that bar. So:

- After a rework fix record, the rework session spawns an independent reviewer for the scoped
  re-review. New blocking findings start another rework round, capped (two rounds per
  invocation is the starting proposal); then it stops and hands back to the human.
- After implement, the session may hand to an independent reviewer the same way, then continue
  into the bounded rework loop.
- Human stops that remain: READY approval, publish approval, merge. A host without sub-agents
  keeps today's behaviour (stop, ask for a fresh session).

Two rules make the delegated reviewer as independent as a fresh session and safe in-tree:

1. **The spawn prompt is the trigger phrase and the ticket id, nothing else.** A fresh session
   receives only "validate ticket T-NNN"; an implementer writing "confirm F1 is fixed" steers the
   reviewer toward the implementer's own framing.
2. **The reviewer returns findings; the parent makes every ticket move.** The review's
   bookkeeping is committed on the base branch while the parent sits on the feature branch in the
   same working tree (under `in-tree`); a sub-agent switching branches under the parent is a
   hazard. The parent does the moves exactly as a validate session does today.

This removes a human round trip, not a gate: NOTES.md § "Rejected outright, so they are not
re-proposed" rules out ordering, ranking, scoring or gating of *tickets*, which this does not
touch. Depends on T-137: the re-review runs its hunt.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: ~16 rework/validate cycles where the human only typed the next trigger in a fresh session
