package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsePushRefs exercises the pure parser against literal stdin fixtures
// — no git involved, per the shape ParsePushRefs is written for.
func TestParsePushRefs(t *testing.T) {
	zero := strings.Repeat("0", 40)
	sha1 := strings.Repeat("a", 40)
	sha2 := strings.Repeat("b", 40)

	for _, tc := range []struct {
		name  string
		stdin string
		want  []PushRef
	}{
		{
			name:  "new branch: remote sha is all-zero",
			stdin: "refs/heads/feat/T-999-demo " + sha1 + " refs/heads/feat/T-999-demo " + zero + "\n",
			want: []PushRef{
				{LocalRef: "refs/heads/feat/T-999-demo", LocalSHA: sha1, RemoteRef: "refs/heads/feat/T-999-demo", RemoteSHA: zero},
			},
		},
		{
			name:  "deletion: local sha is all-zero",
			stdin: "(delete) " + zero + " refs/heads/feat/T-999-demo " + sha2 + "\n",
			want: []PushRef{
				{LocalRef: "(delete)", LocalSHA: zero, RemoteRef: "refs/heads/feat/T-999-demo", RemoteSHA: sha2},
			},
		},
		{
			name: "several refs in one push",
			stdin: "refs/heads/main " + sha1 + " refs/heads/main " + sha2 + "\n" +
				"refs/heads/feat/T-1-x " + sha2 + " refs/heads/feat/T-1-x " + zero + "\n",
			want: []PushRef{
				{LocalRef: "refs/heads/main", LocalSHA: sha1, RemoteRef: "refs/heads/main", RemoteSHA: sha2},
				{LocalRef: "refs/heads/feat/T-1-x", LocalSHA: sha2, RemoteRef: "refs/heads/feat/T-1-x", RemoteSHA: zero},
			},
		},
		{
			name:  "a malformed line is skipped, not fatal",
			stdin: "this line has too few fields\nrefs/heads/main " + sha1 + " refs/heads/main " + sha2 + "\n",
			want: []PushRef{
				{LocalRef: "refs/heads/main", LocalSHA: sha1, RemoteRef: "refs/heads/main", RemoteSHA: sha2},
			},
		},
		{
			name:  "blank lines are ignored",
			stdin: "\n\nrefs/heads/main " + sha1 + " refs/heads/main " + sha2 + "\n\n",
			want: []PushRef{
				{LocalRef: "refs/heads/main", LocalSHA: sha1, RemoteRef: "refs/heads/main", RemoteSHA: sha2},
			},
		},
		{
			name:  "empty stdin",
			stdin: "",
			want:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePushRefs(strings.NewReader(tc.stdin))
			if err != nil {
				t.Fatalf("ParsePushRefs: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParsePushRefs = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("ref %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// newRemoteRepo creates an isolated bare "remote" and a clone of it, wires the
// clone's origin/HEAD (as a real push would), and returns the clone's root —
// a repository CheckPrePush can measure a real merge-base range against.
func newRemoteRepo(t *testing.T, baseBranch string) (clone string, remoteName string) {
	t.Helper()
	isolate(t)
	remote := t.TempDir()
	mustGit(t, remote, "init", "-q", "--bare", "-b", baseBranch)

	clone = t.TempDir()
	mustGit(t, clone, "init", "-q", "-b", baseBranch)
	mustGit(t, clone, "config", "user.email", "test@example.com")
	mustGit(t, clone, "config", "user.name", "test")
	mustGit(t, clone, "config", "commit.gpgsign", "false")
	mustGit(t, clone, "remote", "add", "origin", remote)
	writeConfig(t, clone, "")
	mustGit(t, clone, "add", "-A")
	mustGit(t, clone, "commit", "-qm", "seed")
	mustGit(t, clone, "push", "-q", "-u", "origin", baseBranch)
	return clone, "origin"
}

// pushRefFor builds the single PushRef a real `git push` would feed on stdin
// for a normal (non-deleting, non-new-remote-ref) push of branch at head,
// where the source and destination name the same branch. dir is unused (the
// ref itself carries every field CheckPrePush needs) and was dropped at
// T-100 (T-082 F8) — call sites pass the config's own root separately via
// loadConfig.
func pushRefFor(t *testing.T, branch, head string) PushRef {
	t.Helper()
	return PushRef{
		LocalRef:  "refs/heads/" + branch,
		LocalSHA:  head,
		RemoteRef: "refs/heads/" + branch,
		RemoteSHA: strings.Repeat("0", 40),
	}
}

// pushRefTo builds the PushRef for a split refspec, where the source names
// srcBranch and the destination names dstBranch — the shape T-100 exists
// for, which pushRefFor cannot express since it assumes the two agree.
func pushRefTo(srcBranch, dstBranch, head string) PushRef {
	return PushRef{
		LocalRef:  "refs/heads/" + srcBranch,
		LocalSHA:  head,
		RemoteRef: "refs/heads/" + dstBranch,
		RemoteSHA: strings.Repeat("0", 40),
	}
}

// TestBranchBeingPushed pins the fallback finding F2 fixed (T-082's first
// review): LocalRef alone cannot name the branch for every push shape.
func TestBranchBeingPushed(t *testing.T) {
	for _, tc := range []struct {
		name       string
		ref        PushRef
		wantBranch string
		wantOK     bool
	}{
		{
			name:       "ordinary push: LocalRef names the branch",
			ref:        PushRef{LocalRef: "refs/heads/feat/T-1-x", RemoteRef: "refs/heads/feat/T-1-x"},
			wantBranch: "feat/T-1-x",
			wantOK:     true,
		},
		{
			// `git push origin HEAD:refs/heads/feat/T-1-x`: LocalRef is the
			// literal "HEAD", which has no ref name of its own.
			name:       "HEAD:refs/heads/... push: falls back to RemoteRef",
			ref:        PushRef{LocalRef: "HEAD", RemoteRef: "refs/heads/feat/T-1-x"},
			wantBranch: "feat/T-1-x",
			wantOK:     true,
		},
		{
			name:   "tag push: neither side is under refs/heads/",
			ref:    PushRef{LocalRef: "refs/tags/v1.0.0", RemoteRef: "refs/tags/v1.0.0"},
			wantOK: false,
		},
		{
			// A HEAD-relative tag push: LocalRef falls back to RemoteRef, which
			// is still outside refs/heads/, so this stays skipped too.
			name:   "HEAD:refs/tags/... push: fallback still outside refs/heads/",
			ref:    PushRef{LocalRef: "HEAD", RemoteRef: "refs/tags/v1.0.0"},
			wantOK: false,
		},
		{
			// `git push origin main:refs/heads/feat/T-1-x`: the false pass under
			// the old LocalRef-first precedence — LocalRef named "main", which
			// is not a feature branch, so the ref was skipped before any range
			// was measured. RemoteRef-only decides this is feat/T-1-x (T-100).
			name:       "main:refs/heads/feat/... push: destination decides, not main",
			ref:        PushRef{LocalRef: "refs/heads/main", RemoteRef: "refs/heads/feat/T-1-x"},
			wantBranch: "feat/T-1-x",
			wantOK:     true,
		},
		{
			// `git push origin feat/T-1-x:refs/heads/main`: the false refusal
			// under the old precedence. branchBeingPushed now names "main", and
			// it is CheckPrePush's onFeatureBranch check — not this function —
			// that lets the push through, because the base branch is
			// bookkeeping's correct destination (T-082 decision 3).
			name:       "feat/...:refs/heads/main push: destination decides, not the source branch",
			ref:        PushRef{LocalRef: "refs/heads/feat/T-1-x", RemoteRef: "refs/heads/main"},
			wantBranch: "main",
			wantOK:     true,
		},
		{
			// A mixed refspec: the source is a branch, but the actual
			// destination is a tag. This is the row that FAILS under the
			// rejected "prefer RemoteRef, fall back to LocalRef" design — that
			// design would resolve LocalRef's "refs/heads/x" and wrongly treat
			// a tag push as a branch push. RemoteRef-only, no fallback, keeps
			// this skipped (decision 1 and decision 2).
			name:   "refs/heads/x:refs/tags/... push: a tag destination stays skipped",
			ref:    PushRef{LocalRef: "refs/heads/x", RemoteRef: "refs/tags/v1.0.0"},
			wantOK: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch, ok := branchBeingPushed(tc.ref)
			if ok != tc.wantOK {
				t.Fatalf("branchBeingPushed(%+v) ok = %v, want %v", tc.ref, ok, tc.wantOK)
			}
			if ok && branch != tc.wantBranch {
				t.Errorf("branchBeingPushed(%+v) = %q, want %q", tc.ref, branch, tc.wantBranch)
			}
		})
	}
}

func TestCheckPrePush(t *testing.T) {
	t.Run("feature branch carrying tickets/ is refused", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		mustGit(t, root, "checkout", "-qb", "feat/T-001-x")
		stageTicket(t, root, "T-001-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-001 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")

		t.Chdir(root)
		var msg bytes.Buffer
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{pushRefFor(t, "feat/T-001-x", head)}, &msg)
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if ok {
			t.Fatal("expected a refusal")
		}
		for _, want := range []string{"feat/T-001-x", "tickets/", "--no-verify"} {
			if !strings.Contains(msg.String(), want) {
				t.Errorf("rejection message missing %q:\n%s", want, msg.String())
			}
		}
	})

	t.Run("HEAD:refs/heads/... push carrying tickets/ is refused (F2)", func(t *testing.T) {
		// git push <remote> HEAD:refs/heads/feat/T-001-x sends LocalRef as the
		// literal "HEAD" — the shape that used to bypass the guard silently.
		root, remote := newRemoteRepo(t, "main")
		mustGit(t, root, "checkout", "-qb", "feat/T-001-x")
		stageTicket(t, root, "T-001-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-001 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")
		ref := PushRef{LocalRef: "HEAD", LocalSHA: head, RemoteRef: "refs/heads/feat/T-001-x", RemoteSHA: strings.Repeat("0", 40)}

		t.Chdir(root)
		var msg bytes.Buffer
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{ref}, &msg)
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if ok {
			t.Fatal("HEAD:refs/heads/... push carrying tickets/ was allowed")
		}
		if !strings.Contains(msg.String(), "feat/T-001-x") {
			t.Errorf("rejection message does not name the branch from RemoteRef:\n%s", msg.String())
		}
	})

	t.Run("a tag push is allowed (neither ref is under refs/heads/)", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		mustGit(t, root, "checkout", "-qb", "feat/T-001-x")
		stageTicket(t, root, "T-001-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-001 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")
		mustGit(t, root, "tag", "v1.0.0", head)
		ref := PushRef{LocalRef: "refs/tags/v1.0.0", LocalSHA: head, RemoteRef: "refs/tags/v1.0.0", RemoteSHA: strings.Repeat("0", 40)}

		t.Chdir(root)
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{ref}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Error("a tag push was refused; tags are never this guard's concern")
		}
	})

	t.Run("feature branch carrying only code is allowed", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		mustGit(t, root, "checkout", "-qb", "feat/T-001-x")
		stageCode(t, root, "code.go")
		mustGit(t, root, "commit", "-qm", "feat: code")
		head := mustGit(t, root, "rev-parse", "HEAD")

		t.Chdir(root)
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{pushRefFor(t, "feat/T-001-x", head)}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Fatal("code-only feature branch push was refused")
		}
	})

	// This is the single most important way to get the guard wrong (decision
	// 3): the base branch itself carries tickets/ paths by design.
	t.Run("the base branch carrying tickets/ is allowed", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		stageTicket(t, root, "T-002-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-002 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")

		t.Chdir(root)
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{pushRefFor(t, "main", head)}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Fatal("pushing the base branch's own bookkeeping was refused")
		}
	})

	t.Run("an unresolvable base is allowed, with one stderr line", func(t *testing.T) {
		isolate(t)
		root := t.TempDir()
		mustGit(t, root, "init", "-q", "-b", "feat/T-001-x")
		mustGit(t, root, "config", "user.email", "test@example.com")
		mustGit(t, root, "config", "user.name", "test")
		mustGit(t, root, "config", "commit.gpgsign", "false")
		writeConfig(t, root, "")
		stageTicket(t, root, "T-001-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-001 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")
		// No remote configured at all: every resolveBase candidate fails.

		t.Chdir(root)
		var msg bytes.Buffer
		ok, err := CheckPrePush(loadConfig(t, root), "origin", []PushRef{pushRefFor(t, "feat/T-001-x", head)}, &msg)
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Error("an unresolvable base blocked the push; it must fail open")
		}
		if strings.Count(msg.String(), "\n") != 1 {
			t.Errorf("expected exactly one stderr line, got:\n%s", msg.String())
		}
		if !strings.Contains(msg.String(), "could not resolve a base") {
			t.Errorf("message does not name the resolution failure:\n%s", msg.String())
		}
	})

	t.Run("tickets/ outside this repository is allowed", func(t *testing.T) {
		isolate(t)
		base := t.TempDir()
		writeConfig(t, base, "")
		child := t.TempDir()
		mustGit(t, child, "init", "-q", "-b", "feat/T-001-x")
		mustGit(t, child, "config", "user.email", "test@example.com")
		mustGit(t, child, "config", "user.name", "test")
		stageCode(t, child, "code.go")
		mustGit(t, child, "commit", "-qm", "seed")
		head := mustGit(t, child, "rev-parse", "HEAD")

		t.Chdir(child)
		ok, err := CheckPrePush(loadConfig(t, base), "origin", []PushRef{pushRefFor(t, "feat/T-001-x", head)}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Error("refused a push in a child repo that holds no tickets/")
		}
	})

	t.Run("a branch deletion is allowed", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		zero := strings.Repeat("0", 40)
		ref := PushRef{LocalRef: "refs/heads/feat/T-001-x", LocalSHA: zero, RemoteRef: "refs/heads/feat/T-001-x", RemoteSHA: strings.Repeat("a", 40)}

		t.Chdir(root)
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{ref}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Error("a branch deletion was refused")
		}
	})

	// destination is a feature branch, source is main: the false pass T-100
	// fixes. Under the old LocalRef-first precedence this was allowed because
	// LocalRef resolved to "main". The rejection must name the destination,
	// feat/T-900-x, not the source.
	t.Run("destination is a feature branch, source is main: refused and named correctly (T-100)", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		stageTicket(t, root, "T-900-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-900 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")
		ref := pushRefTo("main", "feat/T-900-x", head)

		t.Chdir(root)
		var msg bytes.Buffer
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{ref}, &msg)
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if ok {
			t.Fatal("main:refs/heads/feat/T-900-x carrying tickets/ was allowed")
		}
		if !strings.Contains(msg.String(), "branch:  feat/T-900-x") {
			t.Errorf("rejection does not name the destination branch:\n%s", msg.String())
		}
		if strings.Contains(msg.String(), "branch:  main") {
			t.Errorf("rejection wrongly named the source branch:\n%s", msg.String())
		}
		// The range line honestly names what was diffed (decision 6): since
		// LocalRef (main) disagrees with the destination branch, printing
		// "...feat/T-900-x" would name a ref that need not exist locally, so
		// it falls back to a short SHA instead.
		if !strings.Contains(msg.String(), "..."+shortSHA(head)) {
			t.Errorf("range line does not fall back to the short SHA on a split refspec:\n%s", msg.String())
		}
		if strings.Contains(msg.String(), "...feat/T-900-x") {
			t.Errorf("range line named a ref (the destination branch) that need not exist locally:\n%s", msg.String())
		}
	})

	// destination is main, source is a feature branch: the false refusal
	// T-100 fixes. The base branch is bookkeeping's correct destination
	// (T-082 decision 3) — this is the one place the rule exists to send
	// bookkeeping to, not merely a safe direction to err in.
	t.Run("destination is main, source is a feature branch: allowed (T-100, T-082 decision 3)", func(t *testing.T) {
		root, remote := newRemoteRepo(t, "main")
		mustGit(t, root, "checkout", "-qb", "feat/T-901-x")
		stageTicket(t, root, "T-901-x.md")
		mustGit(t, root, "commit", "-qm", "board: T-901 demo")
		head := mustGit(t, root, "rev-parse", "HEAD")
		ref := pushRefTo("feat/T-901-x", "main", head)

		t.Chdir(root)
		ok, err := CheckPrePush(loadConfig(t, root), remote, []PushRef{ref}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("CheckPrePush: %v", err)
		}
		if !ok {
			t.Fatal("feat/T-901-x:refs/heads/main carrying bookkeeping was refused")
		}
	})
}

// TestCheckPrePushInALinkedWorktree is the pre-push half of finding F8 (the
// other half, for pre-commit, is TestPreCommitInALinkedWorktree in
// hook_test.go): both rules run through gitHere, not gitAt, and a hardcoded
// root would inspect the main worktree's repository instead of the one being
// pushed.
func TestCheckPrePushInALinkedWorktree(t *testing.T) {
	root, remote := newRemoteRepo(t, "main")

	wt := filepath.Join(t.TempDir(), "wt")
	mustGit(t, root, "worktree", "add", "-q", "-b", "feat/T-001-x", wt)
	if err := os.MkdirAll(filepath.Join(wt, "tickets", "1-to-do"), 0o755); err != nil {
		t.Fatal(err)
	}
	stageTicket(t, wt, "T-001-x.md")
	mustGit(t, wt, "commit", "-qm", "board: T-001 demo")
	head := mustGit(t, wt, "rev-parse", "HEAD")

	t.Chdir(wt)
	var msg bytes.Buffer
	ok, err := CheckPrePush(loadConfig(t, wt), remote, []PushRef{pushRefFor(t, "feat/T-001-x", head)}, &msg)
	if err != nil {
		t.Fatalf("CheckPrePush: %v", err)
	}
	if ok {
		t.Errorf("guard missed the violation inside a linked worktree:\n%s", msg.String())
	}
}
