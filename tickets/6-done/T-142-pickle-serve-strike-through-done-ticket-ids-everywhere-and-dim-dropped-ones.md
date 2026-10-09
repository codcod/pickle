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

### Round 1 — 2026-10-09

- [x] Reviewer independence (step 0): delegated. The orchestrator wrote the branch; a fresh reviewer ran steps 1–4a (hunt: angle 7 only). Its findings were re-verified by hand (F1 reproduced; none discarded).
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned (branch had T-142 in 3-in-development); rebased onto main, re-run clean.
- [x] Implementation audit (steps 1, 2): Tasks 1–6 met in the named files, decisions 1–9 honoured; `just build && just test && just lint && just docs-check` green (actionlint/shellcheck absent locally, CI runs them); the named `-run` subset green.
- [x] Correctness hunt (step 3): host tool path, `code-review` (high) run by the orchestrator on `main...feat/T-142-serve-done-id-strikethrough`, 8 findings, 0 discarded on re-verification (the mid-token match reproduced); angle 7 run separately by the delegated reviewer over 30 hostile inputs, no finding.
- [x] Consistency audit (step 4)
- [x] Documentation audit (step 4a): cli-reference sentence after the serve table + CHANGELOG bullet; `just docs-check` green; F1 makes the docs' "URLs are left untouched" claim false.
- [x] Docs-readability pass (step 4b): skipped — no docs-readability reviewer configured in this session.
- [x] Findings recorded (step 5)
- [x] Ticket moved to `5-rework/` (step 6)

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | In a ticket or artifact body, an id inside a bare URL that GFM linkify does not autolink (dotless host) is styled, contradicting the shipped docs ("Ids inside … URLs are left untouched"); the free-text path leaves the same URL alone. | `renderMarkdown("http://localhost:8080/t/T-001", {T-001: done})` → `<p>http://localhost:8080/t/<span class="tid-ref is-done" …>T-001</span></p>` (overlay probe) | Exclude hits inside a whitespace-delimited run that holds a URL scheme, with one hit rule shared by both paths. |
| F2 | blocking | correctness | — | An id heading a longer hyphenated token (branch name, ticket filename) has just its id part styled, mid-word, in both paths. | `linkifyWith("merged feat/T-001-config-registry", …)` → `feat/<span …>T-001</span>-config-registry`; markdown `see T-001-config-registry.md` likewise (overlay probe) | Reject a hit followed by `-` + alphanumeric. |
| F3 | blocking | correctness | — | The markdown transformer applies `\b` per goldmark Text node, which goldmark splits at delimiters such as `_`, so `snake_T-001` is styled on the ticket page but not in /activity. | code-review probe: `snake_T-001` → `snake_<span …>T-001</span>` via renderMarkdown, plain via linkifyWith | Compute hits once over the whole stripped source, and accept a Text-node match only if it is one of them. |
| F4 | non-blocking | test-gap | fixed inline | The artifact page's id styling is untested; passing nil states there would keep every test green. | `TestDoneAndDroppedIDsAreMarked` covers `/t/`, `/`, `/activity` only | Assert a styled id on an artifact page. |
| F5 | non-blocking | design | fixed inline | `buildActivity` builds `TextHTML` for every History line before truncating to `activityCap`. | `view.go` `buildActivity` | Fill `TextHTML` after the cap. |
| F6 | non-blocking | stale-xref | fixed inline | `linkifyURLs`'s doc comment still calls it the shared path the three views use; production now calls `linkifyWith`, and the wrapper is test-only. | `view.go` `linkifyURLs` | Move the rationale to `linkifyWith`; mark the wrapper test-only. |
| F7 | non-blocking | design | fixed inline | `Event.Text` is now dead: `activity.html` reads `TextHTML`. | grep over `internal/serve` | Delete it. |
| F8 | non-blocking | design | noted | `Entry.Merged`/`Reason` survive only as `{{if}}` guards beside their HTML twins. | `board.html`, `ticket.html` | — (the raw fields are the readable values; cheap to keep) |
| F9 | non-blocking | design | noted | The `{{with $s}} class="is-…" title="…"{{end}}` snippet is repeated in six template places plus `idRefSpan`. | `templates/{layout,board,ticket,activity}.html` | — (a shared block is awkward in attribute context; small) |
| F10 | non-blocking | design | noted | `.is-dropped` opacity stacks on the already-muted `.edge.muted` spans, and `title` is not reliably announced by screen readers, so decision 8's cue is weaker than its intent. | `styles.css` | — (T-142 decision 8 chose `title`; revisit with a contrast pass if it reads poorly) |
| F11 | non-blocking | design | noted | A styled id inside a markdown link puts the span's `title="done"` over the link's own title on hover. | `markdown.go` renderer | — |

Dispositions: 4 fixed inline (F4–F7), 4 noted (F8–F11), 0 folded, 0 new tickets; 3 blocking (F1–F3) → rework.
cost: estimated M, actual M

### Rework fix record — round 1 (commits 366a7c1..874ae39)

Recorded first as `4f9cd3e..fafda75`. A rebase onto main, which picked up the ticket move, rewrote those commits to the range above; the diff is byte-identical (round 2, R4).

- F1–F3 (`44499c1`): `idRefHits` in `view.go` is now the one hit rule for both paths. It skips an id inside a whitespace-delimited run that holds a URL scheme, and an id followed by `-` + alphanumeric. The markdown transformer computes hits once over the whole stripped source, and only splits a Text node at a match that is one of them, so `\b` and the URL check see the same context `linkifyWith` does. Regression cases added to `TestDoneAndDroppedIDsAreMarked`: a dotless bare URL, `feat/T-001-slug` and `snake_T-001` stay plain in the body, and `linkifyWith` agrees. A mutation that dropped the whole-source filter failed the test.
- F4–F7 (`fafda75`, fixed inline): `TestArtifactPageMarksDoneIDs`; `TextHTML` built after the activity cap from an unexported `text` field (the exported `Event.Text` is gone); linkify's rationale moved onto `linkifyWith`, with `linkifyURLs` documented as the test-pinned no-marking form.
- `just build`, `just test`, `just lint` and `just docs-check` are green.


### Round 2 — 2026-10-09 (scoped re-review of F1–F7 and `366a7c1..874ae39`)

- [x] Reviewer independence (step 0): delegated to a fresh reviewer (hunt: one pass over this round's fix diff). Its findings were re-verified by hand (R2 reproduced: 120 KB of `T-001,` took 3.2 s in `linkifyWith`); none discarded.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` clean on the branch.
- [x] Implementation audit (steps 1, 2): F1–F3 closed; mutations disabling the URL-run, hyphen or whole-source filters each fail `TestDoneAndDroppedIDsAreMarked`. F4–F7 hold. The four configured commands are green, and the reviewer re-ran the manual smoke in a throwaway `pickle-test` dir.
- [x] Correctness hunt (step 3): one pass over all eight angles on the fix diff, by the delegated reviewer; angle 7 no finding.
- [x] Consistency audit (step 4)
- [x] Documentation audit (step 4a): the URL claim narrowed to http(s) (R1); `just docs-check` green.
- [x] Docs-readability pass (step 4b): skipped — no docs-readability reviewer configured in this session.
- [x] Findings recorded (step 5)
- [x] Ticket moved to `6-done/` (step 6)
- [x] Other references (step 7): no governing document describes serve's id rendering. `BOARD.md` regenerated by the move.
- [x] Impact sweep (step 8): no open ticket depends on or cites T-142. T-078 and T-079 touch `pickle serve`, but not the rendering paths changed here.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | non-blocking | spec-unclear | fixed inline | The docs and CHANGELOG say ids in "URLs" are left untouched. Only http(s) runs are recognised: `ftp://…/T-001` and `mailto:T-001@…` are styled, and `www.x.com/T-001` is styled in free text, though not in bodies (GFM autolinks it). | overlay probe (delegated reviewer) | Docs and CHANGELOG now say "http(s) URLs" (`7080123`). Recognising more URL shapes is a behaviour change, so it is left as is. |
| R2 | non-blocking | design | fixed inline | `idRefHits` rescanned back to the last whitespace for every id: quadratic on a long whitespace-free run. | 120 KB of `T-001,` → 3.2 s (`linkifyWith`, re-verified); spaced input is fast | URL spans found once up front, then a moving pointer (`7080123`): the same input takes 9 ms with identical output, and a mutation disabling the span check fails the test. |
| R3 | non-blocking | correctness | noted | The mid-token rule is one-sided: `pre-T-001` and `docs/specs/T-001/x.md` are styled mid-token, and in `T-001-T-002` only the second id is. | overlay probe | — (no `x/T-NNN/` path appears in this repo's tickets; promote if it reads badly in practice) |
| R4 | non-blocking | stale-xref | fixed inline | The round-1 fix record cited SHAs that the rebase rewrote. Task 3's prose describes matching after escaping, but the code now matches raw text through `idRefHits`. | `git branch --contains 44499c1` is empty | The record's range is corrected above. The plan is left as the record of intent; the rule as it shipped is described here, in the round-1 F1–F3 row. |
| R5 | non-blocking | stale-xref | fixed inline | `urlSchemeRE`'s comment still said `linkifyURLs` measures each run. | `view.go` | Now names `linkifyWith` (`7080123`). |

Dispositions: 4 fixed inline (R1, R2, R4, R5), 1 noted (R3), 0 folded, 0 new tickets; 0 blocking.
cost: estimated M, actual M

## History

- 2026-10-09 — created (TO DO). source: chat: user asked to explore, then file, strikethrough for done ticket ids across `pickle serve` (board, ticket page, activity, bodies); decisions taken in chat: done-only strikethrough, dropped dimmed, activity text included, DONE-section rows not struck, BOARD.md out of scope
- 2026-10-09 — refinement: Description re-verified against `internal/serve`. T-090 coupling corrected (it is done and merged, not in rework). Body markup changed from `<del>` to a classed `<span>`. `merged`/`reason` cells added to the free-text scope. The ticket page's own id is styled.
- 2026-10-09 — TO DO → READY: plan complete
- 2026-10-09 — READY → IN DEVELOPMENT: picked up
- 2026-10-09 — applicability gate (fresh sub-agent): 0 blocking, 8 non-blocking. 7 (board `reason` cell now linkifies bare URLs too, harmless and consistent with merged/activity) noted. plan amended inline: skip `Text` under images and carry line-break flags onto the last split piece (Task 4); id states built inside the existing builders, signatures unchanged (Task 1); h1 id gets its own span (Task 2); unqualified `.is-*` selectors so family links are covered (Task 5); done/dropped fixtures get History lines, a line-break case, `page-title`-specific h1 check (Task 6); docs sentence goes after the serve table, not under a row.
- 2026-10-09 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-10-09 — IN REVIEW → REWORK: 3 blocking findings (F1–F3): id hits inside bare URLs, mid-token and per-Text-node boundaries
- 2026-10-09 — REWORK → IN REVIEW: findings fixed
- 2026-10-09 — IN REVIEW → DONE: review clean after 1 rework round; round 1: 3 blocking fixed, 4 fixed inline, 4 noted; round 2: 4 fixed inline, 1 noted, 0 new tickets
