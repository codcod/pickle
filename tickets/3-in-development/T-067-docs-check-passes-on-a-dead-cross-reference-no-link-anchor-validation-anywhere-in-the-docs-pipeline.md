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
   orphans — the new tests must reproduce these exact figures as their reference point (a
   material drift is a signal the parser mis-walked the tree, not that the docs regressed).

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

<!-- empty until IN REVIEW -->

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
- 2026-08-22 — READY → IN DEVELOPMENT: picked up
