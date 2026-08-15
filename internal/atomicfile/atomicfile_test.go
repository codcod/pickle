package atomicfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// hardlinkCount returns fi's link count, or 0 if the platform's os.FileInfo
// does not expose one (e.g. Windows), in which case the caller should skip
// rather than assert anything.
func hardlinkCount(t *testing.T, fi os.FileInfo) uint64 {
	t.Helper()
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return uint64(st.Nlink)
}

// writeFixture writes body to dir/name and returns its path.
func writeFixture(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestWriteFileFollowsSymlink is WriteFile's own symlink-resolution
// regression: a symlinked target is followed rather than replaced, so the
// link itself survives and the real target picks up the update. Config's
// Save and SetPayloadVersionInPlace each carry their own integration-level
// version of this (config_test.go's TestSaveFollowsSymlink and
// TestSetPayloadVersionInPlaceFollowsSymlink) through their own callers; this
// one pins the behavior in the primitive itself.
func TestWriteFileFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	real := writeFixture(t, dir, "real.toml", "old\n")
	link := filepath.Join(dir, "link.toml")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := WriteFile(link, []byte("new\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("link.toml is no longer a symlink; WriteFile replaced the link instead of following it")
	}
	got, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Errorf("real.toml content = %q, want %q (the symlink target should have been updated)", got, "new\n")
	}
}

// TestWriteFileSeversHardlink pins one of T-069 D3's documented and declined
// behaviours instead of leaving it to be re-found: os.Rename cannot preserve
// a hardlink's identity, so a second name pointing at the same inode is left
// behind at the old contents once the first is atomically replaced.
func TestWriteFileSeversHardlink(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "a.toml", "old\n")
	other := filepath.Join(dir, "other-name.toml")
	if err := os.Link(path, other); err != nil {
		t.Skipf("hardlinks unavailable: %v", err)
	}

	if err := WriteFile(path, []byte("updated content\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fi, err := os.Stat(other)
	if err != nil {
		t.Fatal(err)
	}
	if nlink := hardlinkCount(t, fi); nlink == 0 {
		t.Skip("platform does not report a usable link count")
	} else if nlink != 1 {
		t.Errorf("other-name.toml nlink = %d, want 1 (severed from the renamed-over path)", nlink)
	}
	got, err := os.ReadFile(other)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "updated content\n" {
		t.Error("the second hardlinked name picked up the update; WriteFile is documented to sever hardlinks, not share them")
	}
}

// TestWriteFileRewritesReadOnlyFile pins the other declined behaviour:
// create-temp-then-rename only needs a writable *directory*, so a read-only
// (0444) file is rewritten anyway — unlike the os.WriteFile it replaced,
// which would have failed on the file's own permission bit. This documents
// the decline (a successful rewrite), not a regression.
func TestWriteFileRewritesReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "a.toml", "old\n")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(path, []byte("updated\n")); err != nil {
		t.Fatalf("WriteFile: %v (expected to succeed even though the file is read-only)", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "updated\n" {
		t.Errorf("content = %q, want %q", got, "updated\n")
	}
}

// TestWriteFileCreateRespectsUmask pins the T-069 rework fix (finding F1):
// when the target does not exist yet, WriteFile must not hard-code a mode
// via Chmod (which ignores the umask) the way it does for an existing
// file's rename-over path — it must behave like plain
// os.WriteFile(path, data, 0o644) and let the umask narrow that, exactly as
// pickle install's other generated files do.
func TestWriteFileCreateRespectsUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	dir := t.TempDir()
	path := filepath.Join(dir, "new.toml")
	if err := WriteFile(path, []byte("x\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want 600 (umask 077 applied to the requested 0644)", got)
	}
}

// TestWriteFileRewriteIgnoresUmask is TestWriteFileCreateRespectsUmask's
// counterpart for the *existing* path: once a file is there, its own mode is
// preserved regardless of the umask in effect at rewrite time — the umask
// only ever governs creation.
func TestWriteFileRewriteIgnoresUmask(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "a.toml", "old\n")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	if err := WriteFile(path, []byte("updated\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o640 {
		t.Errorf("file mode = %o, want 640 (preserved despite umask 077 at rewrite time)", got)
	}
}
