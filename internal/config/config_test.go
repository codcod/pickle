package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeCfg writes a pickle.toml into dir and returns its path.
func writeCfg(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	return path
}

const oneProject = `payload_version = "1.2.3"
[commit]
overarching_auto = true
child_publish_gated = true
[[project]]
name = "pickle"
path = "."
build = "just build"
`

func TestLoadValidAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.PayloadVersion != "1.2.3" {
		t.Errorf("payload_version = %q", c.PayloadVersion)
	}
	if len(c.Projects) != 1 || c.Projects[0].Name != "pickle" {
		t.Fatalf("projects = %+v", c.Projects)
	}
	p := c.Projects[0]
	if p.BranchPrefix != DefaultBranchPrefix {
		t.Errorf("branch_prefix default = %q", p.BranchPrefix)
	}
	if p.WIPInDevelopment != 1 || p.WIPInReview != 1 {
		t.Errorf("WIP defaults = %d/%d", p.WIPInDevelopment, p.WIPInReview)
	}
	if c.Root() != dir {
		t.Errorf("Root = %q, want %q", c.Root(), dir)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"no project": `payload_version = "1"
[commit]
`,
		"empty name": `payload_version = "1"
[[project]]
name = ""
path = "."
`,
		"empty path": `payload_version = "1"
[[project]]
name = "a"
path = ""
`,
		"bad path": `payload_version = "1"
[[project]]
name = "a"
path = "does-not-exist"
`,
		"duplicate name": `payload_version = "1"
[[project]]
name = "dup"
path = "."
[[project]]
name = "dup"
path = "."
`,
		"zero wip": `payload_version = "1"
[[project]]
name = "a"
path = "."
wip_in_review = -1
`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeCfg(t, dir, body)
			if _, err := Load(path); err == nil {
				t.Fatalf("expected error for %q, got nil", name)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := c.Save(""); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if c2.Render() != c.Render() {
		t.Errorf("round-trip not stable:\n--- first ---\n%s\n--- second ---\n%s", c.Render(), c2.Render())
	}
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	writeCfg(t, dir, oneProject)
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(nested)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != filepath.Join(dir, FileName) {
		t.Errorf("Find = %q", got)
	}
	if _, err := Find(t.TempDir()); err == nil {
		t.Error("expected Find to fail with no pickle.toml")
	}
}

func TestAddRemove(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := c.AddProject(Project{Name: "pickle", Path: "."}); err == nil {
		t.Error("expected duplicate-name error")
	}
	if err := c.AddProject(Project{Name: "web", Path: "web"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if p, ok := c.Project("web"); !ok || p.WIPInDevelopment != 1 || p.BranchPrefix != DefaultBranchPrefix {
		t.Errorf("added project missing defaults: %+v ok=%v", p, ok)
	}
	if err := c.RemoveProject("nope"); err == nil {
		t.Error("expected remove-unknown error")
	}
	if err := c.RemoveProject("web"); err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}
	if _, ok := c.Project("web"); ok {
		t.Error("web still present after remove")
	}
}

// TestLoadRepoConfig loads this repository's own pickle.toml (../../ from here).
func TestLoadRepoConfig(t *testing.T) {
	repoCfg, err := filepath.Abs(filepath.Join("..", "..", FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(repoCfg); err != nil {
		t.Skipf("repo pickle.toml not found: %v", err)
	}
	c, err := Load(repoCfg)
	if err != nil {
		t.Fatalf("Load repo config: %v", err)
	}
	if p, ok := c.Project("pickle"); !ok || p.Path != "." {
		t.Errorf("expected sole child 'pickle' at '.', got %+v ok=%v", c.Projects, ok)
	}
}
