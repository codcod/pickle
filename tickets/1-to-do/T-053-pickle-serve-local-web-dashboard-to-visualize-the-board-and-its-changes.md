---
id: T-053
title: pickle serve — local web dashboard to visualize the board and its changes
project: pickle
depends-on: []
spawned-by: []
impact: medium-high
complexity: high
cost: L
---

# T-053 — pickle serve — local web dashboard to visualize the board and its changes

## Description

Add a `pickle serve` command that starts a **local, read-only web server** rendering the
board as a browser dashboard, so a human can *see* the flow — the board grouped by
child-project, each ticket's prose, and how the board changed over time — instead of reading
`tickets/BOARD.md` and grepping `## History` sections.

Today the only views of the flow are the generated `tickets/BOARD.md` table and the ticket
markdown files. That is enough for agents and diffs, but poor for humans: a ticket's real
story (description, plan, review findings, history) lives in one file per ticket, and "what
moved this week" is only reconstructable by reading every ticket's `## History`. `serve`
turns the existing, already-authoritative data into a navigable surface — no new source of
truth, no writes.

**Scope sketch (to be pinned down at refinement).**

- `pickle serve [--addr host:port] [--open]` — starts an HTTP server on a loopback address by
  default; prints the URL; runs until Ctrl-C. **Read-only**: no route mutates a ticket, moves
  a file, or regenerates the board — the CLI stays the only writer.
- **Board view** — the seven statuses as columns/sections, grouped by child-project, ordered
  exactly as `internal/board` orders the generated board (impact descending, ties by id), so
  the dashboard and `BOARD.md` can never disagree.
- **Ticket view** — one page per ticket: frontmatter as a header (project, grades,
  `depends-on:`, `spawned-by:`), then the rendered markdown body (Description, Implementation
  Plan, Review, History), with dependency and lineage ids as links.
- **Changes view** — "and its changes" is the point of the ticket, and has two readings that
  refinement must choose between (or sequence):
  1. **Activity timeline** built from the ticket files themselves — every dated `## History`
     line across all tickets, merged and sorted newest-first (transition, reason, child).
     Pure function of the repo, no git needed.
  2. **Live refresh** — the page updates while the agent works: poll or watch `tickets/` and
     re-render when a ticket file or the board changes (mtime/digest based).
  Reading (1) is the one that adds information the board cannot show; (2) is what makes the
  dashboard usable as an ambient window during a session. Both are plausible for a first cut;
  the split point is a refinement decision.
- **Health/diagnostics** — surface `pickle board audit` findings and the WIP-limit state per
  child as a banner, so an out-of-sync or invariant-breaking board is visible in the UI
  (running the audit read-only, never auto-fixing).

**Tech stack — deliberately minimal, mirroring `~/Projects/private/unity/rick/apps/standards`
(the reference implementation the request pointed at):**

- Go stdlib `net/http` + `http.ServeMux` only — no web framework, no router dependency.
- `html/template` templates and CSS/JS assets under `internal/serve/` (name TBD), **embedded
  with `//go:embed`** so the single `pickle` binary keeps working with no network, no asset
  directory and no build step (same property as the existing skill payload in `assets.go`).
- Hand-written CSS; **htmx** as the only client-side script (vendored, served from the
  embedded FS) for polling/partial swaps — no npm, no bundler, no SPA.
- Markdown → HTML for ticket bodies. `standards` uses `goldmark` (+ GFM extension, needed for
  the board's pipe tables); pickle currently has exactly **one** direct dependency
  (`BurntSushi/toml`), so adding `goldmark` is a real decision to confirm with the user at
  refinement — the alternative is rendering only the structured parts (frontmatter, history,
  board cells) and showing ticket bodies as preformatted text.
- Tests as `httptest`-driven handler tests over a fixture `tickets/` tree, matching the
  existing `internal/*` table-test style; wiring goes in `internal/cli/` next to `board.go`,
  with the command added to the dispatch table and usage text in `internal/cli/cli.go`.
- User-facing surface, so it needs a `docs/user-manual/` chapter and a `just docs-check` pass.

**Soft couplings** (no hard `depends-on:` proposed):

- T-044 made `BOARD.md` a generated artifact with the ticket files as the single source of
  truth — that is the invariant this dashboard reads from, and the reason it can be a pure
  view. `serve` must render from ticket files via `internal/board`/`internal/ticket`, not by
  parsing `BOARD.md`.
- T-049 (rendered cell-width cap) and T-052 (audit vs. registry-changed board) shape what the
  board renderer and audit report expose; if `serve` reuses those, expect small refactors
  rather than duplicated logic.
- T-046 (self-host awareness) is adjacent only: `serve` run in this repo will see the same
  self-host quirks the doctor warnings describe.

**Explicit non-goals** (to keep the first cut minimal): no ticket editing or moving from the
browser, no authentication, no non-loopback binding by default, no multi-repo/remote hosting,
no persistence or database, no git history mining beyond what the ticket files already record.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: chat — request for a `serve` command rendering the
  board and its changes, with a minimal stack modelled on `rick/apps/standards`
