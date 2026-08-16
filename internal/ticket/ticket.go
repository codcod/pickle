// Package ticket is the shared model for ticket files: frontmatter parsing,
// History interpretation, and loading a whole tickets/ tree. Both the board
// audit (internal/audit) and the project remove-guard (internal/cli) build on
// it, so the frontmatter scan lives in exactly one place.
//
// The status vocabulary itself (the seven directories, their display names,
// terminal flags, legal transitions) is not this package's data — it lives in
// internal/flow as a *flow.Definition, and every function here that needs it
// (LoadAll, HistoryEntries, LastHistoryStatus, LastHistoryReason, MergeLine,
// HasMergeLine) takes one as an explicit parameter (T-080).
package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/flow"
)

// Ticket is one parsed ticket file.
type Ticket struct {
	ID        string            // "T-001" (from the filename)
	Num       int               // 1
	Slug      string            // "config-and-project-registry"
	Dir       string            // status directory, e.g. "1-to-do"
	Path      string            // absolute path
	Front     map[string]string // raw frontmatter values (title kept verbatim)
	DependsOn []string          // parsed depends-on ids — hard dependencies; gate pickup
	SpawnedBy []string          // parsed spawned-by ids — lineage only, never a gate
	Family    string            // umbrella ticket id (single, same child) — lineage only, never a gate; "" when none
	// DuplicateKeys lists frontmatter keys that appeared more than once (each
	// listed once, first-seen order) — a malformed-input record only. The parse
	// still keeps the last value for each (Front is unaffected); reporting the
	// duplication is internal/audit's job (T-040).
	DuplicateKeys []string
	Text          string // full file text
}

// Base is the filename without extension, e.g. "T-001-config-and-project-registry".
func (t *Ticket) Base() string { return strings.TrimSuffix(filepath.Base(t.Path), ".md") }

// Project is the project: frontmatter value.
func (t *Ticket) Project() string { return t.Front["project"] }

var (
	filenameRE = regexp.MustCompile(`^([A-Z][A-Z0-9]*-\d+)-[A-Za-z0-9._-]+\.md$`)
	// idRE is the canonical ticket-id shape. The prefix is a per-child
	// ticket_prefix (T-058), so the shape is <PREFIX>-NNN where PREFIX is an
	// uppercase letter followed by uppercase letters/digits; "T-" is just the
	// default prefix, not baked into the shape. Exported through ValidID so
	// creation time (internal/cli) and the audit share one definition rather than
	// growing a second. The shape is still spelled out separately in filenameRE
	// above and in board.rowRE; unifying those is T-042's call (T-040 D6 deferred
	// it there; T-027, which used to own it, was absorbed into T-040 and dropped). ValidID stays a
	// pure *shape* check — that a ticket's prefix matches its project's configured
	// prefix is a config-aware invariant, checked in internal/audit, not here.
	idRE       = regexp.MustCompile(`^[A-Z][A-Z0-9]*-\d+$`)
	fmKeyRE    = regexp.MustCompile(`^([A-Za-z-]+):\s*(.*)$`)
	inlineHash = regexp.MustCompile(`\s+#.*$`)
	// historyRE matches one dated History bullet, capturing only the body as m[1].
	// The date is deliberately *not* captured: LastHistoryStatus, LastHistoryReason
	// and MergeLine all read m[1], and adding a group for the date — named or not,
	// since naming a group in Go does not remove it from the numbering — would
	// shift the body to m[2] and silently break all three. HistoryEntries reads the
	// date with historyDate instead, so this pattern can stay untouched.
	historyRE = regexp.MustCompile(`^-\s*\d{4}-\d{2}-\d{2}\s*[—-]+\s*(.+)$`)
	createdRE = regexp.MustCompile(`(?i)^created\s*\(([^)]+)\)`)
	mergedRE  = regexp.MustCompile(`(?i)^merged\b`)
)

// dateLen is the length of a YYYY-MM-DD date, the only date form historyRE admits.
const dateLen = len("YYYY-MM-DD")

// historyDate returns the date of a History line that historyRE has already
// matched: the match guarantees the first digit in the line opens the date, so
// the ten runes from there are it. Slicing beats adding a capture group, which
// would renumber historyRE's body group out from under three other callers.
// Bytes are safe here: every character up to and including the date is ASCII.
func historyDate(line string) string {
	i := strings.IndexAny(line, "0123456789")
	if i < 0 || len(line) < i+dateLen {
		return ""
	}
	return line[i : i+dateLen]
}

// HistoryKind classifies a History entry's body — the one place every reader of
// `## History` (LastHistoryStatus, LastHistoryReason, MergeLine, and the board
// audit's length check, T-040) decides what kind of line it is looking at,
// instead of each growing its own merge/created/transition test.
type HistoryKind string

const (
	HistoryCreated    HistoryKind = "created"    // "created (TO DO). source: …"
	HistoryMerged     HistoryKind = "merged"     // "merged to <base> (<ref>)" / legacy "MERGED: … → …"
	HistoryTransition HistoryKind = "transition" // "OLD → NEW: reason", NEW a legal status name
	HistoryNote       HistoryKind = "note"       // free-form dated line (gate records, corrections, …)
)

// historyKind classifies one History entry body. Order matters: merged is
// tested before transition, because a legacy merge line ("MERGED: feat/… →
// main (abc123)") also contains a "→" and must not be misread as a status
// transition. def supplies the status vocabulary transitionParts needs to
// decide whether a candidate target is a legal status name.
func historyKind(def *flow.Definition, body string) HistoryKind {
	switch {
	case mergedRE.MatchString(body):
		return HistoryMerged
	case createdRE.MatchString(body):
		return HistoryCreated
	default:
		if _, _, ok := transitionParts(def, body); ok {
			return HistoryTransition
		}
		return HistoryNote
	}
}

// arrow is the transition separator this repo's History lines are written with.
const arrow = "→"

// transitionParts locates the transition inside a History entry body
// ("OLD → NEW[: reason]") and returns NEW plus the reason clause, reporting ok
// only when NEW is a legal status name. It is the single place that decides
// whether a body is a status transition at all: historyKind, HistoryEntries and
// LastHistoryReason all resolve to this one function instead of each re-deriving
// the split, which is what let three readers drift out of sync before T-043.
//
// The rule is **the leftmost arrow whose candidate target is a legal status
// name**, where the candidate ends at the next colon. Three shapes, all real,
// force exactly that rule:
//
//   - "IN REVIEW → DONE: review PASS; 2 non-blocking → fixed inline" — a reason
//     may contain an arrow of its own (T-058's actual History). Searching for the
//     *last* arrow read the reason's arrow and demoted the entry to a note.
//   - "audit fix: TO DO → READY" — a body may carry a colon *before* the
//     transition, so the reason clause cannot be found by splitting on the
//     body's *first* colon; that only worked while every reason came last.
//   - "IN DEVELOPMENT → IN REVIEW → DONE" — two arrows and no reason. The
//     leftmost candidate ("IN REVIEW → DONE") is not a legal status, so the scan
//     continues rather than giving up, and the entry still resolves to DONE.
//
// Requiring the candidate to be a legal status *exactly* (not merely to start
// with one) is what keeps a note that happens to mention an arrow — "clarified
// that a merge → DONE requires a human" — a note.
func transitionParts(def *flow.Definition, body string) (target, reason string, ok bool) {
	for rest := body; ; {
		i := strings.Index(rest, arrow)
		if i < 0 {
			return "", "", false
		}
		rest = rest[i+len(arrow):]
		candidate, tail := rest, ""
		if j := strings.Index(rest, ":"); j >= 0 {
			candidate, tail = rest[:j], strings.TrimSpace(rest[j+1:])
		}
		if name := strings.ToUpper(strings.TrimSpace(candidate)); statusExists(def, name) {
			return name, tail, true
		}
	}
}

// statusExists reports whether name is a legal status display name in def.
func statusExists(def *flow.Definition, name string) bool {
	_, ok := def.ByName(name)
	return ok
}

// HistoryEntry is one dated line from a ticket's `## History` section.
type HistoryEntry struct {
	Date string      // raw YYYY-MM-DD, exactly as written (the regex anchors the shape)
	Text string      // the line's body: a transition, a created line, or a merge note
	Kind HistoryKind // classified from the body's first physical line, stable across continuation folding
	// Target is the transition's destination status display name, or "" unless
	// Kind is HistoryTransition. It is derived from the *same* first physical
	// line Kind is, in the same pass, so the two cannot disagree — a reader that
	// re-derived the target from the folded Text could, and did: "TO DO → READY"
	// with a wrapped continuation line folds to "TO DO → READY <prose>", whose
	// candidate target is no longer a legal status, so the entry classified as a
	// transition and then resolved to no status at all (T-043 review, R1).
	Target string
}

// HistoryEntries returns every dated entry of a ticket's `## History` section in
// file order (oldest first). Created lines and merge notes are included
// deliberately: this is the raw record, and deciding what an entry *means* is the
// caller's job (contrast LastHistoryStatus/MergeLine, which filter). Content
// outside `## History`, and bullets that carry no date, are skipped.
//
// An entry may be wrapped across several source lines — long reasons routinely are,
// and this repo's own tickets wrap them — so continuation lines are folded back
// into one logical entry. Reading only the first physical line would silently cut
// a reason mid-sentence, which is exactly the sort of quiet truncation a history is
// supposed to prevent.
//
// Date stays a string: the format is already anchored by historyRE, ordering is
// lexicographic for YYYY-MM-DD, and no caller needs calendar arithmetic.
func HistoryEntries(def *flow.Definition, text string) []HistoryEntry {
	var out []HistoryEntry
	inHistory := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			inHistory = strings.TrimSpace(line) == "## History"
			continue
		}
		if !inHistory {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if m := historyRE.FindStringSubmatch(trimmed); m != nil {
			body := strings.TrimSpace(m[1])
			e := HistoryEntry{Date: historyDate(trimmed), Text: body, Kind: historyKind(def, body)}
			if e.Kind == HistoryTransition {
				// Same input as historyKind just classified, so Kind and Target
				// are decided together and stay true after folding (T-043 R1).
				e.Target, _, _ = transitionParts(def, body)
			}
			out = append(out, e)
			continue
		}
		// Continuation of the entry above: indented text that opens no new bullet.
		// A bullet without a date is not a continuation — it is a different
		// (undated, and therefore ignored) list item.
		if len(out) == 0 || trimmed == "" || strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "<!--") || line == trimmed {
			continue
		}
		last := &out[len(out)-1]
		last.Text += " " + trimmed
	}
	return out
}

// ParseFrontmatter parses the YAML-ish frontmatter block into a flat map. ok is
// false when there is no leading `---` block. Duplicate keys silently overwrite
// (last wins) — callers that need to know about them use parseFrontmatter.
func ParseFrontmatter(text string) (map[string]string, bool) {
	fm, _, ok := parseFrontmatter(text)
	return fm, ok
}

// parseFrontmatter is the one frontmatter scan (package doc: "the frontmatter
// scan lives in exactly one place") that both ParseFrontmatter and LoadAll build
// on. dupes lists every key that appears more than once, each listed exactly
// once in first-seen order — the map itself still keeps the *last* value for a
// duplicated key; parse semantics are unchanged, only reported (T-040).
func parseFrontmatter(text string) (fm map[string]string, dupes []string, ok bool) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, nil, false
	}
	fm = map[string]string{}
	dupeSeen := map[string]bool{}
	for i := 1; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		m := fmKeyRE.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		key, val := m[1], m[2]
		if key != "title" { // titles may legitimately contain '#'
			val = inlineHash.ReplaceAllString(val, "")
		}
		if _, seen := fm[key]; seen && !dupeSeen[key] {
			dupes = append(dupes, key)
			dupeSeen[key] = true
		}
		fm[key] = strings.Trim(strings.TrimSpace(val), `"'`)
	}
	return fm, dupes, true
}

// ParseDepends parses a bracketed ticket-id list like "[T-001, T-002]" or "[]".
// Brackets are optional, so it also accepts the comma-separated flag form
// ("T-001,T-002"). Used for both `depends-on:` and `spawned-by:` — the two share
// a wire format and differ only in meaning.
func ParseDepends(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ValidID reports whether s has the canonical ticket-id shape (T-<digits>).
// It is a *shape* check only: whether the ticket exists is the audit's job, since
// a forward reference to a not-yet-filed ticket is legal input (rules §3).
//
// Deliberately does not trim: callers that accept human input tokenize first
// (see ParseIDList, which trims via ParseDepends).
func ValidID(s string) bool { return idRE.MatchString(s) }

// ParseIDList parses a comma-separated (or bracketed) ticket-id list like
// ParseDepends, but validates each token's shape and drops duplicates, keeping
// first-seen order. For flag input, where rejecting is possible; ParseDepends
// stays lenient for already-written frontmatter, where it is not.
//
// The error names the offending token so a malformed id is reported as malformed
// rather than as a missing ticket.
func ParseIDList(raw string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, tok := range ParseDepends(raw) {
		if !ValidID(tok) {
			return nil, fmt.Errorf("%q is not a ticket id (expected <PREFIX>-NNN, e.g. T-001)", tok)
		}
		if seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out, nil
}

// LastHistoryStatus returns the display name of the newest status-bearing History
// transition, or "" if none is parseable. Merge notes ("… — MERGED: …") are
// skipped.
//
// Routed through HistoryEntries (rather than re-walking text itself) so a
// transition wrapped across continuation lines folds exactly the way
// HistoryEntries already folds it for every other reader — the two used to be
// able to disagree on where an entry ends, T-043. It reads HistoryEntry.Target
// rather than re-deriving the target from the folded Text: re-derivation is what
// let an entry be classified a transition and still yield no status (T-043 R1).
func LastHistoryStatus(def *flow.Definition, text string) string {
	status := ""
	for _, e := range HistoryEntries(def, text) {
		switch e.Kind {
		case HistoryCreated:
			if c := createdRE.FindStringSubmatch(e.Text); c != nil {
				status = strings.ToUpper(strings.TrimSpace(c[1]))
			}
		case HistoryTransition:
			status = e.Target
		}
	}
	return status
}

// LastHistoryReason returns the ": <reason>" clause of the newest status-bearing
// History transition (the text `move --reason` appends), or "" when the last
// transition carries no reason. Merge notes and created lines are skipped — they
// are not transitions. The board renderer derives the DROPPED `reason` and REWORK
// `open findings` cells from this, so those facts live only in the ticket (D3).
//
// Routed through HistoryEntries for the same reason as LastHistoryStatus
// (T-043): a reason clause wrapped onto a continuation line used to be silently
// truncated to its first physical line, because the old per-line walk never saw
// the fold HistoryEntries already performs for every other reader.
//
// The reason — unlike the target, which is frozen with Kind on the entry's first
// physical line — is read from the *folded* Text, since folding it back together
// is the whole point. When the folded text no longer presents a transition (a
// reason-less "TO DO → READY" whose continuation line is plain prose), there is
// no reason clause to report and "" is the answer.
func LastHistoryReason(def *flow.Definition, text string) string {
	reason := ""
	for _, e := range HistoryEntries(def, text) {
		if e.Kind != HistoryTransition {
			continue // merge note, created line, or free-form note
		}
		reason = ""
		if _, r, ok := transitionParts(def, e.Text); ok {
			reason = r
		}
	}
	return reason
}

// MergeLine returns the text of the newest merge History line ("merged to main
// (MR !12, abc1234)" from "- 2026-07-23 — merged to main (MR !12, abc1234)"), or "" when the
// History records no merge. The board renderer derives the DONE `merged` cell
// from this (D3), so HasMergeLine and the cell can never disagree.
func MergeLine(def *flow.Definition, text string) string {
	inHistory := false
	merge := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			inHistory = strings.TrimSpace(line) == "## History"
			continue
		}
		if !inHistory {
			continue
		}
		if m := historyRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			if body := strings.TrimSpace(m[1]); historyKind(def, body) == HistoryMerged {
				merge = body
			}
		}
	}
	return merge
}

// HasMergeLine reports whether the History records a merge ("… — MERGED: …").
// Defined in terms of MergeLine so the gate check and the board's `merged`
// cell can never disagree.
func HasMergeLine(def *flow.Definition, text string) bool {
	return MergeLine(def, text) != ""
}

// Legal grade values (single values or adjacent-pair ranges). These are the one
// source of truth, consumed by both the board audit and `ticket new`.
var (
	LegalImpact     = gradeSet("low", "medium", "high", "critical", "low-medium", "medium-high", "high-critical")
	LegalComplexity = gradeSet("low", "medium", "high", "low-medium", "medium-high")
	LegalCost       = gradeSet("S", "M", "L", "XL", "S-M", "M-L", "L-XL")
)

func gradeSet(vals ...string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// ValidGrade reports whether v is legal for the grade kind ("impact",
// "complexity", or "cost").
func ValidGrade(kind, v string) bool {
	switch kind {
	case "impact":
		return LegalImpact[v]
	case "complexity":
		return LegalComplexity[v]
	case "cost":
		return LegalCost[v]
	}
	return false
}

// SplitID splits a ticket id like "RICK-058" into its prefix ("RICK") and number
// (58). It splits on the last '-', so a slug's hyphens never confuse it (ids are
// only ever <PREFIX>-<NNN>, but callers pass the id, not the filename). ok is
// false when there is no '-' or the tail is not an integer.
func SplitID(id string) (prefix string, num int, ok bool) {
	i := strings.LastIndex(id, "-")
	if i < 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(id[i+1:])
	if err != nil {
		return "", 0, false
	}
	return id[:i], n, true
}

// NextNum returns the next free ticket number for one prefix: max(number of
// every <prefix>-NNN filename across all status dirs) + 1. IDs are per-child
// counters (T-058), so the prefix scopes the sequence — two children with
// distinct prefixes number independently, and children sharing the default "T"
// share one counter (the legacy global namespace). Scans filenames directly so
// it is robust to files that fail frontmatter parsing.
func NextNum(def *flow.Definition, root, prefix string) int {
	max := 0
	for _, s := range def.States() {
		entries, err := os.ReadDir(filepath.Join(root, "tickets", s.Dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if m := filenameRE.FindStringSubmatch(e.Name()); m != nil {
				if p, n, ok := SplitID(m[1]); ok && p == prefix && n > max {
					max = n
				}
			}
		}
	}
	return max + 1
}

var slugStripRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a title into a filename slug: lowercase, non-alphanumeric runs
// collapsed to single hyphens, trimmed. Empty input yields "untitled".
func Slugify(title string) string {
	s := strings.Trim(slugStripRE.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if s == "" {
		return "untitled"
	}
	return s
}

// SectionHeadings returns the ordered list of top-level ("## ") section headings
// in a ticket/template body — used to keep Scaffold in step with TEMPLATE.md.
func SectionHeadings(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			out = append(out, strings.TrimSpace(line[3:]))
		}
	}
	return out
}

// SectionBody returns the trimmed body text of the top-level ("## <heading>")
// section named heading, and whether that section exists at all. Matching is
// exact and case-sensitive, mirroring SectionHeadings; the walk is the same
// line-prefix scan, so a "## <heading>"-looking line inside a fenced code
// block would be misread the same way SectionHeadings already would be — a
// pre-existing, shared limitation, not a new one. Unlike SectionHeadings
// (which only lists names), this reads content, for callers that must judge
// whether a section is *empty* (T-083's Outcome-presence check) rather than
// merely present.
func SectionBody(text, heading string) (body string, found bool) {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && strings.TrimSpace(line[3:]) == heading {
			start = i + 1
			found = true
			break
		}
	}
	if !found {
		return "", false
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n")), true
}

// htmlCommentRE strips an HTML comment span (the TEMPLATE.md placeholder
// form, e.g. "<!-- TODO: ... -->") when judging whether a section's body is
// substantive. (?s) makes "." match newlines, since a placeholder comment may
// wrap onto its own line.
var htmlCommentRE = regexp.MustCompile(`(?s)<!--.*?-->`)

// SectionMissing reports whether text's "## <heading>" section is absent, or
// its body is empty once HTML comments (the TEMPLATE.md placeholder form) are
// stripped and the remainder is trimmed of whitespace. Originally T-083's
// Outcome-specific predicate, generalised by T-081 into the one "is this
// section substantive" check both SubsectionMissing and GateViolations build
// on. It is deliberately structural, not a prose-quality heuristic:
// presence-and-not-placeholder is mechanical and has no judgement in it. The
// `<…>` angle-bracket placeholder form TEMPLATE.md uses elsewhere in its
// Implementation Plan skeleton is NOT stripped (only HTML comments are) — a
// skeleton pasted verbatim therefore reads as substantive. That is a known,
// documented boundary (T-083 review finding B1), not a defect to fix here.
func SectionMissing(text, heading string) bool {
	body, found := SectionBody(text, heading)
	if !found {
		return true
	}
	return strings.TrimSpace(htmlCommentRE.ReplaceAllString(body, "")) == ""
}

// leadingOrdinalRE strips a leading "N" / "N." / "N)" step number
// ("0. Feature branch" -> "Feature branch") and trailingParenRE strips one
// trailing parenthetical ("Feature branch (mandatory)" -> "Feature branch") —
// the two forms of decoration TEMPLATE.md's own Implementation Plan headings
// carry that a content stem match must see through.
var (
	leadingOrdinalRE = regexp.MustCompile(`^\d+[.)]?\s*`)
	trailingParenRE  = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
)

// normalizeHeading reduces a "### …" heading to the deterministic form
// flow.Requirement.Sub is matched against as a prefix (T-081 decision 7):
// lower-case, strip one leading ordinal and one trailing parenthetical,
// collapse internal whitespace runs, and trim surrounding whitespace and
// trailing punctuation. It is a fixed, declared normalisation — never a
// content judgement — chosen by measuring every "### " heading actually used
// under an Implementation Plan across this repo's own 45 done tickets at
// T-081's refinement: forms like "0. Feature branch (mandatory)",
// "2. Confirmed decisions", "Docs", "6. Finish" and
// "Acceptance test (run verbatim; must be green before review)" all reduce
// to a stem a Requirement.Sub can prefix-match; a heading using neither
// vocabulary (e.g. "### 4. Tests" for what rules §4.5 calls "Acceptance
// test") legitimately does not match, and is not this function's job to fix.
func normalizeHeading(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = leadingOrdinalRE.ReplaceAllString(h, "")
	h = trailingParenRE.ReplaceAllString(h, "")
	h = strings.Join(strings.Fields(h), " ")
	return strings.TrimRight(strings.TrimSpace(h), ".:;,")
}

// SubsectionBody returns the trimmed body of the first "### " heading inside
// the top-level "## section" span whose normalizeHeading form has stem as a
// prefix, bounded by the next "### " or "## " heading, whichever comes first
// (or EOF). found is false when the parent section itself is absent, or no
// "### " heading under it matches stem — callers that don't need to
// distinguish the two (e.g. SubsectionMissing) can fold found==false into
// "missing" directly. Reuses SectionBody's line-prefix walk rather than
// adding a fifth copy of it (T-042): the parent scan is exactly SectionBody's
// own logic, repeated one level down for "### " instead of "## ". Factored
// out of SubsectionMissing (T-105) so a second consumer (a decisions-style
// reader that needs the body's *content*, not just whether it's substantive)
// does not need a sixth copy.
//
// Like SectionBody/SectionHeadings, this walk is blind to a "### "-looking
// line at column 0 inside a fenced code block — an inherited, pre-existing
// limitation (T-083 documented it for SectionHeadings), not a new one.
func SubsectionBody(text, section, stem string) (body string, found bool) {
	parent, ok := SectionBody(text, section)
	if !ok {
		return "", false
	}
	lines := strings.Split(parent, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "### ") && strings.HasPrefix(normalizeHeading(line[4:]), stem) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return "", false
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "### ") || strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n")), true
}

// SubsectionMissing reports whether the top-level "## section" span contains
// a "### " heading whose normalizeHeading form has stem as a prefix, with a
// substantive body (SectionMissing's own predicate: empty once HTML comments
// — the TEMPLATE.md placeholder form — are stripped). An absent parent
// section, or no sub-heading matching stem, both count as missing — the
// caller (GateViolations) does not need to distinguish the two.
func SubsectionMissing(text, section, stem string) bool {
	body, found := SubsectionBody(text, section, stem)
	if !found {
		return true
	}
	return strings.TrimSpace(htmlCommentRE.ReplaceAllString(body, "")) == ""
}

// GateViolation is one unmet flow.Requirement, as evaluated by GateViolations
// against one ticket's text.
type GateViolation struct{ Req flow.Requirement }

// Blocking reports whether this violation refuses a ticket move (as opposed
// to only ever warning) — flow.Requirement.Severity, restated as the
// question its two callers (internal/move, internal/audit) actually ask.
func (v GateViolation) Blocking() bool { return v.Req.Severity == flow.Blocking }

// Message renders the violation in one of exactly two forms, chosen by
// whether the Requirement names a sub-heading (Req.Sub) or the section as a
// whole. The first form, applied to brine's Outcome row, must reproduce
// T-083's original warning byte-for-byte —
// TestGateViolationMessages pins that.
func (v GateViolation) Message() string {
	if v.Req.Sub == "" {
		return fmt.Sprintf("## %s is missing, empty, or still a placeholder — %s", v.Req.Section, v.Req.Hint)
	}
	return fmt.Sprintf("## %s has no substantive \"### %s\" heading (%s) — %s",
		v.Req.Section, v.Req.Sub, v.Req.Label, v.Req.Hint)
}

// GateViolations evaluates every requirement in reqs (a state's gate table,
// via flow.Definition.Requirements) against text, and returns one
// GateViolation per unmet row, preserving table order. Both call sites
// (internal/move, before writing a move; internal/audit, on every ticket in
// its own state) drive from this single evaluator, so an unmet requirement
// is judged identically wherever it is checked (T-081).
func GateViolations(reqs []flow.Requirement, text string) []GateViolation {
	var out []GateViolation
	for _, r := range reqs {
		var missing bool
		if r.Sub == "" {
			missing = SectionMissing(text, r.Section)
		} else {
			missing = SubsectionMissing(text, r.Section, r.Sub)
		}
		if missing {
			out = append(out, GateViolation{Req: r})
		}
	}
	return out
}

// renderIDList renders a ticket-id slice in frontmatter form: "[]" when empty,
// "[T-018, T-019]" otherwise — the inverse of ParseDepends.
//
// Two other renderers do the same job today (internal/move's renderDepends and an
// inline join in internal/sync); collapsing all three onto this one is deliberately
// deferred to T-015 rather than done here.
func renderIDList(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	return "[" + strings.Join(ids, ", ") + "]"
}

// Scaffold renders a fresh, canonical TO DO ticket: filled frontmatter, heading,
// and the standard section skeleton (mirroring TEMPLATE.md's section set). The
// full TEMPLATE.md remains the authoring guide; this is the minimal,
// error-free starting point `pickle ticket new` writes. "Error-free", not
// warning-free: the `## Outcome` placeholder below is deliberately one that
// `board audit` flags (T-083), so a scaffolded ticket nudges its author to say
// what the feature buys until they do.
//
// spawnedBy is the ticket's lineage (nil for none) — provenance only; it never
// gates anything. family is the umbrella ticket id (a single id, same child, "" for
// none) — also lineage-only; unlike spawned-by (always rendered as `[]`) the
// family line is omitted entirely when empty, so a no-family scaffold is
// byte-identical to a ticket that never had the key (it is an optional frontmatter
// field, deliberately absent from the audit's requiredKeys).
func Scaffold(id, title, project, impact, complexity, cost string, spawnedBy []string, family string) string {
	date := time.Now().Format("2006-01-02")
	familyLine := ""
	if family != "" {
		familyLine = "family: " + family + "\n"
	}
	return fmt.Sprintf(`---
id: %s
title: %s
project: %s
depends-on: []
spawned-by: %s
%simpact: %s
complexity: %s
cost: %s
---

# %s — %s

## Outcome

<!-- TODO: 1-3 sentences, in user-observable terms: what changes when this ships. -->

## Description

<!-- TODO: describe this feature in prose (the current spec). Note soft couplings to other
tickets by id; hard dependencies go in depends-on: frontmatter (human-approved). -->

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- %s — created (TO DO). source: pickle ticket new
`, id, title, project, renderIDList(spawnedBy), familyLine, impact, complexity, cost, id, title, date)
}

// LoadAll reads every ticket under root/tickets/<status>/. Missing status dirs are
// treated as empty (git does not track empty dirs). The second return is a list of
// structural load problems (bad filename, no frontmatter) keyed by "<dir>/<file>".
func LoadAll(def *flow.Definition, root string) ([]*Ticket, []string) {
	var tickets []*Ticket
	var issues []string
	for _, s := range def.States() {
		dir := filepath.Join(root, "tickets", s.Dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // absent (vanished-empty) dir is not an error
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			ref := s.Dir + "/" + name
			m := filenameRE.FindStringSubmatch(name)
			if m == nil {
				issues = append(issues, ref+": filename does not match <PREFIX>-NNN-<slug>.md")
				continue
			}
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				issues = append(issues, ref+": "+err.Error())
				continue
			}
			text := string(data)
			fm, dupes, ok := parseFrontmatter(text)
			if !ok {
				issues = append(issues, ref+": no frontmatter block")
				continue
			}
			_, num, _ := SplitID(m[1])
			slug := strings.TrimSuffix(strings.TrimPrefix(name, m[1]+"-"), ".md")
			tickets = append(tickets, &Ticket{
				ID:            m[1],
				Num:           num,
				Slug:          slug,
				Dir:           s.Dir,
				Path:          path,
				Front:         fm,
				DependsOn:     ParseDepends(fm["depends-on"]),
				SpawnedBy:     ParseDepends(fm["spawned-by"]),
				Family:        strings.TrimSpace(fm["family"]),
				DuplicateKeys: dupes,
				Text:          text,
			})
		}
	}
	return tickets, issues
}
