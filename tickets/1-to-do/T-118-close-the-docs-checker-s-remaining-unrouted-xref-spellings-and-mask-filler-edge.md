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

- **T-115** (`4-in-review/` at filing) — shipped the widened patterns every item above
  refines; this ticket owns no file it does not. Both touch only `docs_xref_test.go`, so
  they must not be in development concurrently. T-115's own rework (blocking findings F2
  and F4) adjusts `docsInterDocLinkRe` and the fixture table; item 1 here edits the
  adjacent `docsInterDocXrefBareRe` in the same `var` block, so expect a trivial textual
  conflict if this is picked up before that rework merges. Deliberately kept separate:
  F2 was a defect T-115's branch introduced, item 1 here is a pre-existing gap it did not
  widen (rules §5, causation not authorship).
- **T-067** — the original checker ticket; the ancestor of this whole line.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

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
