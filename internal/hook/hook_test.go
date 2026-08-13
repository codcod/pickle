package hook

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

// TestMain skips the whole package when git is unavailable: every test here
// drives a real repository on purpose, because the bugs this package can have
// (an unborn branch read as detached, an inherited GIT_INDEX_FILE pointing at
// another index) only exist in real git behaviour and cannot be faked.
func TestMain(m *testing.M) {
	if _, err := exec.LookPath("git"); err != nil {
		os.Stdout.WriteString("hook: git not on PATH — skipping package\n")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// isolate detaches the test from the developer's git environment. Both halves
// matter: an inherited GIT_DIR/GIT_INDEX_FILE would point git at another
// repository, and an inherited core.hooksPath in ~/.gitconfig would make the
// install tests write outside t.TempDir() — on the developer's real hooks.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	for _, name := range repoEnv {
		// Blank them through t.Setenv (so the original is restored) and then
		// actually unset: git reads GIT_DIR="" as an empty repository path.
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
}

// mustGit runs git in dir and fails the test on error.
func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, errb.String())
	}
	return strings.TrimSpace(out.String())
}

// newRepo creates an isolated git repository with a pickle.toml and a tickets/
// tree, on the given initial branch, and returns its root.
func newRepo(t *testing.T, branch string, branchPrefix string) string {
	t.Helper()
	isolate(t)
	root := t.TempDir()
	mustGit(t, root, "init", "-q", "-b", branch)
	mustGit(t, root, "config", "user.email", "test@example.com")
	mustGit(t, root, "config", "user.name", "test")
	mustGit(t, root, "config", "commit.gpgsign", "false")
	writeConfig(t, root, branchPrefix)
	return root
}

func writeConfig(t *testing.T, root, branchPrefix string) {
	t.Helper()
	toml := "payload_version = \"test\"\n\n[[project]]\nname = \"demo\"\npath = \".\"\n"
	if branchPrefix != "" {
		toml += "branch_prefix = \"" + branchPrefix + "\"\n"
	}
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tickets", "1-to-do"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// loadConfig loads the repository's pickle.toml.
func loadConfig(t *testing.T, root string) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

// stageTicket stages a ticket file (the bookkeeping the guard cares about).
func stageTicket(t *testing.T, root, name string) {
	t.Helper()
	p := filepath.Join(root, "tickets", "1-to-do", name)
	if err := os.WriteFile(p, []byte("# ticket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", filepath.ToSlash(filepath.Join("tickets", "1-to-do", name)))
}

// stageCode stages a non-bookkeeping file.
func stageCode(t *testing.T, root, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte("code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", name)
}

// preCommitIn runs PreCommit with cwd set to dir, the way git runs a hook.
func preCommitIn(t *testing.T, dir string, cfg *config.Config) (bool, string) {
	t.Helper()
	t.Chdir(dir)
	var msg bytes.Buffer
	ok, err := PreCommit(cfg, &msg)
	if err != nil {
		t.Fatalf("PreCommit: %v", err)
	}
	return ok, msg.String()
}

func TestInstallWritesAnExecutableOwnedShim(t *testing.T) {
	root := newRepo(t, "main", "")

	res, err := Install(root, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !res.Changed {
		t.Fatalf("Install reported no change: %+v", res)
	}
	fi, err := os.Stat(res.Path)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("hook mode = %v, want 0755 (git ignores a non-executable hook)", fi.Mode().Perm())
	}
	body, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), marker()) {
		t.Errorf("hook does not carry the ownership marker %q:\n%s", marker(), body)
	}

	// Re-install is idempotent and says so rather than rewriting.
	again, err := Install(root, false)
	if err != nil {
		t.Fatalf("re-Install: %v", err)
	}
	if again.Changed || again.Skipped != "current" {
		t.Errorf("re-Install = %+v, want unchanged/current", again)
	}
}

// TestShimBlocksOnlyExitCodeOne pins T-057 finding B3: the shim must treat exit
// 1 as the *only* violation signal, so an older pickle on PATH (which exits 2 on
// the unknown `hooks` verb) degrades to no guard instead of blocking every
// commit. Asserting on the text is deliberate — this is a shipped contract.
func TestShimBlocksOnlyExitCodeOne(t *testing.T) {
	s := Shim()
	for _, want := range []string{
		"command -v pickle >/dev/null 2>&1 || {",
		"pickle not found on PATH",
		`[ "$rc" -eq 1 ] && exit 1`,
		"exit 0\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("shim missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "|| exit 1\n") {
		t.Errorf("shim is fail-closed on version skew (`|| exit 1`); see T-057 finding B3:\n%s", s)
	}
}

// TestShimExitCodes runs the real shim against stub pickles to prove the
// contract end to end: only exit 1 stops a commit, and (T-068) the two
// degraded paths — an old binary and no binary at all — both say so on
// stderr instead of failing silently.
func TestShimExitCodes(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	shim := filepath.Join(dir, "pre-commit")
	if err := writeExecutable(shim, Shim()); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		stub       string // body of the fake `pickle` on PATH, "" for none
		wantExit   int
		wantStderr string // substring to require on stderr, "" = don't check
	}{
		{"violation", "#!/bin/sh\nexit 1\n", 1, ""},
		{"allowed", "#!/bin/sh\nexit 0\n", 0, ""},
		{"old binary, unknown verb", "#!/bin/sh\nexit 2\n", 0, "hooks run exited 2"},
		{"internal error", "#!/bin/sh\nexit 7\n", 0, "hooks run exited 7"},
		{"not on PATH", "", 0, "pickle not found on PATH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			if tc.stub != "" {
				if err := writeExecutable(filepath.Join(bin, "pickle"), tc.stub); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(shim)
			cmd.Dir = dir
			cmd.Env = []string{"PATH=" + bin}
			var errBuf bytes.Buffer
			cmd.Stderr = &errBuf
			err := cmd.Run()
			got := 0
			var ee *exec.ExitError
			if err != nil {
				if !errorAs(err, &ee) {
					t.Fatalf("run shim: %v", err)
				}
				got = ee.ExitCode()
			}
			if got != tc.wantExit {
				t.Errorf("shim exit = %d, want %d", got, tc.wantExit)
			}
			if tc.wantStderr != "" && !strings.Contains(errBuf.String(), tc.wantStderr) {
				t.Errorf("shim stderr = %q, want it to contain %q", errBuf.String(), tc.wantStderr)
			}
		})
	}
}

// errorAs is errors.As, spelled locally to keep the import list of this test
// file to what it actually exercises.
func errorAs(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// TestShimPassesShellcheck runs shellcheck over the actual shim text (Shim()),
// with -s sh — not the default dialect inference — because the shebang is
// #!/bin/sh and the shim's portability is the point: it runs on whatever
// /bin/sh a user has, so it must not contain bashisms shellcheck would only
// catch under -s sh.
//
// This exists because Shim() is a Go string literal, not a .sh file, so it is
// the one piece of shell source in the repo that `shellcheck **/*.sh` (or
// lint-ci-surface, T-088) can never see. It is also the highest-stakes shell
// in the project: it is written into every user's .git/hooks/pre-commit and
// runs on every one of their commits, and its exit-code handling is load-
// bearing (T-057 decision 3 — only exit 1 blocks a commit, everything else is
// reported and waved through). A regression here ships to every installed
// hook, so it gets its own check rather than riding along on eyeball review.
func TestShimPassesShellcheck(t *testing.T) {
	if _, err := exec.LookPath("shellcheck"); err != nil {
		t.Skip("shellcheck not installed")
	}

	path := filepath.Join(t.TempDir(), "pre-commit")
	if err := os.WriteFile(path, []byte(Shim()), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("shellcheck", "-s", "sh", path).CombinedOutput()
	if err != nil {
		t.Errorf("shellcheck flagged the pre-commit shim:\n%s", out)
	}
}

func TestInstallRefusesAForeignHookUnlessForced(t *testing.T) {
	root := newRepo(t, "main", "")
	dir, err := HooksDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, HookName)
	foreign := "#!/bin/sh\n# husky\nexit 0\n"
	if err := writeExecutable(path, foreign); err != nil {
		t.Fatal(err)
	}

	res, err := Install(root, false)
	if err == nil {
		t.Fatalf("Install replaced a foreign hook: %+v", res)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("refusal does not mention --force: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != foreign {
		t.Errorf("foreign hook was modified:\n%s", got)
	}

	if _, err := Install(root, true); err != nil {
		t.Fatalf("Install --force: %v", err)
	}
	if got, _ := os.ReadFile(path); !strings.Contains(string(got), marker()) {
		t.Errorf("--force did not install the shim:\n%s", got)
	}
}

// TestInstallHonoursCoreHooksPath is why the hooks directory is resolved by
// asking git instead of assuming .git/hooks: a project using Husky or Lefthook
// redirects it, and a shim written to .git/hooks there would never run.
func TestInstallHonoursCoreHooksPath(t *testing.T) {
	root := newRepo(t, "main", "")
	custom := filepath.Join(root, ".githooks")
	mustGit(t, root, "config", "core.hooksPath", custom)

	res, err := Install(root, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Compare resolved paths: git answers with the real path, and on macOS the
	// temp dir is reached through a /tmp -> /private/tmp symlink.
	if realpath(filepath.Dir(res.Path)) != realpath(custom) {
		t.Errorf("hook installed at %s, want it under %s", res.Path, custom)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", HookName)); err == nil {
		t.Error(".git/hooks/pre-commit was written even though core.hooksPath redirects hooks")
	}
}

func TestUninstallOnlyRemovesPickleOwnedHooks(t *testing.T) {
	root := newRepo(t, "main", "")

	// Absent: nothing to do, no error.
	res, err := Uninstall(root, false)
	if err != nil {
		t.Fatalf("Uninstall (absent): %v", err)
	}
	if res.Changed || res.Kind != KindAbsent {
		t.Errorf("Uninstall (absent) = %+v", res)
	}

	if _, err := Install(root, false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(mustHooksDir(t, root), HookName)

	// Dry run mutates nothing.
	res, err = Uninstall(root, true)
	if err != nil {
		t.Fatalf("Uninstall (dry-run): %v", err)
	}
	if !res.Would || res.Changed {
		t.Errorf("Uninstall (dry-run) = %+v, want Would without Changed", res)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("dry-run removed the hook: %v", err)
	}

	res, err = Uninstall(root, false)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !res.Changed {
		t.Errorf("Uninstall = %+v, want Changed", res)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("hook still present after uninstall: %v", err)
	}

	// A foreign hook survives.
	foreign := "#!/bin/sh\nexit 0\n"
	if err := writeExecutable(path, foreign); err != nil {
		t.Fatal(err)
	}
	res, err = Uninstall(root, false)
	if err != nil {
		t.Fatalf("Uninstall (foreign): %v", err)
	}
	if res.Changed || res.Kind != KindForeign {
		t.Errorf("Uninstall (foreign) = %+v", res)
	}
	if got, _ := os.ReadFile(path); string(got) != foreign {
		t.Errorf("foreign hook was touched:\n%s", got)
	}
}

func mustHooksDir(t *testing.T, root string) string {
	t.Helper()
	dir, err := HooksDir(root)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStatusAndRefresh(t *testing.T) {
	root := newRepo(t, "main", "")
	path := filepath.Join(mustHooksDir(t, root), HookName)

	if st, err := Status(root); err != nil || st.Kind != KindAbsent {
		t.Fatalf("Status (absent) = %+v, %v", st, err)
	}
	if _, err := Install(root, false); err != nil {
		t.Fatal(err)
	}
	st, err := Status(root)
	if err != nil || st.Kind != KindOwned || st.Stale || st.Version != ShimVersion {
		t.Fatalf("Status (owned) = %+v, %v", st, err)
	}

	// Refresh leaves a current shim alone.
	if res, err := Refresh(root); err != nil || res.Changed {
		t.Errorf("Refresh (current) = %+v, %v", res, err)
	}

	// A shim from an older pickle is stale, and Refresh rewrites exactly it.
	stale := strings.Replace(Shim(), marker(), markerPrefix+"0", 1)
	if err := writeExecutable(path, stale); err != nil {
		t.Fatal(err)
	}
	st, err = Status(root)
	if err != nil || st.Kind != KindOwned || !st.Stale || st.Version != 0 {
		t.Fatalf("Status (stale) = %+v, %v", st, err)
	}
	res, err := Refresh(root)
	if err != nil || !res.Changed {
		t.Fatalf("Refresh (stale) = %+v, %v", res, err)
	}
	if got, _ := os.ReadFile(path); string(got) != Shim() {
		t.Errorf("Refresh did not rewrite the shim:\n%s", got)
	}

	// Refresh never adopts a foreign hook and never installs an absent one.
	foreign := "#!/bin/sh\nexit 0\n"
	if err := writeExecutable(path, foreign); err != nil {
		t.Fatal(err)
	}
	if res, err := Refresh(root); err != nil || res.Changed {
		t.Errorf("Refresh (foreign) = %+v, %v", res, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if res, err := Refresh(root); err != nil || res.Changed {
		t.Errorf("Refresh (absent) = %+v, %v — the guard is opt-in", res, err)
	}
}

func TestPreCommit(t *testing.T) {
	for _, tc := range []struct {
		name         string
		branch       string
		branchPrefix string
		stage        func(t *testing.T, root string)
		wantOK       bool
	}{
		{
			name:   "base branch with bookkeeping is allowed",
			branch: "main",
			stage:  func(t *testing.T, root string) { stageTicket(t, root, "T-001-x.md") },
			wantOK: true,
		},
		{
			name:   "feature branch with bookkeeping is refused",
			branch: "feat/T-001-x",
			stage:  func(t *testing.T, root string) { stageTicket(t, root, "T-001-x.md") },
			wantOK: false,
		},
		{
			name:   "feature branch with code only is allowed",
			branch: "feat/T-001-x",
			stage:  func(t *testing.T, root string) { stageCode(t, root, "code.go") },
			wantOK: true,
		},
		{
			name:   "feature branch with a mixed commit is refused",
			branch: "feat/T-001-x",
			stage: func(t *testing.T, root string) {
				stageCode(t, root, "code.go")
				stageTicket(t, root, "T-001-x.md")
			},
			wantOK: false,
		},
		{
			name:   "feature branch with nothing staged is allowed",
			branch: "feat/T-001-x",
			stage:  func(t *testing.T, root string) {},
			wantOK: true,
		},
		{
			name:         "a configured branch_prefix is what counts",
			branch:       "wip/T-001-x",
			branchPrefix: "wip/",
			stage:        func(t *testing.T, root string) { stageTicket(t, root, "T-001-x.md") },
			wantOK:       false,
		},
		{
			name:         "feat/ is not special once branch_prefix is set",
			branch:       "feat/T-001-x",
			branchPrefix: "wip/",
			stage:        func(t *testing.T, root string) { stageTicket(t, root, "T-001-x.md") },
			wantOK:       true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newRepo(t, tc.branch, tc.branchPrefix)
			tc.stage(t, root)
			ok, msg := preCommitIn(t, root, loadConfig(t, root))
			if ok != tc.wantOK {
				t.Fatalf("PreCommit ok = %v, want %v (message: %s)", ok, tc.wantOK, msg)
			}
			if ok {
				if msg != "" {
					t.Errorf("allowed commit still produced a message:\n%s", msg)
				}
				return
			}
			for _, want := range []string{tc.branch, "tickets/", "--no-verify"} {
				if !strings.Contains(msg, want) {
					t.Errorf("rejection message missing %q:\n%s", want, msg)
				}
			}
		})
	}
}

// TestPreCommitUnbornFeatureBranch pins T-057 finding B2. A freshly `git
// init`-ed repository has an unborn branch, which is exactly where `pickle
// install` lands; `git rev-parse --abbrev-ref HEAD` fails there and prints
// "HEAD", which reads as a detached HEAD and would wave the commit through.
func TestPreCommitUnbornFeatureBranch(t *testing.T) {
	root := newRepo(t, "feat/T-001-x", "")
	if out := mustGit(t, root, "rev-list", "--all", "--count"); out != "0" {
		t.Fatalf("expected an unborn branch, got %s commit(s)", out)
	}
	stageTicket(t, root, "T-001-x.md")

	ok, msg := preCommitIn(t, root, loadConfig(t, root))
	if ok {
		t.Error("bookkeeping on an unborn feature branch was allowed (T-057 finding B2)")
	}
	if !strings.Contains(msg, "feat/T-001-x") {
		t.Errorf("message does not name the branch:\n%s", msg)
	}
}

// TestPreCommitDuringAMerge covers the in-progress-operation escape: concluding
// a merge, cherry-pick or revert commits files the user never staged by hand,
// and refusing there would leave them stuck mid-operation.
func TestPreCommitDuringAMerge(t *testing.T) {
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"} {
		t.Run(name, func(t *testing.T) {
			root := newRepo(t, "feat/T-001-x", "")
			stageCode(t, root, "seed.txt")
			mustGit(t, root, "commit", "-qm", "seed")
			stageTicket(t, root, "T-001-x.md")

			head := mustGit(t, root, "rev-parse", "HEAD")
			if err := os.WriteFile(filepath.Join(root, ".git", name), []byte(head+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if ok, msg := preCommitIn(t, root, loadConfig(t, root)); !ok {
				t.Errorf("refused a commit concluding a %s operation:\n%s", name, msg)
			}
		})
	}
}

// TestPreCommitDegradesWithoutGit pins T-057 finding B4: no repository and no
// git at all are normal states, and a guard that cannot decide must not block.
func TestPreCommitDegradesWithoutGit(t *testing.T) {
	t.Run("not a repository", func(t *testing.T) {
		isolate(t)
		root := t.TempDir()
		writeConfig(t, root, "")
		if ok, _ := preCommitIn(t, root, loadConfig(t, root)); !ok {
			t.Error("PreCommit blocked a commit outside a git repository")
		}
	})

	t.Run("git not on PATH", func(t *testing.T) {
		root := newRepo(t, "feat/T-001-x", "")
		stageTicket(t, root, "T-001-x.md")
		cfg := loadConfig(t, root)
		t.Setenv("PATH", t.TempDir()) // an empty PATH: no git
		if ok, _ := preCommitIn(t, root, cfg); !ok {
			t.Error("PreCommit blocked a commit with git unavailable")
		}
	})

	t.Run("Status and Refresh without a repository", func(t *testing.T) {
		isolate(t)
		dir := t.TempDir()
		st, err := Status(dir)
		if err != nil || st.Kind != KindNoRepo {
			t.Errorf("Status = %+v, %v; want KindNoRepo without an error", st, err)
		}
		res, err := Refresh(dir)
		if err != nil || res.Changed {
			t.Errorf("Refresh = %+v, %v; want a silent no-op", res, err)
		}
		if _, err := Install(dir, false); err == nil {
			t.Error("Install succeeded outside a git repository; the user asked for a hook and cannot have one")
		}
	})
}

// TestPreCommitTicketsOutsideTheRepo covers the multi-repo case: a child that is
// its own repository has no bookkeeping of its own to guard.
func TestPreCommitTicketsOutsideTheRepo(t *testing.T) {
	isolate(t)
	base := t.TempDir()
	writeConfig(t, base, "")
	child := filepath.Join(base, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, child, "init", "-q", "-b", "feat/T-001-x")
	mustGit(t, child, "config", "user.email", "test@example.com")
	mustGit(t, child, "config", "user.name", "test")
	stageCode(t, child, "code.go")

	// The config lives above the child repo, so tickets/ is outside it.
	if ok, msg := preCommitIn(t, child, loadConfig(t, base)); !ok {
		t.Errorf("refused a commit in a child repo that holds no tickets/:\n%s", msg)
	}
}

// TestPreCommitInheritsTheHooksEnvironment pins T-057 finding F8. git runs
// pre-commit with GIT_INDEX_FILE set to a *relative* path (and GIT_PREFIX set
// when the commit was started from a subdirectory); a `git -C <root>` call in
// this path would resolve that relative index against the wrong directory and
// silently inspect something other than the commit being made.
func TestPreCommitInheritsTheHooksEnvironment(t *testing.T) {
	root := newRepo(t, "feat/T-001-x", "")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	stageTicket(t, root, "T-001-x.md")
	cfg := loadConfig(t, root)

	// As git would: cwd at the top of the worktree, a relative index path, and
	// GIT_PREFIX naming the directory the commit was started from.
	t.Setenv("GIT_INDEX_FILE", filepath.Join(".git", "index"))
	t.Setenv("GIT_PREFIX", "sub/")
	if ok, _ := preCommitIn(t, root, cfg); ok {
		t.Error("guard missed the violation when run with git's own hook environment")
	}

	// And when invoked by hand from a subdirectory, with no GIT_* at all.
	for _, name := range repoEnv {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	if ok, _ := preCommitIn(t, sub, cfg); ok {
		t.Error("guard missed the violation when run from a subdirectory")
	}
}

// TestPreCommitInALinkedWorktree is the other half of finding F8: in a linked
// worktree GIT_DIR points at .git/worktrees/<name>, so a hardcoded root would
// read the main worktree's index instead of this one's.
func TestPreCommitInALinkedWorktree(t *testing.T) {
	root := newRepo(t, "main", "")
	stageCode(t, root, "seed.txt")
	mustGit(t, root, "add", config.FileName)
	mustGit(t, root, "commit", "-qm", "seed")

	wt := filepath.Join(t.TempDir(), "wt")
	mustGit(t, root, "worktree", "add", "-q", "-b", "feat/T-001-x", wt)
	if _, err := os.Stat(filepath.Join(wt, config.FileName)); err != nil {
		t.Fatalf("worktree is missing pickle.toml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(wt, "tickets", "1-to-do"), 0o755); err != nil {
		t.Fatal(err)
	}
	stageTicket(t, wt, "T-001-x.md")

	// The worktree carries its own checkout of pickle.toml — the config a hook
	// running there discovers.
	if ok, msg := preCommitIn(t, wt, loadConfig(t, wt)); ok {
		t.Errorf("guard missed the violation inside a linked worktree:\n%s", msg)
	}
}

// TestRejectionMessageCapsThePathList keeps the message readable when a whole
// directory of tickets is staged.
func TestRejectionMessageCapsThePathList(t *testing.T) {
	root := newRepo(t, "feat/T-001-x", "")
	for i := range maxListedPaths + 3 {
		stageTicket(t, root, "T-0"+strconv.Itoa(100+i)+"-x.md")
	}
	ok, msg := preCommitIn(t, root, loadConfig(t, root))
	if ok {
		t.Fatal("expected a violation")
	}
	if !strings.Contains(msg, "and 3 more") {
		t.Errorf("message does not summarise the remaining paths:\n%s", msg)
	}
	if n := strings.Count(msg, "tickets/1-to-do/"); n != maxListedPaths {
		t.Errorf("listed %d paths, want %d", n, maxListedPaths)
	}
}
