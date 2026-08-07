---
id: T-080
title: lifecycle as data: extract states, transitions, and terminal/WIP flags into a flow definition
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: high
cost: L
---

# T-080 — lifecycle as data: extract states, transitions, and terminal/WIP flags into a flow definition

## Description

brine's lifecycle is currently spelled out as Go literals in five packages. The seven status
directories, their board headings and terminal flags live in `internal/ticket/ticket.go`; the
legal transition graph and the definition of a "backwards" move live in `internal/move/move.go`;
and the strings `3-in-development` / `4-in-review` are then re-typed in `internal/board/board.go`
(WIP tallying and the rendered limit line), `internal/audit/audit.go` (WIP errors, the
dependency-must-be-done check) and `internal/install/install.go` (the marker block's WIP
bullet). Adding a status, renaming one, or changing a WIP rule means finding all of them.

This ticket extracts that into **one declarative flow definition**, with brine as the shipped
default: states (dir, heading, terminal), legal transitions, which moves are backwards, which
states are WIP-limited, and which state a dependency must reach before a dependent can be
picked up.

**It is worth doing at N=1.** The immediate payoff is the duplication: one source for a set
of strings currently maintained by hand in five places, and a real answer to the most common
real-world request a flow tool gets — *"add a QA column"*, *"rename our statuses"* — which
today is a code change. The second payoff is that it is the precondition for there ever being
a second flow (see T-073): with this and T-081 in place, a `brine-v` is a definition file plus
a prose addendum rather than an engine project.

**The trap to name up front:** `board audit`'s teeth. Several checks are only meaningful
because they know brine's semantics — "in development but dependency is not in `6-done`",
"child is over its `3-in-development` limit". Generalising must keep those checks sharp by
parameterising them from the definition, not by softening them into advisories. If the
refined plan cannot state which checks survive and how, the extraction is not ready.

### Overlap with T-042 — settle before either goes READY

T-042 (collapse duplicated internal predicates into single helpers: status headings, marker
span, test payload root) covers part of the same duplication from the other direction. They
must not both be implemented. Refinement decides one of: T-080 absorbs T-042's status-heading
portion and T-042 narrows to the marker-span and test-payload-root helpers; or T-042 lands
first as the mechanical de-duplication and T-080 builds on it. Doing neither means writing the
same refactor twice.

Soft couplings: T-073 (the `flow` config key) is the seam this definition would eventually be
selected by; T-081 is the natural follow-on and depends on this; the dropped T-015
(consolidate board status-heading matching) is prior art on the same duplication.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — patched by T-073's review impact sweep (step 8): the seam this ticket is written
  against now **exists**. T-073 shipped `flow` in `pickle.toml` with
  `config.DefaultFlowName = "brine"`, a `Config.FlowName()` accessor, and a `Validate()` that
  **rejects any value other than `"brine"`** (`internal/config/config.go`). That last part is the
  load-bearing detail for this ticket: introducing a second flow definition means relaxing that
  check, so plan it as an edit to `Validate()` rather than assuming the key already accepts an
  arbitrary name. `pickle flow list` also exists and prints exactly one entry today — it is the
  natural place to enumerate definitions once they are data. No assumption is invalidated
