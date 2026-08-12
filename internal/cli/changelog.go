package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codcod/pickle/internal/changelog"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
	"github.com/codcod/pickle/internal/vcs"
)

// Changelog mechanics. `changelog check` (T-093) is a read-only report: is
// every ticket that shipped since the last release named in CHANGELOG.md?
// Mentions are all it checks — a ticket whose own file records a decision to
// get no entry is still reported, and the report points at that file so the
// reader can settle it (decision 5, no exemption mechanism). It is advisory
// only (decision 2) — it always exits 0, even with
// candidates, and is never wired into `board audit`, CI, or `ticket move`: a
// merged ticket missing an entry is not an error, because the entry may
// legitimately be written any time before the release.

const changelogCheckUsage = `usage: pickle changelog check [--since <ref>] [--changelog <path>] [--section <name>]`

func runChangelog(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle changelog <check> ...")
		return exitUsage
	}
	switch args[0] {
	case "check":
		return runChangelogCheck(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle changelog: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

func runChangelogCheck(args []string) int {
	fs := flag.NewFlagSet("changelog check", flag.ContinueOnError)
	since := fs.String("since", "", "start of the shipped-commit range, exclusive (default: the last git tag)")
	changelogPath := fs.String("changelog", "CHANGELOG.md", "path to the changelog, relative to the project root")
	section := fs.String("section", "Unreleased", `the "## [<name>]" section to check against (e.g. a version like "0.5.0")`)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, changelogCheckUsage) }
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, changelogCheckUsage)
		return exitUsage
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	root := cfg.Root()

	sinceRef := *since
	if sinceRef == "" {
		tag, err := vcs.Output(root, "describe", "--tags", "--abbrev=0")
		if err != nil {
			return errf("changelog check: no --since given and no git tag found: %v", err)
		}
		sinceRef = tag
	}

	subjects, err := commitSubjects(root, sinceRef)
	if err != nil {
		return errf("changelog check: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, *changelogPath))
	if err != nil {
		return errf("changelog check: %v", err)
	}

	res, err := changelog.Check(subjects, string(data), *section)
	if err != nil {
		return errf("changelog check: %v", err)
	}

	printChangelogCheckReport(root, cfg.FlowName(), sinceRef, *changelogPath, res)
	return exitOK // advisory only — decision 2, never a gate
}

// commitSubjects returns `git log --format=%s <since>..HEAD`, one subject per
// line, oldest-omitted/newest-first exactly as git orders them. It is the
// I/O half of decision 3: "what shipped" comes from commit subjects, not
// ticket History lines.
func commitSubjects(root, since string) ([]string, error) {
	out, err := vcs.Output(root, "log", "--format=%s", since+"..HEAD")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// printChangelogCheckReport renders res. Resolving a candidate to its ticket
// file (decision 5 — "point at each candidate ticket file so that judgement
// is one click away") is best-effort: a ticket that fails to load must not
// hide the report itself, so a lookup miss just omits the path instead of
// erroring.
func printChangelogCheckReport(root, flowName, since, changelogPath string, res changelog.Result) {
	byID := make(map[string]*ticket.Ticket)
	if tickets, _ := ticket.LoadAll(flow.ForName(flowName), root); tickets != nil {
		for _, t := range tickets {
			byID[t.ID] = t
		}
	}

	fmt.Printf("changelog check: since %s, against %s's %q section\n", since, changelogPath, res.Section)
	if len(res.Candidates) == 0 {
		fmt.Println("  no candidates — every shipped ticket is mentioned")
	} else {
		fmt.Printf("  %d candidate(s) shipped but not named in %q:\n", len(res.Candidates), res.Section)
		for _, id := range res.Candidates {
			if t, ok := byID[id]; ok {
				fmt.Printf("    %s  (%s/%s — check for a recorded decision before adding an entry)\n",
					id, t.Dir, filepath.Base(t.Path))
			} else {
				fmt.Printf("    %s  (no ticket file found — check for a recorded decision before adding an entry)\n", id)
			}
		}
	}
	if len(res.Excluded) > 0 {
		fmt.Printf("  excluded %d board: bookkeeping commit(s) from \"shipped\" (relies on the convention holding):\n", len(res.Excluded))
		for _, ex := range res.Excluded {
			fmt.Printf("    %s\n", ex.Subject)
		}
	}
}
