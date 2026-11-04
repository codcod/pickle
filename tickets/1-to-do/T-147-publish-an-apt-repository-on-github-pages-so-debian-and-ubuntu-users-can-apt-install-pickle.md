---
id: T-147
title: publish an apt repository on GitHub Pages so Debian and Ubuntu users can apt install pickle
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-147 — publish an apt repository on GitHub Pages so Debian and Ubuntu users can apt install pickle

## Outcome

A Debian or Ubuntu user adds one signed apt source pointing at a GitHub Pages URL owned by the
project, then runs `sudo apt install pickle` and later gets new versions through `apt upgrade`.
Every tagged release publishes its `.deb` files to that repository with no manual step.

## Description

Today `pickle` installs through Homebrew, `go install`, and the GitHub release archives (T-011,
T-143). There is no `.deb` and no apt repository.

**What exists already (2026-11-04, outside the repo).** As an experiment, v1.4.3's linux
binaries were packaged by hand with `nfpm` and pushed to a Cloudsmith Open-Source repository,
`codcod/pickle-cli`. That repository works: `apt install pickle` was verified on clean Ubuntu
24.04, Ubuntu 26.04 and Debian 12, for amd64 and arm64. Its slug is `pickle-cli` only because
the original `codcod/pickle` was deleted and Cloudsmith keeps the slug reserved. Nothing in
this repo references it, and only v1.4.3 is there.

**What this ticket does.** It makes the apt repository project-owned and release-driven:

1. **Build `.deb` files in goreleaser.** Add an `nfpms` section to `.goreleaser.yaml` for the
   existing `pickle` build: package name `pickle`, binary at `/usr/bin/pickle`, the
   maintainer, the homepage, `license: MIT`, and `amd64` + `arm64`. The `.deb` files are also
   attached to the GitHub release, so `sudo apt install ./pickle_X.Y.Z_amd64.deb` works even
   without the repository.
2. **Generate and sign an apt repository index** in the `release` workflow after goreleaser
   runs: `Packages`, `Release` and `InRelease`, plus the public key, built from every published
   `.deb`. The signing key is a dedicated GPG key stored as a repository secret.
3. **Publish it to GitHub Pages,** for example `https://codcod.github.io/pickle/apt/`, so the
   user setup is a keyring download plus one `deb [signed-by=…] … stable main` line. The line
   is the same on every Debian or Ubuntu release, because the binary is static. That also
   avoids the release-detection failure Cloudsmith's setup script showed on Ubuntu 26.04.
4. **Document it** in the user manual's installation page and the README install section.

**Open decisions for refinement:**

- **GitHub Pages or Cloudsmith.** This ticket is filed as GitHub Pages. The alternative is to
  keep Cloudsmith and only automate `cloudsmith push` in the release workflow, which needs no
  signing key and no index generation. Refinement should confirm the choice. If Cloudsmith
  wins, retitle the ticket, which then shrinks to tasks 1, 4 and a push step.
- **Index tool:** `apt-ftparchive`/`dpkg-scanpackages` on the runner (no extra dependency), or
  `aptly`/`reprepro`.
- **Where the repository's files live:** a `gh-pages` branch in this repository (git history
  grows with every `.deb`), or a separate `codcod/apt` repository, or Pages fed from a release
  artifact so binaries never enter git.
- **How many versions to keep** in the index: all of them, or the latest N.
- **The interim Cloudsmith repository** (`codcod/pickle-cli`): retire it once this ships, or
  keep it as a mirror.
- **The signing key:** who holds the private key, how it rotates, and where the public
  fingerprint is published.

**Soft couplings.** The release workflow's user-manual step (T-086, T-087) runs in the same
pipeline. The new steps must not make a missing manual block the apt publish, or the reverse.
`RELEASING.md` gains a section for the signing-key secret.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-04 — created (TO DO). source: chat: after a hand-built v1.4.3 `.deb` was pushed to the Cloudsmith repository `codcod/pickle-cli` and installed on Ubuntu and Debian, the user asked to make the GitHub Pages variant of an apt repository a ticket, so future releases reach apt without a manual push
