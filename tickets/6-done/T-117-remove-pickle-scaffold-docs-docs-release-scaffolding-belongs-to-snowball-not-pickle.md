---
id: T-117
title: remove pickle scaffold docs — docs/release scaffolding belongs to snowball, not pickle
project: pickle
depends-on: []
spawned-by: [T-116]
impact: medium
complexity: low
cost: S
---

# T-117 — remove pickle scaffold docs — docs/release scaffolding belongs to snowball, not pickle

## Outcome

`pickle scaffold docs` is gone — the subcommand, its embedded `scaffold/docs-template/`
payload, the docs half of `internal/scaffold`, and every mention of it in `pickle help`, the
CLI reference and the manual. `pickle scaffold release` is unaffected and becomes the verb
group's only subcommand. Scaffolding an AsciiDoc docs skeleton, its `snowball.yaml`, justfile
fragments and a release-attach GitHub Action for a project's user manual becomes `snowball`'s
job (tracked as `SNOW-003` in the `unity` workspace), not pickle's.

## Description

T-110 added `pickle scaffold docs`, an opt-in, standalone command unrelated to brine that lays
down a minimal AsciiDoc docs skeleton, shells out to `snowball init` for `snowball.yaml`,
appends justfile `docs-check`/`docs-build` recipes, and drops a GitHub Actions release-attach
workflow. T-110's own Description already named the risk at filing time: *"embedding a copy of
this repo's own docs pipeline as a template means every future change to it... needs a matching
update to the embedded copy or the template silently rots"* — accepted then as a manageable
trade-off for the convenience of a one-shot scaffold.

T-116 (this repo, `2-ready/`) shows that trade-off has already soured, and for a more basic
reason than "pickle's copy goes stale over time": pickle's shipped scaffold has to track
**`snowball`'s own internal defaults and conventions**, which pickle has no authority over and
keeps getting subtly wrong —

- `snowball init`'s default `snowball.yaml` references a second book,
  `docs/developer-handbook.adoc`, that pickle's scaffold never creates. It isn't an arbitrary
  placeholder: it mirrors a real two-book AsciiDoc setup this same workspace already runs
  (`rick`, see `RICK-185`) — snowball's own default is modeled on real usage pickle has no
  visibility into.
- asciidoctor-pdf's PDF theme loader requires a theme file's name to literally end in
  `-theme.yml` (it derives a theme *name* by stripping that suffix, then reloads
  `<name>-theme.yml`) — an asciidoctor-pdf/snowball implementation detail with no reason for
  pickle to know it, discovered only by hitting the resulting silent-fallback-plus-nonzero-exit
  failure firsthand (T-116's refinement).
- The scaffolded chapter's AsciiDoc heading level has to match the `leveloffset` convention
  `snowball`-rendered book masters expect — again a fact about how snowball's toolchain
  (asciidoctor) resolves section nesting, not about pickle.

Every one of these is a fact pickle has to keep re-deriving about a tool it doesn't own, from
outside, by trial and error — exactly the shape of drift T-110 anticipated. `snowball` is the
actual domain owner of `snowball.yaml`'s schema, its own `init` defaults, and the PDF theme
convention; only it can keep a scaffolded doc skeleton and the config it targets in lock-step,
because it controls both sides. Moving the capability there turns three foreign-tool facts
pickle had to chase into one project's own facts about itself.

**Scope of removal:** everything T-110 added, minus the `scaffold` command surface itself
(shared with the shipped `release` subcommand, see below). Verified against the tree at
refinement:

- `scaffold/docs-template/` — the four-file embedded payload tree (`attributes.adoc`,
  `user-manual.adoc`, `user-manual/introduction.adoc`, `workflows/docs-release.yml`).
- The docs half of `internal/scaffold/scaffold.go` — `Docs`, `templateFiles`,
  `scaffoldJustfile`, `hasRecipe`, `justfileRecipe`, `justfileRecipes`,
  `scaffoldSnowballConfig`, plus the docs clauses of the package doc comment.
- The ten `TestDocs*` functions in `internal/scaffold/scaffold_test.go`.
- `runScaffoldDocs` and its `"docs"` case in `internal/cli/scaffold.go`.
- Three `internal/cli/cli_test.go` items: the `"scaffold docs bad flag"` table case,
  `TestScaffoldDocsIsUnrelatedToBrine`, and `TestScaffoldDocsDryRunWritesNothing`.
- The `usage()` block in `internal/cli/cli.go` documenting `scaffold docs [...]`.
- The `[#cmd-scaffold-docs]` section and the Overview row in
  `docs/user-manual/cli-reference.adoc`.
- The docs-template bullet in `assets.go`'s payload comment.

**Explicitly *not* removed** (a filing-time assumption that refinement overturned): the
`internal/scaffold` package survives. T-113 shipped while this ticket sat in TO DO, and
`Release`, `Options`, `Result`, `writeTemplates` and `projectNameToken` all serve
`pickle scaffold release` now. For the same reason `assets.go`'s
`//go:embed all:skill all:agents all:scaffold` line is unchanged — `scaffold/release-template/`
lives under that root. `runScaffold`, and the top-level `case "scaffold":` in
`internal/cli/cli.go`, also stay.

**Resolved at filing (user decisions):**

1. **T-116** (fixed bugs in exactly the feature this ticket deletes) — dropped
   (`7-dropped/`, superseded by this ticket); its findings (the `-theme.yml` naming
   requirement, the phantom-book origin, the heading/`leveloffset` mismatch, and the missing
   check/build regression test) were transplanted into the companion ticket below before it
   was dropped, so none of that investigation is lost.
2. **T-113** (a different, unrelated `release` subcommand under `pickle scaffold`: skeleton
   `CHANGELOG.md`/`RELEASING.md`, no AsciiDoc/snowball involved) **stays** — the `scaffold`
   command surface is not going away, only the `docs` subcommand under it. **Status correction
   at refinement:** T-113 is no longer pending — it is in `6-done/`, and
   `pickle scaffold release` is live. So this is not "leave room for T-113" but "do not break
   what T-113 already shipped"; after this ticket, `release` is the verb group's sole
   subcommand.

**Not a deprecation.** pickle is pre-1.0 (`0.11.0`), and `CHANGELOG.md`'s own preamble states
that breaking changes may land in a minor release below `1.0.0`. `scaffold docs` is therefore
deleted outright, with no deprecation shim and no `docs`-specific error message pointing
elsewhere: after this ticket `pickle scaffold docs` falls through to the generic
`unknown subcommand "docs" (want release)` path, exactly as any other typo would. Nor is the
removal gated on `SNOW-003` landing — the feature is actively wrong today (T-116), so the
capability gap is preferable to shipping the wrong thing while snowball catches up.

Soft coupling: spawned by T-116 (this repo) — filed once T-116's refinement made the
cost-of-tracking-a-foreign-tool's-internals concrete enough to question the whole feature, not
just its bugs. The companion ticket describing the equivalent capability, ported and corrected
(including T-116's findings), is `SNOW-003` in the `unity` workspace's own board, against
`project: snowball` — out of scope for this repo's ticket ids (a separate installation, not a
registered child here).

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is a root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-117-remove-scaffold-docs
```

WIP commits are encouraged; tidy them into atomic commits before presenting (root-path child —
keep the tidied history rather than squashing). Do not push or open an MR without explicit user
approval. Under `layout = "in-tree"`, before pushing verify the remote base is not behind:
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing.

### Prerequisite gate (hard)

None. `depends-on:` is empty, and every prerequisite this ticket once had is already settled:
T-116 is in `7-dropped/` and T-113 is in `6-done/`. `SNOW-003` (the `unity` workspace's
replacement) is deliberately **not** a gate — see the Description.

Start from a clean tree on an up-to-date `main`.

### Confirmed design decisions (do not deviate without asking)

1. **The `internal/scaffold` package survives.** Only the docs half is deleted. `Release`,
   `Options`, `Result`, `writeTemplates` and `projectNameToken` are load-bearing for
   `pickle scaffold release` (T-113, shipped). Deleting the package — which this ticket's own
   Description asked for at filing time, before T-113 landed — would break a working command.
2. **`assets.go`'s `//go:embed all:skill all:agents all:scaffold` line is unchanged.**
   `scaffold/release-template/` lives under that root. Only the *prose bullet* describing
   `scaffold/docs-template/` is removed from the comment above it.
3. **The `scaffold` verb group stays a verb group,** even at one subcommand. `runScaffold` and
   the top-level `case "scaffold":` in `internal/cli/cli.go` are untouched; only the two
   operator-facing strings change to name `release` alone (`expected release`,
   `unknown subcommand %q (want release)`).
4. **No deprecation, no `docs`-specific error, no trace left.** `pickle scaffold docs` falls
   through to the generic unknown-subcommand path. Do not add a migration hint, a
   pointer to `snowball`, or a commented-out remnant. Grepping the tree for `scaffold docs` or
   `docs-template` after this ticket must return hits only in `CHANGELOG.md` (its own history)
   and under `tickets/` — nothing in `internal/`, `docs/`, `scaffold/`, `skill/` or `assets.go`.
5. **The CHANGELOG entry goes under `### Breaking`,** in `## [Unreleased]`. This heading has not
   been used in this file before — that is intended, not an oversight; do not "correct" it to
   Keep a Changelog's `### Removed`.
6. **Released history is immutable; unreleased prose is not.** The `## [0.11.0]` entry
   announcing `pickle scaffold docs` stays exactly as written — it is a true record of what
   0.11.0 shipped. The *unreleased* T-113 entry's phrase "mirroring `pickle scaffold docs`"
   must be reworded, since it will publish alongside the removal and its antecedent will not
   exist (decision 4).
7. **`TestScaffoldDocsDryRunWritesNothing` is deleted without a `release` replacement.**
   `internal/scaffold`'s own `TestReleaseDryRunWritesNothing` still covers the `DryRun` option;
   what lapses is only the CLI-level flag-parse path for `--dry-run`. Accepted deliberately —
   do not add a substitute test as a bonus.
8. **`docs/user-manual/cli-reference.adoc`'s release section must be rewritten to stand
   alone.** It currently opens "unrelated to {flow} in exactly the same sense as `pickle
   scaffold docs` above", whose antecedent this ticket deletes. A dangling comparison is a
   docs defect, not a cosmetic one.

### Tasks

#### Task 1 — delete the embedded payload tree

`git rm -r scaffold/docs-template/` (four files: `attributes.adoc`, `user-manual.adoc`,
`user-manual/introduction.adoc`, `workflows/docs-release.yml`). Leave
`scaffold/release-template/` untouched.

In `assets.go`, delete the `scaffold/docs-template/ — …` bullet from the `payloadFS` doc
comment and fix the sentence beneath it that reads "Embedding all four in the binary lets
pickle scaffold either surface" — there are three payload roots now, and only one scaffold
surface. The `//go:embed` line itself does not change (decision 2).

#### Task 2 — strip the docs half of `internal/scaffold`

In `internal/scaffold/scaffold.go` remove: `Docs`, `templateFiles`, `justfileRecipe`,
`justfileRecipes`, `scaffoldJustfile`, `hasRecipe`, `scaffoldSnowballConfig`. Rewrite the
package doc comment so it describes the `release` verb only, keeping the "unrelated to brine"
and "never read back by `pickle doctor`/`pickle board audit`" points and dropping the T-110
citation. Drop the now-unused imports (`os/exec` and `strings` go; verify with `go build`
rather than by eye — `strings` may still be reachable). Check `Options.DryRun`'s and
`Result.Notes`' doc comments: `Notes` cites "snowball-init follow-up instructions, a
missing-snowball warning" as its examples, which no longer occur — reword to what `release`
actually produces (dry-run previews).

#### Task 3 — strip the docs tests

In `internal/scaffold/scaffold_test.go` delete the ten `TestDocs*` functions
(`TestDocsCreatesAllFilesWithProjectName`, `…DefaultsProjectNameToRootBasename`,
`…RerunWithoutForceSkipsExisting`, `…ForceOverwrites`, `…DryRunWritesNothing`,
`…JustfileMissingIsSkippedNeverCreated`, `…JustfileAppendsMissingRecipesOnly`,
`…JustfileDryRunAppendsNothing`, `…SnowballDryRunDoesNotRunInit`,
`…MissingSnowballIsNonFatalWarning`). Fix `payloadRoot`'s doc comment, which names
`"scaffold/docs-template/..."` as the path it exposes — make it say `release-template`. Then
check the surviving `Test Release*` functions still compile: the `contains` and `mustRead`
helpers and the `os/exec`/`strings` imports may now be unused.

#### Task 4 — remove the CLI surface

In `internal/cli/scaffold.go`: delete `runScaffoldDocs` and the `case "docs":` arm; change the
two messages to `pickle scaffold: expected release` and
`pickle scaffold: unknown subcommand %q (want release)`; update the file's leading comment
("Scaffold commands: pickle scaffold docs, pickle scaffold release").

In `internal/cli/cli.go`, delete the `scaffold docs […]` entry from the "Other scaffolding
(unrelated to brine)" block in `usage()`, keeping the `scaffold release` entry and the block
heading.

In `internal/cli/cli_test.go`, delete the `"scaffold docs bad flag"` table case,
`TestScaffoldDocsIsUnrelatedToBrine` and `TestScaffoldDocsDryRunWritesNothing` (decision 7).
`TestScaffoldReleaseIsUnrelatedToBrine`'s doc comment says "mirroring
TestScaffoldDocsIsUnrelatedToBrine above" — reword; the referent is being deleted. Confirm the
`"scaffold unknown subcommand"` and `"scaffold no subcommand"` cases still pass against the
new messages.

#### Task 5 — docs

In `docs/user-manual/cli-reference.adoc`: delete the `| \`pickle scaffold docs …\`` Overview
row (and its description cell), and the whole `[#cmd-scaffold-docs]` section from its anchor
through to the line before `[#cmd-scaffold-release]`. Rewrite the release section's opening
paragraph per decision 8 so it states the "unrelated to the flow" property directly instead of
by comparison. No other page xrefs `<<cmd-scaffold-docs>>` — confirmed at refinement, but
re-grep, since `just docs-check` will catch a dead anchor anyway.

#### Task 6 — CHANGELOG

Add a `### Breaking` section to `## [Unreleased]` (decision 5) recording the removal of
`pickle scaffold docs`, naming *why*: the scaffold had to track `snowball`'s own internal
defaults — its `init` output, its PDF theme-file naming rule, its `leveloffset` convention —
from outside, which pickle has no authority over and kept getting wrong; the capability moves
to snowball, which owns both sides. Reword the unreleased T-113 `### Added` entry's "mirroring
`pickle scaffold docs`" clause (decision 6). Leave `## [0.11.0]` alone.

### Acceptance test

All four project commands, from the repo root:

```
just build
just test
just lint
just docs-check
```

All four clean. Then, specifically:

1. **The subcommand is gone, with the generic error** (decision 4):
   ```
   ./pickle scaffold docs; echo "exit=$?"
   ```
   prints `pickle scaffold: unknown subcommand "docs" (want release)` and `exit=2`
   (`exitUsage`). No mention of snowball, migration or deprecation.
2. **The bare verb group names only `release`:**
   ```
   ./pickle scaffold; echo "exit=$?"
   ```
   prints `pickle scaffold: expected release`, `exit=2`.
3. **`pickle help` is silent about it:**
   ```
   ./pickle help | grep -c 'scaffold docs'   # 0
   ./pickle help | grep -c 'scaffold release' # 1
   ```
4. **`scaffold release` still works end to end**, from a throwaway dir with the binary renamed
   per this repo's self-modify policy:
   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test scaffold release --project-name demo && ls CHANGELOG.md RELEASING.md
   ```
   Both files exist and contain `demo`.
5. **No trace left** (decision 4) — from the repo root:
   ```
   rg -n 'scaffold docs|docs-template' --glob '!tickets/**' --glob '!CHANGELOG.md' --glob '!graphify-out/**'
   ```
   returns nothing.
6. **The embed root still resolves:** `just build` succeeding is the proof (a `go:embed`
   pattern matching no files is a compile error), so confirm `scaffold/release-template/`'s two
   files are still present and step 4 passed.

### Docs update (mandatory when user-facing)

User-facing: a command is being removed. Task 5 covers
`docs/user-manual/cli-reference.adoc` (Overview row + the `[#cmd-scaffold-docs]` section + the
release section's dangling comparison), and Task 6 covers `CHANGELOG.md`. No other manual page
mentions the command — verified at refinement; `just docs-check` (`snowball check` plus the
`TestDocs*` xref gate) is the mechanical backstop for a dead anchor or a broken include.

Nothing under `skill/` changes: `scaffold docs` was never part of the brine payload, so no
`payload_version` implications and nothing for the foreign-workspace test to catch.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. Docs updated (cli-reference.adoc, CHANGELOG.md) — no registration needed, no new page.
3. Write a **summary**: files deleted, files edited, and explicitly note the two coverage
   items accepted as losses (the CLI-level `--dry-run` flag test, decision 7) and the one
   filing-time assumption overturned (the package survives, decision 1).
4. Suggest a Conventional Commit message, e.g.:

   ```
   feat(scaffold)!: remove pickle scaffold docs (T-117)

   Scaffolding an AsciiDoc docs skeleton required tracking snowball's own
   internal defaults - its init output, its <name>-theme.yml naming rule, its
   leveloffset convention - from outside, and pickle kept getting them wrong
   (T-116). snowball owns both sides of that contract; the capability moves
   there. pickle scaffold release is unaffected.
   ```

5. **Tidy up before presenting** — root-path child: interactive-rebase the WIP commits into a
   small number of atomic, correctly typed commits (a reasonable split: the code/payload
   removal, then the docs + CHANGELOG).
6. Commit locally on the ticket branch. Do **not** push or open an MR without explicit user
   approval. Present the commit message; after approval, keep the tidied history (root-path
   default), verify `git fetch origin main && git diff --name-only origin/main...HEAD | grep
   '^tickets/'` prints nothing, then push and open the MR. Merging is the human's.

## Review

**Verdict: PASS** — no blocking findings. All 6 tasks and all 8 confirmed design decisions
honoured; all four project commands green.

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the
      branch in this same session, so the audits (steps 2–4a) went to **two independently
      spawned reviewers**, each in its own isolated git worktree, briefed adversarially and
      instructed to find defects: one for implementation + quality (steps 2–3), one for
      consistency + docs (steps 4–4a). Both were verified to have made no edits (their result
      branches were byte-identical to the branch under review). Every finding below was
      re-verified by hand before being recorded — delegation buys independence, not accuracy.
- [x] Implementation audit — acceptance test re-run verbatim, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — run on both changed files. `cli-reference.adoc`: all
      suggestions landed on pre-existing prose the ticket did not touch, discarded as
      out of scope. `CHANGELOG.md`: one suggestion targeted this branch's own `### Breaking`
      entry and was applied (folded into F3's fix); the other three were pre-existing prose,
      discarded.
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages presented for approval; bookkeeping committed (step 9)

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | stale-xref | fixed inline | `projectNameToken`'s rewritten doc comment asserted a repo-wide substitution convention that does not exist — a true rationale (GitHub Actions `${{ }}` colliding with Go template delimiters) was replaced with an invented one when the template that justified it was deleted | `internal/scaffold/scaffold.go:22-23`; `rg '__[A-Z_]+__'` over the tree returns only this constant and `RELEASING.md` — no other embedded template is token-substituted at all | replaced with the true reason: one token across two files needs no template engine |
| F2 | non-blocking | stale-xref | fixed inline | `assets.go`'s summary sentence attributed all three embedded payload roots to `pickle scaffold release`, though `skill/` and `agents/` are embedded for the installer; Task 1 asked for this sentence to be fixed and it was narrowed in the wrong direction | `assets.go:19-20`, "Embedding these … lets pickle scaffold release"; the bulleted list it refers to spans three roots, two of them installer-only | reworded to name both surfaces |
| F3 | non-blocking | docs-gap | fixed inline | the new `### Breaking` changelog entry cited `SNOW-003` in the `unity` workspace — an id no reader of pickle's changelog can resolve. Same shape as the foreign-workspace test `AGENTS.md` applies to `skill/`, which `payload_lint_test.go` does not enforce outside that tree | `CHANGELOG.md:28`; the only other foreign-id hit in the file (`CHANGELOG.md:601`, `RICK-001`) is syntax filler in a grammar example, not a lookup, so there was no precedent | dropped the id, kept "the capability moves to snowball" |
| F4 | non-blocking | spec-unclear | noted | the ticket's own Acceptance test step 4 requires both scaffolded files to "contain `demo`", but the release `CHANGELOG.md` template carries no `__PROJECT_NAME__` token by T-113 design, so that half can never hold. Behaviour is correct; the criterion is wrong. The implementer noticed the mismatch during execution and waved it through rather than flagging it | `grep -c __PROJECT_NAME__ scaffold/release-template/CHANGELOG.md` → `0`, `RELEASING.md` → `1`; identical on `main`, so the criterion was never satisfiable | pre-existing at refinement, so not eligible for inline fix (rules §5 causation rule); a re-runner should read step 4 as "both files exist; `RELEASING.md` contains `demo`" |
| F5 | non-blocking | design | noted | `writeTemplates`' `files []templateFile` parameter and the `templateFile` type now have exactly one caller and one instantiation; `Release` is a pass-through wrapper. The generalisation was earned by two callers and now has one | `internal/scaffold/scaffold.go:75` (sole call), `:86` (signature), `:35` (sole `[]templateFile`) | defensible as API stability; collapsing it is a refactor a review should not perform unasked |
| F6 | non-blocking | stale-xref | noted | `Options.ProjectName`'s comment says "(Run fills this in)" but package `scaffold` has no `Run` — it has `Release` | `internal/scaffold/scaffold.go:43`; `git show main:internal/scaffold/scaffold.go` carries the identical line, so it predates this branch | pre-existing, so `noted` not `fixed inline` per rules §5 — "did this branch break it?", not "is it small?" |
| F7 | non-blocking | test-gap | noted | `TestReleaseWritesNoOtherFile` asserts the absence of `justfile`, `snowball.yaml` and `.github` — a list that only made sense while `Docs` existed to write them. It still passes, but can no longer fail for the reason it was written | `internal/scaffold/scaffold_test.go:132`; byte-identical on `main` apart from its comment | an "exactly two entries in root" assertion would restore its teeth |
| F8 | non-blocking | test-gap | noted | the two operator strings this branch changed (`expected release`, `unknown subcommand %q (want release)`) are pinned by no test; the CLI table asserts exit codes only, so decision 3's wording is unguarded | `internal/cli/scaffold.go:19,26` vs `internal/cli/cli_test.go:116-117` | pre-existing gap shape (the old strings were equally unguarded); a golden-output assertion would close it |

**Disposition summary:** 8 findings, 0 blocking, 8 non-blocking — 3 `fixed inline` (F1–F3), 5
`noted` (F4–F8), 0 `folded`, 0 `new ticket`. F5/F7/F8 share a theme (hardening the surviving
`scaffold release` surface) and were weighed together against the promotion test — *would this
actually be scheduled?* — and rejected: three small polish items in a package that just shrank
would not be picked up over anything currently in READY. They stay recorded with evidence, and a
later reviewer can promote them by citing these rows.

`cost: estimated S, actual S`

**Independent-audit note.** The two delegated reviewers converged, without contact, on F1 and F2
— both the falsehoods this branch itself introduced into code comments, and both invisible to
every mechanical gate (`build`/`test`/`lint`/`docs-check` were green over them). That is the
specific defect class step 0 exists to catch: the implementer had just decided each of those
sentences was correct.

## History

- 2026-08-23 — created (TO DO). source: chat: user, refining T-116, concluded docs/release
  scaffolding (doc skeleton, justfile fragments, GH release-attach workflow) is snowball's job,
  not pickle's, and asked for pickle's copy to be removed and the capability moved to snowball
- 2026-08-24 — TO DO → READY: plan complete
- 2026-08-24 — READY → IN DEVELOPMENT: picked up
- 2026-08-24 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-24 — IN REVIEW → DONE: review clean; 8 non-blocking, all dispositioned (3 fixed inline, 5 noted); audits delegated to two independent reviewers
- 2026-08-24 — published: branch pushed, MR #70 opened against main
  (https://github.com/codcod/pickle/pull/70). Awaiting the human merge.
