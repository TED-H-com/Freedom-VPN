//go:build integration

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TEST ONLY management service: setup-key use adds the peer to both auto-groups,
// as management does. No node/entitlement/projector rows are written by fixtures.
type discoveryManagement struct {
	mu              sync.Mutex
	groups          map[string]map[string]any
	key             map[string]any
	calls           int
	beforePeerRead  func()
	beforeKeyUpdate func()
	peerDeleted     bool
	// injectPeerOnGroupRead simulates the node enrolling after the setup-key
	// snapshot and before the correlation-group read. Empty leaves GETs unchanged.
	injectPeerOnGroupRead string
}

func (f *discoveryManagement) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	respond := func(v any) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	switch {
	case path == "policies" || path == "routes" || path == "dns/nameservers":
		respond([]any{})
	case path == "groups" && r.Method == "GET":
		groups := []any{}
		for _, g := range f.groups {
			groups = append(groups, g)
		}
		respond(groups)
	case path == "groups" && r.Method == "POST":
		var g map[string]any
		_ = json.NewDecoder(r.Body).Decode(&g)
		id := fmt.Sprintf("TEST-group-%d", len(f.groups)+1)
		g["id"] = id
		f.groups[id] = g
		respond(g)
	case strings.HasPrefix(path, "groups/"):
		id := strings.TrimPrefix(path, "groups/")
		g := f.groups[id]
		if g == nil {
			w.WriteHeader(404)
			return
		}
		if r.Method == "DELETE" {
			delete(f.groups, id)
			respond(map[string]any{})
			return
		}
		if r.Method == "PUT" {
			_ = json.NewDecoder(r.Body).Decode(&g)
			g["id"] = id
			f.groups[id] = g
		}
		if r.Method == "GET" && f.injectPeerOnGroupRead != "" {
			if name, _ := g["name"].(string); strings.HasPrefix(name, "fvpn-enroll-peer-") {
				g["peers"] = []any{f.injectPeerOnGroupRead}
				if f.key != nil {
					f.key["used_times"] = 1
					f.key["state"] = "overused"
				}
				f.injectPeerOnGroupRead = ""
			}
		}
		respond(g)
	case path == "setup-keys" && r.Method == "GET":
		keys := []any{}
		if f.key != nil {
			keys = append(keys, f.key)
		}
		respond(keys)
	case path == "setup-keys" && r.Method == "POST":
		_ = json.NewDecoder(r.Body).Decode(&f.key)
		f.key["id"], f.key["key"] = "TEST-key", "TEST ONLY generated secret"
		f.key["used_times"], f.key["revoked"], f.key["state"] = 0, false, "valid"
		f.key["expires"] = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
		respond(f.key)
		delete(f.key, "key")
	case path == "setup-keys/TEST-key":
		if f.key == nil {
			w.WriteHeader(404)
			return
		}
		if r.Method == "PUT" {
			if f.beforeKeyUpdate != nil {
				hook := f.beforeKeyUpdate
				f.beforeKeyUpdate = nil
				hook()
			}
			var v map[string]any
			_ = json.NewDecoder(r.Body).Decode(&v)
			for k, value := range v {
				f.key[k] = value
			}
		}
		respond(f.key)
	case path == "peers/TEST-peer" && r.Method == "DELETE":
		f.peerDeleted = true
		respond(map[string]any{})
	case path == "peers/TEST-peer" && r.Method == "GET":
		if f.peerDeleted {
			w.WriteHeader(404)
			return
		}
		if f.beforePeerRead != nil {
			hook := f.beforePeerRead
			f.beforePeerRead = nil
			hook()
		}
		respond(map[string]any{"id": "TEST-peer", "user_id": "", "ephemeral": false, "ip": "100.64.0.42"})
	default:
		w.WriteHeader(404)
	}
}

func (f *discoveryManagement) useKey(peers ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.key["used_times"], f.key["state"] = 1, "overused"
	for _, g := range f.groups {
		g["peers"] = peers
	}
}

type discoveryWorker struct {
	r        *httpwire.Runtime
	enrolled enrollment.EnrollResult
	report   api.ObservationReport
	app      *Application
	jobs     Registry
	fake     *discoveryManagement
	client   *netbird.Client
}

func newDiscoveryWorker(t *testing.T) *discoveryWorker {
	t.Helper()
	r, enrolled, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
	f := &discoveryManagement{groups: map[string]map[string]any{}}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	nb, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: r.Config.NetBirdDeployment, Token: observationToken{}, Bindings: nodelib.NetBirdBindings{Pool: r.Pool}, AllowInsecureHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.WorkerAttempts = 2
	jobs := NodeJobs(r, nb, nil, cfg.WorkerAttempts, exits.NewFakeNetBirdAdapter())
	app, err := Open(t.Context(), Dependencies{Pool: r.Pool, Config: cfg}, Domain{Name: "nodes", Build: func(context.Context, Dependencies) (Built, error) { return Built{Jobs: jobs}, nil }})
	if err != nil {
		t.Fatal(err)
	}
	fw := &discoveryWorker{r: r, enrolled: enrolled, report: report, app: app, jobs: jobs, fake: f, client: nb}
	fw.dispatch(t, "node.provision.requested")
	return fw
}

func (f *discoveryWorker) event(t *testing.T, kind string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT id FROM outbox_events WHERE event_type=$1 AND delivered_at IS NULL AND dead_lettered_at IS NULL ORDER BY created_at LIMIT 1`, kind).Scan(&id); err != nil {
		t.Fatal(kind, err)
	}
	return id
}

func (f *discoveryWorker) selectEvent(t *testing.T, id uuid.UUID) {
	t.Helper()
	if _, err := f.r.Pool.Exec(t.Context(), `UPDATE outbox_events SET available_at=CASE WHEN id=$1 THEN clock_timestamp() ELSE clock_timestamp()+interval '1 day' END`, id); err != nil {
		t.Fatal(err)
	}
}

func (f *discoveryWorker) dispatch(t *testing.T, kind string) uuid.UUID {
	t.Helper()
	id := f.event(t, kind)
	f.selectEvent(t, id)
	if worked, err := f.app.Once(t.Context()); err != nil || !worked {
		t.Fatal(kind, worked, err)
	}
	return id
}

func (f *discoveryWorker) observe(t *testing.T, present bool) {
	t.Helper()
	f.report.ReportId, f.report.ObservedAt = uuid.New(), f.report.ObservedAt.Add(time.Second)
	if present {
		f.report.Netbird.PeerId.Set("TEST-peer")
	}
	if _, err := f.r.Nodes.SubmitObservation(t.Context(), f.report.NodeId, f.report.NodeIncarnationId, f.report); err != nil {
		t.Fatal(err)
	}
	f.dispatch(t, "node.peer.observe")
}

func TestAT06_D055_NullObservationDiscoversManagementPeer(t *testing.T) {
	f := newDiscoveryWorker(t)
	// First provisioning-triggered attempt sees the unused key. The same event
	// recovers after management registration, without another key or group.
	id := f.dispatch(t, "node.peer.discover")
	var pending bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT delivered_at IS NULL AND dead_lettered_at IS NULL AND attempt=1 FROM outbox_events WHERE id=$1`, id).Scan(&pending); err != nil || !pending {
		t.Fatal(pending, err)
	}
	f.fake.useKey("TEST-peer")
	f.observe(t, false)
	if replay := f.dispatch(t, "node.peer.discover"); replay != id {
		t.Fatal("discovery event was replaced")
	}
	var peer, deployment string
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT peer_id,deployment_id FROM node_incarnations WHERE id=$1`, f.report.NodeIncarnationId).Scan(&peer, &deployment); err != nil || peer != "TEST-peer" || deployment != f.r.Config.NetBirdDeployment {
		t.Fatal(peer, deployment, err)
	}
	op, err := nodelib.GetOperation(t.Context(), f.r.Pool, f.enrolled.OperationID)
	if err != nil || op.Status != api.OperationStatusSucceeded {
		t.Fatal(op, err)
	}
	for _, step := range op.Steps {
		if step.Name == "peer_readback" && (step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull()) {
			t.Fatal(step)
		}
	}
	var delivered, dead bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT delivered_at IS NOT NULL,dead_lettered_at IS NOT NULL FROM outbox_events WHERE id=$1`, id).Scan(&delivered, &dead); err != nil || !delivered || dead {
		t.Fatal(delivered, dead, err)
	}
	f.fake.mu.Lock()
	if len(f.fake.groups) != 1 {
		t.Error("temporary group remains", len(f.fake.groups))
	}
	for _, g := range f.fake.groups {
		if g["name"] != "fvpn-unpublished-nodes" {
			t.Error("shared group removed", g["name"])
		}
	}
	f.fake.mu.Unlock()
	// Both agents that cannot supply a management ID and agents that explicitly
	// report the bound peer continue through EnsureNodePeer on later reports.
	f.observe(t, false)
	f.observe(t, true)
	detail, _, err := f.r.Nodes.GetNode(t.Context(), f.report.NodeId)
	if err != nil || detail.ProvisioningState != api.ProvisioningStateReady {
		t.Fatal(detail, err)
	}
}

func TestAT06_D055_DiscoveryConflictIsTerminalAndHeartbeatCannotClearIt(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer", "TEST-second-peer")
	f.observe(t, false)
	id := f.dispatch(t, "node.peer.discover")
	var frozen, noPeer, delivered bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT x.mapping_frozen,i.peer_id IS NULL,o.delivered_at IS NOT NULL FROM node_identity_endpoints x JOIN node_incarnations i ON i.id=x.incarnation_id CROSS JOIN outbox_events o WHERE i.id=$1 AND o.id=$2`, f.report.NodeIncarnationId, id).Scan(&frozen, &noPeer, &delivered); err != nil || !frozen || !noPeer || !delivered {
		t.Fatal(frozen, noPeer, delivered, err)
	}
	f.observe(t, false)
	detail, _, err := f.r.Nodes.GetNode(t.Context(), f.report.NodeId)
	if err != nil || detail.ProvisioningState != api.ProvisioningStateFailed || detail.SafetyStatus != api.SafetyStatusFailed || len(detail.RecentErrors) != 1 || detail.RecentErrors[0].Message != "multiple_group_peers" {
		t.Fatal(detail, err)
	}
	op, err := nodelib.GetOperation(t.Context(), f.r.Pool, f.enrolled.OperationID)
	if err != nil || op.Status != api.OperationStatusFailed {
		t.Fatal(op, err)
	}
	var count int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE operation_id=$1 AND action='node.peer_discovery_failed'`, f.enrolled.OperationID).Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	_, bound, err := (nodelib.NetBirdBindings{Pool: f.r.Pool}).Get(t.Context(), f.r.Config.NetBirdDeployment, "peers", "node:"+f.report.NodeId.String()+":"+f.report.NodeIncarnationId.String())
	if err != nil || bound {
		t.Fatal("invented adapter binding", bound, err)
	}
}

func TestAT06_D055_UnusedKeyExhaustsBoundedRetries(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.observe(t, false)
	id := f.dispatch(t, "node.peer.discover")
	var attempt int
	var delivered, dead bool
	read := func() {
		t.Helper()
		if err := f.r.Pool.QueryRow(t.Context(), `SELECT attempt,delivered_at IS NOT NULL,dead_lettered_at IS NOT NULL FROM outbox_events WHERE id=$1`, id).Scan(&attempt, &delivered, &dead); err != nil {
			t.Fatal(err)
		}
	}
	read()
	if attempt != 1 || delivered || dead {
		t.Fatal(attempt, delivered, dead)
	}
	f.dispatch(t, "node.peer.discover")
	read()
	if attempt != 2 || delivered || !dead {
		t.Fatal(attempt, delivered, dead)
	}
	op, err := nodelib.GetOperation(t.Context(), f.r.Pool, f.enrolled.OperationID)
	if err != nil || op.Status != api.OperationStatusFailed {
		t.Fatal(op, err)
	}
	f.observe(t, false)
	var count int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type='node.peer.discover'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("heartbeat reset retry budget", count, err)
	}
}

func (f *discoveryWorker) revoke(t *testing.T) {
	t.Helper()
	detail, _, err := f.r.Nodes.GetNode(t.Context(), f.report.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.r.Enrollment.RevokeNode(t.Context(), enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, f.report.NodeId, &detail.ResourceVersion, "TEST ONLY discovery revoke")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAT07_D055_RevokedEnrollmentNeverDiscovers(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		t.Run(fmt.Sprint(concurrent), func(t *testing.T) {
			f := newDiscoveryWorker(t)
			f.fake.useKey("TEST-peer")
			if concurrent {
				f.fake.beforePeerRead = func() { f.revoke(t) }
			} else {
				f.revoke(t)
			}
			f.fake.mu.Lock()
			before := f.fake.calls
			f.fake.mu.Unlock()
			f.dispatch(t, "node.peer.discover")
			var noPeer bool
			if err := f.r.Pool.QueryRow(t.Context(), `SELECT peer_id IS NULL FROM node_incarnations WHERE id=$1`, f.report.NodeIncarnationId).Scan(&noPeer); err != nil || !noPeer {
				t.Fatal(noPeer, err)
			}
			_, bound, err := (nodelib.NetBirdBindings{Pool: f.r.Pool}).Get(t.Context(), f.r.Config.NetBirdDeployment, "peers", "node:"+f.report.NodeId.String()+":"+f.report.NodeIncarnationId.String())
			if err != nil || bound {
				t.Fatal("revoked binding", bound, err)
			}
			f.fake.mu.Lock()
			defer f.fake.mu.Unlock()
			if !concurrent && f.fake.calls != before {
				t.Fatal("revoked enrollment reached management")
			}
		})
	}
}

func TestAT06_D055_DiscoveryClaimLostBeforeBinding(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer")
	id := f.event(t, "node.peer.discover")
	f.selectEvent(t, id)
	events, err := db.ClaimOutbox(t.Context(), f.r.Pool, "TEST ONLY discovery", 1, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatal(events, err)
	}
	f.fake.beforePeerRead = func() {
		if _, err := f.r.Pool.Exec(t.Context(), `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, id); err != nil {
			t.Error(err)
		}
	}
	if _, err = f.jobs["node.peer.discover"](t.Context(), events[0]); err != db.ErrLeaseLost {
		t.Fatal("stale discovery committed", err)
	}
	_, bound, err := (nodelib.NetBirdBindings{Pool: f.r.Pool}).Get(t.Context(), f.r.Config.NetBirdDeployment, "peers", "node:"+f.report.NodeId.String()+":"+f.report.NodeIncarnationId.String())
	if err != nil || bound {
		t.Fatal(bound, err)
	}
}

func TestAT07_D055_RevokeCleansBindingBeforeCentralProjection(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer")
	// The adapter has recorded management proof but has not yet returned to
	// the fenced central transaction. Revocation must clean that durable binding
	// even though node_incarnations.peer_id is still null.
	f.fake.beforeKeyUpdate = func() { f.revoke(t) }
	f.dispatch(t, "node.peer.discover")
	var noPeer bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT peer_id IS NULL FROM node_incarnations WHERE id=$1`, f.report.NodeIncarnationId).Scan(&noPeer); err != nil || !noPeer {
		t.Fatal(noPeer, err)
	}
	id := f.dispatch(t, "node.publication_changed")
	var opID uuid.UUID
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT (payload->>'operation_id')::uuid FROM outbox_events WHERE id=$1`, id).Scan(&opID); err != nil {
		t.Fatal(err)
	}
	op, err := nodelib.GetOperation(t.Context(), f.r.Pool, opID)
	if err != nil || op.Status != api.OperationStatusPartiallySucceeded {
		t.Fatal(op, err)
	}
	for _, step := range op.Steps {
		if step.Name == nodelib.StepNetBirdIdentityCleanup && (step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull()) {
			t.Fatal(step)
		}
	}
	f.fake.mu.Lock()
	defer f.fake.mu.Unlock()
	if !f.fake.peerDeleted || f.fake.key["revoked"] != true || len(f.fake.groups) != 1 {
		t.Fatal("withdrawal lacks confirmed identity cleanup")
	}
}

func TestAT07_D055_AmbiguousWithdrawalRetainsCorrelationAndAuditsFailure(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer", "TEST-second-peer")
	f.revoke(t)
	event := f.dispatch(t, "node.publication_changed")
	f.dispatch(t, "node.publication_changed")
	var succeeded, failed int
	var dead bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM audit_events WHERE action='node.identity_withdrawn'),(SELECT count(*) FROM audit_events WHERE action='node.identity_withdrawal_failed' AND after_diff->>'reason'='multiple_group_peers'),(SELECT dead_lettered_at IS NOT NULL FROM outbox_events WHERE id=$1)`, event).Scan(&succeeded, &failed, &dead); err != nil || succeeded != 0 || failed != 2 || !dead {
		t.Fatal("ambiguous withdrawal claimed completion", succeeded, failed, dead, err)
	}
	f.fake.mu.Lock()
	defer f.fake.mu.Unlock()
	if f.fake.peerDeleted {
		t.Fatal("guessed a peer to delete")
	}
	retained := false
	for _, group := range f.fake.groups {
		if name, _ := group["name"].(string); strings.HasPrefix(name, "fvpn-enroll-peer-") {
			retained = true
		}
	}
	if !retained {
		t.Fatal("discarded ambiguous correlation")
	}
}
