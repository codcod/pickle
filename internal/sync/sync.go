// Package sync implements `pickle board sync`: the escape hatch that rebuilds
// tickets/BOARD.md from ground truth (ticket files + frontmatter + pickle.toml)
// when hand-edits drift. It fully regenerates the seven status sections while
// preserving the preamble, the trailing appendix, and human bookkeeping cells
// (DONE merged / DROPPED reason / REWORK open findings). "In sync" is defined as
// internal/audit.Audit reporting zero errors, so a successful sync leaves
// `board audit` clean.
package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// Result records the outcome of a sync.
type Result struct {
	Changed bool     // the board differs from ground truth
	Summary []string // human-readable drift (populated in dry-run and apply)
	Path    string   // board path, relative to root
}

// boardOrder is the fixed skeleton order the status sections are rendered in
// (active work first), by status display name.
var boardOrder = []string{
	"IN DEVELOPMENT", "IN REVIEW", "REWORK", "READY", "TO DO", "DONE", "DROPPED",
}

// defaultHeading is the canonical `## ` heading text for a status absent from the
// board (present ones keep their existing heading text verbatim).
var defaultHeading = map[string]string{
	"IN DEVELOPMENT": "IN DEVELOPMENT",
	"IN REVIEW":      "IN REVIEW",
	"REWORK":         "REWORK",
	"READY":          "READY (impact order, per child)",
	"TO DO":          "TO DO (impact order, per child)",
	"DONE":           "DONE",
	"DROPPED":        "DROPPED",
}

var impactRank = map[string]int{
	"low": 1, "low-medium": 2, "medium": 3, "medium-high": 4,
	"high": 5, "high-critical": 6, "critical": 7,
}

// Sync rebuilds the board for the tickets/ tree under root. With dryRun it reports
// drift without writing; otherwise it writes and runs an audit self-check.
func Sync(root string, cfg *config.Config, dryRun bool) (Result, error) {
	res := Result{Path: filepath.Join("tickets", "BOARD.md")}

	tickets, issues := ticket.LoadAll(root)
	if len(issues) > 0 {
		return res, fmt.Errorf("cannot sync while the board has load problems: %s", strings.Join(issues, "; "))
	}

	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	data, err := os.ReadFile(boardPath)
	if err != nil {
		return res, err
	}
	original := string(data)
	lines := strings.Split(original, "\n")

	carry, err := board.ParseCells(boardPath)
	if err != nil {
		return res, err
	}
	rows, err := board.Parse(boardPath)
	if err != nil {
		return res, err
	}
	listed := map[string]string{} // id -> current section
	for _, r := range rows {
		listed[r.ID] = r.Status
	}

	// --- split the board into preamble / region / appendix ---
	firstStatus, appendixStart, headings := splitBoard(lines)
	var preamble, appendix []string
	if firstStatus == -1 {
		// No status sections at all: treat the whole file as preamble, append region.
		preamble = append([]string{}, lines...)
	} else {
		preamble = append([]string{}, lines[:firstStatus]...)
		appendix = append([]string{}, lines[appendixStart:]...)
	}
	refreshLastUpdated(preamble)

	// --- build + render the region from ground truth ---
	region, desired := renderRegion(tickets, cfg, headings, listed, carry)

	newLines := append(append(append([]string{}, preamble...), region...), appendix...)
	newText := strings.Join(newLines, "\n")

	res.Summary = drift(listed, desired, newText != original)
	res.Changed = newText != original

	if dryRun || !res.Changed {
		return res, nil
	}
	if err := os.WriteFile(boardPath, []byte(newText), 0o644); err != nil {
		return res, err
	}
	// Post-op self-check: the tree must be audit-clean (surfaces a ticket-side
	// problem sync cannot fix by rewriting the board).
	if a := audit.Audit(root, cfg); len(a.Errors) > 0 {
		return res, fmt.Errorf("sync applied but board audit still reports %d error(s): %s",
			len(a.Errors), strings.Join(a.Errors, "; "))
	}
	return res, nil
}

// splitBoard locates the status region. It returns the index of the first status
// heading, the index at which the trailing appendix begins (a non-status `## `
// heading, backed up over a preceding `---` rule), and each status's existing
// heading text (name -> text after "## "). firstStatus is -1 if none is found.
func splitBoard(lines []string) (firstStatus, appendixStart int, headings map[string]string) {
	headings = map[string]string{}
	firstStatus = -1
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "## ") {
			continue
		}
		if name := matchStatus(ln); name != "" {
			if firstStatus == -1 {
				firstStatus = i
			}
			if _, ok := headings[name]; !ok {
				headings[name] = strings.TrimSpace(ln[3:])
			}
		}
	}
	if firstStatus == -1 {
		return -1, len(lines), headings
	}
	// Appendix = first non-status "## " heading after the region.
	appendixStart = len(lines)
	for i := firstStatus; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") && matchStatus(lines[i]) == "" {
			appendixStart = i
			break
		}
	}
	// Back up over a "---" rule (and the blanks between it and the heading) so the
	// separator lands in the appendix, not the regenerated region.
	if appendixStart < len(lines) {
		q := appendixStart - 1
		for q >= firstStatus && strings.TrimSpace(lines[q]) == "" {
			q--
		}
		if q >= firstStatus && strings.TrimSpace(lines[q]) == "---" {
			appendixStart = q
		}
	}
	return firstStatus, appendixStart, headings
}

// matchStatus returns the status display name a "## " heading opens, longest name
// first so a prefix cannot shadow a longer one, or "" if it is not a status.
func matchStatus(headingLine string) string {
	head := strings.ToUpper(strings.TrimSpace(headingLine[3:]))
	names := make([]string, len(ticket.Statuses))
	for i, s := range ticket.Statuses {
		names[i] = s.Name
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, n := range names {
		if strings.HasPrefix(head, n) {
			return n
		}
	}
	return ""
}

func refreshLastUpdated(preamble []string) {
	line := "Last updated: " + time.Now().Format("2006-01-02") + " (board sync)"
	for i, ln := range preamble {
		if strings.HasPrefix(ln, "Last updated:") {
			preamble[i] = line
			return
		}
	}
}

// renderRegion produces the status-section lines from ground truth and returns
// them alongside the desired id -> section map (for the drift summary).
func renderRegion(
	tickets []*ticket.Ticket, cfg *config.Config,
	headings map[string]string, listed map[string]string,
	carry map[string]map[string]string,
) ([]string, map[string]string) {
	var out []string
	desired := map[string]string{}

	for _, name := range boardOrder {
		st, _ := ticket.StatusByName(name)
		heading := headings[name]
		if heading == "" {
			heading = defaultHeading[name]
		}
		out = append(out, "## "+heading, "")

		cols := board.SectionColumns(name)
		for _, p := range cfg.Projects {
			child := p.Name
			group := ticketsFor(tickets, st, child)
			if st.Terminal {
				group = onlyListed(group, listed) // D3: terminal rows only if already on the board
			}
			sortRows(group, name)

			sub := "### " + child
			if name == "IN DEVELOPMENT" {
				sub = fmt.Sprintf("### %s (%d/%d)", child, len(group), p.WIPInDevelopment)
			} else if name == "IN REVIEW" {
				sub = fmt.Sprintf("### %s (%d/%d)", child, len(group), p.WIPInReview)
			}
			out = append(out, sub, "", board.HeaderRow(cols), board.SeparatorRow(cols))
			for _, t := range group {
				out = append(out, rowFor(t, cfg, p, name, carry))
				desired[t.ID] = name
			}
			out = append(out, "")
		}
	}
	return out, desired
}

func ticketsFor(tickets []*ticket.Ticket, st ticket.Status, child string) []*ticket.Ticket {
	var out []*ticket.Ticket
	for _, t := range tickets {
		if t.Dir == st.Dir && t.Project() == child {
			out = append(out, t)
		}
	}
	return out
}

func onlyListed(group []*ticket.Ticket, listed map[string]string) []*ticket.Ticket {
	var out []*ticket.Ticket
	for _, t := range group {
		if _, ok := listed[t.ID]; ok {
			out = append(out, t)
		}
	}
	return out
}

// sortRows orders a sub-group: TO DO/READY by descending impact (tie id asc);
// every other section by id ascending.
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

func rowFor(t *ticket.Ticket, cfg *config.Config, p config.Project, name string, carry map[string]map[string]string) string {
	prefix := config.DefaultBranchPrefix
	if p.BranchPrefix != "" {
		prefix = p.BranchPrefix
	}
	branch := prefix + t.ID + "-" + t.Slug
	d := board.RowData{
		ID:         t.ID,
		Title:      t.Front["title"],
		Impact:     t.Front["impact"],
		Complexity: t.Front["complexity"],
		Cost:       t.Front["cost"],
		DependsOn:  "[" + strings.Join(t.DependsOn, ", ") + "]",
		Branch:     branch,
	}
	switch name {
	case "DONE":
		if m := carry[t.ID]["merged"]; m != "" {
			d.Merged = m
		} else {
			d.Merged = "no — publish-gated (branch " + branch + ")"
		}
	case "DROPPED":
		d.Reason = carry[t.ID]["reason"]
	case "REWORK":
		d.Reason = carry[t.ID]["open findings"]
	}
	return board.RenderRow(name, d)
}

// drift compares the current board membership with the desired membership and
// returns a human-readable summary. reformat is true when the text changed but
// no row moved (ordering / WIP counts / spacing only).
func drift(current, desired map[string]string, changed bool) []string {
	var out []string
	ids := map[string]bool{}
	for id := range current {
		ids[id] = true
	}
	for id := range desired {
		ids[id] = true
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	for _, id := range sorted {
		c, inC := current[id]
		d, inD := desired[id]
		switch {
		case !inC && inD:
			out = append(out, fmt.Sprintf("add %s to %s", id, d))
		case inC && !inD:
			out = append(out, fmt.Sprintf("remove %s from %s (no backing ticket, or aged terminal)", id, c))
		case c != d:
			out = append(out, fmt.Sprintf("move %s: %s -> %s", id, c, d))
		}
	}
	if len(out) == 0 && changed {
		out = append(out, "reformat only (ordering / WIP counts / spacing / Last-updated)")
	}
	return out
}
