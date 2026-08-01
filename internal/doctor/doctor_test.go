package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
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
		Agents:      install.Agents{Claude: true},
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
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
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
			res := Check(root, "test-ver", os.DirFS(payloadRoot()))
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
	res := Check(root, "v9.9.9", os.DirFS(payloadRoot()))
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

// T-026: doctor must not send the user to a command that is going to fail.
// When the installed pickle.toml has a shape the in-place writer refuses (a
// quoted key here; the same holds for a multi-line-string or array value on
// payload_version itself), the warning must name the real remedy — editing
// the file by hand — and must not say "run `pickle upgrade`".
func TestCheckVersionDriftUnstampableSuggestsHandEdit(t *testing.T) {
	root := installFixture(t) // installed at payload_version "test-ver"
	cfgPath := filepath.Join(root, config.FileName)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// A multi-line-string value on payload_version's own key: it decodes into
	// the string field fine (so config.Load succeeds and the version really
	// does "differ"), but the in-place writer still refuses to rewrite it —
	// D3, confirmed at refinement — so this exercises the branch where a
	// legitimately-loadable file cannot be stamped.
	wedged := strings.Replace(string(data), `payload_version = "test-ver"`, "payload_version = \"\"\"\ntest-ver\n\"\"\"", 1)
	if wedged == string(data) {
		t.Fatal("fixture setup: payload_version line not found to wedge")
	}
	if err := os.WriteFile(cfgPath, []byte(wedged), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "v9.9.9", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("an unstampable file must still be a warning, not an error: %v", res.Errors)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "payload_version by hand") {
		t.Errorf("warning should say to edit payload_version by hand, got: %v", res.Warnings)
	}
	if strings.Contains(joined, "run `pickle upgrade`") {
		t.Errorf("warning must not recommend running `pickle upgrade` when it cannot succeed, got: %v", res.Warnings)
	}
}

func rmAll(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}
