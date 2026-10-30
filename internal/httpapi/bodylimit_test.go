package httpapi

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// padJSONObject inserts spaces after the opening brace so the document is
// exactly n bytes. The padding is legal JSON whitespace.
func padJSONObject(t *testing.T, doc []byte, n int) []byte {
	t.Helper()
	if len(doc) == 0 || doc[0] != '{' {
		t.Fatalf("padJSONObject: document must be an object, got %q", doc)
	}
	if n < len(doc) {
		t.Fatalf("padJSONObject: target %d is shorter than the document %d", n, len(doc))
	}
	out := make([]byte, n)
	out[0] = '{'
	pad := n - len(doc)
	for i := range pad {
		out[1+i] = ' '
	}
	copy(out[1+pad:], doc[1:])
	return out
}

func wideRates(c config.Config) config.Config {
	wide := config.Bucket{PerSecond: 1000, Burst: 1000}
	c.IngressRate = wide
	c.BrowserRate = wide
	c.AgentRate = wide
	c.UsageRate = wide
	c.AnonymousRate = wide
	return c
}

func schemaUsageBatch(p auth.Principal) []byte {
	member := uuid.New()
	device := uuid.New()
	body := api.UsageBatchRequest{
		NodeId:            openapi_types.UUID(p.NodeID),
		NodeIncarnationId: openapi_types.UUID(p.IncarnationID),
		LedgerEpoch:       presence.NewNullNullable[openapi_types.UUID](),
		Events: []api.UsageEvent{{
			SchemaVersion:     api.UsageEventSchemaVersionN1,
			NodeId:            openapi_types.UUID(p.NodeID),
			NodeIncarnationId: openapi_types.UUID(p.IncarnationID),
			MeterEpochId:      openapi_types.UUID(uuid.New()),
			EventSeq:          decstr.U64(1),
			MeteringVersion:   api.MeteringVersionForwardAdmissionV1,
			IntervalStart:     time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC),
			IntervalEnd:       time.Date(2026, 10, 28, 12, 1, 0, 0, time.UTC),
			Quality:           api.UsageQualityComplete,
			Records: []api.UsageRecord{{
				MemberId:        openapi_types.UUID(member),
				DeviceId:        openapi_types.UUID(device),
				BindingRevision: decstr.U64(1),
				PeriodId:        api.PeriodId("2026-10"),
				UploadBytes:     decstr.U64(1),
				DownloadBytes:   decstr.U64(1),
			}},
			PayloadSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return raw
}

func schemaEnroll() []byte {
	return []byte(`{"enrollment_request_id":"` + uuid.NewString() + `","node_incarnation_id":"` + uuid.NewString() + `","csr_pem":"-----BEGIN CERTIFICATE REQUEST-----\nMIIB\n","proof":"AAAAAAA=","platform":{"os":"ubuntu","os_version":"24.04","arch":"amd64"},"agent_release":"1.0.0"}`)
}

func schemaRenew(p auth.Principal) []byte {
	return []byte(`{"renewal_request_id":"` + uuid.NewString() + `","node_id":"` + p.NodeID.String() + `","node_incarnation_id":"` + p.IncarnationID.String() + `","csr_pem":"-----BEGIN CERTIFICATE REQUEST-----\nMIIB\n"}`)
}

// TestClassBodyCapAgreesWithJSONCheck posts a schema-valid document padded
// with whitespace to one byte under the class cap, the cap, and one byte over
// it. Within the cap the request passes size, JSON, and schema checks.
// Past the cap the response is 413.
func TestClassBodyCapAgreesWithJSONCheck(t *testing.T) {
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	admin := principal(auth.Admin)
	agentP := principal(auth.Agent)
	cases := []struct {
		name   string
		agent  bool
		p      auth.Principal
		method string
		path   string
		limit  int64
		body   []byte
		tls    bool
	}{
		{"browser", false, admin, http.MethodPost, "/api/v1/admin/groups", c.BrowserBodyBytes, []byte(`{"code":"standard","name":"Standard"}`), false},
		{"agent", true, agentP, http.MethodPost, "/api/v1/agent/certificates/renew", c.AgentBodyBytes, schemaRenew(agentP), true},
		{"usage", true, agentP, http.MethodPost, "/api/v1/agent/usage-batches", c.UsageBodyBytes, schemaUsageBatch(agentP), true},
		{"webhook", false, auth.Principal{}, http.MethodPost, "/api/v1/webhooks/example", c.WebhookBodyBytes, []byte(`{}`), false},
		{"anonymous", false, auth.Principal{}, http.MethodPost, "/api/v1/agent/enroll", c.AgentBodyBytes, schemaEnroll(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := fixtureConfig(t, tc.p, tc.agent, nil, c)
			for _, point := range []struct {
				name string
				n    int64
				want int
			}{
				{"cap-1", tc.limit - 1, http.StatusNotImplemented},
				{"cap", tc.limit, http.StatusNotImplemented},
				{"cap+1", tc.limit + 1, http.StatusRequestEntityTooLarge},
			} {
				t.Run(point.name, func(t *testing.T) {
					body := padJSONObject(t, tc.body, int(point.n))
					r := request(tc.method, tc.path, string(body), tc.p)
					if tc.tls {
						r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
					}
					w := assertStatus(t, h, r, point.want)
					if point.want == http.StatusNotImplemented && bytes.Contains(w.Body.Bytes(), []byte("Invalid JSON document.")) {
						t.Fatalf("body at %d bytes was rejected as malformed JSON", point.n)
					}
				})
			}
		})
	}
}

func TestUsageBodyAboveCanonBoundRejectsTrailingValue(t *testing.T) {
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	p := principal(auth.Agent)
	if int64(len(schemaUsageBatch(p))) > canonjson.MaxBytes {
		t.Fatal("fixture larger than the canonical bound")
	}
	body := padJSONObject(t, schemaUsageBatch(p), canonjson.MaxBytes+32)
	body = append(body, []byte(`{}`)...)
	if int64(len(body)) > c.UsageBodyBytes {
		t.Fatal("trailing fixture exceeds the usage cap")
	}
	h := fixtureConfig(t, p, true, nil, c)
	r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(body), p)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	w := assertStatus(t, h, r, http.StatusBadRequest)
	if !bytes.Contains(w.Body.Bytes(), []byte("Invalid JSON document.")) {
		t.Fatalf("body=%s", w.Body.String())
	}
}
