---
id: T-013
title: install polish (marker spacing, summary labels, cli tests, --agent)
project: pickle
depends-on: [T-004]
spawned-by: []
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
4. ~~**`--agent` handling**~~ *(obsoleted 2026-07-26 by T-009: `--agent` is now fully parsed
   and wired — unknown values are a usage error, `--no-claude` warns deprecated. Nothing left
   here.)* **Replaced by a folded T-009 review finding (F4):** the top-level `pickle help`
   text still says install lays the skill down "for detected agents"
   (`internal/cli/cli.go` usage string) — but there is **no autodetection** (a T-009 locked
   decision): the set is exactly what `--agent` names, default `claude`. Reword the help line
   (e.g. "for the agents named by --agent, default claude").
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
   (`internal/install/install.go:129`) *before* calling `copyPayload` (`:133`), so a presence-based
   "did it already exist?" check always reports "created" on the upgrade path. The
   created-vs-refreshed signal must therefore be a **content** comparison captured before the
   removal, not an existence check. (`scaffoldTickets` is unaffected — `Upgrade` never calls it.)
   **Two more labels, added 2026-07-25 by the T-018 review (non-blocking finding 11):**
   `internal/install/install.go:168` reports an *in-place single-line edit* of `pickle.toml` as
   `+` (created) — the same created-vs-refreshed dishonesty on a new call site; and
   `internal/cli/cli.go:87-88`'s one-line help for `upgrade` never mentions that it stamps
   `payload_version` in `pickle.toml`, which the user manual does mention (`docs/user-manual/cli-reference.adoc`, upgrade section;
   moved out of `README.md` by T-047) and which is the file most
   users care about.
   **Three more, added 2026-07-25 by the T-018 re-review (non-blocking findings R8, R9):**
   - **`verifyStampedVersion` is 100% covered and 0% bound.** `install.go:173-184` is unit-tested
     (`TestVerifyStampedVersion`, three real cases) but **deleting its call site at `:165-167`
     fails no test** — mutation-confirmed, and disclosed by T-018's own rework record. It needs a
     seam: an injectable writer that reports success on a write which reads back wrong. The
     cheapest shape is to make the stamp step a function field on the install/upgrade options
     struct so a test can substitute a lying writer.
   - **`config.Config{}` literals render the *unsafe* commit wording.** The commit-policy default
     now lives in `applyDefaults`, which only runs on the `config.Load` path, so a zero-value
     `Config{}` renders "**not** publish-gated" — verified. No production path does this today
     (`writeConfig` at `install.go:375` is the only literal and sets both explicitly), but nothing
     prevents it, and `writeConfig` **hardcodes `true`/`true` instead of the
     `DefaultOverarchingAuto`/`DefaultChildPublishGated` constants** three files away, so the two
     can silently diverge. Use the constants; consider a constructor so the safe default is not
     reachable-by-omission.
   - **`upgrade` exits 1 from a partially-applied upgrade.** The skill payload is re-copied and
     both marker blocks are rewritten *before* the `pickle.toml` stamp, so a stamp refusal leaves
     the tree half-upgraded while the command reports failure. The file-level contract ("refuse
     and change nothing") holds; the command-level one does not. Either stamp first or state the
     ordering in the docs.

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
     *(Updated 2026-07-25: T-018 makes the `pickle.toml` half of that mutant's damage much
     smaller — `upgrade` now edits one line instead of re-rendering — but the marker-block and
     `uninstall` hazards are unchanged, so the temp-root requirement stands.)*
   - **`runUninstall` ignores stray positionals** — `pickle uninstall foo` performs a real
     uninstall instead of `exitUsage`. This is finding 1's defect class surviving in the sibling
     handler: `runUninstall` parses flags but never checks `fs.NArg()`. T-006's rework deliberately
     left it out of scope (different command), and the re-review confirmed it is latent, not
     shipped-wrong-behaviour for a documented invocation. Fix it with the `fs.NArg() > 0` guard
     from `internal/cli/install.go:90-93` and cover it with an `uninstall stray arg` row.

10. **`injectMarker` hard-codes `0o644` on `AGENTS.md`/`CLAUDE.md`** (added 2026-07-25 by T-018,
    gate finding N10). Four writers in `internal/install/install.go` (`:422`, `:441`, `:455`,
    `:502`) call `os.WriteFile(..., 0o644)`, so every install/upgrade/uninstall silently resets a
    non-default permission on files the user owns. Same family as T-018 (silent loss of user
    state) but on this ticket's `injectMarker` surface, so T-018 deliberately left it alone.
    `internal/config` now has an unexported atomic, mode-preserving `writePreservingMode` helper
    (added by T-018) worth mirroring — see T-012 item 7, which proposes the same fix for
    `config.Save`.

All items are input-hardening / polish on the install surface, hence non-blocking.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: T-004 review (non-blocking findings); via pickle ticket new
- 2026-07-25 — broadened by the T-018 gate + review: added injectMarker file-mode preservation (item 10) and two more summary/help labels (item 8)
- 2026-07-25 — broadened again by the T-018 re-review (R8/R9): verifyStampedVersion test seam, Config{} default leak + writeConfig constants, partially-applied upgrade on refusal (item 8)
- 2026-07-25 — re-anchored by the T-018 S1 re-review: install.go refs (:126 -> :129, :130 -> :133, :162 -> :168, :180-191 -> :173-184, :163-168 -> :165-167)
- 2026-07-26 — item 4 rewritten by the T-009 review (impact sweep): the "--agent is a no-op" premise is obsolete; replaced with the folded F4 finding (`pickle help` says "detected agents", contradicting T-009's no-autodetection decision). Line refs in items 6–10 may have shifted (~200 lines added to internal/install/install.go by T-009); re-anchor at refinement
- 2026-07-26 — patched by the T-047 review (impact sweep): README passage it cited moved to docs/user-manual/cli-reference.adoc
