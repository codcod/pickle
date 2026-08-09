---
id: T-089
title: Record a commit reference alongside the merge History line
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S-M
---

# T-089 — Record a commit reference alongside the merge History line

## Outcome

After this ships, anyone reading a done ticket's `## History` can go straight from its merge
line to the commit that actually shipped the change: the existing `merged to <base> (<MR
ref>)` line also carries a commit reference (short SHA, and — where the child's remote
resolves to a known hosting URL — a link to it), instead of the ticket needing a separate
section or a copy of anything.

## Description

Origin: a chat request to have every ticket carry (a) a commit id/link for its changes and
(b) a copy of the line that landed in `CHANGELOG.md`. Both were challenged before filing;
only a scoped-down version of (a) survives here.

**Why (b) is dropped, not deferred.** Copying a `CHANGELOG.md` line into the ticket creates a
second copy of a fact that already drifts on its own: T-073's review (finding F1) records that
several entries (T-058/T-059/T-061) were written by a human at release-cut time, well after
those tickets were archived in `6-done/`, and plenty of tickets ship with no `CHANGELOG.md`
line at all (T-019, and the internal-fix precedent discussed in T-043's rework). A `6-done/`
ticket is never revisited, so a pasted-in copy has no mechanism keeping it in sync with the one
authoritative line in `CHANGELOG.md` — precisely the failure mode the flow already guards
against elsewhere by making `BOARD.md` generated rather than hand-duplicated. Every
`CHANGELOG.md` bullet already ends in `(T-NNN)`, so the ticket → changelog lookup already works
in the direction that can't drift (`grep T-NNN CHANGELOG.md`); nothing here needs to change.

**Why (a) survives, scoped to the existing merge line.** The flow already has a place this
fact belongs: the `merged to <base> (<MR ref>)` History line (TEMPLATE.md's `## History`
section; `internal/ticket.MergeLine`), written once, by the human, at the one point a merge is
actually final. `historyKind`/`MergeLine` already treat the whole line as free text — nothing
parses the parenthetical today beyond the line starting with "merged" — so recommending a
commit reference alongside the MR ref (e.g. `merged to main (MR !12, a1b2c3d)`) is a convention
change to `tickets/README.md`'s history-line rules, `TEMPLATE.md`'s worked example, and
`review-protocol.md`'s finish-step guidance, not a new ticket section or a new parser.

**Open for refinement:** whether `pickle serve`'s board "merged" column / activity view should
linkify a URL found in this line (the "preferably a link" half of the original ask), or whether
a short SHA plus whatever the human pastes is enough — the flow is deliberately generic across
hosting platforms and today has no notion of a remote's web-URL format. Pin this down during
refinement rather than here.

Soft coupling: T-070 (routes `MergeLine` through `HistoryEntries`) touches the same parsing
path. No hard dependency, but read it first so the two changes don't collide.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-09 — created (TO DO). source: chat — commit-id/changelog-line idea challenged;
  scoped down to enriching the merge History line only, CHANGELOG-copy half dropped
