//go:build integration

package httpwire_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TEST ONLY management API; production bridge uses the real Client method set,
// stable bindings and independent read-backs, never the exits fake.
func TestAT08_PublicationWorkerRealAdapterBridge(t *testing.T) {
	f, enrolled, cert, _, _ := publicationFixture(t)
	var mu sync.Mutex
	objects := map[string]map[string]map[string]any{}
	for _, kind := range []string{"peers", "groups", "routes", "policies", "dns/nameservers"} {
		objects[kind] = map[string]map[string]any{}
	}
	objects["peers"]["TEST-peer"] = map[string]any{"id": "TEST-peer", "name": "TEST ONLY node", "ip": "100.64.1.2", "user_id": "", "ephemeral": false}
	serial, routeCreates, routeReads := 0, 0, 0
	lostCreateReply := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") != "Token TEST ONLY adapter token" {
			t.Error("missing adapter credential")
			w.WriteHeader(401)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/")
		kind, id, _ := strings.Cut(path, "/")
		if path == "dns/nameservers" {
			kind, id = path, ""
		}
		bucket, ok := objects[kind]
		if !ok {
			t.Errorf("unexpected adapter resource %s", kind)
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		respond := func(v any) {
			if err := json.NewEncoder(w).Encode(v); err != nil {
				t.Error(err)
			}
		}
		switch r.Method {
		case "GET":
			if kind == "routes" {
				routeReads++
			}
			if id != "" {
				if value, ok := bucket[id]; ok {
					respond(value)
				} else {
					w.WriteHeader(404)
				}
				return
			}
			list := []any{}
			for _, value := range bucket {
				list = append(list, value)
			}
			respond(list)
		case "DELETE":
			delete(bucket, id)
			respond(map[string]any{})
		case "POST", "PUT":
			var value map[string]any
			if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if r.Method == "POST" {
				serial++
				id = fmt.Sprintf("TEST-object-%d", serial)
				if kind == "routes" {
					routeCreates++
				}
			}
			value["id"] = id
			if kind == "groups" {
				peers := []any{}
				for _, peer := range value["peers"].([]any) {
					peers = append(peers, map[string]any{"id": peer})
				}
				value["peers"] = peers
			}
			bucket[id] = value
			if kind == "routes" && r.Method == "POST" && lostCreateReply {
				lostCreateReply = false
				// Remote creation committed, but the caller cannot confirm it.
				w.WriteHeader(http.StatusServiceUnavailable)
				respond(map[string]string{"message": "TEST ONLY uncertain write"})
				return
			}
			respond(value)
		default:
			t.Errorf("unexpected adapter method %s", r.Method)
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	f.r.Config.NetBirdDeployment = "TEST ONLY publication deployment"
	f.sql(`UPDATE node_incarnations SET deployment_id=$2,peer_id='TEST-peer' WHERE id=$1`, enrolled.NodeIncarnationId, f.r.Config.NetBirdDeployment)
	nb, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: f.r.Config.NetBirdDeployment, Token: testToken{}, Bindings: nodes.NetBirdBindings{Pool: f.r.Pool}, AllowInsecureHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	bridge := httpwire.NewPublicationAdapter(f.r, nb)
	var exitID uuid.UUID
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT id FROM exits WHERE node_id=$1`, enrolled.NodeId).Scan(&exitID); err != nil {
		t.Fatal(err)
	}
	if _, err = bridge.ReadExitState(t.Context(), "TEST ONLY wrong deployment", exitID, ""); err == nil {
		t.Fatal("deployment mismatch accepted")
	}
	op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
	f.acknowledgePublication(enrolled, cert, true)
	f.sql(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, desiredstate.EventNodePublicationChanged)
	d, err := worker.New(f.r.Pool, worker.NodeJobs(f.r, nb, nil, 3), worker.Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Millisecond}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.publicationDispatch(d, op.OperationId)
	if got := f.publicationOperation(op.OperationId); got.Status != api.OperationStatusRunning {
		t.Fatal("uncertain route creation was acknowledged", got)
	}
	// Retry discovers the already-created stable route, even before a binding
	// or external reference could be persisted from the lost create response.
	f.publicationDispatch(d, op.OperationId)
	if got := f.publicationOperation(op.OperationId); got.Status != api.OperationStatusSucceeded {
		t.Fatal(got)
	}
	mu.Lock()
	if routeCreates != 1 || routeReads < 2 || len(objects["routes"]) != 1 {
		t.Error("missing independent route read-back", routeCreates, routeReads)
	}
	var routeID string
	for id, route := range objects["routes"] {
		routeID = id
		if route["peer"] != "TEST-peer" || route["masquerade"] != false || route["skip_auto_apply"] != true {
			t.Error("unsafe route", route)
		}
	}
	objects["routes"][routeID]["masquerade"] = true
	mu.Unlock()
	if _, err = bridge.ReadExitState(t.Context(), f.r.Config.NetBirdDeployment, exitID, routeID); err == nil {
		t.Fatal("unsafe read-back accepted")
	}
	mu.Lock()
	objects["routes"][routeID]["masquerade"] = false
	mu.Unlock()
	// Central withdrawal works while the node is offline, with route/policy
	// deletion and 404 read-backs, without claiming an agent ACK.
	withdraw := f.publicationCommand(enrolled.NodeId, api.PublicationIntentDisabled)
	f.sql(`UPDATE nodes SET control_status='offline',resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId)
	f.publicationDispatch(d, withdraw.OperationId)
	if got := f.publicationOperation(withdraw.OperationId); got.Status != api.OperationStatusPartiallySucceeded {
		t.Fatal(got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(objects["routes"]) != 0 || len(objects["policies"]) != 0 || len(objects["peers"]) != 1 {
		t.Fatal("route/access policy withdrawal failed or identity deleted")
	}
}
