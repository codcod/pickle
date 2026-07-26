---
id: T-038
title: tighten ticket new's title contract: Unicode line terminators and length cap
project: pickle
depends-on: []
spawned-by: [T-030]
impact: low-medium
complexity: low
cost: S
---

# T-038 — tighten ticket new's title contract: Unicode line terminators and length cap

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
whitelist. Update the input contract in the three places T-030 documented it: `README.md`,
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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: pickle ticket new
