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

# Remove build artifacts
clean:
    rm -f pickle
