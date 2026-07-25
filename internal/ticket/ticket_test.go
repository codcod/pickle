package ticket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `---
id: T-001
title: a title with # hash
project: pickle
depends-on: [T-002, T-003]
spawned-by: [T-004]
impact: high
complexity: medium
cost: M
---

# T-001 — a title

## History
- 2026-07-23 — created (TO DO). source: test
- 2026-07-23 — TO DO → READY: refined
- 2026-07-23 — READY → IN DEVELOPMENT: picked up
- 2026-07-23 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-23 — IN REVIEW → DONE: review PASS
- 2026-07-23 — MERGED: feat/T-001 → main (abc1234)
`

func TestParseFrontmatter(t *testing.T) {
	fm, ok := ParseFrontmatter(sample)
	if !ok {
		t.Fatal("expected frontmatter")
	}
	if fm["title"] != "a title with # hash" {
		t.Errorf("title mangled: %q", fm["title"])
	}
	if fm["project"] != "pickle" || fm["impact"] != "high" {
		t.Errorf("fm = %v", fm)
	}
	if _, ok := ParseFrontmatter("no frontmatter here"); ok {
		t.Error("expected ok=false for missing frontmatter")
	}
}

func TestParseDepends(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{{"[]", 0}, {"", 0}, {"[T-001]", 1}, {"[T-001, T-002]", 2}} {
		if got := ParseDepends(c.in); len(got) != c.want {
			t.Errorf("ParseDepends(%q) = %v", c.in, got)
		}
	}
}

func TestLastHistoryStatusSkipsMerge(t *testing.T) {
	if got := LastHistoryStatus(sample); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE (merge line must be skipped)", got)
	}
	if !HasMergeLine(sample) {
		t.Error("HasMergeLine = false, want true")
	}
}

func TestLoadAllToleratesMissingDirs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tickets", "1-to-do")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T-001-foo.md"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	// only 1-to-do exists; the other status dirs are absent (vanished-empty)
	tickets, issues := LoadAll(root)
	if len(issues) != 0 {
		t.Fatalf("unexpected load issues: %v", issues)
	}
	if len(tickets) != 1 || tickets[0].ID != "T-001" || tickets[0].Slug != "foo" {
		t.Fatalf("tickets = %+v", tickets)
	}
	if tickets[0].Base() != "T-001-foo" {
		t.Errorf("Base = %q", tickets[0].Base())
	}
	if got := tickets[0].DependsOn; len(got) != 2 || got[0] != "T-002" {
		t.Errorf("DependsOn = %v, want [T-002 T-003]", got)
	}
	// lineage is parsed with the same parser but kept in its own field, so the
	// two relationships can never be confused for one another
	if got := tickets[0].SpawnedBy; len(got) != 1 || got[0] != "T-004" {
		t.Errorf("SpawnedBy = %v, want [T-004]", got)
	}
}

func TestSlugify(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Review the Jira board", "review-the-jira-board"},
		{"  Trim & Punctuation!!  ", "trim-punctuation"},
		{"MixedCASE_123", "mixedcase-123"},
		{"###", "untitled"},
	} {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNextNum(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"1-to-do/T-001-a.md", "6-done/T-012-b.md", "3-in-development/T-007-c.md"} {
		p := filepath.Join(root, "tickets", filepath.Dir(name))
		os.MkdirAll(p, 0o755)
		os.WriteFile(filepath.Join(root, "tickets", name), []byte(sample), 0o644)
	}
	if got := NextNum(root); got != 13 {
		t.Errorf("NextNum = %d, want 13", got)
	}
	if got := NextNum(t.TempDir()); got != 1 {
		t.Errorf("NextNum(empty) = %d, want 1", got)
	}
}

func TestValidGrade(t *testing.T) {
	if !ValidGrade("impact", "medium-high") || !ValidGrade("cost", "M") || !ValidGrade("complexity", "low") {
		t.Error("expected legal grades to validate")
	}
	if ValidGrade("impact", "banana") || ValidGrade("cost", "XXL") || ValidGrade("nope", "medium") {
		t.Error("expected illegal grades to fail")
	}
}

func TestScaffoldIsAuditClean(t *testing.T) {
	out := Scaffold("T-013", "Review the Jira board", "pickle", "high", "medium", "M", nil)
	fm, ok := ParseFrontmatter(out)
	if !ok {
		t.Fatal("scaffold has no frontmatter")
	}
	for _, k := range []string{"id", "title", "project", "depends-on", "spawned-by", "impact", "complexity", "cost"} {
		if _, has := fm[k]; !has {
			t.Errorf("scaffold frontmatter missing %q", k)
		}
	}
	if fm["id"] != "T-013" || fm["project"] != "pickle" {
		t.Errorf("scaffold frontmatter = %v", fm)
	}
	if got := LastHistoryStatus(out); got != "TO DO" {
		t.Errorf("scaffold LastHistoryStatus = %q, want TO DO", got)
	}
}

func TestScaffoldRendersSpawnedBy(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []string
		want string
	}{
		{"none", nil, "[]"},
		{"empty slice", []string{}, "[]"},
		{"one", []string{"T-001"}, "[T-001]"},
		{"two", []string{"T-001", "T-002"}, "[T-001, T-002]"},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := Scaffold("T-013", "x", "pickle", "high", "medium", "M", c.in)
			fm, ok := ParseFrontmatter(out)
			if !ok {
				t.Fatal("scaffold has no frontmatter")
			}
			if fm["spawned-by"] != c.want {
				t.Errorf("spawned-by = %q, want %q", fm["spawned-by"], c.want)
			}
			// round-trips through the parser the audit uses
			if got := ParseDepends(fm["spawned-by"]); len(got) != len(c.in) {
				t.Errorf("ParseDepends(%q) = %v, want %d ids", fm["spawned-by"], got, len(c.in))
			}
			// decision 7: the pair reads together, lineage directly under deps
			if !strings.Contains(out, "depends-on: []\nspawned-by: "+c.want+"\n") {
				t.Errorf("spawned-by is not immediately after depends-on:\n%s", out)
			}
		})
	}
}

// TestScaffoldSectionsMatchTemplate is the drift guard (T-003 decision 4): the
// scaffold's ## section set must equal the embedded TEMPLATE.md's.
func TestScaffoldSectionsMatchTemplate(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "skill", "resources", "TEMPLATE.md")
	data, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Skipf("TEMPLATE.md not found: %v", err)
	}
	want := SectionHeadings(string(data))
	got := SectionHeadings(Scaffold("T-001", "x", "pickle", "high", "medium", "M", nil))
	if len(want) != len(got) {
		t.Fatalf("section count differs: template %v vs scaffold %v", want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("section %d: template %q != scaffold %q", i, want[i], got[i])
		}
	}
}

func TestLoadAllBadFilename(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tickets", "1-to-do")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "not-a-ticket.md"), []byte("x"), 0o644)
	_, issues := LoadAll(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %v", issues)
	}
}

func TestStatusByToken(t *testing.T) {
	cases := []struct {
		tok string
		dir string
		ok  bool
	}{
		{"3-in-development", "3-in-development", true}, // dir name
		{"in-development", "3-in-development", true},   // dir minus number
		{"IN DEVELOPMENT", "3-in-development", true},   // display name (spaces)
		{"In-Development", "3-in-development", true},   // case-insensitive
		{"to-do", "1-to-do", true},
		{"1-to-do", "1-to-do", true},
		{"done", "6-done", true},
		{"dropped", "7-dropped", true},
		{"nonsense", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := StatusByToken(tc.tok)
		if ok != tc.ok {
			t.Errorf("StatusByToken(%q) ok=%v, want %v", tc.tok, ok, tc.ok)
			continue
		}
		if ok && got.Dir != tc.dir {
			t.Errorf("StatusByToken(%q) = %q, want %q", tc.tok, got.Dir, tc.dir)
		}
	}
}
