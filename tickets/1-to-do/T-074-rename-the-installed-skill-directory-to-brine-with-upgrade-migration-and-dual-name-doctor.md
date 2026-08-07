---
id: T-074
title: rename the installed skill directory to brine, with upgrade migration and dual-name doctor
project: pickle
depends-on: [T-073]
spawned-by: []
impact: medium
complexity: high
cost: L
---

# T-074 — rename the installed skill directory to brine, with upgrade migration and dual-name doctor

## Description

The expensive half of the rename T-073 starts: moving the name from prose onto disk, so the
installed skill is `brine` rather than `ticket-flow` everywhere an agent or a human can see
it.

**This ticket is a migration, not a `sed`.** The occurrences that matter are not the ~85 in
this repo — they are the artifacts already written into other people's projects by a prior
`pickle install`:

- `.agents/skills/ticket-flow/` — the directory itself, whose `SKILL.md` resolves its own
  `resources/…` references relative to it;
- `.claude/skills/ticket-flow` — the symlink Claude Code reads;
- `SKILL.md` frontmatter `name: ticket-flow`, which *is* the agent-visible skill name;
- the `AGENTS.md`/`CLAUDE.md` marker-block body;
- `agents/opencode/opencode.jsonc` and both `docs-readability.ts` extensions;
- this repo's own `.agents/skills/ticket-flow → skill/` symlink and the hardcoded paths in
  its `AGENTS.md`, which must be changed **by hand inside this ticket's diff** — the
  self-modify policy forbids running `pickle install|upgrade` against this repo from a
  feature branch.

So the work is: `upgrade` detects an old-named installation, moves the directory, re-points
the symlink and rewrites the marker; `doctor` recognises **both** names for a deprecation
window and says which one it found; `uninstall` removes either. Users who never upgrade must
not break.

**This ticket is a candidate to drop, and that decision belongs at refinement.** The
argument for it is consistency: a flow named `brine` whose directory says `ticket-flow` is a
seam users will trip over, and pre-1.0 (tags stop at `v0.2.2`, and `CHANGELOG.md` explicitly
permits breaking changes below 1.0) is the cheapest this will ever be. The argument against
is that `ticket-flow` is self-documenting where `brine` is opaque, and that a future
catalogue of flows would want `brine` sitting beside descriptive sibling ids — one cute name
next to technical ones is a worse taxonomy than all-descriptive. Refinement must pick a rule
and record it.

Soft coupling: T-046 (make `doctor` and `upgrade` self-host-aware — skill-symlink detection,
payload-version noise) works on exactly the detection code this ticket changes; whichever
lands second should expect to rebase onto the other. Sequencing them deliberately is
cheaper than merging them.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
