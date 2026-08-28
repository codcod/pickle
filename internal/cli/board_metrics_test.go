package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMetricsTicket writes a minimal, audit-independent ticket file directly
// into tickets/<dir>/, bypassing `ticket new` and the state machine: `board
// metrics` is a read-only projection over ticket.LoadAll + HistoryEntries, so
// these fixtures only need frontmatter and a hand-written `## History` body —
// exactly the shape writeDecisionTicket in board_decisions_test.go uses for
// the same reason. historyBody is pasted verbatim under `## History`.
func writeMetricsTicket(t *testing.T, root, dir, id, project, historyBody string) {
	t.Helper()
	slug := strings.ToLower(strings.ReplaceAll(id, "-", "")) + "-fixture"
	text := "---\n" +
		"id: " + id + "\n" +
		"title: fixture ticket " + id + "\n" +
		"project: " + project + "\n" +
		"depends-on: []\n" +
		"spawned-by: []\n" +
		"impact: low\n" +
		"complexity: low\n" +
		"cost: S\n" +
		"---\n\n" +
		"# " + id + " — fixture ticket\n\n" +
		"## Outcome\n\nSomething observable changes.\n\n" +
		"## Description\n\nFixture prose.\n\n" +
		"## Implementation Plan\n\n" +
		"## Review\n\n" +
		"## History\n\n" + historyBody
	p := filepath.Join(root, "tickets", dir, id+"-"+slug+".md")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// boardMetricsSandbox builds a two-child workspace (the child filter and the
// multi-child "all" aggregate row both need one) seeded with fixed-date
// History across every case TicketIntervals' own fixture test exercises at
// the unit level, plus the two data-quality issues.
func boardMetricsSandbox(t *testing.T) (root string) {
	t.Helper()
	root = newProject(t) // registers child "demo" at "."
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Run(nil, "test", []string{"project", "add", "lib", "lib", "--ticket-prefix", "LIB"}); got != exitOK {
		t.Fatalf("project add lib = %d, want %d", got, exitOK)
	}

	// demo T-001: a full merged lifecycle — closed backlog_dwell (4d) and
	// closed lead_time (11d).
	writeMetricsTicket(t, root, "6-done", "T-001", "demo",
		"- 2026-07-01 — created (TO DO). source: chat: test fixture\n"+
			"- 2026-07-05 — TO DO → READY: plan complete\n"+
			"- 2026-07-06 — READY → IN DEVELOPMENT: picked up\n"+
			"- 2026-07-08 — IN DEVELOPMENT → IN REVIEW: acceptance green\n"+
			"- 2026-07-10 — IN REVIEW → DONE: review clean\n"+
			"- 2026-07-12 — merged to main (MR !1, abc1234)\n")

	// demo T-002: still in the backlog — open_age and open lead_time only.
	writeMetricsTicket(t, root, "1-to-do", "T-002", "demo",
		"- 2026-07-20 — created (TO DO). source: chat: test fixture\n")

	// demo T-003: done but unmerged — open lead_time, no open_age (terminal).
	writeMetricsTicket(t, root, "6-done", "T-003", "demo",
		"- 2026-07-01 — created (TO DO). source: chat: test fixture\n"+
			"- 2026-07-02 — TO DO → READY: plan complete\n"+
			"- 2026-07-03 — READY → IN DEVELOPMENT: picked up\n"+
			"- 2026-07-04 — IN DEVELOPMENT → IN REVIEW: acceptance green\n"+
			"- 2026-07-05 — IN REVIEW → DONE: review clean\n")

	// demo T-004: out-of-order — the departure predates creation.
	writeMetricsTicket(t, root, "2-ready", "T-004", "demo",
		"- 2026-07-10 — created (TO DO). source: chat: test fixture\n"+
			"- 2026-07-05 — TO DO → READY: plan complete\n")

	// demo T-005: no created line at all.
	writeMetricsTicket(t, root, "1-to-do", "T-005", "demo",
		"- 2026-07-01 — plan amended inline: dropped task 3\n")

	// lib LIB-001: one merged ticket under its own prefix, to prove the
	// child filter and the multi-child "all" aggregate row both hold.
	writeMetricsTicket(t, root, "6-done", "LIB-001", "lib",
		"- 2026-07-01 — created (TO DO). source: chat: test fixture\n"+
			"- 2026-07-02 — TO DO → READY: plan complete\n"+
			"- 2026-07-03 — READY → IN DEVELOPMENT: picked up\n"+
			"- 2026-07-04 — IN DEVELOPMENT → IN REVIEW: acceptance green\n"+
			"- 2026-07-05 — IN REVIEW → DONE: review clean\n"+
			"- 2026-07-06 — merged to main (MR !2, def5678)\n")

	return root
}

// TestBoardMetricsDefaultReport is acceptance case 1: the default report
// prints both tables and the resolution note.
func TestBoardMetricsDefaultReport(t *testing.T) {
	boardMetricsSandbox(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "metrics", "--as-of", "2026-08-01"})
	})
	if code != exitOK {
		t.Fatalf("board metrics = %d, want %d:\n%s", code, exitOK, out)
	}
	for _, want := range []string{"METRIC", "TICKET", "backlog_dwell", "lead_time", "open_age", "intervals are whole days"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

// TestBoardMetricsAggregatesHaveSpread is acceptance case 2: the aggregates
// carry the expected n/min/max for a project with a known fixture set.
func TestBoardMetricsAggregatesHaveSpread(t *testing.T) {
	boardMetricsSandbox(t)
	doc := boardMetricsJSON(t, "--as-of", "2026-08-01")
	agg := findAggregate(t, doc, "lead_time", "demo")
	// T-001 (closed, 11d) is the only *closed* lead_time sample for demo —
	// T-002 and T-003 are Open and excluded from the aggregate.
	if agg["n"].(float64) != 1 || agg["max"].(float64) != 11 {
		t.Errorf("lead_time/demo aggregate = %+v, want n=1 max=11", agg)
	}
	// backlog_dwell/demo: T-001 (4d, closed) and T-003 (1d, closed) both
	// contribute; T-002 has not departed yet and T-004's departure predates
	// its own creation, so it is an out_of_order issue instead of a sample.
	dwell := findAggregate(t, doc, "backlog_dwell", "demo")
	if dwell["n"].(float64) != 2 || dwell["min"].(float64) != 1 || dwell["max"].(float64) != 4 {
		t.Errorf("backlog_dwell/demo aggregate = %+v, want n=2 min=1 max=4", dwell)
	}
}

// TestBoardMetricsOpenTicketsListed is acceptance case 3: every open
// interval is listed, none truncated, oldest first.
func TestBoardMetricsOpenTicketsListed(t *testing.T) {
	boardMetricsSandbox(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "metrics", "--as-of", "2026-08-01"}); got != exitOK {
			t.Fatalf("= %d", got)
		}
	})
	// T-002 (created 07-20, asOf 08-01 -> 12d) and T-003 (open lead_time,
	// created 07-01 -> 31d) must both appear; T-003's open age is older.
	i3 := strings.Index(out, "T-003")
	i2 := strings.Index(out, "T-002")
	if i3 < 0 || i2 < 0 {
		t.Fatalf("expected both T-002 and T-003 in the open table, got:\n%s", out)
	}
	if i3 > i2 {
		t.Errorf("expected T-003 (older, 31d) to sort before T-002 (12d), got:\n%s", out)
	}
}

// TestBoardMetricsIssuesReported is acceptance case 4: both data-quality
// cases are named with their ticket id, never silently dropped.
func TestBoardMetricsIssuesReported(t *testing.T) {
	boardMetricsSandbox(t)
	doc := boardMetricsJSON(t, "--as-of", "2026-08-01")
	issues, _ := doc["issues"].([]any)
	found := map[string]bool{}
	for _, raw := range issues {
		is := raw.(map[string]any)
		found[is["kind"].(string)+" "+is["ticket_id"].(string)] = true
	}
	if !found["out_of_order T-004"] {
		t.Errorf("expected an out_of_order issue for T-004, got: %+v", issues)
	}
	if !found["no_created T-005"] {
		t.Errorf("expected a no_created issue for T-005, got: %+v", issues)
	}
}

// TestBoardMetricsDeterministic is confirmed decision 8: a fixed --as-of run
// is byte-identical.
func TestBoardMetricsDeterministic(t *testing.T) {
	boardMetricsSandbox(t)
	run := func() string {
		return captureStdout(t, func() {
			if got := Run(nil, "test", []string{"board", "metrics", "--as-of", "2026-08-01", "--json"}); got != exitOK {
				t.Fatalf("= %d", got)
			}
		})
	}
	a, b := run(), run()
	if a != b {
		t.Errorf("two fixed --as-of runs differ:\n--- first ---\n%s\n--- second ---\n%s", a, b)
	}
}

// TestBoardMetricsProjectFilter is acceptance case 6: filtering by a
// registered child narrows the result and never leaks another child's data.
func TestBoardMetricsProjectFilter(t *testing.T) {
	boardMetricsSandbox(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "metrics", "--project", "lib", "--as-of", "2026-08-01"}); got != exitOK {
			t.Fatalf("= %d", got)
		}
	})
	// LIB-001 is closed (merged), so it never appears in the open table —
	// that is correct, not a leak; the aggregate assertion below is what
	// actually proves the filter took effect.
	if strings.Contains(out, "T-001") || strings.Contains(out, "T-002") || strings.Contains(out, "T-003") {
		t.Errorf("--project lib leaked a demo ticket, got:\n%s", out)
	}
	doc := boardMetricsJSON(t, "--project", "lib", "--as-of", "2026-08-01")
	agg := findAggregate(t, doc, "lead_time", "lib")
	if agg["n"].(float64) != 1 {
		t.Errorf("lead_time/lib aggregate = %+v, want n=1", agg)
	}
}

// TestBoardMetricsEmptyFilterIsNotAnError is acceptance case 6's other half:
// a filter combination matching nothing is exit 0, not an error.
func TestBoardMetricsEmptyFilterIsNotAnError(t *testing.T) {
	newProject(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "metrics", "--as-of", "2026-08-01"})
	})
	if code != exitOK {
		t.Fatalf("board metrics on an empty tree = %d, want %d:\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "0 ticket(s), 0 open") {
		t.Errorf("expected the zero-result summary line, got:\n%s", out)
	}
}

// TestBoardMetricsErrors is acceptance case 7: an unregistered project and a
// malformed --as-of are both errors (exit 1).
func TestBoardMetricsErrors(t *testing.T) {
	boardMetricsSandbox(t)
	cases := []struct {
		name string
		args []string
	}{
		{"unregistered project", []string{"board", "metrics", "--project", "no-such-child"}},
		{"malformed as-of", []string{"board", "metrics", "--as-of", "2026-13-99"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var code int
			errOut := captureStderr(t, func() {
				code = Run(nil, "test", c.args)
			})
			if code == exitOK {
				t.Errorf("%v = exitOK, want a non-zero exit", c.args)
			}
			if errOut == "" {
				t.Error("expected an error message on stderr, got none")
			}
		})
	}
}

// TestBoardMetricsJSON is acceptance case 8: the JSON form is well-formed,
// versioned, and every slice marshals as [], never null.
func TestBoardMetricsJSON(t *testing.T) {
	boardMetricsSandbox(t)
	doc := boardMetricsJSON(t, "--as-of", "2026-08-01")
	if doc["schema"].(float64) != 1 {
		t.Errorf("schema = %v, want 1", doc["schema"])
	}
	if doc["resolution"] != "days" {
		t.Errorf("resolution = %v, want %q", doc["resolution"], "days")
	}
	if doc["as_of"] != "2026-08-01" {
		t.Errorf("as_of = %v, want 2026-08-01", doc["as_of"])
	}
	if _, ok := doc["intervals"].([]any); !ok {
		t.Errorf("intervals did not decode as an array: %v", doc["intervals"])
	}

	t.Run("empty result marshals every slice as [] not null", func(t *testing.T) {
		empty := captureStdout(t, func() {
			if got := Run(nil, "test", []string{"board", "metrics", "--project", "demo", "--as-of", "2026-08-01", "--json"}); got != exitOK {
				t.Fatalf("= %d", got)
			}
		})
		if strings.Contains(empty, "null") {
			t.Errorf("a field marshaled as null, want []:\n%s", empty)
		}
	})
}

// TestBoardMetricsBadFlag exercises the usage-error path directly (also
// covered in TestRunExitCodes).
func TestBoardMetricsBadFlag(t *testing.T) {
	boardMetricsSandbox(t)
	if got := Run(nil, "test", []string{"board", "metrics", "--bogus"}); got != exitUsage {
		t.Errorf("board metrics --bogus = %d, want %d", got, exitUsage)
	}
}

// boardMetricsJSON runs `board metrics --json` with extraArgs and decodes the
// document into a generic map, for assertions that need to reach into
// specific fields without a full struct mirror.
func boardMetricsJSON(t *testing.T, extraArgs ...string) map[string]any {
	t.Helper()
	args := append([]string{"board", "metrics", "--json"}, extraArgs...)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", args); got != exitOK {
			t.Fatalf("board metrics --json %v = %d, want %d", extraArgs, got, exitOK)
		}
	})
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	return doc
}

// findAggregate returns the aggregate row for (metric, project), failing the
// test if none is present.
func findAggregate(t *testing.T, doc map[string]any, metric, project string) map[string]any {
	t.Helper()
	aggs, _ := doc["aggregates"].([]any)
	for _, raw := range aggs {
		a := raw.(map[string]any)
		if a["metric"] == metric && a["project"] == project {
			return a
		}
	}
	t.Fatalf("no aggregate row for metric=%q project=%q in %+v", metric, project, aggs)
	return nil
}
