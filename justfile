# pickle — task runner. Run `just` to list recipes.

# Build version stamped from git (falls back to "dev").
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

default:
    @just --list

# Compile the pickle binary into ./pickle
build:
    go build -ldflags "-X main.version={{version}}" -o pickle .

# Run the test suite
test:
    go test ./...

# vet + gofmt check
lint:
    go vet ./...
    @test -z "$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

# Print the version the binary reports
show-version: build
    ./pickle version

# Validate the goreleaser config (also run in CI).
# GitLab tokens are unset so goreleaser detects the GitHub forge deterministically
# (it prefers GitLab whenever GITLAB_TOKEN is present in the environment).
dist-check:
    env -u GITLAB_TOKEN -u GITLAB_PERSONAL_ACCESS_TOKEN goreleaser check

# Local, unpublished cross-compiled build into ./dist (no tokens, no upload).
dist-snapshot:
    env -u GITLAB_TOKEN -u GITLAB_PERSONAL_ACCESS_TOKEN goreleaser release --snapshot --clean

# Remove build artifacts
clean:
    rm -f pickle
    rm -rf dist
