---
id: T-067
title: docs-check passes on a dead cross-reference: no link/anchor validation anywhere in the docs pipeline
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-067 — docs-check passes on a dead cross-reference: no link/anchor validation anywhere in the docs pipeline

## Outcome

After this ships, `just docs-check` fails the build when the manual contains a dead cross-reference, instead of silently rendering past it the way it does today.

## Description

`just docs-check` (`snowball check`, `justfile:23-24`) renders the manual and discards the
output. Rendering succeeds whether or not a cross-reference resolves, so **the docs gate is
blind to dead links** — the one class of docs defect a build gate is normally trusted to catch.

Measured 2026-08-05 while registering `docs/user-manual/concepts/agent-session-workflow.adoc`.
`<<the-flow>>` was replaced with `<<no-such-anchor-xyz>>` in a page included by
`docs/user-manual.adoc`, and:

| command | result on the injected dead xref |
|---|---|
| `just docs-check` (`snowball check`) | **passes** |
| `asciidoctor --failure-level=WARN -o /dev/null docs/user-manual.adoc` | exit 0, silent |
| `asciidoctor -b docbook -o /dev/null docs/user-manual.adoc` | exit 0, silent |

So neither the configured docs command nor the underlying renderer objects: asciidoctor emits
`[no-such-anchor-xyz]` into the output and moves on. A reviewer following the review protocol's
step 4a (*"broken links/anchors"* is named there explicitly) has no command that answers the
question, and the whole-tree sweep it mandates is currently done by eye.

### Why this is worth a ticket rather than a note

The manual is not small and it is heavily cross-linked: **22 anchors, 14 `<<…>>` targets** at
filing, plus `include::` wiring in `docs/user-manual.adoc` that a page can silently fall out of.
Two adjacent failure modes were both hit within the same hour:

1. `agent-session-workflow.adoc` was committed **unregistered** — no `include::`, so `snowball`
   never built it and its xrefs were never even parsed. `docs-check` passed the whole time.
2. Nothing linked **to** the page once registered; it was reachable only from the TOC.

Both are cheap to detect mechanically and invisible to the current gate. The repo already
guards *payload prose* this way — `TestPayloadDispositionVocabulary` and
`TestPayloadDefersToProjectConfig` (`internal/install/install_test.go`) assert on shipped text —
so a docs-tree guard is consistent with existing machinery, not a new kind of thing.

### Shape of the fix (for refinement)

A ~15-line extraction is enough for the core check, and was used to verify the manual by hand at
filing: collect every anchor definition (`[#id]`, `[[id]]`) and every `<<id>>` / `<<id,text>>`
target across `docs/user-manual.adoc` and its includes, then diff the sets. Refinement should
settle:

- **Where it lives.** A Go test under `internal/` reading `../../docs/` (like the payload guards,
  which reach the repo root via `payloadRoot()`), or a `just` recipe? The Go test runs in `just
  test` and in CI for free; a recipe keeps docs tooling together but needs wiring into
  `docs-check`.
- **How much it checks.** Unresolved xref targets are the core. Candidates to include or reject
  explicitly: orphan pages (an `.adoc` under `docs/user-manual/` with no `include::`), pages with
  no inbound xref (weaker — the TOC is a legitimate entry point), and `link:`/`http` URLs
  (out of scope: needs the network).
- **Inter-document `xref:` forms are a second, distinct class** (added 2026-08-06 by the T-057
  review, finding N3). The manual is **one assembled book** (`docs/user-manual.adoc` +
  `include::`), so every cross-file reference must be `<<anchor>>`. Writing
  `xref:cli-reference.adoc#cmd-hooks[…]` instead is *not* an unresolved anchor — asciidoctor
  resolves it happily, to `cli-reference.pdf#cmd-hooks` in the PDF and `cli-reference.epub#cmd-hooks`
  in the EPUB, **neither of which exists**. T-057 shipped two such links and `just docs-check`
  passed; they were caught only by grepping the rendered artifacts, and were fixed in `a7e2ada`.
  A set-difference checker over `[#id]`/`<<id>>` would have missed both, so the checker must also
  **reject the `xref:<file>.adoc#…` form outright** for any file that is `include::`-d into the
  book — the cheapest half of this ticket, and the one with a proven miss.
- **Auto-generated ids.** Asciidoctor generates section ids when an explicit `[#id]` is absent;
  the checker must not report a reference to one of those as unresolved. At filing every
  `<<…>>` target in this manual resolves to an *explicit* anchor, so the simple version is
  correct today — but that is a property to assert, not to assume.

### Also in scope: `docs-check` does not run in CI at all

Folded in from T-057's pickup gate (finding F11, 2026-08-05). Even a *perfect* anchor checker
wired into `just docs-check` would not run automatically: `.github/workflows/ci.yml` runs
`go vet` / `gofmt` / `go test` / `go build` only, so `snowball check` is a local-only gate that a
contributor (or an agent) can skip by never invoking it. This ticket's fix therefore has two
halves, and the second is what makes the first load-bearing:

1. the anchor/xref check itself (above);
2. **a CI job that runs the docs gate** — which also settles the "where it lives" question above:
   a Go test under `internal/` reaching `../../docs/` runs in CI *for free* today, whereas a
   `just docs-check` extension needs `snowball` installed in the workflow. Refinement should price
   both (a Go test needs no new CI dependency; a `snowball` job also validates the render).

### Applicability-gate addendum (2026-08-22 pickup)

A fresh sub-agent re-verified the plan's load-bearing claims against the live tree at pickup:
book-structure figures (13 files, 30 anchors, 86 `<<…>>` occurrences, 0 unresolved, 0 `xref:`
forms, 0 orphans), `[[id]]`-unused, the precedent files, and the CI premise all confirmed
exactly. Three non-blocking findings, all dispositioned note-and-close, none changing the plan:

1. The Description/plan cite `justfile:23-24` for the `docs-check` recipe; it has drifted to
   `justfile:30-31` (content unchanged, only the line numbers moved).
2. Two literal `\[[project]]` strings in `configuration.adoc`/`multi-project.adoc` describe
   TOML array-of-tables syntax in backtick-quoted prose, not AsciiDoc `[[id]]` anchors — noted
   as a comment in `docs_xref_test.go` so a future naive grep for `[[` isn't misread as
   contradicting decision 4.
3. `.github/workflows/ci.yml`'s `build-test` job has grown a `board audit` step and gained two
   sibling jobs (`goreleaser-check`, `ci-surface`) since the plan was written — decision 1's
   "no new CI job needed" conclusion is unaffected: `go test ./...` still runs unconditionally
   in `build-test`.

### Soft couplings

- **T-066** — also docs-surface, but a *content* gap (undocumented flags, a dropped command
  cited in the payload). Different theme: that ticket fixes wrong text, this one adds the gate
  that would flag a class of it. No file overlap (T-066 owns `cli-reference.adoc`).
- **T-048** — set up the snowball render/`docs-check` pipeline this ticket extends.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-067-docs-xref-check
```

Root-path child. Tidy WIP commits before presenting (Finish, below).

### Prerequisite gate (hard)

None.

### Confirmed design decisions (do not deviate without asking)

1. **Shape: a Go test at the repo root, `package main`**, following T-099's precedent
   (`payload_lint_test.go`) rather than a `just` recipe extension. This settles both of the
   Description's open questions at once: it runs in `go test ./...`, which
   `.github/workflows/ci.yml`'s existing `build-test` job's `test` step already executes — **no
   new CI job is added**, because none is needed. (Rejecting the recipe-only alternative for the
   same reason T-099 did: an optional local recipe can be skipped or silently degrade; a test in
   `go test ./...` cannot.)
2. **`just docs-check` is still extended to also run the new test(s)**, so the ticket's own
   Outcome sentence ("`just docs-check` fails... on a dead cross-reference") is literally true
   for a human/agent who runs only that recipe and never `just test` — even though the check
   already rides along in CI via `go test ./...` regardless. This is deliberate duplication of
   coverage across two entry points, not a new dependency (no `snowball` involvement).
3. **Scope of the checker**, resolving the Description's three open candidates:
   - **In scope:** unresolved `<<id>>`/`<<id,text>>` targets (core); the `xref:<file>.adoc#...`
     inter-document form, rejected outright for any file in the assembled book (T-057 finding
     N3 — proven to slip past `snowball check` twice already); orphan pages — a `.adoc` under
     `docs/user-manual/` (recursive) that no `include::` in the book's include graph reaches
     (the exact failure mode that motivated this ticket at filing).
   - **Out of scope**, per the Description's own steer: pages reachable only from the TOC (no
     inbound `<<xref>>` from sibling prose) — the TOC is a legitimate entry point, not a defect;
     and `link:`/bare URL validation — needs the network.
4. **Anchor matching is explicit-`[#id]`-only** (verified: `[[id]]` form is unused anywhere in
   the book today, `[#id]` is the only style present). The checker does **not** compute
   asciidoctor's auto-generated section-id algorithm. Per the Description, this is a property to
   assert, not assume: the checker's own passing-today result (zero unresolved targets against an
   explicit-only anchor set, verified at refinement against the live tree) **is** that assertion,
   continuously, on every run. If a future xref is ever written to rely on an auto-generated id,
   the fix is to add an explicit `[#id]` to that heading, not to teach the checker
   asciidoctor's slugger — recorded as a comment in the new file so a future false-positive
   report is not mistaken for the checker being broken.
5. **Book-file discovery follows `include::` directives from `docs/user-manual.adoc`
   recursively** (there is only one book, per `snowball.yaml`), resolving each target relative to
   the *including* file's own directory (standard AsciiDoc include resolution), deduplicated.
   Verified at refinement with a throwaway script against the live tree: 13 files reachable
   (`user-manual.adoc`, `attributes.adoc`, and 11 files under `docs/user-manual/`), 30 explicit
   anchors, 86 `<<…>>` occurrences (24 distinct targets), 0 unresolved, 0 `xref:*.adoc` forms, 0
   orphans. **Amended 2026-08-22** (see History; original wording: "the new tests must reproduce
   these exact figures as their reference point (a material drift is a signal the parser
   mis-walked the tree, not that the docs regressed)"). The tests instead assert **structural
   invariants over the scanner's patterns**: every literal `<<` site in the book must be consumed
   by the xref pattern, and each pattern carries positive *and* negative fixtures. The figures
   above stay in the code as history, asserted by nothing. Rationale, from two rounds of review:
   a count reproduces the tree's size on one day, and both ways of asserting it fail — exact
   equality cries wolf on every legitimate docs commit (which trains contributors to bump the
   constant unread), while a floor turns docs growth into slack and is blind to the inflation
   that makes dead references *resolve*. An invariant over the patterns catches narrowing and
   loosening alike, at any book size, and never fires on ordinary work.

### Tasks

#### Task 1 — the checker: `docs_xref_test.go` (repo root, `package main`)
Implement, alongside table-driven unit tests for each piece:
- `bookFiles(master string) ([]string, error)` — recursively follows `^include::([^\[\]]+)\[`
  lines, resolving each path relative to the including file's directory, returning the
  deduplicated, visitation-ordered file list (decision 5).
- A scanner over that file list producing: the set of explicit anchor ids (`^\[#([A-Za-z0-9_-]+)\]\s*$`,
  decision 4); every `<<id>>`/`<<id,text>>` occurrence with its file:line (`<<([A-Za-z0-9_-]+)(?:,[^>]*)?>>`);
  and every `xref:<file>.adoc#...`/`xref:<file>.adoc[...]` occurrence with its file:line
  (`xref:([^\[\s]+\.adoc)[#\[]`).
- `TestDocsXrefsResolve` — walks the real `docs/user-manual.adoc` book; fails listing every
  `file:line: <<target>> does not resolve to any [#target] anchor in the assembled book` (mirror
  `payloadLintFinding`'s teaching-style message, decision — same file's `String()` shape).
- `TestDocsNoInterDocumentXrefForm` — same book; fails listing every `file:line: xref:<file>.adoc...`
  match with the reason (renders as a dead per-chapter link once split into PDF/EPUB, T-057
  finding N3) and the fix (use `<<anchor>>` instead — the book is one document, not many).
- `TestDocsUserManualHasNoOrphanPages` — `filepath.WalkDir("docs/user-manual", ...)` for every
  `*.adoc`, asserts each is in `bookFiles()`'s result; fails naming the orphan and prescribing an
  `include::` line in `docs/user-manual.adoc` (or a page already included).
- `TestDocsXrefCheckerCatchesTheFieldFindings` — a synthetic fixture in `t.TempDir()` reproducing
  both proven-live bugs from the Description at once: a two-file mini "book" where one page has
  `<<no-such-anchor-xyz>>` and another has `xref:sibling.adoc#real-anchor[]` pointing at a real
  anchor in the *other* file; asserts both are flagged (this is the regression proof — it fails
  today's *code* even though it exercises no real docs content).

#### Task 2 — wire `just docs-check` to the new tests (decision 2)
In `justfile` (`:23-24`):
```
docs-check:
    snowball check
    go test . -run '^TestDocs'
```

### Acceptance test

```
just build
go test . -run '^TestDocs' -v
just docs-check
just test
just lint
```
All clean. In addition, prove the regression bar by hand once: temporarily reintroduce the
original field defect (replace one real `<<the-flow>>` with `<<no-such-anchor-xyz>>` in
`docs/user-manual/concepts/the-flow.adoc`, or any live anchor target) and confirm both
`go test . -run '^TestDocs'` and `just docs-check` now fail with a message naming the file, line,
and dangling target; then revert the change before committing.

### Docs update (mandatory when user-facing)

`docs/README.adoc`'s "Validating" section (`:53-57`, verified at refinement) currently claims
`just docs-check` "fail[s] on any warning (broken includes, dangling cross-references)" — which
is the exact claim this ticket's Description proves false at filing (`snowball check` passes on
an injected dead xref). Update that paragraph to describe the real two-part check post-fix:
`snowball check` (render-and-discard) plus the new Go test (unresolved `<<xref>>` targets, the
`xref:<file>.adoc#...` form, and orphan pages). While there, add
`concepts/agent-session-workflow.adoc` to the "Layout" tree listing (`:34-49`) — it is missing
from that listing today, the same gap this ticket exists to catch mechanically going forward.

### Finish (mandatory)

1. Acceptance test green, including the hand-verified regression check (reverted before commit).
2. Docs updated if a contributor-facing pipeline description exists (see above).
3. Write a summary: confirm the two coupled Description questions ("where it lives" / "does it
   run in CI") are both answered by decision 1 — no new `.github/workflows/ci.yml` job was
   needed or added.
4. Suggested commit message:
   ```
   test(docs): fail the build on a dead cross-reference or a foreign xref: form (T-067)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-067 in-review --reason "acceptance green"`.

## Review

### Round 1 — 2026-08-22 — verdict: REWORK (1 blocking)

- [x] Reviewer independence settled (step 0): **delegated**. The reviewing agent authored the
  branch in this same session, so the audits were handed to two independent sub-agents spawned
  fresh and briefed adversarially — one for steps 2–3 (implementation + quality), one for steps
  4–4a (consistency + docs). Classification, severity, disposition and the moves stayed with the
  orchestrating reviewer. **Every delegated finding was re-verified by hand before entering the
  table below**; one was dropped as not a defect (see *Re-verification notes*).
- [x] Implementation audit — acceptance test re-run verbatim (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on `docs/README.adoc` (step 4b) — run; one suggestion touching this
  branch's own prose is pending user approval, the rest target pre-existing text
- [x] Findings recorded with severity, class and disposition; disposition + cost lines present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] Board regenerated by the move (step 7)
- [x] Impact sweep done (step 8) — patches deferred to the concluding re-review, see below
- [ ] Publish gate (step 9) — **not reached**: a blocking finding sends the ticket to rework first

**Acceptance test, re-run verbatim:** `just build` ✓ · `go test . -run '^TestDocs' -v` ✓ (4/4 PASS)
· `just docs-check` ✓ · `just test` ✓ · `just lint` ✓ · `gofmt -l .` clean. The hand-verified
regression check was reproduced independently: injecting `<<no-such-anchor-xyz>>` at
`docs/user-manual/concepts/agent-session-workflow.adoc:7` fails both `go test . -run '^TestDocs'`
and `just docs-check`, each naming file, line and dangling target; tree reverted clean.
Live-tree figures reproduce decision 5's reference exactly (13 book files, 30 anchors, 86
`<<…>>` occurrences, 0 unresolved, 0 `xref:` forms, 0 orphans).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | test-gap | — | **T-067 decision 5** requires the new tests to reproduce the refinement's reference figures "as their reference point"; no figure is asserted anywhere, so the core guard can pass while checking nothing | `docs_xref_test.go:132-155` contains no count assertion. Demonstrated: narrowing `docsXrefTargetRe` to drop the `<<id,text>>` form silently removes 4 live occurrences from coverage and **all four tests still pass** | Assert the reference counts (book files, anchors, `<<…>>` occurrences, distinct targets) in `TestDocsXrefsResolve`, failing with decision 5's own rationale — drift means the parser mis-walked the tree, not that the docs regressed |
| F2 | non-blocking | test-gap | new ticket | Task 1 required "table-driven unit tests for each piece"; none exist — `bookFiles` dedup/cycle/relative-resolution and each regex have no direct unit coverage | no table-driven test anywhere in `docs_xref_test.go` | Add table-driven cases per helper |
| F3 | non-blocking | test-gap | new ticket | The regression-proof fixture re-implements the unresolved check inline rather than calling shared code, so inverting the real comparison leaves it green | `docs_xref_test.go:273-283` duplicates the logic at `:145-146` | Extract `unresolved(anchors, xrefs)` and call it from both the real test and the fixture |
| F4 | non-blocking | design | new ticket | The shorthand inter-document form `<<file.adoc#anchor>>` escapes **both** detectors — the same defect class as T-057 finding N3, in AsciiDoc's other spelling | verified: `<<cli-reference.adoc#cmd-hooks,hooks>>` matches neither `docsXrefTargetRe` (`:46`, excludes `.`/`#`) nor `docsInterDocXrefRe` (`:47`, requires literal `xref:`) | Flag any `<<…>>` target containing `.adoc` or `#` as an inter-document form |
| F5 | non-blocking | design | new ticket | Valid anchor spellings `[#id,reftext]` and `[#id.role]` are unmatched, so a legitimate anchor written either way turns every reference to it into a false "unresolved" | `docs_xref_test.go:45`; both forms verified non-matching; neither appears in the manual today | Widen the anchor pattern, or record the limitation beside the existing `[[id]]` note |
| F6 | non-blocking | design | new ticket | No code-span/fence exemption: the inter-document detector matches inside inline code and listing blocks, so documenting this very rule in the manual would false-positive with no escape | verified `` `xref:foo.adoc#x[]` `` matches; contrast the precedent's `insideBackticks`/`inFence` machinery in `payload_lint_test.go` | Reuse the precedent's context tracking, or state in a comment why it is deliberately omitted |
| F7 | non-blocking | stale-xref | fixed inline | The inter-document failure message asserted the target "is `include::`-d into this book"; the detector flags any `xref:*.adoc`, so the claim (and its prescribed fix) is wrong for an out-of-book target | `docs_xref_test.go:181` | Reworded to "targets another .adoc file" — **done in `ef9bdfc`** |
| F8 | non-blocking | stale-xref | fixed inline | `justfileRecipe`'s doc comment claimed the scaffolded bodies mirror this repo's justfile "verbatim"; this branch made that false by giving `docs-check` a second line | `internal/scaffold/scaffold.go:121-122` | Narrowed the comment to name the divergence — **done in `ef9bdfc`** |
| F9 | non-blocking | docs-gap | noted | `pickle scaffold docs` still writes `# Validate the AsciiDoc manual via snowball (broken includes/xrefs fail the check)` into **other projects'** justfiles — the exact false claim this ticket disproves, in the one surface that reaches foreign workspaces | `internal/scaffold/scaffold.go:136` | Drop `/xrefs`. Pre-existing, so §5's inline bar (causation, not size) excludes it from `fixed inline` |
| F10 | non-blocking | docs-gap | noted | `docs/README.adoc`'s "Concepts" bullet list still omits `agent-session-workflow.adoc`, four lines above the Layout tree this branch did add it to | `docs/README.adoc:16-21` vs `:47` | Add the `link:` entry. Pre-existing; the plan's docs task named only the Layout tree |
| F11 | non-blocking | docs-gap | fixed inline | No `CHANGELOG.md` entry and no recorded decision to skip one; `pickle changelog check` flags T-067 as an unanswered candidate | `./pickle changelog check` → "1 candidate(s) shipped but not named in Unreleased: T-067" | **Recorded here: no entry.** Contributor tooling only, no change to the shipped binary — same call and same reason as T-088 and T-099 |
| F12 | non-blocking | other | noted | `docs/attributes.adoc` is in the book but outside `docs/user-manual/`, so losing its `include::` would not be flagged as an orphan; `docs/proposals/` is unchecked entirely | `docs_xref_test.go:39-40` (orphan walk is scoped to `docsUserManualDir`) | Acceptable under decision 3's explicit scoping; recorded so a later reviewer can promote it |

**Disposition summary:** 12 findings — 1 blocking (F1, → `5-rework/`); 11 non-blocking: 3 fixed
inline (F7, F8, F11), 5 → one batched new ticket (F2–F6, *harden the docs xref checker*, spawned
at the concluding re-review so the rework cannot moot its items first), 3 noted (F9, F10, F12).

`cost: estimated S, actual S` — provisional; the concluding re-review finalises it after the
F1 rework lands.

#### Re-verification notes (step 0's "delegation buys independence, not accuracy")

- **One delegated finding was dropped.** An auditor read `docs_xref_test.go:93-95`'s comment
  ("mirrors `payloadLintFinding`'s file:line + reason shape") as promising a `String()` method
  the type does not have. Re-read by hand: the comment describes the *message* shape, which the
  inline `fmt.Sprintf` calls do produce. Accurate as written — not recorded as a finding.
- **F1's blast radius was narrowed by hand.** The auditor called the core check a silent no-op.
  Two backstops do exist and were verified: `TestDocsXrefCheckerCatchesTheFieldFindings` catches
  a regex that matches *nothing*, and `TestDocsUserManualHasNoOrphanPages` catches a *mis-walked
  include tree* (both confirmed by deliberate sabotage). The residual gap F1 must close is the
  **partial** narrowing — a pattern that still matches the fixture but silently drops real
  occurrences — which is precisely the case decision 5's figures were specified to catch.

### Rework pass — 2026-08-22 — F1 fixed

Scope was F1 alone; no other finding was touched. Fixed in `09f94c8` on the same branch.

**F1 — resolved.** `assertBookCoverageIntact` (`docs_xref_test.go`) now runs inside
`TestDocsXrefsResolve`, before the resolution loop, asserting that the scan still reaches at
least the figures decision 5 measured at refinement: 13 book files, 30 anchors, 86 `<<…>>`
occurrences, 24 distinct targets. The core check can no longer report "no unresolved xrefs"
over a book it never actually read.

**One interpretation to flag for the re-review.** Decision 5's wording is "must reproduce these
exact figures as their reference point". They are asserted as **floors**, not equalities. The
reasoning: the failure mode decision 5 names is the parser mis-walking the tree — coverage
*collapsing* — and documentation only grows, so any drop is the signal. Exact equality would
additionally fail on every legitimate docs addition, and a guard that cries wolf on ordinary
work trains contributors to bump the constant without reading it, which is precisely how it
would rot back into the decoration F1 found. The exact figures are still what the floors are
calibrated to and still appear in the file as the reference point, so this is read as serving
decision 5 rather than departing from it — but it is recorded here rather than left silent, and
the re-review should overrule it if it disagrees.

**Verification that the fix bites** (each by deliberate sabotage, reverted after):

| sabotage | before F1's fix | after |
|---|---|---|
| drop the `<<id,text>>` form from the xref pattern | all 4 tests **passed** (4 live occurrences silently unchecked) | **FAIL** — "occurrences scanned = 82, expected at least 86"; "distinct targets = 23, expected at least 24" |
| include walk stops matching `[leveloffset=+1]` | orphan test only | **FAIL** — all four counts, led by "book files = 2, expected at least 13" |
| anchor pattern narrowed to `[A-Za-z]+` | resolution failures only | **FAIL** — "anchors = 5, expected at least 30", plus 67 unresolved |
| **a normal docs addition** (+1 anchor, +2 xrefs on a live page) | n/a | **passes** — no false positive |

The distinct failure signatures are deliberate: files dropping with everything else points at
the include walk, occurrences dropping alone points at the xref pattern.

Acceptance test re-run verbatim after the fix: `just build` ✓ · `go test . -run '^TestDocs' -v` ✓
(4/4) · `just docs-check` ✓ · `just test` ✓ (20 packages, 0 failures) · `just lint` ✓ ·
`gofmt -l .` clean.

Also applied in the same commit, closing round 1's step 4b: the approved docs-readability
suggestion to `docs/README.adoc`'s Validating section. Not a finding — readability suggestions
never enter the findings table.

### Round 2 — scoped re-review — 2026-08-22 — verdict: REWORK (3 blocking)

Scope: verify F1 only. Reviewer independence: **delegated** again (the orchestrating reviewer
authored the rework) — one sub-agent for the scoped audit, a second asked to adjudicate the
interpretation call the rework flagged, briefed neutrally with both readings. Every finding
re-verified by hand; one was **not reproducible as first reported** and was only recorded after
finding the faithful reproduction (see below).

**F1 is resolved as written — and the fix is inadequate.** `assertBookCoverageIntact` does run
unconditionally, and it does catch all three collapse modes round 1 demonstrated. But the floor
design fails in two ways neither the rework nor its author's own testing surfaced.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F13 | blocking | test-gap | — | **The floors expire on the next ordinary docs commit.** Calibrated to a frozen measurement, they turn documentation growth into slack, and F1's exact defect walks back in through it | Verified by hand: added one legitimate page (5 anchors, 5 refs, `include::`d) — suite green, correct. Then re-applied F1's exact defect (drop the `<<id,text>>` form): **all four tests pass**; 87 ≥ 86 and 28 ≥ 24, with 4 live occurrences silently unchecked again | Replace the count thresholds with a growth-proof invariant: assert every literal `<<` in the book is consumed by `docsXrefTargetRe`. Unmatched `<<` means the pattern narrowed, at any book size |
| F14 | blocking | test-gap | — | `TestDocsNoInterDocumentXrefForm` still has F1's exact vacuous shape, untouched by the rework, and no floor is even possible there (the live count is legitimately 0). Its fixture exercises only the `#` spelling | Verified: narrowing `docsInterDocXrefRe` to `xref:(…\.adoc)[#]` keeps the fixture green; a live `xref:cli-reference.adoc[the CLI reference]` in `quickstart.adoc` then passes **both** `go test` and `just docs-check` — the T-057 N3 class the ticket calls "the half with a proven miss". The shipped pattern does catch it; nothing proves it stays caught | Add a second fixture page using the bare `xref:sibling.adoc[text]` spelling, so both spellings are proven-caught |
| F15 | blocking | test-gap | — | **Floors are blind to inflation**, and inflation is the dangerous direction for anchors: a bigger anchor set means false *resolution* | Verified (see note): adding unanchored `[[id]]` support — a plausible way to accept the legacy spelling — swallows the backticked TOML prose `` `\[[project]]` `` as an anchor named `project`; a genuinely dead `<<project>>` in `configuration.adoc` then passes green. The unmodified checker catches it correctly | A negative fixture asserting `` `\[[project]]` `` is *not* an anchor — i.e. the table test of F16, which subsumes this |
| F16 | non-blocking | test-gap | folded | Both reviewers converged independently on the same superior fix: table-driven positive/negative fixtures for the three regexes, which kills narrowing *and* loosening without ever crying wolf on growth | — | Folded into the F13/F15 fix. **This absorbs round 1's F2**, whose disposition changes from `new ticket` to `folded` — the batched follow-up loses that item |

**Adjudication of the rework's flagged interpretation.** The rework read decision 5's "must
reproduce these exact figures" as licensing floors, and said so rather than hiding it. The
independent adjudicator ruled **deviation requiring sign-off**, and the reasoning holds: floors
are a strictly weaker predicate — every state exact equality accepts, floors accept, plus an
unbounded set more — and the tell was that the rework needed two paragraphs of comment to
justify not doing what the decision said. The cry-wolf argument against exact equality is
*correct*, but it is an argument for **amending decision 5**, not for quietly satisfying it. The
rework's load-bearing premise, "documentation only grows," is asserted rather than true, and F13
shows the growth it counts on is exactly what defeats it.

**Re-verification note (why F15 was nearly dropped).** As first reported, F15 did not reproduce:
the obvious anchored regex loosening leaves the TOML prose unmatched and the dead xref still
caught. It reproduces only with an *unanchored* `[[id]]` alternative — which is the form someone
adding legacy-anchor support would actually write. Recorded only after that reproduction; the
first, failed attempt is noted so a later reader does not "re-verify" it the shallow way and
wrongly strike the row.

**Disposition summary (round 2):** 4 findings — 3 blocking (F13, F14, F15 → `5-rework/`); 1
non-blocking (F16, folded, and it demotes round 1's F2 from `new ticket` to `folded`).

`cost: estimated S, actual M` — two rework cycles and four blocking findings past the estimate;
the guard needed to be a guard against itself, which the S estimate did not price. Provisional
until the concluding review.

**Next rework must not proceed unasked.** The fix requires *amending confirmed decision 5* —
replacing its count-reproduction requirement with structural pattern invariants. Round 2
established that reinterpreting that decision unilaterally was the error; doing it again to fix
the consequences would repeat it. The amendment, and a dated `plan amended inline` History line,
need user sign-off before the rework starts.

### Rework pass 2 — 2026-08-22 — F13, F14, F15 fixed

Scope was the three blocking findings. Fixed in `ef567a1`, after user sign-off on the decision-5
amendment (recorded in the plan and in History).

The count floors are gone. What replaced them:

- **`assertEveryXrefSiteScanned`** — every literal `<<` in the book's prose must be consumed by
  `docsXrefTargetRe`. This is the F13 fix: it holds at any book size, so growth cannot restore
  the slack that let a narrowed pattern hide. Literal blocks (`----`, `....`, `++++`) and inline
  code spans are exempt, because a `<<` there is sample text — without that, the first shell
  heredoc in a code sample would make the guard cry wolf, which is how the previous design was
  argued into existence in the first place.
- **`TestDocsScannerPatternsMatchWhatTheyClaim`** — positive and negative fixtures per pattern.
  This is the F15 fix: the backticked `[[project]]` TOML prose is a negative row, so an anchor
  pattern loosened to accept the legacy spelling fails immediately instead of silently gaining a
  phantom anchor. It also subsumes round 1's **F2** (its disposition moved to `folded`).
- **A third fixture page** using the bare `xref:sibling.adoc[text]` spelling — the F14 fix, so
  both inter-document spellings are proven-caught rather than only the `#` one.

**Verification — seven sabotages, each reverted:**

| sabotage | floors (round 2) | now |
|---|---|---|
| drop the `<<id,text>>` spelling | caught | **caught** |
| **growth, then drop `<<id,text>>`** (F13) | **passed** | **caught** — "4 cross-reference site(s) that docsXrefTargetRe did not consume" |
| **loosen anchors to unanchored `[[id]]`** (F15) | **passed** | **caught** — "pattern loosened: … should not match but does" |
| **narrow interdoc to the `#` spelling** (F14) | **passed** | **caught** |
| mis-walked include tree | caught | **caught** |
| narrow anchors to `[A-Za-z]+` | caught | **caught** |
| ordinary docs growth, alone | passed | **passes** — no false alarm |
| literal block containing `<<EOF` | n/a | **passes** — exemption works |

Acceptance test re-run verbatim: `just build` ✓ · `go test . -run '^TestDocs' -v` ✓ (6 tests
including 3 subtests) · `just docs-check` ✓ · `just test` ✓ (0 failures) · `just lint` ✓ ·
`gofmt -l .` clean. Original-bug regression check re-run: injecting `<<no-such-anchor-xyz>>` still
fails naming file, line and dangling target.

### Round 3 — scoped re-review — 2026-08-22 — verdict: REWORK (2 blocking), then fixed

Reviewer independence: **delegated** (the orchestrating reviewer had authored all three rounds),
briefed to attack the *new* invariant specifically, on the observation that rounds 1 and 2 both
found the fix rather than the original code. All findings re-verified by hand.

**F13, F14, F15 confirmed fixed** by independent sabotage, and scope was clean (`ef567a1` touched
only `docs_xref_test.go`). Two new blocking findings in rework 2's own code:

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| N1 | blocking | test-gap | — | The invariant guarded the **pattern**, not the **scanner**: it re-ran the regex itself, so it could only ever prove the pattern still matched. F1's shape, one level down | Verified: making `scanBook` take only the first match per line passes all six tests, and a real dead reference planted second on a live line (`lifecycle.adoc:51`) goes unreported. Control: with the scanner intact, the same plant fails correctly | Compare reference-shaped sites against what `scanBook` returned (file:line:column), not against a fresh regex run |
| N2 | blocking | spec-unclear | — | The failure message prescribed a fix that breaks the build: "put it in a literal block (----) or an inline code span, which are both exempt" — `scanBook` honoured neither, so a contributor following the advice trades a coverage failure for an unresolved failure with no exit. Introduced by rework 2 | Verified both: a backticked `` `<<anchor-id>>` `` example and the same line inside `----` each fail `TestDocsXrefsResolve` as unresolved | Move literal-block awareness into `scanBook` so the advice is true, or drop the sentence |
| N3 | non-blocking | test-gap | folded | The exemption helpers had no fixtures, though amended decision 5 asks for them per pattern | Broadening `docsLiteralBlockDelim` to any `--` prefix guts the invariant with the suite still green | Folded into the N1/N2 fix — they became load-bearing once `scanBook` depended on them |
| N4 | non-blocking | design | folded | Delimiter model was a kind-blind toggle, so a `----` nested in a `....` block closes it and silently disables checking for the rest of the file | Verified with a control: the tail reference is reported without the nested block, not reported with it | Folded — same reason as N3 |
| N5 | non-blocking | design | new ticket | More inter-document spellings uncovered: extensionless `xref:cli-reference#id[]` and `link:file.adoc#id[]` | both match nothing today | Extends the batched follow-up (joins F3–F6) |
| N6 | non-blocking | design | new ticket | `bookFiles` follows an `include::` shown inside a sample block | pre-existing; verified | Joins the batched follow-up |

**Disposition summary (round 3):** 6 findings — 2 blocking (N1, N2, fixed below); 4 non-blocking:
2 folded into that fix (N3, N4), 2 to the batched follow-up (N5, N6).

#### Rework pass 3 — N1, N2 fixed (`a3e5c03`)

- **N1** — `assertEveryXrefSiteScanned` now compares against `scanBook`'s actual output by
  file:line:column. Losing an occurrence is caught whether the pattern or the scanner is at
  fault; `docsXrefOccurrence` gained a `col` field to make that possible.
- **N2** — literal-block awareness moved into `scanBook`, behind a shared `docsProseLines`
  helper used by both checks, so they agree by construction rather than coincidence. This is
  also the semantically correct reading: asciidoctor does not resolve a cross-reference inside
  a listing block either. Confirmed no anchor or reference sits inside a literal block today,
  so no current result changes.
- **The backtick-parity exemption is gone**, not tuned. It misfired on ordinary wrapped prose —
  `cli-reference.adoc:588` opens with a closing backtick, making the per-line count odd and
  silently exempting a real `<<cmd-doctor>>` from coverage. Reference-shaped sites are now
  defined by requiring a closing `>>`, which excludes heredocs and stream insertions by syntax
  instead of guessing from position. That live reference is covered for the first time.

**Verification — nine sabotages, each reverted:**

| sabotage | result |
|---|---|
| N1 `scanBook` first-match-per-line | **caught** |
| N3 delimiter broadened to any `--` | **caught** |
| N4 kind-blind delimiter toggle | **caught** |
| F13 drop `<<id,text>>` (incl. after growth) | **caught** |
| F15 loosen anchors to unanchored `[[id]]` | **caught** |
| F14 narrow interdoc to `#` only | **caught** |
| N2's advice: example inside `----` | **passes** — the advice now works |
| dead reference in prose | **caught** — no over-exemption |
| second dead reference on a live line | **caught** — N1's real consequence |

Acceptance test re-run verbatim: `just build` ✓ · `go test . -run '^TestDocs' -v` ✓ (13 passing
tests/subtests) · `just docs-check` ✓ · `just test` ✓ (0 failures) · `just lint` ✓ · `gofmt -l .`
clean · original-bug regression still caught.

#### Impact sweep (step 8)

No ticket carries T-067 in `depends-on:` (all matches were prose). Four READY tickets encode
"T-067 has not landed" as a live assumption and will need patching **once the branch merges**,
not now — the assumption is still true while the branch is unmerged:

- **T-111** — says T-067 is in `1-to-do/` (already stale) and instructs verifying anchors by hand
  "until T-067 lands"; its acceptance test repeats the caveat.
- **T-113** — "`docs-check` cannot catch that yet (T-067 …)".
- **T-071**, **T-114** — one-line prose references to the same gap.

Deferred to the concluding re-review so the patches describe what actually shipped.

## History

- 2026-08-05 — created (TO DO). source: pickle ticket new
- 2026-08-05 — filed from a docs session, not a review: registering the Agent Session Workflow page turned up that `snowball check` passes on an injected dead xref, and the manual's references had to be verified by an ad-hoc script instead
- 2026-08-05 — patched by T-057's pickup gate (finding F11, disposition `folded`): scope gained the
  CI half — `docs-check` is not wired into `.github/workflows/ci.yml` at all, so the docs gate
  never runs unattended
- 2026-08-06 — patched by the T-057 review (finding N3, disposition `folded`): scope gained the
  inter-document `xref:<file>.adoc#id[]` class, with two live examples the current gate passed
- 2026-08-15 — patched by **T-099's review impact sweep**. The "Where it lives" bullet framed the
  choice as two options — a Go test under `internal/` reaching the repo root via `payloadRoot()`,
  or a `just` recipe. T-099 shipped a third shape and an argument against one of the two, both of
  which transfer: `payload_lint_test.go` is a Go test **at the repo root in `package main`**,
  reading the embedded `payloadFS` directly rather than walking up to a directory that happens to
  sit beside it — the natural form when the thing under lint is already embedded. And T-099's
  refinement rejected the `just`-recipe option outright, on the grounds that this repo's own
  precedent (`lint-ci-surface`) lets an optional tool **degrade to a warning when missing**, which
  is the one failure mode a regression guard cannot have. Docs are not embedded, so
  `payloadFS`-style direct reading does not carry over to this ticket unchanged — the transferable
  parts are the root-`package main` placement and the anti-warning argument, not the FS. Nothing
  here is invalidated; the bullet just has a third candidate and one fewer live option to weigh
- 2026-08-20 — refined: settled both open design questions — a root-`package main` Go test
  (T-099 precedent), which answers the CI-wiring half for free via the existing `go test ./...`
  step (no new workflow job); `just docs-check` also gets a line invoking it, so the Outcome's
  own claim holds locally too. Prototyped the include-walk + anchor/xref-set-difference approach
  against the live tree (13 book files, 30 anchors, 86 `<<…>>` occurrences, 0 unresolved, 0
  `xref:` forms, 0 orphans today) to confirm the design before committing it to the plan. Found,
  as a side effect, that `docs/README.adoc`'s "Validating" section already claims the exact
  behaviour this ticket ships — folded into the Docs update task rather than filing separately.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — READY → IN DEVELOPMENT: picked up; applicability gate clean (fresh sub-agent;
  3 non-blocking citation-drift findings, all note-and-close — see Description addendum)
- 2026-08-22 — implemented: `docs_xref_test.go` (repo root, `package main`) — `bookFiles`
  follows `include::` recursively from `docs/user-manual.adoc`; `scanBook` collects explicit
  `[#id]` anchors, `<<target>>` occurrences and `xref:<file>.adoc…` occurrences with file:line;
  four tests — `TestDocsXrefsResolve`, `TestDocsNoInterDocumentXrefForm`,
  `TestDocsUserManualHasNoOrphanPages`, `TestDocsXrefCheckerCatchesTheFieldFindings` (synthetic
  fixture reproducing both proven-live bugs from the Description). `justfile`'s `docs-check`
  recipe now also runs `go test . -run '^TestDocs'`. `docs/README.adoc`'s "Validating" section
  rewritten to describe the real two-part check (was claiming behaviour the Description proved
  false); its "Layout" tree gained the previously-missing `agent-session-workflow.adoc` line.
  Hand-verified the regression bar: reintroduced the original field defect (`<<the-flow>>` →
  `<<no-such-anchor-xyz>>` in `agent-session-workflow.adoc`) and confirmed both
  `go test . -run '^TestDocs'` and `just docs-check` fail naming the file, line and dangling
  target; reverted before committing. `just build`, `just docs-check`, `just test`, `just lint`
  all green. No new `.github/workflows/ci.yml` job added — `go test ./...` already covers it
  (decision 1).
- 2026-08-22 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-22 — IN REVIEW → REWORK: review: 1 blocking (F1 decision-5 reference figures unasserted; core guard passes vacuously on a partial pattern narrowing); 11 non-blocking dispositioned (3 fixed inline, 5 -> batched follow-up, 3 noted)
- 2026-08-22 — REWORK → IN REVIEW: findings fixed
- 2026-08-22 — IN REVIEW → REWORK: scoped re-review: F1 resolved but inadequate; 3 blocking (F13 floors expire on next docs commit, F14 sibling test still vacuous, F15 floors blind to inflation)
- 2026-08-22 — plan amended inline: **confirmed decision 5**, with user sign-off. Its requirement
  that "the new tests must reproduce these exact figures as their reference point" is replaced by
  structural invariants over the scanner's patterns — every literal `<<` site in the book must be
  consumed by the xref pattern, and each pattern carries positive and negative fixtures. The
  refinement figures remain in the code as history, asserted by nothing. Why: the review cycle
  established that *neither* way of asserting a count works. Exact equality fails on every
  legitimate docs commit and trains contributors to bump the constant unread; a floor turns docs
  growth into slack (F13 — after one added page, a whole spelling could stop being matched and
  still clear the number) and is blind to inflation (F15 — a loosened anchor pattern swallowing
  prose makes dead references resolve). An invariant over the patterns catches narrowing and
  loosening alike at any book size and never fires on ordinary work. The first rework read the
  original wording as licensing floors and said so; an independent adjudicator ruled that a
  deviation needing sign-off, which is why this amendment is recorded here rather than assumed
- 2026-08-22 — REWORK → IN REVIEW: findings fixed
- 2026-08-22 — IN REVIEW → REWORK: scoped re-review: F13/F14/F15 fixed; 2 new blocking in rework 2 (N1 invariant guards the pattern not the scanner, N2 failure message prescribes a fix that breaks the build)
- 2026-08-22 — REWORK → IN REVIEW: findings fixed
