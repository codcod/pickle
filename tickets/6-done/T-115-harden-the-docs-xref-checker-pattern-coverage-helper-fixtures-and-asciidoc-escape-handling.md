---
id: T-115
title: harden the docs xref checker: pattern coverage, helper fixtures and AsciiDoc escape handling
project: pickle
depends-on: []
spawned-by: [T-067]
impact: low
complexity: low
cost: M
---

# T-115 — harden the docs xref checker: pattern coverage, helper fixtures and AsciiDoc escape handling

## Outcome

After this ships, the manual's cross-reference gate catches the two inter-document spellings
that currently pass unseen, names the `<<file.adoc#id>>` spelling for what it is instead of
blaming its own scanner, stops false-positiving on AsciiDoc's literal-reference escapes, and
fails loudly on an unterminated listing block instead of silently skipping the rest of the
file. Its own helpers gain the fixtures that keep all of that true.

## Description

Batched follow-up from T-067's three review rounds. That ticket shipped the gate; these are the
edges it deliberately left, plus what its own reviews turned up while attacking it. **None is a
live defect** — every one requires modifying the checker before it bites, or names a construct
the manual does not currently contain. They are recorded because the checker's whole purpose is
to stay correct while unattended.

### Pattern coverage

1. **Two inter-document spellings match nothing at all.** Extensionless natural references
   (`xref:cli-reference#cmd-hooks[...]`) and `link:cli-reference.adoc#id[...]` are invisible to
   every pattern in the file — re-verified at refinement, both scoring `strict=false
   inter=false site=false`. This is the group's **only genuine silent hole**, and the one that
   re-admits the T-057 finding-N3 defect class unseen.
2. **The `<<file.adoc#anchor>>` shorthand is caught, but misdiagnosed.** *(Corrected at
   refinement — as filed, this item claimed the shorthand "escapes both detectors … and pass
   everything", and was listed first for that reason.)* It does escape both *detectors*, but
   `docsRefShapedSiteRe` matches it, so `assertEveryXrefSiteScanned` fails the build.
   Re-verified: `<<cli-reference.adoc#cmd-hooks,hooks>>` scores `strict=false inter=false
   **site=true**`. The defect is therefore the **message**, not the coverage: the failure reads
   *"the scanner did not report this site … docsXrefTargetRe stopped matching a spelling the
   docs use"* and sends the reader to inspect a pattern that is working correctly, when the
   real answer is "this is an inter-document reference in `<<>>` clothing — write
   `<<anchor>>`". So the fix is to route the spelling to the inter-document detector for the
   right message, not to close a hole.
3. **Anchor attribute forms are unmatched** — `[#id,reftext]` and `[#id.role]` are valid
   AsciiDoc, and a legitimate anchor written either way would turn every reference to it into a
   false "unresolved". Neither appears in the manual today.

### Escapes and false positives

4. **AsciiDoc's literal-reference escapes are false-positived.** `\<<x>>` and `+<<x>>+` are the
   two documented ways to show a cross-reference without making one; both are read as real
   references and reported unresolved (re-verified: `strict=true`). The unresolved message also
   offers only "add `[#x]` or fix the target", not the literal-block escape its sibling message
   does. This is precisely what documenting the checker *in the manual* would require.
5. ~~**No code-span exemption on the inter-document detector**~~ — **dropped at refinement, and
   deliberately not to be re-added.** The proposed fix is the very change T-067 already made and
   then reverted: `docsRefShapedSiteRe`'s own comment records that an inline-code exemption
   based on counting backticks per line misfired on ordinary wrapped prose in
   `cli-reference.adoc` — a continuation line opening with a closing backtick made the count
   odd and silently exempted a real `<<cmd-doctor>>` from coverage. "A guard that quietly stops
   covering things" is the defect that comment exists to prevent. Documenting the `xref:` rule
   in prose costs nothing instead: put the example in a `----` block, which both the scanner and
   the coverage invariant already skip. Item 4 stays in scope — it removes false positives
   rather than adding an exemption.

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
- **F9 from T-067's round 1 is now moot** — *(updated at refinement)*. It was excluded here as a
  scaffold-payload accuracy question rather than a checker one: `pickle scaffold docs` wrote
  "broken includes/xrefs fail the check" into other projects' justfiles. T-117 removes that
  command and the justfile recipes it appended, so the inaccurate claim ships nowhere and there
  is nothing left to fix. Nothing for this ticket to absorb — recorded so the exclusion is not
  re-litigated as still-open.
- **T-117** (`6-done/`, removed `pickle scaffold docs`) — no file overlap. It touched
  `docs/user-manual/cli-reference.adoc`, which this ticket only *reads* as checker input, and
  left `just docs-check` green. Status updated by T-117's review impact sweep.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is a root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-115-harden-docs-xref-checker
```

WIP commits are encouraged; tidy them into atomic commits before presenting (root-path child —
keep the tidied history rather than squashing). Do not push or open an MR without explicit user
approval. Under `layout = "in-tree"`, before pushing verify the remote base is not behind:
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing.

### Prerequisite gate (hard)

None. `depends-on:` is empty. T-117, which touched
`docs/user-manual/cli-reference.adoc`, is now in `6-done/` — the two were always
order-independent and nothing here waits on it. Start from a clean tree on an up-to-date
`main`, with `just test` and `just docs-check` green before the first edit (this ticket asserts
no *current* manual content is in violation — verified at refinement — so a pre-existing red is
someone else's).

### Confirmed design decisions (do not deviate without asking)

1. **Everything lands in `docs_xref_test.go`.** One file, no new package, no production code:
   the checker is a test. Nothing under `skill/` or `internal/` changes.
2. **Never reintroduce a code-span / backtick-counting exemption.** T-067 shipped one and
   reverted it — `docsRefShapedSiteRe`'s comment records that it misfired on wrapped prose in
   `cli-reference.adoc` and silently stopped covering a real `<<cmd-doctor>>`. Item 5 is dropped
   for this reason. If a future need arises to *show* a cross-reference in the manual, put it in
   a `----` block, which both the scanner and the coverage invariant already skip.
3. **The escape mask must preserve column offsets.** `assertEveryXrefSiteScanned` matches
   scanner output to source sites by `file:line:col`; a mask that shortens the line silently
   breaks that correspondence and re-opens review finding N1. Replace masked runs with an
   equal-length filler.
4. **One definition of "what is exempt", shared by both readers.** The escape mask goes
   *inside* `docsProseLines` (or a helper it calls) so `scanBook` and
   `assertEveryXrefSiteScanned` cannot drift into disagreeing — the same reason the
   literal-block logic was centralised there.
5. **`<<file.adoc#id>>` is routed to the inter-document result, not the `<<>>` result.** It is
   the same defect class as `xref:file.adoc#id[]` and must produce the same "the book is one
   document" message. Two consequences the implementer must honour: `docsXrefOccurrence` needs
   to carry the **spelling as written**, because the inter-document failure message hardcodes
   `xref:%s...` and would otherwise lie about a `<<>>` site; and
   `assertEveryXrefSiteScanned` must count a site as covered if it appears in **either**
   `xrefs` or `interDoc`, or every routed site is reported twice — once correctly, once as a
   phantom coverage gap.
6. **`link:` requires the `.adoc` extension; `xref:` does not.** `link:` is also how the manual
   writes external URLs (`link:https://…[]`), so an extensionless `link:` rule would flag every
   one. `xref:` has no such ambiguity, so it catches both `xref:file.adoc…` and the
   extensionless `xref:name#anchor[…]`. `xref:#local-anchor[text]` — an intra-document
   reference with an empty target — stays legal and must keep its `noMatch` fixture row.
7. **`docs/user-manual/concepts/lifecycle.adoc:137` is a required negative fixture.** It reads
   "Keep the short SHA even when you add the link: the board's …" — prose containing a literal
   `link:` followed by a space. Decision 6's `.adoc` requirement is what keeps it quiet; pin
   that with a fixture row rather than leaving it to luck.
8. **The unterminated-block failure is an error, not a dropped line.** `docsProseLines` gains an
   error return naming the **opening** line number; both callers surface it (`scanBook` wraps it
   with the filename, `assertEveryXrefSiteScanned` calls `t.Fatalf`). Silently returning the
   lines instead would trade one silent hole for another.
9. **Every new pattern gets both a positive and a negative fixture row** in
   `TestDocsScannerPatternsMatchWhatTheyClaim`. That table's stated purpose is catching
   loosening as precisely as narrowing; a match-only row is half a test.

### Tasks

#### Task 1 — fail on an unterminated literal block (item 9)

Do this first: it is the only item that can hide a real defect from CI today. A dead reference
after an unclosed `----` is invisible to `go test`, which is exactly what CI runs; verified at
refinement that `docsProseLines("one\n----\nhidden\nmore\n")` returns just `one` and reports
nothing.

Change `docsProseLines(content string) []docsLine` to
`docsProseLines(content string) ([]docsLine, error)`, tracking the line number where the
currently-open delimiter was opened and returning
`fmt.Errorf("unterminated literal block opened at line %d", n)` when `openDelim != 0` at EOF.
Update the two callers (`scanBook`, wrapping with the filename in the style of its existing
`reading %s: %w`; `assertEveryXrefSiteScanned`, via `t.Fatalf`) and the four subtests in
`TestDocsProseLineSelection`.

Add a subtest asserting an unterminated block **errors** and names the opening line.

#### Task 2 — close the two silent inter-document holes (item 1)

Extend inter-document detection to `xref:name#anchor[…]` (extensionless, non-empty target) and
`link:file.adoc…` (extension required — decisions 6 and 7). Prefer a small set of named
regexps over one unreadable alternation; whatever the shape, `scanBook` appends every match to
`interDoc` with correct `file:line:col`.

Add fixture rows to `TestDocsScannerPatternsMatchWhatTheyClaim` (decision 9):

- match: `xref:cli-reference#cmd-hooks[hooks]`, `link:cli-reference.adoc#id[x]`,
  `link:cli-reference.adoc[x]`
- noMatch: `xref:#local-anchor[text]` (empty target, legal intra-document),
  `link:https://example.com/x[text]` (external URL),
  `Keep the short SHA even when you add the link: the board's` (decision 7)

#### Task 3 — route `<<file.adoc#id>>` to the right message (item 2)

Add a detector for the `<<file.adoc#anchor>>` / `<<file.adoc#anchor,text>>` spelling and send
its matches to `interDoc` (decision 5). Add a `spelling`/`raw` field to `docsXrefOccurrence` so
`TestDocsNoInterDocumentXrefForm`'s message quotes what was actually written instead of
prefixing everything with `xref:`. Update `assertEveryXrefSiteScanned` to treat a site as
covered when it appears in `xrefs` **or** `interDoc`.

Regression-pin it in `TestDocsXrefCheckerCatchesTheFieldFindings`: add a page D containing
`See <<page-a.adoc#real-anchor,Page A>> for details.` and assert it lands in `interDoc`, not in
`xrefs`, and is **not** reported as an uncovered site.

#### Task 4 — anchor attribute forms (item 3)

Widen `docsAnchorLineRe` to accept `[#id,reftext]` and `[#id.role]` while still rejecting the
legacy `[[id]]` spelling, mid-line anchors and trailing prose. Keep the existing `noMatch` rows
intact — especially the two `\[[project]]` TOML-prose rows, which are real text in
`configuration.adoc:61` and `concepts/multi-project.adoc:33` — and add match rows for the two
new forms plus a noMatch row for `[#id,]`-style degenerate input if the chosen pattern rejects
it.

#### Task 5 — escape handling (item 4)

Stop reading `\<<x>>` and `+<<x>>+` as references. Implement as an equal-length mask applied
inside `docsProseLines` (decisions 3 and 4) so the scanner and the coverage invariant agree by
construction and `file:line:col` offsets are unchanged.

Extend the unresolved-reference message in `TestDocsXrefsResolve` to offer the escape as a third
remedy alongside "add `[#x]`" and "fix the target", matching the guidance its sibling message in
`assertEveryXrefSiteScanned` already gives.

Add fixtures: both escape spellings yield no reference; a real `<<x>>` on the same line as an
escaped one is still found (proving the mask is surgical, not line-wide).

#### Task 6 — the three helper fixtures (items 6, 7, 8)

- **Item 6 — scanner-level anchor inflation.** Add a negative fixture that writes
  `configuration.adoc`'s real `\[[project]]` prose to a temp file, calls **`scanBook`** over it,
  and asserts `anchors["project"]` is false. The existing pin is on `docsAnchorLineRe` alone, so
  a legacy `[[id]]` match added *inside `scanBook`* currently passes everything.
- **Item 7 — `docsProseLines` over-exemption.** Add a fixture asserting a delimiter-free
  document yields **every** line, unchanged and in order. `TestDocsProseLineSelection` pins
  `docsLiteralBlockDelim` but never "which lines count as prose", so "skip indented lines" would
  pass today.
- **Item 8 — the regression fixture re-implements its assertion.** Extract the unresolved check
  (`!anchors[x.target]` and the finding it formats) into one helper and call it from both
  `TestDocsXrefsResolve` and `TestDocsXrefCheckerCatchesTheFieldFindings`, so inverting the real
  comparison turns the fixture red.

#### Task 7 — refresh the file's header comment

`docs_xref_test.go`'s opening comment documents the checker's contract, including the
parenthetical about `[[project]]` prose and the anchor-matching policy. Bring it in line with
what now ships: the widened inter-document coverage, the escape handling, and the
unterminated-block error. Record decision 2's prohibition **in the file** — that is where a
future contributor tempted to add a code-span exemption will actually look.

### Acceptance test

All four project commands, from the repo root:

```
just build
just test
just lint
just docs-check
```

All four clean — in particular the real manual still passes with every widened pattern, which
is the assertion that no new false positive shipped (verified at refinement: the manual contains
no `link:*.adoc`, no `xref:`, no escaped `<<`, and no attribute-form anchor today).

Then prove each fix fails when it should. For items 1–4, plant the construct in a real manual
page, confirm `go test . -run '^TestDocs'` **fails with the right message**, and revert:

1. **Extensionless xref** — add `xref:cli-reference#cmd-hooks[hooks]` to
   `docs/user-manual/concepts/the-flow.adoc`. Fails with the inter-document message. Revert.
2. **`link:` form** — add `link:cli-reference.adoc#cmd-hooks[hooks]` to the same page. Fails
   with the inter-document message. Revert.
3. **`<<>>` shorthand names itself** — add `<<cli-reference.adoc#cmd-hooks,hooks>>`. Fails with
   the **inter-document** message quoting the `<<…>>` spelling as written — *not* with
   "reference-shaped site the scanner did not report", which is what it produces today, and
   **not** with both. Revert.
4. **Escapes are quiet** — add a line containing `\<<no-such-anchor-xyz>>` and `+<<also-fake>>+`.
   `go test . -run '^TestDocs'` stays **green** (this one asserts silence). Then add a real
   `<<still-fake>>` to the same line and confirm it fails naming only `still-fake`. Revert.
5. **Unterminated block** — append a lone `----` to a manual page followed by
   `<<no-such-anchor-xyz>>`. Fails naming the **opening line number**, not silently passing —
   the item-9 hole. Confirm `just docs-check` also fails (snowball's half). Revert both.
6. **Attribute-form anchors** — add `[#t115-probe,Probe]` to a page and `<<t115-probe>>`
   elsewhere. Green (the anchor now resolves); today it would fail "unresolved". Revert.

For the three helper fixtures (item 6–8), prove each new fixture is load-bearing by making the
mutation it exists to catch and confirming the suite turns red:

7. Add a legacy `[[id]]` match **inside `scanBook`** → the item-6 fixture fails.
8. Add "skip lines beginning with whitespace" to `docsProseLines` → the item-7 fixture fails.
9. Invert `!anchors[x.target]` in the extracted helper → **both**
   `TestDocsXrefsResolve` and `TestDocsXrefCheckerCatchesTheFieldFindings` fail (today only the
   former would).

Revert every mutation. Finish with a clean `git status` and all four commands green.

### Docs update (mandatory when user-facing)

No user-facing surface: `docs_xref_test.go` is a test, invoked by `just docs-check` and `go
test`. No CLI flag, command or output changes, so nothing in `docs/user-manual/` and no
`CHANGELOG.md` entry is required — this is internal tooling hardening, not a shipped behaviour
change.

The documentation that *is* mandatory is in-file: Task 7's header-comment refresh, including
decision 2's standing prohibition on a code-span exemption. If the implementer judges the
check's contributor-facing rules worth stating in the manual after all, note decision 2's
escape hatch — any `<<…>>` example must sit in a `----` block — and raise it rather than
adding a page unprompted.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean,
   and every planted construct and mutation reverted (`git status` shows only
   `docs_xref_test.go`).
2. In-file docs updated (Task 7). No manual page or CHANGELOG entry expected — say so
   explicitly in the summary if that still holds.
3. Write a **summary**: what each of the eight shipped items changed, and explicitly record the
   two scope decisions a reviewer will look for — item 5 dropped (decision 2) and item 2
   reframed from "silent hole" to "wrong message" (Description, corrected at refinement).
4. Suggest a Conventional Commit message, e.g.:

   ```
   test(docs): harden the manual's cross-reference gate (T-115)

   Catch the two inter-document spellings that passed unseen, name the
   <<file.adoc#id>> spelling instead of blaming the scanner, stop reading
   AsciiDoc's literal-reference escapes as references, and error on an
   unterminated literal block rather than silently dropping the rest of the
   file. Pin each with fixtures, including three that guard the helpers one
   level below where the previous guards sat.
   ```

5. **Tidy up before presenting** — root-path child: interactive-rebase the WIP commits into a
   small number of atomic, correctly typed commits (a reasonable split: the unterminated-block
   fix, the pattern/escape work, the helper fixtures).
6. Commit locally on the ticket branch. Do **not** push or open an MR without explicit user
   approval. Present the commit message; after approval, keep the tidied history (root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the MR. Merging is the human's.

## Review

### Round 1 — 2026-08-24 — verdict: REWORK (2 blocking)

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the
      branch in this same session, so audits (steps 2–4a) went to a fresh sub-agent briefed
      adversarially with the ticket, the branch and the child's commands. Every delegated
      finding was then re-verified by hand against a scratch probe before entering this table;
      all 8 reproduced, 0 were spurious.
- [x] Implementation audit — acceptance test re-run **verbatim** (step 2): all 9 plant-and-revert
      probes pass, each reverted with a clean `git status --short` between them
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit (step 4a) — `just docs-check` green; the plan's "no user-facing docs
      needed" claim verified true: nothing in `docs/user-manual/`, the justfile or `CHANGELOG.md`
      describes the checker
- [x] Docs-readability pass (step 4b) — **skipped: this ticket changed no `.adoc`/`.md` files**
      (`docs_xref_test.go` only), so the reviewer had nothing to read
- [x] Findings recorded with severity, class and disposition; disposition + cost lines present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket lists T-115 in `depends-on:`; the
      only reference is T-067's own Review table, which is provenance and needs no patch
- [ ] Summary + commit message & MR attributes presented for approval — **deferred: blocking
      findings, so nothing is published this round**

**Commands** (all from a clean tree on the feature branch): `just build` ✓ · `just test` ✓ ·
`just lint` ✓ · `just docs-check` ✓.

**Acceptance-test probe results** (all 9 re-run verbatim, independently):

| # | expected | actual | result |
|---|---|---|---|
| 1 | extensionless `xref:` → inter-doc message | inter-doc message, correct line | pass |
| 2 | `link:…adoc#…` → inter-doc message | inter-doc message quoting `link:` | pass |
| 3 | `<<f.adoc#id>>` → inter-doc msg only, **not** the coverage msg, **not** both | exactly one failure, quoting `<<…>>` as written | pass |
| 4 | escapes silent; + a real ref → fails naming only `still-fake` | green, then only `still-fake` | pass |
| 5 | unterminated `----` names the **opening** line; `docs-check` also fails | "opened at line 84"; snowball "unterminated listing block" | pass |
| 6 | attribute-form anchors resolve | green (fails on `main`'s checker, proving the widening) | pass |
| 7 | legacy `[[id]]` inside `scanBook` → item-6 fixture red | that fixture only | pass |
| 8 | skip-indented in `docsProseLines` → item-7 fixture red | that fixture only | pass |
| 9 | invert the extracted helper → **both** tests red | both red | pass |

**Findings**

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | design | new ticket (T-118) | The extensionless `<<name#anchor>>` shorthand still produces the misleading "scanner did not report this site" message that item 2 existed to delete — `docsInterDocAngleRe` requires `.adoc` | `<<cli-reference#cmd-hooks,hooks>>` scores `angle=false … site=true`; planted, it fails with the coverage message blaming `docsXrefTargetRe` | Route the extensionless angle form to `interDoc` too; decision 6 already established the extensionless spelling is the one contributors reach for |
| F2 | **blocking** | correctness | — | `docsInterDocLinkRe` false-positives on any **external URL ending in `.adoc`**, defeating decision 6's own stated rationale ("`link:` is also how the manual writes external URLs… an extensionless rule would flag every one") | `link:https://x/y/README.adoc[src]` → `link=true`; planted, the build fails claiming "no standalone `https://x/y/README.pdf` artifact" — nonsense output | Exclude `:` from the class: `link:([^\[\s:]+\.adoc)[#\[]`, plus a `noMatch` fixture row for an external `.adoc` URL |
| F3 | non-blocking | test-gap | new ticket (T-118) | Path-form and underscore-leading extensionless `xref:` reach **no detector at all** — the ticket's headline "only genuine silent hole" survives in narrower form | `xref:concepts/lifecycle#stage[x]` and `xref:_foo#bar[x]` both score `inter/bare/link/angle/strict/**site** = all false`; asymmetric with sibling `docsInterDocXrefRe`, which allows `/` | Widen `docsInterDocXrefBareRe`'s class toward the sibling's, keeping the empty-target exclusion |
| F4 | **blocking** | test-gap | — | **Decision 9 not honoured**: `docsEscapedXrefBackslashRe` and `docsEscapedXrefPassthroughRe` are new patterns with **no rows** in `TestDocsScannerPatternsMatchWhatTheyClaim`, the table decision 9 designates for exactly this | `docs_xref_test.go:138-139` define them; the table's five `re:` rows (`:584,611,627,640,655`) cover neither | Add a positive and a negative row for each, per decision 9's "a match-only row is half a test" |
| F5a | non-blocking | stale-xref | fixed inline | `docsMaskEscapedXrefs`'s comment claimed the mask "only ever shortens what a pattern can match, never widens it", and that `_` "cannot itself look like a reference" — both false | `_` is inside `docsXrefTargetRe`'s target class | Corrected in `27bfc4a` |
| F5b | non-blocking | correctness | new ticket (T-118) | The mask can **manufacture** a phantom reference: an escape nested in a live reference masks into a valid-looking one | `<<\<<x>>>>` → `<<______>>`, matched by `docsXrefTargetRe` as a reference to anchor `______` | Use a filler character outside that class (a space defeats both patterns) while still preserving column offsets (decision 3) |
| F6 | non-blocking | design | new ticket (T-118) | The widened `docsAnchorLineRe` is approximate both ways: accepts degenerate `[#id,,]` / `[#id, ]`, still rejects legal `[#id.role1.role2]` and `[#id%breakable]` | probe: first two `true`, last two `false` | The rejection direction has teeth — a missed legitimate anchor turns every reference to it into a false "unresolved", which is item 3's own risk |
| F7 | non-blocking | stale-xref | fixed inline | The inter-document failure header still read "`%d inter-document xref: form(s)`" after `link:` and `<<>>` spellings began routing to it | visible in probes 2 and 3 output | Corrected in `27bfc4a` |
| F8 | non-blocking | stale-xref | fixed inline | `docsLine`'s comment still described `text` as verbatim source after masking made it not, though failure messages echo slices of it | `docs_xref_test.go:326` | Corrected in `27bfc4a` |

**Verified sound** (attacked, found correct — recorded so a later reviewer need not redo it): no
double-counting, each construct routed by exactly one pattern (`xref:x.adoc#a` is correctly
rejected by the bare pattern); decision 5 holds in **both** halves — `<<file.adoc#id>>` lands in
`interDoc` and is *not* double-reported as an uncovered site; `docsUnresolvedXref` is load-bearing
in both callers; all three helper fixtures and page D are load-bearing (deleting the pattern each
guards reddens it); the mask's pre-computed-index loop is safe for adjacent and nested escapes
and preserves byte offsets under multi-byte UTF-8; no callers of the changed helpers exist outside
this file; history is three atomic, correctly typed commits.

**Disposition summary:** 9 findings — 2 blocking (F2, F4 → rework, not dispositioned); 7
non-blocking → 3 **fixed inline** (F5a, F7, F8, in `27bfc4a`), 4 **new ticket** (F1, F3, F5b, F6
→ **T-118**, batched by theme), 0 folded, 0 noted.

cost: estimated M, actual M

### Rework — 2026-08-24 — fixed F2 and F4

Scope was exactly the two blocking findings, nothing else, on the same
`feat/T-115-harden-docs-xref-checker` branch (commit `92a8456`).

| id | fix | verification |
|---|---|---|
| F2 | `docsInterDocLinkRe`'s target class now excludes `:` too (`link:([^\[\s:]+\.adoc)[#\[]`), so an external URL ending in `.adoc` no longer matches; added a `noMatch` fixture row (`link:https://example.com/x/y/README.adoc[the source]`) | Re-ran the exact F2 probe by hand: `link:https://x/y/README.adoc[src]` now scores `link=false`; both real forms (`link:cli-reference.adoc[x]`, `link:cli-reference.adoc#id[x]`) still score `link=true`. Re-planted `link:cli-reference.adoc#cmd-hooks[hooks]` in a real manual page — still fails with the inter-document message, confirming no regression |
| F4 | Added two new cases to `TestDocsScannerPatternsMatchWhatTheyClaim`: "escaped cross-reference (backslash)" and "escaped cross-reference (passthrough)", each with 2 match rows and 3 noMatch rows (including cross-checks that each pattern rejects the other's spelling) | `go test . -run '^TestDocsScannerPatternsMatchWhatTheyClaim'` — both new subtests pass |

All four commands re-run clean: `just build` ✓ · `just test` ✓ · `just lint` ✓ · `just docs-check`
✓. All 9 original acceptance-test probes still pass (spot-checked probes 2 and 4, the two this
rework touched, by hand in addition to the full suite). `git status --short` clean; F1, F3, F5b,
F6 are untouched (T-118's scope, not this rework's).

### Round 2 — 2026-08-24 — scoped re-review — verdict: PASS

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent performed the
      rework in this same session, so the audit went to a fresh sub-agent briefed adversarially
      with the ticket's Round 1 findings and the Rework write-up, instructed not to take the
      write-up at face value. Its one new finding (F9, below) was re-verified by hand before
      entering this table.
- [x] Scoped implementation audit (step 2, scoped): **only F2 and F4 re-verified**, per this
      being a re-review of a prior `5-rework/` round — not a full re-audit
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated — T-118 amended with the new item (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket lists T-115 in `depends-on:`;
      T-067's reference is provenance only
- [ ] Summary + commit message & MR attributes presented for approval — pending, see Finish below

**F2 — RESOLVED.** `docsInterDocLinkRe` now excludes `:` from its target class. Verified beyond
the rework's own fixture: adversarial probing of a port number, query string, fragment,
uppercase `.ADOC`, a Windows-style path, a `:` appearing later in the string, and another scheme
(`s3://`) all correctly rejected; both real `link:file.adoc[...]` forms still match; a real
external-URL plant in a manual page now passes where it previously failed, reverted cleanly.

**F4 — RESOLVED.** Both new fixture rows are real and load-bearing, not relabeled duplicates:
mutating `docsEscapedXrefBackslashRe` to drop the backslash requirement turned its new fixture
row red; mutating `docsEscapedXrefPassthroughRe` to drop the trailing `+` turned its new fixture
row red. Both mutations reverted, tree confirmed clean after each.

**Scope discipline held.** The rework commit (`92a8456`) touches only `docsInterDocLinkRe` and
the fixture table. `docsInterDocAngleRe`, `docsInterDocXrefBareRe`, `docsMaskEscapedXrefs` and
`docsAnchorLineRe` — the code F1/F3/F5b/F6 are about — do not appear in its diff.

**New finding from adversarial probing of the F2 fix:**

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F9 | non-blocking | correctness | folded (T-118) | The F2 fix closes the **scheme-based** false positive (`link:https://host/x.adoc[...]`) but not a **scheme-relative** one: `link://host/x.adoc[...]` has no `:` before the target's first char sequence that would trip the new exclusion | probed directly: `link=true` for `link://host/x/README.adoc[x]` | Exclude a leading `//` too, or require the target start with neither `:` nor `//`; folded into T-118 item 5 (same defect class as its item 1: a residual gap in a fix this line already narrowed once) |

**Disposition summary (round 2):** 1 finding, 0 blocking, 1 non-blocking → **folded** (F9 →
T-118 item 5). F2 and F4 both confirmed resolved; no blocking findings remain.

cost: no change (fix already accounted for in round 1's actual)

## History

- 2026-08-22 — created (TO DO). source: review: batched from T-067's three review rounds — round 1 findings F3–F6 (pattern coverage, code-span exemptions, fixture structure), round 3 findings N5–N6, and round 4 findings R1–R4 (helper fixtures, AsciiDoc escapes, and the unterminated-block hole that is invisible to CI). Batched by theme per rules §5 rather than filed one per finding; none is a live defect, all require modifying the checker or naming a construct the manual does not yet contain
- 2026-08-24 — TO DO → READY: plan complete
- 2026-08-24 — plan amended inline: T-117's status corrected from `2-ready/` to `6-done/` in the
  soft-couplings note and the prerequisite gate, by T-117's review impact sweep (rules §8)
- 2026-08-24 — READY → IN DEVELOPMENT: picked up
- 2026-08-24 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-24 — IN REVIEW → REWORK: review round 1: 2 blocking (F2 link: false-positives on external .adoc URLs; F4 decision 9 unmet for the two escape patterns)
- 2026-08-24 — REWORK → IN REVIEW: rework: F2 and F4 fixed
- 2026-08-24 — IN REVIEW → DONE: review round 2 (scoped): F2 and F4 resolved; F9 non-blocking, folded into T-118
