package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBoardStateRequiresJSONFlag pins T-065 confirmed decision 1: --json is
// mandatory, not a default. A bare `pickle board state` must print usage and
// exit non-zero without writing anything to stdout — never a silent dump of
// the whole document at a human exploring the CLI.
func TestBoardStateRequiresJSONFlag(t *testing.T) {
	newProject(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "state"}); got != exitUsage {
			t.Errorf("board state (no --json) = %d, want %d", got, exitUsage)
		}
	})
	if out != "" {
		t.Errorf("board state (no --json) wrote to stdout: %q, want nothing", out)
	}
}

func TestBoardStateUnknownFlag(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"board", "state", "--bogus"}); got != exitUsage {
		t.Errorf("board state --bogus = %d, want %d", got, exitUsage)
	}
}

// TestBoardStateFreshInstall pins acceptance check 11: a fresh install with
// zero tickets still emits a valid, versioned document.
func TestBoardStateFreshInstall(t *testing.T) {
	newProject(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "state", "--json"}); got != exitOK {
			t.Fatalf("board state --json = %d, want %d", got, exitOK)
		}
	})
	var doc struct {
		Schema  int    `json:"schema"`
		Tickets []any  `json:"tickets"`
		Flow    string `json:"flow"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if doc.Schema != 1 {
		t.Errorf("schema = %d, want 1", doc.Schema)
	}
	if len(doc.Tickets) != 0 {
		t.Errorf("tickets = %v, want none on a fresh install", doc.Tickets)
	}
	if doc.Flow == "" {
		t.Error("flow is empty")
	}
}

// TestBoardStateReflectsTicketNew pins that the projection is a live read,
// not a cached one: a ticket created after install shows up in the next
// `board state --json`.
func TestBoardStateReflectsTicketNew(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "a fixture ticket", "--project", "demo"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "state", "--json"}); got != exitOK {
			t.Fatalf("board state --json = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, `"id": "T-001"`) {
		t.Errorf("output does not contain the new ticket T-001:\n%s", out)
	}
}
