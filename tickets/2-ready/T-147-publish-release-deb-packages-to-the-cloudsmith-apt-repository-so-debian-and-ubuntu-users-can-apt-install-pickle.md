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
   the user-manual step. Nothing in goreleaser's own run depends on Cloudsmith.
5. **Pushes are idempotent.** The step uses `cloudsmith push deb … --republish`, so a
   `workflow_dispatch` re-run for an existing tag replaces rather than errors.
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
  to Cloudsmith for $TAG: CLOUDSMITH_API_KEY is not set") and `exit 0`. Otherwise run
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
ticket's History. This changes the user's account, so confirm with the user immediately before
sending it. It is reversible: setting `retention_enabled: false` undoes it.

#### Task 4 — `RELEASING.md`

Add a section **Debian/Ubuntu packages (Cloudsmith)** covering what the step does, the
`CLOUDSMITH_API_KEY` secret (a Cloudsmith user API key; set it with
`gh secret set CLOUDSMITH_API_KEY`), the soft-fail contract and its warning, the retention rule
and how to inspect it, why the slug is `pickle-cli`, and how to re-push an old tag
(`workflow_dispatch` is safe because of `--republish`). Mention the `.deb` files in the list of
what a release produces.

### Acceptance test

1. **Old artifacts unchanged.** On `main` before the change, run `just dist-snapshot` and save
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
   and warns; (c) push failure, simulated with a bogus key: exits 0 and warns. Never run it with
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

<!-- empty until IN REVIEW -->

## History

- 2026-11-04 — created (TO DO). source: chat: after a hand-built v1.4.3 `.deb` was pushed to the Cloudsmith repository `codcod/pickle-cli` and installed on Ubuntu and Debian, the user asked to make the GitHub Pages variant of an apt repository a ticket, so future releases reach apt without a manual push
- 2026-11-04 — refined: user chose Cloudsmith over GitHub Pages, keeping the latest 5 versions, Cloudsmith only (no Pages, no mirror); retitled and filename slug tidied to match, re-graded complexity medium → low and cost M → S, since there is no signing key or index generation
- 2026-11-04 — TO DO → READY: plan complete
