---
id: T-056
title: make the serve dashboard writable: shared write API, tree locking, ticket field writer, and ranking
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: high
cost: XL
---

# T-056 — make the serve dashboard writable: shared write API, tree locking, ticket field writer, and ranking

## Description

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
- **T-040** — frontmatter validation. A genuine prerequisite: duplicate keys currently last-win
  silently, and a field writer on top of that is a data-loss bug. Should probably become a hard
  `depends-on:` at refinement, with user sign-off.
- **T-043** — cli test-harness hardening and ticket-new coverage. Overlaps work area 1 directly
  (same `os.Chdir`/`os.Stdout` globals, same `ticket new` gap); doing both separately means
  writing the same tests twice. Sequence them or absorb one into the other.
- **T-038** — `ticket new`'s title contract, i.e. the `validateTitle` that work area 1 moves.
- **T-042** — internal helper consolidation, touching `internal/board`/`internal/sync`;
  `tickets/NOTES.md` already warns it must not run concurrently with other board work.
- **T-045** — its `user-visible:` axis is an alternative to manual ranking (work area 5).
- **T-054**, **T-055** — `serve` UI work in flight; both touch the same templates/CSS.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: chat — write-path audit prompted by the plan to grow
  `pickle serve` beyond read-only (edit, ranking); supersedes T-053's read-only non-goals
