---
id: T-104
title: "redesign the serve board page: active phases as side-by-side columns per child, with a search filter"
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-104 — redesign the serve board page: active phases as side-by-side columns per child, with a search filter

## Outcome

After this ships, opening `pickle serve`'s board page shows READY, IN DEVELOPMENT, IN REVIEW
and REWORK as side-by-side columns instead of stacked sections, so the tickets actually moving
through the flow right now are visible together at a glance; TO DO, DONE and DROPPED keep their
current full-width list treatment below the columns; and a search box narrows the whole page to
tickets whose id or title matches what you type.

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
   `pickle`, is registered, so there is exactly one row). Because the real data is uneven per
   column (IN DEVELOPMENT and IN REVIEW are capped at the WIP limit, but READY and REWORK are
   not bounded at all), each column carries its own scroll rather than assuming short lists.
2. **TO DO, DONE and DROPPED are out of the column redesign** — all three keep today's stacked,
   full-width `<section class="status">` list rendering, below the columns, in the board's
   existing order.
3. **A search field filters the whole page**, live, by id/title substring — across the new
   columns and the unchanged TO DO/DONE/DROPPED sections alike. It is presentation-only, and
   must keep working alongside the two things already on this page that mutate the DOM: the
   per-child filter bar (T-061) and the 5s htmx auto-refresh that swaps `#board`'s innerHTML
   (T-053/T-061). Search and the child filter **compose** — a ticket is visible only if it
   passes both.

**Explicitly not in scope: ticket ordering.** An earlier draft of this ticket also re-sorted
DONE newest-closed-first; that was cut at refinement on the user's instruction ("keep the order
as-is, no changes in this ticket regarding the order"), and the title was corrected to match.
No section's sort order changes here: every group is still ordered by `board.Sort`, so the
dashboard and `BOARD.md` continue to agree, and the guarantee documented at
`docs/user-manual/cli-reference.adoc` (the board page renders "in exactly the order `BOARD.md`
uses … both views call the same ordering code, so they cannot disagree") survives untouched.
This is a **layout-only** change plus a search affordance.

**Absorbed: T-055.** T-055 ("serve: the board's at-limit WIP badge is never highlighted
(`.count` overrides `.wip-full`)") is a one-line CSS specificity fix to the exact WIP-badge
markup this ticket's column headers re-render. Left as a separate ticket it would conflict
against markup that no longer exists, so it is folded in here as task 6 and T-055 is dropped
with "absorbed by T-104".

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

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/T-104-board-lane-layout
```

The target child `pickle` is a root-path child (`path = "."`), so per rules §0 tidy WIP commits
into atomic ones before presenting, and **keep** that history by default rather than squashing.
Do not push or open an MR without explicit approval.

### Prerequisite gate (hard)

None. `depends-on:` is empty; every ticket this builds on (T-053, T-054, T-061, T-080) is in
`6-done/` and merged. `pickle serve` is read-only — running it against this repo is within the
self-modify policy (it is not `install|upgrade|uninstall`).

### Confirmed design decisions (do not deviate without asking)

1. **Column membership is derived from flow data, never hardcoded.** The four column states are
   *the non-terminal states other than the initial state*. For brine that is exactly READY,
   IN DEVELOPMENT, IN REVIEW, REWORK. No status name string appears in `internal/serve`.
2. **Column order is lifecycle order** (`Definition.States()` order), which yields
   READY → IN DEVELOPMENT → IN REVIEW → REWORK left-to-right. This deliberately differs from
   `BoardOrder` (active-first: IN DEVELOPMENT, IN REVIEW, REWORK, READY), which continues to
   drive the stacked sections below. Both orders come from the flow definition.
3. **No ordering change anywhere.** Within every column and every section, `board.Sort` remains
   the only sort. `internal/board` is not modified by this ticket. (This is the scope cut
   recorded in the Description; it also keeps T-103, which proposes changing `board.Sort`'s
   tie-break, conflict-free with this ticket.)
4. **One row per child.** Each registered child-project gets its own row of the four columns,
   in `cfg.Projects` order. The row wrapper carries `class="child" data-child="<name>"` — the
   same contract `board-filter.js` already hides on — so the T-061 filter keeps working with no
   change to its selector.
5. **Search composes with the child filter, in one script.** Extend the existing
   `internal/serve/static/board-filter.js` rather than adding a second script: two independent
   scripts both toggling `hidden` would fight over the same elements. One `apply()` evaluates
   both the active child filter and the current query.
6. **The search input lives outside `#board`.** `#board` is what the 5s htmx poll swaps; an
   input inside it would lose focus and its typed value every five seconds. The query is
   mirrored onto the persistent `<main class="page">` as `data-query`, exactly as `data-filter`
   already is, and re-applied on `htmx:afterSwap`.
7. **Matching is server-precomputed.** `Entry` gains a `Search` field (lowercased
   `"<id> <title>"`) built in `view.go`, rendered as `data-search`. Logic belongs in `view.go`
   where it is testable without a template (that file's stated rule), and the JS stays a dumb
   substring test. Match is id + title only — not reasons, merge lines or edges.
8. **Counts are board facts, not filter facts.** Column/section heading counts and the filter
   bar's chip counts continue to report what the tree holds; they are not recomputed as the
   user types. A query that hides every card in a column leaves that column visibly empty —
   acceptable, and cheaper than a JS-maintained count that could disagree with the server.
9. **Shape from the existing token set.** Reuse `--panel`, `--panel-2`, `--line`, `--muted`,
   `--accent`, `--mono`; keep the sans body font and the light/dark palette (T-054). Column
   cards reuse the existing `.ticket-row` markup with a scoped column-context override, rather
   than inventing a parallel card component — this keeps the diff small and inherits theming.
   This is the same discipline T-061 decision 4 set and T-062 violated.
10. **The acceptance gate is visual and human.** T-062 shipped twice with green tests and was
    reverted both times for looking worse. Markup assertions here prove wiring, not quality:
    the ticket may not move to IN REVIEW without explicit user sign-off in a browser, in both
    light and dark.

### Tasks

#### Task 1 — expose the active states on the flow definition

`internal/flow/flow.go`: add `func (d *Definition) ActiveStates() []State`, returning the
non-terminal states excluding `d.Initial()`, in lifecycle (`States()`) order. Precompute it in
`New()` alongside `wipStates` and return a `slices.Clone`, matching the existing accessors.
Document *why* the set is defined by exclusion (a flow's "work in flight" is everything past the
backlog entry point that is not an archive), so a future flow inherits the behaviour without
editing `serve`.

`internal/flow/flow_test.go`: assert brine's `ActiveStates()` is exactly
`[2-ready, 3-in-development, 4-in-review, 5-rework]` in that order, and that it excludes both
the initial and both terminal states.

#### Task 2 — restructure the board view model

`internal/serve/view.go`:

- Add `Search string` to `Entry`; populate it in `newEntry` as
  `strings.ToLower(t.ID + " " + title)`.
- Add `Lane` (one child's tickets in one active state: `Status`, `Entries`, `Count`, `Limit`)
  and `ChildRow` (`Child string`, `Lanes []Lane`).
- `BoardView` gains `Rows []ChildRow`; `Sections` now carries **only** the states not in
  `ActiveStates()` (TO DO, DONE, DROPPED for brine), still in `BoardStates()` order.
- In `buildBoard`, build `Rows` by iterating `cfg.Projects` outer and `def.ActiveStates()`
  inner; keep using `board.Sort` and `board.WIPCounts` exactly as today (decision 3). Factor the
  existing per-(state, child) grouping into one helper used by both loops so the grouping rule
  is not written twice.

#### Task 3 — render the columns

`internal/serve/templates/board.html`, in `board-body`:

- Emit, per `ChildRow`: `<div class="child lane-row" data-child="…">` containing a child
  heading and `<div class="lanes">` with one `<section class="lane" data-status="<slug>">` per
  lane. Each lane renders a sticky heading (status name, count, and the `count/limit` WIP badge
  where `Limit` is non-zero) and a `<ul>` of the existing `ticket-row` items.
- Then render `Sections` exactly as today.
- Add `data-search="{{.Search}}"` to every ticket `<li>` in both the lanes and the sections.
- Use the existing `slug` template func for `data-status`; do not introduce status-name
  literals.

#### Task 4 — style the columns

`internal/serve/static/styles.css`, in a new `/* ---- board lanes (T-104) ---- */` block:
horizontal flex row with `overflow-x: auto`; each `.lane` a bordered `--panel-2` column with a
sensible min-width, its own `max-height` + `overflow-y: auto`, and a `position: sticky` heading.
Scope a `.lane .ticket-row` override to stack its contents vertically (the board-width row rule
`flex: 1 1 24rem` on `.ticket-title` wraps badly at column width). Change no existing rule
except where task 6 requires it.

#### Task 5 — search box + composed filtering

- `internal/serve/templates/board.html`: add a search `<input id="board-search" type="search">`
  next to the filter bar, inside the persistent `<main class="page">` and **outside** `#board`
  (decision 6).
- `internal/serve/static/board-filter.js`: extend `apply()` to evaluate both dimensions — hide
  `.child[data-child]` blocks failing the child filter (as today), and hide `[data-search]`
  items whose attribute does not contain the lowercased query. Mirror the query onto the page
  element as `data-query` on each `input` event, and keep the existing `htmx:afterSwap`
  re-application (which now restores both dimensions after the 5s swap).
- Update the file's header comment to describe both filters.

#### Task 6 — absorb T-055 (at-limit WIP badge)

`internal/serve/static/styles.css`: `.count` is declared after `.wip-full`, so a
`<span class="count wip-full">` renders muted and the at-limit warning never shows. Fix by
specificity, not order: add `.count.wip-full { color: var(--warn); font-weight: 600; }`.
Applies to the health banner, the new lane headings and the surviving sections alike.

#### Task 7 — tests

`internal/serve/serve_test.go`:

- `TestBoardLaneLayout`: `/` renders four `.lane` sections in lifecycle order with the expected
  `data-status` slugs; the lane row carries `class="child"`+`data-child`; TO DO/DONE/DROPPED
  still render as `section.status`; no active state renders twice.
- `TestBoardSearchMarkup`: the input exists on `/`, is **absent** from `/fragments/board`
  (proving it survives the swap), and every ticket item in both the page and the fragment
  carries `data-search` containing the lowercased id.
- `TestWIPBadgeHighlightedAtLimit`: a fixture at its dev limit renders `count wip-full`, and the
  stylesheet serves `.count.wip-full` (the T-055 regression).
- Extend `TestBoardPage`/`TestBoardFilterBar` where they assert the old all-sections shape.

### Acceptance test

```
just build && just test && just lint && just docs-check

# serve resolves its root from pickle.toml in the cwd; it is read-only, so running it
# against this repo is within the self-modify policy.
./pickle serve --addr 127.0.0.1:8770 >/tmp/t104.log 2>&1 & echo $! >/tmp/t104.pid
sleep 1
curl -s http://127.0.0.1:8770/ | grep -c 'class="lane"'            # expect 4
curl -s http://127.0.0.1:8770/ | grep -q 'id="board-search"' && echo "SEARCH INPUT OK"
curl -s http://127.0.0.1:8770/fragments/board | grep -q 'id="board-search"' \
  && echo "BUG: input inside the polled fragment" || echo "INPUT OUTSIDE FRAGMENT OK"
curl -s http://127.0.0.1:8770/ | grep -q 'data-search="t-104' && echo "SEARCH ATTR OK"
curl -s http://127.0.0.1:8770/static/styles.css | grep -q '\.count\.wip-full' && echo "T-055 OK"
kill "$(cat /tmp/t104.pid)"
```

Expect `4`, then `SEARCH INPUT OK`, `INPUT OUTSIDE FRAGMENT OK`, `SEARCH ATTR OK`, `T-055 OK`.

**Mandatory visual sign-off (decision 10) — the ticket may not move to IN REVIEW without it.**
With the server running, confirm in a browser, in **both** light and dark:

1. The four columns read left-to-right READY → IN DEVELOPMENT → IN REVIEW → REWORK, side by
   side, above TO DO/DONE/DROPPED.
2. Typing in the search box narrows columns *and* the sections below; clearing restores.
3. The typed query and the child-filter selection both survive a 5s refresh (watch for ≥15s).
4. A column with more tickets than fits scrolls inside itself without stretching the row.
5. Present a screenshot of both themes and get explicit user approval before moving the ticket.

### Docs update (mandatory when user-facing)

`docs/user-manual/cli-reference.adoc`, the `/` row of the `pickle serve` page table
(§`[#cmd-serve]`): describe the new shape — active states as per-child columns, the remaining
states stacked below — and document the search box alongside the existing filter-bar sentence.
**Keep** the existing "in exactly the order `BOARD.md` uses … they cannot disagree" guarantee:
decision 3 means it stays true, and this ticket must not silently weaken it. Re-run
`just docs-check`.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated per the step above.
3. Visual sign-off obtained (decision 10) — record it in the ticket.
4. Write the summary: files touched, decisions honoured, anything deferred.
5. Tidy WIP commits into atomic ones (root-path child, rules §0) and suggest a Conventional
   Commit, e.g. `feat(serve): lay the active statuses out as per-child columns (T-104)`.
6. Commit locally on the branch; **do not push or open an MR without explicit approval**.
   Then `pickle ticket move T-104 in-review --reason "acceptance green + visual sign-off"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-15 — created (TO DO). source: chat: user reviewed two rounds of a throwaway static
  mockup (generated from the real tickets/BOARD.md, discussed and iterated live) reshaping the
  serve board page's layout, then asked to file a ticket to build it for real.
- 2026-08-15 — TO DO → READY: plan complete; scope cut to layout+search (no ordering change), retitled, T-055 absorbed
- 2026-08-15 — READY → IN DEVELOPMENT: picked up
