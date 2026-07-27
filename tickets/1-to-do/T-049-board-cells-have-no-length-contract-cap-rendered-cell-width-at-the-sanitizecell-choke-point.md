---
id: T-049
title: board cells have no length contract: cap rendered cell width at the sanitizeCell choke point
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-049 — board cells have no length contract: cap rendered cell width at the sanitizeCell choke point

## Description

**No rendered board cell has a length contract.** `sanitizeCell`
(`internal/board/board.go:113-124`) normalises three things — pipes to `¦`, newline runs to one
space, outer whitespace trimmed — and deliberately nothing else. Every cell value it receives is
emitted verbatim at whatever length the ticket happens to carry, so one long string in one ticket
makes a whole status table unreadable.

Hit in the field (2026-07-27) while migrating an 84-ticket hand-rolled flow into a fresh
`pickle install` workspace: a `6-done/` ticket whose merge note recorded pipeline ids, job names
and fast-forward reasoning produced a **~1,900-character `merged` cell**, and the DONE table with
it. Not exotic to that migration, though — the trend is already visible in this repo:

- longest merge History line in `tickets/`: **171 characters**;
- the DONE `merged` cells for T-001, T-002, T-008, T-009, T-011 and T-044 already run
  90-125 characters (`yes — MERGED: feat/… → main (…), user-approved; branch deleted`).

### Scope: cap once, at the choke point

The fix is a **render-time** cap — never a stored field, which D3 (T-044) forbids: terminal facts
live only in the ticket, and the board is regenerated from them.

It belongs in `sanitizeCell` and nowhere else. That function's own doc comment already claims the
position — *"the single one-way choke point every rendered cell passes through … Nothing ever
parses a cell back, so there is no escape scheme to keep in sync (T-044 decision 9)"* — and
capping there fixes **every** column at once. Four are unbounded today, not just `merged`:

| column | source | why unbounded |
|---|---|---|
| `merged` | `ticket.MergeLine` (`internal/ticket/ticket.go:261-280`) | returns the whole body of the newest `merged …` History line |
| `reason`, `open findings` | `ticket.LastHistoryReason` (`internal/ticket/ticket.go:220-256`) | returns the whole `: <reason>` clause of the newest transition |
| `title` | `t.Front["title"]` | frontmatter is hand-authorable; `ticket new`'s own title cap is T-038's scope, and does not constrain a hand-written ticket |

A per-column patch in `cellFor` would therefore be the wrong shape: it would put a fourth
length policy beside the one place the T-044 design centralised.

### Confirmed decisions (from the 2026-07-27 triage)

1. **Cap at ~120 runes with a trailing `…`** — enough for a full `MERGED: feat/… → main (sha),
   user-approved` line, short enough to keep a terminal-rendered table legible.
2. **Rune-safe, and applied last.** The cap runs *after* the `|` → `¦` substitution, and `¦`
   (U+00A6) is multi-byte — a byte-length slice can cut mid-rune and emit a replacement
   character into the board. Count and cut runes, not bytes.
3. **Rejected alternative:** rendering only the parenthesised ref (`MR !69, 8acc044`) instead of
   truncating. `mergedRE` is `(?i)^merged\b` (`internal/ticket/ticket.go:106`) — parentheses are
   conventional, not required — so that shape needs an undefined fallback for every merge line
   without them. Truncation has no such hole.
4. `HasMergeLine`/cell agreement is **not** a constraint worth designing around: truncating a
   non-empty string yields a non-empty string, so the yes/no boolean is preserved for any cap ≥ 1.

### Deliberately out of scope

**Detecting the malformed History line that produced the 1,900-character cell.** A merge note
that long violates `skill/resources/TEMPLATE.md:116-124` (*"One line per status transition …
OLD → NEW: one-clause reason"*), so truncation makes the board legible while leaving the ticket
malformed. That is a lint, it belongs to the audit, and it has been **folded into T-040** (the
audit-validation epic — same file `internal/audit/audit.go`, same theme: "the audit is the only
component that sees every ticket however it was authored"). This ticket is the render half only.

### Couplings

Soft couplings (no `depends-on:`, no ordering enforced):

- **T-040** — owns the History-line-shape lint folded out of this ticket. Complementary: this one
  makes the board legible, that one makes the malformation visible. Neither blocks the other.
- **T-038** — caps title length at *input* in `validateTitle`/`Slugify` (`ticket new`). Different
  boundary: it cannot constrain a hand-authored ticket file, which the flow explicitly permits, so
  the render cap is still required. Do not treat either as covering the other.
- **T-042** — touches `internal/board`; sequence, do not run concurrently.
- **T-043** — its item 5 defers to T-044's one-way sanitisation; if it adds `sanitizeCell` tests,
  land this first so the cap is part of the table.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: field finding, pickle 0.1.0 workspace migration; re-scoped at
  triage to the sanitizeCell choke point, audit-lint half folded into T-040
