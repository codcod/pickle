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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-127-serve-multi-dir
```

(`pickle`'s own repo, `path = "."` — root-path child, tidy WIP commits into atomic ones before
presenting, per rules §0.)

### Prerequisite gate (hard)

None. No `depends-on:`, clean tree, nothing else in flight against `internal/serve` or
`internal/cli/serve.go`.

### Confirmed design decisions (do not deviate without asking)

1. **`BasePath` is threaded explicitly through the `page` struct and every internal template
   link — no `<base href>` tag.** Explicit and Go-testable with plain string assertions;
   avoids depending on browser/htmx base-URL resolution semantics.
2. **Zero `--dir` flags is byte-for-byte today's behavior.** No prefix, no index page, no
   switcher, no wording change — the existing single-root code path (`serve.Handler`,
   `serve.Serve`) is untouched and its tests keep passing unmodified.
3. **Any `--dir` flag (even exactly one) switches to "named roots" mode.** Every named root,
   including a lone one, is mounted at `/p/{slug}/`; `/` becomes an index. This is an accepted,
   opt-in URL change scoped to use of the new flag — never a regression for someone who never
   passes it.
4. **Each `--dir <path>` resolves via `config.Find(path)`**, identically to how `loadConfig()`
   resolves cwd today — `--dir` may point at a subdirectory of a project, matching existing
   semantics rather than inventing new ones.
5. **Slug = `filepath.Base(resolvedRoot)` by default; `--dir name=path` pins it explicitly.**
   Two roots landing on the same slug (default or explicit) is a startup error (`exitUsage`),
   never a silent overwrite — same validate-before-bind style as today's `--addr`.
6. **Static assets are served once, unprefixed, at the top-level mux.** Same embedded FS
   regardless of root; prefixing would duplicate identical bytes for no behavioral gain.
7. **The index page and the header switcher are two different pieces of chrome.** The index
   (`GET /` in named-roots mode) lists every served project with a one-line health summary.
   The switcher is a nav item every per-project page gains in named-roots mode, populated from
   `Options.Peers`, so jumping boards never requires returning to `/`. Classic mode (`Peers ==
   nil`) renders neither — zero chrome change for single-root users.
8. **The loopback-warning wording switches to plural ("every ticket in every served
   project") whenever any `--dir` is passed**, tied to the same mode switch as decision 3 —
   never to a `len(dirs) > 1` count, so the wording and the routing mode can never disagree.

### Tasks

#### Task 1 — extend `parseServeArgs` for repeatable `--dir`

`internal/cli/serve.go`. Add a `dirArg{Name, Path string}` type. Change
`parseServeArgs(args []string) (string, int)` to
`parseServeArgs(args []string) (addr string, dirs []dirArg, code int)`. Accept `--dir`/`--dir=`
repeated any number of times; each value splits on the first `=` — `name=path` sets `Name`,
plain `path` leaves `Name` empty (resolved to a default slug in Task 3). Empty value after `=`
or a missing value after a bare `--dir` is `pickle serve: --dir needs a value` +
`serveUsage`, `exitUsage` — same style as the existing `--addr` checks right above it. Update
`serveUsage` to `"usage: pickle serve [--addr|-a host:port] [--dir [name=]path ...]"`.

#### Task 2 — extract `loadTickets` for reuse

`internal/serve/serve.go`. Pull the body of `handler.load` (the `flow.ForName` +
`lock.WithShared` + `ticket.LoadAll` sequence) into a package-level
`func loadTickets(opts Options) []*ticket.Ticket`. `handler.load` becomes a one-line wrapper
(`func (h *handler) load() []*ticket.Ticket { return loadTickets(h.opts) }`) so its doc comment
and every existing caller are unaffected. This is what lets the index page (Task 5) reuse the
same locked, per-project read without duplicating it.

#### Task 3 — `Options.BasePath` / `Options.Peers`, and `MultiHandler`

`internal/serve/serve.go`. Add two fields to `Options`: `BasePath string` (empty = classic
mode) and `Peers []PeerLink` (nil = classic mode, no switcher), plus an exported
`type PeerLink struct { Slug, Name string }`. `newPage` copies both onto `page` (Task 4).

Add `type NamedRoot struct { Slug string; Options Options }` and
`func MultiHandler(roots []NamedRoot) (http.Handler, error)`:

- Reject duplicate slugs up front (`fmt.Errorf("serve: duplicate project name %q", slug)`) —
  the CLI (Task 6) turns this into the user-facing disambiguation message.
- One top-level `http.NewServeMux()`. Register `/static/` once (the same `fs.Sub(assets,
  "static")` + `http.FileServerFS` lines already in `Handler`, factored into a small unexported
  `mountStatic(mux *http.ServeMux) error` both `Handler` and `MultiHandler` call, so the two
  registrations can never drift). Register `GET /healthz` once, top-level, independent of any
  project (per-project `/p/{slug}/healthz` still exists too, via each sub-handler — harmless).
- For each root: set `opts := root.Options`; `opts.BasePath = "/p/" + root.Slug`; `opts.Peers`
  = every *other* root's `PeerLink{Slug, Name: projectName(root.Options.Root)}`; build
  `h, err := Handler(opts)`; mount at `mux.Handle("/p/"+root.Slug+"/", http.StripPrefix("/p/"+
  root.Slug, h))`.
- Register `GET /` to render the index (Task 5) over the full `roots` slice.

Factor the shared shutdown/goroutine body already in `Serve` into an unexported
`func serveHTTP(ctx context.Context, ln net.Listener, h http.Handler) error`. `Serve` becomes
`{h, err := Handler(opts); ...; return serveHTTP(ctx, ln, h)}` (behavior identical, so
`TestServeOnRealListener` needs no change). Add
`func ServeMulti(ctx context.Context, ln net.Listener, roots []NamedRoot) error` following the
same shape with `MultiHandler`.

#### Task 4 — thread `BasePath`/`Peers` through `page` and every template link

`internal/serve/view.go`: add `BasePath string`, `Roots []PeerLink`, `Index *IndexView` to
`page`; `newPage` sets `BasePath: h.opts.BasePath, Roots: h.opts.Peers`.

`internal/serve/templates/*.html`: prefix every internal absolute link with `{{.BasePath}}`
(classic mode's `BasePath == ""` makes this a no-op, so these edits change no rendered output
for an existing single-root caller):

- `layout.html`: `href="{{.BasePath}}/"` (brand + nav), `href="{{.BasePath}}/activity"` (nav),
  the `idlist` block's `href="{{.BasePath}}/t/{{$id}}"`. Wrap the Board/Activity nav in
  `{{if not .Index}}...{{end}}` (they don't apply to the index page). Add, inside `head`, a
  switcher: `{{if .Roots}}<nav class="switcher">{{range .Roots}}<a
  href="/p/{{.Slug}}/">{{.Name}}</a>{{end}}</nav>{{end}}` — an absolute `/p/{slug}/` per peer,
  not `{{.BasePath}}`-relative, since it must reach a *different* project's namespace.
  `href="/static/..."` and the `htmx.min.js` script tag stay absolute (decision 6).
- `board.html`: prefix both `hx-get="/fragments/board"` occurrences, both `href="/t/{{.ID}}"`
  occurrences, `href="/t/{{.Family}}"`.
- `activity.html`: prefix both `hx-get="/fragments/activity"` occurrences,
  `href="/t/{{.ID}}"`.
- `ticket.html`: prefix the crumbs `href="/"` and the family `href="/t/{{.Family}}"`.

#### Task 5 — index page

`internal/serve/view.go`: `type IndexRoot struct { Slug, Name string; Health HealthView }`,
`type IndexView struct { Roots []IndexRoot }`,
`func buildIndex(roots []NamedRoot) IndexView` — for each root, `loadTickets(root.Options)`,
`buildHealth(flow.ForName(root.Options.Cfg.FlowName()), root.Options.Root, tickets,
root.Options.Cfg)`, `projectName(root.Options.Root)`.

`internal/serve/templates/index.html` (new file): `{{define "index"}}{{template "head" .}}<h1>
Boards</h1><ul class="index-list">{{range .Index.Roots}}<li><a
href="/p/{{.Slug}}/">{{.Name}}</a> — {{.Health.Tickets}} tickets, {{if
.Health.OK}}clean{{else}}{{len .Health.Errors}} error(s){{end}}</li>{{end}}</ul>{{template
"foot"}}{{end}}`. `MultiHandler`'s `GET /` handler assembles
`page{Title: "Boards", Project: "pickle", Index: &idx}` (no `Health`/`StaleBoard` — they are
per-project and don't apply here) and renders `"index"`.

#### Task 6 — `runServe`: branch on `len(dirs)`, resolve roots, wire the CLI

`internal/cli/serve.go`. Update the `addr, code := parseServeArgs(args)` call site to
`addr, dirs, code := parseServeArgs(args)`.

- `len(dirs) == 0`: exactly today's body, unchanged (`loadConfig()` from cwd, single
  `serve.Options{Root: cfg.Root(), Cfg: cfg}`, `serve.Serve`). The `isLoopback` warning keeps
  its current singular wording.
- `len(dirs) > 0`: for each `dirArg`, `filepath.Abs(d.Path)` → `config.Find` → `config.Load`;
  a failure at any step is `pickle serve: --dir %s: %v` via `errf` (mirrors `loadConfig`'s own
  error shape). `root := filepath.Dir(cfgPath)`; `slug := d.Name`, defaulting to
  `filepath.Base(root)` when empty. Collect into `[]serve.NamedRoot`; a duplicate slug is
  `pickle serve: duplicate project name %q (from %s and %s) — use --dir name=path to
  disambiguate` (`exitUsage`) before any listener is opened. Bind the listener exactly once
  (same `net.Listen`/`isLoopback` call as today), print one summary line ("pickle serve: N
  boards at http://<addr> (Ctrl-C to stop)") followed by one line per root naming its `/p/
  {slug}/` URL and resolved layout, and call `serve.ServeMulti(ctx, ln, roots)`. The
  `isLoopback` warning text switches to "every ticket in every served project" (decision 8).

### Acceptance test

`just test` green, plus:

- `internal/cli/cli_test.go`: extend `TestParseServeArgs` for the new 3-return signature and
  add cases — `--dir path` (default slug), `--dir name=path` (explicit slug), repeated `--dir`,
  a bare `--dir` with no value (`exitUsage`), an `--dir=` empty value (`exitUsage`). Add
  `TestServeDirDuplicateSlugIsUsageError` (two `--dir` values resolving to the same default
  slug → `exitUsage`, no listener bound) and
  `TestServeDirUnresolvableIsError` (a `--dir` pointing outside any `pickle.toml` tree →
  `exitError`, via `config.Find`'s existing "no pickle.toml found" error).
- `internal/serve/serve_test.go`: `TestMultiHandlerRoutesArePrefixed` — build two fixture trees
  with `standardTree(t)`, `MultiHandler` over two `NamedRoot`s, assert `GET /p/a/` renders that
  root's board and `GET /p/b/t/T-1` renders the other's ticket, and that `GET /p/a/t/<a-b's
  T-1>` is unaffected by the other root's identically-numbered ticket (the id-collision case
  named in the Description). `TestMultiHandlerIndexListsEveryRoot` — `GET /` shows both slugs
  and their ticket counts. `TestMultiHandlerDuplicateSlugRejected` — `MultiHandler` returns an
  error for two roots sharing a slug. `TestClassicModeUnaffected` — render the same fixture
  through plain `Handler` (BasePath/Peers zero-valued) and assert the HTML is byte-identical to
  the pre-change golden output for `/`, `/t/T-1` and `/activity` (guards decision 2). Extend
  `TestStaticAssetsAndHealthz` to also fetch `/static/styles.css` and `/healthz` through a
  `MultiHandler`-built mux and confirm both are reachable unprefixed (decision 6).
- Manual: `just build`, then in a scratch dir
  `D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D"` (per this repo's self-modify
  policy — never run a dev binary against this repo's own tree), create two throwaway
  `pickle install --in-tree` trees side by side, `./pickle-test serve --dir a --dir b=./b`,
  confirm the printed URLs, the index page, the switcher, and that stopping with Ctrl-C
  releases both locks (`lsof`/a second `pickle board audit` in each tree succeeds immediately
  after).

### Docs update (mandatory when user-facing)

`docs/user-manual/cli-reference.adoc`, `#cmd-serve` section (~line 1527): update the usage
synopsis to `pickle serve [--addr|-a host:port] [--dir [name=]path ...]`, document repeated
`--dir`, the default-vs-explicit slug rule, the duplicate-slug startup error, the `/p/{slug}/`
prefix and `/` becoming an index in named-roots mode, and reword the `[WARNING]` block to say
"every ticket in every served project" for that mode. Update the one-line command-table entry
at line 79 (`pickle serve [--addr …]` → `pickle serve [--addr …] [--dir …]`). `README.md`'s
mention needs no change — "visualize the board in a browser" already covers both modes.

### Finish (mandatory)

1. `just test`, `just lint`, `just build` clean; the acceptance test's manual scratch-dir check
   run at least once.
2. Docs updated per above and registered (no new doc file — existing page only).
3. Write the summary: files touched, decisions 1–8 as shipped (or any that changed and why),
   anything deferred.
4. Suggested commit message:

   ```
   feat(serve): support multiple project roots via repeatable --dir (T-127)

   pickle serve now accepts repeatable --dir [name=]path. With no --dir it behaves exactly as
   before (single root from cwd). With one or more, each root is mounted at /p/{slug}/, / is
   an index of every served board, and per-project pages gain a switcher — no aggregation, no
   merged view, so identically-numbered tickets across boards never collide.
   ```

5. Tidy WIP commits into atomic ones (root-path child, rules §0), commit locally on
   `feat/T-127-serve-multi-dir`. Publish only per the project's commit policy (no push / no MR
   without explicit user approval) — present the commit message for approval first.
6. `pickle ticket move T-127 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: chat: user asked to let one `pickle serve` process
  serve multiple project roots instead of one process per port
- 2026-09-02 — TO DO → READY: plan complete
