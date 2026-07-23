// Package board parses BOARD.md into the rows the audit cross-checks against the
// ticket files: which ticket id is listed under which status section and which
// child sub-group.
package board

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"pickle/internal/ticket"
)

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
