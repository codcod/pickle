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
   reviewer: a fresh sub-agent with this skill's `resources/docs-readability.prompt.md` as its
   instructions, read-only and suggestions-only like the other backends. The dedicated backends
   stay the first choice where configured, since they pin a model and deny edit tools. The
   conscious skip then covers only a host with neither.
2. **Changed lines only.** The reviewer gets each changed file plus the ticket's diff hunks
   (`<base>...<branch>`) and suggests changes only inside them. The rule goes in
   `docs-readability.prompt.md`, which every backend reads, and 4b says what to pass. A
   suggestion outside the hunks is discarded the same way a misquoted one is (T-122's rule).
3. Update the user manual's description of the reviewer (`docs/user-manual/cli-reference.adoc`
   § the docs-readability reviewer) and the changelog.

Prose only, no Go change. Payload text must pass the foreign-workspace test and
`payload_lint_test.go`. In particular, name no agent host's review command, and do not single out
Claude Code: "a host that can start a sub-agent" covers it.

**Not in scope:** tightening the skip grounds against T-143's off-protocol skips. That happened
once, and the existing text already rules it out.

**Cost to weigh at refinement:** one extra model pass per review that changes `.md`/`.adoc`.
The step stays optional and never blocks.

Soft couplings: T-122 (quote verification, which the out-of-hunk discard extends), T-096 (pinned
backend model), T-144 (same review of the docs step).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-10-28 — created (TO DO). source: self-host: a month of Claude Code sessions showed step 4b ran in 0 of 12 review rounds, and earlier runs mostly flagged prose outside the diff
