---
id: T-052
title: post-upgrade audit cannot tell a registry-changed board from a hand-edited one
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S-M
---

# T-052 — post-upgrade audit cannot tell a registry-changed board from a hand-edited one

## Description

Registering a child-project and then upgrading — the documented onboarding sequence — ends in
an error and a non-zero exit, for a workspace where nothing is wrong:

```
$ pickle project add snowball snowball --build "just build" --test "just test" --lint "just fmt-check"
registered child-project "snowball" at snowball
$ pickle upgrade
  + .agents/skills/ticket-flow/
  + AGENTS.md (marker updated)
  …
ERROR: BOARD.md is stale or hand-edited — run pickle board sync
pickle: post-upgrade audit found 1 error(s)          # exit 1
$ pickle board sync
  reformat only (ordering / WIP counts / spacing / preamble)
$ pickle doctor && pickle board audit
pickle doctor: 0 error(s), 0 warning(s)
board audit: 88 tickets, 0 error(s), 0 warning(s)
```

The board *is* stale, and legitimately so: registering a child changes the board's **generated
shape** — it gains a per-child section under every status heading and a per-child WIP line — so
a board that was in sync a second earlier no longer matches what the tickets now generate. The
post-upgrade self-check (`runUpgrade` in `internal/cli/install.go`, and its twin in
`runInstall`) audits with no sync step, classifies staleness as an error, and prints only
`ERROR:` lines — so a warning there would today be invisible as well as impossible.

The verdict's *advice* is right (`board sync` is exactly the next command) but its
**classification is not**: "stale **or** hand-edited" collapses a routine consequence of a
registry change into the same bucket as a rule violation. Two costs, both about trust rather
than data: a scripted or agent-driven onboarding sees a failed command in a healthy sequence,
and a human learns to expect one error mid-flow — the habit that hides a real one. Observed
during the `snowball` onboarding of the `unity` workspace (pickle 0.1.0, 2026-07-27).

What must **not** be assumed at refinement: that the fix is "make `upgrade` run
`board sync`". `upgrade` staying out of the board's business is a property worth keeping, and
silently rewriting a board that *might* be hand-edited is the very loss that made `board sync`
an explicit, separate command (see `tickets/NOTES.md` on T-044: the board is a pure generated
artifact precisely so nothing hand-written is at risk). The smaller and more defensible change
is a verdict that **separates the two causes** — registry/ticket state changed since the board
was generated → "run `pickle board sync`" as a *warning* (exit 0); rendered content diverges
from what the tickets generate in a way sync would not explain → error. Whether that
distinction is even computable from the current audit inputs is the first question refinement
has to answer, and a negative answer (with the message merely re-worded) is a legitimate
outcome.

Soft coupling: **T-051** (the same `project add` → `upgrade` sequence leaves five workspace-side
edits unmentioned) — both came from one onboarding session and both concern what that command
pair tells the operator; they ship independently. T-051 is now in `6-done/`, and it left
`runProjectAdd` already re-deriving the *other* generated artifact (the `AGENTS.md` marker
block, via `refreshMarkers`) — which is the precedent decision 6 below builds on.

## Implementation Plan

### Feature branch

`feat/T-052-post-upgrade-audit-cannot-tell-a-registry-changed-board-from-a-hand-edited-one`,
cut in the `pickle` child (the repo root) from `main`. Local WIP commits are fine; **no push
and no MR without explicit user approval**. Never run `pickle install|upgrade|project add`
against this repo from the branch — test installs go to a throwaway dir with the binary copied
in (`AGENTS.md`, self-modify policy).

### Prerequisites

None. No `depends-on:`; T-051 (soft coupling) is done and merged.

### Confirmed design decisions (do not deviate without asking)

1. **The cause is not computable; the harm is.** Nothing can distinguish "a human edited this"
   from "the renderer changed" — the ticket's first question is answered *no* as posed. What is
   computable, from inputs the audit already has, is whether the file still tells the truth
   about the tickets. That is the split the verdict is rebuilt on.
2. **The test: compare rendered ticket rows, as text, section by section.** Build, for the
   file on disk and for a fresh `board.Render`, the multiset of
   `(status section, child sub-group, raw row line)` over every ticket row. **Equal** ⇒ the
   board lists the same tickets, in the same sections, with the same cells, and only generated
   scaffolding differs (preamble, WIP-limit lines, `### <child>` sub-headings and counts, empty
   tables, row order, blank lines) ⇒ **warning**. **Unequal** — a row added, removed, moved to
   another section or child, or any cell text changed ⇒ **error**.
3. **This does not parse cells back into data** (T-044 decision 9). Row lines are compared as
   opaque text against freshly rendered text; nothing is interpreted. Say so in the comment,
   next to `Parse`'s existing "cell contents are never read back" note, or the next reader will
   read this as the escape-scheme hazard T-044 closed.
4. **Row order is deliberately ignored** (multiset, not sequence). A payload upgrade that
   changes the sort (T-059's family grouping was exactly that) reorders identical rows, and
   that is the same class of event as a registry change. The cost — a hand-reordered board is a
   warning, not an error — is accepted: it misinforms nobody about which ticket is where.
5. **Two messages, one advice.** `board sync` remains the fix in both branches; only the
   severity and the sentence change. Exact strings:
   - error: `BOARD.md does not match the ticket files (rows differ) — run pickle board sync`
   - warning: `BOARD.md is out of date in its generated layout only (every ticket row matches) — run pickle board sync`

   The string `stale or hand-edited` disappears from the codebase; it is the classification the
   ticket exists to retire.
6. **`project add` / `project remove` regenerate the board.** The registry is an input to
   `board.Render`, and every other mutator of a render input (`ticket new`, `ticket move`)
   regenerates; `project add` already re-derives the marker block from the same registry. This
   removes the reported instance at source rather than merely down-grading its verdict.
   `upgrade` still never reads or writes anything under `tickets/`.
7. **A failed regeneration never fails a registration.** `pickle.toml` is already saved by
   then; a board that cannot be rendered (ticket load problems) prints a note pointing at
   `pickle board sync` and the command still exits 0 — the same restraint `refreshMarkers`
   documents.
8. **`board sync --dry-run` stays strict.** It answers "would this file change", which is what
   a CI wiring wants; layout-only drift keeps exiting non-zero. Deliberate non-change — record
   it in the summary so it is not re-litigated. `sync`'s own `reformat only (ordering / WIP
   counts / spacing / preamble)` drift line already describes this case and is left alone.
9. **The existing error cases stay errors.** T-044's hand-edited-row and ghost-row tests, and
   T-049's over-long-cell-from-an-old-binary migration note, all change a row line — the docs
   that promise an error for them stay true, with only the sentence updated.

### Tasks

#### Task 1 — classification in `internal/board`

In `internal/board/board.go`:

- add `Line string` to `Row` (the raw, `strings.TrimSpace`d row line as it appears);
- split `Parse` into the file-reading wrapper plus `func ParseText(text string) []Row` holding
  the current loop, so a rendered string can be parsed without a temp file;
- add the verdict type and function next to `NormalizeLastUpdated`:

  ```go
  type Drift int
  const (
      DriftNone   Drift = iota // in sync (Last updated: aside)
      DriftLayout              // generated scaffolding only; every ticket row matches
      DriftRows                // the rows themselves disagree
  )
  func Compare(current, fresh string) Drift
  ```

  `Compare` normalises both sides with `NormalizeLastUpdated`, returns `DriftNone` on equality,
  otherwise compares the `(Status, Child, Line)` multisets of `ParseText` on each side
  (`map[string]int` keyed on the three joined by a separator that cannot occur in a rendered
  cell, e.g. `"\x00"`) and returns `DriftLayout` when they match, `DriftRows` when they do not.

Document decisions 2–4 in the doc comment: what each verdict means, why order is ignored, and
that the comparison is textual.

#### Task 2 — the audit's two-tier verdict

`internal/audit/audit.go`, the board staleness block (currently one `errf` after a normalised
byte comparison): switch on `board.Compare(string(data), board.Render(tickets, cfg, ""))` and
emit decision 5's error / warning / nothing. Keep the missing-file branch (`BOARD.md: %v`) an
error, untouched. Update the block's comment — the board invariant is no longer "one
byte-comparison", it is "the rows must match; the layout should".

#### Task 3 — self-checks print warnings

`internal/cli/install.go`, both self-check blocks (`runInstall` and `runUpgrade`): print
`WARNING: %s` for each `a.Warnings` entry before the error loop, and keep the exit rule exactly
as it is (non-zero only when `len(a.Errors) > 0`). Without this the new warning is emitted into
nothing at the two moments this ticket is about. `runBoardAudit` already prints both.

#### Task 4 — `project add` / `project remove` regenerate the board

`internal/cli/project.go`: after `refreshMarkers(cfg)` in both `runProjectAdd` and
`runProjectRemove`, call a small shared helper (next to `refreshMarkers`, same file) that runs
`board.Regenerate(cfg.Root(), cfg)` and prints `  + tickets/BOARD.md` on success, or
`note: could not regenerate the board (%v) — run pickle board sync` on failure, returning no
error either way (decision 7). Comment it with the reason: the registry is a render input, so
the two derived artifacts — marker block and board — are refreshed together.

#### Task 5 — tests

- `internal/board/board_test.go` — a table over `Compare`: identical text ⇒ `DriftNone`;
  date-only difference ⇒ `DriftNone`; a render before/after adding a second registered child
  (same tickets) ⇒ `DriftLayout`; a hand-bumped WIP-limit preamble line ⇒ `DriftLayout`; rows
  reordered within a section ⇒ `DriftLayout`; a ghost row added ⇒ `DriftRows`; a row deleted ⇒
  `DriftRows`; one cell's text changed (a title, and separately a 121-rune cell that the cap
  would shorten — the T-049 case) ⇒ `DriftRows`; a row moved to another status section ⇒
  `DriftRows`.
- `internal/audit/audit_test.go` — the two existing board cases (`stale board (hand-edited
  row)`, `stale board (ticket changed, board not regenerated)`) keep `wantErr: true` with the
  new substring; add a layout-only case asserting **zero errors and exactly one warning**
  (the existing table is error-oriented — follow whatever the warning-asserting cases, e.g. the
  `.gitkeep` one, already do rather than inventing a second style).
- `internal/cli/cli_test.go` — end-to-end in the sandboxed CWD (pattern:
  `TestProjectAddRefreshesMarkerBlock`): after `install`, `project add <name> <dir>` leaves
  `board audit` at 0 errors / 0 warnings and prints the board line; and a board made
  layout-stale by hand (bump `wip_in_review` in `pickle.toml`) makes `upgrade` print a
  `WARNING:` and still return `exitOK`.

#### Task 6 — docs

- `docs/user-manual/cli-reference.adoc`:
  - the `pickle board audit` bullet at ~505 — replace the "exactly one error" sentence with the
    two-tier verdict and both strings, naming the test (rows vs generated layout);
  - the NOTE at ~534 (120-rune cell cap migration) — it stays an *error*, since the cell text
    changes; update the quoted string;
  - the `pickle board sync` bullet at ~575 — same one-string update;
  - `pickle upgrade` at ~202 ("A post-upgrade `board audit` self-check must pass") — say that
    it fails only on audit *errors*, and that warnings are printed and do not fail the upgrade;
  - `pickle project add | list | remove` — `add`/`remove` now regenerate `BOARD.md`, because a
    registered child changes the board's generated shape.
- `skill/resources/tickets-README.md:323` and `skill/SKILL.md:262` — both currently say a
  stale or hand-edited board is "one error". Reword to the two tiers. These are the shipped
  payload; the repo's own `.agents/skills/ticket-flow/` is a symlink to `skill/`, so no second
  copy needs touching.
- `CHANGELOG.md` — an entry under `## [Unreleased]` → `### Changed`, symptom first (the
  documented onboarding sequence `project add` → `upgrade` ended in an error and exit 1 with
  nothing wrong), then the two-tier verdict, then the `project add` regeneration. Do **not**
  edit the released 0.1.0 section at line ~181; the migration note there describes a cell-text
  change, which is still an error, so only its quoted string is stale — leave the entry as the
  historical record it is and let the docs carry the current wording.

### Acceptance test

From the repo root, all four green:

```
just build && just test && just lint && just docs-check
```

Then the reported sequence and both new branches, in a throwaway dir with the binary copied in
(self-modify policy):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q -b main .
./pk install --project root --agent claude
mkdir child && (cd child && git init -q -b main .)

# 1. the reported sequence — now clean end to end
./pk project add child child            # expect: registered … and "+ tickets/BOARD.md"
./pk upgrade; echo "exit=$?"            # expect: no ERROR, no WARNING, exit=0
./pk board audit                        # expect: 0 error(s), 0 warning(s)

# 2. layout-only staleness — warning, exit 0
sed -i.bak 's/wip_in_review = 1/wip_in_review = 2/' pickle.toml
./pk board audit; echo "exit=$?"        # expect: WARNING: … generated layout only …, 0 error(s), exit=0
./pk upgrade; echo "exit=$?"            # expect: the same WARNING printed, exit=0
./pk board sync && ./pk board audit     # expect: 0 error(s), 0 warning(s)

# 3. row divergence — still an error, exit 1
printf '| T-999 | ghost | low | low | S | [] | |\n' >> tickets/BOARD.md
./pk board audit; echo "exit=$?"        # expect: ERROR: BOARD.md does not match the ticket files (rows differ) …, exit=1
```

Expected results in full: step 1 reproduces the ticket's transcript and ends silent and green
(the board is regenerated at registration, so there is nothing left to warn about); step 2
shows the same divergence class the ticket reported, reached without `project add`, reported as
a warning by both `board audit` and `upgrade` without either exiting non-zero; step 3 shows the
error tier intact, with the ghost row appended *inside* a rendered section so it parses as a
row. No step edits a file under `tickets/` except the deliberate `printf`, and nothing in the
pickle repo itself is installed to or upgraded.

### Docs update

See Task 6 — user-facing in three places (manual, shipped skill/rules payload, changelog).

### Finish

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated (Task 6).
3. Summary: what shipped (the `Compare` verdict, the audit's two tiers, warnings printed by the
   `install`/`upgrade` self-checks, board regeneration on `project add`/`remove`) and what was
   decided **against** — `upgrade` still never touches the board, `board sync --dry-run` stays
   strict, and the cause of a divergence is still not (and cannot be) inferred.
4. Suggested commit message:

   ```
   fix(audit): warn, not error, when only the board's generated layout is stale (T-052)
   ```

5. Commit locally on the ticket branch; present the message and **do not push or open an MR
   without explicit user approval**. Before pushing, `git fetch origin main && git diff
   --name-only origin/main...HEAD | grep '^tickets/'` must print nothing (rules §0).

## Review

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files (step 4b) — one
      applicable suggestion (on prose this ticket authored) applied; the rest were on
      pre-existing text outside this ticket's scope, left alone
- [x] Findings recorded below with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved to `tickets/6-done/`; `## History` appended (step 6)
- [x] Other references updated — T-065's stale soft-coupling note patched (step 7/8)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval; remote-base check before push; next-ticket suggestion (step 9)

**Implementation audit (step 2).** Re-ran `just build && just test && just lint && just docs-check`
— all green. Re-ran the ticket's full acceptance transcript end to end in a throwaway install
(self-modify policy): (1) `project add` → `upgrade` → `board audit` — silent, `0 error(s), 0
warning(s)`, exit 0; (2) a hand-bumped `wip_in_review` — `WARNING: BOARD.md is out of date in
its generated layout only (every ticket row matches)` from both `board audit` and `upgrade`,
exit 0 in both, cleared by `board sync`; (3) an appended ghost row — `ERROR: BOARD.md does not
match the ticket files (rows differ)`, exit 1. Every task (1–6) verified present and matching
the plan's exact shapes (`board.Compare`/`ParseText`/`Row.Line`, the audit's two-tier switch,
`WARNING:` printing in both `runInstall` and `runUpgrade`, board regeneration in both `project
add` and `project remove`, the full test list, and all six doc-update bullets). All **met**.

**Quality / consistency / docs audits (steps 3–4a).** Code is idiomatic and consistent with the
package's existing style (`Compare` sits next to `NormalizeLastUpdated`, mirrors `sync.Sync`'s
comment conventions). `Compare` documents why row order is ignored and why it never parses
cells back into data. Test coverage is thorough: a 9-case `TestCompare` table, two new audit
cases (a layout-only warning alongside the two reworded row-divergence errors), two end-to-end
CLI tests. Docs coverage is complete (manual, shipped skill payload, changelog) and a
whole-tree grep for the retired `"stale or hand-edited"` string found only correctly-preserved
historical references (three done tickets' own records, this ticket's own Description quoting
the original bug report, and the 0.1.0 changelog migration note the plan explicitly left
alone) plus the new changelog entry's own before/after quoting. `just docs-check` green.

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | fixed inline | A doc comment on `TestUpgradeWarnsOnLayoutOnlyBoardDrift` named a sibling test, `TestUpgradeFailsOnBoardDrift`, that does not exist anywhere in the tree — a dead reference this branch authored. | `internal/cli/cli_test.go`, the comment above the test func | Reworded to describe the row-divergence behaviour in prose instead of naming a nonexistent test. |
| F2 | non-blocking | fixed inline | `regenerateBoard`'s doc comment claimed it prints the "same `+/=` idiom" as `refreshMarkers`, but `board.Regenerate` always writes unconditionally and reports no changed/unchanged signal — the function only ever emits `+`, never `=`. | `internal/cli/project.go`, `regenerateBoard` | Comment corrected to state the actual (unconditional) behaviour and why, rather than claiming a parity that isn't implemented. |
| F3 | non-blocking | noted | `board.rowKey` joins `Status`/`Child`/`Line` with a `\x00` separator on the stated assumption that a NUL byte cannot appear in any of the three. True for every reachable CLI path (argv cannot carry a NUL; project names come only from `pickle project add <name>`, a C-string), but not enforced by `config.Validate` — a hand-edited `pickle.toml` using TOML's `\u0000` escape in a project name would defeat the separator and could misclassify `DriftRows` as `DriftLayout` or vice versa. | `internal/board/board.go`, `rowKey` | Purely theoretical (unreachable via any command pickle ships) and low-harm (a classification mistake, not data loss or a crash); a real fix (a struct-keyed map instead of a joined string) is a behaviour change, so it doesn't meet the fixed-inline bar, and it fails the promotion test for a new ticket. Recorded here for a later reviewer to promote if a reachable path ever appears. |
| F4 | non-blocking | fixed inline | The new `CHANGELOG.md` entry was inserted at the *top* of `[Unreleased]`'s `### Changed` list, ahead of the existing T-068 entries — out of the file's established convention (every other section's entries are appended in landing order, e.g. `### Added`'s T-051 → T-057 → T-068 → T-057). | `CHANGELOG.md`, `[Unreleased]` → `### Changed` | Moved to the end of the `### Changed` list, after the T-068 entries. |

**Disposition summary:** 4 findings, 0 blocking. 3 fixed inline (F1, F2, F4); 1 noted (F3).

**Impact sweep (step 8).** Re-read every non-terminal ticket citing T-052. **T-083** cites it
only in a survey table (ticket-opening-style counts) — no assumption to correct. **T-065**
(`1-to-do/`, JSON read projection) recorded a soft coupling assuming T-052's "stale **or**
hand-edited" conflation was still an open vocabulary question a future JSON field would need to
resolve; T-052 already resolved it (`board.Drift`: `DriftNone`/`DriftLayout`/`DriftRows`). Patched
T-065's coupling note to point at the resolved vocabulary instead, with a History line —
committed on this base branch alongside this review, never on the feature branch.

## History

- 2026-07-27 — created (TO DO). source: idea — field finding from adding a second child-project to the `unity` workspace with pickle 0.1.0
- 2026-08-05 — refined: the Description's open question is answered *no as posed, yes as re-framed* — the cause of a divergence is not computable, the harm is, so the verdict splits on whether the rendered ticket rows still match (error) or only the generated layout is stale (warning). Cost re-graded S → S-M for the added `project add`/`remove` board regeneration.
- 2026-08-07 — TO DO → READY: plan complete
- 2026-08-07 — READY → IN DEVELOPMENT: picked up
- 2026-08-07 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-07 — IN REVIEW → DONE: review clean — 4 non-blocking findings (3 fixed inline, 1 noted)
