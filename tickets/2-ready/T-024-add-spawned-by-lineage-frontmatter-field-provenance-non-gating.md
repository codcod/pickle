---
id: T-024
title: add spawned-by: lineage frontmatter field (provenance, non-gating)
project: pickle
depends-on: []
impact: medium
complexity: low
cost: M
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
  frontmatter-invariant list is updated. Populating real historical parents is out of scope —
  it is **T-025**, which hard-depends on this ticket.

### Couplings

- Soft coupling to the review/audit procedures (SKILL.md *make it a ticket* / *validate a
  ticket*), which are where spawned tickets are born and would set this field.
- **T-025** (`depends-on: [T-024]`) backfills the *true* historical lineage from the History
  `source:` lines once this ships. This ticket writes `[]` everywhere, including into T-025's
  own file — T-025 owns every real parent value, its own included.
- **T-027** (no dependency either way) proposes the same self-reference check for the shipped
  `depends-on:` validator, deliberately left out of this ticket's scope.
- No hard `depends-on:`.

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
   same change. **No real historical parents are populated here — every existing ticket gets
   `[]`, T-025's own file included**, even though its parent (T-024) is known. T-025 owns the
   true-lineage backfill wholesale; splitting it would leave two tickets editing the same field.
3. **Non-gating — the defining property.** `spawned-by` adds **no** transition guard and **no**
   done/merged check. A ticket may enter `3-in-development/` regardless of its `spawned-by`
   parents' status (or whether they are terminal). Audit only checks the ids **exist** and that
   a ticket does **not** list **itself**. A code comment at the audit site records that the
   omission of a gate is deliberate. The matching self-reference check for `depends-on:` is
   **out of scope** (it changes shipped-validator behaviour) — it is T-027; do not touch the
   `depends-on` loop's conditions.
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
- `board audit` invariant list: add `spawned-by` to the frontmatter-completeness bullet
  (**line 136**, `frontmatter is complete (…)`) and to the targets-exist bullet (**line 138**,
  `every depends-on: target exists`) — wording it as *exists but never gates*.
- `pickle ticket new` section (**line 266**): add `[--spawned-by "T-NNN[,T-MMM]"]` to the usage
  line and a clause to the prose below it.
- **Leave `PLAN.md` alone** — it is the historical phased design record, not user docs.

#### Task 6 — Backfill existing tickets
- Insert `spawned-by: []` immediately after the `depends-on:` line in **every** ticket file
  under `tickets/1-to-do/` … `tickets/7-dropped/` that lacks it — **27 files at time of
  refinement (T-001…T-027), including T-024 itself**; re-count at implementation time rather
  than trusting that number, and include any ticket filed in the meantime.
- Verify none are skipped:
  `rg -L --files-without-match '^spawned-by:' tickets/*/*.md` must print nothing.
- Real historical parents intentionally **not** populated — all `[]` (decision 2; T-025 owns
  the true-lineage pass).

#### Task 7 — Tests
- `internal/ticket/ticket_test.go`: add `spawned-by: [T-004]` to the `sample` fixture
  (**after line 13**) and assert `LoadAll`/`ParseDepends` yields it; add `"spawned-by"` to the
  required-keys assertion in `TestScaffoldIsAuditClean` (**line 131**); assert `Scaffold(...)`
  emits `spawned-by: []` for `nil` and `spawned-by: [T-001, T-002]` for a two-id slice. Update
  the two `Scaffold(...)` calls (**lines 126, 153**) for the new parameter.
- `internal/audit/audit_test.go`: the `ticketFile(...)` builder (**line 28**) has **11 call
  sites** — do **not** add a parameter. Emit a fixed `spawned-by: []` line in the builder and
  add a `withSpawnedBy(body, ids string) string` helper that string-replaces it (mirroring
  `internal/move/move_test.go:39`'s `depends-on` trick) for the cases that need a value. New
  cases: **(a)** valid non-empty `spawned-by` → clean; **(b)** dangling `spawned-by: [T-404]` →
  error `spawned-by T-404 does not exist`; **(c)** self-reference → error;
  **(d) the key guarantee:** a ticket in `3-in-development` whose `spawned-by` parent is **not**
  in `6-done`/not merged → **still clean** (no gate — contrast the existing `depends-on` gate
  case); **(e)** missing `spawned-by` key → `frontmatter missing "spawned-by"`.
- `internal/cli/cli_test.go`: `ticket new --spawned-by "T-001"` writes `spawned-by: [T-001]`
  and `board audit` is clean; default (no flag) writes `spawned-by: []`.
- `internal/move/move_test.go` (**line 37**) & `internal/sync/sync_test.go` (**line 37**):
  update the `ticket.Scaffold(...)` call sites for the new parameter (pass `nil`).

### Acceptance test

Run from the repo root (the `pickle` child):

```
just build            # compiles with the new flag + model field
just test             # all packages green, incl. the new audit/cli/ticket cases
just lint             # go vet clean
./pickle board audit  # => "... 0 error(s) ..." — proves backfill + required key are consistent
rg -L --files-without-match '^spawned-by:' tickets/*/*.md   # => no output (backfill complete)
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
3. Write a summary (files touched, backfill scope + file count, anything deferred — the
   real-parent backfill is T-025, the `depends-on` self-check is T-027).
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
- 2026-07-25 — re-refined in place (stays READY): plan re-verified against the current tree
  (all paths/symbols/line refs resolve); backfill scope corrected 24 → 27 files; T-025 named as
  the true-lineage follow-up (uniform `[]` confirmed, T-025's own file included); `depends-on`
  self-check split out as T-027; test tasks made concrete (no new `ticketFile` parameter); cost
  range collapsed S-M → M (7 tasks across 10 code/test files, 5 doc surfaces, 27 backfills)
