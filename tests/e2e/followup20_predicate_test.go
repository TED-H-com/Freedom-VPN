//go:build e2e

package e2e

import (
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestD087_RequiresCleanAcceptedProvisionalReport(t *testing.T) {
	node := uuid.New()
	now := time.Now().UTC()
	body := map[string]any{"node_id": node, "provisioning_state": "ready", "safety_status": "passed", "lease": map[string]any{"lease_id": uuid.New()}, "forwarding_block_reasons": []string{"awaiting_member_grants"}, "errors": []map[string]string{{"stage": "lease", "code": "AWAITING_MEMBER_GRANTS"}}, "checks": []map[string]string{{"name": "lease", "error_code": "AWAITING_MEMBER_GRANTS"}, {"name": "qos", "error_code": "AWAITING_MEMBER_GRANTS"}, {"name": "source_binding", "error_code": "AWAITING_MEMBER_GRANTS"}}}
	e := GrantWireEvent{Kind: "observation", Status: 202, Started: now}
	e.Request, _ = json.Marshal(body)
	if _, ok := qualifyingGrantReport(e, node, now.Add(-time.Second), true); !ok {
		t.Fatal("clean report refused")
	}
	for _, v := range []struct {
		key   string
		value any
	}{{"safety_status", "failed"}, {"forwarding_block_reasons", []string{"awaiting_member_grants", "lease_expired"}}, {"member_forwarding_enabled", true}, {"errors", []map[string]string{{"stage": "lease", "code": "AWAITING_MEMBER_GRANTS"}, {"stage": "safety", "code": "GENERATION_MISMATCH"}}}} {
		clone := map[string]any{}
		for k, a := range body {
			clone[k] = a
		}
		clone[v.key] = v.value
		e.Request, _ = json.Marshal(clone)
		if _, ok := qualifyingGrantReport(e, node, now.Add(-time.Second), true); ok {
			t.Fatal("dirty report qualified", v.key)
		}
	}
	e.Request, _ = json.Marshal(body)
	e.Status = 503
	if _, ok := qualifyingGrantReport(e, node, now.Add(-time.Second), true); ok {
		t.Fatal("unaccepted report qualified")
	}
}
