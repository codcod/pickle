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

None. T-144 also edits the payload, but different files (`TEMPLATE.md`, `tickets-README.md`).
Both add a `CHANGELOG.md` `[Unreleased]` entry, so whichever merges second resolves a trivial
conflict there.

### Confirmed design decisions (do not deviate without asking)

1. **Dedicated backend first, host sub-agent as fallback.** Where a pi `docs_readability` tool or
   an opencode `docs-readability` subagent is configured and reachable, it runs. Otherwise a host
   that can start a sub-agent runs a fresh one with the shipped prompt. Only a host with neither
   takes the conscious skip. (User decision, refinement 2026-10-28.)
2. **Read-only is checked for the sub-agent, not trusted.** The reviewing agent records
   `git status --porcelain` before the run and compares it after. On any difference it restores
   the reviewer's changes (`git checkout -- <paths>`, and removes any new untracked files the run
   created) and discards the run as untrusted. That is the existing "output cannot be trusted"
   skip ground, not a new one. (User decision.)
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

<!-- empty until IN REVIEW -->

## History

- 2026-10-28 — created (TO DO). source: self-host: a month of Claude Code sessions showed step 4b ran in 0 of 12 review rounds, and earlier runs mostly flagged prose outside the diff
- 2026-10-28 — refined: user decisions — dedicated reviewer first with host sub-agent fallback, worktree-unchanged check for the sub-agent, changed-lines scope as a post-filter in 4b (no backend or prompt change)
- 2026-10-28 — TO DO → READY: plan complete
