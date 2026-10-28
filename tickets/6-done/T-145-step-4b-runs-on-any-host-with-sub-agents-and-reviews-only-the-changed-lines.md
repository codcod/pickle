---
id: T-145
title: Step 4b runs on any host with sub-agents and reviews only the changed lines
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-145 — Step 4b runs on any host with sub-agents and reviews only the changed lines

## Outcome

After this ships, a review on any host that can start a sub-agent, Claude Code included, runs
the docs-readability pass instead of skipping it. Its suggestions cover only the prose the ticket
changed, not the rest of each file.

## Description

Review protocol step 4b (`skill/resources/review-protocol.md`) runs only when the host provides
a dedicated docs-readability reviewer: the pi `docs_readability` tool, the opencode
`docs-readability` subagent, or `opencode run --agent docs-readability`. Its examples list no
other host, so a Claude Code session has no sanctioned way to run it.

**Evidence (Claude Code sessions, 2026-09-29 to 2026-10-28).** There were 12 review rounds across
T-134, T-135, T-137, T-138, T-140, T-141, T-142 and T-143, and 4b ran in none of them:
- 9 rounds were recorded as "no reviewer configured/reachable": `opencode` is not installed, and
  the pi tool works only inside a pi session.
- 3 rounds (all T-143) skipped on grounds the step does not sanction: "short reference
  passages", and "a later rework revisits the wording".

When it did run in earlier reviews, most suggestions targeted prose the ticket had not touched,
and were discarded or noted:
- T-009: about 20 suggestions, all but three outside the diff, all discarded.
- T-123: 9 and then 11 suggestions on untouched prose, noted.
- T-127: one in-diff suggestion applied, the rest discarded as out of scope.

**Scope.**
1. **Any host with sub-agents runs it.** In 4b, add the host's own sub-agent as a sanctioned
   fallback reviewer: a fresh sub-agent with this skill's `resources/docs-readability.prompt.md`
   as its instructions. The dedicated backends (pi tool, opencode subagent) stay the first choice
   where they are configured and reachable, since they pin a model and enforce read-only access.
   A sub-agent's read-only status is checked instead: the worktree must be unchanged after the
   run. The conscious skip then covers only a host with neither.
2. **Changed lines only, by post-filter.** The reviewing agent already locates every quoted
   passage in its file to verify it. It now also discards any suggestion whose passage lies
   outside the branch's diff hunks (`git diff <base>...<branch> -- <file>`), on the same terms as
   a misquote. No backend changes: the pi tool takes file paths only and cannot receive a diff,
   so the filter sits with the agent that holds the diff.
3. Update the user manual's description of the reviewer (`docs/user-manual/cli-reference.adoc`,
   the *The docs-readability reviewer* paragraph) and the changelog.

Prose only, no Go or TypeScript change. Payload text must pass the foreign-workspace test and
`payload_lint_test.go`. In particular, name no agent host's review command, and do not single out
Claude Code: "a host that can start a sub-agent" covers it.

**Not in scope:** tightening the skip grounds against T-143's off-protocol skips. That happened
once, and the existing text already rules it out.

**Cost to weigh at refinement:** one extra model pass per review that changes `.md`/`.adoc`.
The step stays optional and never blocks.

Soft couplings: T-122 (quote verification, which the out-of-hunk discard extends), T-096 (pinned
backend model), T-144 (same review of the docs step).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-145-docs-readability-subagent
```

Work on this branch, committing locally (WIP commits encouraged). Never push or open a merge
request without explicit user approval. This is a root-path child (`path = "."`), so tidy the WIP
commits into atomic ones before the Finish summary, and keep that history on merge. Before
pushing, `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`
must print nothing.

### Prerequisite gate (hard)

None. T-144 (merged) edited different payload files; its `CHANGELOG.md` `[Unreleased]` bullet is
already on main, so Task 3 appends beside it.

### Confirmed design decisions (do not deviate without asking)

1. **Dedicated backend first, host sub-agent as fallback.** Where a pi `docs_readability` tool or
   an opencode `docs-readability` subagent is configured and reachable, it runs. Otherwise a host
   that can start a sub-agent runs a fresh one with the shipped prompt. Only a host with neither
   takes the conscious skip. (User decision, refinement 2026-10-28.)
2. **Read-only is checked for the sub-agent, not trusted.** The worktree must be clean
   (`git status --porcelain` empty) before the sub-agent starts — commit or stash first, or take
   the conscious skip — so any output after the run is the sub-agent's alone. On any output it
   restores those changes (`git checkout -- <paths>`, and removes any new untracked files the run
   created) and discards the run as untrusted. That is the existing "output cannot be trusted"
   skip ground, not a new one. (User decision; clean-start precondition added at pickup.)
3. **Changed-lines scope is a post-filter in 4b, not a prompt or backend change.** After the
   verbatim-quote check, a suggestion whose located passage falls wholly outside the file's
   diff hunks against the base is discarded. A passage overlapping a hunk is kept. The count of
   such discards is recorded on the 4b checklist line, next to the fabricated-quote count.
   `docs-readability.prompt.md`, `agents/pi/extensions/docs-readability.ts` and
   `agents/opencode/opencode.jsonc` are not touched. (User decision.)
4. **The sub-agent is described generically.** The payload says "a host that can start a
   sub-agent". It names no host product, and no review command of any host
   (`payload_lint_test.go` rule).
5. **4b stays optional and never blocks.** Suggestions are still not findings and never enter the
   findings table.

### Tasks

#### Task 1 — `skill/resources/review-protocol.md` § 4b

- First paragraph: keep the dedicated-backend examples. Add the fallback per decision 1: give a
  fresh sub-agent the shipped prompt as its instructions, and list the changed `.adoc`/`.md`
  files for it to read. State the worktree check from decision 2 in one or two sentences.
- Second paragraph (the optional/skip paragraph): the conscious skip's first ground becomes
  "neither a dedicated reviewer nor a sub-agent is available".
- Verification paragraph: after the verbatim check, add the out-of-hunk discard from decision 3,
  and extend the "record how many … discarded" sentence to cover both counts.
- The step-9 checklist template's 4b line: ask for the out-of-hunk discard count alongside the
  fabricated-quote count, and for which backend ran (dedicated or sub-agent).
- Keep the edit tight: the section is already long. Prefer adding clauses to existing sentences
  over adding new paragraphs, and keep the file's wrap width.

#### Task 2 — `docs/user-manual/cli-reference.adoc`, the *The docs-readability reviewer* paragraph

- Replace "Sessions in other agents (zed, Claude Code, …) can reach the opencode reviewer by
  shelling out … — or record a conscious skip." with: other sessions can still shell out to
  opencode where it is configured; otherwise a host that can start a sub-agent runs one with the
  shipped prompt (the worktree is checked to be unchanged afterwards); only a host with neither
  records a conscious skip. Naming Claude Code is fine here: the user manual is not payload.
- Add that suggestions outside the ticket's changed lines are discarded, next to the existing
  sentence about verbatim quotes.

#### Task 3 — `CHANGELOG.md`

`## [Unreleased]` → `### Changed`: one bullet ending `(T-145)`. It says the docs-readability pass
(review step 4b) now falls back to a host sub-agent instead of skipping, and keeps only
suggestions on the lines a ticket changed.

### Acceptance test

```
just build && just test && just lint && just docs-check          # all green; just test runs payload_lint_test.go
grep -c "sub-agent" skill/resources/review-protocol.md           # expect > 9 (main has 9; 4b now names the fallback)
sed -n '/^## 4b\./,/^## 5\./p' skill/resources/review-protocol.md | grep -n -i -E "sub-agent|porcelain|hunk"
                                                                 # expect ≥ 3 lines: fallback, worktree check, out-of-hunk discard
git diff --stat main...HEAD -- skill/resources/docs-readability.prompt.md agents/   # expect empty (decision 3)
grep -rn -i "Claude Code" skill/                               # expect no output (main has none; decision 4)
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q && ./pickle-test install --agent claude >/dev/null \
  && sed -n '/^## 4b\./,/^## 5\./p' .agents/skills/brine/resources/review-protocol.md | grep -c -i hunk; cd -
                                                                 # expect ≥ 1: the installed payload carries the new 4b
```

Behavioural check, recorded in the review: the review of this ticket runs 4b itself through the
new sub-agent fallback (in a session without opencode or pi), with the worktree check and the
post-filter. It records both discard counts on the checklist line.

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`, the *The docs-readability reviewer* paragraph (Task 2).
- `CHANGELOG.md` `[Unreleased]` → `### Changed` (Task 3).
- `just docs-check` clean.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (Task 2, Task 3).
3. Tidy the WIP commits into atomic ones, then write the summary: files touched, decisions
   honoured, anything deferred.
4. Suggest the commit message, e.g.:

   ```
   feat(skill): run docs-readability via a host sub-agent and keep only changed-line suggestions (T-145)

   Step 4b skipped in every Claude Code review for lack of a dedicated reviewer, and earlier
   runs mostly flagged prose outside the diff. A host sub-agent with the shipped prompt is now
   the fallback, its read-only status is checked against the worktree, and suggestions outside
   the branch's diff hunks are discarded after quote verification.
   ```

## Review

### Round 1 — 2026-10-28

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | correctness | fixed inline | The 4b read-only check missed a sub-agent that commits (porcelain stays empty) or stages (`git checkout -- <paths>` restores from the index, which still holds the edit). Reproduced in a scratch repo. | `skill/resources/review-protocol.md` 4b, paragraph 1 | Done: note `git rev-parse HEAD` before the run; on any status output or a moved HEAD, `git reset --hard <noted HEAD>`. Extends decision 2 within its intent (the clean start makes the reset lossless). |
| F2 | non-blocking | stale-xref | fixed inline | `.opencode/README.md` still called reviewing without the opencode reviewer a sanctioned conscious skip, which this branch made false. | `.opencode/README.md:79-82` | Done: names the sub-agent fallback and points to step 4b. |
| F3 | non-blocking | spec-unclear | fixed inline | `git diff <base>...<branch>` hunks carry 3 context lines, so "wholly outside the hunks" kept suggestions on unchanged context, while the manual and changelog promise changed lines only. | `review-protocol.md` 4b verification paragraph | Done: `git diff -U0`. |

Dispositions: 0 blocking; fixed inline 3 (F1–F3); noted 0; new ticket 0; folded 0.
cost: estimated S, actual S

- [x] Reviewer independence settled (step 0): delegated. A fresh sub-agent ran steps 1–4a with `hunt: one pass`. All 3 of its findings were re-verified by hand (F1 reproduced in a scratch repo) and kept; none discarded.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned that the branch had the ticket in `3-in-development`; the branch was rebased onto `main`, and doctor then reported 0 warnings.
- [x] Implementation audit (steps 1, 2): acceptance test re-run verbatim and green (`just build && just test && just lint && just docs-check`; `sub-agent` count 12; 4b grep 5 lines, 6 after the fixes; prompt and `agents/` untouched; no "Claude Code" in `skill/`; installed payload carries the new 4b). Tasks 1–3 and decisions 1–5 verified, including the gate's amendment.
- [x] Correctness hunt (step 3), with the rest of its quality audit: one-pass angle hunt by the delegated reviewer, no host tool, no sub-agents, angle 7 in the same pass. F1 reproduced. `payload_lint_test.go` passes; foreign-workspace test holds by reading.
- [x] Consistency audit (step 4): repo-wide grep for `docs-readability`, `step 4b`, `conscious skip`; F2 came from it. `skill/SKILL.md` ("both installed backends … read this one file") stays true.
- [x] Documentation audit (step 4a): user manual and `CHANGELOG.md` updated; docs-tree sweep clean; `just docs-check` green.
- [x] Docs-readability pass (step 4b): backend: host sub-agent (no pi tool or opencode here), the ticket's own behavioural check. Clean worktree and HEAD `c9cf1dc` noted before; status empty and HEAD unchanged after, so the run was trusted. 14 rows over the 4 changed files: 2 were "no change" notes, 1 discarded as fabricated (`.opencode/README.md` "(also what the docs-readability reviewer uses)" not in the file), 4 discarded as outside the diff (`.opencode/README.md:67`, `CHANGELOG.md:53`, `cli-reference.adoc:1570`, `review-protocol.md:52`); the 7 in-diff suggestions were presented and all approved by the user and applied.
- [x] Findings recorded with severity, class and disposition (step 5).
- [x] Ticket moved to `tickets/6-done/` (step 6).
- [x] Other references (step 7): `AGENTS.md`/`CLAUDE.md` do not restate step 4b, so no governing document needed reconciling. The board was regenerated by the move.
- [x] Impact sweep (step 8): no open ticket depends on or cites T-145.

## History

- 2026-10-28 — created (TO DO). source: self-host: a month of Claude Code sessions showed step 4b ran in 0 of 12 review rounds, and earlier runs mostly flagged prose outside the diff
- 2026-10-28 — refined: user decisions — dedicated reviewer first with host sub-agent fallback, worktree-unchanged check for the sub-agent, changed-lines scope as a post-filter in 4b (no backend or prompt change)
- 2026-10-28 — TO DO → READY: plan complete
- 2026-10-28 — READY → IN DEVELOPMENT: picked up
- 2026-10-28 — plan amended inline: applicability gate (fresh sub-agent, 3 non-blocking findings, routing approved by user) — decision 2 gains a clean-worktree precondition so the check cannot miss edits or wipe the reviewer's work; Task 1 also updates the checklist template's 4b line for both discard counts and the backend used; prerequisite note updated since T-144 merged (note-and-close)
- 2026-10-28 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-10-28 — IN REVIEW → DONE: review round 1: 0 blocking, 3 fixed inline
- 2026-10-28 — published: user approved; branch `feat/T-145-docs-readability-subagent` pushed and PR #104 opened (https://github.com/codcod/pickle/pull/104); merge is the human's
