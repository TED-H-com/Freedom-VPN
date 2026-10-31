package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

type memberPathExecutor struct {
	enforce.Executor // mutations must never be called
	address          string
	routes4          string
	routes6          string
	routeGet         string
	fail             bool
}

func (x *memberPathExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if name != "ip" {
		return nil, errors.New("unexpected executable")
	}
	if x.fail {
		return nil, errors.New("raw output SECRET 203.0.113.9")
	}
	switch strings.Join(args, " ") {
	case "-j addr show dev fv-fh":
		return []byte(x.address), nil
	case "-j -4 route show table all dev fv-fh":
		return []byte(x.routes4), nil
	case "-j -6 route show table all dev fv-fh":
		return []byte(x.routes6), nil
	case "-j route get 100.64.1.1":
		return []byte(x.routeGet), nil
	default:
		return nil, errors.New("unexpected read")
	}
}
func TestD096_ReadOnlyMemberTopology(t *testing.T) {
	x := memberPathExecutor{
		address:  `[{"ifname":"fv-fh","flags":["UP"],"addr_info":[{"family":"inet","local":"192.0.2.33","prefixlen":30},{"family":"inet6","local":"2001:db8:ff::33","prefixlen":64}]}]`,
		routes4:  `[{"dst":"100.64.0.0/10","gateway":"192.0.2.34","dev":"fv-fh"}]`,
		routes6:  `[{"dst":"2001:db8:a1::/64","gateway":"2001:db8:ff::34","dev":"fv-fh"}]`,
		routeGet: `[{"dst":"100.64.1.1","dev":"fv-fh"}]`,
	}
	m := enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fv-fh"}, Bindings: []enforce.Binding{{AllowedV4: []netip.Prefix{netip.MustParsePrefix("100.64.1.1/32")}}}}
	p := SystemPlane{Executor: &x}
	expected, err := p.MemberPath(t.Context(), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(expected)
	if strings.Contains(string(raw), "192.0.2.") || strings.Contains(string(raw), "100.64.") || strings.Contains(string(raw), "2001:db8") {
		t.Fatal("raw topology persisted")
	}
	good := x
	for _, fault := range []string{"down", "ipv4", "ipv6", "route4", "route6", "wrong_interface", "linkdown", "malformed", "read_error"} {
		t.Run(fault, func(t *testing.T) {
			x = good
			switch fault {
			case "down":
				x.address = strings.Replace(x.address, `["UP"]`, `[]`, 1)
			case "ipv4":
				x.address = strings.Replace(x.address, "192.0.2.33", "192.0.2.35", 1)
			case "ipv6":
				x.address = strings.Replace(x.address, "2001:db8:ff::33", "2001:db8:ff::35", 1)
			case "route4":
				x.routes4 = `[]`
			case "route6":
				x.routes6 = `[]`
			case "wrong_interface":
				x.routeGet = strings.Replace(x.routeGet, "fv-fh", "fv-ih", 1)
			case "linkdown":
				x.routeGet = `[{"dev":"fv-fh","flags":["linkdown"]}]`
			case "malformed":
				x.routes4 = `null`
			case "read_error":
				x.fail = true
			}
			if _, err := p.MemberPath(t.Context(), m, &expected); !errors.Is(err, errMemberPath) {
				t.Fatalf("accepted %s: %v", fault, err)
			}
		})
	}
	x = good
	if _, err := p.MemberPath(t.Context(), m, &expected); err != nil {
		t.Fatal("reconciled topology rejected", err)
	}
}
