---
id: T-036
title: ratify the four review-finding dispositions already in use; make note-and-close the default
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-036 — ratify the four review-finding dispositions already in use; make note-and-close the default

## Description

The shipped flow mints follow-up tickets faster than it retires them.
`resources/review-protocol.md` §5 requires that **every** non-blocking finding become a new
`1-to-do/` ticket ("Spawn a **new ticket** … for each"), and §0 forbids the alternative
("never inline drift"). The only escape hatch is line 116 — "trivial, purely-cosmetic spec
typos in the workflow scaffolding itself". So a reviewer's only legal move on any substantive
polish observation is to mint a ticket, and a competent reviewer can always find polish
observations.

This is a **product bug, not a repo-hygiene problem**. `pickle install` ships this protocol, so
every project that adopts the flow inherits the same arithmetic. This repo only hit it first
because it self-hosts. The fix therefore belongs in the embedded skill payload
(`skill/resources/`), not merely in this repo's board.

### Correction to the measurement this ticket was filed on

Refinement re-measured the claim above and **both halves of it were wrong**. Recorded here
because the original figure is what justified the ticket, and the corrected figure changes
what the ticket must fix.

1. **The stated source does not exist.** The ticket says "measured on this repo's own lineage
   (`spawned-by:`)". `spawned-by:` is `[]` on every ticket in that table — the field shipped in
   T-024 and was never backfilled, and the backfill ticket (**T-025**) was dropped in the
   2026-07-26 triage. The numbers were in fact read out of the `## Review` prose of `6-done/`,
   which is the only real record. The History `source:` lines are no better: T-031 … T-035 all
   say the default `source: pickle ticket new`.
2. **R ≈ 2.3 conflated three different spawn gates.** The table credited T-033/T-034/T-035 to
   T-030's *review*. Its review spawned exactly one ticket (T-038). The other three came from
   two gates the ticket never mentions — T-033 from **refinement** ("split the audit-side
   duplicate-key check out as T-033"), T-034 and T-035 from the **pickup applicability gate**
   ("adjacent findings filed as T-034 and T-035").

Re-measured over all 13 reviews in `6-done/`, counting **new tickets per review**:

| review | non-blocking findings | new tickets | review | non-blocking findings | new tickets |
|---|---|---|---|---|---|
| T-001 | 3 | 1 (T-012)              | T-008 | 2  | 1 (T-015) |
| T-002 | 2 | 0 (folded → T-012)     | T-011 | 0  | 0 |
| T-003 | 3 | 0 (folded → T-012)     | T-018 | 7  | 1 (T-021) |
| T-004 | 5 | 1 (T-013)              | T-024 | 13 | 2 (T-029, T-030) |
| T-005 | 1 | 0 (inline)             | T-029 | 7  | 2 (T-031, T-032) |
| T-006 | — | 3 (T-017, T-018, T-019)| T-030 | 6  | 1 (T-038) |
| T-007 | — | 1 (T-014)              |       |    | |

**R_review ≈ 1.0** (13 new tickets / 13 reviews), not 2.3 — and note the near-total absence of
correlation with finding count: T-018's seven findings produced one ticket, T-003's three
produced none. Reviewers have been batching all along.

The pressure is real, but it is spread across **three gates**:

- **§5, the review** — R ≈ 1.0, the only gate this ticket currently addresses.
- **The pickup applicability gate** — "adjacent work → new `1-to-do/` ticket(s)"
  (`SKILL.md` *implement a ticket* step 3; rules §8). No batching rule, no dispositions.
- **Refinement splits** — rules §3 names them as a lineage source; nothing bounds them.

T-030's single lifecycle spawned **four** tickets across all three (T-033 refining, T-034 +
T-035 at pickup, T-038 at review). Across the project, 32 of 43 tickets (T-012 onward) are
descendants of the 11 seed tickets, from 13 completions — **≈ 2.5 spawns per completed
ticket, all gates**. That is the number the original 2.3 was groping for, and it means
**fixing §5 alone addresses about 40% of the pressure.** Scope must cover all three gates.

### Practice has already overruled the protocol

Every disposition this ticket proposes to "add" is **already in use, undocumented**. Reviewers
routed findings four ways while §5 authorized one:

| disposition | authorized? | observed |
|---|---|---|
| new ticket, **batched** across findings | §5 says "for each" | T-001 (3→1), T-004 (5→1), T-018 (7→1), T-007 |
| **folded** into a ticket that already owns the ground | nowhere | T-002, T-003 (→T-012); T-006 (→T-013 items 6–9); T-024 (→T-015, T-019); T-029 (→T-027); T-030 (→T-019, T-027) |
| **fixed inline** | only "cosmetic spec typos" | 9 findings: T-005, T-008, T-024 (×4), T-030 (×3) |
| **noted, no action** | nowhere | T-011 (0 spawned, reasoned); T-024 N12 ("No action — deliberate") |

So this is not a design problem — it is a **ratification** problem. The T-030 review even
recorded the deviation as precedent to weigh here, and arrived empirically at the bar valve 1
needs: *prose or idiom in code authored on this same branch, no behaviour change.* The one
thing genuinely missing is that **"new ticket" is the stated default** when practice shows it
should be the exception.

### Scope: the two text valves, at all three gates

This ticket ships **one disposition vocabulary**, stated once and referenced by all three
spawn gates:

| disposition | when |
|---|---|
| **fixed inline** | prose or idiom in code authored on this same branch, no behaviour change |
| **folded** | a ticket already owns the ground; add an item to it |
| **new ticket** | would actually be scheduled; **batched** — one ticket per theme, not per finding |
| **noted** | recorded in the findings table, closed there — **the default** |

Both valves are one change, not two: valve 1 (inline) and valve 2 (note-and-close default) are
two rows of that table, and splitting them would mean writing §5 twice.

**Deferred to T-045** (split out at refinement, per user decision): the backlog cap and the
`user-visible:` axis. Both are CLI + schema rather than text, both have unresolved design
problems (an error-severity cap breaks `ticket move`/`board sync`/`install`/`upgrade`, which
run `audit` as a post-op self-check; a required key repeats the `spawned-by:` migration break),
and — decisively — both are **backstops for a leak this ticket plugs**. Their justification
should be re-measured after this lands, not assumed now.

Soft couplings (no hard `depends-on:`): **T-016** adds a Step 4b to the same protocol file and
**T-022** rewrites payload conditionals in the same tree — sequence to avoid edit collisions.
**T-044** owns the board-column mechanics any new grade axis would need (superseded T-039,
2026-07-26; columns become a render-side change). A future automated
spawn-rate warning needs `spawned-by:` to be populated, which is what dropped **T-025** would
have done — resurrect it before building the metric, not after.

**T-036 must land before T-044** (added at pickup, 2026-07-26). T-044 is now READY and second in
the queue, and its Task 7 edits **three of the four payload files this ticket edits**
(`tickets-README.md`, `SKILL.md`, `review-protocol.md`) — with a genuine adjacent-line collision
in the protocol's `## Review` checklist (this ticket rewrites line 176; T-044 rewrites line 178)
and a second one in `install.go`'s `markerBlock`, which amendment 1 below pulls into this
ticket's scope. This ticket is the smaller of the two and is top of READY; landing it first lets
T-044's "grep for board-edit steps" run against a stable tree. This is sequencing, not a
`depends-on:`.

## Implementation Plan

### Feature branch

`feat/T-036-review-disposition-valves`, cut from `main` in the `pickle` child-project (the repo
root, `.`). Local WIP commits encouraged; **no push and no merge request without explicit user
approval**.

> **This repo self-hosts the payload.** `.agents/skills/ticket-flow/` is a symlink to `skill/`,
> so the moment the branch is checked out, every agent in this repo is reading the new rules.
> No `pickle upgrade` is needed locally, and the ticket's own review must be conducted **under
> the new §5** — which makes it the first datapoint for T-045.

### Prerequisite gate

None. No `depends-on:`. Confirm before starting:

- `3-in-development/` is empty for `pickle` (WIP ≤ 1) — T-030 merged, so it should be.
- No other branch is mid-edit in `skill/resources/`. **T-016** (parked) would add a step 4b to
  `review-protocol.md` and **T-022** rewrites payload conditionals in the same tree; neither is
  in flight, but re-check at pickup.

### Confirmed decisions

1. **All three spawn gates are in scope** — §5 (review), the pickup applicability gate, and
   refinement splits. Fixing §5 alone addresses ~40% of the measured pressure.
2. **One vocabulary, stated once.** The normative disposition table lives in
   `tickets-README.md` §5. `review-protocol.md`, `SKILL.md` and `TEMPLATE.md` **reference it by
   name and never restate the list** — restating is precisely how payload files have drifted
   before (T-024 finding N9, and the whole of T-040). The protocol keeps the *procedure* (how
   to record, where to write, when to move); the rules keep the *vocabulary*.
3. **The four dispositions**, in the order a reviewer should consider them:

   | disposition | test |
   |---|---|
   | **fixed inline** | prose or idiom **in code authored on this same branch**, no behaviour change |
   | **folded** | an existing ticket already owns the ground — add an item to it, cite the id |
   | **new ticket** | would actually be scheduled; **batched by theme**, never one per finding |
   | **noted** | none of the above — recorded and closed. **This is the default.** |

4. **The inline bar is T-030's, verbatim** — "prose or idiom in code authored on this same
   branch, no behaviour change". It is field-tested on 9 findings across 4 reviews. Do not
   invent new wording, and do not add a line-count ceiling (uncalibrated).
5. **Inline fixes are never silent.** Every one is recorded in the findings table with
   disposition `fixed inline`. This is what keeps §0's "a review does not re-implement the
   ticket" true in spirit while relaxing its letter.
6. **`noted` is a real disposition, not a euphemism for ignoring.** The finding stays in the
   table with its evidence, permanently, and `7-dropped/`-style recoverability applies: a later
   reviewer can promote it by citing the table row.
7. **Promotion test for `new ticket`:** *"would this actually be scheduled?"* — not *"is this a
   real defect?"*. The 32 descendant tickets prove the second question always answers yes.
8. **Batching is mandatory, not encouraged.** One ticket per *theme* per gate. Reviewers already
   did this (T-018: 7 findings → 1 ticket); the protocol's "for each" was simply never obeyed.
9. **Valves 3–4 are out of scope** → **T-045**.
10. **No `README.md` or `PLAN.md` changes.** README documents commands and audit rules, not
    review dispositions (verified: no matches). `PLAN.md` is a historical record — the standing
    precedent from T-024 finding N12 is "no action — deliberate".

### Tasks

**1. `skill/resources/tickets-README.md` — the vocabulary (single source of truth).**

- §5 (lines 182–199): replace the two-bullet blocking/non-blocking split with the same
  blocking bullet plus a **non-blocking disposition table** (decision 3), the inline bar
  (decision 4), the promotion test (decision 7), and the batching rule (decision 8). State
  explicitly that `noted` is the default and that every disposition is recorded in the
  ticket's `## Review` table.
- §3, lineage bullet (lines 141–155): note that a spawned ticket may carry **several** findings
  (batching), so `spawned-by:` is one id but the parent's Review table is the itemised record.
- §8, the pickup-freshness paragraph (lines 247–252): route the gate's "adjacent work" through
  §5's dispositions instead of straight to new tickets.
- **§8, line 244 — the contradiction (added at pickup, amendment 1).** "Nothing is built directly
  from a chat message, **a review finding**, or a raw idea — only from a ticket whose
  Implementation Plan has met the READY gate." That forbids `fixed inline` outright. It sits in
  §8 but outside the range cited above, so the original task list would have shipped a
  self-contradictory payload — the exact "two overlapping rules for the same case" defect this
  plan invokes to justify deleting the line-116 carve-out. Reword so the pipeline rule keeps its
  force for *features* while the four dispositions govern *findings*.

**2. `skill/resources/review-protocol.md` — the procedure.**

- §0 scope block, lines 21–24: replace "— never inline drift" with the bounded inline path,
  pointing at rules §5 for the vocabulary. Keep "a review does not re-implement the ticket".
- §5, lines 99–117: rewrite the non-blocking bullet to route per rules §5, with the findings
  table gaining a **required `disposition` column**. Delete the "trivial cosmetic spec
  typos" carve-out — it is subsumed by `fixed inline` and its survival would give two
  overlapping rules for the same case. **It spans lines 116–117**, not 116 alone ("… may be
  patched directly" / "and noted here."); deleting only 116 orphans the trailing clause.
- §6b, line 127: "only non-blocking already spawned as new tickets" → "only non-blocking, all
  dispositioned".
- Checklist, line 176: update to "findings classified and **dispositioned** per rules §5".

**3. `skill/SKILL.md` — the two gate summaries.**

- *Procedure: validate a ticket*, step 1 (line 195): "→ new `1-to-do/` ticket(s)" → "→
  dispositioned per rules §5 (default: noted)".
- *Procedure: implement a ticket*, step 3 last bullet (line 162): "adjacent work → new
  `1-to-do/` ticket(s)" → "adjacent work → dispositioned per rules §5, batched by theme".
- *Procedure: refine a ticket* (lines 130–143): add a split rule — split a ticket at refinement
  **only when the split part is independently schedulable**; otherwise it stays a task in the
  plan. (T-033 is the counter-example that motivated this gate's inclusion.)

**4. `skill/resources/TEMPLATE.md` — the Review section shape.**

- Lines 107–110: name the required findings-table columns including `disposition`, and point at
  rules §5 for legal values. Do not restate the four values (decision 2).
- Line 120, the History example: `review clean; non-blocking → T-MMM` →
  `review clean; 6 non-blocking (3 inline, 2 folded, 1 → T-MMM)` — the T-030 shape, which is
  what a correctly-dispositioned review now looks like.

**4b. The generated `AGENTS.md` marker block — same contradiction, three more copies**
(added at pickup, amendment 1).

The sentence from task 1's line 244 is **triplicated** outside the payload, and the original plan
declared all three out of scope. They must move in step or `pickle install` will emit a block that
contradicts the rules it points at:

- `internal/install/install.go:586` — inside `markerBlock`; reword to match.
- `internal/install/testdata/markerblock.golden:4` — regenerate (there is an `-update`-style
  golden path; check the test's own mechanism rather than hand-editing).
- `AGENTS.md:21` — this repo's own rendered block, between the `pickle:begin`/`pickle:end`
  markers. Hand-edit to match the new `markerBlock` output exactly, since `pickle upgrade`
  rewrites this region and any divergence resurfaces as drift.

**5. `internal/install/install_test.go` — pin the two decisions mechanically.**

Add `TestPayloadDispositionVocabulary`, using the existing `payloadRoot()` idiom (`:15`). Read the
**four explicit paths** — `skill/resources/tickets-README.md`, `skill/resources/review-protocol.md`,
`skill/SKILL.md`, `skill/resources/TEMPLATE.md`. (Amendment 3: the original plan said
`skill/resources/*.md`, which does not contain `SKILL.md` — that lives at `skill/SKILL.md` — and
would sweep in `resources/BOARD.md`, which T-044 deletes.) Assert exactly two things — the
*decisions*, not the prose:

- the string `never inline drift` appears **nowhere** in the payload (guards decision 4 against
  a future revert). Verified at pickup: exactly one occurrence today, `review-protocol.md:24`, so
  the assertion is currently red and turns green only when task 2 lands.
- **`fixed inline` and `folded` appear in `tickets-README.md` and in no other payload file.**
  (Amendment 3: the original "does not contain all four tokens" assertion was near-vacuous — it
  passes whenever any *one* token is absent, so a file could restate three of the four and stay
  green. These two are the novel tokens and the only discriminating ones; `new ticket` and `noted`
  legitimately occur in running prose elsewhere.)

Do **not** assert on a `rules §5` reference string (amendment 3): it appears zero times in the
payload today, and `TEMPLATE.md` uses a different established idiom (`tickets/README.md §N`).
Inside `review-protocol.md`, write cross-references as `the rules §5` — the existing idiom at
`:111`/`:166` — because the protocol has its own §5 and a bare "§5" there reads as a
self-reference.

Keep it to those two assertions. A test that pins wording rather than decisions would be
re-writing the diff, and this repo already carries enough drift-guard debt (T-040, T-042).

### Acceptance test

**A. Mechanical — must be green.**

```
just build && just test && just lint
./pickle board audit          # expect: 45 tickets, 0 error(s), 0 warning(s)
./pickle doctor               # expect: 0 error(s), 1 warning(s) — see below
```

Expected: `go test ./...` passes including the new `TestPayloadDispositionVocabulary`;
`gofmt -l` and `go vet ./...` clean. `board audit` ticket count unchanged (no ticket moves in
this task list beyond T-036's own).

**`doctor` cannot report "clean" here — amendment 2.** The original plan expected clean; measured
at pickup, `main` already emits `payload version "0.0.0-skeleton" differs from binary "v0.0.0-…"`
because `pickle.toml` pins the skeleton version while a locally-built binary stamps a VCS
version. So the bar is: **0 errors, exactly 1 warning, and that warning is the pre-existing
payload-version one** — no new warning, and the skill dir still a symlink to `skill/`. Taken
literally, the old wording would have failed the acceptance test through no fault of the change.

**B. Retroactive replay — the real test of "precise enough that two reviewers agree".**

The historical findings tables in `6-done/` are frozen data. Re-classify, against the **new**
rules §5 only, every finding in the four reviews that recorded per-finding dispositions.
Counts verified against the actual tables at pickup:

| review | non-blocking | also replayed | row total |
|---|---|---|---|
| T-008 | 2 (N1–N2) | — | 2 |
| T-024 | 13 (N1–N13) | — | 13 |
| T-029 | 7 (N1–N7) | N8, N9 (`informational, no action`) | 9 |
| T-030 | 6 (N1–N6) | — | 6 |
| **total** | **28** | **2** | **30** |

**Amendment 5 — the set was wrong twice.** T-029 N8/N9 are labelled `informational, no action`
rather than `non-blocking`, so the 28-row set excluded **the only empirical precedent for the
`noted` disposition this ticket is ratifying** (besides T-024 N12). They are now in: replay **30
rows**. And criterion 2 below ranges over 9 inline fixes, **two of which are outside even the
30** — T-005's sole finding (T-005 is not one of the four replayed reviews) and T-008 N3 (labelled
`trivial (patched inline)`, not `non-blocking`). They are replayed as named precedents, not as
table rows.

Pass criteria, all three required:

1. **Every one of the 30 maps to exactly one disposition** — zero findings ambiguous between
   two. Ambiguity is the failure mode the bar exists to prevent; a single ambiguous finding
   means the wording is not done.
2. **All 9 inline fixes are authorized** by the new bar: T-005 ×1 (`T-005:33`, "patched directly
   during review"), T-008 N3, T-024 ×4 (N4, N9, N10, N13), T-030 ×3 (N3, N4, N6). Under the old
   rules only the T-005 one was. The T-005 and T-008 N3 checks are precedent checks against the
   bar's wording, since neither is a row in the 30.
3. **The new rules produce no more new tickets than were actually filed** for those four
   reviews (actual: T-015, T-029, T-030, T-031, T-032, T-038 = 6 — verified at pickup).

Record the 30-row replay table in the ticket's `## Review` section as the evidence. If criterion
1 fails, the fix is wording — not an exception list.

### Docs update

The payload **is** the docs for this change; tasks 1–4b are the documentation.

- `README.md` — verified: zero matches for blocking/non-blocking dispositions; **no change**.
- `PLAN.md` — historical record; **no change** (T-024 N12 precedent). Its line 51 ("reviewing and
  classifying findings blocking vs non-blocking") stays true.
- `AGENTS.md` marker block — **corrected at pickup (amendment 1): this DOES need to change.** The
  original plan's claim that it "states no disposition rules" was wrong; `install.go:586` carries
  the "nothing is built from … a review finding" sentence. See task 4b —
  `markerBlock`, `testdata/markerblock.golden` and `AGENTS.md` all move with it.
- Also verified as **still true, no change**: `tickets-README.md:75` (mermaid `review clean OR only
  non-blocking findings`), `:239` (ASCII `6-done/ (clean / non-blocking only)`), `SKILL.md:3`
  (frontmatter description) and `install.go:616` `ticketsReadme` (points at the protocol by path
  only).

### Finish

Summary of what changed at each of the three gates, plus the completed 28-row replay table.
Commit locally on the branch; **present the message and MR attributes for approval before any
push**.

```
feat(skill): route review findings four ways, note-and-close by default (T-036)

The shipped protocol authorised one disposition for a non-blocking finding —
mint a ticket — and forbade the alternative ("never inline drift"). Practice
overruled it in every review: reviewers batched (T-018's 7 findings → 1
ticket), folded into tickets that already owned the ground, fixed prose and
idiom inline, and closed findings with no action, all undocumented.

Ratify the four dispositions actually in use, make `noted` the default, and
state them once in rules §5 so the protocol, SKILL.md and TEMPLATE.md
reference rather than restate them. The inline bar is the one the T-030
review arrived at empirically and recorded as precedent for this ticket:
prose or idiom in code authored on the same branch, no behaviour change.
Every disposition is recorded in the findings table; nothing is fixed
silently.

Applied at all three spawn gates, not just the review: refinement measured
R_review ≈ 1.0 over 13 reviews, while the pickup applicability gate and
refinement splits account for the rest of the ≈ 2.5 spawns per completed
ticket. T-030 alone spawned four tickets across the three.

The backlog cap and the `user-visible:` axis are split out to T-045, to be
justified by measurement after this lands rather than assumed now.
```

## Review

<!-- the acceptance-test-B evidence in the three "implementer's" subsections at the end was
     produced during implementation, as the plan requires; everything before them is the review -->

**Verdict: PASS — 0 blocking, 11 non-blocking, all dispositioned (5 fixed inline, 2 folded,
4 noted, 0 new tickets). Reviewed on `feat/T-036-review-disposition-valves` @ f05b207 (+ this
review's fixups), un-merged and publish-gated.**

This is the first review conducted **under the rules it is validating** — the repo self-hosts the
payload through the `.agents/skills/ticket-flow → skill/` symlink, so checking the branch out
swapped the protocol in. An 11-finding review that spawns **zero** tickets is the change working:
under the old §5 every one of those 11 was legally obliged to become a `1-to-do/` ticket. Record
that as T-045's first datapoint, with the caveat the implementer states honestly below — one
review is not a spawn rate.

### Checklist

- [x] Implementation audit — acceptance test re-run verbatim (A **and** the 30-row B replay
      re-verified against the frozen tables), tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — payload *is* the docs; whole-tree sweep + fresh-install end-to-end
      check (step 4a)
- [x] Findings recorded with severity **and** disposition per rules §5; disposition summary
      present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6b)
- [x] `BOARD.md` updated (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval; bookkeeping committed
      (step 9)

### Implementation audit (step 2)

| Item | Result | Evidence |
|---|---|---|
| Task 1 — rules §5 rewritten: severity/disposition split, four-row table, inline bar, promotion test, batching | **met** | `tickets-README.md:192-247` |
| Task 1 — §3 lineage bullet notes batching; new "Splitting at refinement" bullet | **met** | `tickets-README.md:156-165` |
| Task 1 — §8 pickup paragraph routed through §5's dispositions | **met** | `tickets-README.md:303-307` |
| Task 1 (amendment 1) — §8's "nothing is built from … a review finding" contradiction reworded | **met** | `tickets-README.md:292-296` — pipeline governs *features*, §5 governs *findings* |
| Task 2 — protocol §0 scope block: "never inline drift" → bounded inline path, "does not re-implement" kept | **met** | `review-protocol.md:21-28` |
| Task 2 — protocol §5 rewritten, `disposition` column required, blocking cell `—`, summary line mandated | **met** | `review-protocol.md:103-130` |
| Task 2 — line-116/117 cosmetic carve-out deleted whole (no orphan clause) | **met** | `rg 'purely-cosmetic\|patched directly' skill/` → 0 hits |
| Task 2 — §6b and the step-5 checklist item updated | **met** | `review-protocol.md:140-142, 190` |
| Task 3 — SKILL.md validate step 1, implement step 3, refine split rule (+ renumber 5→7) | **met** | `SKILL.md:205-211, 170-175, 145-152`; new rules-summary bullet at `:92-97` |
| Task 4 — TEMPLATE.md Review columns incl. `disposition`, pointing at `tickets/README.md §5` | **met** | `TEMPLATE.md:108-112` |
| Task 4 — History example reshaped to a disposition summary | **partially met → N2** | planned text is untestable; corrected inline |
| Task 4b — `install.go:586` `markerBlock`, `testdata/markerblock.golden`, `AGENTS.md:20-23` all moved in step | **met** | `TestMarkerBlockGolden` green; fresh install rendered the new paragraph verbatim |
| Task 5 — `TestPayloadDispositionVocabulary`, four explicit paths, the two amended assertions | **met** | `install_test.go:624-683`; assertion 1 verified **red on `main`** (`git show main:…review-protocol.md` → 1 hit) |
| Acceptance A — `just build && just test && just lint`, `gofmt -l` | **met** | all green |
| Acceptance A — `board audit` 45 tickets / 0 / 0 | **met** | verbatim |
| Acceptance A (amendment 2) — `doctor`: 0 errors, exactly 1 warning, and it is the pre-existing payload-version one | **met** | `payload version "0.0.0-skeleton" differs from binary …` only |
| Acceptance B — 30 rows, 3 criteria | **met, independently re-verified** | see below |
| Decisions 1–3, 5–10 | honoured | — |
| Decision 4 (inline bar *verbatim*) | **deliberately deviated, recorded** | F3; ratification requested — see N10 |

**Acceptance B re-verified against the frozen tables, not the ticket's prose.** Row counts
`rg '^\| N' 6-done/T-{008,024,029,030}` → 3 / 13 / 9 / 6; the replay's set is T-008's 2
non-blocking (N3 is labelled `trivial (patched inline)`, correctly held out as a precedent),
T-024's 13, T-029's 9 (N8/N9 `informational, no action` included per amendment 5), T-030's 6 =
**30** ✓. Splits N3→N3a/N3b on T-029 and T-030 are both justified by the frozen record carrying
two dispositions in one cell (`T-030:688` literally reads "fixed inline (comment reworded);
substantive note folded into T-027") ✓. The summary arithmetic checks: 9 + 8 + 13 + 2 = 32 rows
from 30 findings ✓; the 13 new-ticket rows collapse to exactly the 6 tickets history filed
(T-015, T-029, T-030, T-031, T-032, T-038) ✓. Criterion 1 (no row ambiguous), 2 (all 9 historical
inline fixes authorized) and 3 (≤ 6 new tickets) all hold. One claim inside criterion 2 is
overstated — N5.

**Fresh-install end-to-end (beyond the plan's acceptance test).** `pickle install` into a scratch
repo shipped the rewritten §5 and the reworded marker block byte-for-byte. That is how N1 was
caught: the miscount ships to every adopting project, not just this repo.

### Findings (step 5)

| # | severity | disposition | description | evidence | resolution |
|---|---|---|---|---|---|
| N1 | non-blocking | **fixed inline** | Rules §5 introduced its rule list as "**Four** rules" while **five** bullets follow (F2 added "one finding, one disposition" without updating the count). An off-by-one in the normative section the ticket exists to author, shipped to every installed project. | `tickets-README.md:221` + 5 bullets at `:223,226,231,235,237`; reproduced in a fresh `pickle install` | "Four" → "Five" |
| N2 | non-blocking | **fixed inline** | TEMPLATE.md's History example deviates from task 4's specified text and drops the per-disposition counts §5's own summary rule now requires. **The deviation was forced, not sloppy:** the planned string `(3 inline, 2 folded, 1 → T-MMM)` contains `folded`, which task 5's assertion forbids outside the rules — tasks 4 and 5 are mutually unsatisfiable as written. It went unrecorded, which is the actual defect. | measured: restoring the planned text fails `TestPayloadDispositionVocabulary` (`install_test.go:675`) | Reworded to keep the counts without the token: "6 non-blocking, all dispositioned (3 fixed in review, 2 absorbed by T-KKK, 1 → T-MMM)" |
| N3 | non-blocking | **fixed inline** | The new test was inserted directly beneath `TestVerifyStampedVersion`'s doc comment, so godoc/`go doc` attach that pre-existing paragraph to `TestPayloadDispositionVocabulary` and the older test is left undocumented. The branch did not author the comment — it **made it false**, which is exactly the widened bar's own case. | `install_test.go:624-628` before the fix | Comment moved back onto its function |
| N4 | non-blocking | **fixed inline** | `SKILL.md` contained the literal token `fixed inline` — the one thing the new guard asserts appears only in the rules — and the assertion passed **purely because the phrase happened to soft-wrap** across two lines. The drift-guard's stated invariant was already false in the tree it guards. | `rg -U 'fixed\s+inline' skill/` → `SKILL.md:173-174` | Reworded to "takes the inline disposition"; the invariant now actually holds |
| N5 | non-blocking | **fixed inline** | Criterion 2's parenthetical "only 1 of the 9 was, under the old rules" is wrong, and it is the headline number for the inline valve's measured gain. T-024 **N9 and N13** were typos in `skill/resources/tickets-README.md` — payload prose, i.e. squarely inside the old carve-out's "workflow scaffolding itself". Meanwhile the one it credits (T-005, a Go doc comment at `cli.go:14`) was the *stretch*, authorized only because that review said so. | `T-024:393,396`; old carve-out text at `main:review-protocol.md:116-117`; `T-005:33` | Corrected at the site, below |
| N6 | non-blocking | **noted** | `TestPayloadDispositionVocabulary`'s second assertion is a raw substring test over soft-wrapped markdown, so it fails in both directions: it **under-detects** a genuine restatement that happens to wrap (N4) and **over-constrains** legitimate prose that merely uses the words (N2 — an example History line is not "restating the list", which is all decision 2 forbids). Not fixed here: normalising whitespace or narrowing the assertion changes test behaviour, so the inline bar excludes it, and "loosen a payload assertion" fails the promotion test on its own. | `install_test.go:669-682`; N2 and N4 are the two live instances | Recorded for whoever trips it next; promote by citing this row |
| N7 | non-blocking | **folded** (T-044) | `BOARD.md`'s IN REVIEW `branch` cell reads `feat/T-036-ratify-the-four-…-default` — the filename slug — while the real branch is `feat/T-036-review-disposition-valves`. Pre-existing class (T-029 N9 → T-023, dropped), not this branch's doing. | `BOARD.md:45` vs `git branch --show-current` | T-044 **D2 drops the `branch` column** and "kills the T-023 class" (`T-044:121-122`) — no new item needed. This instance was hand-corrected as step-7 bookkeeping (the DONE row `move` wrote carried the same wrong branch, which a human would try to check out at merge time); the *defect* is still `folded`, not fixed |
| N8 | non-blocking | **folded** (T-044) | `BOARD.md`'s preamble still reads "Last updated: 2026-07-26 (T-036 refined → READY …)" although the row has since moved twice. `pickle ticket move` never refreshes that line; only `board sync` does (`sync.go:184`). The board's most-read line contradicts its own table. | `BOARD.md:28`; `internal/sync/sync.go:184-186`. Measured: `board sync --dry-run` reports `OUT OF SYNC (1 change(s)) — reformat only (… Last-updated)` against the **pre-review** board too (reproduced in a scratch copy of `tickets/` at `HEAD` with T-036 back in `4-in-review/`), so any board `ticket move` last touched is permanently "drifted" | T-044 makes the board generated and normalises `Last updated:` on every write (`T-044:133,146`); this instance refreshed by hand as step-7 bookkeeping |
| N9 | non-blocking | **noted** | Mild tension the ticket did not close: §5 says "a blocking finding is **never** dispositioned: it is fixed", while §8 gives the pickup gate's findings the same four dispositions and calls a plan amendment `fixed inline` — and this very ticket's History labels two such amendments "**blocking** amendments". Coherent if §5's sentence is read as review-scoped (it sits in the `5-rework/` bullet), which is the natural reading; a clause would remove the doubt. | `tickets-README.md:205-206` vs `:303-305`; `T-036` History 2026-07-26 pickup line | Left as is; the review-scoped reading is sound |
| N10 | non-blocking | **noted** | The shipped inline bar is **not** decision 4's verbatim T-030 wording: F3 widened "prose or idiom in code authored on this same branch" to "**authored, or made false**". A recorded, argued deviation from a confirmed decision — normally blocking. It is not, because the plan was **internally unsatisfiable**: acceptance criterion 2 requires T-008 N3 (`PLAN.md:11`, prose the branch *falsified*) to be authorized, and the verbatim bar rejects it. Reverting would fail the ticket's own acceptance test, so rework has no coherent scope. The widening stays bounded — T-024 N11 and T-030 N5 are pre-existing defects with tempting one-line fixes and both still land on `folded`. | ticket F3; `T-008:389`; `tickets-README.md:216,226-230` | **Requires the user's explicit ratification** (see below); no code change |
| N11 | non-blocking | **noted** | Protocol step 9 still frames the hand-back as "findings by severity … any newly-spawned tickets", the pre-change shape; with §5 rewritten it should ask for the disposition summary. Not falsified by the branch, just left incomplete. | `review-protocol.md:162` | Recorded; a one-clause edit for whoever next touches step 9 (T-016 is queued on this file) |

**Disposition summary:** 11 non-blocking findings, 0 blocking. **5 fixed inline** (N1, N2, N3,
N4, N5) · **2 folded** into T-044 (N7, N8) · **4 noted** (N6, N9, N10, N11) · **0 new tickets**.
Under the old §5 all 11 were obliged to become `1-to-do/` tickets.

### The one thing the user must ratify — the widened inline bar (N10)

Confirmed decision 4 said "adopt T-030's bar **verbatim**, do not invent new wording". The shipped
bar is `prose or idiom **this branch authored, or made false** — and no behaviour change`. The
implementer flagged this himself and asked for it to be weighed; the review's position:

- **Accept it.** The verbatim bar's justification was that it is field-tested — but its three
  T-030 datapoints were *all* branch-authored, the one sample that cannot distinguish "authored"
  from "caused". Replaying it over 30 rows is precisely what the acceptance test was for, and it
  broke on the first case outside T-030 (T-008 N3: `PLAN.md` listed `board sync` as pending after
  the branch had delivered it — a one-word fix to a line the change itself made wrong).
- The bar still refuses the dangerous case: pre-existing defects. Two rows in the replay
  (T-024 N11, T-030 N5) and one in *this* review (N7, N8 → T-044) are exactly that, and all
  three come out `folded`, not inline. "Did this branch break it?" is a sharper question than
  "is it small?" and it is the one now written into the rules.
- This review then immediately exercised it: **N3 is a "made false", not an "authored", case.**
  Under the verbatim bar the orphaned doc comment would have had to become a ticket.

If the user rejects the widening, this goes to `5-rework/` with a scope of exactly two things:
revert the bar to T-030's wording and re-cut acceptance criterion 2 (which the verbatim bar
cannot satisfy).

### Consistency & docs notes (steps 3, 4, 4a)

- **Single-source-of-truth held.** The vocabulary is defined once, in rules §5. `review-protocol.md`
  §5 explicitly says "do not restate the list of dispositions here" and doesn't; `TEMPLATE.md`
  points at `tickets/README.md §5`, the established idiom that resolves through the installed
  pointer file (`internal/install/install.go:623`); `SKILL.md` references §5 five times. This is
  the T-024-N9/T-040 drift pattern being deliberately avoided.
- **Cross-reference idiom respected.** Inside `review-protocol.md`, foreign references read "the
  rules §5" (`:24,110,118,122,158,190`) while bare `§5`/`§6a` (`:23`) mean the protocol's own
  sections — the distinction amendment 3 asked for. Two lines apart it is subtle but correct.
- **No stale instruction survives.** `rg 'non-blocking' --glob '!tickets/**'` shows every
  remaining occurrence is either the new severity language or a still-true summary — the mermaid
  edge `IN_REVIEW --> DONE: review clean OR only non-blocking findings` (`:75`) and the ASCII
  `6-done/ (clean / non-blocking only)` (`:287`) both hold under the new §5, as the docs step
  claimed.
- **Docs coverage (4a.1) is complete because the payload *is* the doc.** `README.md` genuinely has
  zero disposition content (verified), so "no change" is right, not an omission. `PLAN.md` is a
  historical record per the standing T-024 N12 precedent, and its line 51 ("classifying findings
  blocking vs non-blocking") remains true. No docs build is configured for this child.
- **The generated block is consistent with the payload it points at.** After task 4b all four
  copies of the reworded sentence agree: `install.go:586`, the regenerated golden, this repo's
  `AGENTS.md:20-23`, and a fresh install's rendering. Had 4b been skipped as the original plan
  intended, `pickle install` would have emitted a block contradicting its own rules — the pickup
  gate's most valuable catch.
- **Quality.** The new test asserts *decisions*, not wording, exactly as task 5 demanded, and its
  doc comment explains why each assertion exists — with the one caveat at N6. No production code
  changed apart from one string literal; risk is confined to prose and one golden file.

### Impact sweep (step 8)

No ticket lists T-036 in `depends-on:` (`board audit` clean, and `rg 'T-036' 1-to-do/ 2-ready/`
returns only narrative references). Four tickets reference it; none has an invalidated assumption:

- **T-045** — measurement-gated on exactly this change. Its assumption ("re-measure the spawn rate
  after T-036 lands") is now *satisfiable*: this review is datapoint 1, and it is a strong one
  (11 findings → 0 tickets). History line added there so the count starts from a recorded base.
- **T-044** — absorbs N7 and N8; its D2 (drop the `branch` column) and generated-board design
  already cover both. Item noted in its Description. Sequencing recorded at pickup — T-036 lands
  first — still holds, and its Task 7 must now expect the *new* §5 text in the three payload files.
- **T-041** (marker-block freshness) — this ticket hand-edited `AGENTS.md` to match `markerBlock`,
  which is the drift T-041 exists to detect. Strengthens its case; no assumption broken.
- **T-040** — its reference was already corrected at T-036's refinement ("the TO DO cap moved from
  T-036 to T-045"); still accurate. The `.gitkeep` finding folded in at pickup is untouched.
- **T-016** and **T-022** — both edit files this ticket rewrote (`review-protocol.md` step 4b;
  payload conditionals). Neither is refined, so no plan text went stale; the soft coupling now
  points at new line numbers, which is a refinement-time concern. N11's one-clause edit to step 9
  is a natural drive-by for whichever lands first.

### Acceptance test B — the 30-row retroactive replay (implementer's run)

Every finding re-classified against the **new** rules §5 only. `→` marks a new ticket.

| # | historical disposition | new disposition | note |
|---|---|---|---|
| **T-008** N1 | → T-015 | **new ticket** → T-015 | batched with N2 |
| T-008 N2 | → T-015 | **new ticket** → T-015 | same ticket, one theme |
| **T-024** N1 | → T-029 | **new ticket** → T-029 | batched with N5, N6 |
| T-024 N2 | → T-030 | **new ticket** → T-030 | distinct theme (input sanitisation) |
| T-024 N3 | absorbed by T-027 | **folded** (T-027) | |
| T-024 N4 | patched during review | **fixed inline** | branch-authored doc comment |
| T-024 N5 | → T-029 | **new ticket** → T-029 | strengthening an assertion changes test behaviour → not inline |
| T-024 N6 | → T-029 | **new ticket** → T-029 | |
| T-024 N7 | → T-015 item 5 | **folded** (T-015) | |
| T-024 N8 | already deferred, T-015 item 4 | **folded** (T-015) | needed the amended `folded` wording — see F1 |
| T-024 N9 | patched during review | **fixed inline** | payload prose edited on this branch |
| T-024 N10 | patched during review | **fixed inline** | sentence inserted by this branch |
| T-024 N11 | → T-019 item 3 | **folded** (T-019) | pre-existing error — correctly *not* inline |
| T-024 N12 | no action — deliberate | **noted** | the disposition's own precedent |
| T-024 N13 | patched during review | **fixed inline** | payload heading this branch made false |
| **T-029** N1 | → T-031 | **new ticket** → T-031 | batched with N2, N3a, N4 |
| T-029 N2 | → T-031 | **new ticket** → T-031 | |
| T-029 N3a | → T-031 | **new ticket** → T-031 | **split** — see F2 |
| T-029 N3b | → T-031 | **fixed inline** | the imprecise comment half, branch-authored |
| T-029 N4 | → T-031 | **new ticket** → T-031 | |
| T-029 N5 | → T-032 | **new ticket** → T-032 | distinct theme (duplication) |
| T-029 N6 | noted in T-027's couplings | **folded** (T-027) | |
| T-029 N7 | → T-031 | **fixed inline** | comment nit, branch-authored — **the valve working**: a ticket item under the old rules |
| T-029 N8 | informational, no action | **noted** | |
| T-029 N9 | already T-023 | **folded** (T-023) | |
| **T-030** N1 | → T-038 | **new ticket** → T-038 | batched with N2 |
| T-030 N2 | → T-038 | **new ticket** → T-038 | |
| T-030 N3a | fixed inline | **fixed inline** | **split** from N3b — see F2 |
| T-030 N3b | folded into T-027 | **folded** (T-027) | |
| T-030 N4 | fixed inline | **fixed inline** | |
| T-030 N5 | folded into T-019 item 4 | **folded** (T-019) | pre-existing — correctly *not* inline |
| T-030 N6 | fixed inline | **fixed inline** | the clearest instance of the bar |

Out-of-set inline precedents (criterion 2 ranges over these, they are not rows in the 30):

| precedent | new disposition | note |
|---|---|---|
| T-005, sole finding (`cli.go:14`) | **fixed inline** | doc comment this branch authored |
| T-008 N3 (`PLAN.md:11`) | **fixed inline** | **only via "made false"** — see F3 |

**Disposition summary:** 32 dispositioned rows from 30 findings (2 splits) — 9 fixed inline,
8 folded, 13 → 6 new tickets, 2 noted. Plus the 2 out-of-set precedents: **11 inline total**.

**Criteria:**

1. **PASS** — all 32 rows map to exactly one disposition, zero ambiguous, after the three
   wording fixes below.
2. **PASS** — all 9 inline fixes authorized (only 1 of the 9 was, under the old rules), and the
   new bar authorizes 2 *more* (T-029 N3b, N7) that were ticket items historically.

   > **Review correction (N5, 2026-07-26).** "Only 1 of the 9" is wrong. The old carve-out was
   > "trivial, purely-cosmetic spec typos **in the workflow scaffolding itself**", so T-024 **N9**
   > and **N13** — both typos in `skill/resources/tickets-README.md` — were squarely authorized,
   > and T-005's Go doc comment (`cli.go:14`) was the one that stretched the rule. **3 of the 9
   > were authorized** under the old rules, not 1; the inline valve's measured gain on this data is
   > 6 newly-authorized fixes plus the 2 that were ticket items (T-029 N3b, N7), not 8 plus 2. The
   > criterion still passes — every one of the 9 is authorized by the new bar — and the direction
   > of the argument is unchanged.
3. **PASS** — exactly **6** new tickets, equal to the 6 actually filed. Not fewer.

### Findings the replay produced — three wording fixes, applied

The replay was run to test the wording, and it failed three times before passing. Each fix is
wording, not an exception list, as the plan requires.

- **F1 — `folded` did not cover the commonest case.** T-024 N8's ground was already an *item* in
  T-015; nothing needed adding. As written ("add an item to it") the row forced a choice between
  `folded` and `noted` — an ambiguity, and criterion 1 fails on a single one. Now: "add an item to
  it — **or cite the item already there**".
- **F2 — compound findings had no rule.** T-030 N3 carried **two** dispositions in the frozen
  record ("fixed inline (comment reworded); substantive note folded into T-027"), and T-029 N3 is
  the same shape. A vocabulary cannot fix that; a rule can. Added: **one finding, one
  disposition** — split separable parts into rows first, because "a row carrying two dispositions
  was never actually classified." This is the one structural gap the replay found that no amount
  of careful wording on the four rows would have closed.
- **F3 — the inline bar under-authorized by exactly one, and it is a deviation from confirmed
  decision 4.** Decision 4 adopted T-030's bar *verbatim* — "prose or idiom in code authored on
  this same branch". Replayed beyond T-030's own findings it fails **T-008 N3**: `PLAN.md:11`
  listed `board sync` as still pending, which T-008 had just delivered. That prose was **not
  authored on the branch** — the branch *falsified* it. Under the verbatim bar it must become a
  ticket or a `noted`, for a one-word edit to a file the change itself made wrong; criterion 2
  fails. The bar is now **"prose or idiom this branch authored, or made false"**, with an
  explicit rule that it is **about causation, not size**: "did this branch break it?", not "is it
  small?".

  This is a deliberate, recorded deviation from a confirmed decision, and the reviewer should
  weigh it. The defence: decision 4's stated reason was that T-030's bar is *field-tested*, and
  it was — on three findings, **all** of them branch-authored, which is precisely the sample that
  cannot distinguish "authored" from "caused". Widening it is what the 30-row replay is for. It
  stays bounded and still discriminates on this data: T-024 N11 and T-030 N5 are pre-existing
  defects with tempting one-line fixes, and both correctly remain `folded`.

### Honest reading of criterion 3

Criterion 3 passes at exactly 6 — it does not improve on history, and the replay explains why:
**these reviewers were already batching well.** 13 new-ticket findings became 6 tickets, and the
worst historical case (T-018: 7 findings → 1 ticket) is not even in the replay set. The
measured gain on this data is therefore the inline valve (9 → 11 authorized) and nothing else.

The rest of the expected effect is **prospective and untested here**: `noted` as the default and
the promotion test bite at the moment a reviewer decides, and this replay cannot simulate that
decision honestly — the historical reviewers did not ask "would this be scheduled?", so
re-answering it for them would be inventing data. What the board does suggest is where it would
have bitten: of the 6 tickets, T-029 and T-030 were built, while **T-015, T-031 and T-032 were
never scheduled and were absorbed into epics** at the 2026-07-26 triage. That is the promotion
test's target, but it is an observation, not a criterion.

This is the first datapoint for **T-045**, and it should be recorded there as one: the spawn rate
cannot be re-measured from a replay, only from the next three real reviews.

## History

- 2026-07-26 — created (TO DO). source: chat — board growth analysis; measured R ≈ 2.3
  spawned tickets per reviewed ticket across T-024 → T-029/T-030 → T-031/T-032/T-033/T-034/T-035,
  with `7-dropped/` empty after 35 tickets
- 2026-07-26 — refined. **The filed measurement did not survive re-verification** and the
  Description was rewritten around the corrected one. Its stated source (`spawned-by:` lineage)
  is empty on every ticket in the original table — the backfill ticket T-025 was dropped, so the
  real source is the `## Review` prose in `6-done/`. And R ≈ 2.3 conflated three spawn gates:
  re-measured over all 13 reviews, **R_review ≈ 1.0**, with the pickup applicability gate and
  refinement splits accounting for the rest of ≈ 2.5 spawns per completed ticket (T-030 alone
  spawned 4 across the three). Scope widened to all three gates on that evidence.
  Second finding: **all four proposed dispositions are already in use, undocumented** — 9
  inline fixes, folds into T-012/T-013/T-015/T-019/T-027, batched tickets (T-018's 7 findings →
  1), and note-only closes (T-011, T-024 N12) — so this is ratification, not design. The T-030
  review recorded its deviation "as precedent to weigh when T-036 is refined"; that precedent
  is adopted as the inline bar verbatim.
- 2026-07-26 — retitled (id stable, slug tidied per rules §3): the old title advertised
  backlog-cap valves that refinement split out, and a wrong title on the board's top ticket is a
  defect in the most-opened surface. Board row updated in the same change.
- 2026-07-26 — 3 decisions taken with the user: (1) cover all three spawn gates, not §5 alone;
  (2) split valves 3–4 (backlog cap, `user-visible:` axis) out to **T-045**, gated on
  re-measuring the spawn rate after this lands rather than assumed now — both are backstops for
  the leak this ticket plugs; (3) adopt T-030's field-tested inline bar verbatim, no line-count
  ceiling. Grades re-checked against the backlog and held at **high / medium / M**: impact is
  the ceiling below `critical` (this changes the flow's core loop for every installed project,
  but does not reshape the product), and the widened three-gate scope plus the 28-finding
  replay puts cost at the top of M rather than into L.
- 2026-07-26 — TO DO → READY: plan complete
- 2026-07-26 — pickup applicability gate run (unbiased sub-agent, rules §8). Verdict **proceed
  with amendments**; nothing invalidated. The gate confirmed 13 of 16 line citations exact (3 off
  by ≤1) and re-verified every historical number the plan rests on — 2/13/7/6 = 28 findings, the
  9 inline fixes, the 6 tickets actually spawned — all exact. Two blocking amendments applied:
  (1) `tickets-README.md:244` ("nothing is built directly from … a review finding") flatly
  contradicts `fixed inline` and sits *outside* the §8 range the plan cited, so the original task
  list would have shipped a self-contradictory payload; the same sentence is triplicated at
  `install.go:586`, `testdata/markerblock.golden:4` and `AGENTS.md:21`, all three of which the plan
  had declared out of scope — **user chose to fix all four surfaces**, adding task 4b;
  (2) acceptance test A expected `./pickle doctor` clean, but the pinned
  `payload_version = "0.0.0-skeleton"` guarantees 1 warning against any dev build, so the test was
  red at baseline regardless of this change. Three non-blocking amendments: task 5's glob missed
  `skill/SKILL.md` (not under `resources/`) and its "not all four tokens" assertion was
  near-vacuous — replaced with "`fixed inline` and `folded` appear in `tickets-README.md` only";
  the `rules §5` reference string does not exist in the payload yet and collides with TEMPLATE.md's
  `tickets/README.md §N` idiom; the line-116 carve-out actually spans 116–117.
  Acceptance test B widened 28 → **30 rows** (user decision): T-029 N8/N9, labelled
  `informational, no action`, were excluded — and they are the `noted` disposition's own
  precedent, so the criterion most at risk was untested. Sequencing recorded: **T-036 lands
  before T-044**, whose Task 7 edits three of the same payload files.
  One adjacent finding **folded into T-040** rather than spawned: none of the seven `.gitkeep`
  files `install.go:311-319` creates is tracked here, so `3-in-development/` has vanished and a
  fresh clone would lack three status dirs, which `board audit` cannot detect by design. Harmless
  for this pickup — `move.go:124` does `MkdirAll` before the rename and `LoadAll` treats a
  vanished-empty dir as zero tickets.
- 2026-07-26 — READY → IN DEVELOPMENT: picked up, branch feat/T-036-review-disposition-valves
- 2026-07-26 — IN DEVELOPMENT → IN REVIEW: acceptance green: A clean, B 30-row replay passes all 3 criteria
- 2026-07-26 — IN REVIEW → DONE: review PASS: 0 blocking, 11 non-blocking all dispositioned (5 fixed inline, 2 folded into T-044, 4 noted, 0 new tickets); acceptance A re-run clean and the 30-row B replay independently re-verified; the widened inline bar (F3, decision-4 deviation) is accepted pending user ratification
- 2026-07-26 — MERGED: feat/T-036-review-disposition-valves → main (5367843, squashed)
