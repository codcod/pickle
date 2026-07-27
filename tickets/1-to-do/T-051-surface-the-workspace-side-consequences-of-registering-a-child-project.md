---
id: T-051
title: surface the workspace-side consequences of registering a child-project
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: S-M
---

# T-051 — surface the workspace-side consequences of registering a child-project

## Description

`pickle project add <name> <path>` writes one `[[project]]` block into `pickle.toml`, and
`pickle upgrade` then re-renders the marker blocks. Everything else that makes a workspace
actually treat the new directory as a child stays a hand edit, discovered by imitating the
first child — and pickle already knows the two facts (name, path) that all of it derives from.

Measured on a real second-child registration (`snowball` added alongside `rick` in the
`unity` workspace, pickle 0.1.0, 2026-07-27): after `project add` reported success, five
edits were still required, none of them prompted or mentioned.

1. **`.gitignore`** — `/snowball/`, so the nested child repo is never staged into the
   overarching repo. This one has teeth: until the entry exists the child *is* stageable, and
   the window is open from the moment the directory appears until a human remembers.
2. The workspace's **own Pi extension** — a `CHILD_DIRS` array that duplicates the child
   registry (`unity-guardrails.ts`).
3. **`opencode.jsonc`** — per-child never-stage glob patterns.
4. **`AGENTS.md` outside the markers** — the per-child records table row and provenance prose.
5. **`development/<child>/`** — the per-child record directory that workspace's convention
   requires.

Items 2–5 are that workspace's inventions and are arguably none of pickle's business; item 1
is not — it is mechanical, derived from `path`, and its omission is a staging accident rather
than a cosmetic gap. The interesting question this ticket must answer is **where the line
sits**, because multi-child is pickle's advertised shape: the second child is not an edge case.

Three candidate shapes, deliberately left un-chosen until refinement (a field-note discipline:
see `tickets/NOTES.md` § *Field-finding triage (2026-07-27)*, where two notes arrived with
implementations attached and both diagnoses were wrong):

- **Tell the truth on stdout** — `project add` prints the consequences it did *not* perform.
  Smallest change; helps only at registration time.
- **Perform the mechanical ones** — e.g. append the `.gitignore` entry. Narrow, but pickle
  starts editing a file it does not own.
- **A `doctor` check** — "registered child `X` has no `.gitignore` entry". The only shape that
  also catches a workspace that has *drifted* (a child registered before the check existed,
  or an entry deleted), not just a fresh registration.

Hard constraints on any of them: pickle must not edit hand-written prose outside its own
markers, and must not touch a workspace's non-pickle extension files — that separation is
precisely why `upgrade` can safely refresh `pickle-guardrails.ts` while leaving
`unity-guardrails.ts` alone.

Soft coupling: **T-052** (the post-upgrade audit error that the same `project add` → `upgrade`
sequence produces) — same onboarding session, same command pair, but a separate defect;
whichever lands second inherits the other's understanding of what that sequence should feel
like. Note also that if a child-directory guard is ever added to the *shipped*
`pickle-guardrails.ts` as part of this work, it must anchor child paths at the pathspec start
(plus `../` climbs) rather than anywhere in the string — the unanchored form also matches
`development/<child>/…`, which is ordinary bookkeeping (defect found and fixed workspace-side
during the same session).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: idea — field finding from adding a second child-project to the `unity` workspace with pickle 0.1.0
