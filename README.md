# pickle

`pickle` installs and operates a **ticket-based, board-driven feature flow** in
any project.

Run `pickle install` in a project and a coding agent (Claude Code, opencode, Pi)
understands requests like *"create a feature to review the Jira board"*: it
authors a correctly-formatted ticket in `tickets/1-to-do/T-NNN-<slug>.md` and
regenerates `BOARD.md`, because `install` laid down the flow skill, the board,
and the agent-instruction markers. Judgment (what a ticket says, how it is
graded, refined, reviewed) stays with the agent; the deterministic mechanics
(id allocation, atomic moves, board rendering, invariant audits) are CLI
commands.

## Install

> **Two different “installs.”** *This* section installs the **`pickle` binary** on your machine.
> The `pickle install` *command* is a separate step you run **inside a project** to lay down the
> flow (skill, board, markers).

**Homebrew** (via the tap):

```sh
brew install codcod/taps/pickle
```

**`go install`** (Go 1.26+, any platform):

```sh
go install github.com/codcod/pickle@latest
```

**From source** (Go 1.26+, optionally [`just`](https://github.com/casey/just)):

```sh
git clone https://github.com/codcod/pickle && cd pickle
just build            # → ./pickle   (or: go build -o pickle .)
```

Then, inside any project:

```sh
pickle install        # lay down the ticket flow
pickle version
```

> `brew install` / `go install` work once the GitHub repo + tap exist and a release tag is cut
> — see [`RELEASING.md`](RELEASING.md).

## Documentation

The user manual (AsciiDoc) lives in [`docs/`](docs/README.adoc) — the book is
[`docs/user-manual.adoc`](docs/user-manual.adoc), in three parts:

- **Start Here** — quickstart, installation, your first project.
- **Concepts** — how the flow works: tickets, lifecycle, the board, multi-project.
- **CLI Reference** — every command, plus the `pickle.toml` reference.

Validate the sources with `just docs-check` (after `bundle install`).
