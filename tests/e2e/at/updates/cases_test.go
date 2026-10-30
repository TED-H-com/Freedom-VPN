//go:build e2e

package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestUpdates runs AT23–AT27 and the first-install ROLLBACK_FAILED finding
// against one dev stack (§19). Blocked product results are recorded and do
// not fail the test. A missing stack, login, enrollment, release registration,
// idempotency conflict or snapshot membership does.
func TestUpdates(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("set FVPN_E2E=1 against a running dev stack")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()
	lb, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	s := &suite{t: t, ctx: ctx, lab: lb, rel: map[string]builtRelease{}}
	s.init()
	defer s.finish()
	defer func() {
		if s.rec["at23-arm64"].Actual == "" {
			s.arm64()
		}
	}()
	s.prepare()
	if s.broken != "" {
		return
	}
	s.idempotency()
	if s.broken != "" {
		return
	}
	s.population()
	if s.broken == "" {
		s.fleetBatch()
	}
	// The first-install journal is the required finding even when a later
	// population or snapshot check cannot continue (§13.9, §13.11).
	if s.primary.NodeID != uuid.Nil {
		s.finding()
	}
	if s.broken != "" {
		return
	}
	s.catchUp()
	s.staleAck()
	s.faults()
	s.versions()
}

type suite struct {
	t        *testing.T
	ctx      context.Context
	lab      *lab
	rel      map[string]builtRelease
	rec      map[string]*Record
	broken   string
	miss     bool
	git      string
	images   string
	policy   string
	inc      string
	primary  nodeIdentity
	varDir   string
	held     string
	heldID   uuid.UUID
	aux      []auxNode
	batch    api.UpdateBatchAccepted
	disabled auxNode
	offline  auxNode
	late     auxNode
	enroll   auxNode
}

func (s *suite) init() {
	s.varDir = filepath.Join(s.lab.state, "node", "var")
	s.git = gitHead(s.lab.root)
	s.images = s.lab.imageDigests()
	s.rec = map[string]*Record{
		"update-good-finding": {
			ID: "update-good-finding", Title: "First install ends recovery_required / ROLLBACK_FAILED",
			Spec: []string{"§13.9", "§13.11", "§19 AT23"}, Status: StatusBlocked,
			Expected: "A first install with no last-good release records ROLLBACK_FAILED after health checks fail, without a symlink restore.",
		},
		"at23": {
			ID: "at23", Title: "One operation updates every architecture and source",
			Spec: []string{"§13.1", "§13.10", "§19 AT23", "UPD01", "UPD02", "NODE01"}, Status: StatusBlocked,
			Expected: "One all_nodes batch snapshots every non-revoked node. Success requires a real version and health confirmation. No per-node SSH.",
		},
		"at24": {
			ID: "at24", Title: "Offline catch-up and inheritance",
			Spec: []string{"§13.1", "§19 AT24", "UPD01", "UPD02"}, Status: StatusBlocked,
			Expected: "Offline and enrolling nodes stay in the original snapshot and run after authentication. A node created after the batch takes the fleet default and is not backfilled.",
		},
		"at25": {
			ID: "at25", Title: "Bad package, full disk, failed switch, recovery",
			Spec: []string{"§13.4", "§13.7", "§13.9", "§19 AT25", "UPD02", "UPD03", "UPD05"}, Status: StatusBlocked,
			Expected: "Unverified content is not installed. A failed download leaves the previous tree. Identity stays. Rollback to a serving last-good is blocked while health cannot pass.",
		},
		"at26": {
			ID: "at26", Title: "Idempotency, races and stale ACK",
			Spec: []string{"§13.8", "§14.5", "§19 AT26", "UPD02", "API01"}, Status: StatusBlocked,
			Expected: "One batch per Idempotency-Key. A different body is 409. Unstarted work is superseded. A stale ACK is history and does not overwrite current. A completed safe switch is blocked on health.",
		},
		"at27": {
			ID: "at27", Title: "Version targets, minimum version, disabled stays disabled",
			Spec: []string{"§5", "§13.2", "§19 AT27", "UPD03", "UPD04", "UPD05", "EXIT03"}, Status: StatusBlocked,
			Expected: "desired != current does not by itself open forwarding. A release outside the allowlist is not a new target. Disabled stays disabled across reboot. Safe last-good restore of forwarding needs a serving node.",
		},
		"at23-arm64": {
			ID: "at23-arm64", Title: "arm64 artifact and operator hardware",
			Spec: []string{"§3.2", "§13.4", "§19 AT23", "NODE01"}, Status: StatusNeedsOperatorRun,
			Expected: "arm64 is packaged and manifest-listed. A physical arm64 node and the three join sources need an operator.",
		},
	}
}

func (s *suite) finish() {
	s.lab.removeAux(s.names()...)
	_ = exec.Command("docker", "rm", "-f", s.lab.prefix+"-armprobe").Run()
	dir := filepath.Join(s.lab.root, "docs", "e2e", "at", "updates")
	for _, rec := range s.rec {
		rec.Git = s.git
		rec.Images = s.images
		rec.PolicyRevision = s.policy
		rec.NodeIncarnation = s.inc
		if err := Write(dir, *rec); err != nil {
			s.t.Errorf("write %s: %v", rec.ID, err)
		}
	}
	if s.broken != "" {
		s.t.Errorf("harness: %s", s.broken)
	}
	if s.miss {
		s.t.Errorf("first-install ROLLBACK_FAILED was not reproduced")
	}
}

func (s *suite) names() []string {
	out := make([]string, 0, len(s.aux))
	for _, a := range s.aux {
		out = append(out, a.container)
	}
	return out
}

func (s *suite) logf(format string, args ...any) {
	s.t.Logf("%s "+format, append([]any{time.Now().UTC().Format(time.RFC3339)}, args...)...)
}

func (s *suite) breakf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.logf("harness %s", msg)
	if s.broken == "" {
		s.broken = msg
	}
}

func (s *suite) obs(id, name, detail string) {
	s.rec[id].Observations = append(s.rec[id].Observations, Observation{Name: name, Detail: detail})
}

func (s *suite) cmd(id, command string) {
	s.rec[id].Commands = append(s.rec[id].Commands, command)
}

func (s *suite) admin() *session {
	sess, err := s.lab.login(s.ctx, "e2e-admin", "admin")
	if err != nil {
		s.breakf("admin login: %v", err)
		return nil
	}
	return sess
}

func (s *suite) prepare() {
	s.logf("prepare")
	s.cmd("at23", "GET /readyz")
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, s.lab.public+"/readyz", nil)
	if err != nil {
		s.breakf("readyz request")
		return
	}
	resp, err := (&http.Client{Transport: s.lab.tr, Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		s.breakf("readyz: %v", err)
		return
	}
	resp.Body.Close()
	s.obs("at23", "readyz", fmt.Sprintf("%d", resp.StatusCode))
	if resp.StatusCode != 200 {
		s.breakf("readyz http %d", resp.StatusCode)
		return
	}
	if _, err := s.lab.db(s.ctx); err != nil {
		s.breakf("postgres")
		return
	}
	admin := s.admin()
	if admin == nil {
		return
	}
	if err := s.lab.insertReleaseSupport(s.ctx); err != nil {
		s.breakf("release support: %v", err)
		return
	}
	s.cmd("at23", "POST /api/v1/admin/enrollments")
	id, token, err := s.issue(admin, "at-primary", 7200)
	if err != nil {
		s.breakf("enroll primary: %v", err)
		return
	}
	if err := s.lab.writeToken(s.varDir, token); err != nil {
		s.breakf("token file")
		return
	}
	wait, cancel := context.WithTimeout(s.ctx, 4*time.Minute)
	ok := waitUntil(wait, time.Second, func() bool {
		b, err := s.lab.cat(s.lab.node, "/var/lib/vpn-node/e2e/status.json")
		return err == nil && strings.Contains(string(b), `"enrolled"`)
	})
	cancel()
	if !ok {
		s.breakf("primary did not enroll")
		return
	}
	ident, err := s.lab.readNodeIdentity(s.lab.node)
	if err != nil {
		s.breakf("identity: %v", err)
		return
	}
	s.primary = ident
	s.inc = ident.Incarnation.String()
	s.obs("at23", "primary_node", id.String())
	s.obs("at23", "primary_incarnation", ident.Incarnation.String())
	if err := s.lab.pause(s.varDir); err != nil {
		s.breakf("pause")
		return
	}
	time.Sleep(4 * time.Second)
	specs := []releaseSpec{
		{name: "old", product: "1.1.0", sequence: "11", commit: strings.Repeat("a", 40), build: "at-old", marker: "old\n"},
		{name: "new", product: "1.2.0", sequence: "12", commit: strings.Repeat("b", 40), build: "at-new", marker: "new\n"},
		{name: "tamper", product: "1.3.0", sequence: "13", commit: strings.Repeat("c", 40), build: "at-tamper", marker: "tamper\n"},
		{name: "schema", product: "1.4.0", sequence: "14", commit: strings.Repeat("d", 40), build: "at-schema", marker: "schema\n", schemaFrom: []int{2}, schemaTo: 2},
		{name: "bulky", product: "1.5.0", sequence: "15", commit: strings.Repeat("e", 40), build: "at-bulky", marker: "bulky\n", blobBytes: 80 * 1024, installed: "200000"},
	}
	var ids []uuid.UUID
	for _, spec := range specs {
		built, err := s.lab.buildRelease(s.ctx, spec)
		if err != nil {
			s.breakf("package %s: %v", spec.name, err)
			return
		}
		reg, err := s.lab.registerRelease(s.ctx, admin, built)
		s.obs("at23", "register_"+spec.name, httpDetail(reg, err))
		if err != nil || (reg.Status != 201 && reg.Status != 202) {
			s.breakf("register %s", spec.name)
			return
		}
		s.rel[spec.name] = built
		ids = append(ids, built.ID)
	}
	s.cmd("at23", "POST /api/v1/admin/releases")
	vctx, vcancel := context.WithTimeout(s.ctx, 3*time.Minute)
	states, err := s.lab.awaitVerified(vctx, admin, ids)
	vcancel()
	for id, state := range states {
		s.obs("at23", "verify_"+id, state)
	}
	if err != nil {
		s.breakf("verify: %v", err)
		return
	}
	for _, id := range ids {
		if states[id.String()] != "verified" {
			s.breakf("release %s not verified", id)
			return
		}
	}
	if err := s.lab.allowReleases(s.ctx, ids); err != nil {
		s.breakf("allowlist: %v", err)
		return
	}
	fleet, err := s.lab.fleet(s.ctx, admin)
	if err != nil {
		s.breakf("fleet: %v", err)
		return
	}
	if fleet.ReleaseSupportRevisionId.IsNull() {
		s.policy = "unset"
	} else if rev, err := fleet.ReleaseSupportRevisionId.Get(); err == nil {
		s.policy = uuid.UUID(rev).String()
	}
	s.obs("at23", "serving_count", fmt.Sprintf("%d", len(fleet.ServingReleaseIds)))
	s.obs("at23", "amd64_sha", s.rel["new"].AMDSHA)
	s.obs("at23", "arm64_sha", s.rel["new"].ARMSHA)
	s.obs("at23-arm64", "manifest_arm64_sha", s.rel["new"].ARMSHA)
	s.obs("at23-arm64", "manifest_amd64_sha", s.rel["new"].AMDSHA)
	s.obs("at23", "signature_check", "central verification_state=verified for every registered manifest")
}

func (s *suite) issue(admin *session, name string, ttl int) (uuid.UUID, string, error) {
	body := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name, ExpiresInSeconds: &ttl}
	resp, err := admin.call(s.ctx, http.MethodPost, s.lab.public+"/api/v1/admin/enrollments", body, map[string]string{"Idempotency-Key": uuid.NewString()})
	if err != nil {
		return uuid.Nil, "", err
	}
	if resp.Status != http.StatusCreated {
		return uuid.Nil, "", fmt.Errorf("enrollment http %d %s", resp.Status, resp.Code)
	}
	var created api.EnrollmentCreated
	if err := json.Unmarshal(resp.Body, &created); err != nil || !created.SecretAvailable || created.Token.IsNull() {
		return uuid.Nil, "", fmt.Errorf("enrollment token absent")
	}
	token, _ := created.Token.Get()
	return uuid.UUID(created.Enrollment.NodeId), token, nil
}
