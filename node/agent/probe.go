package agent

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

const (
	probeTimeout  = 2 * time.Second
	probeFileMax  = 1 << 20
	probeDetailOK = "overlay_up"
)

// ProbeRequest is one data-path probe (§5 data_health, §13.9).
// Unpublished covers NetBird management and relay reachability, plus an
// egress target when one is configured. Published also requires the member
// forwarding interfaces. The probe does not change publication intent and
// does not install a member allow.
type ProbeRequest struct {
	Published bool
	// Egress is one host:port from agent.json. Targets, when non-empty,
	// replace it. A non-null signed egress_probe fills Targets and leaves
	// Egress empty (D-058).
	Egress string
	// Targets are host:port values, IPv6 bracketed. Empty with Egress empty
	// means no egress leg.
	Targets []string
	// Timeout is the dial limit for each target. Zero uses 2000 ms.
	Timeout time.Duration
	// Rejected means a signed probe was present and unusable. The probe
	// does not dial the local target in its place.
	Rejected bool
	Ingress  string
	Uplink   string
}

// ProbeResult is the outcome of one probe. Ran is false only when the
// prober was not asked to run. A failed read is Ran with a failed check.
type ProbeResult struct {
	Ran    bool
	Check  api.HealthCheckStatus
	Health api.DataHealth
	// Detail is a short token. It is not a URL and not a key.
	Detail string
	// NetbirdVersion is daemonVersion from the status document this probe
	// just read. Empty means the field was missing or not a safe token.
	// It is never copied from a health challenge.
	NetbirdVersion string
}

// Prober runs the data-path probe. Tests substitute a fake.
type Prober interface {
	Probe(ctx context.Context, req ProbeRequest) ProbeResult
}

type nbStatus struct {
	// DaemonVersion is the NetBird field name (daemonVersion). The probe
	// reads it from the same document as management and relays. It does
	// not shell out a second time and does not log the document.
	DaemonVersion string `json:"daemonVersion"`
	Management    struct {
		Connected bool `json:"connected"`
	} `json:"management"`
	Relays struct {
		Total     int `json:"total"`
		Available int `json:"available"`
	} `json:"relays"`
}

// judgeProbe maps a parsed NetBird status to data health and the tunnel
// check. Management down, or a status that could not be read, is failed.
// Management up with relays configured but none available is degraded and
// the check still passes: the overlay control session is up. A named egress
// target that does not answer, or a published node whose member interfaces
// are absent, is degraded and the check fails.
func judgeProbe(published, egressNamed, egressOK, memberOK bool, st nbStatus, statusOK bool) (api.DataHealth, api.HealthCheckStatus) {
	if !statusOK || !st.Management.Connected {
		return api.DataHealthFailed, api.HealthCheckStatusFailed
	}
	relayOK := st.Relays.Total == 0 || st.Relays.Available > 0
	egressPass := !egressNamed || egressOK
	memberPass := !published || memberOK
	if relayOK && egressPass && memberPass {
		return api.DataHealthHealthy, api.HealthCheckStatusPassed
	}
	if egressPass && memberPass {
		return api.DataHealthDegraded, api.HealthCheckStatusPassed
	}
	return api.DataHealthDegraded, api.HealthCheckStatusFailed
}

type netbirdProber struct {
	statusFile string
	dial       func(ctx context.Context, addr string) error
	iface      func(name string) error
	run        func(ctx context.Context) ([]byte, error)
}

func newNetbirdProber(cfg Config) netbirdProber {
	return netbirdProber{statusFile: cfg.NetbirdStatusFile}
}

func (p netbirdProber) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	readCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	raw, err := p.read(readCtx)
	cancel()
	st, statusOK := nbStatus{}, false
	if err == nil {
		if jerr := json.Unmarshal(raw, &st); jerr == nil {
			statusOK = true
		}
	}
	targets := req.Targets
	if len(targets) == 0 && req.Egress != "" {
		targets = []string{req.Egress}
	}
	egressNamed := req.Rejected || len(targets) > 0
	egressOK := false
	if egressNamed && !req.Rejected && statusOK && st.Management.Connected {
		egressOK = true
		timeout := req.Timeout
		if timeout <= 0 {
			timeout = probeTimeout
		}
		for _, addr := range targets {
			if p.dialAddr(ctx, addr, timeout) != nil {
				egressOK = false
			}
		}
	}
	memberOK := false
	if req.Published {
		memberOK = p.memberUp(req.Ingress, req.Uplink)
	}
	health, check := judgeProbe(req.Published, egressNamed, egressOK, memberOK, st, statusOK)
	detail := "overlay_down"
	if check == api.HealthCheckStatusPassed && health == api.DataHealthHealthy {
		detail = probeDetailOK
	} else if health == api.DataHealthDegraded {
		detail = "overlay_partial"
	}
	version := ""
	if statusOK {
		version = netbirdVersion(st.DaemonVersion)
	}
	return ProbeResult{Ran: true, Check: check, Health: health, Detail: detail, NetbirdVersion: version}
}

// netbirdVersion keeps a version token that can be written into an
// attestation. A URL, a space, a control character, or a string longer
// than 64 bytes becomes empty so the status document cannot carry a name
// or a key into the health file. Empty fails the updater's version check
// closed; the probe does not invent a version.
func netbirdVersion(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || len(s) > 64 {
		return ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x21 || c > 0x7e || c == '/' || c == ':' {
			return ""
		}
	}
	return s
}

func (p netbirdProber) read(ctx context.Context) ([]byte, error) {
	if p.statusFile != "" {
		f, err := os.Open(p.statusFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return readLimited(f)
	}
	if p.run != nil {
		return p.run(ctx)
	}
	cmd := exec.CommandContext(ctx, "netbird", "status", "--json")
	return cmd.Output()
}

func (p netbirdProber) dialAddr(ctx context.Context, addr string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = probeTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if p.dial != nil {
		return p.dial(ctx, addr)
	}
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return c.Close()
}

func (p netbirdProber) memberUp(ingress, uplink string) bool {
	if ingress == "" || uplink == "" || ingress == uplink {
		return false
	}
	look := p.iface
	if look == nil {
		look = func(name string) error {
			_, err := net.InterfaceByName(name)
			return err
		}
	}
	return look(ingress) == nil && look(uplink) == nil
}

func readLimited(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, probeFileMax+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > probeFileMax {
		return nil, errProbeTooBig
	}
	return b, nil
}

var errProbeTooBig = errString("agent: netbird status is too large")

// egressTargets chooses the dial list. A non-null signed egress_probe on an
// in-force desired state replaces the local host:port. Absent, null, or a
// desired state that is not in force keeps the local target.
func egressTargets(ds *api.DesiredState, desiredOK bool, local string) (targets []string, timeout time.Duration, source string, bad bool) {
	if ds != nil && desiredOK {
		if ep, err := ds.EgressProbe.Get(); err == nil {
			list, limit, ok := signedEgress(ep)
			if !ok {
				return nil, 0, "signed", true
			}
			return list, limit, "signed", false
		}
	}
	if local != "" {
		return []string{local}, probeTimeout, "local", false
	}
	return nil, 0, "none", false
}

func signedEgress(ep api.EgressProbe) ([]string, time.Duration, bool) {
	if len(ep.Targets) < 1 || len(ep.Targets) > 4 {
		return nil, 0, false
	}
	timeout := 2000 * time.Millisecond
	if ep.TimeoutMs != nil {
		if *ep.TimeoutMs < 100 || *ep.TimeoutMs > 10000 {
			return nil, 0, false
		}
		timeout = time.Duration(*ep.TimeoutMs) * time.Millisecond
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ep.Targets))
	for _, t := range ep.Targets {
		if t.Protocol != api.EgressProbeTargetProtocolTcp {
			return nil, 0, false
		}
		addr, err := netip.ParseAddr(t.Address)
		if err != nil || addr.Zone() != "" || t.Port < 1 || t.Port > 65535 {
			return nil, 0, false
		}
		host := net.JoinHostPort(addr.String(), strconv.Itoa(t.Port))
		if _, ok := seen[host]; ok {
			return nil, 0, false
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	return out, timeout, true
}

func probeSourceDetail(source string, bad bool) string {
	if bad {
		return "Signed egress probe was rejected."
	}
	switch source {
	case "signed":
		return "Egress probe used the signed targets."
	case "local":
		return "Egress probe used the local target."
	default:
		return "Egress probe used no extra target."
	}
}
