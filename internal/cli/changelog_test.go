package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitCommit commits every already-`git add`ed change with subject as the
// message, isolated the same way gitInit is (no developer identity/config
// leaking into the fixture).
func gitCommit(t *testing.T, dir, subject string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "commit", "-q", "-m", subject)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit -m %q: %v: %s", subject, err, out)
	}
}

// gitTag tags dir's current HEAD.
func gitTag(t *testing.T, dir, name string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "tag", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git tag %s: %v: %s", name, err, out)
	}
}

// writeAndCommit writes content to relPath (under root) and commits it with
// subject as the message.
func writeAndCommit(t *testing.T, root, relPath, content, subject string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAdd(t, root, relPath)
	gitCommit(t, root, subject)
}

// TestChangelogCheckFlagDefaults pins the flag-default contract T-093's plan
// promises (--since: the last tag; --changelog: CHANGELOG.md; --section:
// Unreleased) and decision 2: candidates are reported but the command still
// exits 0 — it is advisory, never a gate.
func TestChangelogCheckFlagDefaults(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")

	// A child-project commit with no changelog entry yet — the one true
	// candidate this default-path run should surface.
	writeAndCommit(t, root, "src/thing.go", "package thing\n", "feat(thing): add a thing (T-050)")
	// A board: bookkeeping commit — must be excluded from "shipped" entirely,
	// per decision 3/T-084, and reported in the exclusion list (decision 7).
	writeAndCommit(t, root, "tickets/NOTES.md", "note\n", "board: T-050 filed")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitOK {
			t.Fatalf("changelog check = %d, want %d (advisory — must exit 0 even with candidates)", got, exitOK)
		}
	})

	if !strings.Contains(out, "since v0.1.0") {
		t.Errorf("output does not show the default --since resolving to the last tag, got:\n%s", out)
	}
	if !strings.Contains(out, `"Unreleased"`) {
		t.Errorf("output does not show the default --section Unreleased, got:\n%s", out)
	}
	if !strings.Contains(out, "T-050") {
		t.Errorf("output missing the one candidate T-050, got:\n%s", out)
	}
	if !strings.Contains(out, "board: T-050 filed") {
		t.Errorf("output missing the excluded board: commit, got:\n%s", out)
	}
}

// TestChangelogCheckNoCandidates: a shipped ticket the changelog already
// names produces no candidate, and the command still exits 0.
func TestChangelogCheckNoCandidates(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")

	writeAndCommit(t, root, "src/thing.go", "package thing\n", "feat(thing): add a thing (T-050)")
	writeAndCommit(t, root, "CHANGELOG.md",
		"# Changelog\n\n## [Unreleased]\n\n### Added\n\n- added a thing (T-050)\n",
		"docs(changelog): mention T-050 (T-050)")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitOK {
			t.Fatalf("changelog check = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "no candidates") {
		t.Errorf("output should report no candidates once T-050 is mentioned, got:\n%s", out)
	}
}

// TestChangelogCheckExplicitFlags exercises --since, --changelog and
// --section overriding every default at once.
func TestChangelogCheckExplicitFlags(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "docs/HISTORY.md", "# History\n\n## [past]\n", "chore: seed history")
	gitTag(t, root, "v1.0.0")
	writeAndCommit(t, root, "src/thing.go", "package thing\n", "feat(thing): add a thing (T-061)")

	out := captureStdout(t, func() {
		got := Run(nil, "test", []string{
			"changelog", "check",
			"--since", "v1.0.0",
			"--changelog", "docs/HISTORY.md",
			"--section", "past",
		})
		if got != exitOK {
			t.Fatalf("changelog check = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "since v1.0.0") {
		t.Errorf("output does not honour --since, got:\n%s", out)
	}
	if !strings.Contains(out, "docs/HISTORY.md") {
		t.Errorf("output does not honour --changelog, got:\n%s", out)
	}
	if !strings.Contains(out, `"past"`) {
		t.Errorf("output does not honour --section, got:\n%s", out)
	}
	if !strings.Contains(out, "T-061") {
		t.Errorf("output missing the candidate T-061, got:\n%s", out)
	}
}

// TestChangelogCheckUnknownSectionErrors: an absent section is a genuine
// user mistake (a typo'd --section, or a version that was never released),
// unlike a missing candidate — it exits non-zero.
func TestChangelogCheckUnknownSectionErrors(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")

	captureStdout(t, func() {
		got := Run(nil, "test", []string{"changelog", "check", "--section", "9.9.9"})
		if got == exitOK {
			t.Fatalf("changelog check --section 9.9.9 = %d, want non-zero (section does not exist)", got)
		}
	})
}

// TestChangelogCheckUsageErrors pins the usage-error surface for a
// malformed invocation.
func TestChangelogCheckUsageErrors(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"changelog"}); got != exitUsage {
		t.Fatalf("changelog (no subcommand) = %d, want %d", got, exitUsage)
	}
	if got := Run(nil, "test", []string{"changelog", "bogus"}); got != exitUsage {
		t.Fatalf("changelog bogus = %d, want %d", got, exitUsage)
	}
	if got := Run(nil, "test", []string{"changelog", "check", "extra-positional"}); got != exitUsage {
		t.Fatalf("changelog check extra-positional = %d, want %d", got, exitUsage)
	}
}
