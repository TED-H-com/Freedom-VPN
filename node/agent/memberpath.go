package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

var errMemberPath = errors.New("member_path_unconfigured")

// MemberPath retains hashes of the working member interface's topology, not
// traffic destinations. The model specifies peer addresses and interfaces,
// but no local member-link address or next hop. Learn those from the existing
// topology before serving; never replace that expectation while recovering.
type MemberPath struct {
	Interface string   `json:"interface"`
	Addresses []string `json:"addresses"`
	Routes    []string `json:"routes"`
}

func topologyHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (p SystemPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	iface := m.Datapath.MemberIngress
	got := MemberPath{Interface: iface}
	read := func(dst any, args ...string) bool {
		b, err := p.executor().Output(ctx, "ip", args...)
		return err == nil && json.Unmarshal(b, dst) == nil
	}
	var links []struct {
		Name      string   `json:"ifname"`
		Flags     []string `json:"flags"`
		Addresses []struct {
			Family    string `json:"family"`
			Local     string `json:"local"`
			PrefixLen int    `json:"prefixlen"`
			Tentative bool   `json:"tentative"`
			DADFailed bool   `json:"dadfailed"`
		} `json:"addr_info"`
	}
	if !read(&links, "-j", "addr", "show", "dev", iface) || len(links) != 1 || links[0].Name != iface || !slices.Contains(links[0].Flags, "UP") {
		return got, errMemberPath
	}
	hasV4 := false
	for _, a := range links[0].Addresses {
		addr, err := netip.ParseAddr(a.Local)
		if err != nil || a.Tentative || a.DADFailed {
			continue
		}
		hasV4 = hasV4 || addr.Is4()
		got.Addresses = append(got.Addresses, topologyHash([]any{a.Family, addr.String(), a.PrefixLen}))
	}
	if !hasV4 {
		return got, errMemberPath
	}
	type route struct {
		Dst     string          `json:"dst"`
		Gateway string          `json:"gateway"`
		Dev     string          `json:"dev"`
		Table   json.RawMessage `json:"table"`
		Type    string          `json:"type"`
		Scope   string          `json:"scope"`
		Flags   []string        `json:"flags"`
	}
	for _, family := range []string{"-4", "-6"} {
		var routes []route
		if !read(&routes, "-j", family, "route", "show", "table", "all", "dev", iface) || routes == nil {
			return got, errMemberPath
		}
		for _, r := range routes {
			if slices.Contains(r.Flags, "linkdown") {
				return got, errMemberPath
			}
			// Runtime route flags are not part of the configured expectation.
			r.Flags = nil
			got.Routes = append(got.Routes, topologyHash([]any{family, r}))
		}
	}
	// The routing decision for every applied peer must use the member link.
	// This also rejects a default-uplink fallback when its member route vanished.
	for _, binding := range m.Bindings {
		for _, addr := range append(append([]netip.Prefix(nil), binding.AllowedV4...), binding.AllowedV6...) {
			var routes []route
			if !read(&routes, "-j", "route", "get", addr.Addr().String()) || len(routes) != 1 || routes[0].Dev != iface || (routes[0].Type != "" && routes[0].Type != "unicast") || slices.Contains(routes[0].Flags, "linkdown") {
				return got, errMemberPath
			}
		}
	}
	if expected != nil {
		if expected.Interface != iface {
			return got, errMemberPath
		}
		for _, h := range expected.Addresses {
			if !slices.Contains(got.Addresses, h) {
				return got, errMemberPath
			}
		}
		for _, h := range expected.Routes {
			if !slices.Contains(got.Routes, h) {
				return got, errMemberPath
			}
		}
	}
	return got, ctx.Err()
}

func (e *Engine) checkMemberPath(ctx context.Context, model enforce.Model) bool {
	readCtx, cancel := context.WithTimeout(ctx, e.servingReadTimeout())
	defer cancel()
	path, err := e.plane.MemberPath(readCtx, model, e.memberPath)
	if err == nil {
		err = readCtx.Err()
	}
	if err == nil && e.memberPath == nil {
		raw, _ := json.Marshal(path)
		err = e.store.SetMeta(readCtx, "member_path", string(raw))
		if err == nil {
			e.memberPath = &path
		} else {
			e.storageOK = false
		}
	}
	if err != nil {
		e.memberPathPending = true
		e.noteApply(errMemberPath)
		e.applyError.Message = "member_path_unconfigured"
		cleanup, done := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer done()
		e.withdraw(cleanup)
		e.decision = Decide(e.input())
		e.lastObs = time.Time{}
		return false
	}
	e.memberPathPending = false
	return true
}
