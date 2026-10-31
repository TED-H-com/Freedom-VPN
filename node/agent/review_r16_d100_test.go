package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// reviewIP is the ip(8) JSON SystemPlane.MemberPath reads. It does not
// implement the subset rule itself; SystemPlane does.
type reviewIP struct {
	addr, route4, route6, routeGet []byte
	fail                           bool
}

func (p *reviewIP) LookPath(name string) (string, error)   { return name, nil }
func (p *reviewIP) NftBatch(context.Context, string) error { return nil }
func (p *reviewIP) TCBatch(context.Context, string) error  { return nil }
func (p *reviewIP) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if p.fail {
		return nil, errors.New("ip failed")
	}
	key := name + " " + strings.Join(args, " ")
	switch {
	case strings.Contains(key, "addr show"):
		return p.addr, nil
	case strings.Contains(key, "-4") && strings.Contains(key, "route show"):
		return p.route4, nil
	case strings.Contains(key, "-6") && strings.Contains(key, "route show"):
		return p.route6, nil
	case strings.Contains(key, "route get"):
		return p.routeGet, nil
	default:
		return nil, errors.New("unexpected " + key)
	}
}

type reviewPathPlane struct {
	*memPlane
	sys SystemPlane
}

func (p reviewPathPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	return p.sys.MemberPath(ctx, m, expected)
}

func reviewMemberModel() enforce.Model {
	return enforce.Model{
		Datapath: enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.37")},
		Bindings: []enforce.Binding{{AllowedV4: []netip.Prefix{netip.MustParsePrefix("100.64.0.8/32")}}},
	}
}

func reviewTopology(local string) *reviewIP {
	return &reviewIP{
		addr:     []byte(fmt.Sprintf(`[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"%s","prefixlen":24}]}]`, local)),
		route4:   []byte(`[{"dst":"100.64.0.0/10","dev":"fv-fh","gateway":"192.0.2.1","scope":"global","type":"unicast"}]`),
		route6:   []byte(`[]`),
		routeGet: []byte(`[{"dst":"100.64.0.8","dev":"fv-fh","type":"unicast"}]`),
	}
}

func reviewStoredPath(t *testing.T, r *rig) MemberPath {
	t.Helper()
	raw, ok, err := r.eng.store.Meta(t.Context(), "member_path")
	if err != nil || !ok {
		t.Fatal(raw, ok, err)
	}
	var got MemberPath
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

// TestReviewR16_MemberPathDoesNotServeAReplacedAddress persists the first
// clean topology and then removes that address. Serving must stop and the
// stored expectation must stay the first capture. A failed read must not
// write a partial expectation. A superset that still contains the learned
// address does not replace the stored hashes (D-100: refresh only on a clean
// Apply of a newer generation, which this build does not do).
func TestReviewR16_MemberPathDoesNotServeAReplacedAddress(t *testing.T) {
	r := newRig(t)
	ip := reviewTopology("192.0.2.10")
	plane := reviewPathPlane{memPlane: r.plane, sys: SystemPlane{Executor: ip}}
	r.eng.plane = plane
	model := reviewMemberModel()
	if !r.eng.checkMemberPath(t.Context(), model) || r.eng.memberPath == nil {
		t.Fatal("first clean topology was not learned")
	}
	first := reviewStoredPath(t, r)
	if len(first.Addresses) != 1 || len(first.Routes) == 0 {
		t.Fatalf("learned path = %+v", first)
	}

	ip.addr = []byte(`[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.10","prefixlen":24},{"family":"inet","local":"192.0.2.11","prefixlen":24}]}]`)
	if !r.eng.checkMemberPath(t.Context(), model) {
		t.Fatal("superset that still contains the learned address was rejected")
	}
	if got := reviewStoredPath(t, r); len(got.Addresses) != len(first.Addresses) || got.Addresses[0] != first.Addresses[0] {
		t.Fatalf("TODO(review): member-path expectation changed on a same-generation re-check: %+v -> %+v", first, got)
	}

	ip.addr = []byte(`[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.11","prefixlen":24}]}]`)
	if r.eng.checkMemberPath(t.Context(), model) || !r.eng.memberPathPending || r.eng.serving {
		t.Fatalf("TODO(review): node still accepts serving after the learned member address disappeared (pending=%v serving=%v)", r.eng.memberPathPending, r.eng.serving)
	}
	if got := reviewStoredPath(t, r); got.Addresses[0] != first.Addresses[0] {
		t.Fatalf("TODO(review): failed member-path check rewrote the stored expectation: %+v", got)
	}

	ip.fail = true
	if r.eng.checkMemberPath(t.Context(), model) {
		t.Fatal("plane error was treated as a clean path")
	}
	if got := reviewStoredPath(t, r); got.Addresses[0] != first.Addresses[0] {
		t.Fatalf("TODO(review): plane error rewrote the stored expectation: %+v", got)
	}

	// The route returns with the original address. Recovery is immediate.
	ip.fail = false
	ip.addr = []byte(`[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.10","prefixlen":24}]}]`)
	if !r.eng.checkMemberPath(t.Context(), model) || r.eng.memberPathPending {
		t.Fatal("restored member route stayed in the D-096 hold")
	}
}

// TestReviewR16_ForeignFilterCodeIsNotHealthFallback is the D-093 agent map.
// A CheckError reason that is a contract ErrorCode is reported as that code.
func TestReviewR16_ForeignFilterCodeIsNotHealthFallback(t *testing.T) {
	r := newRig(t)
	r.eng.noteApply(&enforce.CheckError{Reason: enforce.ReasonTCForeignFilter})
	if r.eng.applyError == nil || r.eng.applyError.Code != api.ErrorCodeTCFOREIGNFILTER {
		t.Fatalf("TODO(review): TC_FOREIGN_FILTER was reported as %+v", r.eng.applyError)
	}
}
