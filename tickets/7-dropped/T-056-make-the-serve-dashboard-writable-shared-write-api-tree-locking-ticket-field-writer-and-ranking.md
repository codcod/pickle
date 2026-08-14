---
id: T-056
title: make the serve dashboard writable: shared write API, tree locking, ticket field writer, and ranking
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: high
cost: XL
---

# T-056 — make the serve dashboard writable: shared write API, tree locking, ticket field writer, and ranking

## Outcome

After this ships, `pickle serve` can safely accept a second, concurrent writer — edits from the browser, manual ranking — without corrupting the ticket tree, because the write layer's one-writer assumptions (locking, atomicity, field-level writes) have been closed first.

## Description

> **DROPPED 2026-08-14 — split at refinement; do not implement, do not re-file from this
> text.** The audit below was re-verified against the tree on the day it was dropped and its
> *substance* still holds (zero `flock|Mutex|O_EXCL` hits repo-wide, no frontmatter serializer,
> no `internal/api`, nine bare `fmt.Errorf`s in `move.Move`, a GET-only mux). Its **line
> references are stale** and every cited signature now also takes `def *flow.Definition`
> (T-080/T-081, filed after this ticket): `move.go:62`→`:43`, `move.go:129`→`:132`,
> `move.go:142-147`→`:147-151`, `move.go:118`→`:121-126`, `config.go:478`→`:915`,
> `config.go:347`→`:524`/`:544`, `board.go:331`→`board.Regenerate` at `:457`,
> `cli/ticket.go:124`→`:152`, `cli/ticket.go:154`→`:183`, `ticket.go:182`→`:243`,
> `serve.go:62`→`:63-77`, `serve_test.go:416`→`:722`, `audit.go:86`→`:178-186`,
> `project.go:38`→`:42`. Take the successor tickets' references, not these.
>
> **Why it was dropped:** it is an XL umbrella of six work areas that cannot pass the READY
> gate (§4) as one plan, its central premise (a *writable dashboard*) has been downgraded twice
> for unevidenced demand (`NOTES.md:127`, `:151`), and three of its six areas acquired
> co-owners after it was filed. Split at the user's direction on 2026-08-14, one part per
> destination:
>
> | work area | where it went |
> |---|---|
> | 2 — concurrency foundation (+ the live `BOARD.md` truncate race) | **T-101** (READY) — atomic writes, flock tree lock, `O_EXCL` id allocation |
> | 4 — ticket field writer | **T-102** (TO DO) — surgical single-field edit behind a parse-back guard, shipped as a CLI verb |
> | 5 — ranking | closed "don't rank at all" on 2026-08-01 (T-063); its one surviving residue is **T-103** (TO DO), the `cost` tiebreak, filed because the trigger recorded below has fired |
> | 1 + 3 — `internal/api`, typed errors, CAS | **not filed.** Area 1's own text says to drop the extraction if a single audited write chokepoint is not the goal — it is not, while no second writer exists. **T-065** owns the read-projection seam it named. Re-file only when a real in-process caller appears. |
> | 6 — write endpoints in `serve` (CSRF, `Origin`/`Sec-Fetch-Site`, replacing `TestServeNeverWrites`) | **folded into T-079**, which introduces the first non-GET route in `serve` and needs all of it anyway. The renegotiation of T-053's decisions 1 and 9 and both its non-goals goes with it. |
>
> **Errors and stale evidence in the text below, marked so they are not mined back out as
> fact:**
> - *"T-059's `family:` is set on 0 of 63 tickets"* — it is now set on **4 of 100** (all
>   `family: T-075`). The conclusion it supported (weak demand for hand-curated order) is
>   weakened but not reversed.
> - *"`critical` and `high-critical` are unused across all 63 tickets"* — still unused, but the
>   corpus is **100** tickets, and the `medium` group is 7 deep in TO DO (44 of 100 overall),
>   well past the "≥5 deep" trigger the same paragraph sets. T-103 is that trigger firing.
> - The *"a cheaper v1: have the UI shell out to the CLI"* option in work area 6 was never
>   evaluated at refinement, because area 6 left with T-079. It is still the cheapest v1 and
>   T-079 should weigh it.

`pickle serve` (T-053) ships as a **read-only** view of the board, and that is not a
stylistic choice — it is what makes the dashboard safe to run beside an agent. The roadmap
is to grow it into a working surface: editing tickets from the browser, manual ranking, and
more. Doing that turns `serve` into a **second, concurrent writer** of the ticket tree, and
the write layer underneath it is built on the assumption that there is exactly one writer,
running serially, driven by a human or an agent at a shell.

This ticket is the programme that closes that gap. It is deliberately filed as one unit at
the user's direction; see *Splitting* below.

### Why the current write layer is not ready

An audit of every write path turned up the following. Each is harmless today and load-bearing
the moment a browser can POST:

- **No locking anywhere.** `flock|Mutex|O_EXCL` has zero hits across the repo. Every mutating
  command is *load whole tree → compute → write* with no revalidation: `move.Move` reads all
  tickets at `internal/move/move.go:62` and writes at `:129`.
- **One atomic writer exists.** `config.writePreservingMode` (`internal/config/config.go:478`)
  is temp-file + `os.Rename`, guarded by a parse-back check that refuses if any field other
  than the intended one changed (`config.go:347`). Everything else is a bare `os.WriteFile`.
- **`BOARD.md` is truncated in place** (`internal/board/board.go:331`) while `serve` reads it
  via `audit.Audit` on *every request*. That is a live race today, before any of this work.
- **Ticket-id allocation is TOCTOU.** `internal/cli/ticket.go:124` does `os.Stat` then
  `WriteFile`; two concurrent creates get the same id.
- **Parsing is lossy and one-way.** `ticket.ParseFrontmatter` (`internal/ticket/ticket.go:182`)
  drops key order, inline comments and quoting, and silently last-wins duplicate keys. There
  is no serializer: writing a changed field back has no code path at all.
- **`ticket new` has no library form.** `runTicketNew` (`internal/cli/ticket.go:72-138`) does
  validation, id allocation, `Scaffold`, `WriteFile` and `board.Regenerate` inline in the argv
  handler; `validateTitle` (`:154`) is unexported and CLI-only. Creating a ticket requires
  going through `os.Args`.
- **Errors are strings.** `move.Move` returns eight distinct `fmt.Errorf`s — not found,
  illegal transition, WIP limit, missing reason, dependency not done, dependency not merged,
  already there, I/O — all collapsed to exit code 1 by `errf` (`internal/cli/cli.go:30`). A UI
  needs to map these to 404/409/422/400/500 without string-matching.
- **Process globals on the call path.** `cli.Payload`/`cli.Version` are package vars
  (`internal/cli/cli.go:17,20`); `loadConfig` resolves the root from `os.Getwd()`
  (`internal/cli/project.go:38`); handlers print to the real `os.Stdout`, which is why the test
  harness installs a pipe over it (`internal/cli/cli_test.go:150`) and why those tests cannot
  run in parallel. A server has no per-request CWD and no stdout.

### Work areas

1. **A shared write chokepoint (`internal/api`).** Most of the extraction is already done —
   `move.Move`, `board.Regenerate`, `sync.Sync` and `audit.Audit` are pure library calls. What
   is missing is `ticket.Create`, typed/sentinel errors, an explicit `root` parameter in place
   of CWD discovery, and no package globals. Use `internal/`, not a public `pickle/api`:
   `internal/` is visible to a future `cmd/` binary in the same module and carries no semver
   obligation. The justification for a new package is **one audited path every writer must
   pass through** — locking, validation and CAS become a testable claim rather than a
   convention. If that is not the goal, the same three pieces could land in the existing
   packages instead, and the extraction should be dropped.

2. **Concurrency foundation.** Generalize `writePreservingMode` beyond config so every write is
   atomic; kill the BOARD.md truncate-in-place race; add a **filesystem** tree lock (a process
   mutex is useless — CLI and server are separate processes) spanning the whole
   load→check→write call, not just the write. Add `O_EXCL` id allocation.

3. **Semantics that must be decided, not just coded.**
   - *Post-condition audit ownership.* `move.Move` runs `audit.Audit` **after** writing and
     returns an error if the tree is now dirty (`move.go:142-147`). For a CLI that is a loud
     warning to a human whose write already succeeded; as an HTTP response it is a lie in both
     directions. The API needs an explicit third state: applied-but-tree-now-dirty.
   - *Compare-and-swap.* Browsers double-submit and the htmx poll is 5 s, so every UI action is
     issued against stale state. `Move(id, target)` will happily apply a transition against a
     board that has since changed; it needs an `expectedStatus`. Retrofitting this later means
     changing every signature.
   - *Partial failure.* `move.Move` performs three writes (new file, remove old, regenerate
     board) with no rollback; the D7 comment at `move.go:118` deliberately accepts a
     duplicate-id crash window. The contract must state what a caller observes, and `serve`
     must render that state rather than 500.

4. **A ticket field writer.** Needed for title/grade/rank edits (not for move/create). Copy the
   `writePreservingMode` + `verifyOnlyPayloadVersion` pattern: surgical single-field edit,
   re-parse, refuse if anything else moved. Note that **nothing today enforces filename-slug ↔
   `title:` agreement** — the audit checks the id against the filename but never the slug or
   the `# T-NNN — <title>` H1. So the cheapest first edit feature is a title change with no
   file rename, and it is a good one to prove the write path on.

5. **Ranking — the first state not derivable from ticket files.** `board.Sort`
   (`internal/board/board.go:240`) carries an explicit decision **"D1 — deterministic, no
   hand-curated order"**, and `audit` asserts BOARD.md equals a fresh render
   (`internal/audit/audit.go:86`). Manual rank breaks the first and must be reconciled with the
   second. Candidates: a sidecar order file (one hot merge-conflict file, second source of
   truth, tickets stop being self-describing); an integer `rank:` (reordering rewrites N files,
   renumbering merges badly); **fractional indexing** (`rank: "a0h8"`, lexicographic
   insert-between — one file per move, merge-friendly, plain string sort, needs a
   duplicate-key check and occasional rebalancing); or **don't rank at all** and add axes
   instead (refine `impact`, or land T-045's `user-visible:`). The last option is cheap, keeps
   D1, and deserves a real hearing before drag-and-drop is built — the board is ordered by
   impact for a reason.

   **Hearing held 2026-08-01 — the answer is "don't rank at all". This work area is now
   mostly closed; see T-063 (dropped) for the full evidence.** T-063 proposed derived
   value-per-cost ordering, was adversarially reviewed before refinement, and was dropped on
   findings that apply to *any* ranking work here, manual or derived:
   - **The queue this work area would let a user reorder is READY, and READY has never held
     more than 2 rows** in 114 revisions of `tickets/BOARD.md` (`review-protocol.md:192` names
     READY as the pickup queue). Drag-and-drop over a ≤2-row list is not a feature. **Before
     building any ranking UI, re-measure READY occupancy** — if it is still ≤2, cut this work
     area entirely rather than refining it.
   - **The nearest shipped precedent has zero adopters.** T-059's `family:` — curated pickup
     grouping, the same class of mechanism — is set on **0 of 63 tickets** four days after
     merging, through exactly the tie it was built to break. That is the strongest available
     evidence about demand for hand-curated order in this flow, and it is negative.
   - **Ordinal grades do not support arithmetic.** Any score combining `impact` and `cost` is
     a function of an arbitrary linear numbering; renumbering `cost` on an equally defensible
     scale moves 11 of 18 rows. Lexicographic comparison is scale-invariant and strictly
     better behaved.

   **What survives, and is not this ticket's job:** replace the *id* tiebreak in `board.Sort`
   with `cost`, lexicographically beneath impact — ~4 lines plus a `costRank` map, no config,
   no new state, D1 intact. It cuts tied pairs 34 → 10. **Do the free thing first**, though:
   recalibrate `impact`, which `tickets-README.md:139-140` already mandates and T-045:76
   already names as the recommended starting position — `critical` and `high-critical` are
   unused across all 63 tickets. If a recalibration pass leaves the `medium` group ≥5 deep,
   file the cost tiebreak as its own small ticket.

6. **Writes in `serve`, and replacing its safety proof.** Two structural guarantees die on day
   one: the method-qualified mux (`internal/serve/serve.go:62` — anything but GET/HEAD is 405
   before a handler is reached) and `TestServeNeverWrites` (`internal/serve/serve_test.go:416`,
   a sha256 snapshot of the whole tree). **Replace the test, do not delete it**: the same
   snapshot, asserting that *only* the file the request was allowed to touch changed. That is
   a stronger invariant than "nothing changed" and it is what will catch a handler regenerating
   a board it had no business touching. Also required: CSRF tokens and `Origin`/`Sec-Fetch-Site`
   checks. A write endpoint on `127.0.0.1:8745` is reachable from any page the user has open —
   form POSTs are not preflighted, and DNS rebinding defeats naive `Host` checks.

   A cheaper v1 exists and should be evaluated at refinement: **have the UI shell out to the
   CLI** for every mutation. It preserves "the CLI is the only writer" exactly, keeps the
   audit's invariants intact, needs no serializer, and any locking added later benefits both
   surfaces. Ugly and ~10 ms per action, which is free at human pace. If that is chosen, work
   area 1 is not needed for v1.

### Renegotiating T-053

T-053's non-goals state *"no ticket editing or moving from the browser, no authentication"*,
and its decision 9 says *"`serve` is a human surface, not something an agent is told to run."*
Those are the current contract. This ticket must **overturn them on the record** rather than
letting them rot — the refinement should list which T-053 decisions are superseded and why.

### Scope boundary

Binary size and the sidecar-vs-single-binary packaging question are **out of scope** by user
decision, and are not recorded here.

### Splitting

Filed as one ticket at the user's direction. It will **not** pass the READY gate (§4) in this
shape and is expected to be split at refinement; the six work areas above are the natural
seams. A plausible sequence: work area 2 (concurrency foundation, which is worth doing on its
own merits regardless of the dashboard) → 1 → 3 → 4 → 5 → 6.

### Soft couplings

- **T-053** (done) — the dashboard whose decisions and non-goals this supersedes.
- **T-040** (done 2026-08-03) — frontmatter validation. **Read the correction:** T-040 decision D1
  deliberately left parse semantics unchanged — a duplicate key still last-wins in
  `ticket.ParseFrontmatter`; what shipped is `Ticket.DuplicateKeys` plus a `board audit` **error**
  reporting it. So the data-loss hazard for a field writer is *detectable*, not *removed*: the
  writer must still refuse (or repair) a ticket whose frontmatter carries a duplicated key rather
  than assume the parse is lossless. Weigh that when deciding whether this stays a soft coupling
  or becomes a hard `depends-on:` — the useful dependency is now the `DuplicateKeys` signal, which
  already exists.
- **T-043** — cli test-harness hardening and ticket-new coverage: **landed 2026-08-06**, so the
  overlap with work area 1 is now an asset rather than a hazard. The harness it leaves is one
  `capture(t, &os.Stdout|&os.Stderr, fn)` helper that restores early and closes once from either
  exit path, a `TestMain` sandbox that validates the repo root, and cli-level tests for
  `project add|list|remove`, `board audit`, `ticket new` and `install --hooks`. Reuse it — do not
  build a second harness, and note that no test in package `cli` may call `t.Parallel()` (the CWD
  and both std streams are process-global).
- **T-038** — `ticket new`'s title contract, i.e. the `validateTitle` that work area 1 moves.
- **T-042** — internal helper consolidation, touching `internal/board`/`internal/sync`;
  `tickets/NOTES.md` already warns it must not run concurrently with other board work.
- **T-045** — its `user-visible:` axis is an alternative to manual ranking (work area 5).
- **T-063** (dropped) — the "don't rank at all" hearing work area 5 asked for, held and
  answered on 2026-08-01. Its Description carries the measurements; the DROPPED banner carries
  the verdict and marks its own errors.
- **T-054**, **T-055** — `serve` UI work in flight; both touch the same templates/CSS.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: chat — write-path audit prompted by the plan to grow
  `pickle serve` beyond read-only (edit, ranking); supersedes T-053's read-only non-goals
- 2026-08-01 — work area 5 (ranking) answered "don't rank at all" and mostly closed; evidence
  and the surviving cost-tiebreak idea recorded from T-063 (dropped). Re-measure READY
  occupancy before refining any ranking UI.
- 2026-08-03 — patched by T-040's review impact sweep (finding N9): the T-040 soft coupling
  overstated what shipped. T-040 reports duplicate frontmatter keys at audit time but keeps
  last-wins parse semantics (its decision D1), so the field writer still needs its own guard;
  the coupling text now says so.
- 2026-08-06 — patched by T-043's review impact sweep: T-043 landed, so work area 1's “doing both
  separately means writing the same tests twice” hazard is resolved — the harness and the
  `ticket new` cli tests exist now, and this ticket should consume them rather than re-create them
  (the no-`t.Parallel()` rule in package `cli` comes with them)
- 2026-08-14 — refined, and the refinement's verdict was a split: the six work areas went to
  T-101 (area 2, refined to READY the same day), T-102 (area 4), T-103 (area 5's residue),
  T-079 (area 6, folded) and nowhere (areas 1+3, deliberately not filed). Code claims
  re-verified against v0.8.0-3-gca1390f; the corrections are in the DROPPED banner.
- 2026-08-14 — TO DO → DROPPED: split at refinement into T-101/T-102/T-103, area 6 folded into T-079, areas 1+3 not filed
