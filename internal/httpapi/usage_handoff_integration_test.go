//go:build integration

package httpapi_test

import (
	"bytes"
	"context"
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
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

type usageServiceBoundary struct {
	service *usage.Service
	calls   int
}

func (h *usageServiceBoundary) SubmitUsageBatch(ctx context.Context, r gen.SubmitUsageBatchRequestObject) (gen.SubmitUsageBatchResponseObject, error) {
	h.calls++
	p, _ := auth.FromContext(ctx)
	res, err := h.service.IngestBatch(ctx, usage.NodeIdentity{NodeID: p.NodeID, NodeIncarnationID: p.IncarnationID}, *r.Body)
	return gen.SubmitUsageBatch200JSONResponse{Body: res.UsageBatchResult}, err
}

func TestMTR03_D115UsageForeignIdentityRejectedAndAudited(t *testing.T) {
	pool := dbtest.New(t)
	svc, err := usage.NewService(usage.Config{Pool: pool})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../contracts/examples/valid/UsageBatchRequest.handoff-accepted.json")
	if err != nil {
		t.Fatal(err)
	}
	var body api.UsageBatchRequest
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	presenter := auth.Principal{Role: auth.Agent, NodeID: body.NodeId, IncarnationID: body.NodeIncarnationId}
	foreign := uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO nodes(id,lifecycle_state) VALUES($1,'registered'),($2,'registered')`, presenter.NodeID, foreign)
	exec(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch)
		VALUES($1,$2,decode(repeat('11',32),'hex'),'ubuntu','amd64')`, presenter.IncarnationID, presenter.NodeID)
	exec(`UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, presenter.NodeID, presenter.IncarnationID)
	exec(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch,revoked_at)
		VALUES($1,$2,decode(repeat('22',32),'hex'),'ubuntu','arm64','2026-10-28T14:00:00Z')`, *body.OriginIncarnationId, foreign)
	boundary := &usageServiceBoundary{service: svc}
	composite, err := httpapi.Compose(boundary)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := httpapi.New(composite, httpapi.Options{
		Config: config.Defaults(), AgentListener: true,
		Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) { return presenter, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	handler = httpapi.Transport(nil, handler)
	for _, large := range []bool{false, true} {
		for _, foreignBatch := range []bool{false, true} {
			req := body
			status, code, reason := http.StatusConflict, "STATE_CONFLICT", "origin_ineligible"
			if foreignBatch {
				req.NodeId = foreign
				status, code, reason = http.StatusForbidden, "INCARNATION_MISMATCH", "batch_identity"
			}
			encoded, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			if large {
				encoded = append(encoded, bytes.Repeat([]byte(" "), canonjson.MaxBytes)...)
			}
			r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", bytes.NewReader(encoded))
			r.Header.Set("Content-Type", "application/json")
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w := httptest.NewRecorder()
			before := boundary.calls
			handler.ServeHTTP(w, r)
			if w.Code != status || !strings.Contains(w.Body.String(), code) || boundary.calls != before+1 {
				t.Fatalf("large=%v foreignBatch=%v: status=%d calls=%d body=%s", large, foreignBatch, w.Code, boundary.calls, w.Body.String())
			}
			var audits int
			if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='usage.ingest_rejected'
				AND actor=$1 AND resource=$2 AND request_id=$3 AND after_diff->>'error_code'=$4 AND after_diff->>'reason'=$5`,
				presenter.NodeID.String(), "node:"+presenter.NodeID.String(), w.Header().Get("X-Request-ID"), code, reason).Scan(&audits); err != nil {
				t.Fatal(err)
			}
			if audits != 1 {
				t.Fatalf("foreign identity rejection was not durably audited: %d", audits)
			}
		}
	}
	var mutations int
	if err := pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM usage_events)+(SELECT count(*) FROM usage_checkpoints)`).Scan(&mutations); err != nil || mutations != 0 {
		t.Fatalf("foreign identity mutated ledger/checkpoint: %d (%v)", mutations, err)
	}
}
