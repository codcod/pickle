---
id: T-047
title: AsciiDoc user manual in docs/ + slim README to about & install
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: M-L
---

# T-047 — AsciiDoc user manual in docs/ + slim README to about & install

## Description

Today everything a user can learn about `pickle` lives in one 378-line `README.md`: what the
tool is, install instructions, the design principle, the full command reference (one `##`
section per command), the configuration schema, and build-from-source notes. That single file
is both the shop window and the manual, which makes it long, hard to navigate, and prone to
the accuracy drift T-019 catalogues.

Split the two roles:

1. **`README.md` becomes the shop window only** — what this project is about (the
   ticket-based, board-driven flow; the split-judgment-from-mechanics principle in one or two
   paragraphs) and how to install it (binary + build-from-source). Everything else moves to
   `docs/` and the README links to the manual. No content should exist in both places.

2. **`docs/` gains an AsciiDoc user manual**, following the structural patterns of the
   `~/Projects/personal/translator/ai-sdlc/docs/` book tree (the reference layout, not its
   content):
   - a master document `docs/user-manual.adoc` (`:doctype: book`, `:toc:`, `[part]` headings)
     that `include::`s per-section files with `leveloffset=+1`;
   - a shared `docs/attributes.adoc` for product names/paths used across pages;
   - a `docs/README.adoc` index describing the book and how to build it;
   - section files under `docs/user-manual/`.

   Required structure (per the user's request):
   - **Part "Start Here"** — `quickstart.adoc`, `installation.adoc`,
     `your-first-project.adoc` (install → register a child-project → file, refine, implement,
     validate one ticket end-to-end).
   - **Part "Concepts"** — explanation of the process: tickets as the single artifact,
     status = directory, the board as a generated index, the seven statuses and the state
     machine, READY gate, review findings (severity → disposition), WIP limits, dependencies
     vs lineage, the skill/CLI split (judgment vs mechanics).
   - **Part "CLI reference"** — one section per command (`install`, `upgrade`, `uninstall`,
     `doctor`, `project add/remove/list`, `ticket new`, `ticket move`, `board audit`,
     `board sync`), absorbing the README's current per-command sections and the
     `pickle.toml` configuration reference.

3. **A docs build** following the ai-sdlc pattern where it pays for itself: a `Gemfile` with
   `asciidoctor`/`asciidoctor-pdf` and a `just docs-pdf` (or at least `just docs-check`
   rendering/validating the adoc tree) recipe. Whether to ship PDF output or only validate
   the sources is an open decision for refinement. If a docs command lands, register it as
   the child's `docs` command in `pickle.toml` (currently commented out).

Content sources: the current `README.md` command sections, the skill's
`resources/tickets-README.md` rules (§0–§8) — the manual explains the process for humans but
must not fork the rules' normative text; it should summarise and point to the skill as the
source of truth.

**Soft couplings:**
- **T-019 (README accuracy polish)** — partially mooted by this ticket: its items 1–3 target
  README passages that this ticket moves or deletes. Whichever ticket is refined first should
  re-check the other; T-019's item 4 (`PLAN.md` synopsis) is untouched by this ticket.
- The `AGENTS.md` marker block and `install.go`'s `markerBlock()` mention docs in the commit
  policy; no change expected, but verify at refinement.
- Self-host policy applies: this is a `pickle`-child ticket, branch `feat/T-047-<slug>` in
  this repo; no `pickle install|upgrade` runs against the repo.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: chat (user request: AsciiDoc manual in docs/, README reduced to about + install, patterned on ai-sdlc's docs tree)
