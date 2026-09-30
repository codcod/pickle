---
id: T-140
title: board audit finds the merge of a DONE ticket in git history and prints the line to record
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-140 — board audit finds the merge of a DONE ticket in git history and prints the line to record

## Outcome

After this ships, `pickle board audit` prints, under each "DONE but has no 'MERGED'" warning it
can match in git history, the exact `merged to <base> (…)` History line to append, and every brine
trigger starts by fetching the base branch and recording those lines. A merge gets recorded
without the human having to say "pr N merged".

## Description

Evidence, 2026-09-21..29: about 25 "pr N merged" / "record it" prompts, and 17 "DONE but has no
MERGED" audit warnings. T-092 detects the missing line and T-133 scoped its noise; neither finds
the merge.

Checked against the child repos: porth and smppai merge with "Merge pull request #N" subjects
over commits ending `(POR-012)` / `(SMP-026)`; unity has no remote (local base); bookkeeping
commits use `board: <ID> …`, never the trailing form, so they cannot match. Checked against this
repo (2026-09-29): every merge is a GitHub merge commit, `Merge pull request #95 from
codcod/feat/T-134-serve-where-lookup`, and recorded merge lines cite **that merge commit's** SHA
(`PR #95, 2daf1f1`), not a ticket commit's.

**Detection (read-only, offline, local git only).** For each DONE ticket with no merge line, in
its child's repository (`pickle.toml` path; under `umbrella` that is the child's own repo), scan
the commits reachable from the base ref: `origin/HEAD`, else `origin/main`, else `origin/master`,
else local `main`/`master` (a child with no remote). A ticket is found merged by, newest first:

1. a **merge commit** whose subject names the ticket's branch, `<branch_prefix><ID>-…` (GitHub's
   `Merge pull request #N from owner/feat/T-134-…`; GitLab's `Merge branch 'feat/T-134-…'`). It
   cites that commit, with `#N` / `!N` from the subject as the MR ref; or
2. a commit whose subject classifies as child-project code for the id, meaning the trailing
   `(<ID>)` form that `changelog check` already recognises. This reuses `changelog`'s id scanner
   rather than adding a second parser. That covers squash merges (`… (T-5) (#31)`, with the
   `(#31)` suffix as the MR ref) and fast-forward or kept-history merges with no ref.

The suggested line is `- <commit date> — merged to <base> (<PR #N|MR !N>, <short SHA>[, <commit
URL>])`, with the MR ref omitted when there is none and the URL only for a `github.com` /
`gitlab.com` origin. The audit **never fetches and never writes**: a stale remote ref just means
no suggestion, and the warning stays as it is today. It adds no flag, so nothing is added to the
stable surface. `audit.Audit` itself stays git-free: `ticket move`, `board state` and `serve`
all call it and must not start shelling out to git. The lookup runs only in the `board audit`
command, and only for a child that has at least one DONE-unmerged ticket.

No forge API: NOTES.md § Field-finding triage (2026-09-20) rejected teaching WIP about PR state
because it needed pickle's first GitHub/API dependency. This reads only git.

**The audit alone does not remove the "pr N merged" prompt**: it runs only when asked and never
fetches, so right after a merge the local `origin/<base>` does not have it yet. The payload side
removes the prompt. At the start of every brine trigger, `git fetch` each child's base, run
`pickle board audit`, and, on the base branch of the repository that holds `tickets/`, append
each printed line, `board sync` and commit it as `board: <ID> record merge`. On any other branch,
only name the merges found, since bookkeeping never goes on a feature branch (rules §0), and let
the next trigger run on base record them. A fetch failure is reported and the trigger continues.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-140-audit-finds-merges
```

### Prerequisite gate (hard)

None.

### Confirmed design decisions (do not deviate without asking)

1. **Printed line only, no writer.** `board audit` prints the line and the agent (or human)
   appends it, then runs `pickle board sync`. There is no `--fix` and no `ticket` subcommand: a
   read-only audit stays read-only, and no stable-surface flag is added.
2. **`audit.Audit` stays git-free.** The lookup lives in the `board audit` CLI path only. Its
   other callers (`ticket move`, `board state --json`, `pickle serve`) are unchanged.
3. **A merge commit naming the branch wins over a ticket commit.** It is what this repo's
   recorded lines cite. The newest `(<ID>)`-trailing commit is the fallback (squash,
   fast-forward, kept history). Among several matches the newest wins, and the date is that
   commit's committer date (`%cs`), not today.
4. **Branch match is exact on the id.** `<branch_prefix><ID>-` or `<branch_prefix><ID>` at the
   end of the branch name, so `feat/T-50-x` never matches T-5. `board:` subjects never match
   (`changelog.ClassifySubject` already classifies them as Bookkeeping).
5. **Base ref order:** `refs/remotes/origin/HEAD` → `origin/main` → `origin/master` → local
   `main`/`master`. `<base>` in the line is the short branch name (`main`). No base resolves, or
   the child is not a git repository: no suggestion, no error.
6. **Commit URL only for `github.com` / `gitlab.com` origins** (`https://…/commit/<short sha>`,
   GitLab `…/-/commit/<short sha>`, the same 7-character SHA the line cites; from `https://` or `git@host:owner/repo(.git)` remotes). Any
   other host gets SHA only, and the rules make the link optional anyway.
7. **Payload: record only on base.** At the start of every trigger, fetch, audit and record. Off
   the base branch, the agent reports what it found and records nothing. A fetch failure is
   reported and never stops the trigger.

### Tasks

#### Task 1 — pure matcher, `internal/changelog/merge.go` (new)

- `type Commit struct{ SHA string; Parents int; Date, Subject string }`.
- `func FindMerge(log []Commit, id, branchPrefix string, prefixes []string) (Commit, string,
  bool)`. `log` is newest first; it returns the commit to cite, the MR ref (`PR #N` / `MR !N`, or
  "") and whether one was found. Pass 1 covers merge commits (`Parents > 1`) whose subject
  contains `branchPrefix+id` followed by `-`, `'`, whitespace or end of string, with the ref from
  the first `#N` / `!N` token in the subject, if any. Pass 2 covers the newest
  commit where `classifySubject` returns `ChildProject` with this id, with the ref from the
  existing `prTokenRE` suffix. Reuse `newIDPatterns`/`classifySubject`/`prTokenRE`, not copies.
- `func CommitURL(remote, sha string) string` implements decision 6. It returns "" for an unknown
  host.
- `func MergeLine(c Commit, base, ref, url string) string` renders
  `<date> — merged to <base> (<ref>, <short sha>[, <url>])`, dropping an empty ref.

#### Task 2 — base resolution, `internal/vcs/vcs.go`

- `func ResolveBase(root string) (ref, name string, ok bool)` implements decision 5. It uses
  `Output`, and its local-branch fallback is `ResolveLocalBase`, not a copy of it.

#### Task 3 — wire into `board audit`, `internal/cli/board.go`

- In `runBoardAudit`, after `audit.Audit`, load tickets (`ticket.LoadAll`). For each child with
  ≥1 ticket in `def.DependencySatisfied().Dir` and `!ticket.HasMergeLine`, resolve the child
  repo (`filepath.Join(cfg.Root(), p.Path)`) and run `ResolveBase` once and
  `vcs.Output(repo, "log", "--format=%H%x09%P%x09%cs%x09%s", ref)` once. Build a
  `[]changelog.Commit` from that output, then call `FindMerge` for each of the child's unmerged
  tickets, using `p.BranchPrefix` and `ticketPrefixes(cfg)`. Get the origin URL with
  `vcs.Output(repo, "remote", "get-url", "origin")`; an error means no URL.
- Key the suggestions by the warning's ref (`t.Dir + "/" + filepath.Base(t.Path)`). While
  printing `res.Warnings`, print `  → found in git, record: <line>` directly under the warning
  that starts with `ref + ": DONE but has no 'MERGED'"`. The warning count and the exit code are
  unchanged.
- Any git failure means no suggestion for that child, silently. The warning already tells the
  reader what to do.

#### Task 4 — payload, `skill/SKILL.md` and `skill/resources/tickets-README.md`

- `SKILL.md`: add a short section right after `## When to use`, `## Before every procedure:
  record merges`, stating decision 7 in about five lines: fetch each child's base
  (`git -C <child> fetch origin <base>`, skipped for a child with no remote; failure reported,
  continue), run `pickle board audit`, and on the base branch of the repository holding
  `tickets/` append each printed line under its ticket's `## History`, run `pickle board sync`
  and commit `board: <ID>[, <ID> …] record merge` with explicit pathspecs. Off base, name the
  merges and record nothing.
- `tickets-README.md` §3, "When the human reports a merge, append …": widen it to "when the
  human reports a merge, or `pickle board audit` prints the line it found in git history". Keep
  the rest of the sentence.
- The DONE-unmerged warning in `internal/audit/audit.go` (and its quote in
  `cli-reference.adoc`) cites "rules §4" for the merge line; it lives in §3. Correct it to §3.
- Both must pass `payload_lint_test.go`: no ticket-lookup shapes and no repo-only paths.

#### Task 5 — tests

- `internal/changelog/changelog_test.go` (or `merge_test.go`), table-driven `FindMerge`:
  GitHub merge commit naming `feat/T-5-x` gives that commit and `PR #95`; a squash
  `feat(cli): x (T-5) (#31)` gives that commit and `PR #31`; a plain `fix: y (T-5)` gives the
  commit and ""; `board: T-5 done` gives no match; a merge of `feat/T-50-x` and a commit ending
  `(T-50)` give no match for T-5; merge commit and ticket commit both present gives the merge
  commit; two ticket commits give the newest; an unregistered prefix gives no match. `CommitURL`:
  github https, github ssh, gitlab, another host gives "".
- `internal/cli` test, using the `gitInit` (`hooks_test.go`) / `writeAndCommit` (`changelog_test.go`) helpers:
  an in-tree fixture with a DONE ticket T-1 with no merge line, one DONE ticket T-2 not in git,
  and a commit `feat: x (T-1)` on local `main` with no remote. `board audit` prints
  `→ found in git, record: <date> — merged to main (<sha7>)` under T-1's warning and nothing
  under T-2's, reports 2 warnings, and exits 0.

### Acceptance test

```
just build && just test && just lint && just docs-check
```

All green, including the new tests above and `payload_lint_test.go`.

Real-history check, in a throwaway clone with the binary renamed per the self-modify policy:

```
D=$(mktemp -d) && git clone -q . "$D/repo" && cp pickle "$D/pickle-test" && cd "$D/repo" \
  && git remote set-url origin https://github.com/codcod/pickle.git \
  && sed -i '' '/merged to main (PR #95/d' tickets/6-done/T-134-*.md \
  && sed -i '' '/merged to main (PR #31/d' tickets/6-done/T-093-*.md \
  && ../pickle-test board sync && ../pickle-test board audit
```

Expected, each under its ticket's DONE-but-unmerged warning:

- T-134: `2026-09-29 — merged to main (PR #95, 2daf1f1, https://github.com/codcod/pickle/commit/2daf1f1)`
- T-093: `2026-08-12 — merged to main (PR #32, 052510d, https://github.com/codcod/pickle/commit/052510d)`.
  Its branch was merged twice (#31 `212730c`, then #32 `052510d`), so this checks decision 3's
  "newest wins" against real history. `board sync` first, because the `sed` leaves BOARD.md stale.

Plus `board audit: … 2 warning(s)`, exit 0.

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`, `pickle board audit`: extend the "every ticket in
  `6-done/` carries a `merged to <base>` History line" bullet with the git lookup. Say where it
  looks (decision 5), what it matches (decisions 3–4), that it never fetches or writes, and give
  a sample `→ found in git, record:` line.
- `docs/user-manual/your-first-project.adoc` § "7. Merge it — yours": after you merge, the next
  brine trigger on the base branch fetches and records the line for you. By hand is still the
  way when the merge left no trace git can match.
- `CHANGELOG.md` `[Unreleased]` → `### Added`: one entry (T-140). It is additive, targeted at
  1.1.0 with T-141.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated as above.
3. Write a summary: files touched, decisions honoured, anything deferred.
4. Suggested commits, tidied into atomic ones (root-path child), e.g.:

   ```
   feat(audit): find a DONE ticket's merge in git history and print the line to record (T-140)
   feat(skill): record found merges at the start of every trigger (T-140)
   docs: document the board audit merge lookup (T-140)
   ```

5. Commit locally on `feat/T-140-audit-finds-merges`. Do not push or open a PR without explicit
   approval. After approval (in-tree): `git fetch origin main && git diff --name-only
   origin/main...HEAD | grep '^tickets/'` must print nothing; then push and open the PR. Merging
   is the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: merges were recorded only when the human reported them, ~25 times in a week
- 2026-09-29 — TO DO → READY: plan complete
- 2026-09-30 — applicability gate (fresh sub-agent), routing approved by the user. Blocking, fixed in the acceptance block while still READY: T-093's branch was merged twice, so newest-wins cites `PR #32, 052510d`, not #31; the `sed` leaves BOARD.md stale, so run `board sync` before the audit or it exits 1. Non-blocking, amended inline: the commit URL uses the short SHA (decision 6); Task 4 corrects the existing warning's "rules §4" to §3. Non-blocking, noted and closed: GitLab's `!N` is in the merge body and not the subject, so GitLab merges get an empty ref (no registered child uses GitLab); `gitInit` lives in `hooks_test.go`; NOTES' "1.1.0 = T-139 + T-140" is stale since T-139 was dropped (recorded in NOTES.md).
- 2026-09-30 — READY → IN DEVELOPMENT: picked up
