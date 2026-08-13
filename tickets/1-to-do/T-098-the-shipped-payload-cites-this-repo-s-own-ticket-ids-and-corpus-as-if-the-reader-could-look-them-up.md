---
id: T-098
title: the shipped payload cites this repo's own ticket ids and corpus as if the reader could look them up
project: pickle
depends-on: []
spawned-by: [T-085]
impact: low
complexity: low
cost: S
---

# T-098 — the shipped payload cites this repo's own ticket ids and corpus as if the reader could look them up

## Outcome

After this ships, a reader of the installed skill in *any* project is never sent to a ticket id
or a body of evidence that exists only in pickle's own repo: the payload's examples and
justifications either stand on their own or are explicitly labelled as pickle's, so no one
wastes a lookup on `tickets/6-done/T-090` in a workspace that has no such ticket.

## Description

Two findings from T-085's review (N2, N3), batched because they are one defect in two places:
**the shipped payload speaks to its reader as if that reader were this repo.**

1. **`skill/resources/review-protocol.md:170-171`** — "Worked examples from `tickets/6-done/`:
   `T-090 F1` … `T-084 F2`". This is a lookup instruction pointing into the reader's *own*
   `6-done/`, where those ids are either absent or belong to entirely different tickets. The two
   illustrations are genuinely good — they are why `correctness` and `spec-unclear` exist as
   distinct classes — so the fix is to keep the defect shapes and drop the lookup framing, or to
   attribute them explicitly to pickle's corpus.
2. **`skill/resources/review-protocol.md:143`** — the skeleton is justified by "prose-only
   drifted into 13 header variants across **the corpus** before this skeleton existed". The
   corpus is this repo's, is named nowhere, and is unavailable to the reader the sentence
   addresses. The rule is right; its stated warrant does not travel.

**Why this is a real defect and not pedantry.** The payload already distinguishes the two safe
uses of a ticket id from this unsafe one: `tickets-README.md:63-64` uses ids as pure *syntax*
illustration (`board: T-084 ready → in development` — the id is arbitrary filler) and `:237`,
`:428` use them as *provenance tags* ("(T-083)" — naming which ticket introduced a rule). Neither
tells the reader to go and read them. T-085's addition is the first that does.

`tickets/NOTES.md:869-874` records this exact anti-pattern in advance — the **foreign-workspace
test** ("would this help a project that is not pickle?") kept as "an anti-overfitting guard on an
n=36 self-hosted corpus … recorded here for whoever next proposes generalising a self-host
observation into the shipped payload". This ticket is that guard firing for the first time.

### Scope

The two sites above are the known instances. A **sweep** of the rest of the payload for the same
shape belongs in scope — the question "does this sentence assume the reader is pickle?" is worth
asking once across all four payload documents, since T-085 is unlikely to be the only ticket that
ever reached for a local example. Refinement should decide whether the foreign-workspace test
earns a line in the rules as a standing authoring constraint, or stays a `NOTES.md` lesson.

### Couplings

Soft couplings only (no `depends-on:`):

- **T-085** (`spawned-by:`) — the ticket whose review found both instances, and which authored
  the prose in question. Nothing here disturbs the `class` vocabulary or the skeleton itself;
  this is about how they are *justified and illustrated*, not what they say.
- **T-067** (no link/anchor validation in the docs pipeline) — adjacent but different. T-067 would
  mechanically catch a dead cross-reference under `docs/`; this defect is a *semantically* dead
  reference in the payload that no anchor checker could detect, because `tickets/6-done/T-090`
  is a perfectly well-formed path that simply means something else in someone else's repo.
- **T-074** (rename the installed skill directory to brine) — touches the same four payload
  documents. If both are scheduled, doing T-074 first avoids a rebase; neither blocks the other.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-13 — created (TO DO). source: review: batched from T-085's review findings N2 and N3,
  both dispositioned *new ticket* — one theme (the payload addressing its reader as if the reader
  were this repo), two sites, filed as one ticket per rules §5's batching requirement
