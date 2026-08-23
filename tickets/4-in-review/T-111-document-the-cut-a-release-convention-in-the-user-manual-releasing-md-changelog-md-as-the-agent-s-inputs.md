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
- **T-067** (`6-done/`, **merged to `main` 2026-08-22, PR #60** — patched by its review's impact
  sweep) — it added exactly the link/anchor validation this bullet said was missing: a Go test
  wired into both `go test ./...` and `just docs-check` that fails on a dead `<<anchor>>`, an
  inter-document `xref:<file>.adoc` form, or an orphan page. The new page's xrefs
  (`<<releasing>>`, `<<cmd-changelog-check>>`, `<<agent-session-workflow>>`) are therefore
  machine-verified now, and the page must be `include::`-d into `docs/user-manual.adoc` or the
  orphan check fails.
- **T-047** (done) — established the AsciiDoc manual and its `docs-check` pipeline this page
  plugs into.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-111-releasing-convention-docs
```

Root-path child. Tidy WIP commits before presenting (Finish, below).

### Prerequisite gate (hard)

None. T-110, T-093/T-094/T-095/T-097, and T-047 are all `6-done/` (soft couplings only, no
`depends-on:`). T-067 (docs xref/anchor validation) is still `2-ready/`, not done — see the
Acceptance test's manual-xref-check note below, which exists precisely because that ticket
hasn't landed yet.

### Confirmed design decisions (do not deviate without asking)

1. **Open decision 1: manual-only, not the brine payload.** The behaviour is project-specific
   release tooling, exactly the category `skill/resources/` avoids assuming (mirrors T-110's own
   filing rationale for `pickle scaffold docs`, cited in the Description). No payload file is
   touched by this ticket; if a payload-side trigger is ever wanted, it is a new ticket, written
   against `AGENTS.md`'s foreign-workspace test from the start.
2. **Open decision 2: the page documents a pattern, not a promised verb.** "Offer to draft a
   `RELEASING.md`" is written as something an agent following this convention does, in prose —
   no reference to `pickle scaffold release` or any other not-yet-existing command. The page must
   read identically whether or not that verb ever ships.
3. **Open decision 3: the inverse board-vs-range check stays out of scope**, exactly as the
   Description already concludes — it is a code change to `internal/changelog`, not a docs
   change, and belongs in its own ticket if wanted. Not mentioned in the new page as a roadmap
   item (a docs page does not promise future code).
4. **Placement, verified at refinement:** the new page is
   `docs/user-manual/concepts/releasing.adoc`, `include::`-d in `docs/user-manual.adoc`
   immediately after `concepts/agent-session-workflow.adoc` (`:32`) and before the `[part] == CLI
   Reference` line (`:34`) — the register precedent the Description names. Anchor `[#releasing]`,
   so `cli-reference.adoc`'s pointer (Task 3) can cite `<<releasing>>`.
5. **The page must not contradict `#cmd-changelog-check`'s documented contract** (verified at
   refinement, `cli-reference.adoc:1152-1278`): always exits `0` when it finds candidates, never
   a CI gate, checks *mentions only*, one direction (shipped-but-unmentioned). The new page
   frames the command as the mechanical half of a two-part convention; it restates none of that
   contract, only links to it.

### Tasks

#### Task 1 — write `docs/user-manual/concepts/releasing.adoc`
New file, `[#releasing]` anchor, title `= Cutting a Release`, following this outline (mirroring
`agent-session-workflow.adoc`'s register: a documented pattern, explicitly not a rule the flow
enforces):
- **Intro** — "cut a release" is unsupported until *your* project equips it with two files;
  neither is written by `pickle install`; state the no-inference rule up front (an agent never
  guesses a release procedure from repo conventions alone).
- **The two inputs** (a definition list, mirroring `project-structure.adoc`'s "Three kinds of
  file" style): `CHANGELOG.md` — read mechanically by `pickle changelog check`
  (`<<cmd-changelog-check>>`); `RELEASING.md` — prose an agent follows, read by no command at
  all.
- **What makes a `RELEASING.md` useful to an agent** — bullet list, per the Description: version
  scheme; where the version number lives; tag convention (which also determines what to pass to
  `--section`); the ordered steps; per step, whether the agent may run it unprompted or must hand
  back (tie explicitly to the publish gate — a release is mutating/irreversible, so any step that
  publishes always hands back, never auto-runs).
- **The fallback ladder** — the three cases verbatim from the Description (`RELEASING.md`
  missing → run `changelog check`, report, offer to *draft* one from observable evidence,
  execute nothing from the draft; `CHANGELOG.md` missing → say traceability cannot be checked,
  offer a Keep a Changelog skeleton; both missing → "cut a release" produces a setup step, not a
  release). State plainly this ladder degrades toward reporting, never toward guessing.
- **What this is not** — briefly name and reject the three heavier designs from the Description
  (a `release` key in `pickle.toml`; a `pickle changelog cut <version>` verb; a trigger whose
  final step executes the release), one clause each, so a future reader does not re-propose them
  — mirrors `cli-reference.adoc`'s own "why not X" convention used elsewhere in the manual.
- Cross-reference `<<agent-session-workflow>>` once (same register: a suggested pattern) and
  `<<cmd-changelog-check>>` at least twice (the two-inputs section and the fallback ladder).

#### Task 2 — wire the page into the book
In `docs/user-manual.adoc`, add
`include::user-manual/concepts/releasing.adoc[leveloffset=+1]` immediately after the
`agent-session-workflow.adoc` include (`:32`), before the blank line and `[part] == CLI
Reference` (`:34`).

#### Task 3 — pointer from `#cmd-changelog-check`
In `docs/user-manual/cli-reference.adoc`, add one paragraph at the end of the
`#cmd-changelog-check` section (after "...flagging that would be noise.", `:1278`, before
`[#cmd-serve]`, `:1280`) pointing to `<<releasing>>` for the broader convention this command is
the mechanical half of.

#### Task 4 — project-file inventory row
In `docs/user-manual/concepts/project-structure.adoc`'s "What to commit" table (`:73-107`), add
a row before the closing `|===` (`:107`): `CHANGELOG.md`, `RELEASING.md` (optional,
project-authored) — the two inputs `pickle changelog check` and the `<<releasing>>` convention
read; pickle never writes either, and nothing it does requires them.

### Acceptance test

```
just build
just docs-check
just lint
just test
```
All clean. **The by-hand cross-reference check this step used to require is gone**: T-067 merged
to `main` on 2026-08-22 (PR #60), so `just docs-check` and `go test ./...` both fail on a dangling
`<<releasing>>`/`<<cmd-changelog-check>>`/`<<agent-session-workflow>>` target, on an
`xref:<file>.adoc` form, and on the new page if it is never `include::`-d. Just run the commands
above (patched 2026-08-22 by T-067's review impact sweep). Also render the manual locally once
(`just docs-build`) and skim the new page's rendered output for the definition-list and bullet
formatting.

### Docs update (mandatory when user-facing)

This ticket *is* the docs update (Tasks 1–4). No other surface changes.

### Finish (mandatory)

1. Acceptance test green, including the by-hand xref check.
2. Summary: confirm all three open decisions were resolved as documentation-only (no code, no
   payload changes).
3. Suggested commit message:
   ```
   docs(user-manual): document the RELEASING.md + CHANGELOG.md release convention (T-111)
   ```
4. Tidy WIP commits into atomic ones (root-path child) before presenting.
5. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-111 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: chat: a question about what "cut a new release" would
  mean for a newly onboarded project led to a design discussion in which three heavier variants
  (a `release` key in `pickle.toml`, a `pickle changelog cut` verb, and a trigger that executes
  the release) were each challenged and rejected; the user then specified the surviving shape —
  a project-authored `RELEASING.md` + `CHANGELOG.md` the agent reads, with a defined fallback
  when either is absent — and asked for it to be filed as a user-manual change.
- 2026-08-20 — refined: resolved all three open decisions as documentation-only (manual-only, no
  payload change; the fallback's "offer to draft" stays prose, no verb implied or promised; the
  inverse board-vs-range check confirmed out of scope, belongs in its own ticket). Verified the
  new page's placement, its wiring point in `docs/user-manual.adoc`, and the two other touch
  points (`cli-reference.adoc`'s `#cmd-changelog-check` section, `project-structure.adoc`'s
  "What to commit" table) against the current tree. Noted T-067 (docs xref validation) is still
  `2-ready/`, not done, so this ticket's own new cross-references need a by-hand check instead of
  a mechanical one. TO DO → READY: implementation plan complete.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — patched by **T-067's review impact sweep**: T-067 is now `6-done/` (branch `feat/T-067-docs-xref-check` not yet merged), so this ticket's "docs-check cannot catch a dangling xref" assumption holds only until that branch lands. Wording updated to say which state applies and how to tell.
- 2026-08-22 — patched again by **T-067's impact sweep**, on merge: T-067 landed on `main` (PR #60, 2e29b50), so the "docs-check cannot catch a dangling xref" caveat is now simply false and has been removed rather than re-qualified. The gate is live for this ticket's cross-references.
- 2026-08-23 — READY → IN DEVELOPMENT: picked up. Applicability gate (fresh sub-agent): all
  assumptions hold; one non-blocking note (T-071 shifted `cli-reference.adoc` line numbers,
  harmless — Task 3 anchors on text, not lines). No blocking findings.
- 2026-08-23 — implemented per plan, all four tasks; see Description for detail. Acceptance
  green (`just build/docs-check/lint/test`, `just docs-build` rendered clean). `docs_readability`
  applied on the new page only. One atomic commit (`625dc6a`).
- 2026-08-23 — IN DEVELOPMENT → IN REVIEW: acceptance green
