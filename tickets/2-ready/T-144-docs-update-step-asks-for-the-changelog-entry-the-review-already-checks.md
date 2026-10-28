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
grep -n -i "^6\. \*\*Docs update.*changelog" skill/resources/tickets-README.md   # expect one hit
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

<!-- empty until IN REVIEW -->

## History

- 2026-10-28 — created (TO DO). source: self-host: a review of brine's docs step, checked against a month of Claude Code sessions; the template's docs step omits the changelog the review checks for (T-143 F13)
- 2026-10-28 — TO DO → READY: plan complete
