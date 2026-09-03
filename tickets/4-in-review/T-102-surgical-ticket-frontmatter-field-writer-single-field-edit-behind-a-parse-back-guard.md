---
id: T-102
title: surgical ticket frontmatter field writer: single-field edit behind a parse-back guard
project: pickle
depends-on: []
spawned-by: [T-056]
impact: low
complexity: medium
cost: M
---

# T-102 — surgical ticket frontmatter field writer: single-field edit behind a parse-back guard

## Outcome

After this ships, `pickle ticket set <T-NNN> --impact|--complexity|--cost|--family|--title <value>`
changes exactly one of those five fields on a ticket — re-parsing the result and refusing the
write unless every other line in the file is byte-identical to before. Re-grading a ticket,
retitling it, or setting `family:` no longer needs a hand edit in a markdown file; `depends-on:`
still does (out of scope here). The library function (`ticket.SetField` +
`internal/ticketset.Set`) is the point — the CLI verb is how it gets exercised and reviewed.

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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-102-ticket-set-field-writer
```

All work happens in this repo's own tree (`project: pickle`, `path = "."`, `layout = "in-tree"`).
Commit locally as you go; publish only per the project's commit policy (no push/MR without
explicit user approval) — see Finish below.

### Prerequisite gate (hard)

None outstanding. The three soft couplings that could have gated this are all `6-done/` and
merged already: **T-101** (`internal/atomicfile.WriteFile`, `internal/lock.WithExclusive` —
merged `dc982ae`), **T-038** (`validateTitle`'s current shape — merged `e5d2a11`), **T-065**
(confirmed the `schema_version`-style hazard stays un-triggered until this ticket — merged
`83a4d39`). **T-060** (`pickle ticket renumber`) was considered and dropped — there is no prior
renumber shape to check against; that coupling note is moot.

### Confirmed design decisions (do not deviate without asking)

1. **New package `internal/ticketset`** implements `pickle ticket set`'s guarded write,
   mirroring `internal/move`'s split: CLI flag parsing stays in `internal/cli/ticket.go`, the
   locked read-check-write-regenerate lives in its own package.
2. **Settable fields are exactly `impact`, `complexity`, `cost`, `family`, `title`** — the
   ticket's own "Surface" section. `depends-on:` stays a hand-edit; it is list-valued and gates
   pickup, which puts it out of scope here.
3. **Exactly one field per invocation.** `pickle ticket set` refuses (usage error) unless
   exactly one of `--impact`/`--complexity`/`--cost`/`--family`/`--title` is present on the
   command line. Batched multi-field edits are a future ticket if ever wanted.
4. **A `--title` edit rewrites frontmatter `title:` and the `# T-NNN — …` H1 only** — no
   filename/slug rename (Option 2 from the Description's three-option analysis: consistent
   within the file, cheapest way to prove the write path end to end; the slug going stale is
   already legal per rules §3, "may be tidied").
5. **The writer is line-based, never a full frontmatter re-render.** It locates and replaces
   exactly the one targeted key's line (plus, for `--title`, the one H1 line) by index; every
   other line in the file — including a key the binary does not recognize — is copied through
   byte-for-byte. This is what preserves unknown keys "verbatim" with no separate check, and is
   what turns the guard's "nothing else changed" claim into a literal line-diff rather than a
   semantic tree comparison (config's TOML equivalent needed tree-decode because TOML has
   multiple representations of equivalent content; flat one-key-per-line ticket frontmatter does
   not). This directly resolves the `schema_version`-style hazard T-065's refinement parked
   against this ticket (`NOTES.md` § "T-065 refinement — schema_version parked against T-102"):
   there is no re-render step that could ever drop an unrecognized key. (The pre-registration
  itself is recorded in `NOTES.md` § "If you proceed: the first batch", row 1 — not a heading
  named for this ticket specifically, since it predates T-102's own filing.)
6. **The parse-back guard, precisely:** after producing the candidate text, (a) re-parse its
   frontmatter and refuse if it no longer parses, if the targeted key's value doesn't read back
   as intended, or if any duplicate key is now reported; (b) split the old and new text into
   lines and refuse unless the line counts match and the only differing indices are the ones the
   edit intended (one for a grade/family edit, up to two for a title edit). Both checks must pass
   or nothing is written.
7. **Pre-write refusals, checked before any write is attempted:** (a) the ticket's
   `DuplicateKeys` (from `ticket.LoadAll`) is non-empty → refuse, naming the key(s) — never
   repair a duplicate by re-rendering, per the Description's constraint; (b) for `--title`
   specifically, the current H1 does not read exactly `# <id> — <current frontmatter title>` →
   refuse ("heading and frontmatter title already disagree; fix by hand first") — this ticket is
   what first makes that agreement checkable, so it must not silently paper over a ticket that
   has already drifted.
8. **No auto-appended `## History` line and no auto-edit to `## Outcome`/Description.** The
   write touches only the intended field's line(s) — an auto-History side effect would contradict
   this ticket's own Outcome ("provably touches nothing else in the file"). Rules §3's existing
   guidance (write a re-grade's *reason* into Outcome/Description by hand) is unchanged and still
   the human/agent's job.
9. **`validateTitle`/`maxTitleRuneLen` move from `internal/cli/ticket.go` into `internal/ticket`**,
   exported as `ticket.ValidateTitle`/`ticket.MaxTitleRuneLen`, so `ticket new` and `ticket set`
   share one validator (satisfies the T-038 coupling note, now that T-038 is merged). Behaviour is
   byte-for-byte unchanged — this is a relocation, not a rewrite.
10. **A successful `set` always regenerates `BOARD.md`** (impact/complexity/cost/title/family all
    render on the board), inside the same `lock.WithExclusive` critical section as the
    read-check-write, mirroring `move.Move` and `runTicketNew`.

### Tasks

#### Task 1 — Relocate title validation into `internal/ticket`

Move `maxTitleRuneLen` (const, currently `internal/cli/ticket.go:224`) and `validateTitle`
(func, currently `internal/cli/ticket.go:257`), doc comments included, into
`internal/ticket/ticket.go` as exported `MaxTitleRuneLen` and `ValidateTitle`. Update
`runTicketNew` (`internal/cli/ticket.go`) to call `ticket.ValidateTitle(title)` instead of the
now-removed local function. Update every reference in `internal/cli/cli_test.go` that names
`maxTitleRuneLen` (the cap/off-by-one/multi-byte-rune tests) to reference `ticket.MaxTitleRuneLen`
and the relocated function. No behaviour change.

#### Task 2 — Line-based frontmatter/H1 writer in `internal/ticket`

Add (in `internal/ticket/ticket.go`, or a new `internal/ticket/setfield.go` if that keeps
`ticket.go` a reasonable size — call this at implementation time):

- `var SettableFields = []string{"impact", "complexity", "cost", "family", "title"}`
- `func SetField(text, id, field, value string) (string, error)` — dispatches to a
  frontmatter-line setter for `impact`/`complexity`/`cost`/`family`, or the title+H1 setter for
  `title`; runs decision 6's parse-back guard internally; returns the candidate text or a
  refusal naming exactly what would have changed.
- internal `setFrontmatterLine(text, key, value string) (string, error)`: finds the
  `---`-delimited frontmatter block; finds the line matching key `key` inside it (reuse the
  existing key-line shape, `fmKeyRE`); rewrites it to `key: value`. If `key` is `family` and no
  such line exists, inserts a new `family: value` line immediately before the `impact:` line
  (mirrors `Scaffold`'s own key order, `internal/ticket/ticket.go:766`). If any other targeted
  key is missing, refuses (a required key can only be missing on an already-malformed ticket).
- internal `setTitleAndHeading(text, id, newTitle string) (string, error)`: enforces decision
  7b (current H1 must already equal `# <id> — <current frontmatter title>`, else refuse), then
  calls `setFrontmatterLine` for `title`, then finds and replaces the single `# <id> — …` line.
  Refuses if zero or more than one line matches that shape.

Both entry points are pure `string → (string, error)` — no filesystem, no `root` — so they are
unit-testable directly.

#### Task 3 — `internal/ticketset` package (the guarded, locked write)

New `internal/ticketset/ticketset.go`, package doc modeled on `internal/move`'s (what it
protects, what it does not).

- `type Result struct { Path, Field, Old, New string }`
- `func Set(root string, cfg *config.Config, id, field, value string) (Result, error)`:
  wraps everything in `lock.WithExclusive(root, func() error { ... })`, spanning
  load→check→write→regenerate (mirrors `move.Move`'s reasoning for why the lock must span more
  than the write alone). Inside: `ticket.LoadAll`, find by id (not found → error, same shape as
  `move.move`'s); refuse if `t.DuplicateKeys` is non-empty, naming the key(s) (decision 7a); read
  the file; call `ticket.SetField`; on success, `atomicfile.WriteFile` the result, then
  `board.Regenerate`. Populate `Result` from the ticket's path and the field's old/new value.

#### Task 4 — `pickle ticket set` CLI verb

- `internal/cli/ticket.go`: add `case "set": return runTicketSet(args[1:])` to `runTicket`'s
  switch (`internal/cli/ticket.go:22`); add a `ticketSetUsage` const alongside the existing
  `ticketNewUsage`/`ticketMoveUsage`.
- `runTicketSet(args []string) int`: parse the `<T-NNN>` positional plus five string flags
  (`impact`, `complexity`, `cost`, `family`, `title`) via `flag.NewFlagSet`; use `fs.Visit` to
  collect which of the five were actually passed on the command line; refuse (usage) unless
  exactly one was (decision 3); validate the value with the matching existing validator
  (`ticket.ValidGrade` for the three grades, `ticket.ValidID` for `family`, `ticket.ValidateTitle`
  for `title`); load config (`loadConfig()`); call `ticketset.Set`; print
  `set <id>.<field>: <old> → <new>` and the `stageLine` (reusing the existing helper,
  `internal/cli/ticket.go:81`) for the ticket's path plus `tickets/BOARD.md`.
- `internal/cli/cli.go`: add a `ticket set <T-NNN> (--impact V|--complexity V|--cost V|--family
  T-NNN|--title "<title>")` line to the `Flow commands:` help block, next to the existing
  `ticket new`/`ticket move` lines (`internal/cli/cli.go:120-128`).

#### Task 5 — Tests

- `internal/ticket/ticket_test.go`: table-driven `SetField` tests covering: each of the five
  fields against a well-formed fixture ticket string; family-insert-when-absent; refusal when a
  required targeted key is missing; refusal on a title edit when the H1 and frontmatter title
  already disagree; a fixture carrying a made-up unrecognized frontmatter key (e.g.
  `future_key: something`) that must survive an unrelated `impact`/`title` edit byte-for-byte
  (empirically proves decision 5, not just by construction); a duplicate-key fixture is *not*
  `SetField`'s concern to reject (that is `ticketset.Set`'s precondition, tested below) — confirm
  `SetField` itself has no opinion on `DuplicateKeys`.
- `internal/ticketset/ticketset_test.go`: reuse the `newProject`/`newTicket` fixture-builder
  pattern from `internal/move/move_test.go` (same shape, new package) to test: a happy path per
  field (file and `BOARD.md` both updated; `Result` populated correctly); refusal when the
  fixture ticket's frontmatter has a duplicate key (naming it); refusal on an unknown field name;
  refusal when the id does not exist.
- `internal/cli/cli_test.go`: usage-error cases (zero field flags, two field flags, an unknown
  field flag) and one happy-path invocation through `runTicketSet` against a fixture project,
  matching the existing style used for `runTicketMove`/`runTicketNew` tests in that file.

#### Task 6 — Docs (see Docs update below)

### Acceptance test

```
just build
just test
just lint

D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"
./pickle-test install --in-tree
ID=$(./pickle-test ticket new "throwaway" --project pickle | awk '/^created/{print $2}')
./pickle-test ticket set "$ID" --impact high
# → frontmatter impact: is now "high"; git diff on the ticket file touches only that one line
./pickle-test ticket set "$ID" --title "renamed throwaway"
# → frontmatter title: and the H1 both read "renamed throwaway"; filename unchanged;
#   git diff on the ticket file touches only those two lines
./pickle-test ticket set "$ID" --impact high --cost L; echo "exit=$?"
# → refused (usage error): exactly one field flag is required; exit non-zero
./pickle-test board audit; echo "exit=$?"
# → exit 0, no errors or warnings introduced by the two sets above
```

Use the throwaway-dir + `pickle-test`-renamed-binary convention (`AGENTS.md`'s self-modify
policy) — never the in-repo binary path, never against this repo's own tree.

### Docs update (mandatory when user-facing)

- `internal/cli/cli.go`: the `Flow commands:` help block (Task 4).
- `docs/user-manual/cli-reference.adoc`: add a `pickle ticket set` row to the Overview table
  (next to the existing `pickle ticket new`/`pickle ticket move` rows, ~line 54-58); add
  `ticket set` to the intro paragraph's list of lock-taking write commands (~line 7-8); add a new
  `[#cmd-ticket-set]` / `== \`pickle ticket set\`` section after `== pickle ticket move`
  (~line 784), mirroring that section's structure — synopsis, what it guards (decisions 6/7),
  the refusal list, and an example `git add` stage-line output like `ticket new`'s.
- `skill/resources/tickets-README.md` §3: update the **Priority** bullet's re-grade sentence and
  the **Families** bullet's closing sentence ("it is otherwise set by hand-editing frontmatter,
  like `depends-on:`") to name `pickle ticket set` for `impact`/`complexity`/`cost`/`family`/
  `title`, and state that `depends-on:` alone remains a hand-edit.
- `docs/user-manual/concepts/tickets.adoc`: check the title/filename discussion (~line 20) and
  add a pointer to `pickle ticket set --title` if that section already describes how a title is
  set or changed.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated and registered (above).
3. Write a summary of everything done (files touched, decisions made, anything deferred).
4. Suggest a Conventional Commit message, ticket id in brackets at the end of the subject, e.g.
   `feat(ticket): add pickle ticket set — guarded single-field edit (T-102)`.
5. Tidy WIP commits into a small number of atomic, correctly typed/scoped commits before
   presenting them (root-path child, `path = "."`).
6. Commit locally on the ticket branch. Do not push or open a merge request without explicit
   user approval. Under `layout = "in-tree"`, before pushing verify the remote base is not
   behind (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`
   must print nothing, or push `origin main` first) — then push and open the merge request.
   Merging is always the human's. Hand back to the user.

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
- 2026-08-24 — refined: wrote the Implementation Plan (new `internal/ticketset` package + a
  line-based `ticket.SetField`/H1 writer in `internal/ticket`, a `pickle ticket set` CLI verb,
  and the docs it needs). Re-graded impact `low-medium` → **low**: against the current backlog
  (T-075–T-079, all medium/high) this is dev-tooling ergonomics only, not user-facing capability—
  collapsing the range per rules §3 now that scope is concrete. complexity/cost unchanged
  (medium/M). T-060 coupling note resolved as moot: `pickle ticket renumber` was considered and
  dropped, so there is no prior shape to check.
- 2026-09-03 — TO DO → READY: plan complete
- 2026-09-03 — READY → IN DEVELOPMENT: picked up
- 2026-09-03 — IN DEVELOPMENT → IN REVIEW: acceptance green
