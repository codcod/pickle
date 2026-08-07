---
id: T-083
title: a ticket does not say what you get from it: add an Outcome section and warn when it is missing
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low-medium
cost: S-M
---

# T-083 — a ticket does not say what you get from it: add an Outcome section and warn when it is missing

## Description

**Outcome.** After this ships, you can decide whether a TO DO ticket is worth refining by
reading its first two lines, instead of reconstructing the payoff from a mechanism narrative
written for whoever files it. Every ticket states, up front and in user-observable terms, what
changes when it lands — and `board audit` tells you which tickets still do not.

*(That paragraph is the feature, written in its own format. The rest of this Description is
evidence and options, per the house style.)*

### The problem, measured

`TEMPLATE.md:20` gives the Description exactly one instruction — *"The current spec of the
feature, in prose"* — and says nothing about stating a benefit. Nothing downstream asks either.
The result, counted across all 23 tickets in `1-to-do/` by whether sentence 1–2 tells a reader
what they *get*:

| opening shape | count | tickets |
|---|---|---|
| outcome / symptom first | 8 | T-051, T-052, T-055, T-065, T-067, T-077, T-078, T-079 |
| mechanism or provenance first | 9 | T-013, T-038, T-042, T-046, T-070, T-071, T-074, T-076, T-080 |
| mixed / roadmap | 6 | T-050, T-056, T-066, T-075, T-081, T-082 |

**The split correlates with provenance**, which is the actionable part: tickets born from
reviews and audits open with lineage and mechanism (*"Filed by the T-043 review (finding R2)…
It left the fourth reader behind"* — T-070), while tickets born from field use open with the
symptom (*"the documented onboarding sequence ends in an error and a non-zero exit, for a
workspace where nothing is wrong"* — T-052). T-070 is the clearest cost: its actual payoff — a
wrapped merge line is silently truncated in the board's DONE `merged` cell and in `serve`'s
ticket view — sits roughly 20 lines below the fold. T-077 is the best counter-example in the
tree and shows the fix is free: *"the reason the whole family is worth building: reviewing a
400-line solution design in a terminal, before approving it, is miserable."*

### The finding that makes this cheap: the sentence already gets written, into the wrong file

The two `impact` recalibration passes (`NOTES.md`, 2026-08-01 and 2026-08-03) produced **11
one-line justifications** that are exactly the missing text — and left every one of them in a
`NOTES.md` table instead of in the ticket:

| ticket | the sentence that answers "what do I get" | where it lives today |
|---|---|---|
| T-055 | "cosmetic CSS specificity bug" | `NOTES.md` only |
| T-038 | "narrow input hardening on a path that already rejects the dangerous cases" | `NOTES.md` only |
| T-057 | "the only open item guarding against **silent loss** … it has happened three times" | `NOTES.md` only |

Verified: neither `T-055` nor `T-038` mentions its impact rationale anywhere in the ticket file.
A reader opens the ticket, sees `impact: low`, and gets no reason — while a well-written one
sits one directory away. T-019 is the counter-example proving it is writable inline: *"which is
why the impact moves `low` → `low-medium`"*.

So the raw material exists and the authoring habit exists. What is missing is a **home for it in
the ticket** and anything that notices its absence.

### Item 1 — the Outcome lead (payload prose)

Give the ticket a place to say what it buys, and make grading write its reason there:

- `TEMPLATE.md` — an `## Outcome` section (1–3 sentences, user-observable terms, "after this
  ships, X"), placed **above** `## Description`, with the placeholder wording and a worked
  example.
- `tickets-README.md` §1/§3 — define the section, and extend §3's standing *"assess every new
  ticket against the existing backlog … and re-grade the board"* mandate so a grade's **reason
  lands in the ticket**, not only in a triage table.
- `SKILL.md` *make it a ticket* / *refine a ticket* — one clause each.

This is payload text: it ships to every installed workspace and bumps `payload_version`.

### Item 2 — audit the section's presence, as a warning only (Go)

A rule nothing checks is the bet this repo has already lost twice (`NOTES.md:100-101`; T-064's
verdict was *"the rule already exists and is not being followed"*). So Item 1 ships with a check:
`board audit` reports a ticket whose `## Outcome` is absent, empty, or still the template
placeholder.

**It must be `warnf`, never `errf`, and this is a hard constraint rather than a preference.**
`audit.Audit` runs as a post-op self-check at **six** call sites — `move.go:142`,
`cli/board.go:76`, `cli/install.go:89`, `cli/install.go:165`, `sync.go:73` and
`serve/view.go:299`. An error would make `ticket move`, `board sync`, `install`, `upgrade` and
the dashboard's health panel all fail for the 81 tickets that predate the section — including
the moves that would drain the backlog. For the same reason `Outcome` must **not** join
`requiredKeys` (`audit.go:27`); that list is frontmatter-only, and T-045 documented the same
migration break (`upgrade` never touches tickets, so every installed workspace would light up).

**Check structure, not prose.** A heuristic "does this sentence sound like a benefit" check
would be the T-050 defect shape, and this repo's one shipped guardrail has a recorded field
record of **1 false positive, 0 true positives**. Presence-and-not-placeholder is mechanical and
has no judgement in it.

### Why this is not T-045, T-063 or T-064

The theme has a graveyard and `NOTES.md:100-101` requires any new entrant to answer for it:

- **T-063** (value-per-cost ordering) and **T-045 valve 3** (backlog cap) changed how the board
  **sorts**. This changes what a ticket **says**; it adds no ordering, no score and no config.
  The standing "the pickup queue is READY, not TO DO" objection does not bite — the 2026-08-03
  pass explicitly scopes TO DO to *"what gets **refined** next"*, which is the exact activity
  this ticket serves.
- **T-064** proposed a **merit gate** that judges whether a ticket deserves to exist, and died
  because *"a checklist item ('is this worth it? yes') … launders the omission"*. `## Outcome`
  is **descriptive, not evaluative**: it states what changes, and makes no worth claim. Nothing
  is gated, nothing is blocked, no verdict is recorded.
- **T-045 valve 4** was a `user-visible:` **frontmatter axis**, dropped for *"recalibrate
  `impact` instead of adding an axis"*. This adds no key and no axis — it gives the
  recalibration's own prose a place to live, which is the cheaper thing that pass recommended.

The honest residual risk: Item 1 alone is unenforceable prose, which is what failed before.
Item 2 exists because of that, and the two should be weighed as a pair.

### Open questions for refinement

1. **Section, or a lead sentence in the Description?** A lead sentence adds no structure and
   codifies what T-052/T-077 already do; a section is the only form Item 2 can check without
   heuristics. This Description assumes the section **because** Item 2 was asked for — if
   refinement drops Item 2, reconsider.
2. **Does this belong to T-081 instead?** T-081's gate table takes *"a `##` section (and its
   non-emptiness)"* as its unit — literally Item 2's mechanism. If T-081 lands first, Item 2
   should be one row in its table rather than a bespoke check. T-081 is `depends-on: [T-080]`
   (an `L` refactor), so gating a ~25-line warning behind that programme is the thing to weigh.
   Kept **soft** deliberately; a hard `depends-on:` needs sign-off per §3.
3. **Backfill policy.** Whether to write `## Outcome` into the 23 open tickets (the 11 stranded
   `NOTES.md` sentences make a chunk of it near-free), and — separately — whether `6-done/` and
   `7-dropped/` are left alone. Recommended: backfill `1-to-do/`, leave the archives.
4. **Does the section reach the board or `serve`?** Default **no** — board machinery is the
   theme this file warns about. Recorded so it is a decision, not an omission.
5. **Length cap.** T-049 capped rendered board cells; if Outcome ever reaches a cell it needs
   the same discipline. Only relevant under Q4.

### Soft couplings

- **T-081** — gate table as data; overlaps Item 2's mechanism. See Q2.
- **T-074** — renames the installed skill directory and rewrites payload text; both touch
  `TEMPLATE.md`/`tickets-README.md`. Sequence, do not run concurrently.
- **T-022** (done) — the precedent for payload-prose-only tickets, and the source of the
  "another advisory prose block is the bet that just lost" objection Item 2 answers.
- **T-066** — CLI-surface documentation drift; a new `board audit` warning needs a line in
  `docs/user-manual/cli-reference.adoc`, which is that ticket's territory.
- **T-045**, **T-063**, **T-064** (all dropped) — the theme's prior art, addressed above.

### The case against, and what would falsify it

Demand is **n = 1** — the requester's own friction, the same provenance as T-059, T-060 and
T-062, the three recorded wastes (`NOTES.md:87-90`). No ticket has been picked up wrongly, or
skipped, for lack of an Outcome line; the cost is reading time, which nobody has measured. What
is stronger here than in those three is that the defect is **counted** (9 of 23) and the fix's
raw material is **already written** (11 sentences in `NOTES.md`).

**Pre-registered kill criterion**, in the T-045 style: if `## Outcome` ships and the next
refinement pass over `1-to-do/` is not measurably faster to triage — or if the sections are
being filled with restated titles — **drop the audit check and revert to a lead-sentence
convention**. A section that gets padded is worse than no section, and the padding will be
visible in the same scan that produced the table above.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: chat — a proposed "Decision Inputs" ticket section was
  challenged and reduced; the surviving finding was that 9 of 23 TO DO tickets open with
  mechanism rather than outcome, and that the two impact recalibrations wrote 11 usable
  one-line justifications into NOTES.md tables instead of into the tickets
