---
id: T-010
title: Pi guardrail scaffold
project: pickle
depends-on: [T-004]
impact: medium
complexity: medium
cost: M
---

# T-010 — Pi guardrail scaffold

## Description

With `--agent pi`, scaffold `.pi/` with a guardrails extension adapted from `pickle.toml`: a
git-staging deny-list, a publish gate for child-projects (push / MR require confirmation), and a
self-install guard, plus a short `.pi/README`. Mirrors the reference guardrails pattern. This is
the one genuinely agent-specific part of the payload. Needs T-004. Phase P4.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P4)
