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
