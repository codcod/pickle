package serve

import (
	"html/template"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

// The view layer is pure: every builder takes already-loaded tickets plus the
// config and returns data for a template. Nothing here opens a file for writing,
// and nothing reads tickets/BOARD.md — the ticket files are the source of truth
// (T-044), and the board's own freshness is reported by the audit banner instead.

// activityCap bounds the timeline. The page is a "what changed lately" view, not
// an archive: an old project has thousands of History lines and rendering all of
// them helps nobody. The total is always reported, so a truncated list says so.
const activityCap = 200

// Entry is one ticket as the board and timeline views show it.
type Entry struct {
	ID         string
	Num        int
	Title      string
	Project    string
	Status     string // display name, e.g. "IN DEVELOPMENT"
	Impact     string
	Complexity string
	Cost       string
	DependsOn  []string
	SpawnedBy  []string
	Family     string // umbrella ticket id this one groups under ("" when none)
	Reason     string // last transition's reason (the board's DROPPED/REWORK cell)
	Merged     string // merge History line, "" when unmerged
	File       string // filename, for orientation
}

// ChildGroup is one child-project's tickets within one status section.
type ChildGroup struct {
	Child   string
	Entries []Entry
	// WIP is rendered only for the two limited statuses; Limit is 0 elsewhere.
	Count int
	Limit int
}

// Section is one status section of the board.
type Section struct {
	Status   string
	Children []ChildGroup
	Total    int
}

// ChildFilter is one entry in the board's child-project filter bar: a child's
// name and its total ticket count across all statuses. The bar lets the viewer
// collapse the board to a single child (T-061).
type ChildFilter struct {
	Name  string
	Count int
}

// BoardView is the dashboard's board page.
type BoardView struct {
	Sections []Section
	Children []ChildFilter // filter-bar entries, in cfg.Projects order
	Total    int
}

// buildBoard groups tickets by status and child in the board's own order: status
// sections from def.BoardStates(), rows from board.Sort, WIP counts from
// board.WIPCounts. Nothing here re-implements those rules (decision 3).
func buildBoard(def *flow.Definition, tickets []*ticket.Ticket, cfg *config.Config) BoardView {
	wip := board.WIPCounts(def, tickets)
	// Whole-tree index for board.Sort's family-umbrella lookup (T-059); a member's
	// umbrella may live in another status section, so the per-group slice is not
	// enough. Same map the board's own Render builds.
	byID := make(map[string]*ticket.Ticket, len(tickets))
	for _, t := range tickets {
		byID[t.ID] = t
	}
	view := BoardView{Total: len(tickets)}

	for _, st := range def.BoardStates() {
		section := Section{Status: st.Name}
		for _, p := range cfg.Projects {
			var group []*ticket.Ticket
			for _, t := range tickets {
				if t.Dir == st.Dir && t.Project() == p.Name {
					group = append(group, t)
				}
			}
			board.Sort(group, st, byID)

			cg := ChildGroup{Child: p.Name, Count: len(group)}
			if st.WIPKey != "" {
				if limit, ok := p.WIPLimitFor(st.WIPKey); ok {
					cg.Count, cg.Limit = wip[p.Name][st.Dir], limit
				}
			}
			for _, t := range group {
				cg.Entries = append(cg.Entries, newEntry(def, t, st.Name))
			}
			section.Total += len(group)
			section.Children = append(section.Children, cg)
		}
		view.Sections = append(view.Sections, section)
	}

	// Filter-bar entries: one per registered child, in cfg order, with its
	// whole-tree ticket total. Computed here (not in the template) so the chip
	// counts are testable without rendering.
	for _, p := range cfg.Projects {
		count := 0
		for _, t := range tickets {
			if t.Project() == p.Name {
				count++
			}
		}
		view.Children = append(view.Children, ChildFilter{Name: p.Name, Count: count})
	}
	return view
}

func newEntry(def *flow.Definition, t *ticket.Ticket, statusName string) Entry {
	return Entry{
		ID:         t.ID,
		Num:        t.Num,
		Title:      t.Front["title"],
		Project:    t.Project(),
		Status:     statusName,
		Impact:     t.Front["impact"],
		Complexity: t.Front["complexity"],
		Cost:       t.Front["cost"],
		DependsOn:  t.DependsOn,
		SpawnedBy:  t.SpawnedBy,
		Family:     t.Front["family"],
		Reason:     ticket.LastHistoryReason(def, t.Text),
		Merged:     ticket.MergeLine(def, t.Text),
		File:       filepath.Base(t.Path),
	}
}

// TicketView is one ticket's page: the frontmatter as structured fields, the body
// as rendered markdown, and both directions of the dependency and lineage edges.
// The reverse edges are the view's own contribution — a ticket file records what it
// depends on and what spawned it, never what depends on *it*.
type TicketView struct {
	Entry
	Body    template.HTML
	Blocks  []string // tickets whose depends-on names this one
	Spawned []string // tickets whose spawned-by names this one
	Members []string // tickets whose family names this one (this ticket is their umbrella)
	History []ticket.HistoryEntry
}

// buildTicket assembles one ticket's page. all is the whole tree, needed for the
// reverse edges. It returns false when the id is unknown, so the handler can 404.
func buildTicket(def *flow.Definition, all []*ticket.Ticket, id string) (TicketView, bool) {
	var found *ticket.Ticket
	for _, t := range all {
		if t.ID == id {
			found = t
			break
		}
	}
	if found == nil {
		return TicketView{}, false
	}

	statusName := ""
	if st, ok := def.ByDir(found.Dir); ok {
		statusName = st.Name
	}
	view := TicketView{Entry: newEntry(def, found, statusName), History: ticket.HistoryEntries(def, found.Text)}

	body, err := renderMarkdown(found.Text)
	if err != nil {
		// A render failure must not lose the ticket: fall back to the raw body,
		// escaped by the template as ordinary text.
		body = template.HTML("<pre>" + template.HTMLEscapeString(stripFrontmatter(found.Text)) + "</pre>")
	}
	view.Body = body

	for _, t := range all {
		if contains(t.DependsOn, id) {
			view.Blocks = append(view.Blocks, t.ID)
		}
		if contains(t.SpawnedBy, id) {
			view.Spawned = append(view.Spawned, t.ID)
		}
		if t.Family == id {
			view.Members = append(view.Members, t.ID)
		}
	}
	sort.Strings(view.Blocks)
	sort.Strings(view.Spawned)
	sort.Strings(view.Members)
	return view, true
}

func contains(ids []string, id string) bool {
	for _, s := range ids {
		if s == id {
			return true
		}
	}
	return false
}

// urlSchemeRE marks where a candidate URL run begins; linkifyURLs measures each
// run's extent itself (see below) rather than trying to teach one regexp every
// edge case a lookahead-free engine (Go's RE2) can't express directly.
var urlSchemeRE = regexp.MustCompile(`https?://`)

// urlTrailingPunct is trimmed off the end of a URL run before it becomes an
// href: exactly the characters a human's surrounding prose puts right after a
// pasted URL (a MR ref's closing paren, a trailing comma or full stop).
const urlTrailingPunct = ")].,;:"

// linkifyURLs wraps any bare http(s) URL in s in a clickable anchor and
// HTML-escapes everything else (and the URL itself) with
// template.HTMLEscapeString. It exists because the merge History line's own
// convention (T-089) is free text rendered as a plain, auto-escaped string in
// three serve views (the board's "merged" cell, the ticket page's "merged"
// summary line, and the activity timeline's per-entry text) — unlike a
// ticket's `## Description`/body, which already gets a free clickable link
// from goldmark's GFM Linkify extension (markdown.go). This is the one place
// that gap is closed, so every caller shares it instead of each growing its
// own escape-then-wrap logic.
//
// Matching runs on the raw string, before any escaping: each match is found,
// trimmed and validated first, and only the resulting pieces are escaped —
// never the other way around. Escaping first (T-089's original approach) let
// the trim set below swallow the leading `;` of a genuine HTML entity in a
// URL's tail, corrupting it; matching raw removes that failure mode entirely
// (T-090).
func linkifyURLs(s string) template.HTML {
	starts := urlSchemeRE.FindAllStringIndex(s, -1)
	if starts == nil {
		return template.HTML(template.HTMLEscapeString(s))
	}

	var b strings.Builder
	cursor := 0
	for i, loc := range starts {
		start := loc[0]
		// A run is the longest non-whitespace stretch from start, capped at
		// the next scheme occurrence so two adjacent URLs ("a,https://b")
		// don't collapse into one anchor.
		limit := len(s)
		if i+1 < len(starts) {
			limit = starts[i+1][0]
		}
		// strings.IndexFunc decodes each rune before testing it, unlike
		// widening a single byte to a rune (T-090's review, finding F1): the
		// naive `rune(s[i])` scan stopped mid-rune on any UTF-8 continuation
		// byte that happens to satisfy unicode.IsSpace (0x85, 0xA0), truncating
		// the href and emitting invalid UTF-8 for any URL containing e.g. à,
		// Š, Р or a literal NBSP.
		end := limit
		if idx := strings.IndexFunc(s[start:limit], unicode.IsSpace); idx >= 0 {
			end = start + idx
		}

		trimmed := strings.TrimRight(s[start:end], urlTrailingPunct)
		host := strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
		if host == "" {
			// Nothing survived trimming but the scheme itself (e.g.
			// "https://)."): not a real link, leave it as plain text.
			b.WriteString(template.HTMLEscapeString(s[cursor:end]))
			cursor = end
			continue
		}

		b.WriteString(template.HTMLEscapeString(s[cursor:start]))
		escaped := template.HTMLEscapeString(trimmed)
		b.WriteString(`<a href="` + escaped + `" rel="noopener noreferrer" target="_blank">` + escaped + `</a>`)
		b.WriteString(template.HTMLEscapeString(s[start+len(trimmed) : end]))
		cursor = end
	}
	b.WriteString(template.HTMLEscapeString(s[cursor:]))

	return template.HTML(b.String()) //nolint:gosec // every byte is HTMLEscapeString output plus literal anchor markup; href and text share one escaped source, so a decoded quote can't reopen attribute parsing, and matching runs on the raw string (see doc comment) before any escaping happens.
}

// Event is one dated History line, tagged with the ticket it came from.
type Event struct {
	Date    string
	Text    string
	ID      string
	Num     int
	Title   string
	Project string
}

// ActivityView is the timeline page: every ticket's History, merged.
type ActivityView struct {
	Events    []Event
	Total     int
	Truncated bool
}

// buildActivity merges every ticket's History into one newest-first timeline —
// the one view the generated board cannot produce, since the board shows current
// state and this shows movement. Ordering is (date desc, ticket number desc) so a
// day's entries read as "latest ticket first"; within one ticket, file order is
// preserved (History is append-only, so that is chronological).
func buildActivity(def *flow.Definition, tickets []*ticket.Ticket) ActivityView {
	var events []Event
	for _, t := range tickets {
		for _, h := range ticket.HistoryEntries(def, t.Text) {
			events = append(events, Event{
				Date:    h.Date,
				Text:    h.Text,
				ID:      t.ID,
				Num:     t.Num,
				Title:   t.Front["title"],
				Project: t.Project(),
			})
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date > events[j].Date // YYYY-MM-DD sorts lexicographically
		}
		return events[i].Num > events[j].Num
	})

	view := ActivityView{Total: len(events)}
	if len(events) > activityCap {
		view.Events, view.Truncated = events[:activityCap], true
	} else {
		view.Events = events
	}
	return view
}

// ChildWIP is one child's WIP state for the health banner.
type ChildWIP struct {
	Child                 string
	InDevelopment, DevCap int
	InReview, RevCap      int
}

// AtDevLimit reports whether the child is at or over its in-development limit.
func (c ChildWIP) AtDevLimit() bool { return c.InDevelopment >= c.DevCap }

// AtReviewLimit reports whether the child is at or over its in-review limit.
func (c ChildWIP) AtReviewLimit() bool { return c.InReview >= c.RevCap }

// HealthView is the banner: what `pickle board audit` would say right now, plus
// each child's WIP state. It reports; it never repairs. A stale board stays stale
// until a human runs `pickle board sync` — this server does not write.
type HealthView struct {
	Tickets  int
	Errors   []string
	Warnings []string
	Children []ChildWIP
}

// OK reports whether the audit found nothing at all.
func (h HealthView) OK() bool { return len(h.Errors) == 0 && len(h.Warnings) == 0 }

// buildHealth's two badges (InDevelopment/DevCap, InReview/RevCap) are
// template-bound field names, tied to the two config WIP keys — not to
// directory strings. A flow with a third WIP-limited state renders no third
// badge until ChildWIP and the health template both grow one; that is out of
// scope here (T-080).
func buildHealth(def *flow.Definition, root string, tickets []*ticket.Ticket, cfg *config.Config) HealthView {
	res := audit.Audit(root, cfg)
	view := HealthView{Tickets: res.NumTickets, Errors: res.Errors, Warnings: res.Warnings}
	wip := board.WIPCounts(def, tickets)
	devSt, _ := def.StateByWIPKey(config.WIPKeyInDevelopment)
	revSt, _ := def.StateByWIPKey(config.WIPKeyInReview)
	for _, p := range cfg.Projects {
		devLimit, _ := p.WIPLimitFor(config.WIPKeyInDevelopment)
		revLimit, _ := p.WIPLimitFor(config.WIPKeyInReview)
		view.Children = append(view.Children, ChildWIP{
			Child:         p.Name,
			InDevelopment: wip[p.Name][devSt.Dir], DevCap: devLimit,
			InReview: wip[p.Name][revSt.Dir], RevCap: revLimit,
		})
	}
	return view
}

// page is what every template is executed with.
type page struct {
	Title    string
	Project  string // the overarching project's directory name, for the header
	Health   HealthView
	Board    BoardView
	Ticket   TicketView
	Activity ActivityView
}

// projectName is the label in the header: the overarching project root's directory
// name. It is cosmetic, so an unnamed root degrades to "pickle" rather than erroring.
func projectName(root string) string {
	if base := strings.TrimSpace(filepath.Base(root)); base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return "pickle"
}
