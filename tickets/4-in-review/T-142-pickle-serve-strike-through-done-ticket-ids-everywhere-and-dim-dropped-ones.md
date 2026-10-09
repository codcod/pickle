---
id: T-142
title: pickle serve: strike through done ticket ids everywhere and dim dropped ones
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: medium
cost: M
---

# T-142 — pickle serve: strike through done ticket ids everywhere and dim dropped ones

## Outcome

In the `pickle serve` dashboard, every mention of a ticket in DONE renders struck through (~~T-090~~),
and every mention of a ticket in DROPPED renders dimmed. This applies on the board, the ticket page,
the activity timeline and inside rendered ticket bodies, so a reader can tell at a glance whether a
cited ticket is finished without opening it.

## Description

Today a ticket id looks the same whether it is open, done or dropped. A reader skimming a
`depends on` list, an activity line or a ticket body has to click through to learn its status.

**Scope: `internal/serve` only.** `tickets/BOARD.md` (the generated markdown) is out of scope. A
`~~T-NNN~~` form there would belong in the board generator and is a separate change.

**State classes.**
- *Done*: the state's `Columns == flow.ColumnsDone` (brine: `6-done/`). It is read off the flow
  definition, never a hard-coded directory name.
- *Dropped*: `Columns == flow.ColumnsDropped` (brine: `7-dropped/`).
- Build the sets once per request from the already-loaded tickets.

**Where ids render, and the treatment each gets:**

1. **Template-rendered ids**: `<a class="tid">` from view structs.
   - Board rows ("ticket-item", via `Entry`).
   - The `idlist` block in `layout.html` (depends on / blocks / spawned by / spawned / members).
   - Family links.
   - Ticket page title and breadcrumbs.
   - Activity timeline ids (`Event`).
   - Each gets a state class (`is-done` / `is-dropped`) and a `title` attribute naming the state,
     so the cue is not colour- or decoration-only for assistive tech.
   - CSS: `.is-done { text-decoration: line-through }`, `.is-dropped { opacity: .55 }` or the
     theme's muted token.
2. **Rows inside the DONE section are not struck.** Every row there is done by definition, and
   striking them all adds noise without information. The DROPPED section is likewise not dimmed
   wholesale. Ids *referenced from* those rows (depends-on, family…) still get their own state
   treatment.
3. **Activity event text** (`{{linkify .Text}}`), plus the board's `merged` and `reason` cells
   and the ticket page's `merged` line, which share the same free-text path. History lines cite other ids (`spawned T-091`,
   `superseded by T-120`). Ids in that text get the same treatment, applied to the
   already-escaped/linkified output without breaking URLs `linkifyURLs` wraps.
4. **Ticket bodies (and rick artifacts).**
   - `renderMarkdown` (`markdown.go`) gains a goldmark AST transformer. It walks `Text` nodes,
     matches `ticket.IDShapePattern` at word boundaries, and replaces each hit whose id is done
     or dropped with a small custom inline node. That node renders
     `<span class="tid-ref is-done" title="done">T-090</span>`, so the same CSS rules as the
     template ids apply. A `<span>` is used rather than GFM's `<del>`: `<del>` means "deleted
     content", which a done ticket is not, and one node kind covers both states.
   - Code spans (whose children are `Text` nodes, so the walk skips any `Text` under a
     `CodeSpan`), fenced/indented code (line segments, not `Text` nodes) and link destinations
     are left untouched.
   - Membership in the known-id sets is what rules out false positives such as `UTF-8`,
     `SHA-256` or `ISO-8601`: a shape match that is not a known done/dropped ticket is left as
     plain text.
   - `serve.go`'s artifact page also goes through `renderMarkdown`, so it gets the treatment.
   - Raw HTML stays disabled: the transformer adds AST nodes, never `WithUnsafe`.
   - Rejected alternative: regex over the rendered HTML. It would strike ids inside `<code>` and
     `href` attributes.

**Soft couplings.**
- `linkifyURLs` (hardened by T-090, done and merged) escapes text *between* URL runs. Item 3
  hooks into exactly that step, so ids inside a URL are never touched.
- T-104 (board layout) and T-127 (`BasePath` on every leaf view struct): the new state fields
  ride the same leaf structs (`Entry`, `Event`, `IDList`), for the same reason (`$` cannot
  cross `{{template}}`).

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/T-142-serve-done-id-strikethrough
```

Work on this branch only, committing WIP locally. `pickle` is a root-path child (`path = "."`):
tidy WIP into atomic commits before review, and keep that history by default. No push and no
merge request without explicit user approval; merging is the human's. Before pushing, run
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`. It must
print nothing.

### Prerequisite gate (hard)

none. T-090 (`linkifyURLs`), T-104 (board lanes) and T-127 (`BasePath` on leaf view structs)
are all in `6-done/` and merged, and this plan builds on their shipped shape.

### Confirmed design decisions (do not deviate without asking)

1. **Strikethrough marks DONE only; DROPPED is dimmed.** The state class comes from
   `flow.State.Columns` (`ColumnsDone` → `done`, `ColumnsDropped` → `dropped`), never from a
   directory name, so a custom flow is honoured.
2. **Scope is `internal/serve` only.** `tickets/BOARD.md` and its generator are not touched.
3. **A row's own id in the DONE or DROPPED section is not styled.** Only the ids that row
   *references* (depends on, spawned by, family) are styled. Because active lanes and TO DO
   never hold done or dropped tickets, this means a board row's own `.tid`/title never carries a
   state class.
4. **The ticket page's own id (h1 and breadcrumb) is styled.** A page opened from a link
   elsewhere should show its state at a glance, like any other mention.
5. **Activity event text, the `merged` cell/line and the board `reason` cell are styled.** They
   go through one helper, built into `linkifyURLs`'s between-URL escape step. Ids inside a URL
   are never touched.
6. **Ticket bodies and rick artifacts use a goldmark AST transformer, not HTML post-processing.**
   It emits a custom inline node rendering `<span class="tid-ref is-<state>" title="<state>">`.
   It never enables `WithUnsafe` and never emits `ast.RawHTML`.
7. **Only known done or dropped ids are styled.** A shape match (`UTF-8`, `SHA-256`) not in the
   state map is left as plain text. There is no "unknown id" styling.
8. **Every styled id carries `title="<state>"`.** The cue is then not decoration-only.
9. **State travels as one per-request `IDStates` map on the leaf view structs**, beside
   `BasePath` and for the same reason (`$` cannot cross `{{template}}`, see `Entry.BasePath`'s
   doc comment). `funcs.go` stays stateless.

### Tasks

#### Task 1 — `IDStates` and its builder (`internal/serve/view.go`)
- Add `type IDStates map[string]string` with `func (s IDStates) Of(id string) string`
  (returns `"done"`, `"dropped"` or `""`; nil-safe).
- Add `buildIDStates(def *flow.Definition, tickets []*ticket.Ticket) IDStates`. Look up each
  ticket's `def.ByDir(t.Dir)` and map `Columns` per decision 1. Other states are omitted.
- Add `States IDStates` to `Entry` and `IDList`, and `State string` to `Event` (the event's
  own id).
- `buildBoard`, `buildTicket` and `buildActivity` each call `buildIDStates` once from the
  `def` and whole-tree tickets they already receive, and thread it into `newEntry` (and
  `stateChildGroup`). Their signatures and the handlers stay unchanged; only the `artifact`
  handler builds states itself (Task 4).

#### Task 2 — templates and `idListOf` (`funcs.go`, `templates/*.html`)
- `idListOf(basePath, ids, states)` is the third argument. Update every call site in
  `board.html` and `ticket.html`.
- `layout.html` `idlist`: `{{$s := $.States.Of $id}}<a class="tid{{with $s}} is-{{.}}{{end}}"{{with $s}} title="{{.}}"{{end}} …>`.
- `board.html` "ticket-item" family link uses `.States.Of .Family` the same way. The row's own
  `.tid` and title are left unchanged (decision 3).
- `ticket.html`: the breadcrumb `<span>` and a new `<span>` wrapping only the h1's id (so the
  title is not struck) get the class and title from `.States.Of .ID`. So does the family link.
- `activity.html`: the event `.tid` gets the class and title from `.State`.

#### Task 3 — free-text ids (`view.go`)
- Split `linkifyURLs(s)` into `linkifyWith(s string, states IDStates) template.HTML`, and keep
  `linkifyURLs(s) = linkifyWith(s, nil)` so existing callers and tests are unchanged.
- Every `template.HTMLEscapeString(<non-URL piece>)` call becomes
  `escapeMarkIDs(piece, states)`. That function escapes the piece, then wraps each
  `\b` + `ticket.IDShapePattern` + `\b` match whose `states.Of` is non-empty in
  `<span class="tid-ref is-<state>" title="<state>">…</span>`. Ids are `[A-Z0-9-]`, so matching
  after escaping is safe: escaping never alters an id.
- Precompute `Event.TextHTML`, `Entry.MergedHTML` and `Entry.ReasonHTML`
  (`template.HTML`) via `linkifyWith`. Switch `activity.html`, `board.html` and `ticket.html`
  from `{{linkify .X}}`/`{{.Reason}}` to these fields. Keep the `linkify` func in `funcs.go`
  only if something still uses it; otherwise delete it.

#### Task 4 — markdown transformer (`internal/serve/markdown.go`)
- `renderMarkdown(src string, states IDStates)`. Pass the states through
  `parser.WithContext(ctx)` under a package-level `parser.NewContextKey()`, so the global `md`
  stays built once.
- Register on `md`: a `parser.WithASTTransformers` transformer that walks the document. For
  each `*ast.Text` not under an `*ast.CodeSpan`, run the id regex over
  `n.Segment.Value(source)` and split the node: plain `Text` sub-segments plus a `ticketRef`
  node (custom `ast.NodeKind`, holds the id and state) for each hit with a non-empty state. Walk
  first and mutate after, so the walk never sees nodes it inserted. The original node's
  soft/hard line-break flags move to the last piece. `Text` under an `*ast.Image` is skipped
  too: goldmark builds `alt` from `Text` leaves only, so a `ticketRef` there would vanish.
- Register a `renderer.NodeRenderer` for the kind (`renderer.WithNodeRenderers`). It writes
  the span from decision 6, with the id escaped via `util.EscapeHTML`.
- Callers: `buildTicket` passes its states. `serve.go`'s `artifact` handler moves `h.load()`
  above the render and passes `buildIDStates(...)`. The `<pre>` fallback paths stay as they
  are.
- No-op when `states` is empty: the transformer returns immediately.

#### Task 5 — CSS (`internal/serve/static/styles.css`)
- `.is-done { text-decoration: line-through; }` and `.is-dropped { opacity: 0.55; }`, next to
  `.tid`. Unqualified, so they cover `.tid`, `.tid-ref` and the family links (which carry no
  `.tid`) alike. Check the result in both light and dark themes.

#### Task 6 — tests (`internal/serve/serve_test.go`)
Add `TestDoneAndDroppedIDsAreMarked` on a `newTree` with: T-001 in `6-done` and T-002 in
`7-dropped` (each with its own History line, so each has an activity event), and T-003 in `1-to-do` with `depends: "[T-001, T-002]"`, `family: T-001`, a body
of `see T-001, T-002, T-003 and UTF-8; ` + "`T-001`" + ` in code`, and a history line
`… superseded by T-001`. Assert:
- `/t/T-003`: depends-on links carry `class="tid is-done"` / `class="tid is-dropped"` with
  matching `title`. The body contains `<span class="tid-ref is-done" title="done">T-001</span>`
  and the dropped equivalent. T-003 and `UTF-8` stay unwrapped. `<code>T-001</code>` stays
  unwrapped.
- A body line break right after a wrapped id survives (multi-line body case).
- `/t/T-001`: the `page-title` h1's id span carries `is-done` (match `page-title` itself — the
  fixture body's own `# T-001 — …` heading is also wrapped).
- `/`: the DONE-section row for T-001 has no `is-done` on its own `.tid` (decision 3). T-003's
  row's depends-on links do.
- `/activity`: the event text `superseded by T-001` wraps T-001. The T-001 event's own `.tid`
  carries `is-done`.
- A `linkifyWith` unit case: `https://x/T-001 T-001` leaves the URL's href and text unwrapped
  and wraps only the second id.

### Acceptance test

```
just build && just test && just lint && just docs-check
go test ./internal/serve -run 'TestDoneAndDroppedIDsAreMarked|TestLinkify|TestMarkdownDoesNotRenderRawHTML|TestActivity|TestTicketPage' -v
```
All green. Then run a manual smoke in a throwaway dir (self-modify policy):
`D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree`,
then create two tickets, move one to done, cite it from the other's body, and run
`./pickle-test serve`. The citing ticket's page shows the done id struck through in its body
and in `depends on`. The activity page shows the done id struck through in History text.

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc` § `pickle serve`, "What it serves": add one sentence
  after the table (it spans `/`, `/t/T-NNN`, `/activity` and the artifact pages). Every mention of a DONE ticket's id renders struck through
  and every DROPPED one dimmed, in lists, History text and rendered bodies alike. Code spans
  are left untouched.
- `CHANGELOG.md` `## [Unreleased]` → `### Added`: one bullet, ending `(T-142)`.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (cli-reference serve section, CHANGELOG).
3. Summary of files touched, decisions honoured and anything deferred.
4. Suggested commit:

   ```
   feat(serve): strike through done ticket ids and dim dropped ones (T-142)

   Every id the dashboard renders (board edges, ticket page, activity, merged/reason
   text, ticket and rick-artifact bodies) now shows its ticket's state: DONE struck
   through, DROPPED dimmed, with a title attribute naming the state.
   ```

5. Tidy WIP commits into atomic ones (root-path child).
6. Commit locally; no push or merge request. Move to `4-in-review/`, and the implement
   procedure continues into review.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-10-09 — created (TO DO). source: chat: user asked to explore, then file, strikethrough for done ticket ids across `pickle serve` (board, ticket page, activity, bodies); decisions taken in chat: done-only strikethrough, dropped dimmed, activity text included, DONE-section rows not struck, BOARD.md out of scope
- 2026-10-09 — refinement: Description re-verified against `internal/serve`. T-090 coupling corrected (it is done and merged, not in rework). Body markup changed from `<del>` to a classed `<span>`. `merged`/`reason` cells added to the free-text scope. The ticket page's own id is styled.
- 2026-10-09 — TO DO → READY: plan complete
- 2026-10-09 — READY → IN DEVELOPMENT: picked up
- 2026-10-09 — applicability gate (fresh sub-agent): 0 blocking, 8 non-blocking. 7 (board `reason` cell now linkifies bare URLs too, harmless and consistent with merged/activity) noted. plan amended inline: skip `Text` under images and carry line-break flags onto the last split piece (Task 4); id states built inside the existing builders, signatures unchanged (Task 1); h1 id gets its own span (Task 2); unqualified `.is-*` selectors so family links are covered (Task 5); done/dropped fixtures get History lines, a line-break case, `page-title`-specific h1 check (Task 6); docs sentence goes after the serve table, not under a row.
- 2026-10-09 — IN DEVELOPMENT → IN REVIEW: acceptance green
