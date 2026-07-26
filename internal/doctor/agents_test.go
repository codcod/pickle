package doctor

// T-009: the pi agent scaffolds are pickle-owned — doctor warns (never errors)
// when a present one drifts from the embedded copy, and says nothing when the
// scaffolds were simply never installed. opencode.jsonc is user-owned, so it
// gets no checks at all.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/install"
)

// piFixture is installFixture plus the pi (and opencode) scaffolds.
func piFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	_, err := install.Run(os.DirFS(payloadRoot()), root, "test-ver", install.Options{
		ProjectName: "demo",
		ProjectPath: ".",
		Agents:      install.Agents{Claude: true, Opencode: true, Pi: true},
	})
	if err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckAgentScaffoldsCurrent(t *testing.T) {
	root := piFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 || len(res.Warnings) != 0 {
		t.Fatalf("healthy scaffolds flagged: errors=%v warnings=%v", res.Errors, res.Warnings)
	}
	if !strings.Contains(strings.Join(res.Passed, "\n"), "agent scaffold current") {
		t.Errorf("expected agent-scaffold pass entries, got: %v", res.Passed)
	}
}

func TestCheckAgentScaffoldDriftWarns(t *testing.T) {
	root := piFixture(t)
	drifted := filepath.Join(root, filepath.FromSlash(install.PiScaffolds[0].Installed))
	if err := os.WriteFile(drifted, []byte("// drifted"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("scaffold drift must be a warning, not an error: %v", res.Errors)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, install.PiScaffolds[0].Installed) ||
		!strings.Contains(joined, "pickle upgrade") {
		t.Errorf("expected a drift warning naming the file and suggesting upgrade, got: %v", res.Warnings)
	}

	// A user-modified opencode.jsonc is user-owned: never a finding.
	if err := os.WriteFile(filepath.Join(root, install.OpencodeConfigFile), []byte("// mine\n{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res = Check(root, "test-ver", os.DirFS(payloadRoot()))
	for _, list := range [][]string{res.Errors, res.Warnings} {
		if strings.Contains(strings.Join(list, "\n"), "opencode") {
			t.Errorf("doctor flagged the user-owned opencode.jsonc: %v", list)
		}
	}
}

func TestCheckAgentScaffoldsAbsentIsClean(t *testing.T) {
	root := installFixture(t) // claude-only: no pi scaffolds installed
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	joined := strings.Join(append(res.Errors, res.Warnings...), "\n")
	if strings.Contains(joined, ".pi/") {
		t.Errorf("absent pi scaffolds produced findings: %v", joined)
	}
}
