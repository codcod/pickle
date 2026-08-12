// Package changelog implements the logic behind `pickle changelog check`
// (T-093): does CHANGELOG.md account for every ticket that shipped since the
// last release, either with an entry naming it or a recorded decision in the
// ticket saying it deliberately gets none?
//
// It is pure text-in, text-out logic — no subprocess, no filesystem, no
// printing, no exit code — mirroring internal/audit's shape (a Result value,
// fixture/table-testable) so the classification and parsing rules can be
// pinned with literal string fixtures instead of a real repo. The I/O
// (running git via internal/vcs, reading CHANGELOG.md, resolving a candidate
// id to its ticket file) is internal/cli/changelog.go's job, not this
// package's.
package changelog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// CommitKind classifies one `git log --format=%s` subject line for the
// purpose of deciding what "shipped since <since>" means.
type CommitKind int

const (
	// Neither means the subject carries no ticket reference either form
	// below recognises — a merge commit, a release/chore commit, or anything
	// else that isn't from a ticket at all. Not part of "shipped", and not
	// reported as an exclusion either: there is nothing to exclude it from.
	Neither CommitKind = iota
	// ChildProject is a Conventional Commit for the target child-project's
	// own code, with a ticket id in brackets at the *end* of the subject —
	// the form the project's commit convention (AGENTS.md) sanctions, e.g.
	// "feat(cli): add board audit (T-2)". This is the "shipped" set the
	// check diffs against the changelog.
	ChildProject
	// Bookkeeping is a `board: ...` ticket/board bookkeeping commit (T-084's
	// convention). It never carries shipped code — a ticket can be filed,
	// re-graded or moved without a line changing — so it is deliberately
	// excluded from "shipped" (T-093 decision 3). Because the check depends
	// on this convention mechanically, an excluded commit is always
	// reported (decision 7), so a project that stops following it sees a
	// visibly odd exclusion list rather than a silent under-report.
	Bookkeeping
)

var (
	// boardIDRE recognises T-084's bookkeeping form and, where present,
	// captures the ticket id that follows the "board:" prefix.
	boardIDRE = regexp.MustCompile(`^board:\s*([A-Z][A-Z0-9]*-\d+)\b`)
	// trailingIDRE recognises a Conventional Commit's ticket id in brackets
	// at the end of the subject — the only position the project's commit
	// convention sanctions for child-project code.
	trailingIDRE = regexp.MustCompile(`\(([A-Z][A-Z0-9]*-\d+)\)\s*$`)
	// idRE finds any ticket-id-shaped token anywhere in text. Used to scan a
	// changelog section's whole body for every id it mentions, not just a
	// bullet's first line — see sectionIDs.
	idRE = regexp.MustCompile(`\b[A-Z][A-Z0-9]*-\d+\b`)
	// sectionHeadingRE recognises any top-level "## [<name>]" heading, used
	// to find where a named section ends.
	sectionHeadingRE = regexp.MustCompile(`^##\s+\[`)
)

// ClassifySubject classifies one commit subject line and, when the
// classification carries a ticket id, returns it — "" otherwise (a
// Bookkeeping subject that doesn't parse an id after "board:", or any
// Neither subject).
func ClassifySubject(subject string) (CommitKind, string) {
	s := strings.TrimSpace(subject)
	if strings.HasPrefix(s, "board:") {
		if m := boardIDRE.FindStringSubmatch(s); m != nil {
			return Bookkeeping, m[1]
		}
		return Bookkeeping, ""
	}
	if m := trailingIDRE.FindStringSubmatch(s); m != nil {
		return ChildProject, m[1]
	}
	return Neither, ""
}

// Exclusion records one bookkeeping commit the check left out of "shipped",
// so a convention drift (decision 7) is visible in the report rather than
// silently under-counting.
type Exclusion struct {
	Subject string
	ID      string // "" when the subject didn't parse a ticket id at all
}

// Result is the outcome of Check.
type Result struct {
	Section string
	// Shipped is every ChildProject-classified ticket id, deduplicated, in
	// the order the commit log presented them (newest first, matching `git
	// log`'s own default).
	Shipped []string
	// Mentioned is every ticket id Section's body already names, sorted.
	Mentioned []string
	// Candidates is the subset of Shipped that Mentioned does not cover, in
	// Shipped's order — the report's whole payload. Empty means the
	// changelog already accounts for everything that shipped.
	Candidates []string
	// Excluded is every Bookkeeping commit, in commit-log order.
	Excluded []Exclusion
}

// Check classifies subjects (as from `git log --format=%s <since>..HEAD`)
// and diffs the resulting shipped set against the ticket ids already
// mentioned in changelogText's named section (a top-level "## [<section>]"
// heading, e.g. "Unreleased" or a version like "0.5.0"). It returns an error
// only when section cannot be found in changelogText at all.
//
// It reports one direction only — shipped but unmentioned (decision 4) —
// and never the reverse: an entry may legitimately reference an older,
// already-shipped ticket for context (e.g. the real `[0.5.0]` section
// naming T-083, which shipped earlier), and flagging that would be noise.
func Check(subjects []string, changelogText, section string) (Result, error) {
	mentioned, err := sectionIDs(changelogText, section)
	if err != nil {
		return Result{}, err
	}

	var shipped []string
	seen := make(map[string]bool, len(subjects))
	var excluded []Exclusion
	for _, subj := range subjects {
		kind, id := ClassifySubject(subj)
		switch kind {
		case ChildProject:
			if id != "" && !seen[id] {
				seen[id] = true
				shipped = append(shipped, id)
			}
		case Bookkeeping:
			excluded = append(excluded, Exclusion{Subject: subj, ID: id})
		case Neither:
			// not from a ticket at all — neither shipped nor excluded
		}
	}

	var candidates []string
	for _, id := range shipped {
		if !mentioned[id] {
			candidates = append(candidates, id)
		}
	}

	return Result{
		Section:    section,
		Shipped:    shipped,
		Mentioned:  sortedKeys(mentioned),
		Candidates: candidates,
		Excluded:   excluded,
	}, nil
}

// sectionIDs returns every ticket id mentioned anywhere in the named
// section's body — the whole multi-line bullet, not just its first line,
// which is the mistake this ticket's own filing made counting `(T-NNN)`
// references by a first-line-only grep (see the ticket's Description). The
// section runs from its own "## [<section>]" heading to the next top-level
// "## [" heading (any name) or end of file, whichever comes first.
func sectionIDs(changelogText, section string) (map[string]bool, error) {
	heading := regexp.MustCompile(`^##\s+\[` + regexp.QuoteMeta(section) + `\]`)

	lines := strings.Split(changelogText, "\n")
	start := -1
	for i, ln := range lines {
		if heading.MatchString(ln) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("section %q not found (looked for a %q heading)", section, "## ["+section+"]")
	}

	end := len(lines)
	for i := start; i < len(lines); i++ {
		if sectionHeadingRE.MatchString(lines[i]) {
			end = i
			break
		}
	}

	body := strings.Join(lines[start:end], "\n")
	ids := make(map[string]bool)
	for _, m := range idRE.FindAllString(body, -1) {
		ids[m] = true
	}
	return ids, nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
