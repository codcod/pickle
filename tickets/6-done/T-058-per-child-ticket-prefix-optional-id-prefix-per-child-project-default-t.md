---
id: T-058
title: per-child ticket_prefix: optional id prefix per child-project, default T
project: pickle
depends-on: []
spawned-by: [T-045]
impact: medium
complexity: medium
cost: M
---

# T-058 — per-child ticket_prefix: optional id prefix per child-project, default T

## Description

Spawned from exploration under **T-045** (2026-07-28): a comparison of unity's live 91-ticket
board against pickle's own 17-ticket backlog, run to test schema ideas for picking the next
refinement candidate (see T-059 for the other half of that exploration).

Today ids are one global `T-NNN` namespace, monotonically increasing, never reused, shared
across all child-projects (`tickets/README.md` rules §3) — including an explicit "a ticket may
even be re-homed to another child without renumbering" guarantee. The ask (from a user example:
`RICK-137` instead of `T-137` for a `rick` child, `SB-137` instead of `T-138` for `snowball`)
implies **per-child counters**, not a cosmetic prefix over one global counter.

Add an optional `ticket_prefix` key per `[[project]]` in `pickle.toml` (the six config stations
already inventoried by T-045, plus `pickle project list` and the `AGENTS.md` marker block).
Absent ⇒ defaults to `T`.

**Open decisions any refinement must pin down:**

- **D1 — per-child counters vs. a global counter with a decorative prefix.** Per-child counters
  break the "re-home without renumbering" guarantee (re-homing becomes a rename + reference
  rewrite) but avoid sequence gaps. A global counter with a decorative prefix preserves
  re-homing but produces gappy per-child sequences (`RICK-137`, `RICK-140`, `SB-138`), which
  looks broken and defeats the main reason people want per-child numbering. Exploration leaned
  toward **per-child counters**, with a documented "ids are never globally unique; always
  qualify" rule — but this is a user-facing tradeoff, not a mechanical default, and needs
  explicit sign-off at refinement.
- **D2 — prefix shape/uniqueness.** `^[A-Z][A-Z0-9]{0,7}$`, unique across children, validated in
  config; `T` reserved as the default when unset.
- **D3 — keep `ValidID` config-free as a pure shape check.** Today it's `^T-\d+$`
  (`internal/ticket/ticket.go:101`, error text "expected T-NNN"). Widen to
  `^[A-Z][A-Z0-9]*-\d+$` and add a **new** audit invariant: a ticket's id prefix must match its
  `project:`'s configured prefix — a bonus check, since nothing today catches a mis-filed
  `project:` frontmatter.
- **D4 — back-compat.** Prefix absent defaults to `T`. Mixed prefixes must render in one board
  forever (`6-done/`/`7-dropped/` are permanent archives), so sort becomes prefix-then-number.
- **D5 — migration is a real tool, not a bare rename.** Renaming an existing child's tickets
  touches filenames, `id:` frontmatter, every `depends-on`/`spawned-by` reference, `## History`
  lines, the board, branch names, and `NOTES.md` prose (which tooling must not touch). Commit
  subjects already in git citing the old id are unfixable — needs an old→new mapping table, the
  same shape as the artifact already produced for the unity/pickle onboarding migration
  (`development/pickle/onboarding.md` in the unity workspace).

**Code surface identified (exploration pass, not exhaustive):** `filenameRE` + `idRE`
(`internal/ticket/ticket.go:94,101`), `NextNum` (`ticket.go:392`, currently strips the literal
`"T-"`), `fmt.Sprintf("T-%03d", …)` (`internal/cli/ticket.go:120`), `rowRE`
(`internal/board/board.go:51`), config plumbing for `ticket_prefix`, docs (9 `T-NNN` mentions
across `docs/user-manual/*.adoc`, `TEMPLATE.md`, `tickets-README.md`, `review-protocol.md`,
`SKILL.md`). One upside: `NextNum` can stay filename-only (per-prefix max) — unlike T-045's
backlog-cap concern, it never needs a full `ticket.LoadAll` pass, since the id itself carries
the child.

**Motivating evidence:** unity's own `NOTES.md` hand-annotates the child 7 times today (e.g.
"T-189 (snowball) → T-185 (rick)", "pickle T-051") — direct evidence the current id withholds
information humans want. Also: unity's pickle-migration decision record already accepted "52
ticket ids retired; 111 commit subjects cite a dead namespace" as a cost of merging pickle in
without prefixes. With per-child prefixes, a future child-project would merge in untouched
(e.g. `PK-001…PK-057`) — the single strongest argument, but **time-sensitive: it evaporates
once a migration has already run without it.**

**Explicitly does not solve the stated backlog-navigation problem** — in unity, 61 of 62 TO DO
tickets are one child (`rick`), so prefixes leave the 28-way impact tie exactly as it is (see
T-059). Sell this as identity/ergonomics, not backlog navigation.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                                   # pickle is the child; repo root is the child root
git checkout main
git checkout -b feat/T-058-per-child-ticket-prefix
```

Commit locally as you go. **Do not** run `pickle install|upgrade` against this repo from the
branch (self-modify policy, CLAUDE.md): the binary is the artifact under test. Marker-block /
config changes are made **by hand** in the diff; test installs go to a throwaway dir with the
binary copied in.

### 1. Prerequisites & gate

- No hard `depends-on:`. Lineage: `spawned-by: [T-045]`.
- Soft couplings (note, do not block): **T-060** (`pickle ticket renumber`, spawned from this
  refinement — the retroactive migration tool that this ticket's audit invariant makes
  *demanded* but does not itself provide); **T-040** (frontmatter audit) and **T-041** (marker
  block drift) live next door but are independent.

### 2. Confirmed decisions

- **D1 — per-child counters** (user-confirmed 2026-07-28). `NextNum` becomes prefix-scoped:
  `NextNum(root, prefix)` returns `max(number of every id with that prefix across all status
  dirs) + 1`. Stays filename-only — no `LoadAll`. Accepted cost: rules §3's "re-home without
  renumbering" guarantee is retired (re-home now implies renumber, i.e. T-060), and numbers are
  no longer globally unique — **ids must always be qualified** (`RICK-137`, never "137").
- **D2 — prefix shape / uniqueness.** Effective prefix = `ticket_prefix` if set, else `"T"`.
  Shape `^[A-Z][A-Z0-9]{0,7}$`. **Uniqueness is enforced by exempting the effective value `"T"`
  from the duplicate check** — dedupe only non-`T` prefixes (amended 2026-07-28 per applicability
  gate: `Validate` runs *after* `applyDefaults` and gets no `MetaData`, so it cannot distinguish
  an explicit `ticket_prefix = "T"` from a defaulted one; exempting `"T"` is the equivalent,
  implementable rule, correct because `applyDefaults` only ever writes `"T"`). Two children may
  therefore both fall back to `T` (this *is* the pre-feature global namespace — existing
  multi-child workspaces do not break on upgrade); two children may not both set the same non-`T`
  prefix (a `Validate` error — a typo). This is the load-bearing back-compat decision: correctness
  never required prefix uniqueness because `NextNum` scans by prefix regardless of child, so a
  shared prefix is simply a shared counter.
- **D3 — `ValidID` stays config-free** (pure shape). Widen the shape everywhere from `T-\d+` to
  `[A-Z][A-Z0-9]*-\d+`. Add the config-aware invariant in the **audit**, not in `ValidID`.
- **D4 — back-compat.** Absent key ⇒ `T`; every existing `T-NNN` ticket keeps working. Board
  ordering is unaffected (each child sub-group is a single prefix, so numeric `Num` ordering
  within it is unchanged) — but `Num` parsing must strip the prefix *generically*, not the
  literal `"T-"`.
- **D5 — migration is out of scope here** (user-confirmed): shipped as **T-060**. This ticket's
  audit invariant (below) *is* the "refuse a half-migrated child" guard — a child whose
  `ticket_prefix` was changed while it still owns old-prefix tickets goes red until T-060 renames
  them. No extra refusal code needed.

### 3. Tasks (exact paths)

**a. Config plumbing** — `internal/config/config.go`
- Add field to `Project` (struct at :66): `TicketPrefix string \`toml:"ticket_prefix"\`` (no
  `omitempty` — `Render` is hand-written `fmt.Fprintf`, not `toml.Marshal`, so `omitempty` is
  inert; matches `branch_prefix` exactly).
- `applyDefaults` (:130) and `AddProject` (:192): if `p.TicketPrefix == ""` set it to `"T"`
  (mirror the `BranchPrefix` default exactly).
- `Validate` (:152): for each project validate the shape `^[A-Z][A-Z0-9]{0,7}$` against the
  **effective** prefix; track prefixes in a `seenPrefix map[string]bool` and error on a duplicate,
  **but skip the effective value `"T"`** so any number of default children may coexist (see D2).
  Add an exported `func (p *Project) Prefix() string` returning the effective prefix, so callers
  never repeat the `if "" then "T"` fallback.
- `Render` (:239): emit `ticket_prefix = %q` for each project (place it just above
  `branch_prefix`). Add a compile/round-trip check to the existing config tests.

**b. CLI `project add`** — `internal/cli/project.go`
- Add flag `ticketPrefix := fs.String("ticket-prefix", "T", "per-child ticket id prefix")` (:63
  block); set `TicketPrefix: *ticketPrefix` in the `config.Project{…}` literal (:87).
- `runProjectList` (:102): add a `PREFIX` column to the tabwriter header + rows.

**c. Id shape + generic split** — `internal/ticket/ticket.go`
- Add helper `func SplitID(id string) (prefix string, num int, ok bool)` — split on the last
  `-`, `Atoi` the tail.
- Widen `filenameRE` (:94) to `^([A-Z][A-Z0-9]*-\d+)-[A-Za-z0-9._-]+\.md$` and `idRE` (:101) to
  `^[A-Z][A-Z0-9]*-\d+$`. Update the `ValidID` error text at :239 from "expected T-NNN" to
  "expected <PREFIX>-NNN" and the load error at :511 similarly.
- `NextNum` (:389): new signature `NextNum(root, prefix string) int`; replace the
  `TrimPrefix(m[1], "T-")` at :401 with `SplitID` + `p == prefix` filter. **Signature break: the
  two existing `NextNum(root)` calls in `ticket_test.go:186,189` must gain the prefix arg** (folded
  into the §4.1 TestNextNum work).
- `LoadAll` (:526): replace `TrimPrefix(m[1], "T-")` with `SplitID(m[1])` for `Num`.
- **No change needed** in `internal/move/move.go` (keys off `LoadAll`'s `byID`, so widened
  `filenameRE` yields `RICK-001` keys for free) or `internal/serve/serve.go:179` (gates on the
  widened `ValidID`) — per-child ids move and route with no edits. Noted so a reviewer isn't
  surprised they're untouched.

**d. Id allocation** — `internal/cli/ticket.go`
- At :119-120 the project is already fetched/validated; use its `Prefix()`:
  `p, _ := cfg.Project(*project); prefix := p.Prefix(); id := fmt.Sprintf("%s-%03d", prefix,
  ticket.NextNum(root, prefix))`.

**e. Board row regex** — `internal/board/board.go`
- Widen `rowRE` (:51) to `^\|\s*([A-Z][A-Z0-9]*-\d+)\s*\|`. Confirm the `m[1] != "T-NNN"`
  skeleton-row skip at :91 still refers to the literal placeholder text (it does — leave it), and
  that `Sort`/`Num` need no change (per-child sub-groups are single-prefix).

**f. Audit invariant** — `internal/audit/audit.go`
- In the per-ticket loop (:46-84), after the existing registered-child check (:61-65): if the
  project is registered, compute `want := cp.Prefix()` and `got, _, ok := ticket.SplitID(t.ID)`;
  when `ok && got != want`, `r.errf("%s: id prefix %q does not match project %q prefix %q", ref,
  got, t.Project(), want)`. Skip when the project is unregistered or the id is unparseable (both
  already reported). This is the new mis-filed-`project:` / half-migrated-child detector.

**g. Marker block & help text — deliberately minimal (scope guard).**
- Keep the generated `AGENTS.md` marker block (`internal/install/install.go` `markerBlock()`
  :744-826, golden `internal/install/testdata/markerblock.golden`) and CLI help
  (`internal/cli/cli.go:97-100`, usage strings `internal/cli/ticket.go:36-37`) using `T-NNN` as
  the **illustrative** id shape. Per-child templating of the marker-block branch example is an
  accepted non-goal here (would churn the golden per workspace) — note it as a candidate polish
  in the Review if it bites. No golden change expected; if `just test` reports a golden diff,
  that is a real regression to fix, not to re-baseline.

### 4. Acceptance test (runnable)

Unit + integration, all via `just test`:
1. `internal/ticket/ticket_test.go`: add `TestSplitID` (happy + `"T-1"` + malformed `"foo"`);
   extend `TestNextNum` to assert per-prefix counting (files `T-003`, `RICK-001`, `RICK-002` →
   `NextNum(root,"T")==4`, `NextNum(root,"RICK")==3`, `NextNum(root,"SB")==1`).
2. `internal/config/config_test.go`: valid prefix accepted; bad shape (`"rick"`, `"TOOLONGXX"`,
   `"1X"`) rejected; two explicit `"RICK"` → error; two children both omitting the key (both
   effective `T`) → **accepted**; `Render`→`Load` round-trips `ticket_prefix`.
3. `internal/audit/audit_test.go`: a ticket `RICK-001` under project whose prefix is `RICK`
   passes; the same id under a project whose prefix is `T` (or default) yields exactly the new
   prefix-mismatch error; a legacy `T-001` under a default-prefix child still passes.
4. `internal/cli/cli_test.go`: register a child with `--ticket-prefix PK`, run `ticket new`,
   assert the created id is `PK-001` and the file/board row use it.
5. **Manual smoke (self-modify policy)** — never against this repo:
   ```
   D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q child-a child-b
   ./pk install ... && ./pk project add a child-a --ticket-prefix RICK \
     && ./pk project add b child-b --ticket-prefix SB
   ./pk ticket new "first" --project a   # expect RICK-001
   ./pk ticket new "first" --project b   # expect SB-001
   ./pk board audit                      # clean
   ```

Green bar = `just build` && `just test` && `just lint` all pass, plus the manual smoke shows
`RICK-001` / `SB-001` and a clean audit.

### 5. Docs

- `docs/user-manual/cli-reference.adoc`: document `project add --ticket-prefix` and the audit's
  new prefix↔project check.
- `docs/user-manual/concepts/tickets.adoc` (and/or `multi-project.adoc`): add the per-child
  prefix, the default `T`, and the **"ids are never globally unique across children — always
  qualify"** rule.
- `skill/resources/tickets-README.md` §3 (IDs): soften "one global namespace / shared across all
  child-projects / re-home without renumbering" to reflect per-child prefixes and counters; state
  that re-homing now implies renumber (T-060). Leave the umbrella-ticket idiom text (if present)
  untouched.
- Run `just docs-check` (snowball) — must pass.

### 6. Finish

- Write the summary; stage the child diff and commit on `feat/T-058-…` with a Conventional
  Commit subject ending `(T-058)`. **Publish-gated**: no push / MR without explicit approval.
- Overarching bookkeeping (this ticket file, board) committed separately with explicit
  pathspecs.
- `pickle ticket move T-058 in-review --reason "acceptance green"` and hand back.

## Implementation summary

Built on `feat/T-058-per-child-ticket-prefix` (commit `abc2c62`). Every plan task landed as
written after the gate amendments:

- **config** (`internal/config/config.go`): `Project.TicketPrefix` + `Prefix()`;
  `DefaultTicketPrefix = "T"`; defaulted in `applyDefaults`/`AddProject`; `Validate` checks shape
  `^[A-Z][A-Z0-9]{0,7}$` and rejects a duplicate **non-`T`** prefix (default `T` exempt →
  back-compat); `Render` emits `ticket_prefix`.
- **ticket** (`internal/ticket/ticket.go`): `SplitID`; `filenameRE`/`idRE` widened off the literal
  `T-`; `NextNum(root, prefix)`; `Num` parsed via `SplitID`; error texts generalised to
  `<PREFIX>-NNN`.
- **cli**: `ticket new` allocates `fmt.Sprintf("%s-%03d", cp.Prefix(), NextNum(root, prefix))`;
  `project add --ticket-prefix`; `project list` `PREFIX` column.
- **board**: `rowRE` widened. `Sort`/`Num` untouched (single-prefix sub-groups) — confirmed.
- **audit**: prefix↔project invariant added after the registered-child check.
- **move/serve**: untouched, as predicted (work off `LoadAll`/`ValidID`).
- **marker block & help**: left generic (`T-NNN` illustrative) per §3g — no golden churn.

**Acceptance — all green.** `just build`, `just test` (new: `TestSplitID`, extended `TestNextNum`
per-prefix, config prefix shape/duplicate/back-compat + Render round-trip, audit match/mismatch,
`TestTicketNewUsesChildPrefix`), `just lint`, `just docs-check` all pass. Manual throwaway-dir
smoke (never against this repo): a `core`(T)/`rick`(RICK)/`snowball`(SB) workspace produced
`T-001`, `RICK-001`, `RICK-002`, `SB-001` with a clean audit; flipping a populated child's prefix
turned the audit red with `id prefix "RICK" does not match project "rick" ticket_prefix "ZZ"`
(exit 1) — the D5 forcing-function guard, confirmed live.

**Docs:** `skill/resources/tickets-README.md` §3, `docs/user-manual/cli-reference.adoc`,
`concepts/tickets.adoc`, `concepts/multi-project.adoc` all updated for per-child prefixes and the
"ids unique only within a prefix — always qualify" rule.

## Review

**Verdict:** approved → `6-done/`. No blocking findings. Two non-blocking findings, both
**fixed inline** (stale statements this branch falsified; no behaviour change).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — skipped: no docs-readability reviewer reachable in this session (sanctioned conscious skip)
- [x] Findings recorded with severity + disposition; disposition summary present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket in `1-to-do/`/`2-ready/` depends-on T-058; T-059/T-060 cite it as `spawned-by:` lineage only (no assumption to patch)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

**Implementation audit.** All Implementation-Plan tasks landed in the named files:
config (`Project.TicketPrefix`/`Prefix()`, `DefaultTicketPrefix`, `applyDefaults`/`AddProject`
default, `Validate` shape + non-`T` duplicate check, `Render` emit), ticket (`SplitID`, widened
`filenameRE`/`idRE`, `NextNum(root, prefix)`, `Num` via `SplitID`, generalised error text), cli
(`ticket new` uses `Prefix()`, `project add --ticket-prefix`, `project list` `PREFIX` column),
board (`rowRE` widened), audit (prefix↔project invariant). `move`/`serve` untouched as predicted.
Marker block & help left generic per §3g — golden untouched (`git diff main…HEAD` on
`internal/install/testdata/` is empty). Acceptance test re-run verbatim: `just build`, `just test`
(incl. `TestSplitID`, per-prefix `TestNextNum`, config shape/duplicate/back-compat + Render
round-trip, `TestAudit/prefix_mismatch` + `TestAuditPrefixMatch`, `TestTicketNewUsesChildPrefix`),
`just lint`, `just docs-check` — all green (fresh `-count=1`). `./pickle board audit` clean.

**Quality / consistency.** `SplitID` splits on the last `-` and `Atoi`s the tail — robust to the
malformed inputs its test enumerates. `Validate` correctly exempts the effective `"T"` from the
duplicate check (the D2 back-compat decision), verified by `TestTicketPrefixBackCompat`. The
audit invariant fires only for registered, parseable ids, so it never double-reports.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | non-blocking | fixed inline | `pickle ticket new` reference still described id allocation as "one global namespace = `max(existing) + 1`" — falsified by this ticket's per-child counters. | `docs/user-manual/cli-reference.adoc:265` | Reworded to `<PREFIX>-NNN` = `max(existing ids sharing that prefix) + 1`. |
| R2 | non-blocking | fixed inline | Skill summary IDs bullet still said "one global namespace … shared across all children", contradicting the sibling `resources/tickets-README.md` §3 this branch updated. | `skill/SKILL.md:82` | Reworded to `<PREFIX>-NNN` per-child counters, default `T`, "always qualify across children". |

**Disposition summary:** 2 findings — 2 fixed inline (R1, R2); 0 blocking, 0 folded, 0 new
ticket, 0 noted. Both were whole-tree-sweep (step 4a.2) catches of the plan's §5 docs step, which
updated `tickets-README.md`/`tickets.adoc`/`multi-project.adoc`/the `project add` doc but missed
these two locations stating the old ID model.

## History

- 2026-07-28 — created (TO DO). source: pickle ticket new
- 2026-07-28 — applicability gate (pre-pickup): 1 blocking finding dispositioned inline —
  §3a uniqueness rule reworded to "exempt effective T" (Validate has no MetaData to detect
  explicitness); dropped inert `omitempty`; noted the two `NextNum` test-call edits and that
  move/serve need no changes. Plan not invalidated; proceeding.
- 2026-07-28 — TO DO → READY: plan complete: per-child counters (D1), migration split to T-060 (D5)
- 2026-07-28 — READY → IN DEVELOPMENT: picked up; applicability gate clean after inline amendments
- 2026-07-28 — IN DEVELOPMENT → IN REVIEW: acceptance green: build/test/lint/docs-check pass, smoke confirms per-child ids + audit guard
- 2026-07-28 — IN REVIEW → DONE: approved: acceptance green, 2 non-blocking findings fixed inline (stale ID-model docs)
