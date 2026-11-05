---
id: T-147
title: publish release .deb packages to the Cloudsmith apt repository so Debian and Ubuntu users can apt install pickle
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-147 — publish release .deb packages to the Cloudsmith apt repository so Debian and Ubuntu users can apt install pickle

## Outcome

Every tagged release attaches `pickle_X.Y.Z_amd64.deb` and `pickle_X.Y.Z_arm64.deb` to its GitHub
release and pushes both to the Cloudsmith apt repository `codcod/pickle-cli`. A Debian or Ubuntu
user who has added that repository once gets each new version through `apt upgrade`. The GitHub
release archives, `checksums.txt`, the Homebrew formula and the user manual come out exactly as
before, and a failed push never stops them.

## Description

Today `pickle` installs through Homebrew, `go install`, and the GitHub release archives (T-011,
T-143). goreleaser builds no `.deb`, and the release workflow publishes to no apt repository.

**What exists already (2026-11-04, outside the repo).** As an experiment, v1.4.3's linux binaries
were packaged by hand with `nfpm` and pushed to the Cloudsmith repository `codcod/pickle-cli`
(type Open-Source, distribution `any-distro/any-version`). `apt install pickle` from it was
verified on clean Ubuntu 24.04, Ubuntu 26.04 and Debian 12, for amd64 and arm64. Its slug is
`pickle-cli` because the original `codcod/pickle` was deleted and Cloudsmith still reserves that
slug, which a recreate refuses with HTTP 409. Only v1.4.3 is in it, and nothing in this repo
references it.

**What this ticket does.**

1. **goreleaser builds the `.deb` files.** An `nfpms` section for the existing `pickle` build
   produces amd64 and arm64 packages named `pickle`, installing `/usr/bin/pickle`. They become
   extra assets on the GitHub release.
2. **The release workflow pushes them to Cloudsmith** after goreleaser has published, with the
   `cloudsmith` CLI and a `CLOUDSMITH_API_KEY` repository secret.
3. **The repository keeps the latest 5 versions.** A Cloudsmith retention rule, set once,
   deletes older packages.
4. **The docs say how to install with apt**, including the hand-written source line. That line
   works even when Cloudsmith's setup script doesn't recognise a new release, which happened
   once on Ubuntu 26.04 while the repository was new.

**Constraint (user-stated, 2026-11-04): the `.deb`/apt work must not break the existing
release.** Whatever this adds is additive and isolated from what a tag already produces: the
GitHub release with its archives and `checksums.txt`, the Homebrew formula pushed to the tap,
and the user-manual attachment. Concretely:

- The apt steps run **after** goreleaser has published, and a failure in them (index build,
  signing, Pages deploy, missing secret) must not fail or roll back the GitHub release or the
  Homebrew update. They should be `continue-on-error` or their own job, and annotated when they
  miss, the same way the user-manual step is.
- Adding `nfpms` must not rename, remove or change the existing archives, `checksums.txt`
  entries, or the Homebrew formula's download URLs. The `.deb` files are extra release assets.
  `checksums.txt` gains lines and loses none.
- A missing signing-key secret, for example on a fork or a re-run, skips the apt publish with a
  warning. It never blocks the tag's release.
- The acceptance test includes `just dist-snapshot` with the old artifact set still present
  and unchanged, and a dry run of the apt steps that cannot touch the real Pages site.

With Cloudsmith chosen (decision 1 below), "signing" and "Pages deploy" in that constraint mean
the Cloudsmith push, and "signing-key secret" means `CLOUDSMITH_API_KEY`. Cloudsmith signs the
index itself.

**Soft couplings.** The release workflow's user-manual steps (T-086, T-087) share the job. The
new step follows their soft-fail and annotate pattern and touches neither. If Cloudsmith support
later frees the `pickle` slug, moving to it changes one workflow variable and the docs URL; that
is not part of this ticket.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git pull --ff-only
git checkout -b feat/T-147-cloudsmith-deb
```

Do all work on this branch, committing locally as you go. Publish only per the project's commit
policy: no push and no merge request without explicit user approval. Tidy WIP commits into atomic
ones before presenting them (root-path child). Before pushing, run
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`, which must
print nothing. Merging is always the human's.

Never run `pickle install|upgrade|uninstall` against this repo from the branch (self-modify
policy). This ticket touches no payload or marker block, so no test install is needed.

### Prerequisite gate (hard)

- The Cloudsmith repository `codcod/pickle-cli` exists, is Open-Source, and accepts pushes with
  the API key. This was confirmed 2026-11-04. Re-check with
  `curl -s -o /dev/null -w '%{http_code}' https://dl.cloudsmith.io/public/codcod/pickle-cli/gpg.key`,
  which must print `200`.
- A `CLOUDSMITH_API_KEY` **repository secret** on `codcod/pickle` is a human action and must
  exist before the first tag that should publish. The workflow tolerates its absence (decision
  4), so it does not block development or review. Remind the user at Finish.

### Confirmed design decisions (do not deviate without asking)

1. **Host on Cloudsmith, not GitHub Pages.** The user chose this on 2026-11-04, during
   refinement. Cloudsmith signs and serves the index, so the project holds no GPG key and
   generates no apt metadata.
2. **The target is `codcod/pickle-cli`, distribution `any-distro/any-version`.** The binary is
   static, so one package serves every Debian and Ubuntu release. The repository path is one
   `env:` value in the workflow (`CLOUDSMITH_REPO: codcod/pickle-cli`), so a future slug change
   touches one line.
3. **The package is named `pickle`, not `pickle-cli`.** The package name matches the command.
   Only the repository slug carries `-cli`.
4. **The push never fails the release.** It is a separate step after goreleaser and after the
   manual attachment, with `continue-on-error: true`. A missing `CLOUDSMITH_API_KEY` exits 0 with
   a `::warning::` and a `$GITHUB_STEP_SUMMARY` line, as does a failed push, the same pattern as
   the user-manual step. Nothing in goreleaser's own run depends on Cloudsmith. A prerelease
   tag (one containing `-`) is skipped with the same warning and summary line, because the one
   `any-distro/any-version` channel would otherwise put it in front of `apt upgrade` users.
5. **Pushes are idempotent.** The step uses `cloudsmith push deb … --republish`, so pushing a
   version that is already there replaces it rather than erroring. A `workflow_dispatch` re-run
   for an already-published tag never reaches the step (goreleaser fails first with
   `422 already_exists`), so an old tag is re-pushed by hand: `gh release download` the `.deb`
   files and `cloudsmith push deb … --republish` them (amended after review 1, B1).
6. **The `.deb` files are additive.** `nfpms` adds release assets and `checksums.txt` lines. The
   `archives` block, its `name_template`, and the `brews` block are not edited. The `brews` block
   has no `ids:`, and goreleaser's brew pipe only consumes archives, so the formula's URLs are
   unaffected. The acceptance test proves this rather than assuming it.
7. **Keep the latest 5 versions** with a Cloudsmith retention rule, set once through the API and
   not from the workflow: `retention_enabled: true`, `retention_count_limit: 10` (5 versions × 2
   architectures, with `retention_group_by_name: true`), and `retention_days_limit: 0`. The
   current default is 28, and with retention enabled it would delete every package older than 28
   days. Before saving, confirm in Cloudsmith's API docs that `0` means "no day limit", and read
   the rule back afterwards.
8. **The cloudsmith CLI comes from `pipx`**, preinstalled on `ubuntu-latest`, pinned to an exact
   version, rather than from a third-party GitHub Action.

### Tasks

#### Task 1 — `nfpms` in `.goreleaser.yaml`

Add after `archives:`:

```yaml
nfpms:
  - id: pickle-deb
    package_name: pickle
    ids: [pickle]
    formats: [deb]
    vendor: codcod
    homepage: https://github.com/codcod/pickle
    maintainer: codcod <nikos@linux.com>
    description: Ticket-based, board-driven feature flow as a CLI
    license: MIT
    bindir: /usr/bin
    file_name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Arch }}"
```

goreleaser only builds `.deb` files for linux targets, so darwin and windows are unaffected.
Check that `ids:` is the right key name for goreleaser v2's `nfpms`, and that the file name comes
out as `pickle_<ver>_amd64.deb` and `pickle_<ver>_arm64.deb`.

#### Task 2 — Cloudsmith push step in `.github/workflows/release.yml`

Append a step after "Attach user manual to release":

- `name: Publish .deb to Cloudsmith`, `continue-on-error: true`.
- `env:` `CLOUDSMITH_API_KEY: ${{ secrets.CLOUDSMITH_API_KEY }}`,
  `CLOUDSMITH_REPO: codcod/pickle-cli`, `TAG: ${{ inputs.tag || github.ref_name }}`.
- Script: if `CLOUDSMITH_API_KEY` is empty, write a warning and summary line ("No .deb published
  to Cloudsmith for $TAG: CLOUDSMITH_API_KEY is not set") and `exit 0`. If `$TAG`
  contains `-` (a prerelease), warn and summarise likewise and `exit 0` (decision 4). Otherwise run
  `pipx install cloudsmith-cli==<pinned>`. Collect `dist/*.deb` with `nullglob`; if none, warn
  and exit 0. For each file run
  `cloudsmith push deb "$CLOUDSMITH_REPO/any-distro/any-version" "$f" --republish`. On any push
  failure, warn, write the summary line naming the file, and exit 0.
- A comment above the step points to `RELEASING.md` for the secret and the retention rule,
  matching the existing steps' comment style.

#### Task 3 — One-time retention rule (decision 7)

Through the API with the user's key, never printing it:
`PATCH https://api.cloudsmith.io/v1/repos/codcod/pickle-cli/retention/` with the decision 7
values. Read it back with a `GET` and paste the result, without credentials, into the
ticket's History. If neither Cloudsmith's docs nor the UI's label for `0` confirms that
`retention_days_limit: 0` means "no day limit", ask Cloudsmith support and do not enable
retention until it is confirmed. This changes the user's account, so confirm with the user immediately before
sending it. It is reversible: setting `retention_enabled: false` undoes it.

#### Task 4 — `RELEASING.md`

Add a section **Debian/Ubuntu packages (Cloudsmith)** covering what the step does, the
`CLOUDSMITH_API_KEY` secret (a Cloudsmith user API key; set it with
`gh secret set CLOUDSMITH_API_KEY`), the soft-fail contract and its warning, the retention rule
and how to inspect it, why the slug is `pickle-cli`, and how to re-push an old tag
(`workflow_dispatch` is safe because of `--republish`). Mention the `.deb` files in the list of
what a release produces.

### Acceptance test

1. **Old artifacts unchanged.** Needs goreleaser v2.8 or later on the PATH (`brew install goreleaser`). On `main` before the change, run `just dist-snapshot` and save
   `ls dist/*.tar.gz dist/*.zip` plus the archive file names in `dist/checksums.txt`. Then on the
   branch, run `just dist-snapshot` again. Every earlier archive name is still present, unchanged,
   and listed in `checksums.txt`, and the only new entries are the two `.deb` files. Use `diff`
   on the two sorted lists: only `+` lines, both ending `.deb`.
2. **Homebrew formula unchanged.** `diff` the snapshot's generated formula (`dist/homebrew/…`)
   between the two runs. The only differences are version and checksum strings, with no new or
   changed URL.
3. **The `.deb` installs.** Install the snapshot's `pickle_*_amd64.deb` with `dpkg -i` in a clean
   `ubuntu:24.04` container, then the arm64 one in a `--platform linux/arm64` `debian:12`
   container. `pickle --version` prints the snapshot version in both.
4. **The push step's soft-fail paths.** Run the Task 2 script locally with `bash`:
   (a) `CLOUDSMITH_API_KEY` unset: exits 0 and prints the warning; (b) no `dist/*.deb`: exits 0
   and warns; (c) push failure, simulated with a bogus key: exits 0 and warns; (d) a prerelease
   `TAG=v9.9.9-rc1` with a key set: exits 0 and warns without attempting a push. Never run it with
   the real key against the real repository during the test.
5. **Workflow syntax.** `actionlint .github/workflows/release.yml` passes, if available, or run
   it in its Docker image.
6. **Live check, after merge, at the next real release.** This is a release-time check, not a
   gate for review. Record it in History: the release run is green, the "Publish .deb to
   Cloudsmith" step pushes both files, and `apt install pickle` in a clean `ubuntu:24.04`
   container after the setup script shows the new version.
7. `just build`, `just test`, `just lint` and `just docs-check` are green.

### Docs update (mandatory when user-facing)

- `docs/user-manual/installation.adoc`: a new `== Debian and Ubuntu (apt)` section after
  `== Homebrew`. It gives the one-line setup script and `sudo apt install pickle`, then the
  hand-written alternative: download the key to a file, `gpg --dearmor --output`, and a
  `deb [signed-by=…] https://dl.cloudsmith.io/public/codcod/pickle-cli/deb/$ID $VERSION_CODENAME main`
  line built from `/etc/os-release`. Note that `any-distro` is not a valid path. Use the long
  `--output` option, since a mangled `-o` produced a confusing gpg usage error in practice. The
  section also mentions the plain `.deb` from the GitHub release for offline installs.
- `README.md` `## Install`: one apt line next to the Homebrew one, linking to the manual
  section.
- `RELEASING.md`: Task 4.
- `CHANGELOG.md` `[Unreleased]`, under `### Added`: "`.deb` packages for Debian and Ubuntu,
  installable with `apt` from the Cloudsmith repository `codcod/pickle-cli` (T-147)."

### Finish (mandatory)

Summarise what changed and the acceptance results. Suggested commits, after tidying:

- `build(release): build .deb packages with goreleaser nfpms (T-147)`
- `ci(release): push .deb packages to Cloudsmith, soft-failing (T-147)`
- `docs: document apt install from Cloudsmith (T-147)`

Remind the user to run `gh secret set CLOUDSMITH_API_KEY -R codcod/pickle` before the next tag.
Then `pickle ticket move T-147 in-review --reason "acceptance green"` and continue into review.

## Review

Review 1 (2026-11-04), branch `feat/T-147-cloudsmith-deb` at `c095d71`.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| B1 | blocking | correctness | — | The documented re-push path (`workflow_dispatch` for an already-published tag) never reaches the Cloudsmith step: goreleaser fails with `422 already_exists` first, and the step has no `if:`, so it is skipped. RELEASING.md's *Re-pushing an old tag* bullet and the workflow comment both promise it works. That recovery is the one needed when the first post-merge tag runs before `CLOUDSMITH_API_KEY` is set. T-147 decision 5 | unreproduced: needs a real dispatch against a published tag. Traced: release.yml goreleaser step (`release --clean`, no `continue-on-error`) → 422 (seen in T-086's History, run 31222772467) → next step skipped under default `success()`. RELEASING.md *Cutting a release* already documents the 422 | Docs-only: replace the bullet with a by-hand re-push (`gh release download vX.Y.Z -p '*.deb'` + `cloudsmith push deb … --republish`) and correct the workflow comment. Needs user sign-off (decision 5) |
| B2 | blocking | correctness | — | One failed push ends the loop (`skip` exits), so a later `.deb` is never pushed, and the summary says "No .deb published" even when an earlier one was. amd64 and arm64 apt users then see different versions | Reproduced (delegated reviewer, step script with stub `cloudsmith` failing on `*arm64*`): amd64 pushed, warning "No .deb published to Cloudsmith for v1.0.0: pushing pickle_1.0.0_arm64.deb failed", exit 0. With the failure on amd64, the arm64 push is never attempted (loop at release.yml `for f in "${debs[@]}"`) | Try every file, collect failures, warn once listing them, with a heading that is accurate for partial publishes |
| F1 | non-blocking | design | noted | `CLOUDSMITH_API_KEY` is in the environment while `pipx install` runs, and transitive PyPI deps are unpinned | unreproduced: supply-chain path; release.yml step `env:` + `pipx install cloudsmith-cli==1.30.0` | Hide the key from the install (`env -u CLOUDSMITH_API_KEY pipx install …`) if ever touched |
| F2 | non-blocking | docs-gap | fixed inline | RELEASING.md recommends a full-scope user API key | RELEASING.md *Secret* bullet | Recommend a service account scoped to `codcod/pickle-cli` |
| F3 | non-blocking | docs-gap | fixed inline | The offline `dpkg -i` path skips the checksum check that `checksums.txt` makes possible | installation.adoc, offline paragraph | Add a `sha256sum --ignore-missing -c checksums.txt` line |
| F4 | non-blocking | design | noted | The retention-inspection `curl` puts the key on the command line | RELEASING.md *Retention* bullet | — |
| F5 | non-blocking | stale-xref | fixed inline | RELEASING.md states the retention rule as in place, but Task 3 was deferred | RELEASING.md *Retention* bullet; History 2026-11-04 | Mark it as not yet applied and give the PATCH |
| F6 | non-blocking | stale-xref | fixed inline | RELEASING.md's opening paragraph still lists only the GitHub release and the Homebrew formula | RELEASING.md lines 3-5 | Add the Cloudsmith apt repository |
| F7 | non-blocking | spec-unclear | fixed inline | "Every release is published" to apt, but prereleases are skipped by design | installation.adoc, first line of the apt section | "Every final release" |
| F8 | non-blocking | docs-gap | fixed inline | The manual says a mistyped `-o` gives a gpg usage error; `-o` is a valid short form | `gpg --dearmor -o /tmp/x.gpg` → rc=0 in ubuntu:24.04 | Drop the sentence |
| F9 | non-blocking | docs-gap | fixed inline | Re-running the hand-written setup fails: gpg prompts to overwrite the existing keyring and has no tty | ubuntu:24.04, second run → `gpg: cannot open '/dev/tty'`, rc=2 | Add `--yes` |
| F10 | non-blocking | spec-unclear | noted | Acceptance step 3 omits `--platform linux/amd64`, so on an arm64 host the amd64 `.deb` refuses to install | delegated reviewer, AT3 | — |

Disposition summary: 2 blocking (B1, B2) → rework; 7 fixed inline (F2, F3, F5, F6, F7, F8, F9), applied in the rework pass; 3 noted (F1, F4, F10); 0 folded; 0 new tickets.

Checklist:

- [x] Reviewer independence (step 0): delegated. The orchestrator wrote the branch, so steps 1–4a went to a fresh sub-agent with the verbatim spawn prompt (`hunt: angle 7 only`). It first ran angle 7 alone and was re-sent, quoting the prompt, for steps 1, 2, 4 and 4a.
- [x] Stale-branch check (step 0a): `pickle doctor` warned that the branch copy was behind `main`; rebased, then 0 warnings.
- [x] Implementation audit (step 2): acceptance 1–5 and 7 pass (re-run by the delegated reviewer); 6 is release-time. Tasks 1, 2, 4 met; Task 3 deferred by the user. Decisions 1–4, 6, 8 met; 5 partial (B1); 7 deferred.
- [x] Correctness hunt (step 3): host tool path (`/code-review medium` on `main...feat/T-147-cloudsmith-deb`), 6 findings: 4 kept (B2, F5, F8, F9), 2 discarded. Angle 7 delegated: F1–F4. B2 merges the tool's partial-push row with the reviewer's mislabelled-summary row (one loop, one fix).
  - Discarded: README anchor `#debian-and-ubuntu-apt` "will not resolve". GitHub renders AsciiDoc ids dash-style (`user-content-homebrew`), so it resolves.
  - Discarded: hand-written line "404s on Linux Mint". `deb/linuxmint/dists/wilma/Release` → HTTP 200.
- [x] Consistency audit (step 4) and docs audit (step 4a): delegated reviewer; F5, F6, F7. `just docs-check` clean. DESIGN.md's distribution line is reconciled at step 7, after the fix.
- [ ] Docs-readability reviewer (step 4b): consciously skipped. It is optional, and the rework pass rewrites the affected prose anyway.

### Rework fix record — round 1 (commits 02000e9..29981c1)

- **B1:** fixed, docs-only (user's choice; decision 5 amended). RELEASING.md *Re-pushing an old tag* now gives a by-hand `gh release download … -p '*.deb'` + `cloudsmith push deb … --republish`, and says why a `workflow_dispatch` re-run cannot reach the step. The workflow comment no longer claims it can. (`29981c1`, `2638c04`)
- **B2:** fixed. The loop tries every `.deb` and collects failures, then warns once: *Cloudsmith publish incomplete for vX.Y.Z: failed to push …*. The *No .deb published* heading is now only used when nothing was attempted. Re-tested under `bash -e`:
  - stub failing on amd64: both pushes attempted, one warning naming amd64, rc=0;
  - all succeed: no summary, rc=0;
  - real CLI with a bogus key: both pushes 401, one warning naming both, rc=0;
  - (a), (b) and (d): unchanged.

  (`2638c04`)
- **Fixed inline**, all in `29981c1`:
  - F2: a service-account key is recommended.
  - F3: the offline path checks `sha256sum --ignore-missing -c checksums.txt` (both OK in ubuntu:24.04).
  - F5: retention is marked *not applied yet*, with the PATCH body.
  - F6: the RELEASING.md intro names the Cloudsmith repository.
  - F7: "Every final release (not prereleases)".
  - F8: the `-o` sentence is dropped.
  - F9: `--yes` added; two consecutive runs both rc=0 in ubuntu:24.04.
- Gates: actionlint ok; `just build`, `test`, `lint` and `docs-check` all green.

Review 2 (2026-11-04), a scoped re-review of `02000e9..29981c1`.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| B3 | blocking | correctness | — | The by-hand re-push that B1's fix added (RELEASING.md *Re-pushing an old tag*) fails: `cloudsmith push deb` takes exactly one file. T-147 decision 5 (amended) | Reproduced, cloudsmith-cli 1.30.0: `cloudsmith push deb codcod/pickle-cli/any-distro/any-version dist/*.deb --republish` → `Error: Got unexpected extra argument (dist/pickle_1.4.4-snapshot_arm64.deb)` | Loop over the files, one push each |
| F11 | non-blocking | spec-unclear | fixed inline | When every push fails, the summary still says "any other .deb was pushed" | delegated reviewer, stub failing every push | Reword to "the rest, if any, were pushed" |
| F12 | non-blocking | docs-gap | fixed inline | The re-push bullet's `pipx install cloudsmith-cli` is unpinned; the workflow pins `==1.30.0` (decision 8) | RELEASING.md *Re-pushing an old tag* | Pin the same version |

Disposition summary: 1 blocking (B3) → rework round 2; 2 fixed inline (F11, F12), applied in the rework pass; 0 noted, folded or new tickets.

Checklist (review 2):

- [x] Independence: delegated to a fresh sub-agent, verbatim spawn prompt, `hunt: one pass` (scoped re-review).
- [x] Step 1: ticket read from `main`; fix record range resolves.
- [x] Step 2: B2 met (re-tested under `bash -e` with stubs: amd64 fails, arm64 fails, both fail, all succeed, plus (a), (b) and (d), all rc=0). B1 partially met (B3). F2, F3 and F5–F9 present. Gates and actionlint green.
- [x] Step 3: angle hunt, one pass over the fix diff; B3 and F11. Angle 7: nothing new. The orchestrator reproduced B3 with the real CLI before recording it.
- [x] Steps 4 and 4a: no stale `workflow_dispatch` or `republish` claim left; warning headings agree between release.yml and RELEASING.md; F12. `just docs-check` clean.

### Rework fix record — round 2 (commit 873bcf3)

- **B3:** fixed. The RELEASING.md re-push loops `for f in /tmp/pickle-debs/*.deb`, one `cloudsmith push deb … "$f" --republish` per file, and says why. Run as written with cloudsmith-cli 1.30.0 and a bogus key, each file reaches the API (401 twice), with no `unexpected extra argument`.
- **F11:** fixed inline. The summary reads "the rest, if any, were pushed". Checked with every push failing: rc=0.
- **F12:** fixed inline. The re-push bullet pins `cloudsmith-cli==1.30.0`, matching the workflow.
- Gates: actionlint ok; `just build`, `test`, `lint` and `docs-check` all green.

Review 3 (2026-11-04), a scoped re-review of round 2 (`873bcf3`).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F13 | non-blocking | spec-unclear | fixed inline | The RELEASING.md re-push block was not repeatable: `gh release download` refuses to overwrite, and a fixed `/tmp/pickle-debs` would pick up another tag's packages | Reproduced with gh: second download → "checksums.txt already exists (use `--clobber` …)" | Use `D=$(mktemp -d)` per run. Done; the block run twice with a bogus key reaches the API with both files each time (2 × 401). A sentence was added that tags before this change have no `.deb` to download (v1.4.3: 0 `.deb` assets) |

Disposition summary: 0 blocking; 1 fixed inline (F13); 0 noted, folded or new tickets.

cost: estimated S, actual M. The change itself was S; two rework rounds on the push step's failure and re-push paths doubled it.

Checklist (review 3):

- [x] Independence: delegated, fresh sub-agent, verbatim spawn prompt, `hunt: one pass`. Doctor clean.
- [x] Step 2: B3, F11 and F12 met; gates and actionlint green.
- [x] Step 3: angle hunt, one pass over `873bcf3`; F13. Angle 7: nothing new. The orchestrator reproduced F13 before recording it.
- [x] Steps 4 and 4a: headings, pins and re-push claims agree across release.yml and RELEASING.md; `just docs-check` clean.
- [x] Step 7: DESIGN.md's distribution line now names the Cloudsmith apt repository (fixed inline, prose this branch made incomplete). No other ticket references T-147. Board regenerated by the move.
- [x] Step 8: impact sweep. No ticket in `1-to-do/` or `2-ready/` references T-147.
- [x] Branch tidied into 3 atomic commits (`ab1620f`, `b918eb2`, `7fd1e19`), with code identical to the WIP tip `68967bc`. The earlier fix records' SHAs no longer resolve; protocol §1's fallback applies.

## History

- 2026-11-04 — created (TO DO). source: chat: after a hand-built v1.4.3 `.deb` was pushed to the Cloudsmith repository `codcod/pickle-cli` and installed on Ubuntu and Debian, the user asked to make the GitHub Pages variant of an apt repository a ticket, so future releases reach apt without a manual push
- 2026-11-04 — refined: user chose Cloudsmith over GitHub Pages, keeping the latest 5 versions, Cloudsmith only (no Pages, no mirror); retitled and filename slug tidied to match, re-graded complexity medium → low and cost M → S, since there is no signing key or index generation
- 2026-11-04 — TO DO → READY: plan complete
- 2026-11-04 — READY → IN DEVELOPMENT: picked up
- 2026-11-04 — plan amended inline: applicability gate (fresh sub-agent, no blocking findings). (1) Task 3 aborts the retention change unless `retention_days_limit: 0` = "no limit" is confirmed (the OpenAPI spec is silent; range 0–180); (2) acceptance step 1 names goreleaser ≥ v2.8 as a local prerequisite; (3) user chose to skip prerelease (`-`) tags in the Cloudsmith step, added to decision 4, Task 2 and acceptance 4(d)
- 2026-11-04 — Task 3 (retention rule) deferred by the user: not applied; Cloudsmith docs confirm `retention_days_limit: 0` disables the age criterion, so the plan's values stand when it is applied. Pending human action, alongside `gh secret set CLOUDSMITH_API_KEY -R codcod/pickle` before the next tag
- 2026-11-04 — IN DEVELOPMENT → IN REVIEW: acceptance green (1-5, 7); 6 is release-time; Task 3 deferred by user
- 2026-11-04 — IN REVIEW → REWORK: review 1: 2 blocking (B1, B2), 7 fixed inline in rework, 3 noted
- 2026-11-04 — plan amended inline: decision 5 no longer claims a `workflow_dispatch` re-run re-pushes an old tag (it 422s at goreleaser first); the re-push is by hand. User chose the docs-only fix for review 1's B1 over adding `if: !cancelled()` to the step
- 2026-11-04 — REWORK → IN REVIEW: findings fixed (round 1)
- 2026-11-04 — IN REVIEW → REWORK: review 2: 1 blocking (B3), 2 fixed inline in rework
- 2026-11-04 — REWORK → IN REVIEW: findings fixed (round 2)
- 2026-11-04 — IN REVIEW → DONE: review 3 clean: 0 blocking; reviews 1-3: 3 blocking fixed over 2 rework rounds (B1-B3), 10 fixed inline, 3 noted, 0 new tickets. Pending: user approval to publish; Task 3 retention and the CLOUDSMITH_API_KEY secret are human actions
- 2026-11-05 — merged to main (PR #107, 91f9782, https://github.com/codcod/pickle/commit/91f9782)
