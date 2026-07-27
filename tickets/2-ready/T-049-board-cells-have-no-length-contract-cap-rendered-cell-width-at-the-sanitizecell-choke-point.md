---
id: T-049
title: board cells have no length contract: cap rendered cell width at the sanitizeCell choke point
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-049 — board cells have no length contract: cap rendered cell width at the sanitizeCell choke point

## Description

**No rendered board cell has a length contract.** `sanitizeCell`
(`internal/board/board.go:119-128`) normalises three things — pipes to `¦`, newline runs to one
space, outer whitespace trimmed — and deliberately nothing else. Every cell value it receives is
emitted verbatim at whatever length the ticket happens to carry, so one long string in one ticket
makes a whole status table unreadable.

Hit in the field (2026-07-27) while migrating an 84-ticket hand-rolled flow into a fresh
`pickle install` workspace: a `6-done/` ticket whose merge note recorded pipeline ids, job names
and fast-forward reasoning produced a **~1,900-character `merged` cell**, and the DONE table with
it. Not exotic to that migration, though — the trend is already visible in this repo:

- longest merge History line in `tickets/`: **171 characters**;
- the DONE `merged` cells for T-001, T-002, T-008, T-009, T-011 and T-044 already run
  90-125 characters (`yes — MERGED: feat/… → main (…), user-approved; branch deleted`).

### Scope: cap once, at the choke point

The fix is a **render-time** cap — never a stored field, which D3 (T-044) forbids: terminal facts
live only in the ticket, and the board is regenerated from them.

It belongs in `sanitizeCell` and nowhere else. That function's own doc comment already claims the
position — *"the single one-way choke point every rendered cell passes through … Nothing ever
parses a cell back, so there is no escape scheme to keep in sync (T-044 decision 9)"* — and
capping there fixes **every** column at once. Four are unbounded today, not just `merged`:

| column | source | why unbounded |
|---|---|---|
| `merged` | `ticket.MergeLine` (`internal/ticket/ticket.go:261-280`) | returns the whole body of the newest `merged …` History line |
| `reason`, `open findings` | `ticket.LastHistoryReason` (`internal/ticket/ticket.go:220-256`) | returns the whole `: <reason>` clause of the newest transition |
| `title` | `t.Front["title"]` | frontmatter is hand-authorable; `ticket new`'s own title cap is T-038's scope, and does not constrain a hand-written ticket |

A per-column patch in `cellFor` would therefore be the wrong shape: it would put a fourth
length policy beside the one place the T-044 design centralised.

### Confirmed decisions (from the 2026-07-27 triage)

1. **Cap at 120 runes with a trailing `…`** — enough for a full `MERGED: feat/… → main (sha),
   user-approved` line, short enough to keep a terminal-rendered table legible. Refinement pinned
   the exact rule (inclusive of the ellipsis) in the plan's decisions 2-5 and **validated the
   number against the corpus**: the longest cell in this repo's board is 117 runes, so 120 clips
   nothing that legitimately exists today.
2. **Rune-safe, and applied last.** The cap runs *after* the `|` → `¦` substitution, and `¦`
   (U+00A6) is multi-byte — a byte-length slice can cut mid-rune and emit a replacement
   character into the board. Count and cut runes, not bytes.
3. **Rejected alternative:** rendering only the parenthesised ref (`MR !69, 8acc044`) instead of
   truncating. `mergedRE` is `(?i)^merged\b` (`internal/ticket/ticket.go:106`) — parentheses are
   conventional, not required — so that shape needs an undefined fallback for every merge line
   without them. Truncation has no such hole.
4. `HasMergeLine`/cell agreement is **not** a constraint worth designing around: truncating a
   non-empty string yields a non-empty string, so the yes/no boolean is preserved for any cap ≥ 1.

### Deliberately out of scope

**Detecting the malformed History line that produced the 1,900-character cell.** A merge note
that long violates `skill/resources/TEMPLATE.md:116-124` (*"One line per status transition …
OLD → NEW: one-clause reason"*), so truncation makes the board legible while leaving the ticket
malformed. That is a lint, it belongs to the audit, and it has been **folded into T-040** (the
audit-validation epic — same file `internal/audit/audit.go`, same theme: "the audit is the only
component that sees every ticket however it was authored"). This ticket is the render half only.

### Couplings

Soft couplings (no `depends-on:`, no ordering enforced):

- **T-040** — owns the History-line-shape lint folded out of this ticket. Complementary: this one
  makes the board legible, that one makes the malformation visible. Neither blocks the other.
- **T-038** — caps title length at *input* in `validateTitle`/`Slugify` (`ticket new`). Different
  boundary: it cannot constrain a hand-authored ticket file, which the flow explicitly permits, so
  the render cap is still required. Do not treat either as covering the other.
- **T-042** — touches `internal/board`; sequence, do not run concurrently.
- **T-043** — its item 5 defers to T-044's one-way sanitisation; if it adds `sanitizeCell` tests,
  land this first so the cap is part of the table.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the target child and it is this repo (`.`):

```
git checkout main
git checkout -b feat/T-049-board-cell-width-cap
```

Short slug on purpose — the filename slug is 90 characters; precedent is
`feat/T-030-validate-ticket-new-input` and `feat/T-036-review-disposition-valves`.

Commit locally as you go. **Never push or open an MR without explicit user approval**
(publish-gated); end with a summary and a suggested Conventional Commit message.

### Prerequisite gate (hard)

- Clean tree; `just build && just test && just lint && just docs-check` green before the first edit.
- `./pickle board audit` reports **0 errors, 0 warnings** at the start — the acceptance test below
  asserts the board is *unchanged* by this ticket, which is only meaningful from a clean baseline.
- No concurrent work in `internal/board` — **T-042** collides there (`NOTES.md` records the
  sequencing). Check nothing is in `3-in-development/` touching that package.

### Confirmed design decisions (do not deviate without asking)

1. **One cap, in `sanitizeCell`** (`internal/board/board.go:124`). Not in `cellFor`, not per
   column. Every column inherits it: `id`, `title`, `impact`, `complexity`, `cost`, `depends-on`,
   `merged`, `reason`, `open findings`.
2. **`maxCellRunes = 120`**, a package-level constant in `internal/board/board.go` beside
   `cellBreakRE`. **Not** configurable — no `pickle.toml` key, no flag. Validated against the real
   corpus: the longest cell in this repo's board today is **117 runes** (T-045's title), so 120
   clips nothing legitimate that exists.
3. **The cap is inclusive of the ellipsis.** A rendered cell is never more than 120 runes: when
   the sanitised value exceeds 120 runes, keep the first **119** runes and append `…` (U+2026,
   one rune). A value of exactly 120 runes is emitted untouched.
4. **Runes, not bytes, and last.** Order inside `sanitizeCell` is: collapse newline runs →
   substitute `|` → `¦` → trim → **cap**. The substitution must precede the cap because `¦`
   (U+00A6) is two bytes: capping by byte length can slice mid-rune and emit U+FFFD into the
   board. Convert to `[]rune` (or use `utf8.RuneCountInString` + a rune-boundary slice); never
   `s[:120]`.
5. **Truncation is head-preserving** — keep the prefix, drop the tail. This is *why* 120 and not
   60: the `merged` cell's useful payload (`yes — MERGED: feat/… → main (<sha>)`) sits in the
   first ~100 runes, so the sha survives while trailing prose is dropped. Verified against the
   six long DONE cells in this repo (90-100 runes, sha at ~75).
6. **No exemption for `depends-on`.** A truncated list reads `[T-001, T-002, …` and loses ids
   — accepted: nothing parses cells back (`rowRE`, `internal/board/board.go:50`, captures only
   the id), the ticket file is the source of truth, and 120 runes holds ~15 ids. Do not add a
   special case; that is the per-column policy this ticket exists to avoid.
7. **The audit-staleness ripple is accepted and documented, not softened.** `audit.go:86-94`
   errors *"BOARD.md is stale or hand-edited — run pickle board sync"* whenever the file differs
   from a fresh render. After this ships, any project whose board holds a >120-rune cell gets that
   error until `pickle board sync` runs once. That is the correct behaviour for a generated
   artifact (T-044 D6) and the remedy is one command — do **not** weaken the staleness check to
   avoid it. It needs a CHANGELOG line (Task 4). This repo is unaffected (max 117 runes).
8. **Out of scope, deliberately:** the `### <child>` sub-group headings and the WIP-limit preamble
   lines render config values (`p.Name`), not ticket text, and do not pass through `sanitizeCell`.
   A pathological child name is a `pickle.toml` problem, not a board problem. Note it in the
   review if it bothers you; do not fix it here.

### Tasks

#### Task 1 — cap the cell in `sanitizeCell`

In `internal/board/board.go`:

- add `const maxCellRunes = 120` next to `var cellBreakRE` (line 117), with a one-line comment
  saying it is a *legibility* bound on the generated table, that the ticket file remains the
  unabridged source (D3), and that the longest real cell when it was chosen was 117 runes;
- extend `sanitizeCell` (line 124) with the cap as the **final** step, per decisions 3 and 4;
- update `sanitizeCell`'s doc comment (lines 119-123): it currently enumerates exactly three
  normalisations ("pipes become a broken bar … newline runs collapse … trimmed") and that list
  becomes wrong the moment the cap lands. Add the cap as the fourth, and keep the existing
  "nothing ever parses a cell back" sentence — it is the *reason* truncation is safe here.

Also update the package doc comment (lines 1-4) if it enumerates the sanitisation — check;
it currently says only "passes one-way through sanitizeCell", which stays true.

#### Task 2 — test the cap

Add `TestRenderCapsCellWidth` to `internal/board/board_test.go`, next to
`TestRenderSanitizesCells` (line 158), following that function's existing idiom (`mkTicket` +
`loadTree` + `Render`, and a directly-constructed `&ticket.Ticket{}` for values `ticket new`
would reject). Cover, as separate sub-cases:

1. **Over-long title truncated exactly.** A 300-rune title → the rendered cell is 119 runes plus
   `…`; assert the cell's rune count is exactly 120 **and** that it ends in `…`. Assert the rune
   count, not the byte count — a byte assertion would pass on a mid-rune cut.
2. **Boundary: exactly 120 runes is untouched.** No `…`, cell identical to input. Add 119 and 121
   if cheap — off-by-one here is the whole risk surface.
3. **No mid-rune cut.** A title of 200 multi-byte runes (e.g. repeated `é` or CJK) → assert the
   cell contains **no** U+FFFD and is valid UTF-8 (`utf8.ValidString`).
4. **Substitution precedes the cap.** A title of 200 `|` → the cell is 119 `¦` plus `…`, with no
   `|` anywhere in the rendered board. This is the test that fails if someone reorders the steps.
5. **`merged` keeps its prefix and ref.** A `6-done/` ticket with a 600-rune merge History line
   whose sha sits early → the cell still starts `yes — ` and still contains the sha (decision 5).
6. **`HasMergeLine` still agrees with the cell.** Same ticket: `ticket.HasMergeLine` is true and
   the cell does not read `no — publish-gated`.

Confirm `TestRenderSanitizesCells`, `TestRenderDerivedTerminalCells` and the determinism test
(line ~150) stay green untouched — their fixtures are short, so the cap must be invisible to them.

#### Task 3 — documentation

Two passages in `docs/user-manual/cli-reference.adoc` state the sanitisation contract and become
incomplete:

- **line 375-376** (`board sync`, the "everything on the board is derived" list): the bullet
  *"every cell passes one-way sanitisation (`|` → `¦`, newlines → space)"* — add the 120-rune cap
  with `…`, and say the ticket keeps the full text.
- **line 262** (`ticket new`): *"(A `|` in a title is fine: board cells are sanitised one-way at
  render time.)"* — still true; extend only if it reads as an exhaustive list. Judgement call.

Also check `docs/user-manual/your-first-project.adoc:107` ("the board's `merged` cell renders from
that line") — if it implies the cell shows the line verbatim, correct it.

#### Task 4 — CHANGELOG

Add to `## [Unreleased]` → `### Changed`: board cells are capped at 120 runes with `…`; the
ticket file keeps the full text; **projects with a longer cell will see `board audit` report the
board stale until `pickle board sync` runs once** (decision 7). The migration note is the point —
without it this is a silent audit failure in someone else's repo.

### Acceptance test

Run from the repo root:

```
just build && just test && just lint && just docs-check
go test ./internal/board/ -run 'TestRender' -v
```

All green, and `TestRenderCapsCellWidth` present with its six cases passing.

**Regression assertion — this repo's board must not change.** The longest cell here is 117 runes,
so the cap is a no-op on real data:

```
cp tickets/BOARD.md /tmp/board-before.md
./pickle board sync
diff /tmp/board-before.md tickets/BOARD.md      # expect: only the `Last updated:` line, or nothing
./pickle board audit                            # expect: 0 error(s), 0 warning(s)
```

**Cap verified end-to-end on a throwaway workspace** (never against this repo — AGENTS.md
self-modify policy; copy the binary out first):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
git init -q . && ./pk install --project demo --path . >/dev/null
./pk ticket new "short" --project demo >/dev/null
# hand-write an over-long merge History line into the ticket, move it to 6-done/, then:
./pk board sync && ./pk board audit
awk -F'|' '/^\|/ {for(i=2;i<NF;i++){gsub(/^ +| +$/,"",$i); if(length($i)>120) print length($i), $i}}' tickets/BOARD.md
```

Expected: the `awk` prints **nothing** (no cell over 120 characters), the long cell ends in `…`,
and `board audit` is clean. Then confirm decision 7's ripple deliberately: re-render with the
*old* binary to leave a long cell on disk, run the new binary's `board audit`, and observe the
`BOARD.md is stale` error — that is the documented behaviour, not a bug.

### Docs update (mandatory when user-facing)

Tasks 3 and 4 — `docs/user-manual/cli-reference.adoc` (the derived-board list),
`docs/user-manual/your-first-project.adoc` if it overstates the `merged` cell, and `CHANGELOG.md`
with the migration note. `just docs-check` must pass. No skill-payload change: the cap is renderer
behaviour, and `skill/resources/tickets-README.md` §6 describes *what* the board is, not cell
widths — confirm that reading before deciding to leave it alone.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs + CHANGELOG updated (Tasks 3-4).
3. Write a summary: files touched, the exact constant and truncation rule shipped, and anything
   deferred (e.g. the child-name heading path from decision 8).
4. Suggest a Conventional Commit message, ticket id in brackets at the end of the subject:

   ```
   fix(board): cap rendered cell width at 120 runes (T-049)

   <what and why; note the board-sync migration for projects with longer cells>
   ```

5. Commit locally on `feat/T-049-board-cell-width-cap`; **do not push or open an MR without user
   approval**. Present the message, then `pickle ticket move T-049 in-review --reason "acceptance
   green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: field finding, pickle 0.1.0 workspace migration; re-scoped at
  triage to the sanitizeCell choke point, audit-lint half folded into T-040
- 2026-07-27 — refined: plan written; cap pinned to 120 runes inclusive of the ellipsis, measured
  against a 117-rune real maximum; audit-staleness ripple accepted and documented (decision 7)
- 2026-07-27 — TO DO → READY: plan complete
