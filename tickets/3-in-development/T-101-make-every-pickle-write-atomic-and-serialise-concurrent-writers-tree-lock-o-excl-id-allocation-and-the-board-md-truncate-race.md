---
id: T-101
title: make every pickle write atomic and serialise concurrent writers: tree lock, O_EXCL id allocation, and the BOARD.md truncate race
project: pickle
depends-on: []
spawned-by: [T-056]
impact: medium
complexity: medium
cost: L
---

# T-101 — make every pickle write atomic and serialise concurrent writers: tree lock, O_EXCL id allocation, and the BOARD.md truncate race

## Outcome

After this ships, running `pickle serve` (or a second `pickle` command, or an agent) beside a
write no longer risks reading a half-written board or handing two tickets the same id: every
ticket-tree write lands whole or not at all, and concurrent writers queue behind one lock
instead of racing.

## Description

`pickle` is written as if exactly one writer ever runs, serially, driven by a human or an agent
at a shell. That assumption is already false in one shipped configuration — `pickle serve`
re-reads the whole tree on every request and on a 5-second poll while the CLI writes it — and
it is the reason work on a writable dashboard kept stalling on foundations. This ticket builds
the foundations and nothing else: it is worth doing on its own merits, with no dashboard
change, no HTTP write endpoint, and no new UI.

Verified against the tree on 2026-08-14 (`git describe`: v0.8.0-3-gca1390f):

- **Nothing anywhere locks.** `flock`, `Mutex`, `O_EXCL` and `lockfile` have **zero** hits in
  the repo's `.go` files, tests included. Every mutating command is *load whole tree → compute
  → write*, with no revalidation between the load and the write.
- **`BOARD.md` is truncated in place.** `board.Regenerate` (`internal/board/board.go:457`) and
  `sync.Sync` (`internal/sync/sync.go`) both end in a bare `os.WriteFile`, which opens with
  `O_TRUNC`: a concurrent reader can observe an empty or half-written board. `serve` is exactly
  that reader — `handler.load` (`internal/serve/serve.go:128`) plus `buildHealth`'s
  `audit.Audit` (`internal/serve/view.go:378`) traverse the tree on every request. This is a
  live race today, before any write feature exists.
- **One atomic writer already exists, unexported and config-only.**
  `config.writePreservingMode` (`internal/config/config.go:915`) is create-temp-in-the-same-dir
  → write → `Sync` → `Chmod` → `os.Rename`, with a documented list of edge cases deliberately
  declined during T-018's re-review and T-069's rework. It is the right pattern and the wrong
  scope.
- **Ticket-id allocation is TOCTOU.** `runTicketNew` computes `ticket.NextNum`
  (`internal/cli/ticket.go:148`), `os.Stat`s the target path (`:152`) and then `os.WriteFile`s
  it (`:158`). Two concurrent creates get the same id. T-058's per-child `ticket_prefix` widened
  this surface rather than narrowing it: `NextNum` is now per prefix, so each prefix has its own
  race.
- **A move is three writes with no rollback.** `move.Move` writes the new file
  (`internal/move/move.go:132`), removes the old (`:134`) and regenerates the board (`:141`).
  The D7 comment (`move.go:121-126`) deliberately accepts a duplicate-id crash window between
  the first two — the right call for a crash, but a *reader* traversing the tree in that window
  sees the duplicate too, and `audit.Audit` reports it as an error in `serve`'s health banner.

### What this ticket does not do

Three things the parent ticket (T-056, dropped 2026-08-14) bundled with these, deliberately
left out:

- **No `internal/api`, no typed errors, no CAS.** The extraction only pays for itself if a
  second in-process caller exists; the only candidate read surface is T-065's JSON projection,
  which owns that seam now. `move.Move`'s nine `fmt.Errorf`s stay strings. The **one** exception
  is Task 8, which *documents* the partial-failure contract `move.Move` already has — prose over
  behaviour that ships today, changing no signature and adding no state. It is included here
  because this ticket's lock is what makes that contract finally statable: with concurrent
  pickle writers excluded, the remaining windows are crash and I/O failure, and those are
  enumerable.
- **No frontmatter serializer and no field writer** — that is T-102, which can be built on top
  of this without changing anything here.
- **No write endpoints in `serve`.** The method-qualified mux (`internal/serve/serve.go:63-77`)
  and `TestServeNeverWrites` (`internal/serve/serve_test.go:722`) both survive this ticket
  intact, and the design below is specifically arranged so that they do — see decision 6.

### Soft couplings

- **T-056** (dropped 2026-08-14) — the XL umbrella this was split out of; its DROPPED banner
  maps all six work areas to where each one went.
- **T-079** — rick-artifact amendment in `serve`. Its scope contains "atomic write: temp file +
  `os.Rename`" verbatim, and its own text asks that the two share one helper rather than
  inventing two. This ticket ships that helper (`internal/atomicfile`); T-079 should consume it,
  not re-write it. T-079 is three tickets deep behind T-075/T-076/T-077, so this one lands first
  on any plausible ordering.
- **T-042** — collapses duplicated internal predicates and touches `internal/board`,
  `internal/sync`, `move_test.go` and `sync_test.go`; `NOTES.md` already says to sequence rather
  than run it concurrently with board work. Same advice here.
- **T-102** (the field writer) and **T-065** (the JSON read projection) both build on the tree
  lock this ships, and neither is blocked by it — they can be refined in parallel.
- **T-053** (done) — `serve`'s decision 1 ("read-only, absolutely") is *unaffected* by this
  ticket and is not renegotiated here; the renegotiation belongs to T-079, which introduces the
  first non-GET route.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the root-path child (`path = "."`), so the branch is cut in this repo:

```
git checkout main
git checkout -b feat/T-101-atomic-writes-and-tree-lock
```

WIP commits are encouraged. Ticket and board bookkeeping stays on `main` — never stage
`tickets/` on this branch (`pickle hooks install` enforces it locally). Publish only after
explicit approval; tidy the WIP commits into atomic ones first (root-path child, rules §0).

### Prerequisite gate (hard)

None. No `depends-on:`; nothing in flight touches `internal/config`'s writer.

Advisory only: if **T-042** is picked up around the same time it collides in `internal/board`
and `internal/sync` — sequence them, do not run them concurrently.

### Confirmed design decisions (do not deviate without asking)

1. **A new package `internal/atomicfile`, exporting `WriteFile(path string, data []byte)
   error`.** It is `config.writePreservingMode` moved verbatim — including its comment block on
   the edge cases T-018's re-review and T-069's rework deliberately declined (ACLs, setuid bits,
   read-only files, and the umask reasoning for the not-yet-exists path). `internal/config`
   keeps `writePreservingMode` as a one-line wrapper (or calls `atomicfile.WriteFile` directly);
   its `verifyOnlyPayloadVersion` parse-back guard (`config.go:524`, `:544`) stays in `config`,
   because it is config-specific and not part of the write primitive.
2. **Route only the ticket-tree writers through it**: `board.Regenerate`
   (`internal/board/board.go:457`), `sync.Sync`'s board write (`internal/sync/sync.go`), and
   `move.Move`'s new-file write (`internal/move/move.go:132`). **Leave `internal/install` and
   `internal/hook` on `os.WriteFile`** — a scaffold writes files that do not exist yet, into a
   tree no reader is watching, and converting them buys nothing while enlarging the diff. Say so
   in the commit body so the omission does not read as an oversight.
3. **The lock is `syscall.Flock` on a dedicated lock file, not a lockfile-by-`O_EXCL`.** The
   kernel releases a flock when the holder dies, so there is no stale-lock recovery story to
   write; an `O_EXCL` sentinel file needs one and gets it wrong. Shipped targets are darwin and
   linux only (`.goreleaser.yaml:19-24`), where `syscall.Flock` exists — **no new dependency, no
   build tags**. If Windows is ever added to the target list, a build-tagged stub is that
   ticket's problem, and a comment in the package says so.
4. **Lock file location: `<root>/.git/pickle-tree.lock` when `<root>/.git` is a directory,
   otherwise `filepath.Join(os.TempDir(), "pickle-tree-"+sha256(absRoot)[:16]+".lock")`.**
   Neither location is visible to `git status`, which is the point: a lock file inside
   `tickets/` would show up as untracked noise in every installed project and would need
   `.gitignore` management pickle does not currently do. `.git` being a *file* (worktrees,
   submodules) takes the temp-dir branch — do not parse the gitdir pointer.
5. **Acquire with a bounded wait, never an unbounded block.** `LOCK_EX|LOCK_NB` in a retry loop
   (short backoff, ~10 s ceiling), then fail with a message naming the lock file and suggesting
   that another `pickle` command or `pickle serve` may be running. A CLI that hangs forever
   behind a wedged holder is worse than one that refuses.
6. **Readers never create the lock file.** `lock.WithShared` opens it **without** `O_CREATE`;
   if it is absent, the read proceeds unlocked (the tree cannot be mid-write if no writer has
   ever run). This is what keeps `serve` a strict non-writer and keeps `TestServeNeverWrites`
   (`internal/serve/serve_test.go:722`) passing unchanged — the test is a sha256 snapshot of the
   whole tree, and a reader that created a lock file would break it. Writers create it with
   `O_CREATE` and **never delete it**; deleting a lock file is how mutual exclusion gets lost.
7. **The write lock spans load→check→write, not just the write.** In `move.Move` it is taken
   before `ticket.LoadAll` (`move.go:43`) and released after the post-move `audit.Audit`
   (`move.go:147-151`); in `runTicketNew` before `ticket.NextNum` (`cli/ticket.go:148`) and
   after `board.Regenerate`. A lock around the write alone would leave every check it guards
   racing.
8. **`serve` takes the shared lock in two places, not once per request**: inside
   `handler.load` (`serve.go:128`) and around `audit.Audit` inside `buildHealth`
   (`view.go:378`). One acquisition per *filesystem traversal*, not per request — the
   alternative threads an audit result through `newPage` and five handlers, colliding head-on
   with T-065 and T-077 in `view.go` for a display-only consistency gain. The residual
   window (board and health banner from two different instants) is a cosmetic artifact of a
   page that already refreshes every 5 s; record it as an accepted limitation in the package
   comment.
9. **Keep D7's write-new-then-remove-old order** in `move.Move` (`move.go:121-126`) exactly as
   it is. The lock closes the window for other *pickle* processes; D7 still covers the crash
   case, which no lock can. Do not "improve" it into a rename.
10. **`O_EXCL` stays even though the lock makes it redundant.** `os.OpenFile(path,
    O_WRONLY|O_CREATE|O_EXCL, 0o644)` replaces the `os.Stat`-then-`WriteFile` pair at
    `cli/ticket.go:152-158`. Belt and braces: an older binary, or any future caller that forgets
    the lock, still cannot silently overwrite a ticket.

### Tasks

#### Task 1 — `internal/atomicfile`

New package with `WriteFile(path string, data []byte) error`, lifted from
`config.writePreservingMode` (`internal/config/config.go:915-952`) together with its comment
block. Move the existing config tests that cover mode preservation, symlink resolution and the
create-when-absent path into `internal/atomicfile/atomicfile_test.go`, and leave `config`'s
tests asserting only what is config-specific. `internal/config` now calls
`atomicfile.WriteFile`.

#### Task 2 — route the ticket-tree writers through it

`board.Regenerate` (`internal/board/board.go:457`), `sync.Sync`'s board write, and
`move.Move`'s new-file write (`internal/move/move.go:132`). No signature changes. Add a test in
`internal/board` that reads `BOARD.md` in a goroutine while `Regenerate` runs in a loop and
asserts every read is either complete-old or complete-new (non-empty, starts with the generated
header, ends with a full final row) — this is the truncate-in-place regression guard.

#### Task 3 — `internal/lock`

New package: `WithExclusive(root string, fn func() error) error` and `WithShared(root string,
fn func() error) error`, implementing decisions 3–6. Path resolution in one unexported helper
(`lockPath(root string) string`). Package comment states: what the lock does and does not
protect (same-machine `pickle` processes only; not a crash-recovery mechanism; not a substitute
for D7), and the accepted limitation from decision 8. Tests: two goroutines with exclusive locks
serialise (a shared counter or an ordered channel proves it); a shared lock does not exclude
another shared lock; `WithShared` on an absent lock file runs `fn` and creates nothing (assert
with a directory listing before/after).

#### Task 4 — take the write lock on every mutating path

Wrap the bodies of `move.Move` (`internal/move/move.go:33`) and `runTicketNew`
(`internal/cli/ticket.go:88`) in `lock.WithExclusive`, spanning load→check→write per decision 7.
Also wrap `board.Sync`'s CLI entry point (`internal/cli/board.go`) and the `board sync` path.
`board.Regenerate` is called *inside* already-locked paths and must **not** take the lock itself
— flock is per file descriptor, so a nested acquire on a second descriptor would deadlock
against nothing useful. State that in `Regenerate`'s doc comment: "callers hold the tree lock".

#### Task 5 — `O_EXCL` id allocation

Replace `os.Stat` + `os.WriteFile` at `internal/cli/ticket.go:152-158` with a single
`os.OpenFile(..., O_WRONLY|O_CREATE|O_EXCL, 0o644)`; map `os.IsExist` to the existing
`"%s already exists"` message so the user-visible behaviour is unchanged. Add a cli-level test
using the T-043 harness (`capture`, `internal/cli/cli_test.go:212`) that asserts the collision
message; remember **no `t.Parallel()` anywhere in package `cli`** — CWD and both std streams are
process-global.

#### Task 6 — shared lock in `serve`

Per decision 8: `handler.load` (`internal/serve/serve.go:128`) and the `audit.Audit` call in
`buildHealth` (`internal/serve/view.go:378`). Confirm `TestServeNeverWrites`
(`internal/serve/serve_test.go:722`) still passes **unmodified** — if it fails, decision 6 was
violated somewhere.

#### Task 7 — audit and doctor stay quiet about the lock file

Confirm `pickle board audit` does not flag `.git/pickle-tree.lock` (it should not:
`ticket.LoadAll` reads only `tickets/<status>/` and filters on `filenameRE`). Add the check to
the acceptance run rather than a new test if it is already structurally impossible.

#### Task 8 — state `move.Move`'s partial-failure contract in its doc comment

`move.Move` performs three writes with no rollback (new file `internal/move/move.go:132`, remove
old `:134`, regenerate board `:141`) and then a post-condition `audit.Audit` (`:147-151`). Three
of its failure paths return a **non-nil error after the move has already been applied**, and
`Result` is partially populated in two of them — none of which the doc comment says. Write it
down, in `move.Move`'s own comment (`move.go:32`), as the caller-observable states:

1. **not applied** — any failure before the write at `:132` (unknown status, load problems,
   ticket not found, illegal transition, WIP, dependency gate, and now a lock-acquisition
   timeout). `Result` is zero.
2. **applied, board stale** — `board.Regenerate` failed after the file moved (`:141-143`).
   `Result.Path`/`OldPath` are set; the ticket *has* changed status and its History line *is*
   written. Recovery is `pickle board sync`.
3. **applied, tree dirty** — the post-move audit reported errors (`:147-151`).
   `Result.Path`/`OldPath`/`Warnings` are set; the move stands and something else in the tree is
   broken.
4. **applied, clean** — nil error.

The load-bearing sentence is that **a non-nil error does not mean nothing happened**: a caller
must read `Result.Path` before concluding otherwise. Also say what the lock does *not* cover —
D7's duplicate-id window (`move.go:121-126`) is a **crash** window, and no lock closes it.

Documentation only: no signature change, no new state, no behaviour change, and deliberately
**no** typed or sentinel errors — those were left unfiled with T-056 areas 1+3 and a caller who
needs to branch on these states is the trigger to file them, not this comment.

### Acceptance test

Run from the repo root, all must be green:

```
just build && just test && just lint && just docs-check
```

Then the concurrency checks:

1. **Concurrent creates get distinct ids** (the TOCTOU regression). In a throwaway install, per
   the self-modify policy in `AGENTS.md` — never against this repo:

   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q . \
     && ./pickle-test install --project demo --path . >/dev/null \
     && for i in 1 2 3 4 5; do ./pickle-test ticket new "concurrent $i" --project demo & done; wait \
     && ls tickets/1-to-do/ | wc -l   # expect 5
   ```

   Expect five files with five distinct ids (`T-001`…`T-005`) and `./pickle-test board audit`
   reporting zero errors. Before this ticket the same loop produces duplicate ids or fewer than
   five files.

2. **No reader ever sees a truncated board**: `go test ./internal/board/ -run TruncateRace
   -count=20` (the Task 2 test).

3. **The lock actually serialises**: `go test ./internal/lock/ -count=5`.

4. **`serve` still writes nothing**: `go test ./internal/serve/ -run TestServeNeverWrites` —
   passing with the test file unmodified (`git diff --stat internal/serve/serve_test.go` shows
   no change to that test's body).

5. **The partial-failure contract is readable where a caller will look for it**:
   `go doc github.com/codcod/pickle/internal/move Move` prints all four caller-observable states
   from Task 8, including the "a non-nil error does not mean nothing happened" sentence.

6. **`serve` beside a writer stays coherent**, by hand: `./pickle-test serve` in the throwaway
   install, then run `./pickle-test ticket move …` in a loop in another shell for ~30 s while
   the board page polls. Expect no blank page, no 500, and no duplicate-id error in the health
   banner.

### Docs update (mandatory when user-facing)

Behaviour is user-visible in one way — a command can now refuse with a "another pickle process
holds the tree lock" error, and `pickle serve` is officially safe to run beside the CLI.

- `docs/user-manual/cli-reference.adoc`, `== pickle serve` (line ~915): a short subsection
  saying `serve` may be run beside the CLI, that it takes a shared read lock and still never
  writes.
- `docs/user-manual/cli-reference.adoc`, `== Overview` (line ~8): one paragraph on the tree
  lock — where the lock file lives, that it is released automatically when a process exits, and
  what the timeout message means.
- No skill-payload change: this alters no flow rule (and the payload must not name pickle's own
  internals — `AGENTS.md`'s foreign-workspace test).

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated and registered (`docs-check` proves the includes resolve).
3. Write a summary: files touched, the two new packages, and anything deferred (notably: no
   typed errors, no CAS, no serializer — T-102 and T-065 own those; the partial-failure contract
   is *documented* by Task 8, not changed).
4. Suggested Conventional Commit message:

   ```
   feat(write): make ticket-tree writes atomic and serialise concurrent writers (T-101)

   Lift config's temp-file+rename writer into internal/atomicfile and route
   board.Regenerate, sync.Sync and move.Move through it; add an internal/lock
   flock-based tree lock spanning load→check→write on every mutating path, a
   shared read lock in serve, and O_EXCL ticket-id allocation.
   ```

5. Tidy the WIP commits into atomic ones (root-path child) — a plausible split is
   `feat(write): atomicfile` / `feat(write): tree lock` / `fix(ticket): O_EXCL id allocation` /
   `docs(manual): …`.
6. Commit locally on the branch; present the message for approval. Do not push or open an MR
   before then. Before pushing, `git fetch origin main && git diff --name-only origin/main...HEAD
   | grep '^tickets/'` must print nothing.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-14 — created (TO DO). source: chat: refinement split of T-056 (dropped the same day) —
  work area 2 plus the live BOARD.md truncate race, the one part with standalone value and no
  dependency on a writable dashboard
- 2026-08-14 — plan gained Task 8 (document `move.Move`'s partial-failure contract) while still
  in READY: the one fragment of T-056 work area 3 that costs a paragraph and needs no consumer.
  Areas 1+3 otherwise stay unfiled.
- 2026-08-14 — TO DO → READY: plan complete
- 2026-08-15 — patched by T-104's review impact sweep: decision 8's `view.go` collision list
  drops T-055 (dropped, absorbed by T-104); both lock sites it names are unchanged by T-104.
- 2026-08-15 — READY → IN DEVELOPMENT: picked up
