---
id: T-084
title: give bookkeeping commits their own board: convention, distinct from child-project Conventional Commits
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-084 — give bookkeeping commits their own board: convention, distinct from child-project Conventional Commits

## Outcome

After this ships, a bookkeeping commit (a ticket's state change) is written in its own `board: T-NNN <verb phrase>` form instead of being forced through Conventional Commits' `type(scope)` grammar, and this repo's own `pickle` child stops squash-merging its feature branches so its `git log` reads as cleanly as a standalone product repo's would.

## Description

Measured on this repo's own history (2026-08-07 audit; re-verified 2026-08-09 during refinement
and again at pickup — the ratio holds, only the absolute counts moved as history grew): as of
commit `afc2a19`, 164 of 270 commits are overarching bookkeeping (ticket/board mutations), and
virtually all of them carry the scope `(tickets)` — `docs(tickets): …`, `chore(tickets): …`.
Checked against the real diffs: the scope is *accurate* (162 of 164 touch only `tickets/`), so
it isn't a mislabeling bug.
(Re-derive with `git log --oneline | wc -l` for the total, `git log --format=%s | grep -c
'(tickets)'` for the scoped count, and a per-commit `git show --name-only` walk for the
touches-only-`tickets/` count.) It's a category-fit problem:
Conventional Commits' `type(scope)` grammar names **what area of the product changed** — a
bookkeeping commit isn't a product change, it's a **state transition of a ticket**. Forcing every
such commit through that grammar either produces an uninformative scope (`tickets`, always) or an
artificial one (`docs`/`skill`/`agents`/`other`, picked by which directory happened to be touched)
that still misdescribes what the commit actually is.

The symmetric half of the same investigation: this repo's child-project commits (the actual
`pickle` product changes, on `feat/T-NNN-<slug>` branches) are squash-merged today, which is fine
for a child registered at a nested path (it has, or could have, its own separate repo whose log
is naturally clean) but is actively costly for `pickle`'s own setup, where the child **is** the
repo root (`path = "."` in `pickle.toml`). Thought experiment that surfaced it: if `pickle` were
developed as a nested child inside a separate `pickle-meta` repo, the meta repo's log would show
`board: T-NNN …`-style bookkeeping (fine, expected) and `pickle`'s own repo would independently
show a clean, granular `feat`/`fix`/`ci`/`test`/`refactor` history — because bookkeeping would
never have been mixed into it to begin with. Since there is no second repo here, squashing the
child's branch on merge is the only thing destroying that granularity: `git log --invert-grep
--grep '^board:'` on `main` should recover exactly what that hypothetical standalone `pickle`
repo's log would look like, and today it can't, because every ticket's internal commit sequence
(whatever `feat`/`fix`/`ci` structure it had) is flattened into one squash commit carrying one
type/scope for the whole ticket.

**Decisions reached in discussion (2026-08-07), to be locked in refinement:**

1. **Bookkeeping commits stop using Conventional-Commit `type(scope)` entirely.** New flat
   format: `board: T-NNN[, T-MMM …] <verb phrase>`, state moves phrased with an arrow (`picked
   up → in development`), content-only edits as a plain clause, multiple tickets touched in one
   sitting listing all ids. The ticket id leads the subject (it's the actual subject of a
   bookkeeping commit, unlike a code commit where the id is a trailing cross-reference).
2. **Granularity — fold only when adjacent.** A content-only annotation (no board move) folds
   into an *adjacent* same-sitting bookkeeping commit for the same ticket, when one exists (no
   branch switch in between); otherwise it stays its own commit. Checked against T-072's real
   9-commit sequence: this rule is a strict improvement but has narrow effect in practice, because
   most of that sequence already brackets a real branch switch or a distinct trigger invocation
   (pickup/review/rework/re-review), which must stay individually committed — collapsing those
   would reintroduce the exact uncommitted-bookkeeping-crosses-a-branch-switch hazard `pickle
   hooks install` (T-057) and the origin-base check (T-072) exist to prevent. A deliberately
   bigger lever (putting the feature branch in its own `git worktree`, decoupling bookkeeping
   commit timing from branch-switch safety entirely) was considered and set aside for a possible
   follow-up ticket rather than folded into this one.
3. **Root-path child (`path = "."`) defaults to preserving individual commits on merge**
   ("rebase and merge", not "squash and merge") instead of today's squash default. A child
   registered at a nested path is unaffected — squashing there doesn't cost the same thing, since
   its own history isn't sharing a log with bookkeeping. Consequence: the WIP commits that
   survive onto `main` must themselves be well-formed — the **Finish** step gains a tidy-up
   sub-step (interactive rebase into a small number of atomic, correctly typed/scoped commits)
   before the commit sequence is presented for approval, replacing today's "write one commit
   message" with "curate the commit sequence that will actually land".

**Decisions reached in refinement (2026-08-09, confirmed by the user):**

4. **Decision 3 is prose-only** — no `pickle.toml` `merge_strategy` key. Mirrors T-072 staying
   prose-only and spawning T-082 for the mechanical half; a mechanical enforcement of decision 3
   (if ever wanted) is a similar follow-up, not this ticket's job.
5. **The `board:` format ships in the skill payload**, not only this repo's `AGENTS.md` — every
   project hits the same Conventional-Commits category-fit problem once it has any bookkeeping
   commits, and rules §0 already documents the single-repo default (`path = "."`) as the common
   case, not a self-host quirk.
6. **`board:` is scoped to ticket/board state changes** — commits whose staged paths are
   entirely under `tickets/`. A commit that also touches `pickle.toml`, `NOTES.md`, or the
   ticket-flow docs is not a pure state change and keeps ordinary Conventional Commits.
7. **No mention in `review-protocol.md`'s reviewer-facing checklist** — the checklist gates
   review actions; a commit-format line adds no verification value there. The format lives in
   step 9's prose (which the reviewer already reads when composing messages), never restated in
   the checklist.

Placement of the wording: `resources/tickets-README.md` (rules §0, and the Finish item in §4),
`resources/review-protocol.md` (step 9's prose, not its checklist — decision 7),
`resources/TEMPLATE.md` (Finish section), and the two `docs/user-manual/concepts/` pages that
already narrate this ground for end users (`lifecycle.adoc`'s Finish-step summary,
`project-structure.adoc`'s single-repo-default section). This repo's own `AGENTS.md` marker
block (rendered by `install.go`'s `MarkerBlock()`) states the bookkeeping-lands-on-base-branch
policy but never states a commit *message format* today, so it needs no change — verified by
reading its rendered "Where commits land" bullet before writing the plan, avoiding an
unnecessary trip through `TestMarkerBlockGolden` / `TestSelfHostMarkerBlockIsCurrent`.

**Soft couplings:** T-057 (the `pre-commit` hook guarding bookkeeping-on-feature-branch) and
T-072/T-082 (the origin-base check and its proposed `pre-push` guard) all protect the same
invariant this ticket's decision 2 leans on — none of them change, and the three-dot
`origin/<base>...HEAD` check stays valid unchanged whether the child's merge is squash or
rebase-and-merge. T-022 and T-036 are the precedent for filing rather than hand-editing a
payload-prose change to `skill/`.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle`'s target child is the repo root (`path = "."` in `pickle.toml`), so the branch is cut
in this repo itself:

```
git checkout main
git checkout -b feat/T-084-board-commit-convention
```

All work below is docs/prose-only (no Go source changes), so `pickle hooks install`'s
pre-commit guard is irrelevant to the *product* commits on this branch, but this ticket's own
bookkeeping (this file, `BOARD.md`) still must not land here — commit it on `main` per rules
§0, same as any other ticket.

### Prerequisite gate

None. `depends-on: []`.

### Confirmed design decisions (do not deviate without asking)

1. **Format:** `board: T-NNN[, T-MMM …] <verb phrase>`. A state move is phrased with an arrow
   (`picked up → in development`); a content-only edit (no move) is a plain clause; multiple
   tickets touched in one sitting list every id, comma-separated, right after `board:`.
2. **Scope:** applies only to a commit whose staged paths are entirely under `tickets/`. A
   commit that also touches `pickle.toml`, `NOTES.md`, or any ticket-flow doc keeps ordinary
   Conventional Commits — it is not a pure ticket-state change.
3. **Folding:** a content-only annotation folds into an *adjacent* same-sitting `board:` commit
   for the same ticket, when one exists (no branch switch since); otherwise it stays its own
   commit. Never fold across a branch switch or a distinct trigger invocation
   (pickup/review/rework/re-review).
4. **Root-path child merge default:** for a child registered at `path = "."`, Finish defaults
   to preserving commits on merge (rebase, or keep-history) instead of squashing. A child at a
   nested path is unaffected (still squash by default) — its history doesn't share a log with
   bookkeeping. Prose-only guidance; no `pickle.toml` key.
5. **Finish tidy-up:** for a root-path child, before presenting the commit sequence for
   approval, interactive-rebase the branch's WIP commits into a small number of atomic,
   correctly typed/scoped Conventional Commits (this replaces "write one commit message" with
   "curate the commit sequence that will actually land").
6. **No checklist change:** `review-protocol.md`'s reviewer-facing checklist (the `- [ ]` list
   at the end) is not touched — the format lives only in step 9's prose.
7. **`install.go`'s `MarkerBlock()` is not touched.** Its "Where commits land" bullet states
   the bookkeeping-lands-on-base-branch policy but never a commit *message format*, so nothing
   there is now stale. Confirmed by reading `internal/install/install.go:898-915` before
   writing this plan; re-confirm the same lines are unchanged before Finish, since a change
   there would trip `TestMarkerBlockGolden` and `TestSelfHostMarkerBlockIsCurrent` and require
   hand-mirroring this repo's own `AGENTS.md` in the same commit (self-modify policy).

### Tasks

#### Task 1 — define the `board:` format and folding rule in the rules doc

In `skill/resources/tickets-README.md`, §0 "Where commits land", after the existing first
paragraph (ending "...including the ones a *review* performs.") and before the existing
"In the **single-repo default**..." sub-bullet, insert two new sub-bullets:

- One defining the `board:` grammar (decision 1) and its scope (decision 2), including the
  "why not Conventional Commits" rationale already argued in this ticket's Description
  (category-fit, not a mislabeling bug).
- One stating the folding rule (decision 3), including why collapsing across a branch switch
  or trigger invocation would reintroduce the hazard the pre-commit hook (T-057) and the
  origin-base check (T-072) exist to prevent.

Then extend the existing "In the **single-repo default**..." sub-bullet with one more sentence
stating the root-path merge default (decision 4: prefer rebase/keep-history over squash there,
because that history also carries the child's own commits) and a forward pointer to §4 item 7
/ TEMPLATE.md's Finish step for the resulting tidy-up obligation. Leave the three sub-bullets
below it (`pickle hooks install`, the publish-time hook limitation, the mirror-image hazard)
unchanged.

#### Task 2 — extend the READY gate's Finish item

In `skill/resources/tickets-README.md` §4, item 7 ("**Finish** — summary + a suggested
Conventional Commit message..."), append a sentence: for a root-path child, the tidy-up
(interactive rebase into atomic, correctly typed/scoped commits) happens before that summary is
presented, and the merge defaults to preserving that history (rebase/keep-history, not squash)
per §0. Do not touch the rest of item 7 — it still governs the *child's* product-code commit
message, which stays Conventional Commits regardless of merge strategy.

#### Task 3 — update the ticket template's Finish section

In `skill/resources/TEMPLATE.md`, the `### Finish (mandatory)` numbered list: insert a new step
between the existing step 4 (suggest the Conventional Commit message) and step 5 (commit
locally / publish), reading approximately: "For a root-path child (`path = "."`, rules §0),
tidy the branch's WIP commits into a small number of atomic, correctly typed/scoped commits
before presenting them — this is what replaces squash-on-merge as the default for that case; a
child at a nested path can skip this." Renumber the old step 5 to step 6 and adjust its
"finalize the branch (squash or keep history — the user chooses)" clause to name the root-path
default (keep the tidied history) alongside the existing squash option, still user-chosen at
approval time.

#### Task 4 — update the review protocol's Finish step

In `skill/resources/review-protocol.md` §9, the "**Child-project (build target):**" bullet:
after the sentence "Local WIP commits on the ticket branch are fine at any time — the approval
gate governs **publishing**." and before "Only after the user approves all attributes,
finalize...", insert one sentence: for a root-path child (rules §0), tidy the WIP commits into
atomic ones before presenting them, and default to keeping that history on merge rather than
squashing. Do not touch the "### Checklist" section (decision 6/7).

#### Task 5 — mirror the change into the user manual

- `docs/user-manual/concepts/lifecycle.adoc`: the Finish bullet (item 7 of the numbered READY
  gate list, "*Finish* — summary + a suggested Conventional Commit message...") gains a
  trailing sentence naming the root-path tidy-up-then-preserve default, cross-referencing
  `<<project-structure>>`.
- `docs/user-manual/concepts/project-structure.adoc`: in the "root child" section's "What that
  changes in practice" bullet list (after the existing publish-gate bullet), add two bullets:
  one introducing the `board:` format (one line + a pointer to the skill's `tickets-README.md`
  §0 for the exact grammar — the user manual narrates, the skill rules define), and one stating
  the rebase/keep-history-over-squash default with the one-line reason (squashing here would
  flatten the product's own `feat`/`fix`/`ci` structure sharing a log with bookkeeping).

#### Task 6 — correct the Description's measurements (done in refinement)

Already applied above — no separate task; listed here only so the acceptance test's "re-derive
the numbers" step has a fixed point to check against.

### Acceptance test

1. `just build && just test && just lint` — must stay green (no Go source touched by this
   ticket; this proves nothing regressed, notably `TestPayloadDispositionVocabulary`,
   `TestPayloadDefersToProjectConfig`, the TEMPLATE-drift guards in `internal/audit` and
   `internal/ticket`, and `TestMarkerBlockGolden` / `TestSelfHostMarkerBlockIsCurrent`, none of
   which this ticket's edits should touch — confirm by grepping the diff for `install.go`
   or any `internal/**/*.go` and finding none).
2. `just docs-check` — must pass (validates the user-manual's includes/xrefs after Task 5's
   edits, notably the `<<project-structure>>` xref added to `lifecycle.adoc`).
3. Read back all four edited resource files end to end and confirm: the `board:` grammar is
   stated in exactly one place (`tickets-README.md` §0) and the other three files reference it
   rather than restating it (mirrors the existing single-source-of-truth pattern
   `TestPayloadDispositionVocabulary` enforces for the four dispositions — not machine-checked
   here since it's a new vocabulary, but keep the same discipline by inspection).
4. `grep -rn 'board:' skill/resources/tickets-README.md skill/resources/TEMPLATE.md skill/resources/review-protocol.md docs/user-manual/concepts/project-structure.adoc` — confirm the format
   is defined once and referenced, not restated, matching check 3.
5. Manually re-derive the Description's commit-history numbers with the three commands quoted
   there and confirm they still match what's written (they will have moved further by
   implementation time; update the Description's numbers again if so, same as this refinement
   pass did).

### Docs update (mandatory — user-facing)

Covered by Task 5 above: `docs/user-manual/concepts/lifecycle.adoc` and
`docs/user-manual/concepts/project-structure.adoc`. Run `pi`'s `docs_readability` tool (or the
equivalent reviewer step) on both changed `.adoc` files before Finish, since this is prose a
user reads to understand the flow's commit conventions.

### Finish (mandatory)

1. Acceptance test green (`just build && just test && just lint && just docs-check`).
2. Docs updated per Task 5 and passed through a readability pass.
3. Write a summary: which files changed, the six confirmed decisions as shipped, and explicit
   confirmation that `install.go`/`MarkerBlock()` was checked and found not to need a change.
4. Suggest a Conventional Commit message, e.g.:

   ```
   docs(tickets): give bookkeeping commits a board: convention, distinct from
   child Conventional Commits (T-084)

   Bookkeeping commits (ticket/board state changes) drop Conventional Commits'
   type(scope) grammar for a flat `board: T-NNN <verb phrase>` form, scoped to
   commits whose staged paths are entirely under tickets/. Root-path children
   (path = ".") default to preserving commits on merge (rebase/keep-history)
   instead of squashing, with a Finish tidy-up step before presenting the
   commit sequence for approval.
   ```

5. Commit locally on the ticket branch (this ticket's own bookkeeping commits, on `main`, use
   the *new* `board: T-084 <verb phrase>` form once Task 1 lands — eating its own dog food from
   the moment the format is defined). Publish only per the project's commit policy (do not push
   or open a merge request without user approval). Before pushing, verify the remote base is
   not behind local (`git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` must print nothing). Hand back to the user.
6. Move the ticket: `pickle ticket move T-084 in-review --reason "acceptance green"`.

## Review

**Reviewed:** 2026-08-09, commit `b3769c2` on `feat/T-084-board-commit-convention`.
**Verdict:** REWORK — two blocking findings.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — run during implementation on both changed `.adoc`
      files; two suggestions applied. Not re-run for this review, as the prose returns to rework.
- [x] Findings recorded with severity **and** disposition (step 5)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message & MR attributes presented for approval (step 9) — deferred:
      publishing waits until the rework pass closes the blocking findings.

**Gates:** `just build` ✓ · `just test` ✓ · `just lint` ✓ · `just docs-check` ✓. All five
Tasks landed in the files they name; decision 6 (leave the review-protocol checklist alone) was
honoured. The defect is not in what was written — it is in the consistency sweep the change
implied but did not perform.

| id | severity | disposition | finding | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | **blocking** | — | **Decision 7 rests on a false premise.** `MarkerBlock()` *does* state a commit-message format, in the **"Branch & commit"** bullet — a blanket "Conventional Commits with the ticket id in brackets" covering *all* commits, with no bookkeeping carve-out. Refinement checked only the "Where commits land" bullet and concluded no change was needed. Worse, `SKILL.md` declares the marker block **wins on any disagreement** about commit policy — so every installed project now ships two contradicting commit rules with the **stale one authoritative**. | `internal/install/install.go:898`; rendered at `AGENTS.md:53`; precedence clause `skill/SKILL.md:80-81` | Add a carve-out to the bullet ("… for child-project code; ticket/board bookkeeping uses `board: T-NNN …`, rules §0"), update `TestMarkerBlockGolden`, and hand-mirror this repo's `AGENTS.md` in the same commit per the self-modify policy. |
| F2 | **blocking** | — | **The scope rule contradicts itself on a day-one case.** It reads "applies only to a commit whose staged paths are entirely under `tickets/`", then lists "a commit that also touches … `NOTES.md`" as keeping Conventional Commits — but `NOTES.md` **is** `tickets/NOTES.md`, i.e. under `tickets/`. A commit staging only that file satisfies both branches of the rule. | `skill/resources/tickets-README.md:66-68`; `tickets/NOTES.md` exists | State the test by *file set*, not directory: staged paths limited to ticket files + `BOARD.md`. Name `tickets/NOTES.md` explicitly as an exclusion. |
| F3 | non-blocking | fixed inline | `skill/SKILL.md` — the always-loaded entry point — was never touched: its defaults still say "Conventional Commits with the ticket id appended" with no bookkeeping carve-out, and its validate-step still says "finalize the branch (squash or keep history)" with no root-path tidy-up. | `skill/SKILL.md:69-70`, `:249` | One clause in each, deferring to rules §0. |
| F4 | non-blocking | fixed inline | The manual documents `board:` **only** inside project-structure's "root child" list, implying it is a root-path consequence. Decision 5 makes it universal — a multi-child user finds no mention of the convention anywhere. | `docs/user-manual/concepts/project-structure.adoc:161-168` | Move the `board:` bullet somewhere non-root-specific; keep only the merge-default bullet root-scoped. |
| F5 | non-blocking | fixed inline | `TEMPLATE.md` states the finalize choice **twice**; only the second was updated, so one file now says two different things. The un-updated one is in the branch section an implementer reads *first*. | `TEMPLATE.md:61` vs `:126-129` | Add the root-path default at line 61, or delete the duplicate. |
| F6 | non-blocking | fixed inline | New prose is under-specified for execution: no worked example subject line (unlike `feat(cli): add board audit (T-2)` elsewhere in the payload); "verb phrase", "same-sitting" and "adjacent" are undefined; the fold rule never says *how* to fold (amend vs interactive rebase). | `tickets-README.md:57-75` | Add one worked example (`board: T-084 ready → in development`) and name `git commit --amend` as the fold mechanism. |
| F7 | non-blocking | noted | **Dogfooding gap.** `ca3bf57` was committed *after* the format landed on the branch (`b3769c2`) yet still used `chore(tickets): …`, contradicting the plan's Finish step 5 ("use the new form from the moment the format is defined"). Pre-`b3769c2` bookkeeping commits are correctly in the old form. | `git log --format='%h %ad %s' main` | Not worth rewriting `main`'s history for a message. Fix forward: bookkeeping from this review onward uses `board:`. |
| F8 | non-blocking | fixed inline | The product commit's own scope is `docs(tickets)` although it touches **zero** `tickets/` paths (only `skill/` and `docs/`) — the exact "artificial scope picked by whichever directory was touched" failure this ticket argues against. Inherited from the plan's suggested message. | `b3769c2`; `git diff --stat main...HEAD` | Use `docs(skill)` or unscoped `docs:` when re-presenting after rework. |
| F9 | non-blocking | fixed inline | No `CHANGELOG.md` entry, though `[Unreleased]` is open and T-083 (immediately prior, comparable payload change) wrote into it. Precedent is genuinely mixed — T-057/T-073 added entries, T-022/T-036/T-044/T-072 did not — so this is a judgement call, not a rule violation; but a convention change visible in every user's `git log` warrants one. | `CHANGELOG.md:9-19` | Add an `### Changed` entry under `[Unreleased]`. |

**Disposition summary:** 9 findings — 2 blocking (F1, F2; not dispositioned, they are fixed),
7 non-blocking: 6 `fixed inline` (F3, F4, F5, F6, F8, F9 — folded into the rework pass, since the
branch is returning anyway) and 1 `noted` (F7). No follow-up ticket spawned: every non-blocking
finding is a one-clause prose fix in a file the rework already touches, so none passes the
promotion test ("would this actually be scheduled?") as separate work.

**Rework scope:** F1 and F2 are mandatory. F3–F6, F8 and F9 are in scope for the same pass —
they are the same drift in adjacent surfaces, and splitting them across two passes would leave
the payload half-consistent twice.

**Rework fix pass (2026-08-09), commit `04010c5` on the same branch:**

| id | resolution |
|---|---|
| F1 | Added a carve-out to `MarkerBlock()`'s "Branch & commit" bullet in `internal/install/install.go` ("… for child-project code. Ticket/board bookkeeping uses its own `board: T-NNN <verb phrase>` form instead—see 'Where commits land' below"); regenerated `internal/install/testdata/markerblock.golden`; hand-mirrored the identical wording into this repo's own `AGENTS.md` in the same commit, per the self-modify policy. `TestSelfHostMarkerBlockIsCurrent` and `TestMarkerBlockGolden` both pass. |
| F2 | Restated the scope test in `tickets-README.md` §0 by file set ("limited to ticket files … and `tickets/BOARD.md`") instead of by directory, and named `tickets/NOTES.md` explicitly as excluded despite being under `tickets/`. |
| F3 | `skill/SKILL.md`'s "Branch & commit" default now carries the same carve-out; its "Commit policy" bullet and step-4 Finish text both name the root-path tidy-up/keep-history default. |
| F4 | Moved the `board:` mention out of `project-structure.adoc`'s root-child-only list into the universal "The board: generated, and still committed" section, with a note that it applies to every project regardless of child count; the merge-strategy bullet (genuinely root-specific) stayed in the root-child list. |
| F5 | `TEMPLATE.md`'s branch section (the first "finalize the branch…" clause, previously unedited) now states the root-path keep-history default alongside squash, matching the Finish section. |
| F6 | Added a worked example (`board: T-084 ready → in development`) and a second example for a multi-id, content-only case; defined "adjacent" (immediately preceding bookkeeping commit for the ticket, same sitting) and named `git commit --amend` as the fold mechanism. |
| F7 | No code change — disposition was already `noted`; fixed forward (this review's own bookkeeping commit, and this rework's, both use `board:`). |
| F8 | Deferred to the Finish tidy-up when this ticket is re-reviewed and finalized: the branch's commits will be tidied into atomic, correctly scoped commits before the final message is presented, per decision 5/F8's own suggestion. |
| F9 | Added a `### Changed` entry to `CHANGELOG.md`'s `[Unreleased]` section. |

All four commands re-run clean on the fix commit: `just build` ✓ · `just test` ✓ (including
`TestSelfHostMarkerBlockIsCurrent`, `TestMarkerBlockGolden`, `TestPayloadDispositionVocabulary`,
`TestPayloadDefersToProjectConfig`) · `just lint` ✓ · `just docs-check` ✓.

---

### Scoped re-review (2026-08-09)

Scope per the protocol: verify the findings above are resolved; no re-audit from scratch.
Verification was cross-checked by an independent sub-agent (the rework's author reviewing their
own fix carries the same bias the applicability gate spawns a fresh agent to avoid).

**Findings F1–F9 — all resolved.** F1's blanket rule is gone (carve-out at
`install.go:898-901`, golden regenerated, `AGENTS.md` mirrored, `TestSelfHostMarkerBlockIsCurrent`
passes). F2's scope test is now an **allow-set** — walked through every case an agent hits
(ticket file only; `BOARD.md` only; ticket + `BOARD.md`; `NOTES.md` only; ticket + `NOTES.md`;
ticket + `pickle.toml`; `tickets/*/.gitkeep`) and each yields exactly one answer, including the
`tickets/NOTES.md` case that broke the original rule. F3–F6 and F9 are present and mutually
coherent; the grammar is still defined in exactly one place (`tickets-README.md` §0) with the
other surfaces carrying only a shape-reminder plus a pointer. Scope discipline held: `04010c5`
touches 8 files, each traceable to a listed finding. F7 fixed forward (`e986e0a`, `c64e249` both
use `board:`). F8 remains deliberately deferred to the Finish tidy-up.

**Two new findings, introduced by the fix pass itself — both fixed inline in `324bffd`:**

| id | severity | disposition | finding | evidence | resolution |
|---|---|---|---|---|---|
| N1 | non-blocking | fixed inline | The F1 carve-out's cross-reference **dead-ended**: it read "see ‘Where commits land’ below", but that bullet covers only *which branch* bookkeeping lands on — no `board:` mention, no grammar, no scope rule, no pointer onward. In the one surface `SKILL.md` declares authoritative, an agent following it learned nothing about scope. `SKILL.md`'s own carve-out cited "rules §0" correctly, making this an internal inconsistency. | `markerblock.golden:26-27` pointing at `:38-43` | Re-pointed at "the rules §0", matching the block's existing `(rules §5)` idiom; golden regenerated; `AGENTS.md` hand-mirrored. |
| N2 | non-blocking | fixed inline | **Decision 7's premise was unfulfilled.** It asserted the format "lives in step 9's prose (which the reviewer already reads when composing messages)" — but step 9's Overarching-project bullet, the operative moment a reviewer writes the bookkeeping commit, never mentioned `board:`. Missed by the first review too. | `review-protocol.md:214-216` (pre-fix) | Added one clause naming the form and citing the rules §0. The reviewer-facing checklist stays untouched, per decision 6. |

Both qualify for `fixed inline` on the rules §5 bar: N1 is prose **this branch authored**, N2 is
prose **this branch made false**; neither changes behaviour, and neither would plausibly be
scheduled as its own ticket. Fixing them here avoided a rework round for two clauses.

**Disposition summary (re-review):** 2 new findings, both non-blocking, both `fixed inline`
(N1, N2). No blocking findings. No follow-up ticket spawned.

**Gates after the inline fixes** (`324bffd`): `just build` ✓ · `just test` ✓ · `just lint` ✓ ·
`just docs-check` ✓.

**Verdict: DONE.** The convention is internally consistent across all six surfaces that state it
(rules §0 as the single definition; `SKILL.md`, `TEMPLATE.md`, `review-protocol.md`, the marker
block/`AGENTS.md`, and the user manual as referrers), and every pointer now resolves to real
content.

**Impact sweep (step 8):** no ticket in `1-to-do/` or `2-ready/` references T-084, and none lists
it in `depends-on:`. One observation, not a finding: T-082's Description argues from "a
squash-merge folds bookkeeping". Under this ticket's rebase default that mechanism changes shape
(rebase *preserves* the bookkeeping commits rather than folding them), so the hazard stands but
its rationale prose will want a refresh when T-082 is picked up. Deliberately **not** patched
now — T-084 is unmerged and its convention may still shift in rework.

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-09 — TO DO → READY: plan complete
- 2026-08-09 — READY → IN DEVELOPMENT: picked up
- 2026-08-09 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-09 — IN REVIEW → REWORK: 2 blocking: marker block states a blanket commit format (decision 7 false premise); scope rule self-contradicts on tickets/NOTES.md
- 2026-08-09 — REWORK → IN REVIEW: findings fixed
- 2026-08-09 — IN REVIEW → DONE: re-review clean; F1-F9 resolved, 2 new non-blocking fixed inline
