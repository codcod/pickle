---
id: T-013
title: install polish (marker spacing, summary labels, cli tests, --agent)
project: pickle
depends-on: [T-004]
impact: low
complexity: low
cost: S
---

# T-013 — install polish (marker spacing, summary labels, cli tests, --agent)

## Description

Non-blocking follow-ups surfaced by the T-004 review. `pickle install` (T-004) works and is
board-audit-clean; these are cohesive quality/polish items on the same install surface, none of
which change the golden path:

1. **`injectMarker` append spacing** — the separator computation in
   `internal/install/install.go` has a dead `else if` branch (it sets the same value as the
   default), and a file ending in exactly `\n\n` gets an extra blank line before the appended
   block. Simplify to: ensure exactly one blank line precedes the appended marker block
   regardless of the file's trailing whitespace. Add a table test over trailing-whitespace
   variants (no newline / one `\n` / two `\n`).
2. **Summary label accuracy** — on a re-run, `copyPayload` and `scaffoldTickets` report `+`
   (created) even though they only refreshed/ensured already-present files. Distinguish
   *created* from *refreshed* (or report `=` when nothing changed) so the CLI summary is honest.
3. **cli `runInstall` coverage** — the `internal/install` core is tested (~70%), but the
   `internal/cli` wrapper (`runInstall`: flag parsing, defaults, self-check wiring) is only
   exercised by manual acceptance. Add a table-driven test that runs `runInstall` against a
   temp cwd (via a test payload FS) and asserts the exit code + that the produced tree passes
   `audit.Audit`.
4. **`--agent` handling** — the flag is currently accepted but is a pure no-op (Claude is on by
   default; pi/opencode wiring is deferred to T-010/T-009). Until those land, either reject
   unknown `--agent` values with a clear "not yet supported" message, or document it as reserved
   and warn — so a user passing `--agent pi` isn't silently misled.
5. *(moved to T-014)* board-cell escaping — the child-name substitution into `BOARD.md` shares
   the unescaped-`|`/newline gap now tracked, generally, as **T-014 item 2** (cell escaping at
   the `board.renderRow` choke point). Fix it there rather than here.

All items are input-hardening / polish on the install surface, hence non-blocking.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: T-004 review (non-blocking findings); via pickle ticket new
