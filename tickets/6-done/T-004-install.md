---
id: T-004
title: install (scaffold + skill install + marker injection + first child)
project: pickle
depends-on: [T-001]
impact: high
complexity: high
cost: L
---

# T-004 — install (scaffold + skill install + marker injection + first child)

## Description

Implement `pickle install`, run once in an overarching project:

- create `tickets/` with the seven ordered status dirs;
- copy the embedded board skeleton (`skill/resources/BOARD.md`) → `tickets/BOARD.md` (set the
  date);
- write the short `tickets/README.md` pointer;
- install the embedded `skill/` tree → `.agents/skills/ticket-flow/`, and symlink
  `.claude/skills/ticket-flow` → it for Claude Code;
- inject an **idempotent** `<!-- pickle:begin -->` / `<!-- pickle:end -->` marker block into
  `AGENTS.md` (and `CLAUDE.md`, or symlink `CLAUDE.md → AGENTS.md` — a flag) stating "start at
  `tickets/BOARD.md`" + the project configuration;
- write `pickle.toml` and register the first child-project (prompt, or `--project <name> <path>`).

Per-project (never writes to `~/`), idempotent, safe to re-run. Detect/select agents
(`--agent claude,pi,opencode` or auto-detect from existing `.claude/`, `.pi/`, `AGENTS.md`).
This bootstrap repo is the reference for what a correct install produces. Needs T-001. Phase P2.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # child-project 'pickle' is the repo root
git checkout main
git checkout -b feat/T-004-install
```
Local WIP commits fine; **no push / no MR without user approval**.

### Prerequisite gate (hard)

`depends-on: [T-001]` — T-001 done + merged (cdad65e). Clean tree on `main`. Reuses the merged
`internal/config` (Render/Save/AddProject) and `internal/audit` (post-install self-check).

### Confirmed design decisions (do not deviate without asking)

1. **The bootstrap repo is the *structural* reference, not literal.** `pickle install` must
   produce the layout this repo has — `.agents/skills/ticket-flow/`, `.claude/skills/ticket-flow`,
   `AGENTS.md`+`CLAUDE.md` markers, `tickets/` (7 dirs + `BOARD.md` + `README.md`), `pickle.toml`
   with the first child — **except** the skill dir. This repo self-hosts via
   `.agents/skills/ticket-flow -> ../../skill` (a symlink to the payload *source*); a consumer
   project has no `skill/`, so **install COPIES the embedded payload into
   `.agents/skills/ticket-flow/` as real files** (from `fs.Sub(cli.Payload, "skill")`).
2. **Claude view is a symlink:** `.claude/skills/ticket-flow -> ../../.agents/skills/ticket-flow`
   (relative). `.agents/` is the agent-agnostic standard; `.claude/` is the Claude Code view.
3. **Scope = the core reference install only.** `.agents` skill + `AGENTS.md`/`CLAUDE.md` markers
   + `.claude` symlink + `tickets/` + `pickle.toml`+first child. The **pi/opencode** agent
   matrix (`.pi/`, opencode wiring) is **out of scope — deferred to T-010 / T-009** (soft ref).
   A `--agent` flag is accepted but in T-004 only toggles the Claude view (see flags).
4. **First child defaults to the repo root.** `--project <name>` (default: basename of the
   install root) and `--path <path>` (default `"."`). Registered via `config.AddProject`, so the
   child inherits the branch-prefix/WIP defaults. Commands (`build`/`test`/`lint`/`docs`) are
   left empty unless passed (`--build/--test/--lint/--docs`); a consumer edits `pickle.toml`
   after. (Flag form chosen over the Description's inline `--project <name> <path>` for parsing
   clarity + testability; note in the summary.)
5. **Idempotent + safe to re-run, per-project, never writes to `~/` or outside the root.**
   - **Payload + markers are refresh-idempotent:** re-copy the skill, re-inject the marker block
     *between the existing `<!-- pickle:begin -->`/`<!-- pickle:end -->` pair* (append the block
     only if absent) — never duplicate.
   - **Instance data is preserved:** an existing non-empty `tickets/BOARD.md`, any ticket files,
     and an existing `pickle.toml` are **never overwritten** (created only if absent). So
     re-running install on an installed project refreshes the skill+markers and leaves the board
     and config intact. (Full payload-version bump is `upgrade`'s job — T-006.)
   - **Self-host guard:** if `.agents/skills/ticket-flow` already exists **as a symlink**, leave
     it untouched (this is a dev/self-host link like this repo's); only manage it when it is
     absent or a real directory.
6. **CLAUDE.md:** default = inject the **same** marker block into both `AGENTS.md` and
   `CLAUDE.md` (matches the two-file convention). `--claude-symlink` instead makes
   `CLAUDE.md` a symlink to `AGENTS.md`. `--no-claude` skips all Claude artifacts (no
   `.claude/` symlink, no `CLAUDE.md`).
7. **Status dirs get a `.gitkeep`** (all seven: `1-to-do`…`7-dropped`) so empty dirs survive
   git — this is the deferred item from T-002 (git doesn't track empty dirs; `LoadAll`/audit
   ignore non-`T-NNN` files).
8. **Post-install self-check:** after writing everything, run `audit.Audit(root, cfg)` and fail
   the command if it reports any error (a correct install must be board-audit-clean).

### Tasks

#### Task 1 — `internal/install` package
New package `internal/install` with a testable core:
```go
type Options struct {
    ProjectName, ProjectPath        string
    Build, Test, Lint, Docs         string
    Claude, ClaudeSymlink           bool // Claude view on; CLAUDE.md as symlink
}
type Result struct { Created, Skipped []string } // for the summary
func Run(payload fs.FS, root, payloadVersion string, opts Options) (Result, error)
```
Helpers (all confined to `root`, all idempotent per decision 5):
- `copyPayload` — walk `fs.Sub(payload,"skill")` → write real files under
  `.agents/skills/ticket-flow/` (respect the self-host symlink guard, decision 5).
- `scaffoldTickets` — create the 7 status dirs each with `.gitkeep`.
- `writeBoard` — if `tickets/BOARD.md` absent, copy embedded `resources/BOARD.md`, substitute
  `<YYYY-MM-DD>` → today and every `<child-project>` → the child name.
- `writeTicketsReadme` — if absent, write a short generic `tickets/README.md` pointer to the
  skill's rules/template/review-protocol paths + the build-target note (generated text).
- `ensureSymlink(link, target)` — create/repair a relative symlink; error if a real
  file/dir blocks it.
- `injectMarker(path, block)` — replace text between `<!-- pickle:begin -->` and
  `<!-- pickle:end -->` if present, else append the fenced block (create the file with a
  minimal `# <title>` header if absent).
- `markerBlock(cfg)` — build the block text from the resolved config (start-here → BOARD.md,
  skill location + triggers, project config: child/commands/branch+commit policy/WIP).
- `writeConfig` — if `pickle.toml` absent, build a `config.Config` (payload_version =
  `payloadVersion`, commit policy overarching_auto=true / child_publish_gated=true, first child
  via `AddProject`) and `Save`.

#### Task 2 — wire `internal/cli/install.go`
Replace the stub `runInstall`: parse flags (`--project`, `--path`, `--build/--test/--lint/--docs`,
`--no-claude`, `--claude-symlink`, `--agent` accepted/ignored-beyond-claude for now), resolve
`root` = cwd, apply defaults (decision 4), call `install.Run(cli.Payload, root, cli.Version, opts)`,
print the `Result` summary + a "next: pickle ticket new …" hint. Reload the config and run
`audit.Audit` as the self-check (decision 8).

#### Task 3 — README
Document `pickle install`, its flags, the produced layout, idempotency, and the pi/opencode
deferral (T-009/T-010).

### Acceptance test

```
cd /Users/codcod/Projects/private/pickle
just lint && just test && just build
BIN=$PWD/pickle

# fresh install into a throwaway project
rm -rf /tmp/pk-install && mkdir -p /tmp/pk-install
( cd /tmp/pk-install && git init -q && \
  "$BIN" install --project demo --path . && \
  # layout
  test -f .agents/skills/ticket-flow/SKILL.md && \
  test -f .agents/skills/ticket-flow/resources/TEMPLATE.md && \
  test -L .claude/skills/ticket-flow && \
  test -f tickets/BOARD.md && test -f tickets/README.md && \
  test -f tickets/1-to-do/.gitkeep && test -f tickets/7-dropped/.gitkeep && \
  test -f pickle.toml && \
  grep -q '<!-- pickle:begin -->' AGENTS.md && grep -q '<!-- pickle:begin -->' CLAUDE.md && \
  grep -q 'demo' pickle.toml && grep -q 'demo' tickets/BOARD.md && \
  # the installed flow actually works end-to-end
  "$BIN" board audit && \
  "$BIN" ticket new "First feature" --project demo --impact high --complexity low --cost S && \
  "$BIN" board audit && \
  # idempotent re-run: no duplicate markers, board/ticket preserved
  "$BIN" install --project demo --path . && \
  test "$(grep -c '<!-- pickle:begin -->' AGENTS.md)" -eq 1 && \
  test -f tickets/1-to-do/T-001-first-feature.md && \
  "$BIN" board audit )
echo "[exit=$?]"   # expect 0
```
Expected: full layout created; `board audit` clean; `ticket new` works on the freshly-installed
flow; re-running `install` leaves exactly one marker pair and preserves the T-001 ticket +
board. A unit test in `internal/install` installs into `t.TempDir()` and asserts the tree +
idempotency + that `audit.Audit` returns zero errors.

### Docs update (mandatory)

`README.md` — `install` marked done, flags + produced layout + idempotency + pi/opencode
deferral documented.

### Finish (mandatory)

1. Acceptance test green; `just lint`/`test`/`build` clean.
2. README updated.
3. Summary of files + decisions (incl. the flag-form note, decision 4).
4. Suggested commit: `feat(cli): add install (scaffold + skill + markers + first child) (T-004)`.
5. Commit locally on the branch; **do not push / open MR without approval**.

## Review

**Reviewed:** 2026-07-23 on `feat/T-004-install`. **Verdict: PASS** (no blocking findings;
non-blocking follow-ups → T-013).

Protocol: generic (`skill/resources/review-protocol.md`); no overarching/per-child addendum
configured. Dependency gate: T-001 done + merged (cdad65e) — satisfied.

- [x] Implementation audit — acceptance test re-run verbatim (green)
- [x] Quality audit
- [x] Consistency audit
- [x] Documentation audit (README section added; no docs build configured)
- [x] Findings classified & recorded; non-blocking → T-013
- [x] Ticket moved to 6-done; History appended
- [x] BOARD.md updated; impact sweep done

### Implementation audit (step 2) — all met

| Item | Result | Evidence |
|---|---|---|
| Task 1 `internal/install` (copy payload, scaffold, board, readme, config, markers, symlink) | met | package present; 3 unit tests PASS; coverage 70.2% |
| Task 2 cli `runInstall` (flags + defaults + self-check) | met | `install` produces full layout; post-install `audit.Audit` gate wired |
| Task 3 README | met | `## pickle install` section (layout, flags, idempotency, pi/opencode deferral) |
| Acceptance test (verbatim) | met | fresh install → layout + `board audit` clean + `ticket new` works; re-run keeps one marker pair + preserves T-001 ticket; exit 0 |
| Decisions 1–8 | honoured | copy-not-symlink + self-host guard (`TestSelfHostSymlinkGuard`); relative `.claude` symlink; scope limited (pi/opencode deferred); child defaults; idempotent+preserve (`TestRunIsIdempotent`); CLAUDE.md modes; `.gitkeep` ×7; post-install audit self-check |

### Findings (all non-blocking → T-013)

| # | Severity | Description | Evidence |
|---|---|---|---|
| 1 | non-blocking | `injectMarker` separator logic has a dead `else if` branch; a file ending in `\n\n` gets an extra blank line before the appended block | code read; append repro on a pre-existing marker-less AGENTS.md (spacing correct for the common single-`\n` case) |
| 2 | non-blocking | re-run summary reports `+` (created) for `copyPayload`/`scaffoldTickets` even when only refreshing existing files | re-run output showed `+ .agents/skills/ticket-flow/` / `+ tickets/ (7 status dirs)` on an installed tree |
| 3 | non-blocking | cli `runInstall` wrapper has no direct test (only the `internal/install` core is tested) | coverage: install 70.2%, cli untested for install |
| 4 | non-blocking | `--agent` flag is accepted but a pure no-op (pi/opencode deferred) — could mislead a user passing `--agent pi` | code: flag parsed then discarded |
| 5 | non-blocking (minor) | child name substituted into BOARD.md headings without sanitization | `writeBoard` uses `ReplaceAll` on the raw name |

No blocking findings: the golden path is correct, install is idempotent and board-audit-clean,
and every task/decision is honoured. All five items are polish/hardening on the install surface
and were collected into a single new ticket **T-013** (`depends-on: [T-004]`). Two plan
deviations noted at hand-off are accepted as-is: flag form `--project`/`--path` (vs the
Description's inline syntax) and `.gitkeep` closing the empty-dir gap deferred from T-002.

### Impact sweep (step 8)

Tickets depending on T-004: **T-005** (doctor), **T-006** (upgrade+uninstall), **T-009**
(opencode), **T-010** (pi), and the new **T-013**. T-004 fixes the concrete artifact set those
tickets must operate on — recorded a short pointer in T-005 and T-006's Descriptions (verify /
refresh / remove exactly: `.agents/skills/ticket-flow/` copy, `.claude/skills/ticket-flow`
symlink, `AGENTS.md`+`CLAUDE.md` marker pair, `tickets/**/.gitkeep`, `pickle.toml`). T-009/T-010
must extend install for their agents (they were always scoped to). No assumptions invalidated.

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
- 2026-07-23 — TO DO → READY: implementation plan complete (READY gate met); prerequisite T-001 done+merged. Scope pinned to the core reference install; pi/opencode matrix deferred to T-009/T-010.
- 2026-07-23 — READY → IN DEVELOPMENT: picked up, branch feat/T-004-install (applicability gate clean)
- 2026-07-23 — IN DEVELOPMENT → IN REVIEW: acceptance test green (fresh install -> layout + board audit clean + ticket new works; idempotent re-run preserves data, one marker pair)
- 2026-07-23 — IN REVIEW → DONE: review PASS; no blocking findings; 5 non-blocking -> T-013
