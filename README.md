# pickle

`pickle` installs and operates **brine**. Brine is a ticket-based workflow for
software development, designed to be used with any coding agent.

Brine is the name of this particular SDLC. Pickle is a command-line tool that
installs Brine.

Run `pickle install` in a project and a coding agent understands requests like
*"create a feature to enable dark theme"*: it authors a correctly-formatted
ticket in `tickets/1-to-do/T-NNN-<slug>.md` and regenerates `BOARD.md`. All
this is possible because `install` laid down the workflow skill, the board, and
the agent-instruction markers. Judgment (what a ticket says, how it is graded,
refined, reviewed) stays with the agent; the deterministic mechanics (id
allocation, atomic moves, board rendering, invariant audits) are CLI commands.

Use `pickle serve` to visualize the board in a browser.

![Board with tickets](https://raw.githubusercontent.com/codcod/pickle/main/docs/board.png)

For the best UX use `pickle` with [Pi](https://pi.dev).

## Install

> **Two different "installs."** *This* section installs the **`pickle` binary**
> on your machine. The `pickle install` *command* is a separate step you run
> **inside a project** to lay down the workflow (skill, board, markers).

Run:

```sh
brew tap codcod/tap
brew trust --formula codcod/tap/pickle
brew install codcod/tap/pickle
```

On Windows, download `pickle_<version>_windows_amd64.zip` (or `_arm64`) from the
[latest release](https://github.com/codcod/pickle/releases/latest), unzip it, and put
`pickle.exe` on your `PATH`. `pickle install` links `.claude/skills/brine` to the skill
directory. Windows only allows that with Developer Mode on (or as an administrator), so
without it pickle writes a copy of the skill there instead, and `pickle upgrade` keeps that
copy current. If the project you clone already commits those paths as symlinks, turn on
Developer Mode and run `git config --global core.symlinks true` *before* cloning; otherwise
git checks them out as small placeholder files. `pickle doctor` reports those, and
`pickle upgrade` replaces the `.claude/skills/brine` placeholder with a copy that git will
then show as changed. Don't commit that change.

Then, inside any project:

```sh
pickle install        # lay down brine
pickle version
```

## Documentation

Each release offers a current version of the user manual in PDF and EPUB
formats.

Both formats are provided by [snowball](https://github.com/codcod/snowball).
