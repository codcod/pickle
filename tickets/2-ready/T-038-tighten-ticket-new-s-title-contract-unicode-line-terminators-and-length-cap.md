---
id: T-038
title: tighten ticket new's title contract: Unicode line terminators and length cap
project: pickle
depends-on: []
spawned-by: [T-030]
impact: low
complexity: low
cost: S
---

# T-038 — tighten ticket new's title contract: Unicode line terminators and length cap

## Outcome

After this ships, `pickle ticket new` rejects a title containing a Unicode line terminator or an implausibly long string, the same way it already rejects `\n`/`\r` — closing the two gaps the T-030 review found without changing behaviour on any title that was already accepted.

## Description

**T-030** closed the newline-injection hole in `pickle ticket new` by rejecting titles containing
`\n` or `\r` before anything is written. Its independent review (2026-07-26) found two gaps left at
that boundary, both reproduced against the built binary. Neither was blocking — T-030's own decision
1 enumerated exactly `\n`/`\r`, so it was honoured to the letter — but the *rationale* behind that
decision ("a newline would inject frontmatter keys") is satisfied by more characters than the check
lists.

### 1. Unicode line terminators still inject a frontmatter key (review finding N1)

`validateTitle` (`internal/cli/ticket.go:160`) is `strings.ContainsAny(title, "\n\r")`. `U+0085`
(NEL), `U+2028` (line separator) and `U+2029` (paragraph separator) pass it. Measured:

```
pickle ticket new "$(printf 'a\u0085project: nope')" --project demo
→ created T-001, exit 0
   title: a<U+0085>project: nope
   pickle board audit: 1 tickets, 0 error(s), 0 warning(s)
   Ruby Psych:  {"id"=>"T-001", "title"=>"a", "project"=>"demo", ...}
```

Pickle's own `ParseFrontmatter` splits on `"\n"` only (`internal/ticket/ticket.go:110`), so **no
pickle behaviour changes** — the H1 and the board row stay on one physical line and the audit is
rightly clean. But YAML 1.1 readers (Ruby Psych, PyYAML — i.e. Jekyll, Obsidian, most static-site
frontmatter tooling) treat all three as line breaks. To them the title is silently **truncated** to
`"a"` and a phantom `project:` key appears. That is precisely the duplicate-key corruption T-030
exists to prevent, reached through a terminator the check does not enumerate.

The fix is a blacklist, not the character whitelist T-030 Task 3 item 4 forbids: extend the check to
`{'\n','\r','\u0085','\u2028','\u2029'}`, and consider whether `unicode.IsControl` is the better
predicate (it would also catch `\v`/`\f`, which were measured harmless — the slug collapses them and
the frontmatter stays on one line — so that is a judgement call, not a fix).

### 2. An over-long title fails with a raw syscall error (review finding N2)

```
pickle ticket new "$(python3 -c "print('a'*250)")" --project demo
→ pickle: open /…/tickets/1-to-do/T-004-aaa….md: file name too long
```

Exit 1 and the tree stays consistent — no file, no board row, audit clean — because `os.WriteFile`
precedes `board.AddTODORow`. So this is a message-quality bug, not a correctness one. But the guard
is the OS's `NAME_MAX` rather than a validated contract, which makes the boundary platform-dependent
(and it leaks an absolute path plus an `open …` phrase, where every other rejection names what is
wrong with the input). Cap the slug or title length in `validateTitle`/`Slugify` — roughly 120 runes
keeps filenames readable — and emit a title-contract error.

### Scope

Both are one-boundary changes in the same function that T-030 created, which is why they are one
ticket. Extend `validateTitle` (`internal/cli/ticket.go:139-171`), extend
`TestTicketNewRejectsInjectionInTitle`'s hostile table, and keep
`TestTicketNewAcceptsAwkwardButLegalTitle` green — the rejection must not become a character
whitelist. Update the input contract in the three places T-030 documented it (README's copy moved to the
manual by T-047): `docs/user-manual/cli-reference.adoc` (`ticket new` section),
`skill/SKILL.md`, `skill/resources/tickets-README.md` §3.

Explicitly out of scope: everything T-030 deferred and the review re-confirmed as correctly
deferred — board-cell escaping (**T-014**), the row cell-count check and insert-point wall
(**T-034**), audit-side duplicate frontmatter keys (**T-033**), audit-side id-shape validation
(**T-027**).

### Couplings

`spawned-by: [T-030]` — both findings come from its review.

Soft couplings (no `depends-on`, no ordering enforced):

- **T-030** — must be merged before this is picked up, since it creates the function being extended.
- **T-033** — the audit-side detection of duplicate frontmatter keys would *catch* finding 1's
  artifact rather than prevent it. Complementary; neither blocks the other.
- **T-014** — if it makes board cells escape-aware, re-check whether an exotic terminator can still
  reach a cell.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-038-tighten-title-contract
```

Root-path child. Tidy WIP commits before presenting (Finish, below).

### Prerequisite gate (hard)

None (`depends-on: []`). `spawned-by: [T-030]` — T-030 is `6-done/` and merged (it created
`validateTitle`, the function this ticket extends). T-033/T-014/T-027/T-034, named in the
Description as correctly-deferred siblings, are all now `7-dropped/` — confirmed at refinement
this does not change this ticket's scope (they were out of scope regardless of their own fate).

### Confirmed design decisions (do not deviate without asking)

1. **Finding 1: an explicit 5-rune blacklist, not `unicode.IsControl`.** Extend
   `strings.ContainsAny(title, "\n\r")` to `strings.ContainsAny(title, "\n\r\u0085\u2028\u2029")`
   (Go's `ContainsAny` decodes both operands as runes, so this needs no loop). `unicode.IsControl`
   would also reject `\v`/`\f`/`\t`/other C0-C1 controls that the Description's own measurement
   found harmless (`\v`/`\f` specifically: the slug collapses them and the frontmatter stays on
   one physical line) and that review never asked about — rejected as scope creep beyond the two
   findings this ticket closes. Mirrors T-030 decision 1's own precedent: enumerate the unsafe
   characters, never a whitelist.
2. **Finding 2: cap the raw title (rune count), not the derived slug, at 120 runes.** A rune-count
   cap on `title` structurally bounds the slug too (`Slugify` only removes/collapses runes, never
   adds), so capping the input is sufficient to keep `id-slug.md` well under any OS's `NAME_MAX`
   without needing a second check downstream. 120 matches this repo's own existing convention for
   "how long is a single readable line" (`internal/board/board.go`'s `maxCellRunes = 120`,
   verified at refinement) — not reused directly (different package, different concern: a hard
   rejection here, a truncate-with-ellipsis there), but the same number for the same underlying
   reason. Counted with `utf8.RuneCountInString`, not `len()` (byte length would under-cap a
   title with any multi-byte rune).
3. **The error message names the contract, not the filesystem.** "title exceeds 120 runes (it
   becomes part of a filename)" — no path, no `open`/`os` wording — closing finding 2's stated
   leak (an absolute path in a raw syscall error).
4. **`TestTicketNewAcceptsAwkwardButLegalTitle` stays green with zero edits** (verified at
   refinement: all four existing cases are short, plain-ASCII-terminator-free) — confirms this
   fix is a boundary tightening, not a new whitelist.

### Tasks

#### Task 1 — extend the newline blacklist (finding 1 / N1)
In `internal/cli/ticket.go`'s `validateTitle` (func at `:219`, doc comment `:205-217`), change
the `strings.ContainsAny` call per decision 1 and extend the doc comment to name the three
Unicode line terminators and cite the YAML-1.1-reader risk (Ruby Psych, PyYAML) from the
Description's own measurement.

#### Task 2 — add the length cap (finding 2 / N2)
In `internal/cli/ticket.go`, add `const maxTitleRuneLen = 120` near `validateTitle` and a check
(using `unicode/utf8.RuneCountInString`, new import) that rejects a title longer than that with
the decision-3 wording, before the newline/`---` checks or after — order doesn't matter, none of
the checks are mutually exclusive gates.

#### Task 3 — tests
In `internal/cli/cli_test.go`:
- Add four cases to `TestTicketNewRejectsInjectionInTitle`'s table (`:752-763`): `"NEL line
  terminator"` (`"a\u0085project: nope"`, the Description's own N1 repro), `"line separator"`
  (`"a\u2028project: nope"`), `"paragraph separator"` (`"a\u2029project: nope"`), and
  `"over-length title"` (`strings.Repeat("a", 121)`) — all asserted the same way the table
  already does (exit code, no ticket written, board unchanged).
- Add `TestTicketNewOverlongTitleErrorNamesTheContract`: `captureStderr` around the over-length
  case, asserting the message contains `"120"`/`"exceeds"` (decision 3's wording) and does
  **not** contain `"file name too long"` or an absolute path separator run (`os.PathSeparator`
  repeated, or simply assert it does not contain `t.TempDir()`'s own root path) — this is the
  regression proof for finding 2's stated leak.
- Confirm `TestTicketNewAcceptsAwkwardButLegalTitle` needs no edits (decision 4) by running it
  unchanged.

#### Task 4 — update the three documented copies of the title contract
- `docs/user-manual/cli-reference.adoc`'s `ticket new` section (`:683-684`, verified at
  refinement): extend "may not be empty or contain newlines" to name the Unicode line
  terminators and the 120-rune cap.
- `skill/resources/tickets-README.md` §3's "Filename" bullet (`:271-274`, verified at
  refinement): add a foreign-workspace-safe clause — no internal file/line references, no
  implementation detail beyond what a project *not* pickle needs ("and reasonably short:
  `pickle ticket new` rejects one past ~120 runes, since it becomes a filename and a rendered
  table cell") — per `AGENTS.md`'s foreign-workspace test. Do **not** enumerate the specific
  Unicode code points here; "a title is a single line of text" already covers that abstractly
  and correctly, and adding `\u0085`-style detail would be exactly the kind of implementation
  trivia the payload doesn't need.
- `skill/SKILL.md` (`:156-164`, verified at refinement): **no change** — its "title must be a
  single line" wording is already accurate and appropriately abstract; confirmed at refinement,
  recorded here so a reviewer does not wonder why this file is untouched while its sibling
  changed.

### Acceptance test

```
just build
go test ./internal/cli/... -v -run 'TestTicketNewRejectsInjectionInTitle|TestTicketNewAcceptsAwkwardButLegalTitle|TestTicketNewOverlongTitle'
just test
just lint
just docs-check
```
All clean. Manually re-run the Description's own two repro commands against a locally built
binary in a throwaway project (per `AGENTS.md`'s self-modify policy) to confirm both now exit
non-zero with a title-contract message and write no file:
```
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree --project demo
./pickle-test ticket new "$(printf 'a\u0085project: nope')" --project demo
./pickle-test ticket new "$(python3 -c "print('a'*250)")" --project demo
```

### Docs update (mandatory when user-facing)

Task 4: `docs/user-manual/cli-reference.adoc` and `skill/resources/tickets-README.md` (the
second ships in the payload — run the `payload_lint_test.go` suite, already covered by `just
test`, to confirm the new sentence passes the foreign-workspace mechanical checks).

### Finish (mandatory)

1. Acceptance test green, including the two manual repro commands.
2. Docs updated (Task 4); `skill/SKILL.md` deliberately left untouched (decision noted in the
   summary so it reads as a decision, not an oversight).
3. Write a summary naming both fixed findings (N1, N2) and confirming
   `TestTicketNewAcceptsAwkwardButLegalTitle` needed no changes.
4. Suggested commit message:
   ```
   fix(cli): reject Unicode line terminators and over-long titles in ticket new (T-038)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-038 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: pickle ticket new
- 2026-07-26 — patched by the T-047 review (impact sweep): README passage it cited moved to docs/user-manual/cli-reference.adoc
- 2026-08-20 — refined: confirmed T-030 done/merged and the four correctly-deferred siblings
  (T-033/T-014/T-027/T-034) are now all `7-dropped/` — no scope change either way. Decided the
  5-rune explicit blacklist over `unicode.IsControl` (scope discipline: fix exactly what review
  found) and a 120-rune cap on the raw title, not the derived slug (structurally bounds both);
  120 matches this repo's own existing `board.maxCellRunes` convention. Verified
  `TestTicketNewAcceptsAwkwardButLegalTitle` needs no changes. Decided `skill/SKILL.md` needs no
  edit (already abstract/foreign-workspace-safe) while `tickets-README.md` §3 gets one
  foreign-workspace-safe clause. TO DO → READY: implementation plan complete.
- 2026-08-22 — TO DO → READY: plan complete
