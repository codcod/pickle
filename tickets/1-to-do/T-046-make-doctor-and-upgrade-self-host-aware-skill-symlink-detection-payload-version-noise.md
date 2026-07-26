---
id: T-046
title: make doctor and upgrade self-host-aware (skill symlink detection, payload-version noise)
project: pickle
depends-on: []
spawned-by: [T-044]
impact: low
complexity: low
cost: S
---

# T-046 — make doctor and upgrade self-host-aware (skill symlink detection, payload-version noise)

## Description

In a **self-hosting** repo (this one), `.agents/skills/ticket-flow` is a **symlink** to the
payload source (`skill/`), not an installed copy. `pickle upgrade` already respects that
symlink and leaves the skill directory alone — but `pickle doctor` and the `payload_version`
stamp in `pickle.toml` don't know about the arrangement, producing a **standing false
warning**:

```
WARNING: payload version "0.0.0-skeleton" differs from binary "91d5be5-dirty" — run `pickle upgrade`
```

The suggested remedy (`pickle upgrade`) is exactly what the repo's self-modify policy forbids
running from a feature branch (AGENTS.md, "Self-modify policy"): the binary is the artifact
under development, and the marker block / config are dev fixtures. So the warning is permanent
noise — and permanent warnings train people to ignore `doctor`.

Make the tooling **self-host-aware**:

- **doctor**: when the installed skill directory is a symlink (a dev/self-host link), the
  payload is the linked source by construction — the `payload_version`-vs-binary comparison is
  meaningless. Detect the symlink and either skip the check or report an informational
  "self-host link detected; payload version check skipped" line instead of a WARNING. Advice
  to "run pickle upgrade" must not be emitted in this mode.
- **upgrade**: it already skips replacing a symlinked skill dir; decide and document what it
  should do with the `payload_version` stamp in that mode (likely: still stamp it, since the
  marker block is still refreshed — or skip the stamp too and say so). Whatever the choice,
  `doctor` and `upgrade` must agree so one never tells you to run the other in vain.

Soft couplings: born from the T-044 session's self-modify-guard discussion (see AGENTS.md
policy bullet added alongside this ticket). Touches the same `doctor`/`upgrade` surfaces as
T-026 (upgrade refuses legal pickle.toml) — sequence, don't run concurrently.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: pickle ticket new
