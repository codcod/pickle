// Package move implements `pickle ticket move`: a ticket's status transition as
// one operation — relocate the file between tickets/<status>/ dirs, append a dated
// ## History line, and regenerate the board — behind a state machine, per-child
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
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

// Result records a completed move for the CLI summary.
type Result struct {
	From     string // source status display name
	To       string // target status display name
	Path     string // new path, relative to root
	Warnings []string
}

// Move performs the transition of ticket id to the status named by token.
func Move(root string, cfg *config.Config, id, token, reason string) (Result, error) {
	var res Result
	reason = sanitizeReason(reason)
	def := flow.ForName(cfg.FlowName())

	target, ok := def.ByToken(token)
	if !ok {
		return res, fmt.Errorf("unknown status %q", token)
	}

	tickets, issues := ticket.LoadAll(def, root)
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

	from, _ := def.ByDir(t.Dir)
	res.From, res.To = from.Name, target.Name

	if target.Dir == t.Dir {
		return res, fmt.Errorf("%s is already in %s", id, from.Name)
	}
	if _, ok := def.Kind(t.Dir, target.Dir); !ok {
		return res, fmt.Errorf("illegal transition %s → %s (from %s, legal: %s)",
			from.Name, target.Name, from.Name, legalTargets(def, t.Dir))
	}
	if def.RequiresReason(t.Dir, target.Dir) && reason == "" {
		return res, fmt.Errorf("moving %s to %s requires --reason", id, target.Name)
	}

	proj := t.Project()

	// WIP gate: moving into a WIP-limited state (in-development / in-review
	// for brine).
	if target.WIPKey != "" {
		if err := checkWIP(tickets, cfg, proj, target); err != nil {
			return res, err
		}
	}

	// Cross-child dependency + merge gate: pickup only — entering the state a
	// ticket is built in (the one keyed by config.WIPKeyInDevelopment; for
	// brine, IN DEVELOPMENT). depends-on only — spawned-by is lineage and must
	// never gate a pickup, which is what move_test.go's
	// TestSpawnedByDoesNotGatePickup guards.
	pickup, hasPickup := def.StateByWIPKey(config.WIPKeyInDevelopment)
	if hasPickup && target.Dir == pickup.Dir {
		for _, dep := range t.DependsOn {
			dt, ok := byID[dep]
			if !ok {
				return res, fmt.Errorf("cannot pick up %s: dependency %s does not exist", id, dep)
			}
			if dt.Dir != def.DependencySatisfied().Dir {
				st, _ := def.ByDir(dt.Dir)
				return res, fmt.Errorf("cannot pick up %s: dependency %s is in %s (must be %s and merged)",
					id, dep, st.Name, def.DependencySatisfied().Name)
			}
			if !ticket.HasMergeLine(def, dt.Text) {
				return res, fmt.Errorf("cannot pick up %s: dependency %s is %s but not recorded as merged to its child's base",
					id, dep, def.DependencySatisfied().Name)
			}
		}
	}

	// --- apply (all checks passed) ---
	// D7 (T-014·4): write the updated text to the NEW path first, then remove the
	// old file — never append-then-rename. A crash between the two leaves a
	// duplicate id, which the audit reports and a human recovers by deleting the
	// stale copy; it can no longer leave a ticket whose History records a
	// transition that did not happen.
	newText := appendHistory(t.Text, from.Name, target.Name, reason)
	newRel := filepath.Join("tickets", target.Dir, t.Base()+".md")
	newPath := filepath.Join(root, newRel)
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return res, err
	}
	if err := os.WriteFile(newPath, []byte(newText), 0o644); err != nil {
		return res, err
	}
	if err := os.Remove(t.Path); err != nil {
		return res, err
	}
	res.Path = newRel

	if err := board.Regenerate(def, root, cfg); err != nil {
		return res, fmt.Errorf("moved file but failed to regenerate board: %w", err)
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

func checkWIP(tickets []*ticket.Ticket, cfg *config.Config, proj string, target flow.State) error {
	p, ok := cfg.Project(proj)
	if !ok {
		return nil // unregistered project is an audit concern, not a move gate
	}
	limit, ok := p.WIPLimitFor(target.WIPKey)
	if !ok {
		return nil // target names no known WIP key — nothing to enforce
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

func legalTargets(def *flow.Definition, dir string) string {
	var names []string
	for _, st := range def.Allowed(dir) {
		names = append(names, st.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none — terminal status)"
	}
	return strings.Join(names, ", ")
}
