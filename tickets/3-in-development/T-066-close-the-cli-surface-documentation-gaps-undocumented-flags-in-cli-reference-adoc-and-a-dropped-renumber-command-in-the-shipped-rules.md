---
id: T-066
title: close the CLI-surface documentation gaps: undocumented flags in cli-reference.adoc and a dropped renumber command in the shipped rules
project: pickle
depends-on: []
spawned-by: [T-019]
impact: medium
complexity: low
cost: S
---

# T-066 — close the CLI-surface documentation gaps: undocumented flags in cli-reference.adoc and a dropped renumber command in the shipped rules

## Outcome

After this ships, every flag `pickle`'s CLI actually accepts (e.g. `ticket new --family`) is documented in `docs/user-manual/cli-reference.adoc`, and the shipped rules no longer describe a `renumber` command that isn't there.

## Description

Spawned by the **T-019** review (findings N3 and N5), batched by theme: **shipped text that
describes the CLI surface has drifted from the CLI**. T-019 pruned `DESIGN.md` §4–§7 down to a
pointer — *"the live command surface, with every flag, is `docs/user-manual/cli-reference.adoc`"*
— which makes the manual's completeness load-bearing in a way it was not before. It is not
complete today.

### 1. Flags the CLI has and `docs/user-manual/cli-reference.adoc` never mentions

Swept by comparing every `fs.String/Bool` flag in `internal/cli/*.go` against the manual:

- **`pickle ticket new --family <ID>`** (`internal/cli/ticket.go:85`, shipped by T-059) — absent
  from the manual entirely: not in the `<<cmd-ticket-new>>` synopsis (`cli-reference.adoc:647`,
  re-verified at refinement — the anchor has drifted repeatedly as other tickets landed prose
  above it; search the `[#cmd-ticket-new]` heading rather than trusting this number), not in
  its prose, not anywhere under `docs/` (`rg -n family docs/` returns nothing). The command's
  own `usage:` string (`ticket.go:38`) *does* list it, so `pickle help` and the manual disagree.
- **`pickle project add --branch-prefix`, `--wip-dev`, `--wip-review`, `--build`, `--test`,
  `--lint`, `--docs`** (`internal/cli/project.go:69-76`) — the manual documents only
  `--ticket-prefix` and then says per-child keys *"can then be set in `pickle.toml`"*
  (`cli-reference.adoc:224-234`, `[#cmd-project]`'s first bullet — re-verified at refinement),
  which reads as *there are no flags for these*. There are.
- **`pickle flow show|list`** (`internal/cli/flow.go`, shipped by T-073) — a whole command with
  no section in the manual: no `[#cmd-flow]`, no Overview-table row, and no mention of the
  `flow` key it reads in `docs/user-manual/configuration.adoc`'s `pickle.toml` reference.
  `usage()` lists it, so `pickle help` and the manual disagree — the same defect shape as
  `--family` above. Added by T-073's review (finding F4, disposition `folded`), which its own
  Docs-update section pre-authorised: it judged the omission out of scope for a rename ticket
  and named this ticket as the home. Note the shipped value is currently constrained to
  `"brine"` by `Config.Validate()`, so document it as the one legal value rather than as an
  open-ended setting.

### 2. A dropped command cited as the mechanism, in the payload every project installs

`skill/resources/tickets-README.md:266-267` (re-verified at refinement; the exact lines have
drifted twice already as other tickets edited nearby text — see History) states that re-homing
a ticket to a differently-prefixed child *"is a renumber, not a free relabel (`pickle ticket
renumber`)"*, and `internal/audit/audit.go:101-103` comments that a mismatched prefix *"goes
red here until `pickle ticket renumber` (T-060) runs"*. **T-060 is in `7-dropped/`** — dropped 2026-07-28 with the
reason *"automation not needed; unity is the only real case and is a one-time guided manual
migration, not a pickle command"*. So the shipped rules point a user at a command that does not
exist and is not planned. This is the T-041/T-022 defect class (stale text that still reads as
authoritative), and it ships inside the skill payload, so every installed project carries it.

T-019 fixed the same claim where *it* had introduced it (`DESIGN.md` §3 decision 2, disposition
`fixed inline`); these two occurrences are pre-existing and outside a docs-only ticket's scope,
which is why they are here.

### 3. A markup bug that mangles a sentence in the shipped PDF

Added 2026-08-06 by the **T-057** review (finding N10, disposition `folded`) — same file, same
"shipped text is wrong" theme, and pre-existing, so it was not the reviewer's to fix inline.

`cli-reference.adoc`'s `pickle doctor` bullet list writes `` `.pi/extensions/*.ts` ``. The `*`
inside the glob opens an **unconstrained bold** span that runs on to the next `*` — the one
opening `*warning*` two lines later. The PDF therefore renders:

> pickle-owned agent scaffolds (.pi/extensions/.ts) match the shipped version — a drifted file
> is a \*warning pointing at pickle upgrade

— the glob loses its asterisk and the emphasis leaks. Fix: use a passthrough (`+.pi/extensions/*.ts+`)
or escape the asterisk, then re-render and confirm. Cheap, and worth sweeping the file for other
globs inside backticks while it is open.

### Shape of the fix (for refinement)

- Document the missing flags where the manual already documents their siblings — no new page.
- Either re-word the two `renumber` references to describe the manual migration, or state
  plainly that no command exists; the audit comment should cite the dropped ticket as dropped.
- Worth considering as the durable fix for item 1: a test that diffs the flag set registered in
  `internal/cli` against the flags named in `cli-reference.adoc`, so the next flag cannot ship
  undocumented. Price it at refinement — it may be more machinery than the gap justifies.

### Soft couplings

- **T-019** (`DESIGN.md`) — lineage only; its §7 pointer is what raises the stakes here.
- **T-022** / **T-041** — same defect class (stale authoritative text in shipped payload /
  marker block). **T-022 now overlaps in one file** (`skill/resources/tickets-README.md`): it
  added a precedence blockquote to §0 and rewrote §8's pickup-gate paragraph, which is what
  shifted the renumber claim from `:122-123` to `:127-128`. The lines are disjoint from this
  ticket's, but re-verify them at refinement. T-022 also already added the missing
  `ticket_prefix` bullet to `docs/user-manual/configuration.adoc:55` — **do not double-fix it**;
  this ticket still owns `cli-reference.adoc`.
- **T-058** / **T-060** — the `ticket_prefix` schema and its dropped migration command.

## Implementation Plan

### 0. Feature branch (mandatory)

Before any change, in the `pickle` repo (this is an in-tree child, `path = "."`):

```
git checkout main
git checkout -b feat/T-066-close-cli-surface-doc-gaps
```

Commit locally as you go (WIP commits are fine — tidy them into atomic commits before
presenting for approval, per rules §0 for a root-path child). Publish only per this
project's commit policy: no push / no MR without explicit user approval.

### Prerequisite gate (hard)

None. All three re-verifications below were re-run at refinement time (2026-08-22) against
the current tree; every line reference in this plan reflects that pass, not the ticket's
History:

- `internal/cli/ticket.go:38,85,102` — `--family` still exists, still undocumented in
  `cli-reference.adoc` (its `<<cmd-ticket-new>>` synopsis is now at line 647).
- `internal/cli/project.go:69-76` — `--build`, `--test`, `--lint`, `--docs`,
  `--branch-prefix`, `--wip-dev`, `--wip-review` still exist; `cli-reference.adoc:126`
  (`<<cmd-project>>`) still documents only `--ticket-prefix` and says the rest "can then be
  set in `pickle.toml`", which reads as no flags for them.
- `internal/cli/flow.go` (`pickle flow show|list`) still has no `[#cmd-flow]` section, no
  Overview-table row (table at `cli-reference.adoc:24-79`), and no `flow` key in
  `docs/user-manual/configuration.adoc`'s key list (`configuration.adoc:34-58`).
  `internal/config/config.go:278` confirms `"brine"` is still the only value
  `Config.Validate()` accepts.
- `skill/resources/tickets-README.md:266-267` still cites `pickle ticket renumber`;
  `internal/audit/audit.go:101-103`'s comment still cites it too. `tickets/7-dropped/T-060-*`
  confirms it is dropped, reason: "automation not needed; unity is the only real case and is
  a one-time guided manual migration, not a pickle command."
- `docs/user-manual/cli-reference.adoc:368` still renders the unconstrained-bold bug —
  confirmed by rendering the bullet through `asciidoctor` directly: the `*` in
  `.pi/extensions/*.ts` opens a `<strong>` that swallows through to `*warning*` two lines
  later. A sweep of the rest of the file (`` `[^`]*\*[^`]*` `` ) turns up two more
  backtick+asterisk spans (`` `**bold run**` `` and `` `[A-Z][A-Z0-9]*-\d+` ``); both were
  rendered and neither leaks — only line 368 needs the fix.

### Confirmed design decisions (do not deviate without asking)

1. **No flag/manual diff test.** The ticket's own "shape of the fix" section floats a test
   that diffs registered CLI flags against `cli-reference.adoc`'s prose as the durable fix,
   but prices it explicitly at refinement. Building it means either parsing Go AST for every
   `fs.String/Bool/Int` call or maintaining a hand-written flag inventory kept in sync by
   hand — both are more machinery than a low-complexity, small-cost documentation ticket
   justifies. Skipped; not part of this ticket's scope.
2. **Document new flags beside their siblings, no new sections for flags.** `--family` is
   added to `<<cmd-ticket-new>>`'s synopsis and prose next to `--spawned-by`, whose shape it
   mirrors (shape-checked at creation, existence left to the audit, non-gating lineage-like
   field). The seven `project add` flags are added to `<<cmd-project>>`'s existing bullet,
   next to `--ticket-prefix`, rather than a new subsection.
3. **`pickle flow show|list` gets its own manual section**, `[#cmd-flow]`, inserted between
   `<<cmd-project>>` and `<<cmd-upgrade>>` — matching where `pickle help`'s "Setup commands"
   block places it (`internal/cli/cli.go:101-103`, between `project remove` and `upgrade`).
   It documents the shipped value as the one legal constant (`"brine"`), not as a name a
   project can currently choose.
4. **The `renumber` fix rewords rather than invents new prose.** Both occurrences
   (`tickets-README.md`, `audit.go`'s comment) are changed to state plainly that no such
   command exists and that migration is a manual, one-time step — never a promise that one is
   coming. `tickets-README.md` is shipped payload, so its new wording is written so it stands
   on its own to a foreign reader: no ticket id to look up, no "this repo", nothing that
   assumes pickle's own tree.
5. **The PDF markup fix uses a passthrough, not backslash-escaping.** `` `.pi/extensions/+*+.ts` ``
   (AsciiDoc passthrough inside the existing monospace span) was verified end-to-end with
   `asciidoctor` to render `.pi/extensions/*.ts` literally with no stray `<strong>`; a
   backslash-escaped `\*` also fixes the leak but renders a visible backslash in some
   AsciiDoc backends, so passthrough is the safer choice for a glob shipped in prose.

### Tasks

#### Task 1 — Document `--family` on `pickle ticket new`
In `docs/user-manual/cli-reference.adoc`'s `[#cmd-ticket-new]` section:
- Add `[--family T-NNN]` to the synopsis code block (currently ending
  `[--spawned-by "T-NNN[,T-MMM]"]`).
- Add a sentence after the existing `--spawned-by` sentence, e.g.: "`--family <T-NNN>` sets a
  single umbrella ticket id this one groups under for board ordering (same child only); it is
  shape-checked at creation (`<PREFIX>-NNN`) and, like `--spawned-by`, never gates pickup —
  whether the umbrella exists, is in the same child, and is not itself a family member is left
  to `pickle board audit`."

#### Task 2 — Document the undocumented `project add` flags
In `[#cmd-project]`'s first bullet (`cli-reference.adoc:224-234`), reword "per-child keys
(commands, branch prefix, WIP limits) can then be set in `pickle.toml`" to state they can
also be set directly as `add` flags, and list them next to `--ticket-prefix`:
`--build <cmd>`, `--test <cmd>`, `--lint <cmd>`, `--docs <cmd>` (the child's commands),
`--branch-prefix <prefix>` (default `feat/`), `--wip-dev <n>` / `--wip-review <n>` (default
`1` each). Keep the existing `--ticket-prefix` sentence in place; add the new ones as a
follow-on sentence so the bullet reads as one coherent flag list rather than two.

#### Task 3 — Add a `pickle flow show|list` section
In `docs/user-manual/cli-reference.adoc`:
- Add a row to the Overview table (after the `pickle project add\|list\|remove` row, before
  `pickle upgrade`): `` `pickle flow show\|list` `` / "Print, or list, the configured flow
  name."
- Add a `[#cmd-flow]` section immediately after `[#cmd-project]`'s closing content and before
  `[#cmd-upgrade]`, documenting: `flow show` prints the configured `flow` key from
  `pickle.toml` (default `brine`); `flow list` prints every legal flow name; today that is
  exactly one name, `"brine"` — `Config.Validate()` (`internal/config/config.go`) rejects any
  other value, so document it as the one legal constant rather than an open-ended setting.
  Both take no arguments and both require a valid `pickle.toml` (same pre-flight as other
  read commands).
- In `docs/user-manual/configuration.adoc`'s "The keys" → *Overarching:* bullet list, add a
  `flow` bullet next to `layout`: "`flow` — the flow this project runs (`pickle.toml`'s `flow`
  key); defaults to `brine` when unset, currently the only legal value."

#### Task 4 — Reword the two `pickle ticket renumber` references
- `skill/resources/tickets-README.md:266-267`: replace "**re-homing a ticket to a
  differently-prefixed child is a renumber, not a free relabel** (`pickle ticket renumber`)"
  with wording that states plainly no such command ships and that re-homing is a manual,
  one-time step (rename the file, edit the id in its frontmatter and heading, and fix any
  cross-references) — foreign-workspace-safe: no ticket id for the reader to look up, no
  first-person "this repo", stands on its own.
- `internal/audit/audit.go:101-103`: reword the comment so it no longer implies a future
  command is coming — cite that a manual migration is the only path (a code comment, not
  payload, so it may still name `T-060` as the ticket that considered and dropped automating
  this).

#### Task 5 — Fix the unconstrained-bold glob bug
In `docs/user-manual/cli-reference.adoc:368`, change `` `.pi/extensions/*.ts` `` to
`` `.pi/extensions/+*+.ts` `` (passthrough protects the asterisk from opening bold across the
rest of the bullet). No other glob-in-backticks span in the file needs the same fix (see
Prerequisite gate's sweep note).

### Acceptance test

1. `rg -n 'family' docs/user-manual/cli-reference.adoc` shows `--family` in the
   `<<cmd-ticket-new>>` synopsis and prose.
2. `rg -n 'wip-dev|wip-review|branch-prefix|--build|--test|--lint|--docs' docs/user-manual/cli-reference.adoc`
   shows all seven flags documented under `<<cmd-project>>`.
3. `rg -n 'cmd-flow' docs/user-manual/cli-reference.adoc` shows the new section anchor and its
   Overview-table cross-reference target; `rg -n 'flow' docs/user-manual/configuration.adoc`
   shows the new `flow` key bullet.
4. `rg -n 'renumber' skill/resources/tickets-README.md internal/audit/audit.go` shows neither
   occurrence still reads as "run this command" — both describe it as unavailable/manual.
5. `just docs-check` (runs `snowball check`) passes — no broken includes/xrefs from the new
   `<<cmd-flow>>` anchor or table row.
6. Render the fixed bullet and confirm the glob survives intact and bold does not leak:
   `asciidoctor -o - docs/user-manual/cli-reference.adoc 2>/dev/null | grep -A1 'pi/extensions'`
   shows `<code>.pi/extensions/*.ts</code>` with no `<strong>` opened inside it.
7. `just build && just test && just lint` all green (no non-doc files changed except the
   `audit.go` comment, so this is mostly a regression check).

### Docs update (mandatory when user-facing)

This ticket *is* the docs update: `docs/user-manual/cli-reference.adoc` (four spots) and
`docs/user-manual/configuration.adoc` (one spot) gain the missing flag/command coverage and
lose the markup bug; `skill/resources/tickets-README.md` (shipped payload) loses the stale
`renumber` claim. No other doc needs a new pointer — all edits land in files that already
cover their respective commands.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated as above; no separate registration needed.
3. Write a summary: files touched, decisions made (esp. the skipped flag/manual diff test),
   anything deferred.
4. Suggest commit message: `docs(cli-reference): document --family, project-add flags, pickle flow, and drop the renumber claim (T-066)`.
   Tidy WIP commits into atomic ones before presenting (root-path child, rules §0); do not
   push or open an MR without explicit user approval.
5. `pickle ticket move T-066 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-05 — created (TO DO). source: pickle ticket new; spawned by the T-019 review, batching its non-blocking findings N3 (flags shipped but undocumented in cli-reference.adoc) and N5 (`pickle ticket renumber` cited in the skill payload though T-060 was dropped)
- 2026-08-05 — patched by the T-022 review's impact sweep: the renumber claim's line reference re-verified (122-123 → 127-128) and the T-022 coupling corrected — the two tickets do share `skill/resources/tickets-README.md`, and `configuration.adoc`'s `ticket_prefix` bullet is already done
- 2026-08-06 — patched by the T-057 review (finding N10, disposition `folded`): scope gained item 3,
  the unconstrained-bold swallow at the `.pi/extensions/*.ts` glob that mangles the doctor bullet in
  the rendered PDF/EPUB. T-057 added a `[#cmd-hooks]` section to this file (self-contained, per its
  own gate finding F10) plus an Overview-table row — re-verify line references at refinement
- 2026-08-07 — patched by T-073's review impact sweep (step 8): absorbed T-073's finding **F4**
  (disposition `folded`) — `pickle flow show|list` shipped with no manual coverage at all, now
  listed as a third bullet under item 1. T-073 also renamed prose across six `.adoc` files and
  added a `:flow:` attribute to `docs/attributes.adoc`, so **every line reference in this ticket
  must be re-verified at refinement** rather than trusted
- 2026-08-13 — patched by **T-074's review impact sweep**. T-074 added 29 lines to
  `docs/user-manual/cli-reference.adoc`, all below ~`:245`: an `[IMPORTANT]` block under
  `upgrade` describing the legacy sweep, a `doctor` bullet for the new legacy-path error, and a
  sentence under `uninstall`. Consequences for this ticket's anchors: the
  `<<cmd-ticket-new>>` synopsis citation `cli-reference.adoc:277` is stale (it had already
  drifted before T-074 — the synopsis now sits near `:438` pre-merge, `:467` post-merge), while
  `cli-reference.adoc:140-142` (the `project add|list|remove` flag gap) is **above** the
  insertion point and still correct. Neither documentation gap this ticket exists to close was
  filled by T-074 — it documented only its own new behaviour — so the scope is unchanged.
  Re-anchor by searching the heading text at refinement rather than trusting either number
- 2026-08-22 — refined (READY). Re-verified every anchor against the current tree (all four
  gap sites still open, none closed by an intervening ticket) and wrote the Implementation
  Plan: document `--family` on `<<cmd-ticket-new>>` and the seven `project add` flags on
  `<<cmd-project>>`, add a new `[#cmd-flow]` section plus a `configuration.adoc` `flow` key
  bullet, reword both `pickle ticket renumber` references (`tickets-README.md`, `audit.go`'s
  comment) to state plainly that no such command ships, and fix the confirmed
  unconstrained-bold glob bug at `cli-reference.adoc:368` with a passthrough. Declined to build
  the flag/manual diff test the Description's "shape of the fix" floated — priced at
  refinement, it is more machinery than this ticket's scope justifies (recorded as Confirmed
  design decision 1)
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — READY → IN DEVELOPMENT: picked up
