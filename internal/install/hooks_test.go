package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/hook"
)

// gitRepoFixture is an installed project that is also a real git repository.
func gitRepoFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_PREFIX"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if _, err := Run(os.DirFS(payloadRoot()), root, "v1", Options{ProjectName: "demo", ProjectPath: "."}); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	return root
}

func joined(list []string) string { return strings.Join(list, "\n") }

// TestUpgradeRefreshesAStaleHook: `pickle upgrade` owns pickle-written files, so
// a shim from an older binary is refreshed in place.
func TestUpgradeRefreshesAStaleHook(t *testing.T) {
	root := gitRepoFixture(t)
	if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
		t.Fatalf("hook.Install: %v", err)
	}
	path := filepath.Join(root, ".git", "hooks", "pre-commit")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(body), fmt.Sprintf("pickle:hook v%d", hook.ShimVersion), "pickle:hook v0", 1)), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Upgrade(os.DirFS(payloadRoot()), root, "v2")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !strings.Contains(joined(res.Created), "pre-commit (refreshed)") {
		t.Errorf("upgrade did not report the refreshed hook:\n%s", joined(res.Created))
	}
	if got, _ := os.ReadFile(path); string(got) != hook.Shim(hook.PreCommit) {
		t.Errorf("hook not refreshed:\n%s", got)
	}
}

// TestUpgradeRefreshesBothStaleHooks pins T-082: RefreshAll bumps every owned
// hook's shim, not just pre-commit, and leaves a foreign one alone even when
// its sibling hook is pickle-owned and stale.
func TestUpgradeRefreshesBothStaleHooks(t *testing.T) {
	root := gitRepoFixture(t)
	if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
		t.Fatalf("hook.Install(pre-commit): %v", err)
	}
	precommitPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	body, err := os.ReadFile(precommitPath)
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(body), fmt.Sprintf("pickle:hook v%d", hook.ShimVersion), "pickle:hook v0", 1)
	if err := os.WriteFile(precommitPath, []byte(stale), 0o755); err != nil {
		t.Fatal(err)
	}

	// pre-push: a foreign hook, which must survive untouched.
	prepushPath := filepath.Join(root, ".git", "hooks", "pre-push")
	foreign := "#!/bin/sh\n# lefthook\nexit 0\n"
	if err := os.WriteFile(prepushPath, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Upgrade(os.DirFS(payloadRoot()), root, "v2")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !strings.Contains(joined(res.Created), "pre-commit (refreshed)") {
		t.Errorf("upgrade did not report the refreshed pre-commit hook:\n%s", joined(res.Created))
	}
	if strings.Contains(joined(res.Created), "pre-push") {
		t.Errorf("upgrade touched the foreign pre-push hook:\n%s", joined(res.Created))
	}
	if got, _ := os.ReadFile(precommitPath); string(got) != hook.Shim(hook.PreCommit) {
		t.Errorf("pre-commit hook not refreshed:\n%s", got)
	}
	if got, _ := os.ReadFile(prepushPath); string(got) != foreign {
		t.Errorf("foreign pre-push hook was modified:\n%s", got)
	}
}

// TestUpgradeNeverArmsAnAbsentHook: hooks are opt-in, and an upgrade that
// silently started rejecting commits or pushes would be a surprise nobody
// asked for. Covers both hooks, not just pre-commit.
func TestUpgradeNeverArmsAnAbsentHook(t *testing.T) {
	root := gitRepoFixture(t)
	if _, err := Upgrade(os.DirFS(payloadRoot()), root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	for _, name := range hook.Names() {
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", string(name))); !os.IsNotExist(err) {
			t.Errorf("upgrade installed %s, which was not there: %v", name, err)
		}
	}
}

// TestUpgradeWithoutAGitRepo pins T-057 finding B4: the existing fixtures in
// this package are not git repositories, and hook refresh must be a silent
// no-op there rather than failing every upgrade.
func TestUpgradeWithoutAGitRepo(t *testing.T) {
	root := t.TempDir()
	if _, err := Run(os.DirFS(payloadRoot()), root, "v1", Options{ProjectName: "demo", ProjectPath: "."}); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	if _, err := Upgrade(os.DirFS(payloadRoot()), root, "v2"); err != nil {
		t.Fatalf("Upgrade outside a git repo: %v", err)
	}
}

func TestUninstallRemovesAnOwnedHook(t *testing.T) {
	root := gitRepoFixture(t)
	for _, name := range hook.Names() {
		if _, err := hook.Install(root, name, false); err != nil {
			t.Fatalf("hook.Install(%s): %v", name, err)
		}
	}

	// Dry run lists both hooks and changes nothing.
	res, err := Uninstall(os.DirFS(payloadRoot()), root, UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Uninstall (dry-run): %v", err)
	}
	for _, name := range hook.Names() {
		if !strings.Contains(joined(res.Removed), string(name)+" (dry-run)") {
			t.Errorf("dry-run did not list %s:\n%s", name, joined(res.Removed))
		}
		path := filepath.Join(root, ".git", "hooks", string(name))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dry-run removed %s: %v", name, err)
		}
	}

	res, err = Uninstall(os.DirFS(payloadRoot()), root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	for _, name := range hook.Names() {
		if !strings.Contains(joined(res.Removed), string(name)) {
			t.Errorf("uninstall did not report %s:\n%s", name, joined(res.Removed))
		}
		path := filepath.Join(root, ".git", "hooks", string(name))
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s survived uninstall: %v", name, err)
		}
	}
}

func TestUninstallLeavesAForeignHook(t *testing.T) {
	root := gitRepoFixture(t)
	dir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "pre-commit")
	foreign := "#!/bin/sh\n# lefthook\nexit 0\n"
	if err := os.WriteFile(path, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Uninstall(os.DirFS(payloadRoot()), root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if strings.Contains(joined(res.Removed), "pre-commit") {
		t.Errorf("uninstall removed a hook pickle does not own:\n%s", joined(res.Removed))
	}
	if !strings.Contains(joined(res.Skipped), "not pickle's") {
		t.Errorf("uninstall did not report the foreign hook as skipped:\n%s", joined(res.Skipped))
	}
	if got, _ := os.ReadFile(path); string(got) != foreign {
		t.Errorf("foreign hook was modified:\n%s", got)
	}
}

// TestMarkerBlockStatesWhereCommitsLand: the hook enforces a rule, and an
// enforced rule that is written down nowhere is not a rule. The marker block is
// what every agent session reads, so the statement has to be there — but only
// where the rule actually applies (T-109). Under the in-tree layout the board
// shares a repository with the code and the obligation is load-bearing; under
// the umbrella layout no feature branch can fork the board, so stating the
// obligation would hand the reader a rule whose justification is absent from
// their setup and send them hunting for a guard that never fires. Both
// directions are asserted, since a one-sided test would pass on a block that
// states the rule unconditionally — exactly the defect this replaced.
func TestMarkerBlockStatesWhereCommitsLand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		opts     Options
		want     []string
		unwanted []string
	}{
		{
			name: "in-tree states the obligation and its enforcement",
			opts: Options{ProjectName: "demo", ProjectPath: ".", InTree: true},
			want: []string{
				"Where commits land",
				"bookkeeping is committed on the base branch",
				"pickle hooks install",
			},
		},
		{
			name: "umbrella explains why there is nothing to guard",
			opts: Options{ProjectName: "demo", ProjectPath: "child"},
			want: []string{
				"Where commits land",
				"no feature branch can",
			},
			unwanted: []string{
				"bookkeeping is committed on the base branch",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.opts.ProjectPath != "" && tc.opts.ProjectPath != "." {
				if err := os.MkdirAll(filepath.Join(root, tc.opts.ProjectPath), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			res, err := Run(os.DirFS(payloadRoot()), root, "v1", tc.opts)
			if err != nil {
				t.Fatalf("install.Run: %v (%v)", err, res)
			}
			body, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(body), want) {
					t.Errorf("AGENTS.md marker block does not mention %q", want)
				}
			}
			for _, unwanted := range tc.unwanted {
				if strings.Contains(string(body), unwanted) {
					t.Errorf("AGENTS.md marker block states %q, which does not apply in this layout", unwanted)
				}
			}
		})
	}
}
