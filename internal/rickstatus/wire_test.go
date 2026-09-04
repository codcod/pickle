package rickstatus

import "testing"

// TestParseSkipsTicketsWithEmptyID (review finding F9): a wire ticket entry
// with no id — malformed input rick should never actually produce, but this
// package projects untrusted external JSON and must not assume it — is
// dropped rather than added to Report.Tickets under the empty-string key,
// which would make For("") a nonsensical lookup and silently swallow real
// data under a collision if more than one such entry appeared.
func TestParseSkipsTicketsWithEmptyID(t *testing.T) {
	out := []byte(`{
  "schemaVersion": 2,
  "workflow": {
    "tickets": [
      {"id": "", "artifacts": [{"path": "docs/specs//x.md", "kind": "task", "status": "draft"}]},
      {"id": "DR-1", "artifacts": [{"path": "docs/specs/DR-1/x.md", "kind": "task", "status": "draft"}]}
    ]
  }
}`)
	rep, err := parse(out)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if _, ok := rep.Tickets[""]; ok {
		t.Error(`parse() kept an entry under the empty-string key, want it dropped`)
	}
	if len(rep.Tickets) != 1 {
		t.Errorf("Tickets = %v, want exactly the one entry with a real id", rep.Tickets)
	}
	if len(rep.For("DR-1")) != 1 {
		t.Errorf("For(DR-1) = %v, want the one real entry unaffected", rep.For("DR-1"))
	}
}
