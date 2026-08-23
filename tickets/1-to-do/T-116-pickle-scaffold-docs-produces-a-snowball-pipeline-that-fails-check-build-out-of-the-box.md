---
id: T-116
title: pickle scaffold docs produces a snowball pipeline that fails check/build out of the box
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-116 — pickle scaffold docs produces a snowball pipeline that fails check/build out of the box

## Outcome

A freshly scaffolded `pickle scaffold docs` tree passes `snowball check` and `snowball build`
immediately, with no manual edits beyond the `snowball.yaml` `src`/`out` follow-up the command
already tells the user to make.

## Description

Reproduced against the shipped binary (`pickle scaffold docs --project-name <name>` in an
empty directory with `snowball` on `PATH`, no other flags): the command reports success, and
the project-name substitution into `docs/attributes.adoc` / `docs/user-manual.adoc` is correct
— but the resulting tree fails `snowball check` and `snowball build` outright, for three
independent reasons, none of which T-110's acceptance test caught (it only asserted file
existence and `grep`-matched substituted content; it never actually ran `snowball check` or
`snowball build` against the scaffold's own output):

1. **Wrong heading level in the placeholder chapter.** `scaffold/docs-template/user-manual/
   introduction.adoc` opens with `== Introduction` (level 1). `user-manual.adoc` includes it
   with `leveloffset=+1`, bumping it to level 2 — invalid as the first section after the book
   title (asciidoctor expects level 0 or 1 there). `snowball check` fails with: `section title
   out of sequence: expected levels 0 or 1, got level 2`. Compare this repo's own
   `docs/user-manual/quickstart.adoc`, which opens with `= Quickstart` (level 0) precisely so
   `leveloffset=+1` lands it at level 1 — the template should follow the same convention.

2. **`snowball init`'s default config references a book the scaffold never creates.** The
   scaffold shells out to `snowball init` for `snowball.yaml` (by design, T-110 decision 5) but
   never inspects or corrects what it writes. Its default output includes a second `books:`
   entry, `src: docs/developer-handbook.adoc`, which does not exist anywhere in the scaffold.
   `snowball check`/`build` fail on it: `input file .../docs/developer-handbook.adoc is
   missing`.

3. **`snowball init`'s default config references a theme file the scaffold never creates.**
   The same default output sets `theme: docs/pdf-theme/ai-sdlc-theme.yml`; `pickle scaffold
   docs` writes nothing under `docs/pdf-theme/`, unlike this repo's own pipeline (which ships
   `docs/pdf-theme/pickle-theme.yml` and points its own `snowball.yaml` at it). Asciidoctor-pdf
   recovers by silently falling back to its built-in default theme and still renders the PDF,
   but `snowball build`/`check` still exit non-zero (`could not locate or load the pdf theme
   ... reverting to default theme` followed by `Error: exit status 1`) — a hard failure for
   anything scripting on the exit code, including the scaffold's own `.github/workflows/
   docs-release.yml`, whose `snowball build` step this would otherwise trip.

Fix shape (confirm at refinement): (1) is a one-line template fix (`==` → `=`). (2) and (3)
are the same underlying gap — the scaffold's post-`snowball init` guidance note (currently only
covering `src:`/`out:`) needs to also tell the user to drop the phantom second book and either
remove the `theme:` line or point it at a theme the scaffold ships; shipping a small generic
theme file (mirroring `docs/pdf-theme/pickle-theme.yml`, which is already generic — `extends:
default` plus a few font tweaks, not pickle-branded) under `scaffold/docs-template/pdf-theme/`
is the more complete fix and keeps decision 5's "never hand-write snowball.yaml" intact
(pickle still doesn't parse/rewrite the YAML — it ships an asset and documents where to point
at it, same shape as the existing `src`/`out` guidance). Either way, the acceptance test must
gain a step that actually runs `snowball check` (and ideally `snowball build`) against a fresh
scaffold, since that is what would have caught all three of these at review time.

Soft coupling: spawned by manual reproduction of T-110's shipped command, not a review of
T-110 itself (T-110 is already `6-done/`) — filed as a bug against the feature it delivered.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-23 — created (TO DO). source: chat: user ran `pickle scaffold docs` in a foreign project and reported the scaffolded docs looked wrong; reproducing it in a scratch dir surfaced three independent defects (bad heading level, a phantom second book, and a dangling pdf-theme reference) that make a fresh scaffold fail `snowball check`/`build` out of the box
