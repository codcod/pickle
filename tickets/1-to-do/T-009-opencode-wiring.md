---
id: T-009
title: opencode wiring
project: pickle
depends-on: [T-004]
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-009 — opencode wiring

## Description

> **PARKED (triage 2026-07-26).** Real, but explicitly unscheduled: nothing is blocked on
> this and no user has asked for it. Do not pick it up without a demand signal. Unparking is a
> user decision — note it in History.

Extend `install`/`doctor`/`upgrade`/`uninstall` to wire the flow for **opencode**. Confirm
opencode's current skill/instruction mechanism: at minimum inject the `AGENTS.md` marker block
(opencode reads `AGENTS.md`), and add a native skill hook if one exists. Include opencode in the
`--agent` set and in auto-detection. Needs T-004. Phase P4.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P4)
- 2026-07-26 — parked (stays in TO DO, unscheduled). source: board triage — backlog growth analysis
