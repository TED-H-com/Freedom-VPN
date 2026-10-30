//go:build integration

package devices

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestDiscoveryDenialAuditsOncePerDecision(t *testing.T) {
	const issuer = "https://issuer.example.invalid"
	const subject = "TEST-unmapped-subject"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()
	env.addUpstreamPeer(subject, "peer-unmapped-1", "100.64.0.61")
	env.addUpstreamPeer(subject, "peer-unmapped-2", "100.64.0.62")

	count := func(action, reason string) int {
		t.Helper()
		var n int
		err := env.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action=$1 AND result=$2 AND after_diff->>'reason'=$3`, action, api.AuditEventResultDenied, reason).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	total := func() int {
		t.Helper()
		var n int
		if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if got := count("device.authorize", "identity_unmapped"); got != 2 {
		t.Fatalf("first pass device denials = %d, want 2", got)
	}
	if got := count("identity_binding.rejected", "identity_unmapped"); got != 1 {
		t.Fatalf("first pass identity rejections = %d, want 1", got)
	}
	var attributed int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='device.authorize' AND result=$1 AND resource=$2`, api.AuditEventResultDenied, "fleet_config:00000000-0000-0000-0000-000000000001").Scan(&attributed); err != nil || attributed != 2 {
		t.Fatalf("unmapped denials attributed to fleet_config = %d (%v), want 2", attributed, err)
	}
	before := total()
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if after := total(); after != before || count("device.authorize", "identity_unmapped") != 2 || count("identity_binding.rejected", "identity_unmapped") != 1 {
		t.Fatalf("second unchanged pass wrote audits: total %d->%d device=%d identity=%d", before, after, count("device.authorize", "identity_unmapped"), count("identity_binding.rejected", "identity_unmapped"))
	}

	env.upstream.mu.Lock()
	env.upstream.users[subject]["is_blocked"] = true
	env.upstream.mu.Unlock()
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if got := count("identity_binding.rejected", "blocked_user"); got != 1 {
		t.Fatalf("blocked transition audits = %d, want 1", got)
	}
	if got := count("device.authorize", "identity_unmapped"); got != 2 {
		t.Fatalf("unchanged peer reason re-audited: device denials = %d, want 2", got)
	}
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if got := count("identity_binding.rejected", "blocked_user"); got != 1 {
		t.Fatalf("repeated blocked pass audits = %d, want 1", got)
	}

	cmd := entitlement.Command{Actor: entitlement.Actor{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Role: "system"}}
	peer := Peer{DeploymentID: "test-deploy", ID: "peer-unmapped-1"}
	if err := env.devices.recordDiscoveryDenial(ctx, cmd, peer, uuid.Nil, ErrLimit); err != nil {
		t.Fatal(err)
	}
	if got := count("device.authorize", "device_limit_reached"); got != 1 {
		t.Fatalf("reason change audits = %d, want 1", got)
	}
	if err := env.devices.recordDiscoveryDenial(ctx, cmd, peer, uuid.Nil, ErrLimit); err != nil {
		t.Fatal(err)
	}
	if got := count("device.authorize", "device_limit_reached"); got != 1 {
		t.Fatalf("repeated reason change audits = %d, want 1", got)
	}

	// Admission clears the stored denial. The same reason is a new transition.
	env.upstream.mu.Lock()
	env.upstream.users[subject]["is_blocked"] = false
	env.upstream.mu.Unlock()
	member := env.loginMember(issuer, "portal", subject)
	env.grantMember(member, env.revision, time.Hour)
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if got := count("device.authorize", "identity_unmapped"); got != 2 {
		t.Fatalf("admission re-audited unmapped denials: %d", got)
	}
	// The canonical identity row cannot be deleted. A future verification
	// timestamp makes the same subject unmapped again.
	if _, err := env.pool.Exec(ctx, `UPDATE identity_bindings SET verified_at=clock_timestamp()+interval '1 day' WHERE member_id=$1`, member); err != nil {
		t.Fatal(err)
	}
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if got := count("device.authorize", "identity_unmapped"); got != 4 {
		t.Fatalf("denial after admission = %d, want 4", got)
	}
	if got := count("identity_binding.rejected", "identity_unmapped"); got != 2 {
		t.Fatalf("identity denial after admission = %d, want 2", got)
	}
}
