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

- [x] Reviewer independence settled (step 0): **delegated**. The reviewing agent (this session)
  authored the branch, so the implementation/quality/consistency/documentation audits (steps
  2–4a) were run by a fresh, adversarially-briefed sub-agent with no memory of writing the code,
  given the ticket as read from `main`, the branch to audit, and the child's configured
  commands. Every delegated finding below was independently re-verified by hand (repro commands
  re-run myself) before being recorded — one delegated finding (the blocking one, F0) was found
  only during that re-verification and is not in the delegate's own report.
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (steps 1, 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass: **skipped** — already run once during implementation
  (`docs_readability` against the three changed `.adoc`/`.md` files); every suggestion it
  returned was pre-existing prose outside this ticket's diff, none applied. Not re-run at review
  time since the changed files gained only the two inline prose fixes below (F1, F5) since then.
- [x] Findings recorded with severity, class, disposition (step 5)
- [x] Ticket moved (step 6) — **two rework rounds**: round 1 (F0, commit `870d840`), round 2 (G0,
  found while scoped-re-reviewing round 1's own fix, commit `1c8825d`). Both scoped re-reviews
  delegated for the same reason as the original audits; each fix independently re-verified by
  hand (repro commands re-run against both the fixed and the pre-fix commit) before being
  recorded. No blocking findings remain after round 2.
- [x] Other references reconciled (step 7) — re-checked after round 2: unaffected, still holds
- [x] Remaining-tickets impact sweep done (step 8) — re-checked after round 2: unaffected, still holds
- [x] Summary + commit message & MR attributes presented (step 9)

**Implementation audit.** `just build`, `just test`, `just lint`, `just docs-check` all green.
The plan's own acceptance-test script was re-run verbatim (against a throwaway install, per
`AGENTS.md`'s self-modify policy) and every step matched: grade edit touches exactly one line,
title edit touches exactly `title:` + the H1, the two-flag call is refused, `board audit` stays
clean. Every task, and confirmed decisions 1–10, verified against the actual code — met, with
one exception surfaced as F0 below (decision 7b's H1/frontmatter-title comparison is unsound for
a legally whitespace-padded title, not merely "unverified").

**Impact sweep (step 8).** No `2-ready/`/`1-to-do/` ticket lists T-102 in `depends-on:`. T-079's
Description cross-references T-102 as the "tickets/-side sibling" of its own lifecycle-field
guard — still accurate, no patch needed. `tickets/NOTES.md`'s historical triage table entry
naming T-102 is append-only record, not a live assumption to correct.

**Other references (step 7).** No governing document (`DESIGN.md`, `AGENTS.md`'s own rules) made
a claim this branch falsified. Two prose staleness findings in the *shipped* docs tree are
recorded as F1/F5 below (already fixed inline, since both are pure prose with no behaviour
change).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F0 | blocking | correctness | — | `pickle ticket set --title` refuses to retitle **any** ticket whose *original* title has leading/trailing whitespace — a title shape `ticket.ValidateTitle` legally permits — even when the ticket was never hand-edited and H1/frontmatter were written consistently by `Scaffold`. `setTitleAndHeading` (`internal/ticket/setfield.go`) compares the H1's raw, untrimmed captured title against `parseFrontmatter`'s **trimmed** `fm["title"]`; a frontmatter `key: value` line can never preserve the value's own leading whitespace distinguishably from the mandatory separator space (`fmKeyRE`'s `\s*` eats both alike), so the two sides disagree on any padded title regardless of whether the ticket ever drifted. | Repro (throwaway install): `ticket new "  padded title  " --project demo` (legal, `ValidateTitle` only rejects empty/newline/`"---"`/over-length) then `ticket set T-001 --title "clean new title"` → `heading "  padded title  " and frontmatter title "padded title" already disagree; fix by hand first`, on a ticket that was never hand-edited. | Compare both sides on the same basis — trim both the H1 capture and the frontmatter title before the decision-7b equality check — so the check tests title *content* agreement, not an artifact of the frontmatter line format's inherent inability to distinguish a value's own leading whitespace from the separator. |
| F1 | non-blocking | other | fixed inline | `setfield.go`'s package doc claimed a duplicate of an unrelated (non-targeted) frontmatter key is "invisible to this function by design"; empirically `verifyFrontmatterEdit` re-parses the *whole* frontmatter block, so it still detects and refuses on such a duplicate — safe, but the comment overstated what callers must guarantee. | Ad hoc probe: `SetField` on a fixture with `impact` targeted but `complexity` duplicated → refused, naming `complexity`. | Comment corrected in place to describe the guard's actual (stricter) behaviour; no code change. |
| F2 | non-blocking | design | noted | `fmKeyRE` (`^([A-Za-z-]+):\s*(.*)$`, pre-existing, unchanged by this branch) does not match an underscore-named key (e.g. the `schema_version` example this ticket's own package doc uses). A *duplicate* of such a key would bypass both `LoadAll`'s `DuplicateKeys` and `ticketset.Set`'s precondition — though the line-based writer still never corrupts or drops such a key either way, since copy-through is index-based, not regex-dependent. | `grep -n 'fmKeyRE\s*=' internal/ticket/ticket.go` (line predates this branch; `git diff main..feat/T-102-... -- internal/ticket/ticket.go` shows no change to it). | Pre-existing, shared with `board audit`'s own duplicate detection — not this ticket's to fix; noted for whoever next touches `fmKeyRE`. |
| F3 | non-blocking | correctness | noted | `pickle ticket set --family` does not trim the value before validating it, unlike `pickle ticket new --family` (`strings.TrimSpace` in `runTicketNew`, absent from `runTicketSet`) — a padded id that `ticket new` accepts is refused by `ticket set` with a literal "is not a ticket id" message. Fails closed (no corruption), just an ergonomics inconsistency between two commands sharing the same validator. | Throwaway install: `ticket new … --family " T-001 "` → stores `family: T-001` (trimmed); `ticket set T-003 --family " T-001 "` → refused, `--family: " T-001 " is not a ticket id`. | One-line `strings.TrimSpace` in `runTicketSet` — a behaviour change, so not eligible for `fixed inline`; left for whoever next touches this command. |
| F4 | non-blocking | stale-xref | fixed inline | `docs/user-manual/cli-reference.adoc`'s `pickle serve` section said `ticket new`, `ticket move` and `board sync` "remain the only writers" — stale now that `ticket set` also writes (confirmed: every non-no-op `ticket set` triggers a full `board.Regenerate`). | `docs/user-manual/cli-reference.adoc` (the `pickle serve` §, near "It never writes"). | Line corrected in place to add `pickle ticket set` and drop "only" (the safety claim — still lock-protected — is unaffected); no behaviour change. |
| F5 | non-blocking | stale-xref | noted | Six other locations phrase `BOARD.md`'s regeneration as "`ticket new`, `ticket move` and `board sync`" without `ticket set` (already omitting `project add`/`project remove` before this branch, so not a new gap in kind, only in degree): `internal/board/board.go:363` (the header rendered into every generated `BOARD.md`), `internal/install/install.go:1399-1400` (the `AGENTS.md` marker-block payload) and `:1411` (the `tickets/README.md` payload), `internal/board/board_test.go:130` (fixture), `skill/SKILL.md:137-138`, and this repo's own hand-mirrored `AGENTS.md:111-112`. | `grep -rn "pickle board sync" internal/board/board.go internal/install/install.go internal/board/board_test.go skill/SKILL.md AGENTS.md` | Multi-file and touches `install.go`'s baked-in payload plus (per this repo's self-modify policy) a hand-mirrored edit to this repo's own `AGENTS.md` marker block — too broad and too sensitive for an inline fix during this review, and six near-identical one-word insertions don't clear the promotion test for a dedicated ticket on their own; noted for whoever next touches the enumeration (e.g. alongside a future ticket that also adds a `BOARD.md` writer). |
| F6 | non-blocking | docs-gap | noted | `pickle changelog check` flags T-102 as shipped but not named in `CHANGELOG.md`'s "Unreleased" section. | `go run . changelog check` → `1 candidate(s) shipped but not named in "Unreleased": T-102`. | Same precedent as T-038's own review F7: the tool is advisory by design ("the entry may legitimately be written any time before the release"), and `RELEASING.md` opens by running it, so a release catches this — not a review-time gate. |

Disposition summary (round 1): 1 blocking (F0, resolved via rework), 2 fixed inline (F1, F4), 4
noted (F2, F3, F5, F6 — F6 citing established precedent from T-038's own review). No folded, no
new ticket — nothing here clears the promotion test alone.

### Rework fix record — round 1 (commit 870d840)

F0 fixed. Root cause was one level broader than first filed: the trim asymmetry breaks **both**
directions, not only reading a pre-existing padded title. `verifyFrontmatterEdit`'s post-write
check (`fm[key] != value`) also refused setting a **new** padded title on an otherwise pristine,
never-touched ticket, since `parseFrontmatter` always normalizes whatever it re-parses. Both
manifestations share the same fix: a new `normalizeFrontmatterValue` helper (trim whitespace,
then a surrounding quote char — exactly what `parseFrontmatter`'s own per-line scan already does)
is applied to both sides of both comparisons — `setTitleAndHeading`'s H1-vs-frontmatter precheck,
and `verifyFrontmatterEdit`'s re-parsed-vs-intended check. Two new regression tests added
(`TestSetFieldTitleAcceptsPaddedOriginalTitle`, `TestSetFieldTitleAcceptsANewPaddedTitle`);
`TestSetFieldTitleRefusesWhenH1AndFrontmatterDisagree` (genuine drift, no whitespace involved)
re-confirmed still refusing correctly. Re-ran the acceptance test verbatim plus the exact F0 repro
and its new-padded-title counterpart against a throwaway install — both now succeed; `just build`,
`just test`, `just lint`, `just docs-check` all green. No other finding touched — F1–F6 stand as
recorded above.

### Scoped re-review — round 1's fix, reviewer independence

**Delegated again**: the fix commit above was authored by the same session doing this review, so
the scoped re-review (confirm F0 resolved; read the round-1 fix's own diff for new defects, per
`resources/review-protocol.md` §1) was run by a fresh, adversarially-briefed sub-agent with no
memory of writing the fix, given the ticket's `## Review` section, the branch, and the fix commit
to read. It confirmed F0 resolved (both manifestations, plus the genuine-drift path still
correctly refusing) and surfaced one new finding, G0, from reading the fix's own diff —
independently re-verified by hand below (including reproducing the pre-fix-vs-post-fix behaviour
difference on the actual pre-fix commit) before being recorded. The delegate classed G0
non-blocking; re-classified **blocking** here, since it contradicts confirmed decision 7
("must not silently paper over a ticket that has already drifted") rather than merely being a
quality nit — severity is the orchestrating reviewer's call, not the delegate's, per the
protocol's step 0 boundary.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| G0 | blocking | correctness | — | Round 1's fix over-widened `setTitleAndHeading`'s H1-vs-frontmatter drift precheck: it compared the H1's raw capture, normalized (whitespace **and** a surrounding quote char stripped), against `parseFrontmatter`'s already quote-stripped `fm["title"]`. A genuine quote-boundary disagreement between the two copies (H1 quoted, frontmatter not — a plausible result of hand-adding YAML-style quoting to only one copy) was silently treated as agreement instead of refusing, contradicting confirmed decision 7's "must not silently paper over drift". | Throwaway install, commit `870d840`: ticket with frontmatter `title: quote drift victim` and hand-edited H1 `# T-001 — "quote drift victim"` (quotes added to H1 only). `ticket set T-001 --title renamed` → `exit=0`, silently rewrites both lines, no refusal. Re-ran the identical repro against the pre-fix commit `7c2e53e` → correctly refused (`heading "\"quote drift victim\"" and frontmatter title "quote drift victim" already disagree`), confirming this is a regression introduced by the round-1 fix, not pre-existing. | Compare the drift precheck's two sides on raw text, whitespace-trimmed only (not quote-stripped) — that is the one ambiguity F0 actually needs neutralised (a frontmatter line cannot tell a value's own leading whitespace from the mandatory separator space); a boundary quote character carries no such ambiguity in either copy and a genuine difference there must still refuse. Keep `verifyFrontmatterEdit`'s own check (confirming a write reads back as intended) on the full `normalizeFrontmatterValue`, since that comparison is against what a real re-parse actually reports, which does fully normalize. |

Disposition summary (round 1 re-review): 1 blocking (G0, resolved via a second rework round). No
other finding — F1–F6 above are unaffected and still stand as recorded.

### Rework fix record — round 2 (commit 1c8825d)

G0 fixed. `setTitleAndHeading`'s drift precheck no longer routes through `fm["title"]` or the
full `normalizeFrontmatterValue`: it now reads the frontmatter title's raw capture directly (via
`fmKeyRE`, the same regex `findFrontmatterKeyLine` already uses to locate the line) and compares
it against the H1's raw capture with `strings.TrimSpace` only on both sides — symmetric,
whitespace-only normalization, no quote-stripping. `verifyFrontmatterEdit`'s own check (the
round-1 fix) is untouched: it still uses the full `normalizeFrontmatterValue`, correctly, since
it compares against what a real re-parse actually reports. Updated `normalizeFrontmatterValue`'s
doc comment to state it is deliberately *not* used by the drift precheck anymore, and to fix a
found wording error ("trim one surrounding quote character" → `strings.Trim` removes every
matching character in the cutset, not one). New regression test
`TestSetFieldTitleRefusesOnQuoteBoundaryDrift`; re-ran all four cases end to end against a
throwaway install — F0's two manifestations still fixed, the genuine-drift (differing words) case
still refuses, and the new quote-boundary-drift case now refuses too. `just build`, `just test`,
`just lint`, `just docs-check` all green.

cost: estimated M, actual M

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
- 2026-09-03 — IN REVIEW → REWORK: F0 blocking: --title refuses any padded original title, even undrifted
- 2026-09-03 — REWORK → IN REVIEW: findings fixed
- 2026-09-04 — IN REVIEW → REWORK: G0 blocking: round-1 fix's drift precheck masked a quote-boundary disagreement, contradicts decision 7
- 2026-09-04 — REWORK → IN REVIEW: findings fixed
- 2026-09-04 — IN REVIEW → DONE: review PASS after two rework rounds (F0, G0); no blocking findings remain
- 2026-09-04 — merged to main (a4bb5d0, PR #83); all 7 commits kept as history rather than squashed
  (the root-path default, rules §0) — the branch was tidied before publishing, squashing the two
  review-round fix commits into one since the second corrected a defect the first introduced
