package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noSymlinks makes every symlink attempt in this package fail the way an
// unprivileged Windows account does, for the rest of the test (T-143).
func noSymlinks(t *testing.T) {
	t.Helper()
	old := symlink
	symlink = func(oldname, newname string) error {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: errors.New("a required privilege is not held by the client")}
	}
	t.Cleanup(func() { symlink = old })
}

func hasEntryContaining(list []string, sub string) bool {
	for _, e := range list {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

// TestClaudeViewCopyFallback walks the copy view through install, upgrade
// and uninstall on a filesystem that refuses symlinks (T-143 decision 4).
func TestClaudeViewCopyFallback(t *testing.T) {
	noSymlinks(t)
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	view := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))

	res, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasEntryContaining(res.Created, ClaudeSkillLink+" (copy — symlink unavailable: a required privilege") {
		t.Errorf("install did not name the fallback; created: %v", res.Created)
	}
	if !IsClaudeViewCopy(root) {
		t.Fatal("claude view is not a copy holding SKILL.md")
	}

	stale := filepath.Join(view, "stale.md")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = Upgrade(payload, root, "v2")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !hasEntryContaining(res.Created, ClaudeSkillLink+" (copy refreshed — symlink unavailable") {
		t.Errorf("upgrade did not refresh the copy; created: %v", res.Created)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("upgrade left a stale file in the copy view")
	}
	mustExist(t, filepath.Join(view, "SKILL.md"))
	mustExist(t, filepath.Join(view, ClaudeViewSentinel))
	for _, kind := range []string{"tmp", "old"} {
		if exists(claudeViewScratch(root, kind)) {
			t.Errorf("refresh left %s behind", claudeViewScratch(root, kind))
		}
	}

	res, err = Uninstall(payload, root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !hasEntry(res.Removed, ClaudeSkillLink+" (copy)") {
		t.Errorf("uninstall did not report the copy; removed: %v", res.Removed)
	}
	if _, err := os.Lstat(view); !os.IsNotExist(err) {
		t.Error("copy view still present after uninstall")
	}
}

// A real directory without the sentinel is not pickle's copy — even one
// holding a SKILL.md — and is refused, never wiped (review round 1, F2).
func TestClaudeViewForeignDirectoryRefused(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	view := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if err := os.Remove(view); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(view, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"SKILL.md", "notes.md"} {
		if err := os.WriteFile(filepath.Join(view, f), []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Upgrade(payload, root, "v2"); err == nil || !strings.Contains(err.Error(), "is not a symlink; refusing") {
		t.Fatalf("Upgrade err = %v, want the refusal", err)
	}
	if _, err := Uninstall(payload, root, UninstallOptions{}); err == nil {
		t.Fatal("Uninstall removed a directory that is not pickle's copy")
	}
	mustExist(t, filepath.Join(view, "notes.md"))
}

// A copy view reached through a symlinked .claude/skills belongs to whatever
// the link points at: upgrade and uninstall must not delete it (F1).
func TestClaudeViewThroughSymlinkedParentUntouched(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	ext := filepath.Join(outside, "brine")
	if err := os.MkdirAll(ext, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"SKILL.md", ClaudeViewSentinel, "notes.md"} {
		if err := os.WriteFile(filepath.Join(ext, f), []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	skills := filepath.Join(root, ".claude", "skills")
	if err := os.RemoveAll(skills); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, skills); err != nil {
		t.Fatal(err)
	}
	if IsClaudeViewCopy(root) {
		t.Error("a view under a symlinked parent was taken for pickle's copy")
	}
	_, _ = Upgrade(payload, root, "v2")
	_, _ = Uninstall(payload, root, UninstallOptions{})
	mustExist(t, filepath.Join(ext, "notes.md"))
}

// A copy made while symlinks were unavailable becomes a symlink on the first
// run after they are (F3).
func TestClaudeViewCopyConvergesOnSymlink(t *testing.T) {
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	noSymlinks(t)
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	symlink = os.Symlink // symlinks become available; noSymlinks' cleanup still restores the original
	if !IsClaudeViewCopy(root) {
		t.Fatal("expected a copy view")
	}
	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(ClaudeSkillLink)))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("view is not a symlink after upgrade (err %v)", err)
	}
}

// --claude-symlink falls back to the marker form, and a re-run over that
// marker-carrying CLAUDE.md refreshes it instead of refusing (decision 5).
func TestClaudeMDSymlinkFallsBackToMarker(t *testing.T) {
	noSymlinks(t)
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	opts := Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}, ClaudeLink: true}
	res, err := Run(payload, root, "v1", opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasEntryContaining(res.Notes, "CLAUDE.md: symlink unavailable") {
		t.Errorf("fallback not noted; notes: %v", res.Notes)
	}
	claude := filepath.Join(root, "CLAUDE.md")
	if !regularWithMarker(claude) {
		t.Fatal("CLAUDE.md is not a regular file carrying the marker block")
	}
	res, err = Run(payload, root, "v1", opts)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if !regularWithMarker(claude) {
		t.Error("second run lost the marker block")
	}
	// T-143 decision 5: the re-run says why no symlink was made (F4).
	if !hasEntryContaining(res.Notes, "CLAUDE.md is already a regular file with a marker block") {
		t.Errorf("re-run did not note the unhonoured flag; notes: %v", res.Notes)
	}
}

// TestLinkPlaceholders covers a Windows clone that checked committed links
// out as plain files (decision 9): the view placeholder is replaced, the
// CLAUDE.md placeholder is never written to.
func TestLinkPlaceholders(t *testing.T) {
	for _, links := range []bool{true, false} {
		t.Run(map[bool]string{true: "symlinks", false: "no-symlinks"}[links], func(t *testing.T) {
			if !links {
				noSymlinks(t)
			}
			root := t.TempDir()
			payload := os.DirFS(payloadRoot())
			if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
				t.Fatal(err)
			}
			view := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
			claude := filepath.Join(root, "CLAUDE.md")
			if err := os.RemoveAll(view); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(view, []byte(ClaudeSkillTarget), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(claude, []byte(ClaudeMDTarget), 0o644); err != nil {
				t.Fatal(err)
			}

			res, err := Upgrade(payload, root, "v2")
			if err != nil {
				t.Fatalf("Upgrade: %v", err)
			}
			if !hasEntryContaining(res.Notes, "symlink placeholder") {
				t.Errorf("view placeholder replacement not noted; notes: %v", res.Notes)
			}
			fi, err := os.Lstat(view)
			if err != nil {
				t.Fatal(err)
			}
			if links && fi.Mode()&os.ModeSymlink == 0 {
				t.Error("view placeholder was not replaced by a symlink")
			}
			if !links && !IsClaudeViewCopy(root) {
				t.Error("view placeholder was not replaced by a copy")
			}

			if _, err := Run(payload, root, "v2", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
				t.Fatalf("Run over placeholder: %v", err)
			}
			if got, _ := os.ReadFile(claude); string(got) != ClaudeMDTarget {
				t.Errorf("CLAUDE.md placeholder was rewritten: %q", got)
			}

			if _, err := Uninstall(payload, root, UninstallOptions{}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(claude); !os.IsNotExist(err) {
				t.Error("CLAUDE.md placeholder still present after uninstall")
			}
		})
	}
}

// A refresh whose new copy fails keeps the old copy, sentinel and all, and
// leaves no staging or parked directory behind (re-review D2).
func TestClaudeViewRefreshFailureKeepsOldCopy(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a 0o000 file anyway")
	}
	noSymlinks(t)
	root := t.TempDir()
	if _, err := Run(os.DirFS(payloadRoot()), root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	view := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if err := os.WriteFile(filepath.Join(view, "old.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	unreadable := filepath.Join(root, filepath.FromSlash(SkillDir), "SKILL.md")
	if err := os.Chmod(unreadable, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o644) })

	var res Result
	if err := ensureClaudeView(root, &res); err == nil {
		t.Fatal("refresh succeeded with an unreadable payload file")
	}
	if !IsClaudeViewCopy(root) {
		t.Fatal("failed refresh lost the old copy (or its sentinel)")
	}
	mustExist(t, filepath.Join(view, "old.md"))
	for _, kind := range []string{"tmp", "old"} {
		if exists(claudeViewScratch(root, kind)) {
			t.Errorf("failed refresh left %s behind", claudeViewScratch(root, kind))
		}
	}
}

// A refresh killed after parking the old copy leaves no view; the next
// upgrade re-creates it and drops the parked copy, and uninstall clears a
// leftover staging directory (re-review D2).
func TestClaudeViewKilledRefreshRecovers(t *testing.T) {
	noSymlinks(t)
	root := t.TempDir()
	payload := os.DirFS(payloadRoot())
	if _, err := Run(payload, root, "v1", Options{ProjectName: "demo", ProjectPath: ".", Agents: Agents{Claude: true}}); err != nil {
		t.Fatal(err)
	}
	view := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if err := os.Rename(view, claudeViewScratch(root, "old")); err != nil {
		t.Fatal(err)
	}
	if _, err := Upgrade(payload, root, "v2"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !IsClaudeViewCopy(root) {
		t.Fatal("upgrade did not re-create the view after a killed refresh")
	}
	if exists(claudeViewScratch(root, "old")) {
		t.Error("upgrade left the parked copy behind")
	}

	if err := os.MkdirAll(claudeViewScratch(root, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(payload, root, UninstallOptions{}); err != nil {
		t.Fatal(err)
	}
	if exists(claudeViewScratch(root, "tmp")) {
		t.Error("uninstall left a staging directory behind")
	}
}
