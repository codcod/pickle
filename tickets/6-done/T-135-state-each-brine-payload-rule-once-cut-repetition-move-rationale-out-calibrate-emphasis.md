---
id: T-135
title: calibrate the brine payload's emphasis and cut its decision-defence prose
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: M
---

# T-135 — calibrate the brine payload's emphasis and cut its decision-defence prose

## Outcome

After this ships, the brine payload keeps bold, MUST and "unconditionally" for its real rails
only, drops the paragraphs that defend a decision's history rather than explain how to apply
it, and lists itself in a session's skills with the trigger phrases plus one sentence. No rule's
meaning changes.

## Description

Cleanup, landing after the behaviour changes from the same session review (T-137, T-138 and
T-140 are done; T-139 was dropped). Scope:

1. **Emphasis calibrated.** Bold, "unconditionally" and "every time" stay on the real rails —
   READY gate, publish gate, merging is the human's, bookkeeping on the base branch, never
   hand-edit `BOARD.md`. The rest is stated plainly, with its reason. Over-emphasis makes an
   Opus-class model treat every sentence as a hard rail. Measured on `main` at refinement
   (2026-10-02): 336 bold runs across `skill/` (138 in `tickets-README.md`, 94 in
   `review-protocol.md`, 75 in `SKILL.md`), 66 of them run-in paragraph labels; no `MUST`
   anywhere; three "unconditionally" and two "every time".
2. **Decision-defence prose cut** — paragraphs that argue for a decision, recount how it was
   reached or rebut alternatives (e.g. why `review-protocol.md` step 7 defines no severity, step
   4b's justification of its quote-matching check). Whatever pickle still wants on record moves
   to `DESIGN.md`.
3. **`SKILL.md` frontmatter `description`** (121 words, listed in every session whether or not
   brine is used) cut to the trigger phrases plus one sentence.
4. **`review-protocol.md` step 0's "session and tier" advice** cut to one line.

Out of scope, deliberately (challenged 2026-09-29, see NOTES.md § Session review and the 1.x
release plan (2026-09-29)):

- **Cross-file de-duplication.** T-129 loads procedures one at a time; "state it once, point
  there" would make an agent open a second file to get a rule. Repetition across procedure files
  is partly the price of that split. Repetition *within* one file may still be cut.
- **Moving the *why* out.** A rule's reason is what lets a model apply it to a case the rule did
  not foresee, and a foreign workspace has no `DESIGN.md` — the reason would vanish, not move.
  One clause of why stays with each rule; only the defence of it goes.

No field evidence ties a failure to payload length or emphasis — T-139's suspected symptom (the
applicability gate stopping on every pickup) was measured at 7 of 42 clean runs and dropped.
Hence low impact. `payload_lint_test.go`, the docs xref
check and any user-manual citation of payload sections stay green.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-135-calibrate-payload-emphasis
```

Commit locally as you go; publish only at the review's approval gate (root-path child: tidy WIP
commits into atomic ones, keep history by default). Under `layout = "in-tree"`, before pushing,
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing. Merging is the human's.

### Prerequisite gate (hard)

None. T-137 and T-138 (the last payload behaviour changes this cleanup was to follow) are in
`6-done/` and merged; T-139 is dropped.

### Confirmed design decisions (do not deviate without asking)

1. **No rule's meaning changes.** Every edit is wording, emphasis or deletion of defence prose. A
   sentence whose removal would change what an agent does stays. Where unsure, keep it.
2. **Bold survives in four places only.** (a) The five rails: READY gate, publish gate (no push
   or MR without approval), merging is the human's, bookkeeping on the base branch, never
   hand-edit `BOARD.md`. (b) Run-in paragraph labels (`**Trigger.**`, `**Boundary.**`, a
   bulleted `**Name** —` term at its definition, and a numbered-step or list-item lead-in such
   as `1. **Read the ticket in full.**`). (c) The `> **Project configuration wins.**`
   callouts. (d) Bold that is syntax in an example (TEMPLATE's confirmed-decision bold run, the
   §7 example block). Every other bold run is unbolded. User decision, 2026-10-02.
3. **"unconditionally" / "every time" stay only on a rail.** Elsewhere the rule keeps its
   frequency in plain words ("on every pickup") and loses the intensifier. Where the word carries
   meaning rather than emphasis — `review-protocol.md` step 0a and step 1's "unconditionally",
   meaning "not gated on X" — the meaning stays.
4. **Defence prose is deleted, not moved.** A cut paragraph goes to `DESIGN.md` only when it
   records a decision that `DESIGN.md` and `tickets/NOTES.md` do not already hold. Expected:
   nothing moves. User decision, 2026-10-02.
5. **One clause of why stays with each rule.** Only the argument for the rule goes: history,
   rebuttals of alternatives, analogies to other steps, "this is not a new rule" reassurance.
6. **No cross-file de-duplication.** Each procedure file still reads alone (T-129). Repetition
   within one file may be cut.
7. **Text anchors the tests, the parser and the manual read stay byte-identical**: `Project
   configuration wins` in the four files `install_test.go` lists (`SKILL.md`,
   `tickets-README.md`, `review-protocol.md`, `TEMPLATE.md`), and the absence of `pushing a child-project requires explicit user approval`
   and `Pickup is gated by a freshness check` (`internal/install/install_test.go`), the Layout
   tree in `tickets-README.md` (`internal/flow/flow_test.go`), the template sections
   `internal/ticket` and `internal/audit` tests parse; the step 0 *Spawn prompt* block, which
   the implement and rework procedures say to copy verbatim; the findings-table header and the
   "disposition summary" and `cost: estimated` closer lines `internal/state/review.go` parses
   (it tolerates bold, so unbolding them is safe, but the wording stays); the phrase "may be
   tidied" in rules §3, which `cli-reference.adoc` quotes; and every heading, step number and §
   the user manual cites. `docs_xref_test.go` checks only anchors inside the manual, so nothing
   mechanical guards these last citations — the acceptance test greps for them.

### Tasks

#### Task 1 — `skill/SKILL.md` frontmatter `description`

Replace the 121-word description with the trigger phrases plus one sentence, about 70 words:

```
description: Operate the brine ticket flow installed by the `pickle` CLI — one markdown ticket per feature, its status the directory it sits in, a generated BOARD.md as the index. Use when asked to "make it a ticket" (or "file a ticket"), "refine ticket T-NNN" (or "make it ready"), "implement ticket T-NNN", "rework ticket T-NNN", "validate ticket T-NNN" (or "review ticket T-NNN"), "audit the board", or move a ticket between statuses.
```

#### Task 2 — `skill/resources/review-protocol.md` step 0 "Session and tier"

Cut the paragraph to two clauses: "**Session and tier.** Give spawned reviewers the heavier
reasoning tier where the host lets you pin one, and switch to it for step 5's severity calls
where the host can switch mid-session; without sub-agents, start the review in a fresh session
at that tier. Once the verdict is reached, offer to drop back down." Every instruction stays; only
the defence of it goes.

#### Task 3 — named decision-defence passages

- `review-protocol.md` step 0, opening paragraph: keep one sentence (an author is a poor auditor
  of their own branch); drop the comparison to the pickup gate.
- `review-protocol.md` Scope blockquote: drop "That recording is what keeps this scope rule true
  in substance…".
- `review-protocol.md` step 4b, quote-verification paragraph: keep the procedure (match words and
  punctuation, ignore layout; discard, never repair; re-invoke once, then conscious skip; record
  the discard count on the checklist line; discard content-changing suggestions). Drop the
  step-0 analogy and the closing "confirming it followed its own constraints, not applying a new
  rule".
- `review-protocol.md` step 7: cut "This step defines no severity of its own…" to one sentence
  (a governing document this branch made false is an ordinary step-5 finding and, merely lagging,
  non-blocking) and drop
  the "different surface from 4a" closing paragraph's defence while keeping its distinction in
  one clause.

#### Task 4 — sweep every payload file

`skill/SKILL.md` and every `skill/resources/*.md` (`docs-readability.prompt.md` included), in
order of size: `tickets-README.md`, `review-protocol.md`, `SKILL.md`, `TEMPLATE.md`, the rest.
For each, apply decisions 2, 3 and 5: unbold per decision 2; drop intensifiers per decision 3;
delete sentences that argue, recount history or rebut alternatives per decision 5, and collapse
repetition inside the file. Commit per file so the review can read one diff at a time.

#### Task 5 — `DESIGN.md` (conditional)

Per decision 4. If nothing qualifies, record "nothing moved to DESIGN.md" in the summary.

### Acceptance test

Run from the repository root on the feature branch:

```
just build && just test && just lint && just docs-check
for f in skill/SKILL.md skill/resources/*.md; do
  printf '%s main=%s branch=%s\n' "$f" \
    "$(git show main:$f | grep -o '\*\*[^*]*\*\*' | wc -l)" "$(grep -o '\*\*[^*]*\*\*' $f | wc -l)"
done
git show main:skill/SKILL.md | sed -n 3p | wc -w; sed -n 3p skill/SKILL.md | wc -w
git diff --word-diff main -- skill/
grep -n 'may be tidied' skill/resources/tickets-README.md
grep -rnoE '(review-protocol\.md|rules) (step [0-9a-z]+|§[0-9.]+)|step [0-9]+[ab]?' docs/user-manual | sort -u
```

Expected: the four commands pass (`payload_lint_test.go`, the install, flow, ticket and audit
anchors, the docs xref check). Every surviving bold run belongs to one of decision 2's four
categories; that is the criterion. The per-line count above miscounts bold that wraps across
lines, so treat it as a guide only — it should fall well below the `main` total of 336. The
description drops from 121 words to 70 or fewer. "may be tidied" is still present, and every
step and § the manual cites still names the same heading in the payload. Reading the word-diff, the reviewer confirms decision 1 for every
hunk: each one deletes defence prose or removes emphasis, and none changes an instruction.

### Docs update (mandatory when user-facing)

`CHANGELOG.md` `[Unreleased]` → `### Changed`: one line saying the brine payload's emphasis is
calibrated to its real rails and its decision-defence prose is cut, with no rule changed (T-135).
No user-manual change: it cites payload sections by heading and step number, which decision 7
keeps, and it already holds the session-and-tier advice step 0 drops.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. CHANGELOG updated.
3. Summary: files touched, bold counts before and after per file, anything moved to `DESIGN.md`,
   any passage kept that the sweep doubted.
4. Suggested commit: `docs(skill): calibrate payload emphasis and cut decision-defence prose
   (T-135)`. The body names the four bold categories kept.
5. Tidy the per-file WIP commits into atomic ones (root-path child), commit locally, do not push.
   `pickle ticket move T-135 in-review --reason "acceptance green"`, then the review chains in
   per the implement procedure.

## Review

### Review — round 1 (2026-10-02)

Branch `feat/T-135-calibrate-payload-emphasis` at a990964 + 7cd4366 (two commits, tidied).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | spec-unclear | fixed inline | The §0 fold rule's cut left "the hazard" with no referent, losing the rule's one clause of why (decision 5) | `skill/resources/tickets-README.md:92-93` before the fix | Name the hazard — done: "lets uncommitted bookkeeping cross a branch switch", folded into a990964 |
| F2 | non-blocking | docs-gap | noted | Pre-existing: the manual cites "rules §0's explicit-pathspec rule", but §0 has no pathspec rule on `main` either; the rule lives in SKILL.md's commit-policy bullet and review-protocol step 9 | `docs/user-manual/cli-reference.adoc:860`; `git show main:skill/resources/tickets-README.md \| grep -c pathspec` → 0 | Re-point the citation in a later docs pass |

Disposition summary: 1 fixed inline (F1), 1 noted (F2), 0 folded, 0 new tickets; 0 blocking.
cost: estimated M, actual M

Nothing moved to `DESIGN.md` (decision 4): every cut passage defended a decision rather than
recording one, and none recorded a decision missing from `DESIGN.md` or `tickets/NOTES.md`.
Bold runs per line-grep, main → branch: SKILL 75→40, tickets-README 138→51, review-protocol
94→45, TEMPLATE 8→5, make-ticket 8→5, rework 4→0, audit-board 3→0, docs-readability 1→0,
other-projects 5→5 (151 total; the 150 guide is per-line and miscounts wrapped runs — every
survivor was checked against decision 2's categories).

- [x] Reviewer independence settled (step 0): delegated — the orchestrator wrote the branch; one independent reviewer ran steps 1–4a; both of its findings re-verified by hand, none discarded
- [x] In-tree stale-branch check (step 0a): `pickle doctor` clean after rebasing the branch onto the in-review move
- [x] Implementation audit — acceptance test re-run, tasks 1–5 and decisions 1–7 verified, word-diff read hunk by hunk (steps 1, 2)
- [x] Correctness hunt (step 3): host tool path — the host code-review tool on `main...feat/T-135-calibrate-payload-emphasis`, run by the orchestrator, 0 findings (prose-only diff); angle 7 run separately by the delegated reviewer (the tool does not review for security), 0 findings; no correctness rows
- [x] Consistency audit (step 4)
- [x] Documentation audit — CHANGELOG entry present, no manual change needed, `just docs-check` clean (step 4a)
- [x] Docs-readability pass — conscious skip: no docs-readability reviewer configured on this host (step 4b, optional)
- [x] Findings recorded with severity, class, and disposition; disposition summary and cost line present (step 5)
- [x] Ticket moved to `tickets/6-done/`; `## History` appended (step 6)
- [x] Other references: governing documents reconciled — `tickets/NOTES.md` and `DESIGN.md` assert nothing this branch made false; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep: no ticket in `1-to-do/` or `2-ready/` depends on or cites T-135 (step 8)
- [ ] Summary + commit messages presented for approval; under in-tree, `origin/main...HEAD` carries no `tickets/` path before pushing (step 9)

## History

- 2026-09-29 — created (TO DO). source: self-host: payload review for frontier models (2026-09-29 session review): over-emphasis and decision-defence prose in the payload
- 2026-10-02 — TO DO → READY: plan complete
- 2026-10-02 — plan amended inline: applicability gate, 8 non-blocking findings, all approved for inline amendment — description keeps "make it ready" and adds "file a ticket"; tier line keeps the step-5 switch and drop-back; step 7 wording keeps "non-blocking" unhedged; decision 7 adds the spawn prompt, the lines internal/state parses and the manual's quotes and citations; numbered lead-ins count as labels and the bold threshold becomes a guide; decision 3 keeps meaning-bearing "unconditionally"
- 2026-10-02 — READY → IN DEVELOPMENT: picked up
- 2026-10-02 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-10-02 — IN REVIEW → DONE: review clean: 0 blocking; 1 fixed inline (F1), 1 noted (F2)
- 2026-10-02 — published: user approved; main pushed, branch pushed (2 commits, history kept), PR #100 opened; awaiting human merge
