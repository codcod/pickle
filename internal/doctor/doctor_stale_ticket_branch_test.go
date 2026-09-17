// T-128: doctor's in-tree "mirror-image hazard" check — a feature branch's
// own copy of its ticket disagreeing with the base branch's copy (status
// directory, or fewer `## History` entries).

package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

func gitConfigIdentity(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"config", "user.email", "doctor-test@example.com"},
		{"config", "user.name", "doctor test"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func gitCommitAll(t *testing.T, root, msg string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", root, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "commit", "-q", "-m", msg).CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
}

func gitCheckout(t *testing.T, root string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", root, "checkout", "-q"}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git checkout %v: %v: %s", args, err, out)
	}
}

// writeTicket (over)writes a minimal ticket file with historyLines "- "
// bullets in its ## History section, moving it to statusDir (removing any
// stale copy elsewhere first, mirroring a real `pickle ticket move`).
func writeTicket(t *testing.T, root, statusDir, id string, historyLines ...string) {
	t.Helper()
	old, _ := filepath.Glob(filepath.Join(root, "tickets", "*", id+"-*.md"))
	for _, p := range old {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(root, "tickets", statusDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("---\nid: " + id + "\ntitle: demo\nproject: demo\n---\n\n# " + id + "\n\n## History\n\n")
	for _, l := range historyLines {
		b.WriteString("- " + l + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, id+"-demo.md"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// staleTicketRepo builds one real in-tree install (installFixture) with a
// git history that reproduces T-128's three comparable outcomes at once,
// each on its own feature branch, so a scenario test just checks out its
// branch and calls checkStaleTicketBranch:
//
//   - feat/T-1-demo: branched while T-1 sat in 3-in-development; main later
//     moved it to 4-in-review — a status-directory mismatch.
//   - feat/T-2-demo: branched with T-2's History one line short; main later
//     appended a second line without moving the ticket — a History mismatch.
//   - feat/T-3-demo: branched with T-3 identical to what main still has —
//     no mismatch.
//   - feat/T-4-demo: branched right after creating T-4; T-4 never reached
//     main at all — an id that resolves on HEAD but not on base.
func staleTicketRepo(t *testing.T) (root string, cfg *config.Config) {
	t.Helper()
	requireGit(t)
	root = installFixture(t)
	gitInit(t, root) // the outer "." install only faked a bare .git/ dir
	gitConfigIdentity(t, root)
	gitCommitAll(t, root, "init")

	writeTicket(t, root, "3-in-development", "T-1", "created")
	gitCommitAll(t, root, "add T-1")
	gitCheckout(t, root, "-b", "feat/T-1-demo")
	gitCheckout(t, root, "main")
	writeTicket(t, root, "4-in-review", "T-1", "created", "in-development -> in-review")
	gitCommitAll(t, root, "move T-1 to in-review")

	writeTicket(t, root, "3-in-development", "T-2", "created")
	gitCommitAll(t, root, "add T-2")
	gitCheckout(t, root, "-b", "feat/T-2-demo")
	gitCheckout(t, root, "main")
	writeTicket(t, root, "3-in-development", "T-2", "created", "disposition: note and close")
	gitCommitAll(t, root, "append T-2 history")

	writeTicket(t, root, "3-in-development", "T-3", "created")
	gitCommitAll(t, root, "add T-3")
	gitCheckout(t, root, "-b", "feat/T-3-demo")
	gitCheckout(t, root, "main")

	gitCheckout(t, root, "-b", "feat/T-4-demo")
	writeTicket(t, root, "3-in-development", "T-4", "created")
	gitCommitAll(t, root, "add T-4 (feature branch only)")

	cfgPath := filepath.Join(root, config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return root, cfg
}

func TestCheckStaleTicketBranchStatusDirMismatchWarns(t *testing.T) {
	root, cfg := staleTicketRepo(t)
	gitCheckout(t, root, "feat/T-1-demo")

	r := &Result{}
	checkStaleTicketBranch(root, cfg, r)
	if len(r.Warnings) != 1 {
		t.Fatalf("expected exactly one warning, got: %v (passed: %v)", r.Warnings, r.Passed)
	}
	if !strings.Contains(r.Warnings[0], "T-1") || !strings.Contains(r.Warnings[0], "3-in-development") || !strings.Contains(r.Warnings[0], "4-in-review") {
		t.Fatalf("warning missing expected content: %q", r.Warnings[0])
	}
}

func TestCheckStaleTicketBranchHistoryDriftWarns(t *testing.T) {
	root, cfg := staleTicketRepo(t)
	gitCheckout(t, root, "feat/T-2-demo")

	r := &Result{}
	checkStaleTicketBranch(root, cfg, r)
	if len(r.Warnings) != 1 {
		t.Fatalf("expected exactly one warning, got: %v (passed: %v)", r.Warnings, r.Passed)
	}
	if !strings.Contains(r.Warnings[0], "T-2") || !strings.Contains(r.Warnings[0], "History") {
		t.Fatalf("warning missing expected content: %q", r.Warnings[0])
	}
}

func TestCheckStaleTicketBranchIdenticalPasses(t *testing.T) {
	root, cfg := staleTicketRepo(t)
	gitCheckout(t, root, "feat/T-3-demo")

	r := &Result{}
	checkStaleTicketBranch(root, cfg, r)
	if len(r.Warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", r.Warnings)
	}
	if len(r.Passed) != 1 || !strings.Contains(r.Passed[0], "T-3") {
		t.Fatalf("expected one passed line naming T-3, got: %v", r.Passed)
	}
}

// TestCheckStaleTicketBranchUmbrellaNeverRuns: the check must not run at all
// under the umbrella layout — verified by pointing root at a non-repo (any
// git call would error out loudly, not silently return).
func TestCheckStaleTicketBranchUmbrellaNeverRuns(t *testing.T) {
	root := t.TempDir() // not a git repository, and not even installed
	cfg := &config.Config{
		Layout:   config.LayoutUmbrella,
		Projects: []config.Project{{Name: "demo", Path: "sub", BranchPrefix: "feat/"}},
	}

	r := &Result{}
	checkStaleTicketBranch(root, cfg, r)
	if len(r.Warnings) != 0 || len(r.Errors) != 0 || len(r.Passed) != 0 {
		t.Fatalf("expected no findings at all (check must not run), got warnings=%v errors=%v passed=%v", r.Warnings, r.Errors, r.Passed)
	}
}

func TestCheckStaleTicketBranchDetachedHeadIsSilent(t *testing.T) {
	root, cfg := staleTicketRepo(t)
	gitCheckout(t, root, "feat/T-1-demo")
	gitCheckout(t, root, "--detach", "HEAD")

	r := &Result{}
	checkStaleTicketBranch(root, cfg, r)
	if len(r.Warnings) != 0 || len(r.Passed) != 0 {
		t.Fatalf("expected silence on detached HEAD, got warnings=%v passed=%v", r.Warnings, r.Passed)
	}
}

func TestCheckStaleTicketBranchNewTicketNotOnBaseIsSilent(t *testing.T) {
	root, cfg := staleTicketRepo(t)
	gitCheckout(t, root, "feat/T-4-demo")

	r := &Result{}
	checkStaleTicketBranch(root, cfg, r)
	if len(r.Warnings) != 0 || len(r.Passed) != 0 {
		t.Fatalf("expected silence for a ticket absent from base, got warnings=%v passed=%v", r.Warnings, r.Passed)
	}
}
