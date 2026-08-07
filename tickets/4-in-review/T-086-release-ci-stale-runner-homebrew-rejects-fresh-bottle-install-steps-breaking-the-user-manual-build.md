---
id: T-086
title: release CI: stale runner Homebrew rejects fresh bottle install steps, breaking the user-manual build
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-086 — release CI: stale runner Homebrew rejects fresh bottle install steps, breaking the user-manual build

## Description

Spawned from the `v0.3.0` release run
(https://github.com/codcod/pickle/actions/runs/31221987955/job/93008336893).
`.github/workflows/release.yml`'s **"Build user manual (PDF + EPUB)"** step
(`continue-on-error: true`, by design — RELEASING.md: "a broken manual does not block
publishing the binaries") failed on all 3 retries:

```
==> Installing codcod/tap/snowball dependency: node
==> Pouring node--26.7.0.x86_64_linux.bottle.tar.gz
##[error]unknown install step: remove
...
You have disabled automatic updates and have not updated today.
Do not report this issue until you've run `brew update` and tried again.
```

The binaries/checksums/GitHub-release/Homebrew-tap publish all succeeded (that is the
soft-fail contract working as intended) — only the PDF/EPUB manual never got attached to
the `v0.3.0` release.

**Root cause.** The `ubuntu-latest` runner ships a point-in-time Homebrew snapshot. A
bottle in homebrew-core (here, `node`, pulled in as a `snowball` dependency) can be built
with a newer post-install-step DSL than that stale `brew` understands — `"remove"` is a
legal `install_steps.rb` step in current Homebrew, not in the runner's copy — so `brew`
aborts with `unknown install step: remove` instead of installing it. This is **not** the
transient broken-pipe download failure the existing 3-attempt retry loop was written for
(`.github/workflows/release.yml`'s own comment); it is a version mismatch that retrying the
same stale `brew install` three times can never clear, which is exactly what the log shows —
all 3 attempts fail identically.

The step already sets `HOMEBREW_NO_AUTO_UPDATE=1` (to silence brew's terminal-width
probing, which was causing unrelated intermittent broken-pipe failures — see the step's
existing comment). That variable only suppresses the *implicit* update `brew
install`/`upgrade` would otherwise trigger; it does not affect an explicit `brew update`.

**Fix.** Run `brew update --quiet` once, before the install retry loop, so the runner's
Homebrew (and its formula index) is current before it is asked to pour a bottle using
today's install-step DSL. Keep everything else (the retry loop, the quiet env vars) as-is —
they handle a different, still-real failure mode.

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/T-086-stale-runner-homebrew-install-steps
```

(`pickle`'s child path is `.` — this repo itself.)

### Confirmed design decisions

1. Fix in place in the existing step — no new step, no change to the retry loop or the
   `continue-on-error`/soft-fail contract.
2. `brew update --quiet` runs unconditionally, once, before the retry loop (not inside it —
   this is an index refresh, not a per-attempt recovery action).

### Tasks

#### Task 1 — add `brew update --quiet` before the install retry loop

In `.github/workflows/release.yml`'s "Build user manual (PDF + EPUB)" step, right after
`eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"` and before the `for attempt in 1 2
3; do brew install codcod/tap/snowball ...` loop, add `brew update --quiet` with a comment
explaining why (stale-runner-Homebrew vs. transient-broken-pipe are different failure modes,
and why `HOMEBREW_NO_AUTO_UPDATE` does not already cover this).

### Acceptance test

1. `.github/workflows/release.yml` is valid YAML (`python3 -c "import yaml;
yaml.safe_load(open('.github/workflows/release.yml'))"` or equivalent).
2. `just build`, `just test`, `just lint`, `just docs-check` all still pass (the change is
   CI-only; nothing in `internal/` or `docs/` moves).
3. Re-run the `release` workflow for the existing `v0.3.0` tag from this fix's branch
   (`gh workflow run release.yml --ref feat/T-086-stale-runner-homebrew-install-steps -f
   tag=v0.3.0`, after pushing the branch — this is RELEASING.md's documented
   `workflow_dispatch` re-run path) and confirm the "Build user manual" step succeeds and
   `pickle-user-manual-0.3.0.pdf`/`.epub` are attached to the `v0.3.0` release. This doubles
   as backfilling the assets `v0.3.0` shipped without.

### Docs update

None — no user-facing surface; `RELEASING.md`'s documented process is unchanged.

### Finish

As in TEMPLATE.md: summarize, suggest a Conventional Commit message, commit locally, then
publish only after explicit user approval (child_publish_gated).

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-08 — created (TO DO). source: pickle ticket new
- 2026-08-08 — TO DO → READY: plan complete: root-caused, single-line fix, acceptance test defined
- 2026-08-08 — READY → IN DEVELOPMENT: picked up
- 2026-08-08 — IN DEVELOPMENT → IN REVIEW: acceptance green: YAML valid, build/test/lint/docs-check clean; live CI verification deferred to post-approval re-run (requires pushing the branch)
