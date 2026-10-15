---
id: T-143
title: Extend release with Windows builds
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-143 — Extend release with Windows builds

## Outcome

A tagged release publishes `windows/amd64` and `windows/arm64` `pickle.exe` archives (zip) next to
the darwin/linux tarballs, and the binary in them builds, installs and runs the ticket flow on
Windows.

## Description

`.goreleaser.yaml` builds only `darwin` and `linux` (`amd64`, `arm64`), archived as `tar.gz`, and
the only install path in `README.md` is the Homebrew tap. Adding `windows` to `goos` is the
smallest part of this ticket, because **the binary does not compile for Windows today**:
`GOOS=windows go build .` fails in `internal/lock/lock.go`, which uses `syscall.Flock` /
`LOCK_EX` / `LOCK_SH` / `LOCK_NB` (its own comment says it assumes darwin and linux targets only,
citing `.goreleaser.yaml`).

Scope:

- **Make it compile.** Split the flock calls in `internal/lock` behind build tags: keep
  `syscall.Flock` on unix, and use `LockFileEx`/`UnlockFileEx` (via `golang.org/x/sys/windows`,
  or `syscall` directly) on windows. Keep the existing semantics: shared and exclusive locks,
  non-blocking retry up to `acquireTimeout`, and the lock file that is never removed.
- **Check the Windows-sensitive paths behave**, rather than only compile:
  - `install` creates the `.claude/skills/brine` view with `os.Symlink`
    (`internal/install/install.go`). On Windows that needs Developer Mode or admin rights. Decide
    whether to fail with a clear message or fall back (a directory copy, or a junction).
    `doctor`'s `os.Readlink` checks (`internal/doctor/doctor.go`) must agree with that choice.
  - `hooks install` writes `#!/bin/sh` hooks (`internal/hook/hook.go`). Git for Windows runs
    these through its bundled sh, so they probably work, but confirm it.
  - `serve` uses `syscall.SIGTERM` in `signal.NotifyContext`. It compiles on windows, but
    the signal never arrives there, which is harmless.
  - Path handling: ticket paths, the `/T-NNN-` matching and the marker-block writes need
    `filepath` versus `/` checked wherever a path is compared as a string.
- **Release config.** Add `windows` to `builds[].goos`. Add a `format_overrides` entry so the
  windows archives are `zip`. The `brews` pipe stays darwin/linux only. A Scoop manifest or winget
  is out of scope unless someone asks for it.
- **CI.** Add a `GOOS=windows` cross-compile to `ci.yml` so the build cannot silently regress.
  A `windows-latest` job running `go test ./...` is the stronger guard; refinement decides
  whether its cost is worth it, since tests that assume symlinks or unix modes may need skips.
- **Docs.** Add a Windows install note (download the zip from the release) to `README.md` and the
  user manual, and add the new artifacts to `RELEASING.md` if it lists them.

Open decisions for refinement: the symlink fallback on Windows, whether CI runs the full test
suite on `windows-latest` or only cross-compiles, and whether arm64 is worth shipping.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-10-15 — created (TO DO). source: chat: user asked to extend the release with Windows builds; filing found `GOOS=windows` does not compile (`internal/lock` uses `syscall.Flock`)
