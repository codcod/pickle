---
id: T-142
title: pickle serve: strike through done ticket ids everywhere and dim dropped ones
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: medium
cost: S
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
3. **Activity event text** (`{{linkify .Text}}`). History lines cite other ids (`spawned T-091`,
   `superseded by T-120`). Ids in that text get the same treatment, applied to the
   already-escaped/linkified output without breaking URLs `linkifyURLs` wraps.
4. **Ticket bodies (and rick artifacts).**
   - `renderMarkdown` (`markdown.go`) gains a goldmark AST transformer. It walks `Text` nodes,
     matches `ticket.IDShapePattern` at word boundaries, and wraps each hit whose id is in the
     done set in the GFM `Strikethrough` node (renders `<del>`). Dropped hits get a dim span.
   - Code spans, fenced/indented code and link destinations are distinct node types and are
     left untouched.
   - Membership in the known-id sets is what rules out false positives such as `UTF-8`,
     `SHA-256` or `ISO-8601`: a shape match that is not a known done/dropped ticket is left as
     plain text.
   - `serve.go`'s artifact page also goes through `renderMarkdown`, so it gets the treatment.
   - Raw HTML stays disabled: the transformer adds AST nodes, never `WithUnsafe`.
   - Rejected alternative: regex over the rendered HTML. It would strike ids inside `<code>` and
     `href` attributes.

**Soft couplings.**
- T-090 (in rework) hardens `linkifyURLs`. Activity-text treatment (item 3) composes with its
  output and must be rebased onto whatever T-090 lands.
- T-104 (board layout) and T-127 (`BasePath` on every leaf view struct): the new state fields
  ride the same leaf structs (`Entry`, `Event`, `IDList`), for the same reason (`$` cannot
  cross `{{template}}`).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-10-09 — created (TO DO). source: chat: user asked to explore, then file, strikethrough for done ticket ids across `pickle serve` (board, ticket page, activity, bodies); decisions taken in chat: done-only strikethrough, dropped dimmed, activity text included, DONE-section rows not struck, BOARD.md out of scope
