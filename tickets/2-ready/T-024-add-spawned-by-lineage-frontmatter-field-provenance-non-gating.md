---
id: T-024
title: add spawned-by: lineage frontmatter field (provenance, non-gating)
project: pickle
depends-on: []
impact: medium
complexity: low
cost: S-M
---

# T-024 — add spawned-by: lineage frontmatter field (provenance, non-gating)

## Description

Add a new list frontmatter field, **`spawned-by:`**, that captures a ticket's pure **lineage**
— the ticket(s) it was born from — with **no gating semantics**. It sits parallel in shape to
`depends-on:`, but is deliberately the opposite in behaviour: `depends-on:` is a hard, blocking
dependency that gates pickup, while `spawned-by:` is provenance only and gates nothing.

```yaml
---
id: T-142
title: <short title>
depends-on: []              # hard, blocking dependencies (gates pickup)
spawned-by: [T-120]         # lineage only — the ticket(s) this one was born from; [] if none
impact: medium
complexity: low
cost: S
---
```

### Why this is needed

Provenance is currently captured in **one place only**: the first `## History` line —
`created (TO DO). source: <chat | review of T-xxx | audit | idea>`. That is prose,
newest-buried-as-history-grows, and not machine-queryable. The one structured relational field,
`depends-on:`, is deliberately reserved for **hard, blocking** dependencies — it drives the
`3-in-development/` transition guard in both `tickets-README.md` §3 and the board-audit engine.
So the "created because of another ticket" relationship has **no first-class home**, and it must
**not** be folded into `depends-on:` — a review-spawned follow-up is lineage, not a blocker, and
overloading `depends-on:` would wrongly gate its pickup.

### Why this shape

- **Name reuses existing vocabulary.** The skill already describes review findings as
  "**spawned** as new TO DO ticket(s)" and "new tickets **spawned** from non-blocking
  findings", so `spawned-by:` reads naturally. (Alternatives `derived-from:`, `origin:`,
  `parent:` are all fine, but `spawned-by` matches the existing prose.)
- **List of `T-NNN`, `[]` default** — identical format to `depends-on:`, so authors and the
  audit engine's existing `ParseDepends` logic apply unchanged. Usually one id; a list lets an
  audit that consolidates two overlapping findings cite both parents.
- **Explicitly non-gating.** Documented as lineage/provenance with **no transition guard** — a
  `spawned-by` parent need not be DONE (or even non-terminal) for the child to be picked up.
  This is the one line that must be crystal clear, precisely because `depends-on:` sitting right
  above it looks similar but behaves oppositely.
- **Set at creation, immutable** — like the History `source:` line.

### Scope (surfaces to touch — resolved at refinement; see the Implementation Plan)

- **Docs / skill payload** (the embedded skill, shipped into every installed project):
  `skill/resources/TEMPLATE.md`, `skill/resources/tickets-README.md` §3/§7, `skill/SKILL.md`,
  and `skill/resources/review-protocol.md`. The `source:` History line stays — `spawned-by:`
  complements it (structured + queryable), it does not replace it.
- **Ticket model / parsing** (`internal/ticket/ticket.go`): parse `spawned-by:` (reuse
  `ParseDepends`), expose `SpawnedBy` on the `Ticket` struct, and emit it in the scaffold.
  `pickle ticket new` **gains a `--spawned-by` flag**.
- **Audit engine** (`internal/audit/audit.go`): `spawned-by` **is a required key** (with `[]`
  default) like `depends-on`; audit validates its target ids **exist** and that a ticket does
  not list **itself** — **but adds NO** transition guard or done/merged check (the defining
  difference from `depends-on:`).
- **Board**: **no** `spawned-by` column — lineage is provenance, not a scheduling axis.
- **Backfill + root `README.md`**: every existing ticket gets `spawned-by: []`; the README's
  frontmatter-invariant list is updated. Populating real historical parents is out of scope
  (an optional follow-up).

### Couplings

Soft coupling to the review/audit procedures (SKILL.md *make it a ticket* / *validate a
ticket*), which are where spawned tickets are born and would set this field. No hard
`depends-on:`.

## Implementation Plan

### 0. Feature branch (mandatory)

All work targets the **`pickle`** child (the repo root). Before any change:

```
cd .                         # pickle child = repo root
git checkout main
git checkout -b feat/T-024-spawned-by-lineage-frontmatter
```

Local WIP commits are encouraged; **do not push or open an MR without explicit user
approval** (publish-gated child). Finish with a summary + suggested commit message.

### Prerequisite gate (hard)

None. This ticket only touches code, docs, and ticket files it fully owns; `depends-on: []`.
Working tree clean on `main` before branching.

### Confirmed design decisions (do not deviate without asking)

1. **`spawned-by:` is a list of `T-NNN` ids, `[]` default** — identical wire format to
   `depends-on:`, parsed by the existing `ticket.ParseDepends` (a generic bracketed-id-list
   parser reused, not duplicated).
2. **Required key + uniform backfill** (decision A, user-confirmed). `spawned-by` joins the
   audit's `requiredKeys`; **every existing ticket is backfilled with `spawned-by: []`** in this
   same change. No real historical parents are populated here — all existing tickets get `[]`
   (a true-lineage backfill is an optional separate follow-up). *If the user chose optional
   instead: skip the `requiredKeys` addition and the backfill; everything else stands.*
3. **Non-gating — the defining property.** `spawned-by` adds **no** transition guard and **no**
   done/merged check. A ticket may enter `3-in-development/` regardless of its `spawned-by`
   parents' status (or whether they are terminal). Audit only checks the ids **exist** and that
   a ticket does **not** list **itself**. A code comment at the audit site records that the
   omission of a gate is deliberate.
4. **No board column.** Lineage is provenance, not a scheduling axis; `BOARD.md` layout and
   `internal/board` are untouched.
5. **`pickle ticket new --spawned-by "<ids>"`** — new optional flag, comma-separated (brackets
   optional), default empty → `[]`. Ids are passed through to the scaffold; existence is left to
   `pickle board audit` (consistent with how `depends-on` is unvalidated at creation).
6. **Immutable by convention** — set at creation, like the History `source:` line. Documented as
   such; **not** audit-enforced (the audit has no cross-revision view).
7. **Frontmatter position:** the `spawned-by:` line sits **immediately after** `depends-on:` in
   every ticket, template, and scaffold, so the parallel-but-opposite pair reads together.

### Tasks

#### Task 1 — Model + parsing (`internal/ticket/ticket.go`)
- Add `SpawnedBy []string` to the `Ticket` struct (document it: "parsed spawned-by ids —
  lineage only, never a gate").
- In `LoadAll`, populate `SpawnedBy: ParseDepends(fm["spawned-by"])`.
- Update `ParseDepends`'s doc comment to state it parses any bracketed `T-NNN` list (used by
  both `depends-on` and `spawned-by`).
- `Scaffold`: add a `spawnedBy []string` parameter and emit a `spawned-by: <rendered>` line
  directly under `depends-on: []`. Add a tiny `renderIDList([]string) string` helper (`[]` for
  empty, `[T-018, T-019]` otherwise) and use it for the spawned-by line.

#### Task 2 — `ticket new --spawned-by` (`internal/cli/ticket.go`)
- Register `spawnedBy := fs.String("spawned-by", "", "lineage: ticket id(s) this one was born
  from, comma-separated (non-gating)")`.
- Parse it with `ticket.ParseDepends(*spawnedBy)` and pass the slice to `ticket.Scaffold(...)`.
- Update `ticketNewUsage` to include `[--spawned-by "T-NNN[,T-MMM]"]`.

#### Task 3 — Audit (`internal/audit/audit.go`)
- Add `"spawned-by"` to `requiredKeys` (decision 2).
- After the existing `depends-on` existence loop, add a `spawned-by` loop: for each id, error
  `"%s: spawned-by %s does not exist"` if unknown; error `"%s: spawned-by lists itself"` if the
  id equals the ticket's own id. **No** entry in the `3-in-development` dependency-gate loop.
- Add a one-line comment at that loop: lineage is provenance only — intentionally no
  done/merged/transition gate (contrast `depends-on`).

#### Task 4 — Skill payload docs (edits under `skill/`, mirrored via the `.agents` symlink)
- `skill/resources/TEMPLATE.md`: add the `spawned-by: [...]` line under `depends-on:` with a
  comment: `# lineage only — ticket(s) this was born from; [] if none; NEVER gates pickup`.
- `skill/resources/tickets-README.md` §3: add a **Lineage (`spawned-by`)** bullet right after
  the Dependencies bullet — same format as `depends-on`, **explicitly no transition guard**, set
  at creation/immutable, complements (does not replace) the History `source:` line. Add
  `spawned-by` to the §7 frontmatter list (line ~202).
- `skill/SKILL.md`: in *The rules (summary)* add a **Lineage** bullet mirroring the
  Dependencies one; in *Procedure: make it a ticket* (step ~5) and *Procedure: validate a
  ticket* note that a ticket born from a review/audit sets `spawned-by:` to the source
  ticket(s); in *audit the board* add "`spawned-by:` targets exist (but never gate)".
- `skill/resources/review-protocol.md`: where non-blocking findings spawn new TO DO tickets,
  note each new ticket sets `spawned-by: [<reviewed ticket id>]`.

#### Task 5 — Root README (`README.md`)
- Update the frontmatter-completeness list (line ~130) and the "targets exist" line (~132) to
  include `spawned-by`; if the `ticket new` blurb (~264) enumerates flags, add `--spawned-by`.

#### Task 6 — Backfill existing tickets
- Insert `spawned-by: []` immediately after the `depends-on:` line in **every** ticket file
  under `tickets/1-to-do/` … `tickets/7-dropped/` that lacks it (all 24, including T-024
  itself). Mechanical; verify none are skipped. (Real historical parents intentionally **not**
  populated — all `[]`.)

#### Task 7 — Tests
- `internal/ticket/ticket_test.go`: add `spawned-by:` to the parse fixture (line ~13) and
  assert `SpawnedBy` parses; add `"spawned-by"` to the required-keys assertion (line ~131);
  assert `Scaffold(...)` emits a `spawned-by:` line (both empty and non-empty inputs). Update
  the two `Scaffold(...)` calls for the new parameter.
- `internal/audit/audit_test.go`: add `spawned-by: %s` (default `[]`) to the fixture builder
  (line ~29); new cases: **(a)** valid non-empty `spawned-by` → clean; **(b)** dangling
  `spawned-by: [T-404]` → error `spawned-by T-404 does not exist`; **(c)** self-reference →
  error; **(d) the key guarantee:** a ticket in `3-in-development` whose `spawned-by` parent is
  **not** in `6-done`/not merged → **still clean** (no gate); **(e)** missing `spawned-by` key →
  error (required).
- `internal/cli/cli_test.go`: `ticket new --spawned-by "T-001"` writes `spawned-by: [T-001]`
  and `board audit` is clean; default (no flag) writes `spawned-by: []`.
- `internal/move/move_test.go` & `internal/sync/sync_test.go`: update the `ticket.Scaffold(...)`
  call sites for the new parameter (pass `nil`).

### Acceptance test

Run from the repo root (the `pickle` child):

```
just build            # compiles with the new flag + model field
just test             # all packages green, incl. the new audit/cli/ticket cases
just lint             # go vet clean
./pickle board audit  # => "... 0 error(s) ..." — proves backfill + required key are consistent
```

Then exercise the flag end-to-end **in a throwaway scratch checkout or temp dir** (do not leave
a demo ticket in the real board): `pickle ticket new "demo" --project pickle --spawned-by "T-001"`
produces frontmatter containing `spawned-by: [T-001]`, and `pickle board audit` reports 0
errors. The non-empty `--spawned-by` path is also covered non-interactively by the new
`cli_test.go` case, so the live repo stays clean.

Expected results: build/test/lint all green; `board audit` reports **0 errors, 0 warnings** on
the real repo; audit *does* error on the negative fixtures (dangling id, self-reference, missing
key) proving the validation bites; and the in-development-with-un-done-parent fixture stays
clean, proving `spawned-by` never gates.

### Docs update (mandatory when user-facing)

Covered by Tasks 4 and 5: the skill payload (`TEMPLATE.md`, `tickets-README.md` §3/§7,
`SKILL.md`, `review-protocol.md`) and the root `README.md`. `tickets/README.md` is only a
pointer and needs no change.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` and `pickle board audit` clean.
2. Docs (skill payload + README) updated.
3. Write a summary (files touched, the required-vs-optional decision as taken, backfill scope,
   anything deferred — e.g. real-parent backfill follow-up).
4. Suggest a Conventional Commit message, ticket id in brackets, e.g.:

   ```
   feat(ticket): add spawned-by lineage frontmatter (non-gating provenance) (T-024)

   New list field parallel in shape to depends-on but with no transition guard;
   parsed, scaffolded (--spawned-by flag), audited for existence + self-reference,
   documented in the skill payload, and backfilled ([]) across existing tickets.
   ```

5. Commit locally on the branch; **do not push or open an MR without user approval**. Present
   the message; on approval finalize + push + open the MR (merging is the human's).

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: chat
- 2026-07-25 — TO DO → READY: implementation plan complete
