---
id: T-110
title: opt-in scaffold command for a docs/release template (docs skeleton, snowball config, release-attach action, justfile), modeled on pickle's own pipeline
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: medium
cost: M
---

# T-110 — opt-in scaffold command for a docs/release template (docs skeleton, snowball config, release-attach action, justfile), modeled on pickle's own pipeline

## Outcome

Running a new, explicitly opt-in `pickle` subcommand (name to be pinned at refinement, e.g.
`pickle scaffold docs`) in a target repo lays down a minimal AsciiDoc docs skeleton, delegates
to `snowball init` for the render config, adds `justfile` `docs-check`/`docs-build` recipes
(only if a `justfile` already exists — never invents a task runner), and drops a
GitHub-Actions release-attach step — all parameterized by project name, and entirely separate
from `pickle install`, whose contract (scaffold brine only) stays unchanged.

## Description

Origin: a chat request asked `pickle` to scaffold, for other repos, the same docs/user-manual
tooling this repo already has for itself — `docs/`, `snowball.yaml`, a GitHub Action that
attaches the built manual to a release, and justfile targets (`version`, `default`,
`docs-build`) — "all based on the pickle repo."

That idea was challenged before filing, because it doesn't fit `pickle` as shipped:

- **Mission mismatch.** `pickle install` scaffolds exactly one thing — the brine ticket flow
  (`tickets/`, `BOARD.md`, the skill, the `AGENTS.md` marker block). A docs/release build
  pipeline has nothing to do with tickets, boards, or agent workflow; folding it into
  `pickle install` would blur "the brine installer" into a generic repo bootstrapper.
- **Four stacked tool assumptions.** "Modeled on the pickle repo" bakes in AsciiDoc + snowball
  (a sibling project of the author's, distributed via a private Homebrew tap), GitHub as the
  forge (this repo's own `.goreleaser.yaml` had to pin the forge explicitly because goreleaser
  otherwise prefers GitLab when a token is present — the same fragility would leak into every
  scaffolded repo), goreleaser as the release pipeline the attach-Action assumes, and `just` as
  the task runner. None of these follow from "a project adopted brine."
- **Partial duplication of an existing tool.** `snowball init` already writes a starter
  `snowball.yaml` in the current directory (verified: `snowball init --help`). Hand-rolling that
  file inside pickle duplicates a command one `brew install` away.
- **Drift risk.** Embedding a copy of this repo's *own* docs pipeline as a template means every
  future change to it (this repo's own T-047/T-048 and the release-workflow hardening since)
  needs a matching update to the embedded copy or the template silently rots.

Decision (user-approved): build it anyway, but scoped down per the above, as its own **opt-in**
subcommand — never folded into `pickle install`, never registered as an ongoing `pickle.toml`
concern (it is a one-shot scaffold, not something `doctor`/`upgrade` need to keep fresh like the
brine skill payload):

1. **Separate command surface**, not a flag on `pickle install`/`project add`. Exact verb/name
   is an open decision for refinement (`pickle scaffold docs` vs. `pickle docs init` vs.
   similar) — pick whatever reads clearly as unrelated to brine.
2. **Parameterize everything currently hardcoded to "pickle"** in this repo's own setup:
   project/binary name, the rendered artifact name (`pickle-user-manual` → `<project>-user-manual`),
   and the release-tag→version substitution in the attach step.
3. **Delegate to `snowball init`** for `snowball.yaml` instead of writing one from a template —
   shell out to it (or document it as a required manual step) rather than duplicating its
   defaults.
4. **Scaffold a minimal generic doc skeleton**, not this repo's actual manual content: one book
   master plus one placeholder chapter — `docs/user-manual/{installation,cli-reference,
   concepts/*,quickstart}.adoc` is pickle-specific prose and must not be copied verbatim.
5. **justfile recipes are additive, not assumed.** Only add `docs-check`/`docs-build` (and,
   per the ask, confirm `version`/`default` exist rather than overwriting them) when a
   `justfile` is already present; refinement must decide the no-`justfile` behavior (skip with
   a message vs. offer to create one) rather than silently inventing a task runner.
6. **GitHub-only, stated as a documented limitation**, not silently assumed universal — this
   repo's own release-attach pattern (`.github/workflows/release.yml`'s guarded `gh release
   upload` step, soft-failing so a broken docs build never blocks the release) is the only
   reference implementation available; no GitLab/other-forge equivalent is in scope here.

Open questions for refinement: final command name/UX; whether embedded scaffold assets follow
the existing `assets.go` `go:embed` pattern (`payloadFS`) or a new embed target; exact
placeholder content for the minimal doc skeleton and the release-attach Action step; and
whether `pickle doctor`/`board audit` need any awareness of this at all (current expectation:
no — nothing scaffolded here is an ongoing invariant pickle enforces).

Soft coupling: none hard-blocking. Loosely related to **T-047**/**T-048** (this repo's own docs
pipeline, the reference the scaffold is modeled on) and **T-011** (distribution/goreleaser,
the release pipeline the attach-Action assumes) — informative precedent only, not a
`depends-on:`.

## Implementation Plan

### 0. Feature branch (mandatory)

Inside the target child-project's repo (`pickle`, path `.`):

```
git checkout main
git checkout -b feat/T-110-scaffold-docs-command
```

Commit locally as work proceeds. Never push or open a merge request without explicit user
approval; end with a summary and suggested commit message (Finish, below).

### Prerequisite gate (hard)

- Clean tree on `main`; no `depends-on:`.
- `snowball` (Homebrew `codcod/tap/snowball`) installed locally for manual verification of the
  `snowball init` shell-out path (`snowball doctor` reporting toolchain issues is fine — only
  its *presence on PATH* is exercised by this ticket, not a full render).

### Confirmed design decisions (do not deviate without asking)

1. **New verb group `pickle scaffold docs`**, not a flag on `install`/`project add`. Mirrors
   the nested-verb style already used by `project`/`board`/`hooks` in `internal/cli/cli.go`.
   Flags: `--project-name <name>` (default: `filepath.Base(root)`, same default rule as
   `install`'s `--project`), `--force` (overwrite files that already exist), `--dry-run`
   (list what would be created/changed; writes nothing). Always operates on the current
   working directory root (`pickle.toml` is *not* required to exist — this command does not
   touch brine's own tree at all).
2. **New `internal/scaffold` package**, structured like `internal/install`: an exported
   `Docs(payload fs.FS, root string, opts Options) (Result, error)` that returns created /
   skipped / warning entries for the CLI to print, mirroring `install.Run`'s `Result` shape
   (see `internal/install/install_test.go`'s use of `res.Created`).
3. **Embed the templates via the existing `payloadFS`.** Add a third embedded root in
   `assets.go` next to `skill/`/`agents/`: `scaffold/docs-template/` (one more
   `go:embed all:` entry, same FS — no new embed variable).
4. **Plain string substitution, not `text/template`.** The GitHub Actions template contains
   `${{ github.event.release.tag_name }}` — Go's `{{ }}` delimiters would collide with GitHub
   Actions' own `${{ }}` syntax. Use a single literal placeholder token,
   `__PROJECT_NAME__`, replaced with `strings.ReplaceAll` before writing each file. No
   templating engine, no partial-render edge cases.
5. **`snowball.yaml` is never hand-written by pickle.** If `snowball` is found on `PATH`
   (`exec.LookPath`), shell out to `snowball init` in `root` (best-effort: a non-zero exit or
   missing binary is a **warning** in the result, never a fatal error — the rest of the
   scaffold still completes). Either way, print follow-up guidance: edit the generated
   (or to-be-generated) `snowball.yaml` so `src: docs/user-manual.adoc` and
   `out: <project-name>-user-manual`. Do not parse or rewrite the YAML.
6. **justfile: additive only, never invented.** If `justfile` does not exist at `root`, skip
   with a recorded note ("no justfile — skipped") — do not create one. If it exists, append
   `docs-check`/`docs-build` recipes (bodies: `snowball check` / `snowball build -o dist/docs`,
   matching this repo's own `justfile`) **only for whichever of the two recipe names is not
   already present** (match on a line starting with `docs-check:` / `docs-build:`); never
   touch `version`/`default` or any existing recipe body.
7. **The GitHub Action is a standalone workflow, decoupled from goreleaser.** Unlike this
   repo's own `release.yml` (which piggybacks on the goreleaser job), the scaffolded
   `.github/workflows/docs-release.yml` triggers on `release: types: [published]` — it fires
   regardless of what tool cut the release, so it carries no goreleaser (or any other
   build-tool) assumption. It still assumes **GitHub** as the forge; say so in a header
   comment in the generated file itself (decision 6 of the Description).
8. **No `pickle.toml`/`doctor`/`board audit` integration.** This is a one-shot scaffold, not
   an ongoing invariant pickle enforces or upgrades.

### Tasks

#### Task 1 — embed the templates

Create, under `scaffold/docs-template/`:

- `attributes.adoc`:
  ```
  // Shared attribute definitions — included by the book master document.
  :product: __PROJECT_NAME__
  ```
- `user-manual.adoc`:
  ```
  = __PROJECT_NAME__ User Manual
  :doctype: book
  :toc:
  :toclevels: 3
  :sectnums:
  :icons: font

  include::attributes.adoc[]

  include::user-manual/introduction.adoc[leveloffset=+1]
  ```
- `user-manual/introduction.adoc`:
  ```
  == Introduction

  // TODO: replace this placeholder chapter. This file (and user-manual.adoc's include of
  // it) was scaffolded by `pickle scaffold docs` — it is a starting point, not content.

  {product} does ... (describe what it does here).
  ```
- `workflows/docs-release.yml` — a full workflow: header comment stating the GitHub-only
  assumption (decision 7), `permissions: contents: write` declared explicitly (matching this
  repo's own workflows' always-explicit convention — `gh release upload` needs it, and an
  adopting repo's default token permissions should not be relied on), `on: release: types:
  [published]`, one job that installs snowball defensively — `brew update --quiet` before
  `brew install codcod/tap/snowball`, then `snowball setup` — rather than the bare two-command
  form, because a stale preinstalled Homebrew snapshot is a known cause of install failures on
  fresh runners (self-contained hardening, written into the template directly; the generated
  file must stand on its own and must not cite this repo's own ticket ids or script paths),
  runs `snowball build -o dist/docs` with `continue-on-error: true` (soft-fail, matching this
  repo's own `.github/workflows/release.yml` rationale), then a guarded step that uploads
  whatever exists in `dist/docs/*.{pdf,epub}` via `gh release upload "$TAG" ... --clobber` and
  exits 0 with a `::warning::` annotation when nothing was built — same shape as this repo's
  own "Attach user manual to release" step, minus the goreleaser coupling and the
  version-from-archive-name renaming (there is no goreleaser archive here to match names
  against).

Wire the embed in `assets.go`: change `//go:embed all:skill all:agents` to
`//go:embed all:skill all:agents all:scaffold` (one line; `payloadFS` unchanged otherwise).

#### Task 2 — `internal/scaffold/scaffold.go`

- `type Options struct { ProjectName string; Force bool; DryRun bool }`
- `type Result struct { Created, Skipped, Warnings []string }`
- `func Docs(payload fs.FS, root string, opts Options) (Result, error)`:
  1. Default `opts.ProjectName` to `filepath.Base(root)` if empty.
  2. For each of `docs/attributes.adoc`, `docs/user-manual.adoc`,
     `docs/user-manual/introduction.adoc`, `.github/workflows/docs-release.yml`: read the
     matching `scaffold/docs-template/...` file from `payload`, `strings.ReplaceAll` the
     `__PROJECT_NAME__` token, then either write it (creating parent dirs), skip it (exists,
     no `--force`), or just record the intended action (`--dry-run`) — append the outcome to
     `Created`/`Skipped`.
  3. justfile: implement decision 6 exactly (read, check for `docs-check:`/`docs-build:`
     line prefixes, append only what's missing, or skip with a note if no justfile).
  4. snowball: `exec.LookPath("snowball")`; if found and not `--dry-run`, run
     `snowball init` with `Dir: root` (capture combined output; a non-zero exit becomes a
     `Warnings` entry, not a returned `error`); always append the `src`/`out` follow-up
     guidance line to `Warnings` (or a dedicated `Result.Notes` — pick whichever reads more
     naturally when printed, and keep it consistent with `Created`/`Skipped`'s style).
  5. Never touch `pickle.toml`, `tickets/`, or anything brine-owned.

#### Task 3 — `internal/cli/scaffold.go` + dispatch

- `runScaffold(args []string) int`: `args[0]` must be `"docs"` (usage error otherwise,
  `exitUsage`, mirroring `runHooks`'s sub-verb dispatch).
- `runScaffoldDocs(args []string) int`: `flag.NewFlagSet("scaffold docs", ...)` with
  `--project-name`, `--force`, `--dry-run`; resolve `root` the same way `runInstall` does
  (current working directory); call `scaffold.Docs`; print `Created`/`Skipped`/`Warnings`
  (one line each, same prefix style as `install`'s summary); `exitOK` unless `Docs` returned
  an `error` (`exitError`).
- In `internal/cli/cli.go`: add `case "scaffold": return runScaffold(args[1:])` to `Run`'s
  switch, and a new block in `usage()` (e.g. under "Other") documenting
  `scaffold docs [--project-name <name>] [--force] [--dry-run]` in one line, consistent with
  the existing entries' style.

#### Task 4 — tests

- `internal/scaffold/scaffold_test.go` (mirrors `internal/install/install_test.go`'s
  `payloadRoot()`/`mustExist` helpers): fresh scaffold creates all four files with the
  project name substituted; re-run without `--force` skips existing files and reports them
  in `Skipped`; `--force` overwrites; `--dry-run` writes nothing (`os.Stat` still fails after);
  justfile present without the two recipes gets both appended; justfile present with one
  already defined gets only the other appended, verbatim-preserving the existing one; no
  justfile at all → `Skipped` note, no file created; `snowball` absent from `PATH` (manipulate
  `PATH` in the test) → a `Warnings` entry, `Docs` still returns success and every other file
  is still created.
- `internal/cli/scaffold_test.go` (or extend `cli_test.go`, matching its existing style):
  `pickle scaffold` with no sub-verb / an unknown sub-verb → `exitUsage`; `pickle scaffold
  docs` in a temp dir → `exitOK`, files present.

### Acceptance test

From the repo root on the feature branch:

```sh
just build && just test && just lint    # green, including the new packages/tests

D=$(mktemp -d) && cd "$D"
/path/to/repo/pickle scaffold docs --project-name demo
test -f docs/attributes.adoc && grep -q ':product: demo' docs/attributes.adoc
test -f docs/user-manual.adoc && grep -q '= demo User Manual' docs/user-manual.adoc
test -f docs/user-manual/introduction.adoc
test -f .github/workflows/docs-release.yml && grep -q "release:" .github/workflows/docs-release.yml
actionlint .github/workflows/docs-release.yml   # the scaffolded workflow itself lints clean

# re-run without --force: nothing overwritten
/path/to/repo/pickle scaffold docs --project-name demo   # reports Skipped, exit 0

# dry-run on a second empty dir writes nothing
D2=$(mktemp -d) && cd "$D2"
/path/to/repo/pickle scaffold docs --dry-run
test ! -d docs

# justfile additive behaviour
D3=$(mktemp -d) && cd "$D3" && printf 'default:\n\t@just --list\n' > justfile
/path/to/repo/pickle scaffold docs
grep -q '^docs-check:' justfile && grep -q '^docs-build:' justfile
```

(`actionlint` is already a `lint-ci-surface` dependency per the `justfile` — reuse it here
rather than introducing a new YAML linter.)

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`: add a `== \`pickle scaffold docs\`` section (own
  heading, alongside `pickle serve` etc.) documenting the command, its three flags, exactly
  what it creates, the justfile additive rule, and the snowball-init best-effort behaviour;
  add one row to the `== Overview` table at the top of the same file.
- `internal/cli/cli.go`'s `usage()` text: one new line under a command grouping (Task 3).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (cli-reference.adoc + usage()).
3. Write a summary (files touched, decisions honoured, anything deferred).
4. Suggested commit message:

   ```
   feat(cli): add opt-in `scaffold docs` command for docs/release tooling (T-110)

   New `pickle scaffold docs` lays down a minimal AsciiDoc docs skeleton,
   shells out to `snowball init` best-effort, appends justfile docs-check/
   docs-build recipes when a justfile exists, and scaffolds a standalone
   GitHub Action that attaches the built manual to a release — entirely
   separate from `pickle install`, which continues to scaffold brine only.
   ```

5. Commit locally on the ticket branch; do **not** push or open an MR without user approval.
   `pickle ticket move T-110 in-review --reason "acceptance green"` and hand back.

## Review

2026-08-22 — review on `feat/T-110-scaffold-docs-command`, per the review protocol. No layered
addenda configured (`pickle.toml` has neither an overarching nor a per-child
`review_addendum`).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — ran `docs_readability` on the four `.adoc` files this
  ticket touched (`docs/user-manual/cli-reference.adoc` and the three new
  `scaffold/docs-template/**/*.adoc` files). All suggestions on `cli-reference.adoc` target
  prose *outside* this ticket's diff (the file's pre-existing locking/hooks/changelog
  sections) — not applied, same precedent as T-048. One suggestion on
  `scaffold/docs-template/user-manual/introduction.adoc` (content this ticket authored)
  sharpened the placeholder sentence — applied; the other two new template files were judged
  to already read well.
- [x] Findings recorded with severity **and** disposition per the rules §5; summary line present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated / board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket in `1-to-do/`/`2-ready/`
  references T-110
- [x] Summary + commit message & MR attributes presented for approval (step 9)

**Implementation audit** (step 2) — all met, verified on the actual tree (branch checked out,
`3a02a78` + the two review-fix commits below):

- Decisions 1–8 — **met**: `pickle scaffold docs` is its own verb group, untouched by
  `install`/`project add`; `internal/scaffold.Docs` mirrors `internal/install`'s
  `Options`/`Result` shape; the embed is the third `go:embed all:` root in `assets.go`, same
  `payloadFS`; the GitHub Actions template's `${{ }}` is untouched by the plain
  `__PROJECT_NAME__`/`strings.ReplaceAll` substitution; `snowball.yaml` is never hand-written
  (best-effort `snowball init` shell-out, non-fatal on failure/absence, follow-up guidance
  always printed); justfile recipes are additive-only, never inventing a `justfile`; the
  scaffolded Action triggers on `release: published` with no goreleaser coupling, states the
  GitHub-only assumption in its own header, and (post plan-amendment) declares
  `permissions: contents: write` and installs snowball defensively (`brew update` first). No
  `pickle.toml`/`doctor`/`board audit` integration was added.
- Tasks 1–4 — **met**: all four template files present under `scaffold/docs-template/` with
  the exact planned content; `internal/scaffold/scaffold.go` and `internal/cli/scaffold.go`
  implement the plan's `Options`/`Result`/dispatch shape; tests cover every case Task 4 listed.
- Acceptance test — **re-run verbatim on the branch, green**: `just build && just test &&
  just lint` clean; fresh scaffold creates all four files with `demo` substituted;
  `actionlint` on the generated workflow clean; re-run without `--force` reports every file
  skipped; `--dry-run` on an empty dir writes nothing; justfile-additive case appends both
  recipes. `just docs-check` also re-run clean (not in the plan's literal list, but part of
  this project's standing build-green bar).

**Quality/consistency audits** (steps 3–4) — the scaffold-vs-install symmetry, the token
choice (plain string replace, not `text/template`, specifically to avoid colliding with the
workflow's own `${{ }}`), and the additive-justfile/best-effort-snowball edge cases were all
sound. Three findings below were worth recording; none blocked shipping.

**Documentation audit** (step 4a) — `docs/user-manual/cli-reference.adoc` gained an Overview
row and a full `pickle scaffold docs` section (flags, exactly what it writes, the additive
justfile rule, the best-effort snowball behaviour); `internal/cli/cli.go`'s `usage()` gained a
matching line. Whole-tree sweep found one gap (F3, below). Docs build (`just docs-check`)
clean throughout.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | design | fixed inline | The scaffolded `docs-release.yml` pinned `actions/checkout@v4`, while every workflow this repo maintains for itself (`ci.yml`, `release.yml`, `manual-smoke.yml`) pins `@v7` — an asymmetry with no behaviour difference, but an odd one to ship in a template explicitly "modeled on pickle's own pipeline" | `grep -rn 'actions/checkout@' .github/workflows/*.yml scaffold/docs-template/workflows/*.yml` showed `@v7` everywhere except the new template's `@v4` | Bump the template to `@v7` |
| F2 | non-blocking | test-gap | fixed inline | `scaffoldJustfile`'s and `scaffoldSnowballConfig`'s `--dry-run` branches (the "would append"/"would run `snowball init`" notes) had no automated coverage — only exercised by this review's manual smoke test, not by `internal/scaffold/scaffold_test.go` | `TestDocsDryRunWritesNothing` only asserted the four template files; no test constructed a justfile or asserted the snowball-found dry-run note | Add a justfile-present dry-run case and a snowball-on-PATH dry-run case |
| F3 | non-blocking | docs-gap | fixed inline | `CHANGELOG.md`'s `[Unreleased]` section had no entry for this new user-facing command, breaking this repo's own consistent per-ticket convention (T-108/T-105/T-106 each added one) — `pickle changelog check` is deliberately advisory ("always exits 0"), which is why this is non-blocking rather than a 4a.1 blocking gap despite naming `CHANGELOG.md` in the `docs-gap` class scope | `git log -p --follow -- CHANGELOG.md` shows T-108/T-105/T-106 each added an `[Unreleased]` bullet in their own commit; T-110's had none until this review | Add an `[Unreleased]` bullet for T-110 |

**Disposition summary:** 3 findings — 3 fixed inline (F1, F2, F3); 0 blocking, 0 noted, 0
folded, 0 new tickets.

`cost: estimated M, actual M`

**Verdict:** no blocking findings → DONE. All three fixed-inline findings are already on the
branch (`95568d7` pins `@v7` + adds the two dry-run tests; `a90597d` adds the `CHANGELOG.md`
entry); `just build && just test && just lint && just docs-check` and `pickle changelog check`
all re-verified clean after both fix commits.

## History

- 2026-08-21 — created (TO DO). source: chat: user asked pickle to scaffold docs/release tooling (docs dir, snowball.yaml, release-attach GitHub Action, justfile targets) modeled on this repo; idea was challenged (mission mismatch, tool-assumption stacking, `snowball init` duplication) and the user chose to proceed scoped down as a separate opt-in subcommand rather than folding it into `pickle install`
- 2026-08-21 — TO DO → READY: plan complete
- 2026-08-21 — plan amended inline: pre-pickup applicability gate (fresh sub-agent audit, no blocking findings) folded two gaps into Task 1's `docs-release.yml` template — defensive `brew update` before `snowball` install (mirrors this repo's own runner-flakiness fix, written in generically rather than cited by ticket id) and an explicit `permissions: contents: write` block (matches this repo's always-explicit convention; `gh release upload` needs it)
- 2026-08-21 — READY → IN DEVELOPMENT: picked up
- 2026-08-22 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-22 — IN REVIEW → DONE: review clean: 0 blocking, 3 fixed inline (F1 actions/checkout version, F2 dry-run test coverage, F3 changelog entry)
- 2026-08-22 — PR #56 opened (`feat/T-110-scaffold-docs-command` → `main`, commit `cb64264`), user-approved; awaiting merge
- 2026-08-22 — merged to main (PR #56, `323add9`); branch deleted
