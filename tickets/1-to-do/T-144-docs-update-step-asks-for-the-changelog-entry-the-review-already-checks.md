---
id: T-144
title: Docs update step asks for the changelog entry the review already checks
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-144 — Docs update step asks for the changelog entry the review already checks

## Outcome

After this ships, a ticket's `### Docs update` step asks for the changelog entry when the project
keeps one. An implementer who follows the plan as written ships that entry, and the review no
longer has to catch a missing one.

## Description

The review protocol's `docs-gap` class covers `CHANGELOG.md` (`skill/resources/review-protocol.md`
§5, the class table). The planning side never mentions a changelog: neither the template's
`### Docs update (mandatory when user-facing)` placeholder (`skill/resources/TEMPLATE.md`) nor
rules §4 item 6 (`skill/resources/tickets-README.md`) does. Only the review checks for an entry
the plan never asked for.

**Evidence (Claude Code sessions, 2026-09-29 to 2026-10-28).** Eight tickets were refined,
implemented and reviewed: T-134, T-135, T-137, T-138, T-140, T-141, T-142 and T-143. Seven of the
eight plans listed a `CHANGELOG.md` `[Unreleased]` entry, but only because earlier tickets in
this repo did. T-143's plan left it out. Its review raised F4 (round 1, recorded as F13), class
`docs-gap`, "`[Unreleased]` had no entry", and justified the finding by precedent:
"Previous tickets (e.g. T-142) added theirs in the ticket diff". It was non-blocking and fixed
inline. A project without that precedent gets no such cue, so the gap will open more often there.

**Scope.** Name the changelog entry in the docs step, conditionally ("if the project keeps a
changelog"), in the template placeholder and in rules §4 item 6. Optionally mention it in the
template's Finish step 2 as well. Prose only, no Go change. The text ships in the payload, so it
must pass the foreign-workspace test and `payload_lint_test.go`.

**Considered and not included, for lack of evidence in the same sessions:**
- Defining "registered" in "Docs updated and registered". None of the eight plans used the word,
  and no finding concerned registration. The review's coverage check ("registered/linked")
  already reads clearly enough.
- Retitling `### Docs update (mandatory when user-facing)`. All eight plans kept the heading with
  a body, and no session hit a READY-gate refusal for it. Renaming it would also churn every
  existing ticket's heading match for no observed gain.

Soft coupling: T-093 (reconcile merged tickets against `[Unreleased]`) checks after the merge
for the same gap this ticket prevents at plan time. The two do not overlap.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-10-28 — created (TO DO). source: self-host: a review of brine's docs step, checked against a month of Claude Code sessions; the template's docs step omits the changelog the review checks for (T-143 F13)
