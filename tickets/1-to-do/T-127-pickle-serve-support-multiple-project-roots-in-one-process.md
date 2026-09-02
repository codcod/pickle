---
id: T-127
title: pickle serve: support multiple project roots in one process
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: high
cost: L
---

# T-127 — pickle serve: support multiple project roots in one process

## Outcome

After this ships, someone who runs brine across several unrelated projects starts one
`pickle serve` process — from any directory, naming each project root once — and gets one
dashboard with a switcher between them, instead of one `pickle serve` per project root each
bound to its own port.

## Description

`pickle serve` (`internal/cli/serve.go`, `internal/serve/serve.go`) currently resolves exactly
one `pickle.toml` per process: `runServe` calls `loadConfig()`, which walks up from the current
working directory (`config.Find`) to the nearest `pickle.toml`, and passes a single
`serve.Options{Root, Cfg}` into `serve.Serve`. `Options.Root` is "the overarching project root"
— note this is **not** the same axis T-061/T-104 already cover (a `pickle.toml` with several
registered `[[project]]` children, filterable on one board): those children still share one
ticket tree and one config file. This ticket is about **separate overarching projects** —
each with its own `pickle.toml`, its own `tickets/` tree, on its own filesystem path, with no
relation to each other — the case the ticket's Description names: someone who has brine
installed in project A and project B today runs two independent `pickle serve` invocations,
cwd'd into each, each claiming its own port, because the binary has no notion of "more than one
root" anywhere in `internal/config` or `internal/serve`.

The request is to let one process serve N such roots, so a person (or a machine running a
dashboard for a team) does not have to track a per-project port map. This is a real
architecture change, not a flag tweak, and should stay TO DO until refinement has answered at
least:

- **Discovery.** How are the N roots named on the command line / in config — repeated
  `--dir`/`--project` flags, a small workspace list file, or scanning a parent directory for
  `pickle.toml`s? Each has different ergonomics and failure modes (a root that stops existing,
  a root added after the process started).
- **Routing.** `Handler` currently mounts one mux at `/`. Serving several roots needs each
  project's routes disambiguated (e.g. a `/p/{slug}/...` prefix) without breaking the existing
  single-project URLs anyone has bookmarked or scripted against — or an explicit decision to
  break them, stated up front.
- **Isolation.** `handler.load` takes `lock.WithShared(h.opts.Root, ...)` per request against
  one root; multiple roots means multiple independent locks, and one project's malformed
  ticket tree (the load path already tolerates this per-project) must not affect another
  project's page.
  `Options` and `handler` become a map/slice keyed by project rather than a single `Root`/`Cfg`
  pair, and every place that currently reads `h.opts.Root`/`h.opts.Cfg` directly (`load`,
  `newPage`, `staleBoardBranch`, `buildHealth`) needs to resolve "which project" first.
- **Security posture.** The existing loopback warning (`isLoopback` in `internal/cli/serve.go`)
  is scoped to "anyone who can reach this port can read every ticket in *the* project"; with N
  roots behind one port that warning's blast radius grows to every registered project, which the
  warning text and any future auth story should say plainly.
- **UI.** A project switcher in the shared page chrome (`internal/serve/templates`), reusing
  `projectName(root)` per project rather than the single label `newPage` sets today.

Soft coupling: T-061 and T-104 already solved multi-*child* filtering within one project's
board; this ticket's switcher is one level up (multi-*project*) and should look/feel
consistent with that prior art rather than inventing a second pattern, but does not depend on
either being reworked.

Refinement should also weigh a smaller alternative if the full switcher proves too costly for
the payoff: keeping one process per port but adding a lightweight "pickle serve --list" or
similar to at least surface the running instances, and explicitly recording why the full
multi-root dashboard was or wasn't chosen.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: chat: user asked to let one `pickle serve` process
  serve multiple project roots instead of one process per port
