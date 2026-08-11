package flow

// brine is brine's whole lifecycle: the seven statuses, the board's render
// order and heading text, which sections sort by impact, which board column
// profile each section uses, which two states are WIP-limited (and under
// which pickle.toml key), each state's gate table (T-081), the 13 legal
// transitions with their kinds, the state a fresh ticket lands in, the state
// a dependency must reach before a dependent may be picked up, and the state
// a ticket is built in (Pickup).
//
// This is the only place to edit to add a status, rename one, change a WIP
// rule, or change what a state requires — that is this file's entire reason
// to exist (T-080, T-081). The WIP key strings below must match
// internal/config's WIPKeyInDevelopment / WIPKeyInReview constants (and their
// pickle.toml tags); flow.go deliberately does not import internal/config to
// check that itself (it stays a leaf package), so
// TestFlowNamesMatchConfigLegalValues in flow_test.go is what keeps the two
// in agreement.
//
// outcomeStated, readyGate and reviewRecorded are brine's three gate-table
// building blocks, composed into each state's Requires below rather than
// repeated ~32 times as literal rows:
//
//   - outcomeStated (Advisory) sits on every non-terminal state — it is
//     T-083's original "## Outcome is missing" warning, generalised into the
//     table rather than hand-rolled in internal/audit. Its message must stay
//     byte-identical to what T-083 shipped; TestGateViolationMessages pins
//     that in internal/ticket.
//   - readyGate (Blocking, 8 rows: the ## Implementation Plan section plus
//     its seven ### READY-gate steps, rules §4.1-§4.7 in order) sits on
//     2-ready through 5-rework — a ticket must be able to show its plan is
//     complete for as long as it is meant to be acted on, not only at the
//     instant it entered READY.
//   - reviewRecorded (Blocking) sits on 5-rework only: a rework's whole scope
//     is the findings recorded in ## Review (rules §5), so a ticket cannot
//     sit in rework without them written down.
//
// Neither terminal state (6-done, 7-dropped) declares anything: a permanent
// archive is exactly what T-083 exempted by hand (`!st.Terminal`) before this
// ticket, and an unconditional residency check would otherwise warn on an
// old, closed ticket forever for no reader who is ever going to act on it.
// New still permits a terminal state to declare rows — a flow requiring
// something to *enter* DONE is legitimate — brine simply has nothing to ask.
var outcomeStated = Requirement{
	Section:  "Outcome",
	Label:    "Outcome",
	Hint:     "say what changes when this ships, in user-observable terms",
	Severity: Advisory,
}

var readyGate = []Requirement{
	{Section: "Implementation Plan", Label: "Implementation Plan",
		Hint: "fill in the plan before moving past READY", Severity: Blocking},
	{Section: "Implementation Plan", Sub: "feature branch", Label: "feature branch",
		Hint: "name the branch, cut inside the target child's repo (rules §4.1)", Severity: Blocking},
	{Section: "Implementation Plan", Sub: "prerequisite", Label: "prerequisite gate",
		Hint: `state it, or write "none" (rules §4.2)`, Severity: Blocking},
	{Section: "Implementation Plan", Sub: "confirmed", Label: "confirmed design decisions",
		Hint: "name the decisions the implementer must honour (rules §4.3)", Severity: Blocking},
	{Section: "Implementation Plan", Sub: "tasks", Label: "tasks",
		Hint: "list concrete tasks with exact paths in the target child (rules §4.4)", Severity: Blocking},
	{Section: "Implementation Plan", Sub: "acceptance test", Label: "acceptance test",
		Hint: "give runnable commands and expected results (rules §4.5)", Severity: Blocking},
	{Section: "Implementation Plan", Sub: "docs", Label: "docs update",
		Hint: `name the docs to update, or write "no user-facing surface" (rules §4.6)`, Severity: Blocking},
	{Section: "Implementation Plan", Sub: "finish", Label: "finish",
		Hint: "write the summary + suggested commit message step (rules §4.7)", Severity: Blocking},
}

var reviewRecorded = Requirement{
	Section:  "Review",
	Label:    "Review",
	Hint:     "record the blocking findings this rework exists to fix (rules §5)",
	Severity: Blocking,
}
var brine = MustNew(Spec{
	Name: "brine",

	// Lifecycle order: the order a ticket walks tickets/ in, and the order
	// internal/install scaffolds directories in.
	States: []State{
		{
			Dir: "1-to-do", Name: "TO DO",
			Heading:     "TO DO (impact order, per child)",
			ImpactOrder: true, Columns: ColumnsBacklog,
			Requires: []Requirement{outcomeStated},
		},
		{
			Dir: "2-ready", Name: "READY",
			Heading:     "READY (impact order, per child)",
			ImpactOrder: true, Columns: ColumnsBacklog,
			Requires: append([]Requirement{outcomeStated}, readyGate...),
		},
		{
			Dir: "3-in-development", Name: "IN DEVELOPMENT",
			Heading: "IN DEVELOPMENT",
			Columns: ColumnsActive, WIPKey: "wip_in_development",
			Requires: append([]Requirement{outcomeStated}, readyGate...),
		},
		{
			Dir: "4-in-review", Name: "IN REVIEW",
			Heading: "IN REVIEW",
			Columns: ColumnsActive, WIPKey: "wip_in_review",
			Requires: append([]Requirement{outcomeStated}, readyGate...),
		},
		{
			Dir: "5-rework", Name: "REWORK",
			Heading:  "REWORK",
			Columns:  ColumnsRework,
			Requires: append(append([]Requirement{outcomeStated}, readyGate...), reviewRecorded),
		},
		{
			Dir: "6-done", Name: "DONE",
			Heading:  "DONE",
			Terminal: true, Columns: ColumnsDone,
		},
		{
			Dir: "7-dropped", Name: "DROPPED",
			Heading:  "DROPPED",
			Terminal: true, Columns: ColumnsDropped,
		},
	},

	// Board render order: active work first.
	BoardOrder: []string{
		"IN DEVELOPMENT", "IN REVIEW", "REWORK", "READY", "TO DO", "DONE", "DROPPED",
	},

	Transitions: []Transition{
		{From: "1-to-do", To: "2-ready", Kind: Forward},
		{From: "1-to-do", To: "7-dropped", Kind: Abort},

		{From: "2-ready", To: "3-in-development", Kind: Forward},
		{From: "2-ready", To: "1-to-do", Kind: Backward},
		{From: "2-ready", To: "7-dropped", Kind: Abort},

		{From: "3-in-development", To: "4-in-review", Kind: Forward},
		{From: "3-in-development", To: "2-ready", Kind: Backward},
		{From: "3-in-development", To: "7-dropped", Kind: Abort},

		{From: "4-in-review", To: "6-done", Kind: Forward},
		{From: "4-in-review", To: "5-rework", Kind: SendBack},
		{From: "4-in-review", To: "7-dropped", Kind: Abort},

		{From: "5-rework", To: "4-in-review", Kind: Forward}, // scoped re-review: no reason required
		{From: "5-rework", To: "7-dropped", Kind: Abort},
	},

	Initial:             "1-to-do",
	DependencySatisfied: "6-done",
	Pickup:              "3-in-development",
})
