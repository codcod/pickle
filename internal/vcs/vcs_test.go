package vcs

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// gitInit mirrors internal/doctor/hooks_test.go's fixture pattern: a real,
// minimal repository, so ChildState exercises the actual git subcommands
// rather than a fake.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", dir, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	// Give the fixture repo an identity so a later `git add`/commit in a test
	// that needs one does not fail on a machine with no global git config.
	if out, err := exec.Command("git", "-C", dir, "config", "user.email", "vcs-test@example.com").CombinedOutput(); err != nil {
		t.Fatalf("git config user.email: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", dir, "config", "user.name", "vcs test").CombinedOutput(); err != nil {
		t.Fatalf("git config user.name: %v: %s", err, out)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func TestChildStateIgnored(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root)
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/child/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := ChildState(root, "child"); got != Ignored {
		t.Fatalf("ChildState = %v, want Ignored", got)
	}
	if adv := Ignored.Advice("child"); adv != "" {
		t.Fatalf("Advice(Ignored) = %q, want empty", adv)
	}
}

func TestChildStateStageable(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root)
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := ChildState(root, "child"); got != Stageable {
		t.Fatalf("ChildState = %v, want Stageable", got)
	}
	adv := Stageable.Advice("child")
	if adv == "" {
		t.Fatal("Advice(Stageable) is empty, want a message")
	}
	if want := "/child/"; !strings.Contains(adv, want) {
		t.Fatalf("Advice(Stageable) = %q, want it to mention %q", adv, want)
	}
}

func TestChildStateTracked(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	gitInit(t, root)
	nested := filepath.Join(root, "child")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	gitInit(t, nested)
	if out, err := exec.Command("git", "-C", nested, "commit", "-q", "--allow-empty", "-m", "x").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	// `git add` on a directory that is itself a repo records it as a gitlink
	// (a submodule-shaped index entry) without needing `.gitmodules` — exactly
	// the "deliberately tracked" case ChildState must recognise.
	if out, err := exec.Command("git", "-C", root, "add", "child").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	if got := ChildState(root, "child"); got != Tracked {
		t.Fatalf("ChildState = %v, want Tracked", got)
	}
	if adv := Tracked.Advice("child"); adv != "" {
		t.Fatalf("Advice(Tracked) = %q, want empty", adv)
	}
}

func TestChildStateUnknownWhenNotARepo(t *testing.T) {
	requireGit(t)
	root := t.TempDir() // no `git init` at all
	if err := os.Mkdir(filepath.Join(root, "child"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := ChildState(root, "child"); got != Unknown {
		t.Fatalf("ChildState = %v, want Unknown", got)
	}
	if adv := Unknown.Advice("child"); adv != "" {
		t.Fatalf("Advice(Unknown) = %q, want empty", adv)
	}
}

// TestAdviceEntryActuallyIgnores (T-051 review F2) pins the property the
// advice exists to deliver, rather than its spelling: take the .gitignore
// entry Advice names, write exactly that, and the child must flip to Ignored.
//
// The regression it guards is a warning that cannot be silenced by obeying
// it. A child registered as "./child" used to yield `/./child/`, a pattern
// git does not honour, so doctor warned forever. Any path form a user can
// type into `project add` must therefore round-trip.
func TestAdviceEntryActuallyIgnores(t *testing.T) {
	requireGit(t)
	// The nested form is here because the first version of this table held
	// only flat paths, and a second bug of the very same shape survived it
	// (T-051 review R1/R5). The root-denoting forms are *not* here: for those
	// the correct behaviour is to never reach advice at all, which is
	// TestIsRepoRoot's job, not a round-trip this test could assert.
	for _, relPath := range []string{"child", "./child", "child/", "apps/frontend"} {
		t.Run(relPath, func(t *testing.T) {
			root := t.TempDir()
			gitInit(t, root)
			childDir := filepath.Join(root, filepath.FromSlash(path.Clean(relPath)))
			if err := os.MkdirAll(childDir, 0o755); err != nil {
				t.Fatal(err)
			}
			gitInit(t, childDir)

			st := ChildState(root, relPath)
			if st != Stageable {
				t.Fatalf("ChildState(%q) = %v, want Stageable", relPath, st)
			}

			// The entry is the text between the quotes in the sentence.
			adv := st.Advice(relPath)
			parts := strings.Split(adv, `"`)
			if len(parts) < 2 {
				t.Fatalf("Advice(%q) = %q, want a quoted .gitignore entry", relPath, adv)
			}
			entry := parts[1]

			ignore := filepath.Join(root, ".gitignore")
			if err := os.WriteFile(ignore, []byte(entry+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := ChildState(root, relPath); got != Ignored {
				t.Errorf("after writing the advised entry %q to .gitignore, ChildState(%q) = %v, want Ignored\nadvice was: %s",
					entry, relPath, got, adv)
			}
		})
	}
}

// TestIsRepoRoot (T-051 review R1) pins the gate that keeps the overarching
// repository from being reported as a nested child of itself. The raw-string
// comparison it replaced accepted "." only, so a child registered as "./"
// reached the stageable check and the repo root was told to gitignore itself.
func TestIsRepoRoot(t *testing.T) {
	roots := []string{".", "./", "sub/..", "./.", "a/../"}
	children := []string{"child", "./child", "child/", "apps/frontend", "..", "../sibling"}

	for _, p := range roots {
		if !IsRepoRoot(p) {
			t.Errorf("IsRepoRoot(%q) = false, want true (denotes the repo itself)", p)
		}
	}
	for _, p := range children {
		if IsRepoRoot(p) {
			t.Errorf("IsRepoRoot(%q) = true, want false (not the repo root)", p)
		}
	}
}
