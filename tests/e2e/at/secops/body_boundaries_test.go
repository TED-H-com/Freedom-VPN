//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"time"
)

func (p *liveProbe) otherBodyBoundaries(admin, node *liveSession, b api.EnrollResponse, realNode uuid.UUID) {
	var observation []byte
	if e := p.db.QueryRow(p.t.Context(), `SELECT report_payload FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC LIMIT 1`, realNode).Scan(&observation); e != nil {
		p.t.Fatal("real observation unavailable for identity-negative body boundary")
	}
	// Replay an immutable actual node-A report under node-B mTLS. Identity must
	// reject it; no synthetic telemetry or projected row is written.
	req, _ := pkitest.Request(p.t)
	enroll, _ := json.Marshal(req)
	lease, _ := json.Marshal(api.LeaseRenewalRequest{NodeId: b.NodeId, NodeIncarnationId: b.NodeIncarnationId, RenewalRequestId: uuid.New(), CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()})
	anon := &liveSession{client: p.client()}
	probes := []struct {
		Name, Path string
		Session    *liveSession
		Body       []byte
		Headers    map[string]string
	}{
		{"browser", "/api/v1/admin/groups", admin, []byte(`{"code":"secops-body-boundary","name":"TEST ONLY boundary"}`), map[string]string{"Idempotency-Key": uuid.NewString()}},
		{"agent", liveAgent + "/api/v1/agent/bootstrap", node, nil, map[string]string{"Idempotency-Key": ""}},
		{"heartbeat", liveAgent + "/api/v1/agent/observations", node, observation, nil},
		{"lease", liveAgent + "/api/v1/agent/lease-renewals", node, lease, map[string]string{"Idempotency-Key": ""}},
		{"enrollment", "/api/v1/agent/enroll", anon, enroll, map[string]string{"Authorization": "Enrollment TEST-ONLY-invalid"}},
		{"webhook", "/api/v1/webhooks/stripe", anon, []byte(`{}`), nil},
	}
	out := []map[string]any{}
	for _, probe := range probes {
		baseline := p.call(probe.Session, "POST", probe.Path, probe.Body, probe.Headers)
		for _, size := range []int{(1 << 20) - 1, 1 << 20, (1 << 20) + 1} {
			time.Sleep(1100 * time.Millisecond)
			raw := append(append([]byte{}, probe.Body...), bytes.Repeat([]byte(" "), size-len(probe.Body))...)
			r := p.call(probe.Session, "POST", probe.Path, raw, probe.Headers)
			want := baseline.status
			if size > 1<<20 {
				want = 413
			}
			// Bodyless bootstrap rejects payloads at schema (422); the class cap still
			// must run first. Baseline only diagnoses route availability.
			if probe.Name == "agent" && size <= 1<<20 {
				want = 400
			}
			out = append(out, map[string]any{"class": probe.Name, "bytes": size, "baseline_status": baseline.status, "expected_status": want, "actual_status": r.status, "code": r.code, "passed": r.status == want, "body_scope": "real schema-valid request except bodyless bootstrap whitespace and signature-negative webhook; heartbeat is real immutable node A report refused under B identity"})
			if r.status != want {
				p.t.Errorf("body boundary %s bytes=%d expected=%d actual=%d code=%s", probe.Name, size, want, r.status, r.code)
			}
		}
	}
	p.metadata["other_body_boundaries"] = out
}
