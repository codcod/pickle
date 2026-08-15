---
id: T-065
title: expose board and ticket state as a versioned JSON read projection
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: medium
cost: M
---

# T-065 — expose board and ticket state as a versioned JSON read projection

## Outcome

After this ships, `pickle board state --json` prints the whole ticket tree — every child, every
ticket's frontmatter, parsed History and dispositioned review findings, plus WIP counts and audit
health — as one versioned JSON document, so a question like "how are non-blocking findings
distributed across classes?" is a `jq` one-liner instead of a hand-written `awk` pass over
markdown that has to be re-derived (and re-corrected) every time it is asked.

## Description

`pickle` has **no machine-readable output at all** — `rg 'encoding/json' internal/` returns
nothing. Every command prints prose for humans. Any programmatic consumer (an agent harness
extension, a CI step, a git hook, a future web client) must either scrape formatted text or
re-implement the tree walk and the grading/WIP/audit rules that `internal/board` and
`internal/audit` already own.

Add a **read-only, versioned JSON projection** of board and ticket state, as **one new
subcommand**: `pickle board state --json`. The current surface is `install`, `upgrade`,
`doctor`, `uninstall`, `hooks {install,uninstall,status,run}`, `project {add,list,remove}`,
`flow {show,list}`, `ticket {new,move}`, `board {audit,sync}`, `changelog check`, `serve`,
`version` (`internal/cli/cli.go:45-79`, `board.go:15-27`, `ticket.go`) — there is no `board
state`, so nothing has to change shape to accommodate it.

**A projection of this data already exists in Go**, built for the `serve` templates, and reading
it is the fastest way to see what the wire format has to carry:

- `internal/serve/view.go:141` `buildBoard` — grouped board; its own comment notes it builds
  "the same map the board's own Render builds"
- `internal/serve/view.go:236` `buildTicket` — single-ticket view
- `internal/serve/view.go:386` `buildActivity` — History-derived activity
- `internal/serve/view.go:417-427` `ChildWIP.AtDevLimit()` / `AtReviewLimit()`
- `internal/serve/view.go:453` `buildHealth` — wraps `audit.Audit` inside `lock.WithShared`

These are **more** template-shaped than when this ticket was filed, not less: T-080 threaded a
`*flow.Definition` through every builder, and T-104 added `Lane`, `ChildRow`, `ChildFilter` and a
precomputed lowercased `Search` field that exists only so a `<script>` can do a substring test.
They carry `template.HTML` and predicate methods. So this ticket **writes a separate wire type**
and reads the tickets itself; it does not marshal the view structs, and it deliberately does not
rebase `serve` onto the new type either (see decision 2).

### Resolved at refinement (2026-08-15): scope

The two questions this ticket reserved for refinement were put to the user and answered. Both
answers are recorded as confirmed decisions in the plan; the reasoning is here.

**1. Does this ticket exist at all? — yes, as scoped, but narrowed to one command.** The honest
state is unchanged: **there is still no consumer.** Both prospective ones resolved themselves
without this projection — T-085 shipped with an `awk` recipe (`NOTES.md` § *"T-085's
pre-registered criterion — recorded so the 8th review after it ships can find it"*) and T-093
shipped `changelog check` with its own git walk. The rick interop seam runs the other way
(`rick status --json` flows *into* pickle; `NOTES.md` § *"Rick interop — the asks that live
upstream (2026-08-07)"*). What tipped the decision is the counter-evidence sitting in this
ticket's own History: **the finding-count figure quoted below has now been wrong twice** (165 →
347 → ~500), each time because it was hand-counted and then went stale. That is the defect a
projection removes, and it is measured rather than prospective.

**2. Does it project the `## Review` findings table? — yes, the middle option only.** The
closed-vocabulary columns (`id`, `severity`, `class`, `disposition`) plus the two raw closer
lines; **not** the three prose columns (`description`, `evidence`, `suggestion`). The corpus
evidence below is what settles it.

**3. One verb, not two.** `ticket show --json` is dropped from scope. `board state --json` emits
every ticket in the document, so single-ticket lookup is `jq '.tickets[] | select(.id ==
"T-065")'` — a second command would ship a second envelope, a second usage entry and a second
docs section to answer a question the first one already answers.

### The findings-table corpus is not one shape — it is twelve

The claim this ticket carried until now — that T-085 left "exactly one shape to parse against" —
is **false for the existing corpus**, and that is the single most important input to the wire
format. T-085 was **prospective only and explicitly did not backfill** (its confirmed decision 3;
`NOTES.md` records T-025 being dropped as "lineage archaeology with no consumer"). Measured on
`tickets/6-done/` at `00ff6b9`:

| findings-table header | tickets |
|---|---|
| `id \| severity \| disposition \| description \| evidence \| suggestion` | 33 |
| `id \| severity \| class \| disposition \| description \| evidence \| suggestion` (canonical) | 14 |
| `id \| severity \| disposition \| finding \| evidence \| suggestion` | 8 |
| `id \| severity \| disposition \| description \| evidence \| suggestion / resolution` | 4 |
| `severity \| description \| evidence \| disposition` | 2 |
| `id \| severity \| disposition \| description \| evidence \| resolution` | 2 |
| six further one-off variants, including two led by `#` and one ordered `# \| finding \| evidence \| severity \| disposition` | 6 |

Twelve distinct headers; the canonical one covers **8 of the 61** done tickets that carry a
findings table at all. So the parser must be **keyed by column name, never by position** — a
positional reader silently mis-columns three quarters of the corpus, which is worse than not
shipping. Current totals, measured the same way: **~500 finding rows across 61 of 62 done
tickets**, with a `Disposition summary` line in **45 of 62** (the figures this ticket previously
quoted — 347 / 53 / 40-of-53 — were correct on 2026-08-13 and are now stale).

That measurement is also what kills the "include everything" option and confirms the middle one:
a name-keyed parser over twelve headers is a bounded, testable problem when the fields it
extracts are four short closed-vocabulary strings, and an open-ended one when three of them are
multi-sentence prose containing pipes, backticks and embedded tables.

### Envelope and versioning

The payload carries a **top-level envelope** with an integer `schema` and the emitting binary's
version, so a consumer can refuse a dialect it does not understand instead of mis-parsing it.
This deliberately absorbs the version-handshake need that was originally proposed as a per-ticket
`schema_version` frontmatter key: a handshake belongs in the wire format, not in 104 ticket files.

**Pre-registered, deliberately not filed:** a `schema_version` key in ticket frontmatter, with
a fail-closed guard. It is unnecessary today because **no write path re-renders frontmatter** —
re-verified at `00ff6b9`: `internal/move/move.go:182` appends (`newText := appendHistory(t.Text,
…)`) and `internal/ticket/ticket.go:705` `Scaffold()` still has exactly one caller, for
brand-new files only (`internal/cli/ticket.go:163`), while `parseFrontmatter` carries unknown
keys through in `Front`. An unknown field cannot currently be dropped. File that guard **when,
and only when, a frontmatter re-render path is proposed** — i.e. as a prerequisite inside the
ticket field writer, which is what creates the hazard. That writer is **T-102**, whose
Description carries the pre-registration forward.

### Soft couplings (not `depends-on`)

- **T-101** (atomic writes + tree lock) — **done and merged.** It is why this command's traversal
  takes `lock.WithShared` rather than reading a tree a concurrent `ticket move` may be rewriting;
  `internal/serve/serve.go:144` and `internal/serve/view.go:461` are the two call sites to copy.
  T-101's own review concluded this ticket needs no other change from it.
- **T-052** (board drift) — **done.** `board audit`'s old single "stale **or** hand-edited"
  conflation is now `board.Drift` (`DriftNone`/`DriftLayout`/`DriftRows`,
  `internal/board/board.go:380-399`), surfaced as a warning (layout-only) or an error (rows
  differ). Board drift is exposed here under **that** vocabulary; do not invent a third naming.
- **T-085** (per-ticket record aggregable) — **done**, and the reason the findings table is in
  scope. Its pre-registered criterion (`NOTES.md`, § *"T-085's pre-registered criterion"*) is
  decided by counting `class` values across `6-done/`; **8 done tickets now carry the column**,
  so the criterion comes due soon and this projection is what makes evaluating it a `jq` filter
  rather than the `awk` recipe recorded there. Neither blocks the other.
- **T-093** (changelog reconciliation) — **done.** Its Description notes this projection "would
  give this a structured source"; it shipped without one and is not being reworked here.
- **T-043** (CLI test harness) — **done.** A new subcommand is exactly what it covers:
  `newProject(t)` for a throwaway install and `captureStdout`/`captureStderr`
  (`internal/cli/cli_test.go:150,250,259`) are the pattern to copy, and the package's
  no-`t.Parallel()` rule applies.
- **T-102** (ticket field writer) and **T-079** (`serve`'s first write route) — both still in
  `1-to-do/`. Neither is a prerequisite: this ticket writes nothing.

### Honest scope of the benefit, and the grade

**There is no consumer today, and the motivating one was withdrawn the same day this was filed.**
The candidate was an agent-harness extension enforcing flow gates; it was assessed against the
field record hours later and **not filed** — its one evidenced rule already belonged to T-057,
which had itself concluded a harness extension is the wrong primary mechanism (*"a pi extension
only guards a pi session"*). See `tickets/NOTES.md`, § *"Second postscript (2026-08-04)"*. The
other half of the original argument — *"and T-056 work area 1 would have to build this projection
regardless"* — **expired on 2026-08-14**: area 1 was not filed when T-056 was split
(`NOTES.md` § *"T-056 split (2026-08-14) — the XL umbrella dropped, four destinations"*), so no
other ticket will build it. This ticket owns the seam alone.

**Re-graded at refinement (2026-08-15): `low-medium` → `impact: low`.** The range had to collapse
(rules §3), and `low` is the honest end of it: with no consumer, this is a narrow addition that
changes nothing about how the flow behaves, and the 2026-08-04 precedent — applied when T-056 was
downgraded, again to decline a T-081 bump, and again by T-085 — **refuses to credit prospective
demand when grading**. T-085's arrival as a second prospective use does not move it; the rule cuts
both ways. `complexity: medium` and `cost: M` stand: one self-contained package, one command, and
a name-keyed table parser whose difficulty is bounded by the twelve headers measured above.

### What must not be assumed at implementation

- **That the `serve` structs can be marshalled as-is** — see above; they cannot, and this ticket
  neither marshals nor refactors them.
- **That `--json` should be added to the existing read commands** (`doctor`, `board audit`,
  `project list`, `version`, `changelog check`). Separable polish, different audience, explicitly
  out of scope — file it on its own merits if wanted.
- **That the projection should be writable, or served over HTTP.** Read-only, CLI-only. Locking
  is T-101 (**done**), the ticket field writer is T-102, and `serve`'s first write route is
  T-079.
- **That finding values should be mapped onto a vocabulary.** The parser normalises formatting
  (markdown emphasis, whitespace, case) and emits the resulting string verbatim. Mapping
  `**noted**` and `noted` and `→ T-091` onto a canonical enum is exactly the third-naming mistake
  the T-052 coupling above warns against, and it would make the 33 pre-`class` tickets
  unrepresentable.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                                   # `pickle` is the root-path child
git checkout main
git checkout -b feat/T-065-json-read-projection
```

WIP commits encouraged. **Root-path child** (`path = "."`): the Finish step tidies WIP into
atomic commits and keeps that history rather than squashing. Ticket and board bookkeeping stays
on `main`, never on this branch.

### Prerequisite gate (hard)

None. `depends-on:` is empty. T-101 (tree lock, `internal/lock`), T-052 (`board.Drift`), T-043
(CLI test harness) and T-085 (`class` column) are all in `6-done/` and merged; this ticket
consumes them but adds no hard dependency, because it degrades gracefully if any were absent.

### Confirmed design decisions (do not deviate without asking)

1. **One command: `pickle board state --json`.** No `ticket show`, no `--json` on any existing
   command. `--json` is **mandatory**, not a default: an invocation without it exits `2` with a
   usage line. Two reasons — a human who types `pickle board state` exploratively gets a usage
   hint instead of ~500 KB of JSON, and a later human-readable default is then not a breaking
   change for anything already scripted against the flag.
2. **New package `internal/state`, and `serve` is not touched.** Build the wire types and their
   builders there, reading tickets via `ticket.LoadAll` + `config` exactly as `serve` does.
   **Do not refactor `internal/serve` to consume the new types** — that is a second ticket's
   work against 500 lines of view code and 993 lines of tests, and it is not needed to ship this.
   Say so in the package doc comment so the duplication reads as deliberate.
3. **The output is a pure function of (tree, config, binary version).** No timestamp field, no
   wall-clock anything: two runs against an unchanged tree must be byte-identical, so `diff` is
   meaningful and golden-file tests are possible. (`BOARD.md`'s `Last updated:` line is the
   counter-example, and `board.Compare` already has to special-case it — do not repeat that.)
4. **Envelope.** Top-level object with `schema` (integer, `1`), `pickle_version` (the injected
   build version), `flow` (`cfg.FlowName()`), `root` (absolute path), then `states`, `children`,
   `tickets`, `health`. Adding a field is a compatible change; removing or retyping one bumps
   `schema`. State that contract in the docs section.
5. **snake_case keys throughout**, including for kebab-case frontmatter: `depends_on`,
   `spawned_by`. One rule, no per-field judgement.
6. **`states` is projected from the flow definition**, not hardcoded: for each of
   `def.BoardStates()`, emit `name`, `dir`, `terminal`, `wip_key`. A consumer must never have to
   hardcode `"3-in-development"`.
7. **`tickets` is a flat array in board order** — states in `def.BoardStates()` order, children in
   `cfg.Projects` order within a state, and `board.Sort(group, st, byID)` within a child group.
   Flat plus `status`/`project` fields lets a consumer group with `jq` and costs no nesting; the
   board's own ordering is preserved so the array reads the same way the board does.
8. **Ticket fields:** `id`, `num`, `prefix`, `title`, `project`, `status` (display name), `dir`,
   `file` (basename), `slug`, `impact`, `complexity`, `cost`, `depends_on`, `spawned_by`,
   `family`, `duplicate_keys` (from `ticket.Ticket.DuplicateKeys`), `front_matter` (the raw
   `Front` map, so unknown keys survive), `merged` (`ticket.MergeLine`, `""` when absent),
   `history` (`ticket.HistoryEntries` → `{date, kind, target, text}`), and `review`.
   Absolute paths appear only in the envelope's `root`; per-ticket paths are repo-relative, so
   two checkouts produce identical documents.
9. **`review` is the middle option, and nothing more.**
   `{tables, headers, findings, disposition_summary, cost_line}`:
   - `findings` carries **only** `{id, severity, class, disposition}`. The three prose columns
     (`description`/`evidence`/`suggestion`, under any of their aliases) are **not** projected.
   - `disposition_summary` and `cost_line` are the two closer lines, **verbatim and unparsed**
     (the `Disposition summary…` line and the `cost: estimated …, actual …` line). Raw strings
     cost one regex each and carry zero parsing risk.
   - `headers` is the raw, normalised column-name list of each table found — so a consumer (and
     the acceptance test) can see which variant was parsed rather than trusting it silently.
10. **Findings-table detection is by column name, never position.** A markdown table inside
    `## Review` is a findings table **iff** its header row contains, after normalisation, a
    column named exactly `severity` **and** one named exactly `disposition`. This is what
    excludes the near-miss headers already in the corpus (e.g. `check | old trigger | new
    trigger | severity before → after`, whose column is not `severity`). Column aliases to
    resolve, all of them present in the corpus: `#` → `id`. A column the table does not have
    yields `""` — never a value shifted in from a neighbour.
11. **Value normalisation is formatting only.** Strip surrounding `**`/`*`, collapse whitespace,
    lowercase, trim; emit the result verbatim. No mapping onto an enum, no validation, no
    dropping of unrecognised values (decision cross-referenced in the Description).
12. **Every table in `## Review` counts**, not just the first — re-review rounds add a second.
    Concatenate their rows into one `findings` array and report the count in `tables`.
13. **Health reuses existing vocabulary.** `health` is `{tickets, errors, warnings, board_drift}`
    — the first three straight from `audit.Result`, and `board_drift` one of `none`/`layout`/
    `rows` from `board.Compare` (T-052). Do not restructure `audit.Result` into typed findings;
    it is `[]string` today and widening it is a different ticket.
14. **The whole traversal runs inside `lock.WithShared(cfg.Root(), …)`**, copying
    `internal/serve/serve.go:144`. Unlike `buildHealth`, a **lock error here is fatal** — this
    command's contract is "a correct document or a non-zero exit", not a degraded page — so
    report it on stderr and exit `1`.
15. **No payload change.** Nothing under `skill/` is touched: this is a tooling command, not a
    flow rule, and no agent procedure needs to know about it. `payload_version` is untouched.

### Tasks

#### Task 1 — `internal/state`: the wire types

New file `internal/state/state.go`. Declare the envelope and its members as plain structs with
explicit `json:"…"` tags (decisions 4, 5, 6, 7, 8, 9, 13): `Document`, `State`, `Child`, `WIP`,
`Ticket`, `HistoryEntry`, `Review`, `Finding`, `Health`. No methods, no `template.HTML`, no
behaviour — these exist to be marshalled. Package doc comment records decision 2 (why `serve` is
not rebased on this) so the duplication is legible as a choice.

#### Task 2 — `internal/state`: the findings parser

New file `internal/state/review.go`. `parseReview(text string) Review`:

- take `ticket.SectionBody(text, "## Review")` (`internal/ticket/ticket.go:515`);
- walk it for markdown tables; for each, normalise the header cells (trim, strip emphasis,
  lowercase, collapse whitespace) and apply decisions 10 and 12;
- resolve the alias in decision 10, build a column-name → index map, and emit one `Finding` per
  body row (skipping the `|---|` separator), reading each of the four fields **through that map**;
- normalise values per decision 11;
- extract `disposition_summary` and `cost_line` verbatim (decision 9).

Table walking must tolerate what the corpus contains: rows whose prose cells hold backticks,
inline `\|`-escaped pipes, and (in a few tickets) a second unrelated table in the same section.

#### Task 3 — `internal/state`: the builder

New file `internal/state/build.go`. `Build(def *flow.Definition, root string, cfg *config.Config,
version string) (Document, error)`:

- `lock.WithShared(root, …)` around the whole traversal (decision 14);
- `ticket.LoadAll(def, root)` for the tickets; ordering per decision 7 using `board.Sort` and a
  `byID` map (same construction as `internal/serve/view.go:141`'s `buildBoard`);
- `board.WIPCounts(def, tickets)` + `p.WIPLimitFor(...)` for each child's WIP, keyed by
  `def.WIPStates()` rather than the two hardcoded fields `ChildWIP` uses;
- `audit.Audit(root, cfg)` and a `board.Compare` of the on-disk `BOARD.md` against a fresh
  `board.Render` for `board_drift` (decision 13);
- `parseReview` per ticket.

#### Task 4 — the command

- `internal/cli/board.go`: add `case "state": return runBoardState(args[1:])` to `runBoard`'s
  switch and to its usage line; implement `runBoardState` alongside `runBoardSync` — parse args
  (only `--json`, mandatory per decision 1; anything else → usage, exit `2`), `loadConfig()`,
  `state.Build(...)`, then `json.NewEncoder(os.Stdout)` with `SetIndent("", "  ")` and
  `SetEscapeHTML(false)` (ticket prose is full of `&`, `<` and `→`; escaping them helps no
  consumer). Errors go to stderr via the package's `errf` and exit `1`.
- `internal/cli/cli.go`: add `board state --json` to the *Flow commands* block of `usage()`.

#### Task 5 — tests

- `internal/state/review_test.go` — table-driven over the header variants measured in the
  Description: the canonical seven-column header, the 33-ticket six-column one, the
  `finding`-instead-of-`description` one, the `#`-led one, the `severity`-before-`disposition`
  reordering, and a near-miss header that must **not** be detected as a findings table. Assert
  the class column is `""` (not a shifted-in value) for every pre-`class` variant.
- `internal/state/build_test.go` — deterministic output (build twice from one tree, compare
  bytes); ordering matches `board.Render`'s for the same tree.
- `internal/cli/cli_test.go` (or a new `state_test.go` in that package) — `newProject(t)` +
  `captureStdout`: `board state --json` on a fresh install emits valid JSON with `"schema": 1`
  and an empty `tickets` array; `board state` without `--json` exits `2` and prints nothing to
  stdout. Remember: no `t.Parallel()` in `internal/cli`.

### Acceptance test

Run from the repo root on the feature branch. Every check is re-runnable verbatim.

```sh
just build && just test && just lint && just docs-check
```

**Expected: all green** (`go vet` clean, `gofmt` clean, `snowball check` clean).

Then, against this repo's own ticket tree:

| # | check | expected |
|---|---|---|
| 1 | `./pickle board state --json \| jq -e '.schema == 1 and (.pickle_version \| length > 0)'` | exits 0 |
| 2 | `./pickle board state` (no flag) | exit `2`, usage on stderr, **nothing on stdout** |
| 3 | `diff <(./pickle board state --json) <(./pickle board state --json)` | no output (decision 3) |
| 4 | `./pickle board state --json \| jq '.tickets \| length'` vs `find tickets -name 'T-*.md' \| wc -l` | equal |
| 5 | `./pickle board state --json \| jq -r '.tickets[] \| select(.status=="TO DO") \| .id'` vs the ids in `BOARD.md`'s TO DO section, in file order | identical sequences (decision 7) |
| 6 | `./pickle board state --json \| jq '[.tickets[].review.findings \| length] \| add'` | ~500, and within ±2 of `for f in tickets/6-done/*.md; do …; done` counting `\|`-rows under a `severity`+`disposition` header — any larger gap means rows are being dropped |
| 7 | `./pickle board state --json \| jq -r '.tickets[].review.headers[] \| @csv' \| sort -u \| wc -l` | **≥ 10** — proves the parser sees the corpus's real variety rather than only the canonical header (the tautology this check exists to rule out) |
| 8 | `./pickle board state --json \| jq -r '[.tickets[].review.findings[] \| select(.severity=="non-blocking") \| .class] \| group_by(.) \| map({(.[0]): length}) \| add'` | non-empty; the counts for non-`""` classes **match** the `awk` recipe in `NOTES.md` § *"T-085's pre-registered criterion"* run over the same tree |
| 9 | `./pickle board state --json \| jq -r '.states[].dir'` | the seven status dirs in board order, none hardcoded in `internal/state` (`rg '[0-9]-(to-do\|ready\|in-)' internal/state/` prints nothing) |
| 10 | `./pickle board state --json \| jq -e '.health.board_drift == "none" and (.health.errors \| length) == 0'` | exits 0 on a clean tree; after `printf '\n' >> tickets/BOARD.md` it reports `"layout"` or `"rows"`, and reverting restores `"none"` |
| 11 | fresh throwaway install — `D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --project demo && ./pickle-test board state --json \| jq -e '.tickets == [] and .schema == 1'` | exits 0 |
| 12 | concurrency (decision 14): in the throwaway dir, loop `./pickle-test ticket new …` / `ticket move` for ~20 s while looping `./pickle-test board state --json \| jq -e .schema >/dev/null` | zero non-zero exits, zero parse failures |
| 13 | `rg -n 'encoding/json' internal/serve/ internal/board/ internal/audit/` and `git diff --stat main...HEAD -- internal/serve/` | both empty — decision 2 held |

### Docs update (mandatory when user-facing)

User-facing: a new command. Update, all under `docs/user-manual/`:

1. **`cli-reference.adoc` overview table** — add a `pickle board state --json` row after the
   `board sync` row.
2. **`cli-reference.adoc`** — new `[#cmd-board-state] == pickle board state` section after
   `== pickle board sync` (currently `:768`), covering: the synopsis and the mandatory `--json`
   (decision 1); the envelope and its compatibility contract (decision 4 — additive fields are
   compatible, `schema` bumps otherwise); the top-level members; the ticket fields; and an
   explicit **caveat on `review`** — it is a best-effort read of hand-written markdown tables
   across twelve historical header shapes, the three prose columns are deliberately absent, and
   `headers` is there so a consumer can tell what it actually got. Close with two worked `jq`
   one-liners (find one ticket by id; count `class` values across DONE).
3. **`cli-reference.adoc:8-21`** — the shared/exclusive-lock paragraph names `pickle serve` as
   the only shared-lock reader. Add `board state` to it.
4. **`CHANGELOG.md`** — an `### Added` entry under `[Unreleased]`, in user terms, as a
   `docs(changelog): … (T-065)` commit on the branch. Then `./pickle changelog check` must not
   list T-065 as unnamed.

Nothing under `skill/` changes (decision 15), so `concepts/lifecycle.adoc` and
`concepts/tickets.adoc` need no edit — this command records nothing and gates nothing.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated per the step above; `./pickle changelog check` clean for T-065.
3. Write a **summary** of everything done (files touched, decisions made, anything deferred —
   in particular, state plainly that `internal/serve` was left on its own view types per
   decision 2, so a reviewer reads it as intent rather than omission).
4. Suggest a Conventional Commit message, e.g.:

   ```
   feat(cli): add board state --json, a versioned read projection (T-065)

   <body — what and why>
   ```

5. **Tidy up before presenting** — `pickle` is a root-path child (`path = "."`): interactive-rebase
   the WIP commits into a small number of atomic, correctly typed/scoped commits
   (`feat(state)`, `feat(cli)`, `docs(manual)`, `docs(changelog)`), then default to keeping that
   history rather than squashing.
6. Commit locally on the ticket branch. Do **not** push or open a merge request without user
   approval. Present the commit messages; after approval, verify the remote base is not behind
   (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
   print nothing), then push and open the merge request. Merging is the human's. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-04 — created (TO DO). source: chat — the Pi-as-best-tier exploration recorded in
  tickets/NOTES.md (2026-08-04); scope corrected before filing after reading the code (the two
  commands it was to flag did not exist, and the paired `schema_version` guard was found to
  have no reachable hazard, so it was pre-registered here instead of filed)
- 2026-08-06 — patched by T-043's review impact sweep: T-043 landed, so the cli-test harness this
  ticket's acceptance test would have needed already exists — the note now says what to reuse
  instead of what to expect
- 2026-08-07 — scope question added (the `## Review` findings table) and T-085 recorded as a
  second prospective consumer; grade deliberately unchanged per the 2026-08-04 precedent against
  crediting prospective demand
- 2026-08-07 — patched by T-052's review impact sweep: T-052 landed, resolving the vocabulary
  question this ticket's T-052 soft-coupling note had left open (`board.Drift` —
  `DriftNone`/`DriftLayout`/`DriftRows` — replaces the old single "stale or hand-edited"
  conflation); the note now says what to reuse instead of what to agree on
- 2026-08-13 — patched by T-085's review impact sweep (step 8): T-085 shipped the `class` column
  and, more importantly for this ticket, a **single canonical table skeleton** — so the open
  scope question's claim that "`TEMPLATE.md` already fixes" the columns is now false in both
  halves (TEMPLATE.md points rather than states, and the list is seven columns led by `class`).
  The assumption is *strengthened*, not invalidated: the middle option this ticket already
  costed — project the closed-vocabulary columns only — now has a fixed shape to parse against
  instead of 13 header variants. Wording corrected; nothing re-graded
- 2026-08-15 — refined against `00ff6b9`. Description re-verified: five `serve/view.go` anchors,
  two `move.go`/`ticket.go` anchors and the command surface were all stale and are corrected; the
  "no write path re-renders frontmatter" premise re-confirmed, so the `schema_version`
  pre-registration stands. The 2026-08-13 correction was itself **overturned by measurement**:
  T-085 did not backfill, so the corpus still holds **twelve** findings-table headers, of which
  the canonical one covers 8 of 61 done tickets — recorded as a table, and it is what forces a
  name-keyed parser. Figures refreshed (347/53/40 → ~500/61/45). Three reserved decisions taken
  with the user: (1) the ticket stands, (2) the findings table is projected via the middle option
  only, (3) `ticket show --json` is dropped — one verb, `board state --json`
- 2026-08-15 — re-graded `impact: low-medium` → `low` (range collapsed at refinement, rules §3):
  no consumer exists, and the 2026-08-04 precedent refuses to credit prospective demand;
  `complexity: medium` and `cost: M` unchanged. Reason recorded in the Description, not only here
- 2026-08-15 — TO DO → READY: plan complete
