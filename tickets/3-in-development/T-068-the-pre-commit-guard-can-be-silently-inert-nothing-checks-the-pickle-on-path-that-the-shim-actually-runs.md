---
id: T-068
title: the pre-commit guard can be silently inert: nothing checks the pickle on PATH that the shim actually runs
project: pickle
depends-on: []
spawned-by: [T-057]
impact: medium
complexity: medium
cost: M
---

# T-068 — the pre-commit guard can be silently inert: nothing checks the pickle on PATH that the shim actually runs

## Description

T-057 shipped a `pre-commit` guard whose whole point is to stop a failure that had already
happened four times. **Within minutes of arming it in this repo, it was measured to be inert**,
and both `pickle hooks status` and `pickle doctor` reported it as healthy.

### The measurement (2026-08-06, immediately after T-057 merged as `9a9af59`)

`pickle hooks install` wrote the shim, and the two commands that exist to tell you the guard's
state said it was fine:

```
$ pickle hooks status
pre-commit: installed by pickle, current (/…/pickle/.git/hooks/pre-commit)
$ pickle doctor -v | grep pre-commit
ok: pre-commit guard installed and current (/…/pickle/.git/hooks/pre-commit)
```

A live smoke test on a throwaway `feat/T-000-guard-smoke` branch, staging a file under
`tickets/`, then showed the truth:

```
pickle: unknown command "hooks"
pickle: bookkeeping guard skipped (hooks run exited 2)
[feat/T-000-guard-smoke 603e13e] docs(tickets): smoke (T-000)     ← committed anyway
```

The cause is not a bug: the shim resolves `pickle` from `PATH`, and that is
`/opt/homebrew/bin/pickle` at **`0.2.2`** — a binary released before the `hooks` verb existed. It
exits `2` on the unknown verb, and the shim treats anything but `1` as "no violation"
(T-057 decision 3/4). **That fail-open is correct and must stay** — the alternative,
`|| exit 1`, would have blocked *every* commit in this repository instead. The defect is that
**nothing joins the two facts up**: the file on disk is current, the binary that will execute it
is not, and only a deliberate commit-and-observe test reveals it.

**A second, cosmetic half of the same skew** (observed on every commit in this repo since):
the old binary answers the unknown verb by dumping its **entire `usage()` text** before the
shim's one-line notice, so each commit prints ~35 lines of unrelated help. Whatever the fix,
`hooks run` on an unknown verb should be terse — the shim's notice is the message that matters,
and burying it is how a notice becomes noise people filter out.

### Why this matters more than a normal doctor gap

- The guard's value is entirely in the moments nobody is watching. A guard that reports itself
  healthy while doing nothing is **worse than an absent one**, because the absent one is
  reported honestly today (`pre-commit guard not installed (optional …)`).
- The skew is the *normal* state, not an edge case: `pickle hooks install` is run from a
  freshly-built binary (that is what the T-057 handback instructs), while `PATH` usually holds a
  packaged release. Every user who builds from source, uses `go install` alongside Homebrew, or
  keeps a second clone lands in it, and Homebrew lag alone guarantees a window after every
  release.
- It is the *measured* twin of T-057's review finding **F9** (`noted`): GUI/IDE clients (Fork,
  SourceTree, JetBrains) commit with a minimal `PATH` where `command -v pickle` fails outright,
  so the guard is skipped there too. Same failure — *the shim cannot reach a usable pickle* —
  same invisibility. Batched here rather than left as two notes.

### Shape of the fix (for refinement — nothing is decided)

1. **A capability probe, and where it lives.** `hook.Status` currently answers "what is on
   disk". It could also answer "can the binary the shim will call actually run the guard?" —
   `exec.LookPath("pickle")`, then compare against `os.Executable()` and/or probe the verb. Note
   T-057 decision 3 deliberately left room for this: *"a `--probe` handshake"* was one of the
   two sanctioned answers to version skew, and the exit-code contract (`1` = violation, only
   ever) is what makes a probe safe to add without touching the shim.
2. **Warn at install time, which is where the evidence appears.** `pickle hooks install` knows
   its own path and version; if the `pickle` first on `PATH` is a different file or an older
   version, say so *then* — that is the one moment the user is looking, and it would have caught
   this instance without a smoke test.
3. **doctor's verdict.** Today: `ok: pre-commit guard installed and current`. It should be able
   to say `warning: … but the pickle on PATH (0.2.2 at /opt/homebrew/bin/pickle) cannot run it —
   the guard is inert`. Constraint: `internal/doctor` is documented as a pure, fixture-testable
   check, and T-057 confined `os/exec` to `internal/hook` on purpose (decision 12). Any probe
   must stay behind `internal/hook`, and the fixtures must be able to fake both answers.
4. **Should the shim hard-code the installing binary's absolute path?** It would fix both this
   and F9's minimal-`PATH` case outright — and it would break the moment the binary moves or is
   replaced by a package upgrade, and it makes the hook non-portable across clones on shared
   filesystems. Probably no, possibly as an opt-in flag; refinement must decide rather than
   inherit the question.
5. **Does anything else in the tree deserve the same treatment?** `pickle serve`, the agent
   scaffolds and `opencode.jsonc` are all "pickle wrote a file that something else executes".
   Only the hook has a *version-coupled contract* with the binary, so the answer is probably no
   — worth one paragraph to close, not a sweep.

### Re-measured at refinement (2026-08-06)

The Description above still holds verbatim — the skew is live in this repo right now
(`/opt/homebrew/bin/pickle` → `pickle 0.2.2`, `./pickle` → `v0.2.2-71-g889129e`, an owned
current shim in `.git/hooks/pre-commit`). Three facts were added by measurement:

- **The probe has a zero-surface form.** `pickle hooks run pre-commit` executed with its working
  directory in an **empty** `/tmp` dir exits **0** on a hooks-aware binary (`config.Find` finds no
  `pickle.toml` and the guard degrades, `internal/cli/hooks.go`) and **2** on `0.2.2`. Measured
  both ways. So the capability question can be asked with *the very call the shim makes*, needing
  no new verb.
- **`hooks` has never been released.** `0.2.2` is the newest tag and the verb sits under
  `## [Unreleased]` in `CHANGELOG.md`. That is what disqualifies a dedicated `pickle hooks probe`
  subcommand: builds of `main` made between T-057 and this ticket answer `hooks run` but would
  exit 2 on `probe`, i.e. report a *working* guard as inert.
- **The old binary's stderr is 41 lines** (measured), which is the cosmetic half of the theme, and
  the shipped shim's own marker line reads **`# # pickle:hook v1`** — `markerPrefix` already
  carries the `#` and `Shim()` prepends a second one (`internal/hook/hook.go:56,64`). Cosmetic,
  found here, fixed in the same shim bump rather than left to rot.

### Soft couplings

- **T-046** (make doctor/upgrade self-host-aware) — the nearest neighbour and the reason this was
  nearly folded there: both make `doctor` tell the truth instead of a convenient default. Kept
  separate because T-046's ground is *self-host* noise (a symlinked skill dir, a
  `payload_version` stamp that cannot be compared), while this is **version skew in any
  install**, and its evidence is a guarantee that silently does not hold. They touch the same
  file (`internal/doctor/doctor.go`) — sequence them, do not run them concurrently.
- **T-057** — lineage (`spawned-by`), and the source of the fail-open contract this ticket must
  not weaken. Its review finding **F9** is folded in above as the second half of the theme.
- **T-065** (JSON read projection) — if `doctor` ever grows machine-readable output, a
  "guard reachable: yes/no" field belongs in it; no dependency either way.

## Implementation Plan

### Feature branch

`feat/T-068-probe-the-pickle-on-path`, cut from `main` in the `pickle` child (this repo).
Local WIP commits are encouraged; **no push and no MR without explicit user approval** — then
finalize + push + open the MR; merging is the human's.

This branch is subject to the rule the guard it repairs enforces: every `tickets/` edit (this
plan, the move to `3-in-development/`, the move to `4-in-review/`) is committed **on `main`**.

### Prerequisites

None; no `depends-on:`. **Do not run concurrently with T-046** — it edits the same
`internal/doctor/doctor.go` (`checkHooks`/`checkVersion`); whichever lands second rebases.
Self-modify policy: never run `pickle install|upgrade|uninstall|hooks install` against this repo
from this branch — the acceptance transcript runs in a throwaway `/tmp` dir with the binary
copied in.

### Confirmed decisions

1. **The probe is the shim's own call, in a neutral directory.** `<pickle-on-PATH> hooks run
   pre-commit` with `Dir` set to a fresh empty temp dir and a **3 s** timeout: exit `0` ⇒ the
   binary can run the guard, anything else ⇒ it cannot. Highest fidelity (it is literally what
   `.git/hooks/pre-commit` will execute), zero new CLI surface, and no false alarm for any
   hooks-aware build. Rejected: a dedicated `pickle hooks probe` verb (new surface *and* it would
   report every `main` build between T-057 and this ticket as inert — see *Re-measured*), and
   comparing version strings (needs semver ordering and has no answer for `dev`).
   T-057 decision 3 sanctioned exactly this handshake, and its exit-code contract (`1` = violation,
   only ever) is what keeps the probe safe: the probe never sees `1`, because a violation needs a
   git repo on a feature branch with staged `tickets/` and the probe dir is empty.
2. **Same-file short circuit, and path difference is never itself a finding.** If
   `exec.LookPath("pickle")` resolves to the same file as `os.Executable()` (`os.SameFile`), report
   healthy without exec'ing anything. A *different but capable* binary is healthy — warning on the
   path alone would fire on the normal "built from source, Homebrew copy of the same version"
   setup and train users to ignore the warning.
3. **The version is best-effort, and only when the probe fails.** One further
   `<path> version` exec (first line, trimmed, capped) so the message can read
   `0.2.2 at /opt/homebrew/bin/pickle`, as the Description asks. Errors ignored; the *old binary's*
   stderr is never quoted (it is the 41-line usage dump).
4. **Where it surfaces: the three moments the user is looking.** `pickle hooks install` and
   `pickle install --hooks` (right after arming — the one moment the evidence exists and would have
   caught this instance without a smoke test), `pickle hooks status` (a second line), and
   `pickle doctor` (a **warning**). **Not** `pickle upgrade`: its result is a created/skipped path
   list with no warning channel (`internal/install/install.go:334`), and doctor already covers the
   state it would report.
5. **Probe only an owned, installed shim.** `KindAbsent`/`KindForeign`/`KindNoRepo` keep today's
   exact lines and pay no exec cost — PATH reachability is only a defect when pickle's own shim is
   armed. Three verdicts when it is: **not on PATH at all** → warning (the terminal-side twin of
   T-057 finding F9, and a real signal: doctor was itself reached some other way); **found and
   capable** → `ok`, naming the resolved path; **found and incapable** → warning saying the guard
   is **inert**, naming path and version.
6. **Shim v2.** The `command -v pickle` branch stops exiting silently and prints one stderr line,
   for the same reason T-057 decision 4 makes an unexpected exit code speak: silence hides a dead
   guard, and that is the failure this ticket exists to close. Changing the shim text **bumps
   `ShimVersion` to 2** — that is the mechanism's purpose: `pickle upgrade` then refreshes owned
   shims and `doctor` reports a v1 shim as written by an older pickle. The double-hashed marker line
   (`# # pickle:hook v1`) is fixed in the same bump. Detection is unaffected: `markerVersion` does
   an `Index` of `# pickle:hook v`, which still matches both the old and the new line.
7. **The fail-open contract does not move.** Exit `1` stays the only blocking code; nothing in this
   ticket may make a failed probe block a commit. The probe *reports*; the shim still waves
   everything but `1` through.
8. **The shim does not hard-code the installing binary's absolute path** — not even behind a flag.
   It would break the moment a package upgrade or `go install` relocates the binary, it is
   non-portable across clones on a shared filesystem, and decisively: pickle's ownership *and*
   staleness checks rest on byte-comparing the file against `Shim()`
   (`hook.Install`, `hook.Refresh`, `doctor.checkHooks`), so per-install text would fork that
   contract for every one of them. The minimal-`PATH` case (F9) is documented, not engineered
   around.
9. **Unknown commands answer in one line.** `pickle: unknown command "hooks" — run `pickle help``,
   exit 2, no usage dump; bare `pickle` with no arguments still prints the full usage. This is the
   cosmetic half of the theme, and honestly it only pays off in *future* skew windows — today's
   noise in this repo ends when the Homebrew copy catches up.
10. **No sweep of the other "pickle wrote a file something else executes" surfaces.**
    `pickle serve`, the pi scaffolds and `opencode.jsonc` have no version-coupled contract with the
    binary — the scaffolds are drift-checked by content and nothing re-executes them through
    `PATH`. Recorded as a comment in `checkHooks`, not as work.
11. **`os/exec` stays confined to `internal/hook`** (T-057 decision 12) and doctor stays
    fixture-testable: the probe is faked by putting `#!/bin/sh` stubs on a `t.Setenv("PATH", …)`,
    which needs no injection seam and no interface — the same trick `TestShimExitCodes` already
    uses.

### Tasks

**1. New `internal/hook/probe.go`.**

```go
// Reach describes the pickle the installed shim will actually execute.
type Reach struct {
	Path    string // resolved `pickle` on PATH; "" when there is none
	Self    bool   // Path is this very binary (no exec was needed)
	OK      bool   // it can run `hooks run pre-commit`
	Version string // best-effort `<Path> version` line, only when OK is false
}

func Probe() Reach            // uses probeTimeout
func (r Reach) Problem() string // "" when healthy; one sentence otherwise
```

- `const probeTimeout = 3 * time.Second`.
- `exec.LookPath("pickle")` fails → `Reach{}` (the F9 case).
- `os.Executable()` + `os.SameFile` on both stats → `Reach{Path, Self: true, OK: true}`, **no exec**.
- otherwise `exec.CommandContext(ctx, path, "hooks", "run", HookName)` with `Dir` = a fresh
  `os.MkdirTemp` dir (removed with `defer`), `Env = withoutRepoEnv(os.Environ())` (reuse the helper —
  an inherited `GIT_DIR` must not point the probed binary at another repo), and **stdout/stderr
  discarded**: the old binary's usage dump must never reach the user. `ctx` deadline or any non-zero
  exit ⇒ `OK: false`, then fill `Version` via `probeVersion(path)` (`<path> version`, stdout's first
  line, trimmed, capped at 60 runes, all errors ignored).
- `Problem()` is the single source of the wording (three call sites): `"no pickle on PATH — the
  guard is inert (the shim resolves pickle from PATH)"` / `"the pickle on PATH cannot run the guard
  — it is inert (<version-or-"unknown version"> at <path>)"`. Callers add their own prefix.
- Doc comment carries decisions 1, 2, 3 and 7 — in particular *why* an empty working directory
  guarantees exit 0 on a hooks-aware binary (`config.Find` degrade path), because that is the
  invariant the whole check rests on, and it must not be "tidied" into a bare `--help` probe.

**2. `internal/hook/hook.go` — shim v2** (decision 6).

- `ShimVersion = 2`.
- marker line loses the duplicated hash: emit `marker() + " — installed by …"`, not `"# " + marker()`.
- the guard-absent branch speaks:
  ```sh
  command -v pickle >/dev/null 2>&1 || {
    echo "pickle: bookkeeping guard skipped (pickle not found on PATH)" >&2
    exit 0                                     # never blocking
  }
  ```
  POSIX `sh` only, still exit 0. Extend `Shim()`'s doc comment: the notice exists for the same
  reason as the unexpected-exit-code line, and this branch must never grow an `exit 1`.

**3. `internal/hook/hook_test.go` + new `internal/hook/probe_test.go`.**

- Replace the literal `"pickle:hook v1"` in the stale-shim fixtures with a value derived from
  `hook.ShimVersion` (here **and** in `internal/doctor/hooks_test.go`), so the next bump does not
  silently stop testing staleness.
- Assert the shipped shim contains `"\n# pickle:hook v2 —"` and **not** `"# # pickle"`.
- `TestShimNoticesMissingPickle`: run the real shim with `PATH` pointing at an empty dir → exit 0,
  stderr contains `not found on PATH` (extends the existing `TestShimExitCodes` table).
- Probe cases, each a `#!/bin/sh` stub on `t.Setenv("PATH", dir)`: capable (`exit 0`) → `OK`,
  `Problem() == ""`; incapable (`exit 2`) → `!OK` and `Problem()` names the path; incapable +
  `version` handler → `Version == "pickle 0.2.2"`; absent (empty PATH dir) → `Path == ""` and
  `Problem()` mentions PATH; `sleep 10` stub → the 3 s deadline returns `!OK` (assert it returns in
  well under 10 s); `Self` short-circuit by symlinking `os.Executable()` into the fake bin as
  `pickle` → `Self && OK` with no exec.

**4. `internal/doctor/doctor.go` — `checkHooks`.** In the `KindOwned` + not-stale branch only
(decision 5), call `hook.Probe()`:

```go
if p := reach.Problem(); p != "" {
	r.warnf("hooks: %s is installed and current, but %s", st.Path, p)
	return
}
r.ok(fmt.Sprintf("pre-commit guard installed and current (%s), and the pickle on PATH can run it (%s)", st.Path, reach.Path))
```

Keep the existing `ok` wording as the prefix so the passing line stays greppable. Comment records
decision 10 (no sweep) and decision 11 (the exec lives behind `internal/hook`; doctor stays a
findings-returning function that never prints).

**5. `internal/doctor/hooks_test.go`.** With a real `git init` fixture and an owned shim: an old
stub on `PATH` → warning containing `inert` and the stub's path; a capable stub → no warnings and
the `can run it` pass line; an empty `PATH` → warning naming PATH. Plus the negative: with the hook
**absent** and an old stub on `PATH`, `Check` produces **no** findings — the probe must not run for
a guard that is not armed.

**6. `internal/cli/hooks.go` + `internal/cli/install.go` — say it at install time** (decision 4).

- `runHooksInstall`: after a successful `hook.Install` (changed *or* `= (current)`), print
  `pickle hooks install: warning: <Problem()>` to **stderr** plus one remedy line naming the
  installing binary's own path (`os.Executable()`), e.g. `this binary is <path> — put it first on
  PATH, or upgrade the pickle that is`. Exit code stays 0: the file was written correctly.
- `runHooksStatus`: in the `KindOwned` branch add a second line — the problem, or
  `  PATH: <path> can run the guard`.
- `internal/cli/install.go` (`--hooks`, around line 101): same warning after the successful
  `hook.Install`; it must not fail the install.
- Factor the two-line rendering into one unexported helper in `internal/cli/hooks.go` so the three
  call sites cannot drift.

**7. `internal/cli/cli.go` — terse unknown command** (decision 9): replace the
`unknown command` + `usage(os.Stderr)` pair with a single line pointing at `pickle help`, keeping
exit 2. Leave the no-argument path (full usage) alone.

**8. `internal/cli/cli_test.go`.** Unknown command → exit 2, stderr is one line, contains
`pickle help`, and contains **neither** `Usage:` nor `Flow commands:`; no-args → still the full
usage (guard the deliberate asymmetry). `hooks status`/`hooks install` with an old stub first on
`PATH` → the warning line; with the real binary → no warning.

**9. Docs** (`just docs-check` must stay green; keep the `hooks` section self-contained per T-057
finding F10, and add no new xrefs until T-067 lands anchor validation).

- `docs/user-manual/cli-reference.adoc`, `[#cmd-hooks]`: the marker is `# pickle:hook v2`; the
  best-effort/terminal-first bullet gains the **skew** twin (an older `pickle` first on `PATH`
  cannot run the guard, and the shim's fail-open means the commit goes through — `hooks install`,
  `hooks status` and `doctor` all say so now); the fail-open bullet gains the new stderr notice for
  a `pickle` that is not on `PATH` at all; `status` is described as reporting the PATH verdict too.
- same file, `[#cmd-doctor]` bullet list: the guard bullet gains the inert-guard *warning*.
- `docs/user-manual/installation.adoc` (the `pickle hooks install` paragraph, ~line 60): one
  sentence — the guard runs whatever `pickle` is first on your `PATH`, so if you installed from a
  freshly-built binary while `PATH` holds an older release, `pickle doctor` will tell you the guard
  is inert.
- `DESIGN.md:228`: `# pickle:hook v1` → version-agnostic `# pickle:hook` marker, and one clause
  that the binary the shim resolves from `PATH` is probed rather than assumed.
- `CHANGELOG.md` `## [Unreleased]`: correct the T-057 bullet's marker to `# pickle:hook v2` (the
  verb is unreleased, so there is one shipped shim version, not two), and add a bullet for T-068 —
  the PATH probe at install/status/doctor, the shim's new missing-pickle notice, and the terse
  unknown-command output.
- No skill-payload change: the rule the guard enforces is unchanged, only its self-report.

### Acceptance test

All four child commands, from the repo root:

```sh
just build && just test && just lint && just docs-check
```

Then the end-to-end transcript. **Write it to a file under `/tmp` and run it with `sh -e`** — it
cannot be pasted into a pi session in this repo (`.pi/extensions/workspace-guardrails.ts` blocks
`pickle install` segments that do not clearly target a throwaway dir), and the literal `/tmp`
prefix matters: bare `mktemp -d` yields `/var/folders/…` on macOS, which the self-modify guard does
not recognise as throwaway.

```sh
just build
D=$(mktemp -d /tmp/t068.XXXXXX)
mkdir -p "$D/bin" "$D/old" "$D/empty" && cp pickle "$D/bin/pickle"
# a stand-in for 0.2.2: knows `version`, exits 2 on everything else (as `hooks` did not exist)
printf '#!/bin/sh\ncase "$1" in version) echo "pickle 0.2.2";; *) echo "pickle: unknown command" >&2; exit 2;; esac\n' > "$D/old/pickle"
chmod +x "$D/old/pickle"
mkdir -p "$D/repo" && cd "$D/repo" && git init -q -b main .
git config user.email t@example.com && git config user.name test

export PATH="$D/bin:$PATH"
pickle install --project demo --hooks          # 1. armed, and PATH pickle IS this binary → no warning
grep -q '^# pickle:hook v2 ' .git/hooks/pre-commit   #    marker line, single hash (task 2)
pickle doctor -v | grep 'can run it'          #    healthy pass line

# 2. old pickle first on PATH: the measured failure, now reported at all three surfaces
PATH="$D/old:$PATH" "$D/bin/pickle" hooks install 2>&1 | grep -e inert -e 0.2.2
PATH="$D/old:$PATH" "$D/bin/pickle" hooks status  2>&1 | grep inert
PATH="$D/old:$PATH" "$D/bin/pickle" doctor        2>&1 | grep inert   # exit 0: a warning, not an error

# 3. no pickle on PATH at all (T-057 finding F9's twin)
PATH="$D/empty" "$D/bin/pickle" doctor 2>&1 | grep -i 'no pickle on PATH'

# 4. the shim's own notice, and the fail-open it must preserve
git add pickle.toml tickets AGENTS.md && git commit -qm 'chore: scaffold'
git checkout -qb feat/T-001-demo
pickle ticket new "demo" --project demo >/dev/null
git add tickets
( PATH="$D/empty"; git commit -qm 'guard absent' 2>&1 | grep 'not found on PATH' )   # MUST SUCCEED

# 5. the guard still bites when it can run (the contract this ticket must not weaken)
git reset -q --soft HEAD~1 && set +e
git commit -qm 'MUST FAIL'; rc=$?; set -e; test "$rc" -eq 1
git restore --staged tickets/

# 6. shim v2 is a refreshable bump, not a silent text change
sed -i.bak 's/pickle:hook v2/pickle:hook v1/' .git/hooks/pre-commit && rm .git/hooks/pre-commit.bak
pickle doctor 2>&1 | grep 'older pickle'
pickle upgrade >/dev/null && grep -q '^# pickle:hook v2 ' .git/hooks/pre-commit

# 7. unknown verbs answer in one line, not 41
test "$("$D/bin/pickle" nosuchverb 2>&1 | wc -l | tr -d ' ')" = 1
set +e; "$D/bin/pickle" nosuchverb >/dev/null 2>&1; test $? -eq 2; set -e
```

Expected: every step passes, step 4's commit **succeeds** with the new stderr notice, step 5's
commit **fails with exit 1** (the guard, working), and steps 2–3 print warnings while `doctor`
still exits 0. Re-runnable verbatim from a clean `mktemp -d`.

### Docs update

Task 9: `docs/user-manual/cli-reference.adoc` (`[#cmd-hooks]` + the `doctor` bullet list),
`docs/user-manual/installation.adoc`, `DESIGN.md:228`, `CHANGELOG.md`. No skill-payload or
marker-block change — the flow rule is untouched, only pickle's report of whether it can enforce
it — so **no hand-mirrored `AGENTS.md` edit is needed** in this ticket.

### Finish

Summary of what shipped (probe, shim v2, three surfaces, terse unknown command), the deviations
section reviewers rely on, then the acceptance transcript's result. Suggested commit message:

```
feat(hooks): probe the pickle on PATH so an inert guard reports itself (T-068)
```

Local WIP commits on the branch; **no push and no MR without explicit user approval**. Then
`pickle ticket move T-068 in-review --reason "acceptance green"` (committed on `main`) and hand
back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-06 — created (TO DO). source: pickle ticket new
- 2026-08-06 — filed from the T-057 post-merge verification, not from its review: arming the guard
  in this repo and smoke-testing it showed `hooks status` and `doctor` both reporting a healthy
  guard while the `pickle` on `PATH` (0.2.2, pre-`hooks`) made it a no-op. Batches T-057 review
  finding F9 (GUI/IDE minimal `PATH`) as the same theme. Graded medium/low/S: it voids a
  just-shipped guarantee, and the fix is a probe plus two message changes
- 2026-08-06 — refined: not split (every part is a task; none is independently schedulable). The
  Description was re-verified live and holds; three measurements were added. **D1**, the probe, is
  `<pickle-on-PATH> hooks run pre-commit` in an empty temp dir (measured: 0 on a hooks-aware build,
  2 on Homebrew's 0.2.2) — the shim's own call, no new verb; a dedicated `hooks probe` subcommand
  was rejected because `hooks` is unreleased, so every `main` build made since T-057 would be
  misreported as inert. Path difference alone never warns (**D2**); the probe runs only for an
  owned, armed shim (**D5**) and surfaces at `hooks install`, `hooks status` and `doctor` — not
  `upgrade`, which has no warning channel (**D4**). The shim bumps to **v2** (**D6**): the
  guard-absent branch now speaks instead of exiting silently, and the shipped `# # pickle:hook v1`
  double hash is fixed in the same bump. Hard-coding the installing binary's path was **declined**
  (**D8**) — ownership and staleness both rest on byte-comparing the file against `Shim()`.
  Unknown commands answer in one line instead of 41 (**D9**); no sweep of serve / scaffolds /
  `opencode.jsonc` (**D10**). Re-graded complexity low → medium, cost S → M (shim bump + probe +
  four call sites + docs); impact stays medium. T-046 must not run concurrently (same `checkHooks`)
- 2026-08-06 — TO DO → READY: plan complete
- 2026-08-06 — READY → IN DEVELOPMENT: picked up
