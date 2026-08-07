---
id: T-081
title: gate table as data: per-state required-artifact preconditions, audited
project: pickle
depends-on: [T-080]
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-081 — gate table as data: per-state required-artifact preconditions, audited

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

Two things this must decide at refinement:

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

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
