---
id: T-011
title: distribution (goreleaser + Homebrew tap + releases + docs)
project: pickle
depends-on: []
impact: high
complexity: medium
cost: M
---

# T-011 — distribution (goreleaser + Homebrew tap + releases + docs)

## Description

Make `pickle` installable. Switch the module path from the bare `pickle` to the real VCS path
`github.com/codcod/pickle`; wire **goreleaser** for cross-compiled static binaries and GitHub
releases; publish a **Homebrew tap in a separate tap repo** (`github.com/codcod/homebrew-taps`,
so `brew install codcod/taps/pickle`); keep stamping the build version via
`-ldflags -X main.version=…` **and** add a `runtime/debug` fallback so `go install …@version`
reports a real version; write user-facing install docs. Enables `brew install` and
`go install`. The command set (P1–P3) is complete and merged; P4 (opencode/Pi) is **not** a
prerequisite for distribution. Phase P5.

**Confirmed decisions (user-approved 2026-07-24):**

- **D1 — host: GitHub.** CI already runs on GitHub Actions; pickle ships on GitHub (the GitLab
  option is declined).
- **D2 — module path: `github.com/codcod/pickle`.** Every `import "pickle/internal/…"` becomes
  `import "github.com/codcod/pickle/internal/…"`. (`main.version` is unaffected — `main` stays
  the root package, so the `-ldflags -X main.version` stamping is unchanged.)
- **D3 — Homebrew tap: `github.com/codcod/homebrew-taps`** (a *separate* repo), formula pushed
  there by goreleaser → users run `brew install codcod/taps/pickle`.
- **D4 — binary + formula name: `pickle`** (unchanged).
- **D5 — release mechanism:** a `.goreleaser.yaml` (v2) cross-compiling darwin/linux ×
  amd64/arm64 with archives + checksums and auto-publishing the brew formula to the tap, driven
  by a tag-triggered `.github/workflows/release.yml` (push `vX.Y.Z` → release). `goreleaser
  check` runs in CI to keep the config valid.
- **D6 — `go install` version fallback:** `runtime/debug.ReadBuildInfo()` supplies the version
  when ldflags didn't (i.e. `go install github.com/codcod/pickle@vX.Y.Z`); `just`/goreleaser
  keep stamping via ldflags.
- **D7 — scope split (approved):** this ticket delivers all **in-repo plumbing** (module
  rename, version fallback, `.goreleaser.yaml`, release + check CI, docs, LICENSE), verifiable
  locally with `goreleaser release --snapshot --clean` (no publish, no tokens). The
  **account-level steps are the human's** and ship as a documented handoff checklist
  (`RELEASING.md`): create the two GitHub repos, set the tap-push token secret, push `main`,
  cut the first tag.

**Open sub-decision (confirm at implementation):** no `LICENSE` exists yet, and the brew
formula needs a `license:` field. This plan assumes **MIT** (safe default for a CLI tool);
change the `LICENSE` file + the goreleaser `license:` field together if a different license is
wanted.

**Scope boundaries:** no new CLI behaviour — this is packaging + docs only. Actually creating
the GitHub/tap repos, storing secrets, pushing, and tagging are out of scope (handoff
checklist). The payload-version mechanism (`pickle.toml`'s `payload_version`, stamped from the
binary `Version` at install) already exists (T-004) and is unchanged.

## Implementation Plan

**Prerequisite gate.** P1–P3 are DONE and merged to `main` (config/registry, board audit,
`ticket new`, `install`, `ticket move`, `board sync`). `depends-on:` is empty. Confirm
`git checkout main && just build && just test && just lint && ./pickle board audit` is clean
before starting.

**Branch.** `git checkout main && git checkout -b feat/T-011-distribution`

### Confirmed decisions
D1–D7 above are settled (LICENSE = MIT unless changed). Implement exactly to them.

### Tasks

1. **Rename the module → `github.com/codcod/pickle`.**
   - `go mod edit -module github.com/codcod/pickle`
   - Rewrite every internal import (14 `.go` files):
     ```sh
     grep -rl '"pickle/internal' --include='*.go' . \
       | xargs sed -i '' 's#"pickle/internal#"github.com/codcod/pickle/internal#g'   # macOS BSD sed
     ```
     (Linux: `sed -i 's#…#…#g'` without the `''`.)
   - `gofmt -w .` then `go mod tidy`; confirm `go build ./...` and `go vet ./...` pass.
   - Sanity: `! grep -rq '"pickle/internal' --include='*.go' .` (no bare imports remain).

2. **Version fallback** — `main.go`:
   - Add `resolveVersion()` using `runtime/debug`: return the ldflags-stamped `version` when it
     is not `"dev"`/`""`; otherwise, if `debug.ReadBuildInfo()` yields `info.Main.Version` that
     is non-empty and not `"(devel)"`, return that (the `go install …@vX.Y.Z` case); else fall
     back to `version`.
   - Call `cli.Run(payloadFS, resolveVersion(), os.Args[1:])`. Leave the `version` var in
     `assets.go` as-is (`"dev"`, overridden by ldflags).

3. **`.goreleaser.yaml`** (v2 schema) at the repo root:
   - `version: 2`; `project_name: pickle`; `before.hooks: [go mod tidy]`.
   - `builds`: id `pickle`, `main: .`, `binary: pickle`, `env: [CGO_ENABLED=0]`,
     `goos: [darwin, linux]`, `goarch: [amd64, arm64]`,
     `ldflags: ["-s -w -X main.version={{.Version}}"]`.
   - `archives`: `formats: [tar.gz]`, `name_template:
     "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"`.
   - `checksum.name_template: "checksums.txt"`.
   - `changelog`: `use: github`, exclude `^docs:`/`^test:`/`^chore:`.
   - `brews`: one entry — `name: pickle`; `repository: { owner: codcod, name: homebrew-taps,
     branch: main, token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}" }`;
     `homepage: "https://github.com/codcod/pickle"`;
     `description: "Ticket-based, board-driven feature flow as a CLI"`; `license: "MIT"`;
     `install: bin.install "pickle"`; `test: system "#{bin}/pickle", "version"`.

4. **Release workflow** — `.github/workflows/release.yml`:
   - Trigger `on.push.tags: ['v*']`; `permissions.contents: write`.
   - Job `goreleaser` on `ubuntu-latest`: `actions/checkout@v4` (`fetch-depth: 0`),
     `actions/setup-go@v5` (`go-version: '1.26'`), `goreleaser/goreleaser-action@v6`
     (`version: '~> v2'`, `args: release --clean`) with env `GITHUB_TOKEN:
     ${{ secrets.GITHUB_TOKEN }}` and `HOMEBREW_TAP_GITHUB_TOKEN:
     ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}`.

5. **CI config validation** — `.github/workflows/ci.yml`: add a `goreleaser-check` job
   (`goreleaser/goreleaser-action@v6`, `version: '~> v2'`, `args: check`) so a broken
   `.goreleaser.yaml` fails CI. Leave the existing `build-test` job intact.

6. **`justfile` recipes:**
   - `dist-check:` → `goreleaser check`.
   - `dist-snapshot:` → `goreleaser release --snapshot --clean` (local cross-compile into
     `./dist`, no publish).
   - Add `dist/` to the `clean` recipe (`rm -rf dist`) and note it should be gitignored
     (add `dist/` to `.gitignore` — pickle's `.gitignore` is real, unlike the workspace's).

7. **`LICENSE`** — add an MIT `LICENSE` (year, `codcod`) unless the sub-decision above changes
   it; keep the goreleaser `license:` field in sync.

8. **Docs** (`README.md` + new `RELEASING.md`):
   - README: add an **`## Install`** section (above `## Build`) with three paths —
     **Homebrew** (`brew install codcod/taps/pickle`), **`go install`**
     (`go install github.com/codcod/pickle@latest`), and **from source** (the existing
     `just build`). Note the first `brew`/`go install` works only once the repos exist and a
     tag is cut (point at `RELEASING.md`).
   - README: update the stale “module path is `pickle` (bare) … at P5” note (Layout section) to
     state the real path `github.com/codcod/pickle`; flip the **P5** phased-plan bullet to done.
   - New `RELEASING.md`: the maintainer release process (bump → `git tag vX.Y.Z` → push tag →
     CI runs goreleaser → GitHub release + tap formula updated) **and the one-time human handoff
     checklist**: (a) create `github.com/codcod/pickle`; (b) create
     `github.com/codcod/homebrew-taps`; (c) create a PAT with `repo` scope on the tap and add it
     to the `pickle` repo as the `HOMEBREW_TAP_GITHUB_TOKEN` secret; (d) `git remote add origin`
     + push `main`; (e) `git tag v0.1.0 && git push origin v0.1.0`.

### Acceptance test (run verbatim; must be green before review)

```sh
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"      # pickle repo

# module rename
grep -q '^module github.com/codcod/pickle$' go.mod && echo "OK: module renamed"
! grep -rq '"pickle/internal' --include='*.go' . && echo "OK: no bare imports remain"
go build ./... && echo "OK: builds after rename"
just test && echo "OK: tests green"
just lint && echo "OK: lint clean"

# version stamping + go-install fallback
just build >/dev/null && ./pickle version | grep -Eq '^pickle .+' && echo "OK: stamped version"
go build -o /tmp/pk-noflags . && /tmp/pk-noflags version | grep -Eq '^pickle .+' && echo "OK: version fallback non-empty"

# goreleaser config + local cross-compiled snapshot (no publish, no tokens).
# The `just dist-*` recipes unset any ambient GITLAB_TOKEN so goreleaser detects
# the GitHub forge deterministically (see RELEASING.md).
command -v goreleaser >/dev/null || { echo "install goreleaser first: brew install goreleaser"; exit 1; }
just dist-check && echo "OK: .goreleaser.yaml valid (no deprecations)"
just dist-snapshot >/dev/null && echo "OK: snapshot build"
ls dist/pickle_*_darwin_arm64.tar.gz dist/pickle_*_linux_amd64.tar.gz >/dev/null && echo "OK: cross-compiled archives"
test -f dist/checksums.txt && echo "OK: checksums produced"
grep -q 'github.com/codcod/pickle/releases/download' dist/homebrew/Casks/pickle.rb && echo "OK: cask URLs point at GitHub" || { echo "FAIL: cask URLs not GitHub (forge detection)"; exit 1; }
! grep -q 'gitlab.com' dist/homebrew/Casks/pickle.rb && echo "OK: no gitlab.com URLs" || { echo "FAIL: gitlab.com leaked into cask"; exit 1; }

echo "ACCEPTANCE PASS"
```

Also: `just test` (all packages green after the rename) and `just lint` clean; `just clean`
removes `dist/`.

### Docs

Covered by Task 8: README `## Install` section + updated module-path/P5 notes, and a new
`RELEASING.md` (maintainer process + human handoff checklist). No `docs/` tree exists in
pickle — README + `RELEASING.md` are the docs surface.

### Finish

- Local WIP commits on the branch as you go. `goreleaser release --snapshot --clean` is the
  local proof; **do not** push, tag, or publish anything — those are the human handoff steps.
- Present the suggested Conventional Commit message and MR attributes for approval; do not push
  / open an MR without it.
- Suggested commit (broad change, no single scope → omit scope):
  `feat: wire distribution — goreleaser + Homebrew tap + release CI; rename module to github.com/codcod/pickle (T-011)`.
- Move T-011 to `4-in-review/` (History line + board row) and hand back for validation. Include
  the handoff checklist from `RELEASING.md` in the review summary so the human knows exactly
  what remains to actually publish.

## Review

**Verdict: PASS — 0 blocking, 0 new tickets; 2 justified plan deviations recorded + 1 trivial
inline patch. Reviewed on `feat/T-011-distribution` (un-merged, publish-gated).**

- [x] Implementation audit — acceptance re-run **verbatim → ACCEPTANCE PASS** (13/13 checks); tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — README `## Install` + `RELEASING.md`; no docs build (README/MD is the doc surface) (step 4a)
- [x] Findings classified & recorded; no blocking, no new tickets (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6b)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep done (step 8)
- [x] Summary + commit message & MR attributes + human handoff presented for approval (step 9)

### Implementation audit (step 2)

| Task | Result | Evidence |
|---|---|---|
| 1. Module rename → `github.com/codcod/pickle` (14 files) | **met** | `go.mod`; `! grep '"pickle/internal'`; `go build ./...` |
| 2. `resolveVersion()` version fallback | **met** | `main.go`; acceptance "version fallback non-empty" (plain `go build` → pseudo-version via build info) |
| 3. `.goreleaser.yaml` (builds/archives/checksums/brew) | **met, w/ deviation** | see D5′/D3′ below; `goreleaser check` clean, snapshot builds 4 targets |
| 4. `release.yml` tag-triggered | **met** | `.github/workflows/release.yml` (`v*`, GITHUB_TOKEN + HOMEBREW_TAP_GITHUB_TOKEN) |
| 5. CI `goreleaser-check` job | **met** | `.github/workflows/ci.yml` (`args: check`) |
| 6. justfile `dist-check`/`dist-snapshot` + `clean`/gitignore | **met** | `justfile`, `.gitignore` `/dist/`; `just clean` removes dist |
| 7. MIT `LICENSE` | **met** | `LICENSE` (© 2026 codcod) — **confirm license choice** |
| 8. Docs: README `## Install` + module/P5 notes + `RELEASING.md` | **met** | README, `RELEASING.md` |
| `just build`/`test`/`lint` clean | **met** | acceptance |

### Plan deviations (recorded, not findings — justified & documented)

- **D5′ / D3′ — Homebrew *cask*, not *formula*.** goreleaser v2.17 **deprecated `brews`** (and
  the `homebrew_casks.binary` field); `goreleaser check` exits non-zero on deprecations, so the
  plan's literal `brews:` block would fail CI. Migrated to `homebrew_casks:`. **Consequence:**
  casks are **macOS-only** — Linux `brew` users are not served; they use `go install` or the
  release archives. This is documented in `.goreleaser.yaml`, README `## Install`, and
  `RELEASING.md`. The ticket intent (`brew install codcod/taps/pickle`) is met on macOS.
- **`changelog.use: git`, not `github`, + explicit `release.github` + `just dist-*` unset
  `GITLAB_TOKEN`.** Two environment realities forced this: (a) `changelog.use: github` and the
  SCM check need a remote, which doesn't exist pre-handoff; (b) an ambient `GITLAB_TOKEN` in the
  dev shell made goreleaser emit `gitlab.com` cask URLs. Pinning `release.github` + unsetting
  the GitLab tokens in the `just` recipes makes output deterministically GitHub; CI runners
  have neither issue. Guarded by acceptance assertions ("cask URLs point at GitHub" / "no
  gitlab.com").

### Findings

| # | Severity | Description | Disposition |
|---|---|---|---|
| N1 | trivial (patched inline) | `main.go` package comment said "initial skeleton"; `resolveVersion` comment claimed plain `go build` reports `(devel)` — Go 1.26 actually stamps a VCS pseudo-version. | Patched inline (commit `a885155`). |

No blocking findings. No non-blocking tickets spawned: the cask macOS-only limitation is an
inherent, documented consequence of the goreleaser deprecation (no clean cross-distro `brew`
path exists), and `resolveVersion` is exercised by the acceptance test (no separate unit test
warranted for a 12-line fallback in `package main`).

### Consistency & docs notes

- `brew install codcod/taps/pickle` consistent across README + RELEASING.md; tap = `homebrew-taps`
  → tap name `codcod/taps`, cask `pickle`. README anchors (`#build`, `#commands`) resolve.
- `go.mod` module line changed (1 line); `go.sum` untouched (no dependency change). `go vet` +
  `gofmt` clean.
- A local `origin` remote (`https://github.com/codcod/pickle.git`) was added for goreleaser SCM
  detection — **no push occurred**; pushing is a handoff step (RELEASING.md).

### Human handoff (from RELEASING.md — required to actually publish)

1. Create `github.com/codcod/pickle`; `git push -u origin main`.
2. Create `github.com/codcod/homebrew-taps`.
3. Add a `repo`-scoped PAT as the `pickle` repo's `HOMEBREW_TAP_GITHUB_TOKEN` Actions secret.
4. `git tag v0.1.0 && git push origin v0.1.0` → the release workflow does the rest.

### Impact sweep (step 8)

No `2-ready/`/`1-to-do/` ticket lists T-011 in `depends-on:`. The module-path rename is
repo-internal (all imports updated); no other ticket encodes the old bare `pickle` path.
Distribution adds no CLI behaviour, so T-005/T-006/T-009/T-010/T-012/T-013/T-014/T-015 are
unaffected.

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P5)
- 2026-07-24 — TO DO → READY: refined; D1-D7 confirmed
- 2026-07-24 — READY → IN DEVELOPMENT
- 2026-07-24 — IN DEVELOPMENT → IN REVIEW
- 2026-07-24 — IN REVIEW → DONE: review PASS; 0 blocking; D5'/D3' cask deviation recorded; N1 patched inline
