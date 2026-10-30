//go:build e2e && integration

package secops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// §19 AT28/AT31: admission is observed through product projections. This run
// deliberately makes no serving, per-peer byte, or hardware certification claim.
func TestAT28_AT31_Followup6(t *testing.T) {
	phase := os.Getenv("FVPN_SECOPS_FOLLOWUP6")
	if phase != "profile" && phase != "helper" {
		t.Skip("requires fresh assigned Follow-up 6 stack")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "follow_up": 6, "phase": phase, "run_id": uuid.New(), "node_incarnation": nil, "hardware": "shared development host; not candidate hardware", "counted_traffic": false}}
	p.open()
	defer p.db.Close(context.Background())
	defer p.tr.CloseIdleConnections()
	defer p.finish()
	admin := p.login("e2e-admin", "admin")
	p.planInputs(admin)
	p.metadata["gateway_before"] = p.gatewayInventory()
	if phase == "profile" {
		members := make([]*liveSession, 20)
		loginRows := []map[string]any{}
		for i := range members {
			started := time.Now()
			members[i] = p.login(fmt.Sprintf("secops-member-%02d", i), "member")
			loginRows = append(loginRows, map[string]any{"user_index": i, "elapsed_ms": float64(time.Since(started).Microseconds()) / 1000})
			time.Sleep(time.Second)
		}
		p.metadata["login_measurements"] = loginRows
		p.grantMembers(admin, members)
		// Same sequential 20×3 profile as Follow-up 4; only observability added.
		p.scaleNative(members)
		p.cohortSnapshot(members)
		p.controlReadWindow(members)
	} else {
		stack, e := labstack.Open()
		if e != nil {
			t.Fatal("helper open")
		}
		defer stack.Close()
		stop := make(chan struct{})
		scans := make(chan map[string]labstack.NativeLogScan, 1)
		go p.watchTransient(stack, stop, scans)
		started := time.Now()
		result, e := stack.ScaleNativeMembers(t.Context(), labstack.BrowserSession{Client: admin.client, CSRF: admin.csrf}, p.planRevision, "secops6", 20, 3, func(ctx context.Context) (labstack.BrowserSession, error) {
			return stack.Login(ctx, "e2e-admin", "admin")
		})
		close(stop)
		p.metadata["helper_transient_observations"] = <-scans
		p.metadata["native_helper"] = map[string]any{"hook": "ScaleNativeMembers", "counts": result, "elapsed_seconds": time.Since(started).Seconds(), "error": e != nil, "expected_users": 20, "expected_peers": 60, "status": map[bool]string{true: "passed", false: "blocked"}[e == nil && result.Users == 20 && result.Authorized == 60 && result.SimultaneousAuthorized == 60 && result.IdentityMatches == 60 && result.ScannedLogs == 60 && result.ProtectedLogs == 60]}
		if e != nil {
			t.Log("native helper did not complete; retain actual counts")
			return
		}
		// The public helper returns aggregate admission counts, not individual
		// latency samples or pre-delete key counts; independent watcher is qualified.
		members := make([]*liveSession, 20)
		for i := range members {
			members[i] = p.login(fmt.Sprintf("secops6-u%02d", i+1), "member")
			time.Sleep(time.Second)
		}
		p.cohortSnapshot(members)
		p.controlReadWindow(members)
		admin = p.login("e2e-admin", "admin")
		node, cert := p.enrollHost(admin)
		p.anonymousIsolation(admin, cert, node)
		devices := make([][]uuid.UUID, len(members))
		p.bindings = map[uuid.UUID]api.DeviceBinding{}
		var binding api.DeviceBinding
		var revision string
		e = p.db.QueryRow(t.Context(), `SELECT member_id,device_id,binding_revision::text,valid_from,valid_until FROM device_bindings WHERE member_id=$1 AND valid_from<=now() AND valid_until>now() ORDER BY binding_revision DESC LIMIT 1`, members[0].member).Scan(&binding.MemberId, &binding.DeviceId, &revision, &binding.ValidFrom, &binding.ValidUntil)
		if e != nil {
			t.Fatal("helper cohort current binding unavailable")
		}
		if e = json.Unmarshal([]byte(`"`+revision+`"`), &binding.BindingRevision); e != nil {
			t.Fatal("binding revision")
		}
		p.bindings[binding.DeviceId] = binding
		devices[0] = []uuid.UUID{binding.DeviceId}
		p.rateClasses(cert, node, members, devices)
		p.metadata["trusted_proxy_row"] = map[string]any{"status": "blocked", "owner": "w2-control-core follow-up 18", "reason": "not merged on measured source; no principal claim from forwarded headers"}
	}
	p.metadata["gateway_after"] = p.gatewayInventory()
}

func (p *liveProbe) cohortSnapshot(members []*liveSession) {
	ids := make([]uuid.UUID, len(members))
	for i, m := range members {
		ids[i] = uuid.MustParse(m.member)
	}
	var raw []byte
	err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('users',(SELECT count(DISTINCT member_id) FROM member_devices WHERE member_id=ANY($1::uuid[]) AND status='authorized'),'authorized',(SELECT count(*) FROM member_devices WHERE member_id=ANY($1::uuid[]) AND status='authorized'),'binding_devices',(SELECT count(DISTINCT b.device_id) FROM device_bindings b JOIN member_devices d ON d.id=b.device_id WHERE d.member_id=ANY($1::uuid[]) AND d.status='authorized' AND b.valid_from<=now() AND b.valid_until>now()))`, ids).Scan(&raw)
	if err != nil {
		p.t.Error("cohort snapshot")
		return
	}
	p.metadata["simultaneous_cohort"] = json.RawMessage(raw)
}

func (p *liveProbe) controlReadWindow(members []*liveSession) {
	// Fixed 20 readers × 30 device-list reads, one per second per identity.
	expected := make([]map[uuid.UUID]bool, len(members))
	for i, m := range members {
		expected[i] = map[uuid.UUID]bool{}
		rows, err := p.db.Query(p.t.Context(), `SELECT id FROM member_devices WHERE member_id=$1 AND status='authorized'`, m.member)
		if err != nil {
			p.t.Fatal("read expected cohort ownership")
		}
		for rows.Next() {
			var id uuid.UUID
			if rows.Scan(&id) != nil {
				p.t.Fatal("read expected device")
			}
			expected[i][id] = true
		}
		if rows.Err() != nil {
			p.t.Fatal("expected device rows")
		}
		rows.Close()
	}
	rows := make([][]map[string]any, len(members))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, m := range members {
		wg.Go(func() {
			<-start
			for n := 0; n < 30; n++ {
				beg := time.Now()
				r := p.call(m, "GET", "/api/v1/member/devices", nil, nil)
				var page api.MemberDeviceList
				valid := r.status == 200 && json.Unmarshal(r.body, &page) == nil
				wrong := 0
				for _, d := range page.Items {
					if !expected[i][d.DeviceId] {
						wrong++
					}
				}
				rows[i] = append(rows[i], map[string]any{"index": n, "utc": beg.UTC(), "status": r.status, "code": r.code, "elapsed_ms": float64(time.Since(beg).Microseconds()) / 1000, "valid_page": valid, "devices": len(page.Items), "foreign_devices": wrong})
				if pause := time.Second - time.Since(beg); pause > 0 {
					time.Sleep(pause)
				}
			}
		})
	}
	close(start)
	wg.Wait()
	p.metadata["control_reads"] = map[string]any{"profile": "20 concurrent readers ×30 device-list reads at one per second each; no serving load", "rows": rows}
}

func (p *liveProbe) anonymousIsolation(admin, cert *liveSession, node api.EnrollResponse) {
	time.Sleep(6 * time.Second)
	renew := api.LeaseRenewalRequest{NodeId: node.NodeId, NodeIncarnationId: node.NodeIncarnationId, RenewalRequestId: uuid.New(), CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()}
	snap := func(index int) map[string]any {
		a := p.call(admin, "GET", "/api/v1/admin/groups", nil, nil)
		renew.RenewalRequestId = uuid.New()
		b := p.call(cert, "POST", liveAgent+"/api/v1/agent/lease-renewals", renew, map[string]string{"Idempotency-Key": ""})
		p.check("isolated admin", a, 200, "")
		p.check("isolated certified lease", b, 200, "")
		return map[string]any{"after_anonymous_requests": index, "utc": time.Now().UTC(), "admin_status": a.status, "admin_code": a.code, "lease_status": b.status, "lease_code": b.code}
	}
	rows := []map[string]any{snap(0)}
	requests := []map[string]any{}
	before := p.classMetrics()
	started := time.Now()
	anon := &liveSession{client: p.client()}
	for i := 1; i <= 120; i++ {
		r := p.call(anon, "POST", "/api/v1/webhooks/stripe", json.RawMessage(`{}`), nil)
		requests = append(requests, map[string]any{"index": i, "status": r.status, "code": r.code, "retry_after": r.header.Get("Retry-After")})
		if i%40 == 0 {
			rows = append(rows, snap(i))
		}
	}
	p.metadata["anonymous_isolation"] = map[string]any{"source": "one actual connection address; headers do not manufacture principals", "scope": "single-address 120-request anonymous saturation; not a 10000-principal IPv6 table-fill test", "elapsed_seconds": time.Since(started).Seconds(), "expected": "anonymous429 while admin200 and certified lease200", "checks": rows, "requests": requests, "metrics_before": before, "metrics_after": p.classMetrics()}
}

func (p *liveProbe) gatewayInventory() map[string]any {
	// Aggregate socket states and worker counts only, never connection addresses.
	script := `printf 'workers '; for p in /proc/[0-9]*/cmdline; do tr '\000' ' ' < "$p" 2>/dev/null; printf '\n'; done | grep -c '^nginx: worker process'
awk 'FNR>1 {states[$4]++;total++} END {print "tcp_total",total;for(s in states) print "tcp_state_"s,states[s]}' /proc/net/tcp /proc/net/tcp6
printf 'fd_entries '; for p in /proc/[0-9]*/comm; do if [ "$(cat "$p" 2>/dev/null)" = nginx ]; then d=${p%/comm}; ls "$d/fd" 2>/dev/null; fi; done | wc -l`
	raw, e := p.docker("gateway", "sh", "-c", script)
	return map[string]any{"utc": time.Now().UTC(), "read_ok": e == nil, "aggregates": strings.Fields(string(raw)), "configured_workers": 2, "configured_worker_connections": 4096, "source": "deploy/devstack/nginx.conf; process and network-namespace socket inventory, no private addresses", "command": "docker exec own gateway sh: process count, /proc/net/tcp{,6} state counts and nginx fd entries"}
}

func (p *liveProbe) watchTransient(stack *labstack.Stack, stop <-chan struct{}, result chan<- map[string]labstack.NativeLogScan) {
	records := map[string]labstack.NativeLogScan{}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			result <- records
			return
		case <-ticker.C:
			raw, e := exec.CommandContext(p.t.Context(), "docker", "ps", "--filter", "name=^fvpn-w4-secops-member-secops6-", "--format", "{{.Names}}").Output()
			if e != nil {
				continue
			}
			for _, c := range strings.Fields(string(raw)) {
				// Keep retrying an empty or absent file. Once a recognized key is seen,
				// its protected inventory is enough to disprove an all-zero scan claim.
				if records[c].DeviceFlowKeys > 0 {
					continue
				}
				scan, e := stack.ScanNativeJoinLog(p.t.Context(), c)
				if e == nil && scan.Present {
					records[c] = scan
				}
			}
		}
	}
}
