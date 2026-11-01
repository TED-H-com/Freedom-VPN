//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// kernelTrace is TEST ONLY. All commands execute in owned namespaces. Its
// distinct tables only count/mark UDP for nftrace; they never change a verdict.
// netdev ingress precedes ipv6_rcv_core, unlike inet prerouting.
type kernelTrace struct {
	stack          *Stack
	id, table, dir string
	roles          []string
}

func (s *Stack) startKernelTrace(ctx context.Context) (_ *kernelTrace, err error) {
	k := &kernelTrace{stack: s, id: uuid.NewString(), roles: []string{"member"}}
	// Node-wide nftrace mangle is rejected by its foreign-chain verifier.
	// Historical reproduction must be explicit; normal observation only marks
	// member packets. Never weaken the node verifier to accommodate a tracer.
	if os.Getenv("FVPN_E2E_REPRODUCE_TRACE26") == "1" {
		k.roles = append(k.roles, "node")
	}
	k.table = "fvtrace_" + strings.ReplaceAll(k.id, "-", "")[:12]
	k.dir = "/var/lib/vpn-node/e2e/" + k.table
	defer func() {
		if err != nil {
			_, _ = k.stop(context.WithoutCancel(ctx), filepath.Join(s.lab.root, ".agent/tmp/followup24/incomplete", k.id))
		}
	}()
	for _, role := range k.roles {
		c := k.container(role)
		if err = exec.CommandContext(ctx, "docker", "exec", c, "mkdir", "-m", "700", "-p", k.dir).Run(); err != nil {
			return nil, err
		}
		var links []struct {
			Name string `json:"ifname"`
		}
		raw, e := exec.CommandContext(ctx, "docker", "exec", c, "ip", "-j", "link").Output()
		if e != nil || json.Unmarshal(raw, &links) != nil {
			return nil, fmt.Errorf("trace links unavailable")
		}
		script := "add table netdev " + k.table + "\nadd table inet " + k.table + "\n"
		add := func(family, chain, hook string) {
			script += fmt.Sprintf("add counter %s %s %s_source\n", family, k.table, chain)
			script += fmt.Sprintf("add counter %s %s %s_destination\n", family, k.table, chain)
			script += fmt.Sprintf("add chain %s %s %s { type filter hook %s priority -500; policy accept; }\n", family, k.table, chain, hook)
			script += fmt.Sprintf("add rule %s %s %s meta l4proto udp ip6 saddr ::1 counter name %s_source meta nftrace set 1\n", family, k.table, chain, chain)
			script += fmt.Sprintf("add rule %s %s %s meta l4proto udp ip6 daddr ::1 counter name %s_destination meta nftrace set 1\n", family, k.table, chain, chain)
			// A normal IPv6 UDP control demonstrates that the later hooks work.
			script += fmt.Sprintf("add rule %s %s %s ip6 nexthdr udp meta nftrace set 1\n", family, k.table, chain)
		}
		for _, link := range links {
			if link.Name == "lo" || strings.HasPrefix(link.Name, "eth") {
				add("netdev", "ingress_"+link.Name, "ingress device "+link.Name)
			}
		}
		for _, hook := range []string{"prerouting", "input", "output"} {
			add("inet", hook, hook)
		}
		cmd := exec.CommandContext(ctx, "docker", "exec", "-i", c, "nft", "-f", "-")
		cmd.Stdin = strings.NewReader(script)
		if err = cmd.Run(); err != nil {
			return nil, fmt.Errorf("trace table installation: %w", err)
		}
		script = `umask 077; echo $$ > "$1/pid"; exec stdbuf -oL nft monitor trace > "$1/trace.txt" 2> "$1/error.txt"`
		if err = exec.CommandContext(ctx, "docker", "exec", "-d", c, "sh", "-c", script, "trace", k.dir).Run(); err != nil {
			return nil, err
		}
		ready, cancel := context.WithTimeout(ctx, 3*time.Second)
		ok := waitUntil(ready, 50*time.Millisecond, func() bool {
			return exec.CommandContext(ready, "docker", "exec", c, "sh", "-c", `test -f "$1/trace.txt" && read pid < "$1/pid" && kill -0 "$pid"`, "trace", k.dir).Run() == nil
		})
		cancel()
		if !ok {
			return nil, fmt.Errorf("trace monitor not ready")
		}
	}
	return k, nil
}
func (k *kernelTrace) container(role string) string {
	if role == "member" {
		return k.stack.lab.member
	}
	return k.stack.lab.node
}

type kernelTraceResult struct {
	Table string                     `json:"table"`
	Roles map[string]kernelTraceRole `json:"roles"`
}
type kernelTraceRole struct {
	Trace    ContainerExport   `json:"trace_receipt"`
	Errors   ContainerExport   `json:"error_receipt"`
	Counters map[string]uint64 `json:"loopback_udp_counters"`
	Stopped  bool              `json:"stopped"`
	Removed  bool              `json:"removed"`
}

func (k *kernelTrace) stop(ctx context.Context, private string) (kernelTraceResult, error) {
	out := kernelTraceResult{Table: k.table, Roles: map[string]kernelTraceRole{}}
	var errs []error
	for _, role := range k.roles {
		c := k.container(role)
		r := kernelTraceRole{Counters: map[string]uint64{}}
		for _, family := range []string{"netdev", "inet"} {
			raw, e := exec.CommandContext(ctx, "docker", "exec", c, "nft", "-j", "list", "table", family, k.table).Output()
			var doc struct {
				Objects []struct {
					Counter *struct {
						Name    string `json:"name"`
						Packets uint64 `json:"packets"`
					} `json:"counter"`
				} `json:"nftables"`
			}
			if e != nil || json.Unmarshal(raw, &doc) != nil {
				errs = append(errs, fmt.Errorf("%s trace counters unavailable", role))
			} else {
				for _, o := range doc.Objects {
					if o.Counter != nil {
						r.Counters[o.Counter.Name] = o.Counter.Packets
					}
				}
			}
		}
		_ = exec.CommandContext(ctx, "docker", "exec", c, "sh", "-c", `read pid < "$1/pid"; kill -TERM "$pid"`, "trace", k.dir).Run()
		wait, cancel := context.WithTimeout(ctx, 3*time.Second)
		r.Stopped = waitUntil(wait, 50*time.Millisecond, func() bool {
			return exec.CommandContext(wait, "docker", "exec", c, "sh", "-c", `read pid < "$1/pid"; kill -0 "$pid"`, "trace", k.dir).Run() != nil
		})
		cancel()
		var e error
		r.Trace, e = exportCommand(exec.CommandContext(ctx, "docker", "exec", c, "cat", k.dir+"/trace.txt"), filepath.Join(private, role+"-trace.txt"))
		if e != nil {
			errs = append(errs, e)
		}
		r.Errors, e = exportCommand(exec.CommandContext(ctx, "docker", "exec", c, "cat", k.dir+"/error.txt"), filepath.Join(private, role+"-trace-errors.txt"))
		if e != nil {
			errs = append(errs, e)
		}
		r.Removed = true
		for _, family := range []string{"netdev", "inet"} {
			if exec.CommandContext(ctx, "docker", "exec", c, "nft", "delete", "table", family, k.table).Run() != nil {
				r.Removed = false
			}
		}
		if !r.Stopped || !r.Removed || r.Errors.Bytes != "0" {
			errs = append(errs, fmt.Errorf("%s trace stop/cleanup/monitor failed", role))
		}
		out.Roles[role] = r
	}
	if len(errs) > 0 {
		return out, fmt.Errorf("kernel trace incomplete: %v", errs)
	}
	return out, nil
}
