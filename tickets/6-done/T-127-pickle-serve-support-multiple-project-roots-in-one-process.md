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

> **Amended in place, 2026-09-02** (History has the dated line): the paragraphs below
> describe what actually shipped, not the original plan. The original text said
> `{{.BasePath}}` could be read directly wherever a page-level link needed it. That is true
> only for a link inside a template block reached by `{{with}}`/`{{range}}` — it breaks for
> `ticket-item` and `idlist`, both reached via `{{template "name" arg}}`, because Go's
> `{{template}}` action rebinds `$` to `arg` for that invocation, so a nested block can never
> read the page's `BasePath` back through `$`. Verified empirically (a two-line reproduction
> against `text/template`) before writing the fix below, not assumed.

`internal/serve/view.go`: add `BasePath string`, `Roots []PeerLink`, `Index *IndexView` to
`page`; `newPage` sets `BasePath: h.opts.BasePath, Roots: h.opts.Peers`. Additionally, give
**every leaf view struct a `BasePath` field of its own** — `Entry.BasePath`, `Event.BasePath`,
and a new `IDList{BasePath string; IDs []string}` — populated by `newEntry`, `buildActivity`'s
event literal, and a new `idListOf(basePath string, ids []string) IDList` template func
(`funcs.go`) respectively. `TicketView` embeds `Entry`, so it gets `BasePath` for free.
`buildBoard`, `buildActivity` and `buildTicket` each gain a trailing `basePath string`
parameter, threaded down to `newEntry`/`stateChildGroup`; every call site in `serve.go`
(`board`, `boardFragment`, `activity`, `activityFragment`, `ticket`) passes `h.opts.BasePath`.
This means a template reads `.BasePath` off whatever `.` it already has — Entry, Event,
TicketView or IDList — never `$.BasePath`, and the result is correct regardless of how many
`{{template}}` calls sit between the page and the link.

`internal/serve/templates/*.html`: prefix every internal absolute link with `{{.BasePath}}`
(classic mode's `BasePath == ""` makes this a no-op, so these edits change no rendered output
for an existing single-root caller):

- `layout.html`: `href="{{.BasePath}}/"` (brand + nav), `href="{{.BasePath}}/activity"` (nav);
  `.` is `page` at both sites (reached via `{{template "head" .}}`, which passes `page` through
  unchanged, not narrowed). Wrap the Board/Activity nav in `{{if not .Index}}...{{end}}` (they
  don't apply to the index page). Add, inside `head`, a switcher: `{{if .Roots}}<nav
  class="switcher">{{range .Roots}}<a href="/p/{{.Slug}}/">{{.Name}}</a>{{end}}</nav>{{end}}` —
  an absolute `/p/{slug}/` per peer, not `{{.BasePath}}`-relative, since it must reach a
  *different* project's namespace. `href="/static/..."` and the `htmx.min.js` script tag stay
  absolute (decision 6). Change the shared `idlist` block itself to
  `{{define "idlist"}}{{range $i, $id := .IDs}}...href="{{$.BasePath}}/t/{{$id}}"...{{end}}{{end}}`
  — here `$` is correct, not a bug: `idlist` is invoked via `{{template}}`, so `$` inside it is
  the `IDList` the caller passed, which is exactly where its own `BasePath` field lives.
- `board.html`: prefix both `hx-get="/fragments/board"` occurrences (`.` = `page` there, direct).
  Inside `ticket-item` (`.` = `Entry`, via `{{template "ticket-item" .}}`): both
  `href="{{.BasePath}}/t/{{.ID}}"` occurrences and `href="{{.BasePath}}/t/{{.Family}}"` read
  `Entry.BasePath` directly; `{{if .DependsOn}}...{{template "idlist" (idListOf .BasePath
  .DependsOn)}}...{{end}}` (and the same for `.SpawnedBy`) build the `IDList` the shared
  `idlist` block needs, from `Entry.BasePath` still in scope at the call site.
- `activity.html`: prefix both `hx-get="/fragments/activity"` occurrences (`.` = `page`,
  direct). Inside `activity-body` (`.` = `Event`, via `{{range .Events}}` after a `{{template
  "activity-body" .Activity}}` call): `href="{{.BasePath}}/t/{{.ID}}"` reads `Event.BasePath`.
- `ticket.html`: the crumbs `href="{{.BasePath}}/"` and the family
  `href="{{.BasePath}}/t/{{.Family}}"` both sit inside `{{with .Ticket}}` (which, unlike
  `{{template}}`, does *not* rebind `$` — but does rebind `.` to `TicketView`, so `.BasePath`
  resolves via the embedded `Entry` either way); the four `idlist` sites
  (`DependsOn`/`Blocks`/`SpawnedBy`/`Spawned`/`Members`) each become `{{template "idlist"
  (idListOf .BasePath .DependsOn)}}` and so on, same pattern as `ticket-item`.

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

### Round 1 — 2026-09-02

- [x] Reviewer independence settled (step 0): **degraded — recorded conscious skip.** Delegation
      to an independent reviewer was attempted **twice** (fresh sub-agents, briefed adversarially,
      no memory of writing the branch); both runs were terminated by the user mid-audit (~86 min
      and ~50 min). Their partial output survives and is credited below: the first confirmed F1,
      the second was mid-way through confirming F2 when stopped. With no third delegation
      available, the remaining audits were run by the **authoring agent** — the degradation this
      step exists to flag. Mitigation: every finding below is backed by executed evidence
      (command output, rendered HTML, a byte-diff against `main`), not by reading alone.
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (steps 1, 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass (step 4b) — run at implementation time over
      `cli-reference.adoc`; 1 suggestion touching this branch's own new prose applied, the rest
      discarded as out-of-scope pre-existing text. 0 fabricated quotes.
- [x] Findings recorded with severity, class and disposition; disposition + cost lines present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated; governing documents reconciled (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval (step 9)

**Acceptance test re-run verbatim** — `just test` PASS (all 21 packages), `just lint` PASS
(`go vet` + gofmt clean), `just build` PASS, `just docs-check` PASS (snowball render + `TestDocs`).
Manual multi-root exercise re-run in a scratch dir with a renamed `pickle-test` binary per the
self-modify policy: two in-tree trees each carrying their own `T-001`, `--dir a --dir b=./b` —
index, both boards, both colliding `T-001` ticket pages, switcher, shared static, `/healthz`,
wrong-slug 404 all correct; clean `SIGINT` shutdown with both locks released (verified by an
immediate `pickle board audit` in each tree).

**Audited clean** (recorded so the archive can tell "checked" from "not checked"): reserved-looking
slugs (`static`, `healthz`, `t`, `p`) do **not** shadow the real routes — the `/p/` namespace
prevents collision, all four verified serving 200 alongside intact `/static/` and `/healthz`;
write-shaped requests still get **405** on both `/p/{slug}/` and the index, so the "never writes"
posture holds in multi-root mode; `HEAD` works; template link prefixing is **complete** — the only
remaining unprefixed links are `/static/*` (decision 6) and the deliberately-absolute cross-namespace
`/p/{slug}/` switcher/index links; no path-traversal surface (static is an embedded FS, ids go
through `ticket.ValidID`); whole-docs-tree sweep found no page made stale — `multi-project.adoc`
describes the *opposite* axis (one shared board, connected children) and is untouched by this change.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | correctness | fixed inline | The three new `errf` calls in `resolveNamedRoots` add their own `"pickle serve: "` prefix, but `errf` already prepends `"pickle: "` — producing garbled double-prefixed output. Every pre-existing `errf` call in the same file relies on `errf`'s prefix alone. | `internal/cli/serve.go:186,190,194`; observed output `pickle: pickle serve: --dir /x: no pickle.toml found…` | Drop the redundant prefix. **Fixed during review** in `030cc26`; output is now `pickle: --dir /x: …`, matching the file's convention. |
| F2 | **blocking** | correctness | — | The named-roots index page renders a fabricated green health banner. `indexHandler.index` builds `page{Title, Project, Index}` and never sets `Health`, but `layout.html`'s `head` block renders `{{template "health" .Health}}` unconditionally; the zero-valued `HealthView.OK()` returns true, so the banner claims **"0 tickets · board audit clean"** — describing no project at all, and directly contradicting the rows beneath it. | `internal/serve/serve.go` (`indexHandler.index`), `layout.html:30`. Rendered against a root with a real audit error: top banner `class="health health-ok"` → `0 tickets · board audit clean`, project row immediately below → `1 tickets · 1 audit error(s), 0 warning(s)`. | Suppress the page-level banner when `.Index` is set (`{{if not .Index}}{{template "health" .Health}}{{end}}`) — the per-row health already carries the real state, and aggregating instead would be the cross-project aggregation decisions 6/7 rule out. Add a regression test asserting the index carries no `health-ok` banner when a served root has audit errors. |
| F3 | **blocking** | docs-gap | — | No `CHANGELOG.md` entry for a user-facing feature. `## [Unreleased]` is empty; every recent feature ticket (T-124, T-126) carries an entry naming its id. The Implementation Plan's own "Docs update" step omitted CHANGELOG, so the branch faithfully executed a plan that had the gap. | `pickle changelog check` → `1 candidate(s) shipped but not named in "Unreleased": T-127`; `CHANGELOG.md:9` (`## [Unreleased]`, empty) | Add an `### Added` entry under `## [Unreleased]` describing repeatable `--dir`, the `/p/{slug}/` namespacing and its id-collision rationale, the index page, and the unchanged zero-flag default — ending `(T-127)`. |
| F4 | non-blocking | test-gap | noted | `TestClassicModeUnaffected` is weaker than the acceptance criterion it claims to guard. The plan specifies asserting classic-mode HTML is *byte-identical to pre-change golden output*; the test only asserts the absence of `/p/` and of doubled slashes, which would miss a dropped link or altered attribute. | `internal/serve/serve_test.go` (`TestClassicModeUnaffected`); plan's Acceptance test section | Left as `noted`: a golden-file harness is disproportionate here, and this review has now established the stronger property empirically (see F5) and recorded it permanently. A future reviewer can promote this row by citing it. |
| F5 | non-blocking | stale-xref | noted | Decision 2's literal wording ("byte-for-byte today's behavior") is **not** met, though its intent is. Classic-mode output differs from `main` by exactly 27 bytes — 9 blank lines of indentation left behind by the `{{if not .Index}}` / `{{if .Roots}}` conditionals when false. | Rendered `/`, `/t/T-002`, `/activity` on `main` (detached worktree) vs this branch: `diff` shows only whitespace-only added lines; whitespace-normalized outputs are **identical**. | Recorded, not rewritten — the divergence between decision text and shipped behaviour is the datum (same reasoning as the cost line). Anyone later writing the golden test F4 describes must expect these 9 lines, or trim-mark the two conditionals first. |
| F6 | non-blocking | design | noted | `MultiHandler` parses the full template set once for itself and then once more inside every `Handler(opts)` call it makes — N+1 parses of identical embedded templates for N roots. | `internal/serve/serve.go` (`MultiHandler` → `template.New("").Funcs(funcs).ParseFS(…)`, then `Handler` does the same per root) | Startup-only, and N is a handful, so not worth restructuring `Handler`'s signature to inject a shared `*template.Template`. Noted in case a future change makes `Handler` construction hot. |
| F7 | non-blocking | spec-unclear | noted | `parseDirArg` splits on the **first** `=`, so a directory path legitimately containing `=` mis-parses: `--dir /tmp/a=b` silently becomes slug `/tmp/a`, path `b`, then fails resolving `b` against cwd with a confusing message. The `name=path` syntax's precedence is undocumented. | `internal/cli/serve.go` (`parseDirArg`, `strings.Cut(v, "=")`); `cli-reference.adoc` `#cmd-serve` documents `[name=]path` without stating the split rule | Common to every `name=value` CLI convention and rare in practice. If it ever bites, document "split on the first `=`" in `#cmd-serve` rather than adding escaping. |

**Disposition summary:** 7 findings — **2 blocking** (F2 index health banner, F3 missing CHANGELOG
entry) → `5-rework/`; 5 non-blocking: 1 `fixed inline` (F1, commit `030cc26`), 4 `noted`
(F4, F5, F6, F7). 0 `folded`, 0 `new ticket` — nothing here passes the promotion test, and the four
`noted` rows stay recoverable by citation.

cost: estimated M, actual M

**Blocking findings are the entire scope of the rework** (F2, F3). F1 is already fixed; F4–F7 are
closed and must not be re-opened by the fix pass.

### Rework fix record — round 1 (commit 6d2c1a4)

- **F2 fixed.** `internal/serve/templates/layout.html`: the shared `{{template "health"
  .Health}}` block is now guarded `{{if not .Index}}…{{end}}` — the index page (`page.Index`
  set) never had a single project's health to report, so `page.Health`'s zero value read as
  `OK() == true` and rendered a fabricated "board audit clean" banner directly contradicting the
  real per-project error shown in the row beneath it. Regression-tested
  (`TestMultiHandlerIndexNeverFabricatesCleanHealth`, `internal/serve/serve_test.go`): reproduces
  the exact contradiction (a root with a real audit error), asserts the top-level banner is gone
  **and** the real per-project error count still shows, **and** that classic mode and an ordinary
  per-project board page both keep their own banner unchanged — the fix is scoped to the index
  only.
- **F3 fixed.** `CHANGELOG.md`: added the missing `## [Unreleased]` → `### Added` entry
  describing repeatable `--dir`, the `/p/{slug}/` namespacing and its id-collision rationale, the
  index page, and the unchanged zero-flag default, ending `(T-127)`. `pickle changelog check` now
  reports `no candidates — every shipped ticket is mentioned`.

Acceptance test re-run verbatim after the fix: `just test` PASS (all 21 packages, including the
new regression test), `just lint` PASS, `just build` PASS, `just docs-check` PASS.

> **SHA correction (round 2):** this record was written before the branch was rebased onto the
> updated `main`, which rewrote the commit. The fix is **`9a162bb`** on the branch; the cited
> `6d2c1a4` is the pre-rebase original — still resolvable as a dangling object today, but not
> reachable from `HEAD` and gone on any fresh clone. Their code content is identical
> (`git diff 6d2c1a4 9a162bb -- internal/ CHANGELOG.md` is empty); only the bookkeeping files
> differ. Recorded per round 2's R1.

### Round 2 — scoped re-review — 2026-09-02

**Scope** (protocol §1): the two blocking findings from round 1, plus the diff that closed them
(`9a162bb`) read as new work. Not a re-audit of the feature; F4–F7 stay closed and were not
re-opened.

- [x] Reviewer independence (step 0): **degraded — recorded conscious skip, second time.**
      Delegation was attempted twice in round 1 and terminated mid-audit both times; for a
      one-commit, three-file fix diff a third long-running delegation was judged
      disproportionate. Audits run by the authoring agent. Mitigation, beyond round 1's: the
      regression test was **verified to fail when the fix is reverted** (below), and F2 was
      confirmed end-to-end against the real binary on a deliberately corrupted board rather than
      only through the fixer's own unit test.
- [x] Both blocking findings verified closed — evidence below
- [x] Fix diff (`9a162bb`) audited as new work — 2 new non-blocking findings (R1, R2)
- [x] Scope check: the fix touched only `CHANGELOG.md`, `internal/serve/serve_test.go`,
      `internal/serve/templates/layout.html` — no creep beyond F2/F3
- [x] Acceptance test re-run verbatim: `just test`, `just lint`, `just build`, `just docs-check`
      all PASS; `pickle changelog check` → `no candidates`

**F2 — CLOSED.** Verified end-to-end with the built binary in a scratch dir (renamed
`pickle-test`, per the self-modify policy), serving two in-tree roots of which one was
deliberately corrupted to produce a real audit error (`impact: spicy` + a desynced board → 2
errors):

- index: **0** occurrences of `board audit clean`, **0** of `health-ok` — the fabricated banner
  is gone;
- index still reports the true per-root counts side by side: `0 audit error(s)` for the healthy
  root, `2 audit error(s)` for the corrupted one;
- the per-project page keeps its own detailed banner (`/p/broken/` → *illegal impact value
  "spicy"*), and `/p/good/` renders `health-warn` → `1 tickets · 0 audit error(s), 1 warning(s)`,
  which **matches that tree's own `pickle board audit` output exactly** — a stronger property
  than the finding asked for: the served banner and the CLI agree on the same tree.

**The regression test is a real guard, not a tautology:** reverting the one-line template change
makes `TestMultiHandlerIndexNeverFabricatesCleanHealth` **FAIL**, and restoring it makes it pass
— executed both ways during this re-review.

**F3 — CLOSED.** `pickle changelog check` → `no candidates — every shipped ticket is mentioned`.
Entry content re-read against shipped behaviour: accurate on the zero-flag default, the slug
rule, the startup-error-before-listener guarantee, the no-aggregation rationale, the switcher,
and the shared unprefixed static/`healthz`.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | non-blocking | stale-xref | fixed inline | The round-1 rework fix record cites commit `6d2c1a4`, which the post-rework rebase orphaned; the fix is `9a162bb` on the branch. Protocol §1 anticipates exactly this ("a tidy at publish time rewrites them") and directs that it be treated as a finding of its own, never blocking. | `git merge-base --is-ancestor 6d2c1a4 HEAD` → not reachable; `git diff 6d2c1a4 9a162bb -- internal/ CHANGELOG.md` → empty (code identical, only bookkeeping differs) | Corrected by the **SHA correction** note added directly above the round-2 heading, rather than by rewriting the original record — the rebase is part of the history, not something to erase. |
| R2 | non-blocking | stale-xref | fixed inline | Round 1's F5 measured classic-mode divergence from `main` as **27 bytes / 9 blank lines**. The F2 fix's own 5-line `{{/* … */}}` comment emits a bare newline where it sits, adding **1 blank line per page** in *every* mode, so that figure is now stale. | Classic `/`, `/t/T-002`, `/activity` rendered at `a35ef38` (pre-fix) vs `9a162bb`: +3 bytes, 3 blank-line-only additions. Against current `main`: **30 bytes / 12 blank lines**, and whitespace-normalized output is **identical** — still no semantic difference. | Figure corrected here; F5's round-1 row is left as written, since it was accurate when taken. Not chased further: the divergence remains cosmetic HTML whitespace, exactly the class F5 already dispositioned `noted`. Trim markers (`{{- -}}`) would remove it if anyone ever wants byte-parity. |

**Disposition summary (round 2):** 2 findings, both non-blocking, both `fixed inline` (R1, R2) —
record-accuracy corrections, no code change. 0 blocking, 0 `folded`, 0 `new ticket`, 0 `noted`.
Both round-1 blocking findings (F2, F3) verified closed. **Verdict: DONE.**

cost: estimated M, actual M

## History

- 2026-09-02 — created (TO DO). source: chat: user asked to let one `pickle serve` process
  serve multiple project roots instead of one process per port
- 2026-09-02 — TO DO → READY: plan complete
- 2026-09-02 — READY → IN DEVELOPMENT: picked up
- 2026-09-02 — plan amended inline: Task 4's BasePath threading did not account for Go's
  `{{template}}` action resetting `$`, which breaks `ticket-item` and `idlist` (both invoked
  via `{{template}}`); fixed by giving `Entry`, `Event` and a new `IDList` their own `BasePath`
  field instead of relying on `$.BasePath`, verified against a two-line `text/template`
  reproduction before implementing
- 2026-09-02 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-09-02 — IN REVIEW → REWORK: review round 1: 2 blocking (index health banner, missing CHANGELOG entry)
- 2026-09-02 — REWORK → IN REVIEW: findings fixed
- 2026-09-02 — IN REVIEW → DONE: scoped re-review clean: F2 and F3 verified closed; 2 non-blocking (R1, R2) both fixed inline
