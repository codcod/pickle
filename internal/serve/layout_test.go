// T-108: the stale-board warning. serve has no notion of a "base" branch (no
// name is ever guessed) — it checks whether the checked-out branch looks like
// a *feature* branch instead, which every registered child already names via
// its own configured branch_prefix.

package serve

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

// gitInit makes root a git repository on branch, giving it an identity so a
// later commit in a test that needs one does not depend on global git config.
func gitInit(t *testing.T, root, branch string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q", "-b", branch},
		{"config", "user.email", "serve-test@example.com"},
		{"config", "user.name", "serve test"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func gitCheckoutNew(t *testing.T, root, branch string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", root, "checkout", "-q", "-b", branch).CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b %s: %v: %s", branch, err, out)
	}
}

func inTreeCfg() *config.Config {
	return &config.Config{Projects: []config.Project{
		{Name: "demo", Path: ".", BranchPrefix: "feat/", WIPInDevelopment: 1, WIPInReview: 1},
	}}
}

func umbrellaCfg() *config.Config {
	return &config.Config{
		Layout: config.LayoutUmbrella,
		Projects: []config.Project{
			{Name: "demo", Path: "sub", BranchPrefix: "feat/", WIPInDevelopment: 1, WIPInReview: 1},
		},
	}
}

func TestStaleBoardBranchUmbrellaNeverWarns(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root, "main")
	gitCheckoutNew(t, root, "feat/T-001-x")

	if got := staleBoardBranch(root, umbrellaCfg()); got != "" {
		t.Errorf("staleBoardBranch (umbrella) = %q, want \"\" \u2014 no board in this layout can be forked by a code branch", got)
	}
}

func TestStaleBoardBranchInTreeOnBaseIsSilent(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root, "main")

	if got := staleBoardBranch(root, inTreeCfg()); got != "" {
		t.Errorf("staleBoardBranch (in-tree, on main) = %q, want \"\"", got)
	}
}

func TestStaleBoardBranchInTreeOnFeatureBranchWarns(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root, "main")
	gitCheckoutNew(t, root, "feat/T-001-x")

	if got := staleBoardBranch(root, inTreeCfg()); got != "feat/T-001-x" {
		t.Errorf("staleBoardBranch (in-tree, feature branch) = %q, want %q", got, "feat/T-001-x")
	}
}

// TestStaleBoardBranchUnbornBranchStillNames pins a real gap caught during
// implementation: right after `pickle install --in-tree`, before any commit
// exists, HEAD is an "unborn" branch — `git rev-parse --abbrev-ref HEAD` fails
// outright in that state ("ambiguous argument 'HEAD'"), which would have
// silently suppressed the warning at the exact moment it is newly true.
func TestStaleBoardBranchUnbornBranchStillNames(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root, "main") // zero commits
	gitCheckoutNew(t, root, "feat/T-001-x")

	if got := staleBoardBranch(root, inTreeCfg()); got != "feat/T-001-x" {
		t.Errorf("staleBoardBranch (unborn branch, no commits yet) = %q, want %q", got, "feat/T-001-x")
	}
}

func TestStaleBoardBranchInTreeDetachedHeadWarns(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root, "main")
	if out, err := exec.Command("git", "-C", root, "commit", "--allow-empty", "-q", "-m", "init").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "checkout", "-q", "--detach", "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach: %v: %s", err, out)
	}

	if got := staleBoardBranch(root, inTreeCfg()); got != "HEAD" {
		t.Errorf("staleBoardBranch (in-tree, detached) = %q, want %q", got, "HEAD")
	}
}

func TestStaleBoardBranchNoRepoIsSilent(t *testing.T) {
	// Not a git repository at all: vcs.Output fails, and that must degrade to
	// no warning rather than a broken page.
	if got := staleBoardBranch(t.TempDir(), inTreeCfg()); got != "" {
		t.Errorf("staleBoardBranch (no repo) = %q, want \"\"", got)
	}
}

// TestStaleBoardBannerRendersOutsideThePolledFragment mirrors
// TestBoardFilterBar: the banner must appear on the full page but never leak
// into /fragments/board, or a 5s poll would make it flicker in and out.
func TestStaleBoardBannerRendersOutsideThePolledFragment(t *testing.T) {
	requireGit(t)
	root := standardTree(t)
	gitInit(t, root, "main")
	gitCheckoutNew(t, root, "feat/T-001-x")

	h, err := Handler(Options{Root: root, Cfg: inTreeCfg()})
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	body := get(t, h, "/").Body.String()
	if !strings.Contains(body, "feat/T-001-x") || !strings.Contains(body, "in-tree layout") {
		t.Errorf("board page is missing the stale-board banner, got:\n%s", body)
	}

	frag := get(t, h, "/fragments/board").Body.String()
	if strings.Contains(frag, "stale-board") {
		t.Error("stale-board banner leaked into the polled fragment; it must live outside #board")
	}
}

// TestStaleBoardBannerAbsentOnBase pins the contrast: the same in-tree config,
// read from its own base branch, shows nothing.
func TestStaleBoardBannerAbsentOnBase(t *testing.T) {
	requireGit(t)
	root := standardTree(t)
	gitInit(t, root, "main")

	h, err := Handler(Options{Root: root, Cfg: inTreeCfg()})
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	body := get(t, h, "/").Body.String()
	if strings.Contains(body, "stale-board") {
		t.Errorf("banner shown on the base branch itself, got:\n%s", body)
	}
}
