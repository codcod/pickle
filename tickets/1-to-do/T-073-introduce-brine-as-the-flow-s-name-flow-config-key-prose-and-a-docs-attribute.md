---
id: T-073
title: introduce brine as the flow's name: flow config key, prose, and a docs attribute
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: low
cost: M
---

# T-073 — introduce brine as the flow's name: flow config key, prose, and a docs attribute

## Description

The flow `pickle` installs has no name. It is described everywhere by what it is — "the
ticket flow", "the ticket-based, board-driven feature flow" — which was fine while it was
the only one, and is the thing blocking every conversation about there being more than one.
You cannot make something "the default" until it has a proper noun.

This ticket gives it one: **brine** — the medium things pickle in. It is deliberately the
*cheap half* of naming:

- a new optional `flow` key in `pickle.toml`, defaulting to `"brine"`, and a `pickle flow
  show` that prints it (with `pickle flow list` printing exactly one entry). The key earns
  its keep as the seam a second flow would later select, and as something `doctor` and
  `board audit` can report;
- the flow renamed **in prose**: the body of `skill/SKILL.md`, `resources/tickets-README.md`,
  `resources/review-protocol.md`, the `AGENTS.md`/`CLAUDE.md` marker block rendered by
  `install.go:markerBlock()` (and this repo's own marker, hand-edited inside this ticket's
  diff per the self-modify policy in `AGENTS.md`);
- the docs. Smaller than it looks: `docs/attributes.adoc` already defines
  `:skill-dir: .agents/skills/ticket-flow`, so the work is adding a `:flow: brine` attribute
  and routing the remaining literal occurrences across the six `.adoc` files through the
  attributes rather than hand-editing each.

**Nothing on disk moves.** The installed skill directory, the `.claude/skills/` symlink and
the `SKILL.md` frontmatter `name:` all keep saying `ticket-flow`. That is a deliberate
split, not an oversight: renaming paths is a migration problem in every project that has
already run `pickle install`, and it is isolated in T-074 so it can be scheduled — or
dropped — without holding this ticket hostage. The transitional wording is honest: *the
`ticket-flow` skill operates the `brine` flow*.

The standing argument against ever doing T-074 belongs here, because refinement of this
ticket is where it gets settled: `ticket-flow` is self-documenting and `brine` is opaque, and
if a catalogue of flows ever exists, self-documenting directory ids get **more** valuable,
not less. The recommendation on the table is that `brine` is the flow's proper name in
config, prose and docs, while on-disk ids stay descriptive.

Soft couplings, no hard dependency on any: T-080 (lifecycle as data) is the other half of
the seam this key selects; T-046 (make `doctor`/`upgrade` self-host-aware) touches the same
skill-symlink detection T-074 would; T-066 (close the CLI-surface documentation gaps) will
need to document `pickle flow` in `cli-reference.adoc` if it lands after this.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
