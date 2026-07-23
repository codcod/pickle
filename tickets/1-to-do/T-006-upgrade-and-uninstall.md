---
id: T-006
title: upgrade + uninstall
project: pickle
depends-on: [T-004]
impact: medium
complexity: medium
cost: M
---

# T-006 — upgrade + uninstall

## Description

Implement `pickle upgrade` and `pickle uninstall`.

- **upgrade** — rewrite the installed skill payload and the `AGENTS.md`/`CLAUDE.md` marker block
  to this binary's payload version (compare versions; update `payload_version` in `pickle.toml`).
  **Never touches `tickets/` or the board contents.** Idempotent.
- **uninstall** — remove the installed skill dir + symlinks and strip the marker block, leaving
  `tickets/` intact. Idempotent.

Both operate only on the project's own tree (per-project install). Needs T-004. Phase P2.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
