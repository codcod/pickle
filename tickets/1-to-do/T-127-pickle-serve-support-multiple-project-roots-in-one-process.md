---
id: T-127
title: pickle serve: support multiple project roots in one process
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-127 — pickle serve: support multiple project roots in one process

## Outcome

After this ships, someone who runs brine across several unrelated projects starts one
`pickle serve` process from any directory — `pickle serve --dir <a> --dir <b>` — and reaches
every board from one port through a switcher, instead of running one `pickle serve` per project
root and tracking which port each one took.

## Description

`pickle serve` (`internal/cli/serve.go`, `internal/serve/serve.go`) currently resolves exactly
one `pickle.toml` per process: `runServe` calls `loadConfig()`, which walks up from the current
working directory (`config.Find`) to the nearest `pickle.toml`, and passes a single
`serve.Options{Root, Cfg}` into `serve.Serve`. So a person running brine in several projects
runs one `pickle serve` per project, each cwd'd into its own root and each claiming its own
port.

This is **not** the axis T-061/T-104 already cover (one `pickle.toml` with several registered
`[[project]]` children, filterable on one board — those children share one ticket tree and one
config file). This ticket is about **separate overarching projects**: each with its own
`pickle.toml`, its own `tickets/` tree, unrelated to the others.

### Evidence this is field-driven, not hypothetical

Measured on the requesting user's machine (2026-09-02): six real brine workspaces, of which
**three are live boards** — 127, 20 and 18 tickets — plus one umbrella with four registered
children and no tickets yet. **All three live boards are `layout = "in-tree"`**, which is why
`pickle project add` does not already solve this: an in-tree board lives inside its own git
repo, so folding the three into one umbrella would mean moving each `tickets/` tree out of the
repo it belongs to and abandoning in-tree.

### Confirmed scope (user decision, 2026-09-02)

Repeatable `--dir` only, and **no aggregation**. Each root gets its own URL namespace and
renders the existing single-board page unchanged; a switcher links between namespaces. Nothing
is ever merged into a combined view.

This is what keeps the ticket small, and it is smaller than first estimated because T-053's
design is already parameterized: only **13 reads of `h.opts.*`** exist, all in handler methods
in `internal/serve/serve.go`, and every worker function already takes root/cfg as explicit
arguments (`staleBoardBranch(root, cfg)`, `buildHealth(def, root, tickets, cfg)`,
`buildBoard(def, tickets, cfg)`, `projectName(root)`). `view.go` reads no handler state at all,
so the rendering layer needs no refactor — the work is resolving a per-request
`rootState{root, cfg}` from the slug and threading it through those 13 sites.

Work implied:

- **Flags.** `parseServeArgs` accepts repeatable `--dir` (optionally `--dir name=path` to pin
  the slug explicitly — see the collision note below). It is already the seam tested without
  binding a port. Zero `--dir` flags keeps today's behaviour: resolve one root from cwd.
- **Routing.** Project-qualified routes (`/p/{slug}/...`), with `/` an index listing the served
  boards. This is **forced**, not chosen: all three live boards use `ticket_prefix = "T"` and
  all start at T-001, so T-001–T-018 exist in all three at once and today's `GET /t/{id}`
  cannot resolve. Deriving a slug from the directory name can itself collide (two checkouts
  both named `pickle`), which is what the optional `--dir name=path` form is for.
- **Per-root chrome.** The startup line and the T-108 stale-board banner become one per root.
  Both matter here — 3/3 live boards are in-tree, so each can independently sit on a feature
  branch. Mechanical, since `staleBoardBranch` already takes its root.
- **Locks.** Free: `handler.load` already calls `lock.WithShared(root, …)`, so N roots means N
  independent locks with no new machinery, and one project's malformed ticket tree already
  degrades only its own page.
- **Security wording.** `isLoopback`'s warning says "every ticket in the project"; with N roots
  behind one port it must say every ticket in **every served project**.

Because no view aggregates, ticket-id collision needs no dedup logic and **T-104's search stays
unchanged** — it searches within whichever board is being rendered.

### Alternatives rejected (recorded so they are not re-proposed)

The decision axis was *where the list of roots lives*, not flag syntax. Repeatable `--dir` puts
it in argv, so persistence is a shell alias or justfile recipe — the user's own config, not a
format pickle must own, validate, audit and version.

- **A `pickle.workspace.toml` found by walking up from cwd** — a second config format with its
  own loader, validation and audit story. Not worth it for a list of paths.
- **`[[peer]]` entries in an existing `pickle.toml`** — asymmetric: pickle's own committed
  config would name unrelated repos, leaking personal workspace layout into a public repo, and
  there is no principled "primary" among peers.
- **`--scan <parent> [--depth N]`** — genuinely ergonomic (the user's boards are siblings under
  one directory) but must exclude false positives (a `find` during triage turned up two stale
  go-module-cache copies of a workspace), and the served set changes silently as the filesystem
  does. Revisit only if maintaining the `--dir` list becomes a real burden.
- **A `~/.config` registry or a `PICKLE_ROOTS` env var** — both would be firsts for this binary:
  it makes zero `UserHomeDir`/`UserConfigDir`/`os.Getenv`/XDG calls anywhere in `internal/` or
  `cmd/`, and `skill/SKILL.md` ships the promise "Install scope is **per-project** — nothing is
  written to `~/`". A global root registry would contradict that sentence.

Soft coupling: T-061/T-104 solved multi-*child* filtering within one board; this switcher is one
level up (multi-*project*) and should match that prior art rather than invent a second pattern.
Neither needs reworking for this.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: chat: user asked to let one `pickle serve` process
  serve multiple project roots instead of one process per port
