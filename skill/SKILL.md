---
name: ticket-flow
description: Operate a ticket-based, board-driven feature flow (installed by the `pickle` CLI) across one or more connected child-projects. Use when asked to "make it a ticket", "refine ticket T-NNN" (make it ready), "implement ticket T-NNN", "rework ticket T-NNN", "validate ticket T-NNN" (or "review ticket T-NNN"), "audit the board", or move a ticket between statuses. One markdown ticket per feature; a ticket's status is the directory it lives in; each ticket targets one registered child-project via `project:` frontmatter; a hand-maintained BOARD.md is the live index; reviews classify findings as blocking vs non-blocking; pushing a child-project requires explicit user approval.
---

# Ticket flow

A lightweight, repo-native feature flow. Every change to a project flows through **one
artifact: a ticket** — a markdown file whose **status is the directory it lives in**, that
**targets one child-project** (`project:` frontmatter), carrying an append-only history. A
hand-maintained `BOARD.md` is the live index.

This flow is installed and operated with the **`pickle` CLI**: `pickle install` scaffolds it,
`pickle project add` registers connected child-projects, and `pickle ticket`/`pickle board`
perform the deterministic mechanics (id allocation, atomic moves, invariant audit). This skill
defines how an agent *operates* the flow — the judgement `pickle` deliberately does not
automate.

Bundled resources (this skill's own directory, embedded in and installed by `pickle`). **They
are the single source of truth — do not copy them into projects.** The one exception is the
board skeleton, which is instance data:

- `resources/tickets-README.md` — the authoritative rules (§0–§8); projects keep only a
  short pointer `tickets/README.md` that resolves `§N` references here
- `resources/TEMPLATE.md` — author every new ticket from this (read it from the skill;
  projects keep no local copy)
- `resources/BOARD.md` — the board skeleton `pickle install` copies → the project's
  `tickets/BOARD.md`
- `resources/review-protocol.md` — the generic review procedure; a project layers its own
  addenda (overarching + per-child, in `pickle.toml`) keyed to this protocol's step numbers

## When to use

- **"Install / scaffold the ticket flow"** → run `pickle install` (then `pickle project add`
  for further children); see *Install & register*.
- **"Make it a ticket" / "file a ticket"** → *Procedure: make it a ticket*.
- **"Refine ticket T-NNN" / "make it ready"** → *Procedure: refine a ticket*.
- **"Implement ticket T-NNN"** → *Procedure: implement a ticket*.
- **"Rework ticket T-NNN"** → *Procedure: rework a ticket*.
- **"Validate ticket T-NNN"** (or **"review ticket T-NNN"** — synonyms) → *Procedure:
  validate a ticket*.
- **"Audit the board"** → *Procedure: audit the board*.

## Install & register

`pickle install` (run once in the overarching project) creates `tickets/` with the seven
ordered status directories (`1-to-do/` … `7-dropped/`), copies the board skeleton to
`tickets/BOARD.md`, writes the `tickets/README.md` pointer, installs this skill for the
detected agents (`.agents/skills/ticket-flow/`, symlinked into `.claude/skills/` for Claude
Code), injects the `AGENTS.md`/`CLAUDE.md` marker block, writes `pickle.toml`, and registers
the first child-project. Register more connected children with
`pickle project add <name> <path>`. Install scope is **per-project** — nothing is written to
`~/`.

## Project configuration (in `pickle.toml` + the `AGENTS.md` marker block)

The flow is generic; each project plugs in specifics. **Overarching** config: the commit
policy and the optional overarching review addendum. **Per child-project** (`[[project]]` in
`pickle.toml`): name + path, build/validate commands, branch & commit conventions, per-child
WIP limits, and an optional per-child review addendum. Defaults:

- **Build target** — every ticket targets exactly one registered child (`project:`
  frontmatter); its requirements, tasks, paths, and acceptance tests describe **that child's
  repo**.
- **Branch & commit** — `feat/T-NNN-<slug>` cut inside the target child's repo; Conventional
  Commits with the ticket id appended in brackets at the end of the subject.
- **Commit policy** — child-projects are publish-gated (no push/MR without approval; approval →
  finalize + push + open MR; the human merges). The overarching project's bookkeeping (tickets,
  board, docs) may be committed automatically, always with explicit pathspecs.
- **WIP limits** — `3-in-development/` ≤ 1, `4-in-review/` ≤ 1, enforced **per child**.

## The rules (summary — full text in `resources/tickets-README.md`)

- **Status = directory.** A ticket's status *is* the folder it sits in. There is **no `status:`
  field**; record every transition as a dated line in the ticket's `## History`.
- **Child-project target.** Every ticket has a `project:` frontmatter naming one registered
  child; its `feat/` branch is cut in that child's repo.
- **IDs (one global namespace).** `T-NNN`, monotonically increasing, **never reused**, shared
  across all children (`max(existing across all status dirs) + 1`). Filename `T-NNN-<slug>.md`.
- **Priority.** `impact` / `complexity` / `cost` frontmatter; ordering lives in `BOARD.md`,
  within each child's sub-group.
- **Dependencies (may cross children).** `depends-on:` frontmatter. A ticket may not enter
  `3-in-development/` while any dependency is not in `6-done/` **and merged to the base of its
  own child's repo** (done ≠ merged; the human merges, and may lag — see rules §3).
- **Lineage (may cross children).** `spawned-by:` frontmatter — the ticket(s) this one was born
  from (review finding, board audit, refinement split). Same wire format as `depends-on:` and
  the exact opposite in behaviour: **provenance only, it gates nothing**, so never overload
  `depends-on:` to express it.
- **READY gate.** A ticket is READY only when its `## Implementation Plan` is a complete,
  self-contained prompt: feature branch (in the child), prerequisites, confirmed decisions,
  concrete tasks, a runnable acceptance test, a docs step, and a finish step.
- **Board rule.** Update `BOARD.md` on **every** ticket move — same change as the directory
  move and the `## History` append.
- **WIP limits per child.** `3-in-development/` ≤ 1, `4-in-review/` ≤ 1, counted independently
  per child.
- **Every move** = move the file + one dated `## History` line + one `BOARD.md` edit, in the
  same change. Prefer `pickle ticket move` — it does all three atomically and enforces the
  state machine + per-child WIP.

## Procedure: make it a ticket

When asked to turn an idea, finding, or request into a ticket:

1. **Determine the target child-project.** If the request doesn't make it obvious and more than
   one child is registered (`pickle project list`), **ask which child-project the feature
   targets** — do not guess.
2. **Deduplicate first.** Read the titles (and, where plausible matches exist, Descriptions)
   of every ticket in every **non-terminal** directory (`1-to-do/` through `5-rework/`),
   plus the titles in `6-done/`. If the idea substantially overlaps an existing ticket, **do
   not file a duplicate** — extend or cross-reference the existing ticket and tell the user
   which ticket absorbed it. If the overlap is partial, ask the user whether to merge or split.
3. **Author** the ticket: run `pickle ticket new "<title>" --project <name>` to allocate the
   next `T-NNN` (max across *all* status dirs + 1), scaffold from `resources/TEMPLATE.md` into
   `1-to-do/` with `project:` set, and add the board row under that child's sub-group. When the
   ticket is born from another one, add `--spawned-by "T-NNN[,T-MMM]"`. Then fill in the
   Description prose.
4. **Grade it** (impact / complexity / cost) **against the existing backlog** — re-grade
   neighbours if the comparison shifts them.
5. Note any soft couplings (including cross-child ones) in the Description; hard `depends-on:`
   only with user sign-off. Record where the ticket came from: the `created … source: …`
   History line in prose, and — whenever the source is another ticket — that ticket's id in
   `spawned-by:` (lineage; it never blocks pickup, so it needs no sign-off). Confirm the board
   row + `created` History line are present (`pickle board audit`).

## Procedure: refine a ticket

When asked to refine ticket T-NNN (or "make it ready"):

1. The ticket is normally in `1-to-do/`. Read it in full, including `## History`.
2. **Re-verify the Description against the current target child-project** — the ticket may
   predate changes that invalidated its assumptions (paths, names, shipped behaviour). Update
   the Description first; it is the spec.
3. **Surface open decisions to the user** — anything ambiguous the plan must pin down.
4. **Write the Implementation Plan** against the READY gate (rules §4): feature branch (in the
   child's repo), prerequisite gate, confirmed decisions, concrete tasks with exact paths,
   runnable acceptance test, docs step, finish step.
5. **Re-grade** impact/complexity/cost against the backlog; add new hard `depends-on:` entries
   (possibly cross-child) only with user sign-off.
6. When all seven gate items hold: `pickle ticket move T-NNN ready --reason "plan complete"`
   (moves the file, appends History, updates the board).

## Procedure: implement a ticket

When asked to implement ticket T-NNN:

1. **Read the ticket in full.** It must be in `2-ready/` — if not, stop and tell the user why.
2. **Validate dependencies:** every `depends-on:` ticket is in `6-done/` **and merged to the
   base of its own child's repo** (check the board's `merged` column / the dependency's
   History; if unmerged, stop and ask the human to merge first). **Validate WIP** for the
   target child. Stop and report if not satisfied.
3. **Applicability gate — re-verify the plan is still worth executing, before any move or
   branch.** Run on every pickup, unconditionally:
   - **Spawn a fresh sub-agent** for the audit — free of the implementer's sunk-cost bias.
     Brief it with everything (the ticket, `BOARD.md`, soft couplings, the target child's
     locked-decision docs).
   - **Scope the mandate to the ticket's own assumptions plus the board delta since it went
     READY.** For each assumption, confirm it is still **true**, **required**, and **worth it**.
   - The agent returns a **findings list classified like a review**. **Present it and get
     approval on the routing:** clean → proceed; plan invalidated → move back to
     `2-ready/`/`1-to-do/` and re-refine; adjacent work → new `1-to-do/` ticket(s).
4. **Move** the ticket: `pickle ticket move T-NNN in-development --reason "picked up"`.
5. **Create the feature branch** `feat/T-NNN-<slug>` **inside the target child-project's repo**
   (from the agreed base, default `main`).
6. **Execute the Implementation Plan top-to-bottom** — it is the executable prompt.
7. **Run the acceptance test** and the child's build/validate commands until green.
8. **Finish:** write the summary, prepare the suggested commit message (ticket id in brackets).
   Commit locally on the ticket branch; **do not push or open a merge request without user
   approval** (commit policy). `pickle ticket move T-NNN in-review --reason "acceptance green"`
   and hand back.

## Procedure: rework a ticket

When asked to rework ticket T-NNN (a review found blocking findings):

1. The ticket must be in `5-rework/` — if not, stop and explain.
2. Read the ticket's `## Review` section: the **blocking findings are the entire scope**.
   Implement nothing else — any new work needs a new ticket.
3. On the **same** `feat/T-NNN-<slug>` branch (in the target child's repo), fix only the listed
   findings (local commits per the commit policy — they make the re-review diffable).
4. Re-run the acceptance test and the child's build/validate commands until green.
5. Record what was fixed against each finding in `## Review`.
6. `pickle ticket move T-NNN in-review --reason "findings fixed"` and hand back for a **scoped
   re-review**.

## Procedure: validate a ticket

"Validate ticket T-NNN" and "review ticket T-NNN" are the same procedure. Follow
`resources/review-protocol.md` (plus the project's layered addenda — overarching + the ticket's
child). In short:

1. The ticket must be in `4-in-review/`. Audit implementation, quality, consistency, and docs
   (running the child's configured commands); classify each finding **blocking** (→
   `5-rework/`, scoped re-review after the fix) vs **non-blocking** (→ new `1-to-do/` ticket(s),
   each filed with `--spawned-by "<reviewed ticket id>"`; original proceeds to `6-done/`).
2. Findings go into the ticket's own `## Review` section — no separate file.
3. On a concluding verdict, **move the ticket** (`pickle ticket move …`).
4. **Present the child-project commit message** (and merge-request attributes) **to the user
   for approval**. Only after approval: finalize the branch (squash or keep history), push, and
   **create the merge request** in that child's repo. Never push a child or open an MR without
   approval; **merging is the human's**.
5. Overarching-repo bookkeeping (ticket edits, moves, board) is committed per the project's
   commit policy (may be automated, explicit pathspecs only).

## Procedure: audit the board

When asked to audit the board (or after a burst of moves, as a self-check):

```
pickle board audit
```

It verifies the flow's invariants mechanically: every ticket appears exactly once on the
board, in the section matching its directory (under its child-project's sub-group); every board
row has a backing file; ids are unique and match filenames; frontmatter is complete with legal
grade values and a `project:` that names a registered child; `depends-on:` targets exist;
`spawned-by:` targets exist and no ticket cites itself (but lineage never gates); per-child WIP
limits hold; each ticket's last History transition matches its directory; and
in-development tickets have all dependencies done (warning if a done dependency has no `merged`
History line). Fix every error it reports — an error is a broken invariant, not a judgement
call.

## Notes

`pickle` installs this skill **per project** (into `.agents/skills/ticket-flow/`, symlinked for
Claude Code); it does not install globally, and each project pins its own payload version.
`pickle upgrade` refreshes the installed skill + marker block from the binary without touching
tickets.
