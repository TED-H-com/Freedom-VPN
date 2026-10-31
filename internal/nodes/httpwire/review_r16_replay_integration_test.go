//go:build integration

package httpwire_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestReviewR16_InstallerResultReplayReturnsStoredAck submits one installer
// result through the agent HTTP path and repeats the idempotency key. The
// stored ack lives in installer_update_results. Replay still reads
// update_results. wire-nodes follow-up 29 owns the fix.
func TestReviewR16_InstallerResultReplayReturnsStoredAck(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	fleet, err := fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	f.r.Fleet = fleet
	f.r.Nodes = nodes.NewService(f.r.Pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, r api.ObservationReport) error {
		return fleet.ObserveReportTx(ctx, tx, fleetupdate.Identity{NodeID: r.NodeId, IncarnationID: r.NodeIncarnationId}, r)
	}))
	var registration api.ReleaseRegisterRequest
	if err = json.Unmarshal(rf.Request(t, rf.Manifest), &registration); err != nil {
		t.Fatal(err)
	}
	decode[api.Release](t, f.call("POST", "/api/v1/admin/releases", registration, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 201)
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	if err = db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return f.r.Releases.ApplySupportTx(ctx, tx, f.p.ID, support, "TEST ONLY support")
	}); err != nil {
		t.Fatal(err)
	}
	_, enrolled, cert := f.enroll(f.create(nil))
	raw, err := os.ReadFile("../../../contracts/examples/valid/UpdateResult.installer.json")
	if err != nil {
		t.Fatal(err)
	}
	var result api.UpdateResult
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	result.ResultId = uuid.New()
	result.NodeId = enrolled.NodeId
	result.IncarnationId = enrolled.NodeIncarnationId
	result.JobId = uuid.New()
	result.RequestedReleaseId = rf.Manifest.ReleaseId
	result.CurrentReleaseId.Set(rf.Manifest.ReleaseId)
	result.PreviousReleaseId.SetNull()
	if err = f.r.Pool.QueryRow(ctx, `SELECT encode(manifest_hash,'hex') FROM releases WHERE id=$1`, rf.Manifest.ReleaseId).Scan(&result.ManifestSha256); err != nil {
		t.Fatal(err)
	}
	result.Generation = decstr.U64(0)
	result.Attempt = 1
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	result.PayloadHash, err = canonjson.HashHex(body, "payload_hash")
	if err != nil {
		t.Fatal(err)
	}
	key := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call("POST", "/api/v1/agent/update-results", result, cert, key)
	if first.Code != 202 {
		t.Fatalf("installer submit %d %s", first.Code, first.Body.String())
	}
	second := f.call("POST", "/api/v1/agent/update-results", result, cert, key)
	if second.Code != 202 || first.Body.String() != second.Body.String() {
		t.Fatalf("TODO(review): keyed replay of an installer result did not return the stored ack: %d %s", second.Code, second.Body.String())
	}
}
