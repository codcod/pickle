---
id: T-074
title: rename the installed skill directory to brine, with a legacy sweep on upgrade
project: pickle
depends-on: [T-073]
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-074 — rename the installed skill directory to brine, with a legacy sweep on upgrade

## Outcome

After this ships, there is one name on disk for one thing: a project's installed skill lives at `.agents/skills/brine` and is invoked as `/skill:brine`, and `pickle upgrade` sweeps away the old `ticket-flow` directory and symlink so no project ends up carrying two copies of the skill. A project that has not upgraded yet is told so by `pickle doctor`, which fails with the one command that fixes it.

## Description

The expensive half of the rename T-073 starts: moving the name from prose onto disk, so the
installed skill is `brine` rather than `ticket-flow` everywhere an agent or a human can see
it.

**This ticket is not a `sed`, but it is not a migration either — refinement settled that.**
The occurrences that matter are not the 93 lines across 27 files in this repo — they are the
artifacts already written into other people's projects by a prior `pickle install`:

- `.agents/skills/ticket-flow/` — the directory itself, whose `SKILL.md` resolves its own
  `resources/…` references relative to it;
- `.claude/skills/ticket-flow` — the symlink Claude Code reads;
- `SKILL.md` frontmatter `name: ticket-flow`, which *is* the agent-visible skill name;
- the `AGENTS.md`/`CLAUDE.md` marker-block body;
- `agents/opencode/opencode.jsonc` and both `docs-readability.ts` extensions;
- this repo's own `.agents/skills/ticket-flow → skill/` symlink and the hardcoded paths in
  its `AGENTS.md`, which must be changed **by hand inside this ticket's diff** — the
  self-modify policy forbids running `pickle install|upgrade` against this repo from a
  feature branch.

**The drop-or-do question is settled: do it, in full.** T-073's standing recommendation was
to keep descriptive ids on disk, but that leaves three names in play for two things —
`pickle` the tool, `brine` the flow, and `ticket-flow` the directory the flow lives in. The
third name is the cost, and it is paid on every read. Pre-1.0 (tags stop at `v0.2.2`, and
`CHANGELOG.md` explicitly permits breaking changes below 1.0) is the cheapest this will ever
be.

**What is *not* in scope is a migration.** No `os.Rename`, no dual-name predicates, no
deprecation window. Instead a **legacy sweep**: the legacy skill directory is pickle-owned —
`Upgrade` already `RemoveAll`s exactly that path, and `Uninstall` already has the
symlink-vs-directory removal logic — so `upgrade` and `uninstall` delete the old paths and
everything else keeps looking at exactly one name. `doctor` gains a single check that
**errors** when a legacy path is still present, naming `pickle upgrade` as the fix.

Refinement rejected the alternative of shipping nothing and telling users to
uninstall-then-reinstall, because the failure mode is silent rather than manual. On the
documented path (`pickle upgrade`), a renamed `SkillDir` with no sweep leaves
`.agents/skills/ticket-flow/` in place holding a **stale** payload — natively discovered by
pi, opencode and Zed alongside the new one — while the claude-link branch
(`install.go:326`, keyed on an `Lstat` of the *new* link) creates nothing, so Claude Code
sees only the stale copy. `doctor` then reports all green, because it only looks at the new
name. The sweep is ~30 lines and less code than that outcome's release notes.

One breakage is **accepted and documented rather than detected**: `opencode.jsonc` hardcodes
`{file:./.agents/skills/ticket-flow/resources/docs-readability.prompt.md}` and is user-owned
after creation — pickle never merges JSONC, so neither `upgrade` nor a reinstall repairs it.
`doctor` deliberately performs no opencode checks and this ticket does not change that; the
CHANGELOG and the manual state the one line to edit by hand.

Two lineage notes still live: T-046 landed first, so `install.SkillLinked` and its two call
sites come along with the rename (they stay single-name — see decision 2), and its self-host
contract is why a legacy path that is a *symlink* is re-linked at the new name rather than
deleted and re-copied (decision 5). T-073's review finding **F5** is folded in here: its
"don't touch `agents/**`, `doctor` byte-compares it" rationale has since been overtaken by
T-096, which edited those same files anyway — that drift is the normal post-release state
`pickle upgrade` clears, and `checkAgentScaffolds` never looks at `opencode.jsonc` at all.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                     # pickle is the overarching project and its own child
git checkout main
git checkout -b feat/T-074-rename-skill-dir-to-brine
```

Local WIP commits as work progresses; publish only after user approval (finalize, push, open
the MR — merging is the human's), per the project's configured commit policy. This child is
root-path, so tidy the WIP commits into atomic ones and keep that history rather than
squashing.

### Prerequisite gate (hard)

- **T-073 in `6-done/` and merged to `main`** — satisfied (merged, PR referenced in T-073's
  History). This ticket edits the same marker block and docs attributes T-073 introduced.
- Clean worktree before starting.
- **Never run `pickle install|upgrade|uninstall` against this repo** from the feature branch
  (`AGENTS.md` self-modify policy). Every change to this repo's own symlinks, `AGENTS.md`
  marker block and agent scaffolds is made **by hand inside this diff**; end-to-end runs go to
  a throwaway dir with the binary copied in.

### Confirmed design decisions (do not deviate without asking)

1. **Full rename on disk.** `.agents/skills/brine`, `.claude/skills/brine` (target
   `../../.agents/skills/brine`), and `skill/SKILL.md` frontmatter `name: brine` — so the
   agent-visible invocation becomes `/skill:brine`. No third name survives anywhere a user or
   an agent can see one.
2. **No migration — a legacy sweep instead.** Nothing is renamed or moved. `Upgrade` and
   `Uninstall` *remove* the legacy paths; every other consumer keeps looking at exactly one
   name. In particular `install.SkillLinked`, `doctor.checkSkill` and `doctor.checkVersion`
   stay single-name: do **not** add a dual-name predicate or a "which name is installed"
   resolver.
3. **No deprecation window, no dual-name reporting.** `doctor` errors (not warns) when a
   legacy path is present, with `pickle upgrade` as the stated fix. The legacy constants and
   the sweep live in one clearly-marked block, annotated for deletion at 1.0.
4. **`opencode.jsonc`'s dangling prompt path is accepted breakage: documented, not detected.**
   `doctor` performs no opencode checks and this ticket does not add any. The CHANGELOG entry
   and `cli-reference.adoc` state the single line an opencode user edits by hand.
5. **A legacy path that is a symlink is re-linked at the new name, never deleted-then-copied.**
   T-046's self-host contract (`SkillLinked`: install/upgrade never overwrite *through* a link,
   uninstall removes the link and not its target) must survive the sweep — deleting a
   self-host link and letting `copyPayload` write a real directory would silently convert such
   a checkout into an installed copy.
6. **Docs go through the existing attribute.** Change `:skill-dir:` in `docs/attributes.adoc`
   once and route the remaining literal occurrences through it rather than hand-editing each.
7. **`DESIGN.md` keeps its historical prose.** It is an origin doc kept for the *why*. Only
   the locked decision that this ticket falsifies ("Skill name stays `ticket-flow`",
   `DESIGN.md:293`) gets a one-line superseded-by pointer; the narrative is not rewritten.
8. **Historical CHANGELOG entries are not rewritten** — they describe what older versions did.
   Only a new `## [Unreleased]` entry is added.

### Tasks

#### Task 1 — rename the constants, add the marked legacy block (`internal/install/install.go`)

- Lines 43–47: `SkillDir = ".agents/skills/brine"`, `ClaudeSkillLink = ".claude/skills/brine"`,
  `ClaudeSkillTarget = "../../.agents/skills/brine"`.
- Add, in its own block with a comment saying it is deletable at 1.0 (T-074):
  `LegacySkillDir = ".agents/skills/ticket-flow"`,
  `LegacyClaudeSkillLink = ".claude/skills/ticket-flow"`.
- Update `SkillLinked`'s doc comment (`:58-71`) to name the new path. **Behaviour unchanged**
  (decision 2).
- Package doc comment (`:1`) — "the ticket flow" → brine's name (F5 residue, Task 7).

#### Task 2 — `sweepLegacySkill` (`internal/install/install.go`)

One helper, used by both mutators:

```go
// sweepLegacySkill removes the pre-brine install paths … delete at 1.0 (T-074).
func sweepLegacySkill(root string, dryRun bool, res *Result) (linkTarget string, found bool, err error)
```

- `Lstat(LegacySkillDir)`: symlink → capture `Readlink` into `linkTarget` and `os.Remove` the
  link only (never `RemoveAll` — decision 5); real directory → `os.RemoveAll`; absent → skip.
- `Lstat(LegacyClaudeSkillLink)` → `os.Remove`, mirroring `Uninstall`'s existing handling at
  `:427-437`.
- Under `dryRun`, record `" (dry-run)"` labels and mutate nothing, matching `Uninstall`'s
  existing convention.
- Report each removal via `res.removed(...)`; return `found` so callers can branch.

#### Task 3 — wire the sweep into `Upgrade` (`internal/install/install.go:296`)

- Call `sweepLegacySkill(root, false, &res)` **before** the `SkillLinked`/`RemoveAll` block, so
  the new-name copy is written into a tree with no stale duplicate.
- When the sweep returned a `linkTarget` (a self-host legacy link), re-create the symlink at
  `SkillDir` pointing at the same target via `ensureSymlink`, then let `copyPayload` skip it
  through its existing `ModeSymlink` guard (decision 5).
- Fix the Claude relink so an upgraded legacy install is not left without a view: the branch at
  `:326` must fire when the *new* link exists **or** the sweep removed the legacy one.

#### Task 4 — wire the sweep into `Uninstall` (`internal/install/install.go:409`)

Call `sweepLegacySkill(root, opts.DryRun, &res)` alongside the existing new-name removal, so a
new binary can still fully remove an install made by an old one.

#### Task 5 — `doctor`: the legacy check (`internal/doctor/doctor.go`)

- New `checkLegacyPaths(root string, r *Result)`, called from `Check` (`:51-62`) **before** the
  cfg-dependent checks (it is a pure filesystem read and must run even when `pickle.toml` fails
  to parse).
- Either legacy path present → `r.errf` naming the legacy path, the new path, and
  ``run `pickle upgrade` ``. Error, not warning: a tree carrying both is serving a stale skill.
- Neither present → a passed line (visible under `-v`, consistent with T-046's skip line).
- Update the doc comments in `checkSkill` (`:84-88`) and `checkVersion` (`:345-361`) only where
  they spell the old path in prose.

#### Task 6 — the payload and the marker block

- `skill/SKILL.md`: frontmatter `name: brine` (line 2) plus the two path mentions (`:53`,
  `:303`).
- `MarkerBlock` (`internal/install/install.go:915,918` and the rules-path trailer at
  `:962-965`) → new paths.
- Regenerate `internal/install/testdata/markerblock.golden` (`:8,:11`).

#### Task 7 — F5 residue: agent scaffolds + Go package comments

Paths in the shipped scaffolds, and "ticket flow"/"ticket-flow skill" prose:

- `agents/opencode/opencode.jsonc:3,4,32,42` (`:32` is the prompt path).
- `agents/pi/extensions/docs-readability.ts` (5 occurrences, incl. the `PROMPT_PATH` constant —
  note T-096 shifted these line numbers, search the text).
- `agents/pi/extensions/pickle-guardrails.ts:5`.
- Package comments: `main.go:2`, `assets.go:7,10`, `internal/install/install.go:1`,
  `internal/audit/audit.go:1`.

#### Task 8 — tests

- Update the path literals: `internal/install/install_test.go:38,39,48,49,141,187`,
  `internal/cli/agents_test.go:47,48,64`, `internal/install/agents_test.go:58`,
  `internal/doctor/doctor_test.go:39`.
- New `internal/install` tests for `sweepLegacySkill` via `Upgrade`/`Uninstall`: legacy real
  directory removed; legacy symlink removed **and re-linked at the new name with its target
  intact, and the linked tree untouched** (decision 5); absent legacy = no-op; `Uninstall`
  `--dry-run` lists both legacy paths and mutates nothing; a legacy Claude link yields a new
  `.claude/skills/brine` after upgrade.
- New `internal/doctor` tests: legacy directory present → error naming `pickle upgrade`; legacy
  Claude link present → error; neither → passed line; the check still runs when `pickle.toml`
  is unparseable.

#### Task 9 — docs

- `docs/attributes.adoc:6` → `:skill-dir: .agents/skills/brine`; route the remaining literals
  through `{skill-dir}`: `cli-reference.adoc` (12), `concepts/project-structure.adoc` (7),
  `configuration.adoc`, `quickstart.adoc`, `your-first-project.adoc`.
- `cli-reference.adoc`: document the sweep under `upgrade` and `uninstall`, the new `doctor`
  error, and the accepted `opencode.jsonc` breakage with the exact line to hand-edit
  (decision 4).
- `CHANGELOG.md` `## [Unreleased]` → `### Changed`: the rename, the sweep, the `doctor` error,
  the `/skill:brine` invocation change, and the `opencode.jsonc` manual fix — flagged as a
  pre-1.0 breaking change (T-074).
- `DESIGN.md:293`: one-line superseded-by pointer (decision 7).

#### Task 10 — this repo's own artifacts, by hand (self-modify policy)

- `git mv .agents/skills/ticket-flow .agents/skills/brine` (a symlink to `../../skill`); replace
  `.claude/skills/ticket-flow` with `.claude/skills/brine → ../../.agents/skills/brine`.
- `AGENTS.md`: the out-of-block lines `:13,:27` and the marker-block body `:42,:45` — the latter
  must mirror `MarkerBlock()`'s new output byte-for-byte, since `doctor` drift-checks it.
- This repo's `opencode.jsonc:15,25`, `.pi/README.md:24,32,37` (incl. `/skill:brine`),
  `.opencode/README.md:36,40`, and the `.pi/extensions/docs-readability.ts` copy.

### Acceptance test

1. **Build/validate green:**
   ```
   just build && just test && just lint && just docs-check
   ```

2. **No stale name outside its two sanctioned homes** — the only surviving matches must be the
   `Legacy*` constants block, the `doctor` legacy check and its tests, `CHANGELOG.md`'s historical
   entries, and `DESIGN.md`'s historical prose:
   ```
   grep -rn "skills/ticket-flow" --include='*.go' --include='*.ts' --include='*.jsonc' \
     --include='*.adoc' --include='*.md' --include='*.golden' . | grep -v '^./tickets/'
   ```

3. **Self-host links renamed:**
   ```
   test -L .agents/skills/brine && readlink .agents/skills/brine        # ../../skill
   readlink .claude/skills/brine                                        # ../../.agents/skills/brine
   grep -c "name: brine" skill/SKILL.md                                 # 1
   ```

4. **Fresh install, throwaway dir** (never against this repo):
   ```
   D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q .
   ./pk install --project demo --agent claude,opencode,pi
   test -f .agents/skills/brine/SKILL.md && readlink .claude/skills/brine
   ! test -e .agents/skills/ticket-flow
   ./pk doctor -v          # exit 0; shows the "no legacy skill path" passed line
   ```

5. **Legacy install swept by `upgrade`** (same throwaway dir, simulating an old install):
   ```
   mv .agents/skills/brine .agents/skills/ticket-flow
   rm .claude/skills/brine && ln -s ../../.agents/skills/ticket-flow .claude/skills/ticket-flow
   ./pk doctor;  echo "exit=$?"     # exit 1, error names the legacy path and `pickle upgrade`
   ./pk upgrade                    # reports both legacy paths removed
   ! test -e .agents/skills/ticket-flow && ! test -e .claude/skills/ticket-flow
   test -f .agents/skills/brine/SKILL.md && readlink .claude/skills/brine
   ./pk doctor;  echo "exit=$?"     # exit 0
   ```

6. **Self-host legacy link is re-linked, not copied** (decision 5):
   ```
   rm -rf .agents/skills/brine && mkdir -p payload/resources
   cp <a SKILL.md> payload/ && cp <a tickets-README.md> payload/resources/
   ln -s ../../payload .agents/skills/ticket-flow
   ./pk upgrade
   test -L .agents/skills/brine                     # still a link…
   readlink .agents/skills/brine                    # …to ../../payload
   test -f payload/SKILL.md                         # the linked tree survived
   ```

7. **`uninstall` removes an old install, and `--dry-run` lies about nothing:**
   ```
   ./pk uninstall -n | grep ticket-flow             # both legacy paths listed
   test -e .agents/skills/ticket-flow               # still there after the dry run
   ./pk uninstall && ! test -e .agents/skills/ticket-flow
   ```

8. **Board invariants:** `./pickle board audit` exits 0.

### Docs update (mandatory when user-facing)

User-facing, and covered by Task 9: `docs/attributes.adoc` (the `:skill-dir:` value),
`docs/user-manual/cli-reference.adoc` (install/upgrade/uninstall/doctor, plus the accepted
`opencode.jsonc` breakage and its hand-fix), `docs/user-manual/concepts/project-structure.adoc`
(the tree diagram and the ownership table), `configuration.adoc`, `quickstart.adoc`,
`your-first-project.adoc`, a `## [Unreleased]` → `### Changed` CHANGELOG entry flagged as a
pre-1.0 breaking change, and the superseded-decision pointer in `DESIGN.md`. This repo's own
`.pi/README.md` and `.opencode/README.md` are updated in Task 10. `just docs-check` must pass.

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated and registered; CHANGELOG entry added.
3. Write a **summary**: files touched, the sweep's shape, what was deliberately *not* built
   (no migration, no dual-name support, no opencode detection) and the accepted breakage.
4. Suggest a **Conventional Commit message**, e.g.:

   ```
   feat(install)!: rename the installed skill directory to brine (T-074)

   The skill installs at .agents/skills/brine (.claude/skills/brine, name: brine,
   invoked as /skill:brine), leaving one name per thing. upgrade and uninstall sweep
   the pre-brine paths instead of migrating them, and doctor errors while one is still
   present. A user-modified opencode.jsonc keeps a dangling prompt path and must be
   edited by hand — documented, not detected.
   ```

5. **Tidy up before presenting** — root-path child: interactive-rebase the WIP commits into a
   small number of atomic, correctly typed/scoped commits (suggested split: the rename +
   sweep + doctor check; the F5 prose residue; docs + CHANGELOG; this repo's self-host
   artifacts).
6. Commit locally on the ticket branch; publish only per the commit policy. Present the commit
   message; only after approval finalize (keep the tidied history — the root-path default).
   Before pushing, verify the remote base is not behind:
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
   print nothing, or push `origin main` first. Then push and open the merge request — merging is
   always the human's. Hand back to the user.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — patched by T-073's review impact sweep (step 8). T-073 shipped, so the prose half
  of the rename is **done**: the `flow` config key (default `"brine"`), `pickle flow show|list`,
  a `doctor` passed-line, and the name in `SKILL.md`, `tickets-README.md`, `MarkerBlock()`, this
  repo's own `AGENTS.md`/`pickle.toml`/`README.md`, and the manual (via a `:flow:` attribute).
  Two consequences for this ticket: (1) the in-repo occurrence count in the Description is now
  lower than the "~85" it cites — re-count at refinement rather than trusting it; (2) T-073's
  review folded finding **F5** in here — `agents/opencode/opencode.jsonc:3,4,42`,
  `agents/pi/extensions/*.ts`, `.pi/extensions/docs-readability.ts:49` and the Go package
  comments in `main.go:2`, `assets.go:7`, `internal/install/install.go:1` still say "ticket
  flow"/"ticket-flow skill". They were deliberately left alone because `agents/**` is
  byte-compared by `doctor`, so renaming them without this ticket's upgrade path would raise a
  drift warning in every already-installed project. This ticket already lists those scaffolds in
  its scope; the Go comments are the new part
- 2026-08-12 — patched by T-096's review impact sweep. T-096 repinned the docs-readability
  model and genericised the surrounding prose in both `docs-readability.ts` copies and both
  `opencode.jsonc` files — all four of which this ticket lists as rename targets. Nothing here
  is invalidated (T-096 touched the *model* and the word "Gemini", never the words "ticket
  flow"/"ticket-flow skill" this ticket renames), but the folded-F5 line references above have
  drifted: the `ticket-flow skill` comment cited at `.pi/extensions/docs-readability.ts:49` is
  now at `:50`, since T-096 added a line to that doc comment. Search the text rather than the
  line numbers at refinement — as the note above already says for the "~85" occurrence count
- 2026-08-12 — patched by **T-046's review impact sweep**: T-046 has landed, so this ticket is the
  one that rebases. Concretely it added `install.SkillLinked(root) bool` next to the `SkillDir`
  constant in `internal/install/install.go` — a **third** consumer of that constant's value, read
  by `Upgrade`'s skill-refresh guard and by `internal/doctor`'s `Check` (which passes the result
  into `checkSkill` and `checkVersion`). The rename must carry `SkillLinked` and both call sites
  with it, and the dual-name `doctor` this ticket plans has to decide what the predicate means
  while *both* names may exist — a link under the old name and a real dir under the new one is
  exactly the migration state. Not a conflict, one more call site, as T-046's own Description
  predicted
- 2026-08-13 — refined. The drop-or-do question the Description reserved for refinement is
  settled as **do it in full**: three names (`pickle`, `brine`, `ticket-flow`) for two things is
  the cost that decides it, overriding T-073's standing recommendation to keep descriptive
  on-disk ids. The **migration is cut** in favour of a ~30-line legacy sweep in
  `upgrade`/`uninstall` plus one `doctor` error — no `os.Rename`, no dual-name predicate (so
  T-046's `SkillLinked` and its two call sites stay single-name), no deprecation window. The
  alternative of shipping nothing and documenting uninstall→reinstall was rejected: on the
  documented `pickle upgrade` path it leaves a stale duplicate skill that Claude Code prefers
  and `doctor` reports as green. `opencode.jsonc`'s dangling prompt path is accepted breakage,
  documented and not detected. Occurrence count re-counted per the note above: **93 lines across
  27 files**, not ~85. Re-graded complexity high → medium and cost L → M, since the migration
  was the multi-session part
- 2026-08-13 — TO DO → READY: plan complete: full rename, legacy sweep instead of migration
- 2026-08-13 — pickup applicability gate (fresh sub-agent audit): 0 blocking, 2 non-blocking,
  both noted and closed. (1) AGENTS.md drifted the same day as refinement (T-098 landed a new
  paragraph): Task 10's `:42,:45` marker-body citation is stale — the block now starts at line
  52 and its `ticket-flow` mentions are at `:60,:63`; the out-of-block citations `:13,:27` are
  still correct. Harmless: Task 10 is a by-hand edit per the self-modify policy, so the
  implementer reads the live file rather than trusting the line numbers. (2) Task 5's
  instruction to fix doc-comment prose in `doctor.go`'s `checkSkill`/`checkVersion` is vacuous —
  neither spells the old path literally, both go through `install.SkillDir`. No other line
  number, helper name or occurrence count in the plan has drifted; `just build/test/lint/
  docs-check` all pass on current `main`; WIP clear (0/1 both gates); no ticket landed since
  refinement touches the same files at the cited spots
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
