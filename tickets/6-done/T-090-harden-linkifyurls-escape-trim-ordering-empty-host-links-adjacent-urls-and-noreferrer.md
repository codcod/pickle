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
   *(Amended at review, finding F2 — this was written before the algorithm was chosen and the
   rewrite made it literally false. Decision 2's run-cap necessarily changes how concatenated
   and nested schemes split, and moving from Go regexp's `\S` (ASCII `[^\t\n\f\r ]`) to a
   `unicode.IsSpace` scan necessarily changes which characters delimit a run. Read decision 5
   as: the **scheme set** and the **`urlTrailingPunct` cutset** are unchanged, and no
   `net/url` parsing is introduced.)*
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

Reviewed 2026-08-10 on `feat/T-090-harden-linkify-urls` (commit `d153f2a`), ticket read from
`main` per the protocol; audited in a detached worktree so the branch could be exercised without
disturbing the base checkout. The implementer authored this ticket, so the implementation,
quality, consistency and docs audits were also run by an independent sub-agent briefed to be
adversarial; its findings were then re-verified first-hand (a standalone probe harness running
the new helper and T-089's original side by side over ~30 hostile inputs, plus a 300 000-case
randomised fuzz for panics and UTF-8 corruption).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [ ] Docs-readability pass (step 4b) — **not applicable: the ticket changed no `.adoc`/`.md`
      files** (`git diff --name-only main...feat/T-090-harden-linkify-urls` is two `.go` files).
      Nothing for a prose reviewer to read; not a skip, an empty set.
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented; bookkeeping committed (step 9)

*(The checklist above covers the first, full review round. The scoped re-review after rework is
recorded under "Scoped re-review" below — per rules §5 it re-verifies only the blocking finding,
so steps 3, 4, 4a and 8 were deliberately not repeated.)*

### Step 2 — implementation audit

| item | verdict | evidence |
|---|---|---|
| Task 1 — rewrite `linkifyURLs`, delete `urlRE` + dead `trimmed == ""` branch, add `unicode` import | met | `d153f2a` `internal/serve/view.go:222-289`; `grep -rn 'urlRE\b' --include=*.go` returns nothing (no stale refs) |
| Task 2 — `rel` assertion updated in 2 tests, `TestLinkifyURLsHardenedEdgeCases` added with the 4 demanded cases + the ordinary case | met | `serve_test.go:311,349,354-395`; each case asserts an exact string, not `strings.Contains` |
| Task 3 — manual smoke on the three views with a hostile URL | met | re-verified during review: `/t/T-001` and `/activity` render `<a href="https://x.com/&lt;script&gt;" rel="noopener noreferrer" …>`; `/` is covered by `TestMergeLineLinkification` |
| Decisions 1, 2, 3, 4 (raw-first matching; one algorithm for items 1-4; per-segment escaping; `rel="noopener noreferrer"`) | met (4/4) | verified by probe: items 1, 3, 4 and 6 all behave as specified — see F1's table for the exact outputs |
| Decision 5 (no behavioural change beyond the host guard) | **not met** | F2/F3 — `\v` and concatenated/nested schemes now split differently than on `main` |
| Decision 6 (no `CHANGELOG.md` entry) | met | `## [Unreleased]` still carries T-089's linkify entry; latest tag `v0.4.0` predates it, so the code being hardened has never shipped |
| Acceptance 1 — `just build`/`test`/`lint`/`docs-check` green | met | re-run verbatim in the review worktree: all `ok`, `go vet` silent, `gofmt` clean, `snowball check` exit 0 |
| Acceptance 2 — hostile-URL smoke on all three views | met | as Task 3 above |

### Findings

| id | severity | disposition | description | evidence | suggestion / resolution |
|---|---|---|---|---|---|
| F1 | **blocking** | — | `unicode.IsSpace(rune(s[end]))` widens a single **byte** to a rune. Bytes `0x85` and `0xA0` satisfy `unicode.IsSpace` but occur in valid UTF-8 only as **continuation bytes**, so the run scan halts mid-rune: the href is truncated and the output is **invalid UTF-8**. Affects any URL containing a rune whose encoding carries those bytes — `à` (C3 A0), `Å` (C3 85), `Š` (C5 A0), `Π` (CE A0), `Р` (D0 A0), `ą` (C4 85), NBSP, NEL … A **regression**: T-089's `\S+` handled all of them correctly. | `view.go:267`. Probe, new vs old: `https://x.com/à` → new `<a href="https://x.com/\xc3">…</a>\xa0` (`utf8.ValidString` **false**), old `<a href="https://x.com/à">` (true). Same for `https://à.com/x`, `https://example.com/ą`, `https://x.com/a\u00a0b`. 300 000-case fuzz: 0 panics, 0 corruptions on the ASCII alphabet — the defect needs the specific continuation bytes. | Decode properly: `for end < limit { r, sz := utf8.DecodeRuneInString(s[end:]); if unicode.IsSpace(r) { break }; end += sz }`, or `strings.IndexFunc(s[start:limit], unicode.IsSpace)`. **The fix must also state which whitespace semantics it adopts** (rune-aware `unicode.IsSpace`, or ASCII-only to match `main` exactly) — that choice is F3's substance — and add a regression case asserting `utf8.ValidString` on the output. |
| F2 | non-blocking | fixed inline | Confirmed design decision 5 ("no behavioural change to what counts as a URL beyond the host guard … still `\S`-delimited") is literally false of the code it authorised: decision 2's run-cap and the `unicode.IsSpace` scan both change run boundaries. The plan's own prose, made false by the plan's own algorithm. | the decision as written vs. F3's measured deltas | Done above: decision 5 now carries an amendment naming what actually stayed fixed (scheme set, `urlTrailingPunct` cutset, no `net/url`). No behaviour change. |
| F3 | non-blocking | noted | Two measured behaviour deltas vs `main`, both consequences of decision 2 rather than defects: (a) `\v` (U+000B) now terminates a run — Go regexp's `\s` excludes it, `unicode.IsSpace` includes it; (b) concatenated/nested schemes split differently — `https://a.comhttps://b.com` was one anchor, is now two; `https://https://a.com` was one anchor, now leaves a bare `https://` as text. | probe, new vs old, all four inputs | Closed with evidence. (a) is arguably a correctness improvement and (b) is decision 2 working as designed; both are recorded here so a later reader is not surprised. The `\v`/whitespace-semantics question is folded into F1's fix, which has to pick a scan predicate anyway. |
| F4 | non-blocking | noted | Test-table coverage gap beyond F1's regression case: no `https://https://…`, no host-less-URL-followed-by-a-real-URL, no `utf8.ValidString` property assertion, no whitespace-delimiter case. The table is ASCII-only, which is precisely why F1 shipped. | `serve_test.go:354-395` | Closed with evidence; F1's fix carries the non-ASCII regression case, which is the one that matters. The rest do not pass the promotion test — they guard behaviour no caller exercises. |
| F5 | non-blocking | noted | Backtick is not escaped: `` https://x.com/`onmouseover=alert(1) `` renders a literal `` ` `` inside the `href`. Harmless under HTML5 (the attribute is `"`-delimited and all three call sites are element-content contexts), though Go's own attribute escaper escapes `` ` `` → `&#96;` for legacy IE. **Pre-existing** — reproduces identically on `main`, so not this branch's doing. | probe, new and old produce the same output | Closed with evidence. Per rules §5 the inline bar is causation, not size: this branch neither authored nor falsified it. |
| F6 | non-blocking | fixed inline | `TestLinkifyURLsHardenedEdgeCases`' doc comment cited four finding ids for five cases, never described the ampersand case, and omitted item 6 though every `want` pins `rel="noopener noreferrer"`. Prose this branch authored. | `serve_test.go:354-357` as authored in `d153f2a` | Done in `498c14f`: comment now maps each case to its finding id and names the `rel` assertion. Comment-only; suite still green. |

**Disposition summary.** 6 findings, **1 blocking** (F1), 5 non-blocking: **2 fixed inline**
(F2 ticket prose, F6 test comment), **3 noted** (F3 behaviour deltas, F4 residual test-coverage
gap, F5 pre-existing backtick escaping). **0 new tickets** — nothing here passes the promotion
test; F1 is fixed on this branch, not deferred. → `5-rework/`, scoped to F1 alone.

### Rework (F1 only)

Fixed 2026-08-10 on `feat/T-090-harden-linkify-urls` (commit `2707a18`, amended from `a51889d`
per F7 below). Replaced the byte-widening scan
(`for end < limit && !unicode.IsSpace(rune(s[end])) { end++ }`) with
`strings.IndexFunc(s[start:limit], unicode.IsSpace)`, which decodes each rune before testing it.
This is the "rune-aware" branch of F1's suggested resolution, not the "ASCII-only to match
`main`" branch.

The **predicate** stays `unicode.IsSpace`, but decoding it correctly necessarily **widens the
effective whitespace set**: U+2028, U+2029, U+3000, U+1680 and U+202F now terminate a run where
the byte-widened scan kept them inside the href (measured — see F7). That is coherent with F3,
which already accepted the analogous `\v` delta, and is strictly better: those characters must be
percent-encoded in a URL anyway.

Added two regression cases to `TestLinkifyURLsHardenedEdgeCases` (`serve_test.go`): a multi-byte
rune (`à`, `C3 A0`) surviving intact inside a URL, and a literal NBSP (`C2 A0`) still terminating
the run — asserting the exact rendered string, plus `utf8.ValidString(got)` across **every**
subcase in the table, not just the two new ones.

Re-verified independently (not just via the new unit tests): the seven inputs that produced
invalid UTF-8 in the review's probe (`https://x.com/à`, `https://x.com/à/commit`,
`https://example.com/ą`, `https://x.com/a\u00a0b`, `https://x.com/日本`, `https://x.com/a\u0085b`,
plus the ordinary-case control) now all produce `utf8.ValidString == true` output with the full
href intact. A fresh 500 000-case fuzz using an alphabet that includes `à ą Å Š 日 é` (the review's
original fuzz used an ASCII-heavy alphabet, which is exactly why it reported 0 corruptions despite
F1 existing) found 0 panics and 0 UTF-8 corruptions.

`just build && just test && just lint && just docs-check` all green in the rework worktree.

### Scoped re-review (2026-08-10, F1 only)

Re-reviewed on `feat/T-090-harden-linkify-urls` @ `2707a18`, ticket read from `main`, audited in a
detached worktree. Per the protocol's rework clause only F1 was re-verified — the feature was not
re-audited from scratch. An independent sub-agent re-ran the verification adversarially, briefed
specifically to hunt for a defect *introduced by* the fix; its findings were then reproduced
first-hand.

- [x] **F1 resolved.** Every input that previously emitted invalid UTF-8 now yields a complete
      href and valid UTF-8: `https://x.com/à`, `https://à.com/x`, `https://example.com/ą`,
      `…/Å`, `…/Š`, `…/Π`, `…/Р`, NBSP/NEL both inside and after the URL, CJK. Independent
      fuzz, 400 000 cases over an alphabet mixing ASCII URL characters with `à ą Å Š 日 é` and
      six whitespace runes: **new impl 0 invalid-UTF-8 outputs, 0 panics** (63 852 anchors
      produced) vs **41 091 invalid outputs for the old impl over the identical corpus**. A
      further 200 000 deliberately invalid-UTF-8 inputs: no panic (invalid bytes decode to
      `RuneError`, which is not a space, and the scan advances one byte).
- [x] **No new defect.** `IndexFunc == -1 → end = limit` is provably the old loop's terminal
      value; `limit > start` strictly (`FindAllStringIndex` yields non-overlapping ascending
      matches, so `starts[i+1][0] >= start+7`); `start`/`limit` are rune boundaries because
      `https?://` is pure ASCII, so `s[start:limit]` never cuts mid-rune and
      `start <= end <= limit` holds. 0 invariant violations and 0 panics across the fuzz corpus.
- [x] **The regression tests genuinely guard the defect** — demonstrated, not assumed: both new
      cases *fail* against `d153f2a` (`https://x.com/à/commit` → old yields `…/\xc3</a>\xa0/commit`,
      mismatching **and** invalid UTF-8; `https://x.com/a\u00a0tail` likewise).
- [x] **Acceptance test re-run verbatim.** `just build`/`just test`/`just lint`/`just docs-check`
      all green; `go vet` silent, `gofmt -l` empty. End-to-end smoke re-done against a real
      `pickle serve` binary with the merge line `merged to main (MR !1, https://x.com/à/<script>)`:
      the anchor renders with `à` intact, `<script>` escaped, and `rel="noopener noreferrer"` on
      **all three** surfaces — board `/`, ticket page `/t/T-001`, and `/activity` — with every
      response valid UTF-8. (The board cell was silently empty in the previous round's smoke
      because the throwaway fixture's `project:` did not name that tree's registered child; the
      fixture is corrected here, so the third surface is now genuinely evidenced rather than
      inferred from `TestMergeLineLinkification`.)

#### Re-review finding

| id | severity | disposition | description | evidence | suggestion / resolution |
|---|---|---|---|---|---|
| F7 | non-blocking | fixed inline | The rework's own prose overclaimed: commit `a51889d`'s message and this section both said the fix closed the gap "without changing which characters count as whitespace". False — decoding runes correctly *widens* the effective set, because the byte-widened scan had never recognised any multi-byte space. Prose this branch authored, no behaviour change. | measured, new vs `d153f2a`: `https://x.com/a\u2028b`, `…\u3000b`, `…\u1680b`, `…\u202fb` all **DIFFER** — new terminates the run, old kept the character inside the href. Plain ASCII space: same. The **in-code** comment (`view.go:266-271`) never made the claim, so only the prose was wrong. | Done: commit message amended (`a51889d` → `2707a18`, unpushed so no history rewrite of published work) and this section's opening paragraph now states the widening explicitly with its rationale. |

**Re-review disposition summary.** F1 **resolved** — no blocking findings remain. 1 new
non-blocking finding (F7), **fixed inline**. 0 new tickets. → `6-done/`.

Cumulative across both rounds: 7 findings, 1 blocking (F1, fixed in rework), 6 non-blocking
(3 fixed inline — F2, F6, F7; 3 noted — F3, F4, F5), 0 new tickets.

### Steps 3, 4, 4a — quality, consistency, docs

- **Security (step 3).** No injection found. `href` and anchor text share one
  `template.HTMLEscapeString` source; `"`→`&#34;`, `'`→`&#39;`, `<>&` all escaped;
  `javascript:`, `data:` and uppercase `HTTPS://` never match `urlSchemeRE`; all three call
  sites are element-content contexts. The `//nolint:gosec` justification holds. No panic on any
  input tried, including 100 KB strings, overlapping schemes and raw invalid UTF-8; index
  arithmetic is sound (`cursor ≤ start` always, since `end ≤ limit = starts[i+1][0]`, and
  `TrimRight` is rune-aware so `s[start+len(trimmed):end]` cannot cut mid-rune).
- **Consistency (step 4).** No stale references to the deleted `urlRE`; `linkify` remains
  registered once in `funcs.go:23` and used in all three templates (`board.html:47`,
  `ticket.html:19`, `activity.html:27`). The byte-widening idiom behind F1 appears **nowhere
  else** in the tree (`grep -rn 'rune(.*\[[a-zA-Z_]*\])' --include=*.go` returns only
  `view.go:267`), so the fix is genuinely one line in one place.
- **Docs (step 4a).** Nothing user-facing changed: the helper is internal, its behaviour change
  is a bug fix to unreleased code, and no `.adoc`/`.md` under `docs/`/`skill/` mentions
  `linkify`, `noopener` or the anchor's attributes. `just docs-check` green. No coverage owed —
  decision 6 verified correct independently.

### Step 8 — impact sweep

No ticket in `1-to-do/` or `2-ready/` references T-090, `linkify` or `internal/serve/view.go`;
the READY queue is empty. Nothing downstream to patch.

## History

- 2026-08-09 — created (TO DO). source: review of T-089 — six non-blocking findings on
  `linkifyURLs` batched by theme (F1, F3, F4, F5, F6, F7 in T-089's Review table)
- 2026-08-10 — TO DO → READY: plan complete
- 2026-08-10 — READY → IN DEVELOPMENT: picked up
- 2026-08-10 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-10 — IN REVIEW → REWORK: review: 1 blocking (F1 — byte-widened IsSpace splits
  multi-byte runes, invalid UTF-8); 5 non-blocking (2 fixed inline, 3 noted)
- 2026-08-10 — REWORK → IN REVIEW: F1 fixed: rune-decoding scan closes the byte-widened IsSpace gap
- 2026-08-10 — IN REVIEW → DONE: scoped re-review: F1 resolved, no blocking findings; F7 fixed inline
- 2026-08-10 — merged to main (PR #27, b232d0a, https://github.com/codcod/pickle/commit/b232d0a); merge commit, 3 commits kept
