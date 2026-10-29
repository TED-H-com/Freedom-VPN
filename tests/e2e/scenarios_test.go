//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	if nodeID != uuid.Nil {
		run.member(ctx, admin)
		good, bad := run.releases(ctx, admin)
		run.updater(ctx, good, bad)
		run.controlStop(ctx)
		run.revoke(ctx, admin, nodeID)
	}
}

type runner struct {
	t      *testing.T
	lab    *lab
	dir    string
	bad    bool
	nodeID uuid.UUID
	incID  uuid.UUID
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

	wait, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool { return r.lab.stage(nodeContainer, "/var/lib/vpn-node/e2e/status.json") == "enrolled" }) {
		rec.Observations = append(rec.Observations, Observation{Name: "stage", Detail: r.lab.stage(nodeContainer, "/var/lib/vpn-node/e2e/status.json")})
		r.record(rec)
		return uuid.Nil
	}
	if err := r.lab.insertReleaseSupport(ctx); err != nil {
		rec.Observations = append(rec.Observations, Observation{Name: "release_support", Detail: "insert failed"})
		r.record(rec)
		return nodeID
	}
	if !waitUntil(wait, time.Second, func() bool {
		return r.lab.fileNonEmpty(nodeContainer, "/var/lib/vpn-node/identity/setup.key")
	}) {
		rec.Observations = append(rec.Observations, Observation{Name: "setup_key", Detail: "absent"})
		r.record(rec)
		return nodeID
	}
	rec.Observations = append(rec.Observations, Observation{Name: "setup_key_persisted", Detail: "true"})
	body, _ := r.lab.docker(nodeContainer, "wget", "-q", "-O", "-", "--content-on-error",
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
	if !waitUntil(wait, 2*time.Second, func() bool {
		return r.lab.stage(nodeContainer, "/var/lib/vpn-node/e2e/netbird.json") == "netbird_up"
	}) {
		rec.Observations = append(rec.Observations, Observation{Name: "netbird", Detail: r.lab.stage(nodeContainer, "/var/lib/vpn-node/e2e/netbird.json")})
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
	raw, err := r.lab.docker(nodeContainer, "cat", "/var/lib/vpn-node/identity/node.json")
	if err == nil {
		var nf struct {
			Incarnation string `json:"node_incarnation_id"`
		}
		if json.Unmarshal(raw, &nf) == nil {
			r.incID, _ = uuid.Parse(nf.Incarnation)
		}
	}
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
			b, err = r.lab.docker(nodeContainer, "cat", "/var/lib/vpn-node/state/desired-state.json")
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
		r.forwarding(false)
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
	_, accept := r.lab.nftBaseline()
	r.forwarding(accept)
}

func (r *runner) forwarding(accept bool) {
	present, _ := r.lab.nftBaseline()
	status := StatusBlocked
	detail := "lease renewal returns 503 while an idempotency key is present, so forwarding stays off"
	if accept {
		status = StatusPass
		detail = "nft table contains accept"
	}
	r.record(Record{
		ID: "forwarding", Title: "Member forwarding follows a serving lease", Spec: []string{"§5", "§19 AT10"}, Status: status,
		Observations: []Observation{
			{Name: "nft_table_present", Detail: boolText(present)},
			{Name: "forward_accept", Detail: boolText(accept)},
			{Name: "note", Detail: detail},
		},
	})
}

func (r *runner) member(ctx context.Context, admin *session) {
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
	until := time.Now().UTC().Add(time.Hour)
	grant, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/members/"+member.memberID+"/grants", api.GrantCreateRequest{
		Source: api.GrantSourceAdminGrant, Reason: "e2e acceptance grant", ValidUntil: until,
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	rec.Observations = append(rec.Observations, Observation{Name: "grant", Detail: httpDetail(grant, err)})
	r.memberNetBird(ctx, recPathNote(&rec))
	devs, err := member.call(ctx, "GET", r.lab.public+"/api/v1/member/devices", nil, nil)
	rec.Observations = append(rec.Observations, Observation{Name: "devices", Detail: httpDetail(devs, err)})
	checkout, err := member.call(ctx, "POST", r.lab.public+"/api/v1/member/checkout-sessions", api.CheckoutSessionRequest{
		PlanId: openapi_types.UUID(uuid.New()), ReturnUrl: "https://portal.example.invalid:8443/billing/return",
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
	rec.Observations = append(rec.Observations, Observation{Name: "checkout", Detail: httpDetail(checkout, err)})
	rec.Observations = append(rec.Observations, Observation{Name: "traffic", Detail: "not claimed; no authorized device path produced a flow"})
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
	if resp.Code == "NOT_IMPLEMENTED" || resp.Status == 501 {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "usage domain is not mounted; exact bytes were not claimed"})
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
		b, err := r.lab.docker(nodeContainer, "cat", "/var/lib/vpn-node/state/desired-state.json")
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
		return r.lab.fileNonEmpty(nodeContainer, "/var/lib/vpn-node/e2e/updater-result.json")
	})
	_ = os.Remove(filepath.Join(r.lab.state, "node", "var", "e2e", "pause-agent"))
	if !finished {
		rec.Observations = append(rec.Observations, Observation{Name: "updater", Detail: "timed out"})
		r.record(rec)
		return
	}
	result, _ := r.lab.docker(nodeContainer, "cat", "/var/lib/vpn-node/e2e/updater-result.json")
	var exit struct {
		Code int `json:"exit_code"`
	}
	_ = json.Unmarshal(result, &exit)
	blob, _ := r.lab.docker(nodeContainer, "sh", "-c", "cat /var/lib/vpn-node/e2e/updater.out /var/lib/vpn-node/update-journal/* 2>/dev/null || true")
	words := scanWords(string(blob))
	rec.Observations = append(rec.Observations, Observation{Name: "exit_code", Detail: itoa(exit.Code)})
	for _, w := range []string{"locally_succeeded", "rolled_back", "HASH_MISMATCH", "recovery_required", "ISOLATION_BASELINE_MISSING", "artifact url address"} {
		rec.Observations = append(rec.Observations, Observation{Name: w, Detail: boolText(words[w])})
	}
	if words["locally_succeeded"] {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "locally_succeeded was observed"})
		rec.Status = StatusPass
	} else if wantHash && (words["HASH_MISMATCH"] || words["SIGNATURE_INVALID"]) {
		rec.Status = StatusPass
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "tampered bytes were rejected"})
	} else if !wantHash && (words["rolled_back"] || words["ISOLATION_BASELINE_MISSING"] || words["recovery_required"]) {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "stock health checks fail closed, so locally_succeeded was not reached"})
	} else {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "terminal state did not match a claimed success"})
	}
	r.record(rec)
}

func (r *runner) controlStop(ctx context.Context) {
	rec := Record{ID: "lease-expiry", Title: "Control stop leaves forwarding off", Spec: []string{"§5", "§19 AT29"}, Status: StatusBlocked}
	before, acceptBefore := r.lab.nftBaseline()
	cmd := exec.Command("docker", "stop", "-t", "20", prefix+"-control")
	if err := cmd.Run(); err != nil {
		rec.Status = StatusFail
		rec.Observations = append(rec.Observations, Observation{Name: "stop", Detail: "failed"})
		r.record(rec)
		return
	}
	time.Sleep(3 * time.Second)
	after, acceptAfter := r.lab.nftBaseline()
	rec.Observations = append(rec.Observations, Observation{Name: "table_before", Detail: boolText(before)})
	rec.Observations = append(rec.Observations, Observation{Name: "accept_before", Detail: boolText(acceptBefore)})
	rec.Observations = append(rec.Observations, Observation{Name: "table_after", Detail: boolText(after)})
	rec.Observations = append(rec.Observations, Observation{Name: "accept_after", Detail: boolText(acceptAfter)})
	if !acceptBefore {
		rec.Observations = append(rec.Observations, Observation{Name: "note", Detail: "forwarding was never enabled, so lease expiry could not be shown turning it off"})
	} else if !acceptAfter {
		rec.Status = StatusPass
	}
	start := exec.Command("docker", "start", prefix+"-control")
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
	boot, err := r.lab.docker(nodeContainer, "cat", "/etc/vpn-node/boot-id")
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
		"result_url": r.lab.agent + "/api/v1/agent/update-results",
		"mtls_cert":  "/var/lib/vpn-node/identity/cert.pem",
		"mtls_key":   "/var/lib/vpn-node/identity/key.pem",
		"mtls_ca":    "/etc/vpn-node/ca.pem",
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

func (r *runner) memberNetBird(ctx context.Context, note func(name, detail string)) {
	token, err := r.lab.secret("netbird.token")
	if err != nil {
		note("member_netbird", "token unavailable")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.lab.netbird+"/api/groups", nil)
	if err != nil {
		note("member_netbird", "groups request")
		return
	}
	req.Header.Set("Authorization", "Token "+token)
	client := &http.Client{Transport: r.lab.transport, Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		note("member_netbird", "groups unreachable")
		return
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	var groups []struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &groups)
	var group string
	if len(groups) > 0 {
		group = groups[0].ID
	}
	body := map[string]any{"name": "e2e-member", "type": "one-off", "expires_in": 3600, "usage_limit": 1, "auto_groups": []string{}}
	if group != "" {
		body["auto_groups"] = []string{group}
	}
	rawBody, _ := json.Marshal(body)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, r.lab.netbird+"/api/setup-keys", bytes.NewReader(rawBody))
	if err != nil {
		note("member_netbird", "setup key request")
		return
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		note("member_netbird", "setup key unreachable")
		return
	}
	raw, _ = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		note("member_netbird", "setup key HTTP "+itoa(resp.StatusCode))
		return
	}
	var created struct {
		Key string `json:"key"`
	}
	if json.Unmarshal(raw, &created) != nil || created.Key == "" {
		note("member_netbird", "setup key absent")
		return
	}
	path := filepath.Join(r.lab.state, "member", "var", "e2e", "setup.key")
	if writeAtomic(path, []byte(created.Key+"\n")) != nil {
		note("member_netbird", "setup key file")
		return
	}
	created.Key = ""
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	up := waitUntil(wait, 2*time.Second, func() bool {
		return r.lab.stage(memberContainer, "/var/lib/vpn-node/e2e/status.json") == "netbird_up"
	})
	note("member_netbird", r.lab.stage(memberContainer, "/var/lib/vpn-node/e2e/status.json"))
	if !up {
		note("member_stage", "not up")
	}
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

func trim(b []byte) string {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return string(b)
}
