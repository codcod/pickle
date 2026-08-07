---
id: T-087
title: release CI: snowball setup's bundle install can't find the bundle executable after a fresh Homebrew ruby install
project: pickle
depends-on: []
spawned-by: [T-086]
impact: medium
complexity: low-medium
cost: S-M
---

# T-087 — release CI: snowball setup's bundle install can't find the bundle executable after a fresh Homebrew ruby install

## Description

Surfaced verifying T-086's fix live: `gh workflow run release.yml --ref
feat/T-086-stale-runner-homebrew-install-steps -f tag=v0.3.0`
(https://github.com/codcod/pickle/actions/runs/31222772467/job/93010687635).

T-086 added `brew update --quiet` before the install loop, which fixed its target defect —
the run got past `unknown install step: remove` cleanly this time, pouring every
dependency including `ruby--4.0.6.x86_64_linux.bottle.3.tar.gz` without error. But
`snowball setup` (invoked right after the install loop) then failed with:

```
snowball: installing gems into /home/runner/.cache/snowball/toolchain
Error: bundle install: exec: "bundle": executable file not found in $PATH
```

This is a **different** failure from T-086's: `ruby` itself installed cleanly ("19,024
 files, 70MB", no error), but whatever runs `bundle install` on its behalf cannot find a
`bundle` executable on `$PATH` after `eval "$(brew shellenv)"`. Two candidate causes, neither
yet confirmed:

1. homebrew-core's `ruby` formula is keg-only on macOS ("Ruby is provided by OS X"); if it is
   also keg-only on Linuxbrew, `brew link` never runs for it, so `bin/bundle` never reaches
   `/home/linuxbrew/.linuxbrew/bin` — the one directory `brew shellenv` puts on `PATH`.
2. `brew update` (T-086's own fix) may have moved `ruby` to a version/revision whose bundler
   is not vendored the way an earlier revision's was, or changed install/link ordering enough
   to expose a pre-existing gap that every prior run happened to route around (every prior CI
   run failed *before* reaching `snowball setup`, on the `unknown install step: remove`
   defect T-086 fixed — so this may have been silently broken all along, never previously
   reached).

**Not yet re-tested**, since re-running the full `release` workflow against `v0.3.0` a second
time hit an unrelated `422 already_exists` wall on the GitHub-release asset upload (the tag's
binaries/checksums had already published successfully on the first, real run) — that is a
limitation of testing against an already-fully-published tag, not a new defect; it does not
recur on a real, fresh tag push.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-08 — created (TO DO). source: pickle ticket new
