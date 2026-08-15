// Package atomicfile provides a single primitive: replace a file's contents
// without a reader ever observing a half-written or truncated result.
//
// This is T-101's Task 1 — lifted verbatim (logic and documented trade-offs
// both) from internal/config's former writePreservingMode, which was
// unexported and config-only even though nothing about it is config-specific.
// internal/config now calls WriteFile directly; its own
// verifyOnlyPayloadVersion parse-back guard stays in that package, because it
// is config-specific and not part of the write primitive.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile replaces an existing path atomically, keeping its current
// permissions; creating a new path instead respects the umask, the same as
// os.WriteFile (see the note below on why the two cases differ).
//
// A symlinked target is followed rather than replaced: renaming onto the
// link would turn it into a regular file and leave the real target stale,
// which is the opposite of an atomic update.
//
// T-026 (D5): every failure below is reported against path — the file the
// caller actually asked to write — with an actionable cause, never a bare
// errno naming the throwaway temp file. The three shapes measured in
// practice: an unwritable parent directory (CreateTemp), and a file the
// rename cannot replace under either an ACL deny-delete or `chflags uchg`
// (Rename).
//
// The contract (T-069 D3), stated once so it does not have to be re-found: this
// function preserves the permission bits and follows symlinks; it fsyncs the
// temp file before renaming, so the failure window narrows from "rename is
// atomic" to "rename is atomic *and* crash-durable" — without the fsync, a
// crash between write and the filesystem's own writeback could still leave a
// zero-length file behind the rename. It does **not** preserve: hardlink
// identity (`os.Rename` severs a hardlink — `nlink` drops from 2 to 1, and the
// other name is stranded at the old contents); ownership (the temp file
// inherits the *directory's* group, not the original file's); extended
// attributes (Finder tags, Spotlight comments, quarantine state, and any other
// xattr); or mode bits outside `Perm()` (setuid/setgid/sticky, e.g. `2644` →
// `0644`). It also does not refuse a read-only (`0444`) file — create-temp +
// rename needs only a writable directory, where `os.WriteFile` used to fail on
// the file's own permissions. Every one of these was verified real during
// config's T-018 re-review; none is reachable through normal use, and the fix
// cost is disproportionate to the risk, so they stay declined rather than
// silently re-litigated by a future reader.
//
// When path does not exist yet, there is no mode to preserve and nothing
// there yet for a partial write to corrupt, so this takes the plain,
// umask-respecting path instead (config's T-069 rework, finding F1): os.Chmod
// is not umask-filtered the way os.WriteFile's own perm argument is, so
// hard-coding a mode for the create-temp+rename path would make file
// *creation* ignore the umask — the exact defect class this function exists
// to close on existing files, reintroduced at creation time.
func WriteFile(path string, data []byte) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	fi, statErr := os.Stat(path)
	if statErr != nil {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("%s: could not create the file (is the directory writable?): %w", path, err)
		}
		return nil
	}
	mode := fi.Mode().Perm()
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("%s: could not create a temporary file in %s to write the update safely (is the directory writable?): %w",
			path, dir, err)
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }() // no-op once the rename succeeds
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s: writing the update: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s: writing the update: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s: writing the update: %w", path, err)
	}
	if err := os.Chmod(name, mode); err != nil {
		return fmt.Errorf("%s: could not preserve the file's permissions on the update: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("%s: could not replace the file with the update — it may be read-only, immutable (e.g. chflags uchg), or under a permission that denies deletion: %w",
			path, err)
	}
	return nil
}
