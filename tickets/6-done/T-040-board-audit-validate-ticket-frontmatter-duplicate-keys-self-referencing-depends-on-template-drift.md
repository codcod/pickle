---
id: T-040
title: board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift)
project: pickle
depends-on: []
spawned-by: [T-027, T-028, T-033]
impact: medium
complexity: medium
cost: M
---

# T-040 — board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift)

## Description

**Epic — merged from T-027, T-028 and T-033 by the 2026-07-26 board triage.** All three are in
`tickets/7-dropped/` with their full analysis and line references; read them for detail.

Three gaps in what `pickle board audit` considers a valid ticket. All three land in
`internal/audit/audit.go`, all three are "the audit is the only component that sees *every*
ticket however it was authored" — which matters because the flow explicitly permits agents and
humans to write ticket files directly (`pickle ticket new` is a convenience, not a gatekeeper).
One file, one table-driven test, one review.

### Absorbed scope

| from | check | substance |
|---|---|---|
| T-033 | duplicate frontmatter keys | `ticket.ParseFrontmatter` (`internal/ticket/ticket.go:186`) assigns into a `map[string]string`, so a duplicate key **silently overwrites** — last wins — and a ticket with two `impact:` or two `project:` lines audits clean. A duplicate key is malformed however it arrived: a hand-edit, a bad merge resolution leaving two `depends-on:` lines, or a future command. |
| T-027 | self-referencing `depends-on` | The existence loop never checks whether a ticket lists **itself**. `T-042` with `depends-on: [T-042]` audits clean, then silently self-blocks: the pickup gate demands the dependency be in `6-done/`, which it can never be while in development. The failure surfaces as a confusing "dependency not done" error about the ticket itself, at pickup, instead of a frontmatter error at audit time. One condition in the existing loop. |
| T-028 | TEMPLATE.md drift | `audit.requiredKeys` (`internal/audit/audit.go:24`) and `skill/resources/TEMPLATE.md` must agree on the frontmatter key set, and **nothing enforces it**. The only guard, `TestScaffoldSectionsMatchTemplate` (`internal/ticket/ticket_test.go:317-330`), compares `## ` headings and is blind to frontmatter. A key added to the audit while TEMPLATE keeps advertising the old set makes every hand-authored ticket fail audit — **in the user's project, not in this repo's tests**. T-024 walked this tightrope by hand. |

### Correction carried over from the T-030 review (finding N3, 2026-07-26), itself superseded at refinement

T-027's refinement note implied `internal/audit` holds a duplicate `T-\d+` regex to unify. **It
does not** — `internal/audit/audit.go` contains no regex and does not import `regexp`; its only
shape-adjacent checks are `t.Front["id"] != t.ID` (`:52`) and the existence lookups.

The follow-on claim that this ticket would add **the first external caller of `ticket.ValidID`**
is now **wrong twice over** (re-verified 2026-08-01):

- `ValidID` (`internal/ticket/ticket.go:229`) has since acquired two external callers —
  `internal/cli/ticket.go:116` (`--family` shape check) and `internal/serve/serve.go:179`.
- The self-reference check **needs no id shape check at all**. Since T-024 and T-059 the audit
  already carries two self-reference guards to mirror: `src == t.ID` for `spawned-by`
  (`internal/audit/audit.go:79`) and `fam == t.ID` for `family` (`:99`). `depends-on` gets the
  third, one condition in the existing loop, and is the odd one out only by omission.

The `T-\d+` shape *is* still literally duplicated in `filenameRE` (`internal/ticket/ticket.go:95`)
and `board.rowRE` (`internal/board/board.go:51`). Composing all three from one fragment is
**deferred to T-042** (duplicated internals) — decided at refinement, so this ticket touches no
regex.

### Folded in from the T-036 pickup gate (2026-07-26) — a fourth check

**The status directories themselves are never validated.** `tickets/3-in-development/` does not
exist in this repo right now, and `board audit` reports 0 errors. Nor is it a local accident:
`git ls-files tickets/` returns **no `.gitkeep` files at all**, so none of the seven that
`install.go:311-319` creates is tracked. `4-in-review/` and `5-rework/` survive only as untracked
local leftovers — **a fresh clone of this repo would be missing three of its seven status
directories.**

Two independent defects, both in this epic's "the audit is the only component that sees every
ticket" theme:

1. **`audit.Audit` (`internal/audit/audit.go:28`) has no directory-existence check**, and cannot
   acquire one incidentally: it consumes `ticket.LoadAll`, which at
   `internal/ticket/ticket.go:529` deliberately `continue`s past a `ReadDir` error with the
   comment *"absent (vanished-empty) dir is not an error"* — because git does not track empty
   directories. So the absence is swallowed one layer below the audit. Any check must either look
   at the directories directly or have `LoadAll` distinguish "empty" from "absent". Note the
   swallowing is load-bearing well beyond the WIP pre-check: `internal/move/move.go:62-65` turns
   **any** `LoadAll` issue into a hard `cannot move while the board has load problems` error, so
   reporting an absent dir there would break every move in a tree missing one. Do not simply make
   it an error in `LoadAll`.

   Also note the absence is **documented behaviour**: `docs/user-manual/cli-reference.adoc:379`
   states *"Missing empty status directories are treated as empty, not errors"*, while
   `docs/user-manual/concepts/project-structure.adoc:94` promises the `.gitkeep` files keep the
   seven statuses present on a fresh clone. The two already disagree; this ticket resolves it in
   favour of the `.gitkeep` contract.
2. **The `.gitkeep` scaffold is not preserved in this repo.** Whatever the audit learns to detect,
   the seven files should be tracked here — this is the same class of defect as T-028 above:
   a guarantee `install` makes to users that this repo, self-hosting the flow, does not keep.

Deliberately *not* urgent: `internal/move/move.go:124` runs `os.MkdirAll` before the rename, so
every move re-creates what it needs and the flow self-heals. This is a "fresh clone looks broken
and the audit lies about it" defect, not a functional one.

### Folded in from the 2026-07-27 field-finding triage — a fifth check

**History lines have a documented shape that nothing enforces.**
`skill/resources/TEMPLATE.md:116-124` requires *"One line per status transition, dated YYYY-MM-DD,
in the form `OLD → NEW: one-clause reason`"*, with a merge recorded as `merged to <base> (<MR
ref>)`. The audit never checks it, and the flow permits hand-authored tickets — so a migrated or
hand-edited ticket can carry a merge note that is a whole paragraph on one physical line.

Measured in the field (2026-07-27, migrating an 84-ticket hand-rolled flow into a fresh
`pickle install` workspace): one `6-done/` ticket's merge line was **~1,900 characters** — pipeline
ids, job names, fast-forward reasoning — and `board audit` reported it clean.

Why it belongs in this epic and not in **T-049**:

- Same file (`internal/audit/audit.go`), same table-driven test, same theme — *the audit is the
  only component that sees every ticket however it was authored*.
- T-049 caps the **rendered cell**, which makes the board legible while leaving the ticket
  malformed forever. The two are complementary, and the split is deliberate: **truncation must not
  become the way malformed history is hidden.**

Shape notes for refinement:

- The parse already exists and is per-line: `historyRE`
  (`internal/ticket/ticket.go:104`) plus `mergedRE` (`:106`). A check needs no new scanner — it
  needs a length/clause judgement over the bodies `historyRE` already yields.
- It must be a **warning, not an error**: an over-long reason breaks no invariant, and every
  existing ticket in a migrated workspace would fail on import. Compare this epic's other four,
  which are genuine malformations.
- Decide at refinement whether "one clause" is checkable at all, or whether the practical check is
  length only. Do not ship a heuristic that flags this repo's own legitimate multi-clause
  transitions (e.g. T-036's `review clean; 6 non-blocking, all dispositioned`).

**Measured at refinement (2026-08-01)** over all 303 History entries in this repo, classified by
kind, longest entry per kind in runes (continuation lines folded, as `HistoryEntries` folds them):

| kind | count | longest | example of the longest |
|---|---|---|---|
| status transition | 148 | **306** | T-036 `IN REVIEW → DONE: review PASS: 0 blocking, 11 non-blocking …` |
| merge line | 26 | **194** | T-054 `merged to main (PR #3, b70e238), user-approved as an exception; …` |
| created line | 64 | **331** | T-045 `created (TO DO). source: split out of T-036 at refinement …` |
| free-form dated note | 65 | **2199** | T-036's pickup-applicability-gate record |

This kills the blanket length check: the longest entries in the repo are **legitimate** free-form
notes (gate records, fold-ins, corrections) that the flow actively encourages, and they are longer
than the field's 1,900-character defect. The check must therefore be **kind-aware** — see D4/D5.

### Cross-references

- **T-049** — the render half of the same field finding: **DONE and merged**, so the 120-rune
  `maxCellRunes` cap (`internal/board/board.go:129`) already ships. The split holds: the cap makes
  the *board* legible, this ticket makes the *ticket* correct, and truncation must not become the
  way malformed history is hidden.
- **T-044** (which superseded T-039, 2026-07-26): **DONE and merged**. The audit's board
  cross-check is now the single staleness comparison at `internal/audit/audit.go:104-113`, and the
  old row-shape checks are gone — no sequencing needed any more.
- **T-045** (TO DO cap in the audit): **dropped** at its measurement gate, so there is no shared
  test scaffolding to inherit.
- **`pickle doctor`** (`internal/doctor/doctor.go:84`) checks only the skill dir's `SKILL.md` and
  `resources/tickets-README.md` — nothing under `tickets/`. Refinement decided the status-directory
  check belongs in `audit`, not `doctor` (see D3).
## Implementation Plan

### 0. Feature branch (mandatory)

The target child is **`pickle`** — this repo itself, at the root (`.`):

```
git switch -c feat/T-040-audit-frontmatter-and-tree-invariants main
```

Local WIP commits are fine. **No push and no merge request without explicit user approval**;
merging is the human's.

### 1. Prerequisite gate

- `depends-on: []` — nothing to wait for. T-044, T-049, T-058 and T-059 (all touching
  `internal/audit/audit.go`) are DONE **and merged**, so the file is stable.
- Working tree clean and `./pickle board audit` green *before* starting (it currently reports 0
  errors — which is precisely defect 4 in this Description).
- **Self-modify policy** (`AGENTS.md`): do **not** run `pickle install|upgrade` against this repo.
  Task 6 creates the missing directory and the `.gitkeep` files **by hand**; test installs go to a
  throwaway dir with the binary copied in.

### 2. Confirmed decisions

- **D1 — duplicate frontmatter keys are an `ERROR`; parse semantics are unchanged.** Last-wins
  stays (rewriting it would silently change what every existing ticket means); the audit merely
  reports. Detection shares the existing scan loop rather than adding a second one:
  `ParseFrontmatter` becomes a thin wrapper over an unexported
  `parseFrontmatter(text) (map[string]string, []string, bool)` whose middle return is the duplicate
  keys in first-seen order. The package doc's claim that "the frontmatter scan lives in exactly one
  place" must stay literally true.
- **D2 — TEMPLATE drift is guarded by a test, not a runtime check**, and the test lives in
  `internal/audit` — the only package that can see `requiredKeys`, `skill/resources/TEMPLATE.md`
  and `ticket.Scaffold` at once (a test in `internal/ticket` cannot import `internal/audit`:
  import cycle). It asserts three equalities, so drift fails in whichever direction it appears.
  A new `optionalKeys` list in `audit.go` gives the audit's key vocabulary (today: `family`) one
  home, instead of the optional-key concept living only inside an `if`.
- **D3 — a missing status directory is an `ERROR`, checked in `audit`, not `doctor`, and not in
  `LoadAll`.** `doctor` validates the *installation* (skill payload, markers); the seven status
  dirs are a `tickets/`-tree invariant, which is `audit`'s subject. `LoadAll` must keep swallowing
  the absence: `internal/move/move.go:62-65` turns any load issue into a hard failure. Also add a
  **warning** when a status dir exists, is empty and has no `.gitkeep` — the actual predictor of
  the fresh-clone defect, and still dependency-free (no `git` invocation).
  **Known consequence, accepted:** `move` (`move.go:141-147`), `sync` (`sync/sync.go:73`) and the
  post-`install`/post-`upgrade` self-checks (`cli/install.go:87,144`) all treat any audit error as
  a command failure. So in a project whose clone lacks a status dir, `pickle upgrade` now exits
  non-zero — and `upgrade` deliberately never writes under `tickets/`, so it cannot self-heal. The
  error message must therefore name the fix (`pickle install` is idempotent and re-scaffolds the
  seven dirs), and the docs NOTE must state it. This is the same upgrade-visible shape as the
  `spawned-by` required key, already documented at `cli-reference.adoc:366-377`.
- **D4 — the History check is a `WARNING`, length-only, and kind-aware.** "One clause" is not
  checkable: this repo's own transitions are legitimately multi-clause. It applies **only** to
  status transitions and merge lines — the two forms TEMPLATE prescribes — and **not** to `created`
  lines (provenance prose, measured up to 331 runes) or free-form dated notes (measured up to 2199
  runes, all legitimate). Threshold **400 runes**, measured on the continuation-folded logical
  entry (physical lines are meaningless: this repo wraps at 100 columns). That leaves every one of
  the 303 entries here green with ~25 % headroom and still catches the field's ~1,900-rune merge
  line by a factor of ~5.
- **D5 — exactly one History classifier.** The kind test (merge / created / transition / note) is
  implemented once as an unexported `historyKind`, surfaced as `HistoryEntry.Kind`, and
  `LastHistoryStatus`, `LastHistoryReason` and `MergeLine` are refactored to route their
  *skip* decisions through it. Adding a second classifier next to those three is exactly the smell
  T-042 tracks. **Order matters:** merge is tested first, so `T-001`'s
  `MERGED: feat/… → main (cdad65e)` stays a merge line and not a transition. The three scanners
  stay separate — folding them onto `HistoryEntries` would newly apply continuation-folding to
  `LastHistoryReason` and change the board's DROPPED/REWORK reason cell. Out of scope.
- **D6 — no regex touched.** The self-referencing `depends-on` check is `dep == t.ID`, mirroring the
  `spawned-by` (`audit.go:79`) and `family` (`:99`) guards. Unifying the `T-\d+` shape across
  `idRE`/`filenameRE`/`board.rowRE` is **T-042's**.
- **D7 — no new ticket for the self-host fix.** Creating `tickets/3-in-development/` and the seven
  `.gitkeep` files is the check's own fixture: shipping the error while this repo trips it would be
  absurd.

### 3. Tasks

1. **`internal/ticket/ticket.go` — one frontmatter scan, two answers.**
   Split `ParseFrontmatter` (`:186`) into the exported wrapper plus unexported
   `parseFrontmatter(text) (fm map[string]string, dupes []string, ok bool)`; a key already present
   in `fm` appends to `dupes` (each key reported once, first-seen order) and still overwrites.
   Add `DuplicateKeys []string` to `Ticket` (`:78-87`, documented as "malformed-input record; the
   parse still takes the last value") and populate it in `LoadAll` (`:552`) via the unexported
   scanner. No new exported parse function.
2. **`internal/ticket/ticket.go` — the single History classifier.**
   Add `type HistoryKind string` with `HistoryCreated`, `HistoryMerged`, `HistoryTransition`,
   `HistoryNote`; add `historyKind(body string) HistoryKind` (merge → created → arrow-with-legal-
   status → note, in that order, reusing `mergedRE`/`createdRE`/`StatusByName`); add
   `Kind HistoryKind` to `HistoryEntry` (`:136`) and set it in `HistoryEntries` (`:155`).
   Refactor `LastHistoryStatus` (`:256`), `LastHistoryReason` (`:295`) and `MergeLine` (`:336`) to
   decide *which entries to skip* by `historyKind`, keeping their extraction logic and their
   observable behaviour **byte-identical** (the existing tests are the guard).
3. **`internal/audit/audit.go` — the four ticket-level checks.**
   - Duplicate keys, in the per-ticket loop next to the `requiredKeys` check (`:47-51`):
     `errf("%s: frontmatter has duplicate key %q — remove one (the parse keeps the last value)", …)`.
   - Self-referencing `depends-on`, in the existing loop (`:67-71`):
     `errf("%s: depends-on lists itself", ref)` + `continue`, mirroring `:79`.
   - `optionalKeys = []string{"family"}` declared next to `requiredKeys` (`:24`), with a comment
     that D2's drift test reads both.
   - The over-long-entry warning, folded into the existing History ↔ directory loop (`:120-131`)
     so tickets are not walked a third time: iterate `ticket.HistoryEntries(t.Text)`, keep only
     `HistoryTransition` and `HistoryMerged`, and warn above `maxHistoryEntryRunes = 400`
     (a documented const carrying D4's measurements, sibling in spirit to `board.maxCellRunes`).
     Message names the date, the measured length, the limit, the prescribed form, and the fix
     ("put the analysis in the Description or `tickets/NOTES.md`").
4. **`internal/audit/audit.go` — the status-directory check.** New `auditStatusDirs(&r, root)`,
   called from `Audit` before the per-ticket loops, walking `ticket.Statuses`:
   `errors.Is(err, fs.ErrNotExist)` → `errf("tickets/%s/: status directory is missing — re-run
   pickle install (idempotent) to recreate it, and commit its .gitkeep")`; any other `ReadDir`
   error → `errf("tickets/%s/: %v")`; directory present, no `.gitkeep` and no `*.md` →
   `warnf("tickets/%s/: empty and not kept by a .gitkeep — git does not track empty directories,
   so a fresh clone will be missing this status")`.
5. **`internal/audit/audit_test.go` — fixtures first, then cases.**
   `writeGood` currently creates only `tickets/1-to-do/`, so Task 4 would turn every "clean" case
   red: add a `mkStatusDirs(t, root)` helper (seven dirs, each with a `.gitkeep`) and call it from
   `writeGood` (`:70`), `TestAuditPrefixMatch` (`:241`) and `TestAuditFamilyCrossChild` (`:267`).
   `internal/move` and `internal/sync` fixtures go through a real `install.Run`, so they need no
   change — confirm by running the full suite. New table cases in `TestAudit`:
   duplicate key (error `duplicate key`); self-referencing `depends-on` (error `depends-on lists
   itself`); missing status dir (error `status directory is missing`); empty dir without `.gitkeep`
   (**warning**, so it needs a `wantWarn` field — the table currently asserts *zero* warnings on
   clean cases); over-long transition (warning); over-long **free-form note** and a 2199-rune note
   → **clean**, the anti-regression that keeps this repo's own History legal; over-long *created*
   line → clean.
6. **`internal/audit/audit_test.go` — the TEMPLATE drift guard.** New
   `TestFrontmatterKeysMatchTemplate`, reading `../../skill/resources/TEMPLATE.md` with the same
   `t.Skipf`-if-absent idiom as `TestScaffoldSectionsMatchTemplate`
   (`internal/ticket/ticket_test.go:317`). Asserts: (a) TEMPLATE's *active* frontmatter keys
   (`ticket.ParseFrontmatter`) == `requiredKeys` as a set; (b) TEMPLATE's *commented-out*
   frontmatter keys (`^#\s*([a-z-]+):` inside the `---` block) == `optionalKeys`, and are disjoint
   from `requiredKeys`; (c) `ticket.Scaffold(...)`'s frontmatter keys == `requiredKeys` (with no
   family) and == `requiredKeys` + `family` (with one). Leave the hand-written key list in
   `ticket_test.go:240` alone — pre-existing duplication belongs to T-042/T-043.
7. **Fix this repo's own tree (D7).** `mkdir -p tickets/3-in-development` and create
   `tickets/<each of the seven>/.gitkeep`, then `git add` them with **explicit pathspecs**. After
   this, `./pickle board audit` must be error- *and* warning-free.

### 4. Acceptance test

Run from the repo root; all must pass verbatim.

```
just lint
just test
just build
just docs-check
```

Then the behavioural checks, on the real tree:

```
# 1. the repo is now clean on the new checks too (0 errors, 0 warnings)
./pickle board audit

# 2. all seven status dirs exist and every one is kept by a tracked .gitkeep
ls -d tickets/[1-7]-* | wc -l          # => 7
git ls-files 'tickets/*/.gitkeep' | wc -l   # => 7
```

And the new checks firing, in a throwaway copy (never mutate the real tree):

```
D=$(mktemp -d) && cp -R tickets pickle.toml "$D"/ && cp pickle "$D/pk" && cd "$D"

# 3. missing status dir => ERROR
rmdir tickets/3-in-development && ./pk board audit; echo "exit=$?"
#    expect: ERROR: tickets/3-in-development/: status directory is missing … exit=1
mkdir tickets/3-in-development && touch tickets/3-in-development/.gitkeep

# 4. empty dir without .gitkeep => WARNING only
rm tickets/5-rework/.gitkeep && ./pk board audit; echo "exit=$?"
#    expect: WARNING: tickets/5-rework/: empty and not kept by a .gitkeep … exit=0
touch tickets/5-rework/.gitkeep

# 5. duplicate frontmatter key => ERROR
#    (corrected at review: `-pe '… if !$done++'` consumes the guard on line 1 and
#     never reaches the frontmatter, so the original one-liners were no-ops)
T=tickets/1-to-do/T-013-install-polish.md
perl -0pi -e 's/^impact: .*$/impact: low\nimpact: high/m' "$T"
./pk board audit; echo "exit=$?"   # expect: … frontmatter has duplicate key "impact" … exit=1

# 6. self-referencing depends-on => ERROR
perl -0pi -e 's/^depends-on: .*$/depends-on: [T-013]/m' "$T"
./pk board audit   # expect: … depends-on lists itself   (and no "does not exist" error)

# 7. an over-long transition warns; an equally long free-form note does not
#    (append a 600-char transition to one ticket, a 2000-char note to another)
```

Expected end state: checks 3–6 each produce exactly the one named finding at the named severity,
check 7 warns only for the transition, and the real repo (check 1) is silent.

### 5. Docs update

- `docs/user-manual/cli-reference.adoc`, `<<cmd-board-audit>>` (`:343-380`): add the four new
  runtime checks to the bullet list (duplicate frontmatter keys; a ticket citing itself in
  `depends-on:`, alongside the `spawned-by:` clause already there; the seven status directories
  exist, with the `.gitkeep` warning; the History-entry length warning with its 400-rune figure and
  its deliberate exemptions). **Rewrite the trailing sentence at `:379`** — "Missing empty status
  directories are treated as empty, not errors" is now false; state instead that `LoadAll` still
  treats an absent dir as empty while the audit reports it, and why (the `.gitkeep` contract at
  `concepts/project-structure.adoc:94`). Extend the existing upgrade `NOTE` (`:366-377`) with the
  two new findings an *existing* project can trip through no fault of its own, including that
  `pickle upgrade` fails its post-upgrade self-check until a missing dir is restored, and that
  `pickle install` is idempotent and re-scaffolds it.
- `skill/resources/tickets-README.md` §1: one clause recording that a transition/merge History line
  has a length contract the audit warns on (the rules already prescribe the form; nothing else
  moves). The skill is the payload — the same edit reaches installed projects via `pickle upgrade`.
- `skill/SKILL.md`, *Procedure: audit the board*: extend the invariant list with the new checks,
  keeping it a one-sentence summary.
- No change to `AGENTS.md`'s marker block (it does not enumerate audit checks) and no
  `payload_version` bump by hand (it is stamped from the binary version at install/upgrade).

### 6. Finish

Summarise: five checks added at the one component that sees every ticket however it was authored,
one classifier unified, and this repo's own seven status directories made real. Suggested commit
message:

```
feat(audit): validate frontmatter, status dirs, and History shape (T-040)
```

Commit locally on `feat/T-040-audit-frontmatter-and-tree-invariants`. **Do not push and do not open
a merge request without explicit user approval**; after approval, finalize (squash or keep history —
the user chooses), push, open the MR, and leave the merge to the human. Then
`pickle ticket move T-040 in-review --reason "acceptance green"`.

## Review

Reviewed 2026-08-03 against `main` (the branch was merged early, out of protocol order, at
explicit user instruction — commit `8a61d3e`, PR #11). Verdict: **PASS — 0 blocking, 9
non-blocking.**

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — **skipped**: the `docs_readability` reviewer errored
      (`model_not_supported` from the configured Gemini backend), a sanctioned conscious skip (step 4b)
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval; bookkeeping committed with explicit
      pathspecs (step 9)

### Implementation audit

All seven tasks done in the files they name. `just lint`, `just test`, `just build`,
`just docs-check` all green; `./pickle board audit` → **64 tickets, 0 errors, 0 warnings**;
`ls -d tickets/[1-7]-* | wc -l` → 7 and `git ls-files 'tickets/*/.gitkeep' | wc -l` → 7.
Behavioural checks 3–7 re-run in a throwaway copy: missing status dir → the named ERROR (exit 1);
`.gitkeep` removed from an empty dir → the named WARNING only (exit 0); duplicate `impact:` → the
named ERROR; `depends-on: [T-013]` on T-013 → `depends-on lists itself` **and no** "does not
exist" error; a 615-rune transition warns while a 2000-rune free-form note stays silent. The D2
drift guard was mutation-tested (added `owner` to `requiredKeys` and `epic` to `optionalKeys`):
all three assertions fail as designed. The documented recovery path was exercised end-to-end on a
fresh `pk install` tree: with a status dir removed, `pickle upgrade` exits **1** with the named
error, and a re-run of `pickle install` restores the dir + `.gitkeep` and returns the audit to
clean. Every confirmed decision D1–D7 honoured; coverage `internal/audit` 92.7 %,
`internal/ticket` 94.4 %.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | fixed inline | The folded-loop comment claimed "History is scanned once, not twice" — false: `LastHistoryStatus` runs its own line scan in the same loop. The plan's actual claim was "tickets are not walked a third time". | `internal/audit/audit.go:163` | Reworded to the plan's claim. |
| N2 | non-blocking | fixed inline | `optionalKeys`' doc said the keys are "validated only when present", implying the list drives validation; it drives none — `family`'s checks are hand-coded and only the drift test reads the list. | `internal/audit/audit.go:28-34` | Comment now states the list validates nothing at runtime and says why it exists. |
| N3 | non-blocking | fixed inline | `idRE`'s comment deferred the `T-\d+` unification to "T-027's call"; T-027 was absorbed into this epic and dropped, and D6 reassigns the work to T-042. Dead pointer this ticket's own decision made false. | `internal/ticket/ticket.go:107` | Now points at T-042 and records T-027's fate. |
| N4 | non-blocking | fixed inline | The plan's acceptance-test steps 5 and 6 are **no-ops as written**: `perl -i -pe '… if !$done++'` burns the guard on line 1 (`---`) and never reaches the frontmatter, so both "expected ERROR" checks pass silently on an unmodified file. | Re-ran verbatim: 0 errors, file unchanged. `perl -0pi -e 's/…/m'` fires both errors correctly. | Plan's snippet corrected in place, with a note on why. |
| N5 | non-blocking | noted | The 400-rune threshold's stated basis is already outdated by this very ticket: the repo's longest **merge** line is now T-040's own at **352 runes** (refinement measured 194), so real headroom is ~12 %, not the documented ~25 %. Nothing warns today, and the pattern — merge lines carrying approval-exception prose — is the D4 false-positive risk. | `internal/audit/audit.go:37-47`; measured over 308 entries: note 2175, merged 352, created 329, transition 304. | Re-measure before lowering the limit; if merge prose keeps growing, exempt merge lines or raise to 500. |
| N6 | non-blocking | noted | The empty-dir warning tests for `.gitkeep` **or** a `*.md` file, so a status dir holding only non-`.md` content is reported as "empty" although git tracks it and the fresh-clone defect cannot occur. | `internal/audit/audit.go:265-277` | Harmless (a warning, and status dirs hold only tickets by contract); revisit only if it ever fires falsely. |
| N7 | non-blocking | noted | Plan task 5 asked for a **2199-rune** free-form-note anti-regression case; the test uses 450 runes. | `internal/audit/audit_test.go` "over-long free-form note stays clean" | Equivalent in substance — the check is kind-gated, so length is never consulted for notes; verified separately at 2000 runes on a real tree. |
| N8 | non-blocking | folded (T-042) | T-042's cross-reference still reads "if T-040 defers that, it belongs here" — D6 settled it, and its two line references have drifted (`ticket.go:95` → `:100`, `board.go:29` → `:51`). | `tickets/1-to-do/T-042-…md:82-83` | Ticket patched by the step-8 sweep; T-042 now owns the unification outright. |
| N9 | non-blocking | folded (T-056) | T-056's soft coupling assumes T-040 removes the silent last-wins hazard for a field writer. D1 deliberately kept parse semantics unchanged — the audit only *reports* duplicates, so a writer still needs its own guard. | `tickets/1-to-do/T-056-…md:173-175` | Ticket patched by the step-8 sweep; the coupling stays real but is now stated accurately. |

**Disposition summary:** 9 non-blocking — 4 fixed inline (N1–N4), 3 noted (N5–N7), 2 folded into
existing tickets (N8 → T-042, N9 → T-056). 0 blocking, 0 new tickets.

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-027, T-028 and T-033,
  all three moved to 7-dropped/ as absorbed
- 2026-07-26 — a fourth check folded in from the **T-036 pickup applicability gate** (rather than
  spawned as its own ticket): status directories are never validated. `3-in-development/` is
  currently absent and `board audit` reports 0 errors; no `.gitkeep` is tracked anywhere under
  `tickets/`, so a fresh clone lacks three status dirs. Fits this epic's existing theme and file
  (`internal/audit/audit.go`), and the absence is swallowed by `ticket.LoadAll`'s deliberate
  vanished-empty-dir `continue`, which the WIP pre-check depends on — so it needs this epic's
  judgement, not a standalone ticket. Also corrected a reference this epic inherited: the TO DO cap
  moved from T-036 to T-045 at T-036's refinement.
- 2026-07-27 — a fifth check folded in from the field-finding triage: History lines have a
  TEMPLATE-documented shape the audit never enforces (a ~1,900-character merge line audited clean).
  Render half filed separately as T-049.
- 2026-08-02 — refined. Description re-verified against the tree: all five defects still real, but
  the T-030-review correction about `ticket.ValidID` is **superseded** (it now has two external
  callers, and the self-reference check needs no shape check at all — it mirrors the `spawned-by`
  and `family` guards added since by T-024/T-059); stale line references and the T-044/T-045/T-049
  cross-references updated (all three are now terminal). The fifth check was settled by
  **measurement**: all 303 History entries in this repo, classified by kind — transitions max 306
  runes, merge lines 194, created lines 331, free-form notes 2199 — which rules out a blanket length
  rule and fixes the check as a kind-aware, transitions-and-merges-only **warning** at 400 runes.
  Seven decisions taken with the user: (D1) duplicate keys are an error, parse semantics unchanged,
  one scan loop; (D2) TEMPLATE drift is a test in `internal/audit` (import cycle rules out
  `internal/ticket`) with a new `optionalKeys` list; (D3) a missing status dir is an error in
  `audit` — not `doctor`, not `LoadAll` — plus a `.gitkeep` warning, accepting that `move`, `sync`
  and post-`upgrade` self-checks turn it into a command failure until `pickle install` re-scaffolds;
  (D4/D5) the History warning above, implemented behind one unexported `historyKind` that the three
  existing readers are refactored onto; (D6) no regex touched — the `T-\d+` unification stays
  T-042's; (D7) this repo's own missing directory and seven `.gitkeep` files are fixed in the same
  branch. No split: all five checks land in one file with one test table (rules §3). Re-graded
  impact medium-high → **medium** (a quality/consistency win, not an adoption lever), complexity
  low → **medium** (five checks plus the classifier refactor), cost **M** unchanged.
- 2026-08-02 — TO DO → READY: plan complete
- 2026-08-02 — READY → IN DEVELOPMENT: picked up
- 2026-08-02 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-02 — merged to main (PR #11, 8a61d3e), user-approved as an exception **ahead of the
  review protocol** (out of the usual review-then-merge order, at explicit user instruction);
  squashed to one commit; branch deleted. The ticket itself stays in IN REVIEW — the code being
  on main does not substitute for "review ticket T-040" recording findings and a verdict.
- 2026-08-02 — IN REVIEW → DONE: review PASS: 0 blocking, 9 non-blocking (4 fixed inline, 3 noted, 2 folded into T-042/T-056)
