package scaffold

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// payloadRoot returns the dir whose os.DirFS exposes "scaffold/docs-template/..."
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

func TestDocsCreatesAllFilesWithProjectName(t *testing.T) {
	root := t.TempDir()
	// Whether `snowball` happens to be on the test runner's PATH is
	// environment-dependent (it adds its own "snowball.yaml (via `snowball
	// init`)" entry to Created) — TestDocsMissingSnowballIsNonFatalWarning
	// below pins that behaviour explicitly; this test only asserts the four
	// template files it always writes.
	res, err := Docs(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}

	attrs := mustRead(t, filepath.Join(root, "docs/attributes.adoc"))
	if !strings.Contains(attrs, ":product: demo") {
		t.Errorf("attributes.adoc missing substituted project name: %q", attrs)
	}
	manual := mustRead(t, filepath.Join(root, "docs/user-manual.adoc"))
	if !strings.Contains(manual, "= demo User Manual") {
		t.Errorf("user-manual.adoc missing substituted project name: %q", manual)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/user-manual/introduction.adoc")); err != nil {
		t.Errorf("introduction.adoc not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github/workflows/docs-release.yml")); err != nil {
		t.Errorf("docs-release.yml not created: %v", err)
	}
	for _, f := range []string{"docs/attributes.adoc", "docs/user-manual.adoc", "docs/user-manual/introduction.adoc", ".github/workflows/docs-release.yml"} {
		if !contains(res.Created, f) {
			t.Errorf("Created = %v, want it to contain %q", res.Created, f)
		}
	}
}

func TestDocsDefaultsProjectNameToRootBasename(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "my-project")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Docs(payload(), root, Options{}); err != nil {
		t.Fatalf("Docs: %v", err)
	}
	attrs := mustRead(t, filepath.Join(root, "docs/attributes.adoc"))
	if !strings.Contains(attrs, ":product: my-project") {
		t.Errorf("attributes.adoc = %q, want project name defaulted to root basename", attrs)
	}
}

func TestDocsRerunWithoutForceSkipsExisting(t *testing.T) {
	root := t.TempDir()
	if _, err := Docs(payload(), root, Options{ProjectName: "demo"}); err != nil {
		t.Fatalf("first Docs: %v", err)
	}
	res, err := Docs(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("second Docs: %v", err)
	}
	if len(res.Created) != 0 {
		t.Errorf("Created = %v, want none on a re-run without --force", res.Created)
	}
	for _, f := range []string{"docs/attributes.adoc", "docs/user-manual.adoc", "docs/user-manual/introduction.adoc", ".github/workflows/docs-release.yml"} {
		if !contains(res.Skipped, f) {
			t.Errorf("Skipped = %v, want it to contain %q", res.Skipped, f)
		}
	}
}

func TestDocsForceOverwrites(t *testing.T) {
	root := t.TempDir()
	if _, err := Docs(payload(), root, Options{ProjectName: "first"}); err != nil {
		t.Fatalf("first Docs: %v", err)
	}
	if _, err := Docs(payload(), root, Options{ProjectName: "second", Force: true}); err != nil {
		t.Fatalf("second Docs: %v", err)
	}
	attrs := mustRead(t, filepath.Join(root, "docs/attributes.adoc"))
	if !strings.Contains(attrs, ":product: second") {
		t.Errorf("attributes.adoc = %q, want overwritten with the second project name", attrs)
	}
}

func TestDocsDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	res, err := Docs(payload(), root, Options{ProjectName: "demo", DryRun: true})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs")); !os.IsNotExist(err) {
		t.Errorf("docs/ exists after --dry-run (err=%v), want no such file", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); !os.IsNotExist(err) {
		t.Errorf(".github/ exists after --dry-run (err=%v), want no such file", err)
	}
	if len(res.Created) != 0 {
		t.Errorf("Created = %v, want none under --dry-run", res.Created)
	}
	if !contains(res.Notes, "would create") {
		t.Errorf("Notes = %v, want a dry-run preview", res.Notes)
	}
}

func TestDocsJustfileMissingIsSkippedNeverCreated(t *testing.T) {
	root := t.TempDir()
	res, err := Docs(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "justfile")); !os.IsNotExist(err) {
		t.Errorf("justfile was created (err=%v), want pickle to never invent one", err)
	}
	if !contains(res.Skipped, "justfile") {
		t.Errorf("Skipped = %v, want a justfile note", res.Skipped)
	}
}

func TestDocsJustfileAppendsMissingRecipesOnly(t *testing.T) {
	root := t.TempDir()
	original := "default:\n\t@just --list\n\ndocs-check:\n\techo already-here\n"
	if err := os.WriteFile(filepath.Join(root, "justfile"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Docs(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	got := mustRead(t, filepath.Join(root, "justfile"))
	if !strings.Contains(got, "echo already-here") {
		t.Errorf("justfile = %q, want the existing docs-check recipe preserved verbatim", got)
	}
	if !strings.Contains(got, "docs-build:") {
		t.Errorf("justfile = %q, want docs-build appended", got)
	}
	if strings.Count(got, "docs-check:") != 1 {
		t.Errorf("justfile = %q, want exactly one docs-check recipe (not duplicated)", got)
	}
	if !contains(res.Skipped, "docs-check") {
		t.Errorf("Skipped = %v, want docs-check reported as already defined", res.Skipped)
	}
	if !contains(res.Created, "docs-build") {
		t.Errorf("Created = %v, want docs-build reported as appended", res.Created)
	}
}

func TestDocsJustfileDryRunAppendsNothing(t *testing.T) {
	root := t.TempDir()
	original := "default:\n\t@just --list\n"
	justfilePath := filepath.Join(root, "justfile")
	if err := os.WriteFile(justfilePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Docs(payload(), root, Options{ProjectName: "demo", DryRun: true})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	if got := mustRead(t, justfilePath); got != original {
		t.Errorf("justfile = %q, want unchanged under --dry-run", got)
	}
	for _, want := range []string{"docs-check", "docs-build"} {
		if !contains(res.Notes, want) {
			t.Errorf("Notes = %v, want a dry-run preview mentioning %q", res.Notes, want)
		}
	}
}

func TestDocsSnowballDryRunDoesNotRunInit(t *testing.T) {
	if _, err := exec.LookPath("snowball"); err != nil {
		t.Skip("snowball not on PATH in this environment")
	}
	root := t.TempDir()
	res, err := Docs(payload(), root, Options{ProjectName: "demo", DryRun: true})
	if err != nil {
		t.Fatalf("Docs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "snowball.yaml")); !os.IsNotExist(err) {
		t.Errorf("snowball.yaml exists (err=%v), want --dry-run to never run `snowball init`", err)
	}
	if !contains(res.Notes, "would run") {
		t.Errorf("Notes = %v, want a dry-run preview for snowball init", res.Notes)
	}
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
// (T-113): Release must write exactly the two files, nothing that Docs'
// justfile/snowball helpers would add.
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

func TestDocsMissingSnowballIsNonFatalWarning(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", "") // exec.LookPath("snowball") must fail
	res, err := Docs(payload(), root, Options{ProjectName: "demo"})
	if err != nil {
		t.Fatalf("Docs: %v, want success even with snowball absent from PATH", err)
	}
	if !contains(res.Notes, "snowball not found on PATH") {
		t.Errorf("Notes = %v, want a missing-snowball warning", res.Notes)
	}
	// Every other file must still be created.
	for _, f := range []string{"docs/attributes.adoc", "docs/user-manual.adoc", "docs/user-manual/introduction.adoc", ".github/workflows/docs-release.yml"} {
		if !contains(res.Created, f) {
			t.Errorf("Created = %v, want it to contain %q even without snowball", res.Created, f)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "snowball.yaml")); !os.IsNotExist(err) {
		t.Errorf("snowball.yaml exists (err=%v), want none created without the snowball binary", err)
	}
}
