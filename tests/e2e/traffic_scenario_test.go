//go:build e2e

package e2e

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// countedFlow proves admission on this node, not merely an HTTP response on
// another route. The member has no connection to the origin bridge.
func (r *runner) countedFlow(ctx context.Context, note func(string, string)) bool {
	defer func() {
		heartbeat, _ := r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
		note("heartbeat", strings.TrimSpace(string(heartbeat)))
	}()
	ready, stopReady := context.WithTimeout(ctx, 90*time.Second)
	var before uint64
	var ok bool
	found := waitUntil(ready, time.Second, func() bool { before, ok = r.admissionBytes(); return ok && r.lab.forwardChain().Generation > 0 })
	stopReady()
	if !found {
		note("admission_counter", "unavailable after 90 seconds")
		return false
	}
	wait, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var after uint64
	success := waitUntil(wait, 2*time.Second, func() bool {
		probe, stop := context.WithTimeout(wait, 8*time.Second)
		defer stop()
		cmd := exec.CommandContext(probe, "docker", "exec", r.lab.member, "wget", "--timeout=5", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "-qO-", r.lab.releases+"/healthz")
		body, err := cmd.Output()
		after, ok = r.admissionBytes()
		return err == nil && strings.TrimSpace(string(body)) == "ok" && ok && after > before
	})
	note("controlled_origin_http_200_and_counted", boolText(success))
	note("admission_bytes_before", strconv.FormatUint(before, 10))
	note("admission_bytes_after", strconv.FormatUint(after, 10))
	return success
}

func (r *runner) admissionBytes() (uint64, bool) {
	raw, err := r.lab.docker(r.lab.node, "nft", "-j", "list", "table", "inet", "fvpn")
	if err != nil {
		return 0, false
	}
	st, err := enforce.ParseState(raw)
	if err != nil {
		return 0, false
	}
	suffix := strings.ReplaceAll(r.usageDevice.String(), "-", "")
	var total uint64
	for _, key := range []string{"u_" + suffix, "d_" + suffix} {
		c, ok := st.Counters[key]
		if !ok {
			return 0, false
		}
		total += c.Bytes
	}
	return total, true
}
