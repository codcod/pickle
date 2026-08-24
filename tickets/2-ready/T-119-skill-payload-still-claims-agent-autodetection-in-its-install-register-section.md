---
id: T-119
title: skill payload still claims agent autodetection in its Install & register section
project: pickle
depends-on: []
spawned-by: [T-013]
impact: low
complexity: low
cost: S
---

# T-119 — skill payload still claims agent autodetection in its Install & register section

## Outcome

After this ships, the brine skill every installed project reads no longer tells the agent that
`pickle install` detects which agents are present. The skill, the binary's `pickle help`, and the
user manual then all say the same thing: the agent set is exactly what `--agent` names, defaulting
to `claude` — and the artifacts only a named agent produces (the `.claude/skills/` symlink, the
`CLAUDE.md` marker block) are described as the conditional things they are.

## Description

`skill/SKILL.md`'s *Install & register* section states that `pickle install` "installs this skill
for the **detected agents**". There is **no autodetection** — that was a locked decision of T-009,
and the set is exactly what `--agent` names (default `claude`).

This is the same false sentence T-013 removed from the binary's top-level `pickle help` text
(T-013 item 4, itself a folded T-009 review finding). T-013 fixed the `internal/cli` occurrence
only; this one lives in the **payload**, which is the copy every installed project actually reads,
and which `pickle upgrade` re-installs verbatim.

The claim already contradicts the user manual, which states plainly that there is *no
autodetection* and that the set is exactly what you name. So the tree currently disagrees with
itself across two surfaces out of three.

**Why this is its own ticket rather than a fix inside T-013.** The defect is *pre-existing* — it
was not introduced or falsified by T-013's branch — and the flow's disposition rules bar the
`fixed inline` route for exactly that case ("did this branch break it?", not "is it small?").
T-013's own Task 2 was scoped to one named file.

Scope is one sentence, plus a check for the same claim elsewhere in the payload. Note that the
whole payload is read by projects that are not pickle, so the replacement wording must stand on
its own for a foreign workspace.

**Verified at refinement (2026-08-24).** The payload has exactly one autodetection claim — the
*Install & register* sentence — and the sweep turned up one adjacent defect of the same shape: the
*Notes* section states the Claude Code symlink unconditionally, though it is `--agent claude` that
lays it down. The same conditional applies to the `CLAUDE.md` half of the marker block named in
the very sentence being fixed. Both are folded in: the replacement sentence has to say *something*
about those artifacts, and saying it wrongly is the defect this ticket exists to remove. The
authoritative statement remains `docs/user-manual/cli-reference.adoc` § "`--agent` — which coding
agents to wire up"; the payload should restate its contract, not re-derive it.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is a root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-119-payload-agent-autodetection
```

WIP commits are encouraged; tidy them into atomic commits before presenting (root-path child —
keep the tidied history rather than squashing). Do not push or open an MR without explicit user
approval. Under `layout = "in-tree"`, before pushing verify the remote base is not behind:
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing.

### Prerequisite gate (hard)

None. `depends-on:` is empty and `spawned-by: [T-013]` is lineage only. T-013 is already in
`6-done/`, but nothing here waits on it: this ticket edits the payload, which T-013's branch never
touched.

Start from a clean tree on an up-to-date `main`.

### Confirmed design decisions (do not deviate without asking)

1. **The payload states the `--agent` contract, it does not re-derive it.** The user manual's
   `--agent` section is the authority ("There is *no autodetection* — the set is exactly what you
   name", default `claude`). The payload says the same thing in one clause and does not enumerate
   the per-agent artifact table, which would be a second copy to drift.
2. **The replacement names no agent-specific behaviour as unconditional.** Every artifact that
   exists only because an agent was named — the `.claude/skills/` symlink, the `CLAUDE.md` marker
   block — is written as conditional on that agent being in the set. This is what makes the new
   sentence true rather than merely less false.
3. **The *Notes* section is fixed in the same commit.** Its "symlinked for Claude Code" is the same
   unconditional claim one section apart; leaving it would mean shipping a payload that
   contradicts itself two paragraphs after the fix.
4. **No new lint rule or regression test.** `payload_lint_test.go` has a stated charter — the four
   foreign-workspace rules — and an autodetection check is a different category of claim (false
   about pickle, not unresolvable for a foreign reader). Widening that file to "any wrong sentence
   in the payload" is a design change this ticket is not the place for. The acceptance test greps
   instead, once.
5. **No behaviour change.** No file under `internal/`, no flag, no output string. If the fix wants
   a code change, stop and ask — the ticket is wrong, not the code.
6. **The `{skill-dir}` docs attribute is not introduced into the payload.** `.agents/skills/brine/`
   is written literally, as the payload already does in both places; the attribute is an AsciiDoc
   mechanism that does not exist in a markdown skill file.

### Tasks

#### Task 1 — rewrite the *Install & register* sentence (`skill/SKILL.md`)

The first paragraph of *Install & register* currently reads (one sentence, wrapped):

> `pickle install` (run once in the overarching project) creates `tickets/` … writes the
> `tickets/README.md` pointer, **installs this skill for the detected agents**
> (`.agents/skills/brine/`, **symlinked into `.claude/skills/` for Claude Code**), **injects the
> `AGENTS.md`/`CLAUDE.md` marker block**, and writes `pickle.toml` — …

Replace the three bolded fragments so the paragraph reads (proposed wording; keep the surrounding
clauses and the layout/child-registration sentences that follow it untouched):

> … writes the `tickets/README.md` pointer, installs this skill into `.agents/skills/brine/`,
> injects the `AGENTS.md` marker block, and writes `pickle.toml` — recording the **layout** there
> (`umbrella` by default, or `in-tree` with `--in-tree`; rules §0). Which coding agents it wires
> up is exactly the set `--agent` names (default `claude`) — **there is no autodetection**; naming
> `claude` is what adds the `.claude/skills/brine` symlink and the `CLAUDE.md` marker block.

Keep the existing 100-ish-column wrap and the file's markdown emphasis conventions.

#### Task 2 — fix the *Notes* section's unconditional symlink claim (`skill/SKILL.md`)

The last section reads "installs this skill **per project** (into `.agents/skills/brine/`,
symlinked for Claude Code)". Make the parenthetical conditional — e.g. "(into
`.agents/skills/brine/`, with the `.claude/skills/` symlink when `--agent` includes `claude`)".
The rest of the sentence (no global install, per-project payload version, what `upgrade`
refreshes) is correct and stays.

#### Task 3 — sweep the rest of the embedded payload

Both embedded roots are in scope, not just the skill: `assets.go` embeds `all:skill` **and**
`all:agents`. Grep both for the claim and for any other unconditional per-agent statement:

```
grep -rni "detect\|autodetect" skill/ agents/
grep -rni "claude\|opencode\|\bpi\b" skill/ agents/
```

At refinement the first grep hit only `skill/SKILL.md:53`; the second's only unconditional hits
were the two sentences Tasks 1–2 fix (plus `skill/resources/review-protocol.md`'s
docs-readability step, which is already written conditionally — "any other session can shell out
to …" — and needs no change). Fix anything new the greps surface, or record in the summary that
they surfaced nothing.

#### Task 4 — CHANGELOG entry

Add a bullet to `## [Unreleased]` → `### Fixed` in `CHANGELOG.md`, directly after the existing
"`pickle help` no longer claims agent autodetection that does not exist" entry it completes.
State that the same claim also lived in the shipped skill payload — the copy every installed
project reads and `pickle upgrade` re-installs — and that the payload now also stops presenting
the Claude-only artifacts as unconditional. End with `(T-119)`.

### Acceptance test

Run from a clean tree on the feature branch.

1. **The claim is gone from the payload, and the negation is the only survivor:**

   ```
   grep -rni "detected agents" skill/ agents/          # expect: no output
   grep -rn "no autodetection" skill/SKILL.md          # expect: exactly 1 line
   ```

2. **Nothing states a Claude-only artifact unconditionally:** read the *Install & register*
   paragraph and the *Notes* paragraph. Each mention of `.claude/skills/` or `CLAUDE.md` is
   qualified by the agent set. (Judgement check, not a grep — record the verdict in the summary.)

3. **Only the payload and the CHANGELOG moved:** `git diff --name-only main...HEAD` prints exactly
   `CHANGELOG.md` and `skill/SKILL.md` (plus any further payload file Task 3 legitimately
   surfaced). No `internal/` path (decision 5).

4. **The child's configured commands are clean:**

   ```
   just build && just test && just lint && just docs-check
   ```

   `just test` includes `payload_lint_test.go`, so the new wording is checked against the
   foreign-workspace rules by the build itself.

5. **The new sentence is true against the binary** — a throwaway install, per the self-modify
   policy in `AGENTS.md` (never run the WIP binary against this repo; the test binary is always
   named `pickle-test`):

   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q .
   ./pickle-test install --agent opencode
   grep -n "no autodetection" .agents/skills/brine/SKILL.md   # the fixed sentence shipped
   ls -d .claude 2>/dev/null                                  # expect: no such directory
   test -f CLAUDE.md && echo UNEXPECTED || echo "no CLAUDE.md — as the payload now says"
   ```

   This is the point of the ticket made executable: on an install that did not name `claude`,
   neither Claude artifact exists, and the shipped skill no longer says otherwise.

### Docs update (mandatory when user-facing)

The payload *is* the user-facing surface here, so Tasks 1–2 are the docs change.
`docs/user-manual/cli-reference.adoc` § "`--agent` — which coding agents to wire up" is already
correct and stays the authority (decision 1) — re-read it while writing Task 1 and, if the payload
ends up saying anything the manual does not, change the payload, not the manual. `CHANGELOG.md`
is Task 4. No other doc mentions the claim.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated (the payload itself) and the CHANGELOG entry added.
3. Write a summary: the exact before/after of each sentence changed, what the Task 3 greps
   surfaced, and the throwaway-install output from acceptance step 5.
4. Suggest a Conventional Commit message, e.g.:

   ```
   docs(skill): drop the agent-autodetection claim from the payload (T-119)

   The Install & register section told every installed project that install
   detects which agents are present. It does not — the set is exactly what
   --agent names, defaulting to claude (T-009 decision 1). Same sentence T-013
   removed from `pickle help`, in the copy upgrade re-installs. The Claude-only
   artifacts (.claude/skills/ symlink, CLAUDE.md marker block) are now stated as
   conditional on that agent being named, in both places the payload mentions
   them.
   ```

5. Root-path child: interactive-rebase the WIP commits into atomic commits before presenting.
6. Commit locally; present the message for approval. Only after approval, verify
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing, then push and open the MR. Merging is the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-24 — created (TO DO). source: review: T-013's review, non-blocking finding N9
  (disposition: new ticket). The payload half of the autodetection claim T-013 removed from
  `pickle help`; pre-existing, so ineligible for `fixed inline` under rules §5
- 2026-08-24 — TO DO → READY: plan complete
