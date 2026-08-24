package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// payloadRoot returns the dir whose os.DirFS exposes "scaffold/release-template/..."
// at the same paths the binary's embedded payloadFS uses (the repo root, ../..).
func payloadRoot() string { return filepath.Join("..", "..") }

func payload() fs.FS { return os.DirFS(payloadRoot()) }

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if strings.Contains(s, want) {
			return true
		}
	}
	return false
}

func TestReleaseCreatesBothFilesWithProjectName(t *testing.T) {
	root := t.TempDir()
	res, err := Release(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("Release: %v", err)
	}

	releasing := mustRead(t, filepath.Join(root, "RELEASING.md"))
	if !strings.Contains(releasing, "# Releasing demo") {
		t.Errorf("RELEASING.md missing substituted project name: %q", releasing)
	}
	changelog := mustRead(t, filepath.Join(root, "CHANGELOG.md"))
	if !strings.Contains(changelog, "## [Unreleased]") {
		t.Errorf("CHANGELOG.md missing an [Unreleased] section: %q", changelog)
	}
	for _, f := range []string{"CHANGELOG.md", "RELEASING.md"} {
		if !contains(res.Created, f) {
			t.Errorf("Created = %v, want it to contain %q", res.Created, f)
		}
	}
}

func TestReleaseDefaultsProjectNameToRootBasename(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "my-project")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Release(payload(), root, Options{}); err != nil {
		t.Fatalf("Release: %v", err)
	}
	releasing := mustRead(t, filepath.Join(root, "RELEASING.md"))
	if !strings.Contains(releasing, "# Releasing my-project") {
		t.Errorf("RELEASING.md = %q, want project name defaulted to root basename", releasing)
	}
}

func TestReleaseRerunWithoutForceSkipsExisting(t *testing.T) {
	root := t.TempDir()
	if _, err := Release(payload(), root, Options{ProjectName: "demo"}); err != nil {
		t.Fatalf("first Release: %v", err)
	}
	res, err := Release(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("second Release: %v", err)
	}
	if len(res.Created) != 0 {
		t.Errorf("Created = %v, want none on a re-run without --force", res.Created)
	}
	for _, f := range []string{"CHANGELOG.md", "RELEASING.md"} {
		if !contains(res.Skipped, f) {
			t.Errorf("Skipped = %v, want it to contain %q", res.Skipped, f)
		}
	}
}

func TestReleaseForceOverwrites(t *testing.T) {
	root := t.TempDir()
	if _, err := Release(payload(), root, Options{ProjectName: "first"}); err != nil {
		t.Fatalf("first Release: %v", err)
	}
	if _, err := Release(payload(), root, Options{ProjectName: "second", Force: true}); err != nil {
		t.Fatalf("second Release: %v", err)
	}
	releasing := mustRead(t, filepath.Join(root, "RELEASING.md"))
	if !strings.Contains(releasing, "# Releasing second") {
		t.Errorf("RELEASING.md = %q, want overwritten with the second project name", releasing)
	}
}

func TestReleaseDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	res, err := Release(payload(), root, Options{ProjectName: "demo", DryRun: true})
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Errorf("CHANGELOG.md exists after --dry-run (err=%v), want no such file", err)
	}
	if _, err := os.Stat(filepath.Join(root, "RELEASING.md")); !os.IsNotExist(err) {
		t.Errorf("RELEASING.md exists after --dry-run (err=%v), want no such file", err)
	}
	if len(res.Created) != 0 {
		t.Errorf("Created = %v, want none under --dry-run", res.Created)
	}
	if !contains(res.Notes, "would create") {
		t.Errorf("Notes = %v, want a dry-run preview", res.Notes)
	}
}

// TestReleaseWritesNoOtherFile is the regression guard for decision 8
// (T-113): Release must write exactly the two files, nothing else.
func TestReleaseWritesNoOtherFile(t *testing.T) {
	root := t.TempDir()
	if _, err := Release(payload(), root, Options{ProjectName: "demo"}); err != nil {
		t.Fatalf("Release: %v", err)
	}
	for _, f := range []string{"justfile", "snowball.yaml", ".github"} {
		if _, err := os.Stat(filepath.Join(root, f)); !os.IsNotExist(err) {
			t.Errorf("%s exists (err=%v), want Release to write only CHANGELOG.md and RELEASING.md", f, err)
		}
	}
}

// TestReleaseNamesNoCommand is the mechanical guard for decision 1 (T-113):
// the scaffolded RELEASING.md must never point at pickle changelog check (or
// any other command), since a brine-free repo — a supported scaffold
// audience — has no pickle.toml for it to resolve against.
func TestReleaseNamesNoCommand(t *testing.T) {
	root := t.TempDir()
	if _, err := Release(payload(), root, Options{ProjectName: "demo"}); err != nil {
		t.Fatalf("Release: %v", err)
	}
	releasing := mustRead(t, filepath.Join(root, "RELEASING.md"))
	if strings.Contains(releasing, "pickle ") {
		t.Errorf("RELEASING.md = %q, want no command named (decision 1)", releasing)
	}
}
