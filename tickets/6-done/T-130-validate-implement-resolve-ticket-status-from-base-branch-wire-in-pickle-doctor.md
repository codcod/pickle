---
id: T-130
title: validate/implement: resolve ticket status from base branch, wire in pickle doctor
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-130 — validate/implement: resolve ticket status from base branch, wire in pickle doctor

## Outcome

After this ships, a brand-new session told "validate ticket T-NNN" or "implement ticket T-NNN"
under `layout = "in-tree"` resolves the ticket's real current status from the base branch
mechanically — via an unconditional, directory-agnostic lookup and a mandatory `pickle doctor`
pre-flight step — instead of relying on the agent to notice, on its own, that a checked-out
feature branch's worktree copy might be stale.

## Description

T-108 documented the in-tree "mirror-image hazard" (bookkeeping lands on base, so a feature
branch cut earlier carries a stale ticket file) and T-128 built the mechanical detection for it
in `pickle doctor` (`checkStaleTicketBranch`, `internal/doctor/doctor.go:427`) — it already
diffs a feature branch's own ticket against its base-branch copy and warns naming both
directories and the fix (rebase onto base). Plain `pickle doctor` (no `-v` needed — warnings
always print; `-v` only adds the `Passed` lines) already surfaces this warning today. It did not
alleviate the field problem it was built for.

**Why not, concretely.** Neither `skill/resources/review-protocol.md` nor `skill/SKILL.md`
(inline "Procedure: implement a ticket" / "Procedure: validate a ticket") ever tells the agent
to run `pickle doctor` — grepped both, zero hits. The correct, already-shipped check is purely
opt-in, and a fresh session with no memory of T-108/T-128 has no reason to opt in.

Separately, `review-protocol.md` step 1 already carries a *correct* conditional instruction —
*"if the feature branch is already checked out, `git show <base>:tickets/4-in-review/T-NNN-*.md`
rather than the worktree copy"* — but two things blunt it: it is conditional prose an agent can
just not act on (it can simply `Read` whatever ticket file its worktree happens to show and
never reach the `git show` branch of that sentence), and it hardcodes the assumed target
directory (`4-in-review/`), so even a careful reader gets a `git show: path does not exist` dead
end if the ticket is actually somewhere else on base. `SKILL.md`'s inline "Procedure: implement a
ticket" step 1 is plainer still — *"Read the ticket in full. It must be in `2-ready/`"* — with no
base-branch caveat at all, despite T-128's own motivating incident (its History's first line)
being an agent resuming *implement* work on a stale branch, not a *validate* run.

**Fix, two parts, doc-only (no new detection logic — T-128 already built the right one):**

1. **Make ticket lookup unconditional and directory-agnostic.** In `review-protocol.md` step 1
   and `SKILL.md`'s implement-procedure step 1, replace "assume/confirm the directory" with a
   lookup that always resolves the ticket's actual current path from base under
   `layout = "in-tree"` — the same `git ls-tree -r --name-only <base> -- tickets` scan
   `checkStaleTicketBranch` already performs internally (`internal/doctor/doctor.go:471`), given
   as a runnable command in the doc, run regardless of what the current worktree happens to show.
2. **Promote `pickle doctor` to a mandatory step, correctly sequenced.** Add it — same procedural
   weight as existing numbered/lettered steps, not another skippable aside — to both the
   "implement a ticket" and "validate a ticket" procedures, run once the ticket's own feature
   branch is checked out (the hazard only exists once `HEAD` is on `feat/T-NNN-*`; it is safe to
   run unconditionally otherwise too, since the check fails open — silent skip — when no matching
   branch is checked out, e.g. a fresh pickup that hasn't cut its branch yet). Any stale-ticket-
   branch warning must be resolved (rebase onto base, the warning's own suggested fix) before the
   procedure proceeds.

**Soft couplings (not `depends-on`, both already `6-done/`):**

- **T-128** — supplies `checkStaleTicketBranch`, the mechanism this ticket wires in; this ticket
  changes no Go code, only where the procedures point.
- **T-108** — the mirror-image hazard this whole chain (T-108 → T-128 → this ticket) traces back
  to; its `tickets-README.md` §0 prose is unaffected, only the two trigger procedures change.

## Implementation Plan

### 0. Feature branch (mandatory)

Root-path child (`path = "."`):

```
cd .
git checkout main
git checkout -b feat/T-130-validate-implement-resolve-status-from-base
```

Tidy WIP commits into atomic ones before presenting, per `tickets-README.md` §0. Publish only
after user approval (no push/MR without it).

### Prerequisite gate (hard)

None. T-108 and T-128 are both `6-done/`; no `depends-on:` — both couplings are soft (reuse of
already-shipped detection, not a blocker).

### Confirmed design decisions (do not deviate without asking)

1. **Doc-only change; no Go code touched.** `checkStaleTicketBranch` (T-128) already does the
   right detection; this ticket only changes where the two trigger procedures point.
2. **`pickle doctor` is mandated bare, not `pickle doctor -v`.** `internal/cli/install.go`'s
   `runDoctor` prints `Warnings` unconditionally; `-v` only adds `Passed` lines. The stale-ticket-
   branch warning this ticket relies on needs no flag to surface.
3. **New review-protocol.md step is a new top-level `## 0a.` heading, not a renumbered step.**
   `## 4a.`/`## 4b.` already establish the lettered-sub-step convention in this file. Every
   integer step in `review-protocol.md` is cross-referenced by number from outside the file
   (`tickets-README.md`, `docs/user-manual/concepts/*.adoc`, `CHANGELOG.md`, `SKILL.md`, and
   historical `6-done/` tickets) — renumbering step 1 onward would be a much larger, riskier diff
   for no behavioural gain. `## 0a.` sits between `## 0.` (reviewer independence) and `## 1.`
   (load context), after the point where the callout box already tells the reader to check out
   the ticket's feature branch to audit it.
4. **`SKILL.md`'s inline short-form lists gain a preamble sentence, not a renumbered list item.**
   Its "Procedure: implement a ticket" (1–8) and the "In short" list under "Procedure: validate a
   ticket" (1–5) are not cross-referenced by number anywhere else in the repo (grepped), so
   renumbering would be safe either way — but a preamble sentence before item 1 is the smaller
   diff and reads naturally as a precondition rather than another item to satisfy in sequence.
5. **The `git ls-tree` lookup recipe is stated once per call site, not centralized.** Matches this
   payload's existing style — `review-protocol.md`'s "read from base" callout is already a
   restatement of `tickets-README.md` §0's mirror-image-hazard prose, not a cross-reference to
   it. Three short call sites (`review-protocol.md` step 1, `SKILL.md` implement step 1,
   `SKILL.md` validate short-form) don't earn a shared snippet file.
6. **No `docs/user-manual/` edit.** Grepped `docs/` for this procedure's language (`Load
   context`, `mirror-image`, `stale ticket`, `Locate the ticket`) — no hits. The skill payload
   prose being edited *is* the documentation surface for this behaviour; no separate user-manual
   page restates it.
7. **Payload-lint safe.** `payload_lint_test.go` (T-099) fails the build on a ticket-id the
   reader must go look up, a repo-only path, first-person "this repo", or invisible-evidence
   claims, inside `skill/`. New prose uses only generic `T-NNN`/`<base>` placeholders already
   used throughout both files and names no `T-1NN` id inside the payload — ticket-bookkeeping
   files (`tickets/`) are outside payload-lint's scope, so this ticket's own History/Description
   may name T-108/T-128 freely.

### Tasks

#### Task 1 — `skill/resources/review-protocol.md`: unconditional, directory-agnostic lookup
Rewrite the "Locate the ticket" bullet under `## 1. Load context` (currently: *"Locate the
ticket: `tickets/4-in-review/T-NNN-*.md`. Under `layout = "in-tree"`, read it as it exists on the
base branch — if the feature branch is already checked out, `git show
<base>:tickets/4-in-review/T-NNN-*.md` rather than the worktree copy..."*). New text drops the
"if already checked out" condition and the hardcoded `4-in-review/` path: under
`layout = "in-tree"`, first resolve the ticket's actual current path from base —
`git ls-tree -r --name-only <base> -- tickets | grep -- "/T-NNN-"` — then read it with
`git show <base>:<that path>`, unconditionally (not gated on whether a feature branch happens to
be checked out yet). Keep the existing `umbrella`-layout paragraph and the child's-own-worktree
caveat as-is — only the in-tree bullet changes. Keep the box above (the "mirror-image hazard" /
"read the ticket from the base branch" callout before step 0) as-is; it still frames *why*.

#### Task 2 — `skill/resources/review-protocol.md`: new `0a.` mandatory doctor step
Insert a new top-level heading — two `#` characters, "0a. In-tree stale-branch check" — between
`## 0. Reviewer independence` (ends line 105) and `## 1. Load context` (currently line 107),
with this body:

> Under `layout = "in-tree"` only: once the ticket's own `feat/T-NNN-<slug>` branch is checked
> out, run `pickle doctor`. It fails open (silent skip) when no matching feature branch is
> checked out yet, so running it unconditionally here is safe. A `WARNING: ticket T-NNN: this
> branch has it in "X" but <base> has it in "Y" — rebase onto <base> ...` (or the equivalent
> History-drift warning) means this worktree's ticket copy is stale: rebase onto `<base>` before
> continuing, then re-run `pickle doctor` to confirm it is now clean. Under the default
> `umbrella` layout this check does not apply (`checkStaleTicketBranch` is a no-op outside
> `in-tree`) — skip straight to step 1.

Update the checklist template near the end of the file (`- [ ] Reviewer independence settled
(step 0): ...` block, around line 424) with one new line: `- [ ] In-tree stale-branch check
(step 0a, in-tree layout only): pickle doctor run, no unresolved stale-ticket-branch warning —
or n/a under umbrella`.

#### Task 3 — `skill/SKILL.md`: "Procedure: implement a ticket"
Add one preamble sentence directly before its numbered list (before "1. **Read the ticket in
full.**..."): under `layout = "in-tree"`, before reading the ticket, resolve its current status
from base the same way (`git ls-tree -r --name-only <base> -- tickets | grep -- "/T-NNN-"`, then
`git show`) rather than trusting the worktree, and — once any pre-existing `feat/T-NNN-*` branch
for this ticket is checked out (a resumed pickup; a fresh one has no branch yet) — run
`pickle doctor` and resolve any stale-ticket-branch warning first. Do not renumber the existing
1–8 list.

#### Task 4 — `skill/SKILL.md`: "Procedure: validate a ticket"
In the short-form paragraph ("Follow `resources/review-protocol.md` ... Before auditing, settle
reviewer independence (the protocol's step 0) ... In short:"), add one clause referencing the new
step: "...and, under `layout = "in-tree"`, resolve the ticket from the base branch and run
`pickle doctor` (the protocol's step 0a) before auditing." Do not renumber the existing "In
short" 1–5 list.

#### Task 5 — sweep the symlinked view
`.agents/skills/brine/` is a symlink to `skill/` (per this repo's root `CLAUDE.md`) — confirm
`git status` shows no separate diff needed there; if the symlink is somehow not a plain symlink
in this checkout, copy the same edits across and say so in the summary.

### Acceptance test

- `just docs-check` clean (asciidoc xref checker + payload lint both run under it, per
  `pickle.toml`'s configured commands).
- `just test` clean, specifically confirming `payload_lint_test.go`'s
  `TestPayloadSpeaksToAForeignReader` (and its sibling `TestPayloadLintRule*` tests) still pass
  against the new prose in `skill/`.
- Manual read-through: `git show HEAD:skill/resources/review-protocol.md | grep -n "^## "` still
  lists `0`, `0a`, `1`, `2`, `3`, `4`, `4a`, `4b`, `5`–`9` in order, and every external
  cross-reference found in refinement (`tickets-README.md:97,559`, `docs/user-manual/concepts/
  lifecycle.adoc:87,133,141`, `docs/user-manual/concepts/agent-session-workflow.adoc:75`,
  `CHANGELOG.md:122,132,142,277`, `skill/SKILL.md:229`) still resolves to the same step number it
  named before this change — `grep -n "step [0-9]" review-protocol.md` diffed before/after the
  edit, confirming only `0a` and its body are new, nothing renumbered.
- `just lint`, `just build` clean (should be no-ops for a doc-only change, but run as the
  project's standard gate).

### Docs update (mandatory when user-facing)

No `docs/user-manual/` surface — confirmed in refinement (decision 6): the edited skill payload
prose (`skill/SKILL.md`, `skill/resources/review-protocol.md`) is itself the documentation for
this behaviour, and no separate user-manual page restates it.

### Finish (mandatory)

1. `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. No docs registration needed (see above).
3. Write the summary (files touched, decisions made, anything deferred) and hand back.
4. Suggested commit message: `docs(skill): resolve ticket status from base branch, mandate
   pickle doctor in implement/validate (T-130)` — tidy the branch's WIP commits into this one
   atomic commit first (root-path child, `tickets-README.md` §0), then present it; do not push
   without approval.

## Review

- [x] Reviewer independence settled (step 0): **independent** — fresh session with no memory of
  writing the branch, no hand in this ticket's implementation. Audits run directly, no delegation
  needed.
- [x] In-tree stale-branch check (step 0a — not yet shipped on `main` at review time, so this is
  the outgoing protocol's step 1 conditional read; run anyway as a dogfood of what this very
  ticket ships): `./pickle doctor` on the checked-out feature branch reported
  `WARNING: ticket T-130: this branch has it in "3-in-development" but main has it in
  "4-in-review" — rebase onto main to pick up the move` — exactly the mirror-image hazard this
  ticket documents (the branch was cut mid-development, before the later `in-development →
  in-review` bookkeeping move landed on `main`). Ticket read from `main` throughout, per protocol
  step 1; the warning confirms the detection this ticket wires in behaves as described.
- [x] Implementation audit (steps 1, 2): all 5 tasks done as specified —
  `skill/resources/review-protocol.md` gained `## 0a.` (between `## 0.` and `## 1.`) and the
  rewritten directory-agnostic "Locate the ticket" bullet; the checklist template gained the new
  `0a` line; `skill/SKILL.md` gained the preamble sentence before "Procedure: implement a ticket"'s
  numbered list and the added clause in "Procedure: validate a ticket"'s short-form paragraph;
  `.agents/skills/brine/` confirmed a plain symlink, no separate edit needed (`git status`
  clean on the feature branch). Acceptance test re-run on the feature branch: `just build`,
  `just test` (including `go test . -run
  'TestPayloadSpeaksToAForeignReader|TestPayloadLintRule' -v -count=1`, all green, no cache),
  `just lint`, `just docs-check` all clean. `grep -n "^## "` on
  `review-protocol.md` lists `0, 0a, 1, 2, 3, 4, 4a, 4b, 5-9` in order; every external
  cross-reference checked (`docs/user-manual/concepts/lifecycle.adoc`,
  `docs/user-manual/concepts/agent-session-workflow.adoc`, `CHANGELOG.md`, `skill/SKILL.md`)
  still resolves to the same step number as before — nothing renumbered.
- [x] Quality audit (step 3): doc-only change, no Go code touched, matching decision 1. New prose
  is idiomatic with the file's existing style; no code paths, tests, or error handling affected.
- [x] Consistency audit (step 4): new `0a` and rewritten step-1 bullet are internally consistent
  with the unchanged "mirror-image hazard" callout box above them and with each other. One
  pre-existing, unrelated inconsistency surfaced (see F1) — not introduced by this branch.
- [x] Documentation audit (step 4a): confirmed no `docs/user-manual/` page restates this
  procedure's language (grepped for "Load context", "mirror-image", "stale ticket", "Locate the
  ticket" — no hits), matching decision 6; `just docs-check` clean.
- [x] Docs-readability pass (step 4b): no docs-readability reviewer configured in this host —
  conscious skip.
- [x] Findings recorded below; disposition summary and cost line present (step 5).
- Ticket moved per step 6 below.
- [x] Other references / governing documents (step 7): `BOARD.md` needs no hand edit (regenerated
  by the move). F1 below is a pre-existing governing-document drift, out of this branch's
  reach to cause and dispositioned rather than fixed inline.
- [x] Remaining-tickets impact sweep (step 8): no ticket in `1-to-do/` or `2-ready/` references
  T-130 in `depends-on:` or Description — nothing to patch.
- Summary + commit message below, for approval (step 9).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | stale-xref | note and close | `tickets/README.md` points to `.agents/skills/ticket-flow/resources/*` but the skill was renamed to `brine` (T-074, done) and the real symlink is `.agents/skills/brine/` | `tickets/README.md:8-12`, confirmed pre-existing on `main` before this branch (`git show main:tickets/README.md`) | rename the three `ticket-flow` path segments to `brine`; unrelated to T-130's scope, a one-line fix for whoever picks it up next |

Disposition summary: 1 non-blocking finding (F1), noted and closed — 0 blocking, 0 new tickets, 0 fix-now.

cost: estimated S, actual S

## History

- 2026-09-18 — created (TO DO). source: field-use: observed across in-tree brine installs — a
  brand-new session told "validate ticket T-NNN" reads ticket status from the checked-out
  feature branch's stale worktree copy instead of base, reporting an already-moved ticket as
  still in its prior status; T-128 built the correct detection but nothing in the trigger
  procedures invokes it, and `review-protocol.md`'s own step 1 presumes the status directory
  instead of resolving it from base.
- 2026-09-18 — TO DO → READY: plan complete
- 2026-09-18 — READY → IN DEVELOPMENT: picked up
- 2026-09-18 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-18 — IN REVIEW → DONE: review complete: 1 non-blocking finding, noted and closed
