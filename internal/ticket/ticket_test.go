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
		if got := historyKind(c.body); got != c.want {
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
	if got := LastHistoryStatus(sample); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE (merge line must be skipped)", got)
	}
	if !HasMergeLine(sample) {
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
	if got := LastHistoryStatus(doc); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE (reason's own arrow must not be read as the transition)", got)
	}
	want := "review PASS; 2 non-blocking → fixed inline (docs: a.adoc, b.adoc)"
	if got := LastHistoryReason(doc); got != want {
		t.Errorf("LastHistoryReason = %q, want %q (full reason, including its own arrow)", got, want)
	}
	const body = "IN REVIEW → DONE: review PASS; 2 non-blocking → fixed inline (docs: a.adoc, b.adoc)"
	if got := historyKind(body); got != HistoryTransition {
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
	if got := LastHistoryStatus(doc); got != "DONE" {
		t.Errorf("LastHistoryStatus = %q, want DONE", got)
	}
	want := "a long reason that wraps onto a continuation line"
	if got := LastHistoryReason(doc); got != want {
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
			if got := LastHistoryStatus(c.doc); got != c.wantStatus {
				t.Errorf("LastHistoryStatus = %q, want %q", got, c.wantStatus)
			}
			if got := LastHistoryReason(c.doc); got != c.wantReason {
				t.Errorf("LastHistoryReason = %q, want %q", got, c.wantReason)
			}
			// The invariant behind the fix, asserted directly: an entry classified
			// as a transition always names a legal target, and one that is not a
			// transition never names any — whatever folding did to its Text.
			for _, e := range HistoryEntries(c.doc) {
				if e.Kind == HistoryTransition {
					if _, ok := StatusByName(e.Target); !ok {
						t.Errorf("entry %+v is a transition whose Target is not a legal status", e)
					}
				} else if e.Target != "" {
					t.Errorf("entry %+v is not a transition but carries Target %q", e, e.Target)
				}
			}
		})
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
			if got := LastHistoryStatus(c.doc); got != c.want {
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

// TestTemplateAndScaffoldOutcomePlaceholdersAreFlagged is the second drift
// guard for T-083's check: both authoring entry points ship an Outcome
// placeholder, and OutcomeMissing must flag *both*, or the promise made by
// TEMPLATE.md, tickets-README.md §7, cli-reference.adoc and the CHANGELOG —
// "absent, empty, or still the template placeholder" — is false for whichever
// one drifts. It caught exactly that: a TEMPLATE.md placeholder written in the
// `<…>` form the other sections use, which the HTML-comment strip cannot see
// (review 1, finding B1).
func TestTemplateAndScaffoldOutcomePlaceholdersAreFlagged(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "skill", "resources", "TEMPLATE.md")
	data, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Skipf("TEMPLATE.md not found: %v", err)
	}
	if !OutcomeMissing(string(data)) {
		body, _ := SectionBody(string(data), "Outcome")
		t.Errorf("TEMPLATE.md's own ## Outcome placeholder is not flagged as missing; "+
			"write it as an HTML comment so the check sees through it. Body was:\n%s", body)
	}
	if !OutcomeMissing(Scaffold("T-001", "x", "pickle", "high", "medium", "M", nil, "")) {
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

func TestOutcomeMissing(t *testing.T) {
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
			if got := OutcomeMissing(c.text); got != c.want {
				t.Errorf("OutcomeMissing(%q) = %v, want %v", c.text, got, c.want)
			}
		})
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
	// Keyed fields, not positional: Target was added to HistoryEntry by T-043's
	// rework and a positional literal silently re-purposes the next field the
	// next time the struct grows. Asserting Target here also pins the invariant
	// the rework rests on — Kind and Target are decided together, from the
	// entry's first physical line, so a folded entry cannot claim to be a
	// transition to nowhere.
	want := []HistoryEntry{
		{Date: "2026-07-23", Text: "created (TO DO). source: test", Kind: HistoryCreated},
		{Date: "2026-07-24", Text: "TO DO → READY: plain-hyphen separator", Kind: HistoryTransition, Target: "READY"},
		{Date: "2026-07-25", Text: "READY → IN DEVELOPMENT: extra spaces", Kind: HistoryTransition, Target: "IN DEVELOPMENT"},
		{Date: "2026-07-26", Text: "merged to main (abc1234)", Kind: HistoryMerged},
		// Wrapped entries fold back into one logical entry: truncating at the
		// first physical line would cut this reason mid-sentence.
		{Date: "2026-07-27", Text: "IN REVIEW → DONE: a reason so long that it wraps onto a second line and even a third", Kind: HistoryTransition, Target: "DONE"},
		{Date: "2026-07-28", Text: "TO DO → READY: not a continuation of the above", Kind: HistoryTransition, Target: "READY"},
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
		{Date: "2026-07-23", Text: "created (TO DO). source: test wrapped tail", Kind: HistoryCreated},
		{Date: "2026-07-24", Text: "TO DO → READY: second entry", Kind: HistoryTransition, Target: "READY"},
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
