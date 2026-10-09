package serve

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// md renders ticket bodies. GFM is on because tickets use its tables (the review
// findings table) and strikethrough; raw HTML is *not* (no goldmark.WithUnsafe),
// so anything HTML-shaped inside a ticket is escaped rather than executed. That
// is what makes the template.HTML conversion below honest: the string is markdown
// output with every raw tag already neutralised, not arbitrary trusted input.
//
// The id-ref transformer and renderer (T-142) add AST nodes only; they never
// emit raw HTML through ast.RawHTML, so the same guarantee holds for them.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(idRefTransformer{}, 999))),
	goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(idRefRenderer{}, 999))),
)

// idStatesKey carries one render's IDStates to the transformer, so the shared md
// above stays built once rather than per request.
var idStatesKey = parser.NewContextKey()

// kindTicketRef is a done/dropped ticket id found in a Text node (T-142).
var kindTicketRef = ast.NewNodeKind("TicketRef")

type ticketRef struct {
	ast.BaseInline
	ID, State string
}

func (n *ticketRef) Kind() ast.NodeKind { return kindTicketRef }

func (n *ticketRef) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, map[string]string{"ID": n.ID, "State": n.State}, nil)
}

// idRefTransformer splits each Text node around the done/dropped ids it holds.
// Text under a CodeSpan is left alone (code is quoted, not cited), and so is
// Text under an Image: goldmark builds alt from Text leaves only, so a ticketRef
// there would drop the id from the alt text. Fenced and indented code hold
// lines, not Text nodes, and GFM's autolinked URLs are childless AutoLink
// nodes, so neither is ever reached.
type idRefTransformer struct{}

func (idRefTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	states, _ := pc.Get(idStatesKey).(IDStates)
	if len(states) == 0 {
		return
	}
	src := reader.Source()
	// Hits are found once over the whole source, not per Text node: goldmark
	// splits Text at delimiters such as `_`, which would otherwise move the
	// word boundaries and the URL-run check away from linkifyWith's.
	hitAt := map[int]int{}
	for _, m := range idRefHits(string(src)) {
		hitAt[m[0]] = m[1]
	}
	// Collect first, mutate after, so the walk never visits a node it inserted.
	var hits []*ast.Text
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.CodeSpan, *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if entering && idRefRE.Match(n.Segment.Value(src)) {
				hits = append(hits, n)
			}
		}
		return ast.WalkContinue, nil
	})
	for _, t := range hits {
		splitIDRefs(t, src, states, hitAt)
	}
}

// splitIDRefs replaces t with plain Text pieces around a ticketRef per styled
// id. The line-break flags belong to the end of t, so they move to the last
// piece; a match that is not a whole-source hit (hitAt), or not a done/dropped
// id (UTF-8), stays inside its surrounding text.
func splitIDRefs(t *ast.Text, src []byte, states IDStates, hitAt map[int]int) {
	seg := t.Segment
	parent := t.Parent()
	var last ast.Node
	cursor := seg.Start
	for _, m := range idRefRE.FindAllIndex(seg.Value(src), -1) {
		start, stop := seg.Start+m[0], seg.Start+m[1]
		id := string(src[start:stop])
		st := states.Of(id)
		if st == "" || hitAt[start] != stop {
			continue
		}
		if cursor < start {
			parent.InsertBefore(parent, t, ast.NewTextSegment(text.NewSegment(cursor, start)))
		}
		last = &ticketRef{ID: id, State: st}
		parent.InsertBefore(parent, t, last)
		cursor = stop
	}
	if last == nil {
		return // every match was an unknown or open id
	}
	if cursor < seg.Stop {
		// t itself keeps the tail, and with it its own line-break flags.
		t.Segment = text.NewSegment(cursor, seg.Stop)
		return
	}
	// The id ended the node: hand t's flags to an empty trailing Text, which
	// renders nothing but the break.
	t.Segment = text.NewSegment(seg.Stop, seg.Stop)
}

// idRefRenderer renders a ticketRef with the same markup as free-text ids.
type idRefRenderer struct{}

func (idRefRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindTicketRef, func(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			r := n.(*ticketRef)
			_, _ = w.WriteString(idRefSpan(string(util.EscapeHTML([]byte(r.ID))), r.State))
		}
		return ast.WalkContinue, nil
	})
}

// renderMarkdown converts a ticket body to HTML, with the frontmatter block
// stripped first: the views present those fields structurally (grades, project,
// dependencies), and left in place the leading `---` would render as a horizontal
// rule followed by a stray paragraph of key/value lines. states styles every
// done/dropped id mention (T-142); nil leaves the body as plain markdown.
func renderMarkdown(src string, states IDStates) (template.HTML, error) {
	var buf bytes.Buffer
	pc := parser.NewContext()
	pc.Set(idStatesKey, states)
	if err := md.Convert([]byte(stripFrontmatter(src)), &buf, parser.WithContext(pc)); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil //nolint:gosec // goldmark escapes raw HTML: no WithUnsafe
}

// stripFrontmatter removes a leading `---` … `---` block. Input without one is
// returned unchanged, so a hand-mangled ticket still renders its prose instead of
// disappearing.
func stripFrontmatter(src string) string {
	rest, ok := cutFrontmatter(src)
	if !ok {
		return src
	}
	return rest
}

func cutFrontmatter(src string) (string, bool) {
	trimmed := strings.TrimLeft(src, "\uFEFF \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return src, false
	}
	lines := strings.Split(trimmed, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n"), true
		}
	}
	return src, false // unterminated block: not frontmatter, leave it alone
}
