//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// memberNetBird joins the lab member as a native NetBird user. It does not
// insert member_devices, device_bindings, or netbird_user_bindings.
func (r *runner) memberNetBird(ctx context.Context, member *session, note func(name, detail string)) {
	r.usageReady = false
	note("join", "embedded IdP user")
	trigger := filepath.Join(r.lab.state, "member", "var", "e2e", "user-auth")
	if err := os.MkdirAll(filepath.Dir(trigger), 0o755); err != nil {
		note("member_netbird", "trigger directory")
		return
	}
	if writeAtomic(trigger, []byte("1\n")) != nil {
		note("member_netbird", "trigger file")
		return
	}
	cred, err := readMemberUser(filepath.Join(r.lab.state, "netbird", "member-user.json"))
	if err != nil {
		note("member_netbird", "member user unavailable")
		return
	}
	logPath := filepath.Join(r.lab.state, "member", "var", "e2e", "netbird-up.log")
	defer func() {
		_ = os.Remove(logPath)
		_, _ = r.lab.docker(r.lab.member, "rm", "-f", "/var/lib/vpn-node/e2e/netbird-up.log")
	}()

	loginWait, loginCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer loginCancel()
	var flowURL string
	_ = waitUntil(loginWait, time.Second, func() bool {
		if u := deviceFlowURL(r.memberUpLog()); u != "" {
			flowURL = u
			return true
		}
		stage := r.lab.stage(r.lab.member, "/var/lib/vpn-node/e2e/status.json")
		return stage == "netbird_up" || stage == "netbird_failed"
	})
	if flowURL != "" {
		note("device_flow_path", emptyDash(deviceFlowPath(flowURL)))
		r.submitDeviceLogin(ctx, flowURL, cred, note)
	} else {
		note("member_netbird", "device flow url absent")
		if excerpt := excerptMatching(scrubDeviceURL(r.memberUpLog()), []string{"error", "fail", "login", "sso", "device"}, 4); excerpt != "" {
			note("member_up_excerpt", excerpt)
		}
	}

	upWait, upCancel := context.WithTimeout(ctx, 90*time.Second)
	defer upCancel()
	up := waitUntil(upWait, 2*time.Second, func() bool {
		return r.lab.stage(r.lab.member, "/var/lib/vpn-node/e2e/status.json") == "netbird_up"
	})
	note("member_netbird", emptyDash(r.lab.stage(r.lab.member, "/var/lib/vpn-node/e2e/status.json")))
	withUser, total, peerNote := r.netbirdUserPeers(ctx)
	note("peers_with_user", itoa(withUser))
	note("peer_total", itoa(total))
	if peerNote != "" {
		note("peer_read", peerNote)
	}
	if !up {
		note("member_stage", "not up")
	}
	r.awaitMemberBinding(ctx, member, note)
}

// memberUpLog reads the NetBird up log. The member process creates it as
// root mode 0600, so a host read can fail while docker exec still works.
func (r *runner) memberUpLog() string {
	raw, err := os.ReadFile(filepath.Join(r.lab.state, "member", "var", "e2e", "netbird-up.log"))
	if err == nil && len(raw) > 0 {
		return string(raw)
	}
	raw, err = r.lab.docker(r.lab.member, "cat", "/var/lib/vpn-node/e2e/netbird-up.log")
	if err != nil || len(raw) == 0 {
		return ""
	}
	return string(raw)
}

func (r *runner) submitDeviceLogin(ctx context.Context, flowURL string, cred memberUserFile, note func(string, string)) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		note("member_netbird", "cookie jar")
		return
	}
	client := &http.Client{Transport: r.lab.transport, Jar: jar, Timeout: 20 * time.Second}
	loginCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	status, loginErr := completeDeviceLogin(loginCtx, client, flowURL, cred.Email, cred.Password)
	note("device_flow_status", itoa(status))
	if loginErr != nil {
		note("device_login", clipText(scrubDeviceURL(loginErr.Error()), 180))
		return
	}
	note("device_login", "submitted")
}

type memberUserFile struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func readMemberUser(path string) (memberUserFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return memberUserFile{}, err
	}
	var cred memberUserFile
	if json.Unmarshal(raw, &cred) != nil || cred.Email == "" || cred.Password == "" {
		return memberUserFile{}, errors.New("member user unavailable")
	}
	return cred, nil
}

func (r *runner) netbirdUserPeers(ctx context.Context) (withUser, total int, note string) {
	token, err := r.lab.secret("netbird.token")
	if err != nil {
		return 0, 0, "token unavailable"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.lab.netbird+"/api/peers", nil)
	if err != nil {
		return 0, 0, "request"
	}
	req.Header.Set("Authorization", "Token "+token)
	client := &http.Client{Transport: r.lab.transport, Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, "unreachable"
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode != 200 {
		return 0, 0, "HTTP " + itoa(resp.StatusCode)
	}
	var peers []struct {
		UserID string `json:"user_id"`
	}
	if json.Unmarshal(raw, &peers) != nil {
		return 0, 0, "decode"
	}
	for _, peer := range peers {
		total++
		if strings.TrimSpace(peer.UserID) != "" {
			withUser++
		}
	}
	return withUser, total, ""
}

func (r *runner) awaitMemberBinding(ctx context.Context, member *session, note func(string, string)) {
	wait, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var last string
	var deviceID uuid.UUID
	ok := waitUntil(wait, 2*time.Second, func() bool {
		devs, err := member.call(ctx, "GET", r.lab.public+"/api/v1/member/devices", nil, nil)
		if err != nil || devs.Status != 200 {
			last = httpDetail(devs, err)
			return false
		}
		var list api.MemberDeviceList
		if json.Unmarshal(devs.Body, &list) != nil {
			last = "decode"
			return false
		}
		deviceID = uuid.Nil
		authorized := 0
		for _, item := range list.Items {
			if item.Status == api.DeviceStatusAuthorized {
				authorized++
				deviceID = uuid.UUID(item.DeviceId)
			}
		}
		last = "200 authorized " + itoa(authorized) + " of " + itoa(len(list.Items))
		return authorized > 0 && deviceID != uuid.Nil
	})
	note("devices", emptyDash(last))
	note("device_authorized", boolText(ok))
	if !ok {
		note("binding", "not projected")
		return
	}
	bindWait, bindCancel := context.WithTimeout(ctx, 90*time.Second)
	defer bindCancel()
	var binding projectedBinding
	found := waitUntil(bindWait, 2*time.Second, func() bool {
		got, seen := readProjectedBinding(r.desiredRaw(), deviceID)
		if seen {
			binding = got
		}
		return seen
	})
	if !found {
		note("binding", "authorized device has no desired-state binding")
		return
	}
	note("binding_revision", binding.Revision.String())
	note("binding_valid_from", binding.From.UTC().Format(time.RFC3339))
	note("binding_valid_until", binding.Until.UTC().Format(time.RFC3339))
	memberID, err := uuid.Parse(member.memberID)
	if err != nil || binding.Member != memberID {
		note("binding_member", "does not match the session")
		return
	}
	r.usageDevice = binding.Device
	r.usageRevision = binding.Revision
	r.usageFrom = binding.From
	r.usageUntil = binding.Until
	r.usageReady = true
	note("binding", "projected")
}

type projectedBinding struct {
	Device   uuid.UUID
	Member   uuid.UUID
	Revision decstr.U64
	From     time.Time
	Until    time.Time
}

func (r *runner) desiredRaw() []byte {
	raw, err := os.ReadFile(filepath.Join(r.lab.state, "node", "var", "state", "desired-state.json"))
	if err != nil || len(raw) == 0 {
		raw, _ = r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
	}
	return raw
}

func readProjectedBinding(raw []byte, deviceID uuid.UUID) (projectedBinding, bool) {
	if len(raw) == 0 || deviceID == uuid.Nil {
		return projectedBinding{}, false
	}
	var env struct {
		Signed struct {
			DeviceBindings []struct {
				DeviceID        openapi_types.UUID `json:"device_id"`
				MemberID        openapi_types.UUID `json:"member_id"`
				BindingRevision decstr.U64         `json:"binding_revision"`
				ValidFrom       time.Time          `json:"valid_from"`
				ValidUntil      time.Time          `json:"valid_until"`
			} `json:"device_bindings"`
		} `json:"signed"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return projectedBinding{}, false
	}
	for _, item := range env.Signed.DeviceBindings {
		id := uuid.UUID(item.DeviceID)
		if id != deviceID || uint64(item.BindingRevision) == 0 || !item.ValidUntil.After(item.ValidFrom) {
			continue
		}
		return projectedBinding{
			Device: id, Member: uuid.UUID(item.MemberID), Revision: item.BindingRevision,
			From: item.ValidFrom, Until: item.ValidUntil,
		}, true
	}
	return projectedBinding{}, false
}
