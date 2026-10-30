package agent

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
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
	// Egress is host:port. Empty means desired state and the agent config
	// name no target. The signed DesiredState schema has no probe field, so
	// the engine passes cfg.egress_probe_target until the contract grows one.
	Egress  string
	Ingress string
	Uplink  string
}

// ProbeResult is the outcome of one probe. Ran is false only when the
// prober was not asked to run. A failed read is Ran with a failed check.
type ProbeResult struct {
	Ran    bool
	Check  api.HealthCheckStatus
	Health api.DataHealth
	// Detail is a short token. It is not a URL and not a key.
	Detail string
}

// Prober runs the data-path probe. Tests substitute a fake.
type Prober interface {
	Probe(ctx context.Context, req ProbeRequest) ProbeResult
}

type nbStatus struct {
	Management struct {
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
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	raw, err := p.read(ctx)
	st, statusOK := nbStatus{}, false
	if err == nil {
		if jerr := json.Unmarshal(raw, &st); jerr == nil {
			statusOK = true
		}
	}
	egressNamed := req.Egress != ""
	egressOK := false
	if egressNamed && statusOK && st.Management.Connected {
		egressOK = p.dialAddr(ctx, req.Egress) == nil
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
	return ProbeResult{Ran: true, Check: check, Health: health, Detail: detail}
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

func (p netbirdProber) dialAddr(ctx context.Context, addr string) error {
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
