package serve

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// md renders ticket bodies. GFM is on because tickets use its tables (the review
// findings table) and strikethrough; raw HTML is *not* (no goldmark.WithUnsafe),
// so anything HTML-shaped inside a ticket is escaped rather than executed. That
// is what makes the template.HTML conversion below honest: the string is markdown
// output with every raw tag already neutralised, not arbitrary trusted input.
var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

// renderMarkdown converts a ticket body to HTML, with the frontmatter block
// stripped first: the views present those fields structurally (grades, project,
// dependencies), and left in place the leading `---` would render as a horizontal
// rule followed by a stray paragraph of key/value lines.
func renderMarkdown(src string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(stripFrontmatter(src)), &buf); err != nil {
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
