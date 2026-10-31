//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	_ "modernc.org/sqlite"
)

func TestE2E_Acceptance(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("set FVPN_E2E=1 to run the dev stack acceptance")
	}
	lab, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()
	run := &runner{t: t, lab: lab, dir: filepath.Join(lab.root, "docs", "e2e", "runs", time.Now().UTC().Format("2006-01-02"), "followup15")}
	run.runID = uuid.NewString()
	commit, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	run.sourceCommit = strings.TrimSpace(string(commit))
	defer run.finish()

	run.health(ctx)
	run.metrics(ctx)
	run.gatewayPrincipals(ctx)
	admin := run.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	run.backup(ctx, admin)
	run.threatFeed(ctx, admin)
	run.catalogue(ctx, admin)
	run.billing(ctx, admin)
	good, bad := run.releases(ctx, admin)
	nodeID := run.enroll(ctx, admin)
	if nodeID != uuid.Nil {
		run.agentSupervisor(ctx)
	}
	run.desired(ctx)
	if nodeID != uuid.Nil {
		run.threatFeedNode(ctx, admin)
		run.restartNode(ctx)
	}
	run.publish(ctx, admin, nodeID)
	if nodeID != uuid.Nil {
		run.member(ctx, admin)
		run.establishedTransports(ctx)
		run.forcedRelay(ctx)
		run.exitDNS(ctx)
		run.memberIPv6(ctx)
		run.workerRestart(ctx)
		run.deviceRevoke(ctx)
		run.scaleHook(ctx, admin)
		run.gatewayPrivacy(ctx)
		run.threatFeedTamper(ctx, admin)
		run.updater(ctx, good, bad)
		run.controlStop(ctx)
		run.revoke(ctx, nodeID)
		run.secondExit(ctx)
	} else {
		run.forwarding(run.lab.forwardChain())
	}
}

type runner struct {
	t            *testing.T
	lab          *lab
	dir          string
	counts       map[Status]int
	bad          bool
	runID        string
	sourceCommit string
	nodeID       uuid.UUID
	incID        uuid.UUID
	enrolledID   uuid.UUID
	groupID      uuid.UUID
	profileID    uuid.UUID
	planError    string
	planID       uuid.UUID
	planRevID    uuid.UUID
	usageDevice  uuid.UUID
	dnsApplied   bool
	flowCounted  bool
	trafficOwner string
	usageReady   bool
	probePeer    map[string]any
	probe        *UpdaterProbe
	updateOwner  string
}

func (r *runner) record(rec Record) {
	rec.RunID, rec.SourceCommit = r.runID, r.sourceCommit
	if r.counts == nil {
		r.counts = map[Status]int{}
	}
	r.counts[rec.Status]++
	r.t.Logf("scenario %s: %s", rec.ID, rec.Status)
	if rec.Status == StatusBlocked {
		owners := map[string]string{
			"publish":        "w3-wire-nodes: publication operation and production peer projection",
			"member-traffic": r.trafficOwner,
			"forwarding":     r.trafficOwner,
			"usage":          "w3-wire-usage: real sampler/usage projection after counted flow",
			"update-good":    "w4-node-updater and w2-agent-core: D-053 live candidate health",
			"update-bad":     "w4-node-updater: tampered artifact rejection and recovery",
			"lease-expiry":   r.trafficOwner,
		}
		if rec.ID == "usage" && !r.flowCounted {
			owners[rec.ID] = r.trafficOwner
		}
		if (rec.ID == "member-traffic" || rec.ID == "forwarding") && owners[rec.ID] == "" {
			owners[rec.ID] = "w3-wire-members: native device authorization and desired binding prerequisite"
		}
		if rec.ID == "update-good" && r.updateOwner != "" {
			owners[rec.ID] = r.updateOwner
		}
		if owner := owners[rec.ID]; owner != "" {
			rec.Observations = append(rec.Observations, Observation{Name: "owner", Detail: owner})
		}
	}
	if rec.Status != StatusPass {
		r.bad = true
	}
	if err := Write(r.dir, rec); err != nil {
		r.t.Errorf("evidence %s: %v", rec.ID, err)
	}
}

func (r *runner) finish() {
	r.t.Logf("acceptance summary: pass=%d blocked=%d fail=%d run=%s source=%s", r.counts[StatusPass], r.counts[StatusBlocked], r.counts[StatusFail], r.runID, r.sourceCommit)
	if r.bad {
		r.t.Error("one or more scenarios are blocked or failed; see docs/e2e/runs")
	}
}

func (r *runner) health(ctx context.Context) {
	rec := Record{ID: "stack-health", Title: "Dev stack answers", Spec: []string{"§3", "§17"}, Status: StatusPass}
	client := &http.Client{Transport: r.lab.transport, Timeout: 10 * time.Second}
	checks := []struct {
		name, url string
	}{
		{"control", r.lab.public + "/readyz"},
		{"idp", r.lab.idp + "/healthz"},
		{"origin", r.lab.releases + "/healthz"},
	}
	for _, c := range checks {
		resp, err := client.Get(c.url)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			rec.Status = StatusFail
			rec.Observations = append(rec.Observations, Observation{Name: c.name, Detail: "down"})
			r.record(rec)
			r.t.Fatalf("%s is not healthy", c.name)
		}
		resp.Body.Close()
		rec.Observations = append(rec.Observations, Observation{Name: c.name, Detail: "200"})
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "postgres", Detail: "connection failed"})
		r.record(rec)
		r.t.Fatal("postgres")
	}
	_ = conn.Close(ctx)
	rec.Observations = append(rec.Observations, Observation{Name: "postgres", Detail: "connected"})
	r.record(rec)
}

func (r *runner) login(ctx context.Context, subject, role string) *session {
	s, err := r.lab.login(ctx, subject, role)
	rec := Record{ID: role + "-login", Title: "Browser session for " + role, Spec: []string{"§14.2"}, Status: StatusPass}
	if err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "error", Detail: err.Error()})
		r.record(rec)
		r.t.Fatalf("login %s: %v", role, err)
	}
	landing, landingErr := s.call(ctx, "GET", r.lab.public+"/", nil, nil)
	rec.Observations = append(rec.Observations, Observation{Name: "landing", Detail: httpDetail(landing, landingErr)})
	if landingErr != nil || landing.Status != 200 {
		rec.Status = StatusFail
	}
	rec.Observations = append(rec.Observations, Observation{Name: "role", Detail: s.role})
	if s.memberID != "" {
		rec.Observations = append(rec.Observations, Observation{Name: "member_id_present", Detail: "true"})
	}
	r.record(rec)
	return s
}

func (r *runner) enroll(ctx context.Context, admin *session) uuid.UUID {
	rec := Record{ID: "enroll-bootstrap", Title: "Installer token, one bootstrap, NetBird join", Spec: []string{"§6", "§19 AT06"}, Status: StatusFail}
	name := "e2e-exit"
	region := "lab"
	resp, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller, Name: &name, Region: &region,
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	if err != nil || resp.Status != 201 {
		rec.Observations = append(rec.Observations, Observation{Name: "issue", Detail: httpDetail(resp, err)})
		r.record(rec)
		return uuid.Nil
	}
	var created api.EnrollmentCreated
	if json.Unmarshal(resp.Body, &created) != nil || !created.SecretAvailable || created.Token.IsNull() {
		rec.Observations = append(rec.Observations, Observation{Name: "issue", Detail: "token absent"})
		r.record(rec)
		return uuid.Nil
	}
	token, _ := created.Token.Get()
	rec.Observations = append(rec.Observations, Observation{Name: "token_length", Detail: itoa(len(token))})
	rec.Observations = append(rec.Observations, Observation{Name: "secret_available", Detail: "true"})
	path := filepath.Join(r.lab.state, "node", "var", "e2e", "enrollment.token")
	if err := writeAtomic(path, []byte(token+"\n")); err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "issue", Detail: "token file"})
		r.record(rec)
		return uuid.Nil
	}
	token = ""
	nodeID := uuid.UUID(created.Enrollment.NodeId)
	rec.Observations = append(rec.Observations, Observation{Name: "node_id", Detail: nodeID.String()})

	enrollWait, enrollCancel := context.WithTimeout(ctx, 4*time.Minute)
	enrolled := waitUntil(enrollWait, time.Second, func() bool { return r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/status.json") == "enrolled" })
	enrollCancel()
	if !enrolled {
		rec.Observations = append(rec.Observations, Observation{Name: "stage", Detail: r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/status.json")})
		r.record(rec)
		return uuid.Nil
	}
	keyWait, keyCancel := context.WithTimeout(ctx, 2*time.Minute)
	keyReady := waitUntil(keyWait, time.Second, func() bool {
		return r.lab.fileNonEmpty(r.lab.node, "/var/lib/vpn-node/identity/setup.key")
	})
	keyCancel()
	if !keyReady {
		rec.Observations = append(rec.Observations, Observation{Name: "setup_key", Detail: "absent"})
		r.record(rec)
		return nodeID
	}
	rec.Observations = append(rec.Observations, Observation{Name: "setup_key_persisted", Detail: "true"})
	r.enrolledID = nodeID
	r.readIncarnation()
	body, _ := r.lab.docker(r.lab.node, "wget", "-q", "-O", "-", "--content-on-error",
		"--ca-certificate=/etc/vpn-node/ca.pem",
		"--certificate=/var/lib/vpn-node/identity/cert.pem",
		"--private-key=/var/lib/vpn-node/identity/key.pem",
		"--post-data=", r.lab.agent+"/api/v1/agent/bootstrap")
	code := errorCode(body)
	rec.Observations = append(rec.Observations, Observation{Name: "bootstrap_again", Detail: code})
	if code != "BOOTSTRAP_CONSUMED" {
		r.record(rec)
		return nodeID
	}
	nbWait, nbCancel := context.WithTimeout(ctx, 3*time.Minute)
	nbUp := waitUntil(nbWait, 2*time.Second, func() bool {
		return r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/netbird.json") == "netbird_up"
	})
	nbCancel()
	r.noteSignal(recPathNote(&rec))
	if !nbUp {
		rec.Observations = append(rec.Observations, Observation{Name: "netbird", Detail: r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/netbird.json")})
		r.record(rec)
		return nodeID
	}
	count := r.netbirdCount(ctx, "/api/peers")
	keys := r.netbirdCount(ctx, "/api/setup-keys")
	rec.Observations = append(rec.Observations, Observation{Name: "peer_count", Detail: itoa(count)})
	rec.Observations = append(rec.Observations, Observation{Name: "setup_key_count", Detail: itoa(keys)})
	if count < 1 || keys < 1 {
		r.record(rec)
		return nodeID
	}
	r.readIncarnation()
	r.nodeID = nodeID
	rec.Status = StatusPass
	r.record(rec)
	return nodeID
}

func (r *runner) desired(ctx context.Context) {
	rec := Record{ID: "desired-state", Title: "Signed desired state on the node", Spec: []string{"§5", "§8"}, Status: StatusBlocked}
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var raw []byte
	ok := waitUntil(wait, 2*time.Second, func() bool {
		b, err := os.ReadFile(filepath.Join(r.lab.state, "node", "var", "state", "desired-state.json"))
		if err != nil || len(b) == 0 {
			b, err = r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
		}
		if err != nil || len(b) == 0 {
			return false
		}
		raw = b
		return r.lab.verifyDesired(b) == nil
	})
	if !ok {
		rec.Observations = append(rec.Observations, Observation{Name: "envelope", Detail: "absent or rejected"})
		r.record(rec)
		return
	}
	rec.Status = StatusPass
	rec.Observations = append(rec.Observations, Observation{Name: "signature", Detail: "verified"})
	var env struct {
		Signed struct {
			ConfigGeneration string   `json:"config_generation"`
			Serving          []string `json:"serving_release_ids"`
			Update           any      `json:"update"`
		} `json:"signed"`
	}
	_ = json.Unmarshal(raw, &env)
	rec.Observations = append(rec.Observations, Observation{Name: "config_generation", Detail: env.Signed.ConfigGeneration})
	rec.Observations = append(rec.Observations, Observation{Name: "serving_count", Detail: itoa(len(env.Signed.Serving))})
	config, err := os.ReadFile(filepath.Join(r.lab.state, "node", "etc", "agent-run.json"))
	var boot struct {
		ReleaseID string `json:"release_id"`
	}
	if err == nil && json.Unmarshal(config, &boot) == nil {
		rec.Observations = append(rec.Observations, Observation{Name: "boot_release_id", Detail: emptyDash(boot.ReleaseID)})
		rec.Observations = append(rec.Observations, Observation{Name: "boot_release_supported", Detail: boolText(len(env.Signed.Serving) == 0 || containsCSV(strings.Join(env.Signed.Serving, ","), boot.ReleaseID))})
	}
	rec.Observations = append(rec.Observations, Observation{Name: "update_present", Detail: boolText(env.Signed.Update != nil)})
	r.record(rec)
}

func (r *runner) forwarding(view forwardView) {
	status := StatusBlocked
	detail := "forward chain policy is " + emptyDash(view.Policy) + ", generation " + itoa(view.Generation) + ", with " + itoa(view.MemberAccept) + " member accept rules; AT10 is not claimed"
	if view.MemberAccept > 0 && view.Generation >= 1 && r.flowCounted {
		status = StatusPass
		detail = "forward chain generation is " + itoa(view.Generation) + " with member accept rules"
	}
	r.record(Record{
		ID: "forwarding", Title: "Member forwarding follows a serving lease", Spec: []string{"§5", "§19 AT10"}, Status: status,
		Observations: []Observation{
			{Name: "nft_table_present", Detail: boolText(view.Present)},
			{Name: "forward_policy", Detail: emptyDash(view.Policy)},
			{Name: "forward_generation", Detail: itoa(view.Generation)},
			{Name: "baseline_marker", Detail: boolText(view.Baseline)},
			{Name: "member_accept_rules", Detail: itoa(view.MemberAccept)},
			{Name: "forward_excerpt", Detail: emptyDash(view.Excerpt)},
			{Name: "note", Detail: detail},
		},
	})
}

func (r *runner) member(ctx context.Context, admin *session) {
	// Sample forwarding as soon as a counted flow exists. The usage poll below
	// can outlive that chain: on run 4c97ebef the end-of-function sample saw
	// the generation-0 baseline after member-traffic had already passed.
	forwardingSampled := false
	defer func() {
		if !forwardingSampled {
			r.forwarding(r.lab.forwardChain())
		}
	}()
	rec := Record{ID: "member-traffic", Title: "Member device and exit traffic", Spec: []string{"§8", "§19 AT12"}, Status: StatusBlocked}
	finishTelemetry := r.recordTelemetry(ctx, &rec)
	defer func() {
		r.noteReadyLeaseDelay(ctx, &rec)
		finishTelemetry()
		r.record(rec)
	}()
	member, err := r.lab.login(ctx, "e2e-member", "member")
	if err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "login", Detail: "failed"})
		r.usage(ctx, nil)
		return
	}
	rec.Observations = append(rec.Observations, Observation{Name: "member_id_present", Detail: boolText(member.memberID != "")})
	portalSub := r.notePortalSubject(ctx, member.memberID, recPathNote(&rec))
	if r.planRevID == uuid.Nil {
		rec.Observations = append(rec.Observations, Observation{Name: "plan", Detail: r.planError})
		r.usage(ctx, nil)
		return
	}
	rec.Observations = append(rec.Observations, Observation{Name: "plan_id", Detail: r.planID.String()}, Observation{Name: "plan_revision_id", Detail: r.planRevID.String()})
	until := time.Now().UTC().Add(time.Hour)
	grantBody := api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "e2e acceptance grant", ValidUntil: until}
	if r.planRevID != uuid.Nil {
		rev := openapi_types.UUID(r.planRevID)
		grantBody.PlanRevisionId = &rev
	}
	grant, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/members/"+member.memberID+"/grants", grantBody, map[string]string{"Idempotency-Key": uuid.NewString()})
	rec.Observations = append(rec.Observations, Observation{Name: "grant", Detail: httpDetail(grant, err)})
	if err != nil || grant.Status < 200 || grant.Status >= 300 {
		rec.Observations = append(rec.Observations, Observation{Name: "grant_body", Detail: clipText(string(grant.Body), 180)})
	}
	r.memberNetBird(ctx, member, portalSub, recPathNote(&rec))
	r.noteProjection(ctx, member.memberID, recPathNote(&rec))
	planForCheckout := r.planID
	if planForCheckout == uuid.Nil {
		planForCheckout = uuid.New()
	}
	checkout, err := member.call(ctx, "POST", r.lab.public+"/api/v1/member/checkout-sessions", api.CheckoutSessionRequest{
		PlanId: openapi_types.UUID(planForCheckout), ReturnUrl: "https://portal.example.invalid:8443/billing/return",
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	rec.Observations = append(rec.Observations, Observation{Name: "checkout", Detail: httpDetail(checkout, err)})
	if r.usageReady {
		r.flowCounted = r.countedFlow(ctx, recPathNote(&rec))
		if r.flowCounted {
			rec.Status = StatusPass
			r.forwarding(r.lab.forwardChain())
			forwardingSampled = true
		}
	}
	r.noteAdmissionGate(ctx, admin, &rec)
	r.usage(ctx, member)
}

func (r *runner) usage(ctx context.Context, member *session) {
	rec := Record{ID: "usage", Title: "Agent samples after native admitted traffic", Spec: []string{"§10", "§19 AT14", "§19 AT15"}, Status: StatusBlocked}
	if member == nil {
		rec.Observations = append(rec.Observations, Observation{Name: "member", Detail: "no session"})
		r.record(rec)
		return
	}
	resp, err := member.call(ctx, "GET", r.lab.public+"/api/v1/member/usage", nil, nil)
	rec.Observations = append(rec.Observations, Observation{Name: "member_usage", Detail: httpDetail(resp, err)})
	if resp.Status == 501 || resp.Code == "NOT_IMPLEMENTED" {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "GET /member/usage was not served; exact bytes were not claimed"})
		r.record(rec)
		return
	}
	if !r.usageReady {
		rec.Observations = append(rec.Observations, Observation{Name: "batch", Detail: "not sent"})
		rec.Observations = append(rec.Observations, Observation{Name: "member_total_bytes", Detail: emptyDash(memberUsageTotal(resp.Body))})
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "no device binding was projected, so no usage batch was sent"})
		r.record(rec)
		return
	}
	if !r.flowCounted {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "No positive counted flow; no synthetic sample is submitted."})
		r.record(rec)
		return
	}
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var total string
	arrived := waitUntil(wait, 2*time.Second, func() bool {
		again, e := member.call(wait, "GET", r.lab.public+"/api/v1/member/usage", nil, nil)
		total = memberUsageTotal(again.Body)
		return e == nil && again.Status == 200 && total != "" && total != "0"
	})
	rec.Observations = append(rec.Observations, Observation{Name: "member_total_bytes", Detail: total}, Observation{Name: "source", Detail: "real agent sampler after controlled-origin admission; no harness usage batch"})
	if arrived {
		rec.Status = StatusPass
	}
	r.record(rec)
}

func (r *runner) releases(ctx context.Context, admin *session) (good, bad uuid.UUID) {
	rec := Record{ID: "release-register", Title: "Signed real releases and non-empty serving baseline through APIs", Spec: []string{"§13.2", "§13.3", "§13.4"}, Status: StatusFail}
	baseline, err := PrepareServingBaseline(ctx, admin.http, admin.csrf, true)
	for name, digest := range baseline.Digests {
		rec.Observations = append(rec.Observations, Observation{Name: name, Detail: digest})
	}
	if err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "error", Detail: err.Error()})
		r.record(rec)
		return
	}
	rec.Status = StatusPass
	rec.Observations = append(rec.Observations, Observation{Name: "boot_release_id", Detail: baseline.Boot.String()}, Observation{Name: "support_revision_id", Detail: baseline.SupportRevision.String()}, Observation{Name: "support_policy", Detail: "API activated; all three verified releases explicitly permitted"}, Observation{Name: "arm64", Detail: "cross-compiled and packaged; not executed on this host"})
	r.record(rec)
	return baseline.Good, baseline.Bad
}

func (r *runner) updater(ctx context.Context, good, bad uuid.UUID) {
	if good == uuid.Nil || bad == uuid.Nil || r.nodeID == uuid.Nil || r.incID == uuid.Nil {
		r.record(Record{ID: "update-good", Title: "Node update", Spec: []string{"§13", "§19 AT23", "§19 AT25"}, Status: StatusBlocked,
			Observations: []Observation{{Name: "note", Detail: "release or node identity was not ready"}}})
		r.record(Record{ID: "update-bad", Title: "Bad release rejected", Spec: []string{"§13", "§19 AT27"}, Status: StatusBlocked,
			Observations: []Observation{{Name: "note", Detail: "release or node identity was not ready"}}})
		return
	}
	r.oneUpdate(ctx, "update-good", "Good release reaches a terminal updater state", []string{"§13", "§19 AT23", "§19 AT25"}, good, false)
	artifact, err := os.ReadFile(filepath.Join(r.lab.state, "releases", "r", bad.String(), "amd64.tar.zst"))
	if err == nil && len(artifact) > 0 {
		artifact[len(artifact)/2] ^= 1 // Change one real byte, preserving artifact size.
		err = r.lab.putObject(ctx, "r/"+bad.String()+"/amd64.tar.zst", artifact)
	} else if err == nil {
		err = errors.New("empty artifact")
	}
	if err != nil {
		r.record(Record{ID: "update-bad", Title: "Bad release rejected", Spec: []string{"§13", "§19 AT27"}, Status: StatusFail,
			Observations: []Observation{{Name: "overwrite", Detail: "failed"}}})
		return
	}
	r.oneUpdate(ctx, "update-bad", "Tampered artifact fails closed", []string{"§13.4", "§19 AT25", "§19 AT27"}, bad, true)
}

func (r *runner) oneUpdate(ctx context.Context, id, title string, spec []string, releaseID uuid.UUID, wantHash bool) {
	rec := Record{ID: id, Title: title, Spec: spec, Status: StatusBlocked}
	admin := r.mustAdmin(ctx)
	if admin == nil {
		rec.Status = StatusFail
		r.record(rec)
		return
	}
	list, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/releases", nil, nil)
	var page api.ReleaseList
	if err != nil || list.Status != 200 || json.Unmarshal(list.Body, &page) != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "fleet", Detail: httpDetail(list, err)})
		r.record(rec)
		return
	}
	batch, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/update-batches", api.UpdateBatchCreateRequest{
		ReleaseId: openapi_types.UUID(releaseID), Scope: api.UpdateBatchCreateRequestScopeAllNodes,
		ExpectedGeneration: page.Fleet.Generation, SetFleetDefault: api.UpdateBatchCreateRequestSetFleetDefaultTrue,
		Reason: "e2e acceptance",
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	rec.Observations = append(rec.Observations, Observation{Name: "batch", Detail: httpDetail(batch, err)})
	if err != nil || batch.Status != 202 {
		r.record(rec)
		return
	}
	var accepted api.UpdateBatchAccepted
	_ = json.Unmarshal(batch.Body, &accepted)
	rec.Observations = append(rec.Observations, Observation{Name: "fleet_generation", Detail: accepted.FleetGeneration.String()})
	rec.Observations = append(rec.Observations, Observation{Name: "target_count", Detail: itoa(accepted.TargetCount)})
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	seen := waitUntil(wait, 2*time.Second, func() bool {
		b, err := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
		return err == nil && desiredUpdateRelease(b) == releaseID.String()
	})
	rec.Observations = append(rec.Observations, Observation{Name: "desired_contains_release", Detail: boolText(seen)})
	if !seen {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "no signed update instruction for this release within 90s; updater was not invoked"})
		conn, err := r.lab.db(ctx)
		if err == nil {
			defer conn.Close(ctx)
			var row []byte
			err = conn.QueryRow(ctx, `SELECT jsonb_build_object('state',j.state,'stage',j.stage,
				'error_code',j.error_code,'issued_at',j.issued_at,'target_state',t.target_state)
				FROM update_jobs j JOIN update_targets t ON t.id=j.target_id
				WHERE t.batch_id=$1 AND t.node_id=$2 ORDER BY j.attempt DESC LIMIT 1`, accepted.BatchId, r.nodeID).Scan(&row)
			if err == nil {
				rec.Observations = append(rec.Observations, Observation{Name: "stored_update_job", Detail: string(row)})
			}
		}
		r.record(rec)
		return
	}
	var finishProbe func() bool
	issued := false
	var stopProbeWatch func() bool
	if !wantHash {
		r.updateOwner = "w3-devstack-e2e: dedicated synthetic peer and narrow route setup"
		if !r.prepareProbe(ctx, recPathNote(&rec)) {
			recPathNote(&rec)("probe_gate", "dedicated peer/route setup failed")
			r.record(rec)
			return
		}
		r.updateOwner = "w2-agent-core / w1-netlab: challenge exception and observed probe traversal"
		finishProbe = r.startProbeCapture(ctx, recPathNote(&rec))
		// One in-node process polls the short-lived challenge. Per-poll Docker
		// exec missed it while the nft observer was also running.
		watch := exec.CommandContext(ctx, "docker", "exec", "-i", r.lab.node, "fvpn-e2e-nodeconfig", "probe-issued")
		body, err := json.Marshal(r.probePeer)
		if err != nil {
			recPathNote(&rec)("probe_gate", "configured probe could not be sent to the node watcher")
			r.record(rec)
			return
		}
		watch.Stdin = bytes.NewReader(body)
		var issuedOut bytes.Buffer
		watch.Stdout = &issuedOut
		if err = watch.Start(); err != nil {
			recPathNote(&rec)("probe_gate", "challenge watcher did not start")
			r.record(rec)
			return
		}
		stopProbeWatch = func() bool {
			_ = os.WriteFile(filepath.Join(r.lab.state, "node", "var", "e2e", "probe-issued-stop"), []byte("1\n"), 0o644)
			_ = watch.Wait()
			_ = os.Remove(filepath.Join(r.lab.state, "node", "var", "e2e", "probe-issued-stop"))
			return strings.TrimSpace(issuedOut.String()) == "true"
		}
		defer func() {
			if stopProbeWatch != nil {
				stopProbeWatch()
			}
		}()
	}
	probeOK := wantHash
	defer func() {
		if finishProbe != nil {
			finishProbe()
		}
	}()
	pause := filepath.Join(r.lab.state, "node", "var", "e2e", "pause-agent")
	if err := os.WriteFile(pause, []byte("1\n"), 0o644); err != nil {
		rec.Status = StatusFail
		r.record(rec)
		return
	}
	defer os.Remove(pause)
	time.Sleep(4 * time.Second)
	_ = os.Remove(filepath.Join(r.lab.state, "node", "var", "e2e", "updater-result.json"))
	if err := r.writeUpdaterConfig(); err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "updater_config", Detail: "write failed"})
		r.record(rec)
		return
	}
	runWait, runCancel := context.WithTimeout(ctx, 4*time.Minute)
	defer runCancel()
	finished := waitUntil(runWait, 2*time.Second, func() bool {
		_, err := os.Stat(filepath.Join(r.lab.state, "node", "var", "e2e", "updater-result.json"))
		if err == nil {
			return true
		}
		return r.lab.fileNonEmpty(r.lab.node, "/var/lib/vpn-node/e2e/updater-result.json")
	})
	_ = os.Remove(filepath.Join(r.lab.state, "node", "var", "e2e", "pause-agent"))
	if !finished {
		rec.Observations = append(rec.Observations, Observation{Name: "updater", Detail: "timed out"})
		r.record(rec)
		return
	}
	result, _ := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater-result.json")
	var exit struct {
		Code int `json:"exit_code"`
	}
	_ = json.Unmarshal(result, &exit)
	executable, _ := r.lab.docker(r.lab.node, "sh", "-c", `test -s /var/lib/vpn-node/e2e/agent.pid && readlink /proc/$(cat /var/lib/vpn-node/e2e/agent.pid)/exe`)
	candidateRunning := strings.TrimSpace(string(executable)) == "/opt/vpn-node/releases/"+releaseID.String()+"/fvpn-agent"
	rec.Observations = append(rec.Observations, Observation{Name: "running_candidate_executable", Detail: boolText(candidateRunning)})
	journal := r.readJournal(releaseID.String())
	outLog, outLogErr := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.out")
	probeOmissions := strings.Count(string(outLog), "health challenge probe omitted")
	if !wantHash {
		recPathNote(&rec)("probe_agent_config", "/etc/vpn-node/agent-run.json")
		recPathNote(&rec)("updater_log_readable", boolText(outLogErr == nil))
		recPathNote(&rec)("probe_omission_count", itoa(probeOmissions))
		recPathNote(&rec)("invalid_probe_peer", boolText(strings.Contains(string(outLog), "updater: invalid probe_peer")))
		if refusal := excerptMatching(string(outLog), []string{"health challenge probe omitted", "updater: invalid probe_peer"}, 3); refusal != "" {
			recPathNote(&rec)("probe_refusal", refusal)
		}
	}
	blob := journal.Text + "\n" + string(outLog)
	words := scanWords(blob)
	rec.Observations = append(rec.Observations, Observation{Name: "exit_code", Detail: itoa(exit.Code)})
	for _, w := range []string{"locally_succeeded", "rolled_back", "HASH_MISMATCH", "recovery_required", "ISOLATION_BASELINE_MISSING", "artifact url address"} {
		rec.Observations = append(rec.Observations, Observation{Name: w, Detail: boolText(words[w])})
	}
	rec.Observations = append(rec.Observations, Observation{Name: "health_json", Detail: emptyDash(journal.Health)})
	rec.Observations = append(rec.Observations, Observation{Name: "health_attestation", Detail: emptyDash(journal.Attestation)})
	rec.Observations = append(rec.Observations, Observation{Name: "error_code", Detail: emptyDash(journal.ErrorCode)})
	rec.Observations = append(rec.Observations, Observation{Name: "result_last_error", Detail: emptyDash(journal.LastError)})
	if excerpt := excerptMatching(blob, []string{"health_json", "error_code", "tunnel", "dns_policy", "internet_only", "HASH", "rollback", "last_error", "locally_succeeded"}, 6); excerpt != "" {
		rec.Observations = append(rec.Observations, Observation{Name: "journal_excerpt", Detail: excerpt})
	}
	if finishProbe != nil {
		if stopProbeWatch != nil {
			issued = stopProbeWatch()
			stopProbeWatch = nil
		}
		probeOK = finishProbe() && outLogErr == nil && probeOmissions == 0 && issued
		recPathNote(&rec)("configured_probe_issued", boolText(issued))
		finishProbe = nil
	}
	switch {
	case !wantHash && words["locally_succeeded"] && !issued:
		rec.Status = StatusFail
		recPathNote(&rec)("probe_gate", "terminal success without the configured probe in a fresh challenge")
	case !wantHash && !probeOK:
		recPathNote(&rec)("probe_gate", "real candidate result does not prove §13.9; see observed grant and traversal counters; owner w2-agent-core / w1-netlab")
	case wantHash && words["locally_succeeded"]:
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "tampered artifact incorrectly accepted"})
	case words["locally_succeeded"]:
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "locally_succeeded was observed"})
		rec.Status = StatusPass
	case wantHash && (words["HASH_MISMATCH"] || words["SIGNATURE_INVALID"]):
		rec.Status = StatusPass
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "tampered bytes were rejected"})
	default:
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: updaterBlockedNote(journal)})
	}
	r.record(rec)
}

func (r *runner) controlStop(ctx context.Context) {
	rec := Record{ID: "lease-expiry", Title: "Control stop leaves forwarding off", Spec: []string{"§5", "§19 AT29"}, Status: StatusBlocked}
	if !r.flowCounted || !r.countedFlow(ctx, recPathNote(&rec)) {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "No positive counted flow before fault; control stop is not injected."})
		r.record(rec)
		return
	}
	s := &Stack{lab: r.lab}
	if _, err := s.StartEstablishedTransports(ctx, r.usageDevice); err != nil {
		rec.Status = StatusFail
		recPathNote(&rec)("transport_start", err.Error())
		r.record(rec)
		return
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.StopEstablishedTransports(cleanup)
	}()
	active, _ := s.EstablishedStatus(ctx)
	recPathNote(&rec)("established_before", fmt.Sprintf("tcp=%d udp=%d running=%t", active.TCPEchoes, active.UDPReplies, active.Running))
	before := r.lab.forwardChain()
	recPathNote(&rec)("desired_before_stop", expiryView(r.desiredRaw()))
	cmd := exec.Command("docker", "stop", "-t", "20", r.lab.control)
	if err := cmd.Run(); err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "stop", Detail: "failed"})
		r.record(rec)
		return
	}
	waitExpiry, cancelExpiry := context.WithTimeout(ctx, 11*time.Minute)
	waitUntil(waitExpiry, 2*time.Second, func() bool { return r.lab.forwardChain().MemberAccept == 0 })
	cancelExpiry()
	after := r.lab.forwardChain()
	heartbeat, _ := r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
	recPathNote(&rec)("heartbeat_after_wait", strings.TrimSpace(string(heartbeat)))
	recPathNote(&rec)("wait_finished_at", time.Now().UTC().Format(time.RFC3339))
	if after.MemberAccept > 0 {
		recPathNote(&rec)("owner", "w2-agent-core: forwarding remains after 11-minute control-loss window; compare signed desired expiry and heartbeat below; member lease expiry may be later")
	}
	rec.Observations = append(rec.Observations, Observation{Name: "table_before", Detail: boolText(before.Present)})
	rec.Observations = append(rec.Observations, Observation{Name: "policy_before", Detail: emptyDash(before.Policy)})
	rec.Observations = append(rec.Observations, Observation{Name: "member_accept_before", Detail: itoa(before.MemberAccept)})
	rec.Observations = append(rec.Observations, Observation{Name: "table_after", Detail: boolText(after.Present)})
	rec.Observations = append(rec.Observations, Observation{Name: "policy_after", Detail: emptyDash(after.Policy)})
	rec.Observations = append(rec.Observations, Observation{Name: "member_accept_after", Detail: itoa(after.MemberAccept)})
	if before.MemberAccept == 0 {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "the forward chain had no member accept rule before the stop, so AT29 is not claimed"})
	} else if after.MemberAccept == 0 {
		rec.Status = StatusPass
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "member accept rules were present and then gone"})
	}
	// Keep sending on the same sockets while measuring the closed path.
	time.Sleep(3 * time.Second)
	stopped, _ := s.EstablishedStatus(ctx)
	bytesStopped, bytesErr := s.DeviceByteCounters(ctx, r.usageDevice)
	time.Sleep(3 * time.Second)
	later, statusErr := s.EstablishedStatus(ctx)
	bytesLater, laterErr := s.DeviceByteCounters(ctx, r.usageDevice)
	quiet := statusErr == nil && bytesErr == nil && laterErr == nil && bytesStopped.Present && bytesLater.Present && later.Running && later.TCPAttempts > stopped.TCPAttempts && later.UDPAttempts > stopped.UDPAttempts && later.TCPEchoes == stopped.TCPEchoes && later.UDPReplies == stopped.UDPReplies && bytesStopped == bytesLater
	recPathNote(&rec)("active_senders_after_cutoff", boolText(later.Running && later.TCPAttempts > stopped.TCPAttempts && later.UDPAttempts > stopped.UDPAttempts))
	recPathNote(&rec)("established_replies_and_bytes_stopped", boolText(quiet))
	recPathNote(&rec)("established_after", fmt.Sprintf("tcp=%d udp=%d attempts_tcp=%d attempts_udp=%d", later.TCPEchoes, later.UDPReplies, later.TCPAttempts, later.UDPAttempts))
	if !quiet || after.MemberAccept > 0 {
		rec.Status = StatusFail
		recPathNote(&rec)("owner", "w2-agent-core / node enforcement: established traffic cutoff")
	}
	start := exec.Command("docker", "start", r.lab.control)
	if err := start.Run(); err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "start", Detail: "failed"})
		r.record(rec)
		return
	}
	wait, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	client := &http.Client{Transport: r.lab.transport, Timeout: 5 * time.Second}
	up := waitUntil(wait, 2*time.Second, func() bool {
		resp, err := client.Get(r.lab.public + "/readyz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == 200
	})
	rec.Observations = append(rec.Observations, Observation{Name: "control_restarted", Detail: boolText(up)})
	if !up && rec.Status != StatusFail {
		rec.Status = StatusFail
	}
	r.record(rec)
}

func (r *runner) revoke(ctx context.Context, nodeID uuid.UUID) {
	rec := Record{ID: "revoke", Title: "Operator revoke", Spec: []string{"§5", "§19 AT13"}, Status: StatusFail}
	// Control-loss probes can outlast recent_auth_seconds. Reauthenticate
	// through the normal OIDC flow; never extend or bypass the security window.
	admin := r.mustAdmin(ctx)
	recPathNote(&rec)("fresh_admin_auth", boolText(admin != nil))
	if admin == nil {
		r.record(rec)
		return
	}
	detail, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+nodeID.String(), nil, nil)
	if err != nil || detail.Status != 200 {
		rec.Observations = append(rec.Observations, Observation{Name: "get", Detail: httpDetail(detail, err)})
		r.record(rec)
		return
	}
	etag := detail.Header.Get("ETag")
	rev, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/nodes/"+nodeID.String()+"/revoke", api.ReasonRequest{Reason: "e2e acceptance revoke"}, map[string]string{
		"Idempotency-Key": uuid.NewString(), "If-Match": etag,
	})
	rec.Observations = append(rec.Observations, Observation{Name: "revoke", Detail: httpDetail(rev, err)})
	if err != nil || rev.Status != 202 {
		r.record(rec)
		return
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var lifecycle string
	revoked := waitUntil(wait, 2*time.Second, func() bool {
		again, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+nodeID.String(), nil, nil)
		if err != nil || again.Status != 200 {
			return false
		}
		var node struct {
			Lifecycle string `json:"lifecycle_state"`
		}
		if json.Unmarshal(again.Body, &node) != nil {
			return false
		}
		lifecycle = node.Lifecycle
		return node.Lifecycle == "revoked"
	})
	rec.Observations = append(rec.Observations, Observation{Name: "lifecycle", Detail: lifecycle})
	if revoked {
		rec.Status = StatusPass
		rec.Observations = append(rec.Observations, Observation{Name: "connections", Detail: "existing-flow teardown was not separately probed"})
	}
	r.record(rec)
}

func (r *runner) mustAdmin(ctx context.Context) *session {
	s, err := r.lab.login(ctx, "e2e-admin", "admin")
	if err != nil {
		r.t.Logf("admin login: %v", err)
		return nil
	}
	return s
}

func (r *runner) writeUpdaterConfig() error {
	boot, err := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/boot-id")
	if err != nil {
		return err
	}
	bootID, err := uuid.Parse(string(trim(boot)))
	if err != nil {
		return err
	}
	cfg := map[string]any{
		"node_id": r.nodeID.String(), "incarnation_id": r.incID.String(), "boot_id": bootID.String(),
		"revoked": false,
		"platform": map[string]any{
			"os_id": "ubuntu", "os_version": "24.04", "architecture": "amd64", "updater_version": "1.0.0",
			"control_api_major": 1, "policy_schema": 1, "usage_schema": 1, "state_schema": 1,
		},
		"release_trust": "/etc/vpn-node/trust.json", "desired_trust": "/etc/vpn-node/trust.json",
		"desired_state":    "/var/lib/vpn-node/state/desired-state.json",
		"meter_checkpoint": "/var/lib/vpn-node/state/meter.checkpoint",
		"agent_config":     "/etc/vpn-node/agent-run.json",
		"stop_command":     []string{"/usr/local/bin/agent-service.sh", "stop"}, "start_command": []string{"/usr/local/bin/agent-service.sh", "start"},
		"result_url":               r.lab.agent + "/api/v1/agent/update-results",
		"allowed_artifact_origins": []string{r.lab.agent},
		"mtls_cert":                "/var/lib/vpn-node/identity/cert.pem",
		"mtls_key":                 "/var/lib/vpn-node/identity/key.pem",
		"mtls_ca":                  "/etc/vpn-node/ca.pem",
	}
	if r.probePeer != nil {
		cfg["probe_peer"] = r.probePeer
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(r.lab.state, "node", "var", "e2e", "updater.json"), append(raw, '\n'))
}

func (r *runner) netbirdCount(ctx context.Context, path string) int {
	token, err := r.lab.secret("netbird.token")
	if err != nil {
		return -1
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.lab.netbird+path, nil)
	if err != nil {
		return -1
	}
	req.Header.Set("Authorization", "Token "+token)
	client := &http.Client{Transport: r.lab.transport, Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return -1
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		return len(arr)
	}
	return -1
}

func recPathNote(rec *Record) func(string, string) {
	return func(name, detail string) {
		rec.Observations = append(rec.Observations, Observation{Name: name, Detail: detail})
	}
}

func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func (r *runner) readIncarnation() {
	if r.incID != uuid.Nil {
		return
	}
	raw, err := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	if err != nil {
		return
	}
	var nf struct {
		Incarnation string `json:"node_incarnation_id"`
	}
	if json.Unmarshal(raw, &nf) == nil {
		r.incID, _ = uuid.Parse(nf.Incarnation)
	}
}

func (r *runner) noteSignal(note func(string, string)) {
	note("signal_before", "first run client: DeadlineExceeded dialing netbird.example.invalid with no port; failed to connect to the signalling server: dial context deadline exceeded")
	blob, _ := r.lab.docker(r.lab.node, "sh", "-c", "cat /var/lib/vpn-node/e2e/netbird-up.log /var/lib/vpn-node/e2e/netbird-service.log 2>/dev/null || true")
	note("signal_after", emptyDash(excerptMatching(string(blob), []string{"signal", "deadline", "error", "connected", "management"}, 8)))
}

func (l *lab) rememberPlan(plan, rev uuid.UUID) error {
	path := filepath.Join(l.state, "stack.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return errors.New("stack identity")
	}
	doc["plan_id"] = plan.String()
	doc["plan_revision_id"] = rev.String()
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

type nbGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// pickAutoGroup returns the unpublished-nodes group. NetBird's first group
// is All, and a setup key that names All is rejected.
func pickAutoGroup(groups []nbGroup) string {
	for _, g := range groups {
		if g.ID != "" && g.Name == "fvpn-unpublished-nodes" {
			return g.ID
		}
	}
	return ""
}

type journalView struct {
	Text        string
	Health      string
	ErrorCode   string
	LastError   string
	Attestation string
}

func (r *runner) readJournal(releaseID string) journalView {
	var view journalView
	tmp, err := os.CreateTemp("", "fvpn-e2e-journal-*.db")
	if err != nil {
		return view
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)
	defer os.Remove(path + "-wal")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if exec.CommandContext(ctx, "docker", "cp", r.lab.node+":/var/lib/vpn-node/update-journal/journal.db", path).Run() != nil {
		return view
	}
	_ = exec.CommandContext(ctx, "docker", "cp", r.lab.node+":/var/lib/vpn-node/update-journal/journal.db-wal", path+"-wal").Run()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(2000)")
	if err != nil {
		return view
	}
	defer db.Close()
	var stage, code, errorStage, detail, health string
	err = db.QueryRowContext(ctx, `SELECT stage, COALESCE(error_code,''), COALESCE(error_stage,''), COALESCE(error_detail,''), COALESCE(health_json,'') FROM txns WHERE release_id = ? ORDER BY updated_at DESC LIMIT 1`, releaseID).Scan(&stage, &code, &errorStage, &detail, &health)
	if err != nil {
		return view
	}
	view.Health = health
	view.ErrorCode = code
	var b strings.Builder
	fmtLine := func(s string) { b.WriteString(s); b.WriteByte('\n') }
	fmtLine("stage=" + stage + " error_code=" + code + " error_stage=" + errorStage + " error_detail=" + detail)
	fmtLine("health_json " + health)
	rows, err := db.QueryContext(ctx, `SELECT e.seq, e.stage, COALESCE(e.detail,'') FROM txn_events e JOIN txns t ON t.txn_id = e.txn_id WHERE t.release_id = ? ORDER BY e.seq`, releaseID)
	if err == nil {
		for rows.Next() {
			var seq int
			var evStage, evDetail string
			if rows.Scan(&seq, &evStage, &evDetail) == nil {
				fmtLine("event seq=" + itoa(seq) + " stage=" + evStage + " detail=" + evDetail)
				var detail map[string]json.RawMessage
				if json.Unmarshal([]byte(evDetail), &detail) == nil && string(detail["kind"]) == `"health_attest"` {
					safe := map[string]json.RawMessage{}
					for _, key := range []string{"kind", "phase", "reason", "internet_only_reason", "dns_policy_reason", "tunnel_probe_reason", "agent_internet_only", "agent_dns_policy", "agent_tunnel_probe", "agent_version", "netbird_version", "published"} {
						if v, ok := detail[key]; ok {
							safe[key] = v
						}
					}
					b, _ := json.Marshal(safe)
					view.Attestation = string(b)
				}
			}
		}
		rows.Close()
	}
	var state, status, last string
	if db.QueryRowContext(ctx, `SELECT state, status, COALESCE(last_error,'') FROM result_outbox ORDER BY seq DESC LIMIT 1`).Scan(&state, &status, &last) == nil {
		view.LastError = last
		fmtLine("outbox state=" + state + " status=" + status + " last_error=" + last)
	}
	view.Text = b.String()
	return view
}

func updaterBlockedNote(v journalView) string {
	if v.Health == "" && v.ErrorCode == "" {
		return "terminal state did not match a claimed success"
	}
	note := "journal health_json " + emptyDash(v.Health) + "; error_code " + emptyDash(v.ErrorCode) + "; result last_error " + emptyDash(v.LastError)
	if strings.Contains(v.Health, `"internet_only":"failed"`) && strings.Contains(v.Health, `"dns_policy":"failed"`) && strings.Contains(v.Health, `"tunnel_probe":"failed"`) {
		note += ". D-053 requires a live candidate agent attestation; missing or failed checks remain fail closed. Owners: w2-agent-core and w4-node-updater. locally_succeeded was not reached"
	}
	if v.ErrorCode == "ROLLBACK_FAILED" {
		note += ". ROLLBACK_FAILED is the rollback decision when no last-good release can be restored"
	}
	return note
}

func memberUsageTotal(raw []byte) string {
	var doc struct {
		Totals struct {
			Total string `json:"total_bytes"`
		} `json:"totals"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	return doc.Totals.Total
}

func excerptMatching(blob string, needles []string, maxLines int) string {
	var lines []string
	for _, line := range strings.Split(blob, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.Contains(trim, "/ho"+"me/") || strings.Contains(strings.ToLower(trim), "setup key") {
			continue
		}
		low := strings.ToLower(trim)
		for _, needle := range needles {
			if strings.Contains(low, strings.ToLower(needle)) {
				if len(trim) > 180 {
					trim = trim[:180]
				}
				lines = append(lines, trim)
				break
			}
		}
		if len(lines) >= maxLines {
			break
		}
	}
	return strings.Join(lines, " | ")
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func trim(b []byte) string {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return string(b)
}
