// Package lock serialises concurrent pickle processes over one ticket tree
// using an advisory flock on a dedicated lock file — never over the ticket
// tree's own content, and never as a substitute for internal/move's D7
// crash-ordering decision.
//
// What this protects: two or more pickle processes (the CLI, `pickle serve`,
// an agent shelling out) reading and writing the same tickets/ tree on the
// same machine at the same time. WithExclusive serialises writers against
// each other and against readers; WithShared lets any number of readers run
// together but excludes a writer.
//
// What this does not protect: a crash mid-write. flock is released the
// instant its holding process exits or dies — by design, so there is no
// stale-lock recovery story to write — which means a process that crashes
// between two writes leaves exactly the same partial state it would without
// a lock. internal/move's D7 comment already accepts a duplicate-id window
// there for exactly that reason; this package closes the window between
// live pickle processes, not the crash window.
//
// Accepted limitation (T-101 decision 8): a single page render in `pickle
// serve` takes this lock twice — once around loading the ticket tree
// (internal/serve's handler.load) and again around the separate audit
// traversal that builds its health banner (internal/serve's buildHealth) —
// rather than once for the whole request. Between the two acquisitions a
// writer can run, so the rendered board and its health banner can
// occasionally reflect two different instants of the tree in the same page.
// This is cosmetic, not a correctness gap: nothing is corrupted or
// half-written, and the page already re-polls every 5 seconds, so the
// mismatch self-heals on the next refresh. Threading one audit result through
// every handler to close this window was judged not worth the coupling it
// would add across internal/serve's view construction.
//
// Multi-machine or networked filesystems are out of scope: flock's semantics
// over NFS are unreliable across hosts, and pickle assumes a single-machine
// tree the way the rest of the codebase already does (see internal/config's
// same-machine assumptions).
//
// Targets are darwin and linux only (.goreleaser.yaml); syscall.Flock exists
// on both, so this needs no build tags and no new dependency. If a Windows
// target is ever added, a build-tagged stub (e.g. LockFileEx) is that
// ticket's problem — this package does not attempt one.
package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// acquireTimeout bounds how long WithExclusive and WithShared will retry
// before giving up. A command that hangs forever behind a wedged holder is
// worse than one that refuses with an actionable message.
const acquireTimeout = 10 * time.Second

const retryInterval = 50 * time.Millisecond

// lockPath returns the path of the lock file for the ticket tree rooted at
// root. When root/.git is a directory, the lock lives at
// root/.git/pickle-tree.lock — invisible to `git status` and to every
// consumer of the tree, which is the point: a lock file inside tickets/
// would show up as untracked noise in every installed project. When
// root/.git is not a directory (a worktree or submodule's ".git" is a file
// pointing elsewhere, or there is no .git at all), the lock instead lives in
// the OS temp directory, named after a hash of root's absolute path so
// distinct trees never collide and the same tree always resolves to the
// same file. The gitdir pointer inside a worktree/submodule ".git" file is
// deliberately not parsed — the temp-dir fallback is simpler and just as
// correct for this purpose.
func lockPath(root string) string {
	gitPath := filepath.Join(root, ".git")
	if fi, err := os.Stat(gitPath); err == nil && fi.IsDir() {
		return filepath.Join(gitPath, "pickle-tree.lock")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(os.TempDir(), "pickle-tree-"+hex.EncodeToString(sum[:])[:16]+".lock")
}

// WithExclusive runs fn while holding an exclusive lock on the ticket tree
// rooted at root, creating the lock file if it does not already exist.
// Callers should acquire it before the first read of the tree that fn's
// write decision depends on, and release it only after the write (and any
// dependent post-write step, such as a regenerated index) is complete —
// a lock around the write alone leaves every check it guards racing.
//
// The file is created with O_CREATE and is never removed: removing a lock
// file out from under a concurrent waiter is how mutual exclusion gets
// lost, so the file — once it exists — is permanent, harmless, empty-content
// bookkeeping. Only the flock held on it is transient.
func WithExclusive(root string, fn func() error) error {
	f, err := os.OpenFile(lockPath(root), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("could not open the tree lock file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := acquire(f, syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}

// WithShared runs fn while holding a shared (read) lock on the ticket tree
// rooted at root. It excludes a concurrent WithExclusive holder but allows
// any number of concurrent WithShared callers to proceed together.
//
// Unlike WithExclusive, WithShared never creates the lock file: it opens
// without O_CREATE, and if the file is absent it runs fn immediately,
// unlocked. The tree cannot be mid-write if no writer has ever run in this
// tree, so there is nothing to wait for — and just as importantly, a reader
// that created the lock file would turn every read-only command (including
// `pickle serve`, which must never write anything) into a writer of one.
func WithShared(root string, fn func() error) error {
	f, err := os.OpenFile(lockPath(root), os.O_RDONLY, 0)
	if os.IsNotExist(err) {
		return fn()
	}
	if err != nil {
		return fmt.Errorf("could not open the tree lock file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := acquire(f, syscall.LOCK_SH); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}

// acquire retries a non-blocking flock (LOCK_EX or LOCK_SH, both bitwise-or'd
// with LOCK_NB by this function) for up to acquireTimeout before giving up.
// LOCK_NB rather than a bare blocking flock is what makes the bound
// enforceable at all: a plain LOCK_EX/LOCK_SH blocks the calling goroutine
// (and, since flock is a syscall, the OS thread under it) with no way to
// time out.
func acquire(f *os.File, how int) error {
	deadline := time.Now().Add(acquireTimeout)
	fd := int(f.Fd())
	for {
		err := syscall.Flock(fd, how|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return fmt.Errorf("could not lock %s: %w", f.Name(), err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf(
				"timed out after %s waiting for the tree lock (%s) — another pickle command or `pickle serve` may be running against this tree",
				acquireTimeout, f.Name())
		}
		time.Sleep(retryInterval)
	}
}
