// Package flow is brine's lifecycle as data: states, legal transitions, terminal
// and WIP flags, board presentation (order/heading/columns), and the state a
// dependency must reach before a dependent can be picked up — all in one
// declarative Definition instead of Go literals re-typed across ~nine call
// sites in internal/ticket, internal/move, internal/board, internal/audit,
// internal/install, internal/sync, internal/cli and internal/serve (T-080).
//
// This package is a leaf: it imports nothing from internal/ (not config, not
// ticket). A caller resolves a *Definition from config (see ForName) and
// threads it explicitly through every function that needs status vocabulary —
// there is deliberately no package-level default referenced directly by other
// packages, and no definition cached on a ticket. That threading is what
// proves the extraction actually generalises rather than just moving the same
// hard-coded assumptions one file over.
//
// brine (brine.go) is the only definition shipped today. A Spec is plain data
// an author writes; New validates it into an immutable Definition. Reading a
// user-authored Spec from disk is out of scope here — see T-081, the natural
// follow-on this ticket exists to enable.
package flow

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// ColumnProfile names which board column set a state's section renders with.
// internal/board maps each profile to its column list — kept as a board
// concept, chosen per state, so a state naming an unknown profile is a
// definition-validation error rather than a board render producing a
// headerless table.
type ColumnProfile string

const (
	ColumnsBacklog ColumnProfile = "backlog" // TO DO / READY: impact, complexity, cost, depends-on, family
	ColumnsActive  ColumnProfile = "active"  // IN DEVELOPMENT / IN REVIEW: depends-on
	ColumnsRework  ColumnProfile = "rework"  // REWORK: open findings
	ColumnsDone    ColumnProfile = "done"    // DONE: merged
	ColumnsDropped ColumnProfile = "dropped" // DROPPED: reason
)

// columnProfiles is the exhaustive, ordered set New validates a State.Columns
// against.
var columnProfiles = []ColumnProfile{ColumnsBacklog, ColumnsActive, ColumnsRework, ColumnsDone, ColumnsDropped}

// ColumnProfiles lists every column profile a state may name, in the same
// order New validates against. Exported so a caller like internal/board can
// assert it has a column set for every profile this package defines, instead
// of the two lists drifting silently apart.
func ColumnProfiles() []ColumnProfile { return slices.Clone(columnProfiles) }

// TransitionKind classifies a transition for the purposes of whether it
// requires a --reason (RequiresReason: everything but Forward) and whether it
// counts as "backwards" for board/CLI messaging.
type TransitionKind int

const (
	// Forward is ordinary progress along the lifecycle. Never requires a reason.
	Forward TransitionKind = iota
	// Backward moves a ticket to an earlier, non-terminal state because a later
	// stage stalled or its gate no longer holds (the plan itself still valid).
	Backward
	// SendBack moves a ticket from review back to rework because of a blocking
	// finding.
	SendBack
	// Abort moves a ticket to a terminal "abandoned" state.
	Abort
)

func (k TransitionKind) String() string {
	switch k {
	case Forward:
		return "forward"
	case Backward:
		return "backward"
	case SendBack:
		return "send-back"
	case Abort:
		return "abort"
	default:
		return fmt.Sprintf("TransitionKind(%d)", int(k))
	}
}

// State is one status in the lifecycle: its directory, display name, board
// heading, terminal flag, board sort behaviour, column profile, the gate
// table it enforces, and — for a WIP-limited state — the pickle.toml key its
// limit is configured under.
type State struct {
	Dir  string // "3-in-development" — the status directory under tickets/
	Name string // "IN DEVELOPMENT" — the display name

	// Heading is the board's `## ` heading text for this state's section —
	// usually Name, but READY/TO DO append " (impact order, per child)".
	Heading string

	// Terminal states have no outgoing transitions (DONE, DROPPED for brine).
	Terminal bool

	// ImpactOrder states sort their board section by descending impact (ties
	// by id); every other state sorts by id ascending.
	ImpactOrder bool

	// Columns names the board column profile this state's section renders.
	Columns ColumnProfile

	// WIPKey is the pickle.toml key (e.g. "wip_in_development") whose value
	// caps how many tickets of one child-project may sit in this state at
	// once, or "" when this state is not WIP-limited.
	WIPKey string

	// Requires is this state's gate table (T-081): the ticket sections a
	// ticket must carry to enter this state, checked again while it sits
	// here (Requirement's doc comment explains why entry and residency share
	// one table). A terminal state declaring no Requires is how an archived
	// ticket is exempt from a check that would otherwise warn on it forever
	// (see brine.go's terminal states for the worked example) — New still
	// permits a terminal state to declare rows, for a flow that wants one.
	Requires []Requirement
}

// Severity classifies how a Requirement is enforced once its state is
// unmet: Blocking gates the move itself, Advisory only ever warns. The zero
// value is deliberately invalid — New rejects it — so a Requirement cannot
// acquire or lose its teeth by an author simply forgetting the field.
type Severity int

const (
	_ Severity = iota // zero value is invalid: New rejects an unset Severity
	// Blocking means an unmet Requirement refuses the ticket move that would
	// enter this state, and is reported by board audit as an error.
	Blocking
	// Advisory means an unmet Requirement never refuses a move — only board
	// audit (including the one ticket move runs on itself after applying) —
	// reports it, as a warning.
	Advisory
)

func (s Severity) String() string {
	switch s {
	case Blocking:
		return "blocking"
	case Advisory:
		return "advisory"
	default:
		return fmt.Sprintf("Severity(%d)", int(s))
	}
}

// Requirement is one row of a state's gate table (State.Requires): the "##"
// Section a ticket must carry to enter, and remain in, that state —
// optionally narrowed to one "###" sub-heading inside that section (Sub),
// when the state cares about a specific step rather than the section as a
// whole. Label names the requirement for messages; Hint is the one-clause
// remedy every violation message ends with (internal/ticket.GateViolation
// renders both).
//
// Deliberately narrow, twice over:
//
//   - Section is a single string, not a set of acceptable alternatives —
//     rick's phase gates (the model this table copies) let a requirement
//     accept any one of several artifact *kinds* ("an approved design OR an
//     approved task"); brine never needs that today. An any-of set is the
//     extension point a per-phase-artifact flow would add here, not
//     something to speculatively build now.
//   - There is no approval flag. Rick's requirements distinguish an artifact
//     merely existing from one that has been approved; brine has no
//     approval concept for a ticket section, and this type does not invent
//     one.
//
// Entry and residency share this one table by design: a requirement on state
// X is evaluated both when a ticket moves into X (internal/move, which
// refuses on an unmet Blocking row) and while a ticket already sits in X
// (internal/board's audit, run standalone and as ticket move's own
// post-move self-check). A flow wanting an entry-only check — enforced at
// the door but never re-checked at rest — is out of scope; nothing here
// models that distinction.
//
// Sub, when set, must already be in internal/ticket's normalised form
// (lower-case, no leading/trailing whitespace) — New checks that much itself
// even though it cannot check the fuller normalisation rule without
// importing internal/ticket (this package stays a leaf). It is matched as a
// *prefix* of the normalised heading, not an exact match, precisely so
// `"0. Feature branch (mandatory)"` and `"Feature branch"` can both satisfy
// `Sub: "feature branch"`.
type Requirement struct {
	Section  string
	Sub      string
	Label    string
	Hint     string
	Severity Severity
}

// Transition is one legal (From, To) move between two state directories.
type Transition struct {
	From, To string
	Kind     TransitionKind
}

// RequiresReason reports whether a transition of this kind must carry a
// --reason: every kind but Forward.
func (k TransitionKind) RequiresReason() bool { return k != Forward }

// Spec is the plain, exported data a flow definition author writes. New
// validates a Spec into an immutable Definition; nothing else in this package
// or its callers reads a Spec's fields directly once a Definition exists.
type Spec struct {
	// Name is the flow's name, as configured by pickle.toml's flow key
	// ("brine" is the only legal value today — see internal/config.Validate).
	Name string

	// States lists every state in lifecycle (directory) order — the order a
	// ticket walks tickets/ in, and the order internal/install scaffolds
	// directories in.
	States []State

	// BoardOrder lists every state's display Name in the order the board
	// renders its sections (active work first) — a permutation of States,
	// not necessarily the same order.
	BoardOrder []string

	// Transitions lists every legal move. A state with no outgoing entries
	// (and no incoming requirement otherwise) is terminal only if Terminal is
	// also set on its State — the two must agree (see New's validation).
	Transitions []Transition

	// Initial is the dir a freshly created ticket lands in.
	Initial string

	// DependencySatisfied is the dir a depends-on target must reach — and stay
	// merged from — before a ticket naming it may be picked up.
	DependencySatisfied string

	// Pickup is the dir a ticket is built in — the state the dependency+merge
	// gate itself guards entry into (T-080 finding N6). It must name a known,
	// non-terminal state. Before this field existed, both internal/move and
	// internal/audit inferred "the pickup state" as whichever state happened
	// to be keyed by config.WIPKeyInDevelopment — the flow's most
	// load-bearing gate keyed off an unrelated concern (WIP limits), which
	// silently skipped the entire gate on a lookup miss. Pickup makes the
	// concept explicit and mandatory instead.
	Pickup string
}

// Definition is a validated, immutable view of a Spec. Every slice-returning
// accessor returns a fresh copy (slices.Clone) so a caller cannot reorder or
// mutate the flow by mutating what it was handed — the same guarantee
// board.StatusOrder made of its own copy before this package existed.
type Definition struct {
	name         string
	states       []State // lifecycle order
	boardStates  []State // board order
	byDir        map[string]State
	byName       map[string]State
	transitions  []Transition
	allowed      map[string][]Transition // by From dir, in Transitions order
	initial      State
	depSat       State
	pickup       State
	wipStates    []State // board order, WIPKey != ""
	byWIPKey     map[string]State
	requires     map[string][]Requirement // by dir
	activeStates []State                  // lifecycle order, non-terminal, excluding Initial
}

// Name is the flow's configured name ("brine" today).
func (d *Definition) Name() string { return d.name }

// States returns every state in lifecycle (directory) order.
func (d *Definition) States() []State { return slices.Clone(d.states) }

// BoardStates returns every state in board render order.
func (d *Definition) BoardStates() []State { return slices.Clone(d.boardStates) }

// ActiveStates returns every state that is neither the backlog entry point
// (Initial) nor terminal, in lifecycle (States) order — brine's "work in
// flight": READY, IN DEVELOPMENT, IN REVIEW, REWORK. It is defined by
// exclusion rather than as an enumerated list so a caller (the dashboard's
// per-child column layout, T-104) needs no state-name literals of its own and
// keeps working unchanged if a future flow adds or renames a state: whatever
// sits between the backlog and the archive is "active" by construction.
func (d *Definition) ActiveStates() []State { return slices.Clone(d.activeStates) }

// Transitions returns every legal transition, in Spec order.
func (d *Definition) Transitions() []Transition { return slices.Clone(d.transitions) }

// ByDir returns the state for a directory name.
func (d *Definition) ByDir(dir string) (State, bool) {
	s, ok := d.byDir[dir]
	return s, ok
}

// ByName returns the state for a display name.
func (d *Definition) ByName(name string) (State, bool) {
	s, ok := d.byName[name]
	return s, ok
}

var statusNumRE = regexp.MustCompile(`^\d+-`)

// ByToken resolves a user-supplied status token, case-insensitively, in any of
// three forms: the dir name ("3-in-development"), the dir minus its leading
// number ("in-development"), or the display name lower-cased with spaces
// turned to hyphens ("in-development" from "IN DEVELOPMENT").
func (d *Definition) ByToken(tok string) (State, bool) {
	t := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(tok)), " ", "-")
	for _, s := range d.states {
		bare := statusNumRE.ReplaceAllString(s.Dir, "")
		name := strings.ReplaceAll(strings.ToLower(s.Name), " ", "-")
		if t == s.Dir || t == bare || t == name {
			return s, true
		}
	}
	return State{}, false
}

// Initial is the state a freshly created ticket lands in.
func (d *Definition) Initial() State { return d.initial }

// DependencySatisfied is the state a depends-on target must reach (and stay
// merged from) before a ticket naming it may be picked up.
func (d *Definition) DependencySatisfied() State { return d.depSat }

// Pickup is the state a ticket is built in — the state the dependency+merge
// gate guards entry into. Unlike StateByWIPKey, a validated Definition
// always has one: New rejects a Spec whose Pickup does not name a known,
// non-terminal state, so this accessor cannot fail.
func (d *Definition) Pickup() State { return d.pickup }

// Requirements returns dir's gate table (State.Requires), in table order, as
// a clone — the same mutation-safety contract every other slice-returning
// accessor on this type makes. An unknown dir returns nil, mirroring Allowed.
func (d *Definition) Requirements(dir string) []Requirement {
	return slices.Clone(d.requires[dir])
}

// Allowed returns the legal target states from a source directory, in Spec
// transition order. A terminal state (or an unknown one) returns nil.
func (d *Definition) Allowed(fromDir string) []State {
	ts := d.allowed[fromDir]
	if len(ts) == 0 {
		return nil
	}
	out := make([]State, len(ts))
	for i, t := range ts {
		out[i] = d.byDir[t.To]
	}
	return out
}

// Kind returns the transition kind for a (from, to) pair of directories, and
// whether that transition is legal at all.
func (d *Definition) Kind(from, to string) (TransitionKind, bool) {
	for _, t := range d.allowed[from] {
		if t.To == to {
			return t.Kind, true
		}
	}
	return Forward, false
}

// RequiresReason reports whether moving from dir "from" to dir "to" requires a
// --reason: every transition kind but Forward. An illegal (from, to) pair
// reports false — the caller is expected to have already rejected it as
// illegal, per the same order internal/move checks in today.
func (d *Definition) RequiresReason(from, to string) bool {
	k, ok := d.Kind(from, to)
	return ok && k.RequiresReason()
}

// WIPStates returns every WIP-limited state, in board order.
func (d *Definition) WIPStates() []State { return slices.Clone(d.wipStates) }

// StateByWIPKey returns the state whose WIPKey matches key.
func (d *Definition) StateByWIPKey(key string) (State, bool) {
	s, ok := d.byWIPKey[key]
	return s, ok
}

// New validates spec and returns an immutable Definition, or an error
// describing the first invariant violated. See the package doc and T-080's
// ticket for the full list of what this checks.
func New(spec Spec) (*Definition, error) {
	if spec.Name == "" {
		return nil, fmt.Errorf("flow: Name is empty")
	}
	if len(spec.States) == 0 {
		return nil, fmt.Errorf("flow %q: no States", spec.Name)
	}

	byDir := make(map[string]State, len(spec.States))
	byName := make(map[string]State, len(spec.States))
	for _, s := range spec.States {
		if s.Dir == "" {
			return nil, fmt.Errorf("flow %q: a state has an empty Dir", spec.Name)
		}
		if s.Dir == "." || s.Dir == ".." || strings.ContainsAny(s.Dir, "/\\") {
			return nil, fmt.Errorf("flow %q: state Dir %q is not a plain single path element", spec.Name, s.Dir)
		}
		if s.Name == "" {
			return nil, fmt.Errorf("flow %q: state %q has an empty Name", spec.Name, s.Dir)
		}
		if _, dup := byDir[s.Dir]; dup {
			return nil, fmt.Errorf("flow %q: duplicate state Dir %q", spec.Name, s.Dir)
		}
		if _, dup := byName[s.Name]; dup {
			return nil, fmt.Errorf("flow %q: duplicate state Name %q", spec.Name, s.Name)
		}
		if !slices.Contains(columnProfiles, s.Columns) {
			return nil, fmt.Errorf("flow %q: state %q has unknown Columns profile %q", spec.Name, s.Dir, s.Columns)
		}
		if err := validateRequirements(spec.Name, s.Dir, s.Requires); err != nil {
			return nil, err
		}
		byDir[s.Dir] = s
		byName[s.Name] = s
	}

	if len(spec.BoardOrder) != len(spec.States) {
		return nil, fmt.Errorf("flow %q: BoardOrder has %d entries, want %d (one per state)",
			spec.Name, len(spec.BoardOrder), len(spec.States))
	}
	boardStates := make([]State, 0, len(spec.BoardOrder))
	seenBoard := make(map[string]bool, len(spec.BoardOrder))
	for _, name := range spec.BoardOrder {
		s, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("flow %q: BoardOrder names unknown state %q", spec.Name, name)
		}
		if seenBoard[name] {
			return nil, fmt.Errorf("flow %q: BoardOrder repeats state %q", spec.Name, name)
		}
		seenBoard[name] = true
		boardStates = append(boardStates, s)
	}

	initial, ok := byDir[spec.Initial]
	if !ok {
		return nil, fmt.Errorf("flow %q: Initial %q is not a known state", spec.Name, spec.Initial)
	}
	if initial.Terminal {
		return nil, fmt.Errorf("flow %q: Initial %q is terminal", spec.Name, spec.Initial)
	}
	depSat, ok := byDir[spec.DependencySatisfied]
	if !ok {
		return nil, fmt.Errorf("flow %q: DependencySatisfied %q is not a known state", spec.Name, spec.DependencySatisfied)
	}
	pickup, ok := byDir[spec.Pickup]
	if !ok {
		return nil, fmt.Errorf("flow %q: Pickup %q is not a known state", spec.Name, spec.Pickup)
	}
	if pickup.Terminal {
		return nil, fmt.Errorf("flow %q: Pickup %q is terminal", spec.Name, spec.Pickup)
	}

	allowed := make(map[string][]Transition, len(spec.States))
	seenPair := make(map[[2]string]bool, len(spec.Transitions))
	for _, t := range spec.Transitions {
		from, fromOK := byDir[t.From]
		if !fromOK {
			return nil, fmt.Errorf("flow %q: transition names unknown From state %q", spec.Name, t.From)
		}
		if _, toOK := byDir[t.To]; !toOK {
			return nil, fmt.Errorf("flow %q: transition names unknown To state %q", spec.Name, t.To)
		}
		if t.From == t.To {
			return nil, fmt.Errorf("flow %q: transition %s -> %s is a self-loop", spec.Name, t.From, t.To)
		}
		if from.Terminal {
			return nil, fmt.Errorf("flow %q: transition %s -> %s starts from terminal state %s",
				spec.Name, t.From, t.To, t.From)
		}
		pair := [2]string{t.From, t.To}
		if seenPair[pair] {
			return nil, fmt.Errorf("flow %q: duplicate transition %s -> %s", spec.Name, t.From, t.To)
		}
		seenPair[pair] = true
		allowed[t.From] = append(allowed[t.From], t)
	}

	// Reachability is a real graph search from Initial over allowed
	// transitions -- not an incoming-edge test. Marking t.To reached for every
	// transition regardless of whether t.From is itself reachable would let a
	// disconnected island (states that only reach each other, never Initial)
	// pass silently; BFS is what actually proves "reachable from Initial".
	reached := map[string]bool{spec.Initial: true}
	queue := []string{spec.Initial}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, t := range allowed[cur] {
			if !reached[t.To] {
				reached[t.To] = true
				queue = append(queue, t.To)
			}
		}
	}
	for _, s := range spec.States {
		if !reached[s.Dir] {
			return nil, fmt.Errorf("flow %q: state %q is unreachable from Initial %q",
				spec.Name, s.Dir, spec.Initial)
		}
		// Converse of the "transition starts from terminal state" check above:
		// a non-terminal state with no outgoing transitions is a dead end
		// nothing can ever leave, which legalTargets would then mislabel
		// "(none -- terminal status)" for a status that isn't terminal.
		if !s.Terminal && len(allowed[s.Dir]) == 0 {
			return nil, fmt.Errorf("flow %q: state %q is not terminal but has no outgoing transitions",
				spec.Name, s.Dir)
		}
	}

	wipStates := make([]State, 0)
	byWIPKey := make(map[string]State, len(spec.States))
	for _, s := range boardStates {
		if s.WIPKey == "" {
			continue
		}
		if _, dup := byWIPKey[s.WIPKey]; dup {
			return nil, fmt.Errorf("flow %q: duplicate WIPKey %q", spec.Name, s.WIPKey)
		}
		byWIPKey[s.WIPKey] = s
		wipStates = append(wipStates, s)
	}

	requires := make(map[string][]Requirement, len(spec.States))
	for _, s := range spec.States {
		requires[s.Dir] = slices.Clone(s.Requires)
	}

	activeStates := make([]State, 0, len(spec.States))
	for _, s := range spec.States {
		if s.Terminal || s.Dir == initial.Dir {
			continue
		}
		activeStates = append(activeStates, s)
	}

	return &Definition{
		name:         spec.Name,
		states:       slices.Clone(spec.States),
		boardStates:  boardStates,
		byDir:        byDir,
		byName:       byName,
		transitions:  slices.Clone(spec.Transitions),
		allowed:      allowed,
		initial:      initial,
		depSat:       depSat,
		pickup:       pickup,
		wipStates:    wipStates,
		byWIPKey:     byWIPKey,
		requires:     requires,
		activeStates: activeStates,
	}, nil
}

// validateRequirements checks one state's Requires rows: every field is
// non-empty (an empty Hint would render a dangling em dash in every
// violation message), Sub is already in internal/ticket's normalised form as
// far as a leaf package can tell (lower-case, trimmed — the fuller
// normalisation rule, e.g. stripping a leading ordinal, is
// internal/ticket's), Severity is one of the two legal values, and no two
// rows target the same (Section, Sub) pair.
func validateRequirements(flowName, dir string, reqs []Requirement) error {
	seen := make(map[[2]string]bool, len(reqs))
	for _, r := range reqs {
		if r.Section == "" {
			return fmt.Errorf("flow %q: state %q has a Requirement with an empty Section", flowName, dir)
		}
		if r.Label == "" {
			return fmt.Errorf("flow %q: state %q Requirement %q has an empty Label", flowName, dir, r.Section)
		}
		if r.Hint == "" {
			return fmt.Errorf("flow %q: state %q Requirement %q has an empty Hint", flowName, dir, r.Section)
		}
		if r.Sub != strings.ToLower(strings.TrimSpace(r.Sub)) {
			return fmt.Errorf("flow %q: state %q Requirement %q has an un-normalised Sub %q (want lower-case, trimmed)",
				flowName, dir, r.Section, r.Sub)
		}
		if r.Severity != Blocking && r.Severity != Advisory {
			return fmt.Errorf("flow %q: state %q Requirement %q has an invalid Severity %v", flowName, dir, r.Section, r.Severity)
		}
		pair := [2]string{r.Section, r.Sub}
		if seen[pair] {
			return fmt.Errorf("flow %q: state %q has a duplicate Requirement (Section %q, Sub %q)",
				flowName, dir, r.Section, r.Sub)
		}
		seen[pair] = true
	}
	return nil
}

// MustNew is New, panicking on error. Used only for the built-in registered
// definitions (brine.go), whose validity is pinned by a real test
// (TestBrineDefinitionValidates) rather than relied on via this panic alone.
func MustNew(spec Spec) *Definition {
	d, err := New(spec)
	if err != nil {
		panic(err)
	}
	return d
}
