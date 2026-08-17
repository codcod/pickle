---
id: T-106
title: specify the confirmed-decision statement shape and document the <ID> decision <N> citation convention
project: pickle
depends-on: []
spawned-by: [T-105]
impact: low-medium
complexity: low
cost: S
---

# T-106 — specify the confirmed-decision statement shape and document the <ID> decision <N> citation convention

## Outcome

A ticket author writing confirmed design decisions is told the shape to write them in, and anyone
citing a decision from another ticket is told the form to cite it in. Both conventions already
exist and are already relied on across projects; neither is written down anywhere, so today they
are learned only by imitation.

## Description

Two conventions carry real weight in the brine flow and are documented in none of the three payload
resources (`tickets-README.md`, `TEMPLATE.md`, `review-protocol.md` — verified: zero matches):

1. **The decision shape.** A confirmed design decision is written as a numbered item whose leading
   **bold run is the decision statement** and whose remainder is the rationale:
   `1. **The hook performs no network I/O.** No `git fetch`. A stale remote-tracking ref can only …`
2. **The citation form.** A decision is cited from another ticket as `<TICKET-ID> decision <N>` —
   for example, one contract in this repo is inherited across three tickets, each citing the last
   by that form. `review-protocol.md` §5 already makes "contradicts a locked decision" a *blocking*
   severity, so citing a decision precisely is load-bearing at review time, yet the form is
   nowhere specified.

**These are emergent conventions to be codified, not new rules to impose.** Measured at filing
(2026-08-16; figures will drift, re-measure rather than trusting them): the statement shape holds
for **367 of 397 decisions (92%)** in this repo and — the load-bearing datum — **120 of 138 (86%)**
in an unrelated three-child brine workspace that has never seen this ticket. Re-measured at
refinement the same day: **433 of 449 (96%)** here, so the shape is strengthening as the corpus
grows, not eroding. A convention that reached ~9 in 10 unprompted, in two corpora independently, is
a convention the payload should state rather than one the payload would be inventing.

**Foreign-workspace test applies.** This edits `TEMPLATE.md`, which ships to projects that are not
pickle, so per `AGENTS.md` the wording must not cite this repo's ticket ids as things to look up,
must not quote counts from a corpus the reader does not have, and must not phrase paths relative
to this source tree. The reference workspace above is evidence *for the ticket*, not text to ship:
the shipped sentence should teach the shape by example, with a syntactic filler id.

Also note the citation form must be written **prefix-agnostically** (`<ID> decision <N>`, not
`T-NNN decision N`): children set their own `ticket_prefix`, and the reference workspace uses
`RICK-` and `SNOW-` alongside a `T-` child.

### Decisions taken at refinement (were open at filing)

- **`board audit` gains no row. This ticket is documentation only.** The two reasons the ticket
  filed still hold — backfilling is refused by precedent (T-025, archaeology with no consumer), so
  enforcement could never reach 100%, and an audit row is permanent surface for a convention that
  already sustains itself unaided and is *rising*. A third reason settled it: `flow.Requirement`
  (`internal/flow/brine.go`) can express only *"this section or `### ` sub-heading is present and
  non-empty"*. "Every numbered item under this sub-heading opens with a bold run" is not
  expressible in that type, so the row would mean either widening `Requirement` for one consumer
  or hand-rolling a check back inside `internal/audit` — precisely the arrangement T-081 pulled
  out. The cost is structural, not cosmetic, and the ticket already named "no row" a legitimate
  outcome.
- **One home, two pointers.** `tickets-README.md` §7 (ticket structure) documents both conventions
  in full. `TEMPLATE.md` teaches the shape *by example* at the point of use and points at §7.
  `review-protocol.md` §5 gets a one-clause pointer where it already makes contradicting a locked
  decision blocking — the one place a reviewer needs the citation form in hand.
- **pickle's own user manual gets one sentence too.** The payload is not the only reader-facing
  surface: `docs/user-manual/concepts/lifecycle.adoc` § *The READY gate* already enumerates
  "Confirmed design decisions" as gate item 3 and is where a manual reader meets the section.

### Scope fence

- **No audit row, and no other mechanical enforcement** — settled above, not deferred.
- **No retrofit** of the non-conforming decisions in either corpus (T-025 precedent). Everything
  this ticket ships is prospective.
- **No locked-vs-ticket-local marking.** Parked with a pre-registered trigger in `NOTES.md` §
  *"ADR exploration (2026-08-15) — explored, nothing filed; the convention already works"*.
- **No command.** T-105 is the query surface and does not depend on this ticket: it reads the
  corpus as it stands and reports non-conforming items rather than requiring conformance. The two
  are independently schedulable in either order.
- **No `payload_version` bump and no `pickle upgrade` run.** This ticket edits the payload from a
  feature branch, which `AGENTS.md`'s self-modify policy forbids resolving with `upgrade`;
  re-stamping is the human's post-merge action.

### Soft couplings (no hard `depends-on`)

T-105 (the query command, spawned this split), T-098/T-099 (the payload's foreign-workspace rules
and the lint that enforces the mechanical part of them — the new wording must pass
`payload_lint_test.go`).

### Grading rationale

`impact: low-medium` — higher than T-105's `low` because this closes a *present* documentation gap
in a convention the review protocol already depends on, rather than serving prospective demand.
`complexity: low`, `cost: S`, both re-confirmed at refinement now that the audit row is ruled out:
what ships is prose in three payload files plus one manual sentence, a changelog entry, and a
payload-lint/docs-check pass. No Go behaviour changes.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-106-decision-shape-and-citation
```

Local WIP commits are encouraged. Because this is a root-path child, tidy them into atomic commits
before presenting (rules §0) and default to keeping that history rather than squashing. Do **not**
push or open a merge request without explicit user approval. Ticket and board bookkeeping is
committed on `main`, never on this branch.

**`.agents/skills/brine/` is a symlink to `skill/` in this repository — edit `skill/` and the
installed skill follows.** Do not run `pickle install`, `pickle upgrade` or `pickle uninstall`
against this repository from this branch (`AGENTS.md`, self-modify policy), and do not touch
`payload_version` in `pickle.toml`: re-stamping it is the human's post-merge action.

### 1. Prerequisite gate (hard)

None. `depends-on:` is empty. T-105 is a soft coupling in the other direction — it must keep
working against non-conforming items whether or not this ticket has shipped, so neither ticket
waits for the other.

### 2. Confirmed design decisions (do not deviate without asking)

1. **Documentation only. No audit row, no lint rule, no Go behaviour change.** If a task starts
   looking like enforcement, stop and ask — the reasoning against it is in the Description and it
   was a deliberate refinement outcome, not an oversight.
2. **`tickets-README.md` §7 is the single home; the other two payload files point at it.** Do not
   restate the full convention in `TEMPLATE.md` or `review-protocol.md` — the payload's own rule
   is that a thing is stated once and referenced (this is why `resources/` is called the single
   source of truth).
3. **Every shipped example uses a metasyntactic id** (`T-NNN`, `<TICKET-ID>`), never a real one.
   The citation form itself is documented **prefix-agnostically** as `<TICKET-ID> decision <N>`,
   with the note that children set their own id prefix.
4. **Every shipped decision example is self-contained** — no `§` cross-reference and no path
   inside the example text, so it reads correctly to someone who has never seen this repository.
   The foreign-workspace test (`AGENTS.md`) is the standard; `payload_lint_test.go` enforces its
   mechanical half and must stay green.
5. **State the ordinal rule explicitly: number in one unbroken list and never renumber**, because
   a citation elsewhere points at the ordinal as written. This is the half of the convention that
   is currently invisible and the half a tool (T-105) depends on.
6. **Follow each file's existing cross-reference dialect** rather than introducing a fourth:
   `tickets-README.md` refers to its own sections as bare `§N` and to its sibling as
   `review-protocol.md`; `TEMPLATE.md` writes `tickets/README.md §N`; `review-protocol.md` writes
   `rules §N`.
7. **`tickets-README.md` uses no `### ` sub-headings today.** Add the new material as prose (plus
   one fenced example) at the end of §7, keeping the file's flat shape.

### 3. Tasks

#### Task 1 — `skill/resources/tickets-README.md` §7 (the single home)

Append to `## 7. Ticket structure`, after the existing gate-table paragraph, roughly:

> **Confirmed design decisions, and how to cite them.** A confirmed design decision (§4.3) is one
> numbered item whose **leading bold run is the decision statement**; everything after it is the
> rationale.
>
> ```
> 1. **The check never writes to the ticket tree.** It is a read-only report, so a failure can be
>    retried without cleanup.
> ```
>
> The shape earns its keep twice: a reader skimming a plan gets the decision from the bold run
> alone, and a tool reading the ticket tree can lift the statement without guessing where it ends.
> Number the decisions in one unbroken list and **never renumber them** — an ordinal, once
> written, is an address.
>
> Cite a decision from another ticket as **`<TICKET-ID> decision <N>`** — for example
> `T-NNN decision 3`. Write the id exactly as its own child-project writes it (children set their
> own id prefix, so never assume a particular one), and `<N>` as the ordinal written in that
> ticket, never a re-count. The citation is prose, not a link: it costs nothing to write and one
> search to resolve. Precision matters most at review time, where contradicting a locked decision
> is a *blocking* severity (`review-protocol.md` §5).

Wording may be tightened; the five things it must carry are decisions 3–5 above, the bold-run
rule, and the pointer to the blocking severity.

#### Task 2 — `skill/resources/TEMPLATE.md` (teach by example, at the point of use)

Expand the placeholder body under `### Confirmed design decisions (do not deviate without asking)`
so it names the shape inline, keeping the file's `<…>` placeholder form and staying short — the
template is a skeleton, not a manual. It must: state the leading-bold-run shape with one inline
example; state the never-renumber rule and why (another ticket cites the ordinal as
`<TICKET-ID> decision <N>`); keep the existing "pull any project-wide decisions from the project's
own docs / `AGENTS.md`" sentence; and end with a `tickets/README.md §7` pointer.

Check afterwards that `TestFrontmatterKeysMatchTemplate` (`internal/audit/audit_test.go`) still
passes — it reads `skill/resources/TEMPLATE.md`. It only inspects frontmatter keys, so this edit
should be invisible to it; confirm rather than assume.

#### Task 3 — `skill/resources/review-protocol.md` §5 (one pointer, where it is needed)

On the **Blocking** bullet (currently review-protocol.md:176), extend "contradicts a locked
decision" with a parenthetical naming the citation form and its home — one clause, along the
lines of *(cite it as `<TICKET-ID> decision <N>` — rules §7)*. Nothing else in that file changes.

#### Task 4 — pickle's own user manual

In `docs/user-manual/concepts/lifecycle.adoc`, § *The READY gate*, item 3 ("*Confirmed design
decisions* the implementer must honour."), add one sentence: each is a numbered item whose leading
bold run is the decision statement, cited elsewhere as `<TICKET-ID> decision <N>`, and the ordinals
are never renumbered. Keep it to one sentence — the manual describes, the payload prescribes.

#### Task 5 — changelog

Add a `### Changed` (or `### Added`) entry under `## [Unreleased]` in `CHANGELOG.md`, ending
`(T-106)`, in the user-observable voice the existing entries use: the shipped skill now states the
confirmed-decision shape and the citation form that were previously learned only by imitation.

### 4. Acceptance test

Run verbatim from the repository root; all must be green.

```
just build
just test          # includes payload_lint_test.go — the foreign-workspace guard
just lint
just docs-check
```

Then prove the new wording actually ships into a foreign workspace, per `AGENTS.md`'s test-install
rule (throwaway directory, binary copied in and renamed):

```
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"
git init -q . && ./pickle-test install --project demo
grep -n "decision <N>" .agents/skills/brine/resources/tickets-README.md
grep -n "decision <N>" .agents/skills/brine/resources/TEMPLATE.md
grep -n "decision <N>" .agents/skills/brine/resources/review-protocol.md
./pickle-test ticket new "example" --project demo
./pickle-test board audit
cd - && rm -rf "$D"
```

Expected: `install` succeeds; each `grep` prints at least one line, proving the text reached the
installed payload rather than only the working copy; and `board audit` reports **0 errors** —
specifically, **no new warning class appears**, which is the mechanical check that this ticket
stayed documentation-only (decision 1).

*Amendment (inline, at pickup):* the plan as filed also expected a freshly scaffolded ticket
(`tickets/1-to-do/T-001-example.md`) to carry the new `### Confirmed design decisions` guidance.
That expectation does not hold regardless of this ticket's wording: `pickle ticket new` (see
`internal/ticket.Scaffold`) always writes a fixed, empty `## Implementation Plan` placeholder
(`<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->`) — the
`### ` sub-headings, including this one, are filled in only during refinement, from
`TEMPLATE.md`, which the three greps above already prove carries the new text. The `ticket new`
+ `board audit` calls are kept as a sanity check that scaffolding a ticket and auditing the
board still work with the new prose in place, not as a check that the new ticket contains it.

Finally, read the three shipped paragraphs against `AGENTS.md`'s foreign-workspace test by hand.
`payload_lint_test.go` matches four shapes; it cannot judge whether a sentence *means* something
only a pickle reader could resolve. Ask of each new sentence: would this help a project that is
not pickle?

### 5. Docs update (mandatory when user-facing)

Covered by Tasks 1–5, which are themselves the docs: three payload resources
(`skill/resources/tickets-README.md`, `skill/resources/TEMPLATE.md`,
`skill/resources/review-protocol.md`), one user-manual page
(`docs/user-manual/concepts/lifecycle.adoc`), and `CHANGELOG.md`. `just docs-check` must pass. No
`cli-reference.adoc` change — this ticket ships no CLI surface. No `payload_version` change (see
step 0).

### 6. Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean, and
   the throwaway-install check performed.
2. Docs updated per step 5, including the `CHANGELOG.md` entry.
3. Write a summary: files touched, the wording actually shipped, anything deferred.
4. Suggest a Conventional Commit message, e.g.:

   ```
   docs(skill): specify the confirmed-decision shape and citation form (T-106)

   <body — what and why>
   ```

5. **Tidy up before presenting** — `pickle` is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed/scoped commits (plausibly one
   `docs(skill):` commit and one `docs(manual):` commit).
6. Commit locally on the ticket branch. Do **not** push or open a merge request without user
   approval. Present the commit message; after approval, keep the tidied history (the root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the merge request. Merging is the human's.

## Review

**Verdict: REWORK — 1 blocking finding.** Reviewed 2026-08-17 against
`feat/T-106-decision-shape-and-citation` (3 commits, `d8bea5c..dbb1810`), base `main`, following
`resources/review-protocol.md`. No overarching or per-child review addendum is configured in
`pickle.toml`, so the generic protocol is the whole standard.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4) — **found the blocking finding below**
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files (step 4b) — run via the
      session's `docs_readability` reviewer over all five changed files; every suggestion
      targeted prose this branch never touched, so none were applicable/applied
- [x] Findings recorded in `## Review` with severity, class and disposition (step 5) — a
      concluding disposition summary + cost line will be added on the scoped re-review that
      follows the fix, per the done-ticket convention (T-018 precedent: no such line at REWORK)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6)
- [x] Other references checked — nothing in `1-to-do/`/`2-ready/` cites T-106 (step 7/8, impact
      sweep: no dependents)
- [x] Summary presented (step 9); nothing publish-gated to approve yet — the branch is unmerged
      and pending the rework fix

### Implementation audit

Tasks 1–5 are all present in the files and locations the plan named; `just build`, `just test`
(fresh `go clean -testcache`, all 19 packages including `payload_lint_test.go`), `just lint` and
`just docs-check` are all green. The throwaway foreign-workspace install (`AGENTS.md`'s test-install
rule, binary renamed `pickle-test`) shows the new `decision <N>` text reached the installed
payload in all three resource files, and `board audit` reports 0 errors with no new warning class
— confirming decision 1 (documentation only, no enforcement). Confirmed decisions 1, 2, 4, 5, 6, 7
are honoured (verified: no `internal/*.go` change on the branch; `tickets-README.md` is the only
file stating the convention in full; every shipped example is self-contained with no `§`
cross-reference inside it; the never-renumber rule is stated in both `tickets-README.md` and
`TEMPLATE.md`; each file's own cross-reference dialect is followed; `tickets-README.md` gained no
new `### ` sub-heading). **Confirmed decision 3 is where the blocking finding below lives** — it
chose the wrong placeholder token.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | plan-wrong | — | Confirmed decision 3 documents the citation form as `` `<TICKET-ID> decision <N>` ``, shipped identically across all five touched files. But `pickle board decisions` (T-105, done and merged) already shipped and locked the citable form as `` `<ID> decision <N>` `` — T-105's own confirmed decision 4 says a citation "says `<ID> decision 4` and must resolve to the same item", the shipped code/docs/tests all print and assert `<ID>`, and `cli-reference.adoc` (already in the tree) teaches authors `<ID> decision <N>`. This ticket's new prose now teaches a *different* placeholder spelling for the identical convention, in the same payload a reader consults for both, without ever discovering T-105 had already fixed it. | `tickets-README.md:461`, `TEMPLATE.md:90` (pre-fix), `review-protocol.md:177`, `lifecycle.adoc:42`, `CHANGELOG.md:30` — all `<TICKET-ID> decision <N>`; vs `tickets/6-done/T-105-…md` confirmed decision 4 and Description ("first column is literally `<ID> decision <N>`"); `internal/cli/board_decisions_test.go:291` (`TestBoardDecisionsCitationIsPasteable`, pins `"LIB-001 decision 1"`); `docs/user-manual/cli-reference.adoc:66,947,992` (all `<ID>`) | Change all five occurrences from `<TICKET-ID> decision <N>` to `<ID> decision <N>`, and re-run the throwaway-install acceptance check to confirm the corrected form ships. |
| F2 | non-blocking | design | fixed inline | `TEMPLATE.md`'s new sentence wrote `` `tickets/README.md` §7 `` — backtick around the path only, `§7` outside. The file's own dialect for this exact cross-reference is either plain prose (5 other instances: lines 11, 26, 44, 61, 129) or a single backtick span covering the whole `path §N` (3 instances: lines 148, 158, 163); the mixed form this branch introduced matched neither. | `skill/resources/TEMPLATE.md:90` (before fix) | Dropped the backtick around the path, matching the plain-prose majority (`dbb1810`). |

**Disposition (partial — concluding summary follows the scoped re-review):** 1 blocking (F1, to
fix), 1 non-blocking already fixed inline (F2).

## History

- 2026-08-16 — created (TO DO). source: review: split out of T-105 during an adversarial pass that
  found it bundled four separable changes. The format specification and any audit enforcement are
  independently schedulable from the query command and are neither a prerequisite nor a consequence
  of it, so they became this ticket rather than staying tasks in that plan (rules §3). Evidence that
  the conventions are emergent rather than imposed — 86% conformance in an unrelated workspace — was
  gathered in the same pass
- 2026-08-16 — refined: the audit row is ruled **out**, making this ticket documentation only. Two
  reasons were already filed (T-025 refuses backfill so enforcement could never be complete; a
  permanent audit surface for a self-sustaining convention); refinement added the deciding third —
  `flow.Requirement` can express only "section present and non-empty", so a bold-run shape check
  would mean widening that type for one consumer or hand-rolling a check back inside
  `internal/audit`, undoing T-081. Documentation homes settled: `tickets-README.md` §7 in full,
  with pointers from `TEMPLATE.md` (by example, at the point of use) and `review-protocol.md` §5
  (where a locked decision is already blocking), plus one sentence in the user manual's READY-gate
  list. Corpus re-measured: 433 of 449 (96%) conforming here, up from 367/397 (92%) at filing.
  Grades re-confirmed, not inherited
- 2026-08-16 — TO DO → READY: plan complete
- 2026-08-16 — impact note from T-105's review (no assumption invalidated, nothing in the plan
  changed): the query command shipped, so two of its measured facts are now shipped behaviour this
  ticket's §7 prose should stay consistent with. First, a statement's leading bold run **may
  soft-wrap across physical lines** before its closing `**` — common enough in the corpus that a
  first-line-only reader misreads 15 conforming items here — so the shape must not be documented as
  "one line". Second, an item with no leading bold run is reported as unstructured, carrying its raw
  first line, never a guessed statement. Re-measure with `pickle board decisions` rather than an
  `awk` from here on: the ordinal-prefixed heading form (`### 2. Confirmed design decisions`) is one
  the ad-hoc scripts silently skipped, which is why the corpus counts recorded above read 449/433
  where the command reports 470/454
- 2026-08-17 — READY → IN DEVELOPMENT: picked up
- 2026-08-17 — plan amended inline: the acceptance test's expectation that a freshly scaffolded
  ticket (`tickets/1-to-do/T-001-example.md`) would carry the new `### Confirmed design
  decisions` guidance does not hold — `pickle ticket new` always writes a fixed, empty
  `## Implementation Plan` placeholder regardless of `TEMPLATE.md`'s content, since sub-headings
  are filled in only at refinement. The step 4 acceptance test was corrected to check
  `TEMPLATE.md` directly (already covered by the existing grep) and keep `ticket new` +
  `board audit` only as a scaffolding/audit sanity check, not as a carrier check
- 2026-08-17 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-17 — IN REVIEW → REWORK: review found 1 blocking finding: <TICKET-ID> vs T-105's locked <ID> citation form
