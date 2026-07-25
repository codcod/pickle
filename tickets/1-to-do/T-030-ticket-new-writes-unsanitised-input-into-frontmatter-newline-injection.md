---
id: T-030
title: ticket new writes unsanitised input into frontmatter (newline injection)
project: pickle
depends-on: []
spawned-by: [T-024]
impact: medium
complexity: low
cost: S
---

# T-030 — ticket new writes unsanitised input into frontmatter (newline injection)

## Description

`pickle ticket new` interpolates user input straight into the generated frontmatter
(`internal/ticket/ticket.go:293-310` via `internal/cli/ticket.go:119-122`), so a newline in an
argument **injects arbitrary frontmatter keys**. Reproduced against the built binary during the
T-024 review:

```
pickle ticket new "sneaky" --project demo --spawned-by "$(printf 'T-001]\nimpact: critical')"
→ created T-002
---
spawned-by: [T-001]
impact: critical]        ← injected line
impact: medium
---
board audit: 2 tickets, 0 error(s), 0 warning(s)   ← silently clean
```

The result is a corrupted ticket the audit calls clean (`ParseFrontmatter` takes the *first*
occurrence of a key, so the injected `impact` wins over the real one). The same hole **pre-dates
T-024** on the `title` positional: `ticket new "$(printf 'evil\nproject: nope')"` injects
`project: nope`, also audit-clean. `--spawned-by` merely inherits it.

This is a correctness bug rather than a security hole — the input comes from the operator or
their agent, not an untrusted party — but it silently produces malformed tickets that the
board's own validator endorses, which is exactly what the audit exists to prevent.

### Why the fix is obvious

The repo already has the convention. `move.sanitizeReason` (`internal/move/move.go:230-238`)
strips newlines and arrows from a `--reason` "so a reason can never be mis-read by
`ticket.LastHistoryStatus`". Ticket creation needs the same discipline.

### Scope

- Validate id-shaped inputs in `runTicketNew`: every `--spawned-by` (and, when T-027 lands,
  `depends-on`) token must match `^T-\d+$`, rejected with a clear error otherwise. This also
  fixes the confusing message the review logged as N3 — `--spawned-by "banana"` currently
  reports `spawned-by banana does not exist`, framing a malformed token as a missing ticket.
- Sanitise or reject newlines (and stray `---`) in the title before it reaches `Scaffold`.
- Decide whether duplicate ids (`--spawned-by "T-001,T-001"`, currently accepted and rendered
  verbatim) should be de-duplicated or rejected.
- Tests for each: injection attempt → non-zero exit or neutralised output, and `board audit`
  clean afterwards.

### Couplings

`spawned-by: [T-024]` — surfaced by T-024's review (findings N2, N3). Soft coupling to **T-027**
(the `depends-on` self-reference check): both add validation to id lists, and whichever lands
second should reuse the first's helper rather than duplicate it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
