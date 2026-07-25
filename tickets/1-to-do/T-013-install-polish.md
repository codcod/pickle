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

Added by the **T-006 review** (same `internal/cli` setup-command surface, so fixed here rather
than in a new ticket):

6. **Project-root resolution is triplicated in the setup commands.** The
   `os.Getwd()` → `config.Find(wd)` → `filepath.Dir(cfgPath)` preamble is now copy-pasted three
   times in `internal/cli/install.go` (`runDoctor` ~:140-148, `runUpgrade` ~:84-92,
   `runUninstall` ~:177-185), while the sibling family already funnels through `loadConfig()`
   (`internal/cli/project.go:38-52`, used by `board.go:43,72`). Extract one helper (e.g.
   `projectRoot() (string, int)`) and route all three through it. Error wording and exit codes
   are already consistent, so this is pure redundancy removal.

7. **`runUpgrade` loads `pickle.toml` twice.** It calls `config.Load` once for `prevVersion`
   (`internal/cli/install.go:~94`) and again after the upgrade for the audit self-check
   (`~:112`). Have `install.Upgrade` report the previous version in its `Result` instead, and
   drop the first load.

8. **Extend item 2's "summary label accuracy" to `upgrade`.** `pickle upgrade` always prints
   `+ .agents/skills/ticket-flow/` because it routes through `copyPayload`, even when the payload
   is byte-identical — so a no-op upgrade looks like it changed the tree. Whatever
   created-vs-refreshed distinction item 2 lands must cover the `Upgrade` path too.
   **Mechanism constraint this imposes on item 2:** `Upgrade` does `os.RemoveAll` on the skill dir
   (`internal/install/install.go:126`) *before* calling `copyPayload` (`:130`), so a presence-based
   "did it already exist?" check always reports "created" on the upgrade path. The
   created-vs-refreshed signal must therefore be a **content** comparison captured before the
   removal, not an existence check. (`scaffoldTickets` is unaffected — `Upgrade` never calls it.)

9. **Extend item 3's cli-coverage to `runUpgrade`/`runUninstall`.** The T-006 review removed the
   two `exitUnimplement` stub rows from `internal/cli/cli_test.go` and added no replacement, so
   both handlers now have **zero** CLI-level tests (the `internal/install` package tests cover
   the engines only). This is precisely why T-006's blocking argv defect (`pickle upgrade -h`
   performing a real upgrade) reached review uncaught. Add table-driven cases for both against a
   temp root, including bad-flag → `exitUsage` and `--dry-run` → tree unchanged.
   **Scope note (updated 2026-07-24, T-006 scoped re-review).** T-006's rework landed **four**
   argv-rejection rows, not one: `upgrade bad flag`, `upgrade help flag`, `upgrade stray arg`, and
   `uninstall bad flag` (`internal/cli/cli_test.go:17-20`). Those are done. This item's remaining
   scope is therefore:
   - `uninstall --dry-run`/`-n` against a temp root, asserting the tree is unchanged, plus
     happy-path exit codes for both handlers. **The temp root is a hard safety requirement, not
     tidiness.** `go test` runs with the cwd inside this repo, so `runUpgrade`/`runUninstall`'s
     `os.Getwd()` → `config.Find` resolves to **pickle's own root**: any row that gets past argv
     parsing operates on the working repo. Demonstrated during the T-006 re-review — a mutant
     build whose argv guard was removed made the existing rows perform a real upgrade, rewriting
     `pickle.toml` (16 comment lines gone) and `AGENTS.md`'s marker block. A happy-path
     `uninstall` row without a temp root would delete `.agents/skills/ticket-flow` and strip the
     repo's own marker block. The four rows above are safe *only* because they fail during parsing,
     before `config.Find`. Use `t.Chdir(tmp)` (or the harness from item 3) for anything else.
   - **`runUninstall` ignores stray positionals** — `pickle uninstall foo` performs a real
     uninstall instead of `exitUsage`. This is finding 1's defect class surviving in the sibling
     handler: `runUninstall` parses flags but never checks `fs.NArg()`. T-006's rework deliberately
     left it out of scope (different command), and the re-review confirmed it is latent, not
     shipped-wrong-behaviour for a documented invocation. Fix it with the `fs.NArg() > 0` guard
     from `internal/cli/install.go:90-93` and cover it with an `uninstall stray arg` row.

All items are input-hardening / polish on the install surface, hence non-blocking.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: T-004 review (non-blocking findings); via pickle ticket new
