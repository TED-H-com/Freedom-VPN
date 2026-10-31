//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/granttrace"
	"github.com/google/uuid"
)

type GrantWireEvent = granttrace.Event

// SetMemberGrantsReady changes only future renewal eligibility in the dedicated
// TEST ONLY runtime prepared with FVPN_E2E_GRANT_SEAM=1. Existing leases and all
// desired/projector state stay untouched. It fails if the real seam is absent.
func (s *Stack) SetMemberGrantsReady(ctx context.Context, ready bool) (GrantWireEvent, error) {
	var e GrantWireEvent
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.prefix+"-control", "/usr/local/bin/service", "-fixture-command", strconv.FormatBool(ready)).Output()
	if err != nil {
		return e, fmt.Errorf("D-087 test runtime unavailable")
	}
	err = json.Unmarshal(raw, &e)
	return e, err
}

// MemberGrantEvidence includes original requests/responses and HTTP receipt
// times for every renewal, desired-state read and actual observation. No auth
// headers or credentials are retained. Do not combine this with transport faults.
func (s *Stack) MemberGrantEvidence(ctx context.Context) ([]GrantWireEvent, error) {
	var events []GrantWireEvent
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.prefix+"-control", "/usr/local/bin/service", "-fixture-command", "evidence").Output()
	if err != nil {
		return nil, fmt.Errorf("D-087 wire evidence unavailable")
	}
	err = json.Unmarshal(raw, &events)
	return events, err
}

// MemberGrantInvariants is read-only; excludes ordinary observation, lease and
// issuance timestamps and forwarding, which must change during the transition.
func (s *Stack) MemberGrantInvariants(ctx context.Context, node uuid.UUID) (json.RawMessage, error) {
	conn, err := s.lab.db(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	var raw []byte
	err = conn.QueryRow(ctx, `SELECT jsonb_build_object(
 'node',jsonb_build_array(n.lifecycle_state,n.provisioning_state,n.safety_status,n.desired_config_generation::text,n.applied_config_generation::text),
 'publication',jsonb_build_array(e.publication_intent,e.publication_observed),
 'frozen',EXISTS(SELECT 1 FROM node_identity_endpoints WHERE incarnation_id=n.current_incarnation_id AND mapping_frozen),
 'envelope',(SELECT to_jsonb(s) FROM desired_state_envelopes s WHERE s.node_id=n.id),
 'bindings',(SELECT jsonb_agg(to_jsonb(b) ORDER BY b.device_id) FROM device_bindings b),
 'entitlements',(SELECT jsonb_agg(to_jsonb(t) ORDER BY t.member_id) FROM entitlements t),
 'device_projection',(SELECT jsonb_agg(jsonb_build_object('id',d.id,'status',d.status,'authorized_at',d.authorized_at) ORDER BY d.id) FROM member_devices d),
 'offline_seconds',(SELECT max_offline_lease_seconds FROM fleet_config LIMIT 1))
 FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1`, node).Scan(&raw)
	return raw, err
}
