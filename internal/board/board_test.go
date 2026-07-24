package board

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleBoard = `# Board

## IN DEVELOPMENT

### pickle (1/1)

| id | title | branch | depends-on |
|---|---|---|---|
| T-002 | audit | feat/x | [T-001] |

## TO DO (impact order, per child)

### pickle

| id | title |
|---|---|
| T-NNN | template placeholder |
| T-003 | new |

### web

| id | title |
|---|---|
| T-050 | webby |

## Dependency chain

- T-001 → T-002
`

func writeBoard(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "BOARD.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAddTODORowImpactOrder(t *testing.T) {
	body := "# Board\n## TO DO\n### pickle\n\n| id | title | impact | complexity | cost | depends-on |\n|---|---|---|---|---|---|\n| T-004 | four | high | high | L | [] |\n| T-005 | five | medium | low | S | [] |\n\n## DONE\n"
	path := writeBoard(t, body)
	if err := AddTODORow(path, "pickle", "T-013", "new one", "high", "medium", "M"); err != nil {
		t.Fatal(err)
	}
	rows, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, r := range rows {
		if r.Status == "TO DO" {
			order = append(order, r.ID)
		}
	}
	// high T-013 must land after the other high (T-004), before medium T-005
	if strings.Join(order, ",") != "T-004,T-013,T-005" {
		t.Errorf("order = %v, want [T-004 T-013 T-005]", order)
	}
}

func TestAddTODORowCreatesSubgroup(t *testing.T) {
	body := "# Board\n## TO DO\n### pickle\n| id | t |\n|---|---|\n| T-004 | four |\n\n## DONE\n"
	path := writeBoard(t, body)
	if err := AddTODORow(path, "web", "T-050", "webby", "high", "medium", "M"); err != nil {
		t.Fatal(err)
	}
	rows, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range rows {
		if r.ID == "T-050" {
			found = true
			if r.Status != "TO DO" || r.Child != "web" {
				t.Errorf("T-050 landed at %+v, want TO DO/web", r)
			}
		}
	}
	if !found {
		t.Error("T-050 not found after creating sub-group")
	}
}

func TestRenderRowMatchesSection(t *testing.T) {
	got := RenderRow("DONE", RowData{ID: "T-001", Title: "x", Merged: "yes — merged (abc1234)"})
	if want := "| T-001 | x | yes — merged (abc1234) |"; got != want {
		t.Errorf("RenderRow = %q, want %q", got, want)
	}
	if RenderRow("NOPE", RowData{ID: "T-001"}) != "" {
		t.Error("RenderRow for unknown section should be empty")
	}
}

func TestParseCellsRoundTrip(t *testing.T) {
	body := "# Board\n\n## DONE\n\n### pickle\n\n| id | title | merged |\n|---|---|---|\n| T-001 | one | yes — merged (abc1234) |\n\n## DROPPED\n\n### pickle\n\n| id | title | reason |\n|---|---|---|\n| T-002 | two | superseded by T-003 |\n"
	path := writeBoard(t, body)
	cells, err := ParseCells(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cells["T-001"]["merged"]; got != "yes — merged (abc1234)" {
		t.Errorf("T-001 merged = %q", got)
	}
	if got := cells["T-002"]["reason"]; got != "superseded by T-003" {
		t.Errorf("T-002 reason = %q", got)
	}
	if got := cells["T-001"]["title"]; got != "one" {
		t.Errorf("T-001 title = %q", got)
	}
}

func TestParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "BOARD.md")
	if err := os.WriteFile(path, []byte(sampleBoard), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (placeholder + dependency-chain bullets excluded), got %d: %+v", len(rows), rows)
	}
	want := map[string]Row{
		"T-002": {Status: "IN DEVELOPMENT", Child: "pickle", ID: "T-002"},
		"T-003": {Status: "TO DO", Child: "pickle", ID: "T-003"},
		"T-050": {Status: "TO DO", Child: "web", ID: "T-050"},
	}
	for _, r := range rows {
		w, ok := want[r.ID]
		if !ok {
			t.Errorf("unexpected row %+v", r)
			continue
		}
		if r != w {
			t.Errorf("row %s = %+v, want %+v", r.ID, r, w)
		}
	}
}

// fullBoard has one row in IN DEVELOPMENT and the empty target sections a move
// would land rows into.
const fullBoard = `# Board

## IN DEVELOPMENT

### pickle (1/1)

| id | title | branch | depends-on |
|---|---|---|---|
| T-007 | mover | feat/T-007-x | [T-002] |

## IN REVIEW

### pickle (0/1)

| id | title | branch | depends-on |
|---|---|---|---|

## DONE

### pickle

| id | title | merged |
|---|---|---|

## DROPPED

### pickle

| id | title | reason |
|---|---|---|
`

func TestMoveRowRelocatesAndReshapes(t *testing.T) {
	path := writeBoard(t, fullBoard)
	// T-007 IN DEVELOPMENT -> IN REVIEW (same 4-column shape).
	if err := MoveRow(path, "IN REVIEW", "pickle", RowData{
		ID: "T-007", Title: "mover", Branch: "feat/T-007-x", DependsOn: "[T-002]",
	}); err != nil {
		t.Fatal(err)
	}
	rows, _ := Parse(path)
	got := map[string]Row{}
	for _, r := range rows {
		got[r.ID] = r
	}
	if len(rows) != 1 || got["T-007"].Status != "IN REVIEW" {
		t.Fatalf("after move: rows=%+v", rows)
	}

	// -> DONE (3-column shape: merged).
	if err := MoveRow(path, "DONE", "pickle", RowData{
		ID: "T-007", Title: "mover", Merged: "no — publish-gated",
	}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "| T-007 | mover | no — publish-gated |") {
		t.Errorf("DONE row not reshaped to merged column:\n%s", data)
	}
	rows, _ = Parse(path)
	if len(rows) != 1 || rows[0].Status != "DONE" {
		t.Errorf("expected single DONE row, got %+v", rows)
	}
}

func TestMoveRowIntoEmptySubgroupIsAdjacent(t *testing.T) {
	path := writeBoard(t, fullBoard)
	// IN REVIEW has an empty pickle sub-group (header + separator, then a blank).
	if err := MoveRow(path, "IN REVIEW", "pickle", RowData{
		ID: "T-007", Title: "mover", Branch: "feat/T-007-x", DependsOn: "[T-002]",
	}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	// The row must sit immediately after the separator line, and a blank line must
	// still separate the sub-group from the next "## " heading.
	if !strings.Contains(string(data), "|---|---|---|---|\n| T-007 | mover | feat/T-007-x | [T-002] |") {
		t.Errorf("row not adjacent to separator:\n%s", data)
	}
	if strings.Contains(string(data), "[T-002] |\n## REWORK") {
		t.Errorf("missing blank line before next heading:\n%s", data)
	}
}

func TestMoveRowCreatesMissingSubgroup(t *testing.T) {
	// DROPPED section exists but only for pickle; move a web ticket there.
	body := fullBoard + "" // reuse
	path := writeBoard(t, body)
	// First relocate T-007 into DROPPED under a NEW child "web".
	if err := MoveRow(path, "DROPPED", "web", RowData{
		ID: "T-007", Title: "mover", Reason: "obsolete",
	}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "### web") {
		t.Errorf("web sub-group not created:\n%s", data)
	}
	if !strings.Contains(string(data), "| T-007 | mover | obsolete |") {
		t.Errorf("DROPPED row missing/!reshaped:\n%s", data)
	}
}
