package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeDecisionTicket writes a minimal, audit-clean ticket file directly into
// tickets/<dir>/, bypassing `ticket new` and the state machine: these tests
// only exercise the read-only `board decisions` query, so they build the
// on-disk fixture the same direct way internal/decisions' own tests do,
// rather than satisfying the READY gate to move a real ticket through every
// status. decisionsBody is pasted verbatim as the `### Confirmed design
// decisions` subsection's body.
func writeDecisionTicket(t *testing.T, root, dir, id, project, decisionsBody string) {
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
		"### Confirmed design decisions (do not deviate without asking)\n\n" +
		decisionsBody + "\n\n" +
		"## Review\n\n## History\n\n- 2026-08-16 — created (TO DO). source: chat: test fixture\n"
	p := filepath.Join(root, "tickets", dir, id+"-"+slug+".md")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// boardDecisionsSandbox builds a two-child workspace (the child filter has no
// way to be exercised with only one) and seeds it with decisions across
// several statuses and both children, mirroring the six cases T-105's
// acceptance test enumerates.
func boardDecisionsSandbox(t *testing.T) (root string) {
	t.Helper()
	root = newProject(t) // registers child "demo" at "."
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Run(nil, "test", []string{"project", "add", "lib", "lib", "--ticket-prefix", "LIB"}); got != exitOK {
		t.Fatalf("project add lib = %d, want %d", got, exitOK)
	}

	// demo: a topic ("fail-open") recoverable across two done tickets, and one
	// unstructured (raw) decision to exercise the FORM column.
	writeDecisionTicket(t, root, "6-done", "T-001", "demo",
		"1. **The hook is fail-open by contract.** A broken guard must never block a legitimate commit.\n"+
			"2. Plain prose with no leading emphasis run.")
	writeDecisionTicket(t, root, "6-done", "T-002", "demo",
		"1. **The fail-open contract is inherited verbatim from T-001.** Nothing about it changes here.")
	// demo: TO DO holds a ticket with no decisions at all yet (written at
	// refinement, so a fresh TO DO ticket has none).
	writeDecisionTicket(t, root, "1-to-do", "T-003", "demo", "<!-- none yet -->")

	// lib: one decision, under its own prefix, to prove the child filter (and
	// prefix-agnostic ordering) both hold.
	writeDecisionTicket(t, root, "6-done", "LIB-001", "lib",
		"1. **Sprockets ship in the lib child only.** Unrelated to demo's contract.")

	return root
}

// TestBoardDecisionsTopicFilter is acceptance case 1: a topic filter recovers
// a contract chain across tickets, each row citable.
func TestBoardDecisionsTopicFilter(t *testing.T) {
	boardDecisionsSandbox(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "decisions", "--grep", "fail.open"})
	})
	if code != exitOK {
		t.Fatalf("board decisions --grep = %d, want %d:\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "T-001 decision 1") || !strings.Contains(out, "T-002 decision 1") {
		t.Errorf("expected both fail-open decisions cited, got:\n%s", out)
	}
	if strings.Contains(out, "LIB-001") {
		t.Errorf("grep matched an unrelated lib decision, got:\n%s", out)
	}
}

// TestBoardDecisionsTopicAndStatus is acceptance case 2: topic + --status
// narrows to shipped work only.
func TestBoardDecisionsTopicAndStatus(t *testing.T) {
	boardDecisionsSandbox(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "decisions", "--grep", "fail.open", "--status", "6-done"})
	})
	if code != exitOK {
		t.Fatalf("= %d, want %d:\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "2 decision(s)") {
		t.Errorf("expected exactly 2 matching decisions, got:\n%s", out)
	}
}

// TestBoardDecisionsToDoIsEmptyNotError is acceptance case 3: a TO DO ticket
// holds no decisions yet — 0 rows, exit 0, never an error.
func TestBoardDecisionsToDoIsEmptyNotError(t *testing.T) {
	boardDecisionsSandbox(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "decisions", "--status", "1-to-do"})
	})
	if code != exitOK {
		t.Fatalf("= %d, want %d:\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "0 decision(s) from 0 ticket(s)") {
		t.Errorf("expected the zero-result summary line, got:\n%s", out)
	}
}

// TestBoardDecisionsChildFilter is acceptance case 4: a multi-child
// workspace, filtered to one child, matches only that child's tickets — the
// one case T-105's own filed acceptance test could not exercise against this
// single-child repo, hence this Go test (T-105 decision: acceptance test
// needs no foreign workspace).
func TestBoardDecisionsChildFilter(t *testing.T) {
	boardDecisionsSandbox(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "decisions", "--project", "lib"})
	})
	if code != exitOK {
		t.Fatalf("= %d, want %d:\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "LIB-001 decision 1") {
		t.Errorf("expected LIB-001's decision, got:\n%s", out)
	}
	if strings.Contains(out, "T-001") || strings.Contains(out, "T-002") {
		t.Errorf("--project lib leaked a demo decision, got:\n%s", out)
	}
}

// TestBoardDecisionsNoMatchIsEmptyNotError is acceptance case 5: a filter
// combination matching nothing is exit 0 with an empty result.
func TestBoardDecisionsNoMatchIsEmptyNotError(t *testing.T) {
	boardDecisionsSandbox(t)
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "decisions", "--grep", "zzz-no-such-topic-zzz"})
	})
	if code != exitOK {
		t.Fatalf("= %d, want %d:\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "0 decision(s) from 0 ticket(s)") {
		t.Errorf("expected the zero-result summary line, got:\n%s", out)
	}
}

// TestBoardDecisionsErrors is acceptance case 6: an unregistered child, an
// unknown status directory, and an uncompilable regex are each an error
// (exit 1) — the prototype this ticket replaces leaked grep's exit 1 across
// legitimate empty results too, which is exactly what cases 3/5 above prove
// this command no longer does.
func TestBoardDecisionsErrors(t *testing.T) {
	boardDecisionsSandbox(t)
	cases := []struct {
		name string
		args []string
	}{
		{"unregistered project", []string{"board", "decisions", "--project", "no-such-child"}},
		{"unknown status directory", []string{"board", "decisions", "--status", "9-nowhere"}},
		{"uncompilable regex", []string{"board", "decisions", "--grep", "(["}},
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

// TestBoardDecisionsUnknownStatusNamesLegalValues confirms the error message
// for an unknown --status names the legal directories rather than just
// rejecting the value silently — never a hardcoded list (T-105 decision 10).
func TestBoardDecisionsUnknownStatusNamesLegalValues(t *testing.T) {
	boardDecisionsSandbox(t)
	errOut := captureStderr(t, func() {
		Run(nil, "test", []string{"board", "decisions", "--status", "9-nowhere"})
	})
	if !strings.Contains(errOut, "1-to-do") || !strings.Contains(errOut, "6-done") {
		t.Errorf("error does not name the legal status directories, got: %q", errOut)
	}
}

// TestBoardDecisionsJSON exercises the --json form: well-formed, versioned,
// and Decisions is `[]`, never `null`, on an empty result (T-105 decision 13
// / Task 4).
func TestBoardDecisionsJSON(t *testing.T) {
	boardDecisionsSandbox(t)

	var doc struct {
		Schema    int              `json:"schema"`
		Decisions []map[string]any `json:"decisions"`
		Filters   map[string]any   `json:"filters"`
	}
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "decisions", "--json"}); got != exitOK {
			t.Fatalf("board decisions --json = %d, want %d", got, exitOK)
		}
	})
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if doc.Schema != 1 {
		t.Errorf("schema = %d, want 1", doc.Schema)
	}
	if len(doc.Decisions) != 4 { // T-001 x2, T-002 x1, LIB-001 x1
		t.Errorf("decisions = %d, want 4:\n%s", len(doc.Decisions), out)
	}

	t.Run("empty result marshals decisions as [] not null", func(t *testing.T) {
		empty := captureStdout(t, func() {
			if got := Run(nil, "test", []string{"board", "decisions", "--grep", "zzz-no-such-topic-zzz", "--json"}); got != exitOK {
				t.Fatalf("= %d, want %d", got, exitOK)
			}
		})
		if strings.Contains(empty, `"decisions": null`) {
			t.Errorf("decisions marshaled as null, want []:\n%s", empty)
		}
		if !strings.Contains(empty, `"decisions": []`) {
			t.Errorf("expected an empty decisions array, got:\n%s", empty)
		}
	})
}

// TestBoardDecisionsFormColumn pins the FORM column's closed `bold`/`raw`
// vocabulary (T-105 decision 7/8), which cli-reference.adoc documents: a
// decision with a leading bold run renders `bold` and shows the projected
// statement, one without renders `raw` and shows its raw first line. Without
// this, swapping the two labels in renderDecisionsText would pass every other
// test in this file.
func TestBoardDecisionsFormColumn(t *testing.T) {
	boardDecisionsSandbox(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "decisions", "--project", "demo"}); got != exitOK {
			t.Fatalf("= %d, want %d", got, exitOK)
		}
	})
	var structured, unstructured string
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "T-001 decision 1"):
			structured = line
		case strings.HasPrefix(line, "T-001 decision 2"):
			unstructured = line
		}
	}
	if !strings.Contains(structured, "bold") ||
		!strings.Contains(structured, "The hook is fail-open by contract.") {
		t.Errorf("structured row = %q, want FORM \"bold\" and the projected statement", structured)
	}
	if !strings.Contains(unstructured, "raw") ||
		!strings.Contains(unstructured, "Plain prose with no leading emphasis run.") {
		t.Errorf("unstructured row = %q, want FORM \"raw\" and the raw first line", unstructured)
	}
}

// TestBoardDecisionsCitationIsPasteable pins the human-output contract: the
// first column is literally "<ID> decision <N>", with no extra decoration.
func TestBoardDecisionsCitationIsPasteable(t *testing.T) {
	boardDecisionsSandbox(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "decisions", "--project", "lib"}); got != exitOK {
			t.Fatalf("= %d", got)
		}
	})
	if !strings.Contains(out, "LIB-001 decision 1") {
		t.Errorf("citation not pasteable as \"LIB-001 decision 1\", got:\n%s", out)
	}
}
