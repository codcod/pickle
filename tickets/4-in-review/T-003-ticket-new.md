---
id: T-003
title: ticket new (id allocation + template + board row)
project: pickle
depends-on: [T-001]
impact: high
complexity: medium
cost: M
---

# T-003 — ticket new (id allocation + template + board row)

## Description

Implement `pickle ticket new "<title>" --project <name> [--impact .. --complexity .. --cost ..]`.

Behaviour:

- allocate the next `T-NNN` = `max(existing across all status dirs) + 1` (one global namespace);
- instantiate the embedded `skill/resources/TEMPLATE.md` into `tickets/1-to-do/T-NNN-<slug>.md`
  with `id`, `title`, and `project:` set, and the grade fields filled (accept adjacent-pair
  ranges; default to sensible ranges when a flag is omitted);
- add the board row under that child's `### <child>` sub-group in the TO DO section, in impact
  order;
- write the first `created (TO DO)` History line.

Fail clearly if `--project` is not a registered child (needs T-001). The CLI guarantees id +
target + placement + board sync; the agent fills the Description prose afterwards. Phase P1.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # child-project 'pickle' is the repo root
git checkout main
git checkout -b feat/T-003-ticket-new
```
Local WIP commits fine; **no push / no MR without user approval**.

### Prerequisite gate (hard)

`depends-on: [T-001]` — T-001 done **and merged** (cdad65e). Clean tree on `main`. (The board
parse/model layer from T-002 is also merged and is reused here, though not a hard dependency.)

### Confirmed design decisions (do not deviate without asking)

1. **CLI:** `pickle ticket new "<title>" --project <name> [--impact V --complexity V --cost V]`.
   `<title>` is the leading positional (must not start with `-`); flags follow (same split as
   `project add`). `--project` is required and must name a registered child (`cfg.Project`), else
   a clear error.
2. **Id allocation:** next id = `max(numeric part across every status dir) + 1`, formatted
   `T-%03d` (zero-padded to match the existing T-001…T-012). Implement `ticket.NextNum(root)`.
3. **Slug:** `ticket.Slugify(title)` — lowercase, non-`[a-z0-9]` runs → single `-`, trimmed;
   empty → `untitled`. Filename `T-NNN-<slug>.md`.
4. **Scaffold source (CONFIRMED — option A):** the CLI writes a **canonical minimal scaffold**
   via `ticket.Scaffold(...)`, *not* a literal copy of the full `TEMPLATE.md` guidance. The
   scaffold has: filled frontmatter (`id`, `title`, `project`, `depends-on: []`, the three
   grades), the `# T-NNN — <title>` heading, a `## Description` placeholder comment,
   `## Implementation Plan` with `<!-- empty until refined -->`, an empty `## Review` marker,
   and a `## History` with the `created (TO DO). source: pickle ticket new` line. The full
   `TEMPLATE.md` remains the authoring guide the agent consults at refinement (and that
   `install` writes into the skill). **Drift guard:** a unit test asserts the scaffold's
   ordered `##` section headings equal `TEMPLATE.md`'s (`Description`, `Implementation Plan`,
   `Review`, `History`), so a change to the template's section set forces the scaffold to keep
   up (read the template via the relative path `../../skill/resources/TEMPLATE.md`, as
   `config_test.go` reads the repo `pickle.toml`).
5. **Grades:** validate any provided value against the legal set; default when omitted to
   `impact=medium`, `complexity=medium`, `cost=M`. **Move the legal-grade sets into
   `internal/ticket`** (`LegalImpact`/`LegalComplexity`/`LegalCost`) and have `internal/audit`
   consume them — one source of truth (removes the duplicate table in `audit`).
6. **Board row:** insert `| T-NNN | <title> | <impact> | <complexity> | <cost> | [] |` into the
   `## TO DO` section under the ticket's `### <child>` sub-group, in impact order (rank
   `low<low-medium<medium<medium-high<high<high-critical<critical`; insert before the first
   existing row of strictly lower rank, else at the end of the sub-group). Create the
   `### <child>` sub-group (with the standard header row) if it does not exist. Implement
   `board.AddTODORow(boardPath, child string, row)`.
7. **No auto-audit inside the command** (keep it focused); the acceptance test proves
   consistency by running `pickle board audit` afterwards.

### Tasks

1. **`internal/ticket`** — add `NextNum(root) int`, `Slugify(title) string`, the exported
   `LegalImpact/LegalComplexity/LegalCost` sets + a `ValidGrade(kind, v) bool`, and
   `Scaffold(id, title, project, impact, complexity, cost string) string`. Unit tests,
   **including the section-headings parity test against `skill/resources/TEMPLATE.md`**
   (decision 4) and a test that `Scaffold(...)`'s output passes `ParseFrontmatter` +
   `LastHistoryStatus == "TO DO"`.
2. **`internal/audit`** — replace its private `legal` table with `internal/ticket`'s sets
   (behaviour unchanged; keep tests green).
3. **`internal/board`** — add `AddTODORow`. Unit tests (insert into existing sub-group in
   order; create a missing sub-group).
4. **`internal/cli/ticket.go`** — implement `runTicketNew`: parse args, `loadConfig`, validate
   project + grades, allocate id, write `tickets/1-to-do/T-NNN-<slug>.md`, `AddTODORow`, print
   the created path.
5. **Docs** — `README.md`: mark `ticket new` implemented + usage.

### Acceptance test

```
cd /Users/codcod/Projects/private/pickle
just lint && just test && just build

# work on a full copy of the repo's flow so ids/board are realistic and untouched
rm -rf /tmp/pickle-t003 && mkdir -p /tmp/pickle-t003 && cp pickle.toml /tmp/pickle-t003/ && cp -R tickets /tmp/pickle-t003/
BIN=/Users/codcod/Projects/private/pickle/pickle
( cd /tmp/pickle-t003 && \
  "$BIN" ticket new "Review the Jira board" --project pickle --impact high --complexity medium --cost M && \
  test -f tickets/1-to-do/T-013-review-the-jira-board.md && \
  grep -q 'T-013' tickets/BOARD.md && \
  "$BIN" board audit )                                  # <-- dogfood: MUST report 0 errors, exit 0

# failure modes
( cd /tmp/pickle-t003 && "$BIN" ticket new "x" --project nope ; test $? -ne 0 )   # unregistered child
( cd /tmp/pickle-t003 && "$BIN" ticket new ; test $? -eq 2 )                       # missing title -> usage
```
Expected: the new ticket file exists with `id: T-013`, `project: pickle`, filled grades, and a
`created (TO DO)` History line; the board has a `T-013` row under TO DO / `### pickle`;
**`pickle board audit` reports 0 errors** on the mutated tree (the definitive proof); the two
failure modes exit non-zero.

### Docs update (mandatory)

`README.md` — `ticket new` marked implemented + usage line. No `docs/` book yet.

### Finish (mandatory)

1. Acceptance test green; `just lint`/`test`/`build` clean.
2. README updated.
3. Summary of files + decisions.
4. Suggested commit: `feat(cli): add ticket new (id + scaffold + board row) (T-003)`.
5. Commit locally on the branch; **do not push / open MR without approval**.


## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1)
- 2026-07-23 — TO DO → READY: implementation plan complete (READY gate met); prerequisite T-001 done+merged. Scaffold-source decision confirmed (option A + headings-parity test).
- 2026-07-23 — READY → IN DEVELOPMENT: picked up, branch feat/T-003-ticket-new (applicability gate clean)
- 2026-07-23 — IN DEVELOPMENT → IN REVIEW: acceptance test green (ticket new -> board audit 0 errors on the mutated tree; failure modes exit non-zero)
