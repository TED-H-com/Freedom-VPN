//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Run against a fresh prepared prefix. Evidence is isolated from full acceptance.
func TestE2E_HarnessExports(t *testing.T) {
	if os.Getenv("FVPN_E2E_EXPORTS") != "1" {
		t.Skip("set FVPN_E2E_EXPORTS=1 with a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	r := &runner{t: t, lab: l, runID: uuid.NewString(), sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup14", "exports")}
	// Expected product gaps are recorded independently from export failures.
	defer func() {
		t.Logf("exports summary: pass=%d blocked=%d fail=%d run=%s source=%s", r.counts[StatusPass], r.counts[StatusBlocked], r.counts[StatusFail], r.runID, r.sourceCommit)
		if r.counts[StatusFail] > 0 {
			t.Error("export failure")
		}
	}()
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		t.Fatal("admin login")
	}
	r.threatFeed(ctx, admin)
	r.catalogue(ctx, admin)
	good, bad := r.releases(ctx, admin)
	node := r.enroll(ctx, admin)
	if node == uuid.Nil {
		t.Fatal("node enrollment")
	}
	r.desired(ctx)
	r.publish(ctx, admin, node)
	r.member(ctx, admin)
	r.exportsNative(ctx, admin)
	r.updater(ctx, good, bad)
	s := &Stack{lab: l}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	// Enrollment fixture is independent of the serving source and clone target.
	r.exportsEnrollment(ctx, s, b)
	r.exportsClone(ctx, s)
}
func (r *runner) exportsNative(ctx context.Context, admin *session) {
	rec := Record{ID: "export-native-streams", Title: "Bounded final native stream inventories", Spec: []string{"SECOPS-16"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	s := &Stack{lab: r.lab}
	counts, err := s.ScaleNativeMembers(ctx, BrowserSession{Client: admin.http, CSRF: admin.csrf}, r.planRevID, "exports", 1, 2)
	raw, _ := json.Marshal(counts)
	recPathNote(&rec)("final_results", string(raw))
	if err != nil {
		recPathNote(&rec)("error", err.Error())
		return
	}
	ok := len(counts.Streams) == 2 && counts.Authorized == 2
	for _, stream := range counts.Streams {
		ok = ok && stream.Result == "scanned" && stream.Error == "" && stream.Scan.DeviceFlowKeys == 1 && stream.Scan.Mode == "0:600" && stream.Deleted
	}
	// Missing stream and refused target are explicit non-success results.
	missing, e := s.ScanNativeJoinLog(ctx, r.lab.member)
	recPathNote(&rec)("deleted_stream_missing", boolText(e == nil && !missing.Present))
	_, e = s.ScanNativeJoinLog(ctx, "outside-stack")
	recPathNote(&rec)("foreign_container_refused", boolText(e != nil))
	if ok && e != nil {
		rec.Status = StatusPass
	}
}
func (r *runner) exportsEnrollment(ctx context.Context, s *Stack, admin BrowserSession) {
	rec := Record{ID: "export-enrollment", Title: "Harness key replay, race and actual one-time peer bootstrap", Spec: []string{"§6.2", "AT06", "ENR-04"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	if err := s.StartSecondExit(ctx); err != nil {
		note("start", err.Error())
		return
	}
	name := "TEST ONLY harness key race"
	resp, err := s.Call(ctx, admin, "POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var created api.EnrollmentCreated
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &created) != nil {
		note("create", httpDetail(resp, err))
		return
	}
	token, err := created.Token.Get()
	if err != nil {
		return
	}
	a, err := s.GenerateEnrollmentIdentity(ctx, "node2", "key-a")
	if err != nil {
		note("generate", err.Error())
		return
	}
	b, err := s.GenerateEnrollmentIdentity(ctx, "node2", "key-b")
	if err != nil {
		note("generate", err.Error())
		return
	}
	for _, in := range []EnrolledIdentityInput{a, b} {
		if s.StageEnrollmentToken(ctx, "node2", in, token) != nil {
			return
		}
	}
	// Two different keys race one API-issued code; both retain identical bytes for retries.
	type attempt struct {
		In     EnrolledIdentityInput
		Result EnrolledIdentityResult
		Err    error
	}
	done := make(chan attempt, 2)
	for _, in := range []EnrolledIdentityInput{a, b} {
		go func() {
			in.DiscardResponse = true
			in.Join = false
			out, e := s.JoinEnrolledIdentity(ctx, "node2", in)
			done <- attempt{in, out, e}
		}()
	}
	var winner attempt
	won, lost := 0, 0
	for range 2 {
		v := <-done
		raw, _ := json.Marshal(v.Result)
		note("race_result", string(raw))
		if v.Err != nil {
			note("race_error", v.Err.Error())
			return
		}
		if v.Result.EnrollHTTP == 201 || v.Result.EnrollHTTP == 200 {
			won++
			winner = v
		} else if v.Result.EnrollHTTP == 409 {
			lost++
		}
	}
	if won != 1 || lost != 1 {
		note("race_counts", fmt.Sprintf("accepted=%d rejected=%d", won, lost))
		return
	}
	winner.In.DiscardResponse = false
	winner.In.Join = true
	replay, err := s.JoinEnrolledIdentity(ctx, "node2", winner.In)
	raw, _ := json.Marshal(replay)
	note("replay_join", string(raw))
	if err != nil || !replay.Joined || replay.RequestSHA256 != winner.Result.RequestSHA256 {
		if err != nil {
			note("join_error", err.Error())
		}
		return
	}
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var inventory EnrollmentPeers
	ok := waitUntil(wait, time.Second, func() bool {
		inventory, err = s.EnrollmentPeerInventory(wait, admin, created.Enrollment.EnrollmentId)
		return err == nil && inventory.Complete
	})
	raw, _ = json.Marshal(inventory)
	note("peer_inventory", string(raw))
	note("incarnation_history", "current API incarnation observed; complete history/count remains ENR-03, nodes/control + PKI")
	if ok && inventory.Peers == 1 && inventory.SetupKeyUses == 1 && inventory.IncarnationID == replay.IncarnationID {
		rec.Status = StatusPass
	}
}
func (r *runner) exportsClone(ctx context.Context, s *Stack) {
	rec := Record{ID: "export-clone", Title: "Pre-boot cloned enrolled image and hash-only machine identity", Spec: []string{"§6.2", "AT07", "ENR-07"}, Status: StatusFail}
	note := recPathNote(&rec)
	fixture, err := s.StartClonedIdentityFixture(ctx, "node")
	if err != nil {
		note("start", err.Error())
		r.record(rec)
		return
	}
	wait, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	var obs CloneObservation
	ok := waitUntil(wait, time.Second, func() bool {
		obs, err = fixture.Observe(wait)
		return err == nil && obs.SameIdentity && obs.SourceStage == "enrolled" && obs.CloneStage == "enrolled"
	})
	if ok {
		rec.Status = StatusPass
	}
	raw, _ := json.Marshal(obs)
	note("clone_observation", string(raw))
	note("snapshot", "source stopped; durable state and native identity copied before clone boot; source resumed")
	r.record(rec)
	r.cloneConflict(ctx, s, fixture)
}

func (r *runner) cloneConflict(ctx context.Context, s *Stack, fixture *ClonedIdentityFixture) {
	product := Record{ID: "clone-conflict", Title: "Cloned identity conflict and automatic-mapping freeze", Spec: []string{"§6.2", "AT07", "D-072"}, Status: StatusFail}
	note := recPathNote(&product)
	defer func() { r.record(product) }()
	pres, err := s.clonePresentation()
	if err != nil {
		note("presentation", "unavailable")
		return
	}
	note("presentation_subnet", pres.Subnet)
	note("presentation_network", pres.Network)
	admin := r.mustAdmin(ctx)
	if admin == nil || r.nodeID == uuid.Nil {
		note("admin", "unavailable")
		return
	}
	window, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	var view cloneConflictView
	var audits int
	var auditErr error
	frozen := waitUntil(window, 2*time.Second, func() bool {
		view, err = r.readCloneConflict(window, admin)
		if err != nil || !view.Object || !view.Frozen || view.ConflictCount < 1 || view.Provisioning != string(api.ProvisioningStateFailed) || view.Forwarding != "false" {
			return false
		}
		audits, auditErr = r.countConflictAudits(window, admin)
		return auditErr == nil && audits == 1
	})
	if err != nil && !frozen {
		note("admin_read", "failed")
	}
	note("mapping_frozen", boolText(view.Frozen))
	note("conflict_count", itoa(view.ConflictCount))
	note("last_conflict_at_present", boolText(view.LastConflict))
	note("provisioning_state", emptyDash(view.Provisioning))
	note("member_forwarding_observed", view.Forwarding)
	note("identity_conflict_audits", itoa(audits))
	note("audit_read", boolText(auditErr == nil))
	if !frozen || view.Provisioning != string(api.ProvisioningStateFailed) || view.Forwarding != "false" || auditErr != nil || audits != 1 {
		note("owner", "w3-wire-nodes: D-072 lease renewal and observation; identity_conflict, one node.identity.conflict audit, provisioning failed, member forwarding off")
		return
	}
	time.Sleep(8 * time.Second)
	again, againErr := r.countConflictAudits(ctx, admin)
	note("identity_conflict_audits_after_wait", itoa(again))
	if againErr != nil || again != 1 {
		note("owner", "w3-wire-nodes: a later frozen request emitted another node.identity.conflict audit")
		return
	}
	if !r.reconcileClone(ctx, admin, fixture, note) {
		return
	}
	product.Status = StatusPass
	note("note", "second /24 presentation; admin identity_conflict; revoke and entrypoint re-enroll of the same node")
}

type cloneConflictView struct {
	Object        bool
	Frozen        bool
	ConflictCount int
	LastConflict  bool
	Provisioning  string
	Forwarding    string
}

func (r *runner) readCloneConflict(ctx context.Context, admin *session) (cloneConflictView, error) {
	var view cloneConflictView
	resp, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
	if err != nil || resp.Status != 200 {
		return view, fmt.Errorf("node detail")
	}
	var detail api.NodeDetail
	if json.Unmarshal(resp.Body, &detail) != nil {
		return view, fmt.Errorf("node detail")
	}
	view.Provisioning = string(detail.ProvisioningState)
	if detail.MemberForwardingObserved.IsNull() {
		view.Forwarding = "null"
	} else if on, getErr := detail.MemberForwardingObserved.Get(); getErr == nil {
		view.Forwarding = boolText(on)
	}
	if detail.IdentityConflict.IsNull() || !detail.IdentityConflict.IsSpecified() {
		return view, nil
	}
	conflict, err := detail.IdentityConflict.Get()
	if err != nil {
		return view, nil
	}
	view.Object = true
	view.Frozen = conflict.MappingFrozen
	view.ConflictCount = conflict.ConflictCount
	view.LastConflict = !conflict.LastConflictAt.IsNull()
	return view, nil
}

func (r *runner) countConflictAudits(ctx context.Context, admin *session) (int, error) {
	cursor := ""
	total := 0
	for page := 0; page < 20; page++ {
		path := r.lab.public + "/api/v1/admin/audit-events?resource_type=node&resource_id=" + r.nodeID.String() + "&limit=100"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, err := admin.call(ctx, "GET", path, nil, nil)
		if err != nil || resp.Status != 200 {
			return total, fmt.Errorf("audit list")
		}
		var list api.AuditEventList
		if json.Unmarshal(resp.Body, &list) != nil {
			return total, fmt.Errorf("audit list")
		}
		for _, item := range list.Items {
			if item.Action == "node.identity.conflict" {
				total++
			}
		}
		next, err := list.NextCursor.Get()
		if err != nil || next == "" {
			return total, nil
		}
		cursor = next
	}
	return total, fmt.Errorf("audit list truncated")
}

func (r *runner) reconcileClone(ctx context.Context, admin *session, fixture *ClonedIdentityFixture, note func(string, string)) bool {
	_ = exec.CommandContext(ctx, "docker", "stop", "-t", "15", r.lab.prefix+"-node-clone").Run()
	detail, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
	if err != nil || detail.Status != 200 {
		note("revoke_get", httpDetail(detail, err))
		return false
	}
	rev, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String()+"/revoke", api.ReasonRequest{Reason: "e2e clone conflict reconcile"}, map[string]string{
		"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header.Get("ETag"),
	})
	note("revoke", httpDetail(rev, err))
	if err != nil || rev.Status != 202 {
		note("owner", "w3-wire-nodes: revoke of a frozen node")
		return false
	}
	name := "e2e clone reconcile"
	node := openapi_types.UUID(r.nodeID)
	created, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller, Name: &name, NodeId: &node,
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var body api.EnrollmentCreated
	if err != nil || created.Status != 201 || json.Unmarshal(created.Body, &body) != nil {
		note("reenrollment", httpDetail(created, err))
		note("owner", "w3-wire-nodes: createEnrollment on the revoked node")
		return false
	}
	token, err := body.Token.Get()
	if err != nil || token == "" {
		note("reenrollment", "token absent")
		return false
	}
	if err = exec.CommandContext(ctx, "docker", "stop", "-t", "20", r.lab.node).Run(); err != nil {
		note("node_stop", "failed")
		return false
	}
	// identity/ and state/ are root-owned mode 0700. The test process cannot
	// unlink them, and a silent os.Remove leaves the old incarnation in place.
	// The node is stopped, so a one-shot root container removes the files.
	if err = removeStoppedNodeFiles(ctx, r.lab.node, filepath.Join(r.lab.state, "node", "var"), []string{
		"identity/node.json", "identity/cert.pem", "identity/key.pem", "identity/setup.key",
		"state/agent.db", "state/agent.db-wal", "state/agent.db-shm",
	}); err != nil {
		note("identity_clear", "failed")
		note("owner", "w3-devstack-e2e: root-owned identity files were not removed")
		return false
	}
	e2eDir := filepath.Join(r.lab.state, "node", "var", "e2e")
	_ = os.Remove(filepath.Join(e2eDir, "pause-agent"))
	// status.json still says enrolled from the incarnation we just revoked.
	// Leave it and the wait below returns before the entrypoint enrolls.
	if err = os.Remove(filepath.Join(e2eDir, "status.json")); err != nil && !os.IsNotExist(err) {
		note("status_reset", "failed")
		return false
	}
	_ = os.WriteFile(filepath.Join(e2eDir, "agent-selected"), []byte("/usr/local\n"), 0o644)
	if err = writeAtomic(filepath.Join(e2eDir, "enrollment.token"), []byte(token+"\n")); err != nil {
		note("token_file", "failed")
		return false
	}
	token = ""
	if err = exec.CommandContext(ctx, "docker", "start", r.lab.node).Run(); err != nil {
		note("node_start", "failed")
		return false
	}
	note("reenroll_path", "entrypoint waiting_token after identity files and agent.db were removed; not an installer wrapper")
	enrollWait, enrollCancel := context.WithTimeout(ctx, 4*time.Minute)
	enrolled := waitUntil(enrollWait, time.Second, func() bool {
		return r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/status.json") == "enrolled"
	})
	enrollCancel()
	note("reenroll_stage", emptyDash(r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/status.json")))
	if !enrolled {
		note("owner", "w3-devstack-e2e: replacement incarnation did not enroll through the entrypoint")
		return false
	}
	raw, err := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	var identity struct {
		Inc uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &identity) != nil || identity.Inc == uuid.Nil || identity.Inc == fixture.incarnation {
		note("replacement_incarnation", "not a new incarnation")
		return false
	}
	clearWait, clearCancel := context.WithTimeout(ctx, 90*time.Second)
	defer clearCancel()
	var cleared cloneConflictView
	ok := waitUntil(clearWait, 2*time.Second, func() bool {
		cleared, err = r.readCloneConflict(clearWait, admin)
		return err == nil && cleared.Object && !cleared.Frozen && cleared.ConflictCount == 0 && !cleared.LastConflict
	})
	note("reconciled_mapping_frozen", boolText(cleared.Frozen))
	note("reconciled_conflict_count", itoa(cleared.ConflictCount))
	note("reconciled_last_conflict_null", boolText(!cleared.LastConflict && cleared.Object))
	if !ok {
		note("owner", "w3-wire-nodes: new incarnation was not observed as {mapping_frozen:false, conflict_count:0, last_conflict_at:null}")
		return false
	}
	return true
}

// removeStoppedNodeFiles deletes root-owned files on a stopped node's bind
// mount. The helper name is prefix-owned (state-reset) and the container is
// removed when the command exits.
func removeStoppedNodeFiles(ctx context.Context, container, hostVar string, rels []string) error {
	image, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Config.Image}}", container).Output()
	if err != nil {
		return err
	}
	base, ok := strings.CutSuffix(container, "-node")
	if !ok || !strings.HasPrefix(base, "fvpn-") {
		return fmt.Errorf("identity clear name")
	}
	name := base + "-state-reset"
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
	args := []string{"run", "--rm", "--name", name, "--network", "none", "--user", "0",
		"--mount", "type=bind,src=" + hostVar + ",dst=/var/lib/vpn-node",
		"--entrypoint", "rm", strings.TrimSpace(string(image)), "-f"}
	for _, rel := range rels {
		if strings.Contains(rel, "..") {
			return fmt.Errorf("identity clear path")
		}
		args = append(args, "/var/lib/vpn-node/"+rel)
	}
	if err = exec.CommandContext(ctx, "docker", args...).Run(); err != nil {
		return err
	}
	return nil
}
