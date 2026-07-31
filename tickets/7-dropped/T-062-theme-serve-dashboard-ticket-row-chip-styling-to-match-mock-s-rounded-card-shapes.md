---
id: T-062
title: Theme serve dashboard board page to match the mock's framed-card shapes
project: pickle
depends-on: []
spawned-by: [T-061]
impact: low
complexity: medium
cost: M
---

# T-062 — Theme serve dashboard board page to match the mock's framed-card shapes

## Description

`board-filter-mock.html` (repo root) is the visual reference the serve dashboard's board page
(`pickle serve`, T-053) has been partially, but not fully, brought toward. T-061 (board child
filter buttons) deliberately matched the mock's **shape only** for the filter bar — rounded 7px
pills, bordered count chip, accent-tinted active state — while explicitly rejecting the mock's
hardcoded dark palette and monospace font, per that ticket's decision 4: "palette + type from
existing tokens, shape from the mock." That scope never touched the rest of the board: ticket
rows, priority/complexity/cost chips, and the ticket/meta panels still use the pre-T-061,
pre-T-054 shapes.

Comparing a live `pickle serve` screenshot against the mock (prompted by a user side-by-side
review after T-061 shipped) surfaces the gap:

- **Ticket rows** (`.ticket-row`, `internal/serve/static/styles.css:135-139`) use a flat 4px
  border radius on `var(--panel)`. The mock's `.card` (`board-filter-mock.html:134-143`) uses a
  more pronounced 9px radius with more generous padding — a visibly "rounder", more separated
  card, not a thin-bordered strip.
- **Grade chips** (`.grade`, `styles.css:143-146`; rendered via the `grades` template,
  `internal/serve/templates/layout.html:60-64`) are uncolored bordered pills except for the
  `impact-critical`/`impact-high` variants (error/warn border+text). The mock's `.chip` family
  (`board-filter-mock.html:147-156`) has a dedicated `.chip.high` treatment (amber text +
  matching border) and a distinct `.chip.size` (fixed-width, centered) that the real chips don't
  differentiate.
- **Other board panels** — `.meta` and `.body` (ticket detail page, `styles.css:156-176`) share
  the same flat 4px-radius `var(--panel)` treatment as `.ticket-row`, so the same rounding gap
  applies wherever the mock's card language would apply.

**Goal:** bring `.ticket-row`, `.grade`/chip variants, `.meta`, and `.body` closer to the mock's
card geometry (rounder corners, more breathing room, a clearer chip taxonomy for impact levels)
while keeping T-061's decision 4 discipline: reuse existing tokens (`--accent`, `--panel`,
`--panel-2`, `--line`, `--muted`, `--warn`, `--error`), keep the sans-serif body font and the
theme-aware light/dark palette (T-054) — do not adopt the mock's hardcoded dark colors or its
monospace font for prose/body text (mono stays reserved for ids/code per current convention).

**Soft coupling:** spawned by T-061 (board filter buttons, done); builds on the same design
tokens introduced by T-054 (light/dark theming, done) and shares `styles.css`/`board.html` with
T-053 (serve dashboard, done) and T-055 (WIP badge, done — verify its `wip-full` styling still
reads correctly against any radius/padding change). No hard `depends-on:` — all are done.

### Scope expansion (2026-07-31, after first review round)

The first implementation (rounder ticket-row cards + a fixed-width cost chip only) was rejected
on visual review: at a glance the dashboard "doesn't resemble the mock at all." Root cause — the
original scope (decisions 1–2 below) was **too narrow**: the mock's *defining* traits are (a) the
whole page rendered as a **rounded, bordered panel floating on a distinct ground** (`.app`,
`board-filter-mock.html:31-38`: 12px radius, `max-width:1180px`, centered, on a darker `body`),
(b) section-heading **divider rules**, and (c) generous card/chip spacing. Only (c) — partially —
was in the first plan, so the delta was invisible.

The user chose direction **"mock shape, both themes"**: reproduce the mock's *layout* (framed
container + rules + airier cards/chips) in **both** light and dark using the existing tokens, and
**keep** the sans-serif body font and theme-awareness (T-054/T-061 decision 4 stand — the mock's
monospace body and hardcoded dark-only palette are still **not** adopted). This widens the ticket
from "ticket-row/chip styling" to "the board page's frame + card shapes," hence the retitle and
the added decisions 6–7 and tasks 5–7 below.

## Implementation Plan

**Branch (in pickle):** `feat/T-062-card-chip-theming`, from `main`.

**Prerequisites:** none — T-053/T-054/T-061 are all in `6-done/` and merged; no hard
`depends-on:`. `pickle serve` is read-only (no `install|upgrade|uninstall`), so running it
against this repo is within the self-modify policy.

**Confirmed decisions:**

1. **Scope = the board-list surface the mock depicts.** Only `.ticket-row` and the grade chips
   (`.grade`) change. The ticket-detail panels (`.meta`, `.body`, `styles.css:156-176`) are
   **out of scope** — the mock shows only the board list, and "match the mock exactly" therefore
   means matching what the mock renders, not extending its card language to a page it never
   depicts.
2. **Shape from the mock, palette + type from existing tokens** — same discipline as T-061
   decision 4 (and T-054). Adopt the mock's *geometry* (radii, padding, gap, the fixed-width
   size chip, the id column) but **not** its hardcoded colors or its all-mono body: keep
   `var(--panel)`/`var(--line)`/`var(--muted)`/`var(--warn)`/`var(--error)`/`var(--accent)` so
   the rows stay theme-aware in light **and** dark, and keep `var(--mono)` reserved for the
   id/grade text as today (prose stays sans). The mock is authored at a 15px base (its
   `font: 15px/…`), the same base as the dashboard body, so its px values convert to rem 1:1
   (14px ≈ 0.9rem, 18px ≈ 1.15rem, 9px radius, 26px ≈ 1.65rem, 96px = 6rem).
3. **Impact chips need no new colors.** The real grade chips **already exceed** the mock here:
   `impact-critical`/`impact-high-critical` render red (`--error`) and `impact-high`/
   `impact-medium-high` render amber (`--warn`) (`styles.css:147-148`), where the mock only ever
   shows a single amber `.chip.high`. So the Description's "possibly … impact colors beyond
   high/critical" is **dropped** — the only markup change needed is the cost size chip
   (decision 4), not any impact color class.
4. **Cost chip gets the mock's fixed-width `.size` treatment.** The mock's third chip
   (`.chip.size`, `board-filter-mock.html:156`: `min-width:26px; text-align:center`) is the
   M/S/L cost pill. The `grades` template (`internal/serve/templates/layout.html:60-64`) renders
   three generic `.grade` spans, so the cost span is tagged with an extra `size` class and the
   CSS adds `.grade.size`. This is the one template edit.
5. **Keep `flex-wrap: wrap` on `.ticket-row`.** The mock card is a clean three-chip single row;
   real rows carry more (dependency/lineage/family/reason edges — `board.html:44-48`), so
   wrapping stays for content robustness even though the mock card itself does not wrap. Geometry
   (radius/padding/bg/gap) matches the mock; wrap behaviour is retained deliberately.
6. **Framed container (the scope-expansion core).** Wrap the existing header + health banner +
   `<main>` + footer in a single `.app` panel, reproducing the mock's `.app` (`:31-38`) with
   tokens: `max-width: 78rem` (the current `.page` cap, kept), centered with a top/bottom margin,
   `border: 1px solid var(--line)`, `border-radius: 12px`, `overflow: hidden` (so the header's
   and footer's square corners clip to the radius), and `background: var(--bg)`. To make the
   panel read as *floating on a distinct ground* in both themes without inventing a token, the
   `body` background becomes `color-mix(in srgb, var(--bg) 90%, #000)` — a hair darker than the
   panel in dark, a slightly deeper grey margin around the panel in light. The panel's inner
   surface is `var(--bg)` while the ticket cards stay `var(--panel)`, so cards keep popping
   against the panel in both themes (matches the mock: `.app` on `--bg`, `.card` on `--panel`).
   This is a **shared-layout change** (`layout.html` `head`/`foot`) and so affects the ticket and
   activity pages too — intended: they get the same frame, staying consistent.
7. **Section divider rules + title prominence.** The mock separates each status section with a
   full-width rule under an uppercase dim heading (`.sec-h` + `hr.rule`, `:118-126`). The real
   `.status-heading` already carries a full-width `border-bottom` (`styles.css:128-131`) — it
   reads as that rule, so **no structural change**; only bump the brand title to echo the mock's
   prominent 22px header (`.brand-name` `1.05rem → 1.35rem`). No monospace, no palette change
   (decision 2 stands).

**Tasks:**

1. `internal/serve/static/styles.css` — `.ticket-row` (currently `:135-139`): change
   `border-radius: 4px` → `9px`; `padding: 0.4rem 0.6rem` → `0.9rem 1.15rem`;
   `gap: 0.5rem` → `0.9rem`; `margin-bottom: 0.3rem` → `0.65rem`;
   `align-items: baseline` → `center`. Keep `display: flex`, `flex-wrap: wrap`,
   `background: var(--panel)`, `border: 1px solid var(--line)` (decision 5).
2. `internal/serve/static/styles.css` — grade chips: in `.grade` (`:143-146`) change
   `border-radius: 3px` → `5px`, `padding: 0.05rem 0.35rem` → `0.15rem 0.6rem`,
   `font-size: 0.72rem` → `0.75rem` (mock geometry; color/border/mono unchanged). Add two rules
   next to it: `.grade.size { min-width: 1.65rem; text-align: center; }` (mock `.chip.size`, 26px) and
   `.ticket-row > .tid { min-width: 6rem; font-weight: 600; }` — the mock's 96px id column,
   scoped with `>` so inline `.tid` links inside `.edge` dependency lists
   (`layout.html:66` `idlist`) are **not** widened.
3. `internal/serve/templates/layout.html` — in the `grades` define (`:60-64`), change the Cost
   span from `<span class="grade">{{.Cost}}</span` to `<span class="grade size">{{.Cost}}</span`
   (decision 4). Impact and Complexity spans unchanged.
4. `internal/serve/templates/layout.html` — framed container (decision 6): in the `head` define,
   immediately after `<body>` open `<div class="app">`; in the `foot` define, immediately before
   `</body>` close `</div>` (after the footer, so the footer sits inside the panel). Also bump the
   brand title (decision 7) — no markup change needed, just the CSS in task 6.
5. `internal/serve/static/styles.css` — add an `/* ---- app frame (T-062) ---- */` block:
   `.app { max-width: 78rem; margin: 1.5rem auto; background: var(--bg); border: 1px solid var(--line); border-radius: 12px; overflow: hidden; }`
   and change `body`'s `background: var(--bg)` → `background: color-mix(in srgb, var(--bg) 90%, #000)`.
   Drop the now-redundant `max-width`/`margin` from `.page` (`:87`) — the `.app` cap replaces them;
   keep `.page { padding: 1.25rem; }`. Bump `.brand-name` `font-size: 1.05rem` → `1.35rem`
   (decision 7).
6. `internal/serve/serve_test.go` — extend `TestBoardCardStyling` (added in the first round):
   keep the `class="grade size"` / `.grade.size` / `border-radius: 9px` assertions; add that GET
   `/` contains `class="app"` (the frame wraps the page) and GET `/static/styles.css` contains
   `.app {` and `color-mix(in srgb, var(--bg) 90%` (frame + floating-ground reached the served
   asset). Assert nothing about `.meta`/`.body` — the ticket-detail panels stay out of scope.

**Acceptance test:**

```
just build && just test
# serve resolves its root from pickle.toml in the cwd → run from the repo root.
# serve is read-only (no install|upgrade|uninstall) → within the self-modify policy.
./pickle serve --addr 127.0.0.1:8766 >/tmp/t062.log 2>&1 & echo $! >/tmp/t062.pid
sleep 1
curl -s http://127.0.0.1:8766/ | grep -q 'class="grade size"' && echo "SIZE CHIP OK"
curl -s http://127.0.0.1:8766/ | grep -q 'class="app"' && echo "FRAME MARKUP OK"
curl -s http://127.0.0.1:8766/static/styles.css | grep -q '\.grade\.size' && echo "SIZE CSS OK"
curl -s http://127.0.0.1:8766/static/styles.css | grep -q 'border-radius: 9px' && echo "CARD RADIUS OK"
curl -s http://127.0.0.1:8766/static/styles.css | grep -q '\.app {' && echo "FRAME CSS OK"
kill "$(cat /tmp/t062.pid)"
```

Confirm the five `OK` lines print. **Mandatory** visual check against `board-filter-mock.html` in
a browser, **both light and dark** (decision 2) — this round exists because the visual result is
the acceptance criterion: the page must read as a single rounded bordered panel floating on a
distinct ground, section rules present, cards rounder/airier with the fixed-width cost chip, the
amber high-impact chip and `wip-full` badge (T-055) still legible. Capture a screenshot for the
finish summary so the reviewer compares against the mock without re-running serve.

**Docs:** none — the change is purely visual and self-evident in the UI; the CLI-reference `/`
board-page entry (touched by T-061 F1) describes *contents*, not chip geometry, so it needs no
edit. Confirm `just docs-check` stays green.

**Finish:** run `just build && just test && just lint && just docs-check` green; write the
summary (with the screenshot); prepare commit `feat(serve): frame the board page and round its
cards to match the mock (T-062)`; commit locally on the branch; do **not** push/open MR without
approval; `pickle ticket move T-062 in-review`.

## Review

### Round 1 (2026-07-31) — visual, user-driven

| id | severity | disposition | description | evidence | suggestion |
|----|----------|-------------|-------------|----------|------------|
| F1 | **blocking** | → rework (scope widened) | The shipped change (rounder `.ticket-row` + fixed-width cost chip) does not achieve the ticket's goal — the dashboard "doesn't resemble the mock at all." The mock's defining traits (framed panel floating on a distinct ground; section divider rules; airier cards) were mostly out of the round-1 scope, so the visible delta was negligible. | Live `pickle serve` vs `board-filter-mock.html`, user side-by-side. | Widen scope to the framed container + section rules + card/chip spacing in **both** themes via tokens (decisions 6–7, tasks 4–6), keeping sans-serif + theme-awareness. |

**Disposition summary (round 1):** 1 blocking (F1 → `5-rework/`, scope widened per user
decision "mock shape, both themes"); 0 non-blocking. The `superfluous WriteHeader` log the user
saw is **not a finding** — it is the documented-cosmetic T-053 review item N3 (`serve.go:132-144`:
templates stream, so a client hangup mid-response or a mid-stream render error logs it; nothing is
written to disk). It is unrelated to this ticket's CSS/template edits.

### Round 2

<!-- filled after the scope-widened rebuild returns to review -->

## History

- 2026-07-31 — created (TO DO). source: pickle ticket new
- 2026-07-31 — TO DO → READY: plan complete
- 2026-07-31 — applicability gate (pickup): clean — all 7 plan assumptions confirmed against the current tree; 3 non-blocking notes, all note-and-close. Inline amendment: task 2 size chip `1.75rem → 1.65rem` (mock `.chip.size` is 26px; 1.65rem ≈ 26px, 1.75rem was ≈28px).
- 2026-07-31 — READY → IN DEVELOPMENT: picked up
- 2026-07-31 — IN DEVELOPMENT → IN REVIEW: acceptance green: card radius + size chip render, tests pass
- 2026-07-31 — IN REVIEW → REWORK: blocking finding F1 — result does not resemble the mock; scope was too narrow. User chose "mock shape, both themes"; ticket retitled + rescoped (framed container + section rules + card spacing, both themes, sans+tokens; T-054/T-061 decision 4 still stand). Re-graded complexity low→medium, cost S→M.
- 2026-07-31 — IN REVIEW → REWORK: blocking F1: does not resemble mock; scope widened to framed layout (both themes)
- 2026-07-31 — REWORK → IN REVIEW: rework done: framed layout, both themes; 5/5 acceptance + visual verified
- 2026-07-31 — IN REVIEW → DROPPED: reverted per user: both rounds looked worse than the pre-T-062 baseline; no code merged to main; feat branch discarded
