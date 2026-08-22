---
id: T-112
title: review protocol has no bias-mitigation step for a same-session implementer reviewing their own ticket
project: pickle
depends-on: []
spawned-by: [T-050]
impact: medium
complexity: low
cost: S
---

# T-112 — review protocol has no bias-mitigation step for a same-session implementer reviewing their own ticket

## Outcome

After this ships, the review protocol says when a review must hand its audits to an independent
reviewer — the case where the agent auditing the branch is the same one that just wrote it — and
every review records who actually ran them. Reviews stop depending on the reviewer privately
remembering to guard against reviewing their own work, and a reader can tell an independent review
from a self-review by reading the ticket.

## Description

The flow already recognises implementer bias once: rules §8's pickup applicability gate requires
the plan's assumptions to be "re-verified against the current target child-project by a spawned,
**unbiased** sub-agent... free of the implementer's sunk-cost bias", run on every pickup, before a
READY ticket enters `3-in-development/`. That clause exists precisely because the agent about to
implement a ticket is the worst-positioned party to judge whether its own plan still holds.

The review protocol (`resources/review-protocol.md`, driving "validate ticket T-NNN" and "rework
ticket T-NNN") has no equivalent. Nothing in its steps 1-9, nor in the rules' §5 findings
machinery, asks whether the agent about to audit implementation/quality/consistency/docs is the
same agent — in the same session, with full memory of every design choice it just made — that
authored the code under review. The bias this omits is at least as strong as the one §8 already
names: a reviewer who just wrote the code is primed to read its own prose as self-evidently
correct, exactly the sunk-cost condition §8 exists to guard against at pickup.

Concretely, in T-050: one continuous session implemented the guardrail fix, then immediately ran
"review ticket T-050" over its own work, found one real blocking finding (a shipped comment that
failed the project's own foreign-workspace test), reworked it, and then re-reviewed its own
rework and closed the ticket — all without the flow ever prompting a consideration of whether that
self-review setup was appropriate. It happened to catch a genuine defect this time; that is one
data point, not evidence the setup is sound. Nothing stops the same session from missing a defect
in its own reasoning precisely because it is the same reasoning.

### What refinement found: this is an uncodified practice, not a new idea

The filing framed this as a gap to be filled by inventing a mitigation. Surveying the review
history says otherwise, and the correction matters enough to restate the ticket around it:
**delegating a review's audits to an independent sub-agent is already established practice here —
it is simply absent from the written protocol, so it happens or not depending on whether the
reviewing agent thinks of it.**

Of the 60 tickets in `6-done/` carrying a findings table, **10 record that their review audits were
run by an independent sub-agent** — T-018, T-030, T-046, T-051, T-054, T-080, T-084, T-089, T-090,
T-096 — and several state the bias rationale outright, unprompted by any rule:

- T-030: the audit went to a "**sub-agent** rather than the implementer, since the implementer
  wrote the code under review";
- T-084: "cross-checked by an independent sub-agent (the rework's author reviewing their own...)";
- T-096: "handed to a sub-agent with no stake in" the outcome, "briefed adversarially and told not
  to defer to the ticket's" own framing;
- T-051, whose *scoped re-review* says "audit **again** delegated to a fresh sub-agent, every
  finding re-verified by hand" — note both halves: delegate, then verify, rather than trust.

So the practice is proven, repeatedly, by the reviews that reached for it. What is missing is the
rule that makes it reliable rather than dependent on the reviewer's memory. T-050 is the
counter-example that shows the difference is real and unflagged.

**The other ~50 reviews are silent — and silence is ambiguous**, which is the second, separable
half of the defect. The protocol gives a review nowhere to record *who* ran the audit, so a reader
cannot tell an independent review from a self-review by reading the ticket. Absence of a sub-agent
mention is not evidence a review lacked one. That makes the practice unauditable in exactly the
archive that is supposed to be the project's memory.

The two halves are complementary, not alternatives: **delegation** mitigates the bias,
**recording** makes whether it happened legible after the fact.

### The guidance already exists — in the manual, which the agent does not execute

The sharpest version of the defect. `docs/user-manual/concepts/agent-session-workflow.adoc`
already prescribes, in its "pattern mapped to the procedures" table, exactly the handoff that did
not happen:

| step | session | suggested tier | stated reason |
|---|---|---|---|
| *Validate / review a ticket* | **New session** | **Heavier** | "Severity-and-disposition calls benefit from both **independence from the implementer** and stronger judgement." |
| Commit / push / open the MR | Same session as the review, **switch tier** | Lighter | "Mechanical once the verdict is reached — no reason to keep paying the heavier tier's rate for it." |

The same file states it again in prose (`:26-28`): the protocol's load-context step "is designed
to work from a cold session, **independent of whoever implemented the ticket**."

So the project's own documentation already says a review should be a new session at a heavier tier,
*for the independence reason this ticket is about*, and already says to drop back down for the
mechanical publish steps afterwards. **None of that appears anywhere in the skill payload** — not
in `review-protocol.md`, not in `SKILL.md`'s validate procedure, not in the rules. The guidance
lives exclusively in the human-facing manual, and the party that would act on it is the agent,
which reads the payload. An agent following the skill correctly, to the letter, will never suggest
the handoff, because nothing it reads mentions it.

That is why this failed silently rather than loudly: nothing was violated. The rule was in the
wrong artifact for the actor expected to apply it.

### Decisions taken at refinement (user-confirmed)

1. **Trigger:** delegation is mandatory only when the reviewing agent authored the branch under
   review in the same session — mirroring the pickup gate's rationale, and spending nothing when
   the reviewer had no hand in the work.
2. **Recording:** every review records who ran the audits, independent or not, via an always-on
   checklist line.
3. **Delegation boundary:** the audits are delegatable; classification, dispositions, moves and
   the approval gate stay with the orchestrating reviewer, and every delegated finding is
   re-verified by hand before being recorded.
4. **Session/tier handoff:** the protocol surfaces the manual's existing recommendation at both
   boundaries — entering review (new session, heavier tier, for independence) and leaving it
   (drop tier for the mechanical publish steps) — so the agent can offer it instead of the human
   having to know to ask.

### Explicitly out of scope

- **Machine-enforcing the recording line.** Making `pickle board audit` require it would mean a new
  required row in §7's gate-table data and Go changes in `internal/audit` — a materially bigger
  change than the prose rule, and premature before the prose rule has any usage behind it. If the
  line proves to be routinely skipped, that is the moment to file it.
- **The docs-readability reviewer (step 4b).** Already a separate, independent reviewer with its
  own optionality rules; untouched.

This is a **skill-payload** change (`skill/resources/review-protocol.md`, `skill/SKILL.md`'s
"Procedure: validate a ticket" summary, and possibly `skill/resources/tickets-README.md` alongside
§8's own clause), embedded via `assets.go`'s `all:skill` and shipped to every project that installs
brine — it must be phrased so it stands on its own for a foreign reader (no ticket ids the reader
is told to go look up, no "this repo", per `AGENTS.md`'s foreign-workspace test, enforced by
`payload_lint_test.go`), a lesson this same ticket's own provenance (T-050's rework, F1) supplied
first-hand. Note the evidence cited above — the ten ticket ids, the 60/10 counts — is exactly the
kind of material that must **not** cross into the payload: it is this project's own corpus, invisible
to the reader of an installed skill. It belongs in this ticket and in planning notes, and the
shipped rule must stand on its reasoning alone.

### Couplings

Soft coupling only (no `depends-on:`): rules §8's existing "spawned, unbiased sub-agent" clause
(`resources/tickets-README.md`) is the direct precedent and likely gets a cross-reference from
whatever this ticket adds to the review protocol, so both should read as one coherent policy
rather than two independently-invented ones.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-112-review-reviewer-independence
```

Root-path child (`project: pickle`), so tidy WIP commits into atomic ones before presenting
(Finish, below) and default to keeping that history rather than squashing.

### Prerequisite gate (hard)

None. No `depends-on:`. The one soft coupling (rules §8's existing unbiased-sub-agent clause) is
already shipped and is read, not modified in a blocking way, by this ticket.

### Confirmed design decisions (do not deviate without asking)

1. **The rule is added as a new `## 0.` step, and no existing step number changes.** The protocol's
   own intro states that project review addenda are "keyed to this protocol's step numbers", so
   renumbering steps 1–9 would silently invalidate every addendum any installed project has
   written against them. A new step 0 is additive and safe; renumbering is not.
2. **Delegation is required only when the reviewing agent authored the branch under review in the
   same session.** A reviewer with no hand in the work is already independent, and spending a
   sub-agent to prove it buys nothing. This mirrors the pickup gate's trigger, which fires on the
   implementer's own sunk-cost bias rather than on every pickup regardless of who is acting.
3. **The audits are delegatable; judgement is not.** Steps 2–4a may be handed to the independent
   reviewer. Classification and severity, the four dispositions, the ticket moves, and the
   approval gate stay with the orchestrating reviewer — delegating those would replace the
   reviewer rather than de-bias it.
4. **A delegated finding is re-verified by hand before it is recorded.** An independent reviewer
   has no stake, and equally no context: it will report things that are wrong. Delegation buys
   independence, not accuracy, and the protocol must say so — otherwise the rule trades one
   failure mode for a worse one, where unverified sub-agent output enters the findings table with
   a reviewer's authority behind it.
5. **When the host cannot provide an independent reviewer, the step degrades to a recorded
   conscious skip — it never blocks the review.** This mirrors step 4b's already-shipped shape for
   the docs-readability reviewer exactly, including recording the skip in the ticket's `## Review`.
   A mandatory step that a bare harness cannot satisfy would make the protocol unfollowable there,
   and an unfollowable rule is ignored wholesale rather than partially.
6. **The recording line is prose in the checklist, not a machine-checked gate.** No new required
   row in the §7 gate tables and no `internal/audit` change: the enforcement question is
   deliberately deferred until the prose rule has usage behind it (see *Explicitly out of scope*).
7. **The payload states the rule on its own reasoning, citing no ticket id, count, or corpus.**
   Everything in this ticket's Description that argues *from this project's own history* — the ten
   ticket ids, the 60/10 counts, the manual's table — is evidence for the decision, not material
   for the shipped text. A reader of an installed skill has none of it. The rule must read as a
   plain statement of why an author is a poor auditor of their own work.
8. **The manual and the payload cross-reference each other rather than restating the rule twice.**
   The manual's session/tier table stays the detailed treatment; the payload states the rule and
   points at the concept, so the two cannot drift into two different policies.

### Tasks

#### Task 1 — add step 0 to the review protocol
In `skill/resources/review-protocol.md`, insert a new `## 0. Reviewer independence — who runs the
audits` section immediately **after** the `---` at `:64` and **before** `## 1. Load context`
(`:66`). It states, in payload-safe prose (decision 7):
- why an author audits their own work poorly — the work reads as obviously correct to whoever just
  decided it was;
- the trigger (decision 2): when the reviewing agent wrote the branch in this same session,
  delegate the audits;
- what "independent reviewer" means operationally — spawned fresh, no memory of authoring the
  code, briefed adversarially and told to find defects rather than confirm the work;
- the boundary (decision 3) and the hand-verification requirement (decision 4);
- the degradation clause (decision 5), phrased like step 4b's;
- the session/tier handoff (decision 4 in the Description's confirmed list): entering review is a
  natural new-session boundary at a heavier tier, and the mechanical publish steps after the
  verdict do not need that tier — offer the handoff rather than assuming it.

#### Task 2 — add the recording line to the review checklist
In the same file's `### Checklist (paste into the ticket's `## Review` section)` block (`:293`
onward), add one line, placed first so it is answered before the audits it qualifies:
`- [ ] Reviewer independence settled (step 0): audits run independently, delegated, or a recorded
conscious skip — name which`.

#### Task 3 — mirror the rule in the skill summary
In `skill/SKILL.md`'s *Procedure: validate a ticket* (`:260`), add one sentence to the preamble
(before the numbered list, so no item is renumbered) directing the reader to settle step 0 before
auditing. Keep it to the rule plus the pointer — the protocol is the full text and the summary
must not state a rule the protocol does not (a past review found exactly this class of
summary-versus-full-text divergence in the pickup-gate clause, so it is a known failure mode here).

#### Task 4 — cross-reference from the rules' pickup gate
In `skill/resources/tickets-README.md` §8, immediately after the existing "spawned, unbiased
sub-agent" pickup-gate sentence (`:531-535`, the phrase itself on `:533`), add one sentence noting the same principle governs
the review that follows, pointing at `resources/review-protocol.md` step 0. Rationale: §8 is
currently the only place the flow names implementer bias, so a reader of the rules reasonably
concludes it is a pickup-only concern. One clause prevents that misreading.

#### Task 5 — connect the manual to the new step
In `docs/user-manual/concepts/agent-session-workflow.adoc`, add a cross-reference from the
*Validate / review a ticket* row's reasoning (or immediately beneath the table) to the protocol's
new step 0, so the manual's "new session, heavier tier, independence from the implementer"
recommendation and the payload's rule are visibly one policy (decision 8). Do **not** restate the
rule's content here.

#### Task 6 — CHANGELOG entry
Add an `[Unreleased]` → `Added` entry in `CHANGELOG.md` describing the new review step and the
recording line, in the established style (bold lead sentence, then the qualification, ticket id in
trailing parens). This is shipped payload behaviour every installed project sees on its next
`pickle upgrade`, which is the bar the entry convention exists for.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

All clean. `just test` is load-bearing here, not a formality: it runs `payload_lint_test.go`'s
`TestPayloadSpeaksToAForeignReader` over the embedded payload, which is what mechanically enforces
decision 7 for the two `skill/` files this ticket edits.

Then, specifically:

1. **No step was renumbered** (decision 1) — `grep -n '^## [0-9]' skill/resources/review-protocol.md`
   must show the new `## 0.` followed by `## 1.` through `## 9.` with every existing number
   unchanged, and `## 4a.`/`## 4b.` still present.
2. **No ticket id, count or corpus claim entered the payload** (decision 7) —
   `grep -nE 'T-[0-9]|this repo|the corpus' skill/resources/review-protocol.md skill/SKILL.md
   skill/resources/tickets-README.md` returns nothing that this branch added (the pre-existing
   legitimate provenance tags and grammar examples stay; compare against `git show main:<path>`
   rather than assuming a clean grep).
3. **The rule ships, not just exists locally** — install into a throwaway dir per `AGENTS.md`'s
   self-modify policy and confirm the installed skill carries step 0 and the checklist line:
   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree --project demo
   grep -c 'Reviewer independence' .agents/skills/brine/resources/review-protocol.md   # expect 2 (step + checklist)
   ```
4. **Docs build clean and the cross-reference resolves** — `just docs-check` passes and the new
   reference in `agent-session-workflow.adoc` points at a real target.

### Docs update (mandatory when user-facing)

User-facing, in two places, both already named as tasks rather than deferred:
`docs/user-manual/concepts/agent-session-workflow.adoc` gains the cross-reference to the protocol's
new step (Task 5), and `CHANGELOG.md` gains an `[Unreleased]` entry (Task 6). No other manual page
enumerates the protocol's steps — verified at refinement: the only other references are to step 4b
and to §5's findings-table skeleton, neither of which this ticket touches. Run the
`docs_readability` reviewer over both changed files during review (protocol step 4b).

### Finish (mandatory)

1. Acceptance test green, including all four specific checks above.
2. Docs updated (Tasks 5 and 6) and `just docs-check` clean.
3. Write a summary: the five files touched, the step-0 text as shipped, and confirmation that no
   existing step number moved.
4. Suggested commit message:
   ```
   feat(skill): require an independent reviewer when the reviewer wrote the branch (T-112)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-112 in-review --reason "acceptance green"`.

**Note for whoever picks this up:** this ticket's own review is the first natural place to apply
the rule it ships. Doing so — and recording it — is the cheapest available proof the step is
executable as written rather than merely well-phrased.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: review: T-050's implement/review/rework/re-review cycle
  ran in one continuous session with no consideration of self-review bias; the human asked why
  the flow didn't offer a fresh reviewer, which surfaced the gap between rules §8's pickup gate
  (explicitly unbiased) and the review protocol (silent on it)
- 2026-08-22 — refined: Description re-verified and twice reframed by what the survey actually
  found. First, this is not a new idea at n=1: 10 of the 60 reviewed tickets in `6-done/` already
  delegated their review audits to an independent sub-agent (T-018, T-030, T-046, T-051, T-054,
  T-080, T-084, T-089, T-090, T-096), several naming the bias rationale unprompted — so the ticket
  codifies an inconsistently-applied practice rather than inventing one, and the ~50 silent
  reviews are ambiguous only because the protocol offers nowhere to record who audited. Second,
  and sharper: `docs/user-manual/concepts/agent-session-workflow.adoc` **already** prescribes "new
  session, heavier tier" for review, giving "independence from the implementer" as the reason, and
  a tier drop for the mechanical publish steps after — but none of it appears in the skill payload,
  so the agent expected to act on it never reads it. That is why the lapse was silent rather than a
  violation. Four decisions confirmed with the user (trigger = same-session author only; always-on
  recording line; audits delegatable but judgement and hand-verification retained; surface the
  session/tier handoff at both boundaries); four more taken at refinement and written into the plan
  (no renumbering of existing steps, since addenda are keyed to them; step-4b-style degradation to
  a recorded conscious skip; prose-only recording, no gate-table change; payload cites no ticket id
  or corpus). All plan line references verified against the current tree. **Grade deliberately
  unchanged at medium/low/S** — medium-high was considered and rejected: the closest higher-graded
  precedent corrected payload that stated wrong things to every installed project, whereas this
  adds missing guidance to payload that is merely silent, which is less urgent. Kept as one ticket
  (six tasks, five files, one policy — splitting would leave the tree stating two different rules
  mid-flight).
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — READY → IN DEVELOPMENT: picked up
