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

// AddTODORow inserts a ticket row into the TO DO section under the child's
// `### <child>` sub-group, in impact order (highest first; ties keep existing
// order). The sub-group is created (with a standard header) if it does not exist.
func AddTODORow(boardPath, child, id, title, impact, complexity, cost string) error {
	data, err := os.ReadFile(boardPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	newRow := fmt.Sprintf("| %s | %s | %s | %s | %s | [] |", id, title, impact, complexity, cost)

	// TO DO section span [todoStart, todoEnd).
	todoStart, todoEnd := -1, len(lines)
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "## ") {
			continue
		}
		head := strings.ToUpper(strings.TrimSpace(ln[3:]))
		if todoStart == -1 && strings.HasPrefix(head, "TO DO") {
			todoStart = i
		} else if todoStart != -1 && i > todoStart {
			todoEnd = i
			break
		}
	}
	if todoStart == -1 {
		return fmt.Errorf("%s: no TO DO section", boardPath)
	}

	// Child sub-group within the TO DO section.
	subStart := -1
	for i := todoStart + 1; i < todoEnd; i++ {
		if strings.HasPrefix(lines[i], "### ") &&
			strings.TrimSpace(strings.SplitN(lines[i][4:], " (", 2)[0]) == child {
			subStart = i
			break
		}
	}

	if subStart == -1 { // create the sub-group at the end of the TO DO section
		block := []string{
			"",
			"### " + child,
			"",
			"| id | title | impact | complexity | cost | depends-on |",
			"|---|---|---|---|---|---|",
			newRow,
		}
		return write(boardPath, insertLines(lines, todoEnd, block))
	}

	// Sub-group span [subStart, subEnd).
	subEnd := todoEnd
	for i := subStart + 1; i < todoEnd; i++ {
		if strings.HasPrefix(lines[i], "### ") || strings.HasPrefix(lines[i], "## ") {
			subEnd = i
			break
		}
	}

	newRank := impactRank[impact]
	insertAt, lastRow := -1, -1
	for i := subStart + 1; i < subEnd; i++ {
		m := rowRE.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if m == nil || m[1] == "T-NNN" {
			continue
		}
		lastRow = i
		cells := strings.Split(lines[i], "|")
		rowImpact := ""
		if len(cells) > 3 {
			rowImpact = strings.TrimSpace(cells[3])
		}
		if insertAt == -1 && impactRank[rowImpact] < newRank {
			insertAt = i
		}
	}
	if insertAt == -1 {
		if lastRow != -1 {
			insertAt = lastRow + 1
		} else { // empty sub-group: skip blank/header/separator lines
			insertAt = subStart + 1
			for insertAt < subEnd {
				t := strings.TrimSpace(lines[insertAt])
				if t == "" || strings.HasPrefix(t, "| id") || strings.HasPrefix(t, "|---") {
					insertAt++
					continue
				}
				break
			}
		}
	}
	return write(boardPath, insertLines(lines, insertAt, []string{newRow}))
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
