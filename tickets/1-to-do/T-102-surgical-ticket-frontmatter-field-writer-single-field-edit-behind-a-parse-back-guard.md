---
id: T-102
title: surgical ticket frontmatter field writer: single-field edit behind a parse-back guard
project: pickle
depends-on: []
spawned-by: [T-056]
impact: low-medium
complexity: medium
cost: M
---

# T-102 — surgical ticket frontmatter field writer: single-field edit behind a parse-back guard

## Outcome

After this ships, a ticket's grade or title can be changed by a command that provably touches
nothing else in the file — and any program (a future dashboard, an editor plugin, an agent) has
one audited way to write a ticket field instead of hand-editing markdown.

## Description

`pickle` can create a ticket and move it. It cannot **change a field on one**. There is no
frontmatter serializer anywhere in the repo: `ticket.ParseFrontmatter`
(`internal/ticket/ticket.go:243`) is one-way and lossy — it drops key order, inline comments and
quoting, and silently last-wins a duplicated key — and the only writers are `ticket.Scaffold`
(which renders a whole new file) and `move.Move` (which appends a History line to the body).
Re-grading a ticket, retitling one, or setting `family:`/`depends-on:` is therefore a hand edit
in an editor, every time.

The pattern to copy already exists and is proven: `atomicfile.WriteFile`
(`internal/atomicfile/atomicfile.go` — extracted from the former
`config.writePreservingMode` by T-101) plus `verifyOnlyPayloadVersion` (`config.go:544`, invoked
at `:524`) — write a single intended field, re-parse the result, and **refuse** if any field other
than the intended one changed. That guard is what makes a surgical edit a testable claim rather
than a hope.

Two constraints are specific to tickets and must be honoured:

- **Duplicate keys are detectable, not impossible.** T-040 shipped `Ticket.DuplicateKeys`
  (`internal/ticket/ticket.go:41`, populated at `:797`) and a `board audit` error
  (`internal/audit/audit.go:83-85`), but its decision D1 deliberately left parse semantics
  last-wins. A writer that re-renders frontmatter from a parsed map would silently delete the
  losing duplicate. It must **refuse** on a non-empty `DuplicateKeys` rather than repair, and
  say which key.
- **A `schema_version`-style fail-closed guard is pre-registered here.** T-065's refinement
  parked it (`NOTES.md:428`) against this work: the first path that re-renders frontmatter is
  the first path that can silently downgrade a file written by a newer pickle. Decide and
  record: refuse to write a ticket whose frontmatter carries a key the binary does not know, or
  preserve unknown keys verbatim.

### Nothing today enforces title ↔ filename ↔ H1 agreement

`board audit` checks the **id** against the filename (`internal/audit/audit.go`), never the
slug, and never the `# T-NNN — <title>` H1. So a title change has three possible scopes, and
refinement must pick one:

1. `title:` only — cheapest, immediately inconsistent with the filename and H1;
2. `title:` + H1, no rename — consistent within the file, filename slug goes stale (the rules
   already allow a stale slug: §3 says only the slug "may be tidied");
3. all three, with a `git mv` — correct, and the only option that needs to care about git.

Option 2 is the recommended default and the cheapest way to prove the write path end to end.

### Surface

A CLI verb is the honest first consumer — `pickle ticket set <id> --impact high` /
`--complexity` / `--cost` / `--title` / `--family` — because it ships value with no browser and
no HTTP write path, and because demand for a *writable dashboard* remains unevidenced
(`NOTES.md:151`). The library function is the point; the verb is how it gets exercised and
reviewed.

### Soft couplings

- **T-056** (dropped 2026-08-14) — the umbrella this was work area 4 of.
- **T-101** — the tree lock and atomic writer. This ticket should call
  `atomicfile.WriteFile` and take `lock.WithExclusive` rather than inventing either. Not a hard
  dependency: if T-101 has not landed, this can use a plain write and adopt both later — but
  building it *after* T-101 is strictly cheaper.
- **T-065** — the JSON read projection; the `schema_version` guard above was parked against this
  ticket by T-065's refinement.
- **T-079** — its "lifecycle-field guard" is the same `verifyOnlyPayloadVersion` pattern
  inverted, over `docs/specs/**` rather than `tickets/`. Compare notes; do not share code across
  two different file formats just because the pattern rhymes.
- **T-038** — tightens `validateTitle` (`internal/cli/ticket.go:183`). A `--title` edit must
  reuse whatever validation T-038 leaves, so let T-038 land first if both are in flight.
- **T-060** (dropped) and `pickle ticket renumber` — a renumber is the other operation that
  rewrites a ticket's identity fields; check its shape before designing this one.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-14 — created (TO DO). source: chat: refinement split of T-056 (dropped the same day) —
  its work area 4, kept because a guarded single-field writer is schedulable on its own as a
  `pickle ticket set` verb, with no dashboard and no HTTP write path
- 2026-08-15 — patched by T-101's review impact sweep: the pattern this plan says to copy moved.
  `config.writePreservingMode` no longer exists — T-101 extracted it verbatim into
  `internal/atomicfile.WriteFile`, which is exported, so this ticket consumes it rather than
  mirroring an unexported helper. The `verifyOnlyPayloadVersion` parse-back guard is unchanged
  and stays in `internal/config`.
- 2026-08-16 — patched by T-065's review impact sweep (step 8): T-065 shipped read-only, so the
  `schema_version`-style guard parked against this ticket is **confirmed still un-triggered** —
  `internal/state` reads frontmatter into `front_matter` verbatim and re-renders nothing, so no
  write path can yet drop an unknown key. The pre-registration stands unchanged, and this ticket
  is still the first work that creates the hazard. (The `NOTES.md:428` line-number citation above
  is stale as written — cite that file by heading, per AGENTS.md; the parking rationale is in
  T-065's own Description under "Envelope and versioning".) Nothing re-graded
- 2026-08-23 — patched by **T-038's review impact sweep**: T-038 is `6-done/` (branch unmerged),
  so its coupling note above resolves. The validation a `--title` edit must reuse is now concrete:
  `validateTitle` rejects an empty/whitespace-only title, any of the five Unicode line terminators
  (`LF`, `CR`, `U+0085`, `U+2028`, `U+2029`), a padded `"---"`, and any title past
  `maxTitleRuneLen` (120, a new exported-within-package const to reuse rather than re-derive).
  The cited line `internal/cli/ticket.go:183` is stale — the function moved and grew; find it by
  name. "Let T-038 land first" is satisfied.
