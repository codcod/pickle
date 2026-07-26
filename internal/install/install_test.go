package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
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
	mustExist(t, filepath.Join(root, "tickets/NOTES.md"))
	mustExist(t, filepath.Join(root, "tickets/README.md"))
	mustExist(t, filepath.Join(root, "tickets/1-to-do/.gitkeep"))
	mustExist(t, filepath.Join(root, "tickets/7-dropped/.gitkeep"))
	mustExist(t, filepath.Join(root, "pickle.toml"))

	// Claude view symlink resolves to the .agents skill dir.
	link := filepath.Join(root, ".claude/skills/ticket-flow")
	if target, err := os.Readlink(link); err != nil || target != "../../.agents/skills/ticket-flow" {
		t.Errorf("claude symlink = %q, %v", target, err)
	}

	// Markers injected into both files.
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		b, _ := os.ReadFile(filepath.Join(root, f))
		if !strings.Contains(string(b), MarkerBegin) {
			t.Errorf("%s missing marker", f)
		}
	}
	// The board is a fresh zero-ticket render (T-044): byte-identical to
	// board.Render on the empty tree, up to the Last-updated date.
	boardBytes, _ := os.ReadFile(filepath.Join(root, "tickets/BOARD.md"))
	cfgForRender, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	want := board.Render(nil, cfgForRender, "")
	if board.NormalizeLastUpdated(string(boardBytes)) != board.NormalizeLastUpdated(want) {
		t.Errorf("BOARD.md is not a zero-ticket render:\n%s", boardBytes)
	}
	if !strings.Contains(string(boardBytes), "### demo") {
		t.Error("BOARD.md missing child sub-group")
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
	notesPath := filepath.Join(root, "tickets/NOTES.md")
	os.WriteFile(notesPath, []byte("my notes"), 0o644)

	res, err := Run(payload, root, "v2", opts)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(notesPath); string(b) != "my notes" {
		t.Error("re-run clobbered tickets/NOTES.md")
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

// The regression test for T-018: a refresh must not cost the user their
// pickle.toml comments.
func TestUpgradePreservesConfigComments(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: "."}); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, config.FileName)
	handWritten := `# Why this project is configured the way it is.
# A second line of hard-won rationale.

payload_version = "v1"

[commit]
overarching_auto = true
child_publish_gated = true

# The sole child, deliberately.
[[project]]
name = "demo"
path = "."
`
	if err := os.WriteFile(cfgPath, []byte(handWritten), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{
		"# Why this project is configured the way it is.",
		"# A second line of hard-won rationale.",
		"# The sole child, deliberately.",
	} {
		if !strings.Contains(string(got), comment) {
			t.Errorf("upgrade destroyed comment %q\n--- got ---\n%s", comment, got)
		}
	}
	if !strings.Contains(string(got), `payload_version = "v2"`) {
		t.Errorf("payload_version not stamped:\n%s", got)
	}
	// Exactly one line may differ from what the user wrote.
	before, after := strings.Split(handWritten, "\n"), strings.Split(string(got), "\n")
	if len(before) != len(after) {
		t.Fatalf("line count changed: %d -> %d", len(before), len(after))
	}
	diffs := 0
	for i := range before {
		if before[i] != after[i] {
			diffs++
		}
	}
	if diffs != 1 {
		t.Errorf("upgrade changed %d lines, want 1", diffs)
	}
}

func TestUpgradeInsertsMissingPayloadVersion(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: "."}); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, config.FileName)
	noVersion := `# no payload_version anywhere in this file

[commit]
overarching_auto = true
child_publish_gated = true

[[project]]
name = "demo"
path = "."
`
	if err := os.WriteFile(cfgPath, []byte(noVersion), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(payload, root, "v3"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "# no payload_version anywhere in this file") {
		t.Errorf("comment lost when inserting the key:\n%s", got)
	}
	if !strings.Contains(string(got), `payload_version = "v3"`) {
		t.Errorf("payload_version not inserted:\n%s", got)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load after insert: %v", err)
	}
	if cfg.PayloadVersion != "v3" {
		t.Errorf("payload_version = %q, want v3", cfg.PayloadVersion)
	}
}

func TestMarkerBlockRendersChildrenFromConfig(t *testing.T) {
	cfg := &config.Config{
		Commit: config.CommitPolicy{OverarchingAuto: true, ChildPublishGated: true},
		Projects: []config.Project{{
			Name: "alpha", Path: ".", Build: "just build", Test: "just test", Lint: "just lint",
			BranchPrefix: "feat/", WIPInDevelopment: 1, WIPInReview: 1,
		}, {
			Name: "beta", Path: "sub", Build: "make",
			BranchPrefix: "ticket/", WIPInDevelopment: 3, WIPInReview: 2,
		}},
	}
	block := markerBlock(cfg)

	for _, want := range []string{
		"- `alpha`: build `just build` · test `just test` · lint `just lint`",
		"- `beta`: build `make`",
		"- `alpha`: `feat/T-NNN-<slug>`",
		"- `beta`: `ticket/T-NNN-<slug>`",
		"- `alpha`: `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1",
		"- `beta`: `3-in-development/` ≤ 3 · `4-in-review/` ≤ 2",
		"Registered child-projects: `alpha`, `beta`.",
		"publish-gated",
		"may be committed automatically",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("marker block missing %q\n--- block ---\n%s", want, block)
		}
	}
	// The literal placeholder must never reach a reader again.
	if strings.Contains(block, "<branch_prefix>") {
		t.Error("marker block still renders the literal <branch_prefix>")
	}
	// beta defines no test/lint, so they must not be invented.
	if strings.Contains(block, "- `beta`: build `make` · test") {
		t.Error("marker block rendered a command the child does not define")
	}
}

func TestMarkerBlockRendersCommitPolicyAndOmitsEmptyCommands(t *testing.T) {
	cfg := &config.Config{
		Commit: config.CommitPolicy{OverarchingAuto: false, ChildPublishGated: false},
		Projects: []config.Project{{
			Name: "solo", Path: ".", BranchPrefix: "feat/",
			WIPInDevelopment: 1, WIPInReview: 1,
		}},
	}
	block := markerBlock(cfg)

	if !strings.Contains(block, "**not publish-gated**") {
		t.Errorf("ungated policy not rendered:\n%s", block)
	}
	if !strings.Contains(block, "only when the\n  user asks") {
		t.Errorf("non-auto overarching policy not rendered:\n%s", block)
	}
	if strings.Contains(block, "**Commands**") {
		t.Errorf("commands bullet rendered for a child with no commands:\n%s", block)
	}
	// WIP and branch bullets are always derivable, so they must still be there.
	if !strings.Contains(block, "- `solo`: `feat/T-NNN-<slug>`") {
		t.Errorf("branch bullet missing:\n%s", block)
	}
}

// TestMarkerBlockDefaultsToTheCautiousCommitPolicy is the regression test for a
// config with no [commit] table at all. Because the policy is two booleans, an
// omitted table used to decode to false/false and render as "not publish-gated"
// — telling the agent it may push a child-project in a project whose author
// never made that choice, while the skill installed beside it said the opposite.
func TestMarkerBlockDefaultsToTheCautiousCommitPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)
	const noCommitTable = `payload_version = "v1"

[[project]]
name = "solo"
path = "."
`
	if err := os.WriteFile(path, []byte(noCommitTable), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	block := markerBlock(cfg)

	if !strings.Contains(block, "Child-projects are **publish-gated**") {
		t.Errorf("an omitted [commit] table rendered a non-gated policy:\n%s", block)
	}
	if strings.Contains(block, "**not publish-gated**") {
		t.Errorf("an omitted [commit] table rendered 'not publish-gated':\n%s", block)
	}
	if !strings.Contains(block, "may be committed automatically") {
		t.Errorf("an omitted [commit] table rendered a non-auto bookkeeping policy:\n%s", block)
	}

	// An explicit false must still survive — the default may not swallow it.
	explicit := strings.Replace(noCommitTable, "\n[[project]]",
		"\n[commit]\noverarching_auto = false\nchild_publish_gated = false\n\n[[project]]", 1)
	if err := os.WriteFile(path, []byte(explicit), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = config.Load(path)
	if err != nil {
		t.Fatalf("load explicit: %v", err)
	}
	if !strings.Contains(markerBlock(cfg), "**not publish-gated**") {
		t.Errorf("an explicit child_publish_gated = false was overridden by the default:\n%s", markerBlock(cfg))
	}

	// A [commit] table carrying only one of the two keys is the shape the
	// per-key defaulting exists for: the key that is present must be honoured
	// and the one that is absent must still fall back to the cautious default.
	// Defaulting on the table instead of on each key passes every assertion
	// above and fails here.
	partial := []struct {
		name    string
		table   string
		want    string
		notWant string
	}{{
		name:    "only overarching_auto",
		table:   "\n[commit]\noverarching_auto = true\n\n[[project]]",
		want:    "Child-projects are **publish-gated**",
		notWant: "**not publish-gated**",
	}, {
		name:    "only child_publish_gated",
		table:   "\n[commit]\nchild_publish_gated = false\n\n[[project]]",
		want:    "may be committed automatically",
		notWant: "only when the\n  user asks",
	}}
	for _, p := range partial {
		t.Run(p.name, func(t *testing.T) {
			src := strings.Replace(noCommitTable, "\n[[project]]", p.table, 1)
			if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			block := markerBlock(cfg)
			if !strings.Contains(block, p.want) {
				t.Errorf("a partial [commit] table did not default the absent key (want %q):\n%s", p.want, block)
			}
			if strings.Contains(block, p.notWant) {
				t.Errorf("a partial [commit] table rendered %q:\n%s", p.notWant, block)
			}
		})
	}
}

// TestMarkerBlockGolden pins the whole rendered block. The block is the agent's
// primary instruction file and every project's AGENTS.md carries a copy, so any
// wording change should be a deliberate, reviewable diff rather than something
// the substring assertions above quietly let through.
func TestMarkerBlockGolden(t *testing.T) {
	cfg := &config.Config{
		Commit: config.CommitPolicy{OverarchingAuto: true, ChildPublishGated: true},
		Projects: []config.Project{{
			Name: "alpha", Path: ".", Build: "just build", Test: "just test",
			Lint: "just lint", Docs: "just docs",
			BranchPrefix: "feat/", WIPInDevelopment: 1, WIPInReview: 1,
		}, {
			Name: "beta", Path: "sub", Build: "make",
			BranchPrefix: "ticket/", WIPInDevelopment: 3, WIPInReview: 2,
		}},
	}
	got := markerBlock(cfg)

	golden := filepath.Join("testdata", "markerblock.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden updated")
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (regenerate with UPDATE_GOLDEN=1 go test ./internal/install/): %v", err)
	}
	if got != string(want) {
		t.Errorf("marker block differs from %s.\nRegenerate with UPDATE_GOLDEN=1 go test ./internal/install/ if intended.\n--- got ---\n%s", golden, got)
	}
}

// TestPayloadDispositionVocabulary pins the two decisions behind T-036, not the
// prose that expresses them.
//
//  1. The payload must not tell a reviewer that a fix on the branch under review
//     is forbidden. The old protocol said "never inline drift", which made a
//     new ticket the only legal move for any non-blocking finding and left the
//     shipped flow minting tickets faster than any project retires them.
//  2. The four dispositions are defined in exactly one place. Restating a list
//     in a second file is how this payload has drifted before, so the two
//     dispositions that exist only as vocabulary — "fixed inline" and "folded" —
//     must appear in the rules and nowhere else. The other two ("new ticket",
//     "noted") are ordinary English that legitimately occurs in running prose,
//     so asserting on them would fail for reasons unrelated to drift.
func TestPayloadDispositionVocabulary(t *testing.T) {
	const rules = "skill/resources/tickets-README.md"
	// The four payload files a reviewer or implementer actually reads. Note
	// SKILL.md is at skill/SKILL.md, not under skill/resources/.
	files := []string{
		rules,
		"skill/resources/review-protocol.md",
		"skill/SKILL.md",
		"skill/resources/TEMPLATE.md",
	}

	body := make(map[string]string, len(files))
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(payloadRoot(), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		body[rel] = string(b)
	}

	// 1. The prohibition is gone for good.
	for _, rel := range files {
		if strings.Contains(body[rel], "never inline drift") {
			t.Errorf("%s still forbids inline fixes (%q): a bounded, recorded inline fix is now a legal disposition (rules §5)",
				rel, "never inline drift")
		}
	}

	// 2. The vocabulary lives in the rules and is referenced, never restated.
	for _, token := range []string{"fixed inline", "folded"} {
		if !strings.Contains(body[rules], token) {
			t.Errorf("%s does not define the %q disposition; the rules are its single source of truth", rules, token)
		}
		for _, rel := range files {
			if rel == rules {
				continue
			}
			if strings.Contains(body[rel], token) {
				t.Errorf("%s restates the %q disposition; reference the rules §5 instead so the list cannot drift", rel, token)
			}
		}
	}
}

// TestVerifyStampedVersion covers the guard that stops Upgrade reporting a
// version it did not actually put on disk. The config writer's own parse-back
// check makes this unreachable today; it exists so that a future regression
// there surfaces as an error instead of a false success message.
func TestVerifyStampedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)
	write := func(version string) {
		body := "payload_version = \"" + version + "\"\n\n[[project]]\nname = \"solo\"\npath = \".\"\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("v2")
	if err := verifyStampedVersion(path, "v2"); err != nil {
		t.Errorf("matching version reported an error: %v", err)
	}

	write("v1")
	err := verifyStampedVersion(path, "v2")
	if err == nil {
		t.Fatal("a stale payload_version was accepted as stamped")
	}
	if !strings.Contains(err.Error(), "still reads payload_version") {
		t.Errorf("error = %q, want it to name the stale version", err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := verifyStampedVersion(path, "v2"); err == nil {
		t.Error("a missing config was accepted as stamped")
	}
}
