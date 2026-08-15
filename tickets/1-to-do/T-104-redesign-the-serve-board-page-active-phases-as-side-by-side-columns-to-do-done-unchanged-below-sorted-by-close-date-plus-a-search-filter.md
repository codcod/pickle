---
id: T-104
title: redesign the serve board page: active phases as side-by-side columns, TO DO/DONE unchanged below sorted by close date, plus a search filter
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-104 — redesign the serve board page: active phases as side-by-side columns, TO DO/DONE unchanged below sorted by close date, plus a search filter

## Outcome

After this ships, opening `pickle serve`'s board page shows READY, IN DEVELOPMENT, IN REVIEW
and REWORK as side-by-side columns instead of stacked sections, so the tickets actually moving
through the flow right now are visible together at a glance; TO DO and DONE keep their current
full-width list treatment below the columns, DONE now reads newest-closed-first, and a search
box lets the user narrow the whole page to tickets matching an id or title substring.

## Description

The board page (`internal/serve/templates/board.html`, `board-body`) currently renders all seven
status sections the same way: one `<section class="status">` per status
(`def.BoardStates()` order — TO DO, READY, IN DEVELOPMENT, IN REVIEW, REWORK, DONE, DROPPED),
each stacked vertically down the page, each child-project sub-grouped inside. This ticket
redesigns that layout for the four "active" statuses only:

1. **READY, IN DEVELOPMENT, IN REVIEW, REWORK become columns laid out side by side** (a small
   Kanban strip), one column per status, in that left-to-right order. Per the child-first
   grouping already used by `BoardView`/`ChildGroup`, when more than one child-project is
   registered each child gets its own row of these four columns (today only one child,
   `pickle`, is registered, so there is exactly one row) — the ticket's own
   `spawned-by`/exploration discussion settled this as "one row per child" rather than
   interleaving children within a column. Because the real data is wildly uneven per column
   (this repo today: READY 1, IN DEVELOPMENT 1, IN REVIEW 0, REWORK 0 — but IN DEVELOPMENT and
   IN REVIEW routinely fill up to the WIP cap and TO DO/DONE do not bound their length at all),
   each column needs its own scroll behaviour rather than assuming short lists; the exact
   mechanism (independent max-height + scroll per column vs. some other approach) is an
   implementation-plan decision, not fixed here.
2. **TO DO and DONE are explicitly out of the column redesign** — they keep today's stacked,
   full-width `<section class="status">` list rendering, unchanged in shape/markup intent.
   **DROPPED is not mentioned by this ticket's scope** and should be clarified during
   refinement (most likely: unchanged, same as TO DO/DONE, but confirm rather than assume).
3. **DONE's only behavioural change: sort newest-closed-first.** Today DONE (like every
   non-impact-ordered section) sorts by ticket number ascending (`board.Sort`, id order — see
   `internal/board/board.go`), which is creation order, not merge/close order. This ticket
   changes DONE's ordering to descending close date. "Close date" is the ticket's own last
   `## History` entry (in practice the merge line's date), the same date field the activity
   timeline already parses via `ticket.HistoryEntries`/`ticket.MergeLine` — not something read
   from `BOARD.md`'s rendered cell text, which carries no date at all today. This is a sort-order
   change only; DONE's card/row shape does not change.
4. **A search field filters the whole page**, live, by id/title substring — across the new
   columns and the unchanged TO DO/DONE sections alike. It is presentation-only (no server
   round-trip needed in principle, though the concrete mechanism — client-side JS vs. a
   server-rendered fragment — is an implementation-plan decision) and must keep working
   correctly alongside the two things already on this page that mutate the DOM: the existing
   per-child filter bar (T-061) and the 5s htmx auto-refresh that swaps `#board`'s innerHTML
   (T-053/T-061). Whether the search box composes with the child filter bar, replaces it, or
   the two stay independent is an open decision for refinement.

**Prior art / cautionary precedent (read before refining):** T-062 (`7-dropped/`) attempted a
visual-only reshaping of this same board page (rounder cards, a framed container) against a
static mock, went through two implementation rounds, and was **reverted** — "both rounds looked
worse than the pre-T-062 baseline; no code merged to main." That ticket's own review history is
worth reading in full before writing this one's Implementation Plan: it is direct evidence that
a plausible-sounding visual redesign of exactly this page can ship functioning code that still
fails on the only criterion that matters (does it actually look/read better), and that the fix
was a **user-driven visual sign-off step**, not a mechanical acceptance test. This ticket's
Implementation Plan should build in an equivalent live visual-review gate (e.g. running
`pickle serve` locally and getting explicit user sign-off against the intended layout) before
marking acceptance green, rather than relying solely on markup/CSS assertions in
`internal/serve/serve_test.go`.

**Soft couplings (no hard `depends-on:`):** builds on the existing board view model
(`internal/serve/view.go`'s `BoardView`/`Section`/`ChildGroup`, `internal/board.Sort`,
`ticket.HistoryEntries`/`MergeLine`), the child filter bar (T-061), the htmx polling fragment
(T-053), and the theme tokens (T-054). T-055 (WIP badge highlighting, still open in `1-to-do/`)
touches the same WIP-badge markup the IN DEVELOPMENT/IN REVIEW columns render — worth a glance
during refinement in case the two collide, but no hard dependency is implied here.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-15 — created (TO DO). source: chat: user reviewed two rounds of a throwaway static
  mockup (generated from the real tickets/BOARD.md, discussed and iterated live) reshaping the
  serve board page's layout, then asked to file a ticket to build it for real.
