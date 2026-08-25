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

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the
      branch in this same session, so audits 2–4a ran in a freshly spawned independent reviewer,
      briefed adversarially with the ticket (read from `main`), the branch, and the child's
      configured commands. Every finding it returned was re-verified by hand before entering the
      table below; all five reproduced exactly as reported.
- [x] Implementation audit — acceptance test re-run verbatim (all 5 steps), tasks & criteria
      verified (step 2)
- [x] Quality audit (step 3) — including verifying the new claims against `internal/install/`
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.md` files (step 4b) — run; suggestions triaged,
      one in-scope suggestion overlapped F1/F5 and was folded into that inline fix, the rest
      target prose this branch never touched and were discarded as out of scope
- [x] Findings recorded with severity, class and disposition; disposition summary + cost line
      present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket depends on or assumes anything
      from T-119; the only reference is T-121's lineage, filed by this review
- [x] Summary + commit message & MR attributes presented for approval; publish pending —
      approved, pushed, PR opened (#73) and merged

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | docs-gap | fixed inline | The new sentence said naming `claude` adds "the `CLAUDE.md` **marker block**", but `--claude-symlink` makes `CLAUDE.md` a symlink to `AGENTS.md` *instead* of a marker block — so the branch's own replacement was wrong on that path | `internal/install/install.go` — `if opts.ClaudeLink { ensureSymlink(CLAUDE.md → AGENTS.md) } else { injectMarker(CLAUDE.md…) }`; caveat documented at `docs/user-manual/cli-reference.adoc` § `--agent` | Say `CLAUDE.md` rather than "the `CLAUDE.md` marker block" — true on both paths, and does not re-derive the manual's artifact table (decision 1). **Fixed in `01c0348`** |
| F2 | non-blocking | docs-gap | new ticket | The **generated `AGENTS.md` marker block** states a Claude-only artifact unconditionally — the exact shape Task 2 fixed in the payload. Every install writes "Claude Code sees it via `.claude/skills/brine`" regardless of the agent set | `internal/install/install.go`, the `MarkerBlock` text; reproduced in the throwaway `install --agent opencode`, whose `AGENTS.md` carries the sentence beside no `.claude/` directory | Out of scope here — decision 5 bars touching `internal/`. Batched with F3 into **T-121** |
| F3 | non-blocking | docs-gap | new ticket | `pickle help` still condenses install as "inject AGENTS.md/CLAUDE.md markers", unconditional in the same way; T-013 fixed this line's autodetection half but left the pairing | `internal/cli/cli.go`, the `install` usage lines | Same theme as F2; batched into **T-121** |
| F4 | non-blocking | spec-unclear | noted | `resources/review-protocol.md`'s opening tells the reviewer to read "`AGENTS.md`/`CLAUDE.md` first" — the only other payload site pairing the two. Reads as alternatives rather than a claim both exist, so materially weaker than F2 | `skill/resources/review-protocol.md`, step-0 preamble | Genuine but slight; the slash reads as "whichever your agent uses". Recorded and closed — promotable later by citing this row |
| F5 | non-blocking | design | fixed inline | The rewrite left a ragged wrap — two lines at 71 and 58 columns inside a paragraph otherwise wrapped at 88–97, against Task 1's "keep the existing 100-ish-column wrap" | `skill/SKILL.md`, the *Install & register* paragraph as first written | Re-flow the paragraph. **Fixed in `01c0348`**; paragraph now wraps 88–98 throughout |

**Disposition summary:** 5 non-blocking, 0 blocking — 2 fixed inline (F1, F5, both prose this
branch authored), 2 → new ticket **T-121** (F2, F3, batched as one theme), 1 noted (F4).

```
cost: estimated S, actual S
```

**Verdict: passed.** All four tasks met, all six confirmed design decisions honoured, all five
acceptance-test steps reproduced verbatim by an independent reviewer — including the throwaway
`--agent opencode` install, which confirmed live that the shipped skill now states the contract
the binary actually implements. `just build`, `just test`, `just lint`, `just docs-check` all
clean, re-run after the inline fixes. The new prose passes AGENTS.md's foreign-workspace test
both mechanically (`payload_lint_test.go`) and on the judgement the test cannot make.

## History

- 2026-08-24 — created (TO DO). source: review: T-013's review, non-blocking finding N9
  (disposition: new ticket). The payload half of the autodetection claim T-013 removed from
  `pickle help`; pre-existing, so ineligible for `fixed inline` under rules §5
- 2026-08-24 — TO DO → READY: plan complete
- 2026-08-25 — READY → IN DEVELOPMENT: picked up
- 2026-08-25 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-25 — IN REVIEW → DONE: review clean; 5 non-blocking, all dispositioned (2 fixed inline, 2 -> T-121, 1 noted)
- 2026-08-25 — merged to main (PR #73, 11a2567, https://github.com/codcod/pickle/commit/11a2567; merge commit 43c7e20)
