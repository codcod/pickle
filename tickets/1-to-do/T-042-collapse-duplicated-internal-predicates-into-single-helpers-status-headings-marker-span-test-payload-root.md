---
id: T-042
title: collapse duplicated internal predicates into single helpers (skill-dir dry-run labels, test payload root)
project: pickle
depends-on: []
spawned-by: [T-015, T-017, T-032, T-041]
impact: low
complexity: low
cost: S
---

# T-042 — collapse duplicated internal predicates into single helpers (skill-dir dry-run labels, test payload root)

## Outcome

After this ships, a dry-run install preview names the same outcome the real run produces for the skill directory, and every test that needs the repo root resolves it through one CWD-independent helper instead of separately-maintained relative-path guesses.

## Description

**Epic — merged from T-015, T-017 and T-032 by the 2026-07-26 board triage.** All three are in
`tickets/7-dropped/` with their full site inventories; read them for detail.

Three instances of the same defect shape: **one fact written in four or five places, and the
copies have begun to disagree**. Different layers, identical review criteria — extract one
helper, route every caller through it, prove behaviour is unchanged. Reviewing them separately
means making the same "is this the right choke point?" judgement three times.

They are ordered below by severity, because only the first has a **measured behavioural
divergence**; the other two are latent.

### 1. Marker-pair detection — ~~four copies, and the newest one diverges~~ **marker-span half resolved by T-041; the skill-dir dry-run-label sub-item remains** (T-017)

**Resolved by T-041 (2026-08-01).** T-041 needed to *read* the marker block's body (not just
detect its presence) for its drift check, which required exactly the extraction this item asked
for. It added `markerSpan(text) (bi, ei int, ok bool)` in `internal/install/install.go` — ordering
is part of the predicate, so a dry-run and the real run can no longer disagree about whether a
reversed `<!-- pickle:end -->`…`<!-- pickle:begin -->` pair is a marker block at all (T-041 pins
this with `TestUninstallDryRunAgreesOnReversedMarkers`) — and routed every in-package copy
(`injectMarker`, `stripMarker`, the `uninstall --dry-run` check) through it, plus an exported
`install.InstalledMarkerBody(path)` that `internal/doctor` now calls instead of its own
`hasMarkerBlock`. The **measured divergence this item was filed for is gone.**

T-017's *second* item is **not** touched by T-041 and remains this epic's scope: **dry-run labels
don't match the real run's labels** for the skill dir. `internal/install/install.go`: the dry-run
branch reports `res.removed(SkillDir + " (dry-run)")` (`:372`) while the real run reports
`(symlink)` (`:379`) or a bare `SkillDir + "/"` (`:384`) depending on which branch actually ran —
so a dry-run preview can name a different outcome than the real run produces. Same theme as the
marker-pair fix: the preview and the act must agree, this time on the *label*, not the *predicate*.

### 2. Board status-heading matching — ~~four copies~~ **resolved by T-044** (T-015)

**Patched 2026-07-26 by T-044's review impact sweep.** T-044 deleted three of the four copies
(`board.ParseCells`, `sync.matchStatus`, `board.sectionSpan`) along with the whole parse-back
machinery; the heading-match loop now exists exactly once, in the read-only `board.Parse`
(drift summary only). No duplication remains — this item is **dropped from scope**.

T-015's second item (the `TestSyncTerminalMembership` / D3 carry-over test gap) is likewise
obsolete: sync no longer carries cells over at all — terminal cells are derived from ticket
History at render time and every DONE/DROPPED ticket is always rendered (T-044 D3/D4). The
epic's remaining scope is item 1's skill-dir dry-run-label sub-item (its marker-span half was
resolved by T-041, 2026-08-01 — see above) and item 3 (test payload root).

### 3. Test payload root — five copies, four of them CWD-relative (T-032)

| site | idiom |
|---|---|
| `internal/install/install_test.go:17` | `payloadRoot()` returning `filepath.Join("..", "..")` |
| `internal/doctor/doctor_test.go:16` | the same function duplicated verbatim |
| `internal/move/move_test.go:21` | inlined `os.DirFS(filepath.Join("..", ".."))` |
| `internal/sync/sync_test.go:21` | inlined `os.DirFS(filepath.Join("..", ".."))` |
| `internal/cli/cli_test.go:24` (T-029, re-anchored by T-043) | package-level `repoRoot`, absolute, computed in `TestMain`, **now validated against `skill/SKILL.md`** |

The first four break if the test process's CWD ever moves — which **T-043**'s `TestMain` sandbox
does deliberately. Unify on the absolute, CWD-independent form.

### 4. Skill-dir symlink predicate — one exact duplicate left by T-046 (T-046 review)

**Folded here by T-046's review (2026-08-12), two items, both non-blocking.**

T-046 added the exported predicate `install.SkillLinked(root) bool`
(`internal/install/install.go`, beside the `SkillDir` constant) —
`os.Lstat(join(root, SkillDir))`, `err == nil && mode&os.ModeSymlink != 0` — and routed
`Upgrade`'s skill-refresh guard through it. It deliberately left the other `Lstat` sites to this
epic, on the stated rationale that they "need `Lstat`'s *existence* answer as well as the mode".
That rationale is **correct for `Uninstall`** (`:406` branches on dry-run/symlink/dir and needs
`fi`), but **wrong for `copyPayload`** (`internal/install/install.go:541`): its guard is
`if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink != 0`, where `fi` is used for
nothing else — i.e. **character-for-character `SkillLinked(root)`**, since `dst` is the same
`filepath.Join(root, filepath.FromSlash(SkillDir))`. It is a free, exact collapse to
`if SkillLinked(root) { … }`; only the mis-stated reason kept it out of T-046's diff.

Second item, same theme, in the tests: `internal/doctor/doctor_test.go`'s `selfHostFixture`
and `internal/doctor/hooks_test.go`'s `TestCheckSelfHostLinkStillReportsIncapablePATHPickle`
both build the self-host symlink by hand (`RemoveAll` the installed skill dir, `filepath.Abs`
the payload `skill/`, `os.Symlink`). The duplication is *justified* as written — the two need
different bases (`installFixture` vs `gitFixture`) and so cannot compose — but a four-line
`linkSkill(t, root)` taking the root would serve both. Natural to do alongside item 3, which
already reworks these two test files' shared payload-root helper.

### 5. The bookkeeping-guard offender scan — two copies, one per hook (T-082 review)

**Folded here by T-082's review (2026-08-14), non-blocking.**

T-082 added a second rule to `internal/hook` (`pre-push` beside `pre-commit`) and, with it, a
verbatim second copy of the offender scan both rules end with: split the `-z` output of a
`git diff` on `NUL`, then keep every path where
`p == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(p, prefix)`.

| site | rule |
|---|---|
| `internal/hook/hook.go:505-515` | `CheckPreCommit`, over `git diff --cached --name-only -z` |
| `internal/hook/prepush.go:129-139` | `CheckPrePush`, over `git diff --name-only -z <base>...<sha>` |

The two differ only in which `git diff` produced the bytes, so the collapse is a helper taking
the diff output and the prefix and returning the offenders — `ticketsPrefix` and
`maxListedPaths` are already shared, which is exactly why T-082's refinement concluded the
shared code "needs no new plumbing"; the conclusion was right about the plumbing and silent
about the copy. Worth doing because a third hook, or any change to what counts as a `tickets/`
path, would otherwise have to be made twice — the same argument as items 1 and 4.

### Sequencing

- **T-043** (test harness + coverage) **landed first** (2026-08-06, D5): it owns `TestMain` and
  left item 3's five-site unification untouched, so the sequencing question is settled and this
  ticket is now unblocked. One consequence for item 3: `TestMain` no longer merely computes
  `wd/../..` — it also `os.Stat`s the payload marker `skill/SKILL.md` under the resolved root and
  exits with a clear message if it is missing. When item 3 replaces the computation with the shared
  CWD-independent helper, **delete that validation with it** (the helper cannot resolve a wrong
  root, so the check becomes dead weight) rather than leaving a stat against a path the helper no
  longer derives.
- **T-044** (which superseded T-039, 2026-07-26) rewrote `internal/board` and `internal/sync`
  heavily (single renderer, sync becomes regenerate) and **deleted** the helpers item 2 would
  have unified (`ParseCells`, `sectionSpan`, `subgroupSpan`, sync's `matchStatus`) — the sweep
  after T-044's review dropped item 2 from scope (see above). The former collision risk with
  T-044 is gone; only the T-043 sequencing note (item 3) remains live.
- **T-040 deferred the `T-\d+` unification here — it is now this epic's, unconditionally**
  (T-040 decision D6, 2026-08-02: its self-reference check needed no shape check, so it touched no
  regex). The shape is spelled out three times: `idRE` and `filenameRE`
  (`internal/ticket/ticket.go:52-63`) and `board.rowRE` (`internal/board/board.go:35`);
  `ticket.go`'s comment now names T-042 as the owner.
- **T-013 item 6** ("project-root resolution is triplicated" across the setup commands) is the same
  defect shape and a natural fourth item, but T-013 stays standalone; fold it in only if T-013 is
  not picked up first.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-015, T-017 and T-032,
  all three moved to 7-dropped/ as absorbed
- 2026-07-26 — patched by T-044's review impact sweep: item 2 (status-heading duplication +
  D3 carry-over test gap) dropped from scope — its targets were deleted by the generated-board
  rewrite; remaining scope is items 1 and 3
- 2026-08-01 — patched by T-041 (Finish step 3): item 1's marker-span half resolved — T-041
  extracted `markerSpan`/`InstalledMarkerBody` and routed every caller through them while
  building its own drift check; item 1's skill-dir dry-run-label sub-item (T-017) was not in
  T-041's scope and remains. Re-titled and re-graded (impact/complexity unchanged, cost M → S)
  to reflect the smaller remaining surface: item 1's label sub-item + item 3 (test payload root)
- 2026-08-03 — patched by T-040's review impact sweep (finding N8): T-040 deferred the `T-\d+`
  regex unification here by decision D6, so the conditional cross-reference ("if T-040 defers
  that") is now settled fact and its two stale line references were refreshed
  (`ticket.go:95` → `:100-110`, `board.go:29` → `:51`). Scope grows by one small item; not
  re-graded (still low/low/S).
- 2026-08-06 — patched by T-043's review impact sweep: T-043 landed (harness + command-layer
  coverage), so item 3's sequencing note is settled — this ticket goes second, as D5 agreed, and
  the harness inventory row is re-anchored (`cli_test.go:19` → `:24`). Item 3 gains one
  instruction: delete `TestMain`'s new `os.Stat(skill/SKILL.md)` root validation along with the
  `wd/../..` computation it guards. Scope and grade unchanged (low/low/S)
- 2026-08-10 — patched by T-080's review impact sweep (step 8): **re-anchored only — scope,
  substance and grade unchanged.** T-080 moved the status table out of `internal/ticket` and the
  board's order/heading maps out of `internal/board`, which shifted four of this ticket's cited
  line numbers: the id-shape regexes `ticket.go:100-110` → `:52-63` and `board.rowRE`
  `board.go:51` → `:35` (both now much further up their files), plus item 3's
  `install_test.go:15` → `:17` and `move_test.go:20` → `:21` (one import line each). The
  `doctor_test.go` anchor is corrected to `:16` in passing as **pre-existing** drift, not T-080's
  — that file was never touched by this branch. Both items remain live and unaffected in
  substance: the `T-\d+` shape is still spelled out three times (`idRE`, `filenameRE`,
  `board.rowRE`), `ticket.go`'s comment still names T-042 as the owner, and the five test
  payload-root variants are untouched
- 2026-08-12 — patched by **T-046's review impact sweep**: gained **item 4** (skill-dir symlink
  predicate). T-046 introduced `install.SkillLinked` and routed one call site through it, leaving
  `copyPayload`'s guard — which is an *exact* duplicate of the new predicate, not a case needing
  `Lstat`'s existence answer as T-046's plan asserted — plus a justified-but-collapsible
  self-host-symlink fixture duplication across the two `internal/doctor` test files. Both are
  `folded` dispositions from that review, no new ticket
- 2026-08-14 — patched by **T-082's review impact sweep**: gained **item 5** (the bookkeeping-guard
  offender scan, now written once per hook in `internal/hook`). A `folded` disposition from that
  review, no new ticket. Scope grows by one small item; not re-graded (still low/low/S)
