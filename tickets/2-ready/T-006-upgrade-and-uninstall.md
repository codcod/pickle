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

> **T-004 artifact set (what `upgrade` refreshes / `uninstall` removes)**, from `internal/install`:
> `.agents/skills/ticket-flow/` (copied payload), `.claude/skills/ticket-flow` symlink, the
> `<!-- pickle:begin -->`/`<!-- pickle:end -->` marker block in `AGENTS.md`/`CLAUDE.md`, the
> seven `tickets/<status>/.gitkeep` files, and `pickle.toml` (`payload_version`). `upgrade`
> should reuse `install`'s payload-copy + `injectMarker` (they are already idempotent); consider
> extracting a shared `markerBlock`/inject helper so the three commands stay in lockstep.
> `uninstall` strips the marker block (leaving surrounding prose) and the skill dir/symlinks,
> never touching `tickets/`.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle`'s sole child is the repo root (`.`), so the branch is cut in this repo:

```
git checkout main
git checkout -b feat/T-006-upgrade-and-uninstall
```

Local WIP commits encouraged; **no push / MR without explicit user approval** (child is
publish-gated — see `pickle.toml` / `AGENTS.md`).

### Prerequisite gate (hard)

- T-004 (install) is in `6-done/` and merged to `main` (33f05e3) — satisfied.
- T-005 (doctor) is merged to `main` (b199215): this ticket builds on the exported
  `install` constants (`SkillDir`, `MarkerBegin/End`, `ClaudeSkillLink/Target`). Not a hard
  `depends-on` (no code dependency on doctor itself), but the branch must fork from a `main`
  that has b199215. Confirm `git log --oneline -1 main` shows the doctor merge before starting.
- Clean working tree on the new branch.

### Confirmed design decisions (do not deviate without asking)

1. **Both commands live in `internal/install` and reuse its helpers.** Add
   `func Upgrade(payload fs.FS, root, payloadVersion string) (Result, error)` and
   `func Uninstall(root string, opts UninstallOptions) (Result, error)` next to `Run`, reusing
   `copyPayload`, `injectMarker`, `markerBlock`, `ensureSymlink`, and the exported
   `SkillDir`/`MarkerBegin`/`MarkerEnd`/`ClaudeSkillLink`/`ClaudeSkillTarget` constants — so the
   three commands stay in lockstep. Add one new helper, `stripMarker` (the inverse of
   `injectMarker`). Do **not** duplicate the marker/payload logic in the CLI layer.
2. **`upgrade` never touches `tickets/` or the board.** It refreshes the skill payload +
   marker block and rewrites `payload_version` in `pickle.toml`; nothing under `tickets/` is
   read or written.
3. **`uninstall` leaves `tickets/` AND `pickle.toml` intact** (D1). It removes only: the skill
   dir, the `.claude/skills/ticket-flow` symlink, and the pickle marker block(s). Config and
   instance data survive so a later `pickle install`/`upgrade` re-attaches cleanly.
4. **Self-host / symlink safety is mandatory (D2).** When removing or refreshing
   `SkillDir`: if it is a **symlink**, operate on the link only (`os.Remove` for uninstall;
   skip for upgrade) — **never `os.RemoveAll` a symlink target** (that would delete the repo's
   real `skill/`). Only a **real** directory is `RemoveAll`ed. Mirror `copyPayload`'s existing
   `os.Lstat`→`ModeSymlink` guard.
5. **`upgrade` freshness (D3).** For a real skill dir, `os.RemoveAll` then `copyPayload` so
   files deleted from the new payload do not linger; for a symlinked skill dir, skip the copy
   (self-host) exactly as `copyPayload` already does. Re-inject the marker block via
   `injectMarker` (idempotent), then set `cfg.PayloadVersion` and `cfg.Save` (this normalizes
   `pickle.toml` to the canonical layout — expected, it is tool-managed, D5).
6. **Only touch artifacts that exist (D4).** `upgrade`: refresh `AGENTS.md` always; refresh
   `CLAUDE.md` **only if it exists as a regular file** (a symlink `CLAUDE.md → AGENTS.md` is
   left alone); re-`ensureSymlink` the `.claude` link **only if it already exists**. `uninstall`:
   strip the marker from `AGENTS.md`; for `CLAUDE.md`, remove it if it is a symlink, else strip
   its marker if it has one; remove the `.claude/skills/ticket-flow` symlink if present. Absent
   artifacts are skipped, never an error (idempotent).
7. **Both idempotent.** Re-running `upgrade` at the current version reports "already at
   `<ver>`" (still safe to re-run); re-running `uninstall` on an already-clean tree reports
   nothing removed. Both return the existing `Result{Created,Skipped}` shape (add removed
   entries to `Result` — see Task 1).
8. **`uninstall --dry-run` (D6)** lists what *would* be removed/stripped without touching the
   tree; no interactive confirmation prompt (the command is agent-run). `upgrade` needs no
   dry-run.
9. **Error when not a pickle project.** Both resolve the root via `config.Find` from the cwd;
   if none is found, the CLI reports the error and exits non-zero (like `board audit`/`doctor`).

### Tasks

#### Task 1 — `Result` gains a removed list
In `internal/install/install.go`, add `Removed []string` to `Result` plus a
`func (r *Result) removed(f string)` accumulator (mirroring `created`/`skipped`). Leave `Run`
unchanged.

#### Task 2 — `stripMarker` helper
Add `func stripMarker(path string, res *Result) error` to `internal/install/install.go`: read
`path`; if absent, skip. If it contains a `MarkerBegin … MarkerEnd` pair, cut the wrapped block
**and** any now-orphaned blank lines around it, write the remainder back (record
`"<file> (marker stripped)"`); if no marker pair, skip. Do not delete the file itself.

#### Task 3 — `Upgrade`
Add `func Upgrade(payload fs.FS, root, payloadVersion string) (Result, error)`:
1. Load `pickle.toml` (`config.Load(filepath.Join(root, config.FileName))`); error if missing.
2. Refresh the payload per decision #5 (RemoveAll real dir + `copyPayload`; skip symlink).
3. `injectMarker(AGENTS.md, "Ticket flow", markerBlock(cfg), &res)`; if `CLAUDE.md` exists as
   a regular file, inject there too; if the `.claude` link exists, re-`ensureSymlink` it.
4. Record the version transition; set `cfg.PayloadVersion = payloadVersion`; `cfg.Save("")`.
   If the version was already equal, still refresh payload/marker and note "already at <ver>".
5. Return the `Result`.

#### Task 4 — `Uninstall`
Add `type UninstallOptions struct { DryRun bool }` and
`func Uninstall(root string, opts UninstallOptions) (Result, error)`:
1. Remove `SkillDir` per decision #2/#4 (symlink → `os.Remove`; real dir → `os.RemoveAll`;
   absent → skip).
2. Remove the `.claude/skills/ticket-flow` symlink if present (leave the `.claude` dirs).
3. `stripMarker(AGENTS.md)`; for `CLAUDE.md`: symlink → remove; regular file → `stripMarker`.
4. Never touch `tickets/` or `pickle.toml`.
5. Under `opts.DryRun`, compute and record the same entries into `Result.Removed` but perform
   no filesystem mutation (guard each op).

#### Task 5 — wire the CLI (`internal/cli/install.go`)
Replace the `runUpgrade`/`runUninstall` stubs:
- `runUpgrade(args)`: no flags; `config.Find(wd)` → `root`; `install.Upgrade(Payload, root,
  Version)`; print `+`/`=` lines (created/skipped) like `runInstall`; then a post-upgrade
  `audit.Audit` self-check (as `runInstall` does) so a broken result exits non-zero; print a
  version-transition summary.
- `runUninstall(args)`: parse `--dry-run`/`-n` (`flag.NewFlagSet`, unknown → `exitUsage`);
  `config.Find(wd)` → `root`; `install.Uninstall(root, install.UninstallOptions{DryRun})`;
  print `-` lines for `Result.Removed` (and `=` for skipped); a summary noting dry-run.
Remove the `notImplemented("P2", …)` calls.

#### Task 6 — update the stale `Payload` comment
In `internal/cli/cli.go`, the `Payload` doc comment currently says upgrade "will [read from it]
once implemented" — update it now that both install and upgrade read the embedded payload
(doctor/uninstall do not).

#### Task 7 — tests (`internal/install/*_test.go`)
Add to the existing `install` test package (reuse the `payloadRoot()`/`os.DirFS` helper):
- **upgrade**: install at `v1`; `Upgrade(payload, root, "v2")`; assert `pickle.toml`
  `payload_version == "v2"`, the skill files still present, and the AGENTS.md marker present.
  Add a stale-file case: drop a junk file into a **real** skill dir, upgrade, assert it is
  gone. Idempotent case: `Upgrade` twice at the same version → no error.
- **upgrade self-host guard**: make `SkillDir` a symlink to an external temp dir (as
  `TestSelfHostSymlinkGuard` does), upgrade, assert the symlink and its target survive.
- **uninstall**: install (with Claude), `Uninstall(root, {})`; assert `SkillDir` gone,
  `.claude/skills/ticket-flow` gone, AGENTS.md has no marker pair, **and** `tickets/` +
  `pickle.toml` still exist. Dry-run case: `Uninstall(root, {DryRun:true})` reports removals
  but the artifacts still exist afterward.
- **uninstall self-host guard**: `SkillDir` as a symlink to an external temp dir → uninstall
  removes the link but the external target survives.
- In `internal/cli/cli_test.go`, remove the `"upgrade stub"` and `"uninstall stub"` rows
  (both now depend on a project root, like `doctor`; behaviour is covered by the `install`
  package tests).

### Acceptance test

Run from the repo root (use a throwaway install dir — **never uninstall the repo itself**):

```
just build
TMP=$(mktemp -d)
( cd "$TMP" && git init -q && "$OLDPWD/pickle" install --project demo )   # scaffold a victim

# upgrade is idempotent and updates payload_version:
( cd "$TMP" && "$OLDPWD/pickle" upgrade && grep payload_version pickle.toml )
( cd "$TMP" && "$OLDPWD/pickle" upgrade )   # second run: clean, exit 0

# uninstall --dry-run shows removals but changes nothing; real run removes them:
( cd "$TMP" && "$OLDPWD/pickle" uninstall --dry-run && test -d .agents/skills/ticket-flow )
( cd "$TMP" && "$OLDPWD/pickle" uninstall && test ! -e .agents/skills/ticket-flow && test -d tickets && test -f pickle.toml )
rm -rf "$TMP"

just test    # all packages green, incl. new install upgrade/uninstall tests
just lint    # go vet + gofmt clean
```

Expected: `upgrade` exits 0 and `payload_version` matches the binary; a second `upgrade` is a
clean no-op; `uninstall --dry-run` leaves the skill dir in place; the real `uninstall` removes
the skill dir + symlinks + marker while `tickets/` and `pickle.toml` remain; `just test` /
`just lint` clean.

### Docs update (mandatory when user-facing)

Update `README.md`:
- flip the `pickle upgrade` and `pickle uninstall` rows from `[P2]` to `[done: T-006]`.
- update the prose after the command block: with `doctor` (T-005) and now `upgrade`/`uninstall`
  done, **no command stubs remain** — remove the "remaining commands … are stubs" sentence (or
  reword to state the full surface is implemented).

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` clean.
2. `README.md` updated.
3. Write a summary (files touched, decisions, anything deferred).
4. Suggested Conventional Commit:

   ```
   feat(cli): add upgrade and uninstall (T-006)

   Add `pickle upgrade` (refresh skill payload + marker block, rewrite
   payload_version; never touches tickets/) and `pickle uninstall` (remove
   skill dir + .claude/CLAUDE symlinks + marker blocks, with --dry-run;
   leaves tickets/ and pickle.toml). Both reuse internal/install helpers and
   honour the self-host symlink guard.
   ```

5. Commit locally on `feat/T-006-upgrade-and-uninstall`; **do not push or open an MR without
   user approval**. Present the commit message, then move the ticket to IN REVIEW.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
- 2026-07-24 — TO DO → READY: plan complete
