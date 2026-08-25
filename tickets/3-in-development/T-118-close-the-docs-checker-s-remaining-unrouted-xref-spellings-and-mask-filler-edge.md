---
id: T-118
title: close the docs checker's remaining unrouted xref spellings and mask-filler edge
project: pickle
depends-on: []
spawned-by: [T-115]
impact: low
complexity: low
cost: S
---

# T-118 — close the docs checker's remaining unrouted xref spellings and mask-filler edge

## Outcome

After this ships, the manual's cross-reference gate reports the two inter-document
spellings that still reach no detector at all, names the extensionless `<<name#anchor>>`
shorthand for what it is instead of blaming its own scanner, and stops the escape mask
from being able to manufacture a phantom reference.

## Description

Batched follow-up from T-115's review (findings F1, F3, F5b, F6). T-115 widened the
checker's inter-document coverage; these are the edges that survived it. **None is a live
defect** — no construct below appears in the manual today, and each was verified by
planting it against the post-T-115 checker.

**Re-verified at refinement (2026-08-25),** against `main` with T-115 and its two rework
rounds merged: all five items still reproduce exactly as written below, scored by running
every pattern in `docs_xref_test.go`'s `var` block against the planted spellings.

The theme is one sentence: *T-115 routed the spellings its plan named, and the boundary
it drew has four gaps on the far side of it.*

### Still silent — the one that matters

1. **Path-form and underscore-leading extensionless `xref:` reach no detector at all.**
   `docsInterDocXrefBareRe` is `xref:([A-Za-z0-9][A-Za-z0-9_-]*)#…`, which excludes `/`
   and a leading `_`. Verified: `xref:concepts/lifecycle#stage[x]` and `xref:_foo#bar[x]`
   both score `inter=false bare=false link=false angle=false strict=false **site=false**`
   — nothing reports them, which is the *silent* failure mode the checker exists to
   prevent. This is asymmetric with its own sibling `docsInterDocXrefRe`
   (`xref:([^\[\s]+\.adoc)[#\[]`), which *does* allow `/`. Manual pages live under
   `concepts/`, so copying a path minus its extension is the plausible way in. Fix:
   widen the character class toward the sibling's, keeping the empty-target exclusion
   that makes `xref:#local-anchor[text]` legal.

### Caught, but by the wrong detector

2. **The extensionless `<<name#anchor>>` shorthand still gets the misleading message.**
   `docsInterDocAngleRe` requires `.adoc`, so `<<cli-reference#cmd-hooks,hooks>>` scores
   `angle=false … site=true` — it fails the coverage invariant with *"the scanner did not
   report this site … docsXrefTargetRe stopped matching a spelling the docs use"*, sending
   the reader to inspect a pattern that is working correctly. This is exactly the defect
   T-115 item 2 removed for the `.adoc` form, surviving for its extensionless sibling —
   and T-115 decision 6 already established, for `xref:`, that the extensionless spelling
   is the one contributors reach for. Fails loudly, so it is a message defect, not a hole.

### Filler and attribute edges

3. **The escape mask can manufacture a phantom reference.** `docsMaskEscapedXrefs` fills
   masked spans with `_`, which is inside `docsXrefTargetRe`'s target class, so an escape
   nested in a live reference — `<<\<<x>>>>` — masks to `<<______>>` and is then read as a
   reference to an anchor named `______`. Adversarial rather than plausible input, and it
   fails loudly on the right line, so T-115's review recorded it rather than working
   around it (the helper's comment now says so). Fix, if taken: a filler character
   *outside* that class — a space defeats both `docsXrefTargetRe` and
   `docsRefShapedSiteRe`, whose first character class excludes whitespace — rather than a
   special case. Column offsets must still be preserved (T-115 decision 3).
4. **Anchor attribute forms are approximate in both directions.** The widened
   `docsAnchorLineRe` accepts degenerate `[#id,,]` and `[#id, ]`, and still rejects legal
   `[#id.role1.role2]` (multiple roles) and `[#id%breakable]` (an AsciiDoc option). The
   rejection direction is the one with teeth: it is a narrower survival of the very risk
   T-115 item 3 set out to remove — a legitimate anchor the pattern misses turns every
   reference to it into a false "unresolved". Neither form appears in the manual today.
5. **`link:` still false-positives on a scheme-relative URL.** T-115's own rework closed
   the false positive on a *scheme-based* external URL ending in `.adoc`
   (`link:https://host/x/README.adoc[...]`, finding F2) by excluding `:` from
   `docsInterDocLinkRe`'s target class. A **scheme-relative** URL — `link://host/x/README.adoc[...]`,
   no scheme before the `//`, hence no `:` to trip the exclusion — still matches, verified:
   `link=true`. Same defect class as item 1 (a residual gap in a fix this ticket's own line
   already narrowed once), and the same decision-6 rationale ("an external URL should never
   fail the build") is still only half-achieved. Neither form appears in the manual today.
   Fix: exclude `//` immediately after `link:` too, or require the target to contain
   neither `:` nor start with `//`.

### Soft couplings

- **T-115** (`4-in-review/` at filing; **merged to `main` since**) — shipped the widened
  patterns every item above refines; this ticket owns no file it does not. The concurrency
  hazard noted at filing is gone: T-115's rework landed, so the `docsInterDocLinkRe` and
  fixture-table edits item 5 builds on are already in `main` and there is no conflict left
  to expect. Deliberately kept separate: F2 was a defect T-115's branch introduced, item 1
  here is a pre-existing gap it did not widen (rules §5, causation not authorship).
- **T-067** — the original checker ticket; the ancestor of this whole line.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is a root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-118-docs-xref-unrouted-spellings
```

WIP commits are encouraged; tidy them into atomic commits before presenting (root-path child —
keep the tidied history rather than squashing). Do not push or open an MR without explicit user
approval. Under `layout = "in-tree"`, before pushing verify the remote base is not behind:
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing.

### Prerequisite gate (hard)

None. `depends-on:` is empty and `spawned-by: [T-115]` is lineage only. T-115 — the soft
coupling that *was* a scheduling constraint at filing — is in `6-done/` and merged to `main`,
so the whole `var` block this ticket edits is settled.

Start from a clean tree on an up-to-date `main`.

### Confirmed design decisions (do not deviate without asking)

1. **All five items ship together, in one commit-sized change.** They are five edits to the same
   `var` block plus their fixture rows in the same table; splitting them would mean five reviews
   of one file. Item 4 (anchor attribute forms) is the weakest of the five and is taken anyway:
   its rejection direction is a narrower survival of the exact risk T-115 item 3 existed to
   remove.
2. **A widened pattern must not make one site report twice.** `scanBook` appends an occurrence
   per pattern match, so two patterns matching the same span would produce two findings for one
   construct. This is why item 1's widened class admits `/` and a leading `_` but **not** `.`:
   the `.adoc` spelling belongs to `docsInterDocXrefRe`, and admitting `.` here would hand that
   sibling's whole domain a duplicate. Verified during refinement for every widened pattern in
   both directions.
3. **`link:` fixes close false positives without opening silent holes.** An external URL must
   never fail the build (T-115 decision 6), but the fix for one spelling must not stop routing a
   *local* one. So the scheme-relative exclusion rejects a target starting `//` while still
   matching a single leading `/`: `link:/abs/x.adoc[]` stays routed. A blanket "no leading
   slash" would have traded a false positive for a silent miss — the failure mode this file
   exists to prevent.
4. **The escape mask's filler becomes a space, not a special case.** T-115 review F5b already
   recorded the fix: a filler character outside `docsXrefTargetRe`'s target class. A space also
   defeats `docsRefShapedSiteRe`, whose leading class `[^>\s]` excludes whitespace. Byte offsets
   must still be preserved (T-115 decision 3) — the replacement stays equal-length.
5. **Every pattern change gets both a positive and a negative fixture row** (T-115 decision 9).
   A widening is only proven by a row that used to fail, and only bounded by a row that must
   still not match.
6. **The end-to-end fixture proves routing, not just matching.**
   `TestDocsScannerPatternsMatchWhatTheyClaim` pins regexes in isolation; a pattern can match
   while `scanBook` never consults it. Items 1 and 2 — the two that change what reaches a
   detector at all — also get a page in `TestDocsXrefCheckerCatchesTheFieldFindings`, which runs
   the real scan.
7. **No file outside `docs_xref_test.go` changes.** The checker is a test; it ships no
   user-facing surface and no CHANGELOG entry (matching T-067 and T-115, neither of which has
   one). If the fix wants to touch `internal/` or `docs/`, stop and ask.

### Tasks

All edits are in `docs_xref_test.go`. Each task updates the pattern, its doc comment, and its
fixture rows in `TestDocsScannerPatternsMatchWhatTheyClaim` together — the comment is what tells
the next reader why the class is shaped that way.

#### Task 1 — route path-form and underscore-leading extensionless `xref:` (item 1)

Widen `docsInterDocXrefBareRe` (currently `xref:([A-Za-z0-9][A-Za-z0-9_-]*)#([^\[\s]+)\[`) to:

```go
regexp.MustCompile(`xref:([A-Za-z0-9_][A-Za-z0-9_/-]*)#([^\[\s]+)\[`)
```

Verified at refinement: `xref:concepts/lifecycle#stage[x]` and `xref:_foo#bar[x]` now match,
`xref:#local-anchor[text]` (legal intra-document, empty target) still does not, and
`xref:cli-reference.adoc#cmd-hooks[hooks]` / `xref:../proposals/thing.adoc#x[y]` still match only
the `.adoc` sibling — no site matched by both (decision 2).

Extend the doc comment: say *why* `.` stays out of the class (it is `docsInterDocXrefRe`'s
domain, and admitting it here would report one site twice), so the next person widening this
class does not "finish the job" and create duplicates.

Fixture rows for `"inter-document xref: forms (extensionless)"`: add
`xref:concepts/lifecycle#stage[x]` and `xref:_foo#bar[x]` to `match`; keep both existing
`noMatch` rows.

#### Task 2 — give the extensionless `<<name#anchor>>` shorthand its own detector (item 2)

Drop the `.adoc` requirement from `docsInterDocAngleRe`:

```go
regexp.MustCompile(`<<([^>,\s#]+)#([^>,\s]+)(?:,[^>]*)?>>`)
```

Verified at refinement: `<<cli-reference#cmd-hooks,hooks>>` and `<<concepts/lifecycle#stage>>`
now match; `<<cmd-hooks>>`, `<<lifecycle,the lifecycle>>` and `<<cli-reference.adoc>>` (no
anchor) still do not; and no site matches both this and `docsXrefTargetRe`, whose target class
excludes `#` (decision 2).

The failure message needs no change — `TestDocsNoInterDocumentXrefForm`'s
`strings.TrimSuffix(x.target, ".adoc")` already degrades to the target itself for an
extensionless one. Confirm that by reading the rendered message in the acceptance test rather
than by assuming it.

Update the comment above the pattern: it currently explains the misleading-message defect for
the `.adoc` form only. State that both spellings route here now, and that the extensionless one
is the spelling contributors actually reach for (T-115 decision 6).

Also update the file-header comment (the bullet beginning *"Every inter-document spelling the
manual can legally contain now routes to…"*): its enumeration names `<<file.adoc#anchor>>` and
the extensionless `xref:` form, and must now also name the extensionless `<<name#anchor>>` and
the path form. That paragraph is the file's own claim about its coverage boundary — leaving it
stale is the defect this ticket is closing, one level up.

Fixture rows for `"inter-document <<file.adoc#anchor>> shorthand"`: rename the case to drop the
`.adoc` from its name, add `<<cli-reference#cmd-hooks,hooks>>` and `<<concepts/lifecycle#stage>>`
to `match`, and add `<<lifecycle,the lifecycle>>` to `noMatch` alongside the two rows there.

#### Task 3 — make the escape mask's filler inert (item 3)

In `docsMaskEscapedXrefs`, replace `strings.Repeat("_", …)` with `strings.Repeat(" ", …)`.

Rewrite the paragraph of its doc comment that currently records the phantom-anchor defect as
accepted (*"The filler is not inert… recorded here rather than worked around"*): it is fixed
now, and a comment describing a defect that no longer exists is worse than none. The replacement
states the property the filler must have — outside `docsXrefTargetRe`'s target class *and*
outside `docsRefShapedSiteRe`'s leading class — so a future change back to a word character is
visibly a regression rather than a preference.

Add a regression test, `TestDocsMaskEscapedXrefsFillerIsInert`, next to the mask helper's
existing coverage in `TestDocsProseLineSelection`:

- `docsMaskEscapedXrefs(`<<\<<x>>>>`)` must yield a string of the same byte length that neither
  `docsXrefTargetRe` nor `docsRefShapedSiteRe` matches (with `_` filler this produced a
  reference to a phantom anchor `______`);
- a real `<<real>>` later on the same line must still be found, so the fix is not a blanket
  widening of the mask.

#### Task 4 — accept the legal anchor attribute forms, reject the degenerate ones (item 4)

```go
docsAnchorLineRe = regexp.MustCompile(`^\[#([A-Za-z0-9_-]+)(?:[.%][A-Za-z0-9_-]+)*(?:,\s*[^,\s\]][^\]]*)?\]\s*$`)
```

The repeated `[.%]` group accepts multiple roles and AsciiDoc options in any mix
(`[#id.role1.role2]`, `[#id%breakable]`, `[#id.role%breakable]`); the reftext group now requires
at least one non-space, non-comma character, which rejects `[#id,]`, `[#id, ]` and `[#id,,]`.
The id capture stays group 1 — the added group is non-capturing, so `scanBook`'s `m[1]` is
unaffected. Verified at refinement against every row of the existing fixture table, including
the `[[project]]` TOML-prose negatives (F15).

Update the comment above it: today it names "a role" and "reftext" singular; it should name the
full attribute shorthand it now accepts and say why the degenerate forms are refused (an anchor
with an empty reftext is a typo, and accepting it teaches the pattern to accept a shape
AsciiDoc does not render).

Fixture rows for `"anchor definitions"`: add `[#id.role1.role2]`, `[#id%breakable]` and
`[#id.role%breakable]` to `match`; add `[#id, ]` and `[#id,,]` to `noMatch` beside the existing
`[#id,]`.

#### Task 5 — stop `link:` false-positiving on a scheme-relative URL (item 5)

```go
docsInterDocLinkRe = regexp.MustCompile(`link:(/?[^\[\s:/][^\[\s:]*\.adoc)[#\[]`)
```

Verified at refinement: `link://host/x/README.adoc[y]` no longer matches, while
`link:cli-reference.adoc[x]`, `link:cli-reference.adoc#id[x]` and `link:/abs/x.adoc[y]` all
still do (decision 3), and the two existing external-URL negatives plus the `link:`-in-prose
negative (decision 7) still do not.

Extend the comment's existing note about F2 — the `:` exclusion that closed the *scheme-based*
false positive — to say that a scheme-relative URL carries no `:` to trip it, which is why the
class also excludes a second leading `/`, and that a single leading `/` is deliberately kept so
a local absolute path is still routed rather than silently dropped.

Fixture rows for `"inter-document link: forms"`: add `link:/abs/x.adoc[y]` to `match` and
`link://host/x/README.adoc[the source]` to `noMatch`.

#### Task 6 — prove items 1 and 2 end-to-end through `scanBook` (decision 6)

Extend `TestDocsXrefCheckerCatchesTheFieldFindings` with two pages, wired into the fixture book
via `include::` in `book.adoc` exactly as pages A–D are:

- **page E** — `See xref:sub/page-a#real-anchor[Page A] for details.` (item 1: the path form,
  extensionless);
- **page F** — `See <<page-a#real-anchor,Page A>> for details.` (item 2: the extensionless
  shorthand).

Assert for each, in the style of the existing page-D block: the occurrence appears in
`interDoc`, no occurrence from that page landed in `xrefs`, and the closing
`assertEveryXrefSiteScanned(t, files, xrefs, interDoc)` still passes — which is precisely the
coverage-invariant misreport item 2 removes. Note in the page-F comment that before this change
that site scored `angle=false site=true`, i.e. it was reported as a scanner defect rather than
as the inter-document reference it is.

### Acceptance test

Run from a clean tree on the feature branch.

1. **The suite is green, including the real manual:**

   ```
   just test
   ```

   `TestDocsXrefsResolve` and `TestDocsNoInterDocumentXrefForm` run against the real book, so a
   widened pattern that now matches something in `docs/user-manual/` fails here. Expected: pass —
   none of the five constructs appears in the manual today.

2. **Each widened pattern actually widened** (the rows added in Tasks 1–5 are the proof):

   ```
   go test -run TestDocsScannerPatternsMatchWhatTheyClaim -v ./...
   ```

   Expected: every sub-test passes, including the newly named `anchor definitions`,
   `inter-document xref: forms (extensionless)`, `inter-document link: forms` and the renamed
   angle-shorthand case.

3. **The mask filler is inert:**

   ```
   go test -run TestDocsMaskEscapedXrefsFillerIsInert -v ./...
   ```

   Expected: pass. Sanity-check the fix is load-bearing by temporarily restoring `"_"` as the
   filler — this test must fail, and must be the *only* new failure. Restore the space and
   re-run.

4. **Items 1 and 2 route end-to-end:**

   ```
   go test -run TestDocsXrefCheckerCatchesTheFieldFindings -v ./...
   ```

   Expected: pass with pages E and F present. Then temporarily revert Task 1's pattern to its
   pre-change form and re-run: page E must fail with "expected … to be flagged", proving the
   fixture tests the scanner rather than restating the regex. Restore.

5. **The four planted spellings behave as the ticket says.** With the branch's patterns, score
   each construct below and record the result in the summary. Expected, in the same
   `inter/bare/link/angle/strict/site` vocabulary the Description uses:

   | planted construct | expected after this ticket |
   |---|---|
   | `xref:concepts/lifecycle#stage[x]` | `bare=true` — reported as inter-document |
   | `xref:_foo#bar[x]` | `bare=true` |
   | `<<cli-reference#cmd-hooks,hooks>>` | `angle=true` — the correct message, not the coverage misreport |
   | `link://host/x/README.adoc[y]` | `link=false` — external URL, correctly silent |
   | `<<\<<x>>>>` | masked to spaces; `strict=false site=false` |

6. **The child's configured commands are clean:**

   ```
   just build && just test && just lint && just docs-check
   ```

7. **Only the checker moved:** `git diff --name-only main...HEAD` prints exactly
   `docs_xref_test.go` (decision 7).

### Docs update (mandatory when user-facing)

No user-facing surface. The cross-reference checker is a Go test run by `just test`; it ships no
command, flag or output a user of the `pickle` binary sees, and neither T-067 (which introduced
it) nor T-115 (which widened it) carries a CHANGELOG entry or a manual section. The documentation
that *does* change is in-file — the pattern comments and the file-header coverage paragraph
updated by Tasks 1–5 — which is where this checker has always documented itself.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs: none (see above); confirm the in-file comments were updated alongside every pattern.
3. Write a **summary**: the five patterns before/after, the scored table from acceptance step 5,
   the two temporary-revert checks (steps 3 and 4) and what failed in each, and anything the
   widened patterns turned up in the real manual.
4. Suggest a **Conventional Commit message**, e.g.:

   ```
   test(docs): route the remaining xref spellings and make the escape filler inert (T-118)

   <body — what and why>
   ```

5. **Tidy up before presenting** — `pickle` is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed commits first.
6. Commit locally on the ticket branch. Do **not** push or open a merge request without user
   approval. Present the commit message; after approval, keep the tidied history (root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the merge request — merging is always the
   human's. Hand back to the user.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-24 — created (TO DO). source: review: batched from T-115's review — findings F1
  (extensionless `<<name#anchor>>` gets the misleading coverage message), F3 (path-form and
  underscore-leading extensionless `xref:` reach no detector at all — the only silent one),
  F5b (the escape mask's `_` filler can manufacture a phantom reference) and F6 (anchor
  attribute forms approximate in both directions). Batched by theme per rules §5 rather than
  filed one per finding; none is a live defect, each verified by planting it against the
  post-T-115 checker. T-115's own blocking findings (F2, F4) are excluded — they are that
  ticket's rework scope, not this one's
- 2026-08-24 — item 5 folded in: T-115's scoped re-review (round 2) found the F2 rework
  closed the scheme-based `link:` false positive but not a scheme-relative one
  (`link://host/x.adoc[...]`, finding F9) — same defect class as item 1, folded here per
  rules §5 rather than opened as a new ticket
- 2026-08-25 — TO DO → READY: plan complete
- 2026-08-25 — READY → IN DEVELOPMENT: picked up
