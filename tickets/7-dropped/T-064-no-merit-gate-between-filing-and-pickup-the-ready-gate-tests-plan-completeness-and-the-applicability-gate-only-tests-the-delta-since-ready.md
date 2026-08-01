---
id: T-064
title: no merit gate between filing and pickup: the READY gate tests plan completeness and the applicability gate only tests the delta since READY
project: pickle
depends-on: []
spawned-by: [T-063]
impact: high
complexity: high
cost: M-L
---

# T-064 — no merit gate between filing and pickup: the READY gate tests plan completeness and the applicability gate only tests the delta since READY

## Description

> **DROPPED 2026-08-01 — do not implement, do not re-file from this text.** Filed at 12:06 and
> dropped the same day, killed by the same adversarial pass it proposed to institutionalise.
> Kept as the record per `NOTES.md:20-23`, **including its errors**, marked below.
>
> **The verdict in one line: T-063 was a compliance failure, not a design gap.** This ticket
> proposed a new gate to enforce an instruction that already exists and was simply not executed.
>
> 1. **The rule already exists.** `tickets-README.md:139-140` mandates assessing every new
>    ticket against the existing backlog **"and re-grade the board."** The commit that filed
>    T-063 (`829a819`) and the commit that filed this ticket (`a3f749f`) each touched exactly
>    **two files** — the new ticket and `BOARD.md`. **Zero neighbours re-graded, twice.** The
>    filer diagnosed a missing gate while breaking the existing one in the same commit.
> 2. **The premise misreads the text it quotes.** §8's mandate is *"the ticket's own assumptions
>    **plus** the board delta since it went READY"*, tested for *"still **true**, **required**,
>    and **worth it**"* (`tickets-README.md:326-327`, `SKILL.md:177`). **"Worth it" is already
>    in the mandate**, and "the ticket's own assumptions" is not time-bounded. The claim that the
>    gate "tests what changed" is an overstatement of the source.
> 3. **The mechanism it proposed reusing has never returned a negative verdict.** ~15 recorded
>    applicability-gate runs across the tree: `clean`, `proceed`, `proceed-with-corrections`,
>    `0 blocking`. **Zero DROPs, zero route-backs.** This ticket's own "must not be a rubber
>    stamp" section (below) therefore condemns its own proposal.
> 4. **Its one concrete trigger is anti-correlated with the harm.** It nominates *filed-from-chat*
>    as the high-yield population. All three cases of real waste in the project's history —
>    **T-060** (a refinement session paid, then dropped), **T-062** (built, reworked, reviewed,
>    then reverted), **T-059** (shipped, 0 adopters) — carry `source: pickle ticket new`. The
>    trigger fires on **none** of them, and would have fired on T-063, which cost 2h46m and was
>    caught for free.
> 5. **It fails the §5 promotion test it cites.** T-045 already owns backlog-pressure valves, sits
>    at `low-medium`, and has been refinable for days with its measurement gate satisfied. A
>    `high`-graded sibling on the same theme will not be scheduled either.
> 6. **Refinement demonstrably does re-test day-one premises**, contra bullet 2 of the gap
>    argument: T-036's refinement produced a section titled *"Correction to the measurement this
>    ticket was filed on"* and found **both halves of its founding claim wrong**.
>
> **The one surviving item — recorded in `NOTES.md`, deliberately not re-filed as a ticket:**
> §8 is *headed* "a **freshness** check" with a rationale purely about aging, which has collapsed
> the merit clause in practice (0 negative verdicts in 15 runs; T-062's History records the gate
> as "confirmed **against the current tree**"). Two sentences would fix it: state that the scope
> includes assumptions **that were wrong at filing**, and that **DROP is a legal verdict**
> (`1-to-do`/`2-ready` → `7-dropped` is already legal per `move.go:31-38`, so no new mechanism).
> Sweep it in whenever §8 is next edited — T-022 is in those files.
>
> **What actually works, and needs no ticket:** a human asking for an adversarial pass when they
> smell one. 2 for 2 on this backlog, costs nothing when unused, needs no schema and no payload
> bump. Automating it was the error.
>
> **Error in the text below:** the claim that the applicability gate tests only "what changed"
> is wrong, per point 2 above — the gate's stated scope already includes merit.

Nothing in the flow systematically asks **"should this exist at all?"** between the moment a
ticket is filed and the moment it is picked up. Each gate that looks like it should is
scoped elsewhere:

- **The READY gate** (`tickets-README.md:190-213`) has seven items and every one tests whether
  the plan is *executable* — branch named, prerequisites stated, decisions listed, tasks
  concrete, acceptance test runnable, docs step, finish step. **None tests merit.** A ticket
  can satisfy all seven and be a bad idea.
- **The refine procedure** (`SKILL.md` *refine a ticket*) is judged by whether it produces a
  READY ticket. Its step 2 re-verifies the Description against the *current* child — that
  catches assumptions the tree invalidated, not premises that were wrong on day one. Sunk cost
  also accrues *during* refinement: once the plan is written it has been paid for.
- **The pickup applicability gate** (§8, `:324-333`) is the flow's one genuinely adversarial
  check, and deliberately so — spawned sub-agent, "free of the implementer's sunk-cost bias",
  each assumption still "true, **required**, and **worth it**". But its mandate is explicitly
  *"the ticket's own assumptions plus the board delta since it went READY"*. It tests **what
  changed**. A ticket that was unjustified at filing and is equally unjustified at pickup
  passes it, because nothing changed.
- **§5's promotion test** guards tickets *spawned from findings*. It does not touch a ticket
  filed from a chat message.

So the gap is specifically: **a ticket filed from chat, refined competently, is never re-tested
on merit before it consumes a development slot.**

### The evidence this is real, not theoretical

**T-063** (`spawned-by`, now the case study) was filed from a chat exploration on 2026-08-01. It
was filed *carefully*: it had rejected two weak halves of the originating proposal with correct
reasoning, verified its interaction with decision D1 and the audit's fresh-render assertion,
catalogued its own design questions, and written the counter-position and "dropping this is a
legitimate outcome" into its own Description. By the standards of this backlog it was a
well-formed ticket, and it would have refined cleanly into a READY plan.

An explicitly adversarial pass — a fresh sub-agent briefed to prosecute the ticket from first
principles, not to check the delta — killed it in one round on measurements nobody had taken:

- The pickup queue is **READY**, not TO DO (`review-protocol.md:192` — "the next item in
  `BOARD.md`'s READY section"). T-063 optimised the ordering of `1-to-do/`. Across all 114
  revisions of `tickets/BOARD.md`, READY has **never exceeded 2 rows** and is empty in the
  overwhelming majority. The queue it improved is not the queue anyone picks from.
- The precedent it cited in support — T-059's `family:` — has **0 adopters across 63 tickets**
  (`rg "^family:" tickets/*/*.md` → 0), four days after shipping, through exactly the 7-way tie
  it was built to break. The precedent argues against ordering machinery, not for it.
- Half of it (de-rank blocked tickets) applies to **0 tickets**: one non-terminal ticket carries
  a `depends-on` (T-013 → `[T-004]`) and T-004 is done *and* merged.
- Its motivating claim, "nothing reads `cost`", is **false in its own sentence** — `cost` is
  rendered in every TO DO/READY row (`board.go:105`) precisely so a human reads it.

None of that required new information. All of it was available when the ticket was filed and
would have been equally available at every subsequent gate. **The flow had no step that would
have asked.** The cost of the miss was small only because the challenge happened to be
requested; had it not been, the ticket would have consumed a refinement session and a
development slot.

Note also the second-order point: T-063 was *itself* the product of the same gap in reverse —
its original proposal contained a stored-score-plus-trigger design that survived until
someone challenged it. Challenge works when it happens. It just does not happen on a schedule.

### What a solution must not be

- **Not a rubber stamp.** A checklist item ("is this worth it? yes") is worse than nothing: it
  launders the omission. Whatever this becomes must produce *findings with evidence*, like the
  applicability gate does, and be capable of returning DROP.
- **Not a bias amplifier.** The check cannot be run by the party that filed or refined the
  ticket. The applicability gate already solved this with a spawned, briefed sub-agent
  (§8, `:326`); this should reuse that mechanism rather than invent one.
- **Not a tax on every ticket.** A full adversarial pass on all 18 TO DO tickets is
  unaffordable and would itself be low-value. Where the gate fires is the central design
  question.
- **Not a spawn engine.** Its findings must take §5's four dispositions with the same default
  (`noted`), or a merit gate becomes the largest single source of new tickets. §8 already warns:
  "A gate that files a ticket per observation converts every pickup into backlog growth."

### Design questions any plan must answer

1. **Where does the check live?** Candidates, not mutually exclusive: (a) an **eighth READY gate
   item** — a recorded merit justification with evidence, which makes it a refinement
   deliverable; (b) **widen the applicability gate's mandate** from "the delta since READY" to
   include a first-principles pass — cheapest in text, but it moves the check *after* the
   refinement cost is sunk; (c) a **new explicit step between filing and refinement** ("challenge
   ticket T-NNN"), which is where T-063's challenge actually happened and where it was cheapest;
   (d) leave the flow alone and make it a **CLI-surfaced prompt**.
2. **What fires it?** Every ticket, or only ones meeting a trigger (age since filing, `cost`
   ≥ L, filed-from-chat rather than from a finding, impact ≥ high)? The T-063 case suggests
   *filed from chat* is the high-yield population, since finding-born tickets already pass §5's
   promotion test.
3. **Is it a `pickle` command or purely payload text?** The flow's judgement steps are
   deliberately skill text, not automation — the CLI does "the deterministic mechanics… not the
   judgement" (`SKILL.md` intro). A merit gate is judgement, which argues for payload-only. But
   a payload-only gate has no enforcement, exactly like the READY gate — which works, so this
   may be sufficient. Decide deliberately; do not add a command by reflex.
4. **What does DROP look like mechanically?** `1-to-do → 7-dropped` is already legal and
   reason-required (`move.go:31-38,43-50`), and dropped tickets keep their full analysis as the
   record (`NOTES.md:20-23`). So the routing likely needs no new mechanism — verify that.
5. **How is the verdict recorded?** The applicability gate records findings in History; a review
   records them in `## Review`. A merit challenge produces a findings table too. Reusing
   `## Review` for a pre-implementation challenge may be a category error, or may be exactly
   right. Decide.
6. **Does this make refinement cheaper or more expensive overall?** The honest case for it is
   that one cheap adversarial pass *avoids* a refinement session. That is a measurable claim
   and should be measured, not asserted.

### Soft couplings

- **T-063** (`spawned-by`) — the case study, and expected to be in `7-dropped/` as the record.
- **T-045** — the per-child TO DO backlog cap is a crude merit forcing-function ("at cap, filing
  requires dropping something"). This ticket gates merit *directly* rather than bounding
  quantity, so the two are alternatives to weigh against each other, not complements. T-045 is
  measurement-gated; its measurement data (the `## Review` disposition columns T-036 made
  mandatory) is partly the same data this ticket needs.
- **T-036** (done) — ratified the four dispositions this gate's findings must reuse.
- **T-056** — its work area 5 is the ranking question T-063 was spawned to answer; T-063's
  verdict lands there whatever happens to this ticket.
- Changes to `tickets-README.md` §4/§8 or `SKILL.md` are **payload text**, so they ship to every
  installed workspace and bump `payload_version`; the `AGENTS.md` marker block and its golden
  file (`internal/install/testdata/markerblock.golden`) may need updating if the trigger list
  changes.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-01 — created (TO DO). source: chat — observed while asking whether refinement would
  challenge T-063 at all; an adversarial pass then killed T-063 on evidence available at filing
  time, which no gate in the flow would have asked for
- 2026-08-01 — TO DO → DROPPED: compliance failure, not a design gap: tickets-README.md:139-140 already mandates the assessment; the gate it would reuse has 0 rejections in 15 runs; its trigger hits 0 of 3 real waste cases
