---
id: T-121
title: install's generated AGENTS.md marker block and help text state Claude-only artifacts unconditionally
project: pickle
depends-on: []
spawned-by: [T-119]
impact: medium
complexity: low
cost: S
---

# T-121 — install's generated AGENTS.md marker block and help text state Claude-only artifacts unconditionally

## Outcome

After this ships, a project installed without `claude` in its agent set no longer finds its own
generated `AGENTS.md` telling it that "Claude Code sees it via `.claude/skills/brine`" — a
directory that install never created there. `pickle help` stops describing the `CLAUDE.md` marker
as something every install writes.

## Description

T-119 removed the agent-autodetection claim and the unconditional Claude-symlink claim from the
**skill payload**. Two surfaces outside the payload still have the same defect — they state an
artifact that only `--agent claude` produces as though every install produced it. T-119's
confirmed decision 5 explicitly barred it from touching `internal/`, so both were left for this
ticket.

1. **The generated `AGENTS.md` marker block.** `internal/install/install.go` (the `MarkerBlock`
   text, at the line reading ``"  (`resources/review-protocol.md`). Claude Code sees it via
   `.claude/skills/brine`.\n"``) writes that sentence into every project's `AGENTS.md`,
   regardless of the agent set. Reproduced during T-119's review: a throwaway
   `install --agent opencode` produced an `AGENTS.md` carrying the Claude sentence next to no
   `.claude/` directory at all. This is the sharper of the two — it is generated *into the user's
   own repo*, and `pickle upgrade` refreshes it, so it persists.

2. **`pickle help`.** The `install` summary in `internal/cli/cli.go` reads "inject
   AGENTS.md/CLAUDE.md markers". T-013 fixed this line's autodetection half; the
   `AGENTS.md`/`CLAUDE.md` pairing survived and carries the same unconditional reading. Only
   `AGENTS.md` is unconditional — `CLAUDE.md` follows the agent set, and with `--claude-symlink`
   it is a symlink to `AGENTS.md` rather than a marker block at all.

**Note the marker-block constraint.** This repo self-hosts brine, so its own `AGENTS.md` marker
block is maintained **by hand**, mirroring `markerBlock()`, inside the ticket's diff — never by
running `pickle install|upgrade` against this repo from a feature branch (see `AGENTS.md`'s
self-modify policy). Changing the generated text therefore means changing the Go string **and**
hand-mirroring it into this repo's own marker block in the same commit, or `doctor`'s drift check
and the two will disagree.

Worth checking whether the marker block should say anything agent-specific at all, or whether the
sentence is better phrased so it is true for any agent set — that is the design call refinement
should settle, not a mechanical find-and-replace.

**Settled at refinement (2026-08-25): rephrase, do not condition.** The block is rendered by
`MarkerBlock(cfg)`, a pure function of `pickle.toml`, and `internal/doctor`'s drift check
re-renders it from `cfg` alone to ask "would `pickle upgrade` change this file?". Making the
sentence depend on the agent set therefore means either a new `pickle.toml` key (with a
migration story for every project installed before it, whose agent set is unrecorded) or a
filesystem probe that `doctor` would have to repeat identically or start warning about phantom
drift. Neither is warranted by one sentence. Stating the *condition* instead of the *artifact*
makes the sentence true for every agent set while keeping `MarkerBlock(cfg)` pure — the same
shape T-119 used for the payload half (its decision 2).

Soft coupling to T-119 (which fixed the payload half); nothing here is blocked on it.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is a root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-121-conditional-claude-artifacts
```

WIP commits are encouraged; tidy them into atomic commits before presenting (root-path child —
keep the tidied history rather than squashing). Do not push or open an MR without explicit user
approval. Under `layout = "in-tree"`, before pushing verify the remote base is not behind:
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing.

### Prerequisite gate (hard)

None. `depends-on:` is empty and `spawned-by: [T-119]` is lineage only; T-119 is in `6-done/` and
merged, so the payload wording this ticket matches is already on `main`.

Start from a clean tree on an up-to-date `main`.

### Confirmed design decisions (do not deviate without asking)

1. **`MarkerBlock` stays a pure function of `pickle.toml`.** No new config key, no filesystem
   probe, no extra parameter. `internal/doctor`'s `checkMarkerDrift` re-renders the block from
   `cfg` alone and compares byte-exact; any input `MarkerBlock` gains that `doctor` does not have
   turns a correct install into a drift warning.
2. **State the condition, not the artifact.** The replacement sentence names what produces the
   Claude view rather than asserting the view exists — so it is true for every agent set, in a
   block that cannot know which one was chosen. Proposed wording, replacing "Claude Code sees it
   via `.claude/skills/brine`.":

   > Agents that read `.agents/skills/` find it there directly; `pickle install --agent claude`
   > adds a `.claude/skills/brine` view for Claude Code.

3. **The block keeps mentioning the Claude view at all.** Dropping the sentence would leave a
   Claude Code user with no pointer from their primary instruction file to the path their agent
   actually reads. The defect is the unconditional mood, not the subject.
4. **The generated text and this repo's own `AGENTS.md` change in the same commit.** This repo
   self-hosts brine, so its marker block is maintained by hand, mirroring `markerBlock()` —
   never by running `pickle install|upgrade` against this repo from a feature branch (`AGENTS.md`
   self-modify policy). A commit that changes one and not the other ships a tree whose own
   `doctor` reports marker drift.
5. **`pickle help` names only what every install does.** Only the `AGENTS.md` block is
   unconditional; `CLAUDE.md` follows the agent set and, with `--claude-symlink`, is not a marker
   block at all. Proposed line: `inject the AGENTS.md marker block (and CLAUDE.md's when --agent
   includes claude)`.
6. **`docs/user-manual/cli-reference.adoc` § "`--agent`" remains the authority.** Neither surface
   re-derives the per-agent artifact table; both restate its contract in one clause. Same rule
   T-119 followed (its decision 1).
7. **No behaviour change.** No flag, no artifact, no code path. Only rendered text, its golden
   file, its regression test, and the hand-mirrored copy. If the fix seems to need a behaviour
   change, stop and ask.

### Tasks

#### Task 1 — rewrite the sentence in the generated marker block

`internal/install/install.go`, in `MarkerBlock`'s returned string — the line currently reading:

```go
"  (`resources/review-protocol.md`). Claude Code sees it via `.claude/skills/brine`.\n" +
```

Replace with decision 2's wording, re-wrapping the surrounding bullet to the block's existing
~95-column wrap. The bullet's remaining sentences (what the skill holds; the directory is
pickle-owned and replaced wholesale) are correct and stay.

> **Amended 2026-08-25 (T-120 impact sweep).** That last sentence no longer reads
> "`pickle upgrade` replaces it wholesale" — T-120 made `install` prune the skill directory too,
> and corrected the marker block to "`pickle install` and `pickle upgrade` both replace it
> wholesale, so keep hand-written notes outside it", across all three synced copies
> (`MarkerBlock`, this repo's `AGENTS.md`, `testdata/markerblock.golden`). It is still correct
> and still stays; only the wording to preserve has changed. The line Task 1 actually replaces
> (the `.claude/skills/brine` sentence quoted above) is untouched by T-120, so this task's edit
> target is unchanged — but re-read the bullet on the branch before re-wrapping it, rather than
> trusting the pre-T-120 shape quoted here.

#### Task 2 — regenerate the golden and pin the contract

`internal/install/`. `TestMarkerBlockGolden` pins the whole rendered block against
`testdata/markerblock.golden`, so Task 1 fails it by design — that is the reviewable diff the
golden exists to produce. Regenerate:

```
UPDATE_GOLDEN=1 go test ./internal/install/
```

Then **read `git diff internal/install/testdata/markerblock.golden`** and confirm the only change
is the intended sentence.

A golden pins the current text but says nothing about the property that made it wrong, so add
`TestMarkerBlockStatesNoUnconditionalAgentArtifact` in `install_test.go`: render the block for a
config with a child registered, and assert that **every line mentioning `.claude/` or `CLAUDE.md`
also mentions `--agent`**. That is mechanically checkable, it fails for the pre-Task-1 text, and
it keeps failing for any future re-introduction — which a golden, once regenerated, would not.
Name T-121 in the test's doc comment and say what defect it guards (a block generated *into the
user's own repo*, and refreshed there by `pickle upgrade`, asserting a directory install never
created).

#### Task 3 — hand-mirror this repo's own `AGENTS.md` marker block (decision 4)

Edit the same sentence inside the `<!-- pickle:begin -->`…`<!-- pickle:end -->` span of this
repo's `AGENTS.md`, byte-for-byte as `MarkerBlock` now renders it (same wording, same wrap, same
leading indentation). Do **not** run `pickle install` or `pickle upgrade` to do it. Nothing
outside the marker span changes.

The check that the mirror is exact is `doctor`'s own drift comparison, run in the acceptance test
below — it is the literal question "would `pickle upgrade` change this file?", which is what the
policy asks a hand-mirror to satisfy.

#### Task 4 — fix the `pickle help` install summary

`internal/cli/cli.go`, in `usage()`. The `install` entry currently reads "… install the skill for
the agents named by --agent (default claude), inject AGENTS.md/CLAUDE.md markers, write
pickle.toml, and register the first child-project." Apply decision 5's wording, keeping the
block's existing column alignment and its ~86-column right edge. No test asserts this text today
(verified at refinement), so the acceptance test greps the built binary's output instead.

#### Task 5 — sweep the remaining non-payload surfaces

T-119 swept the payload; this ticket owns everything else. Grep and read:

```
grep -rn "CLAUDE.md\|\.claude/" internal/ docs/ README.md
```

At refinement this surfaced: the two defects above; `docs/user-manual/cli-reference.adoc`'s
`--agent` table and its `install` bullet list, both already correct (the manual attributes the
Claude artifacts to `--agent claude` and lists only `AGENTS.md` as unconditionally injected);
`concepts/project-structure.adoc`, which already annotates `CLAUDE.md` with `← --agent claude`;
and the `AGENTS.md`/`CLAUDE.md` shorthand in the `project add|remove` and `uninstall`
descriptions, which describe refreshing or stripping *whatever exists* and are not claims that
both files do. Fix anything the grep surfaces that genuinely asserts a Claude artifact
unconditionally; otherwise record in the summary that the sweep found nothing further. Do not
rewrite correct prose for uniformity.

#### Task 6 — CHANGELOG entry

Add a bullet to `## [Unreleased]` → `### Fixed` in `CHANGELOG.md`, directly after T-119's entry
("The same agent-autodetection claim also lived in the shipped skill payload…"), which this one
completes. State that the two surfaces *outside* the payload said the same thing — the
`AGENTS.md` marker block generated into every project's own repo, and the `pickle help` install
summary — and that the marker block now names the condition (`--agent claude`) rather than
asserting the view exists. End with `(T-121)`.

### Acceptance test

Run from a clean tree on the feature branch.

1. **The suite is green, including the regenerated golden and the new test:**

   ```
   just build && just test && just lint && just docs-check
   ```

2. **The new test is load-bearing:** temporarily restore the old sentence in `MarkerBlock`;
   `TestMarkerBlockStatesNoUnconditionalAgentArtifact` must fail (alongside the golden). Restore
   the fix and re-run.

3. **The claim is true on an install that did not name `claude`** — a throwaway install, per the
   self-modify policy (never against this repo; the test binary is always named `pickle-test`):

   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q .
   ./pickle-test install --agent opencode
   grep -n "agent claude" AGENTS.md      # the qualified sentence shipped
   ls -d .claude 2>/dev/null             # expect: no such directory
   test -f CLAUDE.md && echo UNEXPECTED || echo "no CLAUDE.md — as the block now implies"
   ./pickle-test doctor -v               # expect: no marker-drift warning
   ```

4. **And still true on an install that did name it:**

   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q .
   ./pickle-test install --agent claude
   ls -l .claude/skills/brine            # the view the sentence points at exists
   ./pickle-test doctor -v               # expect: clean, no drift
   ```

5. **`pickle help` no longer pairs the two files:**

   ```
   ./pickle help | grep -A3 '^  install'
   ```

   Expected: the `AGENTS.md` block named as unconditional, `CLAUDE.md` qualified by `--agent`.

6. **This repo's hand-mirror is byte-exact (decision 4):**

   ```
   ./pickle doctor -v | grep -i marker
   ```

   Expected: `AGENTS.md marker block current` — *not* the "block differs from what pickle.toml
   renders" warning. This is read-only and is the one self-host command the policy permits here;
   it fails if Task 3's mirror differs from Task 1's render by even one byte.

7. **Nothing outside the intended surfaces moved:** `git diff --name-only main...HEAD` prints
   exactly `AGENTS.md`, `CHANGELOG.md`, `internal/cli/cli.go`, `internal/install/install.go`,
   `internal/install/install_test.go`, `internal/install/testdata/markerblock.golden` (plus
   anything Task 5 legitimately surfaced). No `skill/` path — that half was T-119's.

### Docs update (mandatory when user-facing)

User-facing text on two surfaces, both fixed in the tasks themselves: the generated `AGENTS.md`
marker block (Task 1) and `pickle help` (Task 4). The user manual needs no change — its `--agent`
section already states the contract correctly and remains the authority (decision 6); Task 5
re-verifies that and records the verdict. `CHANGELOG.md` gains one `Fixed` bullet (Task 6).

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs: CHANGELOG bullet added; the sweep's verdict on the manual recorded in the summary.
3. Write a **summary**: the before/after sentence and help line, the golden diff, the result of
   the load-bearing check (step 2), the two throwaway installs' output, and confirmation that the
   hand-mirror passes `doctor` byte-exact.
4. Suggest a **Conventional Commit message**, e.g.:

   ```
   fix(install): stop stating Claude-only artifacts unconditionally (T-121)

   <body — what and why>
   ```

5. **Tidy up before presenting** — `pickle` is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed/scoped commits first. Keep the
   generated-text change and its hand-mirror in **one** commit (decision 4).
6. Commit locally on the ticket branch. Do **not** push or open a merge request without user
   approval. Present the commit message; after approval, keep the tidied history (root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the merge request — merging is always the
   human's. Hand back to the user.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: review: T-119's review, non-blocking findings F2 and F3
  (disposition: new ticket, batched by theme). The two non-payload surfaces stating Claude-only
  artifacts unconditionally; T-119 decision 5 barred it from touching `internal/`
- 2026-08-25 — TO DO → READY: plan complete
- 2026-08-25 — plan amended inline: T-120's review impact sweep. Task 1's note that the bullet's
  remaining sentences "are correct and stay" quoted the pre-T-120 wording ("`pickle upgrade`
  replaces it wholesale"); T-120 made `install` prune the skill directory too and corrected that
  sentence to name both commands. Task 1's actual edit target is unaffected — only the
  surrounding wording to preserve changed. No scope or grade change
- 2026-08-25 — READY → IN DEVELOPMENT: picked up
