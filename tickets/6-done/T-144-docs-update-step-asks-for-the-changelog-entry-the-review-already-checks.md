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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-144-docs-step-changelog
```

Work on this branch, committing locally (WIP commits encouraged). Never push or open a merge
request without explicit user approval. This is a root-path child (`path = "."`), so tidy the WIP
commits into atomic ones before the Finish summary, and keep that history on merge. Before
pushing, `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`
must print nothing.

### Prerequisite gate (hard)

None. T-145 (in `2-ready/`) edits `review-protocol.md` and `cli-reference.adoc`, not the files
here; both add a `CHANGELOG.md` `[Unreleased]` entry, so whichever merges second resolves a
trivial conflict there.

### Confirmed design decisions (do not deviate without asking)

1. **The changelog is named conditionally.** The wording is "if the project keeps a changelog"
   (or equivalent), never an unconditional `CHANGELOG.md`: a foreign project may keep none, or
   name it differently.
2. **Template placeholder and rules §4 item 6 only; Finish step 2 is left as is.** "Docs updated
   and registered" already points back at the docs step, which now names the entry. Adding it
   there too would restate the same instruction twice.
3. **No heading rename, no definition of "registered".** Per the Description's "considered and
   not included" list.
4. **Prose only.** No Go change, no change to the READY-gate check.

### Tasks

#### Task 1 — `skill/resources/TEMPLATE.md`, the `### Docs update` placeholder

Add one sentence to the placeholder body: if the project keeps a changelog, name the entry this
ticket adds to it (e.g. its unreleased section). Keep the existing "no user-facing surface"
sentence and the file's wrap width.

#### Task 2 — `skill/resources/tickets-README.md`, rules §4 item 6

Extend the item: "which docs to add/update (including the changelog entry, if the project keeps
a changelog), or "no user-facing surface"."

#### Task 3 — `docs/user-manual/concepts/lifecycle.adoc`, the READY-gate list item 6

Mirror Task 2's wording so the user manual matches the rules.

#### Task 4 — `CHANGELOG.md`

`## [Unreleased]` → `### Changed`: one bullet ending `(T-144)`, saying a ticket's docs step now
asks for the changelog entry when the project keeps one.

### Acceptance test

```
just build && just test && just lint && just docs-check          # all green; just test runs payload_lint_test.go
grep -n -i "changelog" skill/resources/TEMPLATE.md               # expect a hit inside the Docs update placeholder
grep -n -A1 '^6\. \*\*Docs update' skill/resources/tickets-README.md | grep -i changelog   # expect a hit
grep -n -i "changelog" docs/user-manual/concepts/lifecycle.adoc  # expect a hit on item 6
grep -n "(T-144)" CHANGELOG.md                                   # expect one bullet under [Unreleased]
```

Foreign-workspace check (by reading): the new sentences name no ticket id, no pickle-only path,
and make no claim drawn from this repo's history.

### Docs update (mandatory when user-facing)

`docs/user-manual/concepts/lifecycle.adoc` (Task 3) and the `CHANGELOG.md` `[Unreleased]` entry
(Task 4). The payload edits are themselves the docs change.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (Tasks 3 and 4).
3. Summary of files touched and anything deferred.
4. Suggested commit: `docs(skill): ask for the changelog entry in the docs step (T-144)`.
5. `pickle ticket move T-144 in-review --reason "acceptance green"`.

## Review

### Round 1 — 2026-10-28

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | design | noted | The READY-gate hint restates rules §4.6 without the new changelog clause. It still points at §4.6, which carries the full text, so it does not mislead. Decision 4 keeps this ticket prose-only. | `internal/flow/brine.go:64-65` | Pick it up if a later Go-touching ticket revisits the hint text. |
| F2 | non-blocking | design | noted | The `docs-gap` class says "(includes `CHANGELOG.md`)" unconditionally, while the plan side now says "if the project keeps a changelog". This was there before this ticket and is outside its file list. | `skill/resources/review-protocol.md:358` | Reword it as "includes the changelog entry, if the project keeps one" the next time that table is edited. |
| F3 | non-blocking | spec-unclear | noted | The changelog clause sits inside the "which docs" branch, which implies a ticket with "no user-facing surface" gets no entry. That matches Keep a Changelog, and `pickle changelog check`'s docs already leave a deliberate omission to the reader. | `TEMPLATE.md:110-113`, `tickets-README.md:363-364` | Revisit only if reviews start disagreeing about entries for internal-only tickets. |

Dispositions: 0 blocking; fixed inline 0; noted 3 (F1–F3); new ticket 0; folded 0.
cost: estimated S, actual S

- [x] Reviewer independence settled (step 0): delegated. A fresh sub-agent ran steps 1–4a with `hunt: one pass`. All 3 of its findings were kept after re-verification by hand; none were discarded. Its class `inconsistency` is not in the vocabulary, so F1 and F2 were re-classed `design`.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned that the branch had the ticket in `3-in-development`; the branch was rebased onto `main`, and doctor then reported 0 warnings.
- [x] Implementation audit (steps 1, 2): acceptance test re-run verbatim and green (`just build && just test && just lint && just docs-check`, all four greps hit); Tasks 1–4 and decisions 1–4 verified; foreign-workspace check passed by reading.
- [x] Correctness hunt (step 3): one-pass angle hunt run by the delegated reviewer, with no host tool and no sub-agents. The diff is prose-only, and the heading and frontmatter are unchanged, so existing tickets still parse. No correctness findings.
- [x] Consistency audit (step 4): repo-wide grep for restatements of the docs step. F1 and F2 came from it; `SKILL.md`, `your-first-project.adoc:76` and `cli-reference.adoc:1083` only name the step and stay accurate.
- [x] Documentation audit (step 4a): the change is mirrored in `lifecycle.adoc`, has its `CHANGELOG.md` entry, and `just docs-check` is green.
- [x] Docs-readability pass (step 4b): skipped. No pi tool or opencode subagent is configured here (`opencode` is not on PATH). 0 suggestions discarded as fabricated.
- [x] Findings recorded with severity, class and disposition (step 5).
- [x] Ticket moved to `tickets/6-done/` (step 6).
- [x] Other references (step 7): `AGENTS.md`/`CLAUDE.md` do not restate the docs step, so no governing document needed reconciling. The board was regenerated by the move.
- [x] Impact sweep (step 8): no ticket depends on T-144. T-145's prerequisite note about the shared `CHANGELOG.md` conflict is still accurate.
- [x] Summary + commit message & MR attributes presented and approved (step 9). `main` was pushed first, the `origin/main...HEAD` tickets/ check came back clean, then the branch was pushed and PR #103 opened.

## History

- 2026-10-28 — created (TO DO). source: self-host: a review of brine's docs step, checked against a month of Claude Code sessions; the template's docs step omits the changelog the review checks for (T-143 F13)
- 2026-10-28 — TO DO → READY: plan complete
- 2026-10-28 — READY → IN DEVELOPMENT: picked up
- 2026-10-28 — applicability gate: no blocking findings. Non-blocking: the READY-gate hint in `internal/flow/brine.go` restates §4.6 without the changelog (noted; it points at §4.6, decision 4 keeps this prose-only); acceptance grep fragile to line wrap (fixed inline); file-wide greps weak but correct (noted); `cli-reference.adoc` lists only the heading (noted).
- 2026-10-28 — plan amended inline: acceptance grep on rules item 6 now uses `grep -A1 … | grep -i changelog` so a wrapped line still matches (gate finding).
- 2026-10-28 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-10-28 — IN REVIEW → DONE: review: no blocking findings; noted 3 (F1–F3)
- 2026-10-28 — merged to main (PR #103, 7c5cda9, https://github.com/codcod/pickle/commit/7c5cda9)
