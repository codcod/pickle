---
id: T-111
title: document the 'cut a release' convention in the user manual: RELEASING.md + CHANGELOG.md as the agent's inputs
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-111 — document the 'cut a release' convention in the user manual: RELEASING.md + CHANGELOG.md as the agent's inputs

## Outcome

A reader of the user manual learns that "cut a release" is a workflow their *own* project
equips, by providing two files — `CHANGELOG.md` (already read mechanically by
`pickle changelog check`) and a `RELEASING.md` the agent follows as prose — and learns exactly
what an agent does when either file is absent: it reports what it can and offers to scaffold,
never inferring a release procedure and running it. Today the manual mentions `RELEASING.md`
twice but only as *pickle's own* maintainer file, so a user has no way to know the convention
exists or applies to them.

## Description

**The gap.** `docs/user-manual/installation.adoc` refers to `RELEASING.md` twice (lines ~30 and
~51), both times as "the repository's `RELEASING.md`" — guidance for someone releasing *pickle
itself* from source. Nothing in the manual tells a user operating brine in their own project
that a `RELEASING.md` + `CHANGELOG.md` pair is what turns "cut a release" from an unsupported
phrase into a procedure an agent can follow. `pickle changelog check` is documented thoroughly
in `cli-reference.adoc` (`#cmd-changelog-check`) but only as a command, never as the
machine-checkable half of a release workflow.

**The convention is already validated by this repo's own practice.** This repo's root
`RELEASING.md` opens by running `pickle changelog check`, then does the changelog surgery
(retitle `[Unreleased]` to `[X.Y.Z] - YYYY-MM-DD`, add a fresh empty `[Unreleased]`, update the
link refs), then tags. That is precisely the shape being proposed for general guidance — so
this ticket generalises a practice already in daily use here rather than inventing one.

**Design, and what it deliberately excludes.** The proposal was challenged before filing, and
three earlier variants were rejected:

- *A `release` command key in `pickle.toml`*, sibling to `build`/`test`/`lint`/`docs`. Rejected:
  those four are **verification** commands — idempotent, safe to re-run, and the agent is
  expected to fire them unprompted at a gate. A release is mutating, irreversible and
  publishing. Placing it in that list lends it the "safe for the agent to just run" semantics of
  its neighbours. (Confirmed while filing: the binary never executes those four itself — they are
  declarative metadata the agent reads — which makes the borrowed semantics the whole risk.)
  A single shell string also cannot express a multi-step procedure with human checkpoints,
  whereas a prose checklist can. This is why the input is a **document, not configuration**.
- *A `pickle changelog cut <version>` verb.* Rejected for this ticket: correct compare-links need
  the remote URL, forge flavour and tag-prefix convention, which is I/O and forge inference —
  against `internal/changelog`'s stated pure text-in/text-out contract — and `git-cliff` /
  `changie` already own that space.
- *A trigger whose final step executes the release.* Rejected: it collides with the publish gate
  (pushes need explicit approval; merging is always the human's). Any trigger must terminate in a
  report plus handoff, with no path from a phrase to a push.

**Proposed shape** (to be pinned at refinement): a new concepts page
`docs/user-manual/concepts/releasing.adoc`, included from `docs/user-manual.adoc` after
`agent-session-workflow.adoc` — that page is the register precedent, documenting a pattern the
agent follows while stating plainly that no command enforces it. Plus two pointers: an xref from
`cli-reference.adoc`'s `#cmd-changelog-check` section, and `CHANGELOG.md`/`RELEASING.md` added to
the project-file inventory in `concepts/project-structure.adoc` (~line 233 already alludes to
"changelog and release tooling") as optional and project-authored.

The page should state the fallback ladder, which degrades toward *reporting*, never toward
guessing: `RELEASING.md` missing → run the preflight that is legitimately available
(`changelog check`), report, then offer to **draft** a `RELEASING.md` from observable evidence
(build manifests, CI workflows, existing tag names) for the human to correct, executing nothing
from that draft; `CHANGELOG.md` missing → say plainly that traceability cannot be checked (the
command exits non-zero on an unreadable changelog) and offer a Keep a Changelog skeleton; both
missing → "cut a release" produces a setup step, not a release. It should also list what makes a
`RELEASING.md` useful to an agent: version scheme, where the version number lives, tag
convention (which also determines what to pass to `--section`), the ordered steps, and per step
whether the agent may run it or must hand back.

**Open decisions for refinement.**

1. **Manual-only, or also the brine payload?** If an agent is to act on "cut a release", the
   behaviour arguably belongs in the skill (`skill/resources/`), not only in pickle's own manual
   — but the payload is read by projects that are not pickle, and a release procedure is exactly
   the kind of project-specific tooling the payload avoids assuming. Decide manual-only versus a
   minimal payload trigger; if the payload is touched at all, the foreign-workspace test in
   `AGENTS.md` governs the wording and `payload_lint_test.go` will enforce its mechanical part.
2. **Whether the fallback's "offer to draft a `RELEASING.md`" is documentation only, or implies a
   future `pickle scaffold release` verb.** T-110 shipped `pickle scaffold docs`, so the verb
   group exists; extending it is a separate ticket, not this one, and this page must read
   correctly whether or not that ever lands.
3. **Whether to add the inverse board-vs-range check** (a ticket in `6-done/` absent from the
   commit range, or a ticket still in `3-in-development/`/`4-in-review/` whose commits are in
   it). It is the one release-time question only brine can answer, and it is pure ticket-domain,
   but it is a code change to `changelog check` and belongs in its own ticket if wanted.

**Soft couplings** (no hard `depends-on:`):

- **T-110** (done) — shipped `pickle scaffold docs`; its filing rationale rejected baking in
  AsciiDoc/snowball/goreleaser/GitHub/`just` assumptions, which is the same reasoning that makes
  a project-authored `RELEASING.md` preferable to a pickle-owned release template. Open decision
  2 above is where the two meet.
- **T-093 / T-094 / T-095 / T-097** (done) — the `changelog check` family this page frames. The
  page must not contradict the command's documented advisory contract: it always exits `0` when
  it finds candidates and must never become a CI gate.
- **T-067** (`1-to-do/`) — `docs-check` performs no link/anchor validation, so the new page's
  xrefs (`<<releasing>>`, `<<cmd-changelog-check>>`, `<<agent-session-workflow>>`) will not be
  machine-verified; check them by hand until T-067 lands.
- **T-047** (done) — established the AsciiDoc manual and its `docs-check` pipeline this page
  plugs into.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: chat: a question about what "cut a new release" would
  mean for a newly onboarded project led to a design discussion in which three heavier variants
  (a `release` key in `pickle.toml`, a `pickle changelog cut` verb, and a trigger that executes
  the release) were each challenged and rejected; the user then specified the surviving shape —
  a project-authored `RELEASING.md` + `CHANGELOG.md` the agent reads, with a defined fallback
  when either is absent — and asked for it to be filed as a user-manual change.
