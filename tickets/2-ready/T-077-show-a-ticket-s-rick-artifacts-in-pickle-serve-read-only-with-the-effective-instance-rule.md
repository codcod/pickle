---
id: T-077
title: show a ticket's rick artifacts in pickle serve, read-only, with the effective-instance rule
project: pickle
depends-on: [T-076]
spawned-by: []
family: T-075
impact: medium
complexity: medium
cost: M
---

# T-077 — show a ticket's rick artifacts in pickle serve, read-only, with the effective-instance rule

## Outcome

After this ships, a human reading `pickle serve`'s board sees a small “N awaiting approval”
mark on any ticket with pending rick artifacts, and opening that ticket shows every artifact
rendered as markdown — with which instance rick will actually consume flagged — instead of
reviewing a 400-line solution design in a terminal scrollback before approving it.

## Description

The reading surface, and the reason the whole family is worth building: reviewing a
400-line solution design in a terminal, before approving it, is miserable. rick's approval
gate asks a human to accept an artifact it has just rendered into a scrollback buffer.
pickle already knows how to render markdown well.

Scope — **read-only, zero writes to `docs/specs/**`**:

- a route (`GET /specs/{key}/{name}`) rendering an artifact through the existing pipeline in
  `internal/serve/markdown.go` — GFM on, `goldmark.WithUnsafe` off, so raw HTML in an
  artifact is escaped rather than executed, exactly as for ticket bodies.
  **Since T-127 this route is per-project, not top-level:** `serve` can now serve several
  project roots from one process, each mounted under `/p/{slug}/` with its own `Options`
  (`MultiHandler`, `internal/serve/serve.go`). Artifacts belong to one specific root, so this
  route must be registered on the per-root mux that `Handler` builds — where it inherits the
  `/p/{slug}/` prefix automatically — and **never** on the top-level mux, which has no way to
  tell whose `docs/specs/**` is meant. Any link to it rendered into a template must be prefixed
  `{{.BasePath}}` (empty in classic single-root mode), like every other internal link;
- on the ticket page, the ticket's artifacts listed with their `Kind` and a `Status` badge
  (`draft` / `complete` / `approved`) from T-076, so "what is this ticket waiting on" is
  visible from the board rather than only from inside the agent session.

### The effective-instance rule (the part that earns its keep)

rick's artifact paths embed a date and topic — `solution-design-2026-06-14-<topic>.md` — so
a single *kind* can accumulate several instances. `[R]evise` replaces at the same path only
within a day; across days it writes a new file. Meanwhile `rick verify` resolves which one
counts by taking the newest: filename-date descending, then mtime, then name — and
**status-blind** (`sdlc-cli/internal/verify/plan.go`). Nothing in rick surfaces that choice
to a human, so it is entirely possible to read, amend and approve an artifact that is not
the one rick will consume.

So this view must: mark which instance is **effective** per kind using rick's own rule, flag
any kind holding more than one instance, and warn when the effective instance is not the
approved one. That is brine's invariant-audit discipline (`internal/audit`) pointed at rick's
tree — a genuine contribution back, not a nicer font.

Refinement must settle whether the duplicate/effective mismatch is a warning in the UI only,
or also a `board audit` finding. The latter is tempting and probably wrong: `board audit`
asserts *brine's* invariants, and a red audit caused by another product's file layout would
violate T-075's fail-open invariant.

Soft coupling: T-104 (the board page's per-child column layout, which absorbed T-055's WIP-badge
fix) rewrote the same board templates — a ticket's row is now the shared `ticket-item` template
rendered by both the lanes and the stacked sections, so any per-ticket artifact badge this ticket
adds lands there once rather than in two places.

### Re-verified at refinement (2026-09-04)

T-076 (READY as of this refinement) shipped the library this ticket consumes:
`rickstatus.Query(root, *config.Project) rickstatus.Report`, fail-open by construction
(`Report.Available`/`Reason`), keyed by pickle ticket id via `ticket_prefix` alone — confirmed
against that ticket's Implementation Plan rather than assumed. T-127's per-root mux constraint
(quoted above) is unchanged and still binding: `Handler` builds one mux per served root, and
`MultiHandler` mounts each whole sub-handler GET-only, so a route added inside `Handler` inherits
`{{.BasePath}}` automatically — nothing extra to wire for multi-root. `internal/serve`'s shape
(`Options`, `handler`, `page`, `Entry`, `TicketView`, the shared `ticket-item` template) is also
unchanged since this ticket was filed.

**Two things this ticket's own text leaves open are settled below, at decisions 5 and 4
respectively: the audit-vs-UI-only question, and the effective-instance tie-break's exact
algorithm** — the latter carries the same caveat T-076 recorded for rick's JSON field names: this
workspace has no checkout of `sdlc-cli`, so `internal/verify/plan.go`'s "filename-date descending,
then mtime, then name" is reproduced from T-075's prose, not read from source. It is UI-only and
advisory (decision 5 makes sure of that), so a wrong guess here mis-flags which instance is
effective rather than breaking anything load-bearing — correctable later from a real capture, the
same bet T-076 already made.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the root-path child (`path = "."`):

```
git checkout main
git checkout -b feat/T-077-rick-artifacts-in-serve
```

Local WIP commits encouraged; tidy into atomic commits before presenting (root-path child, rules
§0), default to keeping that history. Do **not** push or open a merge request without explicit user
approval. Ticket/board bookkeeping stays on `main`.

### 1. Prerequisite gate (hard)

**T-076 must be in `6-done/` and merged to `main`** (`depends-on: [T-076]`) before this ticket may
enter `3-in-development/` — it is READY, not done, as of this refinement; check the board before
picking this up. Everything else this plan needs is already shipped: `internal/serve`'s
Handler/MultiHandler split (T-127), `renderMarkdown` (`internal/serve/markdown.go`),
`ticket.SplitID` (`internal/ticket/ticket.go:493`), the shared `ticket-item` template
(`internal/serve/templates/board.html`, T-104).

### 2. Confirmed design decisions (do not deviate without asking)

1. **The route is `GET /specs/{key}/{name}`, registered inside `Handler`'s mux
   (`internal/serve/serve.go`), never in `MultiHandler`'s top-level mux.** This is what makes it
   inherit `/p/{slug}/` automatically under multi-root (T-127); every link to it rendered into a
   template is prefixed `{{.BasePath}}`, like every other internal link this package renders.
2. **Whitelist-only serving: `{name}` must be the basename of a `Path` that this same request's
   fresh `rickstatus.Query(...).For(key)` actually reported — nothing else is servable, even if
   it physically exists under `specs_root/{key}/`.** This is the route's entire security model:
   rather than a general path-traversal audit over an arbitrary filename, the route only ever
   serves files rick itself just told pickle about. A `{name}` containing `/` or `..` is rejected
   before even reaching the whitelist check (belt-and-braces, since the whitelist alone already
   makes such a name miss). An unknown `{key}` shape, a `{key}` whose prefix matches no
   `rick`-enabled child, a `{name}` not in the whitelist, or a whitelisted file that has since been
   deleted underneath (rick's own `[D]iscard`, T-075 invariant 3) are all **404**, never 500 —
   serve degrades the same way a vanished ticket file already does.
3. **One `rickstatus.Query` per rick-enabled child per request, not per ticket.** The board page
   lists every ticket at once; querying rick once per row would mean one `exec.Command` per
   ticket. Instead `internal/serve/rick.go` adds `buildRickReports(cfg *config.Config, root
   string) map[string]rickstatus.Report`, called once per request (mirroring `handler.load`'s
   per-request-not-per-row shape) and indexed by project name; every builder that needs a
   ticket's artifacts looks it up via `reports[t.Project()].For(t.ID)`.
4. **Effective-instance tie-break, reproduced best-effort (see "Re-verified" above): parse the
   first `\d{4}-\d{2}-\d{2}` run in the artifact's filename as the primary descending key, fall
   back to `Artifact.Date` (T-076's field, whatever rick populates it with) as the secondary
   descending key when either date is missing or two artifacts tie, and the full `Path` string
   descending as the final tiebreak — status-blind throughout, exactly as rick's own rule is
   documented to be.** Computed per `Kind` within one ticket's artifact list; the artifact sorting
   first in this order is `Effective`.
5. **The duplicate/mismatch signal is a UI warning only — never a `board audit` finding.**
   Settles the question the original Description leaves open, for the reason it already gives:
   `board audit` asserts brine's own invariants, and a red audit caused by another product's file
   layout on disk would violate T-075's fail-open invariant. `internal/audit` is not touched by
   this ticket.
6. **Two badge granularities, not one: a compact count on the board row, the full per-kind detail
   only on the ticket page.** `Entry` (`internal/serve/view.go`) gains `RickPending int` — the
   count of this ticket's artifacts whose `Status != "approved"` — rendered as a small badge in
   the shared `ticket-item` template (T-104's soft coupling) only when `> 0`. `TicketView` gains
   `Artifacts []ArtifactView` (`Kind`, `Status`, `Name`, `Href`, `Date`, `Effective`,
   `Duplicate`, `Mismatch`), rendered in full on `ticket.html`. The row deliberately does not grow
   a second effective/duplicate taxonomy — that nuance belongs on the page where there is room to
   explain it.
7. **Status values render verbatim.** `draft`/`complete`/`approved`/anything else rick reports is
   passed straight through as the badge text; pickle never validates it against a fixed
   vocabulary.
8. **Artifact bodies render through the existing `renderMarkdown` (`markdown.go`), unchanged** —
   same GFM-on, `goldmark.WithUnsafe`-off pipeline as ticket bodies, so HTML-shaped content in an
   artifact is escaped exactly as it already is for a ticket.
9. **No htmx polling on `/specs/{key}/{name}`.** The board and activity pages poll because tickets
   move while a human watches; an artifact page does not need that, and rick's own session may
   rewrite or delete the file mid-read (T-075 invariant 3) — a stale render on reload is expected,
   not silently swapped out from under the reader.

### 3. Tasks

#### Task 1 — `internal/serve/rick.go` (new file, pure logic)

`buildRickReports` (decision 3); `ArtifactView` and `buildArtifacts(rep rickstatus.Report,
ticketID, basePath string) []ArtifactView` implementing the grouping/effective/duplicate/mismatch
rule (decisions 4–5); `resolveArtifact(cfg *config.Config, root, key, name string) (path string,
ok bool)` implementing the whitelist (decision 2) — rejects a `name` containing `/` or `..` up
front, then resolves the owning child via `ticket.SplitID(key)` + `config.Project.Prefix()`,
queries it, and checks `name` against the basenames of `Report.For(key)`.

#### Task 2 — `Entry`/`TicketView` extension (`internal/serve/view.go`)

Add `RickPending int` to `Entry`; thread a `reports map[string]rickstatus.Report` parameter
through `newEntry`, `buildBoard` and `buildTicket` so each can set it via `reports[t.Project()]`.
Add `Artifacts []ArtifactView` to `TicketView`, populated in `buildTicket` from
`buildArtifacts(reports[found.Project()], id, basePath)`.

#### Task 3 — wiring (`internal/serve/serve.go`)

`h.board` and `h.ticket` each compute `reports := buildRickReports(h.opts.Cfg, h.opts.Root)`
once and pass it into the builder calls from Task 2 (the fragment routes `boardFragment` do the
same, matching how they already mirror `h.board`). Add `mux.HandleFunc("GET /specs/{key}/{name}",
h.artifact)` to `Handler`'s mux (decision 1) and implement `h.artifact`: resolve via
`resolveArtifact`, 404 on failure, else read the file, render via `renderMarkdown`, and execute a
new `artifact.html` template.

#### Task 4 — templates and styles

- `internal/serve/templates/artifact.html` (new): breadcrumb back to `{{.BasePath}}/t/{{.TicketID}}`,
  a title, the rendered body — mirrors `ticket.html`'s shape.
- `ticket.html`: add an `artifacts` row to the `meta` `<dl>` listing each `ArtifactView` — kind,
  a status badge, an “effective” marker, and a link to the artifact route; a duplicate/mismatch
  warning line per affected kind.
- `board.html`'s `ticket-item` block: a small badge, shown only `{{if .RickPending}}`, reading
  `rick {{.RickPending}} pending`.
- `internal/serve/static/styles.css`: new classes mirroring the existing `.grade`/`.impact-*`
  pattern — `.rick-badge`, per-status modifiers, `.rick-dup`, `.rick-mismatch`.

### 4. Acceptance test

All runnable via `just test` (`go test ./...`) unless noted.

1. **`internal/serve` unit tests for `rick.go`** (`rick_test.go`, no HTTP involved): table-driven
   cases for `buildArtifacts` — a single instance is never `Duplicate`/`Mismatch` regardless of
   `Status`; two instances of one `Kind` with different filename dates → the newer is `Effective`;
   an `Effective` instance with `Status != "approved"` while `Duplicate` is true → `Mismatch`
   true; an `Effective`, approved instance → `Mismatch` false. `resolveArtifact`: a whitelisted
   name resolves; a name absent from the Query result is rejected even when the file exists on
   disk (write it directly into the fixture's `specs_root` and confirm the route still 404s); a
   name containing `/` or `..` is rejected without even calling `rickstatus.Query` (assert via a
   stub `rick` that would fail the test if invoked, as in T-076's own sentinel-file technique).
2. **HTTP tests** (`serve_test.go`, following `newHandler`/`get`/`standardTree`): a `testCfg`
   variant with one project `Rick: true` plus a stub `rick` on `PATH` returning a two-artifact,
   one-duplicate fixture — `GET /t/{id}` shows both artifacts with the right badges and exactly
   one duplicate/mismatch warning; `GET /specs/{key}/{name}` for a whitelisted name renders the
   markdown body (reuse `TestMarkdownDoesNotRenderRawHTML`'s HTML-escaping assertion against this
   route too); the same path for a non-whitelisted name is 404; a ticket belonging to a
   *non*-rick-enabled project renders its page with zero artifacts and no error (fail-open
   passthrough from T-076); `POST /specs/{key}/{name}` is 405 under classic `Handler` **and**
   under `MultiHandler` at `/p/{slug}/specs/{key}/{name}` (mirrors
   `TestMultiHandlerRoutesArePrefixed`'s shape) — confirms decision 1 registered the route
   GET-only in both mux layers.
3. **`TestServeNeverWrites` passes unmodified** — this ticket adds no writer, so the existing
   whole-tree snapshot test is the regression check, not a test to touch.
4. `just lint` and `just docs-check` clean.

### 5. Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc` (`#cmd-serve`, the routes table around line 1620): add a
  `/specs/{key}/{name}` row, and extend the `/t/T-NNN` row's description to mention the artifact
  list and the effective/duplicate badges.
- `docs/user-manual/configuration.adoc`: extend T-076's `rick`/`specs_root` bullets (added by that
  ticket) with one sentence pointing at what turning the flag on now actually surfaces, so the two
  tickets' docs land as one coherent read rather than two disconnected mentions.

### 6. Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint`/`just docs-check` clean.
2. Docs updated and registered (task 5).
3. Write a summary: files touched, the effective-instance-rule provisional caveat carried forward,
   anything deferred.
4. Suggest a Conventional Commit message, e.g. `feat(serve): show a ticket's rick artifacts,
   read-only, with the effective-instance rule (T-077)`.
5. Tidy WIP commits into atomic ones (root-path child).
6. Commit locally on the ticket branch. Publish only per the project's commit policy (do not push
   or open a merge request without user approval).

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-15 — patched by T-104's review impact sweep: soft coupling repointed from T-055
  (dropped, absorbed by T-104) to T-104's shared `ticket-item` board template.
- 2026-09-02 — patched by T-127's review impact sweep: `serve` now serves N project roots from
  one process, so the planned `GET /specs/{key}/{name}` route must be registered per-root (under
  `/p/{slug}/`, on the mux `Handler` builds) rather than top-level, and any template link to it
  must carry `{{.BasePath}}`. Scope and grade otherwise unchanged.
- 2026-09-04 — TO DO → READY: plan complete
