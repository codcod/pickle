package state

import (
	"regexp"
	"strings"

	"github.com/codcod/pickle/internal/ticket"
)

// This file parses a ticket's `## Review` findings table(s) into the
// closed-vocabulary Finding shape (T-065 confirmed decision 9 — the "middle
// option" chosen at refinement). It deliberately projects only four columns
// — id, severity, class, disposition — and never the three prose columns
// (description, evidence, suggestion, under any of their header aliases).
//
// Why: the corpus this parses is not one shape. review-protocol.md §5 fixed
// a canonical seven-column header only from T-085 onward (2026-08-13), and
// T-085 was explicitly prospective-only — it did not backfill the existing
// tables (its own confirmed decision 3). So an established archive of shipped
// tickets holds many header shapes, of which the canonical one is a minority.
// The exact counts are deliberately not restated here: a hand-copied figure
// goes stale the moment another review lands, which is half of why this
// command exists. Re-measure them with the command itself:
//
//	pickle board state --json | jq -r '.tickets[].review.headers[] | @csv' | sort -u
//
// A parser keyed to column *position* would silently mis-column most of that
// corpus. Keying by column *name* instead
// turns that into a bounded, testable problem for four short tokens
// (severity/class/disposition are drawn from small vocabularies; id is a
// short bareword) — and an open-ended one for three columns that are
// multi-sentence prose containing pipes, backticks and embedded tables.

// sepCellRE matches one GFM table-separator cell: optional leading/trailing
// colon (alignment), one or more dashes.
var sepCellRE = regexp.MustCompile(`^:?-+:?$`)

// emphasisRE strips markdown emphasis/code markers (**bold**, *italic*,
// `code`) from a cell before comparing or storing it — formatting only,
// never a value judgement (T-065 confirmed decision 11).
var emphasisRE = regexp.MustCompile("[*`]")

// whitespaceRE collapses any run of whitespace to a single space.
var whitespaceRE = regexp.MustCompile(`\s+`)

// splitRow splits one markdown table row into cells, respecting `\|` as a
// literal escaped pipe (the corpus uses this — e.g. "project add\|list") so
// it never becomes a phantom column boundary. A single leading and single
// trailing empty cell — produced by the row's opening/closing `|`, which are
// GFM's optional delimiters rather than column separators — are dropped.
func splitRow(line string) []string {
	runes := []rune(strings.TrimSpace(line))
	var cells []string
	var buf strings.Builder
	for i := 0; i < len(runes); i++ {
		switch {
		case runes[i] == '\\' && i+1 < len(runes) && runes[i+1] == '|':
			buf.WriteRune('|')
			i++
		case runes[i] == '|':
			cells = append(cells, buf.String())
			buf.Reset()
		default:
			buf.WriteRune(runes[i])
		}
	}
	cells = append(cells, buf.String())
	if len(cells) > 0 && strings.TrimSpace(cells[0]) == "" {
		cells = cells[1:]
	}
	if len(cells) > 0 && strings.TrimSpace(cells[len(cells)-1]) == "" {
		cells = cells[:len(cells)-1]
	}
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// isTableRowLine reports whether line could be a markdown table row at all
// (non-blank, contains an unescaped `|`). Used both to find candidate header
// lines and to decide where a table's row run ends.
func isTableRowLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	return len(splitRow(line)) > 0 && strings.Contains(line, "|")
}

// isSeparatorRow reports whether line is a GFM header/body separator row —
// every cell matching sepCellRE. This is what distinguishes a table's header
// line from an ordinary paragraph that happens to contain a `|`.
func isSeparatorRow(line string) bool {
	cells := splitRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !sepCellRE.MatchString(strings.TrimSpace(c)) {
			return false
		}
	}
	return true
}

// normalizeCell strips emphasis/code markers, collapses whitespace, lowercases
// and trims — formatting normalisation only (decision 11). Used for both
// header names (before alias resolution) and finding values.
func normalizeCell(s string) string {
	s = emphasisRE.ReplaceAllString(s, "")
	s = whitespaceRE.ReplaceAllString(s, " ")
	return strings.ToLower(strings.TrimSpace(s))
}

// headerAliases maps a normalised header cell onto the canonical column name
// the rest of this file keys on. "#" is the only alias the corpus actually
// uses in place of "id" (a couple of one-off review tables lead with a bare
// row number instead of a finding id).
var headerAliases = map[string]string{
	"#": "id",
}

func normalizeHeaderName(s string) string {
	n := normalizeCell(s)
	if canon, ok := headerAliases[n]; ok {
		return canon
	}
	return n
}

// reviewTable is one markdown table found under `## Review`: its normalised
// header row and its data rows (each already split into cells, unnormalised
// — value normalisation happens per-field, only for the columns projected).
type reviewTable struct {
	header []string
	rows   [][]string
}

// parseTables walks body (a ticket's `## Review` section) and returns every
// markdown table found, in source order — findings tables and any other
// kind (e.g. the "Acceptance test" or gate-checklist tables that also live
// under this heading) alike. Callers filter to findings tables with
// isFindingsHeader.
//
// A table is recognised by the GFM shape: a candidate header line
// immediately followed by a valid separator row (isSeparatorRow). Once found,
// every following line that isTableRowLine is a data row; the table ends at
// the first line that is not (blank line, prose, or a `###` heading — all of
// which are common between tables in a multi-round review, so no table ever
// runs into its neighbour).
func parseTables(body string) []reviewTable {
	lines := strings.Split(body, "\n")
	var tables []reviewTable
	i := 0
	for i < len(lines) {
		if isTableRowLine(lines[i]) && i+1 < len(lines) && isSeparatorRow(lines[i+1]) {
			header := make([]string, 0, 8)
			for _, c := range splitRow(lines[i]) {
				header = append(header, normalizeHeaderName(c))
			}
			j := i + 2
			var rows [][]string
			for j < len(lines) && isTableRowLine(lines[j]) {
				rows = append(rows, splitRow(lines[j]))
				j++
			}
			tables = append(tables, reviewTable{header: header, rows: rows})
			i = j
			continue
		}
		i++
	}
	return tables
}

// isFindingsHeader is the detection rule (T-065 confirmed decision 10): a
// table is a findings table iff its header contains a column named exactly
// "severity" and one named exactly "disposition" (after normalisation and
// alias resolution). This is what excludes a near-miss like "check | old
// trigger | new trigger | severity before → after" (that table's fourth
// column, once emphasis is stripped, is "severity before → after", not
// "severity" — no match) without excluding any real findings-table variant in
// the corpus, every one of which carries both columns under exactly those two
// names.
func isFindingsHeader(header []string) bool {
	hasSeverity, hasDisposition := false, false
	for _, h := range header {
		switch h {
		case "severity":
			hasSeverity = true
		case "disposition":
			hasDisposition = true
		}
	}
	return hasSeverity && hasDisposition
}

// columnIndex returns the first index of name in header, or -1.
func columnIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

// cellAt returns the normalised value of row's column named name in header,
// or "" when header has no such column or row is short — never a value
// shifted in from a neighbouring column (decision 10/11).
func cellAt(header []string, row []string, name string) string {
	idx := columnIndex(header, name)
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return normalizeCell(row[idx])
}

// dispositionSummaryRE and costLineRE find the two closer lines under a
// findings table — projected verbatim and unparsed (decision 9): they are
// free prose plus a closed-vocabulary cost pair, and the raw string is more
// useful to a consumer than a partial parse of it.
var (
	dispositionSummaryRE = regexp.MustCompile(`(?i)^\*{0,2}disposition summary\b`)
	costLineRE           = regexp.MustCompile("(?i)^`?cost: estimated")
)

// parseReview builds one ticket's Review projection from its full file text.
func parseReview(text string) Review {
	rev := Review{Headers: [][]string{}, Findings: []Finding{}}
	body, found := ticket.SectionBody(text, "Review")
	if !found || body == "" {
		return rev
	}

	for _, t := range parseTables(body) {
		if !isFindingsHeader(t.header) {
			continue
		}
		rev.Tables++
		rev.Headers = append(rev.Headers, t.header)
		for _, row := range t.rows {
			rev.Findings = append(rev.Findings, Finding{
				ID:          cellAt(t.header, row, "id"),
				Severity:    cellAt(t.header, row, "severity"),
				Class:       cellAt(t.header, row, "class"), // "" for a pre-T-085 table
				Disposition: cellAt(t.header, row, "disposition"),
			})
		}
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if rev.DispositionSummary == "" && dispositionSummaryRE.MatchString(trimmed) {
			rev.DispositionSummary = trimmed
		}
		if rev.CostLine == "" && costLineRE.MatchString(trimmed) {
			rev.CostLine = strings.Trim(trimmed, "`")
		}
	}
	return rev
}
