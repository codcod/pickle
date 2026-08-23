---
id: T-116
title: pickle scaffold docs produces a snowball pipeline that fails check/build out of the box
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-116 — pickle scaffold docs produces a snowball pipeline that fails check/build out of the box

## Outcome

A freshly scaffolded `pickle scaffold docs` tree passes `snowball check` and `snowball build`
once the user makes the `snowball.yaml` edits the command's own printed guidance tells them to
make — that guidance now covers `theme:` and the phantom second book, not just `src`/`out` as
today — instead of failing for three undocumented reasons no guidance mentions at all.

## Description

Reproduced against the shipped binary (`pickle scaffold docs --project-name <name>` in an
empty directory with `snowball` on `PATH`, no other flags): the command reports success, and
the project-name substitution into `docs/attributes.adoc` / `docs/user-manual.adoc` is correct
— but the resulting tree fails `snowball check` and `snowball build` outright, for three
independent reasons, none of which T-110's acceptance test caught (it only asserted file
existence and `grep`-matched substituted content; it never actually ran `snowball check` or
`snowball build` against the scaffold's own output):

1. **Wrong heading level in the placeholder chapter.** `scaffold/docs-template/user-manual/
   introduction.adoc` opens with `== Introduction` (level 1). `user-manual.adoc` includes it
   with `leveloffset=+1`, bumping it to level 2 — invalid as the first section after the book
   title (asciidoctor expects level 0 or 1 there). `snowball check` fails with: `section title
   out of sequence: expected levels 0 or 1, got level 2`. Compare this repo's own
   `docs/user-manual/quickstart.adoc`, which opens with `= Quickstart` (level 0) precisely so
   `leveloffset=+1` lands it at level 1 — the template should follow the same convention.

2. **`snowball init`'s default config references a book the scaffold never creates.** The
   scaffold shells out to `snowball init` for `snowball.yaml` (by design, T-110 decision 5) but
   never inspects or corrects what it writes. Its default output includes a second `books:`
   entry, `src: docs/developer-handbook.adoc`, which does not exist anywhere in the scaffold.
   `snowball check`/`build` fail on it: `input file .../docs/developer-handbook.adoc is
   missing`.

3. **`snowball init`'s default config references a theme file the scaffold never creates.**
   The same default output sets `theme: docs/pdf-theme/ai-sdlc-theme.yml`; `pickle scaffold
   docs` writes nothing under `docs/pdf-theme/`, unlike this repo's own pipeline (which ships
   `docs/pdf-theme/pickle-theme.yml` and points its own `snowball.yaml` at it). Asciidoctor-pdf
   recovers by silently falling back to its built-in default theme and still renders the PDF,
   but `snowball build`/`check` still exit non-zero (`could not locate or load the pdf theme
   ... reverting to default theme` followed by `Error: exit status 1`) — a hard failure for
   anything scripting on the exit code, including the scaffold's own `.github/workflows/
   docs-release.yml`, whose `snowball build` step this would otherwise trip.

Fix shape (confirm at refinement): (1) is a one-line template fix (`==` → `=`). (2) and (3)
are the same underlying gap — the scaffold's post-`snowball init` guidance note (currently only
covering `src:`/`out:`) needs to also tell the user to drop the phantom second book and either
remove the `theme:` line or point it at a theme the scaffold ships; shipping a small generic
theme file (mirroring `docs/pdf-theme/pickle-theme.yml`, which is already generic — `extends:
default` plus a few font tweaks, not pickle-branded) under `scaffold/docs-template/pdf-theme/`
is the more complete fix and keeps decision 5's "never hand-write snowball.yaml" intact
(pickle still doesn't parse/rewrite the YAML — it ships an asset and documents where to point
at it, same shape as the existing `src`/`out` guidance). Either way, the acceptance test must
gain a step that actually runs `snowball check` (and ideally `snowball build`) against a fresh
scaffold, since that is what would have caught all three of these at review time.

Soft coupling: spawned by manual reproduction of T-110's shipped command, not a review of
T-110 itself (T-110 is already `6-done/`) — filed as a bug against the feature it delivered.

**Confirmed at refinement:** the shipped theme asset is named `manual-theme.yml`, not the
generic `theme.yml` first floated. Verified live against `snowball` 0.2.2: asciidoctor-pdf's
theme loader derives a *theme name* from the `theme:` path by stripping a trailing `.yml` and
then a trailing `-theme`, and reloads `<dir>/<name>-theme.yml` — so the configured file **must**
itself end in `-theme.yml` or the reconstructed filename won't match what's on disk. This is
exactly how `pickle-theme.yml` happened to work by accident and a plain `theme.yml` silently
doesn't (it resolves to a bogus `theme-theme.yml`, falls back to the built-in default theme, and
still exits non-zero) — reproduced live in two sibling projects (`morty`, `summer`) that had
independently copied this repo's own `docs/pdf-theme/pickle-theme.yml` verbatim to unblock
themselves; both were corrected to a de-branded `docs/pdf-theme/manual-theme.yml` (generic
comment, no `pickle` naming or ticket-id citation) as part of this refinement, confirming the
fix shape before it was written into the plan below.

## Implementation Plan

### 0. Feature branch (mandatory)

Inside the target child-project's repo (`pickle`, path `.`):

```
git checkout main
git checkout -b feat/T-116-scaffold-docs-snowball-fixes
```

Commit locally as work proceeds. Never push or open a merge request without explicit user
approval; end with a summary and suggested commit message (Finish, below).

### Prerequisite gate (hard)

- Clean tree on `main`; no `depends-on:`.
- `snowball` (Homebrew `codcod/tap/snowball`) installed locally — required this time, not just
  optional: the acceptance test below actually runs `snowball check`/`build` against a fresh
  scaffold, which T-110's acceptance test never did.

### Confirmed design decisions (do not deviate without asking)

1. **Fix the heading level in place, no structural change.** `scaffold/docs-template/
   user-manual/introduction.adoc` changes only its first line, `== Introduction` →
   `= Introduction`, matching this repo's own `docs/user-manual/quickstart.adoc` convention
   (chapter files open at level 0 so `leveloffset=+1` lands them at level 1 under the book
   title). No other line changes.
2. **Ship one new generic theme asset, not a rewrite of `snowball.yaml`.** Decision 5 of T-110
   (pickle never hand-writes or parses `snowball.yaml`) stays intact. Add
   `scaffold/docs-template/pdf-theme/manual-theme.yml`, written to `docs/pdf-theme/
   manual-theme.yml`, as a fifth entry in `templateFiles` (`internal/scaffold/scaffold.go`) —
   same substitution/force/dry-run handling as the existing four, though this file has no
   `__PROJECT_NAME__` token to substitute. Content (de-branded, no `pickle` naming or ticket-id
   citation — this ships to other projects):
   ```yaml
   # AsciiDoctor-PDF theme: extends the built-in default with minor readability tweaks.
   extends: default

   base:
     font-size: 10.5

   heading:
     font-style: bold

   code:
     font-size: 9

   table:
     font-size: 9.5
   ```
3. **The filename must end in `-theme.yml` — this is load-bearing, not stylistic.** Confirmed
   live against `snowball` 0.2.2 (Description): asciidoctor-pdf strips a trailing `-theme.yml`
   from the `theme:` path to derive a theme *name*, then reloads `<dir>/<name>-theme.yml`. A
   file not ending in `-theme.yml` (e.g. a plain `theme.yml`) resolves to the wrong filename,
   silently falls back to the built-in default theme, and still exits non-zero. Do not rename
   this asset to anything not ending in `-theme.yml`.
4. **Extend the existing post-`snowball init` guidance, don't add a second note.** In
   `scaffoldSnowballConfig` (`internal/scaffold/scaffold.go`), change the `guidance` constant
   from:
   ```go
   const guidance = "snowball.yaml: point it at the scaffolded manual — " +
       "set `src: docs/user-manual.adoc` and `out: <project-name>-user-manual`."
   ```
   to also cover the theme and the phantom second book `snowball init` writes by default
   (`docs/developer-handbook.adoc`, which this scaffold never creates):
   ```go
   const guidance = "snowball.yaml: point it at the scaffolded manual — " +
       "set `src: docs/user-manual.adoc`, `out: <project-name>-user-manual`, and " +
       "`theme: docs/pdf-theme/manual-theme.yml`; drop `snowball init`'s second `books:` " +
       "entry (`docs/developer-handbook.adoc` by default) — this scaffold does not create it."
   ```
   This note is already printed unconditionally (snowball found or not, init succeeds or
   fails) — no change to when/whether it fires, only its text.
5. **No `pickle.toml`/`doctor`/`board audit` integration** (unchanged from T-110 decision 8) —
   this remains a one-shot scaffold.

### Tasks

#### Task 1 — fix the heading level

In `scaffold/docs-template/user-manual/introduction.adoc`, change the first line from
`== Introduction` to `= Introduction`. No other change in that file.

#### Task 2 — ship the generic theme asset

- Create `scaffold/docs-template/pdf-theme/manual-theme.yml` with the exact content in
  decision 2 above.
- In `internal/scaffold/scaffold.go`, add
  `{"scaffold/docs-template/pdf-theme/manual-theme.yml", "docs/pdf-theme/manual-theme.yml"}`
  to `templateFiles`.
- No embed-directive change needed — `assets.go`'s `//go:embed all:skill all:agents
  all:scaffold` already covers the new file; update its doc comment (the bullet enumerating
  `scaffold/docs-template/`'s contents) to list `pdf-theme/manual-theme.yml` alongside the
  existing four.

#### Task 3 — complete the post-init guidance

In `scaffoldSnowballConfig` (`internal/scaffold/scaffold.go`), replace the `guidance` constant
with the text in decision 4 above. No other logic in that function changes (still best-effort,
still non-fatal, still printed unconditionally).

#### Task 4 — tests (`internal/scaffold/scaffold_test.go`)

- Update the four existing tests that enumerate the fixed file set
  (`TestDocsCreatesAllFilesWithProjectName`, `TestDocsRerunWithoutForceSkipsExisting`,
  `TestDocsMissingSnowballIsNonFatalWarning`) to include `docs/pdf-theme/manual-theme.yml` as a
  fifth expected `Created`/`Skipped` entry.
- Add `TestDocsIntroductionHeadingIsLevelZero`: after a fresh `Docs(...)`, read
  `docs/user-manual/introduction.adoc` and assert it starts with `= Introduction` (not `==`).
- Add `TestDocsScaffoldPassesSnowballCheck` (skip via `exec.LookPath("snowball")` if absent,
  same pattern as `TestDocsSnowballDryRunDoesNotRunInit`): run a fresh `Docs(payload(), root,
  Options{ProjectName: "demo"})` (letting the real `snowball init` shell-out populate
  `snowball.yaml`), then apply exactly the edits the guidance note now describes — trim
  `books:` to the single `docs/user-manual.adoc` entry and set its `theme:` to
  `docs/pdf-theme/manual-theme.yml` — and run `exec.Command("snowball", "check")` with `Dir:
  root`, asserting a clean exit. This is the regression guard T-110 lacked: it would have
  caught all three defects this ticket fixes.

### Acceptance test

From the repo root on the feature branch:

```sh
just build && just test && just lint    # green

D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"
./pickle-test scaffold docs --project-name demo
test -f docs/pdf-theme/manual-theme.yml
grep -q '^= Introduction' docs/user-manual/introduction.adoc

# apply exactly the edits the printed guidance describes
# (edit snowball.yaml: single book, out: demo-user-manual, theme: docs/pdf-theme/manual-theme.yml)
snowball check     # must exit 0
snowball build -o dist/docs   # must exit 0; dist/docs/*.pdf and *.epub present
```

(`actionlint` on the untouched `docs-release.yml` template is not re-run here — this ticket
doesn't touch that file.)

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`, `== \`pickle scaffold docs\`` section: add
  `docs/pdf-theme/manual-theme.yml` as a fifth row in the "What it writes" table, and update
  the best-effort-snowball bullet to mention the theme and the dropped second book (not just
  `src`/`out`).
- `CHANGELOG.md`: add an `[Unreleased]` bullet for this fix, per this project's per-ticket
  convention (T-110's own review, finding F3).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (cli-reference.adoc + CHANGELOG.md).
3. Write a summary (files touched, decisions honoured, anything deferred).
4. Suggested commit message:

   ```
   fix(scaffold): make scaffolded docs pass snowball check/build (T-116)

   `pickle scaffold docs` shipped a chapter with the wrong AsciiDoc heading
   level, and its post-`snowball init` guidance never mentioned the default
   config's phantom second book or its dangling theme reference — together
   these made every fresh scaffold fail `snowball check`/`build`. Fixes the
   heading, ships a generic `docs/pdf-theme/manual-theme.yml`, and extends
   the guidance to cover both remaining `snowball.yaml` edits.
   ```

5. Commit locally on the ticket branch; do **not** push or open an MR without user approval.
   `pickle ticket move T-116 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-23 — created (TO DO). source: chat: user ran `pickle scaffold docs` in a foreign project and reported the scaffolded docs looked wrong; reproducing it in a scratch dir surfaced three independent defects (bad heading level, a phantom second book, and a dangling pdf-theme reference) that make a fresh scaffold fail `snowball check`/`build` out of the box
- 2026-08-23 — TO DO → READY: plan complete
