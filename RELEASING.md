# Releasing pickle

`pickle` is distributed as cross-compiled static binaries via
[goreleaser](https://goreleaser.com), a GitHub release per tag, and a Homebrew formula published to
a separate tap repo (`github.com/codcod/homebrew-taps` → `brew install codcod/taps/pickle`).

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
- an updated **Homebrew formula** committed to the tap.

The version is stamped into the binary via `-ldflags -X main.version={{.Version}}`; a
`go install github.com/codcod/pickle@vX.Y.Z` build (no ldflags) falls back to the module version
from `runtime/debug.ReadBuildInfo()`.

## Validating locally (no publish)

```sh
just dist-check      # goreleaser check — is .goreleaser.yaml valid?
just dist-snapshot   # cross-compile into ./dist, generate the formula, upload nothing
```

