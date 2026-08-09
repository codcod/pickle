---
id: T-089
title: Record a commit reference alongside the merge History line
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: M
---

# T-089 — Record a commit reference alongside the merge History line

## Outcome

After this ships, anyone reading a done ticket's `## History` can go straight from its merge
line to the commit that actually shipped the change: the existing `merged to <base> (<MR
ref>)` line also carries a commit reference (short SHA, and — where the child's remote
resolves to a known hosting URL — a link to it), instead of the ticket needing a separate
section or a copy of anything.

## Description

Origin: a chat request to have every ticket carry (a) a commit id/link for its changes and
(b) a copy of the line that landed in `CHANGELOG.md`. Both were challenged before filing;
only a scoped-down version of (a) survives here.

**Why (b) is dropped, not deferred.** Copying a `CHANGELOG.md` line into the ticket creates a
second copy of a fact that already drifts on its own: T-073's review (finding F1) records that
several entries (T-058/T-059/T-061) were written by a human at release-cut time, well after
those tickets were archived in `6-done/`, and plenty of tickets ship with no `CHANGELOG.md`
line at all (T-019, and the internal-fix precedent discussed in T-043's rework). A `6-done/`
ticket is never revisited, so a pasted-in copy has no mechanism keeping it in sync with the one
authoritative line in `CHANGELOG.md` — precisely the failure mode the flow already guards
against elsewhere by making `BOARD.md` generated rather than hand-duplicated. Every
`CHANGELOG.md` bullet already ends in `(T-NNN)`, so the ticket → changelog lookup already works
in the direction that can't drift (`grep T-NNN CHANGELOG.md`); nothing here needs to change.

**Why (a) survives, scoped to the existing merge line.** The flow already has a place this
fact belongs: the `merged to <base> (<MR ref>)` History line (TEMPLATE.md's `## History`
section; `internal/ticket.MergeLine`), written once, by the human, at the one point a merge is
actually final. `historyKind`/`MergeLine` already treat the whole line as free text — nothing
parses the parenthetical today beyond the line starting with "merged" — so recommending a
commit reference alongside the MR ref (e.g. `merged to main (MR !12, a1b2c3d)`) is a convention
change to `tickets/README.md`'s history-line rules, `TEMPLATE.md`'s worked example, and
`review-protocol.md`'s finish-step guidance, not a new ticket section or a new parser.

**Decided at refinement:** `pickle serve`'s board "merged" cell, the ticket page's `merged`
summary line, and the activity timeline's per-entry text render as plain escaped strings (not
through the markdown renderer), so a URL pasted into the merge line is inert text in exactly
the three views a human would look at it from — only a URL inside a ticket's `## Description`/
body gets a free clickable link today, via goldmark's GFM `Linkify` extension. Bundled into this
ticket rather than deferred: a small, tested helper that linkifies a URL in those three spots, so
the convention change actually produces a clickable link somewhere, not just a documented habit.
Re-graded impact `low-medium → medium` accordingly (a real, tested dashboard capability, not only
a docs sync) — complexity stays `low` (mechanical, no parsing changes); cost collapsed
`S-M → M` (docs sync across five files plus one new tested Go helper). Deliberately **not**
extended to board.html's `.Reason` span (DROPPED/REWORK cell) — a distinct surface from the merge
line this ticket targets; broadening it there is a follow-up, not blocking this Outcome.

Also found while re-reading the current convention: `internal/ticket.MergeLine`'s own godoc
example (`merged to main (abc1234)`) already disagrees with `TEMPLATE.md`'s worked example
(`merged to main (MR !12)`) — one shows a bare commit hash, the other an MR ref, and neither
shows both. Fixing that agreement is folded into Task 1 below rather than spun into its own
ticket — it's the same one-line change this ticket is already making.

Soft coupling: T-070 (routes `MergeLine` through `HistoryEntries`) touches the same parsing
path. No hard dependency, but read it first so the two changes don't collide.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .            # pickle is the root-path child (path = ".")
git checkout main
git checkout -b feat/T-089-merge-line-commit-reference
```

Commit locally as you go. Publish only per the project's commit policy (never push/open an MR
without approval); default to keeping the tidied history on merge rather than squashing
(root-path child default). Before pushing, verify the remote base is not behind local
(`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
print nothing).

### Prerequisite gate (hard)

None — no `depends-on:`. Soft coupling only: skim T-089's Description note on T-070 (also
touches `internal/ticket.MergeLine`'s neighbourhood) before editing `ticket.go`'s doc comment,
so the two don't collide if T-070 lands first.

### Confirmed design decisions (do not deviate without asking)

1. **No new frontmatter, no new ticket section, no parser change.** The merge History line
   stays free text; `historyKind`/`MergeLine` (`internal/ticket/ticket.go`) are unchanged.
2. **Recommended convention** (not enforced): `merged to <base> (<MR ref>, <commit>)`, where
   `<commit>` is the short SHA and, when the remote resolves to a known hosting URL, a full
   commit link — e.g. `merged to main (MR !12, a1b2c3d)` or
   `merged to main (MR !12, a1b2c3d, https://github.com/org/repo/commit/a1b2c3d1234...)`.
   Recorded as guidance in prose, never as a machine-checked shape (rules §1's 400-rune warning
   already covers pathological outliers; these examples run well under it).
3. **`CHANGELOG.md` duplication is explicitly out of scope** (see Description) — do not add
   anything to `resources/tickets-README.md`/`TEMPLATE.md` that tells anyone to copy a
   changelog line into a ticket.
4. **`pickle serve` linkification is generic (`linkify(s) → template.HTML`), not merge-specific**
   — one helper applied to `Entry.Merged` (board + ticket views) and to each activity
   `Event.Text`, matching the codebase's existing preference for one shared implementation over
   per-view special cases. `board.html`'s `.Reason` span is deliberately left untouched (see
   Description).
5. **The helper HTML-escapes everything itself** (`template.HTMLEscapeString` semantics) and
   only wraps a matched `https?://` run in an anchor — it must not weaken
   `TestEscapingIsTheTemplatesJob`'s guarantee that markup in ticket-derived text is always
   escaped, and must not depend on `goldmark`/`WithUnsafe`.

### Tasks

#### Task 1 — Sync the merge-line convention across the skill payload

Edit (all three, one coherent pass — the flow's own single source of truth for the History
section):

- `skill/resources/tickets-README.md` §1 (the `"YYYY-MM-DD — merged to <base> (<MR ref>)"`
  sentence): broaden to `"merged to <base> (<MR ref>[, <commit>])"` and add one clause
  recommending the commit reference for direct traceability from ticket to shipped commit.
- `skill/resources/tickets-README.md` §3 ("append a dated `merged to <base>` History line to
  the done ticket"): add the same one-clause recommendation.
- `skill/resources/TEMPLATE.md`'s `## History` docstring and its worked example
  (`- 2026-07-17 — merged to main (MR !12)` → `- 2026-07-17 — merged to main (MR !12, a1b2c3d)`).
- `internal/ticket/ticket.go`'s `MergeLine` godoc comment: align its example
  (`"merged to main (abc1234)"`) with the same convention so the code comment and the template
  no longer show two different, disagreeing examples — e.g.
  `"merged to main (MR !12, abc1234)"`. Comment-only; no behavioural change.

(`.agents/skills/ticket-flow/` is a symlink to `skill/` — either path edits the same file; edit
via `skill/resources/…` per AGENTS.md.)

#### Task 2 — Sync the shipped user manual

Same recommendation, mirrored into the docs a `pickle` user (not this repo's own ticket-flow
reader) reads:

- `docs/user-manual/concepts/lifecycle.adoc` ("== Done ≠ merged" section: "append a dated
  `merged to <base> (<MR ref>)` History line").
- `docs/user-manual/your-first-project.adoc` ("== 7. Merge it — yours": same sentence).

Leave `docs/user-manual/cli-reference.adoc`'s `[#cmd-board-audit]`/`[#cmd-board-sync]` sections
alone — they already cite the shape generically (`merged to <base> (<ref>)`), which still holds.

#### Task 3 — `pickle serve`: linkify a URL in merge/history text

- `internal/serve/view.go`: add `linkifyURLs(s string) template.HTML`. Escape `s` with
  `template.HTMLEscapeString`, then find non-overlapping `https?://\S+` runs (a small regexp is
  fine — no need for `net/url` parsing), trim trailing `)`, `,`, `.`, `;` from each match (so
  `(MR !12, https://…/commit/abc123)` doesn't swallow the closing paren into the href), and wrap
  each surviving match in `<a href="…" rel="noopener" target="_blank">…</a>` (href value passed
  through `template.HTMLEscapeString` too — it is already escaped text at that point, so this is
  belt-and-suspenders, not double work).
- `internal/serve/funcs.go`: register it, `"linkify": linkifyURLs,` in `funcs` (logic stays in
  `view.go` per that file's own stated split).
- `internal/serve/templates/board.html`: `<span class="merged">{{.Merged}}</span>` →
  `<span class="merged">{{linkify .Merged}}</span>`.
- `internal/serve/templates/ticket.html`: `<dd>{{.Merged}}</dd>` → `<dd>{{linkify .Merged}}</dd>`.
- `internal/serve/templates/activity.html`: `<span class="event-text">{{.Text}}</span>` →
  `<span class="event-text">{{linkify .Text}}</span>`.
- Tests in `internal/serve/serve_test.go` (or a new `linkify_test.go` in the same package):
  - a merge line with a bare URL renders as a clickable `<a href="…">` on `/`, `/t/T-NNN` and
    `/activity` (reuse the `fixture`/`newTree` helpers already in the file, a `history` entry
    like `"- 2026-07-20 — merged to main (MR !1, https://example.com/commit/abc123)"`);
  - a merge line with no URL renders unchanged (still escaped, no stray anchor);
  - `<script>`/`&` in the surrounding text stays escaped when a URL is also present (guards
    against `linkifyURLs` accidentally un-escaping neighbouring text);
  - a URL immediately followed by `)` or `,` does not pull the punctuation into the `href`.

### Acceptance test

1. `just build && just test && just lint` — all green, including the new `internal/serve` tests
   from Task 3.
2. `just docs-check` — green after the Task 2 edits (no broken cross-references).
3. Manual smoke: in a throwaway tickets tree (`internal/serve`'s existing `newTree` test helper,
   or `D=$(mktemp -d) && cp pickle "$D/pk"` per AGENTS.md's self-modify policy if exercised via
   the built binary instead of `go test`), give a `6-done/` ticket a History line
   `merged to main (MR !1, https://example.com/commit/abc123)` and confirm `pickle serve` renders
   a clickable `<a href="https://example.com/commit/abc123">` on the board, that ticket's page,
   and `/activity`.

### Docs update (mandatory when user-facing)

`skill/resources/tickets-README.md`, `skill/resources/TEMPLATE.md`,
`docs/user-manual/concepts/lifecycle.adoc`, `docs/user-manual/your-first-project.adoc` — all
covered by Tasks 1–2 above. `CHANGELOG.md`: one `### Changed` entry under `## [Unreleased]`
(the merge-line convention changes, and `pickle serve` gains real behaviour — a rendered link —
so this clears the user-facing bar the way T-054/T-057 did).

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint`/`just docs-check` clean.
2. Docs updated and registered (Task 2 + CHANGELOG entry above).
3. Write the summary (files touched per task, the godoc-example fix folded into Task 1, the
   `.Reason`-span boundary decision).
4. Suggest a Conventional Commit message, ticket id in brackets, e.g.
   `docs(tickets): recommend a commit reference on the merge History line (T-089)` — or split
   into a `docs(tickets): …` + `feat(serve): linkify merge/history URLs (T-089)` pair if the
   root-path child's "keep tidied history" default (step 0) makes two atomic commits clearer
   than one broad one; either is fine, human's call at approval.
5. Commit locally on the ticket branch; publish only per the commit policy (approval required).
   `pickle ticket move T-089 in-review --reason "acceptance green"` and hand back.

## Review

Reviewed 2026-08-09 on `feat/T-089-merge-line-commit-reference` (commit `820142b`), ticket read
from `main` per the protocol. The implementer authored this ticket, so the implementation,
quality, consistency and docs audits were also run by an independent sub-agent briefed to be
adversarial; the escaping behaviour and the board-cap arithmetic were then re-verified
first-hand (a throwaway probe of `linkifyURLs` over 13 hostile inputs, and a real `board sync`
render of the GitLab-subgroup merge line).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [ ] Docs-readability pass (step 4b) — **skipped: reviewer unavailable.** The host's
      `docs_readability` tool failed to start (`gemini-2.5-pro` not available on the configured
      `github-copilot` provider — `model_not_supported`). A sanctioned conscious skip; its
      suggestions are polish and never findings.
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented; bookkeeping committed (step 9)

### Step 2 — implementation audit

| item | verdict | evidence |
|---|---|---|
| Task 1 — convention in `tickets-README.md` §1/§3, `TEMPLATE.md`, `MergeLine` godoc | met | `820142b` touches all four sites; `MergeLine`'s example now agrees with `TEMPLATE.md`'s |
| Task 2 — `lifecycle.adoc`, `your-first-project.adoc`; `cli-reference.adoc` left alone | met | both edited; `cli-reference.adoc:489,541,599` verified genuinely generic (`merged to <base> (<ref>)`), so leaving it was correct |
| Task 3 — `linkifyURLs` + `funcs.go` registration + 3 templates, `.Reason` untouched | met | `view.go`, `funcs.go`, `board.html:47`, `ticket.html:19`, `activity.html:27`; `.Reason` span unchanged |
| Decisions 1–5 (no parser change; prose-only recommendation; no CHANGELOG-copy guidance; generic helper on 3 surfaces; self-escaping, goldmark-independent) | met (5/5) | `historyKind`/`mergedRE` unchanged; no machine-checked shape added; helper uses `template.HTMLEscapeString`, no `WithUnsafe` |
| Acceptance 1 — `just build`/`test`/`lint` green | met | re-run verbatim: all `ok`, `go vet` silent, `gofmt` clean |
| Acceptance 2 — `just docs-check` green | met | `snowball check` clean, exit 0 |
| Acceptance 3 — manual smoke of the three views | met, but not evidenced in the record | re-done during review (throwaway tree + `pk serve`): anchor present on `/`, `/t/T-001`, `/activity`. See F12's neighbour note — the plan asked for a manual step that leaves no artifact; `TestMergeLineLinkification` is the durable equivalent |

### Findings

| id | severity | disposition | description | evidence | suggestion / resolution |
|---|---|---|---|---|---|
| F1 | non-blocking | new ticket (T-090) | Escape-then-trim ordering corrupts entity tails. `;` is in the trim set and every escaped HTML-special character ends in `;`, so a URL whose tail is an entity loses its terminator. Not exploitable: attribute delimiters are fixed by the tokenizer before character references decode, and `javascript:`/`data:` never match the pattern. But the rendered URL is wrong. | probe of 13 inputs: `https://x.com/<script>` yields href `https://x.com/&lt;script&gt` with a stray `;` outside the anchor | Trim the raw string and escape each part, or drop `;` from the trim set. Batched into T-090. |
| F2 | non-blocking | fixed inline | The escaping note and the `//nolint:gosec` justification both claimed only an "already-escaped substring" reaches the href. F1 shows the trim can leave a *truncated* entity there, so this branch's own prose was false. | `internal/serve/view.go`, the doc comment and the nolint line as authored in `820142b` | Done in `82603bd`: the note now states the trim's effect, why it is still not a breach, and points at T-090. |
| F3 | non-blocking | new ticket (T-090) | `if trimmed == ""` is dead code: the pattern requires `https?://` and `/` is not in the trim set, so the value can never be empty. | `internal/serve/view.go` `linkifyURLs`; confirmed by reasoning about the cutset and by probe | Delete, or make it the guard F4 needs. Batched into T-090. |
| F4 | non-blocking | new ticket (T-090) | A host-less URL still becomes a link: `https://).` renders an anchor whose href is `https://`, a link to nothing. | probe: input `https://).` yields `<a href="https://">https://</a>).` | Require at least one character of host. Batched into T-090. |
| F5 | non-blocking | new ticket (T-090) | Two comma-separated URLs collapse into a single anchor whose href is both, because `\S+` is greedy and only the tail is trimmed. | probe: `https://a.com/1,https://b.com/2` yields one anchor | Batched into T-090. |
| F6 | non-blocking | new ticket (T-090) | Test gap: nothing asserts escaping *inside* a URL run, the one place the helper is non-obvious and exactly what would have caught F1. The suite is not weak (removing escaping entirely fails `TestLinkifyURLsEscapesSurroundingText`), just narrow. | `internal/serve/serve_test.go`, the three tests added in `820142b` | Add cases for `&`, `<`, host-less and adjacent URLs. Batched into T-090. |
| F7 | non-blocking | new ticket (T-090) | `rel="noreferrer"` absent, so following a link leaks `Referer: http://127.0.0.1:<port>/t/T-NNN` to the external host. `target="_blank"` itself is fine for a dashboard. | the anchor string in `linkifyURLs` | Use `rel="noopener noreferrer"`. Batched into T-090. |
| F8 | non-blocking | fixed inline | `maxCellRunes`'s comment claimed 120 was chosen so "nothing legitimate is clipped" and a full merge cell "survives intact". This branch's own recommended merge line falsifies both. | `internal/board/board.go`; measured: GitHub full form is exactly 120 runes, GitLab subgroup form 122 | Done in `82603bd`: the comment records the new ceiling and why clipping is by design here. |
| F9 | non-blocking | fixed inline | The new recommendation never mentioned the board cap it can exceed, so a reader following it can produce a merge line whose URL is clipped to a dead string in `BOARD.md` while `pickle serve` shows it live. | verified end to end: a `6-done/` fixture with the GitLab subgroup form renders an ellipsis-truncated `merged` cell after `board sync` | Done in `82603bd`: rules §1 and `lifecycle.adoc` now say to keep the short SHA alongside the link, because the SHA is what still reads in the capped cell. Deliberately not repeated in `TEMPLATE.md`/`your-first-project.adoc` to avoid four-way duplication of a render-time detail. |
| F10 | non-blocking | fixed inline | The CHANGELOG entry's headline promised that a commit reference "renders as a clickable link" while its only example, a bare short SHA, produces none; it also credited `tickets/README.md`, which is a pointer file carrying no such recommendation. | `CHANGELOG.md` as authored in `820142b`; `tickets/README.md` read in full, no merge-line text | Done in `82603bd`: reworded, and the changed files named accurately. |
| F11 | non-blocking | fixed inline | Added-vs-Changed: the serve linkification is new behaviour but shipped inside a single `### Changed` bullet. T-068's finding R8 set this repo's precedent, keeping new capability under `### Added` and putting only changed behaviour under `### Changed`. The plan itself mandated one `Changed` entry, so this is a plan-level miss rather than a deviation from it. | `tickets/6-done/T-068-*.md:545`; `CHANGELOG.md` | Done in `82603bd`: split into `### Added` (linkify) + `### Changed` (convention). |
| F12 | non-blocking | noted | The Description lists `review-protocol.md` among the convention's targets, but the Tasks dropped it and it was not touched. Verified harmless: that file carries no `merged to <base>` text, so there was nothing to change. The Description's claim is simply unfulfilled. | `skill/resources/review-protocol.md`, no match for the merge-line form | Closed with evidence. Relatedly, acceptance item 3 asked for a manual smoke test, which by nature leaves no artifact in the record; `TestMergeLineLinkification` is the durable equivalent and future plans should prefer it. |

**Disposition summary.** 12 findings, **0 blocking**, 12 non-blocking: **5 fixed inline**
(F2, F8, F9, F10, F11 — all in `82603bd`), **6 → one new ticket** (F1, F3, F4, F5, F6, F7 →
**T-090**, batched by theme: `linkifyURLs` robustness), **1 noted** (F12).

Discarded, not counted as a finding: the independent audit flagged an untracked
`internal/serve/zz_tmp_probe_test.go` as debris. That was this review's own throwaway probe of
`linkifyURLs`, created and deleted during the audit; verified gone. Its own claim that
`tickets/README.md` does not exist was also wrong — the file exists and was read in full (it is
a pointer, so its conclusion that no second copy of the rules needed updating still holds).

**Verdict: PASS.** No blocking findings. The golden path the ticket set out to serve — a commit
URL on a merge line, clickable in all three `pickle serve` views — is exact, and every
confirmed design decision was honoured. Security was the one place worth attacking and it holds:
escape-before-match plus a scheme allowlist leaves no XSS or attribute-injection reachable.
Proceeds to `6-done/`.

## History

- 2026-08-09 — created (TO DO). source: chat — commit-id/changelog-line idea challenged;
  scoped down to enriching the merge History line only, CHANGELOG-copy half dropped
- 2026-08-09 — TO DO → READY: implementation plan complete
- 2026-08-09 — READY → IN DEVELOPMENT: picked up
- 2026-08-09 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-09 — IN REVIEW → DONE: review PASS; 0 blocking, 12 non-blocking all dispositioned (5 fixed inline, 6 -> T-090, 1 noted)
