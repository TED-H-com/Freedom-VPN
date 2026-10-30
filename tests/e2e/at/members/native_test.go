//go:build e2e

package members

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
)

type nativePeer struct {
	ID   string `json:"id"`
	User string `json:"user_id"`
}

func (l *live) peers() []nativePeer {
	got := l.nb("GET", "/peers", nil)
	var peers []nativePeer
	_ = json.Unmarshal(got.body, &peers)
	return peers
}

// §19 AT01/AT02: reuse the lab image and its user-auth entrypoint unchanged.
// Extra clients need only the same NET_ADMIN/NET_RAW and tun capabilities as
// compose's member. No host ports, new images or identity projection fixtures.
func (l *live) native(r *record, b *browser, service string) string {
	before := map[string]bool{}
	for _, p := range l.peers() {
		before[p.ID] = true
	}
	if service != "member" {
		container, err := l.hook.StartNativePeer(l.ctx, strings.TrimPrefix(service, "member-"))
		if err != nil {
			r.block("Shared StartNativePeer hook failed: " + service)
			return ""
		}
		l.t.Cleanup(func() { l.cleanupDocker("rm", "-fv", container); l.cleanupDocker("volume", "rm", container) })
	}
	if _, err := l.inside(service, "touch", "/var/lib/vpn-node/e2e/user-auth"); err != nil {
		r.block("Native user-auth trigger unavailable: " + service)
		return ""
	}
	defer func() { _, _ = l.inside(service, "rm", "-f", "/var/lib/vpn-node/e2e/netbird-up.log") }()
	var flow string
	found := l.wait(45*time.Second, func() bool {
		raw, _ := l.inside(service, "cat", "/var/lib/vpn-node/e2e/netbird-up.log")
		for _, word := range strings.Fields(string(raw)) {
			u, err := url.Parse(strings.Trim(word, "\"'<>"))
			if err == nil && u.Scheme == "https" && u.Hostname() == "netbird.example.invalid" && strings.HasPrefix(u.Path, "/oauth2/device") {
				flow = u.String()
				return true
			}
		}
		return false
	})
	if !found {
		r.block("Native CLI did not expose the shared issuer device flow: " + service)
		return ""
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Transport: l.transport, Jar: jar, Timeout: 20 * time.Second, CheckRedirect: idplogin.RedirectLimit(20)}
	status, err := idplogin.Walk(l.ctx, client, flow, b.cred.Email, b.cred.Password, idplogin.DeviceDone)
	r.check(service+" shared issuer login", "shared helper completes native device flow", fmt.Sprintf("HTTP=%d; completed=%t", status, err == nil), "touch user-auth; idplogin.Walk(Dex device URL, DeviceDone)", err == nil)
	if err != nil {
		r.block("Shared issuer device login incomplete: " + service)
		return ""
	}
	var peer string
	joined := l.wait(60*time.Second, func() bool {
		for _, p := range l.peers() {
			if !before[p.ID] && p.User == b.cred.ID {
				peer = p.ID
				return true
			}
		}
		return false
	})
	r.check(service+" native peer", "new real peer belongs to the same public subject", fmt.Sprintf("new_peer=%t; same_subject=%t", joined, peer != ""), "GET NetBird /api/peers; compare user_id to shared-login credential ID in memory", joined)
	return peer
}
func (l *live) awaitDevice(peer string) uuid.UUID {
	var id uuid.UUID
	if peer == "" {
		return id
	}
	_ = l.wait(90*time.Second, func() bool {
		err := l.pool.QueryRow(l.ctx, `SELECT id FROM member_devices WHERE deployment_id='e2e' AND peer_id=$1 AND status='authorized'`, peer).Scan(&id)
		return err == nil
	})
	return id
}
func (l *live) admission() (*devices.Service, error) {
	bindings := devices.NetBirdBindings{Pool: l.pool}
	client, err := netbird.New(netbird.Config{BaseURL: "https://netbird.example.invalid/api", DeploymentID: "e2e", Token: netbird.TokenFile(filepath.Join(l.state, "secrets/netbird.token")), Bindings: bindings, HTTPClient: &http.Client{Transport: l.transport, Timeout: 20 * time.Second}})
	if err != nil {
		return nil, err
	}
	identity, err := devices.NewIdentityDiscovery("https://netbird.example.invalid/oauth2", nil, nil)
	if err != nil {
		return nil, err
	}
	store, err := entitlement.NewStore(entitlement.Config{MaxGrantDuration: 24 * time.Hour, DefaultDeviceLimit: 3})
	if err != nil {
		return nil, err
	}
	return &devices.Service{Pool: l.pool, Entitlements: store, Adapter: &devices.NetBirdAdapter{Pool: l.pool, Client: client, Bindings: bindings, DeploymentID: "e2e", OverlayScope: "e2e-overlay", Identity: identity}}, nil
}
func admissionCommand() entitlement.Command {
	return entitlement.Command{Actor: entitlement.Actor{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}
}
func (l *live) devices(r *record) {
	defer l.usageTrace(r)
	first := l.awaitDevice(l.nativePeer)
	if first == uuid.Nil {
		r.block("Worker devices.discover has not projected the first native peer; cannot establish quota race. Inspect shared issuer proof, discovery outbox and member entitlement.")
		return
	}
	secondPeer := l.native(r, l.a, "member-a2")
	second := l.awaitDevice(secondPeer)
	if second == uuid.Nil {
		r.block("Second native device did not project; final-slot precondition unavailable.")
		return
	}
	// B is independently entitled. Its real existing device must be protected.
	g := l.grant(l.b, time.Now().UTC().Add(time.Hour))
	r.check("B entitlement", "finite grant accepted", detail(g), "POST B grant", g.status == 202)
	bp := l.native(r, l.b, "member-b1")
	bd := l.awaitDevice(bp)
	if bd == uuid.Nil {
		r.block("B native device did not project; existing-device ownership test unavailable.")
		return
	}
	list := l.api(l.a, "GET", "/api/v1/member/devices", nil, nil)
	denied := l.api(l.a, "DELETE", "/api/v1/member/devices/"+bd.String(), nil, commandHeaders())
	leaked := strings.Contains(string(list.body), bd.String()) || strings.Contains(string(denied.body), bd.String()) || strings.Contains(string(denied.body), l.b.member)
	r.check("cross-owner existing device delete/list", "403/404; B device absent from A list and unchanged", fmt.Sprintf("list_HTTP=%d; delete_HTTP=%d; field_leak=%t; B_authorized=%d", list.status, denied.status, leaked, l.count(`SELECT count(*) FROM member_devices WHERE id=$1 AND status='authorized'`, bd)), "GET A /member/devices; DELETE B existing UUID as A; read projection", list.status == 200 && (denied.status == 403 || denied.status == 404) && !leaked && l.count(`SELECT count(*) FROM member_devices WHERE id=$1 AND status='authorized'`, bd) == 1)
	svc, err := l.admission()
	if err != nil {
		r.block("Production admission service configuration unavailable")
		return
	}
	// The contract has no single-device GET; exercise its production owner gate
	// directly as well as the HTTP list/delete surfaces, without inventing a route.
	foreign, readErr := svc.Get(l.ctx, entitlement.Actor{ID: uuid.MustParse(l.a.member), Role: "member"}, bd)
	r.check("cross-owner service read", "not found and no returned fields", fmt.Sprintf("not_found=%t; empty_device=%t", errors.Is(readErr, db.ErrNotFound), foreign.ID == uuid.Nil), "devices.Service.Get(A, B existing device)", errors.Is(readErr, db.ErrNotFound) && foreign.ID == uuid.Nil)
	if _, err = l.docker("pause", l.name("worker")); err != nil {
		r.block("Cannot pause own discovery worker for exact race barrier")
		return
	}
	defer func() { _, _ = l.docker("unpause", l.name("worker")) }()
	contenders := []string{l.native(r, l.a, "member-a3"), l.native(r, l.a, "member-a4")}
	if contenders[0] == "" || contenders[1] == "" {
		r.block("Two distinct native contenders unavailable")
		return
	}
	before := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member)
	r.check("final-slot precondition", "two authorized of limit three", fmt.Sprintf("authorized=%d", before), "pause worker after A1/A2 projection; native-join A3/A4; read count", before == 2)
	if before != 2 {
		return
	}
	type outcome struct {
		peer   string
		change devices.Change
		err    error
	}
	start := make(chan struct{})
	out := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, peer := range contenders {
		wg.Go(func() {
			<-start
			c, e := svc.ObserveAndAuthorize(l.ctx, admissionCommand(), "e2e", peer)
			out <- outcome{peer, c, e}
		})
	}
	close(start)
	wg.Wait()
	close(out)
	wins, limits, other := 0, 0, 0
	loser := ""
	for o := range out {
		switch {
		case o.err == nil:
			wins++
		case errors.Is(o.err, devices.ErrLimit):
			limits++
			loser = o.peer
		default:
			other++
		}
	}
	after := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member)
	r.check("final-slot transaction race", "exactly one commit and one ErrLimit", fmt.Sprintf("committed=%d; ErrLimit=%d; other_errors=%d; authorized=%d", wins, limits, other, after), "barrier release two goroutines: ObserveAndAuthorize(real NetBirdAdapter); read committed rows", wins == 1 && limits == 1 && other == 0 && after == 3)
	if loser == "" {
		r.block("Quota race did not identify a loser; dependent slot reuse unavailable")
		return
	}
	loserRows := l.count(`SELECT count(*) FROM member_devices WHERE peer_id=$1`, loser)
	observed, e := svc.Adapter.ObservePeer(l.ctx, "e2e", loser)
	r.check("loser authorization readback", "no device/binding or paid group", fmt.Sprintf("member_device_rows=%d; paid_group=%t; read_ok=%t", loserRows, observed.Authorized, e == nil), "SELECT loser device; real NetBird group membership readback", loserRows == 0 && !observed.Authorized && e == nil)
	_, _ = l.docker("unpause", l.name("worker"))
	l.forwardingGate(r)
	l.traceForwarding(r, "member", first)
	loserService := "member-a3"
	if loser == contenders[1] {
		loserService = "member-a4"
	}
	l.traceForwarding(r, loserService, uuid.Nil)
	if _, err = l.docker("pause", l.name("worker")); err != nil {
		r.block("Could not re-pause worker for slot reuse")
		return
	}
	revoke := l.api(l.a, "DELETE", "/api/v1/member/devices/"+second.String(), nil, commandHeaders())
	freed := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member)
	_, err = svc.ObserveAndAuthorize(l.ctx, admissionCommand(), "e2e", loser)
	restored := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member)
	r.check("revoke releases exactly one slot", "202; 3 to 2 to 3; former loser admitted", fmt.Sprintf("revoke_HTTP=%d; after_revoke=%d; admitted=%t; final_authorized=%d", revoke.status, freed, err == nil, restored), "DELETE A2 through member API; ObserveAndAuthorize former loser", revoke.status == 202 && freed == 2 && err == nil && restored == 3)
	_, _ = l.docker("unpause", l.name("worker"))
	l.forwardingGate(r)
}
