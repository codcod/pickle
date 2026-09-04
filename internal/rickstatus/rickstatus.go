// Package rickstatus consumes `rick status --json` as an artifact-state
// source for one registered child-project (T-076).
//
// rick (the ai-sdlc project, a separate GitLab repo — not a registered
// child-project here) drives a ticket through Research → Plan → Implement →
// Validate, each phase writing an artifact under docs/specs/<KEY>/ that a
// human approves at a gate. `rick status --json` is documented (T-075's
// Description, citing sdlc-cli/internal/status/report.go) as a versioned
// public contract — schemaVersion 2, additive-only — whose
// Workflow.Tickets[].Artifacts[] carries Path/Kind/Status/Date per artifact.
// This package shells out to that command and projects its answer into
// Report/Artifact; it never re-derives rick's state by scanning
// docs/specs/** itself, so it can never fork rick's own kind-detection or
// status rules.
//
// Fail-open is a type-level guarantee, not a convention callers have to
// remember (T-076 decision 8): rick not opted in, not on PATH, erroring,
// timing out, returning malformed JSON, or reporting an unrecognised
// SchemaVersion all collapse to the same shape — Report{Available: false,
// Reason: "…"} — never an error return from Query. T-075's invariant is that
// this must never become a new way for anything to fail loudly in a project
// that has never heard of rick, or hit a transient rick failure.
//
// The wire field names in wire.go were empirically confirmed against a real
// installed `rick` binary during implementation (see wire.go's doc comment)
// rather than taken on trust from T-075/T-076's own Description text —
// which caught one real mismatch (a ticket entry's key is `id`, not `key`).
// Even so, Query's fail-open design means a future field rename upstream
// still degrades to "no artifacts shown" rather than a crash or a wrong
// answer, so a next mismatch remains cheap to fix from a fresh capture.
package rickstatus

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/codcod/pickle/internal/config"
)

// SchemaVersion is the rick status --json schema this build understands.
// Exact-match, not a floor (`>=`) — mirrors internal/state.CurrentSchema's
// precedent rather than assuming rick's additive-only promise makes a higher
// version safe to best-effort parse (T-076 decision 5). Raising this when
// rick ships a schema 3 is this package's own follow-up.
const SchemaVersion = 2

// command is the external binary this package shells out to, resolved from
// PATH only — never a configured path, matching internal/hook.Probe and
// internal/vcs's existing convention of trusting PATH for an external tool.
const command = "rick"

// rickTimeout bounds how long a query may block. A var, not a const, so a
// test can shrink it (mirrors internal/vcs.probeTimeout).
var rickTimeout = 5 * time.Second

// Artifact is one rick artifact, projected from the wire format.
type Artifact struct {
	Path   string
	Kind   string
	Status string // rick's own vocabulary, verbatim — never revalidated here
	Date   string
}

// Report is one child's rick-artifact projection. Available is true only
// when a fresh, parseable, recognised-schema result was obtained; Reason
// explains why not otherwise, for an optional diagnostic line (doctor) —
// never surfaced as an error. Tickets is keyed on whatever rick reports as
// each entry's `id`, verbatim (T-076 decision 7): T-058 already makes a
// pickle ticket id and rick's id the same string when a child's
// ticket_prefix is configured correctly, so no further mapping is needed.
type Report struct {
	Available bool
	Reason    string
	Tickets   map[string][]Artifact
}

// For returns the artifacts rick reported for one ticket id, or nil when
// there are none (including when the whole Report is unavailable).
func (r Report) For(id string) []Artifact {
	return r.Tickets[id]
}

// unavailable builds a Report carrying reason and nothing else — the single
// shape every failure path below returns (T-076 decision 8).
func unavailable(reason string) Report {
	return Report{Reason: reason}
}

// Query shells out to `rick status --json` for one registered child and
// projects the result into a Report. It always returns a usable, non-nil
// Report and never an error: every failure mode is folded into
// Report.Available/Reason (T-076 decision 8).
func Query(root string, p *config.Project) Report {
	if p == nil || !p.Rick {
		return unavailable("rick interop not enabled for this child")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rickTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, "status", "--json")
	cmd.Dir = childDir(root, p)
	out, err := cmd.Output()
	if err != nil {
		return unavailable(execFailureReason(err))
	}

	report, err := parse(out)
	if err != nil {
		return unavailable(err.Error())
	}
	return report
}

// childDir resolves a registered child's directory, the cwd `rick status
// --json` is invoked from (T-076 decision 4). root is already absolute
// (config.Config.Root()); p.Path is relative to it.
func childDir(root string, p *config.Project) string {
	return filepath.Join(root, p.Path)
}

// execFailureReason turns an exec error into a human-readable Reason,
// distinguishing the cases worth naming separately: the binary missing from
// PATH, the command timing out, and a non-zero exit.
func execFailureReason(err error) string {
	var exitErr *exec.ExitError
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return fmt.Sprintf("%s not found on PATH", command)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("%s status --json timed out after %s", command, rickTimeout)
	case errors.As(err, &exitErr):
		return fmt.Sprintf("%s status --json exited %d", command, exitErr.ExitCode())
	default:
		return fmt.Sprintf("%s status --json failed: %v", command, err)
	}
}
