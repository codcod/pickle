---
id: T-085
title: make the per-ticket record aggregable: finding class, plan-amendment line, cost actual, provenance class
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-085 — make the per-ticket record aggregable: finding class, plan-amendment line, cost actual, provenance class

## Outcome

After this ships, the question "what does brine keep getting wrong?" is answered by counting, not by reading 53 tickets: each review records which kind of defect a finding was, each implementation records whether the plan it executed was wrong, and each ticket records what its estimate cost in reality.

## Description

The point of the three records above (see `## Outcome`): a recurring defect class can be
promoted to a `board audit` lint instead of being re-discovered by every reviewer.

### The problem, measured

brine captures more per-ticket evidence than most flows: dated History transitions, a findings
table with severity + disposition + evidence, and a merge line. **The corpus is large and it is
unaggregable.** Re-counted at refinement across the current `6-done/` — **53 tickets**, all 53
carrying a `## Review` heading, 46 of them carrying findings rows (T-001…T-007 predate the
protocol):

| available signal | state today |
|---|---|
| **347 dispositioned finding rows** (314 non-blocking, 33 blocking) | classifiable by *severity* and *disposition*, and by nothing else — there is no field saying what **kind** of defect it was |
| rework rate | **15 of 53** done tickets took a blocking finding (28%); never aggregated |
| review yield | a `Disposition summary` line exists in **40 of 53** — the one pre-aggregated number is present 75% of the time |
| plan-defect rate | the `plan amended inline` History line exists in **exactly one ticket** (T-049, which invented it unilaterally) — unchanged since filing |
| spawn rate | measured **once**, by hand, R=0.625 over 8 reviews — and it killed T-045 on a pre-registered threshold |
| grade rationale | the two `impact` recalibration passes wrote **11 justifications into `NOTES.md` and zero into tickets** (T-083's measurement) |

(The ticket was filed against a 36-ticket corpus with ≈165 findings and a 64% summary rate. Every
number has grown except the two that matter most: `plan amended inline` is *still* n=1, and the
kind-of-defect axis is *still* absent. The case is stronger, not weaker, at more than double the
data.)

The consequence is visible in this repo's own record: **T-045, T-063 and T-064 each proposed new
machinery while the data that would have settled them sat unread** (`NOTES.md:95-101`). The
failure was never capture. It was that nothing could be counted without an archaeology pass, so
counting only happened when someone already suspected the answer.

### The four items (one theme: the per-ticket record becomes countable)

Batched deliberately — §5's *"batching is mandatory, not encouraged"*. All four are the same
change in four places: **a field that already exists in prose becomes a field that can be
grouped by.**

1. **`class` column on the findings table — and the canonical skeleton it needs to be countable.**
   One word per row from a closed vocabulary (pinned at refinement, below). This is the item that
   carries the ticket: 347 findings is the largest structured dataset in the repo, and it is one
   column away from being diagnostic rather than merely countable. A class that recurs is a
   **lint specification** — the T-040 / T-067 conversion, where review labour becomes an audit
   check.

   **Refinement found the actual obstacle, and it is not the missing column.** The table's shape
   exists only as *prose*, stated twice — `review-protocol.md:141-144` and `TEMPLATE.md:141-145` —
   with **no literal table skeleton anywhere in the payload**. The corpus has drifted into **13
   distinct header variants** across 46 tickets; the dominant one accounts for 32 of them.
   Counting the existing 347 rows required an `awk` scan that detects a header containing both
   `severity` and `disposition` and then walks pipe-rows until a non-pipe line — archaeology, for
   exactly the reason this ticket exists. Adding a seventh column to prose would produce a
   fourteenth variant, not a dataset. So the column ships **with** a canonical, pasteable table
   skeleton in one place, and the second statement of the column list becomes a pointer to it.
   This is the precondition for item 1 doing anything at all, not scope creep.
2. **`plan amended inline` promoted from convention to rule.** It is the only direct measurement
   of READY-gate quality. T-049 used it twice, and *both* amendments retracted a **confirmed
   design decision** — decisions the READY gate had certified. If that rate is material, rules §4
   is mis-tuned.

   Refinement found it is weaker than "convention": the line is **specified nowhere** — not in
   `TEMPLATE.md`, not in `tickets-README.md`, not in `review-protocol.md`. It exists only as usage
   in `T-049:366,369`. n=1 is therefore fully explained by the flow never having asked for it, and
   the fix is three lines of prose. Cheapest of the four items, and the only one that measures the
   gate itself.
3. **`cost` actual vs estimate.** `impact` gets re-graded in periodic passes; `cost` is set once
   at refinement and never confronted with reality, which makes it unfalsifiable — the exact
   defect the recalibration passes exist to correct for `impact`.

   **Filed against a table that does not exist.** The ticket said "one row in the review's
   implementation-audit table"; there is no such table. Step 2 of the protocol
   (`review-protocol.md:71-82`) is three prose bullets recording *met / partially met / not met*,
   and `review-protocol.md` never mentions `cost` at all — the implementation-audit tables visible
   in `6-done/` were invented ad hoc by individual reviewers. So `cost` actual lands instead as a
   **one-liner beside the disposition summary** (`review-protocol.md:161-164`): the same
   grep-able locus as the flow's only other pre-aggregated number, and no new structure to drift.
4. **Provenance class on the `created … source:` line.** Field-use vs self-host vs review vs
   audit vs chat. This is the **weight on every other signal**: the two field-finding triages
   (2026-07-27) produced more real defects than any self-host pass, and an n=36 single-repo
   corpus is exactly what one would overfit to. Partially present in prose today, unstructured.

### The `class` vocabulary, derived rather than invented

The candidates filed with this ticket — `stale-xref`, `docs-gap`, `test-gap`, `plan-wrong`,
`naming`, `dead-code`, `other` — were tested at refinement against a ~55-row sample drawn from
five recent reviews (T-080, T-084, T-090, T-095, T-097). They do not survive contact:

- **No class for a plain correctness defect.** T-090 F1 (a byte-widened `unicode.IsSpace` scan
  emitting invalid UTF-8), T-095 N1–N3, T-084 F1 — the *blocking* findings, the ones that matter
  most — all fall through to `other`. The largest bucket would have been the useless one.
- **No class for self-contradictory or under-specified shipped prose.** T-084 F2 (a scope rule
  satisfying both its own branches) and F6 ("verb phrase", "adjacent" left undefined) are not
  `docs-gap` — the docs exist — and not `plan-wrong`.
- **`plan-wrong` silently conflates two different things**: a *confirmed design decision* that was
  false (T-084 F1, T-090 F2) versus *plan prose this branch made stale* (T-080 N4, N5, N8). Only
  the first is the READY-gate signal item 2 exists to take; merging them destroys the measurement.
- **`naming` and `dead-code` are invented, not observed** — roughly two rows in fifty-five.

The pinned vocabulary is therefore eight values: **`correctness`**, **`test-gap`**,
**`docs-gap`**, **`stale-xref`**, **`plan-wrong`** (reserved for a falsified *confirmed decision*;
stale plan prose is `stale-xref`), **`spec-unclear`**, **`design`** (asymmetry, narrowing, dead
code, perf — no behaviour change), **`other`**. Definitions and the reserved-meaning rule are
Task 1's business.

### Pre-registered criterion (the point of the ticket)

Written in at filing, per the T-045 precedent — *"the first decision in the project made by a
pre-registered criterion … worth copying"* (`NOTES.md:112-115`):

> **After 8 further reviews carrying the `class` column:** if any single class accounts for
> **≥25%** of non-blocking findings, that class is promoted to a `board audit` check or a payload
> rule. If the distribution is flat — **no class above 15%** — the column is removed and this
> direction is dropped.

The failure mode is therefore "delete one column", which is what makes the bet cheap enough to
take at all.

### Deliberately out of scope

- **Backfilling the existing 165 findings.** T-025 was dropped as *"lineage archaeology with no
  consumer"*; retro-classifying a corpus nobody has queried yet is the identical mistake.
  **Prospective only.**
- **A metrics command, a retro command, or a dashboard.** T-045's measurement cost one `for`
  loop. The substrate is **T-065** (below), after which every number above is a `jq` one-liner.
- **Anything touching ordering, ranking, scoring or gating.** T-045, T-063 and T-064 are all in
  `7-dropped/` for proposing exactly that; this ticket changes what a ticket *records*, never
  how the board sorts or what it admits.
- **A second prioritisation vocabulary.** The rejection recorded at `NOTES.md:729-741` stands.

### Couplings

Soft couplings (no `depends-on:`, no ordering enforced):

- **T-065** (JSON read projection) — **the substrate.** Its projection must expose parsed
  findings and History for any of this to be queryable without re-parsing markdown. That item was
  added and is live at `T-065:51-74` ("Open scope question: the `## Review` findings table"),
  including a middle option costing "counts and the closed-vocabulary columns only (id, severity,
  disposition, and the `class` column T-085 proposes)". Neither blocks the other: this ticket's
  fields are useful to a human reader with `grep` on day one, and T-065 is useful without them.
  Note that `T-065:55` and `:69-70` copied this ticket's original "≈165 … 36 done tickets" and
  "23 of 36" figures, which refinement has now superseded — corrected in the same pass.
- **T-083** (Outcome section) — closest neighbour and **not** a duplicate. T-083 structures the
  Description's *opening*, for a reader deciding whether to refine; this structures the *Review
  and History*, for a reader counting across tickets. Both add a per-ticket field and both
  contemplate a `board audit` warning for its absence, so they should be sequenced, not merged.
  T-083's "11 justifications in `NOTES.md`, 0 in tickets" measurement is item 4's evidence too.
- **T-081** (gate table as data, **shipped 2026-08-11**) — its unit is a `##` section **plus** an
  optional normalised `###` sub-heading stem inside it, each required present with a non-empty
  body (`flow.Requirement`, per-status `State.Requires`, evaluated by `ticket.GateViolations`).
  That is a strictly stronger substrate than the *"a `##` section (and its non-emptiness)"* this
  note originally recorded: a presence check for the `class` column or the `Disposition summary`
  line would hang off the same table, at either `Severity`. What T-081 does **not** model is a
  per-*line* requirement inside a section, which is exactly the shape both of those checks need —
  so this ticket either adds that row kind or keeps its own check.
  **Refinement's answer: neither — this ticket ships no mechanical check at all.**
  `flow.Requirement` is `{Section, Sub, Label, Hint, Severity}` (`internal/flow/flow.go:188-194`)
  and `ticket.GateViolations` (`internal/ticket/ticket.go:656-677`) has exactly two predicates,
  `SectionMissing` and `SubsectionMissing`; nothing expresses a requirement about a line, a table
  column, or any content inside a section. Adding that row kind is a T-081-sized change, and it
  contradicts the pre-registered criterion below, whose whole cheapness argument is that the
  failure mode is *deleting one column*. Audit enforcement is what the criterion decides **after**
  8 reviews, not what this ticket builds. Prose only, prospective only.
  Recorded as a coupling only: the 2026-08-04 precedent refuses to credit **prospective** demand
  when grading, and T-083 already declined this same bump on that basis. **The rule cuts both
  ways — do not re-grade T-081 for this.**
- **T-036** (the four dispositions) — this adds an orthogonal axis to the table T-036 ratified;
  it must not disturb the disposition vocabulary or its default.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                                   # `pickle` is the root-path child
git checkout main
git checkout -b feat/T-085-aggregable-ticket-record
```

WIP commits encouraged. **Root-path child** (`path = "."`): the Finish step tidies WIP into
atomic commits and keeps that history rather than squashing. Ticket and board bookkeeping stays
on `main`, never on this branch.

### Prerequisite gate (hard)

None. `depends-on:` is empty. T-081 (the gate table) and T-083 (the `## Outcome` row) are both
shipped, and this ticket deliberately builds on neither — it adds no audit check (decision 2).
T-065 is still in `1-to-do/` and is not required: `grep` is the day-one consumer.

### Confirmed design decisions (do not deviate without asking)

1. **Payload prose only. No Go code, no `board audit` check, no new command.** Nothing under
   `internal/` changes. If a task appears to need a code change, stop and ask — it means the
   scope slipped.
2. **No per-line gate row.** Do not extend `flow.Requirement` or `ticket.GateViolations`
   (`internal/flow/flow.go:188-194`, `internal/ticket/ticket.go:656-677`). Whether any of this
   earns mechanical enforcement is what the pre-registered criterion decides after 8 reviews.
3. **Prospective only. Do not backfill.** The existing 347 finding rows are not classified, not
   re-tabulated, and not touched. T-025 was dropped for exactly this.
4. **One canonical table skeleton, one source of truth.** The literal skeleton lives in
   `skill/resources/review-protocol.md` (the normative document). `TEMPLATE.md`'s restatement of
   the column list is replaced by a **pointer** to it. Two statements of one shape is what
   produced 13 header variants; do not leave two.
5. **The `class` vocabulary is closed and exactly these eight**: `correctness`, `test-gap`,
   `docs-gap`, `stale-xref`, `plan-wrong`, `spec-unclear`, `design`, `other`. Do not add, rename
   or re-order without asking — the pre-registered criterion's percentages are meaningless
   against a moving vocabulary.
6. **`plan-wrong` has a reserved meaning** and the prose must say so: a **confirmed design
   decision** that turned out false or unworkable. Plan prose merely made stale by the branch is
   `stale-xref`. This separation is the whole of item 2's measurement.
7. **Do not disturb the disposition vocabulary or its default** (rules §5, ratified by T-036).
   `class` is an orthogonal axis added beside `disposition`, never a replacement, and `noted`
   stays the default.
8. **Blocking findings get a `class` too.** They take no disposition (the cell stays `—`), but
   the kind-of-defect axis is most valuable precisely on the 33 blocking rows. Say so explicitly,
   because the existing prose's "a blocking finding is not dispositioned" invites the wrong
   generalisation.
9. **Self-modify policy** (`AGENTS.md`): never run `pickle install|upgrade|uninstall` against this
   repo from this branch. `.agents/skills/ticket-flow/` is a symlink to `skill/` — edit `skill/`
   and the installed skill follows. Any test install goes to a throwaway dir with the binary
   copied in.

### Tasks

#### Task 1 — the `class` column and the canonical skeleton

`skill/resources/review-protocol.md`, step 5 (`:140-164`):

- Replace the prose column list at `:141-144` with a **literal, pasteable markdown table
  skeleton** — header row plus separator plus one commented example row — carrying the columns:
  `id | severity | class | disposition | description | evidence | suggestion`. `class` sits
  beside `severity` because both describe the finding; `disposition` describes what was done
  about it.
- Add the closed vocabulary as a definition table, one row per value (decision 5), each with a
  one-line test the reviewer can apply:
  - `correctness` — ships wrong behaviour or wrong output
  - `test-gap` — coverage missing, deleted, or tautological (a test that cannot fail)
  - `docs-gap` — user-facing docs missing, wrong, or in the wrong place (includes `CHANGELOG.md`)
  - `stale-xref` — a reference this branch made false: line anchors, cross-references, comments
    or plan prose describing behaviour that changed
  - `plan-wrong` — **reserved**: a *confirmed design decision* was false or unworkable
    (decision 6)
  - `spec-unclear` — shipped prose that is self-contradictory, ambiguous, or under-specified for
    execution
  - `design` — asymmetry, narrowing, dead code, or performance, with no behaviour change
  - `other` — none of the above; if this exceeds ~10% of rows the vocabulary is wrong, say so
- State that a **blocking** finding carries a `class` and an em-dash `disposition` (decision 8).
- Cite `T-090 F1` (invalid UTF-8 from a byte-widened space scan) as the worked `correctness`
  example and `T-084 F2` (a scope rule satisfying both its own branches) as the `spec-unclear`
  one — both are in `6-done/` and both are why those two values exist.

`skill/resources/TEMPLATE.md` (`:141-145`): replace the restated column list with a pointer to
the skeleton above (decision 4). Keep the `## Review` section itself unchanged in shape.

`skill/resources/tickets-README.md` §5 (`:332-333`) says the disposition is "recorded in a
`disposition` column of the ticket's Review table". Add one clause naming the `class` column
alongside it, and point at the protocol for the vocabulary — **do not restate the eight values
here**, mirroring how §5 already refuses to restate the dispositions in the protocol.

#### Task 2 — `plan amended inline` from nowhere to a rule

The line is currently specified in no document (only used at `T-049:366,369`). Give it one home
and two pointers:

- `skill/resources/tickets-README.md` §1 (`:118-133`) — the section that already enumerates the
  History line forms. Define it as a dated, non-transition line:
  `YYYY-MM-DD — plan amended inline: <what changed and why>`. Say explicitly that it is **not** a
  transition line, so the over-long-line warning §1 already describes does not apply to it (that
  warning is scoped to transition and merge lines).
- `skill/resources/TEMPLATE.md` History comment block (`:149-160`) — add it to the worked
  examples.
- `skill/SKILL.md`, *Procedure: implement a ticket* step 3 — it already says an amendment "takes
  the inline disposition (edit the plan, record it in the ticket's History)". Make it name the
  line form now that one exists.
- State **when it is mandatory**: any edit to `## Implementation Plan` made after the ticket left
  `2-ready/`, whether at the applicability gate or mid-build. Note that when the amendment
  retracts a *confirmed design decision*, the matching review finding is classed `plan-wrong`
  (decision 6) — this is the join that makes item 2 and item 1 one measurement rather than two.

#### Task 3 — `cost` actual beside the disposition summary

`skill/resources/review-protocol.md` `:161-164`. There is no implementation-audit table to hold
this (see the Description), so add a second one-line closer directly under the disposition
summary:

```
cost: estimated <S|M|L|XL>, actual <S|M|L|XL>[ — <one clause, only when they differ>]
```

The reviewer is the right author: they have just re-run the acceptance test and read the branch.
Say that the estimate is copied verbatim from the ticket's `cost:` frontmatter and that the
frontmatter is **not** rewritten — the divergence is the datum, and overwriting the estimate
would erase it. Add the line to the step-5 checklist item at `:234`.

#### Task 4 — provenance class on the `created` line

The seed line at `TEMPLATE.md:162` already offers `<chat | review of T-xxx | audit | idea>` — an
unstructured mix of source and kind. Replace it with an explicit leading class token:

```
- YYYY-MM-DD — created (TO DO). source: <field-use|self-host|review|audit|chat>: <prose>
```

- `field-use` — found using pickle on another project
- `self-host` — found operating this repo's own flow
- `review` — spawned from a review finding (pairs with `spawned-by:`)
- `audit` — from `board audit` or a board-audit pass
- `chat` — from discussion, with no triggering incident

Mirror it in `tickets-README.md` §1 (`:123`, which names the `created (TO DO). source: …` form)
and in §3's `spawned-by:` bullet (`:244`), which already ties the two together. Say that the
class weights every other signal in this ticket: an n=53 single-repo corpus is exactly what one
would overfit to, and the two field-finding triages produced more real defects than any
self-host pass.

#### Task 5 — the pre-registered criterion, written where it will be found

The criterion in this ticket's Description is invisible to whoever reviews the 8th ticket after
this ships. Record it in `tickets/NOTES.md` with its filing date, the two thresholds (≥25% →
promote to a check; nothing above 15% → remove the column), and the exact `awk`/`grep` recipe for
counting `class` values over `6-done/`. `NOTES.md` is hand-written prose and outside the payload,
so this is the correct home — not the skill.

#### Task 6 — correct T-065's copied figures

`tickets/1-to-do/T-065-…md` `:55` and `:69-70` carry this ticket's superseded "≈165 … 36 done
tickets" and "23 of 36". Update to 347 / 53 / 40-of-53. Bookkeeping only — commit on `main`, not
on this branch.

### Acceptance test

No Go code changes, so the build must be untouched — that is itself the check:

```sh
just build && just test && just lint && just docs-check
git diff --name-only main...HEAD | grep -E '^(internal|cmd)/|\.go$'   # expect: NO output
```

Then the payload's own consistency:

```sh
# 1. exactly one statement of the findings-table column list, and it is the skeleton
rg -n 'id.*severity.*disposition.*description.*evidence' skill/
#    expect: one hit, in review-protocol.md, inside a literal table

# 2. all eight class values defined in exactly one place
rg -c 'correctness|spec-unclear|stale-xref' skill/resources/review-protocol.md   # expect: >0
rg -l 'spec-unclear' skill/                                                      # expect: review-protocol.md only

# 3. the plan-amendment line now has a home
rg -n 'plan amended inline' skill/
#    expect: hits in tickets-README.md, TEMPLATE.md and SKILL.md (was: zero)

# 4. the provenance classes reach both documents
rg -n 'field-use' skill/resources/TEMPLATE.md skill/resources/tickets-README.md  # expect: both

# 5. the criterion is recorded outside the ticket
rg -n 'class' tickets/NOTES.md   # expect: the threshold entry
```

End-to-end, in a **throwaway directory with the binary copied in** (`AGENTS.md` self-modify
policy — never the in-repo path):

```sh
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
./pk install --project demo . >/dev/null
./pk ticket new "probe" --project demo
./pk board audit          # expect: clean — no new gate row was added (decision 2)
grep -n 'source:' tickets/1-to-do/T-001-probe.md   # expect: `source: pickle ticket new` (unchanged — see note)
```

The `board audit` line is the load-bearing one: a clean audit on a fresh ticket proves this
ticket added no mechanical check, which decision 2 requires.

**Note (plan amended inline, applicability-gate finding):** `internal/ticket/ticket.go`'s
`Scaffold()` hardcodes `source: pickle ticket new` and is not driven by `TEMPLATE.md` — decision
1 forbids touching it. The five-class `source:` form is authored by hand, overwriting that
placeholder, during the *"make it a ticket"* procedure's step 5 ("record where the ticket came
from ... in prose"), exactly like the `## Outcome` TODO placeholder T-083 introduced. The E2E
check therefore verifies the CLI-scaffolded placeholder is unchanged (proving no Go code was
touched) rather than asserting the five-class form appears pre-authoring; checks 1–5 above
already confirm the five-class form is documented everywhere an author would read it.

### Docs update (mandatory when user-facing)

The payload **is** the user-facing surface here, and Tasks 1–4 are the docs change in full:
`skill/resources/review-protocol.md`, `skill/resources/TEMPLATE.md`,
`skill/resources/tickets-README.md`, `skill/SKILL.md`.

Also check, and update only if they state the findings-table shape or the History line forms:
`docs/user-manual/` (the earlier sweep found **no** column list anywhere under `docs/`, so most
likely nothing to do — confirm rather than assume) and `README.md`. Add a `CHANGELOG.md`
`[Unreleased]` entry: this changes what every installed project's reviews record, and T-083 — the
nearest comparable payload change — set that precedent. `just docs-check` must pass.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean; the
   `git diff` guard above prints nothing.
2. Docs updated per Tasks 1–4 plus the `CHANGELOG.md` entry.
3. Write a summary: files touched, the eight pinned `class` values, and confirmation that no
   backfill and no audit check were added.
4. Suggested Conventional Commit message:

   ```
   feat(skill): make the per-ticket record countable (T-085)

   Add a closed-vocabulary `class` column to the review findings table, behind a
   canonical table skeleton that replaces two drifting prose statements of the
   column list. Specify the `plan amended inline` History line, which no document
   had ever asked for; record cost actual-vs-estimate beside the disposition
   summary; and give the `created ... source:` line a provenance class. Prose
   only -- no audit check, no backfill.
   ```

5. **Tidy up before presenting** — root-path child, so interactive-rebase the WIP commits into a
   small number of atomic commits (suggested split: the skeleton + `class` vocabulary; the
   plan-amendment line; the cost and provenance lines; `NOTES.md` + `CHANGELOG.md`) and keep that
   history rather than squashing.
6. Commit locally on the branch. Do **not** push or open an MR without explicit approval. Present
   the commit messages; after approval, verify
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing, then push and open the MR. Merging is the human's. Task 6's T-065 edit is bookkeeping
   and is committed separately, on `main`.

## Review

Reviewed 2026-08-13 on `feat/T-085-aggregable-ticket-record` (HEAD `cc6fc43`, 5 commits).
**Verdict: REWORK** — 2 blocking, 5 non-blocking.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.md` files (step 4b) — ran; 16 suggestions,
      **none** on prose this branch authored, all discarded as out of scope for this ticket
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message & MR attributes presented for approval (step 9) — deferred to the
      scoped re-review

### Implementation audit (step 2)

| item | result | evidence |
|---|---|---|
| Tasks 1–4 (payload prose) | **met** | `skill/resources/review-protocol.md:141-204`, `TEMPLATE.md:141-171`, `tickets-README.md:122-150,262-264,349-355`, `SKILL.md:214-218` |
| Task 5 (criterion in `NOTES.md`) | **met** | `tickets/NOTES.md:882-901` — both thresholds + the `awk` recipe, committed on `main` (`efb6b97`) |
| Task 6 (T-065 figures) | **met** | T-065 now reads 347 / 53 / 40-of-53 (`efb6b97`) |
| Acceptance: `just build && just test && just lint && just docs-check` | **met** | all green |
| Acceptance: no Go/code changes | **met** | `git diff --name-only main...HEAD \| grep -E '^(internal\|cmd)/\|\.go$'` → no output; diff is `CHANGELOG.md` + 4 payload files only |
| Acceptance checks 1–5 (payload consistency) | **met** | one column-list statement (`review-protocol.md:145`); `spec-unclear` only in `review-protocol.md`; `plan amended inline` in all three docs; `field-use` in both; criterion in `NOTES.md` |
| Acceptance: E2E in throwaway dir | **met** | `board audit` → 0 errors (sole warning is T-083's pre-existing `## Outcome` advisory on a fresh scaffold); `source:` placeholder unchanged, per the inline amendment |
| Decision 1 (prose only, no Go) | **met** | see the diff guard above |
| Decision 2 (no per-line gate row) | **met** | clean `board audit` on a fresh ticket; `flow.Requirement`/`GateViolations` untouched |
| Decision 3 (prospective only, no backfill) | **met** | no existing `## Review` table modified |
| Decision 4 (one canonical skeleton) | **met** | `TEMPLATE.md:141-147` is now a pointer; the literal skeleton exists once |
| Decisions 5–8 (vocabulary, `plan-wrong` reserved, disposition untouched, blocking carries a class) | **met** | `review-protocol.md:157-171` (vocabulary + worked examples), `:174-178` (blocking carries a class); `noted` still the default |
| Decision 9 (self-modify policy) | **met** | E2E ran in `$(mktemp -d)` with the binary copied in |
| Docs step — `CHANGELOG.md` `[Unreleased]` entry | **met** | `CHANGELOG.md:13-23` |
| Docs step — `docs/user-manual/` confirmed rather than assumed | **not met** | **F1** below |

One shipped claim was verified against the code rather than taken on trust: §1's assertion that
the over-long-line warning does not apply to a `plan amended inline` line (`tickets-README.md:142-150`) is **correct** —
`internal/audit/audit.go:209` scopes that check to `HistoryTransition`/`HistoryMerged` only, and a
non-transition line parses as `HistoryNote` (`internal/ticket/ticket.go:100-103`). Empirically
confirmed: `board audit` on `main` is clean at 97 tickets with this ticket's own 6-line amendment
line present.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | **blocking** | docs-gap | — | **The user manual was not updated, and the ticket's own docs step required it.** `tickets.adoc` enumerates the History line kinds — "every status transition (`OLD → NEW: reason`), the creation source, and the human's merge record" — which is now a three-of-four list: `plan amended inline` is a new, **mandatory** kind and is absent, and "the creation source" is now a classed form. `lifecycle.adoc` is the manual's canonical account of *what a review records* and describes severity + the four dispositions only: the `class` axis and the `cost: estimated …, actual …` closer are both missing. The plan's docs step conditioned an update on these files "stat[ing] the findings-table shape **or the History line forms**" — `tickets.adoc:94-95` states the History line forms, so this is the plan's own trigger firing, not a scope judgement. Protocol 4a.1: missing coverage of a user-facing change is blocking. | `docs/user-manual/concepts/tickets.adoc:92-95`; `docs/user-manual/concepts/lifecycle.adoc:78-105`. Neither file is in `git diff --name-only main...HEAD`. | Add `plan amended inline` to `tickets.adoc`'s History bullet and class the creation source; add `class` (with a pointer to the protocol for the vocabulary, not a restatement) and the `cost:` closer to `lifecycle.adoc`'s review section. Do **not** restate the eight values in `.adoc` — decision 4's one-source-of-truth rule applies to the manual too. |
| F2 | **blocking** | docs-gap | — | **The provenance class never reaches the authoring path, reproducing the exact n=1 failure this ticket diagnoses.** `pickle ticket new` scaffolds `source: pickle ticket new` from a hardcoded `Scaffold()` string that `TEMPLATE.md` does not drive, so an author never sees the updated seed line. The inline amendment resolved this for the *acceptance test* by declaring the five-class form "authored by hand during the *make it a ticket* procedure's step 5" — quoting that step verbatim — but **step 5 was never updated** and still says only "Record where the ticket came from: the `created … source: …` History line in prose". It names no vocabulary and never says the scaffolded placeholder must be replaced. Task 2 did precisely this for the sibling item (SKILL.md updated to name the line form); Task 4 did not. The ticket's own finding is that `plan amended inline` sat at n=1 because *no document asked for it* — item 4 ships in that same state on the one path every author actually walks. | `skill/SKILL.md:154-158` (unchanged by this branch); `internal/ticket/ticket.go:743`; the amendment's own reasoning at `## History`, 2026-08-13 | One clause in *make it a ticket* step 5: name the five classes (or point at rules §1) and state that the `source: pickle ticket new` placeholder the CLI writes must be replaced with the classed form — mirroring how T-083's `## Outcome` placeholder is handled. Two lines; no Go change, so decision 1 holds. |
| N1 | non-blocking | stale-xref | fixed inline | `TEMPLATE.md` refers to the same file two ways inside one comment block: line 141 says `review-protocol.md`, line 143 says `resources/review-protocol.md §5`. Both files are siblings in `skill/resources/`, so the `resources/` prefix is wrong from `TEMPLATE.md`'s own location, and it contradicts the convention the preceding line sets. | `skill/resources/TEMPLATE.md:141` vs `:143` | Drop the `resources/` prefix at `:143`. |
| N2 | non-blocking | docs-gap | new ticket | **The shipped payload sends every installed project's reviewer to ticket ids that exist only in this repo.** `review-protocol.md:170-171` reads "Worked examples from `tickets/6-done/`: `T-090 F1` … `T-084 F2`" — an instruction to look them up in the reader's *own* `6-done/`, where those ids are either absent or somebody else's tickets. The existing precedent in the payload is materially different: `tickets-README.md:63-64` uses ids as pure *syntax* illustration and `:237,:428` as provenance tags — neither tells the reader to go and read them. `NOTES.md:869-874` records this exact anti-pattern ("the foreign-workspace test as an anti-overfitting guard") as the thing to check before generalising a self-host observation into the payload. | `skill/resources/review-protocol.md:170-171`; cf. `skill/resources/tickets-README.md:63-64,237,428`; `tickets/NOTES.md:869-874` | Keep the two illustrations — they are the best part of the vocabulary section — but strip the lookup framing: state the defect shapes inline without `tickets/6-done/` and without bare ids, or attribute them explicitly to pickle's own corpus. |
| N3 | non-blocking | spec-unclear | new ticket | Same theme as N2: `review-protocol.md:143` justifies the skeleton with "prose-only drifted into 13 header variants across **the corpus** before this skeleton existed". "The corpus" is this repo's, is named nowhere, and is unavailable to the foreign reader the sentence addresses — the rule is sound but its stated warrant does not travel. | `skill/resources/review-protocol.md:143` | Generalise to the mechanism ("restating a table's columns in prose drifts; paste the skeleton instead") and drop the unattributed count, or attribute it. |
| N4 | non-blocking | spec-unclear | noted | **The new `cost:` closer has no defined behaviour on a rework verdict.** The protocol places it at step 5, which runs *before* step 6a routes a ticket to `5-rework/` — so the reviewer is asked for an "actual" cost at the one moment the work is demonstrably not finished. Nothing says whether the scoped re-review overwrites the line, appends a second one, or leaves the first. This review hit the ambiguity on its first use (see the closer below). | `skill/resources/review-protocol.md:195-204` vs `:206-211` (step 6a) | One clause: the `cost:` line is written on the **concluding** verdict (step 6b) and, after a rework round, records the total including rework. |
| N5 | non-blocking | docs-gap | fixed inline | `SKILL.md` was updated for Task 2 (the *implement* procedure names the new History line) but not for Task 1: its rules summary ("Findings — severity, then disposition") and *Procedure: validate a ticket* step 1 still describe the two-axis record. The asymmetry is mild because both defer to `review-protocol.md`, which a reviewer is told to follow — but SKILL.md is the entry point and now under-describes what a review produces. | `skill/SKILL.md:116-119` (rules summary), `:249` (validate step 1) | Add `class` to the summary bullet and to validate-step 1, as a name plus a pointer — do not restate the eight values (decision 4). |

**Disposition summary:** 7 findings — 2 blocking (F1, F2 → rework, no disposition); 5
non-blocking, all dispositioned: 2 *fixed inline* (N1, N5 — one-clause prose corrections in files
this branch authored), 2 *new ticket* (N2, N3 — batched as a single theme, payload genericity;
to be filed with `--spawned-by "T-085"` at the concluding review, not now), 1 *noted* (N4).
Spawn count 1 ticket / 5 non-blocking findings = R 0.2.

```
cost: estimated S, actual S — the four payload edits landed inside one session as scoped; the
rework rounds F1 and F2 add are small and prose-only, so S still holds. Provisional: per N4 this
line has no defined rework semantics, and it is restated on the concluding verdict.
```

### Impact sweep (step 8)

One ticket affected. **T-065** — `:56` still reads "columns `TEMPLATE.md` already fixes: id,
severity, disposition, description, evidence, suggestion". This branch falsifies both halves:
`TEMPLATE.md` no longer fixes the columns (decision 4 made it a pointer, `TEMPLATE.md:141-147`),
and the column list is now seven, led by `class`. T-065's own middle option already anticipates
the extra column, so the assumption is *strengthened*, not invalidated — only its wording is
stale. **Patch deferred to the concluding re-review**, since it should not land while the branch
that makes it true is in rework. (Task 6's figure correction, the other half of T-065's exposure,
is already applied on `main` in `efb6b97`.)

T-074 (`2-ready/`) was checked explicitly because it counts payload occurrences of `ticket-flow`
for its rename sweep: unchanged at 3 lines under `skill/` on both `main` and the branch — no
impact. T-070 (`HistoryEntries`) is unaffected: a `plan amended inline:` line already parses as
`HistoryNote` with no code change. No other `1-to-do/` or `2-ready/` ticket references T-085 or
depends on the review-table shape.

### Notes on what was *not* found

- The eight-value vocabulary survived its first real use: all 7 findings classed without
  reaching for `other` (0%, against the ~10% alarm threshold the protocol sets), and the
  `plan-wrong` reservation held — F2 concerns an amendment's unimplemented consequence, not a
  falsified confirmed decision, so it is `docs-gap`. The `docs-gap` / `spec-unclear` split did
  the work the refinement sample predicted it would.
- Both cited worked examples check out against the corpus: `T-090 F1` is the byte-widened
  `unicode.IsSpace` scan emitting invalid UTF-8, `T-084 F2` is the scope rule satisfying both its
  own branches. The *content* of N2 is accurate; only its framing is the problem.
- No finding under decisions 1, 2, 3, 5, 6, 7 or 8 — the scope discipline this ticket set for
  itself was kept exactly.

## History

- 2026-08-07 — created (TO DO). source: chat — "how can pickle learn from itself" exploration;
  scoped down from a five-part proposal to the four capture items after the retrieval half was
  split out and the mechanism half was noted-not-filed (`NOTES.md`, 2026-08-07)
- 2026-08-11 — patched by T-081's review impact sweep (step 8): T-081 shipped, so the soft-coupling note now records the substrate as built — a `##` section plus a normalised `###` stem, per-status data at either severity — instead of the weaker unit assumed at filing. The gap this ticket must still close is named: T-081 models no per-*line* requirement, which is what a `class` column or `Disposition summary` presence check needs. Assumption strengthened, not invalidated; nothing re-graded, per this ticket's own standing instruction
- 2026-08-13 — refined. Re-measured the corpus, which had more than doubled since filing: 53 done
  tickets (was 36), 347 finding rows (was ≈165), summary line in 40 of 53 / 75% (was 23 of 36 /
  64%), rework 15 of 53 (was 9 of 36) — while `plan amended inline` is still n=1, so the case
  strengthened on every axis that matters. Three filing premises corrected: (a) the obstacle to
  counting is **header drift**, not the missing column — the table shape is prose in two places
  with no skeleton anywhere, and the corpus holds 13 header variants, so the canonical skeleton
  became part of item 1; (b) item 3 was filed against an "implementation-audit table" that does
  not exist (step 2 is three prose bullets and never mentions `cost`), so cost-actual moves
  beside the disposition summary; (c) `plan amended inline` is specified in **no** document, not
  merely under-used, which fully explains n=1. The candidate `class` vocabulary was tested
  against a ~55-row sample from T-080/T-084/T-090/T-095/T-097 and failed — it had no value for a
  plain correctness defect, so every blocking finding would have landed in `other` — and was
  replaced by eight derived values. Decided prose-only: no per-line gate row, since
  `flow.Requirement` models none and building one contradicts the criterion's own
  cheapness argument. Cost collapsed S-M → **S**; impact/complexity unchanged
- 2026-08-13 — TO DO → READY: plan complete: prose-only, 8-value class vocabulary, cost collapsed to S
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
- 2026-08-13 — plan amended inline: acceptance test's E2E check corrected — it asserted
  `pickle ticket new`'s raw output would already show the five-class `source:` seed line, which
  requires editing `internal/ticket/ticket.go`'s hardcoded `Scaffold()` string and so contradicts
  decision 1 (prose only, no Go code). The five-class form is authored by hand during the *"make
  it a ticket"* procedure's step 5, mirroring how `## Outcome`'s placeholder (T-083) is filled in
  post-scaffold, not emitted by the CLI. The E2E check now asserts the placeholder is unchanged
  instead; checks 1–5 already confirm the five-class form is documented
- 2026-08-13 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-13 — IN REVIEW → REWORK: review: 2 blocking (F1 user-manual coverage, F2 provenance class absent from the authoring procedure); 5 non-blocking dispositioned
