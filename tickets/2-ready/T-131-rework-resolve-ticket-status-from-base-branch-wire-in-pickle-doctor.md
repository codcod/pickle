---
id: T-131
title: rework: resolve ticket status from base branch, wire in pickle doctor
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-131 — rework: resolve ticket status from base branch, wire in pickle doctor

## Outcome

After this ships, a brand-new session told "rework ticket T-NNN" under `layout = "in-tree"`
resolves the ticket's real current status from the base branch mechanically — via the same
unconditional, directory-agnostic lookup and mandatory `pickle doctor` pre-flight T-130 wired
into *implement* and *validate* — instead of the agent reading its feature branch's possibly
stale worktree copy and wrongly stopping with "must be in `5-rework/`" when the review verdict
that put it there landed on the base branch, on a branch cut earlier.

## Description

T-130 fixed this exact mirror-image hazard (T-108/T-128) for two of the three trigger
procedures — `resources/review-protocol.md` step 1 and `SKILL.md`'s inline "Procedure: implement
a ticket" — by resolving the ticket's path from base unconditionally
(`git ls-tree -r --name-only <base> -- tickets | grep -- "/T-NNN-"`, then `git show`) and adding a
mandatory `pickle doctor` pre-flight once the ticket's own feature branch is checked out. It did
not touch `resources/procedure-rework.md`, whose own step 1 still reads: *"The ticket must be in
`5-rework/` — if not, stop and explain."* — a worktree read of whatever status the checked-out
`feat/T-NNN-*` branch happens to carry, with no base-branch resolution and no `pickle doctor`
step, the same gap T-130's Description called out in `SKILL.md`'s implement procedure before that
ticket fixed it.

**Confirmed in the field.** An in-tree install (a foreign workspace, not this repo) had T-047
reviewed to a `5-rework/` verdict — bookkeeping committed on the base branch, per this project's
own "where commits land" rule (mirrored in every in-tree install). The agent, still on
`feat/T-047-*` (cut while the ticket was in `3-in-development/`), was then told "rework ticket
T-047", read its own worktree copy per `procedure-rework.md` step 1, saw `3-in-development/`, and
stopped — exactly the failure mode T-108/T-128/T-130 already diagnose and detect, on the one
trigger procedure T-130 left unpatched.

**Fix, same shape as T-130, doc-only:**

1. `resources/procedure-rework.md` step 1: replace "the ticket must be in `5-rework/` — if not,
   stop and explain" with the unconditional base-branch lookup (same `git ls-tree`/`git show`
   recipe T-130 put in `review-protocol.md` step 1) — a ticket "rework ticket T-NNN" is invoked
   on is legitimately read from base, since the branch was necessarily cut before the review
   verdict that moved it to `5-rework/` landed there. Stop-and-explain is still correct, just
   gated on the *base-branch* status, not the worktree's.
2. Add the same `pickle doctor` pre-flight T-130 put in `review-protocol.md`'s new `## 0a.` and
   `SKILL.md`'s implement preamble — `procedure-rework.md` gains an equivalent step before step 1
   (or a preamble sentence, mirroring whichever of T-130's two patterns fits this file's shape),
   run once `feat/T-NNN-*` is checked out, any stale-ticket-branch warning resolved (rebase onto
   base) before proceeding.
3. `SKILL.md`'s "Procedure: rework a ticket" section (currently just "Read
   `resources/procedure-rework.md` and follow it.") needs no independent edit if the base-branch
   resolution and `pickle doctor` step live entirely inside `procedure-rework.md` — confirm during
   refinement whether T-130's split (short-form clause in `SKILL.md` *and* the full step in
   `review-protocol.md`) is needed here too, or whether the single-file procedure makes the
   `SKILL.md` pointer sufficient as-is.

**Soft couplings (not `depends-on`, both already `6-done/`):**

- **T-130** — the precedent this ticket copies verbatim for a third trigger procedure; same
  detection (`checkStaleTicketBranch`, T-128), same doc-only shape.
- **T-128** — supplies the `pickle doctor` mechanism this ticket wires into
  `procedure-rework.md`; no Go code changes here either.

## Implementation Plan

### 0. Feature branch (mandatory)

Root-path child (`project: pickle`, `path = "."`):

```
cd .
git checkout main
git checkout -b feat/T-131-rework-resolve-status-from-base
```

Tidy WIP commits into atomic ones before presenting, per `tickets-README.md` §0. Publish only
after user approval (no push/MR without it).

### Prerequisite gate (hard)

None. T-130 and T-128 are both `6-done/`; no `depends-on:` — both couplings are soft (reuse of
already-shipped detection and an already-shipped doc pattern, not a blocker).

### Confirmed design decisions (do not deviate without asking)

1. **Doc-only change; no Go code touched.** `checkStaleTicketBranch` (T-128) already does the
   right detection; this ticket only changes where `procedure-rework.md` points, exactly as T-130
   did for the other two trigger procedures.
2. **Copy T-130's preamble verbatim, not a new formulation.** `SKILL.md`'s "Procedure: implement
   a ticket" already carries the exact sentence this ticket needs (`SKILL.md:184-188`): resolve
   the ticket's current status from base unconditionally
   (`git ls-tree -r --name-only <base> -- tickets | grep -- "/T-NNN-"`, then `git show`), and once
   any pre-existing `feat/T-NNN-*` branch is checked out, run `pickle doctor` and resolve any
   stale-ticket-branch warning first. `procedure-rework.md` gains the same sentence as its own
   preamble, before its numbered list — a third copy of a pattern already proven twice, not a
   reformulation.
3. **`SKILL.md`'s "Procedure: rework a ticket" section needs no edit.** Unlike "Procedure:
   validate a ticket", which carries its own inline short-form duplicate of
   `review-protocol.md`'s content (and so needed its own added clause in T-130), the rework
   section is only a one-line pointer — *"Read `resources/procedure-rework.md` and follow it."*
   All content lives in the target file; adding the preamble there is sufficient. Confirmed by
   reading `SKILL.md:227-229` in refinement.
4. **Step 1's wording changes, not its position.** *"The ticket must be in `5-rework/` — if not,
   stop and explain"* becomes *"The ticket must be in `5-rework/` **on the base branch** (per the
   lookup above) — if not, stop and explain"* — the stop-and-explain behaviour is correct and
   stays; only which branch's status it checks changes. Steps 2–6 are unaffected and keep their
   numbers — no external file cross-references `procedure-rework.md`'s step numbers (grepped
   repo-wide in refinement: only `tickets/6-done/T-129` and this ticket name the file, neither by
   step number), so there is no renumbering hazard to guard against, unlike `review-protocol.md`
   in T-130.
5. **No `docs/user-manual/` edit.** `docs/user-manual/concepts/project-structure.adoc`
   ("What choosing in-tree costs") already documents the general stale-worktree hazard, naming
   the exact `in review` → `rework` backward-move case this ticket fixes, and frames it as an
   inherent cost of the layout with its own mitigation advice ("check which branch you are on") —
   that prose describes the hazard's existence, not a specific procedure's mechanics, and stays
   accurate regardless of this fix. Grepped for procedure-specific language ("must be in
   `5-rework/`", "stop and explain") — no hits outside the skill payload.
6. **Payload-lint safe.** New prose reuses only generic `T-NNN`/`<base>` placeholders already
   used throughout the payload (verbatim copy of `SKILL.md`'s existing preamble, decision 2) and
   names no `T-1NN` id inside `skill/` — payload-lint's scope excludes `tickets/`, so this
   ticket's own Description/History may name T-047/T-130/T-128 freely.

### Tasks

#### Task 1 — `skill/resources/procedure-rework.md`: preamble + step 1
Insert this paragraph immediately after the file's opening line ("When asked to rework ticket
T-NNN (a review found blocking findings):") and before the numbered list, copied verbatim from
`SKILL.md`'s "Procedure: implement a ticket" preamble:

> Under `layout = "in-tree"`, before reading the ticket, resolve its current status from the base
> branch rather than trusting the worktree — `git ls-tree -r --name-only <base> -- tickets |
> grep -- "/T-NNN-"`, then `git show <base>:<that path>` — and, once any pre-existing
> `feat/T-NNN-*` branch for this ticket is checked out (a resumed pickup; a fresh one has no
> branch yet), run `pickle doctor` and resolve any stale-ticket-branch warning first.

Then rewrite step 1 (currently *"The ticket must be in `5-rework/` — if not, stop and
explain."*) to: *"The ticket must be in `5-rework/` on the base branch (per the lookup above) —
if not, stop and explain."* Steps 2–6 are unchanged.

#### Task 2 — sweep the symlinked view
`.agents/skills/brine/` is a symlink to `skill/` (per this repo's root `CLAUDE.md`) — confirm
`git status` shows no separate diff needed there; if the symlink is somehow not a plain symlink
in this checkout, copy the same edit across and say so in the summary.

### Acceptance test

- `just docs-check` clean (asciidoc xref checker + payload lint both run under it, per
  `pickle.toml`'s configured commands).
- `just test` clean, specifically confirming `payload_lint_test.go`'s
  `TestPayloadSpeaksToAForeignReader` (and its sibling `TestPayloadLintRule*` tests) still pass
  against the new prose in `skill/`.
- Manual read-through: `git show HEAD:skill/resources/procedure-rework.md` shows the new preamble
  before the numbered list, step 1 reworded, steps 2–6 unchanged (`grep -n "^[0-9]\." ` before/
  after the edit, same six numbers).
- `just lint`, `just build` clean (should be no-ops for a doc-only change, but run as the
  project's standard gate).

### Docs update (mandatory when user-facing)

No `docs/user-manual/` surface — confirmed in refinement (decision 5): the existing
`project-structure.adoc` "What choosing in-tree costs" prose already documents the general
hazard without naming a specific procedure's mechanics, and needs no change.

### Finish (mandatory)

1. `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. No docs registration needed (see above).
3. Write the summary (files touched, decisions made, anything deferred) and hand back.
4. Suggested commit message: `docs(skill): resolve ticket status from base branch in rework
   procedure, wire in pickle doctor (T-131)` — tidy the branch's WIP commits into this one atomic
   commit first (root-path child, `tickets-README.md` §0), then present it; do not push without
   approval.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-20 — created (TO DO). source: field-use: an in-tree install had a review verdict move
  T-047 to `5-rework/` via base-branch bookkeeping; a fresh session told "rework ticket T-047"
  read its stale feature-branch worktree copy (still `3-in-development/`) per
  `procedure-rework.md` step 1's unconditional worktree read, and wrongly stopped — the same
  mirror-image hazard T-130 fixed for *implement* and *validate*, left open on *rework*.
- 2026-09-20 — TO DO → READY: plan complete
