---
id: T-150
title: install takes --ticket-prefix and the marker block renders the configured prefix instead of a hardcoded T-NNN
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-150 — install takes --ticket-prefix and the marker block renders the configured prefix instead of a hardcoded T-NNN

## Outcome

A project can pick its ticket prefix at install time (`pickle install --in-tree
--ticket-prefix ABC`), and the generated `AGENTS.md` marker block shows that prefix in its trigger
phrases and bookkeeping grammar, so agents are not pointed at `T-NNN` in a project whose ids are
`ABC-NNN`.

## Description

On `main` (ec216c1): `pickle project add` already takes `--ticket-prefix` (T-058), but `install`
does not, so the first child's prefix can only be changed by hand-editing `pickle.toml`. After that
edit, `pickle upgrade` re-renders the branch line (`feat/ABC-NNN-<slug>`) and `ticket new` allocates
`ABC-001`. But the trigger phrases and the `board: T-NNN <verb phrase>` grammar in `MarkerBlock()`
(`internal/install/install.go`, the hardcoded lines near 1858-1868) stay `T-NNN`.
`pickle doctor` does flag the stale block and names `upgrade`, and the configuration chapter
says so too.

Scope:
- Add `install --ticket-prefix`, validated exactly like `project add`'s flag and written to the
  first child.
- Render the marker block's `T-NNN` placeholders from the prefix when all registered children share
  one; render the literal `<PREFIX>-NNN` when they differ, and `T-NNN` when no child is
  registered (decision 3). Update the golden file.
- No doctor work: `pickle doctor` already warns on marker-block drift and names `pickle upgrade`
  (checked at refinement), so a hand edit of `ticket_prefix` is already caught.

The skill payload's `T-NNN` examples are generic syntax and stay as they are (the payload is shared
by every install).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                          # the pickle child is the repo root
git checkout main && git pull --ff-only
git checkout -b feat/T-150-install-ticket-prefix
```

Commit locally as you go (WIP commits are encouraged). Follow this repo's self-modify policy:
never run `pickle install|upgrade|uninstall` against this repo. Do test installs in a throwaway
dir with the binary copied in as `pickle-test`. `layout = "in-tree"`: before pushing, run
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`, which
must print nothing. No push and no MR without explicit user approval. Merging is always the
human's.

### Prerequisite gate (hard)

None. `main` is clean and `just test` passes on it. T-148 is a soft coupling only; see Task 4.

### Confirmed design decisions (do not deviate without asking)

1. **`install --ticket-prefix` mirrors `project add --ticket-prefix`.** Validation is the same
   (`cfg.AddProject` → `Validate`, `^[A-Z][A-Z0-9]{0,7}$`), and the value is written to the
   first child's `ticket_prefix`.
2. **It needs a child.** The flag defaults to empty, meaning `config.DefaultTicketPrefix`. Given
   without `--path` or `--in-tree`, it is refused with the same message the other child flags get
   (it joins `childFlagsGiven`).
3. **The marker block renders one prefix when the registered children agree.** With no children,
   it renders `T`, the default every new child gets. When all children share prefix P, it renders
   P. When they differ, it renders the literal `<PREFIX>`. This covers the trigger phrases, the
   `board: T-NNN <verb phrase>` grammar and the commit example `(T-2)`. A config whose children
   all use `T` renders byte-identical to today, so no existing install drifts, this repo's
   included.
4. **No new doctor check.** `checkMarkerDrift` (`internal/doctor/doctor.go`) already warns,
   naming `pickle upgrade`, when the marker block differs from what `pickle.toml` renders.
   Verified on 2026-11-07 by reading the code; the acceptance test proves it. The configuration
   chapter already says to re-run `pickle upgrade` after changing `ticket_prefix`.
5. **The skill payload is untouched.** Its `T-NNN` are generic syntax for the default, and the
   payload's "Project configuration wins" note already defers to the marker block.

### Tasks

#### Task 1 — Flag plumbing
- In `internal/cli/install.go`, add the `--ticket-prefix` flag, add it to `childFlagsGiven`, and
  name it in that refusal message.
- In `internal/install/install.go`, add `TicketPrefix string` to `Options`, and pass it into the
  `config.Project` that `writeConfig` builds. An empty value falls through to `AddProject`'s
  default.
- Validate the prefix in `Run` before `copyPayload` writes anything (a throwaway
  `(&config.Config{}).AddProject(...)` with the same fields), so a refused `--ticket-prefix abc`
  leaves the tree byte-identical, as T-148's refusals do.

#### Task 2 — Prefix-aware marker block
In `internal/install/install.go`, add `markerPrefix(cfg *config.Config) string` implementing
decision 3, using `Project.Prefix()`. In `MarkerBlock`, replace the hardcoded `T-NNN` (the
triggers lines and the board-grammar line) and the `(T-2)` example with that prefix. Regenerate
the golden file with `UPDATE_GOLDEN=1 go test ./internal/install/ -run TestMarkerBlockGolden`.
The golden config mixes `T` and `BETA`, so the diff must be exactly those lines turning into
`<PREFIX>`; review it before committing. Add `TestMarkerBlockPrefix`, table-driven: no children
→ `T-NNN`; one child `ABC` → `ABC-NNN` and no `T-NNN`; mixed → `<PREFIX>-NNN`; all `T` → the
same output as before this change.

#### Task 3 — Help line
In `internal/cli/cli.go` `usage()`, add a `--ticket-prefix <P>` clause under `install`, in the
style T-149 established if it has landed, otherwise as one continuation line.

#### Task 4 — T-148 coupling (T-148 is merged: required)
Add `ticket_prefix` to `checkExistingConfig`'s comparison: when `opts.TicketPrefix != ""` and it
differs from the registered child's `Prefix()`, it is a mismatch naming `--ticket-prefix`. Its own
check, not a fifth entry in the command loop (that loop uses one string for both key and flag).

### Acceptance test

```
just build && just test && just lint && just docs-check
./pickle doctor 2>&1 | grep -i 'marker'      # this repo: "AGENTS.md marker block current"
                                             # (decision 3: default T renders unchanged)

D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q
./pickle-test install --in-tree --ticket-prefix ABC
grep -c 'T-NNN' AGENTS.md                    # 0
grep -c 'ABC-NNN' AGENTS.md                  # ≥ 3 (triggers, board grammar, branch line)
./pickle-test ticket new "x" --project "$(basename "$D")"   # created ABC-001

D2=$(mktemp -d) && cp pickle "$D2/pickle-test" && cd "$D2" && git init -q
./pickle-test install --ticket-prefix ABC; echo "exit=$?"            # exit=2, needs --path/--in-tree
./pickle-test install --in-tree --ticket-prefix abc; echo "exit=$?"  # non-zero, names the legal shape

D3=$(mktemp -d) && cp pickle "$D3/pickle-test" && cd "$D3" && git init -q
./pickle-test install --in-tree
sed -i '' 's/^ticket_prefix = "T"/ticket_prefix = "XYZ"/' pickle.toml
./pickle-test doctor 2>&1 | grep 'AGENTS.md block differs'   # warns, names pickle upgrade
./pickle-test upgrade >/dev/null && ./pickle-test doctor 2>&1 | grep -c 'differs'   # 0
grep -c 'XYZ-NNN' AGENTS.md                  # ≥ 3
```

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`: add `[--ticket-prefix <P>]` to the `== pickle install`
  synopsis, with one sentence pointing at `project add`'s identical flag; add `--ticket-prefix` to
  the list of flags a re-install can contradict.
- `docs/user-manual/configuration.adoc`: in the `ticket_prefix` bullet, add that
  `pickle install --ticket-prefix` sets it for the first child.
- `CHANGELOG.md` `[Unreleased]`:
  - *Added*: `pickle install --ticket-prefix` (T-150).
  - *Changed*: the `AGENTS.md` marker block shows the configured ticket prefix in its trigger
    phrases and bookkeeping grammar (T-150).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs and the `CHANGELOG.md` entry above in place.
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggested commit(s), Conventional Commits with the id at the end of the subject:
   - `feat(install): add --ticket-prefix and render the configured prefix in the marker block (T-150)`
5. Root-path child: before presenting, tidy the WIP commits into the atomic commits above.
6. Commit locally. Do not push. `pickle ticket move T-150 in-review --reason "acceptance green"`.

## Review

Review 1 — 2026-11-08, branch `feat/T-150-install-ticket-prefix` (57dc9b4).

- [x] Reviewer independence settled (step 0): delegated. The orchestrator wrote the branch, so the audits ran in a fresh sub-agent given the verbatim spawn prompt (`hunt: one pass`). No delegated findings discarded; all four were re-verified by hand against the code.
- [x] In-tree stale-branch check (step 0a): branch rebased onto `main`; `pickle doctor` 0 errors, 0 warnings.
- [x] Implementation audit (steps 1, 2): acceptance test re-run green; Tasks 1–4, decisions 1–5 verified.
- [x] Correctness hunt (step 3): one-pass angle hunt (all 8 angles) by the delegated reviewer, no sub-agents, no host tool; angle 7 included in the pass. Extra probes: same-prefix re-install accepted; differing prefix refused exit 2; umbrella `project add --ticket-prefix` renders correctly; legacy unset prefix compares as `T`. No correctness rows.
- [x] Consistency audit (step 4): remaining `T-NNN` in `internal/` are generic usage strings, comments and parsing (decision 5).
- [x] Documentation audit (step 4a): cli-reference synopsis, re-install list, configuration bullet, CHANGELOG Added/Changed; whole-tree sweep clean; `just docs-check` green.
- [x] Docs-readability pass (step 4b): conscious skip — three short additive sentences mirroring adjacent prose.
- [x] Findings recorded (step 5).
- [x] Ticket moved to `6-done/` (step 6).
- [x] Other references / governing documents (step 7): `lifecycle.adoc`'s note that `ticket_prefix` renders into the marker block is now more accurate; no edit needed.
- [x] Impact sweep (step 8): T-151 (READY) already plans for T-150's flags via its drift test; no patch.
- [ ] Summary + commit message & MR attributes presented for approval (step 9).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | test-gap | noted | The "all T renders as before" case checks one trigger substring, not the whole block byte for byte. | `internal/install/install_test.go` `TestMarkerBlockPrefix`; byte-identity is proven in practice by this repo's `pickle doctor -v` "marker block current" | Optionally assert the `board: T-NNN` and `(T-2)` lines too. |
| F2 | non-blocking | design | noted | Help line misaligned by one column: `--ticket-prefix <P>` is 19 chars, the flag column 19 incl. padding; listed after `--hooks`. | `./pickle help` | Widen the column if another long flag lands. |
| F3 | non-blocking | design | noted | A lowercase `--ticket-prefix abc` on a re-install is reported as a mismatch (checkExistingConfig runs first), and the hint suggests purge + re-install, which then names the legal shape. | re-install transcript, exit 2 | Edge case; second attempt is self-explanatory. |
| F4 | non-blocking | design | noted | A fresh install validates the child twice (pre-check in `Run`, then `writeConfig`'s `AddProject`). | `internal/install/install.go` `Run`, `writeConfig` | Harmless; keeps `writeConfig` safe alone. |

Dispositions: 4 noted (F1–F4); 0 fixed inline, 0 folded, 0 new tickets.
cost: estimated S, actual S

## History

- 2026-11-07 — created (TO DO). source: chat: user changed ticket_prefix after install and found AGENTS.md still says T; confirmed upgrade fixes only the branch line
- 2026-11-07 — TO DO → READY: plan complete; decisions confirmed with the user 2026-11-07
- 2026-11-08 — READY → IN DEVELOPMENT: picked up
- 2026-11-08 — plan amended inline: applicability gate (fresh sub-agent, no blocking findings) — refreshed stale line refs; Task 1 validates the prefix before any write; Task 4 now required (T-148 merged) with its mechanics; Docs adds --ticket-prefix to the re-install refusal list
- 2026-11-08 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-11-08 — IN REVIEW → DONE: review 1: no blocking findings; 4 noted (F1-F4)
