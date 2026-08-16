package state

import "testing"

// wrap builds a minimal ticket text whose ## Review section is body, so
// parseReview can be exercised directly against just the section content.
func wrap(body string) string {
	return "# T-999 — fixture\n\n## Review\n\n" + body + "\n\n## History\n- 2026-01-01 — created (TO DO). source: test\n"
}

func TestParseReviewCanonicalHeader(t *testing.T) {
	body := `| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | **blocking** | correctness | — | wrong output | file.go:12 | fix it |
| F2 | non-blocking | docs-gap | noted | missing docs | README.md | add a section |

**Disposition summary:** 1 non-blocking, 0 fixed inline.
cost: estimated M, actual M`

	rev := parseReview(wrap(body))
	if rev.Tables != 1 {
		t.Fatalf("Tables = %d, want 1", rev.Tables)
	}
	want := []Finding{
		{ID: "f1", Severity: "blocking", Class: "correctness", Disposition: "—"},
		{ID: "f2", Severity: "non-blocking", Class: "docs-gap", Disposition: "noted"},
	}
	if len(rev.Findings) != len(want) {
		t.Fatalf("Findings = %+v, want %+v", rev.Findings, want)
	}
	for i, f := range want {
		if rev.Findings[i] != f {
			t.Errorf("Findings[%d] = %+v, want %+v", i, rev.Findings[i], f)
		}
	}
	if rev.DispositionSummary == "" {
		t.Error("DispositionSummary not captured")
	}
	if rev.CostLine != "cost: estimated M, actual M" {
		t.Errorf("CostLine = %q", rev.CostLine)
	}
}

// TestParseReviewSixColumnHeader covers the 33-ticket majority variant that
// predates the class column entirely (id|severity|disposition|description|
// evidence|suggestion) — Class must resolve to "", never a value shifted in
// from a neighbouring column.
func TestParseReviewSixColumnHeader(t *testing.T) {
	body := `| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | fixed inline | a small thing | evidence here | do this |`

	rev := parseReview(wrap(body))
	if len(rev.Findings) != 1 {
		t.Fatalf("Findings = %+v, want 1 row", rev.Findings)
	}
	f := rev.Findings[0]
	if f.ID != "n1" || f.Severity != "non-blocking" || f.Disposition != "fixed inline" {
		t.Errorf("Findings[0] = %+v", f)
	}
	if f.Class != "" {
		t.Errorf("Class = %q, want empty (no class column in this header)", f.Class)
	}
}

// TestParseReviewFindingAliasColumn covers the header variant that uses
// "finding" instead of "description" as its prose column name — irrelevant
// to this parser (that column is never projected), but the id/severity/
// disposition columns on either side of it must still resolve correctly.
func TestParseReviewFindingAliasColumn(t *testing.T) {
	body := `| id | severity | disposition | finding | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | noted | something happened | proof | do X |`

	rev := parseReview(wrap(body))
	if len(rev.Findings) != 1 {
		t.Fatalf("Findings = %+v", rev.Findings)
	}
	if got := rev.Findings[0]; got.ID != "f1" || got.Severity != "non-blocking" || got.Disposition != "noted" {
		t.Errorf("Findings[0] = %+v", got)
	}
}

// TestParseReviewHashLedHeader covers the two variants led by "#" instead of
// "id", including one where severity/disposition are reordered relative to
// the canonical layout — the parser must resolve every field by column name,
// never by position.
func TestParseReviewHashLedHeader(t *testing.T) {
	body := `| # | finding | evidence | severity | disposition |
|---|---|---|---|---|
| 1 | it broke | log.txt | blocking | — |`

	rev := parseReview(wrap(body))
	if len(rev.Findings) != 1 {
		t.Fatalf("Findings = %+v", rev.Findings)
	}
	got := rev.Findings[0]
	if got.ID != "1" {
		t.Errorf("ID = %q, want \"1\" (# aliased to id)", got.ID)
	}
	if got.Severity != "blocking" || got.Disposition != "—" {
		t.Errorf("Findings[0] = %+v, want severity=blocking disposition=—", got)
	}
}

// TestParseReviewNearMissNotDetected pins the detection rule (decision 10):
// a table whose 4th column is literally "severity before → after" is not a
// findings table — it has no column named exactly "severity" or
// "disposition" — and must contribute zero rows.
func TestParseReviewNearMissNotDetected(t *testing.T) {
	body := `| check | old trigger | new trigger | severity before → after |
|---|---|---|---|
| gate 1 | warning | error | warning → error |`

	rev := parseReview(wrap(body))
	if rev.Tables != 0 {
		t.Errorf("Tables = %d, want 0 (near-miss header must not be detected)", rev.Tables)
	}
	if len(rev.Findings) != 0 {
		t.Errorf("Findings = %+v, want none", rev.Findings)
	}
}

// TestParseReviewMultipleTables covers a re-review round: two findings
// tables in one ## Review section, separated by prose/headings, both
// contributing rows (decision 12 — every table counts).
func TestParseReviewMultipleTables(t *testing.T) {
	body := `### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | **blocking** | — | first round | e1 | fix |

### Rework pass — 2026-08-06

### Scoped re-review

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | noted | fixed and re-checked | e2 | — |`

	rev := parseReview(wrap(body))
	if rev.Tables != 2 {
		t.Fatalf("Tables = %d, want 2", rev.Tables)
	}
	if len(rev.Findings) != 2 {
		t.Fatalf("Findings = %+v, want 2 rows total", rev.Findings)
	}
	if rev.Findings[0].Severity != "blocking" || rev.Findings[1].Severity != "non-blocking" {
		t.Errorf("Findings = %+v", rev.Findings)
	}
}

// TestParseReviewEscapedPipe pins that a literal "\|" inside a prose column
// is never read as a column boundary — a real corpus case (e.g. "project
// add\|list\|remove" inside an evidence cell) that would otherwise shift a
// later column's index.
func TestParseReviewEscapedPipe(t *testing.T) {
	body := `| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | noted | uses project add\|list\|remove | e1 | do X |`

	rev := parseReview(wrap(body))
	if len(rev.Findings) != 1 {
		t.Fatalf("Findings = %+v, want exactly 1 row (escaped pipe must not split a column)", rev.Findings)
	}
	if got := rev.Findings[0]; got.ID != "f1" || got.Severity != "non-blocking" || got.Disposition != "noted" {
		t.Errorf("Findings[0] = %+v, escaped pipe corrupted column alignment", got)
	}
}

func TestParseReviewNoSection(t *testing.T) {
	rev := parseReview("# T-999 — fixture\n\nno review section here\n")
	if rev.Tables != 0 || len(rev.Findings) != 0 {
		t.Errorf("rev = %+v, want zero value", rev)
	}
	if rev.Headers == nil || rev.Findings == nil {
		t.Error("Headers/Findings must be non-nil empty slices, never nil (so they marshal to [] not null)")
	}
}
