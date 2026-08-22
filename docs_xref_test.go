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
// Anchor matching is explicit-[#id]-only: the checker does not compute asciidoctor's
// auto-generated section-id algorithm. Every <<...>> target in this manual resolves to
// an explicit anchor today (TestDocsXrefsResolve passing IS that assertion, continuously,
// on every run) — if a future xref is ever written to rely on an auto-generated id, add
// an explicit [#id] to that heading rather than teaching this checker asciidoctor's
// slugger. (Two lines in the live tree, docs/user-manual/configuration.adoc and
// docs/user-manual/concepts/multi-project.adoc, contain the literal text `[[project]]`
// describing TOML array-of-tables syntax in backtick-quoted prose — not an AsciiDoc
// [[id]] anchor definition. A naive `grep '\[\['` over the tree will surface them; they
// are not anchors and this checker's [#id]-only pattern does not match them.)
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

var (
	docsIncludeLineRe  = regexp.MustCompile(`^include::([^\[\]]+)\[`)
	docsAnchorLineRe   = regexp.MustCompile(`^\[#([A-Za-z0-9_-]+)\]\s*$`)
	docsXrefTargetRe   = regexp.MustCompile(`<<([A-Za-z0-9_-]+)(?:,[^>]*)?>>`)
	docsInterDocXrefRe = regexp.MustCompile(`xref:([^\[\s]+\.adoc)[#\[]`)
)

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

// docsXrefOccurrence is one <<target>> or xref:file.adoc... match, with enough
// location to produce a teaching-style failure message (mirrors payloadLintFinding's
// file:line + reason shape in payload_lint_test.go).
type docsXrefOccurrence struct {
	file   string
	line   int
	target string
}

// scanBook reads every file in files and returns: the set of explicit anchor ids
// ([#id] lines), every <<id>>/<<id,text>> occurrence, and every xref:<file>.adoc...
// occurrence — each with its file:line.
func scanBook(files []string) (anchors map[string]bool, xrefs, interDoc []docsXrefOccurrence, err error) {
	anchors = map[string]bool{}
	for _, f := range files {
		data, rerr := os.ReadFile(f)
		if rerr != nil {
			return nil, nil, nil, fmt.Errorf("reading %s: %w", f, rerr)
		}
		for i, line := range strings.Split(string(data), "\n") {
			lineNo := i + 1
			if m := docsAnchorLineRe.FindStringSubmatch(line); m != nil {
				anchors[m[1]] = true
			}
			for _, m := range docsXrefTargetRe.FindAllStringSubmatch(line, -1) {
				xrefs = append(xrefs, docsXrefOccurrence{file: f, line: lineNo, target: m[1]})
			}
			for _, m := range docsInterDocXrefRe.FindAllStringSubmatch(line, -1) {
				interDoc = append(interDoc, docsXrefOccurrence{file: f, line: lineNo, target: m[1]})
			}
		}
	}
	return anchors, xrefs, interDoc, nil
}

// TestDocsXrefsResolve walks the real docs/user-manual.adoc book and fails listing
// every <<target>> that does not resolve to any [#target] anchor in the assembled
// book — the core check this ticket exists to add (T-067).
func TestDocsXrefsResolve(t *testing.T) {
	files, err := bookFiles(docsBookMaster)
	if err != nil {
		t.Fatalf("walking book includes from %s: %v", docsBookMaster, err)
	}
	anchors, xrefs, _, err := scanBook(files)
	if err != nil {
		t.Fatal(err)
	}

	var bad []string
	for _, x := range xrefs {
		if !anchors[x.target] {
			bad = append(bad, fmt.Sprintf(
				"%s:%d: <<%s>> does not resolve to any [#%s] anchor in the assembled book\n"+
					"    add [#%s] above the intended heading, or fix the xref target",
				x.file, x.line, x.target, x.target, x.target))
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
		bad = append(bad, fmt.Sprintf(
			"%s:%d: xref:%s... targets another .adoc file — asciidoctor "+
				"resolves it happily as HTML, but the PDF/EPUB build splits each included file "+
				"into its own chapter with no standalone %s.pdf/%s.epub artifact, so the link "+
				"is dead in both real output formats (T-057 finding N3)\n"+
				"    use <<anchor>> instead — the book is one document, not many",
			x.file, x.line, x.target,
			strings.TrimSuffix(x.target, ".adoc"), strings.TrimSuffix(x.target, ".adoc")))
	}
	sort.Strings(bad)
	t.Fatalf("%d inter-document xref: form(s):\n\n%s", len(bad), strings.Join(bad, "\n\n"))
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

// TestDocsXrefCheckerCatchesTheFieldFindings is the regression proof: a synthetic
// two-file fixture under t.TempDir(), reproducing both proven-live bugs from the
// ticket's Description at once — a dead <<anchor>> and an xref:<file>.adoc#...
// pointing at a real anchor in a sibling file. It fails today's *code* if either
// detector regresses, independent of the real docs tree's current content.
func TestDocsXrefCheckerCatchesTheFieldFindings(t *testing.T) {
	dir := t.TempDir()
	master := filepath.Join(dir, "book.adoc")
	pageA := filepath.Join(dir, "page-a.adoc")
	pageB := filepath.Join(dir, "page-b.adoc")

	docsMustWriteFixture(t, master, "include::page-a.adoc[]\n\ninclude::page-b.adoc[]\n")
	docsMustWriteFixture(t, pageA,
		"[#real-anchor]\n== Page A\n\nSee <<no-such-anchor-xyz>> for details.\n")
	docsMustWriteFixture(t, pageB,
		"== Page B\n\nSee xref:page-a.adoc#real-anchor[Page A] for details.\n")

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
		if x.target == "no-such-anchor-xyz" && !anchors[x.target] {
			foundUnresolved = true
		}
	}
	if !foundUnresolved {
		t.Errorf("expected <<no-such-anchor-xyz>> in %s to be flagged as unresolved", pageA)
	}

	foundInterDoc := false
	for _, x := range interDoc {
		if x.target == "page-a.adoc" {
			foundInterDoc = true
		}
	}
	if !foundInterDoc {
		t.Errorf("expected xref:page-a.adoc#... in %s to be flagged as an inter-document form", pageB)
	}
}

func docsMustWriteFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture %s: %v", path, err)
	}
}
