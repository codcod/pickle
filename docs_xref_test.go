package main

// docs_xref_test.go fails the build the moment the user manual (docs/user-manual.adoc
// plus its include::-d pages) gains a dead cross-reference, an inter-document xref:
// form, or a page nothing includes (T-067). All three were live, unnoticed defects at
// filing: `just docs-check` (snowball check, a render-and-discard pass) is blind to a
// dead <<anchor>> because asciidoctor renders "[no-such-anchor-xyz]" into the output
// and moves on rather than failing, and T-057 shipped two xref:<file>.adoc#... links
// that render fine as HTML but point at a dead per-chapter file once split into
// PDF/EPUB (T-057 finding N3) — asciidoctor happily resolves the reference, it is only
// wrong once the single-book rendering pipeline runs.
//
// This runs as a plain `go test ./...` test (already in CI's build-test job — no new
// CI job is needed) and is additionally wired into `just docs-check` so that recipe's
// own claim to catch a dead cross-reference is true for a human/agent who runs only it.
//
// Anchor matching is explicit-[#id]-only (optionally carrying a role, [#id.role], or
// reftext, [#id,reftext]): the checker does not compute asciidoctor's auto-generated
// section-id algorithm. Every <<...>> target in this manual resolves to an explicit
// anchor today (TestDocsXrefsResolve passing IS that assertion, continuously, on every
// run) — if a future xref is ever written to rely on an auto-generated id, add an
// explicit [#id] to that heading rather than teaching this checker asciidoctor's
// slugger. (Two lines in the live tree, docs/user-manual/configuration.adoc and
// docs/user-manual/concepts/multi-project.adoc, contain the literal text `[[project]]`
// describing TOML array-of-tables syntax in backtick-quoted prose — not an AsciiDoc
// [[id]] anchor definition. A naive `grep '\[\['` over the tree will surface them; they
// are not anchors and this checker's [#id]-only pattern does not match them.)
//
// T-115 hardened the inter-document side and closed two edges that would otherwise
// have made the gate quietly narrower than it looked:
//
//   - Every inter-document spelling the manual can legally contain now routes to the
//     same "the book is one document" failure: xref:file.adoc[...] (with or without an
//     anchor), the extensionless "natural xref" xref:name#anchor[...], link:file.adoc[...]
//     (link: requires the .adoc extension — it is also how the manual writes external
//     URLs, so an extensionless rule would flag every one of those instead), and the
//     <<file.adoc#anchor>> shorthand — which looks intra-document but is not. Each
//     occurrence's failure message quotes the construct as it was actually written
//     (docsXrefOccurrence.spelling), so a link: or <<>> site is never misreported as an
//     "xref:..." one.
//   - AsciiDoc's two documented ways to *show* a cross-reference without making one —
//     \<<x>> and +<<x>>+ — are masked (docsMaskEscapedXrefs) before either detector or
//     the coverage invariant sees the line, so neither false-positives on a literal
//     example. The mask replaces only the escaped span with an equal-length run of
//     filler, so it never widens a match, never shifts a later column on the same
//     line, and a real <<x>> earlier or later on that line is still found.
//   - An unterminated literal block (an opened "----"/"...."/"++++" with no matching
//     close) is a hard error naming the opening line, not a silent truncation: before
//     T-115, everything after it simply stopped being scanned, and a dead reference on
//     one of those dropped lines was invisible to `go test` — exactly what CI runs.
//
// Do not add a code-span / backtick-counting exemption to docsRefShapedSiteRe (the
// permissive coverage pattern) to quiet a code-example false positive. T-067 shipped
// exactly that and reverted it: counting backticks per line misfired on ordinary
// wrapped prose in cli-reference.adoc, where a continuation line happens to open with
// a closing backtick, making the count odd and silently exempting a real
// <<cmd-doctor>> from coverage — a guard that quietly stops covering things is the
// defect class this file exists to prevent. To show a <<...>> example in prose
// instead, put it in a literal block (----): both the scanner and the coverage
// invariant already skip those.
import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	docsBookMaster    = "docs/user-manual.adoc"
	docsUserManualDir = "docs/user-manual"
)

// The book measured 13 files / 30 anchors / 86 `<<…>>` occurrences / 24 distinct
// targets at refinement (2026-08-20). Those figures are recorded here as history
// and asserted by nothing — see the amendment to T-067 decision 5.
//
// They were briefly asserted as floors, to answer the question the resolution
// check cannot ask of itself: "did I actually look at the book?" That failed
// twice over. Floors expire, because they turn documentation growth into slack:
// after one ordinary page is added, a pattern can silently stop matching a whole
// spelling and still clear the old number. And floors are blind upward, which is
// the dangerous direction for anchors — a loosened anchor pattern that swallows
// prose inflates the set and makes dead cross-references *resolve*.
//
// What replaces them is structural, so it neither expires nor cries wolf:
//
//   - assertEveryXrefSiteScanned — every literal "<<" in the book must be consumed
//     by docsXrefTargetRe. An unmatched site means the pattern narrowed, at any
//     book size.
//   - TestDocsScannerPatternsMatchWhatTheyClaim — positive and negative fixtures
//     pinning each pattern, so loosening is caught as precisely as narrowing.
//   - TestDocsUserManualHasNoOrphanPages — already catches a mis-walked include
//     tree, since a page the walk stops reaching becomes an orphan.
//
// Together those cover collapse, inflation and mis-walking without a single
// number that a normal docs commit can invalidate.

var (
	docsIncludeLineRe = regexp.MustCompile(`^include::([^\[\]]+)\[`)
	// docsAnchorLineRe matches an explicit anchor line, allowing the legal attribute
	// suffixes AsciiDoc permits alongside an id — a role (".role") and/or reftext
	// (",reftext") — while still requiring the line be nothing else (item 3, T-115).
	// It does not accept the legacy [[id]] spelling, a mid-line anchor, or trailing
	// prose after the closing bracket.
	docsAnchorLineRe   = regexp.MustCompile(`^\[#([A-Za-z0-9_-]+)(?:\.[A-Za-z0-9_-]+)?(?:,[^\]]+)?\]\s*$`)
	docsXrefTargetRe   = regexp.MustCompile(`<<([A-Za-z0-9_-]+)(?:,[^>]*)?>>`)
	docsInterDocXrefRe = regexp.MustCompile(`xref:([^\[\s]+\.adoc)[#\[]`)

	// docsInterDocXrefBareRe catches the extensionless "natural xref" spelling —
	// xref:document-id#anchor[text], with no .adoc suffix — which asciidoctor
	// resolves by document id rather than filename. It requires a non-empty name
	// before "#" (excluding the legal intra-document xref:#local-anchor[text], whose
	// target is empty) and a non-empty anchor after it (item 1, T-115).
	docsInterDocXrefBareRe = regexp.MustCompile(`xref:([A-Za-z0-9][A-Za-z0-9_-]*)#([^\[\s]+)\[`)

	// docsInterDocLinkRe catches link:file.adoc[...] / link:file.adoc#id[...]. Unlike
	// xref:, link: is also how the manual writes external URLs (link:https://...[]),
	// so the .adoc suffix is required rather than optional — an extensionless link:
	// rule would flag every external link (T-115 decision 6).
	docsInterDocLinkRe = regexp.MustCompile(`link:([^\[\s]+\.adoc)[#\[]`)

	// docsInterDocAngleRe catches the <<file.adoc#anchor>> / <<file.adoc#anchor,text>>
	// shorthand. docsXrefTargetRe (the intra-document pattern) already fails to match
	// it — "." and "#" are outside its target character class — but
	// docsRefShapedSiteRe (the permissive coverage pattern) does, so left unrouted it
	// used to fail the coverage invariant with a message blaming the scanner instead
	// of naming the real defect: this is an inter-document reference in <<>> clothing
	// (item 2, T-115 decision 5).
	docsInterDocAngleRe = regexp.MustCompile(`<<([^>,\s]+\.adoc)#([^>,\s]+)(?:,[^>]*)?>>`)

	// docsEscapedXrefBackslashRe and docsEscapedXrefPassthroughRe match AsciiDoc's two
	// documented ways to show a literal cross-reference without making one: a leading
	// backslash, or wrapping in a passthrough "+...+" pair (item 4, T-115).
	docsEscapedXrefBackslashRe   = regexp.MustCompile(`\\<<[^>]*>>`)
	docsEscapedXrefPassthroughRe = regexp.MustCompile(`\+<<[^>]*>>\+`)
)

// docsMaskEscapedXrefs replaces each of AsciiDoc's two literal-reference escapes with
// an equal-length run of "_", so file:line:col offsets used to correlate scanner
// output with source sites are unaffected (decision 3) while nothing downstream reads
// the escaped span as a cross-reference. Only the matched span itself is replaced, so
// a real <<x>> elsewhere on the same line is untouched.
//
// The filler is not inert, and the mask is not purely subtractive: "_" is inside
// docsXrefTargetRe's target class, so an escape nested inside a live reference —
// "<<\<<x>>>>" — masks to "<<______>>" and is then read as a reference to a phantom
// anchor "______". Such input is adversarial rather than plausible, and it still fails
// loudly (reported unresolved, on the right line) rather than silently, so it is
// recorded here rather than worked around; the fix, if it ever matters, is a filler
// character outside that class rather than a special case (T-115 review, finding F5b).
func docsMaskEscapedXrefs(line string) string {
	for _, re := range []*regexp.Regexp{docsEscapedXrefBackslashRe, docsEscapedXrefPassthroughRe} {
		for _, loc := range re.FindAllStringIndex(line, -1) {
			line = line[:loc[0]] + strings.Repeat("_", loc[1]-loc[0]) + line[loc[1]:]
		}
	}
	return line
}

// bookFiles recursively follows include:: directives starting at master, resolving
// each target relative to the including file's own directory (standard AsciiDoc
// include resolution), and returns the deduplicated, visitation-ordered list of files
// that make up the assembled book — master itself first.
func bookFiles(master string) ([]string, error) {
	var order []string
	seen := map[string]bool{}

	var walk func(path string) error
	walk = func(path string) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if seen[abs] {
			return nil
		}
		seen[abs] = true
		order = append(order, path)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		dir := filepath.Dir(path)
		for _, line := range strings.Split(string(data), "\n") {
			m := docsIncludeLineRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			target := filepath.Join(dir, m[1])
			if err := walk(target); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(master); err != nil {
		return nil, err
	}
	return order, nil
}

// docsXrefOccurrence is one <<target>>, xref:..., or link:... match, with enough
// location to produce a teaching-style failure message (mirrors payloadLintFinding's
// file:line + reason shape in payload_lint_test.go).
//
// spelling carries the construct as it was actually written (e.g. "xref:file.adoc",
// "link:file.adoc#id", "<<file.adoc#id,text>>"). For an inter-document occurrence,
// the failure message must quote this rather than assuming every occurrence was
// written as "xref:..." — a link: or <<>> occurrence routed to the same message
// would otherwise be misreported as a spelling the contributor never used (T-115
// decision 5). target keeps its original meaning: the anchor id for an
// intra-document <<id>> match, or the referenced document (as named in the source)
// for an inter-document one.
type docsXrefOccurrence struct {
	file     string
	line     int
	col      int // byte offset of the match within the line
	target   string
	spelling string
}

// scanBook reads every file in files and returns: the set of explicit anchor ids
// ([#id] lines), every <<id>>/<<id,text>> occurrence, and every xref:<file>.adoc...
// occurrence — each with its file:line:column.
//
// Only prose is scanned (docsProseLines): asciidoctor does not resolve a cross-
// reference written inside a listing block either, so reporting one would be a false
// positive with no legal fix — the contributor could neither make it resolve nor
// escape it. Nothing in the manual relies on this today (no anchor or reference sits
// inside a literal block), so it changes no current result; it makes the two checks
// agree by construction rather than by coincidence.
func scanBook(files []string) (anchors map[string]bool, xrefs, interDoc []docsXrefOccurrence, err error) {
	anchors = map[string]bool{}
	for _, f := range files {
		data, rerr := os.ReadFile(f)
		if rerr != nil {
			return nil, nil, nil, fmt.Errorf("reading %s: %w", f, rerr)
		}
		lines, perr := docsProseLines(string(data))
		if perr != nil {
			return nil, nil, nil, fmt.Errorf("reading %s: %w", f, perr)
		}
		for _, pl := range lines {
			lineNo, line := pl.no, pl.text
			if m := docsAnchorLineRe.FindStringSubmatch(line); m != nil {
				anchors[m[1]] = true
			}
			for _, m := range docsXrefTargetRe.FindAllStringSubmatchIndex(line, -1) {
				target := line[m[2]:m[3]]
				xrefs = append(xrefs, docsXrefOccurrence{
					file: f, line: lineNo, col: m[0], target: target,
					spelling: fmt.Sprintf("<<%s>>", target),
				})
			}
			for _, m := range docsInterDocXrefRe.FindAllStringSubmatchIndex(line, -1) {
				target := line[m[2]:m[3]]
				interDoc = append(interDoc, docsXrefOccurrence{
					file: f, line: lineNo, col: m[0], target: target,
					spelling: fmt.Sprintf("xref:%s", target),
				})
			}
			for _, m := range docsInterDocXrefBareRe.FindAllStringSubmatchIndex(line, -1) {
				name, anchor := line[m[2]:m[3]], line[m[4]:m[5]]
				interDoc = append(interDoc, docsXrefOccurrence{
					file: f, line: lineNo, col: m[0], target: name,
					spelling: fmt.Sprintf("xref:%s#%s", name, anchor),
				})
			}
			for _, m := range docsInterDocLinkRe.FindAllStringSubmatchIndex(line, -1) {
				target := line[m[2]:m[3]]
				interDoc = append(interDoc, docsXrefOccurrence{
					file: f, line: lineNo, col: m[0], target: target,
					spelling: fmt.Sprintf("link:%s", target),
				})
			}
			for _, m := range docsInterDocAngleRe.FindAllStringSubmatchIndex(line, -1) {
				refDoc := line[m[2]:m[3]]
				interDoc = append(interDoc, docsXrefOccurrence{
					file: f, line: lineNo, col: m[0], target: refDoc,
					spelling: line[m[0]:m[1]], // the full "<<file.adoc#anchor,text>>" as written
				})
			}
		}
	}
	return anchors, xrefs, interDoc, nil
}

// docsRefShapedSiteRe is a deliberately permissive superset of docsXrefTargetRe: it
// matches anything shaped like a cross-reference, whatever spelling. The coverage
// invariant works by difference — a site this matches that the strict pattern did not
// produce is a reference the scanner has stopped reading.
//
// Requiring a closing ">>" is what keeps it quiet on code: a shell heredoc
// (cat <<EOF) and a C++ stream insertion (cout << x) have no ">>", so they are not
// reference-shaped at all. That is a property of the syntax rather than a guess about
// where the "<<" sits, which is why it replaced an inline-code exemption based on
// counting backticks per line. That heuristic misfired on ordinary wrapped prose:
// cli-reference.adoc has a paragraph whose continuation line opens with a closing
// backtick, making the count odd and silently exempting a real <<cmd-doctor>> from
// coverage. A guard that quietly stops covering things is the defect this ticket
// exists to remove, so the heuristic is gone rather than tuned.
var docsRefShapedSiteRe = regexp.MustCompile(`<<[^>\s][^>]*>>`)

// docsLiteralBlockDelim reports whether a trimmed line delimits an AsciiDoc block
// whose contents are literal — four or more repeats of '-' (listing), '.' (literal)
// or '+' (passthrough) — returning which character, so a block closes only on its own
// delimiter. Tracking the kind matters: a toggle flipped by any delimiter treats a
// "----" nested inside a "...." block as a close, and every line after it in that file
// silently stops being checked.
func docsLiteralBlockDelim(trimmed string) (byte, bool) {
	if len(trimmed) < 4 {
		return 0, false
	}
	c := trimmed[0]
	if c != '-' && c != '.' && c != '+' {
		return 0, false
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != c {
			return 0, false
		}
	}
	return c, true
}

// docsLine is one line of scannable prose, carrying its 1-based number so findings
// keep pointing at the right place in the file after literal blocks are dropped.
//
// text is not necessarily verbatim source: docsProseLines masks AsciiDoc's literal-
// reference escapes in it (docsMaskEscapedXrefs), preserving byte offsets but not
// content, so a failure message that echoes a slice of it may show filler where the
// source had "\<<x>>" or "+<<x>>+".
type docsLine struct {
	no   int
	text string
}

// docsProseLines returns the lines of content outside any literal block, with
// AsciiDoc's literal-reference escapes masked out — the single definition of "text a
// reader sees as a live cross-reference", shared by scanBook and the coverage
// invariant so the two cannot drift into disagreeing about what is exempt. The escape
// mask lives here rather than in each caller for the same reason the literal-block
// logic does: one definition, read by both (decision 4, T-115).
//
// It errors if a literal block is still open at EOF, naming the line where it was
// opened: without this, an unterminated "----" silently drops every line after it
// from both readers at once, and a dead reference on one of those lines is invisible
// to `go test` — exactly what CI runs (item 9, T-115). `just docs-check` does catch
// it (snowball fails an unterminated listing block), but CI does not run snowball.
func docsProseLines(content string) ([]docsLine, error) {
	var out []docsLine
	var openDelim byte
	var openedAt int
	for i, line := range strings.Split(content, "\n") {
		if c, ok := docsLiteralBlockDelim(strings.TrimSpace(line)); ok {
			switch {
			case openDelim == 0:
				openDelim = c
				openedAt = i + 1
			case openDelim == c:
				openDelim = 0
			}
			continue
		}
		if openDelim != 0 {
			continue
		}
		out = append(out, docsLine{no: i + 1, text: docsMaskEscapedXrefs(line)})
	}
	if openDelim != 0 {
		return nil, fmt.Errorf("unterminated literal block opened at line %d", openedAt)
	}
	return out, nil
}

// assertEveryXrefSiteScanned is the invariant that replaced the count floors: every
// reference-shaped site in the book's prose must appear in what scanBook actually
// returned. It answers the question the resolution check cannot ask of itself — "did
// I look at every cross-reference?" — without depending on how big the book is.
//
// It compares against scanBook's own output, by file:line:column, rather than re-running
// the regex here. Re-running it would only ever prove the pattern still matches, which
// left the scanner unguarded: making scanBook take just the first match per line passed
// every test while a second, dead reference on a live line went unreported (review
// finding N1). Checking the output means any way of losing an occurrence is caught,
// pattern or scanner alike.
func assertEveryXrefSiteScanned(t *testing.T, files []string, xrefs, interDoc []docsXrefOccurrence) {
	t.Helper()

	// A site counts as covered if it appears in either result: <<file.adoc#id>> is
	// routed to interDoc, not xrefs (T-115 decision 5), and counting only xrefs would
	// report every such site as a phantom coverage gap on top of its real report.
	scanned := map[string]bool{}
	for _, x := range xrefs {
		scanned[fmt.Sprintf("%s:%d:%d", x.file, x.line, x.col)] = true
	}
	for _, x := range interDoc {
		scanned[fmt.Sprintf("%s:%d:%d", x.file, x.line, x.col)] = true
	}

	var missed []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		lines, perr := docsProseLines(string(data))
		if perr != nil {
			t.Fatalf("reading %s: %v", f, perr)
		}
		for _, pl := range lines {
			for _, loc := range docsRefShapedSiteRe.FindAllStringIndex(pl.text, -1) {
				if !scanned[fmt.Sprintf("%s:%d:%d", f, pl.no, loc[0])] {
					missed = append(missed, fmt.Sprintf("%s:%d: %s", f, pl.no, pl.text[loc[0]:loc[1]]))
				}
			}
		}
	}
	if len(missed) == 0 {
		return
	}

	sort.Strings(missed)
	t.Errorf("%d reference-shaped site(s) the scanner did not report:\n\n%s\n\n"+
		"    Each of these looks like a cross-reference but is absent from scanBook's "+
		"results, so nothing checked whether it resolves. Either docsXrefTargetRe stopped "+
		"matching a spelling the docs use, or scanBook stopped reporting every match.\n"+
		"    If a site above is deliberately not a reference, it still has to resolve — the "+
		"resolution check reads it too. Give it an id that exists, or move the example into "+
		"a literal block (----), which this check and that one both skip.",
		len(missed), strings.Join(missed, "\n"))
}

// docsUnresolvedXref reports whether x — an intra-document <<target>> occurrence —
// fails to resolve to any known anchor. It is shared by TestDocsXrefsResolve and
// TestDocsXrefCheckerCatchesTheFieldFindings so inverting the real comparison turns
// both fixtures red instead of leaving the one that re-implemented it green (item 8,
// T-115).
func docsUnresolvedXref(anchors map[string]bool, x docsXrefOccurrence) bool {
	return !anchors[x.target]
}

// TestDocsXrefsResolve walks the real docs/user-manual.adoc book and fails listing
// every <<target>> that does not resolve to any [#target] anchor in the assembled
// book — the core check this ticket exists to add (T-067).
//
// It first asserts the scan actually covered every cross-reference site (review
// findings F1 and F13): the resolution check below is a loop over what was matched,
// so on a narrowed pattern it would otherwise pass while verifying less than it
// appears to — or, on an empty scan, nothing at all.
func TestDocsXrefsResolve(t *testing.T) {
	files, err := bookFiles(docsBookMaster)
	if err != nil {
		t.Fatalf("walking book includes from %s: %v", docsBookMaster, err)
	}
	anchors, xrefs, interDoc, err := scanBook(files)
	if err != nil {
		t.Fatal(err)
	}

	assertEveryXrefSiteScanned(t, files, xrefs, interDoc)

	var bad []string
	for _, x := range xrefs {
		if docsUnresolvedXref(anchors, x) {
			bad = append(bad, fmt.Sprintf(
				"%s:%d: <<%s>> does not resolve to any [#%s] anchor in the assembled book\n"+
					"    add [#%s] above the intended heading, fix the xref target, or if this "+
					"is meant to show a literal cross-reference rather than make one, escape it "+
					"(\\<<%s>> or +<<%s>>+)",
				x.file, x.line, x.target, x.target, x.target, x.target, x.target))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Fatalf("%d unresolved cross-reference(s):\n\n%s", len(bad), strings.Join(bad, "\n\n"))
	}
}

// TestDocsNoInterDocumentXrefForm walks the real book and fails listing every
// xref:<file>.adoc[...] occurrence. The manual is one assembled book
// (docs/user-manual.adoc + include::), so every cross-file reference must be
// <<anchor>>: asciidoctor resolves xref:<file>.adoc#id[] happily today (it renders as
// an HTML link into the same page), but once split into PDF/EPUB each included file
// becomes its own chapter with no standalone <file>.pdf/<file>.epub artifact, so the
// link is dead in both of the manual's actual distribution formats. T-057 shipped two
// such links (fixed in a7e2ada) and `just docs-check` passed both times — this is the
// half of the ticket with a proven miss (T-057 finding N3).
func TestDocsNoInterDocumentXrefForm(t *testing.T) {
	files, err := bookFiles(docsBookMaster)
	if err != nil {
		t.Fatalf("walking book includes from %s: %v", docsBookMaster, err)
	}
	_, _, interDoc, err := scanBook(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(interDoc) == 0 {
		return
	}

	var bad []string
	for _, x := range interDoc {
		base := strings.TrimSuffix(x.target, ".adoc")
		bad = append(bad, fmt.Sprintf(
			"%s:%d: %s... targets another document — asciidoctor "+
				"resolves it happily as HTML, but the PDF/EPUB build splits each included file "+
				"into its own chapter with no standalone %s.pdf/%s.epub artifact, so the link "+
				"is dead in both real output formats (T-057 finding N3)\n"+
				"    use <<anchor>> instead — the book is one document, not many",
			x.file, x.line, x.spelling, base, base))
	}
	sort.Strings(bad)
	t.Fatalf("%d inter-document reference(s):\n\n%s", len(bad), strings.Join(bad, "\n\n"))
}

// TestDocsUserManualHasNoOrphanPages walks every *.adoc under docs/user-manual/ and
// fails naming any file bookFiles() never reaches from docs/user-manual.adoc — the
// exact failure mode that motivated this ticket at filing (a page committed with no
// include:: line, silently never rendered and never xref-checked).
func TestDocsUserManualHasNoOrphanPages(t *testing.T) {
	files, err := bookFiles(docsBookMaster)
	if err != nil {
		t.Fatalf("walking book includes from %s: %v", docsBookMaster, err)
	}
	included := map[string]bool{}
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			t.Fatalf("abs %s: %v", f, err)
		}
		included[abs] = true
	}

	var orphans []string
	err = filepath.WalkDir(docsUserManualDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".adoc" {
			return nil
		}
		abs, aerr := filepath.Abs(path)
		if aerr != nil {
			return aerr
		}
		if !included[abs] {
			orphans = append(orphans, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", docsUserManualDir, err)
	}
	if len(orphans) == 0 {
		return
	}

	sort.Strings(orphans)
	var b strings.Builder
	fmt.Fprintf(&b, "%d orphan page(s) under %s: no include:: in %s reaches them\n\n",
		len(orphans), docsUserManualDir, docsBookMaster)
	for _, o := range orphans {
		fmt.Fprintf(&b, "%s\n    add an include:: line for it in %s (or in a page it already "+
			"includes)\n\n", o, docsBookMaster)
	}
	t.Fatal(b.String())
}

// TestDocsScannerPatternsMatchWhatTheyClaim pins each scanner pattern with positive
// and negative fixtures. Counts could only ever catch a pattern matching *less*, and
// only until the next docs commit restored the slack (review findings F13, F15); a
// fixture table catches narrowing and loosening alike, at any book size, and says
// which pattern broke instead of only that some number moved.
//
// The negative rows are the ones with teeth. `[[project]]` in backticked prose is
// real text in this manual describing TOML's array-of-tables syntax: an anchor
// pattern loosened to accept the legacy [[id]] spelling swallows it as an anchor
// named "project", and every dead <<project>> reference then resolves against it.
func TestDocsScannerPatternsMatchWhatTheyClaim(t *testing.T) {
	cases := []struct {
		name    string
		re      *regexp.Regexp
		match   []string
		noMatch []string
	}{
		{
			name: "anchor definitions",
			re:   docsAnchorLineRe,
			match: []string{
				"[#the-flow]", "[#cmd-hooks]", "[#a_b-c9]", "[#id]   ",
				"[#id,reftext]", // reftext attribute form (item 3)
				"[#id.role]",    // role attribute form (item 3)
			},
			noMatch: []string{
				"  *Per child — the `\\[[project]]` array*", // TOML prose, not an anchor (F15)
				"Each `\\[[project]]` entry in `pickle.toml`",
				"[[legacy]]",           // legacy spelling: unused here, deliberately not accepted
				"prose [#id] mid-line", // an anchor is a line of its own
				"[#id] trailing prose",
				"[#id,]", // degenerate: comma with no reftext
			},
		},
		{
			name:  "cross-reference targets",
			re:    docsXrefTargetRe,
			match: []string{"<<the-flow>>", "<<lifecycle,the lifecycle>>", "see <<a-b_c9>> here"},
			noMatch: []string{
				"<<>>",
				"<< >>",
				"a << b",
			},
		},
		{
			name: "inter-document xref: forms",
			re:   docsInterDocXrefRe,
			// Both spellings must stay caught: with an anchor (#) and without (F14).
			match: []string{
				"xref:cli-reference.adoc#cmd-hooks[hooks]",
				"xref:cli-reference.adoc[the CLI reference]",
				"xref:../proposals/thing.adoc#x[y]",
			},
			noMatch: []string{
				"<<cmd-hooks>>",
				"xref:#local-anchor[text]", // no .adoc: an intra-document xref, legal
			},
		},
		{
			// The extensionless "natural xref" spelling — item 1, the group's only
			// genuine silent hole at filing.
			name: "inter-document xref: forms (extensionless)",
			re:   docsInterDocXrefBareRe,
			match: []string{
				"xref:cli-reference#cmd-hooks[hooks]",
			},
			noMatch: []string{
				"xref:#local-anchor[text]",                 // empty target: legal intra-document (decision 6)
				"xref:cli-reference.adoc#cmd-hooks[hooks]", // already covered by the .adoc-suffixed pattern
			},
		},
		{
			// link: requires the .adoc extension — link: is also how the manual writes
			// external URLs, so an extensionless rule would flag every one (decision 6).
			name: "inter-document link: forms",
			re:   docsInterDocLinkRe,
			match: []string{
				"link:cli-reference.adoc#id[x]",
				"link:cli-reference.adoc[x]",
			},
			noMatch: []string{
				"link:https://example.com/x[text]",                           // external URL, no .adoc
				"Keep the short SHA even when you add the link: the board's", // decision 7: real prose
			},
		},
		{
			// The <<file.adoc#anchor>> shorthand is inter-document clothing, not the
			// intra-document form: routed here for the right failure message instead
			// of the misleading "scanner did not report this site" (item 2, decision 5).
			name: "inter-document <<file.adoc#anchor>> shorthand",
			re:   docsInterDocAngleRe,
			match: []string{
				"<<cli-reference.adoc#cmd-hooks>>",
				"<<cli-reference.adoc#cmd-hooks,hooks>>",
			},
			noMatch: []string{
				"<<cmd-hooks>>",          // no .adoc: the ordinary intra-document form
				"<<cli-reference.adoc>>", // no anchor at all
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, s := range c.match {
				if !c.re.MatchString(s) {
					t.Errorf("pattern narrowed: %q should match but does not — "+
						"references written this way are going unchecked", s)
				}
			}
			for _, s := range c.noMatch {
				if c.re.MatchString(s) {
					t.Errorf("pattern loosened: %q should not match but does — "+
						"a false positive here, or a phantom anchor that makes dead "+
						"references resolve", s)
				}
			}
		})
	}
}

// TestDocsScannerDoesNotTreatLegacyBracketProseAsAnchor is item 6's negative fixture:
// the pin above is on docsAnchorLineRe alone, which does not protect scanBook itself.
// A legacy [[id]] match added *inside scanBook* — independent of the regex — would
// swallow configuration.adoc's real backticked `\[[project]]` TOML-array prose as an
// anchor named "project", and a planted dead <<project>> would then resolve against
// it and pass every existing test (T-115).
func TestDocsScannerDoesNotTreatLegacyBracketProseAsAnchor(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "page.adoc")
	docsMustWriteFixture(t, page,
		"*Per child — the `\\[[project]]` array* (one entry per connected child):\n")

	anchors, _, _, err := scanBook([]string{page})
	if err != nil {
		t.Fatal(err)
	}
	if anchors["project"] {
		t.Error("expected scanBook to not treat the `\\[[project]]` TOML prose as an " +
			`anchor named "project" — only [#id] lines are anchors`)
	}
}

// TestDocsProseLineSelection pins the literal-block machinery. It earns fixtures of
// its own because scanBook now depends on it: anything it wrongly calls a block stops
// being checked at all, silently. Broadening the delimiter test to any "--" prefix,
// for instance, swallows every AsciiDoc open block and guts the checker while leaving
// the whole suite green (review finding N3).
func TestDocsProseLineSelection(t *testing.T) {
	t.Run("delimiters recognised", func(t *testing.T) {
		for _, s := range []string{"----", "....", "++++", "-----", "........"} {
			if _, ok := docsLiteralBlockDelim(s); !ok {
				t.Errorf("%q should be a literal-block delimiter", s)
			}
		}
	})

	t.Run("non-delimiters rejected", func(t *testing.T) {
		// "--" is AsciiDoc's open block and "====" an example block: neither makes its
		// contents literal, so treating them as such would hide real prose.
		for _, s := range []string{"--", "---", "----x", "x----", "[source,sh]", "====", "", "..."} {
			if _, ok := docsLiteralBlockDelim(s); ok {
				t.Errorf("%q should not be a literal-block delimiter", s)
			}
		}
	})

	t.Run("a block closes only on its own delimiter", func(t *testing.T) {
		// The "----" inside the "...." block must not close it. A kind-blind toggle
		// would reopen here and silently drop every later line in the file (N4).
		got, err := docsProseLines("before\n....\n----\n....\nafter\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var texts []string
		for _, l := range got {
			texts = append(texts, l.text)
		}
		if strings.Join(texts, ",") != "before,after," {
			t.Errorf("expected prose [before after \"\"], got %q", texts)
		}
	})

	t.Run("line numbers survive dropped blocks", func(t *testing.T) {
		got, err := docsProseLines("one\n----\nhidden\n----\nfive\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) < 2 || got[len(got)-2].no != 5 || got[len(got)-2].text != "five" {
			t.Errorf(`expected "five" to keep line number 5, got %+v`, got)
		}
	})

	t.Run("unterminated block errors naming the opening line", func(t *testing.T) {
		got, err := docsProseLines("one\n----\nhidden\nmore\n")
		if err == nil {
			t.Fatalf("expected an error for an unterminated block, got lines %+v", got)
		}
		if !strings.Contains(err.Error(), "line 2") {
			t.Errorf("expected the error to name the opening line (2), got %q", err.Error())
		}
	})

	t.Run("escaped cross-references are masked, surgically", func(t *testing.T) {
		// Both documented escapes on one line, alongside a real reference: the mask
		// must remove exactly the escaped spans and nothing else (item 4, T-115).
		content := `See \<<no-such-anchor-xyz>> and +<<also-fake>>+ but <<real>> resolves.`
		got, err := docsProseLines(content + "\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) == 0 {
			t.Fatalf("expected at least one prose line, got none")
		}
		line := got[0].text
		if len(line) != len(content) {
			t.Errorf("masking must preserve line length (decision 3): got %d bytes, want %d (%q)",
				len(line), len(content), line)
		}
		if matches := docsXrefTargetRe.FindAllString(line, -1); len(matches) != 1 || matches[0] != "<<real>>" {
			t.Errorf("expected only <<real>> to survive masking, got %v (line %q)", matches, line)
		}
		unmasked := strings.Replace(line, "<<real>>", "", 1)
		if docsRefShapedSiteRe.MatchString(unmasked) {
			t.Errorf("expected both escaped spellings fully masked, but a reference-shaped "+
				"site remains: %q", unmasked)
		}
	})

	t.Run("delimiter-free document yields every line unchanged", func(t *testing.T) {
		// Pins "which lines count as prose" itself, not just the delimiter matcher:
		// TestDocsProseLineSelection otherwise only ever exercised documents that
		// contain a literal block, so "skip indented lines" or any other silent
		// over-exemption would pass every existing fixture here (item 7).
		in := "one\n  two indented\nthree\n\nfive\n"
		got, err := docsProseLines(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var texts []string
		for _, l := range got {
			texts = append(texts, l.text)
		}
		wantLines := strings.Split(in, "\n")
		if strings.Join(texts, "\x00") != strings.Join(wantLines, "\x00") {
			t.Errorf("expected every line unchanged and in order, got %q want %q", texts, wantLines)
		}
	})
}

// TestDocsXrefCheckerCatchesTheFieldFindings is the regression proof: a synthetic
// fixture under t.TempDir(), reproducing the proven-live bugs from the ticket's
// Description — a dead <<anchor>>, and both spellings of the inter-document xref:
// form pointing at a real anchor in a sibling file. It fails today's *code* if any
// detector regresses, independent of the real docs tree's current content.
//
// Page C carries the bare xref:sibling.adoc[text] spelling, with no #anchor. Without
// it, narrowing the pattern to the # form alone left the fixture green while a live
// bare xref shipped through both the test and `just docs-check` (review finding F14).
func TestDocsXrefCheckerCatchesTheFieldFindings(t *testing.T) {
	dir := t.TempDir()
	master := filepath.Join(dir, "book.adoc")
	pageA := filepath.Join(dir, "page-a.adoc")
	pageB := filepath.Join(dir, "page-b.adoc")
	pageC := filepath.Join(dir, "page-c.adoc")
	pageD := filepath.Join(dir, "page-d.adoc")

	docsMustWriteFixture(t, master,
		"include::page-a.adoc[]\n\ninclude::page-b.adoc[]\n\ninclude::page-c.adoc[]\n\n"+
			"include::page-d.adoc[]\n")
	docsMustWriteFixture(t, pageA,
		"[#real-anchor]\n== Page A\n\nSee <<no-such-anchor-xyz>> for details.\n")
	docsMustWriteFixture(t, pageB,
		"== Page B\n\nSee xref:page-a.adoc#real-anchor[Page A] for details.\n")
	docsMustWriteFixture(t, pageC,
		"== Page C\n\nSee xref:page-a.adoc[Page A] for details.\n")
	// Page D pins item 2 / decision 5: the <<file.adoc#anchor>> shorthand must be
	// routed to interDoc, not xrefs, and must not be reported as an uncovered
	// reference-shaped site — both were the proven-live defects (a narrowed pattern
	// re-admits the T-057 hole; a missed route re-admits the coverage-invariant hole).
	docsMustWriteFixture(t, pageD,
		"== Page D\n\nSee <<page-a.adoc#real-anchor,Page A>> for details.\n")

	files, err := bookFiles(master)
	if err != nil {
		t.Fatalf("bookFiles: %v", err)
	}
	anchors, xrefs, interDoc, err := scanBook(files)
	if err != nil {
		t.Fatal(err)
	}

	foundUnresolved := false
	for _, x := range xrefs {
		if x.target == "no-such-anchor-xyz" && docsUnresolvedXref(anchors, x) {
			foundUnresolved = true
		}
	}
	if !foundUnresolved {
		t.Errorf("expected <<no-such-anchor-xyz>> in %s to be flagged as unresolved", pageA)
	}

	flaggedIn := map[string]bool{}
	for _, x := range interDoc {
		if x.target == "page-a.adoc" {
			flaggedIn[x.file] = true
		}
	}
	if !flaggedIn[pageB] {
		t.Errorf("expected xref:page-a.adoc#real-anchor[...] in %s to be flagged", pageB)
	}
	if !flaggedIn[pageC] {
		t.Errorf("expected the bare xref:page-a.adoc[...] spelling in %s to be flagged "+
			"(F14: narrowing the pattern to the # form alone must not pass)", pageC)
	}
	if !flaggedIn[pageD] {
		t.Errorf("expected <<page-a.adoc#real-anchor,Page A>> in %s to be flagged as "+
			"inter-document (item 2, decision 5)", pageD)
	}
	for _, x := range xrefs {
		if x.file == pageD {
			t.Errorf("expected <<page-a.adoc#real-anchor,Page A>> in %s to be routed to "+
				"interDoc, not xrefs — it landed in xrefs as %q", pageD, x.target)
		}
	}

	assertEveryXrefSiteScanned(t, files, xrefs, interDoc)
}

func docsMustWriteFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture %s: %v", path, err)
	}
}
