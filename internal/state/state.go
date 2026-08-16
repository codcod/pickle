// Package state builds the JSON wire projection `pickle board state --json`
// prints (T-065): a versioned, machine-readable snapshot of the whole ticket
// tree — every child-project, every ticket's frontmatter, parsed History and
// dispositioned review findings, WIP counts, and audit health.
//
// This is a deliberately separate set of types from internal/serve's view
// layer (Entry, BoardView, TicketView, …), not a marshalling of it. Two
// reasons this package does not reuse or refactor serve's types (T-065
// confirmed decision 2):
//
//  1. serve's types carry template.HTML fields and template-bound predicate
//     methods (Entry.Search, ChildWIP.AtDevLimit, …) that exist only to serve
//     an HTML page. A JSON consumer needs none of that, and marshalling those
//     types as-is would leak rendering concerns (escaped HTML, precomputed
//     search strings) into the wire format.
//  2. serve's shape has been growing to fit its dashboard (T-080 threaded a
//     *flow.Definition through every builder; T-104 added Lane/ChildRow/
//     ChildFilter for the side-by-side board layout) — coupling this
//     projection to that shape would make every future dashboard change a
//     wire-format compatibility question.
//
// internal/serve is therefore untouched by this package: nothing here is
// imported by internal/serve, and nothing in internal/serve is imported here
// beyond the same lower-level packages (internal/ticket, internal/board,
// internal/audit, internal/flow, internal/config, internal/lock) serve
// itself already depends on. The duplication between this package's Build
// and serve's buildBoard/buildTicket/buildHealth is intentional, not an
// oversight — see build.go.
package state

// Document is the whole projection: the envelope plus every section.
//
// Schema versions the wire format itself (T-065 confirmed decision 4): a
// consumer that does not recognise Schema should refuse to parse rather than
// guess. Adding a field to any type below is a compatible change; removing or
// retyping one is not, and must bump Schema.
type Document struct {
	Schema        int      `json:"schema"`
	PickleVersion string   `json:"pickle_version"`
	Flow          string   `json:"flow"`
	Root          string   `json:"root"`
	States        []State  `json:"states"`
	Children      []Child  `json:"children"`
	Tickets       []Ticket `json:"tickets"`
	Health        Health   `json:"health"`
}

// CurrentSchema is the Document.Schema value this build emits.
const CurrentSchema = 1

// State is one status the flow defines (a section of the board), in
// def.BoardStates() order — never hardcoded by a consumer.
type State struct {
	Name     string `json:"name"` // display name, e.g. "IN DEVELOPMENT"
	Dir      string `json:"dir"`  // status directory (the ticket tree's own dir name for this state)
	Terminal bool   `json:"terminal"`
	WIPKey   string `json:"wip_key"` // pickle.toml key, "" when this state has no WIP limit
}

// WIP is one child-project's count against one WIP-limited state.
type WIP struct {
	Key    string `json:"key"`    // the pickle.toml key, e.g. "wip_in_development"
	Dir    string `json:"dir"`    // the state's status directory
	Status string `json:"status"` // the state's display name
	Count  int    `json:"count"`
	Limit  int    `json:"limit"`
}

// Child is one registered child-project, in pickle.toml order.
type Child struct {
	Name string `json:"name"`
	WIP  []WIP  `json:"wip"`
}

// Ticket is one ticket file, projected. Every path here is repo-relative
// (Slug/File), never absolute, so a ticket entry is identical from any
// checkout location. Document.Root is the one absolute path in the whole
// document, and therefore the one field that differs between two checkouts of
// the same tree — decision 3's byte-identical guarantee is per-tree (two runs
// against an unchanged tree), and holds across checkouts once Root is
// excluded.
type Ticket struct {
	ID            string            `json:"id"`
	Num           int               `json:"num"`
	Prefix        string            `json:"prefix"`
	Title         string            `json:"title"`
	Project       string            `json:"project"`
	Status        string            `json:"status"` // display name, e.g. "IN REVIEW"
	Dir           string            `json:"dir"`
	File          string            `json:"file"` // basename, e.g. "T-065-….md"
	Slug          string            `json:"slug"`
	Impact        string            `json:"impact"`
	Complexity    string            `json:"complexity"`
	Cost          string            `json:"cost"`
	DependsOn     []string          `json:"depends_on"`
	SpawnedBy     []string          `json:"spawned_by"`
	Family        string            `json:"family"` // "" when none
	DuplicateKeys []string          `json:"duplicate_keys"`
	FrontMatter   map[string]string `json:"front_matter"` // raw frontmatter, unknown keys included
	Merged        string            `json:"merged"`       // merge History line, "" when unmerged
	History       []HistoryEntry    `json:"history"`
	Review        Review            `json:"review"`
}

// HistoryEntry is one dated `## History` line, parsed.
type HistoryEntry struct {
	Date   string `json:"date"`
	Kind   string `json:"kind"`   // "created" | "merged" | "transition" | "note"
	Target string `json:"target"` // transition destination status, "" unless Kind == "transition"
	Text   string `json:"text"`
}

// Finding is one row of a `## Review` findings table, projected through the
// middle option T-065 settled on at refinement: the four closed-vocabulary
// columns only. The three prose columns (description, evidence, suggestion —
// under any of their header aliases) are deliberately not projected; see
// review.go's package doc for why.
type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Class       string `json:"class"` // "" for a pre-T-085 table, which had no class column
	Disposition string `json:"disposition"`
}

// Review is one ticket's `## Review` section, projected.
type Review struct {
	Tables             int        `json:"tables"`  // number of findings tables detected
	Headers            [][]string `json:"headers"` // each table's normalised column names, in source order
	Findings           []Finding  `json:"findings"`
	DispositionSummary string     `json:"disposition_summary"` // the "Disposition summary: …" line, verbatim; "" if absent
	CostLine           string     `json:"cost_line"`           // the "cost: estimated …, actual …" line, verbatim; "" if absent
}

// Health is `pickle board audit`'s verdict, as data.
type Health struct {
	Tickets    int      `json:"tickets"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
	BoardDrift string   `json:"board_drift"` // "none" | "layout" | "rows" — board.Drift's vocabulary (T-052)
}
