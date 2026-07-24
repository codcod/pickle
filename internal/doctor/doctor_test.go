package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/install"
)

// payloadRoot mirrors install_test.go: the dir whose os.DirFS exposes
// "skill/SKILL.md" at the same paths the embedded payload uses (the repo root).
func payloadRoot() string { return filepath.Join("..", "..") }

// installFixture lays a real, healthy pickle install into a temp dir and gives
// the sole child ("."): a .git dir so it reads as a git repository.
func installFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	_, err := install.Run(os.DirFS(payloadRoot()), root, "test-ver", install.Options{
		ProjectName: "demo",
		ProjectPath: ".",
		Claude:      true,
	})
	if err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	return root
}

func hasErrContaining(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

func TestCheckHealthyInstall(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver")
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors on a healthy install, got: %v", res.Errors)
	}
	if len(res.Passed) == 0 {
		t.Error("expected some passed checks")
	}
}

func TestCheckBrokenArtifacts(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(t *testing.T, root string)
		errSub string
	}{
		{
			name:   "missing skill",
			break_: func(t *testing.T, root string) { rmAll(t, filepath.Join(root, ".agents")) },
			errSub: "skill:",
		},
		{
			name:   "child not a git repo",
			break_: func(t *testing.T, root string) { rmAll(t, filepath.Join(root, ".git")) },
			errSub: "is not a git repository",
		},
		{
			name: "stripped AGENTS.md markers",
			break_: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# nothing here\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			errSub: "AGENTS.md",
		},
		{
			name: "broken claude symlink",
			break_: func(t *testing.T, root string) {
				link := filepath.Join(root, filepath.FromSlash(install.ClaudeSkillLink))
				rmAll(t, link)
				if err := os.Symlink("nowhere", link); err != nil {
					t.Fatal(err)
				}
			},
			errSub: "claude:",
		},
		{
			name: "corrupt pickle.toml",
			break_: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "pickle.toml"), []byte("this = = not toml\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			errSub: "pickle.toml:",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := installFixture(t)
			tc.break_(t, root)
			res := Check(root, "test-ver")
			if len(res.Errors) == 0 {
				t.Fatalf("expected an error, got none (passed: %v)", res.Passed)
			}
			if !hasErrContaining(res.Errors, tc.errSub) {
				t.Errorf("expected an error containing %q, got: %v", tc.errSub, res.Errors)
			}
		})
	}
}

func TestCheckVersionDriftWarns(t *testing.T) {
	root := installFixture(t) // installed at payload_version "test-ver"
	res := Check(root, "v9.9.9")
	if len(res.Errors) != 0 {
		t.Fatalf("version drift must not be an error: %v", res.Errors)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected a payload-drift warning")
	}
	if !strings.Contains(strings.Join(res.Warnings, "\n"), "pickle upgrade") {
		t.Errorf("warning should suggest upgrade, got: %v", res.Warnings)
	}
}

func rmAll(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}
