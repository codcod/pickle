---
id: T-008
title: board sync
project: pickle
depends-on: [T-002]
impact: medium
complexity: medium
cost: M
---

# T-008 — board sync

## Description

Implement `pickle board sync` — the escape hatch that repairs `tickets/BOARD.md` from the
ground truth (ticket files + frontmatter + `pickle.toml`) when hand-edits drift. The board
stays hand-maintainable; sync guarantees it is always *recoverable*. "In sync" is defined as
`internal/audit.Audit` returning **zero errors**, so a successful sync always leaves
`pickle board audit` clean.

Sync **fully regenerates the seven status sections** from ticket state (correct section per
directory, one `### <child>` sub-group per registered child, refreshed `(n/limit)` WIP counts,
deterministic ordering, `depends-on`/`branch` columns), while **preserving** everything it
cannot derive:

- The **preamble** above the first status heading is kept verbatim except the `Last updated:`
  line, which is refreshed.
- The **trailing appendix** (the `---` + "Dependency chain" / "Known soft couplings" free-form
  prose) is kept verbatim.
- **Human bookkeeping cells that are not in frontmatter** — DONE `merged`, DROPPED `reason`,
  REWORK `open findings` — are **carried over** from the existing board row for that ticket; a
  row sync must *add* gets a sensible default (`merged` → `no — publish-gated (branch …)`, the
  same string `ticket move` writes; `reason`/`open findings` → empty).

Shares the parsing/model layer with `board audit` (T-002) and the row model with `ticket move`
(T-007): `internal/ticket` (`Statuses`, `LoadAll`), `internal/board` (`RowData`,
`SectionColumns`, and new `RenderRow`/`ParseCells` helpers), `internal/config` (registered
children, per-child WIP, branch prefix). Phase P3.

**Confirmed decisions (user-approved 2026-07-24):**

- **D1 — never clobber human cells.** `merged`/`reason`/`open findings` are carried over from
  the current board; added rows default as above.
- **D2 — full regenerate**, not surgical minimal-diff: the seven status sections are rebuilt in
  the fixed skeleton board order (IN DEVELOPMENT, IN REVIEW, REWORK, READY, TO DO, DONE,
  DROPPED); preamble + appendix preserved verbatim (Last-updated refreshed).
- **D3 — terminal tickets (DONE/DROPPED) are listed only if already on the board.** Sync
  relocates/repairs/preserves them but never auto-adds every done ticket (matches audit's
  "terminal statuses may age off the board"). Non-terminal tickets always get a row.
- **D4 — deterministic ordering:** TO DO/READY by descending impact (tie → id ascending);
  every other section by id ascending.
- **D5 — `--dry-run`:** report drift, write nothing, exit non-zero if the board would change;
  default (no flag) applies the repair and runs the post-op audit self-check.

**Scope boundaries / cross-references:**

- **Cell escaping is out of scope** — a `|`/newline in a title/reason still malforms a row.
  That is T-014's shared-renderer fix; sync renders raw exactly as `ticket move` does today and
  will inherit the escaping fix when it lands. (Cross-ref T-014.)
- Sync is the "big hammer" full rebuild; `ticket move`'s incremental board edit (T-007) is the
  fast path. They are complementary — sync recovers whatever the incremental path drifts,
  including the T-014 stale-WIP-count and sub-group-spacing findings.

## Implementation Plan

**Prerequisite gate.** T-002 (audit) is DONE and merged (`fca3ea1`); its parsing/model layer
(`internal/ticket`, `internal/board`, `internal/audit`) and T-007's `internal/board`
row-rendering primitives are on `main`. Confirm `just build && ./pickle board audit` is clean
before starting.

**Branch.** `git checkout main && git checkout -b feat/T-008-board-sync`

### Confirmed decisions
D1–D5 above are settled. Implement exactly to them; do not re-open.

### Tasks

1. **Board API additions** — `internal/board/board.go`:
   - Export **`RenderRow(statusName string, d RowData) string`** — a thin wrapper over the
     existing `renderRow(SectionColumns(statusName), d)`; returns `""` for an unknown section.
   - Export **`HeaderRow(cols []string) string`** (`"| " + strings.Join(cols, " | ") + " |"`)
     and **`SeparatorRow(cols []string) string`** (`"|" + strings.Repeat("---|", len(cols))`),
     and refactor `insertIntoBoard` to use them (single-source the table formatting; no
     behaviour change — existing board tests must stay green).
   - Add **`ParseCells(path string) (map[string]map[string]string, error)`** — the carry-over
     reader: walk the board like `Parse`, but for each ticket row capture a map of
     `column-name → cell value` keyed by the section's `SectionColumns`. Return `id → {col:
     value}`. This is what D1 reads to preserve `merged`/`reason`/`open findings`.

2. **New package `internal/sync`** — `internal/sync/sync.go`, package `sync` (mirrors
   `internal/move`'s shape and its `audit.Audit` self-check):
   - `func Sync(root string, cfg *config.Config, dryRun bool) (Result, error)` where
     `type Result struct { Changed bool; Summary []string; Path string }`.
   - **Load ground truth:** `ticket.LoadAll(root)`; if it returns structural issues, fail with
     `cannot sync while the board has load problems: …` (same guard `move.Move` uses — sync
     repairs the board, not broken ticket files).
   - **Read current board** once: raw bytes (for the preamble/appendix split + the
     idempotency compare) and `board.ParseCells` (carry-over) and `board.Parse` (which terminal
     tickets are currently listed, for D3).
   - **Split the board** into three parts:
     - `preamble` = lines before the first `## ` heading whose upper-cased text starts with a
       status display name (reuse the longest-match logic from `board.Parse`). Refresh only the
       `Last updated:` line → ``Last updated: <YYYY-MM-DD> (board sync)``.
     - `region` = the status sections (regenerated wholesale — old content discarded).
     - `appendix` = from the first *non-status* `## ` heading occurring after the region to EOF,
       **backing up over any immediately preceding blank lines and a single `---` rule** so the
       separator is preserved. Empty if there is no such heading.
   - **Preserve each section's heading text.** Capture the existing full `## …` line per status
     from the old board (e.g. `## TO DO (impact order, per child)`); for a status section that
     was entirely absent, fall back to a canonical default heading.
   - **Build the model:** for each status in the fixed board order
     `["3-in-development","4-in-review","5-rework","2-ready","1-to-do","6-done","7-dropped"]`,
     for each registered child in `cfg.Projects` order, collect that child's tickets in that
     status dir. **D3:** for terminal statuses (`6-done`,`7-dropped`) include a ticket only if
     its id is in the current board's row set. **D4 ordering:** `2-ready`/`1-to-do` →
     descending `impactRank` then id ascending; all other sections → id ascending.
   - **Render each row** via a local `rowFor(t, cfg, status, carry)`:
     - Derivable cells from frontmatter (`id`, `title`, `impact`, `complexity`, `cost`,
       `depends-on` via the same `[a, b]` rendering `move` uses) and `branch` from the child's
       branch prefix (`config.DefaultBranchPrefix` unless the child overrides) + `t.ID` + `-` +
       `t.Slug` (identical to `move.rowData`).
     - **D1 non-derivable cells:** `merged`/`reason`/`open findings` ← `carry[t.ID]["merged"]`
       etc. if present, else the default (`merged` → `no — publish-gated (branch <branch>)`;
       others → `""`).
     - Emit via `board.RenderRow(status.Name, d)`.
   - **Assemble the region** to match the skeleton format **byte-for-byte** (so an
     already-synced board round-trips to itself — see idempotency):
     `## <heading>` / blank / for each child: `### <child>`(+` (n/limit)` for in-dev/in-review)
     / blank / `board.HeaderRow(cols)` / `board.SeparatorRow(cols)` / rows… / blank / (next
     sub-group or next section). WIP count `n` = that child's ticket count in the section;
     `limit` from `cfg.Project(child)`.
   - **Compose** `preamble + region + appendix`; compare to the original bytes → `Changed`.
   - **`dryRun`:** populate `Summary` (human-readable drift list, e.g. per added/removed/moved
     id and "WIP count refreshed"), do **not** write, return.
   - **apply:** `os.WriteFile` the board, then **post-op self-check** — `audit.Audit(root,
     cfg)`; if it reports errors, return `sync applied but board audit still reports N
     error(s): …` (surfaces a genuine ticket-side problem sync can't fix).

3. **CLI wiring** — `internal/cli/board.go`:
   - Replace the `notImplemented("P3", "board sync", …)` stub in `runBoard`'s `case "sync"`
     with `return runBoardSync(args[1:])`.
   - `runBoardSync(args []string) int`: parse `--dry-run`; `loadConfig()`; call
     `sync.Sync(cfg.Root(), cfg, dryRun)`. Print the summary. Exit codes: on error →
     `exitError`; **dry-run + `Changed` → `exitError`** (so CI fails on drift); otherwise
     `exitOK`. Add a `boardSyncUsage` and reject unknown flags with `exitUsage`.

4. **Tests** (table/golden style, matching the existing suites):
   - `internal/board/board_test.go`: `TestRenderRowMatchesSection`, `TestParseCellsRoundTrip`
     (a board with DONE `merged` + DROPPED `reason` cells → `ParseCells` returns them), and
     confirm the `HeaderRow`/`SeparatorRow` refactor left `insertIntoBoard` output unchanged
     (existing tests cover this).
   - `internal/sync/sync_test.go`:
     - `TestSyncRepairsDrift` — golden: take a correct board, corrupt it (drop a row, mangle a
       WIP count, reorder a sub-group, add an orphan row, break sub-group spacing), `Sync` →
       assert output equals the expected golden and `audit.Audit` is clean.
     - `TestSyncIsIdempotent` — `Sync` an already-correct board → `Changed == false` and bytes
       unchanged; a second `Sync` after the first is byte-identical.
     - `TestSyncPreservesHumanCells` (D1) — a DONE row with a hand-authored
       `yes — merged … (sha)` merged cell survives a sync that repairs unrelated drift.
     - `TestSyncPreservesPreambleAndAppendix` (D2) — preamble prose + `Last updated` refresh +
       the "Dependency chain"/"soft couplings" appendix all intact.
     - `TestSyncTerminalMembership` (D3) — a done ticket **not** on the board is not added; a
       done ticket listed under the wrong section is relocated with its `merged` cell.
     - `TestSyncDryRunReportsWithoutWriting` (D5) — `dryRun` on a drifted board → `Changed ==
       true`, board bytes unchanged.
   - Test fixtures use `t.TempDir()` + a written `pickle.toml` (single child `pickle`), the
     pattern the config/audit/move tests already use.

### Acceptance test (run verbatim; must be green before review)

```sh
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"      # pickle repo
just build

REPO=/tmp/pk-sync; rm -rf "$REPO"; mkdir -p "$REPO"; BIN="$PWD/pickle"
cd "$REPO"
"$BIN" install --project pickle --path .    # scaffold flow + register child
"$BIN" ticket new "alpha feature" --project pickle --impact high   --complexity low --cost S
"$BIN" ticket new "beta feature"  --project pickle --impact medium --complexity low --cost S
"$BIN" ticket new "gamma feature" --project pickle --impact low    --complexity low --cost S
"$BIN" board audit                           # clean baseline

# take one ticket all the way to DONE and give it a hand-authored merged cell
"$BIN" ticket move T-001 ready
"$BIN" ticket move T-001 in-development
"$BIN" ticket move T-001 in-review
"$BIN" ticket move T-001 done --reason "acceptance walk"
sed -i '' 's#no — publish-gated.*#yes — merged (abc1234)#' tickets/BOARD.md   # human bookkeeping

# corrupt the board: drop T-002's row, mangle a WIP count, add an orphan row
grep -v '| T-002 |' tickets/BOARD.md > tickets/BOARD.tmp && mv tickets/BOARD.tmp tickets/BOARD.md
sed -i '' 's#(0/1)#(9/1)#' tickets/BOARD.md
printf '| T-999 | ghost row | low | low | S | [] |\n' >> tickets/BOARD.md
"$BIN" board audit && { echo "FAIL: audit should have errored on the corrupted board"; exit 1; } || echo "OK: corrupted board fails audit"

# dry-run reports drift, writes nothing, exits non-zero
cp tickets/BOARD.md /tmp/pk-sync-before.md
"$BIN" board sync --dry-run && { echo "FAIL: dry-run should exit non-zero on drift"; exit 1; } || echo "OK: dry-run non-zero on drift"
diff -q tickets/BOARD.md /tmp/pk-sync-before.md && echo "OK: dry-run wrote nothing"

# apply the repair
"$BIN" board sync
"$BIN" board audit                           # MUST be 0 errors
grep -q '| T-002 |' tickets/BOARD.md         && echo "OK: dropped row restored"
grep -q 'yes — merged (abc1234)' tickets/BOARD.md && echo "OK: human merged cell preserved (D1)"
grep -q 'T-999' tickets/BOARD.md             && { echo "FAIL: orphan row not removed"; exit 1; } || echo "OK: orphan row removed"
grep -q '(9/1)' tickets/BOARD.md             && { echo "FAIL: bad WIP count not fixed"; exit 1; } || echo "OK: WIP count refreshed"

# idempotent: a second sync changes nothing and dry-run now passes
cp tickets/BOARD.md /tmp/pk-sync-after.md
"$BIN" board sync
diff -q tickets/BOARD.md /tmp/pk-sync-after.md && echo "OK: sync is idempotent"
"$BIN" board sync --dry-run && echo "OK: dry-run clean after sync"

echo "ACCEPTANCE PASS"
```

Also: `just test` (all packages green, incl. the new `internal/sync` suite) and `just lint`
(`go vet` + gofmt) clean.

### Docs

- `README.md`: add a `## pickle board sync` section next to `## pickle board audit` — what it
  repairs, the D1 carry-over guarantee (never clobbers `merged`/`reason`/`open findings`),
  `--dry-run` for CI, and the "audit-clean is the definition of synced" contract. Note the
  cell-escaping caveat points at T-014.
- Flip the `board sync` line in the CLI-surface / status list from stub to done.

### Finish

- Dogfood: run `pickle board sync --dry-run` against **this** repo's board (expect clean, exit
  0 — proof the tool agrees the live board is already in sync).
- Local WIP commits on the branch as you go. Present the suggested Conventional Commit message
  and MR attributes for approval — do not push / open an MR without it.
- Suggested commit: `feat(cli): add board sync to rebuild BOARD.md from ticket state (T-008)`.
- Move T-008 to `4-in-review/` (History line + board row) and hand back for validation.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P3)
- 2026-07-24 — TO DO → READY: refined; D1–D5 confirmed (never-clobber human cells, full
  regenerate, terminal-only-if-listed, deterministic ordering, --dry-run)
