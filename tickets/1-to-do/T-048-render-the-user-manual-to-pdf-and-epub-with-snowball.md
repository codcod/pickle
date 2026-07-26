---
id: T-048
title: render the user manual to PDF and EPUB with snowball
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-048 — render the user manual to PDF and EPUB with snowball

## Description

T-047 shipped the AsciiDoc user manual (`docs/user-manual.adoc`, book master + parts) with a
deliberately minimal build: sources are the deliverable, and `just docs-check` renders to
`/dev/null` via a `Gemfile`-pinned `asciidoctor` to catch broken includes/xrefs (T-047,
decision 1: "content only — no PDF/EPUB"). The user now wants renderable artifacts: **PDF and
EPUB output for the manual, produced by `snowball`**.

[`snowball`](https://github.com/codcod/snowball) (source: `~/Projects/private/snowball`; the
binary is installed via Homebrew) is a single Go binary that renders AsciiDoc book masters to
PDF and EPUB by orchestrating the native asciidoctor toolchain (`asciidoctor-pdf`,
`asciidoctor-epub3`, optionally `asciidoctor-diagram`/`mmdc`). It is config-driven
(`snowball.yaml`: books, formats, optional PDF theme, shared attributes file, revision
stamping, per-format failure levels) with commands `init`/`setup`/`doctor`/`build`/`check` —
`build` renders artifacts, `check` validates masters and discards output (MR pipelines).

Expected shape (to pin at refinement):

- a `snowball.yaml` at the repo root: one book (`src: docs/user-manual.adoc`,
  `attributes: docs/attributes.adoc`, `formats: [pdf, epub]`);
- `just` recipes wrapping it — e.g. `docs-pdf`/`docs-build` → `snowball build`;
- an output directory (e.g. `dist/docs/`) that is gitignored — artifacts are built, not
  committed;
- docs registration: `docs/README.adoc`'s "Validating" section (which currently states
  "There is no PDF/EPUB build — the sources are the deliverable") and
  `docs/user-manual/installation.adoc`/`cli-reference.adoc` where relevant must be updated —
  T-047's claim becomes false the moment this lands.

Open decisions for refinement:

1. **What happens to `just docs-check` and the `Gemfile`?** `snowball check` covers the same
   ground (render-and-discard validation) with per-format failure levels. Options: keep the
   lightweight asciidoctor `docs-check` as the registered `docs` command and add `snowball`
   recipes alongside; or replace `docs-check` with `snowball check` and drop the `Gemfile`
   (then `pickle.toml`'s `docs` key and the `AGENTS.md` marker block change too — self-host
   policy: hand edits mirroring `markerBlock()`). Note `snowball setup` installs its own
   pinned gems via bundler, so the repo-local `Gemfile` may become redundant.
2. **Revision stamping.** T-047 decision 5 deliberately skipped revnumber plumbing;
   `snowball` offers `revision: from: git-describe`. Adopt or keep static?
3. **PDF theme** — snowball's `theme:` key is optional; default asciidoctor-pdf styling is
   probably fine for a first cut (no `pdf-theme/` dir, mirroring T-047's no-theme stance).
4. **Toolchain gate.** `snowball` shells out to gems + (for diagrams) mermaid/Chrome — the
   manual has no diagrams, so the diagram toolchain should stay out of scope. Where does
   `snowball setup`/`doctor` fit in contributor docs (`installation.adoc` "From source" or
   `docs/README.adoc`)?

Soft couplings: **T-047** (done — established the manual, the `docs-check` recipe, the
`Gemfile`, and the `docs = "just docs-check"` registration this ticket may revisit); **T-019**
(its remaining PLAN.md item is untouched, but if decision 1 changes the `docs` command the
same self-host hand-edit discipline applies). This is repo tooling for pickle's own docs — the
`pickle` binary's behaviour does not change.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: chat (user request: render the manual to PDF/EPUB with the installed `snowball` tool; supersedes T-047's content-only scoping decision by explicit user choice)
