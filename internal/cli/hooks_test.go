package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// pinPathWithPickle sets PATH deterministically for a PATH-probe test
// (T-068): `git` must still resolve (hook.Install/Status need it), but which
// `pickle` the CLI's own probe finds must not depend on whatever happens to
// be installed on the developer's machine or CI runner. `git` is symlinked
// into a fresh, otherwise-empty directory rather than trusting its own
// directory wholesale — a real `pickle` commonly lives right next to `git`
// (e.g. the same Homebrew bin dir), which would leak straight back in.
// binDir, when non-empty, is searched first, so a stub `pickle` placed there
// wins the lookup; "" leaves `pickle` deterministically absent.
func pinPathWithPickle(t *testing.T, binDir string) {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	gitOnly := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(gitOnly, "git")); err != nil {
		t.Fatal(err)
	}
	dirs := []string{gitOnly}
	if binDir != "" {
		dirs = append([]string{binDir}, dirs...)
	}
	t.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

// stubPickleBin writes a fake `pickle` that answers `hooks run pre-commit`
// with rc and `version` with a fixed string, and returns the directory it
// lives in (ready to pass to pinPathWithPickle) — a stand-in for a binary
// that is (rc 0) or is not (any other code, mirroring an older pickle's exit
// 2 on the then-unknown `hooks` verb) able to run the guard.
func stubPickleBin(t *testing.T, rc int) string {
	t.Helper()
	dir := t.TempDir()
	body := fmt.Sprintf("#!/bin/sh\ncase \"$1\" in\n  hooks) exit %d ;;\n  version) echo \"pickle 0.2.2\" ;;\n  *) exit %d ;;\nesac\n", rc, rc)
	if err := os.WriteFile(filepath.Join(dir, "pickle"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// stubPickleBinCounting is stubPickleBin's twin for counting execs: every
// invocation appends one line to countFile before answering exit 0, so a
// test can assert Probe() ran exactly once rather than once per hook (T-082
// finding F3).
func stubPickleBinCounting(t *testing.T, countFile string) string {
	t.Helper()
	dir := t.TempDir()
	body := fmt.Sprintf("#!/bin/sh\necho x >> %q\ncase \"$1\" in\n  hooks) exit 0 ;;\n  version) echo \"pickle 0.2.2\" ;;\n  *) exit 0 ;;\nesac\n", countFile)
	if err := os.WriteFile(filepath.Join(dir, "pickle"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// gitOrSkip skips a test that needs a real repository when git is unavailable.
func gitOrSkip(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

// gitInit turns dir into a repository on the given branch, isolated from the
// developer's git config (a global core.hooksPath would otherwise send the shim
// outside the temp tree).
func gitInit(t *testing.T, dir, branch string) {
	t.Helper()
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
	for _, args := range [][]string{
		{"init", "-q", "-b", branch},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
}

func gitAdd(t *testing.T, dir string, paths ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "add"}, paths...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
}

// gitInitWithRemote upgrades gitInit's repository into a clone of a fresh
// bare "remote", with an initial commit already pushed to baseBranch — the
// shape `hooks run pre-push`'s base resolution (decision 4) needs a real
// origin/<base> to measure against.
func gitInitWithRemote(t *testing.T, dir, baseBranch string) {
	t.Helper()
	gitInit(t, dir, baseBranch)
	remote := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", baseBranch, remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	for _, args := range [][]string{
		{"-C", dir, "remote", "add", "origin", remote},
		{"-C", dir, "add", "-A"},
		{"-C", dir, "commit", "-qm", "seed"},
		{"-C", dir, "push", "-q", "-u", "origin", baseBranch},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

// withStdin feeds content to the process's stdin for the duration of the
// test — `hooks run pre-push` reads its ref list from os.Stdin directly,
// exactly as the installed shim's git invocation does.
func withStdin(t *testing.T, content string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })
}

// pushRefLine renders one `hooks run pre-push` stdin line for a normal
// (non-deleting, first-push-of-a-new-remote-ref) push of branch at head.
func pushRefLine(branch, head string) string {
	zero := strings.Repeat("0", 40)
	return "refs/heads/" + branch + " " + head + " refs/heads/" + branch + " " + zero + "\n"
}

// TestHooksRunExitCodes pins the contract the installed shim depends on
// (T-057 decision 3, finding B3): exit 1 means "violation" and nothing else
// does. If `hooks run` ever returns 1 for a degraded guard, or 2 for a
// violation, the shim either stops blocking real violations or blocks every
// commit in the repository.
func TestHooksRunExitCodes(t *testing.T) {
	gitOrSkip(t)

	t.Run("violation exits 1", func(t *testing.T) {
		root := newProject(t)
		gitInit(t, root, "feat/T-001-x")
		gitAdd(t, root, "tickets")
		if got := Run(nil, "test", []string{"hooks", "run", "pre-commit"}); got != exitError {
			t.Fatalf("hooks run = %d, want %d (violation)", got, exitError)
		}
	})

	t.Run("base branch exits 0", func(t *testing.T) {
		root := newProject(t)
		gitInit(t, root, "main")
		gitAdd(t, root, "tickets")
		if got := Run(nil, "test", []string{"hooks", "run", "pre-commit"}); got != exitOK {
			t.Fatalf("hooks run = %d, want %d", got, exitOK)
		}
	})

	t.Run("not a git repository exits 0", func(t *testing.T) {
		newProject(t) // no git init: the guard cannot decide
		if got := Run(nil, "test", []string{"hooks", "run", "pre-commit"}); got != exitOK {
			t.Fatalf("hooks run = %d, want %d (degraded, must not block)", got, exitOK)
		}
	})

	t.Run("no pickle.toml exits 0", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if got := Run(nil, "test", []string{"hooks", "run", "pre-commit"}); got != exitOK {
			t.Fatalf("hooks run = %d, want %d (not a pickle project)", got, exitOK)
		}
	})

	t.Run("unparseable pickle.toml exits 0", func(t *testing.T) {
		root := newProject(t)
		gitInit(t, root, "feat/T-001-x")
		if err := os.WriteFile(filepath.Join(root, "pickle.toml"), []byte("this is not toml\x00"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := Run(nil, "test", []string{"hooks", "run", "pre-commit"}); got != exitOK {
			t.Fatalf("hooks run = %d, want %d (broken config must not block commits)", got, exitOK)
		}
	})
}

// TestHooksUsageErrors: a bad invocation is exitUsage (2), which the shim reads
// as "not a violation" — the same code an older pickle returns for the unknown
// `hooks` verb.
func TestHooksUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no subcommand", []string{"hooks"}},
		{"unknown subcommand", []string{"hooks", "frobnicate"}},
		{"run without a hook name", []string{"hooks", "run"}},
		{"run with an unknown hook", []string{"hooks", "run", "post-merge"}},
		{"run pre-commit with a stray argument", []string{"hooks", "run", "pre-commit", "extra"}},
		{"run pre-push with too many arguments", []string{"hooks", "run", "pre-push", "origin", "url", "extra"}},
		{"install bad flag", []string{"hooks", "install", "--bogus"}},
		{"status bad flag", []string{"hooks", "status", "--bogus"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Run(nil, "test", tc.args); got != exitUsage {
				t.Fatalf("Run(%v) = %d, want %d", tc.args, got, exitUsage)
			}
		})
	}
}

// TestHooksRunPrePushExitCodes mirrors TestHooksRunExitCodes for the second
// hook (T-082): exit 1 only for a real violation, 0 for every degraded or
// allowed path.
func TestHooksRunPrePushExitCodes(t *testing.T) {
	gitOrSkip(t)

	t.Run("feature branch carrying bookkeeping is refused", func(t *testing.T) {
		root := newProject(t)
		gitInitWithRemote(t, root, "main")
		if out, err := exec.Command("git", "-C", root, "checkout", "-qb", "feat/T-999-demo").CombinedOutput(); err != nil {
			t.Fatalf("git checkout: %v: %s", err, out)
		}
		if err := os.WriteFile(filepath.Join(root, "tickets", "1-to-do", "T-999-demo.md"), []byte("# T-999\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitAdd(t, root, "tickets")
		if out, err := exec.Command("git", "-C", root, "commit", "-qm", "board: T-999 demo").CombinedOutput(); err != nil {
			t.Fatalf("git commit: %v: %s", err, out)
		}
		head := strings.TrimSpace(mustRun(t, "git", "-C", root, "rev-parse", "HEAD"))

		withStdin(t, pushRefLine("feat/T-999-demo", head))
		if got := Run(nil, "test", []string{"hooks", "run", "pre-push", "origin"}); got != exitError {
			t.Fatalf("hooks run pre-push = %d, want %d (violation)", got, exitError)
		}
	})

	t.Run("the base branch carrying bookkeeping is allowed", func(t *testing.T) {
		root := newProject(t)
		gitInitWithRemote(t, root, "main")
		if err := os.WriteFile(filepath.Join(root, "tickets", "1-to-do", "T-998-demo.md"), []byte("# T-998\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitAdd(t, root, "tickets")
		if out, err := exec.Command("git", "-C", root, "commit", "-qm", "board: T-998 demo").CombinedOutput(); err != nil {
			t.Fatalf("git commit: %v: %s", err, out)
		}
		head := strings.TrimSpace(mustRun(t, "git", "-C", root, "rev-parse", "HEAD"))

		withStdin(t, pushRefLine("main", head))
		if got := Run(nil, "test", []string{"hooks", "run", "pre-push", "origin"}); got != exitOK {
			t.Fatalf("hooks run pre-push = %d, want %d", got, exitOK)
		}
	})

	t.Run("feature branch carrying only code is allowed", func(t *testing.T) {
		root := newProject(t)
		gitInitWithRemote(t, root, "main")
		if out, err := exec.Command("git", "-C", root, "checkout", "-qb", "feat/T-997-code").CombinedOutput(); err != nil {
			t.Fatalf("git checkout: %v: %s", err, out)
		}
		if err := os.WriteFile(filepath.Join(root, "code.go"), []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitAdd(t, root, "code.go")
		if out, err := exec.Command("git", "-C", root, "commit", "-qm", "feat: code").CombinedOutput(); err != nil {
			t.Fatalf("git commit: %v: %s", err, out)
		}
		head := strings.TrimSpace(mustRun(t, "git", "-C", root, "rev-parse", "HEAD"))

		withStdin(t, pushRefLine("feat/T-997-code", head))
		if got := Run(nil, "test", []string{"hooks", "run", "pre-push", "origin"}); got != exitOK {
			t.Fatalf("hooks run pre-push = %d, want %d", got, exitOK)
		}
	})

	t.Run("not a git repository exits 0", func(t *testing.T) {
		newProject(t) // no git init: the guard cannot decide
		withStdin(t, "")
		if got := Run(nil, "test", []string{"hooks", "run", "pre-push"}); got != exitOK {
			t.Fatalf("hooks run pre-push = %d, want %d (degraded, must not block)", got, exitOK)
		}
	})

	t.Run("no pickle.toml exits 0", func(t *testing.T) {
		t.Chdir(t.TempDir())
		withStdin(t, "")
		if got := Run(nil, "test", []string{"hooks", "run", "pre-push"}); got != exitOK {
			t.Fatalf("hooks run pre-push = %d, want %d (not a pickle project)", got, exitOK)
		}
	})

	t.Run("missing remote name defaults to origin", func(t *testing.T) {
		root := newProject(t)
		gitInitWithRemote(t, root, "main")
		if out, err := exec.Command("git", "-C", root, "checkout", "-qb", "feat/T-999-demo").CombinedOutput(); err != nil {
			t.Fatalf("git checkout: %v: %s", err, out)
		}
		if err := os.WriteFile(filepath.Join(root, "tickets", "1-to-do", "T-999-demo.md"), []byte("# T-999\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitAdd(t, root, "tickets")
		if out, err := exec.Command("git", "-C", root, "commit", "-qm", "board: T-999 demo").CombinedOutput(); err != nil {
			t.Fatalf("git commit: %v: %s", err, out)
		}
		head := strings.TrimSpace(mustRun(t, "git", "-C", root, "rev-parse", "HEAD"))

		withStdin(t, pushRefLine("feat/T-999-demo", head))
		// No positional argument at all — git always passes one, but the shim's
		// contract (decision) defaults it exactly as `git push` itself does.
		if got := Run(nil, "test", []string{"hooks", "run", "pre-push"}); got != exitError {
			t.Fatalf("hooks run pre-push = %d, want %d (violation, remote defaulted to origin)", got, exitError)
		}
	})
}

// mustRun runs name with args and fails the test on error, returning stdout.
func mustRun(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return string(out)
}

func TestHooksInstallStatusUninstall(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "install"}); got != exitOK {
			t.Fatalf("hooks install = %d", got)
		}
	})
	for _, want := range []string{"pre-commit", "pre-push"} {
		if !strings.Contains(out, want) {
			t.Errorf("hooks install did not report %s:\n%s", want, out)
		}
	}

	out = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "status"}); got != exitOK {
			t.Fatalf("hooks status = %d", got)
		}
	})
	if n := strings.Count(out, "installed by pickle"); n != 2 {
		t.Errorf("hooks status reported ownership %d time(s), want 2 (one per hook):\n%s", n, out)
	}

	out = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "uninstall", "-n"}); got != exitOK {
			t.Fatalf("hooks uninstall -n = %d", got)
		}
	})
	if !strings.Contains(out, "dry-run") {
		t.Errorf("dry-run not reported as such:\n%s", out)
	}
	for _, name := range []string{"pre-commit", "pre-push"} {
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", name)); err != nil {
			t.Errorf("dry-run removed %s: %v", name, err)
		}
	}

	if got := Run(nil, "test", []string{"hooks", "uninstall"}); got != exitOK {
		t.Fatalf("hooks uninstall = %d", got)
	}
	for _, name := range []string{"pre-commit", "pre-push"} {
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", name)); !os.IsNotExist(err) {
			t.Errorf("%s survived uninstall: %v", name, err)
		}
	}

	out = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "status"}); got != exitOK {
			t.Fatalf("hooks status (absent) = %d", got)
		}
	})
	if n := strings.Count(out, "absent"); n != 2 {
		t.Errorf("hooks status reported absent %d time(s), want 2 (one per hook):\n%s", n, out)
	}
}

// TestHooksInstallWarnsWhenPathPickleIsInert pins T-068: the shim written by
// `hooks install` is always current, but that is only half the guarantee — the
// binary the shim will actually resolve from PATH must be able to run it, and
// this is the one moment the evidence exists and the user is looking.
func TestHooksInstallWarnsWhenPathPickleIsInert(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")
	pinPathWithPickle(t, stubPickleBin(t, 2)) // mirrors an old pickle's exit 2 on the unknown `hooks` verb

	errOut := captureStderr(t, func() {
		if got := Run(nil, "test", []string{"hooks", "install"}); got != exitOK {
			t.Fatalf("hooks install = %d", got)
		}
	})
	if !strings.Contains(errOut, "inert") {
		t.Errorf("hooks install did not warn about an inert guard:\n%s", errOut)
	}
	if !strings.Contains(errOut, "0.2.2") {
		t.Errorf("warning does not name the incapable binary's version:\n%s", errOut)
	}
}

// TestHooksInstallNoWarningWhenPathPickleIsCapable is the negative of the
// above: a capable PATH pickle (even a different file from this binary) must
// stay silent — path difference alone is never a finding.
func TestHooksInstallNoWarningWhenPathPickleIsCapable(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")
	pinPathWithPickle(t, stubPickleBin(t, 0))

	errOut := captureStderr(t, func() {
		if got := Run(nil, "test", []string{"hooks", "install"}); got != exitOK {
			t.Fatalf("hooks install = %d", got)
		}
	})
	if strings.Contains(errOut, "warning") {
		t.Errorf("hooks install warned despite a capable PATH pickle:\n%s", errOut)
	}
}

// TestHooksStatusReportsPathCapability: `hooks status` reports the file's own
// state *and* whether the PATH pickle can actually run it — two different
// signals the T-068 measurement showed both saying "healthy" together.
func TestHooksStatusReportsPathCapability(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")
	if got := Run(nil, "test", []string{"hooks", "install"}); got != exitOK {
		t.Fatalf("hooks install = %d", got)
	}
	pinPathWithPickle(t, stubPickleBin(t, 2))

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "status"}); got != exitOK {
			t.Fatalf("hooks status = %d", got)
		}
	})
	if !strings.Contains(out, "inert") {
		t.Errorf("hooks status did not report the inert PATH pickle:\n%s", out)
	}
}

// TestHooksInstallReportsSiblingHookDespiteAForeignOne pins T-082's first
// review finding F4: a per-hook failure (a foreign hook without --force)
// must not suppress the report for a sibling hook that installed cleanly, or
// print an empty parenthetical for the refused one.
func TestHooksInstallReportsSiblingHookDespiteAForeignOne(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")
	foreign := "#!/bin/sh\n# husky\nexit 0\n"
	if err := os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-push"), []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	var exit int
	out := captureStdout(t, func() {
		exit = Run(nil, "test", []string{"hooks", "install"})
	})
	if exit != exitError {
		t.Fatalf("hooks install = %d, want %d (the foreign pre-push refusal)", exit, exitError)
	}
	if !strings.Contains(out, "+ ") || !strings.Contains(out, filepath.Join(root, ".git", "hooks", "pre-commit")) {
		t.Errorf("hooks install did not report the installed pre-commit hook:\n%s", out)
	}
	if strings.Contains(out, "pre-push ()") {
		t.Errorf("hooks install printed an empty parenthetical for the foreign hook:\n%s", out)
	}
	if !strings.Contains(out, "not pickle's") {
		t.Errorf("hooks install did not name why pre-push was skipped:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("pre-commit was not actually installed: %v", err)
	}
}

// TestInstallHooksReportsSiblingHookDespiteAForeignOne is
// TestHooksInstallReportsSiblingHookDespiteAForeignOne's twin for the other
// caller (`pickle install --hooks`), which had the same bug: InstallAll's
// error skipped its whole result-reporting block, so a hook that *did* land
// went unreported.
func TestInstallHooksReportsSiblingHookDespiteAForeignOne(t *testing.T) {
	gitOrSkip(t)
	root := t.TempDir()
	t.Chdir(root)
	gitInit(t, root, "main")
	foreign := "#!/bin/sh\n# husky\nexit 0\n"
	if err := os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-push"), []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if got := Run(os.DirFS(repoRoot), "test", []string{"install", "--in-tree", "--project", "demo", "--hooks"}); got != exitOK {
			t.Fatalf("install --hooks = %d", got)
		}
	})
	if !strings.Contains(out, filepath.Join(root, ".git", "hooks", "pre-commit")) {
		t.Errorf("install --hooks did not report the installed pre-commit hook:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("pre-commit was not actually installed: %v", err)
	}
}

// TestHooksStatusProbesPathOnce pins T-082's first review finding F3: with
// both hooks installed and current, `hooks status` must probe the PATH
// pickle exactly once, not once per hook (it is a per-binary question, both
// hooks ship in the same binary — decision 6), and the doc this branch wrote
// (cli-reference.adoc: "checked once, not once per hook") must hold.
func TestHooksStatusProbesPathOnce(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")
	if got := Run(nil, "test", []string{"hooks", "install"}); got != exitOK {
		t.Fatalf("hooks install = %d", got)
	}
	countFile := filepath.Join(t.TempDir(), "count")
	pinPathWithPickle(t, stubPickleBinCounting(t, countFile))

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "status"}); got != exitOK {
			t.Fatalf("hooks status = %d", got)
		}
	})
	// The line is reported *per hook* ("for an installed, current hook",
	// cli-reference.adoc) — both are owned and current here, so two lines is
	// correct. What must not double is the exec behind it (checked next).
	if n := strings.Count(out, "can run the guard"); n != 2 {
		t.Errorf("hooks status printed the PATH-capability line %d time(s), want 2 (one per hook):\n%s", n, out)
	}
	body, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("reading count file: %v", err)
	}
	if n := strings.Count(string(body), "x"); n != 1 {
		t.Errorf("the stub pickle was exec'd %d time(s) by the probe, want 1", n)
	}
}

// TestUnknownCommandIsTerse pins T-068's cosmetic half: an old pickle answers
// its own unknown-verb case tersely too (that binary cannot be changed
// retroactively), but *this* binary's own unknown-command path must not dump
// the full usage() text ahead of a caller's one-line notice — exactly what
// buried the shim's real message in ~40 lines of noise on every commit.
func TestUnknownCommandIsTerse(t *testing.T) {
	errOut := captureStderr(t, func() {
		if got := Run(nil, "test", []string{"frobnicate"}); got != exitUsage {
			t.Fatalf("Run = %d, want %d", got, exitUsage)
		}
	})
	if strings.Count(errOut, "\n") != 1 {
		t.Errorf("unknown command produced %d line(s), want exactly one:\n%s", strings.Count(errOut, "\n"), errOut)
	}
	if !strings.Contains(errOut, "pickle help") {
		t.Errorf("unknown-command message does not point at `pickle help`:\n%s", errOut)
	}
	for _, absent := range []string{"Usage:", "Flow commands:", "Setup commands:"} {
		if strings.Contains(errOut, absent) {
			t.Errorf("unknown-command message still dumps full usage (%q present):\n%s", absent, errOut)
		}
	}
}

// TestNoArgsStillPrintsFullUsage guards the deliberate asymmetry: only a typed,
// unrecognised command is terse (T-068) — running pickle bare is exactly when
// the full usage is what's wanted.
func TestNoArgsStillPrintsFullUsage(t *testing.T) {
	errOut := captureStderr(t, func() {
		if got := Run(nil, "test", nil); got != exitUsage {
			t.Fatalf("Run = %d, want %d", got, exitUsage)
		}
	})
	if !strings.Contains(errOut, "Usage:") || !strings.Contains(errOut, "Flow commands:") {
		t.Errorf("no-args invocation no longer prints full usage:\n%s", errOut)
	}
}

// TestHooksAreAdvertised: an installed command that `pickle help` does not
// mention is a command nobody runs.
func TestHooksAreAdvertised(t *testing.T) {
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"help"}); got != exitOK {
			t.Fatalf("help = %d", got)
		}
	})
	for _, want := range []string{"hooks install", "hooks status", "hooks run pre-commit"} {
		if !strings.Contains(out, want) {
			t.Errorf("help does not mention %q", want)
		}
	}
}
