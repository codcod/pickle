# pickle — task runner. Run `just` to list recipes.

# Build version stamped from git (falls back to "dev").
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

default:
    @just --list

# Compile the pickle binary into ./pickle
[group('build')]
build:
    go build -ldflags "-X main.version={{version}}" -o pickle .

# Print the version the binary reports
[group('build')]
show-version: build
    ./pickle version

# Remove build artifacts
[group('build')]
clean:
    rm -f pickle
    rm -rf dist

# Run the test suite
[group('test & lint')]
test:
    go test ./...

# vet + gofmt check, plus the CI surface (actionlint/shellcheck) when installed
[group('test & lint')]
lint: lint-ci-surface
    go vet ./...
    @test -z "$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

# Static-check the CI surface: workflow YAML + shell scripts. Both tools are
# optional locally (a bare checkout needs only Go) and mandatory in ci.yml's
# ci-surface job, which is where a finding actually blocks a merge (T-088).
[group('test & lint')]
lint-ci-surface:
    @if command -v actionlint >/dev/null 2>&1; then actionlint; else echo "warning: actionlint not installed — skipping workflow lint (CI still runs it)"; fi
    @if command -v shellcheck >/dev/null 2>&1; then shellcheck .github/scripts/*.sh; else echo "warning: shellcheck not installed — skipping shell lint (CI still runs it)"; fi

# Validate the AsciiDoc manual: snowball check (render-and-discard, catches broken
# includes) plus the Go tests that catch what rendering silently lets through — a
# dead <<anchor>>, an inter-document xref:<file>.adoc form, or an orphaned page
# under docs/user-manual/ (T-067).
[group('docs')]
docs-check:
    snowball check
    go test . -run '^TestDocs'

# Render the user manual to PDF + EPUB into dist/docs/ (never committed)
[group('docs')]
docs-build:
    snowball build -o dist/docs

# Validate the goreleaser config (also run in CI).
# GitLab tokens are unset so goreleaser detects the GitHub forge deterministically
# (it prefers GitLab whenever GITLAB_TOKEN is present in the environment).
# Exit code 2 ("valid but uses deprecated properties") is tolerated: the `brews`
# pipe is used deliberately (see .goreleaser.yaml). Real errors exit 1 and fail.
[group('release')]
dist-check:
    env -u GITLAB_TOKEN -u GITLAB_PERSONAL_ACCESS_TOKEN goreleaser check || [ $? -eq 2 ]

# Local, unpublished cross-compiled build into ./dist (no tokens, no upload).
[group('release')]
dist-snapshot:
    env -u GITLAB_TOKEN -u GITLAB_PERSONAL_ACCESS_TOKEN goreleaser release --snapshot --clean
