---
id: T-110
title: opt-in scaffold command for a docs/release template (docs skeleton, snowball config, release-attach action, justfile), modeled on pickle's own pipeline
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: medium
cost: M
---

# T-110 — opt-in scaffold command for a docs/release template (docs skeleton, snowball config, release-attach action, justfile), modeled on pickle's own pipeline

## Outcome

Running a new, explicitly opt-in `pickle` subcommand (name to be pinned at refinement, e.g.
`pickle scaffold docs`) in a target repo lays down a minimal AsciiDoc docs skeleton, delegates
to `snowball init` for the render config, adds `justfile` `docs-check`/`docs-build` recipes
(only if a `justfile` already exists — never invents a task runner), and drops a
GitHub-Actions release-attach step — all parameterized by project name, and entirely separate
from `pickle install`, whose contract (scaffold brine only) stays unchanged.

## Description

Origin: a chat request asked `pickle` to scaffold, for other repos, the same docs/user-manual
tooling this repo already has for itself — `docs/`, `snowball.yaml`, a GitHub Action that
attaches the built manual to a release, and justfile targets (`version`, `default`,
`docs-build`) — "all based on the pickle repo."

That idea was challenged before filing, because it doesn't fit `pickle` as shipped:

- **Mission mismatch.** `pickle install` scaffolds exactly one thing — the brine ticket flow
  (`tickets/`, `BOARD.md`, the skill, the `AGENTS.md` marker block). A docs/release build
  pipeline has nothing to do with tickets, boards, or agent workflow; folding it into
  `pickle install` would blur "the brine installer" into a generic repo bootstrapper.
- **Four stacked tool assumptions.** "Modeled on the pickle repo" bakes in AsciiDoc + snowball
  (a sibling project of the author's, distributed via a private Homebrew tap), GitHub as the
  forge (this repo's own `.goreleaser.yaml` had to pin the forge explicitly because goreleaser
  otherwise prefers GitLab when a token is present — the same fragility would leak into every
  scaffolded repo), goreleaser as the release pipeline the attach-Action assumes, and `just` as
  the task runner. None of these follow from "a project adopted brine."
- **Partial duplication of an existing tool.** `snowball init` already writes a starter
  `snowball.yaml` in the current directory (verified: `snowball init --help`). Hand-rolling that
  file inside pickle duplicates a command one `brew install` away.
- **Drift risk.** Embedding a copy of this repo's *own* docs pipeline as a template means every
  future change to it (this repo's own T-047/T-048 and the release-workflow hardening since)
  needs a matching update to the embedded copy or the template silently rots.

Decision (user-approved): build it anyway, but scoped down per the above, as its own **opt-in**
subcommand — never folded into `pickle install`, never registered as an ongoing `pickle.toml`
concern (it is a one-shot scaffold, not something `doctor`/`upgrade` need to keep fresh like the
brine skill payload):

1. **Separate command surface**, not a flag on `pickle install`/`project add`. Exact verb/name
   is an open decision for refinement (`pickle scaffold docs` vs. `pickle docs init` vs.
   similar) — pick whatever reads clearly as unrelated to brine.
2. **Parameterize everything currently hardcoded to "pickle"** in this repo's own setup:
   project/binary name, the rendered artifact name (`pickle-user-manual` → `<project>-user-manual`),
   and the release-tag→version substitution in the attach step.
3. **Delegate to `snowball init`** for `snowball.yaml` instead of writing one from a template —
   shell out to it (or document it as a required manual step) rather than duplicating its
   defaults.
4. **Scaffold a minimal generic doc skeleton**, not this repo's actual manual content: one book
   master plus one placeholder chapter — `docs/user-manual/{installation,cli-reference,
   concepts/*,quickstart}.adoc` is pickle-specific prose and must not be copied verbatim.
5. **justfile recipes are additive, not assumed.** Only add `docs-check`/`docs-build` (and,
   per the ask, confirm `version`/`default` exist rather than overwriting them) when a
   `justfile` is already present; refinement must decide the no-`justfile` behavior (skip with
   a message vs. offer to create one) rather than silently inventing a task runner.
6. **GitHub-only, stated as a documented limitation**, not silently assumed universal — this
   repo's own release-attach pattern (`.github/workflows/release.yml`'s guarded `gh release
   upload` step, soft-failing so a broken docs build never blocks the release) is the only
   reference implementation available; no GitLab/other-forge equivalent is in scope here.

Open questions for refinement: final command name/UX; whether embedded scaffold assets follow
the existing `assets.go` `go:embed` pattern (`payloadFS`) or a new embed target; exact
placeholder content for the minimal doc skeleton and the release-attach Action step; and
whether `pickle doctor`/`board audit` need any awareness of this at all (current expectation:
no — nothing scaffolded here is an ongoing invariant pickle enforces).

Soft coupling: none hard-blocking. Loosely related to **T-047**/**T-048** (this repo's own docs
pipeline, the reference the scaffold is modeled on) and **T-011** (distribution/goreleaser,
the release pipeline the attach-Action assumes) — informative precedent only, not a
`depends-on:`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-21 — created (TO DO). source: chat: user asked pickle to scaffold docs/release tooling (docs dir, snowball.yaml, release-attach GitHub Action, justfile targets) modeled on this repo; idea was challenged (mission mismatch, tool-assumption stacking, `snowball init` duplication) and the user chose to proceed scoped down as a separate opt-in subcommand rather than folding it into `pickle install`
