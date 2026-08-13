---
id: T-099
title: guard the payload against speaking to its reader as if the reader were pickle's own repo
project: pickle
depends-on: []
spawned-by: [T-098]
impact: medium
complexity: low
cost: S
---

# T-099 — guard the payload against speaking to its reader as if the reader were pickle's own repo

## Outcome

After this ships, a sentence that only makes sense inside pickle's own repo cannot reach the
shipped payload unnoticed: a mechanical check fails the build the way a broken link would, instead
of depending on someone remembering to run a hand sweep and reading carefully enough to catch what
the last two sweeps missed.

## Description

T-098 removed four sites where the shipped payload (`skill/`, installed into other people's
projects as `.agents/skills/ticket-flow/`) addressed its reader as though that reader were
pickle's own repo. It deliberately built **no** mechanical guard, on T-085's discipline that
machinery waits for recurrence, and pre-registered the trigger in `tickets/NOTES.md`: *file the
check the first time a second instance is found in a review.*

**That trigger has fired, and the evidence is stronger than "n=2".** T-098's refinement ran a
deliberate sweep across all five payload files and declared the payload clean at three sites. Two
more have surfaced since, each found by a different reader after the sweep. **Both are already
fixed on T-098's branch** — they are cited here as the evidence for building the check, not as
cleanup still owed. This ticket is a *regression guard*, and the two sites below are its first
two test cases:

1. **At pickup** — `tickets-README.md:320` cited `pickle`'s own `skill/resources/TEMPLATE.md`, a
   path that only resolves in this repo (installed workspaces have
   `.agents/skills/ticket-flow/resources/TEMPLATE.md`).
2. **At review** — `review-protocol.md:157` froze the `class` vocabulary because "the
   **pre-registered criterion** this column exists to test needs a fixed vocabulary to count
   against". That criterion lives in *this repo's* `tickets/NOTES.md`. No other project has one.

The pattern that matters is not the count but *how* both escaped: **neither is catchable by the
four `rg` patterns T-098 left behind.** Site 1 is a path, site 2 is a definite-article appeal to
evidence the reader does not have. The existing patterns look for ticket ids and the literal
strings `the corpus` / `this repo`. So a third hand sweep is the wrong instrument — it is the
instrument that already failed twice.

### What the check must catch

Beyond the two shapes T-098's patterns already cover (lookup-shaped ticket references, and
`the corpus` / `this repo`), the two that got through:

- **Repo-only paths** — `skill/…`, `internal/…`, `cmd/…`, `docs/…`, `tickets/6-done/…` and
  friends appearing inside `skill/` as if the reader could open them. The subtlety: `tickets/`
  paths *are* legitimate in the payload (every installed project has `tickets/1-to-do/`), so this
  cannot be a blanket path ban — it is specifically paths rooted in **pickle's source tree**.
- **Definite-article appeals to invisible evidence** — "the pre-registered criterion", "the
  corpus", "the 13 variants". Hardest of the four and the one most likely to need judgement
  rather than a regex; a keyword list (`pre-registered`, `the corpus`, `our own`, bare counts
  paired with evidence nouns) is probably the honest 80%.

### Design questions for refinement (do not pre-empt)

- **Where does it live?** A `board audit` check, a `just` recipe in the docs pipeline, or a Go
  test over the embedded payload. Note that `board audit` is shipped-payload behaviour and runs in
  *foreign* projects, where policing pickle's own prose is meaningless — which argues for a
  repo-local lint, not an audit check. **This is the central decision** and it should be settled
  before any code.
- **What is the allowlist mechanism?** T-098 established six legitimate id references that must
  survive; any check needs a way to say "this one is fine" that does not rot.
- **Does it fail or warn?** T-098's Description argued the check is ~20 lines; the two escapes
  suggest the honest version is larger, which is why cost is S and not XS.

### Couplings

Soft couplings only:

- **T-098** (`spawned-by:`) — fixed the four sites and pre-registered this trigger. Its
  `## Review` table (N7) and the `NOTES.md` entry carry the four existing `rg` patterns, so this
  ticket starts from them rather than re-deriving them.
- **T-067** (no link/anchor validation in the docs pipeline) — plausibly the same home. If T-067
  builds a docs-linting harness, this check may be a rule inside it rather than a standalone
  thing. Worth deciding together; neither blocks the other.
- **T-074** (rename the installed skill directory to brine) — changes payload paths, so any
  hard-coded `skill/` or `.agents/skills/ticket-flow` string in this check would need updating.
  Cheaper if T-074 lands first.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-13 — created (TO DO). source: review: T-098's review, finding N7, disposition *new
  ticket*. Filed against a **pre-registered trigger** rather than a fresh judgement call: T-098
  recorded in `tickets/NOTES.md` that a mechanical guard would be built the first time a second
  instance reached a review, deliberately declining to build one at n=1. Two instances have since
  escaped T-098's own hand sweep — a repo-only path found at pickup, and an appeal to a
  pre-registered criterion the reader does not have, found at review — and neither is catchable
  by the four `rg` patterns that sweep left behind, which is the actual argument for machinery
  over a third sweep
- 2026-08-13 — patched by T-098's impact sweep (review step 8): T-098 shipped, so both cited
  instances are now fixed on its branch and the line numbers above (`tickets-README.md:320`,
  `review-protocol.md:157`) refer to the pre-fix text — read them as test cases for the check,
  not as outstanding cleanup. Scope and design questions are otherwise unchanged; nothing here
  was re-graded
