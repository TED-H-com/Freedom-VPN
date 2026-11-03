//go:build e2e

package members

import (
	"time"

	"github.com/google/uuid"
)

// TEST ONLY: after all original revocation observations, keep the same primary
// sibling stream running across a bounded deny-only TC fault and real recovery
// command. Every loss still belongs to the original stream denominator.
func (l *live) siblingApplyFault(r *record, capture *siblingCapture) {
	if capture == nil || capture.finished || capture.sender == nil {
		r.block("Primary sibling failed-Apply row not exercised: continuous sender unavailable")
		return
	}
	ev := map[string]any{"fault_id": uuid.NewString(), "node_role": "node", "started_at": time.Now().UTC(), "test_only": true, "coverage": "requires producer and packet joins"}
	_, err := l.inside("node", "tc", "qdisc", "add", "dev", "fvmb0", "clsact")
	ev["qdisc_created"] = err == nil
	if err == nil {
		remove := func() bool { _, e := l.inside("node", "tc", "qdisc", "del", "dev", "fvmb0", "clsact"); return e == nil }
		defer remove()
		_, err = l.inside("node", "tc", "filter", "add", "dev", "fvmb0", "ingress", "protocol", "ip", "pref", "49152", "flower", "ip_proto", "udp", "dst_port", "9", "action", "drop")
		ev["filter_created"] = err == nil
		if err == nil {
			time.Sleep(18 * time.Second)
		}
		ev["fault_removed"] = remove()
		ev["removed_at"] = time.Now().UTC()
		clean := l.nodeMutation("PATCH", "", map[string]any{"capacity": map[string]any{"down_bps": 231000000, "up_bps": 231000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}})
		ev["clean_generation_command_status"] = clean.status
		ev["clean_generation_command_completed_at"] = time.Now().UTC()
		l.forwardingGate(r)
	}
	ev["finished_at"] = time.Now().UTC()
	r.jsonCheck("primary sibling failed-Apply stimulus", "retain bounded stimulus; coverage requires actual failed Apply during original continuous stream", ev, "TEST ONLY own node namespace clsact UDP/9 deny; remove only created qdisc; real capacity command, no node nft observer", err == nil)
}
