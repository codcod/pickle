// Package decisions implements the read model behind `pickle board decisions`
// (T-105): every confirmed design decision recorded in a ticket tree's
// `### Confirmed design decisions` subsections, in citable `<ID> decision <N>`
// form.
//
// Like internal/audit and internal/changelog, this is a leaf, pure
// text-in/values-out package — no printing, no exit codes, no subprocess, no
// locking of its own (the caller, internal/cli/board.go, wraps the call in
// lock.WithShared, the same pattern internal/state/build.go uses). That
// keeps this package fixture/table-testable with literal ticket-file
// strings, exactly as internal/state/review.go tests its own parsing.
//
// The parsing rules mirror internal/state/review.go's for `## Review`
// findings tables, applied one level down to `### Confirmed design
// decisions` items instead of table rows:
//
//   - Only a projectable, closed-vocabulary shape is projected — the ticket
//     id, the decision's own ordinal, and (when present) the leading bold
//     run as the decision's statement. The multi-sentence rationale that
//     follows is never projected, mirroring T-065 decision 9's choice not to
//     project a findings row's prose columns.
//   - A decision with no leading bold run is reported as unstructured, with
//     its raw first line carried instead — never inferred into a statement
//     it doesn't have (T-105 decision 7/8).
//   - The ordinal is read from the file, never recounted: a gap (1, 2, 4)
//     is reported as 4, because some other ticket may already cite it as
//     "<ID> decision 4" (T-105 decision 4).
package decisions

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

// CurrentSchema is the Document.Schema value this build emits. It versions
// this package's own wire format, independently of internal/state's
// Document.Schema (T-105 decision 13) — the two evolve on separate
// schedules, and folding this projection into `board state --json` stays a
// future, compatible decision rather than one this command forces.
const CurrentSchema = 1

// Decision is one confirmed design decision, projected from its ticket.
//
// Only TicketID/Prefix/Num/Project/Status/Dir/File/Number/Citation identify
// where the decision came from; Structured/Statement/Raw are the decision's
// own content, projected per the package doc's closed-vocabulary rule. The
// rationale prose that follows the statement (or, for an unstructured item,
// every line after the first) is deliberately absent from this type — it is
// still searched by Filter.Grep (see full, below), just never surfaced.
type Decision struct {
	TicketID   string `json:"ticket_id"`
	Prefix     string `json:"prefix"`
	Num        int    `json:"num"`
	Project    string `json:"project"`
	Status     string `json:"status"` // display name, e.g. "IN REVIEW"
	Dir        string `json:"dir"`    // status directory, e.g. "4-in-review"
	File       string `json:"file"`   // basename, e.g. "T-105-…md"
	Number     int    `json:"number"` // the ordinal as written in the file — never recounted
	Citation   string `json:"citation"`
	Structured bool   `json:"structured"` // true when Statement was found; false when Raw was
	Statement  string `json:"statement"`  // the leading bold run, emphasis-stripped; "" unless Structured
	Raw        string `json:"raw"`        // the raw first line; "" when Structured

	// full is the item's whole text — statement and rationale both — used
	// only as Filter.Grep's search target (T-105 decision 9: a topic search
	// must reach the rationale, even though the output never projects it).
	// Unexported: it never reaches JSON or any caller outside this package.
	full string
}

// itemRE matches a column-0 numbered list item opening a decision: "N. " or
// "N) " at the very start of the line. A match requires no leading
// whitespace, which is what keeps an indented, nested ordinal — a sub-point
// inside the current decision's own rationale — from being misread as a
// decision of its own (T-105 decision 5).
var itemRE = regexp.MustCompile(`^(\d+)[.)]\s+(.*)$`)

// leadingBoldRE matches a leading "**…**" run — the decision statement, when
// present (T-105 decision 7). It is matched against the item's whole text
// with soft line-wraps flattened to spaces (flattenItem), not only its first
// physical line: measurement against this corpus found the bold statement
// itself routinely wraps onto a second physical line before its closing
// "**" (e.g. "**Mechanism: a `@media (...)` block that overrides the custom\n
// properties.**", T-054 decision 1) — an ordinary markdown soft wrap, not a
// second paragraph, and a first-line-only match misread over a dozen
// genuinely-structured decisions in this repo alone as unstructured. The
// non-greedy `.+?` still stops at the *nearest* closing "**", which is the
// leading run's own close regardless of how many source lines it spans.
var leadingBoldRE = regexp.MustCompile(`^\*\*(.+?)\*\*`)

// emphasisRE and whitespaceRE mirror internal/state/review.go's own
// normalisation regexes (T-065 decision 11) — same shape, kept as this
// package's own unexported copy rather than an import, since review.go's are
// themselves unexported and this package otherwise has no reason to depend
// on internal/state.
var (
	emphasisRE   = regexp.MustCompile("[*`]")
	whitespaceRE = regexp.MustCompile(`\s+`)
)

// normalizeStatement strips emphasis/code markers and collapses whitespace —
// formatting normalisation only, never a value judgement (T-065 decision 11,
// reapplied here to a decision's bold run instead of a review table cell).
func normalizeStatement(s string) string {
	s = emphasisRE.ReplaceAllString(s, "")
	s = whitespaceRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// Extract parses one ticket's file text and returns every confirmed design
// decision found in its `## Implementation Plan` → `### Confirmed design
// decisions` subsection, in file order. It fills only the item's own fields
// (Number, Structured, Statement, Raw, full) — TicketID and the other
// per-ticket fields are the caller's job (Query), since Extract has no
// ticket to attribute them to.
//
// Absent plan, absent subsection, or an empty one, all yield nil — never an
// error: a ticket with nothing to report is not a defect (T-105 decision 1
// half: the command must work on the corpus as it stands).
//
// Lines inside a fenced code block (``` … ```, tracked regardless of
// indentation) are never scanned for a column-0 ordinal, so a numbered line
// in an illustrative snippet is never counted as a decision of its own
// (T-105 decision 6) — a deliberate, local improvement over
// ticket.SectionBody's documented fence-blindness, not a change to that
// function.
func Extract(text string) []Decision {
	body, found := ticket.SubsectionBody(text, "Implementation Plan", "confirmed")
	if !found || strings.TrimSpace(body) == "" {
		return nil
	}

	var out []Decision
	var cur *Decision
	var firstLine string
	var rest []string
	inFence := false

	flush := func() {
		if cur == nil {
			return
		}
		// flatten joins the item's soft-wrapped lines with spaces — mirroring how
		// a markdown renderer treats a single line break inside one paragraph —
		// so a leading bold run that wraps onto a second physical line is still
		// found as one contiguous "**…**" span.
		flatten := firstLine
		for _, l := range rest {
			flatten += " " + strings.TrimSpace(l)
		}
		if m := leadingBoldRE.FindStringSubmatch(strings.TrimSpace(flatten)); m != nil {
			cur.Structured = true
			cur.Statement = normalizeStatement(m[1])
		} else {
			cur.Structured = false
			cur.Raw = strings.TrimSpace(firstLine)
		}
		full := firstLine
		if len(rest) > 0 {
			full += "\n" + strings.Join(rest, "\n")
		}
		cur.full = strings.TrimSpace(full)
		out = append(out, *cur)
		cur, firstLine, rest = nil, "", nil
	}

	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "```") {
			inFence = !inFence
			if cur != nil {
				rest = append(rest, line)
			}
			continue
		}
		if inFence {
			if cur != nil {
				rest = append(rest, line)
			}
			continue
		}
		if m := itemRE.FindStringSubmatch(line); m != nil {
			flush()
			num, err := strconv.Atoi(m[1])
			if err != nil {
				continue // itemRE's \d+ guarantees this never happens
			}
			cur = &Decision{Number: num}
			firstLine = m[2]
			continue
		}
		if cur != nil {
			rest = append(rest, line)
		}
	}
	flush()
	return out
}

// Filter selects which decisions Query returns. Every field is optional and
// they compose (T-105 decision 2); the zero Filter matches everything.
type Filter struct {
	Project string // must name a registered child when set (else error)
	Status  string // a status directory name, e.g. "6-done" (else error)
	Grep    string // a regexp matched against the whole item — statement and rationale (T-105 decision 9)
}

// Result is Query's return value.
type Result struct {
	Decisions []Decision
}

// legalStatusDirs renders def's status directories for an unknown---status
// error message, in def.States() order (never a hardcoded list — T-105
// decision 10).
func legalStatusDirs(def *flow.Definition) string {
	states := def.States()
	dirs := make([]string, 0, len(states))
	for _, s := range states {
		dirs = append(dirs, s.Dir)
	}
	return strings.Join(dirs, ", ")
}

// Query loads every ticket under root, applies f, and returns every matching
// decision, sorted deterministically and prefix-agnostically: ticket prefix
// ascending, then ticket number ascending, then decision number ascending
// (T-105 decision 12) — a decision chain reads by id, not by status.
//
// An unregistered f.Project, an unknown f.Status directory, or an
// uncompilable f.Grep are all errors (exit 1 at the CLI). A registered
// child, a legal status, or any filter combination that simply matches
// nothing is not an error: Result.Decisions is empty and non-nil (T-105
// decision 11; the exit-0-empty-result contract the acceptance test checks).
func Query(def *flow.Definition, root string, cfg *config.Config, f Filter) (Result, error) {
	if f.Project != "" {
		if _, ok := cfg.Project(f.Project); !ok {
			return Result{}, fmt.Errorf("project %q is not registered", f.Project)
		}
	}
	if f.Status != "" {
		if _, ok := def.ByDir(f.Status); !ok {
			return Result{}, fmt.Errorf("unknown status %q — legal values: %s", f.Status, legalStatusDirs(def))
		}
	}
	var grepRE *regexp.Regexp
	if f.Grep != "" {
		re, err := regexp.Compile("(?i)" + f.Grep)
		if err != nil {
			return Result{}, fmt.Errorf("invalid --grep pattern %q: %w", f.Grep, err)
		}
		grepRE = re
	}

	tickets, _ := ticket.LoadAll(def, root)
	out := make([]Decision, 0, 16)
	for _, t := range tickets {
		if f.Project != "" && t.Project() != f.Project {
			continue
		}
		if f.Status != "" && t.Dir != f.Status {
			continue
		}
		prefix, num, ok := ticket.SplitID(t.ID)
		if !ok {
			continue // malformed id — already reported by ticket.LoadAll's own issues; not this command's job
		}
		status, _ := def.ByDir(t.Dir) // t.Dir came from LoadAll's own walk of def.States(), so this always resolves
		for _, d := range Extract(t.Text) {
			if grepRE != nil && !grepRE.MatchString(d.full) {
				continue
			}
			d.TicketID = t.ID
			d.Prefix = prefix
			d.Num = num
			d.Project = t.Project()
			d.Status = status.Name
			d.Dir = t.Dir
			d.File = filepath.Base(t.Path)
			d.Citation = fmt.Sprintf("%s decision %d", t.ID, d.Number)
			out = append(out, d)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Prefix != out[j].Prefix {
			return out[i].Prefix < out[j].Prefix
		}
		if out[i].Num != out[j].Num {
			return out[i].Num < out[j].Num
		}
		return out[i].Number < out[j].Number
	})

	return Result{Decisions: out}, nil
}

// Document is the JSON envelope `pickle board decisions --json` prints: this
// package's own small, independently versioned wire format (T-105 decision
// 13) — internal/state's Document and CurrentSchema are untouched by this
// command.
type Document struct {
	Schema        int        `json:"schema"`
	PickleVersion string     `json:"pickle_version"`
	Filters       Filters    `json:"filters"`
	Decisions     []Decision `json:"decisions"`
}

// Filters records the query that produced a Document, so a consumer of
// --json output does not have to remember its own invocation.
type Filters struct {
	Project string `json:"project"`
	Status  string `json:"status"`
	Grep    string `json:"grep"`
}
