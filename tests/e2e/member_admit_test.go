//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
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
func (r *runner) memberNetBird(ctx context.Context, member *session, portalSub string, note func(name, detail string)) {
	r.usageReady = false
	peer, err := (&Stack{lab: r.lab}).JoinNativeMember(ctx, BrowserSession{Client: member.http, CSRF: member.csrf, MemberID: member.memberID}, "e2e-member", r.lab.member, "fvpn-member-e2e")
	note("join", "embedded IdP native user; exact peer user id compared with portal subject")
	note("same_subject", boolText(peer.IdentityMatched && portalSub != ""))
	note("device_authorized", boolText(peer.Authorized))
	if err != nil {
		note("member_netbird", err.Error())
		return
	}
	note("member_netbird", "netbird_up")
	r.awaitMemberBinding(ctx, member, note)
}

func (r *runner) notePortalSubject(ctx context.Context, memberID string, note func(string, string)) string {
	issuer, subject, err := r.lab.bindingSubject(ctx, memberID)
	if err != nil || subject == "" {
		note("portal_sub", "unavailable")
		return ""
	}
	note("portal_issuer", issuer)
	note("portal_sub", subject)
	return subject
}

func (r *runner) noteProjection(ctx context.Context, memberID string, note func(string, string)) {
	users, devices, bindings, external, err := r.lab.projectionCounts(ctx, memberID)
	if err != nil {
		note("projection", "unavailable")
		return
	}
	note("netbird_user_bindings", itoa(users))
	note("member_devices", itoa(devices))
	note("device_bindings", itoa(bindings))
	if users == 0 {
		note("projection", "D-052 mapping not observed; owner w3-wire-members")
		return
	}
	note("external_user_id", external)
	note("projection", "mapping present")
}

func containsCSV(list, item string) bool {
	if item == "" || list == "" {
		return false
	}
	for _, part := range strings.Split(list, ",") {
		if part == item {
			return true
		}
	}
	return false
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
