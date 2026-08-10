package flow

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

// TestBrineMatchesLegacyLiterals is the golden test: it pins the brine
// definition against literal expectations written out by hand from the
// pre-T-080 source (internal/ticket.Statuses, internal/move.allowed,
// internal/board.boardOrder/sectionHeading/SectionColumns), so the extraction
// this ticket makes is provably a rename, not a behaviour change.
func TestBrineMatchesLegacyLiterals(t *testing.T) {
	def := Default()
	if def.Name() != "brine" {
		t.Fatalf("Name() = %q, want brine", def.Name())
	}

	wantStates := []State{
		{Dir: "1-to-do", Name: "TO DO", Heading: "TO DO (impact order, per child)", ImpactOrder: true, Columns: ColumnsBacklog},
		{Dir: "2-ready", Name: "READY", Heading: "READY (impact order, per child)", ImpactOrder: true, Columns: ColumnsBacklog},
		{Dir: "3-in-development", Name: "IN DEVELOPMENT", Heading: "IN DEVELOPMENT", Columns: ColumnsActive, WIPKey: "wip_in_development"},
		{Dir: "4-in-review", Name: "IN REVIEW", Heading: "IN REVIEW", Columns: ColumnsActive, WIPKey: "wip_in_review"},
		{Dir: "5-rework", Name: "REWORK", Heading: "REWORK", Columns: ColumnsRework},
		{Dir: "6-done", Name: "DONE", Heading: "DONE", Terminal: true, Columns: ColumnsDone},
		{Dir: "7-dropped", Name: "DROPPED", Heading: "DROPPED", Terminal: true, Columns: ColumnsDropped},
	}
	states := def.States()
	if len(states) != len(wantStates) {
		t.Fatalf("States() has %d entries, want %d", len(states), len(wantStates))
	}
	for i, want := range wantStates {
		if states[i] != want {
			t.Errorf("States()[%d] = %+v, want %+v", i, states[i], want)
		}
	}

	wantBoardOrder := []string{"IN DEVELOPMENT", "IN REVIEW", "REWORK", "READY", "TO DO", "DONE", "DROPPED"}
	boardStates := def.BoardStates()
	if len(boardStates) != len(wantBoardOrder) {
		t.Fatalf("BoardStates() has %d entries, want %d", len(boardStates), len(wantBoardOrder))
	}
	for i, name := range wantBoardOrder {
		if boardStates[i].Name != name {
			t.Errorf("BoardStates()[%d].Name = %q, want %q", i, boardStates[i].Name, name)
		}
	}

	wantTransitions := []Transition{
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
		{From: "5-rework", To: "4-in-review", Kind: Forward},
		{From: "5-rework", To: "7-dropped", Kind: Abort},
	}
	transitions := def.Transitions()
	if len(transitions) != len(wantTransitions) {
		t.Fatalf("Transitions() has %d entries, want %d", len(transitions), len(wantTransitions))
	}
	for i, want := range wantTransitions {
		if transitions[i] != want {
			t.Errorf("Transitions()[%d] = %+v, want %+v", i, transitions[i], want)
		}
	}

	if def.Initial().Dir != "1-to-do" {
		t.Errorf("Initial().Dir = %q, want 1-to-do", def.Initial().Dir)
	}
	if def.DependencySatisfied().Dir != "6-done" {
		t.Errorf("DependencySatisfied().Dir = %q, want 6-done", def.DependencySatisfied().Dir)
	}
}

// TestBrineDefinitionValidates confirms MustNew(brine's Spec) cannot panic at
// init: brine is built with MustNew, so this is the test that would catch a
// broken built-in before anything else does.
func TestBrineDefinitionValidates(t *testing.T) {
	if _, err := New(Spec{
		Name:                brine.name,
		States:              brine.States(),
		BoardOrder:          namesOf(brine.BoardStates()),
		Transitions:         brine.Transitions(),
		Initial:             brine.Initial().Dir,
		DependencySatisfied: brine.DependencySatisfied().Dir,
	}); err != nil {
		t.Errorf("re-validating brine's own Spec failed: %v", err)
	}
}

func namesOf(states []State) []string {
	names := make([]string, len(states))
	for i, s := range states {
		names[i] = s.Name
	}
	return names
}

// minimalSpec is a valid two-state, one-transition Spec, cloned and mutated by
// each TestSpecValidationRejects case so every case starts from something New
// would otherwise accept.
func minimalSpec() Spec {
	return Spec{
		Name: "test",
		States: []State{
			{Dir: "a", Name: "A", Heading: "A", Columns: ColumnsBacklog},
			{Dir: "b", Name: "B", Heading: "B", Terminal: true, Columns: ColumnsDone},
		},
		BoardOrder:          []string{"A", "B"},
		Transitions:         []Transition{{From: "a", To: "b", Kind: Forward}},
		Initial:             "a",
		DependencySatisfied: "b",
	}
}

// TestSpecValidationRejects is table-driven over every invariant New checks,
// one case per rule.
func TestSpecValidationRejects(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*Spec)
	}{
		{"empty Name", func(s *Spec) { s.Name = "" }},
		{"no States", func(s *Spec) { s.States = nil }},
		{"empty Dir", func(s *Spec) { s.States[0].Dir = "" }},
		{"Dir contains a slash", func(s *Spec) { s.States[0].Dir = "a/b" }},
		{"Dir is .", func(s *Spec) { s.States[0].Dir = "." }},
		{"Dir is ..", func(s *Spec) { s.States[0].Dir = ".." }},
		{"empty Name on a state", func(s *Spec) { s.States[0].Name = "" }},
		{"duplicate Dir", func(s *Spec) { s.States[1].Dir = s.States[0].Dir }},
		{"duplicate Name", func(s *Spec) { s.States[1].Name = s.States[0].Name }},
		{"unknown Columns profile", func(s *Spec) { s.States[0].Columns = ColumnProfile("bogus") }},
		{"BoardOrder wrong length", func(s *Spec) { s.BoardOrder = []string{"A"} }},
		{"BoardOrder names unknown state", func(s *Spec) { s.BoardOrder = []string{"A", "NOPE"} }},
		{"BoardOrder repeats a state", func(s *Spec) { s.BoardOrder = []string{"A", "A"} }},
		{"Initial unknown", func(s *Spec) { s.Initial = "nope" }},
		{"Initial terminal", func(s *Spec) { s.Initial = "b" }},
		{"DependencySatisfied unknown", func(s *Spec) { s.DependencySatisfied = "nope" }},
		{"transition unknown From", func(s *Spec) { s.Transitions[0].From = "nope" }},
		{"transition unknown To", func(s *Spec) { s.Transitions[0].To = "nope" }},
		{"transition self-loop", func(s *Spec) { s.Transitions[0].To = s.Transitions[0].From }},
		{"transition starts from terminal", func(s *Spec) {
			s.Transitions[0] = Transition{From: "b", To: "a", Kind: Forward}
		}},
		{"duplicate transition", func(s *Spec) {
			s.Transitions = append(s.Transitions, s.Transitions[0])
		}},
		{"unreachable state", func(s *Spec) {
			s.States = append(s.States, State{Dir: "c", Name: "C", Heading: "C", Columns: ColumnsBacklog})
			s.BoardOrder = append(s.BoardOrder, "C")
		}},
		{"unreachable island (has incoming edges from each other, but no path from Initial)", func(s *Spec) {
			// c and d each have an incoming transition, so an incoming-edge test
			// ("mark t.To reached for every transition") would wrongly accept this
			// spec; only a real reachability search from Initial catches it. This
			// is the T-080 review's finding B3, reproduced as a regression case.
			s.States = append(s.States,
				State{Dir: "c", Name: "C", Heading: "C", Columns: ColumnsBacklog},
				State{Dir: "d", Name: "D", Heading: "D", Columns: ColumnsBacklog},
			)
			s.BoardOrder = append(s.BoardOrder, "C", "D")
			s.Transitions = append(s.Transitions,
				Transition{From: "c", To: "d", Kind: Forward},
				Transition{From: "d", To: "c", Kind: Backward},
			)
		}},
		{"non-terminal state has no outgoing transitions", func(s *Spec) {
			// c is reachable (a -> c) but, unlike b, is not Terminal and has no
			// outgoing transitions of its own -- a dead end nothing can ever
			// leave. legalTargets would mislabel it "(none -- terminal status)"
			// for a status the Spec never marked terminal. T-080 review finding B3.
			s.States = append(s.States, State{Dir: "c", Name: "C", Heading: "C", Columns: ColumnsBacklog})
			s.BoardOrder = append(s.BoardOrder, "C")
			s.Transitions = append(s.Transitions, Transition{From: "a", To: "c", Kind: Forward})
		}},
		{"duplicate WIPKey", func(s *Spec) {
			s.States[0].WIPKey = "k"
			s.States[1].WIPKey = "k"
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := minimalSpec()
			c.break_(&s)
			if _, err := New(s); err == nil {
				t.Errorf("New() = nil error, want a rejection for %q", c.name)
			}
		})
	}
}

// TestRequiresReasonMatchesKinds asserts the exact set of brine (from, to)
// pairs that require a --reason: every kind but Forward — 8 of the 13
// transitions, matching internal/move's pre-T-080 requiresReason exactly
// (including that 5-rework -> 4-in-review, a scoped re-review, is Forward and
// requires none).
func TestRequiresReasonMatchesKinds(t *testing.T) {
	def := Default()
	wantReason := map[[2]string]bool{
		{"1-to-do", "2-ready"}:              false,
		{"1-to-do", "7-dropped"}:            true,
		{"2-ready", "3-in-development"}:     false,
		{"2-ready", "1-to-do"}:              true,
		{"2-ready", "7-dropped"}:            true,
		{"3-in-development", "4-in-review"}: false,
		{"3-in-development", "2-ready"}:     true,
		{"3-in-development", "7-dropped"}:   true,
		{"4-in-review", "6-done"}:           false,
		{"4-in-review", "5-rework"}:         true,
		{"4-in-review", "7-dropped"}:        true,
		{"5-rework", "4-in-review"}:         false,
		{"5-rework", "7-dropped"}:           true,
	}
	if len(wantReason) != 13 {
		t.Fatalf("test table has %d pairs, want 13", len(wantReason))
	}
	for pair, want := range wantReason {
		if got := def.RequiresReason(pair[0], pair[1]); got != want {
			t.Errorf("RequiresReason(%q, %q) = %v, want %v", pair[0], pair[1], got, want)
		}
	}
}

// TestByTokenForms ports the table TestStatusByToken (internal/ticket, pre-T-080)
// exercised, so the parser behind every `pickle ticket move <token>` keeps its
// own regression coverage now that it lives on Definition.ByToken instead of a
// package-level function. Decision 3 required the three accepted forms come
// along unchanged; this is what proves it, not a comment claiming it moved.
func TestByTokenForms(t *testing.T) {
	def := Default()
	cases := []struct {
		tok string
		dir string
		ok  bool
	}{
		{"3-in-development", "3-in-development", true}, // dir name
		{"in-development", "3-in-development", true},   // dir minus number
		{"IN DEVELOPMENT", "3-in-development", true},   // display name (spaces)
		{"In-Development", "3-in-development", true},   // case-insensitive
		{"to-do", "1-to-do", true},
		{"1-to-do", "1-to-do", true},
		{"done", "6-done", true},
		{"dropped", "7-dropped", true},
		{"nonsense", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := def.ByToken(tc.tok)
		if ok != tc.ok {
			t.Errorf("ByToken(%q) ok=%v, want %v", tc.tok, ok, tc.ok)
			continue
		}
		if ok && got.Dir != tc.dir {
			t.Errorf("ByToken(%q) = %q, want %q", tc.tok, got.Dir, tc.dir)
		}
	}
}

// TestByDirAndByName covers the two accessors ByToken is built on directly,
// including their negative cases — ByToken's table exercises them only
// through its dir-name and display-name forms, and never their miss path.
func TestByDirAndByName(t *testing.T) {
	def := Default()

	if s, ok := def.ByDir("1-to-do"); !ok || s.Name != "TO DO" {
		t.Errorf("ByDir(%q) = %+v, %v, want TO DO, true", "1-to-do", s, ok)
	}
	if s, ok := def.ByDir("6-done"); !ok || s.Name != "DONE" {
		t.Errorf("ByDir(%q) = %+v, %v, want DONE, true", "6-done", s, ok)
	}
	if _, ok := def.ByDir("nonsense"); ok {
		t.Error("ByDir(nonsense) ok=true, want false")
	}
	if _, ok := def.ByDir(""); ok {
		t.Error(`ByDir("") ok=true, want false`)
	}

	if s, ok := def.ByName("TO DO"); !ok || s.Dir != "1-to-do" {
		t.Errorf("ByName(%q) = %+v, %v, want 1-to-do, true", "TO DO", s, ok)
	}
	if s, ok := def.ByName("DONE"); !ok || s.Dir != "6-done" {
		t.Errorf("ByName(%q) = %+v, %v, want 6-done, true", "DONE", s, ok)
	}
	// ByName is exact-match, case-sensitive (unlike ByToken) — pin that it does
	// not silently fall back to token-style matching.
	if _, ok := def.ByName("to do"); ok {
		t.Error(`ByName("to do") ok=true, want false (ByName is case-sensitive, exact)`)
	}
	if _, ok := def.ByName("nonsense"); ok {
		t.Error("ByName(nonsense) ok=true, want false")
	}
	if _, ok := def.ByName(""); ok {
		t.Error(`ByName("") ok=true, want false`)
	}
}

// TestAccessorsReturnCopies pins the mutation guarantee every slice-returning
// accessor makes (the same one board.StatusOrder used to make on its own
// copy): a caller cannot reorder or corrupt the definition by mutating what it
// was handed.
func TestAccessorsReturnCopies(t *testing.T) {
	def := Default()

	states := def.States()
	states[0].Dir = "MANGLED"
	if def.States()[0].Dir == "MANGLED" {
		t.Error("States() exposed the definition's own slice")
	}

	boardStates := def.BoardStates()
	boardStates[0].Name = "MANGLED"
	if def.BoardStates()[0].Name == "MANGLED" {
		t.Error("BoardStates() exposed the definition's own slice")
	}

	transitions := def.Transitions()
	transitions[0].To = "MANGLED"
	if def.Transitions()[0].To == "MANGLED" {
		t.Error("Transitions() exposed the definition's own slice")
	}

	allowed := def.Allowed("1-to-do")
	if len(allowed) == 0 {
		t.Fatal("Allowed(1-to-do) returned nothing")
	}
	allowed[0].Dir = "MANGLED"
	if def.Allowed("1-to-do")[0].Dir == "MANGLED" {
		t.Error("Allowed() exposed the definition's own slice")
	}

	wip := def.WIPStates()
	if len(wip) == 0 {
		t.Fatal("WIPStates() returned nothing")
	}
	wip[0].Dir = "MANGLED"
	if def.WIPStates()[0].Dir == "MANGLED" {
		t.Error("WIPStates() exposed the definition's own slice")
	}
}

// TestFlowNamesMatchConfigLegalValues keeps the flow registry and
// internal/config's accepted flow names in agreement despite flow
// deliberately not importing config to check that itself (it stays a leaf
// package, per T-080 decision 4). It also confirms every WIP-limited state in
// every registered definition names a key internal/config.Project.WIPLimitFor
// actually resolves.
func TestFlowNamesMatchConfigLegalValues(t *testing.T) {
	names := Names()
	if len(names) != 1 || names[0] != config.DefaultFlowName {
		t.Fatalf("Names() = %v, want exactly [%q] (config.Validate accepts no other flow name today)",
			names, config.DefaultFlowName)
	}

	p := &config.Project{WIPInDevelopment: 3, WIPInReview: 5}
	for _, name := range names {
		def, ok := Get(name)
		if !ok {
			t.Fatalf("Get(%q) = false, but Names() just listed it", name)
		}
		for _, s := range def.WIPStates() {
			if _, ok := p.WIPLimitFor(s.WIPKey); !ok {
				t.Errorf("flow %q state %q names WIPKey %q, which Project.WIPLimitFor does not resolve",
					name, s.Dir, s.WIPKey)
			}
		}
	}
}

// layoutLineRE matches one `├── <dir>/ ← <DISPLAY NAME>  — ...` line from the
// rules' tree diagram (skill/resources/tickets-README.md's Layout section).
var layoutLineRE = regexp.MustCompile(`(?m)^[├└]── (\S+)/\s+← ([A-Z ]+?)\s{2,}—`)

// TestBrineStatesMatchShippedRules is the prose-drift guard: renaming a status
// in code without renaming it in the shipped rules now fails in CI, instead of
// silently leaving the rules document (and everything built on it) stale.
func TestBrineStatesMatchShippedRules(t *testing.T) {
	// filepath.Join("..", "..") is the existing CWD-relative idiom other
	// packages' tests use to reach the repo root; T-042 owns unifying it into
	// one CWD-independent helper, so this does not invent a sixth variant.
	path := filepath.Join("..", "..", "skill", "resources", "tickets-README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("tickets-README.md not found: %v", err)
	}

	type entry struct{ dir, name string }
	var got []entry
	for _, m := range layoutLineRE.FindAllStringSubmatch(string(data), -1) {
		got = append(got, entry{dir: m[1], name: m[2]})
	}

	def := Default()
	states := def.States()
	if len(got) != len(states) {
		t.Fatalf("rules' Layout block lists %d status dirs, want %d (one per brine state): %+v", len(got), len(states), got)
	}
	for i, s := range states {
		if got[i].dir != s.Dir {
			t.Errorf("Layout entry %d dir = %q, want %q", i, got[i].dir, s.Dir)
		}
		if got[i].name != s.Name {
			t.Errorf("Layout entry %d display name = %q, want %q", i, got[i].name, s.Name)
		}
	}
}
