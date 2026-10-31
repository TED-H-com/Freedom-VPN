package installer

import (
	"encoding/json"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// ifNameRE is the enforce interface rule (AGENTS.md §4).
var ifNameRE = regexp.MustCompile(`^fv[a-z0-9_-]{1,13}$`)

// InstallRequest is one install or re-run (§6.3, PKG01).
type InstallRequest struct {
	ManifestPath         string
	ArtifactPath         string
	TrustPath            string
	DesiredTrustPath     string
	TokenFile            string
	ControlURL           string
	ServerCA             string
	TrustBundle          string
	HostReserveBPS       uint64
	MemberIngress        string
	Uplink               string
	SNAT                 string
	NetbirdManagementURL string
	// FaultAfter is TEST ONLY. The only accepted value is "directories".
	FaultAfter string
}

// Result is the machine-readable outcome. Status already_current is exit 0.
type Result struct {
	Status         string          `json:"status"`
	ReleaseID      string          `json:"release_id,omitempty"`
	ManifestSHA256 string          `json:"manifest_sha256,omitempty"`
	IncarnationID  string          `json:"incarnation_id,omitempty"`
	Message        string          `json:"message,omitempty"`
	UpdaterCommand string          `json:"updater_command,omitempty"`
	ManifestPath   string          `json:"manifest_path,omitempty"`
	RequestID      string          `json:"request_id,omitempty"`
	Baseline       *BaselineResult `json:"baseline,omitempty"`
}

type installedState struct {
	ReleaseID               string `json:"release_id"`
	ManifestSHA256          string `json:"manifest_sha256"`
	ReleaseSequence         string `json:"release_sequence"`
	ProductVersion          string `json:"product_version"`
	Arch                    string `json:"arch"`
	InstalledAt             string `json:"installed_at"`
	NetbirdPackageInstalled bool   `json:"netbird_package_installed"`
}

type nodeInfo struct {
	NodeID            string `json:"node_id"`
	NodeIncarnationID string `json:"node_incarnation_id"`
}

type netbirdConfig struct {
	ManagementURL string `json:"management_url"`
	InterfaceName string `json:"interface_name"`
}

type firstbootConfig struct {
	TokenFile string `json:"token_file"`
}

func (r *Result) Text() string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("status: ")
	b.WriteString(r.Status)
	if r.ReleaseID != "" {
		b.WriteString("\nrelease_id: ")
		b.WriteString(r.ReleaseID)
	}
	if r.ManifestSHA256 != "" {
		b.WriteString("\nmanifest_sha256: ")
		b.WriteString(r.ManifestSHA256)
	}
	if r.IncarnationID != "" {
		b.WriteString("\nincarnation_id: ")
		b.WriteString(r.IncarnationID)
	}
	if r.Message != "" {
		b.WriteString("\n")
		b.WriteString(r.Message)
	}
	if r.UpdaterCommand != "" {
		b.WriteString("\nupdater_command: ")
		b.WriteString(r.UpdaterCommand)
	}
	if r.ManifestPath != "" {
		b.WriteString("\nmanifest_path: " + r.ManifestPath)
	}
	if r.RequestID != "" {
		b.WriteString("\nrequest_id: " + r.RequestID)
	}
	if r.Baseline != nil {
		b.WriteString("\nbaseline: " + r.Baseline.Status)
	}
	b.WriteString("\n")
	return b.String()
}

func validateInstall(req *InstallRequest) error {
	if req.MemberIngress == "" {
		req.MemberIngress = "fvmb0"
	}
	if req.Uplink == "" {
		req.Uplink = "fvup0"
	}
	if req.ManifestPath == "" || req.ArtifactPath == "" || req.TrustPath == "" {
		return errf(ExitUsage, "usage", "manifest, artifact and trust are required")
	}
	if req.ControlURL == "" || req.ServerCA == "" || req.TrustBundle == "" {
		return errf(ExitUsage, "usage", "control URL, server CA and trust bundle are required")
	}
	if req.HostReserveBPS == 0 {
		return errf(ExitUsage, "usage", "host_reserve_bps is required and must be greater than zero")
	}
	u, err := url.Parse(req.ControlURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errf(ExitUsage, "usage", "control URL must be https")
	}
	if req.NetbirdManagementURL == "" {
		return errf(ExitUsage, "usage", "netbird management URL is required")
	}
	nu, err := url.Parse(req.NetbirdManagementURL)
	if err != nil || nu.Scheme != "https" || nu.Host == "" {
		return errf(ExitUsage, "usage", "netbird management URL must be https")
	}
	if !ifNameRE.MatchString(req.MemberIngress) || !ifNameRE.MatchString(req.Uplink) || req.MemberIngress == req.Uplink {
		return errf(ExitUsage, "usage", "datapath interface names must be distinct fv-prefixed names")
	}
	ip := net.ParseIP(req.SNAT)
	if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsUnspecified() {
		return errf(ExitUsage, "usage", "snat must be a non-loopback IPv4 address")
	}
	if req.FaultAfter != "" && req.FaultAfter != "directories" {
		return errf(ExitUsage, "usage", "fault-after must be directories")
	}
	return nil
}

func renderAgent(l Layout, req InstallRequest, manifest api.ReleaseManifest) ([]byte, error) {
	cfg := agent.Config{
		StateDir:         l.StateDir,
		IdentityDir:      l.IdentityDir,
		JournalDir:       l.UpdateJournal,
		LockPath:         l.LockPath,
		ControlURL:       req.ControlURL,
		ServerCAFile:     l.serverCA(),
		TrustBundle:      l.trustBundle(),
		DesiredStatePath: l.StateDir + "/desired-state.json",
		HeartbeatPath:    l.RunDir + "/heartbeat",
		Datapath: agent.DatapathConfig{
			MemberIngress: req.MemberIngress,
			Uplink:        req.Uplink,
			SNAT:          req.SNAT,
		},
		AgentRelease:   manifest.Components.NodeAgent,
		ReleaseID:      manifest.ReleaseId.String(),
		HostReserveBPS: req.HostReserveBPS,
		NetbirdBackend: "kernel",
		UpdaterRecover: l.updaterBin(),
		UpdaterConfig:  l.updaterConfig(),
		UpdaterJournal: l.UpdateJournal + "/journal.db",
		BootIDFile:     l.ProcBootID,
	}
	return marshalIndent(cfg)
}

func renderUpdater(l Layout, p Platform, manifest api.ReleaseManifest, node nodeInfo) ([]byte, error) {
	nodeID, err := uuid.Parse(node.NodeID)
	if err != nil {
		return nil, errf(ExitFailed, "failed", "identity node_id is not a UUID")
	}
	inc, err := uuid.Parse(node.NodeIncarnationID)
	if err != nil {
		return nil, errf(ExitFailed, "failed", "identity incarnation is not a UUID")
	}
	policy, usage := 1, 1
	if len(manifest.Compatibility.PolicySchemaVersions) > 0 {
		policy = manifest.Compatibility.PolicySchemaVersions[0]
	}
	if len(manifest.Compatibility.UsageEventSchemaVersions) > 0 {
		usage = manifest.Compatibility.UsageEventSchemaVersions[0]
	}
	cfg := updater.Config{
		Layout: updater.Layout{
			Releases: l.ReleasesDir,
			Current:  l.CurrentLink,
			Identity: l.IdentityDir,
			State:    l.StateDir,
			Journal:  l.UpdateJournal,
			Lock:     l.LockPath,
			Staging:  l.StagingDir,
			Trust:    l.releaseTrust(),
		},
		NodeID:        nodeID,
		IncarnationID: inc,
		Platform: updater.Platform{
			OSID:            p.OSID,
			OSVersion:       p.OSVersion,
			Architecture:    p.Arch,
			UpdaterVersion:  manifest.Components.NodeUpdater,
			ControlAPIMajor: manifest.Compatibility.ControlApiMajor,
			PolicySchema:    policy,
			UsageSchema:     usage,
			StateSchema:     manifest.Compatibility.StateSchemaTo,
		},
		ReleaseTrust:    l.releaseTrust(),
		DesiredTrust:    l.desiredTrust(),
		DesiredState:    l.StateDir + "/desired-state.json",
		MeterCheckpoint: l.StateDir + "/meter.checkpoint",
		StopCommand:     []string{"systemctl", "stop", UnitAgent, UnitWatchdog},
		StartCommand:    []string{"systemctl", "start", UnitWatchdog, UnitAgent},
		MTLSCert:        l.IdentityDir + "/cert.pem",
		MTLSKey:         l.IdentityDir + "/key.pem",
		MTLSCA:          l.IdentityDir + "/ca.pem",
		AllowLoopback:   false,
	}
	// UUID is an array, so encoding/json's omitempty does not omit a zero
	// value. Shadow the field to genuinely omit the optional boot fallback.
	return marshalIndent(struct {
		updater.Config
		BootID *uuid.UUID `json:"boot_id,omitempty"`
	}{Config: cfg})
}

func readNode(l Layout) (nodeInfo, error) {
	raw, err := os.ReadFile(l.nodeFile())
	if err != nil {
		return nodeInfo{}, err
	}
	var n nodeInfo
	if err := json.Unmarshal(raw, &n); err != nil {
		return nodeInfo{}, errf(ExitFailed, "failed", "identity node file is not readable")
	}
	return n, nil
}

func identityPresent(l Layout) bool {
	_, err := os.Stat(l.nodeFile())
	return err == nil
}

func readInstalled(l Layout) (installedState, bool) {
	raw, err := os.ReadFile(l.installedFile())
	if err != nil {
		return installedState{}, false
	}
	var st installedState
	if json.Unmarshal(raw, &st) != nil || st.ReleaseID == "" {
		return installedState{}, false
	}
	return st, true
}

func marshalIndent(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func writeUpdater(j *Journal, env *Env, p Platform, manifest api.ReleaseManifest) error {
	node, err := readNode(env.layout())
	if err != nil {
		return err
	}
	raw, err := renderUpdater(env.layout(), p, manifest, node)
	if err != nil {
		return err
	}
	return j.WriteFile(env.layout().updaterConfig(), raw, 0o644)
}
