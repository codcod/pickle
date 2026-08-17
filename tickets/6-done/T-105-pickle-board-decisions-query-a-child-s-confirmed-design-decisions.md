---
id: T-105
title: "pickle board decisions: query a child's confirmed design decisions"
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low-medium
cost: S
---

# T-105 — pickle board decisions: query a child's confirmed design decisions

## Outcome

Running `pickle board decisions` prints every confirmed design decision recorded in the ticket
tree, already in citable `<ID> decision <N>` form, filterable by child-project, status and topic,
with `--json` for machine consumers. Today the same answer needs a hand-written `awk` that
re-solves two parsing traps every time and that gets the child filter wrong on any workspace whose
ticket prefix is not `T-`.

## Description

A child's design decisions accumulate in each ticket's `### Confirmed design decisions` subsection
and are currently reachable only by ad-hoc `grep`/`awk`. This ticket adds a supported read-only
query. It is a **human-facing tool**, invoked ad hoc like `pickle doctor` — no procedure in the
brine flow calls it, and none is changed to call it (see *Scope fence*).

**All figures below are measurements and will drift.** They are recorded as the evidence the
ticket was filed and refined on, not as maintained facts; re-measure with the command itself once
it exists. Measured across two independent workspaces:

| corpus | measured | tickets | decisions | `N. **statement.** rationale` form |
| --- | --- | --- | --- | --- |
| this repo | at filing, 2026-08-16 | 105 | 397 | 367 (92%) |
| this repo | at refinement, 2026-08-16 | 106 | 449 | 433 (96%) |
| an unrelated 3-child brine workspace | at filing, 2026-08-16 | 95 | 138 | 120 (86%) |

The third row is the load-bearing one: the statement/rationale shape is an **emergent convention
that already holds in a workspace which has never seen this ticket**, so the command can key on it
without first imposing it. The re-measurement at refinement confirms the shape is not eroding —
conformance rose while the corpus grew. Raising the remaining few per cent is deliberately a
separate concern (T-106).

### Determinism — reuse the existing readers, do not write a third one

The ad-hoc scripts are fragile in three ways a shipped command must not be. All three are already
solved by tested code in this repo:

1. **`project:` must be frontmatter-scoped.** A plain `grep '^project:'` also matches ticket body
   text — this repo has a `project: nope | medium | medium | M |` inside a fixture table
   (`tickets/6-done/T-030-…md`). Use `ticket.LoadAll` + `ticket.ParseFrontmatter`.
2. **The decisions subsection must be bounded**, or numbered items bleed in from `## Tasks` and
   `## History`. `ticket.SectionBody` bounds `## ` sections and `ticket.SubsectionMissing` already
   scans `### ` subsections — but returns only a bool, so the one genuinely new primitive here is
   `SubsectionBody(text, section, stem) (string, bool)`, factored out of the scan
   `SubsectionMissing` already performs. That factoring is aligned with T-042, not in tension with it.
3. **Ticket ids must never be pattern-matched as `T-NNN`.** Children set their own `ticket_prefix`;
   the reference workspace above uses `RICK-` and `SNOW-` alongside a `T-` child. Ids come from the
   loaded ticket.

**What to project, following the rule `internal/state/review.go` already established** (T-065): it
projects only closed-vocabulary fields and never the multi-sentence prose columns. Applied here —
project the **ticket id**, the **decision number**, and the **statement** (the leading bold run,
emphasis-stripped per T-065 decision 11); do **not** project the rationale prose that follows. A
numbered item with no leading bold run is emitted with the structured flag false and its raw first
line — **never inferred**. That is what makes the command deterministic without requiring the
corpus to change first. Filtering is not projecting: `--grep` searches the whole item (statement
*and* rationale), because topics live in the rationale, while the output still carries only the
statement.

### Surface

```
pickle board decisions [--project <child>] [--status <dir>] [--grep <regex>] [--json]
```

Read-only. It lives under `board`, not `project`, because `board` is already this CLI's
**whole-ticket-tree read** namespace — `audit`, `sync` and `state` all walk `tickets/` — whereas
`project` is registry operations over `pickle.toml` (`add`/`list`/`remove`). Making the child a
*filter* rather than a mandatory positional also keeps the command usable in a single-child
workspace, which is the common case.

**Exit semantics matter here** and are the subject of the acceptance test: an **unregistered**
child, an **unknown status directory** and an **uncompilable regex** are all errors; a **registered
child with no matching decisions is exit 0 with an empty result**. The prototype conflated these by
leaking `grep`'s exit 1, and all three empty cases in the acceptance test below are legitimate
answers, not failures. This repo already treats exit codes as contract (T-057/T-082 fail-open,
T-071 exit-1 semantics).

Human-readable output must be pasteable as a citation — the first column is literally
`<ID> decision <N>`; `--json` is the machine form, with its own envelope and its own `schema`.

### Decisions taken at refinement (were open at filing)

- **Namespace.** Challenged as the ticket asked, and moved: `pickle project decisions <child>` →
  `pickle board decisions [--project <child>]`. Reasoning above; the ticket was retitled and its
  file renamed to match.
- **`--json` is standalone, not a `board state --json` schema bump.** Confirmed as filed. T-065
  shipped `schema: 1` for a document describing the whole tree; decisions get their own small
  envelope with its own version, and folding them into `board state` stays a cheap later decision
  that this command does not gate.
- **The acceptance test needs no foreign workspace.** `internal/cli`'s own harness builds sandboxes
  with any number of children (`newProject` + `project add`), so the child-filter case is a Go
  test here rather than a manual run against an unrelated repo. This removes the one part of the
  filed acceptance test that this repo could not execute.

### Scope fence — deliberately out

- **Changing `TEMPLATE.md` and adding an audit row** → T-106. This command must work on the corpus
  as it stands, which is why it reports non-conforming items rather than requiring conformance.
  (T-106's refinement settled on *no* audit row, which makes that separation permanent rather than
  temporary: nothing will ever enforce the shape, so this command must always tolerate its absence.)
- **Locked-vs-ticket-local classification.** It is a judgement, not a property of the text, and the
  only mechanical proxy (citation counting) is fragile — a citation scan missed 6 of 29 real
  citations here on possessives, bold and backticks, and the reference workspace also uses a comma
  variant (`RICK-156, decision 1`). Rationale and the pre-registered trigger for revisiting are in
  `NOTES.md` § *"ADR exploration (2026-08-15) — explored, nothing filed; the convention already
  works"*.
- **Reading citations at all.** The command emits citable output; it does not parse citations.
- **Wiring the command into any brine procedure.** It has no procedural consumer by design.

### Soft couplings (no hard `depends-on`)

T-065 (shipped; its `internal/state` prose-parsing precedent is the pattern to follow), T-106 (the
format spec this ticket deliberately does not do — independently schedulable in either order),
T-066 (a new command needs a `cli-reference.adoc` entry, and this ticket must not add to the gap
that ticket exists to close), T-042 (the `SubsectionBody` factoring).

### Grading rationale

`impact: low` deliberately, consistent with T-065's own re-grade: the need is served today by a
one-off script, the measured cross-ticket citation rate is 7.6%, and the 2026-08-04 precedent
refuses to credit prospective demand. `complexity: low-medium` and `cost: S` re-confirmed at
refinement rather than inherited: the parsing infrastructure already exists, and the plan below is
one extracted helper, one new leaf package, one CLI surface, docs and tests — no new I/O, no
locking question (a read behind the shared tree lock, exactly as `board state` already does), and
no change to any existing wire format.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-105-board-decisions
```

Local WIP commits are encouraged. Because this is a root-path child, tidy them into atomic commits
before presenting (rules §0) and default to keeping that history rather than squashing. Do **not**
push or open a merge request without explicit user approval. Ticket and board bookkeeping is
committed on `main`, never on this branch — `pickle hooks` enforces it.

### 1. Prerequisite gate (hard)

None. `depends-on:` is empty, and every helper this plan builds on is already shipped and merged:
`ticket.LoadAll`, `ticket.ParseFrontmatter`, `ticket.SectionBody`, `ticket.SubsectionMissing`,
`flow.Definition`, `lock.WithShared`. T-106 is a soft coupling only — this command reads the corpus
as it stands and must not be blocked on any documentation change.

### 2. Confirmed design decisions (do not deviate without asking)

1. **The command is `pickle board decisions`, not `pickle project decisions`.** `board` is the
   whole-ticket-tree read namespace (`audit`, `sync`, `state`); `project` is `pickle.toml` registry
   operations. The child is a `--project <name>` *filter*, not a mandatory positional, so the
   command is usable in a single-child workspace.
2. **All four flags are optional and compose.** `--project`, `--status`, `--grep`, `--json`. With
   no flags at all the command prints every decision in the tree — unlike `board state`, which
   requires `--json` (T-065 decision 1), because this command's default output is a short,
   human-shaped table rather than a full document dump.
3. **Ids are never pattern-matched.** Every id, prefix and number comes from the loaded
   `ticket.Ticket`. No regex of the form `T-\d+` appears anywhere in the new code.
4. **The decision number is the ordinal written in the file, not a re-count.** If a ticket's list
   is `1, 2, 4`, the command reports `4`, because a citation elsewhere in the corpus says
   `<ID> decision 4` and must resolve to the same item. Never renumber.
5. **Only a column-0 ordinal opens a decision.** A line matching `^\d+[.)]\s` at the start of a
   line inside the bounded subsection begins an item; an indented ordinal is a nested list inside
   the current item's rationale, not a decision of its own. Every following line up to the next
   column-0 ordinal (or the end of the subsection) belongs to that item.
6. **Lines inside a fenced code block are not scanned.** Track ``` fences while walking the
   subsection body, so a numbered line in an illustrative snippet is never counted as a decision.
   This is a deliberate, local improvement over `SectionBody`'s documented fence-blindness, not a
   change to `SectionBody` itself.
7. **The statement is the leading bold run, or nothing.** If the item — its soft-wrapped lines
   flattened to one line, after the ordinal — opens with `**…**`, the statement is that run's
   content with emphasis/code markers stripped and whitespace collapsed — reuse the normalisation
   shape of `internal/state/review.go`'s `emphasisRE`/`whitespaceRE` (T-065 decision 11). Otherwise
   `structured` is **false** and the statement is **never inferred** from the prose.
   *Amended during implementation (finding F1): this decision originally scoped the match to "the
   item's first line". A bold statement routinely wraps onto a second physical line before its
   closing `**`, and a first-line-only match read 15 genuinely-structured decisions in this corpus
   as unstructured (439 vs 454). The match stays anchored at the item's start, so flattening can
   only rescue a wrapped run — it can never promote a bold run that begins later in the item.*
8. **`raw` is emitted only for an unstructured item.** For a structured item `raw` is `""`. This is
   what keeps the projection inside T-065's rule: the closed-vocabulary field plus, for the items
   that have no such field, the single raw first line — never the multi-line rationale.
9. **`--grep` matches the whole item (statement + rationale), case-insensitively.** Compile the
   pattern with a `(?i)` prefix (a caller who wants case sensitivity writes `(?-i:…)` inline). An
   uncompilable pattern is an error, not zero results. Filtering over the full item while
   projecting only the statement is deliberate: topics live in the rationale.
10. **`--status` takes a status *directory* name (`6-done`), validated against the flow
    definition.** Never a hardcoded list; an unknown value errors and prints the legal values from
    `def.States()`.
11. **`--project` must name a registered child.** An unregistered name is an error (exit 1). A
    registered child that yields nothing is exit 0 with an empty result — as is any filter
    combination that matches nothing. This is the distinction the prototype got wrong.
12. **Ordering is deterministic and prefix-agnostic:** ticket prefix ascending, then ticket number
    ascending, then decision number ascending. Not by status — a decision chain reads by id.
13. **The JSON envelope is this command's own, versioned independently.** `schema: 1` in a new
    type; `internal/state`'s `CurrentSchema` is untouched and `board state --json` gains no field.
14. **The read runs behind the shared tree lock**, like every other tree reader.

### 3. Tasks

#### Task 1 — extract `ticket.SubsectionBody`

In `internal/ticket/ticket.go`, factor the `### `-scan out of `SubsectionMissing` (currently
ticket.go:607) into:

```go
func SubsectionBody(text, section, stem string) (body string, found bool)
```

It returns the trimmed body of the first `### ` heading under `## <section>` whose
`normalizeHeading` form has `stem` as a prefix, bounded by the next `### ` or `## `. Rewrite
`SubsectionMissing` to be exactly `body, found := SubsectionBody(...)` plus the existing
HTML-comment-stripped emptiness test — its behaviour must not change by one byte. Carry the
existing doc comment's fence-blindness caveat onto the new function. Add table tests in
`internal/ticket/ticket_test.go` for: heading present with body, heading absent, parent section
absent, heading last in the file, and a `### ` heading immediately following (empty body).

#### Task 2 — new package `internal/decisions`

New files `internal/decisions/decisions.go` and `internal/decisions/decisions_test.go`. A leaf,
pure text-in/values-out package in the shape of `internal/audit` and `internal/changelog`: no
printing, no exit codes, no subprocess.

- `type Decision struct` with `TicketID`, `Prefix`, `Num`, `Project`, `Status`, `Dir`, `File`,
  `Number`, `Citation`, `Structured`, `Statement`, `Raw` — and the JSON tags for Task 4.
- `func Extract(text string) []Decision` — parse one ticket's file text, applying decisions 4–8.
  It fills only the per-item fields; the caller fills the per-ticket ones.
- `type Filter struct { Project, Status, Grep string }` and
  `func Query(def *flow.Definition, root string, cfg *config.Config, f Filter) (Result, error)` —
  `ticket.LoadAll`, apply the filters, sort per decision 12. Validation errors (unregistered
  project, unknown status dir, bad regex) are returned as errors.

Fixture-test `Extract` with literal strings covering: the structured form; an unstructured item; a
gap in the numbering (asserting `4` is reported as `4`); a nested indented ordinal (not a
decision); a numbered line inside a fenced block (not a decision); a multi-line item; a ticket with
no `## Implementation Plan`; and a ticket whose plan has no `### Confirmed …` subsection.

#### Task 3 — CLI surface

In `internal/cli/board.go`, add `case "decisions": return runBoardDecisions(args[1:])` to
`runBoard`, update the `usage: pickle board <audit|sync|state|decisions>` line, and implement
`runBoardDecisions` following `runBoardState`'s shape: parse flags, `loadConfig`,
`flow.ForName(cfg.FlowName())`, run `decisions.Query` inside `lock.WithShared(cfg.Root(), …)`,
render.

Text rendering uses `text/tabwriter` exactly as `runProjectList` does, with columns
`DECISION | PROJECT | STATUS | FORM | STATEMENT`, where `DECISION` is the pasteable
`<ID> decision <N>` and `FORM` is the closed pair `bold` / `raw`. Close with a summary line
`board decisions: N decision(s) from M ticket(s)`, mirroring `board audit`'s summary line — so the
zero case prints a clear `0` rather than silence.

#### Task 4 — `--json`

Marshal a `decisions.Document{Schema: 1, PickleVersion, Filters, Decisions}` with
`json.NewEncoder` + `SetIndent("", "  ")` + `SetEscapeHTML(false)`, exactly as `runBoardState`
does and for the same reason (ticket prose is full of `&`, `<` and `→`). `Decisions` marshals as
`[]` and never `null` when empty. Add a `CurrentSchema = 1` constant with the same
compatible/incompatible-change contract `internal/state`'s `Document` documents.

#### Task 5 — CLI tests

In `internal/cli`, add tests using the existing harness (`newProject`, `capture`, `t.Chdir`) —
these are the six acceptance cases below, expressed as Go tests. Build the multi-child sandbox by
calling `runProject([]string{"add", …})` against a second directory inside the same root, then
writing ticket fixtures into the status directories directly. Assert exit codes *and* output, and
assert that the `--json` document parses and carries `schema: 1`.

Also extend the `TestRunExitCodes` table with `{"board decisions bad flag", []string{"board",
"decisions", "--bogus"}, exitUsage}`.

### 4. Acceptance test

Run verbatim from the repository root; all must be green.

```
just build
just test
just lint
just docs-check
```

Then exercise the command against this repository's own tree:

```
# 1. topic filter recovers a contract chain — >1 row, each first column citable
./pickle board decisions --grep 'fail.open'

# 2. topic + status narrows to shipped work
./pickle board decisions --grep 'fail.open' --status 6-done

# 3. TO DO holds no decisions (they are written at refinement) — 0 rows, exit 0
./pickle board decisions --status 1-to-do; echo "exit=$?"

# 4. the registered child filter matches (multi-child coverage is Task 5's Go test)
./pickle board decisions --project pickle | tail -1

# 5. a filter combination that matches nothing — 0 rows, exit 0
./pickle board decisions --grep 'zzz-no-such-topic-zzz'; echo "exit=$?"

# 6. errors, each exit 1
./pickle board decisions --project no-such-child; echo "exit=$?"
./pickle board decisions --status 9-nowhere;      echo "exit=$?"
./pickle board decisions --grep '([';            echo "exit=$?"

# 7. the JSON form is well-formed and versioned
./pickle board decisions --json | jq -e '.schema == 1 and (.decisions | type) == "array"'
./pickle board decisions --grep 'zzz-no-such-topic-zzz' --json | jq -e '.decisions == []'
```

Expected: cases 1–2 print rows whose first column pastes as a citation; cases 3–5 print no rows and
exit `0`; case 6 prints an error to stderr and exits `1` three times, the status error naming the
legal directories; case 7 exits `0` twice.

Cross-check the parser against the corpus — the counts must match the `awk` measurement recorded in
the Description, ±0:

```
./pickle board decisions --json | jq '.decisions | length'                      # ≈449
./pickle board decisions --json | jq '[.decisions[] | select(.structured)] | length'   # ≈433
```

A discrepancy is a parser bug, not a corpus fact — investigate before shipping.

### 5. Docs update (mandatory when user-facing)

1. `docs/user-manual/cli-reference.adoc`:
   - add a row to the `== Overview` table:
     `| pickle board decisions [--project …] [--status …] [--grep …] [--json]`;
   - add a `[#cmd-board-decisions]` / `== pickle board decisions` section after
     `== pickle board state`, documenting **every** flag (T-066 exists because flags went
     undocumented — this ticket must not add to that gap), the exit-code contract, the
     `bold`/`raw` `FORM` vocabulary, and the fact that the rationale is deliberately not projected.
2. `internal/cli/cli.go` — add the command to the `help` text, in `board`'s block.
3. `CHANGELOG.md` — an `### Added` entry under `## [Unreleased]`, ending `(T-105)`, in the
   user-observable voice the existing entries use.
4. Run `just docs-check`.

### 6. Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated per step 5, including the `CHANGELOG.md` entry.
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggest a Conventional Commit message, e.g.:

   ```
   feat(cli): add board decisions to query confirmed design decisions (T-105)

   <body — what and why>
   ```

5. **Tidy up before presenting** — `pickle` is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed/scoped commits (a plausible split:
   `refactor(ticket): extract SubsectionBody`, `feat(decisions): …`, `feat(cli): …`,
   `docs(manual): …`).
6. Commit locally on the ticket branch. Do **not** push or open a merge request without user
   approval. Present the commit message; after approval, keep the tidied history (the root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the merge request. Merging is the human's.

## Review

Reviewed 2026-08-16 against `feat/T-105-board-decisions` (4 commits, `cc45072..b9d2af1`),
following `resources/review-protocol.md`. No overarching or per-child review addendum is
configured in `pickle.toml`, so the generic protocol is the whole standard.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — run via the session's
      `docs_readability` reviewer over `docs/user-manual/cli-reference.adoc` and `CHANGELOG.md`;
      9 suggestions returned, 7 of them against prose this branch never touched (the lock intro,
      `cmd-install`, `cmd-project`, `cmd-upgrade`, `cmd-board-state`, `cmd-changelog-check`) and
      therefore out of this ticket's scope. The 2 against this branch's own `CHANGELOG.md` entry
      proposed splitting two long sentences; discarded — readability polish, not findings, and the
      entry matches the voice of the entries around it.
- [x] Findings recorded with severity, class and disposition; disposition summary + cost line below
      (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — see below
- [x] Summary + commit messages presented for approval (step 9)

### Verification

| what | result |
|---|---|
| `just build` · `just test` · `just lint` · `just docs-check` | all clean (`go vet`, `gofmt -l` empty, snowball check green) |
| Acceptance cases 1–2 (topic, topic+status) | rows printed, first column pastes as `<ID> decision <N>` |
| Acceptance cases 3–5 (empty results) | 0 rows, `0 decision(s) from 0 ticket(s)`, exit `0` |
| Acceptance case 6 (three error cases) | exit `1` each; the `--status` error names all seven legal directories |
| Acceptance case 7 (`--json`) | `schema == 1`, `decisions` an array, `[]` (never `null`) when empty |
| Confirmed decisions 1–14 | all honoured; 7 amended in place, see F1. No `T-\d+` pattern appears in the new code (decision 3); ordinals reported as written (decision 4, verified against T-065's own `1,2,4`-style lists); read runs inside `lock.WithShared` (decision 14) |
| `pickle board audit` | 107 tickets, 0 errors, 0 warnings |

**Parser cross-check (the acceptance test's `±0` step).** The command reports 470 decisions /
454 structured across 57 tickets, against the plan's recorded `≈449 / ≈433`. Verified **not** a
parser bug: an independent reimplementation of the plan's own algorithm, run over the same tree,
reproduces 470 / 454 / 57 exactly. The 21-decision delta is exactly this ticket's own 14 plus
T-106's 7, and the reason the refinement-time `awk` missed both is that their subsection heading is
ordinal-prefixed (`### 2. Confirmed design decisions`), which `normalizeHeading` sees through and
an ad-hoc `grep` does not — the third parsing trap the ticket was filed to eliminate, caught by
the command on its own corpus. Two further probes found no projection defect: no statement in the
corpus is truncated by an inner `**` run (0 of 454 close on a word character), and all 16
unstructured items are genuinely non-conforming (they open with a backtick or plain prose).

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | stale-xref | fixed inline | Confirmed decision 7 scoped statement detection to the item's *first line*; the shipped parser matches the leading bold run against the whole item with soft wraps flattened. The widening is right — first-line-only reads 15 genuinely-structured decisions here as unstructured — but the plan then described behaviour the branch does not ship, and no `plan amended inline:` History line recorded it (rules §1). | `internal/decisions/decisions.go` `leadingBoldRE` + `flush`'s `flatten`; independent count 439 first-line-only vs 454 flattened | Decision 7 amended above with the measurement and the reason flattening cannot over-capture; History line added below. |
| F2 | non-blocking | stale-xref | fixed inline | `cli-reference.adoc`'s lock paragraph enumerates the shared-read-lock commands as "`pickle serve` and `pickle board state --json`". This branch added a third, which its own new section (line 948) correctly says takes that lock — so the branch made the enumeration incomplete. | `docs/user-manual/cli-reference.adoc:11-13` vs `internal/cli/board.go:162` | `pickle board decisions` added to the enumeration, with an xref to `<<cmd-board-decisions>>`. |
| F3 | non-blocking | design | fixed inline | `internal/decisions/decisions_test.go` carried a hand-rolled `contains` helper — an 8-line closure reimplementing `strings.Contains`, in a file that already imports `strings`. | `internal/decisions/decisions_test.go`, former `func contains` | Replaced with `strings.Contains` at all three call sites; helper deleted. |
| F4 | non-blocking | test-gap | fixed inline | The `FORM` column's closed `bold`/`raw` vocabulary is documented in the manual but asserted nowhere: swapping the two labels in `renderDecisionsText` passed every test. The sandbox's own comment claimed it seeded a raw decision "to exercise the FORM column", which nothing then exercised. | `internal/cli/board_decisions_test.go:59`; `renderDecisionsText` in `internal/cli/board.go` | Added `TestBoardDecisionsFormColumn`, asserting both rows' `FORM` cell and rendered text; mutation-checked (swapping the label fails it). Fixture prose reworded off the word "bold" so the assertion cannot pass by accident. |
| F5 | non-blocking | stale-xref | noted | The acceptance test requires the corpus cross-check to match `≈449 / ≈433` "±0" and calls any discrepancy "a parser bug"; the shipped command reports 470 / 454. The reference figures were already wrong when they were measured at refinement (the `awk` did not normalise ordinal-prefixed subsection headings), so this is a pre-existing error in the plan, not one this branch made — `noted` rather than a plan rewrite. The reconciliation is recorded under *Parser cross-check* above. | plan step 4 vs `./pickle board decisions --json \| jq '.decisions \| length'` | None. The plan's step is left as written; the row above it is the answer a future reader needs. A re-measurement, if ever wanted, is now one command rather than an `awk`. |

Disposition summary: 4 fixed inline (F1, F2, F3, F4), 1 noted (F5), 0 folded, 0 new tickets — the
four inline fixes are all prose, a test helper and a missing assertion this branch itself authored
or made false, none of them a behaviour change.

```
cost: estimated S, actual S
```

### Impact sweep (step 8)

Only T-106 (`2-ready/`) references this ticket, as `spawned-by`/soft coupling. Nothing in its plan
is invalidated — it deliberately depends on nothing here, and its decision 5 ("number in one
unbroken list and never renumber") is exactly what the shipped parser assumes. Two shipped facts
its §7 prose should stay consistent with are recorded in its own History: the statement's leading
bold run may soft-wrap across physical lines, and an item without one is reported as unstructured
rather than guessed at.

## History

- 2026-08-16 — created (TO DO). source: chat: requested directly after an exploration that had
  recommended against a decisions command; the human overrode that recommendation, and two of the
  three objections lapsed in the meantime — T-065 shipped (so the command no longer pre-empts it)
  and the format was measured 92% regular (so the parser need not be tolerant). Exploration record
  and the objections that still stand: `NOTES.md` § "ADR exploration (2026-08-15)"
- 2026-08-16 — scope narrowed in TO DO after an adversarial pass, and retitled ("from a specified,
  audit-enforced format" dropped). The pass found one blocking defect: the ticket bundled four
  separable changes, and its only cross-ticket consumer (`SKILL.md` applicability gate) needs the
  *locked* decisions this ticket explicitly excludes — so as filed it would have shipped a command
  no procedure calls **and** a format burden it did not need. Split: the `TEMPLATE.md` grammar and
  the audit row moved to T-106; the `board state --json` schema bump deferred; the command now
  reads the corpus as it stands. Re-graded `complexity: medium → low-medium`, `cost: M → S` on the
  finding that `ticket.LoadAll`/`ParseFrontmatter`/`SectionBody` already exist. Also corrected: the
  filing figures are now marked as at-filing measurements, per the warning `internal/state/review.go`
  records against hard-coding counts that go stale
- 2026-08-16 — refined: retitled and renamed (`pickle project decisions <child>` →
  `pickle board decisions [--project <child>]`) on the namespace challenge the ticket itself
  invited — `board` is this CLI's whole-tree read namespace (`audit`/`sync`/`state`), `project` is
  `pickle.toml` registry operations, and a `--project` filter beats a mandatory positional in a
  single-child workspace. Three other open questions closed: `--json` stays standalone (no `board
  state` schema bump), `--grep` searches the full item while the output still projects only the
  statement, and the child-filter case becomes a Go test in `internal/cli` rather than a manual run
  against a foreign workspace — which removes the one filed acceptance case this repo could not
  execute. Corpus re-measured at refinement: 449 decisions, 433 (96%) conforming, up from 397/92%
  at filing, so the shape is not eroding. Grades re-confirmed rather than inherited
- 2026-08-16 — TO DO → READY: plan complete
- 2026-08-16 — READY → IN DEVELOPMENT: picked up
- 2026-08-16 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-16 — plan amended inline: confirmed decision 7 now scopes statement detection to the
  item's soft-wrap-flattened text rather than its first physical line, matching what the branch
  shipped. Recorded during review as finding F1, with the measurement (439 first-line-only vs 454
  flattened structured decisions) and the reason an anchored match cannot over-capture. No other
  decision changed
- 2026-08-17 — IN REVIEW → DONE: review passed: 0 blocking, 4 fixed inline, 1 noted
- 2026-08-17 — MERGED: feat/T-105-board-decisions → main (9343299, PR #51), user-approved; history kept (4 commits); branch not deleted
