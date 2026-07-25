# The "validate ticket T-NNN" protocol

The canonical, self-contained procedure to follow whenever the human says **"validate ticket
T-NNN"** or **"review ticket T-NNN"** (synonyms). Any agent that sees either phrase runs
these steps in order. Read the project's
`AGENTS.md`/`CLAUDE.md` first for the ground rules and the project's configured commands, then
the workflow rules (this skill's `resources/tickets-README.md`), then follow this.

**Layered review addenda (extra project-specific audit rules keyed to these step numbers).**
Pickle supports two optional addendum layers, applied **on top of — never instead of** — this
procedure, in this order:

1. the **overarching** addendum (`review_addendum` at the top of `pickle.toml`) — rules true
   of the whole project (cross-child consistency, commit-title convention, …);
2. the **per-child** addendum (`review_addendum` inside the ticket's target `[[project]]`) —
   that child-project's tech-specific checks.

Reviewing a ticket runs **this generic protocol → overarching addendum → the ticket's own
child addendum** (whichever exist).

> **Scope.** A review does **not** re-implement the ticket. It verifies, audits, records
> findings *into the ticket*, and updates the board. Fixes become either a scoped rework pass
> on the same ticket (blocking findings, §5) or a *new* ticket in `tickets/1-to-do/`
> (non-blocking findings, §5) — never inline drift.
>
> **Review on the ticket's feature branch.** The work under review lives on the un-merged
> `feat/T-NNN-<slug>` branch in the ticket's target child-project — check it out to audit and
> to re-run the acceptance test. Create no *additional* review branch. Local commits on the
> ticket branch are allowed at any time; **never push a child-project or open a merge request
> without explicit user approval** (see step 9). End with a summary, the commit message(s) and
> merge-request attributes for approval, and a next-ticket suggestion.

---

## 1. Load context

- Locate the ticket: `tickets/4-in-review/T-NNN-*.md`. Read it in full — `## Description`,
  `## Implementation Plan` (its acceptance test, tasks, and confirmed decisions are the
  checklist for step 2), and `## History`.
- If this is a **scoped re-review** (the ticket was previously in `tickets/5-rework/`), read the
  existing `## Review` section first — only the findings listed there need re-verification; do
  not re-audit the whole feature from scratch.
- Read any project-wide decisions and the configured build/validate commands from the project's
  `AGENTS.md` and the ticket's target `[[project]]` block in `pickle.toml`.
- Check `depends-on:` frontmatter — every listed ticket must be in `tickets/6-done/` **with
  its feature branch merged to the base of _its own_ child-project's repo** (done ≠ merged; a
  cross-child dependency is satisfied only once its branch is merged in the child it targets;
  check the board's `merged` column / the dependency's History).

## 2. Implementation audit — *was the ticket fully implemented?*

Verify against the **actual project tree** (the ticket's target child-project), not the prose:

- Every **task** in the Implementation Plan is done, in the files it names.
- Every **acceptance criterion** is met. **Re-run the acceptance test verbatim** and record the
  result. Run the child-project's configured build / test / lint / docs commands as applicable.
- Every **confirmed design decision** was honoured, including any project-specific rules named
  in `AGENTS.md`.

Record each item as **met / partially met / not met**, with evidence (path, command output,
line reference).

## 3. Quality audit — *was it done to industry best practice?*

- Idiomatic, correct code for the language and framework.
- Test coverage adequate for the change; tests actually assert behaviour.
- Error handling, edge cases, security (input validation, secret handling, injection).
- Docs accurate, complete, and registered.
- Prompt/config content (if applicable) unambiguous and internally consistent.

## 4. Consistency audit — *inconsistencies, contradictions, errors, ambiguities, redundancies, duplications*

Search both **within** the new code and **between** it and the rest of the project:

- Caller ↔ callee contract drift.
- Contradictory instructions across modules/docs.
- Ambiguous or under-specified behaviour.
- Redundant or duplicated logic/content that should be shared.
- Errors of fact, dead references, stale paths.

Use project-wide search — do not eyeball a single file.

## 4a. Documentation audit — *coverage, whole-tree consistency, build health*

If the project ships documentation:

1. **Coverage.** If the ticket ships or changes anything user-facing (a command, flag,
   behaviour, config option, workflow), confirm the docs actually cover it, in the right place,
   registered/linked, cross-referenced rather than duplicated. Missing coverage is a
   **blocking** finding.
2. **Whole-tree consistency sweep.** Search the **entire** docs tree — not only the pages the
   ticket touched — for contradictions, factual errors, stale references, duplication that
   should be shared, and broken links/anchors.
3. **Build the docs** with the project's configured docs command; fix any source errors inline
   (a broken doc build is a build-correctness bug, not new behaviour) and record the result.

Follow any project-specific documentation rules named in `AGENTS.md`.

## 5. Classify and record findings — blocking vs. non-blocking

Write directly into the ticket's own **`## Review`** section (no separate file):

- A **findings table**: severity (**blocking** / **non-blocking**), description, evidence,
  suggestion.
- **Blocking** — breaks the golden path, ships wrong behaviour, contradicts a locked decision,
  or is missing required docs coverage (4a.1). **Do not fix it inline.** The ticket moves to
  `tickets/5-rework/` (step 6a) for a scoped fix, then back to `tickets/4-in-review/` for a
  scoped re-review.
- **Non-blocking** — quality/consistency/polish that doesn't block shipping. Spawn a **new
  ticket** in `tickets/1-to-do/` (from this skill's `resources/TEMPLATE.md`, graded per the
  rules §3) for each, and reference the new id(s) here. Each spawned ticket records its
  lineage — `spawned-by: [<reviewed ticket id>]`, i.e.
  `pickle ticket new … --spawned-by "T-NNN"` — which is provenance only and does **not** make
  the follow-up wait on the ticket under review. The original ticket proceeds to
  `tickets/6-done/` (step 6b).
- Trivial, purely-cosmetic spec typos in the workflow scaffolding itself may be patched directly
  and noted here.

## 6. Move the ticket

**6a. If any blocking finding exists:** move `tickets/4-in-review/T-NNN-*.md` →
`tickets/5-rework/`; append a `## History` line. A fix pass (**"rework ticket T-NNN"** — see
the skill's procedure) then addresses *only the listed findings* on the same branch and moves
the ticket back to `tickets/4-in-review/` (History line on each move); a **scoped re-review**
verifies just those findings and concludes via 6a/6b again.

**6b. If there are no blocking findings** (zero, or only non-blocking already spawned as new
tickets): move `tickets/4-in-review/T-NNN-*.md` → `tickets/6-done/`; append a `## History` line
noting the verdict and any spawned ids.

## 7. Update the board and other references

- **`BOARD.md`** — move the ticket to its new section (under its child-project's sub-group),
  update that child's WIP counts, add any newly-spawned tickets to the TO DO section
  (impact-ordered, under their child's sub-group), and clear dependency blockers for tickets
  that were waiting on this one.
- Any ticket or doc that referenced this ticket by id.

## 8. Impact sweep — *did T-NNN change assumptions for later tickets?*

Re-read every ticket in `tickets/2-ready/` and `tickets/1-to-do/` that lists this ticket in
`depends-on:` (or references it in Description) and check whether the implementation invalidated
any assumption they encode. Patch the affected tickets, or open a new `tickets/1-to-do/` ticket
if the change is large.

## 9. Finish

- Summarize: what was verified, findings by severity, the ticket's new status, any newly-spawned
  tickets, and the remaining-tickets impact.
- **Child-project (build target):** present the full Conventional Commit message —
  `<type>(<scope>): <description>` (`<scope>` optional and never the ticket id; the ticket id
  is appended in brackets at the end of the subject line, e.g. `(T-42)`), plus body — **and the
  merge-request attributes** (title, description, target branch) **to the user for approval**.
  Local WIP commits on the ticket branch are fine at any time — the approval gate governs
  **publishing**. Only after the user approves all attributes: finalize the branch (squash to
  the single approved commit, or keep the history — the user chooses at approval time), push,
  and **create the merge request** in that child-project's repo. **Never push a child-project or
  open a merge request without approval; merging is always the human's.**
- **Overarching project** (tickets, board, bookkeeping): commit per the project's commit
  policy — may be automated, always with **explicit pathspecs** (`git add <paths>`, never
  `git add -A`/`.`) so deliberately-untracked material is never swept in.
- **Suggest the next ticket**: the next item in `BOARD.md`'s READY section (impact order,
  respecting `depends-on:` and per-child WIP limits), whether its prerequisite gate is
  satisfied — **explicitly stating whether the human must merge the just-reviewed branch first**
  (a dependent ticket may not start until the dependency is done *and merged* in its own child,
  rules §3) — and offer to pick it up.

---

### Checklist (paste into the ticket's `## Review` section)

- [ ] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [ ] Quality audit (step 3)
- [ ] Consistency audit (step 4)
- [ ] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a, if the project ships docs)
- [ ] Findings classified blocking/non-blocking and recorded in the ticket's `## Review`; non-blocking → new `tickets/1-to-do/` ticket(s) (step 5)
- [ ] Ticket moved to `tickets/6-done/` or `tickets/5-rework/`; `## History` appended (step 6)
- [ ] `BOARD.md` updated; other references if needed (step 7)
- [ ] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + child-project commit message & MR attributes presented for approval (commit/push/MR only after approval; merge stays human) + overarching-repo bookkeeping committed per policy + next-ticket suggestion (step 9)
