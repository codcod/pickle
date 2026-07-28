---
id: T-054
title: pickle serve: follow the system light/dark theme instead of hardcoding dark
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-054 — pickle serve: follow the system light/dark theme instead of hardcoding dark

## Description

The dashboard shipped by `pickle serve` (T-053) is **hardcoded dark**: every colour lives in
a single unconditional `:root` block in `internal/serve/static/styles.css:3-14`
(`--bg: #14161a`, `--fg: #e6e8ec`, …). A reader whose OS is in light mode gets a dark page
next to their light terminal and editor, and — because the document never declares a
`color-scheme` — the browser also paints its own UI (scrollbars, form controls, the
`<details>` disclosure marker) for a light document over a dark background.

The dashboard should instead **follow the system preference**: dark stays exactly as it is
today, and a light palette is served when the OS asks for one.

Scope as requested: **automatic only**. Follow `prefers-color-scheme` — no in-page toggle, no
persistence, no `--theme` flag, therefore no client-side state and no cookie. That keeps the
server read-only and stateless (a locked decision of T-053) and keeps the stylesheet
hand-written with no build step (the comment at `styles.css:1`). A manual override is a
separate, later question; note it, do not build it here.

What the change touches:

- `internal/serve/static/styles.css` — the only file that names a colour. The rest of the
  sheet already routes everything through the `--bg`/`--panel`/`--panel-2`/`--line`/`--fg`/
  `--muted`/`--accent`/`--warn`/`--error` variables, so in principle this is a light override
  block plus a `color-scheme` declaration. Refinement must confirm that claim by sweeping the
  sheet for hardcoded colours and for rules that only work on a dark ground — the semantic
  colours in particular (`--accent` `#7cc4a4`, `--warn` `#e0b166`, `--error` `#e0736b` at
  `styles.css:88-91` and `.finding-*`/`.health-*`) will not carry enough contrast on white
  and need light-mode counterparts, not reuse.
- `internal/serve/templates/layout.html` — `<head>` may need `<meta name="color-scheme"
  content="dark light">` so browser-painted UI matches before/independently of the CSS.
- `internal/serve/serve_test.go:259` asserts `/static/styles.css` contains `--accent`; a
  restructured sheet must keep that assertion meaningful (or the test updated deliberately).
- Docs: the user manual's `serve` section, if it characterises the dashboard's appearance.

Open questions for refinement (do not decide here):

1. Which mechanism — a `@media (prefers-color-scheme: light)` override block, or
   `light-dark()` with `color-scheme: light dark`? The latter is one declaration per variable
   but raises a browser-baseline question the project has not had to answer before (the
   dashboard is localhost-only, so the baseline is "whatever the developer runs").
2. Whether dark or light is the fallback when the browser expresses no preference. Dark
   preserves today's behaviour and is the safer default.
3. The light palette itself — it must be authored, not inverted, and must hold contrast for
   the three semantic colours and for `--muted` body text.
4. How the acceptance test proves it, given there is no browser in CI: the honest mechanical
   check is that the served stylesheet contains both palettes and a `color-scheme`
   declaration, plus a manual check in both OS modes.

Soft couplings: T-053 (done, merged — it built this surface); its noted finding **N6** also
touches `internal/serve` but is unrelated to styling. No hard dependency.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: chat — request that the `pickle serve` dashboard
  follow the system theme for light and dark modes; graded low-medium/low/S (cosmetic, one
  stylesheet, no server-side change)
