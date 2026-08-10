package flow

// brine is brine's whole lifecycle: the seven statuses, the board's render
// order and heading text, which sections sort by impact, which board column
// profile each section uses, which two states are WIP-limited (and under
// which pickle.toml key), the 13 legal transitions with their kinds, the
// state a fresh ticket lands in, and the state a dependency must reach before
// a dependent may be picked up.
//
// This is the only place to edit to add a status, rename one, or change a WIP
// rule — that is this file's entire reason to exist (T-080). The WIP key
// strings below must match internal/config's WIPKeyInDevelopment /
// WIPKeyInReview constants (and their pickle.toml tags); flow.go deliberately
// does not import internal/config to check that itself (it stays a leaf
// package), so TestFlowNamesMatchConfigLegalValues in flow_test.go is what
// keeps the two in agreement.
var brine = MustNew(Spec{
	Name: "brine",

	// Lifecycle order: the order a ticket walks tickets/ in, and the order
	// internal/install scaffolds directories in.
	States: []State{
		{
			Dir: "1-to-do", Name: "TO DO",
			Heading:     "TO DO (impact order, per child)",
			ImpactOrder: true, Columns: ColumnsBacklog,
		},
		{
			Dir: "2-ready", Name: "READY",
			Heading:     "READY (impact order, per child)",
			ImpactOrder: true, Columns: ColumnsBacklog,
		},
		{
			Dir: "3-in-development", Name: "IN DEVELOPMENT",
			Heading: "IN DEVELOPMENT",
			Columns: ColumnsActive, WIPKey: "wip_in_development",
		},
		{
			Dir: "4-in-review", Name: "IN REVIEW",
			Heading: "IN REVIEW",
			Columns: ColumnsActive, WIPKey: "wip_in_review",
		},
		{
			Dir: "5-rework", Name: "REWORK",
			Heading: "REWORK",
			Columns: ColumnsRework,
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
})
