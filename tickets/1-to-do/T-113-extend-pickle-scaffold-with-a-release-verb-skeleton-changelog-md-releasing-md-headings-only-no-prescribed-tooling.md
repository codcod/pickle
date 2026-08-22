---
id: T-113
title: extend 'pickle scaffold' with a release verb: skeleton CHANGELOG.md + RELEASING.md (headings only, no prescribed tooling)
project: pickle
depends-on: []
spawned-by: [T-110, T-111]
impact: low-medium
complexity: low
cost: S
---

# T-113 — extend 'pickle scaffold' with a release verb: skeleton CHANGELOG.md + RELEASING.md (headings only, no prescribed tooling)

## Outcome

Running `pickle scaffold release` in a target repo writes two skeleton files — an empty-
`[Unreleased]`-section `CHANGELOG.md` (Keep a Changelog shape) and a `RELEASING.md` with section
headings only (Versioning / Build / Publish / Verify, each a `TODO` placeholder, no prescribed
commands or tools) — additive-only like `pickle scaffold docs` (existing files are left alone
without `--force`). This is exactly the scaffold `docs/user-manual/concepts/releasing.adoc`
(T-111) describes an agent offering when either file is missing, made real, and it is the
deliberately narrowed remainder of a larger scaffold-release design that was proposed and then
challenged down to this.

## Description

**Origin and what this deliberately is not.** A chat design discussion proposed extending
`pickle scaffold` (T-110's verb group: today only `scaffold docs`) with a `scaffold release`
verb that would also write a language-forked (`--lang go|rust`) GitHub Actions release workflow,
additive justfile recipes with language-specific bodies, and a `RELEASING.md`/workflow pair whose
content depended on shelling out to `goreleaser init` for Go and on picking a Rust release
tool (`cargo-dist` vs. plain `cargo publish`) pickle has no consensus reference for. That design
was challenged and rejected before filing, on grounds consistent with T-110's own filing
rationale and this ticket's siblings:

- **False confidence asymmetry.** Treating a Go workflow modeled on this repo's own
  goreleaser+GitHub+Homebrew-tap pipeline as "high-confidence" repeats the exact mistake T-110
  was filed to correct ("modeled on the pickle repo" baking in choices that don't follow from
  "adopted brine") — a pipeline validated in one repo by one author is not a Go convention, and
  Rust's harder-to-guess pipeline just makes the same one-sample bias visible instead of hidden.
- **Fails the competence-boundary test.** A release workflow, a justfile recipe body, and a
  choice of Rust distribution tool require no ticket-domain knowledge (no ticket ids, no board
  state, no prefixes) — the same test that ruled a `release`-executing trigger out of scope for
  T-111 rules this content out of scope for a scaffold command too; renaming the feature from
  "trigger" to "scaffold" doesn't move it across that line.
- **Duplicates better-maintained tools.** `cargo generate`, `cargo-dist init`, and
  `goreleaser init` itself already solve "scaffold a release pipeline for language X," tracked by
  people who follow that ecosystem's churn — the same objection already used in this repo against
  a `pickle changelog cut` verb (`git-cliff`/`changie` own that space), reapplied here to a larger
  surface.
- **Demonstrated drift risk.** T-110's own first review (finding F1) caught its *one* embedded
  workflow template pinning `actions/checkout@v4` against this repo's own `@v7`, on the very first
  review of the very first template shipped. Two more per-language embedded workflow/tooling
  templates multiply that exact, already-observed risk rather than avoiding it.
- **Larger, unaddressed security surface.** A release workflow needs `contents: write` and
  typically a publish token (crates.io, a Homebrew tap PAT, signing keys) — a materially higher
  stakes template to embed and stamp into every scaffolding project than the docs-attach step
  T-110 shipped.
- **Silent re-coupling to `install`.** Branching scaffolded *content* on whether `pickle.toml`
  exists (to decide whether to mention `pickle changelog check`) would make `scaffold`'s output
  depend on `install` having run — precisely the "mission mismatch" T-110 was filed to keep
  `scaffold` free of (decision 8: no `pickle.toml`/doctor/board-audit integration).

**What survives, and is this ticket's actual scope.** Two files only, neither language-forked,
neither carrying a tool opinion:

- **`CHANGELOG.md`** — a Keep a Changelog skeleton with a single empty `[Unreleased]` heading.
  Generic, already implied as a fallback offer by T-111's Description, and exactly what
  `pickle changelog check` needs to find *a* file to check against.
- **`RELEASING.md`** — section headings only (a small fixed set: Versioning, Build, Publish,
  Verify — exact wording pinned at refinement), each body a single `TODO` placeholder comment,
  no commands, no tool names, no language branch. This is structure, not doctrine — the same
  register as `scaffold docs`' own one-placeholder-chapter skeleton
  (`user-manual/introduction.adoc`), not a filled-in procedure.

No `--lang` flag, no GitHub Actions template, no justfile recipes, no shell-out to any
release tool. If those are ever wanted, T-111's fallback-ladder page must keep reading correctly
whether or not they exist (T-111 decision 2: the page documents a pattern, never promises a
verb) — so nothing here is a prerequisite for anything else, and nothing here forecloses a
future, separately-challenged ticket for the richer version.

**Soft couplings** (no hard `depends-on:`):

- **T-110** (done) — the `scaffold` verb group and its `internal/scaffold` package this ticket
  extends; `Docs`'s per-file template-write loop and its additive-justfile-recipe helper are the
  precedent for how `Release` should be built (shared helpers, not a parallel implementation).
- **T-111** (`2-ready/`) — the manual page (`docs/user-manual/concepts/releasing.adoc`) whose
  fallback ladder describes exactly the two files this ticket writes; that page's prose must
  still read correctly regardless of whether this ticket ships (decision 2), and once this
  ships the page could gain one added sentence naming the command — a documentation follow-up,
  not a code dependency, and out of scope for this ticket to add.
- **T-093/T-094/T-095/T-097** (done) — `pickle changelog check`, the command the scaffolded
  `CHANGELOG.md`'s `[Unreleased]` section exists for; this ticket must not alter that command's
  contract.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: chat: a chat design discussion proposed extending
  `pickle scaffold` with a language-forked (`--lang go|rust`) `scaffold release` verb writing a
  GitHub Actions workflow, justfile recipes, and a filled-in `RELEASING.md`; challenged on six
  grounds (false Go/Rust confidence asymmetry, failing the competence-boundary test already
  applied to the rejected release-executing trigger, duplicating cargo-dist/goreleaser-init,
  T-110's own first-review drift finding recurring, an unaddressed publish-secrets security
  surface, and silent re-coupling to `install` via pickle.toml-detection) and narrowed to the
  two skeleton files T-111's fallback ladder already describes; spawned-by T-110 (the verb group
  and package this extends) and T-111 (the manual page this fulfills).
