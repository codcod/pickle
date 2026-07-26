---
id: T-045
title: backlog cap and user-visible axis: decide after measuring whether the T-036 disposition valves lowered the spawn rate
project: pickle
depends-on: []
spawned-by: [T-036]
impact: low-medium
complexity: medium
cost: M
---

# T-045 — backlog cap and user-visible axis: decide after measuring whether the T-036 disposition valves lowered the spawn rate

## Description

Split out of **T-036** at refinement (2026-07-26). T-036 was filed with four valves against an
unbounded spawn rate; two are payload text and ship there. These two are CLI + schema, and both
are **backstops for a leak T-036 plugs** — so this ticket exists to decide whether they are
still needed once T-036 has run for a while, not to build them on the assumption that they are.

**Do not refine this ticket until T-036 has landed and a spawn rate has been re-measured over
at least three reviews.** If R has fallen well below 1, the honest outcome is to drop this
ticket. Re-measuring is cheap: the disposition column in each `## Review` findings table (which
T-036 makes mandatory) is the data.

### Valve 3 — per-child TO DO backlog cap

Configurable in `pickle.toml`, enforced by `pickle board audit` and `pickle ticket new`; at cap,
filing requires dropping something. The original argument: WIP limits bound
`3-in-development/` and `4-in-review/` but leave `1-to-do/` unbounded, so the pressure escapes
where nobody feels it.

Problems found while refining T-036, which any plan here must answer:

- **Severity is load-bearing and error severity is unusable.** `audit.Audit` runs as a post-op
  self-check inside `ticket move` (`move.go:138`), `board sync` (`sync.go:114`), and `install`
  / `upgrade` (`cli/install.go:70,127`). An `errf` cap breach therefore makes all four commands
  exit non-zero — including the moves that *drain* the backlog. Either the check is `warnf`
  (`audit.go:206`), or those call sites need exempting.
- **It red-lines on day one.** This repo has 14 tickets in `1-to-do/`. Any plausible cap ships
  already-breached for the only project using the flow.
- **`ticket new` cannot currently count per child.** It never loads tickets — `NextNum`
  (`ticket.go:269`) scans filenames only, deliberately, to stay robust to unparseable
  frontmatter. A per-child TO DO count needs `ticket.LoadAll`, which also returns issues;
  decide whether `ticket new` inherits `move`'s hard refusal on load problems
  (`move.go:62-65`) or counts leniently. The rejection must land **before** the first write
  (`cli/ticket.go:127`) to preserve the "a rejected invocation writes nothing" invariant pinned
  by `cli_test.go:246,287`.
- Config plumbing is six stations, two of them duplicated: `config.go:34-38` (default),
  `:74-75` (struct), `:142-147` **and** `:202-207` (defaults, `AddProject` bypasses
  `applyDefaults`), `:167-169` (validate), `:251-252` (render). "0 = unlimited" needs the
  `md.IsDefined` idiom already used for the commit booleans at `:125-136`.
- Surfaces: `--todo-cap` flag and `project list` column (`cli/project.go:69-70,87-91,108-111`),
  the `AGENTS.md` marker block (`install.go:560-562`, golden-pinned by
  `testdata/markerblock.golden`), and optionally `### <child> (n/cap)` under TO DO
  (`sync.go:220-225`).

**Alternative worth costing against it:** a **spawn-rate warning** instead of a cap — have
`board audit` warn when spawns-per-completion exceeds a threshold. It measures the disease
rather than a symptom, and cannot deadlock the commands that relieve it. It requires
`spawned-by:` to be populated, which it is not: the field is `[]` on every ticket except T-038
and the T-039…T-043 epics, and the backfill ticket (**T-025**) was dropped. Resurrect T-025
first if this route is taken.

### Valve 4 — a `user-visible:` grade axis

A boolean frontmatter axis so the board can sort hygiene below user-visible work. The original
argument: `impact` collapsed as a signal — no `high` for 17 consecutive tickets (T-019…T-035).

**The premise is weaker than it looked.** The drought ended the moment anyone graded
deliberately: the 2026-07-26 triage regraded T-026 `medium`→`high`, and T-036/T-037/T-039 were
all filed `high`. And the existing definition (rules §3, `tickets-README.md:118-130`) is
*already* user-facing — "major capability/adoption lever" vs "narrow/cosmetic". That points at
misapplication of a fourth axis rather than a missing fifth one.

**Cheapest alternative, and the recommended starting position: recalibrate instead of adding.**
Tighten §3's impact definitions with explicit worked examples ("hygiene no user can observe is
at most `low`, however real the defect"). Zero schema, zero migration, zero board change.

If a real key is still wanted, the costs are:

- **Migration break.** Adding `user-visible` to `requiredKeys` (`audit.go:23`) fails every
  pre-existing ticket — 43 here, plus every installed project, since `pickle upgrade` never
  touches tickets. This is exactly the `spawned-by:` pain already documented at
  `README.md:149-152`. Optional-but-validated-when-present avoids it.
- **Do not extend `ValidGrade`'s caller loop.** `audit.go:55` iterates
  `impact/complexity/cost`, and `ValidGrade` returns `false` for an unknown kind
  (`ticket.go:258-264`) — adding the key to that list fails every ticket. A boolean needs its
  own check and its own message; "adjacent-pair ranges" is nonsense for a bool.
- **`Scaffold` is positional with six strings** (`ticket.go:331`); a seventh argues for the
  params-struct refactor already recorded as T-024 finding N7 (now in **T-042**).
- **TEMPLATE.md drift.** `Scaffold` never reads `TEMPLATE.md`; the only guard compares `## `
  headings (`ticket_test.go:256`), not frontmatter. A frontmatter-superset test is owned by
  **T-040**.
- **A board column depends on T-044 (was T-039, dropped as superseded 2026-07-26).**
  T-044 makes the board fully generated, so a new column becomes a render-side change and the
  fragility below disappears with `insertIntoBoard`. Until it lands, the old mechanics apply:
  `insertIntoBoard` reads impact as a fixed `cells[3]` of
  a plain `|` split (`board.go:282-288`); inserting a column before `impact` silently breaks
  impact-ordered insertion with nothing failing loudly. Column work also touches `board.go:120`,
  `:79-113`, `AddTODORow`'s signature `:236` (5 call sites), `sync.go:272-300`,
  `move.go:147-166`, and the skeleton at `skill/resources/BOARD.md:48-56` — and existing boards
  only pick it up via `board sync`. Two independent sort implementations (`board.go:16-20` and
  `sync.go:49-52`) must be kept in agreement or `sync` reports drift forever.

Soft couplings: **T-044** (generated board — prerequisite in practice for any new column;
superseded T-039), **T-040** (frontmatter validation, TEMPLATE drift test), **T-042** (`Scaffold` params
struct), **T-025** (dropped; needed for a spawn-rate metric), **T-044** (if BOARD.md becomes a
generated artifact, the column-mechanics cost above largely evaporates — re-read that ticket
before planning any board change here).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: split out of T-036 at refinement — valves 3 (backlog
  cap) and 4 (`user-visible:` axis) are CLI+schema backstops for the leak T-036's text valves
  plug, so their justification is to be re-measured after T-036 lands rather than assumed now.
  Both carry unresolved design problems recorded in the Description.
- 2026-07-26 — **measurement datapoint 1 recorded** (patched by T-036's review impact sweep, not a
  status change). T-036's own review ran under the new §5 — the repo self-hosts the payload — and
  produced **11 non-blocking findings → 0 new tickets** (5 fixed inline, 2 folded into T-044,
  4 noted). Under the old §5 all 11 were obliged to become `1-to-do/` tickets. Two more reviews
  are needed before this ticket may be refined, per the gate above. Caveat carried over from
  T-036's own "honest reading": a retroactive replay cannot measure a spawn rate, only real
  reviews can, and one review is not a rate either. The disposition columns are now the data.
