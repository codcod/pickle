---
id: T-137
title: review hunts for correctness bugs: a multi-angle bug hunt with reproduction before the findings table
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-137 — review hunts for correctness bugs: a multi-angle bug hunt with reproduction before the findings table

## Outcome

After this ships, a brine review's quality audit is a correctness hunt. It runs the host's own
code-review tool on the branch diff when the host has one, and otherwise hunts over eight named
angles, fanned out to sub-agents for `medium`/`high` tickets. Every `correctness` finding in the
table carries a reproduction, or an explicit `unreproduced:` reason with the traced code path. A
replay on two pre-fix commits with known bugs shows that the hunt finds them.

## Description

Evidence, sessions of 2026-09-21..29 across smppai, porth, messgr, unity and pickle: after a brine
validate the user ran the host's code-review command on the PR **31 times**. Runs typically
returned 7–10 findings; only 2 explicitly reported no correctness bug, and the user acted on
nearly all findings. Not every finding was a correctness bug — the count of those was not
taken (Task 1 takes it) — but the confirmed ones include smppai PR 42 acking `data_sm` then
dropping it on three paths, PR 46 accepting a connection after the handler disconnected it, PR 40
never answering a bind, porth PR 5 growing an unbounded store, porth PR 7 racing on DLRs. The
brine round-1 review of the ticket behind PR 42 (SMP-007, in porth-umbrella) was one agent, one
blocking finding; its scoped re-review read only the fix diff and passed; the `/code-review high`
run after publish then found ten more, six of them correctness (SMP-007 `## Review` § "Post-review").

Re-verified at refinement against `main` @ 6b36f67 (1.1.0): `review-protocol.md` step 3 ("Quality
audit") is still five generic bullets, and nothing else in the protocol describes *how* to find a
bug. Step 0 already supplies the two pieces this ticket builds on: delegating the audits to an
independent reviewer, and the rule to re-verify every delegated finding before recording it. Step
4b already shows the triage shape for an outside tool's output.

Scope, cheapest first:

1. **A host code-review tool, when present, runs on the branch diff instead of the hunt.** Its
   output is triaged like a delegated audit (step 0's *verify before recording*): each finding
   is re-verified against the code before it enters the table, and the table records which
   path ran. This is what the user did by hand 31 times. Worded generically: no host-specific
   command name (the payload is read by projects on any agent).
2. **Without one, step 3 becomes a correctness hunt** over eight named angles: caller contracts
   and cross-module effects; error, cancellation and shutdown paths; ordering and concurrency;
   boundary inputs; resource lifetimes; breaking changes to API, wire format or config; security
   at trust boundaries; conformance to the ticket's confirmed decisions. Scaled by `complexity`,
   budgeted as a sub-agent count: `low` is one pass with no sub-agents, `medium` at most 3
   sub-agents with the angles grouped, `high` at most 5. The top-level session spawns them —
   a delegated reviewer may be unable to spawn its own on some hosts, so the plan does not
   depend on it.
3. **A `correctness` finding needs a reproduction** (a failing test, a script, a command
   transcript) before it is recorded. Where one is impractical (a race, an external system),
   the evidence cell says `unreproduced: <why>` and traces the code path instead; either kind
   may be blocking.

The scoped re-review (T-124) runs the same path over **this round's fix diff only**, as one pass
with no fan-out.

**Acceptance is a replay**, not a reading: run the new hunt on two pre-fix commits with known bugs
and require it to find them — smppai `1ef5588` (SMP-007's round-1 review tip; known: `data_sm`
acked then dropped when no handler is set, R1–R3 of its post-review table) and porth `8c214c8`
(POR-002's review tip, PR 7; known: POR-013 items 1, 3 and 4). The replay exercises the hunt
path, not the host tool: the tool already found these bugs, so replaying it would prove nothing.

**T-085's class count** is disturbed: a hunt shifts findings toward `correctness`. Reviews after
this ships are counted separately from those before it. (The criterion itself is overdue: 50
reviews carry the column and neither of its thresholds fires — `design` is ~23% of non-blocking
rows. That is a separate decision, not this ticket's.)

CI parity (a child's `lint`/`test` should be its CI entry point) was considered for this ticket
and not included — one project, already fixed there by its own review addendum.

Soft coupling: T-138 (chained re-review) depends on this and runs the re-review path defined
here.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-137-review-correctness-hunt
```

Do all work on this branch, committing locally as you go. Publish only per the project's commit
policy: no push or MR without explicit user approval; tidy WIP commits into atomic ones before
presenting (root-path child). Under `in-tree`, before pushing: `git fetch origin main && git diff
--name-only origin/main...HEAD | grep '^tickets/'` must print nothing. Merging is the human's.
Ticket and board bookkeeping goes on `main`, never on this branch (hooks enforce it).

### Prerequisite gate (hard)

- `main` clean and up to date; `just test` green on it.
- smppai `1ef5588` and porth `8c214c8` still resolve (`git -C ~/Projects/codcod/smppai cat-file -e
  1ef5588 && git -C ~/Projects/umbrella-org/porth-umbrella/projects/porth cat-file -e 8c214c8`).
  If either is gone, stop and ask for a substitute pre-fix commit with known bugs.

### Confirmed design decisions (do not deviate without asking)

1. **A host code-review tool replaces the angle hunt; it does not run alongside it** — except
   angle 7 (security), which still runs whenever the tool does not review for security (amended
   after review F2). One correctness path per review, cheapest first. User decision at refinement
   (2026-10-01).
2. **The payload never names a host-specific command.** It says "a code-review tool or command
   the host provides that audits a diff for bugs", the same register as step 4b's
   docs-readability reviewer. `/code-review` appearing anywhere under `skill/` is a defect.
3. **Host-tool output is triaged, not trusted.** Every finding it reports is re-verified against
   the code before entering the table (step 0's rule), and the review's checklist records which
   path ran (`host tool` / `angle hunt`) and how many tool findings were discarded on
   re-verification.
4. **The eight angles are fixed and named in the protocol**, in this order: caller contracts and
   cross-module effects; error, cancellation and shutdown paths; ordering and concurrency;
   boundary inputs; resource lifetimes; breaking changes to API, wire format or config; security
   at trust boundaries; conformance to the ticket's confirmed decisions.
5. **The fan-out budget is a sub-agent count by `complexity`: `low` 0 (one pass), `medium` ≤ 3,
   `high` ≤ 5.** It is a ceiling, not a quota. User decision at refinement: a count can be checked
   on any host, a token figure cannot.
6. **The top-level session spawns the hunt's sub-agents.** When step 0 delegates, the orchestrator
   spawns the angle reviewers directly instead of one delegated reviewer that would fan out; each
   angle reviewer is independent by construction, so this satisfies step 0 for step 3. Briefing
   per step 0 (ticket as step 1 reads it, branch, configured commands) plus its angle group.
   **Angle reviewers run only step 3 for their group**; steps 2, 4 and 4a go to one delegated
   reviewer (or to the orchestrator itself when it is already independent). **The host tool is
   run by the top-level session**, since a sub-agent may be unable to invoke a host command; its
   output then goes through decision 3's triage.
7. **A `correctness` finding's evidence is a reproduction, or `unreproduced: <why>` plus the
   traced code path.** A reproduction is a failing test, a script, or a command transcript with
   its observed output. The review does not commit it to the branch; it goes in the evidence cell,
   or in a `F<n> reproduction:` block under the findings table when it is too long for a cell.
   Either kind of evidence may be blocking. User decision at refinement.
8. **The scoped re-review runs the same path (host tool or hunt) over this round's fix diff only,
   as one pass with no fan-out**, keeping T-124's bound: each round reads its predecessor's new
   text.
9. **The `class` vocabulary is unchanged.** The hunt finds `correctness` findings; it adds no
   class, severity or disposition.
10. **The T-085 split is recorded in `tickets/NOTES.md`, not in the payload.** The payload lint
    rejects "pre-registered", and the count is this project's, not a foreign reader's.

### Tasks

#### Task 1 — Target and baseline, before any payload edit

Write the target into `tickets/retros/2026-09-29-self-improvement-loop.md` § "Open targets"
(replacing the T-137 bullet), in the § "What a target is" shape:

| Part | Definition |
|---|---|
| Claim | brine's review finds the correctness bugs a separate code-review pass was finding after it |
| Unit | one out-of-band code-review run on a branch whose brine review (validate/review trigger) had already reached a verdict |
| Metric | runs reporting ≥ 1 `correctness` finding ÷ runs |
| Baseline | measured now over 2026-09-21..29 with the script below |
| Threshold | ≤ 25% over ≥ 10 runs |
| Guard | median brine validate-session token use ≤ 2× its baseline over the same window |
| Decision rule | met → keep; missed → re-examine the angles against the escaped bugs; < 10 runs within six weeks of release → inconclusive, re-measure at the next retro |

Save the script as `tickets/retros/<today>-code-review-after-validate.py`, modelled on
`2026-09-29-gate-stops.py` (same `CLAUDE_CONFIG_DIR` handling, `SINCE UNTIL` args). Detect a
run the way `2026-09-29-weekly.py` does (`<command-name>/code-review`) — gate-stops' NOISE filter
drops every `<command-name>` message, so it must not be applied to the run detector. Where the
run reports through a structured findings tool with a category field, count
`category == "correctness"`; otherwise fall back to the run's final text. The same script also
prints the guard's baseline: median token use of brine validate/review sessions in the window.
**Validate by hand on three runs before quoting a number** (the retro's rule). Record the
baseline in the target table.

**Who runs it:** the agent's reads of the transcript store are denied by the host's classifier,
so the agent writes the script and the user runs it (`! python3 tickets/retros/<file> 2026-09-21
2026-09-29`) and spot-checks three runs. Task 1 and Task 7 are `main` bookkeeping: do both on
`main` before cutting the feature branch.

**Premise check:** if the baseline shows fewer than 30% of runs with a correctness finding, stop,
record the result in the ticket's History, and ask the user whether to drop or re-grade — T-139's
route.

#### Task 2 — Rewrite review-protocol step 3 as the correctness hunt

`skill/resources/review-protocol.md` § "3. Quality audit": retitle to *Correctness hunt and
quality audit* and replace the five bullets with:

- **Host tool path** (decisions 1–3): when the host provides a code-review tool or command that
  audits a diff for bugs, run it on `<base>...<branch>`, then re-verify each finding before it
  enters the table; record the path and the discard count on the checklist.
- **Angle hunt path** (decisions 4–6): the eight angles, each a one-line prompt for what to look
  for; the `complexity` → sub-agent-count table; who spawns.
- **Reproduction rule** (decision 7).
- Keep, shortened, the remaining non-correctness checks from the old bullets: tests assert
  behaviour, docs accurate and registered, prompt/config content unambiguous.

Keep the step number (addenda in `pickle.toml` key to step numbers).

#### Task 3 — Wire the hunt into step 0, step 1 and the checklist

- Step 0 *Trigger*: when the hunt fans out, the orchestrator spawns the angle reviewers itself
  (decision 6); one sentence.
- Step 1, scoped re-review bullet: the re-review runs step 3's path over this round's fix diff,
  one pass (decision 8).
- Step 5, after the class-table worked examples: one sentence that a `correctness` row's
  evidence follows step 3's reproduction rule.
- Checklist line "Quality audit (step 3)" →
  `Correctness hunt (step 3): host tool or angle hunt — name which, sub-agents used, tool findings discarded on re-verification; every correctness row reproduced or marked unreproduced`.

#### Task 4 — Summaries that restate the review

- `skill/SKILL.md` § "Procedure: validate a ticket" item 1: "Audit implementation, quality,
  consistency, and docs" → name the correctness hunt (host tool, else named angles).
- `skill/resources/tickets-README.md`: grep for any restatement of the review's audits
  (`grep -n "quality" skill/resources/tickets-README.md`) and align it; if none, nothing to do.

#### Task 5 — Guard the generic wording

`payload_lint_test.go`: add a fifth rule matching the literal `/code-review` with rule 3's
leading boundary `(^|[^\w./-])`, so `resources/review-protocol.md` does not match (a generic
"any slash command" pattern would hit it across the payload). No fence exemption. Give the
escape its own test case rather than filing it under
`TestPayloadLintRulesCatchTheEscapesTheySawInReview` (it was never seen in review), and add
`resources/review-protocol.md` to `TestPayloadLintRulesLeaveLegitimateShapesAlone`. Update every
"four" that counts the rules: `payload_lint_test.go` (~:118, :173, :288, :326 — check each, :173
may refer to rule 4 alone) and the root `AGENTS.md` paragraph above the marker block ("one of its
four rules").

#### Task 6 — Replay (the acceptance evidence)

Run the new step 3 **angle hunt path** (not the host tool — decision 1's rationale is that the tool
already found these) on each pre-fix commit, in a fresh agent session per replay with no memory
of this ticket, confined to its worktree (it must not read the umbrella's `tickets/`, where
SMP-007's post-review table and POR-013 hold the answers), using throwaway worktrees:

```
S=$(mktemp -d)
git -C ~/Projects/codcod/smppai worktree add "$S/smppai" 1ef5588
git -C ~/Projects/umbrella-org/porth-umbrella/projects/porth worktree add "$S/porth" 8c214c8
```

Brief each replay with: the new step 3 text from this branch, the ticket as it stood at its
round-1 review — `git -C ~/Projects/umbrella-org/porth-umbrella show a8379fd:<SMP-007 path>` and
`b5ee3aa:<POR-002 path>` (SMP-007 has a second `4-in-review` add, `7a745f3`, the rework round —
not that one) — the diff `1ef5588~1..1ef5588` / `8c214c8~1..8c214c8` (both single-commit
branches), and the child's commands. Complexity as each ticket was graded (SMP-007 `low`: one
pass; POR-002 `medium`: ≤ 3 angle sub-agents, spawned directly by this implementing session,
which knows the answers). **So judge the pass bar on the angle reviewers' raw output, before any
re-verification by this session**, and brief them with nothing beyond the list above. Save each raw findings table to
`tickets/retros/<today>-t137-replay-{smppai,porth}.md` (`main` bookkeeping).

**Pass bar:** smppai — at least one of R1–R3 (an inbound `data_sm` acked `ESME_ROK` with no
handler set, on client, high-level client or server); porth — at least one of POR-013 items 1, 3,
4 (early multipart receipt dropped; `_drop()` discards queued receipts; `expand_dlr_url` mangles
percent-encoding). Every `correctness` row in both tables has a reproduction or an
`unreproduced:` reason. A miss is reported, not retried until it passes: record which angle
should have caught it and stop for the user.

Remove the worktrees afterwards (`git worktree remove`).

#### Task 7 — T-085 split

Append to `tickets/NOTES.md` § "T-085's pre-registered criterion — recorded so the 8th review
after it ships can find it" a dated paragraph: reviews whose `IN REVIEW →` History line is dated
on or after the date of T-137's `merged to main` History line are counted separately; filter the
recipe on that date once the line exists (the merge has not happened when this is written).
**This is bookkeeping: commit it on `main`, not on the feature branch.**

### Acceptance test

```
just build && just test && just lint && just docs-check
grep -rn '/code-review' skill/ ; test $? -eq 1                   # decision 2: no host command name
grep -c 'angle' skill/resources/review-protocol.md               # > 0
for a in 'caller contracts' 'cancellation' 'concurrency' 'boundary inputs' \
         'resource lifetimes' 'wire format' 'trust boundaries' \
         "conformance to the ticket's confirmed decisions"; do
  grep -qi "$a" skill/resources/review-protocol.md || echo "missing angle: $a"
done                                                             # prints nothing
grep -n 'unreproduced:' skill/resources/review-protocol.md       # ≥ 1 hit (decision 7)
grep -n 'Correctness hunt (step 3)' skill/resources/review-protocol.md   # the checklist line
ls tickets/retros/*code-review-after-validate.py tickets/retros/*t137-replay-*.md
```

Plus the replay pass bar from Task 6, read from the two saved findings tables. A reviewer
re-runs the commands above verbatim and reads the two tables; re-running a replay is optional
(it is an agent run, not deterministic).

### Docs update (mandatory when user-facing)

- `docs/user-manual/concepts/lifecycle.adoc` § "Reviews: severity, then disposition": one
  paragraph — a review hunts for correctness bugs (host tool, else named angles scaled by
  complexity), and a `correctness` finding carries a reproduction.
- `CHANGELOG.md` `## [Unreleased]` → `### Changed`: the review's quality audit is now a
  correctness hunt (T-137).
- `just docs-check` clean.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (lifecycle page, CHANGELOG).
3. Summary: files touched, the baseline measured, both replay results, anything deferred.
4. Suggested commits (atomic, after tidying), e.g.
   `feat(skill): review hunts for correctness bugs with reproduction (T-137)`,
   `test(payload): reject host-specific review command names (T-137)`,
   `docs: document the review's correctness hunt (T-137)`.
   Retro script and replay tables are bookkeeping under `tickets/` and land on `main`.
5. Tidy WIP commits into atomic ones (root-path child); keep history by default.
6. Commit locally; no push or MR without approval; in-tree base check before push; merge is the
   human's. `pickle ticket move T-137 in-review --reason "acceptance green"`.

## Review

Reviewed 2026-10-01 on `feat/T-137-review-correctness-hunt` @ `73ed673` (rebased onto `main` @
`1d61a44`), under the step 3 this branch ships.

- [x] Reviewer independence settled (step 0): **delegated**. This session wrote the branch, so
  steps 2, 4 and 4a went to a fresh, adversarial sub-agent (findings D1–D7). Step 3 took the host
  tool path and was run by the top-level session.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned that the branch had the ticket in
  `3-in-development`; rebased onto `main`, re-run clean.
- [x] Implementation audit (steps 1, 2): all of Tasks 0–7 and decisions 1–10 met. Decision 6 is met
  but under-specified (F1). The acceptance test re-ran verbatim and passed; `just build`, `just
  test`, `just lint` and `just docs-check` passed (lint skipped actionlint and shellcheck, which are
  not installed). Task 6 replay: pass bar met on both repos (`tickets/retros/2026-10-01-t137-replay-*.md`).
- [x] Correctness hunt (step 3): **host tool** (`code-review high` on `main...HEAD`), no sub-agents.
  9 findings (H1–H9); 1 discarded on re-verification (H5: range grades are collapsed at refinement,
  `tickets-README.md` §3), the other 8 confirmed and merged with the delegated findings below. No
  `correctness` row needs a reproduction: the branch is protocol prose plus one lint rule, whose
  behaviour the test suite exercises.
- [x] Consistency audit (step 4): the delegated sweep of "quality audit", "step 3" and "steps 2
  through 4a" across the payload, docs and `pickle.toml` found nothing stale beyond F1–F4 and D3–D6.
- [x] Documentation audit (step 4a): `lifecycle.adoc` and `CHANGELOG.md` cover the change; whole-tree
  sweep clean; docs build clean.
- [x] Docs-readability pass (step 4b): conscious skip. No docs-readability reviewer is configured
  in this Claude Code session.
- [x] Findings recorded (step 5)
- [x] Ticket moved to `5-rework/` (step 6a)
- [x] Other references and governing docs (step 7): the root `AGENTS.md` lint paragraph reconciled in
  the branch (F9).
- [x] Impact sweep (step 8): T-138 Task 2's spawn template must take step 3's ownership from F1's
  fix; patched with a note in T-138 (History line there).
- [ ] Summary + commit/MR for approval (after the re-review)

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | spec-unclear | — | Ownership of step 3's parts is unstated when step 0 delegates. Step 0 delegates "steps 2 through 4a", while step 3 gives the host tool to the top-level session and the angles to angle reviewers, and says "steps 2, 4 and 4a go to one delegated reviewer". Three parts have no owner: the `low` one-pass hunt, the scoped re-review's one pass, and the rest of the quality audit. The checklist line now covers only the hunt, so the remaining quality checks have no line either. Read literally, an author-reviewer runs the one-pass hunt on its own diff, which is the self-review step 0 forbids. That is the common path once rework and re-review chain in one session (T-138). (H1, H3, H4, D1) | `review-protocol.md:76-82` vs `:213-216` and `:151-152`; checklist `:497` | Say once in step 3: without fan-out, step 3 (hunt and rest of the quality audit) goes to the step 0 reviewer; with fan-out, the rest of the quality audit goes to the reviewer of steps 2, 4 and 4a. Narrow step 0's "steps 2 through 4a" to point at it. The checklist line names the quality audit as well as the hunt. |
| F2 | blocking | correctness | — | The host tool path drops the security audit. The old step 3 bullet "Error handling, edge cases, security (input validation, secret handling, injection)" is gone, and the tool replaces the angle hunt, including angle 7. A host whose tool reviews for correctness only (some hosts ship a separate security review) leaves no step that looks for an injectable input or a logged secret. Before this branch, every review checked them. (H2) | `review-protocol.md:182-183` ("does not run alongside it"), `:225-230` (the rest of the quality audit has no security bullet); `git show main:skill/resources/review-protocol.md` step 3 | Keep angle 7 in the rest of the quality audit, run on either path, or require it whenever the host tool does not cover security. |
| F3 | blocking | spec-unclear | — | A scoped re-review is told to run "the host tool" over this round's fix diff only. The tool path is written for `<base>...<branch>`, and host tools commonly take a branch, a PR or the working tree, not a commit or a two-dot range. The text gives no fallback, so a reviewer either re-audits the whole branch (breaking "The bound") or skips the hunt without recording why. (H8, D7) | `review-protocol.md:151-152` vs `:181-182` | If the tool cannot take the fix range, run the angle hunt over it as one pass, and record which path ran. |
| F4 | blocking | spec-unclear | — | Step 3 does not say how to class a bug that a confirmed decision mandates. Both replays found such bugs and deferred to the plan: smppai F3 (R3, ack-and-drop) was classed `design` because Task 5 asked for it; porth A2 and C1 note "the defect is in the plan" and "the gate accepted it". This is how SMP-007's R1–R3 shipped, so leaving it unaddressed defeats this ticket's Outcome. Angle 8 (conformance) pushes the reviewer toward deferring. (D2) | `tickets/retros/2026-10-01-t137-replay-smppai.md` F3; `…-replay-porth.md` A2, C1 | One line under the angles: wrong behaviour stays `correctness` and is classed on its merits even when a decision mandates it; cite `<ID> decision <N>`, and when the decision is the cause, say so (the reserved `plan-wrong` class). |
| F5 | non-blocking | design | noted | Angle 1 (caller contracts) overlaps step 4's "caller ↔ callee contract drift", and angle 8 overlaps step 2's "confirmed decisions honoured". Under fan-out, different agents run each copy. The overlap is deliberate: step 3 reads for behaviour, steps 2 and 4 for presence and consistency. (D3, and the gate's G11) | `review-protocol.md:189-190`, `:203-204` vs step 2 and `:236` | Revisit if reviews record the same finding twice. |
| F6 | non-blocking | spec-unclear | fixed inline | The `SKILL.md` summary said "each bug reproduced", dropping the `unreproduced:` alternative of decision 7. The edited lines in `SKILL.md` and `tickets-README.md` ran to 116 and 123 columns against the files' ~100. (D4) | `skill/SKILL.md:255-256`; `tickets-README.md:400` | Fixed in `709aa11`: "reproduced or traced", both rewrapped. |
| F7 | non-blocking | stale-xref | fixed inline | The host-command rule's comment said a generic pattern would match `resources/review-protocol.md`. It would not, since that slash follows a word character. The real reason for the literal is that a generic pattern flags 11 legitimate lines in `agents/pi/`, which ship only to pi and name pi's commands. (H7) | generic `(^|[^\w./-])/[a-z][\w-]*\b` over the payload: 11 hits, all `agents/pi/extensions/*.ts` | Fixed in `709aa11`. |
| F8 | non-blocking | test-gap | fixed inline | `TestPayloadLintRulesLeaveLegitimateShapesAlone` claims every line is drawn from the live payload, but the new line was invented wording. `TestPayloadLintRuleHostCommand` checked a code span but not a fenced block, so a fence exemption could be added unnoticed. (H9, D6) | `payload_lint_test.go:351`, `:463-481` | Fixed in `709aa11`: pins "provides a code-review tool or command that audits a diff for", adds a fenced case. |
| F9 | non-blocking | stale-xref | fixed inline | `AGENTS.md` said the lint enforces "the mechanical part of this paragraph" and listed the host command name among it, but the paragraph's shapes do not include one. The lint's failure header ("as if the reader were standing in pickle's own repo") misdescribed a host-command hit. (D5) | `AGENTS.md:53-56`; `payload_lint_test.go:288-289` | Fixed in `709aa11`: the shape is named as one the paragraph does not list, and the header is rule-neutral. |
| F10 | non-blocking | design | noted | The host-command rule matches only the literal `/code-review`; `/security-review`, `/review` and path-shaped mentions pass. Deliberate (decision 2, G1): a generic pattern flags `agents/pi/` (F7). (H6) | `payload_lint_test.go:191` | Add another literal when one reaches the payload. |

Disposition summary: 4 blocking (F1–F4 → rework); 4 fixed inline (F6–F9, commit `709aa11`), 2 noted (F5, F10), 0 folded, 0 new tickets. Host tool: 9 findings, 1 discarded on re-verification.
cost: estimated M, actual M

### Rework fix record — round 1 (commit 7234ff3)

Branch rebased onto `main` @ `c9a3bba` (`pickle doctor` stale-ticket warning); tip before the
fix `e4f4c46`.

- **F1** — step 3 gains *Who runs what*: the top-level session runs the host tool and spawns
  fanned-out angle reviewers; everything else in step 3 (a one-pass hunt for `low`, a scoped
  re-review or a host that cannot spawn, angle 7 on the tool path, the rest of the quality audit)
  goes to whoever runs steps 2, 4 and 4a. Step 0's *Trigger* names the two exceptions and points
  there. The checklist line now reads `Correctness hunt (step 3), with the rest of its quality
  audit:` and asks who ran it (the acceptance grep for `Correctness hunt (step 3)` still matches).
- **F2** — the host tool path keeps angle 7 (security) whenever the tool does not review for
  security; the checklist asks whether it ran separately; `lifecycle.adoc` says so in one clause.
- **F3** — step 1's *The hunt*: the host tool only if it can take this round's range, otherwise the
  angle hunt as one pass; record which ran.
- **F4** — new paragraph after the angles, *A decision does not excuse a bug*: plan-mandated wrong
  behaviour stays `correctness`, severity on the behaviour, decision cited as `<ID> decision <N>`
  so the rework knows it needs the user's sign-off; not `plan-wrong`, because the class records
  what ships. (The finding's suggestion offered `plan-wrong` for this case; one class per row, so I
  picked `correctness` to avoid two classes competing for the same bug.)

Acceptance test re-run verbatim: green. `just build`, `just test`, `just lint` (actionlint and
shellcheck not installed, skipped) and `just docs-check` clean.

### Scoped re-review — round 1

Re-reviewed 2026-10-01 on `feat/T-137-review-correctness-hunt` @ `7234ff3`. Scope: F1–F4 plus the
fix diff `7234ff3` (`git show 7234ff3`).

- [x] Reviewer independence settled (step 0): **independent**. This conversation started after a
  context clear and holds no memory of writing the branch or the fix; the audits ran here.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` clean on the branch.
- [x] Implementation audit (steps 1, 2): F1–F4 closed (below). Acceptance test re-run verbatim:
  green; `just build`, `just test`, `just lint` (actionlint, shellcheck not installed) and
  `just docs-check` clean. The retro script and both replay tables are on `main`.
- [x] Correctness hunt (step 3): **angle hunt**, one pass, no sub-agents, run by this session. The
  host's code-review tool takes a branch, PR or path, not a commit range, so per step 1 it fell
  back to the hunt. Findings: R1 (angle 8, the class rule against rules §1), R2 (angle 8,
  decision 1). No `correctness` rows.
- [x] Consistency audit (step 4): `plan-wrong`, "steps 2 through 4a", "Who runs what" and the
  checklist line swept across the payload; T-138 Task 2 already defers to step 3 on `main`.
- [x] Documentation audit (step 4a): the `lifecycle.adoc` clause matches F2's fix; docs build clean.
- [x] Docs-readability pass (step 4b): conscious skip. No reviewer is configured in this session.
- [x] Findings recorded (step 5)
- [x] Ticket moved to `6-done/` (step 6b)
- [x] Other references and governing docs (step 7): rules §1 reconciled (R1); decision 1 amended
  (R2).
- [x] Impact sweep (step 8): T-138 unaffected; its note already says to re-read step 0 and step 3
  on `main`.
- [ ] Summary + commit/MR for approval (step 9)

F1 — closed: *Who runs what* gives the one-pass hunt, angle 7 on the tool path and the rest of the
quality audit to the reviewer of steps 2, 4 and 4a, and step 0 names the two exceptions. F2 —
closed: angle 7 runs when the tool does not review for security, and the checklist asks for it.
F3 — closed: the tool runs only if it takes the range, otherwise one-pass hunt, recorded. F4 —
closed: *A decision does not excuse a bug*.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | non-blocking | spec-unclear | fixed inline | F4's fix classes a bug that a decision mandated as `correctness` "rather than `plan-wrong`". Rules §1 classes the finding behind a retracted decision `plan-wrong`. Both rules claim the same row, and F2 is one: decision 1 mandated it and is now amended. | `review-protocol.md:205-212` vs `tickets-README.md:200-202` | Fixed in `7169b7a`, per the user's choice: §1 names the exception, and the row stays `correctness`. |
| R2 | non-blocking | plan-wrong | fixed inline | F2's fix runs angle 7 alongside the host tool. Decision 1 said the tool "does not run alongside" the hunt, and no `plan amended inline` line recorded the deviation. | decision 1 vs `review-protocol.md:185-187` | Decision 1 amended here, History line added. |

Disposition summary: 0 blocking; 2 fixed inline (R1 `7169b7a`, R2 plan amendment), 0 noted, 0 folded, 0 new tickets. Hunt fell back from the host tool (range unsupported).
cost: estimated M, actual M

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: a separate code-review pass was run after brine validate on 31 PRs, typically returning 7–10 findings, only 2 with no correctness bug
- 2026-10-01 — TO DO → READY: plan complete; host tool else 8-angle hunt, sub-agent budget, repro rule, replay acceptance
- 2026-10-01 — applicability gate (fresh sub-agent, vs main 18e3e7f): clean, 11 non-blocking. G1–G9 fixed inline (user-approved), G10–G11 noted
- 2026-10-01 — plan amended inline: decision 6 says angle reviewers run only step 3 and the top-level session runs the host tool; Task 1 detects runs as weekly.py does, adds a token baseline for the guard, is run by the user (transcript reads denied to the agent) and lands on main before the branch; Task 5 matches literal `/code-review` with its own test and updates the "four rules" counts; Task 6 pins a8379fd/b5ee3aa, confines replays to their worktree, judges raw output; Task 7 keys on the merge line; acceptance greps case-insensitive and specific
- 2026-10-01 — READY → IN DEVELOPMENT: picked up; applicability gate clean (11 non-blocking, G1–G9 fixed inline)
- 2026-10-01 — Task 1 baseline: 34 code-review runs after validate, regex floor 26% (9/34), 10 of 11 classifiable runs had ≥ 1 bug; premise check settled as holding by the user (floor below 30%, direct readings far above); target recorded in tickets/retros/2026-09-29-self-improvement-loop.md
- 2026-10-01 — Task 6 replay: pass bar met on both. smppai found R3 (as design, deferring to the plan), R5, R6; porth found POR-013 items 1, 3 (one row blocking) and 4, plus PR 7 finding 8; tables in tickets/retros/2026-10-01-t137-replay-*.md
- 2026-10-01 — IN DEVELOPMENT → IN REVIEW: acceptance green; replay pass bar met on smppai and porth
- 2026-10-01 — IN REVIEW → REWORK: 4 blocking (F1–F4: step 3 ownership, security on the host-tool path, re-review fix range, plan-mandated bugs); 4 fixed inline, 2 noted
- 2026-10-01 — REWORK → IN REVIEW: findings fixed
- 2026-10-01 — plan amended inline: decision 1 excepts angle 7 (security) when the host tool does not review for it — review F2's fix had already shipped this
- 2026-10-01 — IN REVIEW → DONE: scoped re-review: F1–F4 closed; 0 blocking, 2 fixed inline (R1, R2)
