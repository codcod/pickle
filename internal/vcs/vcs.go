// Package vcs answers one narrow question for a registered child-project:
// would the overarching repository stage this nested directory into itself?
//
// A child registered at a path other than "." is, from git's point of view,
// an ordinary directory the moment it appears — nothing marks it as another
// repository's territory until a `.gitignore` entry (or an index entry, for a
// deliberate submodule/gitlink) says otherwise. This package asks git itself
// rather than parsing `.gitignore` text, because a textual scan misses
// `.git/info/exclude`, negations and nested ignore files and would produce a
// warning that can never be silenced (T-051).
//
// It mirrors internal/hook's git plumbing deliberately rather than sharing
// it: same repo-pinning-variable scrub, same exec-with-timeout shape. The two
// packages are the only ones in pickle that shell out to git, each confined
// to its own file for exactly the reason internal/hook's package doc gives —
// keeping the dependency out of the packages that must stay testable in a
// plain temp dir (board audit, ticket loading).
//
// Three limits of asking git this way, all accepted (T-051 review, F7/F8/R2):
//
//   - The question is answered by whichever repository git discovers from
//     root. If root is not itself a repository but sits inside one, that is
//     the *enclosing* repo — and then the advice is not merely written in the
//     wrong file but is the wrong pattern: Advice anchors the entry at root
//     ("/child/"), while the enclosing repo needs it anchored at its own
//     toplevel ("/sub/child/"). pickle cannot render a usable entry for that
//     layout, so treat the suggestion as indicative there.
//   - The entry is rendered literally, so a child whose directory name
//     contains a gitignore metacharacter (`[`, `]`, `\`, `*`, `?`) yields a
//     pattern that matches something other than itself — `/foo[1]/` does not
//     match `foo[1]`. Escaping is left undone deliberately: such names are
//     vanishingly rare and the wrong escape would be worse than none.
//   - Tracked is inferred from `ls-files` matching anything under the path, so
//     a child whose contents were already committed as ordinary files reads
//     Tracked exactly like a deliberate gitlink — and such a child is *not*
//     safe: `git add <child>` still stages its untracked files. The check
//     therefore prevents the staging accident but goes quiet once it has
//     already happened; reporting it would need a gitlink-vs-blob distinction
//     this package does not draw.
package vcs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"
)

// probeTimeout bounds how long a check may block. A hung or misbehaving git
// (or none on PATH at all) must not hang `pickle doctor` or `pickle project
// add`. A var, not a const, so a test can shrink it.
var probeTimeout = 3 * time.Second

// State is what git would do with a registered child's path today.
type State int

const (
	// Unknown is the zero value: the question could not be answered (no git
	// on PATH, the root is not a repository, or an unexpected exit code) and
	// nothing should be reported. Any failure path returns Unknown by
	// construction rather than by an extra check at each call site.
	Unknown State = iota
	// Ignored means git would refuse to stage the path (a .gitignore entry,
	// .git/info/exclude, or any other ignore mechanism).
	Ignored
	// Tracked means the path is already in the index — most likely a
	// deliberate gitlink/submodule. Reported as fine, not as a problem.
	Tracked
	// Stageable means git has no opinion at all: the path is a plain,
	// unignored, untracked directory that `git add -A`/`.`/`commit -a` would
	// pick up whole, gitlinks and all.
	Stageable
)

// Advice renders the one sentence to show an operator when s is Stageable,
// and "" for every other state. relPath is normalised first, so the entry it
// names is one git will actually honour. It is the single source of the wording so
// `pickle doctor` and `pickle project add`/`install` can never drift apart —
// each caller prepends its own context (e.g. `child "name": <advice>`).
func (s State) Advice(relPath string) string {
	if s != Stageable {
		return ""
	}
	// Normalise before rendering. A child may be registered as "./x" or "x/",
	// and an entry built from the raw string ("/./x/") is one git does not
	// honour — advice that cannot silence the very warning it accompanies,
	// which is the failure mode this package exists to avoid (T-051 review F2).
	// path (not path/filepath) is right here: config paths are slash-form and
	// so are gitignore patterns, on every platform.
	slash := path.Clean(relPath) + "/"
	return slash + " is a nested git repository that this repository does not ignore" +
		" — add \"/" + slash + "\" to .gitignore so it is never staged"
}

// ChildState answers whether the repository rooted at root would stage
// relPath (a registered child's path) as an ordinary file/directory.
//
// It runs `git -C root check-ignore -q -- relPath` first; a match (exit 0)
// is Ignored. A clean miss (exit 1) is followed by `git -C root ls-files
// --error-unmatch -- relPath`: exit 0 there means relPath is already in the
// index (Tracked — a deliberate gitlink), exit 1 means neither ignored nor
// tracked (Stageable). Any other exit code, or git being unavailable at all,
// answers Unknown — the check is silent rather than guessing.
func ChildState(root, relPath string) State {
	switch exitCode(root, "check-ignore", "-q", "--", relPath) {
	case 0:
		return Ignored
	case 1:
		// not ignored — fall through to the tracked check
	default:
		return Unknown
	}

	switch exitCode(root, "ls-files", "--error-unmatch", "--", relPath) {
	case 0:
		return Tracked
	case 1:
		return Stageable
	default:
		return Unknown
	}
}

// exitCode runs `git -C root <args...>` and returns its exit code, or -1 when
// git could not even be started (absent from PATH, timed out, etc.) — a value
// no git subcommand this package calls ever returns on its own, so callers
// can treat it as "unknown" alongside any other unexpected code.
func exitCode(root string, args ...string) int {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	// Repo-pinning variables (GIT_DIR, GIT_WORK_TREE, GIT_INDEX_FILE,
	// GIT_PREFIX) inherited from a surrounding hook or alias would otherwise
	// disagree with -C about which repository is meant — the same lesson
	// internal/hook's withoutRepoEnv exists for, mirrored rather than shared
	// (see the package doc).
	cmd.Env = withoutRepoEnv(os.Environ())
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1 // git not found, context deadline exceeded, etc.
}

// repoEnv are the variables that pin git to a specific repository, index or
// prefix — see internal/hook.repoEnv, which this deliberately mirrors.
var repoEnv = []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_PREFIX"}

// withoutRepoEnv removes repoEnv entries from an environment. They must be
// removed, not blanked: git reads GIT_DIR="" as an empty repository path and
// fails outright rather than falling back to discovery.
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
