//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *runner) selectExit(ctx context.Context, note func(string, string)) bool {
	before, _ := r.lab.docker(r.lab.member, "netbird", "networks", "list")
	note("netbird_networks_before_selection", strings.TrimSpace(string(before)))
	// Select the one product-created exit explicitly: NetBird 'select all'
	// intentionally leaves default routes unselected on this pinned client.
	re := regexp.MustCompile(`ID: (fvpn-exit-[a-z0-9]+)\s+Network: 0\.0\.0\.0/0`)
	matches := re.FindAllStringSubmatch(string(before), -1)
	if len(matches) != 1 {
		note("exit_selection", "expected exactly one API-published exit")
		return false
	}
	out, err := r.lab.docker(r.lab.member, "netbird", "networks", "select", matches[0][1])
	note("exit_selection", strings.TrimSpace(string(out)))
	wait, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	selected := err == nil && waitUntil(wait, time.Second, func() bool {
		route, _ := r.lab.docker(r.lab.member, "ip", "route", "get", r.originIP())
		return strings.Contains(string(route), "dev fvnb0")
	})
	after, _ := r.lab.docker(r.lab.member, "netbird", "networks", "list")
	note("netbird_networks_after_selection", strings.TrimSpace(string(after)))
	return selected
}

func (r *runner) originIP() string {
	// Derive the isolated origin address from the prepared node uplink config.
	raw, _ := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/agent-run.json")
	var cfg struct {
		Datapath struct {
			SNAT string `json:"snat"`
		} `json:"datapath"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return ""
	}
	i := strings.LastIndex(cfg.Datapath.SNAT, ".")
	if i < 0 {
		return ""
	}
	return cfg.Datapath.SNAT[:i] + ".10"
}

func (r *runner) originProbe(ctx context.Context, method, id string) (string, error) {
	secret, err := r.lab.secret("origin.secret")
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(map[string]string{"id": id})
	req, err := http.NewRequestWithContext(ctx, method, r.lab.releases+"/e2e/traffic", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-E2E-Secret", secret)
	resp, err := (&http.Client{Transport: r.lab.transport, Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return "", fmt.Errorf("origin probe HTTP %d", resp.StatusCode)
	}
	return strings.TrimSpace(string(body)), nil
}

func (r *runner) traceSnapshot(label, source string, note func(string, string)) {
	commands := []struct {
		name, container string
		args            []string
	}{
		{"member_route", r.lab.member, []string{"ip", "route", "get", r.originIP()}},
		{"member_tunnel_address", r.lab.member, []string{"ip", "-o", "-4", "addr", "show", "fvnb0"}},
		{"member_routes", r.lab.member, []string{"ip", "-4", "route", "show", "table", "all"}},
		{"node_nft", r.lab.node, []string{"nft", "-j", "list", "table", "inet", "fvpn"}},
		{"conntrack", r.lab.node, []string{"conntrack", "-L", "-s", source, "-d", r.originIP(), "-o", "extended"}},
	}
	for _, c := range commands {
		out, err := r.lab.docker(c.container, c.args...)
		if c.name == "node_nft" {
			out = nftTrace(out)
		}
		detail := strings.TrimSpace(string(out))
		if err != nil {
			detail = "command failed: " + detail
		}
		note(label+"_"+c.name, detail)
	}
	for _, c := range []struct{ container, iface, name string }{{r.lab.member, "fvnb0", "member_tunnel"}, {r.lab.node, "fvmb0", "node_tunnel"}, {r.lab.node, "fvup0", "node_uplink"}} {
		// Only fixed interface names in our own containers; no host networking.
		out, _ := r.lab.docker(c.container, "sh", "-c", `for key in rx_packets rx_bytes tx_packets tx_bytes; do printf '%s=' "$key"; cat /sys/class/net/`+c.iface+`/statistics/"$key"; done`)
		note(label+"_"+c.name, strings.TrimSpace(string(out)))
	}
}

// Preserve every forward/NAT rule with its handles, named counters and timeout
// elements. Marker rules and the product SNAT rule have no counters: say so
// explicitly instead of adding rules or altering the product's enforcement.
func nftTrace(raw []byte) []byte {
	var doc struct {
		Items []map[string]json.RawMessage `json:"nftables"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return raw
	}
	kept := []map[string]json.RawMessage{}
	for _, item := range doc.Items {
		if item["counter"] != nil {
			kept = append(kept, item)
			continue
		}
		if body := item["rule"]; body != nil {
			var rule struct {
				Chain string `json:"chain"`
			}
			_ = json.Unmarshal(body, &rule)
			if rule.Chain == "forward" || rule.Chain == "postrouting" {
				kept = append(kept, item)
			}
		}
		if body := item["set"]; body != nil {
			var set struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(body, &set)
			if set.Name == "allowed_v4" {
				kept = append(kept, item)
			}
		}
	}
	out, _ := json.Marshal(kept)
	return out
}

func (r *runner) tracedFetch(ctx context.Context, note func(string, string)) bool {
	addr, _ := r.lab.docker(r.lab.member, "ip", "-o", "-4", "addr", "show", "fvnb0")
	re := regexp.MustCompile(`inet (100\.[0-9.]+)/`)
	m := re.FindStringSubmatch(string(addr))
	if len(m) != 2 {
		return false
	}
	source := m[1]
	id := uuid.NewString()
	if _, err := r.originProbe(ctx, "PUT", id); err != nil {
		note("origin_probe", err.Error())
		return false
	}
	r.traceSnapshot("before", source, note)
	before, available := r.admissionBytes()
	type capture struct {
		name    string
		cmd     *exec.Cmd
		output  bytes.Buffer
		started bool
	}
	captures := []*capture{}
	wait, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	for _, spec := range []struct{ name, container, iface string }{{"member", r.lab.member, "fvnb0"}, {"node_ingress", r.lab.node, "fvmb0"}, {"node_uplink", r.lab.node, "fvup0"}} {
		c := &capture{name: spec.name, cmd: exec.CommandContext(wait, "docker", "exec", spec.container, "timeout", "-s", "INT", "8", "tcpdump", "-nn", "-q", "-s", "96", "-i", spec.iface, "host", r.originIP(), "and", "tcp", "port", "443")}
		c.cmd.Stdout = &c.output
		c.cmd.Stderr = &c.output
		c.started = c.cmd.Start() == nil
		captures = append(captures, c)
	}
	time.Sleep(time.Second)
	probe, stop := context.WithTimeout(wait, 7*time.Second)
	cmd := exec.CommandContext(probe, "docker", "exec", r.lab.member, "wget", "--timeout=5", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "--header=X-E2E-Probe: "+id, "-qO-", r.lab.releases+"/healthz")
	body, err := cmd.Output()
	stop()
	after, ok := r.admissionBytes()
	for _, c := range captures {
		if c.started {
			_ = c.cmd.Wait()
		}
		note("capture_"+c.name, strings.TrimSpace(c.output.String()))
	}
	r.traceSnapshot("after", source, note)
	origin, e := r.originProbe(ctx, "GET", id)
	note("origin_arrival", origin)
	note("snat_counter_limit", "product postrouting SNAT rule has no counter expression; uplink capture, conntrack reply tuple and authenticated origin source provide translation evidence")
	var arrival struct {
		ID       string `json:"probe_id"`
		Requests int    `json:"requests"`
		Source   string `json:"source_address"`
	}
	arrived := json.Unmarshal([]byte(origin), &arrival) == nil && arrival.ID == id && arrival.Requests > 0 && arrival.Source == strings.TrimSuffix(r.originIP(), ".10")+".20"
	success := arrived && err == nil && strings.TrimSpace(string(body)) == "ok" && available && ok && after > before && e == nil
	note("controlled_origin_http_200_and_counted", boolText(success))
	note("admission_bytes_before", fmt.Sprint(before))
	note("admission_bytes_after", fmt.Sprint(after))
	if success {
		note("first_stopped_hop", "none after explicit NetBird exit selection")
		r.trafficOwner = ""
	} else {
		r.trafficOwner = "w2-agent-core / w1-netlab: compare named drop/admission counters and tunnel captures in member-traffic; lease generation alone is not the gate"
		note("first_stopped_hop", "see packet captures and rule counter deltas; do not infer a lease defect")
	}
	return success
}

// Show only signed deadlines/generation, never bindings or credentials.
func expiryView(raw []byte) string {
	var env struct {
		Signed struct {
			IssuedAt   string `json:"issued_at"`
			ExpiresAt  string `json:"expires_at"`
			Generation string `json:"config_generation"`
		} `json:"signed"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return "unavailable"
	}
	out, _ := json.Marshal(env.Signed)
	return string(out)
}
