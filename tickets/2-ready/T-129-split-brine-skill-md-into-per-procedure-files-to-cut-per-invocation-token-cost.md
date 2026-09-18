---
id: T-129
title: split brine SKILL.md into per-procedure files to cut per-invocation token cost
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-129 — split brine SKILL.md into per-procedure files to cut per-invocation token cost

## Outcome

`/brine` no longer injects the full skill body on every invocation. The `make it a ticket`,
`rework a ticket`, and `audit the board` procedures move into their own `resources/` files,
read only when that specific procedure applies; `SKILL.md` shrinks by about a quarter and every
`validate`/`implement`/`refine` call (the large majority of observed traffic) pays less
fresh-token cost per turn.

## Description

Every `/brine` invocation injects the entirety of `skill/SKILL.md` into context regardless of
which of the six procedures the trigger phrase names — there is no partial loading today.
Local session-log analysis (last ~24h, this repo plus the `messgr` child-project, ~77 observed
invocations total) found the six procedures are hit very unevenly: `validate`/`review ticket`
(~29 calls) and `implement ticket` (~25) dominate, `refine ticket` is a distant third (~18),
and `rework ticket` (~4), `make it a ticket` (~1), and `audit the board` (~0) are rare. The three
rare procedures total 86 of the file's 342 lines (~25%) but load on every single call, including
the ~94% of calls that only ever needed `validate`/`implement`/`refine`.

The fix: move `## Procedure: make it a ticket`, `## Procedure: rework a ticket`, and
`## Procedure: audit the board` verbatim into `resources/procedure-make-ticket.md`,
`resources/procedure-rework.md`, and `resources/procedure-audit-board.md` respectively —
mirroring the pattern already used for `resources/tickets-README.md` and `resources/TEMPLATE.md`
(read on demand, single source of truth, never copied into a project). Replace each moved
section in `SKILL.md` with a one-line pointer ("Read `resources/procedure-X.md` and follow
it.") and list the three new files in `SKILL.md`'s "Bundled resources" section. Leave
`validate`/`implement`/`refine` inline — they're the hot path and would load on nearly every
call regardless, so splitting them buys little while adding an extra Read hop.

No source-code change: this only restructures the skill payload under `skill/`, which
`go:embed all:skill` in `assets.go` already embeds recursively, so no embed-directive or
Go-code edit is needed. `payload_lint_test.go`'s foreign-workspace checks and
`install_test.go`'s payload-content assertions (`TestPayloadDispositionVocabulary`,
`TestPayloadDefersToProjectConfig`) apply to the new files exactly as they do to the existing
ones and must keep passing.

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/T-129-split-brine-skill-cold-procedures
```

Root-path child (`path = "."`): tidy WIP commits into atomic ones before presenting, keep that
history over squashing at approval time.

### Prerequisite gate (hard)

none

### Confirmed design decisions (do not deviate without asking)

1. **Only the three coldest procedures move out — `make it a ticket`, `rework a ticket`,
   `audit the board`.** Session-log analysis (see Description) found these cover ~5 of ~77
   observed `/brine` calls; `validate`/`implement`/`refine` (94% of traffic) stay inline in
   `SKILL.md` since they'd load on nearly every call regardless, so splitting them buys little
   for an extra Read hop each time.
2. **Moved content is relocated verbatim, not rewritten.** The point is to change where the
   bytes live, not to re-litigate the procedures' wording; a content rewrite would conflate two
   different changes in one diff and complicate review.
3. **The pointer left behind in `SKILL.md` is a single line**: `Read
   \`resources/procedure-<name>.md\` and follow it.` — consistent with how the skill already
   points at `resources/TEMPLATE.md` and `resources/review-protocol.md` elsewhere in the file.
4. **No Go code changes.** `assets.go`'s `//go:embed all:skill` already embeds `skill/`
   recursively, so new files under `skill/resources/` need no embed-directive edit.

### Tasks

#### Task 1 — extract the three cold procedures into resource files

Create, verbatim from the current `skill/SKILL.md`:
- `skill/resources/procedure-make-ticket.md` (body of `## Procedure: make it a ticket`)
- `skill/resources/procedure-rework.md` (body of `## Procedure: rework a ticket`)
- `skill/resources/procedure-audit-board.md` (body of `## Procedure: audit the board`)

Each keeps its original `# Procedure: <name>` heading text (as an `#` top-level heading in the
new file, since it is now the whole document) and drops nothing.

#### Task 2 — replace the moved sections in SKILL.md with pointers

In `skill/SKILL.md`, replace the body of each of the three `## Procedure: …` sections with:
`Read \`resources/procedure-<name>.md\` and follow it.` — keep the `## Procedure: …` heading
itself (the "When to use" table already points here by name) and keep
`validate`/`implement`/`refine`/`refine a ticket` untouched.

#### Task 3 — list the new files in "Bundled resources"

Add the three new files to the bulleted list near the top of `skill/SKILL.md` (alongside
`resources/tickets-README.md`, `resources/TEMPLATE.md`, etc.), noting they are read only when
that procedure applies.

### Acceptance test

```
just test   # go test ./...  — must stay green, in particular:
            #   - payload_lint_test.go (foreign-workspace lint over the new files)
            #   - internal/install/install_test.go's TestPayloadDispositionVocabulary and
            #     TestPayloadDefersToProjectConfig (payload-content assertions)
just lint
wc -c skill/SKILL.md   # expect a reduction of roughly a quarter from the pre-change size
```
Manually confirm `skill/SKILL.md` still contains exactly six `## Procedure: ` headings and that
each of the three new resource files' content matches what was removed (no accidental
truncation or duplication) via `git diff` review.

### Docs update (mandatory when user-facing)

no user-facing surface — this is an agent-facing skill payload restructuring with no CLI flag,
config key, or user-manual page affected.

### Finish (mandatory)

1. Acceptance test green; `just lint` clean.
2. No docs to register (see above).
3. Summary: files touched are `skill/SKILL.md` (trimmed) and three new files under
   `skill/resources/`; nothing else.
4. Suggested commit message:
   ```
   refactor(skill): split brine's cold procedures into resource files (T-129)

   make it a ticket, rework a ticket, and audit the board covered ~5 of ~77
   observed /brine calls but loaded on every invocation. Move them into
   resources/procedure-*.md, read on demand, and leave the hot-path
   procedures (validate/implement/refine) inline.
   ```
5. Root-path child: tidy WIP commits into one atomic commit before presenting.
6. Commit locally; present the commit message; publish only after user approval (this repo's
   commit policy is publish-gated).

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-18 — created (TO DO). source: chat: user asked to explore splitting the brine skill
  to cut usage after reviewing a Claude Code usage summary showing `/brine` at 24% of local
  usage; session-log analysis of trigger-phrase frequency and per-call token cost confirmed a
  concrete, narrow win before filing.
- 2026-09-18 — TO DO → READY: plan complete
