---
id: T-093
title: a release procedure for the ticket-flow skill: changelog completeness, SemVer proposal, approval-gated tag
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M-L
---

# T-093 — a release procedure for the ticket-flow skill: changelog completeness, SemVer proposal, approval-gated tag

## Outcome

After this ships, "release a new version" (or "cut a release") is a skill-driven procedure: an
agent sweeps merged tickets against `CHANGELOG.md`'s `[Unreleased]` section for coverage gaps,
proposes a SemVer bump with stated reasoning (asking rather than assuming on anything ambiguous),
drafts the changelog retitle, and — only after explicit approval — tags and pushes, then verifies
the triggered release's artifacts by name (including the user manual, twice missed silently by
this project before, T-086/T-087) instead of trusting a green workflow run alone.

## Description

`RELEASING.md` documents pickle's own release mechanics (update `CHANGELOG.md`, tag, push —
everything else is tag-driven CI), but nothing in the ticket-flow skill knows how to *decide* what
to release or *when* the changelog is actually ready. That decision — is `[Unreleased]` complete,
what should the next version number be, is now the right time — was made by hand, from scratch,
in the session that filed this ticket, cutting `v0.5.0` (see `tickets/6-done/T-081-…md` and this
repo's own `CHANGELOG.md` for the worked example). That session's steps are this ticket's starting
spec, not a hypothetical:

1. Read the last tag and `[Unreleased]`.
2. **Cross-check every ticket merged since the last tag against the changelog** — in that session,
   T-080/T-081/T-089 all had entries already; **T-090 deliberately did not**, because its own
   Implementation Plan recorded a confirmed decision ("no CHANGELOG.md entry" — verified `met` at
   its review) to fold silently into T-089's not-yet-released entry, since the code it hardened had
   never shipped. The sweep found nothing wrong that time, but a session doing this by memory
   *would* eventually miss a real gap; the point of this ticket is that the sweep itself is a named,
   repeatable step, and a documented fold-in decision is distinguished from silence rather than
   both looking identical.
3. **Propose the SemVer bump mechanically from the changelog's own categories** — `Added` present
   → minor; `Fixed`/`Changed` only → patch; anything reading as incompatible → major, *unless* the
   changelog's own header states a pre-`1.0.0` exception (this project's does — read that policy
   from the file's prose, never hardcode a project's SemVer exception into the skill payload).
   **Ask rather than assume** whenever a signal is genuinely ambiguous.
4. **Draft the changelog edit** by mirroring the file's own existing precedent exactly (retitle
   `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`, open a fresh empty one above it, update the compare-
   link block at the bottom against the pattern already there — never invent a new shape).
5. **Present the version, the reasoning, and the changelog diff for explicit approval before
   touching anything** — a hard publish gate, the same weight as a child's MR approval even though
   no single ticket owns this commit.
6. **On approval:** commit, push the base branch, tag, push the tag, then **watch the triggered
   release workflow to completion** rather than fire-and-forget.
7. **Verify release artifacts by name**, not just "the workflow is green": every platform archive,
   `checksums.txt`, the user manual PDF/EPUB (T-086 and T-087 both shipped a release with this
   silently missing, undetected for ten days), and the Homebrew tap's own commit landing.
8. **Report**: version, tag, release URL, per-artifact pass/fail, tap status, anything deferred.

**Shape for the skill payload** (for refinement to confirm, not fixed here):

- A new `resources/release-protocol.md`, mirroring `resources/review-protocol.md`'s own structure
  (numbered steps, a closing checklist), referenced from a new `## Procedure: release a project`
  section in `SKILL.md`'s trigger list ("release a new version", "cut a release", "release
  <project>").
- A new **optional** `[release]` block in `pickle.toml`, absent ⇒ the trigger is inapplicable —
  same optionality pattern the skill already uses for `review_addendum`:
  ```toml
  [release]
  changelog = "CHANGELOG.md"
  tag_prefix = "v"
  release_doc = "RELEASING.md"   # optional pointer to the child's own mechanics
  ```
  `internal/config` gains validation for it (shape only — file existence, not content); `pickle
  doctor` may warn if `changelog` or `release_doc` names a path that does not exist. Whether this
  needs any new CLI surface at all, or is purely a skill-payload + config-validation change, is a
  refinement question — the completeness sweep and version proposal in particular could ship as
  pure agent procedure over `git log`/`gh` first, with a mechanical `pickle changelog check`-style
  command deferred to a later ticket if the prose proves unreliable in practice (the same
  prose-then-data path T-081 walked for the READY gate).

**Explicitly rejected as a separate skill** (considered at filing, not deferred): a standalone
`release-flow` skill would need its own install/upgrade/uninstall plumbing, its own payload-version
stamp, and `doctor` awareness of a second skill directory, for a feature that is still "things an
agent does while operating pickle's flow," just at whole-project rather than per-ticket
granularity. `resources/docs-readability.prompt.md` already shows this skill absorbing an adjacent
concern as a bundled resource rather than a new skill; this ticket follows that precedent.

Soft coupling: **T-092** (finalize a merged ticket) is what keeps a ticket's `merged to <base>`
History line trustworthy — this ticket's completeness sweep (step 2) reads exactly that line to
build its "what merged since the last tag" list. Not a hard dependency: the sweep can read whatever
merge lines already exist in the tree today; T-092 just makes future ones more reliable.

Soft coupling: **T-065** (JSON read projection) would give this ticket's completeness sweep a
structured source for "tickets merged since the last tag" instead of grepping `## History` text
across `6-done/`; nice-to-have, not required — this ticket does not block on it.

Soft coupling: **T-086**/**T-087** (release-CI hardening, both done) are the CI-side half of
"don't ship a release with something silently missing"; this ticket is the agent-side half —
verifying by name that what those fixes made *detectable* was actually *checked* before declaring
the release done.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: chat design discussion following the session that cut
  `v0.5.0` by hand (see `tickets/6-done/T-081-…md`'s own History and this repo's `CHANGELOG.md`);
  that session's exact steps are this ticket's starting spec (Description). Graded
  `medium`/`medium`/`M-L` against the backlog: new resource file plus a new optional `pickle.toml`
  block and its validation puts this above T-092 (pure protocol prose, `S`) and above T-065
  (`low-medium`/`medium`/`M`, a read-only projection with no new config surface), but the
  completeness sweep and SemVer proposal can ship as agent procedure over existing primitives
  (`git log`, `gh`) without new CLI plumbing, so it stops short of `L`/`XL`. Impact `medium`: a
  rarely-run but high-stakes action (a public GitHub release + Homebrew tap update) done by hand
  today, that this repo will keep needing for its own self-hosted releases
