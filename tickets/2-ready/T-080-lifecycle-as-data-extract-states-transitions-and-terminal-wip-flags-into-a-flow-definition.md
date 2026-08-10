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
- add `def *flow.Definition` as the **first parameter** to the four functions that need status
  vocabulary: `LoadAll(def, root)` (walks `def.States()` instead of `Statuses`),
  `HistoryEntries(def, text)`, `LastHistoryStatus(def, text)`, `LastHistoryReason(def, text)`
  (the transition scan's `statusExists` becomes `def.ByName`). `MergeLine`, `HasMergeLine`,
  `OutcomeMissing`, `Scaffold`, `SplitID`, `ValidID` and `ValidGrade` are untouched — they carry
  no status vocabulary.
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
2. **No status literal survives outside the definition** — both must print nothing:
   ```
   rg -n '"[1-7]-[a-z-]+"' --glob '*.go' --glob '!*_test.go' --glob '!internal/flow/**'
   rg -n '"(TO DO|READY|IN DEVELOPMENT|IN REVIEW|REWORK|DONE|DROPPED)"' \
      --glob '*.go' --glob '!*_test.go' --glob '!internal/flow/**'
   ```
   (Test files and prose — the scaffold template, doc comments, the skill payload — are out of
   scope for the guard; the prose is covered by `TestBrineStatesMatchShippedRules`.)
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
no changed rule. `docs/user-manual/concepts/lifecycle.adoc`, `concepts/the-flow.adoc` and
`cli-reference.adoc` describe the seven statuses and `pickle flow list` exactly as they will
still behave — leave them alone (acceptance step 5 checks that). The shipped skill payload
(`skill/**`) is likewise untouched; `TestBrineStatesMatchShippedRules` is how the code now stays
tied to it.

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

<!-- empty until IN REVIEW -->

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
