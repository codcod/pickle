// Package move implements `pickle ticket move`: a ticket's status transition as
// one operation — relocate the file between tickets/<status>/ dirs, append a dated
// ## History line, and rewrite the board row — behind a state machine, per-child
// WIP limits, sign-off (--reason) rules, and a cross-child dependency+merge gate.
// A completed move leaves internal/audit.Audit reporting zero errors.
package move

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

// Result records a completed move for the CLI summary.
type Result struct {
	From     string // source status display name
	To       string // target status display name
	Path     string // new path, relative to root
	Warnings []string
}

// allowed maps a source status dir to the target dirs it may transition to.
var allowed = map[string][]string{
	"1-to-do":          {"2-ready", "7-dropped"},
	"2-ready":          {"3-in-development", "1-to-do", "7-dropped"},
	"3-in-development": {"4-in-review", "2-ready", "7-dropped"},
	"4-in-review":      {"6-done", "5-rework", "7-dropped"},
	"5-rework":         {"4-in-review", "7-dropped"},
	// 6-done and 7-dropped are terminal: no outgoing transitions.
}

// requiresReason reports whether a from->to transition needs a --reason: every
// abort (-> dropped), every send-back (-> rework), and the two backward moves.
// Forward progress moves may omit it.
func requiresReason(from, to string) bool {
	switch to {
	case "5-rework", "7-dropped":
		return true
	}
	return (from == "3-in-development" && to == "2-ready") ||
		(from == "2-ready" && to == "1-to-do")
}

// Move performs the transition of ticket id to the status named by token.
func Move(root string, cfg *config.Config, id, token, reason string) (Result, error) {
	var res Result
	reason = sanitizeReason(reason)

	target, ok := ticket.StatusByToken(token)
	if !ok {
		return res, fmt.Errorf("unknown status %q", token)
	}

	tickets, issues := ticket.LoadAll(root)
	if len(issues) > 0 {
		return res, fmt.Errorf("cannot move while the board has load problems: %s", strings.Join(issues, "; "))
	}
	byID := make(map[string]*ticket.Ticket, len(tickets))
	for _, t := range tickets {
		byID[t.ID] = t
	}
	t, ok := byID[id]
	if !ok {
		return res, fmt.Errorf("ticket %s not found", id)
	}

	from, _ := ticket.StatusByDir(t.Dir)
	res.From, res.To = from.Name, target.Name

	if target.Dir == t.Dir {
		return res, fmt.Errorf("%s is already in %s", id, from.Name)
	}
	if !contains(allowed[t.Dir], target.Dir) {
		return res, fmt.Errorf("illegal transition %s → %s (from %s, legal: %s)",
			from.Name, target.Name, from.Name, legalTargets(t.Dir))
	}
	if requiresReason(t.Dir, target.Dir) && reason == "" {
		return res, fmt.Errorf("moving %s to %s requires --reason", id, target.Name)
	}

	proj := t.Project()

	// WIP gate: moving into in-development / in-review.
	if target.Dir == "3-in-development" || target.Dir == "4-in-review" {
		if err := checkWIP(tickets, cfg, proj, target); err != nil {
			return res, err
		}
	}

	// Cross-child dependency + merge gate: pickup only.
	if target.Dir == "3-in-development" {
		for _, dep := range t.DependsOn {
			dt, ok := byID[dep]
			if !ok {
				return res, fmt.Errorf("cannot pick up %s: dependency %s does not exist", id, dep)
			}
			if dt.Dir != "6-done" {
				st, _ := ticket.StatusByDir(dt.Dir)
				return res, fmt.Errorf("cannot pick up %s: dependency %s is in %s (must be DONE and merged)", id, dep, st.Name)
			}
			if !ticket.HasMergeLine(dt.Text) {
				return res, fmt.Errorf("cannot pick up %s: dependency %s is DONE but not recorded as merged to its child's base", id, dep)
			}
		}
	}

	// --- apply (all checks passed) ---
	newText := appendHistory(t.Text, from.Name, target.Name, reason)
	if err := os.WriteFile(t.Path, []byte(newText), 0o644); err != nil {
		return res, err
	}
	newRel := filepath.Join("tickets", target.Dir, t.Base()+".md")
	newPath := filepath.Join(root, newRel)
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return res, err
	}
	if err := os.Rename(t.Path, newPath); err != nil {
		return res, err
	}
	res.Path = newRel

	if err := board.MoveRow(filepath.Join(root, "tickets", "BOARD.md"), target.Name, proj,
		rowData(t, cfg, target, reason)); err != nil {
		return res, fmt.Errorf("moved file but failed to update board: %w", err)
	}

	// Post-move self-check: the tree must stay audit-clean.
	a := audit.Audit(root, cfg)
	res.Warnings = a.Warnings
	if len(a.Errors) > 0 {
		return res, fmt.Errorf("move applied but board audit now reports %d error(s): %s",
			len(a.Errors), strings.Join(a.Errors, "; "))
	}
	return res, nil
}

func rowData(t *ticket.Ticket, cfg *config.Config, target ticket.Status, reason string) board.RowData {
	d := board.RowData{
		ID:         t.ID,
		Title:      t.Front["title"],
		Impact:     t.Front["impact"],
		Complexity: t.Front["complexity"],
		Cost:       t.Front["cost"],
		DependsOn:  renderDepends(t.DependsOn),
		Reason:     reason,
	}
	prefix := config.DefaultBranchPrefix
	if p, ok := cfg.Project(t.Project()); ok && p.BranchPrefix != "" {
		prefix = p.BranchPrefix
	}
	d.Branch = prefix + t.ID + "-" + t.Slug
	if target.Dir == "6-done" {
		d.Merged = "no — publish-gated (branch " + d.Branch + ")"
	}
	return d
}

func checkWIP(tickets []*ticket.Ticket, cfg *config.Config, proj string, target ticket.Status) error {
	p, ok := cfg.Project(proj)
	if !ok {
		return nil // unregistered project is an audit concern, not a move gate
	}
	limit := p.WIPInDevelopment
	if target.Dir == "4-in-review" {
		limit = p.WIPInReview
	}
	count := 0
	for _, t := range tickets {
		if t.Project() == proj && t.Dir == target.Dir {
			count++
		}
	}
	if count+1 > limit {
		return fmt.Errorf("WIP: child %q already at its %s limit (%d)", proj, target.Name, limit)
	}
	return nil
}

// appendHistory inserts a dated transition bullet at the end of the ## History
// section (before the next ## heading, or EOF).
func appendHistory(text, from, to, reason string) string {
	date := time.Now().Format("2006-01-02")
	line := fmt.Sprintf("- %s — %s → %s", date, from, to)
	if reason != "" {
		line += ": " + reason
	}

	lines := strings.Split(text, "\n")
	histStart := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "## History" {
			histStart = i
			break
		}
	}
	if histStart == -1 { // no History section: append one
		out := strings.TrimRight(text, "\n")
		return out + "\n\n## History\n\n" + line + "\n"
	}
	// End of the History section: next "## " heading, or EOF.
	end := len(lines)
	for i := histStart + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	// Insert after the last non-empty line within (histStart, end).
	insertAt := histStart + 1
	for i := histStart + 1; i < end; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			insertAt = i + 1
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, line)
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n")
}

// sanitizeReason strips newlines and neutralises the transition arrow so a reason
// can never be mis-read by ticket.LastHistoryStatus (see the reason-parse guard).
func sanitizeReason(reason string) string {
	r := strings.TrimSpace(reason)
	r = strings.ReplaceAll(r, "\r", " ")
	r = strings.ReplaceAll(r, "\n", " ")
	r = strings.ReplaceAll(r, "→", "->")
	return strings.TrimSpace(r)
}

func renderDepends(deps []string) string {
	return "[" + strings.Join(deps, ", ") + "]"
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func legalTargets(dir string) string {
	var names []string
	for _, d := range allowed[dir] {
		st, _ := ticket.StatusByDir(d)
		names = append(names, st.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none — terminal status)"
	}
	return strings.Join(names, ", ")
}
