---
id: T-126
title: "pickle board metrics: backlog dwell, lead time and open-ticket age from dated History lines"
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low-medium
cost: S
---

# T-126 — pickle board metrics: backlog dwell, lead time and open-ticket age from dated History lines

## Outcome

Running `pickle board metrics` answers "how long do tickets sit in the backlog before they are
picked up, how long from filing to merge, and what is sitting unrefined right now" from each
ticket's `## History`, with `--json` for machine consumers. Today the same answer needs a
hand-written date-differencing script that has to re-solve History parsing every time.

## Description

Every ticket's `## History` is a dated, append-only log of typed transitions, and
`internal/state` (T-065) already parses it into `HistoryEntry{Date, Kind, Target, Text}` — but
nothing computes the *interval between two entries*. `pickle board audit` checks that History
exists and that the last line matches the directory; that is all any shipped code does with the
dates. This ticket adds a read-only report over those intervals.

The idea came from Anthropic's [*The AI-Native SDLC playbook*](https://claude.com/blog/the-ai-native-sdlc-playbook),
which pairs every play with a leading and a lagging indicator, almost always "elapsed time between
two committed-artifact timestamps". Ticket `## History` is that substrate — with one limit that
turns out to decide the whole scope of this ticket.

### The measurement that set the scope: History records dates, not times

Measured at refinement (2026-08-28) over this repository's own tree — **a measurement of one
corpus, recorded as the evidence this ticket was scoped on, not as a maintained fact; re-measure
with the command itself once it exists.** All 453 adjacent-event intervals across 126 tickets:

| interval | n | median | max | zero-day |
| --- | --- | --- | --- | --- |
| READY→IN DEV ⇒ IN DEV→IN REVIEW ("build time") | 73 | 0 | 1 | 72 |
| TO DO→READY ⇒ READY→IN DEV ("queue time") | 62 | 0 | 1 | 57 |
| IN DEV→IN REVIEW ⇒ IN REVIEW→DONE ("review time") | 49 | 0 | 1 | 47 |
| IN REVIEW→DONE ⇒ merged | 62 | 0 | 1 | 60 |
| rework round-trip | 40 | 0 | 1 | 39 |
| **created ⇒ first departure from TO DO** | 119 | 1 | 30 | 50 |
| **created ⇒ merged** | 91 | 1 | 32 | 29 |

**90% of all intervals are zero days**, and every in-flight interval has median 0 and max 1. A
`## History` line carries a date and no time, so any metric bounded by two transitions inside one
working session is structurally a column of zeros: the resolution is coarser than the phenomenon.
The two intervals that do carry spread are both backlog-side, and so is the third useful number —
the age of what is *still* in the backlog (seven open tickets here, the oldest five at 21 days as
of 2026-08-28).

Consequently this ticket ships **three metrics and no time-in-status table**. The
per-status/per-transition aggregation the original filing proposed is cut deliberately, not
deferred: at day resolution it reports zeros, and the only way to make it report anything is to
add clock times to `## History`, which is a different ticket with a different (and much larger)
argument to win.

### The three metrics

1. **Backlog dwell** — `created` → the ticket's **first** departure from the flow's initial state.
   How long an idea waited before anyone refined it. A ticket dropped straight out of the backlog
   counts: it still waited.
2. **Lead time** — `created` → the newest merge line. Filing to shipped. Only merged tickets have
   one; a ticket abandoned in a terminal state that is not the flow's "done" state is excluded,
   because nothing will ever merge it.
3. **Open age** — for every ticket not yet in a terminal state, `created` → the report's as-of
   date, per ticket and aggregated per state. This is the "what is rotting" number, and the only
   one that needs a clock.

Aggregated per child-project and per metric as **n / min / p50 / p90 / max** — no mean. With a
zero-heavy, day-resolution distribution a mean reads as precision the dates do not have.

### Relationship to neighbouring tickets

- **T-065** (done) parses History into `HistoryEntry`; this ticket consumes `ticket.HistoryEntries`
  directly rather than re-parsing, and adds no field to `board state --json`.
- **T-105** (done) is the shape to copy: a leaf read-model package plus a `board` subcommand with a
  human table and a `--json` envelope carrying its own `schema`.
- **T-085** (done) made the per-ticket *findings* record aggregable. This is the *timeline*, a
  different surface over the same record; neither supersedes the other.

### It overrides a standing rejection, deliberately

`NOTES.md` § *"Rejected outright, so they are not re-proposed"* lists "a metrics command, a retro
command, or a dashboard", on the reasoning "build T-065 and let the queries be ad-hoc". This ticket
is filed against that entry by human direction, exactly as T-105 was (see `NOTES.md` § *"Superseded
next day (2026-08-16) — T-105 filed by human direction"*), and the override is recorded in
`NOTES.md` § *"T-126 refinement (2026-08-28) — a rejected-outright entry overridden, and two thirds
of the ticket cut by its own measurement"* so the archive does not silently disagree with the
board. The rejection's own reasoning is what cut this ticket in half: the part of it that was
genuinely ad-hoc-query-shaped (time in each status) is gone, and what remains is the part a
one-off script keeps getting wrong — parsing History, and knowing which endpoints legitimately
do not exist.

### Grading rationale

`impact: low` deliberately, matching T-065's and T-105's own re-grades: no consumer exists, the
answer is available today with a script, and the 2026-08-04 precedent refuses to credit prospective
demand. `complexity: low-medium` and `cost: S`: one new leaf package over readers that already
exist (`ticket.LoadAll`, `ticket.HistoryEntries`, `flow.Definition`), one CLI surface, docs, tests
— no new I/O, no new wire-format compatibility question, no change to any existing command.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-126-board-metrics
```

Local WIP commits are encouraged. Because this is a root-path child, tidy them into atomic commits
before presenting (rules §0) and default to keeping that history rather than squashing. Do **not**
push or open a merge request without explicit user approval. Ticket and board bookkeeping is
committed on `main`, never on this branch — `pickle hooks` enforces it.

### 1. Prerequisite gate (hard)

None. `depends-on:` is empty and every reader this plan builds on is shipped and merged:
`ticket.LoadAll`, `ticket.HistoryEntries`, `ticket.SplitID`, `flow.Definition` (`Initial`,
`DependencySatisfied`, `States`, `ByDir`), `config.Config`, `lock.WithShared`.

### 2. Confirmed design decisions (do not deviate without asking)

1. **The command is `pickle board metrics`, read-only, with its own envelope.** `board` is this
   CLI's whole-ticket-tree read namespace (`audit`, `sync`, `state`, `decisions`). Nothing is
   written under `tickets/`, and `board state --json`'s `schema` is untouched — folding these
   numbers into that document stays a cheap later decision this command does not force (T-105
   decision 13, same reasoning).
2. **Three metrics only: backlog dwell, lead time, open age.** No time-in-status table, no
   per-transition aggregation. The Description records the measurement that cut them; a reviewer
   asking "where is review turnaround?" should find this decision, not a gap.
3. **All intervals are whole days, and the output says so.** `## History` carries dates, not
   times, so two events on one date are 0 days apart. The text output ends with a one-line
   resolution note; the JSON carries `"resolution": "days"`. Never render a fractional day.
4. **Endpoints come from `ticket.HistoryEntries`, never from a new scan.** Start = the first entry
   whose `Kind` is `created`. Merge = the **newest** entry whose `Kind` is `merged` (this also
   picks up the legacy `MERGED: …` form, which `historyKind` already classifies). Departure from
   the backlog = the **first** entry whose `Kind` is `transition` and whose `From` state is the
   flow's initial state.
5. **No state name, directory or ticket-id pattern is hardcoded.** The initial state is
   `def.Initial()`; "done" is `def.DependencySatisfied()` (the state a dependency must reach);
   "terminal" is `flow.State.Terminal`. Ids come from the loaded `ticket.Ticket` — no `T-\d+` regex
   appears anywhere in the new code (T-105 decision 3; a foreign workspace uses other prefixes).
6. **Backlog dwell uses the first departure from the initial state, and re-entry is not modelled.**
   A ticket sent back to the backlog and picked up again contributes its first wait only. Summing
   repeat visits is the time-in-status machinery decision 2 cut; do not reintroduce it here.
7. **Lead time covers merged tickets only; a non-done terminal ticket is excluded, not zero.** A
   ticket in a terminal state that is not `def.DependencySatisfied()` (brine: `7-dropped/`) will
   never merge, so it contributes no lead time. A *done but unmerged* ticket is not excluded — it
   is still open for lead-time purposes, and appears in the open-lead-time count, because "done ≠
   merged" is a real state of this flow (rules §3).
8. **Open age needs a clock, so the clock is a flag.** `--as-of YYYY-MM-DD` defaults to today's
   local date. With `--as-of` given, the whole report is a pure function of the tree — which is
   what makes it testable and what keeps a fixed-date run byte-identical, the property
   `internal/state`'s decision 3 established for `board state --json`.
9. **A ticket that cannot supply an endpoint is reported, never imputed.** Three data-quality
   cases are counted and named in the output: no `created` entry; an end date earlier than its
   start date, checked per computed interval — created vs. first departure, created vs. newest
   merge — not by scanning every dated line in the file for monotonicity (a note can legally
   carry an earlier or out-of-sequence date than the transition above it, and is not this issue);
   a merge line with no parseable date. Each is excluded from its aggregate and listed with the
   ticket id. No silent drops, and no clamping a negative interval to zero.
   *Amended during implementation: this decision originally cited "two tickets in this repo have
   one" as measured evidence for the out-of-order case. That count came from a whole-file
   monotonicity scan across every dated line — a cruder check than what this decision actually
   specifies. Once implemented, the real check — which only ever looks at the two endpoints each
   metric is defined over, and inherits `ticket.HistoryEntries`'s guard against reading an arrow
   out of a folded continuation line (the exact T-043 defect class) — finds zero such tickets in
   this tree today; the code path itself is proven by `internal/metrics/metrics_test.go`'s own
   out-of-order fixture instead. The cited count was a pre-existing measurement error in the filed
   ticket, not a defect in the shipped parser — see the History line recording this.*
10. **Aggregates are n / min / p50 / p90 / max, never a mean.** Percentiles use the
    nearest-rank method on the sorted sample (`p50` = element at `ceil(0.5·n)`, 1-indexed) — one
    documented convention, stated in the package doc and the manual, so no reader has to guess
    which interpolation was used.
11. **Two grouping levels: per child-project, and one `all` row when more than one child is
    registered.** Never a per-child row omitted because it is empty — a registered child with no
    tickets prints `n=0`, so an empty child is visible rather than absent.
12. **Nothing is truncated.** Every open ticket is listed, sorted by age descending then id —
    no top-N cap. A backlog large enough for that to be unwieldy is itself the finding.
13. **Exit semantics are contract.** Unregistered `--project`, and an `--as-of` that is not a valid
    `YYYY-MM-DD` date, are errors (exit 1). An empty result — no tickets, no merged tickets, no
    open tickets — is exit 0 with zero rows and an explicit `0` in the summary line (T-105
    decision 11's distinction).
14. **The read runs behind the shared tree lock** (`lock.WithShared`), like every other tree
    reader.

### 3. Tasks

#### Task 1 — new package `internal/metrics`

New files `internal/metrics/metrics.go` and `internal/metrics/metrics_test.go`. A leaf, pure
text-in/values-out package in the shape of `internal/decisions` and `internal/audit`: no printing,
no exit codes, no locking of its own.

- `type Interval struct { TicketID, Prefix string; Num int; Project, Status, Dir string; Metric string; Days int; Start, End string; Open bool }` — `Metric` is the closed vocabulary
  `"backlog_dwell" | "lead_time" | "open_age"`; `Open` is true only for `open_age` and for an
  unmerged `lead_time` row.
- `type Issue struct { TicketID, Kind, Detail string }` — decision 9's three cases, `Kind` from the
  closed set `"no_created" | "out_of_order" | "unparseable_date"`.
- `type Aggregate struct { Metric, Project string; N, Min, P50, P90, Max int }`.
- `func TicketIntervals(def *flow.Definition, t *ticket.Ticket, status flow.State, asOf time.Time) ([]Interval, []Issue)` — the whole per-ticket rule set (decisions 4–9), fixture-testable with a
  literal ticket text.
- `type Options struct { Project string; AsOf time.Time }` and
  `func Compute(def *flow.Definition, root string, cfg *config.Config, o Options) (Result, error)`
  — `ticket.LoadAll`, per-ticket intervals, aggregation per decision 10/11, deterministic ordering
  (prefix ascending, then number ascending, then the metric vocabulary's own order).

Fixture-test `TicketIntervals` with literal texts covering: a full merged lifecycle; a ticket still
in the initial state (open age only, no dwell); a ticket dropped out of the backlog (dwell, no lead
time); a done-but-unmerged ticket (open lead time); the legacy `MERGED: …` form; a History with no
`created` line; an out-of-order pair; a ticket that returned to the backlog and left twice
(asserting the *first* departure wins); and a same-day lifecycle (asserting `0`, not absent).
Table-test the percentile helper directly, including n=1 and n=2.

#### Task 2 — CLI surface

In `internal/cli/board.go`, add `case "metrics": return runBoardMetrics(args[1:])` to `runBoard`,
extend the usage line to `pickle board <audit|sync|state|decisions|metrics>`, and implement
`runBoardMetrics` following `runBoardDecisions`'s shape exactly: `flag.NewFlagSet`, `loadConfig`,
`flow.ForName(cfg.FlowName())`, `metrics.Compute` inside `lock.WithShared(cfg.Root(), …)`, render.

```
pickle board metrics [--project <child>] [--as-of YYYY-MM-DD] [--json]
```

Text rendering uses `text/tabwriter`, as `renderDecisionsText` does, in two tables:

- `METRIC | PROJECT | N | MIN | P50 | P90 | MAX`
- `TICKET | PROJECT | STATUS | AGE` — every open ticket, age descending (decision 12)

then the data-quality lines (one per `Issue`, or nothing when there are none), then a summary line
`board metrics: N ticket(s), M open, as of YYYY-MM-DD; intervals are whole days`.

#### Task 3 — `--json`

`metrics.Document{Schema: 1, PickleVersion, AsOf, Resolution: "days", Filters, Aggregates,
Intervals, Issues}` with `CurrentSchema = 1` and the same compatible/incompatible-change contract
`internal/state`'s `Document` documents. Encode with `json.NewEncoder` + `SetIndent("", "  ")` +
`SetEscapeHTML(false)`, for the reason `runBoardState` records. Every slice marshals as `[]`, never
`null`.

#### Task 4 — CLI tests

In `internal/cli`, add `board_metrics_test.go` using the existing harness (`newProject`, `capture`,
`t.Chdir`): a sandbox whose ticket fixtures have hand-written History lines with fixed dates, run
with a fixed `--as-of`, asserting exit codes *and* rendered numbers; the `--json` document parsing
with `schema == 1`; a run of the same fixed-date invocation twice, asserting byte-identical output
(decision 8); the two error cases from decision 13; and the empty-tree case at exit 0. Extend
`TestRunExitCodes` with `{"board metrics bad flag", []string{"board", "metrics", "--bogus"},
exitUsage}`.

### 4. Acceptance test

Run verbatim from the repository root; all must be green.

```
just build
just test
just lint
just docs-check
```

Then exercise the command against this repository's own tree. `--as-of` is pinned so the run is
reproducible; the *counts* below were measured at refinement and will drift as the board moves —
re-measure rather than treating them as fixed.

```
# 1. the default report: two tables, resolution note, non-zero ticket count
./pickle board metrics --as-of 2026-08-28

# 2. backlog dwell and lead time have spread; the p50/p90 columns are not all 0
./pickle board metrics --as-of 2026-08-28 --json \
  | jq -e '[.aggregates[] | select(.metric=="lead_time" and .project=="pickle")][0] | .n > 50 and .max > 20'

# 3. open age names the oldest backlog tickets, age descending, none truncated
./pickle board metrics --as-of 2026-08-28 --json \
  | jq -r '[.intervals[] | select(.metric=="open_age")] | length'      # equals the non-terminal ticket count

# 4. data-quality issues, if any, are reported with the ticket id and never silently dropped
# (this tree currently has none — internal/metrics/metrics_test.go's own fixture is what proves
# the out_of_order/no_created/unparseable_date code paths, per decision 9's amendment above)
./pickle board metrics --as-of 2026-08-28 --json | jq -r '.issues[] | "\(.kind) \(.ticket_id)"'

# 5. determinism: a fixed as-of run is byte-identical
diff <(./pickle board metrics --as-of 2026-08-28 --json) <(./pickle board metrics --as-of 2026-08-28 --json)

# 6. filters and empty results are exit 0
./pickle board metrics --project pickle --as-of 2026-08-28 >/dev/null; echo "exit=$?"

# 7. errors, each exit 1
./pickle board metrics --project no-such-child; echo "exit=$?"
./pickle board metrics --as-of 2026-13-99;      echo "exit=$?"

# 8. the JSON form is well-formed and versioned
./pickle board metrics --json | jq -e '.schema == 1 and .resolution == "days" and (.intervals|type)=="array"'
```

Expected: case 1 prints both tables and the resolution line; cases 2–4 pass their `jq` predicates
(case 4 prints nothing today — zero data-quality issues in this tree — which is itself the pass);
case 5 prints nothing; case 6 exits `0`; case 7 prints an error to stderr and exits `1` twice;
case 8 exits `0`.

Cross-check against the refinement measurement, which used an independent script over the same
tree: `lead_time` n=91 / p50=1 / p90=12 / max=32, `backlog_dwell` n=119 / p50=1 / p90=12 / max=30,
seven open tickets with a maximum age of 21 days at `--as-of 2026-08-28`. A material disagreement is a parser bug in
the new code, not a corpus fact — investigate before shipping (T-105's own cross-check step, and
the reconciliation its review had to write, is the precedent for doing this before review rather
than during it).

### 5. Docs update (mandatory when user-facing)

1. `docs/user-manual/cli-reference.adoc`:
   - add a row to the `== Overview` table:
     `| pickle board metrics [--project …] [--as-of …] [--json]`;
   - add a `[#cmd-board-metrics]` / `== pickle board metrics` section after
     `== pickle board decisions`, documenting **every** flag (T-066 exists because flags went
     undocumented), the exit-code contract, the three metrics and their exact endpoints, the
     nearest-rank percentile convention, the day resolution and what it means for same-day
     transitions, and the fact that there is deliberately no time-in-status table;
   - **add this command to the shared-read-lock enumeration in the intro** (currently naming
     `pickle serve`, `board state --json` and `board decisions`) — T-105's review finding F2 was
     exactly this enumeration going stale when a command was added.
2. `internal/cli/cli.go` — add the command to the `help` text, in `board`'s block.
3. `CHANGELOG.md` — an `### Added` entry under `## [Unreleased]`, ending `(T-126)`, in the
   user-observable voice the existing entries use, and saying plainly that in-flight timings are
   not reported and why.
4. Run `just docs-check`.

### 6. Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated per step 5, including the `CHANGELOG.md` entry.
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggest a Conventional Commit message, e.g.:

   ```
   feat(cli): add board metrics for backlog dwell, lead time and open age (T-126)

   <body — what and why>
   ```

5. **Tidy up before presenting** — `pickle` is a root-path child, so interactive-rebase the WIP
   commits into a small number of atomic, correctly typed/scoped commits (a plausible split:
   `feat(metrics): …`, `feat(cli): …`, `docs(manual): …`).
6. Commit locally on the ticket branch. Do **not** push or open a merge request without user
   approval. Present the commit message; after approval, keep the tidied history (the root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the merge request. Merging is the human's.

## Review

Reviewed 2026-08-28 against `feat/T-126-board-metrics` (3 commits, `03b78fc..1d39267`, plus one
`fixed inline` docs commit made during this review, `4495250`), following
`resources/review-protocol.md`. No overarching or per-child review addendum is configured in
`pickle.toml`, so the generic protocol is the whole standard.

- [x] Reviewer independence settled (step 0) — **delegated**. The reviewing agent authored this
      branch in the same session, so the audits (steps 2–4a) were delegated to three independent
      sub-agents, spawned fresh and briefed adversarially: an implementation audit, a quality audit
      (with mutation testing), and a consistency + documentation audit. Severity, class,
      disposition and the moves stayed with the orchestrating reviewer. Every delegated finding was
      re-verified by hand before entering the table below — delegation buys independence, not
      accuracy; two delegated claims were **downgraded** on that re-verification (see *Delegated
      claims not accepted as written*).
- [x] Implementation audit — acceptance test re-run verbatim, tasks & criteria verified (step 2)
- [x] Quality audit (step 3) — including a mutation-testing pass in a scratch copy
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — run via the session's
      `docs_readability` reviewer over `docs/user-manual/cli-reference.adoc` and `CHANGELOG.md`;
      9 suggestions returned, **all 9 verified verbatim against the files, 0 discarded as
      fabricated**. 8 of the 9 target prose this branch never touched (the lock intro's
      `install` sentence, `cmd-install`'s `--agent` block, `cmd-ticket-new`, `cmd-board-state`'s
      review-field caveat, `cmd-board-decisions`' flag table, and three older `CHANGELOG.md`
      entries) and are therefore out of this ticket's scope. The 1 in-scope suggestion (splitting
      this branch's own `CHANGELOG.md` sentence in three) is held for the user's approval and is
      not applied here — readability polish, never a finding.
- [x] Findings recorded with severity, class and disposition; disposition summary + cost line below
      (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] Other references updated; governing documents checked (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — see below
- [ ] Summary + commit messages presented for approval (step 9) — **not reached**: two blocking
      findings send this to rework before anything is published

### Verification

| what | result |
|---|---|
| `just build` · `just test` · `just lint` · `just docs-check` | all clean (`go vet`, `gofmt -l` empty, snowball check green) |
| Acceptance cases 1–8, re-run verbatim | all 8 behave as the plan's *Expected* paragraph states |
| Refinement cross-check (the plan's ±0 step) | independently reproduced: `backlog_dwell` n=120/p50=1/p90=11/max=30, `lead_time` n=91/p50=1/p90=12/max=32. The plan's recorded `n=119`/`p90=12` is **corpus drift**, not a parser bug — T-126 itself departed TO DO on 2026-08-28, after the refinement measurement, adding one low dwell sample. Verified by an independent reimplementation over the same tree |
| Confirmed decisions 1, 2, 5, 6, 7, 9, 10, 11, 13, 14 | honoured. Decision 5 verified mechanically: no `regexp` import and no `T-\d+` pattern anywhere in the new code. Decision 10 verified exhaustively: `ceil(q·n)` matches exact integer arithmetic for all n ≤ 100000 |
| Confirmed decision 3 ("never render a fractional day") | **violated in substance on the default path** — see F1 |
| Confirmed decision 8 ("`--as-of` defaults to today's local *date*") | **not honoured** — the code uses a local *instant*; see F1 |
| Confirmed decision 4 (endpoint via the initial state) | approximated; `def.Initial()` is never called — see F8 |
| `pickle board audit` | 126 tickets, 0 errors, 0 warnings |

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | The default run (no `--as-of`) computes ages against a local wall-clock **instant** while every `created` parses to **UTC midnight**, so `Sub/24h` truncation straddles the zone boundary: the report's own printed `as_of` and the ages it prints can disagree by a day, and the same tree yields different numbers at different hours. Decision 8 says "today's local **date**"; the code takes `time.Now()`. Structurally untested — every test pins `--as-of`, which lands on UTC midnight, so the only uncovered path is the one a user gets by typing the command with no flags. | `internal/cli/board.go:254` `asOf := time.Now()` vs `internal/metrics/metrics.go:151` `time.Parse`. Reproduced live: `TZ=Pacific/Midway ./pickle board metrics --json` prints `as_of=2026-08-27` with `T-075 open_age=21`, while the authoritative `--as-of 2026-08-27` gives `20` | Truncate to a date before use: `n := time.Now(); asOf = time.Date(n.Year(), n.Month(), n.Day(), 0,0,0,0, time.UTC)`. Add a test that exercises the **default** path under at least one non-UTC `TZ` — the gap that hid this |
| F2 | blocking | correctness | — | The open table lists **every open ticket twice with the same number**. An open `lead_time` row is emitted for *every* non-terminal unmerged ticket, where it equals `open_age` by construction, so the command's flagship human surface — the "what is rotting" table the ticket's own `## Outcome` promises — is doubled with zero added information. Both the manual and the code comment assert this row means the rare done-but-unmerged case, which is false. | `internal/metrics/metrics.go:213-232`: `willNeverMerge` is false for every non-terminal state, so `mergedDate == ""` → `addOpen(MetricLeadTime, …)`, then `addOpen(MetricOpenAge, …)` with identical `Days`. Measured: **7 of 7** open tickets duplicated. False prose at `docs/user-manual/cli-reference.adoc:1270-1272`, `:1314` and `internal/cli/board.go:296-297` | Emit the open `lead_time` row only for a ticket that has **reached the flow's done state but carries no merge line** — decision 7's literal case. For any other non-terminal ticket its open lead time *is* its open age, already reported. Then correct the manual and the code comment to match, including the "one line per ticket" claim at `:1324` (issues are emitted per **interval**: 14 lines for 7 tickets under `--as-of 2026-07-01`) |
| F3 | non-blocking | test-gap | noted | Mutation testing found **12 surviving mutations** in a brand-new package: `mergedDate` first-wins vs last-wins; the `all` row always/never emitted; the `metricOrder` tie-break; `NonNil` gutted to `return s`; `summarize`'s empty row `N = -1`; the open-table id tie-break reversed; three separate corruptions of the summary line's counts; the aggregate table's columns reversed; and the dwell `break` removed. Additionally the `"empty result marshals every slice as [] not null"` subtest runs `--project demo` where `demo` has five fixture tickets, so it asserts nothing — `NonNil` has no effective coverage — and `unparseable_date` is implemented and documented but asserted by no test. Root cause of several: the unit tests' `findMetric` helper returns the *first* match, so no test ever asserts the interval **count** per metric. | mutation run in a scratch copy of the tree; `internal/cli/board_metrics_test.go` subtest `empty result marshals every slice as [] not null`; no occurrence of `IssueUnparseableDate` in any `_test.go` | Left `noted` deliberately: the rework for F1/F2 must land regression tests over exactly these paths (default-clock, one-row-per-open-ticket), and the scoped re-review re-runs the mutation set against the fix diff. Whatever still survives then is citable from this row and can be promoted rather than re-derived |
| F4 | non-blocking | design | noted | A ticket whose `project:` names an **unregistered** child is counted into the hidden `all` sample but gets no per-project row; in a single-child workspace there is no `all` row at all, so it appears in `intervals[]` and in the text table while contributing to **no aggregate** — the exact opposite of decision 11's "visible rather than absent". Requires a tree `board audit` already errors on, which is why it is not blocking. | `internal/metrics/metrics.go:352` appends to `samples[m]["all"]` unconditionally; the row loop at `:344-349` iterates `cfg.Projects` only. Measured with a `ghost` project: `all` n=2 max=100 while its parts are n=1 max=2 and n=0 | Either drop unregistered projects from `all` too, or give them their own row |
| F5 | non-blocking | design | noted | The open table's tie-break compares ticket ids as **strings**, so `T-100` sorts before `T-9`, disagreeing with `Compute`'s own ordering, which splits the id and compares `Num` numerically for exactly this reason. The `issues` sort has the same defect. Invisible in this tree only because every open id happens to be the same width. | `internal/cli/board.go` open-table comparator; `internal/metrics/metrics.go:300-305` issues comparator, vs `:290-299` | Reuse `ticket.SplitID`'s prefix/number ordering on both surfaces |
| F6 | non-blocking | docs-gap | fixed inline | Five claims in the new manual section were false or narrower than the code: the `all` row is also suppressed by `--project` (not only by a single-child registry); `filters.as_of` records the *effective* date, not the flag as given; `lead_time` also reads the legacy `MERGED: …` form; exit `2` was undocumented. | `docs/user-manual/cli-reference.adoc` — `all`-row sentence, `filters` sentence, `lead_time` endpoint row, `Exit codes` section | All four corrected in `4495250`; `just docs-check` green |
| F7 | non-blocking | docs-gap | fixed inline | Two **foreign-workspace-test** violations shipped in user-facing docs (`AGENTS.md` defines the test): a `(T-105 decision 13's pattern…)` citation no reader outside this repo can resolve — shape (a) — and "than this one measured", a claim about a corpus the reader does not have — shape (b). Note `payload_lint_test.go` scopes its mechanical check to `skill/` and `agents/` only, so **nothing would ever have caught these in `docs/`**. | `docs/user-manual/cli-reference.adoc`, JSON-envelope paragraph and the "Why three metrics" paragraph | Both rewritten to stand on their own in `4495250`. Whether the payload lint should extend to `docs/` is left as a question this row records, not a change made here |
| F8 | non-blocking | design | noted | Decision 4 defines the dwell endpoint as "the first transition whose **From** state is the flow's initial state"; the code takes "the first transition of any kind" and `def.Initial()` is never called. Equivalent for any tree built by `ticket new` (a ticket always lands in the initial state), and the reasoning is written into the package doc — but it is a deviation from a confirmed decision, which the plan said not to make without asking. | `internal/metrics/metrics.go:199-211`; `grep Initial() internal/metrics/metrics.go` matches comments only | Either derive the running status (mirroring `ticket.LastHistoryStatus`) and check it against `def.Initial()`, or amend decision 4 to state the equivalence it actually relies on |
| F9 | non-blocking | design | noted | Three smaller taxonomy/robustness gaps, all in the same function: a calendar-invalid **first** `created` date reports `no_created` rather than `unparseable_date`, which exists for exactly that; a **second** `created` line is silently ignored with no issue raised; and the `mergedDate` loop's comment says "newest merge line" while the code takes the **last line in file order** (identical for an append-only history, divergent if dates are ever out of order). | `internal/metrics/metrics.go:147-155` (the unconditional `break`), `:198-202` | Batched here rather than split: one function, one fix pass, none of them reachable in a well-formed tree |
| F10 | non-blocking | stale-xref | noted | `project-structure.adoc` says "`pickle board audit` and `pickle board state --json` read `tickets/` the same way … all three describe the board as it stood last week" — now four commands, five counting `board decisions`. **Pre-existing**: T-105 already omitted `board decisions`, so this branch widened a gap it did not open, and the inline bar is causation, not size. | `docs/user-manual/concepts/project-structure.adoc:203-207` | Left for whoever next touches that page, or a docs-sweep ticket; recorded here with its evidence so it is citable |
| F11 | non-blocking | other | fixed inline | Commit `b53f8af`'s subject carries no `(T-126)` suffix, which this project's commit convention requires for child-project code (the id is in the body as `Refs T-126.` instead). | `git log --format=%s 03b78fc..HEAD` | To be corrected when the branch is tidied for the rework round — the tidy is now happening regardless, so no separate history rewrite is incurred |

Disposition summary: 2 blocking (F1, F2) → `5-rework/`; 9 non-blocking — 3 fixed inline (F6, F7, F11), 6 noted (F3, F4, F5, F8, F9, F10), 0 folded, 0 new tickets. No follow-up ticket was minted: F3 and F9 are the only candidates that could pass the promotion test, and both sit squarely in the code paths the F1/F2 rework must touch, so promoting them now would file work about to be done anyway — the scoped re-review is the gate that decides whether anything survives.

```
cost: estimated S, actual M — provisional, pending the rework round; the implementation itself landed at S, but two blocking defects (an untested default clock path, and a duplicated flagship table) add a fix-and-re-review cycle the S estimate did not carry
```

### Rework fix record — round 1 (commit 9e77eea)

Scope was the two blocking findings and nothing else. One commit on the same branch; the tip before
it was `4495250`, so the scoped re-review reads `git show 9e77eea` (equivalently
`git diff 4495250..9e77eea`).

**F1 — fixed.** `internal/metrics/metrics.go` gains `DateOf(t time.Time) time.Time`, which takes an
instant's calendar date *in its own location* and re-expresses it as midnight UTC — so "today" means
the day the user believes it is, while landing on the same midnight-UTC grid every parsed `## History`
endpoint sits on. `internal/cli/board.go` no longer passes `time.Now()` anywhere near the metric.

The flag-or-clock resolution also moved out of `runBoardMetrics` into a new pure
`resolveMetricsAsOf(flagValue string, now time.Time)`. That is the substantive half of the fix: as
three lines inline, the default branch depended on the process clock *and* on `time.Local`, which Go
caches at first use, so no test could reach it — which is exactly why the defect shipped. As a pure
function of `(flag, instant)` both branches are testable from zones the test machine is not in.

A live sweep confirms self-consistency in every zone tried — the printed `as_of` and the ages beneath
it agree (run at 13:48 UTC):

| `TZ` | printed `as_of` | ages self-consistent with it |
|---|---|---|
| `UTC` | 2026-08-28 | yes |
| `Europe/Warsaw` (+2) | 2026-08-28 | yes |
| `Pacific/Midway` (−11) | 2026-08-28 | yes |
| `Pacific/Kiritimati` (+14) | 2026-08-29 | yes |

Kiritimati reporting the 29th is correct, not a residual defect: it *is* the 29th there, and the ages
match that date.

**What that sweep does and does not prove — stated precisely, because it is weaker than it looks.**
The defect was *hour-dependent*, so a live sweep only catches it during the window when the local
date and the UTC date disagree. At 13:48 UTC, Midway is on 2026-08-28 02:48 — inside the same
calendar day as UTC — so the **pre-fix** code would have looked self-consistent in this sweep too.
The earlier review-time observation (`TZ=Pacific/Midway` printing `as_of=2026-08-27` while computing
ages against the 28th) was taken before 11:00 UTC, when Midway was still on the 27th and the two
dates genuinely diverged. So the sweep above is a sanity check, not the proof; the **deterministic**
proof is `TestResolveMetricsAsOf` and `TestDateOf`, which state the instant and the location
explicitly instead of depending on when the suite happens to run, plus the mutation checks below.

**F2 — fixed.** An open `lead_time` row is now emitted only for a ticket that has reached the flow's
done state without a merge line — decision 7's literal case, and the only one `open_age` does not
cover (a done ticket is terminal, so it has no `open_age`). Every other unmerged ticket gets no
lead-time row, because for a non-terminal ticket the open lead time is `asOf - created`: the same two
endpoints `open_age` already reports. Live: the open table went from 14 rows for 7 tickets to **7 rows
for 7 tickets**, one per ticket. The duplicated data-quality lines F2's suggestion column also named
fell out with it — `--as-of 2026-07-01` now prints 7 issue lines, not 14.

**Regression tests, mutation-checked.** Three mutations were run against the fixed tree in a scratch
copy; each is caught:

| mutation | caught by |
|---|---|
| revert F2 (open `lead_time` for every non-terminal ticket) | `TestTicketIntervals`, `TestTicketIntervalsEmitsOneOpenRowPerTicket`, `TestBoardMetricsOpenTableHasOneRowPerTicket` |
| `resolveMetricsAsOf`'s **body** returns the raw instant | `TestResolveMetricsAsOf` |
| `DateOf` computes from the **UTC** date instead of the local one | `TestDateOf`, `TestResolveMetricsAsOf` |

> **Corrected 2026-08-28 by the scoped re-review (finding R1).** This table originally labelled the
> middle row "revert F1 wiring", and the paragraph that followed claimed: *"an earlier attempt at this
> fix put the F1 regression test only on `DateOf` itself, and reverting the call site still passed the
> entire suite — the same shape of gap as the original defect. The seam exists so that mutation
> fails."* **That was false.** The mutation actually run changed the *body* of `resolveMetricsAsOf`,
> not the wiring; reverting the **call site** — restoring `asOf := time.Now()` inside
> `runBoardMetrics`, leaving the seam defined and green — still passed the whole suite, so round 1
> had moved the untested gap one function outward rather than closing it. The row is relabelled above
> to say what was really tested, and the false paragraph is struck. Closed in round 2 below.

**Docs and CHANGELOG** were corrected to match what now ships: the `lead_time` endpoint row states
that the done-but-unmerged case is the only open `lead_time` row; the text-output section states one
row per open ticket and that issue lines are per *interval*, not per ticket (F2's own suggestion
column flagged that sentence). The one in-scope docs-readability suggestion from step 4b — splitting
this branch's `CHANGELOG.md` sentence — was approved by the user and applied in the same commit.

**Not touched:** F3–F5 and F8–F10 remain `noted` as recorded. Nothing outside the two blocking
findings was changed, and no commit was rewritten — the `(T-126)` subject suffix missing from
`b53f8af` (F11) is still pending the publish-time tidy, where it belongs.

Acceptance test re-run verbatim after the fix: all 8 cases pass, plus the newly-documented exit-`2`
path. `just build` · `just test` · `just lint` · `just docs-check` all clean. `pickle board audit`:
126 tickets, 0 errors, 0 warnings.

### Scoped re-review — round 1's fix (2026-08-28)

Scope per `review-protocol.md` §1: the two blocking findings, **plus the diff that closed them**
(`git show 9e77eea`), the latter read as new work. Not a re-audit of the feature. Independence:
**delegated** again — the reviewing agent wrote the fix in this same session — to a fresh adversarial
sub-agent, with every finding re-verified by hand before recording here.

**F1 and F2 are closed in shipped behaviour.** Verified independently, and more thoroughly than
round 1 managed: a 7-zone sweep including non-hour offsets neither the fix nor its record thought to
try (`Asia/Kathmandu` +5:45, `Pacific/Chatham` +12:45) found the printed `as_of` and every age
beneath it consistent in all 7. `DateOf` is DST-safe by construction — the output location is
hard-wired `time.UTC`, and local midnight is never constructed, so the "midnight does not exist /
happens twice" cases cannot arise. F2 was exercised against a fixture carrying a ticket in **each of
the seven status directories** plus done+merged, dropped+merged, created-in-the-future, and
`--as-of == created`: exactly one open row per open ticket in every case, `0` present rather than
absent for the same-day case, and no `lead_time` at all for a dropped ticket.

**But F1 is not closed as a *finding*, because the coverage its own remediation required still does
not exist** — R1 and R2 below. Nine of ten mutations are killed, including all five this reviewer
named; the tenth is the one that matters.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | blocking | test-gap | — | **Reverting F1 at the call site still passes the whole suite.** Restoring the verbatim pre-fix body of `runBoardMetrics` (`asOf := time.Now()` plus the inline parse), leaving `resolveMetricsAsOf` defined and green, re-ships the entire F1 defect with `just test` clean. `TestResolveMetricsAsOf` tests the *seam*; nothing asserts `runBoardMetrics` actually **uses** it. No test in the tree invokes `board metrics` without `--as-of` and asserts its output — the 11 no-flag invocations are error/usage cases asserting exit codes only. This is the identical shape of gap as the original F1, relocated one function outward. It also **falsifies this ticket's own fix record**, which claims "the seam exists so that mutation fails" and lists a mutation row labelled "revert F1 wiring" that in fact mutates the function *body*. F1's own suggestion column required "a test that exercises the **default** path — the gap that hid this"; that requirement is unmet. | reproduced by hand: pre-fix body restored on top of `179a306` → `ok internal/cli`, `ok internal/metrics` | Add a test that invokes `board metrics` with **no** `--as-of` and asserts the result — e.g. `as_of` equals `metrics.DateOf(time.Now())`, and every `open_age` equals `as_of - created`. Then correct the fix record's claim and relabel the mislabelled mutation row |
| R2 | blocking | test-gap | — | **`TestDateOfKeepsAgesConsistentWithTheDateItReports` cannot detect a wrong date**, and its comment says it can. `want` is derived from `asOf` round-tripped through `Format`/`Parse` — the value under test — so it asserts only that `asOf` sits on the midnight-UTC grid, which `TestDateOf`'s explicit `Location()`/`Clock()` assertions already state directly. Verified: with `DateOf` mutated to `+1 day`, the test **passes in isolation**. The inline comment `// What the printed as-of date itself implies, computed independently.` is false — it is not computed independently. Recorded blocking alongside R1 not because behaviour is wrong, but because F1 cannot be certified closed while the one check that looks like an end-to-end guard cannot fail. | mutated `DateOf` to `d+1`, ran that test alone → `ok internal/metrics` | Derive the expected age from a source independent of `DateOf` (a literal, or the fixture's own dates), or delete the test as subsumed by `TestDateOf` + R1's new default-path test — and fix or remove the comment either way |
| R3 | non-blocking | docs-gap | fixed inline | The rework commit inserted `resolveMetricsAsOf`'s comment block directly against `runBoardMetrics`'s existing one with no blank line, so Go attached the merged block to the new function: `runBoardMetrics` lost its godoc entirely, and `resolveMetricsAsOf`'s godoc opened with "runBoardMetrics implements…". Introduced by `9e77eea`; `gofmt` and `vet` do not catch it. | `go doc -all -u ./internal/cli` before the fix | Each comment moved back to its own function; verified with `go doc`. Fixed in `179a306` |
| R4 | non-blocking | docs-gap | fixed inline | Two claims shipped looser than the code. The manual said any unmerged ticket that has not reached "done" has its open lead time "reported once, there" in `open_age` — but a ticket in a non-"done" **terminal** state (`DROPPED`) is reported by neither, correctly, since nothing is still running. And "one row per open ticket" is absolute in both the manual and `CHANGELOG.md`, while a ticket whose endpoints are unusable contributes an issue line and **zero** rows (verified: `--as-of 2026-07-01` → 7 issue lines, 0 open rows, "0 open"). `metrics.go`'s own comment scoped this correctly; the prose dropped the qualifier. | `docs/user-manual/cli-reference.adoc` `lead_time` row and text-output section; `CHANGELOG.md` | Both qualified ("a **non-terminal** unmerged ticket…", "**at most one** row per open ticket"). Fixed in `179a306` |
| R5 | non-blocking | design | noted | F2's fix rests on "a done ticket is terminal, so `open_age` does not cover it" — an invariant `flow.New` does not enforce. It validates that `Initial` and `Pickup` are non-terminal but places **no** `Terminal` requirement on `DependencySatisfied`, so a user-authored flow (T-081's direction) with a non-terminal "done" state would re-create F2's duplication exactly. Latent only: brine ships as the sole flow, and its `6-done` is terminal. | `internal/flow/flow.go` — `Initial`/`Pickup` terminal checks present, `DependencySatisfied` has none | Either assert the invariant where F2's fix depends on it, or add the validation to `flow.New` when a second flow becomes loadable. Not this ticket's to fix — recorded with evidence so it is citable |

Disposition summary: 2 blocking (R1, R2) → `5-rework/` for round 2; 3 non-blocking — 2 fixed inline (R3, R4) in `179a306`, 1 noted (R5), 0 folded, 0 new tickets.

**Round-1 fix-record claims, audited.** "14 rows → 7 rows", "7 issue lines, not 14", the timezone
table, mutation rows 1 and 3, acceptance cases 1–8 green, and "nothing outside the two blocking
findings changed" were each independently reproduced and hold. The caveat paragraph about what the
live sweep does *not* prove was confirmed accurate. **One claim is false and is the substance of
R1**: "the seam exists so that mutation fails."

**Docs-readability, round 2 (step 4b).** Re-run over both files this round changed: 7 suggestions,
**all 7 verified verbatim, 0 discarded as fabricated**. 5 target prose this branch never touched. The
2 in-scope suggestions (splitting the "Backlog-side durations" sentence in the manual, and
"is structurally a column of zeros" → "produces a column of zeros" in `CHANGELOG.md`) are held for
the user's approval and not applied. Recorded because it nearly went wrong: the reviewer's first
verification pass reported one suggestion as fabricated, which was a **false negative in the
verification script itself** — it stripped a line-leading `*` as a list bullet when the `*` was
emphasis. Re-checked without that rule, the quote is verbatim. The protocol's "ignore layout, compare
words and punctuation" is easy to over-apply in exactly this way, and doing so would have libelled a
clean run.

### Rework fix record — round 2 (commits 179a306..1a1ce1b)

Scope was R1 and R2, the two blocking findings of the scoped re-review. Two commits on the same
branch — `5041b9d` (the R1/R2 fix) and `1a1ce1b` (the two approved docs-readability edits, wording
only) — so the next re-review reads `git diff 179a306..1a1ce1b`, the form the range takes as written.

**R1 — fixed, and the reason the obvious fix does not work is worth recording.** The re-review's
suggestion was "a test that invokes `board metrics` with no `--as-of` and asserts the result". That
alone would **not** have closed this: the F1 defect only changes output when the local date and the
instant's UTC date differ — for local times before the zone's UTC offset (east) or after 24h minus it
(west) — and in a UTC test environment those never differ at any hour. A no-flag end-to-end test
would therefore have passed against the defect on most CI machines and "covered" nothing, which is
the third time this same blind spot has produced a green suite over a real bug.

So the clock itself is now indirected, through an unexported `metricsNow = time.Now` in
`internal/cli/board.go`, and `TestBoardMetricsDefaultPathUsesTheDateNotTheInstant` pins it to
**20:30 in a UTC-11 zone** — already the next day in UTC — before invoking the command with no
`--as-of`. The correct and the defective behaviours give different answers at that instant on any
machine in any zone, so the guard is deterministic rather than environment-dependent. It asserts the
printed `as_of` (`2026-08-27`, the local date, not the instant's UTC date), one hand-computed literal
age (T-002: `2026-07-20` → `2026-08-27` = 38 days; reading the raw instant yields 39), and the
self-consistency invariant across every open row.

**R2 — fixed.** `TestDateOfKeepsAgesConsistentWithTheDateItReports` derived its expected value from
`asOf` round-tripped through `Format`/`Parse` — the value under test — so it could only ever assert
that `asOf` sat on the midnight-UTC grid. Replaced by
`TestDateOfDrivesAgesFromTheLocalCalendarDate`, a table of four zones whose expected as-of date *and*
expected age are **hand-computed literals** (e.g. Midway 2026-08-27 20:30 → as-of `2026-08-27`, age
20 from a `2026-08-07` fixture). The misleading `// … computed independently.` comment went with it.
Proof the tautology is gone: with `DateOf` mutated to add a day, the old test passed in isolation and
the new one fails in isolation on both assertions.

**Mutation results — the round-1 survivor now dies.** Each mutation applied to the tip and the two
packages re-run:

| mutation | round 1 | round 2 |
|---|---|---|
| revert F1 **at the call site** (verbatim pre-fix body) | 🔴 survived | ✅ caught — `TestBoardMetricsDefaultPathUsesTheDateNotTheInstant` |
| `DateOf` identity (`return t`) | ✅ caught | ✅ caught |
| `DateOf` via the **UTC** date | ✅ caught | ✅ caught |
| `DateOf` off by one day | partly — R2's test passed in isolation | ✅ caught, including in isolation |
| revert F2 | ✅ caught | ✅ caught |

**Round 1's fix record has been corrected in place** rather than left standing: the mutation row that
was mislabelled "revert F1 wiring" now says what it actually mutated, and the false claim that "the
seam exists so that mutation fails" is struck with a dated note pointing at R1. The archive should
not keep asserting a coverage property a single command falsifies.

**Not touched:** R5 remains `noted`; R3 and R4 were already fixed inline in `179a306`; F3–F5 and
F8–F10 from the first review are unchanged. No commit was rewritten — F11's missing `(T-126)` subject
on `b53f8af` still awaits the publish-time tidy.

The two in-scope docs-readability suggestions held at the re-review were approved by the user and
applied in `1a1ce1b` — wording only, both verified verbatim against the files before being presented.
`just build` · `just test` · `just lint` · `just docs-check` all clean; acceptance cases re-run green;
`pickle board audit` 126 tickets, 0 errors, 0 warnings.

### Scoped re-review — round 2's fix (2026-08-28) — CONCLUDING

Scope per `review-protocol.md` §1: R1 and R2, **plus the diff that closed them**
(`git diff 179a306..1a1ce1b`), the latter read as new work. Independence: **delegated** a third time,
to a fresh adversarial sub-agent briefed that the two previous rounds had each found real blocking
defects the author's own checks missed. Every claim re-verified by hand before recording.

**Verdict: no blocking findings. R1 and R2 are closed, and the ticket concludes.**

**The arithmetic every other claim rests on was verified independently, twice** — by the delegated
reviewer and by the orchestrating reviewer using separate tooling. A literal that happened to match a
wrong implementation is the one failure mode nothing downstream would catch, so it was checked first:

| check | result |
|---|---|
| R2's four `(zone, wall clock, as-of, days)` rows against a `2026-08-07` fixture | all four correct — and **three of the four are discriminating** (local date ≠ the instant's UTC date), which is what gives the table its kill power |
| R1's `38 days` (T-002 created `2026-07-20` → pinned as-of `2026-08-27`) | correct; the defective path yields 39, so the assertion separates them |
| R1's pinned instant straddles the date boundary | confirmed — 20:30 at UTC−11 is `2026-08-28 07:30` UTC |
| the record's range `179a306..1a1ce1b` | resolves, to exactly `5041b9d` and `1a1ce1b` |

**Mutation testing: six mutations, no survivors** — including the round-1 survivor (reverting F1 at
the call site) and a new one the reviewer added (bypassing `metricsNow` to call `time.Now()`
directly). The `DateOf`-off-by-one mutation now fails **in isolation on all four rows and both
assertions**, where round 1's version passed in isolation. `metricsNow` was attacked specifically as
new mutable global state and came back clean: two readers only, `defer` restore safe under `t.Fatalf`
(`capture` runs `fn` on the test goroutine, so `Goexit` still runs defers), no `t.Parallel` anywhere
in the package, and `-count=2`, `-shuffle=on` and `-race` all green — re-confirmed independently here.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| N1 | non-blocking | docs-gap | fixed inline | The approved docs-readability edit split one long sentence and created a stutter: two consecutive sentences both opening "That is why". It also left an 82-column line in a paragraph that otherwise wraps at 69–80. | `docs/user-manual/cli-reference.adoc`, "Why three metrics" paragraph | Second sentence rewritten as a trailing "— and why this command reports exactly those three…" and the paragraph reflowed. Fixed in `f428608` |
| N2 | non-blocking | stale-xref | fixed inline | A comment called `asOf := metricsNow()` "the verbatim pre-fix body". The verbatim pre-fix body was `asOf := time.Now()`; `metricsNow` did not exist until the fix that comment documents. | `internal/cli/board_metrics_test.go`, `TestBoardMetricsDefaultPathUsesTheDateNotTheInstant` doc | Reworded to "the pre-fix body … or its equivalent under the seam". Fixed in `f428608` |
| N3 | non-blocking | spec-unclear | fixed inline | `metricsNow`'s doc claimed a plain end-to-end test "**provably** cannot guard" the default path. Overstated: on a non-UTC machine such a test catches the defect for part of the day — which is arguably worse than not having it, since it fails by the clock. The following sentence already scoped the claim to a UTC environment, so the absolute reading was never intended. | `internal/cli/board.go`, `metricsNow` doc comment | "cannot *deterministically* guard it", with the partial-catch behaviour spelled out. Fixed in `f428608` |
| N4 | non-blocking | test-gap | fixed inline | **Both guards protecting a blocking finding could silently retire themselves.** Each built its zone with `time.LoadLocation` and `t.Skipf`'d when zoneinfo was missing, so on a scratch or distroless CI image the regression tests for F1 and R1/R2 would skip rather than run — the same class of hole (coverage that looks real and is not) that this ticket has now produced three times. | `internal/cli/board_metrics_test.go` and `internal/metrics/metrics_test.go`, the `LoadLocation`/`Skipf` pairs | Both switched to `time.FixedZone`, which needs no zoneinfo: `DateOf` hard-wires its output location to UTC and never constructs local midnight, so a fixed offset exercises it fully. `TestDateOf` and `TestResolveMetricsAsOf` deliberately keep real named zones as the complementary check that real tzdata behaves identically — those two may still skip, and neither is the primary guard for a finding. Fixed in `f428608`, with the whole mutation set re-run afterwards |
| N5 | non-blocking | docs-gap | fixed inline | Applying the approved readability edit to `CHANGELOG.md` only left the two surfaces describing the same phenomenon differently — "produces a column of zeros" against "is structurally a column of zeros". | `CHANGELOG.md` vs `docs/user-manual/cli-reference.adoc` | Manual aligned to the approved wording. Fixed in `f428608` |

Disposition summary: 0 blocking; 5 non-blocking, all **fixed inline** in `f428608` (N1–N5), 0 folded,
0 noted, 0 new tickets. Every one is prose, a comment, or a test's zone-construction idiom this
branch itself authored; none changes what the command does.

```
cost: estimated S, actual L — three review rounds and two rework rounds against an S estimate. The
feature itself was S, as filed; the overrun is entirely in verification, and specifically in one
defect class: coverage that reads as real and is not. Round 1 shipped a clock bug the whole suite
missed, round 2's fix for it was itself unguarded at the call site, and round 3 found both guards
could skip themselves off a machine without tzdata. Each was cheap to fix and expensive to find.
```

### Why this ticket took three review rounds — the one lesson worth carrying

All five blocking findings across the three rounds (F1, F2, R1, R2, and N4 had it not been caught)
are the same shape: **a check that appears to establish a property and does not.** The suite was
green at every single hand-back. What moved each time was an *independent* reader running a mutation
the author had not thought to run — and twice the author's own recorded mutation table was the thing
that turned out to be wrong, once naming a mutation it had not actually performed.

The transferable rule, and the reason it is recorded here rather than in a new ticket: **when a fix
is justified by "and now a test would catch it", run the mutation before writing that sentence down.**
Delegation is what made this visible — a self-review would have accepted its own coverage claim three
times running, because the claim was made in the same reasoning that produced the code. `NOTES.md`
carries the standing note for the theme; nothing here needs a new ticket, and the promotion test does
not pass for any of it.

### Delegated claims not accepted as written (step 0's "verify before recording")

Two of the three delegated audits disagreed on F2's severity — the implementation audit called the
doubled open table non-blocking ("documented, so a deliberate deviation"), the consistency audit
called it blocking ("ships a false statement about visible output"). Resolved as **blocking** by the
orchestrating reviewer: the duplication is on the surface the ticket's own `## Outcome` names, and
being *documented* does not rescue output that carries no information — especially when the
documentation describing it is itself false.

One delegated claim was **rejected**: an audit reported the refinement cross-check
(`n=119` vs the shipped `n=120`) as a possible parser defect. Re-verified by hand as ordinary corpus
drift — T-126's own departure from TO DO on 2026-08-28 is the 120th dwell sample — and recorded in
the Verification table rather than as a finding.

### Impact sweep (step 8)

No ticket references T-126, none lists it in `depends-on:`, and `2-ready/` is empty, so nothing
downstream encodes an assumption this branch could have invalidated. The six `1-to-do/` tickets
(the five `rick`-interop tickets and T-102) touch neither `## History` parsing nor the `board`
command surface.

### Publish precondition found during review (step 9)

The branch itself is clean against local `main` — `git diff --name-only main...HEAD` carries no
`tickets/` path. But `origin/main` is **4 commits behind** local `main`, so
`origin/main...HEAD` *does* carry four `tickets/` paths today. Under `layout = "in-tree"` that means
`origin main` must be pushed **first**, and the check re-run, before this feature branch is ever
pushed — otherwise the merge request would carry ticket bookkeeping. Recorded here because it will
still be true after the rework round.

## History

- 2026-08-27 — created (TO DO). source: chat: filed while comparing `rick` and `pickle` against
  Anthropic's *The AI-Native SDLC playbook*
  (https://claude.com/blog/the-ai-native-sdlc-playbook); its per-play leading/lagging-indicator
  discipline maps directly onto brine's own dated `## History` lines, which nothing aggregates yet.
- 2026-08-28 — scope cut by its own measurement, and retitled ("Ticket cycle-time / lead-time
  report mined from each ticket's dated History lines" → "pickle board metrics: backlog dwell, lead
  time and open-ticket age from dated History lines"; file renamed to match). All 453 adjacent
  History intervals in this tree were measured: 90% are zero days, and every in-flight interval
  (queue, build, review, rework, done→merged) has median 0 and max 1, because a History line
  carries a date and no time. The time-in-status half of the ticket is therefore **cut, not
  deferred** — it can only report zeros at this resolution. What survives is the three intervals
  that carry spread: backlog dwell, lead time, and the age of tickets still open. Re-graded
  `impact: medium → low` (no consumer exists; the 2026-08-04 precedent refuses prospective demand,
  the same call T-065 and T-105 made), `complexity: medium → low-medium`, `cost: M → S` on the
  finding that `ticket.HistoryEntries` already parses every endpoint this needs. Also settled at
  refinement: own subcommand rather than a `board state --json` schema bump; `--as-of` for the one
  metric that needs a clock, which is also what makes the report deterministic; n/min/p50/p90/max
  and no mean; data-quality cases reported per ticket rather than imputed. The standing rejection
  in `NOTES.md` § "Rejected outright, so they are not re-proposed" is overridden by human
  direction, as T-105's was — recorded in `NOTES.md` § "T-126 refinement (2026-08-28)"
- 2026-08-28 — TO DO → READY: plan complete
- 2026-08-28 — READY → IN DEVELOPMENT: picked up
- 2026-08-28 — plan amended inline: confirmed decision 9's out-of-order example ("two tickets in
  this repo have one", naming T-111 and T-123) is corrected. The refinement-time count came from a
  whole-file dated-line monotonicity scan; the shipped check only compares each metric's own two
  endpoints and inherits `ticket.HistoryEntries`'s first-physical-line-only Kind classification (the
  T-043 guard against reading an arrow out of a folded continuation line), so it does not flag
  either ticket — both carry a note, dated earlier than a neighbouring line, whose *folded* text
  happens to contain a transition-shaped arrow that the correct parser never reads as one. The
  current tree has zero data-quality issues under the real check; the out-of-order code path is
  proven by `internal/metrics/metrics_test.go`'s own fixture instead. Acceptance test case 4 and
  its Expected line amended to match; decision 9 amended in place with the correction quoted
- 2026-08-28 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-28 — IN REVIEW → REWORK: 2 blocking findings (F1 default-clock timezone off-by-one; F2 open table double-lists every open ticket)
- 2026-08-28 — REWORK → IN REVIEW: findings fixed
- 2026-08-28 — IN REVIEW → REWORK: scoped re-review: 2 blocking (R1 call-site mutation survives; R2 consistency test cannot fail)
- 2026-08-28 — REWORK → IN REVIEW: round 2 findings fixed (R1 call-site guard, R2 tautological test)
- 2026-08-28 — IN REVIEW → DONE: review passed: 0 blocking, 5 fixed inline; F1/F2/R1/R2 all closed and mutation-verified
