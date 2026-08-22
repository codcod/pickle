---
id: T-115
title: harden the docs xref checker: pattern coverage, helper fixtures and AsciiDoc escape handling
project: pickle
depends-on: []
spawned-by: [T-067]
impact: low-medium
complexity: low
cost: S
---

# T-115 — harden the docs xref checker: pattern coverage, helper fixtures and AsciiDoc escape handling

## Outcome

After this ships, the manual's cross-reference gate catches two more inter-document spellings,
stops false-positiving on AsciiDoc's literal-reference escapes, and fails loudly on an
unterminated listing block instead of silently skipping the rest of the file. Its own helpers
gain the fixtures that keep all of that true.

## Description

Batched follow-up from T-067's three review rounds. That ticket shipped the gate; these are the
edges it deliberately left, plus what its own reviews turned up while attacking it. **None is a
live defect** — every one requires modifying the checker before it bites, or names a construct
the manual does not currently contain. They are recorded because the checker's whole purpose is
to stay correct while unattended.

### Pattern coverage

1. **The `<<file.adoc#anchor>>` shorthand escapes both detectors.** `docsXrefTargetRe` excludes
   `.` and `#`, and the inter-document detector requires the literal `xref:`. So the T-057
   finding-N3 defect class can re-enter in AsciiDoc's other spelling and pass everything.
   Verified: `<<cli-reference.adoc#cmd-hooks,hooks>>` matches neither pattern.
2. **Two more inter-document spellings match nothing**: extensionless natural references
   (`xref:cli-reference#cmd-hooks[...]`) and `link:cli-reference.adoc#id[...]`.
3. **Anchor attribute forms are unmatched** — `[#id,reftext]` and `[#id.role]` are valid
   AsciiDoc, and a legitimate anchor written either way would turn every reference to it into a
   false "unresolved". Neither appears in the manual today.

### Escapes and false positives

4. **AsciiDoc's literal-reference escapes are false-positived.** `\<<x>>` and `+<<x>>+` are the
   two documented ways to show a cross-reference without making one; both are read as real
   references and reported unresolved. The unresolved message also offers only "add `[#x]` or
   fix the target", not the literal-block escape its sibling message does. This is precisely
   what documenting the checker *in the manual* would require.
5. **No code-span exemption on the inter-document detector** — it matches inside inline code and
   listing blocks, so documenting the `xref:` rule in prose would trip it.

### Helper fixtures (the "one level down" class)

T-067's reviews twice found the guard guarding the wrong layer. Two instances remain, both
reachable only by modifying the checker, and both cheap to pin:

6. **Scanner-level anchor inflation is unguarded.** The fixture table pins `docsAnchorLineRe`,
   but adding a legacy `[[id]]` match *inside `scanBook`* swallows `configuration.adoc`'s
   backticked `\[[project]]` TOML prose as an anchor named `project`; a planted dead
   `<<project>>` then passes every test. Fix: a negative fixture that calls `scanBook` over that
   prose and asserts `anchors["project"]` is false.
7. **`docsProseLines` over-exemption is invisible by construction.** Scanner and coverage
   invariant share it, so anything it wrongly drops is dropped from both at once. Adding "skip
   indented lines" plus a dead reference on a real indented line passes everything.
   `TestDocsProseLineSelection` pins `docsLiteralBlockDelim` but not "which lines count as
   prose". Fix: a fixture asserting a delimiter-free document yields every line unchanged.
8. **The regression fixture re-implements its assertion** rather than calling shared code, so
   inverting the real comparison leaves it green. Fix: extract the unresolved check and have
   both callers use it.

### The one with a CI-visible consequence

9. **An unterminated `----` silently blanks the rest of the file.** `docsProseLines` never
   reports an unclosed block, so a dead reference after one is invisible to `go test` — which
   is exactly what CI runs. `just docs-check` does catch it, because `snowball check` fails on
   an unterminated listing block, but CI does not run `snowball`. Verified both halves. Fix:
   fail when a block is still open at EOF. Worth doing first — it is the only item here that
   can hide a real defect from CI without anyone editing the checker.

### Soft couplings

- **T-067** — shipped the checker; every item above is an edge it scoped out or a review found.
  No file overlap with anything else: this ticket owns `docs_xref_test.go`.
- **F9 from T-067's round 1, still open and dispositioned `noted` there**, is deliberately *not*
  folded in: `pickle scaffold docs` writes "broken includes/xrefs fail the check" into other
  projects' justfiles, which is a scaffold-payload accuracy question, not a checker one.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: review: batched from T-067's three review rounds — round 1 findings F3–F6 (pattern coverage, code-span exemptions, fixture structure), round 3 findings N5–N6, and round 4 findings R1–R4 (helper fixtures, AsciiDoc escapes, and the unterminated-block hole that is invisible to CI). Batched by theme per rules §5 rather than filed one per finding; none is a live defect, all require modifying the checker or naming a construct the manual does not yet contain
