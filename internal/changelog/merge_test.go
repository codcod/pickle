package changelog

import "testing"

func TestFindMerge(t *testing.T) {
	merge := func(sha, subj string) Commit { return Commit{SHA: sha, Parents: 2, Date: "2026-09-29", Subject: subj} }
	plain := func(sha, subj string) Commit { return Commit{SHA: sha, Parents: 1, Date: "2026-09-28", Subject: subj} }
	prefixes := []string{"T"}

	cases := []struct {
		name    string
		log     []Commit
		wantSHA string
		wantRef string
	}{
		{"github merge commit", []Commit{merge("m1", "Merge pull request #95 from codcod/feat/T-5-x")}, "m1", "PR #95"},
		{"gitlab merge commit", []Commit{merge("m1", "Merge branch 'feat/T-5' into 'main'")}, "m1", ""},
		{"squash", []Commit{plain("s1", "feat(cli): x (T-5) (#31)")}, "s1", "PR #31"},
		{"squash gitlab", []Commit{plain("s1", "feat(cli): x (T-5) (!7)")}, "s1", "MR !7"},
		{"plain ticket commit", []Commit{plain("c1", "fix: y (T-5)")}, "c1", ""},
		{"bookkeeping never matches", []Commit{plain("b1", "board: T-5 done")}, "", ""},
		{"T-50 is not T-5", []Commit{merge("m1", "Merge pull request #9 from o/feat/T-50-x"), plain("c1", "fix: y (T-50)")}, "", ""},
		{"merge commit wins over ticket commit", []Commit{plain("c2", "fix: z (T-5)"), merge("m1", "Merge pull request #95 from o/feat/T-5-x"), plain("c1", "feat: y (T-5)")}, "m1", "PR #95"},
		{"newest ticket commit wins", []Commit{plain("c2", "fix: z (T-5)"), plain("c1", "feat: y (T-5)")}, "c2", ""},
		{"newest merge wins", []Commit{merge("m2", "Merge pull request #32 from o/feat/T-5-x"), merge("m1", "Merge pull request #31 from o/feat/T-5-x")}, "m2", "PR #32"},
		{"unregistered prefix", []Commit{plain("c1", "fix: y (X-5)")}, "", ""},
		{"branch name in a non-merge commit", []Commit{plain("c1", "revert feat/T-5-x")}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "T-5"
			if tc.name == "unregistered prefix" {
				id = "X-5"
			}
			c, ref, ok := FindMerge(tc.log, id, "feat/", prefixes)
			if ok != (tc.wantSHA != "") || c.SHA != tc.wantSHA || ref != tc.wantRef {
				t.Errorf("FindMerge = (%q, %q, %v), want (%q, %q)", c.SHA, ref, ok, tc.wantSHA, tc.wantRef)
			}
		})
	}
}

func TestCommitURL(t *testing.T) {
	for remote, want := range map[string]string{
		"https://github.com/codcod/pickle.git": "https://github.com/codcod/pickle/commit/abc1234",
		"git@github.com:codcod/pickle.git":     "https://github.com/codcod/pickle/commit/abc1234",
		"https://gitlab.com/grp/sub/repo":      "https://gitlab.com/grp/sub/repo/-/commit/abc1234",
		"git@gitlab.com:grp/repo.git":          "https://gitlab.com/grp/repo/-/commit/abc1234",
		"https://git.example.com/o/r.git":      "",
		"/tmp/some/bare":                       "",
	} {
		if got := CommitURL(remote, "abc1234"); got != want {
			t.Errorf("CommitURL(%q) = %q, want %q", remote, got, want)
		}
	}
}

func TestMergeLineAndParseLog(t *testing.T) {
	log := ParseLog("2daf1f1aaaa\tp1 p2\t2026-09-29\tMerge pull request #95 from o/feat/T-134-x\nbad line\nc1ffff0\tp1\t2026-09-28\tfix: y (T-1)")
	if len(log) != 2 || log[0].Parents != 2 || log[1].Parents != 1 || log[1].Subject != "fix: y (T-1)" {
		t.Fatalf("ParseLog = %+v", log)
	}
	if got, want := MergeLine(log[0], "main", "PR #95", "https://x/commit/2daf1f1"), "2026-09-29 — merged to main (PR #95, 2daf1f1, https://x/commit/2daf1f1)"; got != want {
		t.Errorf("MergeLine = %q, want %q", got, want)
	}
	if got, want := MergeLine(log[1], "main", "", ""), "2026-09-28 — merged to main (c1ffff0)"; got != want {
		t.Errorf("MergeLine = %q, want %q", got, want)
	}
}
