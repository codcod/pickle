// T-102: a line-based, parse-back-guarded writer for the five
// frontmatter/heading fields `pickle ticket set` may change.
//
// This is deliberately NOT a full frontmatter re-render: it locates and
// rewrites exactly the one targeted line (plus, for a title edit, the H1
// line), leaving every other line — including a key this binary does not
// recognize — byte-for-byte untouched. That is what lets the parse-back
// guard below assert "nothing else changed" as a literal line-diff rather
// than a semantic tree comparison (config's TOML equivalent,
// verifyOnlyPayloadVersion, needs tree-decode because TOML has multiple
// representations of equivalent content; flat one-key-per-line ticket
// frontmatter does not), and it is what preserves an unknown key verbatim
// with no separate check — the schema_version-style hazard T-065's
// refinement parked against this ticket (NOTES.md § "If you proceed: the
// first batch", row 1).
//
// SetField trusts its caller to have already refused on a ticket whose
// DuplicateKeys is non-empty (internal/ticketset.Set does this, over the
// whole ticket, before ever calling in here — the Description requires
// refusing rather than repairing a duplicate, and that precondition
// belongs to the whole-ticket check, not this per-field primitive). Called
// directly on a ticket whose *targeted* key is itself duplicated, the
// parse-back guard below still refuses on its own: last-wins re-parsing
// would report a value that disagrees with the one just written unless the
// edited occurrence happens to be the last one, in which case the edit
// legitimately took effect and the untouched duplicate is unrepaired, not
// corrupted. A duplicate of some *other* key is invisible to this function
// by design; it is the caller's job to have already refused for any
// duplicate, not just the one being edited.
package ticket

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// SettableFields lists the frontmatter/heading fields `pickle ticket set`
// may change. depends-on: is deliberately absent — it is list-valued and
// gates pickup, out of scope for a single-value surgical edit.
var SettableFields = []string{"impact", "complexity", "cost", "family", "title"}

// SetField returns text with field rewritten to value, or a refusal error
// naming what went wrong. id is the ticket's id — used only by the title
// path, to find its one legal `# <id> — …` heading line.
func SetField(text, id, field, value string) (string, error) {
	switch field {
	case "impact", "complexity", "cost", "family":
		return setFrontmatterLine(text, field, value)
	case "title":
		return setTitleAndHeading(text, id, value)
	default:
		return "", fmt.Errorf("field %q is not settable (legal: %s)", field, strings.Join(SettableFields, ", "))
	}
}

// frontmatterBounds returns the line index of the closing "---" delimiter
// (the opening one is always 0), or ok=false if text has no frontmatter
// block at all — the same shape parseFrontmatter itself checks first.
func frontmatterBounds(lines []string) (closeAt int, ok bool) {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return 0, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i, true
		}
	}
	return 0, false
}

// findFrontmatterKeyLine returns the index of the line for key inside
// lines[1:closeAt), or -1 if key is not present. When key appears more than
// once, it returns the *last* occurrence — mirroring parseFrontmatter's own
// last-wins read, so the index found here is the one whose value LoadAll
// would actually have reported as the field's current value.
func findFrontmatterKeyLine(lines []string, closeAt int, key string) int {
	target := -1
	for i := 1; i < closeAt; i++ {
		if m := fmKeyRE.FindStringSubmatch(lines[i]); m != nil && m[1] == key {
			target = i
		}
	}
	return target
}

// setFrontmatterLine rewrites key's line to "key: value". impact, complexity
// and cost are required keys (Scaffold always writes them) and must already
// have a line to rewrite — a missing one means the ticket is already
// malformed, and this refuses rather than guessing where to add it. family
// is optional (T-059): when absent, a new "family: value" line is inserted
// immediately before "impact:", mirroring Scaffold's own key order
// (internal/ticket/ticket.go's Scaffold).
func setFrontmatterLine(text, key, value string) (string, error) {
	lines := strings.Split(text, "\n")
	closeAt, ok := frontmatterBounds(lines)
	if !ok {
		return "", fmt.Errorf("no frontmatter block")
	}

	newLine := key + ": " + value
	target := findFrontmatterKeyLine(lines, closeAt, key)

	var updated []string
	if target != -1 {
		updated = slices.Clone(lines)
		updated[target] = newLine
		if !onlyLinesChanged(lines, updated, target) {
			return "", fmt.Errorf("setting %s would touch more than the one intended line; refusing", key)
		}
	} else {
		if key != "family" {
			return "", fmt.Errorf("frontmatter has no %q key to set (a required key can only be missing on an already-malformed ticket)", key)
		}
		impactAt := findFrontmatterKeyLine(lines, closeAt, "impact")
		if impactAt == -1 {
			return "", fmt.Errorf("frontmatter has no %q key to anchor a new %s: line on (a required key can only be missing on an already-malformed ticket)", "impact", key)
		}
		updated = make([]string, 0, len(lines)+1)
		updated = append(updated, lines[:impactAt]...)
		updated = append(updated, newLine)
		updated = append(updated, lines[impactAt:]...)
		if len(updated) != len(lines)+1 ||
			!slices.Equal(updated[:impactAt], lines[:impactAt]) ||
			updated[impactAt] != newLine ||
			!slices.Equal(updated[impactAt+1:], lines[impactAt:]) {
			return "", fmt.Errorf("inserting %s: would touch more than the one intended line; refusing", key)
		}
	}

	result := strings.Join(updated, "\n")
	if err := verifyFrontmatterEdit(result, key, value); err != nil {
		return "", err
	}
	return result, nil
}

// setTitleAndHeading rewrites the frontmatter title: line and the ticket's
// H1 (`# <id> — <title>`) together. It refuses up front if the two do not
// already agree exactly — this ticket is what first makes that agreement
// checkable, so it must not silently paper over one that has already
// drifted (a stale slug is legal per rules §3; a stale H1 is not the same
// thing and this function has no way to know which of the two disagreeing
// values is the intended one).
func setTitleAndHeading(text, id, newTitle string) (string, error) {
	lines := strings.Split(text, "\n")
	closeAt, ok := frontmatterBounds(lines)
	if !ok {
		return "", fmt.Errorf("no frontmatter block")
	}
	fm, dupes, ok := parseFrontmatter(text)
	if !ok {
		return "", fmt.Errorf("no frontmatter block")
	}
	if len(dupes) > 0 {
		return "", fmt.Errorf("frontmatter has duplicate key(s) %s; refusing rather than guessing which value is current", strings.Join(dupes, ", "))
	}
	currentTitle := fm["title"]
	titleAt := findFrontmatterKeyLine(lines, closeAt, "title")
	if titleAt == -1 {
		return "", fmt.Errorf("frontmatter has no %q key to set (a required key can only be missing on an already-malformed ticket)", "title")
	}

	h1RE := regexp.MustCompile(`^# ` + regexp.QuoteMeta(id) + ` — (.+)$`)
	h1At := -1
	for i := closeAt + 1; i < len(lines); i++ {
		if h1RE.MatchString(lines[i]) {
			if h1At != -1 {
				return "", fmt.Errorf("more than one %q heading; refusing", "# "+id+" — …")
			}
			h1At = i
		}
	}
	if h1At == -1 {
		return "", fmt.Errorf("no %q heading found to update", "# "+id+" — …")
	}
	if got := h1RE.FindStringSubmatch(lines[h1At])[1]; got != currentTitle {
		return "", fmt.Errorf("heading %q and frontmatter title %q already disagree; fix by hand first", got, currentTitle)
	}

	updated := slices.Clone(lines)
	updated[titleAt] = "title: " + newTitle
	updated[h1At] = "# " + id + " — " + newTitle
	if !onlyLinesChanged(lines, updated, titleAt, h1At) {
		return "", fmt.Errorf("setting title would touch more than the two intended lines; refusing")
	}

	result := strings.Join(updated, "\n")
	if err := verifyFrontmatterEdit(result, "title", newTitle); err != nil {
		return "", err
	}
	return result, nil
}

// verifyFrontmatterEdit is the parse-back half of the guard: it re-parses
// the candidate text's frontmatter and refuses unless it still parses, key
// reads back as exactly value, and no duplicate key is now reported. The
// caller's own line-diff check is the other half — together they turn
// "nothing else changed" into a checked claim rather than a hope, the same
// shape as config's verifyOnlyPayloadVersion.
func verifyFrontmatterEdit(after, key, value string) error {
	fm, dupes, ok := parseFrontmatter(after)
	if !ok {
		return fmt.Errorf("setting %s would leave the file without a parseable frontmatter block; set it by hand", key)
	}
	if len(dupes) > 0 {
		return fmt.Errorf("setting %s would leave a duplicate frontmatter key (%s); set it by hand", key, strings.Join(dupes, ", "))
	}
	if got := fm[key]; got != value {
		return fmt.Errorf("could not set %s (it would end up %q, not %q); set it by hand", key, got, value)
	}
	return nil
}

// onlyLinesChanged reports whether a and b are the same length and differ
// only at indices in allowed. An allowed index is *permitted* to be
// unchanged too — setting a field to the value it already had must not
// spuriously refuse just because nothing actually moved.
func onlyLinesChanged(a, b []string, allowed ...int) bool {
	if len(a) != len(b) {
		return false
	}
	allow := make(map[int]bool, len(allowed))
	for _, i := range allowed {
		allow[i] = true
	}
	for i := range a {
		if a[i] != b[i] && !allow[i] {
			return false
		}
	}
	return true
}
