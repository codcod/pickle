---
id: T-047
title: AsciiDoc user manual in docs/ + slim README to about & install
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: M
---

# T-047 — AsciiDoc user manual in docs/ + slim README to about & install

## Description

Today everything a user can learn about `pickle` lives in one 378-line `README.md`: what the
tool is, install instructions, the design principle, the full command reference (one `##`
section per command), the configuration schema, and build-from-source notes. That single file
is both the shop window and the manual, which makes it long, hard to navigate, and prone to
the accuracy drift T-019 catalogues.

Split the two roles:

1. **`README.md` becomes the shop window only** — what this project is about (the
   ticket-based, board-driven flow; the split-judgment-from-mechanics principle in one or two
   paragraphs) and how to install it (binary + build-from-source). Everything else moves to
   `docs/` and the README links to the manual. No content should exist in both places.

2. **`docs/` gains an AsciiDoc user manual**, following the structural patterns of the
   `~/Projects/personal/translator/ai-sdlc/docs/` book tree (the reference layout, not its
   content):
   - a master document `docs/user-manual.adoc` (`:doctype: book`, `:toc:`, `[part]` headings)
     that `include::`s per-section files with `leveloffset=+1`;
   - a shared `docs/attributes.adoc` for product names/paths used across pages;
   - a `docs/README.adoc` index describing the book and how to build it;
   - section files under `docs/user-manual/`.

   Required structure (per the user's request):
   - **Part "Start Here"** — `quickstart.adoc`, `installation.adoc`,
     `your-first-project.adoc` (install → register a child-project → file, refine, implement,
     validate one ticket end-to-end).
   - **Part "Concepts"** — explanation of the process: tickets as the single artifact,
     status = directory, the board as a generated index, the seven statuses and the state
     machine, READY gate, review findings (severity → disposition), WIP limits, dependencies
     vs lineage, the skill/CLI split (judgment vs mechanics).
   - **Part "CLI reference"** — one section per command (`install`, `upgrade`, `uninstall`,
     `doctor`, `project add/remove/list`, `ticket new`, `ticket move`, `board audit`,
     `board sync`), absorbing the README's current per-command sections and the
     `pickle.toml` configuration reference.

3. **A docs build** following the ai-sdlc pattern where it pays for itself: a `Gemfile` with
   `asciidoctor`/`asciidoctor-pdf` and a `just docs-pdf` (or at least `just docs-check`
   rendering/validating the adoc tree) recipe. Whether to ship PDF output or only validate
   the sources is an open decision for refinement. If a docs command lands, register it as
   the child's `docs` command in `pickle.toml` (currently commented out).

Content sources: the current `README.md` command sections, the skill's
`resources/tickets-README.md` rules (§0–§8) — the manual explains the process for humans but
must not fork the rules' normative text; it should summarise and point to the skill as the
source of truth.

**Soft couplings:**
- **T-019 (README accuracy polish)** — partially mooted by this ticket: its items 1–3 target
  README passages that this ticket moves or deletes. Whichever ticket is refined first should
  re-check the other; T-019's item 4 (`PLAN.md` synopsis) is untouched by this ticket.
- The `AGENTS.md` marker block and `install.go`'s `markerBlock()` mention docs in the commit
  policy; no change expected, but verify at refinement.
- Self-host policy applies: this is a `pickle`-child ticket, branch `feat/T-047-<slug>` in
  this repo; no `pickle install|upgrade` runs against the repo.

## Implementation Plan

### 0. Feature branch (mandatory)

Before any change, create a feature branch inside the target child-project's repo (child
`pickle`, path `.` — this repo):

```
git checkout main
git checkout -b feat/T-047-asciidoc-user-manual
```

Do all work on this branch, committing locally as you go. **Never push or open a merge
request without explicit user approval** (publish-gated child); end with a summary and a
suggested commit message (see Finish), and hand back.

### Prerequisite gate (hard)

None — no `depends-on:`; clean tree on `main` before branching.

### Confirmed design decisions (do not deviate without asking)

1. **Content only — no PDF/EPUB.** No `asciidoctor-pdf`, no `pdf-theme/`, no EPUB, no
   diagrams. The only build artifact is a validation pass (`just docs-check`) that renders
   the book to `/dev/null` with `--failure-level=WARN` to catch broken includes/xrefs.
2. **Register the docs command.** `pickle.toml` gains `docs = "just docs-check"` on the
   `[[project]]` (hand edit, self-host policy), and the `AGENTS.md` marker block is updated
   **by hand** to mirror `internal/install/install.go:markerBlock()` output.
3. **README = about + install, nothing else.** No Multi-project section, no Design-principle
   section, no command table, no configuration/command reference, no Build section. From-source
   build counts as install and stays as a short snippet; everything else lives in the manual,
   which the README links to.
4. **T-019 shrinks, is not dropped.** Record in T-019 that its items 1–3 are superseded by
   T-047's README rewrite; its remaining scope is item 4 (stale `PLAN.md` synopsis).
5. **No revnumber plumbing.** `docs/attributes.adoc` holds static attributes only (product
   name, repo URL, brew tap); no git-stamped `revnumber`/`revdate`.
6. **Summarise, never fork, the rules.** Concepts pages explain the process in the manual's own
   words and point to `.agents/skills/ticket-flow/resources/tickets-README.md` (§-references)
   as the normative source. Do not copy normative text wholesale — it would drift on every
   `pickle upgrade`.
7. **Structural pattern** follows the ai-sdlc book tree (layout, not content): book master with
   `[part]` headings, per-section files included with `leveloffset=+1`, shared
   `attributes.adoc`, `docs/README.adoc` index.

### Tasks

#### Task 1 — Docs skeleton

- `Gemfile` (repo root): `asciidoctor ~> 2.0` only (plus `logger` if asciidoctor warns
  without it).
- `docs/attributes.adoc`: `:product: pickle`, `:repo-url: https://github.com/codcod/pickle`,
  `:homebrew-tap: codcod/taps` — the single source for names/URLs used across pages.
- `docs/user-manual.adoc` (book master): `= pickle User Manual`, `:doctype: book`, `:toc:`,
  `:toclevels: 3`, `:sectnums:`, `:icons: font`, `include::attributes.adoc[]`, then three
  `[part]` headings — *Start Here*, *Concepts*, *CLI Reference* — each `include::`ing its
  section files with `leveloffset=+1`.
- `docs/README.adoc`: index — what the book covers, file layout, and how to validate
  (`bundle install`, `just docs-check`).

#### Task 2 — Part “Start Here” (`docs/user-manual/`)

- `quickstart.adoc` — the fastest path: install binary → `pickle install` in a project → ask
  the agent to “make it a ticket” → `tickets/BOARD.md` appears. Under 10 minutes; links out
  for detail.
- `installation.adoc` — the two installs (binary vs `pickle install`), Homebrew, `go install`,
  and from source (absorbs the README **Build** section: `just build/test/lint`,
  `RELEASING.md` pointer).
- `your-first-project.adoc` — end-to-end walkthrough: `pickle install` (incl. `--agent`),
  register a child with `pickle project add`, then one ticket through the whole flow: file →
  refine (READY gate) → implement → review → done, showing which steps are the agent's
  judgment and which are CLI mechanics.

#### Task 3 — Part “Concepts” (`docs/user-manual/concepts/`)

Four pages, each summarising and §-linking the skill's rules (decision 6):

- `the-flow.adoc` — one artifact per feature; status = directory; the generated board +
  `NOTES.md`; the judgment-vs-mechanics split (skill vs CLI) — absorbs the README
  **Design principle** section.
- `tickets.adoc` — ticket anatomy: frontmatter, one global id namespace, grading
  (impact/complexity/cost), `depends-on:` vs `spawned-by:` (gate vs lineage), append-only
  History.
- `lifecycle.adoc` — the seven statuses and the state machine; the READY gate; reviews:
  severity then disposition (the four dispositions, note-and-close default); WIP limits;
  done ≠ merged.
- `multi-project.adoc` — overarching project + connected children; per-child branches, WIP,
  commands; single-child as the degenerate case — absorbs the README **Multi-project**
  section.

#### Task 4 — Part “CLI Reference” (`docs/user-manual/`)

- `cli-reference.adoc` — one section per command with synopsis + behaviour, absorbing the
  README's per-command sections verbatim-in-substance (converted to AsciiDoc): `install`
  (incl. the `--agent` table), `upgrade`, `uninstall`, `doctor`, `project add|list|remove`,
  `ticket new`, `ticket move` (incl. the state-machine diagram as a literal block),
  `board audit`, `board sync`, `version`. Also absorbs the README command table as an
  overview list (drop the `[done: T-NNN]` tags — they are repo history, not user docs;
  this supersedes T-019 items 1–2).
- `configuration.adoc` — the `pickle.toml` reference, absorbing the README **Configuration**
  section (ownership rules per command, key reference, `[commit]` semantics), plus the new
  `docs` key now being exercised.
- While converting, fix the one known factual error rather than porting it (T-019 item 3):
  the `ticket move` gate is stricter than `board audit` **in substance** (it demands a
  `MERGED` History line), not because the audit “only warns” — audit errors on undone deps
  and warns only on done-but-unmerged.

#### Task 5 — Validation build

- `justfile`: add

  ```
  # Validate the AsciiDoc manual (broken includes/xrefs fail the check)
  docs-check:
      bundle exec asciidoctor --failure-level=WARN -o /dev/null docs/user-manual.adoc
  ```

- `pickle.toml`: replace the commented `# docs = ""` line with `docs = "just docs-check"`
  (hand edit per self-host policy).
- `AGENTS.md`: hand-update the marker-block Commands line to mirror `markerBlock()`:
  `` - `pickle`: build `just build` · test `just test` · lint `just lint` · docs `just docs-check` ``

#### Task 6 — README rewrite

Rewrite `README.md` to shop-window only:

- `# pickle` + the existing two-paragraph “what it is” opening (may keep one sentence naming
  the judgment/mechanics split, as *about*-level framing only — the section itself moves to
  `concepts/the-flow.adoc`);
- `## Install`: the two-installs note, Homebrew, `go install`, from-source snippet
  (`just build` / `go build -o pickle .`), and the release-tag caveat pointing at
  `RELEASING.md`;
- `## Documentation`: link to `docs/README.adoc` and `docs/user-manual.adoc`.

Nothing else survives (decision 3). Every deleted section must be traceable to a manual page
(Tasks 2–4) — no content lost, none duplicated.

#### Task 7 — Shrink T-019 (overarching bookkeeping, not part of the child diff)

Edit `tickets/1-to-do/T-019-….md`: add a dated note that items 1 and 3 are superseded by
T-047's README rewrite (item 1's passages deleted/moved; item 3 fixed in `cli-reference.adoc`)
and that item 2 was already mooted upstream by commit `f7b0a0a`, which deleted the
`## Phased build plan` section outright — leaving item 4 (`PLAN.md` synopsis) as the remaining
scope. Append a History line. Commit with explicit pathspecs per the commit policy.

### Acceptance test

From the repo root on the feature branch:

```sh
bundle install                      # installs asciidoctor from the new Gemfile
just docs-check                     # exit 0, no asciidoctor warnings/errors
just build && just test && just lint  # still green (no Go changes expected)
```

Structure and content checks:

```sh
# README is shop-window only: exactly these top-level sections remain
grep '^## ' README.md               # expected: "## Install" and "## Documentation" only
# the manual has the three requested parts
grep '^== ' docs/user-manual.adoc   # expected: Start Here, Concepts, CLI Reference
# every command has a reference section
for c in install upgrade uninstall doctor project ticket board version; do
  grep -q "pickle $c" docs/user-manual/cli-reference.adoc || echo "MISSING: $c"; done   # no output
```

Marker-block fidelity (self-host policy: throwaway dir, copied binary — verifies the Task 5
hand edit matches `markerBlock()` for this config):

```sh
just build && D=$(mktemp -d) && cp pickle "$D/pk" && REPO=$(pwd) && cd "$D" && \
  ./pk install --project pickle --build 'just build' --test 'just test' \
    --lint 'just lint' --docs 'just docs-check' --agent claude && \
  diff <(sed -n '/pickle:begin/,/pickle:end/p' AGENTS.md) \
       <(sed -n '/pickle:begin/,/pickle:end/p' "$REPO/AGENTS.md")   # empty diff
```

Finally `./pickle board audit` — 0 errors (T-019/T-047 bookkeeping edits included).

### Docs update (mandatory when user-facing)

This ticket *is* the docs. Registration of the new surface:

- `docs/README.adoc` indexes the book and states the validation command (Task 1);
- `README.md` links to the manual (Task 6);
- `pickle.toml` + `AGENTS.md` marker block carry the `docs` command (Task 5);
- T-019 updated (Task 7).

### Finish (mandatory)

1. Acceptance test green; `just build/test/lint/docs-check` clean.
2. Write a **summary**: files added/moved/deleted, where each README section landed, decisions
   honoured, anything deferred.
3. Suggested Conventional Commit message:

   ```
   docs: add AsciiDoc user manual, slim README to about + install (T-047)

   README.md becomes the shop window (about + install + link to docs/).
   docs/ gains a book-style manual — Start Here, Concepts, CLI Reference —
   validated by `just docs-check`, registered as the child's docs command.
   ```

4. Commit locally on `feat/T-047-asciidoc-user-manual`; **do not push or open an MR without
   user approval**; after approval, finalize (squash or keep history), push, open the MR —
   merging is the human's. Hand back with the summary.

## Review

### 2026-07-26 — review (protocol v1, no addenda configured)

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files (step 4b) — run via the pi `docs_readability` tool; 13 prose suggestions presented to the user for apply/discard (readability polish, not findings)
- [x] Findings recorded with severity **and** disposition per the rules §5; disposition summary line present (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — T-013, T-038, T-045 patched (each cited a `README.md` passage this ticket moved to `docs/user-manual/cli-reference.adoc`); T-019 already shrunk by Task 7
- [x] Summary + commit message & MR attributes presented for approval (step 9)

**Implementation audit.** All 7 tasks met, verbatim acceptance test green: `bundle install` + `just docs-check` exit 0 with no warnings; `just build/test/lint` green (no Go changes); README `##` sections are exactly Install + Documentation; the book has exactly the three parts; all 8 command names present in `cli-reference.adoc`; marker-block round-trip via throwaway-dir install byte-identical to the hand edit; `pickle board audit` 47 tickets, 0 errors, 0 warnings. All 7 confirmed decisions honoured (no PDF/EPUB, `docs = "just docs-check"` registered in `pickle.toml` + `AGENTS.md` by hand per self-host policy, README shop-window only, T-019 shrunk with History line, static attributes only, rules summarised with §-pointers — no normative text forked, ai-sdlc book layout).

**Quality/consistency audit.** Spot-checked every checkable CLI claim in the manual against the code: state machine + `--reason` rules (`internal/move/move.go:31-50`), pickup `MERGED` gate vs audit warn (`move.go:109-112`, `audit.go`), grade defaults `medium/medium/M` (`cli/ticket.go:81-83`), adjacent-pair ranges legal (`audit.go:58`), `doctor --verbose|-v`, `board sync --dry-run` non-zero on drift, `project remove` refusal on live tickets (`cli/project.go:131`), board cell `|`→`¦` sanitisation (`board.go:126`), `--claude-symlink` / deprecated `--no-claude`, install `--project` default = root dir name. T-019 item 3's factual error was fixed, not ported, in the `ticket move` section as planned. Every deleted README section traced to a manual page; no content lost, none duplicated (README's install snippet is sanctioned by decision 3).

### Findings

| id | severity | disposition | description | evidence | suggestion |
|----|----------|-------------|-------------|----------|------------|
| F1 | non-blocking | fixed inline | `concepts/the-flow.adoc` listed the embedded payload as `resources/{tickets-README,TEMPLATE,review-protocol}.md`, omitting `docs-readability.prompt.md` — contradicting `cli-reference.adoc`'s own install section, which names it | `skill/resources/` holds four files; the omission was inherited from the old README but this branch authored the page | add the fourth file to the brace list (done) |
| F2 | non-blocking | fixed inline | `cli-reference.adoc` said `pickle version` prints "`dev` for an unstamped build", omitting the buildinfo fallback: an unstamped `go install …@vX.Y.Z` binary reports its module tag, and `dev` appears only when no build info exists | `main.go:resolveVersion()` (ldflags → `debug.ReadBuildInfo()` → `dev`) | reword to state the fallback chain (done) |

**Disposition summary:** 2 non-blocking findings, both **fixed inline** (F1, F2) — prose this branch authored, no behaviour change; 0 noted, 0 folded, 0 new tickets. `just docs-check` re-run green after the fixes.

## History

- 2026-07-26 — created (TO DO). source: chat (user request: AsciiDoc manual in docs/, README reduced to about + install, patterned on ai-sdlc's docs tree)
- 2026-07-26 — refined: user pinned the open decisions (content only, no PDF/EPUB; register `docs = "just docs-check"`; README keeps nothing beyond about + install; T-019 shrinks to its item 4; no revnumber plumbing); cost collapsed M-L → M (mostly relocation/summarisation of existing text)
- 2026-07-26 — TO DO → READY: plan complete
- 2026-07-26 — applicability gate: clean (delta since READY = f7b0a0a, already reflected in the plan's README inventory); one non-blocking finding amended inline (Task 7: T-019 item 2 was mooted by f7b0a0a's deletion of the phased-plan section, not by this ticket)
- 2026-07-26 — READY → IN DEVELOPMENT: picked up
- 2026-07-26 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-26 — IN REVIEW → DONE: review clean: 2 non-blocking findings, both fixed inline (F1, F2); acceptance verbatim green
