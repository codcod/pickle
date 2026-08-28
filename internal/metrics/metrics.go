// Package metrics implements the read model behind `pickle board metrics`
// (T-126): three intervals mined from each ticket's dated `## History` lines
// — backlog dwell, lead time, and open-ticket age — aggregated per
// child-project as n/min/p50/p90/max.
//
// Like internal/decisions and internal/audit, this is a leaf, pure
// text-in/values-out package — no printing, no exit codes, no locking of its
// own (the caller, internal/cli/board.go, wraps the call in
// lock.WithShared, the same pattern internal/state/build.go and
// internal/decisions use).
//
// # Why three metrics and not a time-in-status table
//
// The ticket was filed to answer "has review turnaround gotten worse?" —a
// per-status timing question. Refinement measured every adjacent-event
// interval in this repository's own `tickets/` tree (453 of them, across 126
// tickets) and found 90% are zero days, with every in-flight interval
// (queue, build, review, rework, done→merged) at median 0 and max 1. A
// `## History` line carries a date, not a time, so a metric bounded by two
// transitions inside one working session is structurally a column of zeros
// at this resolution. The two intervals that do carry spread are both
// backlog-side — how long an idea waited before refinement, and how long
// from filing to merge — and so is the third useful number: the age of what
// is still open right now. That is what this package computes; a
// time-in-status table is deliberately not built (see tickets/NOTES.md §
// "T-126 refinement (2026-08-28)").
//
// # Endpoints, and what "the initial state" means without a hardcoded name
//
// Every endpoint is read from ticket.HistoryEntries, never from a new scan.
// A ticket always lands in def.Initial() at creation (Spec.Initial's own
// contract — `ticket new` scaffolds nowhere else), so "backlog dwell" is
// simply created ⇒ the first HistoryTransition entry: no HistoryEntry field
// for a transition's origin status needs to exist, and none is added here.
// "Done" is def.DependencySatisfied() — the state a dependency must reach —
// and "will never merge" is any *other* terminal state (flow.State.Terminal).
// No status name, directory, or ticket-id shape is hardcoded; a foreign
// workspace's own flow and prefixes are read from the Definition and the
// loaded Ticket, exactly as internal/decisions already does (T-105
// decision 3).
package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

// CurrentSchema is the Document.Schema value this package emits. It versions
// this package's own wire format independently of internal/state's
// Document.Schema and internal/decisions' — the three evolve on separate
// schedules (T-105 decision 13, applied here the same way).
const CurrentSchema = 1

// The closed metric vocabulary (Interval.Metric / Aggregate.Metric).
const (
	MetricBacklogDwell = "backlog_dwell" // created -> first departure from the flow's initial state
	MetricLeadTime     = "lead_time"     // created -> newest merge line (merged tickets only)
	MetricOpenAge      = "open_age"      // created -> as-of, every ticket not yet in a terminal state
)

// metricOrder is Compute's deterministic tie-break for two intervals of the
// same ticket (decision: "then the metric vocabulary's own order").
var metricOrder = map[string]int{
	MetricBacklogDwell: 0,
	MetricLeadTime:     1,
	MetricOpenAge:      2,
}

// The closed data-quality vocabulary (Issue.Kind) — decision 9's three
// cases. Each excludes the affected interval from its metric rather than
// imputing or clamping it; the ticket is still named here so a reader sees
// what was skipped and why.
const (
	IssueNoCreated       = "no_created"       // no parseable `created (...)` History line at all
	IssueOutOfOrder      = "out_of_order"     // an end date earlier than its start date
	IssueUnparseableDate = "unparseable_date" // a dated line whose date fails to parse as a calendar date
)

// dateLayout is the one date format `## History` lines and --as-of both use.
const dateLayout = "2006-01-02"

// DateOf converts a wall-clock instant into this package's own date
// convention: the calendar date as it reads in t's **own location**,
// re-expressed as midnight UTC.
//
// Every endpoint this package computes comes from time.Parse(dateLayout, …),
// which yields midnight UTC. A caller that passed time.Now() straight in
// would therefore be subtracting a UTC midnight from a local instant, and
// the int(d / 24h) truncation in TicketIntervals would straddle the zone
// boundary: in any non-UTC zone the same tree yields different ages at
// different hours of the day, and the report's own printed as-of date
// disagrees with the ages printed beneath it by one day (T-126 review, F1).
// Converting through the *local calendar date* — not through UTC — is what
// makes "today" mean the day the user believes it is, while still landing on
// the midnight-UTC grid every parsed endpoint sits on.
func DateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Interval is one computed duration for one ticket. Metric is one of the
// constants above. Open is true in exactly two cases: an open_age row (which
// has no "closed" form — it always measures a still-running age), and a
// lead_time row on a ticket that has reached the done state without a merge
// line, since "done ≠ merged" is a real state of this flow (rules §3) and a
// done ticket is terminal, so open_age does not cover it. A non-terminal
// unmerged ticket gets no lead_time row at all — its open lead time and its
// open age are the same two endpoints, so reporting both duplicated every
// open ticket (T-126 review, F2). Compute excludes an Open lead_time row
// from that aggregate's sample, since it is not yet a completed duration and
// would understate the true figure.
type Interval struct {
	TicketID string `json:"ticket_id"`
	Prefix   string `json:"prefix"`
	Num      int    `json:"num"`
	Project  string `json:"project"`
	Status   string `json:"status"` // display name, e.g. "IN REVIEW"
	Dir      string `json:"dir"`
	Metric   string `json:"metric"`
	Days     int    `json:"days"`
	Start    string `json:"start"`         // YYYY-MM-DD
	End      string `json:"end,omitempty"` // YYYY-MM-DD; "" when Open
	Open     bool   `json:"open"`
}

// Issue is one per-ticket data-quality case (decision 9): the ticket, which
// of the three closed Kind values applies, and a one-line human explanation.
type Issue struct {
	TicketID string `json:"ticket_id"`
	Kind     string `json:"kind"`
	Detail   string `json:"detail"`
}

// Aggregate is one metric's n/min/p50/p90/max for one child-project (or the
// synthetic "all" project, emitted only when more than one child is
// registered). Percentiles use the nearest-rank method on the sorted sample
// — element at ceil(q·n), 1-indexed — never interpolated, and there is
// deliberately no mean: a zero-heavy, day-resolution sample is not the place
// for a statistic that implies more precision than the data carries.
type Aggregate struct {
	Metric  string `json:"metric"`
	Project string `json:"project"`
	N       int    `json:"n"`
	Min     int    `json:"min"`
	P50     int    `json:"p50"`
	P90     int    `json:"p90"`
	Max     int    `json:"max"`
}

// TicketIntervals computes every interval and issue for one loaded ticket
// already known to sit in status (the flow.State for t.Dir). Fixture-testable
// with a literal ticket file's text; Compute is the only caller of the
// LoadAll+aggregation half.
func TicketIntervals(def *flow.Definition, t *ticket.Ticket, status flow.State, asOf time.Time) ([]Interval, []Issue) {
	prefix, num, ok := ticket.SplitID(t.ID)
	if !ok {
		return nil, nil // malformed id — not this command's job, mirrors internal/decisions.Query
	}

	entries := ticket.HistoryEntries(def, t.Text)

	var created time.Time
	haveCreated := false
	for _, e := range entries {
		if e.Kind == ticket.HistoryCreated {
			if d, err := time.Parse(dateLayout, e.Date); err == nil {
				created, haveCreated = d, true
			}
			break // the first created line is the one that counts
		}
	}
	if !haveCreated {
		return nil, []Issue{{TicketID: t.ID, Kind: IssueNoCreated,
			Detail: "no parseable `created (...)` History line"}}
	}

	var issues []Issue
	var out []Interval

	base := func(metric string) Interval {
		return Interval{
			TicketID: t.ID, Prefix: prefix, Num: num,
			Project: t.Project(), Status: status.Name, Dir: t.Dir, Metric: metric,
		}
	}

	// addClosed appends a completed [start, end) interval, or an out_of_order
	// issue when end precedes start — never a negative Days, never clamped to
	// zero (decision 9).
	addClosed := func(metric string, start, end time.Time) {
		days := int(end.Sub(start) / (24 * time.Hour))
		if days < 0 {
			issues = append(issues, Issue{TicketID: t.ID, Kind: IssueOutOfOrder,
				Detail: fmt.Sprintf("%s: %s is before %s", metric, end.Format(dateLayout), start.Format(dateLayout))})
			return
		}
		iv := base(metric)
		iv.Days, iv.Start, iv.End = days, start.Format(dateLayout), end.Format(dateLayout)
		out = append(out, iv)
	}

	// addOpen appends a still-running [start, asOf) interval.
	addOpen := func(metric string, start time.Time) {
		days := int(asOf.Sub(start) / (24 * time.Hour))
		if days < 0 {
			issues = append(issues, Issue{TicketID: t.ID, Kind: IssueOutOfOrder,
				Detail: fmt.Sprintf("%s: as-of %s is before created %s", metric, asOf.Format(dateLayout), start.Format(dateLayout))})
			return
		}
		iv := base(metric)
		iv.Days, iv.Start, iv.Open = days, start.Format(dateLayout), true
		out = append(out, iv)
	}

	// Backlog dwell: a ticket always lands in def.Initial() at creation, so
	// the *first* transition entry is, by construction, the first departure
	// from the initial state — no per-entry "from" needs deriving or storing.
	// A ticket that has not transitioned yet contributes no dwell interval
	// (only open_age, below); one sent back to the backlog and picked up
	// again still counts only its first wait (re-entry is not modelled).
	var mergedDate string
	for _, e := range entries {
		if e.Kind == ticket.HistoryTransition {
			if d, err := time.Parse(dateLayout, e.Date); err == nil {
				addClosed(MetricBacklogDwell, created, d)
			} else {
				issues = append(issues, Issue{TicketID: t.ID, Kind: IssueUnparseableDate,
					Detail: "transition date " + e.Date + " is not a valid date"})
			}
			break
		}
	}
	for _, e := range entries {
		if e.Kind == ticket.HistoryMerged {
			mergedDate = e.Date // last one wins: the newest merge line
		}
	}

	// Lead time: excluded entirely for a ticket in a terminal state that is
	// not "done" (brine: 7-dropped/) — nothing will ever merge it, so it is
	// not zero, it is absent.
	//
	// An *open* lead-time row is emitted for exactly one case: a ticket that
	// has reached the done state but carries no merge line. "done != merged"
	// is a real state of this flow (rules §3), and open_age does not cover it
	// (a done ticket is terminal), so without this row that wait would be
	// invisible. It is still excluded from the closed aggregate by Compute.
	//
	// Every *other* unmerged ticket gets no lead-time row at all, because for
	// a non-terminal ticket the open lead time is `asOf - created` — the same
	// two endpoints open_age already reports, hence always the identical
	// number. Emitting both listed every open ticket twice with the same age
	// on the command's flagship human surface, carrying no information the
	// open_age row did not already carry (T-126 review, F2).
	doneDir := def.DependencySatisfied().Dir
	willNeverMerge := status.Terminal && status.Dir != doneDir
	if !willNeverMerge {
		switch {
		case mergedDate != "":
			if d, err := time.Parse(dateLayout, mergedDate); err == nil {
				addClosed(MetricLeadTime, created, d)
			} else {
				issues = append(issues, Issue{TicketID: t.ID, Kind: IssueUnparseableDate,
					Detail: "merge date " + mergedDate + " is not a valid date"})
			}
		case status.Dir == doneDir:
			addOpen(MetricLeadTime, created)
		}
	}

	// Open age: every ticket not yet in a terminal state, created -> as-of.
	if !status.Terminal {
		addOpen(MetricOpenAge, created)
	}

	return out, issues
}

// Options selects Compute's filter and clock. AsOf is mandatory at the CLI
// (it defaults to today there); Compute itself takes whatever it is given,
// which is what makes a fixed AsOf run a pure function of the tree.
type Options struct {
	Project string
	AsOf    time.Time
}

// Result is Compute's return value.
type Result struct {
	AsOf       string
	Aggregates []Aggregate
	Intervals  []Interval
	Issues     []Issue
}

// Compute loads every ticket under root, computes every interval per
// TicketIntervals, and aggregates per metric and child-project. An
// unregistered o.Project is an error; every other filter combination,
// including one that matches nothing, returns an empty (non-nil) Result.
func Compute(def *flow.Definition, root string, cfg *config.Config, o Options) (Result, error) {
	if o.Project != "" {
		if _, ok := cfg.Project(o.Project); !ok {
			return Result{}, fmt.Errorf("project %q is not registered", o.Project)
		}
	}

	tickets, _ := ticket.LoadAll(def, root)
	var intervals []Interval
	var issues []Issue
	for _, t := range tickets {
		if o.Project != "" && t.Project() != o.Project {
			continue
		}
		status, ok := def.ByDir(t.Dir)
		if !ok {
			continue // LoadAll only ever walks def.States() dirs, so this always resolves
		}
		iv, is := TicketIntervals(def, t, status, o.AsOf)
		intervals = append(intervals, iv...)
		issues = append(issues, is...)
	}

	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].Prefix != intervals[j].Prefix {
			return intervals[i].Prefix < intervals[j].Prefix
		}
		if intervals[i].Num != intervals[j].Num {
			return intervals[i].Num < intervals[j].Num
		}
		return metricOrder[intervals[i].Metric] < metricOrder[intervals[j].Metric]
	})
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].TicketID != issues[j].TicketID {
			return issues[i].TicketID < issues[j].TicketID
		}
		return issues[i].Kind < issues[j].Kind
	})

	return Result{
		AsOf:       o.AsOf.Format(dateLayout),
		Aggregates: aggregate(intervals, cfg, o.Project),
		Intervals:  intervals,
		Issues:     issues,
	}, nil
}

// aggregate computes n/min/p50/p90/max per metric and child-project. Every
// registered project (or, with o.Project set, that one project alone) gets a
// row for every metric even when its sample is empty — a registered child
// with no tickets of a given kind prints n=0, so an empty child is visible
// rather than absent (decision 11). An "all" row is added per metric only
// when more than one child is registered, so a single-child workspace's
// output is not doubled for no reason.
//
// A lead_time row on a ticket that has not merged yet (Interval.Open) is
// excluded from this aggregate's sample: it is not yet a completed duration,
// and folding it in would understate lead time by counting an interval that
// is still growing. open_age carries no closed form at all, so every one of
// its rows is Open and every one counts — the metric measures current
// pending age, not a finished one.
func aggregate(intervals []Interval, cfg *config.Config, onlyProject string) []Aggregate {
	metrics := []string{MetricBacklogDwell, MetricLeadTime, MetricOpenAge}

	var projects []string
	if onlyProject != "" {
		projects = []string{onlyProject}
	} else {
		for _, p := range cfg.Projects {
			projects = append(projects, p.Name)
		}
	}

	samples := map[string]map[string][]int{} // metric -> project -> days
	for _, m := range metrics {
		samples[m] = map[string][]int{}
	}
	for _, iv := range intervals {
		if iv.Metric == MetricLeadTime && iv.Open {
			continue
		}
		samples[iv.Metric][iv.Project] = append(samples[iv.Metric][iv.Project], iv.Days)
		samples[iv.Metric]["all"] = append(samples[iv.Metric]["all"], iv.Days)
	}

	var out []Aggregate
	for _, m := range metrics {
		for _, p := range projects {
			out = append(out, summarize(m, p, samples[m][p]))
		}
		if onlyProject == "" && len(projects) > 1 {
			out = append(out, summarize(m, "all", samples[m]["all"]))
		}
	}
	return out
}

// summarize builds one Aggregate row. An empty days yields an explicit
// all-zero row rather than being omitted (decision 11).
func summarize(metric, project string, days []int) Aggregate {
	a := Aggregate{Metric: metric, Project: project}
	if len(days) == 0 {
		return a
	}
	sorted := append([]int(nil), days...)
	sort.Ints(sorted)
	a.N = len(sorted)
	a.Min = sorted[0]
	a.Max = sorted[len(sorted)-1]
	a.P50 = percentile(sorted, 0.5)
	a.P90 = percentile(sorted, 0.9)
	return a
}

// percentile returns the nearest-rank percentile of an already-sorted slice:
// the element at 1-indexed rank ceil(q·n), the convention decision 10 states
// so no reader has to guess which interpolation was used. Defined for any
// n >= 1; the caller (summarize) never calls it with an empty slice.
func percentile(sorted []int, q float64) int {
	n := len(sorted)
	rank := int(math.Ceil(q * float64(n)))
	if rank < 1 {
		rank = 1
	}
	if rank > n {
		rank = n
	}
	return sorted[rank-1]
}

// Document is the JSON envelope `pickle board metrics --json` prints: this
// package's own small, independently versioned wire format (T-105 decision
// 13's pattern, applied again) — internal/state's Document and
// internal/decisions' Document are both untouched by this command.
type Document struct {
	Schema        int         `json:"schema"`
	PickleVersion string      `json:"pickle_version"`
	AsOf          string      `json:"as_of"`
	Resolution    string      `json:"resolution"` // always "days" — see the package doc
	Filters       Filters     `json:"filters"`
	Aggregates    []Aggregate `json:"aggregates"`
	Intervals     []Interval  `json:"intervals"`
	Issues        []Issue     `json:"issues"`
}

// Filters records the query that produced a Document, so a consumer of
// --json output does not have to remember its own invocation.
type Filters struct {
	Project string `json:"project"`
	AsOf    string `json:"as_of"`
}

// NonNil returns s, or an empty non-nil slice when s is nil — so a Document
// field marshals as `[]`, never `null`, on an empty result (mirrors
// internal/state's own nonNil and internal/decisions' JSON contract).
func NonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
