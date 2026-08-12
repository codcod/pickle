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

## Outcome

After this ships, a project that runs `pickle upgrade` gets its installed skill migrated from `.agents/skills/ticket-flow` to `.agents/skills/brine` (the Claude Code symlink, the skill's own name and the marker-block prose along with it), with `doctor` still recognising the old name during the transition.

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
- 2026-08-07 — patched by T-073's review impact sweep (step 8). T-073 shipped, so the prose half
  of the rename is **done**: the `flow` config key (default `"brine"`), `pickle flow show|list`,
  a `doctor` passed-line, and the name in `SKILL.md`, `tickets-README.md`, `MarkerBlock()`, this
  repo's own `AGENTS.md`/`pickle.toml`/`README.md`, and the manual (via a `:flow:` attribute).
  Two consequences for this ticket: (1) the in-repo occurrence count in the Description is now
  lower than the "~85" it cites — re-count at refinement rather than trusting it; (2) T-073's
  review folded finding **F5** in here — `agents/opencode/opencode.jsonc:3,4,42`,
  `agents/pi/extensions/*.ts`, `.pi/extensions/docs-readability.ts:49` and the Go package
  comments in `main.go:2`, `assets.go:7`, `internal/install/install.go:1` still say "ticket
  flow"/"ticket-flow skill". They were deliberately left alone because `agents/**` is
  byte-compared by `doctor`, so renaming them without this ticket's upgrade path would raise a
  drift warning in every already-installed project. This ticket already lists those scaffolds in
  its scope; the Go comments are the new part
- 2026-08-12 — patched by T-096's review impact sweep. T-096 repinned the docs-readability
  model and genericised the surrounding prose in both `docs-readability.ts` copies and both
  `opencode.jsonc` files — all four of which this ticket lists as rename targets. Nothing here
  is invalidated (T-096 touched the *model* and the word "Gemini", never the words "ticket
  flow"/"ticket-flow skill" this ticket renames), but the folded-F5 line references above have
  drifted: the `ticket-flow skill` comment cited at `.pi/extensions/docs-readability.ts:49` is
  now at `:50`, since T-096 added a line to that doc comment. Search the text rather than the
  line numbers at refinement — as the note above already says for the "~85" occurrence count
