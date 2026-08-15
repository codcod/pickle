package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

var testDef = flow.Default()

func testCfg() *config.Config {
	return &config.Config{Projects: []config.Project{
		{Name: "demo", Path: ".", WIPInDevelopment: 1, WIPInReview: 1},
	}}
}

// buildFixture is one ticket file to write into a test tree.
type buildFixture struct {
	dir, id, title, impact string
	review                 string // ## Review body, "" for the TEMPLATE.md placeholder
}

func (f buildFixture) text() string {
	review := f.review
	if review == "" {
		review = "<!-- empty until IN REVIEW -->"
	}
	return fmt.Sprintf(`---
id: %s
title: %s
project: demo
depends-on: []
spawned-by: []
impact: %s
complexity: medium
cost: M
---

# %s — %s

## Outcome

after this ships, something changes.

## Description

test fixture.

## Implementation Plan

### 0. Feature branch (mandatory)

branch

### Prerequisite gate (hard)

none

### Confirmed design decisions (do not deviate without asking)

d1

### Tasks

t1

### Acceptance test

just test

### Docs update (mandatory when user-facing)

no user-facing surface

### Finish (mandatory)

summary

## Review

%s

## History

- 2026-07-20 — created (TO DO). source: test
`, f.id, f.title, f.impact, f.id, f.title, review)
}

// newBuildTree writes a fixture tickets/ tree (every status dir + .gitkeep,
// per fixture file) plus a freshly rendered BOARD.md, so Health.BoardDrift
// reads "none" by default in every test that doesn't deliberately stale it.
func newBuildTree(t *testing.T, fixtures ...buildFixture) string {
	t.Helper()
	root := t.TempDir()
	for _, s := range testDef.States() {
		dir := filepath.Join(root, "tickets", s.Dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range fixtures {
		p := filepath.Join(root, "tickets", f.dir, f.id+"-slug.md")
		if err := os.WriteFile(p, []byte(f.text()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tickets, issues := ticket.LoadAll(testDef, root)
	if len(issues) > 0 {
		t.Fatalf("fixture load issues: %v", issues)
	}
	text := board.Render(testDef, tickets, testCfg(), time.Now().Format("2006-01-02"))
	if err := os.WriteFile(filepath.Join(root, "tickets", "BOARD.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestBuildDeterministic(t *testing.T) {
	root := newBuildTree(t,
		buildFixture{dir: "1-to-do", id: "T-001", title: "alpha", impact: "low"},
		buildFixture{dir: "1-to-do", id: "T-002", title: "beta", impact: "high"},
	)
	doc1, err := Build(testDef, root, testCfg(), "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := Build(testDef, root, testCfg(), "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := json.Marshal(doc1)
	b2, _ := json.Marshal(doc2)
	if string(b1) != string(b2) {
		t.Errorf("two builds against an unchanged tree differ:\n%s\nvs\n%s", b1, b2)
	}
}

func TestBuildOrderMatchesBoardSort(t *testing.T) {
	root := newBuildTree(t,
		buildFixture{dir: "1-to-do", id: "T-001", title: "low impact", impact: "low"},
		buildFixture{dir: "1-to-do", id: "T-002", title: "critical impact", impact: "critical"},
		buildFixture{dir: "1-to-do", id: "T-003", title: "medium impact", impact: "medium"},
	)
	doc, err := Build(testDef, root, testCfg(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tk := range doc.Tickets {
		if tk.Status == "TO DO" {
			got = append(got, tk.ID)
		}
	}
	want := []string{"T-002", "T-003", "T-001"} // impact descending: critical, medium, low
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("order = %v, want %v (board.Sort's impact-descending rule)", got, want)
		}
	}
}

func TestBuildStatesFromFlowDefinition(t *testing.T) {
	root := newBuildTree(t)
	doc, err := Build(testDef, root, testCfg(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.States) != len(testDef.BoardStates()) {
		t.Fatalf("States = %d entries, want %d (one per def.BoardStates())", len(doc.States), len(testDef.BoardStates()))
	}
	for i, st := range testDef.BoardStates() {
		if doc.States[i].Name != st.Name || doc.States[i].Dir != st.Dir {
			t.Errorf("States[%d] = %+v, want name=%s dir=%s", i, doc.States[i], st.Name, st.Dir)
		}
	}
}

func TestBuildHealthClean(t *testing.T) {
	root := newBuildTree(t,
		buildFixture{dir: "1-to-do", id: "T-001", title: "alpha", impact: "low"},
	)
	doc, err := Build(testDef, root, testCfg(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Health.BoardDrift != "none" {
		t.Errorf("BoardDrift = %q, want \"none\" (BOARD.md was freshly rendered)", doc.Health.BoardDrift)
	}
	if len(doc.Health.Errors) != 0 {
		t.Errorf("Errors = %v, want none", doc.Health.Errors)
	}
}

func TestBuildHealthDriftAfterEdit(t *testing.T) {
	root := newBuildTree(t,
		buildFixture{dir: "1-to-do", id: "T-001", title: "alpha", impact: "low"},
	)
	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	data, err := os.ReadFile(boardPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(boardPath, append(data, []byte("\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := Build(testDef, root, testCfg(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Health.BoardDrift == "none" {
		t.Error("BoardDrift = \"none\" after appending to BOARD.md, want \"layout\" or \"rows\"")
	}
}

func TestBuildReviewProjection(t *testing.T) {
	root := newBuildTree(t,
		buildFixture{dir: "6-done", id: "T-001", title: "done ticket", impact: "low", review: `| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | test-gap | noted | a gap | ev | fix later |

**Disposition summary:** 1 non-blocking, 1 noted.
cost: estimated S, actual S`},
	)
	doc, err := Build(testDef, root, testCfg(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	var found *Ticket
	for i := range doc.Tickets {
		if doc.Tickets[i].ID == "T-001" {
			found = &doc.Tickets[i]
		}
	}
	if found == nil {
		t.Fatal("T-001 not found in projection")
	}
	if len(found.Review.Findings) != 1 {
		t.Fatalf("Review.Findings = %+v, want 1", found.Review.Findings)
	}
	f := found.Review.Findings[0]
	if f.ID != "f1" || f.Severity != "non-blocking" || f.Class != "test-gap" || f.Disposition != "noted" {
		t.Errorf("Findings[0] = %+v", f)
	}
	if found.Review.DispositionSummary == "" {
		t.Error("DispositionSummary not projected")
	}
	if found.Review.CostLine != "cost: estimated S, actual S" {
		t.Errorf("CostLine = %q", found.Review.CostLine)
	}
}

func TestBuildEmptyTreeFieldsAreEmptySlicesNotNull(t *testing.T) {
	root := newBuildTree(t)
	doc, err := Build(testDef, root, testCfg(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if want := `"tickets":[]`; !strings.Contains(s, want) {
		t.Errorf("marshalled doc missing %q (tickets should be [] on an empty tree, not null): %s", want, s)
	}
}
