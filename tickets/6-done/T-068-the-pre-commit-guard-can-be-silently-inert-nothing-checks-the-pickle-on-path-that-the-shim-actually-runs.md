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
mkdir -p "$D/bin" "$D/old" "$D/gitonly" && cp pickle "$D/bin/pickle"
# a stand-in for 0.2.2: knows `version`, exits 2 on everything else (as `hooks` did not exist)
printf '#!/bin/sh\ncase "$1" in version) echo "pickle 0.2.2";; *) echo "pickle: unknown command" >&2; exit 2;; esac\n' > "$D/old/pickle"
chmod +x "$D/old/pickle"
ln -s "$(command -v git)" "$D/gitonly/git"   # git, and deliberately no pickle (step 3)
mkdir -p "$D/repo" && cd "$D/repo" && git init -q -b main .
git config user.email t@example.com && git config user.name test

GIT_BIN=$(command -v git)   # step 4 needs git by absolute path: bash drops its command hash on a
                            # PATH reassignment, so `git` inside the subshell would not resolve
export PATH="$D/bin:$PATH"
pickle install --project demo --hooks          # 1. armed, and PATH pickle IS this binary → no warning
grep -q '^# pickle:hook v2 ' .git/hooks/pre-commit   #    marker line, single hash (task 2)
pickle doctor -v | grep 'can run it'          #    healthy pass line

# 2. old pickle first on PATH: the measured failure, now reported at all three surfaces
PATH="$D/old:$PATH" "$D/bin/pickle" hooks install 2>&1 | grep -e inert -e 0.2.2
PATH="$D/old:$PATH" "$D/bin/pickle" hooks status  2>&1 | grep inert
PATH="$D/old:$PATH" "$D/bin/pickle" doctor        2>&1 | grep inert   # exit 0: a warning, not an error

# 3. no pickle on PATH at all (T-057 finding F9's twin). $D/gitonly holds a symlink to git and
#    nothing else: dropping git too would send checkHooks down the *no-repo* path instead, which
#    reports "not applicable" and would test nothing (review finding R2).
PATH="$D/gitonly" "$D/bin/pickle" doctor 2>&1 | grep -i 'no pickle is on PATH'

# 4. the shim's own notice, and the fail-open it must preserve
git add pickle.toml tickets AGENTS.md && git commit -qm 'chore: scaffold'
git checkout -qb feat/T-001-demo
pickle ticket new "demo" --project demo >/dev/null
git add tickets
( PATH="$D/gitonly"; "$GIT_BIN" commit -qm 'guard absent' 2>&1 | grep 'not found on PATH' )  # MUST SUCCEED

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

> **Amended by the review (finding R2, `fixed inline`).** As first recorded this transcript could
> not run: step 3 used a wholly empty `PATH` dir (dropping `git`, which silently diverted
> `checkHooks` to its no-repo branch) *and* grepped `no pickle on PATH` against shipped text that
> reads `no pickle **is** on PATH`; step 4's `PATH=` reassignment made bash lose `git` itself
> (exit 127). The implementation notes below described two of the three fixes in prose but never
> applied them here, while the paragraph above still claimed verbatim re-runnability. The block
> above is now the version that actually ran, in both the implementation and the review.

### Implementation notes (2026-08-06) — deviations from the plan

All 9 tasks shipped on `feat/T-068-probe-the-pickle-on-path` (commit `1c5cc03`). The plan's
design (decisions 1–11) held up as written; three things surfaced only while building it, for
the reviewer:

1. **A real correctness bug caught by the acceptance transcript's own timeout test, not by the
   plan.** The first cut of `probeVersion` used `exec.Command(...).Output()`. `Output()` routes
   the child's stdout through an OS pipe drained by a goroutine that blocks until *every* holder
   of the pipe's write end closes it — and a version-manager shim on `PATH` (mise, asdf,
   direnv-style wrappers are common) that runs the real `pickle` **without** `exec`-replacing
   itself leaves that binary as a live grandchild still holding the pipe open long after `ctx`'s
   deadline kills only the direct child. Measured directly: a stub that runs `sleep 5` under a
   non-exec'd shell took **5.02s** to return through `Output()` despite a 100ms context timeout,
   versus 103ms through `Run()` (no pipe) with the identical stub. This would have defeated the
   one guarantee `probeTimeout` exists to give doctor/install/status — never hang. Fixed by
   writing `version`'s stdout to a real temp file (`os.CreateTemp` + `cmd.Stdout = f`) instead of
   an in-memory buffer: a file has no such wait, since the kernel writes into it directly and
   `Wait()` completing (the direct child reaped) is enough regardless of what an orphaned
   descendant still has open. `probeCapable` was never at risk — it uses `Run()` with no
   Stdout/Stderr set, which Go connects straight to `/dev/null` (no pipe, no copy goroutine).
2. **`probeTimeout` became a `var`, not the plan's implied `const`**, specifically so
   `TestProbeTimesOut` can shrink it instead of either paying a real 3s in every `go test` run or
   faking the deadline; restored via `t.Cleanup`.
3. **Doctor and CLI hook tests needed PATH pinned, not just faked — and pinning it naively
   re-introduced the bug it was meant to prevent.** `checkHooks`'s new probe call meant every
   test exercising an owned+current hook now depends on whatever `pickle` (if any) is first on
   the *test-runner's* real `PATH`. On this machine `git` and a real, pre-`hooks` `pickle` both
   live in the same Homebrew `bin/` — so an initial `pinPATH` that pointed at `filepath.Dir(gitPath)`
   to keep `git` resolvable leaked the real `pickle` straight back in, and the "no pickle on PATH"
   test failed by finding one anyway. Fixed by symlinking `git` alone into a fresh, otherwise-empty
   directory in both `internal/doctor/hooks_test.go`'s `pinPATH` and the equivalent added to
   `internal/cli/hooks_test.go`. The acceptance transcript hit the mirror-image version of the same
   class of bug: bash invalidates its command hash table on a `PATH` reassignment, so
   `( PATH="$D/empty"; git commit ... )` failed to find `git` *itself* (exit 127), not just the
   hook's inner `pickle` lookup as intended — fixed by resolving `git`'s absolute path once,
   before the reassignment, and invoking that directly.
4. **Literal `"pickle:hook v1"` fixtures, beyond the plan's named two files.** The plan named
   `internal/hook/hook_test.go` and `internal/doctor/hooks_test.go`; `internal/hook/hook_test.go`
   turned out to already derive its stale-shim fixture from `marker()`/`markerPrefix` (needed no
   change), but a third hardcoded literal surfaced in `internal/install/hooks_test.go`
   (`TestUpgradeRefreshesAStaleHook`) and was fixed the same way (derived from `hook.ShimVersion`).

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

**Verdict: PASS** (after one rework pass) — the first review found one blocking finding (R1); the
rework fixed it and the scoped re-review confirms it, so the ticket concludes to `6-done/`.
Audited on `feat/T-068-probe-the-pickle-on-path` — `1c5cc03` first (plus the review's own
inline-fix commit `a554c6c`), then `d8f62bf` — base `main`. The ticket was read from `main`
throughout: the branch's own copy is 44 lines stale, exactly the hazard rules §0 describes.

*First-review verdict (2026-08-06): REWORK on R1. The shipped behaviour was correct and fully
verified; what blocked was a user-manual sentence this branch made say the opposite of what it
ships.*

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [ ] Docs-readability pass (step 4b) — **skipped: reviewer unavailable.** The `docs_readability`
      tool errored identically to T-057's attempt (`gemini-2.5-pro` via github-copilot:
      `model_not_supported`), and **retried in the scoped re-review with the same failure**.
      Sanctioned conscious skip both times; the changed `.adoc` prose was read by hand, and the
      review's own xref-label fix (R9) came out of that read.
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `5-rework/` (step 6a), then back to `4-in-review/`, then to `6-done/` after
      the scoped re-review (step 6b); `## History` appended at every move
- [x] Board regenerated by each move; T-046 patched by the impact sweep (step 7)
- [x] Impact sweep done (step 8)
- [x] Summary + commit messages & MR attributes presented for approval; bookkeeping committed on
      `main` with explicit pathspecs (step 9)

### Implementation audit — all 9 tasks met

| task | verdict | evidence |
|---|---|---|
| 1 `internal/hook/probe.go` | met | 176 lines: `Reach{Path,Self,OK,Version}`, `Probe()`, `Problem()`, `sameExecutable`, `probeCapable`, `probeVersion`; `probeTimeout` is a `var` (deviation 2, so the timeout test costs 0.2s not 3s) |
| 2 shim v2 | met | `ShimVersion = 2`; marker line is `# pickle:hook v2 — …` (single hash, verified by `grep -q '^# pickle:hook v2 '`); guard-absent branch prints `pickle: bookkeeping guard skipped (pickle not found on PATH)` then `exit 0`. `sh -n`, `dash -n` and `shellcheck -s sh` all clean on the installed shim |
| 3 hook tests | met | `probe_test.go` (6 tests: capable / incapable / incapable-without-version / absent / timeout / Self short-circuit); `TestShimExitCodes` gained a `wantStderr` column asserting both degraded paths speak; `TestShimBlocksOnlyExitCodeOne` updated for the braces form and still fails if `\|\| exit 1` returns |
| 4 `doctor.checkHooks` | met | probes only the owned+current branch; warning text names path, version and "inert"; pass line gained `, and the pickle on PATH can run it` |
| 5 doctor tests | met | `TestCheckHooksProbesPATH` — 4 subtests incl. the negative ("absent hook never probes") |
| 6 three CLI surfaces | met | `warnIfInert` helper shared by `runHooksInstall` and `install --hooks`; `runHooksStatus` gained the second line; `internal/cli/hooks_test.go` +144 lines |
| 7 terse unknown command | met | `internal/cli/cli.go:78`, one line, exit 2 preserved |
| 8 CLI tests | met | `TestUnknownCommandIsTerse` (exactly one line, no `Usage:`/`Flow commands:`/`Setup commands:`) + `TestNoArgsStillPrintsFullUsage` guarding the asymmetry |
| 9 docs | met, with R1 | `cli-reference.adoc` `[#cmd-hooks]` + new `[#hooks-version-skew]` + doctor bullet, `installation.adoc`, `DESIGN.md`, `CHANGELOG.md` — but see **R1**: the `pickle help` section was left contradicting the task-7 change |

**Acceptance test.** The four child commands are green (`just build`, `just test` — 12 packages,
`just lint`, `just docs-check`). The 7-step transcript **as recorded did not run** (finding R2);
the corrected form passes end to end — `ALL 7 STEPS PASSED`, including step 4's commit succeeding
with the new notice and step 5's guard still blocking with exit 1. Beyond the plan, three
robustness checks were run: `-count=3` on the three affected packages (stable, no flakes from the
new exec-based tests); the suite under a **synthetic CI PATH with no `pickle` at all** (green —
proves no test depends on a real one, which mattered because `git` and a real `pickle` share a
directory on this machine); and shell linting of the shipped shim.

**All 11 confirmed decisions honoured**, including the two verified as *absences*: no `hooks probe`
verb was added (D1), and the shim still hard-codes no absolute path (D8). D7's fail-open contract
is intact — `TestShimExitCodes` and acceptance step 4 both prove a failed/absent probe never
blocks a commit.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | **blocking** | — | The manual's `pickle help` section still says the full usage summary is printed by "any unknown command" — the exact behaviour task 7 replaced. It is the **only** place the manual describes unknown-command output, so this ticket's user-facing D9 change has zero correct coverage and one inverted statement (protocol 4a.1). | `docs/user-manual/cli-reference.adoc:638-639`: "`pickle help` prints the usage summary; so does any unknown command, with a non-zero exit." vs. shipped `pickle: unknown command "nosuchverb" — run `pickle help``. Rendered into the shipped PDF/EPUB. | Replace that clause: `help` prints the summary, an unrecognised command prints one line naming `pickle help`, and both exit non-zero (`help` exits 0). Consider one sentence in `[#cmd-hooks]`'s fail-open bullet noting *why* it is terse (it rides in a hook's stderr). |
| R2 | non-blocking | fixed inline | The recorded acceptance transcript could not run: (a) step 3's `PATH="$D/empty"` also drops `git`, diverting `checkHooks` to its no-repo branch so the F9 case is never exercised (`doctor` printed `0 error(s), 0 warning(s)`); (b) step 3 greps `no pickle on PATH` but the shipped text is `no pickle **is** on PATH` (0 matches); (c) step 4's `PATH=` reassignment makes bash drop its command hash, so `git` itself is unresolvable (exit 127). The deviations section described (a) and (c) in prose but never amended the block, which still claimed verbatim re-runnability. | Measured all three; `grep -c` gave `0` for the ticket's pattern and `1` for `no pickle is on PATH`. | Done: transcript replaced with the version that ran, `$D/gitonly` and `$GIT_BIN` introduced with comments, and an amendment note added under it. |
| R3 | non-blocking | **new ticket T-071** | `probeCapable` accepts only exit 0, but exit `1` means the guard *ran and found a violation* — proof of capability. Since `os.MkdirTemp("")` honours `TMPDIR`, the "empty" probe dir can sit inside a pickle project; on a `feat/` branch with `tickets/` staged the probe then reads a working binary as inert — the inverse of the lie this ticket exists to kill. | Reproduced with the shipped build: `TMPDIR=$D/proj/tmpdir … doctor` → `… it is inert (pickle v0.2.2-74-g1c5cc03 at …)`; control run with default `TMPDIR` warns about nothing. | Accept exit 0 **or** 1 as capable; optionally pin the probe dir outside `TMPDIR`'s reach or set `GIT_CEILING_DIRECTORIES`. |
| R4 | non-blocking | noted | The two surfaces disagree about a *stale* shim: `doctor` returns after the staleness warning and never probes, while `hooks status` probes the whole `KindOwned` branch. Same state, different amount of information. Each matches its own task text (task 4 says "not-stale branch only", task 6 says "the `KindOwned` branch"), so this is a divergence the plan authored rather than a coding slip. | `internal/doctor/doctor.go` (early `return` on `st.Stale`) vs. `internal/cli/hooks.go` `runHooksStatus`. | Leave as-is or make `doctor` mention both; a stale shim whose PATH binary is also inert is arguably the case most worth saying twice, since `pickle upgrade` alone won't fix it. |
| R5 | non-blocking | **new ticket T-071** | `KindForeign` is never probed, yet the manual actively recommends chaining the guard from your own hook (`pickle hooks run pre-commit \|\| exit 1`) — a configuration with identical PATH-skew exposure and no signal from any surface. | `internal/doctor/doctor.go` / `runHooksStatus` handle `KindForeign` without probing; `cli-reference.adoc` `[#cmd-hooks]` recommends the chain. | Grep the foreign hook for `hooks run` before probing, or probe and word it conditionally — or close it with one honest sentence in the foreign-hook line. |
| R6 | non-blocking | noted | `Reach.Self` is written but never read outside tests: `Problem()` short-circuits on `OK`, and no production caller inspects `Self`. Harmless (it is the field that makes the short-circuit assertable, and useful for future diagnostics) but currently exported state with no production consumer. | `grep '\.Self' --include=*.go \| grep -v _test.go` → no hits. | Keep it (documented as diagnostic) or drop it and assert the short circuit by timing; not worth a change on its own. |
| R7 | non-blocking | fixed inline | A comment this branch added to `checkHooks` claimed the function "never execs" — false, since `hook.Probe()` execs up to two processes. Intended meaning: doctor never spawns one *directly*. | `internal/doctor/doctor.go`, comment above the `hook.Probe()` call. | Done: reworded to say this package never spawns a process directly and still only returns findings, while noting it now *causes* an exec through `hook.Probe`. |
| R8 | non-blocking | fixed inline | The whole T-068 entry sat under `### Added` in a file that declares Keep a Changelog, although most of it is changed behaviour (shim notice, terse unknown command, new warnings on three commands). | `CHANGELOG.md` `## [Unreleased]` had only an `### Added` subsection. | Done: the probe stays under `### Added`; a new `### Changed` subsection carries the shim v2 bump and the terse unknown-command output. |
| R9 | non-blocking | fixed inline | Task 9 said to add **no new xrefs** until T-067 lands anchor validation; three `<<hooks-version-skew>>` references to a brand-new anchor were added, and the deviation went unrecorded. They do resolve — verified by rendering, since nothing in the pipeline checks anchors — but unlabelled they inlined the full 12-word section title ("… reporting it as healthy (Version skew: an installed shim is not the same thing as a working guard)"). | `pdftotext` of `just docs-build` output: TOC entry `10.7.1`, no literal `[hooks-version-skew]` anywhere; three body renderings of the long title. | Done: all three now read `<<hooks-version-skew,version skew>>`, re-rendered and re-verified; the deviation is recorded here. |
| R10 | non-blocking | fixed inline | `TestProbeSelfShortCircuit`'s comment justified the test by claiming a real exec of the symlinked test binary "would fail" on unrecognised flags. A Go test binary handed positional arguments generally still exits 0, so that reasoning is unsound — what actually proves the short circuit is the `Self` assertion. The test is correct; only its stated rationale was wrong. | `internal/hook/probe_test.go`, comment above the test. | Done: comment rewritten to name `Self` as the load-bearing assertion and to say explicitly why `OK` alone would prove nothing. |
| R11 | non-blocking | **new ticket T-071** | The "incapable" subtest asserts a warning appears but never that `res.Errors` is empty, so nothing at unit level pins *warning, not error* — a future promotion to an error (making `pickle doctor` exit non-zero over a user's local `PATH`) would pass the suite. Only the acceptance transcript covers it, and transcripts don't run in CI. | `internal/doctor/hooks_test.go`, `TestCheckHooksProbesPATH/incapable pickle on PATH warns and is inert`. | One `len(res.Errors) != 0` assertion. |

**Disposition summary:** 11 findings — **1 blocking** (R1, → `5-rework/`, docs-only scope);
10 non-blocking: **5 fixed inline** (R2, R7, R8, R9, R10 — all committed as `a554c6c` or as this
ticket's own transcript amendment), **3 batched into one new ticket** (R3, R5, R11 → **T-071**,
`spawned-by: [T-068]`), **2 noted** (R4, R6).

### Rework scope (R1 only)

One finding, docs-only:

1. `docs/user-manual/cli-reference.adoc:638-639` — replace the "so does any unknown command"
   clause with what actually ships, and keep the `help`-exits-0 / unknown-exits-2 distinction
   accurate.
2. Re-run `just docs-check` (and ideally `just docs-build`, since anchors and this sentence both
   live in the manual's rendered output).

Nothing else in the rework: R2/R7–R10 are already fixed, R3/R5/R11 belong to T-071, R4 and R6 are
noted and closed.

### Rework pass (2026-08-06) — R1 fixed

On the same branch (`feat/T-068-probe-the-pickle-on-path`, commit `d8f62bf`). Scoped to exactly
the listed finding, per the rework procedure — nothing else touched.

**R1 — fixed.** `docs/user-manual/cli-reference.adoc`'s `pickle version`/`pickle help` section no
longer claims "so does any unknown command": it now states the actual three-way split — `help`
(exit 0) and a bare `pickle` (exit 2) both print the full usage summary; an *unrecognised* command
is terser (one line naming `pickle help`, exit 2) — and gives the reason (a hooks-aware shim must
not bury its own notice under an older binary's usage dump). Re-verified against the actual
binary, not just re-read: `pickle help` exit 0, bare `pickle` exit 2 (both full usage), `pickle
nosuchverb` exit 2 (one line). `just docs-check` and `just docs-build` both clean; `pdftotext` on
the rebuilt PDF confirms the corrected sentence replaced the old one in the shipped artifact
(the exact surface R1's evidence cited).

**Full re-verification, not scoped-only:** `just build && just test && just lint && just
docs-check` all green (12 packages; `internal/hook`/`internal/doctor`/`internal/cli` unaffected by
a docs-only change, re-run anyway).

**Updated disposition summary:** 11 findings — **1 blocking, fixed** (R1); 10 non-blocking:
5 fixed inline (R2, R7–R10), 3 batched into **T-071** (R3, R5, R11), 2 noted (R4, R6).

### Scoped re-review — 2026-08-06 (`d8f62bf`)

Scoped to R1, per the protocol: the rest of the feature was audited at `1c5cc03`/`a554c6c` and is
not re-audited here. Every claim was re-verified independently rather than read off the rework
record.

- **R1 is resolved, and the replacement prose is itself correct.** The fix asserts three facts
  about exit codes, so each was measured against the built binary rather than reasoned about:
  `pickle help` → exit **0**, 45 lines on **stdout**; bare `pickle` → exit **2**, 45 lines on
  **stderr**; `pickle nosuchverb` → exit **2**, exactly **one** line, naming `pickle help`. The
  claim that `help` and a bare `pickle` both print "the full usage summary" is exact, not
  approximate: `diff` reports the two outputs **byte-identical**. A fix that traded one
  inaccuracy for another would have been a fresh blocking finding; this one does not.
- **The false claim is gone everywhere, including the artifact R1's evidence cited.**
  `grep -rn "so does" docs/` → no hits, and the rebuilt PDF contains no `so does any unknown`.
  The corrected sentence is present in the rendered manual.
- **The rework added a new xref, and it resolves.** `<<cmd-hooks>>` renders as "pickle hooks";
  the rebuilt PDF contains no unresolved `[cmd-…]` or `[hooks-version-skew]` marker. Checked
  deliberately because R9 established that nothing in the docs pipeline validates anchors — a
  broken one would have shipped silently (T-067's ground).
- **Scope held.** `d8f62bf` touches one file, +6/-2 lines, all inside the `pickle version` /
  `pickle help` section: no drive-by edits, nothing from T-071's items, no code.
- **Re-verified green:** `just build`, `just test` (12 packages), `just lint`, `just docs-check`,
  plus `just docs-build` for the rendered-artifact checks above.

No new findings from the rework pass.

**Final disposition summary (both passes):** 11 findings — **1 blocking** (R1, **fixed** in the
rework pass and re-verified), 10 non-blocking: **5 fixed inline** (R2, R7, R8, R9, R10),
**3 batched into one new ticket** (R3, R5, R11 → **T-071**), **2 noted** (R4, R6).

### Impact sweep — unchanged by the rework (step 8)

The first pass's sweep stands: **T-046** was patched with the concrete new shape of
`checkHooks`'s `KindOwned` branch (three outcomes, new pass-line text) and the warning it will see
in this repo until the Homebrew `pickle` catches up; **T-069**'s "no collision" note remains
accurate (T-068 touched `internal/doctor`, not `internal/config`). The rework was one docs
sentence, so it invalidates nothing further — re-checked rather than assumed.

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
- 2026-08-06 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-06 — IN REVIEW → REWORK: review REWORK: 1 blocking (R1: manual still says an unknown command prints the usage summary, the behaviour task 7 replaced); 10 non-blocking (5 fixed inline, 3 batched into T-071, 2 noted)
- 2026-08-06 — REWORK → IN REVIEW: findings fixed: R1 resolved (manual now matches the terse unknown-command behaviour)
- 2026-08-06 — IN REVIEW → DONE: review PASS after one rework pass; 11 findings: 1 blocking (R1) fixed and re-verified, 5 fixed inline, 3 batched into T-071, 2 noted
- 2026-08-06 — user-approved publish: the three branch commits squashed to `741f06a` (tree
  byte-identical to the pre-squash state, verified) and pushed; **PR #16** opened against `main`
  — 1 commit, 15 files, zero `tickets/` paths. The overarching bookkeeping (these six
  `docs(tickets):` commits, `889129e..516c6d6`) was fast-forwarded to `origin/main` **first**,
  deliberately: the branch's base was itself an unpushed bookkeeping commit, so opening the PR
  beforehand would have carried `bcd82c3` and `aeb0bb4` plus ~573 lines of `tickets/` churn into
  it, and a squash-merge would have folded ticket bookkeeping into the code commit — rules §0's
  hazard reached through the *publish* step rather than the commit step. Merge is the human's
- 2026-08-06 — merged to main (PR #16, 4136f2d), user-approved; a merge commit, so 741f06a survives
  verbatim on `main` rather than being squashed into a new sha
