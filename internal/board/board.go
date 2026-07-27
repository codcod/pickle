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
}

var rowRE = regexp.MustCompile(`^\|\s*(T-\d+)\s*\|`)

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

	// Status names longest-first so a prefix name can't shadow a longer one.
	names := make([]string, len(ticket.Statuses))
	for i, s := range ticket.Statuses {
		names[i] = s.Name
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })

	var rows []Row
	status, child := "", ""
	for _, line := range strings.Split(string(data), "\n") {
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
			if m := rowRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil && m[1] != "T-NNN" {
				rows = append(rows, Row{Status: status, Child: child, ID: m[1]})
			}
		}
	}
	return rows, nil
}

// SectionColumns is the ordered column list for a status section's table. It
// returns nil for an unknown section name. There is deliberately no `branch`
// column: the real branch lives in the ticket's plan and History (D2).
func SectionColumns(statusName string) []string {
	switch statusName {
	case "TO DO", "READY":
		return []string{"id", "title", "impact", "complexity", "cost", "depends-on"}
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
// a constant, not configuration. 120 was chosen against the real corpus: the
// longest cell in this repo's board was 117 runes, so nothing legitimate is
// clipped, while a full `yes — MERGED: feat/… → main (<sha>)` cell survives
// intact (T-049).
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

// sortRows orders a sub-group: TO DO/READY by descending impact (tie id asc);
// every other section by id ascending (D1 — deterministic, no hand-curated order).
func sortRows(group []*ticket.Ticket, name string) {
	byImpact := name == "TO DO" || name == "READY"
	sort.SliceStable(group, func(i, j int) bool {
		if byImpact {
			ri, rj := impactRank[group[i].Front["impact"]], impactRank[group[j].Front["impact"]]
			if ri != rj {
				return ri > rj
			}
		}
		return group[i].Num < group[j].Num
	})
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
			sortRows(group, name)

			sub := "### " + p.Name
			if name == "IN DEVELOPMENT" {
				sub = fmt.Sprintf("### %s (%d/%d)", p.Name, len(group), p.WIPInDevelopment)
			} else if name == "IN REVIEW" {
				sub = fmt.Sprintf("### %s (%d/%d)", p.Name, len(group), p.WIPInReview)
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
