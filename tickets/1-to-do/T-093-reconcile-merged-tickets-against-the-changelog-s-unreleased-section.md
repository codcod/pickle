---
id: T-093
title: reconcile merged tickets against the changelog's Unreleased section
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: medium
cost: S-M
---

# T-093 — reconcile merged tickets against the changelog's Unreleased section

## Outcome

After this ships, preparing a release starts from a **mechanical reconciliation** instead of a
from-scratch reading of the log: every ticket merged since the last tag either has a
`CHANGELOG.md` entry naming it, or a recorded decision in its own file saying it deliberately has
none — and anything else is reported. Choosing the version number, tagging, and verifying release
artifacts stay exactly where they are today: a human, `RELEASING.md`, and CI.

## Description

**This ticket was filed much larger and has been cut down.** The original filing proposed a whole
release procedure for the skill — changelog sweep, SemVer proposal, approval-gated tag, artifact
verification, and a new optional `[release]` block in `pickle.toml` — generalised from a single
session that cut `v0.5.0` by hand. The filing session then challenged it, and three of those four
parts did not survive (reasons recorded below, so refinement does not re-add them). What remains
is the one part **pickle is uniquely positioned to do**, because it is the only thing in the room
that knows what merged: reconciling merged tickets against the changelog.

**The check.** For every ticket whose History records a merge dated after the last release tag,
assert that `CHANGELOG.md` accounts for it — either an entry naming it, or a recorded, confirmed
decision in the ticket itself that it deliberately gets no entry. Report anything else.

**The unsolved design problem, named honestly, because it is the whole difficulty of this
ticket:** the join between a ticket and a changelog entry is prose on both sides, and is not
currently reliable in either direction.

- Measured at filing: `CHANGELOG.md` has **28** bullets carrying a `(T-NNN)` reference and **20**
  top-level bullets carrying none. Some of that is the pre-ticket `v0.1.0` era, but not all.
- The case that most needs judgement is the one a text match handles worst: **T-090** legitimately
  has no changelog entry, because its own Implementation Plan recorded a confirmed decision to
  fold silently into T-089's not-yet-released entry (the code it hardened had never shipped). A
  sweep that cannot read that decision reports a false positive on precisely the ticket where a
  human already thought carefully.

So the real work here is deciding **what the join key should be** — formalise `(T-NNN)` in entries
and audit for it, add a frontmatter field, require an explicit `changelog: none` decision marker,
or something else — and only then writing the check. A refinement that skips straight to
implementing a grep will produce a noisy check that gets ignored, which is worse than no check.

**REFINEMENT GATE — answer before this can go READY: build vs. buy.** `release-please`,
`git-cliff`, `changesets` and `semantic-release` all generate changelogs and versions from history,
and none of them were considered at filing. Refinement must state why pickle should grow its own
rather than adopt one, or **drop this ticket**. The honest argument for building: those tools
reconcile against *commits*, and none of them know what a ticket is, so none can check "this
merged ticket has no entry, and no recorded decision saying it shouldn't." The honest argument
against: that is a narrow benefit for a project whose value proposition is ticket flow, not
release automation. Dropping is a legitimate outcome of refining this ticket.

**Dropped from the original filing, with reasons:**

- *A SemVer bump proposed from changelog categories* (`Added` → minor, `Fixed`/`Changed` → patch,
  incompatible → major, with a pre-1.0 exception read from the changelog header). **The rule is
  unsound, and the release that inspired it proves it.** T-081's entries sit under `Added` and
  `Changed`, so the rule says *minor* — but T-081 introduced a real migration hazard (a
  pre-existing ticket missing a now-required section makes `pickle upgrade`/`install` fail), which
  post-`1.0.0` is a **major** by any honest reading. `v0.5.0` was correct only because the pre-1.0
  exception masked the error. Category-counting cannot see breakingness; encoding it would produce
  a confident wrong answer. If a version proposal is ever wanted, it should surface evidence and
  refuse to name a number.
- *Release-artifact verification* (all archives, checksums, the manual, the Homebrew tap commit).
  If a release missing the user manual is invalid — and T-086/T-087 establish that it is — then
  `release.yml` should **fail** on it. That is a small CI ticket, and once it exists an agent-side
  re-check is dead weight. Belongs in CI, not in the skill.
- *A new optional `[release]` block in `pickle.toml`* (`changelog`, `tag_prefix`, `release_doc`).
  Generalised from N=1: it encodes pickle's own setup (Keep a Changelog, SemVer, tag-triggered CI)
  as though it were the shape of every child-project. Premature; add config when a second
  project's needs actually disagree.
- *Bundling a release procedure into the shipped skill payload.* The original filing rejected a
  standalone `release-flow` skill on install-plumbing cost — an argument about implementation
  convenience, not conceptual fit — and never asked whether a release procedure should ship to
  every installed project at all. With the scope cut to a reconciliation check, the question is
  moot for now.

Soft coupling: **T-092** (detect an unfinalized merge) makes the `merged to <base>` History lines
this check reads actually reliable — a done ticket silently missing its line is invisible to this
sweep. Not a hard dependency, but this check is worth more after T-092 lands.

Soft coupling: **T-065** (JSON read projection) would give this a structured source for "tickets
merged since the last tag" instead of grepping `## History` across `6-done/`. Nice-to-have.

Soft coupling: **T-086**/**T-087** (release-CI hardening, both done) are where the dropped
artifact-verification work belongs if anyone wants it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — **cut down and retitled** (was: *a release procedure for the ticket-flow skill:
  changelog completeness, SemVer proposal, approval-gated tag*) after the filing session
  challenged its own two new tickets. Three of the four proposed parts were dropped with reasons
  recorded in the Description — the SemVer heuristic (demonstrably wrong on T-081, the very change
  that inspired it), artifact verification (belongs in `release.yml`, not an agent), and the
  `[release]` config block (generalised from N=1) — leaving only the merged-ticket ↔ changelog
  reconciliation, the one part that needs to know what a ticket is. A **build-vs-buy refinement
  gate** was added: `release-please`/`git-cliff`/`changesets`/`semantic-release` were never
  considered at filing, and dropping this ticket is now an explicitly legitimate outcome of
  refining it. Regraded `medium`/`medium`/`M-L` → `low-medium`/`medium`/`S-M`: much smaller scope,
  but complexity holds at `medium` because the ticket↔entry join key is genuinely unsolved (28
  changelog bullets carry `(T-NNN)`, 20 carry none, and T-090's legitimate no-entry decision is
  exactly the case a text match handles worst)
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
