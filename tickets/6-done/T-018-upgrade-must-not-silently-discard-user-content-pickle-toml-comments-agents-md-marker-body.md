---
id: T-018
title: upgrade must not silently discard user content (pickle.toml comments, AGENTS.md marker body)
project: pickle
depends-on: []
impact: high
complexity: medium
cost: M
---

# T-018 — upgrade must not silently discard user content (pickle.toml comments, AGENTS.md marker body)

## Description

Non-blocking follow-up from the T-006 review. `pickle upgrade` shipped as specified (decision D5
deliberately accepted `pickle.toml` normalisation), but in practice it destroys two kinds of
hand-written content without warning. Both are pre-existing behaviours of the install/marker
machinery; T-006 makes them far more reachable by shipping a command whose *entire job* is
refreshing them.

1. **`pickle.toml` comments are silently dropped.** On a version change `Upgrade` calls
   `cfg.Save("")` (`internal/install/install.go:154-158`), which re-renders the file through
   `config.Render` — a canonical writer that emits no comments. This repo's own `pickle.toml`
   is 15 lines of hand-written rationale (`pickle.toml:1-13`) explaining the bootstrap
   chicken-and-egg; a single `pickle upgrade` here erases all of it. Decide and implement:
   preserve comments (round-trip-preserving edit of just the `payload_version` line), or warn
   before rewriting, or at minimum have `Render` re-emit a managed header. **The docs half of
   this is being fixed as a blocking T-006 rework finding** (`README.md:94-95` wrongly enumerates
   only `project add|remove` as normalising) — this ticket is about the *behaviour*.

   **Added 2026-07-24 (T-006 scoped re-review): the same error of fact ships inside the generated
   file itself.** `config.Render` hardcodes the header
   `# Managed by pickle. Hand-edits are preserved on load but normalised to this layout on the
   next pickle project add|remove.` (`internal/config/config.go:196-198`, from T-001 `3dc0c26`).
   That is the *identical* wrong enumeration T-006's rework just corrected in the README — but in
   a surface that ships into **every** installed project, is the first thing a user reads in their
   own `pickle.toml`, and appears in the very file whose comments were just erased, while saying
   nothing about comment loss. Proven by a mutation run of this repo's own tree: one `upgrade`
   replaced 16 lines of hand-written rationale with that header. Kept non-blocking for T-006
   because the README (the surface a user consults) is now correct, the string predates T-006 and
   is in a package T-006 never touched, and whichever remedy this ticket picks *changes this header
   anyway* — fixing it in T-006 would mean writing it twice. Whatever is chosen, the header must
   end up enumerating every writer (`install`, `upgrade`, `project add|remove`) and stating the
   comment contract truthfully.

2. **Hand-maintained content inside the `AGENTS.md` marker block is silently replaced.**
   `install.go:22-23` declares everything between `<!-- pickle:begin -->`/`<!-- pickle:end -->`
   pickle-managed and replaced on re-run, but this repo's `AGENTS.md` keeps genuinely
   repo-specific bullets *inside* that region that `markerBlock` (`install.go:515-544`) does not
   emit: the child's **Commands** (`just build`/`just test`/`just lint`), the **WIP limits**, the
   `skill/` self-host symlink note, and the publish-gate wording. Verified during review:
   grepping `markerBlock` for `just build`/`wip_in_development` returns **0** matches, so
   `pickle upgrade` in this repo would delete them, and `doctor.checkMarkers`
   (`doctor.go:111-137`) only checks marker *presence*, so the loss is invisible. Fix by either
   teaching `markerBlock` to render the child's commands + WIP limits from `pickle.toml` (making
   the block genuinely regenerable), or moving the repo-specific bullets outside the markers, or
   both. Mitigation already in place for the dev workspace only:
   `opencode.jsonc:63` gates `pickle upgrade*` behind `ask`, and `.pi/extensions/workspace-guardrails.ts:99`
   has a self-modify guard — neither protects an end user's project.

3. **While in `markerBlock`: one stale conditional the refresh now propagates.**
   `install.go:544` still reads "Prefer `pickle ticket move` **once available**; otherwise do the
   three edits by hand" — `ticket move` shipped in T-007, and the skill payload states it
   unconditionally (`skill/SKILL.md:93`). Because `upgrade` rewrites this block in every
   installed project, the stale sentence is now actively propagated. Re-word it.

Soft coupling: T-013 (install polish) touches the same `injectMarker`/summary surface; keep the
changes disjoint or sequence them. **T-020** (doctor marker-drift detection) was split out of
this ticket during refinement and should be refined only after this one lands.

**Re-graded 2026-07-25: impact `medium` → `high`.** Every other open ticket is polish,
consolidation, or new surface area; this is the only one where a **shipped** command destroys
user-authored content by default, in every installed project, with no warning before and no
report after. `pickle upgrade` is also the command a user is most likely to run repeatedly and
trust blindly. Complexity `medium` and cost `M` stand: three bounded edits (a line-level config
writer, a `markerBlock` that reads config, a docs pass) with the design questions now closed.

### Refinement note — 2026-07-25 (re-verified against `main` @ `fedfcc8`, post-T-006 merge)

All refs above still resolve. Two corrections and one addition:

- `markerBlock` is at `internal/install/install.go:506+` (not `:515-544`); the stale board-rule
  sentence of item 3 is its **last line**, now reading "Prefer `pickle ticket move` once
  available; otherwise do the three edits by hand".
- **Item 1's error of fact has a *third* instance**, in the same file: the `config` package doc
  (`internal/config/config.go:4-8`) also says the file is "normalised to the canonical layout on
  the next mutation (project add/remove)". `rg 'project add/remove|project add\|remove'` returns
  exactly three hits — `README.md:96` (correct, fixed by T-006), `config.go:8` and `config.go:198`
  (both wrong). All three must agree after this ticket, which *changes the truth again*: once
  `upgrade` stops re-rendering, the honest enumeration is `install` and `project add|remove`.
- **Exact loss measured**, by running the merged binary against a throwaway clone of this repo
  (`pickle upgrade`, then `git diff`): `AGENTS.md` **+16/−21** lines (corrected 2026-07-25 by the
  applicability gate, finding N8 — the figures first recorded here, −37/+32, were wrong; every
  *qualitative* claim below was re-confirmed). Destroyed: the whole
  **Commands** bullet (`just build`/`just test`/`just lint`), the whole **WIP limits** bullet,
  the self-host `skill/`-symlink note, the concrete `feat/T-NNN-<slug>` prefix (→ literal
  `<branch_prefix>`), the "This repo has a single child … at the repo root" prose, and the
  publish-gate's "after approval, finalize (squash or keep history) + push + open the MR" clause.
  `pickle.toml`: **18** comment lines → the 3-line generated header (also corrected by N8; the
  acceptance test's `grep -c '^#'` baseline is therefore **18**). Commands, WIP limits and branch
  prefix are all **in `pickle.toml`** and therefore renderable; only the self-host note is
  genuinely un-derivable.

### Applicability-gate audit — 2026-07-25 (fresh sub-agent, before pickup)

Verdict **PROCEED-WITH-CORRECTIONS**. The core thesis (surgical `payload_version` edit +
config-driven `markerBlock`) was verified TRUE, REQUIRED and WORTH IT. Eleven assumptions came
back clean — every line reference in this ticket resolves; `cfg.Path()` exists
(`internal/config/config.go:103-104`); `Save(path string)` falls back to `c.path` so `Save("")`
is valid; `markerBlock` is a pure `func(*config.Config) string` (`install.go:506`) with every
field D4/D6 needs already present and defaulted; the marker text is duplicated nowhere else; and
**no existing test breaks** (all marker assertions are presence-only, and `TestUpgrade`'s fixture
takes the happy path). The absence of `tickets/3-in-development/` in the working tree is *not* a
defect: `internal/ticket/ticket.go:305-313` explicitly tolerates vanished empty dirs and
`internal/move/move.go:122` recreates the destination.

Findings routed into the plan (all with user sign-off the same day):

- **B1 (blocking)** → decision 8, rewritten. Would have shipped a false statement.
- **N3** → decision 3, superseded: insert instead of falling back destructively.
- **N7** → decision 4, refined: one line per child, always.
- **N9** → decision 10, added: document the skill-dir ownership contract.
- **N1** → Task 2: the sketch does not compile (`err` already declared at `install.go:116`;
  `errors` not imported).
- **N2** → Task 1: "mirror `Save`'s write strategy" is false — `Save` is a bare `os.WriteFile`
  with a hard-coded `0o644` (`config.go:228-236`), neither atomic nor mode-preserving. The new
  writer does it properly; `Save`'s own weakness is handed to **T-012**.
- **N5** → Task 4: byte-identity needs the entire in-marker body regenerated (the current text
  also differs from `markerBlock` by paragraph re-wrapping and a missing sentence), and
  **committed**, since the acceptance test clones `HEAD`.
- **N6** → acceptance test step 4 invoked `./pickle` in a temp dir containing no binary, masked
  by `|| true`; it verified nothing.
- **N8** → the measured figures above.
- **N4** (`Result` has no warning channel) is moot once N3 removes the fallback.
- **N10** (`injectMarker` hard-codes `0o644` on `AGENTS.md`/`CLAUDE.md`, `install.go:418,437,451,498`)
  is the same family of silent user-state loss but belongs to **T-013**'s surface — deliberately
  not touched here.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle`'s sole child is the repo root (`.`), so the branch is cut in this repo:

```
git checkout main
git checkout -b feat/T-018-upgrade-preserve-user-content
```

Local WIP commits encouraged; **no push / MR without explicit user approval** (child is
publish-gated — see `pickle.toml` / `AGENTS.md`).

### Prerequisite gate (hard)

- **T-006 must be merged** — it is: `4bcfc00` on `main` (board `merged` column ticked). This
  ticket edits `Upgrade`, which T-006 introduced.
- No `depends-on:`. Soft couplings, all currently in `1-to-do/` and none required first:
  **T-012** (config hardening — owns `Render`'s TOML escaping; this ticket adds a *second* writer
  path to the same file, so whoever lands second re-reads the other's changes), **T-013** (install
  polish — same `injectMarker`/`Result` surface), **T-017** (unify marker-pair detection — if it
  lands first, reuse its `markerSpan` helper instead of scanning by hand), **T-020** (doctor drift
  detection — split out of this ticket; refine it *after* this lands).
- Clean working tree on the new branch.

### Confirmed design decisions (user sign-off 2026-07-25 — do not deviate without asking)

1. **`pickle.toml`: surgical `payload_version` edit, not a re-render (D1).** `upgrade` must
   rewrite **only** the `payload_version = "…"` line, leaving every other byte — comments,
   blank lines, key order, alignment — untouched. `project add|remove` keep the canonical
   `Render` path, because they change structure. **Rejected alternatives:** warn-before-rewrite
   (content still dies by default), a full comment-preserving TOML round-trip (new dependency /
   hand-rolled parser, overlaps T-012), docs-only (leaves the destruction in place).
2. **No new dependency.** BurntSushi/toml decodes only; the surgical edit is line-based text
   manipulation in `internal/config`. Do **not** swap the TOML library.
3. **~~Fallback is explicit, never silent (D1a).~~ SUPERSEDED 2026-07-25 by the applicability
   gate (finding N3), user sign-off same day: there is no fallback at all.** `Upgrade` only
   reaches the write after `config.Load` **and** `Validate` succeeded, and `Validate` requires at
   least one `[[project]]` — therefore the file always contains at least one table header, and a
   missing `payload_version` can always be **inserted** immediately before the first table header
   while preserving every other byte. No code path may re-render the file. This deletes
   `ErrNoPayloadVersionLine`, the fallback branch in Task 2, the "file normalised" `Result`
   string, and the corresponding test cases. (The superseded fallback was the implementer's
   addition, not a user requirement.)
4. **`markerBlock`: render everything derivable from `pickle.toml` (D2).** Commands
   (`build`/`test`/`lint`), WIP limits (`wip_in_development`/`wip_in_review`) and `branch_prefix`
   are all in the config and must be emitted, so the block is genuinely regenerable. Multi-child
   projects must render correctly (per-child lines) — this repo's single child must not become a
   hidden assumption. **Refined 2026-07-25 by the gate (N7), user sign-off:** render **one line
   per child, always** — no "collapse when all children agree" branch. That branching has no
   current caller (one child exists, no two-child fixture) and would widen the byte-identity
   surface the acceptance test depends on.
5. **Relocate what is not derivable (D2a).** The self-host `skill/`-symlink note and the
   "this repo's sole child is the repo root, tickets describe the `pickle` binary itself" prose
   move **outside** the marker region in this repo's `AGENTS.md`. Nothing repo-specific stays
   inside a block that upgrade regenerates.
6. **Commit policy is rendered from `[commit]`, not hardcoded.** `markerBlock` currently asserts
   "Child-projects are **publish-gated**" unconditionally; it must read
   `cfg.Commit.ChildPublishGated`/`OverarchingAuto` so it cannot state a policy the project does
   not have. Restore the lost "after approval, finalize (squash or keep history) + push + open
   the MR" clause while there — it is generic flow prose, not repo-specific.
7. **Fix the stale sentence (item 3).** `markerBlock`'s last line must state `pickle ticket move`
   unconditionally (it shipped in T-007), matching `skill/SKILL.md:93`.
8. **All three normalisation statements must end up telling the same, new truth.**
   `config.go:4-8` (package doc), `config.go:196-198` (generated header) and `README.md:94-97`.

   **CORRECTED 2026-07-25 — the gate's one blocking finding (B1), user sign-off same day.** The
   wording signed off earlier ("`install` and `project add|remove` re-render canonically and drop
   comments") is **factually false and must not be shipped**: `install` never re-renders an
   existing `pickle.toml`. `writeConfig` (`internal/install/install.go:346-351`) stats the path
   and returns early with `res.skipped(config.FileName + " (exists)")`; it writes only when the
   file is **absent** (`:369`), where there are no comments to lose. Shipping the old wording
   would write a *new* wrong enumeration into `config.go:197-198` — the header that lands in
   every installed project — which is the exact defect this ticket exists to remove.

   The verified writer inventory (`rg '\.Save\('`, non-test call sites):

   | writer | behaviour |
   |---|---|
   | `install` — `internal/install/install.go:369` | creates the file canonically **only when absent** |
   | `project add` — `internal/cli/project.go:95` | re-renders canonically, drops comments |
   | `project remove` — `internal/cli/project.go:137` | re-renders canonically, drops comments |
   | `upgrade` — `internal/install/install.go:155` | after this ticket: edits only the version line |

   **The truth all surfaces must state:** *`project add|remove` re-render the file to the
   canonical layout and drop comments; `install` writes it only when it does not yet exist;
   `upgrade` edits only the `payload_version` line and preserves everything else.*

   Note this also means `README.md:94-97` — the paragraph T-006's rework "corrected" — is
   **still wrong today** (`:95-96` lists `install` and `upgrade` among the re-renderers). It is
   corrected here, not merely updated for the new behaviour.
9. **Drift *detection* is out of scope** — split to **T-020** by user decision. This ticket only
   stops the destruction.
10. **Added 2026-07-25 (gate finding N9, user sign-off): document the skill-dir contract.**
    `Upgrade` does `os.RemoveAll` on `.agents/skills/ticket-flow` (`internal/install/install.go:124-129`)
    when it is a real directory rather than a symlink, deleting any file a user kept there, with
    no `Result` entry naming it. That is intentional (`README.md:190-191`) but unstated as a
    contract — and it is literally this ticket's title. Add **one sentence** to the docs: the
    skill directory is pickle-owned and regenerated wholesale; hand-written notes do not belong
    there. No behaviour change.

### Tasks

#### Task 1 — surgical version writer (`internal/config/config.go`)
Add, next to `Save`:

```go
// SetPayloadVersionInPlace sets payload_version in the file at path, preserving
// every other byte (comments, blank lines, spacing, key order). The key is
// replaced where it exists and inserted before the first table header where it
// does not, so no caller can lose hand-written content.
func SetPayloadVersionInPlace(path, version string) error
```

- Scan lines; match the **first** top-level `payload_version` assignment: a line whose trimmed
  form starts `payload_version` followed by optional spaces and `=`. **Skip commented lines**
  (`#` first non-space) and **stop at the first table header** (a trimmed line starting `[`) so a
  `payload_version` inside `[[project]]` can never be hit.
- Replace with `payload_version = "<version>"`, preserving the original leading whitespace and
  any trailing inline comment on that line.
- **No match → insert** `payload_version = "<version>"` immediately before the first table
  header (decision 3). If the file somehow has no table header, append at end of file. Never
  re-render, never return a "cannot do it" error for this case; there is no
  `ErrNoPayloadVersionLine`.
- Escape `version` the same way `Render` does (see T-012 item 2 — reuse whatever escaping
  helper exists; if none, quote via `strconv.Quote`).
- Write back **atomically** (temp file in the same directory + `os.Rename`) and **preserve the
  original file mode** (`os.Stat` first). Note (gate finding N2) this does **not** mirror `Save`,
  which is a bare `os.WriteFile(..., 0o644)` — the asymmetry is deliberate and `Save`'s hardening
  is T-012's.

#### Task 2 — use it from `Upgrade` (`internal/install/install.go:150-160`)
Replace the `cfg.PayloadVersion = payloadVersion; cfg.Save("")` pair with:

```go
if err = config.SetPayloadVersionInPlace(cfg.Path(), payloadVersion); err != nil {
    return res, err
}
res.created(config.FileName + " (payload_version -> " + payloadVersion + ")")
```

Note (gate finding N1): use `=`, not `:=` — `err` is already declared at `install.go:116`
(`cfg, err := config.Load(...)`). No `errors` import is needed now that decision 3 removed the
sentinel. `res.created` is the correct existing method for a modification — `install.go:440`
already uses it for `(marker updated)`.

Keep the existing early return for the already-at-version case **unchanged** (it must still not
touch the file at all).

#### Task 3 — `markerBlock` renders the config (`internal/install/install.go:506+`)
Extend the `fmt.Sprintf` block, keeping it a pure function of `*config.Config`:

Per decision 4 as refined by N7: **one line per child, always** — no "collapse when children
agree" branch.

- **Commands bullet** — one line per child, naming it, using the actual values from
  `pickle.toml`. Omit a command that is empty; omit the bullet entirely if no child defines any.
- **WIP limits bullet** — ``3-in-development/` ≤ N · `4-in-review/` ≤ M` from each child's
  values, one line per child.
- **Branch & commit bullet** — render each child's real prefix (`feat/T-NNN-<slug>`), not the
  literal `<branch_prefix>`.
- **Commit-policy bullet** — branch on `cfg.Commit.ChildPublishGated` and
  `cfg.Commit.OverarchingAuto` (decision 6), including the restored finalize/push/MR clause.
- **Board rule** — drop "once available; otherwise" (decision 7).

#### Task 4 — relocate this repo's non-derivable prose (`AGENTS.md`)
Move outside the `<!-- pickle:begin -->`/`<!-- pickle:end -->` region (into the existing intro
prose above it): the self-host `skill/`-symlink note and the "single child at the repo root /
tickets describe the `pickle` binary itself" sentences.

Then (gate finding N5) **replace the entire in-marker body with `markerBlock`'s verbatim
output** — run the freshly built `./pickle upgrade` in this repo and commit the result. Moving
the two prose bits is *not* sufficient for byte-identity: the current body also differs from
`markerBlock` by paragraph re-wrapping of otherwise identical sentences, and `markerBlock` emits
a "Registered child-projects" sentence (`install.go:531-532`) that `AGENTS.md` does not contain.
The regenerated file **must be committed**, because the acceptance test clones `HEAD`.

#### Task 5 — make all three normalisation statements true (decision 8)
- `internal/config/config.go:4-8` — package doc.
- `internal/config/config.go:196-198` — the generated header written into **every** installed
  project.
- `README.md:94-97` — the Configuration paragraph; note it is **wrong today**, not merely stale.

All three must state decision 8's verified enumeration: `project add|remove` re-render and drop
comments; `install` writes the file only when absent; `upgrade` edits only the version line.

#### Task 6 — tests
`internal/config/config_test.go`:
- Round-trip: a file with comments, blank lines, inline comments and a `[[project]]` block →
  `SetPayloadVersionInPlace` → assert the output is **byte-identical except the one line**, and
  that `Load` still parses it with the new version.
- `payload_version` appearing **commented out** and **inside `[[project]]`** → both ignored;
  a genuine top-level key later in the file is still found.
- **No top-level key → inserted before the first table header**, every existing byte preserved,
  and `Load` parses the result with the new version (decision 3).
- Version strings needing escaping (`"`, `\`) survive a `Load` round-trip.
- File mode preserved.

`internal/install/install_test.go`:
- `Upgrade` on a config **with comments** → comments still present afterwards, `payload_version`
  updated (this is the regression test for the whole ticket).
- `Upgrade` on a config with no `payload_version` line → key inserted, comments still intact.
- `markerBlock` renders commands/WIP/branch prefix for a single child **and** for two children
  with differing values; a child with no commands omits the bullet; `child_publish_gated = false`
  renders the non-gated wording.

#### Task 7 — self-host no-op check
After Tasks 3–4, `pickle upgrade` run in a clone of this repo must leave `AGENTS.md`
**byte-identical**. This is the acceptance test's centrepiece, so verify it during development,
not only at the end.

### Acceptance test

Run from the repo root. The centrepiece is that `upgrade` becomes a genuine no-op on this
repo's own tree apart from the single version line:

```
just build
CLONE=$(mktemp -d)/clone
git clone -q --local . "$CLONE"
cp pickle "$CLONE"/
( cd "$CLONE" && ./pickle upgrade >/dev/null )

# 1. AGENTS.md must be untouched (nothing hand-written left inside the markers):
( cd "$CLONE" && git diff --quiet AGENTS.md ) && echo "AGENTS.md: byte-identical OK"

# 2. pickle.toml: exactly one changed line, and it is payload_version; all comments survive:
( cd "$CLONE" && git diff --numstat pickle.toml )            # expect: 1  1  pickle.toml
( cd "$CLONE" && git diff -U0 pickle.toml | grep '^[+-]payload_version' )
( cd "$CLONE" && grep -c '^#' pickle.toml )                  # expect: 18, the pre-upgrade count

# 3. Idempotent: a second upgrade changes nothing further:
( cd "$CLONE" && ./pickle upgrade >/dev/null && git diff --numstat )   # expect: still only pickle.toml 1 1

# 4. A fresh install then upgrade still works end-to-end (N6: invoke the binary by
#    absolute path — nothing is copied into $TMP — and do not mask failures):
BIN=$PWD/pickle
TMP=$(mktemp -d)
( cd "$TMP" && git init -q && "$BIN" install --project demo >/dev/null && "$BIN" upgrade >/dev/null )
echo "install+upgrade rc=$?"                                 # expect: 0
rm -rf "$TMP" "$CLONE"

just test    # incl. the new config + install tests
just lint
./pickle doctor
```

Expected: `AGENTS.md` byte-identical after upgrade; `pickle.toml` differs by exactly the
`payload_version` line with its comment count unchanged; the second upgrade adds no further
diff; `just test`/`just lint`/`doctor` clean.

### Docs update (mandatory when user-facing)

- `README.md:94-97` — Configuration paragraph: `install` and `project add|remove` re-render
  canonically and drop comments; **`upgrade` preserves them** (it edits only the version line).
- `internal/config/config.go:4-8` + `:196-198` — package doc and the generated header, per
  decision 8. The header ships into every installed project, so its wording is user-facing docs.
- `README.md` `## pickle upgrade` section (`:185-207`) — it currently warns that stamping a new
  version "re-renders `pickle.toml` and **drops its comments**" and that marker-block hand-edits
  are replaced. Both statements change: comments now survive, and the marker block no longer
  swallows commands/WIP/branch-prefix. Re-word rather than delete the caution (content outside
  the markers is still the safe place for hand-written notes).
- **Skill-dir contract (decision 10)** — one sentence, near `README.md:190-191`, stating that
  `.agents/skills/ticket-flow/` is pickle-owned and regenerated wholesale by `upgrade`, so
  hand-written files must not be kept there.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` clean.
2. Docs updated (all three normalisation statements agree; README upgrade section re-worded).
3. Write a summary (files touched, decisions honoured, anything deferred — expect T-020 to be
   the named follow-up).
4. Suggested Conventional Commit:

   ```
   fix(install): stop upgrade discarding user content (T-018)

   `pickle upgrade` rewrote pickle.toml through the canonical renderer,
   erasing every comment, and regenerated the AGENTS.md marker block from a
   template that omitted the child's commands, WIP limits and branch prefix —
   so a refresh silently deleted hand-written content.

   Rewrite only the payload_version line in place, inserting it before the
   first table header when absent, so comments, spacing and key order always
   survive; and render commands, WIP limits, branch prefix and the commit
   policy into the marker block from pickle.toml so it is genuinely
   regenerable. Correct the normalisation contract in the package doc, the
   generated pickle.toml header and the README: only project add|remove
   re-render and drop comments.
   ```

5. Commit locally on `feat/T-018-upgrade-preserve-user-content`; **do not push or open an MR
   without user approval**. Present the commit message, then move the ticket to IN REVIEW.

## Implementation summary — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `b990cb7`)

All seven tasks landed in a single commit; `just build`/`just test`/`just lint` clean,
`pickle doctor` 0 errors, `pickle board audit` 20 tickets / 0 errors.

**Files touched (6):**

| file | change |
|---|---|
| `internal/config/config.go` | new `SetPayloadVersionInPlace` + pure `setPayloadVersion`, `valueEnd`, `writePreservingMode`; package doc and the generated header rewritten (Tasks 1, 5) |
| `internal/install/install.go` | `Upgrade` uses the surgical writer; `markerBlock` renders children + commit policy from config (Tasks 2, 3) |
| `AGENTS.md` | non-derivable prose relocated above the markers; in-marker body regenerated by running `./pickle upgrade` (Task 4) |
| `README.md` | Configuration paragraph corrected; `## pickle upgrade` re-worded; skill-dir contract stated (Task 5, decision 10) |
| `internal/config/config_test.go` | 4 new tests (Task 6) |
| `internal/install/install_test.go` | 4 new tests (Task 6) |

**Decisions honoured.** D1 surgical edit; D2 no new dependency (line-based text, `%q` quoting as
`Render` does); D3-as-superseded **no fallback at all** — the key is inserted before the first
table header when absent, so no path re-renders; D4 as refined by N7 — one line per child, always;
D5 relocation; D6 commit policy branches on `cfg.Commit.ChildPublishGated`/`OverarchingAuto`;
D7 the "once available" board-rule sentence is gone; D8 as corrected by B1 — all three surfaces
now state that only `project add|remove` re-render and drop comments, `install` writes only when
absent, `upgrade` preserves; D9 drift detection untouched (T-020); D10 skill-dir contract
documented in the README *and* in the marker block itself.

**Acceptance test — run verbatim, all four steps pass** (on a `--local` clone of `b990cb7`):

1. `git diff --quiet AGENTS.md` → **byte-identical**. The marker block is now a true no-op on
   this repo, which is the whole point: nothing hand-written is left inside it to destroy.
2. `pickle.toml` → `1  1  pickle.toml`, the changed line is `payload_version`, comment count
   **18 before and 18 after**.
3. Second `upgrade` → no further diff.
4. Fresh `install` + `upgrade` in an empty repo → rc 0 (step corrected per N6; it previously
   invoked a non-existent `./pickle` behind `|| true` and verified nothing).

**Tests are mutation-verified, not just green.** Reverting `Upgrade` to `cfg.Save("")` fails
`TestUpgradePreservesConfigComments` (3 comments destroyed, 14→18 lines) and
`TestUpgradeInsertsMissingPayloadVersion`. Removing the table-header `break` fails
`TestSetPayloadVersionInPlaceInsertsWhenAbsent`. That second mutation initially passed, because
the in-table decoy sat *below* a real top-level key and was never reached — the fixture was
changed so the guard is genuinely covered.

**Beyond the plan (small, deliberate):** the marker block also renders each child's `docs`
command when set (same "render everything derivable" rationale as build/test/lint), and carries
one clause naming the skill dir pickle-owned. `SetPayloadVersionInPlace` refuses with an error,
rather than guessing, if `payload_version` uses a multi-line TOML string — a case that cannot be
rewritten safely line-by-line.

**Deferred, with the receiving ticket updated in the same change:**

- **T-020** — doctor drift detection, split out during refinement. Now unblocked: the marker
  block is regenerable, so drift is finally a meaningful signal. Refine it next.
- **T-012 item 7** (new) — `config.Save` is neither atomic nor mode-preserving; route it through
  the `writePreservingMode` helper this ticket added, and apply item 2's TOML-safe escaping to
  both writers.
- **T-013 item 10** (new) — `injectMarker` hard-codes `0o644` on `AGENTS.md`/`CLAUDE.md`
  (gate finding N10): same family of silent user-state loss, but on T-013's surface.

## Review

**Verdict: REWORK — 6 blocking findings.** Reviewed on `feat/T-018-upgrade-preserve-user-content`
@ `b990cb7` (+ `d78fb5d` bookkeeping), 2026-07-25. Three independent sub-agents ran the
implementation, quality and consistency/docs audits; every blocking finding below was then
**re-verified by hand** with the shipped binary or by mutation, because the implementer reviewed
its own work.

**The feature works and the acceptance test passes verbatim** — all 4 steps, plus `just build`,
`just test` (10 packages), `just lint`, `doctor`, `board audit`. `AGENTS.md` is byte-identical
after `upgrade`; `pickle.toml` changes by exactly one line with all 18 comments intact. Tasks
1–4, 6, 7 are met; decisions D1–D7, D9, D10 are honoured; no fourth surface carries the old
normalisation wording; `install`/`project add|remove` are unregressed.

What fails is the **robustness of the new writer relative to the `Save` it replaced**, and one
new **safety inversion** in the marker block. On several legal, currently-loadable inputs the new
code destroys user content or reports success it did not achieve — the precise failure class this
ticket exists to eliminate.

### Findings

| # | Severity | Finding | Evidence |
|---|---|---|---|
| 1 | **blocking** | Omitted `[commit]` table renders "**not publish-gated**" — `AGENTS.md` tells the agent it may push freely | reproduced; see below |
| 2 | **blocking** | Scanner has no TOML lexical state: BOM-on-key-line and multi-line strings corrupt the file | reproduced ×3 |
| 3 | **blocking** | `upgrade` reports `payload_version -> X` when it did not set it — permanently | reproduced |
| 4 | **blocking** | A symlinked `pickle.toml` is replaced by a regular file; the real target keeps the old version | reproduced |
| 5 | **blocking** | Mode-preservation test cannot fail; three more mechanisms are deletable with the suite green | mutation-verified |
| 6 | **blocking** | Two shipped statements are false: the writer's doc comment and `README.md:99-100` | reproduced |
| 7 | non-blocking | `project add\|remove` leave a now-much-staler marker block | → **T-021** |
| 8 | non-blocking | `skill/` states commit policy / branch prefix / WIP unconditionally; can contradict the rendered block | → **T-022** |
| 9 | non-blocking | Board `branch` column is derived from the filename slug, not the ticket's real branch | → **T-023** |
| 10 | non-blocking | Writer hardening: CRLF on insert, comment orphaning, `valueEnd` on arrays, no `fsync`, error messages, read-only file | → **T-012** item 7 (extended) |
| 11 | non-blocking | `res.created` labels an in-place edit "created"; `upgrade --help` omits the `pickle.toml` stamp | → **T-013** item 8 (extended) |
| 12 | non-blocking | Bookkeeping defects introduced by this ticket (T-013 list structure, stale anchors, missing History, board branch cell) | patched directly, noted below |

---

#### Blocking 1 — an omitted `[commit]` table inverts the commit policy

`internal/install/install.go:553-561` branches on `cfg.Commit.ChildPublishGated` /
`OverarchingAuto` (decision 6, correctly). But `CommitPolicy` is two plain `bool`s
(`internal/config/config.go:51-52`) and `applyDefaults` (`:114-127`) defaults `BranchPrefix`,
`WIPInDevelopment` and `WIPInReview` — **not** the commit booleans. An absent `[commit]` table
decodes to `false/false`, which the renderer reads as a deliberate opt-out. Reproduced with the
shipped binary on a `pickle.toml` with no `[commit]` table:

```
- **Commit policy.** Child-projects are **not publish-gated**: commit and push as the work
  needs, and open the merge request when it is ready — **merging is always the human's**.
  Overarching bookkeeping (tickets, board, docs) is committed only when the
  user asks, …
```

while the skill installed into that same tree by that same command says the opposite
(`skill/SKILL.md:3`: *"pushing a child-project requires explicit user approval"*).

This is **new in T-018** — the wording was previously hardcoded to publish-gated, so an omitted
table was harmless. It is the highest-severity finding in this review: the marker block is the
agent's primary instruction file, and the failure mode is an agent pushing a child-project
without approval, in a project whose author never made that choice. `install` writes both keys
as `true` (`install.go:359-360`) and `Render` always emits the table, which is exactly why the
golden path hides it.

**Fix:** default both to `true` in `applyDefaults()` — matching what `install` writes and what
every other surface states — and add a `markerBlock` test for a config with no `[commit]` table.

#### Blocking 2 — the scanner has no TOML lexical state

`internal/config/config.go:281-296` classifies lines by prefix only. Three legal, currently
loadable inputs break it. All reproduced end-to-end with the shipped binary:

**(a) BOM on the key's own line** → key not recognised → a *second* `payload_version` inserted →
duplicate key → the file no longer parses:

```
doctor BEFORE:  0 error(s), 1 warning(s)   rc 0
upgrade:        + pickle.toml (payload_version -> d78fb5d)   … then
                pickle: parse …/pickle.toml: toml: line 3: Key 'payload_version' has already been defined.
```

`upgrade` writes the corrupt file, *then* fails its own post-run self-check. Afterwards `doctor`,
`board audit` and `project list` all exit 1 — the project is unusable until hand-repaired. (Note
the trigger is narrower than "any BOM": a BOM before a *comment* is harmless, because the line is
skipped anyway.) A quoted key `"payload_version" = "…"` fails identically.

**(b) A multi-line string above the key** → continuation lines are misread. If a continuation
line starts with `[` it is taken for a table header, so the key is inserted **inside the user's
string**:

```
note = """                    note = """
[warning]            →        payload_version = "d78fb5d"   ← injected into user prose
keep this                     [warning]
"""                           keep this
payload_version = "old"       """
                              payload_version = "old"        ← never updated
```

The file still parses, `doctor` reports 0 errors, and `upgrade` exits 0 — silent corruption plus
a version that is never stamped. If a continuation line *matches* `payload_version =`, the user's
text is overwritten instead.

The author half-knew this: `config.go:306-310` refuses a multi-line *target* value, but nothing
tracks multi-line state while scanning. Half a guard.

**Fix (one change, covers 2, 3 and the quoted-key case):** stop trying to enumerate key
spellings. After building the new text and **before writing**, decode it and require exactly the
intended result; refuse loudly otherwise, leaving the file untouched:

```go
var check struct{ PayloadVersion string `toml:"payload_version"` }
if _, derr := toml.Decode(updated, &check); derr != nil || check.PayloadVersion != version {
    return fmt.Errorf("%s: cannot set payload_version safely; edit it by hand", path)
}
```

That converts every unknown-shape input from silent corruption into an actionable error, which is
the behaviour this ticket is about. Handle the BOM properly as well (strip/re-emit `\ufeff`), so
the common case still succeeds rather than merely failing safely.

#### Blocking 3 — success is reported, not verified

`internal/install/install.go:159-162` calls `res.created(… "payload_version -> " + payloadVersion)`
whenever the writer returns `nil`. The writer returns `nil` when it *wrote something*, not when it
achieved the stated effect. Under 2(b) `upgrade` prints `+ pickle.toml (payload_version -> v2)`
and exits 0 **on every future run** while the file stays at v1 — the no-op short-circuit at
`config.go:270-272` makes the lie stable. Independent of 2, nothing verifies the transform;
the validation gate above fixes it.

#### Blocking 4 — a symlinked `pickle.toml` is destroyed

`internal/config/config.go:378` `os.Rename(name, path)` **replaces** a symlink; the `os.WriteFile`
it replaced **followed** it. Verified: the link becomes a regular file, the shared target keeps
the old version, and `upgrade` reports success. Hardlinks break the same way. This repo guards
symlinks deliberately everywhere else (`install.go:128`, `:141`, `:184`) — including for the
skill dir in this very function's caller — so a symlinked config is inside the product's
worldview.

**Fix:** `if resolved, err := filepath.EvalSymlinks(path); err == nil { path = resolved }` before
writing, plus a test.

#### Blocking 5 — the tests do not hold the code up

`internal/config/config_test.go:307,326-328` chmods the fixture to `0o600` and asserts `0o600` —
which is exactly `os.CreateTemp`'s default mode. Deleting the entire mode-preservation mechanism
from `writePreservingMode` leaves **all nine packages green** (verified by mutation). Three more
mutations also pass green: removing the `\r` re-append (`config.go:313-315`), removing the whole
multi-line-string guard (`:306-310`), and deleting the entire `### Board rule` section from
`markerBlock`. Measured coverage of the new code: `valueEnd` 46%, `SetPayloadVersionInPlace` 67%,
`writePreservingMode` 71% — **every new error path is uncovered**.

The ticket's own summary claims the tests are "mutation-verified"; that was true of the two
comment-preservation tests and false of the rest. The fixtures were drawn from the same mental
model as the parser, which is precisely why finding 2 shipped green.

**Fix:** chmod the fixture to `0o640`; replace the example tests with one invariant table over
adversarial fixtures (BOM, CRLF, no trailing newline, quoted key, literal string, multi-line
string elsewhere, aligned columns, no `[commit]`, empty file) asserting *"if the input parses,
the output parses and carries the new version, and every other line is byte-identical — or the
call refuses"*; add a `FuzzSetPayloadVersion` seeded with them (the seed corpus alone catches all
three shapes in finding 2, in 0.00s); add a golden file for `markerBlock`.

#### Blocking 6 — two false statements shipped

- `internal/config/config.go:254-257`: *"preserving every other byte … so no caller can lose
  hand-written content"*. Falsified by finding 2. Also alignment does **not** survive: `:315`
  re-emits the key with canonical spacing, so an aligned block goes ragged, and a `'literal'`
  value is silently converted to `"basic"` form.
- `README.md:99-100`: *"Comments, blank lines, key order and alignment all survive."* Same error,
  user-facing.

A ticket whose entire subject is making shipped statements true must not ship new false ones.
**Fix:** state the real contract — the rewritten line's own spacing is normalised to
`key = "value"` with its inline comment preserved; every other line is untouched.

#### Blocking 6b — docs coverage gap (step 4a.1)

`README.md` documents neither that `[commit]` drives the `AGENTS.md` commit-policy wording, nor
that the per-child keys are rendered into the block. `README.md:217-220` lists three of the five
rendered fact classes (omits the commit policy and `docs`). Given finding 1, those two booleans
are now the only thing between "never push without approval" and "push as the work needs" in the
agent's primary instruction file — undocumented user-facing behaviour, blocking per 4a.1. Folded
into the finding-1 fix.

### Non-blocking findings — routing

- **T-021** *(new)* — `project add|remove` call `cfg.Save("")` and never re-inject the marker
  block, so `AGENTS.md` goes stale. Pre-existing, but T-018 widened it from a stale *name list*
  to stale commands, branch prefixes and WIP limits — an agent will read `≤ 1` on a project
  configured at 5 and run a build command that no longer exists. Deliberately out of this
  ticket's scope per decision 9.
- **T-022** *(new)* — `skill/SKILL.md` and `skill/resources/*` state the commit policy, `feat/`
  prefix and `≤ 1` WIP limits unconditionally, and now sit beside a marker block that renders the
  project's real values. `skill/resources/tickets-README.md:148` already models the right
  hedge. Ships into every installed project.
- **T-023** *(new)* — `internal/move/move.go` derives the board's `branch` cell from the filename
  slug, so this ticket's row names a branch that does not exist; `board audit` does not check it.
- **T-012 item 7** *(extended)* — in-place writer hardening: CRLF on the insert path
  (`config.go:324-325` omits the `\r` the replace path adds), comment orphaning on insert
  (`:319-323` walks back over blank lines but not the comment block they belong to),
  `valueEnd`'s bare-value branch corrupting arrays/inline tables (`:348-351`), no `fsync` before
  rename, errors naming the temp file rather than the config, a read-only `0444` file now being
  rewritable, and the `func`-vs-method API shape leaving `cfg.PayloadVersion` stale in memory.
- **T-013 item 8** *(extended)* — `install.go:162` labels an in-place edit `+` (created);
  `internal/cli/cli.go:87-88` help text omits the `pickle.toml` stamp.

### Patched directly (trivial scaffolding defects introduced by this ticket)

Per the protocol's allowance for cosmetic scaffolding fixes, and because leaving them would
corrupt other tickets' meaning:

- `tickets/1-to-do/T-013-install-polish.md` — the new item 10 had been inserted *between* item
  9's two sub-bullets, orphaning the `runUninstall` stray-positionals bullet under an unrelated
  item; moved item 10 below it and reverted a gratuitous 5→6-space re-indent.
- Same file — item 10's line anchors were off by four (`:418,437,451,498` → `:422,441,455,502`),
  shifted by T-018's own doc-comment addition.
- `tickets/1-to-do/T-012-…` and `T-013-…` — added the missing dated `## History` lines both
  tickets' own convention requires for a broadening.
- `tickets/1-to-do/T-019-…` — re-anchored `README.md:182-183` → `:189-190` and `:292-303` →
  `:307-318`, following that ticket's standing re-anchoring note.
- `tickets/BOARD.md` — T-018's `branch` cell names a branch that does not exist. **Left as-is
  deliberately:** hand-correcting it immediately puts `board sync --dry-run` into `OUT OF SYNC`,
  because `sync` rebuilds the cell from the same slug derivation. The only stable states are
  "wrong cell, sync-clean" and "right cell, permanently sync-dirty"; the former was chosen and
  the defect filed as **T-023**, which now records that a manual fix is not durable.

### Checklist

- [x] Implementation audit — acceptance test re-run verbatim (4/4 steps), tasks & decisions verified (step 2)
- [x] Quality audit — including coverage measurement and 6 mutation experiments (step 3)
- [x] Consistency audit — whole-tree contract sweep, caller/callee drift, marker-block byte-identity (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, all links/anchors resolve; **no docs build configured** for the `pickle` child (`docs` unset in `pickle.toml`), stated rather than skipped (step 4a)
- [x] Findings classified; non-blocking spawned as T-021/T-022/T-023 and folded into T-012/T-013 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep (step 8)
- [x] Summary + commit message presented; no push without approval (step 9)

### Rework scope (the entire scope — nothing else)

Blocking findings 1–6 only, on the same branch:

1. Default `ChildPublishGated`/`OverarchingAuto` to `true` in `applyDefaults()`; test the
   no-`[commit]` marker-block rendering.
2. Add the post-transform TOML validation gate; handle the BOM; refuse unknown shapes loudly.
3. Covered by 2 — plus assert in `Upgrade` that the reported version is the one on disk.
4. Resolve symlinks before the atomic rename; test.
5. Fix the `0o640` fixture; add the seeded invariant table + fuzz target + `markerBlock` golden.
6. Correct the writer's doc comment and `README.md:99-100`; document `[commit]`'s effect on the
   marker block and add the two missing rendered fact classes to `README.md:217-220`.

## Rework — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `0036e28`)

Scope was the six blocking findings and nothing else. Each is recorded against its finding
number, with the verification that now holds it up.

**1 — commit-policy inversion. Fixed.** `applyDefaults` now takes the decoder's
`toml.MetaData` and defaults both commit booleans to `true` when the key is absent
(`config.go:130-146`, `DefaultOverarchingAuto`/`DefaultChildPublishGated` at `:39-49`).
Defaulting on the zero value would have been wrong: an absent key and an explicit `false`
both decode to `false` and must mean opposite things, so only `md.IsDefined` can tell them
apart. Verified both directions with the shipped binary — no `[commit]` table now renders
"Child-projects are **publish-gated**", and an explicit `child_publish_gated = false` still
renders "**not publish-gated**". `TestMarkerBlockDefaultsToTheCautiousCommitPolicy` covers
both; deleting the defaults fails it.

**2 — no TOML lexical state. Fixed, by not trusting the scanner.** `verifyOnlyPayloadVersion`
(`config.go:341-362`) decodes the before and after text and refuses unless the edit set
`payload_version` to exactly the requested value and left every other value identical; the
file is written only after that passes (`setPayloadVersion`, `:316-332`). Decoding into a
`map[string]any` rather than into `Config` was a deliberate strengthening: keys pickle knows
nothing about are still the user's content, and a value-only check has a hole where the target
version happens to match already. The BOM is held aside before scanning and re-emitted after
(`:305-309`, `bom` at `:296`), so that case now *succeeds* instead of merely failing safely.
All four shapes re-verified end-to-end:

| input | before | after |
|---|---|---|
| BOM on the key's own line | duplicate key, file unparseable, `doctor` rc 1 | rc 0, version stamped |
| quoted `"payload_version"` | same corruption | rc 1, config untouched, `doctor` rc 0 |
| multi-line string with a `[`-leading continuation | key injected into user prose, rc 0 | rc 1, config untouched |
| continuation matching `payload_version =` | user text overwritten, rc 0 | rc 1, config untouched |

**3 — unverified success. Fixed.** The gate above makes the effect a precondition of writing at
all, and `Upgrade` additionally re-reads the file and confirms the version before reporting it
(`install.go:163-168`, `verifyStampedVersion` at `:180-191`). Belt and braces on purpose: the
gate protects the transform, this protects the *report*.

**4 — symlinked config replaced. Fixed.** `writePreservingMode` resolves the path with
`filepath.EvalSymlinks` before the rename (`config.go:472-479`). Verified: the link stays a
link and the real target carries the new version.
`TestSetPayloadVersionInPlaceFollowsSymlink` fails if the resolution is removed.

**5 — tests that could not fail. Fixed, and re-measured.** The mode fixture moved to `0o640`
(`0o600` was `os.CreateTemp`'s own default). Added: `TestSetPayloadVersionInvariant` over 22
adversarial fixtures asserting the single rule *"either the call refuses and changes nothing, or
the output parses, carries the new version, decodes identically in every other respect, and is
byte-identical line-for-line apart from the one line rewritten or inserted"*, with each refusal
pinned to its own message; `TestSetPayloadVersionIsIdempotent`; `FuzzSetPayloadVersion` seeded
from the same corpus; `TestMarkerBlockGolden` (regenerate with `UPDATE_GOLDEN=1`);
`TestVerifyStampedVersion`. ~~The mutation battery was re-run over ten mechanisms — nine now
break at least one test when deleted (CRLF re-append, multi-line guard, BOM handling, the
verify gate, the map-vs-struct choice in the gate, `EvalSymlinks`, mode preservation, the
commit-policy defaults, and the `### Board rule` section of `markerBlock`).~~ Coverage of the new
code: `setPayloadVersion` and `replacePayloadVersionLine` 100%, `verifyOnlyPayloadVersion` 92%,
`valueEnd` 46% → 77%, `writePreservingMode` 74%; package 90.4%.

**CORRECTED 2026-07-25 (re-review finding R3, re-run independently in round 2 — the struck
sentence above overclaimed).** The battery covers **eleven** mechanisms, of which **eight** broke
a test at `0036e28`: CRLF re-append on the replace path, the multi-line target-value guard, BOM
handling, the verify gate, `EvalSymlinks`, `usesCRLF` on the insert path (then caught by the fuzz
corpus file **only**), mode preservation, and the `### Board rule` section of `markerBlock`.
**Three stayed green:**

- **the whole-tree comparison** (the "map-vs-struct choice" the struck sentence miscounted as
  caught): `reflect.DeepEqual(b, b)` leaves all packages green, because every shape that would
  move another value fails the parse or version check first. Defence-in-depth, not a verified
  guarantee → doc claim softened in round 2, code question routed to R7/**T-012**;
- **`verifyStampedVersion`'s wiring** into `Upgrade`: it cannot be made to fail without a seam
  for injecting a broken writer → R8/**T-013**. The function itself is unit-tested;
- **the commit-policy *key path***: deleting the defaults outright is caught, but the plausible
  regression — `md.IsDefined("commit")` instead of the two key paths — was not → closed in
  round 2 (R2).

**6 — false statements. Fixed.** The writer's doc comment (`config.go:299-315`) now states the
real contract: one line touched, that line normalised to `payload_version = "value"` with any
inline comment kept, a `'literal'` value becoming a `"basic"` one, and a loud refusal as the
failure mode. `README.md:97-103` says the same, and no longer claims alignment survives.
**6b:** the `[commit]` keys and their effect on the marker block are now documented
(`README.md:112-117`), and the rendered-facts list (`:227-233`) enumerates all five classes
including `docs` and the commit policy.

### One thing outside the stated scope

`FuzzSetPayloadVersion` immediately found that the **insert** path emitted an LF line into a
CRLF file — the CRLF-on-insert defect this review had already routed to **T-012 item 7** as
non-blocking. Rather than commit a red fuzz target or weaken the invariant to accommodate a
known bug, it was fixed here (`usesCRLF`, `config.go:412-427`, two lines at the insert site) and
the seed kept as a regression corpus file. **T-012 item 7's CRLF sub-item should be struck** —
flagged for the re-review rather than done quietly.

### Verification

Acceptance test re-run verbatim, 4/4: `AGENTS.md` byte-identical; `pickle.toml` numstat `1 1`
with the comment count 18 → 18; second upgrade adds no further diff; fresh install + upgrade
rc 0. `just build` / `just test` (10 packages) / `just lint` clean; `doctor` 0 errors (the one
warning is this repo's deliberate `0.0.0-skeleton` payload version, unchanged from before).
`FuzzSetPayloadVersion` clean over 20.7M executions in 90s.

## Re-review — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `928164f`)

**Verdict: REWORK — 3 blocking findings.** Scoped to the six findings of the first review, per
the protocol's re-review rule. Two independent sub-agents attacked the rework (one on writer
safety, one on test strength and doc truth); **every finding below was then reproduced by hand**,
because the implementer reworked its own work and reviewed it too.

**Four of the six findings are properly closed, and the feature works.** Acceptance test re-run
verbatim, 4/4 — `AGENTS.md` byte-identical, `pickle.toml` numstat `1 1`, comment count 18 → 18,
second upgrade no further diff, fresh install + upgrade rc 0; `just build`/`just test` (9
packages)/`just lint` clean; `doctor` 0 errors; `board audit` 25 tickets 0 errors. Findings 1, 2,
3 and 4 are fixed as prescribed: the commit policy defaults correctly in both directions, no
input found (60 hand-crafted + 18,636 parseable fuzz cases) makes `pickle upgrade` silently
corrupt or mis-stamp a `pickle.toml`, the report is verified against the disk, and a symlinked
config survives — including relative links, cross-directory links and 2-hop chains.

What fails is **findings 5 and 6, the two that were about the work being honest about itself**.
Both failed in the same way as the first time, which is why they are blocking again rather than
being waved through.

### Findings

| # | Severity | Finding | Evidence |
|---|---|---|---|
| R1 | **blocking** | Finding 6 not closed: a **new** false statement shipped in the paragraph the fix rewrote, and a third statement of the same class was missed | reproduced ×2 |
| R2 | **blocking** | Finding 5 not closed for finding 1's mechanism: a one-token regression reintroduces the original bug with all 9 packages green | mutation-verified |
| R3 | **blocking** | The Rework record overclaims its own verification — a mechanism it names as mutation-caught is not caught | mutation-verified |
| R4 | non-blocking | Hardlinked `pickle.toml` is severed at rc 0; the other name keeps stale content | → **T-012** item 7 |
| R5 | non-blocking | setuid/setgid/sticky bits, all xattrs and group ownership discarded on every write | → **T-012** item 7 |
| R6 | non-blocking | Legal configs are permanently un-upgradeable, with messages that misdiagnose and a `doctor` that tells you to run the failing command | → **T-026** *(new)* |
| R7 | non-blocking | The whole-tree comparison branch is unreachable and untested (0 coverage) | → **T-012** item 7 |
| R8 | non-blocking | `verifyStampedVersion` has no seam binding it to `Upgrade`; `Config{}` literals still render the unsafe wording | → **T-013** item 8 |
| R9 | non-blocking | `upgrade` exits 1 from a partially-applied upgrade (skill + marker blocks already rewritten) | → **T-013** item 8 |

---

#### Blocking R1 — finding 6 is not closed: one new false statement, one missed

Finding 6 blocked on two false shipped statements, with the reasoning *"a ticket whose entire
subject is making shipped statements true must not ship new false ones."* The rework corrected
both — and introduced a third in the same paragraph.

**(a) `README.md:102-104` (new, written by the rework):** *"a quoted `"payload_version"` key, or
the key sitting after a multi-line string, makes `upgrade` **refuse and change nothing** rather
than guess."* The second clause is false. Reproduced on a real project whose key sits after a
benign multi-line string:

```
note = """
hello
world
"""
payload_version = "0.0.1"
```
```
$ pickle upgrade
  + pickle.toml (payload_version -> 928164f)
pickle upgrade: 0.0.1 -> 928164f          rc 0
diff: payload_version = "0.0.1" -> "928164f"      ← edited, not refused
```

`upgrade` refuses only when the multi-line string contains a `[table]`-looking line or the key
itself, not merely because a multi-line string precedes the key. The error is in the
*conservative* direction — it claims more caution than exists — and the behaviour it
misdescribes is correct. It is still a false statement about shipped behaviour, in the exact
paragraph finding 6 required to be made true, and it now contradicts the doc comment it is
supposed to mirror (`config.go:333-340`, which hedges correctly).

**(b) `internal/config/config.go:227` (pre-existing, missed):** the header that `Render` writes
into **every generated `pickle.toml`** says *"Comments and layout survive `pickle upgrade`"*.
`README.md` and the doc comment were both corrected to concede that the rewritten line's own
layout does **not** survive; this third copy of the claim was not. Reproduced:

```
BEFORE:  payload_version   = 'v-old'   # aligned, literal, with a comment
AFTER:   payload_version = "928164f"   # aligned, literal, with a comment
```

Alignment normalised, `'literal'` → `"basic"` — the precise carve-out the other two statements
now make. This is the same finding-6 defect class in the same commit's blast radius, and it
ships into every user's config file rather than into docs a user may not read.

**Fix:** correct `README.md:102-104` to state the real refusal condition (the scanner refuses
when it cannot read the shape — a quoted key, or a multi-line string containing a table-looking
line or the key), and qualify `config.go:227` the way the other two statements are qualified.

#### Blocking R2 — finding 5 is not closed for the mechanism finding 1 was about

The invariant table, fuzz target and golden are a genuine improvement, and eight of the eleven
mechanisms tested do now break a test. But the tests still do not protect **finding 1**, the
first review's highest-severity finding. `TestMarkerBlockDefaultsToTheCautiousCommitPolicy`
covers exactly two configs: no `[commit]` table at all (`install_test.go:505`) and both keys
explicitly `false` (`:531`). It never covers a **partially populated** table — which is the one
shape the `md.IsDefined` key-path logic exists to get right.

So the single most plausible regression in this code is invisible. Replacing the two key-level
checks with a table-level one — `md.IsDefined("commit")`, a one-token slip:

```
$ go test -count=1 ./...
ok  internal/config   ok  internal/install   … all 9 packages ok
```

and then, on a config with `[commit]` containing only `overarching_auto = true`:

```
- **Commit policy.** Child-projects are **not publish-gated**: commit and push as the work
  needs, and open the merge request when it is ready …
```

That is blocking finding 1, verbatim, reintroduced with a green suite. The rework's own note
that "deleting the defaults fails it" is true and beside the point: nobody deletes defaults,
they get the key path subtly wrong.

Related, same class: mutation 7 (`usesCRLF` on the insert path) is caught by **one artifact
only** — the fuzz corpus file `testdata/fuzz/FuzzSetPayloadVersion/5984cfcc32e533b4`. No entry
in `payloadVersionFixtures` exercises CRLF + insert (the `crlf` fixture already has the key, so
it walks the replace path). One `git clean -x` on `testdata/` and the guard for the bug the
fuzzer just found goes unprotected.

**Fix:** add two fixtures — a `[commit]` table with only one of the two keys present (asserting
the *other* still defaults to gated), and a CRLF file with **no** `payload_version` (insert
path). Both must fail under their respective mutations.

#### Blocking R3 — the Rework record overclaims, the same way the summary did last round

The Rework section states the battery was re-run over ten mechanisms and *"nine now break at
least one test when deleted"*, listing among the nine *"the map-vs-struct choice in the gate"*.
The substantive form of that mechanism is **not** caught. Neutering the whole-tree comparison so
it always passes (`reflect.DeepEqual(b, b)`) leaves all 9 packages green, and the coverage
profile shows the refusal branch never executes:

```
config.go:357.30,360.3  1  0        ← the "would change other values" refusal: 0 executions
```

No input reaches it: every shape that would change another value fails the parse check or the
version check first, both of which fire earlier. The map-vs-struct choice is defensible as
defence-in-depth against future scanner changes, and it is *not* what the record claims it is —
a mutation-verified guarantee. The honest count is **eight of eleven**, with three green
(whole-tree compare, `verifyStampedVersion` wiring, and the `[commit]` key path of R2).

This is the first review's finding 5 in miniature: the summary claimed "mutation-verified" of
tests that could not fail. Correcting a record's overclaim is a two-line edit; leaving it means
the next reader trusts a number that is wrong.

**Fix:** correct the count and the mechanism list in the Rework section, and either soften
`config.go:333-340`'s claim about the whole-tree comparison to defence-in-depth or find the
input that reaches it (routing the code decision to R7).

### Non-blocking findings — routing

- **T-012 item 7** *(extended)* — (R4) `filepath.EvalSymlinks` fixed symlinks and cannot fix
  **hardlinks**: `os.Rename` unlinks the old inode, so a hardlinked `pickle.toml` silently
  diverges at rc 0 (`nlink 2 → 1`, the other name stranded at the old version). The first review
  named this in finding 4's evidence; the prescribed fix could not address it. (R5)
  `writePreservingMode` preserves only `Mode().Perm()`, so setuid/setgid/sticky bits, **all
  extended attributes** (Finder tags, Spotlight comments, quarantine state) and **group
  ownership** are discarded on every successful write — the temp file inherits the directory's
  group. (R7) the whole-tree comparison branch is unreachable and 0-covered. Plus: the scanner
  can still rewrite the *wrong* line when the target version already matches (an escaped
  `\u0039` inside a multi-line string decodes equal, so the value gate passes) — unreachable
  through `pickle upgrade` today only because `Upgrade` short-circuits the no-op case. Also
  **strike item 7's CRLF-on-insert sub-item**: fixed under this ticket, as the Rework section
  disclosed.
- **T-013 item 8** *(extended)* — (R8) `verifyStampedVersion` is 100% covered and 0% bound: no
  test fails if its call site is deleted, and it needs a seam (an injectable writer that accepts
  a write which reads back wrong). Also `config.Config{}` zero-value literals render the
  **unsafe** commit wording, and `writeConfig` (`install.go:375`) hardcodes `true`/`true`
  instead of the `Default*` constants three files away, so the two defaults can silently
  diverge. (R9) `upgrade` rewrites the skill payload and both marker blocks *before* stamping,
  so a stamp refusal exits 1 from a partially-applied upgrade.
- **T-026** *(new)* — the gate's dominant failure mode is not corruption but **wedging**: a
  legal config with `[ ] checklist` lines at column 0 inside a multi-line string, a `nan` value,
  or a quoted key becomes permanently un-upgradeable (32% of parseable fuzz inputs are refused),
  while `doctor` warns *"run `pickle upgrade`"* — the command that just failed. The messages
  misdiagnose: a `nan` file is refused with *"would change other values in the file"* when
  nothing changed and nothing would (`reflect.DeepEqual(NaN, NaN)` is false); the
  table-looking-line refusal names neither the cause nor the line; three paths surface raw
  errnos naming a temp file the user never created. Refusing is the design (finding 2 asked for
  it) — being unable to say why, and leaving no way forward, is not.

### Checklist

- [x] Implementation audit — acceptance test re-run verbatim (4/4), all six findings re-verified by hand (step 2)
- [x] Quality audit — 11-mechanism mutation battery re-run independently, coverage re-measured, 60 adversarial inputs + 1.9M-exec fuzz (step 3)
- [x] Consistency audit — whole-tree sweep for surviving claims about comment/alignment/preservation; found the third copy (R1b) (step 4)
- [x] Documentation audit — README, doc comments, generated config header, `--help`, `skill/`; **no docs build configured** for the `pickle` child (`docs` unset) (step 4a)
- [x] Findings classified; non-blocking routed to T-026 *(new)* and T-012/T-013 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep (step 8)
- [x] No push — the branch stays local; nothing presented for publish approval while blocking findings stand (step 9)

### Rework scope (the entire scope — nothing else)

R1, R2, R3 only, on the same branch. All three are small; none is a redesign:

1. Correct `README.md:102-104` to the real refusal condition; qualify `config.go:227`'s
   "layout survive" the way the other two statements are qualified.
2. Add the two missing fixtures — partially-populated `[commit]`, and CRLF + insert — and
   confirm each fails under its mutation.
3. Correct the Rework section's mutation count and mechanism list; soften the whole-tree
   comparison's doc claim to defence-in-depth.

## Rework (round 2) — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `21b8b57`)

Scope was R1, R2, R3 and nothing else. One code commit; no production behaviour changed except
the two doc strings the findings required.

**R1 — finding 6 still open: one new false statement, one missed copy. Fixed, after establishing
what the code actually does.** Rather than paraphrase the review, the refusal boundary was
re-derived from the shipped binary (all five cases reproduced end-to-end on a real installed
project, with the file compared byte-for-byte afterwards):

| shape | behaviour | verified |
|---|---|---|
| multi-line string above the key, benign contents | **edited**, rc 0 | the claim the old README got wrong |
| quoted `"payload_version"` key | refuse, file unchanged, rc 1 | `… would leave the file unparseable` |
| multi-line string containing a `[table]`-looking line | refuse, file unchanged, rc 1 | `could not set payload_version` |
| multi-line string containing a `payload_version =` line | refuse, file unchanged, rc 1 | `could not set payload_version` |
| the key's own value is a multi-line string | refuse, file unchanged, rc 1 | `uses a multi-line string; set it by hand` |

`README.md:99-108` now states exactly that boundary — the refusal is triggered by *what the
scanner cannot read*, not by the mere presence of a multi-line string — and concedes in the same
sentence that the rewritten line's own alignment and quoting style are not preserved. **(b)** the
third copy of the claim, `Render`'s generated header (`config.go:227-231`), carries the same
qualification now; that header ships into every installed `pickle.toml`, which is why the review
weighted it above the README. A new fixture, *"benign multi-line string above the key"*, pins the
accepted case in the invariant table, so the README sentence has a test behind it and cannot go
stale silently.

**R2 — finding 5 still open for finding 1's mechanism. Fixed, both gaps.**
`TestMarkerBlockDefaultsToTheCautiousCommitPolicy` now also covers a **partially populated**
`[commit]` table, in both directions: only `overarching_auto` present (the absent
`child_publish_gated` must still render **publish-gated**) and only `child_publish_gated = false`
present (the absent `overarching_auto` must still render *may be committed automatically*).
Mutation-verified: replacing the two key paths with the table-level `md.IsDefined("commit")`
fails **both** sub-cases — the one-token slip the review demonstrated is now visible. Second gap:
the `crlf, key absent` fixture exercises CRLF **plus the insert path**, which no fixture did
(`crlf` already had the key, so it walked the replace path). Verified by deleting the `usesCRLF`
block *with the fuzz corpus directory moved aside*: the named fixture alone fails it, so a
`git clean -x` on `testdata/` no longer unprotects the bug the fuzzer found.

**R3 — the Rework record's overclaim. Corrected, from a re-run rather than from the review.** The
eleven-mechanism battery was executed again, scripted, on this branch; the corrected tally is
recorded in the round-1 Rework section above (eight of eleven at `0036e28`, three green). The
whole-tree comparison's green result was reproduced independently before writing it down. As of
this round the tally is **nine of eleven** — the `[commit]` key path is now caught by R2's
fixtures and `usesCRLF`-on-insert by a named test rather than by the corpus alone — leaving two
green, both routed non-blocking (whole-tree compare → **T-012** item 7 / R7,
`verifyStampedVersion` wiring → **T-013** item 8 / R8). `verifyOnlyPayloadVersion`'s doc comment
(`config.go:333-346`) now says the whole-tree comparison is defence-in-depth that no known input
reaches, kept so a future scanner change cannot quietly start moving other values — replacing the
claim that it is what makes the edit safe.

Also re-verified as caught, though outside the eleven: deleting the commit-policy defaults
entirely (a twelfth mutation) fails `TestMarkerBlockDefaultsToTheCautiousCommitPolicy`.

### Verification

Acceptance test re-run verbatim, 4/4: `AGENTS.md` byte-identical after `upgrade`; `pickle.toml`
numstat `1 1` with the changed line being `payload_version` and the comment count 18 → 18; the
second `upgrade` adds no further diff; fresh `install` + `upgrade` rc 0. `just build` /
`just test` (9 packages) / `just lint` clean; `doctor` 0 errors (the one warning is this repo's
deliberate `0.0.0-skeleton` payload version); `board audit` 26 tickets, 0 errors.
`FuzzSetPayloadVersion` clean over 3.3M executions in 64s, adding no corpus entry. Coverage of
the writer after the new fixtures: `setPayloadVersion` / `replacePayloadVersionLine` /
`usesCRLF` / `applyDefaults` 100%, `verifyOnlyPayloadVersion` 92%, `valueEnd` 77%,
`writePreservingMode` 74%; package 90.5%.

Nothing outside R1–R3 was touched: no new non-blocking finding was fixed opportunistically this
round, and the routings to T-012, T-013 and T-026 stand as the re-review left them.

## Re-review (round 2) — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `8b7b200`)

**Verdict: REWORK — 1 blocking finding.** Scoped to R1, R2, R3 per the protocol's re-review rule.
Every claim the round-2 rework makes about itself was re-derived from the shipped binary or
re-run as a mutation, independently of the record — which is the same discipline the last two
rounds established, and the reason this finding exists.

**R2 and R3 are properly closed, and the feature works.** Acceptance test re-run verbatim, 4/4 —
`AGENTS.md` byte-identical after `upgrade`, `pickle.toml` numstat `1 1` with the changed line
being `payload_version`, comment count 18 → 18, second `upgrade` no further diff, fresh
`install` + `upgrade` rc 0. `just build` / `just test` (9 packages) / `just lint` clean; `doctor`
0 errors (the one warning is this repo's deliberate `0.0.0-skeleton`); `board audit` 26 tickets,
0 errors. No production behaviour changed this round: the diff is two doc strings, two fixtures
and one test table (`git diff 928164f..HEAD -- ':!tickets'`), exactly as claimed.

**R1 is not closed.** The paragraph was corrected for the case the last review named, and the
correction over-generalises in the same direction as the statement it replaced. That is the
third generation of the same error in the same six lines.

### Findings

| # | Severity | Finding | Evidence |
|---|---|---|---|
| S1 | **blocking** | R1 not closed: the rewritten refusal contract claims a refusal that does not happen — "a multi-line string **anywhere in the file**" is false for the below-the-key half | reproduced ×2 with the shipped binary |

No non-blocking findings this round; the routings to T-012, T-013, T-021, T-022, T-023 and T-026
stand as the last re-review left them.

---

#### Blocking S1 — the refusal contract is still wrong, now in the other direction

`README.md:102-108` (written by the round-2 rework) enumerates the shapes that are "beyond" the
line scanner and states that they make `upgrade` **"refuse and change nothing"**. One of the three
is:

> or a multi-line string **anywhere in the file** containing a line that looks like a `[table]`
> header or like the key itself.

The refusal is **positional**, not global: `replacePayloadVersionLine`
(`internal/config/config.go:378-417`) returns as soon as it matches the key, so anything after
that line is never scanned. A multi-line string carrying either offending line **below** the key
is read correctly and edited. Reproduced on a real installed project with the shipped binary,
file compared byte-for-byte afterwards:

| shape | README says | actual |
|---|---|---|
| `[warning]` inside a multi-line string **below** the key | refuse, change nothing | **edited**, rc 0, only the version line changed |
| `payload_version = "sneaky"` inside a multi-line string **below** the key | refuse, change nothing | **edited**, rc 0, only the version line changed |

```
payload_version = "0.0.1"        →   payload_version = "8b7b200"
note = """                           note = """
[warning]                            [warning]
keep this                            keep this
"""                                  """
$ pickle upgrade   →  rc 0, "pickle upgrade: 0.0.1 -> 8b7b200"
```

The five shapes the Rework record tabulates were all re-verified and are all correct — the
above-the-key placements refuse with the messages quoted (`could not set payload_version …` ×2,
`… would leave the file unparseable` for a quoted key, `uses a multi-line string; set it by hand`
for a multi-line target value), and the benign case is edited. The full boundary, measured:

| construct | key present, construct **above** it | key present, construct **below** it | key absent |
|---|---|---|---|
| multi-line string with a `[table]`-looking line | refuse (rc 1, unchanged) | **edited (rc 0)** | refuse if before the first real table header, else edited |
| multi-line string with a `payload_version =` line | refuse (rc 1, unchanged) | **edited (rc 0)** | as above |

This is the same defect class, the same direction and the same paragraph as R1(a): a false
statement about shipped behaviour that claims *more* caution than exists. It is harmless to data —
the over-claim is on the safe side and the behaviour it misdescribes is correct — but the standard
this ticket set for itself, and was blocked on twice, is that a ticket whose entire subject is
making shipped statements true must not ship new false ones. Nothing else in the tree repeats it:
`rg -i 'multi-line|refuse|refusal'` over `README.md`, `internal/`, `skill/` finds only this
sentence making a universal claim; the doc comment (`config.go:292-297`) and the generated header
(`config.go:226-230`) are hedged correctly, so R1(b) **is** closed.

**Root cause, which the fix must address:** each round's wording has been generalised from the
test fixtures rather than from the scanner's control flow, and both refusal fixtures
(`config_test.go:497-498`) place the multi-line string **above** the key. So the sentence had no
test behind it in either direction — the same gap round 2 closed for the benign case with the
`benign multi-line string above the key` fixture.

**Fix (one clause + one fixture):**

1. `README.md:102-108` — make the position explicit and drop the universal. E.g. *"…or a
   multi-line string **before** the key (or, when the key is absent, before the first table
   header) containing a line that looks like a `[table]` header or like the key itself"*, and
   replace the trailing sentence with something that no longer implies placement is irrelevant —
   the scanner stops at the key, so anything below it is not read and cannot trigger a refusal.
2. Add the mirror fixture to `payloadVersionFixtures`: the *same* `[warning]` multi-line string
   placed **after** a real `payload_version` line, `ok: true`. It must fail if the scanner ever
   starts refusing it, so the README sentence has a test behind it in both directions.

#### R2 — closed. Both gaps verified by independent mutation

- **Partially populated `[commit]`.** `TestMarkerBlockDefaultsToTheCautiousCommitPolicy` now runs
  two sub-cases (`install_test.go:547-580`). Replacing both key paths with the table-level
  `md.IsDefined("commit")` — the one-token slip the last review demonstrated — now **fails both**
  (`only_overarching_auto` and `only_child_publish_gated`), where it previously left all 9
  packages green. Re-run here, not taken on trust.
- **CRLF + insert.** Deleting the `usesCRLF` block at the insert site with
  `internal/config/testdata/fuzz/` **moved aside** fails `TestSetPayloadVersionInvariant/crlf,_key_absent`.
  The guard for the bug the fuzzer found no longer depends on a corpus file surviving
  `git clean -x`.

#### R3 — closed. The tally is now honest

The eleven-mechanism battery was re-run from scratch on this branch, scripted, without reading the
record's list first. Result at `8b7b200`: **nine caught, two green** — exactly what the corrected
record states, with the same two green ones (whole-tree comparison, `verifyStampedVersion`
wiring), both routed non-blocking to T-012/R7 and T-013/R8.

| mechanism | mutation | result |
|---|---|---|
| CRLF re-append (replace path) | delete | CAUGHT (`crlf`, `IsIdempotent`) |
| multi-line target-value guard | delete | CAUGHT (`multi-line_value_on_the_key_itself`) |
| BOM hold-aside | delete | CAUGHT (`bom_on_the_key's_own_line`) |
| verify gate | delete call | CAUGHT (4 refusal fixtures) |
| `EvalSymlinks` | delete | CAUGHT (`FollowsSymlink`) |
| `usesCRLF` (insert path) | delete | CAUGHT (`crlf,_key_absent`) |
| mode preservation | delete the `os.Stat` block | CAUGHT (`EscapesAndKeepsMode`) |
| `markerBlock` `### Board rule` | blank the section | CAUGHT (`MarkerBlockGolden`) |
| commit-policy key path | table-level `IsDefined` | CAUGHT (both partial sub-cases) |
| whole-tree comparison | `DeepEqual(b, b)` | **GREEN** — as recorded, → T-012 item 7 / R7 |
| `verifyStampedVersion` wiring | delete call site | **GREEN** — as recorded, → T-013 item 8 / R8 |

Note the earlier "delete `os.Chmod`" formulation of the mode mutation does not compile (it
orphans `mode`), so it proves nothing; deleting the `os.Stat` block is the behavioural form, and
that is caught. The doc-comment softening R3 also asked for is in place
(`config.go:338-343`: defence-in-depth, "no input is known to reach it"), and matches the GREEN
result above rather than overstating it.

### Patched directly (trivial bookkeeping)

`tickets/BOARD.md` was one `board sync` reformat behind: the TO DO rows for T-026 and T-025 —
added by the previous re-review — sat above their impact-ordered positions, so
`board sync --dry-run` reported `OUT OF SYNC (reformat only)` *before* this round's move. Ran
`pickle board sync`; the only change is those two rows' ordering, and the board is now in sync
with `audit` still at 26 tickets / 0 errors. The T-018 `branch` cell is still the slug-derived
one, left alone deliberately per T-023.

### Checklist

- [x] Implementation audit — acceptance test re-run verbatim (4/4), R1/R2/R3 re-verified from the binary and by mutation (step 2)
- [x] Quality audit — 11-mechanism battery re-run independently; round-2 diff confirmed to change no production behaviour (step 3)
- [x] Consistency audit — whole-tree sweep for surviving refusal/preservation claims; only `README.md:104` over-claims (step 4)
- [x] Documentation audit — README refusal contract, doc comments, generated header, `[commit]` coverage, rendered-facts list; **no docs build configured** for the `pickle` child (`docs` unset) (step 4a)
- [x] Findings classified; no new non-blocking findings, existing routings unchanged (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6a)
- [x] `BOARD.md` updated (step 7)
- [x] Impact sweep — no ticket lists T-018 in `depends-on:`; T-024/T-025 reference it only as lineage data, unaffected (step 8)
- [x] No push — the branch stays local; nothing presented for publish approval while a blocking finding stands (step 9)

### Rework scope (the entire scope — nothing else)

S1 only, on the same branch:

1. Correct `README.md:102-108` so the refusal condition is stated positionally (before the key,
   or before the first table header when the key is absent) instead of "anywhere in the file".
2. Add the mirror fixture — the same table-looking multi-line string placed **below** a real
   `payload_version` line, expected `ok: true` — so the corrected sentence is pinned in both
   directions.

## Rework (round 3) — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `c587c3d`)

Scope was S1 and nothing else. One commit; no production code touched — the change is one README
paragraph and two test fixtures.

**S1 — the refusal contract claimed a refusal that does not happen. Fixed, from the control flow
rather than from prose.** The previous two attempts each generalised from the fixtures at hand and
each got the boundary wrong in the same over-cautious direction. This time the rule was read off
`replacePayloadVersionLine` (`internal/config/config.go:378-417`) — the loop returns at the first
top-level `payload_version` assignment, so **nothing below that line is ever scanned** — and then
the whole boundary was measured end-to-end with the freshly built binary before a word was
written. `README.md:102-110` now states it positionally and says *why* position is what decides
it:

> …or a multi-line string holding a line that looks like a `[table]` header or like the key
> itself, sitting **above** the key (above the first `[table]` header, in a file that has no
> `payload_version` yet). … Position is what decides it, not the mere presence of a multi-line
> string: the scan stops at the key, so a multi-line string *below* the key is never read and
> cannot trigger a refusal, and one above it is read correctly as long as it holds neither of
> those two lines.

Every clause of that sentence was then re-measured against the shipped binary on a real installed
project, file compared byte-for-byte afterwards. All eight outcomes match the text:

| shape | README | measured |
|---|---|---|
| quoted `"payload_version"` key | refuse | rc 1, unchanged |
| the key's own value is a multi-line string | refuse | rc 1, unchanged |
| `[table]`-looking line in a multi-line string **above** the key | refuse | rc 1, unchanged |
| key-looking line in a multi-line string **above** the key | refuse | rc 1, unchanged |
| `[table]`-looking line in a multi-line string **below** the key | edited | rc 0, edited |
| key-looking line in a multi-line string **below** the key | edited | rc 0, edited |
| benign multi-line string above the key | edited | rc 0, edited |
| key absent, `[table]`-looking line above the first table header | refuse | rc 1, unchanged |

**The sentence now has tests behind it in both directions.** `payloadVersionFixtures`
(`internal/config/config_test.go:493-500`) gains the two mirrors of the existing refusals — the
table-looking and the key-holding multi-line string placed **after** a real `payload_version`
line, both `ok: true`. They flow into `TestSetPayloadVersionInvariant`,
`TestSetPayloadVersionIsIdempotent` and the fuzz seed corpus automatically.

**Mutation-verified against the defect itself, not just against deletion.** The false README
sentence describes a real alternative implementation, so that implementation was written as the
mutation: a whole-file pre-scan that refuses whenever a multi-line string holds a table-looking
line or the key, regardless of position. With it in place the two new fixtures fail by name
(`table-looking_multi-line_string_below_the_key`,
`multi-line_string_containing_the_key,_below_the_key`) in both the invariant and idempotence
tables, while the eight other packages stay green — i.e. before this round, shipping the behaviour
the README described would have gone unnoticed; now it cannot.

Two fixtures rather than the one the rework scope named: the scope asked for the table-looking
mirror, and the sentence makes the same claim about the key-holding line. Adding both costs one
line and closes the "the other half is still untested" gap instead of leaving it for a fourth
round. Nothing else was touched — no new non-blocking finding was picked up, and the routings to
T-012, T-013, T-021, T-022, T-023 and T-026 stand as the re-reviews left them.

### Verification

Acceptance test re-run verbatim, 4/4: `AGENTS.md` byte-identical after `upgrade`; `pickle.toml`
numstat `1 1` with the changed line being `payload_version` and the comment count 18 → 18; the
second `upgrade` adds no further diff; fresh `install` + `upgrade` rc 0. `just build` /
`just test` (9 packages) / `just lint` clean; `doctor` 0 errors (the one warning is this repo's
deliberate `0.0.0-skeleton`); `board audit` 26 tickets, 0 errors. `FuzzSetPayloadVersion` clean
over 4.5M executions in 47s, adding no corpus entry. Coverage unchanged where it matters:
`setPayloadVersion` / `replacePayloadVersionLine` / `usesCRLF` / `applyDefaults` 100%,
`verifyOnlyPayloadVersion` 92%, `valueEnd` 77%, `writePreservingMode` 74%; package 90.5%.

## Re-review (round 3) — 2026-07-25 (`feat/T-018-upgrade-preserve-user-content` @ `0effc62`)

**Verdict: DONE.** Scoped to S1. Every claim the round-3 rework makes was re-derived from the
shipped binary or re-run as a mutation, independently of the record — the discipline the previous
rounds established, applied one last time.

### S1 — closed

**The statement is now true, and it is true for more shapes than it enumerates.** The corrected
paragraph (`README.md:100-110`) was checked clause by clause against a freshly built binary on a
real installed project, with the file compared byte-for-byte after each run. All eight outcomes
the sentence commits to hold, and two shapes it does not mention behave the way the sentence's
*reasoning* predicts:

| shape | README implies | measured |
|---|---|---|
| quoted `"payload_version"` key | refuse | rc 1, unchanged |
| the key's own value is a multi-line string | refuse | rc 1, unchanged |
| `[table]`-looking line in a `"""` string **above** the key | refuse | rc 1, unchanged |
| key-looking line in a `"""` string **above** the key | refuse | rc 1, unchanged |
| either line in a `"""` string **below** the key | edited | rc 0, edited |
| benign multi-line string above the key | edited | rc 0, edited |
| key absent, `[table]`-looking line above the first table header | refuse | rc 1, unchanged |
| key absent, key-looking line in a multi-line string | refuse | rc 1, unchanged |
| `'''` literal string with a `[table]`-looking line above the key | (covered by "multi-line string") | rc 1, unchanged |
| multi-line array with a `[`-leading continuation, key present above it | (not enumerated) | rc 0, edited |

The remaining prose was re-read for the same defect class and holds: "some shapes are beyond it"
is explicitly non-exhaustive, so the shapes outside the enumeration (`nan`, arrays, duplicate
keys) do not falsify it; the doc comment (`internal/config/config.go:292-297`) and the generated
header (`:226-230`) make no positional claim and so cannot contradict it. `rg -i
'multi-line|refuse|refusal'` over `README.md`, `internal/` and `skill/` finds no other statement
of the contract.

**The sentence is now pinned by tests in both directions.** The two mirror fixtures
(`config_test.go:494-500`) go through `checkPayloadVersionInvariant`, which is a real assertion,
not a smoke test: output must parse, carry the new version, decode identically in every other
respect, keep the BOM and CRLF state, and differ by at most one line which must contain the key.
Worth noting the second fixture is the sharper of the two — its multi-line string *contains*
`payload_version`, so rewriting the wrong line would still produce a line "containing the key";
the version and whole-tree checks are what catch it.

**Mutation re-run independently, and it is the right mutation.** Rather than deleting a mechanism,
the rework implemented the behaviour the false sentence described — a whole-file pre-scan that
refuses regardless of position — which is the only mutation that can prove the fixtures pin
*position*. Re-run here from scratch: both new fixtures fail by name in the invariant table and in
`TestSetPayloadVersionIsIdempotent`, and the other eight packages stay green. (The mutation also
trips the two pre-existing refusal fixtures on their message assertions — expected, and not
claimed otherwise by the rework record.)

**Two fixtures instead of the one the scope named** was disclosed rather than done quietly, and is
the right call: the sentence makes the same claim about both offending lines, so pinning only one
would have left the other half arguable in a fourth round.

**Scope was respected.** `git diff 8b7b200..HEAD -- ':!tickets'` is exactly `README.md` (+8/−5) and
`internal/config/config_test.go` (+6): no production code, no opportunistic fixes, and the
routings to T-012, T-013, T-021, T-022, T-023 and T-026 stand.

**The round-3 record is accurate.** Every number in it was reproduced: acceptance 4/4, 9 packages,
fuzz 4.5M execs clean with no new corpus entry, and the coverage figures
(`setPayloadVersion` / `replacePayloadVersionLine` / `usesCRLF` / `applyDefaults` 100%,
`verifyOnlyPayloadVersion` 92.3%, `valueEnd` 76.9%, `writePreservingMode` 73.7%, package 90.5%).
That closes the R3 defect class too — the record no longer claims more than it can show.

### Whole-ticket verification (concluding run)

- Acceptance test verbatim, 4/4: `AGENTS.md` byte-identical after `upgrade`; `pickle.toml` numstat
  `1 1`, the changed line is `payload_version`, comment count 18 → 18; second `upgrade` adds no
  further diff; fresh `install` + `upgrade` rc 0.
- `just build` / `just test` (9 packages) / `just lint` clean; `doctor` 0 errors (the one warning
  is this repo's deliberate `0.0.0-skeleton`); `board audit` 26 tickets 0 errors;
  `board sync --dry-run` in sync.
- All earlier findings closed: 1–4 and 6b at `0036e28`, 5/6 → R1–R3 at `21b8b57`, R1's residue →
  S1 at `c587c3d`. Decisions D1–D10 honoured, as verified in rounds 1 and 2 and unchanged since.

### Findings

| # | Severity | Finding | Routing |
|---|---|---|---|
| S1 | — | **closed** (see above) | — |
| N1 | non-blocking | The wedge is not specific to multi-line *strings*: a multi-line **array** with a `[`-leading continuation line wedges an otherwise legal config on the *insert* path, refused with "would leave the file unparseable" and naming neither the array nor the line | → **T-026** (added as a fourth shape, with the reproduction) |
| N2 | non-blocking | The `upgrade` bullet is now 11 lines with heavy qualifier density; the same facts read better as a short bullet plus a sub-list of the three refusing shapes | → **T-019** |

**N1** is behaviour this ticket deliberately chose (refusing beats corrupting) and it is *correct*
— it is only the diagnosis and the dead end that are wrong, which is exactly T-026's thesis. It is
recorded there because it shows the cause is the `[`-prefix heuristic itself, not multi-line
strings, and any remedy needs a fixture for it.

**N2** was assessed by an independent readability pass over the changed prose only. Verdict:
"factually dense but reasonably readable", with the long final sentence as the weakness — i.e. not
a blocker. Its proposed sub-list dropped the "above the first `[table]` header, in a file that has
no `payload_version` yet" clause, so T-019 must treat every fact as load-bearing: this paragraph
took three rounds to make true and a restructuring that loses a qualifier re-opens it.

### Patched directly (trivial bookkeeping, no ticket meaning changed)

Line anchors into code **this ticket moved** had drifted in three other tickets, so the next
implementer would read the wrong lines. Each replacement was verified by printing the target line:

- `T-012` item 7 — `config.go:428 → :434` (`usesCRLF`), `:357-360 → :363-366` (the whole-tree
  refusal), `:319-323 → :419-423` (the insert walk-back), `:348-351 → :465-468` (`valueEnd`'s
  bare-value branch), `:372-378 → :486-502` (the atomic write), `:467-471 → :473-477`
  (`writePreservingMode`'s doc comment).
- `T-013` item 8/9 — `install.go:126 → :129` (`os.RemoveAll`), `:130 → :133` (`copyPayload`),
  `:162 → :168` (`res.created` for `pickle.toml`), `:180-191 → :173-184`
  (`verifyStampedVersion`), `:163-168 → :165-167` (its call site).
- `T-026` — `config.go:382 → :388` (the table-header break).
- A dated `## History` line was appended to each of the three, per their own convention.

Observation, not a finding: this is the third round in which anchors had to be re-pointed. The
durable fix is a convention (name the symbol, not the line) rather than a defect in any ticket, so
nothing was spawned for it.

### Checklist

- [x] Implementation audit — acceptance test re-run verbatim (4/4); S1 re-measured over 10 shapes with the shipped binary (step 2)
- [x] Quality audit — invariant helper inspected for real assertions; the false-claim mutation re-run independently; coverage and fuzz figures reproduced (step 3)
- [x] Consistency audit — whole-tree sweep for competing statements of the refusal contract; doc comment and generated header cross-checked against the README (step 4)
- [x] Documentation audit — README paragraph verified clause by clause; independent readability pass (→ T-019); **no docs build configured** for the `pickle` child (`docs` unset in `pickle.toml`) (step 4a)
- [x] Findings classified; N1 → T-026, N2 → T-019; bookkeeping patched directly (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6b)
- [x] `BOARD.md` updated; T-012/T-013/T-026 re-anchored (step 7)
- [x] Impact sweep — no ticket lists T-018 in `depends-on:`; **T-020 is now unblocked** (the marker block is regenerable, so drift is a meaningful signal — refine it next); T-012/T-013/T-021/T-022/T-023/T-026 carry the routed items; T-024/T-025 reference T-018 only as lineage data (step 8)
- [x] Summary + commit message and MR attributes presented for approval; **not pushed** (step 9)

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
- 2026-07-25 — TO DO → READY: plan complete; D1-D9 confirmed with user
- 2026-07-25 — READY → IN DEVELOPMENT: picked up; applicability gate PROCEED-WITH-CORRECTIONS, B1/N3/N7/N9 signed off
- 2026-07-25 — IN DEVELOPMENT → IN REVIEW: acceptance green; AGENTS.md byte-identical, pickle.toml 1 line
- 2026-07-25 — IN REVIEW → REWORK: 6 blocking findings (commit-policy default inversion, TOML lexical state, unverified success, symlink replace, un-failable tests, false statements)
- 2026-07-25 — REWORK → IN REVIEW: findings fixed: 6 blocking closed, acceptance 4/4, fuzz clean
- 2026-07-25 — IN REVIEW → REWORK: re-review: 3 blocking (new false statement + missed third copy, test gap reintroducing finding 1, rework record overclaim); non-blocking -> T-026 + T-012/T-013
- 2026-07-25 — REWORK → IN REVIEW: R1-R3 fixed: refusal contract stated truthfully in README + generated header; partial [commit] and CRLF-insert fixtures mutation-verified; rework record corrected
- 2026-07-25 — IN REVIEW → REWORK: re-review: 1 blocking (R1 not closed: refusal contract over-generalised to "anywhere in the file"); R2/R3 closed, mutation battery 9/11 confirmed
- 2026-07-25 — REWORK → IN REVIEW: S1 fixed: refusal boundary stated positionally, verified end-to-end for all 8 shapes; both mirror fixtures fail under the false-claim mutation
- 2026-07-25 — IN REVIEW → DONE: re-review: S1 closed (refusal contract true for all 10 measured shapes, pinned both directions); non-blocking N1 -> T-026, N2 -> T-019
- 2026-07-25 — merged to main (1485242)
