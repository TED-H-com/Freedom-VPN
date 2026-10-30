//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
)

var nativeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

type NativePeer struct {
	Container       string
	IdentityMatched bool
	Authorized      bool
	DeviceID        uuid.UUID
	LogScan         NativeLogScan
}
type NativeScale struct {
	Users                  int
	Peers                  int
	IdentityMatches        int
	Authorized             int
	ScannedLogs            int
	ProtectedLogs          int
	SimultaneousAuthorized int
}

// JoinNativeMember drives the real device flow on an existing prefixed peer.
// Each peer must have its own NetBird state, socket and network namespace.
func (s *Stack) JoinNativeMember(ctx context.Context, member BrowserSession, subject, container, peerName string) (out NativePeer, err error) {
	out = NativePeer{Container: container}
	if !strings.HasPrefix(container, s.lab.prefix+"-member") {
		return out, fmt.Errorf("member container outside stack")
	}
	wait, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	_, portalSub, err := s.lab.bindingSubject(wait, member.MemberID)
	if err != nil {
		return out, err
	}
	token, err := s.lab.secret("netbird.token")
	if err != nil {
		return out, err
	}
	browser, err := s.lab.newSession()
	if err != nil {
		return out, err
	}
	browser.http.CheckRedirect = idplogin.RedirectLimit(20)
	cred, err := idplogin.CredFor(wait, browser.http, s.lab.netbird+"/api", token, s.lab.state, subject)
	if err != nil {
		return out, err
	}
	logPath := "/var/lib/vpn-node/e2e/netbird-up.log"
	defer func() {
		out.LogScan, _ = s.ScanNativeJoinLog(ctx, container)
		_, _ = s.lab.docker(container, "rm", "-f", logPath)
	}()
	if _, err = s.lab.docker(container, "touch", "/var/lib/vpn-node/e2e/user-auth"); err != nil {
		return out, err
	}
	var flow string
	if !waitUntil(wait, time.Second, func() bool {
		raw, _ := s.lab.docker(container, "cat", logPath)
		flow = deviceFlowURL(string(raw))
		return flow != ""
	}) {
		return out, fmt.Errorf("native device flow timed out")
	}
	if _, err = completeDeviceLogin(wait, browser.http, flow, cred.Email, cred.Password); err != nil {
		return out, fmt.Errorf("native login failed")
	}
	client := &http.Client{Transport: s.lab.transport, Timeout: 10 * time.Second}
	var peerID string
	ok := waitUntil(wait, 2*time.Second, func() bool {
		if s.lab.stage(container, "/var/lib/vpn-node/e2e/status.json") != "netbird_up" {
			return false
		}
		req, e := http.NewRequestWithContext(wait, "GET", s.lab.netbird+"/api/peers", nil)
		if e != nil {
			return false
		}
		req.Header.Set("Authorization", "Token "+token)
		resp, e := client.Do(req)
		if e != nil {
			return false
		}
		defer resp.Body.Close()
		var peers []struct {
			ID, Name string
			UserID   string `json:"user_id"`
		}
		if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&peers) != nil {
			return false
		}
		for _, p := range peers {
			if p.Name == peerName && p.UserID == portalSub && portalSub != "" {
				out.IdentityMatched = true
				peerID = p.ID
			}
		}
		if !out.IdentityMatched {
			return false
		}
		devices, e := s.Call(wait, member, "GET", "/api/v1/member/devices", nil, nil)
		var page api.MemberDeviceList
		if e != nil || devices.Status != 200 || json.Unmarshal(devices.Body, &page) != nil {
			return false
		}
		// Product mapping must point this exact peer at this exact member, then the
		// public device surface must authorize that device. No email inference.
		conn, e := s.lab.db(wait)
		if e != nil {
			return false
		}
		defer conn.Close(wait)
		var device uuid.UUID
		if conn.QueryRow(wait, `SELECT id FROM member_devices WHERE member_id=$1 AND peer_id=$2`, member.MemberID, peerID).Scan(&device) != nil {
			return false
		}
		for _, d := range page.Items {
			if d.DeviceId == device && d.Status == api.DeviceStatusAuthorized {
				out.Authorized = true
				out.DeviceID = device
				return true
			}
		}
		return false
	})
	if !ok {
		return out, fmt.Errorf("native identity/admission timed out (identity=%t authorized=%t)", out.IdentityMatched, out.Authorized)
	}
	return out, nil
}

// StartNativePeer uses the running node's immutable image and existing lab
// network. It creates only this prefix's container and volume, removed by down.
func (s *Stack) StartNativePeer(ctx context.Context, name string) (string, error) {
	if !nativeName.MatchString(name) {
		return "", fmt.Errorf("invalid native peer name")
	}
	container := s.lab.prefix + "-member-" + name
	image, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", s.lab.node).Output()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "--name", container, "--hostname", container, "--network", s.lab.prefix, "--add-host", "releases.example.invalid:"+s.lab.originIP, "--cap-add=NET_ADMIN", "--cap-add=NET_RAW", "--device=/dev/net/tun:/dev/net/tun", "--env", "FVPN_E2E_MEMBER_NAME="+container, "--mount", "type=bind,src="+filepath.Join(s.lab.state, "node", "etc", "ca.pem")+",dst=/etc/vpn-node/ca.pem,readonly", "--mount", "type=volume,src="+container+",dst=/var/lib/vpn-node", "--entrypoint", "/usr/local/bin/entrypoint-member.sh", strings.TrimSpace(string(image)))
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("start native peer: %w", err)
	}
	return container, nil
}

// ScaleNativeMembers is the SECOPS-10 hook. 20x3 is supported by the hook;
// callers choose the execution budget and must use a plan allowing three devices.
// Each native peer and subject is distinct; the return contains only counts.
func (s *Stack) ScaleNativeMembers(ctx context.Context, admin BrowserSession, plan uuid.UUID, namespace string, users, peers int, refresh ...func(context.Context) (BrowserSession, error)) (NativeScale, error) {
	var out NativeScale
	if len(refresh) > 1 {
		return out, fmt.Errorf("at most one admin refresh callback")
	}
	if users < 1 || users > 20 || peers < 1 || peers > 3 || !nativeName.MatchString(namespace) || len(namespace) > 12 || plan == uuid.Nil {
		return out, fmt.Errorf("scale requires 1..20 users, 1..3 peers, plan and short namespace")
	}
	seen := map[string]bool{}
	devices := []uuid.UUID{}
	lastAuth := time.Time{}
	for u := 1; u <= users; u++ {
		if len(refresh) == 1 && time.Since(lastAuth) > 5*time.Minute {
			fresh, e := refresh[0](ctx)
			if e != nil {
				return out, e
			}
			admin = fresh
			lastAuth = time.Now()
		}
		subject := fmt.Sprintf("%s-u%02d", namespace, u)
		member, err := s.Login(ctx, subject, "member")
		if err != nil {
			return out, err
		}
		if member.MemberID == "" || seen[member.MemberID] {
			return out, fmt.Errorf("distinct member identity required")
		}
		seen[member.MemberID] = true
		grant, err := s.Call(ctx, admin, "POST", "/api/v1/admin/members/"+member.MemberID+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY native scale", PlanRevisionId: &plan, ValidUntil: time.Now().UTC().Add(time.Hour)}, map[string]string{"Idempotency-Key": uuid.NewString()})
		if err != nil || grant.Status != 202 {
			return out, fmt.Errorf("scale grant: %s", httpDetail(grant, err))
		}
		out.Users++

		// A user's three independent namespaces join concurrently. Identity
		// and authorization are still checked per peer before counts advance.
		type joined struct {
			peer NativePeer
			err  error
		}
		joinedPeers := make(chan joined, peers)
		for p := 1; p <= peers; p++ {
			name := fmt.Sprintf("%s-p%d", subject, p)
			go func() {
				container, e := s.StartNativePeer(ctx, name)
				if e != nil {
					joinedPeers <- joined{err: e}
					return
				}
				peer, e := s.JoinNativeMember(ctx, member, subject, container, container)
				joinedPeers <- joined{peer: peer, err: e}
			}()
		}
		var firstErr error
		for range peers {
			result := <-joinedPeers
			peer := result.peer
			out.Peers++
			if peer.LogScan.Present {
				out.ScannedLogs++
			}
			if peer.LogScan.Present && peer.LogScan.Mode == "0:600" {
				out.ProtectedLogs++
			}
			if peer.IdentityMatched {
				out.IdentityMatches++
			}
			if peer.Authorized {
				out.Authorized++
				devices = append(devices, peer.DeviceID)
			}
			if result.err != nil && firstErr == nil {
				firstErr = result.err
			}
		}
		if firstErr != nil {
			return out, firstErr
		}
	}
	conn, e := s.lab.db(ctx)
	if e != nil {
		return out, e
	}
	defer conn.Close(ctx)
	if e = conn.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE id=ANY($1::uuid[]) AND status='authorized'`, devices).Scan(&out.SimultaneousAuthorized); e != nil {
		return out, e
	}
	return out, nil
}
