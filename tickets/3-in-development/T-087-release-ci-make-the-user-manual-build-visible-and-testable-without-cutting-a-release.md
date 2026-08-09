---
id: T-087
title: release CI: make the user-manual build visible and testable without cutting a release
project: pickle
depends-on: []
spawned-by: [T-086]
impact: medium
complexity: medium
cost: M
---

# T-087 — release CI: make the user-manual build visible and testable without cutting a release

## Outcome

After this ships, a release either carries its PDF/EPUB manual or the run is annotated with the
tool that was missing — no more silently manual-less releases — and you can exercise the whole
snowball toolchain from a branch, or against an older tag to backfill its manual, without cutting
a release.

## Description

Surfaced verifying T-086's fix live (`gh workflow run release.yml --ref
feat/T-086-stale-runner-homebrew-install-steps -f tag=v0.3.0`,
[run 31222772467](https://github.com/codcod/pickle/actions/runs/31222772467/job/93010687635)):
`snowball setup` failed with

```
snowball: installing gems into /home/runner/.cache/snowball/toolchain
Error: bundle install: exec: "bundle": executable file not found in $PATH
```

**This ticket was re-scoped at refinement.** The `bundle`-resolution defect is snowball's, not
pickle's, and is tracked upstream (see *Upstream: SNOW-002* below). What remains here is
pickle's own share: the release step has been failing **invisibly**, and there is **no way to
exercise the docs toolchain without cutting a release**.

### The evidence, and why the original two hypotheses are both wrong

This ticket originally guessed at (1) a keg-only `ruby` on Linuxbrew and (2) a `ruby` revision
whose bundler is not vendored. Both are disproven:

- `formulae.brew.sh/api/formula/ruby.json` reports `keg_only: false`, and the formula carries no
  `keg_only` line at all — so `brew link` does run for it.
- The actual `x86_64_linux` bottle (`ruby--4.0.6…bottle.3`, downloaded and listed) ships
  `bin/bundle`, `bin/bundler` **and** `lib/ruby/gems/4.0.0/gems/bundler-4.0.16/`.

What the four CI runs actually show is a correlation with ruby's **`post_install`**:

| run | date | ruby `post_install` | `bundle` | manual attached |
|---|---|---|---|---|
| [v0.2.1](https://github.com/codcod/pickle/actions/runs/30476423673) | 2026-07-29 17:40 | aborted (`unknown install step: run`) | worked | ✅ |
| [v0.2.2](https://github.com/codcod/pickle/actions/runs/30490798326) | 2026-07-29 21:02 | aborted (attempts 1–2) | worked | ❌ npm: `UNABLE_TO_GET_ISSUER_CERT_LOCALLY` |
| [v0.3.0](https://github.com/codcod/pickle/actions/runs/31221987955) | 2026-08-07 21:56 | n/a (`unknown install step: remove`) | never reached | ❌ (T-086's defect) |
| [T-086 branch](https://github.com/codcod/pickle/actions/runs/31222772467) | 2026-08-07 22:09 | **completed** | **not found** | ❌ this ticket |

`bundle` is present in exactly the runs where ruby's `post_install` **aborted**, and absent in
the one run where it **completed** — and ruby's `post_install` is precisely the code that deletes
bundler from the prefix gem dir:

```ruby
rm(%W[#{rubygems_bindir}/bundle #{rubygems_bindir}/bundler].select { |file| File.exist?(file) })
rm_r(Dir[HOMEBREW_PREFIX/"lib/ruby/gems/#{api_version}/gems/bundler-*"])
```

So **T-086 did not break this: it un-broke ruby's install, and the earlier manual builds were
succeeding only because ruby was installing wrong.** Whether the end state is a dangling
`$HOMEBREW_PREFIX/bin/bundle` or a de-activated bundler gem is *not* determinable from the logs,
because no run ever printed the ground truth (`ls -l "$(brew --prefix)/bin/bundle"`,
`command -v -a bundle`, `snowball doctor`). That missing evidence is itself part of what this
ticket fixes.

### Three failures in three releases, none of them visible

The step is `continue-on-error: true` by design (RELEASING.md: "a broken manual does not block
publishing the binaries") and the attach step prints `No user manual built; skipping attach.`
and exits 0. The result: **v0.2.2 and v0.3.0 both shipped with no manual** (confirmed via
`gh release view <tag> --json assets`; only v0.2.1 has one), each for a *different* reason —
npm's CA-certificate failure, the stale-brew install-step DSL, and now bundler — and nobody
noticed for ten days.

Compounding it, `ci.yml` never touches snowball, so the toolchain is exercised **only** by a
release run. Every fix therefore costs a tagged release, and each one has so far revealed the
next failure one step further along.

### Upstream: SNOW-002 (soft coupling, not a dependency)

The bundler defect belongs in snowball, by snowball's own charter —
`internal/toolchain/toolchain.go`'s package doc says it "owns language-level deps only — OS
packages … are the environment's responsibility", and bundler is a gem. Yet `Required` lists
Bundler as the environment's job, `Setup()` calls `run(dir, "bundle", "install")` on a bare
`exec.LookPath` with no preflight, and `Doctor()` — which would report
`MISS Bundler — gem install bundler, then snowball setup` — is never consulted by `Setup()`.
snowball's formula (`depends_on "ruby"`) plus its caveat ("Run `snowball setup` once") promises
that `brew install codcod/tap/snowball && snowball setup` works on a clean box; on Linuxbrew it
no longer does.

snowball is a registered brine child-project **in the `unity` workspace** (`ticket_prefix =
SNOW`), so the root fix is filed there as **SNOW-002**. Brine's one id namespace spans a single
overarching project, so that link cannot be `depends-on:` — it is a **soft coupling**, recorded
here in prose (rules §3). Nothing in this ticket blocks on it: the shim below is deliberately
temporary, and the visibility and test-loop work stands regardless.

### Deliberately out of scope

- **Pinning `codcod/tap/snowball`** — a tap formula has no per-version variant to pin to, so
  there is no cheap mechanism; noted, not attempted.
- **The npm CA failure as an upstream concern** — setting `NODE_EXTRA_CA_CERTS` is an OS-level
  environment fix, which snowball's charter explicitly leaves to the caller, so it lands here.

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/T-087-manual-build-visible-and-testable
```

(`pickle`'s child path is `.` — this repo itself. Root-path child: tidy WIP commits into atomic
ones before presenting them, and prefer keeping that history over squashing — rules §0.)

### Prerequisite gate (hard)

1. T-086's `brew update --quiet` is on `main` (commit `ca6a46a`, merged via PR #22) — this plan
   edits the step it created. Verify with
   `git show main:.github/workflows/release.yml | grep -n 'brew update'`.
2. Clean tree; `just build`, `just test`, `just lint`, `just docs-check` green before starting.
3. `snowball` installed locally (`snowball doctor` → `toolchain ok`) — acceptance test item 4
   runs the new script on the dev box.

### Confirmed design decisions (do not deviate without asking)

1. **The root fix stays upstream (SNOW-002).** pickle carries the bundler workaround only as an
   explicitly **temporary shim**, commented with its removal condition — a snowball release
   whose `setup` bootstraps bundler **and makes it resolvable on the caller's `PATH`, not merely
   inside `Setup()`'s own subprocess env** (SNOW-002's sketch may only do the latter, in which
   case Decision 8's `bundle --version` pre-check still needs this shim even after SNOW-002
   ships) — and a pointer to SNOW-002. Do not attempt a "proper" bundler resolution here. Note:
   "SNOW-002 implemented in parallel" does not shorten this — `codcod/tap/snowball` is a
   Homebrew tap formula pinned to one tagged release (still 0.2.1); nothing changes what
   `brew install` resolves until a new snowball tag is cut, released, and the tap formula is
   bumped, none of which is scheduled here. Task 1 step 5's shim is already naturally
   idempotent against that eventual fix: once `bundle` resolves on its own, its "if not found"
   branch short-circuits to a no-op.
2. **One source of truth for the toolchain steps:** a new `.github/scripts/build-manual.sh`,
   called by both workflows. No duplicated YAML block — that block has been edited five times
   already and must not exist twice.
3. **The smoke workflow triggers on path-filtered `push` (any branch) *and* `workflow_dispatch`.**
   A `workflow_dispatch`-only workflow cannot be dispatched until it lives on the default branch,
   which would make this very ticket's acceptance test unrunnable before merge.
4. **Backfilling an older tag's manual goes through the smoke workflow**, which checks out the
   script from the workflow's ref and the docs from the target ref — **not** by dispatching
   `release.yml` against an old tag: that tag's tree predates the script, and re-running the
   release pipeline re-uploads published assets (goreleaser `422 already_exists`, as seen in run
   31222772467).
5. **The release step keeps `continue-on-error`** (RELEASING.md's soft-fail contract is
   deliberate) but must **annotate** a miss: `::warning::` plus a `$GITHUB_STEP_SUMMARY` line, so
   a manual-less release is visible in the run without reading 600 log lines.
6. **`NODE_EXTRA_CA_CERTS` is set to the runner's system CA bundle** (`/etc/ssl/certs/ca-certificates.crt`
   when it exists). v0.2.2's failure was npm's `UNABLE_TO_GET_ISSUER_CERT_LOCALLY` with brew's
   node — deterministic, not a flake, and OS-level, which snowball's charter leaves to the caller
   (see *Deliberately out of scope* in the Description).
7. **The script must be runnable on a developer's macOS box**, not just on the Linux runner:
   load `brew shellenv` only when `brew` is not already on `PATH`, and never hardcode
   `/home/linuxbrew/…` outside that fallback. This is what makes acceptance item 4 a real local
   check and lets a reviewer re-run it.
8. **`snowball doctor` is the post-setup assertion** (`toolchain ok`), and an explicit
   `ruby --version && gem --version && bundle --version` is the pre-`setup` assertion. Do **not**
   assert `doctor` *before* `setup` — it legitimately fails there (gems and mermaid-cli are not
   installed yet); print it as diagnostics only.

### Tasks

#### Task 1 — `.github/scripts/build-manual.sh` (new)

`#!/usr/bin/env bash` + `set -euo pipefail`. Signature: `build-manual.sh <src-dir> <out-dir>`.
In order:

1. If `brew` is not on `PATH` and `/home/linuxbrew/.linuxbrew/bin/brew` exists,
   `eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"` (decision 7).
2. `brew update --quiet` (carried over from T-086, with its comment).
3. The existing 3-attempt `brew install codcod/tap/snowball` retry loop, with its
   broken-pipe comment. Skip the install when `snowball` is already on `PATH` (local runs).
4. **Diagnostics, always printed, never fatal:** `brew --prefix ruby`, `command -v -a ruby gem bundle || true`,
   `ls -l "$(brew --prefix)/bin/bundle" "$(brew --prefix)/bin/bundler" || true`,
   `snowball doctor || true`. This is the ground truth no previous run recorded.
5. **TEMPORARY bundler shim** (decision 1 — comment it as such, cite SNOW-002 and the removal
   condition): prepend `"$(brew --prefix ruby)/bin"` to `PATH`; if `bundle` is still not found,
   `gem install --no-document bundler` and prepend `"$(gem environment gemdir)/bin"`.
6. `export NODE_EXTRA_CA_CERTS=/etc/ssl/certs/ca-certificates.crt` when that file exists
   (decision 6).
7. Pre-`setup` assertions: `ruby --version`, `gem --version`, `bundle --version` (unguarded — a
   failure here must stop the script with the precondition named).
8. `cd "<src-dir>"`, `snowball setup`, then `snowball doctor` (must succeed — decision 8), then
   `snowball build -o "<out-dir>"`.

#### Task 2 — `release.yml`: call the script

Replace the inline `run:` block of **"Build user manual (PDF + EPUB)"** with
`.github/scripts/build-manual.sh . "$RUNNER_TEMP/manual"`, keeping the step's `env:` block and
`continue-on-error: true` unchanged. Move the block's long rationale comments into the script (one
place), leaving in the workflow only why the step exists and why it soft-fails, with a pointer to
the script.

#### Task 3 — `release.yml`: make a miss visible

In **"Attach user manual to release"**, replace the silent
`echo "No user manual built; skipping attach."; exit 0` with an `::warning::` annotation *and* a
`$GITHUB_STEP_SUMMARY` line naming the tag whose release has no manual; still `exit 0`
(decision 5).

#### Task 4 — `.github/workflows/manual-smoke.yml` (new)

```
on:
  push:
    paths: ['docs/**', 'snowball.yaml', '.github/scripts/build-manual.sh', '.github/workflows/manual-smoke.yml']
  workflow_dispatch:
    inputs:
      ref: { description: 'Tag/ref whose docs to render (default: this workflow''s ref)', required: false }
```

Steps: `actions/checkout@v7` (the workflow's own ref — this is where the script comes from); when
`inputs.ref` is non-empty, a second `actions/checkout@v7` with `ref:` and `path: src` (plus
`fetch-depth: 0`, since `snowball.yaml` sets `revision.from: git-describe`); run
`.github/scripts/build-manual.sh <. or src> "$RUNNER_TEMP/manual"`; `actions/upload-artifact@v4`
with the two files. No `continue-on-error` here — this workflow's whole job is to fail loudly.

#### Task 5 — docs

`RELEASING.md`: a new short section covering (a) validating the manual toolchain without cutting
a release (push-triggered + `gh workflow run manual-smoke.yml`), (b) backfilling an older tag's
manual (dispatch with `ref=<tag>`, then `gh release upload <tag> …`), and (c) the temporary
bundler shim with its removal condition and the SNOW-002 pointer. Update the existing
`workflow_dispatch` re-run paragraph to warn that re-running `release.yml` on a fully published
tag hits `422 already_exists` on the binaries.

`CHANGELOG.md` under `## [Unreleased]` → `### Fixed`: releases since v0.2.1 shipped without the
PDF/EPUB manual; the toolchain is now verified by its own workflow and a miss is annotated.

### Acceptance test

Run from the repo root on the feature branch:

1. `bash -n .github/scripts/build-manual.sh` — no syntax errors. (`shellcheck` if installed.)
2. Both workflows are valid YAML:
   `ruby -ryaml -e 'YAML.load_file(".github/workflows/release.yml"); YAML.load_file(".github/workflows/manual-smoke.yml"); puts "ok"'`
3. `just build && just test && just lint && just docs-check` — all clean.
4. **Local end-to-end** (decision 7): `.github/scripts/build-manual.sh . /tmp/manual-check` exits
   0 and leaves `/tmp/manual-check/pickle-user-manual.pdf` and `.epub`; the diagnostics block and
   `snowball doctor` → `toolchain ok` appear in its output.
5. **CI end-to-end** — needs approval to push (publish-gated), so request it as part of Finish
   rather than deferring it: pushing the branch touches
   `.github/{scripts/build-manual.sh,workflows/manual-smoke.yml}` and therefore triggers
   `manual-smoke` (decision 3). Confirm with `gh run watch`: the run is **green**, the artifact
   carries both files, and the log shows the diagnostics block, `bundle --version`, and
   `snowball doctor` → `toolchain ok`.
   **Contingency:** if it fails on a *new* snowball precondition (not `bundle`), that is upstream —
   record the log excerpt in this ticket's `## History`, add it to SNOW-002's theme, and treat
   artifact production as documented-blocked for the review; Tasks 1–5 still stand and item 4
   still has to pass locally.
6. **Not testable pre-merge:** the `release.yml` path itself (it needs a tag push). It runs the
   identical script, so item 5 covers the toolchain; the next real release confirms the wiring.

### Docs update

Task 5 (`RELEASING.md`, `CHANGELOG.md`). No user-facing CLI surface changes, so
`docs/user-manual.adoc` and `docs/cli-reference.adoc` are untouched; `just docs-check` still has
to pass.

### Finish

1. Acceptance items 1–4 green locally; `just build`/`test`/`lint`/`docs-check` clean.
2. Tidy the WIP commits into atomic ones (root-path child, rules §0) — expect roughly
   `ci(release): …` for Tasks 1–4 and `docs(releasing): …` for Task 5.
3. Write the summary; suggest the commit message, e.g.
   `ci(release): verify and annotate the user-manual build (T-087)`.
4. Present for approval, **stating that pushing is required to run acceptance item 5**. Before
   pushing, `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`
   must print nothing (rules §0). Then push, watch `manual-smoke`, open the MR — the human merges.
5. **Post-merge follow-through** (record in `## History`, do not leave implicit): once on `main`,
   `gh workflow run manual-smoke.yml -f ref=v0.3.0` and `-f ref=v0.2.2`, then
   `gh release upload <tag> pickle-user-manual-<version>.pdf .epub` for each, restoring the two
   releases' missing assets.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-08 — created (TO DO). source: pickle ticket new
- 2026-08-09 — re-scoped at refinement: the `bundle` defect is snowball's (its `Setup()` does a
  bare PATH lookup, and bundler is language-level, which its own package doc calls snowball's to
  own), filed upstream as SNOW-002 on the `unity` workspace's board; this ticket keeps pickle's
  share — invisible failures, no test loop short of a release, two releases missing assets — and
  carries the bundler workaround only as a marked temporary shim. Retitled to match. Both
  hypotheses in the original Description are disproven (`ruby` is not keg-only; the Linux bottle
  does ship `bin/bundle`); the evidence table in the Description replaces them.
- 2026-08-09 — TO DO → READY: plan complete
- 2026-08-09 — applicability gate (pickup, "assuming SNOW-002 in parallel"): 0 blocking; amended
  Decision 1's shim-removal condition inline (a tap-formula bump is a separate, unscheduled step
  from SNOW-002 landing in snowball's repo, and SNOW-002's sketch may only fix bundler inside
  `Setup()`'s own subprocess env, not the caller's PATH) — 3 other findings noted, no plan/task
  change otherwise. Proceeding to IN DEVELOPMENT.
- 2026-08-09 — acceptance test green: items 1–4 pass locally (shellcheck, YAML, build/test/
  lint/docs-check, and a local end-to-end run producing both manual files); item 5 (CI,
  `manual-smoke` run 31318708209) is green — its diagnostics confirm the fresh-runner `bundle`
  miss this ticket targets was real (`MISS Bundler`, `type: bundle: not found`) and that the
  temporary shim bootstraps it to a completed build. PR #25 opened against `main`
  (feat/T-087-manual-build-visible-and-testable, 3 commits, kept unsquashed per the root-path
  default).
- 2026-08-09 — READY → IN DEVELOPMENT: picked up
