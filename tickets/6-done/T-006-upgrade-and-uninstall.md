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

**First pass verdict: REWORK — 4 blocking findings** (1 code, 3 docs; all small and confined to
`internal/cli/install.go` + `README.md`/help text). **All 4 fixed in the rework pass of 2026-07-24
(`312450c`) and verified by the scoped re-review of the same day → final verdict: PASS / DONE.**
See "Rework pass" and "Scoped re-review" below. The engine in `internal/install` is sound:
every task landed, every design decision D1–D9 was honoured, the acceptance test passes verbatim,
and `just build`/`just test`/`just lint` are clean. Reviewed on `feat/T-006-upgrade-and-uninstall`
@ b8c5d8e.

### Checklist

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage + whole-tree sweep; docs build N/A (`pickle.toml` `docs` unset) (step 4a)
- [x] Findings classified & recorded; non-blocking → T-017, T-018, T-019 + T-013 items 6–9 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep done (step 8)
- [x] Summary presented; bookkeeping committed. Child commit message/MR **withheld pending rework** (step 9)

### Implementation audit (step 2)

| Item | Result | Evidence |
|---|---|---|
| Task 1 — `Result.Removed` + `removed()` | met | `install.go:49-59` mirrors `created`/`skipped`; `Run` unchanged |
| Task 2 — `stripMarker` | met (one dead branch) | `install.go:458-500`; strips block + orphaned blank lines; never deletes the file. Absent-file branch is unreachable → T-017 item 3 |
| Task 3 — `Upgrade` | met | `install.go:108-160`; config load → payload refresh → markers → `cfg.Save`; same-version path reports "already at" |
| Task 4 — `Uninstall` + `UninstallOptions` | met | `install.go:162-252`; skill dir, `.claude` link, both marker files; `tickets/`/`pickle.toml` never referenced |
| Task 5 — CLI wiring | **partially met** | `runUninstall` parses `--dry-run`/`-n` correctly; `runUpgrade` **discards argv** → finding 1 |
| Task 6 — stale `Payload` comment | met | `cli.go:13-16` now distinguishes install/upgrade (read payload) from doctor/uninstall (don't) |
| Task 7 — tests | met for the engine | 5 new tests in `install_test.go` incl. both self-host symlink guards; `install` at 70.2% coverage. CLI layer left untested → T-013 item 9 |
| Acceptance test | met, verbatim | install → `upgrade` (exit 0, `payload_version` grepped) → 2nd `upgrade` (exit 0) → `uninstall --dry-run` (skill dir survives) → `uninstall` (removed; `tickets/` + `pickle.toml` intact) |
| `just build` / `just test` / `just lint` | met | all green; `gofmt -l` empty |
| Decisions D1–D9 | honoured | reuse of `internal/install` helpers (D1); `tickets/` untouched (D2); `pickle.toml` kept (D3); symlink guard verified by test *and* by `RemoveAll`-avoidance (D4); RemoveAll+recopy freshness proven by the stale-`junk.md` case (D5); exists-only refresh of `CLAUDE.md`/`.claude` (D6); idempotence (D7); `--dry-run`, no prompt (D8); `config.Find` error path (D9) |

Note on D5/D7: the acceptance test's "upgrade updates `payload_version`" leg is vacuous when run
with a binary whose version already matches the install stamp (both runs printed
`already at b8c5d8e`). The real version *transition* is covered by `TestUpgrade` (v1 → v2), so
the criterion is met — but the shell-level test as written cannot fail on it.

### Findings

| # | Severity | Description | Evidence | Suggestion |
|---|---|---|---|---|
| 1 | **blocking** | `runUpgrade` ignores its entire argv, so `pickle upgrade -h`, `--help`, `--bogus`, and `-n` all perform a **real, mutating upgrade** and exit 0 instead of showing help / rejecting the flag. Especially sharp because this same ticket gives `uninstall` a `-n` dry-run, so `pickle upgrade -n` is a natural (and destructive) guess. Every sibling rejects bad invocation (`runInstall` :28, `runDoctor` :136, `runUninstall` :173, `runBoardSync` board.go:39 → `exitUsage`); the only precedent for ignoring argv is `runBoardAudit`, which is read-only so the bug is inert there. Task 5's "no flags" was read as "accept and ignore anything". | `internal/cli/install.go:83` `func runUpgrade(_ []string)`. Verified against the built binary: `upgrade -h` → full refresh output, `rc=0`; `upgrade -n` → same; `uninstall -h` → usage, `rc=2` | Add `flag.NewFlagSet("upgrade", flag.ContinueOnError)` with zero flags; `fs.Parse(args)` error → `exitUsage`. Add a `{"upgrade bad flag", …, exitUsage}` row to `cli_test.go` |
| 2 | **blocking** (docs 4a.1) | The ticket's only user-facing **flag**, `uninstall --dry-run`/`-n`, is documented in no user-facing surface. `README.md:79` is an index row with no flag; there is no `## pickle uninstall` section at all, though every other flag-bearing command has one with a synopsis (`README.md:128` `pickle board sync [--dry-run]`, `README.md:157` install's). `cli.go:90`'s help line omits it. Repo-wide, `--dry-run` in docs appears only for `board sync` (`README.md:128,148`). Sole discovery path is `pickle uninstall -h`, a bare Go usage dump exiting 2. Protocol 4a.1 names "a flag" explicitly and makes missing coverage blocking. | `README.md:72-90` headings (`grep '^#'` shows sections only for audit/sync/install/ticket new/ticket move); `internal/cli/cli.go:90` | Add `## pickle upgrade` + `## pickle uninstall` sections with synopsis `pickle uninstall [--dry-run\|-n]`; add `[--dry-run]` to the help line |
| 3 | **blocking** (docs 4a.1) | The "uninstall keeps `pickle.toml`" contract — decision #3/D1, the thing that lets a later `install`/`upgrade` re-attach cleanly — is documented only in code and in the runtime summary string. Both user-facing texts promise only tickets: `README.md:79` "(keep tickets/)", `cli.go:90` "leave tickets/ intact". A user cannot learn that their config survives. | `internal/install/install.go:167-170` vs `README.md:79`, `internal/cli/cli.go:90`; summary at `internal/cli/install.go:201` | State both survivors (`tickets/` **and** `pickle.toml`) in the README section from finding 2 and in the help line |
| 4 | **blocking** (docs 4a.2, error of fact) | `README.md:94-95` says `pickle.toml` hand-edits are "normalised to a canonical layout on the next `project add|remove`" — an enumeration this ticket falsifies: `Upgrade` calls `cfg.Save("")` on every version change, normalising the file and **dropping every comment**. This repo's own `pickle.toml:1-13` is 15 lines of hand-written rationale that one `pickle upgrade` would erase. The normalisation itself is locked by D5 and is *not* a finding; the doc silently promising otherwise is. | `internal/install/install.go:154-158`; `README.md:94-95`; `pickle.toml:1-13` | Include `install`/`upgrade` in the enumeration and state that comments are not preserved. Behavioural remedy tracked separately as T-018 item 1 |
| 5 | non-blocking → **T-017** | Marker-pair detection now exists in **four** divergent copies; the newest (T-006's dry-run branch) omits the ordering check, so `--dry-run` can contradict the real run. Reproduced: on an `AGENTS.md` with `pickle:end` before `pickle:begin`, dry-run says `- AGENTS.md (marker, dry-run)`, the real run says `= AGENTS.md (no marker)`. Also: dry-run labels don't distinguish symlink vs real skill dir as the real run does; `stripMarker`'s absent-file branch is unreachable. Decision #1's "don't duplicate" was scoped to the CLI layer, so it is technically honoured — hence non-blocking. | `install.go:244` vs `:429-431`, `:475-480`, `doctor.go:139-149`; `install.go:178-179` vs `:186-191`; `install.go:466-468` unreachable from `:251` | Extract one `markerSpan` helper + export `install.HasMarkerBlock`; make dry-run labels match |
| 6 | non-blocking → **T-018** | `upgrade` silently discards hand-written content: all `pickle.toml` comments (via `cfg.Save`), and any hand-maintained bullets inside the `AGENTS.md` marker block. Verified: this repo's `AGENTS.md` keeps the child's `just build`/`just test`/`just lint` commands, WIP limits, and the self-host symlink note *inside* the managed region, while `markerBlock` emits none of them (grep for `just build`/`wip_in_development` in `install.go` → 0 matches), and `doctor.checkMarkers` only checks marker *presence*. Pre-existing in `install`'s re-run path, but T-006 ships the command whose job is refreshing it. Also folds in `markerBlock`'s stale "prefer `ticket move` **once available**" sentence, which `upgrade` now propagates into every installed project. | `install.go:154-158`, `:22-23`, `:515-544`; `AGENTS.md:22-36`; `doctor.go:111-137`; `skill/SKILL.md:93` | Preserve comments / warn; render commands+WIP from `pickle.toml` or move those bullets outside the markers |
| 7 | non-blocking → **T-013 items 6–9** | CLI-layer polish on the same setup surface: root-resolution preamble now triplicated (`runDoctor`/`runUpgrade`/`runUninstall`) while `loadConfig()` already exists for the sibling family; `runUpgrade` loads `pickle.toml` twice; `upgrade` always prints `+ .agents/skills/ticket-flow/` even on a byte-identical no-op; and removing the two stub rows left **both** handlers with zero CLI-level tests — which is exactly why finding 1 reached review uncaught. | `internal/cli/install.go:84-92,140-148,177-185`; `project.go:38-52`; `cli_test.go` | Extend the existing install-polish ticket rather than spawn a new one |
| 8 | non-blocking → **T-019** | README accuracy/redundancy: the new prose at `README.md:87-90` re-lists all eight command→ticket mappings the table's `[done: T-NNN]` tags already carry (drift risk), and "the full command surface is implemented" overstates while `install --agent` remains an accepted no-op; phased-plan tagging is inconsistent (only P5 tagged, though P1/P3 are fully delivered). | `README.md:87-90`, `:246-257`, `:180-181`; `internal/cli/install.go:27` | Let the table's tags be the single source of truth; tag P1/P3 |
| 9 | trivial — **patched inline** | `internal/cli/cli.go:1-4` package doc still claimed "In this skeleton every handler is a **stub** that reports which phase … will implement it" — in the very file whose last two stubs (and `notImplemented` itself) this commit deleted. Task 6 fixed the `Payload` comment 10 lines below and missed this. | `internal/cli/cli.go:1-4` | Reworded (comment-only), per protocol §5 |
| 10 | trivial — **patched inline** | `PLAN.md:11` still listed `doctor`/`upgrade`/`uninstall` and distribution (P5) as "Remaining", though `doctor` shipped in T-005 and P5 in T-011 (e4aaed7). Direct precedent: T-008's review patched this same line for this same reason. | `PLAN.md:11` | Reworded (prose-only), per protocol §5 |

**Verified clean (no action).** Removing the `exitUnimplement` exit code (3) strands nothing: a
repo-wide sweep of every `*.md`, `skill/`, `.agents/`, `.github/`, `justfile`, `opencode.jsonc`,
and `.pi/` found the code-3 contract published in **no** user-facing surface — only three
historical ticket lines, which are correct as plan history. There is no man page or completion
script. The README's `[done: T-NNN]` tags are all accurate. `skill/SKILL.md:216-219` describes
`pickle upgrade` in the present tense and is *made true* by this commit. Step 4a.3 (docs build)
correctly skipped: `pickle.toml:31` leaves `docs` unset.

### Impact sweep (step 8)

Swept all 10 tickets in `1-to-do/` (`2-ready/` is empty). No not-done ticket depended on
`notImplemented()`, on exit code 3, or on either command still being a stub, so deleting
`exitUnimplement` stranded nothing.

| Ticket | Verdict |
|---|---|
| T-009 (opencode wiring) | **satisfied** — its four extension targets (`install`/`doctor`/`upgrade`/`uninstall`) now all exist; empty plan, so no assumption to invalidate. Refinement should inherit T-006's probe-the-filesystem D6 pattern, since agent state is not persisted to `pickle.toml` |
| T-010 (Pi scaffold) | **patched** — it creates a new artifact class but never mentioned `upgrade`/`uninstall`; now that both ship with a *closed, hardcoded* artifact list, a scaffolded `.pi/` would be left behind by `uninstall` and never refreshed by `upgrade`. Symmetry obligation added |
| T-013 (install polish) | **patched ×2** — items 6–9 (added by this review) verified non-duplicative of items 1–5. Recorded that `Upgrade`'s `RemoveAll`-before-`copyPayload` (`install.go:126` → `:130`) makes item 2's created-vs-refreshed signal impossible as a *presence* check (must be a content comparison), and de-conflicted item 9 against this ticket's own rework row |
| T-016 (docs-readability 4b) | **patched** — its soft-coupling line put the add/remove-symmetry burden *on T-006*, presuming T-006 still open. Reworded: T-006 has shipped with a closed scope, so the burden sits with T-016/T-009/T-010 |
| T-017 | **patched** (precision) — added the soft-coupling note that T-013 items 1/2/7/8 edit the same `injectMarker`/`Result` surfaces |
| T-012, T-014, T-015, T-018, T-019 | **no impact** — all cited refs re-verified against the post-T-006 tree. (Noted only, no patch: T-012 item 2's `config.Render` escaping defect is now reachable from `pickle upgrade` via `cfg.Save`, widening its blast radius without changing its fix) |

Forward-looking note for whoever lands the rework: inserting `## pickle upgrade`/`## pickle
uninstall` sections into `README.md` will shift every line below `## pickle install`, breaking
T-019's exact refs (`README.md:180-181`, `:246-257`). Re-anchor them after the rework rather than
now.

### Rework scope (the entire scope — nothing else)

Findings **1–4** only, on the same branch:

1. `runUpgrade`: parse argv with an empty flag set; unknown flag / `-h` → `exitUsage` (not a
   mutation). Add the `cli_test.go` bad-flag row.
2. `README.md`: add `## pickle upgrade` and `## pickle uninstall` sections; document
   `[--dry-run|-n]`.
3. State that `uninstall` preserves **`tickets/` and `pickle.toml`** in both the README section
   and the `cli.go` help line.
4. `README.md:94-95`: include `install`/`upgrade` among the commands that normalise
   `pickle.toml`, and note comments are not preserved.

Then re-run the acceptance test + `just build`/`just test`/`just lint` and return to
`4-in-review/` for a **scoped re-review** of findings 1–4 only.

### Rework pass — 2026-07-24 (all 4 blocking findings fixed)

| # | Fix | Evidence |
|---|---|---|
| 1 | `runUpgrade` now takes `args` and parses them through an empty `flag.NewFlagSet("upgrade", flag.ContinueOnError)`; a parse error (`-h`/`--help`/unknown flag) returns `exitUsage`, and a stray positional (`fs.NArg() > 0`) reports `unexpected argument %q` and returns `exitUsage`. No mutation happens before parsing — the flag set is the first statement in the handler. | `internal/cli/install.go:83-95`. Verified against the rebuilt binary: `upgrade` with each of `-h`, `--help`, `--bogus`, `-n`, `extra` → **`rc=2`**, and `pickle.toml` byte-identical (`diff -q`) after all five calls; bare `pickle upgrade` still `rc=0`. Four new rows in `internal/cli/cli_test.go` (`upgrade bad flag`, `upgrade help flag`, `upgrade stray arg`, `uninstall bad flag`) |
| 2 | Added `## pickle upgrade` and `## pickle uninstall` README sections, each with a synopsis line matching the existing convention; the uninstall synopsis is `pickle uninstall [--dry-run\|-n]` and a dedicated paragraph documents the flag. The command-table row also now shows `[--dry-run]`, and the help line documents it. | `README.md:185-227` (new sections), `README.md:79` (table row), `internal/cli/cli.go:90-91` (help line). `pickle help` output re-checked |
| 3 | Both survivors are now stated wherever the command is described: README section ("**`tickets/` and `pickle.toml` are both left intact**", with *why* — a later `install`/`upgrade` re-attaches to the same configuration), the table row ("keep tickets/ + pickle.toml"), and the help line ("leave tickets/ and pickle.toml intact"). | `README.md:221-223`, `README.md:79`, `internal/cli/cli.go:90-91` |
| 4 | Rewrote the Configuration paragraph: it now names **every** writer that re-renders the file — `install`, `upgrade` (when it stamps a new `payload_version`), and `project add|remove` — and states in bold that the re-render **does not preserve comments**, advising notes be kept outside `pickle.toml`. The `## pickle upgrade` section repeats the warning with a link back to Configuration. | `README.md:94-97`, cross-referenced from `README.md:203-205` |

**Verification.** Acceptance test re-run verbatim and green (install → `upgrade` + `payload_version`
grep → 2nd `upgrade` → `uninstall --dry-run` with skill dir surviving → `uninstall` with `tickets/`
+ `pickle.toml` intact), all `rc=0`. `just build` / `just test` / `just lint` clean, `gofmt -l`
empty, `board audit` 0 errors, `doctor` 0 errors. `TestUpgrade` still covers the real v1→v2
version transition that the shell-level test cannot exercise.

**Scope discipline.** Nothing outside findings 1–4 was touched. Two things deliberately *not*
fixed here, for the re-reviewer:

- `runUninstall` still ignores stray **positional** args (`pickle uninstall foo` uninstalls). Same
  class as finding 1 but a different command, so out of this pass's scope; a `uninstall bad-flag`
  test row was added, but the positional gap is left for **T-013 item 9** (which owns the
  remaining CLI-layer test coverage) to close.
- The behavioural remedies for comment loss and marker-body replacement remain with **T-018**;
  this pass only documents the current behaviour truthfully, as finding 4 required.

### Scoped re-review — 2026-07-24 · verdict: PASS → DONE

Scope per protocol §1: **findings 1–4 only**, not a re-audit of the feature. Re-reviewed on
`feat/T-006-upgrade-and-uninstall` @ `ea4f95d`. Rework diff is `312450c` (4 files, +69/−5) —
inspected in full and confined to the four findings; no scope creep, no collateral edits.

| # | Re-verified | Evidence (independent of the rework record) |
|---|---|---|
| 1 | **fixed** | `internal/cli/install.go:83-95`: flag set is the handler's first statement, so no mutation can precede parsing. Empirically, against a freshly built binary on a throwaway install: `upgrade` with `-h`, `--help`, `--bogus`, `-n`, `--dry-run`, `extra`, and `-- extra` → **all `rc=2`**, and `pickle.toml` byte-identical (`diff -q`) after all seven; bare `upgrade` → `rc=0`. Now **consistent with every sibling**: `install`/`upgrade`/`uninstall`/`doctor` all answer `-h` with `Usage of <cmd>:` + `rc=2`. Positional path gives the better message (`pickle upgrade: unexpected argument "extra"`) and `-- extra` correctly routes to it |
| 2 | **fixed** | `README.md:185-227` adds `## pickle upgrade` + `## pickle uninstall`, synopsis style matching the existing sections; `pickle uninstall [--dry-run\|-n]` at `:212` with a dedicated paragraph at `:225`. Table row `:79` and help line `internal/cli/cli.go:90-91` both carry `[--dry-run]`; `pickle help` output re-checked live. Flag still works after the rework: `--dry-run` and `-n` → `rc=0`, skill dir survives; `--bogus` → `rc=2` |
| 3 | **fixed** | Both survivors stated in all three surfaces: `README.md:221-223` (in bold, **with the rationale** — a later `install`/`upgrade` re-attaches), table row `:79` ("keep tickets/ + pickle.toml"), help line `cli.go:90-91`. Matches `internal/install/install.go:167-170`'s actual behaviour |
| 4 | **fixed, and the enumeration is provably exhaustive** | `README.md:94-97` now names `install`, `upgrade`, `project add\|remove` + bold "does not preserve comments". Verified exhaustive by sweeping **every** `pickle.toml` writer: `rg '\.Save\('` → `install.go:155` (Upgrade), `install.go:369` (install), `project.go:95`/`:137` (project add/remove) — nothing else. The parenthetical "(when it stamps a new `payload_version`)" is **precise**, not hedging: `Upgrade` returns at `install.go:150-153` *before* `cfg.Save`, so a same-version re-run genuinely leaves the file alone. Cross-ref anchor `#configuration--pickletoml` validated against actual headings |

**Tests are load-bearing, not decorative** (protocol §3). Mutation-tested in a throwaway clone:
reverting `runUpgrade` to `func runUpgrade(_ []string)` makes exactly the new rows fail with the
pre-fix symptom — `Run([upgrade --bogus]) = 0, want 2`, same for `-h` and `extra`. So the guard
cannot silently regress.

**Acceptance test** re-run verbatim and green (`payload_version = "ea4f95d-dirty"`, 2nd upgrade
`rc=0`, dry-run preserves the skill dir, real uninstall removes it while `tickets/` + `pickle.toml`
remain). `just build` / `just test` (9 packages) / `just lint` / `gofmt -l` all clean;
`board audit` 0 errors; `doctor` 0 errors, 1 expected payload-version warning.

**Findings 5–10 unchanged** — the rework touched none of their evidence; T-017/T-018/T-019 and
T-013 items 6–9 remain the owners. No new blocking findings. One new non-blocking observation:

| # | Severity | Description | Evidence | Routing |
|---|---|---|---|---|
| 11 | non-blocking → **T-018 item 1** | Finding 4's *twin* ships inside the generated file: `config.Render` hardcodes `# … normalised to this layout on the next pickle project add\|remove.` — the same wrong enumeration the README just fixed, in a surface written into **every** installed project, read first in the user's own `pickle.toml`, present in the very file whose comments were just erased, and silent about comment loss. Not held against T-006: the string predates it (T-001 `3dc0c26`), lives in a package T-006 never touched, the README (what a user consults) is now correct, and every T-018 remedy rewrites this header anyway — fixing it here means writing it twice. | `internal/config/config.go:196-198`; proven by the mutation run, which replaced this repo's 16 lines of bootstrap rationale with that header | T-018 item 1 extended with the exact requirement |

**Considered and deliberately not flagged.** `PLAN.md:223` still reads "leave `tickets/` intact"
(omitting `pickle.toml`, and `--dry-run`). It is a one-clause roadmap summary, not a user-facing
docs surface — the first-pass review examined `PLAN.md` closely enough to patch `:11` as trivial
finding 10 and scoped finding 3 to the README + help line on exactly this basis. Omission, not
contradiction; re-scoping it now would be inconsistent with that call.

**Deferred gap re-confirmed as latent, not shipped-wrong.** `runUninstall` still accepts stray
positionals (`pickle uninstall foo` uninstalls) — the rework left it out of scope, correctly: it is
a different command, undocumented as taking arguments, and no documented invocation misbehaves.
Now explicitly owned by **T-013 item 9** with the fix site named, rather than left as a note in
prose.

**Bookkeeping executed by this re-review** (step 7/8 — the first pass's forward-looking note came
due):

- **T-019 re-anchored.** The new README sections shifted every line below `## pickle install`, so
  item 1's `:180-181` → `:182-183` and item 2's `:246-257` → `:292-303`, `:250-252` → `:296-298`.
  `:87-90`/`:75-83` were above the insertion and needed no change. A dated note records why.
- **T-013 item 9 corrected.** Its scope note claimed T-006's rework adds "the single
  `upgrade bad flag` row"; it landed four, `uninstall bad-flag` among them, so the note overstated
  the remaining work. Rewritten, plus the `uninstall` positional gap, plus a **safety** finding
  from the mutation run: `go test` runs with cwd inside this repo, so `os.Getwd()` → `config.Find`
  resolves to pickle's own root — the mutant's rows performed a real upgrade on the clone
  (`pickle.toml` + `AGENTS.md` rewritten). A future happy-path `uninstall` row without a temp root
  would strip this repo's own install. The four current rows are safe only because they fail
  during parsing, before `config.Find`.

### Re-review checklist

- [x] Rework diff inspected in full; scope confined to findings 1–4 (step 1)
- [x] Findings 1–4 each independently re-verified — not taken from the rework record (step 2)
- [x] Acceptance test re-run verbatim; `just build`/`test`/`lint`, `gofmt`, `board audit`, `doctor` clean (step 2)
- [x] Quality: new tests mutation-tested; sibling-consistency of `-h` confirmed (step 3)
- [x] Consistency: every `pickle.toml` writer swept for finding 4's enumeration; README anchors validated (step 4)
- [x] Docs: coverage of the flag + both contracts confirmed in README, table, and help; docs build N/A (`docs` unset) (step 4a)
- [x] New finding 11 classified non-blocking → T-018 item 1 (step 5)
- [x] No blocking findings → ticket moved to `6-done/`; `## History` appended (step 6b)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep: T-019 re-anchored, T-013 item 9 corrected + hardened (step 8)
- [x] Summary + child commit message & MR attributes presented for approval; bookkeeping committed (step 9)

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
- 2026-07-24 — TO DO → READY: plan complete
- 2026-07-24 — READY → IN DEVELOPMENT: picked up
- 2026-07-24 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-24 — IN REVIEW → REWORK: review: 4 blocking findings (upgrade argv ignored; docs coverage for --dry-run, pickle.toml survival, normalisation)
- 2026-07-24 — REWORK → IN REVIEW: findings fixed
- 2026-07-24 — IN REVIEW → DONE: re-review PASS: findings 1-4 verified fixed; finding 11 -> T-018
