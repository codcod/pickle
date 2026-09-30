package changelog

import (
	"regexp"
	"strings"
)

// Commit is one line of `git log --format=%H%x09%P%x09%cs%x09%s`: what
// FindMerge needs to tell a merge commit from a ticket commit and to cite it
// (T-140).
type Commit struct {
	SHA     string
	Parents int
	Date    string // committer date, YYYY-MM-DD
	Subject string
}

// ParseLog reads the output of `git log --format=%H%x09%P%x09%cs%x09%s`,
// skipping any line that does not have all four fields.
func ParseLog(out string) []Commit {
	var log []Commit
	for _, line := range strings.Split(out, "\n") {
		f := strings.SplitN(line, "\t", 4)
		if len(f) != 4 {
			continue
		}
		log = append(log, Commit{SHA: f[0], Parents: len(strings.Fields(f[1])), Date: f[2], Subject: f[3]})
	}
	return log
}

// mrRefRE finds the first "#N" / "!N" token in a merge commit's subject —
// GitHub's "Merge pull request #95 from …" or a GitLab subject carrying "!N".
var mrRefRE = regexp.MustCompile(`[#!]\d+`)

// FindMerge finds where ticket id was merged in log (newest first, as `git log`
// prints it), and returns the commit to cite, its MR ref ("PR #N" / "MR !N",
// or "" when there is none) and whether one was found (T-140 decisions 3–4).
// A merge commit whose subject names the ticket's branch wins; otherwise the
// newest commit classifying as child-project code for id — the trailing
// "(<ID>)" form — covers squash, fast-forward and kept-history merges.
func FindMerge(log []Commit, id, branchPrefix string, prefixes []string) (Commit, string, bool) {
	// The branch name must end at the id or continue with "-": feat/T-50-x
	// never names T-5. A quote or whitespace also ends it (GitLab's
	// "Merge branch 'feat/T-5' into 'main'").
	branchRE := regexp.MustCompile(regexp.QuoteMeta(branchPrefix+id) + `(?:[-'\s]|$)`)
	for _, c := range log {
		if c.Parents > 1 && branchRE.MatchString(c.Subject) {
			return c, mrRef(mrRefRE.FindString(c.Subject)), true
		}
	}
	p := newIDPatterns(prefixes)
	for _, c := range log {
		if kind, got := classifySubject(c.Subject, p); kind == ChildProject && got == id {
			tok := strings.Trim(strings.TrimSpace(prTokenRE.FindString(strings.TrimSpace(c.Subject))), "()")
			return c, mrRef(tok), true
		}
	}
	return Commit{}, "", false
}

// mrRef renders "#N" as "PR #N" and "!N" as "MR !N", the forms the rules'
// merge line uses.
func mrRef(tok string) string {
	switch {
	case strings.HasPrefix(tok, "#"):
		return "PR " + tok
	case strings.HasPrefix(tok, "!"):
		return "MR " + tok
	}
	return ""
}

// remoteRE splits an https:// or scp-style git@ remote into host and
// owner/repo path.
var remoteRE = regexp.MustCompile(`^(?:https://|git@)([^/:]+)[/:](.+?)(?:\.git)?/?$`)

// CommitURL links sha on a github.com or gitlab.com origin, or returns "" for
// any other host or an unparseable remote (T-140 decision 6).
func CommitURL(remote, sha string) string {
	m := remoteRE.FindStringSubmatch(strings.TrimSpace(remote))
	if m == nil {
		return ""
	}
	switch m[1] {
	case "github.com":
		return "https://github.com/" + m[2] + "/commit/" + sha
	case "gitlab.com":
		return "https://gitlab.com/" + m[2] + "/-/commit/" + sha
	}
	return ""
}

// ShortSHA is the 7-character form a merge line cites.
func ShortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// MergeLine renders the History line to record for c:
// "<date> — merged to <base> (<ref>, <short sha>[, <url>])", dropping an
// empty ref or url.
func MergeLine(c Commit, base, ref, url string) string {
	parts := []string{ShortSHA(c.SHA)}
	if ref != "" {
		parts = append([]string{ref}, parts...)
	}
	if url != "" {
		parts = append(parts, url)
	}
	return c.Date + " — merged to " + base + " (" + strings.Join(parts, ", ") + ")"
}
