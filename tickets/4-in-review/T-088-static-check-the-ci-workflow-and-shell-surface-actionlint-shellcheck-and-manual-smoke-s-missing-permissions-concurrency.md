---
id: T-088
title: static-check the CI workflow and shell surface: actionlint + shellcheck, and manual-smoke's missing permissions/concurrency
project: pickle
depends-on: []
spawned-by: [T-087]
impact: low
complexity: low
cost: M
---

# T-088 — static-check the CI workflow and shell surface: actionlint + shellcheck, and manual-smoke's missing permissions/concurrency

## Outcome

After this ships, a malformed workflow or a sloppy shell script — including the pre-commit hook
shim `pickle` writes into every user's `.git/hooks/` — is caught by `just lint` and by `ci.yml`
on every pull request, instead of by whoever next reads the YAML; and all three workflows
declare a least-privilege token and a concurrency group, so superseded runs cancel where that is
safe and a release run never does.

## Description

Surfaced by T-087's review. That ticket added the repo's first CI shell script
(`.github/scripts/build-manual.sh`, 120 lines) and a third workflow
(`.github/workflows/manual-smoke.yml`), and nothing in the project checks either one:

- **No static analysis for the CI surface.** `just lint` is `go vet` + `gofmt`, and `ci.yml`
  runs `go vet`, `gofmt`, `go test`, `go build` and `goreleaser check` — nothing for shell or
  for workflow YAML. T-087's acceptance test had to say "`shellcheck` if installed", which makes
  the check a property of the developer's laptop rather than of the repo.
- **It would have caught a real finding.** T-087's review found
  `manual-smoke.yml` interpolating `${{ inputs.ref }}` directly into a `run:` body — the standard
  Actions script-injection hole. `actionlint` flags exactly that (and `zizmor` more thoroughly).
  The finding was fixed inline during the review; the *class* of finding is what this ticket
  removes.
- **`manual-smoke` is missing two workflow hygiene declarations.** It has no `concurrency:`, so
  a burst of `docs/**` pushes queues several ~4-minute Homebrew installs against each other.
  (Its `permissions: contents: read` was added inline during T-087's review; the audit that
  should have required it is what is missing.)

Scope sketch (to pin down at refinement): add `actionlint` and `shellcheck` to `just lint` and to
`ci.yml`, decide whether a missing local binary is a hard failure or a skip-with-warning, add
`concurrency:` to `manual-smoke.yml`, and audit the other two workflows for the same hygiene
(`ci.yml` and `release.yml` both lack `concurrency:`; `release.yml`'s `permissions: contents:
write` is deliberate).

**Measured at refinement (2026-08-13), correcting the sketch above.** `ci.yml` has **no
`permissions:` block at all**, so it runs on the repository's default token rather than a
least-privilege one — a gap the sketch missed by only naming `manual-smoke`. Confirmed as
sketched: no workflow declares `concurrency:`; `manual-smoke.yml` already carries
`permissions: contents: read` (added inline during T-087's review); `release.yml`'s
`contents: write` is deliberate and stays.

**Also found at refinement, and now in scope: `hook.Shim()`.** `internal/hook/hook.go:78–92`
returns `/bin/sh` source assembled from a Go string literal, which `pickle hooks install` writes
into every user's `.git/hooks/pre-commit`. It is the project's highest-stakes shell — it runs on
someone else's machine, on every commit, and its exit-code handling is load-bearing (T-057
decision 3) — and nothing checks it, because it is a string rather than a `.sh` file that
`shellcheck **/*.sh` would find. That is precisely the gap this ticket exists to close, so it is
covered here rather than deferred.

**`zizmor` is out of scope — note-and-closed.** The Description names it as "more thorough" than
actionlint, and it is; but it needs a Python/uv toolchain this repo does not otherwise have, for
a marginal gain over actionlint on a three-workflow surface. Revisit if the workflow surface
grows or a finding slips past actionlint.

Soft coupling, no `depends-on:`: T-087's branch must merge first, or the two workflow files this
ticket lints will not yet exist on `main`. **Satisfied — verified at refinement:
`.github/workflows/manual-smoke.yml` and `.github/scripts/build-manual.sh` are both on `main`.**

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-088-ci-surface-lint`, created in the `pickle` child-project's repo (path `.`) before any
change. Local WIP commits are fine; **no push and no MR without explicit user approval**, and
merging is always the human's. Root-path child, so tidy the WIP commits by interactive rebase
into a small number of atomic commits and **keep that history** on merge (rules §0).

Bookkeeping (this ticket file + `BOARD.md`) is committed on `main`, never on this branch.

**Note on verification:** most of this ticket's product only truly runs on GitHub. The
acceptance test is therefore split into a local half and a "first push proves it" half — read it
before starting, since it shapes what to commit when.

### Prerequisite gate (hard)

T-087 merged — **verified at refinement**: `.github/workflows/manual-smoke.yml` and
`.github/scripts/build-manual.sh` are on `main`. Nothing else is required.

Locally you will want `actionlint` and `shellcheck` installed (`brew install actionlint
shellcheck`) — not a hard gate (decision 1 makes them optional for `just lint`), but you cannot
validate the work without them.

### Confirmed design decisions (do not deviate without asking)

1. **`just lint` skips a missing tool with a warning; CI hard-fails.** `just lint` must stay
   green on a bare checkout with only the Go toolchain — that is a property of this repo, not an
   accident. So each new check runs when its binary is on PATH and prints a one-line warning
   naming the tool when it is not. In CI both are installed, so absence cannot occur and any
   finding fails the job. A warning must never be swallowed: do not `|| true` the whole line.
2. **CI installs actionlint with `go install`, at a pinned tag.** The repo's only toolchain is
   Go; `actions/setup-go` is already used by both existing jobs. Use
   `go install github.com/rhysd/actionlint/cmd/actionlint@<tag>` with an explicit version tag
   (**not** `@latest` — a floating lint version turns an unrelated PR red). `v1.7.7` was the
   current release at refinement; use the then-current tag and record it in a comment next to
   the step. No `curl | bash` installer, and no Docker action.
3. **`shellcheck` comes from the `ubuntu-latest` image, but its presence is asserted.** It is
   preinstalled today. Run `shellcheck --version` as its own step **before** the check, so a
   future runner-image change fails the job loudly instead of silently skipping the lint — a
   check that can quietly stop running is the failure mode this whole ticket exists to remove.
4. **A new third job, `ci-surface`, not extra steps in `build-test`.** It is a distinct concern
   with a distinct toolchain, and as its own job it runs in parallel rather than delaying the
   Go signal. Do not fold it into `build-test`.
5. **actionlint runs shellcheck on `run:` bodies automatically** when shellcheck is on PATH —
   that is the mechanism that catches T-087's class of finding, so the two tools must be
   available *together* in the `ci-surface` job, not split across jobs.
6. **The three workflows get deliberately different concurrency settings** (see Task 3). In
   particular `release.yml` gets `cancel-in-progress: false`: a half-cancelled release is worse
   than a queued one. Do not "make them consistent".
7. **`hook.Shim()` is checked by a Go test, not by the justfile.** The shim is a string, not a
   file, so the only place that can lint it is a test that materialises it. `internal/hook`'s
   test writes it to `t.TempDir()` and runs `shellcheck -s sh`, `t.Skip`ping when shellcheck is
   absent (so `just test` stays green on a bare checkout — same principle as decision 1) and
   running for real in CI.
8. **`-s sh`, not the default.** The shim's shebang is `#!/bin/sh` and its portability is the
   point (it runs on whatever `/bin/sh` a user has). Letting shellcheck infer bash would permit
   bashisms the shim must not contain.
9. **Fix what the tools report; suppress only with a written reason.** Neither tool has been run
   on this repo, so expect findings in `build-manual.sh` and possibly the workflows. Fix them.
   Where a rule is genuinely wrong here, disable it narrowly — an inline
   `# shellcheck disable=SCxxxx` with a comment saying why, or a `.github/actionlint.yaml` entry
   — never a blanket exclusion.
10. **`zizmor` is not added.** Note-and-closed at refinement; see the Description.

### Tasks

#### Task 1 — `justfile`: optional-tool lint recipes

Add a recipe and wire it into `lint`:

```just
# vet + gofmt check, plus the CI surface (actionlint/shellcheck) when installed
lint: lint-ci-surface
    go vet ./...
    @test -z "$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

# Static-check the CI surface: workflow YAML + shell scripts. Both tools are
# optional locally (a bare checkout needs only Go) and mandatory in ci.yml's
# ci-surface job, which is where a finding actually blocks a merge (T-088).
lint-ci-surface:
    @if command -v actionlint >/dev/null 2>&1; then actionlint; else echo "warning: actionlint not installed — skipping workflow lint (CI still runs it)"; fi
    @if command -v shellcheck >/dev/null 2>&1; then shellcheck .github/scripts/*.sh; else echo "warning: shellcheck not installed — skipping shell lint (CI still runs it)"; fi
```

Order matters: put `lint-ci-surface` where its output is visible rather than buried — running it
as a dependency (as above) prints its warnings first. If `just`'s dependency ordering makes the
warnings awkward, calling it as the last line of `lint` is equally acceptable; what is not
acceptable is a form where a real finding cannot fail the recipe (decision 1).

#### Task 2 — `.github/workflows/ci.yml`: the `ci-surface` job

Add a third job alongside `build-test` and `goreleaser-check`:

```yaml
  ci-surface:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
          cache: true
      # Pinned, never @latest: a floating lint version turns unrelated PRs red.
      - name: install actionlint
        run: go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
      # Asserted, not assumed: shellcheck ships in the ubuntu-latest image, and if
      # that ever stops being true this must fail loudly rather than silently skip
      # the lint (which is the failure mode T-088 exists to remove).
      - name: shellcheck present
        run: shellcheck --version
      # actionlint shells out to shellcheck for every `run:` body when it is on
      # PATH — that is what catches the ${{ }}-into-run: injection class (T-087).
      - name: actionlint
        run: actionlint
      - name: shellcheck
        run: shellcheck .github/scripts/*.sh
```

#### Task 3 — workflow hygiene, three files, three different settings

- **`.github/workflows/ci.yml`** — add a top-level `permissions: contents: read` (it currently
  has **none**, so it inherits the repository default) and:

  ```yaml
  concurrency:
    group: ${{ github.workflow }}-${{ github.ref }}
    cancel-in-progress: ${{ github.event_name == 'pull_request' }}
  ```

  Cancel superseded *PR* runs; never cancel a `main` push, whose result is the branch's record.

- **`.github/workflows/manual-smoke.yml`** — add:

  ```yaml
  concurrency:
    group: ${{ github.workflow }}-${{ github.ref }}-${{ inputs.ref || 'self' }}
    cancel-in-progress: true
  ```

  The `inputs.ref` key is the point: without it a routine `docs/**` push cancels an in-flight
  `workflow_dispatch` backfill for an older tag, which is a ~4-minute job someone started
  deliberately. `'self'` is the push-event case, where `inputs` is undefined.
  **Risk to check first:** if actionlint rejects `inputs` in a workflow-level `concurrency`
  expression, try `github.event.inputs.ref`; if that also fails, drop the third key and accept
  coarser grouping — record whichever you land on in a comment. Do not disable the rule.

- **`.github/workflows/release.yml`** — add:

  ```yaml
  concurrency:
    group: release-${{ github.ref }}
    cancel-in-progress: false
  ```

  `permissions: contents: write` stays as-is — goreleaser publishes the release.

#### Task 4 — shellcheck the hook shim

In `internal/hook/hook_test.go` (create if absent), add `TestShimPassesShellcheck`:

- `shellcheck -V`-style lookup first: `exec.LookPath("shellcheck")`; on error,
  `t.Skip("shellcheck not installed")` (decision 7).
- Write `Shim()` to `filepath.Join(t.TempDir(), "pre-commit")` and run
  `shellcheck -s sh <path>` (decision 8), failing with the tool's combined output on non-zero.
- The test's doc comment must say *why* this exists: the shim is shell source shipped into every
  user's `.git/hooks/`, its exit-code handling is load-bearing (T-057 decision 3), and being a
  Go string literal is the only reason no linter ever saw it.
- If shellcheck reports something on the current shim, **fix the shim**, and check the fix
  against `Shim()`'s existing doc comment — the exit-code structure it describes must not change
  meaning. If a finding conflicts with that contract, disable that one rule inline with a
  comment (decision 9) rather than rewriting the logic.

#### Task 5 — fix whatever the tools report

Run both tools over the repo and clear the findings (decision 9). Budget for real findings in
`.github/scripts/build-manual.sh` (120 lines, never linted) and for actionlint comments on the
existing `if: ${{ inputs.ref != '' }}` in `manual-smoke.yml`.

### Acceptance test

**Local half** — from the repo root on the feature branch, with both tools installed:

```
brew install actionlint shellcheck   # if not already present
just build && just test && just lint && just docs-check
```

All green, and `just lint`'s output shows actionlint and shellcheck **running** (no warning
lines). Then prove decision 1's skip path is a skip and not a swallow:

```
# absence => warning, exit 0
PATH=/usr/bin:/bin just lint-ci-surface; echo "exit=$?"
```

Expected: two `warning: … not installed …` lines and `exit=0` (adjust the `PATH` if your tools
live in `/usr/bin`). Then prove a real finding still fails:

```
printf '#!/bin/sh\nfoo=$(ls); echo $foo\n' > .github/scripts/_tmp-bad.sh
just lint-ci-surface; echo "exit=$?"
rm .github/scripts/_tmp-bad.sh
```

Expected: shellcheck reports SC2086 (unquoted `$foo`) and `exit` is **non-zero**. A zero here
means the recipe swallows findings — the one outcome decision 1 forbids.

And prove Task 4 is wired both ways:

```
go test ./internal/hook/ -run TestShimPassesShellcheck -v          # PASS
PATH=/usr/bin:/bin go test ./internal/hook/ -run TestShimPassesShellcheck -v   # SKIP, not PASS
```

The second must report `--- SKIP`, not `--- PASS`: a test that silently passes when its tool is
missing is the same silent-skip failure decision 3 rejects.

**CI half** — this is the part only GitHub can run, so it is part of the acceptance test, not an
afterthought. After the first push of the branch (which is publish-gated — so this step happens
at approval time, and its result is reported back before the MR is considered done):

1. The `ci-surface` job appears and is **green**.
2. Its `shellcheck present` step prints a version — confirming decision 3's assumption holds on
   the current image.
3. All three jobs (`build-test`, `goreleaser-check`, `ci-surface`) ran; `ci-surface` started in
   parallel with the other two (decision 4).
4. Push a second commit immediately and confirm the superseded PR run is **cancelled**
   (`ci.yml`'s concurrency, decision 6) — and that the equivalent does *not* happen on `main`.

If the branch is not pushed, say so explicitly in the summary rather than claiming the CI half
passed.

### Docs update (mandatory when user-facing)

No user-facing `pickle` surface changes — nothing in the CLI, the skill payload or the manual is
affected. Two deliberate non-updates, both settled at refinement so the implementer does not
re-open them:

- **No prose doc names `just lint` today** — verified: there is no `CONTRIBUTING.md`, `README.md`
  does not mention it, and `AGENTS.md`'s reference is inside the pickle-owned generated marker
  block (rendered from `pickle.toml`, not hand-edited). So there is no existing section to
  update, and this ticket does not invent one. The discoverability obligation is met **in the
  `justfile` itself**: the `lint-ci-surface` recipe's comment (Task 1) must state that both
  tools are optional locally, that the recipe warns and continues without them, and that CI runs
  them for real — that comment is load-bearing documentation, not decoration, and `just --list`
  surfaces the recipe's doc line.
- **`CHANGELOG.md` gets no entry.** This is CI/repo tooling with no change to the shipped
  binary, and the changelog documents the binary. The decision is recorded here on purpose:
  when `pickle changelog check` later reports T-088 as a candidate, this line is the recorded
  reason it points the reader at (T-093 decision 5 — no exemption mechanism, judgement lives in
  the ticket file).

### Finish (mandatory)

1. Acceptance test's local half green; `just build`, `just test`, `just lint`, `just docs-check`
   clean. CI half reported honestly (see above).
2. Docs updated, or the decision not to recorded.
3. Write a summary of everything done — files touched, the actionlint version pinned, every
   finding the two tools reported and how each was resolved (fixed vs. narrowly suppressed and
   why), and whether `hook.Shim()` needed changing.
4. Suggest a Conventional Commit message, e.g.:

   ```
   ci: static-check the workflow and shell surface (T-088)

   actionlint + shellcheck now run in a dedicated ci-surface job and,
   when installed, from `just lint`; a Go test shellchecks the pre-commit
   shim, which is shell source shipped to users but invisible to a file
   linter. All three workflows declare permissions and concurrency —
   release.yml deliberately never cancels in progress.
   ```

5. **Tidy up before presenting** — root-path child: interactive-rebase the WIP commits into a
   small number of atomic, correctly typed commits (`ci:` for the workflow/justfile work,
   `test:` or `fix:` for the shim) and keep that history.
6. Commit locally on the ticket branch. Do **not** push or open an MR without user approval.
   Present the commit message; after approval, verify the remote base is not behind
   (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing), push, complete the acceptance test's CI half, and open the merge request — merging
   is always the human's. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-09 — created (TO DO). source: T-087's review, batching two non-blocking findings by
  theme (no static check for the CI shell/workflow surface; `manual-smoke`'s missing
  `permissions`/`concurrency`). Graded low/low/S against the backlog: a bounded config diff,
  narrow but real — it is the check that would have caught T-087's script-injection finding
  automatically.
- 2026-08-13 — TO DO → READY: plan complete: actionlint+shellcheck in a ci-surface job and just lint, shim shellchecked, three workflows hygiene; cost S -> M
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
- 2026-08-13 — IN DEVELOPMENT → IN REVIEW: acceptance green
