package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// ifNameRE matches the enforce interface rule (AGENTS.md §4).
var ifNameRE = regexp.MustCompile(`^fv[a-z0-9_-]{1,13}$`)

var releaseRE = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_-]{0,63}$`)

// Config is /etc/vpn-node/agent.json. It holds paths and ids, never a token,
// a private key or a setup key.
type Config struct {
	StateDir                string         `json:"state_dir"`
	IdentityDir             string         `json:"identity_dir"`
	JournalDir              string         `json:"journal_dir"`
	LockPath                string         `json:"lock_path"`
	ControlURL              string         `json:"control_url"`
	ServerCAFile            string         `json:"server_ca_file"`
	TrustBundle             string         `json:"trust_bundle"`
	DesiredStatePath        string         `json:"desired_state_path"`
	HeartbeatPath           string         `json:"heartbeat_path"`
	Datapath                DatapathConfig `json:"datapath"`
	AgentRelease            string         `json:"agent_release"`
	ReleaseID               string         `json:"release_id"`
	RenewLeadSeconds        int            `json:"renew_lead_seconds"`
	HeartbeatSeconds        int            `json:"heartbeat_seconds"`
	WatchdogStaleSeconds    int            `json:"watchdog_stale_seconds"`
	WatchdogIntervalSeconds int            `json:"watchdog_interval_seconds"`
	PollSeconds             int            `json:"poll_seconds"`
	ObservationSeconds      int            `json:"observation_seconds"`
	TestMode                bool           `json:"test_mode"`
	TestExemptV4            []string       `json:"test_exempt_v4"`
	ExtraDeniedV4           []string       `json:"extra_denied_v4"`
	HostV4                  []string       `json:"host_v4"`
	HostReserveBPS          uint64         `json:"host_reserve_bps"`
	NetbirdBackend          string         `json:"netbird_backend"`
	// NetbirdStatusFile, when set, is a NetBird `status --json` document
	// the probe reads instead of running the CLI. Production leaves this
	// empty. The lab container has no NetBird daemon and points it at a
	// fixture. It is not a way to mark the probe passed by itself: the
	// JSON still has to say management is connected.
	NetbirdStatusFile string `json:"netbird_status_file"`
	// EgressProbeTarget is host:port for the data-path probe. Empty skips
	// that leg. A non-null signed egress_probe replaces it (D-058).
	EgressProbeTarget string `json:"egress_probe_target"`
	UpdaterRecover    string `json:"updater_recover"`
	UpdaterConfig     string `json:"updater_config"`
	BootIDFile        string `json:"boot_id_file"`
	// DNS is optional. Absent keeps the resolver applier unset (D-061).
	DNS *DNSConfig `json:"dns,omitempty"`
}

// DatapathConfig names the interfaces enforce binds to.
type DatapathConfig struct {
	MemberIngress string `json:"member_ingress"`
	Uplink        string `json:"uplink"`
	SNAT          string `json:"snat"`
}

// LoadConfig reads agent.json. Unknown fields are rejected. Zero durations
// take the defaults from §16.3 and §16.5.
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	dec := json.NewDecoder(bytesReader(b))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("agent: config: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c *Config) applyDefaults() {
	if c.StateDir == "" {
		c.StateDir = "/var/lib/vpn-node/state"
	}
	if c.IdentityDir == "" {
		c.IdentityDir = "/var/lib/vpn-node/identity"
	}
	if c.JournalDir == "" {
		c.JournalDir = "/var/lib/vpn-node/update-journal"
	}
	if c.LockPath == "" {
		c.LockPath = "/var/lib/vpn-node/update.lock"
	}
	if c.DesiredStatePath == "" {
		c.DesiredStatePath = filepath.Join(c.StateDir, "desired-state.json")
	}
	if c.HeartbeatPath == "" {
		c.HeartbeatPath = "/run/vpn-node/heartbeat"
	}
	if c.RenewLeadSeconds <= 0 {
		c.RenewLeadSeconds = 120
	}
	if c.HeartbeatSeconds <= 0 {
		c.HeartbeatSeconds = 10
	}
	if c.WatchdogStaleSeconds <= 0 {
		c.WatchdogStaleSeconds = 30
	}
	if c.PollSeconds <= 0 {
		c.PollSeconds = 10
	}
	if c.ObservationSeconds <= 0 {
		c.ObservationSeconds = 30
	}
	if c.NetbirdBackend == "" {
		c.NetbirdBackend = "unknown"
	}
}

// ErrHostReserve means host_reserve_bps is missing or zero. The process
// exits with ExitUsage before it touches the kernel or the network (D-034).
var ErrHostReserve = errors.New("agent: host_reserve_bps is required and must be greater than zero")

// ExitUsage is the usage-error status (64, D-034). fvpn-agent returns it
// for ErrHostReserve. The updater process is not modified here.
const ExitUsage = 64

// ExitStatus maps a startup error to a process status. ErrHostReserve is
// ExitUsage. Every other error is 1. A nil error is 0.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, ErrHostReserve) {
		return ExitUsage
	}
	return 1
}

func (c Config) validate() error {
	if c.HostReserveBPS == 0 {
		return ErrHostReserve
	}
	if c.ControlURL == "" || c.ServerCAFile == "" || c.TrustBundle == "" {
		return errors.New("agent: control URL, CA and trust bundle are required")
	}
	if c.AgentRelease == "" || c.AgentRelease == "latest" || !releaseRE.MatchString(c.AgentRelease) {
		return errors.New("agent: agent_release must be a pinned version")
	}
	if c.ReleaseID != "" {
		if _, err := uuid.Parse(c.ReleaseID); err != nil {
			return errors.New("agent: release_id is not a UUID")
		}
	}
	if !c.TestMode && len(c.TestExemptV4) > 0 {
		return errors.New("agent: test_exempt_v4 requires test_mode")
	}
	if _, err := c.DatapathValue(); err != nil {
		return err
	}
	if _, err := parsePrefixes(c.TestExemptV4); err != nil {
		return err
	}
	if _, err := parsePrefixes(c.ExtraDeniedV4); err != nil {
		return err
	}
	if _, err := parsePrefixes(c.HostV4); err != nil {
		return err
	}
	switch c.NetbirdBackend {
	case "kernel", "userspace", "unknown":
	default:
		return errors.New("agent: netbird_backend must be kernel, userspace or unknown")
	}
	if c.EgressProbeTarget != "" {
		if _, _, err := net.SplitHostPort(c.EgressProbeTarget); err != nil {
			return errors.New("agent: egress_probe_target must be host:port")
		}
	}
	if err := c.validateDNS(); err != nil {
		return err
	}
	return nil
}

// DBPath is the agent SQLite file.
func (c Config) DBPath() string { return filepath.Join(c.StateDir, "agent.db") }

// UsageDBPath is the usage ledger. It is a different file from the agent store.
func (c Config) UsageDBPath() string { return filepath.Join(c.StateDir, "usage.db") }

// DatapathValue parses the datapath section.
func (c Config) DatapathValue() (enforce.Datapath, error) {
	if !ifNameRE.MatchString(c.Datapath.MemberIngress) || !ifNameRE.MatchString(c.Datapath.Uplink) {
		return enforce.Datapath{}, errors.New("agent: datapath interface names must start with fv")
	}
	if c.Datapath.MemberIngress == c.Datapath.Uplink {
		return enforce.Datapath{}, errors.New("agent: member ingress and uplink must differ")
	}
	snat, err := netip.ParseAddr(c.Datapath.SNAT)
	if err != nil || !snat.Is4() {
		return enforce.Datapath{}, errors.New("agent: snat must be an IPv4 address")
	}
	return enforce.Datapath{MemberIngress: c.Datapath.MemberIngress, Uplink: c.Datapath.Uplink, SNAT: snat}, nil
}

// ReleaseUUID parses release_id. The nil UUID means the local release is unknown.
func (c Config) ReleaseUUID() uuid.UUID {
	id, err := uuid.Parse(c.ReleaseID)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func parsePrefixes(in []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(in))
	for _, s := range in {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("agent: prefix %q", s)
		}
		out = append(out, p)
	}
	return out, nil
}

type sliceReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *sliceReader { return &sliceReader{b: b} }

func (s *sliceReader) Read(p []byte) (int, error) {
	if s.i >= len(s.b) {
		return 0, io.EOF
	}
	n := copy(p, s.b[s.i:])
	s.i += n
	return n, nil
}
