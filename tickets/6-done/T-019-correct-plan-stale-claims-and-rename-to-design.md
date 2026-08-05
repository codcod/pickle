---
id: T-019
title: correct PLAN.md's stale claims and rename it to DESIGN.md
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-019 — correct PLAN.md's stale claims and rename it to DESIGN.md

## Description

**Re-scoped 2026-08-04 from "README accuracy polish" to `PLAN.md`.** This ticket had already
become a `PLAN.md` ticket without being renamed: T-047 superseded its items 1 and 3 (README
rewritten to a shop window, command reference moved to `docs/user-manual/`) and commit `f7b0a0a`
mooted item 2, leaving only the `PLAN.md` synopsis. Its own note said *"consider retitling to
reflect the PLAN.md-only scope"*. Reading `PLAN.md` end-to-end on 2026-08-04 found four further
stale claims alongside it (a fifth — the `init` naming slip — turned up at refinement on
2026-08-05), so the ticket is now the whole `PLAN.md` correction plus the rename.
The superseded README items and their line-ref re-anchoring notes are gone from this Description
(they described text that no longer exists); the trail is in `## History`.

`PLAN.md` is the origin design doc — *"kept for the why (rationale and locked decisions)"*, with
an explicit precedence rule (*"Where this doc and a ticket disagree, the ticket wins"*). That
mitigates but does not remove the hazard: §3 presents **"six locked decisions"** as binding
architecture, and one of them is now false. This is the T-041 / T-022 defect class — stale text
that still reads as authoritative — which is why the impact moves `low` → `low-medium`. It stays
below T-022's `medium-high` precisely because of the "ticket wins" rule and the historical
framing, which `AGENTS.md`'s marker block and the skill payload do not have.

### The stale claims

Six, and each is a **sweep, not a line**: the same claim is restated in several sections, so a
fix that touches only the section named below leaves the doc contradicting itself (exactly how
§13.4 and §5–§7 came to disagree — item 4). Every occurrence is listed per item; line numbers
are as of 2026-08-05 and shift as the edits land, so work from the quoted text.

1. **§3.2 asserts one global id namespace — a "locked decision" that was later unlocked.** It
   reads *"One ticket-id namespace… a single global sequence, `T-NNN`, `max()+1` across all status
   dirs regardless of child-project"*. Shipped behaviour is **per-child counters**:
   `ticket_prefix` in `pickle.toml` (`internal/config/config.go:82`, legal shape at `:32`), and
   `tickets-README.md:111-124` — *"per-child counters"*, *"numbers are only unique within a
   prefix"*, so an id must always be written qualified. **The most serious of the six**: an agent
   designing multi-child work could act on it. Correct it and note that single-global is now the
   degenerate case (children leaving the prefix unset share the legacy `T` counter).
   **Occurrences:** `PLAN.md:41` (§1, *"one ticket-id namespace span all of them"*), `:75-79`
   (§3 decision 2, the locked-decision text), `:228` (§7, *"`max+1` across *all* status dirs,
   one global namespace"*).
2. **§4 says the board is a copied skeleton.** The payload table's `resources/BOARD.md` row calls
   it *"copied into the project as `tickets/BOARD.md` (the only instance-data copy)"*.
   `internal/install/install.go:544-546` refutes it in a comment: *"a generated artifact (T-044):
   **there is no skeleton to copy**"* — `writeBoard` seeds a fresh render. Overturned by T-044.
   **Occurrences:** `PLAN.md:54` (§2, mechanics = *"copying the board skeleton"*), `:123` (§4
   payload table row), `:127` (§4, *"only the board skeleton is instance data"*), `:213` (§7,
   `install` — *"copy `BOARD.md`"*).
3. **§7 says the board stays hand-maintainable.** Its `board sync` bullet: *"the board stays
   hand-maintainable but this rescues it"*. The T-044 decision is the opposite — generated,
   **never** hand-edited, which is the whole reason `NOTES.md` exists for prose. Same overturn as
   item 2; both are the pre-T-044 design still described as current. **Occurrence:**
   `PLAN.md:243`.
4. **§5, §6 and §7 say `install` is interactive — and now contradict §13.4 of the same file.**
   §5: *"`pickle install` prompts (or takes flags)"*; §6: *"gathers these interactively"*; §7:
   *"prompt for name + path"*. No interactive prompt exists anywhere in `internal/install` or
   `internal/cli/install.go` — flags only. §13.4 was rewritten on 2026-08-04 to record exactly
   that, so the file now disagrees with itself. Fixing §13.4 alone sharpened the contradiction
   instead of resolving it; this item closes it. **Occurrences:** `PLAN.md:155`, `:168`, `:213`.
5. **§7's `ticket new` synopsis is an incomplete flag list** *(the original item 4, from the T-030
   review's sweep, finding N5 — folded in rather than spawned, per the T-036 concern about
   unbounded review spawn)*. `PLAN.md:227` reads
   `pickle ticket new "<title>" --project <name> [--impact … --complexity … --cost …]` — missing
   `--spawned-by` (T-024) and `--family` (T-059), and no input contract (T-030). It does not
   *contradict* behaviour, it is incomplete, which is why it was non-blocking; but it is the only
   `ticket new` synopsis in the tree disagreeing with
   `docs/user-manual/cli-reference.adoc`, `internal/cli/ticket.go` and `pickle help`.
   **Dissolved by the prune, not synced:** §7's per-command descriptions become a pointer to
   `cli-reference.adoc` (see below), so the second synopsis stops existing rather than being
   kept in step with a third copy.
6. **§1 calls the command `init` — it never existed.** *"After `init`, a coding agent…"* and
   *"because `init` installed the skill…"* (`PLAN.md:29,32`); the command is and always was
   `pickle install`, which the same section's code block already shows. A naming slip rather
   than a design claim — which is why the title no longer counts the items.

### The rename, and what to prune

Rename `PLAN.md` → **`DESIGN.md`**. The file spends its first fifteen lines explaining it is *not*
a plan; a name needing a preamble to disclaim it is the wrong name, and it calls itself an
*"Origin design doc"*. **Not** into `docs/` — that tree is the published AsciiDoc manual, and this
is internal rationale.

Prune while there, since these duplicate a live source that will drift again (decided at
refinement, 2026-08-05):

- **§11 (migration/dogfooding) and §12 (phased build plan) go entirely** — pure delivered
  history. All five phases shipped; the board is the record. §12 is also what the removed
  progress preamble kept re-stating. The preamble still says *"All five §12 phases are
  delivered"*, so it must be reworded in the same edit or it dangles.
- **§4–§7 are pruned to rationale plus one pointer** to `docs/user-manual/cli-reference.adoc`.
  What survives is the *why* that does not rot: §4's rule that the rules/template/protocol are
  the skill's and are referenced, never copied; §5's multi-agent strategy (one canonical skill
  copy, marker-block injection, per-project scope); §6's config model (overarching vs per-child,
  layered review addenda); §7's design tension about how much the agent uses the CLI versus
  editing files. What goes is the command-by-command, flag-by-flag description the manual owns.

Keep what does not rot: **§1** (goal), **§2** (split judgment from mechanics — the principle the
whole tool rests on), **§3** (multi-project model, once item 1 is corrected), **§8** (how the
agent understands a request), **§9** (packaging rationale), **§10** (vocabulary), **§13**
(resolved questions, rewritten 2026-08-04). Section numbers are **not** renumbered after the
prune, so the many `§N` citations elsewhere in the tree keep resolving.

**Hazard for the prune — resolved both ways:** `T-057` cites **`PLAN.md:241`** (*"Wire it into CI
+ a pre-commit hint"*) as evidence its pre-commit hook was anticipated. That sentence is inside
§7, and it is rationale rather than command description, so it **survives** the prune; T-057's
citation is re-anchored to `DESIGN.md` §7 in the same commit anyway. Never silently delete a line
another open ticket argues from.

### Reference updates the rename requires

`rg -n "PLAN\.md"` before finishing. Outside `tickets/` the tree is already clean — `README.md`,
`docs/`, `AGENTS.md`, `skill/`, `justfile` and `snowball.yaml` never mention it — so the rename
breaks no build, no `docs-check` and no link. **One live reference to update:**

- `tickets/1-to-do/T-057-*.md:71` — cites `PLAN.md:241`. Re-anchor it to **`DESIGN.md` §7, by
  section and not by line**; line numbers rot, which is how this ticket came to exist.

**Leave every dated and archived reference alone:** `tickets/6-done/` (T-047, T-030, T-024,
T-036, T-008, T-006, T-054 — a permanent archive, rules §3) **and** `tickets/NOTES.md:152`,
whose 2026-08-03 recalibration row cites *"the stale `PLAN.md:227` synopsis"* as the evidence it
weighed that day. Both record what was true when they were written; rewriting them to match a
later rename would falsify them. The breadcrumb that keeps them resolvable is one line in
`DESIGN.md`'s preamble saying it was formerly `PLAN.md`.

### Not in scope

No `CHANGELOG.md` entry: the changelog records user-facing changes to the shipped tool, and
`DESIGN.md` is internal rationale that no release artifact carries.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child is **`pickle`, which is this repo at `.`** (self-hosted). Before any change:

```
git checkout main
git pull --ff-only
git checkout -b feat/T-019-correct-plan-stale-claims-and-rename-to-design
```

Do all work on this branch, committing locally as you go. **Never push or open a merge request
without explicit user approval** (commit policy): end with a summary and a suggested commit
message; the human merges.

### Prerequisite gate (hard)

- `depends-on:` is empty — nothing to wait for.
- Working tree clean and `pickle board audit` green **before** starting, so any audit failure at
  the end is this ticket's doing.
- This ticket edits `tickets/1-to-do/T-057-*.md`. Per `AGENTS.md`'s self-modify policy that is a
  hand edit inside this ticket's diff — do **not** run `pickle install|upgrade` at any point.

### Confirmed design decisions (do not deviate without asking)

1. **The file is renamed to `DESIGN.md`, at the repo root** — `git mv PLAN.md DESIGN.md`, so the
   history follows. Not into `docs/`: that tree is the published AsciiDoc manual, this is
   internal rationale.
2. **Correct by sweep.** Each of the six claims is fixed at *every* occurrence listed in the
   Description. A claim corrected in one section and left standing in another is a fail, not a
   partial pass.
3. **§11 and §12 are deleted outright**, and the preamble's *"All five §12 phases are
   delivered"* sentence is reworded so it does not reference a section that no longer exists.
4. **§4–§7 are pruned to rationale + one pointer** to `docs/user-manual/cli-reference.adoc`.
   Keep §4's referenced-not-copied rule, §5's multi-agent strategy and per-project install
   scope, §6's overarching-vs-per-child config model and layered review addenda, §7's design
   tension paragraph. Drop the per-command and per-flag descriptions.
5. **§7's audit sentence survives the prune** — *"Wire it into CI + a pre-commit hint"* is
   rationale and T-057 argues from it. Keep it as prose in the pruned §7.
6. **T-057's citation is re-anchored to a section, never a line** (`DESIGN.md` §7).
7. **Nothing in `6-done/` and nothing in `tickets/NOTES.md` is touched.** Those are dated
   records; the preamble's "formerly `PLAN.md`" breadcrumb is what keeps them resolvable.
8. **Sections that stay, corrected but not pruned:** §1 (goal), §2 (split judgment from
   mechanics), §3 (multi-project model — still *six* locked decisions; decision 2 is rewritten,
   not removed), §8, §9, §10 (vocabulary), §13 (resolved questions, rewritten 2026-08-04).
   §13 is closed — do not re-litigate or re-word its resolutions.
9. **No `CHANGELOG.md` entry**: internal doc, not a user-facing change.
10. **Docs-only ticket.** Nothing under `internal/`, `skill/` or `docs/` changes. If a claim
    seems to call for a code or skill fix, stop and ask — that is a different ticket.

### Tasks

#### Task 1 — rename the file and rewrite the preamble

`git mv PLAN.md DESIGN.md`. Then in the preamble:

- add the breadcrumb: this file was **formerly `PLAN.md`** (renamed by T-019), so the many
  `PLAN.md:NNN` citations in `tickets/6-done/` and `tickets/NOTES.md` still resolve;
- replace *"All five §12 phases are delivered"* with a sentence that survives §12's deletion —
  everything this doc planned has shipped, and `tickets/BOARD.md` is the only record of what is
  delivered;
- keep the *"Where this doc and a ticket disagree, the ticket wins"* precedence rule and the
  "not the live tracker" framing verbatim — they are what keeps the remaining hazard low.

#### Task 2 — claim 1: per-child id counters (the serious one)

Rewrite all three occurrences (§1, §3 decision 2, §7) to state the shipped rule: `ticket_prefix`
per child in `pickle.toml` (default `T`), ids are `<PREFIX>-NNN` with `max()+1` **within a
prefix**, numbers unique only within a prefix so an id is always written qualified. Say
explicitly that the original single-global namespace is now the **degenerate case** (children
that leave `ticket_prefix` unset share the legacy `T` counter), and mark decision 2 as
**superseded** rather than silently rewriting it — a locked decision that was later unlocked
should read as such. Ground truth: `internal/config/config.go:32,36,82,197-198` and
`skill/resources/tickets-README.md` §3 ("IDs (`<PREFIX>-NNN`, per-child counters)").

#### Task 3 — claims 2 and 3: the board is generated (T-044)

§2, §4 (table row + the sentence after it) and §7: the board is **not** a copied skeleton and
**not** hand-maintainable. `install` seeds `tickets/BOARD.md` as a fresh render only when absent
(`internal/install/install.go` `writeBoard`: *"a generated artifact (T-044): there is no skeleton
to copy"*); it is regenerated wholesale by `ticket new` / `ticket move` / `board sync`, never
hand-edited, and `tickets/NOTES.md` is where prose lives. Note the overturn by T-044 rather than
pretending the doc always said this. The `check-board.py` rows stay — they describe the origin
workspace accurately.

#### Task 4 — claim 4: `install` is non-interactive

§5, §6 and §7: drop *"prompts (or takes flags)"*, *"gathers these interactively"* and *"prompt
for name + path"*. `install` is **flags only** and always writes `pickle.toml`; no interactive
prompt exists in `internal/install` or `internal/cli/install.go`. Make §5/§6/§7 agree with
§13.4, which already records this and gives the reason (non-interactive is what makes `install`
scriptable and testable). §13.4 itself is not edited.

#### Task 5 — claim 6: `init` → `install`

§1's two `init` references (`PLAN.md:29,32`) become `pickle install`, matching the code block
directly above them.

#### Task 6 — the prune (§11, §12, and §4–§7's command descriptions)

Delete §11 and §12 in full. **Renumber nothing** — §13 stays §13, so every existing `§N`
citation elsewhere in the tree keeps resolving. Prune §4–§7 per decision 4, ending §7 with a
pointer along the lines of *"the live command surface, with every flag, is
`docs/user-manual/cli-reference.adoc`"*. Claim 5 (the stale `ticket new` synopsis) must be gone
by the end of this task — verify with `rg -n 'ticket new "<title>"' DESIGN.md` returning nothing.

If deleting §11/§12 leaves a dangling `§11`/`§12` reference anywhere in `DESIGN.md`, fix the
referring sentence in the same task (`rg -n '§1[12]' DESIGN.md`).

#### Task 7 — re-anchor T-057

In `tickets/1-to-do/T-057-guard-ticket-bookkeeping-from-landing-on-a-feat-branch-pre-commit-hook.md`
(line ~71), change `PLAN.md:241` to `DESIGN.md` §7, keeping the quoted *"CI + a pre-commit
hint"* evidence intact. Confirm the quoted phrase still exists in the pruned §7 — if it does
not, Task 6 went too far and Task 6 is what gets corrected, not T-057's argument.

#### Task 8 — final sweep

`rg -n "PLAN\.md" .` must report **only** `tickets/6-done/` files and `tickets/NOTES.md:152`
(dated records, deliberately untouched), plus the `DESIGN.md` breadcrumb and this ticket's own
prose. Any other hit is a missed reference.

### Acceptance test

Run from the repo root on the feature branch; every command must pass.

```sh
# 1. The rename happened, with history preserved.
test ! -e PLAN.md && test -f DESIGN.md
git log --follow --oneline -- DESIGN.md | tail -1   # shows the pre-rename history

# 2. No stale claim survives (each must print nothing).
rg -n 'one global sequence|one global namespace|one ticket-id namespace' DESIGN.md
rg -n 'board skeleton|copy `BOARD.md`|hand-maintainable' DESIGN.md
rg -n 'prompts \(or takes flags\)|gathers these interactively|prompt for name \+ path' DESIGN.md
rg -n '`init`' DESIGN.md
rg -n 'ticket new "<title>"' DESIGN.md

# 3. The prune landed and left nothing dangling.
rg -n '^## 11\.|^## 12\.' DESIGN.md            # nothing
rg -n '§1[12]' DESIGN.md                        # nothing
rg -n '^## 13\. Resolved questions' DESIGN.md   # still there, not renumbered

# 4. What must survive, survived.
rg -n 'formerly `PLAN.md`' DESIGN.md
rg -n 'the ticket wins' DESIGN.md
rg -n 'CI \+ a pre-commit hint' DESIGN.md
rg -n 'cli-reference.adoc' DESIGN.md
rg -n 'DESIGN.md' tickets/1-to-do/T-057-*.md    # re-anchored, and no ':241' remains

# 5. Only dated records still say PLAN.md.
rg -n 'PLAN\.md' --glob '!tickets/6-done/**' --glob '!tickets/NOTES.md' \
   --glob '!tickets/*/T-019-*' --glob '!tickets/BOARD.md' --glob '!DESIGN.md' .   # nothing

# 6. The repo is still sound.
just build && just test && just lint && just docs-check
./pickle board audit
```

Expected: `board audit` reports `0 error(s), 0 warning(s)`; `docs-check` is unaffected (the
manual never referenced `PLAN.md`).

One check a reviewer must make by reading, not grepping: **§3 still presents six locked
decisions**, decision 2 now describes per-child counters and is marked superseded, and the pruned
§4–§7 still read as rationale — not as a stub that lost the *why*.

### Docs update

No user-facing docs change: `docs/user-manual/` never referenced `PLAN.md` and gains nothing from
the rename (`just docs-check` is run only to prove that). No `CHANGELOG.md` entry (decision 9).
The doc change *is* the deliverable.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean;
   `pickle board audit` reports 0 errors, 0 warnings.
2. No docs registration needed (see above).
3. Write a summary: files touched, what each of the six claims now says, what §4–§7 lost, and
   anything kept that the plan listed as prunable (with the reason).
4. Suggest a Conventional Commit message, e.g.:

   ```
   docs(design): correct PLAN.md's stale claims and rename it to DESIGN.md (T-019)

   <body — the six claims, the prune, the T-057 re-anchor>
   ```

5. Commit locally on the ticket branch. **Do not push or open a merge request without user
   approval.** Present the message; after approval finalize (squash or keep history), push and
   open the MR — merging is the human's.

## Review

Reviewed 2026-08-05 on `feat/T-019-correct-plan-stale-claims-and-rename-to-design`
(commit `7a039bf`), against `main`.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — skipped: the session's `docs_readability` reviewer failed to start
      (`model_not_supported` from its Gemini backend), a sanctioned conscious skip (step 4b)
- [x] Findings recorded with severity and disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval; bookkeeping committed (step 9)

### Implementation audit

| item | verdict | evidence |
|---|---|---|
| Task 1 — `git mv` + preamble breadcrumb | met | `git log --follow DESIGN.md` reaches `b2f4373 chore: add PLAN.md …`; preamble line 3 carries *"formerly `PLAN.md`"*; *"All five §12 phases are delivered"* → *"Everything this doc planned has shipped"* |
| Task 2 — claim 1 (per-child counters), §1/§3/§7 sweep | met | §1 *"gets a `<PREFIX>-NNN` id from that child's own counter"*, §2 *"per-child counter, `max()+1` within that child's prefix"*, §3 decision 2 rewritten and marked *(superseded)* with the degenerate case spelled out; matches `internal/ticket/ticket.go` + `tickets-README.md` §3 |
| Task 3 — claims 2 & 3 (board is generated), §2/§4/§7 | met | §4 now *"generated fresh … only when the file is absent … never a copied skeleton, never hand-edited"*; matches `internal/install/install.go:544-560` (`writeBoard` skips when the file exists) |
| Task 4 — claim 4 (`install` non-interactive), §5/§6/§7 | met | §5 *"takes **flags** … there is no interactive prompt (§13.4)"*, §6 *"non-interactively (§13.4)"*; §13.4 untouched (`git diff` shows no change below §10) |
| Task 5 — claim 6 (`init` → `install`) | met | `rg '`init`' DESIGN.md` empty; §1 reads *"After `pickle install`"* |
| Task 6 — the prune | met | §11/§12 gone; `rg '§1[12]'` empty; §13 not renumbered; §9's dangling *"see §11"* rewritten rather than left; `rg 'ticket new "<title>"' DESIGN.md` empty |
| Task 7 — re-anchor T-057 | met | `T-057:71` now cites `DESIGN.md` §7; the quoted *"CI + a pre-commit hint"* survives at `DESIGN.md:226` |
| Task 8 — final sweep | met | `rg 'PLAN\.md'` outside `6-done/`, `NOTES.md`, this ticket, `BOARD.md` and the breadcrumb: nothing |
| Decision 10 — docs-only | met | diff touches `PLAN.md→DESIGN.md`, `T-057`, this ticket, `BOARD.md` only — nothing under `internal/`, `skill/`, `docs/` |
| Acceptance test, steps 1–3 and 5–6 | met | re-run verbatim; `just build && just test && just lint && just docs-check` green; `./pickle board audit` → `65 tickets, 0 error(s), 0 warning(s)` |
| Acceptance test, step 4 | partially met | `rg 'the ticket wins' DESIGN.md` prints nothing — see finding N4; verified by reading instead (preamble lines 14-15) |
| Reader's check — §3 still six decisions, §4–§7 still rationale | met | §3 lists 1–6 with decision 2 rewritten-not-removed; §4 keeps referenced-not-copied, §5 the multi-agent strategy, §6 the config model, §7 the design tension |

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | fixed inline | §3 decision 2, written by this branch, cited **`pickle ticket renumber`** as the mechanism for re-homing a ticket — a command that does not exist. Introducing a new authoritative-sounding false claim in the doc whose purpose is removing them. | `DESIGN.md:87` (pre-fix); `tickets/7-dropped/T-060-*.md:203` — *"READY → DROPPED: automation not needed … not a pickle command"*; no `renumber` subcommand in `internal/cli/` | Fixed: the clause now says the renumber is manual and records that T-060 proposed the command and was dropped. |
| N2 | non-blocking | fixed inline | §7's new prose said the section described a *"pre-T-044, interactive design **years** after both were overturned"*. The repo's whole history is 2026-07-23 → 2026-08-05, and `install` interactivity was never *overturned* — §13.4 records it was never built. | `DESIGN.md:220-222` (pre-fix); `git log` first commit `2026-07-23`; `DESIGN.md` §13.4 | Fixed: the three drifts are now named separately, without the false time span and without calling an unbuilt wizard "overturned". |
| N3 | non-blocking | new ticket **T-066** | §7 now points at `docs/user-manual/cli-reference.adoc` as *"every command, every flag, its full contract"*, which makes the manual load-bearing — but it omits `ticket new --family` (shipped by T-059) and `project add --branch-prefix/--wip-dev/--wip-review/--build/--test/--lint/--docs`. Pre-existing gap, not this branch's to fix, but this branch is what makes it matter. | `rg family docs/` → nothing, vs `internal/cli/ticket.go:85` and its own `usage:` string at `:36`; `cli-reference.adoc:130-145` vs `internal/cli/project.go` | Document the flags where their siblings already are; consider a test diffing registered flags against the manual. |
| N4 | non-blocking | noted | Acceptance-test step 4's `rg -n 'the ticket wins' DESIGN.md` cannot ever match: the phrase wraps across two lines (`**the` / `> ticket wins**`) — as it already did in `PLAN.md` before the edit. The plan authored an unrunnable check, and the *"acceptance green"* History line reports it as passing. | `DESIGN.md:14-15`; `git show main:PLAN.md \| grep 'ticket wins'` → nothing | Substance verified by reading (rule kept verbatim). Lesson for future plans: grep acceptance checks must be written against a phrase that survives reflow. |
| N5 | non-blocking | new ticket **T-066** | The same non-existent `pickle ticket renumber` is cited in the **shipped skill payload** (`tickets-README.md` §3) and in an audit code comment — so every installed project carries it. Pre-existing, and out of scope for a docs-only ticket (decision 10 says stop and ask). | `skill/resources/tickets-README.md:122-123`; `internal/audit/audit.go:101` | Re-word both to describe the manual migration and to cite T-060 as dropped. Batched with N3 into T-066. |

**Disposition summary:** 5 non-blocking findings, 0 blocking — 2 fixed inline (N1, N2, both in
`DESIGN.md` prose this branch authored), 2 into one new ticket **T-066** (N3, N5, batched by
theme: shipped text describing the CLI surface has drifted from the CLI), 1 noted (N4).

### Impact sweep (step 8)

- **T-057** (`1-to-do/`) is the only ticket that argues from this file; its citation was
  re-anchored by Task 7 and its evidence sentence survives the prune. No other TO DO / READY
  ticket references `PLAN.md` or the pruned sections.
- `tickets/6-done/` and `tickets/NOTES.md:152` keep their `PLAN.md:NNN` citations by design; the
  preamble breadcrumb is what keeps them resolvable. Verified untouched by the diff.
- **T-066** is new and depends on nothing (lineage `spawned-by: [T-019]` only).

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
- 2026-07-25 — re-anchored by the T-018 re-review: items 1/2 line refs shifted by T-018's README edits; noted that `README.md:102-104` belongs to T-018's rework
- 2026-07-25 — scope extended: item 3 (README.md:316 misstates board audit's severity), from the T-024 review's whole-tree sweep (finding N11)
- 2026-07-26 — patched by the T-009 review (impact sweep): item 1's "--agent is a no-op" clause and item 2's "Zed/Pi outstanding" premise are obsolete (T-009 shipped the agent breadth and tagged P4); line refs below the install section shifted again
- 2026-07-26 — scope shrunk by T-047: items 1 and 3 superseded (README rewritten, command reference moved to docs/ with item 3's error fixed), item 2 mooted by commit f7b0a0a (phased-plan section deleted); only item 4 (PLAN.md synopsis) remains
- 2026-08-04 — re-scoped and retitled: README items 1-3 were already dead (T-047, f7b0a0a), so the ticket becomes the whole PLAN.md correction — four further stale claims found reading it end-to-end (one an overturned "locked decision") plus the surviving synopsis item — and the rename to DESIGN.md; impact low -> low-medium; superseded README line-ref notes dropped from the Description
- 2026-08-05 — refined: title dropped its "five" count after a sixth claim surfaced (§1's non-existent `init` command); each claim expanded to its full occurrence list (the id-namespace claim is in §1/§3/§7, the board-skeleton claim in §2/§4/§7), so the fix is a sweep per claim; prune decided — §11 and §12 deleted outright, §4–§7 cut to rationale + one pointer to cli-reference.adoc (which dissolves the stale `ticket new` synopsis rather than syncing a third copy), no section renumbering; T-057's `PLAN.md:241` re-anchored to `DESIGN.md` §7 by section, with the cited sentence kept; `tickets/NOTES.md:152` and all of `6-done/` left untouched as dated records, resolvable via a "formerly PLAN.md" breadcrumb in the preamble; no CHANGELOG entry; the four dead README items still sitting in the Description were removed. Grade unchanged (low-medium / low / S)
- 2026-08-05 — TO DO → READY: plan complete
- 2026-08-05 — READY → IN DEVELOPMENT: picked up
- 2026-08-05 — READY → IN DEVELOPMENT: picked up. Applicability gate (fresh sub-agent) confirmed all six claims, the prune scope, ground-truth citations and T-057's reference still hold, and WIP/board are clean; one inline amendment applied — acceptance-test step 5's glob widened from `tickets/1-to-do/T-019-*` to `tickets/*/T-019-*` since the ticket has since moved out of `1-to-do/`
- 2026-08-05 — implementation note: acceptance-test step 5's exclusion glob amended inline to also exclude `tickets/BOARD.md` — its T-019 row is a generated reflection of the ticket's own title (which legitimately names "PLAN.md", the file being renamed), not a stray reference
- 2026-08-05 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-05 — IN REVIEW → DONE: reviewed, no blocking findings; 5 non-blocking — 2 fixed inline, 2 batched into new ticket T-066, 1 noted (see ## Review)
- 2026-08-05 — IN REVIEW → DONE: reviewed: no blocking findings; 5 non-blocking (2 fixed inline, 2 -> T-066, 1 noted)
