---
id: T-013
title: install polish (marker spacing, summary labels, cli tests, --agent)
project: pickle
depends-on: [T-004]
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-013 — install polish (marker spacing, summary labels, cli tests, --agent)

## Outcome

After this ships, a re-run of `pickle install`/`upgrade` reports honestly whether each file was created, refreshed, or left untouched; the help text stops claiming agent autodetection that doesn't exist; a typo'd `pickle uninstall foo` refuses instead of silently uninstalling; and the marker-spacing/file-mode/test-coverage gaps the T-004/T-006/T-018 reviews found are closed.

## Description

**Re-graded at 2026 refinement** from `impact: low-medium / complexity: low / cost: S` to
`impact: medium / complexity: medium / cost: M`: every item below was re-verified against the
current tree (line numbers refreshed) and all remain live defects — none were accidentally fixed
by the seven other tickets that have patched this Description since filing. The set has grown
into six independently-fixable but thematically identical groups (all on the
`internal/cli`/`internal/install` setup-command surface), and one of them (item 9's
`runUninstall` stray-positional gap) is a real, still-open safety hole — a typo'd
`pickle uninstall foo` performs a full uninstall today — which is why impact moved off `low`.
Kept as one ticket rather than split, consistent with every prior review's choice to fold new
findings in here rather than spawn a sibling.

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

3. **cli `runInstall`/`runUninstall` usage-error coverage** — *(re-verified at 2026 refinement:
   narrower than filed)* other tests now exercise `runInstall`'s happy path incidentally via
   `Run(payload, ..., []string{"install", ...})` (`TestInstallHooksFlag`,
   `TestUpgradeWarnsOnLayoutOnlyBoardDrift`), so "only exercised by manual acceptance" no longer
   holds for the happy path. What is still genuinely untested: `runInstall`'s own four
   usage-error branches (`internal/cli/install.go` — `--in-tree`+`--path` conflict, bare
   `--path "."` without `--in-tree`, `--path ""` with child flags given, and a plain bad flag)
   have zero coverage in `TestRunExitCodes` (`internal/cli/cli_test.go:95-131`), which covers
   every other command's usage errors but none of `install`'s.

4. ~~**`--agent` handling**~~ *(obsoleted 2026-07-26 by T-009: `--agent` is now fully parsed
   and wired — unknown values are a usage error, `--no-claude` warns deprecated. Nothing left
   here.)* **Replaced by a folded T-009 review finding (F4):** the top-level `pickle help`
   text still says install lays the skill down "for detected agents"
   (`internal/cli/cli.go` usage string) — but there is **no autodetection** (a T-009 locked
   decision): the set is exactly what `--agent` names, default `claude`. Reword the help line
   (e.g. "for the agents named by --agent, default claude").
5. *(moved to T-014, which was dropped and absorbed into T-039, which was itself dropped and
   superseded by T-044, done)* board-cell escaping — the child-name substitution into
   `BOARD.md` shared the unescaped-`|`/newline gap; `T-044` settled the board's escape-vs-replace
   question by construction (BOARD.md is now a pure generated artifact, manual parse-back render
   paths removed). Nothing left here.

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
   `+ .agents/skills/brine/` because it routes through `copyPayload`, even when the payload
   is byte-identical — so a no-op upgrade looks like it changed the tree. Whatever
   created-vs-refreshed distinction item 2 lands must cover the `Upgrade` path too.
   **Mechanism constraint this imposes on item 2:** `Upgrade` does `os.RemoveAll` on the skill dir
   (`internal/install/install.go:420`, re-verified 2026 refinement) *before* calling `copyPayload`
   (`:424`), so a presence-based "did it already exist?" check always reports "created" on the
   upgrade path. The created-vs-refreshed signal must therefore be a **content** comparison
   captured before the removal, not an existence check. (`scaffoldTickets` is unaffected —
   `Upgrade` never calls it.) *(Re-verified 2026 refinement: `Upgrade`'s pi-scaffold refresh
   (`:452-472`) and hooks refresh already gained a content-diff created/refreshed/current
   distinction since this item was filed — the remaining gap is narrower than originally scoped:
   just the skill-payload copy itself, `scaffoldTickets`, and the two labels below.)*
   **Two more labels, added 2026-07-25 by the T-018 review (non-blocking finding 11):**
   the in-place single-line edits of `pickle.toml` (now `internal/install/install.go:489-496`
   for the layout back-fill and `:507-516` for the `payload_version` stamp, re-verified 2026
   refinement) report `+` (created) — the same created-vs-refreshed wording used for a brand-new
   file, on what is actually a one-line edit of an existing file (each already carries a
   descriptive suffix — `"pickle.toml (layout -> in-tree)"` etc. — so this is a labelling nit,
   not the flat always-`+` dishonesty `copyPayload`/`scaffoldTickets` have; fold it into whatever
   symbol/suffix convention this item settles on, don't let it block the rest); and
   `internal/cli/cli.go:104-105` (re-verified 2026 refinement)'s one-line help for `upgrade`
   never mentions that it stamps `payload_version` in `pickle.toml`, which the user manual does
   mention (`docs/user-manual/cli-reference.adoc`, upgrade section; moved out of `README.md` by
   T-047) and which is the file most users care about.
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

   *(Re-verified 2026 refinement, spot-check only — not re-run through mutation testing: all
   three call sites are unchanged — `verifyStampedVersion`'s call site is still `install.go:513`,
   `writeConfig` still hardcodes `true`/`true` at `:830-833` instead of
   `config.DefaultOverarchingAuto`/`config.DefaultChildPublishGated`, and the skill-copy +
   marker-rewrite still precede the payload_version stamp in `Upgrade`'s body. Treated as still
   live.)*

9. **Extend item 3's cli-coverage to `runUpgrade`/`runUninstall`.** *(Re-verified 2026
   refinement: "zero CLI-level tests" is now stale for `runUpgrade`** —
   `TestUpgradeWarnsOnLayoutOnlyBoardDrift` (`cli_test.go:475`) already drives it through `Run`
   in a temp root, and `TestRunExitCodes` already covers its argv rejections. **`runUninstall`
   is still exactly as bare as filed**: only its bad-flag rejection is tested
   (`cli_test.go:110`) — no happy path, no `--dry-run`, no stray-arg case, at any level.)* Add
   table-driven cases for `runUninstall` against a temp root: bad-flag → `exitUsage` (already
   covered) plus the still-missing `--dry-run` → tree unchanged and a happy-path removal.
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
     from `internal/cli/install.go:191` (`runUpgrade`'s own `fs.NArg() > 0` guard, re-verified
     2026 refinement — the same style, just in the sibling handler) and cover it with an
     `uninstall stray arg` row.

10. **`injectMarker`/`stripMarker` hard-code `0o644` on `AGENTS.md`/`CLAUDE.md`** (added
    2026-07-25 by T-018, gate finding N10; re-anchored 2026 refinement — still four writers, at
    `internal/install/install.go:933,950,964` in `injectMarker` and `:1010` in `stripMarker`)
    call `os.WriteFile(..., 0o644)`, so every install/upgrade/uninstall silently resets a
    non-default permission on files the user owns. Same family as T-018 (silent loss of user
    state) but on this ticket's `injectMarker` surface, so T-018 deliberately left it alone.
    `internal/atomicfile.WriteFile` is now an exported atomic, mode-preserving write primitive
    (added by T-018 as `config.writePreservingMode`, extracted to its own package by T-101)
    worth mirroring — see T-012 item 7, which proposes the same fix for `config.Save`.

All items are input-hardening / polish on the install surface, hence non-blocking.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-013-install-polish
```

All work happens on this branch, inside `pickle`'s own repo (`project: pickle`, root-path
child). Commit locally as you go; per the root-path Finish convention, tidy WIP commits into a
small number of atomic commits before presenting (see Finish, below). Never run
`pickle install|upgrade|uninstall` against this repo itself from this branch — per
`AGENTS.md`'s self-modify policy, all manual testing goes through a throwaway dir with the
built binary copied in as `pickle-test`.

### Prerequisite gate (hard)

`depends-on: [T-004]` — `T-004` is in `6-done/`. No other prerequisites; the tree is otherwise
clean (`pickle board audit` clean before starting).

### Confirmed design decisions (do not deviate without asking)

1. **Created/refreshed/current is a content comparison, not an existence check**, mirroring the
   pattern `Upgrade` already uses for the pi scaffolds and hooks refresh
   (`internal/install/install.go:452-472`): read what is on disk (if anything) before writing,
   compare bytes, and label accordingly. It applies at **whole-tree granularity** for the skill
   payload and the tickets scaffold (one summary line each), not per-file — matching the
   existing convention, not inventing a noisier one.
2. **The `pickle.toml` in-place-edit lines (layout back-fill, `payload_version` stamp) keep their
   `+`/`res.created(...)` treatment and are out of scope for a symbol change** — each already
   carries a descriptive suffix (`"pickle.toml (layout -> in-tree)"` etc.) that states exactly
   what happened, unlike `copyPayload`/`scaffoldTickets`' flat always-`+`. Task 3 below is scoped
   to those two functions plus the `upgrade` help text.
3. **`atomicfile.WriteFile(path, data)` replaces every `os.WriteFile(path, data, 0o644)` inside
   `injectMarker`/`stripMarker` verbatim** — no branching changes at the call sites, since
   `atomicfile.WriteFile` already handles both the create-new-file and
   replace-existing-file cases internally (see its doc comment in
   `internal/atomicfile/atomicfile.go`).
4. **`projectRoot() (string, int)` is added to `internal/cli/project.go`**, next to `loadConfig()`
   (`:43`), resolving `os.Getwd()` → `config.Find(wd)` → `filepath.Dir(cfgPath)` and returning
   `(root, exitOK)` or `("", errf(...))` on failure. `runDoctor` and `runUninstall` replace their
   preambles with a single call; `runUpgrade` also switches to it, then derives
   `cfgPath := filepath.Join(root, config.FileName)` for its own subsequent loads.
5. **`install.Upgrade`'s `Result` gains a `PrevVersion string` field**, set immediately after its
   own initial `config.Load` (before any mutation). `runUpgrade` uses `res.PrevVersion` instead of
   loading `pickle.toml` itself before calling `Upgrade`, and keeps its post-upgrade `config.Load`
   (that one reads mutated state and cannot be dropped).
6. **The "upgrade exits 1 from a partially-applied upgrade" finding (item 8, R9) is resolved by
   documentation, not reordering.** `Upgrade` already refreshes the skill payload, markers, pi
   scaffolds and hooks idempotently before it ever touches `payload_version`; if the stamp step
   then refuses, re-running `pickle upgrade` picks the stamp back up (the other steps just no-op
   on their own idempotency) rather than redoing work. State that explicitly in
   `docs/user-manual/cli-reference.adoc`'s upgrade section instead of serializing the stamp
   first, which would trade one partial-state shape for a riskier one (reporting success before
   the refresh that success is supposed to describe has actually landed).

### Tasks

#### Task 1 — fix `injectMarker`'s append-spacing bug + trailing-newline test (item 1)
In `internal/install/install.go`'s no-marker-found append branch (currently `:967-976`, re-anchored
at pickup — drifted from `:957-966` after T-042), replace
the whole `sep`-computation `if`/`else if` and the `out :=` line with a single trim-then-join:
```go
out := strings.TrimRight(text, "\n") + "\n\n" + wrapped + "\n"
```
This drops the dead `else if` branch entirely and guarantees exactly one blank line before the
block regardless of whether `text` ended with zero, one, or several newlines (an empty `text`
after trimming is fine too — same edge case the current code already tolerates). Add
`TestInjectMarkerAppendSpacing` in `internal/install/install_test.go`: a table over `text` ending
in `""` (no trailing newline), `"\n"`, and `"\n\n"`, asserting the byte-exact output has exactly
one blank line before `MarkerBegin`.

#### Task 2 — reword the "detected agents" help text (item 4 / folded T-009 F4)
In `internal/cli/cli.go:96`, change `"Scaffold tickets/, install the skill for detected agents,"`
to name `--agent` explicitly, e.g. `"...install the skill for the agents named by --agent
(default claude),"`. No autodetection exists (T-009 decision) — the wording must not imply it.

#### Task 3 — created/refreshed/current labels for the skill payload + tickets scaffold, and
mention the version stamp in `upgrade`'s help (items 2 + 8 + N11)
In `internal/install/install.go`:
- `copyPayload` (`:716-748`, re-anchored at pickup — drifted from `:709-734` after T-042): before
  the `fs.WalkDir`, check whether `dst` already existed
  (`os.Lstat`). During the walk, compare each file's new bytes against what is currently on disk
  (if any) and track whether *anything* changed. After the walk, per decision 1: `dst` did not
  exist → `res.created(SkillDir + "/")` (unchanged behaviour, fresh install); `dst` existed and
  something changed → `res.created(SkillDir + "/ (refreshed)")`; `dst` existed and nothing
  changed → `res.skipped(SkillDir + "/ (current)")`.
- `scaffoldTickets` (`:754-771`, re-anchored at pickup — drifted from `:744-757` after T-042):
  track whether any status dir's `.gitkeep` was actually created
  this run (the existing per-dir `continue` already skips ones that exist); if none were, call
  `res.skipped(...)` with an "(already scaffolded)" style message instead of `res.created(...)`.
- `internal/cli/cli.go:104-105`: extend the one-line `upgrade` help to mention it stamps
  `payload_version` in `pickle.toml`, e.g. "...to this binary's version (stamps `payload_version`
  in `pickle.toml`; never touches `tickets/`)."

#### Task 4 — `writeConfig` uses the exported commit-policy defaults (item 8, R8)
In `internal/install/install.go`'s `writeConfig` (`:830-868`, re-anchored at pickup — drifted from
`:817-851` after T-042), replace the literal
`OverarchingAuto: true, ChildPublishGated: true` with
`config.DefaultOverarchingAuto, config.DefaultChildPublishGated` (`internal/config/config.go:57-58`),
so the two can no longer silently diverge from `applyDefaults`'s own defaults.

#### Task 5 — bind `verifyStampedVersion` to a real test (item 8, R9 finding 1)
Give `install.Upgrade` (and `Run`, for symmetry) an injectable stamp step: add a
`stampPayloadVersion func(path, want string) error` field, defaulted to the real
`config.SetPayloadVersionInPlace` + `verifyStampedVersion` pair, that production code never sets
explicitly (so it behaves exactly as today). Add `TestUpgradeReportsStampVerificationFailure` in
`internal/install/install_test.go` that substitutes a lying stamp func (reports success but
leaves the file unstamped) and asserts `Upgrade` returns an error — the mutation-confirmed gap
from the T-018 rework record.

#### Task 6 — document the partial-upgrade retry contract (item 8, R9 finding 3)
In `docs/user-manual/cli-reference.adoc`'s `pickle upgrade` section (after the existing
"Idempotent: ..." paragraph, `:315-318`, re-anchored at pickup — drifted from `:311-314`), add a sentence stating that if the `payload_version`
stamp step itself refuses, everything else (payload, markers, pi scaffolds, hooks) has already
been refreshed and is safe to leave as-is; re-running `pickle upgrade` retries only the stamp.

#### Task 7 — CLI usage-error coverage for `install`'s own validation branches (item 3)
In `internal/cli/cli_test.go`'s `TestRunExitCodes` table (`:84-127`, re-anchored at pickup —
drifted from `:95-131`), add cases exercising
`runInstall`'s four bespoke branches in `internal/cli/install.go:38-51`: `install --in-tree
--path foo` (conflict), `install --path .` (must pass `--in-tree` instead), `install --project
foo` with no `--path`/`--in-tree` (child flags without a path), and a plain `install --bogus` —
all `exitUsage`.

#### Task 8 — `runUninstall` stray-positional guard + full temp-root coverage (item 9)
- In `internal/cli/install.go`'s `runUninstall` (`:292-298`), add the same
  `if fs.NArg() > 0 { ...; return exitUsage }` guard `runUpgrade` already has at `:191-194`.
- In `internal/cli/cli_test.go`, add `{"uninstall stray arg", []string{"uninstall", "extra"},
  exitUsage}` to `TestRunExitCodes`.
- Add a new test (temp root via `newProject(t)`, per its own doc comment on why this is mandatory
  and not merely tidy) asserting `pickle uninstall --dry-run` changes nothing on disk (snapshot
  file list + a content hash of `AGENTS.md` before/after) and reports the items it *would* remove.
- Add a happy-path test: `pickle uninstall` on a freshly installed temp project exits `exitOK` and
  actually removes `{skill-dir}/` and the marker block.

#### Task 9 — extract `projectRoot()` and route the setup commands through it (item 6)
Add `projectRoot() (string, int)` to `internal/cli/project.go` (decision 4, above). Replace the
triplicated `os.Getwd()`/`config.Find`/`filepath.Dir` preamble in `runUpgrade` (`:196-204`),
`runDoctor` (`:263-271`), and `runUninstall` (`:300-308`) with a call to it.

#### Task 10 — `install.Upgrade` reports `PrevVersion`; `runUpgrade` drops the redundant load (item 7)
Add `PrevVersion string` to `install.Result` (`:166-174`, re-anchored at pickup — drifted from
`:166-173`); set it in `Upgrade` (`:393-...`) right
after its own `config.Load`. In `runUpgrade` (`internal/cli/install.go:184-253`, re-anchored at
pickup — drifted from `:184-249`), delete the
`before, err := config.Load(cfgPath)` / `prevVersion := before.PayloadVersion` pair and use
`res.PrevVersion` after the call to `install.Upgrade` instead.

#### Task 11 — mode-preserving writes in `injectMarker`/`stripMarker` (item 10)
In `internal/install/install.go`, replace the four `os.WriteFile(path, []byte(...), 0o644)` calls
at `injectMarker`'s `:943`, `:960`, `:974` and `stripMarker`'s `:1020` (re-anchored at pickup —
drifted from `:933`, `:950`, `:964` and `:1010` after T-042) with
`atomicfile.WriteFile(path, []byte(...))`, adding the `"github.com/codcod/pickle/internal/atomicfile"`
import.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

All must be clean. In particular:
- `go test ./internal/install/... -run 'TestInjectMarkerAppendSpacing|TestUpgradeReportsStampVerificationFailure'`
  exercise Tasks 1 and 5 directly.
- `go test ./internal/cli/... -run TestRunExitCodes` covers Tasks 7 and 8's new usage-error rows.
- The new `runUninstall` dry-run/happy-path tests (Task 8) pass.

Manual smoke (per `AGENTS.md`'s self-modify policy — never against this repo):
```
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"
./pickle-test install --in-tree --project demo
./pickle-test install --in-tree --project demo   # re-run: expect "= .agents/skills/brine/ (current)"
./pickle-test uninstall foo                       # expect a usage error, nothing removed
./pickle-test uninstall --dry-run                 # expect a list, tree unchanged
```

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`: Task 6's one-sentence addition to the `pickle upgrade`
  section (retry-safe partial-upgrade contract).
- No other adoc changes: `install`'s reworded help text (Task 2) and `upgrade`'s extended help
  line (Task 3) are the in-binary `pickle help` surface, which the manual does not quote
  verbatim anywhere (verified at refinement) — the manual already independently states the
  no-autodetection fact and the `payload_version`-stamping fact. The `runUninstall` stray-arg
  rejection (Task 8) needs no doc mention either, matching the existing convention that
  `upgrade`'s own stray-arg rejection isn't separately documented.

### Finish (mandatory)

1. Acceptance test green (`just build/test/lint/docs-check`).
2. Docs updated per above.
3. Write a summary: files touched (`internal/install/install.go`, `internal/cli/cli.go`,
   `internal/cli/install.go`, `internal/cli/project.go`, `internal/cli/cli_test.go`,
   `internal/install/install_test.go`, `docs/user-manual/cli-reference.adoc`), and confirm no
   task was silently dropped (there were 11).
4. Suggested commit message (no single scope — spans install/upgrade/uninstall CLI + engine):
   ```
   fix: honest install/upgrade summaries, uninstall guard, marker fixes (T-013)
   ```
5. Tidy the branch's WIP commits into a small number of atomic, correctly typed/scoped commits
   (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-013 in-review --reason "acceptance green"`.

## Review

**Verdict: REWORK** — three blocking findings (one behavioural regression, one test-gap on the
headline feature, one missing CHANGELOG entry). The other nine findings are dispositioned below.

- [x] Reviewer independence settled (step 0): **delegated**. The reviewing agent authored the
      branch in this session, so both audits were run by independent sub-agents spawned fresh and
      briefed adversarially (impl+quality; consistency+docs). Every delegated finding was
      re-verified by hand before entering the table — two were corrected in the process (see N4,
      and B1's severity was confirmed by differential test against `main`).
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b): run on `docs/user-manual/cli-reference.adoc`. Every
      suggestion returned targeted pre-existing prose outside this ticket's diff; none applied.
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket in `1-to-do/`, `2-ready/` or
      `5-rework/` lists T-013 in `depends-on:` or references it in prose. Nothing to patch.
- [ ] Summary + commit message & MR attributes presented for approval (step 9) — **deferred:**
      the branch goes to rework first, so there is nothing to publish yet.

### Acceptance test (re-run verbatim)

`just build`, `just test` (21 packages ok), `just lint`, `just docs-check` — all clean, on the
branch as submitted. The named targeted runs
(`TestInjectMarkerAppendSpacing|TestUpgradeReportsStampVerificationFailure`, `TestRunExitCodes`,
the new uninstall tests) all pass. The acceptance test as written is **green** — it simply does
not exercise the regression B1 describes, which is itself finding B2.

Additionally verified (not required by the plan): the branch trial-merges cleanly onto the
**new** `origin/main` (T-115 landed mid-implementation, hardening the docs xref checker) and the
full suite plus `just docs-check` pass on that merge. No integration problem.

All 11 tasks and all 6 confirmed design decisions were verified individually as implemented;
mutation testing confirmed the new tests are genuinely binding (deleting the
`verifyStampedVersion` call site, restoring the old `sep` computation, removing the `fs.NArg()`
guard, and neutering the `--path "."` branch each fail a test). No tautological tests.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| B1 | blocking | correctness | — | `Upgrade` no longer replaces the skill dir wholesale: when `skillPayloadDiffers` finds no byte change it skips `RemoveAll` + re-copy entirely, so a tampered tree is left unrepaired **and reported `(current)`** | Differential test vs `main`'s binary, identical scenario. Stale empty dir `brine/leftoverdir/nested`: **survives** here, removed by `main`. Payload file left at `0600`: **stays `0600`** here, restored to `0644` by `main`. Payload file replaced by a **symlink to a file outside the tree: survives** here, replaced by `main`. Also falsifies `install.go`'s `Upgrade` doc ("replaced wholesale") and `cli-reference.adoc` ("re-copied in full", "removing files that the new payload dropped") | Drop the `if existed && !changed { … break }` early-out; always `RemoveAll` + `writeSkillPayload`, keeping the pre-captured `(existed, changed)` **purely for the label**. Restores `main`'s behaviour exactly while keeping the honest labels the ticket wanted |
| B2 | blocking | test-gap | — | The branch's headline behaviour — the created/refreshed/current labels — has **no test binding it**: reverting all three labels to the old always-`+` fails no test. This is precisely the "100% covered, 0% bound" defect class the ticket's own R9 finding exists to close, reintroduced by the change that closes it | `rg '\(current\)\|\(refreshed\)\|already scaffolded' internal/ -g '*_test.go'` matches only pre-existing hook tests; `install_test.go` asserts only `already at v2` | Add tests asserting all three labels for both `copyPayload` and `Upgrade`, **including a regression test that a tampered skill dir is repaired** — that test is what stops B1 recurring |
| B3 | blocking | docs-gap | — | No `CHANGELOG.md` entry, though the branch ships three user-facing changes (summary labels, `uninstall` stray-arg rejection, two `pickle help` lines) | The repo's own tool: `./pickle changelog check --since main --until HEAD` → `1 candidate(s) shipped but not named in "Unreleased": T-013`. Convention is established — `## [Unreleased]` currently carries `(T-113)`, `(T-117)`, `(T-038)` entries | Add an `## [Unreleased]` entry tagged `(T-013)`. Note the ticket's plan never mentions CHANGELOG — a template/plan gap, not an implementer excuse |
| N1 | non-blocking | docs-gap | fixed inline | `type UpgradeOptions` was inserted between `Upgrade`'s doc comment and `func Upgrade`, so `go doc install Upgrade` printed **no documentation** and `UpgradeOptions` carried Upgrade's paragraph | Verified pre-fix with `go doc`; `go vet` cannot catch it | Fixed in `bab0e13`: type moved above the comment block; `go doc` verified restored |
| N2 | non-blocking | design | fixed inline | `skillPayloadDiffers` kept walking the entire payload after a difference was established, with a comment claiming it did so to finish populating `seen` — but the caller returns on `changed` before `seen` is ever consulted | `install.go`, first walk vs the `if changed { return true, true, nil }` that follows it | Fixed in `bab0e13`: returns `fs.SkipAll`, comment corrected |
| N3 | non-blocking | correctness | noted | `runUpgrade`'s trailer still prints `(refreshed payload + markers)` on a run whose every summary line is `=` — item 8's own complaint surviving one line lower | Smoke in a throwaway dir: five `=` lines, then `already at … (refreshed payload + markers)` | Self-resolving: once B1's fix restores the unconditional re-copy, the payload genuinely *is* refreshed and the trailer is true again. **Re-check at the scoped re-review** rather than editing it now |
| N4 | non-blocking | design | noted | `copyPayload` writes every payload file unconditionally and *then* labels the result `(current)`, while every other `=` in the CLI means "not written". Mtimes change on a run reported as current | `install.go` `copyPayload`: `writeSkillPayload` precedes the label `switch` | **Correction to a delegated finding:** the audit also claimed this resets files to `0o644`. That is **false** — `os.WriteFile` preserves an existing file's mode; verified a `0600` payload file survives an install re-run at `0600`. Only the `=`-semantics half of the finding stands |
| N5 | non-blocking | correctness | noted | An `install` re-run can print `(refreshed)` while leaving the tree stale: `copyPayload` detects extra files via `skillPayloadDiffers` but never prunes them, so the label claims an effect the function does not have | Planted a stale file; re-run printed `+ … (refreshed)` and the file remained | Either prune in `copyPayload` too, or exclude the extras check from the label it computes. Worth folding into B1's fix, since both concern what the two paths actually guarantee |
| N6 | non-blocking | design | noted | The created/refreshed/current label triple is now written twice — inline in `Upgrade`'s `default:` branch and in `copyPayload` — with two different mechanisms, so they can drift. The ticket's own item 6 theme is redundancy removal | Two sites in `install.go` | B1's fix should collapse them back to one, since it removes the reason `Upgrade` had its own copy |
| N7 | non-blocking | design | noted | The `os.Getwd` → `config.Find` → `filepath.Dir` preamble item 6 wanted collapsed still has further copies the ticket did not name: `hooks.go`'s `hookRoot()` (byte-for-byte `projectRoot()` modulo its error type), `hooks.go`'s `hookConfig()`, and `project.go`'s `loadConfig()` | `internal/cli/hooks.go`, `internal/cli/project.go` | Genuine but small; `noted` is the default and this does not clear the promotion test on its own. A later reviewer can promote it by citing this row |
| N8 | non-blocking | design | noted | `Upgrade(…, opts ...UpgradeOptions)` silently ignores `opts[1:]`; variadic-for-optional papers over updating the existing test call sites | `install.go` `Upgrade` signature | At minimum reject `len(opts) > 1`; a second exported `UpgradeWith` would be more idiomatic |
| N9 | non-blocking | docs-gap | new ticket | `skill/SKILL.md` still tells every installed project that `pickle install` lays the skill down "for the **detected agents**" — the identical false claim Task 2 removed from `pickle help`, contradicting T-009's no-autodetection decision and `cli-reference.adoc` ("There is *no autodetection*") | `skill/SKILL.md`, Install & register section | **Pre-existing, so not inline-fixable** per rules §5 ("did this branch break it?" — no; the branch in fact improved net consistency, taking the tree from one correct surface out of three to two). Filed as its own ticket |

**Disposition summary:** 12 findings — 3 blocking (B1 correctness, B2 test-gap, B3 docs-gap; not
dispositioned, they are fixed in rework), 9 non-blocking: 2 `fixed inline` (N1, N2 — both in
`bab0e13`), 6 `noted` (N3, N4, N5, N6, N7, N8), 1 `new ticket` (N9 → T-119).

```
cost: estimated M, actual M
```

### Rework round 1 — what was fixed (2026-08-24)

Scope was the three blocking findings and nothing else. Branch:
`feat/T-013-install-polish`, commits `9ec99de` (B1+B2) and `9bb335a` (B3).

**B1 — fixed.** The `if existed && !changed { … break }` early-out is gone from `Upgrade`'s
`default:` branch: `os.RemoveAll` + `writeSkillPayload` now run unconditionally, exactly as they
did before this ticket, and the pre-captured `(existed, changed)` pair is used **only** to choose
the label. Re-ran the differential scenario that exposed it — stale empty dir, `0600` payload
file, payload file swapped for an outside symlink — and all three are repaired again, matching
`main`'s binary. The `(current)` label now means "the bytes on disk already matched", which the
new `Result.labelSkillDir` doc comment states explicitly.

**B2 — fixed.** Four new tests in `internal/install/install_test.go`:
`TestSkillDirLabelsOnInstall`, `TestSkillDirLabelsOnUpgrade`,
`TestScaffoldTicketsLabelsSecondRun`, and `TestUpgradeAlwaysReplacesSkillDirWholesale` (the B1
guard, asserting each of the three repairs separately). All four were **mutation-verified to
bind**, which is the whole point of the finding:

| mutation | result |
|---|---|
| reintroduce B1's early-out | `TestUpgradeAlwaysReplacesSkillDirWholesale` fails on all three assertions |
| `labelSkillDir` → unconditional `created(SkillDir+"/")` | both label tests fail |
| `scaffoldTickets` → always `created` | `TestScaffoldTicketsLabelsSecondRun` fails |

**B3 — fixed.** Four entries added under `CHANGELOG.md`'s `## [Unreleased]` → `### Fixed`, tagged
`(T-013)`, covering the summary labels, the `uninstall` stray-arg rejection, the two `pickle help`
lines, and the marker-file mode preservation. `./pickle changelog check --since main --until HEAD`
now reports `no candidates — every shipped ticket is mentioned`.

**Acceptance test re-run:** `just build`, `just test` (20 packages ok), `just lint`,
`just docs-check` — all clean.

**Non-blocking findings, re-checked but deliberately not worked** (rework scope is blocking only):

- **N3 self-resolved, as the review predicted.** With the unconditional re-copy restored, the
  payload genuinely *is* refreshed on every upgrade, so `runUpgrade`'s
  `(refreshed payload + markers)` trailer is true again. Verified by smoke test.
- **N6 resolved as a side effect of B1's fix.** Both label sites now route through the shared
  `Result.labelSkillDir`, so the duplicated triple the finding warned would drift is gone.
- N4, N5, N7, N8 remain `noted`, untouched, with their evidence in the table above. N9 remains
  T-119.

## History

- 2026-07-23 — created (TO DO). source: T-004 review (non-blocking findings); via pickle ticket new
- 2026-07-25 — broadened by the T-018 gate + review: added injectMarker file-mode preservation (item 10) and two more summary/help labels (item 8)
- 2026-07-25 — broadened again by the T-018 re-review (R8/R9): verifyStampedVersion test seam, Config{} default leak + writeConfig constants, partially-applied upgrade on refusal (item 8)
- 2026-07-25 — re-anchored by the T-018 S1 re-review: install.go refs (:126 -> :129, :130 -> :133, :162 -> :168, :180-191 -> :173-184, :163-168 -> :165-167)
- 2026-07-26 — item 4 rewritten by the T-009 review (impact sweep): the "--agent is a no-op" premise is obsolete; replaced with the folded F4 finding (`pickle help` says "detected agents", contradicting T-009's no-autodetection decision). Line refs in items 6–10 may have shifted (~200 lines added to internal/install/install.go by T-009); re-anchor at refinement
- 2026-07-26 — patched by the T-047 review (impact sweep): README passage it cited moved to docs/user-manual/cli-reference.adoc
- 2026-08-13 — patched by **T-074's review impact sweep**, which touches item 8 twice. (1) The
  string item 8 quotes as `upgrade`'s misleading output is now `+ .agents/skills/brine/`, not
  `+ .agents/skills/ticket-flow/` — the premise (a byte-identical payload still reported as
  created) is unchanged, only the path. Likewise the `uninstall`-damage example at `:119`. (2)
  T-074 gave `runUpgrade` a **new** output class: it prints `res.Removed` as `- <path>` lines
  ahead of the created/skipped ones, for the legacy sweep. So the created-vs-refreshed
  distinction item 8 asks for now has a third label to stay consistent with, and the removed
  lines are a worked example of `upgrade` reporting an action it genuinely took. Item 8's line
  anchors have also moved: `Upgrade`'s `os.RemoveAll`/`copyPayload` pair is no longer at
  `internal/install/install.go:129`/`:133` — the function grew a legacy-sweep prologue and a
  three-way switch — so re-anchor by searching the text at refinement, as the 2026-07-26 note
  above already advises for items 6–10
- 2026-08-15 — patched by T-101's review impact sweep: item's reference to
  `config.writePreservingMode` re-pointed at `internal/atomicfile.WriteFile`, which T-101
  extracted it into. The helper is now exported, so `injectMarker` can call it directly instead
  of mirroring it.
- 2026-08-20 — refined: all ten items re-verified against current line numbers (none had been
  accidentally fixed); items 3/9's premise narrowed (some CLI-level coverage now exists
  incidentally via other tests) and re-scoped to what is still genuinely untested. Re-graded
  impact low-medium → medium, complexity low → medium, cost S → M — the `runUninstall`
  stray-positional gap (item 9) is a live safety hole, not just polish, and the set has grown to
  eleven concrete tasks. Kept as one ticket (user confirmed) rather than split, consistent with
  every prior review's own choice.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-24 — plan amended inline: pickup applicability gate (independent sub-agent audit) found
  no blocking issues; re-anchored Tasks 1/3/4/6/7/10/11's line citations after drift from T-042
  (no code-shape changes, offsets only), and corrected item 5's dangling "moved to T-014" pointer
  to note T-014 was dropped/absorbed into T-039, itself dropped/superseded by T-044 (done), which
  settled the board escape-vs-replace question by construction
- 2026-08-24 — READY → IN DEVELOPMENT: picked up
- 2026-08-24 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-24 — IN REVIEW → REWORK: review: 3 blocking (B1 upgrade no longer replaces skill dir wholesale; B2 no test binds the new labels; B3 no CHANGELOG entry); 9 non-blocking dispositioned (2 fixed inline, 6 noted, 1 -> T-119)
- 2026-08-24 — REWORK → IN REVIEW: rework: B1 wholesale replacement restored, B2 label+repair tests added (mutation-verified), B3 CHANGELOG entries added
