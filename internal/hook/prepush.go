package hook

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/codcod/pickle/internal/config"
)

// This file is the second of only two rules internal/hook enforces, and the
// second to depend on gitHere rather than gitAt (T-057 decision 11, restated
// here because it now has a second reader to convince): git invokes pre-push
// from the top of the worktree with GIT_* variables set for that push, and
// `git -C <root>` could silently inspect a different repository than the one
// actually being pushed.

// PushRef is one line of the pre-push hook's stdin: <local ref> <local sha1>
// <remote ref> <remote sha1>, one line per ref being pushed.
type PushRef struct {
	LocalRef  string
	LocalSHA  string
	RemoteRef string
	RemoteSHA string
}

// ParsePushRefs reads the pre-push hook's stdin format. It is pure — it takes
// an io.Reader, not stdin itself — so the format is testable with a
// strings.Reader and no git at all. A malformed line (not exactly four
// whitespace-separated fields) is skipped, not fatal: the fail-open contract
// (decision 2) applies to parsing too, and one unreadable line must not sink
// every ref in the push.
func ParsePushRefs(r io.Reader) ([]PushRef, error) {
	var refs []PushRef
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 4 {
			continue
		}
		refs = append(refs, PushRef{
			LocalRef:  fields[0],
			LocalSHA:  fields[1],
			RemoteRef: fields[2],
			RemoteSHA: fields[3],
		})
	}
	if err := sc.Err(); err != nil {
		return refs, err
	}
	return refs, nil
}

// isZeroSHA reports whether s is git's all-zero sha1 (or sha256) — the marker
// for "this ref does not exist on this side", used for both a first push of a
// new branch (RemoteSHA) and a branch deletion (LocalSHA).
func isZeroSHA(s string) bool {
	if s == "" {
		return true
	}
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}

// refsHeadsPrefix is the prefix every local branch ref carries on stdin. A
// tag push or anything else outside refs/heads/ is not a branch and is never
// this guard's concern.
const refsHeadsPrefix = "refs/heads/"

// branchBeingPushed names the branch a PushRef pushes, or reports ok=false
// when the ref is not a branch push at all (a tag, say).
//
// LocalRef alone is not enough: `git push <remote> HEAD:refs/heads/feat/x`
// sends the literal string "HEAD" as LocalRef (finding F2, T-082's first
// review) — the ref being pushed "from" has no ref name of its own, only a
// commit. RemoteRef is always the fully-qualified destination ref, for every
// shape of push, so it is the reliable source: this falls back to it whenever
// LocalRef does not resolve, and only then. A tag push's RemoteRef is
// `refs/tags/...`, so it still falls outside refsHeadsPrefix and is still
// skipped.
func branchBeingPushed(ref PushRef) (string, bool) {
	if branch, ok := strings.CutPrefix(ref.LocalRef, refsHeadsPrefix); ok {
		return branch, true
	}
	return strings.CutPrefix(ref.RemoteRef, refsHeadsPrefix)
}

// CheckPrePush is the rule: refuse a push whose local ref is a feature branch
// of a registered child when the range against the remote base still carries
// a tickets/ path. It reports ok=false only for a real violation; every other
// outcome — a deletion, the base branch itself, tickets/ outside this
// repository, an unresolvable base, a git diff that errors — is ok=true,
// because a guard that cannot decide must not block (decision 2, the same
// fail-open contract CheckPreCommit already carries).
//
// The range is *not* the stdin range. On a new branch RemoteSHA is all-zero —
// there is no remote-side range at all — and on a later push the stdin range
// is last-pushed…local, which is not what a forge diffs and would wave
// through tickets/ paths that rode in on an earlier push. The invariant is
// that the MR carries no tickets/ path, so this measures
// <remote>/<base>...<local-sha> — the same three-dot, merge-base form the
// prose check in tickets-README.md uses and a forge uses to compute an MR
// diff (decision 4). remote is the push's target remote, git's own $1.
func CheckPrePush(cfg *config.Config, remote string, refs []PushRef, w io.Writer) (bool, error) {
	prefix, ok := ticketsPrefix(cfg)
	if !ok {
		return true, nil // tickets/ is outside this repository — nothing to guard
	}

	var base string
	baseTried := false
	var baseErr error

	for _, ref := range refs {
		if isZeroSHA(ref.LocalSHA) {
			continue // branch deletion — nothing pushed to inspect
		}
		branch, ok := branchBeingPushed(ref)
		if !ok {
			continue // not a branch (e.g. a tag) — never this guard's concern
		}
		if !onFeatureBranch(cfg, branch) {
			continue // the base branch itself carries tickets/ paths by design
		}

		if !baseTried {
			base, baseErr = resolveBase(remote)
			baseTried = true
			if baseErr != nil {
				fmt.Fprintf(w, "pickle: bookkeeping guard skipped (%v)\n", baseErr)
			}
		}
		if baseErr != nil {
			continue
		}

		out, err := gitHere("diff", "--name-only", "-z", base+"..."+ref.LocalSHA)
		if err != nil {
			continue // undecidable — fail open (decision 2)
		}
		var offenders []string
		for _, p := range strings.Split(out, "\x00") {
			if p == "" {
				continue
			}
			if p == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(p, prefix) {
				offenders = append(offenders, p)
			}
		}
		if len(offenders) == 0 {
			continue
		}
		writePushRejection(w, remote, branch, base, prefix, offenders)
		return false, nil
	}
	return true, nil
}

// resolveBase finds the remote-tracking ref that stands for remote's base
// branch, trying each candidate in turn and stopping at the first that
// resolves (decision 4):
//
//  1. refs/remotes/<remote>/HEAD, dereferenced — the remote's own answer to
//     "what is the default branch", when the remote-tracking HEAD exists.
//  2. refs/remotes/<remote>/main
//  3. refs/remotes/<remote>/master
//
// It performs no network I/O (decision 5): every candidate is checked against
// the remote-tracking refs already on disk, so a stale one can only ever
// widen the diffed range, never narrow it — the failure direction is a false
// *refusal*, never a false pass.
func resolveBase(remote string) (string, error) {
	var tried []string

	if out, err := gitHere("symbolic-ref", "--quiet", "--short", "refs/remotes/"+remote+"/HEAD"); err == nil && out != "" {
		ref := "refs/remotes/" + out
		if _, verr := gitHere("rev-parse", "--verify", "--quiet", ref); verr == nil {
			return ref, nil
		}
	}
	tried = append(tried, "refs/remotes/"+remote+"/HEAD")

	for _, name := range []string{"main", "master"} {
		ref := "refs/remotes/" + remote + "/" + name
		if _, err := gitHere("rev-parse", "--verify", "--quiet", ref); err == nil {
			return ref, nil
		}
		tried = append(tried, ref)
	}
	return "", fmt.Errorf("could not resolve a base branch for remote %q (tried %s)", remote, strings.Join(tried, ", "))
}

// writePushRejection explains the refusal: what was pushed, why it matters,
// and the three ways out — push the base first (the one that actually fixes a
// real violation), fetch when the remote-tracking ref is stale (decision 5),
// and --no-verify when the branch's own product genuinely lives under
// tickets/. Shape mirrors writeRejection's.
func writePushRejection(w io.Writer, remote, branch, base, prefix string, offenders []string) {
	fmt.Fprintf(w, "pickle: refusing to push ticket bookkeeping on a feature branch.\n\n")
	fmt.Fprintf(w, "  branch:  %s\n", branch)
	fmt.Fprintf(w, "  range:   %s...%s\n", base, branch)
	shown := offenders
	if len(shown) > maxListedPaths {
		shown = shown[:maxListedPaths]
	}
	for i, p := range shown {
		label := "  carries: "
		if i > 0 {
			label = "           "
		}
		fmt.Fprintf(w, "%s%s\n", label, p)
	}
	if n := len(offenders) - len(shown); n > 0 {
		fmt.Fprintf(w, "           … and %d more\n", n)
	}
	fmt.Fprintf(w, "\nTicket and board bookkeeping belongs on the base branch. A merge request built\n"+
		"from this range would carry these files too, and the board would then disagree\n"+
		"with the tickets it indexes.\n\n")
	base3 := strings.TrimPrefix(base, "refs/remotes/")
	remedy := [][2]string{
		{"git push " + remote + " " + strings.TrimPrefix(base3, remote+"/"), "push the bookkeeping on the base branch first"},
		{"git fetch " + remote + " " + strings.TrimPrefix(base3, remote+"/"), "if it's already pushed and this ref is just stale"},
		{"git push --no-verify", "when this branch's own product is a file under " + prefix},
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
}
