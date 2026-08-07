---
id: T-073
title: introduce brine as the flow's name: flow config key, prose, and a docs attribute
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: low
cost: M
---

# T-073 — introduce brine as the flow's name: flow config key, prose, and a docs attribute

## Description

The flow `pickle` installs has no name. It is described everywhere by what it is — "the
ticket flow", "the ticket-based, board-driven feature flow" — which was fine while it was
the only one, and is the thing blocking every conversation about there being more than one.
You cannot make something "the default" until it has a proper noun.

This ticket gives it one: **brine** — the medium things pickle in. It is deliberately the
*cheap half* of naming:

- a new optional `flow` key in `pickle.toml`, defaulting to `"brine"`, and a `pickle flow
  show` that prints it (with `pickle flow list` printing exactly one entry). The key earns
  its keep as the seam a second flow would later select, and as something `doctor` and
  `board audit` can report;
- the flow renamed **in prose**: the body of `skill/SKILL.md`, `resources/tickets-README.md`,
  `resources/review-protocol.md`, the `AGENTS.md`/`CLAUDE.md` marker block rendered by
  `install.go:markerBlock()` (and this repo's own marker, hand-edited inside this ticket's
  diff per the self-modify policy in `AGENTS.md`);
- the docs. Smaller than it looks: `docs/attributes.adoc` already defines
  `:skill-dir: .agents/skills/ticket-flow`, so the work is adding a `:flow: brine` attribute
  and routing the remaining literal occurrences across the six `.adoc` files through the
  attributes rather than hand-editing each.

**Nothing on disk moves.** The installed skill directory, the `.claude/skills/` symlink and
the `SKILL.md` frontmatter `name:` all keep saying `ticket-flow`. That is a deliberate
split, not an oversight: renaming paths is a migration problem in every project that has
already run `pickle install`, and it is isolated in T-074 so it can be scheduled — or
dropped — without holding this ticket hostage. The transitional wording is honest: *the
`ticket-flow` skill operates the `brine` flow*.

The standing argument against ever doing T-074 belongs here, because refinement of this
ticket is where it gets settled: `ticket-flow` is self-documenting and `brine` is opaque, and
if a catalogue of flows ever exists, self-documenting directory ids get **more** valuable,
not less. The recommendation on the table is that `brine` is the flow's proper name in
config, prose and docs, while on-disk ids stay descriptive.

Soft couplings, no hard dependency on any: T-080 (lifecycle as data) is the other half of
the seam this key selects; T-046 (make `doctor`/`upgrade` self-host-aware) touches the same
skill-symlink detection T-074 would; T-066 (close the CLI-surface documentation gaps) will
need to document `pickle flow` in `cli-reference.adoc` if it lands after this.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                     # pickle is the overarching project and its own child
git checkout main
git checkout -b feat/T-073-introduce-brine-name
```

Local WIP commits as work progresses; publish only after user approval (finalize, push, open
the MR — merging is the human's), per the project's configured commit policy.

### Prerequisite gate

None.

### Confirmed design decisions (do not deviate without asking)

1. **Nothing on disk moves.** The installed skill directory (`.agents/skills/ticket-flow/`),
   the `.claude/skills/` symlink, and `SKILL.md`'s frontmatter `name: ticket-flow` are
   untouched — every literal path reference in code, tests and docs stays exactly as it is.
   That is T-074's scope.
2. **`brine` is the flow's name in config, prose and docs — nowhere else, yet.** Every change
   below is a rename of *naming* text ("the ticket-flow skill", "## Ticket flow"); it is never
   applied to a path.
3. **Only `"brine"` is a legal `flow` value today.** No second flow exists (T-080/T-081 are the
   precondition for one), so `Config.Validate` rejects any other value rather than silently
   accepting a typo or a name nothing implements.
4. **The marker-block sentinels, not the heading text, anchor `injectMarker`.** Confirmed in
   `internal/install/install.go`: `MarkerBegin`/`MarkerEnd` (`<!-- pickle:begin -->` /
   `<!-- pickle:end -->`) are what `injectMarker` (install.go:729) searches for. Rewording the
   heading inside the block (`## Ticket flow (start here)`) is therefore cosmetic, not a
   marker-detection risk.
5. **`doctor` gains a passed-check line reporting the configured flow name; `board audit` does
   not.** `audit.Result` has no `Passed` list (unlike `doctor.Result`), and "flow: brine" is not
   a ticket invariant — bolting it onto `board audit`'s output would be scope creep dressed as
   parity.

### Tasks

#### Task 1 — `flow` config key

In `internal/config/config.go`:

- Add `Flow string \`toml:"flow,omitempty"\`` to the `Config` struct, next to `PayloadVersion`.
- Add `const DefaultFlowName = "brine"` alongside the existing per-child defaults.
- Add `func (c *Config) FlowName() string` returning `c.Flow` if set, else `DefaultFlowName`.
- In `Validate()`, reject an unknown value up front: `if c.Flow != "" && c.Flow != DefaultFlowName
  { return fmt.Errorf("pickle.toml: flow %q is not a known flow (legal: %s)", c.Flow,
  DefaultFlowName) }`.
- In `Render()`, mirror the existing `ReviewAddendum` pattern: emit the `flow` key right after
  the `payload_version` line, only `if c.Flow != ""`. **Quote it with `tomlQuote(c.Flow)`, not
  `%q`** — T-069 replaced every TOML-rendering `%q` in this function (Go's `%q` emits `\a`,
  `\v` and `\xNN`, none of which TOML accepts). The `fmt.Errorf("… flow %q …")` in `Validate()`
  above is untouched by that rule: it formats an *error message*, never file content.
- Since `flow` is a string field `Render()` quotes, add it to `Validate()`'s invalid-UTF-8
  check alongside `payload_version` and `review_addendum` (T-069 added that gate so a value
  that cannot round-trip never reaches the file).

#### Task 2 — `pickle flow show` / `pickle flow list`

New file `internal/cli/flow.go`: `runFlow(args)` dispatches `show`/`list`
(unknown subcommand → usage + `exitUsage`, matching `runProject`'s style). Both subcommands
take no positional args, load the config via the existing `loadConfig()` helper
(`internal/cli/project.go`), and print `cfg.FlowName()` on a line by itself — `list` prints
exactly one entry today, by design (rules for a future second flow apply to T-080/T-081, not
here). Wire `case "flow": return runFlow(args[1:])` into `Run`'s dispatch in
`internal/cli/cli.go`, and add a `flow show` / `flow list` line to `usage()`'s "Setup
commands:" block, directly after the `project remove` line.

#### Task 3 — `doctor` reports the configured flow name

In `internal/doctor/doctor.go`'s `checkConfig` (or a small sibling check called from `Check`
when `cfg != nil`): `r.ok(fmt.Sprintf("flow: %s", cfg.FlowName()))`. Informational only — no
new error or warning path.

#### Task 4 — rename the generated prose `pickle install` writes into new projects

In `internal/install/install.go`:

- `MarkerBlock()`'s returned string: `"## Ticket flow (start here)"` → `"## Brine (start
  here)"`; `"the **ticket-flow skill** at\`.agents/skills/ticket-flow/\`"` → `"the **brine
  skill** at \`.agents/skills/ticket-flow/\`"` (path unchanged, only the naming word moves).
- The `ticketsReadme` constant: both occurrences of "the ticket-flow skill" ("live in the
  ticket-flow skill:") → "the brine skill".
- Update `internal/install/testdata/markerblock.golden` to match exactly — it is the literal
  expected output of `MarkerBlock(cfg)` under a two-child fixture config and must stay
  byte-identical to what the function now renders, paths untouched.

#### Task 5 — rename in the skill payload prose

- `skill/SKILL.md`: reword the H1 (`# Ticket flow` → `# Brine`) and its opening sentence to
  introduce the name (e.g. "**Brine** is a lightweight, repo-native feature flow."); reword
  the "When to use" trigger label ("Install / scaffold the ticket flow" → "Install / scaffold
  brine"). Leave frontmatter `name: ticket-flow` and both literal `.agents/skills/ticket-flow/`
  path mentions (lines ~53, ~273) untouched.
- `skill/resources/tickets-README.md`: both "the ticket-flow skill" occurrences (the tree
  comment and "live in the ticket-flow skill and are…") → "the brine skill".
- `skill/resources/review-protocol.md` and `skill/resources/TEMPLATE.md`: no occurrences of
  "ticket-flow"/"ticket flow" today — confirmed by grep; no edit needed. Do not introduce one.

#### Task 6 — rename in this repo's own hand-written files (self-modify policy: by hand, in this diff)

- `AGENTS.md` (this repo's own; `CLAUDE.md` is a symlink to it, so no separate edit): the
  preamble sentence "work is planned and tracked through the very ticket flow it ships" →
  "…the very brine flow it ships"; the marker block body (between the `pickle:begin`/`
  pickle:end` sentinels) rewritten to **exactly** match Task 4's new `MarkerBlock()` output for
  this repo's actual config (single child `pickle`), so `pickle doctor` reports no marker drift.
  The `.agents/skills/ticket-flow/` path mention outside the marker stays untouched.
- `pickle.toml` (this repo's own, hand-written): add `flow = "brine"` directly after
  `payload_version`, with a one-line comment, exercising the new key in the one place that
  matters most — self-host.
- `README.md` (repo root): open with the name — "pickle installs and operates **brine**, a
  ticket-based, board-driven feature flow, in any project." — and reword the install-snippet
  comment `# lay down the ticket flow` → `# lay down brine`.

#### Task 7 — docs: a `:flow:` attribute, applied only to naming prose

- `docs/attributes.adoc`: add `:flow: brine` alongside the existing `:product:`/`:skill-dir:`
  attributes.
- Replace **naming** occurrences only (never a literal path) with `{flow}`, in exactly these
  spots: `docs/README.adoc:26`, `docs/user-manual/configuration.adoc:60`,
  `docs/user-manual/quickstart.adoc:34`, `docs/user-manual/your-first-project.adoc:21`,
  `docs/user-manual/concepts/the-flow.adoc:6` and `:43`, and
  `docs/user-manual/cli-reference.adoc:573` ("Starts a local, read-only web view of the ticket
  flow" → "...of the {flow} flow", under `pickle serve`). Example:
  `docs/user-manual/configuration.adoc:60` — "The shipped ticket-flow skill
  (\`.agents/skills/ticket-flow/\`)" → "The shipped {flow} skill (\`.agents/skills/ticket-flow/\`)".
- Every other hit in `docs/user-manual/cli-reference.adoc` and every hit in
  `docs/user-manual/concepts/project-structure.adoc` is a literal path or an ASCII tree
  diagram entry (confirmed by inspection) — **do not touch anything else in either file**.

#### Task 8 — tests

- `internal/config/config_test.go`: `FlowName()` defaults to `"brine"` when unset; an explicit
  `flow = "brine"` round-trips through `Save`/`Load`; `Validate()` rejects `flow = "v-model"`
  (or any other non-`"brine"` value) with an error naming the one legal value.
- `internal/cli/`: a new `flow_test.go` (or additions to `cli_test.go`) covering `pickle flow
  show` and `pickle flow list` against a fixture project (both print `brine`), and an unknown
  `pickle flow bogus` exiting `exitUsage`.
- `internal/install/install_test.go` / `agents_test.go`: the existing test(s) that compare
  `MarkerBlock(cfg)` against `testdata/markerblock.golden` pass unchanged (i.e. the golden file
  update in Task 4 is verified by the existing comparison, not a new test).
- `internal/doctor/doctor_test.go`: assert the new `flow: brine` line appears in `Passed` for a
  config with no explicit `flow` key, and again for one with `flow = "brine"` set explicitly.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

Plus a manual smoke test in a throwaway install — never against this repo's own path, per the
self-modify policy:

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
mkdir proj && cd proj && git init -q
../pk install --project demo
../pk flow show      # → brine
../pk flow list      # → brine
../pk doctor          # exit 0; Passed includes a "flow: brine" line
```

### Docs update (mandatory when user-facing)

`docs/attributes.adoc` (new `:flow:` attribute) and the five files listed in Task 7; no new
page — this is a rename, not a new concept, so no xref/nav changes. `docs/user-manual/
cli-reference.adoc` gains no entry for `pickle flow` in *this* ticket unless reviewers judge the
omission itself a gap; if so, disposition it per rules §5 rather than expanding this ticket —
T-066 (close the CLI-surface documentation gaps) is the natural home for a follow-up.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint`/`just docs-check` clean.
2. Docs updated per Task 7 and registered (attribute + the five naming-prose sites).
3. Write the summary: files touched, and explicitly call out the Task 8 golden-file diff and
   the self-hosted `AGENTS.md`/`pickle.toml` hand-edits (Task 6) so review can verify them
   against a fresh `pickle doctor` run.
4. Suggested commit message:

   ```
   feat(cli): introduce brine as the flow's name (T-073)

   Adds an optional `flow` key to pickle.toml (default "brine") plus `pickle
   flow show|list`, and renames the flow in prose across the skill payload,
   this repo's own self-hosted files, and the docs — no path on disk moves.
   ```

5. Commit locally on the ticket branch. Publish only per the project's commit policy (do not
   push or open a merge request without user approval). Present the commit message; only after
   approval finalize, push, and open the MR — merging is always the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — TO DO → READY: plan complete: config key, CLI subcommand, doctor line, and the naming-only rename traced file-by-file across install.go, the skill payload, this repo's own self-hosted files, and docs
- 2026-08-07 — patched by T-069's review impact sweep (step 8): T-069 removed every
  TOML-rendering `%q` from `Config.Render()` in favour of a `tomlQuote` helper and added an
  invalid-UTF-8 gate to `Validate()`. Task 1 said to emit `flow = %q`, which would have
  reintroduced the defect T-069 closed; it now says `tomlQuote(c.Flow)` and adds `flow` to the
  UTF-8 gate. No other assumption in the plan is affected — nothing else it touches moved
- 2026-08-07 — applicability gate on pickup: 1 blocking (Task 1 assumed T-069's `tomlQuote`/
  UTF-8 gate were on `main`; T-069 was DONE but unmerged — publish-gated) — resolved when the
  user merged `feat/T-069-config-writers-safe` → main (152fea8, #17); T-069's ticket and the
  board updated to reflect the merge. 1 non-blocking (Task 7 missed a naming-prose hit at
  `cli-reference.adoc:573` under `pickle serve`) — fixed inline, added to Task 7's replacement
  list. No other assumption changed
