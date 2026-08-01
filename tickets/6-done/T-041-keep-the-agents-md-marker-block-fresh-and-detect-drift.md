---
id: T-041
title: keep the AGENTS.md marker block fresh and detect drift
project: pickle
depends-on: []
spawned-by: [T-020, T-021]
impact: high
complexity: medium
cost: M
---

# T-041 — keep the AGENTS.md marker block fresh and detect drift

## Description

**Epic — merged from T-020 and T-021 by the 2026-07-26 board triage.** Both are in
`tickets/7-dropped/` with their full reproductions; read them for detail.

The `AGENTS.md`/`CLAUDE.md` marker block is generated from `pickle.toml` and is what every agent
actually reads. Nothing keeps it in step with the config, and nothing detects that it has fallen
out of step. These are the **write half** and the **detect half** of one defect, and splitting
them guarantees two half-fixes: a freshness fix with no way to verify it, or a detector with
nothing to make the block correct.

### Absorbed scope

| from | half | substance |
|---|---|---|
| T-021 | write | `pickle project add` and `pickle project remove` mutate `pickle.toml` (`internal/cli/project.go:95`, `:137` — both call `cfg.Save("")`) and return. Neither calls `injectMarker`, so the block still describes the *previous* set of children. Reproduced: after `pickle project add web sub` with WIP 5/5, `AGENTS.md` still listed only `demo`, with `demo`'s commands and `≤ 1` limits. An agent reading that refuses legitimate work on a project it is not told about. |
| T-020 | detect | `doctor.checkMarkers` (`internal/doctor/doctor.go:111-137`) only checks that a marker **pair is present** — never whether the installed block matches what `markerBlock` (`internal/install/install.go:506+`) would render today. So a hand-edited block gets a clean bill of health and then silently loses those edits on the next `pickle upgrade`; and a block that predates a payload change is equally undetectable. |

### Why the severity grew

T-018 changed the stakes. The block used to inline only the child **name list**; it now renders
each child's **commands, branch prefix and WIP limits** (`markerBlock`,
`internal/install/install.go:516+`). A stale block is no longer a cosmetic name omission — it
publishes wrong build commands and wrong WIP limits as authoritative instructions.

T-020 was split out of T-018 during refinement (user decision, 2026-07-25) precisely because
detection is a distinct feature with its own design questions: what counts as drift when the user
*intended* the edit, and how `doctor` should report it without crying wolf on every legitimate
customisation. That question is now this epic's to settle, and it is the same question the write
half raises from the other side — re-injecting on `project add` must not clobber intentional
hand-edits.

### Re-verified against the code, 2026-08-01 (refinement)

The two halves still reproduce; the line references above have moved and are superseded by the
table below. Four facts changed since filing, and they shape the plan:

| # | finding | consequence for this ticket |
|---|---|---|
| 1 | **The write half is unchanged.** `runProjectAdd` (`internal/cli/project.go:97`) and `runProjectRemove` (`:139`) still `cfg.Save("")` and return; nothing else in the tree calls `cfg.Save` except `install.writeConfig` (`internal/install/install.go:601`). `Upgrade` re-injects at `install.go:261-268`. | The fix is to lift `Upgrade`'s injection pair into one exported entry point and call it from both registry commands, so the two paths cannot drift apart. |
| 2 | **The detect half is unchanged.** `checkMarkers` (`internal/doctor/doctor.go:113-142`) tests presence only, via `hasMarkerBlock` (`:145`). `Check` calls it at `:50`, *before* the `cfg != nil` guard. | A drift check needs `cfg`, so `checkMarkers` must take it and degrade to presence-only when `pickle.toml` did not parse. |
| 3 | **Measured: this repo's installed block is byte-identical to a fresh render** — 40 lines, verified during refinement with a throwaway in-package test comparing `AGENTS.md`'s marker body to `markerBlock(config.Load("pickle.toml"))`. | The premise T-020 wanted confirmed (#4 in its list) holds *after* T-018: drift is a real signal, not near-universal noise. The check is silent on this repo, so it adds no self-host noise and does not wait on T-046. It also means the property is **testable**, and this ticket pins it (Task 6). |
| 4 | **`markerBlock` is now wrong about ticket prefixes.** T-058 shipped per-child `ticket_prefix`, but the branch bullet hardcodes the letter: `fmt.Fprintf(&branches, "\n  - `%s`: `%sT-NNN-<slug>`", p.Name, p.BranchPrefix)` (`install.go:769`). A child with `ticket_prefix = "RICK"` is told to cut `feat/T-NNN-<slug>`. | Same defect class as this ticket's thesis — the block states a wrong fact as authoritative — in the very function both halves are built on. Folded in (decision 6): a drift check that certified this block as correct would be certifying a lie. |

### Cross-references

- **T-018** (done) established that `upgrade` must not silently discard the marker body; its
  surgical-edit approach is the model for re-injecting without destroying user content.
- **T-022** (still standalone) fixes the *skill payload* stating commit policy, branch prefix and
  WIP limits unconditionally — the same contradiction seen from the other surface. When both land,
  the payload defers to the block and the block is finally trustworthy; they are worth sequencing
  together but do not share code.
- **T-036**'s note-and-close valve and **T-044**'s generated-board design (superseded T-039's
  parse-back gate, 2026-07-26) are unrelated in code but share the "never silently destroy user
  content" principle.
- **T-042** item 1 (marker-pair detection — four copies, one with a *reproduced* dry-run/real-run
  divergence) reserves exactly the helper this ticket's detect half needs, because reading the
  block **body** requires the same span the four copies compute. Refinement decision 5: T-041
  extracts it and routes every caller through it, which **closes T-042's item 1** and leaves that
  epic with item 3 (test payload root) alone — a re-grade and re-title, recorded as bookkeeping in
  the Finish step. Adding a fifth copy instead was the alternative and was rejected.
- **T-046** (self-host-aware doctor/upgrade) would have been a prerequisite if the drift check were
  noisy here. Finding 3 above measures that it is not, so the two stay independent.
- **T-056** (writable `serve`) becomes a third mutator of `pickle.toml`. It should call the same
  exported refresh entry point rather than re-deriving the injection pair; noted so the API this
  ticket adds is the one it inherits.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child-project is `pickle` — this repo, registered at `.` (`pickle.toml`). Before any
change:

```
git checkout main
git checkout -b feat/T-041-keep-the-agents-md-marker-block-fresh-and-detect-drift
```

Commit locally as you go. **Never push or open a merge request without explicit user approval**
(commit policy): finish with a summary and a suggested commit message, and let the user choose
squash-or-keep before the branch is pushed. Merging is the human's.

### Prerequisite gate (hard)

- `depends-on:` is empty; nothing blocks pickup.
- Clean tree on `main`, `just build && just test && just lint` green before the first edit — the
  acceptance test compares against a baseline.
- **Self-modify policy (`AGENTS.md`).** Never run `pickle install|upgrade|uninstall` against this
  repo from this branch. Every end-to-end check goes to a throwaway directory with the binary
  copied in (`D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && ./pk install …`).

### Confirmed design decisions (do not deviate without asking)

1. **Drift is a warning, never an error.** A hand-edited block is a legitimate user state until
   they run `upgrade`; erroring would fail `doctor` on a working project. This follows the
   precedent already set by `checkAgentScaffolds` (`internal/doctor/doctor.go:162-182`) and keeps
   `doctor`'s exit-code contract intact (errors → non-zero, warnings → 0).
2. **One line of output, no diff.** Report `markers: AGENTS.md block differs from what pickle.toml
   renders (N line(s) differ) — run \`pickle upgrade\`; hand-written content belongs outside the
   markers`. A unified diff of a 40-line block is not `doctor` output. `N` is the count of
   differing lines under a positional line-by-line comparison (padding the shorter side), which is
   cheap and needs no diff algorithm.
3. **Byte-exact comparison of the marker body**, after trimming leading/trailing newlines from
   both sides — no whitespace normalisation, no case folding. `injectMarker` writes
   `MarkerBegin + "\n" + block + "\n" + MarkerEnd`, so the trimmed body is exactly `MarkerBlock(cfg)`
   when current. The question the check answers is the literal one — *would `upgrade` change this
   block?* — and normalising would let a difference that `upgrade` will silently overwrite pass as
   clean.
4. **The write half re-injects unconditionally.** The block interior is pickle-owned (its own doc
   comment says content that is not derivable from the config "belongs in the surrounding file,
   which pickle never touches"), so `project add|remove` regenerate it rather than trying to guess
   whether an edit was intentional. Decision 1's warning is the safety net for the user who edited
   inside the markers anyway, and the docs say where such content belongs. `injectMarker` already
   reports `(marker current)` on a no-op, so a registry change that does not move the block prints
   nothing new.
5. **One span helper, four callers.** Extract the marker-pair span in `internal/install` and route
   every existing copy through it, closing T-042 item 1 (see Cross-references). The detect half
   needs the span to read the body, so the choice is *extract* or *add a fifth copy*.
6. **Fix the hardcoded `T-NNN` in the branch bullet** (`p.Prefix()` instead of the literal), per
   re-verification finding 4. Verified no self-host churn: `pickle` leaves `ticket_prefix` unset,
   `Prefix()` returns `"T"`, and the rendered block is unchanged for this repo.
7. **No new facts in the block.** This ticket makes the block *fresh* and *correct about what it
   already states*. Rendering each child's `ticket_prefix` as its own bullet, or reformatting
   anything, is out of scope. Also deliberately left alone: the Conventional-Commit example still
   reads `(T-2)` rather than `(T-NNN)` — cosmetic, and changing it would churn the block for every
   installed project for no correctness gain.
8. **`RefreshMarkers` mirrors `Upgrade`'s file policy exactly**, because it *is* `Upgrade`'s code:
   `AGENTS.md` is injected (created with a title header if absent, per `injectMarker`); `CLAUDE.md`
   is injected only when `os.Lstat` says it is a regular file, so a `CLAUDE.md -> AGENTS.md`
   symlink is left alone.

### Tasks

#### Task 1 — extract the marker-span helper, export the two readers (`internal/install/install.go`)

- Add `func markerSpan(text string) (bi, ei int, ok bool)`: `bi = strings.Index(text, MarkerBegin)`,
  `ei = strings.Index(text, MarkerEnd)`, `ok = bi >= 0 && ei > bi`. **Ordering is part of the
  predicate** — that is the bug T-042 item 1 reproduced.
- Route all three in-package copies through it: `injectMarker` (`:661-664`), `stripMarker`
  (`:707-708`), and the `uninstallMarkerFile` dry-run branch (`:458`, today an unordered
  `Contains && Contains`). The dry-run change is behavioural and intended: `uninstall --dry-run`
  and the real `uninstall` must agree on a file whose `pickle:end` precedes its `pickle:begin`.
- Add the one exported reader the other packages need:
  `func InstalledMarkerBody(path string) (body string, ok bool)` — reads `path`, finds the span,
  returns `strings.Trim(text[bi+len(MarkerBegin):ei], "\n")`; `ok == false` for an unreadable file
  or a missing/mis-ordered pair. Presence *and* body from one call, so `doctor` needs no second
  helper.
- Rename `markerBlock` → **`MarkerBlock`** (exported) and update its four in-package callers
  (`:152`, `:165`, `:261`, `:266`) plus the four tests that call it (`install_test.go:449`, `:487`,
  `:517`, `:605`).

#### Task 2 — `install.RefreshMarkers` as the single injection entry point (`internal/install/install.go`)

- Add `func RefreshMarkers(root string, cfg *config.Config) (Result, error)` holding exactly the
  logic currently inline in `Upgrade` at `:261-268` (AGENTS.md always; CLAUDE.md only when a
  regular file), returning its own `Result`.
- Replace that inline block in `Upgrade` with a call that appends the returned `Created`/`Skipped`
  into `Upgrade`'s own `Result` (or takes `*Result` — implementer's choice, as long as `upgrade`'s
  output is byte-identical to today's for an unchanged block *and* for a changed one).
- Doc comment must state the ownership rule the callers rely on: the block interior is regenerated
  from `cfg`, so hand-written content belongs outside the markers.

#### Task 3 — the ticket-prefix fix (`internal/install/install.go`, `MarkerBlock`)

- Branch bullet becomes `` `%s`: `%s%s-NNN-<slug>` `` with `p.Name`, `p.BranchPrefix`, `p.Prefix()`.
- `TestMarkerBlockGolden` (`install_test.go:605`): give the `beta` child `TicketPrefix: "BETA"` so
  the golden **proves** the fix, then regenerate:
  `UPDATE_GOLDEN=1 go test ./internal/install/`. Inspect the resulting
  `internal/install/testdata/markerblock.golden` diff by hand — it must show exactly one changed
  line (`beta`'s branch bullet, `ticket/BETA-NNN-<slug>`) and nothing else.

#### Task 4 — `project add|remove` refresh the block (`internal/cli/project.go`)

- After the successful `cfg.Save("")` in `runProjectAdd` (`:97`) and `runProjectRemove` (`:139`),
  call `install.RefreshMarkers(cfg.Root(), cfg)` and print its `Created`/`Skipped` with the same
  `  + ` / `  = ` prefixes `runInstall`/`runUpgrade` use (`internal/cli/install.go:65-70`).
- Ordering matters: the registry write is the primary effect and must be reported first; the
  refresh follows the existing `registered child-project …` / `removed child-project …` line.
- A refresh failure is an error (`errf`) — the registry is already saved, so the message must say
  what succeeded and point at `pickle upgrade` to finish the job. Do not roll back `pickle.toml`.

#### Task 5 — the drift check (`internal/doctor/doctor.go`)

- `checkMarkers(root string, cfg *config.Config, r *Result)`; the call site (`:50`) moves nothing —
  `cfg` is already in scope there and may be `nil`.
- Delete the private `hasMarkerBlock` (`:145`) and use `install.InstalledMarkerBody`. Presence
  findings keep their exact current wording, so existing expectations in `doctor_test.go` still
  hold.
- When `cfg != nil`, compare the body to `install.MarkerBlock(cfg)` per decision 3 and, on a
  difference, `r.warnf` per decision 2 — for `AGENTS.md` and for a regular-file `CLAUDE.md`
  (a symlinked `CLAUDE.md` has no body of its own and is skipped, as today). When `cfg == nil`,
  presence only: a project whose `pickle.toml` does not parse already has that error, and
  "differs from what pickle.toml renders" is meaningless without a config.
- On a match, add the passed entry (e.g. `AGENTS.md marker block current`) so `doctor -v` shows the
  check ran rather than leaving its silence ambiguous.

#### Task 6 — tests

- `internal/install/install_test.go`:
  - `markerSpan` table: ordered pair, reversed pair, begin-only, end-only, neither.
  - `uninstall --dry-run` vs real `uninstall` on a reversed-marker `AGENTS.md` — assert both report
    *no marker* (the T-042 item 1 regression).
  - `RefreshMarkers`: updates a stale `AGENTS.md`, reports `(marker current)` on a second call,
    injects a regular-file `CLAUDE.md`, leaves a `CLAUDE.md` symlink untouched.
  - `MarkerBlock` renders `p.Prefix()` in the branch bullet (direct assertion, independent of the
    golden).
  - **`TestSelfHostMarkerBlockIsCurrent`** — load `../../pickle.toml`, render `MarkerBlock`, compare
    to `../../AGENTS.md`'s `InstalledMarkerBody`, fail with a line-by-line report that also names
    the hand-mirror obligation ("hand-mirror `AGENTS.md` per the self-modify policy"). This makes
    the hand-mirroring rule in `AGENTS.md` machine-checked instead of a promise, and it passes today
    (re-verification finding 3). Reuse the package's existing `payloadRoot()` helper
    (`internal/install/install_test.go:16`) rather than raw `../..` — it is CWD-*relative*, not
    independent (T-042 item 3 owns replacing it with an absolute root later); it is still correct
    for this test today, so no new helper is warranted here.
- `internal/doctor/doctor_test.go`: extend the fixture-based cases — byte-identical block ⇒ no
  warning + a passed entry; one line edited **inside** the markers ⇒ one warning naming the file,
  0 errors; text appended **outside** the markers ⇒ no warning (the false-positive guard);
  unparseable `pickle.toml` ⇒ no drift warning, presence findings unchanged.
- `internal/cli/cli_test.go`: `project add` on a fixture install rewrites `AGENTS.md` to name the
  new child with its own WIP limits and branch/ticket prefix; `project remove` drops it again.

### Acceptance test

```sh
# 1. In-repo: build + full suite + lint + docs must be green.
just build && just test && just lint && just docs-check

# 2. End-to-end in a throwaway dir (self-modify policy: never against this repo).
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
git init -q . && mkdir sub && (cd sub && git init -q .)
./pk install --project demo --path . --agent claude

# 2a. The write half: the block learns the new child, its limits and its prefixes.
./pk project add web sub --ticket-prefix WEB --branch-prefix ticket/ --wip-dev 5 --wip-review 5
sed -n '/pickle:begin/,/pickle:end/p' AGENTS.md
#   expect: `web` in the registered list; "- `web`: `ticket/WEB-NNN-<slug>`";
#           "- `web`: `3-in-development/` ≤ 5 · `4-in-review/` ≤ 5"
./pk doctor            # expect: 0 error(s), 0 warning(s) about markers

# 2b. The detect half: an edit inside the markers is reported, once, as a warning.
perl -0pi -e 's/(pickle:begin -->\n)/$1WRONG\n/' AGENTS.md
./pk doctor; echo "exit=$?"
#   expect: WARNING: markers: AGENTS.md block differs from what pickle.toml renders (…) ; exit=0

# 2c. Content OUTSIDE the markers must never warn (false-positive guard).
# `board sync` first: `project add` in 2a left BOARD.md stale (a pre-existing,
# out-of-scope gap — T-051/T-052 territory, reproduces on pre-T-041 main too;
# see the Review finding below), which would otherwise fail upgrade's own
# post-upgrade audit before this ticket's check ever runs.
./pk board sync >/dev/null && ./pk upgrade >/dev/null && printf '\n## House rules\n\nmine.\n' >> AGENTS.md
./pk doctor            # expect: no marker warning, 0 error(s)

# 2d. Removal keeps the block honest.
./pk project remove web && sed -n '/pickle:begin/,/pickle:end/p' AGENTS.md | grep -c '`web`'
#   expect: 0

# 2e. The dry-run/real agreement fixed in Task 1.
printf '# X\n\n<!-- pickle:end -->\n<!-- pickle:begin -->\n' > AGENTS.md
./pk uninstall --dry-run | grep AGENTS.md   # expect: "= AGENTS.md (no marker)", matching the real run
```

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`:
  - `<<cmd-project>>` (`:129-157`): `add`/`remove` now also re-inject the `AGENTS.md` (and
    regular-file `CLAUDE.md`) marker block, so the agent instructions never describe a stale
    registry — with the ownership rule: content between the markers is pickle-owned.
  - `<<cmd-doctor>>` (`:207-234`): extend the marker bullet (`:221`) from presence to freshness —
    a block that differs from what `pickle.toml` renders is a **warning** pointing at
    `pickle upgrade`, and hand-written content belongs outside the markers. Mirror the phrasing of
    the agent-scaffold bullet (`:222-224`) so the two drift warnings read alike.
- `docs/user-manual/cli-reference.adoc:188-196` (the marker-block ownership prose, not
  `configuration.adoc` — that file says nothing about the marker block today): extend "regenerated
  from `pickle.toml`" to name `install`, `upgrade` **and** `project add|remove` as the commands that
  do it; the existing "keep notes outside the markers" sentence already covers ownership.
- No change to `skill/` — the payload's unconditional statement of commit policy/branch prefix/WIP
  limits is **T-022**'s scope, not this ticket's.
- Re-run `just docs-check`.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated; `internal/install/testdata/markerblock.golden` regenerated and its diff reviewed
   by hand (one line).
3. **Bookkeeping in the overarching repo** (explicit pathspecs, may be committed automatically):
   edit `tickets/1-to-do/T-042-*.md` to mark **only the marker-span half** of item 1 resolved by
   T-041 — item 1 also carries T-017's second sub-item (skill-dir dry-run labels not matching the
   real run's labels, `install.go:352,354,361`), which this ticket does not touch, so T-042 is left
   with **two** remaining items (item 1's skill-dir-label sub-item + item 3, test payload root), not
   one. Re-grade/re-title accordingly (the same treatment T-044's review sweep gave item 2) and
   refresh item 1's now-stale line references while in there, then `pickle board sync`. Do **not**
   silently delete the resolved sub-item; record it as the T-044 patch did.
4. Confirm the self-host block needs no hand-mirroring: `TestSelfHostMarkerBlockIsCurrent` passing
   *is* that confirmation. If it fails, the block content changed — stop and re-read decision 7
   before touching `AGENTS.md`.
5. Write a summary (files touched, decisions honoured, anything deferred).
6. Suggest a Conventional Commit message, e.g.:

   ```
   fix(install): keep the AGENTS.md marker block fresh and detect drift (T-041)

   project add|remove now re-inject the marker block through one shared
   RefreshMarkers entry point, doctor warns when the installed block differs
   from what pickle.toml renders, and the branch bullet honours each child's
   ticket_prefix.
   ```

7. Commit locally on the ticket branch. Present the message; **do not push or open a merge request
   without user approval**. After approval: finalize (squash or keep history — the user chooses),
   push, open the MR. Merging is the human's.

## Review

**Verdict: no blocking findings — IN REVIEW → DONE (2026-08-01).** Both halves ship and are
pinned by tests; the acceptance test was re-run verbatim and every step matched its documented
expectation.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [ ] Docs-readability pass — **skipped: reviewer unreachable** (the `docs_readability` backend
      failed with `model_not_supported` from the Gemini provider; step 4b is an optional,
      sanctioned conscious skip)
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

### Implementation audit (step 2)

| item | verdict | evidence |
|---|---|---|
| Task 1 — `markerSpan` + three in-package callers + `InstalledMarkerBody` + `markerBlock` → `MarkerBlock` | **met** | `install.go:659-687` (`markerSpan`, `InstalledMarkerBody`), routed in `injectMarker` (`:711`), `stripMarker` (`:755`), `uninstall --dry-run` (`:477`); all four in-package callers and eight test call sites renamed |
| Task 2 — `RefreshMarkers` as the single injection entry point; `upgrade` output unchanged | **met** | `install.go:232-260`; `Upgrade` now calls it (`:283-288`). Verified by diffing `upgrade` output from a `main`-built binary vs this branch, on an unchanged **and** a hand-edited block: identical apart from the `-ldflags` version string |
| Task 3 — branch bullet honours `p.Prefix()`; golden proves it | **met** | `install.go:818`; `markerblock.golden` diff is exactly one line (`beta`: `ticket/BETA-NNN-<slug>`) |
| Task 4 — `project add\|remove` refresh the block, `+`/`=` idiom, error points at `upgrade`, no rollback | **met** | `project.go:99-125,161-166`; live run printed `  + AGENTS.md (marker updated)` / `  + CLAUDE.md (marker updated)` after `project add` |
| Task 5 — drift check: warning-only, one line, byte-exact, `cfg == nil` degrades to presence | **met** | `doctor.go:113-192`; presence wording unchanged, `AGENTS.md marker block current` passed entry added |
| Task 6 — tests | **met** | `TestMarkerSpan`, `TestUninstallDryRunAgreesOnReversedMarkers`, `TestRefreshMarkers`, `TestMarkerBlockRendersTicketPrefixInBranchBullet`, `TestSelfHostMarkerBlockIsCurrent`, four `doctor` drift cases, `TestProjectAddRefreshesMarkerBlock` |
| Acceptance test 1 — `just build && just test && just lint && just docs-check` | **met** | all green (re-run after this review's inline fixes) |
| Acceptance test 2a–2e — throwaway-dir end-to-end | **met** | 2a: `web` + `ticket/WEB-NNN-<slug>` + `≤ 5 · ≤ 5`, doctor clean · 2b: one warning, `exit=0` · 2c: no warning after out-of-marker text · 2d: `0` · 2e: `= AGENTS.md (no marker)` |
| Decisions 1–8 | **honoured** | warning-not-error (1), one line no diff (2), byte-exact trimmed compare (3), unconditional re-inject (4), one span helper (5), `Prefix()` (6), no new facts in the block (7), `RefreshMarkers` mirrors `Upgrade`'s file policy (8 — verified: a `CLAUDE.md` symlink survives, a regular file is injected) |
| Self-modify policy | **honoured** | no `install\|upgrade\|uninstall` run against this repo; every end-to-end check in `mktemp -d` with the binary copied in; `AGENTS.md` untouched and machine-checked current |
| Finish step 3 — T-042 bookkeeping | **met** | T-042 re-titled/re-graded (cost M → S), item 1's marker-span half marked resolved without deleting it, dated History line added, board regenerated |

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | fixed inline | `RefreshMarkers` was inserted **between** `Upgrade`'s doc comment and `func Upgrade`, so the exported `Upgrade` lost its godoc and `RefreshMarkers` inherited prose about stamping `payload_version`. Its own text also claimed `pickle install` re-injects *through* this entry point, which it does not (`Run` injects directly, because only there does `--agent` decide file-vs-symlink). Exactly the defect class this ticket exists to kill, in its own new API docs. | `install.go:231-248` before the fix | Moved `Upgrade`'s comment back above `func Upgrade` and corrected `RefreshMarkers`' caller list (both done in this review; prose only, no behaviour change) |
| F2 | non-blocking | fixed inline | The T-042 patch wrote a `###` heading across **two** lines with a 4-space-indented continuation — in CommonMark that is a heading plus an *indented code block*, with `~~`/`**` left unclosed. | `tickets/1-to-do/T-042-…md:27-28` before the fix | Joined into a single-line heading (done in this review) |
| F3 | non-blocking | fixed inline | Duplicate 100-character error suffix in `runProjectAdd` and `runProjectRemove` — the "pickle.toml is already saved" wording existed twice, one copy away from drifting. | `project.go:102`, `:165` before the fix | Wrapped once inside `refreshMarkers` (`%w`), call sites now `errf("%v", err)`; user-visible message byte-identical (done in this review) |
| F4 | non-blocking | noted | The drift line count overstates the drift for inserts/deletes: inserting **one** line reports `41 line(s) differ`, because the comparison is positional and every following line shifts. Decision 2 locked that algorithm deliberately ("cheap, needs no diff algorithm"), so this is design, not defect — but the message reads as a magnitude claim a user will over-trust. | acceptance step 2b, measured: `markers: AGENTS.md block differs … (41 line(s) differ)` for a one-line insert | If it ever matters, either drop `N` or count non-shared lines as a multiset; both are design changes to decision 2, not fixes |
| F5 | non-blocking | noted | `project add\|remove` now write `AGENTS.md` **and** any regular-file `CLAUDE.md` — including a hand-written `CLAUDE.md` in an `--agent pi` install, which gains a pickle block from a command a user reads as "registry only". Decision 8 mandates mirroring `Upgrade`, and `upgrade` on `main` does exactly the same, so the hazard is pre-existing and unchanged in kind; only its trigger set grew. | `--agent pi` install + hand-written `CLAUDE.md`: `project add` → `1` marker pair; `main`-built `pickle upgrade` → `1` marker pair | None. If it is ever revisited, the fix belongs to the file-ownership policy as a whole (install/upgrade/project), not to this entry point |
| F6 | non-blocking | folded → **T-052** | `project add` leaves `BOARD.md` stale (a new child changes the board's generated shape), so the very next `upgrade` fails its post-upgrade audit with `ERROR: BOARD.md is stale or hand-edited`. This is the finding the amended acceptance test 2c refers to; it reproduces on `main`, so T-041 neither caused nor worsened it. | reproduced with the `main`-built binary: `install` → `project add web sub` → `upgrade` ⇒ `post-upgrade audit found 1 error(s)`, `exit=1` | Already owned, in full, by **T-052** (same command pair, same verdict, its own reproduction). No new ticket; nothing to add to T-052 that it does not already say |
| F7 | non-blocking | noted | The Description's cross-reference claims "**T-056** becomes a third mutator of `pickle.toml`" and should call `RefreshMarkers`. T-056's current scope names no registry write at all (its write surface is the ticket tree, ranking and errors). The advice is harmless if T-056 grows one, but it is stated as fact about another ticket. | `grep -n 'pickle.toml' tickets/1-to-do/T-056-*.md` ⇒ no scope hit; T-056 "Work areas" list | Leave it — a speculative note in a DONE ticket's cross-references, cheap to re-check when T-056 is refined |
| F8 | non-blocking | noted | The mid-implementation amendment to acceptance step 2c (`board sync` inserted, with a comment pointing at "the Review finding below") is not recorded in `## History`, unlike the pickup gate's amendments. The forward reference also dangled until this Review was written (now F6). | ticket diff `main…HEAD`: acceptance-test hunk with no matching History line | Recording it here closes it; the general rule (plan edits made in development earn a History line) is worth remembering rather than ticketing |
| F9 | non-blocking | noted | The positional line-by-line loop exists twice: `doctor.diffLineCount` (production) and the failure report in `TestSelfHostMarkerBlockIsCurrent` (test, different package). Duplication is real but tiny and crosses a prod/test boundary; batching it into T-042 would pad that epic rather than shrink it. | `doctor.go:172-192`, `install_test.go` self-host test | None |

**Disposition summary:** 9 findings, 0 blocking — **3 fixed inline** (F1, F2, F3 — all prose/idiom
this branch authored, no behaviour change), **1 folded** into T-052 (F6), **5 noted** (F4, F5, F7,
F8, F9). No follow-up ticket minted: none passed the promotion test that T-052 does not already
absorb.

### Impact sweep (step 8)

Only one non-terminal ticket references T-041: **T-042**, already patched by this branch's
bookkeeping (verified: its remaining item-1 sub-item at `install.go:372/:379/:384` and item 3's
five payload-root copies still reproduce exactly as written). **T-022** (payload states policy
unconditionally) is strengthened, not invalidated — the block it defers to is now correct about
per-child ticket prefixes too. **T-046** stays independent: this repo's block is byte-identical to
a fresh render, so the new check adds no self-host noise (`TestSelfHostMarkerBlockIsCurrent`
passes). **T-051/T-052** unchanged in premise (see F6). No ticket patch required.

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-020 and T-021, both
  moved to 7-dropped/ as absorbed
- 2026-08-01 — TO DO → READY: plan complete
- 2026-08-01 — applicability gate (pickup): fresh sub-agent audit verdict AMEND — every
  load-bearing claim reproduced (incl. independently re-verifying the byte-identical marker
  block), tree unchanged since READY; 3 findings fixed inline (Docs task retargeted to
  cli-reference.adoc; Finish step 3 corrected — T-041 closes only T-042 item 1's marker-span
  half, two items remain; Task 6's payloadRoot() mischaracterisation struck), 4 note-and-closed
  (orphaned strings import — compiler catches it; T-051 premise now false — no action; call-site
  line-ref undercount — mechanical; self-host test coupling — accepted by design)
- 2026-08-01 — READY → IN DEVELOPMENT: picked up
- 2026-08-01 — IN DEVELOPMENT → IN REVIEW: acceptance test green
- 2026-08-01 — IN REVIEW → DONE: review: 0 blocking; 3 fixed inline (F1-F3), 1 folded into T-052 (F6), 5 noted (F4,F5,F7-F9)
