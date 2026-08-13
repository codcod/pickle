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
//
// A token counts as a ticket id only when its prefix is one the caller's
// project actually registers (T-097): the caller (internal/cli/changelog.go)
// resolves the registered prefixes from pickle.toml and passes them into
// ClassifySubject/Check as a []string. Every id-recognition site in this
// package — the leading "board:" id, a Conventional Commit's trailing
// bracketed id, a parenthesised id anywhere, and a whole-subject/whole-body
// scan — shares that one closed prefix set, so an id-shaped token whose
// prefix isn't registered (SHA-256, UTF-8, RFC-7231, CVE-2024, ...) is never
// mistaken for a ticket id, in either the exclusion summary or the shipped
// candidate list.
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

// defaultPrefix mirrors config.DefaultTicketPrefix. Duplicated rather than
// imported: this package is a leaf and its doc promises pure text-in,
// text-out logic (T-097 decision 6) — it must not import internal/config.
const defaultPrefix = "T"

// prTokenRE (T-094 decision 5) matches a trailing GitHub/GitLab
// squash-merge annotation — "(#31)" or "(!31)" — appended *after* a
// Conventional Commit's own trailing "(T-NNN)". Stripped once, before the
// trailing-id pattern runs, so "feat(cli): add a thing (T-050) (#31)" still
// classifies as ChildProject. Deliberately narrow (digits only, exactly one
// trailing group) rather than widening the trailing-id pattern itself — see
// ClassifySubject. It never carries an id-shaped token itself, so it needs
// no prefix awareness.
var prTokenRE = regexp.MustCompile(`\s*\([#!]\d+\)$`)

// sectionHeadingRE recognises any top-level "## [<name>]" heading, used to
// find where a named section ends. It matches no ticket id, so it too needs
// no prefix awareness.
var sectionHeadingRE = regexp.MustCompile(`^##\s+\[`)

// idPatterns is the one prefix-aware predicate every id-recognition site in
// this package shares (T-097 decision 1): "is this token a ticket id?" is
// answered exclusively by whether its prefix is one of the caller's
// registered ticket-id prefixes, never by shape alone. Building it once per
// Check/ClassifySubject call (rather than reopening the question per site)
// is also what lets Check compile it a single time for a whole subject list
// — see newIDPatterns and classifySubject.
type idPatterns struct {
	// board recognises T-084's bookkeeping form and, where present, captures
	// the ticket id that follows the "board:" prefix.
	board *regexp.Regexp
	// trailing recognises a Conventional Commit's ticket id in brackets at
	// the end of the subject — the only position the project's commit
	// convention sanctions for child-project code.
	trailing *regexp.Regexp
	// paren recognises a ticket id in parentheses anywhere in a subject —
	// the shape Unclassified looks for once trailing has already missed.
	// Anchoring on the parentheses (unlike any below) is what keeps an
	// ordinary merge commit's bare branch-name id from counting.
	paren *regexp.Regexp
	// any finds any registered-prefix ticket-id-shaped token anywhere in
	// text. Used to scan a whole subject or a changelog section's whole body
	// for every id it mentions, not just a bullet's or subject's first
	// match — see subjectIDs and sectionIDs.
	any *regexp.Regexp
}

// newIDPatterns builds an idPatterns whose four shapes recognise only the
// given ticket-id prefixes (T-097 decision 1), closing the alternation group
// each pattern used to leave open to any [A-Z][A-Z0-9]* family. prefixes is
// deduplicated and regexp.QuoteMeta'd before interpolation — belt-and-braces
// against a future loosening of config.ticketPrefixRE
// (`^[A-Z][A-Z0-9]{0,7}$`), which today already guarantees every prefix is
// metacharacter-free (T-097 decision 8). An empty or all-blank prefixes
// falls back to ["T"] (defaultPrefix) rather than reinstating the old
// unrestricted match (T-097 decision 7) — config.Validate guarantees the CLI
// never passes an empty slice, but a future caller must not be able to
// silently regress to the permissive behaviour this ticket removes.
func newIDPatterns(prefixes []string) *idPatterns {
	seen := make(map[string]bool, len(prefixes))
	var uniq []string
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		uniq = append(uniq, regexp.QuoteMeta(p))
	}
	if len(uniq) == 0 {
		uniq = []string{regexp.QuoteMeta(defaultPrefix)}
	}
	alt := "(?:" + strings.Join(uniq, "|") + ")"
	return &idPatterns{
		board:    regexp.MustCompile(`^board:\s*(` + alt + `-\d+)\b`),
		trailing: regexp.MustCompile(`\((` + alt + `-\d+)\)\s*$`),
		paren:    regexp.MustCompile(`\((` + alt + `-\d+)\)`),
		any:      regexp.MustCompile(`\b` + alt + `-\d+\b`),
	}
}

// ClassifySubject classifies one commit subject line and returns the
// *primary* ticket id — the one ChildProject's "shipped" set needs, and the
// one Bookkeeping parses right after "board:" — or "" when there isn't one
// (a Bookkeeping subject with no id after "board:", or any Neither
// subject). It is not the full id inventory for a Bookkeeping or
// Unclassified subject: Check builds that separately, with subjectIDs,
// because a bookkeeping commit may legally name more than one ticket
// (rules §0's "board: T-NNN[, T-MMM …] …" form) and this function's single
// return only ever carried the first.
//
// prefixes is the set of ticket-id prefixes the caller's project registers
// (T-097) — a token whose prefix isn't in this set is never recognised as a
// ticket id by any of the four shapes below. This is a thin wrapper that
// compiles an idPatterns per call; Check instead compiles one and reuses it
// across a whole subject list via the unexported classifySubject.
func ClassifySubject(subject string, prefixes []string) (CommitKind, string) {
	return classifySubject(subject, newIDPatterns(prefixes))
}

func classifySubject(subject string, p *idPatterns) (CommitKind, string) {
	s := strings.TrimSpace(subject)
	if strings.HasPrefix(s, "board:") {
		// Never PR-token-stripped: a bookkeeping commit is never merged
		// through a squash-merge button in this flow (rules §0), so
		// stripping here would only add a way to misparse.
		if m := p.board.FindStringSubmatch(s); m != nil {
			return Bookkeeping, m[1]
		}
		return Bookkeeping, ""
	}
	stripped := prTokenRE.ReplaceAllString(s, "")
	if m := p.trailing.FindStringSubmatch(stripped); m != nil {
		return ChildProject, m[1]
	}
	if m := p.paren.FindStringSubmatch(stripped); m != nil {
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
// prefixes is the set of ticket-id prefixes the caller's project registers
// (T-097) — see idPatterns. It is resolved once into an idPatterns and
// reused across every subject and the changelog body, rather than
// recompiled per call.
//
// It reports one direction only — shipped but unmentioned (decision 4) —
// and never the reverse: an entry may legitimately reference an older,
// already-shipped ticket for context (e.g. the real `[0.5.0]` section
// naming T-083, which shipped earlier), and flagging that would be noise.
func Check(subjects []string, changelogText, section string, prefixes []string) (Result, error) {
	p := newIDPatterns(prefixes)

	mentioned, err := sectionIDs(changelogText, section, p)
	if err != nil {
		return Result{}, err
	}

	var shipped []string
	seen := make(map[string]bool, len(subjects))
	var excluded []Exclusion
	var unclassified []Exclusion
	for _, subj := range subjects {
		kind, id := classifySubject(subj, p)
		switch kind {
		case ChildProject:
			if id != "" && !seen[id] {
				seen[id] = true
				shipped = append(shipped, id)
			}
		case Bookkeeping:
			excluded = append(excluded, Exclusion{Subject: subj, IDs: subjectIDs(subj, p)})
		case Unclassified:
			unclassified = append(unclassified, Exclusion{Subject: subj, IDs: subjectIDs(subj, p)})
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
//
// A changelog body legitimately contains prose like "SHA-256" or "UTF-8"
// alongside real ticket references, so p's closed prefix set (T-097)
// matters just as much here as it does scanning a commit subject: without
// it, a released changelog entry mentioning a non-ticket id-shaped token
// would be misread as mentioning a ticket.
func sectionIDs(changelogText, section string, p *idPatterns) (map[string]bool, error) {
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
	for _, m := range p.any.FindAllString(body, -1) {
		ids[m] = true
	}
	return ids, nil
}

// subjectIDs returns every ticket-id-shaped token p.any finds anywhere in
// subject, deduplicated, in first-seen order. It is the same *whole-subject*
// rule sectionIDs already uses to scan a changelog section's whole body —
// so both halves of the report recognise an id identically — rather than a
// stricter pattern anchored to "board:"'s documented leading-list grammar.
// Measured across this project's entire commit history (T-095 decision 2):
// of every multi-id `board:` subject, only one keeps its extra ids in that
// leading list; the rest carry them in the verb phrase ("board: T-089
// reviewed and done, T-090 filed, T-070 re-graded"), which a grammar-strict
// parser would still miss. What changed (T-097) is not *where* a token may
// appear, only *which* tokens count: p's prefix set is closed to the
// project's registered ticket-id prefixes, so an id-shaped token like
// SHA-256 or RFC-7231 is never counted as a ticket id here either.
func subjectIDs(subject string, p *idPatterns) []string {
	var ids []string
	seen := make(map[string]bool)
	for _, m := range p.any.FindAllString(subject, -1) {
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
