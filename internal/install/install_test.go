package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/config"
)

// payloadRoot returns the dir whose os.DirFS exposes "skill/SKILL.md" etc. at the
// same paths the binary's embedded payloadFS uses (the repo root, ../..).
func payloadRoot() string { return filepath.Join("..", "..") }

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

func TestRunProducesInstall(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())

	res, err := Run(payload, root, "test-ver", Options{ProjectName: "demo", ProjectPath: ".", Claude: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Created) == 0 {
		t.Error("expected created entries")
	}

	mustExist(t, filepath.Join(root, ".agents/skills/ticket-flow/SKILL.md"))
	mustExist(t, filepath.Join(root, ".agents/skills/ticket-flow/resources/TEMPLATE.md"))
	mustExist(t, filepath.Join(root, "tickets/BOARD.md"))
	mustExist(t, filepath.Join(root, "tickets/README.md"))
	mustExist(t, filepath.Join(root, "tickets/1-to-do/.gitkeep"))
	mustExist(t, filepath.Join(root, "tickets/7-dropped/.gitkeep"))
	mustExist(t, filepath.Join(root, "pickle.toml"))

	// Claude view symlink resolves to the .agents skill dir.
	link := filepath.Join(root, ".claude/skills/ticket-flow")
	if target, err := os.Readlink(link); err != nil || target != "../../.agents/skills/ticket-flow" {
		t.Errorf("claude symlink = %q, %v", target, err)
	}

	// Markers injected into both files; BOARD.md carries the child name + a date.
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		b, _ := os.ReadFile(filepath.Join(root, f))
		if !strings.Contains(string(b), markerBegin) {
			t.Errorf("%s missing marker", f)
		}
	}
	board, _ := os.ReadFile(filepath.Join(root, "tickets/BOARD.md"))
	if strings.Contains(string(board), "<child-project>") || strings.Contains(string(board), "<YYYY-MM-DD>") {
		t.Error("BOARD.md placeholders not substituted")
	}
	if !strings.Contains(string(board), "demo") {
		t.Error("BOARD.md missing child name")
	}

	// Config registers the child with the stamped payload version.
	cfg, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.PayloadVersion != "test-ver" {
		t.Errorf("payload_version = %q", cfg.PayloadVersion)
	}
	if _, ok := cfg.Project("demo"); !ok {
		t.Error("child 'demo' not registered")
	}

	// A correct install is board-audit-clean.
	if a := audit.Audit(cfg.Root(), cfg); len(a.Errors) > 0 {
		t.Errorf("post-install audit errors: %v", a.Errors)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Claude: true}

	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}
	// Seed instance data that a re-run must preserve.
	ticketPath := filepath.Join(root, "tickets/1-to-do/T-009-x.md")
	os.WriteFile(ticketPath, []byte("keep me"), 0o644)

	res, err := Run(payload, root, "v2", opts)
	if err != nil {
		t.Fatal(err)
	}

	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if n := strings.Count(string(agents), markerBegin); n != 1 {
		t.Errorf("AGENTS.md marker count = %d, want 1", n)
	}
	if b, _ := os.ReadFile(ticketPath); string(b) != "keep me" {
		t.Error("re-run clobbered an existing ticket")
	}
	// pickle.toml preserved (still v1, not re-stamped to v2 — that's upgrade's job).
	cfg, _ := config.Load(filepath.Join(root, "pickle.toml"))
	if cfg.PayloadVersion != "v1" {
		t.Errorf("re-run changed pickle.toml payload_version to %q", cfg.PayloadVersion)
	}
	// Second run should report skips, not creations, for the preserved files.
	var skippedConfig bool
	for _, s := range res.Skipped {
		if strings.Contains(s, config.FileName) {
			skippedConfig = true
		}
	}
	if !skippedConfig {
		t.Error("expected pickle.toml to be skipped on re-run")
	}
}

func TestSelfHostSymlinkGuard(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, ".agents/skills/ticket-flow")
	os.MkdirAll(filepath.Dir(skillPath), 0o755)
	if err := os.Symlink("../../skill", skillPath); err != nil {
		t.Fatal(err)
	}
	payload := os.DirFS(payloadRoot())

	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Claude: true}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(skillPath)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("self-host symlink was not preserved")
	}
}
