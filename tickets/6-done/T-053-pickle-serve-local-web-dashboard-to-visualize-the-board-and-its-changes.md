---
id: T-053
title: pickle serve — local web dashboard to visualize the board and its changes
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: high
cost: L
---

# T-053 — pickle serve — local web dashboard to visualize the board and its changes

## Description

Add a `pickle serve` command that starts a **local, read-only web server** rendering the
board as a browser dashboard, so a human can *see* the flow — the board grouped by
child-project, each ticket's prose, and how the board changed over time — instead of reading
`tickets/BOARD.md` and grepping `## History` sections.

Today the only views of the flow are the generated `tickets/BOARD.md` table and the ticket
markdown files. That is enough for agents and diffs, but poor for humans: a ticket's real
story (description, plan, review findings, history) lives in one file per ticket, and "what
moved this week" is only reconstructable by reading every ticket's `## History`. `serve`
turns the existing, already-authoritative data into a navigable surface — no new source of
truth, no writes.

**Scope (pinned down at refinement — see the plan's confirmed decisions).**

- `pickle serve [--addr host:port]` — starts an HTTP server on `127.0.0.1:8745` by default;
  prints the URL; runs until Ctrl-C. **Read-only**: no route mutates a ticket, moves a file, or
  regenerates the board — the CLI stays the only writer.
- **Board view** — the seven statuses as columns/sections, grouped by child-project, ordered
  exactly as `internal/board` orders the generated board (impact descending, ties by id), so
  the dashboard and `BOARD.md` can never disagree.
- **Ticket view** — one page per ticket: frontmatter as a header (project, grades,
  `depends-on:`, `spawned-by:`), then the rendered markdown body (Description, Implementation
  Plan, Review, History), with dependency and lineage ids as links.
- **Changes view** — "and its changes" has two readings, and this ticket ships both:
  1. **Activity timeline** built from the ticket files themselves — every dated `## History`
     line across all tickets, merged and sorted newest-first (transition, reason, child).
     Pure function of the repo, no git needed.
  2. **Live refresh** — every request re-reads the ticket files (no cache), so the dashboard is
     already current on reload; htmx polling of the board + activity fragments makes it update
     on its own, so it works as an ambient window while an agent moves tickets.
- **Health/diagnostics** — surface `pickle board audit` findings and the WIP-limit state per
  child as a banner, so an out-of-sync or invariant-breaking board is visible in the UI
  (running the audit read-only, never auto-fixing).

**Tech stack — deliberately minimal, mirroring `~/Projects/private/unity/rick/apps/standards`
(the reference implementation the request pointed at):**

- Go stdlib `net/http` + `http.ServeMux` only — no web framework, no router dependency.
- `html/template` templates and CSS/JS assets under `internal/serve/` (name TBD), **embedded
  with `//go:embed`** so the single `pickle` binary keeps working with no network, no asset
  directory and no build step (same property as the existing skill payload in `assets.go`).
- Hand-written CSS; **htmx** as the only client-side script (vendored at a pinned version and
  served from the embedded FS) for polling/partial swaps — no npm, no bundler, no SPA.
- Markdown → HTML for ticket bodies via **`goldmark` + the GFM extension** (pickle's second
  direct dependency, approved at refinement — same choice as `standards`).
- Tests as `httptest`-driven handler tests over a fixture `tickets/` tree, matching the
  existing `internal/*` table-test style; wiring goes in `internal/cli/` next to `board.go`,
  with the command added to the dispatch table and usage text in `internal/cli/cli.go`.
- User-facing surface, so it needs a `docs/user-manual/` chapter and a `just docs-check` pass.

**Soft couplings** (no hard `depends-on:` proposed):

- T-044 made `BOARD.md` a generated artifact with the ticket files as the single source of
  truth — that is the invariant this dashboard reads from, and the reason it can be a pure
  view. `serve` must render from ticket files via `internal/board`/`internal/ticket`, not by
  parsing `BOARD.md`.
- T-049 (rendered cell-width cap) and T-052 (audit vs. registry-changed board) shape what the
  board renderer and audit report expose; if `serve` reuses those, expect small refactors
  rather than duplicated logic.
- T-046 (self-host awareness) is adjacent only: `serve` run in this repo will see the same
  self-host quirks the doctor warnings describe.

**Explicit non-goals** (to keep the first cut minimal): no ticket editing or moving from the
browser, no authentication, no non-loopback binding by default, no multi-repo/remote hosting,
no persistence or database, no git history mining beyond what the ticket files already record,
no `--open`/browser launching (platform-specific `exec`, deliberately deferred), and no change
to the skill payload — `serve` is a human surface, not something an agent is told to run.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                      # child `pickle` is this repo (self-hosted)
git checkout main
git checkout -b feat/T-053-serve
```

Commit locally as you go. **Never push or open a merge request without explicit user
approval** (publish-gated child): end with a summary and a suggested commit message.

### Prerequisite gate (hard)

- `depends-on: []` — nothing to wait for.
- Clean tree on `main`; `pickle board audit` reports 0 errors before you start.
- `3-in-development/` for child `pickle` is empty (WIP limit 1).
- Network access once, to fetch `goldmark` (`go get`) and the pinned `htmx.min.js`. If the
  environment is offline, stop and say so rather than hand-writing either.
- **Self-modify policy** (`AGENTS.md`): never run `pickle install|upgrade|uninstall` against
  this repo. `serve` is read-only, so running `./pickle serve` here *is* allowed; the install
  smoke test goes to a throwaway dir with the binary copied in.

### Confirmed design decisions (do not deviate without asking)

1. **Read-only, absolutely.** No handler writes, moves, renames or regenerates anything — not
   even `BOARD.md`, not even to "fix" a stale board. `serve` opens files for reading only. The
   CLI (`ticket new|move`, `board sync`) stays the single writer. A test asserts this (Task 7).
2. **Ticket files are the source of truth — `serve` must never call `board.Parse`.** Views are
   built from `ticket.LoadAll(root)` + `pickle.toml`, honouring T-044. The one permitted board
   read is the one *inside* `audit.Audit` (`internal/audit/audit.go:86-93` reads `BOARD.md` and
   compares it to a fresh render — that freshness check is the audit's whole point, and the
   health banner surfaces its verdict). `serve` itself never reads the file (amendment F2).
3. **One ordering *and one WIP count*, shared.** The dashboard must never disagree with
   `BOARD.md`, so the status order, the per-group sort **and the per-child WIP counts** come
   from `internal/board` (Task 2 exports them; do not re-implement `impactRank`, `sortRows` or
   the WIP tally in `internal/serve` — amendment F6).
4. **No `sanitizeCell` in HTML.** The 120-rune cap and the `|` → `¦` substitution are
   markdown-table concerns (T-049). HTML shows full values; escaping is `html/template`'s job.
5. **Command:** top-level `pickle serve [--addr host:port]`. Default addr `127.0.0.1:8745`.
   Unknown flags/stray args → exit code 2 (`exitUsage`), matching `board sync`'s argument
   handling. A non-loopback `--addr` is allowed but prints a prominent warning to stderr (no
   auth, read-only, local tool).
6. **Stack:** Go stdlib `net/http` + `http.ServeMux` (Go 1.22+ method/wildcard patterns),
   `html/template`, `//go:embed` for templates and static assets, hand-written CSS, **vendored
   htmx 2.0.4** (`htmx.min.js`, ~51 KB, plus its `LICENSE`) as the only client script, and
   **`goldmark` + `extension.GFM`** for ticket-body markdown. No web framework, no router, no
   npm/bundler, no CDN reference (offline-capable single binary).
7. **No cache, no file watcher.** Every request re-reads the tree; freshness is by construction.
   Auto-update is htmx polling (`hx-trigger="every 5s"`) of two fragment routes. Do not add
   fsnotify, websockets or SSE.
8. **No `--open`.** Print the URL; the human clicks it.
9. **Skill payload untouched.** No edits under `skill/`, and **no `AGENTS.md` marker-block
   change** — `serve` is not part of the flow rules an agent follows.
10. **Graceful shutdown.** `signal.NotifyContext` on SIGINT/SIGTERM → `http.Server.Shutdown`;
    set `ReadHeaderTimeout`/`ReadTimeout` (15s) rather than leaving them at zero.

### Tasks

#### Task 1 — `ticket.HistoryEntries`: the activity timeline's data source

In `internal/ticket/ticket.go`, add one exported parser beside the existing
`LastHistoryStatus`/`LastHistoryReason`/`MergeLine` (which already scan `## History` with the
unexported `historyRE`):

```go
type HistoryEntry struct { Date, Text string } // Date is the raw YYYY-MM-DD
func HistoryEntries(text string) []HistoryEntry // file order (oldest first)
```

Reuse `historyRE` — do not add a second history regex. Return every dated line, including
created and merge lines (the timeline wants them); classification is the view's job.
Keep `Date` a string: the format is already anchored by the regex, and no view needs
`time.Time`.

**Amendment F1 — the date is not captured today.** `historyRE`
(`internal/ticket/ticket.go:104`) is `^-\s*\d{4}-\d{2}-\d{2}\s*[—-]+\s*(.+)$`: the date is
matched but **not** in a group, and `m[1]` is the *body*, relied on by three callers
(`ticket.go:192`, `:231`, `:272`). Do **not** add a leading group and shift the indices. Either:

- add a **named** group (`(?P<date>\d{4}-\d{2}-\d{2}\)`) and read it via
  `historyRE.SubexpIndex("date")`, leaving every existing `m[1]` untouched; or
- keep the regex byte-identical and slice the date off the matched line separately.

Either way the existing `m[1]` semantics must not move. Extend
`internal/ticket/ticket_test.go` with a table test (multiple entries in order, em-dash *and*
hyphen separators, merge and created lines kept, non-matching bullets and lines outside
`## History` ignored, empty History → nil) **plus explicit regression assertions** that
`LastHistoryStatus`, `LastHistoryReason` and `MergeLine` still return what they did before —
those three are the index-shift blast radius.

#### Task 2 — export the board's ordering from `internal/board`

In `internal/board/board.go`, expose what `Render` already uses so `serve` shares it:

- `func StatusOrder() []string` — a copy of `boardOrder` (return a copy; callers must not be
  able to mutate package state).
- `func Sort(group []*ticket.Ticket, statusName string)` — rename `sortRows` to `Sort`. It has
  exactly **one** call site (`board.go:245`), not two (amendment F5); no wrapper — decision 3
  wants one implementation, so leave no alias behind.
- `func WIPCounts(tickets []*ticket.Ticket, cfg *config.Config) …` — one shared per-child tally
  of `3-in-development/` and `4-in-review/` against the configured limits (amendment F6). The
  count is currently inlined **twice** (`board.go:245-251` for the `(n/limit)` headings and
  `audit.go:136+` in `auditWIP`); `serve` would have made a third. Land the helper, then make
  `board.Render` use it, and — if it is a clean fit — `audit.auditWIP` too. If adapting the
  audit turns out to be more than a mechanical substitution, leave `auditWIP` alone and say so
  in the summary rather than growing this ticket.
- Export `func SectionHeading(statusName string) string` (the `sectionHeading` map,
  `board.go:33`) **only if** the UI actually renders those headings; do not export it "just in
  case".

Add a test in `internal/board/board_test.go` asserting the shared path really is shared: for a
fixture set, the id order of a `Sort`ed TO DO group equals the row order `Render` emits in the
TO DO section (decision 3's guarantee, mechanically enforced).

#### Task 3 — new package `internal/serve`: views

Create `internal/serve/` (package `serve`).

- `view.go` — pure view builders over `[]*ticket.Ticket` + `*config.Config`, no HTTP:
  - `BoardView`: per status (in `board.StatusOrder()`), per registered child (`cfg.Projects`
    order), the `board.Sort`ed tickets; each entry carries id, title, project, impact,
    complexity, cost, `depends-on`, `spawned-by`, the last History reason, and the merge line;
    plus per-child WIP counts and limits for IN DEVELOPMENT/IN REVIEW **from
    `board.WIPCounts`** (amendment F6 — never a local tally).
  - `TicketView`: the ticket, its frontmatter fields, rendered body HTML (Task 4), and its
    dependency/lineage ids as links; also the reverse edges ("blocks" / "spawned") computed by
    scanning all tickets once — information `BOARD.md` cannot show.
  - `ActivityView`: every `ticket.HistoryEntries` line from every ticket, tagged with id, title
    and child, sorted **newest first** (ties: higher ticket number first, then file order) with
    a cap (e.g. 200 entries) and a total count.
  - `HealthView`: `audit.Audit(root, cfg)` errors/warnings + ticket count, and the WIP state per
    child — rendered as a banner. Audit is read-only and pure; never auto-fix.
  Cover each builder with table tests (ordering, grouping, WIP counts, reverse edges, cap).
- `serve.go` — wiring:
  ```go
  type Options struct{ Root string; Cfg *config.Config }
  func Handler(opts Options) (http.Handler, error)          // builds the mux; parses templates once
  func Serve(ctx context.Context, ln net.Listener, opts Options) error  // listener injected
  ```
  Taking a `net.Listener` (not an addr) is what makes the end-to-end test in Task 7 possible on
  port 0. Routes:
  | route | renders |
  |---|---|
  | `GET /` | board dashboard (all statuses, per child) |
  | `GET /t/{id}` | one ticket; unknown or malformed id → 404 (validate with `ticket.ValidID`) |
  | `GET /activity` | the timeline page |
  | `GET /fragments/board` | board fragment (htmx poll target) |
  | `GET /fragments/activity` | timeline fragment (htmx poll target) |
  | `GET /static/` | `http.StripPrefix("/static/", http.FileServerFS(sub))`, `sub, _ := fs.Sub(assets, "static")` |
  | `GET /healthz` | `200 OK`, plain text |
  Register with method-qualified patterns (`"GET /"`) so a POST gets 405 from the mux rather
  than being served — a cheap structural half of decision 1.

  **Amendment F3 — the static route.** `GET /static/...` is *not* ServeMux pattern syntax (the
  legal forms are the `GET /static/` prefix or `GET /static/{path...}`), and because the embed
  is rooted at the package dir, serving `assets` directly would resolve `/static/styles.css` to
  `static/static/styles.css`. Use `fs.Sub(assets, "static")` + `http.StripPrefix` exactly as in
  the table above.

  **Amendment F4 — `GET /` is a catch-all.** With the stdlib mux, `/favicon.ico` and
  `/nonsense` both match `GET /` and would render the board with 200. The root handler must
  start with `if r.URL.Path != "/" { http.NotFound(w, r); return }`, and Task 7 must cover
  `GET /nope → 404`.

#### Task 4 — markdown rendering (`goldmark`)

`go get github.com/yuin/goldmark@v1.8.2` — **pin that exact version** (amendment F10: it is
already in the local module cache, so this task needs no network, and naming it lets review
check the `go.mod`/`go.sum` diff against an intent rather than against "whatever `go get`
picked"). Commit `go.mod` + `go.sum`; no vendoring (CI resolves from `go.mod`).
In `internal/serve/markdown.go`: one package-level renderer
`goldmark.New(goldmark.WithExtensions(extension.GFM))`; a `renderMarkdown(src string)
(template.HTML, error)` that strips the frontmatter block before rendering (the header shows
those fields structurally — don't render `---` as a horizontal rule). goldmark escapes raw HTML
by default: **do not enable `WithUnsafe`**, and note that in a comment, since the input is
repo-local but the output is `template.HTML`.

#### Task 5 — templates + assets

`internal/serve/templates/`: `layout.html` (head, embedded CSS/JS links, header with project
name + health banner), `board.html`, `ticket.html`, `activity.html`, `fragments.html` (the two
polled fragments, defined so page and fragment render from the *same* template block — no
duplicated markup). Parse once in `Handler` via `template.ParseFS`; a parse error is a startup
error, never a 500 at request time.

`internal/serve/static/`: `styles.css` (hand-written; readable status columns/sections, a
compact ticket page, monospace ids), `htmx.min.js` (**htmx 2.0.4**, fetched once from
`https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js`, committed verbatim — record the version and
URL in a comment in `serve.go`), and `htmx.LICENSE` (htmx is 0BSD; ship the license text
alongside the vendored file). Add `//go:embed templates static` in `internal/serve/serve.go`.

**Amendment F9 — measure, don't guess, the size delta.** The earlier "~55 KB" figure counted
only htmx; goldmark links a whole parser/renderer tree and is expected to dominate (order of
~1 MB). Record `ls -l pickle` before and after in the Finish summary as a measured number.

#### Task 6 — CLI wiring

New `internal/cli/serve.go`:

- `runServe(args []string) int` — parse `--addr` (default `127.0.0.1:8745`), reject unknown
  args/stray positionals with `exitUsage` and a `usage: pickle serve [--addr host:port]` line;
  `loadConfig()`; `net.Listen("tcp", addr)` (a bind failure → `errf`, exit 1); print
  `serving the board at http://<addr> (read-only; Ctrl-C to stop)` to stdout and the
  non-loopback warning to stderr; `signal.NotifyContext`; call `serve.Serve`.
- Split argument parsing into a small pure helper (e.g. `parseServeArgs([]string) (addr string,
  code int)`) so Task 7 can test flag handling without binding a port.
- Register `case "serve":` in `internal/cli/cli.go`'s dispatch **and** add it to the usage text
  (own short group, e.g. `Visualize:`), keeping the existing wording style.

#### Task 7 — tests

- `internal/serve/serve_test.go` (`httptest` over a fixture tree built in `t.TempDir()`: a
  `pickle.toml` with one child plus a handful of tickets across statuses — write them with a
  helper, don't shell out to the binary):
  - `GET /` → 200, contains each fixture id, TO DO ids appear in impact order, WIP counts shown.
  - `GET /t/T-002` → 200, body markdown rendered (an `<h2>`/`<li>` from the fixture appears),
    `depends-on` rendered as a link, reverse "blocks" edge shown.
  - `GET /t/T-999` → 404; `GET /t/nonsense` → 404; `POST /` → 405.
  - `GET /activity` → 200, newest-first (assert two known dates' relative positions).
  - `GET /fragments/board` → 200, no `<html`, contains the same ids as the page.
  - `GET /static/styles.css`, `GET /static/htmx.min.js` → 200, non-empty; `/healthz` → 200.
  - **Health banner:** a fixture ticket with an illegal `impact` value surfaces an audit error
    in `GET /` — and the assertion must match the **specific** substring
    (`illegal impact value`), not merely "an error is shown" (amendment F2).
  - **Fixture trees must contain a rendered `BOARD.md`** (write `board.Render(...)` output in
    the helper). `audit.Audit` compares the board to a fresh render, so a fixture without one
    reports a board error on every page — every banner test would pass for the wrong reason
    (amendment F2). Add one negative test: a clean fixture (board rendered) shows **no** audit
    errors, proving the banner discriminates.
  - `GET /nope` → 404 (amendment F4: the root pattern is a catch-all).
  - **Read-only proof (decision 1):** snapshot every file under the fixture root (path →
    size+mtime+sha256) before and after hitting every route, assert byte-identical and that no
    file was created or removed.
  - **XSS/escaping:** a fixture title containing `<script>` and `|` renders escaped, not raw
    (guards decision 4's "no sanitizeCell, escaping is the template's job").
  - **End-to-end:** `net.Listen("tcp", "127.0.0.1:0")` + `serve.Serve` in a goroutine, real
    `http.Get` of `/healthz`, then cancel the context and assert `Serve` returns without error
    (proves shutdown works).
- `internal/cli/cli_test.go`: add exit-code cases `{"serve bad flag", []string{"serve",
  "--bogus"}, exitUsage}`, `{"serve stray arg", ...}`, `{"serve missing addr value", ...}`, and
  a direct `parseServeArgs` table test (default addr, explicit addr). **No test may bind 8745.**

#### Task 8 — docs + CHANGELOG

- `docs/user-manual/cli-reference.adoc`: a row in the Overview table (`pickle serve [--addr …]`
  | "Serve a read-only browser view of the board and its history.") and a
  `[#cmd-serve] == \`pickle serve\`` section after `board sync`, before `pickle version`:
  synopsis, default `127.0.0.1:8745`, the route table, the 5s poll, the health banner, the
  read-only guarantee (and that it therefore never fixes a stale board — that is
  `pickle board sync`), and the explicit "no authentication; do not expose it" note for
  non-loopback `--addr`.
- `docs/user-manual/concepts/the-flow.adoc` (the generated-board paragraph, ~line 30): one
  sentence pointing at `pickle serve` as the browser view of the same generated truth. Follow
  the existing `[#cmd-…]` anchor convention, and **read `snowball check`'s output** rather than
  trusting its exit code: its failure level is WARN (`snowball.yaml`), so a broken
  `xref:cli-reference.adoc#cmd-serve` may not fail the build (amendment F14).
- The stderr warning for a non-loopback `--addr` must contain the words "no authentication", so
  the binary and the manual say the same thing (amendment F15).
- `CHANGELOG.md` under `## [Unreleased]` → `### Added`: the command, its read-only contract, the
  default address, and the two new dependencies (goldmark; vendored htmx) — dependency-set
  changes are exactly what a changelog reader wants to know.
- `README.md`: **no change** (deliberately install-only since T-047) — confirm, don't expand it.
- `just docs-check` must pass.

### Acceptance test

```sh
# 1. Build + full validate (from the repo root, on the feature branch)
just build && just test && just lint && just docs-check

# 2. The new package's own suite, verbose (read-only proof + e2e included)
go test ./internal/serve/... ./internal/board/... ./internal/ticket/... ./internal/cli/... -v

# 3. Read-only smoke against this repo's real board (allowed: serve never writes).
#    Amendments F7/F8: no fixed port, no `git status` purity check (the ticket move itself
#    dirties tickets/), no pkill. Bind :0, discover the port, hash-snapshot the tree.
before=$(find tickets -type f -exec shasum {} \; | sort)
PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')
./pickle serve --addr "127.0.0.1:$PORT" & SRV=$!
for i in $(seq 1 50); do curl -sf "http://127.0.0.1:$PORT/healthz" >/dev/null && break; sleep 0.1; done
curl -sf "http://127.0.0.1:$PORT/healthz"
curl -sf "http://127.0.0.1:$PORT/"          | grep -q 'T-053'
curl -sf "http://127.0.0.1:$PORT/t/T-044"   | grep -q 'single source of truth'
curl -sf "http://127.0.0.1:$PORT/activity"  | grep -q '2026-'
curl -sf "http://127.0.0.1:$PORT/fragments/board" >/dev/null
curl -so /dev/null -w '%{http_code}\n' "http://127.0.0.1:$PORT/t/T-999"  # expect 404
curl -so /dev/null -w '%{http_code}\n' "http://127.0.0.1:$PORT/nope"     # expect 404
curl -so /dev/null -w '%{http_code}\n' -X POST "http://127.0.0.1:$PORT/" # expect 405
kill "$SRV"; wait "$SRV" 2>/dev/null
test "$before" = "$(find tickets -type f -exec shasum {} \; | sort)" && echo "read-only: OK"

# 4. Usage contract (and the default addr is only documented, never bound by a test)
./pickle serve --bogus; echo "exit=$? (expect 2)"
./pickle help | grep -q 'serve'

# 5. Fresh-project smoke, per the self-modify policy: throwaway dir, copied binary
D=$(mktemp -d); cp pickle "$D/pk"; (cd "$D" && git init -q . \
  && ./pk install --project demo --path . --agent opencode \
  && ./pk ticket new "first demo ticket" --project demo \
  && P2=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()') \
  && ./pk serve --addr "127.0.0.1:$P2" & sleep 1; curl -sf "http://127.0.0.1:$P2/" | grep -q 'T-001'; kill %1)
```

Expected: every command exits 0 (except the deliberate 404/404/405/exit-2 checks), the `grep`s
match, and step 3 prints `read-only: OK` — the dashboard rendered this repo's live board
without touching a byte.

### Docs update (mandatory when user-facing)

Task 8: `docs/user-manual/cli-reference.adoc` (overview row + `[#cmd-serve]` section),
`docs/user-manual/concepts/the-flow.adoc` (one pointer sentence), `CHANGELOG.md`
(`### Added`, including the two new dependencies). No `skill/` or `AGENTS.md` marker-block
change (decision 9). `just docs-check` green.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs + CHANGELOG updated (Task 8); `pickle board audit` still 0 errors.
3. Write a **summary**: files added/touched, the htmx version + goldmark version vendored/pinned,
   the route table as shipped, the **measured** binary-size delta (`ls -l pickle` before/after —
   amendment F9), whether `audit.auditWIP` was moved onto `board.WIPCounts` or deliberately left
   alone (amendment F6), and anything deferred (e.g. `--open`, search, filtering, git-history
   mining).
4. Suggest a Conventional Commit message, ticket id in brackets at the end of the subject:

   ```
   feat(serve): add a read-only web dashboard for the board (T-053)

   pickle serve renders the board, each ticket, and a merged History
   timeline from the ticket files on 127.0.0.1:8745. Stdlib net/http +
   html/template + embedded assets; goldmark for ticket bodies, vendored
   htmx for fragment polling. No route writes anything.
   ```

5. Commit locally on `feat/T-053-serve`; **do not push or open a merge request without user
   approval**. Present the commit message, then hand back.

## Review

Reviewed 2026-07-27 on `feat/T-053-serve` (base `main`, 3 commits + 1 review commit).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b)
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

### Verification

| check | result |
|---|---|
| `just build && just test && just lint && just docs-check` | green; `snowball check` **output** read, not just its exit code (amendment F14) |
| `go test ./internal/{serve,board,ticket,cli}/...` | green; `internal/serve` at 92.4% statement coverage |
| Acceptance step 3 (live read-only smoke on this repo's real board) | `read-only: OK`; 404 / 404 / 405 as specified |
| Acceptance step 4 (usage contract) | `exit=2`; `pickle help` advertises `serve` |
| Acceptance step 5 (fresh-project smoke) | passes **once repaired** (finding N7): install → `ticket new` → `/` renders `T-001`, banner reads "board audit clean", `/t/T-001` 200 |
| Decisions 1–10 | all honoured: hash-snapshot proves nothing is written; `internal/serve` never calls `board.Parse` nor reads `BOARD.md`; `WithUnsafe` off; no `skill/` or `AGENTS.md` change; htmx reports `version:"2.0.4"` with its 0BSD text verbatim; goldmark pinned `v1.8.2` |
| Binary size (measured, release flags `-s -w`) | 2 800 178 → 9 546 626 bytes — exactly the ~2.8 → ~9.5 MB the CHANGELOG claims |
| `pickle board audit` | 53 tickets, 0 errors, 0 warnings |
| Amendment F6 | `audit.auditWIP` **was** moved onto `board.WIPCounts`; the tally now exists once for board, audit and dashboard |

Beyond the plan, and accepted: `ticket.HistoryEntries` folds wrapped continuation
lines into one logical entry (tested in `TestHistoryEntriesFoldingBoundaries`) — the
repo's own tickets wrap their reasons, so reading the first physical line only would
truncate them.

### Findings

| id | severity | disposition | finding | evidence | suggestion |
|---|---|---|---|---|---|
| N1 | non-blocking | fixed inline | `go.mod` recorded goldmark as an indirect requirement although `internal/serve/markdown.go` imports it directly — contradicting the ticket, the CHANGELOG ("two new dependencies") and the release pipeline, whose `before` hook runs `go mod tidy` and would rewrite the file mid-release | `go mod tidy` emitted exactly the `// indirect` deletion | ran `go mod tidy`; committed |
| N2 | non-blocking | fixed inline | `handler.render`'s comment said "the header is already written by then, so this can only be logged" — the code then called `http.Error` anyway | `internal/serve/serve.go:132-137` | comment rewritten to describe the actual behaviour and name the fix |
| N3 | non-blocking | noted | that `http.Error` fires after streaming has begun: log noise on any client disconnect, and a mid-stream template failure would append `render error: …` to a half-rendered page | observed live during acceptance step 3 — `http: superfluous response.WriteHeader call from …serve.(*handler).render (serve.go:136)`, triggered by the test's own `curl \| grep -q` closing the pipe | execute into a `bytes.Buffer`, then copy, so a failure yields an honest 500 |
| N4 | non-blocking | fixed inline | the short `-a` alias for `--addr` was accepted but named in no usage line, help text or manual, though the project documents its aliases (`--dry-run` (`-n`) appears in both `pickle help` and `cli-reference.adoc:227,245`) | `internal/cli/serve.go:31`; the "short flag" case in `cli_test.go` | documented in `serveUsage`, `pickle help` and the manual synopsis |
| N5 | non-blocking | noted | `TicketView.History` is built for every ticket page and rendered by no template — the body's `## History` markdown already covers it | `internal/serve/view.go:139`; no `.History` in `templates/` | drop the field, or render a structured history list beside the body |
| N6 | non-blocking | noted | both fragment routes call `newPage` → `buildHealth` → `audit.Audit` (a second `LoadAll` **plus** a full board re-render) and discard the result: neither fragment template renders the banner. Every 5s poll pays for it | `internal/serve/serve.go:184-196`; the fragment blocks in `board.html`/`activity.html` | give fragments a page value without health, or make the banner its own polled fragment |
| N7 | non-blocking | noted | acceptance step 5 cannot fail: `P2` is assigned inside the backgrounded subshell, so the parent's `curl "…:$P2/"` requests an empty port and the `grep -q` status is discarded | rerunning the step verbatim produced no output from the grep; the path was verified manually instead | assign the port in the parent shell before backgrounding |
| N8 | non-blocking | noted | plan drift, deliberate and harmless: Task 5's `fragments.html` became fragment blocks inside `board.html`/`activity.html` — better, since each block sits beside the page that embeds it | `internal/serve/templates/` listing | none; the plan's intent (page and fragment share one block) holds |
| N9 | non-blocking | noted | Task 3 asked for table tests per view builder; only `buildActivity` is called directly — `buildBoard`, `buildTicket` and `buildHealth` are covered through handler assertions | `internal/serve/serve_test.go` | acceptable at 92.4% coverage; add direct builder tests if the view layer grows |

**Disposition summary:** 9 findings, 0 blocking — 3 *fixed inline* (N1, N2, N4), 6 *noted*
(N3, N5, N6, N7, N8, N9), 0 *folded*, 0 *new tickets*.

### Impact sweep (step 8)

`T-043` patched: its Description quoted `internal/cli` coverage at ~29.5% (now 46.7%),
and `TestServeHelpIsAdvertised` is a new consumer of the defective `captureStdout`
its Part 1 item 1 warns about. No other TO DO/READY ticket depends on or references
T-053.

## History

- 2026-07-27 — created (TO DO). source: chat — request for a `serve` command rendering the
  board and its changes, with a minimal stack modelled on `rick/apps/standards`
- 2026-07-27 — TO DO → READY: plan complete; decisions confirmed (goldmark, vendored htmx, 127.0.0.1:8745, read-only)
- 2026-07-27 — applicability audit (fresh sub-agent) before pickup: 15 findings, none blocking.
  F1-F9 amended into the plan inline (history-regex index hazard, audit reads BOARD.md so
  fixtures need one, `/static/` route + root-path 404, one `sortRows` call site, shared
  `board.WIPCounts`, de-flaked acceptance smoke, measured size delta); F10-F15 noted and closed
  (goldmark pinned to v1.8.2, snowball checks at WARN, warning must say "no authentication")
- 2026-07-27 — READY → IN DEVELOPMENT: picked up; plan amended from the applicability audit (F1-F9)
- 2026-07-27 — IN DEVELOPMENT → IN REVIEW: acceptance test green; 8 tasks + 9 plan amendments done
- 2026-07-28 — IN REVIEW → DONE: review clean: 9 findings, 0 blocking — 3 fixed inline (N1 go.mod tidy, N2 render comment, N4 documented the --addr alias), 6 noted (N3, N5-N9); acceptance re-run green, read-only proven, binary 2.8->9.5 MB measured
