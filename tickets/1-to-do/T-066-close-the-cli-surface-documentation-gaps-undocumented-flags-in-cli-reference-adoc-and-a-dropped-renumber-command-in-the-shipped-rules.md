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

## Description

Spawned by the **T-019** review (findings N3 and N5), batched by theme: **shipped text that
describes the CLI surface has drifted from the CLI**. T-019 pruned `DESIGN.md` §4–§7 down to a
pointer — *"the live command surface, with every flag, is `docs/user-manual/cli-reference.adoc`"*
— which makes the manual's completeness load-bearing in a way it was not before. It is not
complete today.

### 1. Flags the CLI has and `docs/user-manual/cli-reference.adoc` never mentions

Swept by comparing every `fs.String/Bool` flag in `internal/cli/*.go` against the manual:

- **`pickle ticket new --family <ID>`** (`internal/cli/ticket.go:85`, shipped by T-059) — absent
  from the manual entirely: not in the `<<cmd-ticket-new>>` synopsis (`cli-reference.adoc:277`),
  not in its prose, not anywhere under `docs/` (`rg -n family docs/` returns nothing). The
  command's own `usage:` string (`ticket.go:36`) *does* list it, so `pickle help` and the manual
  disagree.
- **`pickle project add --branch-prefix`, `--wip-dev`, `--wip-review`, `--build`, `--test`,
  `--lint`, `--docs`** (`internal/cli/project.go`) — the manual documents only
  `--ticket-prefix` and then says per-child keys *"can then be set in `pickle.toml`"*
  (`cli-reference.adoc:140-142`), which reads as *there are no flags for these*. There are.

### 2. A dropped command cited as the mechanism, in the payload every project installs

`skill/resources/tickets-README.md:122-123` states that re-homing a ticket to a
differently-prefixed child *"is a renumber, not a free relabel (`pickle ticket renumber`)"*, and
`internal/audit/audit.go:101` comments that a mismatched prefix *"goes red here until `pickle
ticket renumber` (T-060) runs"*. **T-060 is in `7-dropped/`** — dropped 2026-07-28 with the
reason *"automation not needed; unity is the only real case and is a one-time guided manual
migration, not a pickle command"*. So the shipped rules point a user at a command that does not
exist and is not planned. This is the T-041/T-022 defect class (stale text that still reads as
authoritative), and it ships inside the skill payload, so every installed project carries it.

T-019 fixed the same claim where *it* had introduced it (`DESIGN.md` §3 decision 2, disposition
`fixed inline`); these two occurrences are pre-existing and outside a docs-only ticket's scope,
which is why they are here.

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
  marker block). No overlap in the files touched.
- **T-058** / **T-060** — the `ticket_prefix` schema and its dropped migration command.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-05 — created (TO DO). source: pickle ticket new; spawned by the T-019 review, batching its non-blocking findings N3 (flags shipped but undocumented in cli-reference.adoc) and N5 (`pickle ticket renumber` cited in the skill payload though T-060 was dropped)
