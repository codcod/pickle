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
   prints `flag`-package usage, the flags printed equal the spec's; for every command whose
   `--help` prints a one-line usage string instead (`ticket new|set|move`, `changelog check`,
   `retro`, `board sync|state`, `serve`), the spec's flags equal the `--name`/`-x` tokens in it.
   Exempt from (b), by name in the test: `project add` (its usage says only `[flags]`), and the
   commands that need a `pickle.toml` before reading `--help` or take no flags.
5. **Packaging: Homebrew via `generate_completions_from_executable`; the .deb via a goreleaser
   `before` hook.** User decision, 2026-11-07.
   - The `brews` entry gets an `install` block: `bin.install "pickle"` plus
     `generate_completions_from_executable(bin/"pickle", "completion")`. Homebrew then runs
     `pickle completion bash|zsh|fish` itself.
   - A `before` hook writes `completions/pickle.bash`, `completions/_pickle` and
     `completions/pickle.fish` (`sh -c 'go run . completion <sh> > …'` — goreleaser runs hooks
     without a shell, so the redirect needs one).
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

### Round 1 — 2026-11-08

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | None of the three scripts falls back to filename completion, so installing them (automatic via Homebrew and the .deb) removes path completion for value-taking flags such as `install --path`, `serve --dir`, `changelog check --changelog` | `bash -c 'eval "$(./pickle completion bash)"; COMP_WORDS=(pickle install --path RE); COMP_CWORD=3; _pickle; echo "[${COMPREPLY[@]}]"'` → `[]`, with the bash script registered without `-o default`; zsh `compadd`s only the word list; fish sets a global `complete -c pickle -f` | bash `complete -o default`; zsh `_files` when the word list is empty; fish suppresses files only when the previous token is not a flag; add a bash behaviour case |
| F2 | non-blocking | test-gap | noted | The 4(b) exemption for `hooks run` matches only that exact path, so `walk` still reaches `hooks run pre-commit` and `hooks run pre-push` and runs the real guards with `--help` as an argument | `completion_test.go` exempt map; `hooks.go` `runHooksRun` dispatches `args[1:]` to the guard. Harmless today (TestMain's sandbox CWD is not a git repo) but not hermetic | Exempt by prefix |
| F3 | non-blocking | design | noted | `ticket move` offers status tokens at every non-flag position, including the ticket-id slot and after `--reason` | Both reviewers: `pickle ticket move <TAB>` offers `to-do ready …`. Ticket ids are out of scope (T-151 decision 3), so nothing better can be offered there | Offer statuses only after a positional |
| F4 | non-blocking | design | noted | The bare status token is derived with `strings.Cut(Dir, "-")`, not `flow.ByToken`'s `^\d+-` strip, so the two would disagree for a flow whose dirs lack a numeric prefix | `completion.go` `completionSpec`; only the brine flow exists today, and its dirs all carry the prefix | Export a bare-token helper from `flow` if a second flow ever lands |
| F5 | non-blocking | design | noted | `pickle completion --help` exits 2 on stderr, while the other command groups answer `-h/--help` with exit 0 | T-151 decision 1 mandates exit 2 for any argument other than a shell name | Revisit only alongside decision 1 |
| F6 | non-blocking | design | noted | fish evaluates one `__pickle_is` condition per flag (about 55) on each TAB, and each recomputes the command path | `renderFish` | One `complete` line per command path if latency is ever reported |
| F7 | non-blocking | design | noted | Shell `case` and `echo` words are rendered with Go's `%q`, which is not shell quoting | `renderShHelpers`; every spec word is plain ASCII today, so the output is valid shell | Single-quote helper if a spec word ever needs it |
| F8 | non-blocking | design | noted | `renderFish` compiles a regexp only to strip a `--` prefix | `completion.go` `renderFish` | `strings.CutPrefix` |

Disposition summary: 1 blocking (F1 → rework); 7 non-blocking, all noted (F2–F8); 0 fixed inline, 0 folded, 0 new tickets.
cost: estimated S, actual S

- [x] Reviewer independence settled (step 0): delegated — the orchestrator authored the branch in this session; steps 1–4a ran in a fresh sub-agent (`hunt: angle 7 only`). Delegated findings discarded: none.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned the branch had T-151 in `3-in-development`; rebased onto `main`, re-run clean.
- [x] Implementation audit (steps 1, 2): Tasks 1–3 met; acceptance re-run verbatim green (fish check skipped — fish not installed); decisions 1–6 honoured.
- [x] Correctness hunt (step 3): host tool path — `/code-review` (medium) run by the orchestrator on `main...feat/T-151-shell-completion`, no sub-agents; 8 tool findings, 1 discarded on re-verification (the Homebrew default shell list includes `:pwsh`: not reproduced — the installed Homebrew's `Utils::ShellCompletion.default_completion_shells(nil)` returns `[:bash, :zsh, :fish]`); angle 7 run separately by the delegated reviewer, no findings (hostile `COMP_WORDS` under bash 5, bash 3.2 and zsh executed and globbed nothing). F1 reproduced.
- [x] Consistency audit (step 4): no contradictions.
- [x] Documentation audit (step 4a): coverage met (installation, cli-reference, CHANGELOG); sweep clean; `just docs-check` green.
- [ ] Docs-readability pass (step 4b): conscious skip — optional.
- [x] Findings recorded (step 5).
- [x] Ticket moved to `tickets/5-rework/` (step 6a).
- [ ] Other references (step 7) — deferred to the concluding round.
- [ ] Impact sweep (step 8) — deferred to the concluding round.
- [ ] Summary + approval (step 9) — deferred to the concluding round.

### Rework fix record — round 1 (commit 86d55bc)

- F1 fixed: bash registers `complete -o default -F _pickle pickle`; zsh runs `compadd -- $cands || _files`, so filenames are offered whenever no spec word matches; fish keeps `-f` (no files) only while the previous token is not a flag. Regression test `TestCompletionFallsBackToFiles` asserts each script's fallback line and that bash (5 and `/bin/bash` 3.2) registers `-o default`. zsh fallback verified in a pty: `pickle install --path RE<TAB>` → `README.md`, `pickle tick<TAB>` → `ticket`. fish unverified (not installed).

## History

- 2026-11-07 — created (TO DO). source: chat: user noted pickle generates no shell completion
- 2026-11-07 — TO DO → READY: plan complete; decisions confirmed with the user 2026-11-07
- 2026-11-08 — plan amended inline: decision 4(b) widened to check one-line usage strings and names its exemptions (gate F1); decision 5's before-hook wrapped in `sh -c`, since goreleaser hooks run without a shell (gate F2). Gate F3 (skip the `pickle <command>` usage line) and F4 (completion usage description on its own line) noted and closed
- 2026-11-08 — READY → IN DEVELOPMENT: picked up
- 2026-11-08 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-11-08 — IN REVIEW → REWORK: review round 1: 1 blocking finding (F1, no filename fallback)
- 2026-11-08 — REWORK → IN REVIEW: findings fixed
