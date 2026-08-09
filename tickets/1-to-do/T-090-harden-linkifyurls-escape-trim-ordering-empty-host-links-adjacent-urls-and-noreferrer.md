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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-09 — created (TO DO). source: review of T-089 — six non-blocking findings on
  `linkifyURLs` batched by theme (F1, F3, F4, F5, F6, F7 in T-089's Review table)
