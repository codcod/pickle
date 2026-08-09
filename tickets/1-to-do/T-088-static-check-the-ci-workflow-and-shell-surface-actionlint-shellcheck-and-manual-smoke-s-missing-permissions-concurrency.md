---
id: T-088
title: static-check the CI workflow and shell surface: actionlint + shellcheck, and manual-smoke's missing permissions/concurrency
project: pickle
depends-on: []
spawned-by: [T-087]
impact: low
complexity: low
cost: S
---

# T-088 — static-check the CI workflow and shell surface: actionlint + shellcheck, and manual-smoke's missing permissions/concurrency

## Outcome

After this ships, a malformed workflow or a sloppy shell script is caught by `just lint` and by
`ci.yml` on every pull request, instead of by whoever next reads the YAML — and `manual-smoke`
runs with a least-privilege token and cancels its own superseded runs.

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

Soft coupling, no `depends-on:`: T-087's branch must merge first, or the two workflow files this
ticket lints will not yet exist on `main`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-09 — created (TO DO). source: T-087's review, batching two non-blocking findings by
  theme (no static check for the CI shell/workflow surface; `manual-smoke`'s missing
  `permissions`/`concurrency`). Graded low/low/S against the backlog: a bounded config diff,
  narrow but real — it is the check that would have caught T-087's script-injection finding
  automatically.
