package doctor

import (
	"fmt"
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

// pinPATH sets PATH deterministically for a probe test. `git` must still
// resolve (the fixture's `git init` and internal/hook's gitAt/gitHere calls
// need it), but which `pickle` (if any) internal/hook.Probe() finds must not
// depend on whatever happens to be installed on the developer's machine or CI
// runner (T-068) — without this, TestCheckHooksOwnedAndStale would be flaky
// wherever a real `pickle` sits on PATH already. git is symlinked into a
// **fresh, otherwise-empty** directory rather than trusting its own directory
// wholesale: on this very machine `git` and `pickle` are both installed into
// the same Homebrew bin dir, and pointing PATH at that whole directory would
// leak the real `pickle` straight back in. binDirs, when given, are searched
// before that directory, so a stub `pickle` placed in one of them wins the
// lookup; with no binDirs, `pickle` is deterministically absent.
func pinPATH(t *testing.T, binDirs ...string) {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	gitOnly := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(gitOnly, "git")); err != nil {
		t.Fatal(err)
	}
	dirs := append(append([]string{}, binDirs...), gitOnly)
	t.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

// stubPickle writes a fake `pickle` that answers `hooks run pre-commit` with
// rc and `version` with a fixed string, and returns the directory it lives
// in (ready to pass to pinPATH). It is a stand-in for a real binary skewed
// against the shim: capable (rc 0) or not (any other code, mirroring an
// older pickle's exit 2 on the then-unknown `hooks` verb).
func stubPickle(t *testing.T, rc int) string {
	t.Helper()
	dir := t.TempDir()
	body := fmt.Sprintf("#!/bin/sh\ncase \"$1\" in\n  hooks) exit %d ;;\n  version) echo \"pickle 0.2.2\" ;;\n  *) exit %d ;;\nesac\n", rc, rc)
	path := filepath.Join(dir, "pickle")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
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
	if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
		t.Fatalf("hook.Install: %v", err)
	}

	// A current shim also probes the PATH pickle (T-068): pin PATH to a stub
	// that can run the guard, or this assertion would depend on whatever
	// happens to be installed on the machine running the test.
	pinPATH(t, stubPickle(t, 0))
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Warnings) != 0 {
		t.Errorf("a current hook warned: %v", res.Warnings)
	}
	if !hasPassedContaining(res.Passed, "pre-commit guard installed and current") {
		t.Errorf("current hook not reported: %v", res.Passed)
	}

	// A shim written by an older pickle: the one hook state worth a warning,
	// because `pickle upgrade` fixes it. The stale branch never probes PATH (it
	// already has a warning to give), so PATH is irrelevant here.
	path := filepath.Join(root, ".git", "hooks", "pre-commit")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(body), fmt.Sprintf("pickle:hook v%d", hook.ShimVersion), "pickle:hook v0", 1)
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

// TestCheckSelfHostLinkStillReportsIncapablePATHPickle guards D6 (T-046): the
// self-host version-check skip must not become a blanket self-host mute. The
// hook checks are about the user's PATH and about hand-edits, not about how
// the skill dir is arranged, so a self-hosting checkout with an incapable
// PATH pickle must still warn exactly as an ordinary install would.
func TestCheckSelfHostLinkStillReportsIncapablePATHPickle(t *testing.T) {
	root := gitFixture(t)
	linkSkill(t, root)
	if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
		t.Fatalf("hook.Install: %v", err)
	}
	bin := stubPickle(t, 2) // mirrors an older pickle's exit 2 on the (then) unknown `hooks` verb
	pinPATH(t, bin)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasWarnContaining(res.Warnings, "inert") {
		t.Errorf("self-host checkout with an incapable PATH pickle did not warn: %v", res.Warnings)
	}
	if !hasWarnContaining(res.Warnings, filepath.Join(bin, "pickle")) {
		t.Errorf("warning does not name the incapable binary's path: %v", res.Warnings)
	}
}

// TestCheckHooksProbesPATH pins T-068: an owned, current shim only reports
// itself healthy when the `pickle` the shim will actually resolve from PATH
// can run the guard, and the probe is skipped entirely for a hook that is not
// armed (absent) — it must not pay the exec cost, or produce a finding, for a
// guard nobody installed.
func TestCheckHooksProbesPATH(t *testing.T) {
	t.Run("incapable pickle on PATH warns and is inert", func(t *testing.T) {
		root := gitFixture(t)
		if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
			t.Fatalf("hook.Install: %v", err)
		}
		bin := stubPickle(t, 2) // mirrors an older pickle's exit 2 on the (then) unknown `hooks` verb
		pinPATH(t, bin)
		res := Check(root, "test-ver", os.DirFS(payloadRoot()))
		if !hasWarnContaining(res.Warnings, "inert") {
			t.Errorf("incapable PATH pickle did not warn: %v", res.Warnings)
		}
		if !hasWarnContaining(res.Warnings, filepath.Join(bin, "pickle")) {
			t.Errorf("warning does not name the incapable binary's path: %v", res.Warnings)
		}
		if len(res.Errors) != 0 {
			t.Errorf("inert guard must warn, not error: %v", res.Errors)
		}
	})

	t.Run("guard ran and found a violation still passes", func(t *testing.T) {
		// T-071 R3: exit 1 proves the guard ran and found a violation — that is
		// proof of capability, not incapability, per the shim's own contract
		// (docs/user-manual/cli-reference.adoc's "Both fail open, always"
		// bullet: 1 only for a real violation, never for "can't run").
		root := gitFixture(t)
		if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
			t.Fatalf("hook.Install: %v", err)
		}
		pinPATH(t, stubPickle(t, 1))
		res := Check(root, "test-ver", os.DirFS(payloadRoot()))
		if len(res.Warnings) != 0 {
			t.Errorf("a capable (exit 1) PATH pickle warned: %v", res.Warnings)
		}
		if len(res.Errors) != 0 {
			t.Errorf("a capable (exit 1) PATH pickle errored: %v", res.Errors)
		}
		if !hasPassedContaining(res.Passed, "can run the installed guards") {
			t.Errorf("capable (exit 1) PATH pickle not reported as such: %v", res.Passed)
		}
	})

	t.Run("capable pickle on PATH passes", func(t *testing.T) {
		root := gitFixture(t)
		if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
			t.Fatalf("hook.Install: %v", err)
		}
		pinPATH(t, stubPickle(t, 0))
		res := Check(root, "test-ver", os.DirFS(payloadRoot()))
		if len(res.Warnings) != 0 {
			t.Errorf("a capable PATH pickle warned: %v", res.Warnings)
		}
		if !hasPassedContaining(res.Passed, "can run the installed guards") {
			t.Errorf("capable PATH pickle not reported as such: %v", res.Passed)
		}
	})

	t.Run("no pickle at all on PATH warns", func(t *testing.T) {
		root := gitFixture(t)
		if _, err := hook.Install(root, hook.PreCommit, false); err != nil {
			t.Fatalf("hook.Install: %v", err)
		}
		pinPATH(t) // git only — no `pickle` anywhere on PATH
		res := Check(root, "test-ver", os.DirFS(payloadRoot()))
		if !hasWarnContaining(res.Warnings, "no pickle is on PATH") {
			t.Errorf("absent PATH pickle did not warn: %v", res.Warnings)
		}
	})

	t.Run("absent hook never probes", func(t *testing.T) {
		root := gitFixture(t) // no hook.Install: the guard is not armed
		pinPATH(t, stubPickle(t, 2))
		res := Check(root, "test-ver", os.DirFS(payloadRoot()))
		if len(res.Warnings) != 0 || len(res.Errors) != 0 {
			t.Errorf("an unarmed guard produced findings from an unrelated incapable PATH pickle: warnings=%v errors=%v", res.Warnings, res.Errors)
		}
	})
}

// TestCheckHooksReportsBothHooks pins T-082: pre-commit and pre-push are both
// installed, both reported, and the PATH-capability probe — a per-binary
// question, not a per-hook one (decision 6) — fires at most once, not once
// per owned hook.
func TestCheckHooksReportsBothHooks(t *testing.T) {
	root := gitFixture(t)
	for _, name := range hook.Names() {
		if _, err := hook.Install(root, name, false); err != nil {
			t.Fatalf("hook.Install(%s): %v", name, err)
		}
	}
	pinPATH(t, stubPickle(t, 0))
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Warnings) != 0 {
		t.Errorf("both hooks current and a capable PATH pickle still warned: %v", res.Warnings)
	}
	for _, name := range hook.Names() {
		if !hasPassedContaining(res.Passed, string(name)+" guard installed and current") {
			t.Errorf("%s not reported as installed and current: %v", name, res.Passed)
		}
	}
	n := 0
	for _, p := range res.Passed {
		if strings.Contains(p, "can run the installed guards") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("PATH-capability line appeared %d time(s), want exactly 1 (probed once per binary, not per hook): %v", n, res.Passed)
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
