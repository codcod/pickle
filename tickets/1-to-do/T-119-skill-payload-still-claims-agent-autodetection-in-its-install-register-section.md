---
id: T-119
title: skill payload still claims agent autodetection in its Install & register section
project: pickle
depends-on: []
spawned-by: [T-013]
impact: low-medium
complexity: low
cost: S
---

# T-119 — skill payload still claims agent autodetection in its Install & register section

## Outcome

After this ships, the brine skill every installed project reads no longer tells the agent that
`pickle install` detects which agents are present. The skill, the binary's `pickle help`, and the
user manual then all say the same thing: the agent set is exactly what `--agent` names, defaulting
to `claude`.

## Description

`skill/SKILL.md`'s *Install & register* section states that `pickle install` "installs this skill
for the **detected agents**". There is **no autodetection** — that was a locked decision of T-009,
and the set is exactly what `--agent` names (default `claude`).

This is the same false sentence T-013 removed from the binary's top-level `pickle help` text
(T-013 item 4, itself a folded T-009 review finding). T-013 fixed the `internal/cli` occurrence
only; this one lives in the **payload**, which is the copy every installed project actually reads,
and which `pickle upgrade` re-installs verbatim.

The claim already contradicts the user manual, which states plainly that there is *no
autodetection* and that the set is exactly what you name. So the tree currently disagrees with
itself across two surfaces out of three.

**Why this is its own ticket rather than a fix inside T-013.** The defect is *pre-existing* — it
was not introduced or falsified by T-013's branch — and the flow's disposition rules bar the
`fixed inline` route for exactly that case ("did this branch break it?", not "is it small?").
T-013's own Task 2 was scoped to one named file.

Scope is one sentence, plus a check for the same claim elsewhere in the payload. Note that
everything under `skill/` is read by projects that are not pickle, so the replacement wording must
stand on its own for a foreign workspace.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-24 — created (TO DO). source: review: T-013's review, non-blocking finding N9
  (disposition: new ticket). The payload half of the autodetection claim T-013 removed from
  `pickle help`; pre-existing, so ineligible for `fixed inline` under rules §5
