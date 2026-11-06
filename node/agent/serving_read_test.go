package agent

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestD164_ServingReadPreservesEveryCausalDetail(t *testing.T) {
	at := time.Date(2026, 11, 6, 1, 2, 3, 123456789, time.UTC)
	for _, source := range []string{"apply", "live"} {
		for _, verified := range []bool{false, true} {
			for _, detail := range []string{"nil", "", "Rate policy could not be applied."} {
				t.Run(fmt.Sprintf("%s/%t/%s", source, verified, detail), func(t *testing.T) {
					e := &Engine{}
					evidence := enforce.ServingEvidence{TCSummary: enforce.TCSummary{Generation: decstr.U64(^uint64(0)), Phase: "verify"}}
					e.retainServingRead(evidence, ^uint64(0), at, source, verified)
					causal := detail
					qos := api.NodeCheck{Name: api.NodeCheckNameQos, Status: api.HealthCheckStatusFailed}
					qos.ErrorCode.Set(api.ErrorCodeVALIDATIONFAILED)
					if detail != "nil" {
						qos.Detail = &causal
					}
					binding := "No member binding was admitted."
					rep := api.ObservationReport{ObservedAt: at.Add(24 * time.Hour), SafetyStatus: api.SafetyStatusFailed, Checks: []api.NodeCheck{qos, {Name: api.NodeCheckNameInternetOnly, Status: api.HealthCheckStatusPassed}, {Name: api.NodeCheckNameSourceBinding, Detail: &binding}}}
					before := slices.Clone(rep.Checks)
					e.attachServingRead(&rep)
					got, err := rep.ServingRead.Get()
					wantText := fmt.Sprintf("read_at=%s source=%s candidate=18446744073709551615 verified=%t %s", at.Format(time.RFC3339Nano), source, verified, evidence.ReportDetail())
					if err != nil || got.ReadAt != at || string(got.Source) != source || uint64(got.CandidateGeneration) != ^uint64(0) || got.Verified != verified || got.Diagnostic != wantText {
						t.Fatal("retained diagnostic changed", got, err)
					}
					if !rep.TcEvidence.IsNull() || rep.MemberForwardingEnabled || rep.SafetyStatus != api.SafetyStatusFailed {
						t.Fatal("diagnostic fabricated TC or serving state")
					}
					if detail == "nil" {
						before[0].Detail = &wantText
					}
					if !reflect.DeepEqual(before, rep.Checks) {
						t.Fatal("causal/non-QoS detail changed", rep.Checks)
					}
				})
			}
		}
	}
	// Attachment is independent of whether any QoS or other check exists.
	e := &Engine{}
	e.retainServingRead(enforce.ServingEvidence{}, 9007199254740993, at, "apply", false)
	rep := api.ObservationReport{}
	e.attachServingRead(&rep)
	got, err := rep.ServingRead.Get()
	if err != nil || got.CandidateGeneration.String() != "9007199254740993" || len(rep.Checks) != 0 {
		t.Fatal("no-check report lost diagnostic", err)
	}
	e.servingRead = nil
	e.attachServingRead(&rep)
	if !rep.ServingRead.IsSpecified() || !rep.ServingRead.IsNull() {
		t.Fatal("no retained read must be explicit null")
	}
}

func TestD164_ReportRetainsReadTimeClippingAndDecision(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	at := r.clock.Wall().Add(-24 * time.Hour)
	// TEST ONLY long opaque diagnostic; clipping need not leave valid JSON.
	evidence := enforce.ServingEvidence{TCSummary: enforce.TCSummary{Generation: decstr.U64(^uint64(0)), Phase: strings.Repeat("x", 300), SHA256: strings.Repeat("a", 64)}}
	r.eng.retainServingRead(evidence, ^uint64(0), at, "live", false)
	decision, err := json.Marshal(r.eng.decision)
	if err != nil {
		t.Fatal(err)
	}
	state := r.eng.observationState()
	rep := mustReport(t, r, t.Context())
	requireObservationSchema(t, rep)
	read, err := rep.ServingRead.Get()
	if err != nil {
		t.Fatal(err)
	}
	full := fmt.Sprintf("read_at=%s source=live candidate=18446744073709551615 verified=false %s", at.Format(time.RFC3339Nano), evidence.ReportDetail())
	if len(full) <= 500 || len(read.Diagnostic) != 500 || read.Diagnostic != full[:500] || !read.ReadAt.Equal(at) || read.Verified {
		t.Fatal("clipping/time/verdict changed", len(full), read)
	}
	after, err := json.Marshal(r.eng.decision)
	if err != nil || !bytes.Equal(decision, after) {
		t.Fatal("report mutated decision", err)
	}
	r.clock.Advance(time.Minute)
	next := mustReport(t, r, t.Context())
	value, err := next.ServingRead.Get()
	if err != nil || value != read || !next.ObservedAt.After(rep.ObservedAt) {
		t.Fatal("report refreshed retained read", err)
	}
	// The already-established coalescing semantics exclude timing/diagnostic
	// changes, but still retain generation and outer-verdict transitions.
	evidence.TCSummary.Phase = "verify"
	r.eng.retainServingRead(evidence, ^uint64(0), r.clock.Wall(), "live", false)
	if r.eng.observationState() != state {
		t.Fatal("diagnostic time/text changed coalescing")
	}
	r.eng.retainServingRead(evidence, ^uint64(0), r.clock.Wall(), "live", true)
	if r.eng.observationState() == state {
		t.Fatal("outer verdict transition disappeared")
	}
	r.eng.retainServingRead(evidence, 9007199254740993, r.clock.Wall(), "live", false)
	if r.eng.observationState() == state {
		t.Fatal("candidate transition disappeared")
	}
}

func TestD164_QueuedObservationPresenceSurvivesRealFlush(t *testing.T) {
	for _, mode := range []string{"absent", "null", "value"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			ctx := t.Context()
			pending, err := r.eng.store.Pending(ctx)
			if err != nil || len(pending) != 0 {
				t.Fatal("unexpected initial outbox", err)
			}
			fixture, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.json")
			if err != nil {
				t.Fatal(err)
			}
			var report api.ObservationReport
			if err = json.Unmarshal(fixture, &report); err != nil {
				t.Fatal(err)
			}
			if report.ServingRead.IsSpecified() {
				t.Fatal("legacy fixture no longer proves absence")
			}
			report.ReportId = uuid.New()
			report.NodeId = r.eng.nodeID
			report.NodeIncarnationId = r.eng.incID
			switch mode {
			case "null":
				report.ServingRead.SetNull()
			case "value":
				report.ServingRead.Set(api.ServingReadEvidence{ReadAt: r.clock.Wall().Add(-time.Hour), Source: api.ServingReadEvidenceSourceApply, CandidateGeneration: decstr.U64(^uint64(0)), Verified: false, Diagnostic: "TEST ONLY queued rejected read {"})
			}
			requireObservationSchema(t, report)
			original, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			canonical, err := canonjson.Canonicalize(original)
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(canonical)
			if err = r.eng.store.Enqueue(ctx, report.ReportId, original, r.clock.Wall()); err != nil {
				t.Fatal(err)
			}
			// Read the actual retained SQLite row, including after MarkAccepted removes
			// it from Pending. There is no stored hash column; hashes here are computed.
			database, err := sql.Open("sqlite", r.eng.cfg.DBPath())
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			stored := func(wantStatus string) {
				t.Helper()
				var raw []byte
				var status string
				if err := database.QueryRowContext(ctx, `SELECT payload,status FROM outbox WHERE report_id=?`, report.ReportId.String()).Scan(&raw, &status); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(raw, original) || status != wantStatus {
					t.Fatal("queued bytes/status changed", status)
				}
			}
			var mu sync.Mutex
			var bodies [][]byte
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				raw, err := io.ReadAll(request.Body)
				if err != nil {
					t.Error(err)
					w.WriteHeader(500)
					return
				}
				if request.URL.Path != "/api/v1/agent/observations" || request.Header.Get("Idempotency-Key") != report.ReportId.String() {
					t.Error("wrong observation request identity")
				}
				mu.Lock()
				bodies = append(bodies, raw)
				attempt := len(bodies)
				mu.Unlock()
				if attempt == 1 {
					apierr.Write(w, request, apierr.New(apierr.RateLimited, "TEST ONLY retry").WithRetryAfter(1))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				if err := json.NewEncoder(w).Encode(api.ObservationAccepted{Accepted: true, ReportId: report.ReportId, ReceivedAt: time.Now().UTC()}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			client, err := controlclient.New(controlclient.Config{BaseURL: server.URL, RootCAs: roots})
			if err != nil {
				t.Fatal(err)
			}
			previous := r.eng.client
			r.eng.client = client
			defer func() { client.Close(); r.eng.client = previous }()
			r.eng.retainServingRead(enforce.ServingEvidence{}, 9007199254740993, r.clock.Wall(), "live", true)
			if err = r.eng.flush(ctx); err != nil {
				t.Fatal(err)
			}
			stored(agentstore.OutboxPending)
			pending, err = r.eng.store.Pending(ctx)
			if err != nil || len(pending) != 1 || pending[0].Attempts == 0 || !bytes.Equal(pending[0].Payload, original) {
				t.Fatal("failed attempt changed durable body", err)
			}
			// Preserve the existing retry slot; changed current evidence must not cause
			// a new attachment or bypass backoff on this already queued report.
			r.eng.retainServingRead(enforce.ServingEvidence{}, 7, r.clock.Wall().Add(time.Second), "apply", false)
			if err = r.eng.flush(ctx); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			attempts := len(bodies)
			mu.Unlock()
			if attempts != 1 {
				t.Fatal("retry slot bypassed")
			}
			r.clock.Advance(r.eng.observationRetryAt - r.clock.Mono())
			if err = r.eng.flush(ctx); err != nil {
				t.Fatal(err)
			}
			stored(agentstore.OutboxAccepted)
			pending, err = r.eng.store.Pending(ctx)
			if err != nil || len(pending) != 0 {
				t.Fatal("accepted row remained pending", err)
			}
			mu.Lock()
			received := append([][]byte(nil), bodies...)
			mu.Unlock()
			if len(received) != 2 {
				t.Fatal("wrong HTTP attempt count", len(received))
			}
			for _, raw := range received {
				got, err := canonjson.Canonicalize(raw)
				if err != nil || !bytes.Equal(got, canonical) || sha256.Sum256(got) != hash {
					t.Fatal("retry changed canonical body/hash", err)
				}
				var fields map[string]json.RawMessage
				if err = json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				value, present := fields["serving_read"]
				if (mode == "absent" && present) || (mode == "null" && (!present || string(value) != "null")) || (mode == "value" && (!present || string(value) == "null")) {
					t.Fatal("queued presence changed", mode, string(raw))
				}
			}
		})
	}
}
