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
	if _, err := hook.Install(root, false); err != nil {
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
	if got, _ := os.ReadFile(path); string(got) != hook.Shim() {
		t.Errorf("hook not refreshed:\n%s", got)
	}
}

// TestUpgradeNeverArmsAnAbsentHook: the guard is opt-in, and an upgrade that
// silently started rejecting commits would be a surprise nobody asked for.
func TestUpgradeNeverArmsAnAbsentHook(t *testing.T) {
	root := gitRepoFixture(t)
	if _, err := Upgrade(os.DirFS(payloadRoot()), root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("upgrade installed a hook that was not there: %v", err)
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
	if _, err := hook.Install(root, false); err != nil {
		t.Fatalf("hook.Install: %v", err)
	}
	path := filepath.Join(root, ".git", "hooks", "pre-commit")

	// Dry run lists it and changes nothing.
	res, err := Uninstall(os.DirFS(payloadRoot()), root, UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Uninstall (dry-run): %v", err)
	}
	if !strings.Contains(joined(res.Removed), "pre-commit (dry-run)") {
		t.Errorf("dry-run did not list the hook:\n%s", joined(res.Removed))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dry-run removed the hook: %v", err)
	}

	res, err = Uninstall(os.DirFS(payloadRoot()), root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !strings.Contains(joined(res.Removed), "pre-commit") {
		t.Errorf("uninstall did not report the hook:\n%s", joined(res.Removed))
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("hook survived uninstall: %v", err)
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
// what every agent session reads, so the statement has to be there.
func TestMarkerBlockStatesWhereCommitsLand(t *testing.T) {
	root := t.TempDir()
	res, err := Run(os.DirFS(payloadRoot()), root, "v1", Options{ProjectName: "demo", ProjectPath: "."})
	if err != nil {
		t.Fatalf("install.Run: %v (%v)", err, res)
	}
	body, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Where commits land",
		"bookkeeping is committed on the base branch",
		"pickle hooks install",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("AGENTS.md marker block does not mention %q", want)
		}
	}
}
