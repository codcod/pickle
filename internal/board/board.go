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
	"sort"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/atomicfile"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

// impactRank orders TO DO/READY rows by impact (highest first).
var impactRank = map[string]int{
	"low": 1, "low-medium": 2, "medium": 3, "medium-high": 4,
	"high": 5, "high-critical": 6, "critical": 7,
}

// costRank breaks impact ties by cost (T-103): cheapest first, the inverse
// direction of impactRank (which ranks highest impact first). Covers every
// legal `cost` value (ticket.LegalCost). An illegal/missing cost degrades to
// the Go zero value (rank 0) like impactRank's own degrade, but because the
// comparison direction is inverted this sorts an illegal cost *first*
// (cheapest of all) rather than last as impactRank's degrade does — unreachable
// through the normal flow (`ticket new` always writes a legal cost; `board
// audit` flags an illegal one), so left as a zero-value fallback rather than a
// special case.
var costRank = map[string]int{
	"S": 1, "S-M": 2, "M": 3, "M-L": 4,
	"L": 5, "L-XL": 6, "XL": 7,
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
func Parse(def *flow.Definition, path string) ([]Row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseText(def, string(data)), nil
}

// ParseText is Parse's underlying scan over an in-memory string, so a freshly
// rendered board (which never touches disk) can be parsed the same way as one
// read from BOARD.md — the one caller today is Compare, comparing a render
// against the file without a temp file in between.
func ParseText(def *flow.Definition, text string) []Row {
	// Status names longest-first so a prefix name can't shadow a longer one.
	states := def.States()
	names := make([]string, len(states))
	for i, s := range states {
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

// ColumnsFor is the ordered column list for a board column profile. There is
// deliberately no `branch` column: the real branch lives in the ticket's plan
// and History (D2).
//
// Every flow.ColumnProfile flow.go defines has a case here — an unrecognised
// profile falls back to the Active set rather than returning nil (T-080): a
// nil return once meant a new status rendered a headerless table, which is
// the exact hazard a flow adding a state must not be able to trigger.
func ColumnsFor(profile flow.ColumnProfile) []string {
	switch profile {
	case flow.ColumnsBacklog:
		return []string{"id", "title", "impact", "complexity", "cost", "depends-on", "family"}
	case flow.ColumnsRework:
		return []string{"id", "title", "open findings"}
	case flow.ColumnsDone:
		return []string{"id", "title", "merged"}
	case flow.ColumnsDropped:
		return []string{"id", "title", "reason"}
	case flow.ColumnsActive:
		return []string{"id", "title", "depends-on"}
	default:
		return []string{"id", "title", "depends-on"}
	}
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
func cellFor(def *flow.Definition, t *ticket.Ticket, col string) string {
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
		if m := ticket.MergeLine(def, t.Text); m != "" {
			return "yes — " + m
		}
		return "no — publish-gated"
	case "reason", "open findings":
		return ticket.LastHistoryReason(def, t.Text)
	}
	return ""
}

func renderRow(def *flow.Definition, t *ticket.Ticket, cols []string) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = sanitizeCell(cellFor(def, t, c))
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// WIPCounts tallies, per child-project name (as written in the ticket's
// frontmatter — unregistered names included, so the audit can report them;
// tickets with no project are skipped), how many tickets sit in each of def's
// WIP-limited states, keyed by that state's directory.
//
// It is the single tally behind the board's `(n/limit)` sub-headings, the
// audit's WIP-limit check, and the dashboard's WIP badges. Three independent
// counts of the same thing is three chances to disagree about whether a limit
// is breached.
func WIPCounts(def *flow.Definition, tickets []*ticket.Ticket) map[string]map[string]int {
	wipDirs := make(map[string]bool)
	for _, s := range def.WIPStates() {
		wipDirs[s.Dir] = true
	}
	counts := map[string]map[string]int{}
	for _, t := range tickets {
		p := t.Project()
		if p == "" || !wipDirs[t.Dir] {
			continue // not WIP; do not create an empty entry for it
		}
		if counts[p] == nil {
			counts[p] = map[string]int{}
		}
		counts[p][t.Dir]++
	}
	return counts
}

// Sort orders a sub-group: TO DO/READY by descending impact, then ascending
// cost (T-103), then ascending id, with families kept contiguous (T-059);
// every other section by id ascending (D1 — deterministic, no hand-curated
// order). The cost tiebreak applies wherever impact ties — both between
// different families/loose tickets tied at the same family rank, and between
// members of the same family tied on their own impact — but T-059's
// family-contiguity guarantee is unaffected either way. Exported so the
// dashboard sorts with the board's ordering rather than a copy of it (there is
// deliberately no second implementation of impactRank).
//
// byID is the whole-tree id→ticket map, needed only by the TO DO/READY branch to
// resolve a member's umbrella (which may live in another status section, so it is
// not necessarily in `group`). Other sections ignore it, and callers may pass nil
// for them.
func Sort(group []*ticket.Ticket, st flow.State, byID map[string]*ticket.Ticket) {
	byImpact := st.ImpactOrder
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
			// Families/loose tickets tied at the same impact rank: break by cost
			// (T-103) before falling back to familyKey. famCostRank resolves
			// through the umbrella, so this can never split a family apart.
			if ca, cb := famCostRank(a, byID), famCostRank(b, byID); ca != cb {
				return ca < cb
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
			// Same family, tied on own impact: break by own cost (T-103) before
			// falling back to id.
			if ci, cj := costRank[a.Front["cost"]], costRank[b.Front["cost"]]; ci != cj {
				return ci < cj
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

// famCostRank mirrors famRank for cost (T-103): a family's cost is its
// umbrella's cost when it has one, else its own. Because every member of a
// family resolves to the same rank as its umbrella, comparing famCostRank can
// reorder whole families relative to each other but can never separate a
// member from its umbrella — T-059's contiguity guarantee is an invariant of
// this resolution, not an accident of comparison order.
func famCostRank(t *ticket.Ticket, byID map[string]*ticket.Ticket) int {
	if t.Family != "" {
		if u, ok := byID[t.Family]; ok {
			return costRank[u.Front["cost"]]
		}
	}
	return costRank[t.Front["cost"]]
}

// Render produces the entire BOARD.md text from the ticket files and config —
// a pure function of its inputs (same inputs ⇒ byte-identical output). The
// whole file is generated, preamble included (D5/decision 10): a banner
// comment, the title, a short pointer paragraph, the per-child WIP-limit
// lines, `Last updated: <date>`, then the seven status sections, each child
// sub-grouped with WIP counts computed at render time. Every DONE/DROPPED
// ticket is always rendered (D4 — no aging).
func Render(def *flow.Definition, tickets []*ticket.Ticket, cfg *config.Config, date string) string {
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
	wipStates := def.WIPStates()
	for _, p := range cfg.Projects {
		var parts []string
		for _, s := range wipStates {
			limit, ok := p.WIPLimitFor(s.WIPKey)
			if !ok {
				continue
			}
			parts = append(parts, fmt.Sprintf("`%s/` ≤ %d", s.Dir, limit))
		}
		lines = append(lines, fmt.Sprintf("- `%s`: %s", p.Name, strings.Join(parts, " · ")))
	}
	lines = append(lines, "", "Last updated: "+date)

	wip := WIPCounts(def, tickets)
	byID := ticketsByID(tickets)
	for _, st := range def.BoardStates() {
		lines = append(lines, "", "## "+st.Heading)

		cols := ColumnsFor(st.Columns)
		for _, p := range cfg.Projects {
			var group []*ticket.Ticket
			for _, t := range tickets {
				if t.Dir == st.Dir && t.Project() == p.Name {
					group = append(group, t)
				}
			}
			Sort(group, st, byID)

			sub := "### " + p.Name
			if st.WIPKey != "" {
				if limit, ok := p.WIPLimitFor(st.WIPKey); ok {
					sub = fmt.Sprintf("### %s (%d/%d)", p.Name, wip[p.Name][st.Dir], limit)
				}
			}
			lines = append(lines, "", sub, "", headerRow(cols), separatorRow(cols))
			for _, t := range group {
				lines = append(lines, renderRow(def, t, cols))
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
func Compare(def *flow.Definition, current, fresh string) Drift {
	if NormalizeLastUpdated(current) == NormalizeLastUpdated(fresh) {
		return DriftNone
	}
	curCounts := map[string]int{}
	for _, r := range ParseText(def, current) {
		curCounts[rowKey(r)]++
	}
	freshCounts := map[string]int{}
	for _, r := range ParseText(def, fresh) {
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
// the one write path `ticket new`, `ticket move`, `board sync`, `project add`
// and `project remove` all share. It refuses to render over structural load
// problems rather than generating a board that silently omits the unloadable
// tickets.
//
// The write is atomic (internal/atomicfile): a concurrent reader — notably
// `pickle serve`, which re-reads the tree on every request and on its poll —
// never observes a truncated or half-written BOARD.md. Regenerate does not
// itself take the tree lock (T-101): every caller reaches it from inside an
// already-locked path (`ticket new`, `ticket move`, `board sync`, `project
// add`/`remove`), and flock is per file descriptor, so a nested acquire on a
// second descriptor from the same process would deadlock against nothing
// useful. Callers hold the tree lock.
func Regenerate(def *flow.Definition, root string, cfg *config.Config) error {
	tickets, issues := ticket.LoadAll(def, root)
	if len(issues) > 0 {
		return fmt.Errorf("cannot regenerate the board while tickets have load problems: %s",
			strings.Join(issues, "; "))
	}
	text := Render(def, tickets, cfg, time.Now().Format("2006-01-02"))
	return atomicfile.WriteFile(filepath.Join(root, "tickets", "BOARD.md"), []byte(text))
}
