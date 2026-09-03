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
// belongs to the whole-ticket check, not this per-field primitive). In
// practice a duplicate of *any* key still trips the parse-back guard below
// on its own — verifyFrontmatterEdit re-parses the whole frontmatter block,
// not just the targeted key, so it reports a duplicate wherever one is,
// refusing the edit either way (a stricter behaviour than SetField needs to
// promise, not a documented contract of it — do not rely on this from
// outside the package; the whole-ticket precondition above is still the one
// callers must keep enforcing).
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
	_, dupes, ok := parseFrontmatter(text)
	if !ok {
		return "", fmt.Errorf("no frontmatter block")
	}
	if len(dupes) > 0 {
		return "", fmt.Errorf("frontmatter has duplicate key(s) %s; refusing rather than guessing which value is current", strings.Join(dupes, ", "))
	}
	titleAt := findFrontmatterKeyLine(lines, closeAt, "title")
	if titleAt == -1 {
		return "", fmt.Errorf("frontmatter has no %q key to set (a required key can only be missing on an already-malformed ticket)", "title")
	}
	// currentTitle is read straight from the frontmatter line's own raw
	// capture (fmKeyRE), not through parseFrontmatter's fm["title"] — which
	// also strips a surrounding quote character. A markdown heading has no
	// quoting convention to mirror that, so comparing against the
	// already-quote-stripped parsed value would treat a genuine
	// quote-boundary disagreement between the two copies (H1 quoted,
	// frontmatter not, or vice versa — a plausible result of hand-adding
	// YAML-style quoting to only one copy) as agreement, silently papering
	// over exactly the drift this function exists to catch (T-102 rework
	// round 2, finding G0).
	currentTitleRaw := fmKeyRE.FindStringSubmatch(lines[titleAt])[2]
	currentTitle := strings.TrimSpace(currentTitleRaw)

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
	// Compared trimmed of whitespace only, not raw and not quote-stripped
	// (T-102 rework, F0 then G0): a frontmatter `key: value` line cannot tell
	// the value's own leading whitespace apart from the mandatory separator
	// space (fmKeyRE's `\s*` consumes both alike), so that — and only that —
	// ambiguity is normalized away, symmetrically, on both sides. Anything
	// else that differs between the two raw copies, including a boundary
	// quote character, is a real disagreement and must still refuse.
	h1RawTitle := h1RE.FindStringSubmatch(lines[h1At])[1]
	if got := strings.TrimSpace(h1RawTitle); got != currentTitle {
		return "", fmt.Errorf("heading %q and frontmatter title %q already disagree; fix by hand first", h1RawTitle, currentTitleRaw)
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
// reads back as the intended value (compared normalized — see
// normalizeFrontmatterValue), and no duplicate key is now reported. The
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
	if got, want := fm[key], normalizeFrontmatterValue(value); got != want {
		return fmt.Errorf("could not set %s (it would end up %q, not %q); set it by hand", key, got, value)
	}
	return nil
}

// normalizeFrontmatterValue mirrors exactly what parseFrontmatter's own
// per-line scan does to a value: trim surrounding whitespace, then trim any
// surrounding matching quote characters. verifyFrontmatterEdit needs it
// (T-102 rework, finding F0): title is the one settable field whose legal
// values can carry leading/trailing whitespace (ticket.ValidateTitle does
// not reject it), and a frontmatter `key: value` line cannot preserve a
// value's own leading whitespace distinguishably from the mandatory
// separator space — fmKeyRE's `\s*` consumes both alike — so an exact,
// unnormalized comparison refuses a legal padded value regardless of
// whether anything actually drifted. Comparing what a real re-parse would
// report (which is exactly this normalization, applied by parseFrontmatter
// itself) against this normalized form of the intended value is what makes
// the check test title *content* agreement, not an artifact of the
// frontmatter line format itself.
//
// Deliberately NOT used by setTitleAndHeading's H1-vs-frontmatter drift
// precheck (T-102 rework round 2, finding G0): that check's job is
// detecting whether the *current* ticket already disagrees with itself, and
// a markdown heading has no quoting convention to mirror parseFrontmatter's
// quote-stripping — stripping quotes there would treat a genuine
// quote-boundary disagreement as agreement. That check trims whitespace
// only, symmetrically, on its own raw captures instead.
func normalizeFrontmatterValue(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
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
