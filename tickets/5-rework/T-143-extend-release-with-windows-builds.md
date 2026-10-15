---
id: T-143
title: Extend release with Windows builds
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: L
---

# T-143 — Extend release with Windows builds

## Outcome

A tagged release publishes `pickle_<version>_windows_amd64.zip` and `…_windows_arm64.zip`, each
holding `pickle.exe`, next to the darwin/linux tarballs. That binary installs and runs the ticket
flow on Windows, symlink privilege or not: when Windows refuses a symlink, `pickle install` writes
a copy instead, and `upgrade`, `doctor` and `uninstall` all understand that copy.

## Description

`.goreleaser.yaml` builds `darwin`/`linux` × `amd64`/`arm64` as `tar.gz` only, and the documented
install paths are the Homebrew tap, `go install` and building from source.
`docs/user-manual/installation.adoc` says `go install` works on "any platform", and that is false
today: **the binary does not compile for Windows**.

What I verified while refining (2026-10-15, against `main` at `393bf16`):

- **The only blocker for the binary is `internal/lock/lock.go`.** It calls `syscall.Flock`,
  `LOCK_EX`, `LOCK_SH`, `LOCK_NB` and `LOCK_UN`, none of which exist on windows. Its package
  comment says outright that a Windows target would need a build-tagged `LockFileEx` variant. With
  that package stubbed, `GOOS=windows go build .` succeeds and nothing else is missing.
- **Two test files don't compile for windows.** `internal/atomicfile/atomicfile_test.go:15` uses
  `syscall.Stat_t` and `internal/config/config_test.go:1388` uses `syscall.Umask`. This matters
  because `GOOS=windows go vet ./...` type-checks test files too.
- **Symlinks.** On Windows, `os.Symlink` needs Developer Mode or admin rights.
  `internal/install/install.go`'s `ensureSymlink` makes the links in two cases:
  - **The Claude view, `.claude/skills/brine` → `../../.agents/skills/brine`.** This is the
    default whenever `--agent` includes `claude`. Both `Run` and `Upgrade` call it, after
    `copyPayload` has written `.agents/skills/brine`.
  - **`CLAUDE.md` → `AGENTS.md`.** Only with `--claude-symlink`; by default `CLAUDE.md` gets a
    marker block.

  A third call re-creates a *legacy self-host* link during `upgrade`. That path only runs on
  pickle's own repository, and self-hosting on Windows is out of scope.

  Other code treats a real file or directory at either path as a defect:
  - `doctor`'s `checkClaudeView` reports "exists but is not a symlink".
  - `ensureSymlink` refuses to replace one.
  - `Uninstall` calls `os.Remove` on the view, which fails on a non-empty directory.
- **Hooks** are `#!/bin/sh` shims that call `pickle hooks run …` (`internal/hook/hook.go`). Git for
  Windows runs hooks through its bundled `sh`, so they should work unchanged. The smoke job below
  proves it.
- **`serve`** passes `syscall.SIGTERM` to `signal.NotifyContext`. That compiles on windows and the
  signal never arrives there, which is harmless.

- **Symlinks committed by a unix teammate become plain-text placeholders on a Windows clone.**
  Git for Windows defaults to `core.symlinks=false`, so a tracked `.claude/skills/brine` or
  `CLAUDE.md` link (git mode 120000) checks out as a small regular file whose content is the link
  target. Without handling, three things go wrong:
  - `upgrade` refuses the view (install.go:1121).
  - `doctor` errors (doctor.go:155).
  - `RefreshMarkers` appends a marker block onto the `CLAUDE.md` placeholder, which corrupts the
    tracked link.
- **`os.Symlink` on windows stores the target with backslashes.** `Readlink` then never equals
  `ClaudeSkillTarget`, so every run re-creates a valid link unless the comparison uses
  `filepath.ToSlash`.

Out of scope:
- Scoop and winget manifests.
- Running the full `go test ./...` on Windows. About 20 test files assume symlinks or unix
  permission modes, and an `os.Symlink` that succeeds on an admin CI runner proves little anyway.
- Self-hosting this repository on Windows.
- **Proving the copy fallback on real Windows.** GitHub's `windows-latest` runners can create
  symlinks, so the unprivileged path is covered only by the injected-failure unit tests (Task 3).
  The `windows-smoke` job proves the binary, the lock and the hooks on Windows, not the fallback.
- Detecting drift between the copied Claude view and `.agents/skills/brine`. `upgrade` refreshes
  both together, so drift only comes from hand edits to a pickle-owned directory.

Side finding, folded into the docs step: `installation.adoc` tells readers to run
`brew install codcod/taps/pickle`, while `README.md` and `.goreleaser.yaml` (`homebrew-tap`) say
`codcod/tap`.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                         # child `pickle` is the repo root
git checkout main
git checkout -b feat/T-143-windows-release-builds
```

Do all work on this branch and commit WIP locally as you go. Publish only per the commit policy:
never push or open a merge request without explicit user approval. Because this is a root-path
child, tidy the WIP commits into atomic ones before presenting them; keeping that history is the
default over squashing. This repo is `in-tree`, so before pushing,
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing. Merging is always the human's job.

### Prerequisite gate (hard)

None. No `depends-on:`. The working tree must be clean on `main` before you branch.

### Confirmed design decisions (do not deviate without asking)

1. **Windows locking uses `LockFileEx`/`UnlockFileEx` from `golang.org/x/sys/windows`. This is the
   ticket's one new dependency.** The stdlib `syscall` package does not expose either call on
   windows, and the hand-rolled alternative (`NewLazyDLL` plus an `Overlapped` struct) is more
   code that is harder to read. Keep `syscall.Flock` on unix.
2. **Lock semantics do not change on either OS.** The exclusive and shared modes, the
   non-blocking retry up to `acquireTimeout` every `retryInterval`, the timeout message, and the
   rule that the lock file is never removed all stay as they are. On windows, a busy lock is
   `ERROR_LOCK_VIOLATION`; treat it the way unix treats `EWOULDBLOCK`/`EAGAIN`. Lock one byte at
   offset 0. Locking past EOF is legal, and the file stays empty.
9. **A link placeholder is a regular file of at most 4 KiB whose trimmed content, read with
   `filepath.ToSlash`, equals the link target.** A Windows clone that checked out committed
   symlinks without `core.symlinks` produces exactly this. It is recognised by one predicate,
   `IsLinkPlaceholder(path, target string) bool`, in `internal/install`:
   - **A Claude-view placeholder** (content `../../.agents/skills/brine`) is replaced: `install`
     and `upgrade` remove it, then link or copy as for an absent path. `doctor` reports it as an
     error whose remedy is `pickle upgrade`, and `uninstall` removes it.
   - **A `CLAUDE.md` placeholder** (content `AGENTS.md`) is never written to. `install`
     (with or without `--claude-symlink`) and `RefreshMarkers` skip it and say so. `doctor`
     warns, naming Developer Mode plus `git config core.symlinks true` as the remedy. `uninstall`
     removes it, as it removes a real link.
10. **Symlink-target comparisons normalise with `filepath.ToSlash`** (`ensureSymlink`'s `Readlink`
   check), because windows stores link targets with backslashes.
3. **When a symlink cannot be created, write a copy. Never fail the install.** This applies on any
   OS; Windows privilege is the motivating case. Route every `os.Symlink` call in
   `internal/install` through a package variable `var symlink = os.Symlink` so tests on
   darwin/linux can force the failure.
4. **For the Claude view, the fallback is a full copy of `.agents/skills/brine`, marked as
   pickle's by a sentinel file.** *(Amended 2026-10-15 after review round 1, F1–F3, with user
   sign-off.)*
   - **Building a copy.** The copy goes into a temporary sibling directory with stdlib
     `os.CopyFS`, and the sentinel `.pickle-copy` is written last. The directory is then renamed
     into place. A partial copy therefore never carries the sentinel.
   - **What counts as pickle's copy.** Only a real directory carrying `.pickle-copy` is pickle's
     copy. A real directory without it is refused, as it is today.
   - **On every `install` and `upgrade`.** An existing copy is removed and the symlink is tried
     first. A copy is written only when the symlink fails.
   - **`doctor` and `uninstall`.** `doctor` accepts a copy, and `uninstall` removes it with
     `RemoveAll`.
   - **The guard.** No destructive step touches the view unless its parent,
     `filepath.EvalSymlinks(<root>/.claude/skills)`, resolves inside the project root.
5. **For `CLAUDE.md` under `--claude-symlink`, the fallback is the existing marker-block form, not
   a copy of `AGENTS.md`.** `doctor`, `RefreshMarkers` and `uninstall` already handle a regular
   `CLAUDE.md` that carries markers. Under `--claude-symlink`, a re-run that finds a regular
   `CLAUDE.md` with a marker block refreshes the block and notes it, where today it refuses.
   A regular `CLAUDE.md` with no marker block keeps today's refusal.
6. **The output names every fallback.** A copy is reported as, for example,
   `+ .claude/skills/brine (copy — symlink unavailable: <err>)`. `doctor` reports
   `claude view present (.claude/skills/brine, copy)` as a passed line. A copy is a supported
   state, not a warning.
7. **The release adds `windows` to `goos`, ships both amd64 and arm64, and archives windows as
   `zip`** through `archives[].format_overrides`. The `brews` pipe is unchanged; it only consumes
   darwin/linux archives.
8. **CI guards Windows with a cross-compile and a smoke job, not the full test suite.** The
   cross-compile runs `GOOS=windows go vet ./...` (amd64) plus `go build` for arm64 in
   `ci.yml`'s `build-test`. The smoke job runs on `windows-latest` and drives the real
   `pickle.exe`.

### Tasks

#### Task 1 — Windows file locking (`internal/lock`)

- Split the platform calls out of `internal/lock/lock.go`:
  - `lock_unix.go` (`//go:build unix`) and `lock_windows.go` each provide
    `tryLock(f *os.File, exclusive bool) (busy bool, err error)` and `unlock(f *os.File)`.
  - The unix file wraps `syscall.Flock` with `LOCK_NB`. The windows file uses
    `windows.LockFileEx(h, flags|LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)`, adding
    `LOCKFILE_EXCLUSIVE_LOCK` when `exclusive` is true, and `windows.UnlockFileEx`.
- In `lock.go`, `acquire` loops on `tryLock`. `WithExclusive` and `WithShared` defer `unlock`.
  Remove the `syscall` import from `lock.go`.
- Rewrite the package comment's "Targets are darwin and linux only…" paragraph. It should say
  where each platform's implementation lives and that both keep the same semantics (decision 2).
- `go get golang.org/x/sys@latest && go mod tidy`.

#### Task 2 — Make the test files compile for windows

- Move these out of `internal/atomicfile/atomicfile_test.go` into `atomicfile_unix_test.go`
  (`//go:build unix`):
  - the `syscall.Stat_t`-dependent helper (line 15) and the tests that use it;
  - the two `syscall.Umask` tests, `TestWriteFileCreateRespectsUmask` and
    `TestWriteFileRewriteIgnoresUmask`.
- Do the same for the `syscall.Umask` test in `internal/config/config_test.go`: move it into
  `config_unix_test.go`.
- Do not change any test's logic.

#### Task 3 — Copy fallback for the Claude view and `CLAUDE.md` (`internal/install`, `internal/doctor`)

In `internal/install/install.go`:
- Add `var symlink = os.Symlink` and use it inside `ensureSymlink` (decision 3).
- Add `ensureClaudeView(root string, res *Result) error`, implementing decision 4. Use it at both
  Claude-view call sites, `Run` (near line 245) and `Upgrade` (near line 494), in place of the
  direct `ensureSymlink`. Its behaviour by what is at the path:
  - **A symlink:** do what `ensureSymlink` does today.
  - **A real directory with `SKILL.md`:** `RemoveAll`, then `os.CopyFS` from `SkillDir`, then
    report `(copy refreshed)`.
  - **A real directory without `SKILL.md`:** keep today's refusal error.
  - **Nothing there:** try `symlink`. On error, `os.CopyFS` and report the fallback (decision 6).
- `Run`'s `--claude-symlink` branch, implementing decision 5:
  - **The symlink fails:** fall back to `injectMarker` on `CLAUDE.md` and note it.
  - **`CLAUDE.md` is a regular file with a marker block:** call `injectMarker` instead of
    refusing.
- In `Uninstall` (near line 650), a real directory at the Claude view that has `SKILL.md` is
  removed with `os.RemoveAll`. The dry-run label is `(dry-run, copy)`.
- Add `IsLinkPlaceholder` and wire it as decision 9 specifies:
  - `ensureClaudeView` treats a Claude-view placeholder as absent once it has removed it.
  - `Run`'s `CLAUDE.md` branch (both modes) and `RefreshMarkers` skip a `CLAUDE.md` placeholder,
    reporting `= CLAUDE.md (symlink placeholder — left alone)`.
  - `Uninstall` and `uninstallMarkerFile` remove placeholders.
- In `ensureSymlink`, compare `filepath.ToSlash(cur) == target` (decision 10).

In `internal/doctor/doctor.go`:
- `checkClaudeView` passes a real directory that contains `SKILL.md` with
  `claude view present (.claude/skills/brine, copy)`. A placeholder is an error that names
  `pickle upgrade`. Every other non-symlink is still an error.
- `checkMarkers` warns on a `CLAUDE.md` placeholder (decision 9) instead of reporting "no pickle
  marker block".

Tests, all running on darwin/linux. Each one swaps `symlink` for a function that returns an error
and restores it with `t.Cleanup`:
- `internal/install`: `install` writes a copy view that contains `SKILL.md`, and the output names
  the fallback. Planting a stale extra file in the copy and then running `upgrade` removes it.
  `uninstall` removes the copy. A real `.claude/skills/brine` directory without `SKILL.md` is
  still refused. With `--claude-symlink` and a failing symlink, `CLAUDE.md` is a regular file
  with the marker block, and a second `install --claude-symlink` succeeds.
- Placeholders: `upgrade` turns a planted Claude-view placeholder into a link (or a copy, with
  `symlink` forced to fail). `RefreshMarkers` and `install` leave a planted `CLAUDE.md` placeholder
  byte-for-byte unchanged.
- `internal/doctor`:
  - a copy view yields 0 errors and the `copy` passed line;
  - a Claude-view placeholder is an error;
  - a `CLAUDE.md` placeholder is a warning.

#### Task 4 — Release config (`.goreleaser.yaml`)

- `builds[0].goos`: add `windows`.
- `archives[0]`: add the following, so archive names stay
  `pickle_<version>_windows_<arch>.zip`:
  ```yaml
  format_overrides:
    - goos: windows
      formats: [zip]
  ```
- Update the header comment if it lists targets.

#### Task 5 — CI (`.github/workflows/ci.yml`)

- Add two steps to `build-test`, after `build`:
  ```yaml
  - name: windows cross-compile
    run: |
      GOOS=windows GOARCH=amd64 go vet ./...
      GOOS=windows GOARCH=arm64 go build -o /dev/null .
  ```
- Add a job `windows-smoke` on `runs-on: windows-latest` with `defaults: run: shell: bash`, using
  the same checkout and setup-go steps as `build-test`:
  ```bash
  T="$(cygpath -u "$RUNNER_TEMP")"     # RUNNER_TEMP is D:\a\_temp; its colon would split PATH
  mkdir -p "$T/bin" && go build -o "$T/bin/pickle.exe" .
  export PATH="$T/bin:$PATH"
  mkdir "$T/smoke" && cd "$T/smoke"
  git init -q -b main
  git -c user.name=ci -c user.email=ci@example.com commit -q --allow-empty -m init
  pickle install --in-tree --agent claude
  pickle hooks install
  pickle ticket new "smoke" --project smoke
  pickle ticket move T-001 dropped --reason smoke
  pickle board audit
  pickle doctor
  git checkout -q -b feat/T-001-smoke
  git add tickets
  if git -c user.name=ci -c user.email=ci@example.com commit -q -m x; then
    echo "::error::pre-commit hook did not refuse tickets/ on a feature branch"; exit 1
  fi
  ```
  The child is named after the directory, so it is `smoke`. I rehearsed this exact sequence on darwin on 2026-10-15 and it
  ended in a refused commit.

### Acceptance test

Run from the repo root on the feature branch:

1. `GOOS=windows GOARCH=amd64 go build -o /dev/null . && GOOS=windows GOARCH=arm64 go build -o /dev/null .`
   exits 0.
2. `GOOS=windows GOARCH=amd64 go vet ./...` exits 0. This proves the Task 2 test files compile.
3. `go test ./internal/lock/... ./internal/install/... ./internal/doctor/... ./internal/atomicfile/... ./internal/config/...`
   passes, including the new fallback tests from Task 3.
4. `just build && just test && just lint && just docs-check` are all green.
5. `just dist-check` passes, and `just dist-snapshot` leaves `dist/pickle_*_windows_amd64.zip` and
   `dist/pickle_*_windows_arm64.zip`. Each one's `unzip -l` lists `pickle.exe`. The four
   darwin/linux `tar.gz` archives are still produced.
6. Throwaway install with forced fallback (darwin/linux), per the self-modify policy. Run the
   Task 3 install test by name with `-v`, then confirm in its output that it covered all three
   paths:
   - `+ .claude/skills/brine (copy — symlink unavailable: …)`
   - an `upgrade` refresh
   - `uninstall` removing the copy
7. **On GitHub (after approval to push):** the `windows-smoke` job is green on the PR. This is the
   one check that runs the real `pickle.exe` on Windows.

### Docs update (mandatory when user-facing)

- `README.md` § Install: add a Windows paragraph. It says to download `pickle_<version>_windows_<arch>.zip`
  from the GitHub release and put `pickle.exe` on `PATH`. It also says that without Developer
  Mode, `pickle install` writes a copy of the Claude skill view instead of a symlink, and that
  `pickle upgrade` keeps the copy current.
- `docs/user-manual/installation.adoc`:
  - Add `== Windows` with the same content as the README paragraph.
  - Fix the tap name to `codcod/tap`, to match `README.md` (side finding).
- `docs/user-manual/cli-reference.adoc`:
  - In the `install` artifact table (the `.claude/skills/brine symlink` row), add the copy
    fallback and the `--claude-symlink` → marker fallback.
  - In the `uninstall` row, say that it removes a copy view too.
  - In the `upgrade` "re-linked" text (around line 365), add the copy refresh.
  - In the `doctor` Claude-view bullet (around line 449), add that a copy is accepted and a
    placeholder is reported.
- Add a placeholder note to the Windows paragraph in `README.md` and `installation.adoc`:
  - Prefer Developer Mode plus `git config core.symlinks true` before cloning.
  - Otherwise `pickle upgrade` replaces a Claude-view placeholder with a copy, which git then
    reports as changed. Do not commit that.
- `RELEASING.md`: change the "`darwin`/`linux` × `amd64`/`arm64`" list (around line 39) to
  include `windows` and say that windows archives are `.zip`.
- `internal/lock/lock.go` package comment (Task 1).
- `just docs-check` is green.

### Finish (mandatory)

1. The acceptance test is green, steps 1–6 locally; step 7 waits for the push approval. `just
   build`, `just test`, `just lint` and `just docs-check` are clean.
2. Docs are updated as listed.
3. Write a summary covering files touched, decisions made, and anything deferred. Name the
   out-of-scope items from the Description.
4. Suggested commits after tidying (root-path child, so keep the history):
   ```
   feat(lock): support windows via LockFileEx (T-143)
   feat(install): fall back to a copy when symlinks are unavailable (T-143)
   build(release): ship windows amd64/arm64 zip archives (T-143)
   ci: cross-compile for windows and smoke-test pickle.exe (T-143)
   docs: document windows install and the copy fallback (T-143)
   ```
5. Tidy the WIP commits into the atomic commits above before presenting them.
6. Commit locally on `feat/T-143-windows-release-builds`. Do not push or open an MR. Then run
   `pickle ticket move T-143 in-review --reason "acceptance green"` on `main`, since bookkeeping
   lives on the base branch.

## Review

### Round 1 — 2026-10-15

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | `upgrade` and `uninstall` run `os.RemoveAll` on the Claude view through a symlinked `.claude` or `.claude/skills` parent, which deletes a directory outside the project. `main` refuses this case. The path layout comes from a cloned repo, so this is a trust boundary. | Reproduced: a throwaway install, then `.claude/skills` replaced by a symlink to `../../X/skills`, where `X/skills/brine/` holds `SKILL.md` and `notes.txt`. `pickle-test upgrade` prints `+ .claude/skills/brine (copy refreshed)`, and `X/skills/brine/notes.txt` is gone. A `main` build refuses with "exists and is not a symlink". `IsClaudeViewCopy` (install.go:1217) uses `Lstat` on the last path element only; the deletes are `ensureClaudeView` (install.go:1178-1186) and `Uninstall` (install.go:670-680). | Before any destructive step on the view, require `filepath.EvalSymlinks(filepath.Dir(link))` to stay under `root`. Put the check inside the copy predicate so install, upgrade, uninstall and doctor all get it. Add a regression test. |
| F2 | blocking | correctness | — | Any real directory at `.claude/skills/brine` that contains `SKILL.md` counts as pickle's copy, so `upgrade` wipes and `uninstall` removes a user's own directory. Before, upgrade refused it and uninstall's `os.Remove` failed. This follows T-143 decision 4, so the fix needs sign-off. | Same mechanism as F1 without the symlinked parent: install, then replace the view with a real directory holding `SKILL.md` and `notes.txt`, then `upgrade` → `(copy refreshed)`, and `notes.txt` is lost. Traced: `IsClaudeViewCopy` → `RemoveAll` (install.go:1180). | Mark pickle's copy with a sentinel file that the fallback writes. Only a directory carrying that sentinel is pickle's. |
| F3 | blocking | correctness | — | Once the view is a copy it is refreshed forever and the symlink is never tried again, so a user who later turns on Developer Mode keeps a copy. This follows T-143 decision 4. | Traced: the `IsClaudeViewCopy` case in `ensureClaudeView` (install.go:1175-1187) returns before `ensureSymlink`. | Remove the copy, then retry the symlink, and copy only when the symlink fails. |
| F4 | blocking | correctness | — | Under `--claude-symlink`, a re-run over a marker-carrying `CLAUDE.md` refreshes the block silently. T-143 decision 5 says it "refreshes the block and notes it", so the user is not told that the flag was not applied. | install.go:256-271: the `default:` branch calls `writeMarker()` with no `res.note`. | Add a note naming why no symlink was made, and test it. |
| F5 | blocking | test-gap | — | `smoke.sh` treats any failed `git commit` as the hook refusing, so a hook that crashes (shebang or PATH failure under Git's sh) also passes. This is the only Windows check (acceptance step 7). | `.github/scripts/smoke.sh:33-36`. A failing commit of any kind takes the pass branch. | Capture the commit's stderr and require the guard's refusal text. |
| F6 | non-blocking | correctness | noted | Refreshing a copy is not atomic: `RemoveAll` runs, then `CopyFS`, and a failure partway leaves a partial directory. Today this self-heals, because `SKILL.md` is copied before `resources/`. | install.go:1180-1186; `os.CopyFS` walks entries in lexical order. | Superseded by F2's fix, which writes the copy, sentinel last, into a temporary sibling and renames it into place. A partial copy then never carries the sentinel. |
| F7 | non-blocking | correctness | noted | `ensureSymlink` normalises only the target it reads back, not the one passed in. A target that itself came from `Readlink` (`swept.LinkTarget`, a legacy self-host link on windows) never matches. | install.go:1146 | Self-hosting on Windows is out of scope per the Description. Comparing `ToSlash` on both sides is the fix if it ever comes into scope. |
| F8 | non-blocking | correctness | noted | `IsLinkPlaceholder` matches on content alone, on every OS. A hand-written `CLAUDE.md` whose whole text is `AGENTS.md` would be skipped, removed by uninstall, and warned about by doctor. | install.go:1233 | Such a file is functionally a placeholder already. Claude Code's import convention is `@AGENTS.md`, which does not match. |
| F9 | non-blocking | design | noted | `doctor` accepts a copy view on `SKILL.md` alone and never compares it with `.agents/skills/brine`. | doctor.go:157 | Drift detection is out of scope per the Description. F2's sentinel-last rename removes the partial-copy case. |
| F10 | non-blocking | design | noted | The copy is a second copy path beside `copyPayload`, and it reports a refresh on every upgrade. | install.go:1180 | Already noted and closed at the applicability gate. |
| F11 | non-blocking | docs-gap | fixed inline | The tap-name fix was incomplete: `quickstart.adoc:19` and `attributes.adoc:5` still said `codcod/taps`, so the manual contradicted itself. | `grep -rn codcod/taps docs` | Fixed in `7d46185`. |
| F12 | non-blocking | stale-xref | fixed inline | The `upgrade` text in `cli-reference.adoc` said "`CLAUDE.md` is re-injected only if it is a regular file". A placeholder is a regular file that is now left alone. | cli-reference.adoc:367 vs install.go:352 | Fixed in `7d46185`. |
| F13 | non-blocking | docs-gap | fixed inline | `CHANGELOG.md` `[Unreleased]` had no entry. | `[Unreleased]` was empty | Fixed in `7d46185`. |
| F14 | non-blocking | test-gap | noted | `install --claude-symlink` over a `CLAUDE.md` placeholder, and the `(dry-run, copy)` uninstall label, have no direct test. | `internal/install/symlink_fallback_test.go` | The F4 rework adds a `--claude-symlink` test next to these. |

Dispositions: 5 blocking (F1–F5, → rework); fixed inline 3 (F11, F12, F13); noted 6 (F6–F10, F14); new ticket 0; folded 0.
cost: estimated L, actual L

- [x] Reviewer independence settled (step 0): delegated. A fresh sub-agent ran steps 1–4a with `hunt: angle 7 only`. Its 5 findings are all kept (F1, F11–F14), and none were discarded.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned that the branch had T-143 in `3-in-development`. I rebased onto `main`, then doctor showed 0 errors and 0 warnings.
- [x] Implementation audit (steps 1, 2): the delegated reviewer re-ran AT1–AT4 and AT6, which are green. AT5 (goreleaser snapshot) ran green in the implementation session but could not be re-run, because goreleaser is not installed on the host. AT7 waits for push approval.
- [x] Correctness hunt (step 3): host tool path. `/code-review medium` ran on `main...feat/T-143-windows-release-builds` from the top-level session and produced 9 candidates. I re-verified each against the code: 4 are blocking (F2–F5), 5 are noted (F6–F10), and none were discarded. Angle 7 ran separately in the delegated reviewer and found F1. F1 is reproduced. F2 is reproduced through the same mechanism. F3 and F4 are traced. F6–F10 are traced.
- [x] Consistency audit (step 4): delegated. No caller drift was found for `ensureSymlink`, `IsClaudeViewCopy` or `IsLinkPlaceholder`. It produced F11 and F12.
- [x] Documentation audit (step 4a): delegated. It produced F11–F13. `just docs-check` is green after `7d46185`.
- [x] Docs-readability pass (step 4b): conscious skip. The changed prose is install notes and table rows, and a later rework pass revisits the copy wording.
- [x] Findings recorded (step 5).
- [x] Ticket moved to `tickets/5-rework/` (step 6a).
- [x] Other references and governing documents (step 7): DESIGN.md and AGENTS.md name no platform constraint. `NOTES.md`'s `flock` mentions are dated audit prose, so they are left as they are.
- [x] Impact sweep (step 8): no ticket in `1-to-do/` or `2-ready/` references T-143.
- [ ] Approval gate (step 9): not reached. The verdict is rework.

## History

- 2026-10-15 — created (TO DO). source: chat: user asked to extend the release with Windows builds; filing found `GOOS=windows` does not compile (`internal/lock` uses `syscall.Flock`)
- 2026-10-15 — refined: decisions confirmed by user (copy fallback for symlinks, cross-compile + windows-latest smoke CI, amd64+arm64); verified lock is the sole build blocker; plan written
- 2026-10-15 — TO DO → READY: plan complete
- 2026-10-15 — plan amended inline: applicability gate (fresh sub-agent, 0 blocking) — added link-placeholder handling for Windows clones (decision 9, user-chosen inline), ToSlash link-target compare (decision 10), extra atomicfile Umask tests in Task 2, cygpath PATH fix in the smoke job, doctor/upgrade cli-reference rows, and an out-of-scope line that CI does not exercise the copy fallback on real Windows; noted and closed: copy view always reports "(copy refreshed)" on upgrade
- 2026-10-15 — READY → IN DEVELOPMENT: picked up
- 2026-10-15 — IN DEVELOPMENT → IN REVIEW: acceptance green (steps 1-6 local; step 7 windows-smoke awaits push approval); smoke steps extracted to .github/scripts/smoke.sh so shellcheck covers them and they run locally
- 2026-10-15 — IN REVIEW → REWORK: review round 1: 5 blocking (F1-F5); 3 fixed inline, 6 noted
- 2026-10-15 — plan amended inline: decision 4 rewritten per review round 1 (F1–F3, user sign-off) — `.pickle-copy` sentinel, temp-sibling copy renamed into place, symlink retried before re-copying, parent must resolve inside the root
