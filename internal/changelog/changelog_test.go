package changelog

import (
	"reflect"
	"testing"
)

func TestClassifySubject(t *testing.T) {
	cases := []struct {
		name    string
		subject string
		kind    CommitKind
		id      string
	}{
		{
			name:    "child-project commit with trailing ticket reference",
			subject: "feat(cli): add board audit (T-2)",
			kind:    ChildProject,
			id:      "T-2",
		},
		{
			name:    "board bookkeeping commit is excluded, id still captured",
			subject: "board: T-091 file the incomplete-bookkeeping-commit footgun",
			kind:    Bookkeeping,
			id:      "T-091",
		},
		{
			name:    "board bookkeeping commit with no id after the prefix",
			subject: "board: sync",
			kind:    Bookkeeping,
			id:      "",
		},
		{
			name:    "merge commit carries no ticket reference at all",
			subject: "Merge pull request #30 from codcod/feat/T-081-gate-table",
			kind:    Neither,
			id:      "",
		},
		{
			name:    "a ticket id mid-subject, not trailing in brackets, does not count",
			subject: "docs(changelog): release 0.5.0",
			kind:    Neither,
			id:      "",
		},
		{
			name:    "a trailing GitHub squash-merge PR number is stripped, id still classifies",
			subject: "feat(cli): add a thing (T-050) (#31)",
			kind:    ChildProject,
			id:      "T-050",
		},
		{
			name:    "a trailing GitLab squash-merge MR number is stripped, id still classifies",
			subject: "feat(cli): add a thing (T-050) (!31)",
			kind:    ChildProject,
			id:      "T-050",
		},
		{
			name:    "a bare merge commit's branch-name id must NOT count as shipped or unclassified (T-094 decision 5/6)",
			subject: "Merge pull request #30 from codcod/feat/T-081-gate-table",
			kind:    Neither,
			id:      "",
		},
		{
			name:    "a revert with a parenthesised id neither convention recognises is Unclassified",
			subject: `Revert "feat(x): add a thing (T-050)"`,
			kind:    Unclassified,
			id:      "T-050",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, id := ClassifySubject(c.subject)
			if kind != c.kind || id != c.id {
				t.Fatalf("ClassifySubject(%q) = (%v, %q), want (%v, %q)", c.subject, kind, id, c.kind, c.id)
			}
		})
	}
}

const fixtureChangelog = `# Changelog

## [Unreleased]

### Added

- a multi-line bullet whose reference
  sits on a continuation line (T-070)

## [0.5.0] - 2026-08-12

### Added

- shipped and mentioned in the same breath (T-080)

## [0.4.0] - 2026-08-01

### Added

- an older release, irrelevant to Unreleased (T-999)
`

func TestCheckBoardOnlyTicketExcluded(t *testing.T) {
	subjects := []string{"board: T-091 file a finding"}
	res, err := Check(subjects, fixtureChangelog, "Unreleased")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Shipped) != 0 {
		t.Fatalf("Shipped = %v, want empty — a board: commit never ships", res.Shipped)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("Candidates = %v, want empty", res.Candidates)
	}
	want := []Exclusion{{Subject: "board: T-091 file a finding", ID: "T-091"}}
	if !reflect.DeepEqual(res.Excluded, want) {
		t.Fatalf("Excluded = %+v, want %+v", res.Excluded, want)
	}
}

func TestCheckChildProjectCommitIsShipped(t *testing.T) {
	subjects := []string{"fix(serve): harden linkify (T-090)"}
	res, err := Check(subjects, fixtureChangelog, "Unreleased")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Shipped, []string{"T-090"}) {
		t.Fatalf("Shipped = %v, want [T-090]", res.Shipped)
	}
	if !reflect.DeepEqual(res.Candidates, []string{"T-090"}) {
		t.Fatalf("Candidates = %v, want [T-090] (shipped, not mentioned in Unreleased)", res.Candidates)
	}
}

func TestCheckMultiLineBulletContinuationLineCounts(t *testing.T) {
	// T-070's reference sits on the bullet's *second* line, the exact shape
	// this ticket's own filing miscounted with a first-line-only grep.
	subjects := []string{"feat(x): something (T-070)"}
	res, err := Check(subjects, fixtureChangelog, "Unreleased")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("Candidates = %v, want empty — T-070 is mentioned on a continuation line", res.Candidates)
	}
	if !contains(res.Mentioned, "T-070") {
		t.Fatalf("Mentioned = %v, want it to contain T-070", res.Mentioned)
	}
}

func TestCheckShippedAndMentionedYieldsNoCandidate(t *testing.T) {
	subjects := []string{"feat(x): something (T-080)"}
	res, err := Check(subjects, fixtureChangelog, "0.5.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("Candidates = %v, want empty", res.Candidates)
	}
}

func TestCheckMentionedButUnshippedYieldsNothing(t *testing.T) {
	// T-999 is named in [0.4.0] but never appears in subjects at all — decision
	// 4: the check is one direction only (shipped-but-unmentioned), so an
	// unshipped-but-mentioned ticket must never surface as a candidate or in
	// any other reported field.
	res, err := Check(nil, fixtureChangelog, "0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Shipped) != 0 {
		t.Fatalf("Shipped = %v, want empty (no commits given)", res.Shipped)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("Candidates = %v, want empty", res.Candidates)
	}
}

func TestCheckUnknownSectionErrors(t *testing.T) {
	if _, err := Check(nil, fixtureChangelog, "9.9.9"); err == nil {
		t.Fatal("Check with an absent section = nil error, want one")
	}
}

// TestCheckUnclassifiedSubjectIsNeitherShippedNorExcluded (T-094 decision 6):
// an Unclassified subject must land only in Result.Unclassified, never in
// Shipped (it would be a false candidate) or Excluded (it isn't a board:
// bookkeeping commit either).
func TestCheckUnclassifiedSubjectIsNeitherShippedNorExcluded(t *testing.T) {
	subjects := []string{`Revert "feat(x): add a thing (T-050)"`}
	res, err := Check(subjects, fixtureChangelog, "Unreleased")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Shipped) != 0 {
		t.Fatalf("Shipped = %v, want empty", res.Shipped)
	}
	if len(res.Excluded) != 0 {
		t.Fatalf("Excluded = %v, want empty", res.Excluded)
	}
	want := []Exclusion{{Subject: `Revert "feat(x): add a thing (T-050)"`, ID: "T-050"}}
	if !reflect.DeepEqual(res.Unclassified, want) {
		t.Fatalf("Unclassified = %+v, want %+v", res.Unclassified, want)
	}
}

func TestCheckDeduplicatesShippedIDs(t *testing.T) {
	subjects := []string{
		"fix(serve): part one (T-090)",
		"fix(serve): part two (T-090)",
	}
	res, err := Check(subjects, fixtureChangelog, "Unreleased")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Shipped, []string{"T-090"}) {
		t.Fatalf("Shipped = %v, want a deduplicated [T-090]", res.Shipped)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
