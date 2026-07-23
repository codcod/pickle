---
id: T-009
title: opencode wiring
project: pickle
depends-on: [T-004]
impact: medium
complexity: medium
cost: M
---

# T-009 — opencode wiring

## Description

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
