package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/hook"
)

// gitFixture upgrades installFixture's fake .git directory into a real
// repository, so the hook checks have something to inspect. The default fixture
// deliberately stays fake: that is the "not a git repository" path, and it is
// the shape every other test in this package runs in.
func gitFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := installFixture(t)
	rmAll(t, filepath.Join(root, ".git"))

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
	if out, err := exec.Command("git", "-C", root, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root
}

func hasPassedContaining(passed []string, sub string) bool {
	for _, p := range passed {
		if strings.Contains(p, sub) {
			return true
		}
	}
	return false
}

func hasWarnContaining(warns []string, sub string) bool {
	for _, w := range warns {
		if strings.Contains(w, sub) {
			return true
		}
	}
	return false
}

// TestCheckHooksAbsentIsNotAFinding: the guard is opt-in, so a project without
// it is healthy. Reporting it as a warning would train users to ignore doctor.
func TestCheckHooksAbsentIsNotAFinding(t *testing.T) {
	root := gitFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 || len(res.Warnings) != 0 {
		t.Fatalf("absent hook produced findings: errors=%v warnings=%v", res.Errors, res.Warnings)
	}
	if !hasPassedContaining(res.Passed, "pre-commit guard not installed") {
		t.Errorf("doctor is silent about the hook; it should say how to arm it: %v", res.Passed)
	}
	if !hasPassedContaining(res.Passed, "pickle hooks install") {
		t.Errorf("the absent-hook line does not name the command that fixes it: %v", res.Passed)
	}
}

func TestCheckHooksOwnedAndStale(t *testing.T) {
	root := gitFixture(t)
	if _, err := hook.Install(root, false); err != nil {
		t.Fatalf("hook.Install: %v", err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Warnings) != 0 {
		t.Errorf("a current hook warned: %v", res.Warnings)
	}
	if !hasPassedContaining(res.Passed, "pre-commit guard installed and current") {
		t.Errorf("current hook not reported: %v", res.Passed)
	}

	// A shim written by an older pickle: the one hook state worth a warning,
	// because `pickle upgrade` fixes it.
	path := filepath.Join(root, ".git", "hooks", "pre-commit")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(body), "pickle:hook v1", "pickle:hook v0", 1)
	if err := os.WriteFile(path, []byte(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	res = Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasWarnContaining(res.Warnings, "older pickle") {
		t.Errorf("stale shim did not warn: %v", res.Warnings)
	}
	if !hasWarnContaining(res.Warnings, "pickle upgrade") {
		t.Errorf("stale-shim warning does not name the fix: %v", res.Warnings)
	}
}

func TestCheckHooksForeignIsLeftAlone(t *testing.T) {
	root := gitFixture(t)
	dir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pre-commit"), []byte("#!/bin/sh\n# husky\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Warnings) != 0 || len(res.Errors) != 0 {
		t.Errorf("a foreign hook produced findings: %v %v", res.Errors, res.Warnings)
	}
	if !hasPassedContaining(res.Passed, "not pickle's") {
		t.Errorf("foreign hook not reported: %v", res.Passed)
	}
}

// TestCheckHooksWithoutAGitRepo is the shape of every other fixture in this
// package (an empty .git directory) and of any non-git install: it must be a
// silent pass, not a second complaint about a repo `checkChildren` already
// reports on.
func TestCheckHooksWithoutAGitRepo(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Warnings) != 0 || len(res.Errors) != 0 {
		t.Fatalf("non-git install produced findings: %v %v", res.Errors, res.Warnings)
	}
	if !hasPassedContaining(res.Passed, "not applicable") {
		t.Errorf("expected a 'not applicable' hook line: %v", res.Passed)
	}
}
