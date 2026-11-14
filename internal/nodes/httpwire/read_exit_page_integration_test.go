//go:build integration

package httpwire_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/google/uuid"
)

func TestReadExitPage_MatchesReadExitState(t *testing.T) {
	f := setup(t)
	deployment := "TEST ONLY publication deployment"
	f.r.Config.NetBirdDeployment = deployment
	present, absent, ambiguous := uuid.New(), uuid.New(), uuid.New()
	groupID := uuid.New()
	f.sql(`INSERT INTO member_groups(id,code,name) VALUES($1,$2,$3)`, groupID, "grp-"+groupID.String()[:8], "TEST ONLY group "+groupID.String()[:8])
	seedReadExit := func(exitID uuid.UUID, peer string) {
		t.Helper()
		nodeID, incID := uuid.New(), uuid.New()
		fp := sha256.Sum256([]byte(exitID.String()))
		f.sql(`INSERT INTO nodes(id,lifecycle_state,provisioning_state) VALUES($1,'registered','ready')`, nodeID)
		f.sql(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,arch) VALUES($1,$2,$3,$4,$5,'ubuntu','amd64')`, incID, nodeID, fp[:], deployment, peer)
		f.sql(`UPDATE nodes SET current_incarnation_id=$2, resource_version=resource_version+1 WHERE id=$1`, nodeID, incID)
		f.sql(`INSERT INTO exits(id,node_id,display_name,region) VALUES($1,$2,'TEST ONLY exit','TEST ONLY')`, exitID, nodeID)
	}
	seedReadExit(present, "TEST-peer")
	seedReadExit(absent, "TEST-peer-absent")
	seedReadExit(ambiguous, "TEST-peer-ambiguous")
	f.sql(`INSERT INTO exit_groups(exit_id,group_id) VALUES($1,$2)`, present, groupID)
	if err := (nodes.NetBirdBindings{Pool: f.r.Pool}).Put(t.Context(), netbird.Binding{
		DeploymentID: deployment, Kind: "groups", InternalID: "paid:" + groupID.String(),
		ExternalID: "TEST-group-1", DesiredHash: strings.Repeat("ab", 32), LastObservedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	stable := func(exitID uuid.UUID) string {
		sum := sha256.Sum256([]byte(exitID.String()))
		return "fvpn-exit-" + hex.EncodeToString(sum[:12])
	}
	presentRoute := map[string]any{
		"id": "TEST-route-present", "network_id": stable(present), "network": "0.0.0.0/0",
		"peer": "TEST-peer", "groups": []string{"TEST-group-1"}, "enabled": true,
		"masquerade": false, "skip_auto_apply": true,
	}
	ambiguousName := stable(ambiguous)
	routes := []any{
		presentRoute,
		map[string]any{"id": "TEST-route-ambiguous-1", "network_id": ambiguousName, "network": "0.0.0.0/0", "peer": "TEST-peer-ambiguous", "groups": []string{}, "enabled": true, "masquerade": false, "skip_auto_apply": true},
		map[string]any{"id": "TEST-route-ambiguous-2", "network_id": ambiguousName, "network": "0.0.0.0/0", "peer": "TEST-peer-ambiguous", "groups": []string{}, "enabled": true, "masquerade": false, "skip_auto_apply": true},
	}
	var mu sync.Mutex
	var got []string
	requests := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = append(got, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/")
		var body any
		switch path {
		case "routes":
			body = routes
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
	client, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: deployment, Token: testToken{}, Bindings: nodes.NetBirdBindings{Pool: f.r.Pool}, AllowInsecureHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	adapter := httpwire.NewPublicationAdapter(f.r, client)
	page := []exits.ExitPageRef{{ExitID: present}, {ExitID: absent}, {ExitID: ambiguous, ExternalID: "TEST-route-ambiguous-1"}}
	before := requests()
	if _, err = adapter.ReadExitPage(t.Context(), "TEST ONLY other deployment", page); !errors.Is(err, exits.ErrDeploymentNotServed) || len(requests()) != len(before) {
		t.Fatalf("unserved requests=%d err=%v", len(requests())-len(before), err)
	}
	before = requests()
	if _, err = client.ReadObservedState(t.Context()); err != nil {
		t.Fatal(err)
	}
	one := requests()[len(before):]
	before = requests()
	results, err := adapter.ReadExitPage(t.Context(), deployment, page)
	if err != nil || len(results) != len(page) {
		t.Fatalf("page results=%d err=%v", len(results), err)
	}
	if pageReqs := requests()[len(before):]; !slices.Equal(pageReqs, one) {
		t.Fatalf("page requests %v, one ReadObservedState %v", pageReqs, one)
	}
	for i, item := range page {
		state, readErr := adapter.ReadExitState(t.Context(), deployment, item.ExitID, item.ExternalID)
		if readErr != nil {
			var want, gotEvidence exits.PublicationEvidenceError
			if !errors.As(readErr, &want) || !errors.As(results[i].Evidence, &gotEvidence) || want != gotEvidence || !reflect.DeepEqual(results[i].State, exits.ObservedExitState{}) {
				t.Fatalf("exit %s read=%v page=%+v", item.ExitID, readErr, results[i])
			}
			continue
		}
		if results[i].Evidence != nil || !reflect.DeepEqual(state, results[i].State) {
			t.Fatalf("exit %s state=%+v page=%+v", item.ExitID, state, results[i])
		}
	}
	if !results[0].State.Exists || results[0].State.ExternalID != "TEST-route-present" || results[1].State.Exists || results[1].Evidence != nil {
		t.Fatalf("present/absent=%+v %+v", results[0], results[1])
	}
	var ambiguousEvidence exits.PublicationEvidenceError
	if !errors.As(results[2].Evidence, &ambiguousEvidence) || string(ambiguousEvidence) != "route_ambiguous" {
		t.Fatalf("ambiguous=%v", results[2].Evidence)
	}
}
