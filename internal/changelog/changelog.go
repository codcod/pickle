// Package changelog implements the logic behind `pickle changelog check`
// (T-093): is every ticket that shipped since the last release named in
// CHANGELOG.md? Only the mention is checked here — whether an unmentioned
// ticket needs an entry or already records a decision to get none is the
// reader's judgement, which the CLI supports by pointing at the ticket file
// (T-093 decision 5: no exemption mechanism).
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
	// Unclassified (T-094 decision 6) is the safety net: a subject that is
	// not Bookkeeping, does not classify as ChildProject even after stripping
	// a trailing PR/MR-number token (decision 5), but still names a ticket in
	// parentheses somewhere else in the subject — e.g. a revert,
	// `Revert "feat(cli): add a thing (T-050)"`. Deliberately narrower than
	// "an id anywhere": a plain merge commit's branch name
	// (`Merge pull request #30 from codcod/feat/T-081-gate-table`) carries a
	// bare id too, and counting *that* as noteworthy would add a line per
	// merge to every report — the opposite of what this kind exists for.
	// Not part of "shipped" and not an Exclusion; it gets its own list so a
	// subject that mentions a ticket is never silently neither.
	Unclassified
)

var (
	// boardIDRE recognises T-084's bookkeeping form and, where present,
	// captures the ticket id that follows the "board:" prefix.
	boardIDRE = regexp.MustCompile(`^board:\s*([A-Z][A-Z0-9]*-\d+)\b`)
	// trailingIDRE recognises a Conventional Commit's ticket id in brackets
	// at the end of the subject — the only position the project's commit
	// convention sanctions for child-project code.
	trailingIDRE = regexp.MustCompile(`\(([A-Z][A-Z0-9]*-\d+)\)\s*$`)
	// prTokenRE (T-094 decision 5) matches a trailing GitHub/GitLab
	// squash-merge annotation — "(#31)" or "(!31)" — appended *after* a
	// Conventional Commit's own trailing "(T-NNN)". Stripped once, before
	// trailingIDRE runs, so "feat(cli): add a thing (T-050) (#31)" still
	// classifies as ChildProject. Deliberately narrow (digits only, exactly
	// one trailing group) rather than widening trailingIDRE itself — see
	// ClassifySubject.
	prTokenRE = regexp.MustCompile(`\s*\([#!]\d+\)$`)
	// parenIDRE recognises a ticket id in parentheses anywhere in a subject —
	// the shape Unclassified looks for once trailingIDRE has already missed.
	// Anchoring on the parentheses (unlike idRE below) is what keeps an
	// ordinary merge commit's bare branch-name id from counting.
	parenIDRE = regexp.MustCompile(`\(([A-Z][A-Z0-9]*-\d+)\)`)
	// idRE finds any ticket-id-shaped token anywhere in text. Used to scan a
	// changelog section's whole body for every id it mentions, not just a
	// bullet's first line — see sectionIDs.
	idRE = regexp.MustCompile(`\b[A-Z][A-Z0-9]*-\d+\b`)
	// sectionHeadingRE recognises any top-level "## [<name>]" heading, used
	// to find where a named section ends.
	sectionHeadingRE = regexp.MustCompile(`^##\s+\[`)
)

// ClassifySubject classifies one commit subject line and returns the
// *primary* ticket id — the one ChildProject's "shipped" set needs, and the
// one Bookkeeping parses right after "board:" — or "" when there isn't one
// (a Bookkeeping subject with no id after "board:", or any Neither
// subject). It is not the full id inventory for a Bookkeeping or
// Unclassified subject: Check builds that separately, with subjectIDs,
// because a bookkeeping commit may legally name more than one ticket
// (rules §0's "board: T-NNN[, T-MMM …] …" form) and this function's single
// return only ever carried the first.
func ClassifySubject(subject string) (CommitKind, string) {
	s := strings.TrimSpace(subject)
	if strings.HasPrefix(s, "board:") {
		// Never PR-token-stripped: a bookkeeping commit is never merged
		// through a squash-merge button in this flow (rules §0), so
		// stripping here would only add a way to misparse.
		if m := boardIDRE.FindStringSubmatch(s); m != nil {
			return Bookkeeping, m[1]
		}
		return Bookkeeping, ""
	}
	stripped := prTokenRE.ReplaceAllString(s, "")
	if m := trailingIDRE.FindStringSubmatch(stripped); m != nil {
		return ChildProject, m[1]
	}
	if m := parenIDRE.FindStringSubmatch(stripped); m != nil {
		return Unclassified, m[1]
	}
	return Neither, ""
}

// Exclusion records one commit the check left out of "shipped", with every
// ticket id it names. It carries both non-shipped-but-noteworthy kinds:
// Result.Excluded's bookkeeping commits, so a convention drift (decision 7,
// T-093) is visible in the report rather than silently under-counting, and
// Result.Unclassified's safety-net commits (decision 6, T-094). One struct
// rather than two, because the shape either list needs is identical.
type Exclusion struct {
	Subject string
	// IDs is every ticket id the subject mentions, deduplicated, in the order
	// they first appear (T-095 decision 2: a permissive scan of the whole
	// subject, not just the id ClassifySubject's single return carries — a
	// bookkeeping commit may legally name several tickets, and most that do
	// carry the extra ones outside the leading "board: T-NNN[, T-MMM …]"
	// list, in the verb phrase). Empty only when the subject names no id at
	// all — the loudest possible symptom of a bookkeeping-convention drift
	// (T-094 decision 4); an Unclassified commit's IDs is never empty, since
	// finding one is what made it Unclassified rather than Neither. Computed
	// for both Excluded and Unclassified for symmetry, but only Excluded's
	// IDs is read today — printUnclassified still prints subjects only
	// (T-095 review finding N6).
	IDs []string
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
	// Unclassified is every Unclassified commit, in commit-log order — a
	// subject that names a ticket in parentheses but matches neither the
	// Bookkeeping nor the ChildProject convention (T-094 decision 6).
	Unclassified []Exclusion
}

// Check classifies subjects (as from `git log --format=%s <since>..<until>`)
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
	var unclassified []Exclusion
	for _, subj := range subjects {
		kind, id := ClassifySubject(subj)
		switch kind {
		case ChildProject:
			if id != "" && !seen[id] {
				seen[id] = true
				shipped = append(shipped, id)
			}
		case Bookkeeping:
			excluded = append(excluded, Exclusion{Subject: subj, IDs: subjectIDs(subj)})
		case Unclassified:
			unclassified = append(unclassified, Exclusion{Subject: subj, IDs: subjectIDs(subj)})
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
		Section:      section,
		Shipped:      shipped,
		Mentioned:    sortedKeys(mentioned),
		Candidates:   candidates,
		Excluded:     excluded,
		Unclassified: unclassified,
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

// subjectIDs returns every ticket-id-shaped token idRE finds anywhere in
// subject, deduplicated, in first-seen order. It is deliberately the same
// permissive rule sectionIDs already uses to scan a changelog section's
// whole body — so both halves of the report recognise an id identically —
// rather than a stricter pattern anchored to "board:"'s documented
// leading-list grammar. Measured across this project's entire commit
// history (T-095 decision 2): of every multi-id `board:` subject, only one
// keeps its extra ids in that leading list; the rest carry them in the verb
// phrase ("board: T-089 reviewed and done, T-090 filed, T-070 re-graded"),
// which a grammar-strict parser would still miss.
func subjectIDs(subject string) []string {
	var ids []string
	seen := make(map[string]bool)
	for _, m := range idRE.FindAllString(subject, -1) {
		if !seen[m] {
			seen[m] = true
			ids = append(ids, m)
		}
	}
	return ids
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
