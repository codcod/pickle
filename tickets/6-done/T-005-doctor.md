---
id: T-005
title: doctor
project: pickle
depends-on: [T-004]
impact: medium
complexity: low
cost: M
---

# T-005 — doctor

## Description

Implement `pickle doctor` — the install-side analogue of `board audit`. Verify install
integrity and report actionable diagnostics: the skill is present with the expected payload
version, the `.agents/`/`.claude/` symlinks resolve, the `<!-- pickle:begin/end -->` marker
block is present in `AGENTS.md`/`CLAUDE.md`, `pickle.toml` parses, and every registered
child-project path resolves to a git repository. Exit non-zero when anything is wrong. Needs
T-001 and T-004. Phase P2.

> **T-004 artifact set (what `doctor` verifies)**, as implemented in `internal/install`:
> `.agents/skills/ticket-flow/` (copied payload — real dir, unless a dev/self-host symlink),
> `.claude/skills/ticket-flow -> ../../.agents/skills/ticket-flow`, the
> `<!-- pickle:begin -->`/`<!-- pickle:end -->` pair in `AGENTS.md` (+`CLAUDE.md`, or a
> `CLAUDE.md -> AGENTS.md` symlink under `--claude-symlink`), a `.gitkeep` in each of the seven
> `tickets/<status>/` dirs, `tickets/BOARD.md` + `tickets/README.md`, and `pickle.toml`. Reuse
> `internal/install`'s constants (`skillDir`, `markerBegin/End`) rather than re-hardcoding.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle`'s sole child is the repo root (`.`), so the branch is cut in this repo:

```
git checkout main
git checkout -b feat/T-005-doctor
```

Local WIP commits are encouraged; **do not push or open an MR without explicit user
approval** (child is publish-gated — see `pickle.toml` / `AGENTS.md`).

### Prerequisite gate (hard)

- T-004 (install) is in `6-done/` and merged to `main` (33f05e3) — satisfied.
- Clean working tree on the new branch.

### Confirmed design decisions (do not deviate without asking)

1. **Mirror the `audit` package shape.** `pickle doctor` is the install-side analogue of
   `board audit`: a **pure, fixture-testable** checker in a new package
   `internal/doctor/` that returns findings and never prints or calls `os.Exit`. The CLI
   handler (`internal/cli/install.go` → `runDoctor`) prints findings + a summary and maps
   the result to an exit code, exactly like `runBoardAudit`.
2. **Errors vs. warnings.** Anything broken (missing/blank skill, unresolved symlink, missing
   marker block, `pickle.toml` that does not parse/validate, a child path that is not a git
   repo) is an **error** → non-zero exit (`exitError`). Advisory conditions (payload-version
   drift vs. the running binary) are **warnings** → exit 0. This matches `audit.Result`
   (`Errors`/`Warnings`) and `runBoardAudit`'s exit rule (non-zero iff `len(Errors) > 0`).
3. **Reuse `internal/install` constants — do not re-hardcode.** Export the existing
   identifiers from `internal/install/install.go` and reference them from `doctor`:
   - `skillDir` → `install.SkillDir` (`.agents/skills/ticket-flow`)
   - `markerBegin` → `install.MarkerBegin`, `markerEnd` → `install.MarkerEnd`
   - add `install.ClaudeSkillLink` (`.claude/skills/ticket-flow`) and
     `install.ClaudeSkillTarget` (`../../.agents/skills/ticket-flow`), and use them in
     `Run`'s `ensureSymlink` call so the target string has one home.
   Update all in-package references in `install.go` to the exported names (no behaviour
   change; `install_test.go` must stay green).
4. **Self-host / dev symlink is valid.** `.agents/skills/ticket-flow` may legitimately be a
   **symlink** (this repo self-hosts by linking to `skill/`), just as `copyPayload` skips it.
   Doctor follows symlinks (`os.Stat`) — a resolvable skill dir containing `SKILL.md` and
   `resources/tickets-README.md` passes whether it is a real dir or a symlink.
5. **Claude artifacts are optional.** Install can run with `--no-claude`. Doctor treats the
   Claude view as **present-or-absent, not required**: if `.claude/skills/ticket-flow` exists
   it must be a symlink that resolves; if the whole `.claude/` path is absent, that is not a
   finding. Likewise `CLAUDE.md`: if it exists as a regular file it must carry the marker
   block; if it is a symlink it must resolve; if absent, skip.
6. **`AGENTS.md` marker block is required.** `AGENTS.md` must exist and contain a
   `MarkerBegin … MarkerEnd` pair (begin index < end index) — the install invariant.
7. **Config-parse failure is a finding, not a crash.** Doctor loads `pickle.toml` itself
   (via `config.Load`); a parse/validate failure becomes one error and doctor still runs the
   filesystem checks that do not need `cfg` (skill, symlink, markers). Per-child git checks
   run only when the config loaded.
8. **Git-repo check.** A registered child path "resolves to a git repository" iff a `.git`
   entry exists directly under the child's absolute path (a directory for a normal clone, or a
   file for a worktree/submodule). `config.Validate` already guarantees the path is a
   directory at load time; doctor adds the `.git` presence check on top.

### Tasks

#### Task 1 — export install constants
In `internal/install/install.go`: rename the `const` block identifiers to exported names
(`SkillDir`, `MarkerBegin`, `MarkerEnd`), add `ClaudeSkillLink = ".claude/skills/ticket-flow"`
and `ClaudeSkillTarget = "../../.agents/skills/ticket-flow"`, update every internal reference
(including the `ensureSymlink(... , "../../.agents/skills/ticket-flow", ...)` call in `Run`,
which now passes `install.ClaudeSkillTarget`). Keep `internal/install/install_test.go` green
(it hard-codes the target string in an assertion — leave the test's literal as-is or point it
at the new constant; do not change install behaviour).

#### Task 2 — the `doctor` package
Create `internal/doctor/doctor.go`, package `doctor`, mirroring `internal/audit`:

```go
type Result struct {
    Errors   []string
    Warnings []string
    Passed   []string // human-readable list of checks that held (for -v output)
}

// Check inspects an installed pickle project rooted at root (the dir holding
// pickle.toml). version is the running binary's version, for drift warnings.
func Check(root, version string) Result
```

Checks (each appends to `Errors`/`Warnings`/`Passed`; final slices sorted like `audit`):
- **config** — `config.Load(filepath.Join(root, config.FileName))`; on error, one error
   `"pickle.toml: <err>"` and leave `cfg == nil`.
- **skill** — `install.SkillDir` resolves (`os.Stat`, following symlinks) to a directory
   containing `SKILL.md` and `resources/tickets-README.md`; error otherwise.
- **claude view** — if `install.ClaudeSkillLink` exists: `os.Lstat` says symlink **and**
   `os.Stat` resolves; error if it exists but is not a resolving symlink. Absent ⇒ skip.
- **markers** — read `AGENTS.md`; error if missing or if it lacks a `MarkerBegin`/`MarkerEnd`
   pair with begin < end. For `CLAUDE.md`: regular file ⇒ same marker check; symlink ⇒ must
   resolve; absent ⇒ skip.
- **children** (only if `cfg != nil`) — for each `cfg.Projects`, check `.git` exists under
   `filepath.Join(root, p.Path)`; error `"child <name>: <path> is not a git repository"`
   otherwise.
- **version drift** (only if `cfg != nil`) — if `version != ""`, `version != "dev"`, and
   `cfg.PayloadVersion != version`: warning suggesting `pickle upgrade`.

#### Task 3 — wire the CLI
In `internal/cli/install.go`, replace the `runDoctor` stub with a real handler modelled on
`runBoardAudit`:
- accept an optional `-v`/`--verbose` flag (via `flag.NewFlagSet`); unknown flags ⇒
  `exitUsage`.
- locate the project root: `config.Find(wd)` → `root = filepath.Dir(path)`; if `Find` fails,
  `errf("%v", err)` (not a pickle project) and return `exitError`.
- `res := doctor.Check(root, Version)`; print `WARNING: …` / `ERROR: …` lines (and, under
  `-v`, `ok: …` for each `Passed`), then a summary
  `pickle doctor: N error(s), M warning(s)`.
- return `exitError` iff `len(res.Errors) > 0`, else `exitOK`.
Remove the now-obsolete `notImplemented("P2", "doctor", …)` call.

#### Task 4 — tests
- `internal/doctor/doctor_test.go`: install a fixture into `t.TempDir()` via
  `install.Run(os.DirFS(filepath.Join("..","..")), root, "test-ver", install.Options{...,
  Claude:true})` (same helper pattern as `install_test.go`), create a `.git` dir under the
  child path, and assert `Check` returns **no errors**. Then, in table-driven sub-tests,
  break one artifact at a time and assert the expected error substring appears and
  `len(Errors) > 0`: remove the skill dir; delete the child's `.git`; strip the marker block
  from `AGENTS.md`; point the Claude symlink at a missing target; corrupt `pickle.toml`.
  Add a version-drift sub-test asserting a warning (and zero errors) when
  `cfg.PayloadVersion != version`.
- `internal/cli/cli_test.go`: remove the `"doctor stub" → exitUnimplement` row (doctor is no
  longer a stub; its behaviour is covered by the `doctor` package and depends on a project
  root, so it is not asserted via the payload-less dispatch table).

### Acceptance test

Run from the repo root:

```
just build
./pickle doctor            # self-host install is healthy → exits 0
echo "exit=$?"             # expect exit=0 (a payload-version-drift WARNING is allowed)
./pickle doctor -v         # also lists the passed checks
(cd /tmp && "$OLDPWD/pickle" doctor); echo "exit=$?"   # not a pickle project → non-zero
just test                  # all packages green, incl. internal/doctor
just lint                  # go vet + gofmt clean
```

Expected: `pickle doctor` prints a summary line and exits 0 in the repo (warnings allowed,
no errors); exits non-zero outside any pickle project; `just test` and `just lint` are clean.

### Docs update (mandatory when user-facing)

Update `README.md`:
- in the `## Commands` block, change the `pickle doctor` row from `[P2]` to `[done: T-005]`.
- in the prose below the block, move `doctor` out of the "remaining stubs"
  sentence (leaving `upgrade`/`uninstall`).

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` clean.
2. `README.md` updated.
3. Write a summary (files touched, decisions, anything deferred).
4. Suggested Conventional Commit:

   ```
   feat(cli): add doctor install-integrity check (T-005)

   Add `pickle doctor`: a pure, fixture-testable checker (internal/doctor)
   that verifies the skill payload, .claude symlink, AGENTS.md/CLAUDE.md
   marker block, pickle.toml, and each child's git repo, exiting non-zero on
   any error. Export the reused install constants.
   ```

5. Commit locally on `feat/T-005-doctor`; **do not push or open an MR without user
   approval**. Present the commit message, then move the ticket to IN REVIEW.

## Review

**Verdict: PASS (clean).** No blocking findings; one trivial cosmetic note, patched directly.
Reviewed on `feat/T-005-doctor` @ e498169 (+ this review's fixup).

### Checklist

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — README coverage present; no configured docs build (step 4a)
- [x] Findings classified & recorded (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep done — no ticket depends on T-005 (step 8)
- [x] Summary + commit message presented for approval; bookkeeping committed (step 9)

### Implementation audit (step 2)

| Item | Result | Evidence |
|---|---|---|
| Task 1 — export install constants | met | `install.SkillDir`/`MarkerBegin`/`MarkerEnd`/`ClaudeSkillLink`/`ClaudeSkillTarget`; all in-package refs updated; `install_test.go` green |
| Task 2 — `internal/doctor` package | met | pure `Check(root, version) Result`; config/skill/claude/markers/children/version checks; sorted output like `audit` |
| Task 3 — CLI wiring | met | `runDoctor` mirrors `runBoardAudit`; `-v`/`--verbose`; `config.Find`→root; exit non-zero iff errors |
| Task 4 — tests | met | `doctor_test.go`: healthy + 5 broken-artifact subtests + drift warning; `cli_test.go` stub row removed |
| Acceptance test | met | `./pickle doctor` exit 0 (drift warning only); `-v` lists 6 checks; outside a project exit 1; `just test`/`just lint` clean |
| Decisions D1–D8 | honoured | audit-shape, error/warning split, reused+exported constants, self-host symlink followed, Claude optional, AGENTS.md required, config-failure-is-a-finding, `.git` check |

### Findings

| # | Severity | Description | Evidence | Resolution |
|---|---|---|---|---|
| 1 | non-blocking (trivial) | `internal/cli/cli.go` `Payload` doc comment listed `doctor` among commands that "read from it" — but doctor inspects the on-disk install, not the embedded payload | cli.go:14 | Patched directly during review (protocol §5 trivial-cosmetic) |

**Notes.** A missing child *directory* surfaces via `config.Load` validation as a
`pickle.toml:` error rather than the `checkChildren` git-repo error — expected per decision #8
(`config.Validate` guarantees the dir exists at load time; doctor adds the `.git` check on
top). CLI-layer handler has no dedicated unit test, consistent with `runInstall`/`runBoardAudit`
(behaviour covered by the package tests).

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P2)
- 2026-07-24 — TO DO → READY: plan complete
- 2026-07-24 — READY → IN DEVELOPMENT: picked up, branch feat/T-005-doctor
- 2026-07-24 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-24 — IN REVIEW → DONE: review clean; trivial note patched inline
- 2026-07-24 — merged to main (b199215)
