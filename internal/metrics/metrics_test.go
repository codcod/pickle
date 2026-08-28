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
			// No open lead_time row: a non-terminal ticket's open lead time is
			// `asOf - created`, identical to its open_age, so emitting both
			// double-listed every open ticket (T-126 review, F2).
			wantLead: nil,
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
			wantLead:  nil, // F2: non-terminal, so no open lead_time row
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
			wantLead:  nil, // F2: non-terminal, so no open lead_time row
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

// TestTicketIntervalsEmitsOneOpenRowPerTicket is F2's regression test
// (T-126 review). The defect was an open lead_time row emitted for *every*
// non-terminal unmerged ticket, where it equals open_age by construction, so
// the command's open table listed every open ticket twice with the same
// number. Asserted as a count, because the table-driven test above uses a
// first-match helper and so cannot see a duplicate.
//
// The done-but-unmerged case is the one open lead_time row that carries
// information open_age does not (a done ticket is terminal, so it has no
// open_age at all) — asserted here too, so a fix that removed open lead_time
// altogether would fail rather than pass.
func TestTicketIntervalsEmitsOneOpenRowPerTicket(t *testing.T) {
	def := flow.ForName("brine")
	asOf := mustDate(t, "2026-08-01")

	cases := []struct {
		name        string
		dir         string
		text        string
		wantOpen    int    // how many Open intervals this ticket should yield
		wantMetrics string // which metric carries it
	}{
		{
			name:     "non-terminal: exactly one open row, and it is open_age",
			dir:      "1-to-do",
			text:     "## History\n\n- 2026-07-20 — created (TO DO). source: chat: fixture\n",
			wantOpen: 1, wantMetrics: MetricOpenAge,
		},
		{
			name: "in development: exactly one open row, and it is open_age",
			dir:  "3-in-development",
			text: "## History\n\n- 2026-07-20 — created (TO DO). source: chat: fixture\n" +
				"- 2026-07-21 — TO DO → READY: plan complete\n" +
				"- 2026-07-22 — READY → IN DEVELOPMENT: picked up\n",
			wantOpen: 1, wantMetrics: MetricOpenAge,
		},
		{
			name: "done but unmerged: exactly one open row, and it is lead_time",
			dir:  "6-done",
			text: "## History\n\n- 2026-07-01 — created (TO DO). source: chat: fixture\n" +
				"- 2026-07-02 — TO DO → READY: plan complete\n" +
				"- 2026-07-05 — IN REVIEW → DONE: review clean\n",
			wantOpen: 1, wantMetrics: MetricLeadTime,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tk := &ticket.Ticket{
				ID: "T-900", Dir: c.dir,
				Front: map[string]string{"project": "pickle"},
				Text:  c.text,
			}
			ivs, _ := TicketIntervals(def, tk, statusFor(t, def, c.dir), asOf)

			var open []Interval
			for _, iv := range ivs {
				if iv.Open {
					open = append(open, iv)
				}
			}
			if len(open) != c.wantOpen {
				t.Fatalf("got %d open interval(s) %+v, want %d", len(open), open, c.wantOpen)
			}
			if open[0].Metric != c.wantMetrics {
				t.Errorf("open interval metric = %q, want %q", open[0].Metric, c.wantMetrics)
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

// TestDateOf is F1's regression test (T-126 review): the default `board
// metrics` path converts time.Now() through DateOf, and the bug was that it
// did not — a local wall-clock instant was subtracted from parsed
// midnight-UTC endpoints, so ages moved with the hour and the zone.
//
// Locations are loaded explicitly rather than read from the process TZ: Go
// caches time.Local at first use, so a test that set TZ would not reliably
// change it, and the original defect is precisely a *location* defect. Each
// case therefore states an instant in a named zone and the calendar date a
// user standing in that zone would call "today".
func TestDateOf(t *testing.T) {
	load := func(name string) *time.Location {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Skipf("zoneinfo for %s unavailable: %v", name, err)
		}
		return loc
	}

	cases := []struct {
		name string
		zone string
		// the wall-clock reading in that zone
		y         int
		mo        time.Month
		d, h, min int
		wantDate  string
	}{
		{"UTC midday", "UTC", 2026, time.August, 28, 12, 0, "2026-08-28"},
		{"UTC one minute past midnight", "UTC", 2026, time.August, 28, 0, 1, "2026-08-28"},
		{"UTC one minute to midnight", "UTC", 2026, time.August, 28, 23, 59, "2026-08-28"},
		// UTC+2: just after local midnight is still the *previous* day in UTC.
		// The pre-fix code reported the previous day's age under today's date.
		{"UTC+2 just after midnight", "Europe/Warsaw", 2026, time.August, 28, 0, 30, "2026-08-28"},
		// UTC-11: late local evening is already the *next* day in UTC.
		{"UTC-11 late evening", "Pacific/Midway", 2026, time.August, 27, 20, 30, "2026-08-27"},
		// UTC+14, the far end of the dateline.
		{"UTC+14 early morning", "Pacific/Kiritimati", 2026, time.August, 28, 1, 0, "2026-08-28"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := time.Date(c.y, c.mo, c.d, c.h, c.min, 0, 0, load(c.zone))
			got := DateOf(in)

			if got.Format(dateLayout) != c.wantDate {
				t.Errorf("DateOf(%s) date = %s, want %s", in, got.Format(dateLayout), c.wantDate)
			}
			if got.Location() != time.UTC {
				t.Errorf("DateOf(%s) location = %v, want UTC", in, got.Location())
			}
			if h, m, s := got.Clock(); h != 0 || m != 0 || s != 0 {
				t.Errorf("DateOf(%s) clock = %02d:%02d:%02d, want midnight", in, h, m, s)
			}
		})
	}
}

// TestDateOfDrivesAgesFromTheLocalCalendarDate composes DateOf with
// TicketIntervals and checks the resulting age against a **literal** expected
// value per case — hand-computed from the fixture's created date and the
// calendar date a user in that zone is standing in.
//
// The literals are the point (T-126 re-review, R2). This test previously
// derived its own expectation by re-running the implementation's arithmetic on
// the value under test, which made it near-tautological: it passed with DateOf
// mutated to return the UTC date, and passed again with DateOf mutated to add a
// whole day, while claiming in a comment to compute the expectation
// independently. An expected value that comes from the thing being tested
// cannot falsify it.
func TestDateOfDrivesAgesFromTheLocalCalendarDate(t *testing.T) {
	def := flow.ForName("brine")
	const createdOn = "2026-08-07"

	cases := []struct {
		zone string
		// wall-clock reading in that zone
		y         int
		mo        time.Month
		d, h, min int
		wantAsOf  string // the calendar date a user there calls "today"
		wantDays  int    // hand-computed from createdOn to wantAsOf
	}{
		{"UTC", 2026, time.August, 28, 12, 0, "2026-08-28", 21},
		// East of UTC, just after local midnight: the instant is still 2026-08-27
		// in UTC, so reading it raw gave 20. "Today" is the 28th and the age is 21.
		{"Europe/Warsaw", 2026, time.August, 28, 0, 30, "2026-08-28", 21},
		// West of UTC, late evening: the instant is already 2026-08-28 in UTC, so
		// reading it raw gave 21. "Today" is the 27th and the age is 20.
		{"Pacific/Midway", 2026, time.August, 27, 20, 30, "2026-08-27", 20},
		// Far east of the dateline: genuinely the 29th there.
		{"Pacific/Kiritimati", 2026, time.August, 29, 1, 0, "2026-08-29", 22},
	}

	for _, c := range cases {
		t.Run(c.zone, func(t *testing.T) {
			loc, err := time.LoadLocation(c.zone)
			if err != nil {
				t.Skipf("zoneinfo for %s unavailable: %v", c.zone, err)
			}
			asOf := DateOf(time.Date(c.y, c.mo, c.d, c.h, c.min, 0, 0, loc))

			if got := asOf.Format(dateLayout); got != c.wantAsOf {
				t.Errorf("as-of = %s, want %s", got, c.wantAsOf)
			}

			tk := &ticket.Ticket{
				ID:    "T-900",
				Dir:   "1-to-do",
				Front: map[string]string{"project": "pickle"},
				Text:  "## History\n\n- " + createdOn + " — created (TO DO). source: chat: fixture\n",
			}
			ivs, _ := TicketIntervals(def, tk, statusFor(t, def, "1-to-do"), asOf)
			age := findMetric(ivs, MetricOpenAge)
			if age == nil {
				t.Fatalf("no open_age interval")
			}
			if age.Days != c.wantDays {
				t.Errorf("open_age = %d days, want %d (%s to %s)",
					age.Days, c.wantDays, createdOn, c.wantAsOf)
			}
		})
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
