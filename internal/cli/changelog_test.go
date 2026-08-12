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

	if !strings.Contains(out, "v0.1.0..HEAD") {
		t.Errorf("output does not show the default --since resolving to the last tag and --until defaulting to HEAD, got:\n%s", out)
	}
	if !strings.Contains(out, `"Unreleased"`) {
		t.Errorf("output does not show the default --section Unreleased, got:\n%s", out)
	}
	if !strings.Contains(out, "T-050") {
		t.Errorf("output missing the one candidate T-050, got:\n%s", out)
	}
	// The default report summarizes exclusions (T-094 decision 4) — the full
	// subject is not printed unless --show-excluded is given (see
	// TestChangelogCheckShowExcludedPrintsSubjects).
	if !strings.Contains(out, "excluded 1 board: bookkeeping commit(s) mentioning T-050") {
		t.Errorf("output missing the exclusion summary line, got:\n%s", out)
	}
	if strings.Contains(out, "board: T-050 filed") {
		t.Errorf("default output should not print the full excluded subject, got:\n%s", out)
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
	if !strings.Contains(out, "v1.0.0..HEAD") {
		t.Errorf("output does not honour --since (and default --until HEAD), got:\n%s", out)
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

// TestChangelogCheckUntilExcludesLaterCommits (T-094 decision 2): a commit
// shipped after --until must not appear as a candidate — this is what makes
// a past --section auditable once later work has landed on top of it.
func TestChangelogCheckUntilExcludesLaterCommits(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [0.2.0]\n\n### Added\n\n- shipped in this release (T-050)\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")
	writeAndCommit(t, root, "src/a.go", "package a\n", "feat(a): ship a thing (T-050)")
	gitTag(t, root, "v0.2.0")
	// Lands after v0.2.0 — must not count when auditing the [0.2.0] section.
	writeAndCommit(t, root, "src/b.go", "package b\n", "feat(b): ship another thing (T-060)")

	out := captureStdout(t, func() {
		got := Run(nil, "test", []string{
			"changelog", "check",
			"--since", "v0.1.0",
			"--until", "v0.2.0",
			"--section", "0.2.0",
		})
		if got != exitOK {
			t.Fatalf("changelog check --until v0.2.0 = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(out, "T-060") {
		t.Errorf("a commit after --until must not appear, got:\n%s", out)
	}
	if !strings.Contains(out, "no candidates") {
		t.Errorf("T-050 is shipped and mentioned in [0.2.0], want no candidates, got:\n%s", out)
	}
}

// TestChangelogCheckSinceDefaultsRelativeToUntil pins T-094 decision 3, the
// exact footgun this ticket exists to close: with --until <tag> given and no
// --since, resolving --since from HEAD (instead of from --until) would ask
// git to describe HEAD's own last tag — which, when --until IS the last tag,
// produces the same tag twice and an empty, falsely-passing range. Resolving
// --since as "the tag before --until" (git describe --tags --abbrev=0
// <until>^") gets the non-empty range a bare "--until <tag>" invocation
// obviously means.
func TestChangelogCheckSinceDefaultsRelativeToUntil(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [0.2.0]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")
	writeAndCommit(t, root, "src/a.go", "package a\n", "feat(a): ship a thing (T-050)")
	gitTag(t, root, "v0.2.0")
	// After v0.2.0 — must be excluded by --until, proving --since did not
	// silently fall back to describing HEAD.
	writeAndCommit(t, root, "src/b.go", "package b\n", "feat(b): ship another thing (T-060)")

	out := captureStdout(t, func() {
		got := Run(nil, "test", []string{"changelog", "check", "--until", "v0.2.0", "--section", "0.2.0"})
		if got != exitOK {
			t.Fatalf("changelog check --until v0.2.0 (no --since) = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "v0.1.0..v0.2.0") {
		t.Errorf("--since should default to the tag before --until (v0.1.0), got:\n%s", out)
	}
	if !strings.Contains(out, "T-050") {
		t.Errorf("T-050 shipped in v0.1.0..v0.2.0 and is unmentioned in [0.2.0], want it as a candidate, got:\n%s", out)
	}
	if strings.Contains(out, "T-060") {
		t.Errorf("T-060 shipped after --until and must not appear, got:\n%s", out)
	}
}

// TestChangelogCheckShowExcludedPrintsSubjects: the default report
// summarizes exclusions to one line (T-094 decision 4); --show-excluded adds
// every full subject back, exactly as the pre-T-094 report always printed.
func TestChangelogCheckShowExcludedPrintsSubjects(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")
	writeAndCommit(t, root, "tickets/NOTES.md", "note\n", "board: T-050 filed")

	defaultOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitOK {
			t.Fatalf("changelog check = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(defaultOut, "board: T-050 filed") {
		t.Errorf("default report should summarize, not print the subject, got:\n%s", defaultOut)
	}
	if !strings.Contains(defaultOut, "excluded 1 board: bookkeeping commit(s) mentioning T-050") {
		t.Errorf("default report missing the exclusion summary, got:\n%s", defaultOut)
	}

	verboseOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check", "--show-excluded"}); got != exitOK {
			t.Fatalf("changelog check --show-excluded = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(verboseOut, "board: T-050 filed") {
		t.Errorf("--show-excluded should print the full subject, got:\n%s", verboseOut)
	}
}

// TestChangelogCheckExclusionSummaryNamesEveryID (T-095 decisions 2/3): a
// two-id board: commit must have BOTH ids in the default summary line, and
// the summary must say "mentioning" now that the scan is whole-subject
// rather than leading-id-only.
func TestChangelogCheckExclusionSummaryNamesEveryID(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")
	writeAndCommit(t, root, "tickets/NOTES.md", "note\n", "board: T-010, T-011 re-aimed after review")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitOK {
			t.Fatalf("changelog check = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "excluded 1 board: bookkeeping commit(s) mentioning T-010, T-011") {
		t.Errorf("output missing both ids in the exclusion summary, got:\n%s", out)
	}
}

// TestChangelogCheckUntilFallsBackOnTaggedRootCommit (T-095 decision 7): when
// HEAD is the repository's root commit, <until>^ cannot resolve at all —
// this must fall back to describing <until> itself and still exit 0,
// restoring the pre-T-094 answer on a tagged root commit instead of an
// error that blames a missing tag for what is really a missing parent.
func TestChangelogCheckUntilFallsBackOnTaggedRootCommit(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	// One commit only — HEAD is the root commit — then tag it immediately.
	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitOK {
			t.Fatalf("changelog check on a tagged root commit = %d, want %d (fallback to describing --until itself)", got, exitOK)
		}
	})
	if !strings.Contains(out, "v0.1.0..HEAD") {
		t.Errorf("output does not show the fallback resolving --since to v0.1.0, got:\n%s", out)
	}
}

// TestChangelogCheckUntilFallsBackErrorsNamingUntilNotUntilCaret (T-095
// review finding N4): the fallback's error path — root commit, no tag
// anywhere — was previously verified only by hand. It must name --until
// itself ("HEAD"), not "HEAD^", so the message blames the actual missing
// thing (no tag at all) rather than the fallback's own missing parent.
func TestChangelogCheckUntilFallsBackErrorsNamingUntilNotUntilCaret(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	// One commit only, HEAD is the root commit, and it is never tagged.
	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")

	errOut := captureStderr(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitError {
			t.Fatalf("changelog check on an untagged root commit = %d, want %d", got, exitError)
		}
	})
	if !strings.Contains(errOut, "reachable from HEAD:") {
		t.Errorf("error should name HEAD, not HEAD^, got:\n%s", errOut)
	}
	if strings.Contains(errOut, "HEAD^") {
		t.Errorf("error should not blame HEAD^ once the fallback itself has failed, got:\n%s", errOut)
	}
}

// TestChangelogCheckNoTagBeforeFirstReleaseErrorsRatherThanFalsePasses
// (T-095 review finding B1, blocking): a repository with a reachable parent
// but no tag anywhere before --until (a project's first release, tagged)
// must still error, exactly as it did before T-095's fallback existed — not
// silently describe --until itself, which would resolve to --until's own
// tag and produce the empty, falsely-passing <tag>..<tag> range that T-094
// decision 3's "^" exists to prevent. This is the regression T-095's rework
// fixes: before the fix, this scenario printed "no candidates" at exit 0
// for tickets that genuinely shipped unmentioned.
func TestChangelogCheckNoTagBeforeFirstReleaseErrorsRatherThanFalsePasses(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	writeAndCommit(t, root, "src/a.go", "package a\n", "feat(a): first real feature (T-001)")
	writeAndCommit(t, root, "src/b.go", "package b\n", "feat(a): second feature (T-002)")
	gitTag(t, root, "v1.0.0") // the ONLY tag, and it is on HEAD itself

	errOut := captureStderr(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check"}); got != exitError {
			t.Fatalf("changelog check with no tag before the first release = %d, want %d (not a false pass)", got, exitError)
		}
	})
	if !strings.Contains(errOut, "reachable from HEAD^:") {
		t.Errorf("error should name HEAD^ — a parent exists, it just has no tag before it — got:\n%s", errOut)
	}
}

// TestChangelogCheckTagNoteOnDefaultSection (T-095 decision 8): a tagged
// --until with the default --section prints an advisory note; the same
// --until with an explicit --section does not, since the reader has already
// answered "which release".
func TestChangelogCheckTagNoteOnDefaultSection(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [0.2.0]\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.1.0")
	writeAndCommit(t, root, "src/a.go", "package a\n", "feat(a): ship a thing (T-050)")
	gitTag(t, root, "v0.2.0")

	noteOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check", "--until", "v0.2.0"}); got != exitOK {
			t.Fatalf("changelog check --until v0.2.0 = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(noteOut, "note: v0.2.0 is at tag v0.2.0") {
		t.Errorf("output missing the tagged---until note with the default --section, got:\n%s", noteOut)
	}
	if !strings.Contains(noteOut, "try --section 0.2.0") {
		t.Errorf("note should suggest --section 0.2.0, got:\n%s", noteOut)
	}

	noNoteOut := captureStdout(t, func() {
		got := Run(nil, "test", []string{"changelog", "check", "--until", "v0.2.0", "--section", "0.2.0"})
		if got != exitOK {
			t.Fatalf("changelog check --until v0.2.0 --section 0.2.0 = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(noNoteOut, "note:") {
		t.Errorf("an explicit --section must suppress the tagged---until note, got:\n%s", noNoteOut)
	}
}

// TestChangelogCheckTagNoteNamesTheRefNotJustTheTag (T-095 review finding
// N1): the note must name the --until ref the user actually passed, not
// just the tag it resolved to — "main is at tag v0.2.0", not the false
// "main is a tag" a branch name would have been under the old wording.
func TestChangelogCheckTagNoteNamesTheRefNotJustTheTag(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "v0.2.0")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check", "--until", "main"}); got != exitOK {
			t.Fatalf("changelog check --until main = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "note: main is at tag v0.2.0") {
		t.Errorf("note should name the ref (main) and the tag it resolved to (v0.2.0) separately, got:\n%s", out)
	}
}

// TestChangelogCheckTagNoteVersionTrimsOnlyANumericVPrefix (T-095 review
// finding N2): stripping a leading "v" for the --section suggestion must
// not mangle a tag that merely starts with the letter "v" — versionFromTag
// requires a digit immediately after it.
func TestChangelogCheckTagNoteVersionTrimsOnlyANumericVPrefix(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	writeAndCommit(t, root, "CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n", "chore: seed changelog")
	gitTag(t, root, "version-2")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"changelog", "check", "--until", "version-2"}); got != exitOK {
			t.Fatalf("changelog check --until version-2 = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "try --section version-2") {
		t.Errorf("a non-numeric-vN tag must not be mangled (e.g. into \"ersion-2\"), got:\n%s", out)
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
