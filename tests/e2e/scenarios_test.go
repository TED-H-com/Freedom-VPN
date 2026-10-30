//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
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
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	run := &runner{t: t, lab: lab, dir: filepath.Join(lab.root, "docs", "e2e", "runs", time.Now().UTC().Format("2006-01-02"))}
	defer run.finish()

	run.health(ctx)
	admin := run.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	nodeID := run.enroll(ctx, admin)
	run.desired(ctx)
	run.publish(ctx, admin, nodeID)
	if nodeID != uuid.Nil {
		run.member(ctx, admin)
		good, bad := run.releases(ctx, admin)
		run.updater(ctx, good, bad)
		run.controlStop(ctx)
		run.revoke(ctx, admin, nodeID)
	} else {
		run.forwarding(run.lab.forwardChain())
	}
}

type runner struct {
	t              *testing.T
	lab            *lab
	dir            string
	bad            bool
	nodeID         uuid.UUID
	incID          uuid.UUID
	enrolledID     uuid.UUID
	planID         uuid.UUID
	planRevID      uuid.UUID
	usageDevice    uuid.UUID
	usageRevision  decstr.U64
	usageFrom      time.Time
	usageUntil     time.Time
	usageReady     bool
	usageSentStart time.Time
	usageSentEnd   time.Time
}

func (r *runner) record(rec Record) {
	if rec.Status != StatusPass {
		r.bad = true
	}
	if err := Write(r.dir, rec); err != nil {
		r.t.Errorf("evidence %s: %v", rec.ID, err)
	}
}

func (r *runner) finish() {
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
	if err := r.lab.insertReleaseSupport(ctx); err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "release_support", Detail: "insert failed"})
		r.record(rec)
		return nodeID
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
	rec.Observations = append(rec.Observations, Observation{Name: "update_present", Detail: boolText(env.Signed.Update != nil)})
	r.record(rec)
}

func (r *runner) forwarding(view forwardView) {
	status := StatusBlocked
	detail := "forward chain policy is " + emptyDash(view.Policy) + ", generation " + itoa(view.Generation) + ", with " + itoa(view.MemberAccept) + " member accept rules; AT10 is not claimed"
	if view.MemberAccept > 0 && view.Generation >= 1 {
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
	defer func() { r.forwarding(r.lab.forwardChain()) }()
	rec := Record{ID: "member-traffic", Title: "Member device and exit traffic", Spec: []string{"§8", "§19 AT12"}, Status: StatusBlocked}
	member, err := r.lab.login(ctx, "e2e-member", "member")
	if err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "login", Detail: "failed"})
		r.record(rec)
		r.usage(ctx, nil)
		return
	}
	rec.Observations = append(rec.Observations, Observation{Name: "member_id_present", Detail: boolText(member.memberID != "")})
	if err := r.seedPlan(ctx); err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "plan", Detail: "seed failed"})
	} else {
		rec.Observations = append(rec.Observations, Observation{Name: "plan_id", Detail: r.planID.String()})
		rec.Observations = append(rec.Observations, Observation{Name: "plan_revision_id", Detail: r.planRevID.String()})
	}
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
	r.memberNetBird(ctx, member, recPathNote(&rec))
	planForCheckout := r.planID
	if planForCheckout == uuid.Nil {
		planForCheckout = uuid.New()
	}
	checkout, err := member.call(ctx, "POST", r.lab.public+"/api/v1/member/checkout-sessions", api.CheckoutSessionRequest{
		PlanId: openapi_types.UUID(planForCheckout), ReturnUrl: "https://portal.example.invalid:8443/billing/return",
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	rec.Observations = append(rec.Observations, Observation{Name: "checkout", Detail: httpDetail(checkout, err)})
	rec.Observations = append(rec.Observations, Observation{Name: "traffic", Detail: "not claimed; no flow was counted"})
	r.record(rec)
	r.usage(ctx, member)
}

func (r *runner) usage(ctx context.Context, member *session) {
	rec := Record{ID: "usage", Title: "Sampled usage visible with exact bytes", Spec: []string{"§10", "§19 AT14", "§19 AT15"}, Status: StatusBlocked}
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
	same, firstStatus, secondStatus, accepted, outcome := r.replayUsageBatch(member.memberID)
	if !r.usageSentStart.IsZero() {
		rec.Observations = append(rec.Observations, Observation{Name: "interval_start", Detail: r.usageSentStart.UTC().Format(time.RFC3339)})
		rec.Observations = append(rec.Observations, Observation{Name: "interval_end", Detail: r.usageSentEnd.UTC().Format(time.RFC3339)})
	}
	rec.Observations = append(rec.Observations, Observation{Name: "batch_status", Detail: firstStatus})
	rec.Observations = append(rec.Observations, Observation{Name: "replay_status", Detail: secondStatus})
	rec.Observations = append(rec.Observations, Observation{Name: "replay_results_equal", Detail: boolText(same)})
	rec.Observations = append(rec.Observations, Observation{Name: "batch_outcome", Detail: emptyDash(outcome)})
	again, againErr := member.call(ctx, "GET", r.lab.public+"/api/v1/member/usage", nil, nil)
	total := memberUsageTotal(again.Body)
	rec.Observations = append(rec.Observations, Observation{Name: "member_usage_after_batch", Detail: httpDetail(again, againErr)})
	rec.Observations = append(rec.Observations, Observation{Name: "member_total_bytes", Detail: emptyDash(total)})
	exact := again.Status == 200 && total == "3000"
	rec.Observations = append(rec.Observations, Observation{Name: "exact_bytes", Detail: boolText(exact)})
	switch {
	case resp.Status == 200 && same && exact && accepted:
		rec.Status = StatusPass
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "replay returned the same results and the member total matches the uploaded bytes"})
	case firstStatus == "not-sent":
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: emptyDash(outcome)})
	case resp.Status == 200 && same:
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "member usage is served and the idempotent replay returned the same results; exact byte totals were not claimed"})
	default:
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "usage replay did not show the same results"})
	}
	r.record(rec)
}

func (r *runner) releases(ctx context.Context, admin *session) (good, bad uuid.UUID) {
	rec := Record{ID: "release-register", Title: "fvpn-release packages registered and verified", Spec: []string{"§13.3", "§13.4"}, Status: StatusFail}
	var ids []uuid.UUID
	for i, spec := range []struct{ product, seq, commit, build, byte string }{
		{"1.0.1", "1", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "e2e-good", "good\n"},
		{"1.0.2", "2", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "e2e-bad", "bad\n"},
	} {
		dir := r.t.TempDir()
		pkg := filepath.Join(dir, "tree")
		if err := os.MkdirAll(filepath.Join(pkg, "bin"), 0o755); err != nil {
			rec.Observations = append(rec.Observations, Observation{Name: "package", Detail: "mkdir"})
			r.record(rec)
			return
		}
		if err := os.WriteFile(filepath.Join(pkg, "bin", "marker"), []byte(spec.byte), 0o644); err != nil {
			rec.Observations = append(rec.Observations, Observation{Name: "package", Detail: "write"})
			r.record(rec)
			return
		}
		if err := release.PackageDir(pkg, filepath.Join(dir, "amd64.tar.zst"), time.Unix(1_700_000_000, 0).UTC()); err != nil {
			rec.Observations = append(rec.Observations, Observation{Name: "package", Detail: "amd64"})
			r.record(rec)
			return
		}
		if err := release.PackageDir(pkg, filepath.Join(dir, "arm64.tar.zst"), time.Unix(1_700_000_000, 0).UTC()); err != nil {
			rec.Observations = append(rec.Observations, Observation{Name: "package", Detail: "arm64"})
			r.record(rec)
			return
		}
		id, canonical, err := r.lab.signedRelease(dir, spec.product, spec.seq, spec.commit, spec.build)
		if err != nil {
			rec.Observations = append(rec.Observations, Observation{Name: "manifest", Detail: err.Error()})
			r.record(rec)
			return
		}
		if err := r.lab.uploadRelease(ctx, id, dir, canonical); err != nil {
			rec.Observations = append(rec.Observations, Observation{Name: "upload", Detail: err.Error()})
			r.record(rec)
			return
		}
		reg, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/releases", map[string]any{
			"manifest": json.RawMessage(canonical), "manifest_url": "https://releases.example.invalid/r/" + id.String() + "/manifest.json",
		}, map[string]string{"Idempotency-Key": uuid.NewString()})
		rec.Observations = append(rec.Observations, Observation{Name: "register_" + spec.product, Detail: httpDetail(reg, err)})
		if err != nil || (reg.Status != 202 && reg.Status != 201) {
			r.record(rec)
			return
		}
		ids = append(ids, id)
		if i == 0 {
			good = id
		} else {
			bad = id
		}
	}
	states := map[string]string{}
	wait, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ok := waitUntil(wait, 5*time.Second, func() bool {
		list, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/releases", nil, nil)
		if err != nil || list.Status != 200 {
			return false
		}
		var page api.ReleaseList
		if json.Unmarshal(list.Body, &page) != nil {
			return false
		}
		seen := 0
		for _, item := range page.Items {
			id := uuid.UUID(item.ReleaseId).String()
			for _, want := range ids {
				if id == want.String() {
					states[id] = string(item.VerificationState)
					if item.VerificationState == "verified" || item.VerificationState == "failed" {
						seen++
					}
				}
			}
		}
		return seen == len(ids)
	})
	for id, state := range states {
		rec.Observations = append(rec.Observations, Observation{Name: "verify_" + id, Detail: state})
	}
	if !ok {
		rec.Observations = append(rec.Observations, Observation{Name: "verify", Detail: "timed out"})
		r.record(rec)
		return uuid.Nil, uuid.Nil
	}
	verified := []uuid.UUID{}
	for _, id := range ids {
		if states[id.String()] == "verified" {
			verified = append(verified, id)
		}
	}
	if len(verified) != 2 {
		r.record(rec)
		return uuid.Nil, uuid.Nil
	}
	if err := r.lab.allowReleases(ctx, verified); err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "allowlist", Detail: "update failed"})
		r.record(rec)
		return uuid.Nil, uuid.Nil
	}
	rec.Status = StatusPass
	rec.Observations = append(rec.Observations, Observation{Name: "arm64", Detail: "packaged; this host does not execute it"})
	r.record(rec)
	return good, bad
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
	if err := r.lab.putObject(ctx, "r/"+bad.String()+"/amd64.tar.zst", []byte("not-the-artifact\n")); err != nil {
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
		return err == nil && bytes.Contains(b, []byte(releaseID.String()))
	})
	rec.Observations = append(rec.Observations, Observation{Name: "desired_contains_release", Detail: boolText(seen)})
	if !seen {
		r.record(rec)
		return
	}
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
	journal := r.readJournal(releaseID.String())
	outLog, _ := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/updater.out")
	blob := journal.Text + "\n" + string(outLog)
	words := scanWords(blob)
	rec.Observations = append(rec.Observations, Observation{Name: "exit_code", Detail: itoa(exit.Code)})
	for _, w := range []string{"locally_succeeded", "rolled_back", "HASH_MISMATCH", "recovery_required", "ISOLATION_BASELINE_MISSING", "artifact url address"} {
		rec.Observations = append(rec.Observations, Observation{Name: w, Detail: boolText(words[w])})
	}
	rec.Observations = append(rec.Observations, Observation{Name: "health_json", Detail: emptyDash(journal.Health)})
	rec.Observations = append(rec.Observations, Observation{Name: "error_code", Detail: emptyDash(journal.ErrorCode)})
	rec.Observations = append(rec.Observations, Observation{Name: "result_last_error", Detail: emptyDash(journal.LastError)})
	if excerpt := excerptMatching(blob, []string{"health_json", "error_code", "tunnel", "dns_policy", "internet_only", "HASH", "rollback", "last_error", "locally_succeeded"}, 6); excerpt != "" {
		rec.Observations = append(rec.Observations, Observation{Name: "journal_excerpt", Detail: excerpt})
	}
	switch {
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
	before := r.lab.forwardChain()
	cmd := exec.Command("docker", "stop", "-t", "20", r.lab.control)
	if err := cmd.Run(); err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "stop", Detail: "failed"})
		r.record(rec)
		return
	}
	time.Sleep(3 * time.Second)
	after := r.lab.forwardChain()
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

func (r *runner) revoke(ctx context.Context, admin *session, nodeID uuid.UUID) {
	rec := Record{ID: "revoke", Title: "Operator revoke", Spec: []string{"§5", "§19 AT13"}, Status: StatusFail}
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
		"meter_checkpoint": "/var/lib/vpn-node/e2e/meter.json",
		"stop_command":     []string{"/bin/true"}, "start_command": []string{"/bin/true"},
		"result_url":               r.lab.agent + "/api/v1/agent/update-results",
		"allowed_artifact_origins": []string{r.lab.agent},
		"mtls_cert":                "/var/lib/vpn-node/identity/cert.pem",
		"mtls_key":                 "/var/lib/vpn-node/identity/key.pem",
		"mtls_ca":                  "/etc/vpn-node/ca.pem",
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

func httpDetail(resp reply, err error) string {
	if err != nil {
		return "transport"
	}
	if resp.Code != "" {
		return itoa(resp.Status) + " " + resp.Code
	}
	return itoa(resp.Status)
}

func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
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

func (r *runner) seedPlan(ctx context.Context) error {
	conn, err := r.lab.db(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	planID, revID := uuid.New(), uuid.New()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO plans(id, code) VALUES($1, 'e2e-lab')`, planID); err != nil {
		return err
	}
	exitGroup := uuid.New()
	if _, err := tx.Exec(ctx, `INSERT INTO plan_revisions(id, plan_id, revision, device_limit, exit_group_ids, provider, provider_price_ref, schema_version, payload) VALUES($1,$2,1,3,$4::uuid[],'stripe','e2e-price',1,$3::jsonb)`, revID, planID, `{"display_name":{"zh-TW":"e2e lab"}}`, "{"+exitGroup.String()+"}"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE plans SET active_revision_id=$2, resource_version=resource_version+1 WHERE id=$1`, planID, revID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if err := r.lab.rememberPlan(planID, revID); err != nil {
		return err
	}
	gotPlan, gotRev, err := r.lab.savedPlan()
	if err != nil {
		return err
	}
	r.planID, r.planRevID = gotPlan, gotRev
	return nil
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

func (l *lab) savedPlan() (uuid.UUID, uuid.UUID, error) {
	raw, err := os.ReadFile(filepath.Join(l.state, "stack.json"))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	var doc struct {
		Plan string `json:"plan_id"`
		Rev  string `json:"plan_revision_id"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return uuid.Nil, uuid.Nil, errors.New("stack identity")
	}
	plan, err := uuid.Parse(doc.Plan)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	rev, err := uuid.Parse(doc.Rev)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return plan, rev, nil
}

func (r *runner) replayUsageBatch(memberID string) (same bool, firstStatus, secondStatus string, accepted bool, outcome string) {
	if r.enrolledID == uuid.Nil || r.incID == uuid.Nil {
		return false, "no-node", "no-node", false, ""
	}
	if !r.usageReady {
		return false, "not-sent", "not-sent", false, "no binding was projected"
	}
	memberUUID, err := uuid.Parse(memberID)
	if err != nil {
		return false, "no-member", "no-member", false, ""
	}
	start, end, ok := coveredInterval(r.usageFrom, r.usageUntil, time.Now())
	if !ok {
		return false, "not-sent", "not-sent", false, "binding window does not cover an interval"
	}
	r.usageSentStart, r.usageSentEnd = start, end
	ev, err := usage.WithComputedHash(api.UsageEvent{
		SchemaVersion: api.UsageEventSchemaVersionN1, NodeId: openapi_types.UUID(r.enrolledID),
		NodeIncarnationId: openapi_types.UUID(r.incID), MeterEpochId: openapi_types.UUID(uuid.New()),
		EventSeq: decstr.U64(1), IntervalStart: start, IntervalEnd: end,
		MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete,
		Records: []api.UsageRecord{{
			MemberId: openapi_types.UUID(memberUUID), DeviceId: openapi_types.UUID(r.usageDevice),
			BindingRevision: r.usageRevision, PeriodId: start.UTC().Format("2006-01"),
			UploadBytes: decstr.U64(1000), DownloadBytes: decstr.U64(2000),
		}},
	})
	if err != nil {
		return false, "hash", "hash", false, ""
	}
	raw, err := json.Marshal(api.UsageBatchRequest{
		NodeId: openapi_types.UUID(r.enrolledID), NodeIncarnationId: openapi_types.UUID(r.incID),
		LedgerEpoch: presence.NewNullNullable[openapi_types.UUID](), Events: []api.UsageEvent{ev},
	})
	if err != nil {
		return false, "encode", "encode", false, ""
	}
	if err := os.WriteFile(filepath.Join(r.lab.state, "node", "var", "e2e", "usage-batch.json"), raw, 0o600); err != nil {
		return false, "file", "file", false, ""
	}
	key := uuid.NewString()
	firstBody, firstStatus := r.postUsage(key)
	secondBody, secondStatus := r.postUsage(key)
	left, right := usageResults(firstBody), usageResults(secondBody)
	return left != "" && left == right, firstStatus, secondStatus, strings.Contains(left, `"status":"accepted"`), usageOutcome(firstBody)
}

func (r *runner) postUsage(key string) ([]byte, string) {
	out, _ := r.lab.docker(r.lab.node, "wget", "-q", "-S", "-O", "-", "--content-on-error",
		"--ca-certificate=/etc/vpn-node/ca.pem",
		"--certificate=/var/lib/vpn-node/identity/cert.pem",
		"--private-key=/var/lib/vpn-node/identity/key.pem",
		"--header=Content-Type: application/json",
		"--header=Idempotency-Key: "+key,
		"--post-file=/var/lib/vpn-node/e2e/usage-batch.json",
		r.lab.agent+"/api/v1/agent/usage-batches")
	status := "transport"
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "HTTP/") {
			status = fields[1]
		}
	}
	return out, status
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
	Text      string
	Health    string
	ErrorCode string
	LastError string
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
		note += ". Stock NodeChecks Isolation, DNSPolicy and TunnelProbe return failed with no lab probe. locally_succeeded was not reached"
	}
	if v.ErrorCode == "ROLLBACK_FAILED" {
		note += ". ROLLBACK_FAILED is the rollback decision when no last-good release can be restored"
	}
	return note
}

func usageOutcome(raw []byte) string {
	i := bytes.IndexByte(raw, '{')
	if i < 0 {
		return ""
	}
	var doc struct {
		Results []struct {
			Status string `json:"status"`
			Error  struct {
				Code   string `json:"code"`
				Fields []struct {
					Reason string `json:"reason"`
				} `json:"fields"`
			} `json:"error"`
		} `json:"results"`
	}
	if json.Unmarshal(raw[i:], &doc) != nil || len(doc.Results) == 0 {
		return ""
	}
	one := doc.Results[0]
	if one.Error.Code == "" {
		return one.Status
	}
	reason := ""
	if len(one.Error.Fields) > 0 {
		reason = one.Error.Fields[0].Reason
	}
	return strings.TrimSpace(one.Status + " " + one.Error.Code + " " + reason)
}

func usageResults(raw []byte) string {
	i := bytes.IndexByte(raw, '{')
	if i < 0 {
		return ""
	}
	var doc struct {
		Results json.RawMessage `json:"results"`
	}
	if json.Unmarshal(raw[i:], &doc) != nil || len(doc.Results) == 0 {
		return ""
	}
	var compact bytes.Buffer
	if json.Compact(&compact, doc.Results) != nil {
		return string(doc.Results)
	}
	return compact.String()
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

func clipText(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = s[:n]
	}
	return s
}

func trim(b []byte) string {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return string(b)
}
