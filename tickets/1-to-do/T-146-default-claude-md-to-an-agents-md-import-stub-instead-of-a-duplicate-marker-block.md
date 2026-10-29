---
id: T-146
title: Default CLAUDE.md to an @AGENTS.md import stub instead of a duplicate marker block
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-146 — Default CLAUDE.md to an @AGENTS.md import stub instead of a duplicate marker block

## Outcome

`pickle install --agent claude` writes a two-line `CLAUDE.md` that imports `AGENTS.md` (`@AGENTS.md`)
instead of a second copy of the marker block. Claude Code still sees the full brine guidance,
the project carries one source of truth, and the same file works on every platform — no symlink,
no placeholder, no fallback.

## Description

Today `--agent claude` gives `CLAUDE.md` one of three shapes (`internal/install/install.go`
`Install`, the `switch` on `claudeMD`):

- **default** — its own copy of the marker block, duplicating `AGENTS.md`'s;
- **`--claude-symlink`** — a symlink to `AGENTS.md` (no duplication, but a placeholder file on
  checkouts without `core.symlinks`, which `doctor` warns about);
- **`--claude-symlink` where symlinks are refused** — falls back to the marker-block copy (T-143).

Two of the three duplicate content, and the third is platform-fragile. Claude Code resolves
`@path` imports in `CLAUDE.md`, so a stub that imports `AGENTS.md` gives one source of truth in a
regular file that works everywhere and leaves room for Claude-only notes under the import line.

Scope:

- **install** — when `CLAUDE.md` is absent, write the stub: a pickle sentinel comment (so pickle
  can recognise its own file) plus the line `@AGENTS.md`. An existing `CLAUDE.md` in any shape
  (user file, marker file, symlink, placeholder) is left as it is today, with a note naming the
  stub as the alternative; no forced migration.
- **refresh** — `RefreshMarkers` currently injects a marker block into *any* regular-file
  `CLAUDE.md`; it must skip a stub, or the next `upgrade`/`project add` re-creates the duplicate.
- **doctor** — `checkMarkers`'s `CLAUDE.md` branch (`internal/doctor/doctor.go`) accepts a file
  carrying the `@AGENTS.md` import line, alongside the existing symlink and marker forms.
- **uninstall** — `Uninstall` removes the stub only when it is byte-for-byte pickle's; a stub the
  user added lines to keeps their lines and loses only the sentinel + import, with a note.
- **`--claude-symlink`** — decision for refinement: keep as an opt-in, or deprecate (the stub makes
  it redundant).
- **docs** — `docs/user-manual/installation.adoc`, `concepts/project-structure.adoc`,
  `cli-reference.adoc`, the `install` help text, and the skill's *Install & register* wording
  (it names `CLAUDE.md` as installed).

Out of scope: auto-converting an existing marker-block `CLAUDE.md` to the stub — it may hold user
edits; revisit only on field demand.

This repo is unaffected: its `CLAUDE.md` is already a committed symlink to `AGENTS.md`, which
the self-modify policy leaves alone.

Soft couplings: T-143 (introduced the symlink fallback this replaces as the default path), T-121
(marker block and help text describing Claude artifacts), T-041 (marker refresh/drift, which the
stub opts out of).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-10-29 — created (TO DO). source: chat: user asked to replace the duplicated CLAUDE.md marker block with a `See @AGENTS.md` import; exploration found the default and symlink-fallback install paths duplicate AGENTS.md
