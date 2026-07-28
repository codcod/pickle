package ticket

import (
	"os"
	"path/filepath"
	"slices"
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

func TestValidID(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"T-1", true}, {"T-001", true}, {"T-9999", true},
		{"", false}, {"banana", false}, {"t-001", false}, {"T-", false},
		{"T-001x", false}, {"T-001]", false}, {"T-001\nimpact: critical", false},
		// ValidID deliberately does not trim: tokenizing is the caller's job.
		{" T-001 ", false},
	} {
		if got := ValidID(c.in); got != c.want {
			t.Errorf("ValidID(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseIDList(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"[]", nil},
		{"T-001,T-002", []string{"T-001", "T-002"}},
		{"[T-001, T-002]", []string{"T-001", "T-002"}},
		// Duplicates collapse, first-seen order survives (T-030 decision 3).
		{"T-001,T-001", []string{"T-001"}},
		{"T-002,T-001,T-002", []string{"T-002", "T-001"}},
		// Order is preserved, never sorted.
		{"[T-002, T-001]", []string{"T-002", "T-001"}},
		// Unbalanced brackets are accepted and normalised, not rejected: the
		// brackets are the frontmatter's own list syntax, not part of the id,
		// and ParseDepends strips them. Only what remains must be a valid id.
		{"T-001]", []string{"T-001"}},
		{"[T-001", []string{"T-001"}},
	} {
		got, err := ParseIDList(c.in)
		if err != nil {
			t.Errorf("ParseIDList(%q) returned error %v, want %v", c.in, err, c.want)
			continue
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("ParseIDList(%q) = %v, want %v", c.in, got, c.want)
		}
	}

	for _, c := range []struct{ in, wantSubstr string }{
		{"T-001,banana", "banana"},
		{"banana", "banana"},
		{"T-001]\nimpact: critical", "impact: critical"},
		{"t-001", "t-001"},
	} {
		got, err := ParseIDList(c.in)
		if err == nil {
			t.Errorf("ParseIDList(%q) = %v, want an error", c.in, got)
			continue
		}
		if got != nil {
			t.Errorf("ParseIDList(%q) returned %v alongside its error, want nil", c.in, got)
		}
		if !strings.Contains(err.Error(), c.wantSubstr) {
			t.Errorf("ParseIDList(%q) error = %q, want it to name %q", c.in, err, c.wantSubstr)
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
	// Two prefixes interleaved across status dirs, to prove NextNum counts each
	// prefix independently (per-child counters, T-058) rather than one global max.
	for _, name := range []string{
		"1-to-do/T-001-a.md", "6-done/T-012-b.md", "3-in-development/T-007-c.md",
		"1-to-do/RICK-001-d.md", "6-done/RICK-004-e.md", "2-ready/SB-002-f.md",
	} {
		p := filepath.Join(root, "tickets", filepath.Dir(name))
		os.MkdirAll(p, 0o755)
		os.WriteFile(filepath.Join(root, "tickets", name), []byte(sample), 0o644)
	}
	cases := map[string]int{"T": 13, "RICK": 5, "SB": 3, "PK": 1}
	for prefix, want := range cases {
		if got := NextNum(root, prefix); got != want {
			t.Errorf("NextNum(%q) = %d, want %d", prefix, got, want)
		}
	}
	if got := NextNum(t.TempDir(), "T"); got != 1 {
		t.Errorf("NextNum(empty) = %d, want 1", got)
	}
}

func TestSplitID(t *testing.T) {
	cases := []struct {
		id     string
		prefix string
		num    int
		ok     bool
	}{
		{"T-001", "T", 1, true},
		{"RICK-137", "RICK", 137, true},
		{"T-1", "T", 1, true},
		{"foo", "", 0, false},   // no '-'
		{"T-", "", 0, false},    // empty number
		{"T-0x1", "", 0, false}, // non-integer tail
	}
	for _, c := range cases {
		prefix, num, ok := SplitID(c.id)
		if prefix != c.prefix || num != c.num || ok != c.ok {
			t.Errorf("SplitID(%q) = (%q, %d, %v), want (%q, %d, %v)",
				c.id, prefix, num, ok, c.prefix, c.num, c.ok)
		}
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
	out := Scaffold("T-013", "Review the Jira board", "pickle", "high", "medium", "M", nil, "")
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
			out := Scaffold("T-013", "x", "pickle", "high", "medium", "M", c.in, "")
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

// TestScaffoldFamily: an empty family omits the line entirely (byte-identical to a
// no-family scaffold), while a set family renders on its own line immediately after
// spawned-by, and round-trips through LoadAll into Ticket.Family (T-059).
func TestScaffoldFamily(t *testing.T) {
	none := Scaffold("T-013", "x", "pickle", "high", "medium", "M", nil, "")
	if strings.Contains(none, "family:") {
		t.Errorf("empty family should emit no family line:\n%s", none)
	}

	withFam := Scaffold("T-013", "x", "pickle", "high", "medium", "M", nil, "T-001")
	if !strings.Contains(withFam, "spawned-by: []\nfamily: T-001\nimpact:") {
		t.Errorf("family line is not immediately after spawned-by:\n%s", withFam)
	}

	// round-trips into Ticket.Family via LoadAll
	root := t.TempDir()
	dir := filepath.Join(root, "tickets", "1-to-do")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T-013-x.md"), []byte(withFam), 0o644); err != nil {
		t.Fatal(err)
	}
	tickets, issues := LoadAll(root)
	if len(issues) != 0 {
		t.Fatalf("load issues: %v", issues)
	}
	if len(tickets) != 1 || tickets[0].Family != "T-001" {
		t.Errorf("Ticket.Family = %q, want T-001", tickets[0].Family)
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
	got := SectionHeadings(Scaffold("T-001", "x", "pickle", "high", "medium", "M", nil, ""))
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

// TestHistoryEntries covers the timeline's data source: every dated line, in file
// order, with created lines and merge notes kept (the view classifies, the parser
// does not filter).
func TestHistoryEntries(t *testing.T) {
	const doc = `# T-007 — x

## Description

- 2026-01-01 — this bullet is outside ## History and must be ignored

## History

- 2026-07-23 — created (TO DO). source: test
- 2026-07-24 - TO DO → READY: plain-hyphen separator
-   2026-07-25   —   READY → IN DEVELOPMENT: extra spaces
- not a dated bullet at all
- 2026-07-26 — merged to main (abc1234)
- 2026-07-27 — IN REVIEW → DONE: a reason so long that it wraps
  onto a second line
  and even a third
- 2026-07-28 — TO DO → READY: not a continuation of the above

## Review

- 2026-12-31 — a dated bullet after ## History ends must be ignored
`
	got := HistoryEntries(doc)
	want := []HistoryEntry{
		{"2026-07-23", "created (TO DO). source: test"},
		{"2026-07-24", "TO DO → READY: plain-hyphen separator"},
		{"2026-07-25", "READY → IN DEVELOPMENT: extra spaces"},
		{"2026-07-26", "merged to main (abc1234)"},
		// Wrapped entries fold back into one logical entry: truncating at the
		// first physical line would cut this reason mid-sentence.
		{"2026-07-27", "IN REVIEW → DONE: a reason so long that it wraps onto a second line and even a third"},
		{"2026-07-28", "TO DO → READY: not a continuation of the above"},
	}
	if len(got) != len(want) {
		t.Fatalf("HistoryEntries returned %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestHistoryEntriesEmpty(t *testing.T) {
	for _, doc := range []string{"", "# T-001 — x\n", "## History\n\n<!-- nothing yet -->\n"} {
		if got := HistoryEntries(doc); got != nil {
			t.Errorf("HistoryEntries(%q) = %+v, want nil", doc, got)
		}
	}
}

// TestHistoryEntriesDoesNotShiftPositionalCallers is the regression guard for the
// hazard HistoryEntries was written around: LastHistoryStatus, LastHistoryReason
// and MergeLine all read historyRE's body as m[1], so capturing the date would
// silently renumber it. Read the date and the body from the same lines and assert
// the three keep working.
func TestHistoryEntriesDoesNotShiftPositionalCallers(t *testing.T) {
	if got := LastHistoryStatus(sample); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE", got)
	}
	if got := LastHistoryReason(sample); got != "review PASS" {
		t.Errorf("LastHistoryReason = %q, want %q", got, "review PASS")
	}
	if got := MergeLine(sample); got != "MERGED: feat/T-001 → main (abc1234)" {
		t.Errorf("MergeLine = %q, want the merge note verbatim", got)
	}
	// The bodies HistoryEntries reports must be exactly what those helpers see.
	entries := HistoryEntries(sample)
	if len(entries) != 6 {
		t.Fatalf("HistoryEntries(sample) = %d entries, want 6", len(entries))
	}
	if entries[len(entries)-1].Text != MergeLine(sample) {
		t.Errorf("last entry text %q != MergeLine %q — body group moved",
			entries[len(entries)-1].Text, MergeLine(sample))
	}
	for _, e := range entries {
		if e.Date != "2026-07-23" {
			t.Errorf("entry date = %q, want 2026-07-23 (date slice misaligned)", e.Date)
		}
	}
}

// TestHistoryEntriesFoldingBoundaries pins what does *not* get folded into the
// preceding entry: a fresh bullet, a blank line, an HTML comment, and any
// unindented prose that follows the list.
func TestHistoryEntriesFoldingBoundaries(t *testing.T) {
	const doc = `## History

<!-- append-only; newest last -->

- 2026-07-23 — created (TO DO). source: test
  wrapped tail
- undated bullet must not fold
unindented prose must not fold

- 2026-07-24 — TO DO → READY: second entry
`
	got := HistoryEntries(doc)
	want := []HistoryEntry{
		{"2026-07-23", "created (TO DO). source: test wrapped tail"},
		{"2026-07-24", "TO DO → READY: second entry"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
