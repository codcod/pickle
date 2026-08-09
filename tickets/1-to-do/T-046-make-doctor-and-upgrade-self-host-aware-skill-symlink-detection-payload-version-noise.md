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

After this ships, this self-hosting repo's own `pickle doctor` stops printing a permanent, unactionable payload-version warning, because doctor and the version stamp both know `.agents/skills/ticket-flow` is a symlink to the payload source here, not an installed copy.

## Description

In a **self-hosting** repo (this one), `.agents/skills/ticket-flow` is a **symlink** to the
payload source (`skill/`), not an installed copy. `pickle upgrade` already respects that
symlink and leaves the skill directory alone — but `pickle doctor` and the `payload_version`
stamp in `pickle.toml` don't know about the arrangement, producing a **standing false
warning**:

```
WARNING: payload version "0.0.0-skeleton" differs from binary "91d5be5-dirty" — run `pickle upgrade`
```

The suggested remedy (`pickle upgrade`) is exactly what the repo's self-modify policy forbids
running from a feature branch (AGENTS.md, "Self-modify policy"): the binary is the artifact
under development, and the marker block / config are dev fixtures. So the warning is permanent
noise — and permanent warnings train people to ignore `doctor`.

Make the tooling **self-host-aware**:

- **doctor**: when the installed skill directory is a symlink (a dev/self-host link), the
  payload is the linked source by construction — the `payload_version`-vs-binary comparison is
  meaningless. Detect the symlink and either skip the check or report an informational
  "self-host link detected; payload version check skipped" line instead of a WARNING. Advice
  to "run pickle upgrade" must not be emitted in this mode.
- **upgrade**: it already skips replacing a symlinked skill dir; decide and document what it
  should do with the `payload_version` stamp in that mode (likely: still stamp it, since the
  marker block is still refreshed — or skip the stamp too and say so). Whatever the choice,
  `doctor` and `upgrade` must agree so one never tells you to run the other in vain.

Soft couplings: born from the T-044 session's self-modify-guard discussion (see AGENTS.md
policy bullet added alongside this ticket). Touches the same `doctor`/`upgrade` surfaces as
T-026 (upgrade refuses legal pickle.toml) — sequence, don't run concurrently.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

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
