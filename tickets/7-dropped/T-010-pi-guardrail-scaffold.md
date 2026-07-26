---
id: T-010
title: Pi guardrail scaffold
project: pickle
depends-on: [T-004]
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-010 — Pi guardrail scaffold

## Description

> **ABSORBED by T-009 (2026-07-26, user decision).** The `.pi/` scaffold — including the
> T-006 symmetry obligation below — is now part of T-009's agent-enablement scope
> (`pickle install --agent claude,opencode,pi`). Dropped, not lost: T-009's plan carries
> every requirement stated here.

> **PARKED (triage 2026-07-26).** Real, but explicitly unscheduled: nothing is blocked on
> this and no user has asked for it. Do not pick it up without a demand signal. Unparking is a
> user decision — note it in History.

With `--agent pi`, scaffold `.pi/` with a guardrails extension adapted from `pickle.toml`: a
git-staging deny-list, a publish gate for child-projects (push / MR require confirmation), and a
self-install guard, plus a short `.pi/README`. Mirrors the reference guardrails pattern. This is
the one genuinely agent-specific part of the payload. Needs T-004. Phase P4.

**Symmetry obligation (added by the T-006 review).** T-006 shipped `upgrade`/`uninstall` with a
**closed, hardcoded artifact list** — `install.Upgrade` (`internal/install/install.go:124-148`)
and `install.Uninstall` (`:176-212`) each enumerate the same four artifacts literally
(`SkillDir`, `ClaudeSkillLink`, `AGENTS.md`, `CLAUDE.md`); there is no agent registry and nothing
enforces symmetry. So a scaffolded `.pi/` would today be silently left behind by `uninstall` and
never refreshed by `upgrade`. Whatever this ticket lays down must therefore also be refreshed by
`Upgrade`, removed by `Uninstall`, and checked by `doctor` — and should follow T-006's decision-D6
pattern of *probing the filesystem* ("refresh/remove only what is already present"), because
neither command records which agents were installed (`install.Options`' `Claude`/`ClaudeLink`
booleans are never persisted to `pickle.toml`). T-006's own rework scope is closed, so this
burden sits here, not there.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P4)
- 2026-07-26 — parked (stays in TO DO, unscheduled). source: board triage — backlog growth analysis
- 2026-07-26 — TO DO → DROPPED: absorbed into T-009 (agent enablement owns the pi scaffold + symmetry obligation)
