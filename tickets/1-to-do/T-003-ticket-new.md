---
id: T-003
title: ticket new (id allocation + template + board row)
project: pickle
depends-on: [T-001]
impact: high
complexity: medium
cost: M
---

# T-003 — ticket new (id allocation + template + board row)

## Description

Implement `pickle ticket new "<title>" --project <name> [--impact .. --complexity .. --cost ..]`.

Behaviour:

- allocate the next `T-NNN` = `max(existing across all status dirs) + 1` (one global namespace);
- instantiate the embedded `skill/resources/TEMPLATE.md` into `tickets/1-to-do/T-NNN-<slug>.md`
  with `id`, `title`, and `project:` set, and the grade fields filled (accept adjacent-pair
  ranges; default to sensible ranges when a flag is omitted);
- add the board row under that child's `### <child>` sub-group in the TO DO section, in impact
  order;
- write the first `created (TO DO)` History line.

Fail clearly if `--project` is not a registered child (needs T-001). The CLI guarantees id +
target + placement + board sync; the agent fills the Description prose afterwards. Phase P1.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1)
