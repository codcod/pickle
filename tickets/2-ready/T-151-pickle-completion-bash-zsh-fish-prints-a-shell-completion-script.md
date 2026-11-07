---
id: T-151
title: pickle completion bash|zsh|fish prints a shell completion script
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-151 — pickle completion bash|zsh|fish prints a shell completion script

## Outcome

`pickle completion zsh` (and `bash`, `fish`) prints a completion script, so a user can
tab-complete pickle's subcommands and flags after one line in their shell rc.

## Description

pickle has no shell completion. The CLI is hand-dispatched (stdlib `flag`, no cobra), so
completion means emitting a static script per shell, listing subcommands, sub-subcommands and
their flags. Status names for `ticket move` and child names for `--project` are good targets for
dynamic completion via `pickle project list`. Ticket-id completion is out of scope until someone
asks for it.

Keep the script's command list from drifting: a test asserts that every subcommand in `pickle help`
appears in each script. If the release packaging (Homebrew formula, .deb) installs completion files,
wire them in there; otherwise document the rc-file line in the user manual.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                          # the pickle child is the repo root
git checkout main && git pull --ff-only
git checkout -b feat/T-151-shell-completion
```

Commit locally as you go (WIP commits are encouraged). Follow this repo's self-modify policy:
never run `pickle install|upgrade|uninstall` against this repo. Do test installs in a throwaway
dir with the binary copied in as `pickle-test`. `layout = "in-tree"`: before pushing, run
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'`, which
must print nothing. No push and no MR without explicit user approval. Merging is always the
human's.

### Prerequisite gate (hard)

None. `main` is clean and `just test` passes on it. Whichever of T-148 and T-150 has merged at
pickup, its flags belong in the spec; the drift test in Task 2 enforces that.

### Confirmed design decisions (do not deviate without asking)

1. **`pickle completion <bash|zsh|fish>` prints the script to stdout and nothing else.** Any
   other argument, or none, prints usage and exits 2. There is no self-installing mode.
2. **One Go table drives all three scripts.** `internal/cli/completion.go` holds the spec: each
   top-level command, its subcommands, and each one's flags. Three small renderers turn it into
   the scripts. Stdlib only, with no completion library: the CLI uses `flag`, and `go.mod` has
   no cobra.
3. **Completion is static, plus `ticket move`'s status tokens.** The status tokens are the bare
   forms `flow.ByToken` accepts (`to-do`, `ready`, `in-development`, …), rendered from
   `flow.Default().States()` at generation time. Child names for `--project`, and ticket ids,
   are out of scope.
4. **A test fails when the spec drifts from the CLI.** (a) Every command word in `usage()`'s
   command lines is in the spec, and vice versa. (b) For every spec'd command whose `--help`
   prints `flag`-package usage, the flags printed equal the spec's. Commands that parse flags by
   hand (some `board` subcommands) are listed in the test as exempt from (b).
5. **Packaging: Homebrew via `generate_completions_from_executable`; the .deb via a goreleaser
   `before` hook.** User decision, 2026-11-07.
   - The `brews` entry gets an `install` block: `bin.install "pickle"` plus
     `generate_completions_from_executable(bin/"pickle", "completion")`. Homebrew then runs
     `pickle completion bash|zsh|fish` itself.
   - A `before` hook writes `completions/pickle.bash`, `completions/_pickle` and
     `completions/pickle.fish` (`go run . completion <sh> > …`).
   - The nfpm `contents` install those to `/usr/share/bash-completion/completions/pickle`,
     `/usr/share/zsh/vendor-completions/_pickle` and
     `/usr/share/fish/vendor_completions.d/pickle.fish`.
   - `completions/` is gitignored. Tarballs do not ship the scripts; their users use the
     rc-file line.
6. **The bash script runs on bash 3.2** (macOS's `/bin/bash`): no `mapfile` and no associative
   arrays. The completion function is named `_pickle` in bash and zsh.

### Tasks

#### Task 1 — Command
Create `internal/cli/completion.go` with the spec table, the three renderers and
`runCompletion(args []string) int`. Add `case "completion":` to the dispatch in
`internal/cli/cli.go` `Run`. Add a `usage()` entry under "Other": `completion bash|zsh|fish`,
which prints a shell completion script.

#### Task 2 — Tests
In `internal/cli/completion_test.go`:
- The decision-4 drift test, (a) and (b).
- An exit-2 case for `completion pwsh` and for no argument.
- A syntax check per shell: pipe each script to `bash -n`, `zsh -n` or `fish -n`, calling
  `t.Skip` when that shell is not on `PATH`.
- A bash behaviour check: `eval` the script in `bash -c`, set `COMP_WORDS=(pickle tic)` and
  `COMP_CWORD=1`, call `_pickle`, and assert `COMPREPLY` contains `ticket`.

#### Task 3 — Packaging
In `.goreleaser.yaml`:
- Add a `before.hooks` step that creates `completions/` and writes the three scripts.
- Add the three nfpm `contents` entries.
- Add the `install` block to `brews`.

Add `/completions/` to `.gitignore`.

### Acceptance test

```
just build && just test && just lint && just docs-check
./pickle completion zsh | head -1                         # #compdef pickle
./pickle completion bash | bash -n && echo bash-ok        # bash-ok (macOS bash 3.2)
command -v fish >/dev/null && ./pickle completion fish | fish -n && echo fish-ok
./pickle completion pwsh; echo "exit=$?"                  # exit=2
bash -c 'eval "$(./pickle completion bash)"; COMP_WORDS=(pickle ticket mo); COMP_CWORD=2; _pickle; echo "${COMPREPLY[@]}"'   # move
bash -c 'eval "$(./pickle completion bash)"; COMP_WORDS=(pickle ticket move T-1 in-); COMP_CWORD=4; _pickle; echo "${COMPREPLY[@]}"'   # in-development in-review
just dist-check                                           # goreleaser check passes
just dist-snapshot
tar -xOf dist/pickle_*_amd64.deb data.tar.gz | tar -tzf - | grep completion   # the 3 paths of decision 5
grep -n 'generate_completions_from_executable' dist/homebrew/*.rb             # present
```

If nfpm writes a differently compressed data member, list the deb's members with `tar -tf` and
adjust the member name. The check is that the three paths are present.

### Docs update (mandatory when user-facing)

- `docs/user-manual/installation.adoc`: a new "Shell completion" section. Homebrew and apt
  installs get completion automatically; other installs add one rc line per shell
  (`eval "$(pickle completion bash)"`, `source <(pickle completion zsh)`,
  `pickle completion fish | source`).
- `docs/user-manual/cli-reference.adoc`: an Overview row and a `[#cmd-completion]`
  `== pickle completion` section.
- `CHANGELOG.md` `[Unreleased]` *Added*: `pickle completion bash|zsh|fish`, with completion
  installed automatically by the Homebrew formula and the .deb (T-151).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs and the `CHANGELOG.md` entry above in place.
3. Write a summary: files touched, decisions made, anything deferred.
4. Suggested commit(s), Conventional Commits with the id at the end of the subject:
   - `feat(cli): add pickle completion bash|zsh|fish (T-151)`
   - `build(release): install shell completion from the Homebrew formula and the .deb (T-151)`
   - `docs: document shell completion (T-151)`
5. Root-path child: before presenting, tidy the WIP commits into the atomic commits above.
6. Commit locally. Do not push. `pickle ticket move T-151 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-11-07 — created (TO DO). source: chat: user noted pickle generates no shell completion
- 2026-11-07 — TO DO → READY: plan complete; decisions confirmed with the user 2026-11-07
