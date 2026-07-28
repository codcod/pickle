---
id: T-060
title: pickle ticket renumber: retroactively migrate a child to a new ticket_prefix
project: pickle
depends-on: [T-058]
spawned-by: [T-058]
impact: low
complexity: medium
cost: M
---

# T-060 — pickle ticket renumber: retroactively migrate a child to a new ticket_prefix

## Description

Split out of **T-058** at refinement (2026-07-28). T-058 adds an optional per-child
`ticket_prefix` (default `T`) and a **per-child counter**, but deliberately does **not**
retroactively rename a child's existing tickets — a new workspace sets its prefix before filing
anything, and T-058's audit invariant (a ticket's id prefix must match its project's configured
prefix) simply goes red if you change a populated child's prefix without migrating. This ticket
is that migration.

**Hard dependency on T-058 (pending user sign-off before it goes in `depends-on:`).** Renumbering
is meaningless without the prefix schema T-058 introduces; T-060 must not enter development until
T-058 is done and merged.

The migration itself must:

- rename every `<OLD>-NNN-<slug>.md` in that child to `<NEW>-NNN-<slug>.md` — a **pure
  number-preserving prefix swap** (`T-042` → `RICK-042`). Prefixes are unique per child
  (audit-enforced), so the target namespace is always empty; numbers stay stable and the
  resulting gaps in the old prefix's sequence are legal (rules §3). *(Refinement 2026-07-28
  corrected an earlier aside that claimed numbers are re-sequenced — they are not.)*
- rewrite the `id:` frontmatter, and **every `depends-on:` / `spawned-by:` / `family:` reference
  to a renamed id across all tickets in all children** (references cross children — rules §3);
- rewrite `## History` lines that mention the old id;
- update the child's `ticket_prefix` in `pickle.toml`;
- record the two unfixable classes: commit subjects already in git that cite `(T-NNN)`, and
  branch names already pushed.

**Must never touch `tickets/NOTES.md` prose** (tooling is forbidden to rewrite the hand-written
planning notes) — the mapping table is what lets a human update those by hand.

**Refinement decisions (2026-07-28).** Resolving the ticket's open question — *real command vs
documented manual procedure* — the answer is **a documented manual procedure plus a read-only
mapping-table helper**, not a mutating command. Rationale: a prefix migration is a rare,
one-time-per-child event, and with number-preserving swap the edits are mechanical; the only real
hazards of doing it purely by hand are **substring id matches** (`T-4` inside `T-40`) and
**accidentally editing `NOTES.md`/git history**. A read-only planner removes both by listing the
exact old→new map and every precise reference site to edit, then leaving the human to apply the
edits and verify with `pickle board audit`. This is the "mapping table only" deliverable, made
precise — no atomic multi-file mutation engine to build (why complexity drops `high`→`medium` and
cost `L`→`M`). A future ticket may add an `--apply` mutator if the manual path proves painful.

## Implementation Plan

**Feature branch (in `pickle`):** `feat/T-060-ticket-renumber-planner`, cut from `main`.

**Prerequisite gate.** Hard `depends-on: [T-058]` — the per-child `ticket_prefix` + per-child
counter schema. **T-058 is DONE and merged to `main`** (commit recorded), so the gate is already
satisfied; recorded in frontmatter for provenance and correctness, not because it blocks.

### Confirmed decisions (locked at refinement)

1. **Deliverable = a read-only planner + a documented manual procedure**, NOT a mutating command.
   The planner writes nothing; the human applies the edits by hand, once, and verifies with
   `pickle board audit`.
2. **Number-preserving prefix swap** (`T-042` → `NEW-042`). No re-sequencing. Refuse if the target
   prefix is already in use by any ticket in the tree (so the swap can never collide).
3. **Manual apply.** The `pickle.toml` `ticket_prefix` change, the file renames, the reference
   edits, and any `NOTES.md` updates are all done by hand. The planner only *tells you exactly
   what to change*.
4. **Read-only means read-only.** The planner refuses to run unless the tree is already
   audit-clean (a dirty tree would make the reference scan unreliable), and it exits non-zero on
   any validation failure (unknown child, illegal prefix shape, occupied prefix, dirty tree).
5. **`NOTES.md` is never scanned or listed** — tooling must not even appear to touch the
   hand-written notes.

### Tasks

1. **New package `internal/renumber/renumber.go`** — pure, read-only planning.
   - `type Ref struct { File, Kind, Line string; LineNo int }` (Kind ∈ `id` / `depends-on` /
     `spawned-by` / `family` / `history`).
   - `type Mapping struct { Old, New string }`.
   - `type Plan struct { Child, OldPrefix, NewPrefix string; Mappings []Mapping; Refs []Ref;
     GitCitations []string; Branches []string }`.
   - `func Build(root string, cfg *config.Config, child, newPrefix string) (Plan, error)`:
     - Validate: `cfg.Project(child)` exists (else error); `newPrefix` matches
       `config` prefix shape (reuse the exported check — see task 2 note); `newPrefix` not equal
       to the child's current `cp.Prefix()`; and **not already used** by any loaded ticket id
       (`ticket.SplitID`) — else `fmt.Errorf("prefix %q is already in use", newPrefix)`.
     - Require an **audit-clean tree**: run `audit.Audit(root, cfg)`; if `len(Errors) > 0`, refuse
       with a message pointing at `pickle board audit`.
     - Load all tickets (`ticket.LoadAll`). The migrating set = every ticket whose
       `Project() == child`. Build `Mappings` by number-preserving swap:
       `SplitID(id)` → `NewPrefix + "-" + <zero-padded NNN>` (mirror `cli`'s `%03d`).
     - Build the migrating-id lookup set. Scan **every** ticket's text (all children) line by line;
       for each line, extract id-shaped tokens with a shared token regex
       (`[A-Z][A-Z0-9]*-\d+`, boundary-safe so `T-4` never matches inside `T-40`) and, when a token
       is in the migrating set, emit a `Ref` tagged by which frontmatter key / `## History` the
       line belongs to. **Skip `tickets/NOTES.md` entirely** (it is not a ticket file — `LoadAll`
       already ignores it; assert this in a test).
     - `GitCitations`: `git log --oneline` grepped for `(OLD-NNN)` subjects (shell out via
       `os/exec`; on any git error, set a single "git unavailable — check citations by hand" note
       rather than failing — the planner must work in a non-git checkout too).
     - `Branches`: `git branch --list "*T-NNN*"` style match for migrating ids (same soft-fail).
   - No file writes anywhere in this package.

2. **CLI wiring — `internal/cli/ticket.go` + `internal/cli/cli.go`.**
   - Add `renumber` to `runTicket`'s switch and a `ticketRenumberUsage` const:
     `usage: pickle ticket renumber <child> <NEW-PREFIX>`.
   - `runTicketRenumber(args)`: require exactly two positional args (child, NEW-PREFIX); load config;
     call `renumber.Build`; on error `return errf("%v", err)`; else print, in order: a one-line
     header (`renumber plan: <child>  <OLD> → <NEW>  (read-only; nothing written)`), the **old→new
     mapping table** (aligned columns), the **reference checklist** grouped by file
     (`path:line  [kind]  <id> → <newid>`), the git-citation and branch notes, and a closing
     **manual-steps block** (edit `ticket_prefix` in `pickle.toml`; rename the listed files;
     apply the listed reference edits; update `NOTES.md` by hand if it mentions any migrated id;
     run `pickle board audit`). Return `exitOK`.
   - Export a prefix-shape validator from `internal/config` if one is not already exported
     (`ticketPrefixRE` is currently unexported at `config.go:35`). Add
     `func ValidTicketPrefix(s string) bool { return ticketPrefixRE.MatchString(s) }` and reuse it
     in `renumber.Build` — do **not** grow a second prefix regex.
   - Document the subcommand in `cli.go`'s Flow-commands usage block (one entry, marked
     *read-only*).

3. **Docs — `docs/user-manual/cli-reference.adoc`.**
   - Add a `[#cmd-renumber] == pickle ticket renumber` section: what the planner prints, that it is
     **read-only**, and the ordered **manual migration procedure** (the same steps the CLI's
     manual-steps block lists), ending on `board audit` as the green check. State the
     number-preserving rule and the two unfixable classes (git subjects, pushed branches).
   - Update the existing sentence at `cli-reference.adoc:151`
     (“…until you renumber them.”) to cross-reference `<<cmd-renumber>>`.

4. **Docs — `skill/resources/tickets-README.md`.** The §3 ID bullet already says re-homing “is a
   renumber, not a free relabel (`pickle ticket renumber`)”. Refine that parenthetical to note the
   command is a **read-only planner + manual procedure** (edit canonical `skill/resources/`, not
   the `.claude/skills/` symlink — the drift test reads the canonical copy).

### Acceptance test

Self-host policy: no `install`/`board sync` against this repo. Acceptance = Go unit tests
(`just test`) + one throwaway-dir smoke with a copied binary.

1. **Unit tests — `just test` green**, including new cases:
   - `internal/renumber`:
     - clean plan: a `demo` child on default `T` with tickets `T-001`/`T-042`, plus a second child
       `rick` (prefix `RICK`) whose ticket carries `depends-on: [T-042]` and `spawned-by: [T-001]`
       and a `demo` ticket with `family: T-001` — assert Mappings are number-preserving
       (`T-042`→`NEW-042`), and Refs include the cross-child `depends-on`/`spawned-by` sites and
       the `family` site, each tagged by kind and correct line.
     - **token boundary**: a ticket referencing `T-4` is not matched when `T-40` is the migrating
       id (and vice-versa).
     - **NOTES.md ignored**: a `tickets/NOTES.md` mentioning `T-042` produces no Ref.
     - refusals: unknown child; illegal prefix shape (`rick`, `TOOLONGPREFIX9`); prefix already in
       use (`RICK` when `rick` child exists); prefix equal to current; dirty tree (audit error) —
       each returns an error, and the returned `Plan` is empty/zero.
     - the package performs **no writes**: capture a checksum of the tree before/after `Build` in
       one test.
   - `internal/cli`: `ticket renumber demo NEW` prints a mapping line and the manual-steps block
     and exits `exitOK`; `ticket renumber demo rick` (occupied) exits non-zero; assert the tree is
     byte-unchanged after both (read-only).
   - `internal/config`: `ValidTicketPrefix` accepts `RICK`/`T`, rejects `rick`/``/`AB_C`/9-char.
2. **Throwaway-dir smoke** (copy the freshly built binary; never run WIP against this repo):
   ```
   just build
   D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
   ./pk install --project demo .
   ./pk project add rick ./rick --ticket-prefix RICK        # second child
   ./pk ticket new "demo one" --project demo                # T-001
   ./pk ticket new "demo two" --project demo                # T-002
   ./pk ticket new "rick one" --project rick --spawned-by T-001   # RICK-001 refs T-001
   ./pk board audit                                          # clean
   ./pk ticket renumber demo DEMO                            # READ-ONLY plan
   ./pk board audit                                          # STILL clean — nothing changed
   ./pk ticket renumber demo RICK ; echo "exit=$?"           # occupied prefix → non-zero
   ```
   Assert: the `renumber demo DEMO` output lists `T-001 → DEMO-001`, `T-002 → DEMO-002`, the
   `RICK-001` `spawned-by` reference site, and the manual-steps block; that `board audit` is still
   clean afterward (read-only); and that the occupied-prefix call exits non-zero. Confirm `git
   status`/file mtimes show the tree untouched by the planner.

### Docs step

`just docs-check` green after the `cli-reference.adoc` + `tickets-README.md` edits.

### Finish step

Run `just build`, `just test`, `just lint`, `just docs-check` until all green. Write the
implementation summary. Prepare the child commit message
`feat(cli): add read-only ticket renumber planner + manual prefix-migration procedure (T-060)`
(local commit on the ticket branch; **no push / no MR without user approval** per the commit
policy). Then `pickle ticket move T-060 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: pickle ticket new
- 2026-07-28 — TO DO → READY: plan complete: read-only planner + manual procedure; preserve-numbers; T-058 satisfied
- 2026-07-28 — READY → DROPPED: automation not needed; unity is the only real case and is a one-time guided manual migration, not a pickle command
