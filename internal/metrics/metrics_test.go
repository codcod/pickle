package metrics

import (
	"testing"
	"time"

	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return d
}

// findMetric returns the first interval of the named metric, or nil.
func findMetric(ivs []Interval, metric string) *Interval {
	for i := range ivs {
		if ivs[i].Metric == metric {
			return &ivs[i]
		}
	}
	return nil
}

func statusFor(t *testing.T, def *flow.Definition, dir string) flow.State {
	t.Helper()
	s, ok := def.ByDir(dir)
	if !ok {
		t.Fatalf("no state for dir %q", dir)
	}
	return s
}

func TestTicketIntervals(t *testing.T) {
	def := flow.ForName("brine")

	cases := []struct {
		name      string
		dir       string
		text      string
		asOf      string
		wantDwell *Interval // nil = absent
		wantLead  *Interval
		wantOpen  *Interval
		wantIssue string // "" = none
	}{
		{
			name: "full merged lifecycle",
			dir:  "6-done",
			text: "## History\n\n" +
				"- 2026-07-01 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-05 — TO DO → READY: plan complete\n" +
				"- 2026-07-06 — READY → IN DEVELOPMENT: picked up\n" +
				"- 2026-07-08 — IN DEVELOPMENT → IN REVIEW: acceptance green\n" +
				"- 2026-07-10 — IN REVIEW → DONE: review clean\n" +
				"- 2026-07-12 — merged to main (MR !1, abc1234)\n",
			asOf:      "2026-08-01",
			wantDwell: &Interval{Metric: MetricBacklogDwell, Days: 4, Start: "2026-07-01", End: "2026-07-05"},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 11, Start: "2026-07-01", End: "2026-07-12"},
		},
		{
			name: "still in the initial state: open age only, no dwell, no lead time",
			dir:  "1-to-do",
			text: "## History\n\n" +
				"- 2026-07-20 — created (TO DO). source: chat: filed\n",
			asOf:      "2026-08-01",
			wantDwell: nil,
			wantOpen:  &Interval{Metric: MetricOpenAge, Days: 12, Start: "2026-07-20", Open: true},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 12, Start: "2026-07-20", Open: true},
		},
		{
			name: "dropped straight out of the backlog: dwell counts, lead time absent",
			dir:  "7-dropped",
			text: "## History\n\n" +
				"- 2026-07-01 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-03 — TO DO → DROPPED: no longer worth it\n",
			asOf:      "2026-08-01",
			wantDwell: &Interval{Metric: MetricBacklogDwell, Days: 2, Start: "2026-07-01", End: "2026-07-03"},
			wantLead:  nil, // will never merge
			wantOpen:  nil, // terminal, not open
		},
		{
			name: "done but unmerged: open lead time, no open_age (terminal)",
			dir:  "6-done",
			text: "## History\n\n" +
				"- 2026-07-01 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-02 — TO DO → READY: plan complete\n" +
				"- 2026-07-03 — READY → IN DEVELOPMENT: picked up\n" +
				"- 2026-07-04 — IN DEVELOPMENT → IN REVIEW: acceptance green\n" +
				"- 2026-07-05 — IN REVIEW → DONE: review clean\n",
			asOf:      "2026-07-20",
			wantDwell: &Interval{Metric: MetricBacklogDwell, Days: 1, Start: "2026-07-01", End: "2026-07-02"},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 19, Start: "2026-07-01", Open: true},
			wantOpen:  nil,
		},
		{
			name: "legacy MERGED: form is still read as a merge line",
			dir:  "6-done",
			text: "## History\n\n" +
				"- 2026-07-01 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-02 — TO DO → READY: plan complete\n" +
				"- 2026-07-03 — READY → IN DEVELOPMENT: picked up\n" +
				"- 2026-07-04 — IN DEVELOPMENT → IN REVIEW: acceptance green\n" +
				"- 2026-07-05 — IN REVIEW → DONE: review clean\n" +
				"- 2026-07-09 — MERGED: feat/T-example → main (abc1234)\n",
			asOf:      "2026-08-01",
			wantDwell: &Interval{Metric: MetricBacklogDwell, Days: 1, Start: "2026-07-01", End: "2026-07-02"},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 8, Start: "2026-07-01", End: "2026-07-09"},
		},
		{
			name:      "no created line at all: reported as an issue, nothing computed",
			dir:       "1-to-do",
			text:      "## History\n\n- 2026-07-01 — plan amended inline: dropped task 3\n",
			asOf:      "2026-08-01",
			wantIssue: IssueNoCreated,
		},
		{
			name: "out-of-order History: negative interval reported, not clamped",
			dir:  "2-ready",
			text: "## History\n\n" +
				"- 2026-07-10 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-05 — TO DO → READY: plan complete\n", // earlier than created
			asOf:      "2026-08-01",
			wantIssue: IssueOutOfOrder,
			wantDwell: nil,
			wantOpen:  &Interval{Metric: MetricOpenAge, Days: 22, Start: "2026-07-10", Open: true},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 22, Start: "2026-07-10", Open: true},
		},
		{
			name: "returned to the backlog and left twice: the first departure wins",
			dir:  "3-in-development",
			text: "## History\n\n" +
				"- 2026-07-01 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-05 — TO DO → READY: plan complete\n" + // first departure: day 4
				"- 2026-07-06 — READY → TO DO: re-description needed\n" +
				"- 2026-07-20 — TO DO → READY: plan complete, second time\n" + // must NOT win
				"- 2026-07-21 — READY → IN DEVELOPMENT: picked up\n",
			asOf:      "2026-08-01",
			wantDwell: &Interval{Metric: MetricBacklogDwell, Days: 4, Start: "2026-07-01", End: "2026-07-05"},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 31, Start: "2026-07-01", Open: true},
			wantOpen:  &Interval{Metric: MetricOpenAge, Days: 31, Start: "2026-07-01", Open: true},
		},
		{
			name: "same-day lifecycle: zero, not absent",
			dir:  "6-done",
			text: "## History\n\n" +
				"- 2026-07-01 — created (TO DO). source: chat: filed\n" +
				"- 2026-07-01 — TO DO → READY: plan complete\n" +
				"- 2026-07-01 — READY → IN DEVELOPMENT: picked up\n" +
				"- 2026-07-01 — IN DEVELOPMENT → IN REVIEW: acceptance green\n" +
				"- 2026-07-01 — IN REVIEW → DONE: review clean\n" +
				"- 2026-07-01 — merged to main (MR !1, abc1234)\n",
			asOf:      "2026-08-01",
			wantDwell: &Interval{Metric: MetricBacklogDwell, Days: 0, Start: "2026-07-01", End: "2026-07-01"},
			wantLead:  &Interval{Metric: MetricLeadTime, Days: 0, Start: "2026-07-01", End: "2026-07-01"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tk := &ticket.Ticket{
				ID:    "T-900",
				Dir:   c.dir,
				Front: map[string]string{"project": "pickle"},
				Text:  c.text,
			}
			status := statusFor(t, def, c.dir)
			ivs, issues := TicketIntervals(def, tk, status, mustDate(t, c.asOf))

			check := func(label, metric string, want *Interval) {
				got := findMetric(ivs, metric)
				if want == nil {
					if got != nil {
						t.Errorf("%s: got unexpected %s interval %+v", label, metric, got)
					}
					return
				}
				if got == nil {
					t.Fatalf("%s: missing %s interval", label, metric)
				}
				if got.Days != want.Days || got.Start != want.Start || got.End != want.End || got.Open != want.Open {
					t.Errorf("%s: %s = %+v, want Days=%d Start=%s End=%s Open=%v",
						label, metric, got, want.Days, want.Start, want.End, want.Open)
				}
			}
			check(c.name, MetricBacklogDwell, c.wantDwell)
			check(c.name, MetricLeadTime, c.wantLead)
			check(c.name, MetricOpenAge, c.wantOpen)

			if c.wantIssue == "" {
				if len(issues) != 0 {
					t.Errorf("%s: unexpected issues %+v", c.name, issues)
				}
			} else {
				found := false
				for _, is := range issues {
					if is.Kind == c.wantIssue {
						found = true
					}
				}
				if !found {
					t.Errorf("%s: issues = %+v, want one of kind %q", c.name, issues, c.wantIssue)
				}
			}
		})
	}
}

// TestTicketIntervalsMalformedID mirrors internal/decisions.Query's own
// contract: a malformed id is silently skipped, not this package's job to
// report (ticket.LoadAll's own issues already cover it).
func TestTicketIntervalsMalformedID(t *testing.T) {
	def := flow.ForName("brine")
	tk := &ticket.Ticket{
		ID:    "not-an-id-at-all",
		Dir:   "1-to-do",
		Front: map[string]string{"project": "pickle"},
		Text:  "## History\n\n- 2026-07-01 — created (TO DO). source: chat: filed\n",
	}
	status := statusFor(t, def, "1-to-do")
	ivs, issues := TicketIntervals(def, tk, status, mustDate(t, "2026-08-01"))
	if ivs != nil || issues != nil {
		t.Errorf("malformed id: got ivs=%+v issues=%+v, want nil, nil", ivs, issues)
	}
}

func TestPercentile(t *testing.T) {
	cases := []struct {
		name   string
		sorted []int
		q      float64
		want   int
	}{
		{"n=1 p50", []int{7}, 0.5, 7},
		{"n=1 p90", []int{7}, 0.9, 7},
		{"n=2 p50 takes the lower of the two", []int{1, 10}, 0.5, 1},
		{"n=2 p90 takes the higher of the two", []int{1, 10}, 0.9, 10},
		{"n=10 p50 is the 5th ranked value", []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 0.5, 4},
		{"n=10 p90 is the 9th ranked value", []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 0.9, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := percentile(c.sorted, c.q); got != c.want {
				t.Errorf("percentile(%v, %v) = %d, want %d", c.sorted, c.q, got, c.want)
			}
		})
	}
}
