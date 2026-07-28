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
}
