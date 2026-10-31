package httpapi

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
)

func TestReviewR17_UsageOriginGateIsOperationScoped(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/UsageBatchRequest.handoff-accepted.json")
	if err != nil {
		t.Fatal(err)
	}
	var original gen.UsageBatchRequest
	if err = json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	p := principal(auth.Agent)
	p.NodeID, p.IncarnationID = original.NodeId, original.NodeIncarnationId
	origin := original.OriginIncarnationId.String()

	if usageOriginSelected([]byte(`{}`)) || usageOriginSelected([]byte(`{"origin_incarnation_id":null}`)) || usageOriginSelected([]byte(`{"origin_incarnation_id":"not-a-uuid"}`)) {
		t.Fatal("null, absent, or non-UUID origin was treated as a hand-off")
	}
	if !usageOriginSelected([]byte(`{"origin_incarnation_id":"` + origin + `"}`)) {
		t.Fatal("schema-valid origin was not selected")
	}
	if usageOriginSelected([]byte(`{"events":[{"origin_incarnation_id":"` + origin + `"}]}`)) {
		t.Fatal("a nested origin selected the hand-off")
	}

	var absent gen.UsageBatchRequest
	if err = json.Unmarshal(raw, &absent); err != nil {
		t.Fatal(err)
	}
	absent.OriginIncarnationId = nil
	absentRaw, err := json.Marshal(absent)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		body   string
		status int
		reach  bool
	}{
		{"non-uuid", strings.Replace(string(raw), `"`+origin+`"`, `"not-a-uuid"`, 1), http.StatusBadRequest, false},
		{"null", strings.Replace(string(raw), `"origin_incarnation_id": "`+origin+`"`, `"origin_incarnation_id": null`, 1), http.StatusUnprocessableEntity, false},
		{"absent-mismatch", string(absentRaw), http.StatusForbidden, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := &usageGateProbe{}
			h := fixtureConfig(t, p, true, nil, wideRates(config.Defaults()), probe)
			r := request(http.MethodPost, "/api/v1/agent/usage-batches", tc.body, p)
			r.Header.Del("Idempotency-Key")
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			assertStatus(t, h, r, tc.status)
			if (probe.body != nil) != tc.reach {
				t.Fatalf("service reached=%v", probe.body != nil)
			}
		})
	}

	// A lease renewal that carries the usage field still fails the body
	// identity check. The exemption is SubmitUsageBatch only.
	lease := `{"renewal_request_id":"` + uuid.NewString() + `","node_id":"` + p.NodeID.String() + `","node_incarnation_id":"` + uuid.NewString() + `","origin_incarnation_id":"` + origin + `","current_lease_id":null,"applied_config_generation":null}`
	r := request(http.MethodPost, "/api/v1/agent/lease-renewals", lease, p)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	w := httptest.NewRecorder()
	fixture(t, p, true, nil).ServeHTTP(w, r)
	if w.Code == http.StatusOK || w.Code == http.StatusConflict {
		t.Fatalf("lease identity check depended on the usage exemption: %d %s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Fatalf("lease status %d %s", w.Code, w.Body.String())
	}
}
