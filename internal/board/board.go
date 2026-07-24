// Package board parses BOARD.md into the rows the audit cross-checks against the
// ticket files: which ticket id is listed under which status section and which
// child sub-group.
package board

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"pickle/internal/ticket"
)

// impactRank orders TO DO rows by impact (highest first).
var impactRank = map[string]int{
	"low": 1, "low-medium": 2, "medium": 3, "medium-high": 4,
	"high": 5, "high-critical": 6, "critical": 7,
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
// match first) and the current `### <child>` sub-heading. The `T-NNN` template
// placeholder row is ignored.
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

// RowData carries the section-specific cells a board row can hold. `ticket move`
// (and, later, `board sync`) fill the fields relevant to the target section; the
// section's column list decides which are rendered.
type RowData struct {
	ID         string
	Title      string
	Impact     string
	Complexity string
	Cost       string
	DependsOn  string // "[T-002]" or "[]"
	Branch     string // "feat/T-007-ticket-move"
	Merged     string // DONE column
	Reason     string // DROPPED reason / REWORK open findings
}

func (d RowData) cell(col string) string {
	switch col {
	case "id":
		return d.ID
	case "title":
		return d.Title
	case "impact":
		return d.Impact
	case "complexity":
		return d.Complexity
	case "cost":
		return d.Cost
	case "depends-on":
		return d.DependsOn
	case "branch":
		return d.Branch
	case "merged":
		return d.Merged
	case "reason", "open findings":
		return d.Reason
	}
	return ""
}

// SectionColumns is the ordered column list for a status section's table. It
// returns nil for an unknown section name.
func SectionColumns(statusName string) []string {
	switch statusName {
	case "TO DO", "READY":
		return []string{"id", "title", "impact", "complexity", "cost", "depends-on"}
	case "IN DEVELOPMENT", "IN REVIEW":
		return []string{"id", "title", "branch", "depends-on"}
	case "REWORK":
		return []string{"id", "title", "branch", "open findings"}
	case "DONE":
		return []string{"id", "title", "merged"}
	case "DROPPED":
		return []string{"id", "title", "reason"}
	}
	return nil
}

func renderRow(cols []string, d RowData) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = d.cell(c)
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// AddTODORow inserts a ticket row into the TO DO section under the child's
// `### <child>` sub-group, in impact order (highest first; ties keep existing
// order). The sub-group is created (with a standard header) if it does not exist.
// It is a thin wrapper over the shared section insert used by MoveRow.
func AddTODORow(boardPath, child, id, title, impact, complexity, cost string) error {
	return insertIntoBoard(boardPath, "TO DO", child, RowData{
		ID: id, Title: title, Impact: impact, Complexity: complexity, Cost: cost, DependsOn: "[]",
	})
}

// MoveRow removes any existing row for d.ID from the board and inserts a freshly
// rendered row into the target status section under the ticket's `### <child>`
// sub-group (created if absent). TO DO/READY insert in descending-impact order;
// every other section appends. Now-empty sub-groups are left in place.
func MoveRow(boardPath, statusName, child string, d RowData) error {
	return insertIntoBoard(boardPath, statusName, child, d)
}

func insertIntoBoard(boardPath, statusName, child string, d RowData) error {
	cols := SectionColumns(statusName)
	if cols == nil {
		return fmt.Errorf("unknown board section %q", statusName)
	}
	data, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}
	lines := removeRowByID(strings.Split(string(data), "\n"), d.ID)
	row := renderRow(cols, d)
	impactOrdered := statusName == "TO DO" || statusName == "READY"

	secStart, secEnd := sectionSpan(lines, statusName)
	if secStart == -1 {
		return fmt.Errorf("%s: no %s section", boardPath, statusName)
	}
	subStart, subEnd := subgroupSpan(lines, secStart, secEnd, child)
	if subStart == -1 { // create the sub-group at the end of the section
		sep := "|" + strings.Repeat("---|", len(cols))
		block := []string{"", "### " + child, "", "| " + strings.Join(cols, " | ") + " |", sep, row}
		return write(boardPath, insertLines(lines, secEnd, block))
	}

	newRank := impactRank[d.Impact]
	insertAt, lastRow := -1, -1
	for i := subStart + 1; i < subEnd; i++ {
		m := rowRE.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if m == nil || m[1] == "T-NNN" {
			continue
		}
		lastRow = i
		if impactOrdered && insertAt == -1 {
			cells := strings.Split(lines[i], "|")
			rowImpact := ""
			if len(cells) > 3 {
				rowImpact = strings.TrimSpace(cells[3])
			}
			if impactRank[rowImpact] < newRank {
				insertAt = i
			}
		}
	}
	if insertAt == -1 {
		if lastRow != -1 {
			insertAt = lastRow + 1
		} else { // empty sub-group: insert right after the header separator line
			insertAt = subStart + 1
			for i := subStart + 1; i < subEnd; i++ {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "|---") {
					insertAt = i + 1
				}
			}
		}
	}
	return write(boardPath, insertLines(lines, insertAt, []string{row}))
}

// removeRowByID drops any board row whose id equals id (in any section). Empty
// sub-groups are intentionally left in place (matching the skeleton convention).
func removeRowByID(lines []string, id string) []string {
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if m := rowRE.FindStringSubmatch(strings.TrimSpace(ln)); m != nil && m[1] == id {
			continue
		}
		out = append(out, ln)
	}
	return out
}

// sectionSpan returns [start,end) line indices for the "## <name>..." section, or
// (-1, len) if the section is absent. name is the upper-case status display name.
func sectionSpan(lines []string, name string) (int, int) {
	start, end := -1, len(lines)
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "## ") {
			continue
		}
		head := strings.ToUpper(strings.TrimSpace(ln[3:]))
		if start == -1 && strings.HasPrefix(head, name) {
			start = i
		} else if start != -1 && i > start {
			return start, i
		}
	}
	return start, end
}

// subgroupSpan returns [start,end) for the "### <child>" sub-group inside
// [secStart,secEnd), or (-1,-1) if absent.
func subgroupSpan(lines []string, secStart, secEnd int, child string) (int, int) {
	start := -1
	for i := secStart + 1; i < secEnd; i++ {
		if strings.HasPrefix(lines[i], "### ") &&
			strings.TrimSpace(strings.SplitN(lines[i][4:], " (", 2)[0]) == child {
			start = i
			break
		}
	}
	if start == -1 {
		return -1, -1
	}
	end := secEnd
	for i := start + 1; i < secEnd; i++ {
		if strings.HasPrefix(lines[i], "### ") || strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return start, end
}

func insertLines(lines []string, at int, block []string) []string {
	out := make([]string, 0, len(lines)+len(block))
	out = append(out, lines[:at]...)
	out = append(out, block...)
	out = append(out, lines[at:]...)
	return out
}

func write(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}
