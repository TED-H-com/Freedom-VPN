package healthattest

import (
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// File names in the updater journal directory. The agent already reads that
// directory for forwarding.blocked.
const (
	ChallengeFile   = "health-challenge.json"
	AttestationFile = "health-attest.json"
)

// MaxReasonBytes is the longest check reason the updater will accept.
// Longer text is rejected so a health file cannot carry an unbounded blob.
const MaxReasonBytes = 160

// MaxFileBytes bounds a challenge or attestation read.
const MaxFileBytes = 64 << 10

// Components are the versions a release manifest declares (§13.3).
// node_agent and netbird are compared with the attestation. node_updater and
// enforcement are recorded so the agent can see what was declared; the
// running updater binary during checking is still the previous release.
type Components struct {
	NodeAgent   string `json:"node_agent"`
	Netbird     string `json:"netbird"`
	NodeUpdater string `json:"node_updater"`
	Enforcement string `json:"enforcement"`
}

// ChallengeProbeDest is one exact socket the synthetic probe may reach.
// AddrV4 is an IPv4 prefix string. The agent accepts only a /32.
type ChallengeProbeDest struct {
	AddrV4 string `json:"addr_v4"`
	Proto  string `json:"proto"`
	Port   uint16 `json:"port"`
}

// ChallengeProbe asks the agent to install the §13.9 exception for this
// check only. The updater names the dedicated test peer. The agent chooses
// ExpiresAt from the challenge deadline and the 300s cap. A missing probe
// installs nothing. Fields that are not a /32, a member peer, or an admin
// port on a node address are refused rather than widened.
type ChallengeProbe struct {
	PeerID       string               `json:"peer_id"`
	SourceV4     string               `json:"source_v4"`
	Destinations []ChallengeProbeDest `json:"destinations,omitempty"`
	ResolverDNS  bool                 `json:"resolver_dns,omitempty"`
}

// ProbeOmission distinguishes an absent configuration from a configured probe
// refused by the issuer. PeerID is only for protected local artifacts, never
// logs. No source or destination address is included.
type ProbeOmission struct {
	PeerID string `json:"peer_id"`
	Reason string `json:"reason"`
}

// Valid permits only the issuer's fixed, bounded refusal vocabulary.
func (p ProbeOmission) Valid() bool {
	if strings.TrimSpace(p.PeerID) == "" || len(p.PeerID) > 128 || strings.ContainsFunc(p.PeerID, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return false
	}
	switch p.Reason {
	case "malformed", "denied_destination", "node_source", "node_destination", "control_destination", "resolver_destination",
		"agent_config_unavailable", "node_addresses_unavailable", "control_addresses_unavailable", "resolver_addresses_unavailable":
		return true
	}
	return false
}

// Challenge is the updater's per-check challenge. One file is live at a
// time. A new check overwrites it. Consumed is set only when the updater
// marks the file instead of removing it; a consumed challenge never matches.
// Probe is optional. The updater omits it until it names a test peer.
type Challenge struct {
	TxnID         string          `json:"txn_id"`
	Nonce         string          `json:"nonce"`
	ReleaseID     string          `json:"release_id"`
	NodeID        string          `json:"node_id"`
	IncarnationID string          `json:"incarnation_id"`
	BootID        string          `json:"boot_id"`
	IssuedAt      time.Time       `json:"issued_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	Components    Components      `json:"components"`
	Consumed      bool            `json:"consumed,omitempty"`
	Probe         *ChallengeProbe `json:"probe,omitempty"`
	ProbeOmitted  *ProbeOmission  `json:"probe_omitted,omitempty"`
}

// Check is one agent-reported §13.9 result. Status is a HealthCheckStatus.
// Reason uses the fixed vocabulary in checkReason, never names or packets.
type Check struct {
	Status api.HealthCheckStatus `json:"status"`
	Reason string                `json:"reason"`
}

// Checks are the three layers the updater cannot observe itself.
type Checks struct {
	InternetOnly Check `json:"internet_only"`
	DNSPolicy    Check `json:"dns_policy"`
	TunnelProbe  Check `json:"tunnel_probe"`
}

// Attestation is the candidate agent's answer to one challenge.
// Published is the agent's latest publication intent. It does not open
// member forwarding.
type Attestation struct {
	TxnID          string                `json:"txn_id"`
	Nonce          string                `json:"nonce"`
	ReleaseID      string                `json:"release_id"`
	NodeID         string                `json:"node_id"`
	AgentVersion   string                `json:"agent_version"`
	NetbirdVersion string                `json:"netbird_version"`
	BootID         string                `json:"boot_id"`
	IncarnationID  string                `json:"incarnation_id"`
	GeneratedAt    time.Time             `json:"generated_at"`
	Published      api.PublicationIntent `json:"published"`
	Checks         Checks                `json:"checks"`
}

// Reason is why an attestation was refused. An empty reason means it matched.
type Reason string

const (
	ReasonTxnMismatch          Reason = "txn_mismatch"
	ReasonNonceMismatch        Reason = "nonce_mismatch"
	ReasonReleaseMismatch      Reason = "release_mismatch"
	ReasonNodeMismatch         Reason = "node_mismatch"
	ReasonIncarnationMismatch  Reason = "incarnation_mismatch"
	ReasonBootMismatch         Reason = "boot_mismatch"
	ReasonGeneratedBeforeIssue Reason = "generated_before_issue"
	ReasonGeneratedInFuture    Reason = "generated_in_future"
	ReasonExpired              Reason = "expired"
	ReasonAgentVersion         Reason = "agent_version_mismatch"
	ReasonNetbirdVersion       Reason = "netbird_version_mismatch"
	ReasonUnknownStatus        Reason = "unknown_status"
	ReasonNotRun               Reason = "not_run"
	ReasonBadPublication       Reason = "bad_publication"
	ReasonBadReason            Reason = "bad_reason"
	ReasonMalformed            Reason = "malformed"
	ReasonConsumed             Reason = "consumed"
	ReasonBadMode              Reason = "bad_mode"
	ReasonBadOwner             Reason = "bad_owner"
	ReasonSymlink              Reason = "symlink"
	ReasonTooLarge             Reason = "too_large"
	ReasonPreseeded            Reason = "attestation_precedes_challenge"
	ReasonMissing              Reason = "attestation_missing"
	ReasonChallengeReplaced    Reason = "challenge_replaced"
	ReasonManifestUnavailable  Reason = "manifest_unavailable"
	ReasonUnreadable           Reason = "unreadable"
)

func (r Reason) String() string { return string(r) }
