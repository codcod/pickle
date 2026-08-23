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

`pickle` no longer has a `scaffold docs` subcommand, embedded docs-template payload, or
`internal/scaffold` package; `pickle help`/usage stops mentioning `scaffold docs`. The
`scaffold` command itself stays (T-113 is adding a `release` subcommand under it). Scaffolding
an AsciiDoc docs skeleton, its `snowball.yaml`, justfile fragments and a release-attach GitHub
Action for a project's user manual becomes `snowball`'s job (tracked as `SNOW-003` in the
`unity` workspace), not pickle's.

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

**Scope of removal:** everything T-110 added, minus the `scaffold` command surface itself
(kept for T-113's `release` subcommand, see below) — `internal/scaffold/` (package + tests),
the `runScaffoldDocs` function and its `"docs"` case in `internal/cli/scaffold.go`'s dispatch
(`runScaffold` itself, and the top-level `case "scaffold":` in `internal/cli/cli.go`, stay),
the `scaffold/docs-template/` embedded payload tree, the `all:scaffold` embed root's docs-
template half in `assets.go` (the `go:embed` line itself can likely stay if T-113 needs its own
embedded assets under `scaffold/` — confirm at refinement whichever shape T-113 takes), the
`usage()` line documenting `scaffold docs [...]`, and the `== \`pickle scaffold docs\`` section
+ Overview row in `docs/user-manual/cli-reference.adoc`. A `CHANGELOG.md` entry records the
removal as `### Breaking` (or this project's equivalent convention for a removed command —
confirm at refinement).

**Resolved at filing (user decisions):**

1. **T-116** (fixed bugs in exactly the feature this ticket deletes) — dropped
   (`7-dropped/`, superseded by this ticket); its findings (the `-theme.yml` naming
   requirement, the phantom-book origin, the heading/`leveloffset` mismatch, and the missing
   check/build regression test) were transplanted into the companion ticket below before it
   was dropped, so none of that investigation is lost.
2. **T-113** (extends `pickle scaffold` with a different, unrelated `release` subcommand:
   skeleton `CHANGELOG.md`/`RELEASING.md`, no AsciiDoc/snowball involved) **stays** — the
   `scaffold` command surface is not going away, only the `docs` subcommand under it. This
   ticket's removal must leave `pickle scaffold`'s dispatch structure intact (`runScaffold` in
   `internal/cli/scaffold.go`) for T-113 to add its `release` case to; only the `docs` case, and
   everything solely reachable from it, is removed.

Soft coupling: spawned by T-116 (this repo) — filed once T-116's refinement made the
cost-of-tracking-a-foreign-tool's-internals concrete enough to question the whole feature, not
just its bugs. The companion ticket describing the equivalent capability, ported and corrected
(including T-116's findings), is `SNOW-003` in the `unity` workspace's own board, against
`project: snowball` — out of scope for this repo's ticket ids (a separate installation, not a
registered child here).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-23 — created (TO DO). source: chat: user, refining T-116, concluded docs/release
  scaffolding (doc skeleton, justfile fragments, GH release-attach workflow) is snowball's job,
  not pickle's, and asked for pickle's copy to be removed and the capability moved to snowball
