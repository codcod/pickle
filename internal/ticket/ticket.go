// Package ticket is the shared model for ticket files: the board status table,
// frontmatter parsing, History interpretation, and loading a whole tickets/ tree.
// Both the board audit (internal/audit) and the project remove-guard (internal/cli)
// build on it, so the frontmatter scan lives in exactly one place.
package ticket

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Status is a board status with its directory, display name, and terminal flag.
type Status struct {
	Dir      string
	Name     string
	Terminal bool
}

// Statuses lists every status in board order. Terminal statuses (done/dropped)
// may age off the board.
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

// Ticket is one parsed ticket file.
type Ticket struct {
	ID        string            // "T-001" (from the filename)
	Num       int               // 1
	Slug      string            // "config-and-project-registry"
	Dir       string            // status directory, e.g. "1-to-do"
	Path      string            // absolute path
	Front     map[string]string // raw frontmatter values (title kept verbatim)
	DependsOn []string          // parsed depends-on ids
	Text      string            // full file text
}

// Base is the filename without extension, e.g. "T-001-config-and-project-registry".
func (t *Ticket) Base() string { return strings.TrimSuffix(filepath.Base(t.Path), ".md") }

// Project is the project: frontmatter value.
func (t *Ticket) Project() string { return t.Front["project"] }

var (
	filenameRE = regexp.MustCompile(`^(T-\d+)-[A-Za-z0-9._-]+\.md$`)
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

// ParseDepends parses a depends-on value like "[T-001, T-002]" or "[]".
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

// HasMergeLine reports whether the History records a merge ("… — MERGED: …").
func HasMergeLine(text string) bool {
	inHistory := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			inHistory = strings.TrimSpace(line) == "## History"
			continue
		}
		if !inHistory {
			continue
		}
		if m := historyRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			if mergedRE.MatchString(strings.TrimSpace(m[1])) {
				return true
			}
		}
	}
	return false
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
				Text:      text,
			})
		}
	}
	return tickets, issues
}
