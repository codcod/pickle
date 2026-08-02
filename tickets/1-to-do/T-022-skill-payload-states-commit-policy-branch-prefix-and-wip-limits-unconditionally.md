---
id: T-022
title: skill payload states commit policy, branch prefix and WIP limits unconditionally
project: pickle
depends-on: []
spawned-by: []
impact: medium-high
complexity: low
cost: S
---

# T-022 — skill payload states commit policy, branch prefix and WIP limits unconditionally

## Description

Non-blocking finding 8 from the T-018 review.

The embedded skill — which ships into **every** installed project — states three things
unconditionally that are in fact per-project configuration, and now sits directly beside an
`AGENTS.md` marker block that renders the project's *real* values. When a project is configured
non-default the two surfaces contradict each other, and both read as authoritative.

1. **Commit policy**, stated as an absolute in five places: `skill/SKILL.md:3` (the frontmatter
   description every agent loader reads — "pushing a child-project requires explicit user
   approval"), `skill/SKILL.md:160-161`, `:192-193`, `skill/resources/review-protocol.md:154-155`,
   and `skill/resources/TEMPLATE.md:42-43`. A project with `child_publish_gated = false` gets a
   marker block saying push freely and a skill saying never push.
2. **Branch prefix** — `feat/` hardcoded in the *rules* prose: `skill/SKILL.md:77`, `:155`, `:171`;
   `skill/resources/tickets-README.md:44`, `:62`, `:144`, `:213`;
   `skill/resources/review-protocol.md:27`; `skill/resources/TEMPLATE.md:38`.
   (`skill/SKILL.md:62-70` sits under a `Defaults:` heading and is fine as-is.)
3. **WIP limits** — `skill/SKILL.md:90-91` states `≤ 1` as a rule, with no "tune per project"
   escape.

`skill/resources/tickets-README.md:148` already models the correct hedge — *"The project's
`AGENTS.md` / `pickle.toml` states specifics."* — and `:193` hedges WIP with *"(tune per
project)"*. The work is to apply that deferral consistently to the rules/procedure occurrences
**without** turning every sentence into a caveat: the skill should state the flow's defaults once
and point at the marker block for the project's actual values.

Note this is a payload change: `pickle upgrade` propagates it into every installed project, and in
this repo `.agents/skills/ticket-flow/` is a symlink to `skill/`, so editing it here changes what
the binary ships.

Soft coupling: **T-018** (made the marker block render real values, creating the contradiction) and
**T-016** (docs-readability review step — same prose surface).

### Adjacent payload-text items to sweep while in these files (not this ticket's theme)

**Item A — the §8 "freshness" heading.** A separate §8 wording defect was measured on 2026-08-01 (T-064, dropped; full evidence in
`NOTES.md`). It is **not** an instance of this ticket's "unconditional statement that should
defer to config" problem, so it does not belong in the title — but it edits the same two files
and would cost minutes alongside the work above:

- `tickets-README.md:324-333` heads the pickup applicability gate **"a freshness check"** and
  justifies it purely by aging, while its actual mandate is *"the ticket's own assumptions plus
  the board delta"* tested for *"true, **required**, and **worth it**"* (`SKILL.md:177`).
  Practice has followed the heading, not the mandate: **0 negative verdicts in ~15 recorded
  runs**. Two sentences fix it — say the scope includes assumptions **that were wrong at
  filing**, and that **DROP is a legal verdict** (already legal per `move.go:31-38`; verify, add
  no mechanism).

**Item B — a suggested model tier per flow step (Tier 0 of the 2026-08-03 exploration; full
analysis in `NOTES.md`).** The payload is silent on which model tier a step wants, so every step
runs on whatever the session happens to be. Add **one advisory block** — most naturally in
`SKILL.md` beside the existing `Defaults:` material, which is the same "state the default, defer
the specifics" move this ticket applies elsewhere:

- *audit the board* and every `pickle ticket|board` mechanic need **no model** (`board audit` is
  pure mechanics) — worth saying, because the cheapest tier is sometimes zero;
- *refine*, *review* and the pickup applicability gate are the judgement steps;
- *implement* is the token mass and is designed to be executable (the plan is the prompt);
- the applicability gate wants a **different** model, not merely a better one — its own text
  already demands freedom from the implementer's sunk-cost bias (`SKILL.md:173`), and the measured
  record is 0 negative verdicts in ~15 same-agent runs.

Hard constraints, or this becomes the machinery the theme attracts: **advisory only** (pickle has
no runtime role in a step and cannot enforce a tier), **no `pickle.toml` schema**, **no per-agent
scaffolds**, and no claim that the flow depends on it. Anything beyond a prose block is Tier 1/2
in `NOTES.md` and needs a demand signal plus a pre-registered kill criterion.

If either item is picked up here, widen the title accordingly; if one is judged out of scope,
leave it in `NOTES.md` rather than filing it — the standing lesson is that this theme attracts
machinery.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-08-03 — re-graded medium → **medium-high** by the second impact recalibration pass: same
  defect class as T-041 (which was graded `high` for it) — the payload contradicts the marker
  block's real values, so agents act on wrong project config — one notch down because it bites
  only non-default configurations. Now the top of the TO DO group at complexity low / cost S.
- 2026-08-03 — adjacent item B added (Tier 0 of the model-tier exploration, `NOTES.md`): an
  advisory suggested-tier-per-step block in the payload, constrained to prose — no schema, no
  per-agent scaffolds, no enforcement. Filed here rather than as its own ticket because it fails
  the promotion test alone and edits files this ticket already opens.
