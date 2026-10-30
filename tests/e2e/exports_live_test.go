//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
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
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	r := &runner{t: t, lab: l, runID: uuid.NewString(), sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs/2026-10-30/followup12", strings.TrimPrefix(l.prefix, "fvpn-"))}
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
	// Give both real agents multiple poll/heartbeat intervals to expose conflicts.
	window, c := context.WithTimeout(ctx, 65*time.Second)
	defer c()
	_ = waitUntil(window, 2*time.Second, func() bool { obs, err = fixture.Observe(window); return err == nil && obs.MappingFrozen })
	product := Record{ID: "clone-conflict", Title: "Cloned identity conflict and automatic-mapping freeze", Spec: []string{"§6.2", "AT07"}, Status: StatusBlocked}
	raw, _ = json.Marshal(obs)
	recPathNote(&product)("product_readback", string(raw))
	if obs.MappingFrozen && obs.Conflicts > 0 {
		product.Status = StatusPass
	} else {
		recPathNote(&product)("owner", "PKI/enrollment + node identity adapter: ObserveEndpoint has no production caller; actual endpoint/conflict rows and freeze above")
	}
	r.record(product)
}
