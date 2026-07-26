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

**2. `skill/resources/review-protocol.md` — the procedure.**

- §0 scope block, lines 21–24: replace "— never inline drift" with the bounded inline path,
  pointing at rules §5 for the vocabulary. Keep "a review does not re-implement the ticket".
- §5, lines 99–117: rewrite the non-blocking bullet to route per rules §5, with the findings
  table gaining a **required `disposition` column**. Delete the line-116 "trivial cosmetic spec
  typos" carve-out — it is subsumed by `fixed inline` and its survival would give two
  overlapping rules for the same case.
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

**5. `internal/install/install_test.go` — pin the two decisions mechanically.**

Add `TestPayloadDispositionVocabulary`, using the existing `payloadRoot()` idiom (`:15`) to read
`skill/resources/*.md`. Assert exactly two things — the *decisions*, not the prose:

- the string `never inline drift` appears **nowhere** in the payload (guards decision 4 against
  a future revert);
- each of the four disposition tokens appears in `tickets-README.md`, **and none of the four
  lists is restated** in `review-protocol.md`/`SKILL.md`/`TEMPLATE.md` — approximated as: those
  three files each contain the reference string `rules §5` and do **not** contain all four
  tokens.

Keep it to those two assertions. A test that pins wording rather than decisions would be
re-writing the diff, and this repo already carries enough drift-guard debt (T-040, T-042).

### Acceptance test

**A. Mechanical — must be green.**

```
just build && just test && just lint
./pickle board audit          # expect: 0 error(s), 0 warning(s)
./pickle doctor               # expect: clean; skill dir still a symlink, untouched
```

Expected: `go test ./...` passes including the new `TestPayloadDispositionVocabulary`;
`gofmt -l` and `go vet ./...` clean. `board audit` ticket count unchanged (no ticket moves in
this task list beyond T-036's own).

**B. Retroactive replay — the real test of "precise enough that two reviewers agree".**

The historical findings tables in `6-done/` are frozen data. Re-classify, against the **new**
rules §5 only, every non-blocking finding in the four reviews that recorded per-finding
dispositions:

| review | findings |
|---|---|
| T-008 | 2 |
| T-024 | 13 |
| T-029 | 7 |
| T-030 | 6 |
| **total** | **28** |

Pass criteria, all three required:

1. **Every one of the 28 maps to exactly one disposition** — zero findings ambiguous between
   two. Ambiguity is the failure mode the bar exists to prevent; a single ambiguous finding
   means the wording is not done.
2. **All 9 inline fixes** (T-005 ×1, T-008 ×1, T-024 ×4, T-030 ×3) are **authorized** by the new
   bar. Under the old rules only the T-005 one was.
3. **The new rules produce no more new tickets than were actually filed** for those four
   reviews (actual: T-015, T-029, T-030, T-031, T-032, T-038 = 6).

Record the 28-row replay table in the ticket's `## Review` section as the evidence. If criterion
1 fails, the fix is wording — not an exception list.

### Docs update

The payload **is** the docs for this change; tasks 1–4 are the documentation. Confirmed by
inspection that no other surface documents review dispositions:

- `README.md` — no matches for blocking/non-blocking dispositions; **no change**.
- `PLAN.md` — historical record; **no change** (T-024 N12 precedent).
- `AGENTS.md` marker block — points at `resources/review-protocol.md` by path only, states no
  disposition rules; **no change**, and `markerBlock` (`install.go:535-614`) needs no edit, so
  `testdata/markerblock.golden` is untouched.

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

<!-- empty until IN REVIEW -->

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
