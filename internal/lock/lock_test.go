package lock

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestWithExclusiveSerialises proves two concurrent WithExclusive holders
// never run fn at the same time: each appends its own start/end marker to a
// shared, unsynchronised slice, and if the critical sections overlapped the
// interleaving below would not hold.
func TestWithExclusiveSerialises(t *testing.T) {
	dir := t.TempDir()
	var mu sync.Mutex // guards the trace slice only, not the thing under test
	var trace []string
	record := func(s string) {
		mu.Lock()
		trace = append(trace, s)
		mu.Unlock()
	}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		n := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithExclusive(dir, func() error {
				record("start")
				time.Sleep(50 * time.Millisecond)
				record("end")
				return nil
			})
			if err != nil {
				t.Errorf("goroutine %d: WithExclusive: %v", n, err)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(trace) != 4 {
		t.Fatalf("trace = %v, want 4 entries", trace)
	}
	// Serialised means each "start" is immediately followed by its own
	// "end" before the other holder's "start" appears — i.e. the trace is
	// exactly [start end start end], never [start start end end].
	if !(trace[0] == "start" && trace[1] == "end" && trace[2] == "start" && trace[3] == "end") {
		t.Errorf("trace = %v, want [start end start end] (overlapping critical sections)", trace)
	}
}

// TestWithSharedAllowsConcurrentReaders proves a shared lock does not
// exclude another shared lock: both holders should be inside fn at the same
// time, which this test proves with a barrier both must reach before
// either returns.
func TestWithSharedAllowsConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	// A shared lock never creates the file (WithShared runs unlocked if
	// absent), so create it up front the way a prior WithExclusive call
	// would have.
	if err := os.WriteFile(lockPath(dir), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	both := make(chan struct{}, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithShared(dir, func() error {
				both <- struct{}{}
				<-release
				return nil
			})
			if err != nil {
				t.Errorf("WithShared: %v", err)
			}
		}()
	}

	// Both goroutines must reach the barrier before either is released —
	// if a shared lock excluded another shared lock, this would deadlock
	// and the test would time out.
	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-both:
		case <-timeout:
			t.Fatal("timed out waiting for both shared holders to run concurrently")
		}
	}
	close(release)
	wg.Wait()
}

// TestWithSharedOnAbsentLockFileCreatesNothing is the guarantee that keeps
// serve a strict non-writer: a reader must never bring the lock file into
// existence, only a writer (WithExclusive) may.
//
// T-101 rework (F3): dir must contain a .git directory, matching decision 4's
// primary case, so lockPath(dir) resolves *inside* dir (at
// dir/.git/pickle-tree.lock) rather than falling back to a path under
// os.TempDir() — outside anything this test lists. Without that, the
// before/after comparison below cannot fail no matter what WithShared does:
// proved by mutation (temporarily making WithShared unconditionally create
// the lock file left this test green before this fix). The assertion below
// checks dir/.git specifically, which is exactly where a reader creating the
// lock file would leave evidence.
func TestWithSharedOnAbsentLockFileCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if lockPath(dir) != filepath.Join(gitDir, "pickle-tree.lock") {
		t.Fatalf("lockPath(dir) = %q, want it inside dir/.git so this test can actually observe a created lock file", lockPath(dir))
	}

	before, err := os.ReadDir(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("dir/.git not empty before test: %v", before)
	}

	ran := false
	if err := WithShared(dir, func() error { ran = true; return nil }); err != nil {
		t.Fatalf("WithShared: %v", err)
	}
	if !ran {
		t.Error("fn was not run")
	}

	after, err := os.ReadDir(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Errorf("WithShared created files in dir/.git, where the lock file would land: %v", after)
	}
}

// TestWithExclusiveExcludesWithShared proves the write/read exclusion
// direction: a WithShared call cannot proceed while a WithExclusive holder
// is inside fn.
func TestWithExclusiveExcludesWithShared(t *testing.T) {
	dir := t.TempDir()
	holding := make(chan struct{})
	releaseWriter := make(chan struct{})
	sharedDone := make(chan error, 1)

	go func() {
		_ = WithExclusive(dir, func() error {
			close(holding)
			<-releaseWriter
			return nil
		})
	}()

	<-holding
	go func() {
		sharedDone <- WithShared(dir, func() error { return nil })
	}()

	select {
	case <-sharedDone:
		t.Fatal("WithShared returned while WithExclusive still held the lock")
	case <-time.After(150 * time.Millisecond):
		// expected: the shared reader is still blocked
	}
	close(releaseWriter)

	select {
	case err := <-sharedDone:
		if err != nil {
			t.Errorf("WithShared: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WithShared never returned after the exclusive holder released")
	}
}

// TestLockPathUsesGitDirWhenPresent pins decision 4's primary case.
func TestLockPathUsesGitDirWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := lockPath(dir)
	want := filepath.Join(dir, ".git", "pickle-tree.lock")
	if got != want {
		t.Errorf("lockPath = %q, want %q", got, want)
	}
}

// TestLockPathFallsBackToTempDir pins decision 4's fallback: no .git
// directory (including the worktree/submodule case where .git is a file)
// resolves to a stable, tree-specific path under the OS temp dir.
func TestLockPathFallsBackToTempDir(t *testing.T) {
	dir := t.TempDir()
	got := lockPath(dir)
	if filepath.Dir(got) != filepath.Clean(os.TempDir()) {
		t.Errorf("lockPath = %q, want a path under %q", got, os.TempDir())
	}
	if got != lockPath(dir) {
		t.Errorf("lockPath is not stable across calls for the same root")
	}

	other := t.TempDir()
	if lockPath(dir) == lockPath(other) {
		t.Errorf("lockPath collided for two distinct roots: %q", lockPath(dir))
	}
}
