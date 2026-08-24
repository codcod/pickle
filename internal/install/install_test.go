package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/testutil"
)

// payloadRoot returns the dir whose os.DirFS exposes "skill/SKILL.md" etc. at the
// same paths the binary's embedded payloadFS uses (the repo root). A thin
// wrapper over testutil.RepoRoot() (T-042 item 3), kept because this file has
// too many call sites to update individually in one reviewable sitting.
func payloadRoot() string { return testutil.RepoRoot() }

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

// firstLine returns s up to (not including) the first newline, for readable
// test failure messages.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func TestRunProducesInstall(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())

	res, err := Run(payload, root, "test-ver", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Created) == 0 {
		t.Error("expected created entries")
	}

	mustExist(t, filepath.Join(root, ".agents/skills/brine/SKILL.md"))
	mustExist(t, filepath.Join(root, ".agents/skills/brine/resources/TEMPLATE.md"))
	mustExist(t, filepath.Join(root, "tickets/BOARD.md"))
	mustExist(t, filepath.Join(root, "tickets/NOTES.md"))
	mustExist(t, filepath.Join(root, "tickets/README.md"))
	mustExist(t, filepath.Join(root, "tickets/1-to-do/.gitkeep"))
	mustExist(t, filepath.Join(root, "tickets/7-dropped/.gitkeep"))
	mustExist(t, filepath.Join(root, "pickle.toml"))

	// Claude view symlink resolves to the .agents skill dir.
	link := filepath.Join(root, ".claude/skills/brine")
	if target, err := os.Readlink(link); err != nil || target != "../../.agents/skills/brine" {
		t.Errorf("claude symlink = %q, %v", target, err)
	}

	// Markers injected into both files, each headed by the current on-disk
	// name (T-074 F1 regression: a freshly-created file must not fall back to
	// the pre-rename "Ticket flow", which would sit above a marker block
	// already headed "## Brine (start here)").
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		b, _ := os.ReadFile(filepath.Join(root, f))
		if !strings.Contains(string(b), MarkerBegin) {
			t.Errorf("%s missing marker", f)
		}
		// The expected H1 is spelled out, not built from MarkerTitle: asserting
		// against the constant the production code already uses would move with
		// any edit to it and so could never fail.
		if !strings.HasPrefix(string(b), "# Brine\n") {
			t.Errorf("%s H1 = %q, want %q", f, firstLine(string(b)), "# Brine")
		}
		if strings.Contains(string(b), "Ticket flow") {
			t.Errorf("%s still contains the pre-rename name \"Ticket flow\"", f)
		}
	}
	// The board is a fresh zero-ticket render (T-044): byte-identical to
	// board.Render on the empty tree, up to the Last-updated date.
	boardBytes, _ := os.ReadFile(filepath.Join(root, "tickets/BOARD.md"))
	cfgForRender, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	want := board.Render(flow.ForName(cfgForRender.FlowName()), nil, cfgForRender, "")
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
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}

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
	skillPath := filepath.Join(root, ".agents/skills/brine")
	os.MkdirAll(filepath.Dir(skillPath), 0o755)
	if err := os.Symlink("../../skill", skillPath); err != nil {
		t.Fatal(err)
	}
	payload := os.DirFS(payloadRoot())

	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
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
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}

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
	mustExist(t, filepath.Join(root, ".agents/skills/brine/SKILL.md"))
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
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
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

	// D4 (T-046): upgrade still stamps payload_version even when the skill
	// dir is a self-host link — it keeps refreshing everything else it owns
	// (marker block, pi scaffolds, hook shim), so the stamp truthfully
	// records "this tree was last upgraded by that binary"; and doctor no
	// longer reads the stamp in this mode (T-046), so the two can never
	// disagree about it.
	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.PayloadVersion != "v2" {
		t.Errorf("payload_version = %q, want %q (D4: upgrade keeps stamping over a self-host link)", cfg.PayloadVersion, "v2")
	}
}

func TestUninstall(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	res, err := Uninstall(payload, root, UninstallOptions{})
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
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	res, err := Uninstall(payload, root, UninstallOptions{DryRun: true})
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
	payload := os.DirFS(payloadRoot())
	skillPath := filepath.Join(root, filepath.FromSlash(SkillDir))
	os.MkdirAll(filepath.Dir(skillPath), 0o755)
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "SKILL.md"), []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, skillPath); err != nil {
		t.Fatal(err)
	}

	if _, err := Uninstall(payload, root, UninstallOptions{}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := os.Lstat(skillPath); !os.IsNotExist(err) {
		t.Error("skill symlink still present after uninstall")
	}
	if _, err := os.Stat(filepath.Join(external, "SKILL.md")); err != nil {
		t.Error("external symlink target was removed by uninstall")
	}
}

// legacyInstallFixture installs at the current names via Run, then renames the
// skill dir and re-creates the Claude view under the pre-T-074 legacy names —
// simulating an install made by a pickle older than the brine rename, without
// hand-rolling every other artifact Run also writes (pickle.toml, tickets/,
// markers). Returns the legacy skill dir and legacy Claude link paths.
func legacyInstallFixture(t *testing.T) (root, legacySkill, legacyClaudeLink string) {
	t.Helper()
	root = t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	newSkill := filepath.Join(root, filepath.FromSlash(SkillDir))
	legacySkill = filepath.Join(root, filepath.FromSlash(LegacySkillDir))
	if err := os.Rename(newSkill, legacySkill); err != nil {
		t.Fatal(err)
	}
	newClaudeLink := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	legacyClaudeLink = filepath.Join(root, filepath.FromSlash(LegacyClaudeSkillLink))
	if err := os.Remove(newClaudeLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../"+LegacySkillDir, legacyClaudeLink); err != nil {
		t.Fatal(err)
	}
	return root, legacySkill, legacyClaudeLink
}

// T-074: Upgrade sweeps a legacy real directory (and its Claude view) left by
// a pre-brine pickle, and leaves a normal current-name install in its place.
func TestUpgradeSweepsLegacyRealDirectory(t *testing.T) {
	root, legacySkill, legacyClaudeLink := legacyInstallFixture(t)
	payload := os.DirFS(payloadRoot())

	res, err := Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	var sweptDir, sweptLink bool
	for _, r := range res.Removed {
		if strings.HasPrefix(r, LegacySkillDir) {
			sweptDir = true
		}
		if strings.HasPrefix(r, LegacyClaudeSkillLink) {
			sweptLink = true
		}
	}
	if !sweptDir || !sweptLink {
		t.Errorf("Upgrade did not report sweeping both legacy paths: %v", res.Removed)
	}

	if _, err := os.Lstat(legacySkill); !os.IsNotExist(err) {
		t.Error("legacy skill dir still present after upgrade")
	}
	if _, err := os.Lstat(legacyClaudeLink); !os.IsNotExist(err) {
		t.Error("legacy claude symlink still present after upgrade")
	}
	mustExist(t, filepath.Join(root, filepath.FromSlash(SkillDir), "SKILL.md"))
	newClaudeLink := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if target, err := os.Readlink(newClaudeLink); err != nil || target != ClaudeSkillTarget {
		t.Errorf("new claude symlink = %q, %v", target, err)
	}
}

// T-074 decision 5: a legacy skill path that is itself a self-host symlink is
// re-linked at the new name, never deleted-and-recopied — deleting it and
// letting copyPayload write a real directory would silently turn a
// self-hosting checkout into an installed copy.
func TestUpgradeRelinksLegacySelfHostSymlink(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	newSkill := filepath.Join(root, filepath.FromSlash(SkillDir))
	if err := os.RemoveAll(newSkill); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "SKILL.md"), []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacySkill := filepath.Join(root, filepath.FromSlash(LegacySkillDir))
	if err := os.Symlink(external, legacySkill); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if _, err := os.Lstat(legacySkill); !os.IsNotExist(err) {
		t.Error("legacy skill symlink still present after upgrade")
	}
	fi, err := os.Lstat(newSkill)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("new-name skill path is not a symlink after upgrade")
	}
	if target, err := os.Readlink(newSkill); err != nil || target != external {
		t.Errorf("re-linked target = %q, %v", target, err)
	}
	if _, err := os.Stat(filepath.Join(external, "SKILL.md")); err != nil {
		t.Error("external symlink target was modified/removed by upgrade")
	}
}

// T-074: absent legacy paths are a no-op — an ordinary upgrade of a
// current-name install reports no legacy sweep activity.
func TestUpgradeNoLegacyIsNoOp(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}

	res, err := Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	for _, r := range res.Removed {
		if strings.Contains(r, "ticket-flow") {
			t.Errorf("unexpected legacy sweep activity on a tree with no legacy path: %v", res.Removed)
		}
	}
}

// T-074: a new binary can still fully remove an install made by an old one.
func TestUninstallSweepsLegacyPaths(t *testing.T) {
	root, legacySkill, legacyClaudeLink := legacyInstallFixture(t)
	payload := os.DirFS(payloadRoot())

	res, err := Uninstall(payload, root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Lstat(legacySkill); !os.IsNotExist(err) {
		t.Error("legacy skill dir still present after uninstall")
	}
	if _, err := os.Lstat(legacyClaudeLink); !os.IsNotExist(err) {
		t.Error("legacy claude symlink still present after uninstall")
	}
	var sweptDir bool
	for _, r := range res.Removed {
		if strings.HasPrefix(r, LegacySkillDir) {
			sweptDir = true
		}
	}
	if !sweptDir {
		t.Errorf("Uninstall did not report sweeping the legacy dir: %v", res.Removed)
	}
}

// T-074: --dry-run lists both legacy paths and mutates nothing.
func TestUninstallDryRunListsLegacyPathsAndMutatesNothing(t *testing.T) {
	root, legacySkill, legacyClaudeLink := legacyInstallFixture(t)
	payload := os.DirFS(payloadRoot())

	res, err := Uninstall(payload, root, UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Uninstall dry-run: %v", err)
	}
	var listedDir, listedLink bool
	for _, r := range res.Removed {
		if strings.HasPrefix(r, LegacySkillDir) {
			listedDir = true
		}
		if strings.HasPrefix(r, LegacyClaudeSkillLink) {
			listedLink = true
		}
	}
	if !listedDir || !listedLink {
		t.Errorf("dry-run did not list both legacy paths: %v", res.Removed)
	}
	mustExist(t, legacySkill)
	mustExist(t, legacyClaudeLink)
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
	// T-108: layout is also back-filled on this same upgrade, since the
	// hand-written config predates the key. That is an *inserted* line, not a
	// rewrite of an existing one — the sole child is at ".", so it infers
	// "in-tree".
	if !strings.Contains(string(got), `layout = "in-tree"`) {
		t.Errorf("layout not back-filled:\n%s", got)
	}
	// Exactly one line may be rewritten in place (payload_version), and exactly
	// one line may be newly inserted (layout) — every other original line
	// must survive byte-for-byte, in order.
	before, after := strings.Split(handWritten, "\n"), strings.Split(string(got), "\n")
	if len(after) != len(before)+1 {
		t.Fatalf("line count changed: %d -> %d, want %d (one inserted layout line)", len(before), len(after), len(before)+1)
	}
	afterWithoutLayout := make([]string, 0, len(after))
	for _, l := range after {
		if l != `layout = "in-tree"` {
			afterWithoutLayout = append(afterWithoutLayout, l)
		}
	}
	if len(afterWithoutLayout) != len(before) {
		t.Fatalf("expected exactly one inserted layout line, got %d extra line(s)", len(afterWithoutLayout)-len(before))
	}
	diffs := 0
	for i := range before {
		if before[i] != afterWithoutLayout[i] {
			diffs++
		}
	}
	if diffs != 1 {
		t.Errorf("upgrade changed %d lines (excluding the inserted layout line), want 1", diffs)
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

// hasEntry reports whether list contains want exactly. The labels carry
// suffixes like " (refreshed)", and the point of the tests below is *which*
// label was emitted, so this matches the whole rendered string rather than
// prefixing on the bare path — a prefix match would let "…/" satisfy an
// assertion meant for "…/ (refreshed)".
func hasEntry(list []string, want string) bool {
	for _, e := range list {
		if e == want {
			return true
		}
	}
	return false
}

// TestSkillDirLabelsOnInstall binds the created/refreshed/current vocabulary
// on the install path (T-013 items 2/8, and its review finding B2: before
// these tests, reverting all three labels to the old unconditional
// res.created(SkillDir+"/") failed no test at all).
func TestSkillDirLabelsOnInstall(t *testing.T) {
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", InTree: true, Agents: Agents{}}
	root := t.TempDir()

	fresh, err := Run(payload, root, "v1", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntry(fresh.Created, SkillDir+"/") {
		t.Errorf("fresh install: want %q in Created, got created=%v skipped=%v", SkillDir+"/", fresh.Created, fresh.Skipped)
	}

	// Re-run with nothing touched: the payload on disk is byte-identical, so
	// it must report current rather than claiming it created anything.
	again, err := Run(payload, root, "v1", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntry(again.Skipped, SkillDir+"/ (current)") {
		t.Errorf("unchanged re-run: want %q in Skipped, got created=%v skipped=%v", SkillDir+"/ (current)", again.Created, again.Skipped)
	}
	if hasEntry(again.Created, SkillDir+"/") {
		t.Error("unchanged re-run still reports the skill dir as created")
	}

	// Now make the on-disk copy differ, and it must say so.
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(SkillDir), "SKILL.md"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	drifted, err := Run(payload, root, "v1", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntry(drifted.Created, SkillDir+"/ (refreshed)") {
		t.Errorf("drifted re-run: want %q in Created, got created=%v skipped=%v", SkillDir+"/ (refreshed)", drifted.Created, drifted.Skipped)
	}
}

// TestSkillDirLabelsOnUpgrade is the same contract on the upgrade path, which
// reaches it through a different branch (Upgrade captures the diff itself,
// before its wipe, rather than letting copyPayload do it).
func TestSkillDirLabelsOnUpgrade(t *testing.T) {
	payload := os.DirFS(payloadRoot())
	root := t.TempDir()
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", InTree: true}); err != nil {
		t.Fatal(err)
	}

	unchanged, err := Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntry(unchanged.Skipped, SkillDir+"/ (current)") {
		t.Errorf("byte-identical upgrade: want %q in Skipped, got created=%v skipped=%v", SkillDir+"/ (current)", unchanged.Created, unchanged.Skipped)
	}

	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(SkillDir), "SKILL.md"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	drifted, err := Upgrade(payload, root, "v3")
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntry(drifted.Created, SkillDir+"/ (refreshed)") {
		t.Errorf("drifted upgrade: want %q in Created, got created=%v skipped=%v", SkillDir+"/ (refreshed)", drifted.Created, drifted.Skipped)
	}
}

// TestScaffoldTicketsLabelsSecondRun binds the third label this ticket added.
func TestScaffoldTicketsLabelsSecondRun(t *testing.T) {
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", InTree: true}
	root := t.TempDir()
	if _, err := Run(payload, root, "v1", opts); err != nil {
		t.Fatal(err)
	}
	again, err := Run(payload, root, "v1", opts)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, s := range again.Skipped {
		if strings.Contains(s, "already scaffolded") {
			found = true
		}
	}
	if !found {
		t.Errorf("re-run must report tickets/ as already scaffolded, got skipped=%v created=%v", again.Skipped, again.Created)
	}
}

// TestUpgradeAlwaysReplacesSkillDirWholesale is the T-013 review's B1
// regression test, and the reason the created/refreshed/current labels are
// allowed to exist at all.
//
// The skill directory is pickle-owned and documented as "replaced wholesale";
// upgrade's wipe-and-recopy is what makes that true. An earlier revision of
// this ticket gated that wipe on skillPayloadDiffers — which compares file
// *contents* — and so silently stopped repairing three kinds of tampering it
// cannot see. Each assertion below failed on that revision and passes on
// stock main.
func TestUpgradeAlwaysReplacesSkillDirWholesale(t *testing.T) {
	payload := os.DirFS(payloadRoot())
	root := t.TempDir()
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", InTree: true}); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(root, filepath.FromSlash(SkillDir))

	// (a) a directory the payload does not contain
	staleDir := filepath.Join(skill, "leftoverdir", "nested")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// (b) a payload file left at a non-default mode
	skillMD := filepath.Join(skill, "SKILL.md")
	if err := os.Chmod(skillMD, 0o600); err != nil {
		t.Fatal(err)
	}
	// (c) a payload file replaced by a symlink pointing outside the tree, whose
	// target has identical content — invisible to a content comparison, since
	// reading through the link returns exactly the expected bytes.
	tmpl := filepath.Join(skill, "resources", "TEMPLATE.md")
	original, err := os.ReadFile(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(tmpl); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, tmpl); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if _, err := os.Stat(filepath.Join(skill, "leftoverdir")); !os.IsNotExist(err) {
		t.Error("(a) a stale directory absent from the payload survived upgrade: the skill dir was not replaced wholesale")
	}
	if fi, err := os.Stat(skillMD); err != nil {
		t.Fatal(err)
	} else if fi.Mode().Perm() != 0o644 {
		t.Errorf("(b) SKILL.md mode = %v, want 0644 restored by the re-copy", fi.Mode().Perm())
	}
	if fi, err := os.Lstat(tmpl); err != nil {
		t.Fatal(err)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("(c) a payload file swapped for a symlink outside the tree survived upgrade")
	}
}

// TestUpgradeSurvivesAnUnreadableSkillEntry is the T-013 review's B4
// regression: skillPayloadDiffers is advisory — it picks a label and nothing
// else — but it runs before the wipe, so an error escaping it aborted the
// entire upgrade. An unreadable directory in the skill tree made
// `pickle upgrade` exit 1 without refreshing markers, scaffolds, hooks or the
// payload_version stamp, where before it was simply deleted by the re-copy.
func TestUpgradeSurvivesAnUnreadableSkillEntry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode bits do not deny access, so this case cannot be provoked")
	}
	payload := os.DirFS(payloadRoot())
	root := t.TempDir()
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", InTree: true}); err != nil {
		t.Fatal(err)
	}

	locked := filepath.Join(root, filepath.FromSlash(SkillDir), "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	// Restore the mode even if an assertion fails, so t.TempDir's own cleanup
	// can still remove the tree.
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	res, err := Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatalf("an unreadable directory must not fail the upgrade, got: %v", err)
	}
	if _, statErr := os.Lstat(locked); !os.IsNotExist(statErr) {
		t.Error("the unreadable directory survived the upgrade; the wipe should have removed it")
	}
	// The upgrade must have gone on to do the rest of its job, not just avoid
	// erroring: the stamp is the last thing it touches, so it proves the whole
	// sequence ran.
	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PayloadVersion != "v2" {
		t.Errorf("payload_version = %q, want v2 — the upgrade aborted before stamping", cfg.PayloadVersion)
	}
	if !hasEntry(res.Created, SkillDir+"/ (refreshed)") {
		t.Errorf("want the skill dir reported as refreshed, got created=%v skipped=%v", res.Created, res.Skipped)
	}
}

// TestUpgradeReportsStampVerificationFailure is the T-013 item 8 (R9 finding
// 1) regression: verifyStampedVersion was 100% unit-covered but 0% bound —
// deleting its call site inside Upgrade failed no test. This substitutes a
// lying stamp func (reports success but leaves pickle.toml unstamped) via the
// UpgradeOptions seam and asserts Upgrade surfaces that as an error instead of
// reporting success.
func TestUpgradeReportsStampVerificationFailure(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: "."}); err != nil {
		t.Fatal(err)
	}

	lying := func(path, want string) error { return nil } // claims success, writes nothing

	_, err := Upgrade(payload, root, "v2", UpgradeOptions{StampPayloadVersion: lying})
	if err == nil {
		t.Fatal("expected Upgrade to error when the stamp step lies about success, got nil")
	}

	cfg, loadErr := config.Load(filepath.Join(root, config.FileName))
	if loadErr != nil {
		t.Fatalf("load config: %v", loadErr)
	}
	if cfg.PayloadVersion != "v1" {
		t.Errorf("payload_version = %q, want it left at v1 since the lying stamp never wrote anything", cfg.PayloadVersion)
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
	block := MarkerBlock(cfg)

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

// TestMarkerBlockNoChildrenHasNoDanglingBullets pins a rework fix (T-108
// review finding F4): the umbrella layout's fresh-install state registers no
// child, and "Branch per child:"/"WIP limits (per child):" must not render
// with nothing after the colon — the same empty-case guard the children
// summary bullet already applies.
func TestMarkerBlockNoChildrenHasNoDanglingBullets(t *testing.T) {
	cfg := &config.Config{
		Commit: config.CommitPolicy{OverarchingAuto: true, ChildPublishGated: true},
	}
	block := MarkerBlock(cfg)

	for _, dangling := range []string{
		"Branch per child:\n",
		"(per child):\n",
	} {
		if strings.Contains(block, dangling) {
			t.Errorf("marker block has a dangling bullet %q\n--- block ---\n%s", dangling, block)
		}
	}
	if !strings.Contains(block, "(none yet — register with `pickle project add`)") {
		t.Errorf("marker block missing the no-children build-target clause\n--- block ---\n%s", block)
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
	block := MarkerBlock(cfg)

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
	block := MarkerBlock(cfg)

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
	if !strings.Contains(MarkerBlock(cfg), "**not publish-gated**") {
		t.Errorf("an explicit child_publish_gated = false was overridden by the default:\n%s", MarkerBlock(cfg))
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
			block := MarkerBlock(cfg)
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
			BranchPrefix: "ticket/", TicketPrefix: "BETA", WIPInDevelopment: 3, WIPInReview: 2,
		}},
	}
	got := MarkerBlock(cfg)

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

// TestPayloadDefersToProjectConfig guards T-022's fix: the payload used to state
// commit policy, branch prefix, ticket-id prefix and WIP limits as absolutes
// ("pushing a child-project requires explicit user approval", "feat/T-NNN",
// "≤ 1") even though all four are per-child (or overarching) `pickle.toml`
// settings (internal/config/config.go's DefaultBranchPrefix, DefaultTicketPrefix,
// DefaultWIPInDevelopment/Review, and CommitPolicy.ChildPublishGated) that
// MarkerBlock (internal/install/install.go:796-877) renders per project,
// including the inverted "not publish-gated" wording when child_publish_gated is
// false. A non-default project therefore shipped two authoritative,
// contradicting surfaces: the marker block's real values and the skill's
// absolute prose. Two things must hold for that fix to stick:
//
//  1. Every payload file that states these defaults also states, once, that
//     project configuration wins over the stated default — the literal anchor
//     `Project configuration wins`.
//  2. The specific absolute phrasings that caused the contradiction are gone
//     for good. This blocklist is deliberately short: the payload legitimately
//     keeps stating defaults in this vocabulary ("publish-gated", "≤ 1", …), and
//     a broad sweep for e.g. every "never push" would fail on correct prose that
//     merely states the default. Only the two phrasings that were themselves the
//     defect are checked — the frontmatter description every agent loader reads,
//     and the §8 pickup-gate heading that named the wrong test (T-022 adjacent
//     item A: it read "freshness", its own mandate was assumptions true/required/
//     worth it, so DROP as a legal verdict went unexercised).
func TestPayloadDefersToProjectConfig(t *testing.T) {
	files := []string{
		"skill/SKILL.md",
		"skill/resources/tickets-README.md",
		"skill/resources/review-protocol.md",
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

	// 1. Every payload file names the precedence rule.
	const anchor = "Project configuration wins"
	for _, rel := range files {
		if !strings.Contains(body[rel], anchor) {
			t.Errorf("%s does not contain %q: a project's AGENTS.md marker block renders the"+
				" real branch prefix, ticket prefix, WIP limits and commit policy, and this"+
				" file must say it wins over the default stated here", rel, anchor)
		}
	}

	// 2. The retired absolute phrasings never come back.
	const frontmatterLie = "pushing a child-project requires explicit user approval"
	if strings.Contains(body["skill/SKILL.md"], frontmatterLie) {
		t.Errorf("skill/SKILL.md frontmatter description still states %q as fact: it is false"+
			" for any project with child_publish_gated = false", frontmatterLie)
	}

	const freshnessHeading = "Pickup is gated by a freshness check"
	if strings.Contains(body["skill/resources/tickets-README.md"], freshnessHeading) {
		t.Errorf("tickets-README.md still heads the pickup gate %q: its own mandate (SKILL.md's"+
			" \"true, required, worth it\") is a merit test, not just an aging check, and DROP is"+
			" a legal verdict there", freshnessHeading)
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

// TestMarkerBlockRendersTicketPrefixInBranchBullet is the regression test for
// T-041's re-verification finding 4: the branch bullet must render each
// child's own ticket_prefix (via config.Project.Prefix), not the literal
// letter T.
func TestMarkerBlockRendersTicketPrefixInBranchBullet(t *testing.T) {
	cfg := &config.Config{
		Commit: config.CommitPolicy{OverarchingAuto: true, ChildPublishGated: true},
		Projects: []config.Project{
			{Name: "solo", Path: ".", BranchPrefix: "feat/", TicketPrefix: "RICK", WIPInDevelopment: 1, WIPInReview: 1},
			{Name: "legacy", Path: "legacy", BranchPrefix: "feat/", WIPInDevelopment: 1, WIPInReview: 1},
		},
	}
	block := MarkerBlock(cfg)

	if !strings.Contains(block, "- `solo`: `feat/RICK-NNN-<slug>`") {
		t.Errorf("branch bullet did not honour ticket_prefix %q:\n%s", "RICK", block)
	}
	// A child that leaves ticket_prefix unset still defaults to the legacy T.
	if !strings.Contains(block, "- `legacy`: `feat/T-NNN-<slug>`") {
		t.Errorf("branch bullet did not default an unset ticket_prefix to T:\n%s", block)
	}
}

// TestMarkerSpan pins the single predicate every marker-scanning site now
// shares (T-042 item 1's marker-span half): ordering is part of the
// predicate, not just presence.
func TestMarkerSpan(t *testing.T) {
	cases := []struct {
		name string
		text string
		ok   bool
	}{
		{"ordered pair", MarkerBegin + "\nbody\n" + MarkerEnd, true},
		{"reversed pair", MarkerEnd + "\n" + MarkerBegin, false},
		{"begin only", MarkerBegin, false},
		{"end only", MarkerEnd, false},
		{"neither", "nothing here", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, ok := markerSpan(tc.text)
			if ok != tc.ok {
				t.Errorf("markerSpan(%q) ok = %v, want %v", tc.text, ok, tc.ok)
			}
		})
	}
}

// TestInjectMarkerAppendSpacing is the T-013 item 1 regression: injectMarker's
// no-marker-found append branch must leave exactly one blank line before the
// appended block regardless of how many trailing newlines the file already
// had (zero, one, or several) — the old sep-computation had a dead else-if
// branch and let a file ending in exactly "\n\n" pick up an extra blank line.
func TestInjectMarkerAppendSpacing(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"no trailing newline", "# Title\n\nsome body"},
		{"one trailing newline", "# Title\n\nsome body\n"},
		{"two trailing newlines", "# Title\n\nsome body\n\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "AGENTS.md")
			if err := os.WriteFile(path, []byte(tc.text), 0o644); err != nil {
				t.Fatal(err)
			}
			var res Result
			if err := injectMarker(path, "Title", "body", &res); err != nil {
				t.Fatalf("injectMarker: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.TrimRight(tc.text, "\n") + "\n\n" + MarkerBegin + "\nbody\n" + MarkerEnd + "\n"
			if string(got) != want {
				t.Errorf("injectMarker output = %q, want %q", string(got), want)
			}
		})
	}
}

// TestUninstallDryRunAgreesOnReversedMarkers is the T-042 item 1 regression:
// uninstall --dry-run and the real uninstall must never disagree about
// whether a file has a marker block, including on a file whose
// <!-- pickle:end --> precedes its <!-- pickle:begin -->.
func TestUninstallDryRunAgreesOnReversedMarkers(t *testing.T) {
	reversed := "# X\n\n" + MarkerEnd + "\n" + MarkerBegin + "\n"

	dryRunRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(dryRunRoot, "AGENTS.md"), []byte(reversed), 0o644); err != nil {
		t.Fatal(err)
	}
	dryRes, err := Uninstall(os.DirFS(payloadRoot()), dryRunRoot, UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run Uninstall: %v", err)
	}

	realRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(realRoot, "AGENTS.md"), []byte(reversed), 0o644); err != nil {
		t.Fatal(err)
	}
	realRes, err := Uninstall(os.DirFS(payloadRoot()), realRoot, UninstallOptions{})
	if err != nil {
		t.Fatalf("real Uninstall: %v", err)
	}

	dryHasNoMarker := len(dryRes.Skipped) > 0 && strings.Contains(dryRes.Skipped[0], "no marker")
	realHasNoMarker := len(realRes.Skipped) > 0 && strings.Contains(realRes.Skipped[0], "no marker")
	if !dryHasNoMarker || !realHasNoMarker {
		t.Errorf("dry-run and real uninstall must agree a reversed marker pair is no marker at all; dry-run: created=%v removed=%v skipped=%v; real: created=%v removed=%v skipped=%v",
			dryRes.Created, dryRes.Removed, dryRes.Skipped, realRes.Created, realRes.Removed, realRes.Skipped)
	}
}

// TestUninstallDryRunAgreesOnSkillDirKind is the T-042 item 1 regression:
// the dry-run label must name the same kind (symlink vs directory) the real
// run's label would, not just that something will be removed.
func TestUninstallDryRunAgreesOnSkillDirKind(t *testing.T) {
	// removedSkillDirLabel installs at root (a real skill dir via Run, or a
	// self-host symlink built by hand), then returns the Removed entry naming
	// the skill dir for the given DryRun setting.
	removedSkillDirLabel := func(t *testing.T, symlink bool, dryRun bool) string {
		t.Helper()
		root := t.TempDir()
		payload := os.DirFS(payloadRoot())
		if symlink {
			skillPath := filepath.Join(root, filepath.FromSlash(SkillDir))
			if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
				t.Fatal(err)
			}
			external := t.TempDir()
			if err := os.WriteFile(filepath.Join(external, "SKILL.md"), []byte("external"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, skillPath); err != nil {
				t.Fatal(err)
			}
		} else {
			opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}
			if _, err := Run(payload, root, "v1", opts); err != nil {
				t.Fatal(err)
			}
		}

		res, err := Uninstall(payload, root, UninstallOptions{DryRun: dryRun})
		if err != nil {
			t.Fatalf("Uninstall (dryRun=%v): %v", dryRun, err)
		}
		for _, r := range res.Removed {
			if strings.HasPrefix(r, SkillDir) {
				return r
			}
		}
		t.Fatalf("no Removed entry for %s (dryRun=%v): %v", SkillDir, dryRun, res.Removed)
		return ""
	}

	t.Run("symlink", func(t *testing.T) {
		dry := removedSkillDirLabel(t, true, true)
		real := removedSkillDirLabel(t, true, false)
		if !strings.Contains(dry, "symlink") {
			t.Errorf("dry-run label = %q, want it to name a symlink", dry)
		}
		if !strings.Contains(real, "symlink") {
			t.Errorf("real label = %q, want it to name a symlink", real)
		}
	})

	t.Run("directory", func(t *testing.T) {
		dry := removedSkillDirLabel(t, false, true)
		real := removedSkillDirLabel(t, false, false)
		if strings.Contains(dry, "symlink") {
			t.Errorf("dry-run label = %q, wrongly names a symlink for a real directory", dry)
		}
		if strings.Contains(real, "symlink") {
			t.Errorf("real label = %q, wrongly names a symlink for a real directory", real)
		}
	})
}

func TestRefreshMarkers(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{
		Commit: config.CommitPolicy{OverarchingAuto: true, ChildPublishGated: true},
		Projects: []config.Project{{
			Name: "solo", Path: ".", BranchPrefix: "feat/", WIPInDevelopment: 1, WIPInReview: 1,
		}},
	}

	// AGENTS.md is created from nothing.
	res, err := RefreshMarkers(root, cfg)
	if err != nil {
		t.Fatalf("RefreshMarkers (create): %v", err)
	}
	if len(res.Created) == 0 {
		t.Fatalf("expected AGENTS.md to be created, got: %+v", res)
	}
	body, ok := InstalledMarkerBody(filepath.Join(root, "AGENTS.md"))
	if !ok || !strings.Contains(body, "`solo`") {
		t.Fatalf("AGENTS.md marker body missing solo after create: ok=%v body=%s", ok, body)
	}

	// A second call against an unchanged config is a no-op.
	res, err = RefreshMarkers(root, cfg)
	if err != nil {
		t.Fatalf("RefreshMarkers (no-op): %v", err)
	}
	if len(res.Created) != 0 {
		t.Errorf("an unchanged config re-wrote AGENTS.md: %+v", res)
	}
	if len(res.Skipped) == 0 || !strings.Contains(res.Skipped[0], "marker current") {
		t.Errorf("expected a marker-current skip, got: %+v", res)
	}

	// A regular-file CLAUDE.md is refreshed too.
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# CLAUDE.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RefreshMarkers(root, cfg); err != nil {
		t.Fatalf("RefreshMarkers (CLAUDE.md regular file): %v", err)
	}
	if _, ok := InstalledMarkerBody(filepath.Join(root, "CLAUDE.md")); !ok {
		t.Error("CLAUDE.md regular file was not injected")
	}

	// A CLAUDE.md -> AGENTS.md symlink is left alone.
	symlinkRoot := t.TempDir()
	if _, err := RefreshMarkers(symlinkRoot, cfg); err != nil {
		t.Fatalf("RefreshMarkers (symlink root, create): %v", err)
	}
	if err := os.Symlink("AGENTS.md", filepath.Join(symlinkRoot, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := RefreshMarkers(symlinkRoot, cfg); err != nil {
		t.Fatalf("RefreshMarkers (symlink root, second call): %v", err)
	}
	fi, err := os.Lstat(filepath.Join(symlinkRoot, "CLAUDE.md"))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("RefreshMarkers replaced a CLAUDE.md -> AGENTS.md symlink with a regular file")
	}
}

// TestSelfHostMarkerBlockIsCurrent pins this repo's own hand-mirroring
// obligation (AGENTS.md's self-modify policy: "Marker-block changes are made
// by hand … mirroring install.go's markerBlock()"). If this fails, MarkerBlock
// rendered something new and AGENTS.md was not updated to match in the same
// commit — see T-041 decision 7 before touching AGENTS.md by hand.
func TestSelfHostMarkerBlockIsCurrent(t *testing.T) {
	cfgPath := filepath.Join(payloadRoot(), config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load %s: %v", cfgPath, err)
	}
	want := strings.Trim(MarkerBlock(cfg), "\n")

	agentsPath := filepath.Join(payloadRoot(), "AGENTS.md")
	got, ok := InstalledMarkerBody(agentsPath)
	if !ok {
		t.Fatalf("%s has no readable marker block", agentsPath)
	}
	if got == want {
		return
	}
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	n := len(gotLines)
	if len(wantLines) > n {
		n = len(wantLines)
	}
	t.Errorf("AGENTS.md's marker block is out of date with pickle.toml + MarkerBlock — " +
		"hand-mirror AGENTS.md per the self-modify policy (AGENTS.md, \"Consequences of self-hosting\")")
	for i := 0; i < n; i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			t.Logf("line %d:\n  installed: %q\n  rendered : %q", i+1, g, w)
		}
	}
}
