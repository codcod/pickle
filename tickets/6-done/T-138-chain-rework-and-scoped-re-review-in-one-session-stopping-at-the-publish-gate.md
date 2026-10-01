---
id: T-138
title: chain rework and scoped re-review in one session, stopping at the publish gate
project: pickle
depends-on: [T-137]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-138 — chain rework and scoped re-review in one session, stopping at the publish gate

## Outcome

After this ships, "rework ticket T-NNN" carries on through an independent scoped re-review in
the same session, and "implement ticket T-NNN" carries on into an independent review. Each stops
at the publish gate, or after two rework rounds, instead of the human opening a fresh session to
type the next trigger after every phase. A host without sub-agents behaves as it does today.

## Description

Evidence, 2026-09-21..29: about 16 rework → validate cycles where the human's only input was the
trigger phrase in a fresh session — SMP-015 went validate → rework → validate → rework → validate
across five sessions in 25 minutes; the same for SMP-007/009/016/017, POR-008/010/018,
T-057/058/060 and two unity tickets. Implement → `/clear` → validate shows the same pattern. The
count is unvalidated; Task 1 measures it.

The fresh session exists for reviewer independence, and `review-protocol.md` step 0 already
accepts a delegated reviewer sub-agent with its own context as meeting that bar. Re-verified at
refinement against `main` @ ef375ba: step 0 already draws the boundary this ticket needs —
"Delegation covers the audits only. Classification and severity, the four dispositions, moving
the ticket, and the approval gate (step 9) stay with the orchestrating reviewer." What is
missing is the chaining itself: `SKILL.md` implement step 8 and `procedure-rework.md` step 6 both
end in "hand back", and step 0's *Session and tier* paragraph recommends a fresh session for
review.

The change:

- After a rework fix record, the rework session moves the ticket to `4-in-review/` and runs the
  scoped re-review as its orchestrator, with the audits delegated per step 0. New blocking
  findings start another rework round, **at most two rework rounds per invocation**; then it
  stops and hands back with the ticket in `5-rework/`.
- After implement, the session continues into validate the same way, then into the bounded
  rework loop.
- Human stops that remain: READY approval, publish approval (step 9), merge. A host without
  sub-agents keeps today's behaviour: stop and ask for a fresh session.

Two rules make the delegated reviewer as independent as a fresh session and safe in-tree:

1. **The spawn prompt is a fixed template.** It contains the trigger phrase, the ticket id, the
   ticket path as resolved from the base branch, the branch to audit, and one fixed role clause
   that is word-for-word the same on every spawn. There is no implementer-written prose in it: a
   fresh session receives only "validate ticket T-NNN", while an implementer writing "confirm F1
   is fixed" steers the reviewer toward the implementer's own framing. Literal "trigger + id
   only" was considered and rejected at refinement: a sub-agent handed just "validate ticket
   T-NNN" follows the whole protocol, ticket moves included, which breaks rule 2.
2. **The reviewer returns findings; the parent makes every ticket move, commit and branch
   switch.** The review's bookkeeping is committed on the base branch while the parent sits on
   the feature branch in the same working tree (under `in-tree`); a sub-agent switching branches
   under the parent is a hazard. This is step 0's existing boundary, now stated as a hard rule
   in the spawn template.

With T-137, a first review after implement may fan out to angle reviewers (`medium`/`high`); each
gets the same fixed template plus its fixed angle group. A scoped re-review is one pass.

This removes a human round trip, not a gate: NOTES.md § "Rejected outright, so they are not
re-proposed" rules out ordering, ranking, scoring or gating of *tickets*, which this does not
touch. Depends on T-137: the re-review runs its correctness path, and both edit
`review-protocol.md` step 0.

## Implementation Plan

### 0. Feature branch (mandatory)

Task 1 is bookkeeping and runs on `main` **before** this step; cut the branch after its target
and script are committed there.

```
cd .
git checkout main
git checkout -b feat/T-138-chain-rework-review
```

Do all work on this branch, committing locally as you go. Publish only per the project's commit
policy: no push or MR without explicit user approval; tidy WIP commits into atomic ones before
presenting (root-path child). Under `in-tree`, before pushing: `git fetch origin main && git diff
--name-only origin/main...HEAD | grep '^tickets/'` must print nothing. Merging is the human's.
Ticket and board bookkeeping (the retro target and script under `tickets/`) goes on `main`.

### Prerequisite gate (hard)

- T-137 in `6-done/` **and merged to `main`** (board `merged` column / its History). Cut the
  branch from a `main` that contains it — this ticket edits the step 0 text T-137 rewrote.
- `main` clean; `just test` green.

### Confirmed design decisions (do not deviate without asking)

1. **Chaining is automatic when the host can spawn sub-agents.** Implement continues into
   validate; rework continues into the scoped re-review. No confirmation prompt between phases.
   User decision at refinement (2026-10-01).
2. **At most two rework rounds per invocation.** A round is one rework fix record written in this
   invocation. After the second round's re-review, if blocking findings remain, the parent moves
   the ticket to `5-rework/` as usual and stops. The next "rework ticket T-NNN" starts a fresh
   count. User decision at refinement.
3. **The spawn prompt is a fixed template, written verbatim into `review-protocol.md` step 0**:
   trigger phrase, ticket id, ticket path from the base branch, branch to audit, a `hunt:` field
   (`one pass` | `angle 7 only` | `none`, read off step 3's *Who runs what*), the angle group
   when T-137's hunt fans out, and the fixed role clause. Every field is from a closed set; no
   implementer-written prose. User decision at refinement; fields amended at the gate (G1).
4. **The role clause forbids ticket moves, commits, branch switches and edits to tracked files**
   (build and test outputs of the configured commands excepted; scratch files go in a scratch
   directory), and asks for findings rows plus the checklist lines for what it ran. The parent
   does classification, dispositions, moves, commits and step 9, as step 0 already says, and
   **neither switches branches nor commits while any delegated reviewer is running** — no
   bookkeeping commit on `<base>` while a reviewer audits the same tree. (The parent is already on
   `<base>` when it spawns, having committed the `in-review` move there.) The parent records each delegated
   finding it discards on the checklist with a one-line reason.
5. **The chain always stops at step 9's approval presentation.** Publishing and merging are
   unchanged.
6. **No sub-agents on the host → today's behaviour.** Stop after `move … in-review` and ask for a
   fresh session; record nothing new.
7. **Step 0's "fresh session at a heavier tier" advice becomes the fallback, not the default.**
   On a host with sub-agents, the tier advice moves to the spawned reviewer (pin the heavier tier
   on it where the host allows), and step 5's severity calls get the heavier tier where the host
   can switch mid-session; the post-verdict drop-back advice stays.
8. **Rework procedure steps 1–5 are unchanged**, including recording SHAs untidied: the chained
   re-review reads the fix record exactly as a fresh session would.

### Tasks

#### Task 1 — Target and baseline, before any payload edit

Write the target into `tickets/retros/2026-09-29-self-improvement-loop.md` § "Open targets"
(replacing the T-138 bullet), in the § "What a target is" shape:

| Part | Definition |
|---|---|
| Claim | a review after implement or rework no longer needs the human to type its trigger in a fresh session |
| Unit | one concluded review — a `pickle ticket move <ID> done\|rework` tool call (or the move's History line) — of a ticket whose implement or rework session ended within the previous 2 hours |
| Relaunched | that review's session opened with the human's validate/review trigger as its first non-noise message |
| Metric | relaunched reviews ÷ reviews |
| Baseline | measured now over 2026-09-21..29 with the script below |
| Threshold | ≤ 20% over ≥ 10 reviews |
| Guard | sub-agent (sidechain) transcripts whose first prompt contains the spawn template's fixed role clause and that ran `pickle ticket move`, `git commit`, `git checkout` or `git switch` = 0 |
| Decision rule | met → keep; missed → find where chains stop; < 10 reviews within six weeks → inconclusive, re-measure at the next retro; guard > 0 → blocking bug against this ticket's rule 2 |

Save the script as `tickets/retros/<today>-review-relaunches.py`, modelled on
`2026-09-29-gate-stops.py` (same `CLAUDE_CONFIG_DIR` handling, NOISE filter, `SINCE UNTIL` args).
**Validate by hand on three cycles before quoting a number.** Record the baseline.

**Who runs it.** As with T-137, transcript reads are likely denied to the agent: the agent writes
the script (with a `--show SID8` option that prints one session's matched cycle for hand
validation), the user runs `! python3 tickets/retros/<file> 2026-09-21 2026-09-29` and the
`--show` checks, and the History line records who ran it.

**Premise check:** fewer than 8 relaunches in the window → stop, record it in History, and ask
the user whether to drop or re-grade (T-139's route).

#### Task 2 — Spawn template and boundary in review-protocol step 0

`skill/resources/review-protocol.md` § "0. Reviewer independence":

- Add a **Spawn prompt** paragraph with the template from decisions 3–4 as a fenced block, e.g.

  ```
  validate ticket <ID>
  ticket: <path, resolved from <base>> · branch: <feat/…> · hunt: <one pass | angle 7 only | none>[ · angles: <group>]
  You are a delegated reviewer (step 0 of this skill's resources/review-protocol.md). If
  `angles:` is set, run only step 3's hunt for those angles. Otherwise run steps 1 to 4a, with
  step 3's hunt limited to what `hunt:` says and the rest of step 3's quality audit in full.
  Return findings rows (severity, class, evidence, suggestion) and the checklist lines for what
  you ran. Do not move tickets, commit, switch branches, or edit tracked files (build and test
  outputs of the configured commands excepted); put scratch files in a scratch directory.
  ```

  with the `hunt:` values defined against step 3's *Who runs what* (`one pass`: a `low` ticket,
  a scoped re-review, or no host tool and no fan-out; `angle 7 only`: the host-tool path when
  the tool does not review for security; `none`: the tool covers security, or the hunt fans
  out), and one sentence on why nothing else goes in (decision 3's rationale, without naming a
  ticket). Step 0's *Trigger* says "steps 2 through 4a": note that step 1 here only loads
  context, or match the wording.
- State decision 4's parent duties in the *Boundary* paragraph: stay on the feature branch until
  every reviewer returns; record each discarded delegated finding with a reason.
- Reconcile the *Trigger* paragraph's "Hand it the ticket as step 1 reads it, the branch to
  audit, and the child's configured commands" with the template (the template carries the path
  and branch; the reviewer reads commands from `AGENTS.md`).
- Rewrite *Session and tier* per decision 7.

#### Task 3 — Chain implement into review

`skill/SKILL.md` § "Procedure: implement a ticket" step 8: after `move … in-review`, on a host
that can spawn sub-agents continue into *Procedure: validate a ticket* as its orchestrator
(audits delegated per step 0), then into the bounded rework loop (decision 2), stopping at step
9. Without sub-agents, hand back as today.

#### Task 4 — Chain rework into re-review

`skill/resources/procedure-rework.md` step 6: same continuation for the scoped re-review; state
the two-round cap and that the round count resets per invocation; at the cap with blocking
findings left, the ticket goes to `5-rework/` and the session stops and summarises the open
findings.

#### Task 5 — Wording that assumes a hand-back

- `skill/resources/tickets-README.md` §2 status list, IN REVIEW: "built, acceptance test green,
  handed back; awaiting review" → drop "handed back".
- `skill/resources/TEMPLATE.md` Finish step 6: rewrite the whole step (its publish-approval
  sentence would stop the chain before review) as "Commit locally on the ticket branch; do not
  push or open a merge request — approval and publishing happen at the end of the review (the
  review protocol's step 9). Move the ticket to `4-in-review/`; the implement procedure says
  whether the session continues into review." Keep the in-tree base check where step 9 already
  carries it.
- `skill/resources/procedure-rework.md` step 5: "before handing back" → "before the re-review".
- `grep -rn -i "hand back\|handed back\|handing back\|fresh session" skill/` afterwards: every
  remaining hit is either the no-sub-agent fallback or unrelated.

### Acceptance test

```
just build && just test && just lint && just docs-check
grep -n 'Do not move tickets, commit, switch branches' skill/resources/review-protocol.md   # template present
for f in skill/SKILL.md skill/resources/procedure-rework.md; do grep -qi 'two rework rounds\|at most two' "$f" || { echo "missing: $f"; false; }; done   # cap in both
grep -n 'handed back' skill/resources/tickets-README.md ; test $? -eq 1
grep -n 'Hand back to the user' skill/resources/TEMPLATE.md ; test $? -eq 1
git ls-tree --name-only main tickets/retros/ | grep -q review-relaunches   # Task 1 landed on main
```

Plus live chains, recorded under `## Review` by the implementer. Setup, after `just build`:
`D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q -b main &&
./pickle-test install --in-tree --test '<cmd>'`, an initial commit, then `./pickle-test ticket
new` and a move to READY. Drive a **top-level headless session** in `$D` (`claude -p
--output-format stream-json --verbose`; its stream carries each Agent call's prompt and the
sub-agent's tool calls, so no transcript-store read is needed), answering the applicability
gate's routing with `--resume`; ask the user before granting the session any permissions. Two
chains:

1. "implement ticket T-1" on a correct one-task plan: confirm (a) the review ran as a spawned
   sub-agent whose prompt matches the template exactly, (b) the sub-agent made no move, commit or
   branch switch, (c) the session stopped at step 9.
2. "rework ticket T-1" on a seeded `5-rework/` ticket with a recorded blocking finding and a
   seeded buggy branch: confirm (a) and (b) again, and that the parent fixed, re-reviewed and
   ended either in `6-done/` stopped at step 9, or at the two-round cap in `5-rework/` with a
   stop.

### Docs update (mandatory when user-facing)

- `docs/user-manual/concepts/agent-session-workflow.adoc`: the *Implement* and *Validate* rows
  (session column "New session") → implement continues into review on hosts with sub-agents;
  a new session remains the fallback. Update the tier column per decision 7 (the heavier tier
  goes on the spawned reviewer). Add a *Rework a ticket* row with its chaining and the two-round
  cap.
- `docs/user-manual/concepts/lifecycle.adoc` § "Reviews: severity, then disposition": one
  sentence that rework and re-review chain in one session, stopping at publish approval.
- `CHANGELOG.md` `## [Unreleased]` → `### Changed` (T-138).
- `just docs-check` clean.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (session-workflow page, lifecycle page, CHANGELOG).
3. Summary: files touched, the baseline measured, the live-chain result, anything deferred.
4. Suggested commits (atomic, after tidying), e.g.
   `feat(skill): chain review and rework in one session up to the publish gate (T-138)`,
   `docs: document chained review and rework (T-138)`.
   The retro target and script are bookkeeping under `tickets/` and land on `main`.
5. Tidy WIP commits into atomic ones (root-path child); keep history by default.
6. Commit locally; no push or MR without approval; in-tree base check before push; merge is the
   human's. `pickle ticket move T-138 in-review --reason "acceptance green"`.

## Review

### Implementer notes — live chains (2026-10-01)

Throwaway in-tree installs (`pickle-test`, branch build), a one-task shell project, top-level
headless `claude -p --output-format stream-json` sessions (Opus 5.5) with a scoped tool allowlist
the user approved. Each stream was parsed for Agent prompts and for sub-agent Bash calls matching
`pickle ticket move|git commit|git checkout|git switch`.

| run | chain | (a) spawn prompt = template | (b) sub-agent moves/commits/switches | (c) end state |
|---|---|---|---|---|
| 1 | implement, correct plan | yes, plus one appended line of its own ("Repo root: …") | 0 (11 Bash calls) | `6-done/`, stopped at step 9 |
| 2 | rework, seeded F1 on a buggy branch | **no**: free-form prompt with the author's framing ("is F1 actually fixed — `triple 4` must print 12") | 0 (4) | `6-done/`, stopped at step 9 |
| 1b | implement, after the fix below | yes, verbatim; `hunt: angle 7 only` (host tool ran) | 0 (9) | `6-done/`, stopped at step 9 |
| 2b | rework, after the fix below | yes, verbatim; `hunt: one pass` (host tool cannot take the fix range) | 0 (6) | `6-done/`, stopped at step 9 |

Run 2's parent read `procedure-rework.md`, then grepped `review-protocol.md` and read from step 5
on, never seeing step 0's *Spawn prompt*: "delegated per step 0" did not send it there. Fixed on
the branch: `SKILL.md` implement step 9 and rework step 6 now say to read the *Spawn prompt* in
step 0 and copy it verbatim, filling only its `<…>` fields and adding nothing. Run 2 also showed
decision 4's "stays on the feature branch until every reviewer returns" cannot hold, since the
parent commits the `in-review` move on `<base>` before it spawns; reworded to "neither switches
branches nor commits while a reviewer runs" (plan amended, History line). The two-round cap's stop
path was not exercised (both rework chains closed in one round), as the plan's either-end-state
rule allows.

Also seen: the payload lint's "our own" rule (`payload_lint_test.go`, first-person and
invisible-evidence patterns) has no leading word boundary, so it flags "your own"; the branch
rephrased around it ("add nothing to it"). Not fixed here.

### Review — 2026-10-01

Reviewed on `feat/T-138-chain-rework-review` @ `6cc4848` against `main` @ `026c166`; inline fixes
in `dc27146`.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | spec-unclear | fixed inline | Step 0 *Boundary* forbids the delegated reviewer to switch branches but never says the orchestrator must spawn it with the feature branch checked out; a parent left on `<base>` after its bookkeeping commit would have the reviewer audit and test `<base>` | `review-protocol.md` § 0 *Boundary*; decision 4 says "the parent is already on `<base>` when it spawns"; live streams `chain1b`/`chain2b` show the parent did check out the branch (step 0a) before each spawn, so no wrong audit was observed | Fixed in `dc27146`: "spawns it with the ticket's feature branch already checked out (step 0a's checkout)" |
| F2 | non-blocking | spec-unclear | fixed inline | Spawn prompt nests one placeholder in another, `<path, resolved from <base>>`, while the instruction is to fill only the `<…>` fields | host code-review tool; live fills read `tickets/4-in-review/T-001-….md (resolved from main)`, correct both times, so the truncation the tool predicted did not reproduce | Fixed in `dc27146`: `<path, resolved from the base branch>` |
| F3 | non-blocking | stale-xref | fixed inline | The getting-started tour still says the implement session "hands back" and has the user type `review ticket T-1`, which this branch made false on a host with sub-agents | `docs/user-manual/your-first-project.adoc` § 5–6 (found in the 4a whole-tree sweep) | Fixed in `dc27146`: hand-back is now the no-sub-agent case; § 6 says when the trigger is still typed |
| F4 | non-blocking | spec-unclear | fixed inline | Session table's *Rework* row says "New session", but a rework reached from an implement chain runs in that same session (`SKILL.md` implement step 9) | `agent-session-workflow.adoc` *Rework a ticket* row | Fixed in `dc27146`: "New session, or the same one when a review chained into it" |
| F5 | non-blocking | design | fixed inline | `procedure-rework.md` step 5 left a 153-column unwrapped line after the edit | `awk 'length>100'` on the file | Reflowed in `dc27146` |
| F6 | non-blocking | stale-xref | noted | Decision 4's parenthetical and the 2026-10-01 "plan amended inline" History line say the parent is on `<base>` when it spawns; the live streams show it on the feature branch at each spawn (it returns to `<base>` only after the reviewer finishes) | `chain2b`: `git checkout -q main && … commit … && git checkout -q feat/T-001-triple && pickle doctor`, then the Agent call | Plan prose and append-only History; the shipped payload (F1) now states the correct order |
| F7 | non-blocking | design | noted | The payload lint's `our own\b` pattern has no leading word boundary, so it also flags "your own" (implementer's observation, confirmed) | `payload_lint_test.go:139`, `:179` | Pre-existing, not this branch; noted for whoever next touches the lint |

Disposition summary: 5 fixed inline (F1–F5), 2 noted (F6, F7), 0 folded, 0 new tickets.
cost: estimated M, actual M

- [x] Reviewer independence settled (step 0): **independent**. The reviewing session started from a cleared context with no memory of writing the branch, so it ran the audits itself; nothing delegated, no discards
- [x] In-tree stale-branch check (step 0a): `pickle doctor` on the feature branch, 0 errors, 0 warnings
- [x] Implementation audit (steps 1, 2): acceptance test re-run verbatim, all green (build, test, lint, docs-check; template grep; cap in both files; no "handed back"/"Hand back to the user"; `review-relaunches` on `main`). Tasks 1–5 met: target and baseline in `tickets/retros/2026-09-29-self-improvement-loop.md` with script `2026-10-01-review-relaunches.py` on `main`; spawn template, *Boundary*, *Trigger* and *Session and tier* rewritten (`review-protocol.md` § 0); implement step 9 (`SKILL.md`); rework step 6 (`procedure-rework.md`); Task 5 wording (`tickets-README.md` §2, TEMPLATE Finish 6, rework step 5). Decisions 1–8 hold in the payload. Task 5's sweep: the remaining hits are the no-sub-agent fallback or unrelated. Live chains: implementer's table above, re-read from the `chain1b`/`chain2b` stream files
- [x] Correctness hunt (step 3): **host tool path**. The top-level session ran `/code-review low main...feat/T-138-chain-rework-review`, which reported 1 finding, kept as F2 and reclassed from correctness to spec-unclear after it failed to reproduce; 0 discarded. The tool does not review for security, so the orchestrator ran angle 7 separately: the payload is prose, every spawn field comes from a closed set, and there is no trust boundary, so nothing was found. No sub-agents. Quality: tests unchanged (prose-only diff; `payload_lint_test.go` and `TestDocs` green)
- [x] Consistency audit (step 4): `grep -rn -i "hand back|handed back|hands back|fresh session|new session"` over `skill/`, `docs/`, `README.md`, `AGENTS.md`, `CLAUDE.md` turned up F3 and F4; `releasing.adoc` hits are unrelated (release steps)
- [x] Documentation audit (step 4a): coverage met (session-workflow table, lifecycle, CHANGELOG `[Unreleased]`); the whole-tree sweep found F3; `just docs-check` clean before and after `dc27146`
- [x] Docs-readability pass (step 4b): skipped, because no docs-readability reviewer is configured on this host (no `opencode`); 0 suggestions, 0 discarded
- [x] Findings recorded with severity, class and disposition; summary and cost lines present (step 5)
- [x] Ticket moved to `6-done/`, History appended (step 6)
- [x] Other references (step 7): no ticket or doc cites T-138 outside the retro report, which is already current. Governing documents (`AGENTS.md`, `CLAUDE.md`, `NOTES.md`) make no hand-back claim, so there is nothing to reconcile
- [x] Impact sweep (step 8): no ticket in `1-to-do/` or `2-ready/` depends on or cites T-138
- [ ] Summary, commit messages and MR attributes presented for approval; in-tree base check before any push; bookkeeping committed; next ticket suggested (step 9)

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: ~16 rework/validate cycles where the human only typed the next trigger in a fresh session
- 2026-10-01 — TO DO → READY: plan complete; fixed spawn template, 2-round cap, automatic chaining, host fallback
- 2026-10-01 — impact note from T-137 review: Task 2's spawn template must follow step 3's ownership rule as T-137 rework F1 settles it (note added under Task 2)
- 2026-10-01 — applicability gate (fresh sub-agent, vs main e55df59): clean, 12 non-blocking. G1–G11 and G12's grep fixed inline (user-approved); G12's pre-existing "nothing else audits it" claim noted
- 2026-10-01 — plan amended inline: spawn template gains closed-set `hunt:` field and follows step 3's *Who runs what* (G1); Task 1 counts concluded reviews (G2), is run by the user (G3), lands on main before the branch (G4), guard keyed on the role clause (G5); cap grep checks each file (G6); live chain driven headless as two chains (G7); TEMPLATE Finish step 6 rewritten whole (G8); session-workflow gains a rework row and tier update (G9); parent records discards, step 5 at heavier tier (G10); build outputs excepted, parent stays on branch until reviewers return (G11); sweep covers "handing back" (G12)
- 2026-10-01 — READY → IN DEVELOPMENT: picked up; applicability gate clean (12 non-blocking, G1–G11 + G12 grep fixed inline)
- 2026-10-01 — Task 1 baseline: 72 reviews within 2h of a hand-off, 60 relaunched (83%), 11 chained, guard 0; 3 cycles validated by hand; script run by the user; premise check held (60 ≥ 8); target recorded in tickets/retros/2026-09-29-self-improvement-loop.md
- 2026-10-01 — plan amended inline: decision 4's parent rule is "neither switches branches nor commits while a delegated reviewer runs" — the live chain showed the parent is on `<base>` (having committed the in-review move) when it spawns, so "stays on the feature branch" could not hold
- 2026-10-01 — IN DEVELOPMENT → IN REVIEW: acceptance green; live chains: template verbatim after fix, 0 sub-agent moves/commits/switches, stopped at step 9
- 2026-10-01 — IN REVIEW → DONE: review approved: 0 blocking; 5 fixed inline (F1–F5), 2 noted (F6, F7)
