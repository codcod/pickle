---
id: T-080
title: lifecycle as data: extract states, transitions, and terminal/WIP flags into a flow definition
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: high
cost: L
---

# T-080 — lifecycle as data: extract states, transitions, and terminal/WIP flags into a flow definition

## Outcome

After this ships, brine's states, transitions and WIP rules live in one declarative flow definition instead of being re-typed as string literals across five Go packages, so adding or renaming a status means editing one file instead of finding every copy. Nothing a user sees changes: the same board bytes, the same marker block, the same `pickle flow list` output — now enumerated from the definition registry rather than from the config key.

## Description

brine's lifecycle is currently spelled out as Go literals in five packages. The seven status
directories, their board headings and terminal flags live in `internal/ticket/ticket.go`; the
legal transition graph and the definition of a "backwards" move live in `internal/move/move.go`;
and the strings `3-in-development` / `4-in-review` are then re-typed in `internal/board/board.go`
(WIP tallying and the rendered limit line), `internal/audit/audit.go` (WIP errors, the
dependency-must-be-done check) and `internal/install/install.go` (the marker block's WIP
bullet). Adding a status, renaming one, or changing a WIP rule means finding all of them.

Refinement found three more sites the paragraph above does not name, all of the same shape:
`internal/serve/view.go` switches on the display names `"IN DEVELOPMENT"`/`"IN REVIEW"` for the
dashboard's WIP badges (`:106`) and its health stats (`:350`); `internal/cli/ticket.go:133`
hard-codes `"1-to-do"` as the directory a new ticket lands in; and `internal/board/board.go:265`
decides impact-ordering by `name == "TO DO" || name == "READY"`. `internal/cli/project.go:237`
is the counter-example worth copying — it already asks `st.Terminal` instead of naming dirs.

This ticket extracts that into **one declarative flow definition**, with brine as the shipped
default: states (dir, heading, terminal), legal transitions, which moves are backwards, which
states are WIP-limited, and which state a dependency must reach before a dependent can be
picked up.

**It is worth doing at N=1.** The immediate payoff is the duplication: one source for a set
of strings currently maintained by hand in five places, and a real answer to the most common
real-world request a flow tool gets — *"add a QA column"*, *"rename our statuses"* — which
today is a code change. The second payoff is that it is the precondition for there ever being
a second flow (see T-073): with this and T-081 in place, a `brine-v` is a definition file plus
a prose addendum rather than an engine project.

**The trap to name up front:** `board audit`'s teeth. Several checks are only meaningful
because they know brine's semantics — "in development but dependency is not in `6-done`",
"child is over its `3-in-development` limit". Generalising must keep those checks sharp by
parameterising them from the definition, not by softening them into advisories. If the
refined plan cannot state which checks survive and how, the extraction is not ready.

### Overlap with T-042 — settled at refinement: there is none left

This section previously said the T-042 overlap had to be settled before either ticket went
READY. Re-verification (2026-08-10) found it **already settled, by a third ticket**: T-044's
review impact sweep dropped T-042's status-heading item from that epic's scope on 2026-07-26,
because the generated-board rewrite *deleted* the duplicated matchers (`board.ParseCells`,
`sync.matchStatus`, `board.sectionSpan`) outright. The heading-match loop now exists exactly
once, in the read-only `board.Parse`.

T-042's live scope is the skill-dir dry-run labels, the test payload root, and the ticket-id
**shape** regex unification (`idRE`/`filenameRE`/`board.rowRE`) — none of which is lifecycle
data. **No sequencing constraint in either direction, no absorption, nothing to negotiate.**
The two tickets may be picked up in any order; the only contact point is that both edit
`internal/ticket/ticket.go`, in different declarations.

Soft couplings: T-073 (the `flow` config key) is the seam this definition would eventually be
selected by; T-081 is the natural follow-on and depends on this; the dropped T-015
(consolidate board status-heading matching) is prior art on the same duplication.

**Explicitly not in scope: a user-authored flow.** The definition this ticket ships is data
*in the binary* — one Go file whose whole content is a declarative spec — not a file a project
can write. `config.Validate()` keeps rejecting every flow name but `"brine"`. "Add a QA column"
is still a code change after this ships; it is a one-declaration change instead of a hunt
through nine packages, which is the whole claim. Reading a definition from disk is the ticket
after T-081, and it inherits a validated, self-checking `flow.Spec` to parse into.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # pickle is the root-path child (path = ".")
git checkout main
git checkout -b feat/T-080-flow-definition
```

Commit locally as you go — this is a nine-package refactor and per-task commits are what make
the re-review diffable. Publish only per the project's commit policy (never push / open an MR
without approval); default to keeping the tidied history on merge rather than squashing
(root-path child default). Before pushing, verify the remote base is not behind local
(`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
print nothing).

### Prerequisite gate (hard)

None — `depends-on: []`. Two things a reader might expect to be prerequisites and are not:

- **T-042 is not sequenced against this ticket** (Description, "Overlap with T-042"). Pick up in
  any order.
- **T-073 already landed** the `flow` key, `config.DefaultFlowName`, `Config.FlowName()` and
  `pickle flow show|list`. This ticket consumes that seam; it does not build it.

T-081 declares `depends-on: [T-080]`, so it waits on this one being DONE *and merged*.

### Confirmed design decisions (do not deviate without asking)

1. **The definition is data in the binary, not a file a user writes.** `config.Validate()` keeps
   rejecting any `flow` value but `"brine"` — do **not** relax it here. Adding a status stays a
   code change; it just becomes a one-declaration change.
2. **The definition is threaded, not global.** Production code resolves it from config
   (`flow.ForName(cfg.FlowName())`) and passes `*flow.Definition` down. There is deliberately no
   package-level `flow.Brine` referenced directly from other packages, and no `Def` field cached
   on `ticket.Ticket`: a per-call parameter is what proves the extraction actually generalises,
   and it removes any "which definition did this ticket get?" ambiguity. Threading it is a
   signature change across ~10 internal functions — that cost is bought deliberately.
3. **No back-compatibility shims.** `ticket.Status`, `ticket.Statuses`, `ticket.StatusByDir`,
   `StatusByName` and `StatusByToken` **move** to `internal/flow` and are deleted from
   `internal/ticket`. No aliases, no forwarding wrappers — two names for one thing is the defect
   being fixed.
4. **`internal/flow` is a leaf package.** It imports nothing from `internal/` (not `config`, not
   `ticket`). Anything needing config is resolved at the call site or in a test.
5. **WIP limits keep today's config schema.** `pickle.toml` keeps exactly `wip_in_development`
   and `wip_in_review`; a state carries the *key name* it reads, and `config.Project` grows a
   lookup. Per-state limits in config are a schema change and are out of scope.
6. **Board column sets stay a board concept, chosen per state.** A state names a
   `flow.ColumnProfile`; `internal/board` maps profile → column list. This is what stops a new
   state rendering a headerless table (today `SectionColumns` returns `nil` for an unknown
   name).
7. **Every audit check keeps its teeth — parameterised, never softened.** Specifically: the
   dependency gate reads the definition's dependency-satisfied state (not the literal
   `"6-done"`); the `## Outcome` warning scopes by `State.Terminal` (not by naming two dirs);
   WIP errors iterate the definition's WIP states; status-dir existence iterates its states.
   Severity is unchanged for every one of them (error stays error, warning stays warning).
8. **`requiresReason` becomes a transition *kind*, and must reproduce today's behaviour
   exactly.** Verified during refinement over all 13 brine transitions: `Forward` needs no
   reason; `Backward` (`3→2`, `2→1`), `SendBack` (`4-in-review → 5-rework`) and `Abort`
   (`→ 7-dropped`, five of them) all do. `5-rework → 4-in-review` is `Forward` — a scoped
   re-review needs no `--reason` today and must not start needing one.
9. **Byte-identical output.** The rendered board, the `AGENTS.md` marker block, every CLI
   message and the dashboard HTML are unchanged by this ticket. Error strings that currently
   hard-code a status word (`"must be DONE and merged"`, `"dependency %s is DONE but has no
   'MERGED' History line"`) interpolate the definition's state name instead — which for brine
   renders the same bytes.

### Tasks

#### Task 1 — `internal/flow`: the types (new package)

New file `internal/flow/flow.go`. A `Spec` is plain exported data (what a definition author
writes); a `Definition` is a validated, immutable view of it (what everything else consumes).

```go
type ColumnProfile string   // Backlog, Active, Rework, Done, Dropped
type TransitionKind int     // Forward, Backward, SendBack, Abort

type State struct {
    Dir         string        // "3-in-development" — the status directory
    Name        string        // "IN DEVELOPMENT" — the display name
    Heading     string        // the board's `## ` heading text
    Terminal    bool
    ImpactOrder bool          // board sorts this section by impact, not by id
    Columns     ColumnProfile
    WIPKey      string        // "" when not WIP-limited; else a pickle.toml WIP key
}

type Transition struct{ From, To string; Kind TransitionKind } // dirs

type Spec struct {
    Name                string
    States              []State      // lifecycle (directory) order
    BoardOrder          []string     // display names, board render order
    Transitions         []Transition
    Initial             string       // dir a new ticket is created in
    DependencySatisfied string       // dir a depends-on target must reach before pickup
}
```

`Definition` holds a `Spec` in **unexported** fields and exposes accessors; slice-returning
accessors return `slices.Clone`d copies, preserving the mutation guarantee `board.StatusOrder`
makes today (its doc comment explains why — carry that reasoning over):

- `Name() string`
- `States() []State`, `BoardStates() []State` (board order), `Transitions() []Transition`
- `ByDir(dir) (State, bool)`, `ByName(name) (State, bool)`, `ByToken(tok) (State, bool)` —
  moved verbatim from `internal/ticket` (`StatusByToken`'s three accepted forms and its
  `statusNumRE` come along unchanged)
- `Initial() State`, `DependencySatisfied() State`
- `Allowed(fromDir string) []State` — legal targets, empty for a terminal state
- `Kind(from, to string) (TransitionKind, bool)`
- `RequiresReason(from, to string) bool` — `Kind != Forward`
- `WIPStates() []State` (board order), `StateByWIPKey(key) (State, bool)`

`func New(s Spec) (*Definition, error)` validates and freezes; `func MustNew(s Spec) *Definition`
panics (used only for the built-in, whose validity Task 9 pins with a real test). Validation
rejects:

- an empty `Name`, or zero states;
- a duplicate or empty `Dir`/`Name`; a `Dir` that is not a plain single path element (reject
  `/`, `\`, `.`, `..` — these become directory names under `tickets/`);
- a `BoardOrder` that is not exactly the set of state names, once each;
- an `Initial` that does not exist or is terminal;
- a `DependencySatisfied` that does not exist;
- a transition whose `From`/`To` does not exist, whose `From` is terminal, that is a self-loop,
  or that duplicates another `(From, To)` pair;
- a state unreachable from `Initial` (catches an orphaned column);
- a `Columns` value outside the known profiles;
- a duplicate non-empty `WIPKey`.

WIP keys are *not* checked against config here — decision 4 keeps this package a leaf; Task 9's
cross-check test owns that.

#### Task 2 — `internal/flow`: the brine definition + registry

- `internal/flow/brine.go` — **the one file this ticket exists to create.** A single `Spec`
  literal transcribing today's behaviour: seven states (dirs/names/terminal from
  `ticket.Statuses`; headings from `board.sectionHeading`, including the two
  `" (impact order, per child)"` suffixes; `ImpactOrder` true for TO DO and READY; `Columns`
  per `board.SectionColumns`; `WIPKey` on IN DEVELOPMENT and IN REVIEW), `BoardOrder` from
  `board.boardOrder`, the 13 transitions from `move.allowed` with the kinds fixed by decision 8,
  `Initial: "1-to-do"`, `DependencySatisfied: "6-done"`. Open the file with a comment saying
  what it is: the flow's whole lifecycle, and the only place to edit to add or rename a status.
- `internal/flow/registry.go` — `Get(name) (*Definition, bool)`, `Default() *Definition`,
  `Names() []string` (sorted), and `ForName(name) *Definition`, which falls back to `Default()`
  for an unknown name. Document at that fallback that `config.Validate()` is the enforcement
  point (an unknown name never reaches here on a loaded config) and that Task 9's cross-check
  test is what keeps the two sets agreeing.

#### Task 3 — `internal/config`: name the WIP keys, add the lookup

`internal/config/config.go`:

- exported constants `WIPKeyInDevelopment = "wip_in_development"` and
  `WIPKeyInReview = "wip_in_review"` (the same strings as the existing struct tags — add a
  comment tying tag and constant together, since Go cannot).
- `func (p *Project) WIPLimitFor(key string) (int, bool)` — the two keys above return the
  configured limit; anything else returns `(0, false)`. This is the only place a WIP key is
  resolved to a number.

No schema change, no new TOML keys, no change to `Validate()` (decision 1).

#### Task 4 — `internal/ticket`: delete the status table, take the definition

`internal/ticket/ticket.go`:

- **delete** `Status`, `Statuses`, `StatusByDir`, `StatusByName`, `StatusByToken`, `statusNumRE`
  and `statusExists` (they now live in `internal/flow`, Task 1).
- add `def *flow.Definition` as the **first parameter** to the functions that need status
  vocabulary: `LoadAll(def, root)` (walks `def.States()` instead of `Statuses`),
  `HistoryEntries(def, text)`, `LastHistoryStatus(def, text)`, `LastHistoryReason(def, text)`
  (the transition scan's `statusExists` becomes `def.ByName`). `OutcomeMissing`, `Scaffold`,
  `SplitID`, `ValidID` and `ValidGrade` are untouched — they carry no status vocabulary.
  *(Amended at review, finding N5 — this originally said "the four functions" and listed
  `MergeLine`/`HasMergeLine` among the untouched. Six functions actually take the parameter:
  the two above plus `NextNum`, which genuinely walked `Statuses`, and
  `MergeLine`/`HasMergeLine`, which call `historyKind` — and `historyKind` resolves a
  non-merge, non-created body through `transitionParts`, which needs the status vocabulary.
  The dependency is real, not stylistic.)*
- `Ticket` keeps `Dir string` and gains no definition field (decision 2).

The `HistoryEntry.Target` derivation (T-043 review R1) must keep resolving from the same first
physical line — this is a parameter change, not a restructuring of that scan.

#### Task 5 — `internal/move`: transitions from the definition

`internal/move/move.go`:

- **delete** the `allowed` map and `requiresReason`; `Move` resolves
  `def := flow.ForName(cfg.FlowName())` once at the top and uses `def.ByToken`, `def.ByDir`,
  `def.Allowed`, `def.RequiresReason`.
- `legalTargets` takes the definition and keeps sorting display names, including the
  `"(none — terminal status)"` branch for a terminal source.
- the WIP gate: `if st.WIPKey != ""` instead of naming two dirs; `checkWIP` resolves the limit
  via `cfg.Project(proj)` + `WIPLimitFor(st.WIPKey)`. Keep today's behaviour when the project is
  unregistered (`return nil` — an audit concern, not a move gate); do the same when the key does
  not resolve, and say so in a comment.
- the pickup dependency gate: compare against `def.DependencySatisfied().Dir`, and interpolate
  that state's `Name` into both error messages (decision 9).
- pass `def` into `ticket.LoadAll` and into `board.Regenerate`/`audit.Audit` as their new
  signatures require.

#### Task 6 — `internal/board`: order, headings, columns, WIP counts

`internal/board/board.go`:

- **delete** `boardOrder`, `sectionHeading` and `StatusOrder`. `Render` iterates
  `def.BoardStates()`, using `st.Heading` for the `## ` line.
- `SectionColumns(statusName string)` → `ColumnsFor(profile flow.ColumnProfile) []string`, one
  case per profile, with a `default:` that returns the Active profile's columns rather than
  `nil` (decision 6) — and a comment saying why a nil return was the hazard.
- `Sort(group, st flow.State, byID)` — `byImpact` becomes `st.ImpactOrder`.
- `WIPCounts(def, tickets) map[string]map[string]int` — child → state dir → count, tallied over
  `def.WIPStates()`. It stays the single tally behind the board sub-headings, the audit's limit
  check and the dashboard's badges (keep that doc comment's point). Delete the `WIP` struct's
  `InDevelopment`/`InReview` fields with it.
- `Render`'s per-child sub-heading: `(n/limit)` for any state with a `WIPKey`, resolving the
  limit through `WIPLimitFor`; the WIP-limit preamble lines likewise iterate
  `def.WIPStates()` — for brine this must emit the existing
  `` - `pickle`: `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1 `` line byte-for-byte.
- `Parse`/`ParseText` take `def` for their longest-name-first status matching; `Compare`,
  `Regenerate` and `Render` take `def` (or resolve it from the `cfg` they already hold — prefer
  an explicit parameter on the exported ones so callers cannot pass a mismatched pair).
- `cellFor` gains `def` only because `LastHistoryReason` now needs it; `renderRow` threads it.

#### Task 7 — `internal/audit`: the four teeth, parameterised

`internal/audit/audit.go` (decision 7 — severities unchanged):

- `Audit` resolves the definition from `cfg` and threads it into `LoadAll`, `board.Render`,
  `board.Compare` and the History helpers.
- Outcome warning (`:158`): `st, _ := def.ByDir(t.Dir); if !st.Terminal && ticket.OutcomeMissing(...)`.
- in-development dependency gate (`:217`–`:229`): the source state is the one the definition
  marks as the pickup state — use `def.DependencySatisfied()` for the target comparison and
  drive the loop off `t.Dir == <state with WIPKey WIPKeyInDevelopment>.Dir`. Keep the
  DONE-but-unmerged case a **warning** and the not-done case an **error**; interpolate state
  names into both messages.
- `auditWIP`: iterate `def.WIPStates()`, resolve each limit via `WIPLimitFor`, keep the message
  shape `WIP: child %q has %d tickets in %s (limit %d)` with the state's `Dir` interpolated —
  byte-identical for brine.
- `auditStatusDirs`: iterate `def.States()`.

#### Task 8 — the remaining callers

- `internal/install/install.go`: `scaffoldTickets` iterates `def.States()`; its
  `res.created("tickets/ (7 status dirs)")` label becomes count-derived (`fmt.Sprintf("tickets/
  (%d status dirs)", len(def.States()))`) — same bytes for brine. `writeBoard` passes `def` to
  `LoadAll`. `markerBlock`'s WIP bullet (`:855`) renders from `def.WIPStates()` +
  `WIPLimitFor`, emitting the identical `` `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1 `` text.
- `internal/sync/sync.go`: thread `def` into `LoadAll` and the status lookup at `:60`.
- `internal/cli/ticket.go:133`: the new-ticket path uses `def.Initial().Dir`, not `"1-to-do"`.
- `internal/cli/project.go:234`–`237`: `liveTicketsTargeting` takes `def` (it already asks
  `st.Terminal` — keep that, it is the pattern the rest of this ticket is moving toward).
- `internal/cli/flow.go`: `runFlowList` prints `flow.Names()` (one line, `brine` — unchanged
  output) instead of `cfg.FlowName()`; `runFlowShow` is unchanged. Leave `cli.go`'s help text
  ("exactly one, today") as is — still true.
- `internal/serve/serve.go`/`view.go`: `handler.load()` passes
  `flow.ForName(h.opts.Cfg.FlowName())`; `buildBoard` iterates `def.BoardStates()`; the
  `switch statusName` WIP badge (`:106`) becomes `if st.WIPKey != ""`, resolving the limit via
  `WIPLimitFor`; `buildHealth`'s `Stats` (`:350`) keeps its `InDevelopment`/`DevCap`/
  `InReview`/`RevCap` fields (they are template-bound) but populates them via
  `def.StateByWIPKey(config.WIPKeyInDevelopment)` / `...InReview`, with a comment recording
  that the dashboard's two badges are tied to the two config keys, not to dir strings, and that
  a flow with a third WIP state renders no third badge until the template grows one.

#### Task 9 — tests

New, in `internal/flow`:

- `TestBrineMatchesLegacyLiterals` — the golden test: assert the brine definition's states
  (dir, name, terminal), board order, headings, column profiles, the 13 transitions with their
  kinds, `Initial` and `DependencySatisfied` against literal expectations written out in the
  test. This is the pin that proves the extraction changed nothing; write the expectations by
  hand from the pre-refactor source, not by copying `brine.go`.
- `TestBrineDefinitionValidates` — `New(brineSpec)` returns no error (i.e. `MustNew` cannot
  panic at init).
- `TestSpecValidationRejects` — table-driven, one case per rule in Task 1's reject list.
- `TestRequiresReasonMatchesKinds` — every `(from, to)` pair in brine, asserting the exact
  set that requires a reason (decision 8's 8 pairs).
- `TestAccessorsReturnCopies` — mutating a returned slice does not affect the definition.
- `TestFlowNamesMatchConfigLegalValues` (may import `internal/config`) — `flow.Names()` is
  exactly the set `config.Validate()` accepts, and every `WIPKey` in every registered definition
  resolves through `Project.WIPLimitFor`. This is the test that keeps `ForName`'s fallback
  honest.
- `TestBrineStatesMatchShippedRules` — the prose-drift guard: read
  `skill/resources/tickets-README.md` (via `filepath.Join("..", "..")`, the idiom T-042 will
  unify — do not invent a sixth variant here) and assert the Layout block's
  `├── <dir>/` / `└── <dir>/` entries are exactly the definition's state dirs, in order, and
  that each state's display name appears on its line. Renaming a status in code without
  renaming it in the shipped rules now fails in CI.

Updated: `internal/ticket`, `internal/board`, `internal/move`, `internal/audit`,
`internal/sync`, `internal/install`, `internal/serve`, `internal/cli` test files, for the new
signatures. Two of them earn a new assertion rather than a mechanical edit:

- `internal/board`: `TestEveryColumnProfileHasColumns` — `ColumnsFor` returns a non-empty list
  for every profile `internal/flow` defines (the headerless-table hazard, decision 6).
- `internal/install`: assert `MarkerBlock(cfg)`'s WIP bullet is byte-identical to the string the
  pre-refactor code produced (paste the expected line literally).

Keep `internal/move`'s `TestSpawnedByDoesNotGatePickup` passing untouched — the non-gating
guarantee (T-029) must survive this refactor without its test being edited.

### Acceptance test

Run from the repo root, on the feature branch:

1. `just build && just test && just lint && just docs-check` — all green.
2. **No status literal survives outside the definition** — both must print nothing but
   comment lines:
   ```
   rg -n '"[1-7]-[a-z-]+"' --glob '*.go' --glob '!*_test.go' --glob '!internal/flow/**'
   rg -n '"(TO DO|READY|IN DEVELOPMENT|IN REVIEW|REWORK|DONE|DROPPED)"' \
      --glob '*.go' --glob '!*_test.go' --glob '!internal/flow/**'
   ```
   (Test files and prose — the scaffold template, doc comments, the skill payload — are out of
   scope for the guard; the prose is covered by `TestBrineStatesMatchShippedRules`.)
   *(Amended at review, finding N4 — "must print nothing" was written before the code existed
   and is literally false: three `// e.g. "1-to-do"`-style doc comments match
   (`internal/ticket/ticket.go:31`, `internal/board/board.go:29`,
   `internal/serve/view.go:33`). The parenthetical below always exempted doc comments, so the
   guard's intent held; only its "print nothing" phrasing was wrong.)*
3. **The board renders byte-identically.** Never run the WIP binary against this repo's own
   `tickets/` (AGENTS.md self-modify policy, and a `board sync` here would put bookkeeping on a
   feature branch):
   ```
   D=$(mktemp -d) && cp pickle "$D/pk" && mkdir -p "$D/tickets"
   cp pickle.toml "$D/" && cp -R tickets/. "$D/tickets/"
   (cd "$D" && ./pk board sync)
   diff <(grep -v '^Last updated:' tickets/BOARD.md) \
        <(grep -v '^Last updated:' "$D/tickets/BOARD.md")
   ```
   Expected: **no output** (the `Last updated:` line is the only legitimate difference — it
   carries the run date).
4. **The lifecycle still has teeth**, in the same throwaway tree:
   - `(cd "$D" && ./pk ticket move T-081 in-development)` → refused: T-081 depends on T-080,
     which is not DONE.
   - `(cd "$D" && ./pk ticket move T-080 in-review)` → refused as an illegal transition, listing
     the legal targets for TO DO.
   - `(cd "$D" && ./pk ticket move T-080 dropped)` → refused for a missing `--reason`.
   - `(cd "$D" && ./pk board audit)` → same errors/warnings as `(cd . && just build && ./pickle
     board audit)` printed before the branch (capture that output on `main` first, and diff).
   - `(cd "$D" && ./pk flow list && ./pk flow show)` → `brine` twice.
5. `git diff --stat main -- docs/ skill/` — expected empty: this ticket changes no shipped prose.

### Docs update (mandatory when user-facing)

**No user-facing surface changes** (decision 9): no new flag, no new command, no changed output,
no changed rule. `docs/user-manual/concepts/lifecycle.adoc` and `concepts/the-flow.adoc`
describe the seven statuses exactly as they will still behave — leave them alone (acceptance
step 5 checks that). The shipped skill payload (`skill/**`) is likewise untouched;
`TestBrineStatesMatchShippedRules` is how the code now stays tied to it.
*(Amended at review, finding N8 — this originally also claimed `cli-reference.adoc` "describe[s]
… `pickle flow list` exactly as [it] will still behave". It does not describe it at all: the
manual has no `[#cmd-flow]` section, no Overview-table row, and no mention of the `flow` key
(`rg -n 'flow list' docs/` returns nothing). The omission predates this ticket — T-073 shipped
the command undocumented — and **T-066 explicitly owns it**, naming `pickle flow show|list` as
one of its gaps. Nothing for this ticket to do; the claim was simply false.)*

`CHANGELOG.md`: one `### Changed` entry under `## [Unreleased]`, saying that brine's states,
transitions, terminal/WIP flags and gate targets now come from a single in-binary flow
definition (`internal/flow`), that behaviour and output are unchanged, and that a
project-authored flow is still not supported (`flow = "brine"` remains the only legal value).

### Finish (mandatory)

1. Acceptance test green: all five steps, including the byte-identical board diff and the
   two `rg` guards.
2. Docs: the CHANGELOG entry above; confirm `docs/` and `skill/` are untouched.
3. Interactive-rebase the WIP commits into atomic, correctly typed/scoped ones before presenting
   them (root-path child, rules §0) — the natural split is `feat(flow): …` for Tasks 1–3 and one
   `refactor(<pkg>): …` per consuming package, then `test(flow): …` and `docs(changelog): …`.
   Default to keeping that history on merge rather than squashing.
4. Write the summary: the new package, the deleted duplicates (name each of the nine sites), the
   signature changes, and — explicitly — the list of audit checks with their severities before
   and after, since keeping them sharp is this ticket's named trap.
5. Suggest a Conventional Commit message with the ticket id in brackets, e.g.
   `refactor(flow): make states, transitions and WIP flags one definition (T-080)`.
6. Commit locally on the ticket branch; publish only per the commit policy (approval required).
   `pickle ticket move T-080 in-review --reason "acceptance green"` and hand back.

## Review

Reviewed 2026-08-10 on `feat/T-080-flow-definition` (commit `fea296c`), ticket read from `main`
per the protocol. The implementer authored this ticket, so the implementation, quality,
consistency and docs audits were also run by an independent sub-agent briefed to be adversarial;
every finding it returned was then re-verified first-hand (probe tests compiled against
`internal/flow` and `internal/board`, a real `git merge` into a throwaway worktree, and
old-vs-new line anchoring against the merge-base `b6cc615`).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — empty set at first review (23 `.go` files, no prose),
      then **run at the re-review**: the rework added `CHANGELOG.md`, so the empty-set
      justification went stale and the pass was owed. Result in the re-review section below.
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented; bookkeeping committed (step 9)

### Step 2 — implementation audit

| item | verdict | evidence |
|---|---|---|
| Task 1 — `internal/flow/flow.go`: Spec/Definition/State/Transition, accessors, `New`/`MustNew` | met, **except one validation rule** | `flow.go`; accessors and profiles all present — but the "unreachable from Initial" rule is not implemented as specified: see **B3** |
| Task 2 — `brine.go` Spec literal + `registry.go` | met | 7 states, 13 transitions, `Initial`/`DependencySatisfied`; `Get`/`Default`/`Names`/`ForName` |
| Task 3 — `config`: WIP key constants + `Project.WIPLimitFor` | met | `config.go`; no schema change, `Validate()` still accepts only `"brine"` |
| Task 4 — `ticket`: delete the status table, thread the definition | met (scope wider than written) | status table and helpers deleted, no aliases; six functions took the parameter, not four — **N5** |
| Task 5 — `move`: transitions/WIP/dependency gates from the definition | met | `allowed`/`requiresReason` deleted; kinds reproduce the old table on all 13 pairs |
| Task 6 — `board`: order, headings, columns, WIP counts | met | `boardOrder`/`sectionHeading`/`StatusOrder`/`SectionColumns` deleted; `TestRenderGolden` unchanged and passing |
| Task 7 — `audit`: the four teeth, parameterised | met | all four verified below at unchanged severity |
| Task 8 — remaining callers (install/sync/cli/serve) | met | marker block byte-identical (`TestMarkerBlockGolden`, `TestSelfHostMarkerBlockIsCurrent`) |
| Task 9 — tests | **not met** | the `ByToken` table was deleted, not ported (**B2**); two new tests cannot fail (**N1**, **N2**) |
| Decisions 1, 2, 3, 4, 5, 6, 8, 9 | met (8/8) | `Validate()` untouched; definition threaded, no package-level cross-package default, no `Def` on `Ticket`; no aliases; `go list` shows `internal/flow` imports nothing from `internal/`; config schema unchanged; column profiles per state with a non-nil fallback; `RequiresReason` reproduces the old four cases exactly; output byte-identical |
| Decision 7 — audit teeth parameterised, never softened | met | see the teeth table below |
| Acceptance 1 — `just build`/`test`/`lint`/`docs-check` | met | re-run verbatim: 13 packages `ok`, `go vet` silent, `gofmt` clean, `snowball check` exit 0 |
| Acceptance 2 — the two `rg` guards | met in intent, false as written | three doc-comment hits — **N4** |
| Acceptance 3 — byte-identical board in a throwaway dir | met | `board sync: tickets/BOARD.md already in sync`; diff (ignoring `Last updated:`) empty. Also re-run on the **merged** tree — still identical |
| Acceptance 4 — lifecycle still has teeth | met | illegal transition, WIP limit, missing `--reason`, `flow list`/`show`, `board audit` 90/0/0 — all as before |
| Acceptance 5 — `docs/` and `skill/` untouched | met | `git diff --stat main...HEAD -- docs/ skill/` empty — but see **B1**: `CHANGELOG.md` is neither, and it was required |

**The audit's teeth** (the ticket's named trap), old vs new, on a synthetic tree with two tickets
over a limit of 1, a dependency parked in `2-ready/`, and a DONE ticket with no `## Outcome` —
base and branch output byte-identical:

| check | old trigger | new trigger | severity before → after |
|---|---|---|---|
| `## Outcome` missing | `t.Dir != "6-done" && != "7-dropped"` | `!st.Terminal` from the definition | warning → warning |
| dependency not done | `t.Dir == "3-in-development"`, `dt.Dir != "6-done"` | pickup state via `StateByWIPKey`, `def.DependencySatisfied()` | error → error |
| dependency done but unmerged | same loop, `!HasMergeLine` | same loop, `!HasMergeLine(def, …)` | warning → warning |
| per-child WIP over limit | two hard-coded fields | `def.WIPStates()` × `WIPLimitFor` | error → error |
| status dir missing / no `.gitkeep` | `ticket.Statuses` | `def.States()` | error / warning → unchanged |

**Merge safety (verified, not assumed).** This branch was cut at `b6cc615`, *before* T-090's
`linkifyURLs` rework landed on `main`, and both changed `internal/serve/view.go`. A real
`git merge feat/T-080-flow-definition` into a detached worktree at `main` resolved **cleanly**
(no conflict), kept `main`'s hardened `linkifyURLs` (`strings.IndexFunc(…, unicode.IsSpace)` and
`rel="noopener noreferrer"` both present; the old `urlRE` absent), and the merged tree passed
`go build`, all 13 packages' tests, `gofmt`, `go vet`, and the byte-identical board check. T-080's
hunks touch `buildBoard`/`newEntry`/`buildTicket`/`buildActivity`/`buildHealth` only — no overlap
with the linkify region. **No revert, no damage.**

### Findings

| id | severity | disposition | description | evidence | suggestion / resolution |
|---|---|---|---|---|---|
| B1 | **blocking** | — | The **mandatory `CHANGELOG.md` entry was never written.** The plan's "Docs update (mandatory)" section *and* Finish step 2 both require one `### Changed` entry under `## [Unreleased]`. Acceptance step 5 passes only because `CHANGELOG.md` is neither `docs/` nor `skill/` — the check could not see the omission it was sitting next to. | `git diff --name-only main...HEAD -- CHANGELOG.md` → empty; `grep -n 'T-080\|internal/flow' CHANGELOG.md` → no matches | **Resolved at rework** (`e015302`): added the entry the plan dictated, under `## [Unreleased]` / `### Changed`. |
| B2 | **blocking** | — | **A test was deleted under a comment that falsely claims it moved.** `internal/ticket/ticket_test.go:617` reads "TestStatusByToken moved to internal/flow (Definition.ByToken, T-080)". It did not move: `Definition.ByToken` has **zero** test coverage anywhere in the tree, and so do `ByDir`/`ByName` (exercised only transitively). The deleted table covered 10 cases — all three accepted token forms, case-insensitivity, and two negatives — for the parser behind every `pickle ticket move <token>`. Decision 3 explicitly required those three forms "come along unchanged"; the only test that proved it is gone, in a refactor whose entire claim is "moved, not changed". | `rg -n 'ByToken' --glob '*_test.go' .` → only the comment; deleted table at `b6cc615:internal/ticket/ticket_test.go:610-637` | **Resolved at rework** (`0fa870f`): added `TestByTokenForms` (the original 10-case table, verbatim) and `TestByDirAndByName` (their miss paths and `ByName`'s case sensitivity, neither exercised by the token table); corrected the `ticket_test.go` comment to name the tests that now carry the coverage. |
| B3 | **blocking** | — | **`New()`'s structural validation has two holes, and the test that claims to cover one passes for the wrong reason.** (a) The "unreachable state" rule is an *incoming-edge* test, not reachability: `reached` is seeded with `Initial` and then every transition's `To` is marked regardless of whether its `From` is reachable, so a disconnected island passes. The error string still says "is unreachable from Initial". The table's `"unreachable state"` case only adds a state with *no edges at all*, which the incoming-edge test happens to catch — false confidence. (b) The converse the `Spec.Transitions` doc claims is validated ("terminal and no-outgoing must agree — see New's validation") is not: a **non-terminal dead end** is accepted, and `move.legalTargets` then labels it `"(none — terminal status)"`. No effect on brine, but this is the "validated, self-checking `flow.Spec`" the ticket says T-081 and the read-from-disk follow-on inherit. | probe compiled into the package: island (`a`→`b`, `c`⇄`d`, initial `a`) → **accepted**; non-terminal dead end → **accepted**, `Allowed("b")` = `[]`; `flow.go:333,355,358` | **Resolved at rework** (`2d9068d`): replaced the incoming-edge test with a BFS from `spec.Initial` over `allowed`; added the converse check (`!s.Terminal && len(allowed[s.Dir]) == 0` → reject). Added both regression cases to `TestSpecValidationRejects` — the island, which the old incoming-edge test provably accepted (reproduced from the review's probe), and the non-terminal dead end. |
| N1 | non-blocking | noted | `TestBrineDefinitionValidates` is near-tautological: it rebuilds a `Spec` from the already-validated `Definition`'s own accessors and re-validates it, which `New` (being purely reconstructive) can never reject. The coverage it purports to add already exists — `brine` is built with `MustNew` at init, so a malformed built-in aborts *every* test in the package. | `flow_test.go:87-99`; `brine.go:16` inlines the literal into `MustNew`, which is why `New(brineSpec)` as the ticket wrote it was not possible | Closed with evidence. If it is ever made real, extract `var brineSpec = Spec{…}` and validate that literal. |
| N2 | non-blocking | noted | `TestEveryColumnProfileHasColumns` cannot fail. `ColumnsFor`'s `default:` returns the Active column set for *any* input, so the test is satisfied by `ColumnProfile("bogus")` too and cannot detect the drift it was written for (a profile added in `internal/flow` and left unmapped in `internal/board`). The `default` is itself load-bearing — it is the anti-headerless-table guard of decision 6 — so the guard and the test are in tension by design. Latent only: brine uses all five profiles and both lists are exhaustive, so no profile can currently go unmapped. | probe: `ColumnsFor(flow.ColumnProfile("bogus"))` = `[id title depends-on]`; `board.go` `default:` branch | Closed with evidence. The real fix — back the switch with an explicit `map[flow.ColumnProfile][]string` and assert `flow.ColumnProfiles()` ⊆ its keys — belongs with whoever first adds a profile. |
| N3 | non-blocking | fixed inline | `internal/audit/audit.go`'s comment above the Outcome check still described it as "scoped to the five non-terminal directories only — 6-done/ and 7-dropped/", naming the two literals the code stopped reading when this branch switched it to `state.Terminal`. Prose this branch made false. | `audit.go` comment vs. the `!st.Terminal` condition below it | Done in `c2b1489`: reworded to describe the definition's terminal flag, naming brine's two directories as the example rather than the mechanism. Comment-only; suite still green. |
| N4 | non-blocking | fixed inline | The plan's acceptance step 2 says both `rg` guards "must print nothing"; they print three doc-comment hits. The step's own parenthetical always exempted doc comments, so the guard's *intent* held — the phrasing was just false. Prose this branch authored. | `internal/ticket/ticket.go:31`, `internal/board/board.go:29`, `internal/serve/view.go:33` | Done above: step 2 now reads "nothing but comment lines", with the three hits named. |
| N5 | non-blocking | fixed inline | Task 4 named "the four functions" and listed `MergeLine`/`HasMergeLine` among the untouched; **six** took the parameter. `NextNum` genuinely walked `Statuses`; `MergeLine`/`HasMergeLine` call `historyKind`, which resolves a non-merge body through `transitionParts` — a real dependency. Flagged in commit `95fbeb9`'s message but the plan text stayed wrong. | `ticket.go:385,408` signatures vs. Task 4's list | Done above: Task 4's list corrected with the reason the two extra functions need it. |
| N6 | non-blocking | folded → **T-081** | The pickup gate in both `move.go` and `audit.go` identifies its own state as `def.StateByWIPKey(config.WIPKeyInDevelopment)` — the flow's most load-bearing gate keyed off an unrelated concern (WIP limits), and `!ok` skips the entire gate with no error or warning (fails open). Task 7 prescribed exactly this and brine is unaffected, but `Spec` has explicit `Initial` and `DependencySatisfied` fields and no `Pickup` — an asymmetry this branch introduced. | `move.go` pickup lookup; `audit.go` dependency-gate guard | Folded into **T-081**, which will edit `Spec` anyway (recorded in its History by this review's step 8): give `Spec` a `Pickup` field, or make the `!ok` case an audit error rather than a silent skip. |
| N7 | non-blocking | noted | `internal/serve` re-resolves the definition **7× per request** (`flow.ForName(…)` at seven call sites) instead of resolving once onto `handler`. | `serve.go:129,152,169,176,187,202,209` | Closed with evidence: a map lookup on a 1-entry registry, on a localhost read-only dashboard that re-reads the whole ticket tree per request. Cosmetic. |
| N8 | non-blocking | fixed inline | The plan's docs step claimed `cli-reference.adoc` "describe[s] … `pickle flow list` exactly as [it] will still behave". The manual does not describe it at all — no `[#cmd-flow]` section, no Overview row, no mention of the `flow` key. Prose this branch authored, and false when written. | `rg -n 'flow list\|flow show' docs/` → nothing; no `[#cmd-flow]` among the 12 `[#cmd-*]` anchors | Done above. The underlying gap is **pre-existing** (T-073 shipped the command undocumented) and **T-066 already owns it by name**, so nothing is owed here — only the false claim needed correcting. |
| N9 | non-blocking | noted | This refactor shifted line anchors cited by **T-013, T-038, T-056, T-065, T-066, T-079** (e.g. `board.go:240`, `move.go:62`, `ticket.go:182`, `serve/view.go:77`). | the `.go:NNN` citations in those tickets vs. current line numbers | Closed with evidence. Every one of those citations names its **symbol** alongside the number, so nothing is ambiguous and no encoded assumption is invalidated; chasing dozens of anchors after every refactor is churn that will be stale again next time. **T-042 and T-070 were patched** (step 8) because their cited declarations sit inside the neighbourhoods this branch rewrote. |

**Disposition summary.** 12 findings, **3 blocking** (B1 missing CHANGELOG entry, B2 deleted
`ByToken` test under a false "moved" comment, B3 reachability validation that does not check
reachability), 9 non-blocking: **4 fixed inline** (N3 stale code comment, N4/N5/N8 false claims in
this ticket's own plan), **1 folded** (N6 → T-081), **4 noted** (N1/N2 unfalsifiable new tests,
N7 redundant resolution, N9 line-anchor drift). **0 new tickets** — nothing here passes the
promotion test; all three blocking findings are fixed on this branch, not deferred.
→ `5-rework/`, scoped to B1, B2 and B3 alone.

### Rework (2026-08-10)

All three blocking findings fixed on the same branch, scope held to exactly B1/B2/B3 — no other
work folded in. `just build`/`test`/`lint`/`docs-check` re-run green; the two `rg` guards and the
throwaway-dir board-sync byte-identity re-run clean; the lifecycle gates (illegal transition, a
missing `--reason`, `flow list`/`show`) still refuse/print exactly as before. See the
"Resolved at rework" note in each of B1/B2/B3's own row above for the fix and its commit.
→ `4-in-review/` for a **scoped re-review** of B1, B2 and B3 only (protocol §6a) — not a
full re-audit.

### Scoped re-review (2026-08-10)

Per protocol §6a, this verified **only** B1/B2/B3 plus the rework's own blast radius — no
re-audit of the feature. Because this ticket's blocking findings were themselves *tests and
validation that could not fail*, each fix was checked by **mutation testing** (break the
production code, prove the new test goes red) rather than by watching it go green.

| finding | verdict | how it was verified |
|---|---|---|
| B1 | **resolved** | Entry present under `## [Unreleased]` / `### Changed` (`e015302`). Its three factual claims were each checked, not taken on trust: byte-identical board (re-run), unchanged audit conditions/severities (original review's teeth table), and `flow = "brine"` still the only legal value — a hand-written `flow = "kanban"` config is refused with `flow "kanban" is not a known flow (legal: brine)`. |
| B2 | **resolved** | `TestByTokenForms` (the original 10-case table, verbatim) + `TestByDirAndByName` present and green (`0fa870f`). Mutation-tested: removing `ToLower` (case-insensitivity) → red; making `ByName` case-insensitive → red; `ByDir` always reporting `ok` → red. **Two mutations initially slipped through** — see N10/N11 below; the lost coverage is nonetheless faithfully restored, so B2 itself is resolved. |
| B3 | **resolved** | BFS from `Initial` + the dead-end converse (`2d9068d`). Mutation-tested against the *original bug*: reverting to the incoming-edge form makes the new island case go red; deleting the dead-end check makes its case go red. The review's two original probes now reject with precise messages (`state "c" is unreachable from Initial "a"`; `state "b" is not terminal but has no outgoing transitions`). Also confirmed it does not over-reject: brine validates, and an unreachable *terminal* state is caught by the reachability rule. |

**Scope discipline.** `git diff --stat` over the three rework commits touches exactly the four
files the findings required (`CHANGELOG.md`, `internal/flow/flow.go`, `internal/flow/flow_test.go`,
`internal/ticket/ticket_test.go`) — no other work folded in.

**Rework blast radius.** `just build`/`test`/`lint`/`docs-check` green; both `rg` guards and the
throwaway-dir board byte-identity re-run clean; the lifecycle gates still refuse identically. The
merge into `main` was re-verified in a **detached** worktree: clean merge, all 13 packages pass,
board byte-identical on the merged tree, T-090's hardened `linkifyURLs` intact.

| id | severity | disposition | description | evidence | resolution |
|---|---|---|---|---|---|
| N10 | non-blocking | fixed inline | `TestByTokenForms`' comment claimed it proves "the three accepted forms come along unchanged". It cannot: brine's display names *are* its dir names minus the leading number, so "dir minus number" and "display name normalised" normalise to the **identical** string for all seven states. Prose authored during the rework, overstating its own test. | probe printing both derived forms per state: `to-do`/`to-do`, `in-development`/`in-development`, … — IDENTICAL ×7 | Comment corrected to state what the table does and does not prove (`4cca7b2`). |
| N11 | non-blocking | fixed inline | Consequence of N10: two of `ByToken`'s three branches were **not pinned by anything**. Deleting the `bare` branch or the `name` branch left the whole suite green. Not a regression — the pre-T-080 original had the same blind spot — but B2's fix inherited it while claiming otherwise. | mutations M1/M3 (drop either branch, keeping the var used so it compiles) → suite **green** | Added `TestByTokenDistinguishesAllThreeForms`, using a spec whose display Name is deliberately not its Dir minus the number (`1-open` / `OPEN WORK`) so all three forms differ, plus a whole-vs-substring assertion. Re-ran M1/M3/M8: all three now go **red** (`4cca7b2`). |
| N12 | non-blocking | noted | The dead-end rule added for B3, combined with the pre-existing "Initial must not be terminal" rule, makes a **single-state flow inexpressible**: the lone state may not be terminal (it is `Initial`) and may not be non-terminal (it would have no outgoing transitions). A narrowing the rework introduced as a side effect, not a decision. | probe: a one-state `inbox` spec is rejected with `state "1-inbox" is not terminal but has no outgoing transitions` | Closed with evidence: a single-state flow has no lifecycle to operate — no ticket could ever move — so it is degenerate rather than useful, and `Validate()` admits only brine regardless. Worth knowing when a project-authored flow becomes real (T-081 / the read-from-disk follow-on). |

**Docs-readability pass (step 4b), owed once the rework added `CHANGELOG.md`.** Ran the
readability reviewer over `CHANGELOG.md` — the ticket's only prose file. It returned 10
suggestions and **none of them touch the T-080 entry**; all ten target pre-existing entries
(T-089's two `[Unreleased]` bullets and several `[0.4.0]`/`[0.3.0]` ones). Nothing applied: those
are pre-existing rather than authored or falsified here, and rewriting already-released changelog
prose is churn with no reader benefit. The entry this ticket added drew no comment.

**Re-review disposition summary.** All **3 blocking findings resolved**; **0 remain**. 3 new
non-blocking findings, all from mutation-testing the fixes: **2 fixed inline** (N10 overstated
comment, N11 the unpinned branches it hid — split per rules §5, "a wrong comment *and* the
substantive defect it hid"), **1 noted** (N12). **0 new tickets.**
→ `6-done/`. Cumulative across both passes: 15 findings, 3 blocking (all fixed), 12 non-blocking
(6 fixed inline, 1 folded → T-081, 5 noted), 0 spawned.

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — patched by T-073's review impact sweep (step 8): the seam this ticket is written
  against now **exists**. T-073 shipped `flow` in `pickle.toml` with
  `config.DefaultFlowName = "brine"`, a `Config.FlowName()` accessor, and a `Validate()` that
  **rejects any value other than `"brine"`** (`internal/config/config.go`). That last part is the
  load-bearing detail for this ticket: introducing a second flow definition means relaxing that
  check, so plan it as an edit to `Validate()` rather than assuming the key already accepts an
  arbitrary name. `pickle flow list` also exists and prints exactly one entry today — it is the
  natural place to enumerate definitions once they are data. No assumption is invalidated
- 2026-08-10 — refined. The Description's "settle the T-042 overlap before either goes READY"
  requirement is **discharged, not deferred**: T-044's sweep already deleted the status-heading
  duplication on 2026-07-26, so T-042's live scope (dry-run labels, test payload root, id-shape
  regexes) does not touch lifecycle data and the two tickets are unsequenced. Three further
  duplication sites were found and folded into the Description (serve's WIP-badge switch,
  `cli/ticket.go`'s hard-coded `1-to-do`, board's impact-order name test). Scope confirmed as a
  single ticket (no independently schedulable split) and grade unchanged (high/high/L)
- 2026-08-10 — TO DO → READY: plan complete
- 2026-08-10 — READY → IN DEVELOPMENT: picked up
- 2026-08-10 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-10 — IN REVIEW → REWORK: review: 3 blocking (B1 mandatory CHANGELOG entry never written; B2 ByToken test deleted under a false 'moved' comment; B3 reachability validation is an incoming-edge test); 9 non-blocking (4 fixed inline, 1 folded, 4 noted)
- 2026-08-10 — REWORK → IN REVIEW: rework: B1 CHANGELOG entry added; B2 ByToken/ByDir/ByName coverage ported; B3 reachability is now a real BFS and non-terminal dead ends are rejected — back for scoped re-review
- 2026-08-10 — IN REVIEW → DONE: scoped re-review: B1/B2/B3 all resolved and mutation-verified; 3 new non-blocking (N10/N11 fixed inline, N12 noted); 0 blocking remain, 0 tickets spawned
