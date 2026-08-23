// Package hook installs, inspects and runs pickle's git hooks.
//
// It exists to enforce one rule of the flow that nothing else could enforce:
// code goes on a child's feature branch, while ticket and board bookkeeping is
// committed on the base branch. A squash-merge of a feature branch folds or
// drops whatever bookkeeping rode along on it, and the board then disagrees
// with the tickets it indexes.
//
// Two deliberate properties of this package:
//
//   - It is one of only two packages in pickle that shell out to git (the
//     other is internal/vcs, T-051). Keeping the dependency confined is what
//     lets `board audit` stay git-free and testable in a plain temp dir
//     (T-057 decisions 2 and 12).
//   - The installed hook is a three-line shim that calls back into this binary
//     (`pickle hooks run pre-commit`), so the rule reads the live pickle.toml
//     and cannot go stale the way a generated script with the branch prefixes
//     baked in would.
package hook

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/codcod/pickle/internal/config"
)

// ShimVersion is the version of the shim this binary writes. It is recorded in
// the marker line so `pickle doctor` can spot an older shim and `pickle
// upgrade` can refresh it. Both hooks share one version: a bump refreshes
// both, and there is no per-hook version to keep in sync.
//
// v2 (T-068): the guard-absent branch prints a one-line stderr notice instead
// of exiting silently — the same reasoning as the unexpected-exit-code line
// below: silence hides a dead guard, which is the failure this hook exists to
// prevent. The bump also fixes a cosmetic defect in v1's own marker line (a
// doubled `#`, see marker()/Shim() below) — both are shim-text changes, so
// both ride the same version bump `pickle upgrade` already refreshes on.
//
// v3 (T-082): a second hook, pre-push, joined pre-commit. Adding a hook is
// itself a shim-text change (a new shim body exists where none did), so it
// rides the same version bump rather than growing a hook-specific counter.
const ShimVersion = 3

// Name identifies one of the git hooks pickle installs.
type Name string

// The hooks pickle installs. Names iterates them in the order every plural
// operation (StatusAll, InstallAll, UninstallAll, RefreshAll) uses — the only
// place that ordering is decided, so no caller grows its own loop or its own
// order.
const (
	PreCommit Name = "pre-commit"
	PrePush   Name = "pre-push"
)

// Names returns every hook pickle installs, in the order plural operations
// process them.
func Names() []Name { return []Name{PreCommit, PrePush} }

// markerPrefix identifies a pickle-owned hook. Ownership is recorded in the
// file rather than in pickle.toml (T-057 decision 6): the file on disk is the
// single source of truth, and a config key would need in-place TOML editing to
// avoid dropping the user's comments.
const markerPrefix = "# pickle:hook v"

// ErrNoRepo reports that a directory is not a git repository, or that git is
// not available at all. Both are normal states, not failures: only `pickle
// hooks install` treats them as an error, because there the user asked for a
// hook and cannot have one (T-057 decision 12).
var ErrNoRepo = errors.New("not a git repository (or git is unavailable)")

// Shim returns the text of the named hook.
//
// The exit-code handling is the whole of T-057 decision 3 and must not be
// "simplified" to `pickle hooks run <name> || exit 1`: an *older* pickle
// first on PATH exits 2 on the unknown `hooks` verb, and treating any non-zero
// exit as a violation would block every commit or push in the repository.
// Only exit 1 means "violation". Anything else is reported and waved through,
// because a guard that cannot run must never stop a commit or push — and a
// guard that is silently dead is the exact failure this hook exists to
// prevent, so it says so.
//
// The guard-absent branch (T-068, ShimVersion 2) is held to the same rule: it
// must never grow an `exit 1` — a missing `pickle` is exactly the degraded
// state the fail-open contract exists for — but it must not stay silent
// either, for the same reason the unexpected-exit-code line below isn't
// silent.
//
// Only the one-line description comment and the invocation vary by hook
// (T-082): pre-push additionally forwards argv (`"$@"`, git's own
// `<remote-name> [<remote-url>]`) and inherits stdin unchanged, which a shell
// call does by default when neither is redirected.
func Shim(name Name) string {
	var desc, invocation string
	switch name {
	case PrePush:
		desc = "# Refuses to push a feature branch whose range still carries tickets/ paths. The rule\n" +
			"# lives in the binary so it tracks pickle.toml. Bypass one push with `git push --no-verify`.\n"
		invocation = "pickle hooks run " + string(PrePush) + " \"$@\"\n"
	default: // PreCommit
		desc = "# Refuses ticket bookkeeping (tickets/) staged on a feature branch. The rule lives in the\n" +
			"# binary so it tracks pickle.toml. Bypass one commit with `git commit --no-verify`.\n"
		invocation = "pickle hooks run " + string(PreCommit) + "\n"
	}
	return "#!/bin/sh\n" +
		marker() + " — installed by `pickle hooks install`, removed by `pickle hooks uninstall`.\n" +
		desc +
		"command -v pickle >/dev/null 2>&1 || {\n" +
		"  echo \"pickle: " + string(name) + " guard skipped (pickle not found on PATH)\" >&2\n" +
		"  exit 0                                        # guard absent, never blocking\n" +
		"}\n" +
		invocation +
		"rc=$?\n" +
		"[ \"$rc\" -eq 1 ] && exit 1                     # 1 = violation, and only 1\n" +
		"[ \"$rc\" -ne 0 ] && echo \"pickle: " + string(name) + " guard skipped (hooks run exited $rc)\" >&2\n" +
		"exit 0\n"
}

func marker() string { return markerPrefix + strconv.Itoa(ShimVersion) }

// gitAt runs git in dir. It is for the user-invoked entry points (Install,
// Uninstall, Status, Refresh), which may be called from anywhere, so the
// inherited GIT_* variables of a surrounding hook or alias are cleared: -C and
// an inherited GIT_DIR would otherwise disagree about which repository is meant.
func gitAt(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = withoutRepoEnv(os.Environ())
	return run(cmd)
}

// repoEnv are the variables that pin git to a specific repository, index or
// prefix. A surrounding hook or alias exports them, and they would override the
// -C that gitAt relies on.
var repoEnv = []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_PREFIX"}

// withoutRepoEnv removes repoEnv entries from an environment. They must be
// *removed*, not blanked: git reads GIT_DIR="" as an empty repository path and
// fails with `not a git repository: ”` rather than falling back to discovery.
func withoutRepoEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(repoEnv, name) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// gitHere runs git with the caller's working directory and environment left
// exactly as they are, and is the *only* helper the PreCommit path may use
// (T-057 decision 11). git invokes pre-commit from the top of the worktree with
// GIT_INDEX_FILE set to a relative path, and inside a linked worktree with
// GIT_DIR pointing at .git/worktrees/<name>; a `git -C <root>` call would
// therefore inspect a different index than the one being committed.
func gitHere(args ...string) (string, error) {
	return run(exec.Command("git", args...))
}

func run(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(cmd.Args[1:], " "), detail)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// HooksDir resolves the hooks directory of the repository at root.
//
// It asks git rather than assuming .git/hooks, because that answer honours an
// existing core.hooksPath — so a project that already redirects its hooks
// (Husky, Lefthook, pre-commit.com) gets the shim where git will actually look
// for it instead of in a silently-dead location. Setting core.hooksPath
// ourselves was rejected for the mirror-image reason: it is repo-global and
// would evict those tools (T-057 decision 5).
func HooksDir(root string) (string, error) {
	out, err := gitAt(root, "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err == nil {
		return out, nil
	}
	// git < 2.31 has no --path-format; its --git-path answer is relative to the
	// repository, so resolve it against root by hand.
	out2, err2 := gitAt(root, "rev-parse", "--git-path", "hooks")
	if err2 != nil {
		return "", fmt.Errorf("%w: %s", ErrNoRepo, root)
	}
	if !filepath.IsAbs(out2) {
		out2 = filepath.Join(root, out2)
	}
	return out2, nil
}

// Kind is what pickle found at the hook path.
type Kind string

// The states a hook can be in, from pickle's point of view.
const (
	KindNoRepo  Kind = "no-repo" // not a git repository, or git unavailable
	KindAbsent  Kind = "absent"  // no hook of this name at all
	KindOwned   Kind = "owned"   // written by pickle (carries the marker)
	KindForeign Kind = "foreign" // someone else's hook — never touched
)

// State describes one installed hook.
type State struct {
	Name    Name
	Kind    Kind
	Path    string // resolved hook path (empty when there is no repository)
	Version int    // shim version from the marker, when Kind is KindOwned
	Stale   bool   // owned, but written by an older pickle
}

// Result records what an Install/Uninstall/Refresh call did, for the CLI
// summary. Changed is false under a dry run — Would carries the intent instead.
// Kind is the state the call found, so callers can branch on it without parsing
// Skipped.
type Result struct {
	Name    Name
	Path    string
	Kind    Kind
	Changed bool   // the file was written or removed
	Would   bool   // dry run: the file would have been removed
	Skipped string // why nothing happened, for the caller to report verbatim
}

// Status inspects the named hook of the repository at root. A missing
// repository is reported as KindNoRepo, not as an error.
func Status(root string, name Name) (State, error) {
	dir, err := HooksDir(root)
	if err != nil {
		if errors.Is(err, ErrNoRepo) {
			return State{Name: name, Kind: KindNoRepo}, nil
		}
		return State{Name: name}, err
	}
	path := filepath.Join(dir, string(name))
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Name: name, Kind: KindAbsent, Path: path}, nil
		}
		return State{Name: name, Path: path}, err
	}
	v, ok := markerVersion(string(body))
	if !ok {
		return State{Name: name, Kind: KindForeign, Path: path}, nil
	}
	return State{Name: name, Kind: KindOwned, Path: path, Version: v, Stale: v != ShimVersion}, nil
}

// StatusAll inspects every hook pickle installs, in Names() order.
//
// KindNoRepo is a whole-repository property, not a per-hook one: the first
// hook to observe it short-circuits the rest, so a caller printing one line
// per State never prints "no git repository" once per hook.
func StatusAll(root string) ([]State, error) {
	var out []State
	for _, name := range Names() {
		st, err := Status(root, name)
		if err != nil {
			return out, err
		}
		if st.Kind == KindNoRepo {
			return []State{st}, nil
		}
		out = append(out, st)
	}
	return out, nil
}

// markerVersion extracts the shim version from a pickle-owned hook.
func markerVersion(body string) (int, bool) {
	i := strings.Index(body, markerPrefix)
	if i < 0 {
		return 0, false
	}
	rest := body[i+len(markerPrefix):]
	end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	if end == 0 {
		return 0, false
	}
	if end < 0 {
		end = len(rest)
	}
	v, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return v, true
}

// Install writes the named shim. A foreign hook of that name is refused
// rather than overwritten — someone else's hook is not pickle's to replace —
// unless force is set. Not being a git repository is an error here and
// nowhere else: the user asked for a hook.
func Install(root string, name Name, force bool) (Result, error) {
	st, err := Status(root, name)
	if err != nil {
		return Result{Name: name}, err
	}
	if st.Kind == KindNoRepo {
		return Result{Name: name}, fmt.Errorf("%w: %s", ErrNoRepo, root)
	}
	if st.Kind == KindForeign && !force {
		return Result{Name: name, Path: st.Path, Kind: st.Kind, Skipped: "not pickle's — left in place"}, fmt.Errorf(
			"%s already exists and was not written by pickle; left untouched.\n"+
				"Re-run with --force to replace it, or chain the guard from your own hook:\n"+
				"    pickle hooks run %s || exit 1",
			st.Path, name)
	}
	if st.Kind == KindOwned && !st.Stale {
		if cur, err := os.ReadFile(st.Path); err == nil && string(cur) == Shim(name) {
			return Result{Name: name, Path: st.Path, Kind: st.Kind, Skipped: "current"}, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(st.Path), 0o755); err != nil {
		return Result{Name: name, Path: st.Path, Kind: st.Kind}, err
	}
	if err := writeExecutable(st.Path, Shim(name)); err != nil {
		return Result{Name: name, Path: st.Path, Kind: st.Kind}, err
	}
	return Result{Name: name, Path: st.Path, Kind: KindOwned, Changed: true}, nil
}

// InstallAll installs every hook pickle ships, in Names() order. A repository
// that does not exist at all is a single error, exactly as a single Install
// call would report — trying the second hook after the first already failed
// that way would just repeat the same error. A per-hook failure past that
// point (a foreign hook without force) does not stop the rest: every hook is
// still attempted, and every attempt's Result is returned, so the caller can
// report each one and the caller decides how to surface the combined error.
func InstallAll(root string, force bool) ([]Result, error) {
	var out []Result
	var errs []error
	for _, name := range Names() {
		res, err := Install(root, name, force)
		out = append(out, res)
		if err != nil {
			if errors.Is(err, ErrNoRepo) {
				return out, err
			}
			errs = append(errs, err)
		}
	}
	return out, errors.Join(errs...)
}

// writeExecutable writes body to path with mode 0o755. WriteFile does not
// change the mode of an existing file, so chmod separately: a shim that is not
// executable is a hook git silently ignores.
func writeExecutable(path, body string) error {
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

// Uninstall removes a pickle-owned hook of the given name. A foreign hook, an
// absent hook and a missing repository are all reported as skipped, never as
// errors: uninstall is called from `pickle uninstall` on trees that may never
// have had a hook.
func Uninstall(root string, name Name, dryRun bool) (Result, error) {
	st, err := Status(root, name)
	if err != nil {
		return Result{Name: name}, err
	}
	switch st.Kind {
	case KindNoRepo:
		return Result{Name: name, Kind: st.Kind, Skipped: "no git repository"}, nil
	case KindAbsent:
		return Result{Name: name, Path: st.Path, Kind: st.Kind, Skipped: "absent"}, nil
	case KindForeign:
		return Result{Name: name, Path: st.Path, Kind: st.Kind, Skipped: "not pickle's — left in place"}, nil
	}
	if dryRun {
		return Result{Name: name, Path: st.Path, Kind: st.Kind, Would: true}, nil
	}
	if err := os.Remove(st.Path); err != nil {
		return Result{Name: name, Path: st.Path, Kind: st.Kind}, err
	}
	return Result{Name: name, Path: st.Path, Kind: st.Kind, Changed: true}, nil
}

// UninstallAll removes every hook pickle owns, in Names() order. A missing
// repository is reported once (Status already turns it into a per-hook
// KindNoRepo Result carrying no error), so this still returns one Result per
// hook — unlike StatusAll, there is no exec cost to dedupe here, and callers
// already handle KindNoRepo as an ordinary skip per hook.
func UninstallAll(root string, dryRun bool) ([]Result, error) {
	var out []Result
	for _, name := range Names() {
		res, err := Uninstall(root, name, dryRun)
		out = append(out, res)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// Refresh rewrites an owned shim that an older pickle wrote. It never installs
// a hook that is absent (the hook is opt-in, and `pickle upgrade` must not
// arm a guard the user did not ask for) and never touches a foreign one.
func Refresh(root string, name Name) (Result, error) {
	st, err := Status(root, name)
	if err != nil {
		return Result{Name: name}, err
	}
	if st.Kind != KindOwned || !st.Stale {
		return Result{Name: name, Path: st.Path, Kind: st.Kind}, nil
	}
	if err := writeExecutable(st.Path, Shim(name)); err != nil {
		return Result{Name: name, Path: st.Path, Kind: st.Kind}, err
	}
	return Result{Name: name, Path: st.Path, Kind: st.Kind, Changed: true}, nil
}

// RefreshAll refreshes every stale, owned hook pickle installs, in Names()
// order.
func RefreshAll(root string) ([]Result, error) {
	var out []Result
	for _, name := range Names() {
		res, err := Refresh(root, name)
		out = append(out, res)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// maxListedPaths caps the offending paths quoted in a rejection message.
const maxListedPaths = 10

// ticketsOffenders splits a NUL-separated `git diff -z` output (from either
// CheckPreCommit's staged diff or CheckPrePush's ref-range diff) and returns
// every path lying under prefix (tickets/ itself, or nested inside it). The
// two callers differ only in which git diff produced diffOutput (T-042 item 5).
func ticketsOffenders(diffOutput, prefix string) []string {
	var offenders []string
	for _, p := range strings.Split(diffOutput, "\x00") {
		if p == "" {
			continue
		}
		if p == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(p, prefix) {
			offenders = append(offenders, p)
		}
	}
	return offenders
}

// CheckPreCommit is the rule: reject a commit that stages ticket bookkeeping
// while HEAD is a feature branch of any registered child. It reports ok=false
// only for a real violation; every other outcome — detached HEAD, a merge in
// progress, no git, an unreadable index — is ok=true, because a guard that
// cannot decide must not block (T-057 decision 4).
//
// It takes no root: per T-057 decision 11 it inspects the repository of the
// current working directory through gitHere, which is the index git is actually
// about to commit. cfg supplies the branch prefixes and the tickets/ location.
//
// Named CheckPreCommit, not PreCommit, because Name's own PreCommit constant
// (this file) already claims that identifier — the two are easy to conflate
// but answer different questions ("which hook is this" vs "is this commit a
// violation").
//
// Not covered, by design: `git commit --amend` of a commit that *already*
// contains tickets/ paths. The hook does run, but `git diff --cached` compares
// the index with HEAD and so reports nothing to object to.
func CheckPreCommit(cfg *config.Config, w io.Writer) (bool, error) {
	// symbolic-ref, never `rev-parse --abbrev-ref HEAD`: on an unborn branch (a
	// fresh `git init`, which is exactly where `pickle install` lands) rev-parse
	// exits 128 and prints the literal "HEAD", which reads as a detached HEAD
	// and would wave bookkeeping through on a feature branch. symbolic-ref
	// answers "main" there, and exits 1 only when HEAD really is detached.
	branch, err := gitHere("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || branch == "" {
		return true, nil // detached HEAD, no repository, or no git
	}

	// A merge, cherry-pick or revert being concluded commits files the user did
	// not stage by hand; refusing there would leave them stuck mid-operation.
	// (No rebase check: `rebase --continue` does not run pre-commit, and HEAD is
	// detached during a rebase anyway, so the branch test above already passed.)
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"} {
		p, err := gitHere("rev-parse", "--git-path", name)
		if err != nil {
			return true, nil
		}
		if _, err := os.Stat(p); err == nil {
			return true, nil
		}
	}

	if !onFeatureBranch(cfg, branch) {
		return true, nil
	}

	prefix, ok := ticketsPrefix(cfg)
	if !ok {
		return true, nil // tickets/ is outside this repository — nothing to guard
	}

	staged, err := gitHere("diff", "--cached", "--name-only", "-z")
	if err != nil {
		return true, nil
	}
	offenders := ticketsOffenders(staged, prefix)
	if len(offenders) == 0 {
		return true, nil
	}
	writeRejection(w, branch, prefix, offenders)
	return false, nil
}

// onFeatureBranch reports whether branch looks like a feature branch of any
// registered child. The prefixes are unioned rather than resolved per child:
// one repository can hold branches for several children, and the hook has no
// way to know which child a given branch belongs to.
func onFeatureBranch(cfg *config.Config, branch string) bool {
	if cfg == nil {
		return false
	}
	for _, p := range cfg.Projects {
		prefix := p.BranchPrefix
		if prefix == "" {
			prefix = config.DefaultBranchPrefix
		}
		if strings.HasPrefix(branch, prefix) {
			return true
		}
	}
	return false
}

// ticketsPrefix is the tickets/ directory as git would name it: relative to the
// top of the worktree, slash-separated, with a trailing slash. It reports
// ok=false when tickets/ lies outside this repository, which is the multi-repo
// case — the child being committed is not the repo that holds the board.
func ticketsPrefix(cfg *config.Config) (string, bool) {
	if cfg == nil {
		return "", false
	}
	top, err := gitHere("rev-parse", "--show-toplevel")
	if err != nil || top == "" {
		return "", false
	}
	// Resolve symlinks on both sides before comparing. git reports the real path
	// (on macOS /tmp is a symlink to /private/tmp), while the config root is
	// whatever path the caller walked in on; without this the two never match
	// and the guard would quietly never fire.
	rel, err := filepath.Rel(realpath(top), realpath(filepath.Join(cfg.Root(), "tickets")))
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return strings.TrimPrefix(rel, "./") + "/", true
}

// realpath resolves symlinks, falling back to the input when it cannot (a path
// that does not exist yet is still worth comparing).
func realpath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// writeRejection explains the refusal: what was staged, why it matters, and the
// two ways out. A guard that only says "no" gets disabled.
func writeRejection(w io.Writer, branch, prefix string, offenders []string) {
	fmt.Fprintf(w, "pickle: refusing to commit ticket bookkeeping on a feature branch.\n\n")
	fmt.Fprintf(w, "  branch:  %s\n", branch)
	shown := offenders
	if len(shown) > maxListedPaths {
		shown = shown[:maxListedPaths]
	}
	for i, p := range shown {
		label := "  staged:  "
		if i > 0 {
			label = "           "
		}
		fmt.Fprintf(w, "%s%s\n", label, p)
	}
	if n := len(offenders) - len(shown); n > 0 {
		fmt.Fprintf(w, "           … and %d more\n", n)
	}
	fmt.Fprintf(w, "\nTicket and board bookkeeping belongs on the base branch. Squash-merging this\n"+
		"branch would fold these files into one commit — or drop them — and the board\n"+
		"would then disagree with the tickets it indexes.\n\n")
	// prefix is relative to the top of the worktree, so it is "tickets/" only when
	// the config root *is* the repository root — a child registered deeper renders
	// "sub/dir/tickets/". The comment column is therefore computed rather than
	// hardcoded: a ragged remedy block reads like a typo.
	remedy := [][2]string{
		{"git restore --staged " + prefix, "unstage the bookkeeping"},
		{`git commit -m "…"`, "commit the code alone"},
		{"git switch <base branch>", "and commit the bookkeeping there"},
	}
	width := 0
	for _, r := range remedy {
		if n := len([]rune(r[0])); n > width {
			width = n
		}
	}
	for _, r := range remedy {
		fmt.Fprintf(w, "  %s%s  # %s\n", r[0], strings.Repeat(" ", width-len([]rune(r[0]))), r[1])
	}
	fmt.Fprintf(w, "\nIf the change genuinely belongs on this branch — its product lives under %s —\n"+
		"bypass the guard for one commit:\n\n"+
		"  git commit --no-verify\n", prefix)
}
