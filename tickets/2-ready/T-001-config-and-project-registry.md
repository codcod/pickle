---
id: T-001
title: pickle.toml config model + project registry
project: pickle
depends-on: []
impact: high
complexity: medium
cost: M
---

# T-001 — pickle.toml config model + project registry

## Description

Define and implement the `pickle.toml` schema and a Go package to load, validate, and save it,
plus the `pickle project add|list|remove` commands on top.

Schema (see the hand-written bootstrap `pickle.toml` for the current shape): an overarching
block (`payload_version`, optional overarching `review_addendum`, a `[commit]` policy) and a
`[[project]]` array of registered child-projects — each with `name`, `path`, build/validate
commands (`build`/`test`/`lint`/`docs`), `branch_prefix`, per-child WIP limits
(`wip_in_development`/`wip_in_review`), and an optional per-child `review_addendum`.

`project add <name> <path>` appends a validated `[[project]]` block (unique name, resolvable
path, sensible defaults); `project list` prints the registry; `project remove <name>` refuses
while any live ticket targets that child.

Foundation for the rest of the CLI: **board audit** (T-002) validates each ticket's `project:`
against this registry; **ticket new** (T-003) validates `--project`; **install** (T-004) writes
the file and registers the first child; **doctor** (T-005) resolves child paths. Phase P1/P2
foundation. Soft-coupled to T-002, T-003, T-004.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # the child-project 'pickle' is the repo root
git checkout main
git checkout -b feat/T-001-config-and-project-registry
```

Local WIP commits are fine; **no push / no MR without user approval** (commit policy).

### Prerequisite gate (hard)

None — this is the foundational ticket. Clean tree on `main`.

### Confirmed design decisions (do not deviate without asking)

1. **TOML decoding via `github.com/BurntSushi/toml`** (stable, de-facto standard) — a
   build-time dependency compiled into the static binary; runtime stays dependency-free.
2. **Writing via a canonical hand-written renderer** (`config.Render`), not the TOML encoder —
   deterministic layout + our own header comments. `pickle.toml` is **tool-managed** after this
   ticket; `add`/`remove` mutate the model then re-render (the hand-written bootstrap comments
   are normalised to the canonical header on first mutation — acceptable).
3. **Schema** (matches the bootstrap `pickle.toml`): overarching `payload_version`, optional
   `review_addendum`, `[commit]` (`overarching_auto`, `child_publish_gated`); plus a
   `[[project]]` array — each `name`, `path`, `build`, `test`, `lint`, optional `docs`,
   `branch_prefix` (default `feat/`), `wip_in_development` (default 1), `wip_in_review`
   (default 1), optional `review_addendum`.
4. **Project root discovery** = nearest ancestor directory of the cwd containing `pickle.toml`
   (`config.Find`).
5. **`project remove` refuses** while any **live** ticket (in a non-terminal status dir,
   `1-to-do/`…`5-rework/`) has `project: <name>`.
6. **Validation:** names unique + non-empty; `path` non-empty and resolving (relative to root)
   to an existing directory; WIP limits >= 0; at least the fields above present.

### Tasks

#### Task 1 — `internal/config` package
New package `internal/config` with: the `Config`/`CommitPolicy`/`Project` structs (with
`toml:` tags); `Find(startDir) (path string, err error)`; `Load(path) (*Config, error)`
(decode + `Validate`); `Validate() error`; `Project(name) (*Project, bool)`; `Render() string`
(canonical writer); `Save(path) error`; `AddProject(p Project) error` and
`RemoveProject(name) error` (model mutations). Table-driven unit tests in
`internal/config/config_test.go` covering load/validate (good + each failure mode), round-trip
(`Render`→`Load` is stable), `Find`, and add/remove.

#### Task 2 — wire `pickle project add|list|remove`
Replace the stubs in `internal/cli/project.go`:
- `add <name> <path> [--build .. --test .. --lint .. --docs .. --branch-prefix .. --wip-dev N --wip-review N]`
  — `Find`+`Load`, reject duplicate name / missing dir, append with defaults, `Save`, confirm.
- `list` — `Find`+`Load`, print an aligned table (name, path, build/test/lint, WIP).
- `remove <name>` — `Find`+`Load`, error if unknown, refuse if a live ticket targets it
  (scan `tickets/{1-to-do..5-rework}/T-*.md` frontmatter `project:`), else remove + `Save`.

#### Task 3 — parse the real bootstrap config
Ensure `Load` parses this repo's own `pickle.toml`; add a test that loads it and asserts the
sole child `pickle` at path `.`.

#### Task 4 — docs + dependency note
Update `README.md`: note the `github.com/BurntSushi/toml` build dependency (runtime still
dependency-free) and a short `pickle.toml` schema summary. Run `go mod tidy`.

### Acceptance test

```
cd /Users/codcod/Projects/private/pickle
just lint && just test && just build          # gofmt/vet clean, tests pass, binary builds

# operate on a throwaway copy so the repo's own pickle.toml is untouched
rm -rf /tmp/pickle-t001 && mkdir -p /tmp/pickle-t001/web && cp pickle.toml /tmp/pickle-t001/
( cd /tmp/pickle-t001 && \
  /Users/codcod/Projects/private/pickle/pickle project list && \
  /Users/codcod/Projects/private/pickle/pickle project add web ./web && \
  /Users/codcod/Projects/private/pickle/pickle project list && \
  /Users/codcod/Projects/private/pickle/pickle project remove web && \
  /Users/codcod/Projects/private/pickle/pickle project list )
```
Expected: `list` shows `pickle`; `add web` succeeds and `list` then shows `pickle` + `web`;
`remove web` succeeds and `list` shows only `pickle` again; every command exits 0; the
throwaway `pickle.toml` stays valid TOML.

### Docs update (mandatory when user-facing)

`README.md` — the dependency note + `pickle.toml` schema summary (above). No `docs/` book yet.

### Finish (mandatory)

1. Acceptance test green; `just lint`/`test`/`build` clean.
2. README updated; `go mod tidy` run (go.mod/go.sum committed).
3. Summary of files touched + decisions.
4. Suggested commit: `feat(config): add pickle.toml model + project registry (T-001)`.
5. Commit locally on the branch; **do not push / open MR without approval**.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P1/P2)
- 2026-07-23 — TO DO → READY: implementation plan complete (READY gate met)
