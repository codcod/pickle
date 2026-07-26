// Package ticket is the shared model for ticket files: the board status table,
// frontmatter parsing, History interpretation, and loading a whole tickets/ tree.
// Both the board audit (internal/audit) and the project remove-guard (internal/cli)
// build on it, so the frontmatter scan lives in exactly one place.
package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Status is a board status with its directory, display name, and terminal flag.
type Status struct {
	Dir      string
	Name     string
	Terminal bool
}

// Statuses lists every status in board order.
var Statuses = []Status{
	{"1-to-do", "TO DO", false},
	{"2-ready", "READY", false},
	{"3-in-development", "IN DEVELOPMENT", false},
	{"4-in-review", "IN REVIEW", false},
	{"5-rework", "REWORK", false},
	{"6-done", "DONE", true},
	{"7-dropped", "DROPPED", true},
}

// StatusByDir returns the status for a directory name.
func StatusByDir(dir string) (Status, bool) {
	for _, s := range Statuses {
		if s.Dir == dir {
			return s, true
		}
	}
	return Status{}, false
}

// StatusByName returns the status for a display name.
func StatusByName(name string) (Status, bool) {
	for _, s := range Statuses {
		if s.Name == name {
			return s, true
		}
	}
	return Status{}, false
}

var statusNumRE = regexp.MustCompile(`^\d+-`)

// StatusByToken resolves a user-supplied status token, case-insensitively, in any
// of three forms: the dir name ("3-in-development"), the dir minus its number
// ("in-development"), or the display name lower-cased with spaces to hyphens
// ("in-development" from "IN DEVELOPMENT").
func StatusByToken(tok string) (Status, bool) {
	t := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(tok)), " ", "-")
	for _, s := range Statuses {
		bare := statusNumRE.ReplaceAllString(s.Dir, "")
		name := strings.ReplaceAll(strings.ToLower(s.Name), " ", "-")
		if t == s.Dir || t == bare || t == name {
			return s, true
		}
	}
	return Status{}, false
}

// Ticket is one parsed ticket file.
type Ticket struct {
	ID        string            // "T-001" (from the filename)
	Num       int               // 1
	Slug      string            // "config-and-project-registry"
	Dir       string            // status directory, e.g. "1-to-do"
	Path      string            // absolute path
	Front     map[string]string // raw frontmatter values (title kept verbatim)
	DependsOn []string          // parsed depends-on ids — hard dependencies; gate pickup
	SpawnedBy []string          // parsed spawned-by ids — lineage only, never a gate
	Text      string            // full file text
}

// Base is the filename without extension, e.g. "T-001-config-and-project-registry".
func (t *Ticket) Base() string { return strings.TrimSuffix(filepath.Base(t.Path), ".md") }

// Project is the project: frontmatter value.
func (t *Ticket) Project() string { return t.Front["project"] }

var (
	filenameRE = regexp.MustCompile(`^(T-\d+)-[A-Za-z0-9._-]+\.md$`)
	// idRE is the canonical ticket-id shape, the same `T-\d+` filenameRE anchors.
	// Exported through ValidID so that creation time (internal/cli) and the audit
	// will share one definition rather than growing a second: internal/audit does
	// not validate id shape at all today, and T-027 is to add that using ValidID
	// instead of its own regex. The shape is still spelled out separately in
	// filenameRE above and in board.rowRE; unifying those is T-027's call.
	idRE       = regexp.MustCompile(`^T-\d+$`)
	fmKeyRE    = regexp.MustCompile(`^([A-Za-z-]+):\s*(.*)$`)
	inlineHash = regexp.MustCompile(`\s+#.*$`)
	historyRE  = regexp.MustCompile(`^-\s*\d{4}-\d{2}-\d{2}\s*[—-]+\s*(.+)$`)
	createdRE  = regexp.MustCompile(`(?i)^created\s*\(([^)]+)\)`)
	mergedRE   = regexp.MustCompile(`(?i)^merged\b`)
)

// ParseFrontmatter parses the YAML-ish frontmatter block into a flat map. ok is
// false when there is no leading `---` block.
func ParseFrontmatter(text string) (map[string]string, bool) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	fm := map[string]string{}
	for i := 1; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		m := fmKeyRE.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		key, val := m[1], m[2]
		if key != "title" { // titles may legitimately contain '#'
			val = inlineHash.ReplaceAllString(val, "")
		}
		fm[key] = strings.Trim(strings.TrimSpace(val), `"'`)
	}
	return fm, true
}

// ParseDepends parses a bracketed ticket-id list like "[T-001, T-002]" or "[]".
// Brackets are optional, so it also accepts the comma-separated flag form
// ("T-001,T-002"). Used for both `depends-on:` and `spawned-by:` — the two share
// a wire format and differ only in meaning.
func ParseDepends(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ValidID reports whether s has the canonical ticket-id shape (T-<digits>).
// It is a *shape* check only: whether the ticket exists is the audit's job, since
// a forward reference to a not-yet-filed ticket is legal input (rules §3).
//
// Deliberately does not trim: callers that accept human input tokenize first
// (see ParseIDList, which trims via ParseDepends).
func ValidID(s string) bool { return idRE.MatchString(s) }

// ParseIDList parses a comma-separated (or bracketed) ticket-id list like
// ParseDepends, but validates each token's shape and drops duplicates, keeping
// first-seen order. For flag input, where rejecting is possible; ParseDepends
// stays lenient for already-written frontmatter, where it is not.
//
// The error names the offending token so a malformed id is reported as malformed
// rather than as a missing ticket.
func ParseIDList(raw string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, tok := range ParseDepends(raw) {
		if !ValidID(tok) {
			return nil, fmt.Errorf("%q is not a ticket id (expected T-NNN)", tok)
		}
		if seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out, nil
}

// LastHistoryStatus returns the display name of the newest status-bearing History
// transition, or "" if none is parseable. Merge notes ("… — MERGED: …") are skipped.
func LastHistoryStatus(text string) string {
	inHistory := false
	status := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			inHistory = strings.TrimSpace(line) == "## History"
			continue
		}
		if !inHistory {
			continue
		}
		m := historyRE.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		body := strings.TrimSpace(m[1])
		if mergedRE.MatchString(body) {
			continue // merge note, not a status transition
		}
		if c := createdRE.FindStringSubmatch(body); c != nil {
			status = strings.ToUpper(strings.TrimSpace(c[1]))
			continue
		}
		if idx := strings.LastIndex(body, "→"); idx >= 0 {
			target := body[idx+len("→"):]
			target = strings.TrimSpace(strings.SplitN(target, ":", 2)[0])
			if _, ok := StatusByName(strings.ToUpper(target)); ok {
				status = strings.ToUpper(target)
			}
		}
	}
	return status
}

// LastHistoryReason returns the ": <reason>" clause of the newest status-bearing
// History transition (the text `move --reason` appends), or "" when the last
// transition carries no reason. Merge notes and created lines are skipped — they
// are not transitions. The board renderer derives the DROPPED `reason` and REWORK
// `open findings` cells from this, so those facts live only in the ticket (D3).
func LastHistoryReason(text string) string {
	inHistory := false
	reason := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			inHistory = strings.TrimSpace(line) == "## History"
			continue
		}
		if !inHistory {
			continue
		}
		m := historyRE.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		body := strings.TrimSpace(m[1])
		if mergedRE.MatchString(body) {
			continue // merge note, not a status transition
		}
		idx := strings.LastIndex(body, "→")
		if idx < 0 {
			continue // created line or free-form note
		}
		target := body[idx+len("→"):]
		parts := strings.SplitN(target, ":", 2)
		if _, ok := StatusByName(strings.ToUpper(strings.TrimSpace(parts[0]))); !ok {
			continue
		}
		if len(parts) == 2 {
			reason = strings.TrimSpace(parts[1])
		} else {
			reason = ""
		}
	}
	return reason
}

// MergeLine returns the text of the newest merge History line ("merged to main
// (abc1234)" from "- 2026-07-23 — merged to main (abc1234)"), or "" when the
// History records no merge. The board renderer derives the DONE `merged` cell
// from this (D3), so HasMergeLine and the cell can never disagree.
func MergeLine(text string) string {
	inHistory := false
	merge := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			inHistory = strings.TrimSpace(line) == "## History"
			continue
		}
		if !inHistory {
			continue
		}
		if m := historyRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			if body := strings.TrimSpace(m[1]); mergedRE.MatchString(body) {
				merge = body
			}
		}
	}
	return merge
}

// HasMergeLine reports whether the History records a merge ("… — MERGED: …").
// Defined in terms of MergeLine so the gate check and the board's `merged`
// cell can never disagree.
func HasMergeLine(text string) bool {
	return MergeLine(text) != ""
}

// Legal grade values (single values or adjacent-pair ranges). These are the one
// source of truth, consumed by both the board audit and `ticket new`.
var (
	LegalImpact     = gradeSet("low", "medium", "high", "critical", "low-medium", "medium-high", "high-critical")
	LegalComplexity = gradeSet("low", "medium", "high", "low-medium", "medium-high")
	LegalCost       = gradeSet("S", "M", "L", "XL", "S-M", "M-L", "L-XL")
)

func gradeSet(vals ...string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// ValidGrade reports whether v is legal for the grade kind ("impact",
// "complexity", or "cost").
func ValidGrade(kind, v string) bool {
	switch kind {
	case "impact":
		return LegalImpact[v]
	case "complexity":
		return LegalComplexity[v]
	case "cost":
		return LegalCost[v]
	}
	return false
}

// NextNum returns the next free ticket number: max(numeric part of every T-NNN
// filename across all status dirs) + 1 (one global namespace). Scans filenames
// directly so it is robust to files that fail frontmatter parsing.
func NextNum(root string) int {
	max := 0
	for _, s := range Statuses {
		entries, err := os.ReadDir(filepath.Join(root, "tickets", s.Dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if m := filenameRE.FindStringSubmatch(e.Name()); m != nil {
				if n, err := strconv.Atoi(strings.TrimPrefix(m[1], "T-")); err == nil && n > max {
					max = n
				}
			}
		}
	}
	return max + 1
}

var slugStripRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a title into a filename slug: lowercase, non-alphanumeric runs
// collapsed to single hyphens, trimmed. Empty input yields "untitled".
func Slugify(title string) string {
	s := strings.Trim(slugStripRE.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if s == "" {
		return "untitled"
	}
	return s
}

// SectionHeadings returns the ordered list of top-level ("## ") section headings
// in a ticket/template body — used to keep Scaffold in step with TEMPLATE.md.
func SectionHeadings(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			out = append(out, strings.TrimSpace(line[3:]))
		}
	}
	return out
}

// renderIDList renders a ticket-id slice in frontmatter form: "[]" when empty,
// "[T-018, T-019]" otherwise — the inverse of ParseDepends.
//
// Two other renderers do the same job today (internal/move's renderDepends and an
// inline join in internal/sync); collapsing all three onto this one is deliberately
// deferred to T-015 rather than done here.
func renderIDList(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	return "[" + strings.Join(ids, ", ") + "]"
}

// Scaffold renders a fresh, canonical TO DO ticket: filled frontmatter, heading,
// and the standard section skeleton (mirroring TEMPLATE.md's section set). The
// full TEMPLATE.md remains the authoring guide; this is the minimal, audit-clean
// starting point `pickle ticket new` writes.
//
// spawnedBy is the ticket's lineage (nil for none) — provenance only; it never
// gates anything.
func Scaffold(id, title, project, impact, complexity, cost string, spawnedBy []string) string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`---
id: %s
title: %s
project: %s
depends-on: []
spawned-by: %s
impact: %s
complexity: %s
cost: %s
---

# %s — %s

## Description

<!-- TODO: describe this feature in prose (the current spec). Note soft couplings to other
tickets by id; hard dependencies go in depends-on: frontmatter (human-approved). -->

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- %s — created (TO DO). source: pickle ticket new
`, id, title, project, renderIDList(spawnedBy), impact, complexity, cost, id, title, date)
}

// LoadAll reads every ticket under root/tickets/<status>/. Missing status dirs are
// treated as empty (git does not track empty dirs). The second return is a list of
// structural load problems (bad filename, no frontmatter) keyed by "<dir>/<file>".
func LoadAll(root string) ([]*Ticket, []string) {
	var tickets []*Ticket
	var issues []string
	for _, s := range Statuses {
		dir := filepath.Join(root, "tickets", s.Dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // absent (vanished-empty) dir is not an error
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			ref := s.Dir + "/" + name
			m := filenameRE.FindStringSubmatch(name)
			if m == nil {
				issues = append(issues, ref+": filename does not match T-NNN-<slug>.md")
				continue
			}
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				issues = append(issues, ref+": "+err.Error())
				continue
			}
			text := string(data)
			fm, ok := ParseFrontmatter(text)
			if !ok {
				issues = append(issues, ref+": no frontmatter block")
				continue
			}
			num, _ := strconv.Atoi(strings.TrimPrefix(m[1], "T-"))
			slug := strings.TrimSuffix(strings.TrimPrefix(name, m[1]+"-"), ".md")
			tickets = append(tickets, &Ticket{
				ID:        m[1],
				Num:       num,
				Slug:      slug,
				Dir:       s.Dir,
				Path:      path,
				Front:     fm,
				DependsOn: ParseDepends(fm["depends-on"]),
				SpawnedBy: ParseDepends(fm["spawned-by"]),
				Text:      text,
			})
		}
	}
	return tickets, issues
}
