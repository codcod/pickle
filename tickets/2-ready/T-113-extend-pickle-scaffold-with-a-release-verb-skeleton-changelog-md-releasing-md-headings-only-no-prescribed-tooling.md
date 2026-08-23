---
id: T-113
title: extend 'pickle scaffold' with a release verb: skeleton CHANGELOG.md + RELEASING.md (headings only, no prescribed tooling)
project: pickle
depends-on: []
spawned-by: [T-110, T-111]
impact: low-medium
complexity: low
cost: S
---

# T-113 — extend 'pickle scaffold' with a release verb: skeleton CHANGELOG.md + RELEASING.md (headings only, no prescribed tooling)

## Outcome

Running `pickle scaffold release` in a target repo writes two skeleton files — an empty-
`[Unreleased]`-section `CHANGELOG.md` (Keep a Changelog shape) and a `RELEASING.md` with section
headings only (Versioning / Build / Publish / Verify, each a `TODO` placeholder, no prescribed
commands or tools, plus a header explaining the `[agent]`/`[human]` step-marking convention an
agent must honour) — additive-only like `pickle scaffold docs` (existing files are left alone
without `--force`). This is exactly the scaffold `docs/user-manual/concepts/releasing.adoc`
(T-111) describes an agent offering when either file is missing, made real, and it is the
deliberately narrowed remainder of a larger scaffold-release design that was proposed and then
challenged down to this.

## Description

**Origin and what this deliberately is not.** A chat design discussion proposed extending
`pickle scaffold` (T-110's verb group: today only `scaffold docs`) with a `scaffold release`
verb that would also write a language-forked (`--lang go|rust`) GitHub Actions release workflow,
additive justfile recipes with language-specific bodies, and a `RELEASING.md`/workflow pair whose
content depended on shelling out to `goreleaser init` for Go and on picking a Rust release
tool (`cargo-dist` vs. plain `cargo publish`) pickle has no consensus reference for. That design
was challenged and rejected before filing, on grounds consistent with T-110's own filing
rationale and this ticket's siblings:

- **False confidence asymmetry.** Treating a Go workflow modeled on this repo's own
  goreleaser+GitHub+Homebrew-tap pipeline as "high-confidence" repeats the exact mistake T-110
  was filed to correct ("modeled on the pickle repo" baking in choices that don't follow from
  "adopted brine") — a pipeline validated in one repo by one author is not a Go convention, and
  Rust's harder-to-guess pipeline just makes the same one-sample bias visible instead of hidden.
- **Fails the competence-boundary test.** A release workflow, a justfile recipe body, and a
  choice of Rust distribution tool require no ticket-domain knowledge (no ticket ids, no board
  state, no prefixes) — the same test that ruled a `release`-executing trigger out of scope for
  T-111 rules this content out of scope for a scaffold command too; renaming the feature from
  "trigger" to "scaffold" doesn't move it across that line.
- **Duplicates better-maintained tools.** `cargo generate`, `cargo-dist init`, and
  `goreleaser init` itself already solve "scaffold a release pipeline for language X," tracked by
  people who follow that ecosystem's churn — the same objection already used in this repo against
  a `pickle changelog cut` verb (`git-cliff`/`changie` own that space), reapplied here to a larger
  surface.
- **Demonstrated drift risk.** T-110's own first review (finding F1) caught its *one* embedded
  workflow template pinning `actions/checkout@v4` against this repo's own `@v7`, on the very first
  review of the very first template shipped. Two more per-language embedded workflow/tooling
  templates multiply that exact, already-observed risk rather than avoiding it.
- **Larger, unaddressed security surface.** A release workflow needs `contents: write` and
  typically a publish token (crates.io, a Homebrew tap PAT, signing keys) — a materially higher
  stakes template to embed and stamp into every scaffolding project than the docs-attach step
  T-110 shipped.
- **Silent re-coupling to `install`.** Branching scaffolded *content* on whether `pickle.toml`
  exists (to decide whether to mention `pickle changelog check`) would make `scaffold`'s output
  depend on `install` having run — precisely the "mission mismatch" T-110 was filed to keep
  `scaffold` free of (decision 8: no `pickle.toml`/doctor/board-audit integration).

**What survives, and is this ticket's actual scope.** Two files only, neither language-forked,
neither carrying a tool opinion:

- **`CHANGELOG.md`** — a Keep a Changelog skeleton with a single empty `[Unreleased]` heading.
  Generic, already implied as a fallback offer by T-111's Description, and exactly what
  `pickle changelog check` needs to find *a* file to check against.
- **`RELEASING.md`** — section headings only (a small fixed set: Versioning, Build, Publish,
  Verify — exact wording pinned at refinement), each body a single `TODO` placeholder comment,
  no commands, no tool names, no language branch. This is structure, not doctrine — the same
  register as `scaffold docs`' own one-placeholder-chapter skeleton
  (`user-manual/introduction.adoc`), not a filled-in procedure.

No `--lang` flag, no GitHub Actions template, no justfile recipes, no shell-out to any
release tool. If those are ever wanted, T-111's fallback-ladder page must keep reading correctly
whether or not they exist (T-111 decision 2: the page documents a pattern, never promises a
verb) — so nothing here is a prerequisite for anything else, and nothing here forecloses a
future, separately-challenged ticket for the richer version.

**Soft couplings** (no hard `depends-on:`):

- **T-110** (done) — the `scaffold` verb group and its `internal/scaffold` package this ticket
  extends; `Docs`'s per-file template-write loop and its additive-justfile-recipe helper are the
  precedent for how `Release` should be built (shared helpers, not a parallel implementation).
- **T-111** (`2-ready/`) — the manual page (`docs/user-manual/concepts/releasing.adoc`) whose
  fallback ladder describes exactly the two files this ticket writes; that page's prose must
  still read correctly regardless of whether this ticket ships (decision 2), and once this
  ships the page could gain one added sentence naming the command — a documentation follow-up,
  not a code dependency, and out of scope for this ticket to add.
- **T-093/T-094/T-095/T-097** (done) — `pickle changelog check`, the command the scaffolded
  `CHANGELOG.md`'s `[Unreleased]` section exists for; this ticket must not alter that command's
  contract.

## Implementation Plan

### 0. Feature branch (mandatory)

Root-path child (`pickle`, `path = "."`), so the branch is cut in this repo:

```
git checkout main
git checkout -b feat/T-113-scaffold-release-skeletons
```

Commit locally as work proceeds. Tidy WIP commits into atomic ones before presenting (root-path
default — keep the tidied history rather than squashing). Never push or open an MR without
explicit user approval. `layout = "in-tree"` applies, so before any eventual push verify the
remote base is not behind: `git fetch origin main && git diff --name-only origin/main...HEAD |
grep '^tickets/'` must print nothing.

### Prerequisite gate (hard)

None blocking. Both `spawned-by:` tickets are lineage only, never gates: **T-110** is `6-done/`
and merged (PR #56, `323add9`), and **T-113 does not depend on T-111** — T-111 is still
`2-ready/`, and decision 7 below is what keeps this ticket shippable in either order. Clean tree
on `main`.

### Confirmed design decisions (do not deviate without asking)

1. **The scaffolded `RELEASING.md` never mentions `pickle changelog check`, or any other
   command.** Verified at refinement: `internal/cli/project.go:43` `loadConfig()` resolves
   `config.Find(wd)` and errors when no `pickle.toml` exists, so `pickle changelog check` *fails*
   in a repo without brine — and a brine-free repo is an explicitly supported audience for
   `scaffold` (T-110 decision 1; `TestScaffoldDocsIsUnrelatedToBrine` in
   `internal/cli/cli_test.go:968` asserts it). An unconditional mention would therefore point a
   supported user at a command that errors for them, and a conditional mention would be the
   `pickle.toml`-detection content fork this ticket's Description explicitly rejected. Both
   options are wrong, so the file names no command at all; the `CHANGELOG.md`↔`changelog check`
   connection is documented in the manual (T-111), which is the right home for it.
2. **`RELEASING.md` ships four headings and nothing else:** `## Versioning`, `## Build`,
   `## Publish`, `## Verify`, each body a single HTML-comment `TODO` naming what belongs there
   (version scheme / where the version number lives / tag convention; how an artifact is built;
   how and where it is published; how to confirm it is consumable). No commands, no tool names,
   no language branch — structure, not doctrine, the same register as `scaffold docs`' own
   single placeholder chapter.
3. **The file header states an `[agent]` / `[human]` step-marking convention.** A short comment
   block instructs the author to mark each step they write with who may run it, and states that
   an agent must stop and hand back at every `[human]` step. This is the one piece of *safety*
   structure worth shipping (it is what T-111's page asks a useful `RELEASING.md` to carry), and
   it is tool-neutral — it prescribes no procedure, only how to annotate whatever procedure the
   author writes.
4. **`CHANGELOG.md` is a Keep a Changelog skeleton with one empty `## [Unreleased]` section and
   no link-reference footer.** The footer's compare links need the remote URL, forge flavour and
   tag-prefix convention — the forge inference already rejected in T-111's reasoning against a
   `changelog cut` verb. Body of `[Unreleased]` is a single comment naming the conventional
   subsection headings (Added/Changed/Fixed/…) as options, not pre-writing them.
5. **New templates live at `scaffold/release-template/{CHANGELOG.md,RELEASING.md}` and need no
   `assets.go` change.** Verified at refinement: `assets.go`'s directive is already
   `//go:embed all:skill all:agents all:scaffold`, which embeds the whole `scaffold/` tree —
   unlike T-110, which had to add the embed root. Update only the doc comment above the
   directive to name the new subtree; do **not** add a fourth `all:` entry.
6. **Shared helper, not a parallel implementation.** Extract the per-file write loop currently
   inline in `scaffold.Docs` (`internal/scaffold/scaffold.go:69-110`) into an unexported
   `writeTemplates(payload fs.FS, root string, files []templateFile, name string, opts Options,
   res *Result) error`, and have both `Docs` and the new `Release` call it. Behaviour-preserving
   for `Docs` — every existing `TestDocs*` case must pass untouched. Do **not** touch
   `scaffoldJustfile`, `hasRecipe` or `scaffoldSnowballConfig`: `Release` uses none of them.
7. **No cross-reference to T-111's manual page.** T-111 (`docs/user-manual/concepts/releasing.adoc`,
   anchor `[#releasing]`) is still `2-ready/` and may ship after this ticket; an `<<releasing>>`
   xref would be a dangling target — and since T-067 merged to `main` on 2026-08-22 (PR #60,
   patched here by its review's impact sweep), `just docs-check` and `go test ./...` now **fail
   the build** on one rather than rendering past it. So this is no longer a judgement call: do
   not write the xref until T-111's anchor exists. This ticket's docs cite only `<<cmd-scaffold-docs>>` and its own new anchor.
   Adding the connecting sentence once both exist is a follow-up, not this ticket's work.
8. **`Release` writes exactly two files and nothing else.** No justfile recipes, no GitHub
   Actions workflow, no shell-out to any release tool, no `--lang` flag, no `pickle.toml` read,
   no `doctor`/`board audit` integration (T-110 decision 8 carried forward verbatim).
9. **Flags mirror `scaffold docs` exactly:** `--project-name` (default `filepath.Base(root)`),
   `--force`, `--dry-run`, reusing the existing `scaffold.Options`/`scaffold.Result` types
   unchanged. `__PROJECT_NAME__` is substituted in `RELEASING.md`'s title only; `CHANGELOG.md`
   contains no token (Keep a Changelog's `# Changelog` heading is conventionally bare), which
   makes its substitution a harmless no-op.

### Tasks

#### Task 1 — the two templates

> **Both blocks below are indented by two spaces on purpose — strip that indent when writing the
> real files.** The gate checker's section walk (`internal/ticket/ticket.go`, `SectionBody` /
> `SubsectionBody`) is deliberately blind to fenced code blocks, a limitation its own doc
> comments record as accepted (T-083/T-081/T-105). An unindented `## [Unreleased]` or
> `## Versioning` line inside these fences reads as a real top-level heading and truncates this
> `## Implementation Plan` section, hiding the Acceptance test / Docs update / Finish headings
> from the READY gate — `pickle ticket move T-113 ready` refuses with three "unmet gate
> requirement" errors. Do not "tidy" the indent away.

Create `scaffold/release-template/CHANGELOG.md` (content, de-indented):

```markdown
  # Changelog

  All notable changes to this project are documented in this file.

  The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
  and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

  ## [Unreleased]

  <!-- Add entries here as they land. Conventional headings: Added, Changed,
       Deprecated, Removed, Fixed, Security — use whichever apply.
       When you cut a release, retitle this section to "## [X.Y.Z] - YYYY-MM-DD"
       and open a fresh empty [Unreleased] above it. -->
```

Create `scaffold/release-template/RELEASING.md` (content, de-indented):

```markdown
  # Releasing __PROJECT_NAME__

  <!-- Scaffolded as a skeleton: headings only. Fill each section in with this
       project's actual procedure — the commands, their order, and the
       checkpoints between them. None of it is prescribed here. -->

  <!-- Mark every step you write with who runs it:
         [agent]  safe for a coding agent to run unprompted
         [human]  needs a person — anything that publishes, signs, or is otherwise
                  irreversible
       An agent following this file stops at every [human] step and hands back. -->

  ## Versioning

  <!-- TODO: the version scheme; where the version number lives (a manifest file, a
       constant, or nowhere at all); and the tag convention, e.g. v1.2.3 or 1.2.3. -->

  ## Build

  <!-- TODO: how a release artifact is produced from a clean checkout. -->

  ## Publish

  <!-- TODO: how the artifact is published, and where to. Mark these steps [human]
       unless there is a specific reason not to. -->

  ## Verify

  <!-- TODO: how to confirm the release is really consumable — install it fresh,
       check the published artifact, confirm the tag points where you think. -->
```

In `assets.go`, extend the doc comment above `//go:embed` with a `scaffold/release-template/`
bullet (the two skeletons `pickle scaffold release` writes). **Leave the directive line itself
unchanged** (decision 5).

#### Task 2 — `internal/scaffold`: extract the helper, add `Release`

In `internal/scaffold/scaffold.go`:

- Lift the `for _, tf := range templateFiles { … }` body out of `Docs` into
  `writeTemplates(payload, root, files, name, opts, res)` (decision 6), returning `error`;
  `Docs` now calls it with `templateFiles`.
- Add `var releaseTemplateFiles = []templateFile{`
  `{"scaffold/release-template/CHANGELOG.md", "CHANGELOG.md"},`
  `{"scaffold/release-template/RELEASING.md", "RELEASING.md"}}`.
- Add `func Release(payload fs.FS, root string, opts Options) (Result, error)`: default
  `ProjectName` to `filepath.Base(root)` exactly as `Docs` does, call `writeTemplates` with
  `releaseTemplateFiles`, return. Nothing else — no justfile, no snowball (decision 8).
- Update the package doc comment: it currently says the package "implements
  `pickle scaffold docs`"; make it cover both verbs and state that `release` writes only two
  skeleton files, prescribing no tooling.

#### Task 3 — `internal/cli`: dispatch, flags, usage

- `internal/cli/scaffold.go`: add `case "release": return runScaffoldRelease(args[1:])` to
  `runScaffold`; update its two usage strings (`"pickle scaffold: expected docs"` → `expected
  docs or release`; `(want docs)` → `(want docs or release)`). Add `runScaffoldRelease`, a
  near-copy of `runScaffoldDocs` (same three flags, same `Created`/`Skipped`/`Notes` print loop)
  calling `scaffold.Release`.
- `internal/cli/cli.go` `usage()`: add a `scaffold release [--project-name <name>] [--force]
  [--dry-run]` entry under the existing `Other scaffolding (unrelated to brine):` group
  (`:144-149`), in the same two-column style — one line naming the two files and that it
  prescribes no release tooling.

#### Task 4 — tests

`internal/scaffold/scaffold_test.go` (reuse the existing `payload()`/`mustRead`/`contains`
helpers; name the new cases `TestRelease*` so they sort beside the `TestDocs*` block):

- creates both files; `RELEASING.md` contains `# Releasing demo` (token substituted) and
  `CHANGELOG.md` contains `## [Unreleased]`;
- defaults the project name to the root's basename when `--project-name` is empty;
- re-run without `--force` skips both and reports them in `Skipped`; with `--force` overwrites;
- `--dry-run` writes nothing (`os.Stat` still fails for both afterwards);
- **writes no other file** — assert `justfile`, `snowball.yaml` and `.github/` do *not* exist
  after a run (this is the regression guard for decision 8);
- **names no command** — assert the generated `RELEASING.md` does not contain the substring
  `pickle ` (the mechanical guard for decision 1, which is the decision most likely to be
  "helpfully" undone by a later change).

`internal/cli/cli_test.go`:

- add `{"scaffold release bad flag", []string{"scaffold", "release", "--bogus"}, exitUsage}` to
  the table at `:127-129`;
- add `TestScaffoldReleaseIsUnrelatedToBrine`, mirroring `TestScaffoldDocsIsUnrelatedToBrine`
  (`:964`): run in a bare `t.TempDir()` with no `pickle.toml`, assert `exitOK`, both files
  exist, and nothing brine-owned (`pickle.toml`, `tickets`, `AGENTS.md`, `.agents`) was created.

#### Task 5 — docs

`docs/user-manual/cli-reference.adoc`:

- Overview table (`:78-81`): add a `pickle scaffold release [--project-name …] [--force]
  [--dry-run]` row directly after the existing `scaffold docs` row.
- New `[#cmd-scaffold-release]` section immediately after the `[#cmd-scaffold-docs]` section
  ends (before `[#cmd-version]`, `:1424`), mirroring its structure: synopsis block, the
  "unrelated to {flow}" paragraph, a two-column Path/What table for the two files, the three
  flags, and — explicitly — a short paragraph stating what it deliberately does **not** write
  (no workflow, no justfile recipes, no tool config, no language detection) and that the
  skeletons name no commands, so the reader understands the omission is the design rather than
  an unfinished feature.

`CHANGELOG.md`: add an `[Unreleased]` → `### Added` bullet for T-113, matching the T-110 entry's
voice (`:11-16`). This repo's own per-ticket convention — and T-110's review raised its absence
as finding F3, so it is a known trap here.

### Acceptance test

From the repo root on the feature branch:

```sh
just build && just test && just lint && just docs-check   # all green

# Per AGENTS.md: throwaway dir, binary copied in and renamed pickle-test.
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"

./pickle-test scaffold release --project-name demo
test -f CHANGELOG.md && test -f RELEASING.md
grep -q '^# Releasing demo$' RELEASING.md
grep -q '^## \[Unreleased\]$' CHANGELOG.md
grep -q '## Versioning' RELEASING.md && grep -q '## Verify' RELEASING.md
! grep -q 'pickle ' RELEASING.md          # decision 1: names no command
! test -e justfile && ! test -e snowball.yaml && ! test -d .github   # decision 8
! test -e pickle.toml && ! test -d tickets                           # brine untouched

# re-run without --force: both skipped, nothing overwritten, exit 0
./pickle-test scaffold release --project-name demo

# --dry-run in a clean dir writes nothing
D2=$(mktemp -d) && cp "$D/pickle-test" "$D2/" && cd "$D2"
./pickle-test scaffold release --dry-run
! test -e CHANGELOG.md && ! test -e RELEASING.md

# scaffold docs still behaves identically after the Task 2 refactor
D3=$(mktemp -d) && cp "$D/pickle-test" "$D3/" && cd "$D3"
./pickle-test scaffold docs --project-name demo
test -f docs/user-manual.adoc && grep -q '= demo User Manual' docs/user-manual.adoc
```

The `scaffold docs` re-check at the end is the behaviour-preservation proof for decision 6's
refactor, alongside the untouched `TestDocs*` suite.

### Docs update (mandatory when user-facing)

Task 5 above: `cli-reference.adoc` (Overview row + a new `[#cmd-scaffold-release]` section),
`internal/cli/cli.go`'s `usage()` text (Task 3), the `assets.go` doc comment (Task 1), and a
`CHANGELOG.md` `[Unreleased]` entry. No `skill/` payload change — this is a pickle CLI feature,
not a flow rule, so the foreign-workspace test does not come into play.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated and registered (Task 5); `CHANGELOG.md` entry present.
3. Write a summary: files touched, decisions honoured (call out 1, 5, 6 and 7 explicitly — the
   ones a reasonable implementer might "improve" by undoing), anything deferred.
4. Suggested commit message:

   ```
   feat(cli): add `scaffold release` for CHANGELOG/RELEASING skeletons (T-113)

   New `pickle scaffold release` writes a Keep a Changelog CHANGELOG.md and a
   headings-only RELEASING.md, prescribing no release tooling: no workflow, no
   justfile recipes, no language detection, and no command named in either file.
   Extracts the template-write loop shared with `scaffold docs`.
   ```

5. Root-path child: tidy WIP commits into atomic ones before presenting.
6. Commit locally; do not push or open an MR without explicit user approval.
   `pickle ticket move T-113 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: chat: a chat design discussion proposed extending
  `pickle scaffold` with a language-forked (`--lang go|rust`) `scaffold release` verb writing a
  GitHub Actions workflow, justfile recipes, and a filled-in `RELEASING.md`; challenged on six
  grounds (false Go/Rust confidence asymmetry, failing the competence-boundary test already
  applied to the rejected release-executing trigger, duplicating cargo-dist/goreleaser-init,
  T-110's own first-review drift finding recurring, an unaddressed publish-secrets security
  surface, and silent re-coupling to `install` via pickle.toml-detection) and narrowed to the
  two skeleton files T-111's fallback ladder already describes; spawned-by T-110 (the verb group
  and package this extends) and T-111 (the manual page this fulfills).
- 2026-08-22 — refined: pinned nine decisions, three of them resolved by evidence found while
  refining rather than by preference. (a) The scaffolded `RELEASING.md` names **no** command:
  `loadConfig` (`internal/cli/project.go:43`) errors without a `pickle.toml`, so
  `pickle changelog check` fails in exactly the brine-free repos `scaffold` is documented to
  support — making an unconditional mention wrong, and a conditional one the rejected
  pickle.toml-detection fork; a test asserting the file contains no `pickle ` substring guards
  it. (b) No `assets.go` embed change is needed — the directive is already
  `all:scaffold`, covering the whole subtree (T-110 had to add its embed root; this ticket must
  not "fix" the directive). (c) No `<<releasing>>` xref to T-111, which is still `2-ready/` and
  may ship second — the two tickets are deliberately order-independent. Also confirmed T-042
  (helper consolidation) does not overlap: its sites are marker-span/status-heading, not
  scaffold. Task 1's template blocks are indented two spaces to survive the gate checker's
  fence-blind section walk (a documented, accepted limitation — T-083/T-081/T-105); the plan
  says so inline so the indent is not tidied away.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — patched by **T-067's review impact sweep**: T-067 is now `6-done/` (branch `feat/T-067-docs-xref-check` not yet merged), so this ticket's "docs-check cannot catch a dangling xref" assumption holds only until that branch lands. Wording updated to say which state applies and how to tell.
- 2026-08-22 — patched again by **T-067's impact sweep**, on merge: T-067 landed on `main` (PR #60, 2e29b50), so the "docs-check cannot catch a dangling xref" caveat is now simply false and has been removed rather than re-qualified. The gate is live for this ticket's cross-references.
- 2026-08-23 — patched by **T-111's review impact sweep**: T-111 is now `6-done/` but its branch
  is **not yet merged**, so **decision 7 still stands unchanged** — the `[#releasing]` anchor
  exists only on `feat/T-111-releasing-convention-docs`, not on `main`, and an `<<releasing>>`
  xref written before that merge would still dangle and fail `docs-check`. Re-read decision 7's
  premise ("T-111 is still `2-ready/`") as "not yet merged"; the instruction it yields is
  identical. Once T-111 merges, the xref becomes writable and the connecting sentence its
  coupling note describes becomes an actionable follow-up. Scope and grade unchanged.
- 2026-08-23 — patched again by **T-111's impact sweep**, on merge: T-111 landed on `main`
  (PR #63, 4614178), so the `[#releasing]` anchor now exists on the base branch and **decision 7's
  premise is discharged** — an `<<releasing>>` xref would resolve, not dangle. The decision's
  instruction is therefore no longer forced by the build: whether to add the connecting sentence
  is now an open choice for this ticket's pickup gate rather than a prohibition. Nothing else in
  the plan changes; the two files this ticket scaffolds are unaffected. Scope and grade unchanged.
