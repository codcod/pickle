package ticket

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `---
id: T-001
title: a title with # hash
project: pickle
depends-on: [T-002, T-003]
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
