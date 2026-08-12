---
id: T-092
title: detect an unfinalized merge: audit every DONE ticket for its merge line, and run board audit in CI
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-092 — detect an unfinalized merge: audit every DONE ticket for its merge line, and run board audit in CI

## Outcome

After this ships, forgetting to finalize a merged ticket is **detected instead of merely
documented**: `pickle board audit` warns for any ticket in `6-done/` with no `merged to <base>`
History line — today that check exists but fires only when the done ticket happens to be some
*other* ticket's `depends-on` target — and CI runs `board audit` on push, which today **nothing
does**: this board is audited only when a human remembers to run it locally.

## Description

**Corrected baseline (the original filing of this ticket got this wrong).** This ticket was first
filed claiming *"nothing in the skill tells the agent what to do once the human reports the merge
happened."* That is false. `resources/tickets-README.md` §4 already prescribes it:

> When the human reports a merge, append a dated `merged to <base>` History line (ideally with a
> commit reference alongside the MR ref, per §1) to the done ticket — the board's `merged` cell
> renders from that line — and run `pickle board sync`.

So the append-the-line and regenerate-the-board steps are **already flow law**, and a new "step 10"
restating them in `review-protocol.md` would put two copies of the same instruction in the payload
— exactly the drift T-081 spent a whole ticket removing for gate definitions. The original filing
oversold a narrow gap by mis-stating the baseline; this amendment fixes that and re-aims the
ticket at what is genuinely missing.

**What is actually missing is enforcement, not instruction.** §4 says *what* to do; nothing checks
whether it was done:

1. **The merged-line check exists but is scoped too narrowly.** `internal/audit/audit.go:239`
   already emits `dependency %s is DONE but has no 'MERGED' History line` — but only for a ticket
   reached as some other ticket's `depends-on` target. A done ticket nobody depends on is never
   checked, so the common case (finalize forgotten, nothing downstream points at it) is invisible.
   T-089, below, is exactly that case.
2. **Nothing runs `board audit` automatically.** Verified at filing: `grep -rn "board audit"
   .github/workflows/` returns nothing. The board is audited only when a human remembers to, on a
   laptop.

Those two together are why the T-081 incident (`tickets/6-done/T-081-…md`) went as far as it did:
a bookkeeping commit staged a ticket move's new path plus the regenerated board, but never the old
path the rename deleted, so the stale copy stayed tracked in `HEAD`; `board audit` stayed clean
because it audits the **worktree**, not `HEAD`; and the corruption surfaced only when merging the
PR restored the stale file and a *hand-run* audit finally reported `duplicate id` (fixed live in
`7e3dbb2`). It was caught by luck of ordering, and CI would never have caught it at all.

**Refinement found a second, live instance — the check's value case is not hypothetical.** Sweeping
all 45 tickets in `6-done/` for a merge History line turned up exactly one without: **T-089**,
merged as PR #26 (`1ceaead`) on 2026-08-09, whose finalize step was simply skipped. Its board
`merged` cell had been blank and wrong for three days, and nothing reported it — no ticket lists
T-089 in `depends-on:`, so the existing dependency-scoped check never looked at it. (T-089 is, with
some irony, the ticket that introduced the commit-reference convention for merge lines.) Restored
on `main` in `20bf168` during this refinement; the sweep that found it is one `grep` — which is the
point: the check is cheap, and its absence hid a real error for days.

**Proposed change (data and tooling, not prose):**

- Generalize the `audit.go:239` check: warn for **every** ticket in `6-done/` whose History has no
  `merged to <base>` line — not only dependency targets. Whether this is a warning or an error,
  and whether bookkeeping-only tickets (which have no branch to merge) need an exemption, are
  refinement decisions; an exemption almost certainly is needed, since a docs/board-only ticket
  legitimately never merges anything.
- Run `pickle board audit` in CI on push/PR, so the board is checked by something other than a
  human's memory.
- Optionally, extend §4's existing sentence with the two genuinely-uncovered residual steps
  (delete the feature branch; conclude any impact-sweep deferral the review left pending the
  merge) — as a clause on the rule that already exists, **not** as a new protocol section.

**Deliberately dropped from the original filing**, recorded so refinement does not re-add them:

- *A nine-step "step 10" finalize procedure in `review-protocol.md`.* The remedy for "an agent
  forgot a step" should not be "write the agent another step to remember" — that is precisely the
  lesson T-081 shipped one day earlier (encode the requirement as data, audit it). Detection also
  needs no trigger: a documented procedure only runs if the human remembers to say "finalize
  T-NNN", which is the same failure mode as the forgotten step.
- *"Verify the merge independently — don't trust the report at face value."* Calling `gh pr view`
  is worth doing because it is where the merge SHA and the actual merge strategy come from, not
  because the human might misreport a merge. Keep the call, drop the framing.
- *"Re-run the child's build/test/lint/docs on the merged base."* Duplicates CI on a PR that was
  already green and mergeable; the merge itself is the only new variable, and a semantic conflict
  that survives a green CI is rare enough not to justify a standing step.
- *Unconditional branch deletion.* Stated as "its last legitimate use was the merge", which is
  wrong while a revert is still plausible. If kept at all (above), it belongs as a soft clause.

**Candidate follow-on, not folded in:** a `pickle ticket merged <id> --pr <ref> --sha <sha>`
subcommand that appends the History line and syncs atomically, mirroring how `ticket move` already
bundles move + History + board-regen. Note this is now *less* pressing than at first filing:
detection catches the forgotten line regardless of whether writing it is automated.

Soft coupling: **T-091** (a bookkeeping commit can stage a move's add without its delete) is the
other half of the same incident — it fixes the staging mistake at its source, while this ticket
makes the omission detectable afterwards. Neither blocks the other; either alone is an
improvement.

Soft coupling: **T-093** (reconcile merged tickets against the changelog) is **weaker than first
written**. The original note claimed T-093's sweep "consumes exactly the `merged to <base>` lines
this ticket makes reliable". T-093's own refinement then settled the opposite: its confirmed
decision 3 takes "what shipped" from **commit subjects**, not History lines, because a merge line
answers "was it merged" rather than "did it ship in this range". The two tickets therefore share a
motivation — bookkeeping that silently drifts from what git actually says — but not a data
dependency, and neither blocks nor materially helps the other.

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-092-unfinalized-merge-audit`, created in the `pickle` child-project's repo (path `.`)
before any change. Slug shortened from the filename's, per the precedent set by
`feat/T-081-gate-table`. Local WIP commits are fine; **no push and no MR without explicit user
approval** (`pickle.toml`: `child_publish_gated = true`), and merging is always the human's.
Because this child is root-path (`path = "."`), tidy the WIP commits by interactive rebase into a
small number of atomic commits and **keep that history** on merge (rules §0) rather than squashing.

Bookkeeping (ticket + `BOARD.md`) is committed on `main`, never on this branch.

### Prerequisite gate (hard)

None. `ticket.HasMergeLine` and `flow.Definition` already ship (T-080/T-081); no ticket needs to
land first.

### Confirmed design decisions (do not deviate without asking)

1. **The new check is a WARNING, never an error.** Rules §3/§4 state plainly that "merging is the
   human's and **may lag**" — a ticket legitimately sits in DONE, unmerged, for as long as the
   human takes. An error would fail the board for a state the flow explicitly permits. This also
   matches the existing dependency-scoped check at `internal/audit/audit.go:239`, which is a
   warning for the same reason.
2. **Consequently the check cannot fail CI, and that is intended.** `internal/cli/board.go:85`
   returns `exitError` only when `len(res.Errors) > 0`; warnings never change the exit code. So
   adding `board audit` to CI buys the **error** class — including the `duplicate id` that the
   T-081 incident produced — while the new merge-line warning stays a visible nudge in the log.
   Do **not** add a `--strict`/warnings-as-errors flag to make it gate; that is a separate ticket
   if it is ever wanted, and it would break decision 1.
3. **Keep the existing dependency-scoped warning as well.** It is not made redundant: it fires on
   the *dependent* ticket's ref ("T-095: dependency T-089 is DONE but has no MERGED line"),
   addressed to someone about to pick up work, whereas the new one fires on the *done ticket's own*
   ref, addressed to whoever forgot to finalize it. Different reader, different ref, no duplicated
   line for the common case (a done ticket nobody depends on).
4. **No exemption for "bookkeeping-only" tickets.** Measured at refinement: of the 45 tickets in
   `6-done/`, 44 carried a merge line and the one that did not (T-089) was a genuine miss, since
   fixed on `main` in `20bf168`. There is no observed population of legitimately-never-merged done
   tickets to exempt, and no frontmatter field that could identify one. If such a ticket ever
   appears, record the decision then rather than inventing a field now.
5. **Scope is `6-done/` only — never `7-dropped/`.** Derive the state from
   `def.DependencySatisfied()` (as the existing check does), not a hard-coded `"6-done"`; a dropped
   ticket has nothing to merge by definition.
6. **No new prose in the skill payload.** Rules §4 already prescribes appending the merge line and
   running `board sync`; this ticket adds detection, not instruction. Do not add a
   `review-protocol.md` step — see the Description for why that was dropped.

### Tasks

1. **`internal/audit/audit.go`** — add a check, immediately after the existing in-development
   dependency loop (currently ending at the `dependency %s is DONE but has no 'MERGED' History
   line` warning, ~line 239): iterate all tickets whose `t.Dir` equals
   `def.DependencySatisfied().Dir` and, when `!ticket.HasMergeLine(def, t.Text)`, emit
   `r.warnf`. Message must name both possibilities honestly, since the audit cannot tell them
   apart from the ticket alone — suggested text: `%s: DONE but has no 'MERGED' History line — not
   merged yet, or the merge line was forgotten (rules §4: append it and run pickle board sync)`.
   Use the same `ref := t.Dir + "/" + filepath.Base(t.Path)` form as its neighbours.
2. **`internal/audit/audit_test.go`** — table cases for: a done ticket with a merge line (no
   warning); one without (exactly one warning, naming its own ref); a **dropped** ticket without
   one (no warning — decision 5); and a done ticket without one that is *also* another ticket's
   `depends-on` target (**two** warnings, on different refs — decision 3, and the regression that
   would catch someone "deduplicating" them later).
3. **`.github/workflows/ci.yml`** — in the existing `build-test` job, after the `build` step
   (which already produces `./pickle`), add a `board audit` step running `./pickle board audit`.
   It needs no extra checkout or setup: the job already has the repo and the built binary. Place
   it last so a genuine build/test failure still reports first.
4. **Verify the CI step actually fails on the error class it is being added for** — see the
   acceptance test's step 3; a step that cannot fail is worse than no step.

### Acceptance test

Run from the repo root on the feature branch:

```
just build && just test && just lint && just docs-check
```

All four green. Then, **in a throwaway directory — never against this repo** (self-modify policy,
`AGENTS.md`):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q .
./pk install --project . --yes                 # or the repo's documented install invocation
./pk ticket new "fixture done ticket" --project <child>
# move it to done, then strip its merge line:
./pk ticket move T-001 done --reason "fixture"
./pk board audit
```

Expected: exit code **0**, and stdout contains exactly one line matching
`1-to-do|…/6-done/T-001-…: DONE but has no 'MERGED' History line`, with the summary reporting
`0 error(s), 1 warning(s)`. Then append a `- <date> — merged to main (abc1234)` line to that
fixture's `## History`, re-run `./pk board audit`, and expect `0 error(s), 0 warning(s)`.

Third, prove the CI step can fail (task 4): in the same throwaway dir, duplicate a ticket file
under a second status directory to reproduce the `duplicate id` **error** class from the T-081
incident, run `./pk board audit`, and confirm a **non-zero** exit — that is precisely what the new
CI step would have caught on `main`.

Finally, confirm the repo's own board is unaffected: `go run . board audit` on `main` still reports
`0 error(s), 0 warning(s)` (it does today, since T-089's line was restored in `20bf168`).

### Docs update (mandatory when user-facing)

- **`docs/user-manual/cli-reference.adoc`**, the `[#cmd-board-audit]` bullet list (~line 504):
  add a bullet for the new warning, next to the existing
  `nothing is in 3-in-development/ with a dependency not yet in 6-done/` bullet that documents its
  dependency-scoped sibling. State that it is a warning *because merging may lag*, so it flags a
  state the flow permits as well as one it does not.
- **Same file, the `[#cmd-board-audit]` preamble** (~line 507): it currently reads *"It exits
  non-zero … when any invariant fails"*, which is wrong for warnings and becomes actively
  misleading once a warning exists that is expected to be routinely present. Correct it to say
  errors exit non-zero and warnings do not. In scope: this ticket is what makes the imprecision
  bite.
- **`CHANGELOG.md`** — one entry under `[Unreleased]` → `### Added` for the new audit warning and
  the CI audit step. (`v0.5.0` shipped on 2026-08-12; `[Unreleased]` is currently empty.)
- No skill-payload change: decision 6.

### Finish (mandatory)

Interactive-rebase the WIP into atomic commits — suggested split: one for the audit check + tests,
one for the CI step, one for docs — then present a summary. Suggested Conventional Commit for the
primary commit:

```
feat(audit): warn when a DONE ticket has no merge History line (T-092)
```

with, for the CI commit, `ci: run board audit in the build-test job (T-092)`. Keep history on
merge (rules §0, root-path child), do not squash. Await explicit approval before pushing or
opening the MR.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: chat design discussion following T-081's finalize-and-
  publish sequence, where the gap this ticket closes was hit live (see Description). Graded
  `medium`/`low`/`S` against the backlog: a docs/protocol-only change (one new review-protocol
  step, one new SKILL.md trigger), no code, no new config surface — comparable to T-091's
  `medium`/`low`/`S-M`, the other half of the same incident, slightly cheaper since this ticket
  adds no new type. Impact `medium`, not `low`: the gap it closes already produced one real,
  shipped-then-caught bug, and the fix it proposes is a mandatory gate, not a nice-to-have
- 2026-08-12 — **re-aimed and retitled** (was: *finalize a ticket after its PR merges: a step 10
  for the review protocol*) after the filing session challenged its own two new tickets. Two
  substantive corrections: (1) the Description's central claim was **factually false** — rules §4
  already prescribes appending the `merged to <base>` line and running `board sync`, so the
  proposed step 10 would have duplicated existing flow law; (2) the remedy is flipped from prose
  (a nine-step protocol section) to data/tooling (generalize the `audit.go:239` merged-line check
  beyond dependency targets; run `board audit` in CI, which nothing does today), because
  prescribing another step for an agent to remember is the failure mode T-081 had just shipped a
  fix for. Four weak steps dropped with reasons recorded in the Description. Grade unchanged at
  `medium`/`low`/`S`: scope narrowed from nine prose steps to two mechanical checks, but the work
  moves from docs into `internal/audit` + a CI job, so cost holds
- 2026-08-12 — TO DO → READY: plan complete
- 2026-08-12 — soft-coupling note to T-093 corrected while refining T-093: its sweep reads commit
  subjects, not merge History lines, so this ticket is not its ground truth (plan unaffected)
