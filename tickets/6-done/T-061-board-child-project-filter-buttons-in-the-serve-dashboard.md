---
id: T-061
title: Board child-project filter buttons in the serve dashboard
project: pickle
depends-on: []
spawned-by: [T-053]
impact: medium
complexity: medium
cost: M
---

# T-061 — Board child-project filter buttons in the serve dashboard

## Description

Add a **flat filter bar** to the serve dashboard's board page (`pickle serve`, T-053): one
button per registered child-project, plus an **All** default, that narrows the board to a single
child's tickets. Today `board.html` always renders every child (`buildBoard` walks
`cfg.Projects` inside each status section — `internal/serve/view.go`), so on a multi-child
workspace the board is a long scroll of mostly-irrelevant `none` blocks. The filter lets the
viewer collapse to just the child they care about.

**Behaviour**

- Bar sits between the health banner and the first status section (the band circled in the
  reference mock), on both the full page and the polled fragment.
- Buttons: `All` (active by default) then one per child, in `cfg.Projects` order. Each button
  shows a small count chip (that child's total ticket count; `All` shows the board total).
- Selecting a button shows only that child's `.child` blocks across every status section;
  `All` restores. Status-section headings/counts stay put — only child blocks hide.
- The board fragment re-renders on a 5s htmx poll (`hx-get="/fragments/board"`,
  `hx-swap="outerHTML"` in `board.html`). **The active selection must survive the swap** — a
  naive client-only toggle resets to `All` every 5s. Refinement picks the mechanism (query-param
  filter echoed into `hx-get`, a persistent selection re-applied on `htmx:afterSwap`, or a
  body-level class + CSS); call it out as an open decision.

**Design — match the reference mock exactly in *shape*, source palette + type from existing
tokens.** A working static mock lives at `board-filter-mock.html` (repo root, dark-theme). It is
the spec for geometry: rounded pill buttons (~7px radius), the `Show` uppercase label, generous
horizontal padding, a bordered count chip per button, and the active state = accent tint fill +
accent border + accent text. **Do not copy the mock's hardcoded colors or its monospace font** —
the dashboard is already theme-aware (T-054: `prefers-color-scheme` light/dark via CSS custom
properties in `internal/serve/static/styles.css`) and uses a sans-serif body stack. Implement the
mock's shapes with the existing tokens (`--accent`, `--panel-2`, `--line`, `--muted`, `--fg`) so
the bar looks native in **both** light and dark, and inherits the dashboard font — not the mock's
mono. The mock is a visual target, not a drop-in stylesheet.

**Touched surface (indicative — refinement pins exact edits):**
`internal/serve/templates/board.html` (filter bar markup + `data-child` on each `.child` block),
`internal/serve/static/styles.css` (button/chip/active styles from tokens), possibly
`internal/serve/view.go`/`funcs.go` (expose the child list + per-child totals to the template)
and a small script in `internal/serve/static/` for the toggle if a JS mechanism is chosen. Tests
in `internal/serve/serve_test.go`.

**Soft coupling:** builds on T-053 (serve dashboard, done) and shares its board view with T-054
(theming), T-055 (WIP badge), T-056 (writable dashboard). No hard `depends-on:` — all are done or
independent.

## Implementation Plan

**Branch (in pickle):** `feat/T-061-board-filter-buttons`, from `main`.

**Prerequisites:** none — T-053 (serve dashboard) is in `6-done/`; no hard `depends-on:`.

**Confirmed decisions:**

1. **Persistence across the 5s htmx poll.** The filter bar is rendered *once*, in
   `board.html` **outside** the polled `#board` div (inside `<main class="page">`, before
   `#board`). Buttons and their active state therefore survive every `outerHTML` swap. The
   selected child is held as `data-filter="<child|all>"` on `main.page`. A small script
   (`static/board-filter.js`, loaded only by `board.html`) re-applies the filter to the freshly
   swapped `.child` blocks on `htmx:afterSwap`. **Static per-child CSS is not viable** — child
   names are dynamic (any registered project), so hiding is done in JS (`el.hidden`), not a
   hand-written CSS rule per name.
2. **No server-side filtering, no query param.** All children are always rendered; the filter is
   pure client-side show/hide. Keeps the fragment routes and `buildBoard` output identical to a
   reload (preserves the T-053 "fragment == reload" invariant).
3. **Chip counts are per-child totals**, computed at page render. Because the bar sits outside
   `#board`, the chip counts refresh on full reload, not on the 5s poll — **accepted limitation**
   (section counts inside `#board` stay live; the chip totals are near-static). Note it in the
   finish summary.
4. **Palette + type from existing tokens, shape from the mock.** No hardcoded colors; active
   state uses `color-mix(in srgb, var(--accent) …)` so it is theme-aware in light and dark
   (T-054). Button label + count use `var(--mono)` to echo `.child-heading`; the `Show` label is
   uppercased muted. Rounded 7px pills, count chip bordered — matching `board-filter-mock.html`.

**Tasks:**

1. `internal/serve/view.go` — add `ChildFilter struct { Name string; Count int }` and a
   `Children []ChildFilter` field on `BoardView`. In `buildBoard`, after the section loop, fill
   `view.Children` by iterating `cfg.Projects` in order and counting tickets whose
   `t.Project() == p.Name` across the whole tree. (`view.Total` already holds the board total.)
2. `internal/serve/templates/board.html` — in the `board.html` define, wrap with
   `<main class="page" data-filter="all">`; before `#board` add a `.filter-bar` (role="group",
   aria-label) with an `All` button (`data-child="all"`, chip `{{.Board.Total}}`) then
   `{{range .Board.Children}}` a `<button class="filter-btn" data-child="{{.Name}}">` with chip
   `{{.Count}}`. Mark the `All` button `is-active` by default. Add
   `<script src="/static/board-filter.js" defer></script>` before `{{template "foot" .}}`.
3. `internal/serve/templates/board.html` (board-body) — add `data-child="{{.Child}}"` to the
   `<div class="child">`.
4. `internal/serve/static/board-filter.js` — new file: guard on `.page[data-filter]`; click
   delegation on `.filter-btn` (set `data-filter`, toggle `is-active`, apply); `apply()` sets
   `el.hidden` on `.child[data-child]` not matching the current filter (`all` shows all);
   re-`apply()` on `document.body` `htmx:afterSwap`; run once on load.
5. `internal/serve/static/styles.css` — add a `/* ---- board child filter ---- */` section:
   `.filter-bar` (flex, wrap, gap), `.filter-label` (uppercase muted), `.filter-btn` (transparent,
   1px transparent border, 7px radius, mono, `--muted`), `:hover` (`--panel-2`, `--fg`),
   `.filter-btn.is-active` (accent text + `color-mix` accent border/tint), `.filter-count`
   (bordered chip on `--panel-2`).
6. `internal/serve/serve_test.go` — add `TestBoardFilterBar`: GET `/` contains a `.filter-bar`,
   an `All` button, a `data-child="demo"` button, and each child block carries
   `data-child="demo"`; GET `/static/board-filter.js` returns 200. Assert the fragment
   (`/fragments/board`) does **not** contain `filter-bar` (it lives outside `#board`) — locking
   decision 1.

**Acceptance test:**

```
just build && just test
# serve resolves its root from pickle.toml in the cwd, so run from the repo root.
# serve is read-only (no install|upgrade|uninstall) → within the self-modify policy.
./pickle serve --addr 127.0.0.1:8765 >/tmp/t061.log 2>&1 & echo $! >/tmp/t061.pid
sleep 1
curl -s http://127.0.0.1:8765/ | grep -q 'class="filter-bar"' && echo "FILTER BAR OK"
curl -s http://127.0.0.1:8765/ | grep -q 'data-child="pickle"' && echo "CHILD BUTTON OK"
curl -s http://127.0.0.1:8765/static/board-filter.js | grep -q 'htmx:afterSwap' && echo "SCRIPT OK"
curl -s http://127.0.0.1:8765/fragments/board | grep -q 'filter-bar' && echo "LEAK (bad)" || echo "FRAGMENT CLEAN OK"
kill "$(cat /tmp/t061.pid)"
```

Confirm the four `OK` lines print and `LEAK` does not.
Manual visual check against `board-filter-mock.html` in a browser (light + dark) is optional but
recommended.

**Docs:** if `docs/` documents `pickle serve`'s UI, add one line noting the child filter; else
none (feature is self-evident in the UI). Confirm `just docs-check` stays green.

**Finish:** run `just build && just test && just lint && just docs-check` green; write the summary
(note the chip-count-refresh limitation from decision 3); prepare commit
`feat(serve): add board child-project filter buttons (T-061)`; commit locally on the branch; do
**not** push/open MR without approval; `pickle ticket move T-061 in-review`.

## Review

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — skipped: no docs-readability reviewer reachable in this Claude Code session (step 4b conscious skip)
- [x] Findings recorded with severity + disposition; disposition summary present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval; bookkeeping committed (step 9)

**Reviewed on branch** `feat/T-061-board-filter-buttons` (pickle), commits `c454d0d` (feature)
+ `887eae4` (inline docs fix, this review).

### Implementation audit

All six tasks met against the tree; every acceptance criterion met.

- **Task 1** (`view.go`: `ChildFilter` + `Children` on `BoardView`, filled from `cfg.Projects`
  with whole-tree per-child counts) — met (`internal/serve/view.go:59-73,117-129`).
- **Task 2/3** (`board.html`: `<main data-filter="all">`, `.filter-bar` outside `#board`, `All`
  + per-child buttons with count chips, `data-child` on `.child` blocks) — met
  (`internal/serve/templates/board.html`).
- **Task 4** (`board-filter.js`: `.page[data-filter]` guard, delegated click, `el.hidden` apply,
  re-apply on `htmx:afterSwap`, run on load) — met.
- **Task 5** (`styles.css` filter section from tokens; active state via `color-mix` on
  `--accent`, mono font, 7px pills, bordered chip) — met; theme-aware, no hardcoded mock colors
  (honours decision 4 + T-054).
- **Task 6** (`TestBoardFilterBar` + static-asset assertion) — met.
- **Acceptance test** re-run verbatim: `FILTER BAR OK`, `CHILD BUTTON OK`, `SCRIPT OK`,
  `FRAGMENT CLEAN OK`; no `LEAK`. `just build && just test && just lint && just docs-check` all
  green.
- **Confirmed decisions** 1–4 all honoured: bar rendered once outside the polled `#board`
  (locked by the fragment-clean test), no server-side filtering, chip counts computed at render,
  palette from tokens.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|----|----------|-------------|-------------|----------|------------|
| F1 | non-blocking | fixed inline | The CLI-reference enumerates the `/` board page's contents; this branch added the visible filter bar, leaving that enumeration incomplete. | `docs/user-manual/cli-reference.adoc` `/` row | Added one line describing the child filter + its survival across the 5s refresh (commit `887eae4`). |
| F2 | non-blocking | noted | Filter buttons carry no `aria-pressed`; screen readers get the visual `is-active` class but no announced pressed state. | `board.html` `.filter-btn`; `board-filter.js` toggles class only | If serve a11y is ever revisited, mirror `is-active` with `aria-pressed`. Not scheduled on its own. |

**Disposition summary:** 2 non-blocking findings — 1 fixed inline (F1), 1 noted (F2). 0 blocking.
No follow-up ticket minted.

### Notes

- Chip-count staleness across the 5s poll is a plan-accepted limitation (decision 3), not a
  finding — the bar sits outside `#board`, so chip totals refresh on full reload while section
  counts stay live.
- No-JS degradation shows all children — acceptable for a read-only local dashboard.

## History

- 2026-07-29 — created (TO DO). source: user request + reference mock (`board-filter-mock.html`, repo root); spawned from T-053 serve dashboard
- 2026-07-29 — TO DO → READY: plan complete
- 2026-07-29 — READY → IN DEVELOPMENT: picked up
- 2026-07-29 — IN DEVELOPMENT → IN REVIEW: acceptance green: filter bar renders, survives poll, fragment clean
- 2026-07-29 — IN REVIEW → DONE: validated: 2 non-blocking findings (1 fixed inline, 1 noted); acceptance green
- 2026-07-29 — merged to main (PR #8, 8edb0e6), user-approved as an exception; squashed
