package httpapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

type usageGateProbe struct {
	body *gen.UsageBatchRequest
}

func (p *usageGateProbe) SubmitUsageBatch(_ context.Context, r gen.SubmitUsageBatchRequestObject) (gen.SubmitUsageBatchResponseObject, error) {
	p.body = r.Body
	return nil, apierr.New(apierr.StateConflict, "TEST ONLY service boundary reached")
}

func TestMTR03_D115UsageGateDelegatesOnlyExplicitOrigin(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/UsageBatchRequest.handoff-accepted.json")
	if err != nil {
		t.Fatal(err)
	}
	var original gen.UsageBatchRequest
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	p := principal(auth.Agent)
	p.NodeID, p.IncarnationID = original.NodeId, original.NodeIncarnationId
	for _, large := range []bool{false, true} {
		name := "ordinary"
		if large {
			name = "streaming"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name string
				edit func(*gen.UsageBatchRequest)
				pass bool
			}{
				{"handoff", func(*gen.UsageBatchRequest) {}, true},
				{"live-mismatch", func(r *gen.UsageBatchRequest) { r.OriginIncarnationId = nil }, false},
				{"live-matching", func(r *gen.UsageBatchRequest) {
					r.OriginIncarnationId = nil
					r.Events[0].NodeIncarnationId = p.IncarnationID
				}, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var body gen.UsageBatchRequest
					if err := json.Unmarshal(raw, &body); err != nil {
						t.Fatal(err)
					}
					tc.edit(&body)
					encoded, err := json.Marshal(body)
					if err != nil {
						t.Fatal(err)
					}
					wire := string(encoded)
					if large {
						wire += strings.Repeat(" ", canonjson.MaxBytes)
					}
					probe := &usageGateProbe{}
					h := fixtureConfig(t, p, true, nil, wideRates(config.Defaults()), probe)
					r := request(http.MethodPost, "/api/v1/agent/usage-batches", wire, p)
					r.Header.Del("Idempotency-Key")
					r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
					status := http.StatusForbidden
					if tc.pass {
						status = http.StatusConflict // Sent only by the service-boundary probe.
					}
					assertStatus(t, h, r, status)
					if (probe.body != nil) != tc.pass {
						t.Fatalf("service reached=%v want=%v", probe.body != nil, tc.pass)
					}
					if tc.pass {
						got, err := json.Marshal(probe.body)
						if err != nil || string(got) != string(encoded) {
							t.Fatalf("middleware rewrote event or presenting identity: %s (%v)", got, err)
						}
					}
				})
			}
		})
	}
}

func TestMTR03_D115UsageGateKeepsAuthenticationAndValidation(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/UsageBatchRequest.handoff-accepted.json")
	if err != nil {
		t.Fatal(err)
	}
	var body gen.UsageBatchRequest
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	p := principal(auth.Agent)
	p.NodeID, p.IncarnationID = body.NodeId, body.NodeIncarnationId
	for _, large := range []bool{false, true} {
		for _, tc := range []struct {
			name, body string
			status     int
			noTLS      bool
			key        string
		}{
			{"no-mtls", string(raw), 401, true, ""},
			{"null-origin", strings.Replace(string(raw), `"origin_incarnation_id": "`+body.OriginIncarnationId.String()+`"`, `"origin_incarnation_id": null`, 1), 422, false, ""},
			{"duplicate-key", strings.Replace(string(raw), "{", `{"node_id":"`+body.NodeId.String()+`",`, 1), 400, false, ""},
			{"unknown-field", strings.Replace(string(raw), "{", `{"unknown":true,`, 1), 422, false, ""},
			{"invalid-idempotency", string(raw), 400, false, "invalid"},
		} {
			t.Run(tc.name+fmtUsageSize(large), func(t *testing.T) {
				wire := tc.body
				if large {
					wire += strings.Repeat(" ", canonjson.MaxBytes)
				}
				probe := &usageGateProbe{}
				h := fixtureConfig(t, p, true, nil, wideRates(config.Defaults()), probe)
				r := request(http.MethodPost, "/api/v1/agent/usage-batches", wire, p)
				r.Header.Del("Idempotency-Key")
				if tc.key != "" {
					r.Header.Set("Idempotency-Key", tc.key)
				}
				if !tc.noTLS {
					r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
				}
				assertStatus(t, h, r, tc.status)
				if probe.body != nil {
					t.Fatal("invalid request reached usage service")
				}
			})
		}
	}
	c := wideRates(config.Defaults())
	c.UsageBodyBytes = int64(len(raw) - 1)
	r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(raw), p)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	assertStatus(t, fixtureConfig(t, p, true, nil, c), r, http.StatusRequestEntityTooLarge)

	// The exception is operation-scoped: another agent operation still rejects
	// a mismatched incarnation and ignores caller-supplied identity headers.
	lease := `{"renewal_request_id":"` + uuid.NewString() + `","node_id":"` + p.NodeID.String() + `","node_incarnation_id":"` + uuid.NewString() + `","current_lease_id":null,"applied_config_generation":null}`
	r = request(http.MethodPost, "/api/v1/agent/lease-renewals", lease, p)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	r.Header.Set("X-Node-ID", p.NodeID.String())
	assertStatus(t, fixture(t, p, true, nil), r, http.StatusForbidden)
}

func fmtUsageSize(large bool) string {
	if large {
		return "/streaming"
	}
	return "/ordinary"
}
