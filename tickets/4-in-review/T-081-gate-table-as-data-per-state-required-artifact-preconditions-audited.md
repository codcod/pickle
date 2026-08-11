---
id: T-081
title: gate table as data: per-state required-artifact preconditions, audited
project: pickle
depends-on: [T-080]
spawned-by: []
impact: medium
complexity: high
cost: L
---

# T-081 — gate table as data: per-state required-artifact preconditions, audited

## Outcome

After this ships, `pickle ticket move T-NNN ready` **refuses** a ticket whose Implementation Plan
is missing any of the seven READY-gate steps, and `pickle board audit` reports the same
deficiency as an error for a ticket already sitting in READY or later — instead of the gate
resting on an agent's prose judgement that nothing verifies. Which sections each state requires
is a table in one file (`internal/flow/brine.go`), so changing a gate is a data edit, and the
`## Outcome` warning T-083 hand-rolled becomes one row of that table.

## Description

brine has exactly one entry gate — the READY gate — and it is **prose**: rules §4 lists seven
things an Implementation Plan must contain, and an agent judges whether they hold before
running `pickle ticket move T-NNN ready`. Nothing mechanical checks it, so nothing catches a
plan that quietly lost its acceptance test.

This ticket adds a **gate table**: a declarative, per-state list of preconditions that
`board audit` (and `ticket move`) can evaluate deterministically — "entering state X requires
artifacts/sections of kinds {…}".

The model to copy is rick's, which has already been through this: its phase gates are a
`map[string][]phaseRequirement` in `sdlc-cli/internal/checks/phasegate.go`, whose own comment
reads *"the gate table — data, not code, so new phases edit this map"*. Each requirement is a
label, a set of acceptable kinds, and whether approval is required. `pre-implement` needs an
approved solution-design **and** an approved execution-plan; `pre-plan` needs an approved
analysis *or* task. That is precisely the shape brine's READY gate wants.

Two things this must decide at refinement (**both decided 2026-08-11** — see the Implementation
Plan's confirmed decisions 1–9; in short: the unit is a `##` section *and* a normalised `###`
sub-heading stem inside it, so the seven READY-gate items are individually checkable, and
severity is **per table row**, so the plan items refuse a move and error at audit while T-083's
`## Outcome` row stays a warning):

- **What counts as an "artifact" in brine.** Today everything lives in one ticket file, so
  the gate's unit is a `##` section (and its non-emptiness), not a separate file. Whether
  brine later grows per-phase artifact files — as rick has, under `docs/specs/<KEY>/` — is a
  bigger question this ticket should *enable* but not answer.
- **Error or warning.** A gate failure at `ticket move` should almost certainly refuse the
  move (brine's gates have teeth by design); a gate failure found by `board audit` on an
  already-moved ticket is a broken invariant and therefore an error. Neither should be a
  judgement call at runtime.

With T-073 (the `flow` key), T-080 (states and transitions as data) and this ticket, a second
flow becomes a definition file plus a prose addendum. That is the point of the sequence — but
this ticket is worth having on its own: it turns brine's most important quality gate from a
convention into an auditable invariant.

Soft coupling: T-064 (dropped — "no merit gate between filing and pickup") argued the
adjacent point that brine's gates test plan *completeness*, not worth; a mechanical
completeness gate here does not close that, and should not pretend to.

Soft coupling: **T-083** (shipped) already added one hand-rolled section precondition —
`board audit` warns when a non-terminal ticket's `## Outcome` is absent, empty, or a
placeholder (`internal/audit/audit.go`, `ticket.OutcomeMissing`). It is deliberately a
warning and lives outside any gate table; this ticket's table is where it should end up as
data, so fold that check in rather than leaving a second, parallel mechanism behind. The
coupling stays soft, as T-083's Decision 2 recorded (added by the T-083 review, finding N3).

Soft coupling: **T-042** owns the "one fact written in four places" epic. This ticket adds a
third and fourth caller to `internal/ticket`'s section-walk helpers rather than a fifth copy of
the walk, so it shrinks that epic's surface instead of growing it; no item moves between the two.

Re-graded at refinement (2026-08-11): complexity `medium` → `high` and cost `M` → `L`. The
confirmed scope spans four packages (`flow`, `ticket`, `move`, `audit`), the installed skill
payload and the manual, and it introduces a heading-normalisation contract that needs its own
drift guard against `TEMPLATE.md`. Impact stays `medium`: it makes one existing gate auditable
rather than adding a user-visible capability — the second-flow capability lands with whatever
first reads a project-authored spec.

## Implementation Plan

### 0. Feature branch (mandatory)

Target child-project: **`pickle`** (root path `.`). Before any change:

```
git checkout main
git pull
git checkout -b feat/T-081-gate-table
```

The branch slug is deliberately shortened from the filename slug (as `feat/T-080-flow-definition`
and `feat/T-090-harden-linkify-urls` were) — no board cell renders it.

WIP commits on the branch are encouraged. This is a **root-path child**, so before presenting the
summary, interactive-rebase the WIP commits into a small number of atomic, correctly typed/scoped
commits and default to **keeping that history** on merge rather than squashing (rules §0). Do not
push and do not open a merge request without explicit user approval; before pushing, verify
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
nothing. **Bookkeeping for this ticket (the ticket file, its History lines, `BOARD.md`) is
committed on `main`, never on this branch.**

### Prerequisite gate (hard)

1. **`depends-on: [T-080]` is satisfied** — T-080 is in `6-done/` and merged to `main`
   (`e214ee3`, recorded in its History). Verified 2026-08-11; no other prerequisite.
2. Working tree clean, branch cut from an up-to-date `main`.
3. Baseline recorded before the first edit: `just test`, `just lint`, `just docs-check` green, and
   `go run . board audit` reports **90 tickets, 0 error(s), 0 warning(s)** (measured 2026-08-11).
   The acceptance test compares against this baseline, so capture it rather than assuming it.

### Confirmed design decisions (do not deviate without asking)

1. **The gate table is per-state data on `flow.State`**, a new `Requires []Requirement` field —
   the same shape as the existing `Columns`/`WIPKey` per-state fields, not a new parallel
   mechanism. `internal/flow/brine.go` is the only place the table is edited, and it stays a leaf
   package (no `internal/config`, no `internal/ticket` import).
2. **Requirement shape** — `{Section, Sub, Label, Hint, Severity}`; a single `Section` string, no
   rick-style any-of set of acceptable kinds, and **no `approval` field** (brine has no approval
   concept; do not invent one). Both omissions go in the type's doc comment as deliberate, naming
   the any-of set as the extension point a per-phase-artifact flow would add. Nothing in this
   ticket models artifact *files*.
3. **Semantics: a requirement on state X is checked both when a ticket moves into X and while it
   sits in X.** Entry and residency deliberately share one table — for a non-terminal state they
   are the same predicate, and a second table would double the surface for no caller today. A
   flow wanting an entry-only check is out of scope, and the `Requirement` doc comment says so.
4. **Terminal states declare no requirements, and that is how T-083's archive exemption becomes
   data.** `internal/audit`'s hand-rolled `!st.Terminal` condition and `ticket.OutcomeMissing`
   both disappear — no alias, no second mechanism. `New` still *permits* requirements on a
   terminal state (a flow may legitimately demand something to enter DONE); brine leaves both
   empty, and `brine.go`'s comment records why (residency would otherwise warn forever on a
   permanent archive — rules §3, §7).
5. **Severity is per row, as data:** `flow.Blocking` and `flow.Advisory`. `ticket move` refuses a
   move whose target state has an unmet **Blocking** requirement; `board audit` reports an unmet
   Blocking requirement as an **error** and an unmet Advisory one as a **warning**. `move`
   deliberately does **not** re-report Advisory violations itself — its post-move audit
   self-check already prints them (verified: today's move prints the Outcome warning), and
   emitting both would double-report. The zero value of `Severity` is invalid and `New` rejects
   it, so a row cannot acquire teeth (or lose them) by omission.
6. **Brine's rows:**
   - `## Outcome` — **Advisory**, on all five non-terminal states. Its message must stay
     **byte-identical** to today's T-083 warning: `## Outcome is missing, empty, or still a
     placeholder — say what changes when this ships, in user-observable terms`.
   - `## Implementation Plan` present and substantive — **Blocking**, on `2-ready` … `5-rework`.
   - the **seven READY-gate items** as `###` stems inside `## Implementation Plan` — **Blocking**,
     on `2-ready` … `5-rework`: `feature branch`, `prerequisite`, `confirmed`, `tasks`,
     `acceptance test`, `docs`, `finish` (rules §4.1–§4.7 in order).
   - `## Review` present and substantive — **Blocking**, on `5-rework` only: a rework's scope
     *is* the recorded findings (rules §5). Not on `4-in-review` (the review writes it) and not
     on `6-done` (terminal, per decision 4).
   - `1-to-do` requires **only** `## Outcome`. A ticket must always be able to retreat to
     `1-to-do`, and `2-ready → 1-to-do` is the documented escape hatch from a gate that no longer
     holds (rules §2).
   - The four states `2-ready` … `5-rework` share **one** `readyGate` slice var in `brine.go`
     rather than repeating ~32 literal rows.
7. **Heading matching is a declared, deterministic normalisation — never a content judgement.**
   `normalizeHeading` (unexported, `internal/ticket`): lower-case, strip a leading ordinal
   (`^\d+[.)]?\s*`), strip a trailing parenthetical (`\s*\(.*\)$`), collapse internal runs of
   whitespace, trim surrounding whitespace and trailing punctuation. A requirement matches when
   the normalised heading **has `Sub` as a prefix**. This is what covers the corpus measured at
   refinement over the 45 done tickets — `0. Feature branch (mandatory)`, `Feature branch`,
   `2. Confirmed decisions`, `Docs`, `6. Finish`, `Acceptance test (run verbatim; must be green
   before review)` all match; `### 4. Tests` (one ticket, in `6-done/`, therefore exempt) does
   not. The gate proves a named heading exists with a non-empty body and **nothing about its
   content** — a heading over garbage passes. Say that explicitly in the docs: it is the same
   boundary T-064 named (completeness, not worth), and this ticket must not pretend to close it.
8. **"Substantive" is T-083's predicate, generalised, not a new one:** the section exists and its
   body is non-empty once HTML comments are stripped and the remainder trimmed. The `<…>`
   angle-bracket placeholder form `TEMPLATE.md` uses elsewhere is still invisible to it (T-083
   review finding B1) — so a skeleton pasted verbatim passes structurally. That is the documented
   boundary from decision 7, not a bug to fix here. The walk's **fence blindness is likewise
   inherited and unchanged**: a `### `-looking line at column 0 inside a fenced code block is
   already misread by `SectionHeadings`/`SectionBody` (T-083 documented it), and
   `SubsectionMissing` inherits exactly that, no worse. Verified for this very plan — every
   heading-looking line in its code fences is indented, so the seven headings the checker sees
   are the real ones.
9. **`Spec.Pickup` (the folded T-080 finding N6):** add the field, validated by `New` as a known
   **non-terminal** state, exposed as `Definition.Pickup() State`. `internal/move` and
   `internal/audit` use it for the dependency gate instead of
   `def.StateByWIPKey(config.WIPKeyInDevelopment)`, so the flow's most load-bearing gate stops
   being keyed off WIP limits and the fails-open `!ok` skip disappears (a validated `Definition`
   always has a `Pickup`). `internal/serve/view.go`'s two `StateByWIPKey` calls are genuinely
   WIP-badge lookups and **stay untouched**.
10. **Out of scope** (do not grow the diff): making `pickle flow show` print the table; reading a
    project-authored spec from disk; per-phase artifact *files*; any-of requirement sets;
    relaxing `config.Validate`'s `flow = "brine"`-only check; any grandfathering flag for the new
    audit error.

### Tasks

#### Task 1 — `internal/flow`: the requirement type, the table, and `Pickup`

`internal/flow/flow.go`:

- Add `type Severity int` with an invalid zero value:
  ```go
  const (
  	_        Severity = iota // zero value is invalid: New rejects an unset Severity
  	Blocking                 // unmet -> ticket move refuses; board audit errors
  	Advisory                 // unmet -> board audit warns; no move is refused
  )
  ```
  plus `String()` (`"blocking"`/`"advisory"`) for messages and test output.
- Add `type Requirement struct { Section, Sub, Label, Hint string; Severity Severity }` with the
  doc comment carrying decisions 2, 3 and 7 (single section, no any-of, no approval; entry +
  residency; `Sub` is a *normalised stem*, matched as a prefix by `internal/ticket`).
- Add `Requires []Requirement` to `State`, documented as authoring input read through
  `Definition.Requirements`.
- Add `Pickup string` to `Spec` (doc: the state a ticket is built in — the dependency+merge gate's
  own state, no longer inferred from a WIP key) and `pickup State` to `Definition`, with
  `func (d *Definition) Pickup() State`.
- Add `func (d *Definition) Requirements(dir string) []Requirement` returning a **clone** (the
  package's existing accessor contract). `New` stores a cloned slice per dir.
- Extend `New`'s validation, each with its own error string in the existing house style:
  - `Pickup` names a known state; that state is not `Terminal`.
  - per requirement: `Section` non-empty, `Label` non-empty, `Hint` non-empty (every message ends
    in a hint; an empty one renders a dangling em dash).
  - `Sub` must already be normalised as far as a leaf package can tell:
    `Sub == strings.ToLower(strings.TrimSpace(Sub))`.
  - no duplicate `(Section, Sub)` pair within one state's `Requires`.
  - `Severity` is `Blocking` or `Advisory` (rejects the zero value).

`internal/flow/brine.go`:

- Add `Pickup: "3-in-development"`.
- Add the three data blocks of decision 6 — `outcomeStated` (one `Requirement`), `readyGate`
  (eight rows: the plan section plus the seven `###` stems, in rules §4 order) and
  `reviewRecorded` — above the `Spec`, then wire each state's `Requires` from them
  (`slices.Concat`). Extend the file's header comment: this is now also the one place a gate is
  edited, and it records *why* the two terminal states declare none (decision 4).

#### Task 2 — `internal/ticket`: sub-section lookup, normalisation, and the evaluator

`internal/ticket/ticket.go`:

- Generalise T-083's predicate: `func SectionMissing(text, heading string) bool` (absent, or
  empty once HTML comments are stripped). **Delete `OutcomeMissing`** — no alias, no wrapper.
- Add `normalizeHeading(string) string` (unexported; decision 7) and
  `func SubsectionMissing(text, section, stem string) bool`: locate the `## section` span with the
  existing walk, scan its `### ` headings, normalise each, take the first whose normalised form
  has `stem` as a prefix, and report whether its body (to the next `###`/`##`, HTML comments
  stripped) is empty. An absent parent section, or no matching sub-heading, is also "missing".
  Reuse `SectionBody`'s line-prefix walk rather than adding a fifth copy of it (T-042).
- Add the evaluator both gate sites share:
  ```go
  type GateViolation struct{ Req flow.Requirement }
  func (v GateViolation) Blocking() bool
  func (v GateViolation) Message() string
  func GateViolations(reqs []flow.Requirement, text string) []GateViolation
  ```
  `GateViolations` preserves table order. `Message()` has exactly two forms, chosen by whether
  `Req.Sub` is empty:
  - `## <Section> is missing, empty, or still a placeholder — <Hint>`
  - `## <Section> has no substantive "### <Sub>" heading (<Label>) — <Hint>`

  The first form with brine's Outcome row must reproduce today's T-083 warning byte-for-byte
  (decision 6); pin that with a test, not by eye.

#### Task 3 — `internal/move`: refuse a move into a state whose blocking requirements are unmet

`internal/move/move.go`:

- After the reason check and **before** the WIP gate (cheapest-to-explain ordering: an unmet gate
  is about this ticket's own content, a WIP breach about the child's queue — and the existing
  order already runs per-ticket checks first), evaluate
  `ticket.GateViolations(def.Requirements(target.Dir), t.Text)`, keep the `Blocking()` ones, and
  on any refuse before writing anything:
  ```
  cannot move T-081 to READY: 2 unmet gate requirement(s): <msg>; <msg>
  ```
- Replace the `StateByWIPKey(config.WIPKeyInDevelopment)`/`hasPickup` pickup lookup with
  `def.Pickup()` (decision 9), deleting the fails-open branch and updating the comment above it.
  Check whether `internal/config` is still used by the file; drop the import if not.

#### Task 4 — `internal/audit`: the table-driven check replaces the hand-rolled Outcome branch

`internal/audit/audit.go`:

- In the per-ticket loop, replace the `!st.Terminal && ticket.OutcomeMissing(...)` block with a
  walk over `ticket.GateViolations(def.Requirements(t.Dir), t.Text)`: `Blocking()` → `r.errf`,
  otherwise → `r.warnf`, each prefixed with the existing `ref`. The comment keeps T-083's
  reasoning but restates it as decision 4 (the exemption is now the table's, not this file's).
- Append to a blocking violation's message the two ways out, so the error is actionable and the
  migration path is in the tool rather than only in the CHANGELOG:
  `— write it, or move the ticket back to 1-to-do until the plan is complete`.
- Replace the `StateByWIPKey(config.WIPKeyInDevelopment)` dependency-gate guard with
  `def.Pickup()` (decision 9); the `if ok` wrapper goes away.

#### Task 5 — tests

- `internal/flow/flow_test.go`: extend `TestSpecValidationRejects` with one case per new
  validation (unknown/terminal `Pickup`, empty `Section`/`Label`/`Hint`, un-normalised `Sub`,
  duplicate `(Section, Sub)`, unset `Severity`); assert `Requirements` returns a copy in
  `TestAccessorsReturnCopies`; assert brine's `Pickup()` is `3-in-development` and that every
  terminal state has zero requirements in `TestBrineDefinitionValidates`.
- `internal/ticket/ticket_test.go`: table tests for `normalizeHeading` (every heading form
  measured in decision 7, including the `### 4. Tests` non-match), `SubsectionMissing`
  (absent parent, absent stem, heading present with empty body, heading present with prose,
  a `####` heading not mistaken for a `###` one), and `SectionMissing` — port
  `TestOutcomeMissing`'s six cases verbatim onto it. Retarget
  `TestTemplateAndScaffoldOutcomePlaceholdersAreFlagged` to `SectionMissing(text, "Outcome")`.
  Add `TestGateViolationMessages` pinning both message forms, including the byte-identical
  Outcome text.
- `internal/ticket/ticket_test.go` — **the drift guard** (decision 7's contract with the
  authoring guide, mirroring `TestFrontmatterKeysMatchTemplate`):
  `TestBrineReadyGateMatchesTemplate` reads `../../skill/resources/TEMPLATE.md`, collects the
  `### ` headings inside its `## Implementation Plan`, and asserts (a) every `Sub` stem in
  brine's requirements matches one of them under `normalizeHeading`, and (b) each stem is
  already normalised (`normalizeHeading(Sub) == Sub`). Heading **vocabulary** only — not body
  substance, which the `<…>` placeholders make meaningless here (decision 8). This test lives in
  `internal/ticket` because only it sees both `flow` and `normalizeHeading`.
- `internal/move/move_test.go`: a ticket scaffolded by `newTicket` must now be **refused** at
  `→ ready` (this is the defect the ticket exists to close: measured at refinement, today it
  moves cleanly); the same ticket with a minimal seven-heading plan moves; `5-rework` refuses
  without a `## Review` body; `1-to-do` never refuses. `TestForwardWalkStaysClean` and the other
  existing walks need their fixtures given a plan — extend the `newTicketFull` helper with the
  gate-satisfying body rather than pasting a skeleton into each test.
  `TestSpawnedByDoesNotGatePickup` must keep passing untouched in substance.
- `internal/audit/audit_test.go`: `writeGood` gains a gate-satisfying body; new cases for a
  blocking violation (error, message names the heading and both ways out) and the Advisory
  Outcome violation (still a warning, still byte-identical). Add a case pinning decision 4: a
  ticket in `6-done/` with neither Outcome nor plan produces **no** finding.

#### Task 6 — skill payload (what `pickle install` ships)

- `skill/resources/tickets-README.md` §4: state that the seven items are now **mechanically
  checked as `###` headings inside `## Implementation Plan`**, that `pickle ticket move … ready`
  refuses when one is missing and `board audit` errors for a ticket already past the gate, and
  that the check is structural — it proves the step is *present*, never that it is *good*
  (decision 7). §7: rewrite the T-083 sentence as one row of the per-state gate table, keeping
  "a warning only, never a gate" true for `## Outcome`. §2: note that `2-ready → 1-to-do` is the
  escape hatch when the gate refuses.
- `skill/resources/TEMPLATE.md`: in the Implementation Plan skeleton, change *"Delete if none"*
  to keep the heading and write `none` / `no user-facing surface` in the body, for both
  `### Prerequisite gate (hard)` and `### Docs update (mandatory when user-facing)` — rules §4
  already says "stated (or 'none')", and a deleted heading now fails the gate. Change nothing
  else about the seven headings: `TestBrineReadyGateMatchesTemplate` pins them.
- `skill/SKILL.md`: the READY-gate bullet (mechanically checked, and what that does *not* mean),
  *refine a ticket* step 7 (the move now refuses — expect it, don't work around it), and the
  *audit the board* paragraph's list of checks.
- `internal/install/install.go`'s `markerBlock()`: no change needed (it names the gate but not
  its enforcement) — confirm by reading it, and if it does need one, mirror it **by hand** into
  this repo's `AGENTS.md` inside this ticket's diff, per the self-modify policy.

#### Task 7 — manual + CHANGELOG

- `docs/user-manual/cli-reference.adoc`: in `<<cmd-board-audit>>`, replace the `## Outcome`
  bullet with a gate-table bullet covering both severities and naming the two ways out; in
  `<<cmd-ticket-move>>`, add the gate to the list of gates "enforced before anything is
  written"; extend the upgrade `NOTE` with the one-time migration (a ticket already in
  `2-ready/`…`5-rework/` whose plan predates the table reports errors until a heading is added or
  the ticket is moved back), and say that `board sync` will usually *not* surface it — it only
  re-runs its audit self-check when it rewrites `BOARD.md`, and a plan-only violation leaves the
  board unchanged (verified during implementation; corrected from this task's original text).
- `docs/user-manual/concepts/lifecycle.adoc`: mark which of the seven READY-gate items are
  mechanically checked and how (heading presence + non-empty body), and that `ticket move`
  refuses.
- `docs/user-manual/concepts/agent-session-workflow.adoc:56` — *"The READY gate is a judgement
  call"* is made **partly false** by this ticket; correct it to name what is now mechanical and
  what remains judgement.
- `CHANGELOG.md` under `## [Unreleased]`: an **Added** entry for the gate table (with the
  `Spec.Pickup` fix folded in, crediting T-080's finding N6) and a **Changed** entry for the
  Outcome check moving into the table with identical wording, including the migration note.

### Acceptance test

Run from the repo root, on the feature branch. Every command is re-runnable verbatim.

1. **The child's configured commands:**
   ```
   just build
   just test
   just lint
   just docs-check
   ```
   All four green.

2. **The new units, named:**
   ```
   go test ./internal/flow/... ./internal/ticket/... ./internal/move/... ./internal/audit/... -run \
     'Requirement|Gate|Pickup|Normalize|Subsection|SectionMissing|Template|Outcome|Validation' -v
   ```
   Includes `TestBrineReadyGateMatchesTemplate` passing (the gate table and `TEMPLATE.md` agree).

3. **This repo stays clean — the fold changed no verdict in situ:**
   ```
   go run . board audit
   ```
   Expect **`90 tickets, 0 error(s), 0 warning(s)`** (the prerequisite-gate baseline; the count
   rises by one per ticket filed while this is in flight). This is the proof that folding T-083's
   check into the table is behaviour-preserving and that the 23 `1-to-do/` tickets, which have no
   plans, are not gated.

4. **End-to-end in a throwaway install** (never against this repo — self-modify policy):
   ```
   just build
   D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && ./pk install --project demo --agent claude
   ./pk ticket new "demo gate" --project demo
   ./pk ticket move T-001 ready --reason "plan complete"
   ```
   **Expected: refused, non-zero exit, listing the eight unmet blocking requirements, and
   `tickets/1-to-do/T-001-demo-gate.md` still in place.** (Measured on `main` at refinement,
   2026-08-11: this move **succeeds** today, printing only the Outcome warning — that is the
   defect this ticket closes.)

   Now give the plan the seven headings (this surgery was itself run verbatim at refinement, so
   the `assert` is a real check that `ticket.Scaffold`'s placeholder has not drifted):
   ```
   python3 - <<'EOF'
   import pathlib
   p = pathlib.Path("tickets/1-to-do/T-001-demo-gate.md")
   plan = """### 0. Feature branch (mandatory)

   feat/T-001-demo

   ### Prerequisite gate (hard)

   none

   ### Confirmed design decisions

   d1

   ### Tasks

   t1

   ### Acceptance test

   just test

   ### Docs update

   no user-facing surface

   ### Finish (mandatory)

   summary + suggested commit message
   """
   t = p.read_text()
   ph = "<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->"
   assert ph in t
   p.write_text(t.replace(ph, plan))
   EOF
   ./pk ticket move T-001 ready --reason "plan complete"
   ```
   (The heredoc body must be flush-left when pasted — it is indented here only by this
   ticket's list formatting.) **Expected: moves, exit 0, and still prints the `## Outcome`
   warning** — Advisory, byte-identical to today's. Then remove one required step from the
   now-READY ticket:
   ```
   python3 - <<'EOF'
   import pathlib, re
   p = pathlib.Path("tickets/2-ready/T-001-demo-gate.md")
   p.write_text(re.sub(r"### Acceptance test\n\njust test\n\n", "", p.read_text()))
   EOF
   ./pk board audit
   ./pk board sync
   ```
   **Expected: `board audit` exits non-zero with exactly one error naming `"### acceptance
   test"` and both ways out.** **`board sync` reports "already in sync" and exits 0 — it does
   *not* surface the error**, corrected during implementation from this step's original text
   ("`board sync` refuses while that error stands"): `sync` only re-runs its own audit
   self-check when it actually rewrites `BOARD.md`, and nothing in a ticket's plan body feeds
   any board column, so a plan-only violation leaves the board unchanged and `sync` never
   reaches that check. `board audit` is the command that surfaces this class of error; run it
   directly rather than relying on `sync`. Confirm the escape hatch works:
   `./pk ticket move T-001 to-do --reason "gate no longer holds"` succeeds (`1-to-do` requires
   only `## Outcome`) and `./pk board audit` is back to 0 errors (1 warning). Finally
   `cd - && rm -rf "$D"`.

### Docs update (mandatory when user-facing)

User-facing on three surfaces; all of it is Tasks 6 and 7: the installed skill payload
(`skill/resources/tickets-README.md` §2/§4/§7, `skill/resources/TEMPLATE.md`, `skill/SKILL.md`),
the manual (`docs/user-manual/cli-reference.adoc`, `concepts/lifecycle.adoc`,
`concepts/agent-session-workflow.adoc`), and `CHANGELOG.md`. `just docs-check` must stay green.
No new manual page is registered.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (Tasks 6 and 7); `CHANGELOG.md` has its `Unreleased` entries.
3. Write a summary: files touched, the gate table's final rows, anything deferred (expect: the
   any-of requirement set, `pickle flow show` printing the table, artifact *files*).
4. Suggested commit message:
   ```
   feat(flow): make each state's required sections a gate table (T-081)

   Per-state Requires rows on flow.State, checked when a ticket moves into a
   state and while it sits there: ticket move refuses an unmet blocking
   requirement, board audit errors on it. Folds T-083's hand-rolled ## Outcome
   warning in as one advisory row, and gives Spec an explicit Pickup state
   instead of inferring the pickup gate from a WIP key (T-080 finding N6).
   ```
5. **Tidy up before presenting** (root-path child): interactive-rebase the WIP commits into
   atomic, correctly typed/scoped commits — expect roughly `feat(flow)`, `feat(ticket)`,
   `feat(move)`, `feat(audit)`, `test`, `docs(skill)`, `docs`. Default to keeping that history
   on merge rather than squashing (rules §0).
6. Commit locally on the branch; do **not** push or open an MR without user approval. Present the
   commit messages; after approval verify `git fetch origin main && git diff --name-only
   origin/main...HEAD | grep '^tickets/'` prints nothing, then push and open the MR — merging is
   the human's. Hand back.

## Review

Reviewed 2026-08-11 on `feat/T-081-gate-table` (5 commits, `d744867`..`05cf03b`), ticket read from
`main`. Verdict: **3 blocking findings → `5-rework/`**. The code is sound and the acceptance test
passes verbatim; all three findings are in what the tool and the docs *tell a user to do* about a
gate failure.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — run on
  `lifecycle.adoc` + `agent-session-workflow.adoc`; one suggestion touches text this branch
  authored, noted under the table (readability suggestions are not findings)
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep (step 8) — one candidate patch found (T-085), deferred to
  the concluding re-review since a rework could still move the gate's unit
- [ ] Summary + commit message & MR attributes presented for approval (step 9) — deferred: a
  blocking verdict does not publish

### Acceptance test — re-run verbatim

| step | result | evidence |
|---|---|---|
| 1 — `just build` / `just test` / `just lint` / `just docs-check` | met | all four green, exit 0 |
| 2 — the named unit tests | met | `ok` for `internal/flow`, `internal/ticket`, `internal/move`, `internal/audit`; `TestBrineReadyGateMatchesTemplate` passes |
| 3 — `go run . board audit` on this repo | met | `board audit: 90 tickets, 0 error(s), 0 warning(s)` — the fold is behaviour-preserving in situ |
| 4 — throwaway install, end to end | met | scaffolded ticket refused at `→ ready`, exit 1, **8** unmet requirements listed, file still in `1-to-do/`; with the seven headings it moves, exit 0, still printing the byte-identical `## Outcome` warning; stripping `### Acceptance test` gives exactly one `board audit` error naming `"### acceptance test"` + both ways out, exit 1; `board sync` reports `already in sync`, exit 0; the `→ to-do` escape hatch succeeds and returns to 0 errors / 1 warning |

Confirmed decisions 1–10 all honoured, spot-checked individually: the table is per-state data on
`flow.State` in `brine.go` only (1); `Requirement` has no any-of set and no approval field, both
recorded as deliberate in the doc comment (2, 3); terminal states declare nothing and
`ticket.OutcomeMissing` / `audit.go`'s `!st.Terminal` are both gone (4); severity is per row and
`move` does not re-report Advisory rows (5); brine's rows match decision 6 exactly, including the
single shared `readyGate` slice; `normalizeHeading` implements decision 7's five steps verbatim
and `internal/serve/view.go`'s two `StateByWIPKey` calls are untouched (9); nothing from the
out-of-scope list (10) crept in. `markerBlock()` confirmed to need no change — it names the READY
gate but not its enforcement — so `AGENTS.md` is correctly untouched.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| B1 | blocking | — | `internal/audit` appends the remedy `— write it, or move the ticket back to 1-to-do until the plan is complete` to **every** blocking violation, but `1-to-do` is reachable **only from `2-ready`** (brine's transition table). For a ticket in `3-in-development`, `4-in-review` or `5-rework` the tool prints advice `ticket move` will refuse. It is also semantically wrong for the `## Review` row, which is Blocking on `5-rework` and has nothing to do with a plan. | Throwaway install: a `4-in-review` ticket with `### Acceptance test` stripped gives `ERROR: … — write it, or move the ticket back to 1-to-do until the plan is complete`; following it gives `pickle: illegal transition IN REVIEW → TO DO (from IN REVIEW, legal: DONE, DROPPED, REWORK)`, exit 1. `internal/audit/audit.go` (the `r.errf` in the gate-table loop); `internal/flow/brine.go` `Transitions`. | Make the remedy state-aware, or drop the `1-to-do` clause for a state-agnostic one ("write it — or, from READY, move the ticket back to `1-to-do/`"). Cheapest correct version: derive it from `def.Allowed(t.Dir)`. Add a test asserting every remedy the audit prints names a legal target from the ticket's own state — nothing today would have caught this. |
| B2 | blocking | — | `docs/user-manual/concepts/lifecycle.adoc` states that `board audit` errors *"on one moved back to `1-to-do/`, since the gate no longer holds there either"*. That is false, and it tells the reader the escape hatch is closed. `1-to-do` requires only `## Outcome` (Advisory) — decision 6's whole point. | Verified live: after `ticket move T-001 to-do`, `board audit` reports `1 tickets, 0 error(s), 1 warning(s)`, exit 0. Contradicted by `cli-reference.adoc:609` ("its only requirement is `## Outcome`"), `tickets-README.md` §2 ("the one state that asks nothing of the Implementation Plan") and by the audit's own remedy text. | Delete the clause; say instead that `1-to-do/` is the one state the plan rows do not apply to, which is what makes it the escape hatch. |
| B3 | blocking | — | The migration's blast radius is undocumented (4a.1: a user-facing behaviour change with no coverage). One legacy ticket in `2-ready/`…`5-rework/` now makes **`pickle upgrade` fail**, **`pickle install` fail**, and **every unrelated `pickle ticket move` fail after applying** — the manual's NOTE documents only `board audit` and `board sync`, and never says the command that *delivers* this change aborts. The one place in-tree that recorded this hazard — T-083's comment in `audit.go` ("an error here would make ticket move, board sync, install and upgrade all fail for every ticket that predates the section") — was deleted by this branch, so the fact now lives nowhere. | Throwaway install with one legacy prose plan in `2-ready/`: `pickle: post-upgrade audit found 7 error(s)` exit 1; `pickle: post-install audit found 7 error(s)`; and an *unrelated* `ticket move T-002 dropped` → `move applied but board audit now reports 7 error(s)`, exit 1, with the file already moved. `internal/cli/install.go:91,180`; `internal/move/move.go:144`. | Extend the `cli-reference.adoc` upgrade NOTE (and the CHANGELOG upgrade note) to name `upgrade`/`install` failing and unrelated moves reporting after applying, with the fix-first ordering that follows: clean the legacy plans *before* upgrading, or immediately after. The behaviour itself is decision 10 and stays — only the warning is missing. |
| N1 | non-blocking | note-and-close | Violation messages print the normalised **stem** as though it were the heading: `has no substantive "### prerequisite" heading`, `"### docs"`, `"### confirmed"` — where `TEMPLATE.md` ships `### Prerequisite gate (hard)`, `### Docs update (mandatory when user-facing)`, `### Confirmed design decisions`. A user copying the message writes a heading that passes the gate but drifts from the vocabulary `TestBrineReadyGateMatchesTemplate` exists to hold steady. | Throwaway install, refused `→ ready`: all eight messages quote stems, not template headings. `internal/ticket/ticket.go` `GateViolation.Message`. | Render `Req.Label` in the quoted position and keep `Sub` purely for matching — `Label` is already required non-empty and is otherwise near-duplicate data (see N3). Closed, not scheduled: the message is correct, only less copy-pasteable than it could be. |
| N2 | non-blocking | folded (T-042) | `SubsectionMissing` hand-rolls a `### ` line walk and re-implements `SectionMissing`'s strip-comments-and-trim tail predicate inline, while its doc comment claims it avoids "adding a fifth copy" of the walk. It reuses `SectionBody` for the *parent* only; the sub-level walk and the emptiness test are both new copies. | `internal/ticket/ticket.go` `SubsectionMissing` vs `SectionMissing`. | T-042 ("collapse duplicated internal predicates into single helpers") already owns this ground; the section-walk item there now covers a third level. No re-grade — the ticket's own Description already predicted this shape. |
| N3 | non-blocking | note-and-close | `Requirement.Label` is validated non-empty for every row but is only ever rendered when `Sub != ""`. On the three section-only rows (`Outcome`, `Implementation Plan`, `Review`) it is dead data that merely repeats `Section`. | `internal/flow/flow.go` `validateRequirements`; `internal/ticket/ticket.go` `GateViolation.Message`. | Either render it in the section-only form too (see N1, which wants exactly that) or relax the validation to `Sub != "" ⇒ Label != ""`. Not worth scheduling on its own. |
| N4 | non-blocking | note-and-close | The new upgrade-NOTE paragraph opens *"have the same upgrade-visible shape a third time"* — it is the fourth shape in that NOTE (`spawned-by:`, the 120-rune cell cap, the status directories, then this). | `docs/user-manual/cli-reference.adoc`, the gate-table paragraph of the `cmd-board-audit` NOTE. | One-word fix. The rework pass is rewriting this very paragraph for B3 — correct the count while it is open. |

**Disposition summary:** 7 findings — 3 blocking (B1, B2, B3 → `5-rework/`, no disposition), 4
non-blocking: 3 note-and-close (N1, N3, N4), 1 folded into T-042 (N2), 0 new tickets, 0 fixed
inline. No follow-up ticket passed §5's promotion test.

**Readability pass (step 4b, not findings).** One of the reviewer's suggestions lands on text this
branch authored: split *"All seven are mechanically checked: each is a required `### ` heading
inside `## Implementation Plan`, present with a non-empty body"* into two sentences
(`lifecycle.adoc`). Worth taking while that paragraph is open for B2. Every other suggestion was
against pre-existing prose this ticket did not touch, and is out of scope for its diff.

**Impact sweep (step 8).** No `1-to-do/`/`2-ready/` ticket declares `depends-on: [T-081]`. One
references it: **T-085**, whose coupling note records T-081's unit as *"a `##` section (and its
non-emptiness)"*. As shipped the unit is a `##` section **plus** a normalised `###` stem inside
it — T-085's assumption is strengthened, not invalidated, but its per-*line* presence check (a
`Disposition summary` line) is still unmodelled. The patch is deferred to the concluding
re-review, since a rework could still move the unit. Per T-085's own standing instruction, no
re-grade.

### Rework scope

B1, B2, B3 only, on `feat/T-081-gate-table`. Two of the three are prose; B1 is a message change
plus its regression test. N4 and the readability note may be folded in, since they land in
paragraphs B2/B3 already reopen. Nothing else.

### Fixed (rework, 2026-08-11)

- **B1** — `internal/audit.gateRemedy` (new) derives the "second way out" from `def.Allowed(dir)`,
  naming a move only when a legal, non-terminal target exists whose own `Requires` carries no
  Blocking row; otherwise it falls back to a generic "write it to satisfy the gate" remedy.
  Verified live: a `4-in-review` ticket missing `### Acceptance test` now gets the generic form,
  not the old "move it to 1-to-do" advice `ticket move` would refuse; a `2-ready` ticket in the
  same shape still correctly gets "move the ticket back to 1-to-do", since that move *is* legal
  from READY. New `TestGateRemedyOnlyNamesLegalNonBlockingTargets` (a property test over every
  state in `flow.Default()`, so it keeps holding if brine's transitions change shape later) plus
  a `TestAudit` table case pinning the in-review regression directly. Commit `053a68b`.
- **B2** — `lifecycle.adoc`'s false claim ("errors … on one moved back to `1-to-do/`, since the
  gate no longer holds there either") is corrected: `1-to-do/` requires only the advisory
  `## Outcome` row, which is what makes it the escape hatch, and the escape hatch is now stated
  as reachable specifically from `2-ready/` (matching B1's fix). Commit `2976701`.
- **B3** — the `cli-reference.adoc` upgrade NOTE and the matching `CHANGELOG.md` entry now say
  explicitly that this migration, unlike the other three upgrade-visible shapes in the same NOTE,
  can make **`pickle upgrade` and `pickle install` themselves fail** (both run the same
  post-operation audit self-check), and that every subsequent `ticket move` — even of an
  unaffected ticket — reports a post-move audit error too until the offending ticket is fixed.
  Both now tell the reader to run `board audit` before upgrading. Behaviour unchanged (decision
  10: no grandfathering) — verified live with the same throwaway-install reproduction the finding
  used, confirming the documented failure modes and the corrected remedy text together. Commit
  `2976701`.
- **N4** and the **readability suggestion** folded in alongside B2/B3 in the same commit, per the
  rework scope note above: the NOTE's "a third time" → "a fourth time", and the reopened
  `lifecycle.adoc` paragraph's dense sentence split in two.

Acceptance test re-run verbatim end to end (throwaway install): refusal at `→ ready` unchanged;
the positive path unchanged; stripping `### Acceptance test` from a `2-ready` ticket still gives
the `1-to-do` remedy, from `4-in-review` now gives the generic remedy (B1, confirmed live); an
unrelated `ticket move … dropped` still reports the pre-existing violation after applying (B3's
documented, unchanged behaviour). `just build`/`test`/`lint`/`docs-check` and
`go run . board audit` (`90 tickets, 0 error(s), 0 warning(s)`) all green.

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — patched by T-073's review impact sweep (step 8): T-073 shipped the `flow` key this
  ticket's "a second flow" argument depends on (`config.DefaultFlowName`, `Config.FlowName()`,
  `pickle flow show|list`). Note that `Validate()` currently accepts **only** `"brine"`, so the
  second-flow story here and in T-080 starts by relaxing that check. Nothing else changed
- 2026-08-10 — patched by T-080's review impact sweep (step 8): **the dependency this ticket
  declares has shipped**, so its starting point is now concrete rather than hypothetical.
  `internal/flow` exists as a leaf package holding a `Spec` (plain declarative data) validated by
  `New()` into an immutable `Definition`; per-state data already has a precedent to copy in
  `State.Columns`/`flow.ColumnProfile`, so the gate table is naturally a per-state field on that
  same struct rather than a new parallel mechanism. Two consequences for this ticket's own
  decisions: (1) T-083's `## Outcome` check — the hand-rolled section precondition this ticket's
  Description says to fold in — **is already definition-scoped**: `internal/audit` now tests
  `!state.Terminal` read from the definition instead of naming `6-done`/`7-dropped`, so folding it
  into a gate table is a smaller step than when this ticket was filed; (2) **T-080 review finding
  N6 is folded into this ticket**: `internal/move` and `internal/audit` both identify the pickup
  state as `def.StateByWIPKey(config.WIPKeyInDevelopment)` — the flow's most load-bearing gate
  keyed off an unrelated concern (WIP limits), which silently skips the whole gate when the
  lookup misses. `Spec` has explicit `Initial` and `DependencySatisfied` fields but no `Pickup`;
  since this ticket edits `Spec` anyway, give it one (or make the miss an audit error). Note also
  that `Validate()` still accepts only `"brine"` — T-080 deliberately did not relax it. No
  assumption invalidated; nothing re-graded
- 2026-08-11 — refined. Five open decisions put to the user and confirmed: the gate's unit is a
  `##` section **and** a normalised `###` stem inside it (so the seven READY-gate items are
  individually checkable — `TEMPLATE.md` already prescribes exactly those seven headings);
  single-section `Requirement`, no rick-style any-of set and no `approval` field; **severity per
  table row** (`Blocking` refuses the move and errors at audit, `Advisory` warns), which is what
  lets T-083's `## Outcome` row keep its warning-only promise while the plan rows get teeth;
  terminal states declare no requirements, so T-083's archive exemption becomes data and
  `audit.go`'s `!st.Terminal` test plus `ticket.OutcomeMissing` are deleted; T-080's finding N6 is
  in scope as `Spec.Pickup`. The heading normalisation was fixed by measuring the 45 done tickets'
  plans, and the throwaway-install acceptance steps were run verbatim on `main` to record the
  "before" (the move to READY succeeds today with an empty plan). Re-graded complexity `medium` →
  `high`, cost `M` → `L` (reason in the Description). Nothing split: no part is independently
  schedulable — the table, its evaluator and its two call sites ship or fail together
- 2026-08-11 — TO DO → READY: plan complete
- 2026-08-11 — READY → IN DEVELOPMENT: picked up
- 2026-08-11 — amended during implementation (fixed inline): the acceptance test's step 4 and
  Task 7's doc bullet both asserted "`board sync` refuses while [a gate-table] error stands" —
  run verbatim against a throwaway install, this is false. `internal/sync.Sync` only re-runs its
  post-write audit self-check when it actually rewrites `BOARD.md` (`res.Changed`); a violation
  confined to a ticket's `## Implementation Plan` body feeds no board column, so the board stays
  unchanged and sync returns `nil` without ever reaching that check — confirmed live: `board
  sync` printed "already in sync", exit 0, while `board audit` still reported the error. Both
  spots corrected to say `board audit` is the command that surfaces this class of error; no code
  changed, no behaviour changed — only the plan's and the shipped docs' claim about existing
  `internal/sync` behaviour. cli-reference.adoc and CHANGELOG.md carried the same wrong claim and
  were corrected alongside.
- 2026-08-11 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-11 — IN REVIEW → REWORK: 3 blocking findings: audit remedy names an illegal transition (B1), lifecycle.adoc contradicts the 1-to-do escape hatch (B2), migration blast radius undocumented (B3)
- 2026-08-11 — REWORK → IN REVIEW: findings fixed: B1 (audit remedy), B2 (lifecycle.adoc escape hatch), B3 (upgrade/install blast radius documented)
