# Releasing pickle

`pickle` is distributed as cross-compiled static binaries via
[goreleaser](https://goreleaser.com), a GitHub release per tag, and a Homebrew cask published to
a separate tap repo (`github.com/codcod/homebrew-taps` → `brew install codcod/taps/pickle`).

## One-time setup (human handoff)

These steps touch GitHub accounts/secrets and are **not** done by the build tooling. Do them
once before the first release:

1. **Create the main repo** `github.com/codcod/pickle` and push `main`:
   ```sh
   git remote add origin https://github.com/codcod/pickle.git   # already set locally
   git push -u origin main
   ```
2. **Create the tap repo** `github.com/codcod/homebrew-taps` (empty is fine; goreleaser writes
   `Casks/pickle.rb` into it).
3. **Create a Personal Access Token** with `repo` scope that can push to the tap repo, and add
   it to the **`pickle`** repo's Actions secrets as **`HOMEBREW_TAP_GITHUB_TOKEN`**
   (Settings → Secrets and variables → Actions). The built-in `GITHUB_TOKEN` cannot push to a
   *different* repo, so this separate token is required for the tap.

## Cutting a release

Everything is tag-driven — the [`release`](.github/workflows/release.yml) workflow runs
goreleaser on any `v*` tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The workflow can also be re-run manually for an **existing** tag via *Actions → release →
Run workflow* (`workflow_dispatch` with the tag name), e.g. after fixing a secret.

That produces, for `darwin`/`linux` × `amd64`/`arm64`:

- a **GitHub release** with `.tar.gz` archives + `checksums.txt`;
- an updated **Homebrew cask** committed to the tap.

The version is stamped into the binary via `-ldflags -X main.version={{.Version}}`; a
`go install github.com/codcod/pickle@vX.Y.Z` build (no ldflags) falls back to the module version
from `runtime/debug.ReadBuildInfo()`.

## Validating locally (no publish)

```sh
just dist-check      # goreleaser check — is .goreleaser.yaml valid?
just dist-snapshot   # cross-compile into ./dist, generate the cask, upload nothing
```

> **Forge detection / `GITLAB_TOKEN`.** goreleaser picks its forge from environment tokens and
> will prefer **GitLab** whenever a `GITLAB_TOKEN` (or `GITLAB_PERSONAL_ACCESS_TOKEN`) is
> present — which would point release/cask URLs at `gitlab.com`. The config pins the release
> target to GitHub (`release.github`), and the `just dist-*` recipes unset those GitLab tokens
> so local output is deterministic. CI runners don't have them, so CI is unaffected.

## Config

- [`.goreleaser.yaml`](.goreleaser.yaml) — builds, archives, checksums, and the `homebrew_casks`
  entry targeting the tap.
- [`.github/workflows/release.yml`](.github/workflows/release.yml) — the tag-triggered release
  job (needs `GITHUB_TOKEN` + `HOMEBREW_TAP_GITHUB_TOKEN`).
- [`.github/workflows/ci.yml`](.github/workflows/ci.yml) — runs `goreleaser check` on every push
  / PR so a broken release config fails fast.
