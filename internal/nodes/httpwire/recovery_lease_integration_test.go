//go:build integration

package httpwire_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestAT29_HTTPRecoveryLeaseReplays(t *testing.T) {
	f, enrolled, cert, member, _ := servingFixture(t)
	req := api.LeaseRenewalRequest{
		RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId,
		CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
	}
	key := map[string]string{"Idempotency-Key": uuid.NewString()}
	path := "/api/v1/agent/lease-renewals"
	normal := decode[api.ServingLeaseEnvelope](t, f.call(http.MethodPost, path, req, cert, key), http.StatusOK)
	if len(normal.Signed.Members) != 1 || normal.Signed.Members[0].MemberId != member {
		t.Fatal("normal lease did not contain the entitled member")
	}
	ledger, err := usage.NewService(usage.Config{Pool: f.r.Pool})
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.FreezeIngest(t.Context(), "TEST ONLY actor", "TEST ONLY restore"); err != nil {
		t.Fatal(err)
	}
	for _, headers := range []map[string]string{key, {"Idempotency-Key": uuid.NewString()}} {
		response := f.call(http.MethodPost, path, req, cert, headers)
		body := decode[api.ErrorEnvelope](t, response, http.StatusServiceUnavailable)
		if body.Error.Code != api.ErrorCodeDEPENDENCYUNAVAILABLE || body.Error.Message != "usage ledger recovery in progress" {
			t.Errorf("grant replay error = %+v", body.Error)
		}
	}
	req.RenewalRequestId = uuid.New()
	emptyKey := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call(http.MethodPost, path, req, cert, emptyKey)
	empty := decode[api.ServingLeaseEnvelope](t, first, http.StatusOK)
	if len(empty.Signed.Members) != 0 {
		t.Fatal("recovery lease contained member grants")
	}
	replay := f.call(http.MethodPost, path, req, cert, emptyKey)
	decode[api.ServingLeaseEnvelope](t, replay, http.StatusOK)
	if !bytes.Equal(first.Body.Bytes(), replay.Body.Bytes()) {
		t.Fatal("grantless keyed replay changed the stored envelope")
	}
	var count int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM serving_leases WHERE node_id=$1`, enrolled.NodeId).Scan(&count); err != nil || count != 2 {
		t.Errorf("lease rows after replays = %d (error %v), want 2", count, err)
	}
}
