---
id: T-122
title: Verify the docs-readability reviewer's quoted text before presenting a suggestion
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-122 — Verify the docs-readability reviewer's quoted text before presenting a suggestion

## Outcome

After this ships, a reviewer running the optional docs-readability pass checks each suggestion's
quoted "current text" against the file before presenting it, and the shipped reviewer prompt tells
the reviewer to quote verbatim or say nothing. A run whose quotes match nothing in the tree is
discarded and recorded instead of applied.

## Description

Step 4b of `review-protocol.md` defines the readability pass, names the reviewer, and gives the
invocation syntax for three hosts. It says **nothing about verifying the output.** Grep the
protocol for `fabricat|verbatim|quote|hallucin` and the only hits are re-running an acceptance
test and copying a cost estimate — nothing about the reviewer's own claims.

That matters because this reviewer is an LLM asked to quote source text, which is a failure mode
with a known shape. A reviewer following step 4b as written presents every suggestion for
approval, and applying one means editing prose to match a passage that may not exist. Every
project that installs the skill and runs 4b is exposed: it is a property of the tool the protocol
ships instructions for, not of any one project.

### What refinement confirmed: the protocol already states this rule one step up

Step 0 (reviewer independence) ships exactly the missing sentence, for the analogous case of a
delegated sub-agent reviewer:

> **Verify before recording.** An independent reviewer has no stake in the outcome, but equally no
> context, so expect it to report things that are wrong. Re-verify every delegated finding by hand
> before it enters the findings table — delegation buys independence, not accuracy.

4b is the same shape — an outside agent reporting text it was not made to prove — and carries no
equivalent. So this is a consistency gap in the protocol rather than a new policy, which is what
holds it to a prose-only change: the wording, the reasoning and the "record it" habit are all
already established next door.

The reviewer prompt is the other half. `docs-readability.prompt.md` already asks for a
`current text` field in every suggestion row, and never says that field must be a verbatim copy.
Constraining the generator there and verifying at the consumer in 4b are one fix in two places,
and the prompt is a single shipped file read by **both** installed backends (the pi
`docs_readability` tool and the opencode `docs-readability` subagent), so one line covers both.

### Evidence — and why none of it ships

In one recorded run in a downstream workspace, all 8 suggestions quoted text that existed nowhere
in the tree, and the rule now lives in that workspace's own overarching addendum, protecting that
project only. That is the evidence for the decision, **not** material for the shipped text: a
reader of an installed skill has no access to it, and `payload_lint_test.go` mechanically rejects
counts and workspace claims of exactly that shape. The payload states the failure mode on its own
reasoning (plan decision 8).

### Decisions taken at refinement (user-confirmed)

The four open questions this ticket carried are settled and written as numbered decisions in the
plan: the rule lands in step 4b **and** in the shared reviewer prompt (2); a failed quote means the
suggestion is discarded, not repaired (3); a mostly-failing run is discarded on judgement, with no
numeric threshold (4); the content-versus-wording rule ships in the same step (5); the count is
recorded in `## Review` prose and never as a findings-table row, 4b staying non-blocking (6); and
the checklist gains no twelfth line — the existing 4b line is extended (7).

### Explicitly out of scope

- **Any machine check.** Nothing in `internal/audit` learns about fabricated quotes; 4b is optional
  and produces no findings, so there is no artifact to check.
- **`skill/SKILL.md` and `tickets-README.md`.** Neither describes 4b's internals — SKILL.md names
  the prompt file in its resources list and nothing more — so neither is edited, and the rule has
  exactly one statement.
- **The backend definitions** (`agents/pi/extensions/docs-readability.ts`,
  `agents/opencode/opencode.jsonc`). They configure invocation, not output discipline, and both
  already read the shared prompt this ticket edits.

### Couplings

T-123 edits the same file (`skill/resources/review-protocol.md`) in a different section (step 7).
They are independent and can land in either order; whichever lands second rebases.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-122-verify-readability-quotes
```

Root-path child (`project: pickle`), so tidy WIP commits into atomic ones before presenting
(Finish, below) and default to keeping that history rather than squashing.

### Prerequisite gate (hard)

None. No `depends-on:`. The one soft coupling (T-123, same file, different section) imposes no
ordering — if it lands first, rebase before starting.

### Confirmed design decisions (do not deviate without asking)

1. **The rule is added inside step 4b; no step is renumbered and no new step is created.** The
   protocol's intro states that project review addenda are "keyed to this protocol's step
   numbers", so renumbering silently invalidates every addendum an installed project has written.
   The `## 4b.` heading text is also left alone — the rule is a duty within the step, not a new
   subject.
2. **The fix lands on both sides: the reviewer is told to quote verbatim, and the reviewer's caller
   is told to verify.** `skill/resources/docs-readability.prompt.md` gains the generator-side
   constraint, step 4b gains the consumer-side verification. Constraining the generator alone
   trusts the failure mode not to recur; verifying alone leaves an easy improvement unmade. The
   prompt is one shipped file read by both installed backends, so the generator-side half costs
   one edit for every host.
3. **A suggestion whose quoted "current text" is not found verbatim is discarded, not repaired.**
   Repairing means guessing which real passage the reviewer meant, which is the same invention the
   rule exists to stop — now with a reviewer's authority behind it.
4. **A run that mostly fails verification is discarded whole and re-invoked, on judgement.** No
   numeric threshold: 4b never blocks, and a fraction invites arithmetic on a three-suggestion run.
5. **The content-versus-wording rule ships in the same step.** A suggestion that deletes emphasis
   carrying meaning, drops a documented path, or swaps a precise term is a content edit whatever
   the reviewer labels it, and is discarded. It is phrased as checking that the reviewer honoured
   the hard constraints its own prompt already states — not as a new prohibition.
6. **The discarded-as-fabricated count is recorded in the ticket's `## Review` prose, never as a
   findings-table row, and 4b remains optional and never blocking.** 4b's suggestions are
   explicitly "not findings"; a fabricated one is a discarded suggestion, not a defect in the
   ticket under review. Recording the count is what keeps the pass auditable — the same argument
   step 0 makes for naming which reviewer ran the audits.
7. **The checklist gains no new line; the existing 4b line is extended.** One optional step does
   not warrant a twelfth checklist row, and a separate row would imply a separately-answerable
   obligation when the verification is inseparable from running the pass at all.
8. **The payload states the failure mode on its own reasoning — no counts, no workspace names, no
   ticket ids.** Everything in this ticket's Description that argues from a specific run is
   evidence for the decision, not shipped text. `payload_lint_test.go` enforces the mechanical
   part of this; the judgement part is the author's.

### Tasks

#### Task 1 — the verification duty, in step 4b
In `skill/resources/review-protocol.md`, add a paragraph to `## 4b.` after the
"genuinely optional and never blocks a review" paragraph (currently ending `:193`, immediately
before `## 5.` at `:195`). It states, in payload-safe prose (decision 8):

- **verify every suggestion's quoted "current text" against the file before presenting it** —
  locate the quoted string in the file the suggestion names;
- a suggestion whose quote does not match verbatim is **discarded, not repaired** (decision 3);
- a run where most suggestions fail this way is **discarded whole and re-invoked** rather than
  salvaged (decision 4);
- **why**: the reviewer is a model asked to quote source text, and an unverified quote turns
  "polish the wording" into "edit prose to match a passage that does not exist". Point at step 0's
  *"delegation buys independence, not accuracy"* as the same rule for the analogous case, so the
  two read as one policy rather than two habits;
- **record the number of suggestions discarded as fabricated in the ticket's `## Review`**, next to
  where the step's conscious skip would be noted — prose, not a findings-table row (decision 6).

#### Task 2 — the content-versus-wording rule, same step
In the same paragraph block, add the second rule (decision 5): discard suggestions that change
content rather than wording — deleted emphasis that carries meaning, a dropped documented path, a
swapped precise term — noting that the reviewer's own prompt already forbids these, so the check is
that it honoured its constraints, not a new prohibition. Keep it to two sentences; the prompt file
is the authoritative list and must not be restated here.

#### Task 3 — extend the checklist line
In the same file's `### Checklist (paste into the ticket's `## Review` section)` block, extend the
existing 4b line (`:341`) so the verification is part of the same tick — no new line (decision 7),
e.g.:

`- [ ] Docs-readability pass on the ticket's changed `.adoc`/`.md` files — every suggestion's
quoted text verified against the file, any fabricated ones discarded and counted — or a conscious
skip recorded (step 4b, optional)`

#### Task 4 — the generator-side constraint, in the shared reviewer prompt
In `skill/resources/docs-readability.prompt.md`, add a short paragraph after the output-shape block
(`:17-19`, the `file · anchor/section · current text · proposed text · why` row) requiring that
`current text` be copied **verbatim** from the file, long enough to locate unambiguously, and that
a passage the reviewer cannot quote exactly is **omitted rather than paraphrased or
reconstructed**. Do not touch the existing hard-constraints list (`:7-13`) — it is framed as
"never suggest a change that would …", which a quoting rule does not fit; and do not restate the
consumer-side verification here.

#### Task 5 — the manual's description of the reviewer
In `docs/user-manual/cli-reference.adoc`, in the `*The docs-readability reviewer*` paragraph
(currently `:196-205`), add one clause stating that suggestions are verified against the file
before they are presented and fabricated quotes are discarded, pointing at protocol step 4b as the
rule. One clause only — the manual describes the reviewer, the payload states the rule, and they
must not drift into two policies.

#### Task 6 — CHANGELOG entry
Add an `[Unreleased]` entry in `CHANGELOG.md` in the established style (bold lead sentence, then the
qualification, ticket id in trailing parens), naming both changed payload files
(`resources/review-protocol.md`, `resources/docs-readability.prompt.md`) and the fact that every
installed project picks the rule up on its next `pickle upgrade`.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

All clean. `just test` is load-bearing, not a formality: it runs `payload_lint_test.go` over the
embedded payload, which mechanically enforces decision 8 for both `skill/` files this ticket edits.

Then, specifically:

1. **No step was renumbered and the 4b heading is unchanged** (decision 1) —
   `grep -n '^## [0-9]' skill/resources/review-protocol.md` shows `## 0.` through `## 9.` with
   `## 4a.`/`## 4b.` present and every number as on `main`
   (`git show main:skill/resources/review-protocol.md | grep -n '^## [0-9]'` for the comparison).
2. **Both halves of decision 2 shipped** —
   `grep -c verbatim skill/resources/review-protocol.md skill/resources/docs-readability.prompt.md`
   is non-zero for each, and the protocol hit is inside `## 4b.` (not only the pre-existing
   acceptance-test and cost-estimate hits at `:134`/`:264`).
3. **The checklist gained no line** (decision 7) —
   `grep -c '^- \[ \]' skill/resources/review-protocol.md` returns the same count as
   `git show main:skill/resources/review-protocol.md | grep -c '^- \[ \]'`, and the 4b line now
   mentions the verification.
4. **The rule ships, not just exists locally** — install into a throwaway dir per `AGENTS.md`'s
   self-modify policy and confirm the installed payload carries both halves:
   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree --project demo
   grep -c verbatim .agents/skills/brine/resources/review-protocol.md          # expect ≥ 3
   grep -c verbatim .agents/skills/brine/resources/docs-readability.prompt.md  # expect ≥ 1
   ```
5. **Docs build clean** — `just docs-check` passes with the edited `cli-reference.adoc` paragraph.

### Docs update (mandatory when user-facing)

User-facing, in two places, both tasks rather than deferrals:
`docs/user-manual/cli-reference.adoc` gains the one clause (Task 5) and `CHANGELOG.md` gains an
`[Unreleased]` entry (Task 6). Verified at refinement that no other manual page describes step 4b
or the reviewer's output — the only other mentions are the payload-inventory line in
`concepts/the-flow.adoc`, the install file list, the `--agent pi` file list in
`concepts/project-structure.adoc`, and a migration note about the prompt file's path, none of which
this ticket changes.

### Finish (mandatory)

1. Acceptance test green, including all five specific checks above.
2. Docs updated (Tasks 5 and 6) and `just docs-check` clean.
3. Write a summary: the four files touched, the step-4b and prompt text as shipped, and
   confirmation that no step number moved and no checklist line was added.
4. Suggested commit message:
   ```
   feat(skill): verify docs-readability quotes before presenting suggestions (T-122)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-122 in-review --reason "acceptance green"`.

**Note for whoever picks this up:** this ticket changes only `.md` and `.adoc` prose, so its own
review runs 4b over its own changed files. Doing so — verifying each suggestion's quote and
recording the count — is the cheapest available proof the rule is executable as written rather than
merely well-phrased.

## Review

Reviewed 2026-08-25 against branch `feat/T-122-verify-readability-quotes` (`848a245`, plus the
inline-fix commit `9ecf48c` this review authored).

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the branch
  in this same session, so steps 2–4a went to an independently spawned reviewer, briefed
  adversarially and told to find defects. Every finding it returned was re-verified by hand before
  entering the table below; that re-verification rejected one finding outright (F1) and raised the
  severity of two others (F2, F4)
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files (step 4b, optional) — run
  over `skill/resources/review-protocol.md` and `skill/resources/docs-readability.prompt.md`;
  10 suggestions returned, **0 discarded as fabricated**, all 10 quotes located in the files. Run
  under this branch's own new rule, which is what exposed F2: 1 of the 10 failed byte-for-byte
  matching purely because the quoted passage wraps across a line break
- [x] Findings recorded with severity, class and disposition per the rules §5; disposition summary
  and `cost:` line beneath the table (step 5)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — T-123 patched, see below
- [x] Summary presented (step 9). **No commit message or MR presented for approval**: the verdict is
  rework, so there is nothing to publish yet — the approval gate runs when the branch is finished

**Acceptance test re-ran green** on the branch, independently: `just build`, `just test` (including
`payload_lint_test.go` over the embedded payload), `just lint`, `just docs-check` all clean, plus
all five of the plan's numbered checks — no step renumbered, `verbatim` present in both payload
files with the protocol hit inside 4b, checklist line count unchanged at 11, and a throwaway
`pickle-test` install confirming the installed payload carries both halves. Re-run again after this
review's inline fixes: still green.

**All eight confirmed design decisions honoured** (1–4, 6–8 fully; 5 in substance — see F1, which
was raised against it and rejected).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F2 | **blocking** | spec-unclear | — | "Copied **verbatim** — byte for byte" is unsatisfiable against hard-wrapped source, and collides with the one-line row shape. A quote long enough to locate unambiguously usually crosses a line break, so a literal reading discards *legitimate* suggestions as fabricated — inverting the ticket's own Outcome | Measured on this review's own 4b run: 1 of 10 quotes failed byte-for-byte matching and matched under whitespace normalisation — `review-protocol.md:202-203`, where "Record the number of\nsuggestions discarded as fabricated" wraps mid-passage; generator side `docs-readability.prompt.md:21-22` vs the single-line row shape at `:17-19` | Add a normalisation clause to both sides — compare ignoring line wrapping and leading indentation — so the check catches invention rather than typography |
| F4 | **blocking** | spec-unclear | — | "If most of a run fails this way, discard the run and re-invoke" has no exit condition. Independently this is a corner case; while F2 stands it is reachable on *every* run, so the instruction can loop without terminating | `review-protocol.md:201-202` | Bound it: if the re-invoked run fails the same way, fall through to this step's existing conscious skip |
| F3 | non-blocking | spec-unclear | fixed inline | The count's locus was specified two ways in one sentence — "in the ticket's `## Review` … alongside where a conscious skip would be noted" — but 4b records a skip *on the checklist line*, which is not prose | `review-protocol.md:202-205` as authored | Fixed in `9ecf48c`: the count is recorded on the same checklist line that carries the conscious-skip note |
| F6 | non-blocking | stale-xref | fixed inline | The manual's new clause carried a second, unitalicised `(Step 4b)` back-reference in a paragraph that already cites `*Step 4b*`, and sat after the shell-out sentence so it read as scoped to that path only | `cli-reference.adoc:205-207` as authored vs the existing citation at `:197-198` | Fixed in `9ecf48c`: clause moved adjacent to the existing citation, redundant back-reference dropped |
| F5 | non-blocking | docs-gap | noted | The pi backend's `promptGuidelines` — injected into the *calling* agent's system prompt, the text nearest the call site — still says only "present each suggestion for approval and apply the approved ones yourself", with no verification step. Pre-existing incompleteness the branch did not author or falsify, so it is outside the inline bar; recorded with its evidence for a future payload-consistency pass | `agents/pi/extensions/docs-readability.ts:118-122` **and** the live copy at `.pi/extensions/docs-readability.ts:118-122` — two copies, both stale (the independent reviewer found only the first) | Six words at both call sites: "verify each suggestion's quoted text first" |

**Disposition summary:** 5 findings recorded — 2 blocking (F2, F4, both to rework), 2 fixed inline
(F3, F6), 1 noted (F5). No follow-up ticket: F5 alone does not pass the promotion test, and there is
no second finding to batch it with.

```
cost: estimated S, actual S
```

**Delegated finding re-verified and rejected — F1.** The independent reviewer reported that the
protocol's "the reviewer's own instructions already forbid this" claim is false for two of its three
examples, because bold/italic emphasis appears nowhere in the prompt's hard-constraints list and a
documented path is covered only inside inline code. Hand-verification rejects this: the constraint
reads "change any markup structure, **including**: …", and *including* makes that sub-list
illustrative rather than exhaustive — emphasis is markup structure and is covered; a path is a
"literal identifier"; a precise term is "domain terminology", named outright. Decision 5's framing
therefore holds and no edit was made. Recorded because a rejected delegated finding is part of the
audit trail: the reviewer read the list correctly and the qualifier that defeats the reading is easy
to miss.

**Note on the verdict.** The blocking pair is the feature's own mechanism misfiring, found by
applying the rule to this very branch rather than by reading it — which is the outcome the plan's
closing note predicted would be the cheapest proof the step is executable. The scoped rework is two
clauses in one paragraph, on both sides of the same rule.

---

### Rework (2026-08-25), scoped to F2 and F4 only

On the same branch, commit `7deb400`.

- **F2 — fixed.** Both sides of the rule now define "verbatim" as tolerant of line wrapping and
  leading indentation: `review-protocol.md` ("Treat it as a **verbatim** match even when line
  wrapping and leading indentation differ… once whitespace runs are collapsed") and
  `docs-readability.prompt.md` ("the same words and punctuation, in the same order, regardless of
  how the source happens to be line-wrapped or indented"), replacing the byte-for-byte framing
  that produced the false discard.
- **F4 — fixed.** "Discard the run and re-invoke" is now bounded: "re-invoke **once**; if the
  re-invoked run fails the same way, fall through to this step's existing conscious skip rather
  than retrying indefinitely."

**Regression-tested, not just re-read.** Re-ran the docs-readability pass over both changed
payload files: 13 suggestions returned. Checked each quoted "current text" by hand under the
fixed rule (verbatim, whitespace-normalised) — **0 fabricated, 13/13 verified**, including the
exact case that broke F2 (`review-protocol.md:206-207`, "Record how many suggestions were\ndiscarded as fabricated", a quote spanning the same kind of wrapped line that caused the original
failure) and three quotes drawn from the intro's Markdown blockquote, which a naive whitespace-only
comparison still misses on the `>` continuation marker — confirming "leading indentation" in the
shipped rule is read as covering that case, not only plain paragraph wrapping. One genuine,
verified suggestion from this run (rule-then-rationale reordering of the F2 sentence) was applied
to the branch as further inline polish on text this branch owns.

> **⚠ The last claim in the paragraph above is wrong — falsified by the scoped re-review below
> (R1).** "Leading indentation" is *not* read as covering a `>` continuation marker: the shipped
> rule's operational clause collapses **whitespace runs**, and `>` is not whitespace. The three
> blockquote quotes did not pass under the shipped rule; they passed only because the ad-hoc test
> script stripped `>` markers itself. Left in place rather than rewritten, because a record that
> quietly loses its own falsified claim is worth less than one that carries the correction.

**Acceptance test re-ran green** after the fix: `just build/test/lint/docs-check` clean, and all
five of the plan's numbered checks re-verified — no renumbering, `verbatim` present in both files
with the protocol hits inside 4b (now 2, both from this fix), checklist count unchanged at 11, and
a fresh throwaway `pickle-test` install confirming both halves ship.

**Disposition unchanged** for F1 (rejected), F3/F6 (fixed inline, prior round), F5 (noted, prior
round) — nothing about them was in scope for this pass and nothing about them changed.

```
cost: estimated S, actual S
```

---

### Scoped re-review (2026-08-25) — F2 and F4 only

Against `7deb400`, plus this round's inline-fix commit `bb95025`. Reviewer independence: **delegated**
again — the reviewing agent authored the rework in the same session, so the scoped audit went to a
fresh independent reviewer. Every finding re-verified by hand before recording, which is what caught
the two below: the independent reviewer's own verdict was "return to DONE", and hand-verification
overturned it.

**Verdict: back to `5-rework/`. F2 and F4 are each only *partially* resolved.** The general rule
landed and is a real improvement; the residuals are the same two defects surviving in a narrower
form, which is precisely what a scoped re-review is for.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | **blocking** | spec-unclear | — | **Residual of F2.** The fix's operational clause is "once whitespace runs are collapsed", but a Markdown blockquote `>` or list bullet continuing a wrapped line is *not* whitespace, so it survives normalisation and sits mid-quote — reproducing F2's false discard on any quote that crosses a line break inside such a block | Measured: `"A **non-blocking** finding takes one of the four dispositions…"` fails under the shipped rule and passes only with `>` stripped; the collapsed file reads `…four > dispositions…`. **22% of `review-protocol.md`'s own lines** are blockquote (12%) or list (10%) lines, and 3 of the 13 suggestions in the rework round's own 4b run came from that surface | Stop enumerating layout artifacts — whitespace, then `>`, then bullets, then table pipes is whack-a-mole. State the rule generally: compare the **words and punctuation, ignoring layout entirely** — wrapping, indentation, and any per-line prefix the format adds |
| R2 | **blocking** | spec-unclear | — | **Residual of F4.** The bound now falls through to "this step's existing conscious skip", but that skip's stated triggers are "no reviewer configured, or the session cannot reach one" — a reviewer that is reachable and fabricates twice is neither, so F4's exit names a door that formally does not open for this case | `review-protocol.md:188-191` (skip triggers) against `:204-206` (the new fall-through) | Widen the skip's enumeration to admit output that cannot be trusted, so the exit F4 relies on actually covers the case that reaches it |
| R3 | non-blocking | docs-gap | fixed inline | The `[Unreleased]` CHANGELOG entry still described the retry as "discarded and re-invoked rather than salvaged", omitting the bound the rework added — prose this branch authored, made incomplete by this branch's own later commit | `CHANGELOG.md:14-16` before `bb95025` | Fixed in `bb95025`: names the single retry and the fall-through to a conscious skip |
| R4 | non-blocking | other | fixed inline | Cosmetic wrap regression introduced by the rework commit: one line at 113 chars against 88–90 for its neighbours | `docs-readability.prompt.md:23` at `7deb400` | Fixed in `bb95025`; the paragraph is now 88/90/97/86 |

**Disposition summary:** 4 findings — 2 blocking (R1, R2 → rework), 2 fixed inline (R3, R4). No
follow-up ticket. F1/F3/F5/F6 from the first round are untouched and their dispositions stand.

```
cost: estimated S, actual M — three review rounds on a two-file prose change; the mechanism needed
empirical testing rather than reading, and each test round found a narrower instance of the same defect
```

**Delegated verdict overturned — recorded, because it is the second time in this ticket that
hand-verification changed the outcome.** The independent reviewer classified R1 as a non-blocking
note and recommended DONE. Hand-verification made it blocking on two grounds it had not weighed:
the defect is the *same* one F2 was blocked for, differing only in which layout artifact triggers
it; and it reproduces on a measured 22% of the shipped file's own lines. Consistency is the point —
waving through a failure mode one round after blocking it, because it is now narrower, would make
the severity bar depend on reviewer fatigue rather than on the defect.

**Note on scope for the next rework.** R1's fix should *generalise* the comparison rather than add
`>` to a list of exceptions; R2's should widen the skip's stated triggers. Both are one clause. If
the fix again needs an empirical check, the honest test is a run over a file with blockquotes and
list items in it — not a hand-picked paragraph.

## History

- 2026-08-25 — created (TO DO). source: field-use: a downstream workspace running step 4b got a full run of suggestions quoting text that existed nowhere in its tree, and now carries the verification rule in its own overarching addendum
- 2026-08-25 — TO DO → READY: plan complete
- 2026-08-25 — READY → IN DEVELOPMENT: picked up
- 2026-08-25 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-25 — IN REVIEW → REWORK: F2/F4 blocking: verbatim rule misfires on wrapped prose, re-invoke loop unbounded
- 2026-08-25 — REWORK → IN REVIEW: F2/F4 fixed: verbatim comparison tolerant of wrapping, re-invoke loop bounded
- 2026-08-25 — IN REVIEW → REWORK: R1/R2 blocking: F2 residual on blockquote/list prefixes, F4 exit not admitted by the skip
