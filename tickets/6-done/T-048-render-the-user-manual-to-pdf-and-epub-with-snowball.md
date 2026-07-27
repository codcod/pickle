---
id: T-048
title: render the user manual to PDF and EPUB with snowball
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-048 — render the user manual to PDF and EPUB with snowball

## Description

T-047 shipped the AsciiDoc user manual (`docs/user-manual.adoc`, book master + parts) with a
deliberately minimal build: sources are the deliverable, and `just docs-check` renders to
`/dev/null` via a `Gemfile`-pinned `asciidoctor` to catch broken includes/xrefs (T-047,
decision 1: "content only — no PDF/EPUB"). The user now wants renderable artifacts: **PDF and
EPUB output for the manual, produced by `snowball`**.

[`snowball`](https://github.com/codcod/snowball) (source: `~/Projects/private/snowball`; the
binary is installed via Homebrew) is a single Go binary that renders AsciiDoc book masters to
PDF and EPUB by orchestrating the native asciidoctor toolchain (`asciidoctor-pdf`,
`asciidoctor-epub3`, optionally `asciidoctor-diagram`/`mmdc`). It is config-driven
(`snowball.yaml`: books, formats, optional PDF theme, shared attributes file, revision
stamping, per-format failure levels) with commands `init`/`setup`/`doctor`/`build`/`check` —
`build` renders artifacts, `check` validates masters and discards output (MR pipelines).

Shape (decisions pinned at refinement, 2026-07-26, with the user):

- a `snowball.yaml` at the repo root: one book (`src: docs/user-manual.adoc`,
  `attributes: docs/attributes.adoc`, `formats: [pdf, epub]`);
- `just` recipes: `docs-check` keeps its name but its body becomes `snowball check`;
  a new `docs-build` runs `snowball build -o dist/docs` (output dir is a flag, not
  config — verified against snowball 0.1.3);
- output goes to `dist/docs/` — already gitignored (`/dist/` is goreleaser's dir and
  `just clean` removes it); artifacts are built, not committed;
- docs registration: rewrite `docs/README.adoc`'s "Validating" section (its "There is no
  PDF/EPUB build — the sources are the deliverable" claim becomes false) and add a
  "Building" section covering the snowball toolchain.

Confirmed decisions (from the open questions filed with the ticket):

1. **Replace, don't duplicate.** `just docs-check`'s body becomes `snowball check`; the
   repo-local `Gemfile`/`Gemfile.lock` are deleted (`snowball setup` installs its own pinned
   gem set into snowball's cache and points `BUNDLE_GEMFILE` there — verified in snowball's
   `internal/toolchain`). Because the recipe *name* is unchanged, `pickle.toml`'s
   `docs = "just docs-check"` and the `AGENTS.md` marker block need **no** self-host hand
   edits.
2. **Revision stamping: adopt `git-describe`.** Stamps `revnumber`/`revdate` into rendered
   artifacts only; `docs/attributes.adoc` stays static, so T-047 decision 5 is untouched at
   the source level.
3. **PDF theme: yes** — copied from `~/Projects/personal/translator/ai-sdlc`'s
   `docs/pdf-theme/ai-sdlc-theme.yml` (a small `extends: default` tweak) into
   `docs/pdf-theme/pickle-theme.yml`.
4. **Toolchain docs live in `docs/README.adoc` only.** `installation.adoc`/`cli-reference.adoc`
   document the `pickle` binary for end users and are untouched. The manual has no diagrams,
   so the mermaid/Chrome side of snowball's toolchain stays out of scope.

Soft couplings: **T-047** (done — established the manual, the `docs-check` recipe, the
`Gemfile`, and the `docs = "just docs-check"` registration this ticket may revisit); **T-019**
(its remaining PLAN.md item is untouched, but if decision 1 changes the `docs` command the
same self-host hand-edit discipline applies). This is repo tooling for pickle's own docs — the
`pickle` binary's behaviour does not change.

## Implementation Plan

### 0. Feature branch (mandatory)

Before any change, create a feature branch inside the target child-project's repo (the
`pickle` child is this repo, at `.`):

```
git checkout main
git checkout -b feat/T-048-render-manual-pdf-epub-snowball
```

Do all work on this branch, committing locally as you go. **Never push or open a merge
request without explicit user approval**; end with a summary and a suggested commit message
(see Finish, below).

### Prerequisite gate (hard)

- Clean tree on `main`; no `depends-on:` (T-047 is done and merged — commit `7a0995d`).
- The `snowball` binary is on `PATH` (`brew install codcod/taps/snowball`; v0.1.3 verified)
  and its toolchain is installed: run `snowball setup` once, then `snowball doctor` must
  report `toolchain ok`. If `doctor` fails, stop and report — do not hand-install gems.

### Confirmed design decisions (do not deviate without asking)

1. `just docs-check` keeps its **name** and registered role (`pickle.toml`
   `docs = "just docs-check"`); only its **body** changes to `snowball check`. No edits to
   `pickle.toml` or the `AGENTS.md` marker block.
2. `Gemfile` and `Gemfile.lock` are **deleted** — snowball owns the gem pinning.
3. Revision stamping: `revision: from: git-describe` (snowball's default; state it
   explicitly in `snowball.yaml` to record the decision).
4. PDF theme: `docs/pdf-theme/pickle-theme.yml`, content copied verbatim from the ai-sdlc
   theme (see Task 1). No further styling work in this ticket.
5. Output: `dist/docs/` via `snowball build -o dist/docs`. Nothing under `dist/` is ever
   committed (already gitignored).
6. No mermaid/diagram configuration — the manual has no diagrams; rely on snowball's
   defaults and do not document the diagram toolchain.
7. The `pickle` binary's behaviour does not change: no Go code edits, no new tests.

### Tasks

#### Task 1 — PDF theme

Create `docs/pdf-theme/pickle-theme.yml` with exactly this content (copied from
`~/Projects/personal/translator/ai-sdlc/docs/pdf-theme/ai-sdlc-theme.yml`, renamed for
pickle, with a `#` header comment citing the origin):

```yaml
# pickle PDF theme (asciidoctor-pdf). Origin: ai-sdlc's ai-sdlc-theme.yml (T-048).
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

(The `-theme.yml` suffix is required — snowball splits the path into
`pdf-themesdir=docs/pdf-theme` + `pdf-theme=pickle`.)

#### Task 2 — `snowball.yaml`

Create `snowball.yaml` at the repo root:

```yaml
# snowball config — renders the user manual to PDF/EPUB (T-048).
# Failure levels default to pdf: WARN, epub: ERROR, check: WARN.
books:
  - src: docs/user-manual.adoc
    out: pickle-user-manual
theme: docs/pdf-theme/pickle-theme.yml
attributes: docs/attributes.adoc   # informational — the master includes it itself
formats: [pdf, epub]
revision:
  from: git-describe
```

Omit `mermaid:` and `failure-level:` — snowball's defaults (verified in its
`internal/config`) are exactly what we want.

#### Task 3 — justfile recipes

In `justfile`:

- Change `docs-check`'s body from
  `bundle exec asciidoctor --failure-level=WARN -o /dev/null docs/user-manual.adoc` to
  `snowball check`, and update its comment (still "validate the AsciiDoc manual", now via
  snowball).
- Add a `docs-build` recipe next to it:

  ```
  # Render the user manual to PDF + EPUB into dist/docs/ (never committed)
  docs-build:
      snowball build -o dist/docs
  ```

#### Task 4 — delete the repo Gemfile

`git rm Gemfile Gemfile.lock`. Grep the repo for remaining `bundle`/`Gemfile` references
(`rg -i 'bundle|gemfile' --glob '!tickets/**'`) — after Task 5 the only hits should be in
snowball's own docs, i.e. none in this repo.

#### Task 5 — docs registration (`docs/README.adoc`)

Rewrite the `== Validating` section: drop the "There is no PDF/EPUB build" paragraph and the
`bundle install` snippet; document `just docs-check` (→ `snowball check`, render-and-discard,
fails on warnings). Add a `== Building` section: `just docs-build` renders
`dist/docs/pickle-user-manual.{pdf,epub}`; one-time toolchain setup is
`brew install codcod/taps/snowball && snowball setup` (verify with `snowball doctor`);
artifacts are stamped with `git describe` as the revision and are never committed.

### Acceptance test

From the repo root on the feature branch:

```sh
snowball doctor                       # toolchain ok
just docs-check                       # snowball check — exits 0
just docs-build                       # renders both artifacts
test -s dist/docs/pickle-user-manual.pdf && test -s dist/docs/pickle-user-manual.epub
git check-ignore dist/docs/pickle-user-manual.pdf   # exits 0 — artifacts ignored
test ! -f Gemfile && test ! -f Gemfile.lock         # gem pinning removed
just build && just test && just lint                # child commands stay green
```

Negative check: introduce a temporary broken **include** in the book master, confirm
`just docs-check` fails, revert. (Amended during implementation: a broken *xref* does not
trip plain asciidoctor at `--failure-level=WARN` — unresolved xrefs log below WARN — and the
old Gemfile-based `docs-check` had the identical blind spot, so this is behaviour-preserving,
not a regression. Broken includes are ERROR-level and fail as expected.)

### Docs update (mandatory when user-facing)

Task 5 **is** the docs step (`docs/README.adoc`). No user-manual content changes: the
manual documents the `pickle` binary, which is unchanged.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (Task 5).
3. Write a summary (files touched, decisions honoured, anything deferred).
4. Suggested commit message:

   ```
   build(docs): render the user manual to PDF/EPUB with snowball (T-048)

   Replace the Gemfile-pinned asciidoctor docs-check with `snowball check`,
   add `docs-build` (snowball build -> dist/docs/), a snowball.yaml, and a
   minimal PDF theme; document the toolchain in docs/README.adoc.
   ```

5. Commit locally on the ticket branch; do **not** push or open an MR without user
   approval. `pickle ticket move T-048 in-review --reason "acceptance green"` and hand back.

## Review

2026-07-27 — review on `feat/T-048-render-manual-pdf-epub-snowball` (single commit `18e1cd2`), per
the review protocol. No layered addenda configured.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — ran `docs_readability` on `docs/README.adoc` +
  `README.md`; all four suggestions target prose *outside* this ticket's diff (README intro,
  Install section, manual-index book blurb), so none applied; presented to the user for
  separate consideration
- [x] Findings recorded with severity **and** disposition per rules §5; summary line present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated / board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

**Implementation audit** (step 2) — all met, verified on the actual tree:

- Task 1 `docs/pdf-theme/pickle-theme.yml` — **met**: content verbatim per plan, origin comment present.
- Task 2 `snowball.yaml` — **met**: matches plan exactly (book, theme, attributes note, formats, `revision: from: git-describe`; no `mermaid:`/`failure-level:`).
- Task 3 justfile — **met**: `docs-check` body is `snowball check` (name/registration unchanged → `pickle.toml` and `AGENTS.md` marker untouched, decision 1 honoured); `docs-build` added.
- Task 4 Gemfile removal — **met**: `Gemfile`/`Gemfile.lock` gone; `rg -i 'bundle|gemfile'` finds only the English word "Bundled" in `skill/SKILL.md` prose (grep false positive, not a gem reference).
- Task 5 `docs/README.adoc` — **met**: "Validating" rewritten (no PDF/EPUB-denial, no `bundle install`), "Building" section added (brew install, `snowball setup`/`doctor`, `just docs-build`, git-describe stamping, never-committed note); Layout tree includes `pdf-theme/`. `README.md` docs line updated (per the recorded plan amendment).
- Acceptance test — **re-run verbatim, green**: `snowball doctor` → toolchain ok; `just docs-check` → exit 0; `just docs-build` → both artifacts non-empty in `dist/docs/`; `git check-ignore` → 0; Gemfiles absent; `just build && just test && just lint` → green. Negative check re-run: temporary broken include → `docs-check` fails with asciidoctor ERROR, exit 1; reverted clean.
- Confirmed decisions 1–7 — **met** (no Go code changes; diff touches only the 7 planned files).

**Quality/consistency/docs audits** (steps 3–4a) — clean. Whole-tree sweep: no stale
`bundle`/`asciidoctor`-CLI references remain; `pickle.toml` `docs = "just docs-check"` and the
`AGENTS.md` marker block still true as-is; `dist/` gitignored and removed by `just clean`;
revision stamping verified in the rendered PDF ("Version 18e1cd2, 27 July 2026") and EPUB
metadata. Docs build (`just docs-check`) clean.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | noted | With no git tags in the repo, `git describe` falls back to the bare short commit hash, so the rendered title page reads "Version 18e1cd2" — technically correct but cosmetically odd until a release tag exists in the working clone | `pdftotext` of `dist/docs/pickle-user-manual.pdf` p.1: "Version 18e1cd2, 27 July 2026" | None needed — self-resolves once a `vX.Y.Z` tag is present (goreleaser tags exist upstream); exactly the pinned decision 3 behaviour |

**Disposition summary:** 1 finding — 1 noted (F1); 0 blocking, 0 fixed inline, 0 folded, 0 new
tickets.

**Verdict:** no blocking findings → DONE.

## History

- 2026-07-26 — created (TO DO). source: chat (user request: render the manual to PDF/EPUB with the installed `snowball` tool; supersedes T-047's content-only scoping decision by explicit user choice)
- 2026-07-27 — TO DO → READY: plan complete (refined 2026-07-26: 1B replace docs-check body + drop Gemfile, git-describe revision, ai-sdlc theme, README-only docs)
- 2026-07-27 — READY → IN DEVELOPMENT: picked up; applicability gate clean, branch feat/T-048-render-manual-pdf-epub-snowball
- 2026-07-27 — plan amendment (inline disposition): acceptance negative check corrected from broken xref to broken include — unresolved xrefs never tripped WARN failure-level in the old recipe either; also README.md's `bundle install` reference updated (caught by Task 4's grep)
- 2026-07-27 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-27 — IN REVIEW → DONE: review clean: 0 blocking, 1 noted (F1 git-describe bare-hash cosmetics); acceptance re-run green
- 2026-07-27 — merged to main (commit 6606b63, fast-forward after rebase; user-approved)
