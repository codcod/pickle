---
id: T-094
title: make changelog check usable outside the post-release moment: a range end, a quieter exclusion list, and subjects a squash-merge rewrote
project: pickle
depends-on: []
spawned-by: [T-093]
impact: low-medium
complexity: low
cost: S
---

# T-094 — make changelog check usable outside the post-release moment: a range end, a quieter exclusion list, and subjects a squash-merge rewrote

## Outcome

After this ships, `pickle changelog check` is readable at the moment it is actually run: its
report is candidates first and exclusions on request, it can be aimed at a closed commit range
rather than always running to `HEAD`, and it stops silently missing tickets in projects whose
merge button rewrites the commit subject.

## Description

Three findings from T-093's review (F3, F4, F5), batched because they are one theme: the command
was designed for exactly one moment — standing on `main` just before cutting a release — and is
awkward or wrong anywhere else. None of them is a defect against T-093's confirmed decisions;
all three are the edges those decisions left.

**1. No range end (`--until`), so the shipped set always runs to `HEAD` (F3).** Checking a
*past* section is therefore only meaningful in the seconds after its tag. T-093's own acceptance
test is the proof: `changelog check --since v0.4.0 --section 0.5.0` was specified to yield
exactly one candidate (`T-090`), and it yields two — `T-090` plus `T-093` itself, because the
feature branch's own four commits sit in `v0.4.0..HEAD`. The classifier is right; the range is
just open-ended. A `--until <ref>` (default `HEAD`) makes `<since>..<until>` a closed question
and makes any historical section auditable.

**2. The exclusion list is unconditional and unbounded (F4).** Decision 7 requires the excluded
`board:` commits to be visible so a convention drift shows up rather than under-reporting
silently — but it explicitly permitted "or offer behind a flag". Printing every one, always,
inverts the report: on this repo `--since v0.4.0 --section 0.5.0` prints 2 candidates and 36
exclusion lines, and a release with thirty tickets will print well over a hundred. The signal
survives with a count plus the ids (`excluded 36 board: bookkeeping commit(s) covering T-080,
T-081, …`), with the full subjects behind `--show-excluded`. Keep decision 7's intent: the reader
must still be able to see *what* was excluded without re-running anything they cannot discover.

**3. A squash-merge's rewritten subject is invisible (F5).** `ClassifySubject` requires the
ticket id in brackets at the very end of the subject, which is exactly what T-084's convention
prescribes — but GitHub's squash button appends ` (#N)`, turning
`feat(cli): add a thing (T-050)` into `feat(cli): add a thing (T-050) (#31)`. That classifies as
`Neither`: not shipped, and not listed as an exclusion either, so the ticket vanishes from the
report with nothing to notice. This repo merges rather than squashes, so it does not bite here —
but `changelog check` ships to every installed project, and the squash button is the common case
elsewhere. Tolerating a trailing PR-number token (and/or reporting a `Neither` subject that
contains a ticket id anywhere as an "unclassified" line) closes the one silent-under-report path
the design otherwise defends against everywhere else.

Soft coupling: **T-093** (done) shipped the command; its Description records why each of the
three edges was left where it is, and its `## Review` carries the evidence for all three
findings. Nothing here reopens a T-093 confirmed decision: the check stays read-only and
advisory (decision 2), one-directional (decision 4), and free of any exemption mechanism
(decision 5).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: T-093's review, findings F3/F4/F5, batched by theme per
  the rules §5 (see `tickets/6-done/T-093-…md`'s `## Review`). Graded `low-medium`/`low`/`S`
  against the backlog: all three are small, local changes to one young command, and the
  usability half (F4) is felt at every release
