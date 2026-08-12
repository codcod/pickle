package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
//
// T-094 closes the range (--until, default HEAD, resolving the default
// --since relative to it rather than to HEAD — see runChangelogCheck) and
// quiets the exclusion list (a one-line summary by default, --show-excluded
// for the full subjects), so a past --section stays auditable and a
// release-sized report isn't dominated by bookkeeping noise.

const changelogCheckUsage = `usage: pickle changelog check [--since <ref>] [--until <ref>] [--changelog <path>] [--section <name>] [--show-excluded]`

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
	since := fs.String("since", "", "start of the shipped-commit range, exclusive (default: the last git tag reachable from --until)")
	until := fs.String("until", "HEAD", "end of the shipped-commit range, inclusive")
	changelogPath := fs.String("changelog", "CHANGELOG.md", "path to the changelog, relative to the project root")
	section := fs.String("section", "Unreleased", `the "## [<name>]" section to check against (e.g. a version like "0.5.0")`)
	showExcluded := fs.Bool("show-excluded", false, "print every excluded board: commit's subject instead of a one-line summary")
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

	untilRef := *until
	sinceRef := *since
	if sinceRef == "" {
		// Resolved relative to --until, not HEAD (T-094 decision 3): with a
		// HEAD-relative default, "--until v0.5.0" alone would describe HEAD's
		// own last tag, not v0.5.0's predecessor, and could produce the empty
		// range v0.5.0..v0.5.0 — reporting a false "no candidates" pass. The
		// "^" is what makes a tag-shaped --until name the range *ending*
		// there; with the default --until HEAD this is unchanged from today
		// whenever HEAD itself isn't exactly a tag.
		tag, err := vcs.Output(root, "describe", "--tags", "--abbrev=0", untilRef+"^")
		if err != nil {
			return errf("changelog check: no --since given and no git tag found reachable from %s^: %v", untilRef, err)
		}
		sinceRef = tag
	}

	subjects, err := commitSubjects(root, sinceRef, untilRef)
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

	printChangelogCheckReport(root, cfg.FlowName(), sinceRef, untilRef, *changelogPath, *showExcluded, res)
	return exitOK // advisory only — decision 2, never a gate
}

// commitSubjects returns `git log --format=%s <since>..<until>`, one subject
// per line, oldest-omitted/newest-first exactly as git orders them. It is
// the I/O half of decision 3: "what shipped" comes from commit subjects, not
// ticket History lines; --until (T-094 decision 2) closes the range so a
// past --section stays auditable after later commits land.
func commitSubjects(root, since, until string) ([]string, error) {
	out, err := vcs.Output(root, "log", "--format=%s", since+".."+until)
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
func printChangelogCheckReport(root, flowName, since, until, changelogPath string, showExcluded bool, res changelog.Result) {
	byID := make(map[string]*ticket.Ticket)
	if tickets, _ := ticket.LoadAll(flow.ForName(flowName), root); tickets != nil {
		for _, t := range tickets {
			byID[t.ID] = t
		}
	}

	fmt.Printf("changelog check: %s..%s, against %s's %q section\n", since, until, changelogPath, res.Section)
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
	printExclusions(res.Excluded, showExcluded)
	printUnclassified(res.Unclassified)
}

// printExclusions renders the excluded board: bookkeeping commits. Decision
// 7 (T-093) requires them visible so a convention drift shows up rather than
// silently under-reporting; decision 4 (T-094) is the escape hatch decision
// 7 explicitly allowed ("or offer behind a flag") for the noise that
// unconditional printing caused — a release-sized run had far more exclusion
// lines than candidates. Default: one summary line naming every id covered,
// plus a count of any commit that parsed no id at all (the loudest possible
// symptom of a convention drift, so it must never be the thing the summary
// hides). --show-excluded still prints every subject, as before T-094.
func printExclusions(excluded []changelog.Exclusion, showExcluded bool) {
	if len(excluded) == 0 {
		return
	}
	ids := make(map[string]bool)
	noID := 0
	for _, ex := range excluded {
		if ex.ID == "" {
			noID++
			continue
		}
		ids[ex.ID] = true
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)

	switch {
	case len(sorted) == 0:
		fmt.Printf("  excluded %d board: bookkeeping commit(s), none with a parsable ticket id (--show-excluded for subjects)\n", len(excluded))
	case noID == 0:
		fmt.Printf("  excluded %d board: bookkeeping commit(s) covering %s (--show-excluded for subjects)\n",
			len(excluded), strings.Join(sorted, ", "))
	default:
		fmt.Printf("  excluded %d board: bookkeeping commit(s) covering %s (+%d with no ticket id; --show-excluded for subjects)\n",
			len(excluded), strings.Join(sorted, ", "), noID)
	}
	if showExcluded {
		for _, ex := range excluded {
			fmt.Printf("    %s\n", ex.Subject)
		}
	}
}

// printUnclassified renders the T-094 safety-net list: a subject that names
// a ticket in parentheses but matches neither the board: nor the
// child-project convention (a revert, most plausibly). Always printed in
// full — unlike exclusions, this list is meant to be short (rules §0's own
// conventions cover the ordinary cases), so summarising it would hide the
// one signal it exists to surface.
func printUnclassified(unclassified []changelog.Exclusion) {
	if len(unclassified) == 0 {
		return
	}
	fmt.Printf("  %d commit(s) mention a ticket but match neither convention — check whether they shipped:\n", len(unclassified))
	for _, u := range unclassified {
		fmt.Printf("    %s\n", u.Subject)
	}
}
