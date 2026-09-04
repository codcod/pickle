package ticket

import (
	"strings"
	"testing"
)

// wellFormed is a fixture whose frontmatter title and H1 agree exactly (the
// precondition SetField's title path requires) and whose body has enough
// unrelated lines (History, a Description-shaped paragraph) to prove a
// field/title edit never reaches outside frontmatter+H1.
const wellFormed = `---
id: T-050
title: a well formed ticket
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-050 — a well formed ticket

## Description

Nothing special about this one.

## History
- 2026-07-23 — created (TO DO). source: test
`

func TestSetFieldGrades(t *testing.T) {
	for _, field := range []string{"impact", "complexity", "cost"} {
		t.Run(field, func(t *testing.T) {
			got, err := SetField(wellFormed, "T-050", field, "high")
			if err != nil {
				t.Fatalf("SetField(%s): %v", field, err)
			}
			fm, ok := ParseFrontmatter(got)
			if !ok {
				t.Fatal("result no longer parses")
			}
			if fm[field] != "high" {
				t.Errorf("fm[%s] = %q, want \"high\"", field, fm[field])
			}
			// Every other frontmatter key must survive untouched.
			before, _ := ParseFrontmatter(wellFormed)
			for k, v := range before {
				if k == field {
					continue
				}
				if fm[k] != v {
					t.Errorf("fm[%s] = %q, want unchanged %q", k, fm[k], v)
				}
			}
			// Exactly one line differs from the original.
			assertOnlyLinesDiffer(t, wellFormed, got, 1)
		})
	}
}

func TestSetFieldFamilyInsertWhenAbsent(t *testing.T) {
	got, err := SetField(wellFormed, "T-050", "family", "T-020")
	if err != nil {
		t.Fatalf("SetField(family): %v", err)
	}
	fm, ok := ParseFrontmatter(got)
	if !ok {
		t.Fatal("result no longer parses")
	}
	if fm["family"] != "T-020" {
		t.Errorf("fm[family] = %q, want \"T-020\"", fm["family"])
	}
	lines := strings.Split(got, "\n")
	impactAt := -1
	familyAt := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "impact:") {
			impactAt = i
		}
		if strings.HasPrefix(l, "family:") {
			familyAt = i
		}
	}
	if familyAt == -1 || impactAt == -1 || familyAt != impactAt-1 {
		t.Errorf("family: (line %d) is not immediately before impact: (line %d)", familyAt, impactAt)
	}
	if got, want := len(lines), len(strings.Split(wellFormed, "\n"))+1; got != want {
		t.Errorf("line count = %d, want %d (one inserted line)", got, want)
	}
}

func TestSetFieldFamilyRewriteWhenPresent(t *testing.T) {
	withFamily := strings.Replace(wellFormed, "impact: medium", "family: T-010\nimpact: medium", 1)
	got, err := SetField(withFamily, "T-050", "family", "T-020")
	if err != nil {
		t.Fatalf("SetField(family): %v", err)
	}
	fm, ok := ParseFrontmatter(got)
	if !ok {
		t.Fatal("result no longer parses")
	}
	if fm["family"] != "T-020" {
		t.Errorf("fm[family] = %q, want \"T-020\"", fm["family"])
	}
	assertOnlyLinesDiffer(t, withFamily, got, 1)
}

func TestSetFieldMissingRequiredKeyRefuses(t *testing.T) {
	broken := strings.Replace(wellFormed, "impact: medium\n", "", 1)
	if _, err := SetField(broken, "T-050", "impact", "high"); err == nil {
		t.Fatal("expected a refusal when the targeted key has no line to rewrite")
	}
}

func TestSetFieldUnknownFieldRefuses(t *testing.T) {
	_, err := SetField(wellFormed, "T-050", "depends-on", "[T-001]")
	if err == nil {
		t.Fatal("expected a refusal for a field outside SettableFields")
	}
	if !strings.Contains(err.Error(), "not settable") {
		t.Errorf("error = %q, want it to say the field is not settable", err)
	}
}

func TestSetFieldNoFrontmatterRefuses(t *testing.T) {
	if _, err := SetField("no frontmatter here", "T-050", "impact", "high"); err == nil {
		t.Fatal("expected a refusal with no frontmatter block")
	}
}

func TestSetFieldPreservesUnknownKey(t *testing.T) {
	withUnknown := strings.Replace(wellFormed, "impact: medium", "future_key: something\nimpact: medium", 1)
	got, err := SetField(withUnknown, "T-050", "impact", "high")
	if err != nil {
		t.Fatalf("SetField(impact): %v", err)
	}
	if !strings.Contains(got, "future_key: something") {
		t.Errorf("unrecognized key was not preserved verbatim:\n%s", got)
	}
	assertOnlyLinesDiffer(t, withUnknown, got, 1)
}

func TestSetFieldNoOpSameValueSucceeds(t *testing.T) {
	got, err := SetField(wellFormed, "T-050", "impact", "medium")
	if err != nil {
		t.Fatalf("SetField(impact, same value): %v", err)
	}
	if got != wellFormed {
		t.Errorf("no-op edit changed the text:\n%s", got)
	}
}

func TestSetFieldTitleHappyPath(t *testing.T) {
	got, err := SetField(wellFormed, "T-050", "title", "a renamed ticket")
	if err != nil {
		t.Fatalf("SetField(title): %v", err)
	}
	fm, ok := ParseFrontmatter(got)
	if !ok {
		t.Fatal("result no longer parses")
	}
	if fm["title"] != "a renamed ticket" {
		t.Errorf("fm[title] = %q, want \"a renamed ticket\"", fm["title"])
	}
	if !strings.Contains(got, "# T-050 — a renamed ticket\n") {
		t.Errorf("H1 was not updated:\n%s", got)
	}
	assertOnlyLinesDiffer(t, wellFormed, got, 2)
}

func TestSetFieldTitleRefusesWhenH1AndFrontmatterDisagree(t *testing.T) {
	drifted := strings.Replace(wellFormed, "# T-050 — a well formed ticket", "# T-050 — a DIFFERENT heading", 1)
	_, err := SetField(drifted, "T-050", "title", "a renamed ticket")
	if err == nil {
		t.Fatal("expected a refusal when the heading and frontmatter title already disagree")
	}
	if !strings.Contains(err.Error(), "already disagree") {
		t.Errorf("error = %q, want it to say heading and frontmatter title already disagree", err)
	}
}

func TestSetFieldTitleRefusesOnMissingHeading(t *testing.T) {
	noHeading := strings.Replace(wellFormed, "# T-050 — a well formed ticket\n", "", 1)
	if _, err := SetField(noHeading, "T-050", "title", "a renamed ticket"); err == nil {
		t.Fatal("expected a refusal with no H1 to update")
	}
}

func TestSetFieldTitleRefusesOnDuplicateHeading(t *testing.T) {
	doubled := wellFormed + "\n# T-050 — a well formed ticket\n"
	if _, err := SetField(doubled, "T-050", "title", "a renamed ticket"); err == nil {
		t.Fatal("expected a refusal with more than one matching heading")
	}
}

// TestSetFieldTitleAcceptsPaddedOriginalTitle is the regression test for
// T-102 rework finding F0: a title with legal leading/trailing whitespace
// (ticket.ValidateTitle does not reject it) must not permanently lock a
// ticket out of --title, just because it was never hand-edited and both
// copies of the title happen to carry the same padding.
func TestSetFieldTitleAcceptsPaddedOriginalTitle(t *testing.T) {
	padded := strings.Replace(wellFormed, "title: a well formed ticket", "title:   a well formed ticket  ", 1)
	padded = strings.Replace(padded, "# T-050 — a well formed ticket", "# T-050 —   a well formed ticket  ", 1)

	got, err := SetField(padded, "T-050", "title", "a clean renamed ticket")
	if err != nil {
		t.Fatalf("SetField(title) on a padded-but-undrifted original: %v", err)
	}
	fm, ok := ParseFrontmatter(got)
	if !ok {
		t.Fatal("result no longer parses")
	}
	if fm["title"] != "a clean renamed ticket" {
		t.Errorf("fm[title] = %q, want \"a clean renamed ticket\"", fm["title"])
	}
	if !strings.Contains(got, "# T-050 — a clean renamed ticket\n") {
		t.Errorf("H1 was not updated:\n%s", got)
	}
}

// TestSetFieldTitleRefusesOnQuoteBoundaryDrift is the regression test for
// T-102 rework round 2, finding G0: the round-1 fix for F0 over-widened the
// H1-vs-frontmatter drift check to also strip quote characters, which
// silently treated a genuine quote-boundary disagreement (H1 quoted,
// frontmatter not) as agreement instead of refusing.
func TestSetFieldTitleRefusesOnQuoteBoundaryDrift(t *testing.T) {
	quoteDrifted := strings.Replace(wellFormed, "# T-050 — a well formed ticket", `# T-050 — "a well formed ticket"`, 1)
	_, err := SetField(quoteDrifted, "T-050", "title", "a renamed ticket")
	if err == nil {
		t.Fatal("expected a refusal when the H1 is quoted but the frontmatter title is not")
	}
	if !strings.Contains(err.Error(), "already disagree") {
		t.Errorf("error = %q, want it to say heading and frontmatter title already disagree", err)
	}
}

// TestSetFieldTitleAcceptsANewPaddedTitle is F0's other manifestation: even
// on a pristine, unpadded original, setting a *new* title that itself has
// leading/trailing whitespace must succeed — the parse-back guard compares
// what a re-parse would report, not a byte-for-byte match a frontmatter
// `key: value` line can never round-trip for a padded value in the first
// place.
func TestSetFieldTitleAcceptsANewPaddedTitle(t *testing.T) {
	got, err := SetField(wellFormed, "T-050", "title", "  a new padded title  ")
	if err != nil {
		t.Fatalf("SetField(title, new padded value): %v", err)
	}
	fm, ok := ParseFrontmatter(got)
	if !ok {
		t.Fatal("result no longer parses")
	}
	if fm["title"] != "a new padded title" {
		t.Errorf("fm[title] = %q, want \"a new padded title\" (parseFrontmatter trims)", fm["title"])
	}
	if !strings.Contains(got, "title:   a new padded title  \n") {
		t.Errorf("padded value not written verbatim to frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "# T-050 —   a new padded title  \n") {
		t.Errorf("padded value not written verbatim to the H1:\n%s", got)
	}
}

// assertOnlyLinesDiffer fails the test unless before and after have the same
// line count and differ in exactly want lines — the empirical half of the
// "provably touches nothing else" claim, independent of SetField's own
// internal guard.
func assertOnlyLinesDiffer(t *testing.T, before, after string, want int) {
	t.Helper()
	a := strings.Split(before, "\n")
	b := strings.Split(after, "\n")
	if len(a) != len(b) {
		t.Fatalf("line count changed: %d -> %d", len(a), len(b))
	}
	n := 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	if n != want {
		t.Errorf("%d line(s) differ, want exactly %d", n, want)
	}
}
