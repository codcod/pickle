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

// TestCheckReportsFlowName (T-073): checkConfig reports the effective flow
// name as a passed check, both when pickle.toml has no explicit flow key
// (falls back to the default, "brine") and when one is set explicitly.
func TestCheckReportsFlowName(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Passed, "flow: brine") {
		t.Errorf("expected a passed entry reporting the default flow name, got: %v", res.Passed)
	}

	cfgPath := filepath.Join(root, config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Flow = "brine"
	if err := cfg.Save(""); err != nil {
		t.Fatalf("Save: %v", err)
	}
	res = Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Passed, "flow: brine") {
		t.Errorf("expected a passed entry reporting the explicit flow name, got: %v", res.Passed)
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

// TestCheckMarkersCurrentIsSilent is the false-positive guard: a freshly
// installed, unmodified block must never earn a drift warning, and doctor -v
// should show the check ran (a passed entry naming the block current).
func TestCheckMarkersCurrentIsSilent(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if hasErrContaining(res.Warnings, "markers:") {
		t.Errorf("a freshly installed, unmodified block must not warn: %v", res.Warnings)
	}
	if !hasErrContaining(res.Passed, "AGENTS.md marker block current") {
		t.Errorf("expected a passed entry naming the block current, got: %v", res.Passed)
	}
}

// TestCheckMarkersDriftInsideBlockWarns is the detect-half regression: a line
// changed inside the markers must be reported as a warning (never an error),
// naming the file.
func TestCheckMarkersDriftInsideBlockWarns(t *testing.T) {
	root := installFixture(t)
	path := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "<!-- pickle:begin -->\n", "<!-- pickle:begin -->\nEDITED BY HAND\n", 1)
	if edited == string(data) {
		t.Fatal("fixture setup: marker begin not found to edit")
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("marker drift must not be an error: %v", res.Errors)
	}
	if !hasErrContaining(res.Warnings, "AGENTS.md block differs from what pickle.toml renders") {
		t.Errorf("expected a drift warning naming AGENTS.md, got: %v", res.Warnings)
	}
}

// TestCheckMarkersDriftOutsideBlockIsSilent is the other half of the
// false-positive guard: content appended after the closing marker (the
// project's own house rules, say) must never be mistaken for drift.
func TestCheckMarkersDriftOutsideBlockIsSilent(t *testing.T) {
	root := installFixture(t)
	path := filepath.Join(root, "AGENTS.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n## House rules\n\nThis project also requires X.\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if hasErrContaining(res.Warnings, "markers:") {
		t.Errorf("content outside the markers must never warn as drift: %v", res.Warnings)
	}
}

// TestCheckMarkersUnparseableConfigSkipsDrift: when pickle.toml itself fails to
// parse, checkMarkers has no canonical block to compare against — it must
// fall back to presence-only, not claim a difference that cannot be computed.
func TestCheckMarkersUnparseableConfigSkipsDrift(t *testing.T) {
	root := installFixture(t)
	if err := os.WriteFile(filepath.Join(root, "pickle.toml"), []byte("this = = not toml\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, "pickle.toml:") {
		t.Fatalf("expected the existing pickle.toml parse error, got: %v", res.Errors)
	}
	if hasErrContaining(res.Warnings, "markers:") {
		t.Errorf("an unparseable pickle.toml must not produce a marker drift warning: %v", res.Warnings)
	}
	if !hasErrContaining(res.Passed, "AGENTS.md marker block present") {
		t.Errorf("presence check must still hold with no config: %v", res.Passed)
	}
}
