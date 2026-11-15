//go:build integration

package httpwire_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func TestReconcileNetBird_MalformedPaidBinding(t *testing.T) {
	f := setup(t)
	deployment := "TEST ONLY publication deployment"
	f.r.Config.NetBirdDeployment = deployment
	bad := uuid.MustParse("00000000-0000-4000-8000-000000000071")
	good := uuid.MustParse("00000000-0000-4000-8000-000000000072")
	seed := func(exitID uuid.UUID, peer string) uuid.UUID {
		t.Helper()
		nodeID, incID := uuid.New(), uuid.New()
		fp := sha256.Sum256([]byte(exitID.String()))
		f.sql(`INSERT INTO nodes(id,lifecycle_state,provisioning_state) VALUES($1,'registered','ready')`, nodeID)
		f.sql(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,arch) VALUES($1,$2,$3,$4,$5,'ubuntu','amd64')`, incID, nodeID, fp[:], deployment, peer)
		f.sql(`UPDATE nodes SET current_incarnation_id=$2, resource_version=resource_version+1 WHERE id=$1`, nodeID, incID)
		f.sql(`INSERT INTO exits(id,node_id,display_name,region,publication_intent) VALUES($1,$2,'TEST ONLY exit','TEST ONLY','published')`, exitID, nodeID)
		return nodeID
	}
	badNode := seed(bad, "TEST-peer-bad")
	goodNode := seed(good, "TEST-peer-good")
	if err := (nodes.NetBirdBindings{Pool: f.r.Pool}).Put(t.Context(), netbird.Binding{
		DeploymentID: deployment, Kind: "groups", InternalID: "paid:not-a-uuid",
		ExternalID: "TEST-bad-group", DesiredHash: strings.Repeat("ab", 32), LastObservedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("cd", 32)
	f.sql(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,sync_state) VALUES('exit',$1,'netbird',$2,$3,decode($4,'hex'),'pending')`, bad, deployment, "TEST-route-bad", hash)
	f.sql(`INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,sync_state) VALUES('exit',$1,'netbird',$2,$3,decode($4,'hex'),'pending')`, good, deployment, "TEST-route-good", hash)
	sum := sha256.Sum256([]byte(bad.String()))
	route := map[string]any{
		"id": "TEST-route-bad", "network_id": "fvpn-exit-" + hex.EncodeToString(sum[:12]), "network": "0.0.0.0/0",
		"peer": "TEST-peer-bad", "groups": []string{"TEST-bad-group"}, "enabled": true,
		"masquerade": false, "skip_auto_apply": true,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/")
		var body any
		switch path {
		case "routes":
			body = []any{route}
		case "peers", "groups", "policies", "dns/nameservers":
			body = []any{}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := netbird.New(netbird.Config{
		BaseURL: server.URL + "/api", DeploymentID: deployment, Token: testToken{},
		Bindings: nodes.NetBirdBindings{Pool: f.r.Pool}, AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httpwire.NewPublicationAdapter(f.r, client)
	page := []exits.ExitPageRef{{ExitID: bad, ExternalID: "TEST-route-bad"}, {ExitID: good, ExternalID: "TEST-route-good"}}
	results, err := adapter.ReadExitPage(t.Context(), deployment, page)
	if err != nil || len(results) != 2 {
		t.Fatalf("page results=%d err=%v", len(results), err)
	}
	var pageEvidence exits.PublicationEvidenceError
	if !errors.As(results[0].Evidence, &pageEvidence) || string(pageEvidence) != "group_evidence_unavailable" || results[1].Evidence != nil || results[1].State.Exists {
		t.Fatalf("page bad=%v good=%+v", results[0].Evidence, results[1])
	}
	_, readErr := adapter.ReadExitState(t.Context(), deployment, bad, "TEST-route-bad")
	var stateEvidence exits.PublicationEvidenceError
	if !errors.As(readErr, &stateEvidence) || stateEvidence != pageEvidence {
		t.Fatalf("ReadExitState=%v page=%v", readErr, pageEvidence)
	}
	svc := exits.NewService(f.r.Pool, nil, adapter)
	items, err := svc.ReconcileNetBird(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]exits.ReconcileItem{}
	for _, item := range items {
		got[item.ExitID] = item
	}
	if len(got) != 2 {
		t.Fatalf("results=%+v", items)
	}
	badDrift := nodeDrift(t, f, badNode)
	if len(badDrift) != 1 || badDrift[0].Field != "netbird_route_evidence" || fieldValue(t, badDrift[0].Observed) != "group_evidence_unavailable" || fieldValue(t, badDrift[0].Desired) != "published" {
		t.Fatalf("bad drift=%+v", badDrift)
	}
	goodDrift := nodeDrift(t, f, goodNode)
	if len(goodDrift) != 1 || goodDrift[0].Field != "netbird_route" || fieldValue(t, goodDrift[0].Desired) != "published" || fieldValue(t, goodDrift[0].Observed) != "withdrawn" {
		t.Fatalf("good drift=%+v", goodDrift)
	}
	if len(got[bad].Drift) != 1 || got[bad].Drift[0].Field != "netbird_route_evidence" || len(got[good].Drift) != 1 || got[good].Drift[0].Field != "netbird_route" {
		t.Fatalf("items bad=%+v good=%+v", got[bad], got[good])
	}
	var badSync, badErr, goodSync, goodErr string
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT sync_state, coalesce(last_error,'') FROM external_bindings WHERE resource_type='exit' AND resource_id=$1`, bad).Scan(&badSync, &badErr); err != nil || badSync != "error" || badErr != "evidence: group_evidence_unavailable" {
		t.Fatalf("bad binding sync=%s err=%s scan=%v", badSync, badErr, err)
	}
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT sync_state, coalesce(last_error,'') FROM external_bindings WHERE resource_type='exit' AND resource_id=$1`, good).Scan(&goodSync, &goodErr); err != nil || goodSync != "drifted" || goodErr != "drift on netbird_route" {
		t.Fatalf("good binding sync=%s err=%s scan=%v", goodSync, goodErr, err)
	}
}

func nodeDrift(t *testing.T, f *fixture, nodeID uuid.UUID) []api.StateDrift {
	t.Helper()
	var raw []byte
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT drift FROM nodes WHERE id=$1`, nodeID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var drift []api.StateDrift
	if err := json.Unmarshal(raw, &drift); err != nil {
		t.Fatal(err)
	}
	return drift
}

func fieldValue(t *testing.T, value presence.Nullable[string]) string {
	t.Helper()
	got, err := value.Get()
	if err != nil {
		t.Fatal(err)
	}
	return got
}
