package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"run with a stray argument", []string{"hooks", "run", "pre-commit", "extra"}},
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

func TestHooksInstallStatusUninstall(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "install"}); got != exitOK {
			t.Fatalf("hooks install = %d", got)
		}
	})
	if !strings.Contains(out, "pre-commit") {
		t.Errorf("hooks install did not report the path:\n%s", out)
	}

	out = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "status"}); got != exitOK {
			t.Fatalf("hooks status = %d", got)
		}
	})
	if !strings.Contains(out, "installed by pickle") {
		t.Errorf("hooks status does not report ownership:\n%s", out)
	}

	out = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "uninstall", "-n"}); got != exitOK {
			t.Fatalf("hooks uninstall -n = %d", got)
		}
	})
	if !strings.Contains(out, "dry-run") {
		t.Errorf("dry-run not reported as such:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("dry-run removed the hook: %v", err)
	}

	if got := Run(nil, "test", []string{"hooks", "uninstall"}); got != exitOK {
		t.Fatalf("hooks uninstall = %d", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("hook survived uninstall: %v", err)
	}

	out = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"hooks", "status"}); got != exitOK {
			t.Fatalf("hooks status (absent) = %d", got)
		}
	})
	if !strings.Contains(out, "absent") {
		t.Errorf("hooks status does not report an absent hook:\n%s", out)
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
