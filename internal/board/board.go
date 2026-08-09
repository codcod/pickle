// Package board renders tickets/BOARD.md as a pure generated artifact of the
// ticket files (the single source of truth) and parses it read-only for the
// sync drift summary. Nothing ever parses board cells back into data: every
// cell passes one-way through sanitizeCell at render time (T-044).
package board

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// impactRank orders TO DO/READY rows by impact (highest first).
var impactRank = map[string]int{
	"low": 1, "low-medium": 2, "medium": 3, "medium-high": 4,
	"high": 5, "high-critical": 6, "critical": 7,
}

// boardOrder is the fixed order the status sections are rendered in (active
// work first), by status display name.
var boardOrder = []string{
	"IN DEVELOPMENT", "IN REVIEW", "REWORK", "READY", "TO DO", "DONE", "DROPPED",
}

// sectionHeading is the canonical `## ` heading text per status.
var sectionHeading = map[string]string{
	"IN DEVELOPMENT": "IN DEVELOPMENT",
	"IN REVIEW":      "IN REVIEW",
	"REWORK":         "REWORK",
	"READY":          "READY (impact order, per child)",
	"TO DO":          "TO DO (impact order, per child)",
	"DONE":           "DONE",
	"DROPPED":        "DROPPED",
}

// Row is one ticket listed on the board.
type Row struct {
	Status string // status display name, e.g. "TO DO"
	Child  string // child sub-group, e.g. "pickle" ("" if listed outside any sub-group)
	ID     string // "T-001"
	Line   string // the raw, trimmed row line as it appears — opaque text, never parsed further
}

var rowRE = regexp.MustCompile(`^\|\s*([A-Z][A-Z0-9]*-\d+)\s*\|`)

// Parse reads BOARD.md and returns every ticket row with its section + sub-group.
// Rows are attributed to the current `## <status>` heading (longest status-name
// match first) and the current `### <child>` sub-heading. This is read-only
// membership parsing for the sync drift summary — cell contents are never read
// back (they are sanitised one-way at render time).
func Parse(path string) ([]Row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseText(string(data)), nil
}

// ParseText is Parse's underlying scan over an in-memory string, so a freshly
// rendered board (which never touches disk) can be parsed the same way as one
// read from BOARD.md — the one caller today is Compare, comparing a render
// against the file without a temp file in between.
func ParseText(text string) []Row {
	// Status names longest-first so a prefix name can't shadow a longer one.
	names := make([]string, len(ticket.Statuses))
	for i, s := range ticket.Statuses {
		names[i] = s.Name
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })

	var rows []Row
	status, child := "", ""
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			heading := strings.ToUpper(strings.TrimSpace(line[3:]))
			status, child = "", ""
			for _, n := range names {
				if strings.HasPrefix(heading, n) {
					status = n
					break
				}
			}
		case strings.HasPrefix(line, "### "):
			rest := strings.TrimSpace(line[4:])
			child = strings.TrimSpace(strings.SplitN(rest, " (", 2)[0])
		default:
			if status == "" {
				continue
			}
			trimmed := strings.TrimSpace(line)
			if m := rowRE.FindStringSubmatch(trimmed); m != nil && m[1] != "T-NNN" {
				rows = append(rows, Row{Status: status, Child: child, ID: m[1], Line: trimmed})
			}
		}
	}
	return rows
}

// SectionColumns is the ordered column list for a status section's table. It
// returns nil for an unknown section name. There is deliberately no `branch`
// column: the real branch lives in the ticket's plan and History (D2).
func SectionColumns(statusName string) []string {
	switch statusName {
	case "TO DO", "READY":
		return []string{"id", "title", "impact", "complexity", "cost", "depends-on", "family"}
	case "IN DEVELOPMENT", "IN REVIEW":
		return []string{"id", "title", "depends-on"}
	case "REWORK":
		return []string{"id", "title", "open findings"}
	case "DONE":
		return []string{"id", "title", "merged"}
	case "DROPPED":
		return []string{"id", "title", "reason"}
	}
	return nil
}

var cellBreakRE = regexp.MustCompile(`[\r\n]+`)

// maxCellRunes bounds a rendered cell for legibility: one over-long value in
// one ticket must not make a whole status table unreadable (a migrated ticket
// with a paragraph-long merge History line produced a ~1,900-rune DONE cell).
// It is a render-time bound only — the ticket file keeps the full text, which
// stays the single source of truth (T-044 decision 3) — and it is deliberately
// a constant, not configuration. 120 was chosen against the real corpus of the
// time: the longest cell in this repo's board was 117 runes, and a full
// `yes — MERGED: feat/… → main (<sha>)` cell survived intact (T-049).
//
// That headroom no longer covers every recommended merge line. T-089 recommends a
// commit *link* alongside the MR ref, and the full form reaches the cap exactly
// (GitHub, 120) or exceeds it (GitLab subgroup paths, 122 — clipped with an
// ellipsis, so the board shows a dead URL). This is working as designed rather
// than a defect: the ticket file keeps the whole line, and `pickle serve` renders
// it uncapped and linkified. The short SHA is what reliably survives here, which
// is why the rules recommend it *alongside* — not instead of — the link.
const maxCellRunes = 120

// sanitizeCell is the single one-way choke point every rendered cell passes
// through: pipes become a broken bar (so a title can never split a table row),
// newline runs collapse to one space, the result is trimmed, and finally it is
// capped at maxCellRunes with a trailing ellipsis. Nothing ever parses a cell
// back, so there is no escape scheme to keep in sync (T-044 decision 9) — which
// is also why truncating here is safe.
//
// The cap counts runes, never bytes: a byte-length slice would cut mid-rune on
// any multi-byte content (including the ¦ substituted above) and emit U+FFFD
// into the board. Because `|` → `¦` is a one-for-one *rune* substitution it
// cannot change the rune count, so the cap's position relative to it is
// immaterial; it runs last only so that trimming cannot re-widen a capped cell.
func sanitizeCell(s string) string {
	s = cellBreakRE.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "|", "¦")
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > maxCellRunes {
		// Head-preserving: the ellipsis is inside the budget, so a rendered
		// cell is never wider than maxCellRunes.
		s = string(r[:maxCellRunes-1]) + "…"
	}
	return s
}

func headerRow(cols []string) string {
	return "| " + strings.Join(cols, " | ") + " |"
}

func separatorRow(cols []string) string {
	return "|" + strings.Repeat("---|", len(cols))
}

// cellFor derives one cell's value from the ticket itself — never from a
// previous board (D3: terminal cells come from History; D2: no branch cell).
func cellFor(t *ticket.Ticket, col string) string {
	switch col {
	case "id":
		return t.ID
	case "title":
		return t.Front["title"]
	case "impact", "complexity", "cost":
		return t.Front[col]
	case "depends-on":
		return "[" + strings.Join(t.DependsOn, ", ") + "]"
	case "family":
		return t.Front["family"] // empty for an umbrella or a loose ticket
	case "merged":
		if m := ticket.MergeLine(t.Text); m != "" {
			return "yes — " + m
		}
		return "no — publish-gated"
	case "reason", "open findings":
		return ticket.LastHistoryReason(t.Text)
	}
	return ""
}

func renderRow(t *ticket.Ticket, cols []string) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = sanitizeCell(cellFor(t, c))
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// StatusOrder returns the fixed order the board renders its status sections in
// (active work first). It is a copy: the dashboard (internal/serve) walks the
// same order so the two views can never disagree, and a caller must not be able
// to reorder the board by mutating the slice it was handed.
func StatusOrder() []string {
	return slices.Clone(boardOrder)
}

// WIP is one child-project's live work-in-progress tally.
type WIP struct {
	InDevelopment int
	InReview      int
}

// WIPCounts tallies in-development/in-review tickets per project name, as the
// name is written in the ticket's frontmatter — unregistered names included, so
// the audit can report them; tickets with no project are skipped.
//
// It is the single tally behind the board's `(n/limit)` sub-headings, the audit's
// WIP-limit check, and the dashboard's WIP badges. Three independent counts of
// the same thing is three chances to disagree about whether a limit is breached.
func WIPCounts(tickets []*ticket.Ticket) map[string]WIP {
	counts := map[string]WIP{}
	for _, t := range tickets {
		p := t.Project()
		if p == "" {
			continue
		}
		c := counts[p]
		switch t.Dir {
		case "3-in-development":
			c.InDevelopment++
		case "4-in-review":
			c.InReview++
		default:
			continue // not WIP; do not create an empty entry for it
		}
		counts[p] = c
	}
	return counts
}

// Sort orders a sub-group: TO DO/READY by descending impact (tie id asc), with
// families kept contiguous (T-059); every other section by id ascending (D1 —
// deterministic, no hand-curated order). Exported so the dashboard sorts with the
// board's ordering rather than a copy of it (there is deliberately no second
// implementation of impactRank).
//
// byID is the whole-tree id→ticket map, needed only by the TO DO/READY branch to
// resolve a member's umbrella (which may live in another status section, so it is
// not necessarily in `group`). Other sections ignore it, and callers may pass nil
// for them.
func Sort(group []*ticket.Ticket, name string, byID map[string]*ticket.Ticket) {
	byImpact := name == "TO DO" || name == "READY"
	sort.SliceStable(group, func(i, j int) bool {
		a, b := group[i], group[j]
		if byImpact {
			// A ticket sorts under its family: the umbrella when it has one, else
			// itself (a loose ticket is its own singleton family). famRank is the
			// umbrella's impact, so a whole family floats to where its umbrella
			// ranks; loose tickets interleave by their own impact.
			fa, fb := familyKey(a), familyKey(b)
			ra, rb := famRank(a, byID), famRank(b, byID)
			if ra != rb {
				return ra > rb
			}
			if fa != fb {
				return fa < fb // group families/loose deterministically by umbrella id
			}
			// Same family: umbrella first (its `family` is empty), then members by
			// their own impact descending.
			if ua, ub := a.Family == "", b.Family == ""; ua != ub {
				return ua
			}
			if ri, rj := impactRank[a.Front["impact"]], impactRank[b.Front["impact"]]; ri != rj {
				return ri > rj
			}
		}
		return a.Num < b.Num
	})
}

// ticketsByID indexes the whole tree by id for umbrella lookups during sorting.
// Later duplicate ids (the audit's job to reject) are overwritten last-wins; the
// board still renders rather than crashing on a malformed tree.
func ticketsByID(tickets []*ticket.Ticket) map[string]*ticket.Ticket {
	byID := make(map[string]*ticket.Ticket, len(tickets))
	for _, t := range tickets {
		byID[t.ID] = t
	}
	return byID
}

// familyKey is the id a ticket sorts under: its umbrella when set, else its own id
// (loose ticket = singleton family). Used only to keep a family's rows contiguous.
func familyKey(t *ticket.Ticket) string {
	if t.Family != "" {
		return t.Family
	}
	return t.ID
}

// famRank is the impact rank a ticket sorts by: its umbrella's impact when it has a
// family, else its own. An unresolved umbrella (audit-dirty tree, or byID nil)
// falls back to the ticket's own impact so the board still renders.
func famRank(t *ticket.Ticket, byID map[string]*ticket.Ticket) int {
	if t.Family != "" {
		if u, ok := byID[t.Family]; ok {
			return impactRank[u.Front["impact"]]
		}
	}
	return impactRank[t.Front["impact"]]
}

// Render produces the entire BOARD.md text from the ticket files and config —
// a pure function of its inputs (same inputs ⇒ byte-identical output). The
// whole file is generated, preamble included (D5/decision 10): a banner
// comment, the title, a short pointer paragraph, the per-child WIP-limit
// lines, `Last updated: <date>`, then the seven status sections, each child
// sub-grouped with WIP counts computed at render time. Every DONE/DROPPED
// ticket is always rendered (D4 — no aging).
func Render(tickets []*ticket.Ticket, cfg *config.Config, date string) string {
	lines := []string{
		"<!-- generated by pickle — do not edit; run pickle board sync -->",
		"",
		"# Board",
		"",
		"This file is generated from the ticket files (the single source of truth) by",
		"`pickle ticket new`, `pickle ticket move` and `pickle board sync`. Do not edit it by",
		"hand — edit the tickets. Hand-written planning notes live in [`NOTES.md`](NOTES.md).",
		"",
		"**WIP limits (per child-project):**",
	}
	for _, p := range cfg.Projects {
		lines = append(lines, fmt.Sprintf("- `%s`: `3-in-development/` ≤ %d · `4-in-review/` ≤ %d",
			p.Name, p.WIPInDevelopment, p.WIPInReview))
	}
	lines = append(lines, "", "Last updated: "+date)

	wip := WIPCounts(tickets)
	byID := ticketsByID(tickets)
	for _, name := range boardOrder {
		st, _ := ticket.StatusByName(name)
		lines = append(lines, "", "## "+sectionHeading[name])

		cols := SectionColumns(name)
		for _, p := range cfg.Projects {
			var group []*ticket.Ticket
			for _, t := range tickets {
				if t.Dir == st.Dir && t.Project() == p.Name {
					group = append(group, t)
				}
			}
			Sort(group, name, byID)

			sub := "### " + p.Name
			if name == "IN DEVELOPMENT" {
				sub = fmt.Sprintf("### %s (%d/%d)", p.Name, wip[p.Name].InDevelopment, p.WIPInDevelopment)
			} else if name == "IN REVIEW" {
				sub = fmt.Sprintf("### %s (%d/%d)", p.Name, wip[p.Name].InReview, p.WIPInReview)
			}
			lines = append(lines, "", sub, "", headerRow(cols), separatorRow(cols))
			for _, t := range group {
				lines = append(lines, renderRow(t, cols))
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// NormalizeLastUpdated blanks the date on the `Last updated:` line so two
// renders (or a render and the file on disk) can be compared for real drift.
func NormalizeLastUpdated(text string) string {
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(ln, "Last updated:") {
			lines[i] = "Last updated:"
		}
	}
	return strings.Join(lines, "\n")
}

// Drift classifies how a board on disk (current) differs from a fresh render
// (fresh) — see Compare.
type Drift int

const (
	// DriftNone: current and fresh agree (aside from the `Last updated:` date).
	DriftNone Drift = iota
	// DriftLayout: every ticket row is identical (same status section, same
	// child sub-group, same rendered cells) between current and fresh — only
	// generated scaffolding around the rows differs: the preamble, the
	// per-child WIP-limit lines, a `### <child>` sub-heading or its `(n/limit)`
	// count, an empty child table, row order within a section, or plain
	// spacing. Nothing here misinforms about which ticket is where, so this is
	// advisory (a warning), not a rule violation.
	DriftLayout
	// DriftRows: at least one ticket row disagrees — added, removed, moved to
	// a different status section or child sub-group, or its rendered cell text
	// changed. The board is telling a reader something the tickets do not
	// support, so this stays an error.
	DriftRows
)

// rowKey joins a Row's section, child sub-group and rendered line into one
// comparison key. \x00 cannot appear in a rendered cell (sanitizeCell strips
// newlines and pipes are substituted, but a NUL byte is never produced by any
// ticket field pickle accepts), so it cannot collide across the three parts.
func rowKey(r Row) string {
	return r.Status + "\x00" + r.Child + "\x00" + r.Line
}

// Compare classifies the drift between a board's current text and a fresh
// render of the same tree (current, fresh — same argument order as a diff).
// It never parses a cell back into ticket data (T-044 decision 9): both sides
// are reduced to ParseText's opaque row lines, keyed by (status, child, raw
// line) as a multiset — row *order* is deliberately not part of the key, so a
// renderer change that only reorders identical rows (e.g. a new sort rule)
// counts as layout drift, not row drift; see decision 4. A key present a
// different number of times on either side (a duplicated or missing row) is
// still caught, because the counts must match too.
//
// This is the one invariant `pickle board audit` enforces on BOARD.md: if
// every rendered ticket row still matches, the file is stale only in its
// generated layout (DriftLayout, a warning) — if any row disagrees, the file
// is telling a reader something the tickets do not support (DriftRows, an
// error). The distinction is about *harm*, not cause: nothing here can tell
// a hand-edit apart from a renderer change, and it does not try to.
func Compare(current, fresh string) Drift {
	if NormalizeLastUpdated(current) == NormalizeLastUpdated(fresh) {
		return DriftNone
	}
	curCounts := map[string]int{}
	for _, r := range ParseText(current) {
		curCounts[rowKey(r)]++
	}
	freshCounts := map[string]int{}
	for _, r := range ParseText(fresh) {
		freshCounts[rowKey(r)]++
	}
	if len(curCounts) != len(freshCounts) {
		return DriftRows
	}
	for k, n := range curCounts {
		if freshCounts[k] != n {
			return DriftRows
		}
	}
	return DriftLayout
}

// Regenerate renders the board from the ticket tree under root and writes it —
// the one write path `ticket new`, `ticket move` and `board sync` all share.
// It refuses to render over structural load problems rather than generating a
// board that silently omits the unloadable tickets.
func Regenerate(root string, cfg *config.Config) error {
	tickets, issues := ticket.LoadAll(root)
	if len(issues) > 0 {
		return fmt.Errorf("cannot regenerate the board while tickets have load problems: %s",
			strings.Join(issues, "; "))
	}
	text := Render(tickets, cfg, time.Now().Format("2006-01-02"))
	return os.WriteFile(filepath.Join(root, "tickets", "BOARD.md"), []byte(text), 0o644)
}
