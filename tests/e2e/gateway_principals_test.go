//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

type burstResult struct {
	Counts  map[int]int `json:"status_counts"`
	Codes   int         `json:"rate_limited_codes"`
	Retry   int         `json:"retry_after_present"`
	Address int         `json:"address_refusals"`
	Subject int         `json:"subject_refusals"`
}

func (r *runner) gatewayPrincipals(ctx context.Context) {
	rec := Record{ID: "gateway-principals", Title: "Gateway preserves independent client rate principals", Spec: []string{"§14.2", "§19 AT28", "§19 AT31"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	var stack struct {
		LabPrefix string `json:"lab_prefix"`
	}
	raw, err := os.ReadFile(filepath.Join(r.lab.state, "stack.json"))
	if err != nil || json.Unmarshal(raw, &stack) != nil || stack.LabPrefix == "" {
		return
	}
	central := config.Defaults()
	raw, err = os.ReadFile(filepath.Join(r.lab.state, "config", "central.json"))
	if err != nil || json.Unmarshal(raw, &central) != nil {
		return
	}
	jsonNote(note, "trusted_proxies", central.TrustedProxies)
	if len(central.TrustedProxies) != 1 || central.TrustedProxies[0] != stack.LabPrefix+".21/32" || central.BrowserRate != (config.Bucket{PerSecond: 10, Burst: 30}) || central.IngressRate != (config.Bucket{PerSecond: 20, Burst: 60}) {
		return
	}
	note("rates", "unchanged defaults: browser 10/s burst 30; ingress 20/s burst 60")
	s := &Stack{lab: r.lab}
	a, err := s.Login(ctx, "e2e-member", "member")
	if err != nil {
		return
	}
	b, err := s.Login(ctx, "e2e-member-b", "member")
	if err != nil {
		return
	}
	note("distinct_members", boolText(a.MemberID != "" && b.MemberID != "" && a.MemberID != b.MemberID))
	ca, err := s.StartNativePeer(ctx, "limiter-a")
	if err != nil {
		return
	}
	cb, err := s.StartNativePeer(ctx, "limiter-b")
	if err != nil {
		return
	}
	addresses := []string{}
	for _, container := range []string{ca, cb} {
		out, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", container).Output()
		if err != nil {
			return
		}
		addresses = append(addresses, strings.TrimSpace(string(out)))
	}
	jsonNote(note, "client_source_addresses", addresses)
	if addresses[0] == addresses[1] || !strings.HasPrefix(addresses[0], stack.LabPrefix+".") || !strings.HasPrefix(addresses[1], stack.LabPrefix+".") {
		return
	}
	note("forged_header_scope", "rotating TEST ONLY documentation IPv4 values; source sockets unchanged")
	burst := func(label, container, dial string, session BrowserSession, count int, forge bool) (burstResult, bool) {
		request, _ := http.NewRequest("GET", r.lab.public+"/api/v1/member/devices", nil)
		for _, cookie := range session.Client.Jar.Cookies(request.URL) {
			request.AddCookie(cookie)
		}
		payload, _ := json.Marshal(map[string]any{"dial": dial, "cookie": request.Header.Get("Cookie"), "count": count, "forge": forge})
		wait, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(wait, "docker", "exec", "-i", container, "fvpn-e2e-nodeconfig", "http-burst")
		cmd.Stdin = bytes.NewReader(payload)
		raw, e := cmd.Output()
		var out burstResult
		ok := e == nil && json.Unmarshal(raw, &out) == nil
		if ok {
			note(label, strings.TrimSpace(string(raw)))
		} else {
			note(label, "bounded request failed; no cookie/body emitted")
		}
		return out, ok
	}
	before, ok := r.browserRateMetrics()
	if !ok {
		return
	}
	gateway := stack.LabPrefix + ".21:8443"
	first, okA := burst("member_a_gateway", ca, gateway, a, 120, false)
	forged, okF := burst("member_a_forged_gateway", ca, gateway, a, 120, true)
	other, okB := burst("member_b_gateway", cb, gateway, b, 1, false)
	after, okM := r.browserRateMetrics()
	jsonNote(note, "metrics_before", before)
	jsonNote(note, "metrics_after_gateway", after)
	direct, okD := burst("member_a_forged_direct", ca, stack.LabPrefix+".30:8443", a, 120, true)
	end, okE := r.browserRateMetrics()
	jsonNote(note, "metrics_after_direct", end)
	directOther, okO := burst("member_b_forged_direct", cb, stack.LabPrefix+".30:8443", b, 1, true)
	note("limiter_state_reads", "none; responses and public control metric counters only")
	if okA && okF && okB && okM && okD && okE && okO && a.MemberID != b.MemberID && first.Counts[200] > 0 && first.Subject > 0 && first.Address > 0 && forged.Address > 0 && other.Counts[200] == 1 && direct.Address > 0 && directOther.Counts[200] == 1 && after["address"] > before["address"] && after["subject"] > before["subject"] && end["address"] == after["address"]+direct.Address && first.Codes == first.Counts[429] && first.Retry == first.Counts[429] {
		rec.Status = StatusPass
	}
}

func jsonNote(note func(string, string), name string, value any) {
	raw, _ := json.Marshal(value)
	note(name, string(raw))
}
func (r *runner) browserRateMetrics() (map[string]int, bool) {
	raw, err := r.lab.docker(r.lab.control, "wget", "-q", "-T", "3", "-O-", "http://127.0.0.1:9091/metrics")
	out := map[string]int{"address": 0, "subject": 0}
	if err != nil {
		return out, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "fvpn_control_rate_limited_total{") || !strings.Contains(line, `class="browser"`) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return out, false
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return out, false
		}
		for key := range out {
			if strings.Contains(line, `limiter="`+key+`"`) {
				out[key] = int(value)
			}
		}
	}
	return out, true
}
