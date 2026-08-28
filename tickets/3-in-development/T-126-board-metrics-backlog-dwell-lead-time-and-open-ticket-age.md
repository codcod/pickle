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

<!-- empty until IN REVIEW -->

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
