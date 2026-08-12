---
id: T-046
title: make doctor and upgrade self-host-aware (skill symlink detection, payload-version noise)
project: pickle
depends-on: []
spawned-by: [T-044]
impact: low-medium
complexity: low
cost: S
---

# T-046 — make doctor and upgrade self-host-aware (skill symlink detection, payload-version noise)

## Outcome

After this ships, `pickle doctor` in a self-hosting checkout reports `0 error(s), 0
warning(s)` instead of a permanent payload-version warning: it detects that
`.agents/skills/ticket-flow` is a symlink to the payload source, says so as a `doctor -v`
passed line, and never tells you to run the `pickle upgrade` this repo's policy forbids.
Every other finding — including the inert-guard warning about the `pickle` on `PATH` — still
fires, so the channel real diagnostics arrive on is empty by default again.

## Description

In a **self-hosting** repo (this one), `.agents/skills/ticket-flow` is a **symlink** to the
payload source (`skill/`), not an installed copy. `pickle upgrade` already respects that
symlink and leaves the skill directory alone — but `pickle doctor` and the `payload_version`
stamp in `pickle.toml` don't know about the arrangement, producing a **standing false
warning** (measured in this repo, 2026-08-12, binary built from `b6b583b`):

```
WARNING: payload version "0.5.0" differs from binary "v0.5.0-67-gb6b583b" — run `pickle upgrade`
pickle doctor: 0 error(s), 1 warning(s)
```

That is `doctor`'s *only* output on this repo today — every other check passes — so the
warning is the whole signal, and it is false.

The suggested remedy (`pickle upgrade`) is exactly what the repo's self-modify policy forbids
running from a feature branch (AGENTS.md, "Self-modify policy"): the binary is the artifact
under development, and the marker block / config are dev fixtures. So the warning is permanent
noise — and permanent warnings train people to ignore `doctor`. That is not hypothetical here:
on 2026-08-07 this exact accepted-noise line was the only thing `doctor` said while the
pre-commit guard sat inert (`tickets/NOTES.md`, *Field finding (2026-08-07)*), which is what
re-graded this ticket to `low-medium`.

Make the tooling **self-host-aware**:

- **doctor**: when the installed skill directory is a symlink (a dev/self-host link), the
  payload is the linked source by construction — the `payload_version`-vs-binary comparison is
  meaningless. Detect the symlink, skip the comparison, and report it as an informational
  passed line instead of a WARNING. Advice to "run `pickle upgrade`" must not be emitted in
  this mode, from *either* of `checkVersion`'s two warning branches (T-026).
- **upgrade**: it already skips replacing a symlinked skill dir. This ticket **decides and
  documents** what it does with the `payload_version` stamp in that mode: it keeps stamping
  (D4 below) — which is also what AGENTS.md's self-modify policy already describes as the one
  legitimate self-upgrade — and the decision is pinned by a test rather than left to be
  re-derived. `doctor` and `upgrade` then agree by construction: `doctor` no longer has an
  opinion about the stamp in this mode, so it can never send you to `upgrade` in vain.

The defect is *only* the version comparison. The two neighbouring drift checks were measured
silent in this repo and are out of scope: the `AGENTS.md` marker block is byte-identical to a
fresh render (T-041 pinned this), and the pi scaffolds match the shipped copies.

Soft couplings — all discharged or declined, none blocking:

- Born from the T-044 session's self-modify-guard discussion (see the AGENTS.md policy bullet
  added alongside this ticket, which this ticket must now amend — it currently promises the
  warning is accepted noise *"until it is made self-host-aware"*).
- **T-026** (upgrade refuses legal `pickle.toml`) and **T-068** (inert-guard PATH probe) both
  **landed**; the "don't run concurrently" couplings are spent. Their shapes are folded into
  the plan below (`checkVersion`'s two branches, `checkHooks`'s three).
- **T-071** owns the general PATH-skew half; **T-082** owns the rebase-folds-bookkeeping
  candidate that T-073's History floated for this ticket. Neither is touched here.
- **T-074** renames the installed skill directory to `brine`. It will rename `install.SkillDir`,
  which the helper added here sits next to and reads — so this ticket adds one more call site to
  that rename, not a conflict.
- **T-042** owns collapsing duplicated internal predicates. This ticket adds one exported
  predicate and routes only the two call sites where the predicate alone is what's wanted; the
  remaining `Lstat` sites (which need the existence result too) stay T-042's ground.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child-project is `pickle` at the repo root (`path = "."`), so the feature branch is
cut in this repository:

```
git checkout main
git checkout -b feat/T-046-self-host-aware-doctor
```

Root-path child: local WIP commits are encouraged, then **tidy them into atomic commits before
presenting** (rules §0) and default to keeping that history rather than squashing. No push and
no MR without explicit user approval.

**Self-modify policy (AGENTS.md) applies literally to this ticket.** Never run `pickle
install|upgrade|uninstall` against this repo from the branch. Any manual verification of
`upgrade` behaviour goes to a throwaway directory with the binary copied in:

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && ./pk install …
```

Running `./pickle doctor` **is** allowed — it is read-only.

### Prerequisite gate (hard)

None. `depends-on:` is empty; T-026 and T-068, the two tickets this one had to follow, are both
in `6-done/` and merged to `main`. Clean tree on `main` before branching.

### Confirmed design decisions (do not deviate without asking)

1. **The self-host signal is the symlink, and nothing else.** `.agents/skills/ticket-flow`
   being a symlink (`os.Lstat` + `os.ModeSymlink`) is the whole detection. **No new
   `pickle.toml` key** — a config flag would be a second source of truth for a fact the
   filesystem already states, and `install`/`upgrade`/`uninstall` already key their own
   self-host behaviour off exactly this symlink. Whether the link resolves is irrelevant to the
   predicate: a *broken* self-host link is still not an installed payload copy, and
   `checkSkill` already reports that separately as an error.
2. **One exported predicate, in `internal/install`.** `install.SkillLinked(root string) bool`
   lives next to the `SkillDir` constant that defines the path. `internal/doctor` must not
   re-derive the skill path or re-implement the `Lstat` test.
3. **doctor skips the comparison and says so.** In this mode `checkVersion` emits exactly one
   `Passed` line (`r.ok`, visible under `doctor -v`) and returns: no `Warnings` entry, no
   `Errors` entry, and the string `pickle upgrade` must not appear in the line.
4. **`upgrade` keeps stamping `payload_version` on a symlinked skill dir** — behaviour
   unchanged, now pinned by a test. Rationale: upgrade still refreshes everything else it owns
   (marker block, pi scaffolds, hook shim), so the stamp truthfully records "this tree was last
   upgraded by that binary"; AGENTS.md already names re-stamping as the one legitimate
   self-upgrade; and skipping it would freeze `pickle.toml` at an arbitrary old version forever.
   Since `doctor` no longer reads the stamp in this mode, the two cannot disagree.
5. **The skip sits ahead of *both* of `checkVersion`'s warning branches** (T-026 added a second:
   `run pickle upgrade` when the stamp would succeed, `edit payload_version by hand` when
   `config.PayloadVersionStampable` refuses). It is the **first** statement in the function,
   ahead of the `version == "" || version == "dev"` guard and the equal-version early return, so
   the passed line appears unconditionally in this mode — the check genuinely does not run, and a
   silent skip that only *sometimes* prints would be a third shape to reason about.
6. **Nothing else is suppressed.** `checkHooks`'s three `KindOwned` outcomes (stale warning,
   T-068's inert-`PATH` warning, ok), `checkMarkers`'s drift warning and `checkAgentScaffolds`'s
   drift warning all keep firing verbatim in self-host mode — they are about the user's `PATH`
   and about hand-edits, not about how the skill dir is arranged. A test pins the inert-guard
   warning specifically, because it is the one this ticket's own History flagged as at risk.
7. **Declined (noted, not filed):** replacing the skipped comparison with a *new* check that
   diffs the linked source tree against the binary's embedded payload ("is this binary stale
   relative to `skill/`?"). It is a different, larger feature — a fresh content-drift check with
   its own false-positive story (any dirty working tree differs) — and this ticket's job is to
   stop lying, not to invent a diagnostic.
8. **Scope fence.** No changes to `internal/hook`, `internal/vcs`, the skill payload under
   `skill/`, or the `AGENTS.md` marker block. Because the payload is untouched, no hand-mirrored
   marker-block edit is needed (AGENTS.md's *hand-written* self-modify bullet does change — see
   the Docs step — but that text sits outside the markers).

### Tasks

#### Task 1 — `install.SkillLinked`, the one predicate

In `internal/install/install.go`, next to the `SkillDir` constant block, add:

```go
// SkillLinked reports whether the installed skill directory is a symlink: the
// dev/self-host arrangement in which .agents/skills/ticket-flow points at the
// payload source (this repo's skill/) instead of holding a copy of it. install,
// upgrade and uninstall all already treat that link as "not ours to replace";
// doctor uses it to skip the payload_version comparison, which compares an
// installed copy that does not exist. A broken link still counts — it is still
// not a copy; checkSkill reports the breakage on its own.
func SkillLinked(root string) bool {
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(SkillDir)))
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}
```

Route `Upgrade`'s skill-refresh guard through it (the site that today reads
`if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink == 0`, ~`install.go:288`)
so the wipe-and-recopy branch reads `if !SkillLinked(root) { … }`. Leave `copyPayload`
(~`:528`) and `Uninstall` (~`:394`) alone: both need `Lstat`'s *existence* answer as well as the
mode, so rewriting them is predicate-collapse work that belongs to **T-042** (D2/scope fence).

#### Task 2 — doctor detects once and skips the version check

In `internal/doctor/doctor.go`:

- In `Check`, compute `selfHost := install.SkillLinked(root)` once, before the check calls, and
  pass it to `checkSkill` and `checkVersion`.
- `checkSkill(root string, selfHost bool, r *Result)`: on success, when `selfHost`, make the
  passed line name the arrangement, e.g.
  `skill payload present (.agents/skills/ticket-flow -> <target>, self-host link)`, resolving
  the target with `os.Readlink`. Unresolvable/broken cases keep today's error text unchanged.
- `checkVersion(cfg *config.Config, version string, selfHost bool, r *Result)`: first statement
  (D5) —

  ```go
  if selfHost {
  	r.ok(fmt.Sprintf("payload version check skipped (%s is a self-host link, so the payload is this tree, not an installed copy)", install.SkillDir))
  	return
  }
  ```

  Extend the function's doc comment with the reason, mirroring how the T-026 rationale is
  recorded there today.
- Update the `Check` doc comment: `version` is used for a payload-drift warning *unless the
  skill dir is a self-host link*.

#### Task 3 — doctor tests

In `internal/doctor/doctor_test.go`, add a fixture beside `installFixture`:

```go
// selfHostFixture is installFixture with the installed skill dir replaced by a
// symlink to a real payload tree — the self-hosting arrangement of pickle's own
// repo (.agents/skills/ticket-flow -> ../../skill).
func selfHostFixture(t *testing.T) string
```

It `os.RemoveAll`s the installed dir and symlinks it to an absolute path (`filepath.Abs` of
`payloadRoot()/skill`) so `checkSkill` still resolves `SKILL.md` and
`resources/tickets-README.md`, exactly as `TestUpgradeSelfHostSymlinkGuard` does in the install
package. Then add:

1. `TestCheckSelfHostLinkSkipsVersionCheck` — `selfHostFixture` (stamped `test-ver`) checked
   with `Check(root, "v9.9.9", …)`: **zero** errors, **zero** warnings, and a `Passed` entry
   containing `payload version check skipped`. Also assert no warning contains
   `payload version` and none contains `pickle upgrade`. This is the ticket's headline
   behaviour, and the mirror of `TestCheckVersionDriftWarns`.
2. `TestCheckSelfHostLinkSkipsUnstampableVersionCheck` — same fixture, but first wedge
   `payload_version` into the multi-line-string shape `TestCheckVersionDriftUnstampable…`
   uses. Still zero warnings — this is what proves the skip precedes **both** T-026 branches
   (D5), and it fails if the skip is inserted after the `PayloadVersionStampable` probe.
3. `TestCheckSelfHostLinkStillReportsIncapablePATHPickle` (in `internal/doctor/hooks_test.go`,
   beside `TestCheckHooksProbesPATH` and reusing its `PATH`-manipulation helpers) — a self-host
   fixture whose `PATH` `pickle` cannot run `hooks run pre-commit` still warns. Guards D6: the
   skip must not become a blanket self-host mute.
4. `TestCheckSelfHostLinkNamesTheLink` — the `doctor -v` passed line for the skill check names
   the link target (asserts on `Passed`, so the informational half of the Outcome is pinned).

#### Task 4 — pin upgrade's stamping contract (D4)

Extend the existing `TestUpgradeSelfHostSymlinkGuard` in `internal/install/install_test.go`
(~`:224`, which already upgrades `v1` → `v2` over a symlinked skill dir and asserts the link and
its target survive) with the assertion that the stamp still happened: `config.Load` the
resulting `pickle.toml` and require `PayloadVersion == "v2"`, with a comment naming D4 and the
reason (upgrade still refreshed the markers/scaffolds/shim, and doctor no longer reads the stamp
in this mode). Extending the existing test rather than adding a new one keeps the self-host
upgrade contract in one place.

#### Task 5 — docs

1. `docs/user-manual/cli-reference.adoc`, the `pickle doctor` bullet on `payload_version`
   (~`:284`): add that a **self-host symlinked** skill directory skips the comparison entirely —
   the payload *is* the linked source, so there is nothing to compare — and that the skip is
   reported as a passed line under `--verbose`, never a warning.
2. Same file, the `pickle upgrade` section (~`:206-212`, which already says a self-host
   symlinked skill dir is left untouched): add one sentence that `payload_version` **is** still
   stamped in that mode, since the marker block, scaffolds and hook shim were refreshed — the
   D4 contract, stated where a reader looks for it.
3. `AGENTS.md`, the hand-written *Self-modify policy* section: rewrite the final bullet
   (``pickle doctor``'s standing `payload version … differs` warning is accepted self-host noise
   *until it is made self-host-aware*) to record that it **is** now self-host-aware and the
   warning is gone. Outside the marker block; `pickle upgrade` never touches it.
4. `DESIGN.md`, the *Payload versioning* bullet (~`:277`): one clause that the comparison is
   skipped when the skill dir is a self-host link.
5. `CHANGELOG.md` `[Unreleased]` → `### Changed`: an entry for `pickle doctor` no longer warning
   about `payload_version` in a self-host checkout, naming T-046 and noting that `upgrade` keeps
   stamping.

### Acceptance test

Run from the repo root, on the feature branch:

```
just build
just test
just lint
just docs-check
```

Expected: all four clean. `just test` includes the four new/extended tests above; confirm they
run rather than skip:

```
go test ./internal/doctor/ ./internal/install/ -run 'SelfHost' -v
```

Expected: `TestCheckSelfHostLinkSkipsVersionCheck`,
`TestCheckSelfHostLinkSkipsUnstampableVersionCheck`,
`TestCheckSelfHostLinkStillReportsIncapablePATHPickle`, `TestCheckSelfHostLinkNamesTheLink`,
`TestUpgradeSelfHostSymlinkGuard` all `--- PASS`, none `SKIP`.

The end-to-end proof, on this self-hosting repo itself (read-only, so the self-modify policy
allows it):

```
./pickle doctor
./pickle doctor -v | grep -i 'payload version'
```

Expected: the first prints `pickle doctor: 0 error(s), 0 warning(s)` and exits `0` (before this
ticket: `0 error(s), 1 warning(s)`). The second prints exactly one line, an `ok:` line
containing `payload version check skipped`, and **no** `WARNING:` line and no `pickle upgrade`.

Regression guard that the skip is scoped, still on this repo:

```
./pickle doctor -v | grep -c 'pre-commit guard installed and current'
```

Expected: `1` — the hook check still runs and still reports the `PATH` probe's verdict.

And the negative case, proving an ordinary install is unaffected (throwaway dir, per the
self-modify policy — never `pickle install` in this repo):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q -b main . && ./pk install --project demo
sed -i.bak 's/^payload_version = .*/payload_version = "0.0.1"/' pickle.toml
./pk doctor ; echo "exit=$?"
```

Expected: the warning still appears — `payload version "0.0.1" differs from binary … — run
`pickle upgrade`` — because that tree holds a real installed skill copy. Exit `0` (warnings are
advisory).

Finally, `./pickle board audit` reports no errors.

### Docs update (mandatory when user-facing)

Task 5: `docs/user-manual/cli-reference.adoc` (the `doctor` `payload_version` bullet and the
`upgrade` self-host paragraph), `AGENTS.md` (hand-written self-modify bullet, outside the
markers), `DESIGN.md` (payload-versioning bullet), `CHANGELOG.md` (`[Unreleased]` → `Changed`).

**No skill-payload change and no marker-block change** — the flow's rules are untouched; only
pickle's report about its own install changes. `docs/user-manual/configuration.adoc` needs no
edit: its `payload_version` text is about *how* `upgrade` rewrites the line, which is unchanged.

### Finish (mandatory)

1. `just build`, `just test`, `just lint`, `just docs-check` all clean; the on-repo `doctor`
   transcript above captured (0 warnings) alongside the throwaway-dir negative case.
2. Docs updated per Task 5.
3. Write the summary: the new predicate and its two call sites, doctor's skip and the two passed
   lines, the four tests, the D4 decision now pinned, and anything deferred (D7's declined
   payload-vs-source drift check; the `Lstat` sites left to T-042).
4. Root-path child: interactive-rebase the WIP commits into atomic ones (suggested split: the
   `install` predicate + upgrade routing; the `doctor` change + tests; the docs) and default to
   **keeping** that history rather than squashing. Suggested commit message for the primary
   commit:

   ```
   fix(doctor): skip the payload-version check when the skill dir is a self-host link (T-046)
   ```

5. Commit locally; **no push and no MR without explicit user approval**. Before any push,
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
   print nothing.
6. `pickle ticket move T-046 in-review --reason "acceptance green"` (bookkeeping committed on
   `main`, never on the feature branch) and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: pickle ticket new
- 2026-08-01 — patched by T-026's review impact sweep: T-026 landed on the very code this ticket
  targets. `internal/doctor.checkVersion` now probes `config.PayloadVersionStampable` first and
  has **two** warning branches — ``run `pickle upgrade` `` when the stamp would succeed, and
  "edit `payload_version` by hand (line N)" when it would refuse. The Description's quoted
  warning text is therefore only one of two shapes, and the self-host skip must sit **ahead of
  both** branches, not just the one. No plan to invalidate (this ticket is unrefined); the
  `sequence, don't run concurrently` coupling is now discharged — T-026 landed first
- 2026-08-06 — patched by the T-068 filing (post-merge verification of T-057): **T-068** now owns
  the neighbouring `doctor` defect — the pre-commit guard reported as healthy while the `pickle` on
  `PATH` cannot run it. Deliberately not folded here: that one is version skew in *any* install,
  this ticket's ground is self-host noise. Both edit `internal/doctor/doctor.go`, so sequence them
  rather than running them concurrently, and whichever lands second re-verifies the other's
  message shapes
- 2026-08-06 — patched by the **T-068 review impact sweep**: T-068 has landed (reviewed, one
  blocking docs finding in rework), so this ticket is the one that lands second and inherits the
  re-verification. Concretely, `checkHooks`'s `KindOwned` branch is no longer two outcomes but
  three: stale → warning (unchanged), current-but-`PATH`-pickle-incapable → **new** warning
  (`… is installed and current, but … inert …`), current-and-capable → `ok`, whose text gained a
  trailing `, and the pickle on PATH can run it`. Two consequences for refining this ticket: (a)
  any self-host skip must not accidentally suppress the new inert warning, which is about the
  user's `PATH` rather than about self-hosting; (b) in *this* repo the new warning fires for real
  until the Homebrew `pickle` catches up (measured: 0.2.2 at `/opt/homebrew/bin/pickle`), so it
  joins the standing self-host `doctor` noise this ticket exists to triage — decide explicitly
  whether it is in scope. `internal/hook` also gained `probe.go`, which is the *only* new
  `os/exec` site and stays behind that package
- 2026-08-07 — **impact `low` → `low-medium`**, on an observed incident rather than argument. A
  `docs(tickets)` commit on `main` printed ``pickle: unknown command "hooks"`` followed by
  `bookkeeping guard skipped (hooks run exited 2)`: shim v2 (installed by the local build that
  stamped `payload_version = v0.2.2-54-g92154e5`) calls `pickle hooks run`, and the `PATH` binary
  is Homebrew 0.2.2, which predates the subcommand — so the guard is **inert on every commit in
  this clone**. `pickle doctor` then reported `0 error(s), 1 warning(s)`, and that one warning was
  the `payload version … differs` line `AGENTS.md` designates as *accepted self-host noise*. The
  noise this ticket exists to triage is therefore not cosmetic: it is the channel a real
  diagnostic arrives on, and it masked one. That moves the ticket off the `narrow/cosmetic` floor
  (rules §3) — but only one notch, because the blast radius is still this repo alone: an ordinary
  workspace has an installed skill copy and a matching stamp, so it never sees the noise.
  **Not folded in:** the general half of the incident — a diagnostic that only ships in the binary
  you do not have — is `PATH`-skew, not self-host, and is noted on T-071 instead
- 2026-08-12 — refined. Description re-verified against the current tree and updated: the quoted
  warning now shows the **measured** current text (`"0.5.0"` vs `"v0.5.0-67-gb6b583b"`), and
  `doctor` on this repo is measured at exactly `0 error(s), 1 warning(s)` — that one warning is
  this ticket's target, so the ticket's win is a zero-warning baseline, which the `## Outcome` now
  says. Three earlier History patches discharged: (a) T-068's inert-guard warning **no longer
  fires here** — the `PATH` `pickle` is now 0.5.0 and `doctor` reports `… and the pickle on PATH
  can run it` — so the 2026-08-06 "decide whether it is in scope" question resolves to *out of
  scope, but explicitly protected* (plan D6 + a test); (b) T-026's two-branch `checkVersion` is
  handled by D5 (the skip is the function's first statement, ahead of both); (c) T-073's
  "candidate scope for T-046" (a rebase folding bookkeeping into an MR) is **declined** — T-082
  owns it. Scope decided as one measured defect only: the marker-block and pi-scaffold drift
  checks were re-measured silent here and stay untouched. **Grade kept at `impact: low-medium`
  rather than collapsed** (rules §3 asks refinement to collapse a range): the range is not
  residual uncertainty but a genuine straddle of two measured facts — the 2026-08-07 masking
  incident puts it above the `low` cosmetic floor, while the blast radius (self-host/dev-link
  trees only) keeps it below a general `medium` quality win. Collapsing either way would discard
  one of them
- 2026-08-12 — TO DO → READY: plan complete
- 2026-08-12 — READY → IN DEVELOPMENT: picked up
