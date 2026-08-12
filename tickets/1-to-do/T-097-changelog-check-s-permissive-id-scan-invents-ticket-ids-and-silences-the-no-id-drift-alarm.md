---
id: T-097
title: changelog check's permissive id scan invents ticket ids and silences the no-id drift alarm
project: pickle
depends-on: []
spawned-by: [T-095]
impact: medium
complexity: low
cost: S
---

# T-097 — changelog check's permissive id scan invents ticket ids and silences the no-id drift alarm

## Outcome

After this ships, `pickle changelog check`'s exclusion summary never presents a non-ticket
token (`SHA-256`, `UTF-8`, `CVE-2024`) as a ticket id, and its `(+N with no ticket id)` clause
fires for every bookkeeping subject that names no *ticket*, rather than being silenced by an
id-shaped token that happens to appear in the prose.

## Description

Non-blocking finding N3 from T-095's review. T-095 decision 2 deliberately made the exclusion
summary's id set a **permissive** scan of the whole `board:` subject with the pre-existing
`idRE` (`\b[A-Z][A-Z0-9]*-\d+\b`), rather than a grammar-strict parse anchored to the rules §0
leading-id form. That decision was right about the case it weighed — measured across this
repo's history, 8 of 9 multi-id `board:` subjects carry their extra ids in the verb phrase, so
a strict parser drops most of them — and it explicitly accepted false-positive *noise* as the
price.

What decision 2 did **not** weigh is that a false positive does not merely add noise: it
**silences an alarm**. `printExclusions` counts a subject into `noID` only when
`len(ex.IDs) == 0`, so any id-shaped token anywhere in the subject suppresses that subject's
contribution to the `(+N with no ticket id)` clause — the clause T-094 decision 4 introduced as
"the loudest possible symptom of a convention drift", and which
`docs/user-manual/cli-reference.adoc` currently promises "is never the thing the summary
hides".

Measured (scratch repo, `pickle` at `bf59b7a`), a bookkeeping commit carrying **no ticket id at
all**:

```
$ git commit -m "board: note the SHA-256 subject handling"
$ ./pk changelog check
  excluded 1 board: bookkeeping commit(s) mentioning SHA-256 (--show-excluded for subjects)
```

Both failures at once: `SHA-256` is reported as though it were a ticket id, and the `+1 with no
ticket id` clause that should have fired does not. Tokens verified to match `idRE`: `SHA-256`,
`UTF-8`, `ISO-8601`, `AES-256`, `RFC-7231`, `CVE-2024`, `HTTP-2`, `PR-42`, `MR-7`.

The exposure is currently zero *in this repo* — T-095 measured that no non-`T-` id-shaped token
exists anywhere in its commit-subject history — but `pickle` is installed into other projects,
whose bookkeeping prose this project has never seen, and where a `board:` commit mentioning
`UTF-8` or `CVE-2024` is entirely ordinary.

Fixing this reopens T-095 decision 2, which is why it is a ticket rather than a review fix.
Candidate approaches, cheapest first: (a) recognise only the id prefixes the flow actually
uses, reading the configured ticket-id prefix rather than accepting any `[A-Z][A-Z0-9]*`
family; (b) keep the permissive scan for *display* but compute the `noID` count from a strict
ticket-id rule, so the alarm and the inventory stop sharing one predicate; (c) a small
stop-list. (b) is the most faithful to decision 2's intent — it preserves the superset the
summary wants while restoring the alarm's meaning — and is likely the smallest diff.

Whatever ships must keep T-095 decision 2's measured property: extra ids carried in a `board:`
subject's *verb phrase* must still be named (`board: T-089 reviewed and done, T-090 filed,
T-070 re-graded` → all three). Pin it with the regression tests T-095 already added in
`internal/changelog/changelog_test.go`.

Soft couplings: **T-093** shipped the command and `idRE`; **T-094** decision 4 introduced the
`+N` clause this finding shows can be silenced; **T-095** decision 2 chose the permissive scan
and is the decision this ticket reopens. Related, no action required: `boardIDRE`'s captured id
is now written but never read outside package tests (T-095 review finding N12) — if this ticket
introduces a strict ticket-id rule, that regex is the natural place for it to live.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: T-095's review, non-blocking finding N3, dispositioned
  `new ticket` because fixing it reopens T-095's locked decision 2 (the permissive `idRE` scan)
  rather than correcting an error in its implementation. Graded `medium`/`low`/`S`: impact
  raised above T-095's own `low` because this one can make the report state something false
  (a fabricated id) and suppress the drift alarm the command's docs promise, in *installed*
  projects rather than only at an edge of this one; complexity `low` and cost `S` because the
  likely fix is splitting one predicate into two in `printExclusions`/`Check`, with the
  regression tests already in place
