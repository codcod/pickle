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
		if !strings.Contains(string(b), MarkerBegin) {
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
	if n := strings.Count(string(agents), MarkerBegin); n != 1 {
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

func TestUpgrade(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Claude: true}

	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	// Stale file in a real (non-symlinked) skill dir must not linger after upgrade.
	junk := filepath.Join(root, filepath.FromSlash(SkillDir), "junk.md")
	if err := os.WriteFile(junk, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if len(res.Created) == 0 {
		t.Error("expected created/updated entries")
	}

	cfg, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.PayloadVersion != "v2" {
		t.Errorf("payload_version = %q, want v2", cfg.PayloadVersion)
	}
	mustExist(t, filepath.Join(root, ".agents/skills/ticket-flow/SKILL.md"))
	if _, err := os.Stat(junk); !os.IsNotExist(err) {
		t.Error("stale file in skill dir survived upgrade")
	}
	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(string(agents), MarkerBegin) {
		t.Error("AGENTS.md missing marker after upgrade")
	}

	// tickets/ must never be touched by upgrade.
	ticketPath := filepath.Join(root, "tickets/1-to-do/T-009-x.md")
	if err := os.WriteFile(ticketPath, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("second Upgrade: %v", err)
	}
	if b, _ := os.ReadFile(ticketPath); string(b) != "keep me" {
		t.Error("upgrade touched tickets/")
	}

	// Idempotent: re-running at the same version is still a clean, error-free run.
	res2, err := Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatalf("idempotent Upgrade: %v", err)
	}
	var sawSameVersion bool
	for _, s := range res2.Skipped {
		if strings.Contains(s, "already at v2") {
			sawSameVersion = true
		}
	}
	if !sawSameVersion {
		t.Error("expected a same-version skip entry on idempotent re-run")
	}
}

func TestUpgradeSelfHostSymlinkGuard(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Claude: true}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	// Replace the real skill dir with a symlink to an external target, as a
	// self-hosted checkout would have.
	skillPath := filepath.Join(root, filepath.FromSlash(SkillDir))
	if err := os.RemoveAll(skillPath); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "SKILL.md"), []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, skillPath); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	fi, err := os.Lstat(skillPath)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("skill dir symlink was not preserved by upgrade")
	}
	if target, err := os.Readlink(skillPath); err != nil || target != external {
		t.Errorf("symlink target = %q, %v", target, err)
	}
	if _, err := os.Stat(filepath.Join(external, "SKILL.md")); err != nil {
		t.Error("external symlink target was modified/removed by upgrade")
	}
}

func TestUninstall(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Claude: true}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	res, err := Uninstall(root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Removed) == 0 {
		t.Error("expected removed entries")
	}

	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(SkillDir))); !os.IsNotExist(err) {
		t.Error("skill dir still present after uninstall")
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))); !os.IsNotExist(err) {
		t.Error("claude skill symlink still present after uninstall")
	}
	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if strings.Contains(string(agents), MarkerBegin) {
		t.Error("AGENTS.md still has a marker pair after uninstall")
	}
	mustExist(t, filepath.Join(root, "tickets"))
	mustExist(t, filepath.Join(root, "tickets/BOARD.md"))
	mustExist(t, filepath.Join(root, "pickle.toml"))
}

func TestUninstallDryRun(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Claude: true}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	res, err := Uninstall(root, UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Uninstall dry-run: %v", err)
	}
	if len(res.Removed) == 0 {
		t.Error("expected dry-run to report removals")
	}

	mustExist(t, filepath.Join(root, filepath.FromSlash(SkillDir)))
	mustExist(t, filepath.Join(root, filepath.FromSlash(ClaudeSkillLink)))
	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(string(agents), MarkerBegin) {
		t.Error("dry-run uninstall mutated AGENTS.md")
	}
}

func TestUninstallSelfHostSymlinkGuard(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, filepath.FromSlash(SkillDir))
	os.MkdirAll(filepath.Dir(skillPath), 0o755)
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "SKILL.md"), []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, skillPath); err != nil {
		t.Fatal(err)
	}

	if _, err := Uninstall(root, UninstallOptions{}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := os.Lstat(skillPath); !os.IsNotExist(err) {
		t.Error("skill symlink still present after uninstall")
	}
	if _, err := os.Stat(filepath.Join(external, "SKILL.md")); err != nil {
		t.Error("external symlink target was removed by uninstall")
	}
}
