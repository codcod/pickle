# Releasing pickle

`pickle` is distributed as cross-compiled static binaries via
[goreleaser](https://goreleaser.com), a GitHub release per tag, and a Homebrew formula published to
a separate tap repo (`github.com/codcod/homebrew-tap` → `brew install codcod/tap/pickle`).

## Cutting a release

First, run `pickle changelog check` (T-093): a mechanical reconciliation of every ticket
that shipped since the last tag against `[Unreleased]`, so you start retitling from a
report rather than a from-scratch reading of the log. It is read-only and always exits
`0` — read the candidates it prints (each points at its ticket file) and either add an
entry or confirm the ticket already records a deliberate decision to have none.

Auditing a release that has already been tagged (rather than the in-flight
`[Unreleased]` section) is also read-only: point `--until` at the tag and
`--section` at its version, e.g. `pickle changelog check --until v0.5.0
--section 0.5.0` — the bare command's default range moves with `HEAD`, so
standing on a tag without `--section` audits the *previous* release against
`[Unreleased]` instead (the report says so with a `note:` line if you forget).

Then update [`CHANGELOG.md`](CHANGELOG.md): retitle the `[Unreleased]` section to
`[X.Y.Z] - YYYY-MM-DD`, add a fresh empty `[Unreleased]` above it, update the link
references at the bottom, and commit — the tag should include the changelog.

Then everything is tag-driven — the [`release`](.github/workflows/release.yml) workflow runs
goreleaser on any `v*` tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The workflow can also be re-run manually for an **existing** tag via *Actions → release →
Run workflow* (`workflow_dispatch` with the tag name), for example after fixing a secret. Do not
use it to re-test the user-manual build against an already fully published tag. goreleaser will
hit `422 already_exists` when it re-uploads the binaries/checksums the real first run already
published — see *Testing the user-manual build* below for the non-destructive way to do that.

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

## Testing the user-manual build (T-087)

The release workflow's **"Build user manual (PDF + EPUB)"** step is `continue-on-error: true` —
a broken manual must never block publishing the binaries — which means it can fail silently. The
two releases after v0.2.1 (`v0.2.2` and `v0.3.0`) each hit a *different* toolchain failure there
and shipped with no manual attached, and nobody noticed for ten days. Two things make that easier
to catch:

- **The step is annotated when it misses.** A release whose manual failed to build now shows a
  `::warning::` and a run-summary line ("No user manual attached to vX.Y.Z") instead of a silent
  `exit 0`.
- **[`manual-smoke`](.github/workflows/manual-smoke.yml) exercises the same script
  (`.github/scripts/build-manual.sh`) without cutting a release.** It runs automatically on any
  push touching `docs/`, `snowball.yaml`, the script, or the workflow itself; or dispatch it by
  hand:

  ```sh
  gh workflow run manual-smoke.yml            # render this ref's own docs
  gh workflow run manual-smoke.yml -f ref=v0.3.0   # render a different ref's docs (see below)
  ```

  A green run uploads `pickle-user-manual.{pdf,epub}` as a workflow artifact. The log also
  includes a diagnostics block (`brew --prefix ruby`, `type -a ruby gem bundle`,
  `snowball doctor`) that no prior release run recorded, plus the temporary bundler-shim note
  below.

### Backfilling an older release's manual

Don't dispatch `release.yml` against an old tag for this — that tag's tree predates the script,
and re-running the full release pipeline re-uploads binaries/checksums that already published,
which goreleaser rejects with `422 already_exists`. Instead, dispatch `manual-smoke` with `ref`
set to the tag, then attach its artifact by hand:

```sh
gh workflow run manual-smoke.yml -f ref=v0.2.2
# once the run finishes:
gh run download <run-id> -n pickle-user-manual -D /tmp/manual
version=0.2.2
mv /tmp/manual/pickle-user-manual.pdf  /tmp/manual/pickle-user-manual-$version.pdf
mv /tmp/manual/pickle-user-manual.epub /tmp/manual/pickle-user-manual-$version.epub
gh release upload v$version /tmp/manual/pickle-user-manual-$version.{pdf,epub}
```

### The temporary bundler shim

`build-manual.sh` currently works around a `bundle`-not-found failure that belongs to Homebrew
ruby, not pickle. On Linuxbrew, ruby's `post_install` removes bundler from the *linked* prefix
(`$(brew --prefix)/bin/bundle` and the prefix gem dir), so on a fresh runner `bundle` is off
`PATH` even though `$(brew --prefix ruby)/bin` still ships it — which is why the shim's first
move is simply to put that keg bin dir on `PATH`, and why its `gem install bundler` fallback has
never actually fired in CI.

The root fix belongs in snowball itself and is tracked as **SNOW-002** on the `unity` workspace's
board. Once a snowball release's `setup` makes `bundle` reliably resolvable on the caller's
`PATH` (not merely inside its own subprocess), delete the shim block in `build-manual.sh` — it is
clearly marked and cites this same ticket.

