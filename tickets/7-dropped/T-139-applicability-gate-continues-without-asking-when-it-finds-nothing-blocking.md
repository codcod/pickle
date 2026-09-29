---
id: T-139
title: applicability gate continues without asking when it finds nothing blocking
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-139 — applicability gate continues without asking when it finds nothing blocking

## Outcome

After this ships, picking up a ticket runs the applicability check and goes straight on to
implementation when it finds nothing blocking, recording any non-blocking findings in the
ticket; it stops for the human only when a finding is blocking or would change a confirmed
design decision.

## Description

Evidence, 2026-09-21..29: 44 applicability-gate sub-agent runs, 50–100k tokens and 1–4 minutes
each. About 4–5 returned a blocking finding — the gate now earns its cost, which revises the "0
negative verdicts in ~15 runs" recorded in NOTES.md § Model-tier exploration. But nearly every
run ended in "I need your OK on the routing", and the user answered "go", "approved" or "1".

Change to the implement procedure's gate step: no findings, or only non-blocking findings routed
by the default disposition (note-and-close), → record them and proceed. A blocking finding, a
proposed drop, or an amendment that touches a confirmed design decision → stop and ask, as
today. An amendment within the plan's decisions is recorded with the usual `plan amended inline`
History line and does not stop.

Accepted risk: the gate's own blocking/non-blocking call goes unsupervised. In the week reviewed
the user never overrode it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: the applicability gate asked for approval after nearly every run, including clean ones, and was always waved through
- 2026-09-29 — TO DO → DROPPED: baseline measured before refinement: only 7 of 42 clean gate runs (17%) stopped for the human in 2026-09-21..29, 4 with a bare go; the premise was wrong — see tickets/retros/2026-09-29-self-improvement-loop.md
