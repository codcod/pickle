---
id: T-054
title: pickle serve: follow the system light/dark theme instead of hardcoding dark
project: pickle
depends-on: []
spawned-by: []
impact: low
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

The four questions this ticket was filed with are now answered — see *Confirmed design
decisions* below (mechanism: `@media`; fallback: dark; palette: authored, contrast-verified;
acceptance: a structural test plus a measured contrast table and a two-mode manual check).

The refinement sweep confirmed the Description's central claim: `styles.css` contains **no
colour literal outside the `:root` block** — `grep -nE '#[0-9a-fA-F]{3,8}|rgb|hsl|white|black'`
matches lines 4-12 and nothing else, and the templates carry no `style=` attribute and no
colour at all. So a variable-override block genuinely is the whole change.

Soft couplings: T-053 (done, merged — it built this surface); its noted finding **N6** also
touches `internal/serve` but is unrelated to styling. No hard dependency.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the target child and its path is `.` — the overarching repo *is* the child, so
the branch is cut here:

```
git checkout main
git checkout -b feat/T-054-serve-system-theme
```

Commit locally as you go. **Never push or open a merge request without explicit user
approval** (the project's commit policy): end with a summary and a suggested commit message,
and only finalize/push/open the MR once the user approves. Merging is the human's.

### Prerequisite gate (hard)

- **Commit the flow bookkeeping on `main` first** (amendment F6): the READY move, the
  applicability-audit amendments and T-055 are uncommitted, and branching over them would
  drag ticket bookkeeping into the feature diff. `git add tickets/ && git commit` on `main`
  *before* `git checkout -b`. Explicit pathspecs only — never `git add -A`.
- Clean tree on `main` before branching; T-053 is in `6-done/` **and merged** (PR #2,
  `8c33f5c`), so the surface this ticket restyles exists on `main`.
- No `depends-on:` entries. `3-in-development/` must be empty for `pickle` (WIP ≤ 1) — the
  directory not existing satisfies this vacuously; `pickle ticket move` creates it.

### Confirmed design decisions (do not deviate without asking)

1. **Mechanism: a `@media (prefers-color-scheme: light)` block that overrides the custom
   properties.** Not `light-dark()`. The dark `:root` values stay byte-identical to what
   T-053 shipped, so light is a pure addition and **no authored dark colour can regress**
   (UA-painted chrome does intentionally change — see decision 2 and amendment F3).
2. **Dark is the base; light applies only when the browser asks for it** (amended, F1).
   `prefers-color-scheme: no-preference` was **removed** from Media Queries L5 in 2019: a UA
   with no expressed preference reports **`light`**, so the light block *does* fire for it.
   Dark therefore remains the fallback only where the query is unsupported entirely — which
   is the honest and complete guarantee CSS can give. Distinguishing "no preference" from
   "prefers light" would need JavaScript and client state, which decision 3 forbids.

   Consequently **do not use `color-scheme: dark light` list-order as a default**: its own
   no-preference default is dark, which would pair the *light* palette with *dark*-painted
   scrollbars and `<details>` marker on a white page. Declare the scheme per block instead,
   so it tracks exactly the same condition as the palette:

   ```css
   :root { color-scheme: dark; /* … */ }
   @media (prefers-color-scheme: light) { :root { color-scheme: light; /* … */ } }
   ```
3. **Automatic only.** No toggle, no persistence, no cookie, no `--theme` flag, no
   JavaScript. The server stays read-only and stateless (T-053 decision 1) and the
   stylesheet stays hand-written with no build step (`styles.css:1`).
4. **The light palette is authored, not inverted.** Elevation reverses with the medium: in
   dark, panels are *lighter* than the page; in light, panels are *whiter* and the page
   carries the tint. So `--bg` becomes a soft grey and `--panel` becomes white — not the
   dark values run through a filter.
5. **The three semantic colours get genuine light-mode counterparts.** `--accent` `#7cc4a4`,
   `--warn` `#e0b166` and `--error` `#e0736b` are tuned for a near-black ground and sit
   around 1.9:1–3:1 on white — unreadable. They are re-picked for light, not reused.
6. **Contrast target: WCAG AA (≥ 4.5:1) for every colour used as text**, against the
   background it is actually rendered on. Applies to `--fg`, `--muted`, `--accent` (links),
   `--warn` (`.wip-full`, `.finding-warn`) and `--error` (`.finding-error`) in **both**
   palettes — dark is measured too, not assumed.
7. **Variable parity is a maintained invariant, enforced by a test.** Every custom property
   the light block overrides must exist in `:root`, and every *colour* property in `:root`
   must be overridden in the light block. `--mono` is a font stack, not a colour, and is
   deliberately not overridden — the test must special-case it by name rather than by
   guessing.
8. **No change to any Go file other than `serve_test.go`.** No new dependency, no new route,
   no template restructuring beyond the single `<meta>` tag in Task 2.

### Tasks

#### Task 1 — light palette in `internal/serve/static/styles.css`

Leave lines 3-14 (`:root`) exactly as they are apart from **adding one declaration** to it
(amended per F1 — `dark`, not `dark light`):

```css
  color-scheme: dark;   /* the light block flips this in step with the palette (decision 2) */
```

Then append, immediately after the closing `}` of `:root` and before `* { box-sizing … }`, a
light override block. Use this palette — it is the one whose contrast is recorded in the
acceptance test, so changing a value means re-measuring it:

```css
/* The same variables, re-authored for a light ground (T-054). Elevation inverts:
   the page carries the tint and panels go white. The three semantic colours are
   re-picked, not reused — the dark ones sit near 2:1 on white. */
@media (prefers-color-scheme: light) {
  :root {
    color-scheme: light;
    --bg: #f4f5f7;
    --panel: #ffffff;
    --panel-2: #eceef1;
    --line: #d7dae0;
    --fg: #1a1d23;
    --muted: #5f6570;
    --accent: #1f7a55;
    --warn: #8a5700;
    --error: #b3261e;
  }
}
```

`--warn` is `#8a5700`, **not** the `#9a6100` this ticket was filed with: amendment F2 —
`#9a6100` measures **4.42:1** on `--panel-2`, failing decision 6 on the very pair
`.wip-full`/`.finding-warn` render against. `#8a5700` gives 5.25:1 there.

Two consequences worth checking by eye rather than assuming (they are why the palette is
authored per decision 4):

- `.body pre` (`styles.css:109-112`) paints code blocks with `--bg` inside a `--panel`
  card. In dark that recesses them; in light it makes them a grey block on white — still
  correct, still bordered. Confirm it reads as recessed and not as an artifact.
- `.health` (`styles.css:43-48`) uses `--panel-2` while `.site-header` uses `--panel`. In
  light that is grey-on-white rather than dark's lighter-on-darker; confirm the banner is
  still visibly a distinct band.

Change nothing else in the file. Do not reorder or reformat the existing rules — the diff
should be one added line plus one added block.

#### Task 2 — declare the scheme in `internal/serve/templates/layout.html`

In the `head` block, immediately after the `viewport` meta (line 6), add:

```html
<meta name="color-scheme" content="dark light">
```

This is not redundant with the CSS: it lets the browser paint its own surfaces — scrollbars,
the `<details>` disclosure marker in the health banner, focus rings, form controls — for the
right scheme *before* the stylesheet loads, and it survives a stylesheet that fails to load.
Here `dark light` is correct and stays: the meta declares which schemes the document
*supports* for that pre-CSS paint, and dark first is the right guess before the stylesheet
arrives. The authoritative per-scheme value is the CSS `color-scheme` from Task 1, which
overrides it (amendment F1).

#### Task 3 — a structural regression test in `internal/serve/serve_test.go`

There is no browser in CI, so the test asserts the two facts that can actually be checked
mechanically and that a future edit would plausibly break. Add
`TestStylesheetServesBothPalettes`, served over the real route (not read from disk — the
point is that the *embedded, served* asset carries both):

1. `GET /static/styles.css` is 200 and both regions declare a `color-scheme` — `dark` in the
   base, `light` in the media block (amended per F1; the old single
   `color-scheme: dark light` assertion is gone).
2. It contains a `@media (prefers-color-scheme: light)` block.
3. **Parity (decision 7):** the set of `--*` custom properties declared in the top-level
   `:root` block, minus `--mono`, equals the set declared inside the light block. Report the
   symmetric difference by name on failure — "you added `--foo` to one palette and not the
   other" is the exact regression this catches, and the message should say so.

Keep the existing `{"/static/styles.css", "--accent"}` case in
`TestStaticAssetsAndHealthz` (`serve_test.go:259`) — it still holds and still proves the
sheet is served.

**Implementation note — get the region extraction right (amendment F4).** A small regexp is
sufficient and appropriate; do **not** add a CSS parser dependency. But two traps make a
careless version pass vacuously:

- After Task 1 the sheet contains **two** `:root` blocks. A global `:root\s*\{[^}]*\}` match
  therefore collects both regions and the parity check degenerates to `A ⊇ B`, which is
  trivially true. Take the **base** region as the text *before* the first `@media`, and the
  **light** region as the text inside the media block only.
- The light region contains a **nested** `:root { … }`, so its extent cannot be found by
  scanning to the first `}` (that only works by accident of the current formatting). Count
  braces from `@media (prefers-color-scheme: light)`.

Normalise whitespace before asserting declarations, so `color-scheme:dark` and
`color-scheme: dark` are both accepted — match `color-scheme\s*:\s*dark`, not a literal.
Existing helpers suffice: `get(t, newHandler(t, standardTree(t)), "/static/styles.css")`.

#### Task 4 — measure the contrast (not committed as code)

Compute the WCAG contrast ratio for every text colour against the surface it is rendered on,
in **both** palettes, with a throwaway script (`/tmp`, not committed — decision 8 keeps the
tree free of a contrast library for one stylesheet). Pairs to measure:

| palette | pair |
|---|---|
| both | `--fg` on `--bg`, `--fg` on `--panel` |
| both | `--muted` on `--bg`, `--muted` on `--panel`, `--muted` on `--panel-2` |
| both | `--accent` on `--bg`, `--accent` on `--panel` (links, `.merged`, `.body h3/h4`) |
| both | `--accent` on `--panel-2` — a link wrapping inline code (`styles.css:25` + `:113`) (F5) |
| both | `--warn` on `--panel-2` (`.wip-full`, `.finding-warn` sit in the health banner) |
| both | `--warn` on `--bg` — `board.html:25` puts the WIP badge in `.child-heading`, on the page ground, not the banner (F5) |
| both | `--error` on `--panel-2` (`.finding-error`) |
| both | `--error` on `--panel`, `--warn` on `--panel` — the grade chips (`styles.css:88-89`) use them as text *and* border |

Record the resulting table in the ticket's summary. **Any pair below 4.5:1 is a defect
(decision 6)** — adjust that colour and re-measure rather than shipping it, and note the
adjustment in the summary. The applicability audit already measured this table
independently; the pinned palette above is the corrected one, so the expected result is that
every pair passes. A failure means the palette was mistyped.

Two known-thin margins to report rather than "fix" (F11, noted): light `--accent` lands at
4.85 on `--bg` and 4.55 on `--panel-2` — passing, with little headroom, so any later tweak
to `--bg`/`--panel-2` can break AA silently. Say so in the summary for the next editor.
`--line` borders sit at 1.28–1.40 against their grounds in **both** palettes, below WCAG
1.4.11's 3:1 for non-text contrast; that is pre-existing, purely decorative chrome, and out
of scope here (F9).

#### Task 5 — docs

- `docs/user-manual/cli-reference.adoc`, the `[#cmd-serve]` section: add one short paragraph
  after the refresh paragraph that ends at line 437, before the audit-banner paragraph at
  line 439. State that the dashboard follows the operating system's light/dark preference
  automatically and that there is deliberately no toggle. **Do not repeat the retracted
  no-preference claim** (F1): say that dark is what a browser gets unless it asks for light.
  Keep it to two or three sentences in the manual's voice.
- `CHANGELOG.md`, under `## [Unreleased]`: a `### Changed` entry (the section already
  exists at line 39, holding T-049) — "the `pickle serve` dashboard now follows the system
  light/dark preference (T-054)", noting that the dark palette is unchanged, so a dark-mode
  user's view does not move.
- Nothing in `README.md`, `AGENTS.md` or the skill payload: this ships no new command, flag
  or flow rule.

### Acceptance test

Run from the repo root on `feat/T-054-serve-system-theme`.

1. **Build and validate:**
   ```
   just build && just test && just lint && just docs-check
   ```
   All four green. Read `snowball check`'s *output*, not only its exit code (the standing
   amendment from T-053's review, finding F14).

2. **The new test discriminates.** Prove it fails when the invariant is broken, so it is not
   a tautology:
   ```
   go test ./internal/serve/ -run TestStylesheetServesBothPalettes -v
   ```
   green; then temporarily add `--probe: #000;` to the `:root` block only, re-run, and
   confirm it **fails naming `--probe`**; revert.

3. **Served-asset check** (proves the embedded binary carries it, not just the working tree):
   ```
   ./pickle serve --addr 127.0.0.1:8749 &
   sleep 1
   curl -s 127.0.0.1:8749/static/styles.css | grep -c "prefers-color-scheme: light"   # 1
   curl -s 127.0.0.1:8749/static/styles.css | grep -c "color-scheme: dark;"           # 1
   curl -s 127.0.0.1:8749/static/styles.css | grep -c "color-scheme: light;"          # 1
   curl -s 127.0.0.1:8749/ | grep -c 'name="color-scheme"'                            # 1
   kill %1
   ```
   Run this **interactively** — `&` and `kill %1` need job control and will not behave in a
   non-interactive script (F12, noted).

4. **Read-only unchanged:**
   ```
   go test ./internal/serve/ -run TestServeNeverWrites
   ```
   green — the styling change must not have disturbed T-053's central guarantee.

5. **Manual, both modes** (the only check that proves it actually *looks* right; there is no
   browser in CI, so this is done by hand and reported):
   with `pickle serve` running, view `/`, `/t/T-054` and `/activity` with macOS in **Dark**
   and then in **Light** (System Settings → Appearance). In each mode confirm: body text and
   muted text are comfortably readable; links, the `merged` marker and `.body h3/h4` are
   legible; the health banner is a distinct band from the header; a ticket page's code blocks
   and inline code are distinguishable from the card; grade chips with
   `impact-critical`/`impact-high` are legible.

   Then, on dark mode specifically (**restated per amendment F3** — the original
   "visually identical to `main`" was unsatisfiable, since Task 1 and Task 2 exist precisely
   to change UA-painted surfaces in dark mode): confirm that every **authored** colour is
   byte-identical to `main` — no `:root` value changed, so nothing painted by our stylesheet
   may differ — while the **browser-painted** chrome deliberately flips from light to dark.
   Verify that flip *happened* rather than that it didn't: the scrollbars on a ticket page's
   wide `<pre>`/`<table>` (`styles.css:111,115`) and the health banner's `<details>` marker
   (`layout.html:47`) should now be dark-painted where on `main` they are light. That is the
   `color-scheme` bug T-054 fixes in dark mode, independent of light theming.

6. **Board clean:** `pickle board audit` → 0 errors, 0 warnings.

### Docs update (mandatory when user-facing)

Task 5 above — `docs/user-manual/cli-reference.adoc` `[#cmd-serve]` and `CHANGELOG.md`
`[Unreleased] → Changed`. `just docs-check` must stay green.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated; the measured contrast table (Task 4) recorded in the summary.
3. Write a summary: files touched, the palette shipped, the contrast numbers, and anything
   deferred.
4. Suggested commit message:

   ```
   feat(serve): follow the system light/dark theme (T-054)

   The dashboard's palette was hardcoded dark. It now honours
   prefers-color-scheme with an authored light palette, and declares
   color-scheme per block so the browser paints its own chrome to
   match. Dark's authored colours are unchanged.
   ```

5. Commit locally on the branch. **Do not push or open a merge request without user
   approval.** Present the commit message; after approval, finalize (squash or keep history —
   the user chooses), push, open the MR. Merging is the human's.

## Implementation summary

Built on `feat/T-054-serve-system-theme`, one commit (`5b9edea`). **144 insertions, 0
deletions** across five files — the diff is purely additive, which is the mechanical proof
of decision 1: no authored dark colour could have regressed, because no existing line
changed.

| file | change |
|---|---|
| `internal/serve/static/styles.css` | `color-scheme: dark` added to `:root`; light `@media` block appended (+26) |
| `internal/serve/templates/layout.html` | `<meta name="color-scheme" content="dark light">` (+1) |
| `internal/serve/serve_test.go` | `TestStylesheetServesBothPalettes` + `cssVars`/`braceBlock`/`cssVarDecl` helpers (+96) |
| `docs/user-manual/cli-reference.adoc` | one paragraph in `[#cmd-serve]` (+5) |
| `CHANGELOG.md` | `[Unreleased] → Changed` entry (+16) |

Palette shipped (the F2-corrected one): `--bg #f4f5f7`, `--panel #ffffff`,
`--panel-2 #eceef1`, `--line #d7dae0`, `--fg #1a1d23`, `--muted #5f6570`,
`--accent #1f7a55`, `--warn #8a5700`, `--error #b3261e`.

### Measured contrast (Task 4) — WCAG 2.1, computed, not estimated

| pair | dark | light | rendered at |
|---|---|---|---|
| `--fg` on `--bg` | 14.76 | 15.48 | `styles.css:21` body |
| `--fg` on `--panel` | 13.61 | 16.88 | `:82,105` ticket title, body card |
| `--fg` on `--panel-2` | 12.53 | 14.52 | `:113` inline code |
| `--muted` on `--bg` | 6.49 | 5.37 | `:63,64,71,134` lede, crumbs, headings, footer |
| `--muted` on `--panel` | 5.99 | 5.86 | `:38,86,102` brand-sub, grade chip, meta dt |
| `--muted` on `--panel-2` | 5.51 | 5.04 | `:52` `.health .wip` |
| `--accent` on `--bg` | 8.87 | 4.85 | `:25` links |
| `--accent` on `--panel` | 8.18 | 5.29 | `:25,91,108` links, `.merged`, h3/h4 |
| `--accent` on `--panel-2` | 7.53 | 4.55 | `:25`+`:113` link wrapping inline code (F5) |
| `--warn` on `--bg` | 9.19 | 5.59 | `board.html:25` WIP badge (F5) |
| `--warn` on `--panel` | 8.47 | 6.10 | `:89` impact-high chip, text + border |
| `--warn` on `--panel-2` | 7.80 | 5.25 | `:53,56` `.wip-full`, `.finding-warn` (F2) |
| `--error` on `--panel` | 5.43 | 6.54 | `:88` impact-critical chip, text + border |
| `--error` on `--panel-2` | 5.00 | 5.62 | `:55` `.finding-error` |

**Every text pair ≥ 4.5:1 in both palettes.** The figures reproduce the applicability
audit's independently, including F2's 4.42 → 5.25 correction for `--warn`. Measured with a
throwaway script (`/tmp`, not committed, per decision 8).

For the next editor (F11): light `--accent` has the thinnest margins in the sheet — 4.85 on
`--bg`, 4.55 on `--panel-2`. Any future tweak to `--bg` or `--panel-2` can drop it below AA
silently; re-measure if you touch them. `--line` borders sit at 1.28–1.40 against their
grounds in **both** palettes, under WCAG 1.4.11's 3:1 — pre-existing, decorative, untouched
(F9).

### Acceptance test results

| step | result |
|---|---|
| 1. `just build && just test && just lint && just docs-check` | all green; `snowball check` **output** read, not just its exit code (T-053 amendment F14) — no warnings |
| 2. New test discriminates | passes; `--probe` in `:root` only → fails naming `--probe`; `--probe-light` in the light block only → fails naming it. **Both directions** verified, so the parity check is not the tautology F4 warned about |
| 3. Served-asset check | `prefers-color-scheme: light` ×1, `color-scheme: dark;` ×1, `color-scheme: light;` ×1 in the served sheet; the meta present on `/`, `/t/T-054` **and** `/activity` |
| 4. `TestServeNeverWrites` | green — the read-only guarantee is undisturbed |
| 5. Manual, both OS modes | **NOT DONE — requires a human at a browser.** See below |
| 6. `pickle board audit` | 55 tickets, 0 errors, 0 warnings |

`internal/serve` coverage holds at 92.4%.

### Outstanding: acceptance step 5 is unverified

Step 5 is the only check that proves the dashboard actually *looks* right, and it cannot be
automated — it needs a human switching macOS between Dark and Light with the pages open. It
is **not** ticked. The reviewer must run it before this ticket can be called done:

1. `./pickle serve`, then view `/`, `/t/T-054` and `/activity` in **Light** mode. Confirm
   body and muted text read comfortably; links, the `merged` marker and `.body h3/h4` are
   legible; the health banner is a distinct band from the header; code blocks and inline
   code are distinguishable from the white card; the `impact-critical`/`impact-high` grade
   chips are legible.
2. Switch to **Dark**. Confirm every authored colour is unchanged from `main` (the additive
   diff says it must be) *and* that the browser-painted chrome has flipped as intended
   (F3): the scrollbars on a wide `<pre>`/`<table>` and the audit banner's `<details>`
   marker should now be dark-painted where `main` paints them light.

Two eyeball checks from Task 1 fold into step 1: that light-mode `.body pre` (`--bg` on a
white card) reads as recessed rather than as an artifact, and that `.health`
(`--panel-2`, grey on white) still reads as a distinct band.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: chat — request that the `pickle serve` dashboard
  follow the system theme for light and dark modes; graded low-medium/low/S (cosmetic, one
  stylesheet, no server-side change)
- 2026-07-28 — applicability audit (fresh sub-agent) before pickup: 12 findings, 3 blocking.
  F1-F6 amended into the plan inline — F1 the no-preference claim was wrong (`no-preference`
  was removed from MQ L5, so a UA with no preference reports `light`; `color-scheme` is now
  declared per block instead of relying on `dark light` list-order, which would have paired
  the light palette with dark-painted chrome), F2 `--warn` `#9a6100` measured 4.42:1 on
  `--panel-2` and failed the plan's own AA gate (repinned `#8a5700`, 5.25:1), F3 the
  "dark identical to main" criterion was unsatisfiable given Task 2 (restated), F4 the
  parity test would have been tautological with two `:root` blocks, F5 two missing contrast
  pairs, F6 commit the flow bookkeeping before branching. F7 promoted to **T-055**
  (pre-existing T-053 defect: `.count` overrides `.wip-full`, so the board's at-limit WIP
  badge never highlights). F8-F12 noted and closed
- 2026-07-28 — TO DO → READY: plan complete; decisions confirmed (@media override, dark as no-preference fallback, authored light palette, AA contrast, no toggle); collapsed impact low-medium -> low (cosmetic)
- 2026-07-28 — READY → IN DEVELOPMENT: picked up; plan amended from the applicability audit (F1-F6)
- 2026-07-28 — IN DEVELOPMENT → IN REVIEW: acceptance green except step 5 (manual two-mode visual check, needs a human); 5 tasks done, palette AA-verified in both modes, diff purely additive
