---
id: T-090
title: Harden linkifyURLs: escape/trim ordering, empty-host links, adjacent URLs, and noreferrer
project: pickle
depends-on: []
spawned-by: [T-089]
impact: low-medium
complexity: low
cost: S
---

# T-090 — Harden linkifyURLs: escape/trim ordering, empty-host links, adjacent URLs, and noreferrer

## Outcome

After this ships, a URL in a `pickle serve` merge or History line renders correctly whatever it
contains — a query string with `&`, an odd tail, two URLs side by side — instead of only the
ordinary commit-URL case, and following one leaks no referrer to the external host.

## Description

Spawned by T-089's review, which added `linkifyURLs` (`internal/serve/view.go`) and found six
sharp edges in it. None blocks the golden path a commit URL takes — all were classified
non-blocking — but they are latent and cheap to close together, and T-089's own code comment
already points here. One theme: the helper's robustness. The six items:

1. **Escape-then-trim ordering corrupts entity tails.** `linkifyURLs` escapes with
   `template.HTMLEscapeString` and *then* regex-matches `https?://\S+` on the escaped string,
   trimming `)].,;:` off the match. Because `;` is in that trim set and every escaped
   HTML-special character ends in `;`, a URL whose tail is an entity loses the terminator:
   `https://x.com/a<` → href `https://x.com/a&lt` (broken entity), and
   `https://x.com/<script>` leaves a stray `;` outside the anchor. Not a security hole — an
   attribute's delimiters are fixed by the tokenizer before character references decode, so a
   decoded quote cannot reopen attribute parsing, and `javascript:`/`data:` never match the
   pattern — but the rendered URL is wrong. Fix by trimming the *raw* string and escaping each
   part, or by dropping `;` from the trim set; decide which at refinement.
2. **`trimmed == ""` is dead code.** The pattern requires `https?://` and `/` is not in the trim
   set, so `trimmed` can never be empty. Delete it, or replace it with the guard item 3 needs.
3. **A host-less URL still becomes a link.** `https://).` renders `<a href="https://">` — a
   link to nothing. Require at least one character of host.
4. **Adjacent URLs collapse.** `https://a.com/1,https://b.com/2` yields one anchor whose href is
   both URLs, because `\S+` is greedy and only the tail is trimmed.
5. **No test asserts escaping *inside* a URL run** — the one place the helper is non-obvious, and
   exactly what would have caught item 1. Add cases: `https://x.com/a&b`,
   `https://x.com/<script>`, `https://).`, and two adjacent URLs. (The existing suite is not
   weak — dropping escaping altogether would fail `TestLinkifyURLsEscapesSurroundingText` — just
   narrow.)
6. **`rel="noreferrer"` is absent.** The anchor sets `rel="noopener" target="_blank"`, so
   following a link leaks `Referer: http://127.0.0.1:<port>/t/T-NNN` to the external host. Use
   `rel="noopener noreferrer"`.

Soft coupling: T-089 (the parent) shipped the helper and the merge-line convention it serves;
nothing here changes that convention. No hard dependency — T-089's code is already on `main`'s
feature branch and this ticket only edits the helper and its tests.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # pickle is the root-path child (path = ".")
git checkout main
git checkout -b feat/T-090-harden-linkify-urls
```

Commit locally as you go. Publish only per the project's commit policy (never push/open an MR
without approval); default to keeping the tidied history on merge rather than squashing
(root-path child default). Before pushing, verify the remote base is not behind local
(`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
print nothing).

### Prerequisite gate (hard)

None — no `depends-on:`. T-089's helper is already on `main` (`internal/serve/view.go`); this
ticket only edits that helper and its tests. Re-verified against `main` at refinement time:
`linkifyURLs` (lines 232-267) and its two tests (`TestLinkifyURLsEscapesSurroundingText`,
`TestMergeLineWithoutURLStaysPlain`, `internal/serve/serve_test.go`) still match every sharp
edge the Description describes — nothing has drifted since T-089's review filed this ticket.

### Confirmed design decisions (do not deviate without asking)

1. **Escape-then-trim ordering (item 1) is fixed by matching on the *raw* string first, not by
   dropping `;` from the trim set.** Dropping `;` only patches the one symptom the Description
   found; matching raw and escaping each resulting segment separately removes the whole class
   (an escaped entity can never again straddle a trim boundary, because trimming now runs
   before any escaping happens).
2. **One rewritten algorithm closes items 1-4 together** (they share the same root cause: the
   old code matched `\S+` once, greedily, on already-escaped text). Locate every scheme
   occurrence in the raw string first (`regexp.MustCompile(`https?://`).FindAllStringIndex`),
   then for each one compute its run as the longest non-whitespace stretch starting there,
   *capped at the next scheme occurrence's start* — that cap is what stops two adjacent URLs
   from collapsing into one anchor (item 4), since a run can no longer swallow a second `http(s)`
   token. Trim `urlTrailingPunct` off that raw run (unchanged cutset), then require at least one
   character survive after the `http(s)://` prefix (item 3's guard, replacing item 2's dead
   `trimmed == ""` check outright — do not keep both). A run that fails the host check is left as
   ordinary text, not wrapped.
3. **Escaping happens per segment, on the final (post-trim) boundaries**: the plain-text prefix
   before a run, the trimmed URL itself (used for both `href` and the anchor's visible text),
   and the trailing punctuation/text after it are each passed through
   `template.HTMLEscapeString` independently, then concatenated via a `strings.Builder`. This is
   what makes item 1's entity-tail case render correctly: `https://x.com/a<` trims to itself
   (`<` is not in `urlTrailingPunct`), *then* escapes to `href="https://x.com/a&lt;"` with the
   terminating `;` intact, instead of the old order's truncated `&lt`.
4. **`rel` becomes `"noopener noreferrer"`** (item 6) — the only one-line change with no
   ordering interaction; fold it into the same edit rather than a separate pass.
5. **No behavioural change to what counts as a URL** beyond the host guard: still `https?://`,
   still `\S`-delimited, still the same `urlTrailingPunct` cutset. This ticket hardens the
   existing helper; it does not redesign the matching grammar or add `net/url` parsing.
6. **`CHANGELOG.md` gets no new entry.** T-089's capability ("`pickle serve` now renders a URL
   … as a clickable link") is still under `## [Unreleased]` and has never shipped; T-090 fixes
   edge cases in that same unreleased code before anyone sees them; the entry's own wording
   stays accurate either way. Matches this repo's precedent of not adding a `### Fixed` entry for
   a correctness-only fix with no release-visible behaviour change (T-043's rework, finding R6).

### Tasks

#### Task 1 — Rewrite `linkifyURLs` (`internal/serve/view.go`)

Replace the function body (and its doc comment, which currently narrates the exact bug this
ticket fixes) with the raw-first, per-segment algorithm from decisions 1-4:

```go
// urlSchemeRE marks where a candidate URL run begins; linkifyURLs measures each
// run's extent itself (see below) rather than trying to teach one regexp every
// edge case a lookahead-free engine (Go's RE2) can't express directly.
var urlSchemeRE = regexp.MustCompile(`https?://`)

// urlTrailingPunct is trimmed off the end of a URL run before it becomes an
// href: exactly the characters a human's surrounding prose puts right after a
// pasted URL (a MR ref's closing paren, a trailing comma or full stop).
const urlTrailingPunct = ")].,;:"

// linkifyURLs wraps any bare http(s) URL in s in a clickable anchor and
// HTML-escapes everything else (and the URL itself) with
// template.HTMLEscapeString. It exists because the merge History line's own
// convention (T-089) is free text rendered as a plain, auto-escaped string in
// three serve views (the board's "merged" cell, the ticket page's "merged"
// summary line, and the activity timeline's per-entry text) — unlike a
// ticket's `## Description`/body, which already gets a free clickable link
// from goldmark's GFM Linkify extension (markdown.go). This is the one place
// that gap is closed, so every caller shares it instead of each growing its
// own escape-then-wrap logic.
//
// Matching runs on the raw string, before any escaping: each match is found,
// trimmed and validated first, and only the resulting pieces are escaped —
// never the other way around. Escaping first (T-089's original approach) let
// the trim set below swallow the leading `;` of a genuine HTML entity in a
// URL's tail, corrupting it; matching raw removes that failure mode entirely.
func linkifyURLs(s string) template.HTML {
	starts := urlSchemeRE.FindAllStringIndex(s, -1)
	if starts == nil {
		return template.HTML(template.HTMLEscapeString(s))
	}

	var b strings.Builder
	cursor := 0
	for i, loc := range starts {
		start := loc[0]
		// A run is the longest non-whitespace stretch from start, capped at
		// the next scheme occurrence so two adjacent URLs ("a,https://b")
		// don't collapse into one anchor.
		limit := len(s)
		if i+1 < len(starts) {
			limit = starts[i+1][0]
		}
		end := start
		for end < limit && !unicode.IsSpace(rune(s[end])) {
			end++
		}

		trimmed := strings.TrimRight(s[start:end], urlTrailingPunct)
		host := strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
		if host == "" {
			// Nothing survived trimming but the scheme itself (e.g.
			// "https://)."): not a real link, leave it as plain text.
			b.WriteString(template.HTMLEscapeString(s[cursor:end]))
			cursor = end
			continue
		}

		b.WriteString(template.HTMLEscapeString(s[cursor:start]))
		escaped := template.HTMLEscapeString(trimmed)
		b.WriteString(`<a href="` + escaped + `" rel="noopener noreferrer" target="_blank">` + escaped + `</a>`)
		b.WriteString(template.HTMLEscapeString(s[start+len(trimmed) : end]))
		cursor = end
	}
	b.WriteString(template.HTMLEscapeString(s[cursor:]))

	return template.HTML(b.String()) //nolint:gosec // every byte is HTMLEscapeString output plus literal anchor markup; href and text share one escaped source, so a decoded quote can't reopen attribute parsing, and matching runs on the raw string (see doc comment) before any escaping happens.
}
```

Add `"unicode"` to the import block (`regexp`, `strings`, `template` are already imported).
Delete the old `urlRE` var and its two-paragraph doc comment (lines 220-267 in the current
file) along with the dead `trimmed == ""` branch — decision 2 replaces both the matching and
the guard.

#### Task 2 — Broaden the existing tests, add the missing cases (item 5)

`internal/serve/serve_test.go`:

- `TestLinkifyURLsEscapesSurroundingText` (line 341): update the expected anchor string to
  `rel="noopener noreferrer"` (currently asserts `rel="noopener"`).
- `TestMergeLineWithoutURLStaysPlain` (line 320) and `TestMergeLineLinkification` (line 301):
  read through unchanged — neither depends on `rel`'s exact value or the old matching order — but
  re-run them explicitly during Task 3's acceptance pass since both exercise `linkifyURLs`
  through the three real templates, not just as a unit.
- Add a new `TestLinkifyURLsHardenedEdgeCases` table test, one case per item, each asserting
  the exact rendered string (not just a `strings.Contains` on a substring) so a regression in
  either direction is caught:
  - `https://x.com/a&b` → `&` inside the URL escapes to `&amp;` inside the anchor's href and text.
  - `https://x.com/<script>` → escapes to `&lt;script&gt;` *inside* the anchor, with no stray
    `;` or markup outside it (item 1's exact failure case).
  - `https://).` → renders as plain escaped text, no `<a>` at all (item 3).
  - `https://a.com/1,https://b.com/2` → two separate anchors, `href`s `https://a.com/1` and
    `https://b.com/2`, with the comma as literal text between them (item 4).
  - `https://a.com` alone (no trailing punctuation, no adjacent URL) → still one working anchor,
    guarding against the rewrite over-trimming or under-matching the ordinary case.

#### Task 3 — Manual smoke, once, on the built binary

Per AGENTS.md's self-modify policy: build `pickle` from this branch, then in a throwaway tree
(`D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && ./pk install …`, or reuse
`internal/serve`'s existing `newTree` test helper via `go test -run TestMergeLineLinkification
-v` for a faster non-binary check) confirm a `6-done/` ticket with History line
`merged to main (MR !1, https://x.com/<script>)` renders a correctly-escaped, working anchor on
`/`, `/t/T-NNN`, and `/activity` — the same three surfaces T-089's review checked, now with a
hostile URL instead of a clean one.

### Acceptance test

1. `just build && just test && just lint` — all green, including the new
   `TestLinkifyURLsHardenedEdgeCases` cases and the updated `rel` assertion.
2. Task 3's manual smoke: the hostile-URL merge line renders a clickable, correctly-escaped
   anchor (not a truncated entity, not a stray `;`, not a two-URL href) on all three views.

### Docs update

None required — decision 6. `just docs-check` is unaffected (no `.adoc`/`.md` under
`docs/`/`skill/` changes); still run it as part of Task 1's acceptance pass for parity with
every other ticket's finish step, expecting it green with no diff.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint`/`just docs-check` clean.
2. Write the summary: the raw-first rewrite, which items it closes and how, and the decision to
   skip a `CHANGELOG.md` entry (with the T-043 precedent cited).
3. Suggest a Conventional Commit message, ticket id in brackets, e.g.
   `fix(serve): harden linkifyURLs against escaping, empty hosts, and adjacent URLs (T-090)`.
4. Commit locally on the ticket branch; publish only per the commit policy (approval required).
   `pickle ticket move T-090 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-09 — created (TO DO). source: review of T-089 — six non-blocking findings on
  `linkifyURLs` batched by theme (F1, F3, F4, F5, F6, F7 in T-089's Review table)
- 2026-08-10 — TO DO → READY: plan complete
- 2026-08-10 — READY → IN DEVELOPMENT: picked up
