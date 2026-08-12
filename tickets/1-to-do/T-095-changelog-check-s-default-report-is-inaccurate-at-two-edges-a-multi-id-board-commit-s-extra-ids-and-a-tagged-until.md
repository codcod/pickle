---
id: T-095
title: changelog check's default report is inaccurate at two edges: a multi-id board: commit's extra ids and a tagged --until
project: pickle
depends-on: []
spawned-by: [T-094]
impact: low
complexity: low
cost: S
---

# T-095 — changelog check's default report is inaccurate at two edges: a multi-id board: commit's extra ids and a tagged --until

## Outcome

After this ships, `pickle changelog check`'s default one-line exclusion summary names *every*
ticket id the excluded bookkeeping commits cover, not just the first id of each; and the bare
`changelog check` invocation still answers sensibly when `HEAD` is itself tagged or has no
parent, instead of reporting the previous release against `[Unreleased]` or exiting `1`.

## Description

Two non-blocking findings from T-094's review (F1, F2), batched because they are one theme: the
two *defaults* T-094 introduced are each inaccurate at an edge. Neither is a deviation from a
T-094 confirmed decision — both are edges those decisions left — and neither affects the golden
path (`changelog check` run on an untagged `main` before cutting a release, which is what
`RELEASING.md` prescribes).

**1. The exclusion summary drops a multi-id `board:` commit's extra ids (F1).** T-094 decision 4
made the one-line summary the *only* default view of the excluded set, and specified it as "the
deduplicated ids sorted". But the id set is built from `Exclusion.ID`, and
`boardIDRE` (`^board:\s*([A-Z][A-Z0-9]*-\d+)\b`, pre-existing from T-093) captures exactly one
id — while the rules §0 grammar explicitly sanctions the multi-id form
(`board: T-057, T-072 note the shared origin-base invariant`). Measured in a scratch repo: a
single commit `board: T-010, T-011 re-aimed after review` yields
`excluded 1 board: bookkeeping commit(s) covering T-010` — `T-011` appears only under
`--show-excluded`. Before T-094 every subject printed, so both ids were visible; the summary is
where the second id became invisible. The count is always exact, so this degrades the id
inventory, not the drift alarm. Likely fix: build the summary's id set with
`FindAllStringSubmatch` over the subject's remainder after `board:` (or give `Exclusion` an
`IDs []string`), and pin it with a `Check`/report test carrying a two-id bookkeeping subject.

**2. The `<until>^` default `--since` misbehaves on a tagged or parentless `HEAD` (F2).** T-094
decision 3 resolves the default `--since` as `git describe --tags --abbrev=0 <until>^`, and
scoped its "unchanged from today" guarantee to *an untagged `HEAD`*. On a tagged `HEAD` the
behaviour therefore did change, in two ways nothing documents or tests:

- `HEAD` tagged `v0.2.0` with an earlier `v0.1.0`: bare `changelog check` now reports
  `v0.1.0..HEAD` — i.e. the whole *previous* release — against `[Unreleased]`. Pre-T-094 it
  resolved `v0.2.0..HEAD`, an empty range printing `no candidates`. Neither answer is useful, but
  the new one is loud, and it lands exactly one command after `git tag` in `RELEASING.md`.
- `HEAD` is the root commit (a fresh or shallow clone): `HEAD^` does not resolve, so a previously
  exit-`0` invocation now exits `1` with
  `no --since given and no git tag found reachable from HEAD^: … exit status 128`. The message
  blames a missing tag when the real cause is a missing parent.

Likely fix: fall back to `describe --tags --abbrev=0 <until>` when `<until>^` cannot be resolved
or described, and say in `docs/user-manual/cli-reference.adoc` what a tagged `--until`/`HEAD`
means; pin both with `internal/cli/changelog_test.go` cases.

Soft coupling: **T-093** (done) shipped the command and `boardIDRE`; **T-094** (done) shipped
both defaults, and its `## Review` carries the measured evidence for F1 and F2. Nothing here
reopens a T-093 or T-094 confirmed decision: the check stays read-only and advisory, one
directional, and free of any exemption mechanism.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: T-094's review, findings F1/F2, batched by theme per the
  rules §5 (see `tickets/6-done/T-094-…md`'s `## Review`). Graded `low`/`low`/`S` against the
  backlog: both are a few lines in one young command, both are off the golden path
  (`RELEASING.md` runs the check on an untagged `main`), and both already have their measured
  reproduction recorded — lower impact than T-094 itself, which fixed the edges people actually
  hit
