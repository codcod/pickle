package serve

import (
	"html/template"
	"strings"
)

// funcs is the template helper set — deliberately tiny. Anything that needs real
// logic belongs in view.go, where it is testable without a template.
var funcs = template.FuncMap{
	// slug turns a grade value into a CSS class suffix ("medium-high" stays as is,
	// "" becomes "none" so the class name is never dangling).
	"slug": func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		s = strings.ReplaceAll(s, " ", "-")
		if s == "" {
			return "none"
		}
		return s
	},
	// linkify wraps a bare http(s) URL in s in a clickable anchor; the logic lives
	// in view.go's linkifyURLs (T-089), tested there without a template.
	"linkify": linkifyURLs,
	// idListOf pairs a BasePath (read off whatever struct "idlist" is called
	// from — Entry or TicketView, both of which carry one) with the id slice
	// being linked, so the "idlist" template block always has both, regardless
	// of how deep the enclosing {{template}} calls nest (T-127; see
	// Entry.BasePath's doc comment in view.go).
	"idListOf": func(basePath string, ids []string) IDList {
		return IDList{BasePath: basePath, IDs: ids}
	},
}
