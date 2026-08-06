---
id: T-068
title: the pre-commit guard can be silently inert: nothing checks the pickle on PATH that the shim actually runs
project: pickle
depends-on: []
spawned-by: [T-057]
impact: medium
complexity: low
cost: S
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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-06 — created (TO DO). source: pickle ticket new
- 2026-08-06 — filed from the T-057 post-merge verification, not from its review: arming the guard
  in this repo and smoke-testing it showed `hooks status` and `doctor` both reporting a healthy
  guard while the `pickle` on `PATH` (0.2.2, pre-`hooks`) made it a no-op. Batches T-057 review
  finding F9 (GUI/IDE minimal `PATH`) as the same theme. Graded medium/low/S: it voids a
  just-shipped guarantee, and the fix is a probe plus two message changes
