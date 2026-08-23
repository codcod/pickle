---
id: T-117
title: remove pickle scaffold docs — docs/release scaffolding belongs to snowball, not pickle
project: pickle
depends-on: []
spawned-by: [T-116]
impact: medium
complexity: low
cost: S
---

# T-117 — remove pickle scaffold docs — docs/release scaffolding belongs to snowball, not pickle

## Outcome

`pickle` no longer has a `scaffold docs` command, embedded docs-template payload, or
`internal/scaffold` package; `pickle help`/usage stops mentioning it. Scaffolding an AsciiDoc
docs skeleton, its `snowball.yaml`, justfile fragments and a release-attach GitHub Action for a
project's user manual becomes `snowball`'s job (tracked as a companion ticket in the `unity`
workspace against the `snowball` child), not pickle's.

## Description

T-110 added `pickle scaffold docs`, an opt-in, standalone command unrelated to brine that lays
down a minimal AsciiDoc docs skeleton, shells out to `snowball init` for `snowball.yaml`,
appends justfile `docs-check`/`docs-build` recipes, and drops a GitHub Actions release-attach
workflow. T-110's own Description already named the risk at filing time: *"embedding a copy of
this repo's own docs pipeline as a template means every future change to it... needs a matching
update to the embedded copy or the template silently rots"* — accepted then as a manageable
trade-off for the convenience of a one-shot scaffold.

T-116 (this repo, `2-ready/`) shows that trade-off has already soured, and for a more basic
reason than "pickle's copy goes stale over time": pickle's shipped scaffold has to track
**`snowball`'s own internal defaults and conventions**, which pickle has no authority over and
keeps getting subtly wrong —

- `snowball init`'s default `snowball.yaml` references a second book,
  `docs/developer-handbook.adoc`, that pickle's scaffold never creates. It isn't an arbitrary
  placeholder: it mirrors a real two-book AsciiDoc setup this same workspace already runs
  (`rick`, see `RICK-185`) — snowball's own default is modeled on real usage pickle has no
  visibility into.
- asciidoctor-pdf's PDF theme loader requires a theme file's name to literally end in
  `-theme.yml` (it derives a theme *name* by stripping that suffix, then reloads
  `<name>-theme.yml`) — an asciidoctor-pdf/snowball implementation detail with no reason for
  pickle to know it, discovered only by hitting the resulting silent-fallback-plus-nonzero-exit
  failure firsthand (T-116's refinement).
- The scaffolded chapter's AsciiDoc heading level has to match the `leveloffset` convention
  `snowball`-rendered book masters expect — again a fact about how snowball's toolchain
  (asciidoctor) resolves section nesting, not about pickle.

Every one of these is a fact pickle has to keep re-deriving about a tool it doesn't own, from
outside, by trial and error — exactly the shape of drift T-110 anticipated. `snowball` is the
actual domain owner of `snowball.yaml`'s schema, its own `init` defaults, and the PDF theme
convention; only it can keep a scaffolded doc skeleton and the config it targets in lock-step,
because it controls both sides. Moving the capability there turns three foreign-tool facts
pickle had to chase into one project's own facts about itself.

**Scope of removal:** everything T-110 added — `internal/scaffold/` (package + tests),
`internal/cli/scaffold.go`, the `scaffold` case in `internal/cli/cli.go`'s dispatch and
`usage()`, the `scaffold/docs-template/` embedded payload tree, the `all:scaffold` embed root
in `assets.go`, and the `== \`pickle scaffold docs\`` section + Overview row in
`docs/user-manual/cli-reference.adoc`. A `CHANGELOG.md` entry records the removal as
`### Breaking` (or this project's equivalent convention for a removed command — confirm at
refinement).

**Open questions for refinement (flagging now, not resolved by this filing):**

1. **T-116** (`2-ready/`, this repo) fixes bugs in exactly the feature this ticket deletes.
   Building T-116's fix first only to delete it right after would be wasted work — dropping
   T-116 once this ticket is confirmed seems right, but that's the user's call, not this
   ticket's to make unilaterally.
2. **T-113** (`2-ready/`, this repo) extends `pickle scaffold` with a *different* subcommand
   (`release`: skeleton `CHANGELOG.md`/`RELEASING.md`, no AsciiDoc/snowball involved) and is
   `spawned-by: [T-110, T-111]`. It doesn't touch the docs/snowball pipeline this ticket
   removes, so it may be unaffected in scope — but if the `docs` subcommand is `scaffold`'s only
   reason to exist today, refinement must decide whether the `scaffold` verb group itself stays
   (as an empty shell for T-113's `release` verb to land in later) or whether T-113 is
   reconsidered too. Not this ticket's call either.

Soft coupling: spawned by T-116 (this repo) — filed once T-116's refinement made the
cost-of-tracking-a-foreign-tool's-internals concrete enough to question the whole feature, not
just its bugs. A companion ticket describing the equivalent capability, ported and corrected,
belongs in the `unity` workspace's own board against `project: snowball` — out of scope for
this repo's ticket ids (a separate installation, not a registered child here).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-23 — created (TO DO). source: chat: user, refining T-116, concluded docs/release
  scaffolding (doc skeleton, justfile fragments, GH release-attach workflow) is snowball's job,
  not pickle's, and asked for pickle's copy to be removed and the capability moved to snowball
