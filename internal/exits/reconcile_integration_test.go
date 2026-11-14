//go:build integration

package exits

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type reconcileFixture struct {
	t     *testing.T
	pool  *pgxpool.Pool
	ctx   context.Context
	nb    *FakeNetBirdAdapter
	svc   *Service
	nodes *nodes.Service
}

func newReconcileFixture(t *testing.T) *reconcileFixture {
	t.Helper()
	pool := dbtest.New(t)
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	nb := NewFakeNetBirdAdapter()
	clock := func() time.Time { return now }
	return &reconcileFixture{
		t: t, pool: pool, ctx: t.Context(), nb: nb,
		svc:   NewService(pool, testExitsCompiler(t, pool, clock), nb, WithClock(clock)),
		nodes: nodes.NewService(pool, nodes.WithClock(clock)),
	}
}

func (f *reconcileFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, query, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *reconcileFixture) seedOrdered(exitID uuid.UUID, deployment, intent string) uuid.UUID {
	f.t.Helper()
	nodeID, incID := uuid.New(), uuid.New()
	fp := sha256.Sum256([]byte("fp-" + exitID.String()))
	f.exec(`INSERT INTO nodes(id,lifecycle_state,provisioning_state,control_status,data_health,safety_status) VALUES($1,'registered','ready','online','healthy','passed')`, nodeID)
	f.exec(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,arch) VALUES($1,$2,$3,$4,$5,'ubuntu','amd64')`, incID, nodeID, fp[:], deployment, "peer-"+exitID.String())
	f.exec(`UPDATE nodes SET current_incarnation_id=$2, resource_version=resource_version+1 WHERE id=$1`, nodeID, incID)
	f.exec(`INSERT INTO exits(id,node_id,display_name,region,publication_intent,publication_observed) VALUES($1,$2,'TEST ONLY exit','TEST ONLY',$3,'absent')`, exitID, nodeID, intent)
	return nodeID
}

func (f *reconcileFixture) setDrift(nodeID uuid.UUID, drift []api.StateDrift) {
	f.t.Helper()
	raw, err := json.Marshal(drift)
	if err != nil {
		f.t.Fatal(err)
	}
	f.exec(`UPDATE nodes SET drift=$2::jsonb, resource_version=resource_version+1 WHERE id=$1`, nodeID, raw)
}

func (f *reconcileFixture) nodeState(nodeID uuid.UUID) (string, []api.StateDrift) {
	f.t.Helper()
	var version string
	var raw []byte
	if err := f.pool.QueryRow(f.ctx, `SELECT resource_version::text, drift FROM nodes WHERE id=$1`, nodeID).Scan(&version, &raw); err != nil {
		f.t.Fatal(err)
	}
	var drift []api.StateDrift
	if err := json.Unmarshal(raw, &drift); err != nil {
		f.t.Fatal(err)
	}
	return version, drift
}

func (f *reconcileFixture) intents() map[uuid.UUID]string {
	f.t.Helper()
	rows, err := f.pool.Query(f.ctx, `SELECT id, publication_intent FROM exits`)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var intent string
		if err := rows.Scan(&id, &intent); err != nil {
			f.t.Fatal(err)
		}
		out[id] = intent
	}
	if err := rows.Err(); err != nil {
		f.t.Fatal(err)
	}
	return out
}

func (f *reconcileFixture) sameIntents(before map[uuid.UUID]string) {
	f.t.Helper()
	after := f.intents()
	if len(before) != len(after) {
		f.t.Fatalf("exit count %d want %d", len(after), len(before))
	}
	for id, intent := range before {
		if after[id] != intent {
			f.t.Fatalf("publication_intent for %s changed from %s to %s", id, intent, after[id])
		}
	}
	if calls := f.nb.PublishCalls(); calls != 0 {
		f.t.Fatalf("PublishExit calls = %d", calls)
	}
	if calls := f.nb.WithdrawCalls(); calls != 0 {
		f.t.Fatalf("WithdrawExit calls = %d", calls)
	}
}

func versionPlus(t *testing.T, before, after string, delta uint64) {
	t.Helper()
	old, err := strconv.ParseUint(before, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	next, err := strconv.ParseUint(after, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if next != old+delta {
		t.Fatalf("resource_version %s -> %s, want +%d", before, after, delta)
	}
}

func fieldText(t *testing.T, value presence.Nullable[string]) string {
	t.Helper()
	got, err := value.Get()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestReconcileNetBird_KeepsForeignDrift(t *testing.T) {
	f := newReconcileFixture(t)
	nodeID, _, exitID, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentPublished, "active")
	before := f.intents()
	foreign := []api.StateDrift{
		{Field: "desired_state_restore_observation", Desired: presence.NewNullableWithValue("observation received after 2026-01-01T00:00:00Z"), Observed: presence.NewNullableWithValue("pending fresh observation after restore")},
		{Field: "capacity_report", Desired: presence.NewNullableWithValue("kept-desired"), Observed: presence.NewNullableWithValue("kept-observed")},
	}
	f.setDrift(nodeID, foreign)
	f.nb.InjectRouteState(exitID, ObservedExitState{Exists: true, Enabled: false, ExternalID: "nb-route-" + exitID.String(), PeerID: "peer"})
	items, err := f.svc.ReconcileNetBird(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ExitID != exitID || len(items[0].Drift) != 1 || items[0].Drift[0].Field != driftFieldRoute {
		t.Fatalf("result %+v", items)
	}
	_, drift := f.nodeState(nodeID)
	if len(drift) != 3 {
		t.Fatalf("drift=%+v", drift)
	}
	for i, want := range []string{"desired_state_restore_observation", "capacity_report", driftFieldRoute} {
		if drift[i].Field != want {
			t.Fatalf("drift[%d].field=%s want %s", i, drift[i].Field, want)
		}
	}
	if fieldText(t, drift[0].Desired) != fieldText(t, foreign[0].Desired) || fieldText(t, drift[0].Observed) != fieldText(t, foreign[0].Observed) || fieldText(t, drift[1].Desired) != "kept-desired" || fieldText(t, drift[1].Observed) != "kept-observed" {
		t.Fatalf("foreign drift changed: %+v", drift[:2])
	}
	if fieldText(t, drift[2].Desired) != "published" || fieldText(t, drift[2].Observed) != "withdrawn" {
		t.Fatalf("route drift=%+v", drift[2])
	}
	f.sameIntents(before)
}

func TestReconcileNetBird_ETagStableWithoutChange(t *testing.T) {
	f := newReconcileFixture(t)
	nodeID, _, _, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentUnpublished, "absent")
	before := f.intents()
	if _, err := f.svc.ReconcileNetBird(f.ctx); err != nil {
		t.Fatal(err)
	}
	version, drift := f.nodeState(nodeID)
	_, etag, err := f.nodes.GetNode(f.ctx, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.ReconcileNetBird(f.ctx); err != nil {
		t.Fatal(err)
	}
	again, driftAgain := f.nodeState(nodeID)
	if again != version || len(driftAgain) != len(drift) {
		t.Fatalf("second run changed state version %s -> %s drift %d -> %d", version, again, len(drift), len(driftAgain))
	}
	f.sameIntents(before)
	if _, _, err = f.svc.SetPublication(f.ctx, SetPublicationParams{
		NodeID: nodeID, Principal: "admin@example.com", IdempotencyKey: uuid.New(), IfMatch: etag,
		Request: api.PublicationRequest{PublicationIntent: api.PublicationIntentDisabled, Reason: "TEST ONLY etag still matches"},
	}); err != nil {
		t.Fatalf("SetPublication with pre-run ETag: %v", err)
	}
}

func TestReconcileNetBird_BumpsVersionOnceOnChange(t *testing.T) {
	f := newReconcileFixture(t)
	nodeID, _, exitID, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentPublished, "active")
	beforeIntents := f.intents()
	version, _ := f.nodeState(nodeID)
	f.nb.InjectRouteState(exitID, ObservedExitState{Exists: true, Enabled: false, ExternalID: "nb-route-" + exitID.String()})
	if _, err := f.svc.ReconcileNetBird(f.ctx); err != nil {
		t.Fatal(err)
	}
	after, drift := f.nodeState(nodeID)
	versionPlus(t, version, after, 1)
	if len(drift) != 1 || drift[0].Field != driftFieldRoute {
		t.Fatalf("drift=%+v", drift)
	}
	if _, err := f.svc.ReconcileNetBird(f.ctx); err != nil {
		t.Fatal(err)
	}
	stable, _ := f.nodeState(nodeID)
	if stable != after {
		t.Fatalf("second run bumped %s -> %s", after, stable)
	}
	f.sameIntents(beforeIntents)
}

func TestReconcileNetBird_EvidenceErrorContinues(t *testing.T) {
	f := newReconcileFixture(t)
	badNode, _, badExit, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentPublished, "active")
	goodNode, _, goodExit, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentPublished, "active")
	before := f.intents()
	verified := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	observed := bytes.Repeat([]byte{0xab}, 32)
	f.exec(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,observed_hash,sync_state,last_verified_at) VALUES('exit',$1,'netbird','nb-main',$2,$3,$4,'synced',$5)`, badExit, "nb-bad-"+badExit.String(), bytes.Repeat([]byte{1}, 32), observed, verified)
	f.exec(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,sync_state) VALUES('exit',$1,'netbird','other-deployment',$2,$3,'synced')`, badExit, "nb-decoy-"+badExit.String(), bytes.Repeat([]byte{2}, 32))
	f.exec(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,sync_state) VALUES('exit',$1,'netbird','nb-main',$2,$3,'pending')`, goodExit, "nb-good-"+goodExit.String(), bytes.Repeat([]byte{3}, 32))
	f.nb.SetExitEvidence(badExit, PublicationEvidenceError("route_ambiguous"))
	f.nb.InjectRouteState(goodExit, ObservedExitState{Exists: true, Enabled: false, ExternalID: "nb-good-" + goodExit.String()})
	items, err := f.svc.ReconcileNetBird(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uuid.UUID]ReconcileItem{}
	for _, item := range items {
		seen[item.ExitID] = item
	}
	if len(items) != 2 {
		t.Fatalf("results=%d, want one row per exit: %+v", len(items), items)
	}
	if _, ok := seen[badExit]; !ok {
		t.Fatal("evidence exit missing from result")
	}
	if _, ok := seen[goodExit]; !ok {
		t.Fatal("other exit was not reconciled")
	}
	_, badDrift := f.nodeState(badNode)
	if len(badDrift) != 1 || badDrift[0].Field != driftFieldEvidence || fieldText(t, badDrift[0].Observed) != "route_ambiguous" || fieldText(t, badDrift[0].Desired) != string(api.PublicationIntentPublished) {
		t.Fatalf("evidence drift=%+v", badDrift)
	}
	var syncState, lastError string
	var hash []byte
	var checked time.Time
	if err := f.pool.QueryRow(f.ctx, `SELECT sync_state, coalesce(last_error,''), observed_hash, last_verified_at FROM external_bindings WHERE resource_id=$1 AND deployment_id='nb-main'`, badExit).Scan(&syncState, &lastError, &hash, &checked); err != nil {
		t.Fatal(err)
	}
	if syncState != "error" || lastError != "evidence: route_ambiguous" || !bytes.Equal(hash, observed) || !checked.Equal(verified) {
		t.Fatalf("binding sync=%s err=%s hash_ok=%t verified=%s", syncState, lastError, bytes.Equal(hash, observed), checked)
	}
	var decoyState string
	var decoyErr *string
	if err := f.pool.QueryRow(f.ctx, `SELECT sync_state, last_error FROM external_bindings WHERE resource_id=$1 AND deployment_id='other-deployment'`, badExit).Scan(&decoyState, &decoyErr); err != nil || decoyState != "synced" || decoyErr != nil {
		t.Fatalf("decoy binding changed sync=%s err=%v scan=%v", decoyState, decoyErr, err)
	}
	_, goodDrift := f.nodeState(goodNode)
	if len(goodDrift) != 1 || goodDrift[0].Field != driftFieldRoute {
		t.Fatalf("good drift=%+v", goodDrift)
	}
	var goodSync, goodErr string
	if err := f.pool.QueryRow(f.ctx, `SELECT sync_state, coalesce(last_error,'') FROM external_bindings WHERE resource_id=$1 AND deployment_id='nb-main'`, goodExit).Scan(&goodSync, &goodErr); err != nil || goodSync != "drifted" || goodErr != "drift on netbird_route" {
		t.Fatalf("good binding sync=%s err=%s scan=%v", goodSync, goodErr, err)
	}
	f.sameIntents(before)
}

func TestReconcileNetBird_PageErrorWritesNothing(t *testing.T) {
	f := newReconcileFixture(t)
	first, _, _, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentPublished, "active")
	second, _, _, _, _ := seedExitTestNode(t, f.ctx, f.pool, api.PublicationIntentDisabled, "withdrawn")
	before := f.intents()
	marker := []api.StateDrift{{Field: "capacity_report", Desired: presence.NewNullableWithValue("stay"), Observed: presence.NewNullableWithValue("stay")}}
	f.setDrift(first, marker)
	f.setDrift(second, marker)
	v1, d1 := f.nodeState(first)
	v2, d2 := f.nodeState(second)
	f.nb.SetPageError(errors.New("TEST ONLY page failed"))
	items, err := f.svc.ReconcileNetBird(f.ctx)
	if err == nil || err.Error() != "TEST ONLY page failed" || len(items) != 0 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	a1, ad1 := f.nodeState(first)
	a2, ad2 := f.nodeState(second)
	if a1 != v1 || a2 != v2 || len(ad1) != len(d1) || len(ad2) != len(d2) || ad1[0].Field != "capacity_report" || ad2[0].Field != "capacity_report" {
		t.Fatalf("page written versions %s %s drift %v %v", a1, a2, ad1, ad2)
	}
	f.sameIntents(before)
}

func TestReconcileNetBird_PagesOncePerDeployment(t *testing.T) {
	f := newReconcileFixture(t)
	previous := reconcilePageSize
	reconcilePageSize = 2
	t.Cleanup(func() { reconcilePageSize = previous })
	before := f.intents()
	var exits []uuid.UUID
	for n := 1; n <= 5; n++ {
		exitID := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", n))
		deployment := "dep-a"
		if n%2 == 0 {
			deployment = "dep-b"
		}
		f.seedOrdered(exitID, deployment, string(api.PublicationIntentPublished))
		exits = append(exits, exitID)
	}
	before = f.intents()
	items, err := f.svc.ReconcileNetBird(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]int{}
	for _, item := range items {
		got[item.ExitID]++
		if len(item.Drift) != 1 || item.Drift[0].Field != driftFieldRoute {
			t.Fatalf("exit %s drift=%+v", item.ExitID, item.Drift)
		}
	}
	if len(got) != len(exits) {
		t.Fatalf("reconciled %d want %d", len(got), len(exits))
	}
	for _, id := range exits {
		if got[id] != 1 {
			t.Fatalf("exit %s reconciled %d times", id, got[id])
		}
	}
	calls := f.nb.PageExits()
	if len(calls) != 5 {
		t.Fatalf("page calls=%d want 5: %v", len(calls), calls)
	}
	flat := map[uuid.UUID]int{}
	for _, call := range calls {
		if len(call) != 1 {
			t.Fatalf("call grouped exits of one page incorrectly: %v", calls)
		}
		flat[call[0]]++
	}
	for _, id := range exits {
		if flat[id] != 1 {
			t.Fatalf("exit %s appeared in %d calls", id, flat[id])
		}
	}
	f.sameIntents(before)
}

func TestReconcileNetBird_SkipsUnservedDeployment(t *testing.T) {
	f := newReconcileFixture(t)
	served := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	skipped := uuid.MustParse("00000000-0000-4000-8000-000000000012")
	servedNode := f.seedOrdered(served, "dep-served", string(api.PublicationIntentPublished))
	skippedNode := f.seedOrdered(skipped, "dep-skip", string(api.PublicationIntentPublished))
	before := f.intents()
	marker := []api.StateDrift{{Field: "capacity_report", Desired: presence.NewNullableWithValue("skip"), Observed: presence.NewNullableWithValue("skip")}}
	f.setDrift(skippedNode, marker)
	version, _ := f.nodeState(skippedNode)
	observedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	verified := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	f.exec(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,observed_hash,sync_state,last_verified_at,last_observed_at,last_error) VALUES('exit',$1,'netbird','dep-skip',$2,$3,$4,'synced',$5,$6,$7)`, skipped, "nb-skip-"+skipped.String(), bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 32), verified, observedAt, "TEST ONLY keep")
	var bindingBefore string
	if err := f.pool.QueryRow(f.ctx, `SELECT b::text FROM external_bindings b WHERE resource_type='exit' AND resource_id=$1`, skipped).Scan(&bindingBefore); err != nil {
		t.Fatal(err)
	}
	f.nb.SetUnservedDeployment("dep-skip")
	items, err := f.svc.ReconcileNetBird(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ExitID != served || items[0].NodeID != servedNode {
		t.Fatalf("results=%+v", items)
	}
	after, drift := f.nodeState(skippedNode)
	if after != version || len(drift) != 1 || drift[0].Field != "capacity_report" || fieldText(t, drift[0].Desired) != "skip" {
		t.Fatalf("unserved node written version %s -> %s drift=%+v", version, after, drift)
	}
	for _, call := range f.nb.PageExits() {
		for _, id := range call {
			if id == skipped {
				t.Fatalf("unserved exit was read: %v", f.nb.PageExits())
			}
		}
	}
	if f.nb.PageCalls() != 1 {
		t.Fatalf("page calls=%d", f.nb.PageCalls())
	}
	var bindingAfter string
	if err := f.pool.QueryRow(f.ctx, `SELECT b::text FROM external_bindings b WHERE resource_type='exit' AND resource_id=$1`, skipped).Scan(&bindingAfter); err != nil || bindingAfter != bindingBefore {
		t.Fatalf("skipped binding changed\nbefore %s\nafter  %s err=%v", bindingBefore, bindingAfter, err)
	}
	f.sameIntents(before)
}

func TestReconcileNetBird_GroupsExitsByDeployment(t *testing.T) {
	f := newReconcileFixture(t)
	previous := reconcilePageSize
	reconcilePageSize = 3
	t.Cleanup(func() { reconcilePageSize = previous })
	deps := []string{"dep-a", "dep-a", "dep-b", "dep-a", "dep-b"}
	ids := make([]uuid.UUID, len(deps))
	for i, dep := range deps {
		ids[i] = uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1))
		f.seedOrdered(ids[i], dep, string(api.PublicationIntentPublished))
	}
	if _, err := f.svc.ReconcileNetBird(f.ctx); err != nil {
		t.Fatal(err)
	}
	calls := f.nb.PageExits()
	want := [][]uuid.UUID{{ids[0], ids[1]}, {ids[2]}, {ids[3]}, {ids[4]}}
	if len(calls) != len(want) {
		t.Fatalf("calls=%d want %d: %v", len(calls), len(want), calls)
	}
	for i := range want {
		if len(calls[i]) != len(want[i]) {
			t.Fatalf("call %d = %v want %v", i, calls[i], want[i])
		}
		for j := range want[i] {
			if calls[i][j] != want[i][j] {
				t.Fatalf("call %d = %v want %v", i, calls[i], want[i])
			}
		}
	}
}
