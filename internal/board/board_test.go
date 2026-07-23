package board

import (
	"os"
	"path/filepath"
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
