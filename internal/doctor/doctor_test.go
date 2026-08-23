package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/install"
	"github.com/codcod/pickle/internal/testutil"
)

// payloadRoot mirrors install_test.go: the dir whose os.DirFS exposes
// "skill/SKILL.md" at the same paths the embedded payload uses (the repo
// root). A thin wrapper over testutil.RepoRoot() (T-042 item 3), kept because
// this file has too many call sites to update individually in one reviewable
// sitting.
func payloadRoot() string { return testutil.RepoRoot() }

// installFixture lays a real, healthy pickle install into a temp dir and gives
// the sole child ("."): a .git dir so it reads as a git repository.
func installFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	_, err := install.Run(os.DirFS(payloadRoot()), root, "test-ver", install.Options{
		ProjectName: "demo",
		ProjectPath: ".",
		InTree:      true, // T-108: a root-path child must declare the in-tree layout
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

// selfHostFixture is installFixture with the installed skill dir replaced by a
// symlink to a real payload tree — the self-hosting arrangement of pickle's
// own repo (.agents/skills/brine -> ../../skill). The link points at an
// absolute path, mirroring TestUpgradeSelfHostSymlinkGuard in the install
// package, so checkSkill still resolves SKILL.md and
// resources/tickets-README.md through it.
func selfHostFixture(t *testing.T) string {
	t.Helper()
	root := installFixture(t)
	linkSkill(t, root)
	return root
}

// linkSkill replaces root's installed skill dir with a symlink to this
// repo's own payload skill/ tree — the self-hosting arrangement
// (.agents/skills/brine -> ../../skill). The link points at an absolute
// path, mirroring TestUpgradeSelfHostSymlinkGuard in the install package, so
// checkSkill still resolves SKILL.md and resources/tickets-README.md through
// it. Shared by selfHostFixture (over installFixture's root) and
// TestCheckSelfHostLinkStillReportsIncapablePATHPickle (over gitFixture's
// root, in hooks_test.go) — both package doctor, so no cross-package plumbing
// is needed (T-042 item 4).
func linkSkill(t *testing.T, root string) {
	t.Helper()
	skillPath := filepath.Join(root, filepath.FromSlash(install.SkillDir))
	if err := os.RemoveAll(skillPath); err != nil {
		t.Fatal(err)
	}
	target, err := filepath.Abs(filepath.Join(payloadRoot(), "skill"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, skillPath); err != nil {
		t.Fatal(err)
	}
}

func hasErrContaining(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", dir, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
}

// secondChildFixture extends installFixture (a healthy, self-hosted "."
// install) with a second child registered at a nested, real git repository
// named "child" — the shape T-051's check cares about. The nested repo is
// left unignored by the caller so each test controls whether/when a
// .gitignore entry appears.
func secondChildFixture(t *testing.T) (root string, cfg *config.Config) {
	t.Helper()
	requireGit(t)
	root = installFixture(t)
	gitInit(t, root) // the outer "." install only faked a bare .git/ dir

	nested := filepath.Join(root, "child")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	gitInit(t, nested)

	cfgPath := filepath.Join(root, config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if err := cfg.AddProject(config.Project{Name: "child", Path: "child"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if err := cfg.Save(""); err != nil {
		t.Fatalf("cfg.Save: %v", err)
	}
	cfg, err = config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load (reload): %v", err)
	}
	return root, cfg
}

// TestCheckChildStageableWarns (T-051): a registered child at a nested path
// that git would still stage whole earns exactly one warning and no error —
// doctor's exit code must stay 0 for this alone.
func TestCheckChildStageableWarns(t *testing.T) {
	root, _ := secondChildFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors, got: %v", res.Errors)
	}
	if !hasWarnContaining(res.Warnings, `child "child"`) || !hasWarnContaining(res.Warnings, "/child/") {
		t.Fatalf("expected a warning naming child \"child\" and /child/, got: %v", res.Warnings)
	}
}

// TestCheckChildIgnoredIsSilent (T-051): once .gitignore covers the child,
// the warning disappears and a verbose pass line takes its place.
func TestCheckChildIgnoredIsSilent(t *testing.T) {
	root, _ := secondChildFixture(t)
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/child/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors, got: %v", res.Errors)
	}
	if hasWarnContaining(res.Warnings, `child "child"`) {
		t.Fatalf("expected no child-staging warning once ignored, got: %v", res.Warnings)
	}
	if !hasPassedContaining(res.Passed, `child "child" is git-ignored`) {
		t.Fatalf("expected a passed line reporting the child is ignored, got: %v", res.Passed)
	}
}

// TestCheckSelfHostChildNeverWarns (T-051): a child registered at "." (the
// single-repo default, and this repo's own shape) never triggers the check
// at all — there is nothing to stage, the child *is* the repo.
func TestCheckSelfHostChildNeverWarns(t *testing.T) {
	root := installFixture(t) // sole child at ".", bare .git/ dir, no real repo
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if hasWarnContaining(res.Warnings, "nested git repository") {
		t.Fatalf("expected no nested-repo warning for a \".\" child, got: %v", res.Warnings)
	}
}

// TestCheckHealthyInstallUnaffected reconfirms TestCheckHealthyInstall's
// invariant explicitly for T-051: installFixture's ".git" is a bare directory
// with no real repository inside, so vcs.ChildState resolves Unknown there
// and the new check must stay silent — checkChildren's git-repo error (if
// any) is unrelated and already covered by TestCheckBrokenArtifacts.
func TestCheckHealthyInstallUnaffected(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors on a healthy install, got: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("expected no warnings on a healthy install, got: %v", res.Warnings)
	}
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

// T-074: no legacy skill path present earns a passed line, visible under -v,
// mirroring T-046's self-host skip line.
func TestCheckNoLegacyPathIsPassed(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Passed, "no legacy skill path present") {
		t.Errorf("expected a passed entry for no legacy skill path, got: %v", res.Passed)
	}
}

// T-074: a legacy skill directory still present is an error naming
// `pickle upgrade`, not a warning — a tree carrying both names is serving a
// stale skill copy some agents would discover ahead of the current one.
func TestCheckLegacySkillDirErrors(t *testing.T) {
	root := installFixture(t)
	newSkill := filepath.Join(root, filepath.FromSlash(install.SkillDir))
	legacySkill := filepath.Join(root, filepath.FromSlash(install.LegacySkillDir))
	if err := os.Rename(newSkill, legacySkill); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, "pickle upgrade") || !hasErrContaining(res.Errors, install.LegacySkillDir) {
		t.Errorf("expected an error naming the legacy skill dir and `pickle upgrade`, got: %v", res.Errors)
	}
}

// T-074: a legacy Claude view still present is likewise an error.
func TestCheckLegacyClaudeLinkErrors(t *testing.T) {
	root := installFixture(t)
	newClaudeLink := filepath.Join(root, filepath.FromSlash(install.ClaudeSkillLink))
	legacyClaudeLink := filepath.Join(root, filepath.FromSlash(install.LegacyClaudeSkillLink))
	if err := os.Rename(newClaudeLink, legacyClaudeLink); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, "pickle upgrade") || !hasErrContaining(res.Errors, install.LegacyClaudeSkillLink) {
		t.Errorf("expected an error naming the legacy claude link and `pickle upgrade`, got: %v", res.Errors)
	}
}

// T-074: checkLegacyPaths is a pure filesystem read, so it still runs (and
// still errors) when pickle.toml itself fails to parse.
func TestCheckLegacyPathRunsWithUnparseableConfig(t *testing.T) {
	root := installFixture(t)
	newSkill := filepath.Join(root, filepath.FromSlash(install.SkillDir))
	legacySkill := filepath.Join(root, filepath.FromSlash(install.LegacySkillDir))
	if err := os.Rename(newSkill, legacySkill); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte("not = [valid"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, "pickle.toml:") {
		t.Errorf("expected the config parse error too, got: %v", res.Errors)
	}
	if !hasErrContaining(res.Errors, install.LegacySkillDir) {
		t.Errorf("expected the legacy skill dir error despite the unparseable config, got: %v", res.Errors)
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

// TestCheckSelfHostLinkSkipsVersionCheck is T-046's headline behaviour, the
// mirror of TestCheckVersionDriftWarns: in a self-hosting checkout the skill
// dir is a symlink to the payload source, not an installed copy, so comparing
// payload_version against the running binary is meaningless and must produce
// neither a warning nor an error — only an informational passed line.
func TestCheckSelfHostLinkSkipsVersionCheck(t *testing.T) {
	root := selfHostFixture(t) // installed at payload_version "test-ver"
	res := Check(root, "v9.9.9", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("self-host link must not error: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("self-host link must not warn: %v", res.Warnings)
	}
	if !hasPassedContaining(res.Passed, "payload version check skipped") {
		t.Errorf("expected a passed line naming the skip, got: %v", res.Passed)
	}
	for _, p := range res.Passed {
		if strings.Contains(p, "payload version") && !strings.Contains(p, "skipped") {
			t.Errorf("unexpected payload-version line: %q", p)
		}
		if strings.Contains(p, "pickle upgrade") {
			t.Errorf("self-host skip must never mention `pickle upgrade`: %q", p)
		}
	}
}

// TestCheckSelfHostLinkSkipsUnstampableVersionCheck proves the skip sits ahead
// of *both* of checkVersion's T-026 warning branches (D5): even when the
// installed pickle.toml has a payload_version shape the in-place writer would
// refuse to stamp, the self-host skip fires first and no warning appears. If
// the skip were inserted after the PayloadVersionStampable probe instead, this
// would fail with the "edit payload_version by hand" warning.
func TestCheckSelfHostLinkSkipsUnstampableVersionCheck(t *testing.T) {
	root := selfHostFixture(t) // installed at payload_version "test-ver"
	cfgPath := filepath.Join(root, config.FileName)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	wedged := strings.Replace(string(data), `payload_version = "test-ver"`, "payload_version = \"\"\"\ntest-ver\n\"\"\"", 1)
	if wedged == string(data) {
		t.Fatal("fixture setup: payload_version line not found to wedge")
	}
	if err := os.WriteFile(cfgPath, []byte(wedged), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Check(root, "v9.9.9", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("self-host link must not error: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("self-host link must not warn even with an unstampable payload_version: %v", res.Warnings)
	}
}

// TestCheckSelfHostLinkNamesTheLink pins the informational half of the
// Outcome: doctor -v's passed line for the skill check names the self-host
// link and the target it resolves to, rather than the generic "skill payload
// present" text an ordinary install gets.
//
// It also pins the *other* half of D5. This fixture is checked at a version
// equal to its own stamp, so it is the one case that distinguishes "the skip
// is checkVersion's first statement" from "the skip merely precedes the
// PayloadVersionStampable probe": if the skip were moved below the
// version==""/"dev" guard or the equal-version early return, those would
// return first and the passed line would vanish here, while every other
// self-host test still passed.
func TestCheckSelfHostLinkNamesTheLink(t *testing.T) {
	root := selfHostFixture(t)
	target, err := os.Readlink(filepath.Join(root, filepath.FromSlash(install.SkillDir)))
	if err != nil {
		t.Fatal(err)
	}
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("self-host link must not error: %v", res.Errors)
	}
	if !hasPassedContaining(res.Passed, "self-host link") {
		t.Errorf("expected a passed line naming the self-host link, got: %v", res.Passed)
	}
	if !hasPassedContaining(res.Passed, target) {
		t.Errorf("expected the passed line to name the link target %q, got: %v", target, res.Passed)
	}
	// D5: unconditional — the skip fires even when version == the stamp, which
	// the equal-version early return would otherwise swallow.
	if !hasPassedContaining(res.Passed, "payload version check skipped") {
		t.Errorf("the version-check skip must fire even when version equals the stamp (D5: first statement), got: %v", res.Passed)
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

// TestCheckLayoutInvariantHealthyInTreeIsPassed (T-108 decision 7): the
// default installFixture (one child at ".", layout "in-tree") earns an
// informational passed line, visible under -v, alongside its existing zero
// errors/warnings (already covered by TestCheckHealthyInstall).
func TestCheckLayoutInvariantHealthyInTreeIsPassed(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasPassedContaining(res.Passed, `layout "in-tree" is consistent with 1 root-path child`) {
		t.Errorf("expected the layout consistency line, got passed=%v", res.Passed)
	}
}

// TestCheckLayoutInvariantErrorsWhenInTreeHasNoRootChild (T-108 decision 7):
// an in-tree layout with its sole child moved to a nested path is a broken
// invariant \u2014 an error, not a warning \u2014 since a doctor-clean project must
// never carry a recorded claim the config contradicts.
func TestCheckLayoutInvariantErrorsWhenInTreeHasNoRootChild(t *testing.T) {
	root := installFixture(t)
	cfgPath := filepath.Join(root, config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.Projects[0].Path = "nested"
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(""); err != nil {
		t.Fatalf("cfg.Save: %v", err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, `layout: "in-tree" requires exactly one child registered at "."`) {
		t.Errorf("expected the layout invariant error, got errors=%v", res.Errors)
	}
}

// TestCheckLayoutInvariantErrorsWhenUmbrellaHasRootChild (T-108 decision 7):
// the umbrella layout with a child registered at "." is the same
// contradiction from the other side.
func TestCheckLayoutInvariantErrorsWhenUmbrellaHasRootChild(t *testing.T) {
	root := installFixture(t)
	cfgPath := filepath.Join(root, config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.Layout = config.LayoutUmbrella
	if err := cfg.Save(""); err != nil {
		t.Fatalf("cfg.Save: %v", err)
	}

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, `layout: "umbrella" must have no child registered at "."`) {
		t.Errorf("expected the layout invariant error, got errors=%v", res.Errors)
	}
}
