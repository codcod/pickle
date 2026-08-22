package ticket

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/flow"
)

// testDef is the flow definition every test in this package threads through
// the functions that need status vocabulary — brine, since that is what the
// fixtures below are all written against.
var testDef = flow.Default()

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

// TestParseFrontmatterDuplicateKeys: last-wins parse semantics are unchanged (a
// duplicate key still yields one value, the last one written), but the
// unexported scanner additionally reports which keys were duplicated — the
// record internal/audit reports at T-040. Each offending key is reported once,
// however many times it repeats.
func TestParseFrontmatterDuplicateKeys(t *testing.T) {
	const doc = `---
id: T-001
impact: low
impact: high
impact: critical
cost: M
---
`
	fm, dupes, ok := parseFrontmatter(doc)
	if !ok {
		t.Fatal("expected frontmatter")
	}
	if fm["impact"] != "critical" {
		t.Errorf("impact = %q, want last-wins value %q", fm["impact"], "critical")
	}
	if len(dupes) != 1 || dupes[0] != "impact" {
		t.Errorf("dupes = %v, want [impact] (reported once despite three occurrences)", dupes)
	}

	if _, dupes, _ := parseFrontmatter(sample); len(dupes) != 0 {
		t.Errorf("clean frontmatter reported dupes: %v", dupes)
	}
}

// TestHistoryKind pins the classifier every ## History reader now shares.
// Order matters: a legacy "MERGED: … → main" line contains an arrow and must
// classify as merged, not transition.
func TestHistoryKind(t *testing.T) {
	cases := []struct {
		body string
		want HistoryKind
	}{
		{"created (TO DO). source: chat", HistoryCreated},
		{"merged to main (abc1234)", HistoryMerged},
		{"MERGED: feat/T-001 → main (abc1234)", HistoryMerged},
		{"TO DO → READY: implementation plan complete", HistoryTransition},
		{"a free-form note with no arrow at all", HistoryNote},
		{"pickup applicability gate run → nowhere legal", HistoryNote},
	}
	for _, c := range cases {
		if got := historyKind(testDef, c.body); got != c.want {
			t.Errorf("historyKind(%q) = %q, want %q", c.body, got, c.want)
		}
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
	if got := LastHistoryStatus(testDef, sample); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE (merge line must be skipped)", got)
	}
	if !HasMergeLine(testDef, sample) {
		t.Error("HasMergeLine = false, want true")
	}
}

// TestLastHistoryStatusArrowInReason pins a real, live defect found in T-058's
// actual History (discovered during T-043, not by inspection): a reason clause
// containing its own "→" was read by LastIndex(body, "→") in preference to the
// transition's own arrow, so the whole entry silently reclassified as a note
// and "status" stuck at the previous transition. T-058 only kept auditing clean
// because a later, differently-worded duplicate DONE line happened to mask it.
// transitionTarget/splitTransition fix this by anchoring on the body's first
// colon — by this repo's own convention that colon always separates the
// transition from its reason — before ever searching for an arrow.
func TestLastHistoryStatusArrowInReason(t *testing.T) {
	const doc = `## History
- 2026-07-28 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-28 — IN REVIEW → DONE: review PASS; 2 non-blocking → fixed inline (docs: a.adoc, b.adoc)
`
	if got := LastHistoryStatus(testDef, doc); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE (reason's own arrow must not be read as the transition)", got)
	}
	want := "review PASS; 2 non-blocking → fixed inline (docs: a.adoc, b.adoc)"
	if got := LastHistoryReason(testDef, doc); got != want {
		t.Errorf("LastHistoryReason = %q, want %q (full reason, including its own arrow)", got, want)
	}
	const body = "IN REVIEW → DONE: review PASS; 2 non-blocking → fixed inline (docs: a.adoc, b.adoc)"
	if got := historyKind(testDef, body); got != HistoryTransition {
		t.Errorf("historyKind(%q) = %q, want %q", body, got, HistoryTransition)
	}
}

// TestLastHistoryReasonFoldsContinuations pins the divergence T-043 found: a
// transition's "OLD → NEW:" always sits on an entry's first physical line by
// this repo's own convention (HistoryEntry.Kind is frozen from exactly that
// line, deliberately, so an entry's classification cannot change as more of it
// is read), but a *long reason* routinely wraps onto a continuation line — this
// repo's own tickets do it constantly. LastHistoryReason used to read only the
// first physical line and silently truncate the reason at the wrap point;
// routing it through HistoryEntries's fold (like LastHistoryStatus) fixed that.
func TestLastHistoryReasonFoldsContinuations(t *testing.T) {
	const doc = `## History
- 2026-07-28 — IN REVIEW → DONE: a long reason that
  wraps onto a continuation line
`
	if got := LastHistoryStatus(testDef, doc); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE", got)
	}
	want := "a long reason that wraps onto a continuation line"
	if got := LastHistoryReason(testDef, doc); got != want {
		t.Errorf("LastHistoryReason = %q, want %q (full reason across the wrap, not truncated at the fold)", got, want)
	}
}

// TestTransitionSurvivesContinuationFolding is the T-043 review's blocking
// finding R1, both shapes. The fix that made a reason's own arrow stop hijacking
// the transition (see above) anchored the reason clause on the body's *first*
// colon and then re-derived the target from the entry's *folded* text — which
// broke two other shapes:
//
//   - a reason-less transition with a hand-written continuation line folds to
//     "TO DO → READY <prose>", whose candidate target is no longer a legal
//     status. HistoryEntry.Kind (frozen on the first physical line) still said
//     transition while the re-derivation resolved to nothing, so the entry was a
//     transition to nowhere and the status silently fell back to the entry above;
//   - "audit fix: TO DO → READY" carries a colon *before* the transition, so
//     splitting on the first colon left a head with no arrow in it at all.
//
// Both now resolve, and the first can no longer misresolve by construction:
// Kind and Target are decided together, from the same first physical line.
func TestTransitionSurvivesContinuationFolding(t *testing.T) {
	for _, c := range []struct {
		name, doc, wantStatus, wantReason string
	}{
		{
			name:       "reason-less transition with a continuation line",
			doc:        "## History\n- 2026-08-06 — created (TO DO). source: test\n- 2026-08-06 — TO DO → READY\n  a hand-written note wrapped under the transition\n",
			wantStatus: "READY",
			wantReason: "", // the entry carries no ": reason" clause at all
		},
		{
			name:       "a colon before the transition",
			doc:        "## History\n- 2026-08-06 — created (TO DO). source: test\n- 2026-08-06 — audit fix: TO DO → READY\n",
			wantStatus: "READY",
			wantReason: "",
		},
		{
			name:       "two arrows and no reason: the leftmost legal candidate wins",
			doc:        "## History\n- 2026-08-06 — IN DEVELOPMENT → IN REVIEW → DONE\n",
			wantStatus: "DONE",
			wantReason: "",
		},
		{
			name:       "a note mentioning an arrow and a status name stays a note",
			doc:        "## History\n- 2026-08-06 — TO DO → READY: refined\n- 2026-08-06 — clarified that a merge → DONE requires a human\n",
			wantStatus: "READY",
			wantReason: "refined",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := LastHistoryStatus(testDef, c.doc); got != c.wantStatus {
				t.Errorf("LastHistoryStatus = %q, want %q", got, c.wantStatus)
			}
			if got := LastHistoryReason(testDef, c.doc); got != c.wantReason {
				t.Errorf("LastHistoryReason = %q, want %q", got, c.wantReason)
			}
			// The invariant behind the fix, asserted directly: an entry classified
			// as a transition always names a legal target, and one that is not a
			// transition never names any — whatever folding did to its Text.
			for _, e := range HistoryEntries(testDef, c.doc) {
				if e.Kind == HistoryTransition {
					if _, ok := testDef.ByName(e.Target); !ok {
						t.Errorf("entry %+v is a transition whose Target is not a legal status", e)
					}
				} else if e.Target != "" {
					t.Errorf("entry %+v is not a transition but carries Target %q", e, e.Target)
				}
			}
		})
	}
}

// TestLastHistoryReasonAnchoredAtSameArrowAsTarget is the T-070 review's folded
// finding, T-043 review R7: before this fix, Target froze on the entry's first
// physical line while Reason was re-derived from the *folded* Text, so nothing
// checked the two came from the same arrow. A hand-written continuation line
// that itself contains "OLD → NEW: reason" used to hijack LastHistoryReason,
// even though LastHistoryStatus (reading Target) correctly ignored it. Now
// Reason is resolved in the same transitionParts call as Target, anchored at
// the same accepting arrow, and folding only ever appends raw continuation text
// onto an already-open clause — it never re-parses the continuation for arrows.
func TestLastHistoryReasonAnchoredAtSameArrowAsTarget(t *testing.T) {
	const doc = "## History\n- 2026-08-06 — TO DO → READY\n  and later IN REVIEW → DONE: some clause\n"
	if got := LastHistoryStatus(testDef, doc); got != "READY" {
		t.Errorf("LastHistoryStatus = %q, want READY (target frozen on the first physical line)", got)
	}
	if got := LastHistoryReason(testDef, doc); got != "" {
		t.Errorf("LastHistoryReason = %q, want \"\" (no clause was opened at the same arrow as Target; the continuation's own arrow+colon must not be read as this entry's reason)", got)
	}
}

// TestMergeLineFoldsContinuations pins the T-070 defect: MergeLine used to walk
// `## History` itself, reading only an entry's first physical line, so a merge
// line wrapped onto a continuation line was silently truncated. Routing it
// through HistoryEntries (like LastHistoryStatus/LastHistoryReason already are)
// folds it the same way every other reader does.
func TestMergeLineFoldsContinuations(t *testing.T) {
	const doc = "## History\n- 2026-08-06 — merged to main\n  (abc1234) after review\n"
	want := "merged to main (abc1234) after review"
	if got := MergeLine(testDef, doc); got != want {
		t.Errorf("MergeLine = %q, want %q (continuation folded, not truncated)", got, want)
	}

	// T-089's recommended form: a full commit URL pushes the line past this
	// tree's prose wrap width, which is the realistic case the re-grade cites.
	const wrappedURL = "## History\n" +
		"- 2026-08-10 — merged to main (MR !12,\n" +
		"  https://example.com/group/subgroup/project/-/merge_requests/12/diffs?commit_id=abc1234def5678901234567890abcdef1234567)\n"
	wantURL := "merged to main (MR !12, https://example.com/group/subgroup/project/-/merge_requests/12/diffs?commit_id=abc1234def5678901234567890abcdef1234567)"
	if got := MergeLine(testDef, wrappedURL); got != wantURL {
		t.Errorf("MergeLine = %q, want %q (wrapped commit URL folded in full)", got, wantURL)
	}
	if !HasMergeLine(testDef, wrappedURL) {
		t.Error("HasMergeLine = false, want true")
	}
}

// TestLastHistoryStatusUnexercisedShapes rounds out coverage the ticket found
// missing entirely: no History, an empty one, case-insensitive status matching,
// an unknown target ignored as a note rather than accepted, and a trailing note
// that must not overwrite the last real transition.
func TestLastHistoryStatusUnexercisedShapes(t *testing.T) {
	for _, c := range []struct {
		name, doc, want string
	}{
		{"no History section at all", "# T-001 — x\n\nno history here\n", ""},
		{"empty History section", "## History\n\n<!-- nothing yet -->\n", ""},
		{"lowercase target still matches", "## History\n- 2026-08-06 — to do → ready: refined\n", "READY"},
		{"mixed-case target still matches", "## History\n- 2026-08-06 — To Do → In Review: skipped a step\n", "IN REVIEW"},
		{"unknown target is a note, not a transition", "## History\n- 2026-08-06 — created (TO DO). source: test\n- 2026-08-06 — audit run → nowhere legal\n", "TO DO"},
		{"a trailing note does not overwrite the last transition", "## History\n- 2026-08-06 — TO DO → READY: refined\n- 2026-08-06 — a free-form note about the plan\n", "READY"},
		{"content under a different heading level is not History", "### History\n- 2026-08-06 — TO DO → READY: refined\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := LastHistoryStatus(testDef, c.doc); got != c.want {
				t.Errorf("LastHistoryStatus(%q) = %q, want %q", c.doc, got, c.want)
			}
		})
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
	tickets, issues := LoadAll(testDef, root)
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
		if got := NextNum(testDef, root, prefix); got != want {
			t.Errorf("NextNum(%q) = %d, want %d", prefix, got, want)
		}
	}
	if got := NextNum(testDef, t.TempDir(), "T"); got != 1 {
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
	if got := LastHistoryStatus(testDef, out); got != "TO DO" {
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
	tickets, issues := LoadAll(testDef, root)
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

// TestTemplateAndScaffoldOutcomePlaceholdersAreFlagged is the second drift
// guard for T-083's check: both authoring entry points ship an Outcome
// placeholder, and SectionMissing(text, "Outcome") must flag *both*, or the
// promise made by TEMPLATE.md, tickets-README.md §7, cli-reference.adoc and
// the CHANGELOG — "absent, empty, or still the template placeholder" — is
// false for whichever one drifts. It caught exactly that once: a TEMPLATE.md
// placeholder written in the `<…>` form the other sections use, which the
// HTML-comment strip cannot see (T-083 review 1, finding B1). Retargeted from
// the T-083-specific OutcomeMissing onto its T-081 generalisation.
func TestTemplateAndScaffoldOutcomePlaceholdersAreFlagged(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "skill", "resources", "TEMPLATE.md")
	data, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Skipf("TEMPLATE.md not found: %v", err)
	}
	if !SectionMissing(string(data), "Outcome") {
		body, _ := SectionBody(string(data), "Outcome")
		t.Errorf("TEMPLATE.md's own ## Outcome placeholder is not flagged as missing; "+
			"write it as an HTML comment so the check sees through it. Body was:\n%s", body)
	}
	if !SectionMissing(Scaffold("T-001", "x", "pickle", "high", "medium", "M", nil, ""), "Outcome") {
		t.Errorf("Scaffold's ## Outcome placeholder is not flagged as missing")
	}
}

func TestSectionBody(t *testing.T) {
	text := "# T-001 — x\n\n## Outcome\n\nAfter this ships, readers can tell at a glance.\n\n## Description\n\nsome spec\n"
	body, found := SectionBody(text, "Outcome")
	if !found || body != "After this ships, readers can tell at a glance." {
		t.Errorf("SectionBody(Outcome) = %q, %v", body, found)
	}
	body, found = SectionBody(text, "Description")
	if !found || body != "some spec" {
		t.Errorf("SectionBody(Description) = %q, %v", body, found)
	}
	_, found = SectionBody(text, "Review")
	if found {
		t.Errorf("SectionBody(Review) found = true, want false (no such section)")
	}

	// A section present but with an empty body (last section, EOF).
	empty := "## Description\n\n## Review\n\n"
	body, found = SectionBody(empty, "Review")
	if !found || body != "" {
		t.Errorf("SectionBody(Review) on trailing empty section = %q, %v, want \"\", true", body, found)
	}
}

func TestSectionMissing(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"absent", "## Description\n\nspec\n", true},
		{"placeholder comment only", "## Outcome\n\n<!-- TODO: 1-3 sentences -->\n\n## Description\n", true},
		{"whitespace only", "## Outcome\n\n   \n\n## Description\n", true},
		{"multiline placeholder comment", "## Outcome\n\n<!-- TODO:\nmulti-line\nplaceholder -->\n\n## Description\n", true},
		{"real prose", "## Outcome\n\nAfter this ships, X can Y.\n\n## Description\n", false},
		{"prose alongside a stale comment", "## Outcome\n\nAfter this ships, X can Y. <!-- note -->\n\n## Description\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SectionMissing(c.text, "Outcome"); got != c.want {
				t.Errorf("SectionMissing(%q, Outcome) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}

// TestNormalizeHeading is table-driven over every "### " heading form T-081's
// refinement measured across this repo's own 45 done tickets' Implementation
// Plans, plus the one form that legitimately does not reduce to a
// READY-gate stem ("### 4. Tests", for what rules §4.5 calls "Acceptance
// test") — normalizeHeading must not paper over a heading that genuinely
// uses different words.
func TestNormalizeHeading(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0. Feature branch (mandatory)", "feature branch"},
		{"Feature branch", "feature branch"},
		{"2. Confirmed decisions", "confirmed decisions"},
		{"Confirmed design decisions (do not deviate without asking)", "confirmed design decisions"},
		{"Docs", "docs"},
		{"Docs update (mandatory when user-facing)", "docs update"},
		{"6. Finish", "finish"},
		{"Finish (mandatory)", "finish"},
		{"Acceptance test (run verbatim; must be green before review)", "acceptance test"},
		{"Prerequisite gate (hard)", "prerequisite gate"},
		{"Tasks", "tasks"},
		{"4. Tests", "tests"}, // deliberately does NOT reduce to "acceptance test"
	}
	for _, c := range cases {
		if got := normalizeHeading(c.in); got != c.want {
			t.Errorf("normalizeHeading(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSubsectionBody(t *testing.T) {
	const text = `## Implementation Plan

### 0. Feature branch (mandatory)

Cut the branch.

### Prerequisite gate (hard)

<!-- TODO: fill in -->

### Confirmed design decisions (do not deviate without asking)

1. **First decision.** Rationale one.
2. **Second decision.** Rationale two.

### Tasks

#### Task 1

detail

## Review

findings
`
	cases := []struct {
		name          string
		section, stem string
		wantBody      string
		wantFound     bool
	}{
		{"absent parent section", "Nonexistent", "feature branch", "", false},
		{"absent stem", "Implementation Plan", "acceptance test", "", false},
		{"heading present, body is an HTML-comment placeholder", "Implementation Plan", "prerequisite",
			"<!-- TODO: fill in -->", true},
		{"heading present with prose", "Implementation Plan", "feature branch", "Cut the branch.", true},
		{"heading present with a multi-item body", "Implementation Plan", "confirmed",
			"1. **First decision.** Rationale one.\n2. **Second decision.** Rationale two.", true},
		{"heading bounded by the next ## (last ### in its parent section)", "Implementation Plan", "tasks",
			"#### Task 1\n\ndetail", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, found := SubsectionBody(text, c.section, c.stem)
			if found != c.wantFound {
				t.Errorf("SubsectionBody(text, %q, %q) found = %v, want %v", c.section, c.stem, found, c.wantFound)
			}
			if body != c.wantBody {
				t.Errorf("SubsectionBody(text, %q, %q) body = %q, want %q", c.section, c.stem, body, c.wantBody)
			}
		})
	}

	const lastHeading = `## Implementation Plan

### Confirmed design decisions (do not deviate without asking)

1. **Only decision.** The only rationale.
`
	t.Run("heading is the last thing in the file (bounded by EOF)", func(t *testing.T) {
		body, found := SubsectionBody(lastHeading, "Implementation Plan", "confirmed")
		if !found || body != "1. **Only decision.** The only rationale." {
			t.Errorf("SubsectionBody(lastHeading, ...) = %q, %v, want %q, true",
				body, found, "1. **Only decision.** The only rationale.")
		}
	})

	const emptyLast = `## Implementation Plan

### Confirmed design decisions (do not deviate without asking)
### Tasks

detail
`
	t.Run("### heading immediately followed by another ### (empty body)", func(t *testing.T) {
		body, found := SubsectionBody(emptyLast, "Implementation Plan", "confirmed")
		if !found || body != "" {
			t.Errorf("SubsectionBody(emptyLast, ...) = %q, %v, want \"\", true", body, found)
		}
	})
}

func TestSubsectionMissing(t *testing.T) {
	const text = `## Implementation Plan

### 0. Feature branch (mandatory)

Cut the branch.

### Prerequisite gate (hard)

<!-- TODO: fill in -->

### Tasks

#### Task 1

detail

## Review

findings
`
	cases := []struct {
		name          string
		section, stem string
		want          bool
	}{
		{"absent parent section", "Nonexistent", "feature branch", true},
		{"absent stem", "Implementation Plan", "acceptance test", true},
		{"heading present with empty body (placeholder comment)", "Implementation Plan", "prerequisite", true},
		{"heading present with prose", "Implementation Plan", "feature branch", false},
		{"a #### heading is not mistaken for a ### one", "Implementation Plan", "task 1", true},
		{"### Tasks itself has a substantive body (its #### child)", "Implementation Plan", "tasks", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SubsectionMissing(text, c.section, c.stem); got != c.want {
				t.Errorf("SubsectionMissing(text, %q, %q) = %v, want %v", c.section, c.stem, got, c.want)
			}
		})
	}
}

// TestGateViolationMessages pins both of GateViolation.Message's forms,
// including that the section-only form applied to brine's own Outcome row
// reproduces T-083's original warning text byte-for-byte — the promise T-081
// decision 6 makes explicit.
func TestGateViolationMessages(t *testing.T) {
	sectionForm := GateViolation{Req: flow.Requirement{
		Section: "Outcome", Label: "Outcome",
		Hint: "say what changes when this ships, in user-observable terms", Severity: flow.Advisory,
	}}
	wantSection := "## Outcome is missing, empty, or still a placeholder — say what changes when this ships, in user-observable terms"
	if got := sectionForm.Message(); got != wantSection {
		t.Errorf("section-form Message() = %q, want %q", got, wantSection)
	}
	if sectionForm.Blocking() {
		t.Error("Advisory violation reports Blocking() = true")
	}

	subForm := GateViolation{Req: flow.Requirement{
		Section: "Implementation Plan", Sub: "tasks", Label: "tasks",
		Hint: "list concrete tasks with exact paths in the target child (rules §4.4)", Severity: flow.Blocking,
	}}
	wantSub := `## Implementation Plan has no substantive "### tasks" heading (tasks) — ` +
		"list concrete tasks with exact paths in the target child (rules §4.4)"
	if got := subForm.Message(); got != wantSub {
		t.Errorf("sub-form Message() = %q, want %q", got, wantSub)
	}
	if !subForm.Blocking() {
		t.Error("Blocking violation reports Blocking() = false")
	}
}

func TestGateViolations(t *testing.T) {
	reqs := []flow.Requirement{
		{Section: "Outcome", Label: "Outcome", Hint: "h1", Severity: flow.Advisory},
		{Section: "Implementation Plan", Sub: "tasks", Label: "tasks", Hint: "h2", Severity: flow.Blocking},
	}
	// Neither requirement is met: both violations, in table order.
	violations := GateViolations(reqs, "## Description\n\nspec\n")
	if len(violations) != 2 {
		t.Fatalf("GateViolations() = %d violations, want 2", len(violations))
	}
	if violations[0].Req.Section != "Outcome" || violations[1].Req.Sub != "tasks" {
		t.Errorf("GateViolations() out of table order: %+v", violations)
	}

	// Both requirements met: no violations.
	satisfied := "## Outcome\n\nprose\n\n## Implementation Plan\n\n### Tasks\n\ndetail\n"
	if got := GateViolations(reqs, satisfied); len(got) != 0 {
		t.Errorf("GateViolations() on satisfied text = %+v, want none", got)
	}
}

// TestBrineReadyGateMatchesTemplate is T-081's drift guard between the code
// gate table and the authoring guide, mirroring internal/audit's
// TestFrontmatterKeysMatchTemplate: every Requirement.Sub stem brine's READY
// gate declares must match one of TEMPLATE.md's own "### " headings inside
// its "## Implementation Plan" skeleton (heading *vocabulary* only — the
// `<…>` placeholders make body substance meaningless here, T-081 decision
// 8), and each stem must already be in normalizeHeading's own output form.
// Lives in this package (not internal/flow, which cannot import
// normalizeHeading's normalisation contract without ceasing to be a leaf
// package) and not internal/audit (which cannot see normalizeHeading,
// unexported here).
func TestBrineReadyGateMatchesTemplate(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "skill", "resources", "TEMPLATE.md")
	data, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Skipf("TEMPLATE.md not found: %v", err)
	}
	planBody, found := SectionBody(string(data), "Implementation Plan")
	if !found {
		t.Fatal("TEMPLATE.md has no ## Implementation Plan section")
	}
	var tmplStems []string
	for _, line := range strings.Split(planBody, "\n") {
		if strings.HasPrefix(line, "### ") {
			tmplStems = append(tmplStems, normalizeHeading(line[4:]))
		}
	}

	def := flow.Default()
	ready, _ := def.ByDir("2-ready")
	var checked int
	for _, r := range def.Requirements(ready.Dir) {
		if r.Sub == "" {
			continue
		}
		checked++
		if normalizeHeading(r.Sub) != r.Sub {
			t.Errorf("Requirement.Sub %q is not itself in normalizeHeading's output form (got %q)",
				r.Sub, normalizeHeading(r.Sub))
		}
		if !slices.ContainsFunc(tmplStems, func(s string) bool { return strings.HasPrefix(s, r.Sub) }) {
			t.Errorf("Requirement.Sub %q matches no ### heading in TEMPLATE.md's Implementation Plan (stems: %v)",
				r.Sub, tmplStems)
		}
	}
	if checked != 7 {
		t.Fatalf("checked %d sub-heading requirements, want 7 (the READY-gate items)", checked)
	}
}

func TestLoadAllBadFilename(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tickets", "1-to-do")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "not-a-ticket.md"), []byte("x"), 0o644)
	_, issues := LoadAll(testDef, root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %v", issues)
	}
}

// TestStatusByToken's coverage now lives in internal/flow as
// TestByTokenForms/TestByDirAndByName (Definition.ByToken/ByDir/ByName,
// T-080) — the status vocabulary it exercised no longer lives in this
// package, and this package no longer defines the type those tests assert
// against.

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
	got := HistoryEntries(testDef, doc)
	// Keyed fields, not positional: Target was added to HistoryEntry by T-043's
	// rework and a positional literal silently re-purposes the next field the
	// next time the struct grows. Asserting Target here also pins the invariant
	// the rework rests on — Kind and Target are decided together, from the
	// entry's first physical line, so a folded entry cannot claim to be a
	// transition to nowhere.
	// Reason/reasonOpen were added by T-070 alongside Target: a transition's
	// reason clause is resolved in the same pass, from the same accepting arrow,
	// and folds forward exactly like Text.
	want := []HistoryEntry{
		{Date: "2026-07-23", Text: "created (TO DO). source: test", Kind: HistoryCreated},
		{Date: "2026-07-24", Text: "TO DO → READY: plain-hyphen separator", Kind: HistoryTransition, Target: "READY", Reason: "plain-hyphen separator", reasonOpen: true},
		{Date: "2026-07-25", Text: "READY → IN DEVELOPMENT: extra spaces", Kind: HistoryTransition, Target: "IN DEVELOPMENT", Reason: "extra spaces", reasonOpen: true},
		{Date: "2026-07-26", Text: "merged to main (abc1234)", Kind: HistoryMerged},
		// Wrapped entries fold back into one logical entry: truncating at the
		// first physical line would cut this reason mid-sentence.
		{Date: "2026-07-27", Text: "IN REVIEW → DONE: a reason so long that it wraps onto a second line and even a third", Kind: HistoryTransition, Target: "DONE", Reason: "a reason so long that it wraps onto a second line and even a third", reasonOpen: true},
		{Date: "2026-07-28", Text: "TO DO → READY: not a continuation of the above", Kind: HistoryTransition, Target: "READY", Reason: "not a continuation of the above", reasonOpen: true},
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
		if got := HistoryEntries(testDef, doc); got != nil {
			t.Errorf("HistoryEntries(%q) = %+v, want nil", doc, got)
		}
	}
}

// TestHistoryEntriesDoesNotShiftPositionalCallers is the regression guard for the
// hazard HistoryEntries was written around: it reads historyRE's body as m[1], so
// capturing the date would silently renumber it and every reader that resolves
// through it — LastHistoryStatus, LastHistoryReason and, since T-070, MergeLine
// — would read the wrong group. Read the date and the body from the same lines
// and assert the three keep working.
func TestHistoryEntriesDoesNotShiftPositionalCallers(t *testing.T) {
	if got := LastHistoryStatus(testDef, sample); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE", got)
	}
	if got := LastHistoryReason(testDef, sample); got != "review PASS" {
		t.Errorf("LastHistoryReason = %q, want %q", got, "review PASS")
	}
	if got := MergeLine(testDef, sample); got != "MERGED: feat/T-001 → main (abc1234)" {
		t.Errorf("MergeLine = %q, want the merge note verbatim", got)
	}
	// The bodies HistoryEntries reports must be exactly what those helpers see.
	entries := HistoryEntries(testDef, sample)
	if len(entries) != 6 {
		t.Fatalf("HistoryEntries(sample) = %d entries, want 6", len(entries))
	}
	if entries[len(entries)-1].Text != MergeLine(testDef, sample) {
		t.Errorf("last entry text %q != MergeLine %q — body group moved",
			entries[len(entries)-1].Text, MergeLine(testDef, sample))
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
	got := HistoryEntries(testDef, doc)
	want := []HistoryEntry{
		{Date: "2026-07-23", Text: "created (TO DO). source: test wrapped tail", Kind: HistoryCreated},
		{Date: "2026-07-24", Text: "TO DO → READY: second entry", Kind: HistoryTransition, Target: "READY", Reason: "second entry", reasonOpen: true},
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
