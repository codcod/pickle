package rickstatus

import (
	"encoding/json"
	"fmt"
)

// wireDoc, wireTicket and wireArtifact are `rick status --json`'s own shape,
// decoded and immediately projected into Report/Artifact — never returned to
// a caller directly (T-076 decision 6), the same boundary internal/state's
// package doc draws between a wire format and a package's own types.
//
// CONFIRMED, not guessed: this workspace's development machine happens to
// have the real `rick` CLI installed (`ig/uk/rick` v0.10.0, the exact
// `github.com/ig-private/ai-sdlc/sdlc-cli` binary T-075/T-076 cite), so these
// tags were pinned empirically — a scratch `docs/specs/<ID>/*.md` fixture run
// through `rick status --json` — rather than taken on trust from the
// ticket's prose. That surfaced one real correction: a ticket entry's
// identifying field is `id`, not the `key` this package originally assumed
// (T-076's own refinement flagged this exact field as the one thing that
// could not be verified in advance). `date` is confirmed omitempty — present
// only when the artifact's frontmatter carries a `date:` field, absent for
// e.g. a `research-*.md` artifact with none. The installed binary reports
// `schemaVersion: 1`; T-075's Description cites `schemaVersion = 2` for a
// newer rick, so SchemaVersion (rickstatus.go) still gates on 2, but the
// shape below — `workflow.tickets[].{id, artifacts[].{path, kind, status,
// date}}` — is real, not provisional, and additive-only evolution (the
// contract's own promise) does not touch already-shipped fields.
type wireDoc struct {
	SchemaVersion int `json:"schemaVersion"`
	Workflow      struct {
		Tickets []wireTicket `json:"tickets"`
	} `json:"workflow"`
}

type wireTicket struct {
	ID        string         `json:"id"`
	Artifacts []wireArtifact `json:"artifacts"`
}

type wireArtifact struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Date   string `json:"date,omitempty"`
}

// parse decodes rick status --json's output and projects it into a Report.
// An unrecognised SchemaVersion is refused outright (T-076 decision 5, the
// exact-match policy) rather than best-effort parsed.
func parse(out []byte) (Report, error) {
	var doc wireDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		return Report{}, fmt.Errorf("%s status --json did not return valid JSON: %w", command, err)
	}
	if doc.SchemaVersion != SchemaVersion {
		return Report{}, fmt.Errorf("%s status --json schema %d not supported (want %d)", command, doc.SchemaVersion, SchemaVersion)
	}

	tickets := make(map[string][]Artifact, len(doc.Workflow.Tickets))
	for _, wt := range doc.Workflow.Tickets {
		if wt.ID == "" {
			continue
		}
		artifacts := make([]Artifact, 0, len(wt.Artifacts))
		for _, wa := range wt.Artifacts {
			artifacts = append(artifacts, Artifact{
				Path:   wa.Path,
				Kind:   wa.Kind,
				Status: wa.Status,
				Date:   wa.Date,
			})
		}
		tickets[wt.ID] = artifacts
	}

	return Report{Available: true, Tickets: tickets}, nil
}
